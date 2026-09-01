package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
)

// TestABarrierEatsDamageWhereAShieldEatsStrikes is the whole reason the category
// exists, and it is asserted as the two guards *against each other* rather than
// as one of them against nothing.
//
// A block charge cancels a strike, whole: three small strikes spend three
// charges, and one heavy strike spends one. A pool eats damage: three small
// strikes and one heavy one of the same total drain it by the same amount. So a
// wall of charges is the answer to a heavy blow and multi-strike is the answer to
// it, while a barrier does not care — and a game with only the first has only one
// kind of wall.
//
// ⚠️ **This is what makes it a second shape rather than a smoother version of the
// first.** If the two ever came out the same way against both kinds of attacker,
// one of them would be redundant, and the assertion below is that they do not.
func TestABarrierEatsDamageWhereAShieldEatsStrikes(t *testing.T) {
	// Two attackers of equal total power, one landing it whole and one splitting
	// it three ways. The fixture book's `triple` divides its power across the
	// strikes rather than repeating it, so the totals really are level.
	for _, guard := range []struct {
		name string
		set  func(*testing.T, *battle.Battle, *battle.Unit)
	}{
		{"a wall of charges", charge},
		{"a barrier", barrier},
	} {
		for _, attack := range []struct {
			name  string
			skill string
		}{{"one heavy strike", "heavy"}, {"three light ones", "triple"}} {
			dealt := swing(t, attack.skill, guard.set)
			t.Logf("%-20s vs %-18s took %d", attack.name, guard.name, dealt)
		}
	}

	heavyOnCharges := swing(t, "heavy", charge)
	lightOnCharges := swing(t, "triple", charge)
	heavyOnBarrier := swing(t, "heavy", barrier)
	lightOnBarrier := swing(t, "triple", barrier)

	// A wall of charges is beaten by arriving in pieces: the same total, split
	// three ways, gets through where the single blow did not.
	if lightOnCharges <= heavyOnCharges {
		t.Errorf("against charges a split volley dealt %d and one blow dealt %d: multi-strike is supposed to be the answer to a wall",
			lightOnCharges, heavyOnCharges)
	}
	// A barrier does not care how the damage arrives. ⚠️ Both figures have to be
	// something: a pool large enough to cover either would report the two as
	// equal at nought, which is a green test measuring a guard that was never
	// asked a question. The pool below is deliberately smaller than the volley.
	if lightOnBarrier == 0 || heavyOnBarrier == 0 {
		t.Fatalf("the barrier covered both volleys whole (%d and %d), so nothing here compared anything",
			lightOnBarrier, heavyOnBarrier)
	}
	if lightOnBarrier != heavyOnBarrier {
		t.Errorf("against a barrier a split volley dealt %d and one blow dealt %d: a pool is supposed to be indifferent to the strike count",
			lightOnBarrier, heavyOnBarrier)
	}
}

// TestABarrierSpillsWhatItCannotCover is the other half of the shape: a pool is a
// quantity, so a blow larger than what is left is not cancelled — the remainder
// goes through. That is what a charge can never do, and it is why a barrier is
// not simply a better wall.
func TestABarrierSpillsWhatItCannotCover(t *testing.T) {
	spilled := swing(t, "clout", func(t *testing.T, fight *battle.Battle, unit *battle.Unit) {
		t.Helper()
		apply(t, fight, unit, "aegis")
		// Most of it already spent, so the blow is bigger than what is left.
		unit.Statuses.SpendPool(status.Absorb, unit.Statuses.PoolIn(status.Absorb)-1)
	})
	whole := swing(t, "clout", barrier)
	unguarded := swing(t, "clout", nothing)
	t.Logf("no guard %d, a spent barrier %d, a fresh one %d", unguarded, spilled, whole)
	if spilled == 0 {
		t.Errorf("a barrier with a point left stopped the blow entirely, which is what a charge does")
	}
	if spilled >= unguarded {
		t.Errorf("a barrier with a point left took %d against %d unguarded: it ate nothing", spilled, unguarded)
	}
	if whole != 0 {
		t.Errorf("a fresh barrier let %d through, so the pool is smaller than one blow and nothing above is measuring a spill", whole)
	}
}

// TestNothingStopsAnUnblockableBlow is the counter to both guards, and it is
// checked against both because the flag names both: a skill that only got past
// one of them would be a flag that does half of what it says.
//
// ⚠️ It is also checked to leave the guard **standing**. An unblockable blow is
// not a dispel: it goes past the barrier rather than through it, so the next
// attacker still meets a full one. Spending what it ignores would make one skill
// quietly the strongest strip in the game.
func TestNothingStopsAnUnblockableBlow(t *testing.T) {
	for _, guard := range []struct {
		name string
		set  func(*testing.T, *battle.Battle, *battle.Unit)
	}{{"a wall of charges", charge}, {"a barrier", barrier}} {
		stopped := swing(t, "clout", guard.set)
		through := swing(t, "unstoppable", guard.set)
		t.Logf("%-20s stops an ordinary blow at %d, an unstoppable one at %d",
			guard.name, stopped, through)
		if stopped != 0 {
			t.Fatalf("%s let %d through from an ordinary blow, so it is not guarding and nothing below is measured",
				guard.name, stopped)
		}
		if through == 0 {
			t.Errorf("%s stopped the unstoppable blow", guard.name)
		}
	}

	// And what it ignored is still there afterwards.
	fight := arena(t)
	target, _ := fight.Unit("them")
	barrier(t, fight, target)
	before := target.Statuses.PoolIn(status.Absorb)
	charge(t, fight, target)
	charges := target.Statuses.Stacks("block")
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("unstoppable", target.Cell); err != nil {
		t.Fatalf("act: %v", err)
	}
	if got := target.Statuses.PoolIn(status.Absorb); got != before {
		t.Errorf("the barrier went from %d to %d: an unstoppable blow goes past a guard rather than spending it",
			before, got)
	}
	if got := target.Statuses.Stacks("block"); got != charges {
		t.Errorf("the charges went from %d to %d: an unstoppable blow spends nothing",
			charges, got)
	}
}

// TestABarrierIsSpentBeforeHealthAndSaysSoInTheLog holds the ordering the log
// contract rests on: a reader watching a strike land for its full figure and the
// target's health not move has nothing to account for it, so the guard's line
// comes first and carries what it ate and what is left.
func TestABarrierIsSpentBeforeHealthAndSaysSoInTheLog(t *testing.T) {
	fight := arena(t)
	target, _ := fight.Unit("them")
	barrier(t, fight, target)
	pool := target.Statuses.PoolIn(status.Absorb)
	health := target.HP
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("clout", target.Cell); err != nil {
		t.Fatalf("act: %v", err)
	}
	var soaked, hurt int
	var ate, left int64
	for _, event := range fight.Drain() {
		switch event.Kind {
		case battle.Absorbed:
			if hurt > 0 {
				t.Error("the barrier's line came after the damage it was spent instead of")
			}
			soaked++
			ate, left = event.Amount, event.Remaining
		case battle.Damaged:
			hurt++
		}
	}
	if soaked != 1 {
		t.Fatalf("the log carries %d absorbed lines for one strike", soaked)
	}
	if ate <= 0 {
		t.Errorf("the log says the barrier ate %d", ate)
	}
	if want := pool - ate; left != want {
		t.Errorf("the log says %d left in the barrier, and %d were spent out of %d",
			left, ate, pool)
	}
	if got := target.Statuses.PoolIn(status.Absorb); got != left {
		t.Errorf("the log says %d left and the unit carries %d", left, got)
	}
	if target.HP != health {
		t.Errorf("health moved from %d to %d under a barrier that covered the blow",
			health, target.HP)
	}
}

// arena is one attacker and one target, the target slow enough that the attacker
// always acts first.
func arena(t *testing.T) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 5, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"clout", "heavy", "triple", "unstoppable"}},
		{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
			Skills: []string{"clout"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// swing puts a guard on the target, throws one skill at it, and returns the
// health it actually lost.
func swing(t *testing.T, id string, guard func(*testing.T, *battle.Battle, *battle.Unit)) int64 {
	t.Helper()
	fight := arena(t)
	target, known := fight.Unit("them")
	if !known {
		t.Fatal("no target on the board")
	}
	guard(t, fight, target)
	before := target.HP
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act(id, target.Cell); err != nil {
		t.Fatalf("act %s: %v", id, err)
	}
	return before - target.HP
}

func apply(t *testing.T, fight *battle.Battle, unit *battle.Unit, id string) {
	t.Helper()
	kind, err := fight.Books().Statuses.Lookup(id)
	if err != nil {
		t.Fatalf("lookup %s: %v", id, err)
	}
	unit.Statuses.Apply(kind, pool)
}

// pool is what a barrier in these tests holds, and it is chosen against the
// arena rather than picked large: an attacker at 500 scaling into 300 defence
// lands `heavy` and `triple` for three hundred apiece, so two hundred covers a
// single `clout` whole and leaves a third of the heavier volleys to spill. A pool
// nothing could exhaust would make every comparison here come back level at
// nought.
const pool = 200

func nothing(*testing.T, *battle.Battle, *battle.Unit) {}

func barrier(t *testing.T, fight *battle.Battle, unit *battle.Unit) {
	t.Helper()
	apply(t, fight, unit, "aegis")
}

func charge(t *testing.T, fight *battle.Battle, unit *battle.Unit) {
	t.Helper()
	// Two rather than three, and the count is the measurement: `triple` lands
	// three times, so a wall of two charges is one a split volley gets through
	// and a single blow does not. Three charges would stop both and report the
	// two guards as identical.
	for range 2 {
		apply(t, fight, unit, "block")
	}
}

// TestAnAbsorbIsWorthCasting is the half a mechanic dies without, and the lesson
// is one this repository has already paid for once: `dispelled` sat in the rating
// for a long time pricing a shield at nothing, so an opponent handed a dispel
// declined it and the branch was dead. A guard nobody rates is data.
//
// So the rating is asked directly. Against something that can actually hurt, a
// unit holding a barrier and a poke should put the barrier up; with the barrier
// already at its cap there is nothing left to gain and it should swing.
func TestAnAbsorbIsWorthCasting(t *testing.T) {
	for _, row := range []struct {
		name   string
		warded int
		want   string
	}{
		{"nothing up yet", 0, "ward"},
		// ⚠️ The row that makes the first one mean something. Apply refuses a
		// stack past the cap, so the hypothetical the rating builds is identical
		// to the unit as it stands and the guard is correctly worth nought — the
		// same term that stops two units buffing each other for ever.
		{"already at the cap", 2, "jab"},
	} {
		t.Run(row.name, func(t *testing.T) {
			fight, err := battle.New(books(t), 5, []battle.Roster{
				{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
					Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
					Skills: []string{"ward", "jab"}},
				{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
					Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
					Skills: []string{"heavy"}},
			})
			if err != nil {
				t.Fatalf("new battle: %v", err)
			}
			fight.Begin()
			fight.Drain()
			mine, known := fight.Unit("me")
			if !known {
				t.Fatal("nobody on the board")
			}
			for range row.warded {
				apply(t, fight, mine, "aegis")
			}
			prompt, err := fight.Advance()
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			choice, found := fight.Suggest(prompt)
			if !found {
				t.Fatal("nothing was suggested, so there is no preference to read")
			}
			if choice.Skill != row.want {
				t.Errorf("the rating chose %q and should have chosen %q", choice.Skill, row.want)
			}
		})
	}
}
