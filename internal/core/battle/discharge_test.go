package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// marked builds a board of two identical enemies and puts the counter on one of
// them, or on neither.
func marked(t *testing.T, on string, kit []string) battle.Choice {
	return markedAt(t, on, kit, 1, 3)
}

// markedAt is marked with the counter's height and the carrier's health named,
// which the rows about the chain gate and the kill clamp both have to say.
func markedAt(t *testing.T, on string, kit []string, health int64, stacks int) battle.Choice {
	t.Helper()
	fight, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200), Skills: kit},
		{ID: "near", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1), Skills: []string{"jab"}},
		{ID: "far", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1), Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if on != "" {
		unit, known := fight.Unit(on)
		if !known {
			t.Fatalf("no unit %q", on)
		}
		carrying(t, fight, unit.ID, "mark", stacks)
		if health > 1 {
			unit.HP = health
		}
	}
	return chosen(t, fight)
}

// TestTheDischargeIsWorthSomethingToTheTurnThatFiresIt is the cash-in of the
// conduit playstyle, which was priced nowhere at all.
//
// `ArcPower` appeared in exactly one place in the rating — `spendable`, which
// values a stack to decide whether *laying one down* is worth a turn — so the
// turn that spends it was rated on the skill's own power and nothing else. The
// payload was a free rider.
//
// ⚠️ **Both rows, and the first is what makes the second a measurement.** `blunt`
// is the bigger blow on its own numbers, so on a clean target it has to stay the
// answer; what the charge buys is the difference, and a term that made the
// conduit win everywhere would be a power bonus with a story attached.
func TestTheDischargeIsWorthSomethingToTheTurnThatFiresIt(t *testing.T) {
	for _, row := range []struct {
		name string
		on   string
		want string
	}{
		{"a clean target", "", "blunt"},
		{"a target already carrying the counter", "near", "conduit"},
	} {
		if choice := marked(t, row.on, []string{"blunt", "conduit"}); choice.Skill != row.want {
			t.Errorf("against %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}

// TestADischargeIsAimedWhereTheChargeIs is the half `chainFrom` gates, and it is
// the reason the price is taken through that function rather than off the skill.
//
// A conduit aimed at an uncharged unit is simply its own blow, however much
// charge is standing beside it — the aim gates the whole discharge. So on a board
// where one of two identical enemies is carrying the counter, the cast has to go
// to that one.
func TestADischargeIsAimedWhereTheChargeIs(t *testing.T) {
	for _, on := range []string{"near", "far"} {
		choice := marked(t, on, []string{"conduit"})
		aimed := "near"
		if choice.Aim == hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 2}) {
			aimed = "far"
		}
		if aimed != on {
			t.Errorf("with the counter on %s, Suggest aimed the conduit at %s", on, aimed)
		}
	}
}

// TestADischargeIsWorthNothingWhereItCannotFire collects the three clauses the
// first draft of these tests left unmeasured, and every one of them was a branch
// no mutation could break.
//
// ⚠️ That is the failure this repository keeps paying for in a new shape: a
// condition written for a reason, correct, and asserted by nothing. Each row
// below flips when its own clause is deleted.
func TestADischargeIsWorthNothingWhereItCannotFire(t *testing.T) {
	// The chain gate. `deep` needs three stacks and the carrier has one, so
	// chainFrom is empty and the cast is simply its own blow — which is smaller
	// than `blunt`. Pricing the arc off the stacks alone would take the count as
	// one and buy an arc that never fires.
	if choice := markedAt(t, "near", []string{"blunt", "deep"}, 1, 1); choice.Skill != "blunt" {
		t.Errorf("under a conduit's own threshold, Suggest picked %q, want blunt: "+
			"a discharge that cannot fire is worth nothing", choice.Skill)
	}
	// And over it, the same pair goes the other way, so the row above is a
	// threshold rather than a dislike of `deep`.
	if choice := markedAt(t, "near", []string{"blunt", "deep"}, 1, 3); choice.Skill != "deep" {
		t.Errorf("at a conduit's threshold, Suggest picked %q, want deep", choice.Skill)
	}

	// The chance to connect. An arc fires once per strike that did not miss, so a
	// conduit landing two casts in five is worth two fifths of its discharge —
	// and `flicker` is `conduit` at four hundred accuracy and nothing else.
	if choice := markedAt(t, "near", []string{"blunt", "flicker"}, 1, 3); choice.Skill != "blunt" {
		t.Errorf("with a conduit that connects twice in five, Suggest picked %q, want blunt: "+
			"an arc rides a strike, so it is worth what the strike is likely to be",
			choice.Skill)
	}

}

// TestADischargeBuysNothingOnACarrierTheBlowAlreadyFinishes is the clamp read as
// an aim, because it is the only shape that can be read at all: a rating's own
// figures are not observable and the clamp is worth exactly the target's
// remaining health, which is small by construction.
//
// Both enemies carry the counter. One is on a sliver and DANGEROUS — a kill worth
// taking — and the other is healthy and harmless. The blow alone finishes the
// first, so there is nothing left on it for the arc, and the discharge is worth
// what it can actually take off the second. Counting the sliver's health twice is
// what tips it the other way.
func TestADischargeBuysNothingOnACarrierTheBlowAlreadyFinishes(t *testing.T) {
	fight, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 800, 300, 200),
			Skills: []string{"conduit"}},
		// A sliver, with almost nothing to answer with, so the kill it offers is
		// small beside the health the arc would be counting twice.
		{ID: "dying", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 20, 1, 1),
			Skills: []string{"jab"}},
		{ID: "healthy", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	carrying(t, fight, "dying", "mark", 3)
	carrying(t, fight, "healthy", "mark", 3)
	dying, _ := fight.Unit("dying")
	dying.HP = 200

	choice := chosen(t, fight)
	aimed := "dying"
	if choice.Aim == hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 2}) {
		aimed = "healthy"
	}
	if aimed != "healthy" {
		t.Errorf("Suggest aimed the conduit at %s: the blow alone finishes it, so the "+
			"arc has nothing left to take and counting that health twice is what "+
			"makes the sliver look like the better cast", aimed)
	}
}

// aimedByPile puts a deep pile on one enemy and a thin one on the other, then
// reports which the given conduit is aimed at.
//
// ⚠️ **The deep carrier stands where the tie does NOT put the cast.** With two
// cells worth the same the aim goes to the second row — that is what the guard
// tests measured — so a board with the pile there would read the right answer
// whatever the pricing did, which is exactly how the first draft of both tests
// below passed against their own mutations.
func aimedByPile(t *testing.T, id string, deep, thin int) string {
	t.Helper()
	fight, err := battle.New(books(t), 4, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{id}},
		{ID: "deep", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1),
			Skills: []string{"jab"}},
		{ID: "thin", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4000, 60, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	// Identical in every other way, so the pile is the whole of the difference.
	carrying(t, fight, "deep", "mark", deep)
	carrying(t, fight, "thin", "mark", thin)
	if chosen(t, fight).Aim == hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 2}) {
		return "thin"
	}
	return "deep"
}

// TestARepeatingConduitIsWorthTheStacksItCanFindNotTheStrikesItThrows is the
// bound the resolving loop puts on the discharge, and it is what stops one stack
// being worth as much as three.
//
// One blow spends one charge, so a skill that lands three times discharges three
// times — but only while there is something left to spend. A carrier holding one
// stack answers the first strike and nothing after it, and a rating reading the
// strike count alone prices the two carriers the same.
func TestARepeatingConduitIsWorthTheStacksItCanFindNotTheStrikesItThrows(t *testing.T) {
	if at := aimedByPile(t, "volley", 3, 1); at != "deep" {
		t.Errorf("a three-strike conduit was aimed at the carrier holding one stack: " +
			"two of its strikes would find nothing to spend")
	}
}

// TestANukeIsWorthTheWholePileItTakes is the other shape a consume comes in, and
// the arithmetic that separates them.
//
// A detonate spends the whole pile at once, so its arc is `arc_power` times the
// stacks TAKEN rather than times one. Reading the term alone prices a pile of
// three exactly as a single stack, which would make accumulating pointless in the
// one place the whole playstyle cashes out.
func TestANukeIsWorthTheWholePileItTakes(t *testing.T) {
	if at := aimedByPile(t, "nuke", 3, 1); at != "deep" {
		t.Errorf("a detonate was aimed at the carrier holding one stack rather than " +
			"the one holding three: the pile IS the payment")
	}
}
