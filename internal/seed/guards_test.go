package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// guardSeeds is how many duels each row below is fought over.
const guardSeeds = 150

// TestConvertingIsTheAnswerToAWallRatherThanToArmour is the shipped trait stated
// as one measurement, and it is read against the board it is for rather than
// averaged across the cast.
//
// `rending` sends a quarter of every blow past defence entirely. A spar's overall
// figure cannot see that: a tool aimed at armour is diluted by ten opponents most
// of whom have little. So the same character is fought with two traits against
// two opponents chosen for their defence, and what is read is **how long the
// fight takes** rather than who wins it — against the cast's wall both traits win
// nearly every duel, and the whole difference is in the turns.
//
// Measured, Machamp against each:
//
//	opponent    defence   with rending   with blood_thirst
//	Blastoise       640    100% / 34 t     94.3% / 70 t
//	Poliwrath       560    100% / 14 t     98.6% / 21 t
//	Charizard       400   45.3% /  —      52.0% /  —
//
// ⚠️ **It costs where there is nothing to bypass.** Against the thin, fast one it
// reads seven points worse than the drain it replaces, and that is the trade
// rather than a flaw: a share sent past armour buys nothing from a unit with
// little, while a drain buys the same everywhere. A trait that were better on
// every board would not be a choice.
func TestConvertingIsTheAnswerToAWallRatherThanToArmour(t *testing.T) {
	for _, against := range []struct {
		name     string
		id       string
		stage    string
		armoured bool
	}{
		{"the wall", "pokemon.squirtle", progression.Furthest, true},
		{"the thin one", "pokemon.charmander", progression.Furthest, false},
	} {
		t.Run(against.name, func(t *testing.T) {
			theirs, _, _, _ := fieldedAs(t, against.id, against.stage)
			rending := readGuard(t, "rending", against.id, against.stage)
			draining := readGuard(t, "blood_thirst", against.id, against.stage)
			t.Logf("%s (defence %d): rending %d-%d over %d turns, blood_thirst %d-%d over %d turns",
				against.name, theirs[progression.Defense],
				rending.wins, rending.losses, rending.turns,
				draining.wins, draining.losses, draining.turns)

			if rending.turns == 0 || draining.turns == 0 {
				t.Fatal("no turns were fought, so there is nothing to compare")
			}
			if against.armoured {
				// The reading is the clock rather than the record: both win
				// nearly every duel here, so a rate would report them level while
				// one takes half as long.
				if rending.turns*3 >= draining.turns*2 {
					t.Errorf("against %d defence the converting trait took %d turns against the draining one's %d: sending a share past armour is supposed to be most of the difference",
						theirs[progression.Defense], rending.turns, draining.turns)
				}
				return
			}
			// And the price of it, on a board with little armour to bypass.
			if rending.wins >= draining.wins {
				t.Errorf("against %d defence the converting trait won %d and the draining one %d: a trait better on every board is not a choice",
					theirs[progression.Defense], rending.wins, draining.wins)
			}
		})
	}
}

type guardReading struct {
	wins, losses, turns int
}

// readGuard fights Machamp's shipped four with one trait against one opponent.
//
// One trait rather than the character's whole list, because what is being read is
// the trait: `forge.seedKit` takes the first one a learnset declares, and driving
// the roster instead lets the two rows differ in exactly the field under test.
func readGuard(t *testing.T, trait, foe, stage string) guardReading {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.machop")
	// ⚠️ **The kit is named here rather than read off the learnset**, and that is
	// what keeps this test measuring the thing it is about. The claim is a
	// comparison of two TRAITS; the carrier is incidental to it. Taking the
	// fielded four instead made the reading move the day machop's learnset was
	// reordered for the charge kit — four attacks became two attacks and a skill
	// that stands still, the turns changed, and a claim about rending against
	// armour went red for a reason that had nothing to do with either trait.
	// These four are the ones the table above was measured on.
	kit := []string{"rock_throw", "body_slam", "cross_chop", "vital_throw"}
	theirStats, theirAffinity, theirKit, theirTraits := fieldedAs(t, foe, stage)
	var total guardReading
	for seedValue := 1; seedValue <= guardSeeds; seedValue++ {
		fight, err := battle.New(books, uint64(seedValue), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity,
				Stats: stats, Skills: kit, Passives: []string{trait}},
			{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
				Stats: theirStats, Skills: theirKit, Passives: theirTraits},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		ran, err := fight.RunToEnd(4000)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		total.turns += ran
		fight.Drain()
		if !fight.Finished() {
			continue
		}
		if winner, decided := fight.Winner(); decided && winner == hex.SideAlly {
			total.wins++
		} else if decided {
			total.losses++
		}
	}
	total.turns /= guardSeeds
	return total
}

// TestAGrantedBarrierIsWorthMostWhereTheFrameIsThinnest is what the two shipped
// guard traits measured out to, and it is the opposite of what their names
// suggest.
//
// A pool is a flat quantity, so what it is worth is a share of what its holder
// could already take. That makes the **attack**-scaled one coherent — attack does
// not correlate with survivability, so it lands wherever it lands — and the
// **defence**-scaled one self-defeating: it hands the deepest pool to the unit
// that needs it least, and on the wall it is a worse buy than the permanent
// defence stat it competes with.
//
//	carrier     trait         against       lead trait it replaces
//	Charizard   projection    71.9%         blaze 63.2%
//	Blastoise   carapace      29.2%         endurance 32.1%
//
// Both ship, because a weaker option is still an option and a wall that wants a
// lump of protection rather than a permanent stat is a real thing to want. What
// is held here is the reason, so a later balance pass does not read the figures
// as a bug.
func TestAGrantedBarrierIsWorthMostWhereTheFrameIsThinnest(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the passives: %v", err)
	}
	for _, row := range []struct {
		trait   string
		carrier string
		stat    progression.Kind
	}{
		{"projection", "pokemon.charmander", progression.Attack},
		{"carapace", "pokemon.squirtle", progression.Defense},
	} {
		held, err := passives.Lookup(row.trait)
		if err != nil {
			t.Fatalf("lookup %s: %v", row.trait, err)
		}
		if len(held.Grants) != 1 {
			t.Fatalf("%s grants %d things; this is about the one", row.trait, len(held.Grants))
		}
		grant := held.Grants[0]
		if grant.Power <= 0 {
			t.Errorf("%s grants a guard with no depth, which stops nothing", row.trait)
		}
		if grant.Scaling != row.stat {
			t.Errorf("%s is scaled off %s and the design says %s", row.trait, grant.Scaling, row.stat)
		}
		// It has to be carried, or the measurement above is about a trait nobody
		// can field.
		character, known := book.Get(row.carrier)
		if !known {
			t.Fatalf("no character %q", row.carrier)
		}
		carried := false
		for _, mine := range character.Passives {
			if mine.ID == row.trait {
				carried = true
			}
		}
		if !carried {
			t.Errorf("%s carries no %s, so nothing fields it", row.carrier, row.trait)
		}
	}
}
