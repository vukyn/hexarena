package composition_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestARungAboveTheLargestSquadIsRefused is the rung ceiling, which had no test
// at all until ENG-013 split the cap that feeds it.
//
// **A rung is measured against the ROSTER, so the roster's cap is the ceiling.**
// battle.New resolves the awards while the roster is still a slice of facts and
// nothing recounts, so a summon never earns one — the count a rung is compared
// against can therefore never exceed what a side was FIELDED with. That is
// hex.MaxSquadSize, and it is deliberately not hex.BoardSlots.
//
// Three clauses, and the third is the only one a regression would show up in:
//
//   - a rung AT the cap parses. It is reachable by a full side, so refusing it
//     would take the top of the ladder away.
//   - a rung one above the cap is refused. Nothing can reach it.
//   - ⚠️ **a rung at hex.BoardSlots is refused too.** This is the clause that
//     holds the classification. The board admits more units than a side may
//     field, so pointing this ceiling at the board would let an author declare
//     rungs 6 to 9 — rows that load, draw and can never fire, which is exactly
//     what DAT-002's decision 6 refuses ("a rung that cannot fire is not declared
//     at all, so there is no row for a test to pass vacuously over"). The first
//     two clauses are green whichever constant the ceiling reads.
func TestARungAboveTheLargestSquadIsRefused(t *testing.T) {
	// One rung, at whatever height the case is about. Two stacks of a permanent
	// buff, which is the shape every shipped bonus has.
	book := func(at int) string {
		return fmt.Sprintf(`{"bonuses": [
		  {"id": "kin", "name": "đồng hệ", "axis": "element", "scope": "sharers", "rungs": [
		    {"at": %d, "grants": [{"status": "kinship", "stacks": 1}]}
		  ]}
		]}`, at)
	}

	if _, err := parse(t, book(hex.MaxSquadSize)); err != nil {
		t.Errorf("a rung at %d was refused, and a full side reaches it: %v",
			hex.MaxSquadSize, err)
	}
	for _, at := range []int{hex.MaxSquadSize + 1, hex.BoardSlots} {
		_, err := parse(t, book(at))
		if err == nil {
			t.Errorf("a rung at %d parsed: no side is ever fielded with more than %d, so "+
				"it is a row that loads, draws and never fires", at, hex.MaxSquadSize)
			continue
		}
		if !strings.Contains(err.Error(), "can reach") {
			t.Errorf("the refusal of a rung at %d does not say nobody reaches it: %v", at, err)
		}
	}
}
