package screen

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
	c, lib := start(t, i18n.Vi)
	drawn, _ := NewPassivesScreen(lib).View(c)
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
	c, lib := start(t, i18n.Vi)
	traits := NewPassivesScreen(lib)
	for step := range 5 {
		selected := traits.Passives[traits.Cursor]
		drawn, _ := traits.View(c)
		for _, line := range strings.Split(c.Lang.DescribePassive(selected), "\n") {
			// The sentences wrap to the floor, so a long one is not on screen as
			// one line — its opening is enough to say the right trait is being
			// described, which is what this asserts.
			if !strings.Contains(drawn, firstWords(line)) {
				t.Fatalf("step %d: the cursor is on %q and the screen does not say %q:\n%s",
					step, selected.ID, line, drawn)
			}
		}
		traits, _ = traits.Update(c, press(t, "down"))
	}
}

// TestTheTraitListingSaysWhoLearnsEachTrait is the column that earns its place.
//
// A trait has no restriction mechanism at all — no element, no archetype, no
// species, no character — so "who may carry this" is everybody and answers
// nothing. Who actually *does* is the fact worth a column, and a trait nobody
// learns cannot reach a battle.
func TestTheTraitListingSaysWhoLearnsEachTrait(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	drawn, _ := NewPassivesScreen(lib).View(c)
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
//
// ⚠️ Measured against bodyRoom, which mirrors the client's frame rather than
// being it — see its comment.
func TestTheTraitListingFitsTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		c = atTheFloor(c)
		traits := NewPassivesScreen(lib)
		// The busiest trait, which is the one whose description is longest.
		busiest, most := 0, 0
		for index, held := range traits.Passives {
			if lines := len(strings.Split(c.Lang.DescribePassive(held), "\n")); lines > most {
				busiest, most = index, lines
			}
		}
		traits.Cursor = busiest
		body, _ := traits.View(c)
		if rows := len(drawnLines(body)); rows > bodyRoom(c) {
			t.Errorf("%s: the trait listing takes %d rows of the %d the smallest window gives it:\n%s",
				lang, rows, bodyRoom(c), body)
		}
	}
}
