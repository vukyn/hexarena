package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// SpeciesScreen is what a character can *be*: the declared kinds, each under its
// name, with the note that says where the line is drawn under the cursor.
//
// It is the other half of the restriction column the skills listing draws. That
// column says "chủng loài dragon" and had nowhere to go: a species was reachable
// only through the picker on the new-skill form, so a reader who was not in the
// middle of authoring a restriction could not find out what the word covered,
// and the note the author wrote beside it reached nobody at all.
//
// ⚠️ It does not list the skills kept for a kind either, and for the same reason
// it does not list who is one: both are lists that grow with the books. Four
// dragon skills fit on a line and fourteen do not, and the reader who wants them
// is on the skills listing, whose restriction column names the kind on every row
// that has one. This screen answers "what does this word mean", once, and leaves
// the cross-reference to the listings that are already sorted for it.
//
// ⚠️ There is no **who is one** column, and that is a decision rather than an
// omission. It was drawn here first, as the twin of the trait listing's carriers
// column: a species is not restricted, it is *claimed*, so "who may be one" is
// everybody and answers nothing, while who actually is one is a fact. But the
// two columns scale differently. A trait is carried by the handful of characters
// that learn it; a species is what a character **is**, so every character in the
// book lands in exactly one or two kinds — and a cast of thirty puts thirty ids
// on five rows. A column whose width is the size of the cast is a column that
// stops fitting, and it stops fitting on the row that has the most to say.
//
// What survives is the half that does not grow: a kind **nobody** is still says
// so, in words, under the cursor. That is the fact the column was worth having
// for — a kind may be written before the character that fills it, exactly as a
// trait may, and it is not an error, but it is a gate that cannot open. Who is
// one is answered the other way round, by the browser, where a character says
// what it is.
//
// Read-only, unlike the origins listing. A species is a word plus a note today,
// but the gate it drives is a skill's allowlist, so adding one is only ever half
// a change: the other half is the skill kept for it, and that is authored on the
// skills form. A form here would offer to make a kind nothing can use.
type SpeciesScreen struct {
	Kinds []cast.Species
	// claimed and skills are keyed by species id and read by key only; nothing
	// ranges over either into an ordered output, so neither can reach a rendered
	// line out of order.
	//
	// claimed counts rather than naming, because the names are no longer drawn:
	// all the screen asks is whether anybody is one, and keeping the joined list
	// for a question that is answered by "is it nought" would be a row's worth of
	// string built per kind per refresh and thrown away.
	Claimed map[string]int
	Cursor  int
}

// NewSpeciesScreen is the reference filled from a library, ready to draw.
func NewSpeciesScreen(lib *forge.Library) SpeciesScreen {
	return SpeciesScreen{}.Refresh(lib)
}

// Refresh re-reads the kinds and who claims each, keeping the cursor inside the
// book.
func (s SpeciesScreen) Refresh(lib *forge.Library) SpeciesScreen {
	s.Kinds = lib.Species().All()
	s.Claimed = make(map[string]int, len(s.Kinds))
	for _, kind := range s.Kinds {
		s.Claimed[kind.ID] = len(lib.Characters().OfSpecies(kind.ID))
	}
	s.Cursor = Clamp(s.Cursor, 0, len(s.Kinds)-1)
	return s
}

// Update reads one keystroke: the cursor moves, or the reader leaves.
func (s SpeciesScreen) Update(_ Context, message tea.KeyPressMsg) (SpeciesScreen, Action) {
	switch message.String() {
	case "q":
		return s, Action{Kind: Quit}
	case "esc":
		return s, Action{Kind: Back}
	case "up", "k":
		s.Cursor = Clamp(s.Cursor-1, 0, len(s.Kinds)-1)
	case "down", "j":
		s.Cursor = Clamp(s.Cursor+1, 0, len(s.Kinds)-1)
	}
	return s, Action{}
}

// speciesRoom is how many rows the listing may draw: the window less the heading
// pair above and the note and the empty-kind line below.
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
// frame cuts from the bottom, so what an overrun costs is the line saying nobody
// is this kind, on exactly the kinds whose note is longest.
func speciesRoom(c Context, s SpeciesScreen) int {
	const (
		above = 2 // the heading and the blank line under it
		other = 3 // a blank, a blank, the empty-kind line
	)
	room := c.Height - 4 - above - other - s.longestNote()
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
func (s SpeciesScreen) longestNote() int {
	most := 1
	for _, kind := range s.Kinds {
		if lines := len(WrapWords(kind.Note, MinWidth-3)); lines > most {
			most = lines
		}
	}
	return most
}

// speciesRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart — the arrangement passiveRow beside it
// and skillRow in the client both have, and for the same reason.
//
// A nameColumn of zero drops the name column entirely, which is what English
// gets. The word beside the id is a **data** name — a field on the declaration,
// authored once and in Vietnamese — rather than a compiled gloss that is empty
// in the other language by construction, so drawing it in English would be a
// leak rather than a translation. Dropped rather than blanked, because a column
// of empty cells reads as data the book has lost.
func speciesRow(idColumn, nameColumn int, id, name string) string {
	if nameColumn == 0 {
		return id
	}
	return Pad(id, idColumn) + " " + name
}

// View draws the listing, the note under the kind the cursor is on, and the
// footer.
func (s SpeciesScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.SpeciesFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SpeciesHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.SpeciesSubtitle)) + "\n\n")
	if len(s.Kinds) == 0 {
		out.WriteString("  " + c.Text(i18n.SpeciesEmpty) + "\n")
		return out.String(), footer
	}

	column, nameColumn := 0, 0
	for _, kind := range s.Kinds {
		if width := lipgloss.Width(kind.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(c.Lang.SpeciesName(kind)); width > nameColumn {
			nameColumn = width
		}
	}
	// Each header has to fit the column it names, the same rule the other two
	// listings follow: a header wider than its rows pushes the one beside it off
	// its own column. The name header is measured only when there is a column
	// under it, or English would draw a header over nothing.
	if width := lipgloss.Width(c.Text(i18n.SkillFieldID)); width > column {
		column = width
	}
	if nameColumn > 0 {
		if width := lipgloss.Width(c.Text(i18n.ColumnGloss)); width > nameColumn {
			nameColumn = width
		}
	}
	out.WriteString("  " + c.Style.Dim.Render(speciesRow(column+1, nameColumn,
		c.Text(i18n.SkillFieldID), c.Text(i18n.ColumnGloss))) + "\n")

	from, to := Window(len(s.Kinds), s.Cursor, speciesRoom(c, s))
	for index := from; index < to; index++ {
		kind := s.Kinds[index]
		// No clip. Both cells are bounded by the book rather than by the cast:
		// an id and an authored word, each measured into its own column above.
		// The listings that clip do it because their last cell is a list that
		// grows with the cast, which is exactly the cell this screen no longer
		// draws.
		row := speciesRow(column+1, nameColumn, kind.ID, c.Lang.SpeciesName(kind))
		marker := "  "
		if index == s.Cursor {
			marker = "> "
			row = c.Style.Selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}

	out.WriteString("\n")
	selected := s.Kinds[Clamp(s.Cursor, 0, len(s.Kinds)-1)]
	// The note is authored prose, so it wraps at the floor rather than at the
	// window: a sentence run across a two-hundred-column terminal is a line a
	// reader loses their place in. The kinds that have none say so rather than
	// leaving the pane blank, which reads as the tool failing to answer.
	note := selected.Note
	if strings.TrimSpace(note) == "" {
		note = c.Text(i18n.SpeciesNoNote)
	}
	for _, line := range WrapWords(note, MinWidth-3) {
		out.WriteString("  " + line + "\n")
	}
	// A kind nobody is, said in words and only for the kind being read. It is the
	// one fact about a species that is not on its own row, and the only one worth
	// a line: a gate that cannot open, on a screen whose whole job is to say what
	// the gates are.
	if s.Claimed[selected.ID] == 0 {
		out.WriteString("\n  " + c.Style.Dim.Render(c.Text(i18n.SpeciesNobodyIs)))
	}
	return strings.TrimRight(out.String(), "\n"), footer
}
