package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestAFullSquadStillHasRoomForASummon is what ENG-013 bought, and it is the
// exact inverse of TestASideAtTheTeamCapSummonsNobody.
//
// A side fielded with hex.MaxSquadSize units stands on a board of
// hex.BoardSlots, so there are hex.BoardSlots-hex.MaxSquadSize cells left and a
// summon has somewhere to go. Before the split those two numbers were one
// constant: a full side of the largest format computed room = 0,
// summonWorth priced every summoning skill at nought and the rating never cast
// one, so `split`, `shadow_clone`, `summon_toad` and the diglett.three build
// were dead slots at exactly the board they were most wanted on.
//
// ⚠️ **The cast is forced rather than suggested**, for the same reason its
// inverse forces one: whether the rating would choose it is a different claim,
// and a test that drove Suggest could not tell "the rating declined" from "the
// board had no room". This one asks the board.
//
// ⚠️ **Written against the constants, and the arithmetic is asserted rather than
// a count typed in.** A figure of three would go on passing on the day a
// formation gained a column, while saying something false about why.
func TestAFullSquadStillHasRoomForASummon(t *testing.T) {
	gap := hex.BoardSlots - hex.MaxSquadSize
	if gap < 1 {
		t.Fatalf("a side fields %d of the board's %d slots, so there is no gap to measure",
			hex.MaxSquadSize, hex.BoardSlots)
	}
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"swarm", "jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 5),
			Skills: []string{"jab"}},
	}
	// The caster plus enough bodies to bring the side to exactly what a squad
	// may field — a full 5v5 side, which is the board ENG-013 was raised on.
	for col := hex.FormationCols - 1; col >= 0 && len(roster) < hex.MaxSquadSize+1; col-- {
		for row := range hex.Rows {
			slot := hex.Offset{Col: col, Row: row}
			if slot == (hex.Offset{Col: 2, Row: 1}) || len(roster) >= hex.MaxSquadSize+1 {
				continue
			}
			roster = append(roster, battle.Roster{
				ID: "squad" + slot.String(), Side: hex.SideAlly, Slot: slot,
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 1),
				// lob rather than jab: the back rows are out of a jab's reach and
				// these bodies are here to fill the squad, not to fight.
				Skills: []string{"lob"},
			})
		}
	}
	fight := mustBattle(t, books(t), 5, roster)
	if standing := livingOn(fight, hex.SideAlly); standing != hex.MaxSquadSize {
		t.Fatalf("the ally side holds %d units, and this measures a side fielded with the "+
			"squad cap of %d", standing, hex.MaxSquadSize)
	}
	came := arrivals(casts(t, fight, "swarm"))
	if len(came) == 0 {
		t.Fatalf("a swarm cast by a side of %d put down nobody on a board of %d: the gap "+
			"between the two is the room a summon stands in, and it is %d cells wide",
			hex.MaxSquadSize, hex.BoardSlots, gap)
	}
	// A count is a request rather than a promise: the skill asks for three and
	// the gap allows four, so all three arrive.
	if len(came) > gap {
		t.Errorf("a swarm put down %d copies into a gap of %d", len(came), gap)
	}
	if standing := livingOn(fight, hex.SideAlly); standing != hex.MaxSquadSize+len(came) {
		t.Errorf("the ally side holds %d units after %d copies arrived on a side of %d",
			standing, len(came), hex.MaxSquadSize)
	}
	if standing := livingOn(fight, hex.SideAlly); standing > hex.BoardSlots {
		t.Errorf("the ally side holds %d units and the board admits %d", standing, hex.BoardSlots)
	}
}
