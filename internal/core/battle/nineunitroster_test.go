package battle_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// TestANineUnitRosterIsLegalHereAndRefusedByEveryAuthoringPath is the price of
// ENG-013, written down as a test rather than as a comment somebody can read
// past.
//
// **Where the boundary is.** internal/core/battle has no notion of a format and
// must not gain one — a Roster is a list of facts, and giving enlist a "how many
// may a side field" parameter would carry an authoring rule into the replayable
// core. So this layer bounds a side by the BOARD, hex.BoardSlots, and a
// nine-unit roster handed straight to battle.New is legal. It has to be: a
// summon reaches the board through enlist, so a bound of hex.MaxSquadSize here
// would refuse every copy a full side calls up, which is ENG-013 rebuilt one
// layer down.
//
// **What stops a nine-unit side arriving in the game.** Every path that AUTHORS
// or ACCEPTS a squad counts to hex.MaxSquadSize instead, and there are four:
//
//   - placement.Squad.Validate — asserted below, because it is the one this
//     package can see and the one every other path is built on.
//   - the squad builder (internal/screen, SquadsScreen.addUnit) — stops offering
//     a member at hex.MaxSquadSize.
//   - the room gate (internal/room, squadIsFieldable) — demands exactly
//     Format.Units(), which internal/wire's TestTheLargestFormatIsTheSquadCap
//     holds at or under hex.MaxSquadSize.
//   - the draft (internal/draft, PicksPerSide) — is Format.Units() again.
//
// The last two are held in internal/room by
// TestANineUnitSquadReachesNoRoomAndNoDraft, because this package cannot see
// them and a rule asserted where it does not live is a rule that goes stale.
//
// ⚠️ **So do not "tighten" battle.enlist's bound to the squad cap.** It reads
// like an oversight and it is the mechanism.
func TestANineUnitRosterIsLegalHereAndRefusedByEveryAuthoringPath(t *testing.T) {
	roster := make([]battle.Roster, 0, hex.BoardSlots+1)
	for col := range hex.FormationCols {
		for row := range hex.Rows {
			roster = append(roster, battle.Roster{
				ID: "a" + (hex.Offset{Col: col, Row: row}).String(), Side: hex.SideAlly,
				Slot:     hex.Offset{Col: col, Row: row},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: []string{"lob"},
			})
		}
	}
	if len(roster) != hex.BoardSlots {
		t.Fatalf("the fixture stands %d units where the board admits %d", len(roster), hex.BoardSlots)
	}
	roster = append(roster, battle.Roster{
		ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
		Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
		Skills: []string{"lob"},
	})

	fight, err := battle.New(books(t), 5, roster)
	if err != nil {
		t.Fatalf("a roster of %d a side was refused by the engine, which bounds a side by "+
			"the board: %v", hex.BoardSlots, err)
	}
	if standing := livingOn(fight, hex.SideAlly); standing != hex.BoardSlots {
		t.Errorf("the ally side holds %d units, want the board's %d", standing, hex.BoardSlots)
	}

	// And the authoring half, on the same nine bodies. Validate is deliberately
	// lenient about a half-finished unit — a squad being built has to be savable
	// — so the size is one of the few things it does refuse.
	squad := placement.Squad{ID: "nine", Name: "nine"}
	for _, entry := range roster[:hex.BoardSlots] {
		squad.Units = append(squad.Units, placement.Placement{
			ID: entry.ID, Character: "pokemon.machop",
			Level: progression.LevelCap, Slot: entry.Slot,
		})
	}
	err = squad.Validate()
	if err == nil {
		t.Fatalf("a saved squad of %d was accepted: the board admits that many and no side "+
			"is ever FIELDED with more than %d", hex.BoardSlots, hex.MaxSquadSize)
	}
	if !strings.Contains(err.Error(), "a side can field") {
		t.Errorf("the refusal does not say a side cannot field that many: %v", err)
	}
	// The control: one short of the fielding cap is accepted, so the refusal
	// above is a size rule and not the fixture being malformed.
	squad.Units = squad.Units[:hex.MaxSquadSize]
	if err := squad.Validate(); err != nil {
		t.Errorf("a squad of %d was refused, and that is what a side fields: %v",
			hex.MaxSquadSize, err)
	}
}
