package combat_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/seed"
)

// narrowSwung is Swung exactly as it was written before the product was widened:
// the bonus added, the share taken and the division, all in one int expression.
//
// ⚠️ It is here to prove the widening moved nothing, and it is the one thing in
// this file with a shelf life. Delete it, and TestSwungIsBitIdenticalBelowThe
// Overflow with it, once the change is old enough that nobody is going to ask
// whether it moved a figure. Everything else below is about the behaviour rather
// than about the transition and should stay.
func narrowSwung(power, bonus, share int) int {
	return (power + bonus) * (combat.PermilleBase + share) / combat.PermilleBase
}

// exactSwung is what the expression comes to with no width limit at all,
// saturated at the widest figure an int64 holds.
//
// ⚠️ **math/big here is the argument overflow_test.go already made, and it
// carries over unchanged.** The layer rule binds what `internal/core`'s packages
// compile into — what a battle's determinism rests on — and this is
// `package combat_test`, which no engine build links. It is integer arithmetic
// throughout: big.Int, never big.Float. Quo rather than Div, because Go's `/`
// truncates towards zero and Euclidean division does not; every input here is
// non-negative so the two agree, and the exactness is in saying which one is
// being reproduced.
func exactSwung(power, bonus, share int) int64 {
	quotient := big.NewInt(int64(power))
	quotient.Add(quotient, big.NewInt(int64(bonus)))
	quotient.Mul(quotient, new(big.Int).Add(big.NewInt(int64(combat.PermilleBase)), big.NewInt(int64(share))))
	quotient.Quo(quotient, big.NewInt(int64(combat.PermilleBase)))
	if !quotient.IsInt64() {
		return math.MaxInt64
	}
	return quotient.Int64()
}

// narrowHolds reports whether the old expression could carry this swing — every
// one of its three intermediates inside an int64, which is exactly the range the
// two arithmetics must agree on.
//
// All three are asked rather than only the product, because the old expression
// had three places to wrap and the sum was one of them.
func narrowHolds(power, bonus, share int) bool {
	multiplier := new(big.Int).Add(big.NewInt(int64(combat.PermilleBase)), big.NewInt(int64(share)))
	if !multiplier.IsInt64() {
		return false
	}
	sum := new(big.Int).Add(big.NewInt(int64(power)), big.NewInt(int64(bonus)))
	if !sum.IsInt64() {
		return false
	}
	return new(big.Int).Mul(sum, multiplier).IsInt64()
}

// largestPowerHeld is the greatest power the given predicate still accepts at a
// fixed bonus and share, found by bisection rather than written down.
//
// A walk one power at a time is what overflow_test.go could afford at six
// million; the boundaries here are at 4.9e15 and 4.9e18, so the search has to be
// a bisection. It is still derived from the real arithmetic: the predicate is
// what is asked, and no figure is copied in.
func largestPowerHeld(bonus, share int, holds func(power int) bool) int {
	if !holds(0) {
		return -1
	}
	low, high := 0, math.MaxInt64
	for low < high {
		// Rounded up so the search cannot stall, and written as a half of the
		// gap rather than a sum of the ends: `low + (high-low+1)/2` overflows on
		// the very first step, where the gap is math.MaxInt64 itself.
		middle := low + (high-low)/2 + 1
		if holds(middle) {
			low = middle
		} else {
			high = middle - 1
		}
	}
	return low
}

// The worst bonus and share the shipped book declares, read off it rather than
// invented. Both are measured in TestTheWorstShippedSwingIsTheOneMeasuredAgainst
// so the pair below cannot quietly stop being the worst.
//
// ⚠️ Neither is a bound anybody enforces. `skill.resolve` refuses a negative
// `bonus_power` and an `at_empty` below one and neither above, exactly as it
// bounds `power` below and not above — see TODO.md § *Decided against*. So these
// are facts about the data, and the figures derived from them are measurements of
// the book rather than of a ceiling.
const (
	worstShippedBonus = 1200 // outrage's self_requires
	worstShippedShare = 900  // comeback's self_gradient at_empty
)

// TestTheWorstShippedSwingIsTheOneMeasuredAgainst reads the two figures the
// bounds below are taken at straight off the shipped book.
//
// The constants exist so the measurement is stated where it is used; this is what
// stops them becoming a claim nobody rechecks. It also reports the largest power
// and the largest multiplier the book can actually land, which is the scale every
// bound in this file should be read against.
func TestTheWorstShippedSwingIsTheOneMeasuredAgainst(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	bonus, share, power, landed := 0, 0, 0, 0
	var bonusID, shareID, powerID, landedID string
	for _, shipped := range books.Skills.Skills() {
		if got := shipped.SelfBonus(shipped.SelfRequires.Satisfying()); got > bonus {
			bonus, bonusID = got, shipped.ID
		}
		// The bottom of the caster's bar, which is where a gradient is worth
		// most: the same reading forge's own preview takes.
		if got := shipped.SelfScale(0, 1); got > share {
			share, shareID = got, shipped.ID
		}
		if shipped.Power > power {
			power, powerID = shipped.Power, shipped.ID
		}
		best := combat.Swung(
			shipped.PowerAgainst(shipped.Requires.Satisfying()),
			shipped.SelfBonus(shipped.SelfRequires.Satisfying()),
			shipped.SelfScale(0, 1),
		)
		if best > landed {
			landed, landedID = best, shipped.ID
		}
	}
	if bonus != worstShippedBonus {
		t.Errorf("the largest shipped self bonus is %d on %s, and the constant says %d",
			bonus, bonusID, worstShippedBonus)
	}
	if share != worstShippedShare {
		t.Errorf("the largest shipped gradient share is %d on %s, and the constant says %d",
			share, shareID, worstShippedShare)
	}
	t.Logf("the shipped book's largest declared power is %d (%s); "+
		"the largest multiplier it can land is %d (%s)", power, powerID, landed, landedID)
}

// TestSwungIsBitIdenticalBelowTheOverflow is the property that lets the change
// land without a golden moving: for every swing the old expression could hold,
// the widened one answers the same to the bit.
//
// It sweeps the shipped book's own powers, bonuses and shares rather than
// invented ones, because the figures that must not move are the ones the design
// record is read from — every skill is asked with and without its target's
// condition, with and without its own, and at four points down the caster's bar.
//
// Every case is asserted to fit the old expression as well as to agree with it.
// Without that the sweep could quietly stop testing the transition: a book whose
// powers had grown past the wrap would compare two saturations and pass.
func TestSwungIsBitIdenticalBelowTheOverflow(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	// Full, three quarters, a sliver, and nothing left — the last is the reading
	// forge's preview takes and the one a gradient is worth most at.
	bars := [][2]int64{{4000, 4000}, {3000, 4000}, {1, 4000}, {0, 1}}
	compared := 0
	for _, shipped := range books.Skills.Skills() {
		powers := []int{shipped.Power, shipped.PowerAgainst(shipped.Requires.Satisfying())}
		bonuses := []int{0, shipped.SelfBonus(shipped.SelfRequires.Satisfying())}
		for _, power := range powers {
			for _, bonus := range bonuses {
				for _, bar := range bars {
					share := shipped.SelfScale(bar[0], bar[1])
					if !narrowHolds(power, bonus, share) {
						t.Fatalf("%s at power %d, bonus %d, share %d overflows the old expression: "+
							"the shipped book is supposed to sit well inside it",
							shipped.ID, power, bonus, share)
					}
					want := narrowSwung(power, bonus, share)
					if got := combat.Swung(power, bonus, share); got != want {
						t.Fatalf("%s at power %d, bonus %d, share %d: %d, want %d",
							shipped.ID, power, bonus, share, got, want)
					}
					compared++
				}
			}
		}
	}
	if compared < 500 {
		t.Fatalf("only %d swings compared, which is not a sweep", compared)
	}
}

// TestSwungIsBitIdenticalRightUpToTheOverflow walks the last powers the old
// expression could hold at the worst swing the shipped book declares, so the two
// arithmetics are compared where they are closest to disagreeing rather than only
// in the comfortable middle.
//
// It logs the two bounds this change is measured by: what the old expression held
// and what the new one holds.
func TestSwungIsBitIdenticalRightUpToTheOverflow(t *testing.T) {
	last := largestPowerHeld(worstShippedBonus, worstShippedShare, func(power int) bool {
		return narrowHolds(power, worstShippedBonus, worstShippedShare)
	})
	if last <= 0 {
		t.Fatal("no power fits the old expression, which cannot be right")
	}
	held := largestPowerHeld(worstShippedBonus, worstShippedShare, func(power int) bool {
		return exactSwung(power, worstShippedBonus, worstShippedShare) < math.MaxInt64
	})
	t.Logf("at a bonus of %d and a share of %d the old expression held powers up to %d; "+
		"the widened one holds %d", worstShippedBonus, worstShippedShare, last, held)
	if held <= last {
		t.Fatalf("the widened expression holds %d, no more than the %d the old one did", held, last)
	}
	for _, power := range []int{0, 1, last / 2, last - 2, last - 1, last} {
		want := narrowSwung(power, worstShippedBonus, worstShippedShare)
		if got := combat.Swung(power, worstShippedBonus, worstShippedShare); got != want {
			t.Errorf("power %d: %d, want %d", power, got, want)
		}
		if int64(want) != exactSwung(power, worstShippedBonus, worstShippedShare) {
			t.Errorf("power %d: the oracle says %d and the old expression %d, "+
				"so this case is past the overflow and proves nothing",
				power, exactSwung(power, worstShippedBonus, worstShippedShare), want)
		}
	}
	// One past the end, where the two must now differ: without this the walk
	// above could be inside a range that never reaches the boundary at all.
	beyond := last + 1
	if int64(narrowSwung(beyond, worstShippedBonus, worstShippedShare)) ==
		exactSwung(beyond, worstShippedBonus, worstShippedShare) {
		t.Errorf("a power of %d does not overflow the old expression, so %d was not the boundary",
			beyond, last)
	}
}

// TestTheWrappedSwingIsGone pins the figures the defect was measured at.
//
// Every want is derived from the formula in arbitrary precision rather than read
// off the code being checked, and the wrapped column is what the old expression
// answered. Two things make carrying it worth the room: it says which rows
// actually overflow, so the table cannot quietly stop testing the defect, and it
// names the danger — the wrap does not collapse to something visibly broken, it
// comes back as a large, plausible, wrong number, and on the first row it comes
// back *negative*, which Rules.damage refuses outright and answers with nothing.
func TestTheWrappedSwingIsGone(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		power, bonus, share int
	}{
		{"a power just past what a plain multiply held", 9_223_372_036_854_776, 0, 0},
		{"the same power with the worst shipped swing on it", 9_223_372_036_854_776,
			worstShippedBonus, worstShippedShare},
		{"a power ten times past it", 92_233_720_368_547_760, 0, 0},
		{"a share large enough on its own", 1_000_000_000_000, 0, 90_000_000},
		{"a bonus doing the overflowing", 1, 9_223_372_036_854_775_806, 0},
		{"every term at the widest the type holds", math.MaxInt64, math.MaxInt64, math.MaxInt64},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			want := exactSwung(testCase.power, testCase.bonus, testCase.share)
			if narrowHolds(testCase.power, testCase.bonus, testCase.share) {
				t.Fatalf("this case fits the old expression, so it does not reach the defect")
			}
			wrapped := narrowSwung(testCase.power, testCase.bonus, testCase.share)
			got := combat.Swung(testCase.power, testCase.bonus, testCase.share)
			if int64(got) != want {
				t.Errorf("power %d, bonus %d, share %d: %d, want %d",
					testCase.power, testCase.bonus, testCase.share, got, want)
			}
			t.Logf("power %d, bonus %d, share %d: the old expression answered %d, this one %d",
				testCase.power, testCase.bonus, testCase.share, wrapped, got)
		})
	}
}

// TestSwungIsNonDecreasingInPower is the property the defect actually broke, and
// the one a reader cares about: a skill declaring more power cannot swing for
// less.
//
// The range straddles the old wrap point, and the second half of the test is what
// keeps it honest — it asserts the old expression *fails* here. Without that, a
// range that never reached the overflow would pass this test forever while
// proving nothing.
func TestSwungIsNonDecreasingInPower(t *testing.T) {
	const (
		from = 3_000_000_000_000_000
		to   = 15_000_000_000_000_000
		step = 125_000_000_000_000
	)
	previous := 0
	for power := from; power <= to; power += step {
		got := combat.Swung(power, worstShippedBonus, worstShippedShare)
		if got < previous {
			t.Fatalf("a power of %d swings at %d, less than the %d the power before it swung at",
				power, got, previous)
		}
		previous = got
	}
	drops := 0
	previous = 0
	for power := from; power <= to; power += step {
		got := narrowSwung(power, worstShippedBonus, worstShippedShare)
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

// TestASwingPastTheTypeSaturates covers what a value with nowhere to go comes
// back as.
//
// Swung returns an int and hands it to Rules.damage as skillMultiplier, so even
// an exactly computed enormous product has no room downstream. math.MaxInt64 is
// the answer for the reason the numerator's own guard gives: it is a bound on the
// type rather than on the design, the largest effective health the progression
// limits allow is eleven and a half thousand, and the two saturations compose —
// the multiplier saturates here and the damage it feeds saturates in turn, so the
// blow kills whatever it touches. What it must not do is wrap.
//
// The second case is the one that would still be wrong if only the first half of
// `over`'s guard were kept: the product's high word is 500 against a divisor of
// 1000, so bits.Div64 is perfectly happy and the *quotient* is what will not fit
// a signed 64 bits.
func TestASwingPastTheTypeSaturates(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		power, bonus, share int
	}{
		{"every term at the widest the type holds", math.MaxInt64, math.MaxInt64, math.MaxInt64},
		{"a quotient just past the signed range", math.MaxInt64, 0, 1},
		{"an ordinary power with an absurd share", 1_000_000, 0, math.MaxInt64},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if want := exactSwung(testCase.power, testCase.bonus, testCase.share); want != math.MaxInt64 {
				t.Fatalf("this case resolves to %d, which fits, so it does not reach the guard", want)
			}
			if got := combat.Swung(testCase.power, testCase.bonus, testCase.share); got != math.MaxInt64 {
				t.Errorf("power %d, bonus %d, share %d: %d, want %d",
					testCase.power, testCase.bonus, testCase.share, got, math.MaxInt64)
			}
		})
	}
}

// TestSwungRefusesANegativeTermRatherThanReadingItUnsigned is the guard on the
// three clamps, and every case here is a *behaviour change* rather than a
// preservation: the old expression answered the signed arithmetic and this one
// refuses the term.
//
// None of the three is reachable — see the doc comment for why each — but the
// arithmetic below the clamps is unsigned, so a negative that got through would
// be read as an enormous positive and saturate. Each row is chosen so that
// deleting its own clamp and no other one produces math.MaxInt64 instead of the
// figure below.
//
// ⚠️ The bonus row needs power + bonus to come out *negative*. Unsigned addition
// is two's complement addition, so at a bonus of -500 on a power of 1000 the
// wrapped sum is 500 — exactly what the signed expression gives — and a mutation
// dropping that clamp survives such a case completely.
func TestSwungRefusesANegativeTermRatherThanReadingItUnsigned(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		power, bonus, share int
		want                int
		wasBefore           int
	}{
		{"a negative power", -500, 100, 0, 100, -400},
		{"a bonus that takes the power below nought", 100, -500, 0, 100, -400},
		{"a negative share", 1000, 0, -5000, 1000, -4000},
		{"every term negative at once", -100, -200, -3000, 0, 600},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := narrowSwung(testCase.power, testCase.bonus, testCase.share); got != testCase.wasBefore {
				t.Fatalf("the old expression answers %d here, not the %d this row claims it did",
					got, testCase.wasBefore)
			}
			if got := combat.Swung(testCase.power, testCase.bonus, testCase.share); got != testCase.want {
				t.Errorf("a power of %d with a bonus of %d and a share of %d swings at %d, want %d",
					testCase.power, testCase.bonus, testCase.share, got, testCase.want)
			}
		})
	}
}

// TestSwungReturnsThePowerItWasHandedWhenNothingMovedIt is the identity the
// widening buys over a cheaper overflow check, and the reason the cheaper one was
// refused.
//
// With no bonus and no share the expression is power times a thousand over a
// thousand, so the answer is the power for every power an int holds. A guard that
// saturated whenever the product left an int64 would answer math.MaxInt64 to the
// last three rows — refusing a figure it was handed and could return untouched.
func TestSwungReturnsThePowerItWasHandedWhenNothingMovedIt(t *testing.T) {
	for _, power := range []int{
		0, 1, 2400, 9_223_372_036_854_775, 9_223_372_036_854_776,
		1_000_000_000_000_000_000, math.MaxInt64,
	} {
		if got := combat.Swung(power, 0, 0); got != power {
			t.Errorf("a power of %d with nothing on it swings at %d, want %d", power, got, power)
		}
	}
}
