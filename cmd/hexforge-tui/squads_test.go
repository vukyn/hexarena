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
		if m.picker == nil || len(m.picker.options) == 0 {
			t.Fatal("the trait field raised no picker with rows in it")
		}
		if m.picker.kind != pickPassives {
			t.Errorf("the trait picker is a %v, so its rows are read out of the wrong book",
				m.picker.kind)
		}
		body, _ := m.picker.view(m)
		for _, option := range m.picker.options {
			// A trait sharing a name with a skill would prove nothing either
			// way, so only the ids the skill book really refuses are asked
			// about — and what they must not put on screen is its refusal.
			_, err := m.lib.Skills().Lookup(option.id)
			if err == nil {
				continue
			}
			if refusal := m.lang.Error(err); strings.Contains(body, refusal) {
				t.Errorf("the %s trait picker draws %q where %s's detail belongs:\n%s",
					lang, refusal, option.id, body)
			}
		}
		// And it draws the trait's own name, which is what the listing puts
		// beside an id -- nothing in English, where a data name is not
		// translated and the id is the whole row.
		rows := m.picker.visible()
		held, err := m.lib.Passives().Lookup(rows[m.picker.cursor].id)
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
		if !m.picker.reading {
			t.Fatalf("? opened no description on the %s trait picker", lang)
		}
		body, _ = m.picker.view(m)
		if !strings.Contains(body, m.lang.GlossedPassive(held)) {
			t.Errorf("the %s description does not name %s:\n%s", lang, held.ID, body)
		}
		for _, line := range traitSentences(m, held) {
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
		chosen := append([]string(nil), m.picker.chosen...)
		if len(chosen) == 0 {
			t.Fatal("nothing was chosen, so nothing could be lost")
		}
		rows := m.picker.visible()
		declared, err := m.lib.Skills().Lookup(rows[m.picker.cursor].id)
		if err != nil {
			t.Fatalf("the row under the cursor is not a skill the book holds: %v", err)
		}

		m = typeText(t, m, "?")
		if !m.picker.reading {
			t.Fatalf("? did not open a description on the %s kit picker", lang)
		}
		body, footer := m.picker.view(m)
		if !strings.Contains(body, m.lang.GlossedSkill(declared)) {
			t.Errorf("the %s description does not name %s:\n%s", lang, declared.ID, body)
		}
		for _, line := range skillLines(m, declared) {
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
		if m.picker.reading {
			t.Errorf("esc left the %s description open", lang)
		}
		if !slices.Equal(m.picker.chosen, chosen) {
			t.Errorf("reading cost the %s picker its answer: %v, want %v",
				lang, m.picker.chosen, chosen)
		}
		// And ? is its own way back, which is the blurb screen's contract.
		m = typeText(t, m, "?")
		m = typeText(t, m, "?")
		if m.picker.reading {
			t.Errorf("? did not close the %s description it opened", lang)
		}
		if !slices.Equal(m.picker.chosen, chosen) {
			t.Errorf("the %s answer is %v, want %v", lang, m.picker.chosen, chosen)
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
	if len(m.picker.visible()) < 2 {
		t.Skip("the learnset holds one skill, so there is no next description")
	}
	m = typeText(t, m, "?")
	first, _ := m.picker.view(m)
	was := m.picker.cursor

	// Scrolled first, so the offset can be shown to be dropped rather than
	// carried into an answer it means nothing about.
	m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = key(t, m, "down")
	if !m.picker.reading {
		t.Fatal("moving the cursor closed the description")
	}
	if m.picker.cursor == was {
		t.Fatal("the cursor did not move")
	}
	if m.picker.scroll != 0 {
		t.Errorf("the offset into the last answer survived into this one: %d", m.picker.scroll)
	}
	if second, _ := m.picker.view(m); second == first {
		t.Errorf("the description did not change with the cursor:\n%s", second)
	}
	// And the list comes back on the row that was being read, not the one it
	// was opened on.
	m = key(t, m, "esc")
	if m.picker.cursor == was {
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
		declared, err := m.lib.Skills().Lookup(m.picker.visible()[m.picker.cursor].id)
		if err != nil {
			t.Fatalf("the row under the cursor is not a skill the book holds: %v", err)
		}
		m = typeText(t, m, "?")
		if drawn := m.screenContent(); strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("the %s description is cut by the frame at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		if m.picker.scroll != 0 {
			t.Fatalf("the %s description opened already scrolled to %d", lang, m.picker.scroll)
		}
		m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
		if m.picker.scroll != 0 {
			t.Errorf("pgup at the top of the %s description left it at %d",
				lang, m.picker.scroll)
		}
		// Far past the end, which every line of the answer has to survive.
		for range 40 {
			m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
		}
		body, _ := m.picker.view(m)
		for _, line := range skillLines(m, declared) {
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
// The obvious fix is commit() on every keypress, and commit() also sets unsaved
// — the flag the whole edit loop hangs off, since leaving a squad that carries
// it asks before discarding. A cursor that only passed over a field would then
// leave a squad claiming changes nobody made. So the drawing reads the unit
// under edit and writes nothing, and this is the two halves of that: a key that
// changes nothing leaves the guard down, and a key that moves the cell leaves
// the squad's own copy alone until the member is left.
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
	if m.squad.unsaved {
		t.Fatal("opening a member off a saved squad already claims changes")
	}

	// Walking the fields changes nothing about the unit, so nothing may reach
	// the guard. This is what a commit() at the top of updateUnit would fail.
	for range unitFieldCount + 1 {
		m = key(t, m, "down")
		_ = m.screenContent()
		if m.squad.unsaved {
			t.Fatalf("moving onto field %d raised the unsaved guard", m.squad.field)
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
