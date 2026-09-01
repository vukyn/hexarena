package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
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
		// The fixture's catalogue starts empty, and an empty listing has to say
		// so rather than draw a heading over nothing.
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

// aSquadPicker is a squad under construction with one of its two loadout
// pickers open, driven through the keys a person presses.
func aSquadPicker(t *testing.T, lang i18n.Lang, id string, field int) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, id)
	m = key(t, m, "enter")
	for m.squad.field != field {
		m = key(t, m, "down")
	}
	m = key(t, m, "enter")
	if m.picker == nil {
		t.Fatalf("the field raised no picker")
	}
	return m
}

// TestTheSquadTraitPickerReadsTheTraitBook is the bug the reading state
// uncovered, kept as a regression.
//
// The builder raised its trait picker as pickSkills while handing it trait ids,
// so every row looked itself up in the skill book, missed, and drew "unknown
// skill" in red where its detail belongs. Nothing caught it because the fixture
// cast learns no traits: every test that opened that picker opened an empty one.
func TestTheSquadTraitPickerReadsTheTraitBook(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenSquads)
		m.squad = someSquad(t, m)
		m, holds := aTraitHolder(m)
		if !holds {
			t.Skip("no character in the book learns a trait, so there is no row to draw")
		}
		m = m.openSquadPassives()
		if m.picker == nil || len(m.picker.Options) == 0 {
			t.Fatal("the trait field raised no picker with rows in it")
		}
		if m.picker.Kind != pickPassives {
			t.Errorf("the trait picker is a %v, so its rows are read out of the wrong book",
				m.picker.Kind)
		}
		body, _ := m.picker.View(m.ctx())
		for _, option := range m.picker.Options {
			// A trait sharing a name with a skill would prove nothing either
			// way, so only the ids the skill book really refuses are asked
			// about — and what they must not put on screen is its refusal.
			_, err := m.lib.Skills().Lookup(option.ID)
			if err == nil {
				continue
			}
			if refusal := m.lang.Error(err); strings.Contains(body, refusal) {
				t.Errorf("the %s trait picker draws %q where %s's detail belongs:\n%s",
					lang, refusal, option.ID, body)
			}
		}
		// And it draws the trait's own name, which is what the listing puts
		// beside an id -- nothing in English, where a data name is not
		// translated and the id is the whole row.
		rows := m.picker.Visible()
		held, err := m.lib.Passives().Lookup(rows[m.picker.Cursor].ID)
		if err != nil {
			t.Fatalf("the row under the cursor is not a trait the book holds: %v", err)
		}
		if name := m.lang.PassiveName(held); name != "" && !strings.Contains(body, name) {
			t.Errorf("the %s trait picker does not name %s:\n%s", lang, held.ID, body)
		}

		// And ? reads the trait out of the same book, in the sentences the
		// blurb screen gives -- which is the half a row's detail is too narrow
		// to carry, and the whole reason an English row being a bare id costs
		// nothing.
		m = typeText(t, m, "?")
		if !m.picker.Reading {
			t.Fatalf("? opened no description on the %s trait picker", lang)
		}
		body, _ = m.picker.View(m.ctx())
		if !strings.Contains(body, m.lang.GlossedPassive(held)) {
			t.Errorf("the %s description does not name %s:\n%s", lang, held.ID, body)
		}
		for _, line := range traitSentences(m.ctx(), held) {
			if sentence := strings.TrimSpace(line); !strings.Contains(body, sentence) {
				t.Errorf("the %s description of %s is missing %q:\n%s",
					lang, held.ID, sentence, body)
			}
		}
	}
}

// TestTheSquadPickerDescribesTheRowUnderItsCursor is what ? is for: an author
// choosing four of nine is deciding, and a description of what a skill does is
// the thing that decision needs.
//
// Closing it again is half the test. The picker takes keys before any screen
// does, so a description here cannot be a screen switch — and a reading state
// that lost the answer on the way out would make ? something to be careful
// about rather than something to press.
func TestTheSquadPickerDescribesTheRowUnderItsCursor(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m := aSquadPicker(t, lang, "doc", unitSkills)
		m = key(t, m, "space")
		chosen := append([]string(nil), m.picker.Chosen...)
		if len(chosen) == 0 {
			t.Fatal("nothing was chosen, so nothing could be lost")
		}
		rows := m.picker.Visible()
		declared, err := m.lib.Skills().Lookup(rows[m.picker.Cursor].ID)
		if err != nil {
			t.Fatalf("the row under the cursor is not a skill the book holds: %v", err)
		}

		m = typeText(t, m, "?")
		if !m.picker.Reading {
			t.Fatalf("? did not open a description on the %s kit picker", lang)
		}
		body, footer := m.picker.View(m.ctx())
		if !strings.Contains(body, m.lang.GlossedSkill(declared)) {
			t.Errorf("the %s description does not name %s:\n%s", lang, declared.ID, body)
		}
		for _, line := range skillLines(m.ctx(), declared) {
			if sentence := strings.TrimSpace(line); !strings.Contains(body, sentence) {
				t.Errorf("the %s description is missing %q:\n%s", lang, sentence, body)
			}
		}
		if want := m.text(i18n.PickerReadingFooter); footer != want {
			t.Errorf("the %s footer while reading is %q, want %q", lang, footer, want)
		}

		// esc closes the description and only the description.
		m = key(t, m, "esc")
		if m.picker == nil {
			t.Fatalf("esc while reading closed the whole %s picker", lang)
		}
		if m.picker.Reading {
			t.Errorf("esc left the %s description open", lang)
		}
		if !slices.Equal(m.picker.Chosen, chosen) {
			t.Errorf("reading cost the %s picker its answer: %v, want %v",
				lang, m.picker.Chosen, chosen)
		}
		// And ? is its own way back, which is the blurb screen's contract.
		m = typeText(t, m, "?")
		m = typeText(t, m, "?")
		if m.picker.Reading {
			t.Errorf("? did not close the %s description it opened", lang)
		}
		if !slices.Equal(m.picker.Chosen, chosen) {
			t.Errorf("the %s answer is %v, want %v", lang, m.picker.Chosen, chosen)
		}
		// Enter still hands back exactly what was chosen before any of it.
		m = key(t, m, "enter")
		if !slices.Equal(m.squad.unit.Skills, chosen) {
			t.Errorf("the %s picker handed back %v, want %v", lang, m.squad.unit.Skills, chosen)
		}
	}
}

// TestWalkingWhileReadingMovesToTheNextDescription is why the cursor keys are
// live here at all: comparing two skills means reading one after the other, and
// going back to the list between them is the interaction the blurb screen
// already refused.
func TestWalkingWhileReadingMovesToTheNextDescription(t *testing.T) {
	m := aSquadPicker(t, i18n.Vi, "doc", unitSkills)
	if len(m.picker.Visible()) < 2 {
		t.Skip("the learnset holds one skill, so there is no next description")
	}
	m = typeText(t, m, "?")
	first, _ := m.picker.View(m.ctx())
	was := m.picker.Cursor

	// Scrolled first, so the offset can be shown to be dropped rather than
	// carried into an answer it means nothing about.
	m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = key(t, m, "down")
	if !m.picker.Reading {
		t.Fatal("moving the cursor closed the description")
	}
	if m.picker.Cursor == was {
		t.Fatal("the cursor did not move")
	}
	if m.picker.Scroll != 0 {
		t.Errorf("the offset into the last answer survived into this one: %d", m.picker.Scroll)
	}
	if second, _ := m.picker.View(m.ctx()); second == first {
		t.Errorf("the description did not change with the cursor:\n%s", second)
	}
	// And the list comes back on the row that was being read, not the one it
	// was opened on.
	m = key(t, m, "esc")
	if m.picker.Cursor == was {
		t.Errorf("the list came back on row %d, the one reading started from", was)
	}
}

// TestTheReadingStateIsNotCutAndCannotBeScrolledOffItsAnswer is the offset,
// asserted from both ends.
//
// ⚠️ No shipped description is long enough to need it — one row is at most three
// lines against the seventeen the smallest window leaves, which is why this
// measures the guard rather than a scroll. What it holds is that the offset is
// clamped where the answer is **read** and not where the key moves it: a picker
// scrolled to the bottom of a long answer and walked onto a short one has an
// offset past the end of it, and clamping at the other end would draw nothing.
func TestTheReadingStateIsNotCutAndCannotBeScrolledOffItsAnswer(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m := aSquadPicker(t, lang, "doc", unitSkills)
		m.width, m.height = minWidth, minHeight
		declared, err := m.lib.Skills().Lookup(m.picker.Visible()[m.picker.Cursor].ID)
		if err != nil {
			t.Fatalf("the row under the cursor is not a skill the book holds: %v", err)
		}
		m = typeText(t, m, "?")
		if drawn := m.screenContent(); strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("the %s description is cut by the frame at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		if m.picker.Scroll != 0 {
			t.Fatalf("the %s description opened already scrolled to %d", lang, m.picker.Scroll)
		}
		m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
		if m.picker.Scroll != 0 {
			t.Errorf("pgup at the top of the %s description left it at %d",
				lang, m.picker.Scroll)
		}
		// Far past the end, which every line of the answer has to survive.
		for range 40 {
			m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
		}
		body, _ := m.picker.View(m.ctx())
		for _, line := range skillLines(m.ctx(), declared) {
			if sentence := strings.TrimSpace(line); !strings.Contains(body, sentence) {
				t.Errorf("scrolling past the end of the %s description lost %q:\n%s",
					lang, sentence, body)
			}
		}
	}
}

// markerAt is where a mark sits in a rendered screen, as a line and a column,
// or (-1, -1) when it is not drawn.
//
// It reads the render rather than the state on purpose: everything below is
// about the picture, and every one of these assertions was already true of
// s.unit.Slot while the grid beside it stood still.
func markerAt(body, mark string) (int, int) {
	for index, line := range strings.Split(body, "\n") {
		if at := strings.Index(line, mark); at >= 0 {
			return index, at
		}
	}
	return -1, -1
}

// aSquadMember is a squad with one member under edit, on the field named.
func aSquadMember(t *testing.T, lang i18n.Lang, id string, field int) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, id)
	m = key(t, m, "enter")
	for m.squad.field != field {
		m = key(t, m, "down")
	}
	return m
}

// TestTheFormationFollowsTheArrowsWhileTheCellIsChosen is the defect this grid
// was drawn to be free of.
//
// The slot row stepped s.unit.Slot and the grid under it was built from
// s.editing.Units, which commit() writes and which nothing writes until the
// member is left or a picker is opened. So the picture jumped to the new cell
// only after the choosing was over, which is the one moment it says nothing.
//
// Asserted on the render and not on s.unit.Slot: the cell moving was already
// true before the fix, and a test of it would have passed throughout.
func TestTheFormationFollowsTheArrowsWhileTheCellIsChosen(t *testing.T) {
	m := aSquadMember(t, i18n.En, "live", unitSlot)
	mark := fmt.Sprintf("(%d)", m.squad.unitIndex+1)
	opened := m.screenContent()
	line, column := markerAt(opened, mark)
	if line < 0 {
		t.Fatalf("the member under edit is not marked on the grid:\n%s", opened)
	}

	was := m.squad.unit.Slot
	m = key(t, m, "right")
	if m.squad.unit.Slot == was {
		t.Fatal("the chooser did not move, so there is nothing for the grid to follow")
	}
	stepped := m.screenContent()
	movedLine, movedColumn := markerAt(stepped, mark)
	if movedLine < 0 {
		t.Fatalf("the member under edit vanished off the grid:\n%s", stepped)
	}
	if movedLine == line && movedColumn == column {
		t.Errorf("the cell went %s -> %s and the mark stayed at line %d column %d:\n%s",
			was, m.squad.unit.Slot, line, column, stepped)
	}

	// And it tracks rather than merely differing: stepping back puts the mark
	// where it started, on the cell the arrows are on and not on some other one.
	m = key(t, m, "left")
	if m.squad.unit.Slot != was {
		t.Fatalf("stepping back landed on %s rather than on %s", m.squad.unit.Slot, was)
	}
	backLine, backColumn := markerAt(m.screenContent(), mark)
	if backLine != line || backColumn != column {
		t.Errorf("back on %s the mark is at line %d column %d, want %d and %d:\n%s",
			was, backLine, backColumn, line, column, m.screenContent())
	}
}

// TestTheLiveFormationDrawsWithoutCommitting is the trap the live picture had to
// be built around, kept as a test rather than as a comment.
//
// The obvious fix is commit() on every keypress, and s.editing.Units is shared
// with every model copied off this one — so a write from inside a drawing
// reaches all of them, which is what a value receiver looks like it prevents and
// does not. So the drawing reads the unit under edit and writes nothing, and
// this is the two halves of that: a key that changes nothing leaves the guard
// down, and a key that moves the cell leaves the squad's own copy alone until
// the member is left.
func TestTheLiveFormationDrawsWithoutCommitting(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	m.squad = m.squad.open(m.squad.saved[0])
	m = key(t, m, "enter")
	if m.squad.mode != squadUnit {
		t.Fatalf("enter on the first member opened %v", m.squad.mode)
	}
	if m.squad.dirty() {
		t.Fatal("opening a member off a saved squad already claims changes")
	}

	// Walking the fields changes nothing about the unit, so nothing may reach
	// the guard.
	for range unitFieldCount + 1 {
		m = key(t, m, "down")
		_ = m.screenContent()
		if m.squad.dirty() {
			t.Fatalf("moving onto field %d raised the discard guard", m.squad.field)
		}
	}

	for m.squad.field != unitSlot {
		m = key(t, m, "down")
	}
	index := m.squad.unitIndex
	before := m.squad.editing.Units[index].Slot
	m = key(t, m, "right")
	if m.squad.unit.Slot == before {
		t.Fatal("the chooser did not move, so there is nothing to commit early")
	}
	chosen := m.squad.unit.Slot
	// Drawn more than once, because s.editing.Units is a slice shared with every
	// model copied off this one: a write into it from inside a drawing would
	// reach all of them, value receiver or not.
	for range 3 {
		_ = m.screenContent()
	}
	if now := m.squad.editing.Units[index].Slot; now != before {
		t.Errorf("the squad's own copy moved to %s while the cell was being chosen, want %s",
			now, before)
	}
	// The picture is live all the same, which is what says the two are not the
	// same reading: the mark is on the cell the arrows are on.
	mark := fmt.Sprintf("(%d)", index+1)
	if line, _ := markerAt(m.screenContent(), mark); line < 0 {
		t.Fatalf("the member under edit is not marked on the grid:\n%s", m.screenContent())
	}

	// esc is what commits, and it still does.
	m = key(t, m, "esc")
	if now := m.squad.editing.Units[index].Slot; now != chosen {
		t.Errorf("leaving the member wrote %s back, want %s", now, chosen)
	}
}

// TestTheFormationMarksTheRankThatMeetsTheEnemyFirst is the other half of
// "standing somewhere is a picture": a coordinate does not say which end of the
// grid an attack arrives at, and reach is counted in ranks from that end.
//
// The front column is derived here from hex.Ranks and hex.Place rather than
// taken from the screen's own helper, so the drawing and the reach rule are two
// readings that have to agree.
func TestTheFormationMarksTheRankThatMeetsTheEnemyFirst(t *testing.T) {
	front := frontColumn(t)
	back := -1
	for col := range hex.FormationCols {
		if col != front {
			back = col
		}
	}
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenSquads)
		m.squad = someSquad(t, m)
		s := m.squad
		if len(s.editing.Units) == 0 {
			t.Fatal("the fixture squad is empty, so nothing stands anywhere")
		}
		// One member in the front rank and one behind it, so the mark can be
		// shown to be under the first and not under the second.
		screened := s.editing.Units[0]
		screened.ID, screened.Slot = "sau", hex.Offset{Col: back, Row: 0}
		s.editing.Units = []placement.Placement{s.editing.Units[0], screened}
		s.editing.Units[0].Slot = hex.Offset{Col: front, Row: 0}
		m.squad = s.editUnit(0)
		body := m.screenContent()

		caret := strings.Repeat("^", formationCell)
		caretLine, caretColumn := markerAt(body, caret)
		if caretLine < 0 {
			t.Fatalf("the %s grid marks no front rank:\n%s", lang, body)
		}
		if words := m.text(i18n.SquadFormationFront); !strings.Contains(
			strings.Split(body, "\n")[caretLine], words) {
			t.Errorf("the %s mark does not say %q:\n%s", lang, words, body)
		}
		if _, at := markerAt(body, "(1)"); at != caretColumn {
			t.Errorf("the %s front-rank member is drawn at column %d and the mark at %d:\n%s",
				lang, at, caretColumn, body)
		}
		if _, at := markerAt(body, "[2]"); at == caretColumn {
			t.Errorf("the %s member behind the front rank is drawn under the mark:\n%s",
				lang, body)
		}
	}
}

// TestTheSlotRowSaysWhichRankItStandsIn is the row the grid is beside: it keeps
// the coordinate, because that is what squads.json holds and what an author
// matches a file against, and puts the rank next to it, because that is the half
// a coordinate cannot say.
func TestTheSlotRowSaysWhichRankItStandsIn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m := aSquadMember(t, lang, "rank", unitSlot)
		body := m.screenContent()
		if !strings.Contains(body, m.squad.unit.Slot.String()) {
			t.Errorf("the %s slot row lost the cell %s:\n%s", lang, m.squad.unit.Slot, body)
		}
		want := m.rankLabel(m.squad.unit.Slot)
		if want == "" {
			t.Fatalf("the member stands at %s, which is on no rank", m.squad.unit.Slot)
		}
		if !strings.Contains(body, want) {
			t.Errorf("the %s slot row does not say %q:\n%s", lang, want, body)
		}
		// And the reading follows the arrows too: walking off the front column
		// has to stop saying front rank.
		for range hex.FormationRows {
			m = key(t, m, "right")
		}
		if m.rankLabel(m.squad.unit.Slot) == want {
			t.Fatalf("the chooser is still in the %s after a whole column", want)
		}
		if body := m.screenContent(); !strings.Contains(body, m.rankLabel(m.squad.unit.Slot)) {
			t.Errorf("the %s slot row still reads %q at %s:\n%s",
				lang, want, m.squad.unit.Slot, body)
		}
	}
}

// TestARankIsTheSameDepthOnEitherSide is what lets rankOf ask the question of
// one side and answer it for both, which a squad needs because a squad carries
// no side — placement.Squad.Take fields the same cells as either half.
//
// It holds because hex.Place rotates an enemy formation 180 degrees rather than
// translating it: the depth is preserved and only the column number is not.
func TestARankIsTheSameDepthOnEitherSide(t *testing.T) {
	depthOn := func(side hex.Side, slot hex.Offset) int {
		placed := hex.Place(side, slot)
		for depth, rank := range hex.Ranks(side) {
			for _, cell := range rank {
				if cell == placed {
					return depth
				}
			}
		}
		return -1
	}
	for col := range hex.FormationCols {
		for row := range hex.FormationRows {
			slot := hex.Offset{Col: col, Row: row}
			want := depthOn(hex.SideAlly, slot)
			if want < 0 {
				t.Fatalf("%s is on no ally rank", slot)
			}
			if got := depthOn(hex.SideEnemy, slot); got != want {
				t.Errorf("%s is rank %d as an ally and rank %d as an enemy", slot, want, got)
			}
			if got := rankOf(slot); got != want {
				t.Errorf("the builder reads %s as rank %d, want %d", slot, got, want)
			}
		}
	}
}

// frontColumn is the authoring column hex.Ranks calls depth 0.
func frontColumn(t *testing.T) int {
	t.Helper()
	for col := range hex.FormationCols {
		placed := hex.Place(hex.SideAlly, hex.Offset{Col: col})
		for _, cell := range hex.Ranks(hex.SideAlly)[0] {
			if cell == placed {
				return col
			}
		}
	}
	t.Fatal("no authoring column maps onto the rank that meets the enemy first")
	return -1
}

// aSavedSquad is a squad written to the file and taken back up for editing,
// which is the state the discard guard is about: there has to be something
// written down for what is in hand to differ from.
//
// Its member is built around whichever character in the book learns a trait, so
// the trait list has a row to toggle. Naming one would tie this to content the
// author is free to change, which is what the injected fixture exists to avoid.
func aSavedSquad(t *testing.T) model {
	t.Helper()
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	s := m.squad.begin()
	s.editing.ID, s.editing.Name = "do-luu", "đội lưu"
	s.idInput.SetValue(s.editing.ID)
	s.nameInput.SetValue(s.editing.Name)

	character, learns := aCharacterWithATrait(s.characters)
	if !learns {
		t.Fatal("no character in the book learns a trait, so no member can bring one")
	}
	unit := placement.Placement{
		ID:        "mot",
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: 1},
	}
	kit := character.SkillsAt(unit.Level, progression.Furthest)
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}
	unit.Skills = kit
	unit.Passives = character.PassivesAt(unit.Level, progression.Furthest)[:cast.TraitSlots]
	s.editing.Units = []placement.Placement{unit}

	m.squad = s
	m = withASquadSaved(t, m)
	m.squad = m.squad.open(m.squad.saved[0])
	if m.squad.mode != squadEdit {
		t.Fatalf("the fixture squad opened in %v", m.squad.mode)
	}
	return m
}

func aCharacterWithATrait(characters []cast.Character) (cast.Character, bool) {
	for _, character := range characters {
		if len(character.PassivesAt(progression.LevelCap, progression.Furthest)) > 0 {
			return character, true
		}
	}
	return cast.Character{}, false
}

// TestARoundTripThroughAMemberLeavesTheGuardDown is what the guard being a
// comparison rather than a latch buys, and it is the defect PR #153 recorded and
// deliberately left standing.
//
// commit() writes a member back on the way out of it whether or not a key moved
// anything, so under a flag set from there, *opening* a member and pressing
// escape claimed a change — and arrowing the cell chooser onto another cell and
// back claimed one twice over. Both are round trips: what they put back is what
// they took, so the squad on the file and the squad in hand are the same squad
// and nobody may be asked about discarding it.
func TestARoundTripThroughAMemberLeavesTheGuardDown(t *testing.T) {
	m := aSavedSquad(t)
	if m.squad.dirty() {
		t.Fatal("a squad just read off the file already differs from it")
	}

	// One: open a member and leave it.
	m = key(t, m, "enter")
	if m.squad.mode != squadUnit {
		t.Fatalf("enter on the first member opened %v", m.squad.mode)
	}
	m = key(t, m, "esc")
	if m.squad.dirty() {
		t.Error("opening a member and pressing escape changed the squad")
	}

	// Two: arrow the cell onto another and back.
	m = key(t, m, "enter")
	for m.squad.field != unitSlot {
		m = key(t, m, "down")
	}
	was := m.squad.unit.Slot
	m = key(t, m, "right")
	if m.squad.unit.Slot == was {
		t.Fatal("the chooser did not move, so there is no round trip to make")
	}
	m = key(t, m, "left")
	if m.squad.unit.Slot != was {
		t.Fatalf("stepping back landed on %s rather than on %s", m.squad.unit.Slot, was)
	}
	m = key(t, m, "esc")
	if m.squad.dirty() {
		t.Error("arrowing the cell onto another and back changed the squad")
	}

	// And the whole point of it: leaving asks nothing.
	m = key(t, m, "esc")
	if m.guard != nil {
		t.Errorf("leaving raised %v over changes nobody made", m.guard.question)
	}
	if m.squad.mode != squadList {
		t.Errorf("escape from an unchanged squad landed in %v", m.squad.mode)
	}
}

// TestEveryRealEditRaisesTheGuard is the other side of it, and it is a table
// rather than one case because catching every kind of edit was the latch's one
// virtue: a comparison that missed a field would lose that edit in silence, with
// no question asked and nothing on screen looking wrong.
//
// Every case leaves the model on the squad view, so the escape below is the one
// the guard hangs off. The member cases go in and come back out, because that is
// the route by which a member's own fields reach the squad.
func TestEveryRealEditRaisesTheGuard(t *testing.T) {
	intoTheMember := func(t *testing.T, m model, field int) model {
		t.Helper()
		m = key(t, m, "enter")
		if m.squad.mode != squadUnit {
			t.Fatalf("enter on the first member opened %v", m.squad.mode)
		}
		for m.squad.field != field {
			m = key(t, m, "down")
		}
		return m
	}
	edits := []struct {
		what string
		make func(*testing.T, model) model
	}{
		{"the name", func(t *testing.T, m model) model {
			return typeText(t, m, "x")
		}},
		{"another member", func(t *testing.T, m model) model {
			m = key(t, m, "down")
			m = key(t, m, "enter")
			if len(m.squad.editing.Units) != 2 {
				t.Fatalf("the squad holds %d members", len(m.squad.editing.Units))
			}
			return key(t, m, "esc")
		}},
		{"a member taken out", func(t *testing.T, m model) model {
			m = key(t, m, "ctrl+x")
			if len(m.squad.editing.Units) != 0 {
				t.Fatalf("the squad still holds %d members", len(m.squad.editing.Units))
			}
			return m
		}},
		{"the character", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitCharacter)
			was := m.squad.unit.Character
			m = key(t, m, "right")
			if m.squad.unit.Character == was {
				t.Fatalf("the cast holds only %q, so it cannot be cycled", was)
			}
			return key(t, m, "esc")
		}},
		{"the level", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitLevel)
			was := m.squad.unit.Level
			m = key(t, m, "backspace")
			if m.squad.unit.Level == was {
				t.Fatalf("the level field still reads %d", was)
			}
			return key(t, m, "esc")
		}},
		{"the form", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitStage)
			was := m.squad.unit.Stage
			m = key(t, m, "right")
			if m.squad.unit.Stage == was {
				t.Fatalf("the form chooser stayed on %q", was)
			}
			return key(t, m, "esc")
		}},
		{"the cell", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitSlot)
			was := m.squad.unit.Slot
			m = key(t, m, "right")
			if m.squad.unit.Slot == was {
				t.Fatalf("the cell chooser stayed on %s", was)
			}
			return key(t, m, "esc")
		}},
		{"the kit", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitSkills)
			return throughTheList(t, m)
		}},
		{"the trait", func(t *testing.T, m model) model {
			m = intoTheMember(t, m, unitPassives)
			return throughTheList(t, m)
		}},
	}
	for _, edit := range edits {
		t.Run(edit.what, func(t *testing.T) {
			m := edit.make(t, aSavedSquad(t))
			if m.squad.mode != squadEdit {
				t.Fatalf("the edit left the screen in %v", m.squad.mode)
			}
			if !m.squad.dirty() {
				t.Fatalf("changing %s left the squad reading as the one on the file", edit.what)
			}
			m = key(t, m, "esc")
			if m.guard == nil {
				t.Fatalf("leaving after changing %s discarded it without asking", edit.what)
			}
			if m.guard.question != i18n.SquadDiscard {
				t.Errorf("the question asked was %v", m.guard.question)
			}
		})
	}

	// The id is the one field a saved squad does not offer — changing it would
	// write a second squad rather than rename this one — so it is asked of a
	// squad nobody has written yet, where typing one is the whole of the edit.
	t.Run("the id", func(t *testing.T) {
		m, _, _ := start(t, i18n.En)
		m = menuTo(t, m, screenSquads)
		m = typeText(t, m, "n")
		if m.squad.dirty() {
			t.Fatal("a squad nobody has typed into already claims changes")
		}
		m = typeText(t, m, "moi")
		if !m.squad.dirty() {
			t.Fatal("typing an id left the squad reading as an empty one")
		}
		if m = key(t, m, "esc"); m.guard == nil {
			t.Fatal("leaving after typing an id discarded it without asking")
		}
	})
}

// throughTheList opens the list under the cursor, toggles the row under its own,
// takes the answer and comes back out to the squad.
func throughTheList(t *testing.T, m model) model {
	t.Helper()
	m = key(t, m, "enter")
	if m.picker == nil {
		t.Fatal("the field raised no picker")
	}
	if len(m.picker.Options) == 0 {
		t.Fatal("the list is empty, so there is no row to toggle")
	}
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if m.picker != nil {
		t.Fatal("the list is still open")
	}
	return key(t, m, "esc")
}

// TestSavingSettlesTheGuardAndReopeningStartsClean is the third state a
// comparison has to get right: a write moves the thing being compared against,
// so a squad just saved is a squad with nothing outstanding — and one taken back
// up off the file starts from the file rather than from whatever the screen was
// last holding.
func TestSavingSettlesTheGuardAndReopeningStartsClean(t *testing.T) {
	m := aSavedSquad(t)
	m = typeText(t, m, "x")
	if !m.squad.dirty() {
		t.Fatal("typing into the name left the guard down")
	}
	m = key(t, m, "ctrl+s")
	if m.squad.err != nil {
		t.Fatalf("the save was refused: %v", m.squad.err)
	}
	if m.squad.dirty() {
		t.Error("a squad just written still reads as changed")
	}
	m = key(t, m, "esc")
	if m.guard != nil {
		t.Error("leaving a squad just saved asked before discarding it")
	}

	m = key(t, m, "enter")
	if m.squad.mode != squadEdit {
		t.Fatalf("enter on the catalogue opened %v", m.squad.mode)
	}
	if m.squad.dirty() {
		t.Error("a squad taken back up off the file already differs from it")
	}
}

// TestTheBuilderOffersAShownCharacterAndNotAHeldBackOne is the rule on its own,
// over a cast written here.
//
// Authored rather than read off the shipped book, and that is the point of the
// fixture: a test that passed because the shipped data happens to hide exactly
// one character is a test that breaks the day the data changes, and what it
// would be measuring is cast.json rather than the rule. The one place the
// shipped data is the subject is the golden report.
func TestTheBuilderOffersAShownCharacterAndNotAHeldBackOne(t *testing.T) {
	// The held-back one is declared FIRST on purpose. It is what makes the
	// "not offered" case discriminating at both call sites at once: a new
	// member takes the first character offered, so a filter that did nothing
	// would start every new member on the character an author has taken out of
	// the choice, and that is a different mistake from cycling onto one.
	all := []cast.Character{
		{ID: "book.held", Hidden: true},
		{ID: "book.shown"},
		{ID: "book.other"},
	}

	if got, want := offeredIDs(offeredCharacters(all, "")), []string{"book.shown", "book.other"}; !slices.Equal(got, want) {
		t.Errorf("with nobody held the builder offers %v, want %v", got, want)
	}

	// A member opened on the held-back one is offered it, **in the place it was
	// declared** rather than appended at the end: the chooser walks this list
	// with the arrow keys, so moving a row would mean the key that steps away
	// from a character no longer steps back onto it.
	if got, want := offeredIDs(offeredCharacters(all, "book.held")), []string{"book.held", "book.shown", "book.other"}; !slices.Equal(got, want) {
		t.Errorf("a member opened on the held-back one is offered %v, want %v", got, want)
	}

	// And a member opened on a shown character brings nobody else back — the
	// exemption is for the one character named and for no other.
	if got, want := offeredIDs(offeredCharacters(all, "book.shown")), []string{"book.shown", "book.other"}; !slices.Equal(got, want) {
		t.Errorf("a member opened on a shown character is offered %v, want %v", got, want)
	}
}

func offeredIDs(characters []cast.Character) []string {
	out := make([]string, 0, len(characters))
	for _, character := range characters {
		out = append(out, character.ID)
	}
	return out
}

// TestANewMemberNeverStartsOnAHeldBackCharacter drives the other call site
// through the key an author presses, on a cast whose first character is held
// back.
//
// The cast is written onto the screen rather than into the library because the
// order is the whole fixture: a book grown through SaveCharacter appends, so the
// character this needs at the front of the list could only get there by being
// the one the fixture already ships — which is the coupling the fixture exists
// to avoid.
func TestANewMemberNeverStartsOnAHeldBackCharacter(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	shipped := m.squad.characters
	if len(shipped) == 0 {
		t.Fatal("the fixture cast is empty, so no member can be added at all")
	}
	held := shipped[0]
	held.ID, held.Hidden = "fixture-anime.recluse", true
	shown := shipped[0]
	m.squad.characters = append([]cast.Character{held}, shown)

	m = typeText(t, m, "n")
	m = typeText(t, m, "moi")
	m = key(t, m, "enter")
	if m.squad.mode != squadUnit {
		t.Fatalf("adding a member did not open it, mode is %v", m.squad.mode)
	}
	if m.squad.unit.Character == held.ID {
		t.Errorf("a new member started on %q, which the cast holds back", held.ID)
	}
	if m.squad.unit.Character != shown.ID {
		t.Errorf("a new member started on %q, want the first character still offered, %q",
			m.squad.unit.Character, shown.ID)
	}
}

// TestASquadAlreadyNamingAHeldBackCharacterKeepsIt is the case that would have
// shipped broken, driven through the keys an author would press.
//
// Hiding a character is an authoring convenience the user expects to flip back,
// so it may take a character out of the choices offered and may never reach into
// a squad already on the file and change one. A chooser that simply dropped the
// hidden rows does both: the member's character resolves to nobody, so its forms
// and its learnset go empty and the kit picker will not open at all, and the
// arrow keys write somebody else into a member nobody asked to change — in the
// author's own saved file, from the one screen here that writes it.
func TestASquadAlreadyNamingAHeldBackCharacterKeepsIt(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, lib, _ := start(t, lang)
		held := saveAHeldBackCharacter(t, lib, "fixture-film.recluse", "Recluse")
		// A second one nobody names, so the walk below has something to refuse
		// that is not the exemption. Authored rather than left to whichever
		// character cast.json happens to hold back: leaning on the shipped data
		// would make this test fail the day somebody is un-hidden, which is a
		// data decision and no business of the builder's.
		unnamed := saveAHeldBackCharacter(t, lib, "fixture-film.hermit", "Hermit")
		saveASquadNaming(t, lib, "do-an", held)

		// It still loads, which is half the claim: a squad naming a held-back
		// character is as valid as any other and the catalogue lists it.
		m = menuTo(t, m, screenSquads)
		if len(m.squad.saved) != 1 {
			t.Fatalf("%v: the catalogue holds %d squads, want the one just written", lang, len(m.squad.saved))
		}
		m = key(t, m, "enter")
		if m.squad.mode != squadEdit {
			t.Fatalf("%v: enter on the catalogue opened %v", lang, m.squad.mode)
		}
		m = key(t, m, "enter")
		if m.squad.mode != squadUnit {
			t.Fatalf("%v: enter on the member opened %v", lang, m.squad.mode)
		}
		if m.squad.field != unitCharacter {
			t.Fatalf("%v: the member opened on field %d, want the character row", lang, m.squad.field)
		}

		// The character resolves, so everything read against it is still there.
		// This is what a filtered lookup slice takes away, and it takes it away
		// silently — the row still prints the id.
		character, known := m.squad.character()
		if !known {
			t.Fatalf("%v: the member's own character is not in the cast the screen holds", lang)
		}
		if !character.Hidden {
			t.Fatalf("%v: the fixture character is not held back, so this measures nothing", lang)
		}
		if len(m.squad.stageChoices()) == 0 {
			t.Errorf("%v: the member offers no form at all", lang)
		}
		if len(character.SkillsAt(m.squad.unit.Level, m.squad.form())) == 0 {
			t.Errorf("%v: the member knows nothing, so its kit picker would open empty", lang)
		}

		// The screen says why a character nothing else offers is on the list.
		if body := m.screenContent(); !strings.Contains(body, m.text(i18n.SquadHeldBack)) {
			t.Errorf("%v: the member holds a held-back character and the screen does not say so:\n%s", lang, body)
		}

		// Leaving the member without touching anything leaves the squad the squad
		// that was written down, which is what "not edited behind the author's
		// back" comes to on the file. Asked before a key is pressed at the
		// chooser, because changing the character empties the kit by design and
		// a squad edited on purpose is meant to read as changed.
		m = key(t, m, "esc")
		if m.squad.mode != squadEdit {
			t.Fatalf("%v: esc out of the member landed in %v", lang, m.squad.mode)
		}
		if m.squad.dirty() {
			t.Errorf("%v: opening a member holding a held-back character changed the squad", lang)
		}
		if got := m.squad.editing.Units[0].Character; got != held.ID {
			t.Errorf("%v: the member now names %q, want the %q it was saved with", lang, got, held.ID)
		}

		// ⚠️ The round trip is the assertion, not the single press. Stepping the
		// chooser is an author asking for a change, so one press moving the
		// character is correct either way — what a dropped row costs is the way
		// BACK, because a character off the list is one the arrows can never
		// return to.
		m = key(t, m, "enter")
		if m.squad.mode != squadUnit || m.squad.field != unitCharacter {
			t.Fatalf("%v: reopening the member landed in %v on field %d", lang, m.squad.mode, m.squad.field)
		}
		was := m.squad.unit.Character
		m = key(t, m, "right")
		if m.squad.unit.Character == was {
			t.Fatalf("%v: the chooser did not move at all, so the round trip below measures nothing", lang)
		}
		// Stepped onto a character that is not held back, so the note goes: it
		// reads the answer in hand rather than the exemption behind the list.
		if body := m.screenContent(); strings.Contains(body, m.text(i18n.SquadHeldBack)) {
			t.Errorf("%v: the member now names %q, which is offered like any other, and the screen still calls it held back",
				lang, m.squad.unit.Character)
		}
		m = key(t, m, "left")
		if m.squad.unit.Character != was {
			t.Errorf("%v: stepping the character chooser away and back landed on %q, want %q — the held-back character is off the list",
				lang, m.squad.unit.Character, was)
		}

		// Walked the whole way round, the chooser offers the character it was
		// opened on and no OTHER held-back one. The round trip above cannot say
		// that — it presses one key and comes back — so a chooser that had
		// dropped the filter entirely would satisfy it while offering every
		// character the cast holds back.
		//
		// Anti-vacuity first: the character the walk has to refuse is one this
		// test wrote, so the claim does not rest on the shipped cast holding
		// anybody back.
		var otherHeldBack []string
		for _, candidate := range m.squad.characters {
			if candidate.Hidden && candidate.ID != was {
				otherHeldBack = append(otherHeldBack, candidate.ID)
			}
		}
		if !slices.Contains(otherHeldBack, unnamed.ID) {
			t.Fatalf("%v: the second held-back character %q is not in the cast the screen holds; it holds back %v",
				lang, unnamed.ID, otherHeldBack)
		}
		visited := map[string]bool{}
		for range len(m.squad.characters) + 1 {
			m = key(t, m, "right")
			visited[m.squad.unit.Character] = true
			if m.squad.unit.Character == was {
				break
			}
		}
		if m.squad.unit.Character != was {
			t.Fatalf("%v: walking the chooser right round ended on %q rather than back at %q",
				lang, m.squad.unit.Character, was)
		}
		for _, id := range otherHeldBack {
			if visited[id] {
				t.Errorf("%v: the chooser offered %q, which the cast holds back and no member had chosen", lang, id)
			}
		}

		// And the member the round trip put back is the member that goes back
		// into the squad, rather than only into the copy under edit.
		m = key(t, m, "esc")
		if got := m.squad.editing.Units[0].Character; got != held.ID {
			t.Errorf("%v: the member now names %q, want the %q it was saved with", lang, got, held.ID)
		}
	}
}

// saveAHeldBackCharacter writes a character into the library and holds it back,
// which is the state a squad builder test needs and no Draft field can ask for:
// `hidden` is hand-written into cast.json rather than filled in by a form.
func saveAHeldBackCharacter(t *testing.T, lib *forge.Library, id, name string) cast.Character {
	t.Helper()
	character, err := forge.Draft{
		ID: id, Name: name, Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/adept.svg", Element: "wind/ground",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve the held-back character: %v", err)
	}
	character.Hidden = true
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save the held-back character: %v", err)
	}
	return character
}

// saveASquadNaming writes a one-member squad around a character, through the
// library rather than onto the screen: the claim being set up is that a squad
// **on the file** naming a held-back character is valid, and a squad built in
// memory would not have been asked.
func saveASquadNaming(t *testing.T, lib *forge.Library, id string, character cast.Character) {
	t.Helper()
	unit := placement.Placement{
		ID:        "mot",
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: 1},
	}
	kit := character.SkillsAt(unit.Level, progression.Furthest)
	if len(kit) == 0 {
		t.Fatal("the fixture character knows nothing, so no squad can field it")
	}
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}
	unit.Skills = kit
	if err := lib.SaveSquad(placement.Squad{
		ID: id, Name: id, Units: []placement.Placement{unit},
	}); err != nil {
		t.Fatalf("a squad naming a held-back character was refused: %v", err)
	}
}
