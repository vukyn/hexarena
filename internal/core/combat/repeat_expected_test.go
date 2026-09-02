package combat_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// repeating is a hit that lands once and keeps going on a coin, up to eight.
func repeating(repeat, max int) combat.Hit {
	return combat.Hit{
		Scaling: 1000, Multiplier: 1000, Strikes: 1, Repeat: repeat, MaxStrikes: max,
		Affinity: combat.PermilleBase, Defense: 0,
		SkillAccuracy: combat.PermilleBase, AccuracyStat: 100, DodgeStat: 0,
	}
}

// TestExpectedReadsTheDistributionAndTotalReadsTheFloor is the pair of readings
// this package deliberately keeps apart, asserted against each other so neither
// can quietly become the other.
//
// `Repeat` is a **distribution** rather than a count — mostly the floor,
// occasionally a great deal more — and the field's own comment says a rating
// cannot use the ceiling (it would price every cast as the best cast) and cannot
// use the floor either. `Expected` is the reading for everything outside the
// roll; `Total` is the deterministic figure skills.golden's damage column is
// written from, and it stays on the floor for exactly that reason.
//
// ⚠️ Expected read the floor until 2026-09-02, which made the two identical for
// every skill in the book — including the one that repeats.
func TestExpectedReadsTheDistributionAndTotalReadsTheFloor(t *testing.T) {
	rules := combat.Rules{
		DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250,
		MinHitChance: 150, MaxBlockCharges: 3,
	}
	if err := rules.Validate(); err != nil {
		t.Fatalf("rules: %v", err)
	}

	steady := repeating(0, 0)
	// A skill that cannot repeat has to answer the same to the bit, or this is a
	// change to every figure in the design record rather than to one.
	if got, want := rules.Expected(steady), rules.Total(steady); got != want {
		t.Errorf("a skill that does not repeat is expected at %d and totals %d", got, want)
	}

	// And a repeating one has to answer MORE, without reaching the ceiling: the
	// tail is real and it is not the best case.
	rolls := repeating(900, 8)
	expected, floor := rules.Expected(rolls), rules.Total(rolls)
	ceiling := rules.Total(combat.Hit{
		Scaling: 1000, Multiplier: 1000, Strikes: 8,
		Affinity: combat.PermilleBase, SkillAccuracy: combat.PermilleBase,
		AccuracyStat: 100,
	})
	t.Logf("repeat 900 to 8: floor %d, expected %d, ceiling %d", floor, expected, ceiling)
	if expected <= floor {
		t.Errorf("a skill repeating on nine tenths is expected at %d against a floor of %d: the tail is worth nothing",
			expected, floor)
	}
	if expected >= ceiling {
		t.Errorf("it is expected at %d against a ceiling of %d: a rating reading the ceiling prices every cast as the best cast",
			expected, ceiling)
	}
	// Total is what must NOT have moved, because the design record is read off it.
	if floor != rules.Strike(rolls) {
		t.Errorf("Total reads %d for a one-strike floor and one strike is %d: the deterministic column moved",
			floor, rules.Strike(rolls))
	}
}
