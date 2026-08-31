package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheStatusListingGroupsEveryStatusUnderItsCategory is the property the
// listing exists for.
//
// A flat list of fifteen names would be a reference that answers "what does mire
// do" and not "what does rapid_spin strip", and the second is the question a
// player is left with far more often: a cleanse names a category, and a category
// means nothing without the statuses under it.
//
// It walks the built rows rather than the drawn screen, because the drawn screen
// is windowed — a listing taller than the terminal shows a dozen of them, and
// asserting against what happens to be on screen would be asserting about the
// window rather than about the grouping.
func TestTheStatusListingGroupsEveryStatusUnderItsCategory(t *testing.T) {
	_, lib := start(t, i18n.Vi)
	rows := NewStatusesScreen(lib).Rows

	seen := make(map[string]bool)
	heading := status.Category(0)
	headed := false
	for _, row := range rows {
		if row.Heading {
			heading, headed = row.Category, true
			continue
		}
		if !headed {
			t.Fatalf("%q is listed before any category heading", row.Kind.ID)
		}
		if row.Kind.Category != heading {
			t.Errorf("%q is a %s and is filed under %s", row.Kind.ID, row.Kind.Category, heading)
		}
		seen[row.Kind.ID] = true
	}
	for _, kind := range lib.Statuses().Kinds() {
		if !seen[kind.ID] {
			t.Errorf("%q is declared and the listing never shows it", kind.ID)
		}
	}
}

// TestTheStatusCursorNeverLandsOnAHeading is what the headings-as-rows layout
// costs, asserted rather than assumed.
//
// The headings scroll with the listing, which is why they are rows at all — a
// heading drawn between rows would fall off the top of the window and leave what
// is under it unlabelled. The price is a cursor that can index one, and a cursor
// on a heading has no status to describe: the pane below would blink out for one
// keystroke, which reads as the program losing its place.
//
// Walked to both ends and back, because the two ways to land on one are
// different: stepping onto a boundary, and settling after a refresh.
func TestTheStatusCursorNeverLandsOnAHeading(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	statuses := NewStatusesScreen(lib)
	check := func(where string) {
		t.Helper()
		row := statuses.Rows[statuses.Cursor]
		if row.Heading {
			t.Fatalf("%s: the cursor sits on the %s heading", where, row.Category)
		}
	}
	check("on entering")
	for range len(statuses.Rows) + 2 {
		statuses, _ = statuses.Update(c, press(t, "down"))
		check("walking down")
	}
	for range len(statuses.Rows) + 2 {
		statuses, _ = statuses.Update(c, press(t, "up"))
		check("walking up")
	}
	// A refresh re-files every row, and the cursor it keeps is an index into the
	// list it had before.
	statuses = statuses.Refresh(lib)
	check("after a refresh")
}

// TestTheStatusCaveatSurvivesTheSmallestWindow is the line most easily lost and
// least affordable to lose.
//
// The frame cuts a body that will not fit from the bottom, and the caveat is the
// last line of this one — so a listing one row too tall drops the sentence
// saying the figures above it are the book's rather than the log's. It is the
// whole reason a reader is not surprised when a poison ticks for 650, and it
// goes silently: nothing about a cut screen says which line went.
//
// ⚠️ Measured against bodyRoom, which mirrors the client's frame rather than
// being it. The frame really wrapped round this listing at 120x24 is what the
// client's screens.golden records.
func TestTheStatusCaveatSurvivesTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		c = atTheFloor(c)
		body, _ := NewStatusesScreen(lib).View(c)
		if !strings.Contains(body, c.Text(i18n.BlurbStatusCaveat)) {
			t.Errorf("%s: the caveat is not on the smallest screen:\n%s", lang, body)
		}
		if rows := len(drawnLines(body)); rows > bodyRoom(c) {
			t.Errorf("%s: the listing takes %d rows of the %d the smallest window gives it, so the frame cuts it:\n%s",
				lang, rows, bodyRoom(c), body)
		}
	}
}

// TestTheStatusListingDescribesWhatIsUnderTheCursor is the pane the screen is
// for: the rows carry a name and the description carries everything else, so a
// cursor that moved without the pane following would leave the two disagreeing.
func TestTheStatusListingDescribesWhatIsUnderTheCursor(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	statuses := NewStatusesScreen(lib)
	for step := range 6 {
		selected := statuses.Rows[statuses.Cursor].Kind
		drawn, _ := statuses.View(c)
		for _, line := range strings.Split(c.Lang.DescribeStatus(selected), "\n") {
			if !strings.Contains(drawn, line) {
				t.Fatalf("step %d: the cursor is on %q and the screen does not say %q:\n%s",
					step, selected.ID, line, drawn)
			}
		}
		statuses, _ = statuses.Update(c, press(t, "down"))
	}
}
