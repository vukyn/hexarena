package hex

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// offsetNeighborRule is the odd-q neighbour table written out by hand from the
// board diagram. Neighbors derives the same set by going through cube
// coordinates; the two must agree or one of them is wrong.
var offsetNeighborRule = map[bool][6][2]int{
	// even column
	false: {{+1, -1}, {+1, 0}, {0, -1}, {0, +1}, {-1, -1}, {-1, 0}},
	// odd column, pushed half a cell down
	true: {{+1, 0}, {+1, +1}, {0, -1}, {0, +1}, {-1, 0}, {-1, +1}},
}

func TestOffsetCubeRoundTrip(t *testing.T) {
	for col := -4; col < Cols+4; col++ {
		for row := -4; row < Rows+4; row++ {
			offset := Offset{Col: col, Row: row}
			cube := offset.Cube()
			if sum := cube.X + cube.Y + cube.Z; sum != 0 {
				t.Errorf("%s -> %s: axes sum to %d, want 0", offset, cube, sum)
			}
			if got := cube.Offset(); got != offset {
				t.Errorf("%s -> %s -> %s: round trip lost the coordinate", offset, cube, got)
			}
		}
	}
}

func TestNeighborsMatchOffsetRule(t *testing.T) {
	for _, cell := range Cells() {
		want := make(map[Offset]bool, 6)
		for _, delta := range offsetNeighborRule[cell.Col&1 == 1] {
			want[Offset{Col: cell.Col + delta[0], Row: cell.Row + delta[1]}] = true
		}
		got := make(map[Offset]bool, 6)
		for _, neighbor := range cell.Neighbors() {
			got[neighbor] = true
			if d := cell.DistanceTo(neighbor); d != 1 {
				t.Errorf("%s -> %s: distance %d, want 1", cell, neighbor, d)
			}
		}
		if len(got) != 6 {
			t.Errorf("%s: got %d distinct neighbours, want 6", cell, len(got))
		}
		for neighbor := range want {
			if !got[neighbor] {
				t.Errorf("%s: hand-written rule expects neighbour %s, Neighbors did not return it", cell, neighbor)
			}
		}
		for neighbor := range got {
			if !want[neighbor] {
				t.Errorf("%s: Neighbors returned %s, hand-written rule does not", cell, neighbor)
			}
		}
	}
}

func TestDistanceIsSymmetricAndBounded(t *testing.T) {
	longest := 0
	for _, a := range Cells() {
		if d := a.DistanceTo(a); d != 0 {
			t.Errorf("%s to itself: distance %d, want 0", a, d)
		}
		for _, b := range Cells() {
			forward, backward := a.DistanceTo(b), b.DistanceTo(a)
			if forward != backward {
				t.Errorf("%s <-> %s: %d one way, %d the other", a, b, forward, backward)
			}
			longest = max(longest, forward)
		}
	}
	if longest != 5 {
		t.Errorf("longest distance on the board is %d, want 5 (backline to opposing backline)", longest)
	}
}

// TestRangeLadder freezes the distance tables the formation design was chosen
// from. A change to the coordinate system or the distance metric has to fail
// here rather than quietly re-tune every skill's range.
func TestRangeLadder(t *testing.T) {
	cases := []struct {
		name   string
		origin Offset
		// want is indexed [enemy column - EnemyFrontCol][enemy row].
		want [3][3]int
	}{
		{"ally frontline, top row", Offset{2, 0}, [3][3]int{
			{1, 2, 3},
			{2, 2, 3},
			{3, 3, 4},
		}},
		{"ally frontline, centre row", Offset{2, 1}, [3][3]int{
			{1, 1, 2},
			{2, 2, 2},
			{3, 3, 3},
		}},
		{"ally frontline, bottom row", Offset{2, 2}, [3][3]int{
			{2, 1, 1},
			{3, 2, 2},
			{3, 3, 3},
		}},
		{"ally backline, centre row", Offset{0, 1}, [3][3]int{
			{3, 3, 3},
			{4, 4, 4},
			{5, 5, 5},
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			for columnIndex := range testCase.want {
				for row, want := range testCase.want[columnIndex] {
					target := Offset{Col: EnemyFrontCol + columnIndex, Row: row}
					if got := testCase.origin.DistanceTo(target); got != want {
						t.Errorf("%s -> %s: distance %d, want %d", testCase.origin, target, got, want)
					}
				}
			}
		})
	}
}

// TestPlaceMirrorsBothSides is the balance invariant behind the 180 degree
// rotation: a slot authored by the ally team and the same slot authored by the
// enemy team must see an identical distance profile of the opposing formation.
func TestPlaceMirrorsBothSides(t *testing.T) {
	authored := make([]Offset, 0, FormationCols*Rows)
	for col := 0; col < FormationCols; col++ {
		for row := 0; row < Rows; row++ {
			authored = append(authored, Offset{Col: col, Row: row})
		}
	}
	for _, origin := range authored {
		allyProfile := make([]int, 0, len(authored))
		enemyProfile := make([]int, 0, len(authored))
		for _, target := range authored {
			allyProfile = append(allyProfile,
				Place(SideAlly, origin).DistanceTo(Place(SideEnemy, target)))
			enemyProfile = append(enemyProfile,
				Place(SideEnemy, origin).DistanceTo(Place(SideAlly, target)))
		}
		for i := range allyProfile {
			if allyProfile[i] != enemyProfile[i] {
				t.Errorf("slot %s vs opposing slot %s: ally sees %d, enemy sees %d",
					origin, authored[i], allyProfile[i], enemyProfile[i])
			}
		}
	}
}

func TestPlaceLandsOnTheRightHalf(t *testing.T) {
	for col := 0; col < FormationCols; col++ {
		for row := 0; row < Rows; row++ {
			author := Offset{Col: col, Row: row}
			ally, enemy := Place(SideAlly, author), Place(SideEnemy, author)
			if !ally.OnBoard() || ally.Side() != SideAlly {
				t.Errorf("ally slot %s placed at %s, off board or wrong side", author, ally)
			}
			if !enemy.OnBoard() || enemy.Side() != SideEnemy {
				t.Errorf("enemy slot %s placed at %s, off board or wrong side", author, enemy)
			}
		}
	}
	// Column 2 is each team's own frontline, so the two frontlines must end up
	// adjacent columns rather than overlapping or leaving a gap.
	if got := Place(SideAlly, Offset{2, 1}).Col; got != AllyFrontCol {
		t.Errorf("ally frontline column is %d, want %d", got, AllyFrontCol)
	}
	if got := Place(SideEnemy, Offset{2, 1}).Col; got != EnemyFrontCol {
		t.Errorf("enemy frontline column is %d, want %d", got, EnemyFrontCol)
	}
}

// TestNoSideIsTheZeroValue is what keeps a battle log honest about who won.
//
// A side is written with `omitempty`, so whichever side is zero never reaches
// the wire and a reader recovers it from the absence of a field. With a real
// side at zero that is right by coincidence — until a constant is reordered, or
// until a battle with no winner writes the first side declared. So the zero
// value is nobody, and this is the test that says so.
func TestNoSideIsTheZeroValue(t *testing.T) {
	var unset Side
	if unset != SideNone {
		t.Fatalf("the zero value is %s, want %s", unset, SideNone)
	}
	if SideNone.Fights() {
		t.Error("no side reports that it fights")
	}
	for _, side := range []Side{SideAlly, SideEnemy} {
		if !side.Fights() {
			t.Errorf("%s reports that it does not fight", side)
		}
	}
	// Every cell on the board belongs to one of the two that fight, so nothing
	// a coordinate reports can be the zero value.
	for _, cell := range Cells() {
		if !cell.Side().Fights() {
			t.Errorf("cell %s is on %s", cell, cell.Side())
		}
	}
	// Encoding is what this is all for: an unset side has to leave the field out
	// entirely, and a real one has to be written.
	type carrier struct {
		Side Side `json:"side,omitempty"`
	}
	for _, one := range []struct {
		side    Side
		written bool
	}{{SideNone, false}, {SideAlly, true}, {SideEnemy, true}} {
		raw, err := json.Marshal(carrier{Side: one.side})
		if err != nil {
			t.Fatalf("encode %s: %v", one.side, err)
		}
		if strings.Contains(string(raw), "side") != one.written {
			t.Errorf("%s encodes as %s", one.side, raw)
		}
		var back carrier
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("decode %s: %v", one.side, err)
		}
		if back.Side != one.side {
			t.Errorf("%s came back as %s", one.side, back.Side)
		}
	}
	if err := json.Unmarshal([]byte(`{"side":"neither"}`), &carrier{}); err == nil {
		t.Error("an unknown side was decoded")
	}
}

// TestTheBackCornerIsACellLikeAnyOther is the case the old event field got
// wrong, stated on its own.
//
// `{0,0}` is where an ally back-row unit stands, so a cell there has to survive
// every tag and every round trip that an absent cell is dropped by. This is the
// half a sentinel or an `omitzero` on the coordinate itself would quietly lose.
func TestTheBackCornerIsACellLikeAnyOther(t *testing.T) {
	corner := At(Offset{Col: AllyBackCol, Row: 0})
	raw, err := json.Marshal(corner)
	if err != nil {
		t.Fatalf("encode the back corner: %v", err)
	}
	if got, want := string(raw), `{"col":0,"row":0}`; got != want {
		t.Errorf("the back corner encodes as %s, want %s", got, want)
	}
	type carrier struct {
		Cell Cell `json:"cell,omitzero"`
	}
	held, err := json.Marshal(carrier{Cell: corner})
	if err != nil {
		t.Fatalf("encode a carried back corner: %v", err)
	}
	if !strings.Contains(string(held), `"cell"`) {
		t.Errorf("the back corner was omitted from %s", held)
	}
	empty, err := json.Marshal(carrier{})
	if err != nil {
		t.Fatalf("encode a carried absence: %v", err)
	}
	if strings.Contains(string(empty), "cell") {
		t.Errorf("no cell at all encodes as %s, want the field left out", empty)
	}
}

// TestAnAbsentCellIsTheZeroValue is the other half: absence is what a Cell
// carries when nothing was written, so an omitted field reads back as one.
func TestAnAbsentCellIsTheZeroValue(t *testing.T) {
	var unset Cell
	if offset, filled := unset.Offset(); filled {
		t.Errorf("the zero value reports the cell %s", offset)
	}
	if got := unset.String(); got != "none" {
		t.Errorf("the zero value prints %q, want %q", got, "none")
	}
	// Comparability is the contract --verify rests on: a re-run's events are
	// compared with == against the ones the log holds, so two cells at the same
	// place have to be the same value and an absence must never equal a place.
	somewhere := Offset{Col: 3, Row: 1}
	if At(somewhere) != At(somewhere) {
		t.Error("two cells at one coordinate do not compare equal")
	}
	if At(somewhere) == At(Offset{Col: 3, Row: 2}) {
		t.Error("two cells at different coordinates compare equal")
	}
	if unset == At(Offset{}) {
		t.Error("no cell at all compares equal to the back corner")
	}
	// Round trips, both ways round, through a field that drops the absence.
	type carrier struct {
		Cell Cell `json:"cell,omitzero"`
	}
	for _, one := range []Cell{unset, At(somewhere), At(Offset{})} {
		raw, err := json.Marshal(carrier{Cell: one})
		if err != nil {
			t.Fatalf("encode %s: %v", one, err)
		}
		var back carrier
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
		if back.Cell != one {
			t.Errorf("%s came back as %s, through %s", one, back.Cell, raw)
		}
	}
	// An explicit null is the same absence, so a writer that spells it out and a
	// writer that leaves the field off agree.
	var spelled carrier
	if err := json.Unmarshal([]byte(`{"cell":null}`), &spelled); err != nil {
		t.Fatalf("decode an explicit null: %v", err)
	}
	if spelled.Cell != unset {
		t.Errorf("an explicit null decoded as %s", spelled.Cell)
	}
	if err := json.Unmarshal([]byte(`{"cell":"3,1"}`), &carrier{}); err == nil {
		t.Error("a cell that is not a coordinate was decoded")
	}
}

// TestReachNeededRisesTowardsTheBackColumn is the geometry behind a deadlock,
// stated as the three numbers an author has to size a kit against.
//
// It is asserted against measured distances rather than against the literals
// one, two and three, so the check stays true of whatever board Place produces
// rather than of the board it happens to produce today. The literals are
// asserted separately, because the current answers are worth noticing if they
// ever move.
func TestReachNeededRisesTowardsTheBackColumn(t *testing.T) {
	for col := 0; col < FormationCols; col++ {
		needed := ReachNeeded(col)
		for row := 0; row < Rows; row++ {
			mine := Place(SideAlly, Offset{Col: col, Row: row})
			within := 0
			for _, theirs := range SideCells(SideEnemy) {
				if mine.DistanceTo(theirs) <= needed {
					within++
				}
			}
			if within == 0 {
				t.Errorf("a range of %d from %s reaches nothing, but that is what column %d needs",
					needed, mine, col)
			}
			if needed <= 1 {
				continue
			}
			// One short must reach nobody, or the number is not the shortest
			// range that works and a warning built on it would be wrong.
			for _, theirs := range SideCells(SideEnemy) {
				if mine.DistanceTo(theirs) <= needed-1 {
					t.Errorf("a range of %d already reaches %s from %s, so column %d does not need %d",
						needed-1, theirs, mine, col, needed)
				}
			}
		}
	}
	// The front column is next to the enemy's, and every column behind it is one
	// further out. Worth pinning: these are the numbers a kit is sized against.
	for col, want := range map[int]int{2: 1, 1: 2, 0: 3} {
		if got := ReachNeeded(col); got != want {
			t.Errorf("column %d needs a range of %d, want %d", col, got, want)
		}
	}
	for _, col := range []int{-1, FormationCols, Cols} {
		if got := ReachNeeded(col); got != 0 {
			t.Errorf("column %d is not a formation column but needs %d", col, got)
		}
	}
}

func TestRingAndDiskSizes(t *testing.T) {
	center := Offset{2, 1}.Cube()
	for radius := 0; radius <= 4; radius++ {
		wantRing := 6 * radius
		if radius == 0 {
			wantRing = 1
		}
		if got := len(Ring(center, radius)); got != wantRing {
			t.Errorf("Ring radius %d: %d cells, want %d", radius, got, wantRing)
		}
		for _, cell := range Ring(center, radius) {
			if d := Distance(center, cell); d != radius {
				t.Errorf("Ring radius %d returned %s at distance %d", radius, cell, d)
			}
		}
		wantDisk := 1 + 3*radius*(radius+1)
		if got := len(Disk(center, radius)); got != wantDisk {
			t.Errorf("Disk radius %d: %d cells, want %d", radius, got, wantDisk)
		}
		seen := make(map[Cube]bool)
		for _, cell := range Disk(center, radius) {
			if seen[cell] {
				t.Errorf("Disk radius %d returned %s twice", radius, cell)
			}
			seen[cell] = true
		}
	}
	if got := Ring(center, -1); got != nil {
		t.Errorf("Ring with a negative radius returned %v, want nil", got)
	}
}

func TestInRangeStaysOnBoardAndExcludesOrigin(t *testing.T) {
	for _, origin := range Cells() {
		for radius := 1; radius <= 5; radius++ {
			for _, cell := range InRange(origin, radius) {
				if !cell.OnBoard() {
					t.Errorf("InRange(%s, %d) returned off-board cell %s", origin, radius, cell)
				}
				if cell == origin {
					t.Errorf("InRange(%s, %d) included the origin", origin, radius)
				}
				if d := origin.DistanceTo(cell); d > radius {
					t.Errorf("InRange(%s, %d) returned %s at distance %d", origin, radius, cell, d)
				}
			}
		}
		// Radius 5 reaches every other cell on the board.
		if got, want := len(InRange(origin, 5)), Cols*Rows-1; got != want {
			t.Errorf("InRange(%s, 5): %d cells, want %d", origin, got, want)
		}
	}
}

func TestBoardGolden(t *testing.T) {
	got := boardReport()
	path := filepath.Join("testdata", "board.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/core/hex -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("board report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// boardReport renders every derived property of the board into one text
// snapshot: the layout, the coordinate conversions, the neighbour sets and the
// full distance matrix. Any change to the geometry shows up as a diff.
func boardReport() string {
	var b strings.Builder

	b.WriteString("== layout ==\n")
	b.WriteString(" c0  c1  c2  c3  c4  c5      c0-c2 = ally  (c2 = frontline)\n")
	b.WriteString(" BK  MD  FR  FR  MD  BK      c3-c5 = enemy (c3 = frontline)\n\n")
	b.WriteString(Render(func(cell Offset) string {
		return fmt.Sprintf("%d%d", cell.Col, cell.Row)
	}))
	b.WriteString("\n\n")

	b.WriteString("== offset -> cube ==\n")
	for _, cell := range Cells() {
		fmt.Fprintf(&b, "%s  %s  %s\n", cell, cell.Cube(), cell.Side())
	}

	b.WriteString("\n== formation slot -> board cell ==\n")
	for col := 0; col < FormationCols; col++ {
		for row := 0; row < Rows; row++ {
			author := Offset{Col: col, Row: row}
			fmt.Fprintf(&b, "%s  ally %s  enemy %s\n",
				author, Place(SideAlly, author), Place(SideEnemy, author))
		}
	}

	b.WriteString("\n== neighbours on board ==\n")
	for _, cell := range Cells() {
		parts := make([]string, 0, 6)
		for _, neighbor := range cell.NeighborsOnBoard() {
			parts = append(parts, neighbor.String())
		}
		fmt.Fprintf(&b, "%s  %s\n", cell, strings.Join(parts, " "))
	}

	b.WriteString("\n== distance matrix ==\n")
	b.WriteString("     ")
	for _, cell := range Cells() {
		fmt.Fprintf(&b, "%4s", cell)
	}
	b.WriteString("\n")
	for _, from := range Cells() {
		fmt.Fprintf(&b, "%4s ", from)
		for _, to := range Cells() {
			fmt.Fprintf(&b, "%4d", from.DistanceTo(to))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n== range needed, ally cell -> enemy cell ==\n")
	b.WriteString("          ")
	for _, target := range SideCells(SideEnemy) {
		fmt.Fprintf(&b, "%4s", target)
	}
	b.WriteString("\n")
	for _, origin := range SideCells(SideAlly) {
		fmt.Fprintf(&b, "%9s ", origin)
		for _, target := range SideCells(SideEnemy) {
			fmt.Fprintf(&b, "%4d", origin.DistanceTo(target))
		}
		b.WriteString("\n")
	}

	return b.String()
}
