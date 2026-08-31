package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/seed"
)

// shapeMargin is how much of the gap between the two kits is held.
//
// The reading it was written against is 353 against 281 per mille over six
// hundred battles, so thirty leaves the claim room to move with every character
// that is ever added to the squad around it while still refusing a shape that
// bought nothing. What is held is that the shape is worth something, not that it
// is worth seventy-two.
const shapeMargin = 30

// TestAShapeEarnsItsPowerWhereASparCannotSeeIt prices the thing the bombardier
// preset is, against the tool that is structurally blind to it.
//
// `hexforge spar` fights one unit against one unit, and a skill whose value is
// landing on more than it was pointed at has nowhere to spend that in a duel: a
// column pattern with nobody in the column is a single-target skill that paid
// for a shape it did not get. It is the mender's blind spot arriving from the
// other direction — TestAMenderEarnsItsSlotWhereASparCannotSeeIt asks whether a
// *character* holds a slot, and this asks whether a *shape* is worth what it
// costs, which is a question about two kits rather than two characters.
//
// So one skill is swapped and nothing else is. Both kits are Magnemite's, in the
// same squad, against the same opposition, over the same seeds; the fourth slot
// holds `mirror_shot` in one and `discharge` in the other, and the two are the
// same power. The shaped one is the *worse* deal on every figure a reader can
// see — a longer cooldown, no status, and it is the one skill in the shipped book
// aimed at both sides, so it lands on the caster's own squad as well. Whatever it
// wins by, it wins with the shape alone.
//
// ⚠️ **The shape is worth less than it looks, and the first cut of this test
// measured the wrong thing.** It compared a kit of two shaped debuffs against a
// kit of two heavy single blows and read 101 against 295 per mille — a spread
// that lost three times over. That reading was true and the conclusion drawn
// from it would have been false: what it priced was two turns spent doing no
// damage, not the shape. Against a single-target skill of the *same* power the
// shape wins; against one of nearly twice the power (`flash_cannon`) it still
// loses. A splash lands at half, so a column caught twice is worth one and a
// half hits, and that is the whole of what a pattern buys.
func TestAShapeEarnsItsPowerWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}

	// The opposition, held fixed. Three bodies rather than one is the whole
	// point: two of them stand in the same formation column, so a column pattern
	// has something to catch and the single-target kit is not being punished for
	// a board it could never have used.
	opposition := aSquadOf("plain", aThirdMember("pokemon.machop",
		"rock_throw", "body_slam", "cross_chop", "vital_throw"))

	shaped := aSquadOf("with-the-shape", aThirdMember("pokemon.magnemite",
		"thunder_shock", "discharge", "thunderbolt", "flash_cannon"))
	pointed := aSquadOf("without-it", aThirdMember("pokemon.magnemite",
		"thunder_shock", "mirror_shot", "thunderbolt", "flash_cannon"))

	// Each kit against the opposition, never against each other. Two kits of one
	// character fought head to head measure the twin rather than the kit — the
	// same reason every build reading in this package is taken against a fixed
	// opponent.
	withShape := rateAgainst(t, books, characters, shaped, opposition)
	without := rateAgainst(t, books, characters, pointed, opposition)
	t.Logf("the shaped kit reads %d per mille, the pointed one %d", withShape, without)

	if withShape-without < shapeMargin {
		t.Errorf("swapping a single-target skill for a shaped one of the same power moved the squad from %d to %d per mille, under the %d held: "+
			"a pattern that buys nothing makes the preset a slugger with worse numbers",
			without, withShape, shapeMargin)
	}
}

// rateAgainst is fightSquads read as a rate, and it refuses to answer at all
// when a battle failed to finish rather than quietly reporting the ones that
// did.
func rateAgainst(t *testing.T, books battle.Books, characters *cast.Book,
	home, away placement.Squad) int {
	t.Helper()
	wins, losses, endless := fightSquads(t, books, characters, home, away)
	if endless > 0 {
		t.Fatalf("%s: %d of %d battles never finished, so a rate over the rest is a reading of a different question",
			home.ID, endless, menderSeeds*2)
	}
	decided := wins + losses
	if decided == 0 {
		t.Fatalf("%s: no battle was decided, so there is no rate to read", home.ID)
	}
	return wins * 1000 / decided
}
