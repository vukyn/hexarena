package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// TestAGatedGrantSaysItsGateInTheGrantsCell.
//
// The while column words the *trait's* gate, and a trait carrying a gated grant
// has no trait-level gate at all — the parser refuses both at once. So a reader
// with only that column would see a two-tier trait as two grants that are always
// both on, which is the listing reporting less than the parser accepts: the same
// gap the answers and drains columns were added to close.
//
// ⚠️ **No shipped trait carries a gated grant**, so the traits this reads are
// written into the scratch data directory rather than taken out of the book. A
// test over the shipped passives alone would pass with the clause deleted,
// because no shipped row has one to print — the same reason the gate column's
// own test writes its two traits itself.
func TestAGatedGrantSaysItsGateInTheGrantsCell(t *testing.T) {
	dir := scratchData(t)
	// One per end of the bar, on figures that cannot be confused for each other
	// or for the trait-level gates already in the book, and each beside an
	// ungated grant so the row shows the two-tier shape rather than one clause.
	addPassives(t, dir,
		map[string]any{
			"id": "bulwark_fixture",
			"grants": []any{
				map[string]any{"status": "toughened", "stacks": 1},
				map[string]any{"status": "fortified", "stacks": 1,
					"while": map[string]any{"above_health": 700}},
			},
		},
		map[string]any{
			"id": "lastditch_fixture",
			"grants": []any{
				map[string]any{"status": "toughened", "stacks": 1},
				map[string]any{"status": "fortified", "stacks": 1,
					"while": map[string]any{"below_health": 150}},
			},
		},
	)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load the scratch data: %v", err)
	}
	var drawn strings.Builder
	renderPassives(&drawn, lib)
	listing := drawn.String()

	cases := []struct {
		id, want, wrong string
	}{
		{"bulwark_fixture", "fortified (over 70% health)", "under 70%"},
		{"lastditch_fixture", "fortified (under 15% health)", "over 15%"},
	}
	rows := 0
	for _, test := range cases {
		rows++
		if !rowFor(listing, test.id, test.want) {
			t.Errorf("%q's row does not say %q:\n%s", test.id, test.want, listing)
		}
		if rowFor(listing, test.id, test.wrong) {
			t.Errorf("%q's row says %q, which is the other end of the bar:\n%s",
				test.id, test.wrong, listing)
		}
		// The ungated tier carries no clause, or every grant on the row reads as
		// gated and the two tiers are indistinguishable again.
		if rowFor(listing, test.id, "toughened (") {
			t.Errorf("%q's ungated grant printed a gate clause:\n%s", test.id, listing)
		}
	}
	if rows != len(cases) {
		t.Fatalf("checked %d rows, want %d", rows, len(cases))
	}

	// And every shipped gate still reads out of the while column, so a clause
	// beside a grant did not move a trait-level one out of it. Counted, because
	// a walk that found none would pass silently.
	shipped := 0
	for _, held := range lib.Passives().All() {
		if held.While == nil || strings.HasSuffix(held.ID, "_fixture") {
			continue
		}
		shipped++
		if !rowFor(listing, held.ID, "under ") {
			t.Errorf("%q is gated at the bottom of the bar and its row does not say so:\n%s",
				held.ID, listing)
		}
	}
	if shipped == 0 {
		t.Error("the scratch book holds no trait-level gate of its own, so this measured " +
			"only the two traits it wrote itself")
	}
}
