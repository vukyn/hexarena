package scale_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/scale"
)

func TestSaturateNeverReachesEitherLimit(t *testing.T) {
	const (
		base    = 400
		ceiling = 2400
		floor   = 40
	)
	for _, delta := range []int64{1, 10, 500, 5_000, 500_000, 1 << 40} {
		if got := scale.Saturate(base, delta, ceiling, floor); got >= ceiling {
			t.Errorf("a delta of %d reached %d, the limit is %d and must not be touched", delta, got, ceiling)
		}
		if got := scale.Saturate(base, -delta, ceiling, floor); got <= floor {
			t.Errorf("a delta of %d reached %d, the floor is %d and must not be touched", -delta, got, floor)
		}
	}
}

func TestSaturateAnchors(t *testing.T) {
	const (
		base    = 400
		ceiling = 2400
		floor   = 40
	)
	cases := []struct {
		name  string
		delta int64
		want  int64
	}{
		{"no change", 0, base},
		// A delta the size of the gap covers exactly half of it.
		{"a delta equal to the upward gap", ceiling - base, base + (ceiling-base)/2},
		{"a delta equal to the downward gap", -(base - floor), base - (base-floor)/2},
		// A small delta is worth nearly its face value.
		{"a tenth of the gap", 200, 581},
		{"three tenths of the gap", 600, 861},
		{"a downward tenth", -240, 256},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := scale.Saturate(base, testCase.delta, ceiling, floor); got != testCase.want {
				t.Errorf("delta %d gave %d, want %d", testCase.delta, got, testCase.want)
			}
		})
	}
}

func TestSaturateIsMonotonic(t *testing.T) {
	const (
		base    = 400
		ceiling = 2400
		floor   = 40
	)
	previous := int64(0)
	for delta := int64(-3000); delta <= 3000; delta += 7 {
		got := scale.Saturate(base, delta, ceiling, floor)
		if got < previous {
			t.Fatalf("result fell from %d to %d as delta reached %d", previous, got, delta)
		}
		previous = got
	}
}

func TestSaturateDiminishes(t *testing.T) {
	const (
		base    = 400
		ceiling = 2400
		floor   = 40
		step    = 200
	)
	previousGain := int64(1 << 40)
	for delta := int64(step); delta <= step*12; delta += step {
		gain := scale.Saturate(base, delta, ceiling, floor) - scale.Saturate(base, delta-step, ceiling, floor)
		if gain > previousGain {
			t.Fatalf("the step at delta %d gained %d, more than the previous step's %d", delta, gain, previousGain)
		}
		previousGain = gain
	}
}

// TestSaturateIsScaleInvariant is the property that lets one formula serve stats
// of completely different magnitudes: saturation depends on the ratio of the
// delta to the gap, not on their absolute size.
func TestSaturateIsScaleInvariant(t *testing.T) {
	small := scale.Saturate(200, 100, 400, 0) - 200
	large := scale.Saturate(2000, 1000, 4000, 0) - 2000
	// Equal up to the truncation the smaller scale loses on its single division.
	if difference := large - small*10; difference < 0 || difference > 10 {
		t.Errorf("a gain of %d at one tenth the scale does not match %d", small, large)
	}
}

func TestSaturateAtOrPastALimitDoesNothing(t *testing.T) {
	if got := scale.Saturate(500, 100, 500, 0); got != 500 {
		t.Errorf("a base already at the limit became %d, want 500", got)
	}
	if got := scale.Saturate(500, 100, 400, 0); got != 500 {
		t.Errorf("a base past the limit became %d, want 500", got)
	}
	if got := scale.Saturate(500, -100, 900, 500); got != 500 {
		t.Errorf("a base already at the floor became %d, want 500", got)
	}
}

func TestApply(t *testing.T) {
	cases := []struct {
		value, permille, want int64
	}{
		{1000, scale.Base, 1000},
		{1000, 1500, 1500},
		{1000, 667, 667},
		{3, 667, 2},
		{0, 1500, 0},
	}
	for _, testCase := range cases {
		if got := scale.Apply(testCase.value, testCase.permille); got != testCase.want {
			t.Errorf("Apply(%d, %d) = %d, want %d", testCase.value, testCase.permille, got, testCase.want)
		}
	}
}

// TestTheTwoSharesOverlapAtExactlyOnePoint is the property a pair of traits
// either side of one number rests on, and it is two claims rather than one.
//
// **Nobody falls through.** For every value on the bar at least one of the two
// answers yes, so a trait gated below a threshold and one gated above the same
// threshold cover the whole bar between them — which is why the ends are written
// "at or under" and "at or over" rather than one of them being made strict.
//
// **And the overlap is one point, not a region.** The only value both answer yes
// to is the threshold itself, so the pair costs one point of double cover and
// not a band. Making either end strict would trade that one shared point for a
// one-point hole, which is strictly worse: an overlap is visible in a
// description, a hole is a trait that silently does nothing at one health.
//
// The maximum is chosen so the threshold divides exactly (3000 at 500 per
// thousand is 1500), because on integers the shared point only exists when it
// does — see the sweep's own count for the case where it does not.
func TestTheTwoSharesOverlapAtExactlyOnePoint(t *testing.T) {
	const (
		maximum = int64(3000)
		share   = 500
	)
	both, neither, checked := 0, 0, 0
	for value := int64(0); value <= maximum; value++ {
		below := scale.AtOrBelowShare(value, maximum, share)
		above := scale.AtOrAboveShare(value, maximum, share)
		checked++
		switch {
		case below && above:
			both++
			if value != 1500 {
				t.Errorf("both ends admit a health of %d, and the threshold is 1500", value)
			}
		case !below && !above:
			neither++
			t.Errorf("a health of %d satisfies neither end, so a pair of traits "+
				"either side of the threshold would leave it uncovered", value)
		}
	}
	// The count, so a sweep that ran no iterations cannot pass by walking nothing.
	if checked != int(maximum)+1 {
		t.Fatalf("walked %d values, want %d", checked, maximum+1)
	}
	if both != 1 {
		t.Errorf("%d values satisfy both ends, want exactly one (the threshold)", both)
	}
	if neither != 0 {
		t.Errorf("%d values satisfy neither end, want none", neither)
	}
}

// TestEachShareIsExclusiveOffItsThreshold is the other half of the sentence
// above: either side of the line exactly one of the two answers yes.
//
// Tabled rather than swept, because what a reader wants to check here is the
// three interesting rows and not three thousand.
func TestEachShareIsExclusiveOffItsThreshold(t *testing.T) {
	const share = 500
	cases := []struct {
		name           string
		value, maximum int64
		below, above   bool
	}{
		{"on the threshold, both", 1500, 3000, true, true},
		{"a point under it, only the lower end", 1499, 3000, true, false},
		{"a point over it, only the upper end", 1501, 3000, false, true},
		{"empty, only the lower end", 0, 3000, true, false},
		{"full, only the upper end", 3000, 3000, false, true},
		// Neither end can be answered without a maximum, and both say so the
		// same way rather than one of them dividing by nought.
		{"no maximum, neither", 0, 0, false, false},
		{"no maximum and some health, neither", 10, 0, false, false},
	}
	rows := 0
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			rows++
			if got := scale.AtOrBelowShare(test.value, test.maximum, share); got != test.below {
				t.Errorf("AtOrBelowShare(%d, %d, %d) = %v, want %v",
					test.value, test.maximum, share, got, test.below)
			}
			if got := scale.AtOrAboveShare(test.value, test.maximum, share); got != test.above {
				t.Errorf("AtOrAboveShare(%d, %d, %d) = %v, want %v",
					test.value, test.maximum, share, got, test.above)
			}
		})
	}
	if rows != len(cases) {
		t.Fatalf("ran %d rows, want %d", rows, len(cases))
	}
}

// TestAShareIsNotAFraction is the exactness the cross-multiplication buys, at
// the upper end this time.
//
// 333 parts per thousand is a hair under a third, so a unit at exactly a third
// of its maximum is *above* a threshold written that way and not at or under it.
// Written as a comparison against Apply(maximum, share) the threshold would
// round down to 999 and the answer would be the same by luck; written the other
// way round — Apply(value, Base) against share — it would not. The point of the
// row is that the two ends agree about which side of the line the value is on,
// whatever the arithmetic does.
func TestAShareIsNotAFraction(t *testing.T) {
	const (
		share   = 333
		maximum = int64(3000)
		third   = int64(1000)
	)
	if scale.AtOrBelowShare(third, maximum, share) {
		t.Error("333 per thousand admitted a health of exactly one third as at or under")
	}
	if !scale.AtOrAboveShare(third, maximum, share) {
		t.Error("333 per thousand refused a health of exactly one third as at or over")
	}
	// And the point they really do share, which is 999 rather than 1000.
	if !scale.AtOrBelowShare(999, maximum, share) || !scale.AtOrAboveShare(999, maximum, share) {
		t.Error("the shared point of a 333 threshold on a maximum of 3000 is not 999")
	}
}
