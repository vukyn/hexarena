package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/seed"
)

// The three directions, hardcoded here the way every other design record in this
// package is. TestTheShippedBuildsAreTheOnesTheTestsMeasure is what stops the
// catalogue a player reads drifting away from them.
//
// They are not three answers to one question the way Magnemite's are. Mew has no
// question of its own — no counter to spend, no element to lean on — so what its
// catalogue holds is three different *characters*, and the measurement below is
// that no two of them spend a turn on the same thing.
var (
	feedBuild     = []string{"hypnosis", "dream_eater", "psychic", "recover"}
	borrowedBuild = []string{"cross_chop", "submission", "vital_throw", "body_slam"}
	witherBuild   = []string{"poison_powder", "sludge_bomb", "venoshock", "swift"}
)

// TestTheThreeMewBuildsSpendTheirTurnsOnDifferentThings is the measurement behind
// the catalogue, and each build is held to the column it leads rather than to a
// figure.
//
// Sixty duels each against the shipped Charizard, the same opposition every other
// build reading in this package is taken against — two kits are comparable only
// against something held still.
//
// ⚠️ **Which column a build tops is the claim, not by how much.** The three are
// separated on four axes and each leads exactly one: `feed` is the only one that
// heals at all, `wither` inflicts several times the statuses, and `borrowed`
// deals the most damage and misses by far the most doing it — which is what
// carrying somebody else's heaviest skills on a middling frame comes to.
func TestTheThreeMewBuildsSpendTheirTurnsOnDifferentThings(t *testing.T) {
	feed := readBuild(t, "pokemon.mew", progression.Furthest, feedBuild, "elusive")
	borrowed := readBuild(t, "pokemon.mew", progression.Furthest, borrowedBuild, "endurance")
	wither := readBuild(t, "pokemon.mew", progression.Furthest, witherBuild, "contagion")
	for _, reading := range []struct {
		name string
		got  buildReading
	}{{"feed", feed}, {"borrowed", borrowed}, {"wither", wither}} {
		t.Logf("%-9s %3d turns, dealt %5d, healed %5d, missed %3d, inflicted %3d",
			reading.name, reading.got.turns, reading.got.dealt, reading.got.healed,
			reading.got.missed, reading.got.inflicted)
	}

	// The one that feeds. It is the only build here that takes health back, and
	// nought from the other two is what makes that a direction rather than a
	// degree — a build that healed a little would be the same build, worse.
	if feed.healed == 0 {
		t.Errorf("the feed build healed nothing, so nothing here measured what it is for")
	}
	if borrowed.healed != 0 || wither.healed != 0 {
		t.Errorf("the borrowed and wither builds healed %d and %d: sustain is supposed to be what only one of the three does",
			borrowed.healed, wither.healed)
	}
	if feed.turns <= borrowed.turns || feed.turns <= wither.turns {
		t.Errorf("the feed build took %d turns against %d and %d: outlasting is what it trades its damage for",
			feed.turns, borrowed.turns, wither.turns)
	}

	// The one that borrows. Nothing it carries was written for Mew, so what it
	// has is the cast's heaviest neutral blows and the accuracy that comes with
	// them.
	if borrowed.dealt <= 2*feed.dealt {
		t.Errorf("the borrowed build dealt %d against the feed build's %d: it is supposed to be the direction that only hits",
			borrowed.dealt, feed.dealt)
	}
	if borrowed.missed <= wither.missed || borrowed.missed <= feed.missed {
		t.Errorf("the borrowed build missed %d times against %d and %d: the price of the heaviest skills in the book is not being paid",
			borrowed.missed, wither.missed, feed.missed)
	}

	// The one that withers. It inflicts, and it inflicts far more than either.
	if wither.inflicted <= borrowed.inflicted || wither.inflicted <= feed.inflicted {
		t.Errorf("the wither build inflicted %d statuses against %d and %d: it is not the direction that lays things on",
			wither.inflicted, borrowed.inflicted, feed.inflicted)
	}
}

// TestOneMewBuildCarriesNothingOfItsOwn is the claim the character exists to
// make, and it is the reason `borrowed` is in the catalogue at all.
//
// Mew declares the inert element, so the only skills it may carry are the neutral
// ones — which is every character's plain moves and nobody's signature. That
// turns a restriction into the widest kit in the cast, and `borrowed` is what
// says so: four skills, not one of them written for Mew, every one of them
// already carried by somebody else.
//
// ⚠️ It is a direction rather than an absence, the same way Magnemite's `surge`
// is: the same character built as though it had never been given anything of its
// own. A build that quietly picked up one of Mew's five would still work and
// would no longer be saying anything, which is why this counts rather than
// spot-checks.
func TestOneMewBuildCarriesNothingOfItsOwn(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	elsewhere := map[string][]string{}
	for _, character := range book.All() {
		if character.ID == "pokemon.mew" {
			continue
		}
		for _, carried := range character.Skills {
			elsewhere[carried.ID] = append(elsewhere[carried.ID], character.ID)
		}
	}
	for _, carried := range borrowedBuild {
		holders, known := elsewhere[carried]
		if !known {
			t.Errorf("the borrowed build carries %q, which nobody but Mew has: the build that borrows everything is borrowing nothing",
				carried)
			continue
		}
		t.Logf("%-12s borrowed from %v", carried, holders)
	}

	// And the other half of the claim: Mew's own five are real, so "carries
	// nothing of its own" is a choice rather than the only thing available.
	mew, known := book.Get("pokemon.mew")
	if !known {
		t.Fatalf("no pokemon.mew in the cast")
	}
	own := 0
	for _, carried := range mew.Skills {
		if _, shared := elsewhere[carried.ID]; !shared {
			own++
		}
	}
	if own == 0 {
		t.Fatalf("Mew carries nothing nobody else does, so there is nothing for a build to decline")
	}
	t.Logf("Mew's learnset is %d skills, %d of them its own", len(mew.Skills), own)
}

// TestMewNeitherGainsNorLosesAgainstAnyShippedAffinity is what declaring the
// inert element buys, stated once against the whole cast.
//
// Every other character reads the chart twice a strike — once for what it throws
// and once for what it is hit by — and lives somewhere between two thirds and
// three halves of its own numbers. Mew reads a flat thousand in both directions
// against everybody, which is the only sense in which a character here has "no
// type" rather than a quiet one.
//
// ⚠️ **This is not a claim about matchups and the measurements say so.** Mew's
// duel rates against the cast run from nought to a hundred, exactly as polarised
// as every other character's, because what decides a pairing here is tempo and
// sustain rather than the chart. Removing the elemental term removes the
// elemental term.
func TestMewNeitherGainsNorLosesAgainstAnyShippedAffinity(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the element chart: %v", err)
	}
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	mew, known := book.Get("pokemon.mew")
	if !known {
		t.Fatalf("no pokemon.mew in the cast")
	}
	if mew.Element.IsDual() || mew.Element.Primary() != element.Neutral {
		t.Fatalf("Mew declares %s, and everything below is about the inert element", mew.Element)
	}
	moved := 0
	for _, character := range book.All() {
		out := chart.MultiplierAgainst(element.Neutral, character.Element)
		if out != scale.Base {
			t.Errorf("Mew hits %s at %d per mille", character.ID, out)
		}
		for _, member := range character.Element.Elements() {
			in := chart.MultiplierAgainst(member, mew.Element)
			if in != scale.Base {
				t.Errorf("%s hits Mew with %s at %d per mille", character.ID, member, in)
			}
			// The same reading taken against somebody who is not Mew has to move
			// at least once, or the loop above is measuring a chart with no edges
			// in it rather than an element with no edges in it.
			for _, other := range book.All() {
				if chart.MultiplierAgainst(member, other.Element) != scale.Base {
					moved++
				}
			}
		}
	}
	if moved == 0 {
		t.Fatal("no pairing in the shipped cast reads anything but a flat thousand, so the flat thousands above prove nothing")
	}
}

// TestMewHoldsNoExtremeOnAnyStat is the stat line, and it is a claim about the
// cast rather than about the ceilings.
//
// Every stat sits strictly between the lowest and the highest the shipped top
// forms field, which is the engine's way of saying what "a hundred in everything"
// says in the fiction: nothing to lean on and nothing to cover.
//
// ⚠️ **The obvious version of this was measured and thrown away.** The first
// draft put Mew at seventy per cent of every declared ceiling, which reads as the
// same idea and is not: the cast uses the ceilings very unevenly — dodge tops out
// at less than half of its ceiling and attack at nineteen twentieths of its — so
// an even share of them landed a unit with the cast's best dodge by half again
// and its best speed outright. It sparred at 72.4% and the line here reads 56.4%.
func TestMewHoldsNoExtremeOnAnyStat(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	mew, known := book.Get("pokemon.mew")
	if !known {
		t.Fatalf("no pokemon.mew in the cast")
	}
	mine, _, err := mew.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve Mew: %v", err)
	}
	type bound struct {
		low, high       int64
		lowest, highest string
	}
	bounds := map[progression.Kind]bound{}
	for _, character := range book.All() {
		if character.ID == "pokemon.mew" {
			continue
		}
		forms, err := character.Stages.Furthest(progression.LevelCap)
		if err != nil {
			t.Fatalf("furthest forms of %s: %v", character.ID, err)
		}
		for _, form := range forms {
			theirs := form.Stats.At(progression.LevelCap)
			for kind := progression.Kind(0); int(kind) < progression.KindCount; kind++ {
				held, seen := bounds[kind]
				if !seen || theirs[kind] < held.low {
					held.low, held.lowest = theirs[kind], form.Name
				}
				if !seen || theirs[kind] > held.high {
					held.high, held.highest = theirs[kind], form.Name
				}
				bounds[kind] = held
			}
		}
	}
	for kind := progression.Kind(0); int(kind) < progression.KindCount; kind++ {
		held := bounds[kind]
		if mine[kind] <= held.low {
			t.Errorf("Mew's %s is %d, at or under %s's %d — it is supposed to hold no extreme",
				kind, mine[kind], held.lowest, held.low)
		}
		if mine[kind] >= held.high {
			t.Errorf("Mew's %s is %d, at or over %s's %d — it is supposed to hold no extreme",
				kind, mine[kind], held.highest, held.high)
		}
		t.Logf("%-9s %4d < %4d < %4d  (%s .. %s)",
			kind, held.low, mine[kind], held.high, held.lowest, held.highest)
	}
}

// mewSetupSeeds is how many duels the reading below is taken over. Sixty, like
// every other build reading here.
const mewSetupSeeds = 60

// TestAOneTurnSetupIsAQueueRaceRatherThanACombo is what `dream_eater` turned out
// to be, and it is the thing worth writing down rather than the character.
//
// The skill reads a condition — hit harder into a unit that is stunned — and Mew
// is the one carrying the stun. That looks like a combo and is not one. `stun`
// lasts a single turn and the turn it lasts is the *target's*, which the target
// then spends being skipped; so the whole window in which the condition holds is
// one slot of the turn queue, and who owns that slot is decided by speed alone.
//
// Measured over sixty duels each, with the same kit and the same stun source:
//
//	Blastoise, speed  85   239 stuns ->  50 amplified
//	Venusaur,  speed 100    66 stuns ->   2 amplified
//	Sennin,    speed 134    77 stuns ->   2 amplified
//	Charizard, speed 140    31 stuns ->   0 amplified
//
// ⚠️ **Raising the stun chance makes it worse, not better.** Swapping the source
// for `hypnosis`, which lands more than twice as many, took the conversion the
// wrong way — the turn that lays the status down is the same turn that could have
// spent it, so a kit that stuns more attacks less and reaches the window less
// often. That is why `hypnosis` is not among the four the character brings by
// default, and why shortening the skill's cooldown moved almost nothing.
//
// What is held here is the direction, not the figures: the condition pays against
// a unit slower than Mew and does not against a faster one.
func TestAOneTurnSetupIsAQueueRaceRatherThanACombo(t *testing.T) {
	slowerAmplified, slowerStuns := readSetup(t, "pokemon.squirtle")
	fasterAmplified, fasterStuns := readSetup(t, "pokemon.charmander")
	t.Logf("against the slower unit: %d stuns -> %d amplified", slowerStuns, slowerAmplified)
	t.Logf("against the faster unit: %d stuns -> %d amplified", fasterStuns, fasterAmplified)

	if slowerStuns == 0 || fasterStuns == 0 {
		t.Fatalf("the stun landed %d and %d times: nothing here measured a setup at all",
			slowerStuns, fasterStuns)
	}
	if slowerAmplified == 0 {
		t.Errorf("the condition never held against the slower unit either, so the skill has a clause that does nothing")
	}
	if fasterAmplified >= slowerAmplified {
		t.Errorf("the condition held %d times against the faster unit and %d against the slower one: the window is supposed to belong to whoever is quicker",
			fasterAmplified, slowerAmplified)
	}
}

// readSetup fights Mew's shipped four against one opponent and counts how often
// the stun it lands is still there when the skill that reads it goes off.
func readSetup(t *testing.T, foe string) (amplified, stuns int) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, kit, traits := fielded(t, "pokemon.mew")
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, foe)
	for seedValue := 1; seedValue <= mewSetupSeeds; seedValue++ {
		fight, err := battle.New(books, uint64(seedValue), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: kit, Passives: traits},
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
			case battle.Amplified:
				if event.Skill == "dream_eater" {
					amplified++
				}
			case battle.StatusApplied:
				if event.Status == "stun" {
					stuns++
				}
			}
		}
	}
	return amplified, stuns
}

// TestTheCastHasALineThatDoesNotEvolve is the first one, and the point is that
// nothing in the engine had to change for it: progression.Line has always
// resolved a single form, and no shipped character had ever declared one.
//
// A level still means something on such a line — the stat curve runs from base to
// max exactly as it does on three forms, and the learnset still opens over it —
// so what is given up is the fork and the threshold, not progression.
func TestTheCastHasALineThatDoesNotEvolve(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	mew, known := book.Get("pokemon.mew")
	if !known {
		t.Fatalf("no pokemon.mew in the cast")
	}
	if len(mew.Stages) != 1 {
		t.Fatalf("Mew declares %d forms; this test is about the one-form line", len(mew.Stages))
	}
	young, form, err := mew.Resolve(1, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve Mew at level 1: %v", err)
	}
	grown, sameForm, err := mew.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve Mew at the cap: %v", err)
	}
	if form.Name != sameForm.Name {
		t.Errorf("Mew resolves as %q at level 1 and %q at the cap, on a line with one form",
			form.Name, sameForm.Name)
	}
	if young[progression.HP] >= grown[progression.HP] {
		t.Errorf("Mew's health reads %d at level 1 and %d at the cap: a line that does not evolve still grows",
			young[progression.HP], grown[progression.HP])
	}
	early := mew.SkillsAt(1, form.Name)
	late := mew.SkillsAt(progression.LevelCap, form.Name)
	if len(early) >= len(late) {
		t.Errorf("Mew knows %d skills at level 1 and %d at the cap: the learnset is supposed to open over a level rather than over a form",
			len(early), len(late))
	}
	t.Logf("one form, %d skills at level 1 and %d at the cap", len(early), len(late))
}
