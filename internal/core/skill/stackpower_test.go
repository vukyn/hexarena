package skill_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// TestAPerStackPaymentIsTheCastersSideOfTheBoard is the mirror of the rule that
// keeps a chain and an arc on the target's condition.
//
// An arc and a chain read the board in front of the caster — which bodies are
// carrying what, and where they are standing — so a caster-side one would be
// reading a board it is not pointed at. A per-stack payment into the caster's own
// power reads nothing about the board at all, which is exactly why it may be
// caster-side; and the target's condition already HAS a per-stack currency, so
// allowing a second there would be the ceiling charged twice with nothing
// downstream able to say which half a figure came from.
func TestAPerStackPaymentIsTheCastersSideOfTheBoard(t *testing.T) {
	spend := `{"status":"fuel","min_stacks":2,"consume":true,"stack_power":200}`
	if _, err := parseOne(t, "self_requires", spend, "enemy"); err != nil {
		t.Fatalf("a caster's own per-stack payment was refused: %v", err)
	}
	_, err := parseOne(t, "requires", spend, "enemy")
	if err == nil {
		t.Fatal("a target's condition was allowed a per-stack payment as well as an arc")
	}
	if !strings.Contains(err.Error(), "arc_power") {
		t.Errorf("the refusal says %q, which does not send the author to the currency that side already has", err)
	}
}

// TestAPerStackPaymentIsRefusedWhereItWouldBeFree collects the three shapes an
// author can write that would pay for the same stack twice, or for ever.
func TestAPerStackPaymentIsRefusedWhereItWouldBeFree(t *testing.T) {
	for _, test := range []struct {
		name      string
		condition string
		says      string
	}{
		{
			"paid without spending anything",
			`{"status":"fuel","min_stacks":2,"stack_power":200}`,
			"without consuming anything",
		},
		{
			"paid twice for one spend",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_power":200,"bonus_power":900}`,
			"the same purchase made twice",
		},
		{
			"a rate past the ceiling",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_power":4001}`,
			"the second stack would be worth nothing",
		},
		{
			"a negative rate",
			`{"status":"fuel","min_stacks":2,"consume":true,"stack_power":-1}`,
			"zero or more",
		},
	} {
		_, err := parseOne(t, "self_requires", test.condition, "enemy")
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s said %q, which does not mention %q", test.name, err, test.says)
		}
	}
}

// TestASpendTakesOnlyWhatTheCeilingPaysFor is the clamp read as arithmetic, and
// it is on the STACKS rather than on the power.
//
// MaxSpendPower bounds what one cast may buy. Clamping the bonus would leave the
// caster handing over a pile it did not use — so a full reserve emptied into a
// capped blow would be worth less per stack than a small one, and the rating
// would like the spend best exactly where it wasted most. Clamping what is taken
// means the leftovers stay in the tank.
//
// ⚠️ Both halves are asserted from the same pair of functions: Takes is what
// Battle.spend removes and SelfBonus is what the blow lands with, and the point
// of the rule is that they cannot disagree.
func TestASpendTakesOnlyWhatTheCeilingPaysFor(t *testing.T) {
	const rate = 200
	ceiling := skill.MaxSpendPower / rate
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":2,"consume":true,"stack_power":200}`, "enemy")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spender, err := book.Lookup("probe")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	for _, held := range []int{2, ceiling - 1, ceiling, ceiling + 5, 40} {
		taken := spender.SelfRequires.Takes(held)
		bonus := spender.SelfBonus(skill.Carrying(held))
		want := held
		if want > ceiling {
			want = ceiling
		}
		if taken != want {
			t.Errorf("holding %d, the spend took %d, want %d", held, taken, want)
		}
		if bonus != taken*rate {
			t.Errorf("holding %d, the spend took %d stacks and bought %d power, which is not %d a stack",
				held, taken, bonus, rate)
		}
		if bonus > skill.MaxSpendPower {
			t.Errorf("holding %d, one spend bought %d, over the ceiling of %d",
				held, bonus, skill.MaxSpendPower)
		}
	}
}

// TestTheSpendCeilingIsReadFromTakesRatherThanTheThreshold is why SelfCeiling
// exists beside Satisfying.
//
// Satisfying is the CHEAPEST target a condition holds against, and for a flat
// bonus the cheapest case and the dearest one are the same number — so one
// reading served both until a payment could scale. A bound taken at the threshold
// would be a bound on the smallest blow the skill can throw.
func TestTheSpendCeilingIsReadFromTakesRatherThanTheThreshold(t *testing.T) {
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":2,"consume":true,"stack_power":200}`, "enemy")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spender, _ := book.Lookup("probe")
	atThreshold := spender.SelfBonus(spender.SelfRequires.Satisfying())
	if ceiling := spender.SelfCeiling(); ceiling <= atThreshold {
		t.Errorf("the ceiling reads %d and the threshold reads %d: a scaling payment that measured the same at both is not scaling",
			ceiling, atThreshold)
	}
	// And a flat bonus still answers the same at both, which is what says the two
	// readings only separated where they had to.
	flat, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":2,"consume":true,"bonus_power":900}`, "enemy")
	if err != nil {
		t.Fatalf("parse the flat one: %v", err)
	}
	steady, _ := flat.Lookup("probe")
	if steady.SelfCeiling() != steady.SelfBonus(steady.SelfRequires.Satisfying()) {
		t.Errorf("a flat bonus reads %d at its ceiling and %d at its threshold",
			steady.SelfCeiling(), steady.SelfBonus(steady.SelfRequires.Satisfying()))
	}
}

// TestAPerStackPaymentSurvivesBeingWrittenBack is the round trip every field has
// to make: hexforge reads the book and writes it back, and a field it dropped is
// a field an author loses by opening the tool.
func TestAPerStackPaymentSurvivesBeingWrittenBack(t *testing.T) {
	book, err := parseOne(t, "self_requires",
		`{"status":"fuel","min_stacks":3,"consume":true,"consume_stacks":5,"stack_power":200}`, "enemy")
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
		Status: "fuel", MinStacks: 3, Consume: true, ConsumeStacks: 5, StackPower: 200,
	}) {
		t.Errorf("it came back as %+v", *back.SelfRequires)
	}
}
