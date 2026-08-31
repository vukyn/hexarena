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
		arc := current.Requires.ArcPower
		switch {
		case damped <= 0:
			t.Errorf("skill %q arcs for %d and damps nothing, so the discharge is free and the counter is a bonus rather than a bargain",
				current.ID, arc)
		case arc <= damped:
			t.Errorf("skill %q gives up %d power to arc for %d, so spending a stack is worse than never having one",
				current.ID, damped, arc)
		case arc > damped*2:
			t.Errorf("skill %q gives up %d power to arc for %d, over twice as good: "+
				"a stack may double what the turn was worth, which is where a detonate stops too",
				current.ID, damped, arc)
		}
		if left := current.Requires.DampedPower(current.Power); left <= 0 {
			t.Errorf("skill %q damps its %d power to %d, so it never rolls a strike and never discharges at all",
				current.ID, current.Power, left)
		}
		// One stack per strike, always. Taking the pile is what a detonate does,
		// and it is the one thing that would make accumulating pointless: a
		// magazine emptied by its first shot is a magazine of one.
		if current.Requires.ConsumeStacks != 1 {
			t.Errorf("skill %q spends %d stacks of %q a strike; a conduit spends one, which is what makes a pile worth having",
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
