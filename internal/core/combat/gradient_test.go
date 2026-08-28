package combat_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// TestTheGradientRunsFromFullToEmpty is the shape of the curve, stated at both
// ends and in the middle, because the ends are what an author is writing when
// they pick a number and the middle is what they get for most of a battle.
//
// The figures are the share *added*, so an untouched caster reads nought rather
// than a multiplier of one. That is what lets the log leave the field out
// entirely when nothing was added, the same as Pierce and Refused.
func TestTheGradientRunsFromFullToEmpty(t *testing.T) {
	const atEmpty = 800
	for _, testCase := range []struct {
		name        string
		health, max int64
		want        int
	}{
		{"untouched", 1000, 1000, 0},
		{"a tenth gone", 900, 1000, 80},
		{"half gone", 500, 1000, 400},
		{"a tenth left", 100, 1000, 720},
		{"nothing left", 0, 1000, atEmpty},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := combat.Gradient(testCase.health, testCase.max, atEmpty); got != testCase.want {
				t.Errorf("at %d of %d health the gradient is %d, want %d",
					testCase.health, testCase.max, got, testCase.want)
			}
		})
	}
}

// TestTheGradientIsAStraightLine is the claim that separates it from the
// threshold it sits beside: every point of health lost is worth exactly as much
// as the last, so there is no line for either side to play around.
//
// A threshold is the other shape and `self_requires` is where it lives. If this
// ever stops being linear, the two features have collapsed into one and the
// authored numbers of both stop meaning what they said.
func TestTheGradientIsAStraightLine(t *testing.T) {
	const (
		atEmpty = 1000
		maximum = 1000
		step    = 50
	)
	first := combat.Gradient(maximum-step, maximum, atEmpty) - combat.Gradient(maximum, maximum, atEmpty)
	if first <= 0 {
		t.Fatalf("the first step off full health is worth %d, so the gradient does not rise", first)
	}
	for health := int64(maximum - step); health >= step; health -= step {
		rise := combat.Gradient(health-step, maximum, atEmpty) - combat.Gradient(health, maximum, atEmpty)
		if rise != first {
			t.Errorf("falling from %d to %d is worth %d where the first step was worth %d",
				health, health-step, rise, first)
		}
	}
}

// TestTheGradientIsMonotonic asks the weaker question over every point rather
// than every step, so a curve that was linear at the sample points and wrong
// between them still fails.
func TestTheGradientIsMonotonic(t *testing.T) {
	const maximum = 3100
	previous := 0
	for health := maximum; health >= 0; health-- {
		got := combat.Gradient(int64(health), maximum, 750)
		if got < previous {
			t.Fatalf("the gradient fell from %d to %d as health reached %d", previous, got, health)
		}
		previous = got
	}
	if previous != 750 {
		t.Errorf("at no health the gradient is %d, want %d", previous, 750)
	}
}

// TestTheGradientRefusesToBeAskedNonsense covers the inputs a caller can reach
// but an author cannot write: a unit whose maximum is nought, a skill that
// declares no gradient, and health past either end of its own bar.
//
// Every one of them answers with nought rather than dividing by nought or
// reading past the end of the bar, because a term that cannot say what it means
// must leave the power exactly as it found it.
func TestTheGradientRefusesToBeAskedNonsense(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		health, max int64
		atEmpty     int
		want        int
	}{
		{"no maximum to be a share of", 0, 0, 800, 0},
		{"a negative maximum", 10, -50, 800, 0},
		{"no gradient declared", 100, 1000, 0, 0},
		{"a negative gradient", 100, 1000, -800, 0},
		{"healed past the bar", 1200, 1000, 800, 0},
		{"below the bar", -50, 1000, 800, 800},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := combat.Gradient(testCase.health, testCase.max, testCase.atEmpty); got != testCase.want {
				t.Errorf("the gradient is %d, want %d", got, testCase.want)
			}
		})
	}
}

// TestTheGradientDoesNotOverflow drives the same absurd figures the damage
// formula is checked against, because the product is health times a bonus and
// both are int64 in the caller.
func TestTheGradientDoesNotOverflow(t *testing.T) {
	if got := combat.Gradient(1, 1<<40, 1_000_000); got <= 0 {
		t.Errorf("the gradient at absurd inputs is %d, want a positive multiplier", got)
	}
}

// TestSwungAddsTheBonusBeforeTakingTheShare is the ordering claim, and it had no
// test at all until the authoring preview needed the same expression: swapping
// the two halves of Swung passed the whole suite.
//
// The order is a design. A caster swinging harder swings harder at the power it
// actually has, so the wound is a share of what the skill arrived at rather than
// of what it declared — which is worth most on exactly the skills a gradient is
// written for. The two orders only disagree when both terms are present, which
// is why the cases below carry both.
func TestSwungAddsTheBonusBeforeTakingTheShare(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		power, bonus, share int
		want                int
	}{
		// 1500 doubled, not 2000 plus 500.
		{"both terms", 1000, 500, 1000, 3000},
		{"the bonus alone", 1000, 500, 0, 1500},
		{"the share alone", 1000, 0, 500, 1500},
		{"neither", 1000, 0, 0, 1000},
		// A skill with no power of its own still earns what the bonus brings.
		{"no declared power", 0, 800, 1000, 1600},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := combat.Swung(testCase.power, testCase.bonus, testCase.share); got != testCase.want {
				t.Errorf("a power of %d with a bonus of %d and a share of %d swings at %d, want %d",
					testCase.power, testCase.bonus, testCase.share, got, testCase.want)
			}
		})
	}
}
