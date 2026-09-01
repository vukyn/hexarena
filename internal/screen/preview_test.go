package screen

import (
	"image/color"
	"strings"
	"testing"
)

// The art preview's own drawing, measured at the pixel rather than through a
// screen: these three functions take colours and hand back cells, so nothing
// here needs a library, a Context or a client.
//
// ⚠️ **This is the whole of what any suite says about the drawing.** There is no
// golden entry for the preview and it is not registered in either client's
// screen sweep, deliberately — see TODO.md § Not done: it draws art, so an entry
// would be measuring a rasterisation rather than a sentence, and what such an
// entry should assert has not been decided. So a change to cellRows, to the
// ramp's alphabet or to the luminance weights is caught by this file or by
// nothing.

// TestATransparentCellIsLeftAlone is the one property of the drawing that is not
// about how it looks: a cell with nothing in either half must be a plain space
// with no styling at all.
//
// Anything else paints the terminal's own background over with whatever colour
// this program guessed it to be, which turns a transparent margin into a
// rectangle — and the two new pictures in the shipped cast have transparent
// margins, so the wrong answer here would be visible on the first look.
func TestATransparentCellIsLeftAlone(t *testing.T) {
	clear := color.RGBA{}
	// Just under the floor: a pixel this faint is coverage from anti-aliasing
	// rather than ink, and drawing it would thicken every edge in the picture.
	faint := color.RGBA{R: 10, G: 10, B: 10, A: alphaFloor - 1}
	solid := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	drawn := NewPalette(false)

	for _, test := range []struct {
		name       string
		top, below color.RGBA
	}{
		{"both empty", clear, clear},
		{"both under the floor", faint, faint},
		{"one empty, one under the floor", clear, faint},
	} {
		if got := blockCell(ink(test.top), ink(test.below), drawn); got != " " {
			t.Errorf("%s rendered as %q in colour, want a bare space", test.name, got)
		}
		if got := rampCell(ink(test.top), ink(test.below)); got != " " {
			t.Errorf("%s rendered as %q in monochrome, want a bare space", test.name, got)
		}
	}

	// And a painted cell is never a space, in either drawing: a pixel that reads
	// as nothing turns a filled shape into a hole.
	for _, test := range []struct {
		name       string
		top, below color.RGBA
	}{
		{"the top half", solid, clear},
		{"the bottom half", clear, solid},
		{"both halves", solid, solid},
	} {
		if got := blockCell(ink(test.top), ink(test.below), drawn); strings.TrimSpace(got) == "" {
			t.Errorf("%s rendered as %q in colour, want ink", test.name, got)
		}
		if got := rampCell(ink(test.top), ink(test.below)); got == " " {
			t.Errorf("%s rendered as a space in monochrome, want ink", test.name)
		}
	}

	// A fully white pixel is the case that would read as nothing on the ramp, so
	// it is the one worth naming: white ink is still ink.
	white := ink(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if got := rampCell(white, white); got == " " {
		t.Error("a white pixel drew as a space, so a pale shape would come out hollow")
	}
}

// TestThePaletteRemembersWhichTerminalItWasBuiltFor is the one wire the preview
// hangs on that nothing else in this package reads.
//
// The monochrome path is a *different drawing* rather than the coloured one with
// its escape codes stripped, so the preview cannot find out by rendering through
// a style and noticing nothing happened — it asks Palette.Plain. This package
// reads no environment, so that bool is only ever the one the binary handed to
// NewPalette, and a palette that forgot it would draw half blocks with no colour
// in them: a silhouette in one character, on a terminal that asked for a ramp.
func TestThePaletteRemembersWhichTerminalItWasBuiltFor(t *testing.T) {
	for _, plain := range []bool{true, false} {
		if got := NewPalette(plain).Plain; got != plain {
			t.Errorf("NewPalette(%v) remembers Plain = %v", plain, got)
		}
	}
	// And the two drawings really do differ, or remembering it would buy
	// nothing: the same pixel is a ramp character in one and a half block in the
	// other.
	pixel := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	monochrome := rampCell(ink(pixel), inkColour{})
	coloured := blockCell(ink(pixel), inkColour{}, NewPalette(false))
	if monochrome == coloured {
		t.Errorf("both drawings render a painted top half as %q, so Plain decides nothing",
			monochrome)
	}
}
