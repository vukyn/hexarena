package passive_test

import (
	"strings"
	"testing"
)

// TestAMisspeltFieldOnATraitIsRefused is the skill book's test one book over,
// and both exist because the rule is now the same in all three: a field the book
// does not know is a sentence rather than a silence.
//
// ⚠️ The near miss is deliberately a field that **does** exist elsewhere in the
// same file — `grant` for `grants` — because that is the typo a strict decoder
// is actually for. A field nobody has ever declared is usually noticed by the
// author writing it; a singular where the book wants a plural reads correctly
// and does nothing.
func TestAMisspeltFieldOnATraitIsRefused(t *testing.T) {
	for _, testCase := range []struct {
		name, body, wants string
	}{
		{
			"a field nobody declared",
			`[{"id":"steady","grants":[{"status":"toughened"}],"reach":2}]`,
			"reach",
		},
		{
			"a singular where the book wants a plural",
			`[{"id":"steady","grant":[{"status":"toughened"}]}]`,
			"grant",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parse(t, testCase.body)
			if err == nil {
				t.Fatalf("%s was accepted", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("the refusal does not name the field that was wrong: %v", err)
			}
		})
	}
}

// TestATraitThatDeclaresOnlyWhatItShouldStillReads is what says the stricter
// decoder refuses a typo rather than the shape the book is written in.
func TestATraitThatDeclaresOnlyWhatItShouldStillReads(t *testing.T) {
	book, err := parse(t, `[{"id":"steady","grants":[{"status":"toughened"}]}]`)
	if err != nil {
		t.Fatalf("a well-formed trait was refused: %v", err)
	}
	if len(book.All()) != 1 {
		t.Errorf("the book holds %d traits", len(book.All()))
	}
}
