package skill_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// critSkill is one skill's worth of JSON with a clause written into it, so each
// case below differs by exactly the field it is about.
func critSkill(extra string) string {
	return `{"skills":[{"id":"lunge","element":"neutral","range":1,"pattern":"single",
	  "power":1000,"strikes":1,"accuracy":900,"cooldown":2,"target":"enemy"` + extra + `}]}`
}

// TestACriticalChanceIsRefusedForWhatItCannotMean is the parse rule.
//
// Two refusals, and neither is about taste. A share outside parts per thousand
// is not a chance at all, and a chance on a skill with no power is data nothing
// will ever read: the power <= 0 branch in the battle's own turn never reaches
// combat.Roll, so the roll the field asks for is never made.
func TestACriticalChanceIsRefusedForWhatItCannotMean(t *testing.T) {
	for _, testCase := range []struct {
		name, extra, wantErr string
	}{
		{
			"a share below nothing",
			`,"crit":-1`,
			"crits -1, want a share in parts per thousand",
		},
		{
			"a share past certainty",
			`,"crit":1001`,
			"crits 1001, want a share in parts per thousand",
		},
		{
			// Powerless *and* doing something else, because a skill that is
			// powerless and does nothing at all is refused earlier for being a
			// wasted turn. This one is legal in every other respect and the
			// critical chance is the only thing wrong with it.
			"a chance on damage it never deals",
			`,"power":0,"strikes":0,"applies":[{"status":"poison","chance":500,"stacks":1}],"crit":500`,
			"crits on damage it never deals",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := skill.ParseBook([]byte(critSkill(testCase.extra)), deps(t))
			if err == nil {
				t.Fatalf("want a refusal mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("refusal %q does not say %q", err, testCase.wantErr)
			}
		})
	}
}

// TestACriticalChanceSurvivesBeingWritten is the round trip, and it exists
// because Skill.file copies every field by hand: a field added to the parser and
// not to the writer is dropped on the first save either authoring front-end
// makes, silently, with nothing else to notice.
func TestACriticalChanceSurvivesBeingWritten(t *testing.T) {
	book, err := skill.ParseBook([]byte(critSkill(`,"crit":200`)), deps(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	written, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(written), `"crit": 200`) {
		t.Errorf("a skill that crits wrote no crit key: %s", written)
	}
	back, err := skill.ParseBook(written, deps(t))
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	built, err := back.Lookup("lunge")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if built.Crit != 200 {
		t.Errorf("the skill came back critting %d of the time, want 200", built.Crit)
	}
	// The two ends of the range are legal, and nought is what every shipped
	// skill declares.
	for _, legal := range []int{0, 1000} {
		book, err := skill.ParseBook([]byte(critSkill(`,"crit":`+strconv.Itoa(legal))), deps(t))
		if err != nil {
			t.Fatalf("a critical chance of %d was refused: %v", legal, err)
		}
		built, err := book.Lookup("lunge")
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if built.Crit != legal {
			t.Errorf("a critical chance of %d came back as %d", legal, built.Crit)
		}
	}
	// A skill that declares none must write no key at all, or every book, log
	// and golden measured from one without would move the day this landed.
	plain, err := skill.ParseBook([]byte(critSkill("")), deps(t))
	if err != nil {
		t.Fatalf("parse the plain skill: %v", err)
	}
	bare, err := plain.Marshal()
	if err != nil {
		t.Fatalf("marshal the plain skill: %v", err)
	}
	if strings.Contains(string(bare), `"crit"`) {
		t.Errorf("a skill that cannot crit wrote a crit key: %s", bare)
	}
}
