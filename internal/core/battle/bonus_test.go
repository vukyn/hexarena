package battle_test

import (
	"strconv"
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

// The second axis. `same_element` counts what a squad IS; this counts where it
// STANDS, and the two are different enough that the counting had to be split
// rather than parameterised — an element bonus skips the inert element, and there
// is no inert column.
//
// ⚠️ **The roadmap named the archetype's column and the engine cannot see it.**
// Roster carries no archetype on purpose — settled before a battle, leaves nothing
// behind but numbers — so counting the preset would have meant widening the
// roster, the wire and every log to say something weaker than the slot already
// says. An archetype's column is where a character *wants* to stand; the slot is
// where the player put it, which is the decision a composition bonus rewards.
// aColumnBattle is one side standing in the columns given, against an enemy pair
// that always shares column 2 — so a count that crossed the board would show.
func aColumnBattle(t *testing.T, cols []int) *battle.Battle {
	t.Helper()
	loaded := books(t)
	book, err := composition.ParseBook([]byte(`{"bonuses": [
	  {"id": "same_column", "name": "cùng tuyến", "axis": "column", "scope": "sharers", "rungs": [
	    {"at": 3, "grants": [{"status": "toughened", "stacks": 2}]}
	  ]}
	]}`), composition.Deps{Statuses: loaded.Statuses, Chart: loaded.Chart})
	if err != nil {
		t.Fatalf("parse the column bonus: %v", err)
	}
	loaded.Bonuses = book
	one, err := element.Parse("water")
	if err != nil {
		t.Fatalf("parse water: %v", err)
	}
	affinity, err := element.Single(one)
	if err != nil {
		t.Fatalf("affinity: %v", err)
	}
	stats := progression.Values{
		progression.HP: 900, progression.Attack: 200, progression.Defense: 120,
		progression.Speed: 50, progression.Accuracy: 90, progression.Dodge: 10,
	}
	roster := make([]battle.Roster, 0, len(cols)+2)
	for i, col := range cols {
		roster = append(roster, battle.Roster{
			ID: "a" + strconv.Itoa(i+1), Side: hex.SideAlly,
			Slot: hex.Offset{Col: col, Row: i}, Affinity: affinity, Stats: stats,
			Skills: []string{"strike"}})
	}
	for i := range 2 {
		roster = append(roster, battle.Roster{
			ID: "e" + strconv.Itoa(i+1), Side: hex.SideEnemy,
			Slot: hex.Offset{Col: 2, Row: i}, Affinity: affinity, Stats: stats,
			Skills: []string{"strike"}})
	}
	fight, err := battle.New(loaded, 1, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	return fight
}

// TestAColumnBonusCountsWhereUnitsStand holds both rungs and the miss.
//
// ⚠️ **Rung three is here because no shipped squad reaches it.** s01 stands two
// in one column and every other shipped squad stands its three in three, so the
// top rung of this bonus fires nowhere in the data — which is the "fixture hides a
// branch" shape this repository has paid for five times. It is reachable (a column
// holds hex.FormationRows units, and that is three), and this is where that is
// said out loud.
func TestAColumnBonusCountsWhereUnitsStand(t *testing.T) {
	for _, one := range []struct {
		name   string
		cols   []int
		stacks int
	}{
		{"three in one column", []int{2, 2, 2}, 2},
		// ⚠️ **Two in a column is deliberately NOT a rung**, and that is the whole
		// reason this bonus has one. A rung at two fires on any board that puts a
		// pair in a column, which is most constructed fixtures in this repository —
		// they choose skills and traits deliberately and slots carelessly — so it
		// was measured, found to move two unrelated balance tests, and dropped. A
		// side has to stack ALL of itself to be paid for the shape.
		{"two in one column", []int{2, 2, 0}, 0},
		{"three in three columns", []int{0, 1, 2}, 0},
	} {
		t.Run(one.name, func(t *testing.T) {
			fight := aColumnBattle(t, one.cols)
			held, known := fight.Unit("a1")
			if !known {
				t.Fatal("no first unit on the board")
			}
			if got := held.Statuses.Stacks("toughened"); got != one.stacks {
				t.Errorf("a squad standing in columns %v gave its first unit %d stacks, want %d",
					one.cols, got, one.stacks)
			}
		})
	}
}

// TestAColumnBonusIsCountedPerSide is the rule every bonus obeys and the one a
// second axis is most likely to break: a side is counted on its own, so an enemy
// standing in the same column cannot push this side over a threshold.
func TestAColumnBonusIsCountedPerSide(t *testing.T) {
	// Two allies in column 2, and the two enemies stand there too. Across the
	// board that is four, which would reach the rung; on either side alone it is
	// two, which does not.
	fight := aColumnBattle(t, []int{2, 2, 0})
	held, _ := fight.Unit("a1")
	if got := held.Statuses.Stacks("toughened"); got != 0 {
		t.Errorf("two allies in a column with two enemies in the same one gave %d stacks, "+
			"want none: the count is crossing the board", got)
	}
}
