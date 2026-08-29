package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The picker's detail column, and the one thing it has to be able to do: go
// away.
//
// A trait's name is authored in passives.json in Vietnamese only, so
// Lang.PassiveName answers nothing in English by design — an English reader gets
// the id, which in English is the name. The picker drew that as a padded id and
// a blank cell after it, and on the cursor row the selection bar was painted
// across the whole empty width. The trait listing already drops its gloss column
// outright there, which is the house rule a table follows, and these hold the
// picker to it.
//
// Every assertion below is on the text m.screenContent() actually draws, not on
// a field: a column is a thing on a screen, and a helper answering "is there a
// column" could agree with itself while the rows disagreed with both.
const (
	// A row's marker and state cells as the picker writes them, for a row
	// carrying neither a position in the answer nor a refusal: two cells of
	// marker, then the five `%2s %s ` leaves blank.
	plainRowCells = "       "
	// The same row with the cursor on it. The selection is a style — escape
	// codes — and every test here runs under NO_COLOR, so what is asserted is
	// the *cell* the bar covers rather than the bar.
	selectedRowCells = ">      "
)

// aTraitPicker is the squad builder's trait picker, open over a member that
// actually learns traits.
//
// The character is looked up rather than named, which is the rule the fixture
// exists to keep: the fixture cast's first character declares no traits at all,
// so a picker opened over it would have no rows and every claim here would pass
// over an empty list.
func aTraitPicker(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	m, _, _ := start(t, lang)
	m = m.enter(screenSquads)
	m.squad = someSquad(t, m)
	holder, holds := aTraitHolder(m)
	if !holds {
		t.Skip("nothing in the fixture book learns a trait")
	}
	m = holder.openSquadPassives()
	if m.picker == nil {
		t.Fatal("the trait field raised no picker")
	}
	if len(m.picker.visible()) < 2 {
		t.Fatalf("the trait picker lists %d rows, too few to have an unchosen one",
			len(m.picker.visible()))
	}
	return m
}

// plainRowID is the first row of the picker in hand carrying neither a position
// in the answer nor the cursor, which is the row whose state cell is blank — so
// what precedes its id on the line is exactly plainRowCells.
func plainRowID(t *testing.T, m model) string {
	t.Helper()
	for index, option := range m.picker.visible() {
		if index == m.picker.cursor || slices.Contains(m.picker.chosen, option.id) {
			continue
		}
		if option.refusal != nil {
			continue
		}
		return option.id
	}
	t.Fatal("every row of the picker is chosen, refused or under the cursor")
	return ""
}

// lineStartingWith is the one drawn line beginning with a prefix, and a failure
// when there is none. It is how a claim about the *rest* of a row is made
// without restating how the front of the row is built.
func lineStartingWith(t *testing.T, drawn, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(drawn, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("no line of the screen starts with %q:\n%s", prefix, drawn)
	return ""
}

// An English trait row is the bare id and nothing else: no padding out to the
// widest id, and no separator after it.
func TestTheTraitPickerDrawsNoDetailColumnInEnglish(t *testing.T) {
	m := aTraitPicker(t, i18n.En)
	id := plainRowID(t, m)
	drawn := m.screenContent()

	want := plainRowCells + id
	if !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the English trait picker does not draw %q as its own whole line:\n%s",
			want, drawn)
	}
	// And the shape it used to draw is gone. The exact line above already says
	// so; this names the defect, so a failure reads as "the column came back"
	// rather than as a line that moved.
	if padded := pad(id, m.picker.idColumn()); strings.Contains(drawn, padded) {
		t.Errorf("the English trait picker still pads %q out to a column of %d:\n%s",
			id, m.picker.idColumn(), drawn)
	}
}

// The same picker in Vietnamese keeps the column, so the collapse is
// language-blind in effect without being a language check in code.
func TestTheTraitPickerKeepsItsDetailColumnInVietnamese(t *testing.T) {
	m := aTraitPicker(t, i18n.Vi)
	id := plainRowID(t, m)
	held, err := m.lib.Passives().Lookup(id)
	if err != nil {
		t.Fatalf("look %s up in the trait book: %v", id, err)
	}
	name := m.lang.PassiveName(held)
	if name == "" {
		t.Fatalf("the trait %s carries no Vietnamese name, so this row has no column to keep", id)
	}

	want := plainRowCells + pad(id, m.picker.idColumn()) + " " + name
	if drawn := m.screenContent(); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the Vietnamese trait picker does not draw %q as its own whole line:\n%s",
			want, drawn)
	}
}

// A picker whose rows all carry a detail keeps its column in both languages. The
// kit's is forge.WhoMaySummary, which answers in both — an unrestricted skill
// says "anybody" rather than nothing — so this is the case the collapse must not
// reach.
func TestTheKitPickerKeepsItsDetailColumnInBothLanguages(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = m.enter(screenNew).openKit()
		if m.picker == nil || len(m.picker.visible()) < 2 {
			t.Fatalf("the kit field raised no picker with rows in %s", lang)
		}
		id := plainRowID(t, m)
		// The whole row is not asserted, because a summary long enough to be
		// clipped would make the expected text a second copy of clip. What the
		// column *is* — the id padded, then the separator, then something — is
		// the claim, and it is the claim the collapse would break.
		prefix := plainRowCells + pad(id, m.picker.idColumn()) + " "
		line := lineStartingWith(t, m.screenContent(), prefix)
		if len(line) <= len(prefix) {
			t.Errorf("the kit picker's row for %s in %s carries an empty detail cell: %q",
				id, lang, line)
		}
	}
}

// The collapse is all or nothing. A list where some rows have a detail and some
// do not keeps its column on every row, empty cells included — a column dropped
// per row is a ragged table rather than no table.
func TestAPickerWithSomeDetailsKeepsTheColumnOnEveryRow(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	const (
		named = "fire"
		bare  = "khong-co-ten"
	)
	if m.lang.Gloss(named) == "" {
		t.Fatalf("%s has no Vietnamese gloss, so this list has no detail at all", named)
	}
	if m.lang.Gloss(bare) != "" {
		t.Fatalf("%s glosses to %q, so this list has no empty row", bare, m.lang.Gloss(bare))
	}
	m = m.pick(&pickState{
		title: i18n.PickerElementsTitle, hint: i18n.PickerAllowlistHint,
		kind:    pickElements,
		options: []pickOption{{id: named}, {id: bare}},
	})

	// The empty row keeps its padding and its separator, so the two rows' detail
	// cells start in the same column.
	want := plainRowCells + pad(bare, m.picker.idColumn()) + " "
	if drawn := m.screenContent(); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the row for %s lost the column the row above it keeps; wanted %q:\n%s",
			bare, want, drawn)
	}
}

// A selected row with no column after it: the bar covers the bare id, which is
// what the species and trait listings already draw when they drop theirs.
// Padding is what gives a following cell a column to start at, and with no cell
// after it there is nothing left for the width to be for.
func TestTheSelectedRowOfAColumnlessPickerIsTheBareID(t *testing.T) {
	m := aTraitPicker(t, i18n.En)
	id := plainRowID(t, m)
	at := slices.IndexFunc(m.picker.visible(), func(option pickOption) bool {
		return option.id == id
	})
	m.picker.cursor = at

	want := selectedRowCells + id
	if drawn := m.screenContent(); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the selected row of the English trait picker is not %q:\n%s", want, drawn)
	}
}
