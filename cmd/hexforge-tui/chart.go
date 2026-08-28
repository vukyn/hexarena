package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

// chartScreen is the affinity chart drawn as the rings it was declared in.
//
// The elements listing answers "what does fire lose to" one element at a time,
// which is the question somebody has with a skill in front of them. This answers
// the other one: what is the *shape*. Eleven rows of two strengths and two
// weaknesses is twenty-two facts to hold; three rings and a pair is four, and
// the rings are what the chart was written as — see the package doc on
// internal/core/element, which draws them the same way for the same reason.
//
// A screen of its own rather than a block under the listing. Both halves want
// the height: the listing is eleven rows plus a description, the rings are five
// blocks plus their rates, and a window at the floor cannot hold the two. It is
// raised from the listing, the way the description screen is raised from the
// skills listing, because that is where a reader is when the question occurs to
// them.
//
// No cursor. Nothing here is selected — the whole point is that it is read at
// once, and a cursor would invite a keystroke that has nothing to do.
type chartScreen struct{}

func (s chartScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		// Back to the listing rather than to the menu: this screen is only
		// reachable from there, and a reader who pressed one key to arrive
		// expects one key to undo it.
		m.screen = screenElements
	}
	return m, nil
}

// The notation, which is internal/core/element's package doc's own: "a" beats
// the element after it, and the ring closes on the name in brackets.
//
// ASCII rather than the arrows this would obviously like to use. → and ⇄ are
// East-Asian *ambiguous* width: a terminal that draws them two cells wide while
// the program measures them as one leaves every column after them one cell out,
// and this repository has already been bitten by exactly that with the modifier
// glyphs on a footer. A chart that misaligns on somebody's terminal is worse
// than a chart drawn in plain angle brackets.
const (
	beatsMark  = " > "
	mutualMark = " <> "
)

func (s chartScreen) view(m model) (string, string) {
	footer := m.text(i18n.ChartFooter)
	chart := m.lib.Chart()
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.ChartHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.ChartSubtitle)) + "\n\n")

	rows := chartRows(m, chart)
	if len(rows) == 0 {
		out.WriteString("  " + m.text(i18n.ChartEmpty) + "\n")
		return out.String(), footer
	}
	column := 0
	for _, row := range rows {
		if width := lipgloss.Width(row.label); width > column {
			column = width
		}
	}
	// Every element the chart names, so the colouring can pick them out of a
	// line that has already been wrapped. Word by word inside marked, because a
	// style spanning a wrap opens on one line and closes on the next.
	names := make([]string, 0, element.Count)
	for _, member := range element.All() {
		names = append(names, member.String())
	}
	for _, row := range rows {
		// The chain wraps at the floor rather than at the window, and it breaks
		// on the spaces around the marks, so a ring too long for a line
		// continues under itself instead of running off the edge. A ring is
		// read left to right like a sentence, which is why it wraps like one.
		lines := wrapWords(row.chain, minWidth-3-column-1)
		for index, line := range lines {
			label := ""
			if index == 0 {
				label = row.label
			}
			line = marked(line, names, func(word string) string {
				member, err := element.Parse(word)
				if err != nil {
					return word
				}
				return m.style.element(member).Render(word)
			})
			out.WriteString("  " + m.style.label.Render(pad(label, column)) + " " + line + "\n")
		}
	}

	// The three figures the whole picture is worth, once at the foot. They are on
	// every element's own description too, but a reader looking at the shape is
	// asking what an edge is *for*, and an edge with no price on it is a line
	// between two words.
	rates := chart.Multipliers()
	out.WriteString("\n  " + m.style.dim.Render(m.text(i18n.ChartRates,
		i18n.Share(rates.Advantage), i18n.Share(rates.Neutral),
		i18n.Share(rates.Disadvantage))))
	return out.String(), footer
}

// chartRow is one line of the picture: a name for the grouping and the chain it
// covers.
type chartRow struct {
	label string
	chain string
}

// chartRows is the whole chart as rows, in the order it is read: the rings as
// declared, then the pairs, then whatever is in neither.
//
// The rings keep their authored names — "organic", "cross" — because those are
// what the author grouped by and the id of a ring is the only handle a reader
// has for one. The last two rows are named by the program, since being a pair
// and being inert are facts about the shape rather than things somebody wrote.
func chartRows(m model, chart *element.Chart) []chartRow {
	rows := make([]chartRow, 0, 6)
	for _, cycle := range chart.Cycles() {
		if len(cycle.Chain) == 0 {
			continue
		}
		names := make([]string, 0, len(cycle.Chain)+1)
		for _, member := range cycle.Chain {
			names = append(names, member.String())
		}
		// The ring closes on the name it opened with, in brackets, so that the
		// last member is visibly beating the first rather than the line simply
		// stopping. A chain that stopped would read as an order of precedence,
		// which is the one thing a ring is not.
		names = append(names, "("+cycle.Chain[0].String()+")")
		rows = append(rows, chartRow{cycle.Name, strings.Join(names, beatsMark)})
	}
	for _, pair := range chart.MutualPairs() {
		rows = append(rows, chartRow{
			m.text(i18n.ChartMutual),
			pair[0].String() + mutualMark + pair[1].String(),
		})
	}
	if inert := chart.Inert(); len(inert) > 0 {
		names := make([]string, 0, len(inert))
		for _, member := range inert {
			names = append(names, member.String())
		}
		rows = append(rows, chartRow{m.text(i18n.ChartInert), strings.Join(names, " ")})
	}
	return rows
}
