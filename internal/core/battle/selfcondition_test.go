package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// spendingBooks is the shared books with a skill book written for these tests:
// a plain hit, a column that spends a status, and a single-target one that reads
// its caster's health.
func spendingBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	spending, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"psych","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"haste","chance":1000,"stacks":1}]},
	  {"id":"sweep","element":"neutral","range":1,"pattern":"column",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"haste","min_stacks":1,"bonus_power":900,"consume":true}},
	  {"id":"cornered","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"below_health":500,"bonus_power":900}}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = spending
	return shared
}

// TestASpentStatusIsPaidForOnceHoweverManyItHits is the whole reason the caster's
// condition is read in Act and not in resolveAgainst.
//
// resolveAgainst runs once per cell a shape covers. A condition consumed there
// would charge a column three times and a single-target skill once, for a
// difference written on neither skill — and the author who wrote "spends a haste"
// would have written something that costs whatever the shape happens to catch.
func TestASpentStatusIsPaidForOnceHoweverManyItHits(t *testing.T) {
	fight, err := battle.New(spendingBooks(t), 4, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 800, 400, 200),
			Skills: []string{"psych", "sweep"}},
		{ID: "top", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4000, 60, 400, 1),
			Skills: []string{"jab"}},
		{ID: "bottom", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 60, 400, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	// One haste on, then one sweep that catches both of them.
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("psych", hex.Offset{Col: 2, Row: 1}); err != nil {
		t.Fatalf("psych: %v", err)
	}
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("sweep", hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	consumed, amplified, bitten := 0, 0, map[string]bool{}
	for _, event := range fight.Drain() {
		switch event.Kind {
		case battle.StatusConsumed:
			consumed++
		case battle.Amplified:
			amplified++
		case battle.Damaged:
			bitten[event.Target] = true
		}
	}
	if len(bitten) != 2 {
		t.Fatalf("the sweep hit %d units, so it never covered a column: %v", len(bitten), bitten)
	}
	if consumed != 1 {
		t.Errorf("a status spent once was consumed %d times, one per unit the shape caught", consumed)
	}
	if amplified != 1 {
		t.Errorf("the caster's own condition was announced %d times for one use", amplified)
	}
	// And it is really gone. Counting the consume once would also pass if the
	// status were never taken off at all, so the second sweep is what tells
	// "spent once" from "spent never".
	waitFor(t, fight, "caster")
	if err := fight.Act("sweep", hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	for _, event := range fight.Drain() {
		if event.Kind == battle.Amplified || event.Kind == battle.StatusConsumed {
			t.Errorf("the status was still there to spend a second time: %+v", event)
		}
	}
}

// waitFor advances until the named unit is the one being asked, passing for
// everybody else so the battle keeps moving.
func waitFor(t *testing.T, fight *battle.Battle, id string) {
	t.Helper()
	for turn := 0; turn < 100; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if !prompt.Skipped {
			if prompt.Unit == id {
				return
			}
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		fight.Drain()
	}
	t.Fatalf("%s was never offered another turn", id)
}

// TestBothTargetsOfASpendingSkillGetTheBonus is the other half of paying once:
// the point of a shared cost is that the whole shape is bought with it.
func TestBothTargetsOfASpendingSkillGetTheBonus(t *testing.T) {
	dealt := func(withHaste bool) map[string]int64 {
		t.Helper()
		fight, err := battle.New(spendingBooks(t), 4, []battle.Roster{
			{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 800, 400, 200),
				Skills: []string{"psych", "sweep"}},
			{ID: "top", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
				Affinity: single("neutral"), Stats: stats(4000, 60, 400, 1),
				Skills: []string{"jab"}},
			{ID: "bottom", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 60, 400, 1),
				Skills: []string{"jab"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		fight.Drain()
		if withHaste {
			if _, err := fight.Advance(); err != nil {
				t.Fatalf("advance: %v", err)
			}
			if err := fight.Act("psych", hex.Offset{Col: 2, Row: 1}); err != nil {
				t.Fatalf("psych: %v", err)
			}
			fight.Drain()
		}
		if _, err := fight.Advance(); err != nil {
			t.Fatalf("advance: %v", err)
		}
		if err := fight.Act("sweep", hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})); err != nil {
			t.Fatalf("sweep: %v", err)
		}
		out := map[string]int64{}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Damaged {
				out[event.Target] += event.Amount
			}
		}
		return out
	}

	plain, spent := dealt(false), dealt(true)
	for _, who := range []string{"top", "bottom"} {
		if plain[who] == 0 {
			t.Fatalf("%s took nothing without the bonus, so this compares nothing", who)
		}
		if spent[who] <= plain[who] {
			t.Errorf("%s took %d with the status spent and %d without, so the bonus reached only part of the shape",
				who, spent[who], plain[who])
		}
	}

	// And it is shared the way the rest of the power is. "bottom" is the cell
	// the skill was aimed at and "top" is the edge of the column, so the edge
	// takes the splash share of everything -- base and bonus alike.
	//
	// Asserted as a ratio rather than as a figure because the alternative is
	// invisible any other way: a bonus added after the splash reduction still
	// makes both targets take more, which is all the loop above checks, and the
	// edge would quietly be taking a full share of it.
	for name, took := range map[string]map[string]int64{"without the bonus": plain, "with it spent": spent} {
		primary, edge := took["bottom"], took["top"]
		if doubled := edge * 2; doubled < primary-2 || doubled > primary+2 {
			t.Errorf("%s the edge of the column took %d against the aim's %d, "+
				"which is not the splash share of the same power", name, edge, primary)
		}
	}
}

// TestASelfConditionReadsTheCastersHealthAndNotTheTargets is the mirror of
// TestAConditionReadsTheTargetsHealthAndNotTheCasters, and it exists for exactly
// the same reason.
//
// Two units have health and reading the wrong one compiles, runs, and produces a
// skill that amplifies at the wrong moment. Here the *target* is left nearly
// dead and the caster is untouched: a reading taken from the target would fire
// every time, and a correct one never fires at all.
func TestASelfConditionReadsTheCastersHealthAndNotTheTargets(t *testing.T) {
	fight, err := battle.New(spendingBooks(t), 7, []battle.Roster{
		{ID: "whole", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 800, 400, 200),
			Skills: []string{"cornered", "strike"}},
		{ID: "dying", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(400, 60, 400, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("cornered", hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})); err != nil {
		t.Fatalf("cornered: %v", err)
	}
	for _, event := range fight.Drain() {
		if event.Kind == battle.Amplified {
			t.Errorf("a condition on the caster's own health fired while the caster was untouched "+
				"and only the target was hurt: %+v", event)
		}
	}
}

// TestTheOpponentRatesASpendingSkillAtWhatItWillActuallyLandFor.
//
// Suggest rates a skill by the power it would land and the engine then lands it.
// A rating built from a different reading would make the opponent prefer a skill
// for a bonus it does not get, and nothing would report the disagreement — which
// is the same trap conditionTarget was written to close, arriving through the
// other door.
func TestTheOpponentRatesASpendingSkillAtWhatItWillActuallyLandFor(t *testing.T) {
	fight, err := battle.New(spendingBooks(t), 3, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 800, 400, 200),
			Skills: []string{"psych", "sweep", "strike"}},
		{ID: "foe", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 60, 400, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	// Without the status, sweep is a hundred power against strike's thousand, so
	// the opponent must not reach for it.
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if choice, ok := fight.Suggest(prompt); !ok || choice.Skill == "sweep" {
		t.Errorf("with nothing to spend, the opponent chose %q over a skill ten times its power", choice.Skill)
	}
	if err := fight.Act("psych", hex.Offset{Col: 2, Row: 1}); err != nil {
		t.Fatalf("psych: %v", err)
	}
	fight.Drain()

	// With it, sweep lands for a thousand and is the better of the two.
	prompt, err = fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("the opponent had nothing to suggest")
	}
	if choice.Skill != "sweep" {
		t.Errorf("with a status to spend the opponent chose %q, so it rates the skill at its unspent power",
			choice.Skill)
	}
}
