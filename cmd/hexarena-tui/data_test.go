package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # Where this client's books come from, and the one place it may guess
//
// `forge.DefaultDataDir` is a **relative** path, so before this the client only
// worked from inside a checkout: installed from the module proxy and run
// anywhere else it died with `read internal/seed/data/combat.json: no such file
// or directory`, on a binary that carries a copy of every one of those files.
// loadLibrary is the whole of the fix and it is three lines; what these tests
// hold is that it is those three lines and not the two-line version that reads
// the same.
//
// ⚠️ **The two-line version is "load, and take the embedded copy if that
// failed", and it is why most of this file exists.** It gets the reproducer
// right and quietly changes what an author's own trailing comma means: instead
// of a refusal naming the book that would not parse, the client plays the
// baked-in copy and says nothing, and the author spends the evening looking for
// an edit they can see in the file. So the arms below are weighted towards the
// directories that *are* there and are *wrong* — a missing directory is one
// test and a broken one is three.

// TestTheClientStartsWithNoDataDirectoryReachableAtAll is the reproducer, and
// it is the whole point of the change: an installed binary, standing in a
// directory that holds nothing.
//
// It goes through parseOptions rather than building an options by hand, because
// what a clean `go install` hands this program is an empty argument list and the
// defaults it settles from one — the thing being measured is that *those*
// defaults reach a library. `t.Chdir` is what makes the relative default miss:
// the working directory is a scratch one with nothing under it, so
// `internal/seed/data` names nothing on any path.
//
// The three assertions after the load are what makes it more than "no error":
// the library is the embedded one (no directory at all), it really holds the
// books, and it answers MatchesEmbeddedData — which is what keeps the join
// screen's "your edits will not reach the battle" line quiet for a player who
// has edited nothing and could not have. → the Edited half of joinScreen.Refresh.
func TestTheClientStartsWithNoDataDirectoryReachableAtAll(t *testing.T) {
	t.Chdir(t.TempDir())

	chosen, err := parseOptions(nil, "", "", io.Discard)
	if err != nil {
		t.Fatalf("parse an empty command line: %v", err)
	}
	// The guard on the fixture: if the default ever stops being a relative path,
	// or something arranges a data directory under the scratch one, this test
	// would be measuring the second line of the rule and not the third.
	if !dataDirectoryIsAbsent(chosen.dir) {
		t.Fatalf("%q exists from the scratch working directory, so this test is not the "+
			"case it is named for", chosen.dir)
	}

	lib, err := loadLibrary(chosen)
	if err != nil {
		t.Fatalf("start with no data directory anywhere: %v", err)
	}
	if where := lib.Dir(); where != "" {
		t.Errorf("the library was read from %q, and there was nothing to read from", where)
	}
	if lib.Characters() == nil || len(lib.Characters().All()) == 0 {
		t.Error("the library holds no cast, so it loaded from nowhere in both senses")
	}
	same, err := lib.MatchesEmbeddedData()
	if err != nil {
		t.Fatalf("compare a library that has no directory with the embedded copy: %v", err)
	}
	if !same {
		t.Error("an embedded library reads as differing from the embedded data, so the join " +
			"screen would tell a player their edits will not reach the battle — over data " +
			"nobody could have edited")
	}
}

// TestADataDirectoryThatIsThereAndBrokenIsRefusedRatherThanQuietlyReplaced is
// the trap guard, and it is the test that has to go red if the rule is ever
// rewritten as "fall back when the load returns an error".
//
// Two arms, because the two ways a real data directory goes wrong take different
// routes through forge.Load and a fallback keyed on the error would swallow both:
//
//   - a book that will not parse, which is the author's own trailing comma;
//   - a book that is not there, inside a directory that is, which is the case
//     that most looks like "absent" and is not — a half-copied tree, or a
//     checkout with a file deleted.
//
// ⚠️ **What each refusal names is asserted, and the two name different things.**
// A missing file is reported by the reader, which wraps the whole path, so the
// message carries `combat.json`. A syntax error is reported by the parser, which
// names the **book** — `decode skill book` — and not the file it came out of, so
// that is what is asserted for the first arm. Asserting a file name there would
// be asserting something internal/forge does not do today, and this step may not
// change internal/forge.
func TestADataDirectoryThatIsThereAndBrokenIsRefusedRatherThanQuietlyReplaced(t *testing.T) {
	for _, arm := range []struct {
		name   string
		break_ func(t *testing.T, dir string)
		names  string
	}{
		{
			name: "a book that will not parse",
			break_: func(t *testing.T, dir string) {
				t.Helper()
				at := filepath.Join(dir, "skills.json")
				raw, err := os.ReadFile(at)
				if err != nil {
					t.Fatalf("read %s: %v", at, err)
				}
				// A trailing comma is the mistake this arm is about, so the
				// break is one: the file is otherwise the file that loaded a
				// moment ago.
				if err := os.WriteFile(at, append([]byte("[,"), raw...), 0o644); err != nil {
					t.Fatalf("write a broken %s: %v", at, err)
				}
			},
			names: "skill book",
		},
		{
			name: "a book that is not there",
			break_: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Remove(filepath.Join(dir, "combat.json")); err != nil {
					t.Fatalf("take the combat rules away: %v", err)
				}
			},
			names: "combat.json",
		},
	} {
		t.Run(arm.name, func(t *testing.T) {
			dir := scratchData(t)
			// The anti-vacuity half: the directory loaded before it was broken,
			// so the refusal below is the break and not the fixture.
			if _, err := forge.Load(dir); err != nil {
				t.Fatalf("the scratch directory does not load before it is broken: %v", err)
			}
			arm.break_(t, dir)

			// Both readings, because both are ways a player reaches this
			// directory and only one of them goes past the "was it given" test.
			for _, given := range []bool{true, false} {
				lib, err := loadLibrary(options{dir: dir, dataGiven: given, lang: i18n.En})
				if err == nil {
					t.Fatalf("a broken data directory loaded (dataGiven=%t) and answered a "+
						"library from %q — the rule has been rewritten to fall back on the "+
						"error, which swallows an author's own mistake", given, lib.Dir())
				}
				if !strings.Contains(err.Error(), arm.names) {
					t.Errorf("the refusal (dataGiven=%t) does not name %s: %v",
						given, arm.names, err)
				}
			}
		})
	}
}

// TestADataDirectoryPathBlockedByAFileIsRefusedRatherThanQuietlyReplaced is the
// same "refused rather than quietly replaced" rule one level down: not the load
// failing, but the **probe** failing.
//
// ⚠️ **`dataDirectoryIsAbsent` asks fs.ErrNotExist and not "did the stat
// return an error", and without this test nothing said so.** The two read alike
// and are different programs: a stat can fail because the path is genuinely not
// there, or because it cannot be resolved at all — and only the first is a
// missing data directory. Widening the probe to `err != nil` compiles, passes
// every other test in this package, and turns the one error that names the real
// problem into a silent game on the embedded books.
//
// The case is built without permission tricks, which matters because a test that
// needs an unwritable directory measures the machine it runs on: a plain **file**
// where a path component should be a directory makes every stat below it fail
// with ENOTDIR, which is not fs.ErrNotExist. So `internal/seed` is written as a
// file and `internal/seed/data` — the default this client would look under —
// becomes unstattable while being neither present nor absent.
//
// ⚠️ **The path is asserted through filepath.FromSlash and no book is named.**
// forge.DefaultDataDir is a slash-separated constant and the reader joins with
// the platform separator, so a literal would be a test that only passes off
// Windows; and naming `combat.json` would tie this to which book loadBooks reads
// first, which is not what is being measured. What must hold is that the refusal
// is about *this path*.
//
// ⚠️ **One arm, dataGiven false, deliberately.** The probe is only consulted on
// that arm — `loadLibrary` short-circuits on `chosen.dataGiven` — so a given arm
// would refuse via the first line of the rule, which
// TestANamedDataDirectoryThatIsNotThereIsRefused already holds, and it would
// stay green under the widening this test exists to catch. A row that cannot go
// red for the reason its test is named for makes the suite look stronger than it
// is.
func TestADataDirectoryPathBlockedByAFileIsRefusedRatherThanQuietlyReplaced(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Dir(filepath.Join(root, forge.DefaultDataDir))
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatalf("create the tree above %s: %v", blocked, err)
	}
	if err := os.WriteFile(blocked, []byte("a file where a directory should be\n"), 0o644); err != nil {
		t.Fatalf("write a file at %s: %v", blocked, err)
	}
	t.Chdir(root)

	chosen, err := parseOptions(nil, "", "", io.Discard)
	if err != nil {
		t.Fatalf("parse an empty command line: %v", err)
	}
	// The fixture's own guard, and it is what makes the rest of this test the
	// case it is named for: the path must be unstattable **without** being
	// absent. If either half stops holding, this is measuring the reproducer or
	// the ordinary directory, and would pass for the wrong reason.
	_, statErr := os.Stat(chosen.dir)
	if statErr == nil {
		t.Fatalf("%q stats cleanly, so nothing is blocking it and this test measures "+
			"nothing", chosen.dir)
	}
	if errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("%q reads as absent (%v), so this is the reproducer's case and not the "+
			"unstattable one", chosen.dir, statErr)
	}
	// ⚠️ An Error and not a Fatal, so the behavioural half below still runs. A
	// widened probe is only half the finding — what it *causes* is a library
	// coming back, and a failure that stops here would report the mechanism
	// while leaving the consequence unmeasured.
	if dataDirectoryIsAbsent(chosen.dir) {
		t.Error("the probe calls a blocked path absent, so the fallback is keyed on the " +
			"stat failing rather than on the directory being gone — an error naming the " +
			"real problem is about to be answered with the embedded copy")
	}

	lib, err := loadLibrary(chosen)
	// Both halves are asserted, and the library is the one that matters: "some
	// error happened" would also be true of a fallback that then failed for its
	// own reasons, while a library coming back at all is the fallback having
	// happened.
	if lib != nil {
		t.Errorf("a blocked data directory answered a library, read from %q — the embedded "+
			"copy has been played over a path that could not be resolved", lib.Dir())
	}
	if err == nil {
		t.Fatal("a blocked data directory loaded without complaint")
	}
	if want := filepath.FromSlash(chosen.dir); !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal is not about %s, so the reader never reached the path the "+
			"player would have to fix: %v", want, err)
	}
}

// TestANamedDataDirectoryThatIsNotThereIsRefused is the first line of the rule:
// a directory the player typed is never second-guessed.
//
// Answering `--data /nope` with different data is not an answer to what was
// asked — the player named a directory because they wanted that one, and a
// client that silently played the baked-in copy instead would be hiding a typo
// in the one place a typo is easy to make.
//
// ⚠️ The **same** absent path is loaded both ways in one test, which is what
// makes it about the flag rather than about the path: not given it is the
// reproducer above and answers a library, given it refuses. A test that only
// asserted the refusal would pass with `dataGiven` deleted and the whole
// fallback removed.
//
// ⚠️ **The "given" half goes through parseOptions and not through an options
// built here**, which is the arm a hand-built literal cannot reach: every other
// dataGiven in this file is written by the test, so a wasSet that answered false
// for everything would leave all of them green and ship a client that ignores
// the flag. This is the one place the flag is really typed.
func TestANamedDataDirectoryThatIsNotThereIsRefused(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "no-such-directory")

	typed, err := parseOptions([]string{"-data", absent}, "", "", io.Discard)
	if err != nil {
		t.Fatalf("parse -data %s: %v", absent, err)
	}
	if !typed.dataGiven {
		t.Fatal("-data was typed and options says it was not, so every refusal below is " +
			"about a directory nobody named")
	}
	if typed.dir != absent {
		t.Fatalf("-data was parsed as %q, want %q", typed.dir, absent)
	}
	if _, err := loadLibrary(typed); err == nil {
		t.Error("a typed --data that is not there answered a library")
	}
	// And the default left alone is the other reading of the same field, so the
	// two are told apart by what was typed rather than by what the string is.
	untouched, err := parseOptions(nil, "", "", io.Discard)
	if err != nil {
		t.Fatalf("parse an empty command line: %v", err)
	}
	if untouched.dataGiven {
		t.Error("a run with no --data reads as having named a directory, so the flag's " +
			"default would be refused like a typed path")
	}

	if _, err := loadLibrary(options{dir: absent, dataGiven: true, lang: i18n.En}); err == nil {
		t.Fatal("--data named a directory that is not there and the client played something " +
			"else, which is not an answer to what was asked")
	} else if !strings.Contains(err.Error(), absent) {
		t.Errorf("the refusal does not name the directory that was asked for: %v", err)
	}

	unnamed, err := loadLibrary(options{dir: absent, lang: i18n.En})
	if err != nil {
		t.Fatalf("the same path nobody named refused as well, so the refusal above is about "+
			"the path rather than about the flag: %v", err)
	}
	if where := unnamed.Dir(); where != "" {
		t.Errorf("a path nobody named answered a library from %q", where)
	}
}

// TestTheDirectoryUnderfootIsPreferredToTheEmbeddedCopyWhenNobodyNamedOne is the
// middle line of the rule, and the reason it is not "always embed".
//
// `make play-tui` passes no --data and is run from the module root by an author
// who has just edited a file and wants to see it in the cast browser. Embedding
// regardless would take that away without saying so — the edit would be there in
// the file, absent from every screen, and nothing on the screen would explain it.
//
// ⚠️ **The assertion is a character the embedded copy does not have**, which is
// what stops this passing by coincidence: both libraries hold a cast, so
// "the cast loaded" is true whichever one answered. The fixture cast injected
// into a scratch directory is the difference, and its absence from the embedded
// copy is asserted rather than assumed — a fixture that ever shipped would turn
// this test green for the wrong reason without a word.
func TestTheDirectoryUnderfootIsPreferredToTheEmbeddedCopyWhenNobodyNamedOne(t *testing.T) {
	const injected = "fixture-anime.adept"

	embedded, err := forge.LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	if _, shipped := embedded.Characters().Get(injected); shipped {
		t.Fatalf("%s is in the shipped cast now, so it cannot tell the two libraries "+
			"apart and this test measures nothing", injected)
	}

	// The scratch data goes under the **default** directory name, from a working
	// directory of this test's own: the whole question is what the client does
	// with the path it was not given, so the path has to be that one.
	prepared := scratchDataFrom(t, mustAbs(t, shippedDataDir))
	root := t.TempDir()
	under := filepath.Join(root, forge.DefaultDataDir)
	if err := os.MkdirAll(under, 0o755); err != nil {
		t.Fatalf("create %s: %v", under, err)
	}
	copyTree(t, prepared, under)
	t.Chdir(root)

	chosen, err := parseOptions(nil, "", "", io.Discard)
	if err != nil {
		t.Fatalf("parse an empty command line: %v", err)
	}
	lib, err := loadLibrary(chosen)
	if err != nil {
		t.Fatalf("load the directory underfoot: %v", err)
	}
	if lib.Dir() != forge.DefaultDataDir {
		t.Errorf("the library was read from %q, want the default directory %q",
			lib.Dir(), forge.DefaultDataDir)
	}
	if _, known := lib.Characters().Get(injected); !known {
		t.Errorf("the cast does not hold %s, so the client read the embedded copy over "+
			"the directory it was standing in", injected)
	}
}

// mustAbs resolves a path before a test moves the working directory out from
// under it, which is the trap scratchDataFrom exists for.
func mustAbs(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve %s: %v", path, err)
	}
	return resolved
}

// TestTheArtPreviewOverAnEmbeddedLibraryDoesNotAccuseThePlayer is what the one
// screen that reads a **file** says when there is no directory to read one from,
// driven through this client's own construction.
//
// ⚠️ **It replaces TestTheArtPreviewOverAnEmbeddedLibrarySaysThePictureIsMissing,
// which recorded the opposite claim and said in its own comment that the art
// step was free to move it.** What that test froze was the honest state of the
// program at the time: the art is not embedded — sixty-six files and 16 MB — so
// an embedded library answers the empty path for every picture, os.Stat("")
// fails, and the preview drew `i18n.ArtMissing` in the bad style. For an author
// that word is right and somebody should go and draw the file. For a player who
// installed the binary it is false in every part: nothing is missing, their
// setup is not broken, and there is no directory they could put a picture in.
//
// This is the client's half of the pair. The wording decision itself, and the
// arm that keeps the author's MISSING where it belongs, are in
// internal/screen/nodirectory_test.go — a package test can build both libraries
// side by side, and what only this file can add is that the whole path holds
// through `m.enter`: the browser and the raise, drawn inside `frame`.
//
// *Sees:* a player accused; the raise breaking; the preview reduced to an error
// page. *Cannot see:* the author's state, which this client cannot be in.
func TestTheArtPreviewOverAnEmbeddedLibraryDoesNotAccuseThePlayer(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	if lib.HasDataDirectory() {
		t.Fatal("the embedded library reports a data directory, so this test measures " +
			"the ordinary preview")
	}
	m := newModel(lib, i18n.En, newSession(), nil)
	m.width, m.height = 120, 44

	browser := m.enter(screenCast)
	// The cast browser is on the way to the preview and carries the same verdict
	// on a row of its own, so it is asserted here rather than being passed
	// through: a player meets it one keystroke earlier than the preview.
	if said := browser.text(i18n.ArtMissing); strings.Contains(drawnBody(browser), said) {
		t.Errorf("the cast browser over an embedded library says %q at a player:\n%s",
			said, drawnBody(browser))
	}
	if want := browser.text(i18n.ArtNotShipped); !strings.Contains(drawnBody(browser), want) {
		t.Errorf("the cast browser over an embedded library does not say %q:\n%s",
			want, drawnBody(browser))
	}

	preview := raisedFrom(t, browser, "p", screenPreview)
	drawn := drawnBody(preview)
	if want := preview.text(i18n.PreviewArtNotShipped); !strings.Contains(drawn, want) {
		t.Errorf("the preview over an embedded library does not say the pictures are "+
			"not carried:\n%s", drawn)
	}
	if said := preview.text(i18n.ArtMissing); strings.Contains(drawn, said) {
		t.Errorf("the preview over an embedded library still says %q, which is the "+
			"author's word for a file somebody forgot to draw:\n%s", said, drawn)
	}
	// The character is still named, so this is the screen drawn over a library
	// that works rather than an error page.
	if subject := preview.cast.Subject(); !strings.Contains(drawn, subject.ID) {
		t.Errorf("the preview does not name %s, so it is not the screen it should be:\n%s",
			subject.ID, drawn)
	}
	if _, err := lib.ArtFiles(); !errors.Is(err, forge.ErrNoDataDirectory) {
		t.Errorf("an embedded library offered an art list rather than refusing: %v", err)
	}
}

// TestTheHeaderCarriesNoTrailingSpaceWithNoDataDirectory is the header's half of
// the same state.
//
// `frame` used to write `programName + Dim("  " + lib.Dir())` unconditionally,
// so a client with no directory drew the program's name with two spaces welded
// to the end of it — on every screen, for the whole run. Nothing shows on a
// terminal, which is why it lasted: what it costs is a trailing run on the first
// line of every copy anybody pastes out of one, and a first line no assertion
// can compare without trimming it first.
//
// ⚠️ **The assertion is on the rendered line and not on the expression**, which
// is the difference between measuring the fix and restating it: the separator,
// the styling and the clip all sit between `Dir()` and what a reader sees, and
// only the rendered line has all three in it.
//
// ⚠️ **The control is the whole test.** "No trailing space" is satisfied by a
// header that has stopped naming the directory at all, which would be a
// regression wearing this test's clothes — so the second half runs the same
// window over a real directory and insists the name and its separator are
// both there.
func TestTheHeaderCarriesNoTrailingSpaceWithNoDataDirectory(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	embedded, err := forge.LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	m := newModel(embedded, i18n.Vi, newSession(), nil)
	m.width, m.height = 120, 44
	header := firstLineOf(m.screenContent())
	if header != programName {
		t.Errorf("the header with no data directory is %q, want exactly %q — "+
			"%d trailing space(s)",
			header, programName, len(header)-len(strings.TrimRight(header, " ")))
	}

	// The same header over a directory, which is what says the line above is a
	// separator that went away with its subject rather than a header that lost
	// its subject.
	dir := scratchData(t)
	onDisk, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	withDir := newModel(onDisk, i18n.Vi, newSession(), nil)
	withDir.width, withDir.height = 400, 44
	named := firstLineOf(withDir.screenContent())
	if want := programName + "  " + dir; named != want {
		t.Errorf("the header over a directory is %q, want %q", named, want)
	}
}

// firstLineOf is a drawn screen's header row, kept whole — trailing run and all,
// which is the only reason this is not strings.Cut written inline.
func firstLineOf(drawn string) string {
	line, _, _ := strings.Cut(drawn, "\n")
	return line
}
