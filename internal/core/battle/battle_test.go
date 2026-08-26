package battle_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

func books(t *testing.T) battle.Books {
	t.Helper()
	chart, err := element.ParseChart([]byte(`{
	  "multipliers": {"advantage": 1500, "neutral": 1000, "disadvantage": 667},
	  "cycles": [
	    {"name": "organic", "chain": ["water", "fire", "grass", "ground"]},
	    {"name": "industrial", "chain": ["ice", "metal", "wind", "electric"]},
	    {"name": "cross", "chain": ["water", "metal", "grass", "wind", "fire", "ice", "ground", "electric"]}
	  ],
	  "mutual": [["light", "dark"]],
	  "inert": ["neutral"]
	}`))
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	patterns, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3, "splash_power": 500,
	  "patterns": [
	    {"name": "single", "splash": []},
	    {"name": "column", "splash": [["up"], ["down"]]},
	    {"name": "wedge_left", "splash": [["upper_left"], ["lower_left"]]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "burn", "category": "dot", "max_stacks": 2, "duration": 2, "tick_power": 800},
	    {"id": "weaken", "category": "stat_debuff", "max_stacks": 3, "duration": 3,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": -300}]},
	    {"id": "haste", "category": "buff", "max_stacks": 2, "duration": 3,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 300}]},
	    {"id": "stun", "category": "control", "max_stacks": 1, "duration": 1},
	    {"id": "block", "category": "shield", "max_stacks": 3, "duration": 2},
	    {"id": "fleet", "category": "buff", "max_stacks": 1, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 500}]},
	    {"id": "toughened", "category": "buff", "max_stacks": 2, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 200}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	skills, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"triple","element":"neutral","range":1,"pattern":"single",
	   "power":400,"strikes":3,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"thorn","element":"grass","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"cleave","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","pierce":600},
	  {"id":"venom_edge","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","pierce":1000,
	   "applies":[{"status":"poison","chance":1000}]},
	  {"id":"envenom","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "applies":[{"status":"poison","chance":1000}]},
	  {"id":"scorch","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "applies":[{"status":"burn","chance":1000}]},
	  {"id":"pop","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "requires":{"status":"burn","min_stacks":1,"bonus_power":900,"consume":true}},
	  {"id":"clout","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":3,"target":"enemy"},
	  {"id":"daze","element":"neutral","range":1,"pattern":"single",
	   "power":10,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "applies":[{"status":"stun","chance":1000}]},
	  {"id":"sap","element":"neutral","range":1,"pattern":"single",
	   "power":10,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "applies":[{"status":"weaken","chance":1000}]},
	  {"id":"brace","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"block","chance":1000,"stacks":1}]},
	  {"id":"dash","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"haste","chance":1000}]},
	  {"id":"mend","element":"neutral","range":1,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"ally",
	   "strips":{"categories":["dot","stat_debuff","control"],"stacks":3}},
	  {"id":"quake","element":"neutral","range":2,"pattern":"wedge_left",
	   "power":500,"strikes":1,"accuracy":1000,"cooldown":0,"target":"all"},
	  {"id":"anthem","element":"neutral","range":2,"pattern":"wedge_left",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"all",
	   "applies":[{"status":"haste","chance":1000}]}
	]}`), skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	passives, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"swift","name":"nhanh nhẹn","grants":[{"status":"fleet","stacks":1}]},
	  {"id":"hardy","grants":[{"status":"toughened","stacks":2}]}
	]}`), passive.Deps{Statuses: statuses})
	if err != nil {
		t.Fatalf("passives: %v", err)
	}
	return battle.Books{
		Rules: combat.Rules{DefenseConstant: 300, MinimumDamage: 1, MinHitChance: 150, MaxBlockCharges: 3},
		Chart: chart,
		Bounds: modifier.Bounds{
			Headroom: 3000, FloorFraction: 100, MaxAffinityScale: 1000,
		},
		Limits: progression.Limits{
			LevelCap: progression.LevelCap,
			Ceilings: progression.Values{
				progression.HP: 4800, progression.Attack: 800, progression.Defense: 800,
				progression.Speed: 200, progression.Accuracy: 300, progression.Dodge: 150,
			},
			MaxEffectiveHP: 11500,
		},
		Patterns: patterns, Statuses: statuses, Skills: skills, Passives: passives,
	}
}

func stats(hp, attack, defense, speed int64) progression.Values {
	return progression.Values{
		progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
		progression.Speed: speed, progression.Accuracy: 0, progression.Dodge: 0,
	}
}

func single(id string) element.Affinity {
	member, err := element.Parse(id)
	if err != nil {
		panic(err)
	}
	affinity, err := element.Single(member)
	if err != nil {
		panic(err)
	}
	return affinity
}

// duel is a controlled one on one, both frontline centre, with no accuracy or
// dodge so every roll is decided by the skill alone.
func duel(t *testing.T, allySkills, foeSkills []string, allySpeed, foeSpeed int64) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, allySpeed), Skills: allySkills},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, foeSpeed), Skills: foeSkills},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

func find(events []battle.Event, kind battle.Kind) []battle.Event {
	out := make([]battle.Event, 0, len(events))
	for _, event := range events {
		if event.Kind == kind {
			out = append(out, event)
		}
	}
	return out
}

func TestNewRejects(t *testing.T) {
	valid := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: []string{"strike"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: []string{"strike"}},
	}
	clone := func() []battle.Roster {
		out := make([]battle.Roster, len(valid))
		copy(out, valid)
		return out
	}
	if _, err := battle.New(books(t), 1, valid); err != nil {
		t.Fatalf("the valid roster was rejected: %v", err)
	}

	cases := []struct {
		name    string
		roster  func() []battle.Roster
		wantErr string
	}{
		{"no units", func() []battle.Roster { return nil }, "needs units"},
		{"one side only", func() []battle.Roster { return clone()[:1] }, "no unit is on the enemy side"},
		{"a repeated id", func() []battle.Roster {
			r := clone()
			r[1].ID = "a"
			return r
		}, "listed twice"},
		{"a slot off the formation", func() []battle.Roster {
			r := clone()
			r[0].Slot = hex.Offset{Col: 4, Row: 0}
			return r
		}, "not a formation slot"},
		{"two units in one cell", func() []battle.Roster {
			r := clone()
			r = append(r, battle.Roster{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: []string{"strike"}})
			return r
		}, "both sit at"},
		{"more than a full team", func() []battle.Roster {
			r := clone()
			for i := 0; i < hex.MaxTeamSize; i++ {
				r = append(r, battle.Roster{
					ID: string(rune('m' + i)), Side: hex.SideAlly,
					Slot:     hex.Offset{Col: i % hex.FormationCols, Row: i / hex.FormationCols},
					Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: []string{"strike"},
				})
			}
			return r
		}, "more than"},
		{"no skills", func() []battle.Roster {
			r := clone()
			r[0].Skills = nil
			return r
		}, "has no skills"},
		{"an unknown skill", func() []battle.Roster {
			r := clone()
			r[0].Skills = []string{"nonesuch"}
			return r
		}, "unknown skill"},
		{"the same skill twice", func() []battle.Roster {
			r := clone()
			r[0].Skills = []string{"strike", "strike"}
			return r
		}, "twice"},
		{"a skill of an element the unit lacks", func() []battle.Roster {
			r := clone()
			r[0].Skills = []string{"strike", "thorn"}
			return r
		}, "knows the grass skill"},
		{"stats over the budget", func() []battle.Roster {
			r := clone()
			r[0].Stats = stats(4800, 800, 800, 100)
			return r
		}, "over the budget"},
		{"an affinity that counters itself", func() []battle.Roster {
			r := clone()
			pair, err := element.Dual(element.Water, element.Fire)
			if err != nil {
				t.Fatalf("dual: %v", err)
			}
			r[0].Affinity = pair
			return r
		}, "counter each other"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := battle.New(books(t), 1, testCase.roster())
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestBooksMustBeComplete(t *testing.T) {
	full := books(t)
	for _, testCase := range []struct {
		name  string
		strip func(*battle.Books)
	}{
		{"no chart", func(b *battle.Books) { b.Chart = nil }},
		{"no patterns", func(b *battle.Books) { b.Patterns = nil }},
		{"no statuses", func(b *battle.Books) { b.Statuses = nil }},
		{"no skills", func(b *battle.Books) { b.Skills = nil }},
		{"invalid rules", func(b *battle.Books) { b.Rules.DefenseConstant = 0 }},
		{"invalid bounds", func(b *battle.Books) { b.Bounds.Headroom = 0 }},
		{"invalid limits", func(b *battle.Books) { b.Limits.MaxEffectiveHP = 0 }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			partial := full
			testCase.strip(&partial)
			if _, err := battle.New(partial, 1, nil); err == nil {
				t.Error("a battle was built without complete books")
			}
		})
	}
}

func TestBeginRecordsTheOpeningBoard(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)
	fight.Begin()
	events := fight.Drain()
	opened := find(events, battle.Started)
	if len(opened) != 2 {
		t.Fatalf("%d units were recorded, want 2", len(opened))
	}
	for _, event := range opened {
		if event.Amount <= 0 || event.Note == "" {
			t.Errorf("the opening record for %s is %+v", event.Actor, event)
		}
	}
	if opened[0].Cell == opened[1].Cell {
		t.Error("both units were recorded in the same cell")
	}
	if opened[0].Side == opened[1].Side {
		t.Error("both units were recorded on the same side")
	}
}

// TestTheSameSeedReplaysExactly is the guarantee the whole engine is built for.
func TestTheSameSeedReplaysExactly(t *testing.T) {
	run := func() []battle.Event {
		fight, err := battle.New(books(t), 20240824, []battle.Roster{
			{ID: "a1", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("fire"), Stats: stats(3000, 700, 400, 120),
				Skills: []string{"strike", "scorch", "pop", "brace"}},
			{ID: "a2", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 1},
				Affinity: single("grass"), Stats: stats(2400, 760, 300, 170),
				Skills: []string{"strike", "envenom", "triple"}},
			{ID: "f1", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("grass"), Stats: stats(3200, 640, 500, 100),
				Skills: []string{"strike", "envenom", "daze", "mend"}},
			{ID: "f2", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
				Affinity: single("water"), Stats: stats(2600, 720, 360, 150),
				Skills: []string{"strike", "sap", "dash"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(2000); err != nil {
			t.Fatalf("run: %v", err)
		}
		return fight.Drain()
	}
	first, second := run(), run()
	if len(first) != len(second) {
		t.Fatalf("the two runs produced %d and %d events", len(first), len(second))
	}
	if len(first) < 40 {
		t.Fatalf("the battle produced only %d events, too short to be a real check", len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("event %d differs:\n%+v\n%+v", i, first[i], second[i])
		}
	}
}

// TestControlCostsTheFollowingTurn pins the order inside Advance. A one turn stun
// applied now has to cost the next action; spending its duration before checking
// it would expire it in the very turn it was meant to prevent.
func TestControlCostsTheFollowingTurn(t *testing.T) {
	fight := duel(t, []string{"daze", "jab"}, []string{"strike"}, 100, 100)
	fight.Begin()
	fight.Drain()

	// The ally acts first and dazes the foe.
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("daze", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	if applied := find(fight.Drain(), battle.StatusApplied); len(applied) != 1 || applied[0].Status != "stun" {
		t.Fatalf("the daze applied %+v", applied)
	}

	// The foe's next turn is lost.
	prompt, err = fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "f" || !prompt.Skipped || prompt.Reason != "stun" {
		t.Fatalf("the stunned turn came back as %+v", prompt)
	}
	events := fight.Drain()
	if skipped := find(events, battle.TurnSkipped); len(skipped) != 1 {
		t.Errorf("%d turns were skipped, want 1", len(skipped))
	}
	if expired := find(events, battle.StatusExpired); len(expired) != 1 || expired[0].Status != "stun" {
		t.Errorf("the stun expired as %+v", expired)
	}

	// And the turn after that it acts normally.
	for {
		prompt, err = fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit == "f" {
			break
		}
		if !prompt.Skipped {
			// Jab rather than daze, so the foe is not stunned all over again.
			if err := fight.Act("jab", hex.Offset{Col: 3, Row: 1}); err != nil {
				t.Fatalf("act: %v", err)
			}
		}
		fight.Drain()
	}
	if prompt.Skipped {
		t.Errorf("the foe was still stunned on its next turn: %+v", prompt)
	}
}

// TestPoisonTicksOnlyOnItsHoldersTurn is the join between the turn order and the
// timed effects: a fast attacker acting three times does not advance the poison
// three times.
func TestPoisonTicksOnlyOnItsHoldersTurn(t *testing.T) {
	fight := duel(t, []string{"envenom", "jab"}, []string{"jab"}, 200, 70)
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("envenom", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	fight.Drain()

	allyTurns, foeTurns, ticks := 0, 0, 0
	for i := 0; i < 12 && !fight.Finished(); i++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit == "a" {
			allyTurns++
		} else {
			foeTurns++
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.StatusTicked {
				if event.Actor != "f" {
					t.Errorf("the poison ticked on %s", event.Actor)
				}
				if event.Status != "poison" {
					t.Errorf("the tick named %q", event.Status)
				}
				ticks++
			}
		}
		if !prompt.Skipped {
			choice, ok := fight.Suggest(prompt)
			if ok {
				if err := fight.Act(choice.Skill, choice.Aim); err != nil {
					t.Fatalf("act: %v", err)
				}
			} else {
				fight.Drain()
			}
		}
	}
	if allyTurns <= foeTurns {
		t.Fatalf("the faster unit took %d turns against %d, expected more", allyTurns, foeTurns)
	}
	if ticks != 3 {
		t.Errorf("the poison ticked %d times, want the 3 its duration allows", ticks)
	}
}

// TestABlockChargeCancelsOneStrikeOfMany is block and multi strike meeting in a
// real battle rather than in isolation.
func TestABlockChargeCancelsOneStrikeOfMany(t *testing.T) {
	fight := duel(t, []string{"triple"}, []string{"brace", "jab"}, 70, 200)
	fight.Begin()
	fight.Drain()
	// The faster foe braces first.
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "f" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("brace", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("brace: %v", err)
	}
	fight.Drain()

	for {
		prompt, err = fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit == "a" {
			break
		}
		if !prompt.Skipped {
			// Jab, not brace: a second charge would eat a second strike.
			if err := fight.Act("jab", hex.Offset{Col: 2, Row: 1}); err != nil {
				t.Fatalf("jab: %v", err)
			}
		}
		fight.Drain()
	}
	if err := fight.Act("triple", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("triple: %v", err)
	}
	events := fight.Drain()
	blocked, struck := find(events, battle.Blocked), find(events, battle.Damaged)
	if len(blocked) != 1 {
		t.Errorf("%d strikes were blocked, want 1", len(blocked))
	}
	if len(struck) != 2 {
		t.Errorf("%d strikes landed, want 2", len(struck))
	}
	for _, event := range struck {
		if event.Amount <= 0 {
			t.Errorf("a landing strike dealt %d", event.Amount)
		}
	}
}

func TestCooldownBlocksThenReturns(t *testing.T) {
	fight := duel(t, []string{"clout", "jab"}, []string{"jab"}, 200, 40)
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("clout", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	fight.Drain()

	unavailableFor := 0
	for turn := 0; turn < 8 && !fight.Finished(); turn++ {
		prompt, err = fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit != "a" || prompt.Skipped {
			continue
		}
		available := false
		for _, option := range prompt.Options {
			if option.Skill != "clout" {
				continue
			}
			available = option.Available()
			if !available && !strings.Contains(option.Reason, "cooldown") {
				t.Errorf("the reason given was %q", option.Reason)
			}
		}
		if available {
			if err := fight.Act("clout", hex.Offset{Col: 3, Row: 1}); err != nil {
				t.Errorf("the skill was offered but refused: %v", err)
			}
			break
		}
		unavailableFor++
		if err := fight.Act("clout", hex.Offset{Col: 3, Row: 1}); err == nil {
			t.Error("a skill on cooldown was accepted")
		}
		if err := fight.Act("jab", hex.Offset{Col: 3, Row: 1}); err != nil {
			t.Fatalf("jab: %v", err)
		}
		fight.Drain()
	}
	if unavailableFor != 3 {
		t.Errorf("the skill was unusable for %d of its own turns, want its cooldown of 3", unavailableFor)
	}
}

func TestDetonateAmplifiesAndConsumes(t *testing.T) {
	fight := duel(t, []string{"scorch", "pop"}, []string{"jab"}, 200, 40)
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	_ = prompt
	if err := fight.Act("scorch", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("scorch: %v", err)
	}
	fight.Drain()
	for {
		prompt, err = fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit == "a" && !prompt.Skipped {
			break
		}
		if !prompt.Skipped {
			if err := fight.Act("jab", hex.Offset{Col: 2, Row: 1}); err != nil {
				t.Fatalf("jab: %v", err)
			}
			fight.Drain()
		}
	}
	if err := fight.Act("pop", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("pop: %v", err)
	}
	events := fight.Drain()
	amplified := find(events, battle.Amplified)
	if len(amplified) != 1 {
		t.Fatalf("%d amplifications, want 1", len(amplified))
	}
	if amplified[0].Power != 1000 {
		t.Errorf("the amplified power is %d, want 1000", amplified[0].Power)
	}
	consumed := find(events, battle.StatusConsumed)
	if len(consumed) != 1 || consumed[0].Status != "burn" || consumed[0].Stacks < 1 {
		t.Fatalf("the consumption was %+v", consumed)
	}
	if consumed[0].Amount <= 0 {
		t.Errorf("the consumption reported %d damage given up, want the forgone ticks", consumed[0].Amount)
	}
	foe, _ := fight.Unit("f")
	if foe.Statuses.Has("burn") {
		t.Error("the burn survived being consumed")
	}
}

func TestCleanseStripsWhatItNames(t *testing.T) {
	fight, err := battle.New(books(t), 3, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 500, 400, 200), Skills: []string{"mend", "jab"}},
		{ID: "hurt", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 500, 400, 1), Skills: []string{"jab"}},
		{ID: "foe", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 500, 400, 1), Skills: []string{"envenom", "sap"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	hurt, _ := fight.Unit("hurt")
	statuses := fight.Books().Statuses
	poison, err := statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	weaken, err := statuses.Lookup("weaken")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	hurt.Statuses.Apply(poison, 120)
	hurt.Statuses.Apply(poison, 120)
	hurt.Statuses.Apply(weaken, 0)

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "healer" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("mend", hurt.Cell); err != nil {
		t.Fatalf("mend: %v", err)
	}
	stripped := find(fight.Drain(), battle.StatusStripped)
	if len(stripped) != 1 || stripped[0].Stacks != 3 {
		t.Fatalf("the cleanse was %+v", stripped)
	}
	if hurt.Statuses.Has("poison") || hurt.Statuses.Has("weaken") {
		t.Errorf("the statuses survived: %+v", hurt.Statuses.Snapshot())
	}
}

// TestADebuffMovesTheTurnOrder is the join in the other direction: a status that
// changes tempo has to reorder the queue, or it is worth nothing.
func TestADebuffMovesTheTurnOrder(t *testing.T) {
	fight := duel(t, []string{"dash", "jab"}, []string{"jab"}, 100, 100)
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act("dash", hex.Offset{Col: 2, Row: 1}); err != nil {
		t.Fatalf("dash: %v", err)
	}
	changes := make([]battle.Event, 0, 4)
	for _, event := range fight.Drain() {
		if event.Kind == battle.SpeedChanged && event.Actor == "a" {
			changes = append(changes, event)
		}
	}

	for turn := 0; turn < 6 && !fight.Finished(); turn++ {
		prompt, err = fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.SpeedChanged && event.Actor == "a" {
				changes = append(changes, event)
			}
		}
		if !prompt.Skipped {
			if err := fight.Act("jab", hex.Offset{Col: 3, Row: 1}); err != nil {
				if err := fight.Act("jab", hex.Offset{Col: 2, Row: 1}); err != nil {
					t.Fatalf("jab: %v", err)
				}
			}
			fight.Drain()
		}
	}
	if len(changes) == 0 {
		t.Fatal("the haste never reached the turn order")
	}
	// The first change is the haste taking hold; a later one is it wearing off.
	if first := changes[0]; first.Amount <= first.Before {
		t.Errorf("the haste changed speed from %d to %d, want an increase", first.Before, first.Amount)
	}
}

func TestActRejects(t *testing.T) {
	fight := duel(t, []string{"strike", "clout"}, []string{"strike"}, 200, 40)
	fight.Begin()
	fight.Drain()
	if err := fight.Act("strike", hex.Offset{Col: 3, Row: 1}); err == nil {
		t.Error("acting before a turn began was accepted")
	}
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if _, err := fight.Advance(); err == nil {
		t.Error("advancing twice without acting was accepted")
	}
	if err := fight.Act("nonesuch", hex.Offset{Col: 3, Row: 1}); err == nil {
		t.Error("an unknown skill was accepted")
	}
	if err := fight.Act("strike", hex.Offset{Col: 5, Row: 0}); err == nil {
		t.Error("an unreachable cell was accepted")
	}
	if err := fight.Act("strike", hex.Offset{Col: 2, Row: 1}); err == nil {
		t.Error("an enemy skill aimed at the caster's own cell was accepted")
	}
	if err := fight.Act("strike", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("the legal action was refused: %v", err)
	}
	if err := fight.Act("strike", hex.Offset{Col: 3, Row: 1}); err == nil {
		t.Error("acting twice in one turn was accepted")
	}
}

func TestABattleEndsWhenASideIsGone(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"jab"}, 200, 80)
	fight.Begin()
	turns, err := fight.RunToEnd(4000)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !fight.Finished() {
		t.Fatalf("the battle did not finish in %d turns", turns)
	}
	winner, decided := fight.Winner()
	if !decided || winner != hex.SideAlly {
		t.Errorf("the winner is %v, decided %v, want the ally side", winner, decided)
	}
	events := fight.Drain()
	if died := find(events, battle.Died); len(died) != 1 || died[0].Actor != "f" {
		t.Errorf("the deaths were %+v", died)
	}
	ended := find(events, battle.Ended)
	if len(ended) != 1 || ended[0].Note != "ally" {
		t.Errorf("the battle ended as %+v", ended)
	}
	if _, err := fight.Advance(); err == nil {
		t.Error("a finished battle offered another turn")
	}
	if err := fight.Act("strike", hex.Offset{Col: 3, Row: 1}); err == nil {
		t.Error("a finished battle accepted an action")
	}
	foe, _ := fight.Unit("f")
	if !foe.Dead || foe.HP != 0 {
		t.Errorf("the loser is %+v", foe)
	}
	if fight.Queue().Has("f") {
		t.Error("the dead unit is still in the turn order")
	}
}

func TestEventKindNames(t *testing.T) {
	for kind := 0; kind < battle.KindCount; kind++ {
		if name := battle.Kind(kind).String(); name == "" || strings.HasPrefix(name, "kind(") {
			t.Errorf("kind %d has no name, it renders as %q", kind, name)
		}
	}
	if got := battle.Kind(200).String(); !strings.Contains(got, "200") {
		t.Errorf("an undeclared kind renders as %q", got)
	}
}

// TestNewRefusesASkillKeptForAnotherElement is the one half of a restriction
// the engine can enforce, and it enforces it through the same
// skill.WhyCannotCarry the authoring layer calls.
//
// The other two halves — an archetype and a character identity — are absent
// from a roster entry on purpose, because both are resolved before a battle
// starts. Nothing here should grow a field for either: a restriction the engine
// cannot see is an authoring rule, and cast.ParseBook is where it lives.
func TestNewRefusesASkillKeptForAnotherElement(t *testing.T) {
	shared := books(t)
	restricted, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"oath","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "restrict":{"elements":["grass"]}}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = restricted

	roster := func(affinity string) []battle.Roster {
		return []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single(affinity), Stats: stats(3000, 800, 400, 100), Skills: []string{"oath"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100), Skills: []string{"strike"}},
		}
	}
	_, err = battle.New(shared, 1, roster("neutral"))
	if err == nil {
		t.Fatal("a unit carrying a skill kept for another element was enlisted")
	}
	// The refusal has to be the restriction's rather than the element's: the
	// unit does share the skill's element, since the skill is neutral.
	for _, want := range []string{"oath", "grass"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
	if _, err := battle.New(shared, 1, roster("grass")); err != nil {
		t.Fatalf("a grass unit was refused a skill kept for grass: %v", err)
	}
}

// TestASkillAimedAtBothSidesReachesBoth is the engine half of the "all" target
// side, which is what makes the chooser's fourth option mean anything.
//
// Three things are asserted together because they are one change: the legal aims
// admit a cell on either half, the shape spreads across the midline instead of
// stopping at it, and what is caught on the caster's own half really is hurt.
// That last one is the point of the value rather than a flaw in it — a skill that
// says it hits everybody and quietly spares your own squad is worse than no
// skill.
func TestASkillAimedAtBothSidesReachesBoth(t *testing.T) {
	// The ally holds the area skill aimed at both halves; the foe holds a jab so
	// the fight is a fight.
	fight := duel(t, []string{"quake"}, []string{"jab"}, 120, 100)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the faster unit is %q, want the one holding the area skill", prompt.Unit)
	}
	if len(prompt.Options) != 1 {
		t.Fatalf("the caster was offered %d options, want the one skill", len(prompt.Options))
	}
	// Both occupied cells are legal aims: the foe's, and the caster's own.
	// hex.Cells is column-major, so the caster's own column comes first.
	want := []hex.Offset{{Col: 2, Row: 1}, {Col: 3, Row: 1}}
	if got := prompt.Options[0].Aims; !equalCells(got, want) {
		t.Errorf("an all-sided skill may be aimed at %v, want %v", got, want)
	}

	// Aimed at the foe, and the shape's two left-hand cells land back on the
	// caster's own half — where Targets would have dropped them.
	if err := fight.Act("quake", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("act: %v", err)
	}
	events := fight.Drain()
	hurt := make(map[string]int64)
	for _, event := range find(events, battle.Damaged) {
		hurt[event.Target] += event.Amount
	}
	if hurt["f"] == 0 {
		t.Error("the skill did not hurt the unit it was aimed at")
	}
	if hurt["a"] == 0 {
		t.Error("the skill spared the caster's own half, so it did not cross the midline")
	}
	// The primary takes the whole power and a splash cell the book's share, so
	// the caster — caught by the splash — takes less than the aim.
	if hurt["a"] >= hurt["f"] {
		t.Errorf("the splash cell took %d against the aim's %d, want less", hurt["a"], hurt["f"])
	}
	// And the cap still holds: the shape covers three cells at most, whichever
	// side they are on.
	if targets := len(find(events, battle.Damaged)); targets > 3 {
		t.Errorf("the shape struck %d cells, over the book's limit of 3", targets)
	}
}

// TestTheOtherSidesStillStopAtTheMidline is the other half of the same change:
// admitting one skill across the line must not move any other skill's bounds.
func TestTheOtherSidesStillStopAtTheMidline(t *testing.T) {
	fight := duel(t, []string{"strike", "mend", "brace"}, []string{"jab"}, 120, 100)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	wanted := map[string][]hex.Offset{
		// Aimed at the enemy: only the far half.
		"strike": {{Col: 3, Row: 1}},
		// Aimed at an ally: only the near half, the caster included.
		"mend": {{Col: 2, Row: 1}},
		// Aimed at itself: its own cell and nothing else.
		"brace": {{Col: 2, Row: 1}},
	}
	for _, option := range prompt.Options {
		want, known := wanted[option.Skill]
		if !known {
			t.Fatalf("the caster was offered %q, which this test does not know", option.Skill)
		}
		if !equalCells(option.Aims, want) {
			t.Errorf("%s may be aimed at %v, want %v", option.Skill, option.Aims, want)
		}
	}
}

// TestSuggestLeavesAnAllSidedAttackAlone records what the shallow opponent does
// with the new value, which is nothing.
//
// Suggest rates a skill only if it is aimed at the enemy, so a damaging skill
// aimed at both halves is skipped outright — it is not chosen and not rated. It
// is not a fallback either, because that is only for a skill with no power at
// all. So the opponent will not bomb its own squad, and it will not use the skill
// at any other time either: an all-sided attack is dead weight to the AI and a
// player's tool.
//
// That is a finding rather than a design, and it is pinned here so it cannot
// change by accident. A deeper opponent — see README's roadmap — has to weigh
// what it would do to its own side, and battle.expected deliberately skips a
// unit on the caster's side rather than subtracting it, so relaxing this guard
// alone would produce exactly the opponent that bombs its own team and calls it
// a gain.
func TestSuggestLeavesAnAllSidedAttackAlone(t *testing.T) {
	// The area skill is worth eight times the jab, so anything that rated it
	// would take it.
	fight := duel(t, []string{"quake", "jab"}, []string{"jab"}, 120, 100)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	if choice.Skill != "jab" {
		t.Errorf("Suggest picked %q, want the enemy-aimed jab: a damaging all-sided "+
			"skill is skipped, not rated", choice.Skill)
	}

	// A skill with no power is a different case and does get used, because the
	// fallback does not ask which side it aims at — which is what a
	// battlefield-wide buff or cleanse needs.
	quiet := duel(t, []string{"anthem"}, []string{"jab"}, 120, 100)
	prompt, err = quiet.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok = quiet.Suggest(prompt)
	if !ok || choice.Skill != "anthem" {
		t.Errorf("Suggest picked %q (offered: %v), want the all-sided buff",
			choice.Skill, ok)
	}
}

func equalCells(got, want []hex.Offset) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestASkillNameReachesNoDecision is the layer rule that lets a skill carry a
// display name at all, and it is measured as behaviour rather than by grepping
// for the field.
//
// A skill's name is text for a screen. If any rule ever read it, internal/core
// would be deciding something about a language, and a battle would stop being a
// pure function of its integer arguments in the way that is hardest to see. So
// the same seed and the same roster are played out twice over two books that
// differ in nothing but every skill's name, and the two event logs have to be
// identical event for event — which also covers the log, since a name that
// reached one would show up here.
func TestASkillNameReachesNoDecision(t *testing.T) {
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 110),
			Skills: []string{"strike", "envenom", "clout", "brace"}},
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 0},
			Affinity: single("grass"), Stats: stats(2600, 700, 300, 95),
			Skills: []string{"thorn", "mend", "quake"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("fire"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"scorch", "pop", "daze"}},
		{ID: "g", Side: hex.SideEnemy, Slot: hex.Offset{Col: 1, Row: 2},
			Affinity: single("neutral"), Stats: stats(2400, 650, 350, 105),
			Skills: []string{"triple", "sap", "dash"}},
	}
	play := func(t *testing.T, deck battle.Books) []battle.Event {
		t.Helper()
		fight, err := battle.New(deck, 31, roster)
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		if _, err := fight.RunToEnd(400); err != nil {
			t.Fatalf("run: %v", err)
		}
		return fight.Drain()
	}

	plain := books(t)
	named := books(t)
	named.Skills = withNames(t, named)

	// The names really are there, or this measures two identical books.
	changed := 0
	for _, one := range named.Skills.Skills() {
		before, err := plain.Skills.Lookup(one.ID)
		if err != nil {
			t.Fatalf("lookup %s: %v", one.ID, err)
		}
		if before.Name != "" {
			t.Fatalf("%s already had a name, so this measures nothing", one.ID)
		}
		if one.Name == "" {
			t.Fatalf("%s was not given a name", one.ID)
		}
		changed++
	}
	if changed == 0 {
		t.Fatal("no skill was renamed")
	}

	without, with := play(t, plain), play(t, named)
	if len(without) != len(with) {
		t.Fatalf("naming the skills changed the battle's length: %d events against %d",
			len(with), len(without))
	}
	for i := range without {
		if without[i] != with[i] {
			t.Fatalf("event %d differs once the skills are named:\n%+v\n%+v",
				i, without[i], with[i])
		}
	}
}

// withNames is the book it is given with a display name added to every skill,
// built by rewriting the book's own marshalled form so that nothing but the name
// can differ.
func withNames(t *testing.T, deck battle.Books) *skill.Book {
	t.Helper()
	raw, err := deck.Skills.Marshal()
	if err != nil {
		t.Fatalf("marshal the skill book: %v", err)
	}
	var file struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode the skill book: %v", err)
	}
	for i, declared := range file.Skills {
		// Vietnamese, because that is what these names really are, and because a
		// multi-byte name is the one a naive comparison somewhere would trip on.
		declared["name"] = fmt.Sprintf("chiêu số %d", i+1)
	}
	grown, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("encode the skill book: %v", err)
	}
	book, err := skill.ParseBook(grown,
		skill.Deps{Patterns: deck.Patterns, Statuses: deck.Statuses})
	if err != nil {
		t.Fatalf("the named book does not load: %v", err)
	}
	return book
}

// TestPiercingGoesThroughArmourAndTheLogSaysSo is the mechanism in a real
// battle: the same power against the same defender, once with armour in the way
// and once with most of it ignored.
//
// The log carrying the share is half the test rather than a detail. A reader
// holding the attacker's stats, the power and the multiplier can reproduce every
// other damage figure the engine emits, and a pierced hit that said nothing
// would be the one they could not — which is the log failing at the only job it
// has.
func TestPiercingGoesThroughArmourAndTheLogSaysSo(t *testing.T) {
	hit := func(skillID string) battle.Event {
		t.Helper()
		fight := duel(t, []string{skillID}, []string{"jab"}, 200, 70)
		fight.Begin()
		fight.Drain()
		if _, err := fight.Advance(); err != nil {
			t.Fatalf("advance: %v", err)
		}
		if err := fight.Act(skillID, hex.Offset{Col: 3, Row: 1}); err != nil {
			t.Fatalf("act: %v", err)
		}
		damaged := find(fight.Drain(), battle.Damaged)
		if len(damaged) != 1 {
			t.Fatalf("%s produced %d damage events, want 1", skillID, len(damaged))
		}
		return damaged[0]
	}

	// strike and cleave are the same skill but for the piercing, so the two
	// figures differ by nothing else.
	plain := hit("strike")
	pierced := hit("cleave")
	if pierced.Amount <= plain.Amount {
		t.Errorf("piercing 600 dealt %d against the %d a plain strike dealt",
			pierced.Amount, plain.Amount)
	}
	if plain.Pierce != 0 {
		t.Errorf("a strike that pierces nothing logged a pierce of %d", plain.Pierce)
	}
	if pierced.Pierce != 600 {
		t.Errorf("the pierced hit logged a pierce of %d, want the skill's 600", pierced.Pierce)
	}
	// The figure has to be the one the logged terms produce, or the log is a
	// story about the hit rather than a record of it.
	rules := books(t).Rules
	want := rules.Damage(800, combat.Pierced(400, pierced.Pierce), pierced.Power, pierced.Multiplier)
	if pierced.Amount != want {
		t.Errorf("the log says %d damage but its own terms give %d", pierced.Amount, want)
	}
}

// TestAPiercingSkillDoesNotPierceTheStatusItApplies is the decision this
// mechanism turns on, and the reason it is a test rather than a comment.
//
// A tick's damage is computed once, when the stack is applied, and frozen for
// the stack's whole life. So piercing a tick would be worth as many pierced hits
// as the stack has turns left — three, here — which is a far larger effect than
// the per-strike ratio an author wrote. venom_edge pierces the armour outright
// and its poison still has to tick for exactly what envenom's does.
func TestAPiercingSkillDoesNotPierceTheStatusItApplies(t *testing.T) {
	applied := func(skillID string) battle.Event {
		t.Helper()
		fight := duel(t, []string{skillID}, []string{"jab"}, 200, 70)
		fight.Begin()
		fight.Drain()
		if _, err := fight.Advance(); err != nil {
			t.Fatalf("advance: %v", err)
		}
		if err := fight.Act(skillID, hex.Offset{Col: 3, Row: 1}); err != nil {
			t.Fatalf("act: %v", err)
		}
		events := find(fight.Drain(), battle.StatusApplied)
		if len(events) != 1 {
			t.Fatalf("%s applied %d statuses, want 1", skillID, len(events))
		}
		return events[0]
	}

	plain := applied("envenom")
	pierced := applied("venom_edge")
	if plain.Amount == 0 {
		t.Fatalf("the plain poison froze a tick of nothing, so this proves nothing")
	}
	if pierced.Amount != plain.Amount {
		t.Errorf("the poison of a piercing skill ticks for %d against the %d a plain one ticks for",
			pierced.Amount, plain.Amount)
	}
	// And the figure is the one full defence gives, not merely equal to another
	// wrong one: both would be pierced if Pierced had been folded into the
	// defence the caller passes.
	rules := books(t).Rules
	kind, err := books(t).Statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("poison: %v", err)
	}
	if want := rules.Damage(800, 400, kind.TickPower, 1000); pierced.Amount != want {
		t.Errorf("the frozen tick is %d, want the %d full defence gives", pierced.Amount, want)
	}
}

// TestAPassiveIsInForceBeforeTheFirstWaitIsComputed is the trap this feature was
// warned about in advance, and the only test here that would catch it.
//
// A wait is 1_000_000/speed, and the queue is built while a unit is being
// enlisted. So a trait touching speed has to be on the unit *before* that,
// because a wait computed from the base line is wrong for the whole battle and
// nothing later recomputes the one already served. retuneAll exists because
// exactly this was got wrong once with haste.
//
// The holder is the *slower* unit at its base, and faster only once the trait is
// counted: 100 against 130, and 150 against 130 with the trait on. So it takes
// the first turn if and only if the trait was in force when the queue was built.
//
// Equal speeds do not test this, and that was the first version of it. With both
// at 100 the holder wins the tie-break and goes first whether the trait was
// counted or not, so granting after the queue was built passed clean.
func TestAPassiveIsInForceBeforeTheFirstWaitIsComputed(t *testing.T) {
	const (
		holderSpeed = 100
		rivalSpeed  = 130
	)
	fight := mustBattle(t, books(t), 3, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, holderSpeed),
			Skills: []string{"strike"}, Passives: []string{"swift"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, rivalSpeed),
			Skills: []string{"strike"}},
	})
	fight.Begin()

	holder, _ := fight.Unit("a")
	buffed := fight.Stats(holder)[progression.Speed]
	if buffed <= rivalSpeed {
		t.Fatalf("the trait resolves to speed %d against the rival's %d, so the fixtures cannot tell the two orders apart",
			buffed, rivalSpeed)
	}

	// The first turn is the one a wait computed after the fact cannot fix: it has
	// already been served by the time anything would notice.
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Errorf("the first turn went to %s, want the slower unit whose trait makes it faster", prompt.Unit)
	}
	// And it is not a correction after the fact either. A SpeedChanged on the
	// first turn would mean the queue was built from the base line and then
	// patched, which is the failure this test is about even when the order comes
	// out right.
	for _, event := range find(fight.Drain(), battle.SpeedChanged) {
		if event.Actor == "a" {
			t.Errorf("the holder's speed was corrected from %d to %d on the first turn, so the queue was built without the trait",
				event.Before, event.Amount)
		}
	}
}

// TestAHeldPassiveIsOnTheBoardAndInTheLog is the other half of the constraint: a
// passive that changes a number has to say so, or the log stops being able to
// explain its own figures.
func TestAHeldPassiveIsOnTheBoardAndInTheLog(t *testing.T) {
	fight := mustBattle(t, books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"strike"}, Passives: []string{"hardy"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"strike"}},
	})
	holder, _ := fight.Unit("a")
	// On before Begin, because Begin reports rather than applies.
	if got := holder.Statuses.Stacks("toughened"); got != 2 {
		t.Errorf("the holder carries %d stacks before Begin, want the 2 the trait grants", got)
	}
	if got := fight.Stats(holder)[progression.Defense]; got <= holder.Base[progression.Defense] {
		t.Errorf("the trait resolved to defence %d against a base of %d", got, holder.Base[progression.Defense])
	}

	fight.Begin()
	events := find(fight.Drain(), battle.PassiveHeld)
	if len(events) != 1 {
		t.Fatalf("the opening board carries %d passive events, want 1", len(events))
	}
	held := events[0]
	switch {
	case held.Actor != "a":
		t.Errorf("the event names %q as the holder", held.Actor)
	case held.Passive != "hardy":
		t.Errorf("the event names the trait %q, want hardy", held.Passive)
	case held.Status != "toughened":
		t.Errorf("the event names the status %q, want toughened", held.Status)
	case held.Stacks != 2:
		t.Errorf("the event says %d stacks, want 2", held.Stacks)
	}
	// The unit with no trait gets no line, or the log claims a trait nobody has.
	for _, event := range events {
		if event.Actor == "f" {
			t.Error("a unit holding no trait was reported as holding one")
		}
	}
}

// TestAPassiveSurvivesEverythingThatEndsAStatus is what "permanent" has to mean
// in a real battle rather than in the status package's own tests.
//
// A trait granted only at enlistment and then dispelled would be off for the
// rest of the battle with no way back, which is a far larger effect than
// stripping a buff somebody cast — so a cleanse must not reach it, and the turns
// going by must not either.
func TestAPassiveSurvivesEverythingThatEndsAStatus(t *testing.T) {
	fight := mustBattle(t, books(t), 11, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"mend", "jab"}, Passives: []string{"hardy"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 60),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	holder, _ := fight.Unit("a")

	// mend strips dot, stat_debuff and control rather than buffs, so aim the
	// cleanse at the trait directly instead: it is the mechanism under test, not
	// the skill.
	if removed, _ := holder.Statuses.Remove("toughened", 2); removed != 0 {
		t.Errorf("a dispel took %d stacks off a trait", removed)
	}
	if got := holder.Statuses.Cleanse([]status.Category{status.Buff}, 5); got != 0 {
		t.Errorf("a cleanse took %d stacks off a trait", got)
	}
	if stacks, _ := holder.Statuses.Consume("toughened"); stacks != 0 {
		t.Errorf("consuming took %d stacks off a trait", stacks)
	}

	// And a whole battle's worth of turns, which is what expires anything timed.
	for i := 0; i < 40 && !fight.Finished(); i++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if !prompt.Skipped {
			if choice, ok := fight.Suggest(prompt); ok {
				if err := fight.Act(choice.Skill, choice.Aim); err != nil {
					t.Fatalf("act: %v", err)
				}
			}
		}
		fight.Drain()
	}
	if got := holder.Statuses.Stacks("toughened"); got != 2 {
		t.Errorf("after a battle's worth of turns the trait is down to %d stacks, want 2", got)
	}
}

// TestNewRefusesAPassiveItCannotHonour keeps a data mistake at the load rather
// than at the moment it would have mattered, the way every other book does.
func TestNewRefusesAPassiveItCannotHonour(t *testing.T) {
	entry := func(passives ...string) []battle.Roster {
		return []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"strike"}, Passives: passives},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"strike"}},
		}
	}
	if _, err := battle.New(books(t), 1, entry("nobody-wrote-this")); err == nil {
		t.Error("a unit holding an undeclared trait was enlisted")
	}
	if _, err := battle.New(books(t), 1, entry("hardy", "hardy")); err == nil {
		t.Error("a unit holding the same trait twice was enlisted")
	}
	// Without the book, a trait cannot be honoured, and enlisting anyway would
	// put a unit on the board quietly missing what it was built with.
	bare := books(t)
	bare.Passives = nil
	if _, err := battle.New(bare, 1, entry("hardy")); err == nil {
		t.Error("a unit holding a trait was enlisted with no passive book")
	}
	// And a battle whose units hold none still runs without one, which is what
	// let the field arrive without every caller being found.
	if _, err := battle.New(bare, 1, entry()); err != nil {
		t.Errorf("a battle with no traits refused to start without the book: %v", err)
	}
}

// mustBattle is battle.New for a test that has nothing to say about failure.
func mustBattle(t *testing.T, books battle.Books, seed uint64, roster []battle.Roster) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books, seed, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}
