package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// TestNoScreenHoldsItsOwnWording is the rule that keeps the two languages
// honest: a sentence written here would exist in one of them only, and nothing
// would notice.
//
// ⚠️ **It is the third copy of this walker and it had to be written rather than
// reused, for the reason the second one did.** Every version of it reads
// `os.ReadDir(".")` — its own package directory and nothing else — so the ban
// does not follow code into a new package. #205 recorded that gap when six
// screens moved into internal/screen and added the second copy; this package is
// a whole client, so it arrives with the third rather than being noticed later.
//
// ⚠️ **The golden cannot stand in for it and is not a weaker version of it.** A
// literal moved out of a package renders identically, so `screens.golden` would
// be byte for byte what it was. The two are complements: the golden holds what
// is drawn, this holds where the words that get drawn are allowed to live.
//
// A duplicate rather than an exported helper, deliberately: this is a property
// of a *package's own source*, so the thing that must be per-package is which
// directory is read, and reaching into another package's test file to share four
// small functions would be a dependency between three suites for no saving.
//
// The scan treats a string as something a person would read when it holds two
// words of three letters or more with a space between them, or when it is a
// shouted word — those are the two shapes every line a screen draws has. Format
// skeletons, key names and import paths have neither, which is why they can
// stay.
func TestNoScreenHoldsItsOwnWording(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		excused := environmentNames(file)
		ast.Inspect(file, func(node ast.Node) bool {
			literal, isLiteral := node.(*ast.BasicLit)
			if !isLiteral || literal.Kind != token.STRING || excused[literal.Pos()] {
				return true
			}
			text, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			if reason := readsLikeProse(text); reason != "" {
				t.Errorf("%s holds %q, which %s — put it in internal/i18n", name, text, reason)
			}
			return true
		})
	}
	// A walker that read no files would pass this whether or not a screen held a
	// sentence, which is the failure mode #199 recorded for the client copy of
	// it: the scan is the measurement, so the measurement has to say it happened.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files for wording", scanned)
}

// TestTheWordingWalkerRecognisesProse is the other half of the walk, and the
// half a green run cannot give.
//
// ⚠️ **A clean scan over a package holding no sentences is indistinguishable
// from a scan that recognises nothing.** The `scanned == 0` guard above catches
// a walker that read no files; it says nothing about one whose predicate has
// stopped matching — a `readsLikeProse` returning empty for everything passes
// the whole file. So the predicate is asserted against literals of both shapes
// it is written for, plus the three shapes it must **not** flag, because a
// walker that objected to a format skeleton would be one somebody switched off.
func TestTheWordingWalkerRecognisesProse(t *testing.T) {
	prose := []string{
		"the window is too short for",
		"cửa sổ quá ngắn cho",
		"MISSING",
	}
	for _, text := range prose {
		if readsLikeProse(text) == "" {
			t.Errorf("the walker does not read %q as wording, so it would let one through", text)
		}
	}
	// Every one of these really is in this package's source, which is what makes
	// the list a measurement rather than a guess about what a skeleton looks like.
	notProse := []string{
		"hexarena-tui", "%-24s %-8s %s", "ctrl+l", "data", "  ", "\n", "up", "?",
	}
	for _, text := range notProse {
		if reason := readsLikeProse(text); reason != "" {
			t.Errorf("the walker reads %q as wording (%s), so it flags a format skeleton",
				text, reason)
		}
	}
}

// environmentNames is the literals handed to os.Getenv.
//
// NO_COLOR and TERM are shouted words that nobody reads off a screen: they are
// the names of variables, and recognising them by where they are used rather
// than by a list means a new one needs no maintenance here.
func environmentNames(file *ast.File) map[token.Pos]bool {
	excused := make(map[token.Pos]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		selector, isSelector := call.Fun.(*ast.SelectorExpr)
		if !isSelector || selector.Sel.Name != "Getenv" {
			return true
		}
		for _, argument := range call.Args {
			if literal, isLiteral := argument.(*ast.BasicLit); isLiteral {
				excused[literal.Pos()] = true
			}
		}
		return true
	})
	return excused
}

// readsLikeProse says why a literal looks like something drawn for a person, or
// returns empty when it does not.
func readsLikeProse(text string) string {
	if strings.Contains(text, " ") && len(words(text)) >= 2 {
		return "reads like a sentence"
	}
	for _, word := range words(text) {
		if len(word) >= 4 && word == strings.ToUpper(word) {
			return "reads like a state shouted at the reader"
		}
	}
	return ""
}

// words is the runs of three or more letters in a string, which is what tells
// prose apart from a format skeleton like "%-24s %-8s %s".
func words(text string) []string {
	var found []string
	current := strings.Builder{}
	flush := func() {
		if current.Len() >= 3 {
			found = append(found, current.String())
		}
		current.Reset()
	}
	for _, letter := range text {
		if unicode.IsLetter(letter) {
			current.WriteRune(letter)
			continue
		}
		flush()
	}
	flush()
	return found
}
