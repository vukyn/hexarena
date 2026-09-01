package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The shape diagram: what it draws. Which key opens it and which keys close it
// are driven through the real model in cmd/hexforge-tui, where a keystroke has a
// client to reach.

// TestTheShapeChooserDrawsWhatTheShapeCatches is item three of the form's
// legibility: "< pierce >" names a step chain and says nothing about which cells
// it covers, so the chooser draws them.
//
// What is asserted is the composition rather than the picture: the expectation
// here walks pattern.Targets itself, from the pattern book, and the drawn board
// has to agree cell for cell. So a diagram that marked a plausible-looking set
// of cells that a battle would not actually hit fails, which is the only failure
// worth catching — a shape chooser that lies is worse than one that says nothing.
func TestTheShapeChooserDrawsWhatTheShapeCatches(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	primary := forge.ShapeDiagramCell()
	for _, name := range lib.PatternNames() {
		shape, err := lib.Patterns().Lookup(name)
		if err != nil {
			t.Fatalf("look up %s: %v", name, err)
		}
		caught := shape.Targets(primary)
		if len(caught) == 0 || caught[0] != primary {
			t.Fatalf("%s catches %v from %v, want the primary first", name, caught, primary)
		}
		want := hex.Render(func(cell hex.Offset) string {
			for position, target := range caught {
				if target != cell {
					continue
				}
				if position == 0 {
					return shapeAimMark
				}
				return shapeSplashMark
			}
			return ""
		})

		coverage, err := lib.ShapeCoverage(name, defaultSkillTarget)
		if err != nil {
			t.Fatalf("coverage of %s: %v", name, err)
		}
		if coverage.Covered() != len(caught) {
			t.Errorf("%s reports %d cells covered against Targets' %d",
				name, coverage.Covered(), len(caught))
		}
		if got := shapeBoard(coverage); got != want {
			t.Errorf("the board drawn for %s is not the cells Targets returns:\n%s\nwant:\n%s",
				name, got, want)
		}
		// The two marks are different characters, not two colours: the tests run
		// with NO_COLOR set, so what is counted here is what a monochrome
		// terminal shows.
		board := shapeBoard(coverage)
		if got := strings.Count(board, shapeAimMark); got != 1 {
			t.Errorf("%s marks the aim %d times, want once:\n%s", name, got, board)
		}
		if got, want := strings.Count(board, shapeSplashMark), len(caught)-1; got != want {
			t.Errorf("%s marks %d splash cells over %d caught:\n%s", name, got, want, board)
		}
		if shapeAimMark == shapeSplashMark {
			t.Error("the aim and a splash cell are the same mark, so the board needs colour")
		}

		// And the drawn screen really holds that board, rather than the board
		// being a function nothing calls.
		drawn := NewSkillsScreen(c)
		drawn.Adding = true
		drawn.Field = SkillFieldShape
		drawn.ShapeIndex = IndexOf(lib.PatternNames(), name)
		drawn.ShapeDrawn = true
		body, _ := drawn.View(c)
		for _, line := range strings.Split(want, "\n") {
			if !strings.Contains(body, line) {
				t.Errorf("the %s diagram does not draw %q:\n%s", name, line, body)
			}
		}
	}
}
