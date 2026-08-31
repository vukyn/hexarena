package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// statusRow is one line of the listing: either a category heading or a status
// filed under it.
//
// Headings are rows rather than something drawn between them, because the
// listing scrolls: a heading computed while rendering would fall off the top of
// the window and leave the rows under it with nothing saying what they are.
type statusRow struct {
	heading  bool
	category status.Category
	kind     status.Kind
}

// statusesScreen is the reference for the timed effects: what each buff and
// debuff does, how long it lasts and how many will layer.
//
// It is the sixth listing and the one that was missing. The other five — works,
// species, presets, traits, skills — each name a status somewhere in their rows,
// and until this existed the only way to find out what one of those names meant
// was to open statuses.json. A skill's own description has the same hole from the
// other side: it says *inflicts mire, 70% of the time*, which tells a reader that
// something will happen and not what.
//
// Read-only, unlike the origins and skills listings. A status carries modifier
// terms and a tick power, which are balance rather than content: adding one is a
// change to what every skill inflicting it does, and that belongs in a diff
// rather than behind a form. This screen exists to be read.
type statusesScreen struct {
	rows []statusRow
	// cursor indexes rows and is always on a status. Nothing is learnt by
	// selecting a heading, and a cursor that could land on one would make the
	// description below blink out for a keystroke.
	cursor int
	// from is the screen esc goes back to, and it is a field rather than a
	// constant because this listing is now reachable two ways: from the menu,
	// where back is the menu, and from a trait that named a status, where back
	// is the trait the reader was in the middle of. A reader sent here by one
	// keystroke expects the next one to undo it.
	from screen
}

func newStatusesScreen(lib *forge.Library) statusesScreen {
	return statusesScreen{}.refresh(lib)
}

func (s statusesScreen) refresh(lib *forge.Library) statusesScreen {
	s.rows = nil
	for _, group := range lib.Statuses().Grouped() {
		s.rows = append(s.rows, statusRow{heading: true, category: group.Category})
		for _, kind := range group.Kinds {
			s.rows = append(s.rows, statusRow{category: group.Category, kind: kind})
		}
	}
	s.cursor = s.settle(clamp(s.cursor, 0, len(s.rows)-1), 1)
	return s
}

// settle walks from a row to the nearest status in the given direction, and
// turns round at the end rather than sitting on a heading.
//
// The turn matters for the first row of all, which is always a heading: a cursor
// starting at nought and only ever stepping forward would be right, and one
// starting at nought after a book that lost its last category would not.
func (s statusesScreen) settle(from, step int) int {
	for index := from; index >= 0 && index < len(s.rows); index += step {
		if !s.rows[index].heading {
			return index
		}
	}
	for index := from; index >= 0 && index < len(s.rows); index -= step {
		if !s.rows[index].heading {
			return index
		}
	}
	return from
}

// focus puts the cursor on a named status and reports whether it found one.
//
// It reports rather than settling for the nearest, because the caller is acting
// on a name it read out of a trait: a book that has lost that status is a book
// where the trait's own description is printing a bare id, and moving the cursor
// to whatever sorted next would answer a question nobody asked.
func (s statusesScreen) focus(id string) (statusesScreen, bool) {
	for index, row := range s.rows {
		if row.heading || row.kind.ID != id {
			continue
		}
		s.cursor = index
		return s, true
	}
	return s, false
}

// move steps to the next status in a direction, or stays put at the end of the
// list. It steps over headings rather than through them, so one keypress is one
// status however many category boundaries lie between.
func (s statusesScreen) move(step int) statusesScreen {
	for index := s.cursor + step; index >= 0 && index < len(s.rows); index += step {
		if !s.rows[index].heading {
			s.cursor = index
			return s
		}
	}
	return s
}

func (s statusesScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		// The way back is forgotten as it is used, and the assignment below is
		// what stores that — an early return here would leave the next visit
		// inheriting this one's history.
		m.screen = s.from
		s.from = screenMenu
		m.statuses = s
		return m, nil
	case "up", "k":
		s = s.move(-1)
	case "down", "j":
		s = s.move(1)
	}
	m.statuses = s
	return m, nil
}

// statusesRoom is how many rows the listing may draw: what the window has, less
// the heading pair above it and the description and caveat below.
//
// The description is measured at its longest — three lines, which is what a
// damage-over-time takes — rather than at the height of the one under the
// cursor. A room that shrank and grew as the cursor moved would slide the whole
// listing up and down under it, and a reader would lose their place walking
// between two statuses rather than reading either.
func statusesRoom(m model) int {
	const (
		above = 2 // the heading and the blank line under it
		below = 6 // a blank, the three-line description, a blank, the caveat
	)
	room := m.height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

func (s statusesScreen) view(m model) (string, string) {
	footer := m.text(i18n.StatusesFooter)
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.StatusesHeading)) + "  " +
		m.style.Dim.Render(m.text(i18n.StatusesSubtitle)) + "\n\n")
	if len(s.rows) == 0 {
		out.WriteString("  " + m.text(i18n.StatusesEmpty) + "\n")
		return out.String(), footer
	}

	// The category names are measured with the ids, because a heading sits in
	// the same column as the rows under it: stat_debuff is wider than every id
	// in the book, and a column measured from the ids alone pushed that one
	// heading's wording a cell right of every other.
	column := 0
	for _, row := range s.rows {
		width := lipgloss.Width(row.kind.ID)
		if row.heading {
			width = lipgloss.Width(row.category.String())
		}
		if width > column {
			column = width
		}
	}
	from, to := window(len(s.rows), s.cursor, statusesRoom(m))
	for index := from; index < to; index++ {
		row := s.rows[index]
		if row.heading {
			out.WriteString("  " + m.style.Dim.Render(
				pad(row.category.String(), column+1)+" "+
					m.lang.StatusCategory(row.category.String())) + "\n")
			continue
		}
		// The id and the name it is printed under, and nothing else: every
		// figure a row could carry is on the description below, and a status
		// with a duration column and a stack column beside it would be the same
		// two numbers twice on one screen.
		line := pad(row.kind.ID, column+1) + " " + m.lang.Gloss(row.kind.ID)
		marker := "  "
		if index == s.cursor {
			marker = "> "
			line = m.style.Selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}

	out.WriteString("\n")
	selected := s.rows[clamp(s.cursor, 0, len(s.rows)-1)]
	for _, line := range strings.Split(m.lang.DescribeStatus(selected.kind), "\n") {
		out.WriteString("  " + line + "\n")
	}
	// Once, at the foot, rather than as a line of every description: it is true
	// of all of them, and a warning repeated under every row is a warning nobody
	// finishes reading.
	// No newline after it: the frame pads the body out to the window, and a
	// trailing one is a twenty-first line in a twenty-line body — which costs
	// the caveat itself, since the frame cuts from the bottom.
	out.WriteString("\n  " + m.style.Dim.Render(m.text(i18n.BlurbStatusCaveat)))
	return out.String(), footer
}
