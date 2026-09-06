package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// The three ids the split is made of, named once so a rename cannot leave this
// file measuring something that no longer exists.
const (
	splitID       = "split"
	sunderedCount = "sundered"
	splitBuild    = "diglett.three"
)

// splitOpponents is who the loop is driven against, and the spread is the point.
//
// A cap is a claim about a WHOLE battle, so the only opponent that can disprove
// it is one the battle lasts long enough to reach — Squirtle is the longest game
// in the cast and Riolu the shortest, and a cap that holds across both is a cap
// rather than a matchup.
var splitOpponents = []string{
	"pokemon.squirtle", "pokemon.happiny", "pokemon.riolu", "pokemon.charmander",
}

const splitSeeds = 30

// TestTheShippedSplitIsCappedForAWholeBattle is the half the engine test could
// not reach, and it is here rather than in internal/core/battle because what it
// measures is the shipped DATA making the engine's promise true.
//
// ⚠️ **`below_stacks` caps a skill for as long as the counter survives, and a
// timed counter does not survive a battle.** status.Set.Apply refreshes every
// stack's duration when a new one lands, so a counter is renewed while the skill
// is still being cast — and then the last cast is the last renewal, the stacks
// run out `duration` turns later, and the allowance quietly comes back. The
// ceiling on a duration is six turns and a battle here runs forty to ninety, so
// "twice a battle" spelled with a timed counter is really "twice every seven
// turns".
//
// That is why `sundered` is **permanent**: Set.Tick skips a permanent entry
// whole, so the tally never expires and the cap is what it says. The engine test
// (TestASplitIsCappedForTheWholeBattle) cannot see any of this — it runs a
// handful of turns, which a timed counter survives.
func TestTheShippedSplitIsCappedForAWholeBattle(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	kind, err := books.Statuses.Lookup(sunderedCount)
	if err != nil {
		t.Fatalf("no %q status ships: %v", sunderedCount, err)
	}
	// The premise, held rather than assumed. A timed counter would make every
	// figure below a statement about the first few turns.
	if !kind.Permanent {
		t.Fatalf("%q is timed, so the cap it enforces lapses with it and this test is "+
			"measuring a window rather than a battle", sunderedCount)
	}
	stats, affinity, _, traits := fielded(t, "pokemon.diglett")
	kit := buildNamed(t, splitBuild)

	battles, longest, over := 0, 0, 0
	casts := map[int]int{}
	for _, id := range splitOpponents {
		theirStats, theirAffinity, theirKit, theirTraits := fielded(t, id)
		for value := 1; value <= splitSeeds; value++ {
			fight, err := battle.New(books, uint64(value), []battle.Roster{
				{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity,
					Stats: stats, Skills: kit, Passives: traits},
				{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
					Stats: theirStats, Skills: theirKit, Passives: theirTraits},
			})
			if err != nil {
				t.Fatalf("new battle against %s: %v", id, err)
			}
			fight.Begin()
			// Cast the split at every opportunity rather than letting the rating
			// decide — the rating has no term for it at all (TODO.md § "The rating
			// cannot price hiding"), so a run on autopilot would cast it never and
			// the cap would hold for want of anybody testing it.
			mine, _ := fight.Unit("mine")
			spent := 0
			forcing := func(prompt *battle.Prompt) (battle.Choice, bool) {
				if prompt.Unit == mine.ID {
					for _, option := range prompt.Options {
						if option.Skill == splitID && option.Blocked == battle.BlockNone {
							spent++
							return battle.Choice{Skill: splitID, Aim: mine.Cell}, true
						}
					}
				}
				return fight.Suggest(prompt)
			}
			turns, err := fight.RunToEndWith(4000, forcing)
			if err != nil {
				t.Fatalf("seed %d against %s: %v", value, id, err)
			}
			battles++
			casts[spent]++
			if turns > longest {
				longest = turns
			}
			if spent > 2 {
				over++
			}
		}
	}
	t.Logf("over %d duels of up to %d turns: splits cast %v", battles, longest, casts)

	// The premise again, and this one is about the fixture: a battle shorter than
	// the counter's would-be duration cannot tell a permanent tally from a timed
	// one. Six is the ceiling on any duration, so anything well past it will do.
	if longest <= 12 {
		t.Fatalf("the longest duel here ran %d turns, which a timed counter would have "+
			"survived: this fixture cannot tell the two apart", longest)
	}
	// The claim.
	//
	// Mutation, measured rather than guessed: `sundered` given `"duration": 6`
	// instead of permanent — with the premise above silenced so the run gets past
	// it — reads `map[1:5 2:45 3:10 4:39 5:21]`. **Seventy of a hundred and twenty
	// duels go over the allowance and twenty-one reach five splits**, which is
	// two and a half characters' worth of bodies out of one slot.
	if over > 0 {
		t.Errorf("%d of %d duels cast the split more than twice: the allowance is meant to be "+
			"spent once and for the rest of the battle", over, battles)
	}
	// ⚠️ And it is not vacuous. A cap nobody reaches is a cap nothing measures,
	// which is exactly what this test would be if the rating were left to choose.
	if casts[2]*4 < battles {
		t.Errorf("only %d of %d duels spent both splits, so the bound above is mostly "+
			"measuring battles that ended early", casts[2], battles)
	}
}
