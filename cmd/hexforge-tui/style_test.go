package main

import (
	"runtime"
	"testing"
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
			if got := plainScreen(c.noColour, c.term, c.goos); got != c.plain {
				t.Errorf("plainScreen(%q, %q, %q) = %v, want %v",
					c.noColour, c.term, c.goos, got, c.plain)
			}
		})
	}
}

// TestPlainTerminalReadsTheMachineItIsOn is the other half: the table above is
// only worth having while the exported question is still asked of the real
// environment and the real platform.
//
// It reads `NO_COLOR` through the same `t.Setenv` every other test here uses, so
// what it pins is the wiring rather than the rule — a `plainTerminal` that had
// stopped consulting one of its three inputs would pass every row above.
func TestPlainTerminalReadsTheMachineItIsOn(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !plainTerminal() {
		t.Error("NO_COLOR is set and plainTerminal still wants colour")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if !plainTerminal() {
		t.Error("TERM is dumb and plainTerminal still wants colour")
	}
	t.Setenv("TERM", "")
	if want := runtime.GOOS != "windows"; plainTerminal() != want {
		t.Errorf("with no TERM on %s, plainTerminal() = %v, want %v",
			runtime.GOOS, plainTerminal(), want)
	}
}
