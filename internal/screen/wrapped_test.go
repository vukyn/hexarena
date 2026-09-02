package screen

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestAWrappedRowLeavesTheWindowsLastColumnEmpty holds the one thing every other
// row in these clients already promises: a line may spend UsableWidth() - 1 and
// no more, because a line filling a terminal's final cell wraps on some of them
// and one wrapped line pushes the footer off the bottom.
//
// ⚠️ **No width sweep in either client can see this row.** Both copies of
// TestEveryWordingFitsTheMinimumWidth skip a line carrying free text
// (carriesFreeText), and a wrapped row is exactly such a line — a biography, a
// kit of nine ids, a list of glossed names. So the rule the sweep holds over the
// catalog's wording had to be measured here, over the wrapped rows themselves,
// or WrappedIn is the one row-drawing helper in the package with nothing on it.
// Measured before the fix: browse's biography row came to exactly 120 cells at
// the 120 floor.
//
// **The value is swept rather than written down, and in two families, because
// the interesting length is the one that fills the row exactly.** Wrapping packs
// greedily, so a value of one-letter words reaches the largest odd length that
// fits and one led by a two-letter word reaches the largest even one — between
// them, whatever the room comes to, some line lands on it exactly. That is what
// makes the second assertion possible: the widest line emitted anywhere in the
// sweep must be UsableWidth() - 1, so this cannot pass by drawing everything
// comfortably short of the window. A test that only checked the ceiling would go
// on passing if the room were narrowed by ten.
//
// The narrow arm is the same claim about the other branch: a label column wide
// enough to leave under eight cells clips instead of wrapping, and the clip is
// measured off the corrected room, so it gives up the same final column.
//
// ⚠️ **The sweep's words are short on purpose, and that is a bound on what this
// holds rather than a convenience.** WrapWords gives a word longer than the room
// a line of its own and lets it overflow — an id is a name, and half a name is
// worse than a line the frame cuts — so a value holding one still reaches the
// last column and nothing here would be right to say otherwise. What the fix
// changed is the room, and a word that overflowed the wider room overflowed the
// narrower one already.
func TestAWrappedRowLeavesTheWindowsLastColumnEmpty(t *testing.T) {
	base, _ := start(t, i18n.Vi)
	// 80 is under the floor, so UsableWidth clamps it and the row is drawn for a
	// window this program refuses to draw in at all — which is the reading that
	// helper promises, and the one a row measured against c.Width would get wrong.
	for _, window := range []int{80, MinWidth, 160} {
		c := base
		c.Width = window
		usable := c.UsableWidth()
		label := c.Text(i18n.LabelBiography)
		width := c.DetailLabelWidth()
		widest := 0
		for _, lead := range []string{"", "ab "} {
			// Long enough that the last line is a remainder rather than the whole
			// value: every line before it is a line the packing filled.
			for words := 1; words <= usable; words++ {
				value := lead + strings.TrimSpace(strings.Repeat("a ", words))
				for _, line := range emittedRows(c.Wrapped(label, width, value)) {
					cells := lipgloss.Width(line)
					if cells >= usable {
						t.Fatalf("a wrapped row of %d words at a window of %d drew %d cells, "+
							"which fills the last column of the %d there are: %q",
							words, window, cells, usable, line)
					}
					widest = max(widest, cells)
				}
			}
		}
		if widest != usable-1 {
			t.Errorf("the widest wrapped row at a window of %d is %d cells, want %d — "+
				"the sweep never reached a value that fills the row, so it measures "+
				"nothing about where the row stops", window, widest, usable-1)
		}
		// The clip branch, reached by asking for a label column that leaves the
		// value under the eight cells wrapping needs.
		narrow := usable - 10
		clipped := emittedRows(c.Wrapped(label, narrow, strings.Repeat("a ", usable)))
		if len(clipped) != 1 {
			t.Fatalf("a clipped row drew %d lines at a window of %d, want one", len(clipped), window)
		}
		if cells := lipgloss.Width(clipped[0]); cells != usable-1 {
			t.Errorf("the clipped row at a window of %d drew %d cells, want %d",
				window, cells, usable-1)
		}
	}
}

// TestTheBiographyRowFitsTheWindowItIsDrawnIn is the same property at the call
// site the defect was measured on, over the cast rather than over a swept
// string: a biography is the longest free text this program draws and the row it
// draws it on is the one that filled the window.
//
// It walks both languages because the label column is measured from the wording
// in front (DetailLabelWidth), so the room a bio has is not the same number in
// the two of them.
func TestTheBiographyRowFitsTheWindowItIsDrawnIn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, lib := start(t, lang)
		measured := 0
		for _, window := range []int{MinWidth, 160} {
			c := base
			c.Width = window
			for _, character := range lib.Characters().All() {
				if character.Bio == "" {
					continue
				}
				measured++
				row := c.Wrapped(c.Text(i18n.LabelBiography), c.DetailLabelWidth(), character.Bio)
				for _, line := range emittedRows(row) {
					if cells := lipgloss.Width(line); cells >= c.UsableWidth() {
						t.Errorf("%s: %s's biography drew %d cells at a window of %d, "+
							"filling the last column of the %d there are",
							lang, character.ID, cells, window, c.UsableWidth())
					}
				}
			}
		}
		// A cast with no biography on it would pass every line above by drawing
		// none of them.
		if measured == 0 {
			t.Fatalf("%s: no character in the library carries a biography, so this measures nothing", lang)
		}
		t.Logf("%s: measured %d biography rows", lang, measured)
	}
}

// emittedRows is a drawn row as a frame would split it: the trailing newline
// every row helper here ends with is not a line of its own.
func emittedRows(drawn string) []string {
	return strings.Split(strings.TrimSuffix(drawn, "\n"), "\n")
}
