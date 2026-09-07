package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// swollenBy is a duel where the ally carries the named traits and nothing else
// is different, so a test can read the whole effect of a trait off one unit.
//
// The enemy is deliberately slower than the ally, because every test below reads
// the board before anybody has acted and a battle that had already moved would
// be answering a different question.
func swollenBy(t *testing.T, traits ...string) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 11, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2000, 800, 400, 120),
			Skills: []string{"strike"}, Passives: traits},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2000, 800, 400, 100),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

// TestATraitThatRaisesHealthRaisesTheMaximum is the reading half of the rule, and
// it is the half that did not work at all until Battle.MaxHP moved off the base
// line.
//
// ⚠️ It compares against the same battle without the trait rather than against a
// number written here, because the raise saturates: a term of +200 does not come
// out as exactly a fifth, and a literal would be a second copy of
// modifier.Set.Stat's arithmetic that could drift away from it. What the test
// asserts is the direction and the fact that the term is read at all — the size
// is modifier's own business and has its own tests.
func TestATraitThatRaisesHealthRaisesTheMaximum(t *testing.T) {
	with := swollenBy(t, "broad")
	without := swollenBy(t)
	raised, plain := unitByID(t, with, "a"), unitByID(t, without, "a")
	if got, want := with.MaxHP(raised), without.MaxHP(plain); got <= want {
		t.Fatalf("a unit holding a health trait has a maximum of %d and one without has %d: "+
			"the term is not being read", got, want)
	}
	// The base line is untouched. A trait is a modifier on top of the authored
	// stats, not a rewrite of them, and a raise that edited Base would survive a
	// dispel that took the trait off.
	if got, want := raised.Base[progression.HP], plain.Base[progression.HP]; got != want {
		t.Errorf("the trait moved the authored health line from %d to %d; a modifier "+
			"sits on top of the stat line rather than replacing it", want, got)
	}
}

// TestAUnitStartsTheBattleFullEvenWhenATraitRaisedItsMaximum is the writing half:
// current health is set after the traits and the bonuses are on.
//
// Without it a unit holding a health trait walks onto the board already short of
// a maximum it never fell from — a squad wounded by the very bonus it built for,
// with no event anywhere saying where the health went.
func TestAUnitStartsTheBattleFullEvenWhenATraitRaisedItsMaximum(t *testing.T) {
	fight := swollenBy(t, "broad")
	unit := unitByID(t, fight, "a")
	if got, want := unit.HP, fight.MaxHP(unit); got != want {
		t.Fatalf("the unit opens on %d of %d health: a battle starts full, and a trait "+
			"that raised the maximum has to raise the health it opens with too", got, want)
	}
	if unit.HP <= unit.Base[progression.HP] {
		t.Errorf("the unit opens on %d health and its authored line is %d: the opening "+
			"health is the authored line rather than the raised maximum",
			unit.HP, unit.Base[progression.HP])
	}
}

// TestAGateReadsFullHealthWhileTheTraitsAreStillGoingOn is the ordering trap, and
// it is an ordering the roster happens to choose.
//
// A gate is a ratio. `nearly_whole` opens under nine tenths of health, and
// `broad` raises the maximum by a fifth — so a unit listing `broad` first would,
// without the refill in Battle.grant, be read as holding 2000 of a raised 2400
// when the second trait's gate was asked, which is five sixths, and would open the
// battle already gated on. Which trait is listed first is authoring order, so the
// bug would be an authoring order deciding whether a trait starts on.
//
// ⚠️ The gate has to be shallower than the raise or the test proves nothing. A
// half-health gate against a raise of a fifth never fires either way, and the
// first version of this test used one and passed with the refill deleted.
//
// ⚠️ Both orders are checked. Only one of them can fail, and which one it is
// depends on an implementation detail, which is exactly why the test may not pick
// one.
func TestAGateReadsFullHealthWhileTheTraitsAreStillGoingOn(t *testing.T) {
	for _, order := range [][]string{{"broad", "nearly_whole"}, {"nearly_whole", "broad"}} {
		fight := swollenBy(t, order...)
		unit := unitByID(t, fight, "a")
		if unit.Statuses.Has("toughened") {
			t.Errorf("listed as %v, the unit opens the battle already gated on: a trait "+
				"that raised the maximum made it read as wounded by the raise", order)
		}
		if got, want := unit.HP, fight.MaxHP(unit); got != want {
			t.Errorf("listed as %v, the unit opens on %d of %d health", order, got, want)
		}
	}
}

// TestARaiseAfterTheOpeningLeavesCurrentHealthWhereItIs is the other half of the
// design, and it is the half nothing in the game can reach yet.
//
// Nothing applies a health term mid-battle: status.ParseBook refuses one on a
// timed status, passive.ParseBook refuses one on a gated trait, and a permanent
// status cannot be applied to anybody by a skill. So this reaches past all three
// and holds the status directly, which is the seam a future feature would arrive
// through.
//
// What it pins is that the room opens and nothing fills it. A maximum that
// dragged current health up with it would be free healing arriving with no heal
// event, and a renderer would show a bar that grew while nobody acted.
func TestARaiseAfterTheOpeningLeavesCurrentHealthWhereItIs(t *testing.T) {
	fight := swollenBy(t)
	unit := unitByID(t, fight, "a")
	kind, err := books(t).Statuses.Lookup("swollen")
	if err != nil {
		t.Fatalf("look up the fixture status: %v", err)
	}
	wasMax, wasHP := fight.MaxHP(unit), unit.HP
	if held := unit.Statuses.Hold(kind, 0, 1); held != 1 {
		t.Fatalf("the status went on %d times, want once", held)
	}
	if got := fight.MaxHP(unit); got <= wasMax {
		t.Fatalf("the maximum is %d and was %d: holding the status changed nothing", got, wasMax)
	}
	if unit.HP != wasHP {
		t.Errorf("current health moved from %d to %d when the maximum rose: a raise opens "+
			"room, and filling it is what healing is for", wasHP, unit.HP)
	}
}
