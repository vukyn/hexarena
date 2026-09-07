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
// that satisfies every rule the design has — which is the state `same_element`
// currently papers over, and the reason it may not be retired until this passes
// on its own.
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
	// ⚠️ `same_element` is the fallback that makes the gaps survivable, and this
	// test is written so that it says so out loud rather than passing quietly:
	// while it ships, an unnamed element is reported and not failed.
	var blanket bool
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value == "" {
			blanket = true
		}
	}
	for member, count := range carriersByElement(t) {
		if count < composition.MinimumRung || named[member.String()] {
			continue
		}
		if blanket {
			t.Logf("%s has %d carriers and no bonus of its own; `same_element` still covers it",
				member, count)
			continue
		}
		t.Errorf("%s has %d carriers and no bonus of its own, and no blanket bonus ships: "+
			"a squad built around it is paid nothing", member, count)
	}
}
