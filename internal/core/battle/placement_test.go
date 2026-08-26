package battle_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestALogCarriesThePlacementItWasFoughtWith is what a placement being a choice
// costs the log.
//
// While a roster came out of the embedded data and nothing about it was decided,
// re-running a log meant loading that data again and the two were the same
// battle by construction. Now that a placement picks four skills out of nine, a
// log that did not say which four could not be re-run at all — and --verify
// would report the difference between two different battles as corruption.
func TestALogCarriesThePlacementItWasFoughtWith(t *testing.T) {
	roster := []battle.Roster{
		{ID: "a", Name: "Ally", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"strike"}, Passives: []string{"hardy"}},
		{ID: "f", Name: "Foe", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"strike"}},
	}
	raw, err := battle.MarshalLog(battle.Log{Seed: 7, Roster: roster,
		Events: []battle.Event{{Kind: battle.Started, Actor: "a"}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := battle.ParseLog(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(back.Roster) != len(roster) {
		t.Fatalf("the log came back with %d units, want %d", len(back.Roster), len(roster))
	}
	for i := range roster {
		if back.Roster[i].ID != roster[i].ID ||
			strings.Join(back.Roster[i].Skills, " ") != strings.Join(roster[i].Skills, " ") ||
			strings.Join(back.Roster[i].Passives, " ") != strings.Join(roster[i].Passives, " ") ||
			back.Roster[i].Stats != roster[i].Stats ||
			back.Roster[i].Side != roster[i].Side ||
			back.Roster[i].Slot != roster[i].Slot {
			t.Errorf("unit %d came back as %+v, want %+v", i, back.Roster[i], roster[i])
		}
	}
	// A battle really can be built from what came out of the file, which is the
	// property the field exists for rather than merely surviving the round trip.
	fight, err := battle.New(books(t), back.Seed, back.Roster)
	if err != nil {
		t.Fatalf("a battle could not be built from the log's own roster: %v", err)
	}
	fight.Begin()
	if len(fight.Units()) != len(roster) {
		t.Errorf("the rebuilt battle holds %d units, want %d", len(fight.Units()), len(roster))
	}
}

// TestALogWithoutAPlacementSaysItCannotBeReRun is the honest answer for a file
// written before this, and the reason it is a question rather than a silent
// fallback.
//
// Re-running the shipped roster against such a log would compare two different
// battles and report the difference as corruption — a verifier that lies about
// which of the two is wrong.
func TestALogWithoutAPlacementSaysItCannotBeReRun(t *testing.T) {
	old := battle.Log{Seed: 7, Events: []battle.Event{{Kind: battle.Started, Actor: "a"}}}
	if old.Replayable() {
		t.Error("a log with no roster claims it can be re-run")
	}
	full := battle.Log{Seed: 7, Events: old.Events, Roster: []battle.Roster{{ID: "a"}}}
	if !full.Replayable() {
		t.Error("a log carrying its placement says it cannot be re-run")
	}
	// It still parses and still renders: reading a log takes the events and
	// nothing else, so an old file is readable exactly as it always was.
	raw, err := battle.MarshalLog(old)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := battle.ParseLog(raw); err != nil {
		t.Errorf("a log written before placements were recorded no longer parses: %v", err)
	}
}

// TestAPlacementIsWrittenInSnakeCase is about the file rather than the field.
//
// A log is something somebody opens, and every other file this repository writes
// is snake case. Without the tags the placement would come out in Go's field
// names — the same record spelled two ways depending on which layer produced it.
func TestAPlacementIsWrittenInSnakeCase(t *testing.T) {
	raw, err := battle.MarshalLog(battle.Log{
		Seed: 1,
		Roster: []battle.Roster{{ID: "a", Name: "Ally", Side: hex.SideAlly,
			Affinity: single("neutral"), Skills: []string{"strike"}}},
		Events: []battle.Event{{Kind: battle.Started, Actor: "a"}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var file struct {
		Roster []map[string]json.RawMessage `json:"roster"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(file.Roster) != 1 {
		t.Fatalf("the log holds %d placements, want 1", len(file.Roster))
	}
	for _, want := range []string{"id", "name", "side", "element", "stats", "skills"} {
		if _, named := file.Roster[0][want]; !named {
			t.Errorf("the placement has no %q field: %v", want, keysOf(file.Roster[0]))
		}
	}
	for key := range file.Roster[0] {
		if strings.ToLower(key) != key {
			t.Errorf("the placement writes %q, which is not the snake case every other file uses", key)
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	return out
}
