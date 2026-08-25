package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
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

	err   error
	added string
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
		input := textinput.New()
		input.Prompt = ""
		input.CharLimit = 200
		input.Width = 44
		o.inputs[i] = input
	}
	o.field = originFieldID
	o.mediumIndex = 0
	o.touched = false
	o.err = nil
	o.inputs[o.field].Focus()
	return o
}

func (o originsScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		o.added = ""
	}
	m.origins = o
	return m, nil
}

func (o originsScreen) updateForm(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		if !o.touched {
			o.adding = false
			m.origins = o
			return m, nil
		}
		return m.ask("discard the work being added?", func(m model) model {
			m.origins = m.origins.resetForm()
			m.origins.adding = false
			return m
		}), nil
	case "ctrl+s":
		o = o.save(m)
		m.origins = o
		return m, nil
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
	year := 0
	if raw := strings.TrimSpace(o.inputs[originFieldYear].Value()); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			o.err = fmt.Errorf("the year %q is not a number; leave it empty if it is unknown", raw)
			return o
		}
		year = parsed
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
	added := fmt.Sprintf("added %s (%s) to %s", origin.ID, origin.Medium, m.lib.OriginsPath())
	o = o.refresh(m.lib).resetForm()
	o.adding = false
	o.added = added
	return o
}

func (o originsScreen) view(m model) (string, string) {
	if o.adding {
		return o.viewForm(m)
	}
	footer := "↑/↓ move · a add a work · esc back · q quit"
	var out strings.Builder
	out.WriteString(m.style.heading.Render("origins") + "  " +
		m.style.dim.Render("the works the cast is borrowed from") + "\n\n")
	if len(o.origins) == 0 {
		out.WriteString("  no works in the catalog yet. Press a to add one.\n")
		return out.String(), footer
	}
	for i, origin := range o.origins {
		marker := "  "
		year := "     "
		if origin.Year != 0 {
			year = strconv.Itoa(origin.Year)
		}
		row := fmt.Sprintf("%-16s %-7s %-5s %2d cast  %s",
			origin.ID, origin.Medium, year, o.counts[origin.ID], origin.Title)
		if i == o.cursor {
			marker = "> "
			row = m.style.selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}
	if selected := o.origins[clamp(o.cursor, 0, len(o.origins)-1)]; selected.Note != "" {
		out.WriteString("\n" + m.label("note", "%s", selected.Note))
	}
	if o.added != "" {
		out.WriteString("\n" + m.style.good.Render(o.added) + "\n")
	}
	out.WriteString("\n" + m.style.dim.Render(
		fmt.Sprintf("%d works. media: %s", len(o.origins), strings.Join(cast.MediumNames(), " "))))
	return out.String(), footer
}

func (o originsScreen) viewForm(m model) (string, string) {
	footer := "↑/↓ field · ←/→ medium · ctrl+s add · esc back · ctrl+c quit"
	labels := [originFieldCount]string{"id", "title", "medium", "year", "note"}
	var out strings.Builder
	out.WriteString(m.style.heading.Render("add a work") + "  " +
		m.style.dim.Render("a character can only name a work the catalog holds") + "\n\n")
	for field := range originFieldCount {
		marker := "  "
		if field == o.field {
			marker = "> "
		}
		name := fmt.Sprintf("%-8s", labels[field])
		if field == o.field {
			name = m.style.selected.Render(name)
		} else {
			name = m.style.label.Render(name)
		}
		value := o.inputs[field].View()
		if field == originFieldMedium {
			mediums := cast.MediumNames()
			value = fmt.Sprintf("< %s >  %s", mediums[o.mediumIndex],
				m.style.dim.Render(fmt.Sprintf("%d of %d", o.mediumIndex+1, len(mediums))))
		}
		out.WriteString(marker + name + " " + value + "\n")
	}
	out.WriteString("\n" + m.style.dim.Render(
		"the year may be left empty when it is unknown; the note is free text") + "\n")
	if o.err != nil {
		out.WriteString("\n" + m.style.bad.Render("cannot add: "+o.err.Error()) + "\n")
	}
	return out.String(), footer
}
