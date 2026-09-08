package seed_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/seed"
)

// aFormation is one side's worth of authoring slots under a name, plus whether
// anything ever fights it.
//
// The distinction is the whole point of the fixture: `forge.FightSquads` fights
// the squads and nothing else, so a property held only by the seed roster is a
// property no rate in this repository is taken on.
type aFormation struct {
	name   string
	slots  []hex.Offset
	fought bool
}

// TestAShippedFormationIsCatchableByTheShapesItFields is the occupancy half of
// the pattern contract, and `TestShippedPatternBook` is the other half.
//
// That one asks whether a shape can cover its full width somewhere on the enemy
// half — a statement about CELLS. This one asks whether any shipped formation
// puts two units where one cast catches both — a statement about OCCUPANTS —
// and the two are independent: every shape in the book passes the first while
// catching exactly one unit on every board `forge.FightSquads` fights.
//
// It matters because `splash_power` is only reached from the second cell onward
// (`pattern.Book.Power`). A shape that catches one occupied cell resolves as a
// single-target skill of the same power, so an area skill priced against its
// area is priced against something that board cannot redeem.
//
// ⚠️ **What it holds and what it deliberately does not.** The assertion is over
// every shipped formation, and today it passes on the seed roster alone: that
// board stacks its pair at (1,0)+(1,1) and the five squads do not stack at all.
// So the shipped state is *the axis exists in the game and is absent from every
// board a rate is read off*, which is `DAT-007` and is open. Turning the logged
// half into an assertion would be filing that finding as a red test rather than
// as an item; what this guards is the floor under it — the day the roster is
// flattened too, `splash_power` becomes unreachable everywhere and no shape in
// the book means what it says.
//
// The shapes are derived from what the formations actually field, never from a
// written-down list: a named skill goes stale silently the day a kit is
// rebalanced, and a shape nobody carries is a shape no rate depends on.
func TestAShippedFormationIsCatchableByTheShapesItFields(t *testing.T) {
	skills := mustSkills(t)
	patterns := mustBook(t)

	boards := shippedFormations(t)
	// The shapes some shipped placement really brings, and who brings them.
	fielded := map[string][]string{}
	note := func(where string, carried []string) {
		for _, id := range carried {
			known, err := skills.Lookup(id)
			if err != nil {
				t.Fatalf("%s names skill %q: %v", where, id, err)
			}
			shape, err := patterns.Lookup(known.Pattern)
			if err != nil {
				t.Fatalf("skill %q names shape %q: %v", id, known.Pattern, err)
			}
			if shape.MaxTargets() < 2 {
				continue
			}
			if !slices.Contains(fielded[shape.Name], where) {
				fielded[shape.Name] = append(fielded[shape.Name], where)
			}
		}
	}
	for _, squad := range mustSquads(t) {
		for _, unit := range squad.Units {
			note(squad.ID+"/"+unit.ID, unit.Skills)
		}
	}
	for _, unit := range mustSeedRoster(t) {
		note(unit.ID, unit.Skills)
	}
	// Fail loudly when the search finds nothing. A green run that measured no
	// area skill at all is the failure this test exists to prevent.
	if len(fielded) == 0 {
		t.Fatal("no shipped placement fields a skill whose shape catches more than one cell, " +
			"so this test measured nothing: give some formation an area carrier")
	}

	for _, name := range slices.Sorted(maps.Keys(fielded)) {
		shape, err := patterns.Lookup(name)
		if err != nil {
			t.Fatalf("look up shape %q: %v", name, err)
		}
		best, on, bestFought, onFought := 0, "", 0, ""
		for _, board := range boards {
			caught := mostOccupiedCellsCaught(shape, board.slots)
			if caught > best {
				best, on = caught, board.name
			}
			if board.fought && caught > bestFought {
				bestFought, onFought = caught, board.name
			}
		}
		if best < 2 {
			t.Errorf("%s is fielded by %s and catches %d occupied cell on every shipped formation: "+
				"splash_power is never reached anywhere in the game, so every %s skill resolves "+
				"as a single-target one",
				name, strings.Join(fielded[name], ", "), best, name)
			continue
		}
		if bestFought < 2 {
			// DAT-007. Not an assertion: see the note on this test.
			t.Logf("%-10s catches %d on %s, and %d on every board a rate is read off "+
				"(fielded by %s)", name, best, on, bestFought, strings.Join(fielded[name], ", "))
			continue
		}
		t.Logf("%-10s catches %d on %s, %d on %s (fielded by %s)",
			name, best, on, bestFought, onFought, strings.Join(fielded[name], ", "))
	}
}

// mostOccupiedCellsCaught is the most units one cast of a shape can catch on a
// formation.
//
// It aims at occupied cells only, because `battle.aims` offers nothing else, and
// it counts occupied cells only, because an empty cell in a shape takes no
// damage and buys no splash. It answers for the formation standing on **either**
// half, and takes the better: `forge.FightSquads` fields every squad on both,
// and `hex.Place` turns an enemy formation through 180 degrees, which flips
// column parity and so can move what an arc reaches.
//
// The walk is `covers` reduced to its geometry. A skill that crosses the midline
// is not what this measures, since a formation is one side's worth.
func mostOccupiedCellsCaught(shape pattern.Pattern, slots []hex.Offset) int {
	best := 0
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		occupied := map[hex.Offset]bool{}
		aims := make([]hex.Offset, 0, len(slots))
		for _, slot := range slots {
			cell := hex.Place(side, slot)
			if !occupied[cell] {
				occupied[cell] = true
				aims = append(aims, cell)
			}
		}
		for _, aim := range aims {
			caught := 0
			for _, cell := range shape.Targets(aim) {
				if occupied[cell] {
					caught++
				}
			}
			if caught > best {
				best = caught
			}
		}
	}
	return best
}

// shippedFormations is every placement the game ships: the squads two squads are
// fought as, and the two halves of the seed roster.
func shippedFormations(t *testing.T) []aFormation {
	t.Helper()
	var out []aFormation
	for _, squad := range mustSquads(t) {
		board := aFormation{name: squad.ID, fought: true}
		for _, unit := range squad.Units {
			board.slots = append(board.slots, unit.Slot)
		}
		out = append(out, board)
	}
	halves := map[hex.Side][]hex.Offset{}
	for _, unit := range mustSeedRoster(t) {
		halves[unit.Side] = append(halves[unit.Side], unit.Slot)
	}
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		if slots := halves[side]; len(slots) > 0 {
			out = append(out, aFormation{name: "roster/" + side.String(), slots: slots})
		}
	}
	return out
}

func mustSquads(t *testing.T) []placement.Squad {
	t.Helper()
	squads, err := seed.Squads()
	if err != nil {
		t.Fatalf("load shipped squads: %v", err)
	}
	return squads
}

func mustSeedRoster(t *testing.T) []battle.Roster {
	t.Helper()
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load the shipped roster: %v", err)
	}
	return roster
}
