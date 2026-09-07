package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// `docs/balance.md` carries an instruction that had no method under it: **build a
// wall board that FINISHES before quoting a figure about a guard**. The boards
// used to judge the block-charge discount were blind because they never resolved,
// so the gain read as nothing, and nobody wrote down what a board that resolves
// looks like.
//
// This is that condition, measured and held: **only one side may carry the
// guard.**
//
// ⚠️ A symmetric guard board cannot resolve, and it is not the turn limit being
// short. Measured over 200 seeds at a limit of 600 turns, a mirror of Blastoise
// stat lines: `withdraw` a side is **100% endless**, `carapace` a side is **100%
// endless**, and the same board with neither resolves every seed in 66 turns.
// The two have different causes and only one of them is a defect —
// → `TODO.md` § *A guarded mirror never resolves*.
//
// An asymmetric board is also the right shape for the question. What a guard is
// worth is what it buys the side carrying it, which is a comparison against a
// side that is not — so the board that resolves is the board that measures the
// thing, and the mirror was never going to answer it.

// guardBoardSeeds is how many seeds each row below is fought over. A hundred is
// enough for a claim about whether battles END; the rates a guard measurement
// wants are a different run with a different count.
const guardBoardSeeds = 100

// guardBoardLimit is well past the ~104 turns the asymmetric board takes, so a
// row that reads endless is endless rather than clipped.
const guardBoardLimit = 600

// TestAGuardBoardResolvesOnlyWhenOneSideCarriesIt is the condition, both halves.
//
// The negative half is the load-bearing one: without it this test would pass on
// an engine where every board resolves, and the instruction it exists to answer
// would look satisfied by a fixture that was never the problem.
func TestAGuardBoardResolvesOnlyWhenOneSideCarriesIt(t *testing.T) {
	guarded := []string{"withdraw", "water_gun"}
	plain := []string{"water_gun", "bite"}

	ended, endless := theGuardBoard(t, guarded, nil, plain, nil)
	if endless > 0 {
		t.Errorf("a guard on one side left %d of %d battles unfinished, so the board "+
			"`docs/balance.md` asks for does not resolve after all", endless, ended+endless)
	}

	// The mirror, which is what everybody built first and what nothing said would
	// not work. Asserted rather than described, so the day it starts resolving
	// this test says so instead of quietly guarding nothing.
	mirrored, stalled := theGuardBoard(t, guarded, nil, guarded, nil)
	if stalled != mirrored+stalled {
		t.Errorf("a guard on BOTH sides finished %d of %d battles: the mirror resolves now, "+
			"so the one-sided rule above is no longer the reason and this test is holding "+
			"a condition that has moved", mirrored, mirrored+stalled)
	}

	// And the same mirror with no guard at all, which is what makes the row above
	// a statement about the guard rather than about the stat line.
	clean, never := theGuardBoard(t, plain, nil, plain, nil)
	if never > 0 {
		t.Errorf("a mirror carrying no guard left %d of %d battles unfinished, so the "+
			"stall above is not the guard's doing and this whole file is measuring the "+
			"wrong thing", never, clean+never)
	}
}

// theGuardBoard fights one arrangement over the seed range and reports how many
// battles ended and how many ran out of turns.
//
// One arrangement rather than both ways round, deliberately: the question is
// whether a battle ENDS, and an ending is not a side. Fighting both ways would
// double the cost of a claim neither half can disagree with.
func theGuardBoard(t *testing.T, mine, myTraits, theirs, theirTraits []string) (int, int) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.squirtle")
	ended, endless := 0, 0
	for which := 1; which <= guardBoardSeeds; which++ {
		fight, err := battle.New(books, uint64(which), []battle.Roster{
			{ID: "guarded", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: affinity, Stats: stats, Skills: mine, Passives: myTraits},
			{ID: "other", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: affinity, Stats: stats, Skills: theirs, Passives: theirTraits},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(guardBoardLimit); err != nil {
			t.Fatalf("run: %v", err)
		}
		if fight.Finished() {
			ended++
		} else {
			endless++
		}
	}
	return ended, endless
}
