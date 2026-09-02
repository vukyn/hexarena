package wire

import (
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestTheProtocolCannotReadAClock is the design record's "the clock is not part
// of the battle" made mechanical, and it is the sharpest form the claim has: not
// "no field is a timestamp", which is a judgement about names, but **the concept
// cannot enter this package at all**.
//
// The walk is in the shape of cmd/hexarena-tui's TestNoScreenHoldsItsOwnWording:
// it reads `os.ReadDir(".")`, its own package directory and nothing else, so it
// is per-package by construction and does not follow this code anywhere. That is
// a property rather than a limitation — a room, which is the next item and does
// need a clock, must not inherit this ban, and a copy of this walk in that
// package would be exactly wrong.
//
// What it protects. A turn allowance is **room configuration** and rides on
// Welcome as seconds in an int; what enters the battle when it runs out is a
// Pass, never a timestamp, never a duration, never a reading of a clock. So a
// PvP log stays exactly as verifiable as one from a battle nobody was waiting
// on, and `--verify` cannot tell a timed-out match from any other. A
// time.Duration would also JSON-encode as a count of nanoseconds, which is
// unreadable in a golden and unreadable to anything that is not Go.
func TestTheProtocolCannotReadAClock(t *testing.T) {
	banned := map[string]string{
		"time":                         "a duration or a timestamp on the wire would put a clock reading in the battle's record",
		"math/rand":                    "randomness in this engine comes only from a passed-in rng.Source",
		"math/rand/v2":                 "randomness in this engine comes only from a passed-in rng.Source",
		"os":                           "the protocol is the format and nothing else: no I/O",
		"net/http":                     "the transport is the next item and is confined to its own boundary",
		"github.com/gorilla/websocket": "the transport is the next item and is confined to its own boundary",
	}
	scanned := 0
	for _, entry := range mustReadPackageDir(t) {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
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
	// the failure mode both existing copies of this shape record: the scan is
	// the measurement, so the measurement has to say it happened.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	// And the whole point of the seconds-as-an-int decision, stated where the
	// import ban cannot state it: the allowance is a plain number, so a golden
	// reads 90 rather than 90000000000.
	if got := reflectAllowanceKind(); got != "int" {
		t.Errorf("Welcome.Allowance is a %s; it is seconds as an int, because a "+
			"time.Duration JSON-encodes as a nanosecond count", got)
	}
	t.Logf("scanned %d source files; no clock can enter this package", scanned)
}

// TestNothingInTheProtocolIsThirdParty is the "one stdlib-only package" half of
// the brief, and it is worth holding mechanically for the reason the layer rule
// is: a dependency that reaches the protocol has to answer "what happens to a
// replay when this dependency changes its mind about encoding", and the answer
// for every third-party library is that nobody knows.
//
// Its own module's packages are allowed and are the whole design — placement,
// battle, hex and seed already declare the shapes this protocol carries, and a
// second declaration of any of them would drift.
func TestNothingInTheProtocolIsThirdParty(t *testing.T) {
	const own = "github.com/vukyn/hexarena/"
	scanned, foreign := 0, 0
	for _, entry := range mustReadPackageDir(t) {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
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
			if strings.HasPrefix(path, own) {
				continue
			}
			// A standard library path has no dot in its first element; every
			// module path has one.
			if first, _, _ := strings.Cut(path, "/"); strings.Contains(first, ".") {
				foreign++
				t.Errorf("%s imports %q, which is neither the standard library nor this module", name, path)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files, %d foreign imports", scanned, foreign)
}

// mustReadPackageDir is this package's own directory, read the way both other
// copies of this walker read theirs.
func mustReadPackageDir(t *testing.T) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	return entries
}

// readPackageSource is every non-test source file in this package, concatenated,
// for the one test that has to read the source rather than the behaviour.
func readPackageSource(t *testing.T) string {
	t.Helper()
	var all strings.Builder
	for _, entry := range mustReadPackageDir(t) {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		all.Write(raw)
	}
	if all.Len() == 0 {
		t.Fatal("read no source, so this measures nothing")
	}
	return all.String()
}

// reflectAllowanceKind is what type the turn allowance actually is, read off the
// struct rather than off the source, because this is the one half of "the clock
// is not part of the battle" that an import ban cannot state: `time` could stay
// unimported while the field became an int64 of nanoseconds.
func reflectAllowanceKind() string {
	field, found := reflect.TypeFor[Welcome]().FieldByName("Allowance")
	if !found {
		return "absent"
	}
	return field.Type.String()
}
