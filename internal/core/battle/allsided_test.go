package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// crossfire is a board where an all-sided shape really does catch both halves.
//
// `quake` is a wedge aimed across the midline, so from the caster at 2,1 it
// covers the enemy's front cell 3,1 and comes back onto 2,1 and 2,2 — the caster
// itself and whoever stands behind it. That geometry is the whole point of the
// fixture: an all-sided skill on a board where nothing of the caster's own is in
// the blast measures nothing, because there is no cost to weigh.
//
// `behind` decides whether 2,2 is occupied and `behindHealth` how badly hurt that
// unit is, which is the difference between "an ally takes a graze" and "an ally
// dies for it".
func crossfire(t *testing.T, casterSkills []string, casterHealth int64,
	behind bool, behindHealth int64) *battle.Battle {
	t.Helper()
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 120),
			Skills: casterSkills},
		{ID: "up", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"lob"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"lob"}},
	}
	if behind {
		roster = append(roster, battle.Roster{
			ID: "behind", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"lob"}})
	}
	fight, err := battle.New(books(t), 7, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	atHealth(t, fight, "a", casterHealth)
	if behind {
		atHealth(t, fight, "behind", behindHealth)
	}
	return fight
}

// TestAnAllSidedAttackIsRatedByBothHalvesOfWhatItDoes is the contract that
// replaced a refusal.
//
// Suggest used to skip a damaging all-sided skill outright, and the reason was
// sound: `expected` skips a unit on the caster's own side rather than subtracting
// it, so a rating allowed to see one would have counted the harm and not the
// cost — the opponent that bombs its own squad and calls it a gain. The guard and
// that reason were two halves of one decision, so the guard could only go once
// the other half was answered, which is what `friendlyFire` is.
//
// The two boards here differ in one thing: whether an ally is standing in the
// blast. Nothing about the skill, the caster or the enemy changes.
func TestAnAllSidedAttackIsRatedByBothHalvesOfWhatItDoes(t *testing.T) {
	// Nobody of the caster's own behind it, so the only thing the wedge catches on
	// this side is the caster, which the enemy's share more than pays for: quake's
	// power is 500 against the jab's 60.
	clear := crossfire(t, []string{"quake", "jab"}, 0, false, 0)
	if choice := chosen(t, clear); choice.Skill != "quake" {
		t.Errorf("with nobody in the blast Suggest picked %q, want quake: an "+
			"all-sided attack is rated now, not skipped", choice.Skill)
	}

	// One ally into the same wedge and the arithmetic turns over: the enemy's
	// share is 171, the caster's own graze 85, and the ally's another 85 — so the
	// skill is worth 1 where the jab is worth about 20.
	shared := crossfire(t, []string{"quake", "jab"}, 0, true, 0)
	if choice := chosen(t, shared); choice.Skill != "jab" {
		t.Errorf("with an ally in the blast Suggest picked %q, want jab: the own "+
			"side's share of an all-sided attack is a cost", choice.Skill)
	}
}

// TestAnAllSidedAttackWillNotKillItsOwnSideForAGraze prices the other half of
// friendlyFire: the turns lost with a unit, which is `finished` pointed the other
// way.
//
// A unit's health is what a rating clamps damage at, so a nearly-dead ally in the
// blast is *cheap* on the damage term alone — 60 points where a healthy one is 85.
// It is the kill that makes it expensive, exactly as a kill is what makes a
// finishing blow worth more than the health it takes off.
func TestAnAllSidedAttackWillNotKillItsOwnSideForAGraze(t *testing.T) {
	fight := crossfire(t, []string{"quake", "jab"}, 0, true, 60)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q, want jab: an all-sided attack that finishes "+
			"one of its own is priced for the turns that unit will not take",
			choice.Skill)
	}
}

// TestAnAllSidedAttackWillNotKillItsOwnCaster is the same rule applied to the one
// unit it would be easy to leave out of the sweep.
//
// A shape can cover the cell its own caster stands in — this one does — and
// `resolveAgainst` has never asked whose side a target is on, so the caster
// really does take the hit. A rating that skipped itself would prefer the skill
// that kills nobody but itself.
func TestAnAllSidedAttackWillNotKillItsOwnCaster(t *testing.T) {
	fight := crossfire(t, []string{"quake", "jab"}, 60, false, 0)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q, want jab: the caster is in its own blast",
			choice.Skill)
	}
}

// TestAnAllSidedBuffWillNotPayToHelpTheEnemyToo is the support half of the same
// change, and it is measured on the aim rather than on the skill.
//
// `anthem` hastes everything its wedge covers, so which cell it is aimed at
// decides who it helps. Aimed at the enemy's front it hastes the enemy and the
// caster; aimed at the caster's own half it hastes the caster alone. The same
// gain is available without the gift, so the gift is what a rating drops.
//
// Before this it could not: an all-sided skill rated nothing at all, so `anthem`
// was reached only as the fallback and took the first aim it was offered.
func TestAnAllSidedBuffWillNotPayToHelpTheEnemyToo(t *testing.T) {
	fight := crossfire(t, []string{"anthem"}, 0, false, 0)
	choice := chosen(t, fight)
	if choice.Skill != "anthem" {
		t.Fatalf("Suggest picked %q, want the only skill in the kit", choice.Skill)
	}
	if choice.Aim.Side() != hex.SideAlly {
		t.Errorf("the battlefield-wide haste was aimed at %v (%v side), want the "+
			"caster's own half: hasting an enemy is a cost", choice.Aim,
			choice.Aim.Side())
	}
}

// TestAnAllSidedHarmIsWorthWhatItPutsOnTheEnemy is the other half of the status
// loop, and only a mutation asked for it.
//
// The loop walks the shape and reads each occupant by the branch that fits it:
// harm on an enemy is a gain, harm on one's own side a cost, help the other way
// round. The old guard skipped whichever half a one-sided skill could not reach —
// and left in place for an all-sided one it skipped the **enemy** half, so a
// battlefield-wide poison was worth only what it cost. Every test above still
// passed, because a damaging all-sided skill is priced by expected rather than by
// this loop.
//
// `fume` covers a single cell, which is what makes the two halves separable: aimed
// across the midline it is pure gain, and aimed at the caster's own half pure cost.
func TestAnAllSidedHarmIsWorthWhatItPutsOnTheEnemy(t *testing.T) {
	fight := crossfire(t, []string{"fume", "jab"}, 0, false, 0)
	choice := chosen(t, fight)
	if choice.Skill != "fume" {
		t.Fatalf("Suggest picked %q, want fume: three turns of poison on the enemy "+
			"is worth more than a jab", choice.Skill)
	}
	if choice.Aim.Side() != hex.SideEnemy {
		t.Errorf("the poison was aimed at %v (%v side), want the enemy's half",
			choice.Aim, choice.Aim.Side())
	}
}

// TestAUnitWhoseOnlyAttackIsAllSidedStillThreatensSomebody closes the blind spot
// the change would otherwise have left.
//
// Every defensive term in price.go is finite because it is priced against
// `threat` — what an enemy could take off in one turn — and `threat` is
// `bestStrike` pointed the other way. `bestStrike` used to count only enemy-aimed
// skills, so a unit whose one attack was all-sided read as threatening nobody: a
// heal on the ally it was about to hit was worth nothing, and a shield against it
// ate nothing.
//
// Measured through a heal, because a heal is worth exactly nothing when nothing
// can take health off. The ally is hurt and the enemy holds `sweep` — a column
// aimed at both halves, which is still an attack.
func TestAUnitWhoseOnlyAttackIsAllSidedStillThreatensSomebody(t *testing.T) {
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"bless", "jab"}},
		{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"lob"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"sweep"}},
	}
	fight, err := battle.New(books(t), 7, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	atHealth(t, fight, "mate", 300)
	if choice := chosen(t, fight); choice.Skill != "bless" {
		t.Errorf("Suggest picked %q, want the regeneration on the hurt ally: an "+
			"all-sided attacker is a threat, so healing against it is worth "+
			"something", choice.Skill)
	}
}
