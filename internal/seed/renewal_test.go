package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// aCleanserHolding is the cleanser's slot with one named trait on it, because
// the trait is what this measures and nothing else in the suite fields one: a
// spar brings the FIRST trait a learnset declares and the squad tests bring
// none at all, so a trait late in a learnset ships unmeasured unless a test
// names it.
func aCleanserHolding(trait string) placement.Placement {
	return placement.Placement{
		ID: "third", Character: "pokemon.happiny", Level: progression.LevelCap,
		Stage: progression.Furthest, Slot: hex.Offset{Col: 0, Row: 1},
		Skills:   []string{"egg_bomb", "safeguard", "heal_bell", "soft_boiled"},
		Passives: []string{trait},
	}
}

// TestARenewedBuffIsWorthAboutWhatATraitIsWorth holds the renewal in a band
// rather than at a figure, because what is being asked is whether a trait that
// puts a timed status back every turn prices like a trait at all.
//
// ⚠️ **The first status tried was the wrong one, and the measurement said so
// before anything shipped.** Renewing `regrowth` on a body carrying the largest
// health pool in the cast read 838 per mille and left 62 of 600 battles
// unfinished -- a fountain nothing could drink dry, which is the same failure a
// mender's healing has caused twice before. Gating it under half health and
// dropping the chance to a quarter still left six. A renewed buff that does not
// heal has no such problem: `fury` finished every battle, because a trait that
// ends battles faster cannot stall one, and read 745 -- above every trait this
// character can hold. `veil` sits where a trait should.
func TestARenewedBuffIsWorthAboutWhatATraitIsWorth(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	const (
		floor   = 380
		ceiling = 620
	)
	readings := map[string]int{}
	for _, trait := range []string{"convalescence", "endurance", "composure"} {
		mine := aSquadOf("holding-"+trait, aCleanserHolding(trait))
		theirs := aSquadOf("against-a-slugger", aThirdMember("pokemon.machop",
			"rock_throw", "body_slam", "cross_chop", "vital_throw"))
		wins, losses, endless := fightSquads(t, books, characters, mine, theirs)
		if wins+losses == 0 {
			t.Fatalf("%s: no battle was decided", trait)
		}
		rate := wins * 1000 / (wins + losses)
		readings[trait] = rate
		t.Logf("%-14s %d per mille (%d-%d), %d endless", trait, rate, wins, losses, endless)
	}
	got := readings["convalescence"]
	if got < floor || got > ceiling {
		t.Errorf("the renewal reads %d per mille against a band of %d to %d, which the two traits "+
			"beside it (%d and %d) are inside: a renewed buff should price like a trait",
			got, floor, ceiling, readings["endurance"], readings["composure"])
	}
}
