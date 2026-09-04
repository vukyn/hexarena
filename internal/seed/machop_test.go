package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// chargeBuild is Machop's third direction, and it is the design record for this
// file the way gambleBuild and sureBuild are for theirs.
//
// ⚠️ It is also the FIELDED four, which the other two are not: forge.Spar and
// forge.Weigh both take a carrier's first four learnset entries through seedKit,
// so a kit sitting further down the list cannot be sparred or weighed at all.
// That is why the three new skills open the learnset — see the comment on
// chargeKit below for what it displaced.
var chargeBuild = []string{"brace", "wrecking_swing", "cross_chop", "rock_throw"}

// The three ids, named once so a rename cannot leave half this file measuring a
// skill that no longer exists.
const (
	heftStatus  = "heft"
	builderID   = "brace"
	deepID      = "wrecking_swing"
	chargeKitID = "machop.charge"
)

// chargeOpponents is who the loop is driven against, and the spread is the point.
//
// A charge kit's whole bargain is turns, so the one thing a single opponent
// cannot show is that the bargain survives being rushed. Riolu is the fastest
// thing in the cast and Bulbasaur kills with a status that runs while Machop is
// standing still winding up — measured, those are the two Machop leaves the tank
// unspent against most often — while Squirtle and Gastly are the long games the
// kit is built for. A threshold read off the friendly half alone would be a
// fixture hiding a branch.
var chargeOpponents = []string{
	"pokemon.squirtle", "pokemon.gastly", "pokemon.riolu", "pokemon.bulbasaur",
}

// chargeKitSeeds is how many duels each opponent is fought over. Small on purpose:
// what is read here is whether a mechanism fires and how often, never a rate.
const chargeKitSeeds = 40

// chargeCounts is what one run of the kit came to, counted off the event log.
type chargeCounts struct {
	battles  int
	casts    map[string]int
	consumed map[int]int
	dry      int
}

// driveTheChargeKit fights Machop's charge loadout against each opponent and
// counts what it cast.
//
// ⚠️ Counts, and never a winner. A win rate cannot see this design at all: three
// of the four fielded skills stop being plain attacks, so the rate moves by
// several points whatever the loop does, and it moves by about as much when the
// loop is dead as when it closes. The mutation named on every threshold below is
// chosen for exactly that reason — each one leaves the rate roughly where it was.
func driveTheChargeKit(t *testing.T, kit []string) chargeCounts {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, traits := fielded(t, "pokemon.machop")
	counted := chargeCounts{casts: map[string]int{}, consumed: map[int]int{}}
	for _, id := range chargeOpponents {
		theirStats, theirAffinity, theirKit, theirTraits := fielded(t, id)
		for value := 1; value <= chargeKitSeeds; value++ {
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
			if _, err := fight.RunToEnd(4000); err != nil {
				t.Fatalf("seed %d against %s: %v", value, id, err)
			}
			counted.battles++
			spent := 0
			for _, event := range fight.Drain() {
				if event.Actor != "mine" {
					continue
				}
				switch event.Kind {
				case battle.SkillUsed:
					counted.casts[event.Skill]++
				case battle.StatusConsumed:
					if event.Status == heftStatus {
						counted.consumed[event.Stacks]++
						spent++
					}
				}
			}
			if spent == 0 {
				counted.dry++
			}
		}
	}
	return counted
}

// TestTheShippedChargeLoopClosesOnARealBattle is the half a parser cannot reach:
// the kit is legal, the numbers are in range, the gloss is written, and none of
// that says the rating ever decides that standing still is worth a turn.
//
// It is the machop twin of TestTheShippedFireLoopBanksAndSpendsInARealBattle, and
// what it adds is the gate. A reserve that only amplifies is cast either way, so
// that test could read a spend and still be reading a skill nobody chose to fuel;
// a GATED spender is not on offer at all without its fuel, so every consume
// counted here is a turn the rating spent charging on purpose, at least three
// turns before it got anything back.
func TestTheShippedChargeLoopClosesOnARealBattle(t *testing.T) {
	counted := driveTheChargeKit(t, chargeBuild)
	perBattle := func(id string) float64 {
		return float64(counted.casts[id]) / float64(counted.battles)
	}
	t.Logf("over %d duels: %s %.2f, %s %.2f, cross_chop %.2f per battle; "+
		"depths spent %v; %d duel(s) ended with nothing spent",
		counted.battles, builderID, perBattle(builderID),
		deepID, perBattle(deepID), perBattle("cross_chop"), counted.consumed, counted.dry)

	// ⚠️ **This is a floor and not a price, and the reason is cross_chop's
	// cooldown.** Three of the four fielded skills cost nothing to cast, so on
	// roughly two turns in three the filler is unavailable and `brace` is the only
	// thing on offer — Suggest takes it as a fallback whether or not a stack of
	// fuel is worth anything. Measured: with pricing.selfSpendable's gated arm
	// deleted the count only falls from 5.77 to 4.67, so this threshold alone does
	// NOT hold that arm up.
	//
	// Mutation, and it takes two edits because of the sentence above: delete the
	// `if spends.GatesCast()` arm in pricing.selfSpendable AND set cross_chop's
	// cooldown to nought so the rating always has something else to do. Measured
	// 3.81 a battle with the arm and 1.51 without it, 40 dry duels against 120.
	// What the threshold catches on its own is the coarser failure — a builder
	// nothing ever reaches for — and the spend counts below are what hold the
	// pricing.
	if charges := perBattle(builderID); charges < 3 {
		t.Errorf("%s was cast %.2f times a battle, want at least 3: the rating would rather "+
			"do anything than stand still, so the fuel is worth nothing and both gates are shut",
			builderID, charges)
	}
	// And the other half. Banking without cashing is the failure the fire loop
	// names — a generator whose consumer nobody reaches — and here it has a
	// sharper edge, because a gated spender that is never cast is a skill that was
	// never even OFFERED.
	//
	// ⚠️ Not `min_stacks` above the cap of five: skill.resolve refuses that at
	// parse ("needs 6 stacks of heft, which caps at 5"), so the books never load
	// and this measures nothing.
	//
	// Mutation: set wrecking_swing's power to 1000. Measured, spends fall from
	// 0.45 a battle to 0.03 and the dry duels rise from 93 to 155 — five turns of
	// standing still stop being worth the blow and the rating keeps the filler.
	// ⚠️ **cross_chop's own 2800 does NOT move this**: measured 0.47 a battle,
	// because at five stacks the gated spender is the only thing on offer and is
	// still the biggest single blow on the unit. So what this line holds is that
	// the loop CLOSES, not what the blow is worth — the 3v3 figures above are the
	// only thing that prices the power, and a mutation on the power alone is a
	// weak instrument here on purpose.
	//
	// ⚠️ **The floor is 0.4 and not 1, and that is a fact about the BOARD rather
	// than about the kit.** A duel gives a five-stack rung five turns of standing
	// still against a single opponent already swinging, and measured here it lands
	// in fewer than half of them: 0.45 a battle over 160 duels, 93 of which ended
	// with nothing spent. That is not the loop failing — it is the reason the kit
	// exists. On a 3v3 with two allies covering, the same skill at the same power
	// reads **50.1%** against the identical squad carrying plain machop, with the
	// mirror control at exactly 50.0%. A duel is the wrong board for a kit whose
	// whole bargain is that somebody else holds the line while it charges, and
	// hexforge weigh says so in its own words: it refuses this skill outright with
	// "cast 0 time(s)", because a machop mirror is decided in 11 turns and moving
	// first there is worth 88%.
	//
	// So what this floor holds is the coarse claim — the rating does reach the
	// gate and does cash it — and the fine one is held by the 3v3 figures above.
	spends := float64(counted.consumed[5]) / float64(counted.battles)
	if spends < 0.4 {
		t.Errorf("the tank was cashed %.2f times a battle, want at least 0.40: every stack banked "+
			"expired unspent, which is a turn thrown away rather than a turn invested", spends)
	}
	// ONE rung, and that is a decision rather than an omission. A three-stack
	// rung was authored, measured and dropped: on a 3v3 the kit carrying both read
	// 40.3% against the same squad's plain machop, while either rung alone read
	// 51.2% and 50.1%. The rating fires the shallow rung the moment its gate opens
	// and so never banks the fifth stack — measured, the deep rung was cast NOT
	// ONCE over five battles beside a three-stack rung, and four times over five
	// once it was alone. A cheap gate on the same fuel starves an expensive one,
	// and the kit that brings both is ten points worse than the kit that brings
	// either.
	//
	// Mutation: author the shallow rung back onto the learnset ahead of this one.
	// The count below goes to nought, which is the whole finding in one line.
	if counted.consumed[5] == 0 {
		t.Errorf("%s spent its fuel not once over %d duels, so the gate is shut and the builder is a turn thrown away",
			deepID, counted.battles)
	}
	// A consume reads the count its skill names and never anything else. This is
	// the assertion that would catch consume_stacks going back to meaning "all of
	// them": a spend of two or of four is arithmetically fine and silently makes
	// the two rungs one.
	//
	// ⚠️ Not consume_stacks 0: measured, the rating simply stops casting a spender
	// that empties the tank, so the map loses a key rather than gaining a wrong
	// one and the two thresholds above catch it first.
	//
	// Mutation: set wrecking_swing's consume_stacks to 2, which is legal, parses,
	// and leaves change behind. The map gains a key of 2 and this is the line that
	// names why that is wrong.
	for depth := range counted.consumed {
		if depth != 5 {
			t.Errorf("a spend took %d stacks, and the one shipped rung takes 5: %v",
				depth, counted.consumed)
		}
	}
}

// TestTheChargeKitIsWhatTheCatalogueOffers joins the kit measured above to the
// build a player reads, which is the same join TestTheShippedBuildsAreTheOnesTheTestsMeasure
// makes for the other twenty-three.
//
// It is here rather than there because this one has a second claim: the build and
// the LEARNSET have to agree as well. Spar and Weigh both field the first four
// entries, so a build naming four skills that are not those four measures a
// different unit from the one every figure in this file was taken on.
func TestTheChargeKitIsWhatTheCatalogueOffers(t *testing.T) {
	builds, err := seed.Builds()
	if err != nil {
		t.Fatalf("load the builds: %v", err)
	}
	build, known := builds.Get(chargeKitID)
	if !known {
		t.Fatalf("the catalogue holds no build %q", chargeKitID)
	}
	if len(build.Skills) != len(chargeBuild) {
		t.Fatalf("the build %q holds %d skills and the suite measures %d",
			chargeKitID, len(build.Skills), len(chargeBuild))
	}
	for at, id := range chargeBuild {
		if build.Skills[at] != id {
			t.Errorf("the build %q names %q in slot %d and the suite measures %q",
				chargeKitID, build.Skills[at], at, id)
		}
	}
	_, _, kit, _ := fielded(t, "pokemon.machop")
	for at, id := range chargeBuild {
		if at >= len(kit) {
			t.Fatalf("Machop fields %d skills, and the charge kit needs %d", len(kit), len(chargeBuild))
		}
		if kit[at] != id {
			t.Errorf("Machop fields %q in slot %d where the charge kit wants %q: seedKit takes "+
				"the first four the learnset declares, so neither spar nor weigh would ever see this loop",
				kit[at], at, id)
		}
	}
}

// TestTheGatedRungsAreNotOnTheFlatLadder is the arm #286 added to
// TestTheShippedSpendersCoverTheLadder, checked against real data rather than
// against the fixture it was written on.
//
// The ladder counts flat-bonus spenders by the number of stacks they take, and
// refuses two skills that take the same count, because a rung written twice is
// one rung. A gated spender takes a count too — five, in the deep rung's case,
// which is exactly what thorn_volley takes — but it buys the CAST rather than a
// bonus, so its depth is not a rung on that ladder at all and counting it there
// would refuse a pair that is not a pair.
//
// Mutation: delete the `if spends.GatesCast()` arm in
// TestTheShippedSpendersCoverTheLadder. wrecking_swing and thorn_volley then
// collide at five and that test goes red — which is the collision this one exists
// to prove is real rather than hypothetical.
func TestTheGatedRungsAreNotOnTheFlatLadder(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skills: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	// What every flat rung in the book takes, gated ones left out.
	flat := map[int]string{}
	gated := map[string]int{}
	for _, declared := range skills.Skills() {
		spends := declared.SelfRequires
		if spends == nil || !spends.Consume {
			continue
		}
		kind, err := statuses.Lookup(spends.Status)
		if err != nil || kind.Category != status.Reserve {
			continue
		}
		if spends.Scales() || spends.ScalesRestore() {
			continue
		}
		if spends.GatesCast() {
			gated[declared.ID] = spends.ConsumeStacks
			continue
		}
		flat[spends.ConsumeStacks] = declared.ID
	}
	if len(gated) != 1 {
		t.Fatalf("the book ships %d gated spender(s) and this file measures one: %v", len(gated), gated)
	}
	if _, counted := gated[deepID]; !counted {
		t.Errorf("%s does not read as a gated spender, so its depth would be counted as a rung on the flat ladder", deepID)
	}
	// The collision is real, not theoretical: something in the book already spends
	// this many stacks flat. If that ever stops being true the arm still has to
	// exist, but this test would stop proving why, so say so rather than pass.
	//
	// Mutation: change thorn_volley's consume_stacks. This logs a miss instead of
	// the name, and the reader is told the guard is now untested by the data.
	collided := 0
	for id, depth := range gated {
		if other, clash := flat[depth]; clash {
			collided++
			t.Logf("%s is gated at %d stacks, which %s already spends flat", id, depth, other)
		}
	}
	if collided == 0 {
		t.Errorf("no gated rung shares a depth with a flat one (%v against %v), so nothing in the "+
			"shipped book now shows why the ladder counts them apart", gated, flat)
	}
}

// TestNobodyButItsHolderFuelsTheGroundReserve is the rule that separates this
// reserve from the three beside it, and it is a rule about the whole BOOK rather
// than about these three skills.
//
// swelter, verdure and moisture may all be handed over: overgrow and drench are
// ally-aimed grants, and a second unit spending its turn to fill somebody else's
// tank is a perfectly good way to play them. This one is the unit's own bargain —
// it stands still, and standing still is the whole cost — so a teammate that
// could pay that cost for it would delete the design while leaving every skill in
// it untouched. Nothing would fail: it would parse, gloss, ship and read as a
// better version of the same kit.
//
// So the ban is on `applies` at ANY target, self included, rather than on
// ally-aimed grants: `self_applies` is a promise the caster makes about itself,
// `applies` is a payload that travels, and the second one is what would have to
// be written for a teammate to fuel this.
//
// Mutation: give any shipped skill an `applies` entry naming heft, at any target.
// This goes red and nothing else in the suite moves.
func TestNobodyButItsHolderFuelsTheGroundReserve(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skills: %v", err)
	}
	grants := 0
	for _, declared := range skills.Skills() {
		for _, application := range declared.Applies {
			if application.Status != heftStatus {
				continue
			}
			t.Errorf("%s puts %s on its target through `applies`, and %s is the one reserve "+
				"nobody may fill for its holder: the cost of this loop is the turns the holder "+
				"itself stands still, and a hand-over pays that cost with somebody else's turn",
				declared.ID, heftStatus, heftStatus)
		}
		for _, application := range declared.SelfApplies {
			if application.Status == heftStatus {
				grants++
			}
		}
	}
	// And the reserve has to be fillable at all, or the loop above is a rule about
	// a status nothing grants — which passes for the wrong reason.
	if grants == 0 {
		t.Errorf("nothing in the book grants %s to itself either, so the reserve has no generator "+
			"and this test was measuring an empty set", heftStatus)
	}
	t.Logf("%d skill(s) grant %s, all of them to their own caster", grants, heftStatus)
}

// TestTheTankSurvivesFiveTurnsOfChargingAndDecaysAfter is the duration decision,
// asserted rather than described.
//
// status.Apply refreshes Remaining on every stack already held before it appends
// the new one, so a unit charging on consecutive turns never loses a stack at any
// duration above one — which means the authored figure is not about how deep the
// tank gets, it is about how long an INTERRUPTION costs. Four is what was
// measured: the counts driveTheChargeKit reads are identical at four, five and
// six, and at three the shallow rung loses about an eighth of its casts.
//
// Both halves are asserted here because either alone passes for the wrong reason.
// A duration of one would fail the first; a permanent status would pass the first
// and fail the second.
//
// ⚠️ The decay half reads kind.Duration and would agree with the engine at any
// value, so the authored four is pinned by hand below. That hardcoding is the
// same device gambleBuild and its siblings use: a design decision a data edit may
// not quietly rewrite.
//
// Mutation: set heft's duration to 1 and the charging half goes red after two
// casts ("the tank holds 1"); set it to anything but 4 and the pinned figure
// does.
func TestTheTankSurvivesFiveTurnsOfChargingAndDecaysAfter(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	kind, err := books.Statuses.Lookup(heftStatus)
	if err != nil {
		t.Fatalf("look %s up: %v", heftStatus, err)
	}
	// The measured figure, written out. Four is the smallest duration at which the
	// loop counts are identical to five's and to six's — nothing that would have
	// been spent ever expires above it — while three costs the shallow rung about
	// an eighth of its casts (0.47 a battle against 0.54) and two costs the deep
	// rung a third (0.39 against 0.59).
	const authored = 4
	if kind.Duration != authored {
		t.Errorf("heft lasts %d turn(s) and the sweep that chose this design measured %d",
			kind.Duration, authored)
	}
	if kind.MaxStacks != 5 {
		t.Errorf("heft caps at %d stacks and the deep rung asks for 5", kind.MaxStacks)
	}
	stats, affinity, _, traits := fielded(t, "pokemon.machop")
	theirStats, theirAffinity, _, theirTraits := fielded(t, "pokemon.squirtle")
	fight, err := battle.New(books, 1, []battle.Roster{
		{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
			Skills: chargeBuild, Passives: traits},
		// A target that cannot reach back, so the reading is about the tank and
		// not about whether Machop lived long enough to take it.
		{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
			Stats: theirStats, Skills: []string{"withdraw"}, Passives: theirTraits},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	mine, known := fight.Unit("mine")
	if !known {
		t.Fatal("the charger is not on the board")
	}

	// Charging every turn, to the cap.
	charged := 0
	for charged < kind.MaxStacks {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			t.Fatalf("the battle ended after %d charge(s)", charged)
		}
		if prompt.Unit != "mine" {
			if _, chose := fight.Suggest(prompt); !chose {
				if err := fight.Pass("nothing to do"); err != nil {
					t.Fatalf("their pass: %v", err)
				}
				continue
			}
			choice, _ := fight.Suggest(prompt)
			if err := fight.Act(choice.Skill, choice.Aim); err != nil {
				t.Fatalf("their act %s: %v", choice.Skill, err)
			}
			continue
		}
		if err := fight.Act(builderID, mine.Cell); err != nil {
			t.Fatalf("charge %d: %v", charged+1, err)
		}
		charged++
		if held := mine.Statuses.Stacks(heftStatus); held != charged {
			t.Fatalf("after %d consecutive charge(s) the tank holds %d: a stack applied on "+
				"the turn before ran out while the holder was still filling it",
				charged, held)
		}
	}
	t.Logf("%d consecutive charges fill the tank to its cap of %d, at %d turn(s) left on every stack",
		charged, kind.MaxStacks, mine.Statuses.Remaining(heftStatus))

	// Then stopping. A full tank has Duration turns left on every stack, and Tick
	// spends one at the start of each of the holder's own turns — so the holder
	// gets Duration-1 turns of doing something else before the tank is gone, and
	// the Duration'th finds it empty.
	idle := 0
	for mine.Statuses.Stacks(heftStatus) > 0 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance while idling: %v", err)
		}
		if prompt == nil {
			t.Fatalf("the battle ended after %d idle turn(s) with the tank still holding", idle)
		}
		if prompt.Unit != "mine" {
			if err := fight.Pass("not the charger"); err != nil {
				t.Fatalf("their pass: %v", err)
			}
			continue
		}
		idle++
		if err := fight.Pass("standing off the loop"); err != nil {
			t.Fatalf("idle %d: %v", idle, err)
		}
	}
	if idle != kind.Duration {
		t.Errorf("the tank emptied on the holder's %d turn away from the builder, and heft is "+
			"authored to last %d: the number a reader takes off statuses.json is not the number "+
			"the board plays", idle, kind.Duration)
	}
	t.Logf("a full tank survives %d turn(s) away from the builder and is gone on the next", idle-1)
}

// TestTheGatedSpendersAreShapedTheWayTheDesignSays is the cheap half, and it is
// here so that the expensive halves above cannot pass while measuring a kit that
// quietly stopped being this one.
//
// Every field named here is one another test in this file depends on: the loop
// counts assume the builder is free to cast every turn, the depth assertion
// assumes an exact consume, and the gate is what makes a cooldownless spender a
// design rather than a skill cast every turn for its own figure.
func TestTheGatedSpendersAreShapedTheWayTheDesignSays(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skills: %v", err)
	}
	builder, err := skills.Lookup(builderID)
	if err != nil {
		t.Fatalf("look %s up: %v", builderID, err)
	}
	if builder.Cooldown != 0 {
		t.Errorf("%s waits %d turn(s) between casts, and a builder that cannot be cast every "+
			"turn cannot reach a rung of five before the tank decays under it",
			builderID, builder.Cooldown)
	}
	if builder.Target != skill.Self {
		t.Errorf("%s is aimed at %s: this reserve is the holder's own bargain", builderID, builder.Target)
	}
	for _, id := range []string{deepID} {
		declared, err := skills.Lookup(id)
		if err != nil {
			t.Fatalf("look %s up: %v", id, err)
		}
		spends := declared.SelfRequires
		if spends == nil || !spends.GatesCast() {
			t.Errorf("%s does not gate its cast, so a caster holding nothing casts it every "+
				"turn for its own figure and the whole loop is a suggestion", id)
			continue
		}
		if declared.Cooldown != 0 {
			t.Errorf("%s waits %d turn(s), and the gate is meant to be the only thing holding it back", id, declared.Cooldown)
		}
		if spends.ConsumeStacks != spends.MinStacks {
			t.Errorf("%s opens at %d stacks and takes %d: a rung that leaves change behind is a "+
				"different design from the one measured here",
				id, spends.MinStacks, spends.ConsumeStacks)
		}
		// Flat, and that is the design rather than an omission: the blow does not
		// grow with what the tank happened to hold, so the two rungs are two
		// decisions rather than one curve read at two points.
		if spends.Scales() || spends.BonusPower != 0 {
			t.Errorf("%s pays more for a deeper tank, and both rungs are authored flat", id)
		}
	}
}
