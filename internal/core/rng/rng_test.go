package rng_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/rng"
)

func TestSameSeedGivesTheSameSequence(t *testing.T) {
	first, second := rng.New(0xC0FFEE), rng.New(0xC0FFEE)
	for i := 0; i < 500; i++ {
		if a, b := first.Next(), second.Next(); a != b {
			t.Fatalf("draw %d differs: %d against %d", i, a, b)
		}
	}
}

func TestDifferentSeedsDiverge(t *testing.T) {
	first, second := rng.New(1), rng.New(2)
	same := 0
	for i := 0; i < 500; i++ {
		if first.Next() == second.Next() {
			same++
		}
	}
	if same > 1 {
		t.Errorf("two seeds agreed on %d of 500 draws", same)
	}
}

// TestSeedZeroIsUsable guards a real failure mode of this generator family: an
// implementation that multiplies the state rather than adding to it produces
// nothing but zeroes from a zero seed.
func TestSeedZeroIsUsable(t *testing.T) {
	source := rng.New(0)
	distinct := make(map[uint64]bool)
	for i := 0; i < 100; i++ {
		distinct[source.Next()] = true
	}
	if len(distinct) < 90 {
		t.Errorf("a zero seed produced only %d distinct draws in 100", len(distinct))
	}
}

func TestStateCanBeCapturedAndRestored(t *testing.T) {
	source := rng.New(42)
	for i := 0; i < 10; i++ {
		source.Next()
	}
	captured := source.State()
	want := make([]uint64, 0, 20)
	for i := 0; i < 20; i++ {
		want = append(want, source.Next())
	}
	source.Restore(captured)
	for i, expected := range want {
		if got := source.Next(); got != expected {
			t.Fatalf("after restoring, draw %d is %d, want %d", i, got, expected)
		}
	}
}

func TestCloneDoesNotDisturbTheOriginal(t *testing.T) {
	source := rng.New(7)
	clone := source.Clone()
	for i := 0; i < 50; i++ {
		clone.Next()
	}
	fresh := rng.New(7)
	for i := 0; i < 50; i++ {
		if a, b := source.Next(), fresh.Next(); a != b {
			t.Fatalf("draw %d after cloning is %d, want %d", i, a, b)
		}
	}
}

func TestIntnStaysInRange(t *testing.T) {
	source := rng.New(99)
	for _, bound := range []int{1, 2, 3, 7, 1000, 1 << 20} {
		for i := 0; i < 2000; i++ {
			got := source.Intn(bound)
			if got < 0 || got >= bound {
				t.Fatalf("Intn(%d) returned %d", bound, got)
			}
		}
	}
}

func TestIntnPanicsOnAnEmptyRange(t *testing.T) {
	for _, bound := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Intn(%d) did not panic", bound)
				}
			}()
			rng.New(1).Intn(bound)
		}()
	}
}

// TestIntnIsNotBiased checks the rejection sampling actually spreads evenly. A
// modulo reduction would skew a range that does not divide the generator's
// period, which for a hit roll is a bias nobody notices without auditing.
func TestIntnIsNotBiased(t *testing.T) {
	const (
		bound   = 7
		draws   = 700_000
		want    = draws / bound
		allowed = want / 40
	)
	source := rng.New(0xBEEF)
	counts := make([]int, bound)
	for i := 0; i < draws; i++ {
		counts[source.Intn(bound)]++
	}
	for value, count := range counts {
		if count < want-allowed || count > want+allowed {
			t.Errorf("value %d came up %d times in %d draws, want about %d", value, count, draws, want)
		}
	}
}

func TestChanceEdges(t *testing.T) {
	source := rng.New(5)
	for i := 0; i < 200; i++ {
		if source.Chance(0) {
			t.Fatal("a chance of zero happened")
		}
		if !source.Chance(1000) {
			t.Fatal("a certainty did not happen")
		}
		if !source.Chance(4000) {
			t.Fatal("a chance above certainty did not happen")
		}
		if source.Chance(-50) {
			t.Fatal("a negative chance happened")
		}
	}
}

func TestChanceMatchesItsProbability(t *testing.T) {
	const draws = 200_000
	for _, permille := range []int{50, 250, 500, 900, 975} {
		source := rng.New(uint64(permille))
		hits := 0
		for i := 0; i < draws; i++ {
			if source.Chance(permille) {
				hits++
			}
		}
		rate := hits * 1000 / draws
		if rate < permille-10 || rate > permille+10 {
			t.Errorf("a chance of %d per mille landed %d per mille of the time", permille, rate)
		}
	}
}

func TestRollAndBetween(t *testing.T) {
	source := rng.New(3)
	for i := 0; i < 2000; i++ {
		if got := source.Roll(6); got < 1 || got > 6 {
			t.Fatalf("Roll(6) returned %d", got)
		}
		if got := source.Between(-5, 5); got < -5 || got > 5 {
			t.Fatalf("Between(-5, 5) returned %d", got)
		}
		if got := source.Between(4, 4); got != 4 {
			t.Fatalf("Between(4, 4) returned %d", got)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("Between with an inverted range did not panic")
			}
		}()
		source.Between(5, 1)
	}()
}
