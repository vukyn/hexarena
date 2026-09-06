package skill_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// TestAMisspeltFieldOnASkillIsRefused closes the asymmetry the status book's own
// strict read opened.
//
// Refusing an unknown field on a status while accepting one on a skill is worse
// than accepting both: an author who has learned that a typo is caught carries
// that expectation to the next file, and the next file is the larger one. The
// two typos below are the two shapes it takes — a field nobody declared, and a
// near miss on one that exists.
func TestAMisspeltFieldOnASkillIsRefused(t *testing.T) {
	for _, testCase := range []struct {
		name, extra, wants string
	}{
		{"a field nobody declared", `,"splash":300`, "splash"},
		{"a near miss on a real one", `,"accuarcy":900`, "accuarcy"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := skill.ParseBook([]byte(critSkill(testCase.extra)), deps(t))
			if err == nil {
				t.Fatalf("%s was accepted", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("the refusal does not name the field that was wrong: %v", err)
			}
		})
	}
}

// TestASkillThatDeclaresOnlyWhatItShouldStillReads is the other half: the
// stricter decoder refuses a typo rather than the shape the book is written in.
func TestASkillThatDeclaresOnlyWhatItShouldStillReads(t *testing.T) {
	book, err := skill.ParseBook([]byte(critSkill(`,"crit":200`)), deps(t))
	if err != nil {
		t.Fatalf("a well-formed skill was refused: %v", err)
	}
	if len(book.Skills()) != 1 {
		t.Errorf("the book holds %d skills", len(book.Skills()))
	}
}
