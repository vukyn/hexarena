package seed_test

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// The subject of ENG-006 step 3, spelled once. The trait is the first shipped
// thing to carry a gate at the top of the health bar, and the first to carry a
// gate on a *grant* rather than on the whole trait, so these are the only tests
// in the repository whose subject is shipped data rather than a fixture.
const (
	freshGuard      = "pristine"
	freshGuardFloor = "plated"    // the tier that never lapses
	freshGuardTier  = "fortified" // the tier behind the gate
	freshGuardGate  = 700         // the share of maximum health it holds above
	freshCarrier    = "pokemon.magnemite"
	// endurance is what the trait is a trade against: it grants `toughened`,
	// which is worth MORE than the ungated tier here and less than the two
	// together. → TestTheFreshGuardIsATradeAgainstEnduranceRatherThanAnUpgrade.
	freshRival = "endurance"
	// freshSparring is who wears the holder down in the hand-played battle: a
	// slugger with a real kit, because the gate has to be *crossed* and a
	// sparring partner that cannot cross it measures nothing. → the fixture note
	// in memory/a-gate-fixture-must-be-crossable-by-the-thing-under-test.md.
	freshSparring = "pokemon.machop"
)

// TestTheFreshGuardCarriesTwoRealTiersAndTheSecondSaturatesIntoTheFirst is the
// shape claim and the arithmetic under it, measured on the shipped books.
//
// Two tiers on **one** trait is the whole reason passive.Grant.While exists: a
// trait-level gate can only say "+Z while above Y, nothing below", and splitting
// the two tiers across two traits would cost two of the one trait slot a
// placement has. So the shape asserted here is an ungated grant beside a gated
// one, on a trait with no gate of its own.
//
// ⚠️ **The two tiers do not add up, and that is measured rather than assumed.**
// modifier.Set.Stat saturates a change towards a ceiling instead of applying it,
// so the second tier lands in an already-moved stat and is worth less there than
// it is worth alone. Pricing this trait by adding its two faces would overstate
// it, which is why the item it ships under says to price by measurement — the
// figures are logged, and the *relations* are what is asserted, because the
// figures move the day anybody retunes a ceiling.
func TestTheFreshGuardCarriesTwoRealTiersAndTheSecondSaturatesIntoTheFirst(t *testing.T) {
	held, err := mustPassives(t).Lookup(freshGuard)
	if err != nil {
		t.Fatalf("look up %s: %v", freshGuard, err)
	}
	if held.While != nil {
		t.Fatalf("%s is gated as a whole, so it has one tier rather than two: the "+
			"two-tier shape is an ungated grant beside a grant with its own gate",
			freshGuard)
	}
	if len(held.Grants) != 2 {
		t.Fatalf("%s grants %d things and the two-tier shape is exactly two",
			freshGuard, len(held.Grants))
	}
	floor, tier := held.Grants[0], held.Grants[1]
	if floor.Status != freshGuardFloor || tier.Status != freshGuardTier {
		t.Fatalf("%s grants %q then %q, want %q then %q",
			freshGuard, floor.Status, tier.Status, freshGuardFloor, freshGuardTier)
	}
	if held.GateOver(floor) != nil {
		t.Errorf("the %q tier is gated, so nothing this trait grants is always on",
			floor.Status)
	}
	gate := held.GateOver(tier)
	if gate == nil {
		t.Fatalf("the %q tier carries no gate, so the trait is two tiers of the same thing",
			tier.Status)
	}
	if !gate.AtTop() {
		t.Errorf("the %q tier is gated at the bottom of the bar, which is blaze's "+
			"polarity rather than this one's", tier.Status)
	}
	if gate.Threshold() != freshGuardGate {
		t.Errorf("the gate is written at %d and the measurement it was priced by is at %d",
			gate.Threshold(), freshGuardGate)
	}

	// And the arithmetic, on the carrier's own line, because a share is a share
	// of something and the something is what saturates.
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load the limits: %v", err)
	}
	bounds, err := seed.ModifierBounds()
	if err != nil {
		t.Fatalf("load the bounds: %v", err)
	}
	base, _, _, _ := fieldedFurthest(t, freshCarrier)
	defenceUnder := func(ids ...string) int64 {
		t.Helper()
		carried := status.Set{}
		for _, id := range ids {
			kind, err := statuses.Lookup(id)
			if err != nil {
				t.Fatalf("look up %s: %v", id, err)
			}
			carried.Hold(kind, 0, kind.MaxStacks)
		}
		return carried.Modifiers().Stats(base, limits.Ceilings, bounds)[progression.Defense]
	}
	bare := base[progression.Defense]
	onlyFloor := defenceUnder(freshGuardFloor)
	bothTiers := defenceUnder(freshGuardFloor, freshGuardTier)
	onlyTier := defenceUnder(freshGuardTier)
	faces := int64(0)
	for _, id := range []string{freshGuardFloor, freshGuardTier} {
		kind, err := statuses.Lookup(id)
		if err != nil {
			t.Fatalf("look up %s: %v", id, err)
		}
		for _, term := range kind.Modifiers {
			if term.Target == modifier.Defense {
				faces += int64(term.Amount)
			}
		}
	}
	asked := bare + bare*faces/scale.Base
	t.Logf("defence %d bare, %d under %q alone, %d under %q alone, %d under both; "+
		"the two faces sum to %d parts per thousand, which asks for %d",
		bare, onlyFloor, freshGuardFloor, onlyTier, freshGuardTier, bothTiers,
		faces, asked)

	if onlyFloor <= bare {
		t.Errorf("the %q tier moves the line to %d from %d, so the tier that never "+
			"lapses is worth nothing and the trait has one tier",
			freshGuardFloor, onlyFloor, bare)
	}
	if bothTiers <= onlyFloor {
		t.Errorf("both tiers come to %d and the ungated one alone comes to %d, so the "+
			"gated tier is worth nothing", bothTiers, onlyFloor)
	}
	if bothTiers >= asked {
		t.Errorf("both tiers come to %d and their two faces ask for %d: a stat "+
			"saturates towards its ceiling rather than adding, so two tiers may "+
			"never come to the sum of what they say", bothTiers, asked)
	}
	// The direction of the shortfall, which is the half a reader prices wrong:
	// the gated tier applies into a stat the ungated one has already moved, so it
	// buys less there than it buys on its own.
	if marginal, alone := bothTiers-onlyFloor, onlyTier-bare; marginal >= alone {
		t.Errorf("the gated tier is worth %d on top of the ungated one and %d on its "+
			"own: the second term into one stat has to be the cheaper of the two",
			marginal, alone)
	}
}

// TestTheFreshGuardIsCarried is the rule every shipped mechanism obeys here: a
// trait nobody can field is dead data, and a measurement of one is a measurement
// of nothing.
//
// The carrier is Magnemite, and the reason is three-part rather than thematic
// alone. Its line is the one this trait reads on — a modest defence and a large
// frame, so a share of defence has room to matter; a bombardier is artillery,
// which wants protection before the first blow and has no answer once it is
// being ground down; and it is the carrier the decoupling below was measurable
// on at all, which the item asked for before it asked for anything else.
func TestTheFreshGuardIsCarried(t *testing.T) {
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get(freshCarrier)
	if !known {
		t.Fatalf("no character %q", freshCarrier)
	}
	carried := false
	for _, mine := range character.Passives {
		if mine.ID == freshGuard {
			carried = true
		}
	}
	if !carried {
		t.Errorf("%s carries no %s, so nothing in the game fields the only trait that "+
			"holds a gate at the top of the health bar", freshCarrier, freshGuard)
	}
	// And reachable at the cap, which is where every figure quoted for it was
	// taken: a trait unlocked above the cap is one no placement can name.
	available := character.PassivesAt(progression.LevelCap, progression.Furthest)
	found := false
	for _, id := range available {
		if id == freshGuard {
			found = true
		}
	}
	if !found {
		t.Errorf("%s does not offer %s at the cap, and %v is what it offers",
			freshCarrier, freshGuard, available)
	}
}

// TestTheFreshGuardIsATradeAgainstEnduranceRatherThanAnUpgrade is why the
// ungated tier is smaller than the trait it competes with.
//
// A placement brings one trait, so a trait that is another trait plus something
// is not a choice — it retires the other one. The ungated tier is therefore
// *under* what `endurance` grants outright: below the gate this trait is the
// worse of the two and above it the better, which is what makes the gate a price
// rather than a decoration.
//
// ⚠️ Measured, the size of that floor is the whole of whether the trait is worth
// having: at a floor equal to endurance's own the trait was better against 12 of
// 21 opponents and worse against 2, and at half of it, better against 2 and
// worse against 11. → README.md § *What a guard that holds while its holder is
// fresh is worth*.
func TestTheFreshGuardIsATradeAgainstEnduranceRatherThanAnUpgrade(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load the limits: %v", err)
	}
	bounds, err := seed.ModifierBounds()
	if err != nil {
		t.Fatalf("load the bounds: %v", err)
	}
	passives := mustPassives(t)
	base, _, _, _ := fieldedFurthest(t, freshCarrier)
	under := func(trait string, gateOpen bool) int64 {
		t.Helper()
		held, err := passives.Lookup(trait)
		if err != nil {
			t.Fatalf("look up %s: %v", trait, err)
		}
		carried := status.Set{}
		for _, grant := range held.Grants {
			if held.GateOver(grant) != nil && !gateOpen {
				continue
			}
			kind, err := statuses.Lookup(grant.Status)
			if err != nil {
				t.Fatalf("look up %s: %v", grant.Status, err)
			}
			carried.Hold(kind, 0, grant.Stacks)
		}
		return carried.Modifiers().Stats(base, limits.Ceilings, bounds)[progression.Defense]
	}
	rival := under(freshRival, true)
	worn := under(freshGuard, false)
	fresh := under(freshGuard, true)
	t.Logf("defence: %s %d, %s %d worn and %d fresh", freshRival, rival,
		freshGuard, worn, fresh)
	if worn >= rival {
		t.Errorf("%s comes to %d once its gate has shut and %s comes to %d: a trait "+
			"that is never worse than the one it competes with retires it rather "+
			"than being a choice against it", freshGuard, worn, freshRival, rival)
	}
	if fresh <= rival {
		t.Errorf("%s comes to %d with its gate open and %s comes to %d, so there is "+
			"no state in which taking the gate buys anything",
			freshGuard, fresh, freshRival, rival)
	}
}

// TestTheFreshGuardLapsesAsItsHolderIsWornAndComesBackWhenHealed is the gate
// itself, on the shipped trait, in a battle played by hand.
//
// Three claims, and the health figures are asserted rather than the events
// alone: the tier is on at the opening board, because at enlistment a unit is at
// full health and a gate at the top of the bar is therefore open; it is released
// at or under the share it is written against; and it is held again the moment
// its holder is healed back over that share. The last one is the half that makes
// this a gate rather than a one-way door, and no autopilot battle plays it —
// choosing to heal instead of to strike is exactly what Suggest will not do.
func TestTheFreshGuardLapsesAsItsHolderIsWornAndComesBackWhenHealed(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	stats, affinity, _, _ := fieldedFurthest(t, freshCarrier)
	// A sparring partner that hits hard enough to cross the gate and not hard
	// enough to finish the holder before it can heal, and a holder given a
	// restore its own learnset does not carry: what is under test is the gate,
	// and a battle that cannot climb back over it can only measure half of one.
	theirStats, theirAffinity, _, _ := fieldedFurthest(t, freshSparring)
	fight, err := battle.New(books, 7, []battle.Roster{
		{ID: "holder", Side: hex.SideAlly, Slot: buildSlot, Affinity: affinity,
			Stats: stats, Skills: []string{"thunder_shock", "recover"},
			Passives: []string{freshGuard}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: buildSlot,
			Affinity: theirAffinity, Stats: theirStats,
			Skills: []string{"cross_chop", "rock_throw"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	maximum := stats[progression.HP]
	opening := 0
	for _, event := range fight.Drain() {
		if event.Kind == battle.PassiveHeld && event.Actor == "holder" &&
			event.Status == freshGuardTier {
			opening++
		}
	}
	if opening != 1 {
		t.Fatalf("the opening board announces the %q tier %d times: a gate at the top "+
			"of the bar is open at full health, so it is on before the first turn",
			freshGuardTier, opening)
	}

	var releasedAt, reheldAt int64 = -1, -1
	health := maximum
	for turn := 0; turn < 400 && reheldAt < 0; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if !prompt.Skipped {
			// Both sides are played by hand, in two phases, and that is the
			// experiment rather than a shortcut. Until the tier lapses the
			// sparring partner throws whatever it can and the holder trades; once
			// it has lapsed the partner stands off and the holder drinks the way
			// back up. Playing the second phase is the only way to reach it: the
			// shipped restore is worth a fraction of a shipped blow, so a partner
			// left swinging kills the holder long before it can climb back over
			// the gate — which is what the first version of this test measured.
			//
			// ⚠️ Every skill the partner has is a blow, deliberately. A partner
			// picking its own first available option was the version before that,
			// and it braced for four hundred turns while the holder stood at full
			// health and nothing crossed anything.
			acted := false
			for _, option := range prompt.Options {
				if !option.Available() {
					continue
				}
				if prompt.Unit == "holder" {
					want := "thunder_shock"
					if releasedAt >= 0 {
						want = "recover"
					}
					if option.Skill != want {
						continue
					}
				} else if releasedAt >= 0 {
					break
				}
				if err := fight.Act(option.Skill, option.Aims[0]); err != nil {
					t.Fatalf("act %s: %v", option.Skill, err)
				}
				acted = true
				break
			}
			if !acted {
				if err := fight.Pass("waiting"); err != nil {
					t.Fatalf("pass: %v", err)
				}
			}
		}
		for _, event := range fight.Drain() {
			switch {
			case event.Kind == battle.Damaged && event.Target == "holder":
				health = event.Remaining
			case event.Kind == battle.Healed && event.Actor == "holder":
				health = event.Remaining
			case event.Actor != "holder" || event.Status != freshGuardTier:
			case event.Kind == battle.PassiveReleased && releasedAt < 0:
				releasedAt = health
			case event.Kind == battle.PassiveHeld && releasedAt >= 0 && reheldAt < 0:
				reheldAt = health
			}
		}
		if fight.Finished() {
			break
		}
	}
	if releasedAt < 0 {
		t.Fatal("the holder was never worn down far enough to lose the tier, so the " +
			"gate was never crossed and nothing here measured it")
	}
	if reheldAt < 0 {
		t.Fatal("the tier never came back after the holder was healed over the gate: " +
			"a gate is a door both ways, and one that only shuts is a trait that " +
			"lapses permanently on the first bad turn")
	}
	t.Logf("maximum %d; the tier lapsed at %d and came back at %d, against a gate at "+
		"%d parts per thousand (%d points)",
		maximum, releasedAt, reheldAt, freshGuardGate, maximum*freshGuardGate/scale.Base)
	if scale.AtOrAboveShare(releasedAt, maximum, freshGuardGate) {
		t.Errorf("the tier lapsed at %d of %d, which is still at or above the gate: it "+
			"went off while it was in force", releasedAt, maximum)
	}
	if !scale.AtOrAboveShare(reheldAt, maximum, freshGuardGate) {
		t.Errorf("the tier came back at %d of %d, which is under the gate: it came on "+
			"while it was out of force", reheldAt, maximum)
	}
}

// TestTheFreshGuardsGateSeparatesTwoMatchupsThatItsMagnitudeCannot is the number
// that justifies the trait existing, and it is the item's whole argument put on
// an instrument.
//
// Every dial a trait offered before this one was a **stat**, and a stat cannot
// separate two matchups: both gates it moves are the same event — a strike
// crossing a kill threshold — so every amount moves them together. **Duration is
// not a stat.** The reading below is three arms of the same tier over the same
// seeds, against two opponents:
//
//   - the floor: the ungated tier alone, which is what the trait is worth once
//     its gate has shut;
//   - the subject: the shipped trait, the same tier behind `above_health`;
//   - the payload: the same two tiers with **no gate at all**, which is what a
//     plain stat of this size is worth.
//
// The payload is what makes this a measurement rather than an anecdote. Without
// it "the gate is worth little here" cannot be told from "the tier is worth
// little here", and a null with no calibration beside it is a blind reading.
//
// ⚠️ **A one-way mirror is not a measurement**, so every arm is fought both ways
// round over the same seeds, and the mirror control — the same kit and the same
// trait on both halves — has to read exactly even or the instrument is tilted
// before any arm is read.
func TestTheFreshGuardsGateSeparatesTwoMatchupsThatItsMagnitudeCannot(t *testing.T) {
	books := freshGuardArms(t)
	stats, affinity, kit, _ := fieldedFurthest(t, freshCarrier)
	if mirror := freshMirror(t, books, stats, affinity, kit, freshGuard); mirror != scale.Base/2 {
		t.Fatalf("the mirror reads %d and an identical pair comes to exactly %d: the "+
			"instrument is tilted and no arm below means anything",
			mirror, scale.Base/2)
	}
	type arm struct {
		opponent      string
		floor, gated  int
		payload, gain int
	}
	var arms []arm
	for _, opponent := range []string{freshHeldMatchup, freshLostMatchup} {
		floor := freshDuel(t, books, stats, affinity, kit, freshFloorFixture, opponent)
		gated := freshDuel(t, books, stats, affinity, kit, freshGuard, opponent)
		payload := freshDuel(t, books, stats, affinity, kit, freshAlwaysFixture, opponent)
		arms = append(arms, arm{opponent, floor, gated, payload, gated - floor})
		t.Logf("%-18s floor %d, gated %d, payload %d: the tier is worth %d ungated and "+
			"%d behind the gate", opponent, floor, gated, payload,
			payload-floor, gated-floor)
	}
	held, lost := arms[0], arms[1]
	for _, each := range arms {
		if each.payload <= each.floor {
			t.Fatalf("against %s the ungated tier is worth %d, so there is no payload "+
				"for the gate to deliver a share of and every ratio below is noise",
				each.opponent, each.payload-each.floor)
		}
	}
	// Half one: a magnitude cannot tell the two matchups apart. The same tier
	// with no gate on it is worth about the same against both, so no amount of
	// it would ever have been a lever between them.
	heldPayload, lostPayload := held.payload-held.floor, lost.payload-lost.floor
	if spread := spreadOf(heldPayload, lostPayload); spread > 1500 {
		t.Errorf("the ungated tier is worth %d against %s and %d against %s, %d parts "+
			"per thousand apart: a magnitude already separates these two matchups, so "+
			"they are the wrong pair to show that a duration does something a "+
			"magnitude cannot", heldPayload, held.opponent, lostPayload,
			lost.opponent, spread)
	}
	// Half two: the same tier behind the same gate is worth several times as much
	// against one as against the other. That is the decoupling, and it is the
	// only currency in the game that has ever produced one.
	if spread := spreadOf(held.gain, lost.gain); spread < 3000 {
		t.Errorf("the gated tier is worth %d against %s and %d against %s, only %d "+
			"parts per thousand apart: the gate is not separating the two matchups "+
			"and the trait is a discount on a stat rather than a different kind of "+
			"dial", held.gain, held.opponent, lost.gain, lost.opponent, spread)
	}
}

// spreadOf is the larger of two figures over the smaller, in parts per thousand,
// and nought when either is not positive — a ratio taken over a figure that
// moved the wrong way is a number with no meaning, and the callers above check
// the sign before they read it.
func spreadOf(first, second int) int {
	if first <= 0 || second <= 0 {
		return 0
	}
	if first < second {
		first, second = second, first
	}
	return first * scale.Base / second
}

// The two matchups the decoupling is read on, and they were chosen by the
// measurement rather than by their names.
//
// Against Cleffa, Magnezone's losses are decided while it is still near the top
// of its bar and nine tenths of the tier's whole value sits above the gate;
// against Gastly they are decided down the bar and less than a fifth of it does.
// The two are not a short battle and a long one — Cleffa is the *longer* of the
// two, which is why the item's "a rout that ends at full health" is the axis and
// "a quick battle" is not. → README.md § *What a guard that holds while its
// holder is fresh is worth*.
const (
	freshHeldMatchup = "pokemon.cleffa"
	freshLostMatchup = "pokemon.gastly"
	// freshSeeds is per arrangement, so an arm is fought over twice this many
	// battles. It is the count the reported figures were taken at.
	freshSeeds = 500
)

// The two fixture arms. They are declarations rather than shipped traits because
// neither is a trait anybody should be able to field: the floor is the shipped
// trait with its second tier amputated and the payload is the shipped trait with
// its price removed, and shipping either would be shipping a measurement.
const (
	freshFloorFixture  = "fixture_floor"
	freshAlwaysFixture = "fixture_always"
)

// freshGuardArms is the shipped books with the two fixture arms parsed in beside
// the real trait.
//
// Patched through the *parse* rather than by building a book by hand, and in
// memory rather than by editing the shipped file and putting it back: that is how
// every trait sweep in README.md was taken, it is exact and reproducible, and a
// fixture written into the real data directory is a trait somebody finds in the
// game six months later.
func freshGuardArms(t *testing.T) battle.Books {
	t.Helper()
	raw, err := seed.PassivesFile()
	if err != nil {
		t.Fatalf("read the passive book: %v", err)
	}
	var declared map[string]any
	if err := json.Unmarshal(raw, &declared); err != nil {
		t.Fatalf("decode the passive book: %v", err)
	}
	list, ok := declared["passives"].([]any)
	if !ok {
		t.Fatal("the passive book declares no passives, so there is nothing to patch")
	}
	declared["passives"] = append(list,
		map[string]any{
			"id":     freshFloorFixture,
			"grants": []any{map[string]any{"status": freshGuardFloor, "stacks": 1}},
		},
		map[string]any{
			"id": freshAlwaysFixture,
			"grants": []any{
				map[string]any{"status": freshGuardFloor, "stacks": 1},
				map[string]any{"status": freshGuardTier, "stacks": 1},
			},
		},
	)
	patched, err := json.Marshal(declared)
	if err != nil {
		t.Fatalf("encode the patched passive book: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the statuses: %v", err)
	}
	parsed, err := passive.ParseBook(patched, passive.Deps{Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the patched passive book: %v", err)
	}
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	books.Passives = parsed
	return books
}

// freshDuel fights the carrier's kit under one trait against a shipped
// character, both ways round over the same seeds, and reports the carrier's
// share in parts per thousand.
//
// Both ways for the reason every duel in this package takes both: the turn queue
// breaks a tie by enlistment order, so a one-way figure carries the first slot's
// advantage into the answer.
func freshDuel(t *testing.T, books battle.Books, stats progression.Values,
	affinity element.Affinity, kit []string, trait, opponent string) int {
	t.Helper()
	theirStats, theirAffinity, theirKit, theirTrait := fieldedFurthest(t, opponent)
	won, fought := 0, 0
	for _, mineFirst := range []bool{true, false} {
		for which := 1; which <= freshSeeds; which++ {
			mine := battle.Roster{ID: "mine", Side: hex.SideAlly, Slot: buildSlot,
				Affinity: affinity, Stats: stats, Skills: kit,
				Passives: []string{trait}}
			theirs := battle.Roster{ID: "theirs", Side: hex.SideEnemy, Slot: buildSlot,
				Affinity: theirAffinity, Stats: theirStats, Skills: theirKit,
				Passives: theirTrait}
			order := []battle.Roster{mine, theirs}
			if !mineFirst {
				mine.Side, theirs.Side = hex.SideEnemy, hex.SideAlly
				order = []battle.Roster{theirs, mine}
			}
			result, err := freshFight(books, order, mineFirst, uint64(which))
			if err != nil {
				t.Fatalf("seed %d against %s: %v", which, opponent, err)
			}
			if result == nil {
				continue
			}
			fought++
			if *result {
				won++
			}
		}
	}
	if fought == 0 {
		t.Fatalf("no battle against %s ended, so nothing was measured", opponent)
	}
	return won * scale.Base / fought
}

// freshMirror is the same kit and the same trait on both halves, which reads
// exactly even by construction: a figure that did not would mean the swap is not
// cancelling what it is there to cancel.
func freshMirror(t *testing.T, books battle.Books, stats progression.Values,
	affinity element.Affinity, kit []string, trait string) int {
	t.Helper()
	won, fought := 0, 0
	for _, mineFirst := range []bool{true, false} {
		for which := 1; which <= freshSeeds; which++ {
			mine := battle.Roster{ID: "mine", Side: hex.SideAlly, Slot: buildSlot,
				Affinity: affinity, Stats: stats, Skills: kit,
				Passives: []string{trait}}
			theirs := mine
			theirs.ID, theirs.Side = "theirs", hex.SideEnemy
			order := []battle.Roster{mine, theirs}
			if !mineFirst {
				mine.Side, theirs.Side = hex.SideEnemy, hex.SideAlly
				order = []battle.Roster{theirs, mine}
			}
			result, err := freshFight(books, order, mineFirst, uint64(which))
			if err != nil {
				t.Fatalf("mirror seed %d: %v", which, err)
			}
			if result == nil {
				continue
			}
			fought++
			if *result {
				won++
			}
		}
	}
	if fought == 0 {
		t.Fatal("no mirror battle ended, so the control measured nothing")
	}
	return won * scale.Base / fought
}

// freshFight runs one battle and says whether the carrier won it, or nothing at
// all when it came to no decision.
func freshFight(books battle.Books, order []battle.Roster, mineFirst bool,
	which uint64) (*bool, error) {
	fight, err := battle.New(books, which, order)
	if err != nil {
		return nil, fmt.Errorf("new battle: %w", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(4000); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	winner, decided := fight.Winner()
	if !decided {
		return nil, nil
	}
	mine := (winner == hex.SideAlly) == mineFirst
	return &mine, nil
}

// fieldedFurthest is fielded with a forking evolution line resolved to its first
// arm rather than refused.
//
// fielded asks Resolve for progression.Furthest, which is two answers on a line
// that forks and a refusal rather than a pick — so a sweep over the whole cast
// stops on the one character that forks. Which arm is taken does not matter to a
// figure about somebody else's trait; being able to ask at all does.
func fieldedFurthest(t *testing.T, id string) (progression.Values, element.Affinity, []string, []string) {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get(id)
	if !known {
		t.Fatalf("no character %q", id)
	}
	forms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("the grown forms of %s: %v", id, err)
	}
	names := make([]string, 0, len(forms))
	for _, form := range forms {
		names = append(names, form.Name)
	}
	sort.Strings(names)
	return fieldedAs(t, id, names[0])
}
