package socket

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestTheTransportOwnsTheClockAndPrintsNothing is this package's own half of the
// arrangement internal/room and internal/wire are held to, and it makes two
// claims that pull in opposite directions.
//
// # The clock lives here, and the positive claim is the point
//
// `internal/room` and `internal/wire` both **refuse** to import `time`, held by
// AST walks over their own directories, because whoever owns the transport owns
// the countdown. This package is that owner, so the ban's counterpart is that
// `time` is imported *here* — and a positive claim is worth making because the
// alternative failure is silent: somebody could move the countdown into a fourth
// package and both existing bans would still pass.
//
// # And nothing here prints
//
// ⚠️ The one output this package has is the caller's `Options.Report`, which
// takes an **error** and is handed one only where the code says so. That matters
// because the first message on every connection is a wire.Hello and a hello
// carries the room's password in the clear: `log.Printf("%v", hello)` is safe by
// the type (wire.Password redacts itself), and `log.Printf("%s", raw)` on the
// bytes of one that did not decode is not.
//
// So a logger cannot be reached at all — no `log`, no `log/slog`, no `os` (which
// is where a writer would come from) — and `fmt` is refused its printing verbs
// by name, which an import ban cannot say. `fmt` itself stays, because every
// error in the package is built with it.
//
// ⚠️ **This is a structural guard and not the whole of the claim.**
// TestAWrongPasswordIsRefusedAndNeverPrinted is the behavioural half: it drives a
// wrong password and a malformed hello over real connections and reads the sink
// for the characters. Neither test alone is enough — this one cannot see a
// password handed to `Report`, and that one cannot see a print nothing happened
// to reach on the day it ran.
func TestTheTransportOwnsTheClockAndPrintsNothing(t *testing.T) {
	banned := map[string]string{
		"log":      "the only output here is the caller's Report, which takes an error",
		"log/slog": "the only output here is the caller's Report, which takes an error",
		"os":       "a transport writes to a socket; a writer from here is where a logger comes in",
		// ⚠️ `net` and `net/http` are imported and are deliberately not on this
		// list — this is the package they belong to. That is the whole of the
		// ban in internal/room and internal/wire: the dialling and the listening
		// live in one boundary, and this is it.
	}
	// The printing verbs, by selector, because `fmt` is needed for every error
	// in the package and an import ban cannot tell fmt.Errorf from fmt.Println.
	unprintable := map[string]bool{
		"Print": true, "Printf": true, "Println": true,
		"Fprint": true, "Fprintf": true, "Fprintln": true,
	}
	scanned, clocks, prints := 0, 0, 0
	for _, name := range packageSources(t) {
		scanned++
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
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
			if path == "time" {
				clocks++
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || !unprintable[selector.Sel.Name] {
				return true
			}
			from, named := selector.X.(*ast.Ident)
			if !named || from.Name != "fmt" {
				return true
			}
			prints++
			t.Errorf("%s:%d calls fmt.%s: the only output here is the caller's Report, "+
				"and the bytes of a hello carry the room's password",
				name, fileSet.Position(selector.Pos()).Line, selector.Sel.Name)
			return true
		})
	}
	// A walk that read no files would pass whether or not either claim held,
	// which is the failure mode every copy of this shape records: the scan is the
	// measurement, so the measurement has to say it happened.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	// The positive half. If this goes to nought, the countdown has moved
	// somewhere neither this ban nor the two it is the counterpart of can see.
	if clocks == 0 {
		t.Error("no file in this package imports time, and this package is where the PvP clock lives: " +
			"internal/room and internal/wire both refuse it on the grounds that the transport owns the countdown")
	}
	t.Logf("scanned %d source files, %d of them read a clock, %d printing calls", scanned, clocks, prints)
}

// packageSources is every non-test source file of this package, which is the
// only directory any walk here reads — the shape internal/room, internal/wire
// and both TUI clients already use, so it is per-package by construction and
// does not follow this code anywhere.
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
