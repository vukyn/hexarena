package modifier_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/modifier"
)

// TestAMisspeltFieldInsideAModifierIsRefused is the hole this file closes, and
// the typo is the one that was actually found: `amout` beside `amount`.
//
// ⚠️ It has to be tested **here** rather than through a status, a trait or a
// skill, because that is where it was missed. Every one of those three books can
// set `DisallowUnknownFields` on its own decoder and none of them reaches inside
// a modifier: the flag stops at a custom unmarshaller, and `Modifier` has one.
// So a book asserting its own strictness proves nothing about this level, which
// is exactly how the gap survived the change that closed it one layer up.
func TestAMisspeltFieldInsideAModifierIsRefused(t *testing.T) {
	var held modifier.Modifier
	err := json.Unmarshal([]byte(`{"target":"attack","mode":"flat","amount":100,"amout":100}`), &held)
	if err == nil {
		t.Fatalf("the typo was accepted and the modifier read %v", held)
	}
	if !strings.Contains(err.Error(), "amout") {
		t.Errorf("the refusal does not name the field that was wrong: %v", err)
	}
}

// TestAWellFormedModifierStillReads is the other half, and it is what says the
// stricter decoder refuses a typo rather than the shape it was written for.
func TestAWellFormedModifierStillReads(t *testing.T) {
	for _, written := range []string{
		`{"target":"attack","mode":"flat","amount":100}`,
		`{"target":"speed","mode":"percent","amount":-250}`,
	} {
		var held modifier.Modifier
		if err := json.Unmarshal([]byte(written), &held); err != nil {
			t.Errorf("%s was refused: %v", written, err)
			continue
		}
		if held.Amount == 0 {
			t.Errorf("%s decoded to an empty modifier", written)
		}
	}
}

// TestTheRoundTripSurvivesTheStricterRead is the guard a stricter decoder needs:
// what this package **writes** must still be what it will read. Marshal builds
// the same shape the parser knows, so the two cannot drift — but that is a claim
// about two functions in one file, and the cost of it being wrong is a data file
// the tool wrote and the game refuses to load.
func TestTheRoundTripSurvivesTheStricterRead(t *testing.T) {
	original := modifier.Modifier{Target: modifier.Defense, Mode: modifier.Percent, Amount: -300}
	written, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	var back modifier.Modifier
	if err := json.Unmarshal(written, &back); err != nil {
		t.Fatalf("read back %s: %v", written, err)
	}
	if back != original {
		t.Errorf("the round trip gave %v, wanted %v", back, original)
	}
}
