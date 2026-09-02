package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// shapeMargin is how much of the gap between the two kits is held on a board
// with nothing standing in the way of the shape.
//
// The reading it was written against is 353 against 281 per mille over six
// hundred battles, so thirty leaves the claim room to move with every character
// that is ever added to the squad around it while still refusing a shape that
// bought nothing. What is held is that the shape is worth something, not that it
// is worth seventy-two.
const shapeMargin = 30

// TestAShapeEarnsItsPowerWhereASparCannotSeeIt prices the thing the bombardier
// preset is, against the tool that is structurally blind to it — and then prices
// the answer to it, which is the half this test did not have.
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
//
// ⚠️⚠️ **A WALL STANDING IN THE COLUMN IS THE ANSWER TO A SHAPE, and this test
// used to hide that.** Its opposition carries `withdraw`, and until the rating
// could see a block charge the shape read as winning anyway. It cannot: a charge
// cancels the strike on the body it is standing on, so a column that caught two
// bodies catches one — and a single-target skill simply aims somewhere else,
// because it was never anchored. Measured, the same swap on the same squads:
//
//	opposition's wall     shaped   pointed   the shape is worth
//	carries `withdraw`       286       485                 -199
//	carries an attack        456       400                  +56
//
// So the claim is now BOTH rows. The shape earns its slot where nothing cancels
// it — which is what the preset is for — and it is answered by a wall in the
// column, which is what makes bringing one a decision rather than a formality.
// A test holding only the first row would be a test that could not tell a shape
// from a shape nobody had thought to counter.
func TestAShapeEarnsItsPowerWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}

	shaped := aSquadOf("with-the-shape", aThirdMember("pokemon.magnemite",
		"thunder_shock", "discharge", "thunderbolt", "flash_cannon"))
	pointed := aSquadOf("without-it", aThirdMember("pokemon.magnemite",
		"thunder_shock", "mirror_shot", "thunderbolt", "flash_cannon"))

	for _, board := range []struct {
		name    string
		wallKit []string
		cancels bool
	}{
		{"nothing in the way of it", []string{"water_gun", "bubble", "bite", "brine"}, false},
		{"a wall standing in the column", []string{"water_gun", "bubble", "bite", "withdraw"}, true},
	} {
		// The opposition, held fixed but for the wall's fourth slot. Three bodies
		// rather than one is the whole point: two of them stand in the same
		// formation column, so a column pattern has something to catch and the
		// single-target kit is not being punished for a board it could never have
		// used.
		opposition := placement.Squad{ID: "plain", Units: []placement.Placement{
			{ID: "fire", Character: "pokemon.charmander", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"flamethrower", "fire_spin", "ember", "inferno"},
				Passives: []string{"blaze"}},
			{ID: "wall", Character: "pokemon.squirtle", Level: progression.LevelCap,
				Slot: hex.Offset{Col: 1, Row: 1}, Skills: board.wallKit,
				Passives: []string{"endurance"}},
			aThirdMember("pokemon.machop",
				"rock_throw", "body_slam", "cross_chop", "vital_throw"),
		}}

		// Each kit against the opposition, never against each other. Two kits of
		// one character fought head to head measure the twin rather than the kit —
		// the same reason every build reading in this package is taken against a
		// fixed opponent.
		withShape := rateAgainst(t, books, characters, shaped, opposition)
		without := rateAgainst(t, books, characters, pointed, opposition)
		t.Logf("with %s: the shaped kit reads %d per mille, the pointed one %d",
			board.name, withShape, without)

		if board.cancels {
			if withShape >= without {
				t.Errorf("with %s the shaped kit reads %d against the pointed kit's %d: "+
					"a charge cancels the strike on the body it is standing on, so a "+
					"column that caught two catches one — and if a wall does not answer "+
					"a shape, nothing does",
					board.name, withShape, without)
			}
			continue
		}
		if withShape-without < shapeMargin {
			t.Errorf("with %s the swap moved the squad from %d to %d per mille, under the %d held: "+
				"a pattern that buys nothing makes the preset a slugger with worse numbers",
				board.name, without, withShape, shapeMargin)
		}
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
