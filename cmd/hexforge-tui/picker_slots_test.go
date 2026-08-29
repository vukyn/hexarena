package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The loadout limit as the picker enforces it.
//
// It used to be asked only after enter, by cast.ChooseFrom through
// squadScreen.refuse, so an author picked six skills, learned the answer was
// wrong from a red line under the form, and had to reopen the list to give two
// back. The cap is in toggle now — and the two checks it sits on top of are
// still there, because a loadout hand-edited into squads.json never came through
// a keystroke at all.

// aDeepLearner is the squad in hand with its member pointed at whichever
// character in the book learns the most skills, with the slots already full.
//
// The character is looked up rather than named, which is the rule the fixture
// exists to keep, and it is false when nothing in the book learns more skills
// than a placement may bring — a picker with four rows and four slots can say
// nothing about a fifth.
func aDeepLearner(m model) (model, bool) {
	s := m.squad
	found, most := -1, 0
	for index, character := range s.characters {
		known := len(character.SkillsAt(progression.LevelCap, progression.Furthest))
		if known > most {
			found, most = index, known
		}
	}
	if found < 0 || most <= cast.SkillSlots || len(s.editing.Units) == 0 {
		return m, false
	}
	character := s.characters[found]
	unit := s.editing.Units[0]
	unit.Character, unit.Level, unit.Stage = character.ID, progression.LevelCap, ""
	unit.Skills = character.SkillsAt(unit.Level, progression.Furthest)[:cast.SkillSlots]
	unit.Passives = nil
	s.editing.Units = []placement.Placement{unit}
	m.squad = s.editUnit(0)
	return m, true
}

// aFullSquadKit is the squad builder's kit picker open over a member whose slots
// are already spoken for, which is the state the cap is about.
func aFullSquadKit(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m, deep := aDeepLearner(m)
	if !deep {
		t.Skip("no character in the book learns more skills than a placement may bring")
	}
	m = m.openSquadSkills()
	if m.picker == nil {
		t.Fatal("the kit field raised no picker")
	}
	if len(m.picker.chosen) != cast.SkillSlots {
		t.Fatalf("the picker opened with %d chosen, want the %d slots full",
			len(m.picker.chosen), cast.SkillSlots)
	}
	if len(m.picker.options) <= cast.SkillSlots {
		t.Fatalf("the picker lists %d rows, so there is no row past the slots",
			len(m.picker.options))
	}
	return m
}

// spare is the first row of the picker in hand that is neither chosen nor
// refused, which is the row a full list has to turn away.
func spare(t *testing.T, m model) string {
	t.Helper()
	for _, option := range m.picker.visible() {
		if option.refusal == nil && !slices.Contains(m.picker.chosen, option.id) {
			return option.id
		}
	}
	t.Fatal("every row is either chosen or refused, so nothing is left to refuse for slots")
	return ""
}

// TestAFullKitRefusesAFifthSkill is the cap itself: with the slots spoken for,
// space on a row that has nothing else wrong with it does nothing.
func TestAFullKitRefusesAFifthSkill(t *testing.T) {
	m := aFullSquadKit(t, i18n.Vi)
	fifth := spare(t, m)
	m = pickTo(t, m, fifth)
	m = key(t, m, "space")
	if len(m.picker.chosen) != cast.SkillSlots {
		t.Errorf("a full kit took a fifth skill: %d chosen, want %d",
			len(m.picker.chosen), cast.SkillSlots)
	}
	if slices.Contains(m.picker.chosen, fifth) {
		t.Errorf("the answer holds %q, which arrived past the slots", fifth)
	}
}

// TestAFullKitStillTakesARowBackOut is the branch order, which is the whole
// reason taking one out is the first thing toggle does.
//
// A cap written ahead of it would freeze an over-full loadout solid, and an
// over-full loadout — hand-edited into squads.json — is exactly the thing an
// author opens this picker to fix. Asserted as a round trip rather than as one
// removal, because the removal alone would pass on a picker that had simply
// stopped taking rows at all.
func TestAFullKitStillTakesARowBackOut(t *testing.T) {
	m := aFullSquadKit(t, i18n.Vi)
	held := m.picker.chosen[0]
	m = pickTo(t, m, held)
	m = key(t, m, "space")
	if len(m.picker.chosen) != cast.SkillSlots-1 {
		t.Fatalf("taking a row out of a full kit left %d chosen, want %d",
			len(m.picker.chosen), cast.SkillSlots-1)
	}
	if slices.Contains(m.picker.chosen, held) {
		t.Errorf("the answer still holds %q after it was taken out", held)
	}
	fifth := spare(t, m)
	m = pickTo(t, m, fifth)
	m = key(t, m, "space")
	if !slices.Contains(m.picker.chosen, fifth) {
		t.Errorf("the room just made would not take %q", fifth)
	}
	if len(m.picker.chosen) != cast.SkillSlots {
		t.Errorf("the kit holds %d, want the slots full again at %d",
			len(m.picker.chosen), cast.SkillSlots)
	}
}

// TestTheTraitPickerHoldsOneTraitAtATime is the same claim at the value most
// likely to be special-cased wrong.
//
// One slot is where a cap is tempting to write as a swap — space on a second row
// replacing the first — and a swap would make space two verbs at once, worded
// one way on this picker and another on the kit picker beside it.
func TestTheTraitPickerHoldsOneTraitAtATime(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m, holds := aTraitHolder(m)
	if !holds {
		t.Skip("no character in the book learns a trait, so there is no row to choose")
	}
	m = m.openSquadPassives()
	if m.picker == nil {
		t.Fatal("the trait field raised no picker")
	}
	if len(m.picker.options) <= cast.TraitSlots {
		t.Skip("the character learns no more traits than it may hold")
	}
	if len(m.picker.chosen) != cast.TraitSlots {
		t.Fatalf("the picker opened with %d chosen, want the %d slot(s) full",
			len(m.picker.chosen), cast.TraitSlots)
	}
	held := m.picker.chosen[0]
	second := spare(t, m)
	m = pickTo(t, m, second)
	m = key(t, m, "space")
	if len(m.picker.chosen) != cast.TraitSlots {
		t.Errorf("the trait list holds %d, want %d",
			len(m.picker.chosen), cast.TraitSlots)
	}
	if slices.Contains(m.picker.chosen, second) {
		t.Errorf("%q arrived past the trait slot", second)
	}
	if !slices.Contains(m.picker.chosen, held) {
		t.Errorf("%q was swapped out by the row that should have been refused", held)
	}
}

// anAllowlist is a restriction's character list, which is a picker with no slots
// at all: an author names as many characters as the skill is kept for.
func anAllowlist(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = m.enter(screenSkills)
	m.skills.adding = true
	m = m.openAllowlist(skillFieldKeptForCharacters)
	if m.picker == nil {
		t.Fatal("the allowlist field raised no picker")
	}
	if m.picker.slots != 0 {
		t.Fatalf("the allowlist picker carries %d slot(s), and an allowlist has none",
			m.picker.slots)
	}
	return m
}

// TestAnUncappedPickerTakesMoreThanASquadLoadout is what stops the cap leaking
// into the pickers that must stay uncapped.
//
// Every picker but the squad builder's two is one of these — the five
// restriction allowlists, the statuses a skill inflicts, and the character
// form's own kit — and each takes as many rows as an author names.
func TestAnUncappedPickerTakesMoreThanASquadLoadout(t *testing.T) {
	m := anAllowlist(t, i18n.Vi)
	rows := m.picker.visible()
	if len(rows) <= cast.SkillSlots {
		t.Skipf("the allowlist lists %d rows, which cannot pass a loadout of %d",
			len(rows), cast.SkillSlots)
	}
	wanted := make([]string, 0, cast.SkillSlots+1)
	for _, option := range rows[:cast.SkillSlots+1] {
		wanted = append(wanted, option.id)
		m = pickTo(t, m, option.id)
		m = key(t, m, "space")
	}
	if !slices.Equal(m.picker.chosen, wanted) {
		t.Errorf("an uncapped picker kept %v of the %v it was given", m.picker.chosen, wanted)
	}
}

// TestTheSlotCounterReplacesTheListPosition is the counter, read off the render
// rather than off the field: a picker holding the right number and drawing the
// old wording is exactly the build this is for.
//
// The two are asserted against each other because a list position says nothing
// about what binds — four of nineteen rows is not four of four slots — and both
// languages are rendered because the counter is wording.
func TestTheSlotCounterReplacesTheListPosition(t *testing.T) {
	for _, lang := range i18n.Langs() {
		capped := aFullSquadKit(t, lang)
		content := capped.screenContent()
		slotted := capped.text(i18n.ChoiceSlots, len(capped.picker.chosen), cast.SkillSlots)
		if !strings.Contains(content, slotted) {
			t.Errorf("the %s kit picker draws no slot counter %q", lang, slotted)
		}
		walked := capped.text(i18n.ChoicePosition,
			len(capped.picker.chosen), len(capped.picker.options))
		if strings.Contains(content, walked) {
			t.Errorf("the %s kit picker still draws the list position %q", lang, walked)
		}

		// And the other way on a picker with no slots, so the counter that was
		// there before is the counter those still get. Every count the slot
		// wording could have been drawn with is refused, because which number a
		// leaked cap would print is exactly what the test cannot know.
		open := anAllowlist(t, lang)
		content = open.screenContent()
		walked = open.text(i18n.ChoicePosition,
			len(open.picker.chosen), len(open.picker.options))
		if !strings.Contains(content, walked) {
			t.Errorf("the %s allowlist draws no list position %q", lang, walked)
		}
		for count := range len(open.picker.options) + 1 {
			slotted = open.text(i18n.ChoiceSlots, len(open.picker.chosen), count)
			if strings.Contains(content, slotted) {
				t.Errorf("the %s allowlist draws a slot counter %q, and it has no slots",
					lang, slotted)
			}
		}
	}
}

// TestAnOverFullLoadoutOpensAndCanBeFixed is the state the cap cannot produce
// and the counter is still drawn for: a loadout past its slots, hand-edited into
// squads.json, opened here to be brought back inside them.
//
// The refusing style it is drawn in cannot be asserted — every test here runs
// under NO_COLOR, which is the palette's own rule that meaning never lives in
// colour — so what is measured is the reading and the way out.
func TestAnOverFullLoadoutOpensAndCanBeFixed(t *testing.T) {
	m := aFullSquadKit(t, i18n.Vi)
	m.picker.chosen = append(m.picker.chosen, spare(t, m))
	over := len(m.picker.chosen)
	if !strings.Contains(m.screenContent(), m.text(i18n.ChoiceSlots, over, cast.SkillSlots)) {
		t.Errorf("an over-full kit draws no counter reading %d of %d", over, cast.SkillSlots)
	}
	m = pickTo(t, m, m.picker.chosen[0])
	m = key(t, m, "space")
	if len(m.picker.chosen) != over-1 {
		t.Errorf("an over-full kit would not give a row back: %d chosen, want %d",
			len(m.picker.chosen), over-1)
	}
}
