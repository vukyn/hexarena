package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// buildIndent is how far a build sits inside its character's group.
//
// The grouping is drawn rather than coloured, because a heading and a build are
// two ids of the same shape here — pokemon.bulbasaur over bulbasaur.poison — and
// the dim style that tells the status reference's categories from its statuses is
// nothing at all on a terminal without colour. Two cells is what says which rows
// belong to the row above them when the only difference is the words.
const buildIndent = 2

// buildRow is one line of the listing: a character opening its own group, or one
// of that character's builds.
//
// The character is on every row, including the build rows, because a build names
// its character and a row that had to look one up would be a second index into
// the cast.
//
// Headings are rows rather than something drawn between them, for the reason the
// status reference files its categories the same way: the listing scrolls, and a
// heading computed while rendering falls off the top of the window and leaves the
// rows under it with nothing saying whose they are.
type buildRow struct {
	character cast.Character
	// built is the build the row draws, and its absence is what makes the row a
	// heading.
	built   cast.Build
	heading bool
	// empty marks a heading whose character has no build, which is what puts the
	// note about it on the heading's own row.
	//
	// On its own row it was scrollable away from the character it was about: at the
	// floor the first thing on screen was "no build written for this one yet" with
	// the name of the character it meant one row above the top of the window. A
	// note that can be separated from its subject is a note that says nothing, so
	// it rides on the row that names the subject and the pair cannot come apart.
	empty bool
}

// build reports whether the row is a build, which is the only sort of row the
// cursor may land on. A heading names a character and has no loadout for the pane
// to describe, whether or not the character has any builds.
func (r buildRow) build() bool { return !r.heading }

// buildsScreen is the authored late-game builds, grouped by character, with the
// loadout of the one under the cursor spelled out below.
//
// It is the answer to a question the rest of the tool cannot reach. A character
// screen shows a *learnset* — nine skills and five traits by the cap — and a
// placement spends four slots and one, so what the browser draws is everything a
// character could be rather than anything it is. Until the catalogue existed the
// only kit this repository could name was "the first four declared", which is the
// order the file happens to list and not a decision anybody made.
//
// Read-only, like the status and trait references. A build is authored in
// builds.json against a character's learnset, and the loadout it picks changes
// what a battle measures — so it belongs in a diff, and this screen exists to be
// read.
type buildsScreen struct {
	rows []buildRow
	// cursor indexes rows and is always on a build. Nothing is learnt by selecting
	// a heading or the note under one, and a cursor that could land on either
	// would make the pane below blink out for a keystroke.
	cursor int
}

func newBuildsScreen(lib *forge.Library) buildsScreen {
	return buildsScreen{}.refresh(lib)
}

// refresh re-reads the catalogue, keeping the cursor where the list allows.
//
// It walks the *cast* and asks each character for its builds, rather than walking
// the catalogue and grouping what it finds. The two differ in exactly one place
// and it is the place that matters: a character nobody has written a direction for
// is in the cast and in no build, so walking the catalogue would leave it off the
// screen entirely — the reader would be told the cast is shorter than it is, and
// the character most worth noticing is the one with nothing written for it.
func (b buildsScreen) refresh(lib *forge.Library) buildsScreen {
	b.rows = nil
	for _, character := range lib.Characters().All() {
		found := lib.BuildsOf(character.ID)
		b.rows = append(b.rows,
			buildRow{heading: true, character: character, empty: len(found) == 0})
		for _, built := range found {
			b.rows = append(b.rows, buildRow{character: character, built: built})
		}
	}
	b.cursor = b.settle(clamp(b.cursor, 0, len(b.rows)-1), 1)
	return b
}

// settle walks from a row to the nearest build in the given direction, and turns
// round at the end rather than sitting on a heading.
//
// The turn matters for the first row of all, which is always a heading, and for
// the last character in the cast: a cursor that only ever stepped forward would
// come to rest on the heading of a character whose builds have just been deleted.
// A cast with no build at all leaves it where it was, and the pane draws nothing —
// there is nothing for it to describe, which is the truthful screen.
func (b buildsScreen) settle(from, step int) int {
	for index := from; index >= 0 && index < len(b.rows); index += step {
		if b.rows[index].build() {
			return index
		}
	}
	for index := from; index >= 0 && index < len(b.rows); index -= step {
		if b.rows[index].build() {
			return index
		}
	}
	return from
}

// move steps to the next build in a direction, or stays put at the end of the
// list. It steps over headings rather than through them, so one keypress is one
// build however many characters lie between.
func (b buildsScreen) move(step int) buildsScreen {
	for index := b.cursor + step; index >= 0 && index < len(b.rows); index += step {
		if b.rows[index].build() {
			b.cursor = index
			return b
		}
	}
	return b
}

// selected is the build under the cursor, and whether there is one at all.
//
// A catalogue nobody has written leaves the cursor on a heading, which is the one
// state where this reports nothing: the pane has no loadout to draw and says so by
// not drawing.
func (b buildsScreen) selected() (buildRow, bool) {
	if len(b.rows) == 0 {
		return buildRow{}, false
	}
	row := b.rows[clamp(b.cursor, 0, len(b.rows)-1)]
	return row, row.build()
}

func (b buildsScreen) update(_ draw.Context, message tea.KeyPressMsg) (buildsScreen, draw.Action) {
	switch message.String() {
	case "q":
		return b, draw.Action{Kind: draw.Quit}
	case "esc":
		return b, draw.Action{Kind: draw.Back}
	case "up", "k":
		b = b.move(-1)
	case "down", "j":
		b = b.move(1)
	}
	return b, draw.Action{}
}

// buildsRoom is how many rows the listing may draw: what the window has, less the
// heading pair above it and the loadout pane below.
//
// The pane is measured at its tallest rather than at the height of the one under
// the cursor. A room that shrank and grew as the cursor moved would slide the
// whole listing up and down under it, and a reader would lose their place walking
// between two builds rather than reading either.
func buildsRoom(m model) int {
	const (
		above = 2 // the heading and the blank line under it
		// A blank, the build's own name, the intent over two rows, the kit and its
		// names over two each, and the trait and its names.
		below = 10
	)
	room := m.height - 4 - above - below
	if room < 3 {
		return 3
	}
	return room
}

func (b buildsScreen) view(m model) (string, string) {
	footer := m.text(i18n.BuildsFooter)
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.BuildsHeading)) + "  " +
		m.style.Dim.Render(m.text(i18n.BuildsSubtitle, progression.LevelCap)) + "\n\n")
	if len(b.rows) == 0 {
		out.WriteString("  " + m.text(i18n.BuildsEmpty) + "\n")
		return out.String(), footer
	}

	// The character ids are measured with the build ids and their indent, because
	// the words after both sit in one column: a build id is its character's name
	// and a suffix, so which of the two is wider depends on the cast and on nothing
	// a constant could know.
	column := 0
	for _, row := range b.rows {
		width := buildIndent + lipgloss.Width(row.built.ID)
		if row.heading {
			width = lipgloss.Width(row.character.ID)
		}
		if width > column {
			column = width
		}
	}
	from, to := window(len(b.rows), b.cursor, buildsRoom(m))
	for index := from; index < to; index++ {
		row := b.rows[index]
		if row.heading {
			// A character with no direction written says so on its own row rather
			// than being left with an empty group under it. It is a fact about the
			// catalogue and not a gap in it: a learnset with one obvious kit has
			// nothing to choose between, and a catalogue inventing a second direction
			// for it would be saying something untrue.
			line := row.character.ID
			if row.empty {
				line = pad(line, column+1) + " " + m.text(i18n.BuildsNoneForThisOne)
			}
			out.WriteString("  " + m.style.Dim.Render(line) + "\n")
			continue
		}
		// The id and the name it is called by, and nothing else: everything a build
		// holds is in the pane below, and a skills column beside the name would be
		// the same four ids twice on one screen.
		line := strings.Repeat(" ", buildIndent) +
			pad(row.built.ID, column+1-buildIndent) + " " + m.lang.BuildName(row.built)
		line = strings.TrimRight(line, " ")
		// The marker keeps the column every other listing puts it in, ahead of the
		// group indent rather than inside it: the cursor is the one thing on screen
		// a reader looks for without reading, and a cursor that sits somewhere else
		// on one screen is one they have to find twice.
		marker := "  "
		if index == b.cursor {
			marker = "> "
			line = m.style.Selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}

	selected, found := b.selected()
	if !found {
		return strings.TrimRight(out.String(), "\n"), footer
	}
	out.WriteString("\n" + b.pane(m, selected.built))
	return strings.TrimRight(out.String(), "\n"), footer
}

// pane is one build spelled out: what it is called, why it exists, and the two
// halves of the loadout it spends its slots on.
//
// Every id on it is glossed the way the cast browser glosses a kit — ids on the
// labelled row, names dimmed on the row under it — because these are the same
// ids, and a build read beside a character has to name them the same way. The
// intent is the one line that is neither: it is authored prose with no id behind
// it, so it is printed in both languages, exactly as an origin's note is.
func (b buildsScreen) pane(m model, built cast.Build) string {
	var out strings.Builder
	width := detailLabelWidth(m)
	out.WriteString("  " + m.style.Heading.Render(m.lang.GlossedBuild(built)) + "\n")
	if built.Intent != "" {
		out.WriteString(m.wrapped(m.text(i18n.LabelIntent), width, built.Intent))
	}
	out.WriteString(m.wrapped(m.text(i18n.LabelKit), width, strings.Join(built.Skills, " ")))
	if glossed := m.lang.GlossedKit(m.lib.KitSkills(built.Skills)); glossed != "" {
		out.WriteString(m.wrappedIn("", width, m.style.Dim, glossed))
	}
	// The trait row is drawn even when the build takes none, which is the one place
	// this pane differs from the cast browser's. There a character simply has the
	// traits it has, so a row reading "none" would say nothing; here the slot is
	// spent or deliberately left empty, and an absent row cannot tell a reader
	// which of the two it is looking at.
	if len(built.Passives) == 0 {
		out.WriteString(m.label(m.text(i18n.LabelTraits), "%s",
			m.style.Dim.Render(m.text(i18n.BuildsNoTrait))))
		return out.String()
	}
	out.WriteString(m.wrapped(m.text(i18n.LabelTraits), width, strings.Join(built.Passives, " ")))
	if glossed := m.lang.GlossedPassives(m.lib.KitPassives(built.Passives)); glossed != "" {
		out.WriteString(m.wrappedIn("", width, m.style.Dim, glossed))
	}
	return out.String()
}
