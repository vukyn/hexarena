package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

// What is left here is the wiring: which menu entry opens which reference, and
// that esc comes back. What each of them *draws*, and how its cursor walks,
// moved to internal/screen with the screens.

// menuTo walks the menu cursor onto the entry that opens a screen and presses
// enter, which is the only way a reader reaches either of these references.
//
// Driven rather than assigned, because the wiring is what these tests are about:
// a screen with a view and an update and no menu entry is a screen nobody can
// open, and assigning m.screen would pass either way.
func menuTo(t *testing.T, m model, target screen) model {
	t.Helper()
	m = m.enter(screenMenu)
	for index, item := range menuItems {
		if item.target != target {
			continue
		}
		m.menu = index
		return key(t, m, "enter")
	}
	t.Fatalf("no menu entry opens screen %v", target)
	return m
}

// TestTheMenuOpensBothReferences is the wiring, in both languages.
//
// The chart and the species book were the two data tables with no screen at all:
// every listing in the tool prints an element id or a species id somewhere, and
// the only way to find out what either meant was to open elements.json or
// species.json. A reference nothing routes to is the same as no reference.
func TestTheMenuOpensBothReferences(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, target := range []struct {
			screen  screen
			heading i18n.Key
		}{
			{screenElements, i18n.ElementsHeading},
			{screenSpecies, i18n.SpeciesHeading},
		} {
			m, _, _ := start(t, lang)
			m = menuTo(t, m, target.screen)
			if m.screen != target.screen {
				t.Fatalf("%s: the menu entry landed on screen %v", lang, m.screen)
			}
			if drawn := m.screenContent(); !strings.Contains(drawn, m.text(target.heading)) {
				t.Errorf("%s: the screen it opened is not headed %q:\n%s",
					lang, m.text(target.heading), drawn)
			}
			// And esc goes back, which is the half a reader needs to use it twice.
			if back := key(t, m, "esc"); back.screen != screenMenu {
				t.Errorf("%s: esc from %v went to screen %v", lang, target.screen, back.screen)
			}
		}
	}
}

// indexOfElement is where one element sits in the listing, which is element.All
// order rather than anything the screen chooses.
func indexOfElement(t *testing.T, want element.Element) int {
	t.Helper()
	for index, member := range element.All() {
		if member == want {
			return index
		}
	}
	t.Fatalf("no element %v", want)
	return 0
}
