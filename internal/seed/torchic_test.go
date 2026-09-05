package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// The three ids the snowball is made of, named once so a rename cannot leave
// this file measuring something that no longer exists.
const (
	stokedStatus = "stoked"
	stokingTrait = "quickening"
	rushKit      = "torchic.rush"
)

// snowballOpponents is who the loop is driven against, and the spread is the
// point — the same reasoning chargeOpponents is written under.
//
// A trait that pays off in **turns survived** cannot be read against one
// opponent: a rush that ends the fight in four turns never reaches the cap and
// says nothing, and a wall that never threatens anything reaches it every time
// and says nothing either. Riolu is the fastest thing in the cast and Squirtle
// the longest game, so a threshold that holds across both is a threshold about
// the trait rather than about a matchup.
var snowballOpponents = []string{
	"pokemon.squirtle", "pokemon.riolu", "pokemon.gastly", "pokemon.charmander",
}

const snowballSeeds = 30

// TestTheShippedSnowballReachesItsCapInARealBattle is the half a parser cannot
// reach, and it is the claim the whole character is bought for.
//
// `quickening` renews a timed status once per turn of its holder's own; the
// status caps at five stacks. Nothing in the data says those two facts compose
// into a unit that ends a fight faster than it started one — a renewal that
// landed but never accumulated, a cap never reached, or a status stripped as
// fast as it goes on would each leave the book parsing, the gloss written and
// the character pointless.
//
// ⚠️ **A win rate cannot see this.** Measured: the same kit with `quickening`
// swapped for `blaze` reads 747‰ against itself, so the trait is plainly worth
// something — but a rate that moves says nothing about *how*, and it moves by
// about as much for a trait that adds one stack and stops as for one that
// climbs. What is counted here is the depth reached, never a winner.
func TestTheShippedSnowballReachesItsCapInARealBattle(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	kind, err := books.Statuses.Lookup(stokedStatus)
	if err != nil {
		t.Fatalf("no %q status ships, so there is no snowball to measure: %v", stokedStatus, err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.torchic")
	kit := buildNamed(t, rushKit)

	battles, samples, deepest := 0, 0, 0
	reached := map[int]int{}
	for _, id := range snowballOpponents {
		theirStats, theirAffinity, theirKit, theirTraits := fielded(t, id)
		for value := 1; value <= snowballSeeds; value++ {
			fight, err := battle.New(books, uint64(value), []battle.Roster{
				{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity,
					Stats: stats, Skills: kit, Passives: []string{stokingTrait}},
				{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
					Stats: theirStats, Skills: theirKit, Passives: theirTraits},
			})
			if err != nil {
				t.Fatalf("new battle against %s: %v", id, err)
			}
			fight.Begin()
			mine, standing := fight.Unit("mine")
			if !standing {
				t.Fatal("the unit under measurement is not on the board")
			}
			// The depth is read off the unit rather than off the log, because the
			// log says how many stacks an application ADDED and the claim is about
			// the running total. Sampled through the Chooser, which is called once
			// per open turn with the prompt — so this reads the state at the
			// moment the unit is due to act, after the tick that spends durations
			// and after the renewal that lands behind it.
			best := 0
			sampling := func(prompt *battle.Prompt) (battle.Choice, bool) {
				if prompt.Unit == mine.ID {
					samples++
					if held := mine.Statuses.Stacks(stokedStatus); held > best {
						best = held
					}
				}
				return fight.Suggest(prompt)
			}
			if _, err := fight.RunToEndWith(4000, sampling); err != nil {
				t.Fatalf("seed %d against %s: %v", value, id, err)
			}
			battles++
			reached[best]++
			if best > deepest {
				deepest = best
			}
		}
	}
	t.Logf("over %d duels: %d turns sampled, deepest stack %d of %d; depths reached %v",
		battles, samples, deepest, kind.MaxStacks, reached)

	// The premise, held rather than assumed, and it is about the FIXTURE and not
	// about the trait: a duel that gives this unit one turn cannot show a
	// per-turn payoff whatever the trait does, so the two assertions below would
	// be measuring the matchup. Measured: about ten turns a duel.
	if samples < 2*battles {
		t.Errorf("%d turns sampled over %d duels, want at least two a duel: there are not "+
			"enough turns here for a trait that pays out per turn to have been measured",
			samples, battles)
	}
	// The claim itself, and the one no other test in this repository would
	// notice: the stacks ACCUMULATE to the cap rather than landing and lapsing.
	//
	// Two mutations, both measured, because the mechanism has two halves and each
	// can fail on its own:
	//   - `stoked` given duration 1 — it wears off on the very tick before the
	//     renewal that would have deepened it. Deepest stack falls to **1 of 5**,
	//     every duel, and the share assertion below goes to nought.
	//   - `quickening` pointed at `haste` instead — the renewal lands on something
	//     else and the deepest `stoked` is **nought**.
	//
	// ⚠️ **The obvious third mutation cannot be run and that is worth knowing.**
	// Emptying `renews` on `quickening` does not redden this test: passive.ParseBook
	// refuses a trait that does nothing at all ("grants nothing, renews nothing …
	// so holding it would change nothing"), so the books fail to load and the
	// failure is a parse error rather than a measurement. A mutation that does not
	// run proves nothing — point the renewal somewhere else instead.
	if deepest < kind.MaxStacks {
		t.Errorf("the deepest %s reached over %d duels is %d of %d: the renewal lands and does "+
			"not build, so this character's whole bargain — a turn survived is a turn taken "+
			"back — never pays out", stokedStatus, battles, deepest, kind.MaxStacks)
	}
	// And it is not a fluke of one long fight. ⚠️ Written as a share of battles
	// rather than as "every battle": a duel that ends in four turns cannot reach
	// five stacks however good the trait is, and demanding it would be a test
	// about matchups wearing this one's name. Measured on the shipped data: 102 of
	// 120 duels reach the full five and 13 more reach four, so the quarter this
	// asks for has a wide margin — deliberately, because the margin is what stops
	// a balance change to somebody else's character from reddening this one.
	deep := 0
	for stacks, count := range reached {
		if stacks >= kind.MaxStacks-1 {
			deep += count
		}
	}
	if deep*4 < battles {
		t.Errorf("only %d of %d duels got %s within one stack of its cap: the snowball is "+
			"reaching depth too rarely to be what the character is bought for",
			deep, battles, stokedStatus)
	}
}

// buildNamed is the four skills a shipped build fields, read from builds.json so
// that this file measures the kit the catalogue offers rather than one of its
// own — the same reason chargeBuild names the fielded four.
func buildNamed(t *testing.T, id string) []string {
	t.Helper()
	catalogue, err := seed.Builds()
	if err != nil {
		t.Fatalf("load the shipped builds: %v", err)
	}
	for _, built := range catalogue.All() {
		if built.ID == id {
			return built.Skills
		}
	}
	t.Fatalf("no build %q ships", id)
	return nil
}
