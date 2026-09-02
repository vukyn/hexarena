package combat_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// Carrying a saturated figure, which is the half of the overflow work that the
// widening left undone.
//
// ⚠️ **A saturation that is only produced is not a saturation.** `wide.over`
// answers math.MaxInt64 to a quotient that will not fit, and the argument for
// that is written where it lives: damage past any reachable health is not a wrong
// answer, and it keeps damage non-decreasing in power, which is the property a
// wrap actually breaks. All true, and all of it stops at that one return. The
// value then travels — into a splash share, into a strike count, into a weighted
// average, into a wall of charges — and every one of those was a plain narrow
// product. So the figure that saturated at the widest the type holds came back
// out the other side as a *small* number, sometimes a negative one, which is
// exactly the defect the widening was written to remove.
//
// These tests are about the journey rather than the origin. Every one of them
// feeds a figure at or near the type's edge into an expression downstream of it
// and asks the same question: is the answer still the largest thing this
// arithmetic can say, or has it wrapped round into a small one.

var carryRules = combat.Rules{
	DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250,
	MinHitChance: 150, MaxBlockCharges: 3,
}

// exactScaled is `value * ratio / base` with no width limit at all, saturated at
// the widest figure an int64 holds. It is the oracle the two narrow expressions
// are measured against, and it is big.Int rather than a second copy of the code
// under test for the reason overflow_test.go gives at length: an expected figure
// derived from the formula is worth something, one recorded from the code it
// checks is worth nothing.
func exactScaled(value int64, ratio, base int64) int64 {
	if value <= 0 || ratio <= 0 {
		return 0
	}
	product := big.NewInt(value)
	product.Mul(product, big.NewInt(ratio))
	product.Div(product, big.NewInt(base))
	if !product.IsInt64() {
		return math.MaxInt64
	}
	return product.Int64()
}

// The ladder every sweep below walks. It is geometric rather than dense because
// the question is about magnitudes: the interesting values are the ones either
// side of where a 64-bit product stops fitting, and there is nothing between
// them worth a thousand cases.
func carryLadder() []int64 {
	values := []int64{0, 1, 2, 1_000, 1_000_000}
	for value := int64(1_000_000_000); value < math.MaxInt64/8; value *= 8 {
		values = append(values, value)
	}
	return append(values, math.MaxInt64/2, math.MaxInt64-1, math.MaxInt64)
}

// TestScaledIsTheNarrowExpressionWhereverThatHeld is the claim that makes every
// call site safe to change: this is not new arithmetic, it is the same
// arithmetic with the range it always should have had.
//
// The two halves matter equally. Agreeing with the narrow expression everywhere
// that expression was right is what lets the shipped figures stay bit for bit
// what they were — no golden moves, no battle replays differently. Answering the
// saturated figure everywhere it was wrong is the fix.
func TestScaledIsTheNarrowExpressionWhereverThatHeld(t *testing.T) {
	ratios := []int{0, 1, 500, combat.PermilleBase, 1500, 1_000_000}
	for _, value := range carryLadder() {
		for _, ratio := range ratios {
			want := exactScaled(value, int64(ratio), int64(combat.PermilleBase))
			if got := combat.Scaled(value, ratio); got != want {
				t.Errorf("Scaled(%d, %d) is %d, want %d", value, ratio, got, want)
			}
		}
		// The identity Swung's own comment turns on: a ratio of the base returns
		// the value untouched, however large it is. A guard that saturated on the
		// product alone would refuse a figure it was handed and could have
		// returned three orders of magnitude below where the quotient stops
		// fitting.
		if got := combat.Scaled(value, combat.PermilleBase); got != max(value, 0) {
			t.Errorf("Scaled(%d, base) is %d, want the value back", value, got)
		}
	}
}

// TestRepeatedIsTheNarrowProductWhereverThatHeld is the same claim for the two
// places a per-strike figure is multiplied by a count rather than by a ratio:
// Rules.Total, and the wall of block charges the rating subtracts.
func TestRepeatedIsTheNarrowProductWhereverThatHeld(t *testing.T) {
	for _, value := range carryLadder() {
		for _, count := range []int{0, 1, 2, 3, 5, 1_000} {
			want := exactScaled(value, int64(count), 1)
			if got := combat.Repeated(value, count); got != want {
				t.Errorf("Repeated(%d, %d) is %d, want %d", value, count, got, want)
			}
		}
	}
}

// TestAWeightedStrikeAveragesRatherThanWrapping is the one expression in the
// package whose *answer* always fitted and whose working did not.
//
// A weighted average of two values lies between them, so ExpectedStrike can
// never legitimately need more than an int64 — and it wrapped anyway, because
// `ordinary*(1000-chance) + critical*chance` builds a figure a thousand times
// the answer before dividing it back down. That is the worst way to be wrong:
// not a limit of the type, but arithmetic thrown away on the way to a number
// that was always representable.
func TestAWeightedStrikeAveragesRatherThanWrapping(t *testing.T) {
	// A hit whose ordinary strike already saturates: the power is the widest an
	// int can carry, so the quotient leaves the type and `over` pins it.
	hit := combat.Hit{
		Scaling: 2399, Defense: 0, Multiplier: math.MaxInt64,
		Affinity: combat.PermilleBase, Strikes: 1,
	}
	if ordinary := carryRules.Strike(hit); ordinary != math.MaxInt64 {
		t.Fatalf("the fixture's ordinary strike is %d, want it saturated: this "+
			"test measures nothing unless the strike it weights is at the edge",
			ordinary)
	}

	for _, chance := range []int{1, 100, 500, 999, combat.PermilleBase} {
		weighted := hit
		weighted.Crit = chance
		// Both ends of the average are the saturated figure — the critical
		// multiplier cannot make math.MaxInt64 larger — so whatever weight is put
		// on them, the average is that figure and nothing else.
		if got := carryRules.ExpectedStrike(weighted); got != math.MaxInt64 {
			t.Errorf("at a crit chance of %d the weighted strike is %d, want %d: "+
				"an average of two equal figures is that figure",
				chance, got, int64(math.MaxInt64))
		}
	}
}

// TestNoFigureFallsAsPowerRises is the property the whole file exists for, and
// it is deliberately stated as an inequality rather than as a table.
//
// A table of expected numbers says what the arithmetic does today. This says
// what it must never do: come back smaller for more power. That is the one thing
// a wrap always does and a saturation never does, so it catches a narrow product
// anywhere on the path from a power to a landed figure — including one nobody has
// added yet, which a table of today's numbers could not.
func TestNoFigureFallsAsPowerRises(t *testing.T) {
	for _, measure := range []struct {
		name string
		of   func(multiplier int64) int64
	}{
		{"a single strike", func(multiplier int64) int64 {
			return carryRules.Strike(carryHit(multiplier))
		}},
		{"every strike combined", func(multiplier int64) int64 {
			return carryRules.Total(carryHit(multiplier))
		}},
		{"a strike before it is rolled", func(multiplier int64) int64 {
			hit := carryHit(multiplier)
			hit.Crit = 300
			return carryRules.ExpectedStrike(hit)
		}},
		{"a hit before it is rolled", func(multiplier int64) int64 {
			hit := carryHit(multiplier)
			hit.Crit = 300
			return carryRules.Expected(hit)
		}},
		{"a converted strike", func(multiplier int64) int64 {
			hit := carryHit(multiplier)
			hit.Convert = 400
			return carryRules.Strike(hit)
		}},
		{"a splashed power", func(multiplier int64) int64 {
			return combat.Scaled(multiplier, 500)
		}},
		{"a restored share of a stat", func(multiplier int64) int64 {
			if multiplier > math.MaxInt {
				multiplier = math.MaxInt
			}
			return carryRules.Restore(2399, int(multiplier))
		}},
		{"a part-pierced defence", func(defense int64) int64 {
			return combat.Pierced(defense, 500)
		}},
		{"a power the caster's own terms raised", func(multiplier int64) int64 {
			if multiplier > math.MaxInt64/2 {
				multiplier = math.MaxInt64 / 2
			}
			return int64(combat.Swung(int(multiplier), 1_200, 900))
		}},
	} {
		previous := int64(-1)
		for _, multiplier := range carryLadder() {
			got := measure.of(multiplier)
			if got < 0 {
				t.Errorf("%s at a multiplier of %d is %d, which is negative: a "+
					"blow cannot heal", measure.name, multiplier, got)
			}
			if got < previous {
				t.Errorf("%s fell from %d to %d as the multiplier rose to %d: "+
					"more power has come back as less, which is a wrap",
					measure.name, previous, got, multiplier)
			}
			previous = got
		}
	}
}

// carryHit is a three-strike hit against a bare target, which is the shape that
// puts the largest figure the formula can produce in front of the expressions
// downstream of it.
func carryHit(multiplier int64) combat.Hit {
	return combat.Hit{
		Scaling: 2399, Defense: 0, Multiplier: int(multiplier),
		Affinity: combat.PermilleBase, Strikes: 3,
	}
}

// TestAStrikeCountNobodyWouldShipStillCountsUp is the one bound on this path that
// nothing else enforces.
//
// Skill.Validate refuses a negative strike count and says nothing about a large
// one, so a count is the single figure reaching this arithmetic that no ceiling
// stands in front of. A book with a count that big is a book nobody would ship —
// and the answer to a figure nobody would ship is still not a negative one.
func TestAStrikeCountNobodyWouldShipStillCountsUp(t *testing.T) {
	for _, count := range []int{
		1, 3, 1_000,
		math.MaxInt / combat.PermilleBase,
		math.MaxInt/combat.PermilleBase + 1,
		math.MaxInt,
	} {
		hit := combat.Hit{
			Scaling: 2399, Multiplier: combat.PermilleBase,
			Affinity: combat.PermilleBase, Strikes: count,
		}
		if got := hit.ExpectedStrikes(); got <= 0 {
			t.Errorf("a count of %d expects %d strikes, want a positive figure", count, got)
		}
		if got := carryRules.Total(hit); got <= 0 {
			t.Errorf("a count of %d totals %d, want a positive figure", count, got)
		}
		if got := carryRules.Expected(hit); got <= 0 {
			t.Errorf("a count of %d is expected to come to %d, want a positive figure",
				count, got)
		}
	}
}

// TestATalliedBlowSaturatesRatherThanWrapping is the last narrow product on the
// path, and the only one that is a SUM rather than a scaling.
//
// A multi-strike skill's attempts are tallied one by one, and two saturated
// strikes added in an int64 come to minus two. The two functions here are what a
// caller reads to find out what a blow did, so a wrap in them does not merely
// misprice a choice — it reports a blow that healed.
func TestATalliedBlowSaturatesRatherThanWrapping(t *testing.T) {
	attempts := []combat.Attempt{
		{Outcome: combat.Struck, Damage: math.MaxInt64, Absorbed: math.MaxInt64},
		{Outcome: combat.Struck, Damage: math.MaxInt64, Absorbed: math.MaxInt64},
		{Outcome: combat.Missed},
		{Outcome: combat.Struck, Damage: 7},
	}
	if got := combat.DamageDealt(attempts); got != math.MaxInt64 {
		t.Errorf("three saturated strikes dealt %d, want %d",
			got, int64(math.MaxInt64))
	}
	if got := combat.AbsorbedBy(attempts); got != math.MaxInt64 {
		t.Errorf("two saturated strikes were absorbed for %d, want %d",
			got, int64(math.MaxInt64))
	}
	// And the ordinary case is untouched: a tally of figures that fit is their
	// sum, which is what every battle ever fought reads out of this.
	ordinary := []combat.Attempt{
		{Outcome: combat.Struck, Damage: 120, Absorbed: 30},
		{Outcome: combat.Blocked},
		{Outcome: combat.Struck, Damage: 118, Absorbed: 0},
	}
	if got := combat.DamageDealt(ordinary); got != 238 {
		t.Errorf("an ordinary tally is %d, want 238", got)
	}
	if got := combat.AbsorbedBy(ordinary); got != 30 {
		t.Errorf("an ordinary absorbed tally is %d, want 30", got)
	}
}
