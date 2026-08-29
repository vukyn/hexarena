package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// restoringBooks is the shared books with two skills that restore the same
// amount off the same stat, differing in who they are aimed at.
//
// That pair is the whole point. A restore used to live inside resolveAgainst,
// which Act returns before for a Target: Self skill, so the self-aimed half paid
// nothing while the ally-aimed half paid in full — and no shipped data could show
// it, because both skills that declare `restores` are self-aimed.
//
// ⚠️ They differ in `range` as well, and that is the parser rather than the
// fixture: a self-aimed skill takes range nought and every other aim is refused
// below one. Range cannot reach the payout — an ally-aimed skill ignores reach
// entirely — so the pair still isolates the aim, but it is not literally a
// one-field difference and saying so would have been a claim nobody checked.
func restoringBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	restoring, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"mend_self","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "restores":500},
	  {"id":"mend_ally","element":"neutral","range":1,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"ally",
	   "restores":500},
	  {"id":"poke","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = restoring
	return shared
}

// aWoundedHealer is one hurt caster carrying both restores and an ally standing
// beside it, hurt by exactly the same amount so the two payouts are comparable.
func aWoundedHealer(t *testing.T) (*battle.Battle, *battle.Unit, *battle.Unit) {
	t.Helper()
	fight, err := battle.New(restoringBooks(t), 5, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"mend_self", "mend_ally"}},
		{ID: "friend", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 1),
			Skills: []string{"poke"}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 100, 300, 1),
			Skills: []string{"poke"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	healer, ok := fight.Unit("healer")
	if !ok {
		t.Fatal("no healer in the battle")
	}
	friend, ok := fight.Unit("friend")
	if !ok {
		t.Fatal("no friend in the battle")
	}
	healer.HP = 900
	friend.HP = 900
	return fight, healer, friend
}

// castOnce takes the healer's turn with the named skill and returns what it
// healed, off the log rather than off the health, so a test that asserts nothing
// happened cannot pass on a unit that was hurt back in the same turn.
func castOnce(t *testing.T, fight *battle.Battle, id string, aim hex.Offset) (int64, int) {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "healer" {
		t.Fatalf("the turn did not go to the healer: %+v", prompt)
	}
	if err := fight.Act(id, aim); err != nil {
		t.Fatalf("act %s: %v", id, err)
	}
	var healed int64
	events := 0
	for _, event := range fight.Drain() {
		if event.Kind == battle.Healed {
			healed += event.Amount
			events++
		}
	}
	return healed, events
}

// TestASelfAimedRestoreHeals is the defect. Act returns for a Target: Self skill
// before the walk that used to hold the payout, so a skill whose whole body is a
// restore on itself did nothing at all.
func TestASelfAimedRestoreHeals(t *testing.T) {
	fight, healer, _ := aWoundedHealer(t)
	before := healer.HP
	healed, events := castOnce(t, fight, "mend_self", healer.Cell)
	if events != 1 {
		t.Fatalf("a self-aimed restore emitted %d healed events, want one", events)
	}
	if healed <= 0 {
		t.Fatalf("a self-aimed restore paid out %d", healed)
	}
	if healer.HP != before+healed {
		t.Errorf("the healer went %d -> %d on a payout of %d",
			before, healer.HP, healed)
	}
}

// TestBothAimsPayTheSameRestore is the claim the fix is actually about: one
// declaration, one payout, whoever it is aimed at.
//
// Asserted as an equality between the two halves rather than against a figure,
// because the figure is combat.Rules.Restore's and a test that spelled it would
// be a second copy of the arithmetic — the mistake the file it lives beside keeps
// a list of. The two skills differ in `target` and in nothing else.
func TestBothAimsPayTheSameRestore(t *testing.T) {
	onSelf, healer, _ := aWoundedHealer(t)
	self, _ := castOnce(t, onSelf, "mend_self", healer.Cell)

	onAlly, _, friend := aWoundedHealer(t)
	ally, _ := castOnce(t, onAlly, "mend_ally", friend.Cell)

	if self != ally {
		t.Errorf("the same restore paid %d on itself and %d on an ally", self, ally)
	}
}
