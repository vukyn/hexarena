package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// TestOnlyTheGatedTierOfATwoTierTraitIsLeftOutOfTheHeldLine is what the skip in
// Held has always meant, now that it can be asked one grant at a time.
//
// The figure this function answers is what a character carries for the whole
// fight, and a gated grant is not that at either end — so it is left out, and
// deciding otherwise is a measurement rather than an edit. What changed is the
// *granularity*: the skip used to drop a whole trait the moment anything about
// it was gated, which for a two-tier trait would throw away the tier that is
// always on as well. Now the tier that never lapses is counted and only the
// gated one is left out.
//
// ⚠️ Three readings rather than one, because a single figure cannot say which of
// the two tiers was counted: the two-tier trait has to come to exactly the
// ungated tier's own figure, and that figure has to be above the bare line.
func TestOnlyTheGatedTierOfATwoTierTraitIsLeftOutOfTheHeldLine(t *testing.T) {
	dir := scratchData(t)
	// Written through the file rather than by building a Library by hand,
	// because a gated grant is only admitted by the parse — and no shipped trait
	// carries one, so a test over the shipped book would pass with the whole
	// change reverted.
	addPassivesToScratch(t, dir,
		map[string]any{
			"id":     "first_tier_fixture",
			"grants": []any{map[string]any{"status": "toughened", "stacks": 1}},
		},
		map[string]any{
			"id": "two_tier_fixture",
			"grants": []any{
				map[string]any{"status": "toughened", "stacks": 1},
				map[string]any{"status": "fortified", "stacks": 1,
					"while": map[string]any{"above_health": 700}},
			},
		},
		map[string]any{
			"id":    "whole_gated_fixture",
			"while": map[string]any{"above_health": 700},
			"grants": []any{
				map[string]any{"status": "toughened", "stacks": 1},
				map[string]any{"status": "fortified", "stacks": 1},
			},
		},
	)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load the scratch data: %v", err)
	}
	character, known := lib.Characters().Get("pokemon.charmander")
	if !known {
		t.Fatal("no charmander to measure")
	}
	base, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve charmander: %v", err)
	}
	under := func(trait string) progression.Values {
		t.Helper()
		values, err := lib.Held(base, []string{trait})
		if err != nil {
			t.Fatalf("hold %s: %v", trait, err)
		}
		return values
	}
	one := under("first_tier_fixture")
	two := under("two_tier_fixture")
	whole := under("whole_gated_fixture")

	if one == base {
		t.Fatal("the ungated tier moved nothing, so Held is not applying a grant at all " +
			"and every comparison below is between two copies of the base line")
	}
	if two != one {
		t.Errorf("a two-tier trait reports %s, want the ungated tier's own %s: the gated "+
			"tier has to be left out and the tier that never lapses has to be counted",
			two.String(), one.String())
	}
	if two == base {
		t.Errorf("a two-tier trait reports the bare line %s, so the skip threw away the "+
			"tier that is always on as well as the gated one", base.String())
	}
	// And a trait gated as a whole still counts for nothing, which is the case
	// blaze is and the behaviour that must not have moved.
	if whole != base {
		t.Errorf("a whole-trait gated trait moved the line to %s from %s", whole.String(), base.String())
	}
	t.Logf("defence: %d bare, %d one tier, %d two tier",
		base[progression.Defense], one[progression.Defense], two[progression.Defense])
}

// addPassivesToScratch appends declarations to the passive book in a scratch
// data directory, in the form the parser reads.
//
// It is cmd/hexforge's addPassives one package over, and it is a copy rather
// than a shared helper because it exists only to hand this package's parse a
// declaration nothing ships: two test files needing the same three lines of JSON
// plumbing is not a mechanism worth exporting from a production package.
func addPassivesToScratch(t *testing.T, dir string, extra ...map[string]any) {
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
