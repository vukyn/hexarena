package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The design record for the three characters authored after builds.json existed.
//
// Hardcoded on purpose, exactly as poisonBuild and its five siblings are: these
// are the kits the tests below measured, and TestTheShippedBuildsAreTheOnesTheTestsMeasure
// is what stops the catalogue a player reads drifting away from them.
var (
	flurryBuild  = []string{"pummel", "body_slam", "submission", "water_gun"}
	riptideBuild = []string{"bubble", "bubble_beam", "whirlpool", "hydro_pump"}
	// The third Poliwag direction is fielded as the OTHER arm, which is what a
	// fork buys: the two above are Poliwrath's and this one is Politoed's, and
	// no stat line is shared between them.
	chorusBuild = []string{"rinse", "chorus", "rally", "hydro_pump"}

	gambleBuild = []string{"cross_chop", "submission", "seismic_toss", "inner_focus"}
	sureBuild   = []string{"vital_throw", "body_slam", "rock_throw", "seismic_toss"}

	mendBuild = []string{"moonlight", "wish", "rally", "moonblast"}
	hexBuild  = []string{"charm", "sing", "smokescreen", "solar_beam"}
)

// newBuildSeeds is how many battles each reading is averaged over. Sixty is what
// bulbasaurSeeds and its siblings use, and the differences these assert are far
// larger than the seed-to-seed noise at that count.
const newBuildSeeds = 60

// buildReading is what one kit did over those battles: the first three per
// battle, the last two as totals.
//
// ⚠️ The counts are NOT averaged, and that is the difference between a reading
// and a row of noughts. A duel here runs about a dozen turns and a kit misses a
// handful of times across sixty of them, so `total / newBuildSeeds` in integer
// arithmetic is nought for both kits — which is what the first version of
// TestTheTwoMachopBuildsAreDifferentUnits compared, and it reported the two as
// identical while one was missing several times as often as the other.
type buildReading struct {
	turns     int
	dealt     int64
	healed    int64
	missed    int
	inflicted int
}

// TestTheTwoPoliwagBuildsAreDifferentUnits.
//
// One closes and hits repeatedly, taking health back as it goes; the other stands
// off and slows whatever comes. The split is asserted where it lives — the drain
// and the statuses — rather than by fighting the two against each other, for the
// reason bulbasaurFight's comment gives: a mirror duel measures the twin.
func TestTheTwoPoliwagBuildsAreDifferentUnits(t *testing.T) {
	flurry := readBuild(t, "pokemon.poliwag", "Poliwrath", flurryBuild, "blood_thirst")
	riptide := readBuild(t, "pokemon.poliwag", "Poliwrath", riptideBuild, "spiteful")

	// The drain is the melee build, and nothing in the ranged one gives health
	// back — so a figure above nought there means a skill or a trait moved
	// between the two directions and the split stopped being one.
	if flurry.healed <= 0 {
		t.Error("the melee build recovered nothing, and the health coming back is what its trait is")
	}
	if riptide.healed != 0 {
		t.Errorf("the ranged build recovered %d, and it holds nothing that gives health back",
			riptide.healed)
	}
	// And the statuses are the ranged build: three of its four skills mire, and
	// its trait exists to make them land.
	if riptide.inflicted <= flurry.inflicted {
		t.Errorf("the ranged build landed %d statuses and the melee build %d, "+
			"so the one built to slow is not slowing more", riptide.inflicted, flurry.inflicted)
	}
	// The third direction is the other ARM, and it is not a damage kit at all:
	// one of its four skills deals any, so it has to read well under both of the
	// Poliwrath ones or the fork bought nothing.
	chorus := readBuild(t, "pokemon.poliwag", "Politoed", chorusBuild, "composure")
	if chorus.dealt >= flurry.dealt || chorus.dealt >= riptide.dealt {
		t.Errorf("the chorus build dealt %d against the melee build's %d and the ranged build's %d, "+
			"so the arm built to hold a line is trading blows with the ones built to land them",
			chorus.dealt, flurry.dealt, riptide.dealt)
	}
	t.Logf("flurry %d turns dealing %d recovering %d, %d statuses; "+
		"riptide %d turns dealing %d, %d statuses; "+
		"chorus %d turns dealing %d, %d statuses",
		flurry.turns, flurry.dealt, flurry.healed, flurry.inflicted,
		riptide.turns, riptide.dealt, riptide.inflicted,
		chorus.turns, chorus.dealt, chorus.inflicted)
}

// TestTheTwoMachopBuildsAreDifferentUnits.
//
// This is the one split in the cast that is about **accuracy** rather than about
// damage or sustain: one kit swings for everything behind the worst odds in the
// book, the other brings the only damaging skill that cannot miss and three more
// that rarely do. So it is asserted on misses, which is the axis the two were
// chosen along.
func TestTheTwoMachopBuildsAreDifferentUnits(t *testing.T) {
	gamble := readBuild(t, "pokemon.machop", progression.Furthest, gambleBuild, "berserk")
	sure := readBuild(t, "pokemon.machop", progression.Furthest, sureBuild, "unyielding")

	if gamble.missed <= sure.missed {
		t.Errorf("the gambling build missed %d times and the reliable one %d, so the kit "+
			"built behind long odds is not paying for them", gamble.missed, sure.missed)
	}
	// Both have to actually be fighting, or the comparison above is between two
	// kits that never swung.
	if gamble.dealt <= 0 || sure.dealt <= 0 {
		t.Fatalf("gamble dealt %d and sure dealt %d: a kit that lands nothing measures nothing",
			gamble.dealt, sure.dealt)
	}
	t.Logf("gamble %d turns dealing %d, %d misses; sure %d turns dealing %d, %d misses",
		gamble.turns, gamble.dealt, gamble.missed,
		sure.turns, sure.dealt, sure.missed)
}

// TestTheTwoCleffaBuildsAreDifferentUnits.
//
// ⚠️ Neither of these wins the duel they are measured in, and that is not what is
// being read. A mender loses every duel whatever it holds — see
// TestAMenderEarnsItsSlotWhereASparCannotSeeIt for why — so what a duel can still
// say about it is what it *spent its turns on*, which is exactly what separates
// these two directions.
func TestTheTwoCleffaBuildsAreDifferentUnits(t *testing.T) {
	mend := readBuild(t, "pokemon.cleffa", progression.Furthest, mendBuild, "composure")
	hexed := readBuild(t, "pokemon.cleffa", progression.Furthest, hexBuild, "elusive")

	if mend.healed <= 0 {
		t.Error("the mending build recovered nothing, and the health coming back is the build")
	}
	if hexed.healed != 0 {
		t.Errorf("the hexing build recovered %d, and it holds nothing that gives health back",
			hexed.healed)
	}
	if hexed.inflicted <= mend.inflicted {
		t.Errorf("the hexing build landed %d statuses and the mending one %d, "+
			"so the one built to disable is not disabling more",
			hexed.inflicted, mend.inflicted)
	}
	t.Logf("mend %d turns dealing %d recovering %d, %d statuses; "+
		"hex %d turns dealing %d, %d statuses",
		mend.turns, mend.dealt, mend.healed, mend.inflicted,
		hexed.turns, hexed.dealt, hexed.inflicted)
}

// readBuild fights one kit against the shipped Charizard over newBuildSeeds seeds
// and averages what it did.
//
// Charizard for the reason bulbasaurRun and squirtleRun both give: two kits are
// comparable only against something held still, and it is the heaviest attacker
// in the cast, so a build that cannot answer it shows that quickly.
func readBuild(t *testing.T, who, stage string, kit []string, trait string) buildReading {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fieldedAs(t, who, stage)
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, "pokemon.charmander")

	var total buildReading
	for seedValue := 1; seedValue <= newBuildSeeds; seedValue++ {
		fight, err := battle.New(books, uint64(seedValue), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: kit, Passives: []string{trait}},
			{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
				Stats: theirStats, Skills: theirKit, Passives: theirTraits},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		ran, err := fight.RunToEnd(4000)
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		total.turns += ran
		for _, event := range fight.Drain() {
			switch {
			// A tick names the unit CARRYING the status rather than whoever
			// applied it, so damage over time is counted off the opponent — the
			// same attribution bulbasaurRun spells out, and in a duel the side
			// is enough to make it.
			case event.Kind == battle.Damaged && event.Actor == "mine":
				total.dealt += event.Amount
			case event.Kind == battle.StatusTicked && event.Actor == "theirs":
				total.dealt += event.Amount
			case event.Kind == battle.Healed && event.Actor == "mine":
				total.healed += event.Amount
			case event.Kind == battle.Missed && event.Actor == "mine":
				total.missed++
			case event.Kind == battle.StatusApplied && event.Actor == "mine" &&
				event.Target == "theirs":
				total.inflicted++
			}
		}
	}
	return buildReading{
		turns:     total.turns / newBuildSeeds,
		dealt:     total.dealt / newBuildSeeds,
		healed:    total.healed / newBuildSeeds,
		missed:    total.missed,
		inflicted: total.inflicted,
	}
}
