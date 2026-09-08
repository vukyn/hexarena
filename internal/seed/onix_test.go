package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The two directions, hardcoded here the way every other design record in this
// package is: these are the kits that were measured, and
// TestTheShippedBuildsAreTheOnesTheTestsMeasure is what stops the catalogue a
// player reads drifting away from them.
//
// They are the two halves of what this line is. One takes the fight onto itself
// and lets the armour answer; the other spends its turns breaking the other
// side's.
//
// ⚠️ **Every skill in forgeBuild is metal and every one of them is gated on the
// grown form**, so this is the first entry in the catalogue that is a fact about
// a FORM rather than about a character: the root form is ground and could not
// carry a single one of them.
var (
	wallBuild  = []string{"taunt", "wide_guard", "stone_edge", "dig"}
	forgeBuild = []string{"steel_beam", "flash_cannon", "metal_sound", "metal_claw"}
)

// monolithSeeds is how many duels each duel reading below is taken over. Sixty
// is what every other build reading in this package uses.
const monolithSeeds = 60

// wallReading is what one unit did over those duels, and every column is a
// TOTAL rather than a per-seed average.
//
// ⚠️ The same arithmetic that made the first Machop reading a row of noughts: a
// duel runs a couple of dozen turns, so a figure divided by sixty truncates to
// nothing and two units come back identical while one is doing several times as
// much of something as the other. Every ratio below divides one of these totals
// by another, all of them in the thousands, so none of them can truncate away.
type wallReading struct {
	turns    int
	dealt    int64
	landed   int
	replied  int64
	taken    int64
	received int
}

// eachDealt is what one landed blow came to, which is the arm rather than the
// battle.
func (r wallReading) eachDealt() int64 {
	if r.landed == 0 {
		return 0
	}
	return r.dealt / int64(r.landed)
}

// eachTaken is what one blow received came to, which is the armour. It is the
// figure an elemental multiplier moves directly, so it is also what the second
// element is read off below.
func (r wallReading) eachTaken() int64 {
	if r.received == 0 {
		return 0
	}
	return r.taken / int64(r.received)
}

// perBlowTaken is what the trait slot answered with for each blow that came in,
// which is the one axis a reply trait can be read on: a reply is produced by
// being hit, so a bare total is a reading of how often the other side swung.
func (r wallReading) perBlowTaken() int64 {
	if r.received == 0 {
		return 0
	}
	return r.replied / int64(r.received)
}

// dealtPerHundredTurns is how much of the other unit a kit spent its turns on.
// A share of the battle rather than a count, because the two duels being
// compared do not run for the same number of turns — a bare total is partly a
// reading of how long the fight went.
func (r wallReading) dealtPerHundredTurns() int64 {
	if r.turns == 0 {
		return 0
	}
	return r.dealt * 100 / int64(r.turns)
}

// landedPerHundredTurns is the same share for the count rather than the size,
// which is the closest a duel gets to reading a speed.
func (r wallReading) landedPerHundredTurns() int {
	if r.turns == 0 {
		return 0
	}
	return r.landed * 100 / r.turns
}

// TestAMonolithEarnsItsSlotWhereASparCannotSeeIt is this character's case of the
// measurement the mender needed, taken in the slot the character is actually
// chosen for.
//
// The mender's half of the reason holds here unchanged: a duel is decided by who
// runs out of health first, and half of what this build brings is spent on a
// body that is not there — `taunt` aims a turn nobody else was going to spend at
// it, and `wide_guard` shields a neighbour a duel does not have.
// `hexforge spar --stage Steelix` reads the character at 264 per mille, which is
// a reading of the four *damaging* skills its learnset happens to declare first
// and says nothing about the kit anybody would field.
//
// ⚠️ **The shell swaps the WALL, not the third member, and that is the whole
// difference between a reading and a category error.** `aSquadOf` holds a
// striker and a wall constant and varies the flex slot, which is right for a
// mender and wrong for a second wall: putting one there makes the home squad two
// walls and a striker against two strikers and a wall, so what comes back prices
// the composition rather than the unit. Measured, with the wall build in that
// slot: **381 per mille against a slugger and 360 against a bruiser**, both under
// the mender's floor, with the character behaving exactly as designed. The
// question a wall is actually asked is whether it is worth the wall slot, so the
// striker and the flex member are held and the wall is what changes.
//
// The floor is the mender's own 450 and means the same thing here — no worse than
// five points down on the incumbent is a real alternative rather than a worse
// copy of one. What holds the shell honest is the control: the warden's own squad
// against a copy of itself has to come to exactly 500, because fightSquads runs
// every seed from both slots and a shell answering anything else would be
// reporting its own bias as the unit's.
func TestAMonolithEarnsItsSlotWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	warden := aSquadWalledBy("warden", aWall("pokemon.squirtle", "endurance",
		"water_gun", "bubble", "bite", "withdraw"))

	// The control first: a shell that is not even cannot price anything put in
	// it, and this is the one arrangement whose answer is known in advance.
	wins, losses, endless := fightSquads(t, books, characters,
		aSquadWalledBy("mirror", aWall("pokemon.squirtle", "endurance",
			"water_gun", "bubble", "bite", "withdraw")), warden)
	refuseTooManyStalls(t, "the mirror", endless, menderSeeds*2)
	if decided := wins + losses; decided == 0 || wins*1000/decided != 500 {
		t.Fatalf("the warden's squad against a copy of itself reads %d-%d: a shell that is not "+
			"even reports its own bias as whatever is put in it", wins, losses)
	}

	const floor = 450
	monolith := aSquadWalledBy("with-monolith", aWall("pokemon.onix", "thorns", wallBuild...))
	wins, losses, endless = fightSquads(t, books, characters, monolith, warden)
	refuseTooManyStalls(t, "the monolith", endless, menderSeeds*2)
	decided := wins + losses
	if decided == 0 {
		t.Fatal("no battle was decided, so there is no rate to read")
	}
	rate := wins * 1000 / decided
	t.Logf("the monolith in the wall slot against the warden: %d per mille (%d-%d), %d endless",
		rate, wins, losses, endless)
	if rate < floor {
		t.Errorf("the monolith reads %d per mille in the wall slot against the warden, under the "+
			"floor of %d: a slot the wall already in the cast holds better is a slot this one "+
			"should not be in", rate, floor)
	}
}

// aSquadWalledBy is aSquadOf with the variable slot moved: the striker and the
// flex member are held and the WALL is what differs.
//
// It is a second shell rather than an argument to the first because the two ask
// different questions, and a shell that could be either is a shell whose reading
// has to be read twice to know what it priced.
func aSquadWalledBy(id string, wall placement.Placement) placement.Squad {
	return placement.Squad{
		ID: id,
		Units: []placement.Placement{
			{ID: "fire", Character: "pokemon.charmander", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"flamethrower", "fire_spin", "ember", "inferno"},
				Passives: []string{"blaze"}},
			wall,
			{ID: "third", Character: "pokemon.machop", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 0, Row: 1},
				Skills:   []string{"rock_throw", "body_slam", "cross_chop", "vital_throw"},
				Passives: []string{"endurance"}},
		},
	}
}

// aWall is the slot the shell above varies, with the trait named rather than
// inherited: half of what a wall build is is which trait it spends its one slot
// on, so a helper that chose it would be answering the question.
func aWall(character, trait string, kit ...string) placement.Placement {
	return placement.Placement{
		ID: "wall", Character: character, Level: progression.LevelCap,
		Stage:    progression.Furthest,
		Slot:     hex.Offset{Col: 1, Row: 1},
		Skills:   kit,
		Passives: []string{trait},
	}
}

// TestTheTwoWallsAreTwoDifferentUnits is the claim that the cast now holds two
// walls rather than one wall under two names.
//
// A second character in a role the cast already fills has to be a second
// *answer*, and the way that goes wrong is the reskin: the same shape with the
// numbers nudged, which reads as a new unit on a list and plays as the one
// already there. So the three figures below are the three axes the design says
// the two differ on — the monolith is the harder armour with the heavier arm that
// gets to swing least often, the warden is the deeper health pool that hits
// softer and more often.
//
// ⚠️ **The opponent is Mew, and the choice is not a preference.** Two units are
// comparable only against something held still, and here that has to be true on
// the element chart as well as in the stat line: Blastoise is water and Steelix
// is ground/metal, so an ordinary opponent would have priced two different
// matchups and reported the difference as the units'. The inert element has no
// strength and no weakness in either direction, so Mew's blows land on both walls
// at the same multiplier and both walls' blows land on Mew at the same one —
// which leaves the stat lines, which is what is being read.
//
// All three figures are ratios rather than counts for the same reason: the two
// duels do not run for the same number of turns, so a bare count of anything is
// partly a reading of how long the fight went.
func TestTheTwoWallsAreTwoDifferentUnits(t *testing.T) {
	monolith := readWall(t, "pokemon.onix")
	warden := readWall(t, "pokemon.squirtle")

	for _, reading := range []struct {
		name string
		got  wallReading
	}{{"monolith", monolith}, {"warden", warden}} {
		t.Logf("%-8s %5d turns, %5d landed for %8d (%4d each), %5d received for %8d (%4d each), %d blows a hundred turns",
			reading.name, reading.got.turns, reading.got.landed, reading.got.dealt,
			reading.got.eachDealt(), reading.got.received, reading.got.taken,
			reading.got.eachTaken(), reading.got.landedPerHundredTurns())
	}
	// A walk that fought nothing passes every comparison below, and the two
	// readings would be a pair of noughts.
	if monolith.landed == 0 || warden.landed == 0 || monolith.received == 0 || warden.received == 0 {
		t.Fatalf("the monolith landed %d and took %d, the warden landed %d and took %d: "+
			"a duel nobody swung in measures neither unit",
			monolith.landed, monolith.received, warden.landed, warden.received)
	}

	// The armour, which is what everything else on this line was sold to buy:
	// defence at the ceiling against the warden's 640.
	if monolith.eachTaken() >= warden.eachTaken() {
		t.Errorf("the monolith is hit for %d a blow and the warden for %d: the armour that "+
			"pays for everything else is not turning anything away",
			monolith.eachTaken(), warden.eachTaken())
	}
	// The arm, which is what the shallower health pool bought.
	if monolith.eachDealt() <= warden.eachDealt() {
		t.Errorf("the monolith hits for %d a blow and the warden for %d: the heavier arm the "+
			"thinner health line paid for is not there",
			monolith.eachDealt(), warden.eachDealt())
	}
	// And the turn, which is the other half of the price: the slowest line in
	// the cast gets to swing least often.
	if monolith.landedPerHundredTurns() >= warden.landedPerHundredTurns() {
		t.Errorf("the monolith landed %d blows a hundred turns and the warden %d: the slowest "+
			"line in the cast is not acting less often, so nothing was paid for the armour",
			monolith.landedPerHundredTurns(), warden.landedPerHundredTurns())
	}
}

// TestTheTwoMonolithBuildsAreDifferentUnits is the measurement behind the two
// catalogue entries, and what it holds is the *shape* of each rather than which
// of them wins.
//
// ⚠️ **Neither of them wins this duel and that is not what is being read**, for
// the reason the slot measurement above gives: half of the wall build is spent on
// a squad that a duel does not have. What a duel can still say is what a build
// spends its turns on, and here that is the whole of what tells the two apart.
//
// Both halves of a build are asserted, because a kit is half a build: the four
// skills are read off what goes into the other unit a turn, and the one trait off
// what comes back out of being hit — `thorns` answers at power 80 and `ballast`
// at 50, both scaled by the same defence, so the difference between the two
// directions' trait slots is a figure rather than a claim.
func TestTheTwoMonolithBuildsAreDifferentUnits(t *testing.T) {
	wall := readMonolithKit(t, wallBuild, "thorns")
	forge := readMonolithKit(t, forgeBuild, "ballast")

	for _, reading := range []struct {
		name string
		got  wallReading
	}{{"wall", wall}, {"forge", forge}} {
		t.Logf("%-5s %5d turns, %5d landed for %8d (%5d a hundred turns), %5d received for %8d, replied %7d (%3d a blow taken)",
			reading.name, reading.got.turns, reading.got.landed, reading.got.dealt,
			reading.got.dealtPerHundredTurns(), reading.got.received, reading.got.taken,
			reading.got.replied, reading.got.perBlowTaken())
	}
	if wall.landed == 0 || forge.landed == 0 {
		t.Fatalf("the wall build landed %d blows and the forge build %d: a kit that swings at "+
			"nothing measures nothing", wall.landed, forge.landed)
	}
	if wall.replied == 0 || forge.replied == 0 {
		t.Fatalf("the wall build replied for %d and the forge build for %d: a trait that never "+
			"answered is a trait slot this comparison cannot see", wall.replied, forge.replied)
	}

	// The kit. The forge direction is four damaging skills and the wall direction
	// is two, so what separates them is how much of a turn goes into the other
	// unit.
	if forge.dealtPerHundredTurns() <= wall.dealtPerHundredTurns() {
		t.Errorf("the forge build deals %d a hundred turns and the wall build %d: the direction "+
			"built around the second element is not spending more of its turns on damage",
			forge.dealtPerHundredTurns(), wall.dealtPerHundredTurns())
	}
	// The trait. Both reply off the same defence, so the ordering is the two
	// traits' own powers and nothing else.
	if wall.perBlowTaken() <= forge.perBlowTaken() {
		t.Errorf("the wall build answers a blow with %d and the forge build with %d: the "+
			"direction whose damage is supposed to come out of being hit is not answering harder",
			wall.perBlowTaken(), forge.perBlowTaken())
	}
}

// TestASecondElementIsWorthWhatTheChartSaysItIs is the measurement the stage
// element made possible and nothing in this repository could take before it.
//
// Every other reading about a form is a reading of a stat line, because a form's
// stats and a form's affinity always moved together. Here they are pulled apart:
// **the same body, the same kit, the same trait and the same seeds, with the
// affinity each of the line's two forms resolves to.** What is left between the
// two arms is the chart and nothing else.
//
//	grass -> ground 1500, ground/metal 1000   (metal answers the ground weakness)
//	ice   -> ground 1500, ground/metal 2250   (metal doubles it)
//
// So the sign has to FLIP. The second element makes this unit harder to hurt with
// one of those two and easier with the other, and a reading that moved the same
// way against both would mean the dual affinity is not being applied — which is
// exactly what a fallback resolving every form to the character's own element
// would look like: both arms byte-identical, both deltas nought, and no flip.
//
// ⚠️ **It is read off damage taken per blow, and a win rate here would measure
// NOTHING.** Grass and ice are this character's two counters, and it reads 0 of 80
// against `pokemon.bulbasaur` and 0 of 80 against `pokemon.lapras` in `hexforge
// spar`. A rate would therefore be nought against nought in both arms, and the
// flip would be invisible while the mechanism worked perfectly. A per-blow figure
// is the multiplier itself: it has no floor to be pinned against, and the two
// readings it was written on came back at 143 against 211 (a ratio of 0.68 where
// the chart says 0.67) and 255 against 170 (1.50 where the chart says 1.50).
func TestASecondElementIsWorthWhatTheChartSaysItIs(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get("pokemon.onix")
	if !known {
		t.Fatal("no pokemon.onix is shipped, so there is no line with two affinities to read")
	}
	if len(character.Stages) < 2 {
		t.Fatal("the line has one form, so there is no second affinity to price")
	}
	stats, grown, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve the grown form: %v", err)
	}
	root := character.Stages[0]
	single, dual := character.ElementAt(root), character.ElementAt(grown)
	if single.String() == dual.String() {
		t.Fatalf("both forms resolve to %s, so the two arms below are the same battle twice and "+
			"nothing here prices an element", single)
	}
	kit := character.SkillsAt(progression.LevelCap, grown.Name)
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}

	for _, against := range []struct {
		name     string
		opponent string
		harder   bool // the chart says the second element makes this one hurt MORE
	}{
		{"grass", "pokemon.bulbasaur", false},
		{"ice", "pokemon.lapras", true},
	} {
		t.Run(against.name, func(t *testing.T) {
			one := readAffinity(t, stats, kit, single, against.opponent)
			two := readAffinity(t, stats, kit, dual, against.opponent)
			if one.received == 0 || two.received == 0 {
				t.Fatalf("%s landed %d blows on the single arm and %d on the dual: nothing hit "+
					"this unit, so there is no multiplier to read",
					against.opponent, one.received, two.received)
			}
			delta := two.eachTaken() - one.eachTaken()
			t.Logf("against %s: %s takes %d a blow (%d over %d), %s takes %d a blow (%d over %d), delta %+d",
				against.opponent, single, one.eachTaken(), one.taken, one.received,
				dual, two.eachTaken(), two.taken, two.received, delta)
			switch {
			case against.harder && delta <= 0:
				t.Errorf("against %s the second element changes what a blow costs by %+d, and the "+
					"chart says %s takes 2250 where %s takes 1500: it should hurt MORE",
					against.opponent, delta, dual, single)
			case !against.harder && delta >= 0:
				t.Errorf("against %s the second element changes what a blow costs by %+d, and the "+
					"chart says %s takes 1000 where %s takes 1500: it should hurt LESS",
					against.opponent, delta, dual, single)
			}
		})
	}
}

// TestTheDuelFixtureFieldsTheFormsOwnAffinity is a guard over the helper every
// duel reading in this package is taken through, and it exists because that
// helper is the one place a stage element can go wrong without anything saying
// so.
//
// `fieldedAs` resolves ONE named form and hands back four things about it: the
// stat line, the kit, the traits and the affinity. Three of those are read off
// the form and the fourth used to be read off the character — which is correct
// for twenty-two of the twenty-three shipped and silently wrong for the one line
// whose forms differ, in the direction that is hardest to notice: every figure
// on screen would still be the grown form's while the multiplier behind them was
// the root's.
//
// ⚠️ **Nothing else in the package can catch it.** The readings that go through
// this helper fight Mew and Charizard, and reverting the helper leaves every one
// of them green — Mew is the inert element, so ground and ground/metal take the
// same blow from it, and no other caller fields a line with two affinities at
// all. So the property is asserted directly, and the last clause is what stops it
// becoming a walk that agrees with itself: a cast where no form differs from its
// character would pass this with the helper reverted.
func TestTheDuelFixtureFieldsTheFormsOwnAffinity(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	differing := 0
	for _, character := range book.All() {
		for _, form := range character.Stages {
			_, affinity, _, _ := fieldedAs(t, character.ID, form.Name)
			want := character.ElementAt(form)
			if affinity.String() != want.String() {
				t.Errorf("%s fielded as %s comes back %s and the form resolves to %s: the "+
					"fixture is reading the character's element beside the form's stat line",
					character.ID, form.Name, affinity, want)
			}
			if want.String() != character.Element.String() {
				differing++
			}
		}
	}
	if differing == 0 {
		t.Fatal("no shipped form declares an affinity of its own, so every comparison above " +
			"held whichever element the fixture read and this measured nothing")
	}
	t.Logf("%d shipped form(s) resolve to an affinity that is not their character's", differing)
}

// readWall fights one shipped character's grown form against Mew over
// monolithSeeds duels, fielded the way forge.Spar fields it.
func readWall(t *testing.T, who string) wallReading {
	t.Helper()
	stats, affinity, kit, traits := fieldedAs(t, who, progression.Furthest)
	return readDuel(t, stats, kit, traits, affinity, "pokemon.mew")
}

// readMonolithKit is readWall with the loadout chosen rather than taken off the
// front of the learnset, which is what a build is.
func readMonolithKit(t *testing.T, kit []string, trait string) wallReading {
	t.Helper()
	stats, affinity, _, _ := fieldedAs(t, "pokemon.onix", progression.Furthest)
	return readDuel(t, stats, kit, []string{trait}, affinity, "pokemon.mew")
}

// readAffinity is readDuel with the affinity handed in rather than resolved,
// which is the one thing the two arms of the element reading differ in.
func readAffinity(t *testing.T, stats progression.Values, kit []string,
	affinity element.Affinity, opponent string) wallReading {
	t.Helper()
	return readDuel(t, stats, kit, []string{"endurance"}, affinity, opponent)
}

// readDuel fights one stat line, kit, trait and affinity against a shipped
// character over monolithSeeds duels and totals what happened to it.
//
// A tick names the unit CARRYING the status rather than whoever applied it, so
// damage over time is counted onto the side holding it — in a duel the side is
// enough to say who caused it. A reply carries the trait that answered on
// Event.Passive, which is the only thing telling it apart from a cast.
func readDuel(t *testing.T, stats progression.Values, kit, traits []string,
	affinity element.Affinity, opponent string) wallReading {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, opponent)

	var total wallReading
	for seedValue := 1; seedValue <= monolithSeeds; seedValue++ {
		fight, err := battle.New(books, uint64(seedValue), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: kit, Passives: traits},
			{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot, Affinity: theirAffinity,
				Stats: theirStats, Skills: theirKit, Passives: theirTraits},
		})
		if err != nil {
			t.Fatalf("new battle against %s: %v", opponent, err)
		}
		fight.Begin()
		ran, err := fight.RunToEnd(4000)
		if err != nil {
			t.Fatalf("seed %d against %s: %v", seedValue, opponent, err)
		}
		total.turns += ran
		for _, event := range fight.Drain() {
			switch {
			case event.Kind == battle.Damaged && event.Actor == "mine":
				total.landed++
				total.dealt += event.Amount
				if event.Passive != "" {
					total.replied += event.Amount
				}
			case event.Kind == battle.Damaged && event.Target == "mine":
				total.received++
				total.taken += event.Amount
			case event.Kind == battle.StatusTicked && event.Actor == "theirs":
				total.dealt += event.Amount
			}
		}
	}
	return total
}
