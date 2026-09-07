package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// Pierce was a property of a SKILL and of nothing else: `razor_leaf` pierces
// because that is what razor_leaf is. A squad could not be made to pierce, so the
// obvious composition bonus for a dark tribe — everyone's blows cut deeper — had
// no way to exist.
//
// ⚠️ **It ADDS to the skill's own share, and a skill that pierces nothing is
// still raised.** A share that only applied to skills already piercing would be a
// bonus a kit either has or cannot use, which is a narrower design; this is the
// author's call and it is recorded because the two read the same in a data file
// and not at all the same on a board.

// TestAPierceShareCutsThroughDefenceOnASkillThatHasNone is the effect, on the
// case that decides the design.
func TestAPierceShareCutsThroughDefenceOnASkillThatHasNone(t *testing.T) {
	plain := struckAfter(t, nil)
	sundered := struckAfter(t, []string{"whet"})
	if plain <= 0 {
		t.Fatal("the control dealt nothing, so there is no blow for the share to sharpen")
	}
	if sundered <= plain {
		t.Errorf("a piercing holder dealt %d against a plain %d: the share is not reaching "+
			"the blow", sundered, plain)
	}
}

// struckAfter runs one unit through an optional self-buff and then a plain
// strike, and reports what the strike took off a heavily armoured target.
//
// The target is armoured on purpose: pierce is a share of DEFENCE, so a soft
// target would measure a share of almost nothing and the two readings would
// differ by a rounding error.
func struckAfter(t *testing.T, before []string) int64 {
	t.Helper()
	kit := append(append([]string{}, before...), "jab")
	fight, err := battle.New(books(t), 5, []battle.Roster{
		{ID: "cutter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60), Skills: kit},
		{ID: "armour", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2800, 200, 800, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	armour, _ := fight.Unit("armour")
	cutter, _ := fight.Unit("cutter")
	for _, id := range kit {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "cutter" {
			t.Fatalf("%s is up and this fixture needs the cutter to be", prompt.Unit)
		}
		aim := cutter.Cell
		if id == "jab" {
			aim = armour.Cell
		}
		was := armour.HP
		if err := fight.Act(id, aim); err != nil {
			t.Fatalf("act %s: %v", id, err)
		}
		if id == "jab" {
			return was - armour.HP
		}
	}
	t.Fatal("the strike was never thrown")
	return 0
}

// TestAPierceShareIsWorthSomethingToTheRating is the half the mutation found
// missing, and it is the exact defect `converts` carries a warning about.
//
// A rating that builds its hit without the caster's own share prices every blow a
// piercing unit throws as SMALLER than the one it lands — and against an armoured
// target, which is the only board the effect is for, it prefers the wrong skill.
// Reverting `ai.go`'s Pierce to the skill's own left the whole suite green until
// this existed.
//
// The board is arranged so the answer is not close: the target's defence is most
// of what stands between the two units, so sharpening the blow first is worth more
// than throwing it now.
func TestAPierceShareIsWorthSomethingToTheRating(t *testing.T) {
	fight, err := battle.New(books(t), 5, []battle.Roster{
		{ID: "cutter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60),
			Skills: []string{"whet", "jab"}},
		{ID: "armour", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2800, 200, 800, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "cutter" {
		t.Fatalf("%s is up and this fixture needs the cutter to be", prompt.Unit)
	}
	choice, acted := fight.Suggest(prompt)
	if !acted {
		t.Fatal("the rating passed a turn it had skills for")
	}
	if choice.Skill != "whet" {
		t.Errorf("the rating chose %q over sharpening against a target whose defence is "+
			"most of its survival: a pierce share is being priced at nothing", choice.Skill)
	}
}
