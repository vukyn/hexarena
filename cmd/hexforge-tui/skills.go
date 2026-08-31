package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the new-skill form, in the order they are walked.
//
// Nine core fields, the display name, the statuses it inflicts (both sides), and
// the three allowlists. What is deliberately absent is requires, self_requires,
// self_gradient, strips, scaling and summons: each is a composite worth several
// questions of its own, and a form that asked a dozen more would be worse than an
// edit to skills.json. They survive a save untouched — see skill.Skill.MarshalJSON.
//
// ⚠️ This list was written down in three places (here, forge.resolveOnto, and
// TestTheShippedSkillBookSurvivesBeingWritten) and every copy was wrong the same
// way: each named self_applies, which this form *does* ask about, and none named
// self_requires or summons, which it does not. Corrected when self_gradient
// joined them. A list restated three times drifts three times; if it drifts
// again, derive it.
//
// The name sits second because that is where it is authored: a skill and the name
// it is called by are one thought, which is also why the name is a field on
// skill.Skill and not a translations file beside the book.
const (
	skillFieldID = iota
	skillFieldName
	skillFieldFlavour
	skillFieldElement
	skillFieldTarget
	skillFieldRange
	skillFieldShape
	skillFieldPower
	skillFieldStrikes
	skillFieldAccuracy
	skillFieldCooldown
	skillFieldInflicts
	skillFieldOnItself
	skillFieldPierce
	// Beside pierce, so the form reads in the order the file is written in.
	skillFieldCrit
	skillFieldRestores
	skillFieldDrains
	skillFieldKeptForElements
	skillFieldKeptForRoles
	skillFieldKeptForCharacters
	skillFieldKeptForSpecies
	skillFieldKeptForOrigins
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
	keptKinds    []string
	keptWorlds   []string
	touched      bool
	// shapeDrawn is whether the shape diagram is over the form.
	//
	// A flag rather than a state of its own, because the diagram has no answer
	// to hold: it draws the shape the chooser is already on and the arrow keys
	// on it are the chooser's own, so there is one shapeIndex and the drawing
	// cannot disagree with the field behind it. That is the difference from the
	// picker, which collects an answer and hands it back on enter.
	shapeDrawn bool

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
		input := newInput()
		input.Prompt = ""
		input.CharLimit = 200
		input.SetWidth(24)
		s.inputs[i] = input
	}
	s.inputs[skillFieldID].SetWidth(32)
	s.inputs[skillFieldName].SetWidth(32)
	s.inputs[skillFieldFlavour].SetWidth(48)
	s.inputs[skillFieldInflicts].SetWidth(40)
	s.inputs[skillFieldOnItself].SetWidth(40)
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
	s.keptKinds, s.keptWorlds = nil, nil
	s.field = skillFieldID
	s.touched = false
	s.shapeDrawn = false
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
		{skillFieldName, answers.Name},
		{skillFieldFlavour, answers.Flavour},
		{skillFieldRange, answers.Range},
		{skillFieldPower, answers.Power},
		{skillFieldStrikes, answers.Strikes},
		{skillFieldAccuracy, answers.Accuracy},
		{skillFieldCooldown, answers.Cooldown},
		{skillFieldInflicts, answers.Applies},
		{skillFieldOnItself, answers.SelfApplies},
		{skillFieldPierce, answers.Pierce},
		{skillFieldCrit, answers.Crit},
		{skillFieldRestores, answers.Restores},
		{skillFieldDrains, answers.Drains},
	} {
		s.inputs[filled.field].SetValue(filled.value)
	}
	s.elementIndex = indexOf(forge.ElementNames(), answers.Element)
	s.targetIndex = indexOf(forge.TargetNames(), answers.Target)
	s.shapeIndex = indexOf(lib.PatternNames(), answers.Pattern)
	s.keptElements = forge.SplitList(answers.RestrictElements)
	s.keptRoles = forge.SplitList(answers.RestrictArchetypes)
	s.keptWho = forge.SplitList(answers.RestrictCharacters)
	s.keptKinds = forge.SplitList(answers.RestrictSpecies)
	s.keptWorlds = forge.SplitList(answers.RestrictOrigins)
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
		Name:               s.inputs[skillFieldName].Value(),
		Flavour:            s.inputs[skillFieldFlavour].Value(),
		Element:            at(forge.ElementNames(), s.elementIndex),
		Target:             at(forge.TargetNames(), s.targetIndex),
		Range:              s.inputs[skillFieldRange].Value(),
		Pattern:            at(m.lib.PatternNames(), s.shapeIndex),
		Power:              s.inputs[skillFieldPower].Value(),
		Strikes:            s.inputs[skillFieldStrikes].Value(),
		Accuracy:           s.inputs[skillFieldAccuracy].Value(),
		Cooldown:           s.inputs[skillFieldCooldown].Value(),
		Applies:            s.inputs[skillFieldInflicts].Value(),
		SelfApplies:        s.inputs[skillFieldOnItself].Value(),
		Pierce:             s.inputs[skillFieldPierce].Value(),
		Crit:               s.inputs[skillFieldCrit].Value(),
		Restores:           s.inputs[skillFieldRestores].Value(),
		Drains:             s.inputs[skillFieldDrains].Value(),
		RestrictElements:   strings.Join(s.keptElements, ","),
		RestrictArchetypes: strings.Join(s.keptRoles, ","),
		RestrictCharacters: strings.Join(s.keptWho, ","),
		RestrictSpecies:    strings.Join(s.keptKinds, ","),
		RestrictOrigins:    strings.Join(s.keptWorlds, ","),
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
	case skillFieldKeptForElements, skillFieldKeptForRoles,
		skillFieldKeptForCharacters, skillFieldKeptForSpecies,
		skillFieldKeptForOrigins:
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
func (s skillsScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "?":
		// The description screen keeps no cursor of its own and reads this one,
		// so raising it needs nothing handed over — the same arrangement the art
		// preview has with the browser.
		if len(s.skills) > 0 {
			m.skills = s
			m.blurb.from = screenSkills
			m.screen = screenBlurb
			return m, nil
		}
	}
	m.skills = s
	return m, nil
}

func (s skillsScreen) updateForm(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// The diagram is answered first, before Escape can be read as leaving the
	// form and before Enter can be read as moving to the next field: while it is
	// over the form, both of those close it and nothing else on the form is
	// reachable.
	if s.shapeDrawn {
		switch message.String() {
		case "esc", "enter", "space":
			s.shapeDrawn = false
		case "left":
			s = s.cycle(m, -1)
		case "right":
			s = s.cycle(m, 1)
		}
		m.skills = s
		return m, nil
	}
	// After the diagram and before the switch: saving answers to more than one
	// keystroke, and isSaveKey is the single declaration of which. It comes
	// second because the diagram takes every key while it is open.
	if isSaveKey(message) {
		s = s.save(m)
		m.skills = s
		return m, nil
	}
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
		case "space":
			// Space is the same key the three allowlists open their picker with,
			// and it is free on a chooser: a chooser is stepped with the arrows.
			// Only the shape has anything to open.
			if s.field == skillFieldShape {
				s.shapeDrawn = true
			}
		}
		m.skills = s
		return m, nil
	}
	// The inflicts field is the one text field with a list behind it. Space
	// opens it rather than typing a space, and that costs nothing: the syntax
	// ParseApplications reads has no spaces in it, and every other way of
	// filling this field still works, because the field is the record.
	if (s.field == skillFieldInflicts || s.field == skillFieldOnItself) && message.String() == "space" {
		m.skills = s
		return m.openStatuses(), nil
	}
	if skillListField(s.field) {
		if message.String() == "space" || message.String() == "right" {
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

// openAllowlist raises the picker for one of the four lists.
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
			hint:    i18n.PickerAllowlistHint,
			options: idOptions(forge.ElementNames()), chosen: m.skills.keptElements,
			apply: func(m model, answer pickAnswer) model {
				m.skills.keptElements = answer.Chosen
				m.skills.touched = true
				return m
			},
		})
	case skillFieldKeptForRoles:
		return m.pick(&pickState{
			title: i18n.PickerRolesTitle, kind: pickArchetypes,
			hint:    i18n.PickerAllowlistHint,
			options: idOptions(m.lib.Archetypes().IDs()), chosen: m.skills.keptRoles,
			apply: func(m model, answer pickAnswer) model {
				m.skills.keptRoles = answer.Chosen
				m.skills.touched = true
				return m
			},
		})
	case skillFieldKeptForOrigins:
		return m.pick(&pickState{
			title: i18n.PickerOriginsTitle, kind: pickOrigins,
			hint:    i18n.PickerAllowlistHint,
			options: idOptions(m.lib.Origins().IDs()), chosen: m.skills.keptWorlds,
			apply: func(m model, answer pickAnswer) model {
				m.skills.keptWorlds = answer.Chosen
				m.skills.touched = true
				return m
			},
		})
	case skillFieldKeptForSpecies:
		return m.pick(&pickState{
			title: i18n.PickerSpeciesTitle, kind: pickSpecies,
			hint:    i18n.PickerAllowlistHint,
			options: idOptions(m.lib.Species().IDs()), chosen: m.skills.keptKinds,
			apply: func(m model, answer pickAnswer) model {
				m.skills.keptKinds = answer.Chosen
				m.skills.touched = true
				return m
			},
		})
	default:
		// The one list with a filter, because it is the one that grows: the
		// elements are eleven and fixed and the role presets are five, while the
		// cast is whatever has been authored. It narrows by origin and takes the
		// cast browser's own key, so filtering a list of characters is one
		// interaction wherever it happens.
		return m.pick(&pickState{
			title: i18n.PickerCharactersTitle, kind: pickCharacters,
			hint:    i18n.PickerAllowlistHint,
			footer:  i18n.PickerFilterFooter,
			options: characterOptions(m.lib), groups: m.lib.OriginIDs(),
			chosen: m.skills.keptWho,
			apply: func(m model, answer pickAnswer) model {
				m.skills.keptWho = answer.Chosen
				m.skills.touched = true
				return m
			},
		})
	}
}

// openStatuses raises the picker over the status book and writes what comes back
// into the inflicts field.
//
// The shortest path from "I want a poison" to a valid entry: pick the status out
// of the book, type the chance in the field under the list, enter. The syntax is
// forge.AddApplications' — the same spelling ParseApplications reads back — so
// the screen never spells an entry itself, and nothing about the field changes:
// it is still a text field, an author who knows the syntax still types it, and a
// script writes the same thing.
//
// Nothing is preselected. The field may already hold entries and the picker
// appends to them, so starting with those rows ticked would mean the author had
// to untick them to avoid writing each one twice.
func (m model) openStatuses() model {
	return m.pick(&pickState{
		title: i18n.PickerStatusesTitle, kind: pickStatuses,
		hint: i18n.PickerStatusHint, footer: i18n.PickerStatusFooter,
		options: statusOptions(m.lib),
		typed:   numberField(forge.DefaultApplicationChance),
		label:   i18n.PickerChance,
		apply: func(m model, answer pickAnswer) model {
			if len(answer.Chosen) == 0 {
				return m
			}
			field := &m.skills.inputs[skillFieldInflicts]
			written, err := m.lib.AddApplications(field.Value(), answer.Chosen, answer.Typed)
			if err != nil {
				// A refusal from here can only be a chance that is not a number,
				// which the field refuses a keystroke at a time — so this is
				// unreachable and reported rather than swallowed, on the form's
				// own error line.
				m.skills.err = err
				return m
			}
			field.SetValue(written)
			// The cursor goes to the end, because what was just written is at the
			// end and the author's next move is usually to adjust it.
			field.CursorEnd()
			m.skills.touched = true
			m.skills.err = nil
			m.skills.added, m.skills.edited = nil, nil
			return m
		},
	})
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
// trailing newline leaves", copied from the picker's own count, and this body has no trailing
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
//
// glossColumn of zero drops the translated-name column entirely rather than
// drawing it empty. That one rule covers both cases that need it: English,
// where nothing is glossed, and a book whose ids all happen to be unglossed —
// a column of blanks would read as missing data rather than as a column that
// does not apply.
//
// powerColumn is a parameter for the same reason glossColumn is, and it stopped
// being a constant when the power column's header stopped being the word
// "power": the header is the label the form authored the number with, so a
// column of 8 held "1000x1" but cut "damage multiplier" — or, since pad only
// widens, let the header run 9 cells past the column and push the last column's
// header right of the rows it names. One header out of line with its own rows is
// the one failure this function exists to prevent.
func skillRow(idColumn, glossColumn, powerColumn int, id, gloss, member, power, who string) string {
	name := pad(id, idColumn)
	if glossColumn > 0 {
		name += " " + pad(gloss, glossColumn)
	}
	return fmt.Sprintf("%s %s %s%s", name, pad(member, 9), pad(power, powerColumn), who)
}

// skillPowerColumn is the width the power column takes: enough for the figures
// the rows hold, and enough for the header naming them.
//
// The figures are short by construction — a power and a strike count, "1000x1"
// — so the header is what decides this, and it differs per language for the
// same reason every other measured column here does.
//
// The measured label gets a cell added, and the 8 has one already: it is the
// last column before free text, so without a gap a header exactly as wide as its
// column runs straight into the next one. "hệ số sát thương" is 16 cells, which
// is what made that visible.
func skillPowerColumn(m model) int {
	const figures = 8
	if width := lipgloss.Width(m.text(i18n.SkillFieldPower)) + 1; width > figures {
		return width
	}
	return figures
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
		if width := lipgloss.Width(m.lang.SkillName(current)); width > glossColumn {
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
	// authored it uses — which is the point of naming them from the same keys:
	// an author who has just typed a damage multiplier on the form should find
	// that column called the same thing here, rather than the shorter word the
	// form stopped using.
	powerColumn := skillPowerColumn(m)
	out.WriteString("  " + m.style.dim.Render(skillRow(column+1, glossColumn, powerColumn,
		m.text(i18n.SkillFieldID), m.text(i18n.ColumnGloss), m.text(i18n.LabelElement),
		m.text(i18n.SkillFieldPower), m.text(i18n.ColumnWhoMayCarry))) + "\n")
	for index := from; index < to; index++ {
		current := s.skills[index]
		marker := "  "
		// The power and the strike count are the balance, so they are the two
		// numbers on the row; everything else about a skill is a keypress away
		// on the form that authored it.
		row := skillRow(column+1, glossColumn, powerColumn,
			current.ID, m.lang.SkillName(current),
			current.Element.String(),
			strconv.Itoa(current.Power)+"x"+strconv.Itoa(current.StrikeCount()), "")
		// Measured against the window rather than against the floor. minWidth is
		// the width this program promises to draw in, not a ceiling on what it
		// may spend, and this last column is data: a restriction cut to "để dành
		// cho loài dr…" is a row that stopped saying which species it is for,
		// on a terminal with a hundred spare columns beside it. Prose still
		// wraps at the floor — a paragraph run across a wide terminal is a line
		// a reader loses their place in — but a table cell is read by scanning
		// down it, so width is the one thing it can always use.
		row += clip(m.lang.WhoMaySummary(current), m.usableWidth()-3-lipgloss.Width(row))
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
		out.WriteString(m.label(m.text(i18n.LabelDamage), "%s",
			m.lang.DamageWithin(preview, damageRowRoom(m.usableWidth(), detailLabelWidth(m)))))
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
		skillFieldID: i18n.SkillFieldID,
		// The same key the listing's own column uses, because they are the same
		// thing: an author who has just typed a name here should find the column
		// that shows it called what they typed it into.
		skillFieldName:              i18n.ColumnGloss,
		skillFieldFlavour:           i18n.LabelFlavour,
		skillFieldElement:           i18n.SkillFieldElement,
		skillFieldTarget:            i18n.SkillFieldTarget,
		skillFieldRange:             i18n.SkillFieldRange,
		skillFieldShape:             i18n.SkillFieldShape,
		skillFieldPower:             i18n.SkillFieldPower,
		skillFieldStrikes:           i18n.SkillFieldStrikes,
		skillFieldAccuracy:          i18n.SkillFieldAccuracy,
		skillFieldCooldown:          i18n.SkillFieldCooldown,
		skillFieldInflicts:          i18n.SkillFieldInflicts,
		skillFieldOnItself:          i18n.SkillFieldOnItself,
		skillFieldPierce:            i18n.SkillFieldPierce,
		skillFieldCrit:              i18n.SkillFieldCrit,
		skillFieldRestores:          i18n.SkillFieldRestores,
		skillFieldDrains:            i18n.SkillFieldDrains,
		skillFieldKeptForElements:   i18n.SkillFieldKeptForElements,
		skillFieldKeptForRoles:      i18n.SkillFieldKeptForRoles,
		skillFieldKeptForCharacters: i18n.SkillFieldKeptForCharacters,
		skillFieldKeptForSpecies:    i18n.SkillFieldKeptForSpecies,
		skillFieldKeptForOrigins:    i18n.SkillFieldKeptForOrigins,
	}
	return m.text(keys[field])
}

// skillFieldHelp is the line describing the field the cursor is on: what it
// means, and an answer that would be valid.
//
// One entry per field, and the array is indexed by the field constant for the
// same reason skillFieldLabel's is — a field added without a help line is a
// blank line rather than a build failure, so TestEveryFieldOfTheSkillFormHasHelp
// walks every field and reads the screen.
//
// It replaced a static footnote about parts per thousand. That footnote was true
// of two fields out of fourteen and drawn whichever field had the cursor, which
// is why it explained nothing about the fields nobody could guess: what a shape
// covers, what syntax the statuses take, what an empty allowlist means.
func skillFieldHelp(m model, field int) string {
	keys := [skillFieldCount]i18n.Key{
		skillFieldID:                i18n.SkillHelpID,
		skillFieldName:              i18n.SkillHelpName,
		skillFieldFlavour:           i18n.SkillHelpFlavour,
		skillFieldElement:           i18n.SkillHelpElement,
		skillFieldTarget:            i18n.SkillHelpTarget,
		skillFieldRange:             i18n.SkillHelpRange,
		skillFieldShape:             i18n.SkillHelpShape,
		skillFieldPower:             i18n.SkillHelpPower,
		skillFieldStrikes:           i18n.SkillHelpStrikes,
		skillFieldAccuracy:          i18n.SkillHelpAccuracy,
		skillFieldCooldown:          i18n.SkillHelpCooldown,
		skillFieldInflicts:          i18n.SkillHelpInflicts,
		skillFieldOnItself:          i18n.SkillHelpOnItself,
		skillFieldPierce:            i18n.SkillHelpPierce,
		skillFieldCrit:              i18n.SkillHelpCrit,
		skillFieldRestores:          i18n.SkillHelpRestores,
		skillFieldDrains:            i18n.SkillHelpDrains,
		skillFieldKeptForElements:   i18n.SkillHelpKeptForElements,
		skillFieldKeptForRoles:      i18n.SkillHelpKeptForRoles,
		skillFieldKeptForCharacters: i18n.SkillHelpKeptForCharacters,
		skillFieldKeptForSpecies:    i18n.SkillHelpKeptForSpecies,
		skillFieldKeptForOrigins:    i18n.SkillHelpKeptForOrigins,
	}
	return m.text(keys[clamp(field, 0, skillFieldCount-1)])
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

// The two marks the diagram puts in a cell. hex.Render gives each cell two
// characters, so both are two.
//
// Characters and not colour, for the reason the picker's own marks are: the
// meaning has to survive NO_COLOR, a monochrome terminal and a recording that
// lost its escape codes. The dense mark is the cell the skill is aimed at and
// the sparse one is a cell that only catches the splash share, which is the
// weight each carries as well as the ink.
const (
	shapeAimMark    = "##"
	shapeSplashMark = ".."
)

// viewShape is the shape diagram: the board with the cells this shape catches
// marked, drawn from forge.ShapeCoverage.
//
// It is a sub-screen rather than a pane beside the form, and that was measured
// rather than judged: the form spends nineteen of the twenty body lines an 80x24
// window has — twenty with a refusal under it — and hex.Render is eight lines
// before a heading, a legend or the blanks around it. There was no room, and
// hiding half a board is worse than opening one.
//
// The arrows here are the chooser's own, so the drawing follows the field rather
// than holding a copy of it, and nothing needs applying when it closes.
func (s skillsScreen) viewShape(m model) (string, string) {
	footer := m.text(i18n.SkillShapeFooter)
	shapes := m.lib.PatternNames()
	name := at(shapes, s.shapeIndex)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.SkillShapeHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition,
			clamp(s.shapeIndex, 0, len(shapes)-1)+1, len(shapes))) + "\n")

	// The draft rather than the shape alone: what a shape covers depends on the
	// side the skill aims at, and that is the field above this one.
	draft := s.draft(m)
	coverage, err := m.lib.ShapeCoverage(draft.Pattern, draft.Target)
	if err != nil {
		// Unreachable through the chooser, which offers the book's own names,
		// and drawn rather than swallowed for the same reason the picker draws a
		// lost id: a shape the book cannot resolve is worth seeing.
		out.WriteString(m.style.bad.Render("  "+m.lang.Error(err)) + "\n")
		return out.String(), footer
	}
	// How many cells it really catches, and — when the aim loses one — that it
	// did, so a shape drawing short reads as this aim rather than as this shape.
	caught := m.text(i18n.SkillShapeCoverage, coverage.Covered())
	if !coverage.Whole() {
		caught = m.text(i18n.SkillShapeShort, coverage.Covered(), coverage.Max)
	}
	out.WriteString("  " + m.style.selected.Render(name) + "  " +
		m.style.dim.Render(caught) + "\n")
	out.WriteString("  " + m.style.dim.Render(
		m.text(i18n.SkillShapeDrawnAt, coverage.Primary)) + "\n\n")

	for _, line := range strings.Split(shapeBoard(coverage), "\n") {
		out.WriteString("  " + line + "\n")
	}
	out.WriteString("\n  " + m.style.dim.Render(m.text(i18n.SkillShapeLegend,
		shapeAimMark, shapeSplashMark, m.lib.SplashShare())))
	return out.String(), footer
}

// shapeBoard draws the battlefield with a coverage marked on it.
//
// The board is hex.Render, the same drawing the terminal client shows a
// formation with, and the cells are the ones pattern.Targets returned — so this
// is a rendering of two existing functions and holds no geometry of its own.
func shapeBoard(coverage forge.ShapeCoverage) string {
	return hex.Render(func(cell hex.Offset) string {
		if cell == coverage.Primary {
			return shapeAimMark
		}
		if slices.Contains(coverage.Splash, cell) {
			return shapeSplashMark
		}
		return ""
	})
}

// formRoom is how many field rows the window has.
//
// The body gets m.height-4. This form spends the rest on a heading, a blank, a
// blank, the damage row and the help line — five — plus a row for each ellipsis
// the window draws and one for a refusal when there is one. Counting them here
// rather than guessing is what the listing had to learn twice: a reserve one out
// truncates the screen's own summary and looks like a layout bug rather than an
// arithmetic one.
func (s skillsScreen) formRoom(m model) int {
	spent := 5
	if s.err != nil {
		spent++
	}
	// Room for both ellipses whenever the window cannot hold everything, so the
	// count does not change as the cursor moves and shift every row under it.
	room := m.height - 4 - spent
	if room < skillFieldCount {
		room -= 2
	}
	if room < 1 {
		room = 1
	}
	return room
}

func (s skillsScreen) viewForm(m model) (string, string) {
	if s.shapeDrawn {
		return s.viewShape(m)
	}
	footer := m.text(i18n.SkillFormFooter, saveKeyLabel())
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
	// The form scrolls now. It spent every row an 80x24 window has at fourteen
	// fields, and healing brought three more, so the choice was between a
	// sub-screen for the rest and a window over all of them — and a form split
	// in two makes an author hunt for a field rather than scroll to it.
	//
	// The window follows the cursor rather than the top, so tabbing to the last
	// field brings it into view instead of leaving the cursor off screen.
	from, to := window(skillFieldCount, s.field, s.formRoom(m))
	if from > 0 {
		out.WriteString("  " + m.style.dim.Render(ellipsis) + "\n")
	}
	for field := from; field < to; field++ {
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

	if to < skillFieldCount {
		out.WriteString("  " + m.style.dim.Render(ellipsis) + "\n")
	}

	out.WriteString("\n")
	out.WriteString(s.damageRow(m, width))
	// The help for the field the cursor is on, at the body's own indent rather
	// than in the value column. The label column is measured per language and
	// takes a third of the row in Vietnamese; a sentence that has to say what a
	// field means *and* show a valid answer cannot spend that, and it is not a
	// value belonging to a row anyway.
	//
	// The last line carries no newline of its own, and that is a line rather than
	// a tidy: frame splits the body on newlines, so a trailing one leaves an
	// empty string that costs a row of the twenty an 80x24 window has. This form
	// spent all twenty before the name field arrived; dropping the newline is
	// what paid for it, with nothing to see on screen either way. It is the same
	// accounting skillsRoom records for the listing, which has never had one.
	tail := []string{"  " + m.style.dim.Render(skillFieldHelp(m, s.field))}
	if s.err != nil {
		tail = append(tail,
			m.style.bad.Render(m.text(i18n.WriteRefused, m.lang.Error(s.err))))
	}
	out.WriteString(strings.Join(tail, "\n"))
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
	case skillFieldKeptForSpecies:
		return s.listValue(m, s.keptKinds, labelWidth)
	case skillFieldKeptForOrigins:
		return s.listValue(m, s.keptWorlds, labelWidth)
	case skillFieldAccuracy, skillFieldPower, skillFieldPierce, skillFieldCrit,
		skillFieldRestores, skillFieldDrains:
		// Every one of these is authored in parts per thousand because that is
		// what the engine multiplies and divides by, but nobody reads 850 as a
		// chance or 2200 as "twice over". The percentage sits beside the field rather than
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
	//
	// What is left is measured from the field as it is *drawn* rather than from
	// the Width it was given, and the difference is a real cell: a bubbles text
	// field renders its own trailing cursor, so its View is a cell wider than
	// its Width. A room computed from the declaration left this row 80 cells
	// wide in an 80-column window — inside frame's clip, so nothing was cut, and
	// over the edge on the terminals that wrap a line filling the final cell.
	// It only became visible when the label column grew.
	//
	// Against the window and not the floor: a chance is data, and a list of them
	// cut short on a wide terminal hides one of the numbers being authored.
	room := fieldValueRoom(m.usableWidth(), labelWidth,
		lipgloss.Width(s.inputs[skillFieldInflicts].View()))
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
//
// The list is ids, so it takes the window: an allowlist clipped at the floor
// stops naming the last character it is kept for, and which characters those
// are is the whole content of the field.
func (s skillsScreen) listValue(m model, chosen []string, labelWidth int) string {
	room := fieldValueRoom(m.usableWidth(), labelWidth,
		lipgloss.Width(m.text(i18n.KitChooseHint)))
	if len(chosen) == 0 {
		return m.style.dim.Render(m.text(i18n.WhoAnyone) + "  " + m.text(i18n.KitChooseHint))
	}
	return clip(strings.Join(chosen, " "), room) + "  " +
		m.style.dim.Render(m.text(i18n.KitChooseHint))
}

// damageRowRoom is what the damage reading has to fit in before Lang.DamageWithin
// gives up its reference pair: the row less the marker, the label column and the
// space after it, on a row whose value is the last thing written.
//
// **The window and not the floor, and this one is a classification rather than a
// clip.** What the room decides here is not how much of a value survives — it is
// which of two whole catalog lines is drawn. Over the room, DamageWithin returns
// the short reading and the pair the figures are measured against ("đánh vào 800
// công và 400 phòng") is **silently omitted**: nothing is cut, so there is no
// ellipsis, and the row reads as a complete sentence that has quietly stopped
// saying what its numbers are relative to. Measured at the floor, a
// two-hundred-column terminal was making a drop it had a hundred and twenty
// spare columns not to make.
//
// **It is data, and the mixed content is why that needed arguing.** The line is
// figures wrapped in catalog wording, which is the shape the prose half of the
// rule usually names — but neither of the prose half's two reasons reaches it:
//
//   - The sweep's subject is the **wording**, and the wording is unaffected.
//     DamageLine and DamageLineShort are fixed strings and both are inside the
//     floor for every figure the shipped books can produce (widest 59 cells in
//     Vietnamese, 57 in English, against a floor room of 61 and 64), so
//     TestEveryWordingFitsTheMinimumWidth measures exactly what it measured
//     before. All the room changes is which of the two is chosen, on figures
//     large enough that neither fits — and an author typing a power is who
//     reaches those.
//   - "A paragraph across a wide terminal is a line a reader loses their place
//     in" is about a wrapped block. This is one labelled value on one row, read
//     by looking at it, which is the table cell the data half is about. The
//     reference pair itself is two more figures.
//
// So the pair comes back on a wide terminal, and
// TestTheDamageRowKeepsItsReferencePairOnAWideWindow is what says so.
//
// ⚠️ **One declaration because there are two callers** — the listing's reading of
// the skill under the cursor and the form's preview of an unwritten one — and
// they had the arithmetic written out twice. That is the same shape
// fieldValueRoom exists to have stopped, found again one file over.
//
// ⚠️ **This is a cell over, and it is NOT fixed here — reported instead.** The
// row comes to `2 + labelWidth + 1 + value`, so a value of exactly this room
// fills the window's final cell, which is the one column every other row in this
// program leaves empty because a line filling it wraps on some terminals. The
// correct room is one less. It is left alone because it is a different defect
// from the one this function was written for: it is wrong **at the floor** too,
// where the promise lives, so it deserves a test that builds the figure that
// reaches it rather than riding in on a classification change. No shipped skill
// gets near it — the widest reading is 59 of the 61 there are.
func damageRowRoom(width, labelWidth int) int {
	const marker = 2
	return width - marker - labelWidth - 1
}

// damageRow is the point of authoring a skill on a screen rather than in a file:
// what the power being typed is actually worth, before it is written.
func (s skillsScreen) damageRow(m model, labelWidth int) string {
	preview, err := m.lib.PreviewDraft(s.draft(m))
	if err != nil {
		return m.labelAt(m.text(i18n.LabelDamage), labelWidth, "%s",
			m.style.bad.Render(m.lang.Error(err)))
	}
	return m.labelAt(m.text(i18n.LabelDamage), labelWidth, "%s",
		m.lang.DamageWithin(preview, damageRowRoom(m.usableWidth(), labelWidth)))
}
