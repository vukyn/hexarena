package pattern_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
)

// TestAShapeCoversTheMirrorOfWhatItCoversForTheOtherSide is the property the
// whole board rests on and the one that was silently false.
//
// hex.Place puts the enemy half down under a 180 degree rotation, so a battle
// and the same battle with the sides exchanged are the same battle relabelled —
// and every rule has to commute with that relabelling or they are two different
// games. A shape did not: its steps were walked in absolute board directions, so
// `arc_up` pointed up for both sides where the mirror needs it to point down for
// one of them, and `pierce` spread back towards the midline in the hands of an
// enemy instead of through the formation.
//
// What is asserted, for every shape in the book and every cell of the board:
// the cells an enemy's cast covers are exactly the mirror images of the cells an
// ally's cast covers from the mirrored aim. Not the same count — the same cells.
// A count held while this was broken, which is why counting was not enough.
func TestAShapeCoversTheMirrorOfWhatItCoversForTheOtherSide(t *testing.T) {
	for _, shape := range everyShape() {
		for _, across := range []bool{false, true} {
			for _, aim := range hex.Cells() {
				ally := walk(shape, aim, hex.SideAlly, across)
				enemy := walk(shape, hex.Place(hex.SideEnemy, aim), hex.SideEnemy, across)
				if len(ally) != len(enemy) {
					t.Fatalf("%s across=%v from %v: the ally covers %d cells and the "+
						"enemy covers %d from the mirrored aim",
						shape.Name, across, aim, len(ally), len(enemy))
				}
				for i, cell := range ally {
					want := hex.Place(hex.SideEnemy, cell)
					if enemy[i] == want {
						continue
					}
					t.Fatalf("%s across=%v from %v: the ally's cell %d is %v, whose "+
						"mirror is %v, and the enemy covers %v — the two halves are not "+
						"each other's reflection",
						shape.Name, across, aim, i, cell, want, enemy[i])
				}
			}
		}
	}
}

// TestAnAllysShapeIsUnchangedByTheFrame is the other half, and it is what says
// the fix is a fix rather than a rebalancing of both sides at once.
//
// The direction names are written in the ally's frame, so an ally's cast has to
// come out of the walk exactly as it did when the walk had no frame at all: the
// cells are the primary's cube neighbours along the declared steps, in board
// coordinates. Anything else would mean every figure ever measured of the home
// squad had moved too.
func TestAnAllysShapeIsUnchangedByTheFrame(t *testing.T) {
	for _, shape := range everyShape() {
		for _, aim := range hex.Cells() {
			for _, across := range []bool{false, true} {
				got := walk(shape, aim, hex.SideAlly, across)
				want := absoluteWalk(shape, aim, across)
				if len(got) != len(want) {
					t.Fatalf("%s across=%v from %v: %d cells, want the %d an absolute "+
						"walk gives", shape.Name, across, aim, len(got), len(want))
				}
				for i := range got {
					if got[i] != want[i] {
						t.Fatalf("%s across=%v from %v: cell %d is %v, want %v",
							shape.Name, across, aim, i, got[i], want[i])
					}
				}
			}
		}
	}
}

// everyShape is a shape for each of the six directions, one for each ordered
// pair of them, and one two-step chain per direction.
//
// It is generated rather than read out of the shipped book on purpose: the
// property is about the geometry and must not become a statement about which
// shapes an author happens to have written down today. The shipped book uses
// four of the six directions, so a test over it would leave `lower_left` and
// `upper_left` unwalked — and those are exactly the two an enemy's `pierce`
// lands on now.
func everyShape() []pattern.Pattern {
	out := []pattern.Pattern{{Name: "single"}}
	for _, first := range pattern.Directions() {
		out = append(out, pattern.Pattern{
			Name:   "one-" + first.String(),
			Splash: [][]pattern.Direction{{first}},
		})
		out = append(out, pattern.Pattern{
			Name:   "two-" + first.String(),
			Splash: [][]pattern.Direction{{first}, {first, first}},
		})
		for _, second := range pattern.Directions() {
			out = append(out, pattern.Pattern{
				Name:   first.String() + "+" + second.String(),
				Splash: [][]pattern.Direction{{first}, {second}},
			})
		}
	}
	return out
}

func walk(shape pattern.Pattern, aim hex.Offset, caster hex.Side, across bool) []hex.Offset {
	if across {
		return shape.TargetsAcross(aim, caster)
	}
	return shape.Targets(aim, caster)
}

// absoluteWalk is the walk this package did before it had a caster's frame: the
// steps added to the primary in board coordinates, with the same two drops.
// Written out rather than called, because the point of it is to be the *old*
// implementation.
func absoluteWalk(shape pattern.Pattern, primary hex.Offset, across bool) []hex.Offset {
	if !primary.OnBoard() {
		return nil
	}
	side := primary.Side()
	out := []hex.Offset{primary}
	seen := map[hex.Offset]bool{primary: true}
	for _, chain := range shape.Splash {
		cube := primary.Cube()
		for _, step := range chain {
			cube = cube.Add(step.Step())
		}
		cell := cube.Offset()
		if !cell.OnBoard() || seen[cell] {
			continue
		}
		if !across && cell.Side() != side {
			continue
		}
		seen[cell] = true
		out = append(out, cell)
	}
	return out
}
