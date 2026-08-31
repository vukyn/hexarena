package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// PassivesScreen is the declared traits, with what each one does under the
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
type PassivesScreen struct {
	Passives []passive.Passive
	// carriers is keyed by trait id and read by key only; nothing ranges over it
	// into an ordered output, so it cannot reach a rendered line out of order.
	Carriers map[string]string
	Cursor   int
}

// NewPassivesScreen is the listing filled from a library, ready to draw.
func NewPassivesScreen(lib *forge.Library) PassivesScreen {
	return PassivesScreen{}.Refresh(lib)
}

// Refresh re-reads the declared traits and who carries each, keeping the cursor
// inside the book.
func (p PassivesScreen) Refresh(lib *forge.Library) PassivesScreen {
	p.Passives = lib.Passives().All()
	p.Carriers = make(map[string]string, len(p.Passives))
	for _, held := range p.Passives {
		p.Carriers[held.ID] = forge.TraitCarrierSummary(lib.TraitCarriers(held.ID))
	}
	p.Cursor = Clamp(p.Cursor, 0, len(p.Passives)-1)
	return p
}

// Update reads one keystroke: the cursor moves, the statuses reference is
// raised on the status this trait names, or the reader leaves.
func (p PassivesScreen) Update(_ Context, message tea.KeyPressMsg) (PassivesScreen, Action) {
	switch message.String() {
	case "q":
		return p, Action{Kind: Quit}
	case "esc":
		return p, Action{Kind: Back}
	// The trait's own description is already on screen, so ? here reads as
	// "explain the thing it just named" rather than as "explain this".
	//
	// The first named status rather than a choice between them. Every shipped
	// trait names one, and a picker for a case the data does not hold yet is a
	// second cursor to keep in step with this one — the trade screenPreview and
	// screenBlurb both refused. A trait naming two is the change to make when
	// one exists.
	//
	// ⚠️ The status is named and not reached for. This used to call focus on the
	// statuses screen's own state and write the cursor into it — the only
	// cross-screen read of the six, and a read no screen in a shared package
	// could keep, since it knows nothing about which other screens the client
	// has. So the id rides on the raise and the **client** lands the cursor,
	// including the staying-put when the book no longer holds that status.
	case "?":
		named := i18n.StatusesNamed(p.Passives[Clamp(p.Cursor, 0, len(p.Passives)-1)])
		if len(named) == 0 {
			return p, Action{}
		}
		return p, Action{Kind: Raise, Target: Statuses,
			Subject: Subject{Kind: StatusSubject, ID: named[0]}}
	case "up", "k":
		p.Cursor = Clamp(p.Cursor-1, 0, len(p.Passives)-1)
	case "down", "j":
		p.Cursor = Clamp(p.Cursor+1, 0, len(p.Passives)-1)
	}
	return p, Action{}
}

// passivesRoom is how many rows the listing may draw: the window, less the two
// the heading takes and the six the description below it may.
//
// Six is the most a trait's description runs to — a flavour clause plus the five
// jobs the busiest shipped trait holds — and it is measured at the most rather
// than at the height of the one under the cursor. A room that shrank and grew
// with the cursor would slide the listing up and down under it, and a reader
// would lose their place walking between two traits rather than reading either.
func passivesRoom(c Context) int {
	const (
		above = 2 // the heading and the blank line under it
		below = 8 // a blank, the trait's name, up to six lines, a blank
	)
	room := c.Height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

// View draws the listing, the sentences describing the trait under the cursor
// with the statuses they name marked, and the footer.
func (p PassivesScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.PassivesFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.PassivesHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.PassivesSubtitle)) + "\n\n")
	if len(p.Passives) == 0 {
		return out.String() + "  " + c.Text(i18n.PassivesEmpty), footer
	}

	column, glossColumn := 0, 0
	for _, held := range p.Passives {
		if width := lipgloss.Width(held.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(c.Lang.PassiveName(held)); width > glossColumn {
			glossColumn = width
		}
	}
	if glossColumn > 0 {
		// The header has to fit the column it names, the same rule the skill
		// listing's gloss column follows — and in English nothing is glossed at
		// all, which is what drops the column rather than drawing it empty.
		if width := lipgloss.Width(c.Text(i18n.ColumnGloss)); width > glossColumn {
			glossColumn = width
		}
	}
	out.WriteString("  " + c.Style.Dim.Render(passiveRow(column+1, glossColumn,
		c.Text(i18n.SkillFieldID), c.Text(i18n.ColumnGloss),
		c.Text(i18n.ColumnCarriedBy))) + "\n")

	from, to := Window(len(p.Passives), p.Cursor, passivesRoom(c))
	for index := from; index < to; index++ {
		held := p.Passives[index]
		row := passiveRow(column+1, glossColumn,
			held.ID, c.Lang.PassiveName(held), p.Carriers[held.ID])
		// The window, for the reason the skill listing's last column uses it:
		// the carriers cell is data, and cutting it on a wide terminal hides
		// which characters hold the trait.
		row = Clip(row, c.UsableWidth()-3)
		marker := "  "
		if index == p.Cursor {
			marker = "> "
			row = c.Style.Selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}

	selected := p.Passives[Clamp(p.Cursor, 0, len(p.Passives)-1)]
	// The names this trait's sentences will use, marked where they are printed
	// so that ? has something visible to be about. A miss in the glossary drops
	// out here rather than marking a bare id: an id in the middle of Vietnamese
	// prose is already the odd word on the line.
	names := make([]string, 0, 4)
	for _, id := range i18n.StatusesNamed(selected) {
		if name := c.Lang.Gloss(id); name != "" {
			names = append(names, name)
		}
	}
	out.WriteString("\n  " + c.Style.Label.Render(c.Lang.GlossedPassive(selected)) + "\n")
	for _, sentence := range strings.Split(c.Lang.DescribePassive(selected), "\n") {
		sentence = Marked(sentence, names, func(word string) string {
			return c.Style.Emphasis.Render(word)
		})
		// Wrapped to the floor rather than to the window, for the reason the
		// description screen wraps: these are the program's own prose, and a
		// sentence run across a two-hundred-column terminal is a line a reader
		// loses their place in. TraitIndent is shared with that screen for the
		// same reason.
		for _, line := range WrapWords(sentence, MinWidth-1-TraitIndent) {
			out.WriteString(strings.Repeat(" ", TraitIndent) + line + "\n")
		}
	}
	// A trait nobody learns cannot reach a battle, and the row above says so with
	// an empty cell — which reads as a column that failed to fill rather than as
	// a fact. This says it in words, and only for the trait being read.
	if p.Carriers[selected.ID] == "" {
		out.WriteString("\n  " + c.Style.Dim.Render(c.Text(i18n.PassivesNobodyCarries)))
	}
	return strings.TrimRight(out.String(), "\n"), footer
}

// passiveRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart — the same arrangement the client's own
// skillRow has, and
// for the same reason: a header out of line with its own rows is the one failure
// a shared layout prevents.
//
// A glossColumn of zero drops the name column entirely rather than drawing it
// empty, which is what English gets: a trait's name is authored once and in
// Vietnamese, so an English reader would see a column of blanks.
func passiveRow(idColumn, glossColumn int, id, name, carriers string) string {
	row := Pad(id, idColumn)
	if glossColumn > 0 {
		row += " " + Pad(name, glossColumn)
	}
	return row + " " + carriers
}
