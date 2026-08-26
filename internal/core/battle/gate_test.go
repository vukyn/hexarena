package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// gated is a duel where the ally holds one trait, so a test can say what the
// trait is and nothing else.
//
// Both sides carry the same modest attack and the ally has room to be hurt in
// stages, because the whole subject here is *when* a line is crossed rather than
// who wins. Speeds are chosen so the ally acts first, which is what lets a test
// hurt it and then read the board on its own turn.
func gated(t *testing.T, trait string, allySkills, foeSkills []string) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2000, 800, 400, 120),
			Skills: allySkills, Passives: []string{trait}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 100),
			Skills: foeSkills},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

func unitByID(t *testing.T, fight *battle.Battle, id string) *battle.Unit {
	t.Helper()
	for _, unit := range fight.Units() {
		if unit.ID == id {
			return unit
		}
	}
	t.Fatalf("no unit %q in the battle", id)
	return nil
}

// take advances to the next prompt and uses the named skill on the first cell it
// offers, which is the shortest way to say "let this unit have its turn" in a
// test that cares about something else.
//
// It reports whether the turn happened: a battle that ended, or a unit that lost
// its turn, is not a failure here — the caller is driving a battle to a health
// threshold, and the threshold is what it asserts on.
func take(t *testing.T, fight *battle.Battle, skillID string) bool {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Skipped {
		return false
	}
	for _, option := range prompt.Options {
		if option.Skill != skillID || !option.Available() {
			continue
		}
		if err := fight.Act(skillID, option.Aims[0]); err != nil {
			t.Fatalf("act %s: %v", skillID, err)
		}
		return true
	}
	return false
}

// step advances one turn and takes the first action the unit is offered,
// whichever unit that is. It is what a test wants when the subject is where the
// battle ends up rather than who does what on the way: take names a skill and so
// quietly does nothing on the other unit's turn, which is fine while the two
// alternate and wrong the moment a speed changes.
func step(t *testing.T, fight *battle.Battle) bool {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil {
		return false
	}
	if prompt.Skipped {
		return true
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
	if err := fight.Pass("nothing to do"); err != nil {
		t.Fatalf("pass: %v", err)
	}
	return true
}

// TestAGatedGrantIsOffUntilItsHolderIsHurt is the whole of the feature in one
// battle: a trait that is a stat change *and* a condition.
//
// It was refused at parse until now, because a grant is put on once and the
// status it puts on is permanent so that nothing can dispel it — which left the
// gate with no way to take it back. What is checked here is the first half of
// the answer: the trait is not on at full health, is not announced by the
// opening board, and comes on the moment its holder crosses the line.
func TestAGatedGrantIsOffUntilItsHolderIsHurt(t *testing.T) {
	fight := gated(t, "dug_in", []string{"jab"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")

	if ally.Statuses.Has("toughened") {
		t.Error("a gated trait is in force at full health, so its gate does nothing")
	}
	for _, event := range find(fight.Drain(), battle.PassiveHeld) {
		if event.Actor == "a" {
			t.Errorf("the opening board announces %q on a unit that is not carrying it",
				event.Passive)
		}
	}

	// Hurt it past the line. The ally acts first and jabs for almost nothing, so
	// the enemy's strike is what moves the health that matters.
	for round := 0; round < 4 && !ally.Statuses.Has("toughened"); round++ {
		if !take(t, fight, "jab") {
			break
		}
		if !take(t, fight, "strike") {
			break
		}
	}
	if !ally.Statuses.Has("toughened") {
		t.Fatalf("the trait never came on; the holder is at %d of %d", ally.HP, ally.MaxHP())
	}

	held := find(fight.Drain(), battle.PassiveHeld)
	if len(held) != 1 {
		t.Fatalf("the trait coming on produced %d events, want exactly one", len(held))
	}
	if held[0].Actor != "a" || held[0].Passive != "dug_in" || held[0].Status != "toughened" {
		t.Errorf("the event reads %+v, want dug_in granting toughened to a", held[0])
	}
	if held[0].Stacks != 2 {
		t.Errorf("the event says %d stacks, want the 2 the trait grants", held[0].Stacks)
	}
}

// TestAGatedGrantGoesOffAgainWhenItsHolderIsHealed is the other half, and the
// reason the grant needed a door rather than a flag.
//
// A permanent status is refused by Remove so that no cleanse can strip a trait,
// which leaves Hold and Release as the only way in and out — and a trait that
// came on and could not go off again would be a stat change with no way back,
// which is worse than the refusal it replaced.
func TestAGatedGrantGoesOffAgainWhenItsHolderIsHealed(t *testing.T) {
	// The holder climbs back under its own power: drink returns everything it
	// deals, so nothing here depends on a second unit choosing to heal it.
	fight := gated(t, "dug_in", []string{"jab", "drink"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	for round := 0; round < 6 && !ally.Statuses.Has("toughened"); round++ {
		if !take(t, fight, "jab") {
			break
		}
		if !take(t, fight, "strike") {
			break
		}
	}
	if !ally.Statuses.Has("toughened") {
		t.Fatalf("the trait never came on; the holder is at %d of %d", ally.HP, ally.MaxHP())
	}
	fight.Drain()

	// Drinking back over the line. The enemy is not asked to act, because what is
	// under test is the crossing rather than a race.
	for round := 0; round < 12 && ally.Statuses.Has("toughened"); round++ {
		if !take(t, fight, "drink") {
			break
		}
	}
	if ally.Statuses.Has("toughened") {
		t.Fatalf("the trait is still on at %d of %d, so the gate is a one way door",
			ally.HP, ally.MaxHP())
	}

	released := find(fight.Drain(), battle.PassiveReleased)
	if len(released) != 1 {
		t.Fatalf("the trait going off produced %d events, want exactly one", len(released))
	}
	if released[0].Actor != "a" || released[0].Passive != "dug_in" ||
		released[0].Status != "toughened" {
		t.Errorf("the event reads %+v, want dug_in releasing toughened on a", released[0])
	}
	if released[0].Stacks != 2 {
		t.Errorf("the event says %d stacks went, want the 2 that were on", released[0].Stacks)
	}
	if ally.Statuses.Stacks("toughened") != 0 {
		t.Errorf("a released trait left %d stacks behind, and a grant is a fact rather than a pile",
			ally.Statuses.Stacks("toughened"))
	}
}

// TestAGatedTraitTouchingSpeedRetunesTheQueue is the constraint the first slice
// of passives was built under, now that a trait can change mid-battle.
//
// A wait is 1_000_000/speed, so a speed that changed without the queue being
// told would leave its holder serving a wait it no longer owes — the bug haste
// found once, and the reason retuneAll exists.
//
// The assertion is that the speed change is the *next* event, and that is the
// whole point of it. A turn already ends with a sweep of its own, so a trait
// that came on and said nothing would still have the queue put right before
// anybody acted again — and the log would carry a speed change several events
// later, next to whatever happened to be resolving at the time, with the trait
// that caused it out of sight. A reader cannot join those up. This is the same
// rule that gives the trait an event in each direction rather than leaving its
// effect to be inferred.
func TestAGatedTraitTouchingSpeedRetunesTheQueue(t *testing.T) {
	fight := gated(t, "desperate", []string{"jab"}, []string{"triple"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	before := fight.Stats(ally)[progression.Speed]
	fight.Drain()
	var events []battle.Event
	for round := 0; round < 6 && !ally.Statuses.Has("fleet"); round++ {
		if !take(t, fight, "jab") {
			break
		}
		events = append(events, fight.Drain()...)
		if !take(t, fight, "triple") {
			break
		}
		events = append(events, fight.Drain()...)
	}
	if !ally.Statuses.Has("fleet") {
		t.Fatalf("the trait never came on; the holder is at %d of %d", ally.HP, ally.MaxHP())
	}
	after := fight.Stats(ally)[progression.Speed]
	if after <= before {
		t.Fatalf("the trait is on and speed went %d to %d, so it granted nothing", before, after)
	}

	held := -1
	for i, event := range events {
		if event.Kind == battle.PassiveHeld && event.Actor == "a" {
			held = i
		}
	}
	if held < 0 {
		t.Fatalf("the trait came on and the log never said so: %+v", events)
	}
	if held+1 >= len(events) {
		t.Fatal("the trait coming on is the last thing in the log, so nothing retuned")
	}
	next := events[held+1]
	if next.Kind != battle.SpeedChanged || next.Actor != "a" {
		t.Errorf("the event after the trait came on is %s on %q, want the speed change it caused",
			next.Kind, next.Actor)
	}
	if next.Before != before || next.Amount != after {
		t.Errorf("the speed change reads %d to %d, want %d to %d",
			next.Before, next.Amount, before, after)
	}
}

// TestAGateIsReadEveryTimeHealthMovesAndNotOtherwise is what keeps the gate from
// being a per-turn poll.
//
// Health moves in exactly two places, so those are the two that reconsider — and
// a crossing has to be noticed on the turn it happens rather than on the next
// one, because a trait that turned on a turn late would be a stat change the log
// blamed on the wrong action. What must *not* happen is a second announcement
// each time health moves again on the same side of the line.
func TestAGateIsReadEveryTimeHealthMovesAndNotOtherwise(t *testing.T) {
	fight := gated(t, "dug_in", []string{"jab"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	held := 0
	for round := 0; round < 6; round++ {
		if !take(t, fight, "jab") {
			break
		}
		if !take(t, fight, "strike") {
			break
		}
		held += len(find(fight.Drain(), battle.PassiveHeld))
		if ally.Dead {
			break
		}
	}
	if !ally.Statuses.Has("toughened") && !ally.Dead {
		t.Fatalf("the trait never came on; the holder is at %d of %d", ally.HP, ally.MaxHP())
	}
	if held != 1 {
		t.Errorf("the trait was announced %d times while its holder kept falling, want once", held)
	}
	if stacks := ally.Statuses.Stacks("toughened"); !ally.Dead && stacks != 2 {
		t.Errorf("the holder carries %d stacks of toughened, want the 2 granted once", stacks)
	}
}

// TestAnUngatedGrantStillOpensWithTheBoard is the case that must not have moved.
//
// Reading the gate at enlistment rather than assuming it open is what makes a
// gated trait start off; a trait with no gate has to be unaffected by that, and
// it is the case every existing passive in the shipped data is.
func TestAnUngatedGrantStillOpensWithTheBoard(t *testing.T) {
	fight := gated(t, "hardy", []string{"jab"}, []string{"strike"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	if !ally.Statuses.Has("toughened") {
		t.Fatal("an ungated trait is not in force at full health")
	}
	held := find(fight.Drain(), battle.PassiveHeld)
	found := false
	for _, event := range held {
		if event.Actor == "a" && event.Passive == "hardy" {
			found = true
		}
	}
	if !found {
		t.Errorf("the opening board does not announce an ungated trait: %+v", held)
	}
}

// TestATraitDoesNotComeOnAsItsHolderDies is the case the health guard is for,
// and it is not the same case as the one below.
//
// A holder that is worn down crosses the line long before it falls, so its trait
// is already on by the time it dies and nothing wants to change. The case that
// bites is one blow from full health to nought: the gate would be satisfied for
// the first time by the same damage that killed the unit, and a trait announcing
// itself there would be a stat change on something whose died line is the next
// event.
//
// The strike loop is why a Dead flag is not enough on its own: it leaves a
// target at zero for the rest of the skill and kills it afterwards, so there is
// a window where health says one thing and the flag says another.
func TestATraitDoesNotComeOnAsItsHolderDies(t *testing.T) {
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(600, 800, 100, 90),
			Skills: []string{"jab"}, Passives: []string{"dug_in"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	ally := unitByID(t, fight, "a")
	if ally.Statuses.Has("toughened") {
		t.Fatal("the holder starts with its gated trait on, so this measures nothing")
	}
	fight.Drain()

	if !take(t, fight, "strike") {
		t.Fatal("the enemy did not get its turn")
	}
	if !ally.Dead {
		t.Fatalf("the holder survived the blow at %d of %d, so the killing hit is not the one being measured",
			ally.HP, ally.MaxHP())
	}
	for _, event := range fight.Drain() {
		if event.Actor != "a" {
			continue
		}
		if event.Kind == battle.PassiveHeld || event.Kind == battle.PassiveReleased {
			t.Errorf("the blow that killed the holder also emitted %s for it", event.Kind)
		}
	}
	if ally.Statuses.Has("toughened") {
		t.Error("the holder died holding a trait it never had while it was alive")
	}
}

// TestADeadHolderIsNotReconsidered is the guard on the one path where health
// moves and the unit is past caring.
//
// The strike loop leaves a target at nought and kills it after the skill has
// finished, so for a moment there is a unit at zero health whose Dead flag is
// still false — and a gate re-read there would announce a trait coming on to
// something whose died line is two events away. It is the same rule that stops
// a dead unit being healed.
func TestADeadHolderIsNotReconsidered(t *testing.T) {
	fight := gated(t, "dug_in", []string{"jab"}, []string{"cleave"})
	fight.Begin()
	ally := unitByID(t, fight, "a")
	for round := 0; round < 200 && !ally.Dead; round++ {
		if !step(t, fight) {
			break
		}
	}
	if !ally.Dead {
		t.Fatalf("the holder survived at %d of %d, so there is no corpse to re-read",
			ally.HP, ally.MaxHP())
	}
	events := fight.Drain()
	died := -1
	for i, event := range events {
		if event.Kind == battle.Died && event.Actor == "a" {
			died = i
		}
	}
	if died < 0 {
		t.Fatal("the holder is dead and the log never said so")
	}
	for _, event := range events[died:] {
		if event.Actor != "a" {
			continue
		}
		if event.Kind == battle.PassiveHeld || event.Kind == battle.PassiveReleased {
			t.Errorf("a dead holder emitted %s at or after its own died line", event.Kind)
		}
	}
}
