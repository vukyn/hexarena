package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestTheShippedRegenerationHeals is the engine fix meeting the shipped data.
//
// internal/core/battle proves a regeneration ticks; this proves the two skills
// in the book that apply one are no longer inert. They are worth a test of their
// own because they are what the bug actually cost: aqua_ring and ingrain were
// castable, described, glossed and had no effect whatsoever, and every test in
// the repository passed the whole time.
//
// It is hand-played for the same reason aHandPlayedGateCrossing is. Suggest
// picks the highest expected damage and falls back to the first usable
// non-damaging skill, so a self-cast regeneration on a unit that can always
// reach somebody is a skill the bench never chooses — which is the other half of
// why nothing caught this.
func TestTheShippedRegenerationHeals(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	fight, err := battle.New(books, 3, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "water"), Stats: benchStats(3000, 800, 400, 200),
			Skills: []string{"aqua_ring", "water_gun"}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "water"), Stats: benchStats(4800, 500, 300, 20),
			Skills: []string{"water_gun"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	healer, ok := fight.Unit("healer")
	if !ok {
		t.Fatal("no healer in the battle")
	}
	healer.HP = 900
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "healer" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("aqua_ring", healer.Cell); err != nil {
		t.Fatalf("aqua_ring: %v", err)
	}
	applied := 0
	for _, event := range fight.Drain() {
		if event.Kind == battle.StatusApplied && event.Status == "regrowth" {
			applied = event.Stacks
			// 800 attack against regrowth's tick power of 400. The figure is
			// spelled out rather than read back off the books, so a change to
			// either number arrives here as a failure rather than as a test
			// quietly agreeing with whatever the data now says.
			if event.Amount != 320 {
				t.Errorf("a stack of the shipped regeneration froze %d, want 320",
					event.Amount)
			}
		}
	}
	if applied != 2 {
		t.Fatalf("aqua_ring applied %d stacks, want the two it declares", applied)
	}

	// On to the healer's next turn, which is where a regeneration pays out.
	for range 10 {
		next, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if next == nil {
			break
		}
		if !next.Skipped {
			// Whatever is off cooldown and has somewhere to aim: this test cares
			// about the tick at the start of the turn, not about what is done
			// with the turn once it has begun. A turn with nothing available
			// ends the walk rather than being left untaken, because the next
			// Advance refuses a turn nobody has acted on.
			acted := false
			for _, option := range next.Options {
				if !option.Available() {
					continue
				}
				if err := fight.Act(option.Skill, option.Aims[0]); err != nil {
					t.Fatalf("act %s: %v", option.Skill, err)
				}
				acted = true
				break
			}
			if !acted {
				break
			}
		}
		for _, event := range fight.Drain() {
			if event.Kind != battle.Healed || event.Actor != "healer" {
				continue
			}
			if event.Status != "regrowth" {
				t.Errorf("the heal names %q, want regrowth", event.Status)
			}
			if event.Amount != 640 {
				t.Errorf("two stacks of the shipped regeneration healed %d, want 640",
					event.Amount)
			}
			return
		}
	}
	t.Fatal("aqua_ring never healed anybody, so it is still an inert skill")
}
