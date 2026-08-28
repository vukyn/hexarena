package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The two ways a Blastoise at the cap can be built, and the four slots each
// spends. Written out for the reason fireBuild and dragonBuild are: these are
// the two kits an author would field, and the claim is about them rather than
// about whichever four the learnset happens to declare first.
//
// They share a stat line and cannot do otherwise. Squirtle absorbs 11285 of the
// 11500 effective-health budget, so there is no room to make one of them tougher
// than the other — which is the whole reason the split has to live in the slots.
var (
	tankBuild = []string{"taunt", "withdraw", "wide_guard", "aqua_ring"}
	semiBuild = []string{"skull_bash", "water_gun", "whirlpool", "withdraw"}
)

// squirtleSeeds is how many battles each figure below is averaged over.
const squirtleSeeds = 30

// TestSkullBashScalesOffDefenceAndNotAttack is the mechanism the semi-tank build
// is built on, and it is asserted because nothing else in the repository can see
// it: skull_bash is the **first shipped skill to scale off anything but attack**,
// so until it arrived the field was carried, parsed, marshalled and described
// without one line of shipped data exercising it.
//
// It halves attack rather than raising defence, because raising defence is not
// allowed: battle.New checks the roster's stat line against the effective-health
// budget, and Squirtle is 215 short of it. Halving a stat is always legal.
//
// The comparison needs the second skill. Damage falling when a stat falls proves
// nothing on its own — a weaker unit dies sooner and swings fewer times, which
// would move any figure. What cannot be explained that way is skull_bash coming
// back *exactly equal* while water_gun, fought the same way over the same seeds,
// does not.
func TestSkullBashScalesOffDefenceAndNotAttack(t *testing.T) {
	full, _, _, _ := fielded(t, "pokemon.squirtle")
	halved := full
	halved[progression.Attack] = full[progression.Attack] / 2

	bashFull := squirtleDamage(t, "skull_bash", full)
	bashHalved := squirtleDamage(t, "skull_bash", halved)
	gunFull := squirtleDamage(t, "water_gun", full)
	gunHalved := squirtleDamage(t, "water_gun", halved)

	if bashFull == 0 || gunFull == 0 {
		t.Fatalf("nothing was measured: skull_bash %d, water_gun %d", bashFull, gunFull)
	}
	if bashHalved != bashFull {
		t.Errorf("skull_bash dealt %d on full attack and %d on half of it, so it is reading attack "+
			"somewhere: the skill scales off defence and halving attack must change nothing",
			bashFull, bashHalved)
	}
	if gunHalved >= gunFull {
		t.Errorf("water_gun dealt %d on full attack and %d on half of it, so attack is reaching "+
			"nothing either and the comparison above is vacuous", gunFull, gunHalved)
	}
	t.Logf("skull_bash %d -> %d, water_gun %d -> %d", bashFull, bashHalved, gunFull, gunHalved)
}

// TestBallastIsATradeAndNotAGift is what the trait is for, asserted the way
// TestRecklessIsATradeAndNotAGift asserts its own: by reading the statuses rather
// than by fighting.
//
// A battle cannot see it. Deleting the cost and keeping the bonus makes the
// holder strictly better, and every win rate in the repository would agree that
// the change was an improvement. This asks the one question a rate cannot: is
// anything given up.
//
// ⚠️ The trade is real and it is *heavy*, which is a measured fact and not a
// guess. Squirtle's survival is gated on how often it can cast withdraw, so a
// tenth off its speed is worth more to it than a quarter onto its defence: the
// tank build fielding ballast dies in nearly every battle the same build
// fielding endurance survives. That is why ballast is the attacking build's
// trait and not the tank's, and why the two builds below do not use it.
func TestBallastIsATradeAndNotAGift(t *testing.T) {
	traits, statuses := mustPassives(t), mustStatuses(t)
	held, err := traits.Lookup("ballast")
	if err != nil {
		t.Fatalf("look up ballast: %v", err)
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
	if !raised[modifier.Defense] {
		t.Error("ballast raises no defence, so it is a cost with nothing bought")
	}
	if !lowered[modifier.Speed] {
		t.Error("ballast lowers no speed, so it is a gift rather than a trade: " +
			"the defence is supposed to be paid for out of the turn order")
	}
	for target := range raised {
		if lowered[target] {
			t.Errorf("ballast both raises and lowers %s", target)
		}
	}
	// The reply is the other half of what was asked of the trait -- defence and
	// counter-damage out of one slot -- and it has to come off defence, because
	// a reply priced off attack would be a tank trait paid in the stat a tank
	// does not have.
	if held.Replies == nil || held.Replies.Power == 0 {
		t.Fatal("ballast answers nobody, so it buys defence and nothing else")
	}
	if held.Replies.Scaling.Stat != progression.Defense {
		t.Errorf("ballast answers off %s, which is the stat it does not raise",
			held.Replies.Scaling.Stat)
	}
}

// TestWideGuardShieldsTheAllyAndNotItsCaster is the taunt's lesson pointed the
// other way.
//
// A taunt sits on the unit that casts it; a guard must not. Both are authored as
// one line in the same file and the difference between them is which of two
// fields the status goes in, so an author who reaches for the wrong one writes a
// skill that reads correctly, describes itself correctly, and shields the wrong
// unit. Nothing but watching where the stack lands can tell them apart.
//
// Played by hand rather than left to autopilot, and that is not a convenience:
// battle.Suggest takes a skill of no power only when it can find nothing at all
// to hit, so a guard would essentially never be cast in a battle that runs
// itself. The mechanic is a player's.
func TestWideGuardShieldsTheAllyAndNotItsCaster(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.squirtle")
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, "pokemon.charmander")
	guardSlot := hex.Offset{Col: hex.FormationCols - 1, Row: hex.Rows / 2}
	wardSlot := hex.Offset{Col: hex.FormationCols - 1, Row: 0}
	fight, err := battle.New(books, 1, []battle.Roster{
		{ID: "guard", Side: hex.SideAlly, Slot: guardSlot, Affinity: affinity, Stats: stats,
			Skills: tankBuild, Passives: []string{"endurance"}},
		{ID: "ward", Side: hex.SideAlly, Slot: wardSlot, Affinity: affinity, Stats: stats,
			Skills: semiBuild, Passives: []string{"endurance"}},
		{ID: "them", Side: hex.SideEnemy, Slot: guardSlot, Affinity: theirAffinity,
			Stats: theirStats, Skills: theirKit, Passives: theirTraits},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()

	shielded := ""
	for turn := 0; turn < 400 && shielded == ""; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if !prompt.Skipped {
			switch {
			case prompt.Unit == "guard" && guardable(prompt, wardSlot):
				if err := fight.Act("wide_guard", wardSlot); err != nil {
					t.Fatalf("cast wide_guard at the ally: %v", err)
				}
			default:
				choice, ok := fight.Suggest(prompt)
				if !ok {
					if err := fight.Pass("waiting"); err != nil {
						t.Fatalf("pass: %v", err)
					}
					break
				}
				if err := fight.Act(choice.Skill, choice.Aim); err != nil {
					t.Fatalf("act %s: %v", choice.Skill, err)
				}
			}
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.StatusApplied && event.Skill == "wide_guard" {
				shielded = event.Target
			}
		}
	}
	switch shielded {
	case "":
		t.Fatal("wide_guard never landed its shield, so nothing was watched")
	case "guard":
		t.Error("wide_guard shielded the unit that cast it: the block belongs in applies, " +
			"where it lands on whoever the skill aims at, and not in self_applies")
	case "ward":
	default:
		t.Errorf("wide_guard shielded %q", shielded)
	}
}

// TestTheTwoSquirtleBuildsAreDifferentUnits is the claim the whole change is
// for: one learnset, two kits, and the two are not the same unit playing at
// different speeds.
//
// They cannot be compared by fighting each other. The tank build carries no
// skill with any power at all, so it cannot finish a battle and a win rate
// between them would only ever read a hundred to nothing — which says which one
// wins a duel and nothing about whether either is worth building. What separates
// them is the shape of what they do: how long one stands, and how fast the other
// works.
//
// ⚠️ Both figures are taken on autopilot and both are understatements, for the
// same reason. battle.Suggest attacks whenever it can and reaches for a skill of
// no power only when it cannot, so the tank build's whole kit is used only
// because it has nothing to attack with, and the semi-tank build hardly ever
// stops to guard. A player using either deliberately gets more out of it than
// this measures.
func TestTheTwoSquirtleBuildsAreDifferentUnits(t *testing.T) {
	tankTurns, tankDealt := squirtleFight(t, tankBuild, "thorns")
	semiTurns, semiDealt := squirtleFight(t, semiBuild, "ballast")

	if tankTurns <= semiTurns {
		t.Errorf("the tank build lasted %d turns and the semi-tank build %d, so the one built "+
			"to stand is not standing longer", tankTurns, semiTurns)
	}
	// Per turn rather than in total: the tank build is on the board so much
	// longer that its total creeps up on the other's, which would read as two
	// builds doing the same job slowly and quickly.
	tankRate, semiRate := tankDealt/int64(tankTurns), semiDealt/int64(semiTurns)
	if semiRate <= tankRate*3 {
		t.Errorf("the semi-tank build deals %d a turn and the tank build %d, so the one built "+
			"to hurt is not hurting appreciably faster", semiRate, tankRate)
	}
	// The tank build is not meant to deal nothing. Its damage is its trait
	// answering whoever hits it, which is the only source it has, and a build
	// whose only source stops working is a punching bag with no punch left.
	if tankDealt == 0 {
		t.Error("the tank build dealt nothing at all: its reply is the whole of its offence")
	}
	t.Logf("tank %d turns dealing %d (%d a turn), semi %d turns dealing %d (%d a turn)",
		tankTurns, tankDealt, tankRate, semiTurns, semiDealt, semiRate)
}

// guardable reports whether wide_guard is offered at the ally's cell this turn,
// which it is not while the skill is recharging.
func guardable(prompt *battle.Prompt, at hex.Offset) bool {
	for _, option := range prompt.Options {
		if option.Skill != "wide_guard" || !option.Available() {
			continue
		}
		for _, aim := range option.Aims {
			if aim == at {
				return true
			}
		}
	}
	return false
}

// squirtleDamage is what one skill deals over the sweep, fought alone so the
// figure is that skill's and nothing else's.
func squirtleDamage(t *testing.T, only string, stats progression.Values) int64 {
	t.Helper()
	total := int64(0)
	for seedValue := 1; seedValue <= squirtleSeeds; seedValue++ {
		dealt, _ := squirtleRun(t, []string{only}, "endurance", stats, uint64(seedValue))
		total += dealt
	}
	return total
}

// squirtleFight is a build's average battle: how long it lasted and how much it
// dealt, over the sweep.
func squirtleFight(t *testing.T, kit []string, trait string) (turns int, dealt int64) {
	t.Helper()
	stats, _, _, _ := fielded(t, "pokemon.squirtle")
	total := int64(0)
	lasted := 0
	for seedValue := 1; seedValue <= squirtleSeeds; seedValue++ {
		hurt, ran := squirtleRun(t, kit, trait, stats, uint64(seedValue))
		total += hurt
		lasted += ran
	}
	return lasted / squirtleSeeds, total / squirtleSeeds
}

// squirtleRun fights one Squirtle build against the shipped Charizard and reports
// what it dealt and how long the battle ran.
//
// Charizard rather than the whole cast, because one opponent held still is what
// makes two builds comparable: the question is what the kits do, and rotating
// the other side would answer a different one.
func squirtleRun(t *testing.T, kit []string, trait string,
	stats progression.Values, seedValue uint64) (dealt int64, turns int) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	_, affinity, _, _ := fielded(t, "pokemon.squirtle")
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
		if event.Kind == battle.Damaged && event.Actor == "mine" {
			dealt += event.Amount
		}
	}
	return dealt, turns
}
