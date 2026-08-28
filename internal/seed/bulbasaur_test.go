package seed_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// The two ways a Venusaur at the cap can be built, written out for the reason
// fireBuild/dragonBuild and tankBuild/semiBuild are: these are the kits an
// author would field, and every claim below is about them rather than about
// whichever four the learnset happens to declare first.
//
// The two share razor_leaf and nothing else. That is not laziness — neither
// direction is a weapon, so both have to spend one slot on the same best hit the
// learnset offers, and what separates them is what the other three slots buy.
//
// Traits are one apiece, out of the four kinds Bulbasaur's five traits cover:
// virulence amplifies the poison the first kit spreads, blood_thirst drains for
// the second, which already gives health back with its skills. Both directions
// have a second entry (venom_blood, last_gasp) and TestBulbasaurCanBeBuiltTwoWays
// in cast_test.go is the claim that neither ever loses its last one.
var (
	poisonBuild  = []string{"poison_powder", "sludge_bomb", "venoshock", "razor_leaf"}
	sustainBuild = []string{"leech_seed", "synthesis", "ingrain", "razor_leaf"}
)

// bulbasaurSeeds is how many battles each figure below is averaged over.
const bulbasaurSeeds = 30

// TestTheTwoBulbasaurBuildsAreDifferentUnits is the claim the trait slot and the
// learnset were for: one character, two kits, and the two do different jobs
// rather than the same job at different speeds.
//
// ⚠️ They are NOT compared by fighting each other, and the reason is worth
// writing down because the figure looks like a verdict: over 600 mirror duels
// fought both ways the poison kit wins about one in ten. That says nothing about
// whether either is worth building — a mirror duel is decided by which side
// outlasts the other, the sustain kit is the one built to outlast, and the poison
// kit is built to kill something that is trying to kill it quickly. Fighting a
// build against the thing it is for is the measurement; fighting it against its
// own twin measures the twin.
//
// ⚠️ Both figures are taken on autopilot, which used to mean the sustain kit was
// barely played at all: a skill of no power was reached only when nothing could be
// hit, so synthesis and ingrain were cast in the turns nothing was in range rather
// than in the turns they were wanted. The opponent now prices a heal by the health
// an enemy could otherwise take off, and the kit's figures moved with it — 17 turns
// recovering 964 became 23 turns recovering 2818 on the same seeds. What autopilot
// still cannot do is save a skill for a turn that has not arrived.
func TestTheTwoBulbasaurBuildsAreDifferentUnits(t *testing.T) {
	poisonTurns, poisonDealt, poisonHealed := bulbasaurFight(t, poisonBuild, "virulence")
	sustainTurns, sustainDealt, sustainHealed := bulbasaurFight(t, sustainBuild, "blood_thirst")

	if sustainTurns <= poisonTurns {
		t.Errorf("the sustain build lasted %d turns and the poison build %d, so the one built "+
			"to stay on the board is not staying longer", sustainTurns, poisonTurns)
	}
	// Per turn rather than in total: the sustain build is on the board longer, so
	// its total creeps up on the other's and two builds doing one job slowly and
	// quickly would read the same as two builds doing two jobs.
	poisonRate := poisonDealt / int64(poisonTurns)
	sustainRate := sustainDealt / int64(sustainTurns)
	if poisonRate <= sustainRate*2 {
		t.Errorf("the poison build deals %d a turn and the sustain build %d, so the one built "+
			"to kill is not killing appreciably faster", poisonRate, sustainRate)
	}
	// The sustain build's whole premise is health coming back, and the poison
	// build's is that none of it does: nothing in that kit drains, restores or
	// regrows, so a figure above nought there means a skill or a trait moved
	// between the two directions and the split stopped being a split.
	if sustainHealed <= 0 {
		t.Error("the sustain build recovered nothing at all: the health coming back is the build")
	}
	if poisonHealed != 0 {
		t.Errorf("the poison build recovered %d, and it holds nothing that gives health back",
			poisonHealed)
	}
	t.Logf("poison %d turns dealing %d (%d a turn), sustain %d turns dealing %d (%d a turn) "+
		"recovering %d", poisonTurns, poisonDealt, poisonRate,
		sustainTurns, sustainDealt, sustainRate, sustainHealed)
}

// TestVenoshockIsUnusableOutsideThePoisonBuild is the mechanism that makes the
// first kit a direction rather than three skills that happen to be filed
// together: venoshock's bonus is a condition read against the target, so the
// skill is worth its full power only behind something that put the poison there,
// and the only two things in this learnset that do are both in the same kit.
//
// Asserted against the declared data rather than measured, because a sweep
// cannot tell this apart from noise. venoshock lands its plain power either way
// and a second skill of any kind is a second action, so the difference a sweep
// reports is a few percent — while the fact itself is absolute: in the sustain
// kit the condition can never hold.
func TestVenoshockIsUnusableOutsideThePoisonBuild(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	venoshock, err := books.Skills.Lookup("venoshock")
	if err != nil {
		t.Fatalf("look venoshock up: %v", err)
	}
	if venoshock.Requires == nil || venoshock.Requires.Status != "poison" {
		t.Fatalf("venoshock reads %v, and the whole build is that it reads a poison",
			venoshock.Requires)
	}
	if venoshock.Requires.BonusPower <= 0 {
		t.Error("venoshock's condition carries no bonus, so nothing is gained by setting it up")
	}

	appliers := []string{}
	for _, id := range bulbasaurLearnset(t) {
		skill, err := books.Skills.Lookup(id)
		if err != nil {
			t.Fatalf("look %s up: %v", id, err)
		}
		for _, application := range skill.Applies {
			if application.Status == "poison" {
				appliers = append(appliers, id)
				break
			}
		}
	}
	if len(appliers) == 0 {
		t.Fatal("nothing in the learnset applies a poison, so venoshock's bonus is unreachable")
	}
	for _, id := range appliers {
		if !slices.Contains(poisonBuild, id) {
			t.Errorf("%s applies the poison venoshock reads and is not in the poison build", id)
		}
		if slices.Contains(sustainBuild, id) {
			t.Errorf("%s applies a poison and sits in the sustain build, so the two kits are "+
				"not two directions", id)
		}
	}
}

// TestEverySustainBuildSkillGivesHealthBack is the same claim from the other
// side: the second kit is not "the leftovers", it is three slots that each buy
// health back by a different route — a drain off damage dealt, a flat restore,
// and a regeneration that ticks — plus the one weapon both directions have to
// carry.
//
// The trait is checked with them. A drain on the trait is what makes the kit's
// own weapon feed the same plan, and a sustain direction whose trait stopped
// draining would be a poison direction with worse skills.
func TestEverySustainBuildSkillGivesHealthBack(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	for _, id := range sustainBuild {
		if slices.Contains(poisonBuild, id) {
			continue // the shared weapon, which is in the kit for its damage
		}
		skill, err := books.Skills.Lookup(id)
		if err != nil {
			t.Fatalf("look %s up: %v", id, err)
		}
		regrows := false
		for _, application := range append(skill.SelfApplies, skill.Applies...) {
			entry, err := books.Statuses.Lookup(application.Status)
			if err != nil {
				t.Fatalf("look %s up: %v", application.Status, err)
			}
			if entry.Category == status.Regen {
				regrows = true
			}
		}
		if skill.Drains == 0 && skill.Restores == 0 && !regrows {
			t.Errorf("%s neither drains, restores nor regrows, so it is in the sustain build "+
				"for no reason the data can see", id)
		}
	}

	trait, err := books.Passives.Lookup("blood_thirst")
	if err != nil {
		t.Fatalf("look blood_thirst up: %v", err)
	}
	if trait.Drains <= 0 {
		t.Error("the sustain build's trait drains nothing, so its weapon feeds nothing")
	}
}

// TestBothBulbasaurBuildsAreLegalAtTheCap is the boring one that stops the rest
// from being about kits an author could not field: four skills and one trait
// each, every entry unlocked by the form the cap resolves to.
//
// It matters more here than the slot counts suggest, because sleep_powder is
// stage-gated to the two earlier forms — the learnset a Venusaur reads is not
// the learnset the character declares, and a build written from the file rather
// than from the form would field a move this unit never learns.
func TestBothBulbasaurBuildsAreLegalAtTheCap(t *testing.T) {
	available := bulbasaurLearnset(t)
	for _, build := range [][]string{poisonBuild, sustainBuild} {
		if len(build) != cast.SkillSlots {
			t.Errorf("the build %v spends %d of %d skill slots", build, len(build), cast.SkillSlots)
		}
		for _, id := range build {
			if !slices.Contains(available, id) {
				t.Errorf("%s is in a build and not in what the capped form knows: %v",
					id, available)
			}
		}
	}
}

// bulbasaurLearnset is every skill the form at the cap actually knows.
func bulbasaurLearnset(t *testing.T) []string {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get("pokemon.bulbasaur")
	if !known {
		t.Fatal("the shipped cast holds no bulbasaur")
	}
	_, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve bulbasaur: %v", err)
	}
	return character.SkillsAt(progression.LevelCap, stage.Name)
}

// bulbasaurFight is a build's average battle: how long it lasted, how much it
// dealt and how much health it got back, over the sweep.
func bulbasaurFight(t *testing.T, kit []string, trait string) (turns int, dealt, healed int64) {
	t.Helper()
	lasted := 0
	for seedValue := 1; seedValue <= bulbasaurSeeds; seedValue++ {
		hurt, back, ran := bulbasaurRun(t, kit, trait, uint64(seedValue))
		dealt += hurt
		healed += back
		lasted += ran
	}
	return lasted / bulbasaurSeeds, dealt / bulbasaurSeeds, healed / bulbasaurSeeds
}

// bulbasaurRun fights one build against the shipped Charizard and reports what it
// dealt, what it recovered and how long the battle ran.
//
// Charizard rather than the whole cast for the reason squirtleRun fights one
// opponent: two kits are comparable only against something held still. It is also
// the fight that matters to a grass unit — fire takes 1.5× off it and grass has
// no elemental answer anywhere on the chart.
//
// ⚠️ A poison tick is a StatusTicked on the unit CARRYING the poison, and it
// names no author — Actor is the victim, and nothing on the event says who put it
// there. So damage counted as Damaged-by-me misses every tick this build's whole
// plan is made of: the poison kit reads 106 a turn that way against 139 counted
// properly, i.e. a quarter of its output invisible, and the build it is being
// compared against loses almost nothing to the same mistake. In a duel the side
// is enough to attribute a tick; in a squad it would not be.
func bulbasaurRun(t *testing.T, kit []string, trait string, seedValue uint64) (dealt, healed int64, turns int) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.bulbasaur")
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, "pokemon.charmander")
	fight, err := battle.New(books, seedValue, []battle.Roster{
		{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: kit, Passives: []string{trait}},
		{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
			Stats: theirStats, Skills: theirKit, Passives: theirTraits},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	turns, err = fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, event := range fight.Drain() {
		switch {
		case event.Kind == battle.Damaged && event.Actor == "mine":
			dealt += event.Amount
		case event.Kind == battle.StatusTicked && event.Actor == "theirs":
			dealt += event.Amount
		case event.Kind == battle.Healed && event.Actor == "mine":
			healed += event.Amount
		}
	}
	return dealt, healed, turns
}
