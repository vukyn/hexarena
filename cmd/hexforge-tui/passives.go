package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// passivesScreen is the declared traits, with what each one does under the
// cursor.
//
// It answers a different question from the one the cast browser's `?` answers,
// which is why both exist. That one asks "what is this character carrying" and
// is filtered by a level; this one asks "what traits are there", which is the
// question somebody has before they know which character to look at — and the
// one nothing in the tool could answer, because a trait was only ever reachable
// through a character that already had it.
//
// The column that earns its place is **who carries it**. A trait has no
// restriction mechanism at all, so "who may carry this" is everybody and answers
// nothing; who actually *does* is the fact worth a column, and a trait nobody
// learns is a trait that cannot reach a battle. That is not an error — a catalog
// may be written before the cast that fills it — but it is the sort of thing a
// listing is for.
//
// Read-only, like the status reference and unlike the skills listing. A trait
// carries modifier terms and shares of damage, which is balance rather than
// content: adding one changes what every character holding it does, and that
// belongs in a diff rather than behind a form. Nothing in either front-end
// authors a trait today.
type passivesScreen struct {
	passives []passive.Passive
	// carriers is keyed by trait id and read by key only; nothing ranges over it
	// into an ordered output, so it cannot reach a rendered line out of order.
	carriers map[string]string
	cursor   int
}

func newPassivesScreen(lib *forge.Library) passivesScreen {
	return passivesScreen{}.refresh(lib)
}

func (p passivesScreen) refresh(lib *forge.Library) passivesScreen {
	p.passives = lib.Passives().All()
	p.carriers = make(map[string]string, len(p.passives))
	for _, held := range p.passives {
		p.carriers[held.ID] = forge.TraitCarrierSummary(lib.TraitCarriers(held.ID))
	}
	p.cursor = clamp(p.cursor, 0, len(p.passives)-1)
	return p
}

func (p passivesScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		p.cursor = clamp(p.cursor-1, 0, len(p.passives)-1)
	case "down", "j":
		p.cursor = clamp(p.cursor+1, 0, len(p.passives)-1)
	}
	m.passives = p
	return m, nil
}

// passivesRoom is how many rows the listing may draw: the window, less the two
// the heading takes and the six the description below it may.
//
// Six is the most a trait's description runs to — a flavour clause plus the five
// jobs the busiest shipped trait holds — and it is measured at the most rather
// than at the height of the one under the cursor. A room that shrank and grew
// with the cursor would slide the listing up and down under it, and a reader
// would lose their place walking between two traits rather than reading either.
func passivesRoom(m model) int {
	const (
		above = 2 // the heading and the blank line under it
		below = 8 // a blank, the trait's name, up to six lines, a blank
	)
	room := m.height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

func (p passivesScreen) view(m model) (string, string) {
	footer := m.text(i18n.PassivesFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.PassivesHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.PassivesSubtitle)) + "\n\n")
	if len(p.passives) == 0 {
		return out.String() + "  " + m.text(i18n.PassivesEmpty), footer
	}

	column, glossColumn := 0, 0
	for _, held := range p.passives {
		if width := lipgloss.Width(held.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(m.lang.PassiveName(held)); width > glossColumn {
			glossColumn = width
		}
	}
	if glossColumn > 0 {
		// The header has to fit the column it names, the same rule the skill
		// listing's gloss column follows — and in English nothing is glossed at
		// all, which is what drops the column rather than drawing it empty.
		if width := lipgloss.Width(m.text(i18n.ColumnGloss)); width > glossColumn {
			glossColumn = width
		}
	}
	out.WriteString("  " + m.style.dim.Render(passiveRow(column+1, glossColumn,
		m.text(i18n.SkillFieldID), m.text(i18n.ColumnGloss),
		m.text(i18n.ColumnCarriedBy))) + "\n")

	from, to := window(len(p.passives), p.cursor, passivesRoom(m))
	for index := from; index < to; index++ {
		held := p.passives[index]
		row := passiveRow(column+1, glossColumn,
			held.ID, m.lang.PassiveName(held), p.carriers[held.ID])
		row = clip(row, minWidth-3)
		marker := "  "
		if index == p.cursor {
			marker = "> "
			row = m.style.selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}

	selected := p.passives[clamp(p.cursor, 0, len(p.passives)-1)]
	out.WriteString("\n  " + m.style.label.Render(m.lang.GlossedPassive(selected)) + "\n")
	for _, sentence := range strings.Split(m.lang.DescribePassive(selected), "\n") {
		// Wrapped to the floor rather than to the window, for the reason the
		// trait screen wraps: these are the program's own prose, and a sentence
		// run across a two-hundred-column terminal is a line a reader loses their
		// place in. See traitLines.
		for _, line := range wrapWords(sentence, minWidth-1-traitIndent) {
			out.WriteString(strings.Repeat(" ", traitIndent) + line + "\n")
		}
	}
	// A trait nobody learns cannot reach a battle, and the row above says so with
	// an empty cell — which reads as a column that failed to fill rather than as
	// a fact. This says it in words, and only for the trait being read.
	if p.carriers[selected.ID] == "" {
		out.WriteString("\n  " + m.style.dim.Render(m.text(i18n.PassivesNobodyCarries)))
	}
	return strings.TrimRight(out.String(), "\n"), footer
}

// passiveRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart — the same arrangement skillRow has, and
// for the same reason: a header out of line with its own rows is the one failure
// a shared layout prevents.
//
// A glossColumn of zero drops the name column entirely rather than drawing it
// empty, which is what English gets: a trait's name is authored once and in
// Vietnamese, so an English reader would see a column of blanks.
func passiveRow(idColumn, glossColumn int, id, name, carriers string) string {
	row := pad(id, idColumn)
	if glossColumn > 0 {
		row += " " + pad(name, glossColumn)
	}
	return row + " " + carriers
}
