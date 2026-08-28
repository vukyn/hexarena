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

	// characters is the cast to choose from, held rather than looked up per
	// keystroke because cycling walks it.
	characters []cast.Character

	// editing is the squad in hand. It is a whole squad rather than an index
	// into saved, because a squad being built has not been saved yet and an
	// index would have nothing to point at.
	editing placement.Squad
	// unsaved is true while editing holds something the file does not, which is
	// what the guard on leaving asks about.
	unsaved bool
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
	unit       placement.Placement
	unitIndex  int
	field      int
	levelInput textinput.Model

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
		// The fight is raised from here because this is where a squad is under a
		// cursor. It reads that cursor rather than being handed a copy: a second
		// copy is a second thing to keep in step.
		if len(s.saved) > 0 {
			m.squad = s
			return m.enter(screenFight), nil
		}
	case "d":
		if len(s.saved) > 0 {
			id := s.saved[clamp(s.cursor, 0, len(s.saved)-1)].ID
			m.squad = s
			return m.ask(i18n.SquadDiscardSaved, func(m model) model {
				squad := m.squad
				if err := m.lib.DeleteSquad(id); err != nil {
					squad.err = err
				} else {
					squad.err = nil
					squad.notes = nil
					squad = squad.refresh(m.lib)
				}
				m.squad = squad
				return m
			}), nil
		}
	}
	m.squad = s
	return m, nil
}

// begin starts a squad nobody has written yet.
func (s squadScreen) begin() squadScreen {
	s.mode = squadEdit
	s.editing = placement.Squad{}
	s.unsaved = false
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
	s.unsaved = false
	s.units = 0
	s.editingID = false
	s.err, s.notes = nil, nil
	s.idInput.SetValue(squad.ID)
	s.nameInput.SetValue(squad.Name)
	s.idInput.Blur()
	s.nameInput.Focus()
	return s
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
		if s.unsaved {
			return m.ask(i18n.SquadDiscard, func(m model) model {
				squad := m.squad
				squad.mode = squadList
				squad.unsaved = false
				m.squad = squad.refresh(m.lib)
				return m
			}), nil
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
			s.unsaved = true
			s.err, s.notes = nil, nil
		}
	default:
		field := &s.nameInput
		if s.editingID {
			field = &s.idInput
		}
		updated, command := field.Update(message)
		if updated.Value() != field.Value() {
			s.unsaved = true
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

// addUnit puts the first character of the cast in the first free slot, which is
// a unit an author then edits rather than a blank one they have to fill from
// nothing.
func (s squadScreen) addUnit(m model) (tea.Model, tea.Cmd) {
	if len(s.editing.Units) >= hex.MaxTeamSize || len(s.characters) == 0 {
		m.squad = s
		return m, nil
	}
	character := s.characters[0]
	unit := placement.Placement{
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      s.freeSlot(-1),
	}
	unit.ID = s.freeID(character.ID, -1)
	s.editing.Units = append(s.editing.Units, unit)
	s.unsaved = true
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
	for col := hex.FormationCols - 1; col >= 0; col-- {
		for row := 0; row < hex.FormationRows; row++ {
			out = append(out, hex.Offset{Col: col, Row: row})
		}
	}
	return out
}

func (s squadScreen) editUnit(index int) squadScreen {
	s.mode = squadUnit
	s.unitIndex = index
	// The cursor on the squad behind follows, so coming back out lands on the
	// member that was open rather than wherever it was before.
	s.units = index
	s.unit = s.editing.Units[index].Clone()
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
				s.unsaved = true
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
		s.unsaved = true
	}
	return s
}

func (s squadScreen) cycle(by int) squadScreen {
	switch s.field {
	case unitCharacter:
		if len(s.characters) == 0 {
			return s
		}
		at := 0
		for i, character := range s.characters {
			if character.ID == s.unit.Character {
				at = i
			}
		}
		chosen := s.characters[(at+by+len(s.characters))%len(s.characters)]
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
	s.unsaved = true
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
func (m model) openSquadSkills() model {
	s := m.squad
	character, known := s.character()
	if !known {
		return m
	}
	return m.pick(&pickState{
		title:   i18n.SquadPickSkills,
		kind:    pickSkills,
		options: squadOptions(character.SkillsAt(s.unit.Level, s.form())),
		chosen:  append([]string(nil), s.unit.Skills...),
		apply: func(m model, answer pickAnswer) model {
			squad := m.squad
			squad.unit.Skills = answer.Chosen
			squad.err = squad.refuse(cast.SkillSlots, answer.Chosen, "skill",
				character.SkillsAt(squad.unit.Level, squad.form()), cast.Required)
			squad.unsaved = true
			m.squad = squad.commit()
			return m
		},
	})
}

func (m model) openSquadPassives() model {
	s := m.squad
	character, known := s.character()
	if !known {
		return m
	}
	return m.pick(&pickState{
		title:   i18n.SquadPickPassives,
		kind:    pickSkills,
		options: squadOptions(character.PassivesAt(s.unit.Level, s.form())),
		chosen:  append([]string(nil), s.unit.Passives...),
		apply: func(m model, answer pickAnswer) model {
			squad := m.squad
			squad.unit.Passives = answer.Chosen
			squad.err = squad.refuse(cast.TraitSlots, answer.Chosen, "trait",
				character.PassivesAt(squad.unit.Level, squad.form()), cast.Optional)
			squad.unsaved = true
			m.squad = squad.commit()
			return m
		},
	})
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
		out = append(out, pickOption{id: id})
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
	s.unsaved = false
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
	out.WriteString(m.style.heading.Render(m.text(i18n.SquadsHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.SquadsSubtitle, len(s.saved), hex.MaxTeamSize)) + "\n\n")
	if len(s.saved) == 0 {
		out.WriteString(m.style.dim.Render(m.text(i18n.SquadsEmpty)) + "\n")
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
	out.WriteString("  " + m.style.dim.Render(pad(m.text(i18n.SquadColumnID), width)+" "+
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
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + clip(line, m.usableWidth()-2) + "\n")
	}
	if s.err != nil {
		out.WriteString("\n" + m.style.bad.Render(m.lang.Error(s.err)) + "\n")
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
	out.WriteString(m.style.heading.Render(m.text(i18n.SquadHeading)) + "\n\n")
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
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + clip(line, m.usableWidth()-2) + "\n")
	}
	add := m.text(i18n.SquadAddMember)
	if len(s.editing.Units) >= hex.MaxTeamSize {
		add = m.text(i18n.SquadFull, hex.MaxTeamSize)
	}
	if s.units >= len(s.editing.Units) {
		out.WriteString("> " + m.style.selected.Render(add) + "\n")
	} else {
		out.WriteString("  " + m.style.dim.Render(add) + "\n")
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
	out.WriteString(m.style.heading.Render(m.text(i18n.SquadUnitHeading, s.unit.ID)) + "\n\n")
	width := squadLabelWidth(m)
	rows := []struct {
		label i18n.Key
		value string
	}{
		{i18n.SquadFieldCharacter, s.chooserValue(m, s.unit.Character)},
		{i18n.SquadFieldLevel, s.levelInput.View()},
		{i18n.SquadFieldStage, s.chooserValue(m, s.stageLabel(m))},
		{i18n.SquadFieldSlot, s.chooserValue(m, s.unit.Slot.String())},
		{i18n.SquadFieldSkills, s.listValue(m, s.unit.Skills, cast.SkillSlots)},
		{i18n.SquadFieldPassives, s.listValue(m, s.unit.Passives, cast.TraitSlots)},
	}
	for field, row := range rows {
		marker := "  "
		value := row.value
		if field == s.field {
			marker = "> "
			value = m.style.selected.Render(value)
		}
		out.WriteString(marker + m.labelAt(m.text(row.label), width-2, "%s", value))
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
		return m.style.dim.Render(m.text(i18n.SquadNothingChosen, slots))
	}
	return fmt.Sprintf("%s %s", strings.Join(chosen, " "),
		m.style.dim.Render(m.text(i18n.ChoicePosition, len(chosen), slots)))
}

// formation draws the squad's own 3x3, front column first, with the unit under
// edit marked.
//
// ASCII and nothing else: the box-drawing and arrow glyphs that would make it
// prettier are East-Asian-Ambiguous, which lipgloss measures as one cell and
// half the terminals draw as two — so the grid would overlap whatever is beside
// it on exactly the machines the client is used on.
func (s squadScreen) formation(m model, editing int) string {
	var out strings.Builder
	out.WriteString(m.style.dim.Render(m.text(i18n.SquadFormation)) + "\n")
	for row := 0; row < hex.FormationRows; row++ {
		line := "    "
		for col := hex.FormationCols - 1; col >= 0; col-- {
			slot := hex.Offset{Col: col, Row: row}
			cell := " . "
			for index, unit := range s.editing.Units {
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
	out.WriteString(m.style.dim.Render("    "+m.text(i18n.SquadFormationLegend)) + "\n")
	return out.String()
}

// report is the refusal or the confirmation under whichever view is in front.
func (s squadScreen) report(m model) string {
	if s.err != nil {
		return "\n" + m.style.bad.Render(m.text(i18n.WriteRefused, m.lang.Error(s.err))) + "\n"
	}
	if len(s.notes) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	for index, line := range m.lang.Notes(s.notes) {
		style := m.style.dim
		if index == 0 {
			style = m.style.good
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
