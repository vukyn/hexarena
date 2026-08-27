package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheTraitListingNamesEveryDeclaredTrait is the question this screen exists
// for, and it is a different question from the browser's.
//
// The browser's `?` asks "what is this character carrying" and is filtered by a
// level, so a trait is only reachable through a character that already has it —
// which means a trait nobody has learned yet is reachable from nowhere at all.
// This asks "what traits are there", which is what somebody has before they know
// which character to look at.
func TestTheTraitListingNamesEveryDeclaredTrait(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	drawn := m.screenContent()
	for _, held := range lib.Passives().All() {
		if !strings.Contains(drawn, held.ID) {
			t.Errorf("%q is declared and the listing never shows it:\n%s", held.ID, drawn)
		}
	}
}

// TestTheTraitListingDescribesWhatIsUnderTheCursor is the pane that makes it a
// reference rather than an index: the rows carry an id, a name and who learns
// it, and everything else about a trait is in the sentences below.
func TestTheTraitListingDescribesWhatIsUnderTheCursor(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	for step := range 5 {
		selected := m.passives.passives[m.passives.cursor]
		drawn := m.screenContent()
		for _, line := range strings.Split(m.lang.DescribePassive(selected), "\n") {
			// The sentences wrap to the floor, so a long one is not on screen as
			// one line — its opening is enough to say the right trait is being
			// described, which is what this asserts.
			if !strings.Contains(drawn, firstWords(line)) {
				t.Fatalf("step %d: the cursor is on %q and the screen does not say %q:\n%s",
					step, selected.ID, line, drawn)
			}
		}
		m = key(t, m, "down")
	}
}

// TestTheTraitListingSaysWhoLearnsEachTrait is the column that earns its place.
//
// A trait has no restriction mechanism at all — no element, no archetype, no
// species, no character — so "who may carry this" is everybody and answers
// nothing. Who actually *does* is the fact worth a column, and a trait nobody
// learns cannot reach a battle.
func TestTheTraitListingSaysWhoLearnsEachTrait(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	drawn := m.screenContent()
	named := 0
	for _, held := range lib.Passives().All() {
		carriers := lib.TraitCarriers(held.ID)
		if len(carriers) == 0 {
			continue
		}
		named++
		// The first carrier only: a row is clipped to the window, so a trait
		// three characters learn may not show all three, and the assertion has to
		// be about the column existing rather than about it being complete.
		if !strings.Contains(drawn, carriers[0].Character) {
			t.Errorf("%q is learned by %q and the listing does not say so:\n%s",
				held.ID, carriers[0].Character, drawn)
		}
	}
	if named == 0 {
		t.Skip("no shipped trait is learned by anybody, so there is no column to check")
	}
}

// TestTheTraitListingFitsTheSmallestWindow is the same measurement the status
// reference takes, and for the same reason: the description is the last thing on
// the screen and the frame cuts from the bottom.
func TestTheTraitListingFitsTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenPassives)
		// The busiest trait, which is the one whose description is longest.
		busiest, most := 0, 0
		for index, held := range m.passives.passives {
			if lines := len(strings.Split(m.lang.DescribePassive(held), "\n")); lines > most {
				busiest, most = index, lines
			}
		}
		m.passives.cursor = busiest
		drawn := m.screenContent()
		if strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: the trait listing is cut at the smallest size:\n%s", lang, drawn)
		}
	}
}
