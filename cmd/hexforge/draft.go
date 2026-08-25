package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// draft is every answer that makes a character, held as the strings a flag or a
// prompt produced.
//
// Keeping the answers as text until the very end is what lets one function turn
// a flag-only invocation and a fully prompted one into the same character. That
// function is resolve, and it is pure: it reads the library, writes nothing, and
// touches neither the terminal nor the filesystem, which is what makes it
// testable without capturing stdout.
type draft struct {
	id        string
	name      string
	origin    string
	archetype string
	image     string
	element   string
	bio       string
	// skills is a comma separated list. Empty means "take the archetype's kit".
	skills string
	// stats holds a "base:max" override per stat. An empty entry means "take
	// the archetype's curve".
	stats [progression.KindCount]string
}

// resolve turns a draft into a character, or says which answer is wrong.
//
// Every check it makes it makes by calling into internal/core/cast,
// internal/core/element or internal/core/progression. This tool deliberately
// knows no rules of its own: a check that lived here would be a check the
// game's own load did not make.
func (d draft) resolve(lib *library) (cast.Character, error) {
	if err := cast.ValidateID(d.id); err != nil {
		return cast.Character{}, err
	}
	if _, clash := lib.characters.Get(d.id); clash {
		return cast.Character{}, fmt.Errorf("character %q is already in the cast", d.id)
	}
	if strings.TrimSpace(d.name) == "" {
		return cast.Character{}, fmt.Errorf("character %q needs a display name", d.id)
	}
	if _, known := lib.origins.Get(d.origin); !known {
		return cast.Character{}, fmt.Errorf("unknown origin %q, add it with %q",
			d.origin, "hexforge origins add "+d.origin)
	}
	archetype, known := lib.archetypes.Get(d.archetype)
	if !known {
		return cast.Character{}, fmt.Errorf("unknown archetype %q, want one of %s",
			d.archetype, strings.Join(lib.archetypes.IDs(), ", "))
	}
	if err := cast.ValidateImagePath(d.image); err != nil {
		return cast.Character{}, err
	}
	affinity, err := parseAffinity(d.element)
	if err != nil {
		return cast.Character{}, err
	}
	if err := lib.chart.ValidateAffinity(affinity); err != nil {
		return cast.Character{}, err
	}

	table := archetype.Stats
	for _, kind := range progression.Kinds() {
		override := strings.TrimSpace(d.stats[kind])
		if override == "" {
			continue
		}
		curve, err := parseCurve(override)
		if err != nil {
			return cast.Character{}, fmt.Errorf("%s: %w", kind, err)
		}
		if err := curve.Validate(kind); err != nil {
			return cast.Character{}, err
		}
		table[kind] = curve
	}

	skills := archetype.Skills
	if strings.TrimSpace(d.skills) != "" {
		skills = splitList(d.skills)
	}

	// A character created here has one stage, named after the character. A
	// second stage is an evolution line, and authoring one is an edit to
	// cast.json rather than a wizard question: the whole point of a stage is
	// the curve behind it, and answering twelve numbers twice at a prompt is
	// worse than editing the file.
	character := cast.Character{
		ID: d.id, Name: strings.TrimSpace(d.name),
		Origin: d.origin, Archetype: d.archetype,
		Image: d.image, Element: affinity, Bio: strings.TrimSpace(d.bio),
		Stages: progression.Line{{Name: strings.TrimSpace(d.name), MinLevel: 1, Stats: table}},
		Skills: skills,
	}
	// The last word belongs to the parser: appending validates the character
	// exactly as loading the file would, so nothing can be written that would
	// not load back.
	if _, err := lib.characters.Append(lib.castDeps(), character); err != nil {
		return cast.Character{}, err
	}
	return character, nil
}

// parseAffinity accepts what Affinity.String writes: one element name, or two
// separated by a slash.
func parseAffinity(raw string) (element.Affinity, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return element.Affinity{}, fmt.Errorf("no element given")
	}
	parts := strings.Split(trimmed, "/")
	members := make([]element.Element, 0, len(parts))
	for _, part := range parts {
		member, err := element.Parse(strings.TrimSpace(part))
		if err != nil {
			return element.Affinity{}, err
		}
		members = append(members, member)
	}
	switch len(members) {
	case 1:
		return element.Single(members[0])
	case 2:
		return element.Dual(members[0], members[1])
	default:
		return element.Affinity{}, fmt.Errorf("%q lists %d elements, want one or two separated by a slash",
			trimmed, len(members))
	}
}

// parseCurve reads a "base:max" stat override.
func parseCurve(raw string) (progression.Curve, error) {
	base, max, found := strings.Cut(strings.TrimSpace(raw), ":")
	if !found {
		return progression.Curve{}, fmt.Errorf("%q is not a curve, want base:max", raw)
	}
	first, err := strconv.ParseInt(strings.TrimSpace(base), 10, 64)
	if err != nil {
		return progression.Curve{}, fmt.Errorf("%q has an unreadable base: %w", raw, err)
	}
	last, err := strconv.ParseInt(strings.TrimSpace(max), 10, 64)
	if err != nil {
		return progression.Curve{}, fmt.Errorf("%q has an unreadable max: %w", raw, err)
	}
	return progression.Curve{Base: first, Max: last}, nil
}

// formatCurve is parseCurve's inverse, which is what a prompt's default has to
// be so that pressing Enter reproduces it exactly.
func formatCurve(curve progression.Curve) string {
	return fmt.Sprintf("%d:%d", curve.Base, curve.Max)
}

// splitList reads a comma separated answer, dropping empty entries so a
// trailing comma is a typo rather than an unnamed skill.
func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
