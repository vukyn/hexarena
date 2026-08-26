package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
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

// The two fixed columns of the listing. The art column holds the longest of
// "present", "MISSING" and their Vietnamese counterparts, plus a space.
const (
	checkIDWidth  = 24
	checkArtWidth = 8
)

func (c checkScreen) view(m model) (string, string) {
	footer := m.text(i18n.CheckFooter)
	var out strings.Builder

	// The headline states the verdict in words. A check whose result is only a
	// colour is a check somebody will read wrong at the moment it matters — and
	// that holds in both languages, so neither verdict is a bare mark.
	verdict := m.style.good.Render(m.text(i18n.CheckPassed))
	if !c.report.OK() {
		verdict = m.style.bad.Render(m.text(i18n.CheckFailed, len(c.report.Problems)))
	}
	fmt.Fprintf(&out, "%s  %s\n", m.style.heading.Render(m.text(i18n.CheckHeading)), verdict)
	fmt.Fprintf(&out, "%s\n\n", m.style.dim.Render(m.text(i18n.CheckCounts,
		c.report.Dir, c.report.Origins, c.report.Archetypes, len(c.report.Rows))))

	if len(c.report.Rows) == 0 {
		out.WriteString("  " + m.text(i18n.CheckNothingToCheck) + "\n")
		return out.String(), footer
	}
	fmt.Fprintf(&out, "  %s %s %s\n",
		pad(m.text(i18n.ColumnCharacter), checkIDWidth),
		pad(m.text(i18n.ColumnArt), checkArtWidth),
		m.text(i18n.ColumnEffectiveHP))
	for i, row := range c.report.Rows {
		marker := "  "
		if i == c.cursor {
			marker = "> "
		}
		// The plain words while there is one picture, and the count once a
		// character has several: "thiếu" on a three-stage character leaves the
		// reader asking how many, and the answer is cheap.
		art := m.style.good.Render(pad(m.text(i18n.ArtPresent), checkArtWidth))
		if missing := row.ArtMissing(); missing > 0 {
			said := m.text(i18n.ArtMissing)
			if len(row.Art) > 1 {
				said = m.text(i18n.ArtSomeMissing, missing, len(row.Art))
			}
			art = m.style.bad.Render(pad(said, checkArtWidth))
		}
		detail := ""
		if row.Failure != nil {
			detail = m.style.bad.Render(m.text(i18n.CheckDoesNotResolve, m.lang.Error(row.Failure)))
		} else {
			detail = fmt.Sprintf("%s %d/%d",
				bar(statBarWidth, row.Budget.Effective, row.Budget.Max),
				row.Budget.Effective, row.Budget.Max)
			if row.Budget.Over() {
				detail = m.style.bad.Render(detail + "  " + m.text(i18n.CheckOverBudget))
			}
		}
		name := pad(row.ID, checkIDWidth)
		if i == c.cursor {
			name = m.style.selected.Render(name)
		}
		fmt.Fprintf(&out, "%s%s %s %s\n", marker, name, art, detail)
	}

	if !c.report.OK() {
		out.WriteString("\n")
		for _, problem := range c.report.Problems {
			out.WriteString("  " +
				m.style.bad.Render(m.text(i18n.CheckProblem, m.lang.Problem(problem))) + "\n")
		}
	}
	// Drawn dim rather than bad, and drawn whether or not the check passed: a
	// warning is not a failure, and a passing check is the one place nothing
	// else would say this.
	if len(c.report.Warnings) > 0 {
		out.WriteString("\n")
		for _, warning := range c.report.Warnings {
			out.WriteString("  " +
				m.style.dim.Render(m.text(i18n.CheckWarning, m.lang.Warning(warning))) + "\n")
		}
	}
	out.WriteString("\n" + m.style.dim.Render(m.text(i18n.CheckNote)))
	return out.String(), footer
}
