// These tests are in package seed, not seed_test, unlike the other thirty-five
// test files here. They have to be: the totality guard walks the unexported
// embed.FS and the algorithm tests inject an fs.FS into the unexported digest,
// and both of those are the point. Nothing else in this file reaches inside the
// package.
//
// There is deliberately **no golden on the digest value.** A golden would move
// on every unrelated data commit — this repository ships data PRs constantly and
// from parallel sessions — while catching nothing the properties below do not.
// It would be a merge-conflict generator that measures nothing. Everything
// asserted here is data-change-proof: it is about the algorithm's sensitivity,
// never about today's bytes.
package seed

import (
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// fifteenNamed builds a MapFS holding every name in dataFiles with contents
// distinct per file, so a test that swaps or flips anything is working on bytes
// that could not have collided by accident.
func fifteenNamed() fstest.MapFS {
	fsys := make(fstest.MapFS, len(dataFiles))
	for index, name := range dataFiles {
		fsys[name] = &fstest.MapFile{Data: []byte(name + " payload " + string(rune('a'+index)))}
	}
	return fsys
}

// mustDigest is the shorthand every algorithm test below uses; a digest over an
// injected MapFS has no reason to fail and a failure is the test's own bug.
func mustDigest(t *testing.T, fsys fs.FS, names []string) Digest {
	t.Helper()
	got, err := digest(fsys, names)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	return got
}

// TestEveryEmbeddedDataFileIsNamedInTheDigest is the guard that holds the three
// independent copies of these fifteen names together: the go:embed directive,
// the fifteen ReadFile calls, and dataFiles. Only a walk can, because adding a
// sixteenth file to the directive and forgetting dataFiles is otherwise a silent
// hole in the gate — a peer that changed that file would still pass.
//
// It runs both directions on purpose. Left to right catches the new file nobody
// added to the list; right to left catches a name in the list that the directive
// no longer embeds, which would turn DataDigest into a permanent error.
//
// The walk is also what covers the two things that are not data: a `.DS_Store`
// (macOS writes them into internal/seed/data/, and one has been in this tree
// before) and any future assets/ leak. Both would arrive as extra files in the
// FS and both redden this test.
func TestEveryEmbeddedDataFileIsNamedInTheDigest(t *testing.T) {
	named := make(map[string]bool, len(dataFiles))
	for _, name := range dataFiles {
		if named[name] {
			t.Fatalf("dataFiles names %s twice", name)
		}
		named[name] = true
	}

	walked := make(map[string]bool, len(dataFiles))
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		walked[path] = true
		if !named[path] {
			t.Errorf("the embedded FS holds %s and dataFiles does not name it: the digest cannot see changes to it", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the embedded FS: %v", err)
	}

	for _, name := range dataFiles {
		if !walked[name] {
			t.Errorf("dataFiles names %s and the embedded FS has no such file: DataDigest would only ever error", name)
		}
	}
	if len(walked) != len(dataFiles) {
		t.Errorf("the embedded FS holds %d files and dataFiles names %d", len(walked), len(dataFiles))
	}
}

// TestArtCannotReachTheSimulation is the record's exclusion as a measurement.
// A peer with a newer picture is not a peer that fights a different battle, so
// refusing it at the gate would refuse a match that would have been identical.
//
// What this actually guards is the directive. It names fifteen files today;
// changing it to `data` or `all:data` — the obvious tidy-up — would pull in
// fifty-five SVGs and every stray `.DS_Store`, and the digest would start
// refusing peers over art.
func TestArtCannotReachTheSimulation(t *testing.T) {
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(path, "data/assets") {
			t.Errorf("%s: art is embedded into the data FS and would enter the digest", path)
		}
		if strings.HasSuffix(path, ".svg") {
			t.Errorf("%s: an SVG is embedded into the data FS and would enter the digest", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the embedded FS: %v", err)
	}
	if _, err := fs.Stat(files, "data/assets"); err == nil {
		t.Error("data/assets exists inside the embedded FS")
	}
}

// TestFlippingOneByteInAnyFileChangesTheDigest is table-driven off dataFiles
// rather than a written-out list of fifteen, so the sixteenth file is covered
// the day somebody adds it to that slice.
//
// This is the property the gate rests on: a power edited in skills.json changes
// every battle, and it has to change the digest too.
func TestFlippingOneByteInAnyFileChangesTheDigest(t *testing.T) {
	base := mustDigest(t, fifteenNamed(), dataFiles)
	for _, name := range dataFiles {
		t.Run(name, func(t *testing.T) {
			fsys := fifteenNamed()
			flipped := append([]byte(nil), fsys[name].Data...)
			flipped[0] ^= 0xff
			fsys[name] = &fstest.MapFile{Data: flipped}
			if got := mustDigest(t, fsys, dataFiles); got == base {
				t.Fatalf("one byte of %s changed and the digest did not", name)
			}
		})
	}
}

// TestTheNameAndLengthFrameEveryFileInTheDigest is the test that holds the
// framing in digest — the name and the fixed-width length in front of every
// file's bytes — in place. Four cases, and they are **not** equally sharp, which
// is the whole reason they are written out one at a time: the first is the case
// anyone reaches for first, and it is the one that holds nothing.
//
// Each was measured by removing a piece of the framing and running this test.
// Which case reddens for which piece (⌀ = still green):
//
//	                     | name+length out | name out | length out
//	exchanged            |        ⌀        |    ⌀     |     ⌀
//	repartitioned        |       RED       |    ⌀     |     ⌀
//	renamed              |       RED       |   RED    |     ⌀
//	boundaryUnderAName   |        ⌀        |    ⌀     |    RED
//
// ⚠️ Read the last row before adding a fifth case: it is green with the framing
// gone entirely and red with only the length gone, because the collision it
// builds is a collision *of the framed stream* — take the name out and the two
// arrangements concatenate differently again. A case that reddens under every
// mutation would be measuring the whole framing at once, which is exactly what
// this table exists to take apart.
//
//   - exchanged: two files of the same length trade contents. This is the case
//     that gets named as the one a plain sha256 over the concatenated bytes is
//     blind to — it was named that way here, before it was measured.
//     ⚠️ It is not: the files are read in list order, so a swap moves those bytes
//     to different offsets and a content-only hash sees it too. Kept because it
//     is the case a gate would actually meet, and because a wrong justification
//     is worth keeping next to the measurement that killed it.
//   - repartitioned: two adjacent files hold the same total bytes with the
//     boundary moved. This is the blindness the framing is actually for, and it
//     is caught by the **name** rather than by the length — a name between two
//     files' bytes is also a separator.
//   - renamed: the same bytes under a different name. The name and nothing else.
//   - boundaryUnderAName: a boundary moved *across a copy of the next file's own
//     name*, so the framed stream is byte-identical too and only the length can
//     tell the two apart. Contrived on purpose — the point of the length is that
//     the framing cannot be confused rather than that it is hard to confuse — and
//     it is the only assertion in the package that holds the length prefix.
//
// Two peers in any of these four states do not simulate the same battle; the two
// with a moved boundary do not simulate a battle at all.
func TestTheNameAndLengthFrameEveryFileInTheDigest(t *testing.T) {
	const first, second = "data/skills.json", "data/roster.json"
	names := []string{first, second}

	t.Run("exchanged", func(t *testing.T) {
		straight := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("aaaa")},
			second: &fstest.MapFile{Data: []byte("bbbb")},
		}
		swapped := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("bbbb")},
			second: &fstest.MapFile{Data: []byte("aaaa")},
		}
		if mustDigest(t, straight, names) == mustDigest(t, swapped, names) {
			t.Fatal("two files exchanged contents and the digest did not move")
		}
	})

	t.Run("repartitioned", func(t *testing.T) {
		straight := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("aaaa")},
			second: &fstest.MapFile{Data: []byte("bbbb")},
		}
		// Same eight bytes in the same order, one boundary to the right. The
		// concatenation is identical; only the lengths differ.
		shifted := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("aaaabb")},
			second: &fstest.MapFile{Data: []byte("bb")},
		}
		if mustDigest(t, straight, names) == mustDigest(t, shifted, names) {
			t.Fatal("the boundary between two files moved and the digest did not: the length prefix is gone from digest")
		}
	})

	t.Run("renamed", func(t *testing.T) {
		straight := fstest.MapFS{first: &fstest.MapFile{Data: []byte("aaaa")}}
		renamed := fstest.MapFS{second: &fstest.MapFile{Data: []byte("aaaa")}}
		if mustDigest(t, straight, []string{first}) == mustDigest(t, renamed, []string{second}) {
			t.Fatal("the same bytes under a different name digested the same: the name prefix is gone from digest")
		}
	})

	t.Run("boundaryUnderAName", func(t *testing.T) {
		// Both arrangements frame to the same stream once the lengths are taken
		// out of it: `first || "aa" || second || "P" || second || "Q"`. The
		// second file's own name is the seam, which is why moving the boundary
		// across it hides under the name prefix and shows under the length.
		straight := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("aa")},
			second: &fstest.MapFile{Data: []byte("P" + second + "Q")},
		}
		shifted := fstest.MapFS{
			first:  &fstest.MapFile{Data: []byte("aa" + second + "P")},
			second: &fstest.MapFile{Data: []byte("Q")},
		}
		if mustDigest(t, straight, names) == mustDigest(t, shifted, names) {
			t.Fatal("a boundary moved across a copy of the next file's name digested the same: the length prefix is gone from digest")
		}
	})
}

// TestTheOrderOfTheFilesIsPartOfTheDigest asserts the record's rule rather than
// working around it. The order is the one the go:embed directive declares, so
// digest reads in the order it is handed and does not sort — a sort would make
// the digest agree with a peer whose list is in a different order, which is a
// peer whose dataFiles is not this dataFiles.
func TestTheOrderOfTheFilesIsPartOfTheDigest(t *testing.T) {
	fsys := fifteenNamed()
	reordered := append([]string(nil), dataFiles...)
	reordered[0], reordered[len(reordered)-1] = reordered[len(reordered)-1], reordered[0]
	if mustDigest(t, fsys, dataFiles) == mustDigest(t, fsys, reordered) {
		t.Fatal("the same files in a different order digested the same")
	}
}

// TestAMissingFileIsAnErrorNotAPartialDigest is the one failure mode this gate
// must never soften. A partial digest is worse than no digest: two peers that
// both skipped an unreadable file would agree, join, and then diverge on the
// first turn that read it.
func TestAMissingFileIsAnErrorNotAPartialDigest(t *testing.T) {
	for _, absent := range []string{dataFiles[0], dataFiles[7], dataFiles[len(dataFiles)-1]} {
		t.Run(absent, func(t *testing.T) {
			fsys := fifteenNamed()
			delete(fsys, absent)
			got, err := digest(fsys, dataFiles)
			if err == nil {
				t.Fatalf("%s is missing and digest returned %s with no error", absent, got.Short())
			}
			if !strings.Contains(err.Error(), absent) {
				t.Errorf("the error does not name the file it could not read: %v", err)
			}
			if got != (Digest{}) {
				t.Errorf("digest returned a usable-looking value alongside its error: %s", got)
			}
		})
	}
}

// TestNothingIsParsed is the record's "no parsing" rule as a measurement. A
// digest that depended on parsing would be a second reading of the data, and two
// readings of the data is the thing this repository keeps refusing to have — it
// would also make the gate fail on a peer whose data is fine and whose parser is
// one commit older.
func TestNothingIsParsed(t *testing.T) {
	fsys := make(fstest.MapFS, len(dataFiles))
	for _, name := range dataFiles {
		fsys[name] = &fstest.MapFile{Data: []byte("{ this is not json at all ][")}
	}
	if _, err := digest(fsys, dataFiles); err != nil {
		t.Fatalf("digest read invalid JSON and complained: %v", err)
	}
}

// TestTheShippedDigestIsStableAndNotVacuous is the only test here that touches
// the real embedded data, and it deliberately asserts nothing about the value.
// Two calls agreeing is the equality the gate is, and the two non-vacuity checks
// catch the two ways this could be a hash of nothing.
func TestTheShippedDigestIsStableAndNotVacuous(t *testing.T) {
	first, err := DataDigest()
	if err != nil {
		t.Fatalf("DataDigest: %v", err)
	}
	second, err := DataDigest()
	if err != nil {
		t.Fatalf("DataDigest again: %v", err)
	}
	if first != second {
		t.Fatalf("two calls disagreed: %s then %s", first, second)
	}
	if first == (Digest{}) {
		t.Fatal("the digest is the zero value")
	}
	if first == Digest(sha256.Sum256(nil)) {
		t.Fatal("the digest is sha256 of the empty input: no file was fed to the hash")
	}
	if len(first.String()) != 64 {
		t.Fatalf("String is %d characters, want 64", len(first.String()))
	}
	if strings.ToLower(first.String()) != first.String() {
		t.Fatalf("String is not lowercase hex: %s", first.String())
	}
	if first.Short() != first.String()[:12] {
		t.Fatalf("Short %s is not the first twelve of %s", first.Short(), first.String())
	}
}

// TestAnUneditedDirectoryDigestsAsTheEmbeddedCopyDoes is the claim DigestOf
// exists to make, and it is the one this package's own framing can break without
// either digest looking wrong.
//
// The shipped data directory **is** the embedded copy — go:embed reads exactly
// these files — so the two digests must be equal. They were not: the first
// DigestOf fed the hash `skills.json` where DataDigest feeds it
// `data/skills.json`, and a name is part of the frame, so every unedited
// directory came back as edited. Both digests were well formed and stable, which
// is why nothing else noticed.
//
// What it sees: the prefix dropped from the framed name, a file read from the
// wrong place, and a framing that stops matching DataDigest's for any other
// reason.
// What it cannot see: whether an *edited* directory is reported as edited — the
// half below is what says that.
func TestAnUneditedDirectoryDigestsAsTheEmbeddedCopyDoes(t *testing.T) {
	embedded, err := DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded copy: %v", err)
	}
	onDisk, err := DigestOf(os.DirFS("data"))
	if err != nil {
		t.Fatalf("digest the data directory: %v", err)
	}
	if onDisk != embedded {
		t.Errorf("the shipped data directory digests %s and the copy embedded from it "+
			"digests %s; these are the same bytes, so the two readings disagree about the "+
			"framing rather than about the data", onDisk.Short(), embedded.Short())
	}

	// And the other half: a directory that really has been edited says so. A
	// copy with one byte changed is the smallest edit there is.
	scratch := t.TempDir()
	for _, name := range dataFiles {
		raw, err := files.ReadFile(name)
		if err != nil {
			t.Fatalf("read the embedded %s: %v", name, err)
		}
		if name == "data/skills.json" {
			raw = append(raw, ' ')
		}
		bare := strings.TrimPrefix(name, dataPrefix)
		if err := os.WriteFile(filepath.Join(scratch, bare), raw, 0o600); err != nil {
			t.Fatalf("write %s: %v", bare, err)
		}
	}
	edited, err := DigestOf(os.DirFS(scratch))
	if err != nil {
		t.Fatalf("digest the edited directory: %v", err)
	}
	if edited == embedded {
		t.Error("a data directory with a byte added to skills.json digests the same as the " +
			"embedded copy, so the notice this feeds would never fire")
	}
}
