// The ninth narrow product, and the only one of them a board cannot see.
//
// This file is in package battle rather than battle_test, and it is the one file
// here that is. The reason is the same one internal/seed and internal/wire give
// for their own white-box files: the unexported thing IS the measurement. Eight
// of the nine products #236 widened are observable through a consequence — a
// unit that did or did not take damage, a skill Suggest did or did not pick —
// and `carry_test.go` next door measures them exactly that way. The wall of
// block charges `pastAWall` subtracts has no such consequence, and that is
// arithmetic rather than a missing fixture.
//
// ⚠️ **Measured, so the choice of instrument has its input.** Over 200,431
// per-strike figures against every charge count the rules allow, `combat.Repeated`
// and the plain `perStrike * charges` it replaced disagree 233,105 times — and in
// 233,105 of those 233,105, `Repeated` answered math.MaxInt64. It has to: the two
// disagree exactly when the true product will not fit, and that is exactly when
// `wide.over` pins. So in EVERY disagreement the correct blow past the wall is
// `damage - math.MaxInt64`, which cannot be positive, so the correct rating is
// nought. A rating of nought is a skill Suggest does not pick, and there is
// nothing downstream of it left for a board to read.
//
// ⚠️ **The clamp in `against` is therefore not what hides it**, which is what
// TODO.md said for one session and what this file corrects. `landed > target.HP`
// only ever pulls a figure DOWN to the target's health; the correct figure here
// is nought, and nought is never clamped. Measured over the same sweep: of the
// 299,548 (per-strike, charges, blow) triples where the two arithmetics land on
// different figures, the clamp makes them equal again in **0**. What hides it is
// the `damage <= 0` guard one line below the product, which catches the second
// wrap — `damage - narrow` overflows too, and lands back under nought.
//
// ⚠️ **And the property is the charges axis rather than the power axis.** The
// instrument this family is held with everywhere else is `TestNoFigureFalls
// AsPowerRises`, *"never smaller for more power"*. It does not transfer to this
// site, and that was measured before it was abandoned: on the CORRECT saturating
// code the figure past a wall falls as power rises in 6 of 24 (connecting,
// charges) pairs on the ladder below, ten falls in all — because the blow
// saturates at math.MaxInt64 before the wall's product does, so the subtraction
// shrinks while the power keeps climbing. A test asserting it would fail on the
// code it is meant to protect. The wall has its own axis and it is monotone on
// that one, which is what is asserted here.
package battle

import (
	"math"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// wallLadder is the same geometric shape combat's own carry sweep walks, and for
// the same reason: the question is about magnitudes, and the interesting values
// are the ones either side of where a 64-bit product stops fitting.
//
// ⚠️ It is not tuned. Two separate rungs — math.MaxInt64 and math.MaxInt64-1 —
// break the property under the narrow product, so the test does not hang on one
// modular coincidence, and no rung is derived from anything in the shipped books.
func wallLadder() []int64 {
	values := []int64{0, 1, 2, 1_000, 1_000_000}
	for value := int64(1_000_000_000); value < math.MaxInt64/8; value *= 8 {
		values = append(values, value)
	}
	return append(values, math.MaxInt64/2, math.MaxInt64-1, math.MaxInt64)
}

// walledBy is a target carrying that many block charges and nothing else, so the
// only thing standing between the blow and the answer is the wall.
func walledBy(charges int) status.Set {
	set := status.Set{}
	if charges <= 0 {
		return set
	}
	return set.With(status.Kind{
		ID: blockStatus, Category: status.Shield, MaxStacks: charges, Duration: 2,
	}, 0, charges)
}

// TestNoWallLetsMoreThroughAsItDeepens is the wall's own monotonicity, and it is
// the one statement about this site that a wrap always breaks and a saturation
// never does.
//
// A charge cancels a strike, so a second charge cancels at least as much as the
// first did and a blow past a deeper wall is never a bigger blow. That holds for
// `combat.Repeated` by construction — a saturating product is non-decreasing in
// its count, so the figure subtracted is non-decreasing and what is left is
// non-increasing — and it is the first thing a narrow product loses: a wrapped
// charge product comes back SMALLER than the shallower wall's, sometimes
// negative, so the deeper wall subtracts less and lets more through.
//
// ⚠️ The charge counts run past `Rules.MaxBlockCharges` deliberately. This
// function is handed whatever stacks the set is holding rather than a figure the
// rules capped, and a sweep that stopped at today's cap would be a test sensitive
// to a number in a book.
func TestNoWallLetsMoreThroughAsItDeepens(t *testing.T) {
	// One expected strike, three, five: the blow saturates at a different power
	// from the wall in each, which is the whole of what makes this arithmetic
	// rather than a single subtraction. `1` is the degenerate end, where the blow
	// is a thousandth of a strike.
	for _, connecting := range []int64{1, 1_000, 3_000, 5_000} {
		for _, perStrike := range wallLadder() {
			damage := combat.Scaled(perStrike, int(connecting))
			previous := int64(math.MaxInt64)
			for charges := range 6 {
				fight := &Battle{}
				target := &Unit{Statuses: walledBy(charges)}
				got := fight.pastAWall(target, skill.Skill{}, perStrike, connecting, damage)
				if got < 0 {
					t.Errorf("a per-strike of %d over %d connecting strikes came "+
						"past a wall of %d as %d, which is negative: a wall cannot "+
						"heal what it is standing in front of",
						perStrike, connecting, charges, got)
				}
				if got > previous {
					t.Errorf("a per-strike of %d over %d connecting strikes came "+
						"past a wall of %d as %d, having come past a wall of %d as "+
						"%d: a deeper wall has let MORE through, which is what a "+
						"charge product that wrapped looks like from here",
						perStrike, connecting, charges, got, charges-1, previous)
				}
				previous = got
			}
		}
	}
}
