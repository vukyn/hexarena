// Package hex implements the battlefield geometry for the arena.
//
// The battlefield is one shared odd-q offset hex grid, 6 columns by 3 rows.
// Columns run left to right from the ally backline to the enemy backline; odd
// columns sit half a cell lower than even ones, which is what gives a
// frontline cell contact with two cells of the next column.
//
//	 c0  c1  c2  c3  c4  c5      c0-c2 = ally  (c2 = frontline)
//	 BK  MD  FR  FR  MD  BK      c3-c5 = enemy (c3 = frontline)
//
//	 __    __    __
//	/00\__/20\__/40\__
//	\__/10\__/30\__/50\
//	/01\__/21\__/41\__/
//	\__/11\__/31\__/51\
//	/02\__/22\__/42\__/
//	\__/12\__/32\__/52\
//	   \__/  \__/  \__/
//
// Offset coordinates exist only for authoring formations and for rendering.
// Every distance and area calculation runs on cube coordinates, where the hex
// metric is a plain max-of-absolute-differences and there is no diagonal case
// to get wrong.
//
// Everything here is a pure function of its integer arguments: no floating
// point, no map iteration, no randomness, no clock. Results are therefore
// stable across runs and platforms, which is what lets the battle engine be
// replayed from a seed.
package hex

import (
	"fmt"
	"strings"
)

// Battlefield dimensions. Columns run ally backline to enemy backline.
const (
	Cols = 6
	Rows = 3

	AllyBackCol   = 0
	AllyFrontCol  = 2
	EnemyFrontCol = 3
	EnemyBackCol  = 5

	// FormationCols is the width of a single team's authoring grid: every
	// team authors its formation as 3 columns of 3 rows regardless of side.
	FormationCols = 3
	// FormationRows mirrors Rows and exists for symmetry at call sites.
	FormationRows = Rows
	// MaxTeamSize is how many of the nine formation slots a team may fill.
	MaxTeamSize = 5
)

// Side identifies which half of the battlefield a coordinate belongs to.
type Side uint8

const (
	SideAlly Side = iota
	SideEnemy
)

func (s Side) String() string {
	switch s {
	case SideAlly:
		return "ally"
	case SideEnemy:
		return "enemy"
	default:
		return "unknown"
	}
}

// Offset is an odd-q offset coordinate: odd columns sit half a cell lower
// than even ones.
type Offset struct {
	Col, Row int
}

func (o Offset) String() string { return fmt.Sprintf("%d,%d", o.Col, o.Row) }

// OnBoard reports whether the coordinate lies inside the battlefield.
func (o Offset) OnBoard() bool {
	return o.Col >= 0 && o.Col < Cols && o.Row >= 0 && o.Row < Rows
}

// Side reports which half of the battlefield the coordinate sits in. It is
// only meaningful for on-board coordinates.
func (o Offset) Side() Side {
	if o.Col <= AllyFrontCol {
		return SideAlly
	}
	return SideEnemy
}

// Cube converts to cube coordinates, where all distance math happens.
func (o Offset) Cube() Cube {
	x := o.Col
	z := o.Row - (o.Col-(o.Col&1))/2
	return Cube{X: x, Y: -x - z, Z: z}
}

// Cube is a cube hex coordinate; the three axes always sum to zero.
type Cube struct {
	X, Y, Z int
}

func (c Cube) String() string { return fmt.Sprintf("(%d,%d,%d)", c.X, c.Y, c.Z) }

// Offset converts back to odd-q offset coordinates.
func (c Cube) Offset() Offset {
	return Offset{Col: c.X, Row: c.Z + (c.X-(c.X&1))/2}
}

// Add returns the coordinate translated by d.
func (c Cube) Add(d Cube) Cube { return Cube{c.X + d.X, c.Y + d.Y, c.Z + d.Z} }

// Scale returns the coordinate multiplied by n, used to step n cells along a
// direction.
func (c Cube) Scale(n int) Cube { return Cube{c.X * n, c.Y * n, c.Z * n} }

// Directions holds the six cube neighbour offsets in a fixed order, so every
// list derived from them — neighbours, rings, disks — has a stable ordering.
var Directions = [6]Cube{
	{+1, -1, 0},
	{+1, 0, -1},
	{0, +1, -1},
	{-1, +1, 0},
	{-1, 0, +1},
	{0, -1, +1},
}

// Distance returns the hex distance between two cube coordinates.
func Distance(a, b Cube) int {
	return max(abs(a.X-b.X), abs(a.Y-b.Y), abs(a.Z-b.Z))
}

// DistanceTo returns the hex distance between two offset coordinates. This is
// the number an attack range is compared against.
func (o Offset) DistanceTo(p Offset) int { return Distance(o.Cube(), p.Cube()) }

// Neighbors returns the six adjacent coordinates in Directions order,
// including any that fall outside the battlefield.
func (o Offset) Neighbors() []Offset {
	c := o.Cube()
	out := make([]Offset, 0, 6)
	for _, d := range Directions {
		out = append(out, c.Add(d).Offset())
	}
	return out
}

// NeighborsOnBoard returns only the adjacent coordinates inside the
// battlefield. Edge cells have fewer than six.
func (o Offset) NeighborsOnBoard() []Offset {
	out := make([]Offset, 0, 6)
	for _, n := range o.Neighbors() {
		if n.OnBoard() {
			out = append(out, n)
		}
	}
	return out
}

// Ring returns the cells exactly radius steps from center, walking the six
// directions from a fixed start so the order is reproducible. A radius of
// zero yields the center itself.
func Ring(center Cube, radius int) []Cube {
	if radius < 0 {
		return nil
	}
	if radius == 0 {
		return []Cube{center}
	}
	out := make([]Cube, 0, 6*radius)
	current := center.Add(Directions[4].Scale(radius))
	for i := range Directions {
		for step := 0; step < radius; step++ {
			out = append(out, current)
			current = current.Add(Directions[i])
		}
	}
	return out
}

// Disk returns every cell within radius steps of center, center first, then
// ring by ring outward.
func Disk(center Cube, radius int) []Cube {
	out := make([]Cube, 0, 1+3*radius*(radius+1))
	for r := 0; r <= radius; r++ {
		out = append(out, Ring(center, r)...)
	}
	return out
}

// InRange returns the on-board cells within radius of origin, excluding
// origin itself. This is the candidate set a skill with that range may target
// before any side or occupancy filter is applied.
func InRange(origin Offset, radius int) []Offset {
	out := make([]Offset, 0, 3*radius*(radius+1))
	for _, c := range Disk(origin.Cube(), radius) {
		cell := c.Offset()
		if cell == origin || !cell.OnBoard() {
			continue
		}
		out = append(out, cell)
	}
	return out
}

// Place maps a formation-authoring slot onto the shared battlefield.
//
// Both teams author their formation the same way — column 0 is their own
// backline, column 2 their frontline — and the enemy formation is rotated 180
// degrees about the centre of the board. Without that rotation the two halves
// would not mirror each other: odd columns are pushed downward, so an ally
// frontline slot and the enemy slot authored identically would have different
// distance profiles and the matchup would be silently unbalanced.
func Place(side Side, author Offset) Offset {
	if side == SideEnemy {
		return Offset{Col: Cols - 1 - author.Col, Row: Rows - 1 - author.Row}
	}
	return author
}

// Cells returns every battlefield coordinate, column-major.
func Cells() []Offset {
	out := make([]Offset, 0, Cols*Rows)
	for col := 0; col < Cols; col++ {
		for row := 0; row < Rows; row++ {
			out = append(out, Offset{Col: col, Row: row})
		}
	}
	return out
}

// SideCells returns the nine battlefield coordinates belonging to one side,
// column-major.
func SideCells(side Side) []Offset {
	out := make([]Offset, 0, FormationCols*Rows)
	for _, cell := range Cells() {
		if cell.Side() == side {
			out = append(out, cell)
		}
	}
	return out
}

// Render draws the battlefield as interlocking ASCII hexes, which is how the
// terminal client shows a formation. label is called once per cell and may
// return up to two characters; anything shorter is padded, longer is cut.
func Render(label func(Offset) string) string {
	const (
		width  = 3*Cols + 1
		height = 2*Rows + 2
	)
	canvas := make([][]byte, height)
	for i := range canvas {
		canvas[i] = []byte(strings.Repeat(" ", width))
	}
	put := func(x, y int, b byte) {
		if y >= 0 && y < height && x >= 0 && x < width {
			canvas[y][x] = b
		}
	}
	for _, cell := range Cells() {
		x := 3 * cell.Col
		y := 2*cell.Row + (cell.Col & 1)
		put(x+1, y, '_')
		put(x+2, y, '_')
		put(x, y+1, '/')
		put(x+3, y+1, '\\')
		put(x, y+2, '\\')
		put(x+1, y+2, '_')
		put(x+2, y+2, '_')
		put(x+3, y+2, '/')

		text := label(cell)
		if len(text) > 2 {
			text = text[:2]
		}
		for len(text) < 2 {
			text += " "
		}
		put(x+1, y+1, text[0])
		put(x+2, y+1, text[1])
	}
	lines := make([]string, 0, height)
	for _, row := range canvas {
		lines = append(lines, strings.TrimRight(string(row), " "))
	}
	return strings.Join(lines, "\n")
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
