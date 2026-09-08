package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// driveTo advances the battle until the ally's health crosses into want, or gives
// up after enough turns to say so.
//
// It exists because every case below is "put the holder on the other side of a
// line and read the board", and the number of turns that takes is a fact about
// the fixture's damage rather than about the gate. It reports whether the
// crossing happened, so a caller can fail with the health it actually reached.
//
// ⚠️ It gives up the moment a named skill could not be used rather than
// advancing again, because take has already opened that turn — a second Advance
// over an unacted turn is an error, and the caller would report it instead of
// reporting the health the holder got to. That matters when a mutation stops the
// gate moving at all: what a reader wants then is "it never crossed, and here is
// where it stopped".
func driveTo(t *testing.T, fight *battle.Battle, ally *battle.Unit, skills []string, want func() bool) bool {
	t.Helper()
	for round := 0; round < 20 && !want(); round++ {
		for _, id := range skills {
			if !take(t, fight, id) {
				return want()
			}
		}
		if ally.Dead {
			break
		}
	}
	return want()
}

// TestAGrantBehindItsOwnGateComesAndGoes is the whole of step 2 in one battle.
//
// ⚠️ **The trait carries no gate of its own**, and that is the discriminating
// part rather than a detail: a fixture that also had a trait-level `while` would
// pass with the per-grant field ignored entirely, since every grant on such a
// trait is gated either way. Only a trait gated *nowhere but on one grant* can
// tell a per-grant gate from nothing at all — which is also why the early return
// in reconsider had to stop reading Passive.While alone.
//
// What is read is both directions and both grants: the gated tier is on at full
// health, lapses when its holder falls under the line, comes back when it climbs
// over, and the ungated tier never moves through any of it.
func TestAGrantBehindItsOwnGateComesAndGoes(t *testing.T) {
	fight := gated(t, "two_tier", []string{"jab", "drink"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")

	// At enlistment the holder is at full health, so a gate at the top of the
	// bar is open and both tiers are on.
	if !ally.Statuses.Has("toughened") {
		t.Fatal("the ungated tier is not on at full health, so the trait grants nothing at all")
	}
	if !ally.Statuses.Has("fortified") {
		t.Fatal("a grant gated above 70% health is off at full health, so its gate reads the wrong end")
	}
	opening := 0
	for _, event := range find(fight.Drain(), battle.PassiveHeld) {
		if event.Actor == "a" && event.Passive == "two_tier" {
			opening++
		}
	}
	if opening != 2 {
		t.Errorf("the opening board announces %d of the trait's grants, want both", opening)
	}

	// Hurt it under the line. The gated tier goes and the ungated one stays.
	if !driveTo(t, fight, ally, []string{"jab", "strike"},
		func() bool { return !ally.Statuses.Has("fortified") }) {
		t.Fatalf("the gated tier never lapsed; the holder is at %d of %d",
			ally.HP, fight.MaxHP(ally))
	}
	if !ally.Statuses.Has("toughened") {
		t.Error("the ungated tier went off with the gated one, so the gate moves the whole trait")
	}
	released := 0
	for _, event := range find(fight.Drain(), battle.PassiveReleased) {
		if event.Actor != "a" {
			continue
		}
		released++
		if event.Status != "fortified" {
			t.Errorf("the crossing released %q, want only the gated tier", event.Status)
		}
	}
	if released != 1 {
		t.Errorf("the crossing produced %d release events, want exactly one", released)
	}

	// And back over it under its own power, because a gate that only shuts is a
	// one way door — the failure the grant needed Hold and Release for.
	if !driveTo(t, fight, ally, []string{"drink"},
		func() bool { return ally.Statuses.Has("fortified") }) {
		t.Fatalf("the gated tier never came back; the holder is at %d of %d",
			ally.HP, fight.MaxHP(ally))
	}
	held := 0
	for _, event := range find(fight.Drain(), battle.PassiveHeld) {
		if event.Actor != "a" {
			continue
		}
		held++
		if event.Status != "fortified" {
			t.Errorf("the crossing back held %q, want only the gated tier", event.Status)
		}
	}
	if held != 1 {
		t.Errorf("crossing back produced %d hold events, want exactly one", held)
	}
	if !ally.Statuses.Has("toughened") {
		t.Error("the ungated tier is gone after two crossings, so something released it")
	}
}

// TestTheTwoTierShapeIsWorthBothTiersAndThenOne is the shape end to end, read on
// the stat rather than on the status flags.
//
// The claim is the one the feature exists for: X always and Z as well while
// above Y, with the ungated tier surviving the gated one lapsing. Three battles
// name the three figures — nothing, the first tier alone, and both — because a
// single reading cannot say whether the second tier was worth anything, and the
// figure after the lapse has to come back to the *first tier's* number rather
// than to the base.
//
// ⚠️ The two tiers do not add up to the sum of their faces: modifier.Set
// saturates a change towards a ceiling rather than applying it, so the second
// tier is worth less than its own percentage. That is why this compares the four
// figures against each other and never against arithmetic.
func TestTheTwoTierShapeIsWorthBothTiersAndThenOne(t *testing.T) {
	defenceUnder := func(trait string) int64 {
		t.Helper()
		var fight *battle.Battle
		if trait == "" {
			fight = gated(t, "first_tier", []string{"jab"}, []string{"strike"})
			// The bare line, read off the enemy: it carries no trait at all and
			// the same base defence, which is the honest way to say "nothing".
			fight.Begin()
			return fight.Stats(unitByID(t, fight, "f"))[progression.Defense]
		}
		fight = gated(t, trait, []string{"jab"}, []string{"strike"})
		fight.Begin()
		return fight.Stats(unitByID(t, fight, "a"))[progression.Defense]
	}
	bare := defenceUnder("")
	one := defenceUnder("first_tier")
	both := defenceUnder("two_tier")
	if one <= bare {
		t.Fatalf("the first tier moved defence %d to %d, so the fixture grants nothing", bare, one)
	}
	if both <= one {
		t.Fatalf("the second tier moved defence %d to %d, so a gated grant is not applied "+
			"at enlistment even with its gate open", one, both)
	}
	t.Logf("defence: %d bare, %d one tier, %d both tiers", bare, one, both)

	// Now wear the holder under the line and read it again.
	fight := gated(t, "two_tier", []string{"jab"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	if got := fight.Stats(ally)[progression.Defense]; got != both {
		t.Fatalf("the two-tier holder opens at %d defence, want the %d measured above", got, both)
	}
	if !driveTo(t, fight, ally, []string{"jab", "strike"},
		func() bool { return !ally.Statuses.Has("fortified") }) {
		t.Fatalf("the gated tier never lapsed; the holder is at %d of %d",
			ally.HP, fight.MaxHP(ally))
	}
	after := fight.Stats(ally)[progression.Defense]
	if after == both {
		t.Errorf("defence is still %d after the gated tier lapsed, so the release moved no stat", after)
	}
	if after != one {
		t.Errorf("defence fell to %d after the gated tier lapsed, want the first tier's own %d "+
			"— the tier that is always on has to survive the other going off", after, one)
	}
	if after <= bare {
		t.Errorf("defence fell to %d, at or under the %d of a unit carrying nothing", after, bare)
	}
}

// TestAGrantGatedAtTheBottomOfTheBarStartsOff is the other polarity at
// enlistment, and it is the arm that says hold reads a gate rather than
// assuming one.
//
// A two-tier trait may be written either way round — a tier that arrives as its
// holder is worn down is as authorable as one that lapses — and enlistment is
// the moment the two differ: one opens on and one opens off, from one rule.
func TestAGrantGatedAtTheBottomOfTheBarStartsOff(t *testing.T) {
	fight := gated(t, "late_tier", []string{"jab"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	if !ally.Statuses.Has("toughened") {
		t.Fatal("the ungated tier is off at full health")
	}
	if ally.Statuses.Has("fortified") {
		t.Fatal("a grant gated below half health is on at full health, so its gate does nothing")
	}
	// And the opening board does not announce it, because announcing a grant the
	// board does not show would describe a different unit from the one drawn.
	for _, event := range find(fight.Drain(), battle.PassiveHeld) {
		if event.Actor == "a" && event.Status == "fortified" {
			t.Error("the opening board announces a gated grant the holder is not carrying")
		}
	}
	if !driveTo(t, fight, ally, []string{"jab", "strike"},
		func() bool { return ally.Statuses.Has("fortified") }) {
		t.Fatalf("the gated tier never came on; the holder is at %d of %d",
			ally.HP, fight.MaxHP(ally))
	}
	if !ally.Statuses.Has("toughened") {
		t.Error("the ungated tier went off when the gated one came on")
	}
}

// TestATraitLevelGateStillMovesEveryGrantTogether is the case that must not have
// moved.
//
// A gate on the *trait* gates the whole of it, so a trait with two grants brings
// both on at one crossing and takes both off at the other. That was true before
// a grant could carry a gate of its own and has to stay true: the per-grant field
// is an addition, not a replacement, and a version of reconsider that only ever
// looked at Grant.While would leave a trait-level gated trait frozen at whatever
// enlistment put it at.
//
// ⚠️ Two grants rather than one, because every trait-level gated fixture in this
// package grants a single status — and a trait with one grant cannot tell "moves
// the whole trait" from "moves this grant".
func TestATraitLevelGateStillMovesEveryGrantTogether(t *testing.T) {
	fight := gated(t, "dug_in_deep", []string{"jab", "drink"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	for _, id := range []string{"toughened", "fortified"} {
		if ally.Statuses.Has(id) {
			t.Fatalf("a trait gated below half health carries %q at full health", id)
		}
	}
	fight.Drain()

	if !driveTo(t, fight, ally, []string{"jab", "strike"},
		func() bool { return ally.Statuses.Has("toughened") }) {
		t.Fatalf("the trait never came on; the holder is at %d of %d", ally.HP, fight.MaxHP(ally))
	}
	if !ally.Statuses.Has("fortified") {
		t.Error("one grant came on and the other did not, so a trait-level gate has stopped " +
			"moving the whole trait")
	}
	held := map[string]int{}
	for _, event := range find(fight.Drain(), battle.PassiveHeld) {
		if event.Actor == "a" && event.Passive == "dug_in_deep" {
			held[event.Status]++
		}
	}
	if held["toughened"] != 1 || held["fortified"] != 1 {
		t.Errorf("the crossing announced %v, want one event per grant", held)
	}

	// And off again together.
	if !driveTo(t, fight, ally, []string{"drink"},
		func() bool { return !ally.Statuses.Has("toughened") }) {
		t.Fatalf("the trait never went off again; the holder is at %d of %d",
			ally.HP, fight.MaxHP(ally))
	}
	if ally.Statuses.Has("fortified") {
		t.Error("one grant went off and the other stayed, so a trait-level gate is now per-grant")
	}
	released := map[string]int{}
	for _, event := range find(fight.Drain(), battle.PassiveReleased) {
		if event.Actor == "a" && event.Passive == "dug_in_deep" {
			released[event.Status]++
		}
	}
	if released["toughened"] != 1 || released["fortified"] != 1 {
		t.Errorf("the crossing back announced %v, want one event per grant", released)
	}
}
