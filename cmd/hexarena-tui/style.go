package main

import (
	"os"
	"runtime"

	draw "github.com/vukyn/hexarena/internal/screen"
)

// This file is the one thing internal/screen may not do: read the machine. The
// palette itself and the rule behind plainTerminal both live in that package,
// because two full-screen clients draw the same screens and must not have a
// second answer to either. Reading the environment is the binary's business;
// answering the question is not.
//
// It is cmd/hexforge-tui/style.go with the text-field forwarder taken off:
// nothing this client draws has a text field on it, because every form in
// internal/screen is reached through a key this client does not offer.

// newPalette picks the styles for the terminal this program is attached to.
func newPalette() draw.Palette { return draw.NewPalette(plainTerminal()) }

// plainTerminal reports whether colour would be noise rather than help.
//
// The three inputs are read here and the rule is draw.Plain's, which takes them
// as parameters because `runtime.GOOS` cannot be faked and both of its answers
// have to be assertable from either sort of machine — an unset TERM is a dumb
// terminal only away from Windows, which is a rule nothing in a suite running on
// one platform can reach.
func plainTerminal() bool {
	return draw.Plain(os.Getenv("NO_COLOR"), os.Getenv("TERM"), runtime.GOOS)
}
