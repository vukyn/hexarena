package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/forge"
)

// checkScreen is forge.Inspect drawn.
//
// It answers the one question internal/core is not allowed to ask — is the art
// a character names really on disk — plus what every character spends of the
// stat budget. The finding is entirely the package's; this screen chooses only
// the columns.
type checkScreen struct {
	report forge.Report
	cursor int
}

func newCheckScreen(lib *forge.Library) checkScreen {
	return checkScreen{}.refresh(lib)
}

func (c checkScreen) refresh(lib *forge.Library) checkScreen {
	c.report = lib.Inspect()
	c.cursor = clamp(c.cursor, 0, len(c.report.Rows)-1)
	return c
}

func (c checkScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "r":
		c = c.refresh(m.lib)
	case "up", "k":
		c.cursor = clamp(c.cursor-1, 0, len(c.report.Rows)-1)
	case "down", "j":
		c.cursor = clamp(c.cursor+1, 0, len(c.report.Rows)-1)
	}
	m.check = c
	return m, nil
}

func (c checkScreen) view(m model) (string, string) {
	footer := "↑/↓ move · r re-read the files · esc back · q quit"
	var out strings.Builder

	// The headline states the verdict in words. A check whose result is only a
	// colour is a check somebody will read wrong at the moment it matters.
	verdict := m.style.good.Render("PASSED — no problems found")
	if !c.report.OK() {
		verdict = m.style.bad.Render(fmt.Sprintf("FAILED — %d problem(s)", len(c.report.Problems)))
	}
	fmt.Fprintf(&out, "%s  %s\n", m.style.heading.Render("check"), verdict)
	fmt.Fprintf(&out, "%s\n\n", m.style.dim.Render(fmt.Sprintf(
		"%s: %d origins, %d archetypes, %d characters",
		c.report.Dir, c.report.Origins, c.report.Archetypes, len(c.report.Rows))))

	if len(c.report.Rows) == 0 {
		out.WriteString("  no characters to check.\n")
		return out.String(), footer
	}
	fmt.Fprintf(&out, "  %-24s %-8s %s\n", "character", "art", "absorbs of the budget at the cap")
	for i, row := range c.report.Rows {
		marker := "  "
		if i == c.cursor {
			marker = "> "
		}
		art := m.style.good.Render("present")
		if !row.ImageExists {
			art = m.style.bad.Render("MISSING")
		}
		detail := ""
		if row.Failure != nil {
			detail = m.style.bad.Render("does not resolve: " + row.Failure.Error())
		} else {
			detail = fmt.Sprintf("%s %d of %d",
				bar(statBarWidth, row.Budget.Effective, row.Budget.Max),
				row.Budget.Effective, row.Budget.Max)
			if row.Budget.Over() {
				detail = m.style.bad.Render(detail + "  OVER")
			}
		}
		name := fmt.Sprintf("%-24s", row.ID)
		if i == c.cursor {
			name = m.style.selected.Render(name)
		}
		fmt.Fprintf(&out, "%s%s %-8s %s\n", marker, name, art, detail)
	}

	if !c.report.OK() {
		out.WriteString("\n")
		for _, problem := range c.report.Problems {
			out.WriteString("  " + m.style.bad.Render("problem: "+problem) + "\n")
		}
	}
	out.WriteString("\n" + m.style.dim.Render(
		"this reads the files from disk; the game boots from the embedded copy, so an\n"+
			"edit needs a rebuild before it reaches a battle"))
	return out.String(), footer
}
