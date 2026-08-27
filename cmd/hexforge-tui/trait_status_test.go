package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// ask is the ? key, which the helper map has no name for because it is a
// printable rune rather than a named key.
func ask(t *testing.T, m model) model {
	t.Helper()
	return send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
}

// onTrait puts the traits listing's cursor on a named trait.
func onTrait(t *testing.T, m model, id string) model {
	t.Helper()
	for index, held := range m.passives.passives {
		if held.ID == id {
			m.passives.cursor = index
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
	row := m.statuses.rows[m.statuses.cursor]
	if row.heading {
		t.Fatal("the jump landed on a category heading, which describes nothing")
	}
	if row.kind.ID != "toughened" {
		t.Errorf("? on endurance opened %q, want the toughened it grants", row.kind.ID)
	}
	// And what it opened is a description rather than a row: the reference is
	// worth the jump only if the sentence the trait would not say is now on
	// screen.
	drawn := m.screenContent()
	opening := strings.SplitN(m.lang.DescribeStatus(row.kind), "\n", 2)[0]
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
	before := m.passives.cursor

	m = ask(t, m)
	m = key(t, m, "esc")
	if m.screen != screenPassives {
		t.Fatalf("esc after a jump from a trait landed on screen %v", m.screen)
	}
	if m.passives.cursor != before {
		t.Errorf("the trait cursor moved from %d to %d across the jump",
			before, m.passives.cursor)
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
		if named := i18n.StatusesNamed(m.passives.passives[m.passives.cursor]); len(named) != 0 {
			t.Fatalf("%s names %v, so this test measures nothing", id, named)
		}
		if after := ask(t, m); after.screen != screenPassives {
			t.Errorf("? on %s, which names nothing, moved to screen %v", id, after.screen)
		}
	}
}

// TestATraitsStatusNamesAreMarkedWordByWord is the marking, and the word by word
// part is the whole of it.
//
// The sentences are wrapped after they are marked, and the wrap splits on
// spaces. A style spanning "kiên cường" survives that only until the two words
// land on different lines, and then the first line opens a sequence the second
// one closes — so every word is marked whole instead.
func TestATraitsStatusNamesAreMarkedWordByWord(t *testing.T) {
	got := marked("Luôn mang kiên cường.", []string{"kiên cường"},
		func(word string) string { return "<" + word + ">" })
	if want := "Luôn mang <kiên> <cường>."; got != want {
		t.Errorf("marked %q, want %q", got, want)
	}
}

// TestAMarkedNameNeverTakesAnotherNamesOpening is the trap a replacement per
// name falls into whichever order the names are tried in.
//
// Longest first and "bỏng" matches inside the output "bỏng nặng" just produced;
// shortest first and "bỏng nặng" never matches at all. Only one left-to-right
// pass, taking the longest name that starts at each position, gets both right -
// and the first version of this marked the first word twice.
func TestAMarkedNameNeverTakesAnotherNamesOpening(t *testing.T) {
	got := marked("bỏng nặng và bỏng.", []string{"bỏng", "bỏng nặng"},
		func(word string) string { return "<" + word + ">" })
	if want := "<bỏng> <nặng> và <bỏng>."; got != want {
		t.Errorf("marked %q, want %q — one pass, longest match wins", got, want)
	}
}

// TestNothingIsMarkedInASentenceThatNamesNothing keeps the marking from being a
// search-and-replace over prose: only the names the trait actually declares are
// looked for, so a flavour clause carrying a status name by coincidence is left
// alone unless that trait names it.
func TestNothingIsMarkedInASentenceThatNamesNothing(t *testing.T) {
	sentence := "Chịu đòn quen rồi, đau tới đâu cũng đứng vững tới đó."
	if got := marked(sentence, nil, func(word string) string { return "<" + word + ">" }); got != sentence {
		t.Errorf("marked %q with no names to mark", got)
	}
}
