package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/status"
)

// TestAPriceIsChargedForEverySkillAndNotOnlyAnAllSidedOne is the bug this
// shipped with, kept as a test because the shape of it is worth more than the
// fix.
//
// A skill's health cost is a cost of *acting*, so it was first added inside
// friendlyFire — which is where every other cost of acting lives. But rate calls
// that arm only when `declared.Target == skill.All`, so a single-target skill
// charging a quarter of its caster's health was charged nothing whatsoever.
// Measured before the move: a Magnezone holding one cast it three times a
// battle, handed over seven tenths of itself and lost 120 of 120 duels, against
// 69-51 for the same kit without it.
//
// **A cost filed in a branch that does not run is a cost nobody pays**, and the
// rating had no way to say so — a skill that is free is simply a good skill.
func TestAPriceIsChargedForEverySkillAndNotOnlyAnAllSidedOne(t *testing.T) {
	// Two identical single-target skills, one of which charges for itself. A
	// rating that sees the price prefers the free one; a rating that does not
	// cannot tell them apart, and takes whichever comes first.
	fight := priced(t)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, found := fight.Suggest(prompt)
	if !found {
		t.Fatal("nothing was suggested, so there is no preference to read")
	}
	if choice.Skill != "clout" {
		t.Errorf("the rating chose %q over the identical skill that costs nothing", choice.Skill)
	}
}

// TestAPriceIsPaidUpFrontAndNeverLethal holds the three things that make a cost a
// cost rather than a recoil.
//
// ⚠️ **Paid whether or not anything lands.** A share taken out of the damage
// dealt would be free on a turn the skill missed, and a skill that is free when
// it fails has no decision in it. So the fixture below throws one that cannot
// possibly connect and checks the caster paid anyway.
//
// ⚠️ **Never the last point.** Suggest prices what a turn is worth to the board
// and has no term at all for "and then I am not here", so a cast that could be
// lethal would be rated as pure gain. Leaving a point standing keeps the question
// out of the rating entirely.
func TestAPriceIsPaidUpFrontAndNeverLethal(t *testing.T) {
	t.Run("paid on a miss", func(t *testing.T) {
		fight := priced(t)
		mine, _ := fight.Unit("me")
		them, _ := fight.Unit("them")
		// Nothing can land: the skill is authored at a low accuracy and the
		// target is given a dodge nothing gets through, so the volley is all
		// misses and the price is the only health that moves.
		them.Base[progression.Dodge] = 100000
		before := mine.HP
		cast(t, fight, "bloodprice", them.Cell)
		misses, landed, paid := 0, 0, int64(0)
		for _, event := range fight.Drain() {
			switch event.Kind {
			case battle.Missed:
				misses++
			case battle.Damaged:
				landed++
			case battle.Paid:
				paid = event.Amount
			}
		}
		// ⚠️ Asserted rather than only counted. The first version of this test
		// read the miss count and never checked it, on a skill authored at a
		// thousand accuracy — which Rules.Chance answers before it ever reads
		// dodge. So the volley landed every time and the test was green while
		// measuring the opposite of what it says.
		if misses == 0 || landed > 0 {
			t.Fatalf("%d strikes missed and %d landed: nothing here measured a cast that failed",
				misses, landed)
		}
		if paid <= 0 {
			t.Fatal("nothing was paid for a cast that charges for itself")
		}
		if got := before - mine.HP; got != paid {
			t.Errorf("the caster lost %d and the log says it paid %d", got, paid)
		}
	})

	t.Run("never the last point", func(t *testing.T) {
		fight := priced(t)
		mine, _ := fight.Unit("me")
		them, _ := fight.Unit("them")
		mine.HP = 1
		cast(t, fight, "bloodprice", them.Cell)
		if mine.HP < 1 {
			t.Errorf("the caster paid its way to %d health", mine.HP)
		}
		if mine.Dead {
			t.Error("the caster killed itself paying for a skill")
		}
	})
}

// TestAGrantedGuardIsSizedFromTheHolder is the trait side of a barrier: a grant
// used to be a switch, and this one carries a number.
//
// ⚠️ **Set.Hold applied a nought amount** on the argument that a permanent status
// can be neither a damage-over-time nor a regeneration and so has nothing to
// snapshot. That was true of every permanent status there was, and stopped being
// true the moment a guard could be permanent — a barrier granted that way parses,
// applies, appears in the log and stops nothing at all.
//
// The two shipped traits differ in one field, so this checks that the field is
// what decides: the same power off two different stats gives two different pools.
func TestAGrantedGuardIsSizedFromTheHolder(t *testing.T) {
	fight, err := battle.New(books(t), 3, []battle.Roster{
		{ID: "hitter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 200, 100),
			Skills: []string{"clout"}, Passives: []string{"projecting"}},
		{ID: "wall", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 200, 800, 100),
			Skills: []string{"clout"}, Passives: []string{"carapaced"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	hitter, _ := fight.Unit("hitter")
	wall, _ := fight.Unit("wall")
	fromAttack := hitter.Statuses.PoolIn(status.Absorb)
	fromDefence := wall.Statuses.PoolIn(status.Absorb)
	t.Logf("attack 800 grants %d; defence 800 grants %d", fromAttack, fromDefence)
	if fromAttack == 0 || fromDefence == 0 {
		t.Fatalf("a granted barrier holds %d and %d: a guard that stops nothing is a guard nobody can see is broken",
			fromAttack, fromDefence)
	}
	// The two units are mirror images — 800 in the stat each trait names and 200
	// in the other — so a grant reading the stat it declares gives them the same
	// pool, and one reading the wrong stat gives them different ones.
	if fromAttack != fromDefence {
		t.Errorf("the same power off attack gave %d and off defence gave %d, on units that mirror each other: one of the traits is reading the other's stat",
			fromAttack, fromDefence)
	}
	// And it is a share of the stat rather than a copy of it.
	if fromAttack >= 800 {
		t.Errorf("a barrier of %d off a stat of 800 is not a share of anything", fromAttack)
	}
}

// priced is two units and a caster holding one skill that charges for itself and
// one identical skill that does not.
func priced(t *testing.T) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 5, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"bloodprice", "clout"}},
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

func cast(t *testing.T, fight *battle.Battle, id string, aim hex.Offset) {
	t.Helper()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act(id, aim); err != nil {
		t.Fatalf("act %s: %v", id, err)
	}
}
