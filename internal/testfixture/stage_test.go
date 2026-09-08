package testfixture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// These are the tests for the staging key and for what the staging leaves
// behind. The ones that matter to a reader of the suites this speeds up are in
// internal/forge, because they need a real library to inject through and this
// package may not import one — internal/forge's own tests use this package, so
// the import back would be a cycle in that test binary.

// TestTheStageKeyCoversTheBooksAndTheFixture is the staleness guard, and it is
// the most important test in this file.
//
// ⚠️ **A staged injection handed out under a key that does not cover its inputs
// is the one failure here that leaves every test in the repository green.** Every
// scratch directory would be built from books that have since moved, and nothing
// would say so — no assertion reads the shipped files directly, they are read
// through the fixture. So the key is asserted to move for each input separately:
// a key that covered only the books would pass an assertion made over an edited
// book alone, and a key that covered only the fixture would pass the other.
//
// *Sees:* a key built from the directory's path, its size, its modification time
// or anything else that a rewritten book need not change; a key that forgot the
// fixture material the injection actually writes.
// *Cannot see:* the injector itself changing, which is why nothing is written to
// disk between runs — see injectOnce.
func TestTheStageKeyCoversTheBooksAndTheFixture(t *testing.T) {
	dir := aDataDir(t)
	base, err := stageKey(dir, "the fixture material")
	if err != nil {
		t.Fatalf("key the data: %v", err)
	}
	again, err := stageKey(dir, "the fixture material")
	if err != nil {
		t.Fatalf("key the data a second time: %v", err)
	}
	if again != base {
		t.Errorf("the same books and fixture keyed as %s and then %s, so nothing could ever "+
			"be reused", base, again)
	}

	moved, err := stageKey(dir, "the fixture material, edited")
	if err != nil {
		t.Fatalf("key an edited fixture: %v", err)
	}
	if moved == base {
		t.Error("editing the fixture material left the key where it was, so a staged " +
			"injection would be handed out for a fixture it was not built from")
	}

	book := filepath.Join(dir, "skills.json")
	if err := os.WriteFile(book, []byte(`{"skills": [], "edited": true}`), 0o644); err != nil {
		t.Fatalf("edit the book: %v", err)
	}
	edited, err := stageKey(dir, "the fixture material")
	if err != nil {
		t.Fatalf("key edited books: %v", err)
	}
	if edited == base {
		t.Error("editing a book left the key where it was, so a staged injection would be " +
			"handed out for books it was not built from")
	}
}

// TestTheStageKeyRefusesADirectoryWithNoBooks is the count every walk in this
// repository owes: a digest over nothing agrees with any other digest over
// nothing, so an unreadable or misspelled data directory has to be an error
// rather than a key.
func TestTheStageKeyRefusesADirectoryWithNoBooks(t *testing.T) {
	if _, err := stageKey(t.TempDir(), "the fixture material"); err == nil {
		t.Error("an empty directory was keyed, so a mistyped data directory would stage " +
			"an empty fixture and every suite would fail somewhere else")
	}
}

// TestAnInjectionLeavesNothingBehind is the answer to what cleans the staging
// up: it cleans itself up, because what survives it is bytes rather than a
// directory.
//
// ⚠️ **The alternative is worth naming, because it was the obvious design and it
// has no correct cleanup.** A staged *directory* handed out to every caller
// cannot be registered with t.Cleanup — t.TempDir and t.Cleanup are per test, so
// the first test to ask would delete it while the rest were still reading it —
// and a TestMain in five packages is a lot of machinery for a quarter of a
// megabyte. Holding the books in memory removes the question.
//
// It is asserted on the failing path because the removal is one deferred
// statement covering both, and a failing injection is the arm a test can reach
// without a library to inject through. internal/forge's
// TestStagingLeavesNothingOnDisk is the same claim on the succeeding path.
func TestAnInjectionLeavesNothingBehind(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { InjectRoot = "" })
	InjectRoot = root

	refused := errors.New("no library here")
	if _, err := injectOnce(aDataDir(t), func(string) (Saver, error) { return nil, refused }); err == nil {
		t.Fatal("the injection succeeded without a library, so nothing here is measured")
	}
	left, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	if len(left) != 0 {
		t.Errorf("%d directories were left in %s, starting with %s: an injection that does not "+
			"clean up after itself leaks one per test binary and nothing deletes it",
			len(left), root, left[0].Name())
	}
}
