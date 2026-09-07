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
func aHiderAndAnAlly(t *testing.T, allyDefence int64) (*Battle, *Unit, *Unit, *Unit) {
	t.Helper()
	fight, err := New(hidingBooks(t), 7, []Roster{
		{ID: "hider", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, 60),
			Skills: []string{"burrow", "jab"}},
		{ID: "ally", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: neutralOnly(t), Stats: healStats(3000, 500, allyDefence, 2),
			Skills: []string{"jab"}},
		{ID: "hitter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: neutralOnly(t), Stats: healStats(4800, 500, 300, 1),
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
// for as long as the hiding lasts.
func TestHidingIsWorthTheDifferenceOverTheTurnsItLasts(t *testing.T) {
	fight, hider, ally, hitter := aHiderAndAnAlly(t, 800)
	prices := fight.newPricing()

	onHider := fight.bestAgainst(hitter, hider)
	onAlly := fight.bestAgainst(hitter, ally)
	if onHider <= onAlly {
		t.Fatalf("the attacker would rather hit the ally (%d) than the hider (%d), so this "+
			"fixture measures the nought case rather than the arithmetic", onAlly, onHider)
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

// TestHidingFromAnAttackerWithABetterTargetIsWorthNothing is the nought case, and
// it is the one that stops the term becoming "always hide".
//
// An attacker whose best blow was aimed at somebody else loses nothing when this
// unit goes underground — it hits that somebody else, exactly as it was going to.
func TestHidingFromAnAttackerWithABetterTargetIsWorthNothing(t *testing.T) {
	fight, hider, ally, hitter := aHiderAndAnAlly(t, 1)
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
