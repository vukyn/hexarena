package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # Choosing both sides, and the one rule a two-row catalogue cannot measure
//
// The away side used to be *the next row on the file, wrapping*. It is a chosen
// row now, and the two rules agree on almost every fixture there is: with one
// side saved they are the same row, and with two they are the same row whenever
// the reader has not moved the away cursor. So the guard below builds a
// catalogue where the two answers differ and names the one it wants — a test
// written on the shipped fixture would pass with the old line put back.

// aThirdSideSaved adds one more side to a catalogue twoSidesSaved has already
// filled, around a character neither of those two fields.
//
// ⚠️ **Built here rather than in the fixture, and the reason is the goldens.**
// twoSidesSaved is what every recorded render is drawn over, so a third side
// there would move this client's screens.golden — the squad catalogue's rows,
// its id column, the join screen's chooser — for the sake of one test's
// arithmetic. What this needs is a catalogue of three in one test, which is a
// carrier a test builds.
func aThirdSideSaved(t *testing.T, lib *forge.Library) placement.Squad {
	t.Helper()
	saved := lib.Squads()
	if len(saved) != 2 {
		t.Fatalf("the catalogue holds %d sides, so this is not the fixture it is written "+
			"against", len(saved))
	}
	fielded := map[string]bool{}
	for _, squad := range saved {
		fielded[squad.Units[0].Character] = true
	}
	for _, character := range lib.Characters().All() {
		if fielded[character.ID] {
			continue
		}
		third := aSideOf(t, character, len(saved))
		// A character the fixture cannot field is ordinary — a learnset the slots
		// cannot fill — so the next one is tried, exactly as twoSidesSaved does.
		if err := lib.SaveSquad(third); err != nil {
			continue
		}
		return third
	}
	t.Fatalf("no third character in the book could be fielded, so a catalogue of three " +
		"cannot be built")
	return placement.Squad{}
}

// aLoneSideSaved fills an empty catalogue with exactly one side.
func aLoneSideSaved(t *testing.T, lib *forge.Library) placement.Squad {
	t.Helper()
	if held := len(lib.Squads()); held != 0 {
		t.Fatalf("the catalogue already holds %d sides, so this is not the empty fixture",
			held)
	}
	for _, character := range lib.Characters().All() {
		only := aSideOf(t, character, 0)
		if err := lib.SaveSquad(only); err != nil {
			continue
		}
		return only
	}
	t.Fatalf("no character in the book could be fielded, so a catalogue of one cannot be " +
		"built")
	return placement.Squad{}
}

// awayPointedAt walks the away cursor onto a row with the key a reader presses
// and hands the chooser back standing there.
//
// ⚠️ **Pressed rather than assigned.** The away side is a standing answer
// somebody gave, so a test that wrote model.against by hand would be measuring
// its own idea of the chooser — and the whole defect being replaced here is an
// away side that was a perfectly good row nobody could choose.
func awayPointedAt(t *testing.T, m model, row int) model {
	t.Helper()
	chooser := m.enter(screenPairing)
	rows := len(chooser.squads.Saved)
	if row < 0 || row >= rows {
		t.Fatalf("row %d is not one of the catalogue's %d", row, rows)
	}
	for range rows {
		if chooser.against == row {
			return chooser
		}
		chooser = key(t, chooser, "right")
	}
	t.Fatalf("%d presses of right left the away cursor on row %d, want row %d",
		rows, chooser.against, row)
	return m
}

// TestTheOpponentIsTheRowChosenRatherThanTheNextOneOnTheFile is the guard the
// whole change is for.
//
// It is written over a catalogue of **three** because that is the smallest one
// on which the two rules disagree while the home side stays put: with the home
// cursor on the first row, *the next row wrapping* is the second and the chooser
// is pointed at the third. The disagreement is asserted rather than assumed, so
// a fixture that stopped producing it fails here instead of quietly agreeing
// with both rules.
//
// ⚠️ **The assertion is on the battle rather than on the cursor.** A chooser
// that moved perfectly and was read by nothing would pass a test that looked at
// model.against, and that is exactly the shape of defect this replaces: the old
// away side was a perfectly good row nobody could choose.
func TestTheOpponentIsTheRowChosenRatherThanTheNextOneOnTheFile(t *testing.T) {
	m, lib, _ := start(t, i18n.En)
	aThirdSideSaved(t, lib)

	chooser := m.enter(screenPairing)
	sides := chooser.squads.Saved
	if len(sides) != 3 {
		t.Fatalf("the catalogue holds %d sides, and three is what makes the two rules "+
			"disagree", len(sides))
	}
	// One press of right, from the row the away cursor opens on: the third side.
	chosen := key(t, chooser, "right")
	if chosen.taking != 0 || chosen.against != 2 {
		t.Fatalf("the cursors read home %d and away %d, want the first row and the third",
			chosen.taking, chosen.against)
	}
	wrapping := sides[(chosen.taking+1)%len(sides)]
	if wrapping.ID == sides[chosen.against].ID {
		t.Fatalf("the next side wrapping and the chosen one are both %q, so this measures "+
			"nothing", wrapping.ID)
	}

	fought := key(t, chosen, "enter")
	if fought.screen != screenBattle {
		t.Fatalf("enter on the chooser landed on screen %v", fought.screen)
	}
	if got, want := fought.battle.Home.ID, sides[0].ID; got != want {
		t.Errorf("the battle opened on %q as the home side, want %q", got, want)
	}
	if got, want := fought.battle.Away.ID, sides[2].ID; got != want {
		t.Errorf("the battle opened on %q as the other side, want the chosen %q; %q is the "+
			"next side on the file, wrapping", got, want, wrapping.ID)
	}
}

// TestBothCursorsMoveOnTheirOwnKeysAndTheBattleOpensOnBoth is the chooser being
// two choosers.
//
// Each key is pressed on its own and **both** cursors are read after each one,
// because a screen that moved them together would satisfy every assertion made
// about the cursor that was meant to move. The battle at the end is opened on
// exactly the two rows chosen, read off what it holds rather than off the
// screen the keys were pressed on.
func TestBothCursorsMoveOnTheirOwnKeysAndTheBattleOpensOnBoth(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	aThirdSideSaved(t, lib)
	chooser := m.enter(screenPairing)
	sides := chooser.squads.Saved
	if len(sides) != 3 {
		t.Fatalf("the catalogue holds %d sides, want three so a cursor can be moved twice "+
			"without coming back to where it started", len(sides))
	}

	for _, step := range []struct {
		name       string
		home, away int
	}{
		{"down", 1, 1},
		{"down", 2, 1},
		{"right", 2, 2},
		{"up", 1, 2},
		{"left", 1, 1},
		{"left", 1, 0},
	} {
		chooser = key(t, chooser, step.name)
		if chooser.taking != step.home || chooser.against != step.away {
			t.Fatalf("%q left the cursors at home %d and away %d, want %d and %d",
				step.name, chooser.taking, chooser.against, step.home, step.away)
		}
	}

	fought := key(t, chooser, "enter")
	if fought.screen != screenBattle {
		t.Fatalf("enter on the chooser landed on screen %v", fought.screen)
	}
	if got, want := fought.battle.Home.ID, sides[1].ID; got != want {
		t.Errorf("the battle holds %q as the home side, want the chosen %q", got, want)
	}
	if got, want := fought.battle.Away.ID, sides[0].ID; got != want {
		t.Errorf("the battle holds %q as the other side, want the chosen %q", got, want)
	}
	// The player commands the side they chose as theirs, which is what makes the
	// two halves mean anything: Take fields the home side as the ally half.
	if fought.battle.Side != hex.SideAlly {
		t.Errorf("the battle put the player on %v, want the ally half", fought.battle.Side)
	}
	// And the way back is the chooser, so a reader who wants the other pairing
	// gets it with esc rather than by going round through the menu.
	if back := key(t, fought, "esc"); back.screen != screenPairing {
		t.Errorf("esc from the battle went to screen %v, want the chooser that opened it",
			back.screen)
	}
}

// TestAnEmptyCatalogueStillReachesTheOneRefusal is the state a reader who has
// never built a side arrives in.
//
// The chooser has nothing to choose between, so it draws no rows and — this is
// the decision — says nothing about that itself: what it means is that a side
// has to be built, and that sentence belongs to draw.PlayScreen.Open, which is
// one keystroke away. A second wording here would be the refusal in two places,
// which is the thing pairing is written not to do.
func TestAnEmptyCatalogueStillReachesTheOneRefusal(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m := startEmpty(t, lang)
		chooser := m.enter(screenPairing)
		if held := len(chooser.squads.Saved); held != 0 {
			t.Fatalf("the empty fixture holds %d sides in %s", held, lang)
		}
		// Every key that moves a cursor, pressed on a list with no rows: a
		// modulo by nought is a panic and a clamp against -1 is a row that does
		// not exist.
		for _, name := range []string{"up", "down", "left", "right"} {
			chooser = key(t, chooser, name)
		}
		// The body alone, not the framed screen: the footer names the two
		// cursors by the same wordings the rows are labelled with, so a scan
		// over the whole drawing would find them whatever the rows did.
		body, _ := chooser.parts()
		for _, said := range []i18n.Key{i18n.PairingHome, i18n.PairingAway, i18n.PairingSameSide} {
			if strings.Contains(body, chooser.text(said)) {
				t.Errorf("the chooser over an empty catalogue draws %q in %s:\n%s",
					chooser.text(said), lang, body)
			}
		}

		opened := key(t, chooser, "enter")
		if opened.screen != screenBattle {
			t.Fatalf("enter on the empty chooser landed on screen %v in %s", opened.screen, lang)
		}
		if opened.battle.Fight != nil {
			t.Errorf("an empty catalogue built a battle in %s", lang)
		}
		if opened.battle.Err != nil {
			t.Errorf("an empty catalogue reported %v in %s, want no pairing rather than a "+
				"refusal", opened.battle.Err, lang)
		}
		if want := opened.text(i18n.SquadsEmpty); !strings.Contains(opened.screenContent(), want) {
			t.Errorf("the battle with no pairing does not say a side has to be built in %s:\n%s",
				lang, opened.screenContent())
		}
	}
}

// TestACatalogueOfOneFightsACopyOfItselfAndTheLogTellsThemApart is the state the
// owner of this program is actually in, and the one the chooser does the least
// for.
//
// Both cursors land on the only row there is, so the battle is a side against a
// copy of itself — which pairing.go argues is a real opponent rather than a
// degenerate one, on the grounds that placement.Squad.Take prefixes every unit
// id with the side it is fielded as. That is asserted here rather than taken on
// trust: it is the whole of what makes the two halves of such a battle
// distinguishable in a log.
func TestACatalogueOfOneFightsACopyOfItself(t *testing.T) {
	m := startEmpty(t, i18n.En)
	only := aLoneSideSaved(t, m.lib)

	chooser := m.enter(screenPairing)
	if held := len(chooser.squads.Saved); held != 1 {
		t.Fatalf("the catalogue holds %d sides, want the one just saved", held)
	}
	if !strings.Contains(drawnBody(chooser), chooser.text(i18n.PairingSameSide)) {
		t.Errorf("a catalogue of one does not say both halves are the same side:\n%s",
			drawnBody(chooser))
	}
	fought := key(t, chooser, "enter")
	if fought.battle.Fight == nil {
		t.Fatalf("a catalogue of one opened no battle: %v", fought.battle.Err)
	}
	if fought.battle.Home.ID != only.ID || fought.battle.Away.ID != only.ID {
		t.Fatalf("the battle holds %q against %q, want %q on both halves",
			fought.battle.Home.ID, fought.battle.Away.ID, only.ID)
	}

	ally, enemy := hex.SideAlly.String()+".", hex.SideEnemy.String()+"."
	seen := map[string]bool{}
	halves := map[string]int{}
	for _, unit := range fought.battle.Fight.Units() {
		if seen[unit.ID] {
			t.Errorf("two units in the battle are both called %q, so the log cannot tell "+
				"them apart", unit.ID)
		}
		seen[unit.ID] = true
		switch {
		case strings.HasPrefix(unit.ID, ally):
			halves[ally]++
		case strings.HasPrefix(unit.ID, enemy):
			halves[enemy]++
		default:
			t.Errorf("the unit %q is on neither half by its id, and the side prefix is what "+
				"tells the two copies apart", unit.ID)
		}
	}
	if halves[ally] != sideSize || halves[enemy] != sideSize {
		t.Errorf("the battle fields %d units named for the ally half and %d for the enemy "+
			"one, want %d each", halves[ally], halves[enemy], sideSize)
	}
}

// TestThePairingChooserDoesNotReachThePvPSquad is the claim pairing.go's own
// comment rests on, measured in both of the ways it can go wrong.
//
// A squad taken into a **match** is chosen on the join screen, out of that
// screen's own chooser, and model.dialling is what puts it in wire.Hello.Squad.
// The hot-seat cursors are not on that path, and this holds them off it:
//
//   - **By behaviour**: every key the pairing chooser answers, pressed, leaves
//     the join screen's choice exactly where it was.
//   - **By source**: model.taking and model.against are read in the two files
//     that own them and nowhere else, so a later edit that wires either into a
//     hello fails here rather than shipping. The behavioural half cannot state
//     that — it would pass a client that read a cursor on a path this test does
//     not walk.
func TestThePairingChooserDoesNotReachThePvPSquad(t *testing.T) {
	m, lib, _ := start(t, i18n.En)
	aThirdSideSaved(t, lib)

	joining := m.enter(screenJoin)
	// Moved off the first side, so a chooser reset to nought would be caught.
	joining = key(t, joining, "right")
	chosen, have := joining.join.Chosen()
	if !have {
		t.Fatalf("the join screen chose no squad to bring")
	}

	moved := joining.enter(screenPairing)
	for _, name := range []string{"up", "down", "left", "right", "left"} {
		moved = key(t, moved, name)
	}
	if moved.taking == m.taking && moved.against == m.against {
		t.Fatalf("no key moved either pairing cursor, so nothing was measured")
	}
	after, still := moved.join.Chosen()
	if !still || after.ID != chosen.ID {
		t.Errorf("the join screen would now bring %q (%v), want the unchanged %q",
			after.ID, still, chosen.ID)
	}
	if moved.join.Squad != joining.join.Squad {
		t.Errorf("the join screen's chooser moved from %d to %d",
			joining.join.Squad, moved.join.Squad)
	}

	// The source half. A field is read through a selector, so the declaration
	// itself and the one keyed literal that seeds it are not hits.
	owned := []string{"pairing.go", "subject.go"}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	scanned, read := 0, 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, is := node.(*ast.SelectorExpr)
			if !is || (selector.Sel.Name != "taking" && selector.Sel.Name != "against") {
				return true
			}
			read++
			if !slices.Contains(owned, name) {
				t.Errorf("%s reads %s, and the hot-seat cursors belong to the local pairing "+
					"alone — a squad going out on the wire is joinScreen.Chosen",
					name, selector.Sel.Name)
			}
			return true
		})
	}
	if scanned == 0 || read == 0 {
		t.Fatalf("scanned %d source files and found %d reads of the two cursors, so the walk "+
			"is measuring nothing", scanned, read)
	}
}
