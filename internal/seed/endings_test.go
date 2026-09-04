package seed

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// # The bytes, and the one property of them that is not about the game
//
// DataDigest hashes the fifteen files **unparsed**, which is the whole reason
// two peers can agree that they will fight the same battle without either of
// them having a canonical form of every book. The cost of that is a digest that
// moves when the bytes move for a reason that has nothing to do with the data —
// and exactly one such reason exists in practice.
//
// It happened: a Mac hosting, a Windows machine joining, no `.gitattributes` in
// the repository, and git's `core.autocrlf=true` default on that machine, so
// every one of the fifteen was checked out with CRLF. Measured over the shipped
// data:
//
//	as LF    df3bed25a5c5
//	as CRLF  f0ea65c2abb0
//
// The room refused the join with `data_mismatch` and **the data was not
// different**: all fifteen parse to identical values either way, because JSON
// allows \r between tokens and a `\n` inside a string is the two characters `\`
// and `n`, which the rewrite never touches. So the two would have fought the
// same battle.
//
// The two tests below are the two halves of the fix, and they are deliberately
// not one test. One measures the **bytes** — the thing the digest actually
// hashes — and the other measures the **rule** that keeps those bytes the same
// on every machine. Either can fail without the other: a CRLF file committed
// from a checkout with the rule locally overridden reddens the first alone, and
// somebody deleting the rule reddens the second alone.
//
// ⚠️ **Neither of them would have gone red on the author's machine before the
// fix, and that is not a gap in them.** The reporting user's Windows checkout
// is where the CR arrives, so the first test is red *there* and green here —
// which is the right place for it to be red, because that is the machine whose
// binary announces the wrong digest.

// TestNoEmbeddedDataFileCarriesACarriageReturn is the mechanical half of what
// `.gitattributes` promises, measured against the bytes go:embed actually baked
// in rather than against a git configuration.
//
// ⚠️ **It walks the embedded FS rather than dataFiles**, which is this package's
// existing habit (→ digest_test.go, which walks it in both directions rather
// than trusting any one of the three copies of those fifteen names): a file
// added to the go:embed directive and not to dataFiles is still a file this
// binary carries, and its endings are still a thing a platform can rewrite.
// Nothing but the fifteen is in there today — the directive names them one by
// one and `data/assets` is not embedded at all — and the count assertion below
// is what says so.
//
// What it can see: a CR anywhere in any embedded file, whatever put it there —
// a checkout, an editor, a generator writing \r\n.
//
// What it cannot see: anything about the *other* machine. This reads the copy
// compiled into this test binary, so it is a statement about the tree the test
// was built from and about no other.
func TestNoEmbeddedDataFileCarriesACarriageReturn(t *testing.T) {
	seen := 0
	walk := func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		seen++
		raw, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		at := bytes.IndexByte(raw, '\r')
		if at < 0 {
			return nil
		}
		t.Errorf("%s carries a carriage return at byte %d of %d, so this binary digests "+
			"different bytes from one built on a platform that does not — which is a room "+
			"refusing a join with data_mismatch over data that is not different. → "+
			".gitattributes", path, at, len(raw))
		return nil
	}
	if err := fs.WalkDir(files, ".", walk); err != nil {
		t.Fatalf("walk the embedded data: %v", err)
	}
	// A walk that read nothing passes the loop above however many CRs are in the
	// tree, which is the vacuity this package already guards elsewhere. The
	// count is the fifteen the digest is over, so it is checked exactly rather
	// than loosely: a sixteenth file appended to the directive and not to
	// dataFiles is a file outside the digest, which is its own finding.
	if seen != len(dataFiles) {
		t.Errorf("the walk read %d embedded files and dataFiles names %d, so either this "+
			"measures the wrong set or the go:embed directive and dataFiles have drifted",
			seen, len(dataFiles))
	}
}

// TestTheRepositoryPinsItsLineEndings is the half above it cannot be: the rule
// that keeps a checkout from introducing the CR in the first place.
//
// ⚠️ **It lives in this package rather than at the module root because the
// promise is about the bytes THIS package hashes**, and because reading a real
// path from a test here is already the local habit — digest_test.go opens
// `os.DirFS("data")` for the same reason. Go runs a package's tests with the
// package directory as the working directory, which is what makes `../..` the
// module root and not a guess.
//
// What it can see: the rule being deleted, weakened to `text` alone (which
// normalises the repository and leaves the *checkout* to core.autocrlf, so it
// fixes everything except the machine that was refused), or narrowed to a
// pattern that no longer covers everything.
//
// What it cannot see: whether any given machine honours it. A local
// `.git/info/attributes`, a `--no-renormalize` commit or an editor writing CRLF
// straight past git all defeat it, and the test above is what catches the
// result. It also cannot see the thing the file's own comment has to say
// instead: adding the rule does **not** repair a checkout that already holds
// CRLF — measured, a pull leaves such a tree exactly as it was.
func TestTheRepositoryPinsItsLineEndings(t *testing.T) {
	path := filepath.Join("..", "..", ".gitattributes")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v — without it a Windows checkout rewrites every embedded "+
			"data file to CRLF and this binary announces a data digest no other "+
			"platform produces", path, err)
	}
	pinned := false
	for _, line := range strings.Split(string(raw), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] != "*" {
			continue
		}
		attributes := make(map[string]bool, len(fields))
		for _, attribute := range fields[1:] {
			attributes[attribute] = true
		}
		// Both halves, because they answer different questions: eol=lf is what
		// pins the working tree on a machine whose core.eol is native, and
		// text=auto is what leaves a binary file added later alone instead of
		// corrupting it on checkout.
		if attributes["eol=lf"] && attributes["text=auto"] {
			pinned = true
		}
	}
	if !pinned {
		t.Errorf("%s has no `* text=auto eol=lf`, so a checkout is free to rewrite the "+
			"fifteen files DataDigest hashes and refuse an otherwise identical peer at a "+
			"room's gate:\n%s", path, raw)
	}
}
