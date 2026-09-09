package testfixture

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// This file is the guard five packages make about the sharing in data.go, and
// the reason it is one file rather than five copies of an assertion is the same
// reason CopyData is one function rather than five copyTrees: a rule written out
// five times drifts, and a guard that drifted is one that is off in a package
// nobody looked at.
//
// ⚠️ **The guard is about the MECHANISM and must never become a statement about
// the machine.** Sharing is a hard link first, a symbolic link second and a byte
// copy last, and the copy is designed — on Windows with the repository on one
// volume and the temporary directory on another, a hard link is refused across
// the volumes and a symbolic link is refused for want of privilege, so the copy
// is that box's normal, correct path. Asserting "the art was shared" there fails
// permanently and says nothing about the code. So the guard PROBES: it tries the
// two links itself, in the very directory the shares would land in, and then
// either skips quoting the refusal it got, or asserts. What it may never do is
// read what CopyData did and skip on that — a skip keyed on "the art was copied"
// is the guard deleting itself on every machine, forever.

// Reporter is the part of a test these guards report through.
//
// It is an interface of this package's own rather than testing.TB because
// testing.TB cannot be implemented outside the standard library — it holds an
// unexported method — and **both arms of the decision below have to be
// demonstrated by a test**: one machine shape that skips, one that fails. A
// suite that can only observe whichever arm it happens to be on is exactly the
// blindness this file exists to remove, so the seam that lets a test watch a
// skip happen is part of the design rather than a convenience.
//
// *testing.T and *testing.B satisfy it as they are.
type Reporter interface {
	Helper()
	Skipf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// artProbeName is the name a probe gives itself inside a scratch directory.
//
// It is removed again as soon as it is made, and it is a dotfile so that a
// removal that somehow failed leaves something no book walk and no art walk in
// this repository will pick up.
const artProbeName = ".hexarena-share-probe"

// RequireSharedArt is the guard: a scratch data directory must not hold its own
// copy of the shipped pictures, on a machine that could have shared them.
//
// It asserts the mechanism and not a stopwatch. A timing assertion would be a
// flake, and would pass anyway on a machine fast enough to make the copy look
// cheap. What it is watching for is a change that put every picture on the copy
// path — the fallback quietly becoming the normal path leaves every test in
// every one of those packages green and every one of them back at its old cost,
// which is the only failure a green suite cannot show.
//
// Where neither kind of link can be made this skips instead, naming the two
// refusals it was given. That is not the assertion being softened: the probe
// below uses the same two calls the sharing does, so a machine that CAN link and
// did not still fails here, which is the regression the guard is for.
func RequireSharedArt(t Reporter, shipped, scratch string) {
	t.Helper()
	if !sharingIsAvailable(t, shipped, scratch) {
		return
	}
	copied, walked, err := CopiedArt(shipped, scratch)
	if err != nil {
		t.Fatalf("compare the art: %v", err)
		return
	}
	if walked == 0 {
		t.Fatalf("no pictures were compared, so nothing here is measured")
		return
	}
	if len(copied) != 0 {
		t.Errorf("%d of %d shipped pictures were copied rather than shared, starting with %v: "+
			"this filesystem accepted a link a moment ago, so this is the fallback becoming "+
			"the normal path rather than a machine that cannot share",
			len(copied), walked, copied[:min(3, len(copied))])
	}
}

// SkipWithoutArtSharing skips t where this machine cannot give a scratch
// directory a second name for a shipped picture.
//
// It is RequireSharedArt's first half on its own, for the one caller that makes
// the same claim by a different reading: internal/forge counts the BYTES a
// scratch directory wrote, which catches what comparing inodes cannot. That
// assertion fails on a cross-volume Windows box for exactly the same reason the
// inode one does — a copied picture is seventeen megabytes however the reading
// is taken — so the two share this decision rather than each making up their own.
func SkipWithoutArtSharing(t Reporter, shipped, scratch string) {
	t.Helper()
	sharingIsAvailable(t, shipped, scratch)
}

// sharingIsAvailable is the decision, written once. It reports whether the
// caller should go on and assert; it has already skipped or failed t if not.
func sharingIsAvailable(t Reporter, shipped, scratch string) bool {
	t.Helper()
	refusal, err := artSharingRefusal(shipped, scratch)
	if err != nil {
		t.Fatalf("probe whether %s can share a picture from %s: %v", scratch, shipped, err)
		return false
	}
	if refusal != "" {
		// The refusal names both paths itself — os.Link and os.Symlink return a
		// *os.LinkError carrying the two — so this does not say them again.
		t.Skipf("this filesystem refuses both ways of giving a scratch directory a second name "+
			"for a shipped picture, so a byte copy is the designed path here and there is no "+
			"sharing to assert. The refusals were — %s", refusal)
		return false
	}
	return true
}

// artSharingRefusal asks this machine, in this place, whether it can give a
// picture under shipped a second name inside scratch — and says how it was
// refused when it cannot.
//
// ⚠️ **It is a probe and it may not be a reading of what CopyData did.** The two
// answer different questions and only one of them is a guard. "The art was
// copied" is true both on a machine that cannot link and on a mechanism that
// stopped linking, so a skip keyed on it is off everywhere and forever; "the
// link this very filesystem was just offered was refused" is true only in the
// first case. So this makes the calls itself, through the same two seams
// shareFile uses, from a real shipped picture to a real name inside the scratch
// directory — which is the pair of volumes that matters, the source being beside
// the repository and the destination under the temporary directory.
//
// The order is the sharing's own order, hard link then symbolic link, because a
// probe that tried them the other way round would answer about a mechanism
// nobody runs.
func artSharingRefusal(shipped, scratch string) (string, error) {
	source, err := anArtFile(shipped)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", source, err)
	}
	probe := filepath.Join(scratch, artProbeName)
	if err := clearProbe(probe); err != nil {
		return "", err
	}
	hard := linkFile(absolute, probe)
	if hard == nil {
		return "", clearProbe(probe)
	}
	soft := symlinkFile(absolute, probe)
	if soft == nil {
		return "", clearProbe(probe)
	}
	return fmt.Sprintf("hard link: %v; symbolic link: %v", hard, soft), nil
}

// clearProbe takes the probe's name back off the filesystem.
//
// It runs before the probe as well as after it: a name left behind by a run that
// was killed would make the next hard link fail as "file exists", which is a
// refusal about the leftover rather than about the volumes and would skip a
// guard that had every reason to run.
func clearProbe(probe string) error {
	if err := os.Remove(probe); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clear %s: %w", probe, err)
	}
	return nil
}

// anArtFile is the picture a probe borrows: the first regular file under a data
// directory's art.
//
// The order is filepath.WalkDir's, which is lexical, so the same picture is
// probed with on every run and on every machine — a probe that picked a
// different file each time would be a different measurement each time. A
// directory with no picture in it is an error rather than a pass: it is the same
// hole CopiedArt's walked count exists to close, and answering "nothing was
// refused" for a directory holding nothing would skip the assertion by the back
// door.
func anArtFile(shipped string) (string, error) {
	root := filepath.Join(shipped, ArtDir)
	var found string
	walk := func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		found = name
		return fs.SkipAll
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return "", fmt.Errorf("look for a picture under %s: %w", root, err)
	}
	if found == "" {
		return "", fmt.Errorf("no picture under %s, so there is nothing to probe with", root)
	}
	return found, nil
}
