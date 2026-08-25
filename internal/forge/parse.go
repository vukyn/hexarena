package forge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// ParseAffinity accepts what Affinity.String writes: one element name, or two
// separated by a slash.
func ParseAffinity(raw string) (element.Affinity, error) {
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

// ParseCurve reads a "base:max" stat override.
func ParseCurve(raw string) (progression.Curve, error) {
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

// FormatCurve is ParseCurve's inverse, which is what a prompt's default and a
// text field's starting content both have to be, so that accepting either
// reproduces the preset exactly.
func FormatCurve(curve progression.Curve) string {
	return fmt.Sprintf("%d:%d", curve.Base, curve.Max)
}

// SplitList reads a comma separated answer, dropping empty entries so a
// trailing comma is a typo rather than an unnamed skill.
func SplitList(raw string) []string {
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
