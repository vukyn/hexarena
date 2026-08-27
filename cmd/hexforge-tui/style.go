package main

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// palette is every style the screens draw with.
//
// Colour is decoration and never information. A missing file says "MISSING", a
// kit an affinity cannot carry says which skill it cannot carry, a selected row
// carries a "> " marker, and a bar prints the numbers beside it — so the whole
// program reads on a monochrome terminal, through a screen reader, and in a
// terminal recording that lost its escape codes. That is not politeness: the
// one thing this client exists to show is whether a character is legal, and an
// answer nobody can read is not an answer.
type palette struct {
	title    lipgloss.Style
	heading  lipgloss.Style
	label    lipgloss.Style
	dim      lipgloss.Style
	good     lipgloss.Style
	bad      lipgloss.Style
	selected lipgloss.Style
	// emphasis marks a data name inside the program's own prose. Bold and no
	// colour, because it is standing in a sentence rather than in a column, and
	// a coloured word mid-paragraph reads as a link to somewhere the terminal
	// cannot take you.
	emphasis lipgloss.Style
	footer   lipgloss.Style
}

// newPalette picks the styles for the terminal in front of it.
//
// NO_COLOR and a dumb TERM both mean "write plain text". The check is explicit
// rather than left to the rendering library so that the behaviour is this
// program's own, testable, and does not change under it when a dependency
// revises its own idea of when colour is welcome.
func newPalette() palette {
	if plainTerminal() {
		plain := lipgloss.NewStyle()
		return palette{
			title: plain, heading: plain, label: plain, dim: plain,
			good: plain, bad: plain, selected: plain, emphasis: plain,
			footer: plain,
		}
	}
	return palette{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		heading:  lipgloss.NewStyle().Bold(true),
		label:    lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
		dim:      lipgloss.NewStyle().Faint(true),
		good:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		bad:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")),
		emphasis: lipgloss.NewStyle().Bold(true),
		footer:   lipgloss.NewStyle().Faint(true),
	}
}

// newInput is a text field dressed the way this program draws them.
//
// Two decisions, and both are here rather than at the four call sites so that a
// fifth field cannot be born looking different from the other four.
func newInput() textinput.Model {
	input := textinput.New()
	input.SetStyles(newInputStyles())
	// A virtual cursor is drawn as reverse video, which is an escape code, so a
	// plain terminal gets none. That is not a loss against what shipped: under
	// bubbletea v1 the rendering library stripped the attribute itself, so the
	// plain path never showed a cursor either. What is new is only that the
	// program has to say so.
	input.SetVirtualCursor(!plainTerminal())
	return input
}

// newInputStyles picks the styles a text field draws itself with, on the same
// terms as the palette above.
//
// It exists because bubbletea v2 moved the decision about colour from the
// rendering library to the program: lipgloss now writes escape codes
// unconditionally and the *program* downsamples them for the terminal it is
// attached to. A field left on its own defaults would therefore keep its colours
// under NO_COLOR, which is the one thing this program says it does not do — and
// the palette's rule is only worth having if every drawn thing obeys it.
//
// Under v1 nothing had to say this: the library detected the absence of a
// terminal and wrote plain text by itself. That is exactly why it is written
// down now — a behaviour that used to be free is a behaviour that will be lost
// again if nobody names it.
func newInputStyles() textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	if !plainTerminal() {
		return styles
	}
	plain := lipgloss.NewStyle()
	blank := textinput.StyleState{
		Text: plain, Placeholder: plain, Suggestion: plain, Prompt: plain,
	}
	styles.Focused, styles.Blurred = blank, blank
	styles.Cursor.Color = nil
	return styles
}

// plainTerminal reports whether colour would be noise rather than help.
func plainTerminal() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	switch os.Getenv("TERM") {
	case "", "dumb":
		return true
	}
	return false
}

// How wide the two meters are drawn. Both are fixed rather than proportional
// to the window: a meter that changes width as the terminal is resized cannot
// be compared against the one drawn a moment ago.
const (
	budgetBarWidth = 24
	statBarWidth   = 12
)

// bar draws a proportion in characters, never in colour alone.
//
// The caller prints the numbers next to it. A value over its limit fills the
// whole bar and is marked with a '!' so that "full" and "too much" are not the
// same picture — being over the budget is the single thing an author most needs
// to notice, and a bar that saturates silently hides exactly that.
func bar(barWidth int, value, limit int64) string {
	if limit <= 0 {
		return strings.Repeat("-", barWidth)
	}
	over := value > limit
	if over {
		value = limit
	}
	if value < 0 {
		value = 0
	}
	filled := int(value * int64(barWidth) / limit)
	if filled > barWidth {
		filled = barWidth
	}
	mark := "="
	if over {
		mark = "!"
	}
	return "[" + strings.Repeat(mark, filled) + strings.Repeat("-", barWidth-filled) + "]"
}
