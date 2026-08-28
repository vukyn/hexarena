package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheMenuOpensTheSquadBuilder is the wiring: a screen with a view and an
// update and no menu entry is a screen nobody can open, and assigning m.screen
// would pass either way.
func TestTheMenuOpensTheSquadBuilder(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenSquads)
		if m.screen != screenSquads {
			t.Fatalf("the menu entry opened %v", m.screen)
		}
		// The catalogue ships empty, and an empty listing has to say so rather
		// than draw a heading over nothing.
		if body := m.screenContent(); !strings.Contains(body, m.text(i18n.SquadsEmpty)) {
			t.Errorf("the empty listing in %v does not say it is empty:\n%s", lang, body)
		}
	}
}

// TestASquadIsBuiltAndSavedAndComesBack is the whole feature end to end, driven
// through the keys a person presses.
func TestASquadIsBuiltAndSavedAndComesBack(t *testing.T) {
	m, lib, dir := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)

	// n starts one, and the id is typed once.
	m = typeText(t, m, "n")
	if m.squad.mode != squadEdit {
		t.Fatalf("n did not open a squad to build")
	}
	m = typeText(t, m, "greens")

	// enter on the row under the members adds somebody, which opens that unit.
	m = key(t, m, "enter")
	if m.squad.mode != squadUnit {
		t.Fatalf("adding a member did not open it, mode is %v", m.squad.mode)
	}
	if len(m.squad.editing.Units) != 1 {
		t.Fatalf("the squad holds %d members", len(m.squad.editing.Units))
	}
	// A member arrives with a character, a level and a cell, because a blank one
	// would be a form somebody has to fill from nothing.
	unit := m.squad.editing.Units[0]
	if unit.Character == "" || unit.Level == 0 {
		t.Errorf("the member arrived as %+v", unit)
	}
	if !unit.Slot.OnBoard() {
		t.Errorf("the member stands at %s", unit.Slot)
	}

	// Saving now is refused, and refused by the rule rather than by the screen:
	// a placement that brings nothing cannot act.
	m = key(t, m, "esc")
	m = key(t, m, "ctrl+s")
	if m.squad.err == nil {
		t.Fatal("a squad whose member brings no skills was saved")
	}
	if !strings.Contains(m.squad.err.Error(), "chooses no skills") {
		t.Errorf("the refusal is %q, want the loadout rule's own words", m.squad.err)
	}

	// Choose a kit: down to the member, enter to open it, down to the skills
	// field, enter to raise the picker, space to take the row under the cursor.
	m = key(t, m, "up")
	m = key(t, m, "enter")
	for m.squad.field != unitSkills {
		m = key(t, m, "down")
	}
	m = key(t, m, "enter")
	if m.picker == nil {
		t.Fatal("the skills field raised no picker")
	}
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if len(m.squad.unit.Skills) != 1 {
		t.Fatalf("the picker handed back %v", m.squad.unit.Skills)
	}
	// And the choice reached the squad, not only the unit under edit.
	if len(m.squad.editing.Units[0].Skills) != 1 {
		t.Errorf("the squad's own copy brings %v", m.squad.editing.Units[0].Skills)
	}

	m = key(t, m, "esc")
	m = key(t, m, "ctrl+s")
	if m.squad.err != nil {
		t.Fatalf("saving a finished squad was refused: %v", m.squad.err)
	}
	if len(m.squad.notes) == 0 {
		t.Error("a write said nothing")
	}

	// It is on disk, and a library loaded fresh sees it — which is what the
	// simulation will read.
	if _, err := os.Stat(filepath.Join(dir, "squads.json")); err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	reloaded, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	squads := reloaded.Squads()
	if len(squads) != 1 || squads[0].ID != "greens" {
		t.Fatalf("the reloaded catalogue holds %+v", squads)
	}
	// The payoff: it resolves into a side of a battle, ids prefixed so a squad
	// fought against a copy of itself has two halves that can be told apart.
	side, err := squads[0].Take(hex.SideEnemy, reloaded.Characters())
	if err != nil {
		t.Fatalf("the saved squad will not take the field: %v", err)
	}
	if len(side) != 1 || side[0].Side != hex.SideEnemy {
		t.Fatalf("it took the field as %+v", side)
	}
	if !strings.HasPrefix(side[0].ID, "enemy.") {
		t.Errorf("the fielded unit is %q, want the side in front of it", side[0].ID)
	}
	// The listing shows it on the way back.
	m = key(t, m, "esc")
	if body := m.screenContent(); !strings.Contains(body, "greens") {
		t.Errorf("the listing does not hold the squad just saved:\n%s", body)
	}
	_ = lib
}

// TestTwoMembersCannotStandOnOneCell is the chooser stepping over what is taken.
//
// A cell holding two units is not a formation, and a chooser that stops on one
// is a chooser that can be left in a state the save has to reject — so the
// arrow keys skip it rather than the write refusing it later.
func TestTwoMembersCannotStandOnOneCell(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, "pair")
	m = key(t, m, "enter")
	m = key(t, m, "esc")
	// Down off the member just added, onto the row that adds another.
	m = key(t, m, "down")
	m = key(t, m, "enter")
	if len(m.squad.editing.Units) != 2 {
		t.Fatalf("the squad holds %d members", len(m.squad.editing.Units))
	}
	first := m.squad.editing.Units[0].Slot
	if m.squad.editing.Units[1].Slot == first {
		t.Fatalf("both members arrived at %s", first)
	}
	// Walking the chooser all the way round never lands on the taken cell.
	for m.squad.field != unitSlot {
		m = key(t, m, "down")
	}
	for range hex.FormationCols * hex.FormationRows {
		m = key(t, m, "right")
		if m.squad.unit.Slot == first {
			t.Fatalf("the chooser landed on %s, where the other member stands", first)
		}
	}
}

// TestASquadMemberIsReadAgainstItsOwnCharacter is the field that everything
// under it depends on: changing who the unit is empties a kit that was chosen
// from somebody else's learnset.
func TestASquadMemberIsReadAgainstItsOwnCharacter(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, "swap")
	m = key(t, m, "enter")
	for m.squad.field != unitSkills {
		m = key(t, m, "down")
	}
	m = key(t, m, "enter")
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if len(m.squad.unit.Skills) == 0 {
		t.Fatal("nothing was chosen to be lost")
	}
	was := m.squad.unit.Character
	for m.squad.field != unitCharacter {
		m = key(t, m, "up")
	}
	m = key(t, m, "right")
	if m.squad.unit.Character == was {
		t.Skip("the fixture cast has one character, so there is nothing to change to")
	}
	if len(m.squad.unit.Skills) != 0 {
		t.Errorf("the kit survived the character changing: %v", m.squad.unit.Skills)
	}
}

// TestTheSquadKitIsCappedByTheSameRuleTheWriteUses is the live refusal: an
// over-filled kit is a line under the picker rather than a surprise at the save.
//
// It asks the screen's own check rather than pressing space five times, because
// how many skills the fixture cast happens to know at the cap is not what is
// being tested — and a test that skipped itself on a short learnset would be a
// test of the fixture.
func TestTheSquadKitIsCappedByTheSameRuleTheWriteUses(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, "over")
	m = key(t, m, "enter")
	s := m.squad
	character, known := s.character()
	if !known {
		t.Fatal("the member names no character in the book")
	}
	available := character.SkillsAt(s.unit.Level, s.form())
	if len(available) == 0 {
		t.Fatal("the member knows nothing at its own level")
	}
	// One more than the slots hold, taken from what it does know so the refusal
	// is about the count rather than about a name.
	overfull := make([]string, 0, cast.SkillSlots+1)
	for len(overfull) <= cast.SkillSlots {
		overfull = append(overfull, available[len(overfull)%len(available)])
	}
	err := s.refuse(cast.SkillSlots, overfull, "skill", available, cast.Required)
	if err == nil {
		t.Fatalf("%d skills into %d slots was accepted", len(overfull), cast.SkillSlots)
	}
	// Whichever way it is wrong first — too many, or the same one twice — the
	// words are the rule's own and not the screen's.
	if !strings.Contains(err.Error(), "slot(s)") && !strings.Contains(err.Error(), "twice") {
		t.Errorf("the refusal is %q, want the loadout rule's own words", err)
	}
}
