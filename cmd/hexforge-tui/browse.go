package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
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

func (b browseScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.style.heading.Render(m.text(i18n.BrowseHeading)),
		m.style.dim.Render(m.text(i18n.BrowseShowing,
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
			name = m.style.selected.Render(name)
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
	out.WriteString(m.style.heading.Render(character.ID+" — "+character.Name) + "\n")
	out.WriteString(m.label(m.text(i18n.LabelFrom), "%s", from))
	out.WriteString(m.label(m.text(i18n.LabelPlaystyle), "%s", m.lang.Glossed(character.Archetype)))
	out.WriteString(m.label(m.text(i18n.LabelElement), "%s", m.lang.GlossedAffinity(character.Element)))
	// Wrapped, not clipped: nine ids are longer than any terminal, and half an
	// id is worse than a second row.
	out.WriteString(m.wrapped(m.text(i18n.LabelKit), detailLabelWidth(m),
		strings.Join(character.Skills, " ")))
	// The kit's names go under it rather than beside it: five skills glossed
	// inline is five brackets on one row, which does not fit. Nothing is drawn
	// at all when there is nothing to say — in English, or for a kit of skills
	// the table has no names for — rather than an empty row under a full one.
	if glossed := m.lang.GlossedKit(m.lib.KitSkills(character.Skills)); glossed != "" {
		out.WriteString(m.style.dim.Render(
			m.wrapped("", detailLabelWidth(m), glossed)))
	}
	art := character.Image
	if m.lib.ImageExists(character.Image) {
		art = m.style.good.Render(art + "  " + m.text(i18n.ArtPresent))
	} else {
		art = m.style.bad.Render(art + "  " + m.text(i18n.ArtMissing))
	}
	out.WriteString(m.label(m.text(i18n.LabelArt), "%s", art))
	out.WriteString(m.label(m.text(i18n.LabelStages), "%s", m.lang.StageSummary(character)))
	if character.Bio != "" {
		out.WriteString(m.wrapped(m.text(i18n.LabelBiography), detailLabelWidth(m), character.Bio))
	}

	atLevel := m.text(i18n.LabelAtLevel, b.level)
	values, stage, err := character.Resolve(b.level)
	if err != nil {
		out.WriteString(m.label(atLevel, "%s", m.style.bad.Render(m.lang.Error(err))))
		return out.String()
	}
	// Six stats and a stage name come to 88 cells at the floor, so this wraps
	// too. It was being clipped by the frame, and what fell off the end was the
	// stage — the one part of the row that says which form the numbers belong to.
	out.WriteString(m.wrapped(atLevel, detailLabelWidth(m),
		fmt.Sprintf("%s   %s", values, m.text(i18n.StageInWords, stage.Name))))
	budget := m.lib.Budget(values)
	out.WriteString(m.label(m.text(i18n.LabelEffectiveHP), "%s", budgetLine(m, budget)))
	return out.String()
}

// budgetLine is the joint health-and-defence bound drawn as a meter and as
// numbers. The numbers are what makes it readable without colour, and being
// over the bound is said in words in both languages, because it is the state
// rather than the styling.
func budgetLine(m model, budget forge.Budget) string {
	meter := bar(budgetBarWidth, budget.Effective, budget.Max)
	if budget.Over() {
		return m.style.bad.Render(m.lang.Budget(meter, budget))
	}
	return m.style.good.Render(m.lang.Budget(meter, budget))
}

// clamp keeps an index or a level inside its range, and returns the low bound
// when the range is empty.
func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
