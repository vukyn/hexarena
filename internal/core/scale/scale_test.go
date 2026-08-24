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
