package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the new-character form, in the order they are walked.
//
// The order is hexforge's prompt order and it is load-bearing in one place:
// the kit comes before the element, because the kit is what decides which
// elements are legal. Here it matters less than at a prompt — everything is on
// screen at once and can be revisited — but the two front-ends walking the same
// order is what makes them comparable, and the live carry check reads better
// when the kit above it is already settled.
const (
	fieldID = iota
	fieldName
	fieldOrigin
	fieldArchetype
	fieldImage
	fieldKit
	fieldElement
	fieldBio
	fieldStatBase
)

const fieldCount = fieldStatBase + progression.KindCount

// formScreen authors one character.
//
// Nothing here decides whether an answer is acceptable. The budget comes from
// forge.Library.Budget, which is progression.EffectiveHP; the carry check comes
// from forge.Library.ValidateElement, which is skill.CanCarry; and the write
// goes through forge.Draft.Resolve, which appends to the book and therefore
// applies exactly the checks a load applies. What this type owns is when to ask
// those questions — which is on every keystroke, because they are integer
// arithmetic and a map lookup and cost nothing.
type formScreen struct {
	inputs     []textinput.Model
	origins    []cast.Origin
	archetypes []cast.Archetype

	originIndex    int
	archetypeIndex int
	cursor         int

	// touched is whether anything has been typed or chosen. It is the whole of
	// the unsaved-changes guard: a form nobody has edited is one an Escape may
	// throw away in silence.
	touched bool

	// These three record whether a field is still following the value something
	// else supplies it — the art path follows the id, the kit and the curves
	// follow the archetype — so that changing the id or the preset updates them,
	// and editing one by hand stops it being overwritten under the author.
	imageFollowsID    bool
	kitFollowsPreset  bool
	statFollowsPreset [progression.KindCount]bool

	// err is the last refusal from a save, and notes is what the last successful
	// one reported. Both are kept as internal/forge produced them — an error
	// value and a slice of forge.Note — rather than as the lines they will be
	// drawn as, so that switching language redraws them instead of leaving the
	// previous language's sentence sitting under a translated form.
	err   error
	notes []forge.Note
}

func newFormScreen(lib *forge.Library) formScreen {
	f := formScreen{
		inputs:           make([]textinput.Model, fieldCount),
		origins:          lib.Origins().All(),
		archetypes:       lib.Archetypes().All(),
		imageFollowsID:   true,
		kitFollowsPreset: true,
	}
	for kind := range f.statFollowsPreset {
		f.statFollowsPreset[kind] = true
	}
	for i := range f.inputs {
		input := textinput.New()
		input.Prompt = ""
		input.CharLimit = 200
		input.Width = 40
		f.inputs[i] = input
	}
	f.inputs[fieldBio].CharLimit = 400
	// A curve is nine characters at its longest, so a narrow field leaves room
	// for the meter and the numbers beside it inside the minimum window.
	for _, kind := range progression.Kinds() {
		f.inputs[fieldStatBase+int(kind)].Width = 11
	}
	f = f.applyPreset()
	f.inputs[f.cursor].Focus()
	return f
}

// applyPreset refills every field that is still following something else.
func (f formScreen) applyPreset() formScreen {
	f.inputs[fieldImage].SetValue(f.imageValue())
	if len(f.archetypes) == 0 {
		return f
	}
	preset := f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)]
	if f.kitFollowsPreset {
		f.inputs[fieldKit].SetValue(strings.Join(preset.Skills, ","))
	}
	for _, kind := range progression.Kinds() {
		if f.statFollowsPreset[kind] {
			f.inputs[fieldStatBase+int(kind)].SetValue(forge.FormatCurve(preset.Stats[kind]))
		}
	}
	return f
}

// imageValue is the art path the id suggests, unless one was typed.
func (f formScreen) imageValue() string {
	if !f.imageFollowsID {
		return f.inputs[fieldImage].Value()
	}
	return forge.SuggestedImage(strings.TrimSpace(f.inputs[fieldID].Value()))
}

// Draft is the answers as internal/forge wants them. It is the only thing this
// screen hands outwards, which is what makes "the form produces the character
// the command line produces" a statement a test can make.
func (f formScreen) draft() forge.Draft {
	draft := forge.Draft{
		ID:      strings.TrimSpace(f.inputs[fieldID].Value()),
		Name:    f.inputs[fieldName].Value(),
		Image:   strings.TrimSpace(f.inputs[fieldImage].Value()),
		Skills:  f.inputs[fieldKit].Value(),
		Element: strings.TrimSpace(f.inputs[fieldElement].Value()),
		Bio:     f.inputs[fieldBio].Value(),
	}
	if len(f.origins) > 0 {
		draft.Origin = f.origins[clamp(f.originIndex, 0, len(f.origins)-1)].ID
	}
	if len(f.archetypes) > 0 {
		draft.Archetype = f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)].ID
	}
	for _, kind := range progression.Kinds() {
		draft.Stats[kind] = f.inputs[fieldStatBase+int(kind)].Value()
	}
	return draft
}

// choiceField reports whether a field is picked from a list rather than typed.
// An origin and an archetype are ids in books, so typing one is a way to get it
// wrong; the list cannot produce an answer the book does not hold.
func choiceField(field int) bool {
	return field == fieldOrigin || field == fieldArchetype
}

func (f formScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		return f.leave(m), nil
	case "ctrl+s":
		f = f.save(m)
		m.form = f
		return m, nil
	case "up", "shift+tab":
		f = f.moveTo(f.cursor - 1)
		m.form = f
		return m, nil
	case "down", "tab", "enter":
		f = f.moveTo(f.cursor + 1)
		m.form = f
		return m, nil
	}
	if choiceField(f.cursor) {
		switch message.String() {
		case "left":
			f = f.cycle(-1)
		case "right":
			f = f.cycle(1)
		}
		m.form = f
		return m, nil
	}

	updated, command := f.inputs[f.cursor].Update(message)
	changed := updated.Value() != f.inputs[f.cursor].Value()
	f.inputs[f.cursor] = updated
	if changed {
		f.touched = true
		f.err = nil
		f.notes = nil
		switch {
		case f.cursor == fieldID:
			// The art path follows the id until somebody types one.
			f.inputs[fieldImage].SetValue(f.imageValue())
		case f.cursor == fieldImage:
			f.imageFollowsID = false
		case f.cursor == fieldKit:
			f.kitFollowsPreset = false
		case f.cursor >= fieldStatBase:
			f.statFollowsPreset[f.cursor-fieldStatBase] = false
		}
	}
	m.form = f
	return m, command
}

// leave goes back to the menu, asking first if there is anything to lose.
func (f formScreen) leave(m model) model {
	if !f.touched {
		m.screen = screenMenu
		return m
	}
	return m.ask(i18n.FormDiscard, func(m model) model {
		m.form = newFormScreen(m.lib)
		m.screen = screenMenu
		return m
	})
}

// moveTo changes the focused field, wrapping at both ends so the form is a ring
// rather than a dead end.
func (f formScreen) moveTo(target int) formScreen {
	f.inputs[f.cursor].Blur()
	f.cursor = (target + fieldCount) % fieldCount
	if !choiceField(f.cursor) {
		f.inputs[f.cursor].Focus()
	}
	return f
}

// cycle steps a chooser, and refills whatever the new preset supplies.
func (f formScreen) cycle(by int) formScreen {
	switch f.cursor {
	case fieldOrigin:
		if len(f.origins) > 0 {
			f.originIndex = (f.originIndex + by + len(f.origins)) % len(f.origins)
			f.touched = true
		}
	case fieldArchetype:
		if len(f.archetypes) > 0 {
			f.archetypeIndex = (f.archetypeIndex + by + len(f.archetypes)) % len(f.archetypes)
			f.touched = true
			f = f.applyPreset()
		}
	}
	f.err = nil
	f.notes = nil
	return f
}

// save resolves the draft and writes it.
//
// Both halves belong to internal/forge: Resolve refuses a character a load
// would refuse, SaveCharacter writes through the temp-file-then-rename that
// keeps a crash from truncating cast.json, and SaveNotes is the same
// confirmation hexforge prints — including the warning that the art is not
// there yet and the reminder that the game boots from the embedded copy.
func (f formScreen) save(m model) formScreen {
	character, err := f.draft().Resolve(m.lib)
	if err != nil {
		f.err = err
		f.notes = nil
		return f
	}
	if err := m.lib.SaveCharacter(character); err != nil {
		f.err = err
		f.notes = nil
		return f
	}
	notes := m.lib.SaveNoteFacts(character)
	fresh := newFormScreen(m.lib)
	fresh.notes = notes
	return fresh
}

// fieldLabel is what each row is called.
//
// Stat rows are named by their short label — hp, atk, def — in every language,
// and forge.ShortStat says why: those six are the flag names and the data
// files' own keys, so an author needs them as they are written to act on them.
func fieldLabel(m model, field int) string {
	labels := map[int]i18n.Key{
		fieldID:        i18n.FieldID,
		fieldName:      i18n.FieldName,
		fieldOrigin:    i18n.FieldOrigin,
		fieldArchetype: i18n.FieldArchetype,
		fieldImage:     i18n.FieldArt,
		fieldKit:       i18n.FieldKit,
		fieldElement:   i18n.FieldElement,
		fieldBio:       i18n.FieldBiography,
	}
	// A map read by key, never ranged over into anything drawn: the order on
	// screen is the field order, which is the constants above.
	if key, named := labels[field]; named {
		return m.text(key)
	}
	return forge.ShortStat(progression.Kind(field - fieldStatBase))
}

// formLabelWidth is the column the field names sit in. It is measured from the
// labels themselves rather than declared, because the longest of them is
// "archetype" in one language and "mẫu vai trò" in the other, and a constant
// would be wrong for one of them the next time either is reworded.
func formLabelWidth(m model) int {
	widest := 0
	for field := range fieldCount {
		if width := lipgloss.Width(fieldLabel(m, field)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

func (f formScreen) view(m model) (string, string) {
	footer := m.text(i18n.FormFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.FormHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.FormSubtitle, progression.LevelCap)) + "\n\n")

	width := formLabelWidth(m)
	for field := range fieldCount {
		out.WriteString(f.row(m, field, width))
	}
	out.WriteString("\n")
	out.WriteString(f.liveChecks(m, width))
	return out.String(), footer
}

// row draws one field: a marker, its name, and either its text or its choice.
func (f formScreen) row(m model, field, labelWidth int) string {
	marker := "  "
	if field == f.cursor {
		marker = "> "
	}
	name := pad(fieldLabel(m, field), labelWidth)
	if field == f.cursor {
		name = m.style.selected.Render(name)
	} else {
		name = m.style.label.Render(name)
	}

	var value string
	switch {
	case field == fieldOrigin:
		value = f.choice(m, f.originIndex, len(f.origins), f.originLabel())
	case field == fieldArchetype:
		value = f.choice(m, f.archetypeIndex, len(f.archetypes), f.archetypeLabel(m))
	case field >= fieldStatBase:
		value = f.statRow(m, progression.Kind(field-fieldStatBase))
	default:
		value = f.inputs[field].View()
	}
	return marker + name + " " + value + "\n"
}

// choice renders a picked value with its position, so "there are more of these"
// is visible without pressing anything.
func (f formScreen) choice(m model, index, total int, label string) string {
	if total == 0 {
		return m.style.bad.Render(m.text(i18n.NoneCatalogued))
	}
	return fmt.Sprintf("< %s >  %s", label,
		m.style.dim.Render(m.text(i18n.ChoicePosition, index+1, total)))
}

func (f formScreen) originLabel() string {
	if len(f.origins) == 0 {
		return ""
	}
	origin := f.origins[clamp(f.originIndex, 0, len(f.origins)-1)]
	return fmt.Sprintf("%s — %s", origin.ID, origin.Title)
}

func (f formScreen) archetypeLabel(m model) string {
	if len(f.archetypes) == 0 {
		return ""
	}
	preset := f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)]
	return fmt.Sprintf("%s — %s", preset.ID, m.lang.PresetSummary(preset))
}

// statRow is a curve as "base:max", a meter of where its maximum sits against
// the ceiling for that stat, and the ceiling itself.
func (f formScreen) statRow(m model, kind progression.Kind) string {
	input := f.inputs[fieldStatBase+int(kind)]
	ceiling := m.lib.Limits().Ceilings[kind]
	curve, err := forge.ParseCurve(input.Value())
	meter := m.style.dim.Render(strings.Repeat("-", statBarWidth+2))
	trailing := ""
	if err != nil {
		// The refusal is forge.ParseCurve's, worded by the catalog. This screen
		// does not decide what is wrong with a curve, only where to put it.
		return fmt.Sprintf("%s %s %s", input.View(), meter, m.style.bad.Render(m.lang.Error(err)))
	}
	meter = bar(statBarWidth, curve.Max, ceiling)
	trailing = m.text(i18n.CurveAgainstCeiling, curve.Base, curve.Max, ceiling)
	if curve.Max > ceiling {
		trailing = m.style.bad.Render(trailing + "  " + m.text(i18n.OverTheCeiling))
	} else {
		trailing = m.style.dim.Render(trailing)
	}
	return fmt.Sprintf("%s %s %s", input.View(), meter, trailing)
}

// liveChecks is the two answers that make this screen worth having: what the
// stat line spends of the joint budget, and whether the affinity can carry the
// kit. Both are recomputed on every keystroke because both are cheap, and both
// come from internal/forge so that neither can disagree with the write.
func (f formScreen) liveChecks(m model, labelWidth int) string {
	var out strings.Builder
	draft := f.draft()

	table, err := draft.Table(m.lib)
	if err != nil {
		out.WriteString(m.labelAt(m.text(i18n.LabelBudget), labelWidth, "%s", m.style.bad.Render(m.lang.Error(err))))
	} else {
		values := table.At(progression.LevelCap)
		out.WriteString(m.labelAt(m.text(i18n.LabelBudget), labelWidth, "%s", budgetLine(m, m.lib.Budget(values))))
	}

	out.WriteString(m.labelAt(m.text(i18n.LabelCarries), labelWidth, "%s", f.carryLine(m, draft)))

	switch {
	case f.err != nil:
		out.WriteString("\n" +
			m.style.bad.Render(m.text(i18n.WriteRefused, m.lang.Error(f.err))) + "\n")
	case len(f.notes) > 0:
		out.WriteString("\n")
		for i, note := range m.lang.Notes(f.notes) {
			if i == 0 {
				out.WriteString(m.style.good.Render(note) + "\n")
				continue
			}
			out.WriteString(m.style.dim.Render(note) + "\n")
		}
	}
	return out.String()
}

// carryLine says whether the affinity carries every skill in the kit, and names
// the first one it cannot.
//
// The judgement is not this screen's doing: forge.CheckCarry, reached through
// ValidateElement, decides it and hands back a *forge.CarryError holding the
// affinity, the skill and the skill's element, so the form, the prompt and the
// parser cannot disagree about a mismatch. Only the sentence is chosen here,
// and only by asking the catalog for it.
func (f formScreen) carryLine(m model, draft forge.Draft) string {
	names := draft.KitNames(m.lib)
	kit, err := m.lib.LookupKit(names)
	if err != nil {
		return m.style.bad.Render(m.lang.Error(err))
	}
	if strings.TrimSpace(draft.Element) == "" {
		return m.style.dim.Render(m.text(i18n.CarryNoElementYet, m.lang.KitSummary(kit)))
	}
	if err := m.lib.ValidateElement(draft.Element, kit); err != nil {
		return m.style.bad.Render(m.text(i18n.CarryRefused, m.lang.Error(err)))
	}
	return m.style.good.Render(m.text(i18n.CarryAccepted, draft.Element))
}
