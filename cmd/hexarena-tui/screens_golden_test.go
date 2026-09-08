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
	"github.com/vukyn/hexarena/internal/wire"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// # This client's screens as a golden
//
// The third golden over one set of screens, and none of the three can stand in
// for another. `internal/screen/testdata/screens.golden` records the **drawing**
// — a body and a footer, as the package that owns them answers with them.
// `cmd/hexforge-tui/testdata/screens.golden` records the **authoring tool's
// framing** of them: the header, the blank, the vertical cut and its marker, the
// horizontal clip, and the eleven screens *that* client composes. This one
// records **this** client's, which differs in three ways that a property test
// cannot see and a golden can:
//
//   - The three footers that switch on `screen.Context.Authoring` are drawn in
//     their **read-only** spellings, which neither other golden holds. Measured
//     before this file existed: `i18n.SkillsReadFooter`, `i18n.OriginsReadFooter`
//     and `i18n.SquadsReadFooter` came back at **nought hits in both**, in both
//     languages, because they had no client to be drawn by.
//   - The squad catalogue is drawn **with rows on it**. The authoring client's
//     fixture deletes `squads.json` before it loads anything, so no test over
//     there draws a catalogue with a side in it — measured after #222, widening
//     that screen's id column by one cell leaves the whole authoring suite green.
//   - The battle is drawn at **three a side**, so the budget is deciding. A
//     one-a-side board, roster, order line and option list come to exactly the
//     twenty rows the floor leaves, so nothing is ever dropped and the notice
//     naming what the window was too short for is drawn by nothing.
//   - The **ban and pick** is drawn framed, and a **picker** is drawn over it.
//     `draw.DraftScreen` lives in `internal/screen` like every other screen this
//     client draws — see `screenDraft` for why it is not one of the lobby's
//     three — so its drawing is recorded there as well; what is only here is the
//     framing of it, and the two loadout lists, which are the first
//     `draw.PickState` any golden of *this* client has ever held: every picker in
//     the vocabulary was the authoring half of it until a drafted loadout arrived.
//
// ## What is recorded, and what is deliberately not
//
// Every entry `everyScreen` registers — the screens and the states, which is
// what makes a state registered there worth registering — in **both** languages
// at **two** sizes: the declared floor, where the budget and the clipping bite,
// and a roomy window where nothing clips. A banner names each render, so a diff
// says which screen moved rather than which line number did.
//
// ⚠️ **The header is not recorded, on purpose.** `frame` opens every screen with
// `programName` and `m.lib.Dir()`, so the first line of every render names a
// directory on the machine that drew them. It is **asserted and then dropped**
// rather than scrubbed: a regex over the output is free to stop matching, and a
// fixture that silently stops removing anything is a fixture that silently starts
// recording the wrong line. `bodyOf` demands the line it is about to drop really
// is the header and fails naming the render if it is not.
//
// ⚠️ **Dropping the header is not sufficient, which is why the data directory is
// named by a RELATIVE path.** The saved battle's own note names the file it
// wrote, in the **body**, and a temp directory's name length is not even stable
// between two runs on one machine — it decides where that line clips. So the
// fixture hands `forge.Load` a relative name and the note reads
// `data/battles/…`. `noAbsolutePath` is what says so about every recorded line,
// including the ones nothing here thought of.
//
// ⚠️ **The names are sorted before they are walked.** `everyScreen` hands back a
// map, Go randomises a map range, and a golden built off one churns at random —
// the authoring client's golden's first bug was exactly that, and it is the same
// rule `internal/core` states as "no map iteration in anything that reaches an
// output", one layer up.
//
// ## What this golden does and does not prove
//
// It records **today's** bytes, so on its own it says nothing about whether the
// client is right — it would bake in a defect as readily as a fidelity. What it
// is for is everything after today: a moved clip point, a column that lost a
// cell, a row that quietly stopped being drawn. The claims about *correctness*
// are the sweeps beside it, which is why neither may be dropped for the other.
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
// The floor is where this client's layout argument lives: the battle's budget
// drops sections, `draw.Clip` marks the lines it cuts, and the footers sit one
// cell inside the edge. The roomy one is the same screens with nothing taken
// away, which is what makes a diff at the floor readable — a row that moved in
// both is a row that moved, and a row that moved only at the floor is the cut
// moving.
var goldenSizes = [][2]int{
	{minWidth, minHeight},
	{160, 60},
}

// bannerMark opens the line that names a render. Counting it counts renders.
const bannerMark = "===== "

// relativeDataDir is the name the books are read under, and the reason it is a
// bare word is that it reaches the screen: the header names it, and so does the
// note a saved battle leaves.
const relativeDataDir = "data"

// everyScreenDrawn is the whole golden: every screen, both languages, both
// sizes.
//
// The fixtures are built **once per language, in the roomy window**, and each is
// then drawn at both sizes. A screen reads the window while it draws rather than
// while it is built, which is what makes that sound. The one thing it costs: the
// two entries that set a window for themselves — the squeezed battle and the
// scrolled log — are overwritten here, so their roomy rows are ordinary battle
// screens and only the floor row of the first is squeezed. What survives is the
// state each was built for, which is the part a window cannot rebuild.
func everyScreenDrawn(t *testing.T) string {
	t.Helper()
	// Resolved before the fixture moves the working directory, for the reason the
	// golden path above is.
	shipped, err := filepath.Abs(shippedDataDir)
	if err != nil {
		t.Fatalf("resolve the shipped data directory: %v", err)
	}
	var out strings.Builder
	for _, lang := range i18n.Langs() {
		base := startOverARelativeDataDir(t, lang, shipped)
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
	body := withoutTheRunningDataDigest(t, out.String())
	noAbsolutePath(t, body)
	return body
}

// digestPlaceholder stands in for the data digest wherever a screen draws it.
//
// ⚠️ **Same length as what it replaces**, which is the reason this redacts rather
// than dropping the line the way bodyOf drops the header. The digest sits in the
// *body* of a screen, so its row's presence and its position are part of what the
// golden holds — and a row that clips has to clip here where it clips there. A
// shorter stand-in would move a clip point and record a screen nobody draws.
//
// ⚠️ **Only the short form, because only the short form is drawn.** A stand-in
// for the sixty-four-character `Digest.String()` was written and then deleted: no
// screen prints it — the join line is `Short()` and `-version` is not a screen —
// so the branch could not fire, and a branch that cannot fire reads as a
// protection somebody is relying on. If a screen ever starts drawing the long
// form, TestTheCommittedGoldenNamesNoDataDigest fails on it and the redaction is
// what to add.
const digestPlaceholder = "xxxxxxxxxxxx"

// withoutTheRunningDataDigest takes this build's data digest out of the recorded
// body.
//
// ⚠️ **This golden moved on every data commit, and only this line did it.**
// Measured 2026-09-06: hiding one character left `internal/screen`'s record
// holding and moved this one on six entries, every one of them the join screen's
// `máy này — data <digest>`. That is not a defect in the line — the digest is
// what a player reads to a friend before blaming the game, and `-version` prints
// the same figure — so the cost had a reason and the question was only whether a
// screen record needs to carry it. It does not: the line is asserted whole, in
// both languages, by TestTheJoinScreenDrawsThisBinarysOwnVersion, and
// TestTheJoinScreenDrawsTheDigestItHoldsRatherThanOneOfItsOwn holds the half a
// screen reading the real digest cannot give. Both are about the digest; this
// golden is about the drawing.
//
// ⚠️ **A no-op redaction is a failure here rather than a quiet return**, which is
// the same rule bodyOf's assertion follows. If the wording ever draws the digest
// some other way, ReplaceAll matches nothing and the raw digest goes straight back
// into the capture. Three things would then notice, at three different moments,
// and only the first says why:
//
//   - this guard, the moment the wording moves, naming the cause;
//   - the golden comparison, as a mismatch between a raw digest and a redacted
//     record — a diff that says the screen moved, which is not what happened;
//   - TestTheCommittedGoldenNamesNoDataDigest, but only *after* somebody accepts
//     that diff with `make golden` — measured: with this guard deleted and the
//     wording moved, that test is still green until the golden is rewritten.
//
// So the guard is not the only net and is the only clear one.
func withoutTheRunningDataDigest(t *testing.T, body string) string {
	t.Helper()
	local, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read this build's data digest: %v", err)
	}
	short := local.Data.Short()
	if len(short) != len(digestPlaceholder) {
		t.Fatalf("a short digest is %d characters and the stand-in is %d: a redaction of a "+
			"different length moves a clip point", len(short), len(digestPlaceholder))
	}
	if !strings.Contains(body, short) {
		t.Fatalf("no render draws the data digest %s, so this redaction took nothing out. "+
			"Either the join screen stopped drawing it, or the wording changed and this "+
			"golden is about to start moving on every data commit again", short)
	}
	return strings.ReplaceAll(body, short, digestPlaceholder)
}

// TestTheCommittedGoldenNamesNoDataDigest is the anti-churn guard itself, and it
// reads the file on disk rather than the freshly drawn body.
//
// The redaction above runs inside the capture, so a capture that stopped
// redacting would simply record the digest and this file's own diff would be the
// only complaint — a diff a person accepts with `make golden` without reading, on
// a line that looks like every other data-commit churn. This asserts the
// **committed** bytes, which is where the cost was being paid.
func TestTheCommittedGoldenNamesNoDataDigest(t *testing.T) {
	held, err := os.ReadFile(filepath.Join("testdata", "screens.golden"))
	if err != nil {
		t.Fatalf("read the golden: %v", err)
	}
	local, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read this build's data digest: %v", err)
	}
	body := string(held)
	for _, form := range []string{local.Data.Digest.String(), local.Data.Short()} {
		if strings.Contains(body, form) {
			t.Errorf("the committed golden carries this build's data digest %s: it will move "+
				"on the next data commit, which is what withoutTheRunningDataDigest exists "+
				"to stop", form)
		}
	}
	if !strings.Contains(body, digestPlaceholder) {
		t.Error("the committed golden holds no redacted digest at all, so either the join " +
			"screen is not in the record or the redaction stopped running")
	}
}

// startOverARelativeDataDir is startIn over a data directory named by a
// relative path, so no render carries an absolute one.
//
// It is the ordinary fixture — `scratchDataFrom`, so the golden is drawn over
// the same injected cast every sweep here measures — copied into a scratch tree
// that this test then works from. The shipped source is handed in already
// resolved, because this moves the working directory and the second language's
// turn would otherwise look for the books relative to the tree it just made.
func startOverARelativeDataDir(t *testing.T, lang i18n.Lang, shipped string) model {
	t.Helper()
	prepared := scratchDataFrom(t, shipped)
	root := t.TempDir()
	here := filepath.Join(root, "cmd", "hexarena-tui")
	data := filepath.Join(here, relativeDataDir)
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatalf("create %s: %v", data, err)
	}
	copyTree(t, prepared, data)
	t.Chdir(here)
	m, _, _ := startIn(t, lang, relativeDataDir)
	return m
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
		t.Fatalf("%s: the first line is %q, not the header this drops — recording it would "+
			"take a body line instead", where, first)
	}
	return strings.Join(lines[1:], "\n") + "\n"
}

// noAbsolutePath refuses a golden carrying this machine in it.
//
// The two things it looks for are a `/`-rooted path and the OS temp directory,
// and either is a **defect in this file** rather than a fact to accept: a golden
// that names a directory cannot be committed, and one that names a directory
// whose length varies cannot even be reproduced twice on the same machine.
//
// The rooted-path rule is a separator at the front and another one further
// along, which is what tells a path from the bare `/` several footers name as a
// key — the skill listing's filter is bound to it.
func noAbsolutePath(t *testing.T, body string) {
	t.Helper()
	separator := string(filepath.Separator)
	temp := filepath.Clean(os.TempDir())
	for number, line := range strings.Split(body, "\n") {
		if strings.Contains(line, temp) {
			t.Fatalf("line %d of the golden names the temp directory:\n%s", number+1, line)
		}
		for _, word := range strings.Fields(line) {
			// filepath.IsAbs as well as the two-separator test, and the pair is what
			// makes this walk mean the same thing on every platform.
			//
			// The separator test alone is a Windows false positive: the separator
			// there is a backslash and these screens draw hex art out of them, so a
			// cell like \__/..\__ reads as rooted and fails a golden nobody has
			// broken — and because this runs before the -update branch writes, it
			// also made the golden impossible to regenerate on that platform at
			// all. IsAbs answers no to it, since a rooted path with no volume is
			// not absolute on Windows, and yes to C:\Users\..., which is the thing
			// being kept out of a committed file. The two-separator half stays
			// because IsAbs alone would flag a bare /, one cell of the same drawing
			// on a Unix machine.
			if !filepath.IsAbs(word) || !strings.Contains(word[len(separator):], separator) {
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
// `vi | 120x24 | a battle` has already told the reader which screen to look at.
//
// It is the other two goldens' helper of the same name, copied rather than
// shared, for the reason the fixture is: a test helper is not code three suites
// may drift over, and reaching across would edit the files this whole exercise
// exists to stop depending on.
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
