package testfixture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These are the tests for the sharing mechanism itself. The tests that matter to
// a reader of the suites this speeds up are elsewhere — each client's own
// fixture asserts that its scratch directory really is sharing — because a
// mechanism that works and a caller that uses it are two different claims, and
// the second is the one a refactor breaks.

// stamped is a past moment, so "the modification time was preserved" is a
// statement about the source rather than about how fast a test runs.
var stamped = time.Date(2021, time.March, 4, 5, 6, 7, 0, time.UTC)

// aDataDir builds the shape of a data directory: a book, a picture beside it and
// a picture one folder deeper, which is how the fixture's own art is filed.
func aDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skills.json"), []byte(`{"skills": []}`), 0o644); err != nil {
		t.Fatalf("write the book: %v", err)
	}
	nested := filepath.Join(dir, ArtDir, "fixture")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("make %s: %v", nested, err)
	}
	for _, name := range []string{filepath.Join(dir, ArtDir, "hero.svg"), filepath.Join(nested, "adept.svg")} {
		if err := os.WriteFile(name, []byte(Art), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if err := os.Chtimes(name, stamped, stamped); err != nil {
			t.Fatalf("stamp %s: %v", name, err)
		}
	}
	return dir
}

// sameFile says whether two paths are one file, which is what every form of
// sharing here comes to and what a copy can never be.
func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	first, err := os.Stat(a)
	if err != nil {
		t.Fatalf("stat %s: %v", a, err)
	}
	second, err := os.Stat(b)
	if err != nil {
		t.Fatalf("stat %s: %v", b, err)
	}
	return os.SameFile(first, second)
}

// TestCopyDataSharesTheArtAndCopiesTheBooks is the whole arrangement in one
// test: the pictures are the same file, the books are not.
//
// Both halves are asserted because either alone has a wrong answer that passes.
// Sharing everything would be fast and would let a test that writes a skill
// write into the directory it was copied from; copying everything is what this
// replaced.
func TestCopyDataSharesTheArtAndCopiesTheBooks(t *testing.T) {
	source := aDataDir(t)
	target := t.TempDir()
	done, err := CopyData(source, target)
	if err != nil {
		t.Fatalf("copy the data: %v", err)
	}

	for _, image := range []string{"hero.svg", filepath.Join("fixture", "adept.svg")} {
		from := filepath.Join(source, ArtDir, image)
		to := filepath.Join(target, ArtDir, image)
		if !sameFile(t, from, to) {
			t.Errorf("%s was copied rather than shared", image)
		}
	}
	if sameFile(t, filepath.Join(source, "skills.json"), filepath.Join(target, "skills.json")) {
		t.Error("the book is shared: a test that writes a skill would write into the source")
	}

	if done.Shared() != 2 || done.Copied != 1 {
		t.Errorf("shared %d and copied %d, want 2 pictures shared and 1 book copied (%+v)",
			done.Shared(), done.Copied, done)
	}
	if done.BytesCopied != int64(len(`{"skills": []}`)) {
		t.Errorf("%d bytes were written, want only the book's %d",
			done.BytesCopied, len(`{"skills": []}`))
	}
}

// TestASharedPictureKeepsItsSizeAndModificationTime is the property the art
// preview's cache reads. It keys on those two and nothing else, so a mechanism
// that changed either would change what TestThePreviewRasterisesOncePerFileAndSize
// measures without that test being able to say so.
func TestASharedPictureKeepsItsSizeAndModificationTime(t *testing.T) {
	source := aDataDir(t)
	target := t.TempDir()
	if _, err := CopyData(source, target); err != nil {
		t.Fatalf("copy the data: %v", err)
	}
	shared, err := os.Stat(filepath.Join(target, ArtDir, "hero.svg"))
	if err != nil {
		t.Fatalf("stat the shared picture: %v", err)
	}
	if shared.Size() != int64(len(Art)) {
		t.Errorf("the shared picture is %d bytes, want %d", shared.Size(), len(Art))
	}
	if !shared.ModTime().Equal(stamped) {
		t.Errorf("the shared picture is stamped %v, want the source's %v", shared.ModTime(), stamped)
	}
}

// TestCopyDataFallsBackToASymlinkWhenAHardLinkIsRefused is the filesystem
// answer: a hard link cannot cross one, and a temporary directory is free to be
// on a different one from the repository.
func TestCopyDataFallsBackToASymlinkWhenAHardLinkIsRefused(t *testing.T) {
	refuseLinks(t, true, false)
	source := aDataDir(t)
	target := t.TempDir()
	done, err := CopyData(source, target)
	if err != nil {
		t.Fatalf("copy the data: %v", err)
	}
	if done.Symlinked != 2 || done.Linked != 0 {
		t.Errorf("%d symlinked and %d linked, want both pictures symlinked (%+v)",
			done.Symlinked, done.Linked, done)
	}
	picture := filepath.Join(target, ArtDir, "hero.svg")
	entry, err := os.Lstat(picture)
	if err != nil {
		t.Fatalf("lstat the picture: %v", err)
	}
	if entry.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the picture is %v, want a symbolic link", entry.Mode())
	}
	if !sameFile(t, filepath.Join(source, ArtDir, "hero.svg"), picture) {
		t.Error("the symbolic link does not resolve to the picture it was made for")
	}
}

// TestCopyDataFallsBackToACopyWhenNeitherLinkIsAvailable is the last resort, and
// the one that has to be exactly as correct as the fast path. A machine where
// both refusals happen — Windows without developer mode, across two volumes --
// gets the old speed and must not get a different answer.
//
// The modification time is what makes that more than a byte comparison: the
// preview's cache keys on it, so a copy that took today's stamp would be a
// fallback that quietly changed a cache key.
func TestCopyDataFallsBackToACopyWhenNeitherLinkIsAvailable(t *testing.T) {
	refuseLinks(t, true, true)
	source := aDataDir(t)
	target := t.TempDir()
	done, err := CopyData(source, target)
	if err != nil {
		t.Fatalf("copy the data: %v", err)
	}
	if done.Shared() != 0 || done.Copied != 3 {
		t.Errorf("shared %d and copied %d, want nothing shared and all three written (%+v)",
			done.Shared(), done.Copied, done)
	}
	picture := filepath.Join(target, ArtDir, "hero.svg")
	if sameFile(t, filepath.Join(source, ArtDir, "hero.svg"), picture) {
		t.Fatal("the picture is still shared, so this measures the fast path")
	}
	raw, err := os.ReadFile(picture)
	if err != nil {
		t.Fatalf("read the copied picture: %v", err)
	}
	if string(raw) != Art {
		t.Error("the copied picture does not hold the bytes it was copied from")
	}
	copied, err := os.Stat(picture)
	if err != nil {
		t.Fatalf("stat the copied picture: %v", err)
	}
	if !copied.ModTime().Equal(stamped) {
		t.Errorf("the copy is stamped %v, want the source's %v: the preview's cache reads that",
			copied.ModTime(), stamped)
	}
}

// TestAWriteThroughAShareIsNamedRatherThanSilent is the guard on the one way
// this arrangement could damage the repository.
//
// A shared picture is the same file, so a test that writes to art it was handed
// edits the committed one. Nothing does today and PrivateArt is how a test says
// it is about to — but "nothing does today" is not a property, so the next
// scratch directory to ask for the same picture refuses and names it.
func TestAWriteThroughAShareIsNamedRatherThanSilent(t *testing.T) {
	guardEverySource(t)
	source := aDataDir(t)
	if _, err := CopyData(source, t.TempDir()); err != nil {
		t.Fatalf("the first copy: %v", err)
	}

	// A write through a share, which is what a test rewriting its art would do.
	picture := filepath.Join(source, ArtDir, "hero.svg")
	if err := os.WriteFile(picture, []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write through the share: %v", err)
	}

	_, err := CopyData(source, t.TempDir())
	if err == nil {
		t.Fatal("the second copy said nothing, so a rewritten picture would go unnoticed")
	}
	if !strings.Contains(err.Error(), "hero.svg") || !strings.Contains(err.Error(), "PrivateArt") {
		t.Errorf("the refusal is %q; it has to name the file and what to do instead", err)
	}
}

// TestPrivateArtTakesOnePictureOutOfTheShare is the door the guard points at.
func TestPrivateArtTakesOnePictureOutOfTheShare(t *testing.T) {
	source := aDataDir(t)
	target := t.TempDir()
	if _, err := CopyData(source, target); err != nil {
		t.Fatalf("copy the data: %v", err)
	}
	image := ArtDir + "/hero.svg"
	if err := PrivateArt(target, image); err != nil {
		t.Fatalf("take a private copy: %v", err)
	}
	from, to := filepath.Join(source, ArtDir, "hero.svg"), filepath.Join(target, ArtDir, "hero.svg")
	if sameFile(t, from, to) {
		t.Fatal("the picture is still shared")
	}
	if err := os.WriteFile(to, []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write the private copy: %v", err)
	}
	raw, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("read the source: %v", err)
	}
	if string(raw) != Art {
		t.Error("writing the private copy reached the picture it was copied from")
	}
}

// refuseLinks makes one or both of the sharing calls fail, and puts them back.
func refuseLinks(t *testing.T, hard, soft bool) {
	t.Helper()
	link, symlink := linkFile, symlinkFile
	t.Cleanup(func() { linkFile, symlinkFile = link, symlink })
	if hard {
		linkFile = func(string, string) error { return errors.New("cross-device link") }
	}
	if soft {
		symlinkFile = func(string, string) error { return errors.New("a required privilege is not held") }
	}
}

// guardEverySource turns off the "a temporary directory is itself scratch"
// exemption, since everything a test can build is in one.
func guardEverySource(t *testing.T) {
	t.Helper()
	root := scratchRoot
	t.Cleanup(func() { scratchRoot = root })
	scratchRoot = filepath.Join(root, "a-folder-no-test-writes-in")
}
