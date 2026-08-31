package screen

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
// ⚠️ **It is the client's test of the same name, and it had to be built again
// rather than reused.** That one scans `os.ReadDir(".")` — its own package
// directory and nothing else — so the ban did not follow a screen out of
// cmd/hexforge-tui, and #199 recorded the gap and left it because only
// punctuation and format skeletons had moved. Six real screens live here now, so
// the walker had to arrive with them or the two-language rule would silently stop
// applying to exactly the code that draws the wording.
//
// ⚠️ **The golden cannot stand in for this and is not a weaker version of it.**
// A literal moved out of the client renders identically, so `screens.golden` is
// byte for byte what it was — the two are complements: the golden holds what is
// drawn, this holds where the words that get drawn are allowed to live.
//
// A duplicate walker rather than an exported one, deliberately: this is a
// property of a *package's own source*, so the thing that must be per-package is
// which directory is read, and the three helpers under it are small enough that
// exporting them from a test file somewhere would be a dependency between two
// test suites for no saving.
//
// The scan treats a string as something a person would read when it holds two
// words of three letters or more with a space between them, or when it is a
// shouted word — those are the two shapes every line a screen draws used to have.
// Format skeletons, key names and import paths have neither, which is why they
// can stay.
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
		ast.Inspect(file, func(node ast.Node) bool {
			literal, isLiteral := node.(*ast.BasicLit)
			if !isLiteral || literal.Kind != token.STRING {
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
	// sentence, which is the failure mode #199 recorded for the client's copy.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files for wording", scanned)
}

// readsLikeProse says why a literal looks like something drawn for a person,
// or returns empty when it does not.
func readsLikeProse(text string) string {
	if strings.Contains(text, " ") && len(words(text)) >= 2 {
		return "reads like a sentence"
	}
	for _, word := range words(text) {
		if len(word) >= 4 && word == strings.ToUpper(word) {
			return "reads like a state shouted at the author"
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
