package pattern_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
)

func TestDirectionStepsAreTheSixHexNeighbours(t *testing.T) {
	centre := hex.Offset{Col: 4, Row: 1}
	reached := make(map[hex.Offset]bool)
	for _, direction := range pattern.Directions() {
		cell := centre.Cube().Add(direction.Step()).Offset()
		if got := centre.DistanceTo(cell); got != 1 {
			t.Errorf("%s from %s reaches %s at distance %d, want 1", direction, centre, cell, got)
		}
		if reached[cell] {
			t.Errorf("%s duplicates a cell already reached", direction)
		}
		reached[cell] = true
	}
	if len(reached) != 6 {
		t.Errorf("the six directions reached %d distinct cells", len(reached))
	}
	// Every neighbour the geometry knows about must have a name.
	for _, neighbour := range centre.Neighbors() {
		if !reached[neighbour] {
			t.Errorf("no direction names the neighbour %s", neighbour)
		}
	}
}

func TestDirectionNamesMatchTheBoardLayout(t *testing.T) {
	// The names have to hold for both column parities, since odd columns sit
	// half a cell lower and a naive offset delta would flip.
	cases := []struct {
		from      hex.Offset
		direction pattern.Direction
		want      hex.Offset
	}{
		{hex.Offset{Col: 4, Row: 1}, pattern.Up, hex.Offset{Col: 4, Row: 0}},
		{hex.Offset{Col: 4, Row: 1}, pattern.Down, hex.Offset{Col: 4, Row: 2}},
		{hex.Offset{Col: 4, Row: 1}, pattern.UpperLeft, hex.Offset{Col: 3, Row: 0}},
		{hex.Offset{Col: 4, Row: 1}, pattern.LowerLeft, hex.Offset{Col: 3, Row: 1}},
		{hex.Offset{Col: 4, Row: 1}, pattern.UpperRight, hex.Offset{Col: 5, Row: 0}},
		{hex.Offset{Col: 4, Row: 1}, pattern.LowerRight, hex.Offset{Col: 5, Row: 1}},
		{hex.Offset{Col: 3, Row: 1}, pattern.Up, hex.Offset{Col: 3, Row: 0}},
		{hex.Offset{Col: 3, Row: 1}, pattern.Down, hex.Offset{Col: 3, Row: 2}},
		{hex.Offset{Col: 3, Row: 1}, pattern.UpperRight, hex.Offset{Col: 4, Row: 1}},
		{hex.Offset{Col: 3, Row: 1}, pattern.LowerRight, hex.Offset{Col: 4, Row: 2}},
		{hex.Offset{Col: 3, Row: 1}, pattern.UpperLeft, hex.Offset{Col: 2, Row: 1}},
		{hex.Offset{Col: 3, Row: 1}, pattern.LowerLeft, hex.Offset{Col: 2, Row: 2}},
	}
	for _, testCase := range cases {
		got := testCase.from.Cube().Add(testCase.direction.Step()).Offset()
		if got != testCase.want {
			t.Errorf("%s from %s reached %s, want %s", testCase.direction, testCase.from, got, testCase.want)
		}
	}
}

func TestTargetsStayOnTheTargetsOwnSide(t *testing.T) {
	// The enemy frontline is column 3, so a shape reaching left from it would
	// otherwise land on the ally half.
	shape := pattern.Pattern{Name: "wedge_left", Splash: [][]pattern.Direction{
		{pattern.UpperLeft}, {pattern.LowerLeft},
	}}
	got := shape.Targets(hex.Offset{Col: 3, Row: 1})
	if len(got) != 1 || got[0] != (hex.Offset{Col: 3, Row: 1}) {
		t.Errorf("aiming left from the enemy frontline covered %v, want the primary alone", got)
	}
	for _, cell := range got {
		if cell.Side() != hex.SideEnemy {
			t.Errorf("%s is on the %s side", cell, cell.Side())
		}
	}
}

func TestTargetsDropsCellsOffTheBoard(t *testing.T) {
	shape := pattern.Pattern{Name: "column", Splash: [][]pattern.Direction{
		{pattern.Up}, {pattern.Down},
	}}
	// Row 0 has nothing above it, so the shape covers two cells rather than three.
	top := shape.Targets(hex.Offset{Col: 4, Row: 0})
	if len(top) != 2 {
		t.Errorf("aiming at the top row covered %v, want two cells", top)
	}
	middle := shape.Targets(hex.Offset{Col: 4, Row: 1})
	if len(middle) != 3 {
		t.Errorf("aiming at the middle row covered %v, want three cells", middle)
	}
	if middle[0] != (hex.Offset{Col: 4, Row: 1}) {
		t.Errorf("the primary is %s, want it first", middle[0])
	}
}

func TestTargetsRejectsAPrimaryOffTheBoard(t *testing.T) {
	shape := pattern.Pattern{Name: "single"}
	if got := shape.Targets(hex.Offset{Col: 9, Row: 9}); got != nil {
		t.Errorf("a primary off the board covered %v, want nothing", got)
	}
}

func TestTargetsDeduplicates(t *testing.T) {
	// Up then down returns to the primary, which must not be counted twice.
	shape := pattern.Pattern{Name: "loop", Splash: [][]pattern.Direction{
		{pattern.Up, pattern.Down},
	}}
	got := shape.Targets(hex.Offset{Col: 4, Row: 1})
	if len(got) != 1 {
		t.Errorf("a chain returning to the primary covered %v, want one cell", got)
	}
}

func TestPierceReachesPastItsNeighbour(t *testing.T) {
	shape := pattern.Pattern{Name: "pierce", Splash: [][]pattern.Direction{
		{pattern.UpperRight}, {pattern.UpperRight, pattern.UpperRight},
	}}
	got := shape.Targets(hex.Offset{Col: 3, Row: 2})
	if len(got) != 3 {
		t.Fatalf("the shape covered %v, want three cells", got)
	}
	if distance := got[0].DistanceTo(got[2]); distance != 2 {
		t.Errorf("the far cell %s is %d from the primary, want 2", got[2], distance)
	}
}

func TestPatternValidateRejects(t *testing.T) {
	cases := []struct {
		name    string
		shape   pattern.Pattern
		limit   int
		wantErr string
	}{
		{"no name", pattern.Pattern{}, 3, "needs a name"},
		{"over the target limit", pattern.Pattern{Name: "wide", Splash: [][]pattern.Direction{
			{pattern.Up}, {pattern.Down}, {pattern.UpperRight},
		}}, 3, "over the limit"},
		{"an empty chain", pattern.Pattern{Name: "empty", Splash: [][]pattern.Direction{{}}}, 3, "no steps"},
		{"an undeclared direction", pattern.Pattern{Name: "bad", Splash: [][]pattern.Direction{
			{pattern.Direction(9)},
		}}, 3, "not a direction"},
		{"a limit below one", pattern.Pattern{Name: "single"}, 0, "at least 1"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.shape.Validate(testCase.limit)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestParseBookRejects(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"malformed json", "{", "decode pattern book"},
		{"no target limit", `{"max_targets":0,"splash_power":500,"patterns":[{"name":"single","splash":[]}]}`, "max_targets"},
		{"a splash share above a whole", `{"max_targets":3,"splash_power":1500,"patterns":[{"name":"single","splash":[]}]}`, "splash_power"},
		{"no patterns", `{"max_targets":3,"splash_power":500,"patterns":[]}`, "empty"},
		{"no single shape", `{"max_targets":3,"splash_power":500,"patterns":[{"name":"wide","splash":[["up"]]}]}`, `no "single" shape`},
		{"an unknown direction", `{"max_targets":3,"splash_power":500,"patterns":[{"name":"single","splash":[]},{"name":"odd","splash":[["sideways"]]}]}`, "unknown direction"},
		{"a duplicate name", `{"max_targets":3,"splash_power":500,"patterns":[{"name":"single","splash":[]},{"name":"single","splash":[["up"]]}]}`, "declared twice"},
		{"a shape over the limit", `{"max_targets":2,"splash_power":500,"patterns":[{"name":"single","splash":[]},{"name":"wide","splash":[["up"],["down"]]}]}`, "over the limit"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pattern.ParseBook([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestBookLookupAndPower(t *testing.T) {
	book, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3,
	  "splash_power": 500,
	  "patterns": [
	    {"name": "single", "splash": []},
	    {"name": "column", "splash": [["up"], ["down"]]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, err := book.Lookup("column"); err != nil || got.MaxTargets() != 3 {
		t.Errorf("Lookup(column) gave %v, %v", got, err)
	}
	if _, err := book.Lookup("nowhere"); err == nil {
		t.Error("an unknown name was accepted")
	}
	if got, want := book.Power(0), 1000; got != want {
		t.Errorf("the primary takes %d, want the full %d", got, want)
	}
	if got, want := book.Power(1), book.SplashPower; got != want {
		t.Errorf("a splash cell takes %d, want %d", got, want)
	}
	if got := len(book.Patterns()); got != 2 {
		t.Errorf("the book holds %d shapes, want 2", got)
	}
	// Patterns returns a copy, so a caller cannot rewrite the book.
	book.Patterns()[0] = pattern.Pattern{Name: "tampered"}
	if got := book.Patterns()[0].Name; got != "single" {
		t.Errorf("the book was modified through its own accessor, first shape is now %q", got)
	}
}

// TestTargetsAcrossKeepsWhatTheMidlineDrops is the one difference between the two
// walks, and that it is the only one.
//
// The midline is the right bound for every skill aimed at one side, so the plain
// walk keeps it. A skill aimed at both halves is the case where dropping a cell
// for being on the wrong side is wrong, and this is what says the two agree on
// everything else — the board's edge, duplicate cells, and the primary coming
// first.
func TestTargetsAcrossKeepsWhatTheMidlineDrops(t *testing.T) {
	wedge := pattern.Pattern{Name: "wedge_left", Splash: [][]pattern.Direction{
		{pattern.UpperLeft}, {pattern.LowerLeft},
	}}
	// The enemy frontline: both of the shape's cells sit one column back, which
	// is the caster's own half.
	frontline := hex.Offset{Col: 3, Row: 1}
	stopped := wedge.Targets(frontline)
	crossed := wedge.TargetsAcross(frontline)
	if len(stopped) != 1 {
		t.Errorf("the midline let %v through from %v, want the primary alone",
			stopped, frontline)
	}
	if len(crossed) != wedge.MaxTargets() {
		t.Errorf("crossing the midline caught %v from %v, want all %d cells",
			crossed, frontline, wedge.MaxTargets())
	}
	if crossed[0] != frontline {
		t.Errorf("the crossing walk returned %v first, want the primary", crossed[0])
	}
	for _, cell := range crossed[1:] {
		if cell.Side() == frontline.Side() {
			t.Errorf("%v is on the primary's own side, so this measures nothing", cell)
		}
	}

	// Everything else the walk drops, it still drops. The board's edge is not a
	// side.
	edge := hex.Offset{Col: 0, Row: 1}
	if got := wedge.TargetsAcross(edge); len(got) != 1 {
		t.Errorf("crossing the midline also crossed the board's edge: %v", got)
	}
	if got := wedge.TargetsAcross(hex.Offset{Col: -1, Row: 0}); got != nil {
		t.Errorf("a primary off the board caught %v, want nothing", got)
	}
	// A shape whose chains land on one cell twice still reports it once.
	twice := pattern.Pattern{Name: "twice", Splash: [][]pattern.Direction{
		{pattern.UpperLeft}, {pattern.UpperLeft},
	}}
	if got := twice.TargetsAcross(frontline); len(got) != 2 {
		t.Errorf("a repeated cell was counted twice: %v", got)
	}

	// And for a shape that never leaves its own half, the two walks agree
	// exactly, which is what keeps every shipped skill where it was.
	column := pattern.Pattern{Name: "column", Splash: [][]pattern.Direction{
		{pattern.Up}, {pattern.Down},
	}}
	for _, cell := range hex.Cells() {
		if got, want := column.TargetsAcross(cell), column.Targets(cell); !equalOffsets(got, want) {
			t.Errorf("from %v the two walks give %v and %v", cell, got, want)
		}
	}
}

func equalOffsets(got, want []hex.Offset) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
