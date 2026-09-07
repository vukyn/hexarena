// The window the credit's share has to sit in, held as the two numbers it was
// measured at.
//
// This file is in package battle rather than battle_test for the reason
// hiding_term_test.go and carry_wall_test.go give: the two boards that found
// these edges are in guardcredit_test.go and pool_test.go and they hold the
// BEHAVIOUR, one edge each. What neither can say is where the edge actually is —
// each of them is a single point, green on one side of its own boundary and red
// on the other, and a reader looking at either one cannot tell whether the
// shipped share is sitting on the edge or in the middle of a wide window.
package battle

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/scale"
)

// TestTheGuardCreditSitsInsideItsMeasuredWindow records where the share came
// from, in the shape TestRangeLadder and TestDefenseCurveAnchors use: a design
// figure written down so the shipped value cannot drift from the reading it was
// taken at.
//
// ⚠️ **The entry that costed this fix said no sweep could choose the share**, and
// that was true of the sweep it ran: asked only whether the guarded board
// resolves, every value from a tenth upwards answers yes and answers identically,
// because the option being repaired was reading exactly nought and only has to
// become positive. A step is not a curve and cannot be read for a maximum.
//
// A second board closes it from the other side. The two edges, swept at five per
// mille either way:
//
//   - **50** is the floor, and it is a truncation rather than a preference. The
//     credit is one division; on the twenty-one-point blow the guarded mirror
//     throws, a share of 45 comes back nought and that board stands still again —
//     which the resolve-only sweep would have scored as a success.
//   - **999** is the ceiling, and it is a rule rather than a reading: at parity a
//     target carrying a guard and a target standing bare rate the same, `take`
//     keeps the first aim it saw, and pastAPool's own finding — a rating
//     preferring the target it cannot empty — comes back through the other door.
//
// ⚠️ **An earlier ceiling of 199 was real and is gone**, and it is worth knowing
// why rather than deleting: it came from a flat credit, which paid for a bite out
// of a TIMED guard as well. That version out-rated an unblockable blow behind a
// barrier at 200 — and, worse, turned every shipped mirror endless at 25. Once the
// credit was narrowed to permanent guards both of those boards stopped depending
// on the share at all.
//
// If the guarded fixture is re-tuned, re-take the floor before moving the
// constant: it is a fact about that fixture's blow size and does not survive a
// change to it.
func TestTheGuardCreditSitsInsideItsMeasuredWindow(t *testing.T) {
	const (
		floor   = 50  // below this the guarded mirror stalls again
		ceiling = 999 // above this a point of guard is worth a point of health
	)
	if guardCredit < floor {
		t.Errorf("the credit is %d, under the measured floor of %d: the division truncates "+
			"to nought on a small blow and the guarded board stands still again",
			guardCredit, floor)
	}
	if guardCredit > ceiling {
		t.Errorf("the credit is %d, over the ceiling of %d: a point taken out of a guard "+
			"is worth a point of health, so a guarded target and a bare one rate the same "+
			"and the aim walk decides which the rating prefers", guardCredit, ceiling)
	}
	// And the rule the window sits inside, which holds whatever the two boards
	// are re-tuned to: a point taken out of a guard is worth less than a point of
	// health, because health is what the battle is decided on.
	if guardCredit >= scale.Base {
		t.Errorf("the credit is %d against a base of %d: a point of guard is worth a point "+
			"of health, so the rating has no reason to prefer the enemy it can kill",
			guardCredit, scale.Base)
	}
}
