package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// speciesScreen is what a character can *be*, with the skills that fact unlocks
// under the cursor.
//
// It is the other half of the restriction column the skills listing draws. That
// column says "chủng loài dragon" and had nowhere to go: a species was reachable
// only through the picker on the new-skill form, so a reader who was not in the
// middle of authoring a restriction could not find out what the word covered,
// and the note the author wrote beside it reached nobody at all.
//
// The column that earns its place is **who is one**, the same shape the trait
// listing's carriers column has and for the same reason: a species is not
// restricted, it is claimed, so "who may be one" is everybody and answers
// nothing. A species nobody is is not an error — a kind may be written before
// the character that fills it, exactly as a trait may — but it is a gate that
// cannot open, and a listing is where that shows.
//
// Read-only, unlike the origins listing. A species is a word plus a note today,
// but the gate it drives is a skill's allowlist, so adding one is only ever half
// a change: the other half is the skill kept for it, and that is authored on the
// skills form. A form here would offer to make a kind nothing can use.
type speciesScreen struct {
	kinds []cast.Species
	// members and skills are keyed by species id and read by key only; nothing
	// ranges over either into an ordered output, so neither can reach a rendered
	// line out of order.
	members map[string]string
	skills  map[string]string
	cursor  int
}

func newSpeciesScreen(lib *forge.Library) speciesScreen {
	return speciesScreen{}.refresh(lib)
}

func (s speciesScreen) refresh(lib *forge.Library) speciesScreen {
	s.kinds = lib.Species().All()
	s.members = make(map[string]string, len(s.kinds))
	s.skills = make(map[string]string, len(s.kinds))
	for _, kind := range s.kinds {
		names := make([]string, 0, 4)
		for _, character := range lib.Characters().OfSpecies(kind.ID) {
			names = append(names, character.ID)
		}
		s.members[kind.ID] = strings.Join(names, " ")
		s.skills[kind.ID] = strings.Join(lib.SkillsForSpecies(kind.ID), " ")
	}
	s.cursor = clamp(s.cursor, 0, len(s.kinds)-1)
	return s
}

func (s speciesScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		s.cursor = clamp(s.cursor-1, 0, len(s.kinds)-1)
	case "down", "j":
		s.cursor = clamp(s.cursor+1, 0, len(s.kinds)-1)
	}
	m.species = s
	return m, nil
}

// speciesRoom is how many rows the listing may draw: the window less the heading
// pair above and the note, the skills line and the empty-kind line below.
//
// All of the lines below are reserved whether or not the kind under the cursor
// draws them, for the reason every other listing reserves its worst case: a room
// that grew on a kind with no note would move the rows under the reader as they
// walked the list.
//
// The note is measured rather than counted as one line, and that is the half a
// constant gets wrong. A note is authored prose of no fixed length — it is the
// one place in this book somebody writes a sentence — so it wraps, and a reserve
// of one line for a note that takes three lets the body overrun the window. The
// frame cuts from the bottom, so what an overrun costs is the skills line: the
// derived half of the pane, silently gone, on exactly the kinds whose note is
// longest.
func speciesRoom(m model, s speciesScreen) int {
	const (
		above = 2 // the heading and the blank line under it
		other = 4 // a blank, the skills line, a blank, the empty-kind line
	)
	room := m.height - 4 - above - other - s.longestNote()
	if room < 3 {
		return 3
	}
	return room
}

// longestNote is the tallest the note pane gets over the whole book, in lines as
// the pane wraps them.
//
// The whole book rather than the kind under the cursor, because that is what
// makes the listing hold still: a reserve that tracked the cursor would give the
// rows one height per kind and slide them as a reader walked down.
func (s speciesScreen) longestNote() int {
	most := 1
	for _, kind := range s.kinds {
		if lines := len(wrapWords(kind.Note, minWidth-3)); lines > most {
			most = lines
		}
	}
	return most
}

// speciesRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart — the arrangement passiveRow and skillRow
// both have, and for the same reason.
//
// A nameColumn of zero drops the name column entirely, which is what English
// gets. The word beside the id is a **data** name — a field on the declaration,
// authored once and in Vietnamese — rather than a compiled gloss that is empty
// in the other language by construction, so drawing it in English would be a
// leak rather than a translation. Dropped rather than blanked, because a column
// of empty cells reads as data the book has lost.
func speciesRow(idColumn, nameColumn int, id, name, members string) string {
	row := pad(id, idColumn)
	if nameColumn > 0 {
		row += " " + pad(name, nameColumn)
	}
	return row + " " + members
}

func (s speciesScreen) view(m model) (string, string) {
	footer := m.text(i18n.SpeciesFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.SpeciesHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.SpeciesSubtitle)) + "\n\n")
	if len(s.kinds) == 0 {
		out.WriteString("  " + m.text(i18n.SpeciesEmpty) + "\n")
		return out.String(), footer
	}

	column, nameColumn := 0, 0
	for _, kind := range s.kinds {
		if width := lipgloss.Width(kind.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(m.lang.SpeciesName(kind)); width > nameColumn {
			nameColumn = width
		}
	}
	// Each header has to fit the column it names, the same rule the other two
	// listings follow: a header wider than its rows pushes the one beside it off
	// its own column. The name header is measured only when there is a column
	// under it, or English would draw a header over nothing.
	if width := lipgloss.Width(m.text(i18n.SkillFieldID)); width > column {
		column = width
	}
	if nameColumn > 0 {
		if width := lipgloss.Width(m.text(i18n.ColumnGloss)); width > nameColumn {
			nameColumn = width
		}
	}
	out.WriteString("  " + m.style.dim.Render(speciesRow(column+1, nameColumn,
		m.text(i18n.SkillFieldID), m.text(i18n.ColumnGloss),
		m.text(i18n.ColumnWhoIs))) + "\n")

	from, to := window(len(s.kinds), s.cursor, speciesRoom(m, s))
	for index := from; index < to; index++ {
		kind := s.kinds[index]
		// Against the window rather than the floor, as the skills and traits
		// listings measure their last column: the members cell is data, and
		// cutting it on a wide terminal hides which characters are one.
		row := clip(speciesRow(column+1, nameColumn, kind.ID, m.lang.SpeciesName(kind),
			s.members[kind.ID]), m.usableWidth()-3)
		marker := "  "
		if index == s.cursor {
			marker = "> "
			row = m.style.selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}

	out.WriteString("\n")
	selected := s.kinds[clamp(s.cursor, 0, len(s.kinds)-1)]
	// The note is authored prose, so it wraps at the floor rather than at the
	// window: a sentence run across a two-hundred-column terminal is a line a
	// reader loses their place in. The kinds that have none say so rather than
	// leaving the pane blank, which reads as the tool failing to answer.
	note := selected.Note
	if strings.TrimSpace(note) == "" {
		note = m.text(i18n.SpeciesNoNote)
	}
	for _, line := range wrapWords(note, minWidth-3) {
		out.WriteString("  " + line + "\n")
	}
	// What being this kind unlocks, and only when it unlocks something: a kind no
	// skill is kept for is a gate nothing is behind, and an empty list after the
	// colon reads as a lookup that failed rather than as an answer.
	if kept := s.skills[selected.ID]; kept != "" {
		out.WriteString("  " + m.text(i18n.SpeciesKeptSkills, kept) + "\n")
	}
	// And the row above says "nobody is one" with an empty cell, which reads as a
	// column that failed to fill rather than as a fact. This says it in words, and
	// only for the kind being read.
	if s.members[selected.ID] == "" {
		out.WriteString("\n  " + m.style.dim.Render(m.text(i18n.SpeciesNobodyIs)))
	}
	return strings.TrimRight(out.String(), "\n"), footer
}
