package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheMenuOpensTheBuildCatalogue is the wiring, in both languages, driven
// through the menu rather than by assigning the screen: a screen with a view and
// an update and no menu entry is a screen nobody can open.
func TestTheMenuOpensTheBuildCatalogue(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenBuilds)
		if m.screen != screenBuilds {
			t.Fatalf("%s: the menu entry landed on screen %v", lang, m.screen)
		}
		if drawn := m.screenContent(); !strings.Contains(drawn, m.text(i18n.BuildsHeading)) {
			t.Errorf("%s: the screen it opened is not headed %q:\n%s",
				lang, m.text(i18n.BuildsHeading), drawn)
		}
		// And esc goes back, which is the half a reader needs to use it twice.
		if back := key(t, m, "esc"); back.screen != screenMenu {
			t.Errorf("%s: esc went to screen %v", lang, back.screen)
		}
	}
}

// TestTheBuildListingGroupsEveryBuildUnderItsCharacter is the property the screen
// exists for.
//
// A flat list of six names would answer "what is rải độc" and not "what are my
// choices for Bulbasaur", and the second is the question a build is authored to
// settle: a build only means anything against the other directions the same
// learnset could have gone.
//
// It walks the built rows rather than the drawn screen, because the drawn screen
// is windowed — at the floor it shows eight of them — and asserting against what
// happens to be on screen would be asserting about the window rather than about
// the grouping.
func TestTheBuildListingGroupsEveryBuildUnderItsCharacter(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	rows := m.enter(screenBuilds).builds.rows

	seen := make(map[string]bool)
	heading, headed := "", false
	for _, row := range rows {
		if row.heading {
			heading, headed = row.character.ID, true
			continue
		}
		if !headed {
			t.Fatalf("the build %q is listed before any character", row.built.ID)
		}
		if row.built.Character != heading {
			t.Errorf("the build %q is for %q and is filed under %q",
				row.built.ID, row.built.Character, heading)
		}
		seen[row.built.ID] = true
	}
	for _, built := range lib.Builds() {
		if !seen[built.ID] {
			t.Errorf("%q is authored and the listing never shows it", built.ID)
		}
	}
}

// TestACharacterWithNoBuildSaysSoRatherThanVanishing is the honest half of the
// listing, and the half a screen built from the catalogue would get wrong.
//
// Walking the catalogue and grouping what it holds is the obvious way to draw
// this, and it silently omits every character nobody has written a direction for —
// so the reader is told the cast is shorter than it is, and the character left off
// is the one most worth noticing. Naruto is that character today.
//
// The note is asserted on the same *line* as the id, which is the second thing
// this fixes rather than a detail of the layout: on a row of its own it could
// scroll away from the character it was about, and at the floor it did — the first
// thing on screen was "no build written for this one yet" with the name of whoever
// it meant one row above the top of the window.
func TestACharacterWithNoBuildSaysSoRatherThanVanishing(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, lib, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenBuilds)

		bare := 0
		for _, character := range lib.Characters().All() {
			if len(lib.BuildsOf(character.ID)) > 0 {
				continue
			}
			bare++
			found := false
			for _, row := range m.builds.rows {
				if row.heading && row.character.ID == character.ID {
					found = true
					if !row.empty {
						t.Errorf("%s: %s has no build and its heading does not say so",
							lang, character.ID)
					}
				}
			}
			if !found {
				t.Errorf("%s: %s has no build and the listing leaves it out",
					lang, character.ID)
			}
		}
		if bare == 0 {
			t.Fatalf("%s: every character has a build, so nothing here proves anything", lang)
		}
		note := m.text(i18n.BuildsNoneForThisOne)
		said := 0
		for _, line := range strings.Split(m.screenContent(), "\n") {
			if !strings.Contains(line, note) {
				continue
			}
			said++
			named := false
			for _, character := range lib.Characters().All() {
				if strings.Contains(line, character.ID) {
					named = true
				}
			}
			if !named {
				t.Errorf("%s: %q is drawn with nobody on the line it is about: %q",
					lang, note, line)
			}
		}
		if said == 0 {
			t.Errorf("%s: the note is nowhere on the screen:\n%s", lang, m.screenContent())
		}
	}
}

// TestTheBuildCursorNeverLandsOnACharacter is what the headings-as-rows layout
// costs, asserted rather than assumed — the same price the status reference pays.
//
// A cursor on a heading has no loadout to describe, so the pane below would blink
// out for one keystroke, which reads as the program losing its place. The rows a
// heading can be are two here rather than one: an ordinary character with builds
// under it, and a character with none, which is a heading with no build after it
// at all and therefore the one a naive step lands on.
//
// Walked to both ends and back, because the two ways to land on one differ:
// stepping onto a boundary, and settling after a refresh.
func TestTheBuildCursorNeverLandsOnACharacter(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenBuilds)
	check := func(where string) {
		t.Helper()
		row := m.builds.rows[m.builds.cursor]
		if row.heading {
			t.Fatalf("%s: the cursor sits on %s, which is a character", where, row.character.ID)
		}
	}
	check("on entering")
	for range len(m.builds.rows) + 2 {
		m = key(t, m, "down")
		check("walking down")
	}
	for range len(m.builds.rows) + 2 {
		m = key(t, m, "up")
		check("walking up")
	}
	// A refresh re-files every row, and the cursor it keeps is an index into the
	// list it had before.
	m.builds = m.builds.refresh(m.lib)
	check("after a refresh")
}

// TestTheBuildListingSpellsOutWhatIsUnderTheCursor is the pane the screen is for.
//
// The rows carry a name and the pane carries the loadout, so a cursor that moved
// without the pane following would leave the two disagreeing — and the loadout is
// the whole content: a build's name says which direction it is and only the four
// skills and the one trait say what that direction does.
//
// Both languages, at the floor, because the two draw different things: the names
// beside the ids are Vietnamese data and are dropped in English, while the intent
// is prose with no id to fall back to and is printed in both.
func TestTheBuildListingSpellsOutWhatIsUnderTheCursor(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenBuilds)
		steps := 0
		for step := range len(m.builds.rows) {
			selected, found := m.builds.selected()
			if !found {
				t.Fatalf("%s: step %d left the cursor on no build", lang, step)
			}
			steps++
			drawn := m.screenContent()
			if strings.Contains(drawn, m.text(i18n.Truncated)) {
				t.Fatalf("%s: the screen is cut at %dx%d:\n%s", lang, minWidth, minHeight, drawn)
			}
			for _, id := range selected.built.Skills {
				if !strings.Contains(drawn, id) {
					t.Errorf("%s: the cursor is on %q and the screen does not name %q:\n%s",
						lang, selected.built.ID, id, drawn)
				}
			}
			for _, id := range selected.built.Passives {
				if !strings.Contains(drawn, id) {
					t.Errorf("%s: the cursor is on %q and the screen does not name its trait %q:\n%s",
						lang, selected.built.ID, id, drawn)
				}
			}
			// The intent is why the direction exists, and it is the one line no other
			// screen in the tool can show.
			//
			// Searched for across the wrap rather than in one line of the screen: an
			// intent is a clause and the floor is eighty cells, so it arrives on two
			// rows with the label column's spaces in front of the second. Asserting
			// on the opening alone would pass with the tail lost, which is the one
			// failure wrapping can produce.
			if intent := selected.built.Intent; intent != "" &&
				!strings.Contains(flattened(drawn), flattened(intent)) {
				t.Errorf("%s: the cursor is on %q and the screen does not say %q:\n%s",
					lang, selected.built.ID, intent, drawn)
			}
			next := key(t, m, "down")
			if next.builds.cursor == m.builds.cursor {
				break
			}
			m = next
		}
		if steps < 2 {
			t.Errorf("%s: the listing walked %d builds, so it proves nothing about following the cursor",
				lang, steps)
		}
	}
}

// flattened is a screen, or a sentence, with every run of whitespace as one
// space, so that a line the layout has wrapped can still be searched for whole.
func flattened(text string) string { return strings.Join(strings.Fields(text), " ") }

// TestABuildThatTakesNoTraitSaysSo is the empty slot, which is a decision rather
// than a gap.
//
// cast.ParseBuilds insists on the kit and not on the trait — a unit with no skills
// cannot act, while a unit with no trait is an ordinary one — so a build may spend
// four slots and leave the fifth. The cast browser draws no traits row for a
// character that has none, and that is right there: a character simply has what it
// has. Here the slot was either spent or deliberately left, and an absent row
// cannot tell a reader which of the two they are looking at.
//
// Nothing shipped is traitless, which is exactly why the state is built by hand:
// the alternative is a row nobody ever renders.
func TestABuildThatTakesNoTraitSaysSo(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.builds = withNoTraitTaken(t, m.enter(screenBuilds).builds)
		m.screen = screenBuilds
		drawn := m.screenContent()
		if !strings.Contains(drawn, m.text(i18n.BuildsNoTrait)) {
			t.Errorf("%s: a build with no trait draws no row saying so:\n%s", lang, drawn)
		}
		// And the label is still there, so the empty slot reads as the trait row
		// rather than as a stray sentence.
		if !strings.Contains(drawn, m.text(i18n.LabelTraits)) {
			t.Errorf("%s: the traits row is gone rather than empty:\n%s", lang, drawn)
		}
	}
}
