package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// The frozen worth of one stack of mending, from the fixture book: attack 800
// against a tick power of 400. Written out rather than computed so a change to
// either number has to be looked at rather than followed.
const oneStackOfMending = 320

// mender is a battle where one unit does nothing but heal itself, and the other
// is slow enough that it never gets in the way.
//
// Defence is a parameter because it is the thing a regeneration must not read:
// the two callers that pass different ones are asserting that nothing moves.
func mender(t *testing.T, defense int64) *battle.Battle {
	t.Helper()
	return mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, defense, 200),
			Skills: []string{"bloom", "dew", "jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab", "sap"}},
	})
}

// blooms hurts the healer, gives it its turn, and returns what the cast logged.
// The wound is what makes the healing measurable: heal stops at full health, so
// a unit at its maximum would produce a log with nothing in it and a test that
// passed on an engine which had healed nothing at all.
func blooms(t *testing.T, fight *battle.Battle, using string, hp int64) []battle.Event {
	t.Helper()
	fight.Begin()
	fight.Drain()
	unitByID(t, fight, "a").HP = hp
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act(using, unitByID(t, fight, "a").Cell); err != nil {
		t.Fatalf("act %s: %v", using, err)
	}
	return fight.Drain()
}

// nextHeal drives the battle on until the healer's regeneration ticks, and
// returns that event.
func nextHeal(t *testing.T, fight *battle.Battle) battle.Event {
	t.Helper()
	for range 20 {
		if !anyTurn(t, fight) {
			break
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Healed && event.Actor == "a" {
				return event
			}
		}
	}
	t.Fatal("the regeneration never healed anybody")
	return battle.Event{}
}

// TestAnAppliedRegenerationHealsEveryTurn is the bug this file was written for.
//
// regrowth was declared, glossed and described, and inert: inflict computed a
// tick only for a damage-over-time, so a regeneration stack was applied carrying
// nought, Set.Tick added nought to the healing total, and the whole downstream
// path — which was already written and already correct — skipped it. Two skills
// in the shipped book did nothing at all when cast, and nothing in this package
// touched a regeneration, which is how it survived the healing work, the
// amplifier work and the drain work untouched.
func TestAnAppliedRegenerationHealsEveryTurn(t *testing.T) {
	fight := mender(t, 400)
	applied := find(blooms(t, fight, "bloom", 900), battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("the cast applied %d regenerations, want one", len(applied))
	}
	// Per stack, which is what a stack freezes and what the applied event
	// carries. The tick below carries the sum, and the two are different
	// numbers on purpose.
	if applied[0].Amount != oneStackOfMending {
		t.Errorf("a stack froze %d, want %d — attack 800 at a tick power of 400",
			applied[0].Amount, oneStackOfMending)
	}
	if applied[0].Stacks != 2 {
		t.Fatalf("the cast applied %d stacks, want the skill's own two", applied[0].Stacks)
	}

	healed := nextHeal(t, fight)
	if want := int64(2 * oneStackOfMending); healed.Amount != want {
		t.Errorf("two stacks healed %d, want %d", healed.Amount, want)
	}
	// The status is on the event because a reader seeing health go up has no
	// other way to learn what put it there — a regeneration still on the board
	// and a cast that is already over are the same number otherwise.
	if healed.Status != "mending" {
		t.Errorf("the heal names %q, want the regeneration that caused it", healed.Status)
	}
	if want := int64(900 + 2*oneStackOfMending); healed.Remaining != want {
		t.Errorf("the healer is on %d health, want %d", healed.Remaining, want)
	}
}

// TestARegenerationIsOneEventAndNotTwo guards the shape of the log rather than
// the arithmetic.
//
// The per-status loop names what healed and heal reports what landed, and
// before this feature worked both fired: one Healed from the loop and a second
// from the total underneath it, for a single tick. Nothing caught it because no
// regeneration had ever ticked. A reader adding the amounts up would have had
// every regeneration in the game worth twice what it is.
func TestARegenerationIsOneEventAndNotTwo(t *testing.T) {
	fight := mender(t, 400)
	blooms(t, fight, "bloom", 900)
	if !anyTurn(t, fight) {
		t.Fatal("the healer never got a second turn")
	}
	heals := find(fight.Drain(), battle.Healed)
	if len(heals) != 1 {
		t.Fatalf("one tick of one regeneration logged %d heals, want one: %+v", len(heals), heals)
	}
}

// TestARegenerationSkipsTheDefenceCurve is the first of the two things Restore
// drops that Damage keeps.
//
// combat.Rules.Restore records the reason: defence turns away what is coming at
// a unit and has nothing to do with what is helping it, so dividing here would
// make a unit's own armour quietly weaken its own regeneration. Twice the
// defence, and the number must not move.
func TestARegenerationSkipsTheDefenceCurve(t *testing.T) {
	bare := nextHealAfterBloom(t, mender(t, 400))
	armoured := nextHealAfterBloom(t, mender(t, 800))
	if bare != armoured {
		t.Errorf("a regeneration healed %d at 400 defence and %d at 800; armour is being read",
			bare, armoured)
	}
	if want := int64(2 * oneStackOfMending); bare != want {
		t.Errorf("the regeneration healed %d, want %d — a control, so that "+
			"two equal wrong numbers cannot pass this test", bare, want)
	}
}

func nextHealAfterBloom(t *testing.T, fight *battle.Battle) int64 {
	t.Helper()
	blooms(t, fight, "bloom", 900)
	return nextHeal(t, fight).Amount
}

// TestARegenerationIgnoresTheElementalChart is the second thing Restore drops.
//
// The chart prices what one creature threw at another. A grass unit healing a
// fire ally is not throwing anything at it, so reading the chart would make the
// same cast on the same ally worth two thirds of itself for a reason written on
// neither of them — and it takes an ally to show, because a unit may only carry
// skills of its own element, so a self-cast can never be off-chart.
func TestARegenerationIgnoresTheElementalChart(t *testing.T) {
	blessed := func(affinity string) int64 {
		t.Helper()
		fight := mustBattle(t, books(t), 5, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("grass"), Stats: stats(3000, 800, 400, 200),
				Skills: []string{"bless", "jab"}},
			{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
				Affinity: single(affinity), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"jab"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
				Skills: []string{"jab"}},
		})
		fight.Begin()
		fight.Drain()
		ally := unitByID(t, fight, "b")
		ally.HP = 900
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "a" {
			t.Fatalf("the first turn went to %s", prompt.Unit)
		}
		if err := fight.Act("bless", ally.Cell); err != nil {
			t.Fatalf("bless: %v", err)
		}
		applied := find(fight.Drain(), battle.StatusApplied)
		if len(applied) != 1 {
			t.Fatalf("the blessing applied %d regenerations, want one", len(applied))
		}
		return applied[0].Amount
	}

	// Fire beats grass on the organic cycle, so a chart being read here would
	// price this cast at two thirds.
	if onFire, onGrass := blessed("fire"), blessed("grass"); onFire != onGrass {
		t.Errorf("the same blessing froze %d on a fire ally and %d on a grass one; "+
			"the chart is being read", onFire, onGrass)
	}
	if got := blessed("fire"); got != oneStackOfMending {
		t.Errorf("the blessing froze %d, want %d — the control", got, oneStackOfMending)
	}
}

// TestARegenerationsWorthIsFrozenWhenItIsApplied is the promise status.Regen
// already made and nothing was honouring: the amount is snapshotted at the
// moment the stack lands, exactly as a poison's tick is.
//
// It matters because the alternative reads the caster's attack every turn, and
// then a debuff landing after the cast would quietly reach back and shrink a
// regeneration that was already paid for.
func TestARegenerationsWorthIsFrozenWhenItIsApplied(t *testing.T) {
	fight := mender(t, 400)
	blooms(t, fight, "bloom", 900)

	// Weaken the healer after the stacks are on it. Applied directly rather than
	// cast, because what is being measured is the stack that already exists and
	// not the turn order that would deliver the debuff.
	weaken, err := fight.Books().Statuses.Lookup("weaken")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	healer := unitByID(t, fight, "a")
	healer.Statuses.Apply(weaken, 0)
	if fight.Stats(healer)[progression.Attack] >= 800 {
		t.Fatalf("the debuff left attack at %d, so this test measures nothing",
			fight.Stats(healer)[progression.Attack])
	}

	if healed := nextHeal(t, fight).Amount; healed != 2*oneStackOfMending {
		t.Errorf("a regeneration applied at full attack healed %d after a debuff, want %d",
			healed, 2*oneStackOfMending)
	}
}

// TestEachApplicationFreezesItsOwnWorth is the same promise across two casts.
//
// status.Regen puts it as two casters each contributing what their own attack
// was worth; one caster at two different attacks is the same mechanism and the
// arrangement a test can drive. The sum is what proves it: an engine freezing
// once and reusing it would give twice the first figure, and one reading attack
// live would give three times the second.
func TestEachApplicationFreezesItsOwnWorth(t *testing.T) {
	fight := mender(t, 400)
	blooms(t, fight, "bloom", 900)

	weaken, err := fight.Books().Statuses.Lookup("weaken")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	healer := unitByID(t, fight, "a")
	healer.Statuses.Apply(weaken, 0)
	weakened := fight.Stats(healer)[progression.Attack]

	// The third stack goes on at the reduced attack.
	if !anyTurnUsing(t, fight, "dew") {
		t.Fatal("the healer never got a turn to cast the third stack")
	}
	applied := find(fight.Drain(), battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("the second cast applied %d regenerations, want one", len(applied))
	}
	third := applied[0].Amount
	if want := weakened * 400 / 1000; third != want {
		t.Fatalf("the third stack froze %d, want %d — attack %d at a tick power of 400",
			third, want, weakened)
	}

	healed := nextHeal(t, fight).Amount
	if want := 2*int64(oneStackOfMending) + third; healed != want {
		t.Errorf("three stacks frozen at two different attacks healed %d, want %d "+
			"(%d + %d + %d)", healed, want, oneStackOfMending, oneStackOfMending, third)
	}
}

// anyTurnUsing gives the healer a turn with a named skill, skipping over anybody
// else's turn on the way.
func anyTurnUsing(t *testing.T, fight *battle.Battle, using string) bool {
	t.Helper()
	for range 20 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			return false
		}
		if prompt.Skipped || prompt.Unit != "a" {
			continue
		}
		if err := fight.Act(using, unitByID(t, fight, "a").Cell); err != nil {
			t.Fatalf("act %s: %v", using, err)
		}
		return true
	}
	return false
}

// TestARegenerationStopsAtFullHealthAndSaysWhatLanded is the clamp, and it is
// the reason healing is resolved one status at a time rather than from a total.
//
// heal stops at the maximum, so the last tick of a regeneration on a nearly full
// unit is worth less than the stacks say. Resolving from a total would have left
// the log claiming the full figure with only part of it landing.
func TestARegenerationStopsAtFullHealthAndSaysWhatLanded(t *testing.T) {
	fight := mender(t, 400)
	blooms(t, fight, "bloom", 2900)

	healed := nextHeal(t, fight)
	if healed.Amount != 100 {
		t.Errorf("a unit a hundred short of full healed %d, want the hundred that fits",
			healed.Amount)
	}
	if healed.Remaining != 3000 {
		t.Errorf("the healer ended on %d, want its maximum of 3000", healed.Remaining)
	}
}

// TestHealingLandsBeforeDamageInTheSameTick is the order two totals resolve in,
// and it decides who lives: a regeneration that would carry a unit past a poison
// tick has to do so, rather than the order of two numbers settling it.
//
// It was commented in tickStatuses before this and could not be tested, because
// the healing half of it never happened.
//
// ⚠️ It asserts survival rather than the order of the events, and the first
// draft of it asserted the order and was worthless. wound emits nothing — the
// status_ticked beside it comes from the loop above, which runs either way — so
// swapping the two leaves the log in exactly the same order and changes only who
// is standing at the end of it. A mutation putting damage first survived the
// event-order version and is caught by this one.
func TestHealingLandsBeforeDamageInTheSameTick(t *testing.T) {
	fight := mender(t, 400)
	// A hundred and fifty health against a poison worth two hundred: the tick
	// kills outright if it lands first, and is survived with room to spare if
	// the regeneration goes first.
	blooms(t, fight, "bloom", 150)

	poison, err := fight.Books().Statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	unitByID(t, fight, "a").Statuses.Apply(poison, 200)

	anyTurn(t, fight)
	healer := unitByID(t, fight, "a")
	if healer.Dead {
		t.Fatal("the poison killed the healer, so the damage was taken before the healing")
	}
	if want := int64(150 + 2*oneStackOfMending - 200); healer.HP != want {
		t.Errorf("the healer is on %d health, want %d — healed for %d, then poisoned for 200",
			healer.HP, want, 2*oneStackOfMending)
	}
}
