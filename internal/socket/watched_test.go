package socket

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// # A watching CLIENT, which is the half step four left out
//
// Step four's spectator reads bodies off a socket and never builds a battle,
// deliberately: what those tests are about is what reaches the wire and in what
// order, and a mirror would fold a run of turns into a board that no longer says
// which of them arrived. This file is the other half — a real socket.Client with
// a real Mirror, running the real Play loop over a whole match — and the claims
// are about what a *client* does with what it is handed.
//
// ⚠️ **The bug this file exists for.** Mirror.asking derives "it is my turn" from
// `unit.Side != m.side`, and m.side comes off wire.Start — which for a watcher is
// the **host's** side, because that is what the room records and a spectator
// plays neither half. So without a guard a watcher believes it is being asked
// every time the host's unit is on turn: Play calls the chooser, sends a wire.Act,
// and the room refuses it. That does not corrupt the match — step two's test is
// what says so — it makes a spectator a machine that spams refusals at a match it
// came to look at. TestAWatchingClientIsNeverAskedOnAnyTurn is the net, and its
// mutation is putting the comparison back.

// viewer is a watching client with its Play loop running, everything the loop
// saw, and a chooser that is a **trap** rather than a rating.
//
// ⚠️ **The chooser answers with a real move rather than refusing.** A chooser
// that returned false would make a wrongly-asked watcher send a wire.Pass, which
// is a smaller and quieter failure than the one being guarded against; one that
// panicked would take the test's own goroutine out with a stack rather than a
// sentence. So it takes the engine's own suggestion, exactly as a player's does —
// the mistake is then as loud on the wire as it would be in production, and the
// counter below is what says it happened.
type viewer struct {
	*Client
	name string

	mu sync.Mutex
	// asked is how many times Play called the chooser, which for a watcher must
	// be nought and for a player is once a turn.
	asked int
	// samples is one reading per message this client took in: whether the mirror
	// said it was being asked, and which half the open turn belongs to.
	samples []sample
	// finished is Play having returned, and err what it returned.
	err      error
	finished chan struct{}
}

// sample is one reading of a watching mirror, taken on the Play goroutine after
// a message has been applied.
//
// ⚠️ **It records whose turn it is as well as whether this client was asked**,
// because the two together are what makes the measurement mean anything: "never
// asked" is trivially true of a client that never saw the host's unit on turn, so
// the count of host turns is the premise and the count of asks is the claim.
type sample struct {
	asking bool
	// open is true when the mirror is stopped on a prompt at all, host when that
	// prompt's unit stands on the half this client's wire.Start named — which for
	// a watcher is the host's.
	open bool
	host bool
}

// watching dials a watcher, starts its Play loop, and records a reading after
// every message.
func (l *listener) watching(t *testing.T, code wire.RoomCode, name string,
	dependencies room.Deps) *viewer {
	t.Helper()
	out := &viewer{name: name, finished: make(chan struct{})}
	// The hook needs the client and the client needs the hook, which is the knot
	// listener.drafter is tied with and is untied the same way: nothing calls the
	// hook until Play starts, and Play starts below.
	client, err := Dial(context.Background(), code, watchingHello(t, name), dependencies.Books,
		ClientOptions{
			Timings:    l.timings,
			Characters: dependencies.Characters,
			Stepped:    out.observe,
		})
	if err != nil {
		t.Fatalf("%s dials room %s to watch: %v", name, code, err)
	}
	if seat := client.Seat(); seat.Valid() {
		t.Fatalf("%s was welcomed into the %q seat, and a watcher takes none", name, seat)
	}
	if !client.Watching() {
		t.Fatalf("%s was welcomed with no seat and does not read as watching", name)
	}
	out.Client = client
	t.Cleanup(client.Close)
	go func() {
		defer close(out.finished)
		// ⚠️ A **real** chooser is handed in on purpose. "A watcher must not need
		// a chooser" is worth nothing if the test proves it by passing nil: what
		// has to hold is that one passed in is never reached.
		out.err = client.Play(context.Background(), out.choose)
	}()
	return out
}

// choose is the trap. → the type comment for why it answers rather than refusing.
func (w *viewer) choose(prompt *battle.Prompt) (battle.Choice, bool) {
	w.mu.Lock()
	w.asked++
	w.mu.Unlock()
	fight := w.Mirror().Battle()
	if fight == nil {
		return battle.Choice{}, false
	}
	return fight.Suggest(prompt)
}

// observe takes one reading, on the Play goroutine and with no lock held — which
// is what ClientOptions.Stepped guarantees and why the reading can take the
// mirror's own.
func (w *viewer) observe() {
	var taken sample
	w.Mirror().Read(func(sight Sight) {
		taken.asking = sight.Asking != nil
		if sight.Fight == nil {
			return
		}
		prompt := sight.Fight.Pending()
		if prompt == nil {
			return
		}
		taken.open = true
		unit, known := sight.Fight.Unit(prompt.Unit)
		taken.host = known && unit.Side == sight.Side
	})
	w.mu.Lock()
	w.samples = append(w.samples, taken)
	w.mu.Unlock()
}

// seen is everything the loop recorded, read after it has returned.
func (w *viewer) seen(t *testing.T) (asked int, samples []sample, err error) {
	t.Helper()
	w.wait(t)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.asked, append([]sample(nil), w.samples...), w.err
}

// wait blocks until this watcher's Play loop has returned.
func (w *viewer) wait(t *testing.T) {
	t.Helper()
	select {
	case <-w.finished:
	case <-time.After(theWholeMatch):
		t.Fatalf("%s was still in Play after %s", w.name, theWholeMatch)
	}
}

// aWatchedBo3 is a whole bo3 with the given watchers attached before it starts.
//
// A **bo3** rather than a bo1, because a watcher's arithmetic for "the match is
// over" is Mirror.Over — the series length off the welcome and each battle's
// outcome off its own Ended event — and a bo1 cannot tell that from stopping at
// the first ending.
func aWatchedBo3(t *testing.T, held *listener, dependencies room.Deps,
	code wire.RoomCode) (host, guest *Client, done ending) {
	t.Helper()
	host = held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
		dependencies.Books)
	ctx := context.Background()
	hostPlay := play(ctx, host, rating(host))
	guest = held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""),
		dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))
	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	return host, guest, held.finished(t)
}

// TestAWatchingClientIsNeverAskedOnAnyTurn is the regression test for the bug at
// the head of this file: a real spectator over a real socket, watching a whole
// bo3, asked for nothing on any turn of it.
//
// Three readings, and the order matters:
//
//  1. **The chooser was never called.** That is the production consequence —
//     Play calls it and sends what it answers, so a call is a wire.Act leaving a
//     spectator's socket.
//  2. **The mirror never said it was asking**, on any of the readings taken after
//     any of the messages. That is the derivation itself, sampled rather than
//     inferred from (1): a chooser could go uncalled because the loop never got
//     round to it, and this is the fact the loop reads.
//  3. **The host's unit really was on turn**, on a good number of those readings.
//     ⚠️ Without this the whole test is satisfied by a match nobody watched: the
//     bug fires on the *host's* turns specifically, so a run in which none was
//     ever observed would pass with the guard deleted.
func TestAWatchingClientIsNeverAskedOnAnyTurn(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	watching := held.watching(t, code, "the.watcher", dependencies)
	host, _, done := aWatchedBo3(t, held, dependencies, code)
	asked, samples, err := watching.seen(t)
	if err != nil {
		t.Fatalf("the watcher's Play returned %v", err)
	}

	if asked != 0 {
		t.Errorf("the watcher's chooser was called %d times, and a spectator answers nothing: "+
			"every one of those is a wire.Act the room then has to refuse", asked)
	}
	onTurn, hostTurns := 0, 0
	for _, taken := range samples {
		if taken.asking {
			onTurn++
		}
		if taken.open && taken.host {
			hostTurns++
		}
	}
	if onTurn != 0 {
		t.Errorf("the watcher's mirror said it was being asked on %d of %d readings",
			onTurn, len(samples))
	}
	// The premise, stated as a number rather than assumed. The bug is about the
	// host's turns, so a run that never saw one measures nothing at all — and the
	// bound is against the match's own arithmetic rather than against a figure
	// this test made up.
	if compared := host.Mirror().Compared(); hostTurns*4 < compared {
		t.Fatalf("the watcher observed the host's unit on turn %d times over %d turns of the "+
			"match, which is too few for \"never asked\" to be about anything",
			hostTurns, compared)
	}
	// ⚠️ **A bo3 is over at two-nil, so two battles is a whole match and three is
	// the other whole match.** Asserting the configured number here was the first
	// version of this line and it was wrong on every run: Mirror.Over stops the
	// series the moment a side is past taking back, which is the arithmetic the
	// welcome's Battles exists for. What is worth asserting is that more than one
	// was played, so the wire.Start-per-battle path was walked.
	if played := len(done.reading.Played); played < 2 || played > configuration.Battles {
		t.Fatalf("the room played %d battles of a best-of-%d, so the watcher watched part of "+
			"a match", played, configuration.Battles)
	}
	t.Logf("the watcher took %d readings of a %d-battle match, %d of them with the host's unit "+
		"on turn, and was asked %d times",
		len(samples), len(done.reading.Played), hostTurns, asked)
}

// TestAWatchingClientSendsNothingForAWholeMatch is the same claim measured at the
// far end instead of at this one.
//
// ⚠️ **The instrument is the room's own refusal, and that is what makes it a
// reading of the wire rather than of intent.** room.Deliver refuses **every**
// message from a sender with no seat — wire.CodeNotYourTurn, step two's test — and
// a refusal comes back down the sender's own socket, where Mirror.Receive records
// it. So an empty Mirror.Refusals over a whole match is the room saying it was
// never spoken to.
//
// ⚠️ **The second arm is the instrument's own proof.** An empty list is also what
// a broken instrument produces, so a second watcher sends one wire.Act by hand and
// the refusal has to appear — without that arm this test would pass on a client
// that recorded no refusals at all.
func TestAWatchingClientSendsNothingForAWholeMatch(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	quiet := held.watching(t, code, "the.quiet.watcher", dependencies)
	loud := held.watching(t, code, "the.loud.watcher", dependencies)
	// One act, sent by hand on the loud watcher's own connection before the match
	// starts, so the room is certainly there to answer it.
	if err := loud.conn.send(context.Background(), wire.Act{Skill: "tackle", Aim: hex.At(hex.Offset{})}); err != nil {
		t.Fatalf("the loud watcher could not send its act: %v", err)
	}

	host, guest, done := aWatchedBo3(t, held, dependencies, code)
	quiet.wait(t)
	loud.wait(t)

	if refusals := quiet.Mirror().Refusals(); len(refusals) != 0 {
		t.Errorf("the quiet watcher was refused %v, and the room only ever answers a watcher "+
			"that spoke to it", refusals)
	}
	if refusals := loud.Mirror().Refusals(); len(refusals) != 1 || refusals[0] != wire.CodeNotYourTurn {
		t.Fatalf("the loud watcher sent one act and was refused %v: without a refusal here the "+
			"empty list above is a measurement of nothing", refusals)
	}
	// And the two players were not refused either, which is what says the loud
	// watcher's act did not land on somebody's real turn.
	for _, client := range []*Client{host, guest} {
		if refusals := client.Mirror().Refusals(); len(refusals) != 0 {
			t.Errorf("%s was refused %v while two people were watching", client.Seat(), refusals)
		}
	}
	// Two or three, for the reason the test above states: a bo3 ends at two-nil.
	if played := len(done.reading.Played); played < 2 || played > configuration.Battles {
		t.Errorf("the room played %d battles of a best-of-%d", played, configuration.Battles)
	}
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors: %q", len(said), said)
	}
}

// TestAWatchersDigestsAgreeWithTheRoomsForEveryTurn is the claim that a watcher
// is a **mirror** like any other rather than a screen being told what to draw.
//
// It checks the same digest on the same turn as the two players do — Mirror.apply
// hashes the events its own engine produced and compares them against the room's,
// and Mirror.Compared counts how many times it did. That check is free for a
// spectator and it is worth having: it is the one thing that says the board on a
// spectator's screen is the board the players are looking at, rather than a
// plausible one computed from the same decisions by a client that had drifted.
//
// ⚠️ **A divergence is an error out of Play**, so "no error" is half the claim and
// the count is the other half: a client that checked nothing returns nil too.
func TestAWatchersDigestsAgreeWithTheRoomsForEveryTurn(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	watching := held.watching(t, code, "the.watcher", dependencies)
	host, guest, done := aWatchedBo3(t, held, dependencies, code)
	_, _, err := watching.seen(t)
	if err != nil {
		var diverged *Divergence
		if ok := asDivergence(err, &diverged); ok {
			t.Fatalf("the watcher diverged from the room: %v", diverged)
		}
		t.Fatalf("the watcher's Play returned %v", err)
	}
	compared := watching.Mirror().Compared()
	if compared != host.Mirror().Compared() || compared != guest.Mirror().Compared() {
		t.Fatalf("the watcher checked %d digests, the host %d and the guest %d: a spectator "+
			"applies the same turns and therefore checks the same number",
			compared, host.Mirror().Compared(), guest.Mirror().Compared())
	}
	if compared == 0 {
		t.Fatal("nobody checked a digest, so this test measured nothing")
	}
	// And it settled the same series, from its own Ended events rather than from
	// anything on the wire — there is no standing message.
	fought := watching.Mirror().Fought()
	if len(fought) != len(done.reading.Played) {
		t.Fatalf("the watcher settled %d battles and the room played %d",
			len(fought), len(done.reading.Played))
	}
	for at, one := range fought {
		theirs := host.Mirror().Fought()[at]
		if one.Outcome != theirs.Outcome || one.Seed != theirs.Seed || one.Turns != theirs.Turns {
			t.Errorf("battle %d: the watcher settled %s and the host %s",
				one.Battle, side(one), side(theirs))
		}
	}
	t.Logf("a watcher checked %d digests over %d battles, the same as both players",
		compared, len(fought))
}

// asDivergence is errors.As for the one error type this test wants to name.
func asDivergence(err error, into **Divergence) bool {
	diverged, ok := err.(*Divergence)
	if ok {
		*into = diverged
	}
	return ok
}

// TestAWatcherJoiningMidMatchEndsOnTheSameBoard is the mid-joiner, measured on
// the thing a spectator actually looks at.
//
// Step four holds that a late watcher is handed the same **bodies** as an early
// one. That is the transport's claim; this is the client's, and they are not the
// same claim: a client that applied a catch-up run in the wrong order, or opened
// its battle on the wrong wire.Start, would hold the same bodies and draw a
// different board.
//
// So what is compared is every unit's cell, health and death — which is the board
// — plus the series each of them settled.
func TestAWatcherJoiningMidMatchEndsOnTheSameBoard(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 1, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	early := held.watching(t, code, "the.early.watcher", dependencies)

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
		dependencies.Books)
	ctx := context.Background()
	choose, halfway := stepped(rating(host), 12)
	hostPlay := play(ctx, host, choose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""),
		dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, halfway, "the host's twelfth decision")
	behind := host.Mirror().Compared()
	if behind < 12 {
		t.Fatalf("the host had checked %d digests when the late watcher joined, so there was "+
			"nothing for it to catch up on", behind)
	}
	late := held.watching(t, code, "the.late.watcher", dependencies)

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	early.wait(t)
	late.wait(t)

	board := func(w *viewer) []string { return boardOf(t, w) }
	one, two := board(early), board(late)
	if len(one) == 0 {
		t.Fatal("the early watcher ended holding no board, so nothing below is compared")
	}
	if len(one) != len(two) {
		t.Fatalf("the early watcher ended with %d units and the late one with %d",
			len(one), len(two))
	}
	for at := range one {
		if one[at] != two[at] {
			t.Errorf("unit %d: the early watcher reads %q and the late one %q",
				at, one[at], two[at])
		}
	}
	if e, l := early.Mirror().Compared(), late.Mirror().Compared(); e != l {
		t.Errorf("the early watcher checked %d digests and the late one %d, and a catch-up is "+
			"applied through the same Replay a live turn is", e, l)
	}
	if len(early.Mirror().Fought()) != len(done.reading.Played) {
		t.Errorf("the early watcher settled %d battles and the room played %d",
			len(early.Mirror().Fought()), len(done.reading.Played))
	}
	t.Logf("a watcher that joined after %d turns ended on the same %d-unit board as one that "+
		"was there from the first message", behind, len(one))
}

// boardOf is a watcher's final board as a comparable list: one line a unit, in
// the battle's own order.
//
// ⚠️ **Read inside Mirror.Read and flattened to strings before it leaves**, which
// is that method's own rule: nothing it hands over may outlive the call, and a
// []*battle.Unit kept afterwards is a pointer into a battle another goroutine may
// still be stepping.
func boardOf(t *testing.T, w *viewer) []string {
	t.Helper()
	var out []string
	w.Mirror().Read(func(sight Sight) {
		if sight.Fight == nil {
			return
		}
		for _, unit := range sight.Fight.Units() {
			out = append(out, fmt.Sprintf("%s at %v hp %d dead %v side %s",
				unit.ID, unit.Cell, unit.HP, unit.Dead, unit.Side))
		}
	})
	return out
}

// TestAWatcherSurvivesBothEndings is the two ways a match can stop, from a
// spectator's chair — and they are two different mechanisms rather than one.
//
// A match that simply **finishes** sends a watcher nothing: there is no
// series-standing message, so the client computes the ending from its own Ended
// events, wire.Welcome.Battles and the turn cap, exactly as a player does. A match
// **abandoned** by a departure is the one ending no mirror can compute, and it
// arrives off the room's record as a wire.Closed{ClosureLeft}.
//
// ⚠️ **The abandoned arm is what says the record's third append site reaches a
// client at all.** The other two — a battle's wire.Start and each wire.Turn — are
// exercised by every test in this file; the closure is appended on one path only,
// and a watcher is the only reader it has.
func TestAWatcherSurvivesBothEndings(t *testing.T) {
	dependencies := deps(t)

	t.Run("a match that finishes", func(t *testing.T) {
		configuration := config(11, 3, room.DefaultAllowance)
		held := listening(t, Timings{})
		code := held.open(t, configuration, dependencies)
		watching := held.watching(t, code, "the.watcher", dependencies)
		_, _, done := aWatchedBo3(t, held, dependencies, code)
		if _, _, err := watching.seen(t); err != nil {
			t.Fatalf("the watcher's Play returned %v", err)
		}
		if closure, closed := watching.Mirror().Closure(); closed {
			t.Errorf("the watcher was sent the closure %q for a match that ended on the board, "+
				"and a client computes that ending itself", closure)
		}
		if !watching.Mirror().Over() {
			t.Error("the watcher does not think the match is over, so it worked the ending out " +
				"from neither its own battles nor the series length")
		}
		if len(watching.Mirror().Fought()) != len(done.reading.Played) {
			t.Errorf("the watcher settled %d battles and the room played %d",
				len(watching.Mirror().Fought()), len(done.reading.Played))
		}
	})

	t.Run("a match somebody walked out of", func(t *testing.T) {
		configuration := config(11, 3, room.DefaultAllowance)
		held := listening(t, Timings{})
		code := held.open(t, configuration, dependencies)
		watching := held.watching(t, code, "the.watcher", dependencies)

		host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
			dependencies.Books)
		ctx := context.Background()
		choose, halfway := stepped(rating(host), 6)
		hostPlay := play(ctx, host, choose)
		guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""),
			dependencies.Books)
		guestPlay := play(ctx, guest, rating(guest))

		reachedOr(t, halfway, "the host's sixth decision")
		// The guest walks out mid-match, which is the only thing that puts a
		// wire.Closed on the record.
		guest.Close()

		if err := guestPlay.wait(t, "the guest"); err != nil {
			t.Fatalf("the guest's match: %v", err)
		}
		if err := hostPlay.wait(t, "the host"); err != nil {
			t.Fatalf("the host's match: %v", err)
		}
		if _, _, err := watching.seen(t); err != nil {
			t.Fatalf("the watcher's Play returned %v", err)
		}
		closure, closed := watching.Mirror().Closure()
		if !closed || closure != wire.ClosureLeft {
			t.Fatalf("the watcher was told the match closed for %q (closed=%v), want %q off the "+
				"room's own record", closure, closed, wire.ClosureLeft)
		}
		if !watching.Mirror().Over() {
			t.Error("a closure did not end the watcher's match")
		}
		// The host was told the same thing down its own socket, which is what
		// says the watcher read the record rather than being handed somebody
		// else's message.
		if got, closed := host.Mirror().Closure(); !closed || got != closure {
			t.Errorf("the host was told %q (closed=%v) and the watcher %q", got, closed, closure)
		}
	})
}
