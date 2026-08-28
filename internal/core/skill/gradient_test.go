package skill_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// gradientSkill is one skill's worth of JSON with a gradient block written into
// it, so each case below differs by exactly the clause it is about.
func gradientSkill(extra string) string {
	return `{"skills":[{"id":"lunge","element":"neutral","range":1,"pattern":"single",
	  "power":1000,"strikes":1,"accuracy":900,"cooldown":2,"target":"enemy"` + extra + `}]}`
}

// TestAGradientIsRefusedForWhatItCannotMean is the parse rule, and every case is
// a skill that would otherwise load and then quietly do nothing.
//
// Nothing here is a bound on taste. A gradient worth two thousand at the bottom
// is a triple and is allowed, because unlike a pierce there is no point past
// which the number stops meaning anything — the refusals are all about a curve
// with nothing to be a curve of, or two curves off one number.
func TestAGradientIsRefusedForWhatItCannotMean(t *testing.T) {
	for _, testCase := range []struct {
		name, extra, wantErr string
	}{
		{
			"nothing at the bottom",
			`,"self_gradient":{"at_empty":0}`,
			"want a share in parts per thousand",
		},
		{
			"a share below nothing",
			`,"self_gradient":{"at_empty":-500}`,
			"want a share in parts per thousand",
		},
		{
			// Powerless *and* doing something else, because a skill that is
			// powerless and does nothing at all is refused a line earlier for
			// being a wasted turn. This one is a legal skill in every other
			// respect and the gradient is the only thing wrong with it.
			"a share of no power",
			`,"power":0,"strikes":0,"applies":[{"status":"poison","chance":500,"stacks":1}]` +
				`,"self_gradient":{"at_empty":500}`,
			"the whole curve is worth nothing",
		},
		{
			// Range nought as well, because a self-aimed skill that reaches
			// anywhere is refused before this rule is reached.
			"aimed at its own caster",
			`,"target":"self","range":0,"self_gradient":{"at_empty":500}`,
			"aimed at itself",
		},
		{
			"beside a threshold on the same health",
			`,"self_requires":{"below_health":400,"bonus_power":500},"self_gradient":{"at_empty":500}`,
			"two curves off one number",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			raw := gradientSkill(testCase.extra)
			// The last two cases override a key rather than adding one, and the
			// last-wins reading of a duplicate key is what makes that work. Stated
			// because a reader checking the fixture will notice the repeat.
			_, err := skill.ParseBook([]byte(raw), deps(t))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

// TestAGradientComposesWithAThresholdOnAStatus is the other side of the refusal
// above, and the reason it asks what the condition *reads* rather than whether
// there is one.
//
// Two curves off the caster's health is unreadable. A threshold on a status and a
// gradient off health are two different questions about two different things, and
// a skill is allowed to ask both.
func TestAGradientComposesWithAThresholdOnAStatus(t *testing.T) {
	book, err := skill.ParseBook([]byte(gradientSkill(
		`,"self_requires":{"status":"poison","min_stacks":1,"bonus_power":500},`+
			`"self_gradient":{"at_empty":500}`)), deps(t))
	if err != nil {
		t.Fatalf("a status threshold beside a gradient was refused: %v", err)
	}
	built, err := book.Lookup("lunge")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if built.SelfRequires == nil || built.SelfGradient == nil {
		t.Fatalf("the skill kept requires=%v gradient=%v", built.SelfRequires, built.SelfGradient)
	}
}

// TestAGradientSurvivesBeingWritten is the round trip, and it exists because the
// writer copies every sub-field by hand: a block added to the parser and not to
// Skill.file() is dropped on the first save the authoring tool makes, silently
// and with no test anywhere else to notice.
func TestAGradientSurvivesBeingWritten(t *testing.T) {
	book, err := skill.ParseBook([]byte(gradientSkill(`,"self_gradient":{"at_empty":750}`)), deps(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	written, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := skill.ParseBook(written, deps(t))
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	built, err := back.Lookup("lunge")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if built.SelfGradient == nil {
		t.Fatal("the gradient did not survive being written")
	}
	if built.SelfGradient.AtEmpty != 750 {
		t.Errorf("the gradient came back worth %d at the bottom, want 750", built.SelfGradient.AtEmpty)
	}
	// A skill that declares none must still write no block at all, or every log
	// and golden measured from a book without one would move the day this landed.
	plain, err := skill.ParseBook([]byte(gradientSkill("")), deps(t))
	if err != nil {
		t.Fatalf("parse the plain skill: %v", err)
	}
	bare, err := plain.Marshal()
	if err != nil {
		t.Fatalf("marshal the plain skill: %v", err)
	}
	if strings.Contains(string(bare), "self_gradient") {
		t.Errorf("a skill with no gradient wrote one: %s", bare)
	}
}

// TestSelfScaleIsNilSafeAndReadsTheCastersBar covers the accessor the battle and
// the rating both go through, including the case no data file can produce: a
// skill that declares nothing being asked anyway.
func TestSelfScaleIsNilSafeAndReadsTheCastersBar(t *testing.T) {
	book, err := skill.ParseBook([]byte(gradientSkill(`,"self_gradient":{"at_empty":800}`)), deps(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	built, err := book.Lookup("lunge")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	for _, testCase := range []struct {
		name        string
		health, max int64
		want        int
	}{
		{"untouched", 1000, 1000, 0},
		{"a quarter gone", 750, 1000, 200},
		{"nothing left", 0, 1000, 800},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := built.SelfScale(testCase.health, testCase.max); got != testCase.want {
				t.Errorf("the share at %d of %d is %d, want %d",
					testCase.health, testCase.max, got, testCase.want)
			}
		})
	}
	if got := (skill.Skill{}).SelfScale(1, 1000); got != 0 {
		t.Errorf("a skill declaring no gradient is worth %d, want nothing", got)
	}
}
