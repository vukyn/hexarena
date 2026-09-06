package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// lockTakers is every name that reaches the mirror's own RWMutex, and it is a
// written-down list because the walk below cannot follow a call across a package
// boundary.
//
// The eighteen exported Mirror methods that take the lock are derived rather than
// remembered — `grep '^func (m \*Mirror) [A-Z]' internal/socket/*.go` and keep the
// ones whose body locks — and `seat` is this client's own forwarder onto one of
// them (session.seat takes the session's mutex, releases it, then calls
// Client.Seat, which is Mirror.Seat). session.pool is deliberately **not** here:
// it takes the session's mutex and nothing else.
var lockTakers = map[string]string{
	"seat": "session.seat forwards to Client.Seat, which is Mirror.Seat",

	"Asking": "Mirror.Asking", "Battle": "Mirror.Battle", "Capped": "Mirror.Capped",
	"Closure": "Mirror.Closure", "Compared": "Mirror.Compared", "Decide": "Mirror.Decide",
	"DecideDraft": "Mirror.DecideDraft", "Draft": "Mirror.Draft",
	"DraftAsking": "Mirror.DraftAsking", "Events": "Mirror.Events", "Fought": "Mirror.Fought",
	"Over": "Mirror.Over", "Receive": "Mirror.Receive", "Refusals": "Mirror.Refusals",
	"Seat": "Mirror.Seat", "Side": "Mirror.Side", "Stale": "Mirror.Stale",
	"Welcome": "Mirror.Welcome",
}

// TestNoReadCallbackTakesTheMirrorsLockAgain is a lock-discipline rule that cost
// a bug on the day it was written, and that **nothing else in this repository can
// see**.
//
// ⚠️ session.read holds the mirror's RWMutex read lock across the callback it is
// handed. A second RLock on the same mutex from inside that callback is fine on
// its own and **self-deadlocks the moment a writer is queued**, because Go admits
// a waiting writer ahead of new readers — so a Receive arriving between the two
// locks hangs the client. Measured at about one run in ten with `model.stepped`
// reading the seat inside the callback, and ⚠️ **the whole suite was green and
// `-race` was clean**: a self-deadlock is not a data race, and every test that
// drives a match lets its own goroutines finish, so nothing was waiting on the
// lock at the moment it mattered.
//
// It is a walk rather than a runtime test for the reason the clock allowlist is
// one: reproducing a one-in-ten hang costs a bounded loop and a re-run, while the
// *shape* — a lock-taking call inside the literal — is exactly what a reader
// writes by accident and exactly what an AST can see every time.
//
// The rule is narrower than "no method calls": session.pool is called inside a
// callback on purpose and is safe, because it takes the session's own mutex and
// never the mirror's. What is banned is the eighteen Mirror methods that lock,
// and this client's one forwarder onto them. → lockTakers.
//
// ⚠️ The value the bug wanted is on the sight already: wire.Welcome carries the
// seat and Sight carries the welcome, so `sight.Welcome.Seat` needs no lock at
// all. A callback that finds itself wanting one of these should look there first.
func TestNoReadCallbackTakesTheMirrorsLockAgain(t *testing.T) {
	set := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read this package's own directory: %v", err)
	}
	var callbacks, scanned int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, entry.Name(), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		scanned++
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			outer, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || outer.Sel.Name != "read" {
				return true
			}
			for _, argument := range call.Args {
				literal, ok := argument.(*ast.FuncLit)
				if !ok {
					continue
				}
				callbacks++
				ast.Inspect(literal.Body, func(inner ast.Node) bool {
					inside, ok := inner.(*ast.CallExpr)
					if !ok {
						return true
					}
					named, ok := inside.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					because, banned := lockTakers[named.Sel.Name]
					if !banned {
						return true
					}
					t.Errorf("%s:%d calls %s inside a read callback, and %s takes the mirror's "+
						"RWMutex read lock the callback is already inside: a second RLock "+
						"self-deadlocks the moment a writer is queued. The seat is on "+
						"sight.Welcome.Seat, which needs no lock; anything else has to be read "+
						"before the callback",
						entry.Name(), set.Position(inside.Pos()).Line, named.Sel.Name, because)
					return true
				})
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("no non-test source file was scanned, so this walk measured nothing")
	}
	if callbacks == 0 {
		t.Fatalf("no read callback was found in %d source files, so this walk measured nothing "+
			"— session.read was renamed or its callers changed shape and this test has to follow",
			scanned)
	}
	t.Logf("scanned %d source files; %d read callbacks, none taking the mirror's lock again",
		scanned, callbacks)
}
