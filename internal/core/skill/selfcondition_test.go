package skill_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// TestBothConditionsAreRefusedTheSameWay is why there is one validator rather
// than two.
//
// Every rule below is about the shape of a condition and nothing about whose
// health or whose stacks it counts, so a self_requires that accepted something a
// requires refuses would be a second set of rules nobody wrote down — and the
// looser of the two would be the one an author found. The table runs each case
// through both fields and expects the same answer.
func TestBothConditionsAreRefusedTheSameWay(t *testing.T) {
	for _, test := range []struct {
		name      string
		condition string
		says      string
	}{
		{"asks nothing", `{"bonus_power":900}`, "asks nothing"},
		{"stacks with no status", `{"min_stacks":2,"below_health":500,"bonus_power":900}`, "names no status"},
		{"a share outside the scale", `{"below_health":1400,"bonus_power":900}`, "parts per thousand"},
		{"negative power", `{"below_health":500,"bonus_power":-1}`, "zero or more"},
		{"consumes what it never names", `{"below_health":500,"bonus_power":900,"consume":true}`, "names none"},
		{"consumes for nothing", `{"status":"poison","consume":true}`, "for neither a bonus, a discharge nor a per-stack payment"},
		{"more stacks than exist", `{"status":"poison","min_stacks":9,"bonus_power":900}`, "caps at"},
		{"a status nobody declared", `{"status":"nonesuch","bonus_power":900}`, "nonesuch"},
	} {
		for _, field := range []string{"requires", "self_requires"} {
			_, err := parseOne(t, field, test.condition, "enemy")
			if err == nil {
				t.Errorf("%s was accepted as %s", test.name, field)
				continue
			}
			if !strings.Contains(err.Error(), test.says) {
				t.Errorf("%s as %s said %q, which does not mention %q",
					test.name, field, err, test.says)
			}
			// The message names which of the two was wrong, because a skill
			// carrying both otherwise gets a refusal it cannot act on.
			if !strings.Contains(err.Error(), field) {
				t.Errorf("%s as %s said %q, which never names the field", test.name, field, err)
			}
		}
	}
}

// TestASelfConditionOnASelfAimedSkillIsRefused, because such a skill never
// reaches a target and its power lands nowhere.
//
// The alternative is a field an author writes, a tool accepts and a battle
// ignores, which is the failure a status carrying a health term is already
// refused for.
func TestASelfConditionOnASelfAimedSkillIsRefused(t *testing.T) {
	_, err := parseOne(t, "self_requires", `{"below_health":500,"bonus_power":900}`, "self")
	if err == nil {
		t.Fatal("a self-aimed skill was allowed a power bonus it can never land")
	}
	if !strings.Contains(err.Error(), "deals none") {
		t.Errorf("the refusal says %q, which does not say why", err)
	}
	// Without the bonus it is fine: consuming a status to do something that is
	// not damage is a perfectly ordinary thing to write.
	if _, err := parseOne(t, "self_requires", `{"status":"poison","min_stacks":1}`, "self"); err != nil {
		t.Errorf("a self-aimed skill reading its own status without a power bonus was refused: %v", err)
	}
}

// TestASelfConditionSurvivesBeingWrittenBack is the round trip every field has
// to make: hexforge reads the book, writes it back, and a field it dropped would
// be a field an author loses by opening the tool.
func TestASelfConditionSurvivesBeingWrittenBack(t *testing.T) {
	book, err := parseOne(t, "self_requires",
		`{"status":"poison","min_stacks":2,"bonus_power":900,"consume":true}`, "enemy")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	written, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := skill.ParseBook(written, deps(t))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	back, err := again.Lookup("probe")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if back.SelfRequires == nil {
		t.Fatal("the caster's own condition did not survive being written back")
	}
	if *back.SelfRequires != (skill.Condition{
		Status: "poison", MinStacks: 2, BonusPower: 900, Consume: true,
	}) {
		t.Errorf("it came back as %+v", *back.SelfRequires)
	}
	if back.Requires != nil {
		t.Error("a target condition appeared out of nowhere, so the two fields are crossed")
	}
}

// parseOne builds a one-skill book with the given condition on the given field.
func parseOne(t *testing.T, field, condition, target string) (*skill.Book, error) {
	t.Helper()
	// A self-aimed skill must declare a range of nought and no power of its own,
	// which are rules of their own and not what any of this is about.
	// A self-aimed skill declares no range and no power of its own, and needs
	// something to do or it is refused as a wasted turn. None of that is what
	// these tests are about, so it is arranged here rather than in each of them.
	reach, power, extra := "1", "100", ""
	if target == "self" {
		reach, power = "0", "0"
		extra = `"self_applies":[{"status":"block","chance":1000}],`
	}
	return skill.ParseBook([]byte(`{"skills":[
	  {"id":"probe","element":"neutral","range":`+reach+`,"pattern":"single",
	   "power":`+power+`,"strikes":1,"accuracy":1000,"cooldown":0,"target":"`+target+`",
	   `+extra+`"`+field+`":`+condition+`}
	]}`), deps(t))
}
