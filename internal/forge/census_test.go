package forge

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
)

// played is every skill a build cast at least once against any authored squad,
// stopping at the first board that leaves nothing silent.
//
// ⚠️ **The walk moved into the package**, because `hexforge census` asks the same
// question and a rule worded twice is the mistake this repository keeps a list
// of. The reasoning for the walk, and for the seed count it is taken at, is on
// Library.CensusWalk and CensusSeeds; what is left here is the t.Fatalf.
func played(t *testing.T, lib *Library, build cast.Build) ([]string, []CastCensus) {
	t.Helper()
	walk, err := lib.CensusWalk(build, CensusSeeds)
	if err != nil {
		t.Fatalf("census %s: %v", build.ID, err)
	}
	return walk.Silent, walk.Taken
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

// TestACensusWalkStopsAtTheFirstBoardThatPlaysEverySlot is the affordability
// half, and it is asserted because nothing else would notice it going.
//
// A walk that kept measuring after the answer was in would still report the same
// verdict — Silent is empty either way — while costing every remaining board.
// Most builds are done after the first one, so that is most of the cost of the
// catalogue test.
func TestACensusWalkStopsAtTheFirstBoardThatPlaysEverySlot(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped library: %v", err)
	}
	squads := lib.Squads()
	if len(squads) < 2 {
		t.Skip("one authored squad, so there is no early stop to observe")
	}
	build, known := lib.builds.Get("machop.charge")
	if !known {
		t.Fatal("machop.charge is not in the shipped catalogue")
	}
	walk, err := lib.CensusWalk(build, CensusSeeds)
	if err != nil {
		t.Fatalf("census walk: %v", err)
	}
	if !walk.Plays() {
		t.Fatalf("%s left %v silent, so this test is not measuring the stop", build.ID, walk.Silent)
	}
	if walk.Boards() == len(squads) {
		t.Errorf("the walk took all %d boards on a build that plays its kit; it should have "+
			"stopped at the first board that left nothing silent", len(squads))
	}
	// The board it stopped on is the last one taken, and it is the one that
	// settled the question. Said out loud because a reader of the report takes
	// the verdict off the set and the last table off this.
	last := walk.Taken[len(walk.Taken)-1]
	if len(last.Silent()) != 0 {
		t.Errorf("the walk stopped on %s, which left %v silent", last.Against, last.Silent())
	}
}

// TestACensusWalkOverAKitThatCannotFireTriesEveryBoard is the other arm: a slot
// that can never fire is a slot no board redeems, so the walk pays for all of
// them and says which slot it was.
func TestACensusWalkOverAKitThatCannotFireTriesEveryBoard(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped library: %v", err)
	}
	starved := cast.Build{
		ID: "machop.starved", Character: "pokemon.machop", Name: "no fuel",
		Skills:   []string{"wrecking_swing", "cross_chop", "rock_throw", "seismic_toss"},
		Passives: []string{"berserk"},
	}
	walk, err := lib.CensusWalk(starved, CensusSeeds)
	if err != nil {
		t.Fatalf("census walk: %v", err)
	}
	if walk.Plays() {
		t.Fatal("a kit that cannot fuel wrecking_swing was reported as playing every slot")
	}
	if !slices.Contains(walk.Silent, "wrecking_swing") {
		t.Errorf("the walk found %v silent, want the gated slot among them", walk.Silent)
	}
	if walk.Boards() != len(lib.Squads()) {
		t.Errorf("the walk took %d of %d boards on a build that never plays its kit: nothing "+
			"redeemed it, so there was no board to stop on", walk.Boards(), len(lib.Squads()))
	}
}
