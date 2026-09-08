package seed_test

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/skill"
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

// anAllyAimedCarrier is one placement that really brings an ally-aimed area
// skill, and the halves its formation is fought on.
type anAllyAimedCarrier struct {
	where string
	skill string
	shape pattern.Pattern
	slots []hex.Offset
	// sides is the halves this formation really stands on. A squad is fought on
	// both, because forge.FightSquads swaps the arrangement; a seed roster entry
	// stands on the one side it declares.
	sides []hex.Side
}

// TestAnAllyAimedShapeIsCatchableFromItsOwnCarriersHalf is the friendly half of
// the same occupancy question, and it is a sibling of
// TestAShippedFormationIsCatchableByTheShapesItFields rather than a widening of
// it, because it needs a different reduction.
//
// That one takes the BEST of the two halves, which is the right question for an
// enemy-aimed shape: a squad only has to find one arrangement where its area
// skill pays. For an ally-aimed one it is the wrong question. forge.FightSquads
// fights every squad on both halves over the same seeds, so what a support shape
// is worth to a squad is bounded by the WORSE half, not the better one — a
// reading taken off the max would quote a squad a number it holds in half its
// battles.
//
// # What an ally-aimed skill may aim at
//
// Every occupied cell on the caster's own half, its own cell included, and no
// range filter at all: reach is counted in ranks from the far side, so helping
// the squad you stand in is not a question of distance
// (internal/core/battle/turn.go:588-610, and the two rules hanging off the rank
// rule in CLAUDE.md). That is the non-obvious half — the caster is not
// restricted to buffing itself, and a support unit in the back rank may aim at
// the front one — which is why this walk aims over the whole occupied half
// rather than from the carrier's own cell outward.
//
// # What it holds, and what it deliberately does not
//
// The floor is that a fielded ally-aimed area skill covers at least one ally.
// Nought would mean a support unit could spend a turn buffing nobody, which the
// aim rule above makes impossible and which no reading in DAT-009 would be
// interpretable without.
//
// ⚠️ **Whether the two halves AGREE is logged and not asserted, on purpose.**
// They do agree, on all five shipped squads and for every shape in the book —
// hex.Place rotates an enemy formation through 180 degrees and every shipped
// squad formation is symmetric under that rotation, so no shape can tell the
// halves apart on any board a rate is read off. An assertion whose branch no
// shipped data can exercise is a fixture hiding a branch, and this repository
// has paid for that five times. The divergence is real on roster.json (arc_up
// reaches 2 on one half and 3 on the other) and roster.json fields no ally-aimed
// skill at all.
//
// # DAT-009, measured and refused
//
// rally on s01's clefable is the ONLY ally-aimed area skill any shipped squad
// fields, and it catches 1 on both halves. DAT-009 proposed moving a support
// column onto arc_up to fix that; arc_up catches 1 on s01 too, exactly as column
// does, and an arm that reshaped rally onto it reproduced the baseline battle for
// battle. pierce is the one shape in the book that catches 2 on s01, and it was
// measured: the aggregate over the three boards whose baseline rate is inside
// 50…950‰ moved from 145‰ to 145‰ — one battle in twelve hundred — against a
// gate of 50‰ written down first, and s04 moved the opposite way from s02 and
// s03. So the count stays 1 and this test logs it rather than reddening: filing
// an open item as a failing test is not filing it, which is the same reason the
// neighbouring test logs its own half.
//
// The carriers are derived from what the formations really field, never from a
// written-down skill list, for the reason recorded on that test.
func TestAnAllyAimedShapeIsCatchableFromItsOwnCarriersHalf(t *testing.T) {
	skills := mustSkills(t)
	patterns := mustBook(t)

	var carriers []anAllyAimedCarrier
	collect := func(where string, slots []hex.Offset, carried []string, sides ...hex.Side) {
		for _, id := range carried {
			known, err := skills.Lookup(id)
			if err != nil {
				t.Fatalf("%s names skill %q: %v", where, id, err)
			}
			if known.Target != skill.Ally {
				continue
			}
			shape, err := patterns.Lookup(known.Pattern)
			if err != nil {
				t.Fatalf("skill %q names shape %q: %v", id, known.Pattern, err)
			}
			if shape.MaxTargets() < 2 {
				continue
			}
			carriers = append(carriers, anAllyAimedCarrier{
				where: where, skill: id, shape: shape, slots: slots, sides: sides,
			})
		}
	}
	for _, squad := range mustSquads(t) {
		slots := make([]hex.Offset, 0, len(squad.Units))
		for _, unit := range squad.Units {
			slots = append(slots, unit.Slot)
		}
		for _, unit := range squad.Units {
			collect(squad.ID+"/"+unit.ID, slots, unit.Skills, hex.SideAlly, hex.SideEnemy)
		}
	}
	halves := map[hex.Side][]hex.Offset{}
	for _, unit := range mustSeedRoster(t) {
		halves[unit.Side] = append(halves[unit.Side], unit.Slot)
	}
	for _, unit := range mustSeedRoster(t) {
		collect("roster/"+unit.ID, halves[unit.Side], unit.Skills, unit.Side)
	}

	// Fail loudly when the search finds nothing, for the reason its neighbour
	// does: a green run that measured no ally-aimed area skill at all has not
	// held anything down.
	if len(carriers) == 0 {
		t.Fatal("no shipped placement fields an ally-aimed skill whose shape catches more than " +
			"one cell, so this test measured nothing: give some formation a support carrier")
	}

	for _, carrier := range carriers {
		worst, best := 0, 0
		perHalf := make([]string, 0, len(carrier.sides))
		for index, side := range carrier.sides {
			caught := alliesCaughtOnHalf(carrier.shape, carrier.slots, side)
			perHalf = append(perHalf, side.String()+" "+strconv.Itoa(caught))
			if index == 0 || caught < worst {
				worst = caught
			}
			best = max(best, caught)
		}
		if worst < 1 {
			t.Errorf("%s carries %s (%s) and catches no ally at all on %s: a support skill "+
				"always covers its own aim, so nought means the walk or the aim rule has moved",
				carrier.where, carrier.skill, carrier.shape.Name, strings.Join(perHalf, ", "))
			continue
		}
		if worst < 2 {
			// DAT-009, measured and refused. Not an assertion: see the note on
			// this test.
			t.Logf("%-18s carries %-10s (%-10s) and buffs %d ally on the half it is worst on "+
				"(%s): every cast of it is a single-target one",
				carrier.where, carrier.skill, carrier.shape.Name, worst, strings.Join(perHalf, ", "))
			continue
		}
		t.Logf("%-18s carries %-10s (%-10s) reaches %d..%d allies (%s)",
			carrier.where, carrier.skill, carrier.shape.Name, worst, best, strings.Join(perHalf, ", "))
	}
}

// alliesCaughtOnHalf is the most occupied ALLY cells one cast of a shape catches
// on a formation standing on one named half.
//
// It differs from mostOccupiedCellsCaught in the reduction and nothing else: one
// half at a time rather than the better of the two. The aim set is every
// occupied cell on that half, which is exactly what battle.aims offers an
// ally-aimed skill — no range filter and the caster's own cell included — so the
// carrier's own slot does not narrow it. That is a property of the friendly half
// alone: on the hostile half reach is counted in occupied ranks and the aim set
// really does depend on where the caster stands.
func alliesCaughtOnHalf(shape pattern.Pattern, slots []hex.Offset, side hex.Side) int {
	occupied := map[hex.Offset]bool{}
	aims := make([]hex.Offset, 0, len(slots))
	for _, slot := range slots {
		cell := hex.Place(side, slot)
		if !occupied[cell] {
			occupied[cell] = true
			aims = append(aims, cell)
		}
	}
	best := 0
	for _, aim := range aims {
		caught := 0
		for _, cell := range shape.Targets(aim) {
			if occupied[cell] {
				caught++
			}
		}
		best = max(best, caught)
	}
	return best
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
//
// ⚠️ It is deliberately carrier-blind and takes the BETTER half, which is the
// right reduction for an enemy-aimed shape and the wrong one for an ally-aimed
// one — see TestAnAllyAimedShapeIsCatchableFromItsOwnCarriersHalf and DAT-009.
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
