package cast_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

func TestParseSpeciesRejections(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  string
		want string
	}{
		{"not json", `{`, "decode species book"},
		{"no id", `{"species":[{"name":"rồng"}]}`, "species id is empty"},
		{"id is not a slug", `{"species":[{"id":"Dragon","name":"rồng"}]}`, "lowercase letters"},
		{"no name", `{"species":[{"id":"dragon"}]}`, "has no name"},
		{"blank name", `{"species":[{"id":"dragon","name":"   "}]}`, "has no name"},
		{"declared twice",
			`{"species":[{"id":"dragon","name":"rồng"},{"id":"dragon","name":"khác"}]}`,
			`"dragon" is declared twice`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := cast.ParseSpecies([]byte(test.raw))
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("the refusal is %q, want it to mention %q", err, test.want)
			}
		})
	}
}

// TestAnEmptySpeciesCatalogIsAllowed is the same judgement the origin catalog
// gets: a project where nothing has needed the axis is a starting point, and
// every character's list of what it is may be empty.
func TestAnEmptySpeciesCatalogIsAllowed(t *testing.T) {
	book, err := cast.ParseSpecies([]byte(`{"species":[]}`))
	if err != nil {
		t.Fatalf("an empty catalog was refused: %v", err)
	}
	if len(book.All()) != 0 || len(book.IDs()) != 0 {
		t.Errorf("an empty catalog holds %v", book.All())
	}
	if _, known := book.Get("dragon"); known {
		t.Error("an empty catalog answered a lookup")
	}
}

func TestSpeciesBookMarshalIsSortedAndReParses(t *testing.T) {
	book, err := cast.ParseSpecies([]byte(`{
	  "species": [
	    {"id": "turtle", "name": "rùa"},
	    {"id": "dragon", "name": "rồng", "note": "a lineage"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Sorted by id, which is what makes a tool's rewrite a one-line diff rather
	// than a reshuffle nobody can review.
	if strings.Index(string(raw), "dragon") > strings.Index(string(raw), "turtle") {
		t.Errorf("the file is not sorted by id:\n%s", raw)
	}
	again, err := cast.ParseSpecies(raw)
	if err != nil {
		t.Fatalf("what Marshal wrote does not parse: %v", err)
	}
	if len(again.All()) != 2 {
		t.Fatalf("the round trip kept %d of 2", len(again.All()))
	}
	// A note survives and an absent one stays absent, which omitempty is what
	// makes true.
	if got, _ := again.Get("dragon"); got.Note != "a lineage" {
		t.Errorf("the note came back as %q", got.Note)
	}
	if strings.Contains(string(raw), `"note": ""`) {
		t.Errorf("an absent note was written out:\n%s", raw)
	}
}

func TestSpeciesBookDoesNotHandOutItsInternals(t *testing.T) {
	book := speciesBook(t)
	all := book.All()
	all[0].Name = "rewritten"
	if again, _ := book.Get(all[0].ID); again.Name == "rewritten" {
		t.Error("writing into what All returned changed the book")
	}
	ids := book.IDs()
	ids[0] = "rewritten"
	if book.IDs()[0] == "rewritten" {
		t.Error("writing into what IDs returned changed the book")
	}
}

// TestACharacterIsCheckedAgainstTheSpeciesCatalog is the parse-time half: what a
// character claims to be has to exist, and it has to be said once.
func TestACharacterIsCheckedAgainstTheSpeciesCatalog(t *testing.T) {
	for _, test := range []struct {
		name    string
		species []string
		want    string
	}{
		{"unknown", []string{"dragoon"}, `unknown species "dragoon"`},
		{"named twice", []string{"dragon", "dragon"}, `the species "dragon" twice`},
	} {
		t.Run(test.name, func(t *testing.T) {
			declared := baseCharacter()
			declared["species"] = test.species
			_, err := parse(t, declared)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("the refusal is %q, want it to mention %q", err, test.want)
			}
		})
	}

	// And the refusal lists what there is, because a name that is wrong is
	// usually a name that is nearly right.
	declared := baseCharacter()
	declared["species"] = []string{"dragoon"}
	_, err := parse(t, declared)
	if err == nil || !strings.Contains(err.Error(), "dragon lizard") {
		t.Errorf("the refusal does not list the catalog: %v", err)
	}
}

// TestNamingASpeciesWithoutTheCatalogIsRefused is the same rule the passive book
// gets: a list that names something needs the book it names it from, so a data
// file cannot be checked or unchecked depending on how the caller wired itself
// up.
func TestNamingASpeciesWithoutTheCatalogIsRefused(t *testing.T) {
	declared := baseCharacter()
	declared["species"] = []string{"dragon"}
	raw, err := json.Marshal(map[string]any{"characters": []map[string]any{declared}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	without := deps(t)
	without.Species = nil
	if _, err := cast.ParseBook(raw, without); err == nil ||
		!strings.Contains(err.Error(), "without the species book") {
		t.Errorf("a character naming a species with no catalog gave %v", err)
	}

	// A character claiming nothing does not need the catalog at all, which is
	// what keeps the axis optional rather than a new dependency for everybody.
	if _, err := cast.ParseBook([]byte(`{"characters":[]}`), without); err != nil {
		t.Errorf("an empty cast needed the species book: %v", err)
	}
	plain, err := json.Marshal(map[string]any{"characters": []map[string]any{baseCharacter()}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := cast.ParseBook(plain, without); err != nil {
		t.Errorf("a character that is nothing in particular needed the species book: %v", err)
	}
}

// TestASkillKeptForALineageAsksWhatTheCarrierIs is the carry rule, and the three
// cases are the whole of it: being one of the named kinds carries the skill,
// being some other kind does not, and being nothing in particular does not
// either.
//
// The last is the one worth stating. An empty list is a real answer here rather
// than a question nobody has reached — the form's reading, which forge.Carrier
// documents — so a lineage skill refuses a character that claims nothing.
func TestASkillKeptForALineageAsksWhatTheCarrierIs(t *testing.T) {
	for _, test := range []struct {
		name    string
		species []string
		refused bool
	}{
		{"is one", []string{"dragon"}, false},
		{"is one of several", []string{"lizard", "dragon"}, false},
		{"is something else", []string{"lizard"}, true},
		{"is nothing in particular", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			declared := baseCharacter()
			if test.species != nil {
				declared["species"] = test.species
			}
			declared["skills"] = []string{"strike", "lineage_roar"}
			book, err := parse(t, declared)
			switch {
			case test.refused && err == nil:
				t.Fatal("the skill was carried by something it is kept from")
			case !test.refused && err != nil:
				t.Fatalf("the skill was refused to something it is kept for: %v", err)
			}
			if test.refused {
				if !strings.Contains(err.Error(), `only a dragon may carry`) {
					t.Errorf("the refusal is %q, and does not say what it is kept for", err)
				}
				return
			}
			// And the accepted character keeps what it claimed, in the order it
			// claimed it: the list is authored and nothing here sorts it.
			held, known := book.Get("a-series.warden")
			if !known {
				t.Fatal("the character that parsed is not in the book")
			}
			if strings.Join(held.Species, " ") != strings.Join(test.species, " ") {
				t.Errorf("it came back as %v, want %v", held.Species, test.species)
			}
		})
	}
}

// TestASkillKeptForAnUnknownSpeciesIsRefusedByItsCarrier is the typo case, and
// the reason it is worth its own refusal: an allowlist naming a kind nobody
// declared admits nobody, which reads as "kept for a dragoon" rather than as the
// misspelling it is.
func TestASkillKeptForAnUnknownSpeciesIsRefusedByItsCarrier(t *testing.T) {
	book, err := skillsWithLineage(t, "dragoon")
	if err != nil {
		t.Fatalf("the skill book itself is legal: %v", err)
	}
	declared := baseCharacter()
	declared["species"] = []string{"dragon"}
	declared["skills"] = []string{"strike", "lineage_roar"}
	raw, err := json.Marshal(map[string]any{"characters": []map[string]any{declared}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	with := deps(t)
	with.Skills = book
	if _, err := cast.ParseBook(raw, with); err == nil ||
		!strings.Contains(err.Error(), `unknown species "dragoon"`) {
		t.Errorf("a skill kept for a species nobody declared gave %v", err)
	}
}

// skillsWithLineage is a skill book whose lineage skill is kept for a named
// species, so a test can point that allowlist at something the catalog does not
// declare without touching the shared fixture.
func skillsWithLineage(t *testing.T, kept string) (*skill.Book, error) {
	t.Helper()
	patterns, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3, "splash_power": 500,
	  "patterns": [{"name": "single", "splash": []}]
	}`))
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [{"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500}]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	raw := `{"skills":[
	  {"id": "strike", "element": "neutral", "range": 1, "pattern": "single",
	   "power": 1000, "accuracy": 950, "target": "enemy"},
	  {"id": "riptide", "element": "water", "range": 2, "pattern": "single",
	   "power": 1600, "accuracy": 900, "target": "enemy"},
	  {"id": "lineage_roar", "element": "neutral", "range": 1, "pattern": "single",
	   "power": 1200, "accuracy": 900, "target": "enemy",
	   "restrict": {"species": ["` + kept + `"]}}
	]}`
	return skill.ParseBook([]byte(raw), skill.Deps{Patterns: patterns, Statuses: statuses})
}

// TestAPresetCannotHoldASkillKeptForALineage is the archetype half, and it is the
// same refusal a character-restricted skill gets: a preset says how a character
// fights and nothing about what it is, so every character built from one that is
// not of those kinds would be refused — and the refusal would land on whoever
// wrote the character.
//
// This is also why the two shipped skills that read as a *body* — roots,
// photosynthesis — stay restricted by element: they are in a preset's kit, and
// moving them onto a species would make the preset itself illegal.
func TestAPresetCannotHoldASkillKeptForALineage(t *testing.T) {
	declared := baseArchetype()
	declared["skills"] = []string{"strike", "lineage_roar"}
	_, err := archetypes(t, declared)
	if err == nil {
		t.Fatal("a preset held a skill kept for a lineage")
	}
	for _, want := range []string{"lineage_roar", "only a dragon", "rather than what it is"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, want it to mention %q", err, want)
		}
	}
}

// TestOfSpeciesReturnsEveryoneThatIsOne is the lookup a listing and the golden
// report use, and the property that separates it from OfOrigin: a work is a
// partition of the cast and a species is not, so one character is in two answers.
func TestOfSpeciesReturnsEveryoneThatIsOne(t *testing.T) {
	first := baseCharacter()
	first["species"] = []string{"lizard", "dragon"}
	second := baseCharacter()
	second["id"] = "a-series.other"
	second["name"] = "Other"
	second["species"] = []string{"lizard"}
	third := baseCharacter()
	third["id"] = "a-series.plain"
	third["name"] = "Plain"
	book, err := parse(t, first, second, third)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, test := range []struct {
		id   string
		want []string
	}{
		{"lizard", []string{"a-series.warden", "a-series.other"}},
		{"dragon", []string{"a-series.warden"}},
		{"turtle", nil},
	} {
		got := make([]string, 0, 3)
		for _, character := range book.OfSpecies(test.id) {
			got = append(got, character.ID)
		}
		if strings.Join(got, " ") != strings.Join(test.want, " ") {
			t.Errorf("OfSpecies(%q) is %v, want %v", test.id, got, test.want)
		}
	}
}

// TestSpeciesAppendValidatesBeforeItAccepts keeps the authoring path honest: a
// tool grows the catalog through Append, so an entry it would refuse on a load
// must be refused there too.
func TestSpeciesAppendValidatesBeforeItAccepts(t *testing.T) {
	book := speciesBook(t)
	if _, err := book.Append(cast.Species{ID: "turtle"}); err == nil {
		t.Error("a species with no name was appended")
	}
	if _, err := book.Append(cast.Species{ID: "dragon", Name: "rồng"}); err == nil {
		t.Error("a species already in the catalog was appended")
	}
	grown, err := book.Append(cast.Species{ID: "turtle", Name: "rùa"})
	if err != nil {
		t.Fatalf("a legal species was refused: %v", err)
	}
	if _, known := grown.Get("turtle"); !known {
		t.Error("the appended species is not in the new catalog")
	}
	if _, known := book.Get("turtle"); known {
		t.Error("Append changed the catalog it was called on")
	}
}
