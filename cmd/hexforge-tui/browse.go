package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// browseScreen is the authored cast, filtered by origin, with one character
// resolved beside it.
//
// The level is the reason this screen is worth having over `hexforge cast`: a
// character is a curve rather than a stat line, and walking the level with the
// arrow keys is how the shape of that curve — and where it starts costing the
// budget — becomes visible.
type browseScreen struct {
	characters []cast.Character
	// origins is the catalogued work ids, and filter indexes them from one.
	// Zero is the filter that hides nothing, and it is a number rather than a
	// name in the list because its name is a translated word while every other
	// entry is an id that is the same in every language.
	origins []string
	filter  int
	cursor  int
	level   int
}

func newBrowseScreen(lib *forge.Library) browseScreen {
	return browseScreen{level: progression.LevelCap}.refresh(lib)
}

// refresh re-reads the library, keeping the level and clamping the cursor —
// entering the screen after a write should show the new character rather than
// a stale list.
func (b browseScreen) refresh(lib *forge.Library) browseScreen {
	b.characters = lib.Characters().All()
	b.origins = lib.OriginIDs()
	if b.level == 0 {
		b.level = progression.LevelCap
	}
	if b.filter > len(b.origins) {
		b.filter = 0
	}
	b.cursor = clamp(b.cursor, 0, len(b.rows())-1)
	return b
}

// filterID is the work in force, or empty when every work is shown.
func (b browseScreen) filterID() string {
	if b.filter <= 0 || b.filter > len(b.origins) {
		return ""
	}
	return b.origins[b.filter-1]
}

// filterName is what the filter is called on screen: an id, or the translated
// word for "everything".
func (b browseScreen) filterName(m model) string {
	if id := b.filterID(); id != "" {
		return id
	}
	return m.text(i18n.BrowseAllOrigins)
}

// rows is the characters the current filter leaves visible.
func (b browseScreen) rows() []cast.Character {
	wanted := b.filterID()
	if wanted == "" {
		return b.characters
	}
	out := make([]cast.Character, 0, len(b.characters))
	for _, character := range b.characters {
		if character.Origin == wanted {
			out = append(out, character)
		}
	}
	return out
}

func (b browseScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		b.cursor = clamp(b.cursor-1, 0, len(b.rows())-1)
	case "down", "j":
		b.cursor = clamp(b.cursor+1, 0, len(b.rows())-1)
	case "left", "h":
		b.level = clamp(b.level-1, 1, progression.LevelCap)
	case "right", "l":
		b.level = clamp(b.level+1, 1, progression.LevelCap)
	case "home":
		b.level = 1
	case "end":
		b.level = progression.LevelCap
	case "f":
		b.filter = (b.filter + 1) % (len(b.origins) + 1)
		b.cursor = clamp(b.cursor, 0, len(b.rows())-1)
	case "p":
		// The preview keeps no cursor and no level of its own, so raising it
		// needs nothing handed over: it reads both back off this screen.
		m.browse = b
		m.screen = screenPreview
		return m, nil
	case "?":
		// The same arrangement, for the trait sentences. The row above prints a
		// trait's id and its name and stops there, which is the whole of what
		// this tool ever said about a trait — so an author moving virulence from
		// 300 to 200 watched a figure change and never read the line a player
		// gets. The screen borrows this cursor and this level rather than
		// keeping either, so the traits it describes are the ones in force at
		// the level being walked.
		m.browse = b
		m.blurb.from = screenBrowse
		m.screen = screenBlurb
		return m, nil
	}
	m.browse = b
	return m, nil
}

// The two fixed columns of the listing. Both hold ids, which are the same in
// every language and as long as the data makes them, so these are constants
// where a column of labels would be measured. The element column after them
// takes no width of its own: it is last, so its gloss spends the row's slack
// rather than another column's.
const (
	browseIDWidth     = 24
	browseOriginWidth = 14
)

func (b browseScreen) view(m model) (string, string) {
	footer := m.text(i18n.BrowseFooter)
	rows := b.rows()
	var out strings.Builder
	fmt.Fprintf(&out, "%s  %s\n\n",
		m.style.Heading.Render(m.text(i18n.BrowseHeading)),
		m.style.Dim.Render(m.text(i18n.BrowseShowing,
			b.filterName(m), len(rows), len(b.characters))))
	if len(rows) == 0 {
		out.WriteString("  " + m.text(i18n.BrowseNothingHere) + "\n\n")
		if len(b.characters) == 0 {
			out.WriteString("  " + m.text(i18n.BrowseNothingAuthored) + "\n")
		} else {
			out.WriteString("  " + m.text(i18n.BrowseNoneFromThisWork) + "\n")
		}
		return out.String(), footer
	}
	for i, character := range rows {
		marker := "  "
		// The element column is glossed here as well as in the detail pane
		// below, and it is the only list column that is: it is the last one on
		// the row, so the bracket eats the row's slack rather than another
		// column's. The worst case any pair of the eleven elements can make was
		// measured against both fixed columns at full width — 72 cells of the
		// 79 there are, on metal/electric. The two fixed columns take no gloss
		// for the same reason: an id past its column would push this one over.
		name := pad(character.ID, browseIDWidth) + " " +
			pad(character.Origin, browseOriginWidth) + " " +
			m.lang.GlossedAffinity(character.Element)
		if i == b.cursor {
			marker = "> "
			name = m.style.Selected.Render(name)
		}
		out.WriteString(marker + name + "\n")
	}
	out.WriteString("\n")
	out.WriteString(b.detail(m, rows[clamp(b.cursor, 0, len(rows)-1)]))
	return out.String(), footer
}

// detail is one character resolved at the chosen level.
func (b browseScreen) detail(m model, character cast.Character) string {
	var out strings.Builder
	origin, known := m.lib.Origins().Get(character.Origin)
	from := character.Origin
	if known {
		from = fmt.Sprintf("%s (%s, %s)", origin.Title, origin.Medium, character.Origin)
	}
	out.WriteString(m.style.Heading.Render(character.ID+" — "+character.Name) + "\n")
	out.WriteString(m.label(m.text(i18n.LabelFrom), "%s", from))
	// The preset and the element read as ids with their names on the row under
	// them, which is the one convention this pane has rather than two.
	//
	// They used to bracket the name beside the id — "blighter (kẻ gieo độc)" —
	// while the kit, the species and the traits put theirs underneath, and the
	// split was invisible to a reader: it followed where the name came from, a
	// compiled table against a data file, which is a fact about this repository
	// and not about the character on screen. The list above still brackets,
	// because a table column has no second row to give.
	//
	// An unglossed id draws no second row rather than an empty one, so an English
	// screen is exactly what it was.
	out.WriteString(m.label(m.text(i18n.LabelPlaystyle), "%s", character.Archetype))
	if name := m.lang.Gloss(character.Archetype); name != "" {
		out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, name))
	}
	out.WriteString(m.label(m.text(i18n.LabelElement), "%s", character.Element))
	if names := m.lang.AffinityNames(character.Element); names != "" {
		out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, names))
	}
	// Wrapped, not clipped: nine ids are longer than any terminal, and half an
	// id is worse than a second row.
	out.WriteString(m.wrapped(m.text(i18n.LabelKit), detailLabelWidth(m),
		forge.UnlockSummaryAt(character.Skills, b.level)))
	// The kit's names go under it rather than beside it: five skills glossed
	// inline is five brackets on one row, which does not fit. Nothing is drawn
	// at all when there is nothing to say — in English, or for a kit of skills
	// the table has no names for — rather than an empty row under a full one.
	if glossed := m.lang.GlossedKit(m.lib.KitSkills(cast.LearnedIDs(character.Skills))); glossed != "" {
		out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, glossed))
	}
	// Traits sit under the kit because they read as the other half of it: what
	// the character uses, then what it simply has. Drawn only when there are
	// any — a row saying "none" on every character in a cast that holds none is
	// a row that says nothing.
	out.WriteString(m.label(m.text(i18n.LabelStages), "%s", m.lang.StageSummary(character)))
	if character.Bio != "" {
		out.WriteString(m.wrapped(m.text(i18n.LabelBiography), detailLabelWidth(m), character.Bio))
	}

	atLevel := m.text(i18n.LabelAtLevel, b.level)
	values, stage, err := character.Resolve(b.level, progression.Furthest)
	if err != nil {
		out.WriteString(m.label(atLevel, "%s", m.style.Bad.Render(m.lang.Error(err))))
		return out.String()
	}
	// The art row keeps its place in the block that says what the character is,
	// but the picture it names is the one the level resolved to. A character
	// whose forms have their own pictures has no single picture, so walking the
	// level with the arrow keys is what shows which one a form uses — the same
	// reason the level is worth walking for the stats. It is drawn after Resolve
	// rather than before it because there is no picture to name until the stage
	// is known, and a character that will not resolve has already returned.
	out.WriteString(m.label(m.text(i18n.LabelArt), "%s", b.artLine(m, character, stage)))
	// What it is, when it is anything: the axis a skill kept for a lineage reads,
	// and the one fact on this screen that the numbers cannot imply. Only when
	// there is one, on the same terms as the traits row under it.
	if len(character.Species) > 0 {
		out.WriteString(m.wrapped(m.text(i18n.FieldSpecies), detailLabelWidth(m),
			strings.Join(character.Species, " ")))
		if glossed := m.lang.GlossedSpecies(m.lib.KitSpecies(character.Species)); glossed != "" {
			out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, glossed))
		}
	}
	// Traits sit with the art rather than with the kit, because both answer "at
	// this level": a gate still ahead is printed and one already passed is not, so
	// walking the level is what shows a trait coming in. The names go under it the
	// way the kit's do, and only the ones actually in force are named — glossing a
	// trait the unit does not have yet would read as a trait it has.
	if len(character.Passives) > 0 {
		out.WriteString(m.wrapped(m.text(i18n.LabelTraits), detailLabelWidth(m),
			forge.UnlockSummaryAt(character.Passives, b.level)))
		if glossed := m.lang.GlossedPassives(
			m.lib.KitPassives(character.PassivesAt(b.level, progression.Furthest))); glossed != "" {
			out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, glossed))
		}
	}
	// Six stats and a stage name come to 88 cells at the floor, so this wraps
	// too. It was being clipped by the frame, and what fell off the end was the
	// stage — the one part of the row that says which form the numbers belong to.
	out.WriteString(m.wrapped(atLevel, detailLabelWidth(m),
		fmt.Sprintf("%s   %s", values, m.text(i18n.StageInWords, stage.Name))))
	budget := m.lib.Budget(values)
	out.WriteString(m.label(m.text(i18n.LabelEffectiveHP), "%s", budgetLine(m, budget)))
	// The pierced floor goes under the row rather than beside it, dim and with
	// no label of its own, which is the shape the kit's names above already use
	// for a second reading of the row before it. Lang.BudgetPierced records why
	// it cannot be a clause on the line itself.
	out.WriteString(m.wrappedIn("", detailLabelWidth(m), m.style.Dim, m.lang.BudgetPierced(budget)))
	return out.String()
}

// artLine is the picture the character shows at the resolved stage, and whether
// it is on disk.
func (b browseScreen) artLine(m model, character cast.Character, stage progression.Stage) string {
	art := character.StageArt(stage)
	if m.lib.ImageExists(art) {
		return m.style.Good.Render(art + "  " + m.text(i18n.ArtPresent))
	}
	return m.style.Bad.Render(art + "  " + m.text(i18n.ArtMissing))
}

// budgetLine is the joint health-and-defence bound drawn as a meter and as
// numbers. The numbers are what makes it readable without colour, and being
// over the bound is said in words in both languages, because it is the state
// rather than the styling.
func budgetLine(m model, budget forge.Budget) string {
	meter := draw.Bar(draw.BudgetBarWidth, budget.Effective, budget.Max)
	if budget.Over() {
		return m.style.Bad.Render(m.lang.Budget(meter, budget))
	}
	return m.style.Good.Render(m.lang.Budget(meter, budget))
}

// clamp moved to model.go, beside pad and clip: the six reference screens took
// it with them into internal/screen, and what is left here is the forwarder the
// rest of this package still calls. Its comment there carries the argument.
