package screen

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheBuilderStopsAtTheSquadCapAndNotAtTheBoard is the squad builder's share
// of the ENG-013 boundary.
//
// internal/core/battle bounds a side by hex.BoardSlots and will admit nine — it
// has to, because a summon reaches the board through the same enlist a roster
// does. What keeps a nine-unit side out of the game is that every path which
// AUTHORS a squad counts to hex.MaxSquadSize instead, and this is one of the
// four. The others are held where they live: placement.Squad.Validate and the
// engine half in internal/core/battle
// (TestANineUnitRosterIsLegalHereAndRefusedByEveryAuthoringPath), the room gate
// and the draft in internal/room (TestANineUnitSquadReachesNoRoomAndNoDraft).
//
// ⚠️ **The point of the test is the number it stops at, not that it stops.** Both
// caps are ints and reading the wrong one compiles: a builder on hex.BoardSlots
// would go on refusing a tenth member and look entirely correct while offering a
// squad no room would seat and no format could field.
func TestTheBuilderStopsAtTheSquadCapAndNotAtTheBoard(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	s := aNewSquad(t, c, "cap")
	for range hex.BoardSlots + 1 {
		s = s.addUnit()
	}
	if len(s.Editing.Units) != hex.MaxSquadSize {
		t.Errorf("the builder saved %d members: a side is fielded with %d, and the board's "+
			"%d is what the ENGINE admits, not what a squad may be built to",
			len(s.Editing.Units), hex.MaxSquadSize, hex.BoardSlots)
	}
}
