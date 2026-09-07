package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
)

// `heal_cut` had no twin: a status could take a share off the healing its holder
// receives and nothing could add one. The arithmetic was never the obstacle —
// battle.healingFor computes `scale.Base + share` and clamps at nought, so a
// positive share has always raised a heal — the parser simply refused to let any
// category carry one.
//
// ⚠️ It is a category rather than a relaxed bound on `heal_cut`, because a
// category prints as a PREDICATE on the statuses reference: `heal_cut` reads
// "cuts healing received", and a status that raised healing under that heading
// would be a lie on screen. Taunt and HealCut each paid for that lesson already.

// TestAHealMendRaisesTheHealingItsHolderReceives is the effect, read off the
// board rather than off the arithmetic.
func TestAHealMendRaisesTheHealingItsHolderReceives(t *testing.T) {
	plain := healedWith(t, nil)
	mended := healedWith(t, []string{"attune"})
	if plain <= 0 {
		t.Fatal("the control healed nothing, so there is no heal for the status to raise")
	}
	if mended <= plain {
		t.Errorf("a heal_mend holder was healed %d against a plain %d: the share is not "+
			"reaching the payout", mended, plain)
	}
	// Half again, which is the fixture's own share: the assertion is the
	// arithmetic and not a direction, so a status that raised healing by one point
	// would fail here rather than pass as "more".
	if want := plain * 3 / 2; mended != want {
		t.Errorf("a five-hundred share healed %d, want %d — half again on %d",
			mended, want, plain)
	}
}

// healedWith runs one unit through a self-heal, optionally attuning first, and
// reports how much health the heal actually put back.
func healedWith(t *testing.T, before []string) int64 {
	t.Helper()
	kit := append(append([]string{}, before...), "tonic")
	fight, err := battle.New(books(t), 3, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60), Skills: kit},
		{ID: "other", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	healer, _ := fight.Unit("healer")
	// Hurt deeply on purpose: a heal is clamped at the room there is, so a nearly
	// full unit would measure the clamp rather than the share.
	healer.HP = 500
	for _, id := range kit {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "healer" {
			t.Fatalf("%s is up and this fixture needs the healer to be", prompt.Unit)
		}
		was := healer.HP
		if err := fight.Act(id, healer.Cell); err != nil {
			t.Fatalf("act %s: %v", id, err)
		}
		if id == "tonic" {
			return healer.HP - was
		}
	}
	t.Fatal("the heal was never cast")
	return 0
}

// TestAHealMendIsWorthSomethingToTheRating is the half a status like this is
// most likely to be shipped without.
//
// A category whose whole effect is a number `price.go` does not read prices at
// **nought**, and a rating that prices a turn at nought never spends one on it —
// which is exactly how the burrow mechanic shipped invisible. The two heal
// categories therefore have two arms in two switches: `heal_cut` is harm and
// reaches `inflictedOn`, `heal_mend` is a gift and reaches `granted`.
//
// ⚠️ **The regeneration is applied directly rather than cast**, and the kit is cut
// to two skills on purpose. The first version of this test gave the healer a
// regen skill as well and asserted the rating preferred the boost; it chose the
// regen, which proves nothing — a preference measures every other term in the
// file at the same time. What is asked here is the narrowest question the
// external package can ask: between a heal boost on a deeply hurt unit with
// healing owed, and a strike, the boost wins. A deleted arm makes it nought and
// the strike wins.
func TestAHealMendIsWorthSomethingToTheRating(t *testing.T) {
	loaded := books(t)
	fight, err := battle.New(loaded, 3, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60),
			Skills: []string{"attune", "jab"}},
		{ID: "other", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	healer, _ := fight.Unit("healer")
	healer.HP = 500

	// The healing the boost acts on. A target with nothing owed is correctly
	// worth nothing, so a fixture that forgot this would pass against a deleted
	// price.
	regen, err := loaded.Statuses.Lookup("mending")
	if err != nil {
		t.Fatalf("look up the regeneration: %v", err)
	}
	for range 3 {
		healer.Statuses.Apply(regen, 400)
	}
	if healer.Statuses.PendingIn(status.Regen) <= 0 {
		t.Fatal("no healing is owed, so the boost is correctly worth nothing and this " +
			"test would pass against a deleted price")
	}

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "healer" {
		t.Fatalf("%s is up and this fixture needs the healer to be", prompt.Unit)
	}
	choice, acted := fight.Suggest(prompt)
	if !acted {
		t.Fatal("the rating passed a turn it had skills for")
	}
	if choice.Skill != "attune" {
		t.Errorf("the rating chose %q over the heal boost on a deeply hurt unit with "+
			"healing owed: a heal_mend is being priced at nothing", choice.Skill)
	}
}
