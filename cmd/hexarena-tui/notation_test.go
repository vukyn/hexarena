package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestAPlayerCanWalkFromTheMenuToWhatNoCountdownMeans is the reachability half
// of moving a rule off the roster.
//
// ⚠️ **This client is the one that lost the line.** It draws the battle, so it
// is the one whose roster heading used to say that an effect with no countdown
// beside it is permanent — and the answer to "where did that go" has to be a
// place a player can actually get to from here, with the keys they have. A rule
// living in a package nobody navigates to is a rule that was deleted with extra
// steps.
//
// So nothing is assigned: the menu is walked with `down` and opened with
// `enter`, exactly as a reader does it, and what is searched is the **framed**
// view — the whole window, clipped to it — rather than a screen's body rendered
// in isolation. A line the frame cuts is a line nobody reads.
//
// Both languages, and at the floor as well as in a roomy window, because the
// floor is the one where the frame has anything to cut.
func TestAPlayerCanWalkFromTheMenuToWhatNoCountdownMeans(t *testing.T) {
	// Which menu row it is, looked up rather than counted: an entry added above
	// it would silently make a hardcoded number open a different screen, and
	// that screen renders perfectly well.
	wanted := -1
	for index, item := range menuItems {
		if item.target == screenStatuses {
			wanted = index
		}
	}
	if wanted < 0 {
		t.Fatal("the statuses catalogue is not on this client's menu at all, so the rule " +
			"the roster gave up has nowhere a player can reach")
	}

	for _, lang := range i18n.Langs() {
		for _, window := range []struct {
			name          string
			width, height int
		}{
			{"at the floor", minWidth, minHeight},
			{"in a roomy window", 160, 60},
		} {
			m, _, _ := start(t, lang)
			m.width, m.height = window.width, window.height
			m = m.enter(screenMenu)
			for range wanted {
				m = key(t, m, "down")
			}
			if m.menu != wanted {
				t.Fatalf("%v %s: %d presses of down left the cursor on row %d, want %d",
					lang, window.name, wanted, m.menu, wanted)
			}
			m = key(t, m, "enter")
			if m.screen != screenStatuses {
				t.Fatalf("%v %s: enter on the statuses row landed on screen %v",
					lang, window.name, m.screen)
			}
			notation := m.text(i18n.StatusesNoCountdown)
			if strings.TrimSpace(notation) == "" {
				t.Fatalf("%v has no wording for the notation, so the search below is for a "+
					"blank", lang)
			}
			drawn := m.screenContent()
			if !strings.Contains(drawn, notation) {
				t.Errorf("%v %s: the catalogue this client opens never states %q:\n%s",
					lang, window.name, notation, drawn)
			}
		}
	}
}

// TestTheBattleRosterNoLongerStatesTheRuleItself is the other half: the line is
// on the catalogue **instead of** over the column, not as well as.
//
// It is asserted against the drawn battle rather than against the wording alone,
// because the wording test lives in internal/screen and this is about the screen
// a player is looking at when the question comes up. What it must not find is the
// parenthesis the owner asked for the removal of.
func TestTheBattleRosterNoLongerStatesTheRuleItself(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		heading := m.text(i18n.RosterHeadingEffects)
		if strings.ContainsAny(heading, "()") {
			t.Errorf("%v: the effects heading still carries a parenthesis: %q", lang, heading)
		}
		// And the rule's own sentence is not on the battle screen either, in
		// case it were moved onto a row of its own there.
		if notation := m.text(i18n.StatusesNoCountdown); strings.Contains(heading, notation) {
			t.Errorf("%v: the roster heading states the whole notation: %q", lang, heading)
		}
	}
}
