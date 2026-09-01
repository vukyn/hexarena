package main

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// The squad builder is three views of one thing, and they are one screen rather
// than three because they are one decision taken at three depths: which squads
// exist, who is in this one, and what that unit brings. Splitting them would put
// the half-built squad somewhere both of the others could reach, which is the
// shape that lets two screens disagree about what is being edited.
type squadMode int

const (
	squadList squadMode = iota
	squadEdit
	squadUnit
)

// The fields of one unit, in the order they are asked.
//
// Character first because everything under it is read against the character:
// the level bounds the forms, the form bounds the learnset, and the kit is
// chosen out of that. Slot last of the settled facts because it is the only one
// that says nothing about the unit itself — where it stands is a fact about the
// squad.
const (
	unitCharacter = iota
	unitLevel
	unitStage
	unitSlot
	unitSkills
	unitPassives
	unitFieldCount
)

type squadScreen struct {
	mode squadMode
	// saved is the catalogue as it is on disk, refreshed when the screen is
	// entered so a squad written by the other front-end is not invisible here.
	saved  []placement.Squad
	cursor int

	// characters is the WHOLE cast, held rather than looked up per keystroke
	// because cycling walks it.
	//
	// The whole cast rather than the offered part of it, because this slice
	// answers two different questions and only one of them is about choosing.
	// character() looks a member's character up here to read its forms, its
	// learnset and its traits, and a squad on the file may name a character that
	// has since been hidden — so a filtered slice would leave that member with no
	// forms and an empty kit picker. What may be *chosen* is offeredCharacters,
	// asked of this one at each of the two sites that choose.
	characters []cast.Character

	// editing is the squad in hand. It is a whole squad rather than an index
	// into saved, because a squad being built has not been saved yet and an
	// index would have nothing to point at.
	editing placement.Squad
	// baseline is the squad as it was last written down: what open() read off
	// the file, what save() put back, or the empty squad begin() started from.
	// The guard on leaving compares against it.
	//
	// It is a reading rather than a latched flag because a latch cannot tell a
	// squad that changed from one that was merely touched. commit() writes a
	// member back whether or not anything moved, and it runs on the way out of
	// every member, so opening one and pressing escape used to be
	// indistinguishable from editing it — the question was asked over changes
	// nobody had made, which is how a question stops being read.
	//
	// It is a Clone, not the value editing was set from: editing is mutated in
	// place, and a baseline sharing its slices would compare equal to itself
	// for ever.
	baseline placement.Squad
	// units is the cursor over the squad's members, and it may sit one past the
	// last, which is the row that adds another.
	units int

	idInput   textinput.Model
	nameInput textinput.Model
	// editingID is true while the id field is the one being typed into. An id is
	// asked once, when a squad is created: changing it later would write a
	// second squad rather than rename this one, and a field that silently makes
	// a copy is worse than one that is not offered.
	editingID bool

	// unit is the member under edit, and index is where it sits in the squad.
	unit      placement.Placement
	unitIndex int
	// unitOpenedAs is the character this member named when it was opened, and it
	// is what the character chooser keeps offering however held back that
	// character is. See offeredCharacters.
	//
	// Read from here rather than from unit.Character — the obvious spelling —
	// because the obvious spelling makes the list change shape while it is being
	// walked: stepping off a held-back character drops it, so `right` then
	// `left` lands one row short of where it started and the character is
	// unreachable for the rest of the edit. A chooser is a fixed list the arrows
	// walk, so what may be chosen has to be settled when the member is opened
	// rather than recomputed from the answer in hand.
	unitOpenedAs string
	field        int
	levelInput   textinput.Model

	err   error
	notes []forge.Note
}

func newSquadScreen(lib *forge.Library) squadScreen {
	s := squadScreen{
		idInput:    newInput(),
		nameInput:  newInput(),
		levelInput: *numberField(""),
	}
	s.idInput.Prompt, s.nameInput.Prompt = "", ""
	s.idInput.CharLimit, s.nameInput.CharLimit = 32, 48
	s.idInput.SetWidth(34)
	s.nameInput.SetWidth(40)
	return s.refresh(lib)
}

// refresh re-reads the catalogue, and leaves a squad under edit alone: entering
// the screen while building one is the ordinary way back from a picker.
func (s squadScreen) refresh(lib *forge.Library) squadScreen {
	s.saved = lib.Squads()
	s.characters = lib.Characters().All()
	s.cursor = clamp(s.cursor, 0, len(s.saved)-1)
	return s
}

// offeredCharacters is who the builder proposes: every character not held back,
// plus — always — the one named by `keep`.
//
// keep is squadScreen.unitOpenedAs, the character the member under edit named
// when it was opened, and the empty string when there is none, which is what a
// brand new member passes. It is deliberately not "whatever is chosen right
// now" — see the field's own comment for why a list that moves while it is
// walked is a chooser the arrows cannot bring back.
//
// ⚠️ **That second half is the whole reason this is a function and not an `if`
// at the call site.** A squad on the file may name a character that has since
// been hidden, and a chooser that simply dropped it would step off it the first
// time an author touched the row: cycle() finds no match, falls back to index 0
// and writes somebody else into a member nobody asked to change — a silent edit
// to the author's own saved file, in the one screen here that writes it. Hidden
// therefore means *not offered for a new choice*, never *taken away from a
// choice already made*, and declaration order is preserved so keeping the one
// named costs the list nothing but its own row.
//
// It filters at the call site rather than through a `cast.Book` accessor, and
// that is a judgement rather than an accident. The rule this applies is not a
// fact about the cast — "the one this member already holds" is a fact about the
// squad being edited, which no book can know — so an accessor could answer only
// the first half and this screen would still have to union the second back in.
// That is a second vocabulary for "the cast" that no caller could use unaided,
// while Book.All() stays the honest answer to "the cast" for the browser, the
// builds screen, `hexforge list`, the spar, the roster and the restriction
// picker, none of which are choosing a side to fight with. The day a second
// screen wants the offered list, the argument reverses and the accessor is the
// change to make.
func offeredCharacters(all []cast.Character, keep string) []cast.Character {
	out := make([]cast.Character, 0, len(all))
	for _, character := range all {
		if character.Hidden && character.ID != keep {
			continue
		}
		out = append(out, character)
	}
	return out
}

func (s squadScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch s.mode {
	case squadEdit:
		return s.updateEdit(m, message)
	case squadUnit:
		return s.updateUnit(m, message)
	}
	return s.updateList(m, message)
}

func (s squadScreen) updateList(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc", "q":
		return m.enter(screenMenu), nil
	case "up", "k":
		s.cursor = clamp(s.cursor-1, 0, len(s.saved)-1)
	case "down", "j":
		s.cursor = clamp(s.cursor+1, 0, len(s.saved)-1)
	case "n":
		s = s.begin()
	case "enter":
		if len(s.saved) > 0 {
			s = s.open(s.saved[clamp(s.cursor, 0, len(s.saved)-1)])
		}
	case "f":
		// The fight carries its own two sides, so this hands it one: the squad
		// under the cursor becomes the home side, which is what f has always
		// meant here. Seeding it is the whole of the difference — the fight
		// reads its own field afterwards, so walking the catalogue behind it
		// cannot change what is being measured.
		if len(s.saved) > 0 {
			m.squad = s
			m.fight.home = clamp(s.cursor, 0, len(s.saved)-1)
			return m.enter(screenFight), nil
		}
	case "d":
		if len(s.saved) > 0 {
			id := s.saved[clamp(s.cursor, 0, len(s.saved)-1)].ID
			m.squad = s
			return m.ask(i18n.SquadDiscardSaved, screenSquads,
				guardSubject{Kind: guardsASavedSquad, ID: id}), nil
		}
	}
	m.squad = s
	return m, nil
}

// Confirmed answers whichever of the builder's two questions was asked, told
// apart by what the question was about rather than by what mode the screen is
// in.
//
// ⚠️ **This is the one screen with two confirms**, and it is why a pending
// question carries a subject at all. The mode would answer it — the guard
// freezes every other key, so the screen cannot have changed depth between the
// question and the answer — and that is state being read to recover something
// the question could simply have said. The delete also needs a value the screen
// does not hold: the id under the catalogue's cursor when `d` was pressed.
//
// Both stay on this screen, so both hand back the zero action.
func (s squadScreen) Confirmed(c draw.Context, about guardSubject) (squadScreen, draw.Action) {
	switch about.Kind {
	case guardsASavedSquad:
		// The refusal is kept on the screen rather than thrown, because a file
		// that would not go is something the reader has to be told about where
		// they asked — and a failed delete leaves the catalogue exactly as it
		// was, so nothing is refreshed.
		if err := c.Lib.DeleteSquad(about.ID); err != nil {
			s.err = err
			return s, draw.Action{}
		}
		s.err = nil
		s.notes = nil
		return s.refresh(c.Lib), draw.Action{}
	case guardsTheSquadInHand:
		s.mode = squadList
		// Discarding is what the question said, so what is in hand goes back to
		// the squad last written rather than being left changed behind a mode
		// that no longer reads it.
		s.editing = s.baseline.Clone()
		return s.refresh(c.Lib), draw.Action{}
	}
	// guardsNothing, which the builder never asks. Named by falling through
	// rather than by a default that does something: a question this screen did
	// not ask has no answer here, and guessing at one would be the delete taking
	// whatever id happened to be around.
	return s, draw.Action{}
}

// begin starts a squad nobody has written yet.
func (s squadScreen) begin() squadScreen {
	s.mode = squadEdit
	s.editing = placement.Squad{}
	s.baseline = s.editing.Clone()
	s.units = 0
	s.editingID = true
	s.err, s.notes = nil, nil
	s.idInput.SetValue("")
	s.nameInput.SetValue("")
	s.idInput.Focus()
	s.nameInput.Blur()
	return s
}

// open takes a saved squad up for editing, on its own copy.
func (s squadScreen) open(squad placement.Squad) squadScreen {
	s.mode = squadEdit
	s.editing = squad.Clone()
	s.baseline = s.editing.Clone()
	s.units = 0
	s.editingID = false
	s.err, s.notes = nil, nil
	s.idInput.SetValue(squad.ID)
	s.nameInput.SetValue(squad.Name)
	s.idInput.Blur()
	s.nameInput.Focus()
	return s
}

// dirty reports whether the squad in hand differs from the one last written
// down, which is the whole of what the guard on leaving asks.
//
// It is a comparison rather than a flag, so a round trip that changes nothing —
// opening a member and leaving it, stepping the cell chooser onto another cell
// and back — leaves the question unasked. Everything under edit has reached
// s.editing by the time this is read: the only route out of a member commits
// first, and so does opening either of its lists, while the id and the name are
// written on every keypress that reaches them.
func (s squadScreen) dirty() bool {
	return !s.editing.Equal(s.baseline)
}

func (s squadScreen) updateEdit(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if isSaveKey(message) {
		s = s.save(m)
		m.squad = s
		return m, nil
	}
	switch message.String() {
	case "esc":
		m.squad = s
		if s.dirty() {
			return m.ask(i18n.SquadDiscard, screenSquads,
				guardSubject{Kind: guardsTheSquadInHand}), nil
		}
		s.mode = squadList
		m.squad = s.refresh(m.lib)
		return m, nil
	case "tab":
		s.editingID = !s.editingID && s.fresh()
		s = s.focus()
	case "up":
		s.units = clamp(s.units-1, 0, len(s.editing.Units))
	case "down":
		s.units = clamp(s.units+1, 0, len(s.editing.Units))
	case "enter":
		if s.units >= len(s.editing.Units) {
			return s.addUnit(m)
		}
		s = s.editUnit(s.units)
	case "ctrl+x":
		if s.units < len(s.editing.Units) {
			s.editing.Units = append(s.editing.Units[:s.units], s.editing.Units[s.units+1:]...)
			s.units = clamp(s.units, 0, len(s.editing.Units))
			s.err, s.notes = nil, nil
		}
	default:
		field := &s.nameInput
		if s.editingID {
			field = &s.idInput
		}
		updated, command := field.Update(message)
		if updated.Value() != field.Value() {
			s.err, s.notes = nil, nil
		}
		*field = updated
		s.editing.ID = strings.TrimSpace(s.idInput.Value())
		s.editing.Name = strings.TrimSpace(s.nameInput.Value())
		m.squad = s
		return m, command
	}
	m.squad = s
	return m, nil
}

// fresh reports whether the squad in hand is one that has never been saved,
// which is the only time its id may still be typed.
func (s squadScreen) fresh() bool {
	for _, saved := range s.saved {
		if saved.ID == s.editing.ID && s.editing.ID != "" {
			return false
		}
	}
	return true
}

func (s squadScreen) focus() squadScreen {
	if s.editingID {
		s.idInput.Focus()
		s.nameInput.Blur()
		return s
	}
	s.idInput.Blur()
	s.nameInput.Focus()
	return s
}

// addUnit puts the first character the builder offers in the first free slot,
// which is a unit an author then edits rather than a blank one they have to fill
// from nothing.
//
// It asks for the offered list with nothing held, because a member being added
// has chosen nobody yet: a hidden character is never what a new member starts
// as. A cast that is hidden all the way down therefore adds nobody, the way an
// empty one does.
func (s squadScreen) addUnit(m model) (tea.Model, tea.Cmd) {
	offered := offeredCharacters(s.characters, "")
	if len(s.editing.Units) >= hex.MaxTeamSize || len(offered) == 0 {
		m.squad = s
		return m, nil
	}
	character := offered[0]
	unit := placement.Placement{
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      s.freeSlot(-1),
	}
	unit.ID = s.freeID(character.ID, -1)
	s.editing.Units = append(s.editing.Units, unit)
	s = s.editUnit(len(s.editing.Units) - 1)
	m.squad = s
	return m, nil
}

// freeSlot is the first formation cell nobody else in the squad stands on.
func (s squadScreen) freeSlot(except int) hex.Offset {
	taken := map[hex.Offset]bool{}
	for i, unit := range s.editing.Units {
		if i != except {
			taken[unit.Slot] = true
		}
	}
	for _, slot := range formationSlots() {
		if !taken[slot] {
			return slot
		}
	}
	return hex.Offset{}
}

// freeID is a unit id nobody in the squad answers to, built from the
// character's own name so a log reads as the cast rather than as unit1, unit2.
func (s squadScreen) freeID(character string, except int) string {
	base := character
	if cut := strings.LastIndex(base, "."); cut >= 0 {
		base = base[cut+1:]
	}
	taken := map[string]bool{}
	for i, unit := range s.editing.Units {
		if i != except {
			taken[unit.ID] = true
		}
	}
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := base + strconv.Itoa(n)
		if !taken[candidate] {
			return candidate
		}
	}
}

// formationSlots is every cell of one side's formation, front column first: a
// squad is read from the rank that meets the enemy backwards, because that is
// the order an attack arrives in.
func formationSlots() []hex.Offset {
	out := make([]hex.Offset, 0, hex.FormationCols*hex.FormationRows)
	for _, col := range formationColumns() {
		for row := 0; row < hex.FormationRows; row++ {
			out = append(out, hex.Offset{Col: col, Row: row})
		}
	}
	return out
}

// formationColumns is the authoring columns in the order a rank is met: the
// entry at index 0 is the column that stands in front of the rest.
//
// It is read off hex.Ranks rather than counted down from FormationCols, for the
// reason the shape chooser draws its cells from pattern.Targets: an authoring
// column is not a depth. The ally half counts down from its own frontline and
// the enemy half counts up, so a drawing that decided which end was the front by
// looking at the column number would be right for whichever side whoever wrote
// it had in mind — which is the mistake hex.Ranks exists to have made once.
func formationColumns() []int {
	out := make([]int, hex.FormationCols)
	for col := range hex.FormationCols {
		if depth := rankOf(hex.Offset{Col: col}); depth >= 0 {
			out[depth] = col
		}
	}
	return out
}

// rankOf is how deep into its own half a formation cell stands, counted from the
// rank that meets the enemy first, or -1 for a cell that is on no formation.
//
// A squad carries no side, so the question is asked of one and the answer holds
// for both: hex.Place maps an enemy formation through a 180 degree rotation, so
// a cell authored at a given depth is at that depth whichever half it is fielded
// as. TestARankIsTheSameDepthOnEitherSide is what says so.
func rankOf(slot hex.Offset) int {
	placed := hex.Place(hex.SideAlly, slot)
	for depth, rank := range hex.Ranks(hex.SideAlly) {
		for _, cell := range rank {
			if cell == placed {
				return depth
			}
		}
	}
	return -1
}

// rankNames is one wording per rank, front first. It is an array of exactly the
// ranks there are, so a board given a fourth column is a compile error here
// rather than a fourth rank quietly drawing no name.
var rankNames = [hex.FormationCols]i18n.Key{
	i18n.SquadRankFront, i18n.SquadRankMiddle, i18n.SquadRankBack,
}

// rankLabel is what to call the rank a cell stands in, and empty for a cell that
// is on no formation — a slot a hand-edited file put off the grid draws its
// coordinate alone rather than borrowing a rank it is not in.
func (m model) rankLabel(slot hex.Offset) string {
	depth := rankOf(slot)
	if depth < 0 || depth >= len(rankNames) {
		return ""
	}
	return m.text(rankNames[depth])
}

func (s squadScreen) editUnit(index int) squadScreen {
	s.mode = squadUnit
	s.unitIndex = index
	// The cursor on the squad behind follows, so coming back out lands on the
	// member that was open rather than wherever it was before.
	s.units = index
	s.unit = s.editing.Units[index].Clone()
	s.unitOpenedAs = s.unit.Character
	s.field = unitCharacter
	s.levelInput.SetValue(strconv.Itoa(s.unit.Level))
	s.levelInput.Blur()
	s.err, s.notes = nil, nil
	return s
}

func (s squadScreen) updateUnit(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		s = s.commit()
		s.mode = squadEdit
		s = s.focus()
		m.squad = s
		return m, nil
	case "up":
		s = s.moveField(-1)
	case "down":
		s = s.moveField(1)
	case "left":
		s = s.cycle(-1)
	case "right":
		s = s.cycle(1)
	case "enter":
		switch s.field {
		case unitSkills:
			m.squad = s.commit()
			return m.openSquadSkills(), nil
		case unitPassives:
			m.squad = s.commit()
			return m.openSquadPassives(), nil
		}
	default:
		if s.field == unitLevel && numberKey(message) {
			updated, command := s.levelInput.Update(message)
			s.levelInput = updated
			if level, err := strconv.Atoi(strings.TrimSpace(updated.Value())); err == nil {
				s.unit.Level = level
				s = s.settleStage()
			}
			s.err = nil
			m.squad = s
			return m, command
		}
	}
	m.squad = s
	return m, nil
}

func (s squadScreen) moveField(by int) squadScreen {
	s.field = (s.field + by + unitFieldCount) % unitFieldCount
	if s.field == unitLevel {
		s.levelInput.Focus()
		return s
	}
	s.levelInput.Blur()
	return s
}

// commit writes the unit under edit back into the squad.
func (s squadScreen) commit() squadScreen {
	if s.unitIndex >= 0 && s.unitIndex < len(s.editing.Units) {
		s.editing.Units[s.unitIndex] = s.unit.Clone()
	}
	return s
}

func (s squadScreen) cycle(by int) squadScreen {
	switch s.field {
	case unitCharacter:
		// The character this member was opened with is on this list whether or
		// not it is held back, so stepping off it is a decision an author took
		// rather than one the filter took for them — and stepping back onto it
		// works, because the list does not move while it is being walked.
		offered := offeredCharacters(s.characters, s.unitOpenedAs)
		if len(offered) == 0 {
			return s
		}
		at := 0
		for i, character := range offered {
			if character.ID == s.unit.Character {
				at = i
			}
		}
		chosen := offered[(at+by+len(offered))%len(offered)]
		// A kit belongs to the character it was chosen from, so changing the
		// character empties it rather than carrying names the new one has never
		// heard of into a refusal at save time.
		s.unit.Character = chosen.ID
		s.unit.Skills, s.unit.Passives = nil, nil
		s.unit.ID = s.freeID(chosen.ID, s.unitIndex)
		s = s.settleStage()
	case unitSlot:
		slots := formationSlots()
		at := 0
		for i, slot := range slots {
			if slot == s.unit.Slot {
				at = i
			}
		}
		// Cells somebody else stands on are stepped over rather than refused:
		// two units on one cell is not a squad, and a chooser that stops on one
		// is a chooser that can be left in a state the save has to reject.
		for n := 1; n <= len(slots); n++ {
			candidate := slots[(at+by*n+len(slots)*len(slots))%len(slots)]
			if s.occupant(candidate) < 0 {
				s.unit.Slot = candidate
				break
			}
		}
	case unitStage:
		forms := s.stageChoices()
		at := 0
		for i, form := range forms {
			if form == s.unit.Stage {
				at = i
			}
		}
		s.unit.Stage = forms[(at+by+len(forms))%len(forms)]
		s.unit.Skills, s.unit.Passives = nil, nil
	default:
		return s
	}
	s.err = nil
	return s
}

// occupant is who else in the squad stands on a cell, or -1.
func (s squadScreen) occupant(slot hex.Offset) int {
	for i, unit := range s.editing.Units {
		if i != s.unitIndex && unit.Slot == slot {
			return i
		}
	}
	return -1
}

// stageChoices is the forms this unit may be fielded as: the empty string, which
// is the furthest the level reaches, and then every form by name.
//
// The empty option is offered first because it is what a placement means when it
// says nothing — and it is deliberately not the only option, because a line that
// forks has no furthest and the arms have to be nameable.
func (s squadScreen) stageChoices() []string {
	out := []string{""}
	character, known := s.character()
	if !known {
		return out
	}
	stages, err := character.StagesAt(s.unit.Level)
	if err != nil {
		return out
	}
	return append(out, progression.StageNames(stages)...)
}

// settleStage drops a chosen form the new character or level no longer offers,
// so the chooser never shows a name that is not on the list under it.
func (s squadScreen) settleStage() squadScreen {
	if s.unit.Stage == "" {
		return s
	}
	for _, form := range s.stageChoices() {
		if form == s.unit.Stage {
			return s
		}
	}
	s.unit.Stage = ""
	return s
}

// holdsAHiddenCharacter is whether the member under edit names a character the
// cast has held back.
//
// A member naming nobody the book knows is not this: that is a squad pointing at
// a deleted character, which is a different fault with a different answer, and
// saying "held back" about it would be a guess.
func (s squadScreen) holdsAHiddenCharacter() bool {
	character, known := s.character()
	return known && character.Hidden
}

func (s squadScreen) character() (cast.Character, bool) {
	for _, character := range s.characters {
		if character.ID == s.unit.Character {
			return character, true
		}
	}
	return cast.Character{}, false
}

// form is the stage name the unit's learnset is read against: the one chosen, or
// the furthest the level reaches when nothing is.
func (s squadScreen) form() string {
	if s.unit.Stage != "" {
		return s.unit.Stage
	}
	character, known := s.character()
	if !known {
		return ""
	}
	_, stage, err := character.Resolve(s.unit.Level, progression.Furthest)
	if err != nil {
		return ""
	}
	return stage.Name
}

// openSquadSkills and openSquadPassives are two pickers over one rule: what this
// character knows at this level as this form. They are separate methods rather
// than one taking a field because each writes a different half of the loadout
// back, and a picker that decided which half it was would be a switch inside a
// callback.
//
// Each names its own hint for the reason the allowlists do: the picker's default
// is the form's, and the form is choosing out of the whole skill book, where a
// row may carry a refusal. Here the options are the learnset already, so a mark
// for what cannot be taken names something no row can ever draw — and on the
// trait list an order says nothing either, since there is one slot to fill.
//
// These two are also the only pickers that carry slots, and they carry the
// constants the write reads rather than numbers of their own. The cap stops an
// over-full answer being built; refuse below and the save still ask the rule
// through cast.ChooseFrom, because the cap is a courtesy on top of the rule and
// never a replacement for it — a loadout hand-edited into squads.json never came
// through a keystroke at all.
func (m model) openSquadSkills() model {
	s := m.squad
	character, known := s.character()
	if !known {
		return m
	}
	return m.pick(&pickState{
		Title:   i18n.SquadPickSkills,
		Hint:    i18n.SquadKitHint,
		Kind:    pickSkills,
		Slots:   cast.SkillSlots,
		Options: squadOptions(character.SkillsAt(s.unit.Level, s.form())),
		Chosen:  append([]string(nil), s.unit.Skills...),
		Into:    pickIntoSquadKit,
	})
}

func (m model) openSquadPassives() model {
	s := m.squad
	character, known := s.character()
	if !known {
		return m
	}
	return m.pick(&pickState{
		Title:   i18n.SquadPickPassives,
		Hint:    i18n.SquadTraitHint,
		Kind:    pickPassives,
		Slots:   cast.TraitSlots,
		Options: squadOptions(character.PassivesAt(s.unit.Level, s.form())),
		Chosen:  append([]string(nil), s.unit.Passives...),
		Into:    pickIntoSquadTrait,
	})
}

// Picked is what the builder's two lists write back: one half of the member's
// loadout each, judged by the rule the write uses and committed to the squad in
// hand.
//
// ⚠️ **The character is looked up here rather than carried from the raise.** The
// two closures this replaces captured it at the moment the list opened, and the
// capture was doing nothing a lookup does not: the picker takes every key while
// it is up, so the member under edit cannot have changed character between the
// question and the answer — and the screen already knows how to find its own,
// because every other row on it is read the same way.
//
// ⚠️ **The destination arrives as an `any`** — see formScreen.Picked for why
// there are two destination vocabularies to tell apart now.
func (s squadScreen) Picked(_ draw.Context, into any, answer pickAnswer) (squadScreen, draw.Action) {
	character, known := s.character()
	if !known {
		// The raise declines on the same reading, so this is unreachable from a
		// keystroke; it is here because the learnset cannot be read without one.
		return s, draw.Action{}
	}
	switch into {
	case pickIntoSquadKit:
		s.unit.Skills = answer.Chosen
		s.err = s.refuse(cast.SkillSlots, answer.Chosen, "skill",
			character.SkillsAt(s.unit.Level, s.form()), cast.Required)
	case pickIntoSquadTrait:
		s.unit.Passives = answer.Chosen
		s.err = s.refuse(cast.TraitSlots, answer.Chosen, "trait",
			character.PassivesAt(s.unit.Level, s.form()), cast.Optional)
	default:
		// A destination this screen does not own; see formScreen.Picked.
		return s, draw.Action{}
	}
	// Both stay on the member being edited, so both hand back the zero action.
	return s.commit(), draw.Action{}
}

// refuse is the loadout rule asked early, so an over-filled slot is a line under
// the picker rather than a surprise at the save.
//
// It is the same call the write makes — cast.ChooseFrom — because a builder that
// judged a choice for itself would be a second copy of the rule, and the copy
// would be the one an author trusted.
func (s squadScreen) refuse(slots int, chosen []string, kind string,
	available []string, insist bool) error {
	_, err := cast.ChooseFrom(fmt.Sprintf("unit %q", s.unit.ID), kind, chosen,
		available, slots, s.unit.Level, insist)
	return err
}

func squadOptions(ids []string) []pickOption {
	out := make([]pickOption, 0, len(ids))
	for _, id := range ids {
		out = append(out, pickOption{ID: id})
	}
	return out
}

// save writes the squad, and refuses exactly as a battle would.
func (s squadScreen) save(m model) squadScreen {
	s.editing.ID = strings.TrimSpace(s.idInput.Value())
	s.editing.Name = strings.TrimSpace(s.nameInput.Value())
	if err := m.lib.SaveSquad(s.editing); err != nil {
		s.err = err
		s.notes = nil
		return s
	}
	s.err = nil
	s.notes = []forge.Note{
		{Kind: forge.NoteWrote, ID: s.editing.ID, Path: m.lib.SquadsPath()},
		{Kind: forge.NoteRebuild},
	}
	s.baseline = s.editing.Clone()
	return s.refresh(m.lib)
}

func (s squadScreen) view(m model) (string, string) {
	switch s.mode {
	case squadEdit:
		return s.viewEdit(m), m.text(i18n.SquadEditFooter, saveKeyLabel())
	case squadUnit:
		return s.viewUnit(m), m.text(i18n.SquadUnitFooter)
	}
	return s.viewList(m), m.text(i18n.SquadsFooter)
}

func (s squadScreen) viewList(m model) string {
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.SquadsHeading)) + "  " +
		m.style.Dim.Render(m.text(i18n.SquadsSubtitle, len(s.saved), hex.MaxTeamSize)) + "\n\n")
	if len(s.saved) == 0 {
		out.WriteString(m.style.Dim.Render(m.text(i18n.SquadsEmpty)) + "\n")
		return strings.TrimRight(out.String(), "\n")
	}
	// The id column is measured over what is drawn, header included, because a
	// squad is named by whoever built it and nothing bounds how long that is.
	width := lipgloss.Width(m.text(i18n.SquadColumnID))
	for _, squad := range s.saved {
		if drawn := lipgloss.Width(squad.ID); drawn > width {
			width = drawn
		}
	}
	out.WriteString("  " + m.style.Dim.Render(pad(m.text(i18n.SquadColumnID), width)+" "+
		m.text(i18n.SquadColumnMembers)) + "\n")
	from, to := window(len(s.saved), s.cursor, squadRoom(m))
	for index := from; index < to; index++ {
		squad := s.saved[index]
		marker := "  "
		line := pad(squad.ID, width) + " " +
			m.text(i18n.SquadMemberCount, len(squad.Units))
		if squad.Name != "" {
			line += "  " + squad.Name
		}
		if index == s.cursor {
			marker = "> "
			line = m.style.Selected.Render(line)
		}
		out.WriteString(marker + clip(line, m.usableWidth()-2) + "\n")
	}
	if s.err != nil {
		out.WriteString("\n" + m.style.Bad.Render(m.lang.Error(s.err)) + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

// squadRoom is how many rows the listing has, measured the way every listing
// here measures: the window less the frame, the heading, the column row and
// whatever is drawn under the list.
func squadRoom(m model) int {
	const frame, heading, columns = 4, 2, 1
	room := m.height - frame - heading - columns - 2
	if room < 3 {
		return 3
	}
	return room
}

func (s squadScreen) viewEdit(m model) string {
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.SquadHeading)) + "\n\n")
	width := squadLabelWidth(m)
	id := s.idInput.View()
	if !s.fresh() {
		id = s.editing.ID
	}
	out.WriteString(m.labelAt(m.text(i18n.SquadFieldID), width, "%s", id))
	out.WriteString(m.labelAt(m.text(i18n.SquadFieldName), width, "%s", s.nameInput.View()))
	out.WriteString("\n")
	for index, unit := range s.editing.Units {
		marker := "  "
		line := s.unitLine(m, index, unit)
		if index == s.units {
			marker = "> "
			line = m.style.Selected.Render(line)
		}
		out.WriteString(marker + clip(line, m.usableWidth()-2) + "\n")
	}
	add := m.text(i18n.SquadAddMember)
	if len(s.editing.Units) >= hex.MaxTeamSize {
		add = m.text(i18n.SquadFull, hex.MaxTeamSize)
	}
	if s.units >= len(s.editing.Units) {
		out.WriteString("> " + m.style.Selected.Render(add) + "\n")
	} else {
		out.WriteString("  " + m.style.Dim.Render(add) + "\n")
	}
	out.WriteString("\n" + s.formation(m, -1))
	out.WriteString(s.report(m))
	return strings.TrimRight(out.String(), "\n")
}

// unitLine is one member of the squad as a row: who it is, how grown, where it
// stands and what it brings.
func (s squadScreen) unitLine(m model, index int, unit placement.Placement) string {
	form := unit.Stage
	if form == "" {
		form = m.text(i18n.SquadFurthest)
	}
	return fmt.Sprintf("%d %s %s@%d %s %s %s",
		index+1, unit.ID, unit.Character, unit.Level, form, unit.Slot,
		m.text(i18n.SquadLoadoutCount, len(unit.Skills), cast.SkillSlots,
			len(unit.Passives), cast.TraitSlots))
}

func (s squadScreen) viewUnit(m model) string {
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.SquadUnitHeading, s.unit.ID)) + "\n\n")
	width := squadLabelWidth(m)
	rows := []struct {
		label i18n.Key
		value string
	}{
		{i18n.SquadFieldCharacter, s.chooserValue(m, s.unit.Character)},
		{i18n.SquadFieldLevel, s.levelInput.View()},
		{i18n.SquadFieldStage, s.chooserValue(m, s.stageLabel(m))},
		{i18n.SquadFieldSlot, s.chooserValue(m, s.slotLabel(m))},
		{i18n.SquadFieldSkills, s.listValue(m, s.unit.Skills, cast.SkillSlots)},
		{i18n.SquadFieldPassives, s.listValue(m, s.unit.Passives, cast.TraitSlots)},
	}
	for field, row := range rows {
		marker := "  "
		value := row.value
		if field == s.field {
			marker = "> "
			value = m.style.Selected.Render(value)
		}
		out.WriteString(marker + m.labelAt(m.text(row.label), width-2, "%s", value))
	}
	// Drawn only while the member holds a character the cast has held back, which
	// is the one case offeredCharacters keeps something on the list that nothing
	// else offers. Without it the row is a character id like any other and the
	// flag reads as not working.
	if s.holdsAHiddenCharacter() {
		out.WriteString("  " + m.style.Dim.Render(m.text(i18n.SquadHeldBack)) + "\n")
	}
	out.WriteString("\n" + s.formation(m, s.unitIndex))
	out.WriteString(s.report(m))
	return strings.TrimRight(out.String(), "\n")
}

// chooserValue draws a value that is stepped through with the arrow keys, in the
// decoration the character form's choosers already use.
func (s squadScreen) chooserValue(m model, value string) string {
	return fmt.Sprintf("< %s >", value)
}

// slotLabel is where the unit stands, said twice: as the cell the data writes
// and as the rank a formation is imagined in.
//
// Both rather than the reading alone. The coordinate is what squads.json holds
// and what a roster beside it is authored in, so an author matching the screen
// against a file needs it — and the rank is the half that means something,
// because reach is counted in ranks and a coordinate does not say which end of
// the grid is met first.
func (s squadScreen) slotLabel(m model) string {
	rank := m.rankLabel(s.unit.Slot)
	if rank == "" {
		return s.unit.Slot.String()
	}
	return fmt.Sprintf("%s  %s", s.unit.Slot, rank)
}

func (s squadScreen) stageLabel(m model) string {
	if s.unit.Stage == "" {
		return m.text(i18n.SquadFurthest)
	}
	return s.unit.Stage
}

// listValue is a chosen list with how full its slots are, so an unfinished kit
// says so before the save does.
func (s squadScreen) listValue(m model, chosen []string, slots int) string {
	if len(chosen) == 0 {
		return m.style.Dim.Render(m.text(i18n.SquadNothingChosen, slots))
	}
	return fmt.Sprintf("%s %s", strings.Join(chosen, " "),
		m.style.Dim.Render(m.text(i18n.ChoicePosition, len(chosen), slots)))
}

// The grid's own measurements. A cell is three characters wide because that is
// what "[n]" is, and n is a member number of a squad that fields hex.MaxTeamSize
// — one digit — so nothing widens it. The marker under the front column is built
// from the same figure rather than written out, so the two cannot part company.
const (
	formationIndent = "    "
	formationCell   = 3
)

// formation draws the squad's own 3x3, front column first, with the unit under
// edit marked and the front rank called out under it.
//
// editing is the member the arrows are moving, or -1 where there is none. It is
// what makes the picture live: see unitsDrawn.
//
// ASCII and nothing else: the box-drawing and arrow glyphs that would make it
// prettier are East-Asian-Ambiguous, which lipgloss measures as one cell and
// half the terminals draw as two — so the grid would overlap whatever is beside
// it on exactly the machines the client is used on. That is why the front rank
// is marked with carets rather than with an arrow.
func (s squadScreen) formation(m model, editing int) string {
	var out strings.Builder
	out.WriteString(m.style.Dim.Render(m.text(i18n.SquadFormation)) + "\n")
	units := s.unitsDrawn(editing)
	columns := formationColumns()
	for row := 0; row < hex.FormationRows; row++ {
		line := formationIndent
		for _, col := range columns {
			slot := hex.Offset{Col: col, Row: row}
			cell := " . "
			for index, unit := range units {
				if unit.Slot != slot {
					continue
				}
				cell = fmt.Sprintf("[%d]", index+1)
				if index == editing {
					cell = fmt.Sprintf("(%d)", index+1)
				}
			}
			line += cell + " "
		}
		out.WriteString(line + "\n")
	}
	// The marker sits under the first column drawn, which is the front rank by
	// construction: formationColumns orders them by depth.
	out.WriteString(m.style.Dim.Render(formationIndent+strings.Repeat("^", formationCell)+
		" "+m.text(i18n.SquadFormationFront)) + "\n")
	out.WriteString(m.style.Dim.Render(formationIndent+m.text(i18n.SquadFormationLegend)) + "\n")
	return out.String()
}

// unitsDrawn is the squad as the formation should picture it: every committed
// member, with the one under edit replaced by the copy the arrows are moving.
//
// This is the whole of what makes the grid live. ←/→ steps s.unit and commit()
// is what writes that back, and commit() runs only when the member is left or a
// picker is opened — so a drawing off the committed list alone stays on the old
// cell for the entire time the cell is being chosen, which is exactly when the
// picture is being looked at.
//
// It is not fixed by committing on every keypress either. s.editing.Units is
// shared with every model copied off this one, so a write from inside a drawing
// reaches all of them — which is what a value receiver looks like it prevents
// and does not. Reading the unit under edit costs nothing and writes nothing.
//
// The guard on leaving no longer rides on that: it compares the squad in hand
// against the one last written (see dirty), so a cursor that passed over a cell
// and came back leaves it down by arithmetic rather than by nobody having
// called commit().
//
// editing is -1 from the squad view, where there is no member under edit and the
// committed list is the whole truth. The substitution replaces rather than
// appends, so the cell a member steps off empties in the same draw the cell it
// steps onto fills — and the copy is into a new slice, because s.editing.Units
// is shared with every model copied off this one and a write into it would reach
// them all from inside a drawing.
func (s squadScreen) unitsDrawn(editing int) []placement.Placement {
	if editing < 0 || editing >= len(s.editing.Units) {
		return s.editing.Units
	}
	out := append([]placement.Placement(nil), s.editing.Units...)
	out[editing] = s.unit
	return out
}

// report is the refusal or the confirmation under whichever view is in front.
func (s squadScreen) report(m model) string {
	if s.err != nil {
		return "\n" + m.style.Bad.Render(m.text(i18n.WriteRefused, m.lang.Error(s.err))) + "\n"
	}
	if len(s.notes) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	for index, line := range m.lang.Notes(s.notes) {
		style := m.style.Dim
		if index == 0 {
			style = m.style.Good
		}
		out.WriteString(style.Render(line) + "\n")
	}
	return out.String()
}

// squadLabelWidth is measured rather than fixed, like every other label column
// in this program: the two languages word these differently and a constant would
// be right for one of them.
func squadLabelWidth(m model) int {
	width := 0
	for _, key := range []i18n.Key{
		i18n.SquadFieldID, i18n.SquadFieldName, i18n.SquadFieldCharacter,
		i18n.SquadFieldLevel, i18n.SquadFieldStage, i18n.SquadFieldSlot,
		i18n.SquadFieldSkills, i18n.SquadFieldPassives,
	} {
		if drawn := lipgloss.Width(m.text(key)); drawn > width {
			width = drawn
		}
	}
	return width + 3
}
