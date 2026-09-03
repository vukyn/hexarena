package screen

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
)

// The art preview's own drawing, measured at the pixel rather than through a
// screen: these functions take colours and hand back cells, so nothing here
// needs a library, a Context or a client.
//
// ⚠️ **This file used to be the whole of what any suite said about the drawing,
// and it only ever asserted ink versus blank.** Measured then: swapping the red
// and green weights in `luminance`, and swapping `▀` for `▄` in `blockCell`,
// each left `go test ./...` entirely green. The preview is now in both clients'
// screen sweeps and has a golden entry of its own — but a golden is taken under
// NO_COLOR, so it records `rampCell` and **cannot see `blockCell` at all**, which
// is why the two properties those mutations break are written down here rather
// than left to the record.
//
//	the weights   TestTheRampWeighsGreenOverRedOverBlue
//	the two halves TestEachPixelIsDrawnInItsOwnHalfOfTheCell

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

// TestARecoveredColourSaturates is about the one arithmetic step between a
// rasterised pixel and a cell: alpha-premultiplied storage means the colour has
// to be divided back out, and division back out can leave a byte.
//
// A channel above its own alpha is not something a rasteriser produces, so this
// is a guard rather than a case. It is worth holding anyway because of what the
// wrong answer looks like: without the ceiling the conversion wraps, and a
// wrapped bright colour comes out DARK rather than merely wrong. A hole in the
// middle of a lit shape reads as a bug in the picture, and it would be hunted
// in the rasteriser rather than here.
func TestARecoveredColourSaturates(t *testing.T) {
	for _, test := range []struct {
		name           string
		channel, alpha uint8
		want           uint8
	}{
		// The ordinary path: half coverage stored dark, handed back bright.
		{"half covered", 100, 200, 127},
		{"fully covered", 200, 255, 200},
		// Above its alpha. 120 over 100 recovers as 306, which wraps to 50 — a
		// dark cell where the brightest one belongs.
		{"brighter than its coverage", 120, 100, 255},
		{"far past its coverage", 255, 41, 255},
		// No coverage at all never reaches here through ink, but the division
		// still has to have an answer.
		{"no coverage", 200, 0, 0},
	} {
		if got := unpremultiply(test.channel, test.alpha); got != test.want {
			t.Errorf("%s: %d over %d recovered as %d, want %d",
				test.name, test.channel, test.alpha, got, test.want)
		}
	}
}

// TestTheRampWeighsGreenOverRedOverBlue is the monochrome drawing's one
// arithmetic decision, and the first of the two mutations that used to pass.
//
// `luminance` is Rec. 601 in integer thousandths — 299 red, 587 green, 114 blue —
// and what those three numbers buy is that the ramp keeps the shading rather than
// merely the shape: a leaf and the shadow under it are the same silhouette and
// different weights. The ordering is the whole of that promise, so it is the
// ordering that is asserted rather than the constants: **green weighs most, blue
// weighs least**, at equal channel strength.
//
// ⚠️ Asserted **through rampCell as well as through luminance**, because the
// ramp is inverted against the weight — a heavier character for darker ink — and
// an ordering that held on the number while the inversion turned over would be a
// picture drawn inside out with this test green. So the second half walks the
// alphabet: a brighter pixel is a character further towards Ramp's light end.
//
// Swapping the red and green weights turns green from the brightest primary into
// the second-darkest, which reverses both orderings at once.
func TestTheRampWeighsGreenOverRedOverBlue(t *testing.T) {
	const full = 255
	red := ink(color.RGBA{R: full, A: full})
	green := ink(color.RGBA{G: full, A: full})
	blue := ink(color.RGBA{B: full, A: full})
	if !(green.luminance() > red.luminance() && red.luminance() > blue.luminance()) {
		t.Errorf("the weights read green %d, red %d, blue %d; the ramp keeps the shading "+
			"only while green weighs most and blue least",
			green.luminance(), red.luminance(), blue.luminance())
	}
	// And the same ordering as the reader sees it: Ramp runs lightest to
	// heaviest, so a brighter pixel has to land on an earlier character.
	at := func(pixel inkColour) int {
		drawn := rampCell(pixel, pixel)
		place := strings.Index(Ramp, drawn)
		if place < 0 {
			t.Fatalf("rampCell drew %q, which is not in the ramp %q", drawn, Ramp)
		}
		return place
	}
	if !(at(green) < at(red) && at(red) < at(blue)) {
		t.Errorf("the ramp draws green at %d, red at %d and blue at %d of %q; a brighter "+
			"pixel has to draw a lighter character or the picture is inside out",
			at(green), at(red), at(blue), Ramp)
	}
}

// TestEachPixelIsDrawnInItsOwnHalfOfTheCell is the second mutation that used to
// pass, and the property the golden cannot hold.
//
// Two pixel rows go into one terminal row, so every cell carries two pixels and
// the only thing saying which is which is **which half block is drawn and which
// way round the two colours are hung on it**. `▀` paints the foreground on top and
// the background under it; `▄` paints the foreground underneath. So the four
// branches of blockCell are four spellings of one claim — the top pixel is drawn
// in the top half — and swapping the two characters keeps every branch
// well-formed while turning the picture upside down, one cell at a time.
//
// ⚠️ **A golden cannot see any of this.** Goldens here are taken under NO_COLOR,
// where the screen draws `rampCell` instead, so the coloured half of the drawing
// is measured by this test or by nothing at all.
//
// It needs no environment: blockCell builds a bare lipgloss style rather than
// asking the Palette it is handed, so it writes a truecolor sequence whatever
// the terminal is — which the fixture checks before it reads one, since a cell
// with no sequence in it would make every colour claim below vacuously true.
func TestEachPixelIsDrawnInItsOwnHalfOfTheCell(t *testing.T) {
	red := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	green := color.RGBA{R: 40, G: 200, B: 40, A: 255}
	style := NewPalette(false)
	if drawn := blockCell(ink(red), ink(green), style); !strings.Contains(drawn, "\x1b[") {
		t.Fatalf("a painted cell rendered as %q with no colour in it, so nothing below "+
			"measures which half a pixel landed in", drawn)
	}
	for _, test := range []struct {
		name        string
		top, bottom color.RGBA
	}{
		{"the top half alone", red, color.RGBA{}},
		{"the bottom half alone", color.RGBA{}, green},
		{"both halves", red, green},
	} {
		upper, lower := halvesOf(t, blockCell(ink(test.top), ink(test.bottom), style))
		if want := paintedAs(test.top); upper != want {
			t.Errorf("%s: the cell's upper half is %s, want the top pixel's %s",
				test.name, upper, want)
		}
		if want := paintedAs(test.bottom); lower != want {
			t.Errorf("%s: the cell's lower half is %s, want the bottom pixel's %s",
				test.name, lower, want)
		}
	}
}

// blank is what a half of a cell holds when nothing was painted into it: the
// terminal's own background, which is a colour this program never names.
const blank = "unpainted"

// paintedAs is the colour a pixel should come out as, written the way halvesOf
// reads one back.
func paintedAs(pixel color.RGBA) string {
	drawn := ink(pixel)
	if !drawn.painted {
		return blank
	}
	return fmt.Sprintf("%d;%d;%d", drawn.r, drawn.g, drawn.b)
}

// halvesOf takes a rendered cell apart into the colour in its top half and the
// colour in its bottom half.
//
// ⚠️ **It reads the half block rather than trusting it**, which is the whole
// point: `▀` hangs the foreground on the upper half and the background on the
// lower, `▄` does the opposite, and a cell with neither is the plain space that
// paints nothing. That mapping is the terminal's, not this program's, so it is
// the one fact the assertion may take for granted.
func halvesOf(t *testing.T, cell string) (upper, lower string) {
	t.Helper()
	foreground, background := blank, blank
	body := cell
	if opened := strings.Index(body, "\x1b["); opened == 0 {
		closed := strings.Index(body, "m")
		if closed < 0 {
			t.Fatalf("the cell %q opens a sequence it never closes", cell)
		}
		fields := strings.Split(body[len("\x1b["):closed], ";")
		// Each colour is five parameters: 38 or 48, the 2 that says "these are
		// three channels", and the channels.
		for at := 0; at+5 <= len(fields); at++ {
			if fields[at+1] != "2" {
				continue
			}
			value := strings.Join(fields[at+2:at+5], ";")
			switch fields[at] {
			case "38":
				foreground = value
			case "48":
				background = value
			}
		}
		body = body[closed+1:]
	}
	body = strings.TrimSuffix(body, "\x1b[m")
	switch body {
	case " ":
		return blank, blank
	case "▀":
		return foreground, background
	case "▄":
		return background, foreground
	}
	t.Fatalf("the cell %q draws %q, which is neither half block nor a space — a third "+
		"character in the coloured drawing needs its own half named here", cell, body)
	return blank, blank
}
