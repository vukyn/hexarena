package seed_test

import (
	"fmt"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// The shipped reserve kits against the best four-attack kit the same character
// can field, both sides fielded as forge.Spar fields them and the fourth slot
// held constant so the two rows differ only in the pair under test:
//
//	carrier      reserve pair                    spar    the four-attack kit
//	Charizard    bank_embers + pyre  (all, 230)  60.5%   68.9%
//	Venusaur     sprout + bloom_burst     (10)   68.3%   65.5%
//	Blastoise    soak + deluge       (all, 200)  59.7%   63.4%
//	Poliwrath    soak + tide_break        (20)   59.4%   58.6%
//
// ⚠️ **The deepest rung belongs to the FASTER carrier, and that was measured
// rather than chosen.** `tide_break` asks for twenty stacks, which is four
// banking turns, and four turns is a far larger share of a slow unit's battle
// than of a quick one's: on Blastoise the same kit read 7.4% against a 12.1%
// control. Moving it to Poliwrath brought it level. A rung's depth is a cost paid
// in tempo, so how deep a carrier can bank is a fact about its speed.
//
// reserveSeeds is how many duels each loop below is fought over. Small, because
// what is read is whether a mechanic fires at all rather than a rate.
const reserveSeeds = 40

// TestEveryShippedReserveHasBothHalvesOfItsLoop is the guard against the failure
// this repository keeps paying for in a new shape each time: something authored,
// parsed, validated, glossed and shipped, that nothing on the board ever reaches.
//
// A reserve is worth exactly nothing without a skill that fills it and a skill
// that spends it, and neither half tells you the other is missing — a generator
// with no consumer is a turn thrown away, and a consumer with no generator is a
// condition that never holds. `spendable` says the same thing in the rating and
// returns nought rather than a guess; this says it in the data.
func TestEveryShippedReserveHasBothHalvesOfItsLoop(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skills: %v", err)
	}
	reserves := 0
	for _, kind := range statuses.Kinds() {
		if kind.Category != status.Reserve {
			continue
		}
		reserves++
		fills, spends := 0, 0
		for _, declared := range skills.Skills() {
			for _, application := range append(append([]skill.Application{},
				declared.Applies...), declared.SelfApplies...) {
				if application.Status == kind.ID {
					fills++
				}
			}
			if declared.SelfRequires != nil && declared.SelfRequires.Consume &&
				declared.SelfRequires.Status == kind.ID {
				spends++
			}
		}
		if fills == 0 {
			t.Errorf("%s is a reserve nothing in the book fills", kind.ID)
		}
		if spends == 0 {
			t.Errorf("%s is a reserve nothing in the book spends, so every stack of it is a turn thrown away", kind.ID)
		}
		t.Logf("%-9s %d skill(s) fill it, %d spend it", kind.ID, fills, spends)
	}
	if reserves == 0 {
		t.Fatal("the shipped book declares no reserve at all, so nothing above ran")
	}
}

// TestTheShippedSpendersCoverTheLadder is the reason there is more than one
// spender per element.
//
// A counter whose only consumer takes a fixed one is a counter that never grows,
// and one whose only consumer takes the lot is a counter with no decision in it.
// What makes banking a choice is that the SAME pile can be cashed at different
// depths, so the shipped set has to span more than one size — and "all of it" has
// to be spelled as a per-stack payment, because a flat bonus paid for emptying a
// tank pays the same for emptying two stacks as for emptying twenty.
func TestTheShippedSpendersCoverTheLadder(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skills: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	sizes, scaling := map[int]string{}, 0
	for _, declared := range skills.Skills() {
		spends := declared.SelfRequires
		if spends == nil || !spends.Consume {
			continue
		}
		kind, err := statuses.Lookup(spends.Status)
		if err != nil || kind.Category != status.Reserve {
			continue
		}
		// The question is what SHAPE the payment is, not what it buys. A spender
		// paid per stack names no rung — it is the "spend what is there" arm
		// whatever currency it is paid in — so a heal that scales would otherwise
		// fall into the fixed-rung map below and collide with the blow that really
		// does spend that count flat.
		//
		// ⚠️ Asked as two predicates rather than by widening Scales(), which the
		// description reads to decide whether to promise a per-stack bonus to the
		// blow. See skill.Condition.ScalesRestore.
		if spends.Scales() || spends.ScalesRestore() {
			scaling++
			continue
		}
		if seen, clash := sizes[spends.ConsumeStacks]; clash {
			t.Errorf("%s and %s both spend %d stacks flat, which is one rung written twice",
				seen, declared.ID, spends.ConsumeStacks)
		}
		sizes[spends.ConsumeStacks] = declared.ID
	}
	if len(sizes) < 3 {
		t.Errorf("the book spends a fixed count at %d different depths (%v), which is not a ladder", len(sizes), sizes)
	}
	if scaling == 0 {
		t.Error("nothing in the book pays per stack, so `spend everything` is unwritable and a deep reserve buys what a shallow one does")
	}
	t.Logf("fixed rungs %v, and %d skill(s) paid per stack", sizes, scaling)
}

// TestTheShippedFireLoopBanksAndSpendsInARealBattle is the half a data change
// cannot get from a parser: the kit is legal, the numbers are in range, and none
// of that says the loop ever closes on a board.
//
// It is fought rather than inspected because everything between the two halves is
// a decision — the rating has to think banking is worth a turn, and then think
// cashing it in is worth the next one. A loop that parses and is never chosen is
// the same dead branch as one that is never written.
//
// ⚠️ The kit is driven rather than taken from the learnset. `forge.Spar` fields
// the first four skills a learnset declares, and these sit further down it, so a
// test that read the learnset would measure Charizard's opening four and report
// nothing about the reserve at all.
func TestTheShippedFireLoopBanksAndSpendsInARealBattle(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, traits := fielded(t, "pokemon.charmander")
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, "pokemon.squirtle")

	banked, spent, deepest := 0, 0, 0
	for value := 1; value <= reserveSeeds; value++ {
		fight, err := battle.New(books, uint64(value), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills:   []string{"bank_embers", "pyre", "ember", "smokescreen"},
				Passives: traits},
			{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
				Stats: theirStats, Skills: theirKit, Passives: theirTraits},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", value, err)
		}
		for _, event := range fight.Drain() {
			switch {
			case event.Kind == battle.StatusApplied && event.Status == "swelter":
				banked += event.Stacks
			case event.Kind == battle.StatusConsumed && event.Status == "swelter":
				spent += event.Stacks
				if event.Stacks > deepest {
					deepest = event.Stacks
				}
			}
		}
	}
	t.Logf("over %d duels: %d stacks banked, %d spent, deepest single spend %d",
		reserveSeeds, banked, spent, deepest)
	if banked == 0 {
		t.Fatal("nothing banked a single stack, so the generator is never worth a turn")
	}
	if spent == 0 {
		t.Fatal("every stack banked expired unspent, so the loop is a generator with a consumer nobody reaches")
	}
	// ⚠️ Deeper than the threshold, not merely non-zero. A spend at exactly the
	// minimum is a skill firing the instant it may, which is what a rating that
	// cannot see the cost does — and it would pass a bare `spent > 0` while
	// proving that banking further buys nothing.
	spender, err := books.Skills.Lookup("pyre")
	if err != nil {
		t.Fatalf("lookup pyre: %v", err)
	}
	if deepest <= spender.SelfRequires.MinStacks {
		t.Errorf("the deepest spend was %d stacks against a threshold of %d: the reserve is cashed the instant it may be, so banking further buys nothing",
			deepest, spender.SelfRequires.MinStacks)
	}
}

// TestAShippedSpenderReadsTheDepthItIsCastAt is the arithmetic of the shipped
// skill rather than of a fixture, and it is asserted across the ladder in one
// table so a rung that stopped scaling cannot hide behind the two beside it.
func TestAShippedSpenderReadsTheDepthItIsCastAt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	for _, id := range []string{"pyre", "deluge"} {
		declared, err := books.Skills.Lookup(id)
		if err != nil {
			t.Fatalf("lookup %s: %v", id, err)
		}
		spends := declared.SelfRequires
		if !spends.Scales() {
			t.Errorf("%s is meant to pay per stack and does not", id)
			continue
		}
		rows := make([]string, 0, 3)
		last := 0
		for _, held := range []int{spends.MinStacks, spends.MinStacks * 2, spends.MinStacks * 4} {
			bonus := declared.SelfBonus(skill.Carrying(held))
			rows = append(rows, fmt.Sprintf("%d→+%d", held, bonus))
			if bonus <= last {
				t.Errorf("%s at %d stacks paid %d, which is no more than the %d it paid one rung down",
					id, held, bonus, last)
			}
			last = bonus
		}
		t.Logf("%-7s %v (ceiling %d, capped at %d stacks)",
			id, rows, declared.SelfCeiling(), spends.Takes(skill.MaxSpendPower))
	}
}
