package screen

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheStatusesCatalogueSaysWhatNoCountdownMeans is the new home of a rule
// that used to live over a column, and this is the test that it really arrived.
//
// ⚠️ **A rule moved out of one screen has to be drawn by another or it is
// deleted.** The roster's heading dropped its parenthesis, which is a saving of
// a line in every window of every battle; what a reader is owed in exchange is a
// place the convention is stated at all, and "it is in the catalogue" is a claim
// about a drawing rather than about a wording. So this renders the catalogue and
// looks for the line.
//
// Three things are asserted and each answers a different way of getting this
// wrong:
//
//   - The line is **in the body**, in both languages. A key with no caller is a
//     wording nobody can read, and i18n's orphan test would not have caught one
//     referenced from a comment.
//   - The line is **inside the room the frame leaves**, at the floor as well as
//     in a roomy window. The frame cuts from the bottom and this line is at the
//     bottom, so a room that did not reserve for it would put the convention on
//     a row nobody ever sees — which is the same as not moving it at all.
//   - The line uses **BlurbStatusAlways' own word** for permanence, which is the
//     rule the roster heading carried before it: two Vietnamese terms for one
//     property would be this module's glossary disagreeing with itself between
//     the screen that states the rule and the descriptions under it.
func TestTheStatusesCatalogueSaysWhatNoCountdownMeans(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		screen := NewStatusesScreen(lib)
		if len(screen.Rows) == 0 {
			t.Fatalf("%v: the catalogue has no rows, so it is not the screen this rule was "+
				"moved onto", lang)
		}
		notation := c.Text(i18n.StatusesNoCountdown)
		if strings.TrimSpace(notation) == "" {
			t.Fatalf("%v has no wording for the notation, so every line below is about a "+
				"blank", lang)
		}
		if permanent := c.Text(i18n.BlurbStatusAlways); !strings.Contains(notation, permanent) {
			t.Errorf("%v states the notation as %q, which does not use the word the "+
				"descriptions under it use for permanence (%q)", lang, notation, permanent)
		}
		for _, window := range []struct {
			name string
			ctx  Context
		}{
			{"at the floor", atTheFloor(c)},
			{"in a roomy window", roomyContext(c)},
		} {
			body, _ := screen.View(window.ctx)
			rows := drawnLines(body)
			room := bodyRoom(window.ctx)
			found := -1
			for index, row := range rows {
				if strings.Contains(row, notation) {
					found = index
				}
			}
			if found < 0 {
				t.Errorf("%v %s: the catalogue never states the notation %q:\n%s",
					lang, window.name, notation, body)
				continue
			}
			if found >= room {
				t.Errorf("%v %s: the notation is on row %d of a body the frame cuts to %d, "+
					"so no reader ever sees it", lang, window.name, found+1, room)
			}
			// The line has to fit across as well as down: a wording wider than
			// the window is one the frame clips mid-sentence.
			if width, budget := lipgloss.Width(rows[found]), window.ctx.Width-1; width > budget {
				t.Errorf("%v %s: the notation row is %d cells against a budget of %d:\n%s",
					lang, window.name, width, budget, rows[found])
			}
		}
	}
}

// TestTheRosterHeadingNoLongerCarriesTheRule is the other side of that move, and
// it is the half that would otherwise be held by a golden this change accepts.
//
// The parenthesis is gone from the heading in **both** languages — that is what
// the owner asked for — and the check is written against the words rather than
// against a length, because a heading trimmed to fit would satisfy a length.
func TestTheRosterHeadingNoLongerCarriesTheRule(t *testing.T) {
	for _, lang := range i18n.Langs() {
		heading := lang.Text(i18n.RosterHeadingEffects)
		if strings.ContainsAny(heading, "()") {
			t.Errorf("%v still qualifies the effects heading: %q", lang, heading)
		}
		if permanent := lang.Text(i18n.BlurbStatusAlways); strings.Contains(heading, permanent) {
			t.Errorf("%v still states the permanence rule over the column: %q", lang, heading)
		}
		// And it still names the column, or the parenthesis took the label with
		// it.
		if strings.TrimSpace(heading) == "" {
			t.Errorf("%v has no effects heading left at all", lang)
		}
	}
}

// roomyContext is the same Context in the roomy window the goldens record, which
// is the one size other than the floor anything here is drawn at.
func roomyContext(c Context) Context {
	c.Width, c.Height = roomyWindow, 60
	return c
}
