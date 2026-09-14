package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestALogLargerThanALogCanBeIsRefused is the bound, measured at its own edge.
//
// ⚠️ **The fixture is one byte over, not a megabyte over**, because an off-by-one
// in a limit is exactly the mistake a generous fixture cannot see: reading
// `longestLog` bytes with `io.LimitReader(file, longestLog)` and comparing
// `len(raw) > longestLog` would accept every file in the world, and a 100MB
// fixture would pass that broken version too.
func TestALogLargerThanALogCanBeIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.json")
	if err := os.WriteFile(path, make([]byte, longestLog+1), 0o600); err != nil {
		t.Fatalf("write the oversized fixture: %v", err)
	}
	_, err := readLog(path)
	if err == nil {
		t.Fatal("a file one byte over the limit was read")
	}
	// The refusal names the number, because a reader who cannot see the limit
	// cannot tell an honestly long battle from the wrong path.
	if !strings.Contains(err.Error(), strconv.Itoa(longestLog)) {
		t.Errorf("the refusal does not say what the limit is: %v", err)
	}
	// ⚠️ **And it names the file, which is asserted HERE rather than beside the
	// missing-file case for a measured reason.** This sentence is built from
	// nothing — no wrapped error under it — so the path is in it only because
	// this code puts it there. The open failure's path comes out of the standard
	// library's own `*fs.PathError`; see TestAMissingLogIsStillClassifiable.
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal does not name the file: %v", err)
	}
}

// TestALogExactlyAtTheLimitIsStillRead is the other side of the same edge. A
// limit that refused the largest legal file would be a bug wearing a bound's
// clothes, and it is the failure mode of reading exactly `longestLog` bytes
// instead of one more.
func TestALogExactlyAtTheLimitIsStillRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exact.json")
	if err := os.WriteFile(path, make([]byte, longestLog), 0o600); err != nil {
		t.Fatalf("write the at-the-limit fixture: %v", err)
	}
	raw, err := readLog(path)
	if err != nil {
		t.Fatalf("a file exactly at the limit was refused: %v", err)
	}
	if len(raw) != longestLog {
		t.Errorf("read %d bytes of a %d byte file", len(raw), longestLog)
	}
}

// TestAnOrdinaryLogIsUnaffected keeps the cap honest about its own size. The
// number in readLog's doc is derived from a measurement — 67,447 bytes for 255
// events — and a real log has to sit far enough under it that no honest battle is
// ever refused.
func TestAnOrdinaryLogIsUnaffected(t *testing.T) {
	path := writeLog(t, honestLog())
	raw, err := readLog(path)
	if err != nil {
		t.Fatalf("an ordinary log was refused: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("an ordinary log read as nothing")
	}
	if len(raw) > longestLog/1000 {
		t.Errorf("a real log is %d bytes against a %d byte limit, which is less headroom "+
			"than readLog's derivation claims — re-derive the number rather than widening it",
			len(raw), longestLog)
	}
}

// TestAMissingLogIsStillClassifiable keeps what routing through a new function
// could actually lose: the wrap.
//
// ⚠️ **This test was first written as "the error still names the file", and a
// mutation proved it was measuring the standard library.** Deleting the path from
// the wrapper left it GREEN — because `os.Open` returns a `*fs.PathError` that
// names the file itself, so the assertion held whatever this code did with it.
// That is the exact shape of vacuous test this repository keeps a record of, and
// it is worth the paragraph because the assertion *looked* like it was about this
// function.
//
// What is left is a claim about this code: the cause is still reachable through
// `errors.Is`, which a `%v` or a `fmt.Errorf("...%s", err)` would have flattened
// into a string and a caller could no longer ask about.
func TestAMissingLogIsStillClassifiable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-here.json")
	_, err := readLog(path)
	if err == nil {
		t.Fatal("a missing file read successfully")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the wrap flattened the cause, so a caller cannot tell a missing "+
			"file from any other failure: %v", err)
	}
	// Kept, but as the standard library's guarantee rather than this code's: if a
	// future Go stops putting the path in a PathError, the line above is the one
	// that still holds.
	if !strings.Contains(err.Error(), path) {
		t.Errorf("neither this wrapper nor os.Open names the file: %v", err)
	}
}
