package screen

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The arrange screen, tested with no protocol anywhere near it
//
// Every fixture here builds a DraftLive by hand, for draft_test.go's own reason:
// the screen cannot see internal/socket, so a test here cannot drive it through a
// mirror and does not want to. What holds the whole vertical — two clients
// drafting, arranging and fighting to a finish over a loopback listener — is
// cmd/hexarena-tui, which is the package that owns both vocabularies.

// TestNoKeystrokeOnTheDraftScreenTakesAnArrangement is the assertion a **mode**
// would have deleted, and it is why the arrangement is a screen of its own.
//
// ⚠️ **It moved here from the client at step 5c.** The end-to-end draft test used
// to carry it, because the arrange phase was where a drafting match stopped; that
// test now plays through the phase and out the other side, so the guarantee needs
// a home that does not depend on the match stopping. Here it holds wherever the
// screen is drawn.
//
// What it protects: the draft screen goes on holding its last reading through the
// whole arrange phase, so every key it answers is a key pressed on a screen that
// is no longer the decision in front. `enter` on a candidate and `s` on a ban are
// the two that take a decision at all.
func TestNoKeystrokeOnTheDraftScreenTakesAnArrangement(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepBan, true)
	live := screen.Live
	live.Step, live.Arranging, live.Yours, live.OnTurn = DraftStepArrange, true, true, ""
	live.Candidates = nil
	live.Recorded = 10
	screen = screen.Attach(c, live)
	for _, name := range []string{"enter", "s", "up", "down", "left", "right", "u"} {
		_, result := screen.Update(c, press(t, name))
		if result.Decided {
			t.Errorf("%q on the draft screen took a %s during the arrange phase, which is the "+
				"arrange screen's decision and no other screen's", name, result.Decision.Step)
		}
	}
}

// TestTheArrangeScreenDrawsTheWholeBoard is the drawing decision this step turns
// on, and the reason is a measurement rather than symmetry.
//
// ⚠️ Placement here is **purely defensive** and its whole value is rank depth:
// the shipped roster's aces moved from the front column to the back read 27.6% →
// 47.3% ally over 4000 seeds, and splitting the screening pair read 31.1% against
// the adjacent pair's 47.3%. A player who cannot see which rank a cell is cannot
// take that decision, so what is asserted is both halves of the board — every
// far-side cell marked as the other side's — and the rank named beside the cell
// under the cursor.
func TestTheArrangeScreenDrawsTheWholeBoard(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := anArrangement(t, c, lib, true)
	drawn, _ := screen.View(c)
	// The far half is drawn: nine cells of it, one mark each.
	if marks := strings.Count(screen.board(c), arrangeTheirsMark); marks != len(hex.SideCells(hex.SideEnemy)) {
		t.Errorf("the board carries %d marks for the other half against its %d cells, so this "+
			"is one side's 3x3 rather than the board:\n%s",
			marks, len(hex.SideCells(hex.SideEnemy)), drawn)
	}
	// And it really is hex.Render's own drawing rather than a second one written
	// here, which is what makes the cells the engine's cells.
	if !strings.Contains(drawn, "  "+strings.Split(hex.Render(func(hex.Offset) string {
		return ""
	}), "\n")[0]) {
		t.Errorf("the board is not hex.Render's drawing:\n%s", drawn)
	}
	// The rank of the cell in hand is named, which is the fact the whole board is
	// drawn for. It is asserted through RankLabel rather than against a word, so
	// the claim is "the screen says which rank" and not "the screen says front".
	rank := RankLabel(c, screen.Cursor)
	if rank == "" {
		t.Fatalf("the cursor is on %s, which is on no formation", screen.Cursor)
	}
	if !strings.Contains(drawn, c.Text(i18n.ArrangeCursorAt, screen.Cursor.String(), rank)) {
		t.Errorf("the screen does not say which rank the cursor is on:\n%s", drawn)
	}
}

// TestTheCursorStepsOverACellAlreadyTaken is the squad builder's own answer
// applied to a grid: two units on one cell is not a formation, and a cursor that
// stops on one is a cursor that can be left in a state the send has to refuse.
//
// It walks the whole grid in both axes from a board with a unit already on it, so
// what is measured is every landing rather than the first one.
func TestTheCursorStepsOverACellAlreadyTaken(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := anArrangement(t, c, lib, true)
	screen, _ = screen.Update(c, press(t, "enter"))
	if len(screen.Slots) != 1 {
		t.Fatalf("enter placed %d picks, want the first one", len(screen.Slots))
	}
	standing := screen.Slots[0]
	for _, name := range []string{"up", "down", "left", "right"} {
		walked := screen
		// Twice round each axis, so a cursor that only skips on its first step is
		// caught as well as one that never does.
		for range 2 * hex.FormationCols * hex.FormationRows {
			walked, _ = walked.Update(c, press(t, name))
			if walked.Cursor == standing {
				t.Fatalf("%q put the cursor on %s, where %s already stands",
					name, standing, screen.Picks()[0].Character)
			}
			if RankLabel(c, walked.Cursor) == "" {
				t.Fatalf("%q put the cursor on %s, which is on no formation", name, walked.Cursor)
			}
		}
	}
}

// TestAnArrangementIsSentInPickOrder is the one thing about this decision that
// cannot be seen by looking at the screen.
//
// ⚠️ `Slots[i]` is the cell for that side's **i-th pick** — draft.Arrange has no
// other way to know who a cell is for — so a send that sorted the cells, or
// collected them in the order they were chosen on the board, would field somebody
// else's formation without a word. The cells here are placed so that pick order
// and every plausible ordering of the cells themselves disagree: read down the
// board they would be 0,0 then 1,0 then 2,0, and the picks take them backwards.
func TestAnArrangementIsSentInPickOrder(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := anArrangement(t, c, lib, true)
	wanted := []hex.Offset{{Col: 2, Row: 0}, {Col: 1, Row: 0}, {Col: 0, Row: 0}}
	if len(wanted) != len(screen.Picks()) {
		t.Fatalf("the fixture holds %d picks and this test places %d",
			len(screen.Picks()), len(wanted))
	}
	if slices.IsSortedFunc(wanted, compareCells) {
		t.Fatal("the cells this test places are already in board order, so a send that sorted " +
			"them would pass")
	}
	for _, cell := range wanted {
		screen = walkedTo(t, c, screen, cell)
		screen, _ = screen.Update(c, press(t, "enter"))
	}
	if !screen.Whole() {
		t.Fatalf("only %d of %d picks were placed", len(screen.Slots), len(screen.Picks()))
	}
	next, result := screen.Update(c, press(t, "enter"))
	if !result.Decided {
		t.Fatalf("enter on a whole arrangement took no decision (err %v)", next.Err)
	}
	if result.Decision.Step != DraftStepArrange {
		t.Errorf("the decision is a %s", result.Decision.Step)
	}
	if !slices.Equal(result.Decision.Slots, wanted) {
		t.Errorf("the arrangement went out as %v, want %v in pick order",
			result.Decision.Slots, wanted)
	}
	// And it is a copy: a decision holding the screen's own slice would let a
	// later keystroke edit a decision already sent.
	result.Decision.Slots[0] = hex.Offset{Col: 0, Row: 2}
	if next.Slots[0] != wanted[0] {
		t.Error("editing the decision reached into the screen's own arrangement")
	}
}

// compareCells orders two cells the way a board is read, which is what the test
// above needs in order to prove its own cells are not in that order.
func compareCells(a, b hex.Offset) int {
	if a.Col != b.Col {
		return a.Col - b.Col
	}
	return a.Row - b.Row
}

// TestTwoUnitsOnOneCellIsRefusedInTheRulesOwnWords is the legality rule asked
// once and drawn rather than restated.
//
// ⚠️ **placement.Squad.Validate is the single declaration** — it already refuses
// an empty squad, more than five, a duplicated id, a cell off the 3x3 and two
// units on one cell — so what this asserts is that the screen asks it, keeps its
// words and **blocks the send**. Blocking is not politeness: a refused draft
// decision is left open and re-sent, so an illegal one is an unbounded hot loop.
//
// The state is built by hand because the cursor cannot reach it: the movement
// steps over an occupied cell, which is the guard above this one.
func TestTwoUnitsOnOneCellIsRefusedInTheRulesOwnWords(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := anArrangement(t, c, lib, true)
	shared := hex.Offset{Col: 1, Row: 1}
	screen.Slots = []hex.Offset{shared, shared, {Col: 0, Row: 0}}
	if len(screen.Slots) != len(screen.Picks()) {
		t.Fatalf("the fixture holds %d picks and this state names %d cells",
			len(screen.Picks()), len(screen.Slots))
	}
	next, result := screen.Update(c, press(t, "enter"))
	if result.Decided {
		t.Fatal("an arrangement with two units on one cell was sent, so the room would refuse " +
			"it and this client would send it again")
	}
	if next.Err == nil {
		t.Fatal("the screen says nothing about an arrangement the rule refuses")
	}
	want := next.Squad().Validate()
	if want == nil || next.Err.Error() != want.Error() {
		t.Errorf("the screen says %v where the rule says %v, so this is a second wording of "+
			"one rule", next.Err, want)
	}
	drawn, _ := next.View(c)
	if !strings.Contains(drawn, firstLineOf(c.Lang.Error(next.Err))) {
		t.Errorf("the refusal is not drawn:\n%s", drawn)
	}
	// And the line under the picks does not go on announcing a send that is
	// refused, which is the loadout's own rule: a row promising a keystroke the
	// screen declines is worse than no row.
	if strings.Contains(drawn, c.Text(i18n.ArrangeSendReady)) {
		t.Errorf("the screen says enter sends a formation it refuses:\n%s", drawn)
	}
}

// TestTakingOnePlacementBackFreesItsCell is the key without which a misplaced
// unit could only be answered by letting the allowance run out — and a draft that
// runs out of time is cancelled outright rather than resumed, so the cost of no
// undo here is the whole match.
func TestTakingOnePlacementBackFreesItsCell(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := anArrangement(t, c, lib, true)
	screen, _ = screen.Update(c, press(t, "enter"))
	placed := screen.Slots[0]
	screen, _ = screen.Update(c, press(t, "u"))
	if len(screen.Slots) != 0 {
		t.Fatalf("backspace left %d placements", len(screen.Slots))
	}
	if screen.Cursor != placed {
		t.Errorf("the cursor is on %s and the cell that was freed is %s: a player taking a "+
			"placement back is looking at the cell they took it off", screen.Cursor, placed)
	}
	if _, holding := screen.InHand(); !holding {
		t.Error("nothing is in hand after a placement came back")
	}
}

// TestTheArrangeScreenWordsTheAllowanceRatherThanCountingItDown is the corrected
// reading of the clock, held so that a countdown cannot arrive later.
//
// ⚠️ **The phase's allowance is not one allowance for the phase.**
// Server.settled re-arms off the reading after every batch, so whichever side
// arranges first hands its opponent a **fresh full** allowance and the worst case
// is about twice one. Two readings of this were written into an earlier brief and
// both were wrong. A single countdown on a screen with two decisions open would
// draw the wrong one of them in figures, so the fact is worded — and the reading
// carries a clock, so a screen that drew one would compile and look right.
func TestTheArrangeScreenWordsTheAllowanceRatherThanCountingItDown(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		c, lib := start(t, lang)
		placing := anArrangement(t, c, lib, true)
		states := map[string]ArrangeScreen{
			"placing": placing,
			"sent":    asSent(t, c, placing),
		}
		for name, state := range states {
			drawn, _ := state.View(c)
			if !strings.Contains(drawn, firstLineOf(c.Text(i18n.ArrangeAllowance))) {
				t.Errorf("the %s state in %s does not say the allowance is each side's own:\n%s",
					name, lang, drawn)
			}
			// ⚠️ **The reading really does carry a countdown**, so a screen that
			// drew one would compile and look right. What is held is that this
			// drawing does not read it at all: the same state with a clock ticking
			// on it draws the same bytes, in the body and in the footer.
			counting := state
			counting.Live.Clock = aCountdown(PlayClockYou)
			counted, countedFooter := counting.View(c)
			if body, footer := drawn, state.footer(c); counted != body || countedFooter != footer {
				t.Errorf("the %s state in %s drew differently with a countdown on the reading, "+
					"which reads as one allowance for a phase that has one per side:\n%s",
					name, lang, counted)
			}
		}
	}
}

// TestAnArrangementAlreadySentSaysWhyItIsWaiting is the state the phase's own
// design makes necessary: it is simultaneous and secret, so a screen that only
// said it was waiting would read as a match that had stalled.
//
// ⚠️ **And the keys stay live, which is the half that looks like a bug.** Nothing
// on any reading says an arrangement landed — the record takes both in one step —
// so what a client knows is that it pressed send, not that the room took it. A
// refused arrangement is one to send **again**, and the retry is the only thing
// that ever gets a turned-away decision through, so the memo here is display and
// nothing else. → ArrangeScreen.Sent.
func TestAnArrangementAlreadySentSaysWhyItIsWaiting(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := asSent(t, c, anArrangement(t, c, lib, true))
	drawn, footer := screen.View(c)
	if !strings.Contains(drawn, firstLineOf(c.Text(i18n.ArrangeSent))) {
		t.Errorf("a sent arrangement does not say what it is waiting for:\n%s", drawn)
	}
	if footer != c.Text(i18n.ArrangeSendFooter) {
		t.Errorf("the footer is %q, want the one that still names the keys a refused "+
			"arrangement is re-sent with", footer)
	}
	again, result := screen.Update(c, press(t, "enter"))
	if !result.Decided || !slices.Equal(result.Decision.Slots, screen.Slots) {
		t.Errorf("a sent arrangement cannot be sent again, so a room that refused one would "+
			"leave this client with no way to answer (%v)", again.Err)
	}
}

// TestAPhaseThatHasClosedAnswersNoKey is the frame between the record taking both
// arrangements and the wire.Start that follows it: the reader is still on this
// screen and there is no decision left anywhere.
//
// A footer naming a key on a decision that has closed is the program promising
// something it does not do, and here the promise is a whole match.
func TestAPhaseThatHasClosedAnswersNoKey(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := asHandedOver(c, wholeArrangement(t, c, anArrangement(t, c, lib, true)))
	drawn, footer := screen.View(c)
	if !strings.Contains(drawn, firstLineOf(c.Text(i18n.ArrangeSent))) {
		t.Errorf("a closed phase does not say what it is waiting for:\n%s", drawn)
	}
	if footer != c.Text(i18n.ArrangeWaitingFooter) {
		t.Errorf("the footer is %q, want the one that names no decision key", footer)
	}
	for _, name := range []string{"enter", "up", "down", "left", "right", "u"} {
		after, result := screen.Update(c, press(t, name))
		if result.Decided {
			t.Errorf("%q sent an arrangement after the phase closed, which the room refuses "+
				"by name", name)
		}
		if after.Cursor != screen.Cursor || len(after.Slots) != len(screen.Slots) {
			t.Errorf("%q moved a screen whose phase has closed", name)
		}
	}
}

// TestNothingOnTheArrangeScreenIsTheOtherSidesFormation is the secrecy of the
// phase held as a property of the reading rather than of the drawing.
//
// The opponent's picks are on the reading — both sides' are, because the draft
// screen draws them — and their **cells** are on no reading at all. So what this
// asserts is that the screen never reaches for the other side's picks: it draws
// this side's, and a reading whose other side is loaded up with picks draws the
// same bytes.
func TestNothingOnTheArrangeScreenIsTheOtherSidesFormation(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := wholeArrangement(t, c, anArrangement(t, c, lib, true))
	bare, _ := screen.View(c)
	live := screen.Live
	theirs := live.theirs()
	live.Picks[theirs] = []DraftPick{
		{Character: live.Pool[9].ID, Stage: "one", Skills: []string{"a"}},
		{Character: live.Pool[10].ID, Stage: "two", Skills: []string{"b"}},
		{Character: live.Pool[11].ID, Stage: "three", Skills: []string{"c"}},
	}
	loaded, _ := screen.Attach(c, live).View(c)
	if bare != loaded {
		t.Errorf("the screen drew differently once the other side had picks on the reading, so "+
			"something here reads them:\n%s\n---\n%s", bare, loaded)
	}
}

// TestTheArrangeFitsTheFloorInEveryStateItDraws is the vertical half of the width
// rule, and this screen is the one that needs it: the board is ten rows of the
// twenty a frame leaves at 120x24, and it may not be what gives way.
func TestTheArrangeFitsTheFloorInEveryStateItDraws(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		c, lib := start(t, lang)
		floor := atTheFloor(c)
		for name, screen := range everyArrangeState(t, floor, lib) {
			body, _ := screen.View(floor)
			if got, room := len(drawnLines(body)), bodyRoom(floor); got > room {
				t.Errorf("the %s state draws %d body lines in %s against the %d a frame "+
					"leaves:\n%s", name, got, lang, room, body)
			}
		}
	}
}

// TestEveryPickKeepsARowAtTheFloor is what that budget is spent on: a formation
// screen that drops one of the units being placed is a screen a player cannot
// finish the decision on.
//
// ⚠️ **Three a side is the format the game ships**, and five is held back
// (TestFiveASideIsHeldBackHereAndNowhereElse); at five the window genuinely
// cannot hold every row and the list windows, which is poolRows' own answer. So
// what is pinned is the shipped board.
func TestEveryPickKeepsARowAtTheFloor(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		c, lib := start(t, lang)
		floor := atTheFloor(c)
		screen := anArrangement(t, floor, lib, true)
		body, _ := screen.View(floor)
		for _, pick := range screen.Picks() {
			if !strings.Contains(body, pick.Character) {
				t.Errorf("%s has no row at the floor in %s:\n%s", pick.Character, lang, body)
			}
		}
	}
}

// anArrangement is the phase open, this reader still owing a formation, over the
// named fixture pool.
//
// ⚠️ **The picks come out of draftFixturePoolIDs and never off the shipped cast**,
// which is the trap step 5b had to fix: a golden built from seed.Cast() moves on
// every content commit, and this repository ships a character every few hours.
func anArrangement(t *testing.T, c Context, lib *forge.Library, yours bool) ArrangeScreen {
	t.Helper()
	pool := draftFixturePool(t, lib)
	live := DraftLive{
		Pool:      pool,
		Seats:     [2]string{fixtureHost, fixtureGuest},
		You:       fixtureHost,
		Step:      DraftStepArrange,
		Yours:     yours,
		Units:     3,
		BanSlots:  2,
		Arranging: true,
		Recorded:  10,
	}
	live.Bans[0] = []string{pool[0].ID}
	live.Bans[1] = []string{pool[1].ID}
	live.Picks[0] = []DraftPick{
		{Character: pool[2].ID, Stage: "one", Skills: []string{"a"}, Passives: []string{"b"}},
		{Character: pool[3].ID, Stage: "two", Skills: []string{"c"}, Passives: []string{"d"}},
		{Character: pool[4].ID, Stage: "three", Skills: []string{"e"}, Passives: []string{"f"}},
	}
	screen := NewArrangeScreen().Attach(c, live)
	if len(screen.Picks()) != 3 {
		t.Fatalf("the fixture holds %d picks, want the three a 3v3 drafts", len(screen.Picks()))
	}
	return screen
}

// wholeArrangement is every pick placed, through the keys a player presses.
func wholeArrangement(t *testing.T, c Context, screen ArrangeScreen) ArrangeScreen {
	t.Helper()
	for range len(screen.Picks()) {
		screen, _ = screen.Update(c, press(t, "enter"))
	}
	if !screen.Whole() {
		t.Fatalf("%d of %d picks were placed", len(screen.Slots), len(screen.Picks()))
	}
	if screen.Err != nil {
		t.Fatalf("the arrangement this fixture built is refused: %v", screen.Err)
	}
	return screen
}

// asSent is the arrangement handed over, through the key that hands it over.
//
// ⚠️ **It cannot be built out of the reading**, which is the finding this state
// exists for: the record takes both arrangements in one step, so a client that has
// arranged reads exactly what it read before. → ArrangeScreen.Sent.
func asSent(t *testing.T, c Context, screen ArrangeScreen) ArrangeScreen {
	t.Helper()
	sent, result := wholeArrangement(t, c, screen).Update(c, press(t, "enter"))
	if !result.Decided {
		t.Fatalf("the arrangement was not sent: %v", sent.Err)
	}
	if !sent.Sent || !sent.Waiting() {
		t.Fatal("the screen does not know it sent")
	}
	return sent
}

// asHandedOver is the one frame between the record closing the phase and the
// wire.Start that follows it: the reading says the phase is no longer open and no
// board has arrived, so the reader is on this screen with nothing left to do.
func asHandedOver(c Context, screen ArrangeScreen) ArrangeScreen {
	live := screen.Live
	live.Arranging, live.Yours, live.Step = false, false, DraftStepNone
	return screen.Attach(c, live)
}

// walkedTo puts the cursor on a cell with the arrow keys, which is the only way a
// player has of getting it there.
func walkedTo(t *testing.T, c Context, screen ArrangeScreen, cell hex.Offset) ArrangeScreen {
	t.Helper()
	for range hex.FormationCols {
		if screen.Cursor.Col == cell.Col {
			break
		}
		screen, _ = screen.Update(c, press(t, "right"))
	}
	for range hex.FormationRows {
		if screen.Cursor.Row == cell.Row {
			break
		}
		screen, _ = screen.Update(c, press(t, "down"))
	}
	if screen.Cursor != cell {
		t.Fatalf("the cursor could not be walked to %s and stopped at %s", cell, screen.Cursor)
	}
	return screen
}

// everyArrangeState is the states of this screen that draw a line no other state
// draws, which is what the golden registers and what the floor is measured over.
//
// ⚠️ **Every one of them asserts it drew the line it exists for**, which is the
// rule this package's golden is written under: a registered state that renders
// nothing passes every sweep over it, and this repository has shipped that fixture
// more than once.
func everyArrangeState(t *testing.T, c Context, lib *forge.Library) map[string]ArrangeScreen {
	t.Helper()
	opening := anArrangement(t, c, lib, true)
	// One placed, which is the only state that draws a board with both a number
	// and a cursor on it and a list with both kinds of row.
	part, _ := opening.Update(c, press(t, "enter"))
	whole := wholeArrangement(t, c, opening)
	refused := whole
	refused.Slots = []hex.Offset{{Col: 1, Row: 1}, {Col: 1, Row: 1}, {Col: 0, Row: 0}}
	refused, _ = refused.Update(c, press(t, "enter"))
	// A refusal the room sent, which on this screen can only be an arrangement it
	// would not take — the one state where the send got through and the room still
	// said no.
	turned := part
	turned.Live.Refusal = fixtureRefusal
	states := map[string]ArrangeScreen{
		"an arrangement":                      opening,
		"an arrangement part way":             part,
		"a whole arrangement":                 whole,
		"a refused arrangement":               refused,
		"an arrangement the room turned down": turned,
		"an arrangement sent":                 asSent(t, c, opening),
		"an arrangement handed over":          asHandedOver(c, whole),
	}
	assert := func(name string, want string) {
		t.Helper()
		drawn, _ := states[name].View(c)
		if !strings.Contains(drawn, want) {
			t.Fatalf("the %q state draws no line saying so:\n%s", name, drawn)
		}
	}
	assert("an arrangement", c.Text(i18n.ArrangeInHand, opening.Picks()[0].Character))
	assert("an arrangement part way", c.Text(i18n.ArrangePlaced, 1, 3))
	assert("a whole arrangement", c.Text(i18n.ArrangeSendReady))
	assert("a refused arrangement", firstLineOf(c.Lang.Error(refused.Err)))
	assert("an arrangement the room turned down", c.Text(i18n.DraftRefused))
	assert("an arrangement sent", firstLineOf(c.Text(i18n.ArrangeSent)))
	// The two waiting states are told apart by their footers, which the golden
	// records separately: one still answers every key, because a refused
	// arrangement is one to send again, and the other answers none.
	if _, footer := states["an arrangement sent"].View(c); footer != c.Text(i18n.ArrangeSendFooter) {
		t.Fatalf("a sent arrangement stopped naming the keys that would send it again: %q", footer)
	}
	if _, footer := states["an arrangement handed over"].View(c); footer != c.Text(i18n.ArrangeWaitingFooter) {
		t.Fatalf("a phase that has closed still names a decision key: %q", footer)
	}
	return states
}
