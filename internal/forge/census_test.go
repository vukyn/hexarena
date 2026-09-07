package forge

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
)

// censusSeeds is how many battles a census fights each arrangement over.
//
// Small, because of what is being asserted: "was this slot ever used" needs one
// cast to answer yes, not a confident rate.
//
// ⚠️ **It is not as small as it can be, and the floor was measured rather than
// guessed.** At two seeds `machop.charge` reads `wrecking_swing` silent on all
// four boards — that skill is gated on five stacks of `heft` and `brace` grants
// them a battle at a time, so its gate is only crossed in a battle that runs long
// enough. A gated slot needs board TIME rather than more boards, which is exactly
// what a seed buys and what another opponent does not.
const censusSeeds = 6

// played is every skill a build cast at least once against any authored squad,
// stopping at the first board that leaves nothing silent.
//
// ⚠️ **The walk is the rule and the census is one board.** A slot silent
// against one squad is a fact about that matchup — `split` is uncast against
// eight of the twenty-one characters and cast four hundred times across the rest
// — so the question a catalogue has to answer is whether a slot fires
// *anywhere*. Stopping early is what keeps that affordable: most builds are done
// after the first board, and only the ones with something to explain pay for the
// rest.
func played(t *testing.T, lib *Library, build cast.Build) ([]string, []CastCensus) {
	t.Helper()
	taken := []CastCensus{}
	silent := []string{}
	for _, against := range lib.Squads() {
		census, err := lib.Census(build, against.ID, censusSeeds)
		if err != nil {
			t.Fatalf("census %s against %s: %v", build.ID, against.ID, err)
		}
		taken = append(taken, census)
		if silent = census.Silent(); len(silent) == 0 {
			return nil, taken
		}
	}
	return silent, taken
}

// TestEveryShippedBuildPlaysItsOwnKit is the authoring rule for the build
// catalogue, held rather than written down twice.
//
// **A build may not name a skill it never casts.** A build is a *decision* about
// four slots, so a slot the opponent never spends a turn on is a decision the
// author did not get — and no rate can show that, which is what let the reading
// behind RAT-006 blame the wrong skill for a whole session.
//
// ⚠️ **The rule is measured rather than counted, and the count proposed instead
// is refused by the shipped data.** RAT-006 asked for "one turn-spending
// self-cast per kit". `bulbasaur.parasite` holds two — `synthesis` and `ingrain`
// — and plays both, because a heal's price moves with the health it is aimed at
// and two such skills therefore take turns; `squirtle.fortress` holds three and
// plays all three. A count would refuse two builds that use every slot they
// name.
func TestEveryShippedBuildPlaysItsOwnKit(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped library: %v", err)
	}
	builds := lib.Builds()
	if len(builds) == 0 {
		t.Fatal("the shipped catalogue holds no builds, so this test asserts nothing")
	}
	for _, build := range builds {
		if silent, taken := played(t, lib, build); len(silent) > 0 {
			t.Errorf("%s never cast %v against any authored squad: a build may not name a "+
				"skill it does not play (%d boards, last %v)",
				build.ID, silent, len(taken), taken[len(taken)-1].Casts)
		}
	}
}

// TestACensusSeesASlotThatCannotFire is the half that stops the test above being
// green for the wrong reason.
//
// Every shipped build passes it, so on its own it might be asserting nothing at
// all — the fixture-shaped mistake this repository keeps a list of. So the
// instrument is handed a kit with a slot that *cannot* fire and has to say so:
// `wrecking_swing` is gated on five stacks of `heft` and `brace` is the only
// thing that grants any, so without it in the kit beside it the skill is
// unusable for a whole battle.
//
// A gate rather than a price, deliberately: a price is a balance figure that
// could move under a re-tune and take this test's premise with it, while a kit
// that cannot fuel its own skill is unusable by construction.
func TestACensusSeesASlotThatCannotFire(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped library: %v", err)
	}
	starved := cast.Build{
		ID: "machop.starved", Character: "pokemon.machop", Name: "no fuel",
		Skills:   []string{"wrecking_swing", "cross_chop", "rock_throw", "seismic_toss"},
		Passives: []string{"berserk"},
	}
	silent, _ := played(t, lib, starved)
	if !slices.Contains(silent, "wrecking_swing") {
		t.Errorf("the census found %v silent in a kit that cannot fuel wrecking_swing", silent)
	}

	// The control, and what makes the reading above about the kit rather than
	// about the skill: the shipped build carries the fuel and plays it.
	fuelled, known := lib.builds.Get("machop.charge")
	if !known {
		t.Fatal("machop.charge is not in the shipped catalogue")
	}
	if !slices.Contains(fuelled.Skills, "wrecking_swing") ||
		!slices.Contains(fuelled.Skills, "brace") {
		t.Fatalf("machop.charge no longer holds the gated skill and its fuel (%v), so the "+
			"control proves nothing", fuelled.Skills)
	}
	if silent, taken := played(t, lib, fuelled); slices.Contains(silent, "wrecking_swing") {
		t.Errorf("the fuelled kit never cast wrecking_swing either, so the starved reading "+
			"is not about the fuel (%v)", taken[len(taken)-1].Casts)
	}
}
