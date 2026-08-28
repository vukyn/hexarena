package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestTheShippedSummonerCanActuallyPutACloneDown is the data half of the
// summoning work, and it is a different question from the engine half.
//
// internal/core/battle proves a summon arrives against fixtures built for it.
// This proves the shipped skill does, with the shipped books, from a placement
// the roster rules would accept — which is where the mistakes an author makes
// actually live: a formation with no free slot, a summon whose skills the summoned
// element may not carry, a stat share that lands outside the progression limits.
//
// Hand-played, and it stays hand-played now that autopilot casts summons too.
// What this test is about is the skill: driving the cast directly means a failure
// here is the data's, and cannot be Suggest having preferred something else on
// the turn the test happened to look at. TestAutopilotCastsTheShippedSummons
// below is the other question, asked separately.
func TestTheShippedSummonerCanActuallyPutACloneDown(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("cast: %v", err)
	}
	naruto, known := characters.Get("naruto.naruto")
	if !known {
		t.Skip("no summoner is shipped, so there is nothing here to drive")
	}
	values, _, err := naruto.Resolve(60, "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	fight, err := battle.New(books, 4, []battle.Roster{
		{ID: "ninja", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Name: naruto.Name, Affinity: naruto.Element, Stats: values,
			Skills: []string{"shadow_clone", "kunai"}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Name: "sparring", Affinity: mustAffinity(t, "metal"),
			Stats: benchStats(4800, 300, 300, 40), Skills: []string{"bite"}},
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
	if prompt.Unit != "ninja" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	unit, ok := fight.Unit("ninja")
	if !ok {
		t.Fatal("no ninja in the battle")
	}
	if err := fight.Act("shadow_clone", unit.Cell); err != nil {
		t.Fatalf("shadow_clone: %v", err)
	}

	arrived := 0
	for _, event := range fight.Drain() {
		if event.Kind != battle.Summoned {
			continue
		}
		arrived++
		copied, ok := fight.Unit(event.Target)
		if !ok {
			t.Fatalf("the log announced %q and the battle has no such unit", event.Target)
		}
		if copied.Side != hex.SideAlly {
			t.Errorf("%s stands on the %s side", copied.ID, copied.Side)
		}
		// Two fifths of the caster, which is what the skill declares. Checked as
		// a ratio rather than a figure so a retune of Naruto's curve does not
		// have to be repeated here — what this is about is the share arriving,
		// not what the share is worth this week.
		if want := values[0] * 400 / 1000; copied.MaxHP() != want {
			t.Errorf("%s has %d health, want %d — two fifths of the caster's %d",
				copied.ID, copied.MaxHP(), want, values[0])
		}
	}
	// Two, because the skill asks for two and the formation has room: three
	// units on a side of five leaves two slots, and a duel leaves four.
	if arrived != 2 {
		t.Errorf("the shipped clone put %d units on the board, want the two it declares", arrived)
	}
}

// TestAutopilotCastsTheShippedSummons is the opponent's half, and it is a
// different question from whether the skill works.
//
// battle.Suggest used to reach a summoning skill only as its fallback — the
// option taken when nothing at all could be hurt — because a summon has no power
// of its own. So the shipped summoner never called anybody up while it had a
// kunai in reach, and every figure ever measured of it was measured with its own
// mechanism sitting idle. internal/core/battle proves the price against
// fixtures; this proves the shipped skills clear it, with the shipped stat
// curves, which is the half a fixture cannot answer.
//
// ⚠️ It asserts a summon is cast and not which one, or how many turns in. The
// order Naruto opens with is a balance figure and not a rule — it moved once
// already, from nothing at all — and a test naming it would fail on every retune
// of a curve it is not about.
func TestAutopilotCastsTheShippedSummons(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("cast: %v", err)
	}
	naruto, known := characters.Get("naruto.naruto")
	if !known {
		t.Skip("no summoner is shipped, so there is nothing here to drive")
	}
	values, _, err := naruto.Resolve(progression.LevelCap, "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	summoning := make([]string, 0, 2)
	for _, learned := range cast.LearnedIDs(naruto.Skills) {
		declared, err := books.Skills.Lookup(learned)
		if err != nil {
			t.Fatalf("look up %s: %v", learned, err)
		}
		if declared.Summons.Summons() {
			summoning = append(summoning, learned)
		}
	}
	if len(summoning) == 0 {
		t.Skip("the shipped summoner no longer knows a summoning skill")
	}

	// A sparring dummy at the health ceiling with one weak attack, so the battle
	// lasts long enough for every cooldown to come round and nothing about the
	// dummy decides what the ninja reaches for. The loop stops early if it dies
	// anyway — a ninja that killed it before ever summoning is the failure this
	// test is for, and it has to be reported as that rather than as an error
	// from advancing a finished battle.
	fight, err := battle.New(books, 4, []battle.Roster{
		{ID: "ninja", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Name: naruto.Name, Affinity: naruto.Element, Stats: values,
			Skills: append([]string{"kunai"}, summoning...)},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Name: "sparring", Affinity: mustAffinity(t, "metal"),
			Stats: benchStats(4800, 120, 300, 40), Skills: []string{"bite"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	used := make(map[string]bool, len(summoning))
	arrived := 0
	for turn := 0; turn < 40 && !fight.Finished(); turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Skipped {
			continue
		}
		choice, ok := fight.Suggest(prompt)
		if !ok {
			if err := fight.Pass(battle.NoActionReason); err != nil {
				t.Fatalf("pass: %v", err)
			}
			continue
		}
		if prompt.Unit == "ninja" {
			used[choice.Skill] = true
		}
		if err := fight.Act(choice.Skill, choice.Aim); err != nil {
			t.Fatalf("act %s: %v", choice.Skill, err)
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Summoned {
				arrived++
			}
		}
	}
	for _, id := range summoning {
		if !used[id] {
			t.Errorf("autopilot never cast %s in forty turns; a summon priced at nothing "+
				"is a summon the opponent will not use", id)
		}
	}
	if arrived == 0 {
		t.Error("no unit arrived on the board, so nothing was summoned at all")
	}
}
