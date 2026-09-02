package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestATauntIsWorthTheAimItTakesAway is a category `inflictedOn` fell through
// for, and it is priced in `granted` instead — because the status sits on the
// unit DOING the taunting, not on the unit being taunted.
//
// ⚠️ The first cut of this put it in `inflictedOn`, which is where a harmful
// self-application is charged as a cost, so a unit that taunted was billed its own
// best strike three times over for doing it — worse than the nothing it was worth
// before. The shipped `taunt` is a self-aimed skill whose whole body is
// `self_applies`, which is what made the mistake invisible to a fixture that put
// the status on an enemy instead.
//
// What a taunt buys is the aim it takes off every enemy at once. A taunt is not a
// Control and pricing it as one would be wrong the other way — a stunned unit does
// not act, a taunted one acts and is simply not allowed to pick.
//
// ⚠️ **Both rows, and the second is the whole of what keeps it honest.** When the
// taunter is already their best target the taunt changes nobody's mind, so it has
// to be worth nothing — a term that paid for it anyway would have every unit
// taunting every turn.
func TestATauntIsWorthTheAimItTakesAway(t *testing.T) {
	for _, row := range []struct {
		name      string
		mateGuard int64
		want      string
	}{
		{"a squishy ally the enemy would rather hit", 60, "goad"},
		{"an ally harder to reach than the taunter", 800, "jab"},
	} {
		fight, err := battle.New(books(t), 7, []battle.Roster{
			// The taunter is the armoured one, so the enemy's best cast is aimed
			// past it in the first row and at it in the second.
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 200, 400, 120),
				Skills: []string{"jab", "goad"}},
			{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
				Affinity: single("neutral"), Stats: stats(3000, 200, row.mateGuard, 20),
				Skills: []string{"jab"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"strike"}},
		})
		if err != nil {
			t.Fatalf("%s: new battle: %v", row.name, err)
		}
		fight.Begin()
		if choice := chosen(t, fight); choice.Skill != row.want {
			t.Errorf("with %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}

// TestATauntOnAnEnemyIsACostRatherThanAGain is the same term read from the other
// end, and it is what says the price is about the holder rather than about the
// caster.
//
// Nothing shipped hands an enemy a taunt, but `provoke` in the fixture book does,
// and the answer has to be that it is a bad idea: a taunt on an enemy forces the
// caster's own side onto that enemy. A term that lived in `inflictedOn` would have
// read it as a gain.
func TestATauntOnAnEnemyIsACostRatherThanAGain(t *testing.T) {
	fight := squad(t, []string{"jab", "provoke"}, []string{"jab"}, []string{"strike"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q, want jab: handing an enemy a taunt narrows the "+
			"caster's OWN side onto it", choice.Skill)
	}
}

// ticking puts stacks of a ticking status on a unit at a chosen amount, which the
// rows below need because the default two hundred a tick swamps every figure it
// would otherwise be compared against.
func ticking(t *testing.T, fight *battle.Battle, id, statusID string, stacks int, tick int64) {
	t.Helper()
	unit, known := fight.Unit(id)
	if !known {
		t.Fatalf("no unit %q", id)
	}
	kind, err := fight.Books().Statuses.Lookup(statusID)
	if err != nil {
		t.Fatalf("look %s up: %v", statusID, err)
	}
	for range stacks {
		unit.Statuses.Apply(kind, tick)
	}
}

// TestAHealCutIsWorthTheHealingItDenies is the eighth category `inflictedOn` had
// no case for, and the consequence of its absence was written down when `fester`
// shipped: the opponent never aimed one at a healer on purpose.
//
// The cut is read through `healingFor`, the expression the battle pays a heal
// with, so the price and the effect cannot disagree about how much of one a stack
// takes — and through `worthHealing`, so health nobody could have banked is not
// health anybody can deny.
//
// ⚠️ **Five rows, and four of them are refusals.** `rot` is ten power against
// `jab`'s sixty, so on a target with nothing to cut it has to lose: what the cut
// buys is the difference, and only where there is healing that would actually
// have landed. Every clause in the price has a row here, because the first draft
// had three that no mutation could break.
func TestAHealCutIsWorthTheHealingItDenies(t *testing.T) {
	for _, row := range []struct {
		name   string
		health int64
		set    func(*testing.T, *battle.Battle)
		want   string
	}{
		{
			"an enemy mending itself, with room for it", 1000,
			func(t *testing.T, f *battle.Battle) { carrying(t, f, "f", "mending", 3) },
			"rot",
		},
		{"an enemy with nothing to cut", 1000, nil, "jab"},
		{
			// The room clamp. A unit at full health banks none of its own
			// regeneration, so there is nothing there to deny.
			"an enemy mending itself at full health", 0,
			func(t *testing.T, f *battle.Battle) { carrying(t, f, "f", "mending", 3) },
			"jab",
		},
		{
			// The category. Pending totals every ticking stack without asking
			// which way the sign points, so a cut priced off it would be paid for
			// the POISON somebody else put on.
			"an enemy rotting from a poison rather than mending", 1000,
			func(t *testing.T, f *battle.Battle) { carrying(t, f, "f", "poison", 3) },
			"jab",
		},
		{
			// The share. One stack of `fester` takes four tenths, so a regeneration
			// small enough that four tenths of it is worth less than a jab has to
			// leave the jab as the answer — while the whole of it would not.
			"an enemy mending itself barely", 1000,
			func(t *testing.T, f *battle.Battle) { ticking(t, f, "f", "mending", 1, 10) },
			"jab",
		},
	} {
		fight := squad(t, []string{"jab", "rot"}, []string{"jab"}, []string{"strike"},
			0, 0, row.health)
		if row.set != nil {
			row.set(t, fight)
		}
		if choice := chosen(t, fight); choice.Skill != row.want {
			t.Errorf("against %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}
