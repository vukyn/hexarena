package forge

import (
	"reflect"
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// `self_bonus` — the power a skill's own condition adds when it holds — as a
// weighable field.
//
// ⚠️ **It is the other half of the pair TODO.md's `ENG-004` is about**, and it
// shipped while the two-number grid that entry asks for did not. The reason is in
// `internal/core/combat`'s swungsurface_test.go: the bonus and the gradient's
// share meet in one expression, as one product, so the pair has no interaction
// for a grid to show. What an author needs is each of the two along its own line,
// and until this field existed only one of the two had one.

// bonusCarrier and bonusSkill are the carrier a self condition's bonus is priced
// on and the skill that carries it, for the reason gradientCarrier and
// gradientSkill are a pair: nothing in the fixture cast fields `vent`, so a
// carrier has to be asked for before the field has a subject.
//
// ⚠️ `vent` declares the condition and **no bonus** — it is a gate that consumes
// — which is exactly the shape this field has to work on. Seven of the twelve
// shipped skills carrying a `self_requires` declare no bonus either, so a control
// row at nought is the ordinary case rather than the awkward one, and the sweep's
// job is to price what putting one there would buy.
const (
	bonusCarrier = "fixture-anime.bonus"
	bonusSkill   = "vent"
)

// bringsTheBonus saves a copy of the weighing's carrier that fields `vent`, on
// the same terms and for the same reason as bringsTheGradient one file over: the
// fixture cast is what a hundred goldens draw, so a carrier a test needs is built
// by the test rather than added to the book.
func bringsTheBonus(t *testing.T, lib *Library) {
	t.Helper()
	character, known := lib.Characters().Get(weighCarrier)
	if !known {
		t.Fatalf("no character is called %q", weighCarrier)
	}
	// ⚠️ **The element moves with the kit, and the kit is replaced whole rather
	// than in its last slot.** `vent` is fire because `swelter` is the fire
	// reserve it spends, and the fixture cast is water/ice and grass/electric —
	// so unlike the gradient's carrier this one cannot be the adept with one
	// entry swapped. `skill.CanCarry` refuses that outright, in the words the
	// authoring layer would give an author: *is water/ice and cannot carry
	// "vent", which is fire*. The three skills beside it are neutral, which is
	// carryable by anybody.
	// ⚠️ **And the fuel has to be authored, because the bench declares none.**
	// `vent` gates on three stacks of `swelter` and nothing in the bench applies
	// one, so the carrier above casts it nought times and the weighing refuses
	// the row — correctly, and in the words it gives an author: *cast 0 time(s)
	// and landed none*. A condition the bench can declare and cannot reach is
	// the shape this repository keeps a list of, so the stoker is built here
	// rather than the gate being quietly dropped: dropping it would author a
	// different skill and price one the game does not have.
	stoker := stokesSwelter(t, lib)
	character.ID = bonusCarrier
	character.Element = fireAffinity(t)
	character.Skills = []cast.Unlock{
		{ID: bonusSkill}, {ID: stoker}, {ID: "strike"}, {ID: "bolt"},
	}
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save %s as %s: %v", weighCarrier, bonusCarrier, err)
	}
	saved, known := lib.Characters().Get(bonusCarrier)
	if !known {
		t.Fatalf("%s was saved and is not in the book", bonusCarrier)
	}
	fielded, err := lib.duellist(saved, progression.LevelCap, "")
	if err != nil {
		t.Fatalf("field %s at the cap: %v", bonusCarrier, err)
	}
	if !slices.Contains(fielded.Skills, bonusSkill) {
		t.Fatalf("%s brings %v at the cap, which does not include %s, so nothing here would be weighed",
			bonusCarrier, fielded.Skills, bonusSkill)
	}
}

// stokesSwelter authors the fuel `vent` spends and returns its id.
//
// It is written into the scratch library rather than into the fixture bench, and
// that is the same decision bringsTheGradient takes about its carrier: the bench
// is what a hundred goldens draw, so a skill only one test needs is a skill only
// one test should see. Adding a thirtieth entry to `testfixture.Skills` would
// move every skill listing in the repository for a fixture nobody else fields.
func stokesSwelter(t *testing.T, lib *Library) string {
	t.Helper()
	const id = "stoke"
	built := skill.Skill{
		ID: id, Element: element.Neutral, Range: 0, Pattern: "single",
		Power: 0, Strikes: 0, Accuracy: scale.Base, Cooldown: 0,
		Target: skill.Self, Scaling: skill.DefaultScaling(),
		SelfApplies: []skill.Application{{Status: "swelter", Chance: scale.Base, Stacks: 3}},
	}
	if err := lib.SaveSkill(built); err != nil {
		t.Fatalf("author %s: %v", id, err)
	}
	return id
}

// fireAffinity is the one element this file needs, asked for through the parser
// rather than written as a constant so a chart that renamed it says so here.
func fireAffinity(t *testing.T) element.Affinity {
	t.Helper()
	member, err := element.Parse("fire")
	if err != nil {
		t.Fatalf("parse fire: %v", err)
	}
	affinity, err := element.Single(member)
	if err != nil {
		t.Fatalf("single fire: %v", err)
	}
	return affinity
}

// TestASelfBonusIsPricedOnTheCarrierThatBringsIt is `self_bonus` end to end, and
// it is written as the gradient's twin because the two fail the same way.
//
// The half that can fail quietly is the second: a `set` that assigned nothing
// would give two rows fought with the same skill under two ids, which is an even
// row and a plausible-looking price of nought.
func TestASelfBonusIsPricedOnTheCarrierThatBringsIt(t *testing.T) {
	lib := sparLibrary(t)
	bringsTheBonus(t, lib)
	shipped, err := lib.Skills().Lookup(bonusSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", bonusSkill, err)
	}
	if shipped.SelfRequires == nil {
		t.Fatalf("%s declares no condition of its own, so there is nothing to put a bonus on",
			bonusSkill)
	}
	declared := WeighSelfBonus.of(shipped)
	// Big enough that the row is not noise and small enough that the sweep can
	// still price it: a bonus the size of `vent`'s own 2400 power saturates the
	// board — the harness says so, *one slot wins 100.0% of what it decides and
	// the other 100.0%* — and a saturated row is refused rather than reported,
	// which is the instrument working.
	const added = 400

	report, err := lib.Weigh(WeighRequest{
		Character: bonusCarrier, Skill: bonusSkill, Field: WeighSelfBonus,
		Values: []int{added}, Level: progression.LevelCap, Seeds: weighSeeds,
	})
	if err != nil {
		t.Fatalf("weigh %s: %v", bonusSkill, err)
	}
	if report.Shipped != declared {
		t.Errorf("the report says the book declares %d, and it declares %d", report.Shipped, declared)
	}
	control := controlRow(t, report)
	if control.Value != declared {
		t.Errorf("the control row is %d rather than the declared %d", control.Value, declared)
	}
	if control.Rate != scale.Base/2 {
		t.Errorf("the control came to %d rather than an even %d: %+v",
			control.Rate, scale.Base/2, control.Tally)
	}
	raised, found := rowAt(report, added)
	if !found {
		t.Fatalf("the sweep has no row at %d: %+v", added, report.Rows)
	}
	if raised.Strikes.Damage == control.Strikes.Damage {
		t.Errorf("a bonus of %d dealt the control's %d damage exactly, so the number never "+
			"reached the battle", added, control.Strikes.Damage)
	}
}

// TestMovingASelfBonusLeavesTheRestOfTheConditionAlone is the half a battle
// cannot see, and it is the reason `set` copies the condition rather than
// building one.
//
// A condition says more than its bonus: which status it reads, how many stacks
// it wants, whether it gates the cast and whether it consumes what it read. A
// weighing that reset any of those would be pricing a different skill — `vent`
// without its gate is castable on a turn the real one refuses — and the report
// would look exactly the same.
func TestMovingASelfBonusLeavesTheRestOfTheConditionAlone(t *testing.T) {
	lib := sparLibrary(t)
	shipped, err := lib.Skills().Lookup(bonusSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", bonusSkill, err)
	}
	before := *shipped.SelfRequires
	if before.Status == "" || !before.Gates || !before.Consume {
		t.Fatalf("%s's condition is %+v, and this test needs one that reads a status, gates "+
			"and consumes — otherwise it asserts about fields nobody set", bonusSkill, before)
	}
	moved := WeighSelfBonus.set(shipped, 777)
	if moved.SelfRequires == shipped.SelfRequires {
		t.Fatal("the variant shares the shipped skill's condition pointer, so moving the " +
			"bonus moves the book's own copy and both sides of the duel at once")
	}
	after := *moved.SelfRequires
	if after.BonusPower != 777 {
		t.Errorf("the bonus came out at %d rather than 777", after.BonusPower)
	}
	// Everything else, asserted by putting the bonus back and comparing whole.
	after.BonusPower = before.BonusPower
	if !reflect.DeepEqual(after, before) {
		t.Errorf("moving the bonus changed the condition from %+v to %+v", before, after)
	}
	// And the book still holds what it held.
	if again, err := lib.Skills().Lookup(bonusSkill); err != nil {
		t.Fatalf("look %s up again: %v", bonusSkill, err)
	} else if again.SelfRequires.BonusPower != before.BonusPower {
		t.Errorf("the book's own %s now declares a bonus of %d", bonusSkill,
			again.SelfRequires.BonusPower)
	}
}
