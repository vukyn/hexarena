package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// TestAPermanentEffectDrawsNoCountdownAndATimedOneStillDraws holds the two
// halves of one distinction, in one test, because the distinction is the thing
// worth keeping and either half alone passes while the other is broken.
//
// A permanent entry drawing ` (always)` spelled the common case out 1123 times
// across the goldens against 192 timed ones, and a timed entry already
// identifies itself — `poison x3 (2t)`. So the absence of a duration IS the
// notation for permanent, and what an implicit convention costs is that nothing
// on the row says so; the roster heading is where it is said.
//
// ⚠️ Asserted per snapshot entry rather than against a whole expected string.
// A fixture spelling out `phalanx x2, poison (3t)` would go red for a stack
// count moving, a status being renamed or the bench being re-kitted — none of
// which is what this is about.
func TestAPermanentEffectDrawsNoCountdownAndATimedOneStillDraws(t *testing.T) {
	fight, _ := opening(t)
	// A unit holding exactly one permanent effect, found rather than named: the
	// bench hands out traits and composition bonuses, and naming the unit that
	// happens to carry one today is a test that stops testing when the bench
	// moves under it.
	//
	// ⚠️ **Exactly one, not merely at least one**, so that adding the poison
	// below leaves a two-entry column that fits its room whichever way this is
	// drawn. Taking the first unit that holds anything permanent picks one
	// carrying three of them, and restoring ` (always)` then overflows the
	// column — which reddens this test on its elision premise instead of on the
	// claim it is about, a refusal that is right for the wrong reason.
	var unit *battle.Unit
	for _, candidate := range fight.Units() {
		held := candidate.Statuses.Snapshot()
		if len(held) == 1 && held[0].Permanent {
			unit = candidate
			break
		}
	}
	if unit == nil {
		t.Fatal("no unit on the bench holds exactly one permanent effect, so nothing here measures one")
	}
	poison, err := fight.Books().Statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup poison: %v", err)
	}
	unit.Statuses.Apply(poison, 120)

	drawn := tui.Effects(unit)
	// The premise, held rather than assumed: nothing is elided here, so every
	// entry really is on the line being read. A column that had dropped one
	// would pass the loop below by never meeting the entry that fails it.
	if strings.Contains(drawn, "+") {
		t.Fatalf("the column elided something, so this reads only part of it:\n%s", drawn)
	}
	entries := strings.Split(drawn, ", ")
	byID := make(map[string]string, len(entries))
	for _, entry := range entries {
		byID[strings.Fields(entry)[0]] = entry
	}

	permanents, timed := 0, 0
	for _, entry := range unit.Statuses.Snapshot() {
		drawnEntry, found := byID[entry.ID]
		if !found {
			t.Errorf("%q is held but not drawn:\n%s", entry.ID, drawn)
			continue
		}
		if entry.Permanent {
			permanents++
			if strings.Contains(drawnEntry, "(") {
				t.Errorf("the permanent %q draws a duration in %q; no duration is what permanent means",
					entry.ID, drawnEntry)
			}
			continue
		}
		timed++
		if want := fmt.Sprintf("(%dt)", entry.Remaining); !strings.HasSuffix(drawnEntry, want) {
			t.Errorf("the timed %q draws %q, want it to end in %q — a timed effect is the one that still says how long",
				entry.ID, drawnEntry, want)
		}
	}
	// Both arms have to have been exercised, or half of this measured nothing.
	if permanents == 0 || timed == 0 {
		t.Fatalf("the unit holds %d permanent and %d timed effects, so the distinction was never drawn",
			permanents, timed)
	}
	// And the wording it replaced is gone rather than merely shortened.
	if strings.Contains(drawn, "always") {
		t.Errorf("the column still spells the common case out:\n%s", drawn)
	}
}

// TestTheMergedColumnStillPutsTheTagWhereTheLogNamesIt is the constraint on
// merging the tag and the name into one column.
//
// The tag is how the log cross-references a unit — `A2 uses pummel` is only
// useful if `A2` can be found on the roster at a glance — so the merge is only
// allowed if the tag stays first, keeps its own cells, and still picks out
// exactly one row. That last part is the one a `strings.Contains` would miss:
// a tag buried inside a row is a tag a reader has to hunt for, and a tag
// matching two rows is not a cross-reference at all.
func TestTheMergedColumnStillPutsTheTagWhereTheLogNamesIt(t *testing.T) {
	fight, tags := opening(t)
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		lines := strings.Split(tui.Roster(lang, fight, tags), "\n")
		if len(lines) < 2 {
			t.Fatalf("%v: the roster drew %d lines", lang, len(lines))
		}
		// The heading over that column is whatever the catalog calls it, and it
		// stands at the first cell — asserted against the catalog rather than
		// against a word, because a word here is one language's.
		if heading := lang.Text(i18n.RosterHeadingUnit); !strings.HasPrefix(lines[0], heading) {
			t.Errorf("%v: the heading line %q does not open with %q", lang, lines[0], heading)
		}
		rows := lines[1:]
		if got, want := len(rows), len(fight.Units()); got != want {
			t.Fatalf("%v: the roster drew %d rows for %d units", lang, got, want)
		}
		for index, unit := range fight.Units() {
			tag := tags[unit.ID]
			if tag == "" {
				t.Fatalf("unit %q has no tag, so this row proves nothing about finding one", unit.ID)
			}
			row := rows[index]
			// First: the row opens with the tag and then the name, in that order
			// and in the cells the column gives each.
			if want := fmt.Sprintf("%-2s %s", tag, unit.Name); !strings.HasPrefix(row, want) {
				t.Errorf("%v: row %d is %q, want it to open with %q", lang, index, row, want)
			}
			// Second: findable. One row and one only begins with this tag.
			matches := 0
			for _, candidate := range rows {
				if strings.HasPrefix(candidate, tag+" ") {
					matches++
				}
			}
			if matches != 1 {
				t.Errorf("%v: %d rows open with the tag %q, want exactly the one the log means",
					lang, matches, tag)
			}
		}
	}
}

// TestACrowdedRosterGolden is the state the effects column elides in, put back
// into the record.
//
// ⚠️ **That state left the record when this table got its room back.** The
// goldens held four rows carrying a `+N` marker before the change; they hold
// none now, because a permanent entry stopped spelling out ` (always)` and
// nothing the bench fields overflows sixty cells any more. The *behaviour* is
// still held — TestTheEffectsColumnStaysInsideItsRoom goes red if the marker is
// dropped — but a unit test holds a promise and a golden is what shows a reader
// the shape of one, and the next step on this table takes its remaining cells
// out of effectsRoom, which turns eliding from "never happens" into the ordinary
// case.
//
// It lives in this package's own golden rather than a screen's for two reasons:
// the elision is this package's rule, so this is where a reader looks for it;
// and a screen golden would picture it framed, at one window, through a fixture
// that could stop fielding a crowded unit without anything noticing.
//
// ⚠️ Drawn in both languages, which no other roster golden does. The heading is
// now the one part of this table a language changes, and a record of one of them
// cannot show that the two stand over the same columns.
func TestACrowdedRosterGolden(t *testing.T) {
	fight, tags := opening(t)
	unit := crowdedUntilElided(t, fight)
	var b strings.Builder
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		fmt.Fprintf(&b, "== %v ==\n%s\n\n", lang, tui.Roster(lang, fight, tags))
	}
	// Named so a reader of the diff knows which unit the crowded row is.
	fmt.Fprintf(&b, "crowded: %s carries %d effects\n", unit.Name, len(unit.Statuses.Snapshot()))

	got := b.String()
	path := filepath.Join("testdata", "crowded.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/tui -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the crowded render differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// crowdedUntilElided loads the busiest unit on the board with real statuses off
// the shipped book until its effects column has to leave one out.
//
// ⚠️ **Reached by property rather than by typing a row.** What makes the column
// elide is an arithmetic fact about entry lengths and effectsRoom, and both move:
// a hand-built list that overflows sixty cells today is a fixture that quietly
// stops overflowing, leaving a golden that records an un-elided row under a name
// that says it elides. Applying real statuses until the marker appears either
// reaches the state or fails here saying it could not.
//
// The busiest unit rather than the first, because a unit already holding the
// traits and composition bonuses a side brought is the one a crowded row is
// really about — which is what the four rows that used to elide were.
func crowdedUntilElided(t *testing.T, fight *battle.Battle) *battle.Unit {
	t.Helper()
	var unit *battle.Unit
	for _, candidate := range fight.Units() {
		if unit == nil || len(candidate.Statuses.Snapshot()) > len(unit.Statuses.Snapshot()) {
			unit = candidate
		}
	}
	if unit == nil {
		t.Fatal("the bench fielded nobody")
	}
	// Timed statuses only: a permanent one is held rather than applied, and what
	// crowds a row in play is a pile of debuffs on top of what a side brought.
	for _, kind := range fight.Books().Statuses.Kinds() {
		if leftOut(unit) > 0 {
			return unit
		}
		if kind.Permanent {
			continue
		}
		unit.Statuses.Apply(kind, 120)
	}
	if leftOut(unit) == 0 {
		t.Fatalf("every timed status in the book is on %s and the column still draws all %d, "+
			"so this golden would record an un-elided row:\n%s",
			unit.Name, len(unit.Statuses.Snapshot()), tui.Effects(unit))
	}
	return unit
}

// leftOut is how many of a unit's effects the column did not draw.
//
// ⚠️ **Counted rather than read off the ` +N` marker**, and the difference is the
// whole point of the golden above. Looking for a "+" makes the marker both the
// thing under test and the thing that decides whether the test runs: deleting it
// then stops this fixture ever reaching its state, so the golden is never
// compared and the red arrives as "could not build the fixture" instead of as a
// row that has quietly stopped saying what it hid.
func leftOut(unit *battle.Unit) int {
	held := len(unit.Statuses.Snapshot())
	drawn := tui.Effects(unit)
	if held == 0 || drawn == "-" {
		return 0
	}
	return held - len(strings.Split(drawn, ", "))
}
