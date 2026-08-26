package cast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// TestATraitIsHeldFromTheLevelItUnlocksAt is the gate: a character declares when
// each trait comes in, and asking at a level answers with the ones in force.
//
// The three levels that matter are the one before, the one at, and the one after:
// a boundary written with the wrong comparison passes two of the three.
func TestATraitIsHeldFromTheLevelItUnlocksAt(t *testing.T) {
	entry := baseCharacter()
	entry["passives"] = []map[string]any{
		{"id": "endurance"},
		{"id": "resolve", "at_level": 16},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, ok := book.Get("a-series.warden")
	if !ok {
		t.Fatal("the parsed character is not in the book")
	}
	cases := []struct {
		level int
		want  []string
	}{
		{1, []string{"endurance"}},
		{15, []string{"endurance"}},
		{16, []string{"endurance", "resolve"}},
		{17, []string{"endurance", "resolve"}},
		{progression.LevelCap, []string{"endurance", "resolve"}},
	}
	for _, test := range cases {
		if got := character.PassivesAt(test.level); !reflect.DeepEqual(got, test.want) {
			t.Errorf("at level %d the character holds %v, want %v", test.level, got, test.want)
		}
	}
}

// TestAnUnlockedListKeepsTheOrderItWasDeclaredIn is not a tidiness preference.
// The result reaches a roster entry and then an event log, so a list ordered by
// anything other than the file — by the gate, or by whatever a map decided —
// would stop a battle replaying from its seed.
func TestAnUnlockedListKeepsTheOrderItWasDeclaredIn(t *testing.T) {
	entry := baseCharacter()
	entry["passives"] = []map[string]any{
		{"id": "resolve", "at_level": 16},
		{"id": "endurance"},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, _ := book.Get("a-series.warden")
	if got := character.PassivesAt(progression.LevelCap); !reflect.DeepEqual(
		got, []string{"resolve", "endurance"}) {
		t.Errorf("the list came back as %v, want the order it was declared in", got)
	}

	// A character with no traits answers with nothing rather than tripping over
	// the empty list, which is the common case and therefore the one worth
	// checking.
	plain, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bare, _ := plain.Get("a-series.warden")
	if got := bare.PassivesAt(progression.LevelCap); len(got) != 0 {
		t.Errorf("a character with no traits holds %v", got)
	}
}

// TestUnlockedIDsTakesTheListRatherThanTheCharacter is the reason the function
// is shaped the way it is: the kit will ask the same question of a different
// list, and a helper reading a character could not be reused for it.
func TestUnlockedIDsTakesTheListRatherThanTheCharacter(t *testing.T) {
	entries := []cast.Unlock{
		{ID: "first", AtLevel: 1},
		{ID: "second", AtLevel: 30},
		{ID: "third", AtLevel: 60},
	}
	if got := cast.UnlockedIDs(entries, 29); !reflect.DeepEqual(got, []string{"first"}) {
		t.Errorf("at 29 the list gives %v", got)
	}
	if got := cast.UnlockedIDs(entries, 30); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Errorf("at 30 the list gives %v", got)
	}
	if got := cast.UnlockedIDs(entries, 60); len(got) != 3 {
		t.Errorf("at the cap the list gives %v, want all three", got)
	}
	if got := cast.UnlockedIDs(nil, 60); len(got) != 0 {
		t.Errorf("an empty list gives %v", got)
	}
	// Unlocked is the single comparison every caller goes through, so it is
	// worth pinning at its own boundary rather than only through the list.
	gate := cast.Unlock{ID: "x", AtLevel: 16}
	if gate.Unlocked(15) || !gate.Unlocked(16) || !gate.Unlocked(17) {
		t.Error("the boundary is off by one")
	}
}

// TestTheLearnsetSurvivesTheFileAndOmitsTheGateEverybodyPasses is the writer's
// half. The gate is the thing an author edits, so it has to come back as it went
// in — and a level of one is not written, because a gate nobody can fail is noise
// on every entry and would bury the one entry with a real gate.
func TestTheLearnsetSurvivesTheFileAndOmitsTheGateEverybodyPasses(t *testing.T) {
	entry := baseCharacter()
	entry["passives"] = []map[string]any{
		{"id": "endurance"},
		{"id": "resolve", "at_level": 32},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"at_level": 32`) {
		t.Errorf("the gate was not written:\n%s", raw)
	}
	if strings.Contains(string(raw), `"at_level": 1`) {
		t.Errorf("a level of one was written out:\n%s", raw)
	}
	reparsed, err := cast.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v\n%s", err, raw)
	}
	returned, ok := reparsed.Get("a-series.warden")
	if !ok {
		t.Fatal("the round trip lost the character")
	}
	original, _ := book.Get("a-series.warden")
	if !reflect.DeepEqual(returned.Passives, original.Passives) {
		t.Errorf("the round trip changed the learnset:\n%+v\n%+v",
			returned.Passives, original.Passives)
	}
}
