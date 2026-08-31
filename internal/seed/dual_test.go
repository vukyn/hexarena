package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/seed"
)

// theFirstDual is the character the claims below are about. Named once rather
// than repeated, because when a second dual is authored these are the tests that
// have to say which one they meant.
const theFirstDual = "pokemon.magnemite"

// TestADualAffinityCancelsAsMuchAsItStacks is the design record for the first
// two-element character shipped, and the reason it is not simply a bonus.
//
// element.Affinity has carried a second element since the chart was written, and
// nothing used it: every character declared one name, so Dual, ValidateAffinity,
// MultiplierAgainst and the whole "stacks the two multipliers" paragraph were a
// mechanism with no user. Magnemite is the first, and what it demonstrates is
// that a pair whose halves are unrelated is mostly *cancellation*: every element
// that counters one half is countered by the other, so the pair is neutral to
// nearly the whole book rather than doubly weak or doubly strong.
//
// The chart's own arithmetic is tested where it lives. What is held here is the
// reading of it that the character was authored against — that the defensive
// half of a dual is worth close to nothing — because that is the claim its stat
// line was priced under, and a chart edit that quietly turned it into a stacked
// resistance would leave the thinnest frame in the cast attached to the sturdiest
// element pairing with every other test still green.
func TestADualAffinityCancelsAsMuchAsItStacks(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	who, known := characters.Get(theFirstDual)
	if !known {
		t.Fatalf("%s is not in the shipped cast", theFirstDual)
	}
	if !who.Element.IsDual() {
		t.Fatalf("%s declares %s, a single element, so nothing here is measuring a dual affinity",
			theFirstDual, who.Element)
	}

	neutral := chart.Multipliers().Neutral
	var beats, resisted []element.Element
	for _, member := range element.All() {
		switch against := chart.MultiplierAgainst(member, who.Element); {
		case against > neutral:
			beats = append(beats, member)
		case against < neutral:
			resisted = append(resisted, member)
		}
	}
	// One of each. Two elements each have two counters and two victims, so a
	// stacking pair would read four and four; that this reads one and one is the
	// cancellation, counted rather than described.
	if len(beats) != 1 || len(resisted) != 1 {
		t.Errorf("%s is beaten by %v and resists %v: a pair whose halves cancel leaves one of each, and a longer list means the two elements are pulling the same way",
			who.Element, beats, resisted)
	}
	// And the cancelling is named rather than inferred from the count above. Each
	// half is counted by whoever counters *it*, on its own, and then asked again
	// of the pair: an element that beat one half and comes out level against both
	// was cancelled by the other, which is the whole claim written as the
	// difference between two lists rather than as an arithmetic identity.
	var alone, cancelled []element.Element
	for _, half := range who.Element.Elements() {
		single, err := element.Single(half)
		if err != nil {
			t.Fatalf("single %s: %v", half, err)
		}
		for _, member := range element.All() {
			if chart.MultiplierAgainst(member, single) <= neutral {
				continue
			}
			alone = append(alone, member)
			if chart.MultiplierAgainst(member, who.Element) <= neutral {
				cancelled = append(cancelled, member)
			}
		}
	}
	t.Logf("%s is countered by %d elements as two singles and by %d as a pair; %v cancel",
		who.Element, len(alone), len(beats), cancelled)
	if len(cancelled) == 0 {
		t.Errorf("every element that beats a half of %s still beats the pair, so the second element bought no cover at all and the pairing is a stacked risk",
			who.Element)
	}
	if len(alone) <= len(beats) {
		t.Errorf("%s is countered %d times as singles and %d times as a pair: a pair that is not easier to attack than its halves is not the trade this character was priced under",
			who.Element, len(alone), len(beats))
	}

	// ⚠️ **Anti-vacuity, and it is not decoration.** Everything above would read
	// the same on a chart where cancelling is what pairing always does, and then
	// it would be holding a property of duals rather than a choice somebody made.
	// The chart validates itself hard enough that no legal edit to elements.json
	// reaches these assertions — every mutation tried was refused for an unrelated
	// reason first — so the counter-example has to be built here. metal/light is
	// as legal a pairing as the shipped one, unrelated halves and all, and it
	// cancels nothing: whichever half an attacker picks, the other stands aside.
	// Choosing a pair whose counters cover each other was a decision, and this is
	// the line that says so.
	stacking, err := element.Dual(element.Metal, element.Light)
	if err != nil {
		t.Fatalf("build the counter-example: %v", err)
	}
	if err := chart.ValidateAffinity(stacking); err != nil {
		t.Fatalf("%s is not a legal pairing, so it is no counter-example: %v", stacking, err)
	}
	stacked := 0
	for _, member := range element.All() {
		if chart.MultiplierAgainst(member, stacking) > neutral {
			stacked++
		}
	}
	if stacked <= len(beats) {
		t.Errorf("%s is beaten by %d elements and %s by %d, so pairing cancels by itself and %s chose nothing",
			stacking, stacked, who.Element, len(beats), who.Element)
	}
}

// TestADualAffinityIsAChoiceRatherThanABonus is the offensive half, and the
// reason the character is worth authoring at all.
//
// A unit may only learn skills of an element its affinity holds, so a dual
// carries two books instead of one — which sounds like a straight gain until the
// board is read. Against half the shipped cast the two elements *disagree*: one
// is an advantage and the other is a disadvantage against the very same target,
// so a kit that brings both and picks wrong throws a third of its damage away.
// That disagreement is the cost the second element is paid for with, and it is
// what this holds.
func TestADualAffinityIsAChoiceRatherThanABonus(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	who, known := characters.Get(theFirstDual)
	if !known {
		t.Fatalf("%s is not in the shipped cast", theFirstDual)
	}
	members := who.Element.Elements()
	if len(members) < 2 {
		t.Fatalf("%s declares %s, so there is no second element to choose between",
			theFirstDual, who.Element)
	}

	neutral := chart.Multipliers().Neutral
	disagreed := 0
	for _, other := range characters.All() {
		if other.ID == who.ID {
			continue
		}
		best, worst := 0, 0
		for _, member := range members {
			against := chart.MultiplierAgainst(member, other.Element)
			if best == 0 || against > best {
				best = against
			}
			if worst == 0 || against < worst {
				worst = against
			}
		}
		// Never worse off for having two: whichever way the pair leans, one of
		// the two elements is at least level against anybody. A dual that had a
		// target both halves were bad into would be a character with a hole no
		// kit could cover.
		if best < neutral {
			t.Errorf("against %s the better of %s still reads %d per mille: a dual with no answer at all is worse than either element alone",
				other.ID, who.Element, best)
		}
		if best > neutral && worst < neutral {
			disagreed++
		}
	}
	if disagreed == 0 {
		t.Errorf("%s never has to choose: no shipped character is one its two elements disagree about, so the second element is a bonus rather than a decision",
			who.Element)
	}
	t.Logf("%s disagrees with itself about %d of the shipped cast", who.Element, disagreed)
}
