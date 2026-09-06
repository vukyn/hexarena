package battle_test

import (
	"fmt"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// mirrorRoster is the same squad on both halves, at the same authoring slots, so
// the two sides differ in nothing whatsoever.
//
// The order the two halves are listed in is the only variable: the queue breaks a
// tie by enlistment, so whichever side is listed first acts first. That is a real
// advantage and it is not the subject — what is measured below is that swapping
// it swaps the *outcome*, which is what "the two halves cancel" means.
func mirrorRoster(t *testing.T, members int, allyFirst bool) []battle.Roster {
	t.Helper()
	slots := []hex.Offset{{Col: 1, Row: 1}, {Col: 1, Row: 0}, {Col: 0, Row: 1}}
	if members > len(slots) {
		t.Fatalf("the fixture has %d slots and was asked for %d", len(slots), members)
	}
	half := func(side hex.Side, prefix string) []battle.Roster {
		out := make([]battle.Roster, 0, members)
		for i := range members {
			out = append(out, battle.Roster{
				ID: fmt.Sprintf("%s%d", prefix, i), Side: side, Slot: slots[i],
				Affinity: single("neutral"),
				Stats:    stats(2400, 600, 300, 90),
				Skills:   []string{"strike", "sweep"},
			})
		}
		return out
	}
	ally, enemy := half(hex.SideAlly, "a"), half(hex.SideEnemy, "e")
	if allyFirst {
		return append(ally, enemy...)
	}
	return append(enemy, ally...)
}

// TestASwappedMirrorSwapsItsWinnerAtEverySquadSize is the property the aim order
// exists to keep, and it is asserted per seed rather than on a tally.
//
// A squad against a copy of itself has no advantage over it, so a battle fought
// with the halves listed the other way round must come out the other way round —
// every time, not on average. A tally that summed to a thousand while individual
// seeds disagreed would be two errors cancelling, and that is exactly what was
// happening: the aims were walked in absolute board order, `hex.Place` puts an
// enemy down under a 180 degree rotation, and a rotation reverses rows where a
// column-major walk does not. `Suggest` keeps the first aim that reaches the best
// value, so a tie fell to a different unit depending on which half was being
// looked at.
//
// ⚠️ **One unit a side could never see it**, which is why this fixture runs up to
// three. With a single unit each there is one enemy to aim at and no tie to
// break: measured over 400 seeds, one a side already summed to 1000 per mille
// exactly, while two summed to **1035** and three to **1330**.
func TestASwappedMirrorSwapsItsWinnerAtEverySquadSize(t *testing.T) {
	const seeds = 60
	for members := 1; members <= 3; members++ {
		t.Run(fmt.Sprintf("%d a side", members), func(t *testing.T) {
			held := 0
			for seed := uint64(1); seed <= seeds; seed++ {
				one := mirrorWinner(t, members, true, seed)
				two := mirrorWinner(t, members, false, seed)
				if one == two {
					held++
					if held == 1 {
						t.Errorf("seed %d came out %s whichever half was listed first: "+
							"the two arms are not each other's reflection, so a rate summed "+
							"over both is measuring the order rather than the squads", seed, one)
					}
				}
			}
			if held > 0 {
				t.Errorf("%d of %d seeds kept their winner across the swap", held, seeds)
			}
		})
	}
}

func mirrorWinner(t *testing.T, members int, allyFirst bool, seed uint64) hex.Side {
	t.Helper()
	fight, err := battle.New(books(t), seed, mirrorRoster(t, members, allyFirst))
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(4000); err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
	winner, decided := fight.Winner()
	if !decided {
		t.Fatalf("seed %d, %d a side, ally first %v: the mirror never resolved, so it "+
			"measures nothing", seed, members, allyFirst)
	}
	return winner
}

// TestTheAimOrderIsTheSameSequenceOnEitherHalf is the mechanism under the
// property above, and it is a separate test because the property can be restored
// by accident — a fixture whose ties never arise passes it while the order is
// still wrong.
//
// What is held: the cells a unit is offered, read as the authoring slots they
// were placed from, are the same list for an ally and for an enemy. That is the
// definition of an order that commutes with the mirror.
func TestTheAimOrderIsTheSameSequenceOnEitherHalf(t *testing.T) {
	slotsOffered := func(side hex.Side) []hex.Offset {
		// The acting unit is the one listed first, because the queue breaks a tie
		// by enlistment and every unit here is identical — so listing the half
		// under test first is how this asks that half's question.
		fight, err := battle.New(books(t), 1, mirrorRoster(t, 2, side == hex.SideAlly))
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			t.Fatal("the battle opened with nobody to act")
		}
		actor, known := fight.Unit(prompt.Unit)
		if !known || actor.Side != side {
			t.Fatalf("the first turn went to %q on the %s side, wanted the %s side",
				prompt.Unit, actor.Side, side)
		}
		var out []hex.Offset
		for _, option := range prompt.Options {
			if option.Skill != "strike" {
				continue
			}
			for _, aim := range option.Aims {
				// The slot the cell was placed from: Place is its own inverse on
				// the enemy half, so this reads a cell back into the authoring
				// frame whichever side it belongs to.
				out = append(out, hex.Place(aim.Side(), aim))
			}
		}
		return out
	}
	ally, enemy := slotsOffered(hex.SideAlly), slotsOffered(hex.SideEnemy)
	if len(ally) == 0 {
		t.Fatal("the fixture offered no aims, so nothing is measured")
	}
	if len(ally) != len(enemy) {
		t.Fatalf("an ally is offered %d aims and an enemy %d", len(ally), len(enemy))
	}
	for i := range ally {
		if ally[i] != enemy[i] {
			t.Fatalf("aim %d is slot %s for an ally and %s for an enemy: the order does not "+
				"mirror, so a tie falls to a different unit depending on which half asks",
				i, ally[i], enemy[i])
		}
	}
}
