package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The three keystrokes that cross screens: ? on a trait raises the statuses
// reference on the status that trait names, and esc comes back to the trait.
// Those are this client's raise and its one-slot memory of where Back goes, so
// they stay here — while Marked, which the traits listing draws its sentences
// with, moved to internal/screen with the listing.

// ask is the ? key, which the helper map has no name for because it is a
// printable rune rather than a named key.
func ask(t *testing.T, m model) model {
	t.Helper()
	return send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
}

// onTrait puts the traits listing's cursor on a named trait.
func onTrait(t *testing.T, m model, id string) model {
	t.Helper()
	for index, held := range m.passives.Passives {
		if held.ID == id {
			m.passives.Cursor = index
			return m
		}
	}
	t.Fatalf("no trait %q in the listing", id)
	return m
}

// TestAskingAboutATraitOpensTheStatusItNames is the jump the traits listing was
// missing.
//
// The listing tells a reader that endurance "luôn mang kiên cường" and had no
// way to say what kiên cường is — the one thing the sentence leaves open, and
// the reference that answers it was two screens and a menu away.
func TestAskingAboutATraitOpensTheStatusItNames(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	m = onTrait(t, m, "endurance")

	m = ask(t, m)
	if m.screen != screenStatuses {
		t.Fatalf("? on a trait left the reader on screen %v", m.screen)
	}
	row := m.statuses.Rows[m.statuses.Cursor]
	if row.Heading {
		t.Fatal("the jump landed on a category heading, which describes nothing")
	}
	if row.Kind.ID != "toughened" {
		t.Errorf("? on endurance opened %q, want the toughened it grants", row.Kind.ID)
	}
	// And what it opened is a description rather than a row: the reference is
	// worth the jump only if the sentence the trait would not say is now on
	// screen.
	drawn := m.screenContent()
	opening := strings.SplitN(m.lang.DescribeStatus(row.Kind), "\n", 2)[0]
	if !strings.Contains(drawn, firstWords(opening)) {
		t.Errorf("the status screen does not describe toughened:\n%s", drawn)
	}
}

// TestComingBackFromAStatusReturnsToTheTrait is the other half of one keystroke.
//
// The statuses listing is reached two ways now, and esc went to the menu from
// both. A reader sent here by ? from a trait has not finished with that trait,
// and dropping them at the menu makes them walk back in through it.
func TestComingBackFromAStatusReturnsToTheTrait(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	m = onTrait(t, m, "virulence")
	before := m.passives.Cursor

	m = ask(t, m)
	m = key(t, m, "esc")
	if m.screen != screenPassives {
		t.Fatalf("esc after a jump from a trait landed on screen %v", m.screen)
	}
	if m.passives.Cursor != before {
		t.Errorf("the trait cursor moved from %d to %d across the jump",
			before, m.passives.Cursor)
	}
	// And the way back is forgotten once used: esc from the statuses listing
	// reached through the menu has to go to the menu, or the second visit
	// inherits the first one's history.
	m = m.enter(screenStatuses)
	m = key(t, m, "esc")
	if m.screen != screenMenu {
		t.Errorf("esc from the menu's own statuses listing went to screen %v", m.screen)
	}
}

// TestAskingAboutATraitThatNamesNoStatusStaysPut is the case the shipped data
// holds twice.
//
// blood_thirst and last_gasp only drain, and a drain names no status at all. A
// jump to whatever the cursor happened to be on would answer a question the
// reader did not ask, and is worse than not moving.
func TestAskingAboutATraitThatNamesNoStatusStaysPut(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenPassives)
	for _, id := range []string{"blood_thirst", "last_gasp"} {
		m = onTrait(t, m, id)
		if named := i18n.StatusesNamed(m.passives.Passives[m.passives.Cursor]); len(named) != 0 {
			t.Fatalf("%s names %v, so this test measures nothing", id, named)
		}
		if after := ask(t, m); after.screen != screenPassives {
			t.Errorf("? on %s, which names nothing, moved to screen %v", id, after.screen)
		}
	}
}
