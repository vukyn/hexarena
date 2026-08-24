package hex

import (
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
