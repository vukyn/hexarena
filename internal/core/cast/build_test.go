package cast_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
)

// buildOwner is a character with two forms and a learnset the cap gates, which
// is what a build has to be checked against: the entry a build may name is not
// the entry the file lists but the one the final form has learned.
func buildOwner() map[string]any {
	character := baseCharacter()
	character["stages"] = []map[string]any{
		{"name": "Warden", "min_level": 1, "stats": table()},
		{"name": "Bastion", "min_level": 32, "stats": table()},
	}
	character["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide"},
		// Known to the first form and never to the second: the case a build
		// written from the file rather than from the form would get wrong.
		{"id": "signature_throw", "stages": []string{"Warden"}},
	}
	character["passives"] = []map[string]any{{"id": "endurance"}, {"id": "resolve"}}
	return character
}

// buildBook parses a catalogue against a cast that holds buildOwner.
func buildBook(t *testing.T, declarations ...map[string]any) (*cast.BuildBook, error) {
	t.Helper()
	characters, err := parse(t, buildOwner())
	if err != nil {
		t.Fatalf("the owning cast should parse: %v", err)
	}
	raw, err := json.Marshal(map[string]any{"builds": declarations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return cast.ParseBuilds(raw, characters)
}

// baseBuild is a build that parses, for the reason baseCharacter exists.
func baseBuild() map[string]any {
	return map[string]any{
		"id": "warden.wall", "character": "a-series.warden",
		"name": "bức tường", "intent": "Đứng chắn và không đi đâu cả",
		"skills": []string{"strike", "riptide"}, "passives": []string{"endurance"},
	}
}

func TestParseBuildsAcceptsTheBaseDeclaration(t *testing.T) {
	book, err := buildBook(t, baseBuild())
	if err != nil {
		t.Fatalf("the base build should parse: %v", err)
	}
	build, known := book.Get("warden.wall")
	if !known {
		t.Fatal("the parsed build is not in the book")
	}
	if build.Name != "bức tường" || build.Character != "a-series.warden" {
		t.Errorf("resolved %+v", build)
	}
	if !slices.Equal(build.Skills, []string{"strike", "riptide"}) {
		t.Errorf("the kit resolved to %v", build.Skills)
	}
	if got := book.Of("a-series.warden"); len(got) != 1 {
		t.Errorf("Of returned %d builds for the character that has one", len(got))
	}
	// A character nobody has written a direction for is the honest empty case,
	// not a missing entry: the catalogue answers rather than failing.
	if got := book.Of("a-series.nobody"); len(got) != 0 {
		t.Errorf("Of returned %d builds for a character with none", len(got))
	}
	if got := book.Count(); got != 1 {
		t.Errorf("the book counts %d builds", got)
	}
}

// TestParseBuildsAcceptsAnEmptyCatalogue is the same deliberate leniency
// cast.ParseBook has: a game whose characters have no authored directions yet is
// a starting point, and refusing it would mean the file could not exist before
// the first build did.
func TestParseBuildsAcceptsAnEmptyCatalogue(t *testing.T) {
	book, err := buildBook(t)
	if err != nil {
		t.Fatalf("an empty catalogue should parse: %v", err)
	}
	if got := book.Count(); got != 0 {
		t.Errorf("the empty book holds %d builds", got)
	}
}

// TestParseBuildsHandsOutCopies is the same claim cast.Book makes about
// characters: a caller editing what it was given must not reach the book.
func TestParseBuildsHandsOutCopies(t *testing.T) {
	book, err := buildBook(t, baseBuild())
	if err != nil {
		t.Fatalf("the base build should parse: %v", err)
	}
	held, _ := book.Get("warden.wall")
	held.Skills[0] = "signature_throw"
	again, _ := book.Get("warden.wall")
	if again.Skills[0] != "strike" {
		t.Errorf("editing a handed-out build changed the book's: %v", again.Skills)
	}
}

func TestParseBuildsRejections(t *testing.T) {
	// Each case changes one field of a declaration that otherwise parses, so a
	// failure names the rule rather than the fixture.
	cases := []struct {
		name    string
		change  func(map[string]any)
		wanting string
	}{
		{"no id", func(b map[string]any) { delete(b, "id") }, "needs an id"},
		{"no character", func(b map[string]any) { delete(b, "character") }, "names no character"},
		{"unknown character", func(b map[string]any) { b["character"] = "a-series.ghost" },
			"is in no cast book"},
		{"no name", func(b map[string]any) { delete(b, "name") }, "has no name"},
		// The fiction rule, and the reason it is in the parser rather than in a
		// test over the shipped file: a build's numbers are its skills' numbers,
		// so a figure written here is a second place for them to drift from.
		{"a number in the name", func(b map[string]any) { b["name"] = "tường 2" }, "spells a number"},
		{"a number in the intent", func(b map[string]any) { b["intent"] = "Chắn 3 đòn" },
			"spells a number"},
		{"no skills", func(b map[string]any) { delete(b, "skills") }, "chooses no skills"},
		{"more skills than slots", func(b map[string]any) {
			b["skills"] = []string{"strike", "riptide", "strike", "riptide", "strike"}
		}, "and there are 4 slot(s)"},
		{"a skill twice", func(b map[string]any) { b["skills"] = []string{"strike", "strike"} },
			"brings the skill \"strike\" twice"},
		{"a skill it never learned", func(b map[string]any) {
			b["skills"] = []string{"strike", "ember_lance"}
		}, "has not learned"},
		// The case the stage gate exists for: the first form knows signature_throw and
		// the form a build is checked at does not.
		{"a skill only an earlier form knew", func(b map[string]any) {
			b["skills"] = []string{"strike", "signature_throw"}
		}, "has not learned"},
		{"more traits than slots", func(b map[string]any) {
			b["passives"] = []string{"endurance", "resolve"}
		}, "and there are 1 slot(s)"},
		{"a trait it never had", func(b map[string]any) { b["passives"] = []string{"blaze"} },
			"has not learned"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			declaration := baseBuild()
			test.change(declaration)
			if _, err := buildBook(t, declaration); err == nil ||
				!strings.Contains(err.Error(), test.wanting) {
				t.Errorf("error is %v, want one mentioning %q", err, test.wanting)
			}
		})
	}
}

// TestParseBuildsRefusesTwoBuildsAnAuthorCannotTellApart covers the two
// collisions a catalogue can hold, and they are two rules rather than one: an id
// is what a screen or a placement refers to, and a name is what a person reads.
// A duplicate of either leaves somebody unable to say which build they meant.
func TestParseBuildsRefusesTwoBuildsAnAuthorCannotTellApart(t *testing.T) {
	sameID := baseBuild()
	sameID["skills"] = []string{"riptide"}
	if _, err := buildBook(t, baseBuild(), sameID); err == nil ||
		!strings.Contains(err.Error(), "two builds are called") {
		t.Errorf("error is %v, want one about a repeated id", err)
	}

	sameName := baseBuild()
	sameName["id"] = "warden.wall-again"
	if _, err := buildBook(t, baseBuild(), sameName); err == nil ||
		!strings.Contains(err.Error(), "two builds named") {
		t.Errorf("error is %v, want one about a repeated name", err)
	}
}

// TestParseBuildsNeedsTheCast is the dependency written as a refusal: nothing
// but the cast book can say whether a kit is one its character could field, so a
// catalogue parsed without one would be a list of strings nobody checked.
func TestParseBuildsNeedsTheCast(t *testing.T) {
	if _, err := cast.ParseBuilds([]byte(`{"builds": []}`), nil); err == nil ||
		!strings.Contains(err.Error(), "without the cast book") {
		t.Errorf("error is %v, want one about the missing cast book", err)
	}
}

// TestParseBuildsRefusesAKitWithoutTheTrait is the one leniency worth stating on
// purpose: a build may leave its trait slot empty, because "the plain version" is
// a decision an author is allowed to record — while an empty kit is not, since a
// unit that brings no skills cannot act.
func TestParseBuildsAllowsAnEmptyTraitSlot(t *testing.T) {
	declaration := baseBuild()
	delete(declaration, "passives")
	book, err := buildBook(t, declaration)
	if err != nil {
		t.Fatalf("a build declining its trait should parse: %v", err)
	}
	build, _ := book.Get("warden.wall")
	if len(build.Passives) != 0 {
		t.Errorf("the trait slot resolved to %v", build.Passives)
	}
}
