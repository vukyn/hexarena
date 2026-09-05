package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// BonusesScreen is the reference for what a squad is paid for what it shares:
// every declared threshold, who receives it and what reaching it grants.
//
// It exists because a composition bonus is the one thing in the game a player
// cannot find out by looking at the board. A skill is on a unit's list, a trait
// is on its character, a status is on the roster line — but a bonus is a
// property of the *squad that was brought*, decided before the first turn, and
// the only trace of it afterwards is a permanent buff that looks exactly like a
// trait's. Without this screen the way to learn that fielding two of an element
// is worth anything is to read bonuses.json.
//
// Read-only, like the statuses and elements listings and for the same reason: a
// rung is balance rather than content, so adding one is a change to every squad
// that could reach it and belongs in a diff rather than behind a form.
type BonusesScreen struct {
	Rows []composition.Bonus
	// Cursor indexes Rows. Every row is a bonus — there are no headings here,
	// unlike the statuses listing, because the thing a heading would group by is
	// the scope and the scope is already a column.
	Cursor int
}

// NewBonusesScreen is the reference filled from a library, ready to draw.
func NewBonusesScreen(lib *forge.Library) BonusesScreen {
	return BonusesScreen{}.Refresh(lib)
}

// Refresh re-reads the book, keeping the cursor inside it.
func (s BonusesScreen) Refresh(lib *forge.Library) BonusesScreen {
	s.Rows = lib.Bonuses().All()
	s.Cursor = Clamp(s.Cursor, 0, len(s.Rows)-1)
	return s
}

// Update reads one keystroke: the cursor moves, or the reader leaves.
func (s BonusesScreen) Update(_ Context, message tea.KeyPressMsg) (BonusesScreen, Action) {
	switch message.String() {
	case "q":
		return s, Action{Kind: Quit}
	case "esc":
		return s, Action{Kind: Back}
	case "up", "k":
		s.Cursor = Clamp(s.Cursor-1, 0, len(s.Rows)-1)
	case "down", "j":
		s.Cursor = Clamp(s.Cursor+1, 0, len(s.Rows)-1)
	}
	return s, Action{}
}

// bonusesRoom is how many rows the listing may draw: what the window has, less
// the heading pair above it and the description and caveat below.
//
// ⚠️ The description is reserved at the height of the **widest bonus in the
// book** rather than of the one under the cursor, which is the rule
// statusesRoom states for itself: a room that shrank and grew as the cursor
// moved would slide the listing up and down under a reader walking it. It is
// counted rather than fixed because a bonus's block is two lines plus one per
// grant, and the rungs are exactly what is expected to grow — two today, four
// when 5v5 opens — so a constant here would be a number that silently stops
// being right.
func bonusesRoom(c Context, rows []composition.Bonus) int {
	const (
		above = 2 // the heading and the blank line under it
		blank = 2 // the blank over the description and the one over the caveat
		fixed = 2 // what the bonus counts, and who it goes to
	)
	grants := 0
	for _, bonus := range rows {
		count := 0
		for _, rung := range bonus.Rungs {
			count += len(rung.Grants)
		}
		grants = max(grants, count)
	}
	room := c.Height - 4 - above - blank - fixed - grants - 1 // the caveat
	if room < 3 {
		return 3
	}
	return room
}

// View draws the listing, the description of the bonus under the cursor, the
// caveat that is true of all of them, and the footer.
func (s BonusesScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.BonusesFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.BonusesHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.BonusesSubtitle)) + "\n\n")
	if len(s.Rows) == 0 {
		out.WriteString("  " + c.Text(i18n.BonusesEmpty) + "\n")
		return out.String(), footer
	}

	// Two columns measured over the rows rather than over the book's widest
	// possible anything: the ids are what is drawn, and a column measured from
	// something not on the screen leaves a gap a reader reads as an empty field.
	ids, names := 0, 0
	for _, bonus := range s.Rows {
		ids = max(ids, lipgloss.Width(bonus.ID))
		names = max(names, lipgloss.Width(c.Lang.BonusName(bonus)))
	}
	from, to := Window(len(s.Rows), s.Cursor, bonusesRoom(c, s.Rows))
	for index := from; index < to; index++ {
		bonus := s.Rows[index]
		// The id, the authored name, and the scope as the **data** writes it —
		// `sharers` or `squad`, the same reading the statuses listing gives a
		// category, with the sentence explaining it left to the description
		// below rather than said twice on one screen.
		//
		// The scope is in a column at all because it is the one fact two bonuses
		// granting the same status do not share: a reader looking at a unit
		// carrying `kinship` cannot tell from the board whether its whole side
		// has it, and this column is where that is answered.
		line := Pad(bonus.ID, ids+1) + " " + Pad(c.Lang.BonusName(bonus), names+1) + " " +
			bonus.Scope.String()
		marker := "  "
		if index == s.Cursor {
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + strings.TrimRight(line, " ") + "\n")
	}

	out.WriteString("\n")
	selected := s.Rows[Clamp(s.Cursor, 0, len(s.Rows)-1)]
	glosses := c.Lang.LogGlosses(nil, c.Lib.Statuses().Kinds(), nil, nil)
	for _, line := range strings.Split(c.Lang.DescribeBonus(selected, glosses), "\n") {
		out.WriteString("  " + line + "\n")
	}
	// Once, at the foot, for the reason the statuses listing puts its caveat
	// there: what it says — counted once, never recounted, several at a time —
	// is true of every bonus, and a warning repeated under every row is a
	// warning nobody finishes reading. No newline after it: the frame pads the
	// body to the window and cuts from the bottom, so a trailing one costs the
	// caveat itself.
	out.WriteString("\n  " + c.Style.Dim.Render(c.Text(i18n.BlurbBonusCaveat)))
	return out.String(), footer
}
