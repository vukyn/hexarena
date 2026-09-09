package testfixture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the tests for the decision in guard.go, and the thing they are for
// is that **both arms are demonstrated on a machine that can link**. The arm
// that matters most is unreachable here otherwise: a box where neither kind of
// link is available is a Windows one with the repository and the temporary
// directory on different volumes, and a fix nobody has run on the platform it is
// for is the defect SCR-013 is about. The seams linkFile and symlinkFile are how
// that platform is brought here, and the refusals below are the two that box
// really gave, word for word, so the message a reader will see is the message
// measured here.

// The two refusals a Windows box gave with the repository on H: and TMP under
// C:\…\Temp. They are named rather than inlined so that the sentence a reader
// meets and the sentence under test are the same string.
const (
	windowsCrossVolume = "The system cannot move the file to a different disk drive."
	windowsNoPrivilege = "Administrator privilege required for this operation."
)

// recorded is a Reporter that writes down what it was told instead of acting on
// it, which is the only way one test can watch another guard skip.
//
// ⚠️ A real Skipf or Fatalf ends the test where it is called; this one returns.
// That is what makes the `return` after every terminal call in guard.go load
// bearing rather than tidy, and it is asserted here rather than assumed: the
// skip arm below checks that nothing was reported AFTER the skip.
type recorded struct {
	skips  []string
	errors []string
	fatals []string
}

func (r *recorded) Helper() {}

func (r *recorded) Skipf(format string, args ...any) {
	r.skips = append(r.skips, fmt.Sprintf(format, args...))
}

func (r *recorded) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recorded) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

// quiet says whether a guard said nothing at all, which is what passing is.
func (r *recorded) quiet() bool {
	return len(r.skips)+len(r.errors)+len(r.fatals) == 0
}

func (r *recorded) String() string {
	return fmt.Sprintf("skips %q errors %q fatals %q", r.skips, r.errors, r.fatals)
}

// refuseBothLinksLikeWindows makes both sharing calls fail with the errors a
// cross-volume Windows box gives, in the shape the standard library wraps them
// in, and puts them back afterwards.
func refuseBothLinksLikeWindows(t *testing.T) {
	t.Helper()
	link, symlink := linkFile, symlinkFile
	t.Cleanup(func() { linkFile, symlinkFile = link, symlink })
	linkFile = func(old, new string) error {
		return &os.LinkError{Op: "link", Old: old, New: new, Err: errors.New(windowsCrossVolume)}
	}
	symlinkFile = func(old, new string) error {
		return &os.LinkError{Op: "symlink", Old: old, New: new, Err: errors.New(windowsNoPrivilege)}
	}
}

// copiedScratchDir builds a scratch directory the way a cross-volume Windows box
// would: both links refused, so every picture is written out.
//
// The refusals are put back before it returns, so the caller is a machine that
// CAN link looking at a directory whose art was copied — which is the shape the
// two arms differ on and the only one that can tell them apart.
func copiedScratchDir(t *testing.T, source string) string {
	t.Helper()
	link, symlink := linkFile, symlinkFile
	linkFile = func(old, new string) error {
		return &os.LinkError{Op: "link", Old: old, New: new, Err: errors.New(windowsCrossVolume)}
	}
	symlinkFile = func(old, new string) error {
		return &os.LinkError{Op: "symlink", Old: old, New: new, Err: errors.New(windowsNoPrivilege)}
	}
	target := t.TempDir()
	done, err := CopyData(source, target)
	linkFile, symlinkFile = link, symlink
	if err != nil {
		t.Fatalf("copy the data: %v", err)
	}
	if done.Shared() != 0 {
		t.Fatalf("%d pictures were shared, want none: this is meant to be the copy path (%+v)",
			done.Shared(), done)
	}
	return target
}

// TestTheArtGuardSkipsWhereNeitherKindOfLinkIsAvailable is the first arm, and it
// is the reported failure: on Windows across two volumes the byte copy is the
// designed path, so the guard has nothing to assert and must say so rather than
// failing forever.
func TestTheArtGuardSkipsWhereNeitherKindOfLinkIsAvailable(t *testing.T) {
	refuseBothLinksLikeWindows(t)
	source := aDataDir(t)
	target := t.TempDir()
	if _, err := CopyData(source, target); err != nil {
		t.Fatalf("copy the data: %v", err)
	}

	var said recorded
	RequireSharedArt(&said, source, target)

	if len(said.skips) != 1 {
		t.Fatalf("the guard skipped %d times, want exactly once: %v", len(said.skips), &said)
	}
	if len(said.errors) != 0 || len(said.fatals) != 0 {
		t.Errorf("the guard went on reporting after it skipped: %v", &said)
	}
	for _, quoted := range []string{windowsCrossVolume, windowsNoPrivilege} {
		if !strings.Contains(said.skips[0], quoted) {
			t.Errorf("the skip does not quote %q, so a reader cannot tell why it fired: %q",
				quoted, said.skips[0])
		}
	}
	t.Logf("what a cross-volume Windows box now prints:\n%s", said.skips[0])
}

// TestTheArtGuardFailsWhereLinkingWorksAndTheArtWasCopiedAnyway is the second
// arm, and it is the whole reason the first one is a probe.
//
// ⚠️ **A skip that fires whenever the art was copied would pass this test's
// premise and delete the guard on every machine, forever.** The directory here
// holds copied pictures — built exactly as the Windows box builds one — and the
// machine looking at it can link. Those two facts together are the regression
// the guard exists for, and a fix that cannot tell this apart from the arm above
// is not a fix.
func TestTheArtGuardFailsWhereLinkingWorksAndTheArtWasCopiedAnyway(t *testing.T) {
	source := aDataDir(t)
	target := copiedScratchDir(t, source)

	var said recorded
	RequireSharedArt(&said, source, target)

	if len(said.skips) != 0 {
		t.Fatalf("the guard skipped on a machine that can link, which is the guard deleting "+
			"itself: %v", &said)
	}
	if len(said.errors) != 1 {
		t.Fatalf("the guard reported %d failures, want exactly one: %v", len(said.errors), &said)
	}
	if !strings.Contains(said.errors[0], "2 of 2 shipped pictures were copied") {
		t.Errorf("the failure does not count the pictures: %q", said.errors[0])
	}
}

// TestTheArtGuardSaysNothingWhereLinkingWorksAndTheArtWasShared is the ordinary
// machine, and the count is what stops it passing by walking nothing.
func TestTheArtGuardSaysNothingWhereLinkingWorksAndTheArtWasShared(t *testing.T) {
	source := aDataDir(t)
	target := t.TempDir()
	if _, err := CopyData(source, target); err != nil {
		t.Fatalf("copy the data: %v", err)
	}

	var said recorded
	RequireSharedArt(&said, source, target)
	if !said.quiet() {
		t.Fatalf("the guard reported something about a directory that shared its art: %v", &said)
	}

	copied, walked, err := CopiedArt(source, target)
	if err != nil {
		t.Fatalf("compare the art: %v", err)
	}
	if walked != 2 {
		t.Errorf("the guard compared %d pictures, want the fixture's 2: a walk over nothing "+
			"agrees with any assertion made over it", walked)
	}
	if len(copied) != 0 {
		t.Errorf("%v was copied, so this test is not measuring the quiet case", copied)
	}
}

// TestTheProbeReportsNoRefusalWhereLinkingWorks is the mechanism under the test
// above, said directly: the skip cannot fire on a machine that can link.
//
// Both are here because the guard could stop skipping for a reason that has
// nothing to do with the probe, and the probe could stop refusing for a reason
// that never reaches the guard.
func TestTheProbeReportsNoRefusalWhereLinkingWorks(t *testing.T) {
	source := aDataDir(t)
	refusal, err := artSharingRefusal(source, t.TempDir())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if refusal != "" {
		t.Errorf("the probe refused on a machine that can link, which would skip a guard that "+
			"has every reason to run: %s", refusal)
	}
}

// TestTheProbeLeavesNothingBehind is the price of probing in the directory under
// test: it writes there, so it has to take it back off again — on the path where
// the link worked and on the path where both were refused.
func TestTheProbeLeavesNothingBehind(t *testing.T) {
	source := aDataDir(t)
	for _, arm := range []struct {
		name    string
		refused bool
	}{
		{name: "the link works", refused: false},
		{name: "both links refused", refused: true},
	} {
		t.Run(arm.name, func(t *testing.T) {
			if arm.refused {
				refuseBothLinksLikeWindows(t)
			}
			target := t.TempDir()
			if _, err := artSharingRefusal(source, target); err != nil {
				t.Fatalf("probe: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(target, artProbeName)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the probe is still there afterwards (%v)", err)
			}
		})
	}
}

// TestALeftoverProbeDoesNotTurnTheGuardOff is the failure mode of probing under
// a fixed name: a hard link onto a name that already exists is refused as "file
// exists", which is a refusal about the leftover rather than about the volumes —
// and it would skip a guard on a machine that shares perfectly well.
func TestALeftoverProbeDoesNotTurnTheGuardOff(t *testing.T) {
	source := aDataDir(t)
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, artProbeName), []byte("left behind"), 0o644); err != nil {
		t.Fatalf("leave a probe behind: %v", err)
	}
	refusal, err := artSharingRefusal(source, target)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if refusal != "" {
		t.Errorf("a leftover probe made the machine look unable to share: %s", refusal)
	}
}

// TestTheProbeRefusesToPassADirectoryWithNoArt closes the back door: answering
// "nothing was refused" for a directory holding no picture would let a suite
// whose data directory has lost its art skip the guard rather than fail it.
func TestTheProbeRefusesToPassADirectoryWithNoArt(t *testing.T) {
	bare := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bare, ArtDir), 0o755); err != nil {
		t.Fatalf("make an empty art folder: %v", err)
	}

	var said recorded
	RequireSharedArt(&said, bare, t.TempDir())
	if len(said.fatals) != 1 {
		t.Fatalf("the guard reported %d fatals over a data directory with no art, want one: %v",
			len(said.fatals), &said)
	}
	if len(said.skips) != 0 {
		t.Errorf("the guard skipped over a data directory with no art: %v", &said)
	}
	if !strings.Contains(said.fatals[0], "nothing to probe with") {
		t.Errorf("the refusal does not say what is missing: %q", said.fatals[0])
	}
}

// TestSkipWithoutArtSharingIsTheSameDecision is what stops the byte-counting
// caller in internal/forge growing a second opinion about when to skip. It is
// the one thing that could drift back into two rules.
func TestSkipWithoutArtSharingIsTheSameDecision(t *testing.T) {
	source := aDataDir(t)

	var quiet recorded
	SkipWithoutArtSharing(&quiet, source, t.TempDir())
	if !quiet.quiet() {
		t.Errorf("it reported something on a machine that can link: %v", &quiet)
	}

	refuseBothLinksLikeWindows(t)
	var skipped recorded
	SkipWithoutArtSharing(&skipped, source, t.TempDir())
	if len(skipped.skips) != 1 {
		t.Fatalf("it skipped %d times where neither link is available, want once: %v",
			len(skipped.skips), &skipped)
	}
	if !strings.Contains(skipped.skips[0], windowsCrossVolume) {
		t.Errorf("the skip does not quote the refusal: %q", skipped.skips[0])
	}
}
