package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// unreachable is two units that cannot touch each other: the far corners of the
// two halves, which `hex.Place` puts four cells apart — past every range in the
// fixture's book.
//
// The pair is what the roster's own reach rule exists to keep off the board, and
// that rule is exactly why this has to be built by hand: nothing shipped can get
// here, and the engine still has to answer for it.
func unreachable(t *testing.T, allySkills, foeSkills []string) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 2},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 120), Skills: allySkills},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 1, Row: 2},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 100), Skills: foeSkills},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	return fight
}

// TestABoardNobodyCanReachIsADrawEvenWhileSomethingIsStillTicking is the hole in
// the deadlock predicate, closed.
//
// `frozen` used to refuse to call a draw while *anything* timed was on the board,
// on the grounds that a duration left to spend is a promise the board is not
// final. For a poison that is true — it kills, and a kill empties a side. For a
// regeneration it is not: it changes a number for ever and changes nothing else.
//
// So a unit refreshing its own regeneration every turn held a frozen board open
// permanently. The draw was never declared and the battle ran the turn limit out
// instead, which is worse than a wrong answer: a log of it says nothing at all
// about what happened. A draft roster using slot `1,2` hit exactly this in 5 seeds
// of 4000.
func TestABoardNobodyCanReachIsADrawEvenWhileSomethingIsStillTicking(t *testing.T) {
	// Both sides hold a self-aimed regeneration and a weapon that cannot reach.
	// The regeneration is the only thing either will ever manage to do.
	fight := unreachable(t, []string{"strike", "bloom"}, []string{"strike", "bloom"})
	turns, err := fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if turns >= 4000 {
		t.Fatalf("the battle ran %d turns, so it hit the limit rather than being "+
			"declared a draw", turns)
	}
	if _, decided := fight.Winner(); decided {
		t.Error("a board nobody can reach produced a winner")
	}
	ended := false
	for _, event := range fight.Drain() {
		if event.Kind != battle.Ended {
			continue
		}
		ended = true
		if event.Outcome != battle.Stalemate {
			t.Errorf("the battle ended as %v, want a stalemate", event.Outcome)
		}
	}
	if !ended {
		t.Error("the battle stopped without an ended event, so nothing in the log " +
			"says why it stopped")
	}
	t.Logf("declared a draw after %d turns", turns)
}

// TestAPoisonedDeadlockIsStillNotADeadlock is the other half of the same
// predicate, and it is the reason the fix names categories rather than dropping
// the clause: a poison on a board nobody can reach *will* end it, by killing
// somebody and emptying a side.
func TestAPoisonedDeadlockIsStillNotADeadlock(t *testing.T) {
	fight := unreachable(t, []string{"strike", "brace"}, []string{"strike", "brace"})
	// Low enough that the ticks finish it: the claim is that a poison ends the
	// battle, not that any amount of poison ends any unit.
	atHealth(t, fight, "f", 150)
	carrying(t, fight, "f", "poison", 3)

	turns, err := fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	side, decided := fight.Winner()
	if !decided {
		t.Fatalf("a poisoned unit on an unreachable board produced no winner after "+
			"%d turns", turns)
	}
	if side != hex.SideAlly {
		t.Errorf("the poisoned unit's side won: %v", side)
	}
}

// TestATauntedUnitWithNoReachIsNotADeadlockEither is the case that looks like a
// deadlock and must not be called one: an enemy in reach, and a taunting enemy out
// of it, so every aim is narrowed onto the unit nobody can touch. There is a unit
// waiting to be killed the moment the taunt runs out.
//
// ⚠️ It passes for a reason that is *not* the timed-status clause, and finding that
// out is why this test exists. A taunt is not in outcomeChanging, and it does not
// need to be: `aims` offers a taunted unit its taunter whether or not the taunter is
// in reach, so the unit always has an aim and the board is never frozen. Adding the
// category as well looked right and was a claim no test could reach — a mutation
// removing it changed nothing at all.
//
// So what this test really pins is the *aims* behaviour, from the outside: the day
// a taunted unit that cannot reach its taunter stops being offered it — which the
// taunt work left open on purpose — this board becomes a wrongly declared draw, and
// this is where that shows up.
func TestATauntedUnitWithNoReachIsNotADeadlockEither(t *testing.T) {
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"strike"}},
		// The reachable enemy holds nothing it can point at anybody, so it cannot
		// keep the board open on its own account — the only thing standing between
		// this board and a wrongly declared draw is the taunt.
		{ID: "near", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(600, 100, 100, 100),
			Skills: []string{"brace"}},
		{ID: "far", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 100, 100, 20),
			Skills: []string{"brace"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	carrying(t, fight, "far", "taunting", 1)

	turns, err := fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, decided := fight.Winner(); !decided {
		t.Errorf("the battle was called a draw after %d turns while a taunt was still "+
			"deciding who may be aimed at, and an enemy in reach was waiting to be "+
			"killed the moment it expired", turns)
	}
}
