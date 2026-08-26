package progression_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// threeStages is a line with room to choose in: a level past the last threshold
// may be fielded as any of the three.
func threeStages(t *testing.T) progression.Line {
	t.Helper()
	return progression.Line{
		{Name: "Seed", MinLevel: 1, Stats: growing(100)},
		{Name: "Sprout", MinLevel: 16, Stats: growing(200)},
		{Name: "Bloom", MinLevel: 32, Stats: growing(300)},
	}
}

func growing(base int64) progression.Table {
	return progression.Table{
		progression.HP:       {Base: base * 10, Max: base * 30},
		progression.Attack:   {Base: base, Max: base * 3},
		progression.Defense:  {Base: base, Max: base * 3},
		progression.Speed:    {Base: base / 10, Max: base / 3},
		progression.Accuracy: {Base: base / 10, Max: base / 3},
		progression.Dodge:    {Base: base / 20, Max: base / 6},
	}
}

// TestALevelAllowsAFormRatherThanDictatingOne is the whole of chosen evolution.
//
// The stage used to be derived: whatever the level reached, with no decision
// anywhere in it — which made "may evolve" and "does evolve" the same sentence.
func TestALevelAllowsAFormRatherThanDictatingOne(t *testing.T) {
	line := threeStages(t)

	allowed, err := line.Allowed(40)
	if err != nil {
		t.Fatalf("allowed at 40: %v", err)
	}
	if got := progression.StageNames(allowed); strings.Join(got, " ") != "Seed Sprout Bloom" {
		t.Errorf("level 40 may be fielded as %v, want every form", got)
	}
	// A prefix rather than a filter: reaching Bloom reaches the two before it.
	allowed, err = line.Allowed(20)
	if err != nil {
		t.Fatalf("allowed at 20: %v", err)
	}
	if got := progression.StageNames(allowed); strings.Join(got, " ") != "Seed Sprout" {
		t.Errorf("level 20 may be fielded as %v, want the two it has reached", got)
	}

	// And the choice reaches the stat line, which is the point of making it.
	grown, _, err := line.Resolve(40, "Bloom")
	if err != nil {
		t.Fatalf("resolve as Bloom: %v", err)
	}
	young, stage, err := line.Resolve(40, "Seed")
	if err != nil {
		t.Fatalf("resolve as Seed: %v", err)
	}
	if stage.Name != "Seed" {
		t.Errorf("fielding Seed reported %q", stage.Name)
	}
	if young[progression.Attack] >= grown[progression.Attack] {
		t.Errorf("the earlier form is not weaker: Seed %d, Bloom %d",
			young[progression.Attack], grown[progression.Attack])
	}
}

// TestNotChoosingIsTheAnswerEverybodyHadBefore is what keeps every caller that
// has no placement behind it working as it always did.
func TestNotChoosingIsTheAnswerEverybodyHadBefore(t *testing.T) {
	line := threeStages(t)
	for _, level := range []int{1, 15, 16, 31, 32, progression.LevelCap} {
		reached, err := line.StageAt(level)
		if err != nil {
			t.Fatalf("stage at %d: %v", level, err)
		}
		values, stage, err := line.Resolve(level, progression.Furthest)
		if err != nil {
			t.Fatalf("resolve at %d: %v", level, err)
		}
		if stage.Name != reached.Name {
			t.Errorf("level %d without a choice fields %q, want the furthest %q",
				level, stage.Name, reached.Name)
		}
		named, _, err := line.Resolve(level, reached.Name)
		if err != nil {
			t.Fatalf("resolve at %d as %q: %v", level, reached.Name, err)
		}
		if named != values {
			t.Errorf("level %d resolves differently named and unnamed", level)
		}
	}
}

// TestAFormAheadOfTheLevelIsRefusedRatherThanClamped is the one outcome worse
// than saying no: fielding a different unit from the one written down.
//
// The two refusals are told apart because they are different mistakes. A name
// the line does not answer to is a typo; a name that is merely ahead of the
// level is a placement that has not grown into it.
func TestAFormAheadOfTheLevelIsRefusedRatherThanClamped(t *testing.T) {
	line := threeStages(t)

	_, _, err := line.Resolve(20, "Bloom")
	if err == nil {
		t.Fatal("a level 20 placement was fielded as a form that begins at 32")
	}
	if !strings.Contains(err.Error(), "begins at level 32") {
		t.Errorf("the refusal reads %q, want it to say when the form begins", err)
	}

	_, _, err = line.Resolve(40, "Nonesuch")
	if err == nil {
		t.Fatal("a placement was fielded as a form the line does not have")
	}
	if !strings.Contains(err.Error(), "Seed") || !strings.Contains(err.Error(), "Bloom") {
		t.Errorf("the refusal reads %q, want it to offer the forms there are", err)
	}
}

// TestALineWithOneFormStillResolves is the ordinary character, which is most of
// them: one stage, nothing to choose, and no new failure mode.
func TestALineWithOneFormStillResolves(t *testing.T) {
	line := progression.Line{{Name: "Only", MinLevel: 1, Stats: growing(100)}}
	if _, _, err := line.Resolve(30, progression.Furthest); err != nil {
		t.Errorf("a single-stage line without a choice: %v", err)
	}
	if _, _, err := line.Resolve(30, "Only"); err != nil {
		t.Errorf("a single-stage line named: %v", err)
	}
	allowed, err := line.Allowed(30)
	if err != nil {
		t.Fatalf("allowed: %v", err)
	}
	if len(allowed) != 1 {
		t.Errorf("a single-stage line offers %d forms", len(allowed))
	}
}
