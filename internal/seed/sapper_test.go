package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// aSquadBesides builds the three-slot squad of TestASapperIsWorthMoreBesideTheRot:
// one fixed striker, one second member the comparison varies deliberately, and a
// third member that is the slot under test.
//
// It is a second builder rather than a parameter on aSquadOf because the two ask
// different questions. aSquadOf holds the pair constant to price a slot; this one
// changes the pair on purpose, because what is being priced is not the slot but
// whether the slot is worth more next to one partner than another.
func aSquadBesides(id string, partner, third placement.Placement) placement.Squad {
	return placement.Squad{
		ID: id,
		Units: []placement.Placement{
			{ID: "fire", Character: "pokemon.charmander", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"flamethrower", "fire_spin", "ember", "inferno"},
				Passives: []string{"blaze"}},
			partner,
			third,
		},
	}
}

func aPartner(character string, kit ...string) placement.Placement {
	return placement.Placement{
		ID: "partner", Character: character, Level: progression.LevelCap,
		Stage: progression.Furthest, Slot: hex.Offset{Col: 1, Row: 1},
		Skills: kit,
	}
}

// TestASapperIsWorthMoreBesideTheRot is the pairing claim, and it is asked as a
// difference because a rate cannot carry it.
//
// ⚠️ **Three measurement designs failed before this one, each for its own
// reason, and each looked like an answer.** Pricing the slot against a striker
// measured what the squad's other two slots were short of, which is a fact about
// squads and not about poison. Pricing it beside two different partners measured
// the partners. And holding the partner constant across both squads — the shape
// TestAMenderEarnsItsSlotWhereASparCannotSeeIt uses, correctly, to price a slot —
// hands the same synergy to the opponent, so it cancels and reads nought. A
// mirror measures nothing; that rule reaches squad composition too.
//
// What isolates it is one skill. The same character stands in both squads with
// the same kit but for a single slot: poison_powder against leech_seed, both
// powders of no power, so what differs is whether the partner poisons and
// nothing else. Measured 636 per mille, and 575 when venoshock stands in both
// kits — higher without it, so the edge is the rot compounding rather than the
// blighter's own cash-in.
func TestASapperIsWorthMoreBesideTheRot(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	sapper := func() placement.Placement {
		return aThirdMember("pokemon.oddish", "acid", "mega_drain", "toxic", "acid_spray")
	}
	poisons := aSquadBesides("partner-poisons", aPartner("pokemon.bulbasaur",
		"vine_whip", "razor_leaf", "poison_powder", "synthesis"), sapper())
	doesNot := aSquadBesides("partner-does-not", aPartner("pokemon.bulbasaur",
		"vine_whip", "razor_leaf", "leech_seed", "synthesis"), sapper())
	wins, losses, endless := fightSquads(t, books, characters, poisons, doesNot)
	if endless > 0 {
		t.Errorf("%d of %d battles never finished, so the rest are a reading of the ones that did",
			endless, menderSeeds*2)
	}
	decided := wins + losses
	if decided == 0 {
		t.Fatal("no battle was decided, so there is no rate to read")
	}
	rate := wins * 1000 / decided
	t.Logf("the sapper beside a partner that poisons, against the same squad whose partner does not: "+
		"%d per mille (%d-%d)", rate, wins, losses)
	// The floor is a long way under the reading on purpose. What is held is the
	// claim — a second source of rot is worth more to this character than the same
	// slot spent on anything else its partner knows — and not the figure, which
	// moves with every one of the three characters standing in the squad.
	const floor = 560
	if rate < floor {
		t.Errorf("the sapper's squad reads %d per mille when its partner poisons, under the floor of %d: "+
			"the second source of rot buys nothing, so the pairing is a story rather than a mechanism",
			rate, floor)
	}
}
