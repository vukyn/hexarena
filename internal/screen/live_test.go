package screen

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The battle screen over a battle it does not drive
//
// A PvP client holds **one** engine and it is the mirror's: the mirror steps it
// from the wire.Turn that comes back, deliberately, so a screen that also called
// Fight.Act would step the same battle twice and diverge from the room on turn
// one. Live mode is the whole of what that costs this screen — a flag, a cursor
// of its own, and six keys answered differently — and the four tests below are
// what hold each half of it.
//
// Nothing here needs a socket. What live mode is *about* is two consumers of one
// battle, and a test can be the second consumer as easily as a mirror can.

// aBattleNobodyHereDrives is a battle built through the local path and then
// handed to a live screen, which is exactly the shape a mirror produces: an
// engine somebody else owns, and a screen reading it through a cursor of its
// own.
//
// ⚠️ It hands back the **driver** as well, because the whole point is that the
// battle is stepped from outside this screen and a test with no way to step it
// would be measuring a still board.
func aBattleNobodyHereDrives(t *testing.T, c Context, side int) (*battle.Battle, *battle.Prompt) {
	t.Helper()
	local := atABattleOf(t, c, side)
	if local.Pending == nil {
		t.Fatal("the battle opened with no turn for anybody, so there is nothing to attach to")
	}
	return local.Fight, local.Pending
}

// steppedByTheRoom takes one turn the way a room does — the engine's own rating,
// applied to the engine — and hands back the prompt it stopped on.
//
// It drives the **battle** rather than any screen, which is what makes it the
// second driver: a screen stepping it would be the very defect these tests are
// about.
func steppedByTheRoom(t *testing.T, fight *battle.Battle, prompt *battle.Prompt) *battle.Prompt {
	t.Helper()
	if prompt == nil {
		t.Fatal("no turn is open, so the room has nothing to take")
	}
	choice, acted := fight.Suggest(prompt)
	var err error
	if acted {
		err = fight.Act(choice.Skill, choice.Aim)
	} else {
		err = fight.Pass(battle.NoActionReason)
	}
	if err != nil {
		t.Fatalf("the room could not take %q's turn: %v", prompt.Unit, err)
	}
	if fight.Finished() {
		return nil
	}
	opened, err := fight.Advance()
	if err != nil {
		t.Fatalf("the room could not open the next turn: %v", err)
	}
	return opened
}

// TestALiveBattleTakesNoTurnOfItsOwn is the "two drivers over one battle" defect
// itself, pressed rather than reasoned about.
//
// Every key that spends a turn locally is pressed on a live screen and the
// battle's own record is held still across all of them. That record is the right
// thing to watch: Act, Pass, Advance, Replay and Suggest all leave events
// behind, so a surviving call into any of them moves it.
//
// *Sees:* a guard deleted from any of the six keys — n, u, the save key, a,
// enter and p — because each of them steps or writes on the local path.
// *Cannot see:* whether the answer reaches the wire. That is the client's
// TestAJoinedMatchPlaysToItsEndOverALoopbackListener.
func TestALiveBattleTakesNoTurnOfItsOwn(t *testing.T) {
	c, _ := start(t, i18n.En)
	fight, prompt := aBattleNobodyHereDrives(t, c, 3)
	live := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt, Seed: 7})
	if !live.Live || live.Pending == nil {
		t.Fatal("the screen did not attach to the open turn, so no key below is being " +
			"pressed on a live battle")
	}

	recorded := fight.Recorded()
	if recorded == 0 {
		t.Fatal("the battle has recorded nothing, so a record that does not move proves nothing")
	}
	for _, name := range []string{"n", "u", "ctrl+s", "a"} {
		after, action := asking(t, c, live, name)
		if got := fight.Recorded(); got != recorded {
			t.Errorf("%q on a live battle moved the record from %d to %d events, so this "+
				"screen stepped a battle it does not drive", name, recorded, got)
		}
		if len(after.Script) != 0 {
			t.Errorf("%q on a live battle wrote %d decisions down; a live screen keeps no "+
				"script, because undo is off and the room writes the log", name, len(after.Script))
		}
		if len(after.Notes) != 0 {
			t.Errorf("%q on a live battle left a save note behind", name)
		}
		if action.Kind != Stay {
			t.Errorf("%q on a live battle asked for a %v, want nothing", name, action.Kind)
		}
		if after.Seed != live.Seed {
			t.Errorf("%q on a live battle moved the seed to %d; a match's seeds are the "+
				"room's", name, after.Seed)
		}
	}

	// And the two that DO answer: an Action carrying the decision, and still no
	// step.
	option := live.Pending.Options[live.Option]
	struck, action := asking(t, c, live, "enter")
	for action.Kind == Stay && struck.Aiming {
		// A skill with several cells asks where first, which is the same two
		// questions the local screen asks. The second enter is the decision.
		struck, action = asking(t, c, struck, "enter")
	}
	if action.Kind != Answer {
		t.Fatalf("enter on a live battle asked for a %v, want an Answer", action.Kind)
	}
	if !action.Answer.Acted {
		t.Error("enter on a live battle answered with a pass")
	}
	if action.Answer.Choice.Skill != option.Skill {
		t.Errorf("enter answered with %q, want the option under the cursor %q",
			action.Answer.Choice.Skill, option.Skill)
	}
	if !slices.Contains(option.Aims, action.Answer.Choice.Aim) {
		t.Errorf("enter answered aiming at %v, which is not one of the cells the option offers",
			action.Answer.Choice.Aim)
	}
	if got := fight.Recorded(); got != recorded {
		t.Errorf("enter on a live battle moved the record from %d to %d events", recorded, got)
	}
	if !struck.Answered {
		t.Error("a live battle that answered did not mark the turn answered, so it would " +
			"offer the same one again")
	}

	passed, action := asking(t, c, live, "p")
	if action.Kind != Answer {
		t.Fatalf("p on a live battle asked for a %v, want an Answer", action.Kind)
	}
	if action.Answer.Acted {
		t.Error("p on a live battle answered with a strike")
	}
	if action.Answer.Choice != (battle.Choice{}) {
		t.Errorf("a pass carried %+v; wire.Pass carries nothing at all, and the reason "+
			"lives on battle.Decision", action.Answer.Choice)
	}
	if got := fight.Recorded(); got != recorded {
		t.Errorf("p on a live battle moved the record from %d to %d events", recorded, got)
	}
	if !passed.Answered {
		t.Error("a live battle that passed did not mark the turn answered")
	}

	// A second press in the same turn is dropped, which is what Answered is for.
	again, action := asking(t, c, passed, "enter")
	if action.Kind != Stay {
		t.Errorf("a second enter in the same turn asked for a %v; one decision per turn",
			action.Kind)
	}
	_ = again
}

// TestALiveBattleReadsTheRecordWithoutDrainingIt is the second cursor.
//
// ⚠️ **Drain is not exclusive and it is not free either.** It is Since(b.drained)
// over an append-only record, so two consumers of one battle is exactly what the
// record-and-cursor work bought — but it *writes* b.drained, and a live battle is
// read under a lock that admits several readers. So live mode keeps its own
// cursor and calls Since, which is a pure read.
//
// The measurement is in two halves and both are needed. The screen's history has
// to grow by the run the battle actually produced (or the cursor is not being
// advanced), and a Drain afterwards has to still hand back that whole run (or the
// cursor was b.drained all along and the mirror's own reading has been eaten).
//
// *Sees:* live mode reaching for Drain, and a cursor that never moves.
// *Cannot see:* the lock itself — that is internal/socket's
// TestAMirrorIsSafeToDrawWhileItIsStepped.
func TestALiveBattleReadsTheRecordWithoutDrainingIt(t *testing.T) {
	c, _ := start(t, i18n.En)
	fight, prompt := aBattleNobodyHereDrives(t, c, 3)
	// Everything the local build already drained is behind us; from here the
	// battle's own drain position is what this test is about.
	fight.Drain()

	// ⚠️ **The first attach reads the WHOLE record and that is right**, which was
	// measured here rather than guessed: a live screen's cursor starts at nought,
	// not at the battle's drain position, because a renderer that joined a battle
	// part-way through would draw a log with no beginning. It is the mirror's own
	// reading — Mirror.open starts its cursor after the opening board because the
	// first *digest* covers the first decision, and a screen is not digesting
	// anything.
	live := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt})
	if got, want := len(live.Events), fight.Recorded(); got != want {
		t.Fatalf("the first attach read %d of the battle's %d events; a live screen's "+
			"history starts at the beginning of the battle", got, want)
	}
	if len(live.Events) == 0 {
		t.Fatal("the battle has recorded nothing, so neither half below measures anything")
	}

	before := fight.Recorded()
	for range 4 {
		prompt = steppedByTheRoom(t, fight, prompt)
		if prompt == nil {
			break
		}
	}
	produced := fight.Recorded() - before
	if produced == 0 {
		t.Fatal("four turns produced no events, so neither half below measures anything")
	}

	held := len(live.Events)
	live = live.Attach(c, PlayLive{Fight: fight, Asking: prompt})
	if got := len(live.Events) - held; got != produced {
		t.Errorf("the second attach read %d events over the %d the battle produced, so "+
			"this screen's own cursor is not where it thinks it is", got, produced)
	}

	drained := fight.Drain()
	if len(drained) != produced {
		t.Errorf("a Drain after two live attaches hands back %d events over the %d "+
			"produced, so live mode moved the battle's own drain position — which is a "+
			"write, under a read lock, taking the mirror's events away from it",
			len(drained), produced)
	}
}

// TestALiveBattleSaysTheOtherPlayerIsDeciding is the one drawn line live mode
// adds.
//
// ⚠️ **Locally an empty tail means "the engine's own units act", which resolves
// in microseconds.** In a match it is the other player thinking, for up to the
// whole ninety-second allowance, and a screen with nothing where the moves go
// reads as frozen. It covers the answered turn too, because from this side of
// the wire the two are one state: the decision has gone and the board is waiting
// on the other end.
//
// *Sees:* the state a sweep entry would otherwise record as an ordinary battle
// with a blank where the option list was.
func TestALiveBattleSaysTheOtherPlayerIsDeciding(t *testing.T) {
	c, _ := start(t, i18n.En)
	fight, prompt := aBattleNobodyHereDrives(t, c, 3)

	waiting := NewPlayScreen().Attach(c, PlayLive{Fight: fight})
	if fight.Finished() {
		t.Fatal("the battle is over, so the waiting line is drawn for the wrong reason")
	}
	drawn, _ := waiting.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PlayLiveWaiting)) {
		t.Errorf("a live battle with no turn of its own says nothing about waiting:\n%s", drawn)
	}

	// The same line once this side has answered, which is the half a test over
	// the nil prompt alone would miss.
	answered, action := asking(t, c,
		NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt}), "p")
	if action.Kind != Answer {
		t.Fatalf("p asked for a %v, so the answered state was never reached", action.Kind)
	}
	drawn, _ = answered.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PlayLiveWaiting)) {
		t.Errorf("a live battle that has answered still offers the turn it answered:\n%s", drawn)
	}

	// And the local screen must NOT have grown it, which is the half that keeps
	// the hot-seat goldens still.
	local := atABattleOf(t, c, 3)
	drawn, _ = local.View(c)
	if strings.Contains(drawn, c.Text(i18n.PlayLiveWaiting)) {
		t.Errorf("a local battle draws the live waiting line, so the guard leaked:\n%s", drawn)
	}
}

// TestALiveBattleDrawsTheRefusalItWasSent is the one place three of the ten
// protocol refusals can ever be read.
//
// not-your-turn, illegal-action and unknown-message arrive **during** a match —
// a refusal does not end a connection — so no lobby screen can show them and
// nothing else would.
//
// *Sees:* the notes slot being spent on something else on a live screen, and a
// name that never reaches the language book.
// *Cannot see:* whether the client hands over the *latest* refusal. That is the
// client's own liveOf.
func TestALiveBattleDrawsTheRefusalItWasSent(t *testing.T) {
	c, _ := start(t, i18n.En)
	fight, prompt := aBattleNobodyHereDrives(t, c, 3)
	const refused = "not_your_turn"
	live := NewPlayScreen().Attach(c,
		PlayLive{Fight: fight, Asking: prompt, Refusal: refused})
	want := c.Lang.Refusal(refused)
	if want == refused {
		t.Fatalf("the language book leaves %q at its own id, so the assertion below "+
			"would pass on a screen drawing the raw name", refused)
	}
	drawn, _ := live.View(c)
	// The sentence is wrapped, so the opening of it is what a line holds.
	opening := WrapWords(want, MinWidth-1)[0]
	if !strings.Contains(drawn, opening) {
		t.Errorf("a live battle sent %q draws nothing about it:\n%s", refused, drawn)
	}
	// And a live screen with no refusal draws none of it, which is what stops
	// this passing on a screen that always drew something.
	clean, _ := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt}).View(c)
	if strings.Contains(clean, opening) {
		t.Errorf("a live battle nobody refused draws a refusal:\n%s", clean)
	}
}

// TestTheLiveFootersNameNoKeyTheScreenIgnores is readonly_test.go's four-part
// claim, taken over the three live footers.
//
// A key announced on a screen that ignores it is worse than one nobody was told
// about, so the list is **derived** rather than written down: every key this
// suite can send is pressed on a live screen, the ones that do something are
// collected, and the two sets are held equal in both directions.
//
// ⚠️ **The footers name pairs and aliases rather than keys**, which is the one
// place a table is unavoidable: `↑/↓` stands for four keystrokes and `[/]` for
// four more, and those spellings are the ones every other footer in the catalog
// already uses. The table below is therefore a **declaration** and is the thing
// this test cannot see past — what it can see is a key on neither side of it.
//
// *Sees:* a footer promising u, n or the save key on a live battle; a live key
// nobody was told about; a footer that outgrew the floor.
// *Cannot see:* whether a named key does the right thing — that is the first
// test in this file.
func TestTheLiveFootersNameNoKeyTheScreenIgnores(t *testing.T) {
	// The spellings a footer uses for a group of keys, which is the vocabulary
	// every footer in this catalog is written in.
	named := map[string][]string{
		"↑/↓":   {"up", "down", "k", "j"},
		"[/]":   {"pgup", "pgdown", "[", "]"},
		"enter": {"enter", "space"},
	}
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		fight, prompt := aBattleNobodyHereDrives(t, c, 3)

		// The three states, each with the footer it draws.
		states := map[string]PlayScreen{}
		states["a turn"] = NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt})
		aiming := states["a turn"]
		aiming.Aiming = true
		states["aiming"] = aiming
		over := aFinishedLiveBattle(t, c)
		states["over"] = over

		// ⚠️ **A fourth state that draws no footer of its own, and it is not
		// padding.** The forward half of the log's pair does nothing on a log
		// that is already following its tail, so a derivation over the three
		// above finds `[` and `pgup` and misses `]` and `pgdown` — and would
		// then report the footer as promising two keys the screen ignores. The
		// screen does not ignore them; the fixture was standing at the bottom.
		scrolled := playing(t, c, over, "pgup")
		if scrolled.LogFollow {
			t.Fatalf("in %s the finished battle's log would not scroll back, so the "+
				"forward half of [/] is driven by nothing", lang)
		}
		driven := map[string]PlayScreen{"scrolled back": scrolled}
		for where, state := range states {
			driven[where] = state
		}

		announced := map[string]bool{}
		for where, state := range states {
			_, footer := state.View(c)
			if width := lipgloss.Width(footer); width > MinWidth-1 {
				t.Errorf("the live %s footer in %s is %d cells over the %d the floor leaves:\n%s",
					where, lang, width, MinWidth-1, footer)
			}
			t.Logf("the live %s footer in %s is %d cells", where, lang, lipgloss.Width(footer))
			for _, key := range keysNamedIn(footer) {
				for _, spelled := range expand(key, named) {
					announced[spelled] = true
				}
			}
		}
		if len(announced) == 0 {
			t.Fatalf("no live footer in %s names a key, so both directions below are "+
				"satisfied by nothing", lang)
		}

		// Every key that does something on a live screen, derived by pressing it.
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
		// ctrl+l is the client's, not the screen's: the language toggle never
		// reaches a screen's Update, and every footer in the catalog names it.
		delete(announced, "ctrl+l")

		missing := difference(answered, announced)
		promised := difference(announced, answered)
		if len(missing) > 0 {
			t.Errorf("a live battle in %s answers %v, which no live footer names", lang, missing)
		}
		if len(promised) > 0 {
			t.Errorf("the live footers in %s name %v, which a live battle ignores", lang, promised)
		}
		// And the three name none of the keys live mode turns off, which is the
		// half a derivation cannot state: those keys do nothing, so they would
		// never appear in `answered` and a footer naming one would be caught only
		// by the `promised` direction — which is exactly what this asserts, by
		// name, so the failure says *which* clause was left in.
		for _, dropped := range []string{"u", "n", "a", "ctrl+s"} {
			if announced[dropped] {
				t.Errorf("a live footer in %s names %q, which a live battle ignores because "+
					"a match's seeds are the room's, the opponent has already seen what undo "+
					"would take back, the decision is the player's, and the room writes the log",
					lang, dropped)
			}
		}
	}
}

// aFinishedLiveBattle is a live screen over a battle that has ended, which is
// the state the over footer belongs to.
func aFinishedLiveBattle(t *testing.T, c Context) PlayScreen {
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
		t.Fatal("the battle never ended, so the over footer is drawn by nothing")
	}
	return NewPlayScreen().Attach(c, PlayLive{Fight: fight})
}

// everyKeyHere is every keystroke this suite can send, which is what a derived
// list of "keys that do something" has to be taken over.
func everyKeyHere() []string {
	// ⚠️ It is what this package's own `press` can build, and nothing more: a
	// name it cannot build fails the helper rather than being silently skipped,
	// so widening the list is a decision rather than an accident. ctrl+l is
	// deliberately absent — the language toggle is the *client's* chord and never
	// reaches a screen's Update at all.
	keys := []string{
		"up", "down", "left", "right", "enter", "esc", "tab", "shift+tab",
		"space", "pgup", "pgdown", "backspace", "ctrl+s", "ctrl+x",
	}
	for _, letter := range "abcdefghijklmnopqrstuvwxyz0123456789/?[]+-" {
		keys = append(keys, string(letter))
	}
	slices.Sort(keys)
	return keys
}

// keysNamedIn is the keystrokes a footer advertises: clauses split on the middle
// dot both languages point a list with, and the key is each clause's first word.
//
// Read off the rendered line rather than off a second declaration of what a
// footer holds, which is the only way this can disagree with the wording an
// author actually typed.
func keysNamedIn(footer string) []string {
	var out []string
	for _, clause := range strings.Split(footer, "·") {
		if fields := strings.Fields(clause); len(fields) > 0 {
			out = append(out, fields[0])
		}
	}
	return out
}

// expand turns a footer's spelling into the keystrokes it stands for.
func expand(spelled string, named map[string][]string) []string {
	if keys, grouped := named[spelled]; grouped {
		return keys
	}
	return []string{spelled}
}

// fingerprintLive is a live screen and what it asked for, as one comparable
// string. The Action is in it because one of the answers changes nothing drawn:
// a pass on a turn already answered leaves the same body.
func fingerprintLive(c Context, p PlayScreen, action Action) string {
	body, footer := p.View(c)
	return action.Kind.String() + "\x00" + body + "\x00" + footer
}

// difference is what is in the first set and not the second, sorted so a failure
// reads the same way twice.
func difference(from, without map[string]bool) []string {
	var out []string
	for key := range from {
		if !without[key] {
			out = append(out, key)
		}
	}
	slices.Sort(out)
	return out
}
