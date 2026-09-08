package seed_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// mustBattleBooks is every book a battle is built from.
func mustBattleBooks(t *testing.T) battle.Books {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	return books
}

// The line these tests field is a shipped ground character with one thing
// changed: its grown form declares an affinity of its own, and it learns a metal
// skill kept for that form.
//
// ⚠️ **It is built here rather than added to internal/testfixture, and that is
// deliberate.** The fixture cast is injected into every scratch data directory
// five packages build, so a character added to it moves three screen goldens and
// several width sweeps — measured once at 656 golden lines for a single skill.
// This one exists only inside the book this file parses, which nothing else
// reads, so the mechanism is fought without any golden moving for it.
//
// It is also why the whole mechanism is worth having a test at all: with no
// shipped stage declaring an element, Character.ElementAt returns the
// character's for every form in the repository and every read site produces
// byte-identical output — a mechanism nothing fields is a mechanism nothing
// measures.
const (
	shiftingID    = "pokemon.shifting-stone"
	shiftingRoot  = "Diglett"
	shiftingGrown = "Dugtrio"
	shiftingMetal = "metal_claw"
)

// aLineThatChangesElement is the cast book with that character in it, and the
// character itself.
func aLineThatChangesElement(t *testing.T) (*cast.Book, cast.Character) {
	t.Helper()
	deps, err := seed.CastDeps()
	if err != nil {
		t.Fatalf("the shipped cast dependencies: %v", err)
	}
	book := mustCast(t)
	original, known := book.Get("pokemon.diglett")
	if !known {
		t.Fatal("the shipped cast has no pokemon.diglett to build a line out of")
	}
	if len(original.Stages) < 2 {
		t.Fatalf("pokemon.diglett has %d forms, and this needs a line that grows",
			len(original.Stages))
	}
	grownAffinity, err := element.Dual(element.Ground, element.Metal)
	if err != nil {
		t.Fatalf("ground/metal is meant to be a legal affinity: %v", err)
	}
	changed := original
	changed.ID = shiftingID
	changed.Name = "Shifting Stone"
	changed.Stages = slices.Clone(original.Stages)
	grown := &changed.Stages[len(changed.Stages)-1]
	if grown.Name != shiftingGrown {
		t.Fatalf("the grown form is %q, and this test names %q", grown.Name, shiftingGrown)
	}
	grown.Element = &grownAffinity
	// Kept for the grown form, which is the whole of what makes the entry legal
	// on a line whose root is ground: an ungated one is refused by the parser,
	// because the root could be fielded holding it.
	changed.Skills = append(slices.Clone(original.Skills), cast.Unlock{
		ID: shiftingMetal, AtLevel: grown.MinLevel, Stages: []string{shiftingGrown},
	})
	grownBook, err := book.Append(deps, changed)
	if err != nil {
		t.Fatalf("a line whose grown form gains an element should parse: %v", err)
	}
	found, known := grownBook.Get(shiftingID)
	if !known {
		t.Fatalf("%s went into the book and did not come back out", shiftingID)
	}
	return grownBook, found
}

// aSideOf is one squad of the given forms of that character, chosen from what
// each form knows rather than from a written-down list.
func aSideOf(t *testing.T, character cast.Character, forms ...string) placement.Squad {
	t.Helper()
	squad := placement.Squad{ID: "forms"}
	for i, form := range forms {
		known := character.SkillsAt(progression.LevelCap, form)
		if len(known) == 0 {
			t.Fatalf("%s knows nothing as %s at the cap", character.ID, form)
		}
		if len(known) > cast.SkillSlots {
			known = known[:cast.SkillSlots]
		}
		squad.Units = append(squad.Units, placement.Placement{
			ID: form, Character: character.ID, Level: progression.LevelCap, Stage: form,
			Slot: hex.Offset{Col: 0, Row: i}, Skills: slices.Clone(known),
		})
	}
	return squad
}

// TestAFormsOwnElementReachesTheBattleThatFieldsIt is the measurement the whole
// mechanism rests on: the affinity a stage declares is what the ENGINE receives.
//
// It resolves through placement.Squad.Take, which is the producer every played
// battle and every saved squad goes through, and then hands the result to
// battle.New — so what is asserted is not that a field round-trips but that a
// battle was built out of two units of one character standing on different
// points of the element chart.
//
// ⚠️ It is a **fielding-time** property and this is where that is visible: the
// two forms are two rosters, settled before the first turn, and nothing inside
// internal/core/battle knows a stage exists. No unit evolves mid-fight, so no
// replay can be affected by any of this.
func TestAFormsOwnElementReachesTheBattleThatFieldsIt(t *testing.T) {
	books := mustBattleBooks(t)
	casted, character := aLineThatChangesElement(t)

	ours, err := aSideOf(t, character, shiftingRoot, shiftingGrown).Take(hex.SideAlly, casted)
	if err != nil {
		t.Fatalf("field both forms: %v", err)
	}
	theirs, err := aSideOf(t, character, shiftingRoot).Take(hex.SideEnemy, casted)
	if err != nil {
		t.Fatalf("field the opponent: %v", err)
	}

	root, grown := ours[0], ours[1]
	if want := character.Element; root.Affinity != want {
		t.Errorf("the root form was fielded as %s, and the character is %s", root.Affinity, want)
	}
	wantGrown, err := element.Dual(element.Ground, element.Metal)
	if err != nil {
		t.Fatalf("ground/metal: %v", err)
	}
	if grown.Affinity != wantGrown {
		t.Errorf("the grown form was fielded as %s, and its stage declares %s",
			grown.Affinity, wantGrown)
	}
	if root.Affinity == grown.Affinity {
		t.Fatal("both forms were fielded as the same affinity, so the stage's own element reached nothing")
	}

	if _, err := battle.New(books, 11, append(slices.Clone(ours), theirs...)); err != nil {
		t.Fatalf("a battle holding both forms should start: %v", err)
	}
}

// TestTheTwoFormsTakeDifferentDamageFromTheSameAttacker is what the mechanism is
// FOR, stated as a number rather than as a field.
//
// The two units are the same character at the same level with the same kit and
// differ in one thing, so the multiplier is the whole difference between them.
// ⚠️ **The sign has to flip between the two attackers.** Ice is strong against
// both halves and stacks to 2250 against the grown form where the root takes
// 1500; grass is strong against ground and weak against metal, so it cancels
// back to 1000 where the root takes 1500. A second element that only ever made
// a unit easier to kill would be a downside wearing a mechanism's clothes, and a
// reading taken against one attacker cannot tell the two apart.
func TestTheTwoFormsTakeDifferentDamageFromTheSameAttacker(t *testing.T) {
	books := mustBattleBooks(t)
	casted, character := aLineThatChangesElement(t)
	ours, err := aSideOf(t, character, shiftingRoot, shiftingGrown).Take(hex.SideAlly, casted)
	if err != nil {
		t.Fatalf("field both forms: %v", err)
	}
	root, grown := ours[0], ours[1]

	for _, attacker := range []struct {
		element   element.Element
		wantRoot  int
		wantGrown int
	}{
		{element.Ice, 1500, 2250},
		{element.Grass, 1500, 1000},
	} {
		gotRoot := books.Chart.MultiplierAgainst(attacker.element, root.Affinity)
		gotGrown := books.Chart.MultiplierAgainst(attacker.element, grown.Affinity)
		if gotRoot != attacker.wantRoot || gotGrown != attacker.wantGrown {
			t.Errorf("%s lands at %d on %s and %d on %s, want %d and %d",
				attacker.element, gotRoot, root.Affinity, gotGrown, grown.Affinity,
				attacker.wantRoot, attacker.wantGrown)
		}
		if gotRoot == gotGrown {
			t.Errorf("%s lands the same on both forms, so the second element is worth nothing",
				attacker.element)
		}
	}
}

// TestTheEngineRefusesAFormFieldedWithTheCharactersElement is why the parser's
// carry rule is "every form" rather than "some form".
//
// battle.enlist applies skill.CanCarry again against the affinity the unit was
// FIELDED with, so a parser that accepted a book as long as one form could carry
// it would hand back a character the engine throws out at the moment somebody
// plays it — the split between the authoring layer and the engine that
// cast.ParseBook exists to close. The tampered roster is the proof that the
// second check is real: the same unit, the same kit, the character's own
// affinity instead of the form's, and the battle will not start.
func TestTheEngineRefusesAFormFieldedWithTheCharactersElement(t *testing.T) {
	books := mustBattleBooks(t)
	casted, character := aLineThatChangesElement(t)

	squad := aSideOf(t, character, shiftingGrown)
	// The metal skill by name, because what is being tampered with is exactly
	// the pairing of that skill with an affinity that cannot hold it.
	squad.Units[0].Skills = []string{shiftingMetal}
	ours, err := squad.Take(hex.SideAlly, casted)
	if err != nil {
		t.Fatalf("field the grown form with its metal skill: %v", err)
	}
	theirs, err := aSideOf(t, character, shiftingRoot).Take(hex.SideEnemy, casted)
	if err != nil {
		t.Fatalf("field the opponent: %v", err)
	}
	if _, err := battle.New(books, 11, append(slices.Clone(ours), theirs...)); err != nil {
		t.Fatalf("the grown form carrying its own element's skill should be enlisted: %v", err)
	}

	tampered := slices.Clone(ours)
	tampered[0].Affinity = character.Element
	if _, err := battle.New(books, 11, append(tampered, theirs...)); err == nil {
		t.Fatal("the engine enlisted a ground unit carrying a metal skill, so nothing downstream is checking the carry rule and the parser's per-form quantifier is measuring nothing")
	}
}

// TestAGatedSkillIsInvisibleToTheFormItIsNotKeptFor is the half that makes the
// mechanism cheap: the enforcement machinery already shipped.
//
// Unlock.Stages is an allowlist and Unlock.Available filters SkillsAt,
// ChooseLoadout and the seed kit through it, so keeping a metal skill for the
// grown form makes it simply absent from the root form's choices — the root can
// never be fielded holding it, at any level, and no new rule was needed to say
// so.
func TestAGatedSkillIsInvisibleToTheFormItIsNotKeptFor(t *testing.T) {
	_, character := aLineThatChangesElement(t)
	asRoot := character.SkillsAt(progression.LevelCap, shiftingRoot)
	asGrown := character.SkillsAt(progression.LevelCap, shiftingGrown)
	if slices.Contains(asRoot, shiftingMetal) {
		t.Errorf("%s is offered to the root form at the cap: %v", shiftingMetal, asRoot)
	}
	if !slices.Contains(asGrown, shiftingMetal) {
		t.Errorf("%s is not offered to the grown form at the cap: %v", shiftingMetal, asGrown)
	}
}
