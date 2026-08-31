package screen

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
)

// TestAnUnsetTermIsOnlyDumbAwayFromWindows is the one thing in this program that
// has a different right answer on two platforms, so it is the one thing that has
// to be asserted for both from either.
//
// The bug it holds down shipped and was invisible from the machine it was written
// on: `TERM` is terminfo's convention and no native Windows terminal sets one, so
// reading its absence as a dumb terminal painted every cmd.exe, PowerShell and
// Windows Terminal session in plain text — while macOS and Linux, which always set
// it, never reached the branch at all. Nothing in the suite could see it either,
// because every other test here sets `NO_COLOR` and returns at the line above.
func TestAnUnsetTermIsOnlyDumbAwayFromWindows(t *testing.T) {
	for _, c := range []struct {
		name     string
		noColour string
		term     string
		goos     string
		plain    bool
	}{
		{"unset TERM on windows draws colour", "", "", "windows", false},
		{"unset TERM anywhere else is dumb", "", "", "darwin", true},
		{"unset TERM on linux is dumb", "", "", "linux", true},
		{"a dumb TERM is dumb on windows too", "", "dumb", "windows", true},
		{"a dumb TERM is dumb elsewhere", "", "dumb", "darwin", true},
		{"an ordinary TERM draws colour", "", "xterm-256color", "darwin", false},
		{"a TERM set under windows is believed", "", "xterm-256color", "windows", false},
		// NO_COLOR is asked first and is the whole answer: a reader who has said
		// so gets plain text on every platform and whatever TERM says.
		{"NO_COLOR wins on windows", "1", "xterm-256color", "windows", true},
		{"NO_COLOR wins elsewhere", "1", "xterm-256color", "darwin", true},
		{"NO_COLOR wins with no TERM at all", "1", "", "windows", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Plain(c.noColour, c.term, c.goos); got != c.plain {
				t.Errorf("Plain(%q, %q, %q) = %v, want %v",
					c.noColour, c.term, c.goos, got, c.plain)
			}
		})
	}
}

// TestEveryElementHasAStyle is the completeness the array buys.
//
// The colours are indexed by the enum, so a twelfth element added tomorrow gets a
// zero value rather than a missing key — which draws plain and looks deliberate.
// Only neutral is meant to be undecorated.
func TestEveryElementHasAStyle(t *testing.T) {
	for _, member := range element.All() {
		if member == element.Neutral {
			continue
		}
		if elementColours[member] == "" {
			t.Errorf("%v has no colour, so it draws like the inert element", member)
		}
	}
}
