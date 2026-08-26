package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// blurbScreen shows what the skill under the listing's cursor does, in the
// sentences a player reads rather than in the figures an author types.
//
// It closes the loop the authoring tool was missing: every other row on the form
// says what a field *is*, and none of them said what the skill would sound like
// once somebody had to decide whether to use it. An author tuning a bonus from
// 1000 to 700 could see the damage move and not that "doubles" had become
// "amplifies a bit".
//
// Like previewScreen it keeps nothing of its own — no cursor, no scroll — and
// reads the listing it was raised from. A second cursor here could disagree with
// the one on the screen behind it, and a description of a skill the author is not
// looking at is worse than none.
//
// # Why a screen rather than a block under the form
//
// The form has nineteen fields and an eighty-by-twenty-four window shows
// thirteen of them; the comments in skills.go record fighting for a single row
// twice. A description is three lines and wraps to more, so putting it under the
// form would cost a quarter of the fields for something read occasionally. A
// screen costs nothing until it is asked for.
type blurbScreen struct{}

func (b blurbScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "?":
		m.screen = screenSkills
		return m, nil
	// The cursor moves from here too, so an author reading one description can
	// read the next without going back and forth. It walks the listing's own
	// cursor, which is the one thing this screen borrows.
	case "up", "k":
		m.skills.cursor = clamp(m.skills.cursor-1, 0, len(m.skills.skills)-1)
	case "down", "j":
		m.skills.cursor = clamp(m.skills.cursor+1, 0, len(m.skills.skills)-1)
	}
	return m, nil
}

func (b blurbScreen) view(m model) (string, string) {
	footer := m.text(i18n.BlurbFooter)
	skills := m.skills.skills
	if len(skills) == 0 {
		return "  " + m.text(i18n.NoneCatalogued) + "\n", footer
	}
	declared := skills[clamp(m.skills.cursor, 0, len(skills)-1)]

	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.lang.GlossedSkill(declared)) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition,
			clamp(m.skills.cursor, 0, len(skills)-1)+1, len(skills))) + "\n\n")
	for _, line := range strings.Split(m.lang.Describe(declared, m.lib.Patterns()), "\n") {
		out.WriteString("  " + line + "\n")
	}
	return out.String(), footer
}
