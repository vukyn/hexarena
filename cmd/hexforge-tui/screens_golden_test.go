package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// # The screens as a golden
//
// This package draws about ten thousand lines of screen and, until this file,
// held **no golden at all** — measured: `cmd/hexforge-tui/testdata` did not
// exist. What stood in for one was a set of *property* tests: every line fits
// the minimum width, every screen speaks the language it was asked for, no
// screen holds its own wording, no id reaches the screen without its gloss.
// Each of those is a promise about a shape. None of them is a promise about
// bytes, so a misplaced space, a changed clip point, a lost column or a row that
// quietly stopped being drawn passed **every test in the repository**.
//
// That gap matters now rather than in general: the reference-screen extraction
// (`TODO.md` § *Groundwork*, `README.md` § *PvP over a LAN*) has four steps left
// and each of them **moves a screen** into `internal/screen` for a second client
// to draw. "No golden moved" says nothing about a screen while no golden holds
// one, so this file is the net those steps are done over. It moves no code.
//
// ## What is recorded, and what is deliberately not
//
// Every entry `everyScreen` registers — the screens and the states, which is
// what makes a state registered there worth registering — in **both** languages
// at **two** sizes: the declared floor, where the budget and the clipping bite,
// and a roomy window where nothing clips. A banner names each render, so a diff
// says which screen moved rather than which line number did.
//
// ⚠️ **The header is not recorded, on purpose.** `screenContent` opens every
// screen with `programName` and `m.lib.Dir()`, so the first line of all two
// hundred renders names a directory on the machine that drew them. A committed
// golden cannot hold `/var/folders/cc/…/T/TestX3701085091/001`. It is
// **asserted and then dropped** rather than scrubbed: a regex over the output
// is free to stop matching, and a fixture that silently stops removing anything
// is a fixture that silently starts recording the wrong line. `bodyOf` demands
// the line it is about to drop really is the header and fails naming the render
// if it is not.
//
// ⚠️ **Dropping the header is not sufficient, which is why the data directory is
// named by a RELATIVE path.** Measured: the check screen's own count line
// (`i18n.CheckCounts`, `check.go`) prints `report.Dir` in the **body**, one line
// per check render, and a temp directory's name length is not even stable
// between two runs on one machine — it decides where that line clips. So the
// fixture hands `forge.Load` a relative name (see `startOverARelativeDataDir`)
// and the golden reads `data: 5 nguồn, …`. Every line of every screen is then
// recorded whole, and `noAbsolutePath` is what says so: two `everyScreen`
// entries build their own model over their own scratch directory, so a path can
// still arrive from somewhere this file did not think of, and that is a bug here
// rather than something to accept.
//
// ## What this golden does and does not prove
//
// It records **today's** bytes, so on its own it says nothing about whether the
// step before it changed a render — it would bake in a change as readily as a
// fidelity. That was measured separately and apart from this file, on the pair
// of commits that isolates the drawing-context extraction with no data change in
// between: `c576d7c` (before #199) and `f3e6676` (#199 itself, whose parent is
// c576d7c). Same capture, run from a detached worktree of each, dumping outside
// the repository: **8200 lines each, byte for byte identical — #199 moved no
// screen at all.**
//
// ⚠️ **`origin/main` was the wrong base and the difference is not academic**:
// #198 landed Politoed in between, and the same capture reads **232 diff lines**
// across the two — **42 of the 200 renders, on 12 distinct screens** (the skill
// listing and its three filter states, the kit picker, the blurb, the builds pair,
// the traits screen, the check and the spar), for reasons that have nothing to do
// with an extraction. The fixture keeps the shipped cast beside the injected one,
// so a data change reaches this golden; that is the golden working, and it is
// exactly why the extraction had to be measured on a commit pair with no data
// change in it.
//
// ⚠️ **The instrument was checked both ways before the reading was believed.**
// Run twice on one tree it is identical to itself (a naive version of it is not:
// `t.TempDir()`'s name length moves a clip point, and a fixture-written squad
// can reach the bytes through an entry that builds its own model). And widening
// `screen.Ellipsis` by one glyph moves **8 lines** of the dump — four elided rows
// in the skill listing, and four lines `clip` marked: the form's archetype row and
// the traits screen's carrier row, in each language — so a capture that passes is
// a capture that could have failed. ⚠️ The two halves of that check are not
// interchangeable: the identical-to-itself half is what says a difference means
// something, and the able-to-fail half is what says an identical pair means
// something. A capture with only the first is a capture that can only ever agree.
func TestEveryScreenDrawsWhatTheGoldenHolds(t *testing.T) {
	// Resolved before anything changes the working directory: the fixture below
	// moves into a scratch tree, and a golden path relative to "testdata" would
	// follow it there.
	path, err := filepath.Abs(filepath.Join("testdata", "screens.golden"))
	if err != nil {
		t.Fatalf("resolve the golden path: %v", err)
	}
	got := everyScreenDrawn(t)
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
	t.Errorf("the screens differ from %s; accept with `make golden` and read the diff\n%s",
		path, firstDifference(string(want), got))
}

// goldenSizes is the two windows every screen is recorded at.
//
// The floor is where this client's whole layout argument lives: the budget drops
// sections, `clip` marks the lines it cuts, and the footers sit one cell inside
// the edge. The roomy one is the same screens with nothing taken away, which is
// what makes a diff at the floor readable — a row that moved in both is a row
// that moved, and a row that moved only at the floor is the cut moving.
var goldenSizes = [][2]int{
	{minWidth, minHeight},
	{160, 60},
}

// bannerMark opens the line that names a render. Counting it counts renders.
const bannerMark = "===== "

// everyScreenDrawn is the whole golden: every screen, both languages, both
// sizes.
//
// ⚠️ **`everyScreen` returns a map, so the names are sorted before they are
// walked.** Go randomises a map range, and a golden built off one churns at
// random and is worth nothing — this is the same rule `internal/core` states as
// "no map iteration in anything that reaches an output", one layer up.
//
// The fixtures are built **once per language, in the roomy window**, and each is
// then drawn at both sizes. A screen reads `m.width` and `m.height` while it
// draws rather than while it is built, which is what makes that sound — and it
// is what `everyScreen`'s own `a squeezed battle` entry and
// `TestEveryWordingFitsTheMinimumWidth` both already do. The one thing it
// costs: that entry sets itself to the floor and is overwritten here, so its
// roomy row is an ordinary battle screen and only its floor row is squeezed.
func everyScreenDrawn(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	for _, lang := range i18n.Langs() {
		base := startOverARelativeDataDir(t, lang)
		roomy := goldenSizes[len(goldenSizes)-1]
		base.width, base.height = roomy[0], roomy[1]
		screens := everyScreen(t, base)
		names := slices.Sorted(maps.Keys(screens))
		for _, size := range goldenSizes {
			for _, name := range names {
				drawn := screens[name]
				drawn.width, drawn.height = size[0], size[1]
				where := fmt.Sprintf("%s | %dx%d | %s", lang, size[0], size[1], name)
				fmt.Fprintf(&out, "%s%s %s\n", bannerMark, where, strings.TrimSpace(bannerMark))
				out.WriteString(bodyOf(t, drawn, where))
			}
		}
	}
	body := out.String()
	noAbsolutePath(t, body)
	return body
}

// bodyOf is one screen's content with its header line dropped.
//
// ⚠️ The drop is **asserted**, which is the whole difference between this and a
// scrub: if the first line is ever not the header, this fails naming the render
// instead of quietly recording the screen one line short.
func bodyOf(t *testing.T, m model, where string) string {
	t.Helper()
	lines := strings.Split(m.screenContent(), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], programName) {
		first := ""
		if len(lines) > 0 {
			first = lines[0]
		}
		t.Fatalf("%s: the first line is %q, not the header this drops — "+
			"recording it would take a body line instead", where, first)
	}
	return strings.Join(lines[1:], "\n") + "\n"
}

// noAbsolutePath refuses a golden carrying this machine in it.
//
// The two things it is looking for are a `/`-rooted path and the OS temp
// directory, and either is a **defect in this file** rather than a fact to
// accept: a golden that names a directory cannot be committed, and one that
// names a directory whose length varies cannot even be reproduced twice on the
// same machine. It is the backstop for the relative data directory above,
// because `everyScreen` builds two of its models over scratch directories of
// their own and nothing here chose those names.
func noAbsolutePath(t *testing.T, body string) {
	t.Helper()
	separator := string(filepath.Separator)
	temp := filepath.Clean(os.TempDir())
	for number, line := range strings.Split(body, "\n") {
		if strings.Contains(line, temp) {
			t.Fatalf("line %d of the golden names the temp directory:\n%s", number+1, line)
		}
		for _, word := range strings.Fields(line) {
			// A separator at the front and another one further along, which is
			// what tells a rooted path from the footer key `/` — the skill
			// listing's filter is bound to it, so a bare separator is ordinary
			// wording on several of these screens.
			if !strings.HasPrefix(word, separator) || !strings.Contains(word[len(separator):], separator) {
				continue
			}
			t.Fatalf("line %d of the golden names a filesystem path, which cannot be "+
				"committed — find where it enters and give it a relative name:\n%s",
				number+1, line)
		}
	}
}

// relativeDataDir is the name the books are read under, and the reason it is a
// bare word is that it reaches the screen: the header names it and so does the
// check screen's count line.
const relativeDataDir = "data"

// startOverARelativeDataDir is startIn over a data directory named by a
// relative path, so no render carries an absolute one.
//
// It is the ordinary fixture — `scratchData`, so the golden is drawn over the
// same injected cast every other sweep here measures — copied into a scratch
// tree that this test then works from. ⚠️ The tree has a **second** copy in it,
// pristine, at wherever `shippedDataDir` lands from the working directory:
// `everyScreen` builds two of its models by calling `start` itself, so that path
// has to resolve after the move. It is computed from the constant rather than
// written out again, and checked to stay inside the scratch root — a
// `shippedDataDir` with one more `..` in it would otherwise have this test
// creating directories beside the repository.
func startOverARelativeDataDir(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	prepared := scratchData(t)
	root := t.TempDir()
	here := filepath.Join(root, "cmd", "hexforge-tui")
	data := filepath.Join(here, relativeDataDir)
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatalf("create %s: %v", data, err)
	}
	copyTree(t, prepared, data)
	shipped := filepath.Clean(filepath.Join(here, shippedDataDir))
	if !strings.HasPrefix(shipped, root+string(filepath.Separator)) {
		t.Fatalf("%s is outside the scratch root %s, so the fixture would write beside "+
			"the repository", shipped, root)
	}
	if err := os.MkdirAll(shipped, 0o755); err != nil {
		t.Fatalf("create %s: %v", shipped, err)
	}
	copyTree(t, shippedDataDir, shipped)
	t.Chdir(here)
	m, _, _ := startIn(t, lang, relativeDataDir)
	return m
}

// firstDifference names what moved: the render it happened under, the line
// number, and the two lines.
//
// The house style for a golden here is to print the whole of what was drawn,
// which works for a table and does not for eight thousand lines of screen. The
// banner is what makes a single line readable instead — a diff that says
// `vi | 120x24 | check` has already told the reader which screen to look at.
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
