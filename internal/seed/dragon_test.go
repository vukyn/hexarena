package seed_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/seed"
)

// The two ways a Charizard at the cap can be built, and the four slots each
// spends. Written out rather than derived, because that is the claim: these are
// the two kits an author would field, and the test is about them and not about
// whatever the first four declared happen to be.
var (
	fireBuild   = []string{"flamethrower", "inferno", "ember", "fire_spin"}
	dragonBuild = []string{"dragon_claw", "outrage", "dragon_rage", "dragon_dance"}
)

// buildSeeds is how many battles each half of the comparison is fought over.
const buildSeeds = 300

// TestTheDragonBuildIsASidegradeAndNotAnUpgrade fights the two builds against
// each other, which is the only measurement that can answer the question.
//
// `hexforge spar` cannot: it fields the first four skills a learnset declares
// for both sides, and Charmander beats the rest of the cast about a hundred
// times in a hundred with any kit worth writing, so every figure it reports for
// this character is a ceiling. Two builds of one character meeting in the middle
// of the board is the only pairing where the difference is the whole of what is
// being measured.
//
// Both ways round, for the reason forge.Spar is: the turn queue breaks a tie by
// enlistment, and in a Charmander mirror the first slot is worth close to fifty
// points. A one-way figure here would be that advantage plus the build.
//
// ⚠️ The figure is a floor rather than a balance point, and the reason is worth
// knowing before anybody tunes against it: battle.Suggest never buffs, so
// dragon_dance is never cast and the dragon build fights this with three of its
// four slots. The fire build has no setup slot and spends all four. What the
// band therefore says is that a build the engine cannot play properly still
// takes four battles in ten -- and that the day Suggest learns to buff, this
// number moves up on its own and the band is where to notice it.
func TestTheDragonBuildIsASidegradeAndNotAnUpgrade(t *testing.T) {
	dragon, fire := 0, 0
	for _, arrangement := range []struct {
		first, second []string
		firstTrait    string
		secondTrait   string
		dragonIsFirst bool
	}{
		{dragonBuild, fireBuild, "reckless", "blaze", true},
		{fireBuild, dragonBuild, "blaze", "reckless", false},
	} {
		for seed := 1; seed <= buildSeeds; seed++ {
			winner, decided := theWinner(t,
				arrangement.first, arrangement.firstTrait,
				arrangement.second, arrangement.secondTrait, uint64(seed))
			if !decided {
				continue
			}
			if (winner == hex.SideAlly) == arrangement.dragonIsFirst {
				dragon++
			} else {
				fire++
			}
		}
	}

	fought := dragon + fire
	if fought == 0 {
		t.Fatal("no battle between the two builds ended, so nothing was measured")
	}
	rate := dragon * scale.Base / fought
	// A wide band on purpose. This is not a tuning target that has to be hit to
	// the point; it is the statement that neither build is the obvious one, and
	// it should only fail when somebody has moved a number far enough that one
	// of them stops being a choice.
	const lowest, highest = 350, 650
	if rate < lowest || rate > highest {
		t.Errorf("the dragon build wins %d.%d%% of %d battles against the fire build, outside %d..%d: "+
			"one of the two builds is no longer a decision",
			rate/10, rate%10, fought, lowest, highest)
	}
	t.Logf("dragon %d, fire %d over %d battles: %d.%d%%", dragon, fire, fought, rate/10, rate%10)
}

// TestRecklessIsATradeAndNotAGift is the whole of what the trait is for, and it
// is asserted here because nothing else can see it.
//
// A win rate cannot: removing the cost and leaving the bonus makes the holder
// strictly better, which every battle test happily agrees with. A golden cannot
// either -- it moves whenever any number in the file moves, so it says a change
// happened and never that the change was wrong. This reads the two statuses the
// trait grants and asks the one question that matters: is something given up.
func TestRecklessIsATradeAndNotAGift(t *testing.T) {
	traits, statuses := mustPassives(t), mustStatuses(t)
	held, err := traits.Lookup("reckless")
	if err != nil {
		t.Fatalf("look up reckless: %v", err)
	}
	raised, lowered := map[modifier.Target]bool{}, map[modifier.Target]bool{}
	for _, grant := range held.Grants {
		kind, err := statuses.Lookup(grant.Status)
		if err != nil {
			t.Fatalf("look up %s: %v", grant.Status, err)
		}
		for _, term := range kind.Modifiers {
			if term.Amount > 0 {
				raised[term.Target] = true
			}
			if term.Amount < 0 {
				lowered[term.Target] = true
			}
		}
	}
	if !raised[modifier.Attack] {
		t.Error("reckless raises no attack, so it is a cost with nothing bought")
	}
	if !lowered[modifier.Defense] {
		t.Error("reckless lowers no defence, so it is a gift rather than a trade: " +
			"the whole point of the trait is that the attack is paid for")
	}
	// Nothing may be raised and lowered at once, which would be two grants
	// arguing about one stat and leaving the trade unreadable on screen.
	for target := range raised {
		if lowered[target] {
			t.Errorf("reckless both raises and lowers %s", target)
		}
	}
}

// TestEveryDragonSkillIsNeutral is the mechanism the whole build rests on,
// stated where a reader will find it.
//
// There is no dragon element -- dragon is a species -- so the line cannot buy an
// advantage on the chart and does not try to. What it buys instead is evenness:
// neutral is inert, so a dragon skill is 1.0x into everything while a fire one is
// 1.5x into grass and 0.667x into water. A dragon skill that took an element
// would be a fire skill wearing the wrong restriction, and the build would stop
// meaning anything.
func TestEveryDragonSkillIsNeutral(t *testing.T) {
	book := mustSkills(t)
	found := 0
	for _, current := range book.Skills() {
		if current.Restrict == nil || !slices.Contains(current.Restrict.Species, "dragon") {
			continue
		}
		found++
		if current.Element != element.Neutral {
			t.Errorf("%q is kept for a dragon and is %s: the dragon line buys evenness by giving up the chart, "+
				"and an element takes that back", current.ID, current.Element)
		}
	}
	if found == 0 {
		t.Fatal("no skill is kept for a dragon, so this asserts nothing")
	}
}

// TestTheDragonBuildIsFlatterThanTheFireOne is the other half of what a
// sidegrade means, and the half a win rate cannot say.
//
// Fire is spiky by construction: the chart pays it 1.5x into grass and charges
// it 0.667x into water, so a fire Charizard is a different unit depending on who
// it meets. Every dragon skill is neutral, which the chart leaves alone. So the
// dragon build's two matchups should sit closer together than the fire build's,
// and if they ever do not then the neutral element has stopped being the point
// of the build.
func TestTheDragonBuildIsFlatterThanTheFireOne(t *testing.T) {
	fireSpread := theSpread(t, fireBuild, "blaze")
	dragonSpread := theSpread(t, dragonBuild, "reckless")
	if dragonSpread >= fireSpread {
		t.Errorf("the dragon build's matchups are %d apart and the fire build's %d, "+
			"so the neutral element is buying no evenness at all", dragonSpread, fireSpread)
	}
	t.Logf("spread in turns to a kill: fire %d, dragon %d", fireSpread, dragonSpread)
}

// theSpread is how differently a build treats the two elements it can meet,
// measured in turns to finish rather than in wins.
//
// Turns rather than a rate, because a rate cannot see it: Charmander wins both
// matchups either way, and the whole difference between the builds is how long
// each one takes. A build that kills grass in eight turns and water in thirty is
// two units; one that takes twenty against both is one.
func theSpread(t *testing.T, kit []string, trait string) int {
	t.Helper()
	against := map[string]int{}
	for _, opponent := range []string{"pokemon.bulbasaur", "pokemon.squirtle"} {
		total, counted := 0, 0
		for seed := 1; seed <= buildSeeds/10; seed++ {
			turns, ended := theLength(t, kit, trait, opponent, uint64(seed))
			if ended {
				total += turns
				counted++
			}
		}
		if counted == 0 {
			t.Fatalf("no battle against %s ended", opponent)
		}
		against[opponent] = total / counted
	}
	spread := against["pokemon.bulbasaur"] - against["pokemon.squirtle"]
	if spread < 0 {
		spread = -spread
	}
	return spread
}

// buildSlot is where every unit in these tests stands: the front column, middle
// row, which is the one slot that asks nothing of a kit's range.
var buildSlot = hex.Offset{Col: hex.FormationCols - 1, Row: hex.Rows / 2}

// theWinner fights one build against another and reports which side took it.
func theWinner(t *testing.T, first []string, firstTrait string,
	second []string, secondTrait string, seedValue uint64) (hex.Side, bool) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.charmander")
	fight, err := battle.New(books, seedValue, []battle.Roster{
		{ID: "first", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: first, Passives: []string{firstTrait}},
		{ID: "second", Side: hex.SideEnemy, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: second, Passives: []string{secondTrait}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(4000); err != nil {
		t.Fatalf("run: %v", err)
	}
	return fight.Winner()
}

// theLength fights one build against a character out of the shipped cast and
// reports how long it took.
func theLength(t *testing.T, kit []string, trait, opponent string, seedValue uint64) (int, bool) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.charmander")
	theirStats, theirAffinity, theirKit, theirTrait := fielded(t, opponent)
	fight, err := battle.New(books, seedValue, []battle.Roster{
		{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: kit, Passives: []string{trait}},
		{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity, Stats: theirStats,
			Skills: theirKit, Passives: theirTrait},
	})
	if err != nil {
		t.Fatalf("new battle against %s: %v", opponent, err)
	}
	fight.Begin()
	turns, err := fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run against %s: %v", opponent, err)
	}
	return turns, fight.Finished()
}

// fielded is a shipped character as forge.Spar would field it: the resolved stat
// line of its furthest form, its element, the first four skills its learnset
// declares and the first trait. One helper for both halves, so the two builds
// and the two opponents are all read the same way.
func fielded(t *testing.T, id string) (progression.Values, element.Affinity, []string, []string) {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get(id)
	if !known {
		t.Fatalf("no character %q", id)
	}
	stats, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve %s: %v", id, err)
	}
	kit := character.SkillsAt(progression.LevelCap, stage.Name)
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}
	traits := character.PassivesAt(progression.LevelCap, stage.Name)
	if len(traits) > cast.TraitSlots {
		traits = traits[:cast.TraitSlots]
	}
	return stats, character.Element, kit, traits
}
