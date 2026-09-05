package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// theBonusBook is one sharers-only bonus over the element axis, with a rung at
// two and a heavier one at three, granting a permanent attack buff.
func theBonusBook(t *testing.T) *composition.Book {
	t.Helper()
	loaded := books(t)
	book, err := composition.ParseBook([]byte(`{"bonuses": [
	  {"id": "same_element", "name": "đồng hệ", "axis": "element", "scope": "sharers", "rungs": [
	    {"at": 2, "grants": [{"status": "toughened", "stacks": 1}]},
	    {"at": 3, "grants": [{"status": "toughened", "stacks": 2}]}
	  ]}
	]}`), composition.Deps{Statuses: loaded.Statuses, Chart: loaded.Chart})
	if err != nil {
		t.Fatalf("parse the bonus fixture: %v", err)
	}
	return book
}

// bonusRoster is two units a side, and which element each carries is the whole
// experiment: the ally pair shares one and the enemy pair does not.
func bonusRoster(t *testing.T, allyShares bool) []battle.Roster {
	t.Helper()
	single := func(name string) element.Affinity {
		one, err := element.Parse(name)
		if err != nil {
			t.Fatalf("parse %q: %v", name, err)
		}
		affinity, err := element.Single(one)
		if err != nil {
			t.Fatalf("build the affinity for %q: %v", name, err)
		}
		return affinity
	}
	stats := progression.Values{
		progression.HP: 900, progression.Attack: 200, progression.Defense: 120,
		progression.Speed: 50, progression.Accuracy: 90, progression.Dodge: 10,
	}
	second := "fire"
	if allyShares {
		second = "water"
	}
	return []battle.Roster{
		{ID: "A1", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("water"), Stats: stats, Skills: []string{"strike"}},
		{ID: "A2", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 1},
			Affinity: single(second), Stats: stats, Skills: []string{"strike"}},
		{ID: "E1", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats, Skills: []string{"strike"}},
		{ID: "E2", Side: hex.SideEnemy, Slot: hex.Offset{Col: 1, Row: 1},
			Affinity: single("ground"), Stats: stats, Skills: []string{"strike"}},
	}
}

func defenceOf(t *testing.T, fight *battle.Battle, id string) int64 {
	t.Helper()
	unit, ok := fight.Unit(id)
	if !ok {
		t.Fatalf("no unit is called %q", id)
	}
	return fight.Stats(unit)[progression.Defense]
}

// TestASharedElementIsOnTheUnitBeforeTheFirstTurn is the whole mechanism in one
// reading: two units of one element walk in carrying the grant, and the two that
// share nothing walk in without it.
func TestASharedElementIsOnTheUnitBeforeTheFirstTurn(t *testing.T) {
	loaded := books(t)
	loaded.Bonuses = theBonusBook(t)
	fight, err := battle.New(loaded, 1, bonusRoster(t, true))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := battle.New(books(t), 1, bonusRoster(t, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"A1", "A2"} {
		with, without := defenceOf(t, fight, id), defenceOf(t, plain, id)
		if with <= without {
			t.Errorf("%s shares an element and its defence reads %d against %d with no bonus book", id, with, without)
		}
	}
	for _, id := range []string{"E1", "E2"} {
		with, without := defenceOf(t, fight, id), defenceOf(t, plain, id)
		if with != without {
			t.Errorf("%s shares nothing and its defence moved %d -> %d", id, without, with)
		}
	}
}

// TestABonusIsCountedPerSide is the half a shared count would get wrong: the
// enemy's water unit is not the ally's kin, so a side is counted on its own.
func TestABonusIsCountedPerSide(t *testing.T) {
	loaded := books(t)
	loaded.Bonuses = theBonusBook(t)
	roster := bonusRoster(t, false)
	// One water on each side, and nobody shares with anybody.
	roster[2].Affinity = roster[0].Affinity
	fight, err := battle.New(loaded, 1, roster)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := battle.New(books(t), 1, roster)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"A1", "A2", "E1", "E2"} {
		if with, without := defenceOf(t, fight, id), defenceOf(t, plain, id); with != without {
			t.Errorf("%s was awarded across the board: defence %d -> %d", id, without, with)
		}
	}
}

// TestTheOpeningBoardSaysWhichThresholdPaid is the log half. A permanent buff
// the log does not account for is the trap PassiveHeld was added to close, and a
// bonus is the same shape one layer out.
func TestTheOpeningBoardSaysWhichThresholdPaid(t *testing.T) {
	loaded := books(t)
	loaded.Bonuses = theBonusBook(t)
	fight, err := battle.New(loaded, 1, bonusRoster(t, true))
	if err != nil {
		t.Fatal(err)
	}
	fight.Begin()
	var held []battle.Event
	for _, event := range fight.Drain() {
		if event.Kind == battle.BonusHeld {
			held = append(held, event)
		}
	}
	if len(held) != 2 {
		t.Fatalf("two units shared an element and the log opened with %d bonus lines", len(held))
	}
	for _, event := range held {
		if event.Bonus != "same_element" || event.Shared != "water" || event.Count != 2 {
			t.Errorf("the line does not say what fired: %+v", event)
		}
		if event.Status != "toughened" || event.Stacks != 1 {
			t.Errorf("the line does not say what was granted: %+v", event)
		}
		if event.Actor != "A1" && event.Actor != "A2" {
			t.Errorf("a bonus line names %q, which shares nothing", event.Actor)
		}
	}
}

// TestABonusIsOnBeforeTheQueueReadsTheSpeed is the ordering rule, and it is the
// one a correction cannot fix: a wait is 1_000_000/speed and the first one has
// been served by the time anything could retune it.
//
// ⚠️ The fixture is built so the two answers differ. The sharing pair is
// **slower** at its base line and faster only with the grant counted, so a queue
// built before the award would put the enemy first and one built after puts the
// ally first. Equal speeds would pass on the tie-break either way, which is how
// this class of test passes without measuring anything.
func TestABonusIsOnBeforeTheQueueReadsTheSpeed(t *testing.T) {
	loaded := books(t)
	quickening, err := composition.ParseBook([]byte(`{"bonuses": [
	  {"id": "same_element", "axis": "element", "scope": "sharers", "rungs": [
	    {"at": 2, "grants": [{"status": "fleet", "stacks": 1}]}
	  ]}
	]}`), composition.Deps{Statuses: loaded.Statuses, Chart: loaded.Chart})
	if err != nil {
		t.Fatalf("parse the quickening fixture: %v", err)
	}
	loaded.Bonuses = quickening
	roster := bonusRoster(t, true)
	for i := range roster {
		stats := roster[i].Stats
		if roster[i].Side == hex.SideAlly {
			stats[progression.Speed] = 100
		} else {
			stats[progression.Speed] = 103
		}
		roster[i].Stats = stats
	}
	fight, err := battle.New(loaded, 1, roster)
	if err != nil {
		t.Fatal(err)
	}
	fight.Begin()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatal(err)
	}
	if prompt == nil {
		t.Fatal("the battle opened with nobody to act")
	}
	if prompt.Unit != "A1" && prompt.Unit != "A2" {
		t.Errorf("the first turn went to %q: the shared-element pair is slower at its base line and "+
			"faster with the grant on, so the queue was built before the award", prompt.Unit)
	}
	// And with the bonus switched off the same roster goes the other way, which
	// is what says the fixture measures the award rather than the tie-break.
	loaded.Bonuses = quickening.Without("same_element")
	off, err := battle.New(loaded, 1, roster)
	if err != nil {
		t.Fatal(err)
	}
	off.Begin()
	first, err := off.Advance()
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || (first.Unit != "E1" && first.Unit != "E2") {
		t.Fatalf("with the bonus off the first turn went to %v, wanted the faster enemy pair", first)
	}
}

// TestABattleRunsWithNoBonusBook is what keeps every caller that predates the
// file working, the same promise the trait book carries.
func TestABattleRunsWithNoBonusBook(t *testing.T) {
	loaded := books(t)
	loaded.Bonuses = nil
	fight, err := battle.New(loaded, 1, bonusRoster(t, true))
	if err != nil {
		t.Fatal(err)
	}
	fight.Begin()
	for _, event := range fight.Drain() {
		if event.Kind == battle.BonusHeld {
			t.Fatalf("a battle with no bonus book emitted %+v", event)
		}
	}
}
