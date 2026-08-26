package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// duelWith is a one-on-one where the ally holds traits and knows one skill, so a
// single turn is a single measurement.
func duelWith(t *testing.T, skillID string, traits ...string) *battle.Battle {
	t.Helper()
	return mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{skillID}, Passives: traits},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}},
	})
}

// act takes the ally's turn with the named skill and hands back what came of it.
func act(t *testing.T, fight *battle.Battle, skillID string, aim hex.Offset) []battle.Event {
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
	if err := fight.Act(skillID, aim); err != nil {
		t.Fatalf("act: %v", err)
	}
	return fight.Drain()
}

var acrossTheBoard = hex.Offset{Col: 3, Row: 1}

// TestATraitAddsItsOwnApplicationToAnAttack is the third of the four things a
// passive does: an effect on top of what the unit already does.
//
// strike inflicts nothing of its own, so the poison in the log is the trait's and
// could not be anything else.
func TestATraitAddsItsOwnApplicationToAnAttack(t *testing.T) {
	events := act(t, duelWith(t, "strike", "venomous"), "strike", acrossTheBoard)
	applied := find(events, battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("a venomous unit's attack applied %d statuses, want 1: %+v", len(applied), applied)
	}
	if applied[0].Status != "poison" {
		t.Errorf("the trait applied %q", applied[0].Status)
	}
	if applied[0].Target != "f" {
		t.Errorf("the trait's status landed on %q", applied[0].Target)
	}
	// The control: the same skill without the trait inflicts nothing at all.
	plain := act(t, duelWith(t, "strike"), "strike", acrossTheBoard)
	if applied := find(plain, battle.StatusApplied); len(applied) != 0 {
		t.Errorf("strike inflicted %+v on its own, so the test above proves nothing", applied)
	}
}

// TestATraitsApplicationRidesOnlyOnADamagingSkill is the rule that keeps a
// hostile rider off a friendly skill.
//
// mend is aimed at an ally and deals no damage, so a trait that rode along
// anyway would poison the unit it was cleansing. resolveAgainst deliberately
// never asks which side a target is on, so "deals damage to it" is the available
// way to say the skill is an attack.
//
// It has to be mend rather than a self-shield: a skill aimed at the caster
// returns before resolveAgainst is reached at all, so brace could never carry a
// rider whatever the rule said — and that was the first version of this test,
// which passed with the rule deleted.
func TestATraitsApplicationRidesOnlyOnADamagingSkill(t *testing.T) {
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		// The healer stands behind the ally it heals: every unit has to be able to
		// reach somebody or enlistment refuses the roster, and a range-1 cleanse
		// reaches an adjacent ally while a range-1 attack has to reach the enemy.
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"mend"}, Passives: []string{"venomous"}},
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"jab"}},
	})
	// The ally's own cell, taken from the board rather than worked out here.
	mate, ok := fight.Unit("b")
	if !ok {
		t.Fatal("the roster has no second ally")
	}
	events := act(t, fight, "mend", mate.Cell)
	for _, event := range find(events, battle.StatusApplied) {
		if event.Status == "poison" {
			t.Errorf("a cleanse carried the trait's poison to %q", event.Target)
		}
	}
	// And the skill did reach the ally, or the absence above proves nothing.
	if used := find(events, battle.SkillUsed); len(used) != 1 {
		t.Fatalf("the cleanse was not cast: %+v", events)
	}
}

// TestARiderGoesThroughTheSameRefusalAsASkillsOwnApplication is why the trait
// contributes to the same list instead of getting a pass of its own: a second
// path would be a second place for the roll, the resistance and the event to be
// got wrong.
func TestARiderGoesThroughTheSameRefusalAsASkillsOwnApplication(t *testing.T) {
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"strike"}, Passives: []string{"venomous"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}, Passives: []string{"clean_blood"}},
	})
	events := act(t, fight, "strike", acrossTheBoard)
	if applied := find(events, battle.StatusApplied); len(applied) != 0 {
		t.Errorf("a poison-immune target took the rider anyway: %+v", applied)
	}
	refused := find(events, battle.StatusResisted)
	if len(refused) != 1 {
		t.Fatalf("the log holds %d refusals, want the rider's one", len(refused))
	}
	if refused[0].Refused != 1000 {
		t.Errorf("the rider was refused at %d, want the immunity's thousand", refused[0].Refused)
	}
}

// TestAGatedTraitOnlyActsWhileItsConditionHolds is the fourth job: a trait that
// waits until its holder is hurt.
//
// The two ends of the gate are measured on the same battle rather than on two,
// because what is being checked is that the condition is read *live* — a trait
// read once at enlistment would answer the same both times.
func TestAGatedTraitOnlyActsWhileItsConditionHolds(t *testing.T) {
	// At full health the gate is shut and the attack inflicts nothing.
	whole := act(t, duelWith(t, "strike", "cornered"), "strike", acrossTheBoard)
	for _, event := range find(whole, battle.StatusApplied) {
		if event.Status == "burn" {
			t.Errorf("a trait gated below half health acted at full health")
		}
	}

	// Hurt to the threshold exactly, which is where an off-by-one lives: the
	// condition is "at or under", so half counts.
	fight := duelWith(t, "strike", "cornered")
	holder, ok := fight.Unit("a")
	if !ok {
		t.Fatal("the roster has no holder")
	}
	holder.HP = holder.MaxHP() / 2
	events := act(t, fight, "strike", acrossTheBoard)
	burned := false
	for _, event := range find(events, battle.StatusApplied) {
		if event.Status == "burn" {
			burned = true
		}
	}
	if !burned {
		t.Errorf("a trait gated below half health did not act at exactly half:\n%+v",
			find(events, battle.StatusApplied))
	}
}

// TestAGatedResistanceOnlyProtectsWhileItHolds is the same gate on the other
// half of a trait, because one read of the condition per site is one place each
// for it to be forgotten.
func TestAGatedResistanceOnlyProtectsWhileItHolds(t *testing.T) {
	poisonAt := func(health int64) []battle.Event {
		t.Helper()
		fight := mustBattle(t, books(t), 5, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
				Skills: []string{"envenom"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
				Skills: []string{"jab"}, Passives: []string{"last_stand"}},
		})
		if health > 0 {
			unit, ok := fight.Unit("f")
			if !ok {
				t.Fatal("the roster has no target")
			}
			unit.HP = health
		}
		return act(t, fight, "envenom", acrossTheBoard)
	}

	// envenom cannot fail its own roll, so an application that does not land is
	// one the target refused.
	whole := poisonAt(0)
	if applied := find(whole, battle.StatusApplied); len(applied) != 1 {
		t.Errorf("at full health the gated immunity protected anyway: %+v",
			find(whole, battle.StatusResisted))
	}
	hurt := poisonAt(1500)
	if applied := find(hurt, battle.StatusApplied); len(applied) != 0 {
		t.Errorf("at half health the gated immunity did not protect: %+v", applied)
	}
	refused := find(hurt, battle.StatusResisted)
	if len(refused) != 1 || refused[0].Refused != 1000 {
		t.Errorf("the refusal reads %+v, want the immunity's thousand", refused)
	}
}
