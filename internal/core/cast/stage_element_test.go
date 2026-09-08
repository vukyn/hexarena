package cast_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// aGrowingCharacter is the base declaration with a second form, as a single
// element so that a stage naming two of them is a change with something to
// measure.
//
// The grown form's element is passed in as the JSON writes it — a name, a list,
// or nil for a form that declares none — because every case below is about what
// that one field is allowed to say.
func aGrowingCharacter(grown any) map[string]any {
	entry := baseCharacter()
	entry["element"] = "water"
	forms := []map[string]any{
		{"name": "Warden", "min_level": 1, "stats": table()},
		{"name": "Prime", "min_level": 30, "stats": table()},
	}
	if grown != nil {
		forms[1]["element"] = grown
	}
	entry["stages"] = forms
	return entry
}

// gatedFor is a learnset entry kept for named forms.
func gatedFor(id string, level int, forms ...string) map[string]any {
	entry := map[string]any{"id": id, "at_level": level}
	if len(forms) > 0 {
		entry["stages"] = forms
	}
	return entry
}

// TestAStageCarriesItsOwnElementAndFallsBackToTheCharacters is the fallback
// itself, which has exactly one home.
//
// A form that declares nothing is the character's — that is what every line in
// the repository is today and what makes the field optional — and a form that
// declares one is that one. A caller reading progression.Stage.Element directly
// would get a nil pointer for the first case, which is why Character.ElementAt
// is the only door and why no second caller may invent this.
func TestAStageCarriesItsOwnElementAndFallsBackToTheCharacters(t *testing.T) {
	book, err := parse(t, aGrowingCharacter([]string{"water", "wind"}))
	if err != nil {
		t.Fatalf("a grown form with an element of its own should parse: %v", err)
	}
	character, known := book.Get("a-series.warden")
	if !known {
		t.Fatal("the character did not come back out of the book")
	}
	water, err := element.Single(element.Water)
	if err != nil {
		t.Fatalf("water: %v", err)
	}
	waterWind, err := element.Dual(element.Water, element.Wind)
	if err != nil {
		t.Fatalf("water/wind: %v", err)
	}
	if got := character.ElementAt(character.Stages[0]); got != water {
		t.Errorf("the root form is %s, and it declares none so it should be the character's %s",
			got, water)
	}
	if got := character.ElementAt(character.Stages[1]); got != waterWind {
		t.Errorf("the grown form is %s, and it declares %s", got, waterWind)
	}
	// And the declaration is still readable as an absence on the form that made
	// none, which is what makes the pointer worth its awkwardness: the zero
	// affinity is a legal single neutral, so a value field would have made this
	// form neutral rather than unanswered.
	if character.Stages[0].Element != nil {
		t.Errorf("the root form declares %v, and the fixture gives it none",
			*character.Stages[0].Element)
	}
}

// TestASkillOnlyOneFormsElementCarriesNeedsThatFormsGate is the parser rule the
// mechanism costs, and the quantifier is the whole of it.
//
// A line that is water as a root and water/wind as a grown form may hold a wind
// skill — but only the grown form may, so the entry has to say so. Ungated, the
// root could be fielded holding it at any level from its own upwards, and
// battle.enlist would refuse that unit at the moment somebody played it. "Some
// form can carry it" is therefore not a rule the authoring layer may apply.
func TestASkillOnlyOneFormsElementCarriesNeedsThatFormsGate(t *testing.T) {
	t.Run("kept for the form whose element carries it", func(t *testing.T) {
		entry := aGrowingCharacter([]string{"water", "wind"})
		entry["skills"] = []map[string]any{
			{"id": "strike"}, {"id": "riptide"},
			gatedFor("gale_slash", 30, "Prime"),
		}
		if _, err := parse(t, entry); err != nil {
			t.Fatalf("a wind skill kept for the form that is wind should parse: %v", err)
		}
	})

	t.Run("ungated, so the root form could be fielded holding it", func(t *testing.T) {
		entry := aGrowingCharacter([]string{"water", "wind"})
		entry["skills"] = []map[string]any{
			{"id": "strike"}, {"id": "riptide"},
			gatedFor("gale_slash", 30),
		}
		_, err := parse(t, entry)
		if err == nil {
			t.Fatal("an ungated wind skill was accepted on a line whose root is water, and battle.New will refuse the root form holding it")
		}
		if !strings.Contains(err.Error(), "cannot carry") {
			t.Errorf("the refusal is %q, want it to say what cannot be carried", err)
		}
		// The refusal has to name the FORM, because "this character is water"
		// is not true of every form of it any more and a reader told only that
		// has nowhere to look.
		if !strings.Contains(err.Error(), "Warden") {
			t.Errorf("the refusal is %q, want it to name the form that cannot carry it", err)
		}
	})
}

// TestAStageElementIsRefusedByTheChartAndByItsOwnDecoder covers both halves of
// what may be written in that field, because they are refused in two different
// places and a test that only reached one would leave the other unmeasured.
//
// The chart's half is the pair that already counters itself: a unit that is both
// the answer to and the victim of its own second element is incoherent, and
// progression cannot say so because it has no chart. The decoder's half is
// everything an affinity is refused for anywhere — a name that is no element, a
// list that is not one or two names, a repeat, and a pairing with the inert
// element, which adds nothing.
func TestAStageElementIsRefusedByTheChartAndByItsOwnDecoder(t *testing.T) {
	for _, test := range []struct {
		name    string
		declare any
		wantIn  string
	}{
		{"a pair that already counters itself", []string{"water", "fire"}, "counter each other"},
		{"a pairing with the inert element", []string{"water", "neutral"}, "pairs with neutral"},
		{"the same element twice", []string{"wind", "wind"}, "listed twice"},
		{"a name that is no element", "grit", "grit"},
		{"three elements", []string{"water", "wind", "ice"}, "want 1 or 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, aGrowingCharacter(test.declare))
			if err == nil {
				t.Fatalf("a grown form declaring %v was accepted", test.declare)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the refusal is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

// TestAStageElementSurvivesTheRoundTrip is what keeps the authoring tool from
// deleting the field on the next write.
//
// `hexforge new` reads the whole book and writes it back, so a field the writer
// does not know about is a field the next append silently drops — the reason
// Hidden is read as well as written. Marshal goes through Character, whose
// Stages are the progression.Line verbatim, so this holds for free; it is
// asserted because "for free" is a property of the current shape rather than a
// rule anybody wrote down.
//
// The second half is that a form declaring nothing writes nothing: omitempty on
// a nil pointer is what makes every character already in the repository come
// back byte for byte, so this whole mechanism ships without touching any
// existing data.
func TestAStageElementSurvivesTheRoundTrip(t *testing.T) {
	book, err := parse(t, aGrowingCharacter([]string{"water", "wind"}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if counted := strings.Count(string(raw), `"element"`); counted != 2 {
		t.Errorf("the written book names an element %d times, want 2 — the character's and the one form that declares one:\n%s",
			counted, raw)
	}
	reparsed, err := cast.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("the written book should parse back: %v", err)
	}
	character, known := reparsed.Get("a-series.warden")
	if !known {
		t.Fatal("the character did not survive the round trip")
	}
	waterWind, err := element.Dual(element.Water, element.Wind)
	if err != nil {
		t.Fatalf("water/wind: %v", err)
	}
	if got := character.ElementAt(character.Stages[1]); got != waterWind {
		t.Errorf("after the round trip the grown form is %s, want %s", got, waterWind)
	}
}

// TestProgressionDoesNotJudgeAStageElement records the division of labour, so
// that a later reader does not "complete" the validation one layer down.
//
// Line.Validate is pure stat arithmetic and a stage name check; it has no
// element chart and cannot get one without progression importing a book. So a
// pair the chart refuses passes here and is refused by cast.resolveCharacter —
// exactly as an image path is shaped by cast.ValidateImagePath rather than by
// the package that holds the field.
func TestProgressionDoesNotJudgeAStageElement(t *testing.T) {
	refused, err := element.Dual(element.Water, element.Fire)
	if err != nil {
		t.Fatalf("water/fire is decodable and only the chart refuses it: %v", err)
	}
	if chart(t).ValidateAffinity(refused) == nil {
		t.Fatal("the chart accepts water/fire, so this test measures nothing")
	}
	// The same curves the JSON fixture writes, as the resolved type.
	stats := progression.Table{
		progression.HP:       {Base: 930, Max: 3100},
		progression.Attack:   {Base: 150, Max: 500},
		progression.Defense:  {Base: 240, Max: 800},
		progression.Speed:    {Base: 27, Max: 90},
		progression.Accuracy: {Base: 24, Max: 80},
		progression.Dodge:    {Base: 9, Max: 30},
	}
	line := progression.Line{
		{Name: "Warden", MinLevel: 1, Stats: stats},
		{Name: "Prime", MinLevel: 30, Element: &refused, Stats: stats},
	}
	if err := line.Validate(limits(t), rules(t)); err != nil {
		t.Errorf("progression refused a stage element, and it has no chart to refuse one with: %v", err)
	}
}
