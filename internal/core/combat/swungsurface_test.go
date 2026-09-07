package combat_test

import (
	"fmt"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// Whether the caster's two own terms make a SURFACE, asked of the one expression
// that composes them.
//
// `Swung(power, bonus, share)` is `(power + bonus) * (1000 + share) / 1000`: the
// bonus is added to one factor and the share to the other, and the two meet in a
// single product with a single truncation. So the pair is a **rank-one** product,
// and the answer is a function of `(power + bonus) * (1000 + share)` alone rather
// than of the two numbers apart.
//
// ⚠️ **This is the whole of TODO.md's `ENG-004`.** That entry asks for a
// two-number report over exactly this pair, and the reason it is refused lives
// here rather than in the tool: a grid over the two would spend its battles
// rediscovering that every cell on one hyperbola is the same cell.

// TestSwungIsAProductAndNotASurface holds the collapse as the identity it is.
//
// The pairs in each row differ in **both** numbers and agree in the product,
// which is what makes this a test of separability rather than of arithmetic: a
// function that read the bonus and the share apart — weighted them differently,
// capped one, rounded between them — would come apart on at least one row.
func TestSwungIsAProductAndNotASurface(t *testing.T) {
	rows := []struct {
		power int
		pairs [][2]int // bonus, share
	}{
		// (1000+b)(1000+s) = 2,000,000, four ways. The product is a whole number
		// of thousands, so nothing is truncated and the row is the identity in
		// its plainest form.
		{1000, [][2]int{{0, 1000}, {1000, 0}, {250, 600}, {600, 250}}},
		// (500+b)(1000+s) = 1,500,000 — a different power, so the identity is
		// not being read off one magnitude. The last pair floors its share and
		// therefore lands on a different hyperbola on purpose.
		{500, [][2]int{{1000, 0}, {250, 1000}, {2000, -600}}},
		// ⚠️ **And a product that is NOT a whole number of thousands**, which is
		// the row that says the truncation does not break the identity:
		// 1500 × 1333 and 1333 × 1500 both come to 1,999,500 and both truncate
		// to the same figure, because the division is taken once, of the product.
		{1000, [][2]int{{500, 333}, {333, 500}}},
	}
	for _, row := range rows {
		t.Run(fmt.Sprintf("power %d", row.power), func(t *testing.T) {
			// The row's figure comes from its first pair rather than being
			// written down twice, so a row cannot be wrong about itself.
			want := combat.Swung(row.power, row.pairs[0][0], row.pairs[0][1])
			for _, pair := range row.pairs {
				bonus, share := pair[0], pair[1]
				if share < 0 {
					// Swung floors a negative share at nought, so such a pair
					// sits on a different hyperbola and is not this row's
					// subject. It is here to say the flooring is known rather
					// than to be asserted against the row.
					if got := combat.Swung(row.power, bonus, share); got <= want {
						t.Errorf("bonus %d with a share of %d came to %d, at or under the row's %d: "+
							"a floored share should still leave the bonus paying for itself",
							bonus, share, got, want)
					}
					continue
				}
				if got := combat.Swung(row.power, bonus, share); got != want {
					t.Errorf("power %d: bonus %d share %d comes to %d where bonus %d share %d "+
						"comes to %d, on the same product — the two terms are being read apart",
						row.power, bonus, share, got, row.pairs[0][0], row.pairs[0][1], want)
				}
			}
		})
	}
}

// TestSwungReadsBothTermsAtAll is the mutation guard under the rows above: a row
// of equal products is also satisfied by an expression that ignores one of the
// two terms entirely, so this asks that each of them moves the answer.
func TestSwungReadsBothTermsAtAll(t *testing.T) {
	const power = 1000
	base := combat.Swung(power, 0, 0)
	if bonused := combat.Swung(power, 500, 0); bonused <= base {
		t.Errorf("a bonus of 500 moved %d to %d, so the bonus is not being read", base, bonused)
	}
	if shared := combat.Swung(power, 0, 500); shared <= base {
		t.Errorf("a share of 500 moved %d to %d, so the share is not being read", base, shared)
	}
}
