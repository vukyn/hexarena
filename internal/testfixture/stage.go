package testfixture

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Data fills target with a data directory a test may write to: the shipped books
// with the fixture injected, the shipped art shared, and the fixture's own art
// its own.
//
// # Why this exists rather than a CopyData followed by an Inject
//
// That is what five packages did, and Inject is the expensive half. It reloads
// the whole library before every write — once for the skill dependencies and
// once per skill, origin and character, which is fifty-odd full parses of every
// book — because each save rewrites a whole file and a library opened once would
// hold a stale copy of the one it is about to replace. Measured on this
// repository, that is about 180ms of a 200ms scratch directory, and the copying
// the previous half of this work removed was 22ms of it.
//
// So the injection is done **once per process** and its result — the book files,
// as bytes — is held in memory. A scratch directory is then those bytes written
// out, the shipped pictures linked, and the fixture's three pictures written. The
// books are still private per directory, which is the property the authoring
// suites rest on: they write, and two of them over one file would be a suite
// whose answers depended on the order it ran in.
//
// # Why it cannot go stale
//
// ⚠️ **The failure worth designing against is a staged directory built from old
// books while every test still passes**, so two things are true of this and both
// are deliberate. Nothing is written to disk between runs, so a later process can
// never pick up an earlier one's injection — the injector compiled into the test
// binary is always the one that ran. And within a run the stage is keyed on a
// digest of the books it was built from *and* of the fixture constants it wrote,
// so books edited while the tests are running produce a different stage rather
// than a stale one. See stageKey.
//
// load is given the directory to read, because the injection happens somewhere
// the caller has never heard of. It is a function rather than the library itself
// so that internal/forge's own tests can use this without an import cycle.
func Data(target, shipped string, load func(dir string) (Saver, error)) (DataCopy, error) {
	books, err := stagedBooks(shipped, load)
	if err != nil {
		return DataCopy{}, err
	}
	var done DataCopy
	for _, name := range slices.Sorted(maps.Keys(books)) {
		written := filepath.Join(target, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(written), 0o755); err != nil {
			return done, fmt.Errorf("create %s: %w", filepath.Dir(written), err)
		}
		if err := os.WriteFile(written, books[name], 0o644); err != nil {
			return done, fmt.Errorf("write %s: %w", written, err)
		}
		done.Copied++
		done.BytesCopied += int64(len(books[name]))
	}
	if err := shareArt(&done, shipped, target); err != nil {
		return done, err
	}
	if err := WriteArt(target); err != nil {
		return done, err
	}
	done.Copied += len(FixtureArt)
	done.BytesCopied += int64(len(FixtureArt) * len(Art))
	return done, nil
}

var (
	stageMu  sync.Mutex
	stages   = map[string]map[string][]byte{}
	stagings int
)

// InjectRoot is where injectOnce does its work, empty meaning os.TempDir().
//
// It is a seam for one assertion that cannot be made any other way: the
// directory an injection happens in is deleted before the injection returns, and
// "nothing was left behind" is only checkable against a directory the test owns.
// Globbing the real temporary directory would be a flake — `make check` runs
// five of these suites at once, each staging its own — so a test that wants to
// look points this somewhere private first.
var InjectRoot = ""

// Stagings reports how many times the fixture has been injected in this process.
//
// It is here so a test can assert the arrangement rather than a stopwatch: a
// staging that quietly stopped being shared would leave every test passing and
// the suite back at its old cost, which is the same failure CopiedArt names one
// layer up. A clock cannot say it, because a fast machine makes an unshared
// injection look cheap.
func Stagings() int {
	stageMu.Lock()
	defer stageMu.Unlock()
	return stagings
}

// stagedBooks hands back the injected books for one shipped directory, building
// them the first time it is asked.
//
// The lock is held across the build, which takes about a fifth of a second. That
// is deliberate: two callers arriving together should wait for one injection
// rather than run two.
func stagedBooks(shipped string, load func(dir string) (Saver, error)) (map[string][]byte, error) {
	key, err := stageKey(shipped, Skills, Archetypes, Origins, Characters)
	if err != nil {
		return nil, err
	}
	stageMu.Lock()
	defer stageMu.Unlock()
	if held, ok := stages[key]; ok {
		return held, nil
	}
	built, err := injectOnce(shipped, load)
	if err != nil {
		return nil, err
	}
	stages[key] = built
	stagings++
	return built, nil
}

// injectOnce copies the books somewhere private, injects the fixture into them
// and hands back what the files then held.
//
// The directory it works in is removed before this returns — the value that
// survives is the bytes, in memory, which is what makes a stage from a previous
// run impossible rather than merely unlikely. It is also what answers the
// question a staged *directory* could not: nothing has to clean it up later,
// because there is nothing left to clean up.
//
// The art is deliberately left out of the copy. Nothing in the injection reads a
// picture — the library validates the shape of an image path and only
// Library.ImageExists asks whether the file is really there — so staging the art
// would be linking sixteen megabytes into a directory about to be deleted.
func injectOnce(shipped string, load func(dir string) (Saver, error)) (map[string][]byte, error) {
	dir, err := os.MkdirTemp(InjectRoot, "hexarena-fixture-")
	if err != nil {
		return nil, fmt.Errorf("make somewhere to inject the fixture: %w", err)
	}
	defer os.RemoveAll(dir)
	var done DataCopy
	if err := copyInto(&done, shipped, dir, false, wantBooks); err != nil {
		return nil, err
	}
	if err := Inject(dir, func() (Saver, error) { return load(dir) }); err != nil {
		return nil, fmt.Errorf("inject the fixture: %w", err)
	}
	return readBooks(dir)
}

// readBooks reads every file in a data directory except the art.
func readBooks(dir string) (map[string][]byte, error) {
	books := map[string][]byte{}
	root := filepath.Clean(dir)
	walk := func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ArtDir && name != root {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		books[filepath.ToSlash(relative)] = raw
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil, fmt.Errorf("read the books under %s: %w", root, err)
	}
	if len(books) == 0 {
		return nil, fmt.Errorf("no books under %s, so there is nothing to inject into", root)
	}
	return books, nil
}

// stageKey is the digest a staged injection is held under.
//
// ⚠️ **It covers everything the injection reads, and that is the whole guard.**
// A key that did not would be a cache handing out books built from something
// that has since changed, with every test green — the one failure this
// arrangement could produce that nothing else would notice. So it hashes the
// shipped books byte for byte, and the fixture material the injection writes.
//
// Two things are deliberately *not* in it. The art is not, because no picture is
// staged: a scratch directory's shipped pictures are linked straight from the
// shipped directory, so they are the same bytes by construction, and the
// fixture's own are written from the Art constant every time. And the injector
// itself is not, because it cannot be — but it does not need to be, since
// nothing survives the process: the code that built a stage is always the code
// running now.
//
// Each piece is hashed with its length in front so that two different splits
// cannot come to one digest.
func stageKey(shipped string, fixture ...string) (string, error) {
	books, err := readBooks(shipped)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	for _, name := range slices.Sorted(maps.Keys(books)) {
		hashPiece(sum, []byte(name))
		hashPiece(sum, books[name])
	}
	for _, held := range fixture {
		hashPiece(sum, []byte(held))
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// hashPiece writes one length-prefixed piece into a digest.
func hashPiece(sum hash.Hash, raw []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(raw)))
	sum.Write(length[:])
	sum.Write(raw)
}
