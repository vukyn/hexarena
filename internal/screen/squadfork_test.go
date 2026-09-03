package screen

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # A placement on a line that FORKS, and the two things the builder used to
// leave unsaid
//
// The read-only views' own fork work — form.go, and the three entries in this
// package's golden — stopped at the browser. The builder has the same state and
// a worse one, because a browser only reads and this screen writes the author's
// file:
//
//   - the form row called an empty stage *furthest*, and on a line with two ends
//     that word names nothing at all; and
//   - Form hands cast.SkillsAt/PassivesAt an empty form, which holds no gate, so
//     the two loadout pickers quietly offered only what every arm learns. A list
//     cannot say why a row is not on it, so nothing anywhere said so.
//
// Both are measured below rather than asserted, because the size of the second
// is a fact about the shipped books: see TestAnUnnamedForkNarrowsBothLists.
//
// ⚠️ **The fix names the state, it does not settle it.** An arm picked for the
// author would be the wrong learnset written into their own file, which is the
// silent wrong answer progression.Line.StageAt refuses — so the empty stage goes
// on refusing, and what changed is that the screen says which arms there are,
// which key names one, and what not naming one already costs. The offer itself
// needed nothing new: the form field is a chooser and StageChoices has listed
// both arms by name since it was written.

// aForkedMember is the builder with one member open on the shipped character
// whose evolution line forks, at a level the fork is open at, naming no arm.
//
// ⚠️ **Found rather than named, and fatal when there is none**, which is
// theShippedFork's own rule: a fork is a fact about the data this fixture
// copies, and a helper that quietly settled for a linear character would turn
// "the books changed" into "every test and every golden entry below measures
// nothing" without a word.
//
// The kit is read at progression.Furthest, which is what the builder itself
// offers in this state — so the member is exactly the one an author would have
// built, narrowed lists and all, rather than one assembled from an arm nobody
// has chosen.
func aForkedMember(t *testing.T, c Context, lib *forge.Library) SquadsScreen {
	t.Helper()
	character, level := theShippedFork(t, lib)
	unit := placement.Placement{
		ID:        "mot",
		Character: character.ID,
		Level:     level,
		Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: 1},
	}
	kit := character.SkillsAt(level, progression.Furthest)
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}
	unit.Skills = kit
	if traits := character.PassivesAt(level, progression.Furthest); len(traits) > 0 {
		unit.Passives = traits[:cast.TraitSlots]
	}
	s := NewSquadsScreen(c)
	s.Editing = placement.Squad{
		ID: "re-nhanh", Name: "đội rẽ nhánh", Units: []placement.Placement{unit},
	}
	s.Baseline = s.Editing.Clone()
	s.IDInput.SetValue(s.Editing.ID)
	s.NameInput.SetValue(s.Editing.Name)
	s = s.EditUnit(0)
	if s.Unit.Stage != "" {
		t.Fatalf("the fixture member names the form %q, so it is not the unnamed "+
			"fork this measures", s.Unit.Stage)
	}
	if arms := s.unnamedArms(s.Unit); len(arms) < 2 {
		t.Fatalf("%s at level %d reaches %v, so the fixture member is on a line that "+
			"does not fork", character.ID, level, progression.StageNames(arms))
	}
	return s
}

// aLinearMember is the same builder on a character whose line does not fork,
// which is the control: every wording and every golden line of this state has to
// be exactly what it was.
func aLinearMember(t *testing.T, c Context, lib *forge.Library) SquadsScreen {
	t.Helper()
	for _, character := range lib.Characters().All() {
		if len(FormArms(character, forkLevel)) != 1 {
			continue
		}
		s := NewSquadsScreen(c)
		unit := placement.Placement{
			ID: "mot", Character: character.ID, Level: forkLevel,
			Slot: hex.Offset{Col: hex.FormationCols - 1, Row: 1},
		}
		s.Editing = placement.Squad{ID: "thang", Units: []placement.Placement{unit}}
		return s.EditUnit(0)
	}
	t.Fatalf("every character in the cast forks at level %d, so nothing here is the "+
		"control the unchanged wording is promised against", forkLevel)
	return SquadsScreen{}
}

// TestAnUnnamedForkIsNotCalledFurthest is the first half of the defect, and it
// fails on the code as it was: stageLabel drew i18n.SquadFurthest for any empty
// stage, so a level with two grown ends was labelled with a superlative that
// picks out neither of them.
//
// It asserts the replacement as well as the absence. A row that simply stopped
// saying *furthest* would leave the author reading a blank where a decision
// belongs — and the arms are named on screen, because "there is a choice here"
// without "and these are the choices" is a refusal that still cannot be acted
// on.
func TestAnUnnamedForkIsNotCalledFurthest(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		s := aForkedMember(t, c, lib)
		arms := s.unnamedArms(s.Unit)
		drawn, _ := s.View(c)
		if strings.Contains(drawn, c.Text(i18n.SquadFurthest)) {
			t.Errorf("in %s the member's form row calls %v %q, and that word names "+
				"neither end of a line that forks:\n%s",
				lang, progression.StageNames(arms), c.Text(i18n.SquadFurthest), drawn)
		}
		if !strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
			t.Errorf("in %s the member's form row does not say the line forks:\n%s", lang, drawn)
		}
		for _, arm := range arms {
			if !strings.Contains(drawn, arm.Name) {
				t.Errorf("in %s the member says nothing about the arm %q, so the reader "+
					"is told there is a choice and not what it is between:\n%s",
					lang, arm.Name, drawn)
			}
		}
	}
}

// TestTheSquadRowSaysTheSameThingAsTheFormRow is the second place the word was
// written: the member's row in the squad under edit resolves an empty stage
// itself, so a fix in the open member alone would leave the row behind it still
// calling a fork furthest.
//
// One key rather than two wordings, for the reason every other pair here is one:
// the row and the field are one fact drawn at two depths.
func TestTheSquadRowSaysTheSameThingAsTheFormRow(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	s := aForkedMember(t, c, lib)
	s = s.Commit()
	s.Mode = SquadEdit
	drawn, _ := s.View(c)
	if strings.Contains(drawn, c.Text(i18n.SquadFurthest)) {
		t.Errorf("the member's row in the squad calls an unnamed fork %q:\n%s",
			c.Text(i18n.SquadFurthest), drawn)
	}
	if !strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
		t.Errorf("the member's row in the squad does not say the line forks:\n%s", drawn)
	}
}

// TestALinearMemberIsStillCalledFurthest is the promise the whole change is
// bounded by: a line with one end says exactly what it always said, and the note
// the fork draws is drawn by nothing here.
func TestALinearMemberIsStillCalledFurthest(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		s := aLinearMember(t, c, lib)
		drawn, _ := s.View(c)
		if !strings.Contains(drawn, c.Text(i18n.SquadFurthest)) {
			t.Errorf("in %s a member on a line that does not fork no longer reads %q:\n%s",
				lang, c.Text(i18n.SquadFurthest), drawn)
		}
		if strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
			t.Errorf("in %s a member on a line that does not fork is told it forks:\n%s", lang, drawn)
		}
	}
}

// TestAnUnnamedForkNarrowsBothLists is the second half of the defect, and it is
// a measurement: it reads how much shorter the two pickers are with no arm named
// than with each arm named, and insists the screen says so.
//
// ⚠️ **It is fatal when the narrowing is nothing**, rather than passing quietly.
// The size of it is a fact about the shipped learnsets — an arm that gates
// nothing costs nothing — so a day when no arm gates an entry is a day this test
// measures the wording of a state that no longer exists, and that has to be read
// rather than inherited.
func TestAnUnnamedForkNarrowsBothLists(t *testing.T) {
	c, lib := start(t, i18n.En)
	s := aForkedMember(t, c, lib)
	arms := s.unnamedArms(s.Unit)
	unnamedSkills := len(s.OpenSkills().Options)
	unnamedTraits := len(s.OpenPassives().Options)
	narrowed := false
	for _, arm := range arms {
		named := s
		named.Unit.Stage = arm.Name
		skills, traits := len(named.OpenSkills().Options), len(named.OpenPassives().Options)
		if skills < unnamedSkills || traits < unnamedTraits {
			t.Fatalf("naming %s offers %d skills and %d traits, fewer than the %d and %d "+
				"an unnamed fork offers — the unnamed reading is meant to be the "+
				"intersection", arm.Name, skills, traits, unnamedSkills, unnamedTraits)
		}
		if skills > unnamedSkills || traits > unnamedTraits {
			narrowed = true
		}
		t.Logf("%s at level %d: unnamed offers %d skills and %d traits; as %s, %d and %d",
			s.Unit.Character, s.Unit.Level, unnamedSkills, unnamedTraits,
			arm.Name, skills, traits)
	}
	if !narrowed {
		t.Fatalf("neither arm of %s gates a skill or a trait at level %d, so nothing "+
			"here measures the narrowing this wording exists to explain",
			s.Unit.Character, s.Unit.Level)
	}
	drawn, _ := s.View(c)
	for _, line := range forkNote(c, arms) {
		if !strings.Contains(drawn, line) {
			t.Errorf("the two lists are short by an arm's worth and the member does "+
				"not say %q:\n%s", line, drawn)
		}
	}
}

// forkNote is the line naming the arms as the screen writes it: wrapped at the
// floor, which is what makes it several lines on a window this narrow.
//
// It is asked for rather than spelt out, and asked for **line by line** rather
// than as one sentence, because a Contains over the whole sentence is a test
// that goes green only while the note happens to fit on one row — which is the
// wrong reason for either answer.
func forkNote(c Context, arms []progression.Stage) []string {
	return WrapWords(c.Text(i18n.SquadForkArms,
		strings.Join(progression.StageNames(arms), " / ")), MinWidth-3)
}

// TestAnUnnamedForkIsRefusedBeforeItIsFought records which bug this is, because
// "builds and then fails to fight" and "builds and fights something" are
// different faults and only the second is a wrong battle.
//
// It is the first: placement.Squad.Take — the one call forge.SaveSquad and
// forge.FightSquads both make — refuses the member outright, so an unnamed fork
// is not a squad that fights the wrong unit, it is a squad that cannot be
// written down. That is what makes this a wording defect rather than an engine
// one, and it is also why the note names the save: before this, the first
// mention of a fork an author got was the refusal under the save key.
func TestAnUnnamedForkIsRefusedBeforeItIsFought(t *testing.T) {
	c, lib := start(t, i18n.En)
	s := aForkedMember(t, c, lib)
	if _, err := s.Editing.Take(hex.SideAlly, lib.Characters()); err == nil {
		t.Fatal("a squad naming no arm of a fork is fielded, so this is a wrong " +
			"battle rather than a refused save and the wording has to say something else")
	} else {
		t.Logf("the save and the fight both refuse it: %v", err)
	}
	// And the way out is the key the note names: the form chooser already holds
	// both arms, and naming either one makes the same member fieldable. Without
	// this the test above would be satisfied by a screen that refused everything.
	for _, arm := range s.unnamedArms(s.Unit) {
		named := s
		named.Unit.Stage = arm.Name
		named = named.Commit()
		if _, err := named.Editing.Take(hex.SideAlly, lib.Characters()); err != nil {
			t.Errorf("naming the arm %q still leaves the squad unfieldable: %v", arm.Name, err)
		}
	}
	if !slices.Contains(s.StageChoices(), s.unnamedArms(s.Unit)[0].Name) {
		t.Errorf("the form chooser does not offer %q, so the note names a key that "+
			"reaches nothing: it offers %v",
			s.unnamedArms(s.Unit)[0].Name, s.StageChoices())
	}
}
