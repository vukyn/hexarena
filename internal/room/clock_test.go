package room_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestTheRoomReadsNoClock is the design record's "a timeout is an input, not a
// reading" made mechanical, and it is the sharpest form the claim has: not "no
// field is a timestamp", which is a judgement about names, but **the concept
// cannot enter this package at all**.
//
// ⚠️ internal/wire's own copy of this walk says in its comment that a room "does
// need a clock" and that a copy of the ban here "would be exactly wrong". That
// expectation turned out to be wrong, which is why this file exists: handing the
// timeout in as an input costs nothing — the transport already owns the
// countdown, because it owns the connection — and it buys three things a room
// with a clock in it cannot have. Three consecutive timeouts forfeit becomes
// pure counting, testable with no clock anywhere near it. A whole match plays
// out in-process at the speed of the engine rather than in real seconds. And a
// PvP log stays exactly as verifiable as one from a battle nobody was waiting
// on, because what enters the battle is a Pass with a constant reason.
//
// The walk reads os.ReadDir("."), this package's directory and nothing else, so
// it is per-package by construction and does not follow this code anywhere —
// the same shape internal/wire and both TUI clients use.
func TestTheRoomReadsNoClock(t *testing.T) {
	banned := map[string]string{
		"time":         "a timeout is an input the transport reports, so the room never reads a clock or holds a duration",
		"math/rand":    "the only randomness in a match is the seed, and every battle's seed is derived from it",
		"math/rand/v2": "the only randomness in a match is the seed, and every battle's seed is derived from it",
		// ⚠️ crypto/sha256 is imported and is deliberately not on this list:
		// SeedFor hashes a pair of integers, which is a pure function. crypto/rand
		// is a source of entropy, which is the thing being banned — the two sit
		// beside each other in the standard library and only one of them can make
		// a match irreproducible.
		"crypto/rand":                  "a match's every battle is derived from the room's one seed, so nothing here draws entropy",
		"os":                           "the room is a state machine over messages: no I/O",
		"net":                          "the transport is its own item and is confined to its own boundary",
		"net/http":                     "the transport is its own item and is confined to its own boundary",
		"sync":                         "a room owns its battle in one goroutine and shares it with nothing; the registry takes the mutex",
		"github.com/gorilla/websocket": "the transport is its own item and is confined to its own boundary",
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
	t.Logf("scanned %d source files; no clock and no goroutine can enter this package", scanned)
}

// TestNothingHereDrainsTheBattle is the invariant the server has to respect,
// held mechanically because it is the sort of thing a later edit does on
// autopilot: Drain is the call every other consumer of a battle in this
// repository makes, there are 261 of them, and reaching for it here is one
// keystroke.
//
// ⚠️ Drain **empties** its consumer's cursor into the battle itself, so it is
// single-consumer by construction — and a room has two players, a log to write
// and spectators later. Whichever consumer called it would silently take the
// events the others were about to read. So the room drains once, into the
// append-only record the battle already keeps, and reads it through Since with a
// cursor of its own.
//
// The walk looks for the **selector** rather than the string, so a comment
// explaining why Drain is not used does not redden it and a `fight.Drain()`
// anywhere does.
func TestNothingHereDrainsTheBattle(t *testing.T) {
	scanned, calls := 0, 0
	for _, name := range packageSources(t) {
		scanned++
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || selector.Sel.Name != "Drain" {
				return true
			}
			calls++
			t.Errorf("%s:%d reads the battle with Drain, which empties the buffer: "+
				"a room has two players and a log, so it holds a cursor and calls Since",
				name, fileSet.Position(selector.Pos()).Line)
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files, %d Drain references", scanned, calls)
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
