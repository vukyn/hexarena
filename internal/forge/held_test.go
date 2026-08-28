package forge

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// heldOvershoot is the widest a trait may carry a character past the joint
// health-and-defence bound, in parts per thousand of the bound itself.
//
// # The bound is the paper line's, deliberately
//
// A ceiling and the budget bound what an **author** may write into a character
// at the level cap. Going past either in a battle is not a leak: a buff is for
// exactly that, and a trait, and whatever a rune turns out to be. What holds the
// fought line is not the budget but the saturation — modifier.Set.Stat rescales
// every change against ceiling × headroom, so no stack of anything reaches three
// times the ceiling, however much is piled on.
//
// So this is a tripwire and not a bound. A trait is allowed out of the budget and
// this says how far, because a trait that doubled a character's durability would
// be a balance change nobody was told about, and it would pass every other test
// in the repository.
const heldOvershoot = 200

// TestNoTraitCarriesACharacterFarPastTheBudget is that tripwire.
//
// ⚠️ It does not assert that nothing is over, because being over is the design.
// Two shipped pairs are: Squirtle comes to 12413 under endurance and 13043 under
// ballast against a bound of 11500, and both are fine. What is refused is the
// figure growing while nobody is looking.
func TestNoTraitCarriesACharacterFarPastTheBudget(t *testing.T) {
	report := mustInspect(t)
	worst, worstAt := int64(0), ""
	for _, row := range report.Rows {
		for _, carried := range row.Traits {
			if carried.Budget.Max == 0 {
				t.Fatalf("%s under %s reports no bound at all", row.ID, carried.Trait)
			}
			over := carried.Budget.Effective * 1000 / carried.Budget.Max
			if over > worst {
				worst, worstAt = over, row.ID+" under "+carried.Trait
			}
		}
	}
	if worst == 0 {
		t.Fatal("no character reports a line for any trait, so nothing was measured")
	}
	if worst > 1000+heldOvershoot {
		t.Errorf("%s fights at %d.%d%% of the budget, past the %d.%d%% this allows: "+
			"a permanent grant is a stat line the budget never sees, and it has got wide enough "+
			"that the bound means nothing",
			worstAt, worst/10, worst%10, (1000+heldOvershoot)/10, (1000+heldOvershoot)%10)
	}
	t.Logf("the widest is %s at %d.%d%% of the budget", worstAt, worst/10, worst%10)
}

// TestTheHeldTableReportsBothSidesOfTheBound is the table earning its place.
//
// A column that read the same on every row would be a column nobody needs to
// look at. What makes this one worth printing is that the same character is
// inside the bound under one trait and outside it under another — Squirtle is at
// 11285 bare, 11285 under thorns, 12413 under endurance and 13043 under ballast
// — so the figure is a fact about the *pairing* and not about either half.
func TestTheHeldTableReportsBothSidesOfTheBound(t *testing.T) {
	report := mustInspect(t)
	over, inside, pairs := 0, 0, 0
	for _, row := range report.Rows {
		for _, carried := range row.Traits {
			pairs++
			if carried.Budget.Max == 0 {
				t.Fatalf("%s under %s reports no bound at all", row.ID, carried.Trait)
			}
			if carried.Budget.Over() {
				over++
				continue
			}
			inside++
		}
	}
	if pairs == 0 {
		t.Fatal("no character reports a line for any trait, so nothing was measured")
	}
	if over == 0 {
		t.Error("no shipped pair fights outside the bound, so the table says the same thing " +
			"as the one above it and neither is worth two tables")
	}
	if inside == 0 {
		t.Error("every shipped pair fights outside the bound, so the column is a constant")
	}
	t.Logf("%d pairs: %d outside the bound, %d inside", pairs, over, inside)
}

// TestAGatedTraitIsNotCountedAgainstTheBudget is the one exclusion Held makes on
// purpose.
//
// blaze grants its holder a permanent attack buff and holds it only below a third
// health, so counting it here would price a Charizard as though it walked in
// already burning. A trait's gate is checked every turn against a health no
// character has outside a battle, and a figure that assumed the worst turn of a
// fight would not be the figure the table above it is comparing against.
func TestAGatedTraitIsNotCountedAgainstTheBudget(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	gated, err := lib.Passives().Lookup("blaze")
	if err != nil {
		t.Fatalf("look up blaze: %v", err)
	}
	if gated.While == nil {
		t.Fatal("blaze is no longer gated, so this asserts nothing")
	}
	if len(gated.Grants) == 0 {
		t.Fatal("blaze grants nothing, so counting it would change nothing either way")
	}
	character, known := lib.Characters().Get("pokemon.charmander")
	if !known {
		t.Fatal("no charmander to measure")
	}
	values, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve charmander: %v", err)
	}
	held, err := lib.Held(values, []string{"blaze"})
	if err != nil {
		t.Fatalf("hold blaze: %v", err)
	}
	if held != values {
		t.Errorf("blaze moved the stat line to %s from %s: a gated trait is off until its "+
			"condition holds, and no character has a condition outside a battle",
			held.String(), values.String())
	}
	// The same call with the ungated trait has to move something, or the one
	// above is passing because Held does nothing at all.
	moved, err := lib.Held(values, []string{"reckless"})
	if err != nil {
		t.Fatalf("hold reckless: %v", err)
	}
	if moved == values {
		t.Error("reckless moved nothing either, so Held is not applying a grant at all")
	}
}

// mustInspect is the shipped data inspected, which is what these read.
//
// The shipped directory rather than a scratch copy: what is being measured is
// the cast the game ships, and a fixture injected beside it would be a character
// nobody plays showing up in a table about balance.
func mustInspect(t *testing.T) Report {
	t.Helper()
	report, err := Inspect(shippedDataDir)
	if err != nil {
		t.Fatalf("inspect the shipped data: %v", err)
	}
	return report
}
