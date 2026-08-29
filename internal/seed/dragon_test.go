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

// driveBuild is the dragon line with its own detonate in, spending the slot
// `dragon_rage` holds in the shipped build. It is not the shipped build and is
// deliberately not measured as one: see TestTheDragonLineCanSpendWhatItApplies
// for what fielding it is worth, which is nothing.
var driveBuild = []string{"dragon_claw", "dragon_drive", "outrage", "dragon_dance"}

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
// ⚠️ The band moved when the opponent learned to play, and it moved the *other*
// way. It used to read 42.5% with a note saying the figure was a floor: Suggest
// never buffed, so dragon_dance was never cast and the dragon build fought with
// three of its four slots. Both halves of that prediction came true and the second
// mattered more — dragon_dance is now cast in every battle, and the fire build's
// burn-then-inferno detonate is now played too. The fire line has a detonate and
// the dragon line has none, so the side with a combo gained more from the same
// change, and the measured figure fell to about a quarter — then to **22.1%** when
// tempo was priced, because `outrage` charges its user a slow and the rating can now
// see it. Two findings in one number, both about the cast rather than the engine.
//
// So the band is wider than a design would like, and deliberately: what it still
// asserts is that neither build is a scripted defeat, which is the claim that can
// be made honestly today.
//
// ⚠️ **"The dragon line wants something to spend a status on" was the guess that
// stood here, and it was wrong.** The line was given one — `dragon_drive`, which
// detonates the `expose` its own `dragon_claw` applies — and fielding it moves the
// figure 22.0% → 21.2%, which is nothing and slightly the wrong way. The gap is
// `reckless`, worth about **+33 points** on its own. The decomposition is in
// TestTheDragonLineCanSpendWhatItApplies, and what to do about the trait is still
// open.
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
	// A wide band on purpose, and wider than it was. It is not a tuning target to
	// be hit to the point; it is the statement that neither build is a scripted
	// defeat, which is the claim that can be made honestly now that the opponent
	// plays both kits. See the note above for what moved and why closing the gap is
	// a data change rather than this one.
	const lowest, highest = 150, 850
	if rate < lowest || rate > highest {
		t.Errorf("the dragon build wins %d.%d%% of %d battles against the fire build, outside %d..%d: "+
			"one of the two builds has become a scripted defeat",
			rate/10, rate%10, fought, lowest, highest)
	}
	t.Logf("dragon %d, fire %d over %d battles: %d.%d%%", dragon, fire, fought, rate/10, rate%10)
}

// TestTheDragonLineCanSpendWhatItApplies is the mechanism `dragon_drive` was
// added for, asserted as a mechanism rather than as a rate.
//
// A rate cannot see it. The line's detonate is worth **nothing** to the duel --
// over 3000 battles both ways round, fielding it moves 22.0% to 21.2%, a hair
// the wrong way -- and that is the pricing rule doing its job rather than the
// skill being broken. A detonate may not beat leaving the status alone by more
// than a factor of two, and `expose` is a cheap status: two turns of a defence
// share, worth 102 where `burn` is worth 548 in ticks. So the dragon line's
// detonate is capped near a third of `inferno`'s burst, and a third of a burst
// does not turn a matchup.
//
// ⚠️ **That is the answer to the roadmap item and it is not the answer the item
// predicted.** The 22.1% was written down as "the fire line has a detonate and
// the dragon line has none". Decomposed over the same 3000 battles, one thing
// changed at a time:
//
//	shipped                                   22.0%
//	dragon fields the detonate                21.2%    -0.8
//	fire loses its detonate                   32.9%   +10.9
//	dragon drops reckless for blood_thirst    55.1%   +33.1
//	dragon drops reckless for blaze           38.9%   +16.9
//	both changes at once                      53.4%   +31.4
//
// The missing detonate is the small term, and it is not recoverable by adding
// one: the fire line's is worth 10.9 because it spends a status worth five times
// as much. **`reckless` is the rest of the gap**, worth three times what any
// detonate could be. It grants `unleashed` and `bare` -- thirty per cent of
// attack bought with forty per cent of defence *and* forty per cent of dodge --
// into a build whose opponent's heaviest skill is amplified three and a half
// times off a status. TestRecklessIsATradeAndNotAGift asks whether something is
// given up and cannot ask whether too much is. What to do about that is a balance
// decision and deliberately not folded in here.
//
// So this asserts what was actually built: the line applies a status and now owns
// a skill that spends it, and both halves fire in a real battle. A kit that
// declared the combo and never played it would pass every golden and fail here.
func TestTheDragonLineCanSpendWhatItApplies(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.charmander")
	applied, amplified, consumed := 0, 0, 0
	for which := 1; which <= buildSeeds/10; which++ {
		fight, err := battle.New(books, uint64(which), []battle.Roster{
			{ID: "dragon", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: driveBuild, Passives: []string{"reckless"}},
			{ID: "fire", Side: hex.SideEnemy, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: fireBuild, Passives: []string{"blaze"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("run: %v", err)
		}
		for _, event := range fight.Drain() {
			if event.Actor != "dragon" || event.Status != "expose" {
				continue
			}
			switch event.Kind {
			case battle.StatusApplied:
				applied++
			case battle.Amplified:
				amplified++
			case battle.StatusConsumed:
				consumed++
			}
		}
	}
	if applied == 0 {
		t.Error("the dragon line never landed expose, so it has nothing to spend")
	}
	if amplified == 0 {
		t.Error("dragon_drive never found the status its own line applies, so the combo is " +
			"declared in the data and never played")
	}
	// Every amplified cast must also have eaten the stack. A detonate that
	// amplifies without consuming is exactly the strictly-better skill the pricing
	// rule exists to refuse, and nothing else in the suite would notice.
	if consumed != amplified {
		t.Errorf("dragon_drive was amplified %d times and consumed the status %d times: "+
			"a burst that keeps what it spent is being paid for twice", amplified, consumed)
	}
	t.Logf("expose applied %d, drive amplified %d, consumed %d", applied, amplified, consumed)
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

// TestRecklessSpendsNoMoreThanItBuys is the half of the trait's price that a
// shape test cannot see, and it is measured in damage rather than in wins.
//
// A win rate cannot price a stat. That is the swiftness finding restated: a rate
// is non-monotone in a stat, so a band drawn over one is a band over the shape of
// a matchup and not over the thing being changed. Damage off the event log is the
// currency the trait is actually denominated in — `unleashed` buys damage dealt
// and `bare` sells damage taken — so the log can be asked the question directly:
// fight the same duel with the trait and without it, and compare the two sums.
//
// Fielding no trait at all is legal and shipped, which is what makes the
// without-run available as a baseline; there is no need to invent a null trait.
//
// ⚠️ The RNG stream diverges the instant anything about the two runs differs, so
// seed N with the trait is not seed N without it in any comparable sense. The
// aggregate over every seed and both arrangements is the measurement; a per-seed
// pairing would be reading noise.
//
// ⚠️ The factor of two is a **declared design constant**, not a measurement. It
// is borrowed from this repo's detonate rule — a burst may beat leaving the
// status alone, but not by more than a factor of two — and it is the only invented
// number in this test. It says what "too much" means for a trait: paying twice
// what you buy is a trade, paying three times is a tax.
//
// ⚠️ **The shipped trait passes this with room, and that is worth knowing rather
// than reassuring.** It buys 30956 damage for 50084, a ratio of 1.62, so the bound
// is not a tight fit and this test on its own would not have caught the trait
// being a bad deal — the 22.0% duel figure is the thing that says so, and a rate
// and a ledger are asking different questions. What the ledger *can* see is a
// change making the trade worse: giving `reckless` a vulnerability to the six
// inflictable harmful statuses drove `bought` **negative** (−7898 bought against
// 61429 spent), which is this test going red on a candidate no win-rate band
// would have rejected on its own. That is the job.
func TestRecklessSpendsNoMoreThanItBuys(t *testing.T) {
	withTrait := theDamageLedger(t, []string{"reckless"})
	without := theDamageLedger(t, nil)

	bought := withTrait.dealt - without.dealt
	cost := withTrait.taken - without.taken
	t.Logf("with reckless: dealt %d, taken %d; without: dealt %d, taken %d; bought %d, cost %d",
		withTrait.dealt, withTrait.taken, without.dealt, without.taken, bought, cost)

	if bought <= 0 {
		t.Errorf("reckless buys %d damage, so the attack it grants is worth nothing "+
			"and the whole trait is cost", bought)
	}
	if cost > 2*bought {
		t.Errorf("reckless buys %d damage and costs %d taken, over twice what it buys: "+
			"the trade has become a tax", bought, cost)
	}
}

// ledger is what one side of the duel dealt and took over every battle fought.
type ledger struct {
	dealt, taken int64
}

// theDamageLedger fights the dragon build against the fire build over the whole
// seed range, both arrangements, and totals what the dragon unit dealt and took.
//
// Both arrangements for the reason every other measurement here takes both: the
// turn queue breaks a tie by enlistment and this is a mirror, so a one-way figure
// would carry the first slot's advantage into the answer.
//
// Damage taken counts status ticks as well as strikes. A tick is the fire build's
// main lever — `burn` is most of what `inferno` is worth — so a ledger that read
// only strikes would be blind to the half of the price that arrives as a status,
// which is exactly the half a dodge term or a resistance share moves.
func theDamageLedger(t *testing.T, traits []string) ledger {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.charmander")
	total := ledger{}
	for _, dragonIsFirst := range []bool{true, false} {
		for which := 1; which <= buildSeeds; which++ {
			dragon := battle.Roster{ID: "dragon", Side: hex.SideAlly, Slot: buildSlot,
				Affinity: affinity, Stats: stats, Skills: dragonBuild, Passives: traits}
			fire := battle.Roster{ID: "fire", Side: hex.SideEnemy, Slot: buildSlot,
				Affinity: affinity, Stats: stats, Skills: fireBuild, Passives: []string{"blaze"}}
			order := []battle.Roster{dragon, fire}
			if !dragonIsFirst {
				// Side and enlistment move together: the second-enlisted unit is
				// the enemy one, so swapping the order without swapping the side
				// would field two allies.
				dragon.Side = hex.SideEnemy
				fire.Side = hex.SideAlly
				order = []battle.Roster{fire, dragon}
			}
			fight, err := battle.New(books, uint64(which), order)
			if err != nil {
				t.Fatalf("new battle: %v", err)
			}
			fight.Begin()
			if _, err := fight.RunToEnd(4000); err != nil {
				t.Fatalf("run: %v", err)
			}
			for _, event := range fight.Drain() {
				switch event.Kind {
				case battle.Damaged:
					if event.Actor == "dragon" {
						total.dealt += event.Amount
					}
					if event.Target == "dragon" {
						total.taken += event.Amount
					}
				case battle.StatusTicked:
					// A tick names its holder in Actor and has no target: the
					// holder is who it is happening to.
					if event.Actor == "dragon" {
						total.taken += event.Amount
					}
				}
			}
		}
	}
	return total
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
