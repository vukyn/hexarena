package combat_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/seed"
)

// narrowDamage is the damage expression exactly as it was written before the
// numerator was widened: five factors multiplied into one int64, then divided.
//
// ⚠️ It is here to prove the widening moved nothing, and it is the one thing in
// this file with a shelf life. Delete it, and TestDamageIsBitIdenticalBelowThe
// Overflow with it, once the change is old enough that nobody is going to ask
// whether it moved a figure. Everything else below is about the behaviour rather
// than about the transition and should stay.
//
// It is deliberately a second copy of critical_test.go's closedForm rather than
// a reuse of it: closedForm exists to check the critical-strike identity and
// will outlive this, and giving one function two reasons to exist is how the
// wrong one gets deleted.
func narrowDamage(r combat.Rules, attack, defense int64, multiplier, affinity, crit int) int64 {
	numerator := attack * int64(multiplier) * int64(affinity) * int64(crit) * r.DefenseConstant
	denominator := int64(combat.PermilleBase) * int64(combat.PermilleBase) *
		int64(combat.PermilleBase) * (r.DefenseConstant + defense)
	damage := numerator / denominator
	if damage < r.MinimumDamage {
		return r.MinimumDamage
	}
	return damage
}

// numerator is the damage numerator in arbitrary precision.
//
// ⚠️ **math/big in this file is not a breach of the layer rule.** The rule binds
// what `internal/core`'s packages compile into — what a battle's determinism
// rests on — and this is `package combat_test`, which no engine build links. It
// is also integer arithmetic throughout: big.Int, never big.Float. What the rule
// forbids is a float or a clock reaching a damage figure, and an oracle that
// only ever reads one cannot be that. Using it is the whole point: the expected
// figures below are derived from the formula rather than recorded from the code
// they are checking.
func numerator(r combat.Rules, attack int64, multiplier, affinity, crit int) *big.Int {
	product := big.NewInt(attack)
	for _, factor := range []int64{int64(multiplier), int64(affinity), int64(crit), r.DefenseConstant} {
		product.Mul(product, big.NewInt(factor))
	}
	return product
}

// exactDamage is what the formula comes to with no width limit at all, floored
// at MinimumDamage and saturated at the widest figure an int64 holds.
func exactDamage(r combat.Rules, attack, defense int64, multiplier, affinity, crit int) int64 {
	quotient := numerator(r, attack, multiplier, affinity, crit)
	divisor := big.NewInt(int64(combat.PermilleBase))
	divisor.Mul(divisor, big.NewInt(int64(combat.PermilleBase)))
	divisor.Mul(divisor, big.NewInt(int64(combat.PermilleBase)))
	divisor.Mul(divisor, big.NewInt(r.DefenseConstant+defense))
	quotient.Div(quotient, divisor)
	if !quotient.IsInt64() {
		return math.MaxInt64
	}
	if got := quotient.Int64(); got >= r.MinimumDamage {
		return got
	}
	return r.MinimumDamage
}

// numeratorFitsInt64 reports whether the old expression could have held this
// hit's numerator, which is exactly the range the two arithmetics must agree on.
func numeratorFitsInt64(r combat.Rules, attack int64, multiplier, affinity, crit int) bool {
	return numerator(r, attack, multiplier, affinity, crit).IsInt64()
}

// The reachable extremes of every factor the numerator reads, as the shipped
// data files set them. They are the worst case the wrap has to be measured
// against, and each one is a bound somebody else already enforces:
//
//   - attack: the scaling stat saturates towards three times its progression
//     ceiling (headroom 3000 against a ceiling of 800) and never reaches it, so
//     2399 is the largest a stat can be on the board. Health is refused as a
//     scaling stat, so the 4800 ceiling is not in play.
//   - affinity: the chart's advantage is 1500, and an affinity buff doubles the
//     deviation from neutral at most, because the term is clamped at
//     max_affinity_scale of 1000. 1000 + 500*2 = 2000.
//   - crit: the game-wide critical_multiplier.
const (
	saturatedAttack   = 2399
	strongestAffinity = 2000
)

// TestTheWrappedNumeratorIsGone pins the figures the defect was measured at.
//
// The reference pair is the one skills.golden and the authoring preview both
// use — the attack ceiling against half the defence ceiling, neutral matchup,
// no critical — so the arithmetic is small enough to state in a sentence:
// the numerator is 800 * power * 1000 * 1000 * 300 and the denominator is
// 1000^3 * (300 + 400), which cancels to floor(12 * power / 35). Every want
// below is that fraction worked out by hand, not a reading taken off the code.
//
// The wrapped column is what the old expression answered. Two things make it
// worth carrying: it says which cases in the table actually overflow, so the
// test cannot quietly stop testing the defect, and it names the danger — the
// first and fourth rows are not the visible collapse to MinimumDamage but large,
// plausible, wrong numbers.
func TestTheWrappedNumeratorIsGone(t *testing.T) {
	r := rules()
	const (
		attack  = 800
		defense = 400
	)
	cases := []struct {
		power   int
		want    int64
		wrapped int64
	}{
		{90_000_000, 30_857_142, 4_504_651},
		{120_000_000, 41_142_857, 1},
		{150_000_000, 51_428_571, 1},
		{180_000_000, 61_714_285, 9_009_302},
		{200_000_000, 68_571_428, 1},
	}
	for _, testCase := range cases {
		if want := int64(12) * int64(testCase.power) / 35; want != testCase.want {
			t.Fatalf("power %d: the table says %d but 12*power/35 is %d, so the table is wrong",
				testCase.power, testCase.want, want)
		}
		if got := narrowDamage(r, attack, defense, testCase.power, combat.PermilleBase, combat.PermilleBase); got != testCase.wrapped {
			t.Fatalf("power %d: the old expression gives %d, not the %d this table was written against",
				testCase.power, got, testCase.wrapped)
		}
		if testCase.wrapped == testCase.want {
			t.Fatalf("power %d does not overflow the old expression, so it tests nothing", testCase.power)
		}
		if got := r.Damage(attack, defense, testCase.power, combat.PermilleBase); got != testCase.want {
			t.Errorf("a power of %d deals %d, want %d (it used to deal %d)",
				testCase.power, got, testCase.want, testCase.wrapped)
		}
	}
}

// TestDamageIsBitIdenticalBelowTheOverflow is the property that lets the change
// land without a golden moving: for every hit the old expression could hold, the
// widened one answers the same to the bit.
//
// It sweeps the shipped skill book rather than invented powers, because the
// figures that must not move are the ones the design record is read from, and it
// checks the critical strike beside the ordinary one so the folded-in multiplier
// is covered too.
func TestDamageIsBitIdenticalBelowTheOverflow(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	r := books.Rules
	attacks := []int64{1, 7, 400, 800, saturatedAttack}
	defenses := []int64{0, 1, 300, 400, 800, saturatedAttack}
	affinities := []int{444, 667, combat.PermilleBase, 1500, strongestAffinity, 2250}
	compared := 0
	for _, shipped := range books.Skills.Skills() {
		if shipped.Power <= 0 {
			continue
		}
		for _, attack := range attacks {
			for _, defense := range defenses {
				for _, affinity := range affinities {
					hit := combat.Hit{
						Scaling: attack, Multiplier: shipped.Power,
						Affinity: affinity, Defense: defense, Pierce: shipped.Pierce,
					}
					faced := combat.Pierced(defense, shipped.Pierce)
					for _, crit := range []struct {
						multiplier int
						got        int64
					}{
						{combat.PermilleBase, r.Strike(hit)},
						{r.CriticalMultiplier, r.CriticalStrike(hit)},
					} {
						if !numeratorFitsInt64(r, attack, shipped.Power, affinity, crit.multiplier) {
							t.Fatalf("%s at attack %d, affinity %d, crit %d overflows the old expression: "+
								"the shipped book is supposed to sit well inside it",
								shipped.ID, attack, affinity, crit.multiplier)
						}
						want := narrowDamage(r, attack, faced, shipped.Power, affinity, crit.multiplier)
						if crit.got != want {
							t.Fatalf("%s at attack %d, defence %d, affinity %d, crit %d: %d, want %d",
								shipped.ID, attack, defense, affinity, crit.multiplier, crit.got, want)
						}
						compared++
					}
				}
			}
		}
	}
	if compared < 1000 {
		t.Fatalf("only %d hits compared, which is not a sweep", compared)
	}
}

// TestDamageIsBitIdenticalRightUpToTheOverflow walks the last powers the old
// expression could hold at the worst factors the game can reach, so the two
// arithmetics are compared where they are closest to disagreeing rather than
// only in the comfortable middle.
func TestDamageIsBitIdenticalRightUpToTheOverflow(t *testing.T) {
	r := rules()
	// The largest power whose numerator still fits, at a saturated attack, the
	// strongest affinity the chart and its buff can produce, and a critical.
	// Found by walking down from a power that certainly does not fit, so the
	// figure is derived here rather than copied in.
	last := 0
	for power := 6_000_000; power > 0; power-- {
		if numeratorFitsInt64(r, saturatedAttack, power, strongestAffinity, r.CriticalMultiplier) {
			last = power
			break
		}
	}
	if last == 0 {
		t.Fatal("no power fits the old expression, which cannot be right")
	}
	t.Logf("the old expression held powers up to %d at the worst reachable factors", last)
	for _, power := range []int{1, last / 2, last - 2, last - 1, last} {
		for _, defense := range []int64{0, 400, saturatedAttack} {
			want := narrowDamage(r, saturatedAttack, defense, power, strongestAffinity, r.CriticalMultiplier)
			hit := combat.Hit{
				Scaling: saturatedAttack, Multiplier: power,
				Affinity: strongestAffinity, Defense: defense,
			}
			if got := r.CriticalStrike(hit); got != want {
				t.Errorf("power %d against defence %d: %d, want %d", power, defense, got, want)
			}
			if got := exactDamage(r, saturatedAttack, defense, power, strongestAffinity, r.CriticalMultiplier); got != want {
				t.Errorf("power %d against defence %d: the oracle says %d and the old expression %d, "+
					"so this case is past the overflow and proves nothing", power, defense, got, want)
			}
		}
	}
	// One past the end, where the two must now differ: without this the test
	// above could be walking a range that never reaches the boundary at all.
	beyond := last + 1
	if narrowDamage(r, saturatedAttack, 0, beyond, strongestAffinity, r.CriticalMultiplier) ==
		exactDamage(r, saturatedAttack, 0, beyond, strongestAffinity, r.CriticalMultiplier) {
		t.Errorf("a power of %d does not overflow the old expression, so %d was not the boundary", beyond, last)
	}
}

// TestDamageIsNonDecreasingInPower is the property the defect actually broke,
// and the one a reader cares about: a stronger skill cannot hit for less.
//
// The range straddles the old wrap point, and the second half of the test is
// what keeps it honest — it asserts the old expression *fails* here. Without
// that, a range that never reached the overflow would pass this test forever
// while proving nothing.
func TestDamageIsNonDecreasingInPower(t *testing.T) {
	r := rules()
	const (
		attack  = 800
		defense = 400
		from    = 30_000_000
		to      = 260_000_000
		step    = 250_000
	)
	previous := int64(0)
	for power := from; power <= to; power += step {
		got := r.Damage(attack, defense, power, combat.PermilleBase)
		if got < previous {
			t.Fatalf("a power of %d deals %d, less than the %d the power before it dealt",
				power, got, previous)
		}
		previous = got
	}
	drops := 0
	previous = 0
	for power := from; power <= to; power += step {
		got := narrowDamage(r, attack, defense, power, combat.PermilleBase, combat.PermilleBase)
		if got < previous {
			drops++
		}
		previous = got
	}
	if drops == 0 {
		t.Fatalf("the old expression is monotone across %d..%d, so this range does not straddle the wrap",
			from, to)
	}
	t.Logf("the old expression dropped %d times across the same range", drops)
}

// TestAQuotientPastTheTypeSaturates covers the guard on the division.
//
// bits.Div64 panics when the quotient will not fit, and a panic inside the
// damage formula is worse than the wrap it replaced: it takes the battle down
// rather than printing a wrong number. The answer is the widest figure the type
// holds, which is a bound on the type rather than on the design — the largest
// effective health the progression limits allow is 11,500, so anything reaching
// here already kills whatever it touches.
//
// The last case is the one that would still panic if only the second half of the
// guard were kept: the high word is below the divisor, so bits.Div64 is happy,
// and the quotient is nonetheless past what an int64 holds.
func TestAQuotientPastTheTypeSaturates(t *testing.T) {
	r := rules()
	cases := []struct {
		name       string
		attack     int64
		defense    int64
		multiplier int
		affinity   int
		crit       int
	}{
		{"every factor at its worst and a power past anything", saturatedAttack, 0,
			math.MaxInt64, strongestAffinity, r.CriticalMultiplier},
		{"a power past what the 128-bit product itself holds", math.MaxInt64, 0,
			math.MaxInt64, math.MaxInt64, math.MaxInt64},
		// The affinity of 1001 is doing real work. At a flat PermilleBase the
		// quotient lands on math.MaxInt64 exactly and the guard is never asked;
		// one part per thousand above neutral pushes it one step past, with the
		// numerator's high word (149,999,999,999) still comfortably below the
		// divisor (300,000,000,000) so bits.Div64 itself is perfectly happy.
		{"a quotient just past the signed range", math.MaxInt64, 0,
			combat.PermilleBase, combat.PermilleBase + 1, combat.PermilleBase},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			want := exactDamage(r, testCase.attack, testCase.defense,
				testCase.multiplier, testCase.affinity, testCase.crit)
			if want != math.MaxInt64 {
				t.Fatalf("this case resolves to %d, which fits, so it does not reach the guard", want)
			}
			hit := combat.Hit{
				Scaling: testCase.attack, Multiplier: testCase.multiplier,
				Affinity: testCase.affinity, Defense: testCase.defense,
			}
			got := r.Strike(hit)
			if testCase.crit != combat.PermilleBase {
				got = r.CriticalStrike(hit)
			}
			if got != math.MaxInt64 {
				t.Errorf("damage is %d, want it saturated at %d", got, int64(math.MaxInt64))
			}
		})
	}
}

// TestADamageThatCannotResolveDoesNotPanic is the other half of the guard.
//
// ⚠️ **The denominator has an overflow of its own and this change did not fix
// it.** It is a thousand cubed times K plus defence, still in an int64, and a sum
// of 2^55 sends it to exactly nought — Rules.Validate lets a defence constant
// that large through, since it asks only that the constant be positive. Before
// the guard that was a division by zero and the process died. The wrap is
// reported as a separate defect rather than repaired here, so this test does not
// pretend the answer means anything: it asserts only that the formula answers,
// because with a wrapped denominator there is no right figure to want.
//
// ⚠️ The two rows reach the wrap by the two different routes there are, because
// a mutation showed one row could not tell them apart. A huge *constant* is also
// a huge factor of the numerator, so the high word comes out well above nought;
// a huge *defence* leaves the numerator small enough for the high word to be
// nought. Both are caught by the same `high >= divisor` comparison — an unsigned
// high word is never below nought — which is how the guard came to have one
// clause rather than two.
func TestADamageThatCannotResolveDoesNotPanic(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		constant int64
		defense  int64
		attack   int64
		power    int
	}{
		{"a defence constant that wraps the denominator", 1 << 55, 0, 800, 1800},
		{"a defence that wraps it, with the shipped constant", 300, 1<<55 - 300, 1, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			r := rules()
			r.DefenseConstant = testCase.constant
			if err := r.Validate(); err != nil {
				t.Fatalf("these rules are supposed to be ones Validate accepts: %v", err)
			}
			denominator := int64(combat.PermilleBase) * int64(combat.PermilleBase) *
				int64(combat.PermilleBase) * (r.DefenseConstant + testCase.defense)
			if denominator != 0 {
				t.Fatalf("the denominator is %d, so this case no longer reaches the guard", denominator)
			}
			if got := r.Damage(testCase.attack, testCase.defense, testCase.power, combat.PermilleBase); got < r.MinimumDamage {
				t.Errorf("damage is %d, want at least the floor of %d", got, r.MinimumDamage)
			}
		})
	}
}

// TestANonPositiveDefenseConstantDealsNothing covers the fifth factor's guard.
//
// Validate refuses such a rules set, so no loading path reaches this; a Rules
// built by hand does, and the widened numerator converts every factor to
// unsigned, where a negative constant would become an enormous positive one.
func TestANonPositiveDefenseConstantDealsNothing(t *testing.T) {
	for _, constant := range []int64{0, -1, -300} {
		r := rules()
		r.DefenseConstant = constant
		if got := r.Damage(800, 400, 1800, combat.PermilleBase); got != 0 {
			t.Errorf("a defence constant of %d deals %d, want nothing at all", constant, got)
		}
	}
}
