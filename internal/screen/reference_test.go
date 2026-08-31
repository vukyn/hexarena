package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The two references the menu opens: the elements listing and the species book.
// What each draws, and how its cursor walks — the client's half, which menu entry
// reaches them and where esc lands, stays in cmd/hexforge-tui.

// TestTheChartScreenDrawsTheEdgesUnderTheCursor is the answer the screen exists
// to give: which elements this one beats, which beat it, and at what rate.
func TestTheChartScreenDrawsTheEdgesUnderTheCursor(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	listing := ElementsScreen{Cursor: indexOfElement(t, element.Fire)}

	drawn, _ := listing.View(c)
	want := c.Lang.DescribeElement(element.Fire, c.Lib.Chart())
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
	if !strings.Contains(drawn, c.Text(i18n.BlurbElementCaveat)) {
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
		c, _ := start(t, lang)
		chart := c.Lib.Chart()
		if len(chart.Strengths(element.Neutral)) > 0 || len(chart.Weaknesses(element.Neutral)) > 0 {
			t.Fatal("neutral is no longer inert in the shipped chart, so this measures nothing")
		}
		listing := ElementsScreen{Cursor: indexOfElement(t, element.Neutral)}
		want := c.Lang.Say(i18n.BlurbElementInert, "100%")
		if drawn, _ := listing.View(c); !strings.Contains(drawn, want) {
			t.Errorf("%s: the inert element does not say %q:\n%s", lang, want, drawn)
		}
	}
}

// TestTheEnglishChartDrawsNoColumnOfBlanks is the column that does not apply.
//
// An element's name is a compiled gloss, so it is empty in English by
// construction. Padding the ids out to a column anyway drew eleven rows with
// trailing whitespace where a name would go, which reads as a book that has lost
// its names rather than as a language that has none.
func TestTheEnglishChartDrawsNoColumnOfBlanks(t *testing.T) {
	c, _ := start(t, i18n.En)
	if gloss := c.Lang.Gloss(element.Fire.String()); gloss != "" {
		t.Fatalf("English now glosses fire as %q, so there is a column to draw", gloss)
	}
	body, _ := ElementsScreen{}.View(c)
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimRight(line, " ") != line {
			t.Errorf("the English chart pads a row out to an empty column: %q", line)
		}
	}
}

// TestTheSpeciesScreenSaysWhatTheWordMeans is the half the skills listing could
// not give: its restriction column says "chủng loài dragon" and had nowhere to
// go, and the note that answers it reached nobody.
func TestTheSpeciesScreenSaysWhatTheWordMeans(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	kinds := speciesTo(t, NewSpeciesScreen(lib), "dragon")

	kind, known := lib.Species().Get("dragon")
	if !known || kind.Note == "" {
		t.Fatal("the shipped dragon carries no note, so this measures nothing")
	}
	drawn, _ := kinds.View(c)
	if !strings.Contains(drawn, firstWords(kind.Note)) {
		t.Errorf("the species screen drops the authored note:\n%s", drawn)
	}
}

// TestTheSpeciesScreenListsNoCastAndNoKit is the shape the screen deliberately
// does not have.
//
// Both cells it once drew were lists that grow with the books — who is one grows
// with the cast, the kit kept for a kind grows with the skills — and a column
// whose width is the size of another book stops fitting on the row that has the
// most to say. Written down as a test because the argument for adding either one
// back is a good one every time it is made, and the reason not to only shows up
// at a scale the shipped data does not have yet.
func TestTheSpeciesScreenListsNoCastAndNoKit(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		kinds := speciesTo(t, NewSpeciesScreen(lib), "dragon")
		drawn, _ := kinds.View(c)
		for _, character := range lib.Characters().OfSpecies("dragon") {
			if strings.Contains(drawn, character.ID) {
				t.Errorf("%s: the species screen names the dragon %q:\n%s",
					lang, character.ID, drawn)
			}
		}
		for _, declared := range lib.Skills().Skills() {
			if len(declared.Restrict.SpeciesNames()) == 0 {
				continue
			}
			if strings.Contains(drawn, declared.ID) {
				t.Errorf("%s: the species screen names the restricted skill %q:\n%s",
					lang, declared.ID, drawn)
			}
		}
	}
}

// TestAKindNobodyIsSaysSoInWords is the one fact about a species that is not on
// its own row, and the shipped cast never puts it on screen.
//
// Every kind in the book is claimed today, so this is only ever reached by a book
// that has an unclaimed one. It is the whole of what the members column was worth
// having for, and the half of it that does not grow with the cast.
func TestAKindNobodyIsSaysSoInWords(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		kinds := speciesTo(t, NewSpeciesScreen(lib), "dragon")
		if kinds.Claimed["dragon"] == 0 {
			t.Fatal("nobody is a dragon in the shipped cast, so clearing it measures nothing")
		}
		kinds = withNobodyClaiming(kinds)
		if drawn, _ := kinds.View(c); !strings.Contains(drawn, c.Text(i18n.SpeciesNobodyIs)) {
			t.Errorf("%s: an unclaimed kind does not say so:\n%s", lang, drawn)
		}
	}
}

// TestTheSpeciesNameIsVietnameseOnly is the leak the reference would otherwise
// be: a species' name is a data field rather than a compiled gloss, so it is
// Vietnamese whoever asks and an English screen has to drop the column.
func TestTheSpeciesNameIsVietnameseOnly(t *testing.T) {
	vi, lib := start(t, i18n.Vi)
	kinds := NewSpeciesScreen(lib)
	kind, known := lib.Species().Get("dragon")
	if !known || kind.Name == "" {
		t.Fatal("the shipped dragon has no authored name, so this measures nothing")
	}
	if drawn, _ := kinds.View(vi); !strings.Contains(drawn, kind.Name) {
		t.Errorf("the Vietnamese species screen drops the name %q:\n%s", kind.Name, drawn)
	}
	// The English listing carries the ids alone. Measured on the rows rather than
	// on the whole screen, because the note under them is authored prose printed
	// in both languages — it says "chất rồng", and that is the author's sentence
	// rather than a lookup the program performed. A note has no id to fall back
	// to, so dropping it in English would leave the pane blank.
	en := vi
	en.Lang = i18n.En
	drawn, _ := kinds.View(en)
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
//
// ⚠️ Measured against bodyRoom, which mirrors the client's frame rather than
// being it — see its comment. The client's screens.golden is what records these
// two at 120x24 with the frame really around them.
func TestBothReferencesFitTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		c = atTheFloor(c)

		body, _ := ElementsScreen{}.View(c)
		if rows := len(drawnLines(body)); rows > bodyRoom(c) {
			t.Errorf("%s: the chart takes %d rows of the %d a %dx%d window gives it:\n%s",
				lang, rows, bodyRoom(c), MinWidth, MinHeight, body)
		}

		kinds := speciesTo(t, NewSpeciesScreen(lib), longestNoteKind(t, NewSpeciesScreen(lib)))
		body, _ = kinds.View(c)
		if rows := len(drawnLines(body)); rows > bodyRoom(c) {
			t.Errorf("%s: the species reference takes %d rows of the %d a %dx%d window gives it:\n%s",
				lang, rows, bodyRoom(c), MinWidth, MinHeight, body)
		}
		// And the note itself survives whole, which is what makes the reserve worth
		// spending rather than a number that happens to be big enough: its last
		// line is the one the frame would cut first.
		selected := kinds.Kinds[kinds.Cursor]
		wrapped := WrapWords(selected.Note, MinWidth-3)
		if last := wrapped[len(wrapped)-1]; !strings.Contains(body, last) {
			t.Errorf("%s: the smallest window cuts the note short of %q:\n%s",
				lang, last, body)
		}
	}
}

// longestNoteKind is the kind whose note wraps to the most lines, which is the
// tallest the note pane ever draws.
func longestNoteKind(t *testing.T, s SpeciesScreen) string {
	t.Helper()
	found, most := "", 0
	for _, kind := range s.Kinds {
		if lines := len(WrapWords(kind.Note, MinWidth-3)); lines > most {
			found, most = kind.ID, lines
		}
	}
	if found == "" {
		t.Fatal("no shipped kind carries a note")
	}
	return found
}

// TestBothReferencesHoldTheirCursorInsideTheBook is the walk a reader takes: the
// cursor stops at both ends rather than running off a listing whose length is
// the data's rather than the program's.
func TestBothReferencesHoldTheirCursorInsideTheBook(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	chart := ElementsScreen{}
	for range len(element.All()) + 3 {
		chart, _ = chart.Update(c, press(t, "down"))
	}
	if chart.Cursor != len(element.All())-1 {
		t.Errorf("the chart cursor ran to %d of %d", chart.Cursor, len(element.All()))
	}
	for range len(element.All()) + 3 {
		chart, _ = chart.Update(c, press(t, "up"))
	}
	if chart.Cursor != 0 {
		t.Errorf("the chart cursor ran back to %d", chart.Cursor)
	}

	kinds := NewSpeciesScreen(lib)
	total := len(kinds.Kinds)
	if total == 0 {
		t.Fatal("the shipped book declares no species")
	}
	for range total + 3 {
		kinds, _ = kinds.Update(c, press(t, "down"))
	}
	if kinds.Cursor != total-1 {
		t.Errorf("the species cursor ran to %d of %d", kinds.Cursor, total)
	}
	for range total + 3 {
		kinds, _ = kinds.Update(c, press(t, "up"))
	}
	if kinds.Cursor != 0 {
		t.Errorf("the species cursor ran back to %d", kinds.Cursor)
	}
}

// speciesTo puts the species listing's cursor on a named kind.
func speciesTo(t *testing.T, s SpeciesScreen, id string) SpeciesScreen {
	t.Helper()
	for index, kind := range s.Kinds {
		if kind.ID == id {
			s.Cursor = index
			return s
		}
	}
	t.Fatalf("no species %q in the listing", id)
	return s
}

// withNobodyClaiming is the species reference as a book with an unclaimed kind
// draws it: the kind under the cursor counted down to nobody, which is what puts
// the "nobody is one" line on screen.
//
// A copy of the map rather than a write into it, because a screen is a value and
// a shared map would clear the count on every copy of it.
//
// It is the client's own fixture helper of the same name, copied rather than
// shared: `everyScreen` registers this state too, so the client needs its copy,
// and reaching across for one would tie two test suites together.
func withNobodyClaiming(s SpeciesScreen) SpeciesScreen {
	claimed := make(map[string]int, len(s.Claimed))
	for id, count := range s.Claimed {
		claimed[id] = count
	}
	if len(s.Kinds) > 0 {
		claimed[s.Kinds[Clamp(s.Cursor, 0, len(s.Kinds)-1)].ID] = 0
	}
	s.Claimed = claimed
	return s
}
