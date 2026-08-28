package cast_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// TestASkillKeptForAWorkAsksWhereTheCarrierIsFrom is the axis a character
// allowlist could not express.
//
// A work outlives every character in it, so naming the one character who has
// the skill today says "only this one may carry it" when it means "only
// somebody out of this story may" — and it would have to be edited again the
// next time that work lends the cast anybody.
func TestASkillKeptForAWorkAsksWhereTheCarrierIsFrom(t *testing.T) {
	for _, test := range []struct {
		name    string
		origin  string
		refused bool
	}{
		{"is out of it", "a-series", false},
		{"is out of another", "a-game", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			declared := baseCharacter()
			declared["origin"] = test.origin
			declared["skills"] = learn("strike", "signature_throw")
			_, err := parse(t, declared)
			switch {
			case test.refused && err == nil:
				t.Fatal("a character out of another work carried a skill kept from it")
			case !test.refused && err != nil:
				t.Fatalf("a character out of the work it is kept for was refused: %v", err)
			}
			if !test.refused {
				return
			}
			for _, want := range []string{"signature_throw", "only somebody from a-series"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal is %q, want it to mention %q", err, want)
				}
			}
		})
	}
}

// TestASkillKeptForAnUnknownWorkIsRefusedByItsCarrier is the typo case, worth
// its own refusal for the reason the species one is: an allowlist naming a work
// nobody declared admits nobody, and reads as a rule rather than as the
// misspelling it is.
//
// ⚠️ Unlike the species check there is no "without the book" case to cover. The
// origin catalog is not optional — cast.Deps.validate refuses to check anything
// without it — because every character names an origin while only some claim a
// species.
func TestASkillKeptForAnUnknownWorkIsRefusedByItsCarrier(t *testing.T) {
	book, err := skillsWithOrigin(t, "a-serial")
	if err != nil {
		t.Fatalf("the skill book itself is legal: %v", err)
	}
	declared := baseCharacter()
	declared["skills"] = learn("strike", "signature_throw")
	raw, err := json.Marshal(map[string]any{"characters": []map[string]any{declared}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	with := deps(t)
	with.Skills = book
	if _, err := cast.ParseBook(raw, with); err == nil ||
		!strings.Contains(err.Error(), `unknown origin "a-serial"`) {
		t.Errorf("a skill kept for a work nobody declared gave %v", err)
	}
}

// skillsWithOrigin is the shared fixture's signature skill pointed at a work the
// caller names, so the typo case can be built without touching that fixture.
func skillsWithOrigin(t *testing.T, kept string) (*skill.Book, error) {
	t.Helper()
	patterns, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3, "splash_power": 500,
	  "patterns": [{"name": "single", "splash": []}]
	}`))
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [{"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500}]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	raw := `{"skills":[
	  {"id": "strike", "element": "neutral", "range": 1, "pattern": "single",
	   "power": 1000, "accuracy": 950, "target": "enemy"},
	  {"id": "riptide", "element": "water", "range": 2, "pattern": "single",
	   "power": 1600, "accuracy": 900, "target": "enemy"},
	  {"id": "signature_throw", "element": "neutral", "range": 1, "pattern": "single",
	   "power": 1200, "accuracy": 900, "target": "enemy",
	   "restrict": {"origins": ["` + kept + `"]}}
	]}`
	return skill.ParseBook([]byte(raw), skill.Deps{Patterns: patterns, Statuses: statuses})
}

// TestAPresetMayHoldASkillKeptForAWork is the deliberate asymmetry, and it is
// tested because it looks like a hole.
//
// A preset refuses a skill kept for named characters and one kept for a lineage,
// on the ground that a preset is shared so the refusal would land on the next
// author. That sentence is just as true of a work, and the ban would still be
// wrong: a lineage is exceptional and a work is universal. Every skill comes out
// of some fiction, so the ban would empty every preset in the directory rather
// than trim two entries off two of them — and an empty kit does not load.
//
// What is left of the harm lands where it belongs. A character out of another
// work built from this preset *without naming its own skills* is refused by the
// character check above, which says which work the kit is out of.
func TestAPresetMayHoldASkillKeptForAWork(t *testing.T) {
	declared := baseArchetype()
	declared["skills"] = []string{"strike", "signature_throw"}
	if _, err := archetypes(t, declared); err != nil {
		t.Errorf("a preset was refused a skill kept for a work: %v", err)
	}
}
