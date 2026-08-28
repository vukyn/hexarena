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
// ⚠️ Measured rather than reasoned, in the share of the turn order the trait
// buys: 150 hands its holder **14.8% more turns** than `endurance` gets in the
// same battles, where 80 hands it 7.9% and 30 hands it 2.6%. The sweep and why
// it is counted in turns rather than in wins are in
// TestSwiftnessBuysAShareOfTheTurnsAndNotAllOfThem.
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

// TestSwiftnessBuysAShareOfTheTurnsAndNotAllOfThem is the figure the trait was
// priced against, and it is a share of turns rather than a win rate on purpose.
//
// ⚠️ A win rate was tried first and abandoned, which is worth the paragraph
// because the mirror duel *looks* like the obvious measurement. Over 300 duels
// both ways round it does not order the amounts it is supposed to be measuring:
//
//	+30  59.6%    +40  63.3%    +50  74.0%    +60  63.0%
//	+80  73.0%    +100 57.0%    +150 59.0%
//
// Priced at the house figure of 150 the trait comes back *below* the same trait
// priced at 50. That is not the trait being subtle: the turn queue is discrete,
// so what a few points of speed buy is whether an extra turn lands before the
// other unit acts, and which side of that line a seed falls on is lumpy. Since
// battle.Suggest learned to cast a skill with no power there is a summon in the
// queue as well, and the lumps got larger. A band over that number would have
// let a trait priced at 150 sit comfortably inside it.
//
// The share of turns is the same measurement without the noise, because it is
// what the trait actually does rather than what eventually comes of it:
//
//	+30 2.6%    +40 3.5%    +50 4.4%    +60 5.6%
//	+80 7.9%    +100 9.9%   +150 14.8%
//
// Monotone across the whole sweep. The band keeps the trait buying a real share
// of the turn order without buying the fight: below the floor it is a trait
// nobody would spend a slot on, and above the ceiling `endurance` is not a
// choice.
func TestSwiftnessBuysAShareOfTheTurnsAndNotAllOfThem(t *testing.T) {
	swift, tough := narutoTurnShare(t)
	if tough == 0 {
		t.Fatal("the other side never acted, so nothing was measured")
	}
	over := (swift - tough) * scale.Base / tough
	const lowest, highest = 40, 130
	if over < lowest || over > highest {
		t.Errorf("swiftness takes %d.%d%% more turns than endurance in the same battles, "+
			"outside %d.%d..%d.%d%%: under the floor the trait buys nothing worth a slot, "+
			"over the ceiling endurance is not a choice",
			over/10, over%10, lowest/10, lowest%10, highest/10, highest%10)
	}
	t.Logf("swiftness acts %d.%d%% more often: %d turns against %d",
		over/10, over%10, swift, tough)
}

// narutoTurnShare is how the two traits split the turns of the battles they
// fight against each other, counted off the log rather than worked out from the
// stat line.
//
// Both figures come out of the same battles on purpose: whatever else moves the
// length of a fight moves it for both sides at once, so the comparison survives
// a change to what the opponent decides to cast.
func narutoTurnShare(t *testing.T) (swift, tough int) {
	t.Helper()
	for seedValue := 1; seedValue <= swiftSeeds/5; seedValue++ {
		fight := narutoBattle(t, "swiftness", "endurance", uint64(seedValue))
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("run: %v", err)
		}
		for _, event := range fight.Drain() {
			if event.Kind != battle.TurnBegan {
				continue
			}
			switch event.Actor {
			case "mine":
				swift++
			case "theirs":
				tough++
			}
		}
	}
	return swift, tough
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
