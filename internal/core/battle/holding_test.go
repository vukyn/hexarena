package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestAScarceSkillIsNotSpentOnWhatACommonOneBuys is the whole of what "holding a
// skill for a later turn" comes to in a rating one turn deep.
//
// Damage is clamped at a target's remaining health — a finishing blow is not rated
// above one that would kill twice over — so against a unit standing at a sliver
// the heaviest skill in a kit and the filler beside it are worth **exactly** the
// same. Before this the tie went to whichever came first in the kit, so `clout`
// was burnt on ten points of health and cooled down for three turns for it while
// `jab`, which is always there, was the option that would have done it.
//
// `clout` is deliberately listed first, because kit order is what the tie-break
// replaces: a version of this test with the cheap skill first passes either way.
func TestAScarceSkillIsNotSpentOnWhatACommonOneBuys(t *testing.T) {
	fight := squad(t, []string{"clout", "jab"}, []string{"lob"}, []string{"jab"},
		0, 0, 10)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q against a unit at ten health, want jab: two "+
			"options worth the same are not the same to spend", choice.Skill)
	}
}

// TestASummonIsSpentOnTheSameTermsAsAnything is the branch a mutation caught,
// and it caught it because nothing shipped summons.
//
// A summoning skill is rated before the shape — it has no power of its own, so it
// would otherwise be the fallback it used to be — and that makes it the one place
// in Suggest where the comparison could quietly have been written a second time.
// Two casts that put the same body on the same cell are worth the same, so the one
// that will be available again is the one to spend.
func TestASummonIsSpentOnTheSameTermsAsAnything(t *testing.T) {
	fight := squad(t, []string{"slow_copy", "copy"}, []string{"lob"},
		[]string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "copy" {
		t.Errorf("Suggest picked %q, want copy: the two calls up the same body, "+
			"and one of them cools down for three turns", choice.Skill)
	}
}

// TestTheCheaperSkillDoesNotWinOnValue is the other half, and it is the half that
// keeps the tie-break a tie-break.
//
// A cooldown says nothing about what a skill is worth, so it may only decide
// between options already worth the same. Against a healthy unit `clout` deals
// more than `jab` and is the answer, cooldown and all — a rating that discounted
// scarcity would be pricing turns it cannot see, which is the mistake tempo
// already had to be corrected for.
func TestTheCheaperSkillDoesNotWinOnValue(t *testing.T) {
	fight := squad(t, []string{"clout", "jab"}, []string{"lob"}, []string{"jab"},
		0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "clout" {
		t.Errorf("Suggest picked %q against a healthy unit, want clout: the "+
			"tie-break may not outrank what an option is worth", choice.Skill)
	}
}

// TestAPassBuysNoCooldownAnActDoesNot is why *waiting* is not a thing this engine
// can be made to want, and it is the arithmetic rather than an opinion.
//
// spendCooldowns brings **every** cooldown down by the turn just served, and it
// runs at the end of Act, at the end of Pass and on a turn control took. Act then
// starts a cooldown on the one skill it cast and on nothing else. So the skill a
// unit might be said to be "waiting for" comes back on exactly the same turn
// whether the unit acts or not: across its next two turns, acting is worth
// `bestValue` now plus next turn's best, and waiting is worth nought now plus the
// same next turn's best. Acting dominates by exactly `bestValue`, and an option
// priced below nought is already declined — so there is no residue for a waiting
// rule to collect.
//
// This test is that dominance made executable. It is what fails if somebody ever
// makes a Pass cheaper than an Act — a pass that spent two turns of cooldown, or
// an act that spent none — because that is the one change that would give waiting
// something to buy, and it would otherwise be found only by a rating quietly
// getting worse.
func TestAPassBuysNoCooldownAnActDoesNot(t *testing.T) {
	// slow_copy is the skill being held for later: it is put on cooldown by hand
	// so that there is something for a wait to be waiting on. clout is what the
	// acting fixture spends its turn on, and it has a cooldown of its own so that
	// the *only* difference the two fixtures may show is the cast skill's own.
	const held, cast = "slow_copy", "clout"
	kit := []string{held, cast, "jab"}

	waited := squad(t, kit, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	coolingDown(t, waited, "a", held, 3)
	if _, err := waited.Advance(); err != nil {
		t.Fatalf("advance the waiting fixture: %v", err)
	}
	if err := waited.Pass("held"); err != nil {
		t.Fatalf("pass: %v", err)
	}

	acted := squad(t, kit, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	coolingDown(t, acted, "a", held, 3)
	prompt, err := acted.Advance()
	if err != nil {
		t.Fatalf("advance the acting fixture: %v", err)
	}
	aim, aimed := firstAim(prompt, cast)
	if !aimed {
		t.Fatalf("%s could not be aimed at anything, so the fixture proves nothing", cast)
	}
	if err := acted.Act(cast, aim); err != nil {
		t.Fatalf("act: %v", err)
	}

	waiting, acting := cooldowns(t, waited, "a"), cooldowns(t, acted, "a")
	for index, id := range kit {
		if id == cast {
			continue
		}
		if waiting[index] != acting[index] {
			t.Errorf("%s cools to %d after a pass and to %d after an act: a turn spent "+
				"acting has to buy exactly as much cooldown as a turn spent waiting, or "+
				"waiting is worth something and Suggest is wrong to refuse it",
				id, waiting[index], acting[index])
		}
	}
	// The held skill has to have actually moved, or the loop above is comparing
	// two numbers that were never going to differ.
	if waiting[0] != 2 {
		t.Errorf("%s sits at %d after one turn of a three-turn cooldown, want 2: a turn "+
			"that bought no cooldown at all is a fixture that tests nothing", held, waiting[0])
	}
	// And the one difference an act is allowed to make is the skill it cast.
	if acting[1] == waiting[1] {
		t.Errorf("%s reads %d after being cast and %d after a pass: an act pays its own "+
			"cooldown, and a fixture where it does not cannot tell the two apart",
			cast, acting[1], waiting[1])
	}
}

// coolingDown puts a skill on cooldown by hand, the way atHealth puts a unit at a
// health: a battle has no way to spend a turn on purpose, and a case about a skill
// that is already recharging should not have to fight one down to get there.
func coolingDown(t *testing.T, fight *battle.Battle, id, skillID string, turns int) {
	t.Helper()
	unit, known := fight.Unit(id)
	if !known {
		t.Fatalf("no unit %q", id)
	}
	for index, carried := range unit.Skills {
		if carried == skillID {
			unit.Cooldowns[index] = turns
			return
		}
	}
	t.Fatalf("%s does not carry %s", id, skillID)
}

// cooldowns is a unit's cooldowns as they stand.
func cooldowns(t *testing.T, fight *battle.Battle, id string) []int {
	t.Helper()
	unit, known := fight.Unit(id)
	if !known {
		t.Fatalf("no unit %q", id)
	}
	return unit.Cooldowns
}

// firstAim is where a named option may be pointed, or false when it is not on
// offer at all.
func firstAim(prompt *battle.Prompt, skillID string) (hex.Offset, bool) {
	for _, option := range prompt.Options {
		if option.Skill == skillID && option.Available() {
			return option.Aims[0], true
		}
	}
	return hex.Offset{}, false
}

// TestTheFallbackFollowsTheTieBreakToo is the same rule as the three tests above,
// in the one arm of Suggest that had been left out of it.
//
// The tie-break exists because two options worth the SAME are not the same to
// spend, and options worth **nothing** are the sharpest case of that rather than
// an exception to it: whichever is cast buys nought either way, so the only thing
// that separates them is what casting it costs. `scour` and `wipe` strip the same
// two categories off an ally carrying neither, so both rate nought — and one of
// them is gone for three turns afterwards.
//
// ⚠️ **Measured before it was written**: this arm kept "the first such skill in
// kit order" long after `take` stopped doing so, so kit `[scour, wipe]` cast
// `scour` and kit `[wipe, scour]` cast `wipe`. Kit order was the whole of the
// decision. The shipped shape of it is `rapid_spin` — power nought, cooldown
// three, strips one stack — cast on a board with nothing to strip, which is a
// cleanse spent for three turns and nothing bought.
//
// ⚠️ Both orders are asserted, and the second is the control: it passes whichever
// rule is in force, so a test that ran only that way round would be green against
// kit order.
func TestTheFallbackFollowsTheTieBreakToo(t *testing.T) {
	for _, kit := range [][]string{{"scour", "wipe"}, {"wipe", "scour"}} {
		// No enemy skill and nothing on anybody to strip, so every option the
		// actor holds is worth exactly nought and the fallback is the whole
		// decision. A single rated option anywhere would make this test about
		// `take` instead.
		fight := squad(t, kit, []string{"jab"}, []string{"jab"}, 0, 0, 0)
		if choice := chosen(t, fight); choice.Skill != "wipe" {
			t.Errorf("with kit %v Suggest picked %q, want wipe: two options worth "+
				"nothing are not the same to spend, and one of them is gone for "+
				"three turns", kit, choice.Skill)
		}
	}
}

// TestTheFallbackStillLosesToAnythingWorthDoing is the other half, and it keeps
// the tie-break a tie-break here exactly as TestTheCheaperSkillDoesNotWinOnValue
// keeps it one for a rated option.
//
// A cooldown says nothing about what a skill is worth. `scour` is the cheapest
// thing in this kit to spend and it is still the wrong answer the moment there is
// something in reach worth hitting.
func TestTheFallbackStillLosesToAnythingWorthDoing(t *testing.T) {
	fight := squad(t, []string{"wipe", "jab"}, []string{"jab"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q with an enemy in reach, want jab: the fallback "+
			"is taken only when nothing at all was worth doing", choice.Skill)
	}
}
