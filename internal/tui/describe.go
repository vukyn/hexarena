package tui

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
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

// DetailStatus is one timed effect as a block a prompt can print, with the
// caveat under it.
//
// The caveat is here rather than in the sentence, because it is true of every
// status and a reader asking about one is asking once: repeating "a trait may
// change this" fifteen times down a reference trains a reader to stop reading it.
func DetailStatus(lang i18n.Lang, kind status.Kind) string {
	title := kind.ID
	if name := lang.Gloss(kind.ID); name != "" && name != kind.ID {
		title = fmt.Sprintf("%s · %s", kind.ID, name)
	}
	return block(title, lang.DescribeStatus(kind)) + "\n\n" +
		indent(lang.Text(i18n.BlurbStatusCaveat), "  ")
}

// DetailStatuses is the whole reference: every declared status under its
// category, and the caveat once at the foot.
//
// Grouped rather than listed flat, because the grouping is the half a flat list
// cannot carry: a skill that strips a stat_debuff and a dot is unreadable to
// somebody who cannot see which statuses those two words cover.
func DetailStatuses(lang i18n.Lang, groups []status.Group) string {
	var b strings.Builder
	for _, group := range groups {
		fmt.Fprintf(&b, "  %s · %s\n",
			group.Category, lang.StatusCategory(group.Category.String()))
		b.WriteString("  " + strings.Repeat("-", 46) + "\n")
		for _, kind := range group.Kinds {
			name := kind.ID
			if gloss := lang.Gloss(kind.ID); gloss != "" && gloss != kind.ID {
				name = fmt.Sprintf("%s · %s", kind.ID, gloss)
			}
			b.WriteString("  " + name + "\n")
			b.WriteString(indent(lang.DescribeStatus(kind), "    ") + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(indent(lang.Text(i18n.BlurbStatusCaveat), "  "))
	return b.String()
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
