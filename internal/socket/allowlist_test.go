package socket

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// # Where a clock may be read, for the whole module
//
// ⚠️ **The three clock bans in this repository are directory-scoped, so a fourth
// package is invisible to all three.** internal/room's TestTheRoomReadsNoClock,
// internal/wire's TestTheProtocolCannotReadAClock and this package's own
// TestTheTransportOwnsTheClockAndPrintsNothing each read os.ReadDir(".") — their
// own package directory and nothing else — which is exactly right for what each
// of them claims and says nothing at all about anywhere else. The countdown in
// cmd/hexarena-tui is the fourth package, and it landed with all three of them
// green.
//
// So this is the module's **positive** claim, the same shape the transport's own
// test makes one package wide: walk every non-test source file in the module,
// find every one that can read a clock, and hold that set equal to a written
// allowlist with a reason against each entry. Adding a clock anywhere then costs
// a line here — a deliberate act with a red test in the way — rather than being
// something nobody notices.
//
// It lives in internal/socket because this is the package that **owns** the
// clock: "and here is everywhere else that may have one" reads correctly beside
// the timer a room's allowance is actually enforced by.
//
// ## What counts as reading one, and why an import is not enough
//
// ⚠️ **A file can read a clock without importing `time`, and one in this package
// does.** connection.go calls context.WithTimeout with a duration off a struct
// somebody else built — a deadline is a clock whether or not the word "time"
// appears in the imports — so the walk looks for the calls as well as the
// import. socket.Allowance is on the list of calls for the same reason: it is
// the one conversion from the protocol's seconds into a duration, so a caller of
// it is doing clock arithmetic by definition.
//
// ## ⚠️ Dot-directories are skipped, and that is not tidiness
//
// .claude/worktrees holds **other checkouts of this repository**. Descending
// into one would count the wrong files — a clock on another branch is not a
// clock here — and would fail outright whenever a checkout is sitting on an
// unresolved conflict, which is this branch turning red for work on another. The
// rule is copied from internal/i18n's TestNoKeyIsOrphaned, which is the
// precedent for a module-root walk here.

// clockAllowlist is every non-test file in the module that may read a clock, and
// why it has one. Paths are relative to the module root, with forward slashes.
var clockAllowlist = map[string]string{
	"internal/socket/socket.go": "the transport's own timings — the keepalive, the close " +
		"threshold and the write deadline — and Allowance, the one conversion in the " +
		"module from the protocol's seconds-as-an-int into a duration",
	"internal/socket/table.go": "the per-prompt timer a room's allowance is actually " +
		"enforced by, and the keepalive ticker: the room is told a seat timed out, it " +
		"never asks what time it is",
	"internal/socket/connection.go": "the write deadline and the bounded close handshake, " +
		"both context.WithTimeout over Timings — a clock with no time import at all",
	"internal/socket/server.go": "the settling poll a bounded shutdown measures itself with",
	"cmd/hexarena-host/main.go": "the shutdown grace: the host is the process that decides " +
		"how long a stop waits for two people's sockets to close",
	"cmd/hexarena-tui/clock.go": "the game client's countdown and the chooser's third arm. " +
		"⚠️ It is one file on purpose — this list is only worth keeping if the answer to " +
		"'where is the clock in this package' stays short",
}

// clockCalls is the calls that read a clock without necessarily importing one.
var clockCalls = map[string]string{
	"context.WithTimeout":  "a deadline",
	"context.WithDeadline": "a deadline",
	"tea.Tick":             "a timer",
	"tea.Every":            "a timer",
	"socket.Allowance":     "the protocol's seconds turned into a duration",
}

// TestEveryClockInTheModuleIsOnTheAllowlist is the walk.
//
// ⚠️ **The count is asserted as well as the set**, because a walk that read
// nothing agrees with any allowlist at all — including the two lines about
// internal/screen and cmd/hexarena-tui that this whole feature turns on. The
// same guard is written into every copy of this shape in the repository.
//
// *Sees:* a `time` import in a package that has none today — internal/screen and
// internal/core most of all, which is where the layer rule lives; a deadline
// taken out with no import to show for it; the countdown moved out of clock.go
// into the rest of its client.
// *Cannot see:* a clock read through a helper in another package that this list
// has never heard of. socket.Allowance is on the call list precisely because it
// is the one such helper that exists.
func TestEveryClockInTheModuleIsOnTheAllowlist(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("find the module root: %v", err)
	}
	scanned := 0
	found := map[string]string{}
	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		for _, imported := range file.Imports {
			quoted, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if quoted == "time" {
				found[relative] = "imports time"
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, isSelector := node.(*ast.SelectorExpr)
			if !isSelector || selector.Sel == nil {
				return true
			}
			from, named := selector.X.(*ast.Ident)
			if !named {
				return true
			}
			if what, reads := clockCalls[from.Name+"."+selector.Sel.Name]; reads {
				found[relative] = from.Name + "." + selector.Sel.Name + ", which is " + what
			}
			return true
		})
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatalf("walk the module: %v", err)
	}
	// A walk that read no files, or one whose predicate has stopped matching
	// anything, agrees with every allowlist there is.
	if scanned == 0 {
		t.Fatal("the walk read no source files, so it measures nothing")
	}
	if len(found) == 0 {
		t.Fatal("the walk found no clock in the whole module, and the transport is built " +
			"on one: it is measuring nothing")
	}
	if len(found) != len(clockAllowlist) {
		t.Errorf("%d files in the module read a clock and the allowlist has %d entries",
			len(found), len(clockAllowlist))
	}
	for _, name := range slices.Sorted(maps.Keys(found)) {
		if _, allowed := clockAllowlist[name]; !allowed {
			t.Errorf("%s reads a clock (%s) and is not on the allowlist. A clock outside "+
				"the transport is a decision: put it on the list with the reason, or put "+
				"the reading where the clock already is", name, found[name])
		}
	}
	for _, name := range slices.Sorted(maps.Keys(clockAllowlist)) {
		if _, reads := found[name]; !reads {
			t.Errorf("%s is on the allowlist for %q and reads no clock at all; an entry "+
				"for a clock that is gone is a line nobody will ever fail on",
				name, clockAllowlist[name])
		}
	}
	t.Logf("walked %d source files, %d of them read a clock", scanned, len(found))
}

// TestTheClocklessPackagesAreStillClockless names the packages the allowlist is
// kept for, so a failure says what was lost rather than only that a count moved.
//
// ⚠️ **internal/screen is the one this feature could most easily have broken.**
// It draws the countdown, and it draws it out of two counts of seconds handed in
// already computed — the same arrangement internal/room is under one layer down,
// where Allowance is a number the room carries and hands to its clients rather
// than anything it ever reads. A screen that worked out its own remaining time
// would be a second answer to when a turn opened.
func TestTheClocklessPackagesAreStillClockless(t *testing.T) {
	clockless := map[string]string{
		"internal/screen": "a screen is a pure function of what it is handed, and the " +
			"countdown is handed in as seconds",
		"internal/core":  "the layer rule: no clock, no randomness but an rng.Source, no floats",
		"internal/room":  "a timeout is an INPUT to the room, never a reading it takes",
		"internal/wire":  "the protocol carries an allowance, never a moment",
		"internal/tui":   "a renderer reads the event log and nothing else",
		"internal/i18n":  "a language book is a table",
		"internal/forge": "authoring is files and validation",
		"internal/seed":  "the embedded data is bytes",
	}
	for _, name := range slices.Sorted(maps.Keys(clockAllowlist)) {
		for prefix, because := range clockless {
			if strings.HasPrefix(name, prefix+"/") {
				t.Errorf("%s is on the clock allowlist, and %s may not read one: %s",
					name, prefix, because)
			}
		}
	}
}
