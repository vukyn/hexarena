// A condition that decides whether the skill may be CAST, rather than what the
// cast pays out.
//
// Every other condition in this package is an amplifier, and the four refusals
// below are the four places that difference bites: a gate is read once per skill
// on the caster alone, and everything about it has to stay askable there.
package skill_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// TestAGatedSpendNeedsNoPaymentOnItsCondition is the exemption the whole shape
// rests on, and it is the one clause an author would trip over first.
//
// "Consumes for neither a bonus, a discharge, a per-stack payment nor a rider"
// is right for every other condition here, because those conditions are paid ON
// TOP of a cast that was happening anyway — so a consume buying none of them
// really is a status handed over for nothing. A gate is not on top of anything:
// the consume buys the cast itself, and the flat power on the skill's own face is
// the whole figure rather than a figure the condition failed to move.
//
// It is the StackRestore exemption one currency along. There the entire heal
// lives on the condition, so the skill reads `restores: 0` and would have been
// refused as a wasted turn; here the entire purchase is the cast, so the
// condition reads as paying nothing.
func TestAGatedSpendNeedsNoPaymentOnItsCondition(t *testing.T) {
	gated := `{"status":"fuel","min_stacks":3,"gates":true,"consume":true,"consume_stacks":3}`
	book, err := parseOne(t, "self_requires", gated, "enemy")
	if err != nil {
		t.Fatalf("a gated spend paying nothing on its condition was refused: %v", err)
	}
	spender, err := book.Lookup("probe")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !spender.SelfRequires.GatesCast() {
		t.Error("the condition parsed but does not report itself as gating, so nothing downstream would gate on it")
	}
	// The control: the same condition without the gate is still refused, so what
	// the case above measures is the exemption rather than a rule that stopped
	// being enforced.
	ungated := `{"status":"fuel","min_stacks":3,"consume":true,"consume_stacks":3}`
	if _, err := parseOne(t, "self_requires", ungated, "enemy"); err == nil {
		t.Error("a consume paying nothing and gating nothing was accepted, so the exemption above swallowed the whole rule")
	} else if !strings.Contains(err.Error(), "throws the status away for nothing") {
		t.Errorf("the ungated refusal says %q, which is not the clause the exemption was cut out of", err)
	}
}

// TestAGateIsRefusedWhereItCouldNotBeRead is the three shapes a gate may not
// take, and each is refused because the reading would have to be taken somewhere
// it cannot be.
func TestAGateIsRefusedWhereItCouldNotBeRead(t *testing.T) {
	for _, test := range []struct {
		name      string
		field     string
		condition string
		says      string
	}{
		{
			// options() builds one reason per SKILL. A target's condition is read
			// per aim, so a gate there would have to live in aims() -- which is the
			// one function required to stay blind to a gate, because four callers
			// ask it hypothetically and one asks it on an empty tank on purpose.
			"a gate on the target's condition",
			"requires",
			`{"status":"fuel","min_stacks":3,"gates":true,"consume":true,"consume_stacks":3,"bonus_power":900}`,
			"only the caster's own condition may gate a cast",
		},
		{
			// A gate that spends nothing is a permanent unlock: the caster crosses
			// the threshold once and the skill is free from that turn on. What makes
			// a gate a price is that casting empties what opened it.
			"a gate that consumes nothing",
			"self_requires",
			`{"status":"fuel","min_stacks":3,"gates":true,"bonus_power":900}`,
			"the threshold would be crossed once and never paid again",
		},
		{
			// Health is read off a unit, so "may this be cast" would become a
			// question with one answer per cell. That is a different mechanic, and
			// it is one aims() would have to answer.
			"a gate on health alone",
			"self_requires",
			`{"below_health":500,"gates":true,"consume":true}`,
			"a gate on health alone is a reading taken per target rather than per cast",
		},
	} {
		_, err := parseOne(t, test.field, test.condition, "enemy")
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s said %q, which does not mention %q", test.name, err, test.says)
		}
		if !strings.Contains(err.Error(), test.field) {
			t.Errorf("%s said %q, which never names the field", test.name, err)
		}
	}
}

// TestAGateSurvivesBeingWrittenBack is the round trip every field has to make:
// hexforge reads the book and writes it back, and a field it dropped is a field
// an author loses by opening the tool.
//
// ⚠️ **Compared whole-value rather than field by field, and that is the whole
// test.** A gate is a bool, so a dropped one comes back as `false` — which parses,
// round-trips again and reads as "an ordinary amplifier". A field-by-field check
// covers only the fields somebody remembered to list, so it would pass against a
// conditionFile literal that never mentioned Gates at all. reflect.DeepEqual on
// the whole Condition is what makes the next field somebody adds covered too.
//
// ⚠️ **Two conditions, and the second is the one that measures the comparison.**
// A gate paying nothing else is illegal without the gate, so dropping the field
// from the writer breaks the RE-PARSE and any assertion at all would catch it. A
// gate beside a flat bonus is legal either way: strip the gate and it round-trips
// clean, as a perfectly ordinary amplifier that is simply a different skill. Only
// a whole-value comparison can see that one.
func TestAGateSurvivesBeingWrittenBack(t *testing.T) {
	for _, test := range []struct {
		name      string
		condition string
		want      skill.Condition
	}{
		{
			"a gate that pays nothing else",
			`{"status":"fuel","min_stacks":3,"gates":true,"consume":true,"consume_stacks":3}`,
			skill.Condition{Status: "fuel", MinStacks: 3, Gates: true, Consume: true, ConsumeStacks: 3},
		},
		{
			"a gate beside a flat bonus",
			`{"status":"fuel","min_stacks":3,"gates":true,"consume":true,"consume_stacks":3,"bonus_power":900}`,
			skill.Condition{Status: "fuel", MinStacks: 3, Gates: true, Consume: true, ConsumeStacks: 3, BonusPower: 900},
		},
	} {
		// A subtest each, so the case that can only fail at the comparison still
		// runs when the case that fails at the re-parse has already stopped.
		t.Run(test.name, func(t *testing.T) {
			book, err := parseOne(t, "self_requires", test.condition, "enemy")
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
			if !reflect.DeepEqual(*back.SelfRequires, test.want) {
				t.Errorf("it came back as %+v, want %+v", *back.SelfRequires, test.want)
			}
		})
	}
}
