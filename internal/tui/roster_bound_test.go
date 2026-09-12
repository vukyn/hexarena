package tui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// rowBudget is what one drawn line may spend.
//
// The window floor is 120 columns and every line leaves the last cell empty,
// because a line that fills it wraps on some terminals — so the budget is 119,
// which is the same figure internal/screen measures its wordings against. It is
// written down here rather than imported because internal/screen imports this
// package and not the other way round.
const rowBudget = 119

// TestTheWidestRosterRowLeavesTheRoomTheNextColumnsNeed records what this change
// bought, so the next one cannot quietly spend it twice.
//
// Before it the table's arithmetic ceiling was 119 of a budget of 119 — the
// widest row in any golden was exactly that — which is the whole reason the
// element, attack and defence columns had nowhere to go.
//
// ⚠️ **Two of the three figures below are derived from the format string rather
// than restated**, because a ceiling written down beside the thing it bounds
// stops being a measurement of it. `%-17s` and `%-26s` pad an empty string to
// their own widths and `%4d` pads a nought to four, so formatting the row with
// nothing in it but a full effects column measures exactly the columns the row
// has.
func TestTheWidestRosterRowLeavesTheRoomTheNextColumnsNeed(t *testing.T) {
	format := strings.TrimSuffix(rosterRow, "\n")
	widest := utf8.RuneCountInString(
		fmt.Sprintf(format, "", "", 0, strings.Repeat("e", effectsRoom)))

	// The premise: the effects column really can fill its room, so this ceiling
	// is a row that can exist rather than one the elision never reaches.
	if got := len(elided([]string{strings.Repeat("x", effectsRoom)}, effectsRoom)); got != effectsRoom {
		t.Fatalf("the effects column tops out at %d of the %d it has, so %d is not a reachable row",
			got, effectsRoom, widest)
	}

	// 110 is 17 + 26 + 4 + 3 + 60: the merged tag-and-name column, the health
	// bar, the speed, the gap after it and the effects. It was 119 — the merge
	// is the whole of the difference, because `effectsRoom` deliberately did not
	// grow to swallow the cells the merge freed.
	if want := 110; widest != want {
		t.Errorf("the widest roster row is %d cells, want %d", widest, want)
	}
	// ⚠️ **Nine cells, and that is the finding rather than the twenty-eight the
	// brief for this change expected.** Dropping ` (always)` freed eighteen cells
	// of *content* on the worst row, but `elided` spends content, not width: the
	// cap stayed at 60, so the freed cells went to drawing an effect the column
	// had been eliding. Only the merge moved the row's bound.
	//
	// What change 1 did buy is entry cost — a permanent entry fell from 19 cells
	// to 10 — which is what makes this column cheap to cut when the three new
	// columns need more than 9. Cutting it is that step's to make and to measure.
	if room := rowBudget - widest; room != 9 {
		t.Errorf("the row leaves %d of its %d-cell budget; the recorded room is 9",
			room, rowBudget)
	}
}

// TestTheRosterHeadingStandsOverTheColumnsItNames is what a hand-written
// heading cannot promise, and what a translated one needs twice.
//
// Two claims, and both are about a line that is no longer a literal. First, the
// heading's four fields start in the cells rosterRow puts the row's four fields
// in — measured off the two format strings rather than counted by eye, which is
// what used to leave `effects` one cell left of the effects it named. Second,
// the whole heading fits the row budget **in every language**: it is the one
// part of this table a language changes, and a sentence that fits in English
// says nothing about the language it was translated into.
func TestTheRosterHeadingStandsOverTheColumnsItNames(t *testing.T) {
	// Where each field begins, found by putting a marker in one field at a time
	// and reading back where it landed. ⚠️ The speed's marker is a four-digit
	// figure rather than a character, because that field is `%4d` in the row: a
	// shorter number is right-aligned and would report the field as starting
	// wherever the figure happened to start, which is a fact about the speed
	// rather than about the column.
	const mark = "|"
	const wide = 1234
	row := strings.TrimSuffix(rosterRow, "\n")
	rowCells := []int{
		strings.Index(fmt.Sprintf(row, mark, "", 0, ""), mark),
		strings.Index(fmt.Sprintf(row, "", mark, 0, ""), mark),
		strings.Index(fmt.Sprintf(row, "", "", wide, ""), "1"),
		strings.Index(fmt.Sprintf(row, "", "", 0, mark), mark),
	}
	headingCells := []int{
		strings.Index(fmt.Sprintf(rosterHeadingRow, mark, "", "", ""), mark),
		strings.Index(fmt.Sprintf(rosterHeadingRow, "", mark, "", ""), mark),
		strings.Index(fmt.Sprintf(rosterHeadingRow, "", "", mark, ""), mark),
		strings.Index(fmt.Sprintf(rosterHeadingRow, "", "", "", mark), mark),
	}
	for index, want := range rowCells {
		if want < 0 {
			t.Fatalf("column %d was not found in the row at all, so this measured nothing", index)
		}
		if got := headingCells[index]; got != want {
			t.Errorf("column %d: the heading starts at cell %d and the row at %d", index, got, want)
		}
	}

	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		drawn := rosterHeading(lang)
		width := utf8.RuneCountInString(drawn)
		if width > rowBudget {
			t.Errorf("the %v heading is %d cells against a budget of %d:\n%s",
				lang, width, rowBudget, drawn)
		}
		// And it is really worded, rather than the catalog answering blank for a
		// key nobody filled in — which would fit the budget beautifully.
		for _, key := range []i18n.Key{i18n.RosterHeadingUnit, i18n.RosterHeadingEffects} {
			if strings.TrimSpace(lang.Text(key)) == "" {
				t.Errorf("%v has no wording for %v, so the heading above measured a blank", lang, key)
			}
		}
		t.Logf("%v heading: %d cells of %d\n%s", lang, width, rowBudget, drawn)
	}
}

// TestEveryNameTheRosterCanDrawFitsItsColumn is the bound the merge removed the
// slack from.
//
// The name used to have twenty-one cells and the longest in the data is
// thirteen, so eight were never drawn on any row; the merged column is sized to
// the thirteen exactly. That is the trade — ⚠️ **a character authored later with
// a longer name would push the health bar right on its row and only its row,
// which is a table that misaligns quietly.** So the bound is a test that names
// the offender rather than slack nobody wrote down.
//
// ⚠️ Counted in runes, not bytes. Go's fmt measures a string verb's width in
// runes, so `phân thân 1` is eleven cells and thirteen bytes; measuring the
// bytes would refuse a name that fits.
func TestEveryNameTheRosterCanDrawFitsItsColumn(t *testing.T) {
	// The column's own width, read back out of the format string, so the format
	// and the bound below it cannot part company.
	column := strings.Index(fmt.Sprintf(strings.TrimSuffix(rosterRow, "\n"), "", "|", 0, ""), "|")
	if want := rosterTagRoom + rosterNameRoom + 1; column != want {
		t.Fatalf("the unit column is %d cells wide and the bound is written for %d", column, want)
	}

	for _, candidate := range namesTheRosterCanDraw(t) {
		if got := utf8.RuneCountInString(candidate.name); got > rosterNameRoom {
			t.Errorf("%s is %d cells and the roster column holds %d — that row's health bar would be pushed out of line by %d",
				candidate.what, got, rosterNameRoom, got-rosterNameRoom)
		}
	}
}

// rosterName is one name the roster column can be asked to draw, and where it
// came from — the message above is only useful if it names the offender.
type rosterName struct {
	what string
	name string
}

// namesTheRosterCanDraw collects every name the shipped data can put in that
// column.
//
// Three sources, and the third is the one a walk over the cast alone misses:
//
//  1. every form in the cast, because seed.ParseRoster resolves a placement to
//     the *form's* name rather than the character's;
//  2. every name the shipped roster draws, which also covers a flat entry —
//     one that writes its own name instead of naming a character;
//  3. every summon label, which is **not** a name in any book: battle.summon
//     numbers its units, so the drawn label is the summon's name, a space and a
//     counter. ⚠️ That is why the walk cannot stop at the cast — `phân thân`
//     fits in nine cells and `phân thân 12` needs twelve.
func namesTheRosterCanDraw(t *testing.T) []rosterName {
	t.Helper()
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("read the shipped cast: %v", err)
	}
	out := make([]rosterName, 0, 64)
	// Every form. The character's own name goes in too: it is what a summon
	// falls back to, and what an author reads when naming a first stage.
	bare := make([]string, 0, 64)
	for _, character := range characters.All() {
		out = append(out, rosterName{fmt.Sprintf("the character name %q", character.Name), character.Name})
		bare = append(bare, character.Name)
		for _, stage := range character.Stages {
			out = append(out, rosterName{
				fmt.Sprintf("%q's form %q", character.ID, stage.Name), stage.Name})
			bare = append(bare, stage.Name)
		}
	}
	placed, err := seed.Roster()
	if err != nil {
		t.Fatalf("read the shipped roster: %v", err)
	}
	for _, entry := range placed {
		out = append(out, rosterName{
			fmt.Sprintf("the roster entry %q", entry.ID), entry.Name})
	}
	// The summon labels. The counter is budgeted at two digits: it counts every
	// unit one caster has ever summoned in one battle and never resets, so a
	// third digit needs a hundred of them out of a single unit.
	const counter = " 99"
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	summons := 0
	for _, declared := range skills.Skills() {
		if declared.Summons == nil {
			continue
		}
		summons++
		if declared.Summons.Name != "" {
			out = append(out, rosterName{
				fmt.Sprintf("%q's summons, numbered", declared.ID),
				declared.Summons.Name + counter})
			continue
		}
		// A summon with no name of its own is numbered off its caster, so every
		// name a carrier could have is a label this skill can draw. Nothing
		// ships in this shape today; the arm is here so that the day one does,
		// the walk widens instead of going quiet.
		for _, name := range bare {
			out = append(out, rosterName{
				fmt.Sprintf("%q's summons off a caster called %q, numbered", declared.ID, name),
				name + counter})
		}
	}
	// Premises, held rather than assumed: a walk that found no forms and no
	// summons would pass this test having measured nothing at all.
	if len(out) < len(placed) || len(placed) == 0 {
		t.Fatalf("the walk collected %d names over a roster of %d", len(out), len(placed))
	}
	if summons == 0 {
		t.Fatal("no shipped skill summons anything, so the numbered-label arm measured nothing")
	}
	return out
}
