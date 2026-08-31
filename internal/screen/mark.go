package screen

import (
	"sort"
	"strings"
)

// TraitIndent is how far the sentences sit under the trait they belong to. Four
// rather than two, so a wrapped line is still visibly part of the trait above it
// and not the start of the next one.
//
// It is here rather than beside one of its readers because there are three of
// them in two packages now — the traits listing and the chart in this package,
// and the description screen still in cmd/hexforge-tui — and an indent written
// down twice is two paragraphs that stop lining up.
const TraitIndent = 4

// Marked is one sentence of a trait's description with the status names in it
// picked out.
//
// Word by word rather than name by name, because the caller wraps what comes
// back: a style spanning two words survives strings.Fields only until the wrap
// puts the words on different lines, and then the first line carries an escape
// sequence the second one closes. Marking each word whole keeps every word
// self-contained however the line breaks.
//
// Longest first, so a name that begins another name cannot take its opening.
// The marker is a parameter rather than a style read off a Context, which is
// what makes this testable: the tests run with NO_COLOR set, so every style is
// the identity and a test asserting on the palette's own would be asserting that
// nothing happened.
func Marked(sentence string, names []string, mark func(string) string) string {
	sorted := append([]string(nil), names...)
	sort.SliceStable(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })

	// One left-to-right pass rather than a replacement per name. A pass per name
	// re-marks its own output — "bỏng" matches inside what "bỏng nặng" just
	// produced, however the names were ordered — and the ordering only decides
	// which of the two wrong answers comes out. Scanning once and taking the
	// longest name that starts here means every character of the sentence is
	// looked at exactly once, so nothing marked can be marked again.
	var out strings.Builder
	for at := 0; at < len(sentence); {
		hit := ""
		for _, name := range sorted {
			if name != "" && strings.HasPrefix(sentence[at:], name) {
				hit = name
				break
			}
		}
		if hit == "" {
			out.WriteByte(sentence[at])
			at++
			continue
		}
		words := strings.Fields(hit)
		for index, word := range words {
			words[index] = mark(word)
		}
		out.WriteString(strings.Join(words, " "))
		at += len(hit)
	}
	return out.String()
}
