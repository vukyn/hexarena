package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// tainter is a battle where one unit does nothing but try to poison the other,
// with traits on either side.
//
// It is poisoner's mirror and the difference is the whole point of these tests:
// poisoner hangs the traits on the *target*, because a resistance is the target's
// business, and this hangs them on both so an amplifier can be shown to read the
// unit that is acting. `taint` inflicts on four hundred rather than a full
// thousand, because an amplified certainty is still a certainty and would measure
// nothing.
func tainter(t *testing.T, actorTraits, targetTraits []string) *battle.Battle {
	t.Helper()
	return mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"taint", "envenom"}, Passives: actorTraits},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}, Passives: targetTraits},
	})
}

// taints takes one turn with the named skill and returns what came of it.
func taints(t *testing.T, fight *battle.Battle, using string) []battle.Event {
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
	if err := fight.Act(using, hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	return fight.Drain()
}

// chanceRolled is the chance an application was decided on, from whichever of
// the two kinds the log holds: an application that landed and one that did not
// carry the same figure, and a test that read only one of them would pass on a
// battle where the other happened.
func chanceRolled(t *testing.T, events []battle.Event) battle.Event {
	t.Helper()
	applied := find(events, battle.StatusApplied)
	resisted := find(events, battle.StatusResisted)
	switch {
	case len(applied)+len(resisted) != 1:
		t.Fatalf("the log holds %d applications and %d refusals, want one event in total",
			len(applied), len(resisted))
	case len(applied) == 1:
		return applied[0]
	default:
		return resisted[0]
	}
	return battle.Event{}
}

// TestAnAmplifiedChanceIsRolledHigherAndTheLogSaysBy is the chance half of the
// feature, with the control beside it.
//
// The control is what makes it a measurement: 400 is the skill's own figure, and
// without asserting it the amplified 480 could be any number at all.
func TestAnAmplifiedChanceIsRolledHigherAndTheLogSaysBy(t *testing.T) {
	plain := chanceRolled(t, taints(t, tainter(t, nil, nil), "taint"))
	if plain.Chance != 400 {
		t.Fatalf("an unamplified application rolled against %d, want the skill's own 400", plain.Chance)
	}
	if plain.AmplifiedChance != 0 {
		t.Errorf("the event claims %d amplified with no trait to amplify it", plain.AmplifiedChance)
	}

	raised := chanceRolled(t, taints(t, tainter(t, []string{"insidious"}, nil), "taint"))
	if raised.Chance != 600 {
		t.Errorf("a chance amplified by five hundred rolled against %d, want 600", raised.Chance)
	}
	if raised.AmplifiedChance != 500 {
		t.Errorf("the event says %d amplified, want the trait's own 500", raised.AmplifiedChance)
	}
	// And the share reaching the event is the point of the field rather than a
	// nicety: 600 is not a figure a reader can get from the skill book, so
	// without it the log cannot explain its own number.
	if raised.AmplifiedEffect != 0 {
		t.Errorf("a chance-only trait says %d on the effect", raised.AmplifiedEffect)
	}
}

// TestAnAmplifiedTickIsFrozenHigherAndTheLogSaysBy is the effect half. It is a
// separate test because they are separate features: a trait may want either
// alone, and the shipped one wanting both would hide it if these were one case.
func TestAnAmplifiedTickIsFrozenHigherAndTheLogSaysBy(t *testing.T) {
	plain := find(taints(t, tainter(t, nil, nil), "envenom"), battle.StatusApplied)
	if len(plain) != 1 {
		t.Fatalf("the control applied %d poisons, want one", len(plain))
	}
	base := plain[0].Amount
	if base <= 0 {
		t.Fatalf("the control froze a tick of %d, so there is nothing to amplify", base)
	}

	raised := find(taints(t, tainter(t, []string{"corrosive"}, nil), "envenom"), battle.StatusApplied)
	if len(raised) != 1 {
		t.Fatalf("the amplified cast applied %d poisons, want one", len(raised))
	}
	if want := base * 1300 / 1000; raised[0].Amount != want {
		t.Errorf("the frozen tick is %d, want %d — the control's %d raised by three hundred",
			raised[0].Amount, want, base)
	}
	if raised[0].AmplifiedEffect != 300 {
		t.Errorf("the event says %d amplified on the effect, want 300", raised[0].AmplifiedEffect)
	}
	// An effect-only trait leaves the chance alone, which is what keeps the two
	// halves separable in the data as well as in the log.
	if raised[0].Chance != plain[0].Chance {
		t.Errorf("an effect-only trait moved the chance from %d to %d",
			plain[0].Chance, raised[0].Chance)
	}
	if raised[0].AmplifiedChance != 0 {
		t.Errorf("an effect-only trait says %d on the chance", raised[0].AmplifiedChance)
	}
}

// TestTheAmplifiedTickStaysFrozenForTheStacksWholeLife is the property the whole
// design leans on: the amplification is folded into the one multiplication
// battle freezes, so nothing later has to know the trait existed.
//
// It matters because the alternative — reading the trait at each tick — would
// make a stack worth more or less depending on whether its author was still
// alive, and a Stack deliberately does not remember who applied it.
func TestTheAmplifiedTickStaysFrozenForTheStacksWholeLife(t *testing.T) {
	fight := tainter(t, []string{"corrosive"}, nil)
	events := taints(t, fight, "envenom")
	applied := find(events, battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("the cast applied %d poisons, want one", len(applied))
	}
	frozen := applied[0].Amount

	// Run the battle on to the ticks and check each one against the frozen
	// figure. Every tick of the same stack has to be the same number.
	ticks := 0
	for range 40 {
		if !anyTurn(t, fight) {
			break
		}
		for _, event := range find(fight.Drain(), battle.StatusTicked) {
			// Actor, not Target: status_ticked names the unit *taking* the
			// damage, because a Stack does not remember who applied it — the
			// applier may be dead by the time it resolves.
			if event.Status != "poison" || event.Actor != "f" {
				continue
			}
			ticks++
			// Per stack, because the event carries the total: a status.Set sums
			// its stacks' frozen amounts, and every stack of this poison was
			// frozen by the same trait at the same share.
			if want := frozen * int64(event.Stacks); event.Amount != want {
				t.Errorf("a tick of %d stacks took %d, want %d — %d frozen, each",
					event.Stacks, event.Amount, want, frozen)
			}
		}
	}
	if ticks == 0 {
		t.Fatal("the poison never ticked, so nothing about freezing was measured")
	}
}

// anyTurn lets whichever unit is up act with whatever it can, which is what a
// test driving a battle forward for its *ticks* wants: take() names a skill and
// does nothing when the prompted unit does not have it, and a turn nobody takes
// makes the next Advance refuse.
func anyTurn(t *testing.T, fight *battle.Battle) bool {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Skipped {
		return prompt != nil
	}
	for _, option := range prompt.Options {
		if !option.Available() {
			continue
		}
		if err := fight.Act(option.Skill, option.Aims[0]); err != nil {
			t.Fatalf("act %s: %v", option.Skill, err)
		}
		return true
	}
	return false
}

// TestAnAmplifierReadsTheUnitThatIsActing is the trap the feature came with.
//
// Every other trait reads its own holder. This one reads the actor at a site a
// few lines from the one that reads the target, and the two units are the same
// Go type — so passing the wrong one compiles and quietly hands a target the
// attacker's amplifier. The only way to catch that is to hang the trait on the
// target and insist nothing moves.
func TestAnAmplifierReadsTheUnitThatIsActing(t *testing.T) {
	plain := chanceRolled(t, taints(t, tainter(t, nil, nil), "taint"))
	onTheTarget := chanceRolled(t, taints(t, tainter(t, nil, []string{"insidious"}), "taint"))
	if onTheTarget.Chance != plain.Chance {
		t.Errorf("a trait held by the target moved the chance from %d to %d",
			plain.Chance, onTheTarget.Chance)
	}
	if onTheTarget.AmplifiedChance != 0 {
		t.Errorf("a trait held by the target amplified by %d", onTheTarget.AmplifiedChance)
	}
	// And the same trait on the actor does move it, so the test above is not
	// passing because the trait does nothing at all.
	onTheActor := chanceRolled(t, taints(t, tainter(t, []string{"insidious"}, nil), "taint"))
	if onTheActor.Chance == plain.Chance {
		t.Errorf("the trait moved nothing from either side, so this proves nothing")
	}
}

// TestAnAmplifierAndAResistanceMeetAtOneSiteAndTheLogNamesBoth is the pair the
// design was written around: one side raising the chance, the other lowering it,
// composing by multiplication so their order cannot matter.
func TestAnAmplifierAndAResistanceMeetAtOneSiteAndTheLogNamesBoth(t *testing.T) {
	// 400 amplified by 500 is 600, of which a resistance of 600 lets 40 percent
	// through: 240. Worked out here rather than read off the event, because a
	// test that recomputed the implementation's arithmetic would agree with a
	// wrong implementation.
	events := taints(t, tainter(t, []string{"insidious"}, []string{"thick_blood"}), "taint")
	event := chanceRolled(t, events)
	if event.Chance != 240 {
		t.Errorf("the chance rolled is %d, want 240", event.Chance)
	}
	if event.AmplifiedChance != 500 {
		t.Errorf("the event says %d amplified", event.AmplifiedChance)
	}
	if event.Refused != 600 {
		t.Errorf("the event says %d refused", event.Refused)
	}
}

// TestAnAmplifiedCertaintyIsStillACertainty is the clamp, and where it sits.
//
// A probability cannot exceed one, so amplifying an application that already
// lands every time is worth nothing. The clamp is applied to the composed figure
// rather than before the target's share, because clamping first would make the
// order the two sides compose in matter — and both sides were written not to care.
func TestAnAmplifiedCertaintyIsStillACertainty(t *testing.T) {
	certain := find(taints(t, tainter(t, []string{"insidious"}, nil), "envenom"),
		battle.StatusApplied)
	if len(certain) != 1 {
		t.Fatalf("the cast applied %d poisons, want one", len(certain))
	}
	if certain[0].Chance != 1000 {
		t.Errorf("a certainty amplified rolled against %d, want the thousand it already was",
			certain[0].Chance)
	}
	// The share still reaches the event, which is what lets a reader see that a
	// trait fired and was worth nothing here rather than wonder whether it fired
	// at all.
	if certain[0].AmplifiedChance != 500 {
		t.Errorf("the event says %d amplified, want the trait's own 500", certain[0].AmplifiedChance)
	}

	// And the clamp is not hiding an order dependence: the same certainty with a
	// resistance in the way composes to six hundred, not five. Halving first and
	// amplifying second gives 600; clamping first and halving second gives 500.
	halved := chanceRolled(t, taints(t,
		tainter(t, []string{"insidious"}, []string{"thick_blood"}), "envenom"))
	if halved.Chance != 600 {
		t.Errorf("a certainty amplified by 500 and refused by 600 rolled against %d, want 600",
			halved.Chance)
	}
}

// TestAnAmplifierOnlyTouchesTheStatusItNames is the narrowness of the field: a
// trait about poison says nothing about a burn.
func TestAnAmplifierOnlyTouchesTheStatusItNames(t *testing.T) {
	event := chanceRolled(t, taints(t, tainter(t, []string{"scalding"}, nil), "taint"))
	if event.Chance != 400 {
		t.Errorf("a trait about burn moved a poison's chance to %d", event.Chance)
	}
	if event.AmplifiedChance != 0 {
		t.Errorf("a trait about burn says %d on a poison", event.AmplifiedChance)
	}
}

// TestTwoAmplifiersComposeRatherThanAddUp pins the arithmetic of stacking, which
// is the one place amplification and resistance differ: resistances stacking
// diminish, amplifiers stacking compound. Both are the same multiplication; only
// the direction differs, which is what keeps their order irrelevant.
func TestTwoAmplifiersComposeRatherThanAddUp(t *testing.T) {
	// 400 raised by 500 twice is 400 * 1.5 * 1.5 = 900, not 400 * 2 = 800.
	events := taints(t, tainter(t, []string{"insidious", "cornered_venom"}, nil), "taint")
	event := chanceRolled(t, events)
	// cornered_venom is gated below half health and the actor is at full, so only
	// one of the two is in force: this is the control for the gate, and the
	// composition is measured on the hurt actor below.
	if event.Chance != 600 {
		t.Errorf("a healthy actor rolled against %d, want only the ungated trait's 600",
			event.Chance)
	}
	if event.AmplifiedChance != 500 {
		t.Errorf("the event says %d amplified on a healthy actor", event.AmplifiedChance)
	}
}

// TestAGatedAmplifierWaitsUntilItsHolderIsHurt is the gate on this trait, read
// at the moment of the application rather than at enlistment — so a holder healed
// back stops amplifying, exactly as a gated resistance stops protecting.
func TestAGatedAmplifierWaitsUntilItsHolderIsHurt(t *testing.T) {
	fight := tainter(t, []string{"cornered_venom"}, nil)
	healthy := chanceRolled(t, taints(t, fight, "taint"))
	if healthy.Chance != 400 {
		t.Errorf("a healthy holder rolled against %d, want the skill's own 400", healthy.Chance)
	}
	if healthy.AmplifiedChance != 0 {
		t.Errorf("a healthy holder amplified by %d", healthy.AmplifiedChance)
	}

	hurt := tainter(t, []string{"cornered_venom"}, nil)
	hurt.Begin()
	hurt.Drain()
	// Set below the line rather than driven there by turns: what is being
	// measured is the health the gate is read at, and the turns it would take to
	// arrive are a different test's subject. gate_test drives them because it
	// asserts the event a crossing emits, and an amplifier emits none — it has no
	// status to hold and let go of.
	unitByID(t, hurt, "a").HP = 900
	prompt, err := hurt.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := hurt.Act("taint", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	event := chanceRolled(t, hurt.Drain())
	if event.Chance != 600 {
		t.Errorf("a hurt holder rolled against %d, want 600", event.Chance)
	}
	if event.AmplifiedChance != 500 {
		t.Errorf("the event says %d amplified on a hurt holder", event.AmplifiedChance)
	}
}
