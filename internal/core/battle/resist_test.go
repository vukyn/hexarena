package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// poisoner is a battle where one unit does nothing but try to poison the other,
// with the target holding whatever traits a case wants.
//
// envenom inflicts on a full thousand, so every refusal in these tests is the
// target's doing and never the dice — which is the only way to measure a
// resistance without counting rolls.
func poisoner(t *testing.T, traits ...string) *battle.Battle {
	t.Helper()
	return mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"envenom"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}, Passives: traits},
	})
}

// attack takes one turn with the poisoner and returns what came of it.
func attack(t *testing.T, fight *battle.Battle) []battle.Event {
	t.Helper()
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("envenom", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	return fight.Drain()
}

// TestAFullResistanceIsImmunityAndSaysSoInTheLog is the case the feature was
// asked for: a trait that refuses a status outright.
//
// envenom cannot fail its own roll, so a target that is not poisoned by it is a
// target that refused it — and the log has to say which of the two happened,
// because the kind is called status_resisted either way.
func TestAFullResistanceIsImmunityAndSaysSoInTheLog(t *testing.T) {
	events := attack(t, poisoner(t, "clean_blood"))

	if applied := find(events, battle.StatusApplied); len(applied) != 0 {
		t.Errorf("an immune target was poisoned: %+v", applied)
	}
	refused := find(events, battle.StatusResisted)
	if len(refused) != 1 {
		t.Fatalf("the log holds %d refusals, want 1", len(refused))
	}
	switch {
	case refused[0].Refused != 1000:
		t.Errorf("the event says %d refused, want the whole thousand", refused[0].Refused)
	case refused[0].Chance != 0:
		t.Errorf("the chance rolled was %d, want nothing left of it", refused[0].Chance)
	case refused[0].Status != "poison":
		t.Errorf("the event names %q", refused[0].Status)
	}
	// And the unit itself carries nothing, which is the fact the log is claiming.
	fight := poisoner(t, "clean_blood")
	attack(t, fight)
	if stacks := target(t, fight).Statuses.Stacks("poison"); stacks != 0 {
		t.Errorf("the immune target carries %d stacks of poison", stacks)
	}
}

// TestATargetWithNoTraitIsUnaffected is the control. Without it the test above
// would pass just as well against a battle where nothing is ever poisoned.
func TestATargetWithNoTraitIsUnaffected(t *testing.T) {
	events := attack(t, poisoner(t))
	applied := find(events, battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("a target with no trait was poisoned %d times, want once", len(applied))
	}
	if applied[0].Refused != 0 {
		t.Errorf("the event says %d refused on a target that resists nothing", applied[0].Refused)
	}
	if applied[0].Chance != 1000 {
		t.Errorf("the chance was %d, want the skill's own thousand", applied[0].Chance)
	}
}

// TestAPartialResistanceTakesItsShareOfTheChance is the ratio rather than the
// switch: six hundred refused leaves four hundred, at face value.
//
// A single resistance has to be exact. Composing it through a saturation helper
// would land a declared six hundred somewhere near three seventy-five, which is
// a trait quietly weaker than it was authored to be.
func TestAPartialResistanceTakesItsShareOfTheChance(t *testing.T) {
	fight := poisoner(t, "thick_blood")
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("envenom", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	// Either outcome is legal at four hundred, so what is measured is the
	// arithmetic the log reports rather than which way the die fell.
	for _, event := range fight.Drain() {
		if event.Kind != battle.StatusApplied && event.Kind != battle.StatusResisted {
			continue
		}
		if event.Refused != 600 {
			t.Errorf("the event says %d refused, want the 600 the trait declares", event.Refused)
		}
		if event.Chance != 400 {
			t.Errorf("the chance was %d, want the 400 left of a thousand", event.Chance)
		}
	}
}

// TestTwoResistancesComposeRatherThanAddUp is why no saturation helper is
// needed: two of six hundred leave sixteen percent, not none.
//
// Adding them would reach a thousand and hand a stack the absolute that only an
// author is meant to be able to declare.
func TestTwoResistancesComposeRatherThanAddUp(t *testing.T) {
	fight := poisoner(t, "thick_blood", "cold_blood")
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("envenom", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	seen := 0
	for _, event := range fight.Drain() {
		if event.Kind != battle.StatusApplied && event.Kind != battle.StatusResisted {
			continue
		}
		seen++
		// 600 and 600: each lets four hundred through, so 160 survives and 840
		// is refused. Adding would give 1200, clamped to an immunity nobody
		// declared.
		if event.Refused != 840 {
			t.Errorf("two resistances of 600 refused %d, want 840", event.Refused)
		}
		if event.Chance != 160 {
			t.Errorf("the chance was %d, want 160", event.Chance)
		}
	}
	if seen == 0 {
		t.Error("the turn produced no application at all, so nothing was measured")
	}
}

// TestAResistanceOnlyBitesTheStatusItNames is the precision an id buys over a
// category: poison refused, and the stun from the same battle untouched.
func TestAResistanceOnlyBitesTheStatusItNames(t *testing.T) {
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"daze"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}, Passives: []string{"clean_blood"}},
	})
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("daze", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	events := fight.Drain()
	applied := find(events, battle.StatusApplied)
	if len(applied) != 1 || applied[0].Status != "stun" {
		t.Fatalf("the stun did not land on a target immune to poison: %+v", applied)
	}
	if applied[0].Refused != 0 {
		t.Errorf("a trait naming poison refused %d of a stun", applied[0].Refused)
	}
}

// target is the unit a poisoner is aimed at, for a test that wants its state
// rather than the log.
func target(t *testing.T, fight *battle.Battle) *battle.Unit {
	t.Helper()
	unit, ok := fight.Unit("f")
	if !ok {
		t.Fatal("the roster has no target")
	}
	return unit
}

// TestANegativeShareIsAVulnerability is the feature, measured at the only place
// it is observable: the chance actually rolled.
//
// It borrows amplify_test's tainter because `taint` inflicts on four hundred
// rather than a full thousand — a vulnerability measured against a certainty
// would be measuring the clamp — and hangs the trait on the target, which is
// whose business a resistance is.
//
// A trait that refuses -500 lets 1500 through, so a skill declaring 400 rolls
// 600 against its holder. The share is reported as a negative refusal, which is
// the decision this feature turned on: Refused is the amount the target took off
// the chance, and a target that put some on took off a negative.
func TestANegativeShareIsAVulnerability(t *testing.T) {
	bare := chanceRolled(t, taints(t, tainter(t, nil, nil), "taint"))
	if bare.Chance != 400 {
		t.Fatalf("the skill rolled %d against an untraited target, want its declared 400", bare.Chance)
	}

	got := chanceRolled(t, taints(t, tainter(t, nil, []string{"thin_blood"}), "taint"))
	if got.Chance != 600 {
		t.Errorf("a -500 share rolled %d, want 400 raised by half to 600", got.Chance)
	}
	if got.Refused != -500 {
		t.Errorf("the event reports %d refused, want the share as a negative", got.Refused)
	}
}

// TestAVulnerabilityAndAResistanceDoNotCancel is the arithmetic somebody will
// expect to be addition and is not.
//
// Chances compose by multiplying what each lets through, so a -500 and a +600
// leave 1500*400/1000 = 600 surviving rather than nothing. Written down because
// "they cancel out" is the natural reading and is wrong.
func TestAVulnerabilityAndAResistanceDoNotCancel(t *testing.T) {
	got := chanceRolled(t, taints(t,
		tainter(t, nil, []string{"thin_blood", "cold_blood"}), "taint"))
	if got.Chance == 400 {
		t.Fatal("the pair left the chance untouched, so they were read as cancelling")
	}
	if got.Chance != 240 || got.Refused != 400 {
		t.Errorf("the pair rolled %d with %d refused, want 240 and 400", got.Chance, got.Refused)
	}
}

// TestAVulnerabilityCannotPushAChancePastCertainty is the clamp, which lives at
// the call site rather than inside resist so that the two sides compose in any
// order.
func TestAVulnerabilityCannotPushAChancePastCertainty(t *testing.T) {
	// envenom declares the full thousand, so a vulnerability has nowhere to go.
	got := chanceRolled(t, taints(t, tainter(t, nil, []string{"thin_blood"}), "envenom"))
	if got.Chance != 1000 {
		t.Errorf("a certain application rolled %d, want it clamped to certainty", got.Chance)
	}
	if got.Refused != -500 {
		t.Errorf("the event reports %d refused; the clamp is on the chance, not on the share", got.Refused)
	}
}
