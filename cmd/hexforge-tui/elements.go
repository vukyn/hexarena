package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

// elementsScreen is the affinity chart: the eleven elements, and for the one
// under the cursor, what it beats, what beats it, and at what rate.
//
// It is the reference the tool was missing hardest. Every other listing prints
// an element somewhere — a skill's own element, a character's affinity, the
// "hệ grass" in a restriction — and the chart those ids resolve against was
// readable only in elements.json, which spells it as three cycles and a mutual
// pair rather than as "what does fire lose to". That is the one table every
// damage figure in the game passes through, so it is the one an author is most
// often guessing at.
//
// Read-only, like the status and trait references. The chart is balance in its
// purest form: adding an edge changes every battle at once, and the three
// multipliers are the game's whole scale. That belongs in a diff.
//
// The rows are element.All() rather than anything read off the library, because
// the elements are a Go enum: the chart may say nothing about a member — neutral
// is inert on purpose — but it cannot invent a twelfth one, and a listing built
// from the edges would silently drop the element that has none.
type elementsScreen struct {
	cursor int
}

func (s elementsScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	members := element.All()
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
	case "g":
		// The shape of the chart, which this listing answers one row at a time.
		// A key rather than a menu entry: the question "so what beats what" is
		// one a reader has while looking at a row, not one they leave the
		// reference to go and ask.
		m.screen = screenChart
	case "up", "k":
		s.cursor = clamp(s.cursor-1, 0, len(members)-1)
	case "down", "j":
		s.cursor = clamp(s.cursor+1, 0, len(members)-1)
	}
	m.elements = s
	return m, nil
}

// elementsRoom is how many rows the listing may draw: what the window has, less
// the heading pair above it and the description and caveat below.
//
// The description is reserved at its longest — two lines, a strength and a
// weakness — rather than at the height of the one under the cursor. A room that
// shrank on neutral, which describes itself in one line, would slide the whole
// listing under the reader as they walked past it.
func elementsRoom(m model) int {
	const (
		above = 2 // the heading and the blank line under it
		below = 5 // a blank, the two-line description, a blank, the caveat
	)
	room := m.height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

func (s elementsScreen) view(m model) (string, string) {
	footer := m.text(i18n.ElementsFooter)
	members := element.All()
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.ElementsHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.ElementsSubtitle)) + "\n\n")

	// The name column is dropped whole where nothing is named, which is what
	// English gets: an element's gloss is a compiled one and is empty in the
	// other language by construction, so measuring the ids alone would leave
	// eleven rows padded out to a column of blanks — data the book has lost,
	// rather than a column that does not apply. Same rule the traits and species
	// listings follow.
	column, glossColumn := 0, 0
	for _, member := range members {
		if width := lipgloss.Width(member.String()); width > column {
			column = width
		}
		if width := lipgloss.Width(m.lang.Gloss(member.String())); width > glossColumn {
			glossColumn = width
		}
	}
	from, to := window(len(members), clamp(s.cursor, 0, len(members)-1), elementsRoom(m))
	for index := from; index < to; index++ {
		member := members[index]
		// The id and its name, and no third column. A count of edges is the one
		// figure a row could carry, and it is two words of the description
		// directly below — the same reason the status listing prints no numbers.
		// The id in the element's own colour, which is the same colour the chart
		// draws it in: a reader who walks from one screen to the other is
		// following the word, and a word that changes colour on the way is a
		// second word. Decoration only — the id is right there in text.
		id := m.style.element(member).Render(member.String())
		line := id
		if glossColumn > 0 {
			line = id + strings.Repeat(" ", column+1-lipgloss.Width(member.String())) +
				" " + m.lang.Gloss(member.String())
		}
		marker := "  "
		if index == clamp(s.cursor, 0, len(members)-1) {
			marker = "> "
			// The selection is bold and takes the row whole, colour and all: a
			// cursor that recoloured the id would hide the one thing the colour
			// is for.
			line = m.style.selected.Render(member.String())
			if glossColumn > 0 {
				line = m.style.selected.Render(
					pad(member.String(), column+1) + " " + m.lang.Gloss(member.String()))
			}
		}
		out.WriteString(marker + line + "\n")
	}

	out.WriteString("\n")
	selected := members[clamp(s.cursor, 0, len(members)-1)]
	for _, line := range strings.Split(m.lang.DescribeElement(selected, m.lib.Chart()), "\n") {
		out.WriteString("  " + line + "\n")
	}
	// Once, at the foot, for the reason the status caveat sits there: a dual
	// affinity multiplies both of its halves, which is true of every row here and
	// of none of them individually.
	// No newline after it — the frame pads the body out and cuts from the bottom,
	// so a trailing one costs the caveat itself.
	out.WriteString("\n  " + m.style.dim.Render(m.text(i18n.BlurbElementCaveat)))
	return out.String(), footer
}
