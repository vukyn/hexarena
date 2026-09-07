package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// makefilePath is the repository's Makefile, read from this package rather than
// from a package of its own because there is no package at the repository root
// and this is the command whose target was the one left out. → the test below.
const makefilePath = "../../Makefile"

// targetLine is a rule's left-hand side: a name at the start of a line, followed
// by a colon. It deliberately does not match a variable assignment (`X := y`),
// an indented recipe line, or a `.PHONY:`-style special target, which is what
// the leading character class and the exclusion below are for.
var targetLine = regexp.MustCompile(`(?m)^([a-z][a-z0-9-]*):(?:[^=]|$)`)

// TestEveryMakefileTargetIsPhony holds the one rule a Makefile of thin wrappers
// owes: every target here names a *command to run*, not a file to build, so
// every one of them belongs in .PHONY.
//
// ⚠️ **It is not a style rule.** A target absent from .PHONY stops working the
// day a file or directory of that name appears beside the Makefile — make sees
// something up to date and prints "nothing to be done", silently, instead of
// running the recipe. `host` was the one that was missing, and it was one
// `mkdir host` away from a build command that quietly did nothing. `build`
// writes into bin/ today, but nothing stops a future target from being named
// after a directory somebody adds.
//
// The count is what stops the walk from passing on a Makefile it failed to
// parse: a regexp that matched nothing would otherwise agree that every target
// it found is phony.
func TestEveryMakefileTargetIsPhony(t *testing.T) {
	raw, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("read %s: %v", makefilePath, err)
	}
	text := string(raw)

	declared := phonyTargets(t, text)
	if len(declared) == 0 {
		t.Fatal("the Makefile declares no .PHONY targets at all, so this walk is looking for " +
			"the wrong shape and would agree with anything")
	}

	found := 0
	for _, match := range targetLine.FindAllStringSubmatch(text, -1) {
		name := match[1]
		found++
		if !declared[name] {
			t.Errorf("the Makefile's %q target is not in .PHONY: every target here runs a command "+
				"rather than building a file of its own name, and one left out stops running the "+
				"day a file or directory called %q appears beside the Makefile", name, name)
		}
	}
	if found == 0 {
		t.Fatal("no targets were found in the Makefile, so nothing above was checked")
	}
	// Both directions: a .PHONY entry naming a target that no longer exists is a
	// line nobody will delete, and it is the half a walk over the targets cannot
	// see.
	for name := range declared {
		if !strings.Contains(text, "\n"+name+":") {
			t.Errorf(".PHONY names %q and no target of that name exists", name)
		}
	}
	t.Logf("checked %d targets against %d .PHONY entries", found, len(declared))
}

// phonyTargets is every name on the .PHONY line, which the Makefile keeps as one
// line; a second .PHONY line would be read here too.
func phonyTargets(t *testing.T, text string) map[string]bool {
	t.Helper()
	declared := make(map[string]bool)
	for _, line := range strings.Split(text, "\n") {
		rest, isPhony := strings.CutPrefix(line, ".PHONY:")
		if !isPhony {
			continue
		}
		for _, name := range strings.Fields(rest) {
			declared[name] = true
		}
	}
	return declared
}
