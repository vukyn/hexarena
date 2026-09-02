package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The typed filter on the skill listing. Two things are being held down here and
// only one of them is the feature.
//
// The feature is that a query narrows the rows, by id or by the Vietnamese name,
// without a Vietnamese keyboard. The other thing is the defect the feature
// introduces: skillsScreen.cursor counts the rows the filter left, so every read
// of it that reached into the book instead would act on a different skill than
// the one under the marker — and the two agree exactly while nothing is typed,
// which is every other test in this package. So the tests below that matter most
// are the ones that filter to something whose first match is *not* the first
// skill in the book and then press the keys that read a row.

// The two queries the fixture is driven with, hardcoded because they are design
// facts about the data rather than layout: the shipped book calls its five dragon
// skills "long nộ", "long vũ", "long trảo", "cuồng nộ long" and "long xung",
// and **no id in either book holds "long"** — so the query exercises the *name*
// half of the match, and its first hit is the seventeenth skill declared rather
// than the first. `dragon` is the same five rows less one, reached through the
// *id* half, which is what tells the two halves apart.
//
// Nothing anywhere holds a z, in either language: a Vietnamese word cannot and no
// shipped or fixture id does.
const (
	someSkillQuery   = "long"
	someIDSkillQuery = "dragon"
	noSkillQuery     = "zzzz"
)

// openSkillFilter enters the listing and presses the key that opens the filter.
func openSkillFilter(t *testing.T, m model) model {
	t.Helper()
	opened := typeText(t, m.enter(screenSkills), "/")
	if !opened.skills.Filtering {
		t.Fatal("/ did not open the filter on the skill listing")
	}
	return opened
}

// filterSkillsTo opens the filter, types a query and hands the keyboard back to
// the rows with enter — which is the state every key that reads a row is asked
// about below.
func filterSkillsTo(t *testing.T, m model, query string) model {
	t.Helper()
	filtered := key(t, typeText(t, openSkillFilter(t, m), query), "enter")
	if filtered.skills.Filtering {
		t.Fatal("enter did not hand the keyboard back to the listing")
	}
	if filtered.skills.Query != query {
		t.Fatalf("the filter holds %q after typing %q", filtered.skills.Query, query)
	}
	return filtered
}

// TestTheSkillListingFitsTheSmallestWindowWhileFiltering is the other half of
// skillsRoom's arithmetic: the filter row is the tenth line the reserve pays for,
// and the reserve is what stops the listing overflowing its budget.
//
// The busiest filtered state is the field open with **nothing** typed, not a
// query in it: an empty query hides nothing, so the listing is as long as the
// window allows and the row above it costs the same either way. The cursor sits
// on a skill with a condition and an edit has been reported, which is what the
// unfiltered measurement (TestTheSkillListingFitsTheSmallestWindowAfterAnEdit)
// found to be the two lines that make the case worth measuring.
// Height only. The lines this screen can draw past the window are the two that
// carry the data directory's path — the header's and the write note's — and those
// are free text that frame clips on purpose; the width sweep in language_test.go
// is what measures the rest, with the filter's three states registered in
// everyScreen so it can see them at all.
func TestTheSkillListingFitsTheSmallestWindowWhileFiltering(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, query := range []string{"", someSkillQuery, noSkillQuery} {
			m, _, _ := start(t, lang)
			m.width, m.height = minWidth, minHeight
			m = m.enter(screenSkills)
			m = skillListTo(t, m, "detonate")
			if selected, _ := m.skills.Selected(); selected.Requires == nil {
				t.Fatalf("%s no longer has a condition, so this measures the wrong case",
					selected.ID)
			}
			m = typeText(t, openSkillFilter(t, m), query)
			m.skills.Edited = someSkillChange(t, m)
			drawn := m.screenContent()
			if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
				strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
				t.Errorf("the %s listing filtered by %q is truncated at %dx%d:\n%s",
					lang, query, minWidth, minHeight, drawn)
			}
		}
	}
}

// TestEscapeLeavesTheSkillListingByGoingBack is the client half of the listing's
// escape, and the half internal/screen cannot state.
//
// The screen asks for a draw.Back — it names no view, because a view is this
// binary's own enum — so *where* that lands is a claim only this package can
// make. It is asserted from both ends: an escape while the filter is open closes
// the field and stays, and an escape on the plain listing goes back.
//
// ⚠️ The pair is one test rather than two, because the first is what says the
// second is about the screen. A screen that left on every escape would pass an
// "escape reaches the menu" test perfectly and would make the filter impossible
// to dismiss.
func TestEscapeLeavesTheSkillListingByGoingBack(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	filtering := key(t, openSkillFilter(t, m), "esc")
	if filtering.screen != screenSkills {
		t.Errorf("escape while the filter was open left for screen %v", filtering.screen)
	}

	plain := m.enter(screenSkills)
	if left := key(t, plain, "esc"); left.screen != screenMenu {
		t.Errorf("escape on the skill listing went to screen %v, want the menu it was "+
			"reached from", left.screen)
	}

	// And it really is a Back rather than a hard-coded menu: entering the listing
	// from somewhere else comes back to that somewhere else. Nothing in the
	// shipped client raises the listing, so the way back is written by hand —
	// which is exactly the fact a Back exists to carry.
	raised := m.enter(screenSkills)
	raised.raisedFrom = screenBuilds
	if left := key(t, raised, "esc"); left.screen != screenBuilds {
		t.Errorf("escape on a listing raised from the build catalogue went to screen %v",
			left.screen)
	}
}

// TestTheDescriptionRaisedFromTheListingIsTheFilteredRow is the landing half of
// the `?` key, and it is the half neither net in internal/screen can see.
//
// ⚠️ Measured in #205 and still true: an off-by-one on a raise's subject leaves
// the whole screen package green and both goldens green, because a raise's
// landing is the client's claim. So the subject is asserted over there
// (screen.TestEveryKeyThatReadsARowReadsTheFilteredOne) and the paragraph that
// arrives is asserted here.
//
// The listing is filtered first, because that is what makes the assertion
// discriminating: with nothing typed the visible rows and the book are the same
// list and a wrong read gives the right answer.
func TestTheDescriptionRaisedFromTheListingIsTheFilteredRow(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	filtered := filterSkillsTo(t, m, someSkillQuery)
	rows := filtered.skills.Rows()
	if len(rows) < 2 {
		t.Fatalf("%q leaves %d rows, so the cursor has nowhere to be wrong",
			someSkillQuery, len(rows))
	}
	want := rows[0]
	if want.ID == filtered.skills.Skills[0].ID {
		t.Fatalf("%q's first match is also the book's first skill (%s), so reading "+
			"either list gives the same answer", someSkillQuery, want.ID)
	}

	described := typeText(t, filtered, "?")
	if described.screen != screenBlurb {
		t.Fatalf("? left the screen at %v rather than raising the description", described.screen)
	}
	if described.raisedFrom != screenSkills {
		t.Errorf("the description records screen %v as its way back", described.raisedFrom)
	}
	body, _ := described.blurb.View(described.ctx())
	if named := described.lang.GlossedSkill(want); !strings.Contains(body, named) {
		t.Errorf("the description does not name %q:\n%s", named, body)
	}
	if wrong := described.lang.GlossedSkill(filtered.skills.Skills[0]); strings.Contains(body, wrong) {
		t.Errorf("the description names %q, which is the book's first skill and not the "+
			"filtered one:\n%s", wrong, body)
	}
	// And the position beside the name counts the narrowed list rather than the
	// book, so "1 / 5" does not read as "1 / 66".
	if position := described.text(i18n.ChoicePosition, 1, len(rows)); !strings.Contains(body, position) {
		t.Errorf("the description does not place the skill in the narrowed list (%q):\n%s",
			position, body)
	}
	// esc comes back to the listing the raise came from, which is what
	// model.raisedFrom is for.
	if back := key(t, described, "esc"); back.screen != screenSkills {
		t.Errorf("escaping the description went to screen %v", back.screen)
	}
}
