package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The traits every case here reads, built by hand for the same reason gatePair
// is.
//
// ⚠️ **Nothing shipped carries a gated grant** — step 2 ships the shape and step
// 3 ships the first subject — so every walk in this package that reads the
// shipped passives is blind to the field, and a test built out of shipped data
// here would pass with the whole branch deleted. It stays that way until a
// shipped trait carries one, at which point `describe.golden` moves and *that*
// diff is the design record. Nothing in this package can record the wording
// before then.
//
// tiered is the two-tier shape the field exists for: an ungated grant carrying
// the first tier and a gated one carrying the difference, on a trait with no
// gate of its own. lowTiered is the same thing written at the other end of the
// bar, and wholeGated is the whole-trait case that must not have moved.
func tieredTraits() (tiered, lowTiered, wholeGated passive.Passive) {
	tiered = passive.Passive{
		ID: "bulwark",
		Grants: []passive.Grant{
			{Status: "toughened", Stacks: 1},
			{Status: "fortified", Stacks: 1, While: &passive.Condition{AboveHealth: 700}},
		},
	}
	lowTiered = passive.Passive{
		ID: "lastditch",
		Grants: []passive.Grant{
			{Status: "toughened", Stacks: 1},
			{Status: "fortified", Stacks: 1, While: &passive.Condition{BelowHealth: 250}},
		},
	}
	wholeGated = passive.Passive{
		ID:    "dug_in",
		While: &passive.Condition{BelowHealth: 250},
		Grants: []passive.Grant{
			{Status: "toughened", Stacks: 1},
			{Status: "fortified", Stacks: 1},
		},
	}
	return tiered, lowTiered, wholeGated
}

// clauseOpening is the clause a wording begins with, ahead of its first blank.
//
// Matched by that rather than by a literal, so a translation reworded tomorrow
// keeps these honest where a copied English string would quietly stop matching
// and pass — the same reading TestAGatedTraitIsNotDescribedAsAlways and
// TestBothEndsOfAGateAreWordedInBothLanguages take of the same wordings.
func clauseOpening(text string) string {
	return strings.TrimSpace(strings.SplitN(text, "%", 2)[0])
}

// comparison is the words a wording puts *between* its two blanks.
//
// ⚠️ **The opening clause cannot tell the two ends of a grant's gate apart, and
// the first version of the test below read it and measured nothing.** Both
// wordings open on the same word — the status comes first, so everything ahead
// of it is "Carries" / "Mang" — and the comparison that is the whole of what a
// reader is deciding on sits after it. Deleting the AtTop branch in describe.go
// left every clause-opening assertion green.
//
// The trait-level pair does not have this problem (their comparisons are ahead
// of their only blank), which is why gateends_test.go can read openings and this
// cannot: the same reading is discriminating there and blind here.
func comparison(text string) string {
	parts := strings.SplitN(text, "%s", 3)
	if len(parts) < 3 {
		return ""
	}
	return parts[1]
}

// TestAGatedGrantSaysWhenInItsOwnSentence is the wording claim for step 2.
//
// A per-grant gate has nowhere else to be said. The trailing BlurbTraitWhile
// line words a *trait's* gate and qualifies every line above it, and a trait
// carrying a gated grant has no trait-level gate at all — so a description that
// reused the "carries" wording for a gated grant would state the grant and never
// state its condition, which is a rule with no sentence.
//
// Four things are read, in both languages: the gated grant's line carries its
// own clause and its own figure, the ungated grant beside it still reads
// "always", neither line borrows the other end's sentence, and the trait grows
// no trailing gate line it has no gate for.
func TestAGatedGrantSaysWhenInItsOwnSentence(t *testing.T) {
	tiered, lowTiered, _ := tieredTraits()
	kinds := shippedStatuses(t)
	checked := 0
	for _, lang := range i18n.Langs() {
		upper := lang.Text(i18n.BlurbTraitGrantsWhileAbove)
		lower := lang.Text(i18n.BlurbTraitGrantsWhile)
		if strings.TrimSpace(upper) == "" || strings.TrimSpace(lower) == "" {
			t.Errorf("%s: a gated grant has no wording at all", lang)
			continue
		}
		if upper == lower {
			t.Errorf("%s: both ends of a grant's gate are worded %q, so a description "+
				"cannot say which end it means", lang, upper)
		}
		// Two blanks in each — the status and the figure — and both wordings take
		// them in one order, which is the rule
		// TestTheSameBlanksInEveryLanguage holds for the pair.
		for name, text := range map[string]string{"upper": upper, "lower": lower} {
			if got := strings.Count(text, "%s"); got != 2 {
				t.Errorf("%s: the %s end's wording takes %d blanks, want the status and the figure: %q",
					lang, name, got, text)
			}
		}

		high := lang.DescribePassive(tiered, kinds)
		low := lang.DescribePassive(lowTiered, kinds)
		// One line per grant and nothing else: a trait with no gate of its own
		// must not grow the trailing gate line, and the gated grant's clause
		// must not be appended as a second line either.
		for id, description := range map[string]string{tiered.ID: high, lowTiered.ID: low} {
			if lines := len(strings.Split(strings.TrimSpace(description), "\n")); lines != 2 {
				t.Errorf("%s: %q declares two grants and is described in %d lines:\n%s",
					lang, id, lines, description)
			}
		}
		// The comparison rather than the opening clause, because the opening
		// clause is the same word at both ends — see comparison, and the note on
		// it saying that reading the opening here measured nothing.
		above, below := comparison(upper), comparison(lower)
		if above == "" || below == "" {
			t.Errorf("%s: a grant's gate wording has no text between its two blanks, so "+
				"nothing here can tell the two ends apart: %q / %q", lang, upper, lower)
			continue
		}
		if above == below {
			t.Errorf("%s: both ends of a grant's gate compare with %q, so a description "+
				"cannot say which end it means", lang, above)
		}
		if !strings.Contains(high, above) {
			t.Errorf("%s: a grant gated above 70%% health is described %q, which never says "+
				"it is in force at the top of the bar", lang, high)
		}
		if !strings.Contains(low, below) {
			t.Errorf("%s: a grant gated below 25%% health is described %q, which never says "+
				"it is in force at the bottom of the bar", lang, low)
		}
		// And neither borrows the other's comparison, which is the assertion that
		// catches a branch pointing at the wrong key: the figure would still be
		// right and the line would still read as a sentence.
		if strings.Contains(high, below) {
			t.Errorf("%s: a grant gated above 70%% health is described with the lower "+
				"end's comparison: %q", lang, high)
		}
		if strings.Contains(low, above) {
			t.Errorf("%s: a grant gated below 25%% health is described with the upper "+
				"end's comparison: %q", lang, low)
		}
		// The figure is the grant's own gate. A branch reading BelowHealth for a
		// gate written at the top of the bar prints "0%" here.
		if !strings.Contains(high, "70%") {
			t.Errorf("%s: a grant gated above 700 per thousand is described %q, which never "+
				"says 70%%", lang, high)
		}
		if !strings.Contains(low, "25%") {
			t.Errorf("%s: a grant gated below 250 per thousand is described %q, which never "+
				"says 25%%", lang, low)
		}
		// And the ungated tier beside it still reads "always", which is the half
		// of the two-tier sentence that makes the pair legible: "always carries
		// X" above "carries Z at or above Y". A trait-level gate is what takes
		// the word away, and there is none here.
		if !strings.Contains(high, clauseOpening(lang.Text(i18n.BlurbTraitGrants))) {
			t.Errorf("%s: the ungated tier of %q does not read as always in force:\n%s",
				lang, tiered.ID, high)
		}
		// The trailing trait-level gate line is absent, because there is no
		// trait-level gate. A description carrying it would be stating a
		// condition over the ungated tier as well.
		if strings.Contains(high, clauseOpening(lang.Text(i18n.BlurbTraitWhileAbove))) {
			t.Errorf("%s: %q has no gate of its own and still closes with the trait-level "+
				"gate line:\n%s", lang, tiered.ID, high)
		}
		checked++
	}
	if checked != len(i18n.Langs()) {
		t.Fatalf("worded %d languages, want %d", checked, len(i18n.Langs()))
	}
}

// TestAWholeTraitGateIsStillWordedAsOneClosingLine is the case that must not have
// moved.
//
// A trait gated as a whole keeps the arrangement it had: its grants read
// "carries" with no clause, and one closing line says when — because that line
// qualifies every line above it and "always carries" beside it would be two
// sentences of one paragraph contradicting each other. The per-grant wording is
// an addition, and a branch that had reached for it here would say the condition
// once per grant and then again at the end.
func TestAWholeTraitGateIsStillWordedAsOneClosingLine(t *testing.T) {
	_, _, wholeGated := tieredTraits()
	kinds := shippedStatuses(t)
	for _, lang := range i18n.Langs() {
		description := lang.DescribePassive(wholeGated, kinds)
		// Two grants and one gate: three lines and no more.
		if lines := len(strings.Split(strings.TrimSpace(description), "\n")); lines != 3 {
			t.Errorf("%s: a whole-trait gate over two grants is described in %d lines:\n%s",
				lang, lines, description)
		}
		if !strings.Contains(description, clauseOpening(lang.Text(i18n.BlurbTraitWhile))) {
			t.Errorf("%s: a gated trait's description never says when:\n%s", lang, description)
		}
		if strings.Contains(description, clauseOpening(lang.Text(i18n.BlurbTraitGrants))) {
			t.Errorf("%s: a gated trait still says it always applies:\n%s", lang, description)
		}
		// And the figure appears exactly once. A branch that reached for the
		// per-grant wording here would state the condition once per grant and
		// then again on the closing line — three times for one gate, which reads
		// as three rules.
		if got := strings.Count(description, "25%"); got != 1 {
			t.Errorf("%s: a whole-trait gate's figure appears %d times, want once:\n%s",
				lang, got, description)
		}
	}
}

// TestEveryGrantIsNamedWhateverItsGateIs keeps StatusesNamed in step with the
// description, which is the contract that lets a screen mark the names it
// printed without reading the prose back.
//
// A gate decides how a grant is worded and never whether it is worded, so a
// gated grant is a named status exactly as an ungated one is. The failure this
// catches is a filter added to one of the two functions.
func TestEveryGrantIsNamedWhateverItsGateIs(t *testing.T) {
	tiered, _, wholeGated := tieredTraits()
	for _, held := range []passive.Passive{tiered, wholeGated} {
		named := i18n.StatusesNamed(held)
		for _, grant := range held.Grants {
			found := false
			for _, id := range named {
				if id == grant.Status {
					found = true
				}
			}
			if !found {
				t.Errorf("%q grants %q and its description does not name it: %v",
					held.ID, grant.Status, named)
			}
		}
	}
}
