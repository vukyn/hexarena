package screen

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// # The moved screens as a golden, in the package that now owns them
//
// Six listings moved out of cmd/hexforge-tui into this package. What did **not**
// move with them was the only test holding their layout: measured after that
// move, widening the status category column by one cell —
// `Pad(row.Category.String(), column+1)` → `column+2` in statuses.go — left
// **every test in this package green** and was caught by
// `cmd/hexforge-tui/testdata/screens.golden` alone.
//
//	internal/screen   → ok (all green)
//	client golden     → golden "  dot          sát thương mỗi lượt"
//	                    drawn  "  dot           sát thương mỗi lượt"
//
// That is real coverage sitting in another package, and it gets worse rather
// than better: two more screens move next, and after `cmd/hexarena` stands up,
// a screen the authoring tool stops drawing loses its only layout net **in
// silence**. So this file is that net, here, while the package holds eight
// screens rather than ten.
//
// ## What is recorded
//
// Sixteen entries over those eight screens — the six listings, the description
// screen in both of its readings, the two states nothing shipped can draw (a
// species kind nobody claims, a build that spends no trait slot) and the five
// states of the picker, which is handed its list and so has no one shape — in
// **both** languages at **two** sizes: the MinWidth x MinHeight floor, where the
// Room helpers bite, and 160x60, where nothing is squeezed. The entry names are
// the client's own, so a reader holding both diffs is looking at the same
// words.
//
// Each render carries a banner naming it, so a diff says which screen moved
// rather than which line number did. The names are **sorted** before they are
// walked: `everyMovedScreen` hands back a map, Go randomises a map range, and a
// golden built off one churns at random — the client's golden's first bug was
// exactly that, and it is the same rule `internal/core` states as "no map
// iteration in anything that reaches an output", one layer up.
//
// A screen answers with a body *and* a footer, so both are recorded. The footer
// is where the trims live — every wording squeeze in `CLAUDE.md` is a footer —
// and a golden holding the body alone would not see one.
//
// ## Two things the client's golden had to do that this one does not
//
// It drops its header line and hands `forge.Load` a **relative** directory,
// because `screenContent` draws `m.lib.Dir()` and the check screen prints the
// data directory as a body line. **Neither applies here.** Measured: no file in
// this package calls `.Dir()`, and `check` did not move — so the books are
// loaded **straight** from `shippedDataDir`, with no temp copy anywhere near the
// bytes. These seven screens are read-only and nothing here writes.
//
// ⚠️ The cast browser's art row is the one line that names a **file**, and it
// names a relative one: `cast.Character.StageArt` is a path out of the data
// files, and whether it is on disk is a yes or a no rather than a directory. So
// the golden records `assets/…` and never where this checkout happens to sit —
// which is exactly the claim `noAbsolutePath` below is here to keep true.
//
// ⚠️ **`noAbsolutePath` asserts that anyway.** A property that holds by
// construction today is one a later change breaks quietly, and the cost of
// finding out from a committed golden naming somebody's home directory is higher
// than the cost of the walk.
//
// ⚠️ **This is not a strict superset of the client's golden and may not replace
// it.** That one records the same screens *as the application draws them* —
// through `model.enter`, inside `frame`, with the header, the blank and the
// footer composed into one window and every line clipped to it. Measured both
// ways: a column width inside a moved `View` reddens **both**; widening
// `Ellipsis`, which only `frame`'s clip can reach on these screens, reddens the
// client's alone. Two goldens over one set of screens, measuring the drawing and
// the framing of it.
func TestEveryMovedScreenDrawsWhatTheGoldenHolds(t *testing.T) {
	path := filepath.Join("testdata", "screens.golden")
	got := everyMovedScreenDrawn(t)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s: %d renders, %d lines",
			path, strings.Count(got, bannerMark), strings.Count(got, "\n"))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: make golden): %v", err)
	}
	if got == string(want) {
		return
	}
	t.Errorf("the moved screens differ from %s; accept with `make golden` and read the diff\n%s",
		path, firstDifference(string(want), got))
}

// goldenSizes is the two windows every screen is recorded at.
//
// The floor is where the layout argument lives: every Room helper in this package
// spends `c.Height - 4` and then reserves what its panes need, so that is where a
// row budgeted wrong shows up. The roomy one is the same screens with nothing
// taken away, which is what makes a diff at the floor readable — a row that moved
// in both is a row that moved, and a row that moved only at the floor is a budget
// moving.
var goldenSizes = [][2]int{
	{MinWidth, MinHeight},
	{160, 60},
}

const (
	// bannerMark opens the line that names a render. Counting it counts renders.
	bannerMark = "===== "
	// footerMark separates a body from the footer under it. A screen hands back
	// the two apart, so the record has to say where one ends — and it is written
	// on a line of its own **after** the body verbatim, never after a trim, so a
	// body ending in a newline is a visible blank line here rather than a
	// difference the golden cannot show. A trailing empty row is a row the
	// client's frame budgets against; miscounting one is what once truncated the
	// picker.
	footerMark = "----- footer -----"
)

// everyMovedScreenDrawn is the whole golden: every screen, both languages, both
// sizes.
func everyMovedScreenDrawn(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	for _, lang := range i18n.Langs() {
		c, lib := startOverTheShippedBooks(t, lang)
		screens := everyMovedScreen(t, c, lib)
		names := slices.Sorted(maps.Keys(screens))
		for _, size := range goldenSizes {
			sized := c
			sized.Width, sized.Height = size[0], size[1]
			for _, name := range names {
				body, footer := screens[name].View(sized)
				fmt.Fprintf(&out, "%s%s | %dx%d | %s %s\n",
					bannerMark, lang, size[0], size[1], name, strings.TrimSpace(bannerMark))
				out.WriteString(body)
				out.WriteString("\n" + footerMark + "\n")
				out.WriteString(footer + "\n")
			}
		}
	}
	body := out.String()
	noAbsolutePath(t, body)
	return body
}

// drawable is the one thing every screen in this package has in common and the
// only thing this file needs of them. Six concrete types with six different
// constructors go into one map through it.
type drawable interface {
	View(Context) (string, string)
}

// everyMovedScreen is the sixteen entries, named as cmd/hexforge-tui's
// `everyScreen` names them.
//
// ⚠️ **Every hand-built state asserts that it drew the line it exists for.**
// A registered state that renders nothing passes every sweep over it, and this
// repository has shipped that fixture more than once — a screen entered at its
// early return, a battle already finished. Two of them are not reachable from
// the shipped books at all (every kind is claimed, every build spends its trait
// slot), so there is nothing else to notice if the construction stops working;
// the two blurbs are reachable, and are checked for the same reason — a subject
// built here with the wrong id draws the describer's "nothing to describe" arm,
// which is a perfectly well-formed screen that measures none of the code the
// entry was added for.
//
// ⚠️ **The blurb answers three subject kinds and gets two entries.** A listed
// skill and a battle option are **one** kind (see Subject.SkillSubject: same id,
// same paragraph, same footer — only At and Of differ), so a third entry would
// record the same render under a second name; and the third kind, NoSubject, is
// the arm a raise cannot reach, which a client's applier is what proves. The art
// preview moved with the blurb and is deliberately **not** here: it draws
// rasterised art, so what such an entry would assert is an open question rather
// than an oversight — see TODO.md § Not done.
func everyMovedScreen(t *testing.T, c Context, lib *forge.Library) map[string]drawable {
	t.Helper()
	species := NewSpeciesScreen(lib)
	unclaimed := withNobodyClaiming(species)
	if drawn, _ := unclaimed.View(c); !strings.Contains(drawn, c.Text(i18n.SpeciesNobodyIs)) {
		t.Fatalf("the unclaimed-kind state draws no line saying so, so the golden records "+
			"an ordinary species screen twice:\n%s", drawn)
	}
	builds := NewBuildsScreen(lib)
	traitless := withNoTraitTaken(t, builds)
	if drawn, _ := traitless.View(c); !strings.Contains(drawn, c.Text(i18n.BuildsNoTrait)) {
		t.Fatalf("the traitless-build state draws no row saying so, so the golden records "+
			"an ordinary build catalogue twice:\n%s", drawn)
	}
	listing := NewSkillsScreen(c)
	return map[string]drawable{
		// The cast browser as a raise leaves it: the first row, at the level
		// cap, with no origin filter. That is the state the client's own sweep
		// registers under this name, and a cursor moved here would be this
		// record answering a different question from that one.
		"allowlist picker": allowlistPicker(t, c, lib),
		"browse":           NewBrowseScreen(lib),
		"builds":           builds,
		"chart":            ChartScreen{},
		"elements":         ElementsScreen{},
		"filtered picker":  filteredPicker(t, c, lib),
		"kit picker":       kitPicker(t, c, lib),
		"reading a skill":  readingPicker(t, c, lib),
		"skill blurb":      skillBlurb(t, c, lib),
		"species":          species,
		"status picker":    statusPicker(t, c, lib),
		"statuses":         NewStatusesScreen(lib),
		"trait blurb":      traitBlurb(t, c, lib),
		"traitless build":  traitless,
		"traits":           NewPassivesScreen(lib),
		"unclaimed kind":   unclaimed,
		// The skill listing and the seven states of it the client's own sweep
		// registers, under the same names, for the reason every other entry
		// here carries the client's name: a reader holding both diffs is
		// looking at the same words.
		"skills":                  listing,
		"add a skill":             addingASkill(t, c, listing),
		"edit a skill":            editingASkill(t, c, listing),
		"edited a skill":          anEditedSkill(t, c, lib, listing),
		"filtering skills":        theFilterOpen(t, c, listing),
		"filtered skills":         theFilterFinding(t, c, listing),
		"skills filtered to none": theFilterFindingNothing(t, c, listing),
		"shape diagram":           theShapeDiagram(t, c, lib, listing),
	}
}

// # The skill listing's seven states, and what each is here for
//
// The listing is the ninth screen in this package and the busiest: it is a
// listing, a form over that listing, a diagram over that form, and a typed
// filter that is a mode of its own. The states below are the paths through View
// that share no line with each other, which is the same rule the picker's five
// were chosen under.
//
// ⚠️ **The three filter states are driven with the keys an author would press**,
// exactly as the client's fixture drives them, because the query is what decides
// which rows there are — a hand-set field would record this test's idea of the
// filter rather than the one `/` opens. The client's fixture has twice measured a
// screen's early exit instead of the screen; a driven state cannot do that.
//
// ⚠️ Each asserts it drew the line it exists for. A registered state that renders
// nothing passes every sweep over it, and two of these (an empty result, a
// reported edit) draw a *different* line rather than no line, so the failure is
// silent in both directions.

// addingASkill is the form over an empty draft: every field at its default,
// which is the state `a` reaches.
func addingASkill(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	form := pressOn(t, c, listing, "a")
	if !form.Adding || form.Editing != "" {
		t.Fatalf("a did not open the new-skill form (adding %v, editing %q)",
			form.Adding, form.Editing)
	}
	if drawn, _ := form.View(c); !strings.Contains(drawn, c.Text(i18n.SkillFormHeading)) {
		t.Fatalf("the new-skill form draws no heading of its own:\n%s", drawn)
	}
	return form
}

// editingASkill is the same form over a skill that already exists, which is the
// widest it ever draws: every field prefilled from the book rather than empty.
func editingASkill(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	if len(listing.Skills) == 0 {
		t.Fatal("the shipped book holds no skills to edit")
	}
	form := listing.Prefill(c, listing.Skills[0])
	if form.Editing != listing.Skills[0].ID {
		t.Fatalf("the form opened over %q, want %q", form.Editing, listing.Skills[0].ID)
	}
	if drawn, _ := form.View(c); !strings.Contains(drawn, c.Text(i18n.SkillFormEditHeading)) {
		t.Fatalf("the edit form draws the new-skill heading:\n%s", drawn)
	}
	return form
}

// anEditedSkill is the listing as an edit leaves it: two lines rather than one,
// the second of which is the damage before and after.
//
// The change is built by hand rather than written, because nothing in this
// package writes: the books are loaded straight out of internal/seed/data and a
// golden that edited them would edit the repository. What it needs is a
// forge.SkillChange, which is four values and no file.
func anEditedSkill(t *testing.T, c Context, lib *forge.Library, listing SkillsScreen) SkillsScreen {
	t.Helper()
	if len(listing.Skills) == 0 {
		t.Fatal("the shipped book holds no skills to edit")
	}
	before := listing.Skills[0]
	after := before
	after.Power = before.Power*2 + 1000
	reported := listing
	reported.Edited = &forge.SkillChange{
		Before: before, After: after,
		BeforeDamage: lib.PreviewDamage(before),
		AfterDamage:  lib.PreviewDamage(after),
	}
	if !reported.Edited.MovesDamage() {
		t.Fatal("the fixture edit moves no damage, so the second line is drawn by nothing")
	}
	drawn, _ := reported.View(c)
	if want := c.Text(i18n.SkillEdited, after.ID, lib.SkillsPath()); !strings.Contains(drawn, want) {
		t.Fatalf("the listing does not report the edit %q:\n%s", want, drawn)
	}
	return reported
}

// theFilterOpen is the field just opened with nothing in it, which says what to
// type rather than how much of the book is left.
func theFilterOpen(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	opened := pressOn(t, c, listing, "/")
	if !opened.Filtering {
		t.Fatal("/ did not open the filter, so its three states are drawn by nothing here")
	}
	if drawn, _ := opened.View(c); !strings.Contains(drawn, c.Text(i18n.SkillsFilterPrompt)) {
		t.Fatalf("the opened filter says nothing about what to type:\n%s", drawn)
	}
	return opened
}

// theFilterFinding is a query that has found several rows: the row above says
// how much of the book is left, and the listing under it is narrowed.
//
// ⚠️ The fixture's own discrimination is asserted, which no assertion downstream
// can see: a query that has quietly stopped matching turns this into a second
// copy of the state below, and both would still render.
func theFilterFinding(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	found := typeInto(t, c, theFilterOpen(t, c, listing), someSkillQuery)
	rows, all := len(found.Rows()), len(found.Skills)
	if rows < 2 || rows >= all {
		t.Fatalf("the query %q finds %d of %d skills, so the filtered listing is not "+
			"a narrowed one", someSkillQuery, rows, all)
	}
	return found
}

// theFilterFindingNothing is a query nothing answers to, which says so where the
// rows would have been and draws no column header over them.
func theFilterFindingNothing(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	nothing := typeInto(t, c, theFilterOpen(t, c, listing), noSkillQuery)
	if found := len(nothing.Rows()); found != 0 {
		t.Fatalf("the query %q finds %d skills, so the empty result is drawn by nothing",
			noSkillQuery, found)
	}
	if drawn, _ := nothing.View(c); !strings.Contains(drawn, c.Text(i18n.SkillsFilterNothing)) {
		t.Fatalf("the empty listing does not say the filter found nothing:\n%s", drawn)
	}
	return nothing
}

// theShapeDiagram is the board with a shape's coverage marked on it, which is a
// state of the form rather than a screen of its own.
//
// The shape is the one that catches the most cells, for the reason kitPicker
// takes the most-refused carrier: a golden of a layout is for the case that
// spends the most of it, and a single-cell shape draws a board with one mark.
func theShapeDiagram(t *testing.T, c Context, lib *forge.Library, listing SkillsScreen) SkillsScreen {
	t.Helper()
	shapes := lib.PatternNames()
	if len(shapes) == 0 {
		t.Fatal("the shipped book declares no shapes")
	}
	widest, most := shapes[0], -1
	for _, name := range shapes {
		coverage, err := lib.ShapeCoverage(name, defaultSkillTarget)
		if err != nil {
			t.Fatalf("coverage of %s: %v", name, err)
		}
		if coverage.Covered() > most {
			widest, most = name, coverage.Covered()
		}
	}
	diagram := addingASkill(t, c, listing)
	diagram.Field = SkillFieldShape
	diagram.ShapeIndex = IndexOf(shapes, widest)
	diagram.ShapeDrawn = true
	drawn, _ := diagram.View(c)
	if !strings.Contains(drawn, shapeSplashMark) {
		t.Fatalf("the diagram for %s marks no splash cell, so it records a board with "+
			"one mark on it:\n%s", widest, drawn)
	}
	return diagram
}

// # The five picker states, and why they are built here rather than raised
//
// The multi-select is the eighth screen in this package and the first with no
// listing of its own: it is handed a list, so a picker is a *state* rather than
// a screen with one shape, and which state is recorded is a decision.
//
// ⚠️ **They are hand-built and the client's are raised, and the names are the
// same on purpose.** The three screens that raise a picker — the character form,
// the skill form, the squad builder — are all still in cmd/hexforge-tui, so a
// state reached through one of them would be a state this package cannot make.
// What is recorded here is therefore the *drawing* under a list that says what
// this entry is for, and the client's golden of the same name is the same screen
// as its own raiser leaves it. Two records of one screen, which is the whole
// arrangement of this file.
//
// The five are the paths through View that share no line with each other: rows
// carrying a refusal and a detail column (the kit), rows with a filter line over
// them (the allowlist), that filter narrowed (the filtered one), a field and its
// percentage under the list (the statuses), and the reading pane, which replaces
// the list outright.
//
// ⚠️ Each asserts it drew the line it exists for, as every hand-built state in
// this file does. A picker built with the wrong options still renders a
// perfectly good screen — an empty list, an unmarked column, a filter that hides
// nothing — and every sweep over it would pass.

// kitPicker is the character form's kit picker over the shipped book, for the
// carrier that refuses the most of it.
//
// The most refused rather than the first character, for the reason skillBlurb
// takes the widest reading: what a golden of a layout is for is the case that
// spends the most of it, and here that is a list drawing both sorts of row. A
// carrier answering nothing restricts nothing, so the empty one draws a list
// with no marks on it at all.
func kitPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Fatal("the shipped book holds no characters to build a carrier from")
	}
	found, most := forge.Carrier{}, -1
	for _, character := range characters {
		who := forge.Carrier{
			ID: character.ID, Archetype: character.Archetype,
			Affinity: character.Element, HasAffinity: true,
			Species: character.Species, Origin: character.Origin,
		}
		refused := 0
		for _, option := range KitOptions(lib, who) {
			if option.Refusal != nil {
				refused++
			}
		}
		if refused > most {
			found, most = who, refused
		}
	}
	picker := (&PickState{
		Title: i18n.PickerKitTitle, Kind: PickSkills, Options: KitOptions(lib, found),
	}).Raise()
	offered := len(picker.Options) - most
	if most == 0 || offered == 0 {
		t.Fatalf("the kit picker refuses %d of %d rows, so it records a list with "+
			"only one sort of row on it", most, len(picker.Options))
	}
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "!") {
		t.Fatalf("no row of the kit picker carries the unavailable mark:\n%s", drawn)
	}
	return picker
}

// allowlistPicker is a restriction's character list: no slots, no refusals, and
// the one picker with a filter over it.
func allowlistPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := (&PickState{
		Title: i18n.PickerCharactersTitle, Hint: i18n.PickerAllowlistHint,
		Footer: i18n.PickerFilterFooter, Kind: PickCharacters,
		Options: CharacterOptions(lib), Groups: lib.OriginIDs(),
	}).Raise()
	if len(picker.Groups) == 0 {
		t.Fatal("the allowlist picker carries no works, so it draws no filter line")
	}
	drawn, _ := picker.View(c)
	showing := c.Text(i18n.PickerShowing, picker.groupName(c), len(picker.Visible()),
		len(picker.Options))
	if !strings.Contains(drawn, showing) {
		t.Fatalf("the allowlist picker draws no filter line %q:\n%s", showing, drawn)
	}
	return picker
}

// filteredPicker is that same list narrowed to one work, which is a line the
// unfiltered one does not draw and a row count the unfiltered one does not have.
//
// It is a second picker rather than the one above with the filter stepped on,
// because NextFilter mutates in place: sharing one would put the entry beside
// this into a state it is not here to measure.
func filteredPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := allowlistPicker(t, c, lib)
	picker.NextFilter()
	group := picker.Group()
	if group == "" {
		t.Fatal("stepping the filter left it on everything, so this records the " +
			"unfiltered list twice")
	}
	if len(picker.Visible()) == len(picker.Options) {
		t.Fatalf("the %q filter hides nothing of the %d rows", group, len(picker.Options))
	}
	if drawn, _ := picker.View(c); !strings.Contains(drawn, group) {
		t.Fatalf("the filtered picker does not name %q:\n%s", group, drawn)
	}
	return picker
}

// statusPicker is what a skill inflicts: the one picker of the five carrying a
// field, and the only entry here whose body has a row under the list.
//
// One status is chosen so the answer line under the list is the ids rather than
// the nothing-chosen wording, and the field is left empty so the placeholder and
// its percentage reading are what is recorded — which is the state a raise
// leaves it in, and the one the reading has to be right for.
func statusPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	statuses := StatusOptions(lib)
	if len(statuses) == 0 {
		t.Fatal("the shipped book declares no statuses")
	}
	picker := (&PickState{
		Title: i18n.PickerStatusesTitle, Hint: i18n.PickerStatusHint,
		Footer: i18n.PickerStatusFooter, Kind: PickStatuses,
		Options: statuses, Chosen: []string{statuses[0].ID},
		Typed: NumberField(c.Style.Plain, forge.DefaultApplicationChance),
		Label: i18n.PickerChance,
	}).Raise()
	drawn, _ := picker.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PickerChance)) {
		t.Fatalf("the status picker does not name its chance field:\n%s", drawn)
	}
	if !strings.Contains(drawn, statuses[0].ID) {
		t.Fatalf("the status picker does not draw %q as chosen:\n%s", statuses[0].ID, drawn)
	}
	return picker
}

// readingPicker is the kit picker with the description of the row under the
// cursor in front of the list, which is the picker's other state and shares no
// line with the list.
//
// It is the kit rather than the traits because the shipped skill book is the
// list this pane is most often reached from, and the row is walked to the
// longest reading for the reason skillBlurb is: this is the entry that records
// the line saying there is more to read.
func readingPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := kitPicker(t, c, lib)
	longest, most := 0, 0
	for index, option := range picker.Visible() {
		declared, err := lib.Skills().Lookup(option.ID)
		if err != nil {
			continue
		}
		if lines := len(SkillLines(c, declared)); lines > most {
			longest, most = index, lines
		}
	}
	picker.Cursor, picker.Reading = longest, true
	id := picker.Visible()[longest].ID
	if drawn, _ := picker.View(c); !strings.Contains(drawn, id) {
		t.Fatalf("the reading pane does not name %q, so it records the "+
			"nothing-to-pick arm:\n%s", id, drawn)
	}
	return picker
}

// skillBlurb is the description screen over the shipped skill whose reading
// takes the most lines.
//
// The widest reading rather than the first row, for the reason the client's
// fixture picks its widest element and its widest trait row: what a golden of a
// layout is for is the case that spends the most of it. Measured in **one**
// language whatever the reader's is, so both records describe the same skill —
// which skill is the widest is a fact about the book, and a record that changed
// subject between the two halves of the file would be two records.
//
// The position is where the skill sits in the book, exactly as the listing that
// raises this counts it: the raiser hands over At and Of, and Of at nought is how
// it says there was nothing to describe.
func skillBlurb(t *testing.T, c Context, lib *forge.Library) BlurbScreen {
	t.Helper()
	skills := lib.Skills().Skills()
	if len(skills) == 0 {
		t.Fatal("the shipped book holds no skills, so there is nothing to describe")
	}
	found, most := 0, 0
	for index, declared := range skills {
		lines := len(strings.Split(i18n.Vi.Describe(declared, lib.Patterns()), "\n"))
		if lines > most {
			found, most = index, lines
		}
	}
	blurb := BlurbScreen{Subject: Subject{
		Kind: SkillSubject, ID: skills[found].ID, At: found + 1, Of: len(skills),
	}}
	if drawn, _ := blurb.View(c); !strings.Contains(drawn, skills[found].ID) {
		t.Fatalf("the skill blurb does not name %q, so the golden records the "+
			"describer's nothing-to-describe arm:\n%s", skills[found].ID, drawn)
	}
	return blurb
}

// traitBlurb is the description screen over the shipped character carrying the
// most traits at the level cap.
//
// The most traits, because that is the state the screen's scroll exists for: five
// at the cap wrap past a 120x24 window, so this is the entry that records the
// line saying there is more to read. It is recorded **unscrolled**, which is
// where a raise leaves it.
func traitBlurb(t *testing.T, c Context, lib *forge.Library) BlurbScreen {
	t.Helper()
	characters := lib.Characters().All()
	found, most := 0, 0
	for index, character := range characters {
		held := len(lib.KitPassives(
			character.PassivesAt(progression.LevelCap, progression.Furthest)))
		if held > most {
			found, most = index, held
		}
	}
	if most < 2 {
		t.Fatalf("the widest shipped character carries %d traits at the cap, so this "+
			"entry records the carries-nothing line rather than the sentences", most)
	}
	blurb := BlurbScreen{Subject: Subject{
		Kind: CharacterSubject, ID: characters[found].ID, Level: progression.LevelCap,
		At: found + 1, Of: len(characters),
	}}
	held := lib.KitPassives(
		characters[found].PassivesAt(progression.LevelCap, progression.Furthest))
	drawn, _ := blurb.View(c)
	if !strings.Contains(drawn, held[0].ID) {
		t.Fatalf("the trait blurb does not name %q, the first trait %s carries at the "+
			"cap:\n%s", held[0].ID, characters[found].ID, drawn)
	}
	return blurb
}

// startOverTheShippedBooks is a Context over internal/seed/data itself.
//
// The rest of this package's tests go through `start`, which copies the data into
// a temp directory and injects the fixture cast — they need characters of their
// own to make an assertion about. A golden needs the opposite: the cast the
// repository actually ships, so the diff is the design record, and no temp path
// anywhere near the bytes. Nothing here writes, so a direct load is safe as well
// as simpler.
//
// NO_COLOR, as every fixture in this package sets: the styles then render as
// plain text, which is the whole reason a golden of a styled screen is readable
// at all.
func startOverTheShippedBooks(t *testing.T, lang i18n.Lang) (Context, *forge.Library) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load %s: %v", shippedDataDir, err)
	}
	return Context{
		Lib: lib, Lang: lang, Style: NewPalette(plainHere()),
		Width: MinWidth, Height: MinHeight,
	}, lib
}

// noAbsolutePath refuses a golden carrying this machine in it.
//
// ⚠️ It cannot fail today and is here for that reason. The books are loaded from
// a relative directory, no screen in this package prints one, and the two states
// are built in memory — so this holds **by construction**, and a property that
// holds by construction is a property a later change breaks without saying so. A
// golden naming `/var/folders/…` cannot be committed, and one naming a directory
// whose length varies cannot be reproduced twice on the same machine.
//
// The rooted-path rule is the client's: a separator at the front and another one
// further along, which is what tells a path from the bare `/` that several
// footers name as a key.
func noAbsolutePath(t *testing.T, body string) {
	t.Helper()
	separator := string(filepath.Separator)
	temp := filepath.Clean(os.TempDir())
	for number, line := range strings.Split(body, "\n") {
		if strings.Contains(line, temp) {
			t.Fatalf("line %d of the golden names the temp directory:\n%s", number+1, line)
		}
		for _, word := range strings.Fields(line) {
			if !strings.HasPrefix(word, separator) || !strings.Contains(word[len(separator):], separator) {
				continue
			}
			t.Fatalf("line %d of the golden names a filesystem path, which cannot be "+
				"committed — find where it enters and give it a relative name:\n%s",
				number+1, line)
		}
	}
}

// firstDifference names what moved: the render it happened under, the line
// number, and the two lines.
//
// The house style for a golden here is to print the whole of what was drawn,
// which works for a table and does not for thousands of lines of screen. The
// banner is what makes a single line readable instead — a diff that says
// `vi | 120x24 | statuses` has already told the reader which screen to look at.
//
// It is the client's helper of the same name, copied rather than shared, for the
// reason `firstWords` in this package's fixture is: a test helper is not code two
// suites may drift over, and reaching across would edit the file this whole
// exercise exists to stop depending on.
func firstDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	banner := "(before any banner)"
	for i := range max(len(wantLines), len(gotLines)) {
		if i < len(gotLines) && strings.HasPrefix(gotLines[i], bannerMark) {
			banner = strings.TrimSpace(gotLines[i])
		}
		switch {
		case i >= len(wantLines):
			return fmt.Sprintf("the drawn screens are longer: %d lines against the golden's %d; "+
				"line %d, under %s, is %q", len(gotLines), len(wantLines), i+1, banner, gotLines[i])
		case i >= len(gotLines):
			return fmt.Sprintf("the drawn screens are shorter: %d lines against the golden's %d; "+
				"line %d, under %s, was %q", len(gotLines), len(wantLines), i+1, banner, wantLines[i])
		case wantLines[i] != gotLines[i]:
			return fmt.Sprintf("line %d, under %s:\n  golden %q\n  drawn  %q",
				i+1, banner, wantLines[i], gotLines[i])
		}
	}
	return "the lines all match, so the difference is the trailing bytes"
}
