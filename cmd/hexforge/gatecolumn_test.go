package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// TestTheGateColumnSaysTheRightWordForEachEnd.
//
// The cell was `"under " + percent + " health"` with the word a literal, which
// was true while a gate could only sit at the bottom of the health bar. The
// moment one could sit at the top of it the same literal made the column state
// the *opposite* of the data — and an author reading it would build a squad
// around a trait that lapses exactly when the listing says it arrives.
//
// ⚠️ **No shipped trait is gated at the top of the bar**, so the trait this reads
// is written into the scratch data directory rather than taken out of the book.
// A test over the shipped passives alone would pass with the literal put back:
// every shipped gate really is at the bottom, so every row really does say
// "under". That is the same reason internal/i18n needs a hand-built trait.
func TestTheGateColumnSaysTheRightWordForEachEnd(t *testing.T) {
	dir := scratchData(t)
	// One trait per end, on numbers that cannot be confused for one another, so
	// a cell showing the wrong figure is as visible as one showing the wrong
	// word. They are appended to the scratch copy, which is a per-test temporary
	// directory — the shipped book is not touched and no golden can see this.
	addPassives(t, dir,
		map[string]any{
			"id":     "unbowed_fixture",
			"grants": []any{},
			"while":  map[string]any{"above_health": 900},
			"drains": 250,
		},
		map[string]any{
			"id":     "cornered_fixture",
			"grants": []any{},
			"while":  map[string]any{"below_health": 250},
			"drains": 250,
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
		{"unbowed_fixture", "over 90% health", "under"},
		{"cornered_fixture", "under 25% health", "over"},
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
	}
	if rows != len(cases) {
		t.Fatalf("checked %d rows, want %d", rows, len(cases))
	}
	// And every shipped gate still reads, so the change did not trade one end
	// for the other. Counted, because a walk that found none would pass silently.
	shipped := 0
	for _, held := range lib.Passives().All() {
		if held.While == nil || strings.HasSuffix(held.ID, "_fixture") {
			continue
		}
		shipped++
		if !rowFor(listing, held.ID, "under ") {
			t.Errorf("%q is gated at the bottom of the bar and its row does not "+
				"say so:\n%s", held.ID, listing)
		}
	}
	if shipped == 0 {
		t.Error("the scratch book holds no gate of its own, so this measured only " +
			"the two traits it wrote itself")
	}
}

// addPassives appends declarations to the passive book in a scratch data
// directory, in the form the parser reads.
//
// Written through the file rather than by building a forge.Library by hand
// because the cell under test is rendered off a *parsed* book: a hand-built one
// would skip the parse, which is where a gate at the top of the bar is admitted
// at all.
func addPassives(t *testing.T, dir string, extra ...map[string]any) {
	t.Helper()
	path := filepath.Join(dir, "passives.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the scratch passive book: %v", err)
	}
	var book struct {
		Passives []map[string]any `json:"passives"`
	}
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatalf("decode the scratch passive book: %v", err)
	}
	book.Passives = append(book.Passives, extra...)
	written, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		t.Fatalf("encode the scratch passive book: %v", err)
	}
	if err := os.WriteFile(path, append(written, '\n'), 0o600); err != nil {
		t.Fatalf("write the scratch passive book: %v", err)
	}
}
