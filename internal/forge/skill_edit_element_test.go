package forge

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
)

// TestASkillEditIsBlamedOnTheFORMThatCannotCarryIt is the authoring half of the
// per-form carry rule, and it is asked of the walk directly.
//
// whyCharacterBreaks is what names the carrier an edit has broken, and it used
// to ask the question once against the character's own affinity. That is no
// longer a question with one answer: a line may be one element as a root and two
// as a grown form, so a skill kept for the grown form is carried by an affinity
// the character has not got. Asked per character, a perfectly legal line reads as
// broken — and because the walk stops at its first refusal and reports whoever it
// found, the wrong carrier would be named for an edit it has nothing to do with.
//
// ⚠️ **The character is built here rather than saved into a data directory.** The
// walk takes the character as a parameter, so the branch is reachable without a
// write, and a fixture cast that grew a field would move goldens in three other
// packages.
func TestASkillEditIsBlamedOnTheFORMThatCannotCarryIt(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	character, known := lib.Characters().Get("pokemon.diglett")
	if !known {
		t.Fatal("the shipped cast has no pokemon.diglett, so this measures nothing")
	}
	if len(character.Stages) < 2 {
		t.Fatalf("pokemon.diglett has %d forms and this needs a line that grows",
			len(character.Stages))
	}
	grownAffinity, err := element.Dual(element.Ground, element.Metal)
	if err != nil {
		t.Fatalf("ground/metal: %v", err)
	}
	character.Stages = slices.Clone(character.Stages)
	grown := &character.Stages[len(character.Stages)-1]
	grown.Element = &grownAffinity
	learnt := slices.Clone(character.Skills)

	// Kept for the grown form, which is the only form whose affinity holds it.
	character.Skills = append(slices.Clone(learnt), cast.Unlock{
		ID: "metal_claw", AtLevel: grown.MinLevel, Stages: []string{grown.Name},
	})
	reason, asked := whyCharacterBreaks(lib.Skills(), character)
	if !asked {
		t.Fatal("the walk could not resolve the kit, so it asked nothing")
	}
	if reason != nil {
		t.Errorf("a metal skill kept for the metal form is reported as broken: %v", reason)
	}

	// Ungated, the root form could be fielded holding it, and the root form is
	// ground — so this one really is broken, and the refusal has to say which
	// form it is about rather than only which character.
	character.Skills = append(slices.Clone(learnt), cast.Unlock{
		ID: "metal_claw", AtLevel: grown.MinLevel,
	})
	reason, asked = whyCharacterBreaks(lib.Skills(), character)
	if !asked {
		t.Fatal("the walk could not resolve the kit, so it asked nothing")
	}
	if reason == nil {
		t.Fatal("an ungated metal skill on a ground root form is reported as fine, and battle.New will refuse that unit")
	}
	if !strings.Contains(reason.Error(), "metal_claw") {
		t.Errorf("the refusal is %q, want it to name the skill", reason)
	}
}
