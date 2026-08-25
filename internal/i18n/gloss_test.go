package i18n

import (
	"sort"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/element"
)

// The gloss table's guarantees, which are deliberately not the catalog's. The
// catalog is held complete because a gap in it is a bug; this is held *lenient*
// because a gap in it is an id somebody has not named yet, and the tests below
// are mostly about the shape of a miss rather than the absence of one.

// TestEveryElementIsGlossed is the one completeness rule that applies here.
//
// Elements are not data in the sense skills are: element.Element is a Go enum
// and a twelfth element is a Go change, which is also when element.Chart
// refuses to validate an unclassified one. So an element with no name is the
// same kind of oversight as a catalog gap, and is treated as one.
func TestEveryElementIsGlossed(t *testing.T) {
	for _, member := range element.All() {
		if Vi.Gloss(member.String()) == "" {
			t.Errorf("the element %s has no Vietnamese name", member)
		}
	}
	if got, want := len(elementGloss), element.Count; got != want {
		t.Errorf("the element table holds %d names for %d elements", got, want)
	}
}

// TestAnIDWithNoGlossIsNormal is the whole difference between this table and
// the catalog, asserted rather than described.
//
// A skill or an archetype arrives by editing JSON, so an id this table has
// never heard of is the expected case and not a failure — a bare id on screen,
// never "bolt ()" and never a placeholder. The absent ids below are deliberately
// not in any data file: they stand in for the next skill somebody adds.
func TestAnIDWithNoGlossIsNormal(t *testing.T) {
	for _, absent := range []string{"tidal_hymn", "warden", "no_such_element", ""} {
		if got := Vi.Gloss(absent); got != "" {
			t.Errorf("the absent id %q was glossed %q", absent, got)
		}
		if got := Vi.Glossed(absent); got != absent {
			t.Errorf("the absent id %q rendered as %q, want it bare", absent, got)
		}
	}
	// A kit of nothing but unknown skills draws no gloss line rather than a
	// line of repeated ids.
	if got := Vi.GlossedKit([]string{"tidal_hymn", "warden"}); got != "" {
		t.Errorf("a kit of unglossed skills rendered as %q, want nothing", got)
	}
	// A kit that is partly known keeps the unknown one in place, so the line
	// stays in the same order as the ids above it.
	if got, want := Vi.GlossedKit([]string{"strike", "tidal_hymn"}),
		"đòn đánh · tidal_hymn"; got != want {
		t.Errorf("a partly known kit reads %q, want %q", got, want)
	}
}

// TestAnAffinityWithNoGlossRendersBare is the same rule for the one name that
// is not a plain id.
//
// The element table is held complete, so there is no real affinity this can
// happen to — which is exactly why it is worth pinning: "unreachable" is a
// property of today's enum, and the branch has to be right before the day it is
// reached. The gap is made by taking a name out of the table for the duration
// of the test.
func TestAnAffinityWithNoGlossRendersBare(t *testing.T) {
	forgetTheNameOf(t, element.Ice)

	single, err := element.Single(element.Ice)
	if err != nil {
		t.Fatalf("the ice affinity: %v", err)
	}
	if got, want := Vi.GlossedAffinity(single), "ice"; got != want {
		t.Errorf("an unglossed single affinity reads %q, want %q", got, want)
	}

	// A pair with nothing glossed is bare in the same way: no empty bracket.
	forgetTheNameOf(t, element.Metal)
	both, err := element.Dual(element.Metal, element.Ice)
	if err != nil {
		t.Fatalf("the metal/ice affinity: %v", err)
	}
	if got, want := Vi.GlossedAffinity(both), "metal/ice"; got != want {
		t.Errorf("an unglossed dual affinity reads %q, want %q", got, want)
	}

	// A pair with one half named keeps the other half's id inside the bracket,
	// so the two sides stay positional and it is visible which is missing.
	half, err := element.Dual(element.Water, element.Ice)
	if err != nil {
		t.Fatalf("the water/ice affinity: %v", err)
	}
	if got, want := Vi.GlossedAffinity(half), "water/ice (nước/ice)"; got != want {
		t.Errorf("a half-glossed dual affinity reads %q, want %q", got, want)
	}
}

// forgetTheNameOf takes one element out of the table for the length of a test,
// which is the only way to reach the missing-gloss branch for an affinity.
func forgetTheNameOf(t *testing.T, member element.Element) {
	t.Helper()
	id := member.String()
	name, known := elementGloss[id]
	if !known {
		t.Fatalf("the element %s already has no name, so nothing is being proved", id)
	}
	delete(elementGloss, id)
	t.Cleanup(func() { elementGloss[id] = name })
}

// TestADualAffinityGlossesAsOnePair is the format, exactly.
//
// One bracket for the pair rather than one each: "grass (cỏ)/electric (điện)"
// reads as two things a unit has, and a dual affinity is one thing.
func TestADualAffinityGlossesAsOnePair(t *testing.T) {
	pair, err := element.Dual(element.Grass, element.Electric)
	if err != nil {
		t.Fatalf("the grass/electric affinity: %v", err)
	}
	if got, want := Vi.GlossedAffinity(pair), "grass/electric (cỏ/điện)"; got != want {
		t.Errorf("a dual affinity reads %q, want %q", got, want)
	}
	single, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("the fire affinity: %v", err)
	}
	if got, want := Vi.GlossedAffinity(single), "fire (lửa)"; got != want {
		t.Errorf("a single affinity reads %q, want %q", got, want)
	}
	// The ids on the left are the affinity's own spelling, so the pair on the
	// right cannot be separated differently from the pair it explains.
	if !strings.HasPrefix(Vi.GlossedAffinity(pair), pair.String()+" ") {
		t.Errorf("the gloss does not sit after the affinity as written: %q",
			Vi.GlossedAffinity(pair))
	}
}

// TestEnglishAddsNothing holds the other half of the feature: in English a data
// name is shown as the data writes it, with no bracket and nothing appended.
func TestEnglishAddsNothing(t *testing.T) {
	for _, id := range glossedIDs() {
		if got := En.Gloss(id); got != "" {
			t.Errorf("English glossed %q as %q", id, got)
		}
		if got := En.Glossed(id); got != id {
			t.Errorf("English rendered %q as %q", id, got)
		}
	}
	pair, err := element.Dual(element.Grass, element.Electric)
	if err != nil {
		t.Fatalf("the grass/electric affinity: %v", err)
	}
	if got, want := En.GlossedAffinity(pair), "grass/electric"; got != want {
		t.Errorf("English rendered the affinity as %q, want %q", got, want)
	}
	if got := En.GlossedKit([]string{"strike", "riptide"}); got != "" {
		t.Errorf("English rendered a kit gloss %q, want nothing", got)
	}
}

// TestNoIDIsGlossedTwice keeps the three tables disjoint.
//
// The lookup walks them in order, so an id in two of them takes the first
// silently — a wrong name rather than a missing one, which is the worse of the
// two failures and the only one this file's leniency cannot excuse.
func TestNoIDIsGlossedTwice(t *testing.T) {
	seen := make(map[string]int, len(elementGloss)+len(archetypeGloss)+len(skillGloss))
	for table, glossary := range glossaries {
		for id := range glossary {
			if first, twice := seen[id]; twice {
				t.Errorf("the id %q is glossed by table %d and table %d", id, first, table)
				continue
			}
			seen[id] = table
		}
	}
}

// TestEveryGlossIsOneCellPerLetter is the layout rule, applied here because a
// gloss lands in the same padded columns a wording does.
//
// Same measurement as TestEveryWordingIsOneCellPerLetter and the same reason: a
// name written decomposed measures a cell short of what it draws, and every
// column on that screen drifts.
func TestEveryGlossIsOneCellPerLetter(t *testing.T) {
	for _, id := range glossedIDs() {
		name := Vi.Gloss(id)
		if got, want := lipgloss.Width(name), utf8.RuneCountInString(name); got != want {
			t.Errorf("the gloss of %s measures %d cells over %d runes: %q", id, got, want, name)
		}
		for _, letter := range name {
			if unicode.Is(unicode.Mn, letter) {
				t.Errorf("the gloss of %s holds the combining mark %U; write Vietnamese composed",
					id, letter)
			}
		}
		if strings.TrimSpace(name) != name {
			t.Errorf("the gloss of %s is padded: %q", id, name)
		}
	}
}

// glossedIDs is every id the tables name, in a fixed order so a failure reads
// the same way twice.
func glossedIDs() []string {
	var ids []string
	for _, glossary := range glossaries {
		for id := range glossary {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
