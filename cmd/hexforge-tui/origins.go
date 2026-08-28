package main

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the add-a-work form.
const (
	originFieldID = iota
	originFieldTitle
	originFieldMedium
	originFieldYear
	originFieldNote
	originFieldCount
)

// originsScreen is the catalog of works the cast is borrowed from, and the
// form that adds one.
//
// A work has to exist before a character can name it, so this screen is the
// other half of the new-character form rather than a listing for its own sake:
// an author who finds the origin they want missing can add it and go straight
// back without leaving the program.
type originsScreen struct {
	origins []cast.Origin
	counts  map[string]int
	cursor  int

	// adding is whether the form is in front of the catalog.
	adding      bool
	inputs      []textinput.Model
	mediumIndex int
	field       int
	touched     bool

	err error
	// added is the last work written, kept as what it was rather than as the
	// line announcing it, so a language switch redraws the announcement.
	added *cast.Origin
}

func newOriginsScreen(lib *forge.Library) originsScreen {
	return originsScreen{}.refresh(lib).resetForm()
}

func (o originsScreen) refresh(lib *forge.Library) originsScreen {
	o.origins = lib.Origins().All()
	o.counts = make(map[string]int, len(o.origins))
	for _, origin := range o.origins {
		// A map keyed by id, read by key. Nothing here ranges over it into an
		// ordered output; the order on screen is the book's.
		o.counts[origin.ID] = len(lib.Characters().OfOrigin(origin.ID))
	}
	o.cursor = clamp(o.cursor, 0, len(o.origins)-1)
	if o.inputs == nil {
		o = o.resetForm()
	}
	return o
}

func (o originsScreen) resetForm() originsScreen {
	o.inputs = make([]textinput.Model, originFieldCount)
	for i := range o.inputs {
		input := newInput()
		input.Prompt = ""
		input.CharLimit = 200
		input.SetWidth(44)
		o.inputs[i] = input
	}
	o.field = originFieldID
	o.mediumIndex = 0
	o.touched = false
	o.err = nil
	o.inputs[o.field].Focus()
	return o
}

func (o originsScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if o.adding {
		return o.updateForm(m, message)
	}
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		o.cursor = clamp(o.cursor-1, 0, len(o.origins)-1)
	case "down", "j":
		o.cursor = clamp(o.cursor+1, 0, len(o.origins)-1)
	case "a":
		o = o.resetForm()
		o.adding = true
		o.added = nil
	}
	m.origins = o
	return m, nil
}

func (o originsScreen) updateForm(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Before the switch, because saving answers to more than one keystroke and
	// isSaveKey is the single declaration of which.
	if isSaveKey(message) {
		o = o.save(m)
		m.origins = o
		return m, nil
	}
	switch message.String() {
	case "esc":
		if !o.touched {
			o.adding = false
			m.origins = o
			return m, nil
		}
		return m.ask(i18n.OriginFormDiscard, func(m model) model {
			m.origins = m.origins.resetForm()
			m.origins.adding = false
			return m
		}), nil
	case "up", "shift+tab":
		o = o.moveTo(o.field - 1)
		m.origins = o
		return m, nil
	case "down", "tab", "enter":
		o = o.moveTo(o.field + 1)
		m.origins = o
		return m, nil
	}
	if o.field == originFieldMedium {
		mediums := cast.MediumNames()
		switch message.String() {
		case "left":
			o.mediumIndex = (o.mediumIndex - 1 + len(mediums)) % len(mediums)
			o.touched = true
		case "right":
			o.mediumIndex = (o.mediumIndex + 1) % len(mediums)
			o.touched = true
		}
		m.origins = o
		return m, nil
	}
	updated, command := o.inputs[o.field].Update(message)
	if updated.Value() != o.inputs[o.field].Value() {
		o.touched = true
		o.err = nil
	}
	o.inputs[o.field] = updated
	m.origins = o
	return m, command
}

func (o originsScreen) moveTo(target int) originsScreen {
	o.inputs[o.field].Blur()
	o.field = (target + originFieldCount) % originFieldCount
	if o.field != originFieldMedium {
		o.inputs[o.field].Focus()
	}
	return o
}

// save writes the work through forge.Library.SaveOrigin, which validates it
// exactly as a load would and replaces the file atomically.
func (o originsScreen) save(m model) originsScreen {
	// The year's refusal is forge.ParseYear's, like every other refusal on this
	// screen: a rule worded in a front-end is a rule declared twice.
	year, err := forge.ParseYear(o.inputs[originFieldYear].Value())
	if err != nil {
		o.err = err
		return o
	}
	medium, err := cast.ParseMedium(cast.MediumNames()[o.mediumIndex])
	if err != nil {
		o.err = err
		return o
	}
	origin := cast.Origin{
		ID:     strings.TrimSpace(o.inputs[originFieldID].Value()),
		Title:  strings.TrimSpace(o.inputs[originFieldTitle].Value()),
		Medium: medium,
		Year:   year,
		Note:   strings.TrimSpace(o.inputs[originFieldNote].Value()),
	}
	if err := m.lib.SaveOrigin(origin); err != nil {
		o.err = err
		return o
	}
	o = o.refresh(m.lib).resetForm()
	o.adding = false
	o.added = &origin
	return o
}

func (o originsScreen) view(m model) (string, string) {
	if o.adding {
		return o.viewForm(m)
	}
	footer := m.text(i18n.OriginsFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.OriginsHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.OriginsSubtitle)) + "\n\n")
	if len(o.origins) == 0 {
		out.WriteString("  " + m.text(i18n.OriginsEmpty) + "\n")
		return out.String(), footer
	}
	counted := originsCountWidth(m)
	for i, origin := range o.origins {
		marker := "  "
		// Blank rather than a figure when nobody recorded one, which is what
		// makes a zero year readable: pad below gives the cell its five cells
		// either way, so the guard is about the word and not the width.
		// Blank rather than a figure when nobody recorded one, which is what
		// makes a zero year readable: pad below gives the cell its five cells
		// either way, so the guard is about the word and not the width.
		year := ""
		if origin.Year != 0 {
			year = strconv.Itoa(origin.Year)
		}
		row := fmt.Sprintf("%s %s %s %s  %s",
			pad(origin.ID, 16), pad(origin.Medium.String(), 7), pad(year, 5),
			pad(m.text(i18n.OriginsCastCount, o.counts[origin.ID]), counted), origin.Title)
		if i == o.cursor {
			marker = "> "
			row = m.style.selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}
	if selected := o.origins[clamp(o.cursor, 0, len(o.origins)-1)]; selected.Note != "" {
		out.WriteString("\n" + m.label(m.text(i18n.LabelNote), "%s", selected.Note))
	}
	if o.added != nil {
		out.WriteString("\n" + m.style.good.Render(m.text(i18n.OriginAdded,
			o.added.ID, o.added.Medium, m.lib.OriginsPath())) + "\n")
	}
	out.WriteString("\n" + m.style.dim.Render(m.text(i18n.OriginsTally,
		len(o.origins), strings.Join(cast.MediumNames(), " "))))
	return out.String(), footer
}

// originsCountWidth is the column the "how many characters" cell sits in. It is
// measured rather than declared because the cell is a counted noun — two words
// in Vietnamese, one in English — and the widest count is the whole cast.
func originsCountWidth(m model) int {
	widest := 0
	for _, origin := range m.lib.Origins().All() {
		width := lipgloss.Width(m.text(i18n.OriginsCastCount,
			len(m.lib.Characters().OfOrigin(origin.ID))))
		if width > widest {
			widest = width
		}
	}
	return widest
}

// originFieldLabel is what each row of the add-a-work form is called.
func originFieldLabel(m model, field int) string {
	keys := [originFieldCount]i18n.Key{
		originFieldID:     i18n.OriginFieldID,
		originFieldTitle:  i18n.OriginFieldTitle,
		originFieldMedium: i18n.OriginFieldMedium,
		originFieldYear:   i18n.OriginFieldYear,
		originFieldNote:   i18n.OriginFieldNote,
	}
	return m.text(keys[field])
}

func (o originsScreen) viewForm(m model) (string, string) {
	footer := m.text(i18n.OriginFormFooter, saveKeyLabel())
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.OriginFormHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.OriginFormSubtitle)) + "\n\n")
	width := 0
	for field := range originFieldCount {
		if measured := lipgloss.Width(originFieldLabel(m, field)); measured > width {
			width = measured
		}
	}
	width++
	for field := range originFieldCount {
		marker := "  "
		if field == o.field {
			marker = "> "
		}
		name := pad(originFieldLabel(m, field), width)
		if field == o.field {
			name = m.style.selected.Render(name)
		} else {
			name = m.style.label.Render(name)
		}
		value := o.inputs[field].View()
		if field == originFieldMedium {
			mediums := cast.MediumNames()
			value = fmt.Sprintf("< %s >  %s", mediums[o.mediumIndex],
				m.style.dim.Render(m.text(i18n.ChoicePosition, o.mediumIndex+1, len(mediums))))
		}
		out.WriteString(marker + name + " " + value + "\n")
	}
	out.WriteString("\n" + m.style.dim.Render(m.text(i18n.OriginFormHint)) + "\n")
	if o.err != nil {
		out.WriteString("\n" +
			m.style.bad.Render(m.text(i18n.AddRefused, m.lang.Error(o.err))) + "\n")
	}
	return out.String(), footer
}
