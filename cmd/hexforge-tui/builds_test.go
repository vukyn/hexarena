package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// What is left here is the wiring. The grouping, the note on a character with no
// build, where the cursor may land and what the pane spells out moved to
// internal/screen with the screen.

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
