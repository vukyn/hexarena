package main

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
	m, lib, _ := start(t, i18n.Vi)
	rows := m.enter(screenStatuses).statuses.rows

	seen := make(map[string]bool)
	heading := status.Category(0)
	headed := false
	for _, row := range rows {
		if row.heading {
			heading, headed = row.category, true
			continue
		}
		if !headed {
			t.Fatalf("%q is listed before any category heading", row.kind.ID)
		}
		if row.kind.Category != heading {
			t.Errorf("%q is a %s and is filed under %s", row.kind.ID, row.kind.Category, heading)
		}
		seen[row.kind.ID] = true
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
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenStatuses)
	check := func(where string) {
		t.Helper()
		row := m.statuses.rows[m.statuses.cursor]
		if row.heading {
			t.Fatalf("%s: the cursor sits on the %s heading", where, row.category)
		}
	}
	check("on entering")
	for range len(m.statuses.rows) + 2 {
		m = key(t, m, "down")
		check("walking down")
	}
	for range len(m.statuses.rows) + 2 {
		m = key(t, m, "up")
		check("walking up")
	}
	// A refresh re-files every row, and the cursor it keeps is an index into the
	// list it had before.
	m.statuses = m.statuses.refresh(m.lib)
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
func TestTheStatusCaveatSurvivesTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenStatuses)
		drawn := m.screenContent()
		if !strings.Contains(drawn, m.text(i18n.BlurbStatusCaveat)) {
			t.Errorf("%s: the caveat is not on the smallest screen:\n%s", lang, drawn)
		}
		if strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: the listing is cut at the smallest size:\n%s", lang, drawn)
		}
	}
}

// TestTheStatusListingDescribesWhatIsUnderTheCursor is the pane the screen is
// for: the rows carry a name and the description carries everything else, so a
// cursor that moved without the pane following would leave the two disagreeing.
func TestTheStatusListingDescribesWhatIsUnderTheCursor(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenStatuses)
	for step := range 6 {
		selected := m.statuses.rows[m.statuses.cursor].kind
		drawn := m.screenContent()
		for _, line := range strings.Split(m.lang.DescribeStatus(selected), "\n") {
			if !strings.Contains(drawn, line) {
				t.Fatalf("step %d: the cursor is on %q and the screen does not say %q:\n%s",
					step, selected.ID, line, drawn)
			}
		}
		m = key(t, m, "down")
	}
}
