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

// The two directions, hardcoded here the way every other design record in this
// package is. `origin` is the mirror of Mew's `borrowed`: one carries nothing of
// its own and the other carries nothing but the original's.
var (
	breachBuild = []string{"psystrike", "psycho_cut", "shadow_ball", "dark_pulse"}
	originBuild = []string{"psychic", "hypnosis", "dream_eater", "recover"}
)

// TestPierceIsWhatMewtwoIsFor is the character stated as one measurement, and it
// is taken with the confound removed rather than around it.
//
// The obvious reading — Mewtwo's damage against three targets of different
// armour, next to somebody else's — cannot say anything, because the three
// attackers differ in element and each pairing reads the chart differently. So
// this fights ONE attacker carrying TWO skills against the same targets, and
// compares how each falls off. `psystrike` pierces eight hundred per mille and
// `body_slam` pierces nothing; both are thrown by the same unit at the same
// target on the same turn order, and Mewtwo's dark is inert against every element
// here, so the elemental term is a constant that divides out.
//
// Sixty duels a row:
//
//	target      def   psystrike   body_slam
//	Blastoise   640         524         181
//	Machamp     460         596         246
//	Magnezone   340         635         292
//
// Across the whole armour range the cast fields, the piercing blow moves about a
// fifth and the ordinary one about three fifths. That is the entire character:
// what it hits for is nearly a property of itself rather than of what it is
// pointed at.
func TestPierceIsWhatMewtwoIsFor(t *testing.T) {
	const armoured, bare = "pokemon.squirtle", "pokemon.magnemite"
	thick := readBySkill(t, []string{"psystrike", "body_slam"}, armoured)
	thin := readBySkill(t, []string{"psystrike", "body_slam"}, bare)
	for _, row := range []struct {
		name string
		got  map[string]int64
	}{{armoured, thick}, {bare, thin}} {
		t.Logf("%-20s psystrike %4d  body_slam %4d",
			row.name, row.got["psystrike"], row.got["body_slam"])
	}
	for _, id := range []string{"psystrike", "body_slam"} {
		if thick[id] == 0 || thin[id] == 0 {
			t.Fatalf("%s landed nothing against one of the two, so there is nothing to compare", id)
		}
		if thin[id] <= thick[id] {
			t.Fatalf("%s hit the armoured unit for %d and the bare one for %d: armour is not doing anything at all here, so the comparison below is empty",
				id, thick[id], thin[id])
		}
	}
	// The fall-off, in per mille of what the skill manages against the bare
	// unit. Lower is flatter.
	pierced := scale.Base - thick["psystrike"]*scale.Base/thin["psystrike"]
	plain := scale.Base - thick["body_slam"]*scale.Base/thin["body_slam"]
	t.Logf("armour costs the piercing blow %d per mille and the plain one %d", pierced, plain)
	if pierced*2 >= plain {
		t.Errorf("armour costs the piercing blow %d per mille against the plain blow's %d: piercing eight hundred is supposed to be most of the difference",
			pierced, plain)
	}
}

// readBySkill fights Mewtwo carrying exactly the given skills and returns what
// each of them came to a blow.
func readBySkill(t *testing.T, kit []string, foe string) map[string]int64 {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fielded(t, "pokemon.mewtwo")
	theirStats, theirAffinity, theirKit, theirTraits := fielded(t, foe)
	blows, dealt := map[string]int64{}, map[string]int64{}
	for seedValue := 1; seedValue <= newBuildSeeds; seedValue++ {
		fight, err := battle.New(books, uint64(seedValue), []battle.Roster{
			{ID: "mine", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity, Stats: stats,
				Skills: kit, Passives: []string{"swiftness"}},
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
			if event.Kind == battle.Damaged && event.Actor == "mine" {
				blows[event.Skill]++
				dealt[event.Skill] += event.Amount
			}
		}
	}
	each := map[string]int64{}
	for id, count := range blows {
		each[id] = dealt[id] / count
	}
	return each
}

// TestTheMutualPairFinallyHasBothHalves is what shipping a dark character does to
// the chart, and it is a claim about the data rather than about Mewtwo.
//
// Light and dark are the only mutual pair the chart declares — strong against
// each other, inert against everything else — and it had exactly one member for
// as long as Cleffa was the cast's only light unit. A mutual pair with one half
// is an edge nobody can stand on: Cleffa's element was, in play, indistinguishable
// from an inert one.
//
// So this asserts the pair is now *fielded*: both halves are carried, and the
// pairing between them reads advantage in both directions, which no other pairing
// in the cast does.
func TestTheMutualPairFinallyHasBothHalves(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the element chart: %v", err)
	}
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	pairs := chart.MutualPairs()
	if len(pairs) != 1 {
		t.Fatalf("the chart declares %d mutual pairs; everything below is about the one", len(pairs))
	}
	carried := map[element.Element][]string{}
	for _, character := range book.All() {
		for _, member := range character.Element.Elements() {
			carried[member] = append(carried[member], character.ID)
		}
	}
	for _, half := range pairs[0] {
		if len(carried[half]) == 0 {
			t.Errorf("nobody in the cast carries %s, so half the only mutual pair is still unfielded", half)
		}
		t.Logf("%-6s carried by %v", half, carried[half])
	}

	// Both ways, which is what mutual means and what a cycle never gives.
	left, right := pairs[0][0], pairs[0][1]
	forward, back := chart.Multiplier(left, right), chart.Multiplier(right, left)
	advantage := chart.Multipliers().Advantage
	if forward != advantage || back != advantage {
		t.Errorf("%s hits %s at %d and %s hits %s at %d, and both should be the advantage %d",
			left, right, forward, right, left, back, advantage)
	}

	// And nothing else in the cast is like that, or "the pair" would be a name
	// for something the chart does everywhere.
	both := 0
	for _, mine := range book.All() {
		for _, theirs := range book.All() {
			out := chart.MultiplierAgainst(mine.Element.Primary(), theirs.Element)
			in := chart.MultiplierAgainst(theirs.Element.Primary(), mine.Element)
			if out > scale.Base && in > scale.Base {
				both++
			}
		}
	}
	if both == 0 {
		t.Fatal("no pairing in the cast is strong in both directions, so the pair is declared and not fielded")
	}
	t.Logf("%d ordered pairings in the cast are strong both ways", both)
}

// TestTheCloneKnowsWhatTheOriginalKnowsExceptHowToBeSomethingElse is the pair's
// story told as a carry rule, and it costs nothing: `mythic` is a species, so a
// skill gated on it is open to both of them by construction.
//
// That makes `mythic` the first species in the book with two members — `dragon`,
// `plant` and the rest each gate for exactly one character — and therefore the
// first time the axis does the thing it was added for: a lineage outliving the
// character it was written on.
//
// The exception is the whole point. `transform` is gated on the character rather
// than the species, so the one thing the clone cannot do is be anything else.
func TestTheCloneKnowsWhatTheOriginalKnowsExceptHowToBeSomethingElse(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skill book: %v", err)
	}
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	mythics := []string{}
	for _, character := range book.All() {
		for _, kind := range character.Species {
			if kind == "mythic" {
				mythics = append(mythics, character.ID)
			}
		}
	}
	if len(mythics) != 2 {
		t.Fatalf("the species is carried by %v; this is about the pair", mythics)
	}

	carries := func(who, what string) bool {
		character, known := book.Get(who)
		if !known {
			t.Fatalf("no character %q", who)
		}
		for _, carried := range character.Skills {
			if carried.ID == what {
				return true
			}
		}
		return false
	}
	shared, exceptions := 0, []string{}
	for _, declared := range skills.Skills() {
		if declared.Restrict == nil {
			continue
		}
		gated := false
		for _, kind := range declared.Restrict.Species {
			if kind == "mythic" {
				gated = true
			}
		}
		if !gated {
			continue
		}
		if carries("pokemon.mew", declared.ID) && carries("pokemon.mewtwo", declared.ID) {
			shared++
			continue
		}
		exceptions = append(exceptions, declared.ID)
	}
	if shared == 0 {
		t.Fatal("no skill is gated on the species and carried by both, so nothing here measured the axis")
	}
	if len(exceptions) != 0 {
		t.Errorf("%v are gated on the species and not carried by both: a species gate that one of its two members declines is a character gate written the long way",
			exceptions)
	}
	t.Logf("%d skills gated on the species, carried by both", shared)

	// And the one the clone does not get, which is gated the other way on
	// purpose.
	if carries("pokemon.mewtwo", "transform") {
		t.Errorf("the clone carries transform: being anything else is the one thing it was not given")
	}
	if !carries("pokemon.mew", "transform") {
		t.Errorf("the original does not carry transform, so there is no exception to be the exception")
	}
}

// TestTheSameLoadoutIsWorseOnTheClone is the two builds, and the second of them
// is measured against the character it was taken from.
//
// `mewtwo.origin` carries Mew's own four on Mewtwo's frame, so the two readings
// differ in nothing but the stat line — which is the only measurement in this
// package that puts one loadout on two bodies, and the only way to say what the
// lab actually changed.
//
// ⚠️ **It changed it for the worse, on every column the kit is about.** Given
// Mew's skills the clone lasts fewer turns, heals less and deals less: `recover`
// scales off attack and Mewtwo's is the lower of the two, and nothing in that kit
// pierces anything. What Mewtwo bought is not "more" — it is speed, and eight
// hundred per mille of pierce it can only spend on skills that have it, which is
// what `breach` is and why it out-damages both by better than two to one.
func TestTheSameLoadoutIsWorseOnTheClone(t *testing.T) {
	breach := readBuild(t, "pokemon.mewtwo", progression.Furthest, breachBuild, "berserk")
	clone := readBuild(t, "pokemon.mewtwo", progression.Furthest, originBuild, "elusive")
	original := readBuild(t, "pokemon.mew", progression.Furthest, originBuild, "elusive")
	for _, reading := range []struct {
		name string
		got  buildReading
	}{{"breach", breach}, {"origin on the clone", clone}, {"origin on Mew", original}} {
		t.Logf("%-20s %3d turns, dealt %5d, healed %5d, inflicted %3d",
			reading.name, reading.got.turns, reading.got.dealt,
			reading.got.healed, reading.got.inflicted)
	}

	// The same four skills, two bodies, and the clone is behind on all of it.
	if clone.turns >= original.turns {
		t.Errorf("the clone lasted %d turns on the original's kit against %d: the thinner frame is supposed to cost it",
			clone.turns, original.turns)
	}
	if clone.healed >= original.healed {
		t.Errorf("the clone healed %d on the original's kit against %d: the heal reads attack and the clone's is lower",
			clone.healed, original.healed)
	}
	if clone.dealt >= original.dealt {
		t.Errorf("the clone dealt %d on the original's kit against %d: with nothing in that kit piercing, there is nothing for its own numbers to buy",
			clone.dealt, original.dealt)
	}

	// And what it did buy, spent on skills that can spend it.
	if breach.dealt <= 2*clone.dealt {
		t.Errorf("the breach build dealt %d against the borrowed kit's %d on the same body: piercing is supposed to be most of what this character is",
			breach.dealt, clone.dealt)
	}
	if breach.turns >= clone.turns {
		t.Errorf("the breach build took %d turns against %d: it is the direction that ends a fight rather than sits in one",
			breach.turns, clone.turns)
	}
}
