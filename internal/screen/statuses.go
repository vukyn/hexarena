package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// StatusRow is one line of the listing: either a category heading or a status
// filed under it.
//
// Headings are rows rather than something drawn between them, because the
// listing scrolls: a heading computed while rendering would fall off the top of
// the window and leave the rows under it with nothing saying what they are.
type StatusRow struct {
	Heading  bool
	Category status.Category
	Kind     status.Kind
}

// StatusesScreen is the reference for the timed effects: what each buff and
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
type StatusesScreen struct {
	Rows []StatusRow
	// cursor indexes rows and is always on a status. Nothing is learnt by
	// selecting a heading, and a cursor that could land on one would make the
	// description below blink out for a keystroke.
	Cursor int
}

// NewStatusesScreen is the reference filled from a library, ready to draw.
func NewStatusesScreen(lib *forge.Library) StatusesScreen {
	return StatusesScreen{}.Refresh(lib)
}

// Refresh re-files every row against a library that may have been written to
// since the screen was last drawn, keeping the cursor on a status.
func (s StatusesScreen) Refresh(lib *forge.Library) StatusesScreen {
	s.Rows = nil
	for _, group := range lib.Statuses().Grouped() {
		s.Rows = append(s.Rows, StatusRow{Heading: true, Category: group.Category})
		for _, kind := range group.Kinds {
			s.Rows = append(s.Rows, StatusRow{Category: group.Category, Kind: kind})
		}
	}
	s.Cursor = s.settle(Clamp(s.Cursor, 0, len(s.Rows)-1), 1)
	return s
}

// settle walks from a row to the nearest status in the given direction, and
// turns round at the end rather than sitting on a heading.
//
// The turn matters for the first row of all, which is always a heading: a cursor
// starting at nought and only ever stepping forward would be right, and one
// starting at nought after a book that lost its last category would not.
func (s StatusesScreen) settle(from, step int) int {
	for index := from; index >= 0 && index < len(s.Rows); index += step {
		if !s.Rows[index].Heading {
			return index
		}
	}
	for index := from; index >= 0 && index < len(s.Rows); index -= step {
		if !s.Rows[index].Heading {
			return index
		}
	}
	return from
}

// Focus puts the cursor on a named status and reports whether it found one.
//
// It reports rather than settling for the nearest, because the caller is acting
// on a name it read out of a trait: a book that has lost that status is a book
// where the trait's own description is printing a bare id, and moving the cursor
// to whatever sorted next would answer a question nobody asked.
func (s StatusesScreen) Focus(id string) (StatusesScreen, bool) {
	for index, row := range s.Rows {
		if row.Heading || row.Kind.ID != id {
			continue
		}
		s.Cursor = index
		return s, true
	}
	return s, false
}

// move steps to the next status in a direction, or stays put at the end of the
// list. It steps over headings rather than through them, so one keypress is one
// status however many category boundaries lie between.
func (s StatusesScreen) move(step int) StatusesScreen {
	for index := s.Cursor + step; index >= 0 && index < len(s.Rows); index += step {
		if !s.Rows[index].Heading {
			s.Cursor = index
			return s
		}
	}
	return s
}

// Update reads one keystroke: the cursor steps over the headings, or the reader
// leaves.
func (s StatusesScreen) Update(_ Context, message tea.KeyPressMsg) (StatusesScreen, Action) {
	switch message.String() {
	case "q":
		return s, Action{Kind: Quit}
	case "esc":
		// Back, and where that is comes off the client's one-slot memory of who
		// raised this screen. This listing is reachable two ways — from the menu,
		// where back is the menu, and from a trait that named a status, where
		// back is the trait the reader was in the middle of — and it used to
		// carry that answer itself, in a `from screen` field it cleared here as
		// it used it. The clearing moved with the memory; the semantics did not,
		// because the slot defaults to the menu exactly as an unset field did.
		return s, Action{Kind: Back}
	case "up", "k":
		s = s.move(-1)
	case "down", "j":
		s = s.move(1)
	}
	return s, Action{}
}

// statusesRoom is how many rows the listing may draw: what the window has, less
// the heading pair above it and the description and caveat below.
//
// The description is measured at its longest — three lines, which is what a
// damage-over-time takes — rather than at the height of the one under the
// cursor. A room that shrank and grew as the cursor moved would slide the whole
// listing up and down under it, and a reader would lose their place walking
// between two statuses rather than reading either.
func statusesRoom(c Context) int {
	const (
		above = 2 // the heading and the blank line under it
		below = 6 // a blank, the three-line description, a blank, the caveat
	)
	room := c.Height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

// View draws the grouped listing, the description of the status under the
// cursor, the caveat under both, and the footer.
func (s StatusesScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.StatusesFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.StatusesHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.StatusesSubtitle)) + "\n\n")
	if len(s.Rows) == 0 {
		out.WriteString("  " + c.Text(i18n.StatusesEmpty) + "\n")
		return out.String(), footer
	}

	// The category names are measured with the ids, because a heading sits in
	// the same column as the rows under it: stat_debuff is wider than every id
	// in the book, and a column measured from the ids alone pushed that one
	// heading's wording a cell right of every other.
	column := 0
	for _, row := range s.Rows {
		width := lipgloss.Width(row.Kind.ID)
		if row.Heading {
			width = lipgloss.Width(row.Category.String())
		}
		if width > column {
			column = width
		}
	}
	from, to := Window(len(s.Rows), s.Cursor, statusesRoom(c))
	for index := from; index < to; index++ {
		row := s.Rows[index]
		if row.Heading {
			out.WriteString("  " + c.Style.Dim.Render(
				Pad(row.Category.String(), column+1)+" "+
					c.Lang.StatusCategory(row.Category.String())) + "\n")
			continue
		}
		// The id and the name it is printed under, and nothing else: every
		// figure a row could carry is on the description below, and a status
		// with a duration column and a stack column beside it would be the same
		// two numbers twice on one screen.
		line := Pad(row.Kind.ID, column+1) + " " + c.Lang.Gloss(row.Kind.ID)
		marker := "  "
		if index == s.Cursor {
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}

	out.WriteString("\n")
	selected := s.Rows[Clamp(s.Cursor, 0, len(s.Rows)-1)]
	for _, line := range strings.Split(c.Lang.DescribeStatus(selected.Kind), "\n") {
		out.WriteString("  " + line + "\n")
	}
	// Once, at the foot, rather than as a line of every description: it is true
	// of all of them, and a warning repeated under every row is a warning nobody
	// finishes reading.
	// No newline after it: the frame pads the body out to the window, and a
	// trailing one is a twenty-first line in a twenty-line body — which costs
	// the caveat itself, since the frame cuts from the bottom.
	out.WriteString("\n  " + c.Style.Dim.Render(c.Text(i18n.BlurbStatusCaveat)))
	return out.String(), footer
}
