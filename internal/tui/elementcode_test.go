package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vukyn/hexarena/internal/core/element"
)

// TestEveryElementDrawsItsOwnCodeWithNoColourAtAll is the claim the whole column
// rests on, and it is the one a golden cannot make.
//
// ⚠️ **The abbreviation carries the meaning and the colour only reinforces it.**
// Every golden in this repository is recorded under NO_COLOR, where the palette
// is the identity and this column is eleven bare words — so if two elements drew
// the same three letters, the design record would hold a table that cannot be
// read and would hold it silently. A reader on a monochrome terminal, a reader
// through a screen reader and a reader of a recording that lost its escape codes
// are in exactly that position, and they are the readers the palette's own rule
// is written for.
//
// So the ink here is nil — which is what a plain palette renders as, since a
// lipgloss style with nothing set returns the string it was handed — and the
// eleven cells have to be eleven different things on that evidence alone.
//
// ⚠️ **Exhaustive over the enum rather than over a sample**, because the pair
// that clashes is the pair nobody picked: `grass` and `ground` agree on their
// first two letters and disagree on the third, so a two-letter scheme fails here
// and a three-letter one passes, and no sample that happened to miss one of them
// could tell the two schemes apart. element.All is the whole book.
func TestEveryElementDrawsItsOwnCodeWithNoColourAtAll(t *testing.T) {
	drawnBy := map[string]element.Element{}
	for _, member := range element.All() {
		single, err := element.Single(member)
		if err != nil {
			t.Fatalf("a single %s affinity is refused, so it cannot be drawn at all: %v",
				member, err)
		}
		cell := strings.TrimRight(elementCell(single, nil), " ")
		if strings.Contains(cell, "\x1b[") {
			t.Errorf("%s draws %q with no ink asked for, so this column writes escape "+
				"codes a plain terminal has to read as text", member, cell)
		}
		if already, taken := drawnBy[cell]; taken {
			t.Errorf("%s and %s both draw %q, so a reader with no colour cannot tell them "+
				"apart — and neither can any golden in this repository",
				already, member, cell)
			continue
		}
		drawnBy[cell] = member
	}
	// The premise, held rather than assumed: a walk that drew nothing would have
	// found no collision and passed.
	if got := len(drawnBy); got != element.Count {
		t.Fatalf("the walk drew %d distinct cells over %d elements", got, element.Count)
	}
}

// TestEveryElementCodeIsItsOwnIdShortened is why the codes may be the same in
// both languages, asserted rather than left to the comment that says so.
//
// A code is an id with its tail cut off, so `gra` is `grass` to a reader of
// either language and `wat` is `water`. That is the entire argument for not
// translating them — internal/i18n keeps element ids as they are in Vietnamese
// and in English, and a code derived from a *translation* would be a different
// string on each screen for the same row of the same data file.
//
// The length is asserted too, and both directions of it matter: shorter than the
// declared length would mean a code that is not the shortening the column was
// sized for, and longer would mean a cell past its column.
func TestEveryElementCodeIsItsOwnIdShortened(t *testing.T) {
	for _, member := range element.All() {
		id, code := member.String(), ElementCode(member)
		if !strings.HasPrefix(id, code) {
			t.Errorf("%s draws the code %q, which is not the front of its own id %q — a "+
				"reader cannot read it back to the data file", member, code, id)
		}
		if got := utf8.RuneCountInString(code); got != elementCodeLength {
			t.Errorf("%s draws the %d-letter code %q against a scheme of %d",
				member, got, code, elementCodeLength)
		}
	}
	// ⚠️ The shortening has to be a real one, or every claim above is about the
	// names themselves: `ice` is exactly the code length, so at least one element
	// must be longer than its code for this to be measuring an abbreviation.
	shortened := 0
	for _, member := range element.All() {
		if utf8.RuneCountInString(member.String()) > elementCodeLength {
			shortened++
		}
	}
	if shortened == 0 {
		t.Fatal("no element's id is longer than a code, so nothing here was abbreviated")
	}
}

// TestADualElementUnitDrawsBothOfItsCodes is the half a single-element walk
// cannot see.
//
// A column sized and filled for one element draws every single affinity
// perfectly and joins two of them wrongly — which is the case that misaligns the
// two stat columns beside it, on one row, in a table where every other row is
// straight. So this walks every pair element.Dual admits and asks for three
// things a joined cell has to have: both codes, in the order the affinity
// declares them, with the separator between.
//
// ⚠️ It is not compared against a second copy of elementCell's own arithmetic —
// a copy agrees with a broken original. What it compares against is the codes and
// the separator, assembled here.
func TestADualElementUnitDrawsBothOfItsCodes(t *testing.T) {
	pairs := 0
	for _, primary := range element.All() {
		for _, secondary := range element.All() {
			affinity, err := element.Dual(primary, secondary)
			if err != nil {
				// The pairs no unit can hold: the same element twice, and
				// anything paired with the inert one.
				continue
			}
			pairs++
			cell := strings.TrimRight(elementCell(affinity, nil), " ")
			want := ElementCode(primary) + elementCodeJoiner + ElementCode(secondary)
			if cell != want {
				t.Errorf("%s draws %q, want %q", affinity, cell, want)
			}
			// And the order is the affinity's rather than the enum's: a cell
			// that sorted its halves would match half the pairs above and read
			// as a different affinity on the other half.
			if first, second := ElementCode(primary), ElementCode(secondary); first != second &&
				strings.Index(cell, first) > strings.Index(cell, second) {
				t.Errorf("%s draws %q with its secondary first", affinity, cell)
			}
		}
	}
	if pairs == 0 {
		t.Fatal("element.Dual admitted no pair at all, so this measured nothing")
	}
}

// TestEachHalfOfADualIsInkedForItsOwnElement is the claim a palette makes and
// this package has to make possible.
//
// A cell inked once, in the primary's colour, would draw `gra/ele` entirely
// green — which says something untrue about the second half, and says it in the
// one place the colour is about the data rather than about the layout. So the
// ink is offered each element separately, and what is asserted is that it was
// offered the right one with the right code.
//
// ⚠️ **It is held here rather than in internal/screen** because that package's
// fixture cast carries no dual affinity at any squad size — measured — so a test
// over a fielded bench there would have been a walk over singles reporting
// success. Here an affinity is built rather than fielded, and every pair
// element.Dual admits is walked.
func TestEachHalfOfADualIsInkedForItsOwnElement(t *testing.T) {
	pairs := 0
	for _, primary := range element.All() {
		for _, secondary := range element.All() {
			affinity, err := element.Dual(primary, secondary)
			if err != nil {
				continue
			}
			pairs++
			var offered []element.Element
			var codes []string
			elementCell(affinity, func(member element.Element, code string) string {
				offered = append(offered, member)
				codes = append(codes, code)
				return code
			})
			if len(offered) != 2 {
				t.Errorf("%s offered the ink %d elements, want both halves", affinity, len(offered))
				continue
			}
			if offered[0] != primary || offered[1] != secondary {
				t.Errorf("%s offered the ink %v and %v", affinity, offered[0], offered[1])
			}
			for index, member := range offered {
				if want := ElementCode(member); codes[index] != want {
					t.Errorf("%s offered the ink %q for %v, want %q",
						affinity, codes[index], member, want)
				}
			}
		}
	}
	if pairs == 0 {
		t.Fatal("element.Dual admitted no pair at all, so this measured nothing")
	}
}

// TestAnInkedElementCellIsTheSameWidthAsAPlainOne is the arrangement the column
// stands on, and it is invisible to every golden.
//
// Go's `%-8s` pads by counting runes. An inked cell carries escape sequences,
// which are runes a terminal never draws, so a coloured cell handed to the format
// verb would be counted as far past its column and padded by nothing — and every
// row carrying one would sit a few cells left of every row that did not. That is
// why elementCell pads itself off the codes rather than off what the ink
// returned.
//
// ⚠️ **Nothing recorded can see this.** The goldens are taken with the palette
// plain, so they hold the uninked cell; the bug lives only in the drawing a
// reader with colour gets. The ink here is therefore a real one — a marker on
// each side of the code — and the measurement is of the *plain* content, which
// is what a terminal actually puts on the row.
func TestAnInkedElementCellIsTheSameWidthAsAPlainOne(t *testing.T) {
	const opening, closing = "\x1b[31m", "\x1b[0m"
	ink := func(_ element.Element, code string) string { return opening + code + closing }
	for _, primary := range element.All() {
		for _, secondary := range element.All() {
			affinity, err := element.Dual(primary, secondary)
			if err != nil {
				single, singleErr := element.Single(primary)
				if singleErr != nil {
					continue
				}
				affinity = single
			}
			plain := elementCell(affinity, nil)
			inked := elementCell(affinity, ink)
			// The ink really landed, or the comparison below is the plain cell
			// against itself.
			if !strings.Contains(inked, opening) {
				t.Fatalf("%s came back from an inking ink as %q, so nothing here is "+
					"measuring an inked cell", affinity, inked)
			}
			bare := strings.ReplaceAll(strings.ReplaceAll(inked, opening, ""), closing, "")
			if bare != plain {
				t.Errorf("%s draws %q inked, which is %q once the escapes are taken out, "+
					"against %q plain", affinity, inked, bare, plain)
			}
			if got := utf8.RuneCountInString(bare); got != rosterElementCell {
				t.Errorf("%s draws %d visible cells inked against a column of %d",
					affinity, got, rosterElementCell)
			}
		}
	}
}
