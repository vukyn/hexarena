package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The two gates every case here reads, either side of one threshold, built by
// hand rather than taken out of the shipped book.
//
// ⚠️ **They have to be built by hand, and that is the finding rather than a
// convenience.** No shipped trait carries a gate at the top of the health bar —
// the term arrived before the first subject for it, deliberately — so every walk
// in this package that reads the shipped passives is blind to the upper end and
// will stay blind until a trait uses it. A test built out of shipped data here
// would pass with the whole branch deleted.
func gatePair() (fresh, hurt passive.Passive) {
	fresh = passive.Passive{
		ID:     "unbowed",
		While:  &passive.Condition{AboveHealth: 900},
		Drains: 250,
	}
	hurt = passive.Passive{
		ID:     "cornered",
		While:  &passive.Condition{BelowHealth: 900},
		Drains: 250,
	}
	return fresh, hurt
}

// TestBothEndsOfAGateAreWordedInBothLanguages is the wording claim, and it is
// four assertions rather than one.
//
// Each language words each end, the two wordings are *different* in each
// language, and neither language falls back on the other's sentence. A missing
// key would print an empty line or the raw format, and a branch that forgot the
// upper end would print the lower end's sentence about a gate that means the
// opposite — which is the failure worth catching, because the figure in it would
// still be right and the line would still read as a sentence.
func TestBothEndsOfAGateAreWordedInBothLanguages(t *testing.T) {
	fresh, hurt := gatePair()
	checked := 0
	for _, lang := range i18n.Langs() {
		upper := lang.Text(i18n.BlurbTraitWhileAbove)
		lower := lang.Text(i18n.BlurbTraitWhile)
		if strings.TrimSpace(upper) == "" {
			t.Errorf("%s: the upper end of a gate has no wording at all", lang)
			continue
		}
		if upper == lower {
			t.Errorf("%s: both ends of a gate are worded %q, so a description "+
				"cannot say which end it means", lang, upper)
		}
		// One blank in each, because the figure is the whole of what varies.
		for name, text := range map[string]string{"upper": upper, "lower": lower} {
			if got := strings.Count(text, "%s"); got != 1 {
				t.Errorf("%s: the %s end's wording takes %d blanks, want one: %q",
					lang, name, got, text)
			}
		}

		freshLine := lang.DescribePassive(fresh, shippedStatuses(t))
		hurtLine := lang.DescribePassive(hurt, shippedStatuses(t))
		// Matched by the clause ahead of the blank rather than by a literal, so
		// a translation reworded tomorrow keeps this honest — the same reading
		// TestAGatedTraitIsNotDescribedAsAlways takes of the same wordings.
		opening := func(text string) string {
			return strings.TrimSpace(strings.SplitN(text, "%", 2)[0])
		}
		if !strings.Contains(freshLine, opening(upper)) {
			t.Errorf("%s: a gate above 90%% health is described %q, which never "+
				"says it is in force at the top of the bar", lang, freshLine)
		}
		if !strings.Contains(hurtLine, opening(lower)) {
			t.Errorf("%s: a gate below 90%% health is described %q, which never "+
				"says it is in force at the bottom of the bar", lang, hurtLine)
		}
		// And neither borrows the other's sentence. The two openings differ only
		// in their comparison, so this is the assertion that catches a branch
		// pointing at the wrong key.
		if opening(upper) != opening(lower) {
			if strings.Contains(freshLine, opening(lower)) {
				t.Errorf("%s: a gate above 90%% health is described with the lower "+
					"end's sentence: %q", lang, freshLine)
			}
			if strings.Contains(hurtLine, opening(upper)) {
				t.Errorf("%s: a gate below 90%% health is described with the upper "+
					"end's sentence: %q", lang, hurtLine)
			}
		}
		// The figure is the gate's own. A branch reading BelowHealth for a gate
		// written at the top of the bar prints "0%" here, so this is the
		// assertion that catches it — the sentence would still read as a
		// sentence, and only the figure would be wrong.
		if !strings.Contains(freshLine, "90%") {
			t.Errorf("%s: a gate above 900 per thousand is described %q, which "+
				"never says 90%%", lang, freshLine)
		}
		checked++
	}
	// The count, so a walk over an empty language list cannot pass.
	if checked != len(i18n.Langs()) {
		t.Fatalf("worded %d languages, want %d", checked, len(i18n.Langs()))
	}
}

// TestAGateAtTheTopOfTheBarIsDescribedInOneLine is the shape rule the shipped
// walk holds for every other clause, applied to the new end: a trait declares
// some number of things and is described in exactly that many lines.
//
// ⚠️ It is here rather than folded into TestEveryTraitDescriptionSaysEveryThingItDoes
// because that walk reads the shipped book, and no shipped trait carries this
// end. A branch that appended *both* sentences for one gate would pass there and
// fail here.
func TestAGateAtTheTopOfTheBarIsDescribedInOneLine(t *testing.T) {
	fresh, hurt := gatePair()
	for _, lang := range i18n.Langs() {
		for _, held := range []passive.Passive{fresh, hurt} {
			description := lang.DescribePassive(held, shippedStatuses(t))
			// A drain and a gate, so two lines and no more.
			if lines := len(strings.Split(description, "\n")); lines != 2 {
				t.Errorf("%s: %q declares a drain and a gate and is described in "+
					"%d lines:\n%s", lang, held.ID, lines, description)
			}
		}
	}
}
