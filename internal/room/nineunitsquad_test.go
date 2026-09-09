package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestANineUnitSquadReachesNoRoomAndNoDraft is the half of the ENG-013 boundary
// that lives here.
//
// internal/core/battle bounds a side by hex.BoardSlots and will admit a
// nine-unit roster — deliberately, because a summon reaches the board through
// the same enlist a roster does, and a fielding bound there would refuse every
// copy a full side calls up. What keeps such a side out of a game is that every
// path which ACCEPTS a squad counts to hex.MaxSquadSize instead, and two of the
// four are in this package's reach.
//
// → internal/core/battle,
// TestANineUnitRosterIsLegalHereAndRefusedByEveryAuthoringPath for the engine
// half and for placement.Squad.Validate.
//
// ⚠️ **The gate's rule is an equality, not a ceiling**, and that is what makes it
// bite in both directions: a 3v3 room takes squads of three, so nine is refused
// there for being too many and five would be refused for the same reason.
func TestANineUnitSquadReachesNoRoomAndNoDraft(t *testing.T) {
	dependencies := deps(t)
	squad := placement.Squad{ID: "nine.squad", Name: "nine"}
	for col := range hex.FormationCols {
		for row := range hex.Rows {
			slot := hex.Offset{Col: col, Row: row}
			squad.Units = append(squad.Units,
				twinUnit(t, dependencies.Characters, "pokemon.machop", slot.String(), slot))
		}
	}
	if len(squad.Units) != hex.BoardSlots {
		t.Fatalf("the fixture brings %d units where the board admits %d",
			len(squad.Units), hex.BoardSlots)
	}
	// Every unit is legal on its own — level, form and loadout — so what the gate
	// refuses below is the size and nothing else.
	for _, unit := range squad.Units {
		if unit.Level != progression.LevelCap {
			t.Fatalf("unit %q is at level %d, and the gate refuses anything under the cap",
				unit.ID, unit.Level)
		}
	}

	opened := newRoom(t, config(7, 1))
	admitted, out, err := opened.Join(hello(t, squad, "Nine"))
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if admitted.Seat.Valid() {
		t.Errorf("a squad of %d was seated in a %s room: the board admits that many and no "+
			"side is ever FIELDED with more than %d",
			len(squad.Units), config(7, 1).Format, hex.MaxSquadSize)
	} else if code := onlyCode(t, out); code != wire.CodeSquadRefused {
		t.Errorf("the gate refused the squad with %q, want the squad refusal", code)
	}

	// And the draft cannot produce one either, at any format the protocol
	// offers: a drafted squad is what a side picks, and a side picks the
	// format's own unit count.
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		if picks := draft.PicksPerSide(format); picks > hex.MaxSquadSize {
			t.Errorf("a %s draft picks %d a side against a squad cap of %d",
				format, picks, hex.MaxSquadSize)
		}
	}
	if hex.MaxSquadSize >= hex.BoardSlots {
		t.Errorf("a side is fielded with %d of the board's %d slots, so nothing above is "+
			"holding a boundary — there is no gap left for a summon either",
			hex.MaxSquadSize, hex.BoardSlots)
	}
}
