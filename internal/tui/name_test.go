package tui_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

// aNamedTrait is a shipped trait that actually carries an authored name, found
// rather than named.
//
// Naming one would pass vacuously the day that trait loses its name or leaves
// the book, which is the fixture blind spot this repository has been bitten by
// more than once: a test that spells its own subject stops testing the moment
// the data moves under it.
func aNamedTrait(t *testing.T) passive.Passive {
	t.Helper()
	book, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("read the shipped traits: %v", err)
	}
	for _, one := range book.All() {
		if name := strings.TrimSpace(one.Name); name != "" && name != one.ID {
			return one
		}
	}
	t.Fatal("no shipped trait carries a name, so this proves nothing")
	return passive.Passive{}
}

// aNamedSkill is the same question asked of the skill book, and it is allowed to
// find nothing: every shipped skill falls back to the compiled table today, so a
// skill with an authored Name is a case the data does not currently hold.
func aNamedSkill(t *testing.T) (skill.Skill, bool) {
	t.Helper()
	book, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	for _, declared := range book.Skills() {
		if name := i18n.Vi.SkillName(declared); name != "" && name != declared.ID {
			return declared, true
		}
	}
	return skill.Skill{}, false
}

func shapeBook(t *testing.T) *pattern.Book {
	t.Helper()
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("read the shipped shapes: %v", err)
	}
	return shapes
}

// TestATraitBlockDropsItsNameInEnglish is the defect: DetailPassives built its
// heading from passive.Passive.Name rather than from Lang.PassiveName, so a name
// authored once and in Vietnamese was printed unchanged on an English screen.
func TestATraitBlockDropsItsNameInEnglish(t *testing.T) {
	one := aNamedTrait(t)
	drawn := tui.DetailPassives(i18n.En, "tag unit", []passive.Passive{one})
	if !strings.Contains(drawn, one.ID) {
		t.Errorf("the English block does not name the trait %q:\n%s", one.ID, drawn)
	}
	if strings.Contains(drawn, one.Name) {
		t.Errorf("the English block holds the authored name %q:\n%s", one.Name, drawn)
	}
}

// TestATraitBlockKeepsItsNameInVietnamese is the other half, and it is what
// stops the fix being "print the id and never the name".
func TestATraitBlockKeepsItsNameInVietnamese(t *testing.T) {
	one := aNamedTrait(t)
	drawn := tui.DetailPassives(i18n.Vi, "tag unit", []passive.Passive{one})
	want := one.ID + " · " + one.Name
	if !strings.Contains(drawn, want) {
		t.Errorf("the Vietnamese block does not draw %q:\n%s", want, drawn)
	}
}

// TestEveryDetailBlockAsksLangForTheName is the claim this change is actually
// about: the three blocks in describe.go answer the same question the same way.
//
// It can only compare them through what they draw, since a skill, a trait and a
// status are different types — so the assertion is the one property all three
// share, that an English block carries the bare id and no Vietnamese word. A
// block that read its field raw fails here whichever of the three it is, which
// was measured rather than hoped for: reverting Detail to declared.Name fails
// this and nothing else, and reverting DetailPassives to one.Name fails this and
// the trait test beside it.
func TestEveryDetailBlockAsksLangForTheName(t *testing.T) {
	one := aNamedTrait(t)
	blocks := []struct {
		what  string
		drawn string
		leak  string
	}{
		{"trait", tui.DetailPassives(i18n.En, "tag unit", []passive.Passive{one}), one.Name},
	}
	if declared, found := aNamedSkill(t); found {
		blocks = append(blocks, struct {
			what  string
			drawn string
			leak  string
		}{"skill", tui.Detail(i18n.En, declared, shapeBook(t)), i18n.Vi.SkillName(declared)})
	}
	for _, one := range blocks {
		if one.leak == "" {
			t.Errorf("the %s block has no name to drop, so this proves nothing", one.what)
			continue
		}
		if strings.Contains(one.drawn, one.leak) {
			t.Errorf("the English %s block holds the name %q:\n%s",
				one.what, one.leak, one.drawn)
		}
	}
}

// TestATraitBlockWithNothingHeldStillSaysSo holds the branch above the loop: a
// unit carrying no trait is answered rather than framed blank.
func TestATraitBlockWithNothingHeldStillSaysSo(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		drawn := tui.DetailPassives(lang, "tag unit", nil)
		if strings.TrimSpace(drawn) == "" {
			t.Errorf("%v draws nothing for a unit with no trait", lang)
		}
		if !strings.Contains(drawn, lang.DescribePassive(passive.Passive{})) {
			t.Errorf("%v does not say a unit holds no trait:\n%s", lang, drawn)
		}
	}
}
