package main

import (
	"runtime"
	"testing"
)

// TestPlainTerminalReadsTheMachineItIsOn is the half that stayed behind when the
// rule moved into internal/screen: the table over there
// (TestAnUnsetTermIsOnlyDumbAwayFromWindows) is only worth having while the
// question is still asked of the real environment and the real platform.
//
// It reads `NO_COLOR` through the same `t.Setenv` every other test here uses, so
// what it pins is the wiring rather than the rule — a `plainTerminal` that had
// stopped consulting one of its three inputs would pass every row over there.
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
