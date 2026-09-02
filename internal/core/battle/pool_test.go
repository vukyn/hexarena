package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
)

// barriered is one enemy of a pair put behind a pool nothing on the board can
// empty, or neither when the name is empty.
func barriered(t *testing.T, on string) battle.Choice {
	t.Helper()
	fight, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"strike"}},
		// The walled one is the SOFTER target by a wide margin, so a rating blind
		// to the pool prefers it — and cannot land a point of it. Without that the
		// two cells would tie and the answer would be whichever comes first.
		{ID: "walled", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 60, 1),
			Skills: []string{"jab"}},
		{ID: "bare", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4000, 60, 400, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if on != "" {
		unit, known := fight.Unit(on)
		if !known {
			t.Fatalf("no unit %q", on)
		}
		kind, err := fight.Books().Statuses.Lookup("aegis")
		if err != nil {
			t.Fatalf("lookup aegis: %v", err)
		}
		unit.Statuses.Apply(kind, 100000)
	}
	return chosen(t, fight)
}

// TestARatingDoesNotSwingIntoABarrierItCannotEmpty is the half of a guard this
// rating could not see.
//
// `shielded` and `guarded` pay to PUT a guard up and nothing discounted a blow
// INTO one, so the rating bought walls and treated the enemy's as absent.
// Measured: offered two identical enemies, one of them softer and carrying a pool
// of a hundred thousand, it aimed at the softer one every time.
//
// ⚠️ **Three arrangements, and the third is the control.** With no pool anywhere
// the softer target really is the right answer, so a rating that had simply
// started avoiding the first cell would fail that row — which is what the first
// draft of this test could not tell apart, because it read the same answer in
// all three and that answer was the tie order.
func TestARatingDoesNotSwingIntoABarrierItCannotEmpty(t *testing.T) {
	for _, row := range []struct {
		name string
		on   string
		want string
	}{
		{"the soft one behind a pool", "walled", "bare"},
		{"the hard one behind a pool", "bare", "walled"},
		{"nobody behind a pool", "", "walled"},
	} {
		choice := barriered(t, row.on)
		aimed := "bare"
		if choice.Aim == (hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})) {
			aimed = "walled"
		}
		if aimed != row.want {
			t.Errorf("with %s, Suggest aimed at %s, want %s", row.name, aimed, row.want)
		}
	}
}

// TestAnUnblockableBlowIsNotDiscountedByAPool is the keyword read as a decision
// rather than as a resolution rule.
//
// `resolveAgainst` skips both guards for an unblockable skill, so the rating has
// to as well or the keyword is free: a skill that goes through a barrier was
// rated as though the barrier stopped it, which is the one board it exists for.
func TestAnUnblockableBlowIsNotDiscountedByAPool(t *testing.T) {
	fight, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"strike", "unstoppable"}},
		{ID: "you", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	// Without the pool `strike` is ten times the power and is the answer, which is
	// what makes the row below a measurement rather than a preference.
	if choice := chosen(t, fight); choice.Skill != "strike" {
		t.Fatalf("with no pool up, Suggest picked %q, want strike: the fixture proves nothing", choice.Skill)
	}

	guarded, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"strike", "unstoppable"}},
		{ID: "you", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	guarded.Begin()
	guarded.Drain()
	them, _ := guarded.Unit("you")
	kind, err := guarded.Books().Statuses.Lookup("aegis")
	if err != nil {
		t.Fatalf("lookup aegis: %v", err)
	}
	them.Statuses.Apply(kind, 100000)
	if got := them.Statuses.PoolIn(status.Absorb); got <= 0 {
		t.Fatalf("the pool came out at %d, so nothing was in the way", got)
	}
	if choice := chosen(t, guarded); choice.Skill != "unstoppable" {
		t.Errorf("behind a pool of a hundred thousand, Suggest picked %q, want unstoppable: "+
			"a blow the barrier cannot stop is the only one worth throwing", choice.Skill)
	}
}
