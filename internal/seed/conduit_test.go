package seed_test

import (
	"slices"
	"sort"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestAConduitPaysForWhatItDischarges is the bound the detonate rule cannot
// draw, for the statuses it cannot price.
//
// A counter does nothing to its holder, so "what consuming it gives up" — the
// whole of TestADetonateIsWorthLessThanItsBreakEven's arithmetic — has no answer,
// and the rule would be bounding a burst by nothing. What bounds a conduit
// instead is that **it pays out of its own pocket**: the arc is funded by the
// share of its own blow it damps, so a stack is worth what that trade is worth
// and not a coin more.
//
// Three clauses, and each is a way the trade could have been a free lunch:
//
//   - the arc must beat the damping, or nobody would ever charge;
//   - and not beat it *twice over*, which is the ceiling the detonate rule draws
//     from the other side — a stack may double what the turn was worth, no more;
//   - and the skill must survive being damped, because a strike of no power is
//     never rolled and a strike that is never rolled never discharges.
//
// ⚠️ **Both halves of the trade are read at the SMALLEST pile the skill will
// fire on, and getting that wrong squeezed the nuke into a shape it could not
// occupy.** The damping is paid once a strike; the arc is paid per *stack*. For a
// drip those are the same number because it takes one stack a strike — so the
// rule was written comparing one against one and looked correct. A nuke fires on
// two at the least, so comparing its per-stack figure against the whole damping
// asked it to beat a full blow with half a payment, and every number that
// satisfied it made the skill worthless. What the trade actually is: what the
// skill gives up, against what the least pile it accepts hands back.
func TestAConduitPaysForWhatItDischarges(t *testing.T) {
	book, statuses := mustSkills(t), mustStatuses(t)
	found := 0
	for _, current := range book.Skills() {
		if !current.Requires.Arcs() {
			continue
		}
		found++
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		if kind.Category != status.Charge {
			t.Errorf("skill %q discharges %q, a %s rather than a counter: what a conduit spends has to be something a unit is carrying for that purpose",
				current.ID, kind.ID, kind.Category)
		}
		damped := current.Power - current.Requires.DampedPower(current.Power)
		arc := current.Requires.ArcPower * current.Requires.MinStacks
		switch {
		case damped <= 0:
			t.Errorf("skill %q arcs for %d and damps nothing, so the discharge is free and the counter is a bonus rather than a bargain",
				current.ID, arc)
		case arc <= damped:
			t.Errorf("skill %q gives up %d power to fire %d off its smallest pile, so meeting the condition is worse than missing it",
				current.ID, damped, arc)
		case arc > damped*2:
			t.Errorf("skill %q gives up %d power to fire %d off its smallest pile, over twice as good: "+
				"the least pile it accepts may double what the turn was worth, which is where a detonate stops too",
				current.ID, damped, arc)
		}
		if left := current.Requires.DampedPower(current.Power); left <= 0 {
			t.Errorf("skill %q damps its %d power to %d, so it never rolls a strike and never discharges at all",
				current.ID, current.Power, left)
		}
		// Two shapes and no third. A conduit takes ONE stack a strike, which is
		// what makes a pile worth having; a NUKE takes the pile, which is what
		// makes a pile worth waiting for. Any count in between is a magazine that
		// empties in three shots — worse than either at both of their jobs, and a
		// figure no reader could say the point of.
		switch current.Requires.ConsumeStacks {
		case 1:
			// A drip. Nothing more to say about it here.
		case 0:
			// A nuke, and the two things that keep it from being simply better.
			if current.Requires.ChainsOn() {
				t.Errorf("skill %q takes the whole pile of %q AND chains: one cast would empty the board's counters and be paid for every one of them",
					current.ID, kind.ID)
			}
			// It has to be worth *waiting* for, which means it may not fire on the
			// first stack somebody happens to be carrying.
			if current.Requires.MinStacks < 2 {
				t.Errorf("skill %q takes the whole pile of %q but fires on %d stack: a nuke that goes off on the first one is a drip with a longer cooldown",
					current.ID, kind.ID, current.Requires.MinStacks)
			}
		default:
			t.Errorf("skill %q spends %d stacks of %q: a conduit spends one a strike and a nuke spends the pile, and there is no third thing for a count in between to be",
				current.ID, current.Requires.ConsumeStacks, kind.ID)
		}
		if current.Requires.BonusPower != 0 {
			t.Errorf("skill %q is paid both %d bonus power and an arc for one stack, which is the same purchase made twice",
				current.ID, current.Requires.BonusPower)
		}
	}
	if found == 0 {
		t.Skip("nothing discharges a counter, so there is nothing here to hold")
	}
	t.Logf("%d conduits shipped", found)
}

// TestANukePaysBetterPerStackAndWorsePerTurn is the relationship between the two
// shapes, and the reason both are worth authoring.
//
// ⚠️ **This was first written the other way round and the reasoning was wrong.**
// A nuke takes more stacks at once, so it looked like the larger purchase and was
// held to a *poorer* rate per stack. Measured, that made it a skill nobody would
// ever bring: at a pile of six it dealt 472 in one turn on twice the cooldown,
// against a drip landing about the same on the primary *and* on everything the
// chain reached — after five turns of charging that the drip would have spent as
// it went.
//
// What that missed is what each shape actually costs. A drip converts stacks the
// moment it has them and does it across a whole chain, so it is paid in **breadth
// and immediacy**. A nuke concentrates on one body and has to survive the wait,
// so it is paid in neither. Time and breadth are real prices, and a better rate
// per stack is what they are owed. The ceiling is that it may not be worth twice
// the drip's rate — hoarding is then a choice rather than the answer.
//
// The rule stays comparative, because nothing on a nuke's own declaration says how
// big a pile it will find. What can be said is how the two exchange rates sit
// against each other, and that is checkable across the shipped book without
// inventing a number.
func TestANukePaysBetterPerStackAndWorsePerTurn(t *testing.T) {
	book, statuses := mustSkills(t), mustStatuses(t)
	worstDrip := map[string]int{}
	nukes := map[string][]string{}
	for _, current := range book.Skills() {
		if !current.Requires.Arcs() {
			continue
		}
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil || kind.Category != status.Charge {
			continue
		}
		switch current.Requires.ConsumeStacks {
		case 1:
			if held, seen := worstDrip[kind.ID]; !seen || current.Requires.ArcPower < held {
				worstDrip[kind.ID] = current.Requires.ArcPower
			}
		case 0:
			nukes[kind.ID] = append(nukes[kind.ID], current.ID)
		}
	}
	if len(nukes) == 0 {
		t.Skip("nothing takes a whole pile, so there is no rate to compare")
	}
	for id, ids := range nukes {
		floor, known := worstDrip[id]
		if !known {
			t.Errorf("%q is taken by the pile and by nothing a stack at a time, so there is no drip for %v to be a worse deal than: the comparison this rule rests on does not exist",
				id, ids)
			continue
		}
		for _, one := range ids {
			current, err := book.Lookup(one)
			if err != nil {
				t.Errorf("skill %q: %v", one, err)
				continue
			}
			switch {
			case current.Requires.ArcPower <= floor:
				t.Errorf("skill %q arcs %d a stack of %q against the %d a drip gets, and it also waits and hits one body: it is worse on every axis, so nobody brings it",
					one, current.Requires.ArcPower, id, floor)
			case current.Requires.ArcPower > floor*2:
				t.Errorf("skill %q arcs %d a stack of %q against a drip's %d, over twice the rate: hoarding stops being a choice and becomes the answer",
					one, current.Requires.ArcPower, id, floor)
			}
			// And the cooldown, which is the other half of "worse per turn". A nuke
			// on a drip's cooldown would be a drip that also waits, which is the
			// same skill with the waiting added for nothing.
			for _, other := range book.Skills() {
				if other.Requires.Arcs() && other.Requires.ConsumeStacks == 1 &&
					other.Requires.Status == id && current.Cooldown <= other.Cooldown {
					t.Errorf("skill %q takes the pile on a %d-turn cooldown against %q's %d: it gets the better rate AND the same tempo",
						one, current.Cooldown, other.ID, other.Cooldown)
				}
			}
			t.Logf("%s arcs %d a stack against the drip's %d, takes the pile, and waits %d turns",
				one, current.Requires.ArcPower, floor, current.Cooldown)
		}
	}
}

// TestARepeatingStrikeIsBoundedAtBothEnds holds the shape of the other new roll.
//
// A rolled strike count is a distribution rather than a number, and both ends of
// it are a way to get it wrong. Unbounded, a skill can take a whole battle in one
// turn however rarely — and "however rarely" is not a bound. Bounded too tightly
// against its own floor, the roll is a fixed count wearing a distribution's
// clothes, and every figure derived from it is the mean of one outcome.
//
// The mean is what is held, because it is what everything outside the roll reads:
// a rating using the floor never picks the skill and one using the ceiling picks
// nothing else.
func TestARepeatingStrikeIsBoundedAtBothEnds(t *testing.T) {
	found := 0
	for _, current := range mustSkills(t).Skills() {
		if !current.Repeats() {
			continue
		}
		found++
		mean := current.ExpectedStrikes()
		floor := current.StrikeCount() * scale.Base
		ceiling := current.MaxStrikes * scale.Base
		t.Logf("%s: %d strike at worst, %d.%03d on average, %d at best",
			current.ID, current.StrikeCount(), mean/scale.Base, mean%scale.Base, current.MaxStrikes)
		if mean <= floor {
			t.Errorf("skill %q repeats at %d per mille and still averages its own floor of %d strikes: the roll decides nothing",
				current.ID, current.Repeat, current.StrikeCount())
		}
		if mean >= ceiling {
			t.Errorf("skill %q averages %d per mille strikes against a cap of %d, so the cap is the ordinary case rather than the tail",
				current.ID, mean, current.MaxStrikes)
		}
		// The tail has to stay a tail. A repeat over half grows the count faster
		// than it stops it, and the cap rather than the odds becomes what the
		// skill is worth — a ceiling doing a distribution's job.
		if current.Repeat > scale.Base/2 {
			t.Errorf("skill %q repeats at %d per mille, over half: the cap would decide its damage rather than the roll",
				current.ID, current.Repeat)
		}
		// And it still divides its power, which is the trade every other
		// multi-strike skill here makes.
		if total := current.Power * mean / scale.Base; total > 3*scale.Base {
			t.Errorf("skill %q averages %d power in all, which is a whole battle's worth of damage on one turn's cooldown",
				current.ID, total)
		}
	}
	if found == 0 {
		t.Skip("no shipped skill repeats, so there is nothing here to hold")
	}
}

// TestACounterCarriesNoEffectOfItsOwn holds the argument its own stack cap rests
// on, over the shipped book rather than at the parser.
//
// status.ParseBook refuses a charge carrying a modifier, and this is the same
// claim asked of the data: the reason a counter may stack to nine hundred and
// ninety nine while everything else stops at five is that nine hundred and
// ninety nine of it does nothing. A charge that ticked, cut healing or moved a
// stat would be an effect with no bound at all, and the cap would have gone from
// a decision to an oversight.
func TestACounterCarriesNoEffectOfItsOwn(t *testing.T) {
	statuses := mustStatuses(t)
	counted := 0
	for _, kind := range statuses.Kinds() {
		if kind.Category != status.Charge {
			continue
		}
		counted++
		if kind.MaxStacks <= statuses.MaxStacks {
			t.Errorf("status %q caps at %d stacks, which the ordinary limit of %d already allows: it needs no category of its own",
				kind.ID, kind.MaxStacks, statuses.MaxStacks)
		}
		if len(kind.Modifiers) != 0 || kind.TickPower != 0 || kind.HealShare != 0 {
			t.Errorf("status %q is a counter that also does something: %d modifier(s), %d tick power, %d heal share",
				kind.ID, len(kind.Modifiers), kind.TickPower, kind.HealShare)
		}
		if kind.Permanent {
			// Remove refuses a permanent status, which is what keeps a trait from
			// being dispelled — and it is Remove that a consume goes through. A
			// permanent counter would therefore be a counter nothing could ever
			// spend, accumulating for ever and paying out never.
			t.Errorf("status %q is a permanent counter, so nothing can ever spend it", kind.ID)
		}
	}
	if counted == 0 {
		t.Skip("no counter is shipped, so there is nothing here to hold")
	}
}

// TestTheChainStopsAtTheMidline is the rule the chain nearly broke, and the one
// it is the only shape on this board that had to be told.
//
// Every other shape is a pattern, and pattern.Targets already drops a splash cell
// landing on the far side — Side.CrossesSides is the single thing that lifts it,
// and only a skill declaring `all` has it. A chain reads the board instead of a
// pattern, so it obeyed none of that: aimed at an enemy standing next to a
// charged teammate, the current walked straight back across the midline and took
// two of that teammate's stacks along with 272 of its health. Nothing arranges
// that on purpose; the *enemy's* own chargers arrange it for free.
//
// So the board here is the one that found it. `mate` stands at 2,1, which is a
// hex neighbour of the enemy front cell 3,1, and everybody is charged by hand —
// what is being measured is where the current may travel, not who can be
// persuaded to charge whom.
func TestTheChainStopsAtTheMidline(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	kind, err := books.Statuses.Lookup("charge")
	if err != nil {
		t.Fatalf("look up the counter: %v", err)
	}
	roster := []battle.Roster{
		{ID: "mag", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: mustAffinity(t, "electric"), Stats: benchStats(3000, 700, 300, 200),
			Skills: []string{"charge_beam", "electro_ball"}},
		{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "metal"), Stats: benchStats(4800, 300, 300, 1),
			Skills: []string{"body_slam"}},
		{ID: "foe", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "metal"), Stats: benchStats(4800, 300, 300, 1),
			Skills: []string{"body_slam"}},
	}
	sort.Slice(roster, func(a, b int) bool { return roster[a].ID < roster[b].ID })
	fight, err := battle.New(books, 3, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	mate, _ := fight.Unit("mate")
	foe, _ := fight.Unit("foe")
	// The fixture authors its own condition rather than hoping the board produces
	// one: if these two ever stop being neighbours the test measures nothing, and
	// this is what says so out loud.
	if !slices.Contains(mate.Cell.NeighborsOnBoard(), foe.Cell) {
		t.Fatalf("%v and %v are not neighbours, so no chain could have crossed anyway",
			mate.Cell, foe.Cell)
	}
	if mate.Cell.Side() == foe.Cell.Side() {
		t.Fatalf("both cells are on the %s side, so there is no midline between them", mate.Cell.Side())
	}
	for _, unit := range []*battle.Unit{mate, foe} {
		for range 4 {
			unit.Statuses.Apply(kind, 0)
		}
	}

	before := mate.HP
	held := mate.Statuses.Stacks(kind.ID)
	for range 40 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "mag" {
			if !prompt.Skipped {
				_ = fight.Pass("not the caster")
			}
			fight.Drain()
			continue
		}
		if prompt.Skipped {
			fight.Drain()
			continue
		}
		if err := fight.Act("electro_ball", foe.Cell); err != nil {
			t.Fatalf("electro_ball: %v", err)
		}
		break
	}
	// It reached the enemy, or the assertions below are about a cast that did
	// nothing at all.
	if foe.Statuses.Stacks(kind.ID) == held {
		t.Fatalf("the conduit spent none of the enemy's %d stacks, so nothing here measured a discharge", held)
	}
	if mate.HP != before {
		t.Errorf("the caster's own teammate lost %d health to a skill aimed at the enemy: the chain crossed the midline",
			before-mate.HP)
	}
	if left := mate.Statuses.Stacks(kind.ID); left != held {
		t.Errorf("the caster's own teammate went from %d stacks to %d: the chain spent a counter on its own side",
			held, left)
	}
}
