package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The species picker's detail cell, and the one rule it had been going round:
// the word beside a species id is authored in species.json in Vietnamese only,
// so Lang.SpeciesName answers nothing in English by design and an English reader
// gets the id, which in English is the name. The picker read kind.Name straight
// off the book instead and drew "dragon  rồng" on an English screen — a leak
// rather than a translation, and the shape the species *listing* over the same
// book already refuses.
//
// Every assertion below is on the text m.screenContent() actually draws, and the
// words are looked up in the book rather than written down here: a test that
// spelled "rồng" would pass over a book that had renamed it.

// aSpeciesPicker is the character form's species picker, opened the way a
// keystroke opens it — the form, then the field — rather than by building a
// pickState, so what is measured is the screen the program reaches.
func aSpeciesPicker(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = m.enter(screenNew).openSpecies()
	if m.picker == nil {
		t.Fatal("the species field raised no picker")
	}
	if m.picker.kind != pickSpecies {
		t.Fatalf("the species field raised a %d picker", m.picker.kind)
	}
	if len(m.picker.visible()) < 2 {
		t.Fatalf("the species picker lists %d rows, too few to say anything",
			len(m.picker.visible()))
	}
	return m
}

// speciesRowFront is the marker and state cells the picker writes in front of
// the row at an index. A fresh character form has answered no species, so no row
// carries a position in the answer and none can be refused — which leaves the
// two fronts picker_column_test.go already names.
func speciesRowFront(t *testing.T, m model, index int) string {
	t.Helper()
	if len(m.picker.chosen) != 0 {
		t.Fatalf("the fresh form has already chosen %v, so a row's front is not one of the two known ones",
			m.picker.chosen)
	}
	if index == m.picker.cursor {
		return selectedRowCells
	}
	return plainRowCells
}

// An English row is the bare id: the species book's Vietnamese word appears
// nowhere on the screen, and every row is its id with nothing after it.
func TestTheSpeciesPickerDrawsNoVietnameseNameInEnglish(t *testing.T) {
	m := aSpeciesPicker(t, i18n.En)
	drawn := m.screenContent()

	named := 0
	for index, option := range m.picker.visible() {
		kind, known := m.lib.Species().Get(option.id)
		if !known {
			t.Fatalf("the picker offers %s, which the species book does not know", option.id)
		}
		want := speciesRowFront(t, m, index) + option.id
		if !strings.Contains(drawn, "\n"+want+"\n") {
			t.Errorf("the English species picker does not draw %q as its own whole line:\n%s",
				want, drawn)
		}
		// And the name the row used to carry is on no line of the screen at all,
		// which is the claim the whole-line one above cannot make: a name drawn
		// somewhere other than this row would still be a gloss in English.
		if name := strings.TrimSpace(kind.Name); name != "" {
			named++
			if strings.Contains(drawn, name) {
				t.Errorf("the English species picker holds %s's authored name %q:\n%s",
					option.id, name, drawn)
			}
		}
	}
	// The shipped book names every kind, so a book that named none would make
	// every claim above vacuously true.
	if named == 0 {
		t.Error("no species in the book carries an authored name, so this screen proves nothing")
	}
}

// The same picker in Vietnamese still draws the authored name beside each id, so
// the fix is the accessor doing its job rather than the cell being emptied.
func TestTheSpeciesPickerKeepsTheAuthoredNameInVietnamese(t *testing.T) {
	m := aSpeciesPicker(t, i18n.Vi)
	drawn := m.screenContent()

	for index, option := range m.picker.visible() {
		kind, known := m.lib.Species().Get(option.id)
		if !known {
			t.Fatalf("the picker offers %s, which the species book does not know", option.id)
		}
		name := m.lang.SpeciesName(kind)
		if name == "" {
			t.Fatalf("the species %s carries no Vietnamese name, so this row has no column to keep",
				option.id)
		}
		want := speciesRowFront(t, m, index) + pad(option.id, m.picker.idColumn()) + " " + name
		if !strings.Contains(drawn, "\n"+want+"\n") {
			t.Errorf("the Vietnamese species picker does not draw %q as its own whole line:\n%s",
				want, drawn)
		}
	}
}

// With nothing in any English cell the whole column goes, which is the behaviour
// #164 built and this screen reaches for free — asserted rather than assumed,
// since a list where every detail is empty is exactly the case a per-row collapse
// would also pass.
func TestTheEnglishSpeciesPickerDropsItsDetailColumn(t *testing.T) {
	m := aSpeciesPicker(t, i18n.En)
	rows := m.picker.visible()
	if column := m.picker.detailColumn(m, rows); column != 0 {
		t.Errorf("the English species picker keeps a detail column of %d", column)
	}
	drawn := m.screenContent()
	for index, option := range rows {
		// No padding: the id is not widened to the column the ids would share.
		padded := speciesRowFront(t, m, index) + pad(option.id, m.picker.idColumn())
		if m.picker.idColumn() > len(option.id) && strings.Contains(drawn, padded) {
			t.Errorf("the English species picker still pads %q out to a column of %d:\n%s",
				option.id, m.picker.idColumn(), drawn)
		}
		// And no separator: the line stops at the id.
		front := speciesRowFront(t, m, index)
		if strings.Contains(drawn, "\n"+front+option.id+" ") {
			t.Errorf("the English species picker draws a separator after %q:\n%s",
				option.id, drawn)
		}
	}
	// The Vietnamese picker over the same book keeps its column, so the collapse
	// is a reading of the rows rather than a check on the language.
	vi := aSpeciesPicker(t, i18n.Vi)
	if column := vi.picker.detailColumn(vi, vi.picker.visible()); column == 0 {
		t.Error("the Vietnamese species picker dropped its detail column too")
	}
}

// The species picker is one of the screens the width and translation sweeps
// walk. It was not, which is why the leak above lived: those sweeps iterate
// everyScreen, and everyScreen registered no species picker at all.
//
// ⚠️ Membership by name would prove nothing — an entry called "species picker"
// holding some other model would satisfy it while the sweeps still never
// rendered this screen. So the entry is found by what its model *is* (a picker
// over the species book) and then held to what it *draws*: byte for byte the
// screen an independently opened species picker renders.
func TestTheScreenSweepWalksTheSpeciesPicker(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		want := m.enter(screenNew).openSpecies()
		want.width, want.height = 200, 60

		found := ""
		for name, screen := range everyScreen(t, m) {
			if screen.picker == nil || screen.picker.kind != pickSpecies {
				continue
			}
			screen.width, screen.height = want.width, want.height
			if screen.screenContent() != want.screenContent() {
				t.Errorf("the %s entry in %s holds a species picker that draws a different screen:\n%s",
					name, lang, screen.screenContent())
				continue
			}
			found = name
		}
		if found == "" {
			t.Errorf("no entry of everyScreen renders the species picker in %s, "+
				"so no width or translation test measures it", lang)
		}
	}
}
