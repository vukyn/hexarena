package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestACleanserEarnsItsSlotWhereASparCannotSeeIt prices the slot, and the slot
// template is the right instrument for that -- which is worth saying, because
// the same template is the wrong one for a pairing and this repository now
// carries a note about that too.
//
// Blissey reads 0 per mille against every shipped opponent in a spar, and the
// event log says why rather than leaving it to be guessed: over twenty turns it
// used fourteen skills and SKIPPED SIX, because half its kit is worth nothing
// with nobody beside it. `heal_bell` cleanses a column of allies and a duel has
// none, `defense_curl` buys a buff the rating prices at nothing, so the AI
// declines the turn rather than spending a cooldown on either. A duel cannot
// price a unit whose kit is aimed at a squad.
//
// ⚠️ **What fixed it was the kit and not the stat line.** The first draft read
// 133 per mille against the slugger, because a cleanse answers a threat that may
// not be there -- 381 against the blighter, which brings one, and 133 against the
// slugger, which does not. Pushing attack until the squad won reached the floor
// at 660, which is a bruiser's attack on a character whose whole shape is that it
// has none. Trading a self-buff the rating prices at nothing for `safeguard`, the
// first column-wide absorb in the book, reached it at the stats already written:
// 526 and 478 per mille, against a mender's 525 and 543.
//
// So the question is asked the way it is decided: the same striker and the same
// wall in two squads, differing only in the third slot, fought both ways round.
func TestACleanserEarnsItsSlotWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	cleanser := aSquadOf("with-cleanser", aThirdMember("pokemon.happiny",
		"egg_bomb", "safeguard", "heal_bell", "soft_boiled"))
	const floor = 450
	for _, against := range []struct {
		name  string
		squad placement.Squad
	}{
		{"a slugger", aSquadOf("with-slugger", aThirdMember("pokemon.machop",
			"rock_throw", "body_slam", "cross_chop", "vital_throw"))},
		{"a blighter", aSquadOf("with-blighter", aThirdMember("pokemon.bulbasaur",
			"vine_whip", "razor_leaf", "poison_powder", "venoshock"))},
	} {
		t.Run(against.name, func(t *testing.T) {
			wins, losses, endless := fightSquads(t, books, characters, cleanser, against.squad)
			if endless > 0 {
				t.Errorf("%d of %d battles never finished, so the rest are a reading of the ones that did",
					endless, menderSeeds*2)
			}
			decided := wins + losses
			if decided == 0 {
				t.Fatal("no battle was decided, so there is no rate to read")
			}
			rate := wins * 1000 / decided
			t.Logf("the cleanser's squad against %s: %d per mille (%d-%d)", against.name, rate, wins, losses)
			if rate < floor {
				t.Errorf("the cleanser's squad reads %d per mille against %s, under the floor of %d: "+
					"a slot a striker holds better is a slot the cleanser should not be in",
					rate, against.name, floor)
			}
		})
	}
}
