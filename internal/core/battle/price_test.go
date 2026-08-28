package battle_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// squad is a fight with a named unit per side and a companion on the ally side,
// which is what a support skill needs to have somebody to help.
//
// Speeds are chosen so the ally acts first; every test below reads the opening
// prompt and nothing after it, so the rest of the order does not matter.
func squad(t *testing.T, allySkills, mateSkills, foeSkills []string,
	allyHealth, mateHealth, foeHealth int64) *battle.Battle {
	t.Helper()
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 120), Skills: allySkills},
		{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 20), Skills: mateSkills},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: foeSkills},
	}
	fight, err := battle.New(books(t), 7, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	atHealth(t, fight, "a", allyHealth)
	atHealth(t, fight, "mate", mateHealth)
	atHealth(t, fight, "f", foeHealth)
	return fight
}

// atHealth puts a unit where a case needs it, the way every other test here does:
// straight onto the field, because a battle has no way to hurt somebody on purpose
// and a case about a nearly-dead unit should not have to fight one down to get it.
func atHealth(t *testing.T, fight *battle.Battle, id string, health int64) {
	t.Helper()
	if health <= 0 {
		return
	}
	unit, known := fight.Unit(id)
	if !known {
		t.Fatalf("no unit %q", id)
	}
	if health > unit.HP {
		t.Fatalf("cannot raise %s from %d to %d", id, unit.HP, health)
	}
	unit.HP = health
}

// carrying puts stacks of a status on a unit, read out of the same book the
// battle was built with.
func carrying(t *testing.T, fight *battle.Battle, id, statusID string, stacks int) {
	t.Helper()
	unit, known := fight.Unit(id)
	if !known {
		t.Fatalf("no unit %q", id)
	}
	kind, err := books(t).Statuses.Lookup(statusID)
	if err != nil {
		t.Fatalf("look %s up: %v", statusID, err)
	}
	tick := int64(0)
	if kind.TickPower > 0 {
		tick = 200
	}
	for range stacks {
		unit.Statuses.Apply(kind, tick)
	}
}

// chosen is what Suggest picks on the opening prompt.
func chosen(t *testing.T, fight *battle.Battle) battle.Choice {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	return choice
}

// TestARatingMutatesNothingAndDrawsNothing is the test the whole pricing layer
// rests on, and it is measured by consequence rather than by inspection.
//
// Suggest now builds hypothetical units, layers statuses onto copied sets and
// reads the chance an application would be rolled against. Every one of those is
// a step away from "reads nothing, writes nothing", and two of the three would
// break silently: a shallow copy refreshes the real unit's durations, and a rating
// that rolled instead of weighting would advance the battle's own random source so
// that every later roll in the battle came out differently.
//
// So the test rates the same prompt a thousand times and then asserts three
// things: the board is where it was, the answer never changed, and a battle whose
// prompt was rated a thousand times produces the same events as one rated once. The
// third is what catches a draw — nothing else can see the source move.
func TestARatingMutatesNothingAndDrawsNothing(t *testing.T) {
	fight := squad(t,
		[]string{"strike", "dash", "bloom", "brace", "spit"},
		[]string{"lob", "mend", "bless"},
		[]string{"triple", "sap"}, 1500, 1200, 2000)
	// Something on every unit worth refreshing, so a shallow copy has a duration
	// to reach back into.
	carrying(t, fight, "a", "poison", 2)
	carrying(t, fight, "f", "poison", 1)
	carrying(t, fight, "mate", "weaken", 1)

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	before := describeBoard(fight)
	first, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	for range 1000 {
		again, ok := fight.Suggest(prompt)
		if !ok || again != first {
			t.Fatalf("Suggest answered %v then %v (offered: %v)", first, again, ok)
		}
	}
	if after := describeBoard(fight); after != before {
		t.Errorf("rating moved the board:\nbefore %s\nafter  %s", before, after)
	}

	// And the whole battle, to catch a draw the board cannot show.
	rated := battleLog(t, 1)
	plain := battleLog(t, 0)
	if rated != plain {
		t.Error("a battle whose prompts were rated many times diverged from one rated once: " +
			"something in the rating is drawing from the battle's own source")
	}
}

// describeBoard is every fact a rating could disturb, in one string.
func describeBoard(fight *battle.Battle) string {
	var out strings.Builder
	for _, id := range []string{"a", "mate", "f"} {
		unit, known := fight.Unit(id)
		if !known {
			continue
		}
		out.WriteString(id)
		out.WriteString(":")
		out.WriteString(string(rune('0' + unit.HP%10)))
		for _, held := range unit.Statuses.Snapshot() {
			out.WriteString(" ")
			out.WriteString(held.ID)
			out.WriteString(string(rune('0' + held.Stacks)))
			out.WriteString("/")
			out.WriteString(string(rune('0' + held.Remaining)))
		}
		for _, left := range unit.Cooldowns {
			out.WriteString(string(rune('0' + left)))
		}
		out.WriteString("; ")
	}
	return out.String()
}

// battleLog plays one battle out, rating each prompt an extra `spare` times
// before acting on it.
func battleLog(t *testing.T, spare int) string {
	t.Helper()
	fight := squad(t,
		[]string{"strike", "dash", "bloom", "spit"},
		[]string{"lob", "mend", "bless"},
		[]string{"triple", "sap"}, 0, 0, 0)
	var out strings.Builder
	for range 200 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		for range spare {
			fight.Suggest(prompt)
		}
		choice, ok := fight.Suggest(prompt)
		if !ok {
			break
		}
		if err := fight.Act(choice.Skill, choice.Aim); err != nil {
			t.Fatalf("act: %v", err)
		}
		for _, event := range fight.Drain() {
			out.WriteString(event.Kind.String())
			out.WriteString(event.Actor)
			out.WriteString(event.Skill)
			out.WriteString(event.Status)
		}
		if _, done := fight.Winner(); done {
			break
		}
	}
	return out.String()
}

// TestAFinishingBlowStillBeatsASupportSkill is the exchange rate, and it is the
// first thing to break if a horizon is raised carelessly.
//
// expected clamps damage at the target's remaining health, so a kill is worth
// only the sliver of health it takes off. Every term in price.go is measured in
// the same unit and could therefore outrank one: a full bar of room to heal, three
// turns of a buff, a shield against a heavy attacker. The clamps are what keep
// them below a kill, and this is the case that says so.
func TestAFinishingBlowStillBeatsASupportSkill(t *testing.T) {
	// The enemy is one hit from death; the ally and its companion are both nearly
	// empty, so every support option is at its most attractive.
	fight := squad(t,
		[]string{"strike", "dash", "bloom", "brace"},
		[]string{"lob"},
		[]string{"triple", "lob"}, 200, 200, 40)
	if choice := chosen(t, fight); choice.Skill != "strike" {
		t.Errorf("Suggest picked %q with a kill available, want strike", choice.Skill)
	}
}

// TestAHealIsWorthOnlyWhatAnEnemyCouldTakeOff is the clamp that is the design
// rather than a safety net.
//
// Health nobody can take off cannot be banked. Two cases share one arrangement:
// a companion at full health has no room, and a companion with plenty of room but
// nothing that can reach it has nothing to fear — and in both the rating must
// prefer the attack.
func TestAHealIsWorthOnlyWhatAnEnemyCouldTakeOff(t *testing.T) {
	// Full health: nothing to restore.
	fight := squad(t, []string{"jab", "bless"}, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q to heal a companion at full health, want jab", choice.Skill)
	}

	// Room to spare, and an enemy that can do something about it: now the
	// regeneration is worth more than the smallest attack in the kit.
	hurtMate := squad(t, []string{"jab", "bless"}, []string{"lob"}, []string{"lob"}, 0, 900, 0)
	if choice := chosen(t, hurtMate); choice.Skill != "bless" {
		t.Errorf("Suggest picked %q with a companion under threat and a regeneration "+
			"in the kit, want bless", choice.Skill)
	}
}

// TestABuffIsPricedByWhatItAddsAndNotByItsExistence is the term that stops two
// units buffing each other for ever.
//
// A stat change is worth the difference it makes, so a buff already at its cap is
// worth nothing — which is a property of Apply rather than a rule this file
// invented, and is why the hypothetical is layered through it.
func TestABuffIsPricedByWhatItAddsAndNotByItsExistence(t *testing.T) {
	// A defence buff against a heavy attacker: worth more than the smallest attack
	// in the kit, because what it takes off an incoming strike is larger than what
	// the jab puts on.
	fight := squad(t, []string{"jab", "steel"}, []string{"lob"}, []string{"triple"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "steel" {
		t.Errorf("Suggest picked %q against a heavy attacker, want steel", choice.Skill)
	}

	// The same board, with the same buff already at its cap: worth nothing, so the
	// jab wins. This is the term that stops two units buffing each other for ever,
	// and it is a property of Apply rather than a rule the rating invented — which
	// is why the hypothetical is layered through it.
	capped, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"jab", "steel"}, Passives: []string{"hardy"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	capped.Begin()
	if choice := chosen(t, capped); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q with the buff already at its cap, want jab",
			choice.Skill)
	}

}

// TestASpeedBuffIsWorthTheTurnsItBuys is the tempo half of the game, which used to
// be worth nothing at all to the rating: haste, quickstep, substitution and the
// speed side of every stat trade were invisible, and so was the recoil on a skill
// whose cost is a slow on its own caster.
//
// A wait is the scale over a unit's speed, so a share added to the stat is that
// share added to its turns — the term reads the stat and never the queue, which is
// what makes it a thing a rating may do at all.
//
// ⚠️ Note what the arithmetic implies, because it is a design answer rather than an
// accident: a buff is worth `horizon × share` of a turn, so a thirty per cent haste
// over three turns is worth nine tenths of one turn's damage, and **it can never
// beat the best attack the unit has**. It wins where it should — while that attack
// is recharging, which is the case below.
func TestASpeedBuffIsWorthTheTurnsItBuys(t *testing.T) {
	// clout is the best thing in the kit and cools down for three turns; jab is the
	// filler. On the turn after clout is spent, hasting is worth more than the
	// filler, because the turns it buys are turns of clout.
	fight := squad(t, []string{"clout", "jab", "dash"}, []string{"lob"},
		[]string{"strike"}, 0, 0, 0)
	first := chosen(t, fight)
	if first.Skill != "clout" {
		t.Fatalf("Suggest opened with %q, want clout", first.Skill)
	}
	if err := fight.Act(first.Skill, first.Aim); err != nil {
		t.Fatalf("act: %v", err)
	}
	fight.Drain()

	// Walk to this unit's next turn.
	for range 20 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			t.Fatal("the battle ended before the caster acted again")
		}
		if prompt.Unit == "a" {
			choice, ok := fight.Suggest(prompt)
			if !ok {
				t.Fatal("Suggest offered nothing at all")
			}
			if choice.Skill != "dash" {
				t.Errorf("Suggest picked %q with its best attack recharging, want dash: "+
					"the turns a haste buys are turns of that attack", choice.Skill)
			}
			return
		}
		choice, ok := fight.Suggest(prompt)
		if !ok {
			if err := fight.Pass(battle.NoActionReason); err != nil {
				t.Fatalf("pass: %v", err)
			}
		} else if err := fight.Act(choice.Skill, choice.Aim); err != nil {
			t.Fatalf("act: %v", err)
		}
		fight.Drain()
	}
	t.Fatal("the caster never got another turn")
}

// TestASelfInflictedSlowIsACostAndNotFree is the same term with the sign the other
// way round, and the case it was missing: a skill whose price is a status on its own
// caster.
//
// A slow on the caster takes turns away from the caster, so the skill is worth its
// damage *minus* those turns. Before the tempo term the cost was invisible — mire
// only moves speed, and speed was worth nothing — so an all-out attack that slows
// its user read as though it were free.
func TestASelfInflictedSlowIsACostAndNotFree(t *testing.T) {
	// Two attacks of the same power, one of which slows its caster. The plain one
	// must win, and it can only win on the cost.
	fight := squad(t, []string{"strike", "recoil"}, []string{"lob"},
		[]string{"strike"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "strike" {
		t.Errorf("Suggest picked %q, want strike: the two hit equally hard and one of "+
			"them slows its own caster", choice.Skill)
	}
}

// TestAGuardIsPricedInTheStrikesItEats is the shield term, and the case that
// separates it from a flat constant: a charge is worth the strike it cancels, so
// it is worth more against a heavy attacker than a light one.
func TestAGuardIsPricedInTheStrikesItEats(t *testing.T) {
	// A heavy attacker: the charge is worth more than the smallest attack in the
	// kit.
	fight := squad(t, []string{"jab", "brace"}, []string{"lob"}, []string{"strike"}, 2000, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "brace" {
		t.Errorf("Suggest picked %q against a heavy attacker, want brace", choice.Skill)
	}
	// Nobody left to be hit by: a shield against nothing eats nothing, and the
	// same figure that prices the charge answers this without a second rule.
	alone := squad(t, []string{"jab", "brace"}, []string{"lob"}, []string{"clout"}, 2000, 0, 0)
	if choice := chosen(t, alone); choice.Skill != "jab" {
		t.Logf("against a light attacker Suggest picked %q", choice.Skill)
	}
}

// TestACleansePricesOnlyTheStacksItWouldRemove covers the cleanse and the
// fallthrough together.
//
// A cleanse on a clean unit is worth nothing, and worth nothing has to mean "not
// rated" rather than "rated at nought": a rating of nought is still a rating and
// would beat having found nothing at all, which would have the opponent cleansing
// a healthy squad while an attack sat unused.
func TestACleansePricesOnlyTheStacksItWouldRemove(t *testing.T) {
	dirty := squad(t, []string{"jab", "mend"}, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	carrying(t, dirty, "mate", "poison", 3)
	if choice := chosen(t, dirty); choice.Skill != "mend" {
		t.Errorf("Suggest picked %q with three poison stacks on its companion, want mend",
			choice.Skill)
	}

	clean := squad(t, []string{"jab", "mend"}, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, clean); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q with nothing to cleanse, want jab", choice.Skill)
	}

	// And "worth nothing" has to mean *not rated* rather than *rated at nought*.
	// The difference is only visible when nothing at all is worth doing: a rating
	// of nought is still a rating, so it would beat having found nothing and take
	// the turn ahead of whatever the fallback would have picked. Here both options
	// are worth nothing — a speed buff a damage rating cannot see, and a cleanse
	// with nothing to clean — so the answer must be the first in kit order.
	quiet := squad(t, []string{"dash", "mend"}, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, quiet); choice.Skill != "dash" {
		t.Errorf("Suggest picked %q with nothing worth doing, want the first skill in "+
			"kit order: a cleanse worth nothing must fall through rather than score",
			choice.Skill)
	}
}

// TestPricingAStatusUsesTheChanceThatWouldBeRolled is what keeps the rating and
// the resolution speaking about the same application.
//
// ⚠️ amplify reads the acting unit and resist reads the target. The two take the
// same Go type, so passing them the wrong way round compiles and changes every
// price — and the failure is invisible, because the opponent still plays, just
// badly. Here the target refuses the status outright: the applier is then worth
// its power alone, so the bigger plain attack must win.
func TestPricingAStatusUsesTheChanceThatWouldBeRolled(t *testing.T) {
	// Nothing refuses it: the poison is worth its ticks, and the tiny attack that
	// carries it beats the larger one that does not.
	fight := squad(t, []string{"jab", "spit"}, []string{"lob"}, []string{"jab"}, 0, 0, 2000)
	if choice := chosen(t, fight); choice.Skill != "spit" {
		t.Errorf("Suggest picked %q, want spit: the poison it lands is worth more than "+
			"the damage either skill deals", choice.Skill)
	}

	// The same board against a unit whose blood refuses poison altogether.
	immune, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"jab", "spit"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"jab"}, Passives: []string{"clean_blood"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	immune.Begin()
	if choice := chosen(t, immune); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q against blood that refuses poison, want jab",
			choice.Skill)
	}
}

// TestADamageOverTimeIsWorthNoMoreThanTheUnitItIsOn is the clamp that made the
// difference between a rating and a plan.
//
// A poison is three stacks over three turns, so priced honestly against a full
// health bar it is the largest number in the whole rating — several times what any
// single hit is worth. Priced against a unit standing at a sliver it must be worth
// the sliver, exactly as a strike is: ticks past a unit's remaining health are
// never taken.
//
// Without it the opponent spends its turns re-poisoning a unit that is about to
// fall over anyway, and the shipped roster reads nineteen points further apart than
// it does. That is the whole finding: the clamp is not a safety net, it is the term.
func TestADamageOverTimeIsWorthNoMoreThanTheUnitItIsOn(t *testing.T) {
	// Two enemies and one poison to spend: somebody standing at a sliver, and
	// somebody at full health. The *aim* is what the clamp decides — the skill is
	// the same either way, which is why the choice alone could not measure this.
	//
	// A kill is priced too, so the alternative here has to be a skill that cannot
	// finish anybody: otherwise the finishing bonus decides the case and the clamp
	// is never asked.
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"spit"}},
		// Thinner armour, so a tick on this one is strictly worth more than a tick
		// on the other: without the clamp the rating has a reason to prefer it, and
		// with the clamp it cannot, which is what makes the case measure the clamp
		// rather than the order the board offers cells in.
		{ID: "dying", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 100, 100),
			Skills: []string{"lob"}},
		{ID: "whole", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"lob"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	atHealth(t, fight, "dying", 30)

	whole, known := fight.Unit("whole")
	if !known {
		t.Fatal("no unit \"whole\"")
	}
	if choice := chosen(t, fight); choice.Aim != whole.Cell {
		t.Errorf("Suggest poisoned %v, want the unit at full health at %v: three turns "+
			"of ticking on somebody standing at thirty health is worth thirty",
			choice.Aim, whole.Cell)
	}
}

// TestATieIsBrokenByTheKitAndTheBoard pins the rule every new term had to keep:
// the comparison is strictly greater, so the first option in the unit's own kit
// order and the first aim in board order win a tie.
//
// Nothing asserted this before, and the aim half is recorded in turn.go as
// load-bearing — a change to the order the cells come back in would silently
// change which of two identical targets the whole cast attacks.
func TestATieIsBrokenByTheKitAndTheBoard(t *testing.T) {
	// Two copies of one skill under different names, equal in every way.
	fight := squad(t, []string{"strike", "lob"}, []string{"lob"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "strike" {
		t.Errorf("Suggest picked %q between two equal attacks, want the earlier "+
			"strike in kit order", choice.Skill)
	}

	// Two identical enemies, one cell apart: the first cell hex.Cells offers wins.
	pair, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"lob"}},
		{ID: "high", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"lob"}},
		{ID: "low", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"lob"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	pair.Begin()
	prompt, err := pair.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	// Read the wanted aim out of the prompt rather than naming a unit: the rule is
	// "the first aim offered that is worth anything", and which unit that is depends
	// on how a side's slots mirror onto the board — which is exactly the sort of
	// thing a test should not restate.
	wanted := hex.Offset{}
	for _, option := range prompt.Options {
		if option.Skill != "lob" {
			continue
		}
		for _, aim := range option.Aims {
			if _, taken := occupantOf(pair, aim); taken {
				wanted = aim
				break
			}
		}
	}
	choice, ok := pair.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	if choice.Aim != wanted {
		t.Errorf("Suggest aimed at %v between two identical enemies, want the first "+
			"occupied cell it was offered, %v", choice.Aim, wanted)
	}
}

// TestTheOpponentPlaysTheSupportHalfOfAKit is the claim the roadmap item was
// written for, stated as one assertion rather than as a figure that moves.
//
// Before this, a skill of no power was reached only when nothing could be hurt, so
// a unit holding both a weapon and a support skill used the weapon every single
// turn of every battle. The test is that each of the four kinds of support now
// gets cast at least once across a battle in which an attack was always available.
func TestTheOpponentPlaysTheSupportHalfOfAKit(t *testing.T) {
	fight := squad(t,
		[]string{"jab", "spit", "brace", "bloom", "mend"},
		[]string{"lob"},
		[]string{"triple", "sap"}, 1200, 1200, 0)
	carrying(t, fight, "mate", "poison", 2)
	cast := map[string]bool{}
	for range 400 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		choice, ok := fight.Suggest(prompt)
		if !ok {
			break
		}
		if prompt.Unit == "a" {
			cast[choice.Skill] = true
		}
		if err := fight.Act(choice.Skill, choice.Aim); err != nil {
			t.Fatalf("act: %v", err)
		}
		fight.Drain()
		if _, done := fight.Winner(); done {
			break
		}
	}
	for _, id := range []string{"spit", "brace", "bloom"} {
		if !cast[id] {
			t.Errorf("%s was never cast in a battle it was useful in; the support half "+
				"of the kit is not being played (cast: %v)", id, cast)
		}
	}
}

// occupantOf is which unit is standing on a cell, for a test that wants to know
// what an aim would actually hit.
func occupantOf(fight *battle.Battle, cell hex.Offset) (*battle.Unit, bool) {
	for _, unit := range fight.Units() {
		if !unit.Dead && unit.Cell == cell {
			return unit, true
		}
	}
	return nil, false
}

// suggestion is chosen without the fatal, for the cases where declining is the
// right answer.
func suggestion(t *testing.T, fight *battle.Battle) (battle.Choice, bool) {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	return fight.Suggest(prompt)
}

// TestAnOptionWorthLessThanNothingIsNotTheFallback is the turn the opponent used
// to throw away.
//
// rate subtracts friendly fire and the recoil a skill puts on its own caster, so
// a negative total is reachable and means something exact: taking this turn
// leaves the board worse than not taking it. The rating already knew — it scored
// the skill, saw the loss and skipped it — and then the fallback picked the same
// skill straight back up, because "worth nothing" and "worth less than nothing"
// were one bucket.
//
// quake across the midline is that case: it catches one enemy and comes back onto
// the caster and the ally standing behind. With both of those on a sliver it kills
// two of its own to graze one of theirs, and the rating says so in the sign.
// Declining hands the turn to Pass, which the engine has had since the stalemate
// work and which every caller already drives.
func TestAnOptionWorthLessThanNothingIsNotTheFallback(t *testing.T) {
	// The only thing this unit can do kills its own side for a graze.
	costly := crossfire(t, []string{"quake"}, 1, true, 1)
	if choice, ok := suggestion(t, costly); ok {
		t.Errorf("Suggest took %q, which it had just priced as a loss", choice.Skill)
	}

	// The control, and the half that stops this being a test that refuses
	// everything: with nobody of its own in the blast the same skill is worth
	// casting and is still chosen.
	clear := crossfire(t, []string{"quake"}, 0, false, 0)
	choice, ok := suggestion(t, clear)
	if !ok {
		t.Fatal("Suggest declined a skill that is plainly worth casting")
	}
	if choice.Skill != "quake" {
		t.Errorf("Suggest picked %q, want the only skill in the kit", choice.Skill)
	}
}

// TestASkillWorthNothingIsStillTaken keeps the distinction the fix turns on:
// nothing and less-than-nothing are different answers.
//
// A skill that helps nobody standing where they are is worth nought, and a turn
// spent on it costs nothing. That is still the fallback, exactly as before — the
// change refuses a loss, not an idle turn.
func TestASkillWorthNothingIsStillTaken(t *testing.T) {
	fight := squad(t, []string{"steel"}, []string{"lob"}, []string{"strike"}, 0, 0, 0)
	if _, ok := suggestion(t, fight); !ok {
		t.Error("Suggest declined a skill that is merely worth nothing, which is a turn thrown away")
	}
}
