package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestADrainIsWorthTheHealthItTakesBack is the term four shipped things were
// worth nothing without.
//
// `leech_seed`, `dream_eater`, `blood_thirst` and `last_gasp` all hand health
// back out of the damage they deal, and `rate` charged for none of it — offered a
// plain hit and the same hit returning all of itself, the rating took the plain
// one. A drain is the exact mirror of `restores`, so it is priced the same way
// and collects the same clamps.
//
// ⚠️ **Both directions, because the clamp is the whole design.** `cleave` deals
// strictly more than `drink` — same power, six tenths of the defence pierced — so
// at full health, where there is no room to heal into, it has to stay the answer.
// A term that made the draining skill win everywhere would be a flat bonus
// wearing a health check.
func TestADrainIsWorthTheHealthItTakesBack(t *testing.T) {
	for _, row := range []struct {
		name   string
		health int64
		want   string
	}{
		{"a caster with room to heal into", 400, "drink"},
		{"a caster at full health", 0, "cleave"},
	} {
		fight := squad(t, []string{"cleave", "drink"}, []string{"jab"}, []string{"strike"},
			row.health, 0, 0)
		if choice := chosen(t, fight); choice.Skill != row.want {
			t.Errorf("with %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}

// TestATraitsDrainIsPricedLikeASkillsIs is the half a reading of
// `declared.Drains` alone would have missed, and it is half the shipped cases:
// `blood_thirst` and `last_gasp` are traits, not skills.
//
// The share is `drainShare(declared.Drains + lifesteal(actor))`, which is the
// expression the resolving side pays it with — sum first, then bound, because two
// drains simply both drain and a share of damage dealt is not a chance.
//
// The board is a hurt caster under real threat, where shielding itself and
// hitting back are close enough that the drain is what decides. Without the
// trait it braces; with it, hitting back also heals and is the better turn.
func TestATraitsDrainIsPricedLikeASkillsIs(t *testing.T) {
	for _, row := range []struct {
		name   string
		traits []string
		want   string
	}{
		{"no trait", nil, "brace"},
		{"a draining trait", []string{"thirst"}, "strike"},
	} {
		fight, err := battle.New(books(t), 7, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
				Skills: []string{"brace", "strike"}, Passives: row.traits},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"strike"}},
		})
		if err != nil {
			t.Fatalf("%s: new battle: %v", row.name, err)
		}
		fight.Begin()
		atHealth(t, fight, "a", 900)
		if choice := chosen(t, fight); choice.Skill != row.want {
			t.Errorf("with %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}
