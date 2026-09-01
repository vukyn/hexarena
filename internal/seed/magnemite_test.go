package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The three directions, hardcoded here the way every other design record in this
// package is: these are the kits that were measured, and
// TestTheShippedBuildsAreTheOnesTheTestsMeasure is what stops the catalogue a
// player reads drifting away from them.
//
// They are the three answers to one question — what does this character do with
// the counter — and the third is the one that says the question has more than a
// yes and a no in it.
var (
	trickleBuild = []string{"charge_beam", "magnetise", "electro_ball", "spark"}
	surgeBuild   = []string{"zap_cannon", "thunderbolt", "flash_cannon", "discharge"}
	hoardBuild   = []string{"charge_beam", "magnetise", "overload", "thunderbolt"}
)

// magnemiteSeeds is how many duels each build is read over. Sixty is what every
// other build reading in this package uses, and the separations below are whole
// multiples rather than margins.
const magnemiteSeeds = 60

// counterReading is what one build did with the counter over those duels, and
// every column is a TOTAL rather than an average.
//
// ⚠️ The same arithmetic that made the first Machop reading a row of noughts:
// a duel runs about a dozen turns, so a figure divided by sixty truncates to
// nothing in integer arithmetic and two builds come back identical while one is
// doing several times as much of something as the other.
type counterReading struct {
	blows  int
	dealt  int64
	arc    int64
	spent  int
	cashes int
}

// each is what one blow came to, which is the shape of the damage rather than
// its size.
func (r counterReading) each() int64 {
	if r.blows == 0 {
		return 0
	}
	return r.dealt / int64(r.blows)
}

// perCash is how many stacks one discharge took, in hundredths, because the
// whole difference between the two shapes that spend the counter is whether that
// figure is one or several.
func (r counterReading) perCash() int {
	if r.cashes == 0 {
		return 0
	}
	return r.spent * 100 / r.cashes
}

// TestTheThreeMagnemiteBuildsAnswerTheCounterDifferently is the measurement
// behind the catalogue, and what it holds is the *shape* of each build rather
// than which of them wins.
//
// ⚠️ **None of them wins.** All three read 0 or 1 of sixty against the shipped
// Charizard, which is the heaviest attacker in the cast against the thinnest
// frame in it — and that figure is not what is being read, exactly as it was not
// for the mender's two. What a duel can still say is what a build spends its
// turns on, and here that is the whole of what distinguishes them:
//
//   - `trickle` converts the counter as fast as it lays it down — the most blows,
//     the smallest each, and exactly one stack a discharge;
//   - `surge` ignores the counter entirely, which is a real direction rather than
//     an absence: it is the same character built as though the mechanism were not
//     there, and it lands the fewest blows and the largest;
//   - `hoard` waits and takes the pile, so it cashes a third as often and takes
//     several stacks each time.
//
// The three claims are made against each other rather than against numbers, so
// what is held is the ordering — which is the part a balance edit must not
// silently reverse.
func TestTheThreeMagnemiteBuildsAnswerTheCounterDifferently(t *testing.T) {
	trickle := readCounter(t, trickleBuild, "swiftness")
	surge := readCounter(t, surgeBuild, "endurance")
	hoard := readCounter(t, hoardBuild, "elusive")
	for _, reading := range []struct {
		name string
		got  counterReading
	}{{"trickle", trickle}, {"surge", surge}, {"hoard", hoard}} {
		t.Logf("%-7s %5d blows of %3d, arc %6d, %4d stacks over %3d discharges (%d.%02d each)",
			reading.name, reading.got.blows, reading.got.each(), reading.got.arc,
			reading.got.spent, reading.got.cashes,
			reading.got.perCash()/100, reading.got.perCash()%100)
	}

	// The build that ignores the counter has to actually ignore it, or "three
	// answers" is two answers and a duplicate.
	if surge.spent != 0 || surge.arc != 0 {
		t.Errorf("the surge build spent %d stacks for %d arc damage, so it is not the direction that leaves the counter alone",
			surge.spent, surge.arc)
	}
	for _, spender := range []struct {
		name string
		got  counterReading
	}{{"trickle", trickle}, {"hoard", hoard}} {
		if spender.got.cashes == 0 {
			t.Fatalf("the %s build never discharged, so nothing here measured the counter at all", spender.name)
		}
	}

	// The shape of the damage. More pieces, each smaller, is the whole of what
	// the drip direction is for.
	if trickle.blows <= surge.blows {
		t.Errorf("the trickle build landed %d blows against the surge build's %d: the damage is not arriving in more pieces",
			trickle.blows, surge.blows)
	}
	if trickle.each() >= surge.each() {
		t.Errorf("the trickle build hits for %d a blow against the surge build's %d: the pieces are not smaller",
			trickle.each(), surge.each())
	}

	// And the shape of the spending, which is the drip and the nuke told apart.
	// A drip takes one a strike by construction, so anything but exactly one
	// means the reading is not reading what it thinks it is.
	if trickle.perCash() != 100 {
		t.Errorf("the trickle build took %d.%02d stacks a discharge; a drip takes one, and anything else means these figures are counting something other than the two shapes",
			trickle.perCash()/100, trickle.perCash()%100)
	}
	if hoard.perCash() <= trickle.perCash() {
		t.Errorf("the hoard build took %d.%02d stacks a discharge against the trickle build's %d.%02d: it is not waiting for a pile",
			hoard.perCash()/100, hoard.perCash()%100,
			trickle.perCash()/100, trickle.perCash()%100)
	}
	if hoard.cashes >= trickle.cashes {
		t.Errorf("the hoard build discharged %d times against the trickle build's %d: it is cashing as often as the drip does, which is the drip",
			hoard.cashes, trickle.cashes)
	}
}

// readCounter fights one build against the shipped Charizard over
// magnemiteSeeds duels and reads what it did off the log.
//
// Charizard for the reason readBuild gives: two kits are comparable only against
// something held still, and it is the heaviest attacker in the cast, so a build
// that cannot answer it shows that quickly.
//
// ⚠️ **A duel is a chain of one**, so nothing here reads the chain — the shape
// that travels is measured by TestAccumulatingIsAWayOfFightingRatherThanASlowerOne
// on a squad, where there is somebody for the current to step to. What a duel
// can say is how a build spends its own turns, which is what these three differ
// in.
func readCounter(t *testing.T, kit []string, trait string) counterReading {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fieldedAs(t, "pokemon.magnemite", progression.Furthest)
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, "pokemon.charmander")

	var total counterReading
	for seedValue := 1; seedValue <= magnemiteSeeds; seedValue++ {
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
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		for _, event := range fight.Drain() {
			if event.Actor != "mine" {
				continue
			}
			switch event.Kind {
			case battle.Damaged:
				total.blows++
				total.dealt += event.Amount
				// The arc is told apart by the status on the event, which is the
				// one thing that distinguishes the charge going off from the
				// skill's own blow — the same field a reply is told apart by.
				if event.Status == "charge" {
					total.arc += event.Amount
				}
			case battle.StatusConsumed:
				total.spent += event.Stacks
				total.cashes++
			}
		}
	}
	return total
}
