package forge

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// heldOvershoot is the widest a trait may carry a character past the joint
// health-and-defence bound before this stops being a figure worth looking at and
// becomes a bound nobody is keeping, in parts per thousand of the bound itself.
//
// It is set above where the shipped cast sits rather than at it. The number is
// not a target — the point of the table this guards is that the bound is checked
// against a line nobody fights on, and closing that properly means deciding
// whether 11500 belongs to the paper line or the fought one, which moves every
// stat line in the game. What this refuses in the meantime is the hole getting
// quietly wider: a new trait handing out half again as much durability as the
// bound allows would pass every other test in the repository.
const heldOvershoot = 200

// TestNoTraitCarriesACharacterFarPastTheBudget is the bound on the hole.
//
// ⚠️ It does not assert that nothing is over. Two shipped pairs are — Squirtle
// absorbs 12413 under endurance and 13043 under ballast against a bound of 11500
// — and a test that refused them would have to refuse endurance, which is the
// default trait of three of the four archetypes and therefore the whole cast.
// The figure is on screen now, in `hexforge check`; this keeps it from growing
// while nobody is looking.
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

// TestTheHeldBudgetIsWarnedAboutExactlyWhereItIsOver is the wiring, and it is
// asserted in both directions because only one of them is interesting.
//
// A warning on everything is a warning on nothing: three of the four shipped
// characters stay inside the bound under every trait they can carry, and if they
// were warned about too then the two that matter would be four lines down a list
// nobody reads.
func TestTheHeldBudgetIsWarnedAboutExactlyWhereItIsOver(t *testing.T) {
	report := mustInspect(t)
	warned := map[string]bool{}
	for _, warning := range report.Warnings {
		held, is := warning.(*HeldBudgetWarning)
		if !is {
			continue
		}
		warned[held.ID+"/"+held.Trait] = true
	}
	over, inside := 0, 0
	for _, row := range report.Rows {
		for _, carried := range row.Traits {
			key := row.ID + "/" + carried.Trait
			switch {
			case carried.Budget.Over() && !warned[key]:
				t.Errorf("%s absorbs %d of %d and nothing said so",
					key, carried.Budget.Effective, carried.Budget.Max)
			case carried.Budget.Over():
				over++
			case warned[key]:
				t.Errorf("%s absorbs %d of %d, which is inside the bound, and was warned about anyway",
					key, carried.Budget.Effective, carried.Budget.Max)
			default:
				inside++
			}
		}
	}
	if over == 0 {
		t.Fatal("no shipped pair is over the bound, so the warning is never reached")
	}
	if inside == 0 {
		t.Fatal("every shipped pair is over the bound, so the warning says nothing")
	}
	t.Logf("%d pairs over the bound, %d inside", over, inside)
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
