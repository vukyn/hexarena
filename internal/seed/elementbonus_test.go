package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// The per-element table replaces one bonus that paid a squad for sharing
// *anything* with a table that says what each tribe is FOR. What it buys is a
// reason to build one way rather than another; what it costs is a rung per
// element that nobody can reach unless the cast can field it.
//
// ⚠️ This file is the reachability half, and it is checked BY PROPERTY rather
// than by a list. A hand-kept table of "which elements have enough carriers" goes
// stale the day a character is authored, and it goes stale silently — a bonus
// nobody can trigger loads, draws and never fires, which is the one failure
// composition.Axis exists to prevent one level up.

// carriersByElement counts the shipped cast by the elements they carry, which is
// the ceiling on every rung a per-element bonus can declare.
//
// The inert element is skipped for the reason Awards skips it: sharing the
// element with no matchup is sharing the absence of one, and no bonus may name it
// — parse refuses that outright.
func carriersByElement(t *testing.T) map[element.Element]int {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}
	counted := make(map[element.Element]int)
	for _, character := range book.All() {
		for _, member := range character.Element.Elements() {
			if isInert(chart, member) {
				continue
			}
			counted[member]++
		}
	}
	return counted
}

func isInert(chart *element.Chart, member element.Element) bool {
	for _, one := range chart.Inert() {
		if one == member {
			return true
		}
	}
	return false
}

// TestEveryElementBonusRungIsReachable is the guard the roadmap asked for five
// times over: a table declared with rungs the cast cannot field would ship with
// its top half dead and nothing would say so.
func TestEveryElementBonusRungIsReachable(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	carriers := carriersByElement(t)
	var checked int
	for _, held := range bonuses.All() {
		if held.Axis != composition.AxisElement || held.Value == "" {
			continue
		}
		member, err := element.Parse(held.Value)
		if err != nil {
			t.Fatalf("%s counts %q, which is no element: %v", held.ID, held.Value, err)
		}
		fieldable := carriers[member]
		if fieldable > hex.MaxTeamSize {
			fieldable = hex.MaxTeamSize
		}
		checked++
		for _, rung := range held.Rungs {
			if rung.At > fieldable {
				t.Errorf("%s declares a rung at %d and the cast fields %d %s carriers: "+
					"a rung nobody can reach loads, draws and never fires",
					held.ID, rung.At, carriers[member], member)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no bonus names an element, so this measures nothing")
	}
}

// TestEveryFieldableElementHasABonus is the other direction, and it is what stops
// the table being finished by forgetting.
//
// An element two of the cast carry is an element a player can build a tribe
// around. If the table has no entry for it, that player gets nothing for a squad
// that satisfies every rule the design has.
//
// ⚠️ This used to REPORT rather than fail, because `same_element` shipped and
// covered every gap — a blanket bonus paying whichever element a side happened to
// share. That escape hatch is gone with it: the eighth element got its own bonus,
// the blanket was retired, and an uncovered element is now a squad that is paid
// nothing rather than a squad paid something generic.
func TestEveryFieldableElementHasABonus(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	named := make(map[string]bool)
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value != "" {
			named[held.Value] = true
		}
	}
	for member, count := range carriersByElement(t) {
		if count < composition.MinimumRung || named[member.String()] {
			continue
		}
		t.Errorf("%s has %d carriers and no bonus of its own: a squad built around it "+
			"is paid nothing", member, count)
	}
}

// TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier is the gap the
// retirement opens, pinned so that it cannot grow.
//
// ⚠️ **`same_element` was not covering nothing.** `carriersByElement` counts the
// CAST, and TestEveryFieldableElementHasABonus reads "one carrier" as "no squad
// can field a tribe of it" — which is true of a DRAFTED squad, because the pool
// is exclusive and a drafted side is six or ten different characters by
// construction, and false of a SAVED one: `internal/draft` records both, and a
// saved squad may field the same character twice. So a saved squad of two Lapras
// reaches rung 2 on ice, and after the retirement it is paid **nothing**, where
// the blanket used to pay it.
//
// That is accepted rather than fixed, and the reason is that fixing it means
// inventing two effects for two tribes only a doubled-up squad can field — a
// bonus authored for a case nobody builds towards, which is the opposite of the
// rule the table was built under. What may not happen is the set growing quietly,
// so it is named here: exactly the elements with one carrier, and no others. A
// third element losing coverage fails, and so does a second light or ice
// character shipping without a bonus following it.
func TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	named := make(map[string]bool)
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value != "" {
			named[held.Value] = true
		}
	}
	for member, count := range carriersByElement(t) {
		switch {
		case named[member.String()] && count < composition.MinimumRung:
			t.Errorf("%s has a bonus of its own and only %d carrier(s): a rung it takes "+
				"a doubled-up saved squad to reach is not what the table was built for",
				member, count)
		case !named[member.String()] && count >= composition.MinimumRung:
			t.Errorf("%s has %d carriers and no bonus of its own", member, count)
		}
	}
}

// TestNoBlanketElementBonusShips is the retirement itself, held as a property.
//
// The per-element table and a blanket that pays for sharing *anything* are two
// answers to one question, and both paying at once is what the table was built to
// replace — a tribe would collect its own bonus and the generic one on top, so
// what a player is really choosing between is every element plus a constant.
//
// It names no id, because the thing being refused is the SHAPE. A second blanket
// under another name would be the same design arriving by another door, and a
// test naming `same_element` would let it through.
func TestNoBlanketElementBonusShips(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value == "" {
			t.Errorf("bonus %q counts the element axis and names no element: it pays every "+
				"tribe on top of the bonus that tribe already has", held.ID)
		}
	}
}
