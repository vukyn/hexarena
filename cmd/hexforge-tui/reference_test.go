package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

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

// TestTheChartScreenDrawsTheEdgesUnderTheCursor is the answer the screen exists
// to give: which elements this one beats, which beat it, and at what rate.
func TestTheChartScreenDrawsTheEdgesUnderTheCursor(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenElements)
	m.elements.cursor = indexOfElement(t, element.Fire)

	drawn := m.screenContent()
	want := m.lang.DescribeElement(element.Fire, m.lib.Chart())
	if want == "" {
		t.Fatal("the shipped chart says nothing about fire, so this measures nothing")
	}
	for _, line := range strings.Split(want, "\n") {
		if !strings.Contains(drawn, line) {
			t.Errorf("the chart screen never draws %q:\n%s", line, drawn)
		}
	}
	// The listing above it names every element, so a reader can walk to the one
	// they were reading about in another screen's column.
	for _, member := range element.All() {
		if !strings.Contains(drawn, member.String()) {
			t.Errorf("the chart screen never lists %q:\n%s", member, drawn)
		}
	}
	// And the caveat, which is the one fact none of the rows carry: a dual unit
	// multiplies both of its halves.
	if !strings.Contains(drawn, m.text(i18n.BlurbElementCaveat)) {
		t.Errorf("the chart screen drops the dual-affinity caveat:\n%s", drawn)
	}
}

// TestTheInertElementSaysSoRatherThanDrawingTwoBlanks is the row the chart has
// nothing to say about.
//
// neutral is in no cycle and in no mutual pair, so both of its lists are empty.
// Two sentences reading "beats nothing" and "nothing beats it" would read as an
// entry that failed to load; being inert is one fact, and it is the whole of what
// neutral is for.
func TestTheInertElementSaysSoRatherThanDrawingTwoBlanks(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		chart := m.lib.Chart()
		if len(chart.Strengths(element.Neutral)) > 0 || len(chart.Weaknesses(element.Neutral)) > 0 {
			t.Fatal("neutral is no longer inert in the shipped chart, so this measures nothing")
		}
		m = m.enter(screenElements)
		m.elements.cursor = indexOfElement(t, element.Neutral)
		want := m.lang.Say(i18n.BlurbElementInert, "100%")
		if drawn := m.screenContent(); !strings.Contains(drawn, want) {
			t.Errorf("%s: the inert element does not say %q:\n%s", lang, want, drawn)
		}
	}
}

// TestTheSpeciesScreenSaysWhatBeingOneUnlocks is the half the skills listing
// could not give: its restriction column says "chủng loài dragon" and had
// nowhere to go.
func TestTheSpeciesScreenSaysWhatBeingOneUnlocks(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSpecies)
	m = speciesTo(t, m, "dragon")

	kept := m.lib.SkillsForSpecies("dragon")
	if len(kept) == 0 {
		t.Fatal("no shipped skill is kept for a dragon, so this measures nothing")
	}
	drawn := m.screenContent()
	if want := m.text(i18n.SpeciesKeptSkills, strings.Join(kept, " ")); !strings.Contains(drawn, want) {
		t.Errorf("the species screen never says %q:\n%s", want, drawn)
	}
	// The note beside the id, which is the only prose a species carries and was
	// reaching nobody before this screen existed.
	kind, known := m.lib.Species().Get("dragon")
	if !known {
		t.Fatal("the shipped book has lost the dragon")
	}
	if !strings.Contains(drawn, firstWords(kind.Note)) {
		t.Errorf("the species screen drops the authored note:\n%s", drawn)
	}
	// And who is one, which is the column that earns its place: a kind nobody is
	// is a gate that cannot open.
	for _, character := range m.lib.Characters().OfSpecies("dragon") {
		if !strings.Contains(drawn, character.ID) {
			t.Errorf("the species screen never names %q, which is a dragon:\n%s",
				character.ID, drawn)
		}
	}
}

// TestAKindNobodyIsSaysSoInWords is the empty cell the shipped cast never draws.
//
// Every kind in the book is claimed today, so the row is only ever reached by a
// book that has one — and an empty last cell reads as a column that failed to
// fill rather than as the fact that nothing is one.
func TestAKindNobodyIsSaysSoInWords(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = m.enter(screenSpecies)
		m = speciesTo(t, m, "dragon")
		if m.species.members["dragon"] == "" {
			t.Fatal("nobody is a dragon in the shipped cast, so emptying it measures nothing")
		}
		m.species = withNobodyClaiming(m.species)
		if drawn := m.screenContent(); !strings.Contains(drawn, m.text(i18n.SpeciesNobodyIs)) {
			t.Errorf("%s: an unclaimed kind does not say so:\n%s", lang, drawn)
		}
	}
}

// TestTheSpeciesNameIsVietnameseOnly is the leak the reference would otherwise
// be: a species' name is a data field rather than a compiled gloss, so it is
// Vietnamese whoever asks and an English screen has to drop the column.
func TestTheSpeciesNameIsVietnameseOnly(t *testing.T) {
	vi, _, _ := start(t, i18n.Vi)
	vi = vi.enter(screenSpecies)
	kind, known := vi.lib.Species().Get("dragon")
	if !known || kind.Name == "" {
		t.Fatal("the shipped dragon has no authored name, so this measures nothing")
	}
	if drawn := vi.screenContent(); !strings.Contains(drawn, kind.Name) {
		t.Errorf("the Vietnamese species screen drops the name %q:\n%s", kind.Name, drawn)
	}
	// The English listing carries the ids alone. Measured on the rows rather than
	// on the whole screen, because the note under them is authored prose printed
	// in both languages — it says "chất rồng", and that is the author's sentence
	// rather than a lookup the program performed. A note has no id to fall back
	// to, so dropping it in English would leave the pane blank.
	en, _, _ := start(t, i18n.En)
	en = en.enter(screenSpecies)
	drawn := en.screenContent()
	for _, line := range strings.Split(drawn, "\n") {
		if !strings.Contains(line, kind.ID) {
			continue
		}
		if strings.Contains(line, kind.Name) {
			t.Errorf("the English species row holds the Vietnamese name %q: %q", kind.Name, line)
		}
	}
	if !strings.Contains(drawn, firstWords(kind.Note)) {
		t.Errorf("the English species screen drops the authored note:\n%s", drawn)
	}
}

// TestBothReferencesFitTheSmallestWindow is what the reserved room under each
// listing is for, measured on the kind whose note is longest.
//
// A species' note is authored prose of no fixed length, so it wraps — and the
// frame cuts from the bottom, so a listing that reserved one line for a note
// taking three would lose the skills line underneath it. Losing the *derived*
// half of a pane to the length of its *authored* half is the failure this holds
// shut, and it would land on exactly the kinds somebody wrote most about.
func TestBothReferencesFitTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		chart := m.enter(screenElements)
		if drawn := chart.screenContent(); strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: the chart is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}

		kinds := m.enter(screenSpecies)
		kinds = speciesTo(t, kinds, longestNoteKind(t, kinds))
		drawn := kinds.screenContent()
		if strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: the species reference is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		// And the line the overrun would have eaten is really on screen, which is
		// what makes the reserve worth spending rather than a number that happens
		// to be big enough.
		selected := kinds.species.kinds[kinds.species.cursor]
		if kept := kinds.lib.SkillsForSpecies(selected.ID); len(kept) > 0 {
			want := kinds.text(i18n.SpeciesKeptSkills, strings.Join(kept, " "))
			if !strings.Contains(drawn, want) {
				t.Errorf("%s: the smallest window drops %q:\n%s", lang, want, drawn)
			}
		}
	}
}

// longestNoteKind is the kind whose note wraps to the most lines, which is the
// tallest the note pane ever draws.
func longestNoteKind(t *testing.T, m model) string {
	t.Helper()
	found, most := "", 0
	for _, kind := range m.species.kinds {
		if lines := len(wrapWords(kind.Note, minWidth-3)); lines > most {
			found, most = kind.ID, lines
		}
	}
	if found == "" {
		t.Fatal("no shipped kind carries a note")
	}
	return found
}

// TestTheEnglishChartDrawsNoColumnOfBlanks is the column that does not apply.
//
// An element's name is a compiled gloss, so it is empty in English by
// construction. Padding the ids out to a column anyway drew eleven rows with
// trailing whitespace where a name would go, which reads as a book that has lost
// its names rather than as a language that has none.
func TestTheEnglishChartDrawsNoColumnOfBlanks(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	if gloss := m.lang.Gloss(element.Fire.String()); gloss != "" {
		t.Fatalf("English now glosses fire as %q, so there is a column to draw", gloss)
	}
	body, _ := m.enter(screenElements).elements.view(m)
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimRight(line, " ") != line {
			t.Errorf("the English chart pads a row out to an empty column: %q", line)
		}
	}
}

// TestBothReferencesHoldTheirCursorInsideTheBook is the walk a reader takes: the
// cursor stops at both ends rather than running off a listing whose length is
// the data's rather than the program's.
func TestBothReferencesHoldTheirCursorInsideTheBook(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	chart := m.enter(screenElements)
	for range len(element.All()) + 3 {
		chart = key(t, chart, "down")
	}
	if chart.elements.cursor != len(element.All())-1 {
		t.Errorf("the chart cursor ran to %d of %d", chart.elements.cursor, len(element.All()))
	}
	for range len(element.All()) + 3 {
		chart = key(t, chart, "up")
	}
	if chart.elements.cursor != 0 {
		t.Errorf("the chart cursor ran back to %d", chart.elements.cursor)
	}

	kinds := m.enter(screenSpecies)
	total := len(kinds.species.kinds)
	if total == 0 {
		t.Fatal("the shipped book declares no species")
	}
	for range total + 3 {
		kinds = key(t, kinds, "down")
	}
	if kinds.species.cursor != total-1 {
		t.Errorf("the species cursor ran to %d of %d", kinds.species.cursor, total)
	}
	for range total + 3 {
		kinds = key(t, kinds, "up")
	}
	if kinds.species.cursor != 0 {
		t.Errorf("the species cursor ran back to %d", kinds.species.cursor)
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

// speciesTo puts the species listing's cursor on a named kind.
func speciesTo(t *testing.T, m model, id string) model {
	t.Helper()
	for index, kind := range m.species.kinds {
		if kind.ID == id {
			m.species.cursor = index
			return m
		}
	}
	t.Fatalf("no species %q in the listing", id)
	return m
}
