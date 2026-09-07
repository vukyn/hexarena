package i18n

import (
	"sort"
	"testing"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestNoStatusGlossNamesSomethingUnshipped is the direction the suite did not
// have, and retiring a status is what showed the hole.
//
// TestEveryShippedStatusIsGlossed walks the shipped book and asks the table for
// each id. That catches a status arriving without a name and, by its own comment,
// deliberately lets an id the table names but nothing ships go by — "an unshipped
// id is still free to miss, so this does not become a second place a status has to
// be registered". Free to *miss* is not the same as free to *linger*: `kinship`
// went out of statuses.json with `same_element`, and had its gloss stayed behind,
// nothing anywhere would have said so. The table would have gone on translating an
// effect no unit in the game can hold, and the next reader would have taken the
// entry as evidence the status still exists.
//
// ⚠️ The three column values are glossed out of this same table and are not
// statuses at all — `column0` and friends are values on the composition axis, and
// a bonus line reading "three of 2" says nothing. So what the table may name is
// derived from the axes rather than listed: a status a unit can hold, an element a
// squad can share, or a column it can stand in. Through `composition.ColumnValue`
// rather than through a second "column" + itoa here, and through
// `hex.FormationCols` rather than through the three that exist today, because a
// fourth column arrives with 5v5 and a hardcoded list would refuse it.
func TestNoStatusGlossNamesSomethingUnshipped(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	// What the table is allowed to name, derived from the axes rather than listed.
	// A gloss earns its place by being something a tally can actually produce: a
	// status a unit can hold, an element a squad can share, or a column it can
	// stand in.
	known := make(map[string]bool)
	for _, kind := range statuses.Kinds() {
		known[kind.ID] = true
	}
	for _, member := range element.All() {
		known[member.String()] = true
	}
	for column := range hex.FormationCols {
		known[composition.ColumnValue(column)] = true
	}
	if len(known) == 0 {
		t.Fatal("nothing ships, so this asserts nothing")
	}
	orphans := []string{}
	for id := range statusGloss {
		if !known[id] {
			orphans = append(orphans, id)
		}
	}
	sort.Strings(orphans)
	for _, id := range orphans {
		t.Errorf("statusGloss names %q and nothing in the shipped data does: a gloss for a "+
			"status or a value that was retired goes on translating an effect no unit can "+
			"hold, and reads as evidence the thing still exists", id)
	}
}
