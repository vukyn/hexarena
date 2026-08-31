package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The width rule, both halves of it, measured rather than commented.
//
//	Prose wraps at the floor (minWidth). A data / table cell uses the window.
//
// Prose at the floor because a sentence measured against the real terminal
// would have two shapes — one per window — and TestEveryWordingFitsTheMinimumWidth
// would have nothing to hold; and because a paragraph run across two hundred
// columns is a line a reader loses their place in. Data at the window because
// minWidth is the width this program promises to draw in, not a ceiling on what
// it may spend: a restriction cut to "để dành cho loài dr…" is a row that
// stopped saying which species it is for, on a terminal with a hundred spare
// columns beside it.
//
// ⚠️ **Both directions are tested, and that is the point of the file.** A test
// that only says "a wide window widens the data" is satisfied by widening
// everything, which is how prose gets two shapes; a test that only says "the
// floor still draws what it drew" is satisfied by never widening anything at
// all. So TestAWideWindowWidensTheDataCells goes red if a data cell is put back
// on the floor, TestTheFloorDrawsWhatTheFloorAlwaysDrew goes red if the floor
// itself moves, and TestAWideWindowStillWrapsProseAtTheFloor goes red if
// somebody later "fixes" the prose the same way.
//
// Every case here is asserted against **what the cell says** — the whole value,
// read off the same accessor the screen reads it off — and never against a
// count of cells. A cell count is the arithmetic under test written out a second
// time, and it agrees with a wrong answer as readily as with a right one.

// dataCell is one clipped data value, measured at whatever width is asked for.
//
// whole is what the value says with nothing cut, taken from the same call the
// screen makes rather than spelt out here: a literal would be a second copy of
// the fixture and would drift from it in silence.
type dataCell struct {
	name  string
	whole string
	at    func(width int) string
}

// wideWindow is the terminal this file calls wide. Two hundred is what
// TestEveryWordingFitsTheMinimumWidth already renders at, so a data cell that
// spends it is spending a width the rest of the suite has agreed exists.
const wideWindow = 200

// TestAWideWindowWidensTheDataCells is the data half of the rule, at the four
// sites that clip a data value.
//
// Each case asserts three things and the first of them is the anti-vacuity one:
// the value has to be too long for the floor, or the other two assertions are
// about a value nothing was ever going to cut and the test measures nothing. The
// fixtures below therefore *author* the overflow — a wide restriction, a full
// allowlist, a deep art folder — rather than hoping the shipped books keep
// producing one. The widest detail the shipped status book happens to draw is a
// single cell over the floor today, and a width test resting on that is a width
// test one balance edit away from passing for the wrong reason.
func TestAWideWindowWidensTheDataCells(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, cell := range widenedCells(t, lang) {
			floor, wide := cell.at(minWidth), cell.at(wideWindow)
			if strings.Contains(floor, cell.whole) {
				t.Errorf("%s/%s: the floor already holds the whole of %q, so this fixture no longer overflows and the case measures nothing:\n%s",
					lang, cell.name, cell.whole, floor)
				continue
			}
			if !strings.Contains(floor, ellipsis) {
				t.Errorf("%s/%s: the floor neither holds the value whole nor marks it cut:\n%s",
					lang, cell.name, floor)
			}
			if !strings.Contains(wide, cell.whole) {
				t.Errorf("%s/%s: a %d-column window still does not say %q — the cell is measuring the floor rather than the window:\n%s",
					lang, cell.name, wideWindow, cell.whole, wide)
			}
			if strings.Contains(wide, ellipsis) {
				t.Errorf("%s/%s: a %d-column window still cut the value:\n%s",
					lang, cell.name, wideWindow, wide)
			}
		}
	}
}

// TestTheFloorDrawsWhatTheFloorAlwaysDrew is the half that stops the widening
// from becoming a change to the promise.
//
// Two windows, and the second is the one worth naming: bubbletea has not sent a
// WindowSizeMsg when the first frame is drawn, so m.width is nought and
// usableWidth stands the floor in for it. A cell reading m.width raw would draw
// itself a room of about minus twenty there and vanish, which is a defect no
// screenshot of a running program can show — the program has always been
// measured after its first resize.
func TestTheFloorDrawsWhatTheFloorAlwaysDrew(t *testing.T) {
	const beforeTheFirstSizeMessage = 0
	for _, lang := range i18n.Langs() {
		for _, cell := range widenedCells(t, lang) {
			floor := cell.at(minWidth)
			if unsized := cell.at(beforeTheFirstSizeMessage); unsized != floor {
				t.Errorf("%s/%s: before the first size message the cell draws\n%s\nand at the floor it draws\n%s",
					lang, cell.name, unsized, floor)
			}
			for _, line := range strings.Split(floor, "\n") {
				if width := lipgloss.Width(line); width > minWidth-1 {
					t.Errorf("%s/%s: at the floor the cell draws a line %d cells wide, over the %d there are:\n%s",
						lang, cell.name, width, minWidth-1, line)
				}
			}
		}
	}
}

// TestAWideWindowStillWrapsProseAtTheFloor is the second direction, and the one
// a change like this is most likely to break by accident.
//
// The sites here carry the program's own sentences, so they wrap at the floor
// whatever the window is: the same two short lines on a hundred-column terminal
// as on an eighty-column one. Widening them would read as consistency and would
// take TestEveryWordingFitsTheMinimumWidth's subject away — it renders at two
// hundred and measures against seventy-nine, which only says anything while the
// wording is wrapped to the floor before it gets there.
//
// The line count is the assertion rather than a width, because a width is what
// a widened version would still satisfy: prose wrapped to the window is inside
// the window by construction. What changes is how many lines it takes.
func TestAWideWindowStillWrapsProseAtTheFloor(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)

		// The fight's caution: the sentence under the win rate that stops the
		// figure being read as the roster's.
		fight := menuTo(t, base, screenFight)
		caution := func(width int) []string {
			m := fight
			m.width, m.height = width, 60
			return strings.Split(strings.TrimRight(m.fight.caution(m), "\n"), "\n")
		}

		// A save's own note, which is the other sentence wrapped this way.
		played := key(t, atABattleOf(t, base, 3), "ctrl+s")
		if played.play.err != nil {
			t.Fatalf("%s: the save failed, so no note was measured: %v", lang, played.play.err)
		}
		note := func(width int) []string {
			m := played
			m.width, m.height = width, 60
			return m.play.wrote(m)
		}

		for _, prose := range []struct {
			name string
			at   func(width int) []string
		}{
			{"the fight's caution", caution},
			{"a save's note", note},
		} {
			floor, wide := prose.at(minWidth), prose.at(wideWindow)
			if len(floor) < 2 {
				t.Fatalf("%s/%s: it does not wrap at the floor at all, so nothing here could tell a widened version apart",
					lang, prose.name)
			}
			if !slices.Equal(wide, floor) {
				t.Errorf("%s/%s: a %d-column window wrapped it into %d lines against the floor's %d — prose moved to the window:\n%s",
					lang, prose.name, wideWindow, len(wide), len(floor),
					strings.Join(wide, "\n"))
			}
		}
	}
}

// widenedCells is the four data values that used to be clipped at the floor, set
// up so that each of them overflows it.
//
// One builder for both tests, because the two are the same four cells asked
// different questions, and a second arrangement of the same fixtures is a second
// thing that can stop being the fixture the other one measures.
func widenedCells(t *testing.T, lang i18n.Lang) []dataCell {
	t.Helper()
	dir := scratchData(t)
	probe := aSkillTheWholeCastIsNamedOn(t, dir)
	artPath := someDeeplyFiledArt(t, dir)
	base, lib, _ := startIn(t, lang, dir)

	everyone := make([]string, 0, len(lib.Characters().All()))
	for _, character := range lib.Characters().All() {
		everyone = append(everyone, character.ID)
	}
	if len(everyone) < 2 {
		t.Fatalf("the fixture cast holds %d characters, too few for an allowlist to overflow a row", len(everyone))
	}

	// 1. picker.go — the per-row detail column. The row is the probe skill,
	//    whose restriction names the whole cast, and the cursor is put on it so
	//    the list's window contains it: a picker draws a frame around its cursor
	//    and a row nobody is looking at is a row nobody measures.
	kit := base.enter(screenNew).openKit()
	at := slices.IndexFunc(kit.picker.options, func(option pickOption) bool {
		return option.id == probe
	})
	if at < 0 {
		t.Fatalf("the kit picker does not list %q, so the detail column has nothing long in it", probe)
	}
	kit.picker.cursor = at
	detail := dataCell{
		name:  "the picker's detail column",
		whole: kit.picker.detail(kit, probe),
		at: func(width int) string {
			m := kit
			m.width, m.height = width, 60
			body, _ := m.picker.view(m)
			return lineHolding(t, body, probe)
		},
	}

	// 2. picker.go — the chosen line, which is a list of ids and has no slot cap
	//    on an allowlist: the whole cast may be on it.
	allowing := base.enter(screenSkills)
	allowing.skills.adding = true
	allowing.skills.keptWho = everyone
	allowlist := allowing.openAllowlist(skillFieldKeptForCharacters)
	chosen := dataCell{
		name:  "the picker's chosen line",
		whole: strings.Join(everyone, " "),
		at: func(width int) string {
			m := allowlist
			m.width, m.height = width, 60
			body, _ := m.picker.view(m)
			// The line holding two ids at once. A list row holds one, so this
			// cannot pick a row up by mistake — and looking for the first id
			// alone would find the row for that character instead.
			return lineHolding(t, body, everyone[0], everyone[1])
		},
	}

	// 3. skills.go — skillValueRoom through listValue, the same allowlist as it
	//    is read back on the form that opens the picker.
	list := dataCell{
		name:  "the form's allowlist row",
		whole: strings.Join(everyone, " "),
		at: func(width int) string {
			m := allowing
			m.width, m.height = width, 60
			return m.skills.value(m, skillFieldKeptForCharacters, skillLabelWidth(m))
		},
	}

	// 3b. skills.go — skillValueRoom through chanceHint, the other caller of the
	//     one declaration. A skill may apply any number of statuses and the
	//     reading grows a figure per status, so it is the same unbounded shape.
	applying := base.enter(screenSkills)
	applying.skills.adding = true
	typed, chances := someStatusesAndTheirChances(t, lib)
	applying.skills.inputs[skillFieldInflicts].SetValue(typed)
	inflicts := dataCell{
		name:  "the form's chance reading",
		whole: chances,
		at: func(width int) string {
			m := applying
			m.width, m.height = width, 60
			return m.skills.value(m, skillFieldInflicts, skillLabelWidth(m))
		},
	}

	// 4. form.go — artRoom. A path is the one chooser value with no bound, which
	//    is what its own comment says and what makes it data.
	art := base.enter(screenNew)
	for range fieldImage {
		art = key(t, art, "down")
	}
	art = chooseArt(t, art, artPath)
	image := dataCell{
		name:  "the art chooser's path",
		whole: artPath,
		at: func(width int) string {
			m := art
			m.width, m.height = width, 60
			return strings.TrimRight(m.form.row(m, fieldImage, formLabelWidth(m)), "\n")
		},
	}

	return []dataCell{detail, chosen, list, inflicts, image}
}

// lineHolding is the one line of a screen that says all of the given words.
//
// It reports rather than stops. A missing line is a broken fixture and worth
// failing on, but failing *fatally* here would take the other cases down with
// it: these run in one test over five cells, and the first one to lose its row
// would leave the rest unmeasured and the report saying nothing about them.
func lineHolding(t *testing.T, body string, words ...string) string {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		found := true
		for _, word := range words {
			if !strings.Contains(line, word) {
				found = false
				break
			}
		}
		if found {
			return line
		}
	}
	t.Errorf("no line of the screen says all of %v:\n%s", words, body)
	return ""
}

// aSkillTheWholeCastIsNamedOn writes a skill into a scratch data directory whose
// restriction names every character in the book, and hands back its id.
//
// The fixture authors the overflow instead of borrowing one. A restriction is
// worded around the ids it lists, so a skill kept for the whole cast draws a
// detail cell no eighty-column window can hold in either language — which is
// what the detail column has to be measured on, since the widest one the shipped
// books produce is over the floor by a single cell and only in Vietnamese.
//
// It is written as JSON rather than through Library.SaveSkill because the point
// is the file the next Load reads: the model is built from a directory, and a
// skill added to a library the model does not hold is a skill no screen lists.
func aSkillTheWholeCastIsNamedOn(t *testing.T, dir string) string {
	t.Helper()
	const id = "probe"
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	everyone := make([]any, 0, len(lib.Characters().All()))
	for _, character := range lib.Characters().All() {
		everyone = append(everyone, character.ID)
	}

	path := filepath.Join(dir, "skills.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var book map[string]any
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	declared, ok := book["skills"].([]any)
	if !ok || len(declared) == 0 {
		t.Fatalf("%s holds no skills to copy a shape from", path)
	}
	first, ok := declared[0].(map[string]any)
	if !ok {
		t.Fatalf("%s holds a skill that is not an object", path)
	}
	// A copy of a skill the book already accepts, so nothing here has to know
	// which fields the parser demands — only the two being changed.
	built := make(map[string]any, len(first)+1)
	for key, value := range first {
		built[key] = value
	}
	built["id"] = id
	built["name"] = id
	built["restrict"] = map[string]any{"characters": everyone}
	book["skills"] = append(declared, built)

	grown, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		t.Fatalf("write the book back: %v", err)
	}
	if err := os.WriteFile(path, grown, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return id
}

// someDeeplyFiledArt puts a piece of art under a folder deep enough that its
// path cannot fit an eighty-column row, and hands back the path.
func someDeeplyFiledArt(t *testing.T, dir string) string {
	t.Helper()
	folder := strings.Repeat("deep-folder-", 4) + "end"
	if err := os.MkdirAll(filepath.Join(dir, "assets", folder), 0o755); err != nil {
		t.Fatalf("create the folder: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "assets", folder, "hero.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write the art: %v", err)
	}
	return "assets/" + folder + "/hero.svg"
}

// someStatusesAndTheirChances is a list for the inflicts field long enough that
// its reading runs past the floor, and the reading it parses to.
//
// The reading comes from forge.ApplicationChances, the call the row itself
// makes. Spelling out "30% · 30% · …" here would be the same string written
// twice and would go on passing after the separator changed.
func someStatusesAndTheirChances(t *testing.T, lib *forge.Library) (string, string) {
	t.Helper()
	const enoughToOverflow = 8
	book := lib.StatusBook()
	if len(book) < enoughToOverflow {
		t.Fatalf("the status book holds %d kinds, too few for a chance reading to overflow a row", len(book))
	}
	entries := make([]string, 0, enoughToOverflow)
	for _, kind := range book[:enoughToOverflow] {
		entries = append(entries, kind.ID+":300")
	}
	typed := strings.Join(entries, ", ")
	applications, err := lib.ParseApplications(typed)
	if err != nil {
		t.Fatalf("the fixture list %q does not parse: %v", typed, err)
	}
	return typed, forge.ApplicationChances(applications)
}
