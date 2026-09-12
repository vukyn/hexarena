package screen

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// # The battle screen as a spectator reads it
//
// A watching screen is the live one with everything a **decision** needs taken
// away and nothing else changed: the same board, the same roster, the same queue
// line, the same log and the same budget over all of them. So what these tests
// are about is the taking-away — that no key reaches the match, that no footer
// promises one, and that the three sentences which address the reader as one of
// the two players are replaced rather than left standing.
//
// ⚠️ **Nothing here declares "a watcher is never asked".** That lives at one
// declaration, socket.Mirror.asking, and what reaches this package is its
// consequence: PlayLive.Asking is nil on every turn of a watched match, so
// PlayScreen.Pending is nil and the `p.Pending == nil` return in Update is what
// every key falls past. These tests measure that consequence rather than restate
// the rule — → internal/socket's own watching tests for the rule.

// aWatchedBattle is a live screen over a battle **neither** side of which is this
// client's, which is the whole of what a spectator has: the mirror answers no
// prompt for a watcher, so the reading carries none.
//
// ⚠️ It is handed a nil Asking rather than the prompt with a flag beside it,
// because that is the only reading socket.Mirror can produce for a watcher. A
// fixture that handed over both would be recording a state the program cannot
// reach and would then be measuring itself.
func aWatchedBattle(t *testing.T, c Context, side int) PlayScreen {
	t.Helper()
	local := withAFullLog(t, c, atABattleOf(t, c, side))
	p := NewPlayScreen().Attach(c, PlayLive{
		Fight: local.Fight, Side: local.Side, Seed: local.Seed,
		Watching: true,
		Clock:    aCountdown(PlayClockYou),
	})
	if !p.Live || !p.Watching {
		t.Fatalf("the watched battle came back live=%v watching=%v", p.Live, p.Watching)
	}
	if p.Pending != nil {
		t.Fatal("the watched battle holds a turn of its own, so every key below would be " +
			"pressed on a screen that is being asked for one")
	}
	return p
}

// aWatchedBattleOver is the same screen on a battle that has ended, which is the
// state the over footer belongs to and the one a spectator sits on last.
func aWatchedBattleOver(t *testing.T, c Context) PlayScreen {
	t.Helper()
	fight, prompt := aBattleNobodyHereDrives(t, c, 1)
	for range PlayTurnLimit {
		if fight.Finished() {
			break
		}
		prompt = steppedByTheRoom(t, fight, prompt)
		if prompt == nil && !fight.Finished() {
			t.Fatal("the battle stopped opening turns without finishing")
		}
	}
	if !fight.Finished() {
		t.Fatal("the battle never ended, so the watched over state is drawn by nothing")
	}
	return NewPlayScreen().Attach(c, PlayLive{Fight: fight, Watching: true})
}

// TestAWatchedBattleSaysNothingAboutTheReaderTakingATurn is the three wordings
// live mode addresses the reader with, replaced.
//
// ⚠️ **It asserts the live sentence is GONE as well as the watching one being
// there.** A screen that drew both would pass a test that only looked for the new
// line, and "waiting on the other player" on a spectator's screen is the exact
// lie this mode exists to remove: a watcher is one of nobody's two players and is
// waiting on both of them.
func TestAWatchedBattleSaysNothingAboutTheReaderTakingATurn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		watched := aWatchedBattle(t, c, 3)
		drawn, footer := watched.View(c)
		if want := c.Text(i18n.PlayWatchWaiting); !strings.Contains(drawn, want) {
			t.Errorf("in %s a watched battle draws no line saying it is being watched:\n%s",
				lang, drawn)
		}
		if unwanted := c.Text(i18n.PlayLiveWaiting); strings.Contains(drawn, unwanted) {
			t.Errorf("in %s a watched battle tells the reader it is waiting on the other "+
				"player, and the reader is neither of them:\n%s", lang, drawn)
		}
		if want := c.Text(i18n.PlayWatchFooter); footer != want {
			t.Errorf("in %s a watched battle draws the footer %q, want %q", lang, footer, want)
		}

		// And the same screen between battles of a series, which is the second
		// site the sentence is drawn from and the one a single reading misses.
		between := NewPlayScreen().Attach(c, PlayLive{Watching: true})
		body, _ := between.View(c)
		if want := c.Text(i18n.PlayWatchWaiting); !strings.Contains(body, want) {
			t.Errorf("in %s a watcher between battles draws no line saying so:\n%s", lang, body)
		}
		if unwanted := c.Text(i18n.PlayLiveWaiting); strings.Contains(body, unwanted) {
			t.Errorf("in %s a watcher between battles is told it is waiting on the other "+
				"player:\n%s", lang, body)
		}

		// The finished state keeps the live over footer, which is the one live
		// wording that is already true of a spectator: both keys it names are
		// there and `esc` really does lead to the result.
		over := aWatchedBattleOver(t, c)
		if _, ended := over.View(c); ended != c.Text(i18n.PlayLiveOverFooter) {
			t.Errorf("in %s a finished watched battle draws %q, want the live over footer %q",
				lang, ended, c.Text(i18n.PlayLiveOverFooter))
		}
	}
}

// TestTheWatchingClockNamesTheHalvesTheBoardLabels holds the one thing about
// those two wordings a reader cannot check: the letters in them are the letters
// on the board.
//
// ⚠️ **They are spelled inside the wording rather than passed in**, because the
// alternative is a format taking four arguments where two of them are always the
// same two characters. What that costs is a second place `A` and `E` are written
// down, so this is what makes a rename in tui.Tags reach them — without it the
// clock would go on naming halves the board had stopped labelling that way, in
// both languages, with every golden green.
func TestTheWatchingClockNamesTheHalvesTheBoardLabels(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		watched := aWatchedBattle(t, c, 3)
		tags := tui.Tags(watched.Fight.Units())
		letters := map[hex.Side]string{}
		for _, unit := range watched.Fight.Units() {
			tag := tags[unit.ID]
			if tag == "" {
				t.Fatalf("the unit %q has no tag, so the board labels nothing", unit.ID)
			}
			letters[unit.Side] = tag[:1]
		}
		if len(letters) != 2 {
			t.Fatalf("the board labels %d sides, so one of the two clocks is unmeasured", len(letters))
		}
		ally, enemy := letters[hex.SideAlly], letters[hex.SideEnemy]
		for _, test := range []struct {
			waiting PlayClockSeat
			want    i18n.Key
		}{
			{PlayClockYou, i18n.PlayWatchTurnAlly},
			{PlayClockThem, i18n.PlayWatchTurnEnemy},
		} {
			state := watched
			state.Clock = aCountdown(test.waiting)
			drawn := state.clocks(c)
			if drawn != c.Text(test.want, playClock(state.Clock.Yours), playClock(state.Clock.Theirs)) {
				t.Errorf("in %s the watched clock for %v drew %q, which is not the wording it "+
					"is supposed to be", lang, test.waiting, drawn)
			}
			for _, letter := range []string{ally, enemy} {
				if !strings.Contains(drawn, letter) {
					t.Errorf("in %s the watched clock %q does not name the half the board "+
						"labels %q", lang, drawn, letter)
				}
			}
			// And it says nothing about a `you`, which is what the live pair says
			// and the whole reason these two exist.
			if strings.Contains(drawn, c.Text(i18n.PlayClockYours, "0:00", "0:00")) {
				t.Errorf("in %s the watched clock drew the live wording: %q", lang, drawn)
			}
		}
		// A watching screen with no turn being counted draws no clock, exactly as
		// a live one does: nought is nobody.
		none := watched
		none.Clock = PlayClock{}
		if drawn := none.clocks(c); drawn != "" {
			t.Errorf("in %s a watched battle with no open turn drew the clock %q", lang, drawn)
		}
	}
}

// watchingKeysSwept is how many keystrokes the sweep below sends.
//
// ⚠️ **It is written down so a walk over an empty list cannot pass.** everyKeyHere
// is derived from a slice plus a string of letters, and a fixture that came back
// with nothing would make every assertion under it vacuously true — which is the
// shape of defect this repository has paid for more than once. The number is
// asserted rather than logged, so widening the vocabulary is a decision somebody
// takes here rather than a silent change in what "every key" means.
const watchingKeysSwept = 56

// TestNoKeyAWatcherPressesReachesTheMatch is the sweep decision three asked for:
// every keystroke this suite can send, on every state a spectator can be in, with
// the match asserted untouched afterwards.
//
// What "untouched" means is three separate readings, because each of them catches
// a different way of getting this wrong:
//
//   - **The battle was not stepped.** battle.Recorded is how many events the
//     engine has produced, so a screen that called Act, Pass or Advance moves it.
//     This is the reading that matters most: the battle behind a watching screen
//     is the *mirror's*, and another goroutine is stepping it from the wire.
//   - **No decision left the screen.** An Action of kind Answer is how a live
//     screen sends one, so a watcher that produced one would be a spectator
//     putting a move into somebody else's match. Back is allowed and is the only
//     thing that is: esc leaves.
//   - **The screen did not start offering a turn.** Pending, Answered and Aiming
//     stay where they were, so no key can walk a watcher into the option list or
//     the aim list by a route the drawing does not show.
//
// ⚠️ **The log keys really do something and that is the point rather than an
// exception.** Reading back through the history is the one thing a spectator is
// there to do, so `[`, `]`, `pgup` and `pgdown` are expected to move the frame —
// and the assertion is about the *match*, not about the drawing, which is why
// they need no exemption here at all.
func TestNoKeyAWatcherPressesReachesTheMatch(t *testing.T) {
	keys := everyKeyHere()
	if len(keys) != watchingKeysSwept {
		t.Fatalf("the sweep sends %d keys and is written down as %d: widening the vocabulary "+
			"is a decision, not a silent change to what \"every key\" means",
			len(keys), watchingKeysSwept)
	}
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		states := map[string]PlayScreen{
			"a turn on the board": aWatchedBattle(t, c, 3),
			"over":                aWatchedBattleOver(t, c),
		}
		scrolled := playing(t, c, states["over"], "pgup")
		if scrolled.LogFollow {
			t.Fatalf("in %s the watched log would not scroll back, so the scrolled state is "+
				"the same state twice", lang)
		}
		states["scrolled back"] = scrolled

		for where, state := range states {
			if state.Fight == nil {
				t.Fatalf("the %s state holds no battle, so nothing below is measured", where)
			}
			recorded := state.Fight.Recorded()
			for _, name := range keys {
				next, action := state.Update(c, press(t, name))
				switch action.Kind {
				case Stay, Back, Raise:
				default:
					t.Errorf("in %s %q on a watched battle (%s) answered with a %s action, "+
						"and the only thing a spectator's keystroke may ask for is a way out",
						lang, name, where, action.Kind)
				}
				if action.Kind == Raise {
					t.Errorf("in %s %q raised %v on a watched battle (%s); the description "+
						"screen is raised off the option under the cursor and a watcher has "+
						"no option list", lang, name, action.Target, where)
				}
				if got := state.Fight.Recorded(); got != recorded {
					t.Fatalf("in %s %q on a watched battle (%s) stepped the battle: %d events "+
						"before and %d after — and that battle belongs to the mirror",
						lang, name, where, recorded, got)
				}
				if next.Pending != state.Pending || next.Answered != state.Answered ||
					next.Aiming != state.Aiming {
					t.Errorf("in %s %q on a watched battle (%s) opened a decision: pending %v→%v, "+
						"answered %v→%v, aiming %v→%v", lang, name, where,
						state.Pending, next.Pending, state.Answered, next.Answered,
						state.Aiming, next.Aiming)
				}
				if next.Watching != state.Watching || next.Live != state.Live {
					t.Errorf("in %s %q on a watched battle (%s) changed the mode: live %v→%v, "+
						"watching %v→%v", lang, name, where,
						state.Live, next.Live, state.Watching, next.Watching)
				}
				if next.Err != nil {
					t.Errorf("in %s %q on a watched battle (%s) failed: %v",
						lang, name, where, next.Err)
				}
				if len(next.Notes) > 0 {
					t.Errorf("in %s %q on a watched battle (%s) wrote a file: %v",
						lang, name, where, next.Notes)
				}
			}
		}
	}
}

// TestTheWatchingFootersNameNoKeyTheScreenIgnores is
// TestTheLiveFootersNameNoKeyTheScreenIgnores over the two footers a spectator
// reads, and it is the half of decision three that a key sweep cannot state:
// a key that does nothing never appears in the derived set, so a footer naming
// one is only ever caught from this direction.
//
// *Sees:* a watching footer promising enter, `?`, `p` or ↑/↓; a key a spectator
// can press that no footer names; a footer over the floor.
// *Cannot see:* whether a named key does the right thing — that is the sweep
// above, which asserts it does nothing at all.
func TestTheWatchingFootersNameNoKeyTheScreenIgnores(t *testing.T) {
	named := map[string][]string{
		"↑/↓":   {"up", "down", "k", "j"},
		"[/]":   {"pgup", "pgdown", "[", "]"},
		"enter": {"enter", "space"},
	}
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		states := map[string]PlayScreen{
			"a turn on the board": aWatchedBattle(t, c, 3),
			"over":                aWatchedBattleOver(t, c),
		}
		scrolled := playing(t, c, states["over"], "pgup")
		if scrolled.LogFollow {
			t.Fatalf("in %s the watched log would not scroll back, so the forward half of "+
				"[/] is driven by nothing", lang)
		}
		driven := map[string]PlayScreen{"scrolled back": scrolled}
		for where, state := range states {
			driven[where] = state
		}

		announced := map[string]bool{}
		for where, state := range states {
			_, footer := state.View(c)
			if width := lipgloss.Width(footer); width > MinWidth-1 {
				t.Errorf("the watched %s footer in %s is %d cells over the %d the floor "+
					"leaves:\n%s", where, lang, width, MinWidth-1, footer)
			}
			t.Logf("the watched %s footer in %s is %d cells: %s",
				where, lang, lipgloss.Width(footer), footer)
			for _, key := range keysNamedIn(footer) {
				for _, spelled := range expand(key, named) {
					announced[spelled] = true
				}
			}
		}
		if len(announced) == 0 {
			t.Fatalf("no watched footer in %s names a key, so both directions below are "+
				"satisfied by nothing", lang)
		}

		answered := map[string]bool{}
		for _, name := range everyKeyHere() {
			for _, state := range driven {
				rest := fingerprintLive(c, state, Action{})
				next, action := state.Update(c, press(t, name))
				if fingerprintLive(c, next, action) != rest {
					answered[name] = true
				}
			}
		}
		delete(announced, "ctrl+l")

		if missing := difference(answered, announced); len(missing) > 0 {
			t.Errorf("a watched battle in %s answers %v, which no watching footer names",
				lang, missing)
		}
		if promised := difference(announced, answered); len(promised) > 0 {
			t.Errorf("the watching footers in %s name %v, which a watched battle ignores",
				lang, promised)
		}
		// And by name, the four the live footer carries and this one may not: a
		// key that does nothing never reaches `answered`, so a clause left in
		// would only ever show up in the direction above — and this says which
		// clause, rather than leaving the reader to work out why `enter` is in a
		// list.
		for _, dropped := range []string{"enter", "space", "?", "p", "up", "down", "k", "j"} {
			if announced[dropped] {
				t.Errorf("a watching footer in %s names %q, which a spectator's screen "+
					"ignores: the decision is not this client's to take and there is no "+
					"option list under the cursor to describe", lang, dropped)
			}
		}
	}
}

// TestAWatchedBattleDrawsTheSameBoardAsAPlayerOfIt is the other half of "a mode
// rather than a screen of its own", asserted rather than argued: everything a
// spectator reads is the drawing a player reads, and the difference is the rows
// that address the reader.
//
// ⚠️ **The comparison is over the sections rather than over the whole body**, and
// it has to be: the tail row and the footer are exactly what differ, so a
// body-to-body comparison would be a test that could only ever fail.
func TestAWatchedBattleDrawsTheSameBoardAsAPlayerOfIt(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	local := withAFullLog(t, c, atABattleOf(t, c, 3))
	reading := PlayLive{Fight: local.Fight, Side: local.Side, Seed: local.Seed}
	playing := NewPlayScreen().Attach(c, reading)
	reading.Watching = true
	watching := NewPlayScreen().Attach(c, reading)
	if playing.Pending != nil {
		// Both are handed a nil Asking, so neither is being asked and the two
		// draw the same tail. That is what makes the sections comparable.
		t.Fatal("the player's screen holds a turn the watcher's cannot, so the two are not "+
			"the same drawing with different words", playing.Pending)
	}
	if got, want := watching.LogRows(c), playing.LogRows(c); !slices.Equal(got, want) {
		t.Errorf("a watcher reads %d rows of history and a player %d", len(got), len(want))
	}
	if got, want := watching.read(c).board, playing.read(c).board; got != want {
		t.Errorf("a watcher's board is not the player's:\n%s\n%s", got, want)
	}
	if got, want := watching.read(c).roster, playing.read(c).roster; got != want {
		t.Errorf("a watcher's roster is not the player's:\n%s\n%s", got, want)
	}
	if got, want := watching.read(c).order, playing.read(c).order; got != want {
		t.Errorf("a watcher's queue line is not the player's:\n%s\n%s", got, want)
	}
}

// TestAWatchedBattleRefusesToBeOpenedLocally is the guard Open already carries,
// asserted for the mode that arrived after it: a client whose menu still offered
// a battle while a match was being watched would build a second one over the
// mirror's, and the reader would be playing a hot seat game against themselves
// while two other people fought the one on screen.
func TestAWatchedBattleRefusesToBeOpenedLocally(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	watched := aWatchedBattle(t, c, 3)
	squad := aSquadOfSide(t, c, 3)
	reopened := watched.Open(c, squad, squad.Clone())
	if reopened.Fight != watched.Fight {
		t.Error("a watched battle was opened on a local pairing, so the screen is now driving " +
			"a battle of its own over the top of the one it is watching")
	}
	if !reopened.Watching {
		t.Error("opening a watched battle took the mode off it")
	}
}

// TestAWatchedBattleTakesEveryEventTheMirrorProduces is the reading half: a
// spectator is a renderer over a battle somebody else steps, so the events it
// draws have to be the ones the engine emitted, in order and with none missed.
//
// It steps the battle the way a room does between Attach calls, which is exactly
// the shape a match has: the client is handed a wire.Turn, the mirror applies it,
// and the screen is re-attached to whatever the mirror now holds.
func TestAWatchedBattleTakesEveryEventTheMirrorProduces(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	fight, prompt := aBattleNobodyHereDrives(t, c, 3)
	watched := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Watching: true})
	for range 12 {
		if fight.Finished() {
			break
		}
		prompt = steppedByTheRoom(t, fight, prompt)
		watched = watched.Attach(c, PlayLive{Fight: fight, Watching: true})
	}
	whole, _ := fight.Since(0)
	if len(watched.Events) != len(whole) {
		t.Fatalf("the watched screen holds %d events and the battle has produced %d",
			len(watched.Events), len(whole))
	}
	for at := range whole {
		if !sameEvent(watched.Events[at], whole[at]) {
			t.Fatalf("event %d differs: the screen holds %v and the battle emitted %v",
				at, watched.Events[at], whole[at])
		}
	}
	if len(whole) == 0 {
		t.Fatal("the battle produced no events, so nothing above is measured")
	}
}

// sameEvent compares the fields a renderer reads, which is what "the same event"
// means to a screen.
func sameEvent(a, b battle.Event) bool {
	return a.Kind == b.Kind && a.Actor == b.Actor && a.Target == b.Target &&
		a.Skill == b.Skill && a.Amount == b.Amount && a.Turn == b.Turn
}
