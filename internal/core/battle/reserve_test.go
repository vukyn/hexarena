package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// reserveBooks is the shared books with a counter its holder builds on itself
// and a kit that spends it three different ways.
//
// Its own status book rather than the shared one, because the shared one has no
// reserve and no counter cap — and a reserve is the one category whose whole
// point is a stack count the ordinary cap would never allow.
//
// `bank` and `share` are the two ways fuel arrives: on the caster, and on
// somebody else. `tap` spends a fixed one, `dump` spends what it can use, and
// `paid` and `free` are the same blow with and without a price, which is the pair
// the rating is read through.
func reserveBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6, "max_counter_stacks": 40,
	  "kinds": [
	    {"id": "fuel", "category": "reserve", "max_stacks": 40, "duration": 6},
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "haste", "category": "buff", "max_stacks": 2, "duration": 3,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 300}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	shared.Statuses = statuses
	skills, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"bank","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_applies":[{"status":"fuel","chance":1000,"stacks":3}]},
	  {"id":"share","element":"neutral","range":1,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"ally",
	   "applies":[{"status":"fuel","chance":1000,"stacks":3}]},
	  {"id":"tap","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":1,"consume":true,
	    "consume_stacks":1,"bonus_power":900}},
	  {"id":"dump","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":2,"consume":true,
	    "stack_power":200}},
	  {"id":"paid","element":"neutral","range":1,"pattern":"single",
	   "power":150,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":20,"consume":true,
	    "consume_stacks":20,"bonus_power":900}},
	  {"id":"free","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":20,"bonus_power":900}},
	  {"id":"deep","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":8,"consume":true,
	    "stack_power":200}},
	  {"id":"sip","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":8,"consume":true,
	    "consume_stacks":1,"bonus_power":900}},
	  {"id":"wash","element":"neutral","range":1,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "strips":{"categories":["reserve"],"stacks":40}},
	  {"id":"scrub","element":"neutral","range":1,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"ally",
	   "strips":{"categories":["dot"],"stacks":40}}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = skills
	return shared
}

// tank is a fixed board: one caster that spends, one enemy that does nothing
// worth reacting to.
func tank(t *testing.T, kit ...string) *battle.Battle {
	t.Helper()
	fight, err := battle.New(reserveBooks(t), 4, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 200),
			Skills: kit},
		{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 60, 300, 1),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// fuelled puts a given number of stacks on the caster by hand, which is what
// lets a test name the pile size it is measuring instead of casting `bank` a
// number of times and hoping the arithmetic came out where it meant to.
func fuelled(t *testing.T, fight *battle.Battle, stacks int) *battle.Unit {
	t.Helper()
	caster, known := fight.Unit("caster")
	if !known {
		t.Fatal("no caster on the board")
	}
	kind, err := fight.Books().Statuses.Lookup("fuel")
	if err != nil {
		t.Fatalf("lookup fuel: %v", err)
	}
	for range stacks {
		caster.Statuses.Apply(kind, 0)
	}
	if got := caster.Statuses.Stacks("fuel"); got != stacks {
		t.Fatalf("the caster holds %d fuel, want %d: the fixture cannot measure what it cannot bank", got, stacks)
	}
	return caster
}

// spendOnce throws one skill at the enemy and reports what it took off, along with
// the fuel the caster was left holding.
func spendOnce(t *testing.T, fight *battle.Battle, id string) (dealt int64, left int) {
	t.Helper()
	caster, _ := fight.Unit("caster")
	target, known := fight.Unit("them")
	if !known {
		t.Fatal("no target on the board")
	}
	before := target.HP
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act(id, target.Cell); err != nil {
		t.Fatalf("act %s: %v", id, err)
	}
	fight.Drain()
	return before - target.HP, caster.Statuses.Stacks("fuel")
}

// TestASpendTakesTheStacksItNamesAndNotThePile is the bug this change found, and
// it had been shipped since the day a caster's own condition could consume.
//
// ⚠️ Battle.spend called Set.Consume, which empties a status. So `consume_stacks`
// on a self_requires parsed, validated, round-tripped through the file and was
// then thrown away: whatever an author asked for, the whole pile went. Nothing
// noticed because nothing shipped ever spent anything of its own — the field was
// exercised only by a skill book written for a test, and that skill spent
// everything by design.
//
// A magazine that empties itself on the first shot is not a magazine, and it is
// the entire difference between the two shapes a spend comes in.
func TestASpendTakesTheStacksItNamesAndNotThePile(t *testing.T) {
	fight := tank(t, "tap")
	fuelled(t, fight, 5)
	dealt, left := spendOnce(t, fight, "tap")
	if dealt <= 0 {
		t.Fatalf("the spend landed %d damage, so nothing here measured a spend", dealt)
	}
	if left != 4 {
		t.Errorf("a spend of one stack off a pile of five left %d, want 4", left)
	}
}

// TestASpendPastTheCeilingLeavesTheRemainder is the other half of Takes, and it
// is a rule about STACKS rather than about power.
//
// MaxSpendPower bounds what one cast may buy, and the obvious way to apply it is
// to clamp the bonus. That would have the caster hand over a pile it did not use:
// a full reserve emptied into a capped blow would be worth less per stack than a
// small one, so the rating would prefer to spend at exactly the wrong moment —
// which is the shape of the clamp bug this engine shipped one field along, where
// a health price read through its own floor got cheapest as it got most fatal.
//
// So the clamp is on what is taken, the leftovers stay in the tank, and "spend
// all of it" means "all of it this skill can use".
func TestASpendPastTheCeilingLeavesTheRemainder(t *testing.T) {
	const ceiling = skill.MaxSpendPower / 200 // `dump` pays 200 a stack.
	fight := tank(t, "dump")
	fuelled(t, fight, ceiling+10)
	dealt, left := spendOnce(t, fight, "dump")
	if dealt <= 0 {
		t.Fatalf("the spend landed %d damage, so nothing here measured a spend", dealt)
	}
	if left != 10 {
		t.Errorf("a spend off a pile of %d left %d, want 10: the ceiling is meant to stop it TAKING what it cannot pay for",
			ceiling+10, left)
	}
	// And the blow is the ceiling rather than the pile: the same reading from the
	// other side, so a clamp moved onto the power could not pass both.
	capped, _ := spendOnce(t, tankAt(t, ceiling), "dump")
	if capped != dealt {
		t.Errorf("a pile of %d landed %d and a pile of %d landed %d: the two are the same spend and must land the same blow",
			ceiling, capped, ceiling+10, dealt)
	}
}

func tankAt(t *testing.T, stacks int) *battle.Battle {
	t.Helper()
	fight := tank(t, "dump")
	fuelled(t, fight, stacks)
	return fight
}

// TestADeeperReserveHitsHarder is the whole of what stack_power is for: a flat
// bonus makes "spend everything" a sentence with no arithmetic in it, because
// emptying nine hundred stacks pays exactly what emptying two does.
//
// ⚠️ Read as a strict ordering across three depths rather than as two figures
// being different. One comparison passes on a payment that only fires at all,
// and what is being asserted is that the payment SCALES.
func TestADeeperReserveHitsHarder(t *testing.T) {
	dealt := make([]int64, 0, 3)
	for _, held := range []int{2, 6, 12} {
		hit, left := spendOnce(t, tankAt(t, held), "dump")
		if left != 0 {
			t.Fatalf("a spend of everything off %d left %d behind", held, left)
		}
		dealt = append(dealt, hit)
		t.Logf("%2d stacks spent for %d damage", held, hit)
	}
	for i := 1; i < len(dealt); i++ {
		if dealt[i] <= dealt[i-1] {
			t.Errorf("the blow went %d then %d as the reserve deepened, which is a payment that does not scale",
				dealt[i-1], dealt[i])
		}
	}
}

// TestAReserveIsNotSomethingItsHolderWantsGone is the category's own predicate
// asserted as behaviour, and it is the one fact that stopped a reserve being a
// charge with a different name.
//
// A charge goes on an ENEMY and is Harmful, so the victim's side washes it off:
// `rinse` is a shipped cleanse a squad points at its own ally naming dot,
// stat_debuff and charge. A reserve is its holder's own fuel, so the same cleanse
// pointed at the same ally must leave it exactly where it is — and what takes it
// off is an enemy's dispel.
func TestAReserveIsNotSomethingItsHolderWantsGone(t *testing.T) {
	fight := tank(t, "scrub")
	caster := fuelled(t, fight, 6)
	poison, err := fight.Books().Statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup poison: %v", err)
	}
	caster.Statuses.Apply(poison, 100)

	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("scrub", caster.Cell); err != nil {
		t.Fatalf("scrub: %v", err)
	}
	fight.Drain()
	// The cleanse did something, so the silence below is a refusal rather than a
	// strip that never ran.
	if caster.Statuses.Has("poison") {
		t.Fatal("the cleanse left the poison on, so it never reached the caster")
	}
	if got := caster.Statuses.Stacks("fuel"); got != 6 {
		t.Errorf("a cleanse of debuffs left %d fuel, want 6: a reserve is not something its holder wants gone", got)
	}
	if status.Reserve.Harmful() {
		t.Error("Reserve reports itself harmful, so a cleanse aimed at debuffs would empty its holder's own tank")
	}
}

// TestSpendingTheReserveIsPaidFor is the term this engine has now shipped
// without three times, and each time in the same shape: a cost written into a
// branch the rating never reads.
//
// `dispelled` priced a stripped shield at nothing, `guarded` skipped the stack
// cap, and `Skill.Cost` sat inside the arm `rate` calls only for an all-sided
// skill — so a quarter of a caster's health was handed over for free. A reserve
// is the same trap one field along: SelfBonus is read inside `expected`, so the
// damage a spend BUYS was counted from the first line of this change, and the
// stacks it hands over were counted by nothing.
//
// The pair below is the same blow at two prices. `paid` hits harder and empties
// twenty stacks; `free` hits for less and empties nothing. Uncosted, the rating
// takes the bigger number every time — which is a unit that dumps a full tank on
// the opening turn for fifty per mille of power.
func TestSpendingTheReserveIsPaidFor(t *testing.T) {
	fight := tank(t, "paid", "free")
	fuelled(t, fight, 20)
	// ⚠️ The control is on the same board rather than asserted from the numbers:
	// `paid` really is the larger blow, so a rating that picks `free` can only be
	// pricing what the larger one costs.
	if choice := chosen(t, fight); choice.Skill != "free" {
		t.Errorf("Suggest picked %q with twenty stacks banked, want free: `paid` lands more and empties the tank, and a rating that cannot see the tank prefers it",
			choice.Skill)
	}
	// And with nothing to spend the preference reverses, which is what says the
	// figure above is a price rather than a dislike of one of the two skills.
	// Neither condition holds at nought stacks, so both land their own power and
	// `paid` is simply the bigger one.
	bare := tank(t, "paid", "free")
	if choice := chosen(t, bare); choice.Skill != "paid" {
		t.Errorf("Suggest picked %q with an empty tank, want paid: with nothing to spend there is nothing to charge for",
			choice.Skill)
	}
}

// TestFuelIsPricedByTheKitOfWhoeverHoldsIt is the rule that separates the two
// counters inside `spendable`, and getting it wrong makes one of them worth
// nothing.
//
// A CHARGE is laid on an enemy and cashed by hitting them, so anybody on the
// charger's side who carries a consumer counts — laying one down for a squadmate
// to discharge is the whole of what a support charger does. A RESERVE sits on its
// holder and buys the holder's own skills through self_requires, so a squadmate's
// kit is irrelevant: reading the side there would price fuel by what somebody
// else could have done with it.
//
// ⚠️ **The board is built so that a side-wide reading cannot pass it.** A
// companion holding the spender stands on both boards, so the caster's side
// always contains one — but that companion is already at the status cap, so
// fuelling it gains no stacks and is worth nothing whichever rule is in force.
// What changes between the rows is only whether the unit actually being offered
// the fuel can spend it. Without the capped companion the two rows would differ
// in whether the side held a consumer at all, and a side-reading rating would
// pass for the wrong reason.
func TestFuelIsPricedByTheKitOfWhoeverHoldsIt(t *testing.T) {
	for _, mate := range []struct {
		name string
		kit  []string
		want string
	}{
		{"an ally who can spend it", []string{"dump"}, "share"},
		{"an ally who cannot", []string{"jab"}, "jab"},
	} {
		fight, err := battle.New(reserveBooks(t), 4, []battle.Roster{
			{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 60, 300, 200),
				Skills: []string{"share", "jab"}},
			{ID: "companion", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
				Skills: []string{"dump"}},
			{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
				Affinity: single("neutral"), Stats: stats(4000, 500, 300, 1),
				Skills: mate.kit},
			{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 60, 300, 1),
				Skills: []string{"jab"}},
		})
		if err != nil {
			t.Fatalf("%s: new battle: %v", mate.name, err)
		}
		fight.Begin()
		fight.Drain()
		companion, known := fight.Unit("companion")
		if !known {
			t.Fatalf("%s: no companion on the board", mate.name)
		}
		kind, err := fight.Books().Statuses.Lookup("fuel")
		if err != nil {
			t.Fatalf("lookup fuel: %v", err)
		}
		for range kind.MaxStacks {
			companion.Statuses.Apply(kind, 0)
		}
		if got := companion.Statuses.Stacks("fuel"); got != kind.MaxStacks {
			t.Fatalf("%s: the companion holds %d fuel of a cap of %d, so it is still a profitable target and the board proves nothing",
				mate.name, got, kind.MaxStacks)
		}
		if choice := chosen(t, fight); choice.Skill != mate.want {
			t.Errorf("with %s, Suggest picked %q, want %q", mate.name, choice.Skill, mate.want)
		}
	}
}

// TestADispelOfAReserveIsWorthTaking is the counter-play, and it is the third
// strip term for the reason the first two exist.
//
// Everything else `dispelled` reads is a stat or a tick, so a category that
// changes neither comes off an enemy for a price of nothing — which is how a
// shipped dispel spent a long time rating a stripped shield at nought.
//
// ⚠️ **The first version of this test was green with `unfuelled` deleted, and
// that is the same trap this repository has now sprung twice.** `dispelled` also
// reads the enemy's best blow before and after the strip, and on a board where
// the spender IS the enemy's best blow that term sees the fuel by itself — so the
// test passed for a reason that had nothing to do with the term it named. The
// board below keeps the spender BELOW the enemy's plain attack at the depth it is
// fought at, so the best-blow term cannot move and what is left to price is the
// tank itself: casts the enemy will now never make, which is exactly what the
// other two strip terms count for a shield and a barrier.
func TestADispelOfAReserveIsWorthTaking(t *testing.T) {
	for _, board := range []struct {
		name string
		fuel int
		want string
	}{
		{"an enemy with something banked", 3, "wash"},
		{"an enemy with an empty tank", 0, "jab"},
	} {
		fight, err := battle.New(reserveBooks(t), 4, []battle.Roster{
			{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 60, 300, 200),
				Skills: []string{"wash", "jab"}},
			// `strike` is a thousand of power and `dump` is a hundred plus two
			// a stack, so at three stacks the spender is the smaller blow and
			// taking the fuel off moves the enemy's best attack by nothing.
			{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 500, 300, 1),
				Skills: []string{"strike", "dump"}},
		})
		if err != nil {
			t.Fatalf("%s: new battle: %v", board.name, err)
		}
		fight.Begin()
		fight.Drain()
		them, known := fight.Unit("them")
		if !known {
			t.Fatalf("%s: no enemy on the board", board.name)
		}
		kind, err := fight.Books().Statuses.Lookup("fuel")
		if err != nil {
			t.Fatalf("lookup fuel: %v", err)
		}
		for range board.fuel {
			them.Statuses.Apply(kind, 0)
		}
		if choice := chosen(t, fight); choice.Skill != board.want {
			t.Errorf("against %s, Suggest picked %q, want %q", board.name, choice.Skill, board.want)
		}
	}
}

// TestBankingIsWorthRepeatingUpToWhatTheKitCanCash is the one place the two
// counters are priced by different arithmetic, and the difference is a sentence
// taken from the charge's own comment.
//
// A pile of charges is worth far less than a stack times its height, because a
// conduit cashes ONE per blow: the second stack needs one more turn to go right
// than the first, so each is worth half the one before it. That reasoning is
// exact for a charge and false for a reserve — a reserve spender cashes a whole
// run at once, so the second stack goes off in the same cast as the first and
// pricing it at half prices the mechanic as if it were the one it was built to be
// the opposite of.
//
// So a reserve is flat up to what its HOLDER can cash in one go, and halved above
// that, where a second cast really is speculation.
//
// ⚠️ **Measured before it was written.** With the halving in force, the shipped
// fire loop banked 456 stacks over forty duels and spent NONE of them: every
// second bank rated at a fraction of the first, so the rating filled the tank
// once and never again, and the deep rungs of the ladder were unreachable by
// construction. The two boards below are the same board except for how much one
// cast can take.
func TestBankingIsWorthRepeatingUpToWhatTheKitCanCash(t *testing.T) {
	for _, row := range []struct {
		name   string
		spends string
		want   string
	}{
		{"a spender that cashes the run", "deep", "bank"},
		{"a spender that cashes one", "sip", "strike"},
	} {
		fight := tank(t, "bank", row.spends, "strike")
		// Below the threshold on both boards, so neither spender is castable and
		// the choice really is "bank again or hit them".
		fuelled(t, fight, 3)
		if choice := chosen(t, fight); choice.Skill != row.want {
			t.Errorf("with %s, Suggest picked %q, want %q", row.name, choice.Skill, row.want)
		}
	}
}
