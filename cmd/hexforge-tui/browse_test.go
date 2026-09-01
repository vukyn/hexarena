package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// What is left here is the wiring: the menu entry that opens the cast browser,
// where esc lands, and that its two keys really reach the two describers. What
// it *draws* — the level, the origin filter, the art row, the trait gate, the
// gloss widths — moved to internal/screen with the screen, because a drawing is
// not something this binary decides.

// TestTheMenuOpensTheCastBrowserAndEscComesBack is the wiring, in both
// languages, driven through the menu rather than by assigning the screen: a
// screen with a view and an update and no menu entry is a screen nobody can
// open.
//
// The esc half is the one that changed shape. The browser used to write
// screenMenu itself; it asks for a draw.Back now, and where that lands is this
// client's one-slot raisedFrom — so the claim is the client's and belongs here.
func TestTheMenuOpensTheCastBrowserAndEscComesBack(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenBrowse)
		if m.screen != screenBrowse {
			t.Fatalf("%s: the menu entry landed on screen %v", lang, m.screen)
		}
		if drawn := m.screenContent(); !strings.Contains(drawn, m.text(i18n.BrowseHeading)) {
			t.Errorf("%s: the screen it opened is not headed %q:\n%s",
				lang, m.text(i18n.BrowseHeading), drawn)
		}
		if back := key(t, m, "esc"); back.screen != screenMenu {
			t.Errorf("%s: esc went to screen %v", lang, back.screen)
		}
	}
}

// TestTheBrowserRaisesBothDescribersAndStillLeaves is the pair of raises and the
// thing they must not cost.
//
// ⚠️ The second half is the whole reason this test is two trips rather than two
// assertions. A raise writes the way back into raisedFrom, and both describers
// answer esc **themselves** rather than through navigate — so unless each of
// them forgets the slot as it uses it, a reader who has looked at the art or at
// the traits and come back finds the browser's own esc sending them to the
// browser. That is a screen nobody can leave, and it costs nothing to reach: two
// keystrokes.
func TestTheBrowserRaisesBothDescribersAndStillLeaves(t *testing.T) {
	for _, trip := range []struct {
		name   string
		key    tea.KeyPressMsg
		raised screen
	}{
		{"the art preview", tea.KeyPressMsg{Code: 'p', Text: "p"}, screenPreview},
		{"the description", tea.KeyPressMsg{Code: '?', Text: "?"}, screenBlurb},
	} {
		t.Run(trip.name, func(t *testing.T) {
			m, _, _ := start(t, i18n.Vi)
			m = menuTo(t, m, screenBrowse)

			m = send(t, m, trip.key)
			if m.screen != trip.raised {
				t.Fatalf("the key left the reader on screen %v, want %v", m.screen, trip.raised)
			}
			// The description screen is the one that keeps its own way back, and
			// it is filled in by the raise rather than by the describer: its two
			// other raisers never come through navigate at all.
			if trip.raised == screenBlurb && m.blurb.from != screenBrowse {
				t.Errorf("the description thinks it was raised from screen %v", m.blurb.from)
			}
			m = key(t, m, "esc")
			if m.screen != screenBrowse {
				t.Fatalf("esc from %s went to screen %v, want the browser", trip.name, m.screen)
			}
			if m.raisedFrom != screenMenu {
				t.Errorf("the way back survived being used: it still reads screen %v",
					m.raisedFrom)
			}
			if left := key(t, m, "esc"); left.screen != screenMenu {
				t.Errorf("after %s, esc from the browser went to screen %v",
					trip.name, left.screen)
			}
		})
	}
}
