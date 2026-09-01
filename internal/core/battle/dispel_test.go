package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestADispelIsPricedForEachOfTheThreeThingsAStripCanTake is the test the
// pricing arm never had, and the reason it never had one is that nothing in the
// shipped data reached it.
//
// `Suggest` has rated a strip pointed at an enemy since dispelled was written,
// and it priced exactly one of the three things such a strip can take. A stat
// buff moves a number, so the hypothetical reads it for free. A **shield** and a
// **regeneration** move no stat at all, so both came back nought — and an
// opponent handed a dispel would decline it in favour of a ten-power poke, which
// is what the first two rows below measured before the two terms were added.
//
// The last row is the design rather than a case: a regeneration on a unit at
// full health is worth nothing to take away, because the healing it owes cannot
// be banked. That is worthHealing's own clamp, read through the same function a
// heal is priced by rather than written again here.
func TestADispelIsPricedForEachOfTheThreeThingsAStripCanTake(t *testing.T) {
	for _, row := range []struct {
		name    string
		holding []string
		hurt    bool
		want    string
	}{
		{"nothing to take", nil, false, "jab"},
		// ⚠️ Read on a target at **full** health, and that is not incidental. On a
		// hurt one the rating prefers the strip anyway, for a reason that is not
		// the shield: deleting the shield term leaves this row green and only the
		// unhurt one goes red. A row that passes without the term it exists to
		// hold is a row that proves nothing.
		{"three block charges", []string{"block", "block", "block"}, false, "unmake"},
		// And a regeneration is read on a hurt one, because the next row is what
		// happens when there is no room to heal into.
		{"three regeneration stacks", []string{"mending", "mending", "mending"}, true, "unmake"},
		{"both at once", []string{"block", "block", "mending"}, true, "unmake"},
		// ⚠️ Not an oversight. Health above the room there is cannot be banked,
		// so a regeneration on a unit standing at full is owed to nobody and
		// taking it away is worth nothing. A dispel that read it as worth
		// something would be spending a turn to deny healing that was never
		// going to land.
		{"a regeneration nobody can bank", []string{"mending", "mending", "mending"}, false, "jab"},
	} {
		t.Run(row.name, func(t *testing.T) {
			fight, err := battle.New(books(t), 7, []battle.Roster{
				{ID: "mine", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
					Affinity: single("neutral"), Stats: stats(3000, 500, 400, 200),
					Skills: []string{"unmake", "jab"}},
				{ID: "theirs", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
					Affinity: single("neutral"), Stats: stats(3000, 500, 400, 1),
					Skills: []string{"jab"}},
			})
			if err != nil {
				t.Fatalf("new battle: %v", err)
			}
			fight.Begin()
			fight.Drain()
			theirs, known := fight.Unit("theirs")
			if !known {
				t.Fatal("no enemy on the board")
			}
			if row.hurt {
				theirs.HP = theirs.MaxHP() / 2
			}
			for _, id := range row.holding {
				kind, err := fight.Books().Statuses.Lookup(id)
				if err != nil {
					t.Fatalf("lookup %s: %v", id, err)
				}
				theirs.Statuses.Apply(kind, 120)
			}
			prompt, err := fight.Advance()
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			choice, found := fight.Suggest(prompt)
			if !found {
				t.Fatal("nothing was suggested, so there is no preference to read")
			}
			if choice.Skill != row.want {
				t.Errorf("the rating chose %q and should have chosen %q", choice.Skill, row.want)
			}
		})
	}
}
