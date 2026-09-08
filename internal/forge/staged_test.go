package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vukyn/hexarena/internal/testfixture"
)

// These are the tests for the staged injection, and they live here rather than
// in internal/testfixture because they need a real library to inject through:
// that package may not import this one, since this package's own tests import
// it. Its stage_test.go holds the half it can reach on its own — the key, and
// what an injection leaves behind.

// TestAScratchDataDirectoryHoldsTheInjectedBooks is "did the staging stage the
// right thing".
//
// ⚠️ **The whole point of building the injection once is that a scratch
// directory is no longer built by running it**, so the thing that could now
// silently go missing is the injection itself: a stage that wrote out the
// shipped books untouched would leave every test that names a fixture character
// failing somewhere far away, with a message about an unknown id rather than
// about the fixture. This says it here.
//
// The three ids are derived from the constants rather than written down, so a
// fixture that grows a character is covered without this being edited, and the
// counts are asserted for the reason every walk in this repository asserts one:
// a list that came back empty agrees with any assertion made over it.
func TestAScratchDataDirectoryHoldsTheInjectedBooks(t *testing.T) {
	library, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load a scratch data directory: %v", err)
	}

	var origins []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(testfixture.Origins), &origins); err != nil {
		t.Fatalf("read the fixture origins: %v", err)
	}
	if len(origins) == 0 {
		t.Fatal("the fixture declares no origins, so nothing here is measured")
	}
	for _, origin := range origins {
		if _, ok := library.Origins().Get(origin.ID); !ok {
			t.Errorf("the origin %s is not in the scratch directory's book", origin.ID)
		}
	}

	var characters []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(testfixture.Characters), &characters); err != nil {
		t.Fatalf("read the fixture characters: %v", err)
	}
	if len(characters) == 0 {
		t.Fatal("the fixture declares no characters, so nothing here is measured")
	}
	for _, character := range characters {
		if _, ok := library.Characters().Get(character.ID); !ok {
			t.Errorf("the character %s is not in the scratch directory's book", character.ID)
		}
	}

	var skills []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(testfixture.Skills), &skills); err != nil {
		t.Fatalf("read the fixture skills: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("the fixture declares no skills, so nothing here is measured")
	}
	for _, held := range skills {
		if _, err := library.Skills().Lookup(held.ID); err != nil {
			t.Errorf("the skill %s is not in the scratch directory's book: %v", held.ID, err)
		}
	}

	// And the presets, which are the one book no library method writes and are
	// therefore merged into the file instead. They are the arm a staging that
	// only re-ran the savers would miss.
	var presets []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(testfixture.Archetypes), &presets); err != nil {
		t.Fatalf("read the fixture presets: %v", err)
	}
	if len(presets) == 0 {
		t.Fatal("the fixture declares no presets, so nothing here is measured")
	}
	for _, preset := range presets {
		if _, ok := library.Archetypes().Get(preset.ID); !ok {
			t.Errorf("the preset %s is not in the scratch directory's book", preset.ID)
		}
	}
}

// TestAScratchDataDirectoryStillSharesRatherThanCopies is the arithmetic, read
// off what Data reports rather than off a stopwatch.
//
// ⚠️ **The books now come from memory and the art from the shipped directory,
// which are two different sources, and only one of them may ever be written
// out.** A change that started copying the pictures instead of linking them
// would leave every test in the repository green and put seventeen megabytes
// back on every scratch directory — the same failure CopiedArt names, said here
// as a number, because CopiedArt compares inodes and cannot see how much was
// written.
func TestAScratchDataDirectoryStillSharesRatherThanCopies(t *testing.T) {
	done, err := testfixture.Data(t.TempDir(), shippedDataDir, load)
	if err != nil {
		t.Fatalf("build a scratch data directory: %v", err)
	}
	if done.Shared() == 0 {
		t.Error("no picture was shared, so every one of them was written out")
	}
	if done.Copied == 0 {
		t.Error("nothing was written, so this scratch directory has no books in it")
	}
	// The shipped books are about 280 KB and the art is about 16 MB. A megabyte
	// is comfortably above the one and far below the other, so this cannot be
	// tuned past by a book growing and cannot be passed by a picture being copied.
	const roomForTheBooksAndNoPictures = 1 << 20
	if done.BytesCopied > roomForTheBooksAndNoPictures {
		t.Errorf("%d bytes were written for one scratch directory, want under %d: that is "+
			"the art being copied rather than shared", done.BytesCopied, roomForTheBooksAndNoPictures)
	}
}

// TestTheFixtureIsStagedOncePerProcess is the arrangement this whole change is
// for, asserted as a count rather than as a stopwatch.
//
// A clock cannot say it: the injection is about a fifth of a second and a
// scratch directory built without staging still passes every other test in the
// repository, only slower — which is exactly the failure that leaves a suite
// green and back at its old cost. So the injections are counted.
//
// The first call is made before the count is taken, because whether this test
// runs first in the binary is not something it should depend on.
func TestTheFixtureIsStagedOncePerProcess(t *testing.T) {
	scratchData(t)
	staged := testfixture.Stagings()
	if staged == 0 {
		t.Fatal("the fixture has never been injected, so the counter is measuring nothing")
	}
	for range 3 {
		scratchData(t)
	}
	if again := testfixture.Stagings(); again != staged {
		t.Errorf("three more scratch directories ran %d more injections, want 0: the "+
			"injection is being repeated per directory, which is what staging it removed",
			again-staged)
	}
}

// TestTheFixturesOwnArtIsPrivateToEachScratchDirectory is the half of the
// arrangement the staging could have taken away.
//
// ⚠️ **The shipped pictures are shared and the fixture's are not, and the
// difference is deliberate.** A scratch directory's shipped art is a hard link
// to a committed file, which is why testfixture.PrivateArt exists and why
// TestThePreviewRasterisesOncePerFileAndSize calls it. The fixture's own three
// pictures are written out per directory instead, so the one test that rewrites
// the art it was handed can never be reaching a picture another test is also
// looking at. Staging them alongside the books would have been cheap and would
// have made that quietly untrue.
func TestTheFixturesOwnArtIsPrivateToEachScratchDirectory(t *testing.T) {
	first, second := scratchData(t), scratchData(t)
	if len(testfixture.FixtureArt) == 0 {
		t.Fatal("the fixture names no art, so nothing here is measured")
	}
	for _, image := range testfixture.FixtureArt {
		here := filepath.Join(first, testfixture.ArtDir, filepath.FromSlash(image))
		there := filepath.Join(second, testfixture.ArtDir, filepath.FromSlash(image))
		hereInfo, err := os.Stat(here)
		if err != nil {
			t.Fatalf("stat %s: %v", here, err)
		}
		thereInfo, err := os.Stat(there)
		if err != nil {
			t.Fatalf("stat %s: %v", there, err)
		}
		if os.SameFile(hereInfo, thereInfo) {
			t.Errorf("%s is one file in both scratch directories, so a test rewriting it "+
				"would be rewriting another test's picture", image)
		}
	}
}

// TestEditedBooksReachTheNextScratchDirectory is the staleness guard end to end,
// and the failure it names is the one this arrangement could produce that
// nothing else in the repository would notice.
//
// ⚠️ **A stage handed out under a key that does not cover the books it was built
// from would make every scratch directory in the suite hold old data, with every
// test still green.** internal/testfixture's TestTheStageKeyCoversTheBooksAndTheFixture
// holds the key itself; this holds the thing the key is for — that the key is
// actually consulted, over a directory whose books have moved under it.
//
// The shipped books are copied somewhere private first: editing the committed
// ones to make a point is exactly the damage the sharing guard exists to catch.
func TestEditedBooksReachTheNextScratchDirectory(t *testing.T) {
	shipped := t.TempDir()
	copyTree(t, shippedDataDir, shipped)

	before := t.TempDir()
	if _, err := testfixture.Data(before, shipped, load); err != nil {
		t.Fatalf("build a scratch directory: %v", err)
	}

	const marker = "edited while the tests were running"
	edited := editTheFirstOriginsNote(t, shipped, marker)

	after := t.TempDir()
	if _, err := testfixture.Data(after, shipped, load); err != nil {
		t.Fatalf("build a second scratch directory: %v", err)
	}

	was := noteOfOrigin(t, before, edited)
	if was == marker {
		t.Fatalf("the first scratch directory already carried the edit for %s, so this test "+
			"could not tell a stale stage from a fresh one", edited)
	}
	if now := noteOfOrigin(t, after, edited); now != marker {
		t.Errorf("%s reads %q in a scratch directory built after the books were edited, want "+
			"%q: the staged injection is being handed out for books it was not built from",
			edited, now, marker)
	}
}

// TestStagingLeavesNothingOnDisk is the succeeding arm of the claim
// internal/testfixture's TestAnInjectionLeavesNothingBehind makes on the failing
// one: an injection works in a directory of its own and removes it, so what
// answers "who cleans the staging up" is that there is nothing to clean.
//
// A shipped directory of its own is used so that a fresh stage is forced —
// against the real one the stage is already built by the time this runs, and a
// test that measures a cache hit measures nothing.
func TestStagingLeavesNothingOnDisk(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { testfixture.InjectRoot = "" })
	testfixture.InjectRoot = root

	shipped := t.TempDir()
	copyTree(t, shippedDataDir, shipped)
	editTheFirstOriginsNote(t, shipped, "so this directory is not the one already staged")

	staged := testfixture.Stagings()
	if _, err := testfixture.Data(t.TempDir(), shipped, load); err != nil {
		t.Fatalf("build a scratch directory: %v", err)
	}
	if testfixture.Stagings() != staged+1 {
		t.Fatal("no injection ran, so nothing here is measured")
	}
	left, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	if len(left) != 0 {
		t.Errorf("%d directories were left in %s, starting with %s: an injection that keeps "+
			"its working directory leaves one per test binary, and nothing deletes it",
			len(left), root, left[0].Name())
	}
}

// load is the reload the staged injection writes through.
func load(dir string) (testfixture.Saver, error) { return Load(dir) }

// editTheFirstOriginsNote rewrites one free-text field in a data directory's
// origins and says which entry it moved.
//
// A note is chosen because it is the one field nothing validates: an edit that
// has to stay legal is an edit that can quietly stop being an edit.
func editTheFirstOriginsNote(t *testing.T, dir, note string) string {
	t.Helper()
	name := filepath.Join(dir, "origins.json")
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var book struct {
		Origins []map[string]any `json:"origins"`
	}
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	if len(book.Origins) == 0 {
		t.Fatalf("%s declares no origins, so there is nothing to edit", name)
	}
	book.Origins[0]["note"] = note
	id, _ := book.Origins[0]["id"].(string)
	if id == "" {
		t.Fatalf("the first origin in %s has no id", name)
	}
	out, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	if err := os.WriteFile(name, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return id
}

// noteOfOrigin reads one origin's note out of a data directory.
func noteOfOrigin(t *testing.T, dir, id string) string {
	t.Helper()
	library, err := Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	origin, ok := library.Origins().Get(id)
	if !ok {
		t.Fatalf("%s holds no origin %s", dir, id)
	}
	return origin.Note
}
