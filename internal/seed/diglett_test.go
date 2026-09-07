package seed_test

import (
	"slices"
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

// The whole build's four slots, and the same four with the slot spent the way it
// used to be. Written out rather than derived, because that is the claim: these
// are the two kits an author would field.
//
// ⚠️ **`wholeBuild` is checked against the shipped data below and not assumed to
// be it.** The first version of this test named both kits and nothing else, and it
// stayed green with `burrow` taken back out of builds.json — it held a claim about
// two lists of strings and said nothing about what ships. A measurement is only a
// reason to ship something if it is measuring the thing that ships.
var (
	wholeBuild   = []string{"earthquake", "stone_edge", "dig", "burrow"}
	wholeNoHide  = []string{"earthquake", "stone_edge", "dig", "rock_throw"}
	wholeBuildID = "diglett.whole"
)

// wholeSeeds is how many battles each half of the comparison is fought over,
// both ways round.
const wholeSeeds = 200

// TestBurrowIsWhatMakesTheWholeBuildAMatchup is what the fourth slot was changed
// for, and it is a comparison rather than a rate.
//
// ⚠️ **`rock_throw` was the slot and Charmander was the reason.** The whole build
// wins nearly every matchup it has and loses that one almost as a script — 9.5%
// over 400 battles, which is the shape `TestTheDragonBuildIsASidegradeAndNotAnUpgrade`
// exists to refuse. Hiding is the only thing in the line's learnset that answers a
// unit which out-damages it, because it is the only skill that buys turns instead
// of dealing damage.
//
// The control is fought on the same run rather than quoted, for the reason every
// referent in this repository is re-taken: a figure from another session is a
// measurement of an engine that no longer exists.
//
// ⚠️ **It is a trade and the other half is stated.** Against Machop the burrow
// build reads 85.0% where the old one reads 99.0%, because a turn spent
// underground is a turn not spent swinging and Machop was already losing. Buying
// twenty-one points in the matchup that was a script for fourteen in one that was
// already won is the trade, and it is the reason this is a sidegrade and not an
// upgrade.
//
// ⚠️ **The same skill in the OTHER build is a disaster and must stay out of it.**
// `diglett.three` fought with burrow in place of `rock_throw` reads 11.0% against
// Machop where it otherwise reads 72.5% — and the mechanism is not the slot: the
// same build with the slot simply empty reads 78.0%. Burrow crowds out `split`,
// because `Suggest` is a greedy one-turn rating and hiding is worth more *this*
// turn than a body is. Measured over a whole duel it cast the split **not once**.
// Two self-cast skills in one kit is a decision the rating cannot make.
func TestBurrowIsWhatMakesTheWholeBuildAMatchup(t *testing.T) {
	// What ships, read rather than assumed. Everything below is a statement about
	// `wholeBuild`, and it is a reason to have changed the data only for as long
	// as the data is that.
	builds, err := seed.Builds()
	if err != nil {
		t.Fatalf("load the shipped builds: %v", err)
	}
	built, known := builds.Get(wholeBuildID)
	if !known {
		t.Fatalf("no build %q ships, so this measures a kit nobody can field", wholeBuildID)
	}
	if !slices.Equal(built.Skills, wholeBuild) {
		t.Fatalf("%s ships %v and this test measures %v: the figures below are about a "+
			"kit that is not the one in the data", wholeBuildID, built.Skills, wholeBuild)
	}

	hiding := theWholeBuildAgainst(t, wholeBuild, "pokemon.charmander")
	swinging := theWholeBuildAgainst(t, wholeNoHide, "pokemon.charmander")

	// The premise, held rather than assumed: the slot this replaced really does
	// lose that matchup. If the control ever climbs out on its own, the change has
	// no reason left and this test should say so rather than pass.
	if swinging >= 200 {
		t.Fatalf("the kit without hiding already wins %d.%d%% against Charmander, so the "+
			"scripted defeat this answers is gone and the slot is no longer paid for",
			swinging/10, swinging%10)
	}
	if hiding <= swinging {
		t.Errorf("hiding wins %d.%d%% against Charmander and swinging wins %d.%d%%: the "+
			"fourth slot is not buying the matchup it was spent on",
			hiding/10, hiding%10, swinging/10, swinging%10)
	}
	t.Logf("against Charmander: hiding %d.%d%%, swinging %d.%d%%",
		hiding/10, hiding%10, swinging/10, swinging%10)
}

// theWholeBuildAgainst fights one kit of the whole build against a shipped
// character over the seed range, both ways round, and reports the win rate in
// parts per thousand.
//
// Both ways for the reason every duel here takes both: the turn queue breaks a
// tie by enlistment, so a one-way figure carries the first slot's advantage into
// the answer.
func theWholeBuildAgainst(t *testing.T, kit []string, opponent string) int {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.diglett")
	theirStats, theirAffinity, theirKit, theirTrait := fielded(t, opponent)
	won, fought := 0, 0
	for _, mineFirst := range []bool{true, false} {
		for which := 1; which <= wholeSeeds; which++ {
			mine := battle.Roster{ID: "mine", Side: hex.SideAlly, Slot: buildSlot,
				Affinity: affinity, Stats: stats, Skills: kit, Passives: []string{"elusive"}}
			theirs := battle.Roster{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot,
				Affinity: theirAffinity, Stats: theirStats, Skills: theirKit, Passives: theirTrait}
			order := []battle.Roster{mine, theirs}
			if !mineFirst {
				mine.Side, theirs.Side = hex.SideEnemy, hex.SideAlly
				order = []battle.Roster{theirs, mine}
			}
			fight, err := battle.New(books, uint64(which), order)
			if err != nil {
				t.Fatalf("new battle: %v", err)
			}
			fight.Begin()
			if _, err := fight.RunToEnd(4000); err != nil {
				t.Fatalf("run: %v", err)
			}
			winner, decided := fight.Winner()
			if !decided {
				continue
			}
			fought++
			if (winner == hex.SideAlly) == mineFirst {
				won++
			}
		}
	}
	if fought == 0 {
		t.Fatalf("no battle against %s ended, so nothing was measured", opponent)
	}
	return won * 1000 / fought
}
