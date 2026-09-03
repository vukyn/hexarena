// A reserve spent on health rather than on a blow, from both ends: what the
// engine pays out, and what the rating thought it would pay.
//
// This file is in package battle rather than battle_test, and it is the second
// file here that is. The reason carry_wall_test.go gives applies word for word:
// two of the four claims below are about `pricing`, and a price is not a board
// state — `restored` and `spentCounter` have no consequence a fixture can watch
// for, because what they change is which skill Suggest would rather cast, and a
// test that read the choice would be measuring every other term in the rating at
// the same time. The figures ARE the measurement, so they are read directly.
//
// ⚠️ It therefore carries its own books rather than the shared ones next door,
// which live in package battle_test and are invisible from here.
package battle

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// healRate is the fixture's health per stack, and healThreshold what its spender
// needs before it holds. Named rather than spelled into each assertion, because
// every figure below is derived from them and a test that repeated the literal
// would be a second copy of the arithmetic under measurement.
const (
	healRate      = 200
	healThreshold = 2
)

// reserveHealBooks is a counter its holder banks on itself and one skill that
// spends it for health.
//
// `mend` is the whole subject: aimed at its caster, no power of its own, and its
// entire payout written on the condition — which is the shape a flat `restores`
// cannot express, because a condition is an amplifier and a flat restore beside
// one pays out in full to a caster holding nothing.
func reserveHealBooks(t *testing.T) Books {
	t.Helper()
	chart, err := element.ParseChart([]byte(`{
	  "multipliers": {"advantage": 1500, "neutral": 1000, "disadvantage": 667},
	  "cycles": [
	    {"name": "organic", "chain": ["water", "fire", "grass", "ground"]},
	    {"name": "industrial", "chain": ["ice", "metal", "wind", "electric"]}
	  ],
	  "mutual": [["light", "dark"]],
	  "inert": ["neutral"]
	}`))
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	patterns, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3, "splash_power": 500,
	  "patterns": [{"name": "single", "splash": []}]
	}`))
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6, "max_counter_stacks": 40,
	  "kinds": [
	    {"id": "fuel", "category": "reserve", "max_stacks": 40, "duration": 6}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	skills, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"mend","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_requires":{"status":"fuel","min_stacks":2,"consume":true,
	    "stack_restore":200}},
	  {"id":"bank","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_applies":[{"status":"fuel","chance":1000,"stacks":3}]},
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"}
	]}`), skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	return Books{
		Rules: combat.Rules{
			DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250,
			MinHitChance: 150, MaxBlockCharges: 3,
		},
		Chart:  chart,
		Bounds: modifier.Bounds{Headroom: 3000, FloorFraction: 100, MaxAffinityScale: 1000},
		Limits: progression.Limits{
			LevelCap: progression.LevelCap,
			Ceilings: progression.Values{
				progression.HP: 4800, progression.Attack: 800, progression.Defense: 800,
				progression.Speed: 200, progression.Accuracy: 300, progression.Dodge: 150,
			},
			MaxEffectiveHP: 11500,
		},
		Patterns: patterns, Statuses: statuses, Skills: skills,
	}
}

// aTankAndAWound is one hurt caster carrying the heal, and one enemy quick
// enough to be a real threat and slow enough never to take a turn first.
//
// The caster is hurt on purpose and hurt DEEPLY: every figure a price puts on a
// heal is clamped by the room there is, so a fixture that healed a nearly-full
// unit would be measuring the clamp rather than the payment.
func aTankAndAWound(t *testing.T, stacks int) (*Battle, *Unit) {
	t.Helper()
	fight, err := New(reserveHealBooks(t), 7, []Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 800, 200, 200),
			Skills: []string{"mend", "bank"}},
		{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 800, 200, 1),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	caster, known := fight.Unit("caster")
	if !known {
		t.Fatal("no caster on the board")
	}
	caster.HP = 400
	kind, err := fight.books.Statuses.Lookup("fuel")
	if err != nil {
		t.Fatalf("lookup fuel: %v", err)
	}
	for range stacks {
		caster.Statuses.Apply(kind, 0)
	}
	if got := caster.Statuses.Stacks("fuel"); got != stacks {
		t.Fatalf("the caster holds %d fuel, want %d", got, stacks)
	}
	return fight, caster
}

func healStats(hp, attack, defense, speed int64) progression.Values {
	return progression.Values{
		progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
		progression.Speed: speed, progression.Accuracy: 0, progression.Dodge: 0,
	}
}

func neutralOnly(t *testing.T) element.Affinity {
	t.Helper()
	member, err := element.Parse("neutral")
	if err != nil {
		t.Fatalf("parse neutral: %v", err)
	}
	affinity, err := element.Single(member)
	if err != nil {
		t.Fatalf("single neutral: %v", err)
	}
	return affinity
}

// healOnce takes the caster's turn with the heal and reports what it paid out,
// off the log rather than off the health, so a test asserting nothing happened
// cannot pass on a unit that was hurt back inside the same turn.
func healOnce(t *testing.T, fight *Battle, caster *Unit) (int64, int) {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "caster" {
		t.Fatalf("the turn did not go to the caster: %+v", prompt)
	}
	if err := fight.Act("mend", caster.Cell); err != nil {
		t.Fatalf("act mend: %v", err)
	}
	total, events := int64(0), 0
	for _, event := range fight.Drain() {
		if event.Kind == Healed {
			total += event.Amount
			events++
		}
	}
	return total, events
}

// TestAReserveHealIsReadBeforeTheSpendEmptiesTheTank is the ordering defect this
// whole mechanism turns on.
//
// Act reads the swing, spends the stacks and only then pays out, in that order —
// so a restore that asked the caster what it was carrying would be asking an
// emptied tank. It would not fail loudly: it would heal nought on every cast for
// ever, on a skill that parsed, glossed and shipped. The figure therefore rides
// in on `swing`, and this is what says so.
//
// ⚠️ Asserted as an EQUALITY against the arithmetic, not as "more than nothing".
// A reading taken one line later would still heal on a cast that happened to take
// fewer stacks than it spent, and a test that only asked for a positive number
// would pass on it.
func TestAReserveHealIsReadBeforeTheSpendEmptiesTheTank(t *testing.T) {
	const stacks = 5
	fight, caster := aTankAndAWound(t, stacks)
	rules := fight.books.Rules
	want := rules.Restore(fight.Stats(caster)[progression.Attack], healRate*stacks)

	healed, events := healOnce(t, fight, caster)
	if events != 1 {
		t.Fatalf("a reserve heal emitted %d healed events, want one", events)
	}
	if healed != want {
		t.Errorf("the heal paid %d, want %d: the payout was read off a tank the spend had already emptied",
			healed, want)
	}
	if left := caster.Statuses.Stacks("fuel"); left != 0 {
		t.Errorf("the caster kept %d stacks, so the heal was not paid for by the whole spend", left)
	}
}

// TestAReserveHealWithNoFuelPaysNothing is the half a flat `restores` cannot
// express, and it is asserted as no event rather than as a small one.
//
// A condition is an amplifier: failing it charges the caster nothing and stops
// nothing, so a skill declaring `restores` beside one pays out in full to a
// caster with an empty tank. The heal has to be paid for by the pile or the pile
// is not paying for it, and the only shape that says so is a per-stack rate off a
// flat figure of nought.
func TestAReserveHealWithNoFuelPaysNothing(t *testing.T) {
	fight, caster := aTankAndAWound(t, 0)
	healed, events := healOnce(t, fight, caster)
	if events != 0 || healed != 0 {
		t.Errorf("a caster holding no fuel was healed %d over %d events, want none at all",
			healed, events)
	}
}

// TestPricingAReserveHealReadsTheHealthTheEngineWouldPay is the rule this
// repository keeps paying for in a new shape: a price built from a second reading
// lets the opponent prefer a skill for something the skill does not do, or refuse
// one for something it does.
//
// `restored` used to read `declared.Restores` alone, which is nought on every
// skill here — so all three shipped reserve heals would have rated at nothing and
// Suggest would never have cast one. It now goes through swingOf, the same
// function Battle.restore is handed its half from.
//
// ⚠️ Asserted as an EQUALITY, and the fixture is arranged so that neither clamp
// inside worthHealing binds: the caster is deeply hurt so there is room, and the
// enemy hits hard enough that the heal is inside the horizon. Both are checked
// rather than assumed, because a fixture that drifted into a clamp would make this
// test pass on two figures that agreed only by being cut to the same ceiling.
func TestPricingAReserveHealReadsTheHealthTheEngineWouldPay(t *testing.T) {
	const stacks = 5
	fight, caster := aTankAndAWound(t, stacks)
	declared, err := fight.books.Skills.Lookup("mend")
	if err != nil {
		t.Fatalf("lookup mend: %v", err)
	}
	prices := fight.newPricing()
	priced := prices.restored(caster, caster, declared)
	if priced <= 0 {
		t.Fatalf("the rating priced a reserve heal at %d, so Suggest would never cast one", priced)
	}
	raw := fight.books.Rules.Restore(fight.Stats(caster)[progression.Attack], healRate*stacks)
	if priced != raw {
		t.Fatalf("the price came out at %d against an unclamped %d, so the fixture is measuring worthHealing rather than the reading",
			priced, raw)
	}

	healed, events := healOnce(t, fight, caster)
	if events != 1 {
		t.Fatalf("the cast emitted %d healed events, want one", events)
	}
	if healed != priced {
		t.Errorf("the rating expected %d health and the cast paid %d", priced, healed)
	}
}

// TestASpentReserveIsChargedForAHealAsForABlow is the cost half, and it is the
// finding rather than a completeness check.
//
// `selfSpendable` skipped every skill of no power, which is exactly the shape a
// reserve-paid heal has — so a unit whose only spender was a heal valued its whole
// tank at nought. It would never think banking was worth a turn, cashing in would
// cost it nothing, and a dispel aimed at its reserve would be free.
//
// ⚠️ Proved by mutation rather than by inspection: putting `|| declared.Power ==
// 0` back on the lookup in selfSpendable makes both halves below read nought.
func TestASpentReserveIsChargedForAHealAsForABlow(t *testing.T) {
	const stacks = 5
	fight, caster := aTankAndAWound(t, stacks)
	declared, err := fight.books.Skills.Lookup("mend")
	if err != nil {
		t.Fatalf("lookup mend: %v", err)
	}
	kind, err := fight.books.Statuses.Lookup("fuel")
	if err != nil {
		t.Fatalf("lookup fuel: %v", err)
	}
	prices := fight.newPricing()
	if worth := prices.spendable(caster, caster, kind); worth <= 0 {
		t.Errorf("one stack of a reserve whose only spender is a heal is worth %d, so banking it is a turn thrown away",
			worth)
	}
	if spent := prices.spentCounter(caster, declared); spent <= 0 {
		t.Errorf("cashing in %d stacks for a heal cost the rating %d, so it would empty the tank the moment it could",
			stacks, spent)
	}
	// And the ceiling on what one cast may take is the same figure the spend
	// removes, read through the one function both sides ask.
	if takes := declared.SelfRequires.Takes(stacks); takes != stacks {
		t.Errorf("a spend of %d stacks took %d", stacks, takes)
	}
	if most := declared.SelfRequires.Takes(999); most != skill.MaxSpendRestore/healRate {
		t.Errorf("a spend against a full reserve took %d stacks, want the %d the ceiling pays for",
			most, skill.MaxSpendRestore/healRate)
	}
	if declared.SelfRequires.MinStacks != healThreshold {
		t.Errorf("the fixture's threshold moved to %d", declared.SelfRequires.MinStacks)
	}
}
