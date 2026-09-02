package main

import (
	"encoding/json"
	"fmt"
	"math"
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
	draw "github.com/vukyn/hexarena/internal/screen"
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
// as on a two-hundred-column one. Widening them would read as consistency and would
// take TestEveryWordingFitsTheMinimumWidth's subject away — it renders at two
// hundred and measures against seventy-nine, which only says anything while the
// wording is wrapped to the floor before it gets there.
//
// The line count is the assertion rather than a width, because a width is what
// a widened version would still satisfy: prose wrapped to the window is inside
// the window by construction. What changes is how many lines it takes.
//
// ⚠️ **The fight's caution left this table when the floor moved to 120, and that
// is a reported loss of coverage rather than a tidy-up.** Its sentence took two
// lines at a 77-cell room and takes **one** at a 117-cell one, so floor-wrapped
// and window-wrapped now produce the identical single line and no assertion here
// can tell them apart — the anti-vacuity guard below said exactly that, by name.
// It is still wrapped at the floor in fight.go and it is still right to be; what
// is gone is anything that would notice if it stopped. The width sweep cannot
// cover it either: the sentence measures under 119 cells, so a window-wrapped
// version would pass that too. It comes back the day the wording grows.
//
// The species note replaces it — authored prose, wrapped at the floor by
// species.go for the same stated reason, and two lines at 117 on the longest
// kinds in the book.
func TestAWideWindowStillWrapsProseAtTheFloor(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)

		// A species note: the authored sentence under the kind listing. The
		// cursor is put on the kind with the longest note, which is the only one
		// that wraps at all — a one-line note could not tell the two rules apart
		// any more than the caution can.
		kinds := menuTo(t, base, screenSpecies)
		longest := 0
		for index, kind := range kinds.species.Kinds {
			if lipgloss.Width(kind.Note) > lipgloss.Width(kinds.species.Kinds[longest].Note) {
				longest = index
			}
		}
		kinds.species.Cursor = longest
		opening := firstWords(kinds.species.Kinds[longest].Note)
		note := func(width int) []string {
			m := kinds
			m.width, m.height = width, 60
			body, _ := m.species.View(m.ctx())
			return theBlockOpening(t, body, opening)
		}

		// A save's own note, which is the other sentence wrapped this way.
		played := key(t, atABattleOf(t, base, 3), "ctrl+s")
		if played.play.Err != nil {
			t.Fatalf("%s: the save failed, so no note was measured: %v", lang, played.play.Err)
		}
		wrote := func(width int) []string {
			m := played
			m.width, m.height = width, 60
			return m.play.Wrote(m.ctx())
		}

		for _, prose := range []struct {
			name string
			at   func(width int) []string
		}{
			{"the species note", note},
			{"a save's note", wrote},
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

// TestTheDamageRowKeepsItsReferencePairAtEveryWindow is what the floor moving to
// 120 turned the old wide-window test into, and the change of subject is the
// finding rather than a rename.
//
// ⚠️ **The row has no short form any more, and this is the bound that made
// removing it safe.** `Lang.DamageWithin` used to drop the reference pair when
// the composed line would not fit, and `damageRowRoom` computed the room it
// compared against. Both are gone: the line is four numbers in fixed wording,
// two of them the ceilings — 800 and 400, three digits each and never anything
// else — so the only room to grow is the per-strike figure and the total, and
// both are int64, which is nineteen decimal digits at the very most. That
// ceiling is measured below rather than argued: it comes to **89 cells in
// Vietnamese and 87 in English**, against a narrowest room of **97 and 98** —
// the two forms'; the two listings' are 100 and 103. There is no skill,
// authored or absurd, whose reading could have reached the drop.
//
// It used to be reachable and the old test reached it: at a floor of 80 the room
// was 61 cells and the fixture below cleared it with a power of 180,000,000 over
// 200 strikes. Forty more columns put it out of range for good, and PR #177's
// floor of 120 is what turned a runtime fallback into dead code.
//
// So the test asserts what is now true, in two parts, and each is a mutation
// somebody could make:
//
//   - **The bound.** The widest line four figures can ever compose is smaller
//     than the smallest room either site has. A floor lowered back under about
//     112 goes red here, and it goes red with the arithmetic printed rather than
//     with a fixture failing to overflow.
//   - **The row really draws the full reading**, at the floor and at a wide
//     window, at both sites, on the widest skill the fixture can author.
//
// ⚠️ **Neither figure is written down**, which is what stops this going vacuous
// now that nothing can fail it by overflowing. The ceiling comes out of
// `Lang.Damage` itself at `math.MaxInt64`, so a longer sentence moves it; the
// room is measured off the row the screen actually drew — the rendered line less
// the reading inside it is the marker, the label column and the space after it —
// so a wider label or a lower floor moves that. Nothing here counts cells by
// hand.
func TestTheDamageRowKeepsItsReferencePairAtEveryWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		dir := scratchData(t)
		heavy := aSkillWorthMoreThanTheShippedBookCanReach(t, dir)
		base, lib, _ := startIn(t, lang, dir)

		declared, err := lib.Skills().Lookup(heavy)
		if err != nil {
			t.Fatalf("%s: the book does not hold the fixture skill %q: %v", lang, heavy, err)
		}
		preview := lib.PreviewDamage(declared)
		full := lang.Damage(preview)

		// The widest reading there can be: both authored figures at the largest
		// value their type holds, the two ceilings as the row always draws them.
		// Taken from Lang.Damage rather than counted, so the wording is the
		// wording and a longer sentence moves the bound with it.
		widest := lipgloss.Width(lang.Damage(forge.SkillPreview{
			PerStrike: math.MaxInt64, Total: math.MaxInt64,
			Attack: preview.Attack, Defense: preview.Defense,
		}))

		listing := base.enter(screenSkills)
		listing.skills.Cursor = slices.IndexFunc(listing.skills.Skills,
			func(candidate skill.Skill) bool { return candidate.ID == heavy })
		if listing.skills.Cursor < 0 {
			t.Fatalf("%s: the listing does not show %q", lang, heavy)
		}

		form := base.enter(screenSkills)
		form.skills = form.skills.Prefill(base.ctx(), declared)

		for _, site := range []struct {
			name string
			at   func(width int) string
		}{
			{"the listing's damage row",
				func(width int) string {
					m := listing
					m.width, m.height = width, 60
					body, _ := m.skills.View(m.ctx())
					return body
				}},
			{"the form's damage row",
				func(width int) string {
					m := form
					m.width, m.height = width, 60
					return m.skills.DamageRow(m.ctx(), draw.SkillLabelWidth(m.ctx()))
				}},
		} {
			floor, wide := site.at(minWidth), site.at(wideWindow)
			if !strings.Contains(floor, full) {
				t.Errorf("%s/%s: the floor did not draw the whole reading. It wanted\n%s\nand drew\n%s",
					lang, site.name, full, floor)
				continue
			}
			if !strings.Contains(wide, full) {
				t.Errorf("%s/%s: a %d-column window did not draw the whole reading. It wanted\n%s\nand drew\n%s",
					lang, site.name, wideWindow, full, wide)
			}

			room := theRoomTheRowLeavesItsReading(t, floor, full)
			if widest > room {
				t.Errorf("%s/%s: the widest reading four figures can compose is %d cells, "+
					"and the row has %d of the %d-column floor — %d spent on the marker, "+
					"the label column and the space after it, and one column left empty "+
					"so a full line cannot wrap. Lang.DamageWithin and damageRowRoom "+
					"were deleted on the strength of %d <= %d, which no longer holds — "+
					"so this row needs a way to shorten itself again, or the floor has "+
					"to go back up",
					lang, site.name, widest, room, minWidth, minWidth-1-room, widest, room)
			}
			t.Logf("%s/%s: widest possible reading %d cells, room at the %d-column floor %d",
				lang, site.name, widest, minWidth, room)
		}
	}
}

// theRoomTheRowLeavesItsReading measures what a damage row has left for its
// value, off the row the screen actually drew.
//
// ⚠️ **Derived rather than written down, and that is the whole point of it.**
// The room used to come out of damageRowRoom, which this change deleted, and
// re-spelling `minWidth - 2 - labelWidth - 1` here would be that function living
// on in a test — free to disagree with the row, and unable to notice a label
// column that grew. So the overhead is read off the line instead: the rendered
// row less the reading inside it is exactly the marker, the label column and the
// space after it, whatever those come to today.
//
// The floor less one, because the window's last column is left empty — a line
// filling it wraps on some terminals, which is the same allowance
// TestEveryWordingFitsTheMinimumWidth makes. ⚠️ damageRowRoom did **not** make
// it (it spent `width - 2 - labelWidth - 1`), which is the one-cell defect filed
// at #175; deleting the function is what fixed it, and this is where that cell
// is now accounted for.
func theRoomTheRowLeavesItsReading(t *testing.T, screen, reading string) int {
	t.Helper()
	for _, line := range strings.Split(screen, "\n") {
		if !strings.Contains(line, reading) {
			continue
		}
		return minWidth - 1 - (lipgloss.Width(line) - lipgloss.Width(reading))
	}
	t.Fatalf("no line of the screen holds the reading %q, so its room cannot be measured:\n%s",
		reading, screen)
	return 0
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
		at := slices.IndexFunc(kit.picker.Options, func(option pickOption) bool {
			return option.ID == refused
		})
		if at < 0 {
			t.Fatalf("%s: the kit picker does not list %q", lang, refused)
		}
		kit.picker.Cursor = at
		if kit.picker.Options[at].Refusal == nil {
			t.Fatalf("%s: %q is carryable, so there is no refusal on screen to measure", lang, refused)
		}
		sentence := kit.lang.Error(kit.picker.Options[at].Refusal)
		if width := lipgloss.Width(sentence); width <= minWidth-3 {
			t.Fatalf("%s: the refusal is %d cells, which the floor's %d could hold whole — the fixture no longer overflows",
				lang, width, minWidth-3)
		}

		body := func(width int) string {
			m := kit
			m.width, m.height = width, 60
			drawn, _ := m.picker.View(m.ctx())
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

	everyone := enoughOfTheCastToOverflowTheFloor(t, lib)

	// 1. picker.go — the per-row detail column. The row is the probe skill,
	//    whose restriction names the whole cast, and the cursor is put on it so
	//    the list's window contains it: a picker draws a frame around its cursor
	//    and a row nobody is looking at is a row nobody measures.
	kit := base.enter(screenNew).openKit()
	at := slices.IndexFunc(kit.picker.Options, func(option pickOption) bool {
		return option.ID == probe
	})
	if at < 0 {
		t.Fatalf("the kit picker does not list %q, so the detail column has nothing long in it", probe)
	}
	kit.picker.Cursor = at
	detail := dataCell{
		name:  "the picker's detail column",
		whole: kit.picker.Detail(kit.ctx(), probe),
		at: func(width int) string {
			m := kit
			m.width, m.height = width, 60
			body, _ := m.picker.View(m.ctx())
			return lineHolding(t, body, probe)
		},
	}

	// 2. picker.go — the chosen line, which is a list of ids and has no slot cap
	//    on an allowlist: as many as the book declares may be on it.
	//
	//    ⚠️ **The species allowlist rather than the character one, and the floor
	//    is why.** It read the whole cast, which is six ids of 106 cells — over a
	//    floor of 80 and inside one of 120, so the cell stopped overflowing the
	//    day the floor moved and measured nothing. The cast is a *fixture* and
	//    cannot be grown without authoring characters; the species book is
	//    already grown to order by someLongSpeciesIDs, which sizes its ids off
	//    minWidth. Same call site, same unbounded shape, a fixture that survives
	//    the next floor. The form's own allowlist row below still reads the cast,
	//    but only as much of it as the floor cannot hold — see
	//    enoughOfTheCastToOverflowTheFloor.
	allowing := base.enter(screenSkills)
	allowing.skills.Adding = true
	allowing.skills.KeptWho = everyone
	allowing.skills.KeptKinds = speciesIDs
	allowlist := allowing.pick(allowing.skills.OpenAllowlist(allowing.ctx(), draw.SkillFieldKeptForSpecies))
	chosen := dataCell{
		name:  "the picker's chosen line",
		whole: strings.Join(speciesIDs, " "),
		at: func(width int) string {
			m := allowlist
			m.width, m.height = width, 60
			body, _ := m.picker.View(m.ctx())
			// The line holding two ids at once. A list row holds one, so this
			// cannot pick a row up by mistake — and looking for the first id
			// alone would find the row for that species instead.
			return lineHolding(t, body, speciesIDs[0], speciesIDs[1])
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
			return m.skills.FieldValue(m.ctx(), draw.SkillFieldKeptForCharacters, draw.SkillLabelWidth(m.ctx()))
		},
	}

	// 3b. skills.go — fieldValueRoom through chanceHint, another caller of the
	//     one declaration. A skill may apply any number of statuses and the
	//     reading grows a figure per status, so it is the same unbounded shape.
	applying := base.enter(screenSkills)
	applying.skills.Adding = true
	typed, chances := someStatusesAndTheirChances(t, lib)
	applying.skills.Inputs[draw.SkillFieldInflicts].SetValue(typed)
	inflicts := dataCell{
		name:  "the form's chance reading",
		whole: chances,
		at: func(width int) string {
			m := applying
			m.width, m.height = width, 60
			return m.skills.FieldValue(m.ctx(), draw.SkillFieldInflicts, draw.SkillLabelWidth(m.ctx()))
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

// theBlockOpening is the run of lines a wrapped paragraph occupies: the line
// that opens with the given words and every non-blank line under it.
//
// The run rather than a count taken from wrapWords, because the count is the
// thing under test — asking wrapWords again would be the arithmetic written out a
// second time and would agree with a wrong answer as readily as with a right one.
func theBlockOpening(t *testing.T, body, opening string) []string {
	t.Helper()
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if !strings.Contains(line, opening) {
			continue
		}
		block := []string{line}
		for _, next := range lines[index+1:] {
			if strings.TrimSpace(next) == "" {
				break
			}
			block = append(block, next)
		}
		return block
	}
	t.Fatalf("no line of the screen opens with %q:\n%s", opening, body)
	return nil
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

// namedCharacters is how many characters the probe's restriction lists.
//
// It is a fixed count rather than "the whole cast", and the two assertions it
// sits between are why: the detail cell has to be **too long for the floor and
// short enough for the wide window**, and a list that grows with the book only
// ever satisfies the first. It named everybody until 2026-08-31, which held at
// four characters and at five, and broke on the sixth — the wording ran past two
// hundred columns, so the cell was still cut at a width that is supposed to hold
// it whole and the case turned red for a reason that had nothing to do with the
// rule under test.
//
// Five clears the floor with room to spare in both languages and leaves about as
// much again before the wide window. A future id long enough to upset that fails
// one of the two assertions by name rather than silently, which is the whole
// reason both of them are asserted.
const namedCharacters = 5

// aSkillTheWholeCastIsNamedOn writes a skill into a scratch data directory whose
// restriction names namedCharacters of the book's characters, and hands back its
// id.
//
// The fixture authors the overflow instead of borrowing one. A restriction is
// worded around the ids it lists, so a skill kept for a row of them draws a
// detail cell no window at the floor can hold in either language — which is
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
	cast := lib.Characters().All()
	if len(cast) < namedCharacters {
		t.Fatalf("the fixture cast holds %d characters and the probe names %d, so the restriction "+
			"cannot be built to the width this measures", len(cast), namedCharacters)
	}
	named := make([]any, 0, namedCharacters)
	for _, character := range cast[:namedCharacters] {
		named = append(named, character.ID)
	}
	appendSkills(t, dir, []string{id}, func(built map[string]any, _ string) {
		built["restrict"] = map[string]any{"characters": named}
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
// ⚠️ **The repeat count is computed from minWidth rather than written down.** It
// was a literal 5 and that cleared a floor of 80 by 27 cells and a floor of 120
// by none at all — so the day the floor moved the fixture stopped overflowing and
// the guard below fired, which is the guard working and is also a fixture that
// has to be re-tuned by hand every time. It is derived now, so the overflow is a
// property of the fixture rather than of the floor it was written under.
func aKitOfLongSkillIDs(t *testing.T, dir string) []string {
	t.Helper()
	ids := longIDs(cast.SkillSlots, "long_", func(index int) string {
		return fmt.Sprintf("%d", index)
	})
	if width := lipgloss.Width(strings.Join(ids, " ")); width <= minWidth {
		t.Fatalf("the fixture kit is %d cells, which the floor's %d could hold", width, minWidth)
	}
	appendSkills(t, dir, ids, nil)
	return ids
}

// longIDs builds count ids out of a repeated stem, repeated as often as it takes
// for the space-joined list to clear the floor whole.
//
// Whole rather than the row's share of it: what each caller needs is a value no
// window at the floor could hold however the row is laid out, and the row's own
// arithmetic is the thing under test rather than something a fixture should
// reproduce. One helper for both lists so the two cannot drift into clearing
// different bars.
func longIDs(count int, stem string, suffix func(index int) string) []string {
	for repeat := 4; ; repeat++ {
		ids := make([]string, 0, count)
		for index := range count {
			ids = append(ids, strings.Repeat(stem, repeat)+suffix(index))
		}
		if lipgloss.Width(strings.Join(ids, " ")) > minWidth {
			return ids
		}
	}
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
// enoughOfTheCastToOverflowTheFloor is as many character ids as it takes to be
// too long for the floor, and no more.
//
// ⚠️ **A fixture may not be however big the shipped cast happens to be.** This
// cell reads real character ids, because a character allowlist is a list of
// them and a synthesised id would be measuring a row the picker cannot offer.
// But taking the *whole* cast makes the fixture grow every time somebody ships
// a character, and a value that grows without a ceiling eventually fails the
// other half of the rule: the seventh character pushed this row past the wide
// window, so the test went red over an authoring decision it has no opinion
// about. Sized off minWidth instead, the row overflows the floor by
// construction and stays well inside the window no matter how large the cast
// gets — the same shape someLongSpeciesIDs uses, pointed at a book this test
// cannot write to.
func enoughOfTheCastToOverflowTheFloor(t *testing.T, lib *forge.Library) []string {
	t.Helper()
	ids := make([]string, 0, len(lib.Characters().All()))
	for _, character := range lib.Characters().All() {
		ids = append(ids, character.ID)
		// One id past the floor: the first prefix the floor cannot hold. Stopping
		// at the first one *over* rather than at the first one that fits is what
		// makes the clipping assertion true by construction.
		if lipgloss.Width(strings.Join(ids, " ")) > minWidth {
			return ids
		}
	}
	t.Fatalf("the whole fixture cast is %d cells of ids, which the floor's %d could hold, so an allowlist row of it overflows nothing",
		lipgloss.Width(strings.Join(ids, " ")), minWidth)
	return nil
}

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
	ids := longIDs(count, "long-", func(index int) string { return fmt.Sprintf("%d", index) })
	for _, id := range ids {
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

// aSkillWorthMoreThanTheShippedBookCanReach writes a skill whose damage reading
// runs far past anything the balance data can produce, and hands back its id.
//
// A power rather than a longer name, because what has to grow is the *figures*:
// the reading is four numbers in a fixed sentence, and two of them are the
// ceilings the preview always draws against. The shipped book tops out at 59
// cells, so no skill in it makes the row worth measuring at all.
//
// ⚠️ **It was named `aSkillWorthMoreThanTheRowCanHold` and that stopped being
// true.** It was built to overflow `Lang.DamageWithin`'s room at a floor of 80,
// where the row had 61 cells; at 120 the row has ninety-odd, the drop became
// unreachable, and the fallback it exercised was deleted. What the fixture is
// for now is the other half of the same test — that the row draws the whole
// reading, on a skill whose reading is nothing like the shipped ones.
func aSkillWorthMoreThanTheShippedBookCanReach(t *testing.T, dir string) string {
	t.Helper()
	const id = "heavy_probe"
	// skill.resolve puts no ceiling on either field — it refuses a negative one
	// and nothing else — so both are values the parser accepts rather than values
	// forced past a bound. The strikes are here because the two figures are the
	// per-strike damage and the total, and multiplying the second is cheaper than
	// growing the first (see the ceiling below).
	//
	// ⚠️ **It has to be this absurd, and that is a finding rather than a fixture
	// detail.** The reading is four numbers in a fixed sentence, so it only grows
	// past what a balanced skill produces once the two being authored run to eight
	// and nine figures. Ninety million power was tried first and gave a
	// seven-figure reading of 63 cells — nothing like the shipped 59, but nothing
	// like a bound either. No skill anybody would author comes near this.
	//
	// ⚠️ **These two numbers are a fossil of a defect that has since been fixed,
	// and are kept only because nothing needs them changed.** combat.Rules.damage
	// used to build `attack × power × affinity × crit × DefenseConstant` in an
	// int64 before dividing, so the product wrapped in exactly this range and what
	// came back was **not monotone in the power**: measured, per strike,
	// 90,000,000 → 4,504,651 · 120,000,000 → **1** · 180,000,000 → 9,009,302, the
	// ones being MinimumDamage off a wrapped numerator. So the power could not
	// simply be raised, and 180,000,000 was picked as a **measured** point that
	// came back large, with the strikes carrying the rest of the width. #180
	// built that numerator in 128 bits and the function is monotone now, so the
	// power is free to be a round number — left alone because this fixture's job
	// is to be far past the shipped book, which it is either way, and rewriting a
	// working fixture's constants is not a comment correction.
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
	// ⚠️ **This id is squeezed from both ends, which is why it is half the floor
	// rather than all of it.** The sentence quoting it has to be too long for the
	// floor *and* short enough that a two-hundred-column window holds it whole —
	// the test asserts both — so an id sized to clear the floor on its own
	// overshoots and the wide case starts failing instead of the floor case. Half
	// the floor plus the forty-odd cells of fixed wording lands between the two,
	// and the band is wide because wideWindow is 200 against a floor of 120.
	id := strings.Repeat("long_", max(minWidth/10, 6)) + "skill"
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
// path cannot fit a row at the floor, and hands back the path.
func someDeeplyFiledArt(t *testing.T, dir string) string {
	t.Helper()
	// Repeated until the folder alone clears the floor — a literal four cleared
	// 80 and not 120, so the row held the path whole and the cell measured
	// nothing. See aPathPartLongerThanTheFloor.
	folder := aPathPartLongerThanTheFloor("deep-folder-", "end")
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
// ⚠️ **How many statuses is grown until the reading is long enough, never
// written down.** It was a constant eight, which overflowed a floor of 80 and sat
// 44 cells inside one of 120 — so the day the floor moved the cell stopped
// overflowing and measured nothing.
//
// The bar is **half the floor** rather than the whole of it, and that is the one
// number here worth arguing about: this reading does not get the row to itself —
// it is drawn after the typed list, which takes the rest — so a fixture sized to
// clear the whole floor overshoots and the row drops the chances entirely instead
// of cutting them. Half is the share that lands between, and the cell's own three
// assertions are what actually hold it: too short and the floor holds the value
// whole, too long and the wide window still cuts it.
func someStatusesAndTheirChances(t *testing.T, lib *forge.Library) (string, string) {
	t.Helper()
	book := lib.StatusBook()
	entries := make([]string, 0, len(book))
	for _, kind := range book {
		entries = append(entries, kind.ID+":300")
		typed := strings.Join(entries, ", ")
		applications, err := lib.ParseApplications(typed)
		if err != nil {
			t.Fatalf("the fixture list %q does not parse: %v", typed, err)
		}
		chances := forge.ApplicationChances(applications)
		if lipgloss.Width(chances) > minWidth/2 {
			return typed, chances
		}
	}
	t.Fatalf("the whole status book of %d kinds reads shorter than the %d cells a chance "+
		"reading has to clear at a floor of %d", len(book), minWidth/2, minWidth)
	return "", ""
}

// TestEveryFloorWrappedBlockTakesTheRowsItTakes writes down how many lines each
// floor-wrapped block occupies, so that moving the floor cannot change a row
// count in silence.
//
// ⚠️ **This is the vertical half of the width rule and it had nothing holding it.**
// Prose wraps at minWidth, so the floor decides how many *rows* a paragraph
// spends — and screens budget rows around those paragraphs. When the floor moved
// 80 → 120 the fight's caution went from two lines to one and the save's second
// note from three to two, and the only thing that noticed was a comment in
// play_test.go claiming five rows for a pair of notes that had become four. A
// width sweep cannot see this: every one of those lines is comfortably inside the
// window at either count.
//
// The numbers are hardcoded on purpose, which is the same decision the goldens
// under testdata are: they are the design record rather than a fixture to be
// regenerated. A change here is a change to how much of a screen a sentence eats,
// and it should be read rather than accepted.
//
// The save note is measured on its **second** note alone. The first names the
// file it wrote, so it carries a temp directory path — free text, as long as
// whatever the test framework handed out — and a count over it would be a count
// of the machine it ran on.
func TestEveryFloorWrappedBlockTakesTheRowsItTakes(t *testing.T) {
	// At minWidth = 120. See the constant's comment in internal/screen for where
	// 120 came from; these are what it costs vertically.
	rows := map[i18n.Lang]struct {
		caution, speciesNote, traitDescription, saveNote int
	}{
		i18n.Vi: {caution: 1, speciesNote: 2, traitDescription: 4, saveNote: 2},
		i18n.En: {caution: 1, speciesNote: 2, traitDescription: 3, saveNote: 2},
	}
	for _, lang := range i18n.Langs() {
		want, ok := rows[lang]
		if !ok {
			t.Fatalf("%s has no row counts written down, so a whole language is unmeasured", lang)
		}
		base, lib, _ := start(t, lang)
		base.width, base.height = minWidth, 60

		// fight.go — the caution under the win rate. Measured through the screen
		// rather than through wrapWords, so it is the row count the screen really
		// spends and not a second reading of the same expression.
		fight := menuTo(t, base, screenFight)
		fight.width, fight.height = minWidth, 60
		caution := strings.Split(strings.TrimRight(fight.fight.caution(fight), "\n"), "\n")
		if got := len(caution); got != want.caution {
			t.Errorf("%s: the fight's caution takes %d lines at the floor, want %d",
				lang, got, want.caution)
		}

		// species.go — the longest authored note, which is what speciesRoom
		// reserves for through longestNote.
		longest := 0
		for _, kind := range lib.Species().All() {
			if lines := len(wrapWords(kind.Note, minWidth-3)); lines > longest {
				longest = lines
			}
		}
		if longest != want.speciesNote {
			t.Errorf("%s: the longest species note takes %d lines at the floor, want %d",
				lang, longest, want.speciesNote)
		}

		// passives.go — the longest trait description, which is what
		// passivesRoom reserves six lines for.
		worst := 0
		for _, one := range lib.Passives().All() {
			lines := 0
			for _, sentence := range strings.Split(base.lang.DescribePassive(one), "\n") {
				lines += len(wrapWords(sentence, minWidth-1-draw.TraitIndent))
			}
			if lines > worst {
				worst = lines
			}
		}
		if worst != want.traitDescription {
			t.Errorf("%s: the longest trait description takes %d lines at the floor, want %d",
				lang, worst, want.traitDescription)
		}

		// play.go — the save note that is catalog wording rather than a path.
		played := key(t, atABattleOf(t, base, 3), "ctrl+s")
		if played.play.Err != nil {
			t.Fatalf("%s: the save failed, so no note was measured: %v", lang, played.play.Err)
		}
		notes := played.lang.Notes(played.play.Notes)
		if len(notes) < 2 {
			t.Fatalf("%s: a save left %d notes, so the one without a path in it is not there",
				lang, len(notes))
		}
		if got := len(wrapWords(notes[1], minWidth-1)); got != want.saveNote {
			t.Errorf("%s: the save's second note takes %d lines at the floor, want %d:\n%s",
				lang, got, want.saveNote, notes[1])
		}
	}
}
