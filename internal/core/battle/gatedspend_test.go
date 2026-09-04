// A cooldownless spender that may only be cast on fuel it charged itself, from
// all three ends: what the parser lets an author write, what the board offers,
// and what the rating thinks a stack of that fuel is worth.
//
// This file is in package battle rather than battle_test, and it is the third
// file here that is. The reason carry_wall_test.go gives applies word for word:
// one of the claims below is about `pricing`, and a price is not a board state —
// `selfSpendable` has no consequence a fixture can watch for, because what it
// changes is which skill Suggest would rather cast, and a test that read the
// choice would be measuring every other term in the rating at the same time. The
// figure IS the measurement, so it is read directly.
//
// ⚠️ It therefore carries its own books rather than the shared ones next door,
// which live in package battle_test and are invisible from here.
package battle

import (
	"strings"
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

// The fixture's own numbers, named because every assertion below is derived from
// them and a test that spelled the literal twice would be a second copy of the
// arithmetic under measurement.
const (
	gatedCap  = 5 // the reserve's ceiling, so the deeper spend empties it exactly
	gatedNeed = 3 // what `release` needs and takes
	gatedDeep = 5 // what `unleash` needs and takes
	// gatedLasts is the fuel's duration, and it is deliberately SHORTER than the
	// tank is deep.
	//
	// ⚠️ A duration at or past the depth makes the refresh in status.Apply
	// invisible: the first stack outlives the last one's arrival on its own, so a
	// fixture written that way passes with the refresh ripped out. Measured --
	// with the duration at 5 the mutation survived. At three, the first stack has
	// expired by the fourth turn unless topping the tank up carried it.
	gatedLasts = 3
)

// gatedBooks is a reserve its holder builds on itself and two spenders that may
// only be cast on it.
//
// `stoke` is the whole of the charging half: cooldown nought, aimed at its own
// caster, and one stack a turn. `release` and `unleash` are the spending half and
// are the subject — cooldown nought as well, a flat power with no bonus anywhere
// on the condition, and a `gates` that is the only thing standing between them
// and being cast every turn for the unamplified blow.
//
// Two spenders rather than one, at three stacks and at five, because the rung
// they sit on is the only thing that makes a tank worth topping up rather than
// cashing the moment it opens — and because a spend that took the whole pile
// would put the deeper one out of reach for ever, which is a failure only a
// second rung can see.
//
// ⚠️ **`jab` is not padding.** It is the cooldownless blow that is worth
// SOMETHING, and without it a caster holding only `stoke` and `release` charges
// whatever the rating thinks fuel is worth: with the spender gated off there is
// nothing else on offer, so Suggest's fallback arm casts the charger for want of
// anything to do. Measured — a loop test on that kit passes against a
// selfSpendable with the gated arm ripped out, which is the whole term it was
// supposed to be checking. A filler on the board is what makes charging a
// decision instead of the only thing left.
func gatedBooks(t *testing.T) Books {
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
	    {"id": "fuel", "category": "reserve", "max_stacks": 5, "duration": 3}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	skills, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"stoke","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"fuel","chance":1000,"stacks":1}]},
	  {"id":"release","element":"neutral","range":1,"pattern":"single",
	   "power":5800,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":3,"gates":true,
	    "consume":true,"consume_stacks":3}},
	  {"id":"unleash","element":"neutral","range":1,"pattern":"single",
	   "power":9700,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_requires":{"status":"fuel","min_stacks":5,"gates":true,
	    "consume":true,"consume_stacks":5}},
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":600,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":300,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"}
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

// aChargerAndATarget is one caster carrying the kit named and one enemy slow
// enough never to take a turn first, with the caster's tank filled by hand.
//
// Filled by hand rather than by casting `stoke` a number of times, so a test can
// name the depth it is measuring instead of hoping the arithmetic came out where
// it meant to. The one test that cares how a tank is really built casts for it.
func aChargerAndATarget(t *testing.T, stacks int, kit ...string) (*Battle, *Unit) {
	t.Helper()
	fight, err := New(gatedBooks(t), 11, []Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: gatedStats(4800, 400, 200, 200),
			Skills: kit},
		{ID: "them", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: gatedStats(4800, 200, 200, 1),
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
	if stacks > 0 {
		kind, err := fight.books.Statuses.Lookup("fuel")
		if err != nil {
			t.Fatalf("lookup fuel: %v", err)
		}
		for range stacks {
			caster.Statuses.Apply(kind, 0)
		}
		if got := caster.Statuses.Stacks("fuel"); got != stacks {
			t.Fatalf("the caster holds %d fuel, want %d: the fixture cannot measure what it cannot bank", got, stacks)
		}
	}
	return fight, caster
}

func gatedStats(hp, attack, defense, speed int64) progression.Values {
	return progression.Values{
		progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
		progression.Speed: speed, progression.Accuracy: 0, progression.Dodge: 0,
	}
}

// TestAGatedSpenderMakesItsFuelWorthATurn is the term the whole feature stands
// on, and it is the exact state the shipped code was in before this change: no
// other test in the repository notices it.
//
// selfSpendable reads a stack's worth off StackPower, and failing that off
// BonusPower divided by the stacks one cast takes. A gated spender carries
// NEITHER — its figure is the flat `power` on its own face, which is the whole
// point of the shape — so `gain` came out nought, the loop moved on, and the
// builder's grant was priced at exactly nothing. Measured on the tree with the
// parser change alone and this arm missing: 0.
//
// A grant worth nothing is a turn Suggest never spends, so the tank never fills,
// so the gate never opens and the spender is unreachable. The kit would parse,
// gloss, ship, and never be cast.
func TestAGatedSpenderMakesItsFuelWorthATurn(t *testing.T) {
	fight, caster := aChargerAndATarget(t, 0, "stoke", "release")
	kind, err := fight.books.Statuses.Lookup("fuel")
	if err != nil {
		t.Fatalf("lookup fuel: %v", err)
	}
	prices := fight.newPricing()
	worth := prices.spendable(caster, caster, kind)
	t.Logf("one stack of fuel is worth %d to a unit whose only spender is gated", worth)
	if worth <= 0 {
		t.Errorf("one stack of a reserve whose only spender is gated is worth %d, so charging is a turn thrown away and the gate never opens",
			worth)
	}
}

// mustPrompt opens the caster's turn, or says so.
func mustPrompt(t *testing.T, fight *Battle) *Prompt {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "caster" {
		t.Fatalf("the turn did not go to the caster: %+v", prompt)
	}
	return prompt
}

// offeredBy is a prompt's options by skill, so a test asking about two of them
// reads one prompt rather than trying to open a second turn.
func offeredBy(t *testing.T, prompt *Prompt) map[string]Option {
	t.Helper()
	out := make(map[string]Option, len(prompt.Options))
	for _, option := range prompt.Options {
		out[option.Skill] = option
	}
	return out
}

// prompted takes the caster's turn and hands back the option for one skill,
// which is what every question about the offer is really asking.
func prompted(t *testing.T, fight *Battle, id string) Option {
	t.Helper()
	offered := offeredBy(t, mustPrompt(t, fight))
	option, listed := offered[id]
	if !listed {
		t.Fatalf("the caster was never offered %q at all, so nothing here measured an offer", id)
	}
	return option
}

// TestAGatedSpenderIsNotOnOffer is the gate as the board sees it, and it is what
// makes a cooldownless spender a decision rather than a filler.
//
// min_stacks on its own is an amplifier: a caster short of the fuel casts anyway
// and simply pays no bonus. With a cooldown of nought that is a skill cast every
// single turn for its unamplified blow, which deletes the kit. Gated, the option
// goes out with no aims at all and Suggest passes over it exactly as it passes
// over a cooldown.
func TestAGatedSpenderIsNotOnOffer(t *testing.T) {
	fight, _ := aChargerAndATarget(t, gatedNeed-1, "stoke", "release", "unleash")
	offered := offeredBy(t, mustPrompt(t, fight))
	option := offered["release"]
	if option.Available() {
		t.Errorf("a spender needing %d stacks was offered %d aim(s) to a caster holding %d",
			gatedNeed, len(option.Aims), gatedNeed-1)
	}
	if option.Blocked != BlockFuel {
		t.Errorf("the option was blocked as %d, want BlockFuel (%d)", option.Blocked, BlockFuel)
	}
	if option.Status != "fuel" {
		t.Errorf("the block names %q as the fuel it is short of, want fuel", option.Status)
	}
	if option.Need != gatedNeed {
		t.Errorf("the block asks for %d stacks, want %d", option.Need, gatedNeed)
	}
	// ⚠️ **Held is asserted separately and against a DIFFERENT figure**, because
	// the obvious bug is filling it in from MinStacks -- the number already in
	// hand at that line. An assertion that only read the flag, or that let Need
	// and Held be the same number, would pass against a field that never looked at
	// the caster at all.
	if option.Held != gatedNeed-1 {
		t.Errorf("the block says the caster holds %d, want %d: Held reads the CASTER, not the threshold",
			option.Held, gatedNeed-1)
	}
	// The sentence is still there and still English. PR 3 moves it deliberately;
	// nothing here may.
	if option.Reason == "" {
		t.Error("the blocked option carries no reason, so every renderer in the repository prints a blank")
	}
	// And the rung above is blocked too, at its own figures: the two spenders are
	// gated apart rather than together, which is what makes the tank's depth a
	// decision rather than a switch.
	if deeper := offered["unleash"]; deeper.Blocked != BlockFuel || deeper.Need != gatedDeep {
		t.Errorf("the deeper spender was blocked as %d needing %d, want BlockFuel needing %d",
			deeper.Blocked, deeper.Need, gatedDeep)
	}
}

// TestAGatedSpenderIsOnOfferTheMomentTheFuelIsThere is the other side of the
// threshold, and it is a separate test because the comparison has two ways to be
// wrong and only one of them is visible from the case above.
func TestAGatedSpenderIsOnOfferTheMomentTheFuelIsThere(t *testing.T) {
	fight, _ := aChargerAndATarget(t, gatedNeed, "stoke", "release")
	option := prompted(t, fight, "release")
	if !option.Available() {
		t.Errorf("a caster holding exactly the %d stacks the spender asks for was refused it: %q",
			gatedNeed, option.Reason)
	}
	if option.Blocked != BlockNone {
		t.Errorf("an available option reports itself blocked as %d, want BlockNone", option.Blocked)
	}
}

// TestActRefusesAGatedSpenderTheOptionsListAlreadyRefused is why the gate is
// written twice.
//
// options() builds a PROMPT, and a client is under no obligation to read one.
// internal/room drives a PvP turn straight through Act off a decision that
// arrived over the wire, so a gate written only into the offer is a gate a peer
// walks past — and walking past it is casting a 5800-power blow on an empty tank.
func TestActRefusesAGatedSpenderTheOptionsListAlreadyRefused(t *testing.T) {
	fight, _ := aChargerAndATarget(t, gatedNeed-1, "stoke", "release")
	them, known := fight.Unit("them")
	if !known {
		t.Fatal("no target on the board")
	}
	mustPrompt(t, fight)
	before := them.HP
	err := fight.Act("release", them.Cell)
	if err == nil {
		t.Fatalf("Act cast a gated spender on a tank of %d, taking %d health off",
			gatedNeed-1, before-them.HP)
	}
	for _, want := range []string{"fuel", "3", "2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal says %q, which never mentions %q: it has to name the status, the need and the holding",
				err, want)
		}
	}
	// And nothing was spent on the way to the refusal.
	if them.HP != before {
		t.Errorf("the refused cast still took %d health off the target", before-them.HP)
	}
}

// TestTheGateDoesNotReachTheHypotheticals is the decision behind where the gate
// was put, asserted rather than left as a comment.
//
// aims() is asked by four callers that are not about to cast anything, and
// pricing asks one of them — bestStrike, through pricing.strike — with the tank
// EMPTY on purpose, because what it is deciding is whether charging is worth a
// turn. A gate inside aims() would answer nought at exactly the moment that price
// has to be positive: the rating would never charge, the tank would never fill,
// and the gate would never open.
func TestTheGateDoesNotReachTheHypotheticals(t *testing.T) {
	fight, caster := aChargerAndATarget(t, 0, "stoke", "release")
	if held := caster.Statuses.Stacks("fuel"); held != 0 {
		t.Fatalf("the caster holds %d fuel, and this test measures an EMPTY tank", held)
	}
	declared, err := fight.books.Skills.Lookup("release")
	if err != nil {
		t.Fatalf("lookup release: %v", err)
	}
	them, _ := fight.Unit("them")
	wanted := fight.expected(caster, declared, them.Cell)
	if wanted <= 0 {
		t.Fatalf("the gated spender is expected to deal %d, so nothing here measured anything", wanted)
	}
	if got := fight.bestStrike(caster); got < wanted {
		t.Errorf("bestStrike on an empty tank reads %d against the gated spender's own %d, so the gate reached aims() and charging is priced at nothing",
			got, wanted)
	}
}

// TestAGatedSpendTakesExactlyWhatItNames is the magazine rule on the shape that
// needs it most: a gated spender names an exact price, and taking the whole tank
// for it would make the deeper spender beside it unreachable for ever.
func TestAGatedSpendTakesExactlyWhatItNames(t *testing.T) {
	fight, caster := aChargerAndATarget(t, gatedDeep, "stoke", "release")
	them, _ := fight.Unit("them")
	mustPrompt(t, fight)
	if err := fight.Act("release", them.Cell); err != nil {
		t.Fatalf("act release: %v", err)
	}
	consumed := 0
	for _, event := range fight.Drain() {
		if event.Kind == StatusConsumed && event.Status == "fuel" {
			consumed += event.Stacks
		}
	}
	if consumed != gatedNeed {
		t.Errorf("a spend of %d stacks off a tank of %d consumed %d", gatedNeed, gatedDeep, consumed)
	}
	if left := caster.Statuses.Stacks("fuel"); left != gatedDeep-gatedNeed {
		t.Errorf("it left %d stacks behind, want %d: a spend that empties the tank makes the deeper spender unreachable",
			left, gatedDeep-gatedNeed)
	}
}

// TestTheTankSurvivesBeingToppedUp is the half of the loop nothing else here
// measures: the fuel has a DURATION, and a builder granting one stack a turn is
// racing it.
//
// status.Apply refreshes the remaining turns of every stack already there when a
// new one lands, which is what makes sustained charging arrive at a full tank
// rather than at a rolling window. Without that refresh the first stack expires
// under the fifth, the deep spender is never reachable, and nothing else in the
// engine would say so.
//
// ⚠️ **The fuel lasts fewer turns than the tank is deep, and that is the whole
// fixture.** See gatedLasts: written the other way round the first stack outlives
// the last one's arrival by itself, and this test passes against a status.Apply
// with the refresh loop deleted -- measured.
func TestTheTankSurvivesBeingToppedUp(t *testing.T) {
	fight, caster := aChargerAndATarget(t, 0, "stoke")
	// The premise, read off the book rather than trusted: a fuel that outlasts the
	// tank's depth would make everything below pass without a refresh anywhere.
	kind, err := fight.books.Statuses.Lookup("fuel")
	if err != nil {
		t.Fatalf("lookup fuel: %v", err)
	}
	if kind.Duration != gatedLasts || gatedLasts >= gatedCap {
		t.Fatalf("the fuel lasts %d turns against a tank %d deep, and this test only measures the refresh while it lasts FEWER",
			kind.Duration, gatedCap)
	}
	for turn := 1; turn <= gatedCap; turn++ {
		mustPrompt(t, fight)
		if err := fight.Act("stoke", caster.Cell); err != nil {
			t.Fatalf("stoke on turn %d: %v", turn, err)
		}
		fight.Drain()
		if held := caster.Statuses.Stacks("fuel"); held != turn {
			t.Fatalf("after %d turns of charging the caster holds %d stacks, want %d: a stack expiring under the one above it is a tank that never fills",
				turn, held, turn)
		}
	}
	if held := caster.Statuses.Stacks("fuel"); held != gatedCap {
		t.Errorf("%d turns of charging arrived at %d stacks, want the full %d", gatedCap, held, gatedCap)
	}
}

// TestTheRatingChargesUntilTheGateOpens is the loop closing on a real board,
// which is the half no price can be read off on its own.
//
// Everything else here is a rule asserted at the place it is written. This is
// both rules meeting: the rating has to think spending a turn on fuel is worth
// it — which is only true because selfSpendable has the gated arm — and then, at
// the threshold, has to think cashing it in beats charging again. A kit that
// parses and is never cast is the same dead branch as one that was never written.
//
// ⚠️ **The kit carries a FILLER, and the test is worthless without it.** Given
// only the charger and the gated spender, the rating charges either way: with the
// spender gated off nothing at all is on offer, so Suggest's fallback arm casts
// the cooldownless charger for want of anything to do. Measured — that kit passes
// against a selfSpendable with the gated arm deleted, which is exactly the term
// this test exists to hold. `jab` is worth something every turn, so charging has
// to out-price it, and it can only do that if a stack of fuel has a price at all.
func TestTheRatingChargesUntilTheGateOpens(t *testing.T) {
	fight, caster := aChargerAndATarget(t, 0, "jab", "stoke", "release")
	charged, released, jabbed := 0, 0, 0
	for range 12 {
		prompt, err := fight.Advance()
		if err != nil || prompt == nil {
			break
		}
		if prompt.Unit == "caster" {
			choice, chose := fight.Suggest(prompt)
			if !chose {
				t.Fatalf("the rating declined a turn holding %d fuel, so it would rather do nothing than charge",
					caster.Statuses.Stacks("fuel"))
			}
			if err := fight.Act(choice.Skill, choice.Aim); err != nil {
				t.Fatalf("act %s: %v", choice.Skill, err)
			}
			switch choice.Skill {
			case "stoke":
				charged++
			case "release":
				released++
			case "jab":
				jabbed++
			}
		} else {
			choice, chose := fight.Suggest(prompt)
			if !chose {
				if err := fight.Pass("nothing to do"); err != nil {
					t.Fatalf("pass: %v", err)
				}
			} else if err := fight.Act(choice.Skill, choice.Aim); err != nil {
				t.Fatalf("their act %s: %v", choice.Skill, err)
			}
		}
		fight.Drain()
		if released > 0 {
			break
		}
	}
	t.Logf("the rating charged %d time(s), jabbed %d and released %d", charged, jabbed, released)
	if charged < gatedNeed {
		t.Errorf("the rating charged %d time(s) and jabbed %d before the gate opened, want at least %d charges: a grant worth nothing is a turn it spends on the filler instead",
			charged, jabbed, gatedNeed)
	}
	if released == 0 {
		t.Error("the rating never cast the spender at all, so the fuel was banked and never cashed")
	}
}
