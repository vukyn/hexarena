package testfixture

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ⚠️ Nothing here imports `time`, and that is a module rule rather than a
// preference: internal/socket's TestEveryClockInTheModuleIsOnTheAllowlist walks
// every non-test file and refuses one that imports it without a line on the
// allowlist saying why. Nothing below reads a clock — a file's stamp is copied
// and compared, never taken — so the honest answer is to hold the stamp as the
// string it prints as and to hand os.Chtimes a value it was given, rather than
// to put a fixture on a list kept for the transport.

// ArtDir is the folder inside a data directory that art lives under.
//
// internal/forge declares the same folder for the game itself, and this package
// cannot read that declaration: internal/forge's own tests import this package,
// so importing it back would be an import cycle in that test binary. A second
// spelling of one folder name is otherwise exactly the mistake internal/forge
// names in its own constant's comment, which is why the two are held together by
// a test over there rather than by hope.
const ArtDir = "assets"

// DataCopy is what CopyData did, so a caller may make an assertion about the
// mechanism instead of about a stopwatch.
//
// The counts are here because the fallbacks below must never become the normal
// path without somebody noticing. A copy that quietly stopped sharing is a
// change that does nothing while every test still passes — which is the shape
// this whole file was written to remove, so it would be a poor thing to
// reintroduce one layer down.
type DataCopy struct {
	Copied      int   // files written out byte for byte
	Linked      int   // art given a second name for the same bytes
	Symlinked   int   // art pointed at, where a second name was refused
	BytesCopied int64 // bytes actually written
}

// Shared is the art this copy did not have to write.
func (d DataCopy) Shared() int { return d.Linked + d.Symlinked }

// linkFile and symlinkFile are seams, so the fallbacks below can be exercised on
// a machine where the real calls succeed. A fallback nothing has ever run is not
// a fallback, it is code with an opinion about a platform nobody tested.
var (
	linkFile    = os.Link
	symlinkFile = os.Symlink
)

// CopyData puts one data directory's contents into another: the books are
// copied, and the art is SHARED.
//
// The split is the whole point. A test that drives the authoring tools needs a
// data directory it may write to — the save key, `skills add` and `skills edit`
// all write — so a single shared copy is not available. What those tests do not
// need is their own pictures: the shipped assets folder is 16 MB of SVG against
// 350 KB of JSON, and no assertion in this repository reads an asset's bytes
// except the art preview, which only ever reads them. Copying the books and
// sharing the art therefore costs a few hundred kilobytes per scratch directory
// instead of seventeen megabytes, and — this is the direction that matters — it
// stops the per-test cost growing every time a picture is added.
//
// Sharing is a hard link first, a symbolic link second and a byte copy last.
// A hard link is preferred because it is the one that works where this hurts
// most: on Windows a hard link needs no privilege on a volume, while a symbolic
// link needs developer mode, and Windows is where a scratch copy was measured at
// four to ten seconds. A hard link can be refused across filesystems, which is
// what the symbolic link answers, and a symbolic link can be refused by
// privilege, which is what the copy answers. All three preserve the file's size
// and modification time, deliberately: forge's art preview keys its cache on
// exactly those two, so a mechanism that changed either would be a mechanism
// that changed what is under test.
func CopyData(from, to string) (DataCopy, error) {
	var done DataCopy
	if err := copyInto(&done, from, to, false); err != nil {
		return done, err
	}
	return done, nil
}

// copyInto is CopyData over one directory, told whether it is inside the art.
func copyInto(done *DataCopy, from, to string, art bool) error {
	entries, err := os.ReadDir(from)
	if err != nil {
		return fmt.Errorf("read %s: %w", from, err)
	}
	for _, entry := range entries {
		source, destination := filepath.Join(from, entry.Name()), filepath.Join(to, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return fmt.Errorf("create %s: %w", destination, err)
			}
			if err := copyInto(done, source, destination, art || entry.Name() == ArtDir); err != nil {
				return err
			}
			continue
		}
		if art {
			shared, err := shareFile(done, source, destination)
			if err != nil {
				return err
			}
			if shared {
				continue
			}
		}
		if err := copyFile(done, source, destination, art); err != nil {
			return err
		}
	}
	return nil
}

// shareFile gives one picture a second name, and says whether it managed it.
//
// An error here is a refusal to share that is worth reporting rather than
// working around — a source that cannot be read, or a guard below that has
// caught a test writing through a share. Everything else falls through to a
// copy, which is always correct and only ever slow.
func shareFile(done *DataCopy, source, destination string) (bool, error) {
	info, err := os.Stat(source)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	if err := rememberArt(source, info); err != nil {
		return false, err
	}
	absolute, err := filepath.Abs(source)
	if err != nil {
		return false, fmt.Errorf("resolve %s: %w", source, err)
	}
	if err := linkFile(absolute, destination); err == nil {
		done.Linked++
		return true, nil
	}
	if err := symlinkFile(absolute, destination); err == nil {
		done.Symlinked++
		return true, nil
	}
	return false, nil
}

// copyFile writes one file out. Art keeps its modification time as well as its
// bytes, because the copy is the fallback for a share and a fallback that
// behaves differently is a second thing under test rather than a safety net —
// the preview's cache reads that stamp.
func copyFile(done *DataCopy, source, destination string, art bool) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	if err := os.WriteFile(destination, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", destination, err)
	}
	done.Copied++
	done.BytesCopied += int64(len(raw))
	if !art {
		return nil
	}
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat %s: %w", source, err)
	}
	if err := os.Chtimes(destination, info.ModTime(), info.ModTime()); err != nil {
		return fmt.Errorf("stamp %s: %w", destination, err)
	}
	return nil
}

// PrivateArt takes one picture out of the share, so a test may write to it.
//
// A shared file is the same file: writing through it reaches whatever it was
// shared from, which for the shipped art is a committed picture. Exactly one
// test in this repository rewrites art it was handed rather than art it made,
// and it calls this first. The name is what makes that visible — a test that
// mutates art and does not say so is the one way this arrangement could damage
// the repository, and a silent one.
//
// The replacement is a plain copy of the bytes, made by writing a new file over
// the link rather than through it. The modification time is deliberately NOT
// preserved: a caller taking a private copy is about to change the file, so a
// stamp from before the change would be a lie the preview's cache would believe.
func PrivateArt(dir, image string) error {
	name := filepath.Join(dir, filepath.FromSlash(image))
	raw, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("unshare %s: %w", name, err)
	}
	if err := os.WriteFile(name, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// artStamp is what a shared picture looked like the first time it was shared.
//
// The stamp is held as the string it prints as rather than as a time: this file
// may not import `time` (see the note at the top), the comparison wanted is
// equality rather than order, and the refusal below has to print it anyway.
type artStamp struct {
	size    int64
	modTime string
	mode    fs.FileMode
}

var (
	stampsMu sync.Mutex
	stamps   = map[string]artStamp{}
)

// rememberArt is the guard that turns a write through a share into a red test
// instead of a corrupted repository.
//
// Sharing means a scratch directory's picture and the committed one are the same
// bytes, so a test that writes to art it was handed writes into the repository.
// Nothing in the suite does today and PrivateArt is how a test says it is about
// to — but "nothing does today" is not a property, so every share checks that
// the source still looks exactly as it did the first time this process shared
// it. The first test to write through is then named by the next scratch
// directory that asks for the same picture, which for the shipped data is the
// very next test.
//
// A source inside the temporary directory is skipped: that is itself a scratch
// tree about to be deleted, so remembering it would cost memory to protect
// nothing. Being wrong about that costs a map entry, never a wrong answer.
func rememberArt(source string, info fs.FileInfo) error {
	if inScratch(source) {
		return nil
	}
	now := artStamp{size: info.Size(), modTime: info.ModTime().String(), mode: info.Mode()}
	stampsMu.Lock()
	defer stampsMu.Unlock()
	was, seen := stamps[source]
	if !seen {
		stamps[source] = now
		return nil
	}
	if was == now {
		return nil
	}
	return fmt.Errorf("%s changed while the tests were running (%d bytes %v %s, was %d bytes %v %s): "+
		"a test wrote through a shared picture and has edited the file it was shared from. "+
		"A test that means to rewrite art must call testfixture.PrivateArt first",
		source, now.size, now.mode, now.modTime, was.size, was.mode, was.modTime)
}

// scratchRoot is where a throwaway tree lives. It is a variable rather than a
// call so that this package's own tests can exercise the guard above over a
// source they control — everything a test can create is under the temporary
// directory, so a guard that ignores that directory is otherwise a guard no
// test can reach.
var scratchRoot = os.TempDir()

// inScratch reports whether a path is under the temporary directory.
func inScratch(name string) bool {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return false
	}
	root, err := filepath.Abs(scratchRoot)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absolute, root+string(filepath.Separator))
}

// CopiedArt walks a shipped data directory's art and reports which of it a
// scratch copy has its own bytes for, and how many pictures it looked at.
//
// It is here rather than in each suite because five packages make scratch data
// directories and all five need the same guard: the fallbacks in CopyData are
// correct and slow, so a machine or a refactor that put every picture on the
// slow path would leave every test passing and the arithmetic back where it
// started. The count comes back so that a suite whose data directory has no art
// in it cannot pass this by walking nothing.
func CopiedArt(shipped, scratch string) (copied []string, walked int, err error) {
	root := filepath.Join(shipped, ArtDir)
	walk := func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		here, err := os.Stat(name)
		if err != nil {
			return err
		}
		there, err := os.Stat(filepath.Join(scratch, ArtDir, relative))
		if err != nil {
			return err
		}
		walked++
		if !os.SameFile(here, there) {
			copied = append(copied, filepath.ToSlash(relative))
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil, walked, fmt.Errorf("compare the art under %s: %w", root, err)
	}
	return copied, walked, nil
}
