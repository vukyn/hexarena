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

// forkedStages is a line that splits: one root, two arms at the same threshold,
// and a further stage on one of them so the tree is deeper than the fork.
//
//	Seed ──16──> Vine ──32──> Bloom
//	  └───16──> Fang
//
// Every stage carries the same curve on purpose: this fixture is about the shape
// of the line, and the joint health-and-defence bound is not what it is testing.
func forkedStages(t *testing.T) progression.Line {
	t.Helper()
	return progression.Line{
		{Name: "Seed", MinLevel: 1, Stats: growing(100)},
		{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
		{Name: "Fang", MinLevel: 16, After: "Seed", Stats: growing(100)},
		{Name: "Bloom", MinLevel: 32, After: "Vine", Stats: growing(100)},
	}
}

// TestALineMayFork is the whole of the feature: a level can allow two forms that
// are alternatives rather than one that follows the other.
//
// Two stages at one threshold used to be unwritable — Validate refused a stage
// starting at or before "the previous", which on an ordered list is the only
// predecessor there is. A stage naming what it grows out of is what makes the
// list a tree, and siblings then share a level without arguing.
func TestALineMayFork(t *testing.T) {
	line := forkedStages(t)
	if err := line.Validate(limits(), rules()); err != nil {
		t.Fatalf("a forked line is refused: %v", err)
	}

	// Both arms are offered, because each is a form the level may be fielded as.
	// Nothing here marks an arm as taken: choosing is the placement's job, and a
	// line knows about no unit.
	allowed, err := line.Allowed(20)
	if err != nil {
		t.Fatalf("allowed at 20: %v", err)
	}
	if got := strings.Join(progression.StageNames(allowed), " "); got != "Seed Vine Fang" {
		t.Errorf("level 20 may be fielded as %q, want both arms and the root", got)
	}
	// Past the second threshold on one arm only: the deeper stage replaces the
	// arm it grows out of, and the other arm is untouched by it.
	if furthest, err := line.Furthest(40); err != nil {
		t.Fatalf("furthest at 40: %v", err)
	} else if got := strings.Join(progression.StageNames(furthest), " "); got != "Fang Bloom" {
		t.Errorf("level 40's furthest forms are %q, want the tip of each arm", got)
	}

	// And the choice reaches the stat line, which is the point of making it.
	vine, stage, err := line.Resolve(20, "Vine")
	if err != nil {
		t.Fatalf("resolve as Vine: %v", err)
	}
	if stage.Name != "Vine" {
		t.Errorf("resolving as Vine fielded %q", stage.Name)
	}
	if _, _, err := line.Resolve(20, "Fang"); err != nil {
		t.Fatalf("resolve as Fang: %v", err)
	}
	if vine == (progression.Values{}) {
		t.Error("resolving an arm produced an empty stat line")
	}
	// A stage on the far arm is not reachable by walking past the near one.
	if _, _, err := line.Resolve(20, "Bloom"); err == nil {
		t.Error("Bloom resolved at level 20, before its own threshold")
	}
}

// TestAForkHasNoFurthestAndSaysSo is the half that fails silently if it is got
// wrong, which is why it is asserted on its own.
//
// progression.Furthest is what every caller with nobody to choose for it passes
// — a browser, hexforge check's budget row, a balance harness fielding a cast.
// With two arms reachable there is no answer to give, and the one thing that must
// not happen is picking whichever the file lists last: a stat line is a plausible
// answer, so nothing downstream would notice it was the wrong form's.
func TestAForkHasNoFurthestAndSaysSo(t *testing.T) {
	line := forkedStages(t)

	if _, err := line.StageAt(20); err == nil {
		t.Fatal("a level reaching both arms resolved to one of them")
	} else {
		for _, arm := range []string{"Vine", "Fang"} {
			if !strings.Contains(err.Error(), arm) {
				t.Errorf("the refusal %q does not name %s, which is what the caller has to choose between",
					err, arm)
			}
		}
	}
	if _, _, err := line.Resolve(20, progression.Furthest); err == nil {
		t.Error("resolving a fork with nobody choosing produced a stat line")
	}
	// Below the fork there is still exactly one answer, so a line that forks
	// later behaves like any other line until it does.
	stage, err := line.StageAt(10)
	if err != nil {
		t.Fatalf("below the fork: %v", err)
	}
	if stage.Name != "Seed" {
		t.Errorf("level 10 is %q, want Seed", stage.Name)
	}
	// And a line that does not fork answers as it always did, which is what
	// makes this change invisible to every shipped character.
	if stage, err := threeStages(t).StageAt(40); err != nil || stage.Name != "Bloom" {
		t.Errorf("a linear line answers %v, %v; want Bloom", stage.Name, err)
	}
}

// TestALineIsReadByOrderOrByNameAndNeverBoth is the rule that keeps the file
// honest.
//
// A line saying nothing about predecessors is read by order, which is what every
// line meant before forks existed and why no shipped file moved. A line saying
// anything is read by name only — because a file that names some edges and
// leaves the rest to the order has the order deciding parentage in a document
// that also states it, and the wrong answer would be a stat line rather than an
// error.
func TestALineIsReadByOrderOrByNameAndNeverBoth(t *testing.T) {
	for _, testCase := range []struct {
		name string
		line progression.Line
		want string
	}{
		{"a stage left to the order beside one that names its predecessor", progression.Line{
			{Name: "Seed", MinLevel: 1, Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
			{Name: "Fang", MinLevel: 16, Stats: growing(100)},
		}, "names no predecessor"},
		{"a root that grows out of something", progression.Line{
			{Name: "Seed", MinLevel: 1, After: "Nothing", Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
		}, "nothing comes before the first"},
		{"a predecessor declared after its own stage", progression.Line{
			{Name: "Seed", MinLevel: 1, Stats: growing(100)},
			{Name: "Bloom", MinLevel: 32, After: "Vine", Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
		}, "not declared before it"},
		{"a stage that grows out of nobody in the line", progression.Line{
			{Name: "Seed", MinLevel: 1, Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Root", Stats: growing(100)},
		}, "not declared before it"},
		{"an arm that starts at or before what it grows out of", progression.Line{
			{Name: "Seed", MinLevel: 16, Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
		}, "want 1"},
		{"two stages of one name", progression.Line{
			{Name: "Seed", MinLevel: 1, Stats: growing(100)},
			{Name: "Vine", MinLevel: 16, After: "Seed", Stats: growing(100)},
			{Name: "Vine", MinLevel: 32, After: "Vine", Stats: growing(100)},
		}, "two stages are called"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.line.Validate(limits(), rules())
			if err == nil {
				t.Fatalf("the line was accepted, want a refusal mentioning %q", testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("refused with %q, want it to mention %q", err, testCase.want)
			}
		})
	}
}

// TestALeafIsAFactAboutTheLineRatherThanTheLevel is the leaf rule, and it is
// measured on a **forking fixture** rather than on shipped data on purpose.
//
// The rule's caller is a PvP room's join gate, which insists a squad field fully
// grown units. What makes it worth a predicate of its own is that the obvious
// spelling — "the furthest form the level reaches" — is a different *question*:
// on a forking line at a level between the two thresholds the two answers
// **differ**, which is the discriminator below. `Furthest(20)` names Vine,
// because level 20 has not reached the stage that grows out of it, while Vine is
// not a leaf and never becomes one.
//
// ⚠️ **What this test does NOT hold, and cannot: that the gate needed the
// distinction.** At the cap the two agree on every line `Validate` accepts,
// because `Validate` refuses a stage whose MinLevel is past the cap — so
// `Furthest(LevelCap)` is the tip of each arm by construction. Measured:
// substituting it for `Leaves` inside `IsLeaf` passes this test and all twenty
// of the gate's, and no test can be written that it fails, because no legal line
// exhibits the difference at that level. The gate asks only at 60 (it insists on
// level 60 first), so the level a caller would have to supply is the whole value
// of the name. The assertions below are still worth having — they are what says
// the predicate answers the question it is named after — but they are not
// evidence that the gate would have been wrong without it.
//
// ⚠️ CLAUDE.md records that "nothing shipped forks yet"; that is **stale** —
// `pokemon.poliwag` ships as `Poliwag → Poliwhirl → (Poliwrath | Politoed)`, and
// the gate's own test in internal/room measures it there. This one stays because
// a fixture reaches what shipped data cannot: an interior stage of a fork at a
// level below its child's threshold.
func TestALeafIsAFactAboutTheLineRatherThanTheLevel(t *testing.T) {
	line := forkedStages(t)

	leaves, err := line.Leaves()
	if err != nil {
		t.Fatalf("the leaves of a forked line: %v", err)
	}
	if got := strings.Join(progression.StageNames(leaves), " "); got != "Fang Bloom" {
		t.Errorf("the tips of the line are %q, want the tip of each arm", got)
	}

	for _, one := range []struct {
		stage string
		leaf  bool
		why   string
	}{
		{stage: "Bloom", leaf: true, why: "the deep arm's tip"},
		{stage: "Fang", leaf: true, why: "the shallow arm's whole extent"},
		{stage: "Vine", leaf: false, why: "Bloom grows out of it"},
		{stage: "Seed", leaf: false, why: "both arms grow out of it"},
	} {
		leaf, err := line.IsLeaf(one.stage)
		if err != nil {
			t.Fatalf("is %q a leaf: %v", one.stage, err)
		}
		if leaf != one.leaf {
			t.Errorf("%q reports leaf=%v, want %v — %s", one.stage, leaf, one.leaf, one.why)
		}
	}

	// The discriminator, stated as the comparison rather than as two readings: at
	// a level between the thresholds the level's furthest forms include an
	// interior stage, and the line's tips do not.
	furthest, err := line.Furthest(20)
	if err != nil {
		t.Fatalf("furthest at 20: %v", err)
	}
	if got := strings.Join(progression.StageNames(furthest), " "); got != "Vine Fang" {
		t.Fatalf("level 20's furthest forms are %q; this test needs the interior stage among them", got)
	}
	interior, err := line.IsLeaf("Vine")
	if err != nil {
		t.Fatalf("is Vine a leaf: %v", err)
	}
	if interior {
		t.Error("Vine is a furthest form at level 20 and is reported as a leaf, so the two questions have collapsed into one")
	}

	// A name the line does not answer to is an error rather than a false: a typo
	// and a form with something after it are different mistakes, and a gate whose
	// job is to say which rule a squad broke cannot be handed one answer for both.
	if _, err := line.IsLeaf("Nonesuch"); err == nil {
		t.Error("a stage the line does not have was reported as not-a-leaf rather than refused")
	} else if !strings.Contains(err.Error(), "Nonesuch") {
		t.Errorf("the refusal reads %q, want it to name the stage", err)
	}

	// On a line that does not fork there is one tip, and it is the one Furthest
	// gives at the cap — which is precisely why the shipped cast cannot tell the
	// two questions apart and why this test is written on a fixture.
	linear, err := threeStages(t).Leaves()
	if err != nil {
		t.Fatalf("the leaves of a linear line: %v", err)
	}
	if got := progression.StageNames(linear); len(got) != 1 || got[0] != "Bloom" {
		t.Errorf("a linear line has the tips %q, want just its last stage", got)
	}

	// A line with no stage has no tip, which is a refusal rather than an empty
	// answer: a caller handed nothing would field nothing and hear nothing.
	if _, err := (progression.Line{}).Leaves(); err == nil {
		t.Error("a line with no stage answered with tips")
	}
}
