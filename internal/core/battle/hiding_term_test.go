// What hiding is worth, read as a figure rather than as a decision.
//
// This file is in package battle rather than battle_test, and it is here for the
// reason reserveheal_test.go and carry_wall_test.go give: `hidden` has no
// consequence a fixture can watch for. What it changes is which skill Suggest
// would rather cast, and a test reading the choice measures every other term in
// the rating at the same time — hiding_price_test.go holds the two decisions, and
// what is left over from those is the arithmetic, which only the figure can say.
//
// ⚠️ The horizon is the half a decision test cannot reach at all. Dropping
// `× turnsOf(kind, buffHorizon)` from the term halves every price it produces and
// reddens no decision on any board that is not tuned to the exact rung where the
// halving flips a comparison — which is a fixture this repository has learned not
// to write. Asked directly, it is one equality.
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

func hidingBooks(t *testing.T) Books {
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
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [{"id": "burrowed", "category": "buff", "max_stacks": 1, "duration": 2}]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	skills, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"nudge","element":"neutral","range":1,"pattern":"single",
	   "power":40,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"graze","element":"neutral","range":1,"pattern":"single",
	   "power":40,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"burrow","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"burrowed","chance":1000,"stacks":1}]}
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

// aHiderAndAnAlly is one hider, one ally whose toughness the caller chooses, and
// one attacker that can reach both.
//
// ⚠️ **The attacker's speed is the caller's, because the term reads it.** A hide
// counts down on its holder's turns and denies the attacker's, so the two speeds
// are half the arithmetic; the fixture that leaves them equal is the one where
// the conversion is a multiplication by one and the rest of the equality can be
// read on its own.
func aHiderAndAnAlly(t *testing.T, allyDefence, hitterSpeed int64) (*Battle, *Unit, *Unit, *Unit) {
	t.Helper()
	fight, err := New(hidingBooks(t), 7, []Roster{
		{ID: "hider", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, 60),
			Skills: []string{"burrow", "jab"}},
		{ID: "ally", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: neutralOnly(t), Stats: healStats(3000, 500, allyDefence, 2),
			Skills: []string{"jab"}},
		{ID: "hitter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, hitterSpeed),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	hider, _ := fight.Unit("hider")
	ally, _ := fight.Unit("ally")
	hitter, _ := fight.Unit("hitter")
	return fight, hider, ally, hitter
}

// TestHidingIsWorthTheDifferenceOverTheTurnsItLasts is the arithmetic, as one
// equality.
//
// The ally is tougher than the hider, so the attacker prefers the hider and the
// denial is real; what it is worth is how much better the hider was as a target,
// for as long as the hiding lasts — converted into the attacker's own turns,
// which at equal speeds is a multiplication by one and is why this fixture leaves
// them equal.
func TestHidingIsWorthTheDifferenceOverTheTurnsItLasts(t *testing.T) {
	fight, hider, ally, hitter := aHiderAndAnAlly(t, 800, 60)
	prices := fight.newPricing()

	onHider := fight.bestAgainst(hitter, hider)
	onAlly := fight.bestAgainst(hitter, ally)
	if onHider <= onAlly {
		t.Fatalf("the attacker would rather hit the ally (%d) than the hider (%d), so this "+
			"fixture measures the nought case rather than the arithmetic", onAlly, onHider)
	}
	// The cap is the OTHER half of the term and is measured on its own below, so
	// this fixture has to be one it does not bind in.
	if ordinary := prices.turnWorth(hitter); ordinary < onHider-onAlly {
		t.Fatalf("an ordinary turn of the attacker's is worth %d against a denied "+
			"difference of %d, so this fixture is reading the cap rather than the "+
			"difference", ordinary, onHider-onAlly)
	}

	kind := status.Kind{ID: burrowStatus, Category: status.Buff, MaxStacks: 1, Duration: 2}
	want := (onHider - onAlly) * turnsOf(kind, buffHorizon)
	if got := prices.hidden(hider, kind); got != want {
		t.Errorf("hiding is priced at %d, want the denied difference %d over %d turns = %d",
			got, onHider-onAlly, turnsOf(kind, buffHorizon), want)
	}

	// And the horizon carries its own weight: twice the turns is twice the
	// denial, which is the multiplier stated on its own so that dropping it
	// reddens something even if the equality above were ever loosened.
	brief := status.Kind{ID: burrowStatus, Category: status.Buff, MaxStacks: 1, Duration: 1}
	short, long := prices.hidden(hider, brief), prices.hidden(hider, kind)
	if turnsOf(brief, buffHorizon) != 1 || turnsOf(kind, buffHorizon) != 2 {
		t.Fatalf("the two kinds last %d and %d turns, and this comparison needs 1 and 2",
			turnsOf(brief, buffHorizon), turnsOf(kind, buffHorizon))
	}
	if short <= 0 || long != 2*short {
		t.Errorf("one turn of hiding is worth %d and two are worth %d, want exactly twice: "+
			"the term is not counting the turns it lasts", short, long)
	}
}

// TestHidingIsCountedInTheAttackersTurnsAndNotTheHolders is the conversion,
// asked as a figure because no decision on any board can see it on its own.
//
// A hiding status counts down on its HOLDER's turns, so the horizon above is the
// holder's — and what a hide denies is the attacker's. Charging the horizon
// straight through billed a fast hider the attacker's whole blow once per turn of
// a window the attacker might never act in, which made hiding the largest figure
// on the board and the only turn such a unit ever took: measured on the shipped
// Diglett, the split build reads 725‰ against a Machop, the same build with
// `burrow` in a slot reads 110‰, and with that slot simply empty 780‰.
//
// Half the speed is half the denial, and the fixture picks a ratio the division
// is exact at so the statement is an equality rather than a bound.
func TestHidingIsCountedInTheAttackersTurnsAndNotTheHolders(t *testing.T) {
	kind := status.Kind{ID: burrowStatus, Category: status.Buff, MaxStacks: 1, Duration: 2}

	fight, hider, _, _ := aHiderAndAnAlly(t, 800, 60)
	together := fight.newPricing().hidden(hider, kind)
	if together <= 0 {
		t.Fatalf("hiding at equal speeds is priced at %d, so there is nothing to halve",
			together)
	}

	half, halfHider, _, _ := aHiderAndAnAlly(t, 800, 30)
	if got := half.newPricing().hidden(halfHider, kind); got != together/2 {
		t.Errorf("an attacker at half the holder's speed is denied %d, want half of %d: "+
			"the window is being counted in the holder's turns", got, together)
	}
}

// TestHidingFromAnAttackerWithABetterTargetIsWorthNothing is the nought case, and
// it is the one that stops the term becoming "always hide".
//
// An attacker whose best blow was aimed at somebody else loses nothing when this
// unit goes underground — it hits that somebody else, exactly as it was going to.
func TestHidingFromAnAttackerWithABetterTargetIsWorthNothing(t *testing.T) {
	fight, hider, ally, hitter := aHiderAndAnAlly(t, 1, 60)
	prices := fight.newPricing()

	onHider := fight.bestAgainst(hitter, hider)
	onAlly := fight.bestAgainst(hitter, ally)
	if onAlly < onHider {
		t.Fatalf("the attacker prefers the hider (%d) to the ally (%d), so this fixture "+
			"measures the arithmetic rather than the nought case", onHider, onAlly)
	}

	kind := status.Kind{ID: burrowStatus, Category: status.Buff, MaxStacks: 1, Duration: 2}
	if got := prices.hidden(hider, kind); got != 0 {
		t.Errorf("hiding from an attacker with a better target is priced at %d, want nought: "+
			"the whole attack is being denied rather than the difference", got)
	}
}

// TestHidingIsCappedAtAnOrdinaryTurnOfTheAttackers is the other half of the
// term, and it is the correction turnWorth is already the written statement of.
//
// A denied turn is not another cast of the attacker's heaviest skill — that one
// is on cooldown most of the time — it is an ordinary turn of that attacker's.
// The cap is what stops a kit holding one big skill and three small ones from
// billing its whole best blow for every turn of a hide.
//
// The two arms hold the attacker's BEST blow fixed and move only the rest of its
// kit, so what is read is the mean rather than the maximum. Without the cap both
// arms price identically, which is what makes this a test rather than a
// restatement.
func TestHidingIsCappedAtAnOrdinaryTurnOfTheAttackers(t *testing.T) {
	kind := status.Kind{ID: burrowStatus, Category: status.Buff, MaxStacks: 1, Duration: 2}

	// One skill, so an ordinary turn of this attacker's is its heaviest blow and
	// the cap cannot bind.
	narrow, hider, _, _ := aHiderAndAnAllyFacing(t, []string{"jab"})
	uncapped := narrow.newPricing().hidden(hider, kind)
	if uncapped <= 0 {
		t.Fatalf("hiding is priced at %d against a single-skill attacker, so there is "+
			"nothing for the cap to bite into", uncapped)
	}

	// The same heaviest blow with a small skill beside it. Nothing about what
	// this attacker can do to the holder at its best has changed.
	broad, wider, _, hitter := aHiderAndAnAllyFacing(t, []string{"jab", "nudge", "graze"})
	prices := broad.newPricing()
	if ordinary, best := prices.turnWorth(hitter), prices.strike(hitter); ordinary >= best {
		t.Fatalf("an ordinary turn of this attacker's is worth %d against a best blow of "+
			"%d, so the fixture holds nothing for the cap to be about", ordinary, best)
	}
	if got := prices.hidden(wider, kind); got >= uncapped {
		t.Errorf("hiding from an attacker with a small skill beside its big one is priced "+
			"at %d against %d for the big one alone: the term is still reading the "+
			"heaviest blow", got, uncapped)
	}
}

// aHiderAndAnAllyFacing is aHiderAndAnAlly with the attacker's kit chosen, for
// the one property that is about that kit rather than about its best blow.
func aHiderAndAnAllyFacing(t *testing.T, hitterSkills []string) (*Battle, *Unit, *Unit, *Unit) {
	t.Helper()
	fight, err := New(hidingBooks(t), 7, []Roster{
		{ID: "hider", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, 60),
			Skills: []string{"burrow", "jab"}},
		{ID: "ally", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: neutralOnly(t), Stats: healStats(3000, 500, 800, 2),
			Skills: []string{"jab"}},
		{ID: "hitter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, 60),
			Skills: hitterSkills},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	hider, _ := fight.Unit("hider")
	ally, _ := fight.Unit("ally")
	hitter, _ := fight.Unit("hitter")
	return fight, hider, ally, hitter
}
