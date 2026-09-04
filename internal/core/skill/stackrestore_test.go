package skill_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// Every case here is written on a SELF-aimed skill, and that is the parser
// rather than the fixture. A caster's own condition pays its own caster, so a
// per-stack restore on an enemy-aimed skill is refused by resolve before any
// condition is looked at — and a table that used the enemy aim would measure
// that one refusal over and over while believing it measured these.

// TestAPerStackHealIsTheCastersSideOfTheBoard is the health twin of the rule that
// keeps a per-stack power payment off the target's condition.
//
// A caster's own condition reads the caster and pays the caster, which is the
// whole of what it does. A target's condition reads the unit at the aim, so a
// health payment filed there would be the enemy's pile buying the caster health,
// with nothing downstream able to say whose stacks paid for it.
func TestAPerStackHealIsTheCastersSideOfTheBoard(t *testing.T) {
	spend := `{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":200}`
	if _, err := parseOne(t, "self_requires", spend, "self"); err != nil {
		t.Fatalf("a caster's own per-stack heal was refused: %v", err)
	}
	_, err := parseOne(t, "requires", spend, "self")
	if err == nil {
		t.Fatal("a target's condition was allowed to pay the caster health")
	}
	if !strings.Contains(err.Error(), "only the caster's own condition may do") {
		t.Errorf("the refusal says %q, which does not say whose side the payment is on", err)
	}
}

// TestAPerStackHealIsRefusedWhereItWouldBeFree collects every shape an author can
// write that would be paid for nothing, paid twice, or paid off two ceilings.
//
// ⚠️ **Both fields, and they answer differently on purpose.** The table runs each
// case through `self_requires` and `requires` the way TestBothConditionsAre
// RefusedTheSameWay does, but only the first is asserted against its own message:
// the health currency is caster-side, so on `requires` the caster-side rule
// answers first and answers every one of these with the same sentence. What is
// asserted there is that the target's condition refuses the shape at all, which is
// the property that matters — the looser of two fields is the one an author finds.
func TestAPerStackHealIsRefusedWhereItWouldBeFree(t *testing.T) {
	for _, test := range []struct {
		name      string
		condition string
		says      string
	}{
		{
			"paid without spending anything",
			`{"status":"fuel","min_stacks":2,"stack_restore":200}`,
			"without consuming anything",
		},
		{
			"paid twice for one spend",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":200,"bonus_power":900}`,
			"the same purchase made twice",
		},
		{
			"paid in both currencies at once",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":200,"stack_power":200}`,
			"two ceilings answering one question",
		},
		{
			"a rate past the ceiling",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":2001}`,
			"the second stack would be worth nothing",
		},
		{
			"a negative rate",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":-1}`,
			"zero or more",
		},
	} {
		_, err := parseOne(t, "self_requires", test.condition, "self")
		if err == nil {
			t.Errorf("%s was accepted", test.name)
		} else {
			if !strings.Contains(err.Error(), test.says) {
				t.Errorf("%s said %q, which does not mention %q", test.name, err, test.says)
			}
			if !strings.Contains(err.Error(), "self_requires") {
				t.Errorf("%s said %q, which never names the field", test.name, err)
			}
		}
		if _, err := parseOne(t, "requires", test.condition, "self"); err == nil {
			t.Errorf("%s was accepted as a target's condition, so the two fields have different rules", test.name)
		}
	}
}

// TestAHealSpendTakesOnlyWhatTheCeilingPaysFor is the clamp read as arithmetic,
// and it is on the STACKS rather than on the health.
//
// MaxSpendRestore bounds what one cast may buy. Clamping the health would leave
// the caster handing over a pile it did not use, so a full reserve emptied into a
// capped heal would be worth less per stack than a shallow one and the rating
// would like the spend best exactly where it wasted most.
//
// ⚠️ **This is the only place the 999-stack case is reachable.** The status book
// caps a reserve at 999 and no board banks anywhere near it, so a clamp that
// stopped working would show up in nothing that is fought.
func TestAHealSpendTakesOnlyWhatTheCeilingPaysFor(t *testing.T) {
	const rate = 200
	ceiling := skill.MaxSpendRestore / rate
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":2,"consume":true,"stack_restore":200}`, "self")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spender, err := book.Lookup("probe")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	for _, held := range []int{2, ceiling - 1, ceiling, ceiling + 5, 999} {
		taken := spender.SelfRequires.Takes(held)
		healed := spender.SelfRestore(skill.Carrying(held))
		want := held
		if want > ceiling {
			want = ceiling
		}
		if taken != want {
			t.Errorf("holding %d, the spend took %d, want %d", held, taken, want)
		}
		if healed != taken*rate {
			t.Errorf("holding %d, the spend took %d stacks and bought %d health, which is not %d a stack",
				held, taken, healed, rate)
		}
		if healed > skill.MaxSpendRestore {
			t.Errorf("holding %d, one spend bought %d health, over the ceiling of %d",
				held, healed, skill.MaxSpendRestore)
		}
	}
	// And the ceiling names the same figure, which is what the report prints and
	// what a bound on the payout has to be read against.
	if got := spender.SelfRestoreCeiling(); got != ceiling*rate {
		t.Errorf("the ceiling reads %d, want %d", got, ceiling*rate)
	}
}

// TestAHealSpendPaysNothingOnAnEmptyTank is the whole reason the payment is
// per-stack rather than a flat `restores` beside a condition.
//
// A condition is an amplifier: a caster who fails it is charged nothing and
// stopped by nothing, so a flat restore on such a skill pays out in full to a
// caster holding no fuel at all. Written per stack, "no fuel, no heal" is
// arithmetic and there is no gate anywhere for anybody to forget.
func TestAHealSpendPaysNothingOnAnEmptyTank(t *testing.T) {
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":5,"consume":true,"stack_restore":200}`, "self")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spender, _ := book.Lookup("probe")
	for _, held := range []int{0, 1, 4} {
		if healed := spender.SelfRestore(skill.Carrying(held)); healed != 0 {
			t.Errorf("holding %d stacks against a threshold of 5, the spend paid %d", held, healed)
		}
	}
	if healed := spender.SelfRestore(skill.Carrying(5)); healed != 1000 {
		t.Errorf("holding the threshold exactly, the spend paid %d, want 1000", healed)
	}
}

// TestAPerStackHealSurvivesBeingWrittenBack is the round trip every field has to
// make: hexforge reads the book and writes it back, and a field it dropped is a
// field an author loses by opening the tool.
func TestAPerStackHealSurvivesBeingWrittenBack(t *testing.T) {
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":3,"consume":true,"consume_stacks":5,"stack_restore":200}`, "self")
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
	// Compared through reflect rather than with ==, because a Condition carries a
	// rider list and a struct holding a slice is not comparable. The whole value
	// is still what is asserted: a field-by-field check would stop covering the
	// next field somebody adds, which is the failure this test exists to catch.
	if want := (skill.Condition{
		Status: "fuel", MinStacks: 3, Consume: true, ConsumeStacks: 5, StackRestore: 200,
	}); !reflect.DeepEqual(*back.SelfRequires, want) {
		t.Errorf("it came back as %+v, want %+v", *back.SelfRequires, want)
	}
}
