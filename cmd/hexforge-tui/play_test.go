package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
)

// atTheBattle is a model with a squad saved, the fight raised over it and the
// battle opened: the state every test below starts from.
func atTheBattle(t *testing.T, m model) model {
	t.Helper()
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	if m.screen != screenPlay {
		t.Fatalf("p opened %v rather than the battle", m.screen)
	}
	if m.play.err != nil {
		t.Fatalf("the battle would not start: %v", m.play.err)
	}
	return m
}

// TestTheFightRaisesABattleYouPlay is the wiring, and that the battle arrives
// already waiting on the player rather than on a key nobody knows to press.
func TestTheFightRaisesABattleYouPlay(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	if m.play.fight == nil {
		t.Fatal("no battle was built")
	}
	if m.play.pending == nil {
		t.Fatal("the battle opened without a turn for the player")
	}
	// The turn waiting is the player's own, which is the whole claim: every
	// other side's turn is taken on the way here.
	unit, known := m.play.fight.Unit(m.play.pending.Unit)
	if !known || unit.Side != m.play.side {
		t.Fatalf("the battle is waiting on %q", m.play.pending.Unit)
	}
	// And the cursor is on something that can actually be taken.
	if option := m.play.pending.Options[m.play.option]; !option.Available() {
		t.Errorf("the cursor opened on %q, which cannot be used: %s", option.Skill, option.Reason)
	}
	if back := key(t, m, "esc"); back.screen != screenFight {
		t.Errorf("esc left the battle for %v", back.screen)
	}
}

// TestATurnTakenMovesTheBattleOn is the loop: the player acts, the engine
// answers, and the next thing waiting is the player again.
func TestATurnTakenMovesTheBattleOn(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	before := len(m.play.script)
	events := len(m.play.events)
	m = key(t, m, "enter")
	if len(m.play.script) <= before {
		t.Fatal("nothing was written down")
	}
	if len(m.play.events) <= events {
		t.Error("a turn was taken and the battle recorded nothing")
	}
	// The opponent answered on the way back, so the script grew by more than the
	// one decision the player made.
	if len(m.play.script) < before+2 && !m.play.fight.Finished() {
		t.Errorf("the script holds %d decisions, want the player's and the engine's",
			len(m.play.script))
	}
	if m.play.pending == nil && !m.play.fight.Finished() {
		t.Error("the battle stopped without a turn for the player and without ending")
	}
}

// TestUndoTakesBackYourOwnTurnAndNotTheEngines is the one thing a hand-played
// battle needs that a simulation does not.
//
// It works because a battle is a pure function of its seed and the decisions
// taken: undo is not an unwinding, it is a shorter list replayed. That is the
// same property --verify rests on, which is why this is worth asserting here
// rather than trusting.
func TestUndoTakesBackYourOwnTurnAndNotTheEngines(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	opening := m.play.script
	m = key(t, m, "enter")
	if len(m.play.script) <= len(opening) {
		t.Fatal("nothing to take back")
	}
	fought := len(m.play.script)
	m = typeText(t, m, "u")
	if len(m.play.script) >= fought {
		t.Fatalf("undo left %d decisions of %d", len(m.play.script), fought)
	}
	// What is left ends with somebody else's turn: undo cuts at the player's
	// last decision, so nothing of theirs survives it.
	for _, decision := range m.play.script {
		unit, known := m.play.fight.Unit(decision.Unit)
		if known && unit.Side == m.play.side {
			t.Errorf("a turn of the player's own survived the undo: %+v", decision)
		}
	}
	// And the battle is waiting on the player again rather than on nobody.
	if m.play.pending == nil && !m.play.fight.Finished() {
		t.Error("after an undo the battle waits on nothing")
	}
	// Nothing to take back is not an error, it is a key that does nothing.
	fresh, _, _ := start(t, i18n.En)
	fresh = atTheBattle(t, fresh)
	fresh = typeText(t, fresh, "u")
	if fresh.play.err != nil {
		t.Errorf("undo with nothing of the player's in the script reported %v", fresh.play.err)
	}
}

// TestAnotherSeedIsAnotherBattle is what a player asks for when a pairing has
// been played once.
func TestAnotherSeedIsAnotherBattle(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	m = key(t, m, "enter")
	seed := m.play.seed
	fought := len(m.play.script)
	m = typeText(t, m, "n")
	if m.play.seed != seed+1 {
		t.Errorf("n moved the seed to %d, want %d", m.play.seed, seed+1)
	}
	if len(m.play.script) >= fought {
		t.Errorf("the new battle kept %d decisions from the old one", len(m.play.script))
	}
	if m.play.err != nil {
		t.Errorf("the new battle would not start: %v", m.play.err)
	}
}

// TestAimingIsAskedOnlyWhenItIsADecision is the second question: a skill with
// one legal cell does not ask it, because a question with one answer is not a
// decision.
func TestAimingIsAskedOnlyWhenItIsADecision(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	// A second member, so an enemy-aimed skill has two cells to choose between.
	second := m.squad.editing.Units[0].Clone()
	second.ID = "hai"
	second.Slot = hex.Offset{Col: hex.FormationCols - 1, Row: 0}
	m.squad.editing.Units = append(m.squad.editing.Units, second)
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	if m.play.pending == nil {
		t.Fatal("the battle opened without a turn for the player")
	}
	// Walk to a skill with more than one cell, if the opening turn has one.
	found := false
	for index, option := range m.play.pending.Options {
		if option.Available() && len(option.Aims) > 1 {
			m.play.option = index
			found = true
			break
		}
	}
	if !found {
		t.Skip("no skill on the opening turn has two cells to choose between")
	}
	m = key(t, m, "enter")
	if !m.play.aiming {
		t.Fatal("a skill with two cells acted without asking where")
	}
	body := m.screenContent()
	if !strings.Contains(body, strings.Fields(m.text(i18n.PlayAimAt, "x"))[0]) {
		t.Errorf("the aim list is not drawn:\n%s", body)
	}
	// esc backs out of the second question without spending the turn.
	before := len(m.play.script)
	m = key(t, m, "esc")
	if m.play.aiming {
		t.Error("esc did not leave the aim list")
	}
	if m.screen != screenPlay {
		t.Errorf("esc left the battle for %v", m.screen)
	}
	if len(m.play.script) != before {
		t.Error("backing out of an aim spent the turn")
	}
}

// TestABattlePlayedOutEndsAndSaysHow is the far end of the screen, driven by
// the key that hands each turn to the engine.
func TestABattlePlayedOutEndsAndSaysHow(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	for range playTurnLimit {
		if m.play.fight.Finished() || m.play.err != nil {
			break
		}
		m = typeText(t, m, "a")
	}
	if m.play.err != nil {
		t.Fatalf("the battle broke: %v", m.play.err)
	}
	if !m.play.fight.Finished() {
		t.Fatal("the battle never ended")
	}
	// It says which of the four endings it was, in words rather than in a code.
	body := m.screenContent()
	said := false
	for _, ending := range []i18n.Key{i18n.PlayWon, i18n.PlayLost, i18n.PlayDrawn, i18n.PlayEmptied} {
		if strings.Contains(body, m.text(ending)) {
			said = true
		}
	}
	if !said {
		t.Errorf("the battle ended and the screen does not say how:\n%s", body)
	}
	// The script is the whole battle, both sides in it, so what was played can
	// be replayed.
	sides := map[hex.Side]int{}
	for _, decision := range m.play.script {
		if unit, known := m.play.fight.Unit(decision.Unit); known {
			sides[unit.Side]++
		}
	}
	if sides[hex.SideAlly] == 0 || sides[hex.SideEnemy] == 0 {
		t.Errorf("the script holds %v, want turns from both sides", sides)
	}
	var _ battle.Script = m.play.script
}
