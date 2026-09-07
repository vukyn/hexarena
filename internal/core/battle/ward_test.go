package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// A trait could refuse a NAMED status — `passive.Resistance` carries a Status and
// an Amount — and nothing could refuse a share of whatever was thrown. A data
// file listing every status in the book would be a rule nobody could read or keep
// current, so "resists effects" as a squad bonus had no way to exist.
//
// ⚠️ **The guard at the top of `resist` was the trap.** It left early when a unit
// held no passive at all, which was right while a trait was the only thing that
// could refuse anything — and would have made a warding status do nothing on
// exactly the units most likely to hold one, because a composition bonus is
// granted to units that may carry no trait.

// TestAWardRefusesAShareOfWhateverIsThrown is the effect, read off the event the
// resolution emits rather than off the arithmetic.
func TestAWardRefusesAShareOfWhateverIsThrown(t *testing.T) {
	plain := refusedAgainst(t, false)
	warded := refusedAgainst(t, true)
	if plain != 0 {
		t.Fatalf("a unit carrying no ward refused %d, so the control is not a control", plain)
	}
	if warded <= 0 {
		t.Errorf("a warded unit refused %d of an application aimed at it: the share is not "+
			"reaching resist", warded)
	}
}

// refusedAgainst throws one status-applying skill at a target that is optionally
// warded, and reports what the event says was refused.
func refusedAgainst(t *testing.T, warded bool) int {
	t.Helper()
	loaded := books(t)
	fight, err := battle.New(loaded, 4, []battle.Roster{
		{ID: "thrower", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60),
			Skills: []string{"maul"}},
		{ID: "target", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 200, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	target, _ := fight.Unit("target")
	if warded {
		kind, err := loaded.Statuses.Lookup("shroud")
		if err != nil {
			t.Fatalf("look up the ward: %v", err)
		}
		target.Statuses.Apply(kind, 0)
	}
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "thrower" {
		t.Fatalf("%s is up and this fixture needs the thrower to be", prompt.Unit)
	}
	if err := fight.Act("maul", target.Cell); err != nil {
		t.Fatalf("maul: %v", err)
	}
	for _, event := range fight.Drain() {
		if event.Target == "target" && event.Refused != 0 {
			return event.Refused
		}
	}
	return 0
}
