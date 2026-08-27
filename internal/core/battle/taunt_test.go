package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// tauntBooks is the shared books with a skill that taunts, one that reaches only
// the next cell, one that reaches across the board, and one aimed at an ally.
func tauntBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	provoking, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"provoke","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "self_applies":[{"status":"taunting","chance":1000,"stacks":1}]},
	  {"id":"poke","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"snipe","element":"neutral","range":5,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"guard","element":"neutral","range":2,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"ally",
	   "applies":[{"status":"haste","chance":1000,"stacks":1}]}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = provoking
	return shared
}

// The two enemy slots. "tank" stands at the front where a range-1 skill can
// already reach it; "squishy" stands at the back where one cannot, so that the
// attacker's own preference and the taunt pull in opposite directions.
var (
	tankSlot    = hex.Offset{Col: 2, Row: 1}
	squishySlot = hex.Offset{Col: 0, Row: 1}
)

// aTauntedField is an attacker facing a tank and a squishy, with the tank about
// to provoke.
func aTauntedField(t *testing.T) *battle.Battle {
	t.Helper()
	fight, err := battle.New(tauntBooks(t), 6, []battle.Roster{
		// Nearly as fast as the tank, so it acts before the taunt has ticked
		// twice. A status is timed in its *holder's* turns, so a taunter much
		// faster than its victim spends the taunt on its own turns and the victim
		// never sees it -- which is why the slowest unit in a squad makes the best
		// taunter, and is a fact about the mechanic rather than an awkwardness of
		// this fixture.
		{ID: "attacker", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 190),
			Skills: []string{"poke", "snipe", "guard"}},
		{ID: "helper", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 100, 400, 1),
			Skills: []string{"snipe"}},
		{ID: "tank", Side: hex.SideEnemy, Slot: tankSlot,
			Affinity: single("neutral"), Stats: stats(30, 100, 400, 200),
			Skills: []string{"provoke", "poke"}},
		{ID: "squishy", Side: hex.SideEnemy, Slot: squishySlot,
			Affinity: single("neutral"), Stats: stats(3000, 100, 400, 1),
			Skills: []string{"snipe"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	// The tank is the fastest, so it provokes before anybody swings.
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "tank" {
		t.Fatalf("the first turn went to %q, so nothing has taunted yet", prompt.Unit)
	}
	if err := fight.Act("provoke", hex.Place(hex.SideEnemy, tankSlot)); err != nil {
		t.Fatalf("provoke: %v", err)
	}
	fight.Drain()
	return fight
}

// turnOf advances until the named unit is the one being asked, passing for
// everybody else, and hands back its prompt.
//
// The prompt rather than one skill's aims, because a turn is one thing: a test
// that read two skills by calling this twice would be asking the queue to walk
// past a unit that has not acted.
func turnOf(t *testing.T, fight *battle.Battle, unit string) *battle.Prompt {
	t.Helper()
	for turn := 0; turn < 40; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if prompt.Skipped {
			continue
		}
		if prompt.Unit == unit {
			return prompt
		}
		if err := fight.Pass("waiting"); err != nil {
			t.Fatalf("pass: %v", err)
		}
		fight.Drain()
	}
	t.Fatalf("%s never got a turn", unit)
	return nil
}

// offered is one skill's aims on the named unit's turn.
func offered(t *testing.T, fight *battle.Battle, unit, id string) []hex.Offset {
	t.Helper()
	for _, option := range turnOf(t, fight, unit).Options {
		if option.Skill == id {
			return option.Aims
		}
	}
	t.Fatalf("%s was not offered %s", unit, id)
	return nil
}

// TestATauntedUnitMayAimAtNobodyElse is the whole of the mechanic.
//
// The attacker can reach both enemies without the taunt and would rather hit the
// one it can kill. With the taunt up, the taunter is the only cell it is offered.
func TestATauntedUnitMayAimAtNobodyElse(t *testing.T) {
	fight := aTauntedField(t)
	onto := hex.Place(hex.SideEnemy, tankSlot)
	prompt := turnOf(t, fight, "attacker")
	for _, option := range prompt.Options {
		if option.Skill == "guard" {
			continue
		}
		if len(option.Aims) != 1 || option.Aims[0] != onto {
			t.Errorf("%s was offered %v while the tank at %s was taunting",
				option.Skill, option.Aims, onto)
		}
	}
}

// TestATauntIgnoresRange is the rule that makes a taunt worth anything on a board
// where nothing moves.
//
// The taunter stands where a range-one skill cannot reach it. A taunt that
// respected range would be answered by standing far enough away, which is
// exactly what the long-ranged attackers a tank most needs to pull would do.
func TestATauntIgnoresRange(t *testing.T) {
	fight, err := battle.New(tauntBooks(t), 6, []battle.Roster{
		{ID: "attacker", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"poke", "snipe"}},
		// Slower than the unit it taunts, deliberately. A status is timed in its
		// holder's turns, so a taunter faster than its victim spends the taunt on
		// its own turns before the victim ever acts -- which is a real thing an
		// author has to price and not something to arrange around here.
		{ID: "far", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 100, 400, 200),
			Skills: []string{"provoke", "snipe"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("provoke", hex.Place(hex.SideEnemy, hex.Offset{Col: 0, Row: 1})); err != nil {
		t.Fatalf("provoke: %v", err)
	}
	fight.Drain()

	onto := hex.Place(hex.SideEnemy, hex.Offset{Col: 0, Row: 1})
	attacker, _ := fight.Unit("attacker")
	if reach := attacker.Cell.DistanceTo(onto); reach <= 1 {
		t.Fatalf("the taunter is %d cells away, which a range-one skill reaches anyway", reach)
	}
	aims := offered(t, fight, "attacker", "poke")
	if len(aims) != 1 || aims[0] != onto {
		t.Errorf("a range-one skill was offered %v against a taunter %d cells away",
			aims, attacker.Cell.DistanceTo(onto))
	}
	// And the hit lands, because nothing past the legality of the aim has ever
	// read distance.
	if err := fight.Act("poke", onto); err != nil {
		t.Fatalf("poke: %v", err)
	}
	hit := false
	for _, event := range fight.Drain() {
		if event.Kind == battle.Damaged && event.Target == "far" {
			hit = true
		}
	}
	if !hit {
		t.Error("the forced aim was offered and then dealt nothing")
	}
}

// TestATauntTakesTheChoiceOfEnemyAndNotTheTurn, which is what makes it a
// different category from a stun.
//
// A taunted unit may still help its own side. Taking that away too would make a
// taunt a stun that also happens to point somewhere.
func TestATauntTakesTheChoiceOfEnemyAndNotTheTurn(t *testing.T) {
	fight := aTauntedField(t)
	aims := offered(t, fight, "attacker", "guard")
	if len(aims) == 0 {
		t.Fatal("a taunted unit was not allowed to help its own side")
	}
	onto := hex.Place(hex.SideEnemy, tankSlot)
	for _, aim := range aims {
		if aim == onto {
			t.Errorf("an ally-aimed skill was offered the taunter's cell %s", onto)
		}
	}
}

// TestTheOpponentObeysATauntWithoutBeingToldTo.
//
// Suggest reads the aims it is offered and nothing else, so a taunt that narrows
// the list narrows the opponent's choice with it. This asserts that rather than
// assuming it: an AI that built its own list would walk straight past the whole
// mechanic.
func TestTheOpponentObeysATauntWithoutBeingToldTo(t *testing.T) {
	fight := aTauntedField(t)
	onto := hex.Place(hex.SideEnemy, tankSlot)
	for turn := 0; turn < 40; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil || prompt.Skipped {
			continue
		}
		if prompt.Unit != "attacker" {
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
			fight.Drain()
			continue
		}
		choice, ok := fight.Suggest(prompt)
		if !ok {
			t.Fatal("the opponent had nothing to suggest")
		}
		if choice.Aim != onto {
			t.Errorf("the opponent aimed at %s while the tank at %s was taunting",
				choice.Aim, onto)
		}
		return
	}
	t.Fatal("the attacker never got a turn")
}

// TestATauntDiesWithItsTaunter, which is the reason the status sits on the unit
// doing the taunting.
//
// Nothing has to be cleaned up: the answer to "who must I attack" is read off the
// board, and a corpse is not on it.
func TestATauntDiesWithItsTaunter(t *testing.T) {
	fight := aTauntedField(t)
	onto := hex.Place(hex.SideEnemy, tankSlot)

	// Forced onto the taunter, and it is frail enough that the forced blow is
	// the last one it takes.
	turnOf(t, fight, "attacker")
	if err := fight.Act("poke", onto); err != nil {
		t.Fatalf("poke: %v", err)
	}
	died := false
	for _, event := range fight.Drain() {
		if event.Kind == battle.Died && event.Actor == "tank" {
			died = true
		}
	}
	if !died {
		t.Fatal("the taunter survived the forced blow, so nothing was taken off the board")
	}

	prompt := turnOf(t, fight, "attacker")
	for _, option := range prompt.Options {
		if option.Skill != "snipe" {
			continue
		}
		if len(option.Aims) == 0 {
			t.Fatal("with the taunter gone the attacker was offered nothing at all")
		}
		for _, aim := range option.Aims {
			if aim == onto {
				t.Errorf("the attacker was still pointed at the dead taunter's cell %s", onto)
			}
		}
		return
	}
	t.Fatal("the attacker was not offered snipe")
}
