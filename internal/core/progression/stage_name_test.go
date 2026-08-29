package progression_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// TestAStageNameIsAnIdentifier is the whole of the rule, from both sides.
//
// A stage name is a **key** rather than display text — Line.Resolve compares it
// with ==, Stage.After names a predecessor by it, a placement and a learnset gate
// spell it by hand, and Line.Validate refuses two stages sharing one — so it has
// to be typeable and to have exactly one spelling. ValidateStageName's doc
// comment carries that argument in full; this holds it in both directions, since
// a rule that only refuses is as wrong as a rule that only accepts.
//
// The accepted half deliberately includes names nobody has authored yet. The
// rule has to survive a plausible future form, or the next author works round it
// rather than with it.
func TestAStageNameIsAnIdentifier(t *testing.T) {
	for _, name := range []string{
		// Every stage the cast ships today.
		"Bulbasaur", "Ivysaur", "Venusaur",
		"Charmander", "Charmeleon", "Charizard",
		"Squirtle", "Wartortle", "Blastoise",
		"Naruto", "Shippuden", "Sennin",
		// The fork the mechanism was built for, which ships nothing yet.
		"Eevee", "Vaporeon", "Jolteon",
		// And forms that are plausible rather than authored: two words, a
		// hyphen, an apostrophe, a period, a trailing digit, a trailing letter.
		"Sage Mode", "Mega Charizard X", "Ho-Oh", "Farfetch'd", "Mr. Mime",
		"Porygon2", "Nidoran-M",
	} {
		if err := progression.ValidateStageName(name); err != nil {
			t.Errorf("%q is refused as a stage name: %v", name, err)
		}
	}

	for _, testCase := range []struct {
		label string
		name  string
		want  string
	}{
		{"nothing at all", "", "has no name"},
		// The defect this rule was written for: a Vietnamese phrase where every
		// other identifier in the repository is a language-neutral proper noun.
		{"the phrase this rule was written for", "Tiên nhân", "not written in ASCII"},
		// The same phrase decomposed — a bare e and a followed by a combining
		// circumflex. It draws identically to the line above and == calls the two
		// different names, which is the concrete reason the rule is not merely
		// "no words out of a language". Written as escapes rather than as the
		// characters, because the difference is invisible on screen and an editor
		// or a paste would silently normalise it back into the line above.
		{"that phrase spelled the other way", "Tie\u0302n nha\u0302n", "not written in ASCII"},
		{"the form this line used to end with", "Vĩ thú hoá", "not written in ASCII"},
		// Not a Latin script at all, and not a special case either.
		{"not a Latin script", "仙人", "not written in ASCII"},
		// A name with nothing in it to read as a name.
		{"digits alone", "42", "no letter in it"},
		{"punctuation alone", "---", "no letter in it"},
		// Invisible differences between two spellings of one key.
		{"a leading space", " Ivysaur", "space at one end"},
		{"a trailing space", "Ivysaur ", "space at one end"},
		{"a doubled space", "Mega  Charizard", "two spaces in a row"},
		// Unprintable bytes are outside printable ASCII and refused with it.
		{"a tab", "Ivy\tsaur", "not written in ASCII"},
		{"a newline", "Ivysaur\n", "not written in ASCII"},
	} {
		t.Run(testCase.label, func(t *testing.T) {
			err := progression.ValidateStageName(testCase.name)
			if err == nil {
				t.Fatalf("%q was accepted, want a refusal mentioning %q", testCase.name, testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("%q refused with %q, want it to mention %q", testCase.name, err, testCase.want)
			}
		})
	}
}

// TestTheStageNameRuleIsTheParsersAndNotAScreens is why this is a refusal rather
// than a test over the shipped data.
//
// A test over internal/seed would bind what ships. The refusal binds every line
// anybody ever writes — a character authored through hexforge into somebody's own
// data directory, not only the four in cast.json — and it is the same call an
// authoring form can make as the name is typed, which is exactly why
// cast.ValidateID and cast.ValidateImagePath are exported too. The cost is that
// it is a new refusal on existing data, so every shipped line still has to load;
// TestAStageNameIsAnIdentifier lists all twelve shipped names among the accepted
// ones, and internal/seed's own parse is what proves it end to end.
func TestTheStageNameRuleIsTheParsersAndNotAScreens(t *testing.T) {
	line := progression.Line{
		{Name: "Seed", MinLevel: 1, Stats: growing(100)},
		{Name: "Tiên nhân", MinLevel: 16, Stats: growing(200)},
	}
	err := line.Validate(limits(), rules())
	if err == nil {
		t.Fatalf("a line naming a stage in Vietnamese was accepted")
	}
	// The index is the parser's, so an author is told which stage rather than
	// being left to find the one name the file will not answer to.
	if got := err.Error(); !strings.Contains(got, "stage 1") || !strings.Contains(got, "not written in ASCII") {
		t.Errorf("refused with %q, want it to name stage 1 and say why", got)
	}
}
