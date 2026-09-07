package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # Watching a match in the real program
//
// The join screen is where watching is **reached**, and the reason it is there
// rather than behind a flag or a menu entry is the refusal: a client cannot know
// a room is full until it has been welcomed, so the way anybody arrives at this
// at all is *typed the code, pressed enter, was told room_full*. The code and the
// password are already in the fields at that moment, so the second attempt is one
// chord and one enter. → joinScreen.Watch.
//
// What a watching client then draws is draw.PlayScreen in its third reading —
// mirrored, and never asked — and everything below is about the two halves of
// that: it is really reached, and nothing on it reaches the match.

// aPlayer is a plain socket.Client on one of the room's two seats, answering off
// its own mirror. Two of them are what makes a match this client can watch: a
// watcher plays neither half, so both have to be somebody else.
func aPlayer(t *testing.T, held *aRoom, squad placement.Squad, name string) chan error {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	client, err := socket.Dial(context.Background(), held.code, wire.Hello{
		Version: version, Squad: squad, Name: name,
	}, held.books, socket.ClientOptions{})
	if err != nil {
		t.Fatalf("%s could not join: %v", name, err)
	}
	failed := make(chan error, 1)
	go func() {
		failed <- client.Play(context.Background(), func(prompt *battle.Prompt) (battle.Choice, bool) {
			fight := client.Mirror().Battle()
			if fight == nil {
				return battle.Choice{}, false
			}
			return fight.Suggest(prompt)
		})
	}()
	t.Cleanup(client.Close)
	return failed
}

// watchingKeysSwept is how many keystrokes the sweep below sends, written down so
// a walk over an empty list cannot pass.
//
// ⚠️ It is everyKeyPressed minus **esc**, which is left out rather than expected
// to do nothing: esc on a live battle leaves the match by design — nobody
// forfeits — so pressing it would end the very thing the rest of the sweep is
// measuring. Everything else this client can send is in.
const watchingKeysSwept = 59

// keysAWatcherMayPress is the sweep's list, and the one key it holds back.
func keysAWatcherMayPress(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, name := range everyKeyPressed() {
		if name == "esc" {
			continue
		}
		out = append(out, name)
	}
	if len(out) != watchingKeysSwept {
		t.Fatalf("the sweep sends %d keys and is written down as %d: widening what this suite "+
			"can press is a decision, not a silent change to what \"every key\" means",
			len(out), watchingKeysSwept)
	}
	return out
}

// TestAMatchIsWatchedFromTheJoinScreenToTheResultOverALoopbackListener is the
// whole vertical for a spectator, and it is the test decision two owes: watching
// has to be reachable in the **real program** rather than only in a unit test of
// a screen.
//
// A real registry, a real server, a real listener; two plain socket.Clients take
// the room's two seats and this client joins with the watch toggle on, driven
// through the keys a reader presses until it lands on the result.
//
// Four claims, each of which could be quietly wrong while a match still ran:
//
//  1. **ctrl+w and enter really reach wire.Hello.Watch.** The welcome comes back
//     naming **no seat**, which is the room's whole answer to the flag, and the
//     waiting screen says so instead of drawing an empty one.
//  2. **The battle screen goes live and watching, and never holds a prompt** —
//     on any redraw, over the whole match, including the host's turns.
//  3. **Every key this client can send changes nothing.** → the sweep.
//  4. **Nothing was sent.** The room answers every message from an unseated
//     sender with wire.CodeNotYourTurn, so an empty refusal list over a whole
//     match is the room saying it was never spoken to.
func TestAMatchIsWatchedFromTheJoinScreenToTheResultOverALoopbackListener(t *testing.T) {
	held, library := openARoom(t, 1)
	m, fake := joining(t, held, library, i18n.Vi)

	// Claim 1, the reaching: the toggle is a chord on the join screen, and the
	// row under the squad chooser is what says it is on.
	m = key(t, m, "ctrl+w")
	if !m.join.Watch {
		t.Fatal("ctrl+w on the join screen did not turn watching on, so nothing below joins " +
			"as a spectator")
	}
	if want := m.text(i18n.JoinWatchOn); !strings.Contains(drawnBody(m), want) {
		t.Fatalf("the join screen does not say it is going to watch:\n%s", drawnBody(m))
	}

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the watching dial was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	if m.screen != screenWaiting {
		t.Fatalf("a joined room landed on screen %v, want the waiting screen", m.screen)
	}
	if m.waiting.Seat.Valid() {
		t.Fatalf("the room seated this client in the %q seat, and a watcher takes none",
			m.waiting.Seat)
	}
	if !m.waiting.Seated || !m.waiting.Welcome.Watching() {
		t.Fatalf("the welcome does not read as a watcher's: %+v", m.waiting)
	}
	if want := m.text(i18n.WaitingNoSeat); !strings.Contains(drawnBody(m), want) {
		t.Fatalf("the waiting screen does not say this client is watching:\n%s", drawnBody(m))
	}

	// The two people whose match it is.
	hostFailed := aPlayer(t, held, held.squads[0], "Host")
	guestFailed := aPlayer(t, held, held.squads[1], "Guest")

	keys := keysAWatcherMayPress(t)
	swept, redraws, live := false, 0, 0
	deadline := time.Now().Add(theWholeMatch)
	for m.screen != screenResult && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
			redraws++
			if m.screen != screenBattle {
				continue
			}
			live++
			// Claim 2, on every redraw rather than once: the mirror is never
			// asking, so the screen never holds a turn — and the footer is the
			// watching one rather than the live one, which is what stops the
			// program naming keys it ignores.
			if !m.battle.Live || !m.battle.Watching {
				t.Fatalf("the battle screen is live=%v watching=%v",
					m.battle.Live, m.battle.Watching)
			}
			if m.battle.Pending != nil {
				t.Fatalf("a watching screen was handed %q's turn %d",
					m.battle.Pending.Unit, m.battle.Pending.Turn)
			}
			if m.battle.Answered {
				t.Fatal("a watching screen recorded an answer, and it answers nothing")
			}
			if _, footer := m.parts(); footer == m.text(i18n.PlayLiveFooter) {
				t.Fatalf("a watching battle draws the live footer, which names enter, ? and "+
					"p:\n%s", footer)
			}
		}
		// Claim 3, once, at the first moment there is a live board under the
		// keys. Doing it on every redraw would make this a test of how long a
		// sweep takes rather than of what a key does.
		if !swept && m.screen == screenBattle && m.battle.Fight != nil {
			m = sweepAWatcher(t, m, keys)
			swept = true
		}
	}
	if m.screen != screenResult {
		t.Fatalf("the watched match did not reach the result inside %s; this client is on "+
			"screen %v", theWholeMatch, m.screen)
	}
	if !swept {
		t.Fatal("the key sweep never ran, so claim three measured nothing")
	}
	if live == 0 {
		t.Fatal("this client never drew the battle screen, so it watched nothing")
	}
	if err := <-hostFailed; err != nil {
		t.Fatalf("the host's loop: %v", err)
	}
	if err := <-guestFailed; err != nil {
		t.Fatalf("the guest's loop: %v", err)
	}

	// Claim 4, at the far end: an empty refusal list is the room saying nothing
	// was ever sent to it from this connection.
	var refusals []wire.Code
	var compared int
	var fought []socket.Fought
	m.session.read(func(sight socket.Sight) {
		refusals = append(refusals, sight.Refusals...)
		fought = append(fought, sight.Fought...)
	})
	compared = comparedBy(m)
	if len(refusals) != 0 {
		t.Errorf("the watching client was refused %v; the room answers a watcher only when the "+
			"watcher has spoken to it, so every one of those is a message this client sent",
			refusals)
	}
	if compared == 0 {
		t.Error("this client compared no digests, so it drew a board it never checked")
	}
	if len(fought) == 0 {
		t.Error("the watching client settled no battle, so it computed no ending of its own")
	}
	if m.result.Err != nil {
		t.Errorf("the watched match ended with %v", m.result.Err)
	}
	t.Logf("watched %d battles over %d redraws (%d on the board), %d digests compared, "+
		"%d keys pressed, nothing sent", len(fought), redraws, live, compared, len(keys))
}

// sweepAWatcher presses every key this client can send on a live watching battle
// and asserts the screen came back watching, unasked and unanswered.
//
// ⚠️ **What it does NOT assert is the battle's event count**, and that is not an
// omission: the battle under a watching screen is the mirror's, and the Play
// goroutine is stepping it while these keys are being pressed, so a count read
// here would move for reasons that have nothing to do with a keystroke. What the
// match being untouched is measured by is the refusal list at the far end, after
// the match — → claim four.
func sweepAWatcher(t *testing.T, m model, keys []string) model {
	t.Helper()
	for _, name := range keys {
		m = key(t, m, name)
		if m.screen != screenBattle {
			t.Fatalf("%q took a watching client off the battle screen to %v", name, m.screen)
		}
		if !m.battle.Watching || !m.battle.Live {
			t.Fatalf("%q took a watching battle out of its mode: live=%v watching=%v",
				name, m.battle.Live, m.battle.Watching)
		}
		if m.battle.Pending != nil || m.battle.Answered || m.battle.Aiming {
			t.Fatalf("%q opened a decision on a watching battle: pending=%v answered=%v aiming=%v",
				name, m.battle.Pending, m.battle.Answered, m.battle.Aiming)
		}
		if m.battle.Err != nil {
			t.Fatalf("%q on a watching battle failed: %v", name, m.battle.Err)
		}
		if len(m.battle.Notes) > 0 {
			t.Fatalf("%q on a watching battle wrote a file: %v", name, m.battle.Notes)
		}
		if m.picker != nil {
			t.Fatalf("%q raised a picker over a watching battle", name)
		}
	}
	return m
}

// TestTheWatchToggleSurvivesARefusalAndIsSentOnTheNextTry is the constraint
// decision two was given: a client cannot know a room is full until it has been
// **welcomed**, so whatever reaches watching has to work after the first attempt
// comes back refused.
//
// It presses the whole sequence a refused player presses — the refusal arrives,
// the toggle goes on, enter goes again — and asserts the code was still in the
// field and the second hello really carried the flag.
//
// ⚠️ **The refusal is delivered as the message a failed dial produces** rather
// than by filling a room up, which is what keeps this a test of the *screen's*
// behaviour after one: room_full is the transport's to produce and
// internal/socket already measures it.
func TestTheWatchToggleSurvivesARefusalAndIsSentOnTheNextTry(t *testing.T) {
	held, library := openARoom(t, 1)
	m, _ := joining(t, held, library, i18n.Vi)

	typed := m.join.Code.Value()
	if typed == "" {
		t.Fatal("no code was typed, so there is nothing for the second attempt to keep")
	}
	m = send(t, m, matchFailedMsg{err: &socket.Refusal{Code: wire.CodeRoomFull}})
	if m.join.Refused != wire.CodeRoomFull.String() {
		t.Fatalf("the join screen holds the refusal %q, want room_full", m.join.Refused)
	}
	if m.join.Code.Value() != typed {
		t.Fatalf("the refusal emptied the code field: %q", m.join.Code.Value())
	}

	m = key(t, m, "ctrl+w")
	if !m.join.Watch {
		t.Fatal("ctrl+w after a refusal did not turn watching on")
	}
	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("the second enter asked for no command")
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the second attempt was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	if !m.waiting.Seated || !m.waiting.Welcome.Watching() {
		t.Fatalf("the second attempt was not welcomed as a watcher: %+v", m.waiting)
	}
	// And the toggle is a toggle rather than a latch: pressing it twice puts it
	// back wherever it started, so a reader who changed their mind is not stuck
	// watching. ⚠️ It is asserted as an **involution** rather than as "twice is
	// off", because the flag survives Refresh on purpose — somebody watching an
	// evening's matches joins several rooms in a row — so the screen this lands
	// on already has it on and "twice is off" would be a claim about the start
	// state rather than about the key.
	once := key(t, m.enter(screenJoin), "ctrl+w")
	twice := key(t, once, "ctrl+w")
	if once.join.Watch == twice.join.Watch {
		t.Errorf("ctrl+w left watching at %v both times, so the key only goes one way",
			once.join.Watch)
	}
	if twice.join.Watch != m.enter(screenJoin).join.Watch {
		t.Error("ctrl+w twice did not put the toggle back where it started")
	}
}

// TestTheWatchRowIsDrawnInBothStates is the half a footer cannot state: a toggle
// whose off state draws nothing is a toggle a reader cannot tell they have not
// pressed.
func TestTheWatchRowIsDrawnInBothStates(t *testing.T) {
	held, library := openARoom(t, 1)
	for _, lang := range i18n.Langs() {
		m, _ := joining(t, held, library, lang)
		off := drawnBody(m)
		if want := m.text(i18n.JoinWatchLabel); !strings.Contains(off, want) {
			t.Errorf("in %s the join screen draws no watch row:\n%s", lang, off)
		}
		if want := m.text(i18n.JoinWatchOff); !strings.Contains(off, want) {
			t.Errorf("in %s the join screen does not say watching is off:\n%s", lang, off)
		}
		on := drawnBody(key(t, m, "ctrl+w"))
		if want := m.text(i18n.JoinWatchOn); !strings.Contains(on, want) {
			t.Errorf("in %s the join screen does not say watching is on:\n%s", lang, on)
		}
		if want := m.text(i18n.JoinWatchOff); strings.Contains(on, want) {
			t.Errorf("in %s the join screen says watching is both on and off:\n%s", lang, on)
		}
	}
}

// TestAWatchingReadingReachesTheBattleScreen is liveOf's own half, measured
// without a socket: the model turns a socket.Sight into a draw.PlayLive, and the
// one field this step added has to survive that.
//
// ⚠️ It is worth a test of its own because a dropped field in a keyed struct
// literal is silent — the same trap DraftLive's two walks exist for — and the
// consequence here is a spectator handed the player's footer and the player's
// clock.
func TestAWatchingReadingReachesTheBattleScreen(t *testing.T) {
	for _, watching := range []bool{false, true} {
		live := liveOf(socket.Sight{Watching: watching}, draw.PlayClock{})
		if live.Watching != watching {
			t.Errorf("a sight with Watching=%v produced a reading with Watching=%v",
				watching, live.Watching)
		}
	}
}
