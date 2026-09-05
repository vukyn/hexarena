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

// TestATraitSparesAHealthCostAtBothSites is the whole of what passive.Spares is
// for, and it is written as one test on purpose: the two halves are the two
// places a cost is read, and a trait applied to one of them is worse than a trait
// applied to neither.
//
// ⚠️ **Applied only at Battle.spendHealth, the trait makes a unit that never
// casts the skill it holds the trait for.** The rating still prices the full ask,
// so Suggest keeps declining `bloodprice` for its free twin — the holder pays
// nothing for a cast it never makes. That failure is invisible to a test that
// only reads health, which is what the second subtest exists to stop.
//
// ⚠️ **Applied only at pricing.spentHealth, the unit bleeds while believing it
// does not**, which is the same defect wearing the other face and is what the
// first subtest reads.
//
// The fixture is the one TestAPriceIsChargedForEverySkillAndNotOnlyAnAllSidedOne
// already stands on — `bloodprice` costs a fifth of maximum health, `clout` is
// its free twin — so "the rating prefers clout" is a claim this file has already
// measured without the trait.
func TestATraitSparesAHealthCostAtBothSites(t *testing.T) {
	priceWith := func(t *testing.T, traits ...string) *battle.Battle {
		t.Helper()
		fight, err := battle.New(books(t), 5, []battle.Roster{
			{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
				Skills: []string{"bloodprice", "clout"}, Passives: traits},
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
	// What the caster actually hands over, read off the log rather than the
	// health so that a cast which paid nothing is told apart from one that never
	// paid at all.
	paidFor := func(t *testing.T, fight *battle.Battle) (paid int64, charged bool) {
		t.Helper()
		them, _ := fight.Unit("them")
		cast(t, fight, "bloodprice", them.Cell)
		for _, event := range fight.Drain() {
			if event.Kind == battle.Paid {
				return event.Amount, true
			}
		}
		return 0, false
	}

	t.Run("the charge", func(t *testing.T) {
		// The control, and it is the reason the figures below mean anything: the
		// same skill on the same frame with no trait at all.
		full, charged := paidFor(t, priceWith(t))
		if !charged || full <= 0 {
			t.Fatalf("the untrained caster paid %d (charged %v), so there is no price to spare",
				full, charged)
		}
		half, charged := paidFor(t, priceWith(t, "halfbilled"))
		if !charged {
			t.Fatal("a half-spared cast was not charged at all, so the share was not applied")
		}
		if want := full / 2; half != want {
			t.Errorf("a trait sparing half a cost left a price of %d, want %d of %d",
				half, want, full)
		}
		// A trait sparing the whole of it charges nothing, which reaches
		// spendHealth's own `paid <= 0` return and emits no event — the absence
		// is the assertion.
		if paid, charged := paidFor(t, priceWith(t, "unbilled")); charged || paid != 0 {
			t.Errorf("a fully spared cast still charged %d (charged %v)", paid, charged)
		}
	})

	t.Run("the rating", func(t *testing.T) {
		chosen := func(t *testing.T, fight *battle.Battle) string {
			t.Helper()
			prompt, err := fight.Advance()
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			choice, found := fight.Suggest(prompt)
			if !found {
				t.Fatal("nothing was suggested, so there is no preference to read")
			}
			return choice.Skill
		}
		// The premise, held rather than assumed: without the trait the rating
		// declines the costly skill. If it stopped doing that the subtest below
		// would pass on a rating that never read a price at all.
		if got := chosen(t, priceWith(t)); got != "clout" {
			t.Fatalf("with no trait the rating chose %q, and this subtest is about a rating "+
				"that declines a price", got)
		}
		if got := chosen(t, priceWith(t, "unbilled")); got != "bloodprice" {
			t.Errorf("the caster pays nothing for %q and the rating still chose %q: the sparing "+
				"reached the charge and not the price", "bloodprice", got)
		}
	})
}

// TestASplitMovesMaximumHealthIntoTheCopy is the whole of skill.Summon.Splits,
// and it is written against the one property that separates it from a cost: what
// leaves the caster does not come back.
//
// ⚠️ **A cost and a split look identical on the health bar and are different
// creatures underneath.** Battle.spendHealth takes health and leaves the maximum
// alone, so a mender undoes it; this takes the maximum, so nothing does. A test
// reading only HP would pass on an implementation that charged the caster and
// handed the copy a share of nothing.
func TestASplitMovesMaximumHealthIntoTheCopy(t *testing.T) {
	fight := splitting(t)
	mine, _ := fight.Unit("me")
	wasMax, wasHP := mine.MaxHP(), mine.HP
	cast(t, fight, "split", mine.Cell)

	moved := int64(0)
	copies := make([]string, 0, 2)
	for _, event := range fight.Drain() {
		switch event.Kind {
		case battle.Split:
			moved = event.Amount
			if event.Remaining != mine.MaxHP() {
				t.Errorf("the log says the caster's maximum is now %d and it is %d",
					event.Remaining, mine.MaxHP())
			}
		case battle.Summoned:
			copies = append(copies, event.Target)
		}
	}
	if moved <= 0 {
		t.Fatal("nothing moved, so there is no split to measure")
	}
	if want := wasMax * 3 / 10; moved != want {
		t.Errorf("a split of three tenths moved %d of a maximum of %d, want %d",
			moved, wasMax, want)
	}
	// The maximum, which is the claim. A caster whose maximum was untouched has
	// paid a cost rather than split, however its current health reads.
	if got, want := mine.MaxHP(), wasMax-moved; got != want {
		t.Errorf("the caster's maximum is %d after giving away %d of %d, want %d",
			got, moved, wasMax, want)
	}
	// The current health follows it down rather than being clamped to it, so a
	// wounded caster cannot split out of its own headroom for free.
	if got, want := mine.HP, wasHP-moved; got != want {
		t.Errorf("the caster is on %d after giving away %d of %d, want %d",
			got, moved, wasHP, want)
	}
	if len(copies) != 1 {
		t.Fatalf("the split put down %d units, want one", len(copies))
	}
	copied, standing := fight.Unit(copies[0])
	if !standing {
		t.Fatal("the copy is not on the board")
	}
	if copied.MaxHP() != moved {
		t.Errorf("the copy holds %d health and %d left the caster", copied.MaxHP(), moved)
	}
	// ⚠️ Named apart from the health above, because the two come from different
	// halves of the declaration and only one of them is the split: `share: 900`
	// sized every other stat and `splits: 300` overrode the health. An
	// implementation that let the split size everything would leave the copy on
	// three tenths of the attack, which no assertion about health could see.
	if got, want := copied.Base[progression.Attack], mine.Base[progression.Attack]*9/10; got != want {
		t.Errorf("the copy attacks off %d, want %d — nine tenths of its maker's, which is what "+
			"the share says and what the split does not touch", got, want)
	}
}

// TestASplitIsCappedForTheWholeBattle is skill.Condition.BelowStacks, and it is
// the first bound in this engine on how many times a skill may be used at all.
//
// ⚠️ **A cooldown cannot express this.** It counts the caster's turns and so
// bounds the *rate*; a battle long enough gives any cooldown as many casts as it
// likes. A permanent status cannot either — status.Set.Remove refuses one on
// purpose, so a budget granted by a trait could never be drawn down. What is left
// is a counter the skill itself applies and a gate that reads it, which is what
// this measures.
func TestASplitIsCappedForTheWholeBattle(t *testing.T) {
	fight := splitting(t)
	mine, _ := fight.Unit("me")
	// ⚠️ **Advance until the CASTER is up, rather than reading whatever prompt
	// comes back.** The copies act too, and they are fast — nine tenths of their
	// maker's speed — so the third prompt in this battle belongs to a copy and
	// not to the caster. Written the naive way this test passed with the cap
	// disabled entirely: a copy knows only `jab`, so "the split is not offered"
	// was true of the copy and said nothing at all about the caster. Measured, so
	// it is a fixture that hid a branch rather than a worry.
	upNext := func(t *testing.T) *battle.Prompt {
		t.Helper()
		for range 20 {
			prompt, err := fight.Advance()
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			if prompt.Unit == mine.ID {
				return prompt
			}
			choice, ok := fight.Suggest(prompt)
			if !ok {
				t.Fatalf("%s has nothing to do, and this fixture needs every other unit to "+
					"take its turn so the caster comes back up", prompt.Unit)
			}
			if err := fight.Act(choice.Skill, choice.Aim); err != nil {
				t.Fatalf("%s acts: %v", prompt.Unit, err)
			}
			fight.Drain()
		}
		t.Fatal("the caster never came up again")
		return nil
	}
	offered := func(prompt *battle.Prompt) bool {
		for _, option := range prompt.Options {
			if option.Skill == "split" && option.Blocked == battle.BlockNone {
				return true
			}
		}
		return false
	}

	// Two casts, and the cooldown is nought so nothing but the cap stands between
	// the caster and a third.
	for spent := range 2 {
		prompt := upNext(t)
		if !offered(prompt) {
			t.Fatalf("the split is not offered for cast %d of two", spent+1)
		}
		if err := fight.Act("split", mine.Cell); err != nil {
			t.Fatalf("split %d: %v", spent+1, err)
		}
		fight.Drain()
	}
	// The third is refused, and refused for the stated reason rather than for
	// having run out of somewhere to stand.
	prompt := upNext(t)
	for _, option := range prompt.Options {
		if option.Skill != "split" {
			continue
		}
		if option.Blocked == battle.BlockNone {
			t.Error("a third split is on offer: two a battle and no more")
		}
		// ⚠️ Its own refusal rather than the fuel one, and the difference is what
		// a player does next: short of fuel you wait and fill the tank, spent out
		// you are finished with the skill for the battle. One wording for both
		// would tell somebody to wait for something that is never coming.
		if option.Blocked != battle.BlockSpent {
			t.Errorf("the third split is refused as %v, want the spent-out refusal", option.Blocked)
		}
		if option.Need != 2 || option.Held != 2 {
			t.Errorf("the refusal says %d of %d spent, want 2 of 2 so a screen can draw it",
				option.Held, option.Need)
		}
		return
	}
	t.Error("the split is not among the caster's options at all, so nothing here says why")
}

// splitting is a caster holding a gated, capped split beside a plain attack, and
// an opponent that cannot reach it — so the turns it takes are its own.
func splitting(t *testing.T) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "me", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: []string{"split", "clout"}},
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

// TestABurrowedUnitCannotBeAimedAtButIsStillSplashed is the whole of the hiding,
// and the three clauses are written as one test because the mechanic is the
// difference between them.
//
// `aims` is the cells a skill may be POINTED at; `covers` is the cells the shape
// then catches. Only the first is touched, and that gap is what makes hiding a
// decision rather than a wall: a unit underground is safe from anything thrown at
// it and not from anything thrown next to it.
//
// ⚠️ **The status half is refused rather than resisted.** A resistance is
// declared per status and this is every status there is, so it lands at inflict
// as a refusal — which is also why the caster's OWN applications still work: a
// hidden unit may still buff itself, and a trait renewing something on its holder
// is the holder's own doing.
func TestABurrowedUnitCannotBeAimedAtButIsStillSplashed(t *testing.T) {
	fight, err := battle.New(books(t), 11, []battle.Roster{
		{ID: "hitter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 50),
			Skills: []string{"arc", "daze"}},
		{ID: "exposed", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
			Skills: []string{"jab"}},
		{ID: "hidden", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 60),
			Skills: []string{"burrow", "jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	// The hidden unit is fastest, so it goes under before anything is thrown.
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "hidden" {
		t.Fatalf("%s is up first and this test needs the hider to be, or it measures a unit "+
			"that was never underground", prompt.Unit)
	}
	hidden, _ := fight.Unit("hidden")
	if err := fight.Act("burrow", hidden.Cell); err != nil {
		t.Fatalf("burrow: %v", err)
	}
	fight.Drain()
	if hidden.Statuses.Stacks("burrowed") == 0 {
		t.Fatal("the hider is not underground, so nothing below measures the hiding")
	}

	prompt, err = fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "hitter" {
		t.Fatalf("%s is up and this test needs the attacker to be", prompt.Unit)
	}

	// One: it is off the list of things a skill may be pointed at.
	for _, option := range prompt.Options {
		for _, aim := range option.Aims {
			if aim == hidden.Cell {
				t.Errorf("%q may still be aimed at the hider's cell %v", option.Skill, aim)
			}
		}
	}
	// ⚠️ The premise, held rather than assumed: the OTHER enemy is still
	// aimable. Without it an aims() that returned nothing at all would pass every
	// assertion above.
	exposed, _ := fight.Unit("exposed")
	aimable := false
	for _, option := range prompt.Options {
		if option.Skill != "arc" {
			continue
		}
		for _, aim := range option.Aims {
			if aim == exposed.Cell {
				aimable = true
			}
		}
	}
	if !aimable {
		t.Fatal("the attacker cannot aim at the unhidden enemy either, so the assertions above " +
			"would hold on an engine that offered no aims at all")
	}

	// Two and three: aimed at the neighbour, the shape still catches the hider —
	// and the rider that rode in with it does not stick.
	wasHP := hidden.HP
	if err := fight.Act("arc", exposed.Cell); err != nil {
		t.Fatalf("arc: %v", err)
	}
	refused := 0
	for _, event := range fight.Drain() {
		if event.Kind == battle.StatusResisted && event.Target == "hidden" {
			refused++
		}
	}
	if hidden.HP >= wasHP {
		t.Errorf("the hider is on %d of %d after a blast landed beside it: splash is what "+
			"hiding does not stop", hidden.HP, wasHP)
	}
	if hidden.Statuses.Stacks("poison") != 0 {
		t.Error("the hider took the poison that rode in with the blast")
	}
	if refused == 0 {
		t.Error("nothing in the log says the hider refused the status, so a reader cannot tell " +
			"a refusal from a roll that missed")
	}
	// The neighbour it was aimed at took both, which is what says the skill
	// itself works and the hiding is the only reason the hider did not.
	if exposed.Statuses.Stacks("poison") == 0 {
		t.Error("the unhidden enemy took no poison either, so the refusal above proves nothing")
	}
}
