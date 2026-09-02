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
//
// ⚠️ **numberField went with the squad builder and is not coming back.** It was
// the digits-only twin of newInput below, forwarded here on the rule pad, clip,
// clamp and window follow — a body in internal/screen and a call site left
// reading as it read. The squad builder's level field was its **last** caller in
// this package once the skill form's chance field had moved, so what was left
// was a forwarder nothing in the program called: draw.NumberField takes the
// answer off the Palette it is handed, which is where a moved screen already
// gets it.

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
