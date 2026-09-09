package hex

import "testing"

// TestTheBoardAdmitsEveryFormationSlot is the first half of the ENG-013 split,
// and it is the half a literal would quietly break.
//
// BoardSlots is a claim ABOUT the grid — how many units one side can have
// standing at once — so it is derived from the grid rather than written out. A
// number typed here would go on saying nine on the day somebody gave a formation
// a fourth column, and every summon would then stop one short of the board with
// nothing to say so.
func TestTheBoardAdmitsEveryFormationSlot(t *testing.T) {
	if BoardSlots != FormationCols*FormationRows {
		t.Errorf("the board admits %d units a side against %dx%d formation slots",
			BoardSlots, FormationCols, FormationRows)
	}
	if FormationRows != Rows {
		t.Errorf("a formation is %d rows deep and the board is %d", FormationRows, Rows)
	}
}

// TestASquadTheBoardCannotSeatIsRefused is the invariant BETWEEN the two caps,
// and no other test in the module would catch it breaking.
//
// A side is fielded with MaxSquadSize units and stands on BoardSlots cells, so a
// fielding cap above the board would author squads that cannot be enlisted —
// battle.New would refuse a legal saved squad, which is a failure with no
// sensible message at either end.
//
// ⚠️ The inequality is STRICT on purpose, and that is the whole of ENG-013. Equal
// caps are exactly what shipped: a full side then had no cell left over,
// battle.summonPlaces computed nought room, summonWorth priced every summoning
// skill at nought and the rating never cast one. The gap is not slack, it is
// where a summon stands.
func TestASquadTheBoardCannotSeatIsRefused(t *testing.T) {
	if MaxSquadSize >= BoardSlots {
		t.Errorf("a side is fielded with %d units on a board of %d slots: a summon lives "+
			"in the gap between the two, and there has to be one", MaxSquadSize, BoardSlots)
	}
	if MaxSquadSize < 1 {
		t.Errorf("a side is fielded with %d units, which is nobody", MaxSquadSize)
	}
}
