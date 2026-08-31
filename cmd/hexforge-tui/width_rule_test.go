package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/skill"
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

// TestAWideWindowWidensTheDataCells is the data half of the rule, at the six
// sites that clip a data value.
//
// Each case asserts three things and the first of them is the anti-vacuity one:
// the value has to be too long for the floor, or the other two assertions are
// about a value nothing was ever going to cut and the test measures nothing. The
// fixtures below therefore *author* the overflow — a wide restriction, a full
// allowlist, a deep art folder, a kit of long ids — rather than hoping the
// shipped books keep producing one. The widest detail the shipped status book
// happens to draw is a
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

// TestTheDamageRowKeepsItsReferencePairOnAWideWindow is the data half again, at
// the one site where the floor was not clipping a value but **dropping** half of
// one.
//
// It is a test of its own rather than a widenedCells row because what the room
// decides here is a different act. Every cell in that table is cut and marks
// itself cut with an ellipsis; this row is never cut — over the room,
// Lang.DamageWithin returns the *short* reading and the reference pair the two
// figures are measured against is gone, leaving a whole sentence that has quietly
// stopped saying what it is relative to. So the assertion cannot be "the floor
// holds an ellipsis"; it is which of the two catalog lines came back.
//
// ⚠️ **The fixture authors a skill worth more than the row can hold, because the
// shipped book cannot reach this at all.** The widest reading the shipped skills
// produce is 59 cells in Vietnamese and 57 in English against a floor room of 61
// and 64, so every shipped row keeps its pair at every width and a test resting
// on one would be measuring nothing in either direction. An author typing a power
// is who reaches the drop, which is why the fixture types one.
//
// Both sites are covered — the listing's reading of the skill under the cursor
// and the form's preview of an unwritten one — because they are two call sites
// and reverting either alone has to go red.
//
// Both directions, as the file demands: the floor must still drop the pair (a
// mutation that always returns the full line goes red on the floor case) and the
// window must keep it (a mutation back to minWidth goes red on the wide case).
func TestTheDamageRowKeepsItsReferencePairOnAWideWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		dir := scratchData(t)
		heavy := aSkillWorthMoreThanTheRowCanHold(t, dir)
		base, lib, _ := startIn(t, lang, dir)

		declared, err := lib.Skills().Lookup(heavy)
		if err != nil {
			t.Fatalf("%s: the book does not hold the fixture skill %q: %v", lang, heavy, err)
		}
		preview := lib.PreviewDamage(declared)
		// Both readings off the same calls the screen makes. A room of one cell
		// cannot hold the full line, so it is how the short one is asked for
		// without spelling either of them out here.
		//
		// There is no anti-vacuity assertion on these two: a one-cell room drops
		// the pair whatever the figures are, so comparing them here would pass on
		// a fixture that overflows nothing. The real check is per site below —
		// the floor has to *draw* the short one — because the two sites measure
		// against different label columns and one can overflow while the other
		// does not.
		full := lang.Damage(preview)
		short := lang.DamageWithin(preview, 1)

		listing := base.enter(screenSkills)
		listing.skills.cursor = slices.IndexFunc(listing.skills.skills,
			func(candidate skill.Skill) bool { return candidate.ID == heavy })
		if listing.skills.cursor < 0 {
			t.Fatalf("%s: the listing does not show %q", lang, heavy)
		}

		form := base.enter(screenSkills)
		form.skills = form.skills.prefill(lib, declared)

		for _, site := range []struct {
			name string
			at   func(width int) string
		}{
			{"the listing's damage row", func(width int) string {
				m := listing
				m.width, m.height = width, 60
				body, _ := m.skills.view(m)
				return body
			}},
			{"the form's damage row", func(width int) string {
				m := form
				m.width, m.height = width, 60
				return m.skills.damageRow(m, skillLabelWidth(m))
			}},
		} {
			floor, wide := site.at(minWidth), site.at(wideWindow)
			if !strings.Contains(floor, short) {
				t.Errorf("%s/%s: the floor does not draw the short reading %q at all, so nothing here is the row under test:\n%s",
					lang, site.name, short, floor)
				continue
			}
			if strings.Contains(floor, full) {
				t.Errorf("%s/%s: the floor kept the reference pair, so this fixture no longer overflows and the case measures nothing:\n%s",
					lang, site.name, floor)
			}
			if !strings.Contains(wide, full) {
				t.Errorf("%s/%s: a %d-column window still dropped the reference pair — the row is measuring the floor rather than the window. It wanted\n%s\nand drew\n%s",
					lang, site.name, wideWindow, full, wide)
			}
		}
	}
}

// TestTheRefusalUnderTheCursorSpendsTheWindow is the picker's refusal sentence,
// and it is filed apart from the data cells on purpose.
//
// ⚠️ **This is prose and it takes the window anyway, so reading it as a data cell
// would draw the wrong conclusion from it.** The reason is in picker.go beside
// the line: frame already clips every line it draws at m.width, so measuring the
// floor here bought nothing frame was not doing and only did it seventy-nine
// cells early — and of the twenty-four sites that render a refusal this was the
// only one clipping at all, so it alone was cut while its siblings ran to the
// edge of a wide terminal. None of that is the data rule, and prose still wraps
// at the floor everywhere it wraps; what is different is that this one cannot
// wrap, which is the third assertion below.
//
// Three things, and each is a mutation somebody could make:
//
//   - The floor still cuts, and marks the cut. A mutation deleting clip and
//     leaving the sentence to frame goes red here: frame's MaxWidth cuts
//     silently, so the line would be inside the window with no ellipsis on it.
//   - The window is spent. A mutation back to minWidth goes red here.
//   - **The line count does not move.** (*pickState).room counts this refusal as
//     one of the seven rows the screen spends, so a "fix" that wrapped the
//     sentence instead of clipping it would push the bottom of the list under
//     frame's cut — the exact failure a scrolling list exists to prevent, and one
//     no width assertion can see.
func TestTheRefusalUnderTheCursorSpendsTheWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		dir := scratchData(t)
		refused := aRefusalTooLongForTheFloor(t, dir)
		base, _, _ := startIn(t, lang, dir)

		kit := base.enter(screenNew).openKit()
		at := slices.IndexFunc(kit.picker.options, func(option pickOption) bool {
			return option.id == refused
		})
		if at < 0 {
			t.Fatalf("%s: the kit picker does not list %q", lang, refused)
		}
		kit.picker.cursor = at
		if kit.picker.options[at].refusal == nil {
			t.Fatalf("%s: %q is carryable, so there is no refusal on screen to measure", lang, refused)
		}
		sentence := kit.lang.Error(kit.picker.options[at].refusal)
		if width := lipgloss.Width(sentence); width <= minWidth-3 {
			t.Fatalf("%s: the refusal is %d cells, which the floor's %d could hold whole — the fixture no longer overflows",
				lang, width, minWidth-3)
		}

		body := func(width int) string {
			m := kit
			m.width, m.height = width, 60
			drawn, _ := m.picker.view(m)
			return drawn
		}
		// The refusal is found by its own opening rather than by the skill id,
		// which the option's row carries too. It is the same needle
		// carriesFreeText recognises a line by.
		floorBody, wideBody := body(minWidth), body(wideWindow)
		floor := lineHolding(t, floorBody, firstWords(sentence))
		wide := lineHolding(t, wideBody, firstWords(sentence))

		if strings.Contains(floor, sentence) {
			t.Errorf("%s: the floor held the whole refusal, so nothing was measured:\n%s", lang, floor)
		}
		if !strings.Contains(floor, ellipsis) {
			t.Errorf("%s: the floor cut the refusal without marking it cut — frame's silent clip, not this one:\n%s",
				lang, floor)
		}
		if width := lipgloss.Width(floor); width > minWidth-1 {
			t.Errorf("%s: at the floor the refusal is %d cells, over the %d there are:\n%s",
				lang, width, minWidth-1, floor)
		}
		if !strings.Contains(wide, sentence) {
			t.Errorf("%s: a %d-column window still cut the refusal — it is measuring the floor:\n%s",
				lang, wideWindow, wide)
		}
		if strings.Contains(wide, ellipsis) {
			t.Errorf("%s: a %d-column window still marked the refusal cut:\n%s", lang, wideWindow, wide)
		}
		if floorRows, wideRows := len(strings.Split(floorBody, "\n")),
			len(strings.Split(wideBody, "\n")); floorRows != wideRows {
			t.Errorf("%s: the picker draws %d rows at the floor and %d at %d columns — the refusal changed the screen's row count, which is what room counts",
				lang, floorRows, wideRows, wideWindow)
		}
	}
}

// widenedCells is the data values that used to be clipped at the floor, set up
// so that each of them overflows it.
//
// One builder for both tests, because the two are the same cells asked different
// questions, and a second arrangement of the same fixtures is a second thing that
// can stop being the fixture the other one measures.
func widenedCells(t *testing.T, lang i18n.Lang) []dataCell {
	t.Helper()
	dir := scratchData(t)
	// Everything authored into the books is authored *before* the model starts,
	// because the model is built from the directory: an id written afterwards is
	// an id no screen lists and no picker can offer, so a cell reading one would
	// be measuring a state the program cannot reach.
	probe := aSkillTheWholeCastIsNamedOn(t, dir)
	artPath := someDeeplyFiledArt(t, dir)
	kitIDs := aKitOfLongSkillIDs(t, dir)
	speciesIDs := someLongSpeciesIDs(t, dir)
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

	// 3. skills.go — fieldValueRoom through listValue, the same allowlist as it
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

	// 3b. skills.go — fieldValueRoom through chanceHint, another caller of the
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

	// 5. form.go — kitValue, the chosen kit read back on the row the picker
	//    writes into. The kit is bounded (cast.SkillSlots is 4) and bounded is
	//    not the same as fitting, so the ids are authored long rather than taken
	//    from the book: four of the widest shipped ids come to 53 cells against a
	//    room in the fifties, which is a case that could stop overflowing on the
	//    day somebody renames a skill.
	kitted := base.enter(screenNew)
	kitted.form.kit = kitIDs
	kitRow := dataCell{
		name:  "the form's kit row",
		whole: strings.Join(kitIDs, " "),
		at: func(width int) string {
			m := kitted
			m.width, m.height = width, 60
			return strings.TrimRight(m.form.row(m, fieldKit, formLabelWidth(m)), "\n")
		},
	}

	// 6. form.go — speciesValue, kitValue's twin one field down. It is a case of
	//    its own rather than a second reading of the same arithmetic: the two are
	//    separate call sites, and reverting either alone has to go red.
	claimed := base.enter(screenNew)
	claimed.form.species = speciesIDs
	speciesRow := dataCell{
		name:  "the form's species row",
		whole: strings.Join(speciesIDs, " "),
		at: func(width int) string {
			m := claimed
			m.width, m.height = width, 60
			return strings.TrimRight(m.form.row(m, fieldSpecies, formLabelWidth(m)), "\n")
		},
	}

	return []dataCell{detail, chosen, list, inflicts, image, kitRow, speciesRow}
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
	appendSkills(t, dir, []string{id}, func(built map[string]any, _ string) {
		built["restrict"] = map[string]any{"characters": everyone}
	})
	return id
}

// aKitOfLongSkillIDs writes a full kit's worth of skills whose ids are long
// enough that the kit row cannot hold them at the floor, and hands back the ids.
//
// It authors the overflow rather than borrowing the widest ids the book happens
// to hold, and here that matters more than it does for the restriction: the kit
// is **bounded**, at cast.SkillSlots, so it is the one case where "it fits
// today" is a plausible reading. Four of the widest shipped ids come to 53 cells
// against a room in the fifties — which is a fixture that stops overflowing when
// somebody renames a skill, and a test that then passes while measuring nothing.
// The ids here are sized off minWidth so the join clears the whole floor, never
// mind the row's share of it.
func aKitOfLongSkillIDs(t *testing.T, dir string) []string {
	t.Helper()
	ids := make([]string, 0, cast.SkillSlots)
	for index := range cast.SkillSlots {
		ids = append(ids, fmt.Sprintf("%s%d", strings.Repeat("long_", 5), index))
	}
	if width := lipgloss.Width(strings.Join(ids, " ")); width <= minWidth {
		t.Fatalf("the fixture kit is %d cells, which the floor's %d could hold", width, minWidth)
	}
	appendSkills(t, dir, ids, nil)
	return ids
}

// someLongSpeciesIDs writes species whose ids overflow the species row, and
// hands back the ids.
//
// The species list is the **unbounded** twin of the kit — a character may be as
// many things at once as the book declares — so what this authors is not a
// bigger-than-usual case but the ordinary one at a size the shipped book has not
// reached yet. A species is an id and a word and nothing else, which is why this
// writes the objects itself rather than copying a shape the way the skill
// fixtures do.
func someLongSpeciesIDs(t *testing.T, dir string) []string {
	t.Helper()
	const count = 4
	path := filepath.Join(dir, "species.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var book map[string]any
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	declared, ok := book["species"].([]any)
	if !ok {
		t.Fatalf("%s declares no species list", path)
	}
	// Hyphens rather than the underscores the skill fixtures use: a species id is
	// held to lowercase letters, digits and hyphens, and a skill id is not. The
	// two spellings are the parsers disagreeing, not this fixture being tidy.
	ids := make([]string, 0, count)
	for index := range count {
		id := fmt.Sprintf("%s%d", strings.Repeat("long-", 5), index)
		ids = append(ids, id)
		declared = append(declared, map[string]any{"id": id, "name": id})
	}
	if width := lipgloss.Width(strings.Join(ids, " ")); width <= minWidth {
		t.Fatalf("the fixture species list is %d cells, which the floor's %d could hold",
			width, minWidth)
	}
	book["species"] = declared
	writeBook(t, path, book)
	return ids
}

// aSkillWorthMoreThanTheRowCanHold writes a skill whose damage reading cannot
// fit the row at the floor, and hands back its id.
//
// A power rather than a longer name, because what has to overflow is the
// *figures*: Lang.DamageWithin drops the reference pair on the width of the line
// it composes, and the line is four numbers in a fixed sentence. The shipped book
// tops out at 59 cells of the 61 there are, so no skill in it can reach the drop
// and a power is the only dial that gets there.
//
// The size is checked rather than trusted: the caller asserts that the floor
// really drew the short reading, so a power that stopped being enough fails
// loudly instead of quietly measuring a row that never dropped anything.
func aSkillWorthMoreThanTheRowCanHold(t *testing.T, dir string) string {
	t.Helper()
	const id = "heavy_probe"
	// skill.resolve puts no ceiling on either field — it refuses a negative one
	// and nothing else — so both are values the parser accepts rather than values
	// forced past a bound. The strikes are here because the two figures are the
	// per-strike damage and the total, and multiplying the second is cheaper than
	// growing the first (see the ceiling below).
	//
	// ⚠️ **It has to be this absurd, and that is a finding rather than a fixture
	// detail.** The pair is dropped on the width of a line holding four numbers,
	// so it only goes once the two being authored run to eight and nine figures.
	// Ninety million power was tried first: it produced a seven-figure reading of
	// 63 cells against the listing's 64 and kept its pair — one cell short of
	// measuring anything. So the drop is unreachable on any balanced skill, and
	// this fixture is the only thing in the suite that exercises it.
	//
	// ⚠️ **And the power cannot simply be raised, which is a defect one layer
	// down and the reason these two numbers are what they are.**
	// combat.Rules.damage builds
	// `attack × power × affinity × crit × DefenseConstant` before it divides, so
	// the product passes int64 in this range — and what comes back is **not
	// monotone in the power**, which is what makes it a trap rather than a
	// ceiling. Measured, per strike: 90,000,000 → 4,504,651 · 120,000,000 → **1**
	// · 150,000,000 → **1** · 180,000,000 → 9,009,302 · 200,000,000 → **1**. The
	// ones are MinimumDamage, which is what a wrapped numerator divides down to,
	// so the form draws a plausible-looking row for a power it has silently lost.
	// Out of scope here and reported rather than fixed; what it costs this fixture
	// is that the power is a **measured** point rather than a round one, and the
	// strikes carry the rest of the width.
	const (
		power   = 180_000_000
		strikes = 200
	)
	appendSkills(t, dir, []string{id}, func(built map[string]any, _ string) {
		built["power"] = power
		built["strikes"] = strikes
	})
	return id
}

// aRefusalTooLongForTheFloor writes a skill the blank character form cannot
// carry, with an id long enough that the refusal sentence runs past the floor,
// and hands back the id.
//
// **The restriction is derived rather than copied.** The form opens on the first
// origin in the book, so any *other* origin is a work this character is not out
// of and forge.CheckSkill refuses the skill for it — which is a property of the
// two ids rather than of whatever restriction the first declared skill happens to
// carry today. Inheriting a shape would tie the fixture to a balance decision
// somebody could reverse without ever looking at this file.
//
// The id is what makes the sentence long. The wording around it is about sixty
// cells in English and the ids are what push it over, which is the same reason
// whoMayCarry is exempt from the width sweep: the length of this line is decided
// by data the program cannot promise a size for.
func aRefusalTooLongForTheFloor(t *testing.T, dir string) string {
	t.Helper()
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	works := lib.Origins().All()
	if len(works) < 2 {
		t.Fatalf("the book catalogues %d works, so no skill can be kept for one the form is not on", len(works))
	}
	id := strings.Repeat("long_", 6) + "skill"
	appendSkills(t, dir, []string{id}, func(built map[string]any, _ string) {
		built["restrict"] = map[string]any{"origins": []any{works[1].ID}}
	})
	return id
}

// appendSkills copies the first declared skill's shape once per id, applies the
// caller's change to each copy, and writes the grown book back.
//
// A copy of a skill the book already accepts, so no fixture has to know which
// fields the parser demands — only the ones it is changing.
//
// It is written as JSON rather than through Library.SaveSkill because the point
// is the file the next Load reads: the model is built from a directory, and a
// skill added to a library the model does not hold is a skill no screen lists.
func appendSkills(t *testing.T, dir string, ids []string, change func(built map[string]any, id string)) {
	t.Helper()
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
	for _, id := range ids {
		built := make(map[string]any, len(first)+1)
		for key, value := range first {
			built[key] = value
		}
		built["id"] = id
		built["name"] = id
		if change != nil {
			change(built, id)
		}
		declared = append(declared, built)
	}
	book["skills"] = declared
	writeBook(t, path, book)
}

// writeBook puts a decoded book back on disk.
func writeBook(t *testing.T, path string, book map[string]any) {
	t.Helper()
	grown, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		t.Fatalf("write the book back: %v", err)
	}
	if err := os.WriteFile(path, grown, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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
