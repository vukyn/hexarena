package screen

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// What the squad builder draws, and how its cursor, its three modes and its
// six fields behave.
//
// ⚠️ **Nothing here writes a file, and that is a rule rather than an accident.**
// `start` copies the data into a temp directory, so a save *could* be driven —
// but the one golden in this package loads the books straight out of
// ../seed/data with no temp copy anywhere near them, and a suite where some
// squad tests write and others do not is one edit away from a golden that
// authored a squad into the repository. Everything below therefore builds its
// squads as values: `Saved` is a field, `Open` sets the baseline off whatever it
// is handed, and neither needs `SaveSquad` to have run. The tests that genuinely
// need a directory — the whole end-to-end write, the save settling the guard,
// and a squad on the file naming a held-back character — stayed in
// cmd/hexforge-tui, which is also where the delete's own two confirms are driven
// through the real `y`.

// pressSquads sends one named key and hands back the screen and what it asked
// the client for.
func pressSquads(t *testing.T, c Context, s SquadsScreen, name string) (SquadsScreen, Action) {
	t.Helper()
	next, action, _ := s.Update(c, press(t, name))
	return next, action
}

// onSquads is pressSquads for a test that is not asking about the action.
func onSquads(t *testing.T, c Context, s SquadsScreen, name string) SquadsScreen {
	t.Helper()
	next, _ := pressSquads(t, c, s, name)
	return next
}

// typeIntoSquads sends one rune per message, which is what a keyboard does.
func typeIntoSquads(t *testing.T, c Context, s SquadsScreen, text string) SquadsScreen {
	t.Helper()
	for _, letter := range text {
		s = onSquads(t, c, s, string(letter))
	}
	return s
}

// aNewSquad is a squad started from the catalogue with an id typed into it, and
// its first member added and opened — which is the state most of the tests below
// are about.
func aNewSquad(t *testing.T, c Context, id string) SquadsScreen {
	t.Helper()
	s := onSquads(t, c, NewSquadsScreen(c), "n")
	if s.Mode != SquadEdit {
		t.Fatalf("n did not open a squad to build, mode is %v", s.Mode)
	}
	s = typeIntoSquads(t, c, s, id)
	s = onSquads(t, c, s, "enter")
	if s.Mode != SquadUnit {
		t.Fatalf("adding a member did not open it, mode is %v", s.Mode)
	}
	return s
}

// onField walks the member's cursor to the field named.
func onField(t *testing.T, c Context, s SquadsScreen, field int) SquadsScreen {
	t.Helper()
	for range SquadUnitFieldCount + 1 {
		if s.Field == field {
			return s
		}
		s = onSquads(t, c, s, "down")
	}
	t.Fatalf("the cursor never reached field %d", field)
	return s
}

// TestTwoMembersCannotStandOnOneCell is the chooser stepping over what is taken.
//
// A cell holding two units is not a formation, and a chooser that stops on one
// is a chooser that can be left in a state the save has to reject — so the
// arrow keys skip it rather than the write refusing it later.
func TestTwoMembersCannotStandOnOneCell(t *testing.T) {
	c, _ := start(t, i18n.En)
	s := aNewSquad(t, c, "pair")
	s = onSquads(t, c, s, "esc")
	// Down off the member just added, onto the row that adds another.
	s = onSquads(t, c, s, "down")
	s = onSquads(t, c, s, "enter")
	if len(s.Editing.Units) != 2 {
		t.Fatalf("the squad holds %d members", len(s.Editing.Units))
	}
	first := s.Editing.Units[0].Slot
	if s.Editing.Units[1].Slot == first {
		t.Fatalf("both members arrived at %s", first)
	}
	// Walking the chooser all the way round never lands on the taken cell.
	s = onField(t, c, s, SquadUnitSlot)
	for range hex.FormationCols * hex.FormationRows {
		s = onSquads(t, c, s, "right")
		if s.Unit.Slot == first {
			t.Fatalf("the chooser landed on %s, where the other member stands", first)
		}
	}
}

// TestASquadMemberIsReadAgainstItsOwnCharacter is the field that everything
// under it depends on: changing who the unit is empties a kit that was chosen
// from somebody else's learnset.
func TestASquadMemberIsReadAgainstItsOwnCharacter(t *testing.T) {
	c, _ := start(t, i18n.En)
	s := aNewSquad(t, c, "swap")
	s = onField(t, c, s, SquadUnitSkills)
	s, action := pressSquads(t, c, s, "enter")
	if action.Kind != Pick || action.Picker == nil {
		t.Fatalf("the skills field asked for %v rather than a picker", action.Kind)
	}
	picker := action.Picker.Raise()
	picker, result := picker.Update(c, press(t, "space"))
	_, result = picker.Update(c, press(t, "enter"))
	if !result.Answered {
		t.Fatal("the picker did not hand an answer back")
	}
	s, _ = s.Picked(c, result.Into, result.Answer)
	if len(s.Unit.Skills) == 0 {
		t.Fatal("nothing was chosen to be lost")
	}

	was := s.Unit.Character
	s = onField(t, c, s, SquadUnitCharacter)
	s = onSquads(t, c, s, "right")
	if s.Unit.Character == was {
		t.Skip("the fixture cast has one character, so there is nothing to change to")
	}
	if len(s.Unit.Skills) != 0 {
		t.Errorf("the kit survived the character changing: %v", s.Unit.Skills)
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
	c, _ := start(t, i18n.En)
	s := aNewSquad(t, c, "over")
	character, known := s.Character()
	if !known {
		t.Fatal("the member names no character in the book")
	}
	available := character.SkillsAt(s.Unit.Level, s.Form())
	if len(available) == 0 {
		t.Fatal("the member knows nothing at its own level")
	}
	// One more than the slots hold, taken from what it does know so the refusal
	// is about the count rather than about a name.
	overfull := make([]string, 0, cast.SkillSlots+1)
	for len(overfull) <= cast.SkillSlots {
		overfull = append(overfull, available[len(overfull)%len(available)])
	}
	err := s.Refuse(cast.SkillSlots, overfull, "skill", available, cast.Required)
	if err == nil {
		t.Fatalf("%d skills into %d slots was accepted", len(overfull), cast.SkillSlots)
	}
	// Whichever way it is wrong first — too many, or the same one twice — the
	// words are the rule's own and not the screen's.
	if !strings.Contains(err.Error(), "slot(s)") && !strings.Contains(err.Error(), "twice") {
		t.Errorf("the refusal is %q, want the loadout rule's own words", err)
	}
}

// TestTheSquadTraitPickerReadsTheTraitBook is the bug the reading state
// uncovered, kept as a regression.
//
// The builder raised its trait picker as PickSkills while handing it trait ids,
// so every row looked itself up in the skill book, missed, and drew "unknown
// skill" in red where its detail belongs. Nothing caught it because the fixture
// cast learns no traits: every test that opened that picker opened an empty one.
func TestTheSquadTraitPickerReadsTheTraitBook(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		s, holds := aSquadOfATraitHolder(t, c)
		if !holds {
			t.Skip("no character in the book learns a trait, so there is no row to draw")
		}
		raised := s.OpenPassives()
		if raised == nil || len(raised.Options) == 0 {
			t.Fatal("the trait field raised no picker with rows in it")
		}
		picker := raised.Raise()
		if picker.Kind != PickPassives {
			t.Errorf("the trait picker is a %v, so its rows are read out of the wrong book",
				picker.Kind)
		}
		body, _ := picker.View(c)
		for _, option := range picker.Options {
			// A trait sharing a name with a skill would prove nothing either
			// way, so only the ids the skill book really refuses are asked
			// about — and what they must not put on screen is its refusal.
			_, err := c.Lib.Skills().Lookup(option.ID)
			if err == nil {
				continue
			}
			if refusal := c.Lang.Error(err); strings.Contains(body, refusal) {
				t.Errorf("the %s trait picker draws %q where %s's detail belongs:\n%s",
					lang, refusal, option.ID, body)
			}
		}
		// And it draws the trait's own name, which is what the listing puts
		// beside an id -- nothing in English, where a data name is not
		// translated and the id is the whole row.
		rows := picker.Visible()
		held, err := c.Lib.Passives().Lookup(rows[picker.Cursor].ID)
		if err != nil {
			t.Fatalf("the row under the cursor is not a trait the book holds: %v", err)
		}
		if name := c.Lang.PassiveName(held); name != "" && !strings.Contains(body, name) {
			t.Errorf("the %s trait picker does not name %s:\n%s", lang, held.ID, body)
		}

		// And ? reads the trait out of the same book, in the sentences the
		// blurb screen gives -- which is the half a row's detail is too narrow
		// to carry, and the whole reason an English row being a bare id costs
		// nothing.
		picker, _ = picker.Update(c, press(t, "?"))
		if !picker.Reading {
			t.Fatalf("? opened no description on the %s trait picker", lang)
		}
		body, _ = picker.View(c)
		if !strings.Contains(body, c.Lang.GlossedPassive(held)) {
			t.Errorf("the %s description does not name %s:\n%s", lang, held.ID, body)
		}
		for _, line := range TraitSentences(c, held) {
			if sentence := strings.TrimSpace(line); !strings.Contains(body, sentence) {
				t.Errorf("the %s description of %s is missing %q:\n%s",
					lang, held.ID, sentence, body)
			}
		}
	}
}

// aSquadOfATraitHolder is a squad of one member pointed at whichever character
// in the book learns the most traits, and false when none does.
//
// Named apart from picker_test.go's aTraitHolder, which answers the narrower
// question — which character — and is what the kit picker's own fixture asks.
//
// It looks the character up rather than naming one, which is the rule the
// fixture exists to keep — and it is needed at all because the fixture cast
// learns no traits, so every screen that had opened the squad trait picker had
// opened an empty one and nothing measured a row of it.
func aSquadOfATraitHolder(t *testing.T, c Context) (SquadsScreen, bool) {
	t.Helper()
	s := aNewSquad(t, c, "trait")
	found, most := -1, 0
	for index, character := range s.Characters {
		if held := len(character.PassivesAt(
			progression.LevelCap, progression.Furthest)); held > most {
			found, most = index, held
		}
	}
	if found < 0 {
		return s, false
	}
	character := s.Characters[found]
	unit := s.Editing.Units[0]
	unit.Character, unit.Level, unit.Stage = character.ID, progression.LevelCap, ""
	known := character.SkillsAt(unit.Level, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}
	unit.Skills = known
	unit.Passives = character.PassivesAt(unit.Level, progression.Furthest)[:cast.TraitSlots]
	s.Editing.Units = []placement.Placement{unit}
	return s.EditUnit(0), true
}

// markerAt is where a mark sits in a rendered screen, as a line and a column,
// or (-1, -1) when it is not drawn.
//
// It reads the render rather than the state on purpose: everything below is
// about the picture, and every one of these assertions was already true of
// s.Unit.Slot while the grid beside it stood still.
func markerAt(body, mark string) (int, int) {
	for index, line := range strings.Split(body, "\n") {
		if at := strings.Index(line, mark); at >= 0 {
			return index, at
		}
	}
	return -1, -1
}

// drawn is a screen's body, which is what every assertion about the picture
// reads.
func drawn(c Context, s SquadsScreen) string {
	body, _ := s.View(c)
	return body
}

// TestTheFormationFollowsTheArrowsWhileTheCellIsChosen is the defect this grid
// was drawn to be free of.
//
// The slot row stepped s.Unit.Slot and the grid under it was built from
// s.Editing.Units, which Commit writes and which nothing writes until the
// member is left or a picker is opened. So the picture jumped to the new cell
// only after the choosing was over, which is the one moment it says nothing.
//
// Asserted on the render and not on s.Unit.Slot: the cell moving was already
// true before the fix, and a test of it would have passed throughout.
func TestTheFormationFollowsTheArrowsWhileTheCellIsChosen(t *testing.T) {
	c, _ := start(t, i18n.En)
	s := onField(t, c, aNewSquad(t, c, "live"), SquadUnitSlot)
	mark := fmt.Sprintf("(%d)", s.UnitIndex+1)
	opened := drawn(c, s)
	line, column := markerAt(opened, mark)
	if line < 0 {
		t.Fatalf("the member under edit is not marked on the grid:\n%s", opened)
	}

	was := s.Unit.Slot
	s = onSquads(t, c, s, "right")
	if s.Unit.Slot == was {
		t.Fatal("the chooser did not move, so there is nothing for the grid to follow")
	}
	stepped := drawn(c, s)
	movedLine, movedColumn := markerAt(stepped, mark)
	if movedLine < 0 {
		t.Fatalf("the member under edit vanished off the grid:\n%s", stepped)
	}
	if movedLine == line && movedColumn == column {
		t.Errorf("the cell went %s -> %s and the mark stayed at line %d column %d:\n%s",
			was, s.Unit.Slot, line, column, stepped)
	}

	// And it tracks rather than merely differing: stepping back puts the mark
	// where it started, on the cell the arrows are on and not on some other one.
	s = onSquads(t, c, s, "left")
	if s.Unit.Slot != was {
		t.Fatalf("stepping back landed on %s rather than on %s", s.Unit.Slot, was)
	}
	backLine, backColumn := markerAt(drawn(c, s), mark)
	if backLine != line || backColumn != column {
		t.Errorf("back on %s the mark is at line %d column %d, want %d and %d:\n%s",
			was, backLine, backColumn, line, column, drawn(c, s))
	}
}

// TestTheLiveFormationDrawsWithoutCommitting is the trap the live picture had to
// be built around, kept as a test rather than as a comment.
//
// The obvious fix is Commit on every keypress, and s.Editing.Units is shared
// with every copy of the screen — so a write from inside a drawing reaches all
// of them, which is what a value receiver looks like it prevents and does not.
// So the drawing reads the unit under edit and writes nothing, and this is the
// two halves of that: a key that changes nothing leaves the guard down, and a
// key that moves the cell leaves the squad's own copy alone until the member is
// left.
func TestTheLiveFormationDrawsWithoutCommitting(t *testing.T) {
	c, _ := start(t, i18n.En)
	s := aSavedSquad(t, c)
	s = onSquads(t, c, s, "enter")
	if s.Mode != SquadUnit {
		t.Fatalf("enter on the first member opened %v", s.Mode)
	}
	if s.Dirty() {
		t.Fatal("opening a member off a saved squad already claims changes")
	}

	// Walking the fields changes nothing about the unit, so nothing may reach
	// the guard.
	for range SquadUnitFieldCount + 1 {
		s = onSquads(t, c, s, "down")
		_ = drawn(c, s)
		if s.Dirty() {
			t.Fatalf("moving onto field %d raised the discard guard", s.Field)
		}
	}

	s = onField(t, c, s, SquadUnitSlot)
	index := s.UnitIndex
	before := s.Editing.Units[index].Slot
	s = onSquads(t, c, s, "right")
	if s.Unit.Slot == before {
		t.Fatal("the chooser did not move, so there is nothing to commit early")
	}
	chosen := s.Unit.Slot
	// Drawn more than once, because s.Editing.Units is a slice shared with every
	// copy of this value: a write into it from inside a drawing would reach all
	// of them, value receiver or not.
	for range 3 {
		_ = drawn(c, s)
	}
	if now := s.Editing.Units[index].Slot; now != before {
		t.Errorf("the squad's own copy moved to %s while the cell was being chosen, want %s",
			now, before)
	}
	// The picture is live all the same, which is what says the two are not the
	// same reading: the mark is on the cell the arrows are on.
	mark := fmt.Sprintf("(%d)", index+1)
	if line, _ := markerAt(drawn(c, s), mark); line < 0 {
		t.Fatalf("the member under edit is not marked on the grid:\n%s", drawn(c, s))
	}

	// esc is what commits, and it still does.
	s = onSquads(t, c, s, "esc")
	if now := s.Editing.Units[index].Slot; now != chosen {
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
		c, _ := start(t, lang)
		s := aNewSquad(t, c, "front")
		if len(s.Editing.Units) == 0 {
			t.Fatal("the fixture squad is empty, so nothing stands anywhere")
		}
		// One member in the front rank and one behind it, so the mark can be
		// shown to be under the first and not under the second.
		screened := s.Editing.Units[0]
		screened.ID, screened.Slot = "sau", hex.Offset{Col: back, Row: 0}
		s.Editing.Units = []placement.Placement{s.Editing.Units[0], screened}
		s.Editing.Units[0].Slot = hex.Offset{Col: front, Row: 0}
		s = s.EditUnit(0)
		body := drawn(c, s)

		caret := strings.Repeat("^", formationCell)
		caretLine, caretColumn := markerAt(body, caret)
		if caretLine < 0 {
			t.Fatalf("the %s grid marks no front rank:\n%s", lang, body)
		}
		if words := c.Text(i18n.SquadFormationFront); !strings.Contains(
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
		c, _ := start(t, lang)
		s := onField(t, c, aNewSquad(t, c, "rank"), SquadUnitSlot)
		body := drawn(c, s)
		if !strings.Contains(body, s.Unit.Slot.String()) {
			t.Errorf("the %s slot row lost the cell %s:\n%s", lang, s.Unit.Slot, body)
		}
		want := RankLabel(c, s.Unit.Slot)
		if want == "" {
			t.Fatalf("the member stands at %s, which is on no rank", s.Unit.Slot)
		}
		if !strings.Contains(body, want) {
			t.Errorf("the %s slot row does not say %q:\n%s", lang, want, body)
		}
		// And the reading follows the arrows too: walking off the front column
		// has to stop saying front rank.
		for range hex.FormationRows {
			s = onSquads(t, c, s, "right")
		}
		if RankLabel(c, s.Unit.Slot) == want {
			t.Fatalf("the chooser is still in the %s after a whole column", want)
		}
		if body := drawn(c, s); !strings.Contains(body, RankLabel(c, s.Unit.Slot)) {
			t.Errorf("the %s slot row still reads %q at %s:\n%s",
				lang, want, s.Unit.Slot, body)
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

// aSavedSquad is a squad taken up for editing off a catalogue that already holds
// it, which is the state the discard guard is about: there has to be something
// written down for what is in hand to differ from.
//
// ⚠️ **It is built as a value rather than written**, which is the rule at the top
// of this file — `Saved` is a field and `Open` takes its baseline off whatever it
// is handed, so nothing here needs `SaveSquad` to have run. Its member is built
// around whichever character in the book learns a trait, so the trait list has a
// row to toggle; naming one would tie this to content the author is free to
// change, which is what the injected fixture exists to avoid.
func aSavedSquad(t *testing.T, c Context) SquadsScreen {
	t.Helper()
	s := NewSquadsScreen(c)
	character, learns := aCharacterWithATrait(s.Characters)
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
	squad := placement.Squad{
		ID: "do-luu", Name: "đội lưu", Units: []placement.Placement{unit},
	}
	s.Saved = []placement.Squad{squad}
	s = s.Open(squad)
	if s.Mode != SquadEdit {
		t.Fatalf("the fixture squad opened in %v", s.Mode)
	}
	if s.Dirty() {
		t.Fatal("a squad just read off the catalogue already differs from it")
	}
	return s
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
// Commit writes a member back on the way out of it whether or not a key moved
// anything, so under a flag set from there, *opening* a member and pressing
// escape claimed a change — and arrowing the cell chooser onto another cell and
// back claimed one twice over. Both are round trips: what they put back is what
// they took, so the squad on the file and the squad in hand are the same squad
// and nobody may be asked about discarding it.
func TestARoundTripThroughAMemberLeavesTheGuardDown(t *testing.T) {
	c, _ := start(t, i18n.En)
	s := aSavedSquad(t, c)

	// One: open a member and leave it.
	s = onSquads(t, c, s, "enter")
	if s.Mode != SquadUnit {
		t.Fatalf("enter on the first member opened %v", s.Mode)
	}
	s = onSquads(t, c, s, "esc")
	if s.Dirty() {
		t.Error("opening a member and pressing escape changed the squad")
	}

	// Two: arrow the cell onto another and back.
	s = onSquads(t, c, s, "enter")
	s = onField(t, c, s, SquadUnitSlot)
	was := s.Unit.Slot
	s = onSquads(t, c, s, "right")
	if s.Unit.Slot == was {
		t.Fatal("the chooser did not move, so there is no round trip to make")
	}
	s = onSquads(t, c, s, "left")
	if s.Unit.Slot != was {
		t.Fatalf("stepping back landed on %s rather than on %s", s.Unit.Slot, was)
	}
	s = onSquads(t, c, s, "esc")
	if s.Dirty() {
		t.Error("arrowing the cell onto another and back changed the squad")
	}

	// And the whole point of it: leaving asks nothing.
	s, action := pressSquads(t, c, s, "esc")
	if action.Kind != Stay {
		t.Errorf("leaving asked for %v over changes nobody made", action.Kind)
	}
	if s.Mode != SquadList {
		t.Errorf("escape from an unchanged squad landed in %v", s.Mode)
	}
}

// TestEveryRealEditAsksBeforeItIsThrownAway is the other side of it, and it is a
// table rather than one case because catching every kind of edit was the latch's
// one virtue: a comparison that missed a field would lose that edit in silence,
// with no question asked and nothing on screen looking wrong.
//
// Every case leaves the screen on the squad view, so the escape below is the one
// the question hangs off. The member cases go in and come back out, because that
// is the route by which a member's own fields reach the squad.
//
// ⚠️ **It asserts the Action rather than a client's pending question.** What this
// screen owes is an Ask carrying the right wording and the right subject; that a
// client turns one into a question a `y` answers is that client's own test.
func TestEveryRealEditAsksBeforeItIsThrownAway(t *testing.T) {
	intoTheMember := func(t *testing.T, c Context, s SquadsScreen, field int) SquadsScreen {
		t.Helper()
		s = onSquads(t, c, s, "enter")
		if s.Mode != SquadUnit {
			t.Fatalf("enter on the first member opened %v", s.Mode)
		}
		return onField(t, c, s, field)
	}
	edits := []struct {
		what string
		make func(*testing.T, Context, SquadsScreen) SquadsScreen
	}{
		{"the name", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			return typeIntoSquads(t, c, s, "x")
		}},
		{"another member", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = onSquads(t, c, s, "down")
			s = onSquads(t, c, s, "enter")
			if len(s.Editing.Units) != 2 {
				t.Fatalf("the squad holds %d members", len(s.Editing.Units))
			}
			return onSquads(t, c, s, "esc")
		}},
		{"a member taken out", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = onSquads(t, c, s, "ctrl+x")
			if len(s.Editing.Units) != 0 {
				t.Fatalf("the squad still holds %d members", len(s.Editing.Units))
			}
			return s
		}},
		{"the character", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = intoTheMember(t, c, s, SquadUnitCharacter)
			was := s.Unit.Character
			s = onSquads(t, c, s, "right")
			if s.Unit.Character == was {
				t.Fatalf("the cast holds only %q, so it cannot be cycled", was)
			}
			return onSquads(t, c, s, "esc")
		}},
		{"the level", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = intoTheMember(t, c, s, SquadUnitLevel)
			was := s.Unit.Level
			s = onSquads(t, c, s, "backspace")
			if s.Unit.Level == was {
				t.Fatalf("the level field still reads %d", was)
			}
			return onSquads(t, c, s, "esc")
		}},
		{"the form", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = intoTheMember(t, c, s, SquadUnitStage)
			was := s.Unit.Stage
			s = onSquads(t, c, s, "right")
			if s.Unit.Stage == was {
				t.Fatalf("the form chooser stayed on %q", was)
			}
			return onSquads(t, c, s, "esc")
		}},
		{"the cell", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			s = intoTheMember(t, c, s, SquadUnitSlot)
			was := s.Unit.Slot
			s = onSquads(t, c, s, "right")
			if s.Unit.Slot == was {
				t.Fatalf("the cell chooser stayed on %s", was)
			}
			return onSquads(t, c, s, "esc")
		}},
		{"the kit", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			return throughTheList(t, c, intoTheMember(t, c, s, SquadUnitSkills))
		}},
		{"the trait", func(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
			return throughTheList(t, c, intoTheMember(t, c, s, SquadUnitPassives))
		}},
	}
	for _, edit := range edits {
		t.Run(edit.what, func(t *testing.T) {
			c, _ := start(t, i18n.En)
			s := edit.make(t, c, aSavedSquad(t, c))
			if s.Mode != SquadEdit {
				t.Fatalf("the edit left the screen in %v", s.Mode)
			}
			if !s.Dirty() {
				t.Fatalf("changing %s left the squad reading as the one written down", edit.what)
			}
			s, action := pressSquads(t, c, s, "esc")
			if action.Kind != Ask {
				t.Fatalf("leaving after changing %s asked for %v rather than a question",
					edit.what, action.Kind)
			}
			if action.Question != i18n.SquadDiscard {
				t.Errorf("the question asked was %v", action.Question)
			}
			if got, want := action.About, (SquadsAsk{Kind: SquadsAskSquadInHand}); got != want {
				t.Errorf("the question is about %v, want %v", got, want)
			}
			// And answering it puts the file's version back rather than only
			// leaving, which is what "discard" says.
			answered, back := s.Confirmed(c, action.About)
			if back.Kind != Stay {
				t.Errorf("confirming the discard asked for %v", back.Kind)
			}
			if answered.Mode != SquadList {
				t.Errorf("confirming the discard left the builder in %v", answered.Mode)
			}
			if answered.Dirty() {
				t.Error("the squad in hand still differs from the one written down")
			}
		})
	}

	// The id is the one field a saved squad does not offer — changing it would
	// write a second squad rather than rename this one — so it is asked of a
	// squad nobody has written yet, where typing one is the whole of the edit.
	t.Run("the id", func(t *testing.T) {
		c, _ := start(t, i18n.En)
		s := onSquads(t, c, NewSquadsScreen(c), "n")
		if s.Dirty() {
			t.Fatal("a squad nobody has typed into already claims changes")
		}
		s = typeIntoSquads(t, c, s, "moi")
		if !s.Dirty() {
			t.Fatal("typing an id left the squad reading as an empty one")
		}
		if _, action := pressSquads(t, c, s, "esc"); action.Kind != Ask {
			t.Fatalf("leaving after typing an id asked for %v", action.Kind)
		}
	})
}

// throughTheList opens the list under the cursor, toggles the row under its own,
// takes the answer and comes back out to the squad.
func throughTheList(t *testing.T, c Context, s SquadsScreen) SquadsScreen {
	t.Helper()
	s, action := pressSquads(t, c, s, "enter")
	if action.Kind != Pick || action.Picker == nil {
		t.Fatalf("the field asked for %v rather than a picker", action.Kind)
	}
	picker := action.Picker.Raise()
	if len(picker.Options) == 0 {
		t.Fatal("the list is empty, so there is no row to toggle")
	}
	picker, _ = picker.Update(c, press(t, "space"))
	closed, result := picker.Update(c, press(t, "enter"))
	if closed != nil {
		t.Fatal("the list is still open")
	}
	if !result.Answered {
		t.Fatal("the picker came down with no answer")
	}
	s, _ = s.Picked(c, result.Into, result.Answer)
	return onSquads(t, c, s, "esc")
}

// TestEscFromTheCatalogueAsksTheClientToGoBack is the one key on this screen
// that used to name a view of the client that drew it.
//
// It went to that client's menu by name, which is true today and is a fact about
// which client is in front rather than about the catalogue — the same argument
// the chart's own way back was converted under.
func TestEscFromTheCatalogueAsksTheClientToGoBack(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	for _, name := range []string{"esc", "q"} {
		_, action := pressSquads(t, c, NewSquadsScreen(c), name)
		if action.Kind != Back {
			t.Errorf("%q on the catalogue asked for %v, want a step back", name, action.Kind)
		}
	}
}

// aCatalogueOfTwoSquads is a builder sitting on a catalogue with two squads in
// it, cursor on the first.
//
// ⚠️ **Two, and that is what makes the two tests below assertions rather than
// counts.** With one squad on the file any id at all names it, so a key reading
// the wrong row — the one after the cursor, the last one, the first one
// regardless — passes a one-squad fixture completely.
func aCatalogueOfTwoSquads(t *testing.T, c Context) SquadsScreen {
	t.Helper()
	s := aSavedSquad(t, c)
	second := s.Editing.Clone()
	second.ID, second.Name = "do-hai", "đội hai"
	s.Mode = SquadList
	s.Saved = []placement.Squad{s.Baseline.Clone(), second}
	s.Cursor = 0
	if s.Saved[0].ID == s.Saved[1].ID {
		t.Fatalf("both fixture squads are called %q, so nothing can tell them apart",
			s.Saved[0].ID)
	}
	return s
}

// TestTheDeleteQuestionNamesTheSquadUnderTheCursor is the half of the delete
// that belongs to this screen: which squad the question is about.
//
// The id travels with the question rather than being read back off the cursor
// when the answer arrives, and this is what says so.
func TestTheDeleteQuestionNamesTheSquadUnderTheCursor(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	s := aCatalogueOfTwoSquads(t, c)
	for _, cursor := range []int{0, 1} {
		s.Cursor = cursor
		_, action := pressSquads(t, c, s, "d")
		if action.Kind != Ask {
			t.Fatalf("d on the catalogue asked for %v rather than a question", action.Kind)
		}
		if action.Question != i18n.SquadDiscardSaved {
			t.Errorf("the question asked was %v", action.Question)
		}
		want := SquadsAsk{Kind: SquadsAskSavedSquad, ID: s.Saved[cursor].ID}
		if action.About != want {
			t.Errorf("with the cursor on row %d the question is about %v, want %v",
				cursor, action.About, want)
		}
	}
	// And an empty catalogue asks nothing at all rather than asking about a
	// squad that is not there.
	empty := s
	empty.Saved = nil
	if _, action := pressSquads(t, c, empty, "d"); action.Kind != Stay {
		t.Errorf("d on an empty catalogue asked for %v", action.Kind)
	}
}

// TestTheCatalogueRaisesTheFightAboutTheSquadUnderTheCursor is `f`, and it is
// the one raise in this package that names a screen the package will never draw.
//
// ⚠️ **It names the squad by id.** Reading it back off the catalogue's cursor
// after the raise had arrived would work — the client owns both screens — and
// that is the mistake Action.About exists to have stopped for a question, made
// worse: a pending question at least freezes the keyboard while it waits.
func TestTheCatalogueRaisesTheFightAboutTheSquadUnderTheCursor(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	s := aCatalogueOfTwoSquads(t, c)
	for _, cursor := range []int{0, 1} {
		s.Cursor = cursor
		_, action := pressSquads(t, c, s, "f")
		if action.Kind != Raise || action.Target != Fight {
			t.Fatalf("f asked for %v at %v, want a raise of the fight",
				action.Kind, action.Target)
		}
		want := Subject{Kind: SquadSubject, ID: s.Saved[cursor].ID}
		if action.Subject != want {
			t.Errorf("with the cursor on row %d the raise is about %v, want %v",
				cursor, action.Subject, want)
		}
	}
	// Nothing to fight is not a fight against whatever sorted first.
	empty := s
	empty.Saved = nil
	if _, action := pressSquads(t, c, empty, "f"); action.Kind != Stay {
		t.Errorf("f on an empty catalogue asked for %v", action.Kind)
	}
}

// TestTheBuilderOffersAShownCharacterAndNotAHeldBackOne is the rule on its own,
// over a cast written here.
//
// Authored rather than read off the shipped book, and that is the point of the
// fixture: a test that passed because the shipped data happens to hide exactly
// one character is a test that breaks the day the data changes, and what it
// would be measuring is cast.json rather than the rule.
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
	c, _ := start(t, i18n.En)
	s := NewSquadsScreen(c)
	shipped := s.Characters
	if len(shipped) == 0 {
		t.Fatal("the fixture cast is empty, so no member can be added at all")
	}
	held := shipped[0]
	held.ID, held.Hidden = "fixture-anime.recluse", true
	shown := shipped[0]
	s.Characters = append([]cast.Character{held}, shown)

	s = onSquads(t, c, s, "n")
	s = typeIntoSquads(t, c, s, "moi")
	s = onSquads(t, c, s, "enter")
	if s.Mode != SquadUnit {
		t.Fatalf("adding a member did not open it, mode is %v", s.Mode)
	}
	if s.Unit.Character == held.ID {
		t.Errorf("a new member started on %q, which the cast holds back", held.ID)
	}
	if s.Unit.Character != shown.ID {
		t.Errorf("a new member started on %q, want the first character still offered, %q",
			s.Unit.Character, shown.ID)
	}
}

// TestEverySquadsPickDestinationWritesItsOwnField is the effect half of the
// client's totality walk: a destination with an entry that writes the wrong
// field passes a walk over the count completely.
func TestEverySquadsPickDestinationWritesItsOwnField(t *testing.T) {
	c, _ := start(t, i18n.En)
	base := aNewSquad(t, c, "into")
	character, known := base.Character()
	if !known {
		t.Fatal("the member names no character in the book")
	}
	lists := map[SquadsPick]struct {
		offered []string
		read    func(SquadsScreen) []string
	}{
		SquadsPickKit: {
			offered: character.SkillsAt(base.Unit.Level, base.Form()),
			read:    func(s SquadsScreen) []string { return s.Unit.Skills },
		},
		SquadsPickTrait: {
			offered: character.PassivesAt(base.Unit.Level, base.Form()),
			read:    func(s SquadsScreen) []string { return s.Unit.Passives },
		},
	}
	// The walk: every destination but the zero has to be here, and the count is
	// what says so rather than a list somebody remembered to extend.
	if got, want := len(lists), int(SquadsPickCount)-1; got != want {
		t.Fatalf("this table covers %d destinations of the %d declared", got, want)
	}

	for destination, list := range lists {
		if len(list.offered) == 0 {
			t.Logf("the fixture character offers nothing for destination %d, so its "+
				"answer is the empty one", destination)
		}
		chosen := append([]string(nil), list.offered...)
		if len(chosen) > cast.TraitSlots && destination == SquadsPickTrait {
			chosen = chosen[:cast.TraitSlots]
		}
		if len(chosen) > cast.SkillSlots {
			chosen = chosen[:cast.SkillSlots]
		}
		landed, action := base.Picked(c, destination, PickAnswer{Chosen: chosen})
		if action.Kind != Stay {
			t.Errorf("destination %d asked for %v; a pick fills in a field and leaves the "+
				"reader on the member", destination, action.Kind)
		}
		if got := list.read(landed); !slices.Equal(got, chosen) {
			t.Errorf("destination %d left its own field %v, want %v", destination, got, chosen)
		}
		// And nowhere else. This is what a destination pointed at the wrong field
		// looks like, and nothing else here can see it.
		for other, otherList := range lists {
			if other == destination || len(chosen) == 0 {
				continue
			}
			if got := otherList.read(landed); len(got) != 0 {
				t.Errorf("destination %d also wrote %v into destination %d's field",
					destination, got, other)
			}
		}
		// And the answer reached the squad rather than only the member under
		// edit, which is what Commit is for.
		if got := list.read(landed); len(got) > 0 &&
			!slices.Equal(list.read(SquadsScreen{Unit: landed.Editing.Units[landed.UnitIndex]}), got) {
			t.Errorf("destination %d wrote %v into the member and not into the squad",
				destination, got)
		}
	}

	// And a destination this screen does not own lands nowhere rather than in
	// whichever field sorted first — the picker carries an `any`, so a value from
	// another screen's vocabulary can arrive here.
	untouched, action := base.Picked(c, "not one of ours", PickAnswer{Chosen: []string{"x"}})
	if action.Kind != Stay {
		t.Errorf("a destination this screen does not own asked for %v", action.Kind)
	}
	if len(untouched.Unit.Skills) != 0 || len(untouched.Unit.Passives) != 0 {
		t.Error("a destination this screen does not own still filled in a field")
	}
}

// TestAQuestionThisScreenDidNotAskIsDeclined is the other end of Action.About
// being an `any`: a value out of somebody else's vocabulary arrives here, and
// guessing at one would be the delete taking whatever id happened to be around.
func TestAQuestionThisScreenDidNotAskIsDeclined(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	before := aCatalogueOfTwoSquads(t, c)
	for _, about := range []any{nil, "not one of ours", SquadsAsk{}} {
		after, action := before.Confirmed(c, about)
		if action.Kind != Stay {
			t.Errorf("a question about %v asked for %v", about, action.Kind)
		}
		if len(after.Saved) != len(before.Saved) {
			t.Errorf("a question about %v left %d squads of the %d there were",
				about, len(after.Saved), len(before.Saved))
		}
		if after.Mode != before.Mode {
			t.Errorf("a question about %v moved the screen to %v", about, after.Mode)
		}
	}
}
