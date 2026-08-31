package main

import (
	"os"
	"runtime"

	"charm.land/bubbles/v2/textinput"

	draw "github.com/vukyn/hexarena/internal/screen"
)

// What is left in this file is the one thing internal/screen may not do: read
// the machine. The palette, the text field's dress and the rule behind
// plainTerminal all live in that package now, because a second full-screen
// client draws the same screens and must not have a second answer to any of
// them. Reading the environment is the binary's business; answering the question
// is not.

// newPalette picks the styles for the terminal this program is attached to.
func newPalette() draw.Palette { return draw.NewPalette(plainTerminal()) }

// newInput is a text field dressed the way this program draws them.
func newInput() textinput.Model { return draw.NewInput(plainTerminal()) }

// plainTerminal reports whether colour would be noise rather than help.
//
// The three inputs are read here and the rule is draw.Plain's, which takes them
// as parameters because `runtime.GOOS` cannot be faked and both of its answers
// have to be assertable from either sort of machine.
func plainTerminal() bool {
	return draw.Plain(os.Getenv("NO_COLOR"), os.Getenv("TERM"), runtime.GOOS)
}
