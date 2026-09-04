package draft_test

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestTheDraftReadsNoClock is internal/room's TestTheRoomReadsNoClock copied
// into this package, deliberately and not by habit.
//
// A draft is a **pure function of the decisions taken** — that is the whole of
// why the mirror trick transfers to it, why a client can replay a draft from the
// decisions alone, and why the server needs to send only those decisions. A
// clock or a source of entropy in here would break it for exactly the reason it
// would break a room: two peers replaying the same decisions would stop
// agreeing, and the draft's record would stop being a complete one. The timeout
// a draft runs under is an **input** the transport reports, the same way a
// room's is.
//
// ⚠️ **The copy is required rather than optional, and this repository has
// already been bitten by the reason.** Every clock ban here reads
// os.ReadDir("."), its own package directory and nothing else, which is exactly
// right for what each one claims and says nothing at all about anywhere else. A
// new package is invisible to all of them by construction: the countdown in
// cmd/hexarena-tui landed with internal/room's, internal/wire's and
// internal/socket's copies all green, which is what internal/socket's
// module-wide TestEveryClockInTheModuleIsOnTheAllowlist was then written to
// catch. So a package that means to stay clockless brings its own ban with it,
// and the module-wide allowlist is the second net rather than the first.
//
// *Sees:* a `time` import, an `os` read, a socket dialled, entropy drawn — any
// of them in a non-test file of this package.
// *Cannot see:* a clock reached through a helper in another package, which is
// what internal/socket's call list is for; and anything at all outside this
// directory.
func TestTheDraftReadsNoClock(t *testing.T) {
	banned := map[string]string{
		"time": "a draft's timeout is an input the transport reports, so nothing here " +
			"reads a clock or holds a duration",
		"math/rand": "a draft is a pure function of the decisions taken: there is no " +
			"randomness in it to draw",
		"math/rand/v2": "a draft is a pure function of the decisions taken: there is no " +
			"randomness in it to draw",
		// ⚠️ crypto/sha256 is deliberately absent for the reason internal/room
		// gives: hashing is a pure function, where crypto/rand is a source of
		// entropy, and only one of the two can make a draft stop replaying.
		"crypto/rand": "a draft is a pure function of the decisions taken, so nothing " +
			"here draws entropy",
		"os": "the pool is handed in as characters and the arithmetic is arithmetic: no I/O. " +
			"internal/seed owns the embedded data and its caller owns any real file",
		"net":      "the transport lives in internal/socket and is confined to its own boundary",
		"net/http": "the transport lives in internal/socket and is confined to its own boundary",
		"github.com/coder/websocket": "the transport lives in internal/socket and is " +
			"confined to its own boundary",
	}
	scanned := 0
	for _, name := range packageSources(t) {
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("%s: unquote %s: %v", name, imported.Path.Value, err)
			}
			if because, refused := banned[path]; refused {
				t.Errorf("%s imports %q: %s", name, path, because)
			}
		}
	}
	// A walk that read no files would pass whether or not the ban held, which is
	// the failure mode every copy of this shape records: the scan is the
	// measurement, so the measurement has to say it happened.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files; no clock can enter this package", scanned)
}

// packageSources is every non-test source file of this package, which is the
// only directory any walk here reads.
func packageSources(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, name)
	}
	return out
}
