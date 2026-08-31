package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The catalogue: how it is grouped, what it says about a character nobody has
// written a direction for, where its cursor may land and what the pane under it
// spells out. Which menu entry opens it is the client's and stays there.

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
	_, lib := start(t, i18n.Vi)
	rows := NewBuildsScreen(lib).Rows

	seen := make(map[string]bool)
	heading, headed := "", false
	for _, row := range rows {
		if row.Heading {
			heading, headed = row.Character.ID, true
			continue
		}
		if !headed {
			t.Fatalf("the build %q is listed before any character", row.Built.ID)
		}
		if row.Built.Character != heading {
			t.Errorf("the build %q is for %q and is filed under %q",
				row.Built.ID, row.Built.Character, heading)
		}
		seen[row.Built.ID] = true
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
		c, lib := start(t, lang)
		c = atTheFloor(c)
		builds := NewBuildsScreen(lib)

		bare := 0
		for _, character := range lib.Characters().All() {
			if len(lib.BuildsOf(character.ID)) > 0 {
				continue
			}
			bare++
			found := false
			for _, row := range builds.Rows {
				if row.Heading && row.Character.ID == character.ID {
					found = true
					if !row.Empty {
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
		note := c.Text(i18n.BuildsNoneForThisOne)
		body, _ := builds.View(c)
		said := 0
		for _, line := range strings.Split(body, "\n") {
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
			t.Errorf("%s: the note is nowhere on the screen:\n%s", lang, body)
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
	c, lib := start(t, i18n.Vi)
	builds := NewBuildsScreen(lib)
	check := func(where string) {
		t.Helper()
		row := builds.Rows[builds.Cursor]
		if row.Heading {
			t.Fatalf("%s: the cursor sits on %s, which is a character", where, row.Character.ID)
		}
	}
	check("on entering")
	for range len(builds.Rows) + 2 {
		builds, _ = builds.Update(c, press(t, "down"))
		check("walking down")
	}
	for range len(builds.Rows) + 2 {
		builds, _ = builds.Update(c, press(t, "up"))
		check("walking up")
	}
	// A refresh re-files every row, and the cursor it keeps is an index into the
	// list it had before.
	builds = builds.Refresh(lib)
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
		c, lib := start(t, lang)
		c = atTheFloor(c)
		builds := NewBuildsScreen(lib)
		steps := 0
		for step := range len(builds.Rows) {
			selected, found := builds.selected()
			if !found {
				t.Fatalf("%s: step %d left the cursor on no build", lang, step)
			}
			steps++
			drawn, _ := builds.View(c)
			if rows := len(drawnLines(drawn)); rows > bodyRoom(c) {
				t.Fatalf("%s: the screen takes %d rows of the %d a %dx%d window gives it:\n%s",
					lang, rows, bodyRoom(c), MinWidth, MinHeight, drawn)
			}
			for _, id := range selected.Built.Skills {
				if !strings.Contains(drawn, id) {
					t.Errorf("%s: the cursor is on %q and the screen does not name %q:\n%s",
						lang, selected.Built.ID, id, drawn)
				}
			}
			for _, id := range selected.Built.Passives {
				if !strings.Contains(drawn, id) {
					t.Errorf("%s: the cursor is on %q and the screen does not name its trait %q:\n%s",
						lang, selected.Built.ID, id, drawn)
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
			if intent := selected.Built.Intent; intent != "" &&
				!strings.Contains(flattened(drawn), flattened(intent)) {
				t.Errorf("%s: the cursor is on %q and the screen does not say %q:\n%s",
					lang, selected.Built.ID, intent, drawn)
			}
			next, _ := builds.Update(c, press(t, "down"))
			if next.Cursor == builds.Cursor {
				break
			}
			builds = next
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
		c, lib := start(t, lang)
		drawn, _ := withNoTraitTaken(t, NewBuildsScreen(lib)).View(c)
		if !strings.Contains(drawn, c.Text(i18n.BuildsNoTrait)) {
			t.Errorf("%s: a build with no trait draws no row saying so:\n%s", lang, drawn)
		}
		// And the label is still there, so the empty slot reads as the trait row
		// rather than as a stray sentence.
		if !strings.Contains(drawn, c.Text(i18n.LabelTraits)) {
			t.Errorf("%s: the traits row is gone rather than empty:\n%s", lang, drawn)
		}
	}
}

// withNoTraitTaken is the build catalogue as a build that spends no trait slot
// draws it: the build under the cursor with its trait taken off, which is what
// puts the "takes no trait" row on screen.
//
// The rows are copied rather than written into, because a screen is a value and a
// shared slice would empty the build on every copy of it.
//
// It is the client's own fixture helper of the same name, copied rather than
// shared: `everyScreen` registers this state too, so the client needs its copy,
// and reaching across for one would tie two test suites together.
func withNoTraitTaken(t *testing.T, b BuildsScreen) BuildsScreen {
	t.Helper()
	rows := append([]BuildRow(nil), b.Rows...)
	found := false
	for index, row := range rows {
		if !row.Build() {
			continue
		}
		rows[index].Built.Passives = nil
		b.Cursor, found = index, true
		break
	}
	if !found {
		t.Fatal("the catalogue holds no build, so there is no trait to take off")
	}
	b.Rows = rows
	return b
}
