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
// than better: three more screens move next, and after `cmd/hexarena` stands up,
// a screen the authoring tool stops drawing loses its only layout net **in
// silence**. So this file is that net, here, while the package holds six screens
// rather than nine.
//
// ## What is recorded
//
// The eight things the client's golden covers for these six screens — the six
// screens plus the two states nothing shipped can draw (a species kind nobody
// claims, a build that spends no trait slot) — in **both** languages at **two**
// sizes: the MinWidth x MinHeight floor, where the Room helpers bite, and 160x60,
// where nothing is squeezed. The entry names are the client's own, so a reader
// holding both diffs is looking at the same words.
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
// bytes. These six screens are read-only and nothing here writes.
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

// everyMovedScreen is the ten entries, named as cmd/hexforge-tui's
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
	return map[string]drawable{
		"builds":          builds,
		"chart":           ChartScreen{},
		"elements":        ElementsScreen{},
		"skill blurb":     skillBlurb(t, c, lib),
		"species":         species,
		"statuses":        NewStatusesScreen(lib),
		"trait blurb":     traitBlurb(t, c, lib),
		"traitless build": traitless,
		"traits":          NewPassivesScreen(lib),
		"unclaimed kind":  unclaimed,
	}
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
