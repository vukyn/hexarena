package main

import (
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
	draw "github.com/vukyn/hexarena/internal/screen"
)

// What is left here after the squad builder moved into internal/screen: the
// wiring, the file, and the picker as this client puts it in front.
//
// The split is the one every step since #205 has taken — a test that asserts
// **where you land**, or that a key **reaches** something, is a client test and
// stays; a test that asserts what the screen **draws**, or how its cursor, its
// modes and its fields behave, went with it. What that leaves here is the menu
// entry, the whole feature driven end to end onto a real `squads.json`, the two
// states that need a directory the test wrote into, and the reading pane, which
// is drawn over this client's own `m.picker` and inside its own `frame`.

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

// TestTheCatalogueGoesBackToWhereverRaisedIt is the client half of the
// catalogue's esc: internal/screen asserts the screen asks for a draw.Back, and
// this asserts what a Back off this screen costs.
func TestTheCatalogueGoesBackToWhereverRaisedIt(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = menuTo(t, m, screenSquads)
	if back := key(t, m, "esc"); back.screen != screenMenu {
		t.Errorf("escape from the catalogue landed on screen %v", back.screen)
	}
}

// TestASquadIsBuiltAndSavedAndComesBack is the whole feature end to end, driven
// through the keys a person presses.
//
// It stays in this client because it is about the **file**: the write goes onto
// a scratch directory, a library loaded fresh off it has to see the squad, and
// the squad has to take the field. Nothing in internal/screen writes.
func TestASquadIsBuiltAndSavedAndComesBack(t *testing.T) {
	m, lib, dir := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)

	// n starts one, and the id is typed once.
	m = typeText(t, m, "n")
	if m.squad.Mode != draw.SquadEdit {
		t.Fatalf("n did not open a squad to build")
	}
	m = typeText(t, m, "greens")

	// enter on the row under the members adds somebody, which opens that unit.
	m = key(t, m, "enter")
	if m.squad.Mode != draw.SquadUnit {
		t.Fatalf("adding a member did not open it, mode is %v", m.squad.Mode)
	}
	if len(m.squad.Editing.Units) != 1 {
		t.Fatalf("the squad holds %d members", len(m.squad.Editing.Units))
	}
	// A member arrives with a character, a level and a cell, because a blank one
	// would be a form somebody has to fill from nothing.
	unit := m.squad.Editing.Units[0]
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
	if m.squad.Err == nil {
		t.Fatal("a squad whose member brings no skills was saved")
	}
	if !strings.Contains(m.squad.Err.Error(), "chooses no skills") {
		t.Errorf("the refusal is %q, want the loadout rule's own words", m.squad.Err)
	}

	// Choose a kit: down to the member, enter to open it, down to the skills
	// field, enter to raise the picker, space to take the row under the cursor.
	m = key(t, m, "up")
	m = key(t, m, "enter")
	for m.squad.Field != draw.SquadUnitSkills {
		m = key(t, m, "down")
	}
	m = key(t, m, "enter")
	if m.picker == nil {
		t.Fatal("the skills field raised no picker")
	}
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if len(m.squad.Unit.Skills) != 1 {
		t.Fatalf("the picker handed back %v", m.squad.Unit.Skills)
	}
	// And the choice reached the squad, not only the unit under edit.
	if len(m.squad.Editing.Units[0].Skills) != 1 {
		t.Errorf("the squad's own copy brings %v", m.squad.Editing.Units[0].Skills)
	}

	m = key(t, m, "esc")
	m = key(t, m, "ctrl+s")
	if m.squad.Err != nil {
		t.Fatalf("saving a finished squad was refused: %v", m.squad.Err)
	}
	if len(m.squad.Notes) == 0 {
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

// aSquadPicker is a squad under construction with one of its two loadout
// pickers open, driven through the keys a person presses.
func aSquadPicker(t *testing.T, lang i18n.Lang, id string, field int) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = menuTo(t, m, screenSquads)
	m = typeText(t, m, "n")
	m = typeText(t, m, id)
	m = key(t, m, "enter")
	for m.squad.Field != field {
		m = key(t, m, "down")
	}
	m = key(t, m, "enter")
	if m.picker == nil {
		t.Fatalf("the field raised no picker")
	}
	return m
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
		m := aSquadPicker(t, lang, "doc", draw.SquadUnitSkills)
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
		for _, line := range draw.SkillLines(m.ctx(), declared) {
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
		if !slices.Equal(m.squad.Unit.Skills, chosen) {
			t.Errorf("the %s picker handed back %v, want %v", lang, m.squad.Unit.Skills, chosen)
		}
	}
}

// TestWalkingWhileReadingMovesToTheNextDescription is why the cursor keys are
// live here at all: comparing two skills means reading one after the other, and
// going back to the list between them is the interaction the blurb screen
// already refused.
func TestWalkingWhileReadingMovesToTheNextDescription(t *testing.T) {
	m := aSquadPicker(t, i18n.Vi, "doc", draw.SquadUnitSkills)
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
//
// It stays in this client because the cut it refuses is `frame`'s, which is this
// client's own.
func TestTheReadingStateIsNotCutAndCannotBeScrolledOffItsAnswer(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m := aSquadPicker(t, lang, "doc", draw.SquadUnitSkills)
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
		for _, line := range draw.SkillLines(m.ctx(), declared) {
			if sentence := strings.TrimSpace(line); !strings.Contains(body, sentence) {
				t.Errorf("scrolling past the end of the %s description lost %q:\n%s",
					lang, sentence, body)
			}
		}
	}
}

// aSavedSquad is a squad written to the file and taken back up for editing,
// which is the state the discard guard is about: there has to be something
// written down for what is in hand to differ from.
//
// Its member is built around whichever character in the book learns a trait, so
// the trait list has a row to toggle. Naming one would tie this to content the
// author is free to change, which is what the injected fixture exists to avoid.
//
// ⚠️ **This one really writes**, unlike its namesake in internal/screen, which
// builds the same state as a value. The two confirms it feeds are about a file:
// one asks whether the catalogue on disk still holds what it held, and the other
// deletes a row of it.
func aSavedSquad(t *testing.T) model {
	t.Helper()
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	s := m.squad.Begin()
	s.Editing.ID, s.Editing.Name = "do-luu", "đội lưu"
	s.IDInput.SetValue(s.Editing.ID)
	s.NameInput.SetValue(s.Editing.Name)

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
	s.Editing.Units = []placement.Placement{unit}

	m.squad = s
	m = withASquadSaved(t, m)
	m.squad = m.squad.Open(m.squad.Saved[0])
	if m.squad.Mode != draw.SquadEdit {
		t.Fatalf("the fixture squad opened in %v", m.squad.Mode)
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

// TestSavingSettlesTheGuardAndReopeningStartsClean is the third state a
// comparison has to get right: a write moves the thing being compared against,
// so a squad just saved is a squad with nothing outstanding — and one taken back
// up off the file starts from the file rather than from whatever the screen was
// last holding.
//
// It needs the write, so it stays here.
func TestSavingSettlesTheGuardAndReopeningStartsClean(t *testing.T) {
	m := aSavedSquad(t)
	m = typeText(t, m, "x")
	if !m.squad.Dirty() {
		t.Fatal("typing into the name left the guard down")
	}
	m = key(t, m, "ctrl+s")
	if m.squad.Err != nil {
		t.Fatalf("the save was refused: %v", m.squad.Err)
	}
	if m.squad.Dirty() {
		t.Error("a squad just written still reads as changed")
	}
	m = key(t, m, "esc")
	if m.guard != nil {
		t.Error("leaving a squad just saved asked before discarding it")
	}

	m = key(t, m, "enter")
	if m.squad.Mode != draw.SquadEdit {
		t.Fatalf("enter on the catalogue opened %v", m.squad.Mode)
	}
	if m.squad.Dirty() {
		t.Error("a squad taken back up off the file already differs from it")
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
//
// ⚠️ **It stays here because the claim is about the FILE.** The screen-level
// rule — a held-back character stays offered where it is already chosen — is
// `screen.TestTheBuilderOffersAShownCharacterAndNotAHeldBackOne` and
// `screen.TestANewMemberNeverStartsOnAHeldBackCharacter`; what this adds is that
// a squad **on disk** naming one still loads, still opens and still comes back
// out unchanged.
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
		if len(m.squad.Saved) != 1 {
			t.Fatalf("%v: the catalogue holds %d squads, want the one just written", lang, len(m.squad.Saved))
		}
		m = key(t, m, "enter")
		if m.squad.Mode != draw.SquadEdit {
			t.Fatalf("%v: enter on the catalogue opened %v", lang, m.squad.Mode)
		}
		m = key(t, m, "enter")
		if m.squad.Mode != draw.SquadUnit {
			t.Fatalf("%v: enter on the member opened %v", lang, m.squad.Mode)
		}
		if m.squad.Field != draw.SquadUnitCharacter {
			t.Fatalf("%v: the member opened on field %d, want the character row", lang, m.squad.Field)
		}

		// The character resolves, so everything read against it is still there.
		// This is what a filtered lookup slice takes away, and it takes it away
		// silently — the row still prints the id.
		character, known := m.squad.Character()
		if !known {
			t.Fatalf("%v: the member's own character is not in the cast the screen holds", lang)
		}
		if !character.Hidden {
			t.Fatalf("%v: the fixture character is not held back, so this measures nothing", lang)
		}
		if len(m.squad.StageChoices()) == 0 {
			t.Errorf("%v: the member offers no form at all", lang)
		}
		if len(character.SkillsAt(m.squad.Unit.Level, m.squad.Form())) == 0 {
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
		if m.squad.Mode != draw.SquadEdit {
			t.Fatalf("%v: esc out of the member landed in %v", lang, m.squad.Mode)
		}
		if m.squad.Dirty() {
			t.Errorf("%v: opening a member holding a held-back character changed the squad", lang)
		}
		if got := m.squad.Editing.Units[0].Character; got != held.ID {
			t.Errorf("%v: the member now names %q, want the %q it was saved with", lang, got, held.ID)
		}

		// ⚠️ The round trip is the assertion, not the single press. Stepping the
		// chooser is an author asking for a change, so one press moving the
		// character is correct either way — what a dropped row costs is the way
		// BACK, because a character off the list is one the arrows can never
		// return to.
		m = key(t, m, "enter")
		if m.squad.Mode != draw.SquadUnit || m.squad.Field != draw.SquadUnitCharacter {
			t.Fatalf("%v: reopening the member landed in %v on field %d", lang, m.squad.Mode, m.squad.Field)
		}
		was := m.squad.Unit.Character
		m = key(t, m, "right")
		if m.squad.Unit.Character == was {
			t.Fatalf("%v: the chooser did not move at all, so the round trip below measures nothing", lang)
		}
		// Stepped onto a character that is not held back, so the note goes: it
		// reads the answer in hand rather than the exemption behind the list.
		if body := m.screenContent(); strings.Contains(body, m.text(i18n.SquadHeldBack)) {
			t.Errorf("%v: the member now names %q, which is offered like any other, and the screen still calls it held back",
				lang, m.squad.Unit.Character)
		}
		m = key(t, m, "left")
		if m.squad.Unit.Character != was {
			t.Errorf("%v: stepping the character chooser away and back landed on %q, want %q — the held-back character is off the list",
				lang, m.squad.Unit.Character, was)
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
		for _, candidate := range m.squad.Characters {
			if candidate.Hidden && candidate.ID != was {
				otherHeldBack = append(otherHeldBack, candidate.ID)
			}
		}
		if !slices.Contains(otherHeldBack, unnamed.ID) {
			t.Fatalf("%v: the second held-back character %q is not in the cast the screen holds; it holds back %v",
				lang, unnamed.ID, otherHeldBack)
		}
		visited := map[string]bool{}
		for range len(m.squad.Characters) + 1 {
			m = key(t, m, "right")
			visited[m.squad.Unit.Character] = true
			if m.squad.Unit.Character == was {
				break
			}
		}
		if m.squad.Unit.Character != was {
			t.Fatalf("%v: walking the chooser right round ended on %q rather than back at %q",
				lang, m.squad.Unit.Character, was)
		}
		for _, id := range otherHeldBack {
			if visited[id] {
				t.Errorf("%v: the chooser offered %q, which the cast holds back and no member had chosen", lang, id)
			}
		}

		// And the member the round trip put back is the member that goes back
		// into the squad, rather than only into the copy under edit.
		m = key(t, m, "esc")
		if got := m.squad.Editing.Units[0].Character; got != held.ID {
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
