package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the new-skill form, in the order they are walked.
//
// Nine core fields, the statuses it inflicts, and the three allowlists. What is
// deliberately absent is requires, strips, scaling and self_applies: each is a
// composite worth several questions of its own, and a form that asked twelve
// more would be worse than an edit to skills.json. They survive a save
// untouched — see skill.Skill.MarshalJSON.
const (
	skillFieldID = iota
	skillFieldElement
	skillFieldTarget
	skillFieldRange
	skillFieldShape
	skillFieldPower
	skillFieldStrikes
	skillFieldAccuracy
	skillFieldCooldown
	skillFieldInflicts
	skillFieldKeptForElements
	skillFieldKeptForRoles
	skillFieldKeptForCharacters
	skillFieldCount
)

// skillsScreen is the skill book, and the form that adds to it and changes it.
//
// It is the other half of the new-character form in the same way the origins
// screen is: a kit can only name a skill the book holds, so an author who finds
// the skill they want missing can add it and go straight back.
//
// Nothing here decides whether an answer is acceptable, and nothing here works
// out what a skill is worth. The damage comes from forge.Library.PreviewDraft,
// which is combat.Rules.Damage against the reference pair skills.golden's own
// table is measured from, and the write goes through forge.SkillDraft.Resolve,
// which appends to the book and therefore applies exactly the checks a load
// applies.
type skillsScreen struct {
	skills []skill.Skill
	cursor int

	// adding is whether the form is in front of the listing to author a new
	// skill, and editing is the id it is in front to change. They are two fields
	// rather than one because one form serves both jobs and the difference
	// decides three things: the heading, what Escape is asking to throw away, and
	// whether the write appends or replaces. formInFront is the two together, and
	// they are never both set.
	adding  bool
	editing string
	inputs  []textinput.Model
	field   int
	// The three choosers and the three allowlists, held as their answers rather
	// than as text fields: an element, a side and a shape are all ids out of a
	// book, so typing one is only a way to get it wrong.
	elementIndex int
	targetIndex  int
	shapeIndex   int
	keptElements []string
	keptRoles    []string
	keptWho      []string
	touched      bool

	err error
	// added is the last skill written, kept as what it was rather than as the
	// line announcing it, so a language switch redraws the announcement.
	added *skill.Skill
	// edited is the last change written, kept whole rather than as its id: the
	// damage before and after is what an author needs from an edit, and it is
	// carried so a language switch redraws that too.
	edited *forge.SkillChange
}

// formInFront reports whether the form is over the listing, either to author a
// skill or to change one.
func (s skillsScreen) formInFront() bool { return s.adding || s.editing != "" }

func newSkillsScreen(lib *forge.Library) skillsScreen {
	return skillsScreen{}.refresh(lib).resetForm(lib)
}

func (s skillsScreen) refresh(lib *forge.Library) skillsScreen {
	s.skills = lib.Skills().Skills()
	s.cursor = clamp(s.cursor, 0, len(s.skills)-1)
	if s.inputs == nil {
		s = s.resetForm(lib)
	}
	return s
}

func (s skillsScreen) resetForm(lib *forge.Library) skillsScreen {
	s.inputs = make([]textinput.Model, skillFieldCount)
	for i := range s.inputs {
		input := textinput.New()
		input.Prompt = ""
		input.CharLimit = 200
		input.Width = 24
		s.inputs[i] = input
	}
	s.inputs[skillFieldID].Width = 32
	s.inputs[skillFieldInflicts].Width = 40
	// The defaults are the shape of an ordinary single-target attack, and the
	// element among them is the one worth spelling out: neutral is the common
	// pool, so a skill authored without an opinion about its element is one
	// every character can take. Power and accuracy have none, because both are
	// balance and a default would write a number nobody chose.
	s.inputs[skillFieldRange].SetValue(defaultSkillRange)
	s.inputs[skillFieldStrikes].SetValue(defaultSkillStrikes)
	s.inputs[skillFieldCooldown].SetValue(defaultSkillCooldown)
	s.elementIndex = indexOf(forge.ElementNames(), defaultSkillElement)
	s.targetIndex = indexOf(forge.TargetNames(), defaultSkillTarget)
	s.shapeIndex = indexOf(lib.PatternNames(), defaultSkillPattern)
	s.keptElements, s.keptRoles, s.keptWho = nil, nil, nil
	s.field = skillFieldID
	s.touched = false
	s.err = nil
	s.editing = ""
	s.inputs[s.field].Focus()
	return s
}

// prefill is resetForm over a skill that already exists, which is what makes one
// form serve both jobs.
//
// Every value comes from forge.SkillAnswers rather than being formatted here, so
// that accepting the form as it stands reproduces the skill exactly. A screen
// that wrote its own "1200" or turned an absent restriction into an empty list
// would turn opening the form into a change.
func (s skillsScreen) prefill(lib *forge.Library, current skill.Skill) skillsScreen {
	s = s.resetForm(lib)
	answers := forge.SkillAnswers(current)
	for _, filled := range []struct {
		field int
		value string
	}{
		{skillFieldID, answers.ID},
		{skillFieldRange, answers.Range},
		{skillFieldPower, answers.Power},
		{skillFieldStrikes, answers.Strikes},
		{skillFieldAccuracy, answers.Accuracy},
		{skillFieldCooldown, answers.Cooldown},
		{skillFieldInflicts, answers.Applies},
	} {
		s.inputs[filled.field].SetValue(filled.value)
	}
	s.elementIndex = indexOf(forge.ElementNames(), answers.Element)
	s.targetIndex = indexOf(forge.TargetNames(), answers.Target)
	s.shapeIndex = indexOf(lib.PatternNames(), answers.Pattern)
	s.keptElements = forge.SplitList(answers.RestrictElements)
	s.keptRoles = forge.SplitList(answers.RestrictArchetypes)
	s.keptWho = forge.SplitList(answers.RestrictCharacters)
	s.editing = current.ID
	return s
}

// The defaults a skill takes when nobody says otherwise. They are the same
// strings cmd/hexforge's prompts default to; both front-ends offering different
// defaults would be two answers to one question.
const (
	defaultSkillElement  = "neutral"
	defaultSkillTarget   = "enemy"
	defaultSkillPattern  = "single"
	defaultSkillRange    = "1"
	defaultSkillStrikes  = "1"
	defaultSkillCooldown = "0"
)

func indexOf(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return 0
}

// draft is the answers as internal/forge wants them, which is the only thing
// this screen hands outwards.
func (s skillsScreen) draft(m model) forge.SkillDraft {
	return forge.SkillDraft{
		ID:                 strings.TrimSpace(s.inputs[skillFieldID].Value()),
		Element:            at(forge.ElementNames(), s.elementIndex),
		Target:             at(forge.TargetNames(), s.targetIndex),
		Range:              s.inputs[skillFieldRange].Value(),
		Pattern:            at(m.lib.PatternNames(), s.shapeIndex),
		Power:              s.inputs[skillFieldPower].Value(),
		Strikes:            s.inputs[skillFieldStrikes].Value(),
		Accuracy:           s.inputs[skillFieldAccuracy].Value(),
		Cooldown:           s.inputs[skillFieldCooldown].Value(),
		Applies:            s.inputs[skillFieldInflicts].Value(),
		RestrictElements:   strings.Join(s.keptElements, ","),
		RestrictArchetypes: strings.Join(s.keptRoles, ","),
		RestrictCharacters: strings.Join(s.keptWho, ","),
	}
}

func at(values []string, index int) string {
	if len(values) == 0 {
		return ""
	}
	return values[clamp(index, 0, len(values)-1)]
}

// chooserField reports whether a field is stepped through rather than typed.
func skillChooserField(field int) bool {
	switch field {
	case skillFieldElement, skillFieldTarget, skillFieldShape:
		return true
	default:
		return false
	}
}

// listField reports whether a field is a list chosen on the sub-screen.
func skillListField(field int) bool {
	switch field {
	case skillFieldKeptForElements, skillFieldKeptForRoles, skillFieldKeptForCharacters:
		return true
	default:
		return false
	}
}

// update routes a keystroke on the listing.
//
// e is the edit key. It does not collide with anything: this screen is a list
// rather than a form, so no field has the keyboard, and its only other letters
// are q, k, j and a. It sits beside a for the reason those two belong together —
// adding a skill and changing one are the same form reached two ways.
func (s skillsScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.formInFront() {
		return s.updateForm(m, message)
	}
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		s.cursor = clamp(s.cursor-1, 0, len(s.skills)-1)
	case "down", "j":
		s.cursor = clamp(s.cursor+1, 0, len(s.skills)-1)
	case "a":
		s = s.resetForm(m.lib)
		s.adding = true
		s.added, s.edited = nil, nil
	case "e":
		if len(s.skills) > 0 {
			s = s.prefill(m.lib, s.skills[clamp(s.cursor, 0, len(s.skills)-1)])
			s.added, s.edited = nil, nil
		}
	}
	m.skills = s
	return m, nil
}

func (s skillsScreen) updateForm(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		if !s.touched {
			s.adding, s.editing = false, ""
			m.skills = s
			return m, nil
		}
		// The question names what is being thrown away, which is a different
		// thing in each case: a skill nobody has written yet, or a set of changes
		// to one that is already in the book.
		question := i18n.SkillFormDiscard
		if s.editing != "" {
			question = i18n.SkillFormEditDiscard
		}
		return m.ask(question, func(m model) model {
			m.skills = m.skills.resetForm(m.lib)
			m.skills.adding = false
			return m
		}), nil
	case "ctrl+s":
		s = s.save(m)
		m.skills = s
		return m, nil
	case "up", "shift+tab":
		s = s.moveTo(s.field - 1)
		m.skills = s
		return m, nil
	case "down", "tab", "enter":
		s = s.moveTo(s.field + 1)
		m.skills = s
		return m, nil
	}
	if skillChooserField(s.field) {
		switch message.String() {
		case "left":
			s = s.cycle(m, -1)
		case "right":
			s = s.cycle(m, 1)
		}
		m.skills = s
		return m, nil
	}
	if skillListField(s.field) {
		if message.String() == " " || message.String() == "right" {
			m.skills = s
			return m.openAllowlist(s.field), nil
		}
		m.skills = s
		return m, nil
	}
	updated, command := s.inputs[s.field].Update(message)
	if updated.Value() != s.inputs[s.field].Value() {
		s.touched = true
		s.err = nil
		s.added, s.edited = nil, nil
	}
	s.inputs[s.field] = updated
	m.skills = s
	return m, command
}

// openAllowlist raises the picker for one of the three lists.
//
// The list is offered rather than typed for the reason the origin and the
// archetype are on the other form: every entry is an id out of a book, so a
// picker cannot produce a name that does not exist — and an allowlist naming
// somebody who does not exist is the same mistake as an empty one, satisfied by
// nobody.
func (m model) openAllowlist(field int) model {
	switch field {
	case skillFieldKeptForElements:
		return m.pick(&pickState{
			title: i18n.PickerElementsTitle, kind: pickElements,
			options: idOptions(forge.ElementNames()), chosen: m.skills.keptElements,
			apply: func(m model, chosen []string) model {
				m.skills.keptElements = chosen
				m.skills.touched = true
				return m
			},
		})
	case skillFieldKeptForRoles:
		return m.pick(&pickState{
			title: i18n.PickerRolesTitle, kind: pickArchetypes,
			options: idOptions(m.lib.Archetypes().IDs()), chosen: m.skills.keptRoles,
			apply: func(m model, chosen []string) model {
				m.skills.keptRoles = chosen
				m.skills.touched = true
				return m
			},
		})
	default:
		return m.pick(&pickState{
			title: i18n.PickerCharactersTitle, kind: pickCharacters,
			options: idOptions(m.lib.CharacterIDs()), chosen: m.skills.keptWho,
			apply: func(m model, chosen []string) model {
				m.skills.keptWho = chosen
				m.skills.touched = true
				return m
			},
		})
	}
}

func (s skillsScreen) moveTo(target int) skillsScreen {
	s.inputs[s.field].Blur()
	s.field = (target + skillFieldCount) % skillFieldCount
	if !skillChooserField(s.field) && !skillListField(s.field) {
		s.inputs[s.field].Focus()
	}
	return s
}

func (s skillsScreen) cycle(m model, by int) skillsScreen {
	step := func(index int, total int) int {
		if total == 0 {
			return 0
		}
		return (index + by + total) % total
	}
	switch s.field {
	case skillFieldElement:
		s.elementIndex = step(s.elementIndex, len(forge.ElementNames()))
	case skillFieldTarget:
		s.targetIndex = step(s.targetIndex, len(forge.TargetNames()))
	case skillFieldShape:
		s.shapeIndex = step(s.shapeIndex, len(m.lib.PatternNames()))
	}
	s.touched = true
	s.err = nil
	s.added = nil
	return s
}

// save resolves the draft and writes it, as an addition or as a change to the
// skill the form was opened on.
//
// Every half belongs to internal/forge: Resolve and ResolveEdit each refuse a
// skill a load would refuse, SaveSkill and EditSkill each write through the
// temp-file-then-rename that keeps a crash from truncating skills.json, and the
// second of those refuses an edit no character or preset could survive. Nothing
// on this screen decides which of those is true.
func (s skillsScreen) save(m model) skillsScreen {
	if s.editing != "" {
		return s.saveEdit(m)
	}
	built, err := s.draft(m).Resolve(m.lib)
	if err != nil {
		s.err = err
		return s
	}
	if err := m.lib.SaveSkill(built); err != nil {
		s.err = err
		return s
	}
	s = s.refresh(m.lib).resetForm(m.lib)
	s.adding = false
	s.added = &built
	return s
}

func (s skillsScreen) saveEdit(m model) skillsScreen {
	built, err := s.draft(m).ResolveEdit(m.lib, s.editing)
	if err != nil {
		s.err = err
		return s
	}
	change, err := m.lib.EditSkill(built)
	if err != nil {
		s.err = err
		return s
	}
	// resetForm clears editing, which is what closes the form: the listing behind
	// it is refreshed from the library the write went through, so the changed row
	// is the changed row.
	s = s.refresh(m.lib).resetForm(m.lib)
	s.adding = false
	s.edited = &change
	return s
}

// skillsRoom is how many rows the listing has, measured from the window in hand:
// the body has m.height-4 lines and this screen spends nine of them on a
// heading, a blank, a column header, a blank, the two damage rows, the two lines
// a write leaves behind, and the tally.
//
// The nine are enumerated because two of them are the busiest state rather than
// every state, and the reserve is for the busiest: the second damage row is only
// drawn for a skill with a condition, and the second write line only after an
// edit that moved the damage. A reserve that counted what is about to be drawn
// would be a reserve that changes under the author, which is worse than one row
// spent on a screen that does not need it.
//
// It stayed at nine when editing added the second write line, and the reason is
// worth recording rather than leaving as a coincidence: the number was one too
// high before. The count it came with listed "the empty string the body's
// trailing newline leaves", copied from pickerRoom, and this body has no trailing
// newline — the tally is written without one. So the real spend was eight against
// a reserve of nine, and the second write line is what that spare line has now
// gone on. There is no spare left, which is what
// TestTheSkillListingFitsTheSmallestWindowAfterAnEdit measures at the 80x24
// floor. A tenth line on this screen needs a tenth here.
func skillsRoom(m model) int {
	room := m.height - 4 - 9
	if room < 3 {
		return 3
	}
	return room
}

// skillRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart.
// skillRow lays out one line of the skill list.
//
// glossColumn of zero drops the translated-name column entirely rather than
// drawing it empty. That one rule covers both cases that need it: English,
// where nothing is glossed, and a book whose ids all happen to be unglossed —
// a column of blanks would read as missing data rather than as a column that
// does not apply.
func skillRow(idColumn, glossColumn int, id, gloss, member, power, who string) string {
	name := pad(id, idColumn)
	if glossColumn > 0 {
		name += " " + pad(gloss, glossColumn)
	}
	return fmt.Sprintf("%s %s %s%s", name, pad(member, 9), pad(power, 8), who)
}

func (s skillsScreen) view(m model) (string, string) {
	if s.formInFront() {
		return s.viewForm(m)
	}
	footer := m.text(i18n.SkillsFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.SkillsHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.SkillsSubtitle)) + "\n\n")

	column, glossColumn := 0, 0
	for _, current := range s.skills {
		if width := lipgloss.Width(current.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(m.lang.Gloss(current.ID)); width > glossColumn {
			glossColumn = width
		}
	}
	if glossColumn > 0 {
		// The header has to fit the column it names, and a newly authored skill
		// has no gloss, so this width is the widest of the two rather than of
		// the glosses alone.
		if width := lipgloss.Width(m.text(i18n.ColumnGloss)); width > glossColumn {
			glossColumn = width
		}
	}
	from, to := window(len(s.skills), s.cursor, skillsRoom(m))
	anyone := 0
	for _, current := range s.skills {
		if forge.AnyoneMayCarry(current) {
			anyone++
		}
	}
	// The header names the one column nobody could guess. The other three are
	// an id, an element and a power, each labelled with the word the form that
	// authored it uses.
	out.WriteString("  " + m.style.dim.Render(skillRow(column+1, glossColumn,
		m.text(i18n.SkillFieldID), m.text(i18n.ColumnGloss), m.text(i18n.LabelElement),
		m.text(i18n.SkillFieldPower), m.text(i18n.ColumnWhoMayCarry))) + "\n")
	for index := from; index < to; index++ {
		current := s.skills[index]
		marker := "  "
		// The power and the strike count are the balance, so they are the two
		// numbers on the row; everything else about a skill is a keypress away
		// on the form that authored it.
		row := skillRow(column+1, glossColumn, current.ID, m.lang.Gloss(current.ID),
			current.Element.String(),
			strconv.Itoa(current.Power)+"x"+strconv.Itoa(current.StrikeCount()), "")
		row += clip(m.lang.WhoMaySummary(current), minWidth-3-lipgloss.Width(row))
		if index == s.cursor {
			marker = "> "
			row = m.style.selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}
	out.WriteString("\n")
	// What the skill under the cursor is worth, against the same reference the
	// form previews an unwritten one against — so a power being authored can be
	// compared with the powers already in the book without leaving the program.
	if len(s.skills) > 0 {
		selected := s.skills[clamp(s.cursor, 0, len(s.skills)-1)]
		preview := m.lib.PreviewDamage(selected)
		out.WriteString(m.label(m.text(i18n.LabelDamage), "%s", m.lang.Damage(preview)))
		if preview.Amplified > 0 {
			out.WriteString(m.continued("%s",
				m.text(i18n.DamageAmplified, preview.Amplified)))
		}
	}
	if s.added != nil {
		out.WriteString(m.style.good.Render(m.text(i18n.SkillAdded,
			s.added.ID, m.lib.SkillsPath())) + "\n")
	}
	if s.edited != nil {
		out.WriteString(m.style.good.Render(m.text(i18n.SkillEdited,
			s.edited.After.ID, m.lib.SkillsPath())) + "\n")
		// The before and after, and only when there is something to compare: an
		// edit to a restriction or a targeting side moves no damage, and a line
		// saying a number did not change has to be read to learn nothing.
		if s.edited.MovesDamage() {
			out.WriteString(m.style.dim.Render(m.lang.DamageMoved(*s.edited)) + "\n")
		}
	}
	out.WriteString(m.style.dim.Render(m.text(i18n.SkillsTally, len(s.skills), anyone)))
	return out.String(), footer
}

// skillFieldLabel is what each row of the form is called.
func skillFieldLabel(m model, field int) string {
	keys := [skillFieldCount]i18n.Key{
		skillFieldID:                i18n.SkillFieldID,
		skillFieldElement:           i18n.SkillFieldElement,
		skillFieldTarget:            i18n.SkillFieldTarget,
		skillFieldRange:             i18n.SkillFieldRange,
		skillFieldShape:             i18n.SkillFieldShape,
		skillFieldPower:             i18n.SkillFieldPower,
		skillFieldStrikes:           i18n.SkillFieldStrikes,
		skillFieldAccuracy:          i18n.SkillFieldAccuracy,
		skillFieldCooldown:          i18n.SkillFieldCooldown,
		skillFieldInflicts:          i18n.SkillFieldInflicts,
		skillFieldKeptForElements:   i18n.SkillFieldKeptForElements,
		skillFieldKeptForRoles:      i18n.SkillFieldKeptForRoles,
		skillFieldKeptForCharacters: i18n.SkillFieldKeptForCharacters,
	}
	return m.text(keys[field])
}

// skillLabelWidth is the column the field names sit in, measured from the labels
// themselves rather than declared: the longest is "cooldown" in one language and
// "để dành cho mẫu" in the other.
func skillLabelWidth(m model) int {
	widest := 0
	for field := range skillFieldCount {
		if width := lipgloss.Width(skillFieldLabel(m, field)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

func (s skillsScreen) viewForm(m model) (string, string) {
	footer := m.text(i18n.SkillFormFooter)
	// The heading is the whole of what tells an author which of the two jobs this
	// form is doing, so it is not shared: every field is prefilled on an edit, and
	// a prefilled form under "new skill" reads as a form that has remembered the
	// last thing typed into it.
	heading := i18n.SkillFormHeading
	if s.editing != "" {
		heading = i18n.SkillFormEditHeading
	}
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(heading)) + "  " +
		m.style.dim.Render(m.text(i18n.SkillFormSubtitle)) + "\n\n")

	width := skillLabelWidth(m)
	for field := range skillFieldCount {
		marker := "  "
		if field == s.field {
			marker = "> "
		}
		name := pad(skillFieldLabel(m, field), width)
		if field == s.field {
			name = m.style.selected.Render(name)
		} else {
			name = m.style.label.Render(name)
		}
		out.WriteString(marker + name + " " + s.value(m, field, width) + "\n")
	}

	out.WriteString("\n")
	out.WriteString(s.damageRow(m, width))
	out.WriteString(m.labelAt("", width, "%s", m.style.dim.Render(m.text(i18n.SkillFormHint))))
	if s.err != nil {
		out.WriteString(m.style.bad.Render(m.text(i18n.WriteRefused, m.lang.Error(s.err))) + "\n")
	}
	return out.String(), footer
}

// value is what one row shows: a chooser, a chosen list, or what was typed.
func (s skillsScreen) value(m model, field, labelWidth int) string {
	choice := func(values []string, index int) string {
		if len(values) == 0 {
			return m.style.bad.Render(m.text(i18n.NoneCatalogued))
		}
		return fmt.Sprintf(choiceFormat, m.lang.Glossed(at(values, index)),
			m.style.dim.Render(m.text(i18n.ChoicePosition,
				clamp(index, 0, len(values)-1)+1, len(values))))
	}
	switch field {
	case skillFieldElement:
		return choice(forge.ElementNames(), s.elementIndex)
	case skillFieldTarget:
		return choice(forge.TargetNames(), s.targetIndex)
	case skillFieldShape:
		return choice(m.lib.PatternNames(), s.shapeIndex)
	case skillFieldKeptForElements:
		return s.listValue(m, s.keptElements, labelWidth)
	case skillFieldKeptForRoles:
		return s.listValue(m, s.keptRoles, labelWidth)
	case skillFieldKeptForCharacters:
		return s.listValue(m, s.keptWho, labelWidth)
	case skillFieldAccuracy, skillFieldPower:
		// Both are authored in parts per thousand because that is what the
		// engine multiplies and divides by, but nobody reads 850 as a chance or
		// 2200 as "twice over". The percentage sits beside the field rather than
		// replacing it: the number written to the file is still the number on
		// screen.
		return s.inputs[field].View() + s.percentHint(m, field)
	case skillFieldInflicts:
		// The chances in this field are parts per thousand too, but the field
		// holds a whole list in the syntax ParseApplications reads, so the
		// reading goes beside it rather than into it.
		return s.inputs[field].View() + s.chanceHint(m, labelWidth)
	default:
		return s.inputs[field].View()
	}
}

// chanceHint reads out the chances in the inflicts field. A list being typed is
// unparseable most of the time, and that is not an error to announce, so it says
// nothing until the whole list parses.
func (s skillsScreen) chanceHint(m model, labelWidth int) string {
	typed := strings.TrimSpace(s.inputs[skillFieldInflicts].Value())
	if typed == "" {
		return ""
	}
	applications, err := m.lib.ParseApplications(typed)
	if err != nil || len(applications) == 0 {
		return ""
	}
	// The field itself is a fixed width, so the row's only unbounded part is
	// this reading: a skill may apply any number of statuses, and five of them
	// would push the row past the floor. Clipping the reading is right where
	// clipping the value would not be — a chance you cannot see is still
	// written in the field beside it.
	room := minWidth - 3 - labelWidth - s.inputs[skillFieldInflicts].Width - 2
	return "  " + m.style.dim.Render(clip(forge.ApplicationChances(applications), room))
}

// percentHint is the dim reading of a parts-per-thousand field, or nothing at
// all while the field does not hold one. A half-typed number is the normal
// state of a text field, so it is not an error to say nothing about.
func (s skillsScreen) percentHint(m model, field int) string {
	permille, err := strconv.Atoi(strings.TrimSpace(s.inputs[field].Value()))
	if err != nil || permille == 0 {
		// A support skill declares no power, and "0%" says nothing the zero did
		// not.
		return ""
	}
	return "  " + m.style.dim.Render(forge.Percent(permille))
}

// listValue draws one of the three allowlists: what is in it, or that anybody
// may carry the skill, which is what an empty list means.
func (s skillsScreen) listValue(m model, chosen []string, labelWidth int) string {
	room := minWidth - 3 - labelWidth - lipgloss.Width(m.text(i18n.KitChooseHint)) - 2
	if len(chosen) == 0 {
		return m.style.dim.Render(m.text(i18n.WhoAnyone) + "  " + m.text(i18n.KitChooseHint))
	}
	return clip(strings.Join(chosen, " "), room) + "  " +
		m.style.dim.Render(m.text(i18n.KitChooseHint))
}

// damageRow is the point of authoring a skill on a screen rather than in a file:
// what the power being typed is actually worth, before it is written.
func (s skillsScreen) damageRow(m model, labelWidth int) string {
	preview, err := m.lib.PreviewDraft(s.draft(m))
	if err != nil {
		return m.labelAt(m.text(i18n.LabelDamage), labelWidth, "%s",
			m.style.bad.Render(m.lang.Error(err)))
	}
	return m.labelAt(m.text(i18n.LabelDamage), labelWidth, "%s", m.lang.Damage(preview))
}
