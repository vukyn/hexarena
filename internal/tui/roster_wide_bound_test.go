package tui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// goldenWindows are the two window widths every screen is recorded at.
//
// They are restated here rather than imported because internal/screen imports
// this package and not the other way round — the same reason rowBudget is. The
// threshold has to land strictly between them: at the floor the narrow table is
// what ships, and at the roomy one the three columns appear, which is what makes
// the golden diff of this change readable.
const (
	goldenFloorWindow = 120
	goldenRoomyWindow = 160
)

// TestTheWideRosterThresholdIsComputedFromItsFormat is the guard the owner asked
// for, and it is a walk over the source because no arithmetic test can be one.
//
// ⚠️ **Two numbers that agree today cannot tell a derivation from a constant.**
// A test asserting `rosterWideWidth == <the ceiling measured some other way>`
// passes against `var rosterWideWidth = 139`, because both sides are 139 until
// somebody widens a column — and by then the damage is a threshold that has
// stopped following the table it is about. The only thing that can see the
// difference the day it is written is the declaration itself, so this reads it.
//
// What it requires is narrow on purpose: the value comes from a call to
// rosterWidth whose first argument is rosterWideRow. That is the whole claim —
// the threshold is a function of the format string — and it fails on a literal,
// on a copy of the arithmetic written out by hand, and on a call handed some
// other format.
func TestTheWideRosterThresholdIsComputedFromItsFormat(t *testing.T) {
	const declared = "rosterWideWidth"
	file, err := parser.ParseFile(token.NewFileSet(), "tui.go", nil, 0)
	if err != nil {
		t.Fatalf("parse tui.go: %v", err)
	}
	var value ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for index, name := range spec.Names {
			if name.Name == declared && index < len(spec.Values) {
				value = spec.Values[index]
			}
		}
		return true
	})
	if value == nil {
		t.Fatalf("%s is not declared with a value in tui.go, so this measured nothing", declared)
	}
	call, ok := value.(*ast.CallExpr)
	if !ok {
		t.Fatalf("%s is assigned a %T rather than a call — the threshold has to be computed "+
			"from the row's format, not written down beside it", declared, value)
	}
	callee, ok := call.Fun.(*ast.Ident)
	if !ok || callee.Name != "rosterWidth" {
		t.Fatalf("%s is assigned a call to something other than rosterWidth", declared)
	}
	if len(call.Args) == 0 {
		t.Fatalf("%s calls rosterWidth with no format at all", declared)
	}
	format, ok := call.Args[0].(*ast.Ident)
	if !ok || format.Name != "rosterWideRow" {
		t.Errorf("%s is measured off %s rather than off rosterWideRow", declared,
			formatted(call.Args[0]))
	}
}

// formatted is an expression as it reads in the source, for a message that has
// to name what it found rather than its type.
func formatted(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if ok {
		return ident.Name
	}
	return fmt.Sprintf("%T", expr)
}

// TestRosterWidthReadsTheFormatItIsHanded is the other half of the pair above:
// the walk says the threshold goes through this function, and this says the
// function is really arithmetic over its argument rather than a number in a
// wrapper.
//
// Three synthetic layouts rather than one, with answers that are obvious by
// inspection, because a helper that ignored its format and returned a constant
// would satisfy any single case.
func TestRosterWidthReadsTheFormatItIsHanded(t *testing.T) {
	cases := []struct {
		format string
		fields []any
		want   int
	}{
		// Five cells, a separator, three cells, and the cell every line leaves
		// empty.
		{"%-5s|%-3s\n", []any{"", ""}, 10},
		// The same layout one cell wider in its first column: the answer has to
		// move with it, which is the property the threshold rests on.
		{"%-6s|%-3s\n", []any{"", ""}, 11},
		// A numeric verb pads the same way, and a format with no newline to trim
		// is measured whole.
		{"%4d", []any{0}, 5},
	}
	for _, one := range cases {
		if got := rosterWidth(one.format, one.fields...); got != one.want {
			t.Errorf("rosterWidth(%q) is %d, want %d", one.format, got, one.want)
		}
	}
}

// TestTheWideRosterThresholdIsOneMoreThanTheWidestRowItCanDraw measures the same
// ceiling a second way, and the second way is a row with real content in it.
//
// The declaration pads empty strings out to their columns, which is the cheapest
// measurement and the one a reader has to take on trust. This one fills every
// column with the widest thing that column is allowed to hold — a name at
// rosterNameRoom, an affinity at rosterElementRoom, four-digit figures, a full
// effects column — so it would catch a column whose padding and whose bound
// disagree, which the empty-string measurement cannot see.
//
// ⚠️ **Its premise is that the effects column really fills**, held rather than
// assumed, exactly as the narrow bound test holds it: a ceiling the elision never
// reaches is a ceiling no row can stand at.
func TestTheWideRosterThresholdIsOneMoreThanTheWidestRowItCanDraw(t *testing.T) {
	if got := len(elided([]string{strings.Repeat("x", effectsRoom)}, effectsRoom)); got != effectsRoom {
		t.Fatalf("the effects column tops out at %d of the %d it has, so the ceiling below "+
			"is not a row that can exist", got, effectsRoom)
	}
	const widestFigure = 1234
	if digits := len(strconv.Itoa(widestFigure)); digits != rosterStatRoom {
		t.Fatalf("the fixture figure is %d digits against a column of %d, so it does not "+
			"fill the column it is measuring", digits, rosterStatRoom)
	}
	drawn := fmt.Sprintf(strings.TrimSuffix(rosterWideRow, "\n"),
		unitColumn("A1", strings.Repeat("n", rosterNameRoom)),
		HealthBar(4800, 4800),
		widestFigure,
		strings.Repeat("e", rosterElementRoom),
		widestFigure, widestFigure,
		strings.Repeat("x", effectsRoom))
	widest := utf8.RuneCountInString(drawn)
	if got, want := rosterWideWidth, widest+1; got != want {
		t.Errorf("the wide table says it needs a window of %d and the widest row it can draw "+
			"is %d cells, which needs %d:\n%s", got, widest, want, drawn)
	}
	t.Logf("the wide row's ceiling is %d cells and its threshold is a window of %d",
		widest, rosterWideWidth)
}

// TestTheWideRosterThresholdLandsBetweenTheTwoGoldenWindows is the landing
// check, and it is the weakest of these three on purpose — it says where the
// derived number falls rather than that it was derived.
//
// Both directions, because a threshold under the floor would make the wide table
// what ships and a threshold over the roomy window would make it something no
// recorded screen ever draws — and either of those is a change nothing else here
// would report as a change.
func TestTheWideRosterThresholdLandsBetweenTheTwoGoldenWindows(t *testing.T) {
	if RosterIsWide(goldenFloorWindow) {
		t.Errorf("a window at the floor of %d takes the wide table, so what ships at the "+
			"floor is not what shipped before", goldenFloorWindow)
	}
	if !RosterIsWide(goldenRoomyWindow) {
		t.Errorf("a window of %d does not take the wide table, so no recorded screen draws it",
			goldenRoomyWindow)
	}
	// And the boundary itself, both sides of it, so the comparison cannot quietly
	// become a strict one.
	if RosterIsWide(rosterWideWidth - 1) {
		t.Errorf("a window of %d takes a table that needs %d", rosterWideWidth-1, rosterWideWidth)
	}
	if !RosterIsWide(rosterWideWidth) {
		t.Errorf("a window of exactly %d refuses the table it has room for", rosterWideWidth)
	}
}

// TestEveryAffinityTheRosterCanDrawFitsItsColumn is the element column's bound,
// and it is taken over the element book rather than over the cast.
//
// ⚠️ **An observed maximum is not a bound.** The widest affinity anything ships
// today is `electric/metal`, and a column sized to that would be a fact about the
// current cast — a fixture library or a later character pairing two longer names
// would push the stat columns right on one row, which is a table that misaligns
// quietly. What bounds the column is what element.Affinity can hold at all: one
// element, or two distinct non-inert ones joined by a slash.
//
// ⚠️ **What is measured is the CELL, not the affinity's name.** The column draws
// codes now, so a test asserting `Affinity.String()` fits would be measuring a
// string the table stopped putting there — it would have gone red on a column
// that was correct, and would say nothing at all about one that was not.
// elementCell is what the row is handed, so elementCell is what this walks.
func TestEveryAffinityTheRosterCanDrawFitsItsColumn(t *testing.T) {
	// The column's own width, read back out of the format string so the format
	// and the bound below it cannot part company.
	start := strings.Index(fmt.Sprintf(strings.TrimSuffix(rosterWideRow, "\n"),
		"", "", 0, "|", 0, 0, ""), "|")
	next := strings.Index(fmt.Sprintf(strings.TrimSuffix(rosterWideRow, "\n"),
		"", "", 0, "", 1234, 0, ""), "1")
	if start < 0 || next < 0 {
		t.Fatal("the element column was not found in the row at all, so this measured nothing")
	}
	if got, want := next-start, rosterElementCell; got != want {
		t.Fatalf("the element column and its gap are %d cells and the bound is written for %d",
			got, want)
	}

	singles, duals := 0, 0
	for _, primary := range element.All() {
		single, err := element.Single(primary)
		if err != nil {
			t.Fatalf("a single %s affinity is refused: %v", primary, err)
		}
		singles++
		measure(t, single)
		for _, secondary := range element.All() {
			dual, err := element.Dual(primary, secondary)
			if err != nil {
				// Refused pairs are the ones that cannot be drawn at all: the
				// same element twice, and anything paired with the inert one.
				continue
			}
			duals++
			measure(t, dual)
		}
	}
	// Premises, held rather than assumed: a walk that built no duals would pass
	// having measured only the shorter half of what this column holds.
	if singles < element.Count || duals == 0 {
		t.Fatalf("the walk measured %d single and %d dual affinities over %d elements",
			singles, duals, element.Count)
	}
}

// measure is one affinity's drawn cell against the column it is drawn in.
//
// ⚠️ **Exactly the column, rather than at most it.** The cell pads itself, so a
// short one is as wrong as a long one: a cell narrower than the column would let
// the format's own verb do the padding, which is the arrangement an inked cell
// breaks — and it would break it silently, in colour only, where no golden looks.
//
// Counted in runes for the reason the name bound is: Go's fmt pads a string verb
// by runes, so measuring bytes would refuse a name that fits.
func measure(t *testing.T, affinity element.Affinity) {
	t.Helper()
	drawn := elementCell(affinity, nil)
	if got := utf8.RuneCountInString(drawn); got != rosterElementCell {
		t.Errorf("the affinity %q draws the cell %q, which is %d cells against a column of "+
			"%d — that row's attack figure would be out of line by %d",
			affinity, drawn, got, rosterElementCell, got-rosterElementCell)
	}
}

// TestEveryStatTheRosterCanDrawFitsItsColumn is the attack and defence columns'
// bound, and finding it was the point.
//
// ⚠️ **The progression ceiling is not the bound.** A stat on the board is
// modifier.Set.Stat's answer, which saturates a buffed value towards
// `ceiling * Headroom / 1000` — 2400 under the shipped books against a ceiling of
// 800 — so a roster row can and does draw figures the ceiling forbids. What makes
// it a bound at all rather than an observation is that scale.Saturate never
// reaches its limit: `base + gap*delta/(gap+delta)` is strictly under `base+gap`
// in integer arithmetic, so the widest figure the engine can produce is one below
// the limit, and a unit's base is held at or under the ceiling by
// progression.Limits.CheckValues at enlistment — every unit, a summon included.
//
// So the bound is two books multiplied, and it is derived from the shipped copies
// of both. Raising either `ceilings.attack` or `headroom` past the point where
// the product needs a fifth digit reddens this, which is the day the column has
// to grow.
func TestEveryStatTheRosterCanDrawFitsItsColumn(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	for _, kind := range []progression.Kind{progression.Attack, progression.Defense} {
		ceiling := books.Limits.Ceilings[kind]
		limit := ceiling * books.Bounds.Headroom / modifier.PercentBase
		if limit <= ceiling {
			t.Fatalf("%s saturates towards %d against a ceiling of %d, so the books do not "+
				"leave a buff anywhere to go and this measured nothing", kind, limit, ceiling)
		}
		// The widest figure the engine can hand this column: the limit is never
		// reached, so it is one below.
		widest := limit - 1
		if digits := len(strconv.FormatInt(widest, 10)); digits > rosterStatRoom {
			t.Errorf("%s tops out at %d, which is %d digits against a column of %d — that "+
				"row's effects would be pushed out of line by %d",
				kind, widest, digits, rosterStatRoom, digits-rosterStatRoom)
		}
		t.Logf("%s: ceiling %d, headroom %d‰, so the widest figure is %d",
			kind, ceiling, books.Bounds.Headroom, widest)
	}
}

// TestAStatPastItsColumnPushesItsOwnRowRatherThanLosingADigit says what the
// column does when the bound above stops holding, because "sized to the bound"
// is only half an answer.
//
// Go's `%4d` is a minimum and not a maximum: a five-digit figure draws five cells
// and everything after it on that row moves one right. That is the same trade the
// name column already makes — a misalignment on one row, with every digit still
// readable — and it is deliberately not a clip, because a figure with a digit
// missing is a number a reader would believe.
//
// ⚠️ This is a statement about the drawing, so it is asserted by drawing. The
// guard against ever reaching it is the test above.
func TestAStatPastItsColumnPushesItsOwnRowRatherThanLosingADigit(t *testing.T) {
	row := strings.TrimSuffix(rosterWideRow, "\n")
	const marker = "EFFECTS"
	within := fmt.Sprintf(row, "", "", 0, "", 9999, 0, marker)
	past := fmt.Sprintf(row, "", "", 0, "", 10000, 0, marker)

	if !strings.Contains(past, "10000") {
		t.Errorf("a five-digit attack is not drawn whole, so the column is clipping a figure:\n%s", past)
	}
	if got, want := utf8.RuneCountInString(past)-utf8.RuneCountInString(within), 1; got != want {
		t.Errorf("one digit past the column widens the row by %d cells, want %d", got, want)
	}
	if got, want := strings.Index(past, marker)-strings.Index(within, marker), 1; got != want {
		t.Errorf("one digit past the column moves the rest of the row %d cells, want %d", got, want)
	}
}

// TestTheWideRosterHeadingStandsOverTheColumnsItNames is the wide table's half of
// the claim the narrow one already holds.
//
// Three columns arrived and the heading gained three fields, so the two format
// strings have three more chances to disagree — measured off both rather than
// counted by eye, which is what used to leave `effects` one cell left of the
// effects it named. And the whole heading has to fit the wide row's own budget in
// **every** language: the element column is the one field here a language
// changes.
func TestTheWideRosterHeadingStandsOverTheColumnsItNames(t *testing.T) {
	const mark = "|"
	// A four-digit figure rather than a character for each numeric field: those
	// are `%4d` in the row, so a shorter number is right-aligned and would report
	// the field as starting wherever the figure happened to start.
	const wide = 1234
	row := strings.TrimSuffix(rosterWideRow, "\n")
	rowCells := []int{
		strings.Index(fmt.Sprintf(row, mark, "", 0, "", 0, 0, ""), mark),
		strings.Index(fmt.Sprintf(row, "", mark, 0, "", 0, 0, ""), mark),
		strings.Index(fmt.Sprintf(row, "", "", wide, "", 0, 0, ""), "1"),
		strings.Index(fmt.Sprintf(row, "", "", 0, mark, 0, 0, ""), mark),
		strings.Index(fmt.Sprintf(row, "", "", 0, "", wide, 0, ""), "1"),
		strings.Index(fmt.Sprintf(row, "", "", 0, "", 0, wide, ""), "1"),
		strings.Index(fmt.Sprintf(row, "", "", 0, "", 0, 0, mark), mark),
	}
	headingCells := []int{
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, mark, "", "", "", "", "", ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", mark, "", "", "", "", ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", "", mark, "", "", "", ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", "", "", mark, "", "", ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", "", "", "", mark, "", ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", "", "", "", "", mark, ""), mark),
		strings.Index(fmt.Sprintf(rosterWideHeadingRow, "", "", "", "", "", "", mark), mark),
	}
	for index, want := range rowCells {
		if want < 0 {
			t.Fatalf("column %d was not found in the row at all, so this measured nothing", index)
		}
		if got := headingCells[index]; got != want {
			t.Errorf("column %d: the heading starts at cell %d and the row at %d", index, got, want)
		}
	}

	budget := rosterWideWidth - 1
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		drawn := rosterWideHeading(lang)
		if width := utf8.RuneCountInString(drawn); width > budget {
			t.Errorf("the %v wide heading is %d cells against a budget of %d:\n%s",
				lang, width, budget, drawn)
		}
		// And the element heading is really worded, rather than the catalog
		// answering blank for a key nobody filled in — which would fit
		// beautifully and name nothing.
		if strings.TrimSpace(lang.Text(i18n.RosterHeadingElement)) == "" {
			t.Errorf("%v has no wording for the element heading, so the line above measured a blank", lang)
		}
		// The two stat headings are ids and stay ids, which is this row's half of
		// internal/i18n's stat-label rule. Asserted because a later reader
		// translating them is the likely mistake, not an unlikely one.
		for _, label := range []string{"atk", "def"} {
			if !strings.Contains(drawn, label) {
				t.Errorf("the %v wide heading does not name %q:\n%s", lang, label, drawn)
			}
		}
	}
}
