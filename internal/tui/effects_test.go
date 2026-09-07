package tui

import (
	"strings"
	"testing"
)

// The effects column had no bound at all until a squad could hold three
// composition bonuses at once. One unit carrying `phalanx x2 (always)`,
// `tidewell x2 (always)`, `kinship x2 (always)` and a poison draws 124 cells into
// a row that has 60, and `TestEveryWordingFitsTheMinimumWidth` is what said so.
//
// ⚠️ Bonuses stack by design and a per-element table is eight more of them, so
// this row was always going to meet the floor — the third bonus is simply where
// it did. What is asserted here is the arithmetic, because the width sweep can
// only say that some screen was too wide, never which clause spent the cells.

func TestTheEffectsColumnStaysInsideItsRoom(t *testing.T) {
	many := []string{
		"phalanx x2 (always)", "tidewell x2 (always)", "kinship x2 (always)",
		"poison x3 (2t)", "burn (3t)",
	}
	drawn := elided(many, effectsRoom)
	if len(drawn) > effectsRoom {
		t.Errorf("the column drew %d cells into the %d it has:\n%s",
			len(drawn), effectsRoom, drawn)
	}
	// And it says how much it left out, rather than trailing off: a reader who can
	// see that two are hidden knows to open the unit.
	if !strings.Contains(drawn, "+") {
		t.Errorf("%d of %d effects were dropped and the column does not say so:\n%s",
			len(many), len(many), drawn)
	}
	// The premise, held rather than assumed: this list really is too long. A
	// fixture that fitted would pass every line above while measuring nothing.
	if whole := strings.Join(many, ", "); len(whole) <= effectsRoom {
		t.Fatalf("the fixture is %d cells and fits in %d, so nothing was elided",
			len(whole), effectsRoom)
	}
}

func TestAColumnThatFitsIsDrawnWhole(t *testing.T) {
	few := []string{"poison x3 (2t)", "burn (3t)"}
	drawn := elided(few, effectsRoom)
	if want := strings.Join(few, ", "); drawn != want {
		t.Errorf("a list that fits drew %q, want %q", drawn, want)
	}
}

// TestAColumnWithNoRoomAtAllCountsRatherThanBlanks is the edge a narrower row
// reaches: drawing nothing reads as "no effects", which is the one thing it must
// not say about a unit carrying five.
func TestAColumnWithNoRoomAtAllCountsRatherThanBlanks(t *testing.T) {
	if got := elided([]string{"poison x3 (2t)", "burn (3t)"}, 4); got != "+2" {
		t.Errorf("a column with no room drew %q, want a count", got)
	}
	if got := elided(nil, effectsRoom); got != "-" {
		t.Errorf("a unit with no effects drew %q, want a dash", got)
	}
}
