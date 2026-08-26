package tui

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// Detail is a description as a block a prompt can print: the skill's name, then
// its sentences, indented under a rule.
//
// The sentences themselves are i18n.Lang.Describe. This package holds only the
// framing, because the framing is what belongs to a screen: the menu above it is
// a table of figures, and a paragraph dropped into one without a break reads as
// another row that has lost its columns.
//
// The language is a parameter rather than a constant even though the caller
// passes Vietnamese today. The rest of this package is English and stays that way
// until the whole battle screen is translated in one piece; this block is the one
// part read to *decide*, so it is worth being legible before the rest catches up,
// and the mixed screen is a stated cost. Making it a parameter is what keeps that
// a one-word decision rather than a rewrite.
func Detail(lang i18n.Lang, declared skill.Skill, shapes *pattern.Book) string {
	title := declared.ID
	if name := lang.SkillName(declared); name != "" && name != declared.ID {
		title = fmt.Sprintf("%s · %s", declared.ID, name)
	}
	return block(title, lang.Describe(declared, shapes))
}

// DetailPassives is the traits a unit is carrying, as one block. A unit with
// none says so rather than printing an empty frame, because a blank answer to a
// question the player asked reads as the tool failing to answer it.
func DetailPassives(lang i18n.Lang, name string, held []passive.Passive) string {
	if len(held) == 0 {
		return block(name, lang.DescribePassive(passive.Passive{}))
	}
	parts := make([]string, 0, len(held))
	for _, one := range held {
		heading := one.ID
		if one.Name != "" {
			heading = fmt.Sprintf("%s · %s", one.ID, one.Name)
		}
		parts = append(parts, heading+"\n"+indent(lang.DescribePassive(one), "  "))
	}
	return block(name, strings.Join(parts, "\n"))
}

func block(title, body string) string {
	var b strings.Builder
	b.WriteString("  " + title + "\n")
	b.WriteString("  " + strings.Repeat("-", 46) + "\n")
	b.WriteString(indent(body, "  "))
	return b.String()
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
