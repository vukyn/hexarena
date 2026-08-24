// Package pattern holds the shapes an area skill covers.
//
// A pattern is a primary cell plus a list of step chains leading away from it.
// Because the battlefield's columns are fixed — an ally always faces east — the
// six hex directions have stable names, so a shape can be authored as "the
// primary and the cell above it" or "the primary and both cells deeper in"
// without the engine tracking facing.
//
// Resolution drops any cell that falls off the board or onto the other half of
// the battlefield. A shape aimed at an enemy therefore never catches an ally,
// and a shape aimed at the edge of a formation covers fewer cells than one aimed
// at its middle. That is not an edge case to smooth over: it is the reason where
// a skill is aimed matters as much as which skill it is.
package pattern

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// Direction is one of the six hex neighbours, named for the board's fixed
// orientation: columns run left to right and odd columns sit half a cell lower.
type Direction uint8

const (
	Up Direction = iota
	Down
	UpperRight
	LowerRight
	UpperLeft
	LowerLeft
)

// DirectionCount is the number of directions.
const DirectionCount = int(LowerLeft) + 1

var directionNames = [DirectionCount]string{
	Up:         "up",
	Down:       "down",
	UpperRight: "upper_right",
	LowerRight: "lower_right",
	UpperLeft:  "upper_left",
	LowerLeft:  "lower_left",
}

// cubeSteps maps each named direction onto the cube offset it moves by.
var cubeSteps = [DirectionCount]hex.Cube{
	Up:         {X: 0, Y: +1, Z: -1},
	Down:       {X: 0, Y: -1, Z: +1},
	UpperRight: {X: +1, Y: 0, Z: -1},
	LowerRight: {X: +1, Y: -1, Z: 0},
	UpperLeft:  {X: -1, Y: +1, Z: 0},
	LowerLeft:  {X: -1, Y: 0, Z: +1},
}

func (d Direction) String() string {
	if int(d) >= DirectionCount {
		return fmt.Sprintf("direction(%d)", uint8(d))
	}
	return directionNames[d]
}

// Valid reports whether the value is a declared direction.
func (d Direction) Valid() bool { return int(d) < DirectionCount }

// Step returns the cube offset the direction moves by.
func (d Direction) Step() hex.Cube {
	if !d.Valid() {
		return hex.Cube{}
	}
	return cubeSteps[d]
}

// ParseDirection resolves a direction name as written in the data files.
func ParseDirection(name string) (Direction, error) {
	for i, candidate := range directionNames {
		if candidate == name {
			return Direction(i), nil
		}
	}
	return 0, fmt.Errorf("unknown direction %q", name)
}

// Directions returns every direction in declaration order.
func Directions() []Direction {
	out := make([]Direction, 0, DirectionCount)
	for i := 0; i < DirectionCount; i++ {
		out = append(out, Direction(i))
	}
	return out
}

// Pattern is one area shape: the primary cell plus a chain of steps for every
// splash cell.
type Pattern struct {
	Name string
	// Splash holds one step chain per additional cell, each walked from the
	// primary. A chain of one step is a neighbour; a chain of two reaches past
	// it, which is how a shape can pierce into a formation rather than spread
	// across it.
	Splash [][]Direction
}

// MaxTargets is how many cells the shape covers when nothing is dropped.
func (p Pattern) MaxTargets() int { return 1 + len(p.Splash) }

// Validate rejects a shape that cannot be resolved or that covers more cells
// than the design allows.
func (p Pattern) Validate(maxTargets int) error {
	if p.Name == "" {
		return fmt.Errorf("a pattern needs a name")
	}
	if maxTargets < 1 {
		return fmt.Errorf("pattern %q: the target limit is %d, want at least 1", p.Name, maxTargets)
	}
	if p.MaxTargets() > maxTargets {
		return fmt.Errorf("pattern %q covers %d cells, over the limit of %d",
			p.Name, p.MaxTargets(), maxTargets)
	}
	for i, chain := range p.Splash {
		if len(chain) == 0 {
			return fmt.Errorf("pattern %q: splash %d has no steps, which would repeat the primary", p.Name, i)
		}
		for _, step := range chain {
			if !step.Valid() {
				return fmt.Errorf("pattern %q: splash %d uses %s, which is not a direction", p.Name, i, step)
			}
		}
	}
	return nil
}

// Targets returns the cells the shape covers when aimed at a primary, primary
// first. Cells off the board, cells on the other half of the battlefield, and
// duplicates are dropped, so the result can be shorter than MaxTargets.
func (p Pattern) Targets(primary hex.Offset) []hex.Offset {
	if !primary.OnBoard() {
		return nil
	}
	side := primary.Side()
	out := make([]hex.Offset, 0, p.MaxTargets())
	out = append(out, primary)
	seen := map[hex.Offset]bool{primary: true}
	for _, chain := range p.Splash {
		cube := primary.Cube()
		for _, step := range chain {
			cube = cube.Add(step.Step())
		}
		cell := cube.Offset()
		if !cell.OnBoard() || cell.Side() != side || seen[cell] {
			continue
		}
		seen[cell] = true
		out = append(out, cell)
	}
	return out
}

// Book is a named collection of shapes plus the design limits they obey.
type Book struct {
	// MaxTargets is the most cells any shape may cover.
	MaxTargets int
	// SplashPower is the share of a skill's power a splash cell takes, in parts
	// per thousand. The primary always takes the full amount.
	SplashPower int
	patterns    []Pattern
	byName      map[string]Pattern
}

type bookFile struct {
	MaxTargets  int `json:"max_targets"`
	SplashPower int `json:"splash_power"`
	Patterns    []struct {
		Name   string     `json:"name"`
		Splash [][]string `json:"splash"`
	} `json:"patterns"`
}

// ParseBook reads a shape declaration. It never touches the filesystem.
func ParseBook(raw []byte) (*Book, error) {
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode pattern book: %w", err)
	}
	if file.MaxTargets < 1 {
		return nil, fmt.Errorf("max_targets is %d, want at least 1", file.MaxTargets)
	}
	if file.SplashPower < 0 || file.SplashPower > 1000 {
		return nil, fmt.Errorf("splash_power is %d, want a share in parts per thousand", file.SplashPower)
	}
	book := &Book{
		MaxTargets:  file.MaxTargets,
		SplashPower: file.SplashPower,
		byName:      make(map[string]Pattern, len(file.Patterns)),
	}
	for _, declared := range file.Patterns {
		shape := Pattern{Name: declared.Name}
		for _, chain := range declared.Splash {
			steps := make([]Direction, 0, len(chain))
			for _, name := range chain {
				step, err := ParseDirection(name)
				if err != nil {
					return nil, fmt.Errorf("pattern %q: %w", declared.Name, err)
				}
				steps = append(steps, step)
			}
			shape.Splash = append(shape.Splash, steps)
		}
		if err := shape.Validate(book.MaxTargets); err != nil {
			return nil, err
		}
		if _, clash := book.byName[shape.Name]; clash {
			return nil, fmt.Errorf("pattern %q is declared twice", shape.Name)
		}
		book.byName[shape.Name] = shape
		book.patterns = append(book.patterns, shape)
	}
	if len(book.patterns) == 0 {
		return nil, fmt.Errorf("the pattern book is empty")
	}
	if _, ok := book.byName["single"]; !ok {
		return nil, fmt.Errorf(`the pattern book has no "single" shape, which every non-area skill needs`)
	}
	return book, nil
}

// Patterns returns every shape in declaration order.
func (b *Book) Patterns() []Pattern {
	out := make([]Pattern, len(b.patterns))
	copy(out, b.patterns)
	return out
}

// Lookup returns a shape by name.
func (b *Book) Lookup(name string) (Pattern, error) {
	shape, ok := b.byName[name]
	if !ok {
		return Pattern{}, fmt.Errorf("unknown pattern %q", name)
	}
	return shape, nil
}

// Power returns the share of a skill's power a cell takes: the full amount for
// the primary, the book's splash share for anything else.
func (b *Book) Power(index int) int {
	if index == 0 {
		return 1000
	}
	return b.SplashPower
}
