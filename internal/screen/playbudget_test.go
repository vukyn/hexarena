package screen

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The battle screen's budget, re-taken
//
// The screen's own comment states the arithmetic that made it budget at all: at
// the declared floor the body's purse is twenty rows, and a five-a-side pairing
// asks for twenty-eight before a single blank or log line. That figure was taken
// once, by hand, and everything since has been asked to trust it — while the
// effects column, the composition bonuses and a live reconnection line all landed
// on the same screen.
//
// So it is taken again here, by the screen, in figures the screen itself
// produces. What this file is for is not the arithmetic — playFit already has
// tests for that — but the **premise**: that there is no row to spare, which is
// the sentence three separate decisions have been justified with (the countdown
// on the heading row, the board dropped whole, the log last).

// theFloorHeight is the declared floor this repository budgets against.
//
// ⚠️ Written here rather than read off a constant because there is none to read:
// the floor is 120x24 and lives in the wording tests and in CLAUDE.md as a pair
// of numbers. A test that computed it from PlayBodyRoom would be deriving the
// question from the answer.
const theFloorHeight = 24

// TestABattleAtTheFloorAsksForMoreRowsThanItHas is the premise, measured.
//
// ⚠️ **It asserts the SHORTFALL and not the sections**, and the difference is
// what keeps it from being a golden of the layout. Which section costs what moves
// whenever anybody touches a drawing — that is what playFit is for — and the
// claim being pinned is the one every budgeting decision rests on: at the floor,
// a full pairing cannot fit, so no row is free for anything new.
func TestABattleAtTheFloorAsksForMoreRowsThanItHas(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	c = inAWindow(c, theFloorHeight)
	purse := PlayBodyRoom(c.Height)

	for _, side := range []int{1, hex.MaxSquadSize} {
		p := atABattleOf(t, c, side)
		sizes := p.drawings(c).sizes()
		// The heading is a row the body always writes, and every section below it
		// is what the sizes report. Counted the way the screen's own comment
		// counts it, so a reader can put the two side by side.
		asked := 1 + blockRows(sizes.tail) + sizes.board + sizes.units + 1 + 1 + PlayLogWanted
		t.Logf("%d a side at %dx%d: purse %d, asks %d (heading 1, tail %d, board %d, "+
			"roster %d+1, order 1, log wants %d)",
			side, c.Width, c.Height, purse, asked, blockRows(sizes.tail),
			sizes.board, sizes.units, PlayLogWanted)
		if side < hex.MaxSquadSize {
			continue
		}
		if asked <= purse {
			t.Errorf("a %d-a-side battle asks for %d rows and the floor gives it %d: the "+
				"budget's whole premise is that it cannot fit, and three decisions rest on "+
				"it — the countdown on the heading row, the board dropped whole, the log "+
				"last. If this is now true, those are worth re-opening", side, asked, purse)
		}
	}
}

// TestTheBoardAndTheRosterAloneOutgrowTheFloor is the sharper half, and it is the
// one a summon reaches.
//
// ⚠️ **A squad is five a side and a BOARD is nine**, because a summon puts units
// down past the ones the squad brought — up to the formation slots a side. So the
// figure that bounds this screen is not the pairing a player builds, it is the
// board a battle can reach, and two sections alone pass the purse before the
// heading, the order line, the log or the option list have asked for anything.
func TestTheBoardAndTheRosterAloneOutgrowTheFloor(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	c = inAWindow(c, theFloorHeight)
	purse := PlayBodyRoom(c.Height)

	p := atABattleOf(t, c, hex.MaxSquadSize)
	sizes := p.drawings(c).sizes()
	// The roster's header goes with the first unit, so a board of N units is N+1
	// rows. The units on the board today are the squads; the slots are what a
	// summon can fill.
	slots := hex.FormationCols * hex.Rows
	fullest := sizes.board + 1 + slots
	t.Logf("board %d + a roster of every one of the %d formation slots a side = %d, "+
		"against a purse of %d", sizes.board, slots, fullest, purse)
	// ⚠️ **They come to exactly the purse**, which is worse than passing it
	// rather than better: two sections consume every row there is, so the
	// heading, the order line, the log and the option list are all in deficit
	// before anything is drawn. The assertion is therefore "at least", not "more
	// than" — a board that fitted with rows to spare is what would re-open the
	// priority list, and that is not this.
	if fullest < purse {
		t.Errorf("the board and a full roster come to %d rows and the floor gives the body "+
			"%d, so there are %d rows spare: the screen budgets because those two alone "+
			"leave nothing, and if they now leave something, the priority list is worth "+
			"re-reading", fullest, purse, purse-fullest)
	}
}

// TestALiveBattleCostsTheWaitingRowItWasPredictedTo is the PvP half of the item
// this file answers.
//
// ⚠️ **The tail is the section the whole budget is arranged around**, and live
// mode is what makes that matter: where a local battle between turns draws
// nothing there and resolves in microseconds, this draws a row and can hold it
// for a whole allowance. So the record's "PvP adds a waiting row on top" is
// correct — and it is a row taken from a purse that was already in deficit,
// which is the answer to the question the item actually asked.
func TestALiveBattleCostsTheWaitingRowItWasPredictedTo(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	c = inAWindow(c, theFloorHeight)

	local := atABattleOf(t, c, hex.MaxSquadSize)
	waiting := NewPlayScreen().Attach(c, PlayLive{
		Fight: local.Fight, Side: local.Side, Seed: local.Seed,
	})
	if waiting.Pending != nil {
		t.Fatal("the waiting live screen still holds a turn, so this measures an ordinary one")
	}
	sizes := waiting.drawings(c).sizes()
	if sizes.board != 10 || sizes.units == 0 {
		t.Fatalf("the live screen drew board %d and roster %d, so it is not drawing a "+
			"battle at all", sizes.board, sizes.units)
	}
	asked := 1 + blockRows(sizes.tail) + sizes.board + sizes.units + 1 + 1 + PlayLogWanted
	t.Logf("live and waiting, %d a side: purse %d, asks %d (tail %d)",
		hex.MaxSquadSize, purseAtTheFloor(c), asked, blockRows(sizes.tail))
	if blockRows(sizes.tail) == 0 {
		t.Error("a live battle waiting on the other player spends no row saying so, and the " +
			"record says it spends one")
	}
	if asked <= purseAtTheFloor(c) {
		t.Errorf("a live 5-a-side asks %d rows against a purse of %d", asked, purseAtTheFloor(c))
	}
}

// purseAtTheFloor is the body's own purse, read through the screen's arithmetic
// rather than written down a second time. → PlayBodyRoom.
func purseAtTheFloor(c Context) int { return PlayBodyRoom(c.Height) }
