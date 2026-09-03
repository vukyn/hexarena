package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// # The form key, from this client's side
//
// A line that forks reaches two grown forms at one level, and
// progression.Line.StageAt refuses to name one rather than hand a reader
// whichever arm the file lists last. What the read-only views got instead is a
// chooser: `s` walks the arms, and the row naming the one in front is drawn on
// all three of them.
//
// ⚠️ **The screen package owns the walk and this client owns *where it is
// pressed*, which is exactly the half the sweeps cannot see.** The two
// describers keep no cursor and no level of their own, so a key pressed over one
// of them has to move the browser behind it and re-push — the same arrangement
// the level already has in describe.go, and the same one that made the preview a
// screen able to live in internal/screen at all. A form key wired into the
// browser and not into `updatePreview` would leave an author looking at a
// picture, pressing the key the picture itself offers, and watching nothing
// happen.

// TestTheFormKeyWalksTheArmsFromEitherDescriber presses `s` on the two screens
// an author can be on when they see the form row, and reads the drawn screen
// back.
func TestTheFormKeyWalksTheArmsFromEitherDescriber(t *testing.T) {
	base, _, _ := start(t, i18n.En)
	forked := theForkedBrowser(t, base)
	character := forked.browse.Rows()[forked.browse.Cursor]
	arms, err := character.FurthestAt(forked.browse.Level)
	if err != nil || len(arms) < 2 {
		t.Fatalf("%s does not fork at level %d", character.ID, forked.browse.Level)
	}

	for _, raise := range []struct {
		name string
		key  string
		want screen
	}{
		{"the art preview", "p", screenPreview},
		{"the traits blurb", "?", screenBlurb},
	} {
		raised := typeText(t, forked, raise.key)
		if raised.screen != raise.want {
			t.Fatalf("%q on the forked row landed on screen %v", raise.key, raised.screen)
		}
		if drawn := raised.screenContent(); !strings.Contains(drawn,
			raised.text(i18n.FormChoice, arms[0].Name, 1, len(arms))) {
			t.Fatalf("%s opens without naming which arm it is showing:\n%s", raise.name, drawn)
		}
		walked := typeText(t, raised, "s")
		if walked.screen != raise.want {
			t.Errorf("s left %s for screen %v", raise.name, walked.screen)
		}
		drawn := walked.screenContent()
		if want := walked.text(i18n.FormChoice, arms[1].Name, 2, len(arms)); !strings.Contains(drawn, want) {
			t.Errorf("s on %s does not move it to %s:\n%s", raise.name, arms[1].Name, drawn)
		}
		// And the browser behind moved with it, which is what makes going back
		// land on the arm the author chose rather than on the one it opened with.
		if got := walked.browse.Subject().Stage; got != arms[1].Name {
			t.Errorf("s on %s left the browser behind on %q, want %s",
				raise.name, got, arms[1].Name)
		}
	}
}

// TestTheFormKeyDoesNothingOnALineThatDoesNotFork is the other half, and it is
// what keeps `s` from being a key that quietly does something on every
// character.
//
// The key is live on every row because the screens are one screen; what a
// character with one grown form must get out of it is nothing at all, and no row
// offering it.
func TestTheFormKeyDoesNothingOnALineThatDoesNotFork(t *testing.T) {
	base, _, _ := start(t, i18n.En)
	browser := base.enter(screenBrowse)
	before := browser.browse.Subject()
	arms, err := browser.browse.Rows()[browser.browse.Cursor].FurthestAt(browser.browse.Level)
	if err != nil || len(arms) != 1 {
		t.Fatalf("the row this listing opens on forks, so this measures the wrong case")
	}
	if drawn := browser.screenContent(); strings.Contains(drawn, "< "+arms[0].Name+" >") {
		t.Errorf("a line that does not fork draws a form chooser:\n%s", drawn)
	}
	if after := typeText(t, browser, "s").browse.Subject(); after != before {
		t.Errorf("s moved a character with one form: %+v became %+v", before, after)
	}
}
