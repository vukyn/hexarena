package screen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// TestTheRostersElementColoursAreThePalettesOwn is the wiring, measured end to
// end: the colour a roster row is drawn in is the colour the one element table
// in this program declares, and not a second opinion about it.
//
// ⚠️ **A palette that had to be built for this cannot be checked by a golden.**
// Every golden here is recorded under NO_COLOR, where the ink is the identity, so
// the coloured drawing is measured by this test or by nothing at all — which is
// also why the first thing it does is confirm that a coloured cell carries an
// escape at all. Without that, every claim below would hold of a palette that
// had quietly stopped colouring anything.
func TestTheRostersElementColoursAreThePalettesOwn(t *testing.T) {
	c, _ := start(t, i18n.En)
	coloured := NewPalette(false)
	if coloured.Plain {
		t.Fatal("NewPalette(false) came back plain, so nothing below is measuring colour")
	}
	// The premise: a style off this palette really writes an escape under `go
	// test`, where stdout is a pipe. If it does not, every comparison below is
	// one bare code against another.
	if drawn := coloured.Element(element.Fire).Render("fir"); !strings.Contains(drawn, "\x1b[") {
		t.Fatalf("fire renders as %q with no escape in it, so the colour claims below are "+
			"vacuously true", drawn)
	}

	local := atABattleOf(t, c, 3)
	rows := strings.Split(tui.RosterWide(c.Lang, local.Fight, local.Tags,
		coloured.ElementInk()), "\n")[1:]
	if len(rows) != len(local.Fight.Units()) {
		t.Fatalf("the wide roster drew %d rows for %d units", len(rows), len(local.Fight.Units()))
	}

	drawn := 0
	for index, unit := range local.Fight.Units() {
		for _, member := range unit.Affinity.Elements() {
			drawn++
			// The palette's own answer for this element, over the code the
			// renderer draws. Built from Element rather than from a colour
			// written down here: what is being held is that the roster and the
			// chart screen ask the same table, not that the table says red.
			want := coloured.Element(member).Render(tui.ElementCode(member))
			if !strings.Contains(rows[index], want) {
				t.Errorf("%s's row is not drawing %s in the palette's own ink (%q):\n%q",
					unit.ID, member, want, rows[index])
			}
		}
	}
	// The premise: some element was actually drawn. A bench that fielded nothing
	// would have satisfied the loop above without asking the palette once.
	if drawn == 0 {
		t.Fatal("no unit on the bench carries an element at all, so the ink was never asked " +
			"for one")
	}

	// ⚠️ **This fixture's cast carries no dual affinity**, measured at every
	// squad size from one to five — battleCast picks by the size of a kit, and
	// the characters that come back are single-element ones. So the claim that
	// each half of a dual is inked for its *own* element cannot be made here at
	// all, and is made in internal/tui, where an affinity can be built rather
	// than fielded: TestEachHalfOfADualIsInkedForItsOwnElement.
	for _, unit := range local.Fight.Units() {
		if unit.Affinity.IsDual() {
			t.Logf("%s carries %s, so this fixture has gained a dual affinity — the note "+
				"above is out of date", unit.ID, unit.Affinity)
		}
	}
}

// TestEveryElementIsDrawnThroughThePalettesOwnTable is the same claim taken over
// the enum rather than over a bench.
//
// A cast is a fact about today's data and carries six or seven of the eleven
// elements; the ink is offered every one of them. So this asks the ink directly,
// for each declared element, and requires the exact string the palette's own
// style produces — which is what would stop agreeing the day somebody wired a
// second list of colours in.
func TestEveryElementIsDrawnThroughThePalettesOwnTable(t *testing.T) {
	coloured := NewPalette(false)
	ink := coloured.ElementInk()
	inked := 0
	for _, member := range element.All() {
		code := tui.ElementCode(member)
		if got, want := ink(member, code), coloured.Element(member).Render(code); got != want {
			t.Errorf("%s is inked as %q and the palette draws it %q", member, got, want)
		}
		if strings.Contains(ink(member, code), "\x1b[") {
			inked++
		}
	}
	// The premise: at least some of them are really coloured. Neutral is faint by
	// design and a palette that had lost its colours entirely would make every
	// equality above hold between two bare codes.
	if inked == 0 {
		t.Fatal("no element is drawn with an escape at all, so the equalities above compare " +
			"plain codes")
	}

	// And under a plain palette the ink is the identity, which is what lets every
	// golden in this package record the codes bare.
	plain := NewPalette(true).ElementInk()
	for _, member := range element.All() {
		code := tui.ElementCode(member)
		if got := plain(member, code); got != code {
			t.Errorf("%s is drawn as %q under a plain palette, want the bare code %q",
				member, got, code)
		}
	}
}

// TestOnlyOneTableOfElementColoursIsDeclared is the guard the behaviour tests
// cannot make, and it is a walk over this package's own source.
//
// ⚠️ **A second table would agree on the day it was written.** Copy
// elementColours, wire the roster to the copy, and every equality above still
// holds — until somebody decides ground should be brown and changes one of them.
// The defect is then two screens disagreeing about what an element looks like,
// which is precisely the reason the colours were put in one place with their
// reasons beside them. Nothing comparing two renderings can see it; what can is
// the declaration count.
//
// The rule is written as widely as it can be: **no package-level declaration in
// internal/screen may be indexed or keyed by an element except elementColours.**
// That catches a copied array, a map from element to colour, and a parallel list
// of styles alike, without naming any of their shapes.
func TestOnlyOneTableOfElementColoursIsDeclared(t *testing.T) {
	const allowed = "elementColours"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	found, walked := []string{}, 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		walked++
		file, parseErr := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", name, parseErr)
		}
		for _, declaration := range file.Decls {
			general, isGeneral := declaration.(*ast.GenDecl)
			if !isGeneral || (general.Tok != token.VAR && general.Tok != token.CONST) {
				continue
			}
			for _, spec := range general.Specs {
				value, isValue := spec.(*ast.ValueSpec)
				if !isValue || !typedByAnElement(value) {
					continue
				}
				for _, declared := range value.Names {
					found = append(found, name+": "+declared.Name)
					if declared.Name != allowed {
						t.Errorf("%s declares %s, a second table indexed by element — the "+
							"colours live in %s and nowhere else, so that two screens "+
							"cannot draw one element two ways",
							name, declared.Name, allowed)
					}
				}
			}
		}
	}
	// Premises, held rather than assumed: a walk that read no file, or that found
	// no element-indexed declaration at all, would have passed having measured
	// nothing.
	if walked == 0 {
		t.Fatal("no source file was walked, so this measured nothing")
	}
	if len(found) != 1 {
		t.Fatalf("the walk found %d element-indexed declarations (%v), and the one it is "+
			"written around is %s", len(found), found, allowed)
	}
}

// typedByAnElement reports whether a declaration's type is indexed or keyed by
// the element enum.
//
// ⚠️ **Both places a type can sit are read, and the second is the one that
// matters.** `var elementColours = [element.Count]string{…}` states its type on
// the *composite literal* and leaves the spec's own Type nil, so a walk reading
// only the spec found nothing at all — and reported a clean package. That was
// this test's first version, and it passed.
func typedByAnElement(value *ast.ValueSpec) bool {
	if value.Type != nil && namesAnElement(value.Type) {
		return true
	}
	for _, expr := range value.Values {
		literal, isLiteral := expr.(*ast.CompositeLit)
		if isLiteral && literal.Type != nil && namesAnElement(literal.Type) {
			return true
		}
	}
	return false
}

// namesAnElement reports whether a type expression is indexed or keyed by the
// element enum — `[element.Count]T`, `map[element.Element]T` and anything else
// that mentions the package in its type.
//
// Written over the identifiers it holds rather than over its shape, because the
// shapes a second table could take are not a list worth keeping: what they have
// in common is the word.
func namesAnElement(expr ast.Expr) bool {
	var out strings.Builder
	ast.Inspect(expr, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			out.WriteString(ident.Name + ".")
		}
		return true
	})
	return strings.Contains(out.String(), "element.")
}

// TestTheElementColumnStaysOffTheNarrowTable is the other side of the ink: the
// table a window at the floor draws has no element column, so it has no colour
// in it either, whatever palette the client is holding.
//
// ⚠️ Measured with a **coloured** palette on purpose. The narrow table is drawn
// by tui.Roster, which takes no ink at all, and the claim worth holding is that
// the route the ink travels cannot reach it — not that it happens to be plain in
// a suite that runs under NO_COLOR.
func TestTheElementColumnStaysOffTheNarrowTable(t *testing.T) {
	c, _ := start(t, i18n.En)
	coloured := NewPalette(false)
	local := atABattleOf(t, c, 3)
	narrow := tui.Roster(c.Lang, local.Fight, local.Tags)
	if strings.Contains(narrow, "\x1b[") {
		t.Errorf("the narrow roster carries an escape code:\n%q", narrow)
	}
	if got := lipgloss.Width(narrow); got == 0 {
		t.Fatal("the narrow roster drew nothing, so this measured nothing")
	}
	// And the wide one, off the same battle and the same palette, does carry one
	// — or the assertion above is about a client that has stopped colouring
	// anything at all.
	wide := tui.RosterWide(c.Lang, local.Fight, local.Tags, coloured.ElementInk())
	if !strings.Contains(wide, "\x1b[") {
		t.Fatal("the wide roster carries no escape either, so the narrow one being plain " +
			"says nothing")
	}
}
