package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/seed"
)

// narutoKit is the four slots these fight over. The trait is the whole variable,
// so the kit is held still and written out rather than read off the learnset.
var narutoKit = []string{"kunai", "wind_shuriken", "rasengan", "shadow_clone"}

// swiftSeeds is how many battles each arrangement below is fought over.
const swiftSeeds = 150

// TestSwiftnessBuysSpeedAndNothingElse is the trait read rather than fought.
//
// A grant is a status id in a JSON file, and the file it points at is edited by
// somebody else on another day: a second modifier added to `quickened` would
// make this trait quietly buy two things, and every win rate in the repository
// would agree the change was an improvement without saying what it was.
func TestSwiftnessBuysSpeedAndNothingElse(t *testing.T) {
	traits, statuses := mustPassives(t), mustStatuses(t)
	held, err := traits.Lookup("swiftness")
	if err != nil {
		t.Fatalf("look up swiftness: %v", err)
	}
	if len(held.Grants) != 1 {
		t.Fatalf("swiftness grants %d statuses, want the one", len(held.Grants))
	}
	kind, err := statuses.Lookup(held.Grants[0].Status)
	if err != nil {
		t.Fatalf("look up %s: %v", held.Grants[0].Status, err)
	}
	if !kind.Permanent {
		t.Error("swiftness grants a timed status, so it wears off with nothing to put it back")
	}
	if len(kind.Modifiers) != 1 {
		t.Fatalf("%s moves %d stats, want the one", kind.ID, len(kind.Modifiers))
	}
	if term := kind.Modifiers[0]; term.Target != modifier.Speed || term.Amount <= 0 {
		t.Errorf("%s moves %s by %d, want speed upwards", kind.ID, term.Target, term.Amount)
	}
}

// TestASpeedTraitIsPricedBelowTheOtherPermanentBuffs is the number that had to
// break a convention, recorded where the next author will look for it.
//
// Every other permanent buff a trait grants is 150 per mille — `toughened`,
// `kindled`, `unleashed` — and that reads as the house figure. It does not
// transfer, because a point of speed is worth more here than a point of anything
// else: speed is turns, and a turn is every other stat applied again.
//
// ⚠️ Measured rather than reasoned. At 150 a Naruto holding this beats the same
// Naruto holding `endurance` in **84 mirror duels out of 100**, which is not a
// choice between two traits, it is one trait and one trap. At 80 it is 64, which
// is a favourite — and a favourite is what a trait suiting a fast, fragile
// character is supposed to be.
func TestASpeedTraitIsPricedBelowTheOtherPermanentBuffs(t *testing.T) {
	statuses := mustStatuses(t)
	quick, err := statuses.Lookup("quickened")
	if err != nil {
		t.Fatalf("look up quickened: %v", err)
	}
	quickest := quick.Modifiers[0].Amount
	compared := 0
	for _, kind := range statuses.Kinds() {
		if !kind.Permanent || kind.ID == "quickened" {
			continue
		}
		for _, term := range kind.Modifiers {
			if term.Amount <= 0 || term.Target == modifier.Speed {
				continue
			}
			compared++
			if quickest >= term.Amount {
				t.Errorf("quickened raises speed by %d and %s raises %s by %d: a point of speed "+
					"is worth more than a point of anything else, so pricing it level makes the "+
					"other trait a trap rather than a choice",
					quickest, kind.ID, term.Target, term.Amount)
			}
		}
	}
	if compared == 0 {
		t.Fatal("no other permanent buff raises a stat, so nothing was compared")
	}
}

// TestSwiftnessIsAFavouriteAndNotALock is the claim the figure above was tuned
// against, fought rather than read.
//
// One Naruto against another with the same four skills, so the trait is the whole
// difference — and both ways round, because atb.Queue breaks a tie by enlistment
// and a one-way mirror would report that instead.
//
// ⚠️ The band is wide and deliberately one that a **trait suiting its character**
// sits inside. Naruto is fast and thin; a defensive trait was never going to be
// its best. What the upper bound refuses is `endurance` becoming unpickable.
func TestSwiftnessIsAFavouriteAndNotALock(t *testing.T) {
	swift, tough := 0, 0
	for _, swap := range []bool{false, true} {
		for seedValue := 1; seedValue <= swiftSeeds; seedValue++ {
			winner, decided := narutoMirror(t, swap, uint64(seedValue))
			if !decided {
				continue
			}
			if (winner == hex.SideAlly) != swap {
				swift++
			} else {
				tough++
			}
		}
	}
	fought := swift + tough
	if fought == 0 {
		t.Fatal("no mirror duel ended, so nothing was measured")
	}
	rate := swift * scale.Base / fought
	const lowest, highest = 500, 750
	if rate < lowest || rate > highest {
		t.Errorf("swiftness takes %d.%d%% of %d mirror duels against endurance, outside %d..%d: "+
			"below the floor it buys nothing, above the ceiling endurance is not a choice",
			rate/10, rate%10, fought, lowest, highest)
	}
	t.Logf("swiftness %d, endurance %d over %d duels: %d.%d%%",
		swift, tough, fought, rate/10, rate%10)
}

// TestSwiftnessActuallyBuysTurns is what a speed trait is *for*, and the thing a
// win rate cannot say.
//
// A rate says which side won; it does not say why, and a trait that raised speed
// without the queue reading it would still win more often through whatever else
// changed. This counts the turns each side is handed.
func TestSwiftnessActuallyBuysTurns(t *testing.T) {
	swiftTurns, toughTurns := narutoTurns(t, "swiftness"), narutoTurns(t, "endurance")
	if swiftTurns <= toughTurns {
		t.Errorf("a Naruto holding swiftness took %d turns and one holding endurance %d: "+
			"the trait raises speed and speed is turns, so it has to be handed more of them",
			swiftTurns, toughTurns)
	}
	t.Logf("turns taken: swiftness %d, endurance %d", swiftTurns, toughTurns)
}

// narutoMirror fights one Naruto trait against the other over one seed.
func narutoMirror(t *testing.T, swap bool, seedValue uint64) (hex.Side, bool) {
	t.Helper()
	first, second := "swiftness", "endurance"
	if swap {
		first, second = second, first
	}
	fight := narutoBattle(t, first, second, seedValue)
	if _, err := fight.RunToEnd(4000); err != nil {
		t.Fatalf("run: %v", err)
	}
	return fight.Winner()
}

// narutoTurns is how many turns a Naruto under one trait is handed across the
// sweep, counted off the log rather than worked out from the stat line.
func narutoTurns(t *testing.T, trait string) int {
	t.Helper()
	taken := 0
	for seedValue := 1; seedValue <= swiftSeeds/5; seedValue++ {
		fight := narutoBattle(t, trait, "endurance", uint64(seedValue))
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("run: %v", err)
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.TurnBegan && event.Actor == "mine" {
				taken++
			}
		}
	}
	return taken
}

// narutoBattle is one Naruto against another, the kit held still and the trait
// the only difference between them.
func narutoBattle(t *testing.T, mine, theirs string, seedValue uint64) *battle.Battle {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "naruto.naruto")
	fight, err := battle.New(books, seedValue, []battle.Roster{
		{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: narutoKit, Passives: []string{mine}},
		{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: narutoKit, Passives: []string{theirs}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	return fight
}
