package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
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
// Hand-played, because Suggest takes the highest expected damage and falls back
// to the first usable non-damaging skill, so a skill whose whole effect is
// putting somebody on the board is one autopilot reaches for last. That is a gap
// in the opponent rather than in the skill — see the roadmap's *A deeper
// opponent* — and it is why this drives the cast itself.
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
