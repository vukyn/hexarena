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
	"encoding/json"
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
	// SideNone is no side at all, and it is the zero value on purpose.
	//
	// A side is written to a battle log with `omitempty`, so whichever side is
	// zero is the one that never appears on the wire — and a reader would then
	// be recovering it from the absence of a field rather than from the log.
	// That is exactly the trap a draw with no stated reason sets: the fact is
	// right by coincidence until the day somebody reorders a constant. With
	// nothing at zero, an absent side means what it says, and a battle whose
	// winner is undecided writes no winner rather than quietly writing the
	// first one declared.
	SideNone Side = iota
	SideAlly
	SideEnemy
)

// Fights reports whether the side is one a unit can be on. SideNone is not.
func (s Side) Fights() bool { return s == SideAlly || s == SideEnemy }

func (s Side) String() string {
	switch s {
	case SideAlly:
		return "ally"
	case SideEnemy:
		return "enemy"
	case SideNone:
		return "none"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the side by name.
//
// A number would tie a saved battle log to the order these constants happen to
// be declared in, so inserting a side later would silently reinterpret every log
// already written.
func (s Side) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// UnmarshalJSON reads a side written by name.
func (s *Side) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode side: %w", err)
	}
	switch name {
	case "ally":
		*s = SideAlly
	case "enemy":
		*s = SideEnemy
	case "none":
		*s = SideNone
	default:
		return fmt.Errorf("unknown side %q", name)
	}
	return nil
}

// Offset is an odd-q offset coordinate: odd columns sit half a cell lower
// than even ones.
type Offset struct {
	Col int `json:"col"`
	Row int `json:"row"`
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

// Cell is a place on the board that may be no place at all, which an Offset
// cannot be.
//
// Offset's zero value is already spoken for: {0,0} is the ally back corner, a
// real cell somebody stands in, so there is no coordinate left over to mean
// "nowhere". That is what made a cell on a battle event dishonest. `omitempty`
// does nothing to a struct field, so every event with no cell to report wrote
// the back corner into the log regardless — and only a handful of the kinds ever
// meant one, while `omitzero` on the offset would have gone wrong the other way
// and dropped the corner from the events that did. Neither tag can tell the two
// apart, because the coordinate carries no room to say. So the absence lives in
// a second field instead, where it is a fact of its own rather than a
// coordinate pressed into doubling as one, and the zero value means absent — an
// omitted field therefore reads back as the Cell it was written from.
//
// It is a value and not a pointer because a log is verified by comparing whole
// events with ==, and two pointers satisfy that by address: a re-run would
// differ from the battle it had just reproduced exactly, and nothing would fail
// to compile on the way there.
type Cell struct {
	offset Offset
	filled bool
}

// At is the cell at a coordinate. The zero Cell, which this never returns, is
// the one that is nowhere.
func At(offset Offset) Cell { return Cell{offset: offset, filled: true} }

// Offset reports the coordinate and whether there is one, in the comma-ok shape
// a caller cannot read past without answering the second question.
func (c Cell) Offset() (Offset, bool) { return c.offset, c.filled }

func (c Cell) String() string {
	if !c.filled {
		return "none"
	}
	return c.offset.String()
}

// MarshalJSON writes the bare coordinate, so a cell reads on the wire exactly as
// an offset always did, and null when there is no cell.
func (c Cell) MarshalJSON() ([]byte, error) {
	if !c.filled {
		return []byte("null"), nil
	}
	return json.Marshal(c.offset)
}

// UnmarshalJSON reads a coordinate, taking null — and, by never being called at
// all, an absent field — as no cell.
func (c *Cell) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		*c = Cell{}
		return nil
	}
	var offset Offset
	if err := json.Unmarshal(raw, &offset); err != nil {
		return fmt.Errorf("decode cell: %w", err)
	}
	*c = At(offset)
	return nil
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
//
// SideNone is placed as an ally would be. There is nothing better to do with a
// side that is not one, and refusing it here would put an error in the geometry:
// whoever builds a roster is where a unit on no side is caught.
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

// ReachNeeded is the shortest range that can touch anybody at all from a
// formation column: the distance to the nearest opposing slot, taken from the
// worst row of that column so the answer holds wherever in it a unit stands.
//
// Nothing on this board moves, so a unit's reach is settled the moment it is
// placed, and this is the number that settles it. Today the answers are one from
// the front column, two from the middle and three from the back — but they are
// measured through Place rather than written down, because the mirroring is what
// produces them and a constant would go quietly wrong the day the board changed
// shape.
//
// A column outside the formation grid reports zero, which is no reach at all.
func ReachNeeded(col int) int {
	if col < 0 || col >= FormationCols {
		return 0
	}
	needed := 0
	for row := 0; row < Rows; row++ {
		mine := Place(SideAlly, Offset{Col: col, Row: row})
		nearest := 0
		for _, theirs := range SideCells(SideEnemy) {
			if distance := mine.DistanceTo(theirs); nearest == 0 || distance < nearest {
				nearest = distance
			}
		}
		if nearest > needed {
			needed = nearest
		}
	}
	return needed
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
