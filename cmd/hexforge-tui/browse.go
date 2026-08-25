package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
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
	// filters is "every work" followed by the catalogued origin ids.
	filters []string
	filter  int
	cursor  int
	level   int
}

// everyOrigin is the filter that hides nothing.
const everyOrigin = "all origins"

func newBrowseScreen(lib *forge.Library) browseScreen {
	return browseScreen{level: progression.LevelCap}.refresh(lib)
}

// refresh re-reads the library, keeping the level and clamping the cursor —
// entering the screen after a write should show the new character rather than
// a stale list.
func (b browseScreen) refresh(lib *forge.Library) browseScreen {
	b.characters = lib.Characters().All()
	b.filters = append([]string{everyOrigin}, lib.OriginIDs()...)
	if b.level == 0 {
		b.level = progression.LevelCap
	}
	if b.filter >= len(b.filters) {
		b.filter = 0
	}
	b.cursor = clamp(b.cursor, 0, len(b.rows())-1)
	return b
}

// rows is the characters the current filter leaves visible.
func (b browseScreen) rows() []cast.Character {
	if b.filter <= 0 || b.filter >= len(b.filters) {
		return b.characters
	}
	wanted := b.filters[b.filter]
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
		b.filter = (b.filter + 1) % len(b.filters)
		b.cursor = clamp(b.cursor, 0, len(b.rows())-1)
	}
	m.browse = b
	return m, nil
}

func (b browseScreen) view(m model) (string, string) {
	footer := "↑/↓ character · ←/→ level · home/end 1 or the cap · f filter · esc back · q quit"
	rows := b.rows()
	var out strings.Builder
	fmt.Fprintf(&out, "%s  %s\n\n",
		m.style.heading.Render("cast"),
		m.style.dim.Render(fmt.Sprintf("showing %s (%d of %d characters)",
			b.filters[b.filter], len(rows), len(b.characters))))
	if len(rows) == 0 {
		out.WriteString("  nothing to show here.\n\n")
		if len(b.characters) == 0 {
			out.WriteString("  No characters have been authored yet. Pick \"new character\" from the menu.\n")
		} else {
			out.WriteString("  No character is borrowed from this work. Press f for the next filter.\n")
		}
		return out.String(), footer
	}
	for i, character := range rows {
		marker := "  "
		name := fmt.Sprintf("%-24s %-14s %s", character.ID, character.Origin, character.Element)
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
	out.WriteString(m.label("from", "%s", from))
	out.WriteString(m.label("tuned from", "%s", character.Archetype))
	out.WriteString(m.label("element", "%s", character.Element))
	out.WriteString(m.label("kit", "%s", strings.Join(character.Skills, " ")))
	art := character.Image
	if m.lib.ImageExists(character.Image) {
		art = m.style.good.Render(art + "  present")
	} else {
		art = m.style.bad.Render(art + "  MISSING")
	}
	out.WriteString(m.label("art", "%s", art))
	out.WriteString(m.label("stages", "%s", forge.StageSummary(character)))
	if character.Bio != "" {
		out.WriteString(m.label("bio", "%s", character.Bio))
	}

	values, stage, err := character.Resolve(b.level)
	if err != nil {
		out.WriteString(m.label(fmt.Sprintf("level %d", b.level), "%s", m.style.bad.Render(err.Error())))
		return out.String()
	}
	out.WriteString(m.label(fmt.Sprintf("level %d", b.level), "%s   %s",
		values, m.style.dim.Render("stage "+stage.Name)))
	budget := m.lib.Budget(values)
	out.WriteString(m.label("absorbs", "%s", budgetLine(m, budget)))
	return out.String()
}

// budgetLine is the joint health-and-defence bound drawn as a meter and as
// numbers. The numbers are what makes it readable without colour, and the
// wording of "over the budget" is the state, not the styling.
func budgetLine(m model, budget forge.Budget) string {
	text := fmt.Sprintf("%s %d of %d, %d to spare",
		bar(budgetBarWidth, budget.Effective, budget.Max), budget.Effective, budget.Max, budget.Headroom)
	if budget.Over() {
		return m.style.bad.Render(fmt.Sprintf("%s %d of %d, OVER THE BUDGET by %d",
			bar(budgetBarWidth, budget.Effective, budget.Max), budget.Effective, budget.Max, -budget.Headroom))
	}
	return m.style.good.Render(text)
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
