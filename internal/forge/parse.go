package forge

import (
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
		return element.Affinity{}, &MissingElementError{}
	}
	parts := strings.Split(trimmed, "/")
	members := make([]element.Element, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		member, err := element.Parse(name)
		if err != nil {
			return element.Affinity{}, &UnknownElementError{Name: name, Err: err}
		}
		members = append(members, member)
	}
	switch len(members) {
	case 1:
		return element.Single(members[0])
	case 2:
		return element.Dual(members[0], members[1])
	default:
		return element.Affinity{}, &AffinityCountError{Raw: trimmed, Count: len(members)}
	}
}

// ParseCurve reads a "base:max" stat override.
func ParseCurve(raw string) (progression.Curve, error) {
	base, max, found := strings.Cut(strings.TrimSpace(raw), ":")
	if !found {
		return progression.Curve{}, &CurveShapeError{Raw: raw}
	}
	first, err := strconv.ParseInt(strings.TrimSpace(base), 10, 64)
	if err != nil {
		return progression.Curve{}, &CurveNumberError{Raw: raw, Half: CurveBase, Err: err}
	}
	last, err := strconv.ParseInt(strings.TrimSpace(max), 10, 64)
	if err != nil {
		return progression.Curve{}, &CurveNumberError{Raw: raw, Half: CurveMax, Err: err}
	}
	return progression.Curve{Base: first, Max: last}, nil
}

// FormatCurve is ParseCurve's inverse, which is what a prompt's default and a
// text field's starting content both have to be, so that accepting either
// reproduces the preset exactly.
func FormatCurve(curve progression.Curve) string {
	return strconv.FormatInt(curve.Base, 10) + ":" + strconv.FormatInt(curve.Max, 10)
}

// ParseYear reads the year a work came out. An empty answer is a year nobody
// knows, which is a legal state for a work rather than a mistake.
func ParseYear(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, &YearError{Raw: trimmed, Err: err}
	}
	return year, nil
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
