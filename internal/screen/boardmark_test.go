package screen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestEveryBoardMarkIsTwoASCIIBytes is a requirement of the drawing that was
// stated in two comments and held by **nothing but a golden diff**.
//
// ⚠️ `hex.Render` slices a cell's label with `len` and `text[:2]`, so a mark of
// anything but two ASCII bytes is **cut in the middle of a rune** and puts broken
// UTF-8 on the board. Measured: swapping `arrangeCursorMark` to a two-*rune* mark
// reddens `internal/screen`'s golden and nothing else — and a golden is the wrong
// net for this, because the failure it reports is a byte diff a reader clears
// with `make golden`. Accepting that diff ships the mojibake.
//
// So the rule is asserted by name. The walk finds the constants rather than
// listing them, which is what makes a mark added later covered the day it is
// written: this package had four when this was put in, two screens apart, and the
// comment on each pointed at the other rather than at a test.
//
// ⚠️ **The scope is the files that CALL hex.Render, not every constant named
// Mark**, and that was measured rather than assumed: chart.go declares four —
// `beatsMark`, `backMark`, `mutualMark`, `chainMark` — which are the arrows
// between elements in the affinity chart. They are 3 to 6 bytes, they are correct,
// and they never reach a board. A walk keyed on the name alone reports all four
// and says nothing true about any of them.
//
// ⚠️ It is **two** bytes rather than "one cell", and those are different claims.
// A single-byte mark is not cut, but `text[:2]` would take the byte after it, so
// the cell borrows whatever the format string put there — a width bug rather than
// an encoding one, and just as invisible in a golden nobody reads closely.
func TestEveryBoardMarkIsTwoASCIIBytes(t *testing.T) {
	set := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read this package's own directory: %v", err)
	}
	var found, scanned int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, entry.Name(), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		source, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		if !strings.Contains(string(source), "hex.Render(") {
			continue
		}
		scanned++
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for at, name := range spec.Names {
				if !strings.HasSuffix(name.Name, "Mark") || at >= len(spec.Values) {
					continue
				}
				literal, ok := spec.Values[at].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				mark, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Errorf("%s: %s is not a plain string literal: %v", entry.Name(), name.Name, err)
					continue
				}
				found++
				if len(mark) != 2 {
					t.Errorf("%s: %s is %q, %d bytes — hex.Render slices a label with text[:2], "+
						"so anything else is cut mid-rune or borrows the byte after it",
						entry.Name(), name.Name, mark, len(mark))
				}
				if mark != "" && utf8.RuneCountInString(mark) != len(mark) {
					t.Errorf("%s: %s is %q, which is %d runes in %d bytes — a board mark has to be "+
						"ASCII, because the slice that draws it counts bytes",
						entry.Name(), name.Name, mark, utf8.RuneCountInString(mark), len(mark))
				}
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("no file in this package calls hex.Render, so this walk measured nothing — the " +
			"board drawing moved and this test has to follow it")
	}
	if found < 4 {
		t.Fatalf("the walk found %d board marks across %d board-drawing files, and this package "+
			"had four when the rule was written — they were renamed off the Mark suffix and this "+
			"test has to follow them", found, scanned)
	}
	t.Logf("scanned %d board-drawing files; all %d board marks are two ASCII bytes", scanned, found)
}
