package cast_test

import (
	"encoding/json"
	"github.com/vukyn/hexarena/internal/core/passive"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// The fixtures are inline rather than loaded, because this package parses bytes
// and a test that read a file would be testing internal/seed instead.

func chart(t *testing.T) *element.Chart {
	t.Helper()
	// Every element has to be classified exactly once, so an inline chart has
	// to cover all of them. This is the shipped layout.
	built, err := element.ParseChart([]byte(`{
	  "multipliers": {"advantage": 1500, "neutral": 1000, "disadvantage": 667},
	  "cycles": [
	    {"name": "organic", "chain": ["water", "fire", "grass", "ground"]},
	    {"name": "industrial", "chain": ["ice", "metal", "wind", "electric"]},
	    {"name": "cross", "chain": ["water", "metal", "grass", "wind", "fire", "ice", "ground", "electric"]}
	  ],
	  "mutual": [["light", "dark"]],
	  "inert": ["neutral"]
	}`))
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	return built
}

func skills(t *testing.T) *skill.Book {
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
	book, err := skill.ParseBook([]byte(`{
	  "skills": [
	    {"id": "strike", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy"},
	    {"id": "riptide", "element": "water", "range": 2, "pattern": "single",
	     "power": 1600, "accuracy": 900, "target": "enemy"},
	    {"id": "ember_lance", "element": "fire", "range": 2, "pattern": "single",
	     "power": 1800, "accuracy": 900, "target": "enemy"},
	    {"id": "gale_slash", "element": "wind", "range": 2, "pattern": "single",
	     "power": 1500, "accuracy": 900, "target": "enemy"},
	    {"id": "lineage_roar", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1200, "accuracy": 900, "target": "enemy",
	     "restrict": {"species": ["dragon"]}}
	  ]
	}`), skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	return book
}

func limits(t *testing.T) progression.Limits {
	t.Helper()
	parsed, err := progression.ParseLimits([]byte(`{
	  "level_cap": 60,
	  "ceilings": {"hp": 4800, "attack": 800, "defense": 800, "speed": 200, "accuracy": 300, "dodge": 150},
	  "max_effective_hp": 11500
	}`))
	if err != nil {
		t.Fatalf("limits: %v", err)
	}
	return parsed
}

func rules(t *testing.T) combat.Rules {
	t.Helper()
	parsed, err := combat.ParseRules([]byte(`{
	  "defense_constant": 300, "minimum_damage": 1, "min_hit_chance": 150, "max_block_charges": 3
	}`))
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	return parsed
}

func origins(t *testing.T) *cast.OriginBook {
	t.Helper()
	book, err := cast.ParseOrigins([]byte(`{
	  "origins": [
	    {"id": "a-series", "title": "A Series", "medium": "series", "year": 2004},
	    {"id": "a-game", "title": "A Game", "medium": "game"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("origins: %v", err)
	}
	return book
}

// curve is a "base:max" pair as the data files write it.
func curve(base, max int64) map[string]any {
	return map[string]any{"base": base, "max": max}
}

// table is a full stat table that fits the budget at every level.
func table() map[string]any {
	return map[string]any{
		"hp": curve(930, 3100), "attack": curve(150, 500), "defense": curve(240, 800),
		"speed": curve(27, 90), "accuracy": curve(24, 80), "dodge": curve(9, 30),
	}
}

func archetypes(t *testing.T, declarations ...map[string]any) (*cast.ArchetypeBook, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"archetypes": declarations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: skills(t), Passives: passives(t), Limits: limits(t), Rules: rules(t),
	})
}

// baseArchetype is a preset that parses, so a negative case can change exactly
// one thing and prove that change is what breaks it.
func baseArchetype() map[string]any {
	return map[string]any{
		"id": "sentinel", "role": "armour rather than health", "column": 2,
		"stats": table(), "skills": []string{"strike", "riptide"},
	}
}

func archetypeBook(t *testing.T) *cast.ArchetypeBook {
	t.Helper()
	book, err := archetypes(t, baseArchetype())
	if err != nil {
		t.Fatalf("the base archetype should parse: %v", err)
	}
	return book
}

func deps(t *testing.T) cast.Deps {
	t.Helper()
	return cast.Deps{
		Origins: origins(t), Archetypes: archetypeBook(t), Skills: skills(t),
		Passives: passives(t), Species: speciesBook(t),
		Chart: chart(t), Limits: limits(t), Rules: rules(t),
	}
}

// speciesBook is the catalog the fixtures claim to be. Two kinds, because one
// would not show that a character may be several things at once.
func speciesBook(t *testing.T) *cast.SpeciesBook {
	t.Helper()
	book, err := cast.ParseSpecies([]byte(`{
	  "species": [
	    {"id": "dragon", "name": "rồng"},
	    {"id": "lizard", "name": "thằn lằn", "note": "a body rather than a lineage"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("species: %v", err)
	}
	return book
}

// passives is the trait book the fixtures name. One permanent status and one
// trait granting it is the whole of what this package has to check: whether a
// trait *does* anything is the passive package's own business.
func passives(t *testing.T) *passive.Book {
	t.Helper()
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "toughened", "category": "buff", "max_stacks": 2, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 200}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	book, err := passive.ParseBook(
		[]byte(`{"passives":[
		  {"id":"endurance","grants":[{"status":"toughened"}]},
		  {"id":"resolve","grants":[{"status":"toughened","stacks":2}]}
		]}`),
		passive.Deps{Statuses: statuses})
	if err != nil {
		t.Fatalf("passives: %v", err)
	}
	return book
}

// baseCharacter is a character that parses, for the same reason
// baseArchetype exists.
func baseCharacter() map[string]any {
	return map[string]any{
		"id": "a-series.warden", "name": "Warden",
		"origin": "a-series", "archetype": "sentinel",
		"image": "assets/a-series/warden.svg", "element": []string{"water", "ice"},
		"stages": []map[string]any{
			{"name": "Warden", "min_level": 1, "stats": table()},
		},
		"skills": []string{"strike", "riptide"},
	}
}

func parse(t *testing.T, declarations ...map[string]any) (*cast.Book, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"characters": declarations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return cast.ParseBook(raw, deps(t))
}

func TestParseBookAcceptsTheBaseDeclaration(t *testing.T) {
	book, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("the base declaration should parse: %v", err)
	}
	found, ok := book.Get("a-series.warden")
	if !ok {
		t.Fatal("the parsed character is not in the book")
	}
	if found.Name != "Warden" || found.Origin != "a-series" || found.Archetype != "sentinel" {
		t.Errorf("resolved %+v", found)
	}
	if got, want := found.Element.String(), "water/ice"; got != want {
		t.Errorf("element is %s, want %s", got, want)
	}
	values, stage, err := found.Resolve(progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve at the cap: %v", err)
	}
	if stage.Name != "Warden" {
		t.Errorf("the cap lands in stage %q, want %q", stage.Name, "Warden")
	}
	if got, want := values[progression.HP], int64(3100); got != want {
		t.Errorf("health at the cap is %d, want %d", got, want)
	}
	if got := book.OfOrigin("a-series"); len(got) != 1 {
		t.Errorf("OfOrigin returned %d characters, want 1", len(got))
	}
	if got := book.OfOrigin("a-game"); len(got) != 0 {
		t.Errorf("OfOrigin returned %d characters for an origin nobody uses, want 0", len(got))
	}
}

// TestParseBookAcceptsAnEmptyCast is the deliberate difference from
// skill.ParseBook: a project with no characters yet is a starting point, and a
// game with no skills is not.
func TestParseBookAcceptsAnEmptyCast(t *testing.T) {
	book, err := cast.ParseBook([]byte(`{"characters": []}`), deps(t))
	if err != nil {
		t.Fatalf("an empty cast should parse: %v", err)
	}
	if got := len(book.All()); got != 0 {
		t.Errorf("the empty book holds %d characters", got)
	}
}

func TestParseBookRejections(t *testing.T) {
	// Each case changes one field of a declaration that otherwise parses, so a
	// failure names the rule rather than an unrelated mistake.
	cases := []struct {
		name    string
		change  func(map[string]any)
		wantIn  string
		another map[string]any
	}{
		{
			name:   "unknown origin",
			change: func(entry map[string]any) { entry["origin"] = "nowhere" },
			wantIn: "unknown origin",
		},
		{
			name:   "unknown archetype",
			change: func(entry map[string]any) { entry["archetype"] = "berserker" },
			wantIn: "unknown archetype",
		},
		{
			name:   "unknown skill",
			change: func(entry map[string]any) { entry["skills"] = []string{"strike", "meteor"} },
			wantIn: "unknown skill",
		},
		{
			name:   "empty skill list",
			change: func(entry map[string]any) { entry["skills"] = []string{} },
			wantIn: "knows no skills",
		},
		{
			name:   "duplicate skill",
			change: func(entry map[string]any) { entry["skills"] = []string{"strike", "strike"} },
			wantIn: "twice",
		},
		{
			name:   "non-slug id",
			change: func(entry map[string]any) { entry["id"] = "A-Series.Warden" },
			wantIn: "lowercase letters",
		},
		{
			name:   "two dots in an id",
			change: func(entry map[string]any) { entry["id"] = "a.b.c" },
			wantIn: "at most one",
		},
		{
			name:   "no display name",
			change: func(entry map[string]any) { entry["name"] = "  " },
			wantIn: "display name",
		},
		{
			name:   "no element",
			change: func(entry map[string]any) { delete(entry, "element") },
			wantIn: "does not declare an element",
		},
		{
			// Water beats fire on the chart, so a unit of both is its own
			// counter and its own victim.
			name:   "illegal element pair",
			change: func(entry map[string]any) { entry["element"] = []string{"water", "fire"} },
			wantIn: "counter each other",
		},
		{
			name:   "bad image extension",
			change: func(entry map[string]any) { entry["image"] = "assets/a-series/warden.jpg" },
			wantIn: `want .svg or .png`,
		},
		{
			name:   "absolute image path",
			change: func(entry map[string]any) { entry["image"] = "/assets/a-series/warden.svg" },
			wantIn: "absolute path",
		},
		{
			name:   "image climbing out of the data directory",
			change: func(entry map[string]any) { entry["image"] = "assets/../../etc/warden.svg" },
			wantIn: "climbs out",
		},
		{
			name:   "no image",
			change: func(entry map[string]any) { entry["image"] = "" },
			wantIn: "declares no image",
		},
		{
			name: "a stage line that does not start at level 1",
			change: func(entry map[string]any) {
				entry["stages"] = []map[string]any{
					{"name": "Warden", "min_level": 5, "stats": table()},
				}
			},
			wantIn: "want 1",
		},
		{
			name:   "no stages at all",
			change: func(entry map[string]any) { delete(entry, "stages") },
			wantIn: "at least one stage",
		},
		{
			name: "a stage whose table breaks the joint durability budget",
			change: func(entry map[string]any) {
				breaking := table()
				breaking["hp"] = curve(1440, 4800)
				breaking["defense"] = curve(240, 800)
				entry["stages"] = []map[string]any{
					{"name": "Warden", "min_level": 1, "stats": breaking},
				}
			},
			wantIn: "over the budget",
		},
		{
			// The rule lives in skill.CanCarry and battle.enlist applies the
			// same call, so a character that writes cleanly here is a character
			// that loads.
			name:   "a kit the affinity cannot carry",
			change: func(entry map[string]any) { entry["skills"] = []string{"strike", "ember_lance"} },
			wantIn: "cannot carry",
		},
		{
			name:    "duplicate character id",
			change:  func(map[string]any) {},
			another: baseCharacter(),
			wantIn:  "declared twice",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			entry := baseCharacter()
			test.change(entry)
			declarations := []map[string]any{entry}
			if test.another != nil {
				declarations = append(declarations, test.another)
			}
			_, err := parse(t, declarations...)
			if err == nil {
				t.Fatalf("the declaration parsed, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

func TestParseBookNeedsEveryDependency(t *testing.T) {
	full := deps(t)
	cases := map[string]cast.Deps{
		"no origins":    {Archetypes: full.Archetypes, Skills: full.Skills, Chart: full.Chart, Limits: full.Limits, Rules: full.Rules},
		"no archetypes": {Origins: full.Origins, Skills: full.Skills, Chart: full.Chart, Limits: full.Limits, Rules: full.Rules},
		"no skills":     {Origins: full.Origins, Archetypes: full.Archetypes, Chart: full.Chart, Limits: full.Limits, Rules: full.Rules},
		"no chart":      {Origins: full.Origins, Archetypes: full.Archetypes, Skills: full.Skills, Limits: full.Limits, Rules: full.Rules},
	}
	// The map is only a set of independent cases; nothing here reaches an
	// output whose order matters.
	for name, incomplete := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := cast.ParseBook([]byte(`{"characters": []}`), incomplete); err == nil {
				t.Error("an incomplete set of dependencies was accepted")
			}
		})
	}
}

func TestParseArchetypesRejections(t *testing.T) {
	cases := []struct {
		name    string
		change  func(map[string]any)
		wantIn  string
		another map[string]any
	}{
		{
			name:   "no id",
			change: func(entry map[string]any) { entry["id"] = "" },
			wantIn: "archetype id is empty",
		},
		{
			name:   "non-slug id",
			change: func(entry map[string]any) { entry["id"] = "Sentinel" },
			wantIn: "lowercase letters",
		},
		{
			name:   "no role",
			change: func(entry map[string]any) { entry["role"] = "" },
			wantIn: "no role line",
		},
		{
			name:   "a role over two lines",
			change: func(entry map[string]any) { entry["role"] = "front line\nand also a healer" },
			wantIn: "more than one line",
		},
		{
			name:   "column off the formation grid",
			change: func(entry map[string]any) { entry["column"] = 3 },
			wantIn: "sits in column 3",
		},
		{
			name:   "negative column",
			change: func(entry map[string]any) { entry["column"] = -1 },
			wantIn: "sits in column -1",
		},
		{
			name:   "no stat table",
			change: func(entry map[string]any) { delete(entry, "stats") },
			wantIn: "declares no stat curve",
		},
		{
			name:   "empty kit",
			change: func(entry map[string]any) { entry["skills"] = []string{} },
			wantIn: "knows no skills",
		},
		{
			name:   "duplicated kit entry",
			change: func(entry map[string]any) { entry["skills"] = []string{"riptide", "riptide"} },
			wantIn: "twice",
		},
		{
			name:   "unknown skill",
			change: func(entry map[string]any) { entry["skills"] = []string{"meteor"} },
			wantIn: "unknown skill",
		},
		{
			// A preset that does not fit the budget would hand every author a
			// stat line that fails later, which is worse than failing here.
			name: "a preset over the joint durability budget",
			change: func(entry map[string]any) {
				breaking := table()
				breaking["hp"] = curve(1440, 4800)
				breaking["defense"] = curve(240, 800)
				entry["stats"] = breaking
			},
			wantIn: "over the budget",
		},
		{
			name: "a preset over a single ceiling",
			change: func(entry map[string]any) {
				breaking := table()
				breaking["speed"] = curve(60, 260)
				entry["stats"] = breaking
			},
			wantIn: "over the ceiling",
		},
		{
			// No affinity can hold three elements, so a preset demanding three
			// could never produce a character that enters a battle.
			name: "a kit demanding three elements",
			change: func(entry map[string]any) {
				entry["skills"] = []string{"riptide", "ember_lance", "gale_slash"}
			},
			wantIn: "demanding 3 elements",
		},
		{
			name:    "duplicate preset id",
			change:  func(map[string]any) {},
			another: baseArchetype(),
			wantIn:  "declared twice",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			entry := baseArchetype()
			test.change(entry)
			declarations := []map[string]any{entry}
			if test.another != nil {
				declarations = append(declarations, test.another)
			}
			_, err := archetypes(t, declarations...)
			if err == nil {
				t.Fatalf("the preset parsed, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

func TestParseArchetypesNeedsEveryDependency(t *testing.T) {
	raw := []byte(`{"archetypes": []}`)
	if _, err := cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Limits: limits(t), Rules: rules(t),
	}); err == nil {
		t.Error("presets were validated without the skill book they name skills from")
	}
	if _, err := cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: skills(t), Rules: rules(t),
	}); err == nil {
		t.Error("presets were validated without a stat budget to check the curves against")
	}
	if _, err := cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: skills(t), Limits: limits(t),
	}); err == nil {
		t.Error("presets were validated without the combat rules the budget is measured with")
	}
}

func TestParseOriginsRejections(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		wantIn string
	}{
		{"no id", `{"origins": [{"title": "A", "medium": "film"}]}`, "origin id is empty"},
		{"non-slug id", `{"origins": [{"id": "A Film", "title": "A", "medium": "film"}]}`, "lowercase letters"},
		{"no title", `{"origins": [{"id": "a-film", "title": " ", "medium": "film"}]}`, "no title"},
		{"no medium", `{"origins": [{"id": "a-film", "title": "A"}]}`, "which medium"},
		{"unknown medium", `{"origins": [{"id": "a-film", "title": "A", "medium": "opera"}]}`, "unknown medium"},
		{"implausible year", `{"origins": [{"id": "a-film", "title": "A", "medium": "film", "year": 1400}]}`, "plausible year"},
		{
			"duplicate id",
			`{"origins": [{"id": "a-film", "title": "A", "medium": "film"}, {"id": "a-film", "title": "B", "medium": "film"}]}`,
			"declared twice",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := cast.ParseOrigins([]byte(test.raw))
			if err == nil {
				t.Fatalf("the catalog parsed, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

// TestMediumRoundTrip holds the wire format: a medium is a name, both ways.
func TestMediumRoundTrip(t *testing.T) {
	for _, medium := range cast.Mediums() {
		raw, err := json.Marshal(medium)
		if err != nil {
			t.Fatalf("encode %s: %v", medium, err)
		}
		if got, want := string(raw), `"`+medium.String()+`"`; got != want {
			t.Errorf("%s encoded as %s, want %s", medium, got, want)
		}
		var back cast.Medium
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
		if back != medium {
			t.Errorf("%s decoded back as %s", medium, back)
		}
	}
	var back cast.Medium
	if err := json.Unmarshal([]byte(`3`), &back); err == nil {
		t.Error("a medium decoded from a number, which would let an inserted constant reinterpret a saved file")
	}
}

func TestValidateImagePath(t *testing.T) {
	accepted := []string{
		"assets/a.svg", "assets/nested/deep/a.png", "a.SVG", "assets/a.PNG",
	}
	for _, image := range accepted {
		if err := cast.ValidateImagePath(image); err != nil {
			t.Errorf("%q was rejected: %v", image, err)
		}
	}
	rejected := []string{
		"", "assets/a.jpg", "assets/a", "/assets/a.svg", "C:/assets/a.svg",
		`assets\a.svg`, "../a.svg", "assets/../../a.svg",
	}
	for _, image := range rejected {
		if err := cast.ValidateImagePath(image); err == nil {
			t.Errorf("%q was accepted", image)
		}
	}
}

// TestMarshalIsStableAndReParses is what the authoring tool depends on: it
// rewrites the whole file on every addition, so the bytes have to be a function
// of the content and nothing else.
func TestMarshalIsStableAndReParses(t *testing.T) {
	second := baseCharacter()
	second["id"] = "a-game.emberling"
	second["name"] = "Emberling"
	second["origin"] = "a-game"
	second["image"] = "assets/a-game/emberling.png"
	second["element"] = "fire"
	second["skills"] = []string{"ember_lance"}
	// Two stages, one with its own picture and one without, so the trip covers
	// both halves of an optional field: the one that has to survive it and the
	// one that has to stay absent.
	second["stages"] = []map[string]any{
		{"name": "Emberling", "min_level": 1, "stats": table()},
		{"name": "Emberlord", "min_level": 40, "stats": table(),
			"image": "assets/a-game/emberlord.png"},
	}

	book, err := parse(t, baseCharacter(), second)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	first, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal twice: %v", err)
	}
	if string(first) != string(again) {
		t.Error("two marshals of the same book differ")
	}
	// Sorted by id, whatever order the characters were declared in.
	if !strings.Contains(string(first), `"id": "a-game.emberling"`) {
		t.Fatalf("the rendering does not name both characters:\n%s", first)
	}
	if strings.Index(string(first), "a-game.emberling") > strings.Index(string(first), "a-series.warden") {
		t.Error("the rendering is not sorted by id")
	}

	reparsed, err := cast.ParseBook(first, deps(t))
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v", err)
	}
	// The comparison is on sorted ids, because Marshal imposes an order and a
	// re-parse then preserves the order it was given: the cast is the same, the
	// declaration order is not.
	if !reflect.DeepEqual(sortedIDs(reparsed.All()), sortedIDs(book.All())) {
		t.Errorf("the round trip changed the cast: %v vs %v",
			sortedIDs(reparsed.All()), sortedIDs(book.All()))
	}
	third, err := reparsed.Marshal()
	if err != nil {
		t.Fatalf("marshal the round trip: %v", err)
	}
	if string(third) != string(first) {
		t.Errorf("a round trip changed the bytes:\n--- first ---\n%s\n--- again ---\n%s", first, third)
	}
	// Every field a character carries has to survive the trip, not just its id.
	original, _ := book.Get("a-game.emberling")
	returned, ok := reparsed.Get("a-game.emberling")
	if !ok {
		t.Fatal("the round trip lost a character")
	}
	if !reflect.DeepEqual(original, returned) {
		t.Errorf("the round trip changed a character:\n%+v\n%+v", original, returned)
	}
	// A stage's picture is written only when it has one. The whole reason the
	// field could be added without moving a golden is that a stage declaring
	// none writes exactly the bytes it did before the field existed.
	if strings.Count(string(first), `"image": "assets/a-game/emberlord.png"`) != 1 {
		t.Errorf("the stage's picture is not written once:\n%s", first)
	}
	if strings.Contains(string(first), `"image": ""`) {
		t.Errorf("a stage with no picture wrote an empty one:\n%s", first)
	}
}

func sortedIDs(characters []cast.Character) []string {
	out := make([]string, 0, len(characters))
	for _, character := range characters {
		out = append(out, character.ID)
	}
	sort.Strings(out)
	return out
}

func TestOriginBookMarshalRoundTrip(t *testing.T) {
	book := origins(t)
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, err := cast.ParseOrigins(raw)
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v", err)
	}
	// Compared after sorting both, because Marshal imposes an order: the
	// catalog is the same, the declaration order it was authored in is not.
	before, after := book.All(), reparsed.All()
	sort.Slice(before, func(i, j int) bool { return before[i].ID < before[j].ID })
	if !reflect.DeepEqual(after, before) {
		t.Errorf("the round trip changed the catalog:\n%+v\n%+v", after, before)
	}
	again, err := reparsed.Marshal()
	if err != nil {
		t.Fatalf("marshal twice: %v", err)
	}
	if string(again) != string(raw) {
		t.Error("a round trip changed the bytes")
	}
	// An id sorts ahead of the declaration order it was authored in, which is
	// the one place ordering is imposed rather than preserved.
	if got := reparsed.All()[0].ID; got != "a-game" {
		t.Errorf("the first origin after a round trip is %q, want %q", got, "a-game")
	}
}

func TestAppendValidatesBeforeItAccepts(t *testing.T) {
	book, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	broken, _ := book.Get("a-series.warden")
	broken.ID = "a-series.other"
	broken.Skills = []string{"meteor"}
	if _, err := book.Append(deps(t), broken); err == nil {
		t.Error("a character naming an unknown skill was appended")
	}
	if got := len(book.All()); got != 1 {
		t.Errorf("the original book now holds %d characters; Append must not mutate it", got)
	}
	good, _ := book.Get("a-series.warden")
	good.ID = "a-series.second"
	good.Name = "Second"
	grown, err := book.Append(deps(t), good)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if got := len(grown.All()); got != 2 {
		t.Errorf("the grown book holds %d characters, want 2", got)
	}
	if got := len(book.All()); got != 1 {
		t.Errorf("the original book now holds %d characters; Append returns a new one", got)
	}
}

// TestResolveAtAStageBoundary is the whole point of progression.Line: the level
// a stage declares is the first level it owns, not the last of the one before.
func TestResolveAtAStageBoundary(t *testing.T) {
	early := table()
	early["hp"] = curve(660, 1500)
	late := table()
	late["hp"] = curve(1100, 2200)
	entry := baseCharacter()
	entry["stages"] = []map[string]any{
		{"name": "Sprout", "min_level": 1, "stats": early},
		{"name": "Bloom", "min_level": 30, "stats": late},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, ok := book.Get("a-series.warden")
	if !ok {
		t.Fatal("the parsed character is not in the book")
	}
	cases := []struct {
		level int
		stage string
	}{
		{1, "Sprout"}, {29, "Sprout"}, {30, "Bloom"}, {31, "Bloom"}, {progression.LevelCap, "Bloom"},
	}
	for _, test := range cases {
		_, stage, err := character.Resolve(test.level)
		if err != nil {
			t.Fatalf("resolve at level %d: %v", test.level, err)
		}
		if stage.Name != test.stage {
			t.Errorf("level %d lands in stage %q, want %q", test.level, stage.Name, test.stage)
		}
	}
	// A level outside the range is an error rather than a clamp: a roster
	// asking for level 0 has a mistake in it, and silently answering with
	// level 1 hides it.
	if _, _, err := character.Resolve(0); err == nil {
		t.Error("level 0 resolved")
	}
	if _, _, err := character.Resolve(progression.LevelCap + 1); err == nil {
		t.Error("a level past the cap resolved")
	}
}

// TestBookDoesNotHandOutItsInternals guards the slices a caller could otherwise
// edit under the book's feet.
func TestBookDoesNotHandOutItsInternals(t *testing.T) {
	book, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	borrowed, _ := book.Get("a-series.warden")
	borrowed.Skills[0] = "meteor"
	borrowed.Stages[0].Name = "vandalised"
	again, _ := book.Get("a-series.warden")
	if again.Skills[0] != "strike" {
		t.Errorf("the book's kit was edited through Get: %v", again.Skills)
	}
	if again.Stages[0].Name != "Warden" {
		t.Errorf("the book's stage line was edited through Get: %v", again.Stages)
	}
}

// TestArchetypeDemandIsDerivedFromTheKit is why Archetype has no authored
// affinity field: a hint would be free to drift from the kit it described, and
// the drift would only surface when a character built from the preset was
// refused by battle.New.
func TestArchetypeDemandIsDerivedFromTheKit(t *testing.T) {
	cases := []struct {
		name     string
		kit      []string
		want     []element.Element
		rendered string
	}{
		{"all neutral", []string{"strike"}, nil, ""},
		{"one element", []string{"strike", "riptide"}, []element.Element{element.Water}, "water"},
		{
			"two, in the order the skills were listed",
			[]string{"gale_slash", "riptide"},
			[]element.Element{element.Wind, element.Water},
			"wind water",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			entry := baseArchetype()
			entry["skills"] = test.kit
			book, err := archetypes(t, entry)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			preset, known := book.Get("sentinel")
			if !known {
				t.Fatal("the preset is not in the book")
			}
			if len(preset.Demands) != len(test.want) {
				t.Fatalf("demands %v, want %v", preset.Demands, test.want)
			}
			for i := range preset.Demands {
				if preset.Demands[i] != test.want[i] {
					t.Fatalf("demands %v, want %v", preset.Demands, test.want)
				}
			}
			if got := strings.Join(preset.DemandNames(), " "); got != test.rendered {
				t.Errorf("DemandNames is %q, want %q", got, test.rendered)
			}
		})
	}
}

// TestArchetypeDemandIsNotAuthorable guards the json:"-" tag. A preset that
// could declare its own demand could declare a wrong one.
func TestArchetypeDemandIsNotAuthorable(t *testing.T) {
	entry := baseArchetype()
	entry["skills"] = []string{"strike", "riptide"}
	entry["demands"] = []string{"fire", "wind"}
	entry["Demands"] = []string{"fire", "wind"}
	book, err := archetypes(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	preset, _ := book.Get("sentinel")
	if len(preset.Demands) != 1 || preset.Demands[0] != element.Water {
		t.Errorf("an authored demand was believed: %v", preset.Demands)
	}
}

// TestBookDoesNotHandOutItsDemand is the same guard as the kit and the stage
// line: a caller may not edit the book through what Get returns.
func TestBookDoesNotHandOutItsDemand(t *testing.T) {
	entry := baseArchetype()
	entry["skills"] = []string{"strike", "riptide"}
	book, err := archetypes(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	borrowed, _ := book.Get("sentinel")
	borrowed.Demands[0] = element.Fire
	again, _ := book.Get("sentinel")
	if again.Demands[0] != element.Water {
		t.Errorf("the preset's demand was edited through Get: %v", again.Demands)
	}
}

// TestCarryRuleAcceptsWhatTheKitAllows is the positive half: a dual affinity is
// what makes a two-element kit carryable at all.
func TestCarryRuleAcceptsWhatTheKitAllows(t *testing.T) {
	entry := baseCharacter()
	// riptide is water and gale_slash is wind; water and wind do not counter
	// each other, so the pair is a legal affinity and carries both.
	entry["skills"] = []string{"strike", "riptide", "gale_slash"}
	entry["element"] = []string{"water", "wind"}
	if _, err := parse(t, entry); err != nil {
		t.Fatalf("a dual affinity covering its kit was refused: %v", err)
	}
	// Dropping the second element leaves the same kit uncarryable.
	entry["element"] = "water"
	_, err := parse(t, entry)
	if err == nil {
		t.Fatal("a single affinity carried a two-element kit")
	}
	if !strings.Contains(err.Error(), "gale_slash") {
		t.Errorf("the rejection is %q, want it to name the skill it cannot carry", err)
	}
	if !strings.Contains(err.Error(), "wind") {
		t.Errorf("the rejection is %q, want it to name the skill's element", err)
	}
}

// The fixtures below add restricted skills to the inline book, so that the
// three halves of a restriction can be exercised where each is enforced: the
// element one here and in the engine, the archetype and character ones here
// only, because a roster entry carries neither.

func restrictedSkills(t *testing.T) *skill.Book {
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
	book, err := skill.ParseBook([]byte(`{
	  "skills": [
	    {"id": "strike", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy"},
	    {"id": "riptide", "element": "water", "range": 2, "pattern": "single",
	     "power": 1600, "accuracy": 900, "target": "enemy"},
	    {"id": "oath", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"elements": ["fire", "metal"]}},
	    {"id": "sentinel_oath", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"archetypes": ["sentinel"]}},
	    {"id": "bulwark_oath", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"archetypes": ["bulwark"]}},
	    {"id": "warden_only", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"characters": ["a-series.warden"]}},
	    {"id": "pair_only", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"characters": ["a-series.warden", "a-series.ghost"]}},
	    {"id": "nobody_only", "element": "neutral", "range": 1, "pattern": "single",
	     "power": 1000, "accuracy": 950, "target": "enemy",
	     "restrict": {"characters": ["a-series.nobody"]}}
	  ]
	}`), skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	return book
}

// restrictedPreset is a preset over the restricted book, whose kit the caller
// chooses — which is what the preset checks are made of.
func restrictedPreset(t *testing.T, kit ...string) (*cast.ArchetypeBook, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"archetypes": []map[string]any{{
		"id": "sentinel", "role": "armour rather than health", "column": 2,
		"stats": table(), "skills": kit,
	}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: restrictedSkills(t), Limits: limits(t), Rules: rules(t),
	})
}

func restrictedDeps(t *testing.T) cast.Deps {
	t.Helper()
	presets, err := restrictedPreset(t, "strike", "riptide")
	if err != nil {
		t.Fatalf("presets: %v", err)
	}
	return cast.Deps{
		Origins: origins(t), Archetypes: presets, Skills: restrictedSkills(t),
		Chart: chart(t), Limits: limits(t), Rules: rules(t),
	}
}

func parseRestricted(t *testing.T, declarations ...map[string]any) (*cast.Book, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"characters": declarations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return cast.ParseBook(raw, restrictedDeps(t))
}

// ghost is a second character of the same origin, for the checks that need two.
func ghost(kit ...string) map[string]any {
	built := baseCharacter()
	built["id"] = "a-series.ghost"
	built["name"] = "Ghost"
	built["image"] = "assets/a-series/ghost.svg"
	built["stages"] = []map[string]any{{"name": "Ghost", "min_level": 1, "stats": table()}}
	built["skills"] = kit
	return built
}

func withKit(kit ...string) map[string]any {
	built := baseCharacter()
	built["skills"] = kit
	return built
}

// TestARestrictionIsEnforcedWhereACharacterIsAuthored is the whole of part one
// from the cast's side: each of the three allowlists accepts and refuses, and
// each refusal names the skill and what the restriction allows, because whoever
// reads it did not necessarily write it.
func TestARestrictionIsEnforcedWhereACharacterIsAuthored(t *testing.T) {
	cases := []struct {
		name    string
		kit     []string
		wants   []string
		refused bool
	}{
		{name: "an unrestricted kit", kit: []string{"strike", "riptide"}},
		{name: "a skill kept for this preset", kit: []string{"strike", "sentinel_oath"}},
		{name: "a skill kept for this character", kit: []string{"strike", "warden_only"}},
		{
			name: "a skill kept for another element", kit: []string{"strike", "oath"},
			refused: true, wants: []string{"oath", "fire or metal", "water/ice"},
		},
		{
			name: "a skill kept for another preset", kit: []string{"strike", "bulwark_oath"},
			refused: true, wants: []string{"bulwark_oath", "bulwark", "sentinel"},
		},
		{
			name: "a skill kept for another character", kit: []string{"strike", "nobody_only"},
			refused: true, wants: []string{"nobody_only", "a-series.nobody"},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseRestricted(t, withKit(test.kit...))
			if !test.refused {
				if err != nil {
					t.Fatalf("%s was refused: %v", test.name, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			for _, want := range test.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal %q does not mention %q", err, want)
				}
			}
		})
	}
}

// TestACharacterAllowlistMayNameSomebodyDeclaredLater is why the name check
// runs after the whole book has been read. A skill kept for two characters is
// carried by the first of them, and the second is further down the file: a check
// made while the book was half-read would refuse ordinary authoring for being in
// the wrong order.
func TestACharacterAllowlistMayNameSomebodyDeclaredLater(t *testing.T) {
	book, err := parseRestricted(t,
		withKit("strike", "pair_only"),
		ghost("strike", "pair_only"),
	)
	if err != nil {
		t.Fatalf("a forward reference should parse: %v", err)
	}
	if got := len(book.All()); got != 2 {
		t.Errorf("the book holds %d characters, want 2", got)
	}

	// The same allowlist with one name nobody answers to is a refusal, and it
	// says which name.
	_, err = parseRestricted(t, withKit("strike", "nobody_only"))
	if err == nil {
		t.Fatal("a skill kept for a character the cast does not hold was accepted")
	}
	if !strings.Contains(err.Error(), "a-series.nobody") {
		t.Errorf("the refusal %q does not name the missing character", err)
	}
}

// TestASkillNobodyCarriesIsNotCheckedAgainstTheCast is the deadlock this
// avoids: a unique skill cannot be written after the character that carries it
// (the character's kit names the skill) and the character cannot be written
// after the skill (the skill names the character), so one of the two orders has
// to work. The skill goes in first, carried by nobody.
func TestASkillNobodyCarriesIsNotCheckedAgainstTheCast(t *testing.T) {
	if _, err := parseRestricted(t, withKit("strike", "riptide")); err != nil {
		t.Fatalf("a cast that carries none of the restricted skills was refused: %v", err)
	}
	if _, err := cast.ParseBook([]byte(`{"characters": []}`), restrictedDeps(t)); err != nil {
		t.Fatalf("an empty cast beside restricted skills was refused: %v", err)
	}
}

// TestAPresetCannotHoldASkillKeptForSomebody is the check that has to live with
// the archetype book, because a preset is a starting point for every character
// built from it: a kit entry only certain characters may carry would refuse
// everyone else, and the refusal would land on the author of the character
// rather than the author of the preset.
func TestAPresetCannotHoldASkillKeptForSomebody(t *testing.T) {
	cases := []struct {
		name    string
		kit     []string
		wants   []string
		refused bool
	}{
		{name: "an unrestricted kit", kit: []string{"strike", "riptide"}},
		{name: "a skill kept for this very preset", kit: []string{"strike", "sentinel_oath"}},
		{name: "a skill kept for an element", kit: []string{"strike", "oath"}},
		{
			name: "a skill kept for one character", kit: []string{"strike", "warden_only"},
			refused: true, wants: []string{"warden_only", "a-series.warden", "shared"},
		},
		{
			name: "a skill kept for two characters", kit: []string{"strike", "pair_only"},
			refused: true, wants: []string{"pair_only", "a-series.ghost"},
		},
		{
			name: "a skill kept for another preset", kit: []string{"strike", "bulwark_oath"},
			refused: true, wants: []string{"bulwark_oath", "bulwark"},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := restrictedPreset(t, test.kit...)
			if !test.refused {
				if err != nil {
					t.Fatalf("%s was refused: %v", test.name, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			for _, want := range test.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal %q does not mention %q", err, want)
				}
			}
		})
	}
}

// TestStageArtFollowsTheFormAndFallsBackToTheCharacter is the whole per-stage
// art feature in one place: a form that names its own picture shows it, and a
// form that names none shows the character's rather than nothing.
//
// The fallback is what makes the field optional, and optional is what keeps
// every character authored before it existed valid. A stage drawing nothing
// would have been the alternative, and it would have shipped as a blank sprite
// rather than as an error.
func TestStageArtFollowsTheFormAndFallsBackToTheCharacter(t *testing.T) {
	entry := baseCharacter()
	entry["stages"] = []map[string]any{
		{"name": "Sprout", "min_level": 1, "stats": table()},
		{"name": "Bloom", "min_level": 30, "stats": table(),
			"image": "assets/a-series/bloom.png"},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, ok := book.Get("a-series.warden")
	if !ok {
		t.Fatal("the parsed character is not in the book")
	}
	cases := []struct {
		level int
		want  string
	}{
		{1, "assets/a-series/warden.svg"},
		{29, "assets/a-series/warden.svg"},
		{30, "assets/a-series/bloom.png"},
		{progression.LevelCap, "assets/a-series/bloom.png"},
	}
	for _, test := range cases {
		_, stage, err := character.Resolve(test.level)
		if err != nil {
			t.Fatalf("resolve at level %d: %v", test.level, err)
		}
		if got := character.StageArt(stage); got != test.want {
			t.Errorf("level %d shows %q, want %q", test.level, got, test.want)
		}
	}
}

// TestArtIsEveryDistinctPictureInDeclarationOrder is what a checker walks, so
// the two properties that matter are that nothing is missed and that nothing is
// counted twice.
func TestArtIsEveryDistinctPictureInDeclarationOrder(t *testing.T) {
	entry := baseCharacter()
	entry["stages"] = []map[string]any{
		// The first stage names none, so it contributes no row of its own.
		{"name": "Sprout", "min_level": 1, "stats": table()},
		{"name": "Bloom", "min_level": 20, "stats": table(), "image": "assets/a-series/bloom.png"},
		// A stage naming the same picture as another adds no row either: a
		// checker asking the filesystem twice about one file would report one
		// missing file as two problems.
		{"name": "Late", "min_level": 40, "stats": table(), "image": "assets/a-series/bloom.png"},
		// And a stage naming the character's own picture is the same case.
		{"name": "Last", "min_level": 50, "stats": table(), "image": "assets/a-series/warden.svg"},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, _ := book.Get("a-series.warden")
	art := character.Art()
	want := []cast.ArtEntry{
		{Stage: "", Image: "assets/a-series/warden.svg"},
		{Stage: "Bloom", Image: "assets/a-series/bloom.png"},
	}
	if !reflect.DeepEqual(art, want) {
		t.Errorf("Art() is %+v, want %+v", art, want)
	}
	// A character with no stage art at all is the ordinary case, and it is one
	// row rather than none.
	plain, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("parse the plain character: %v", err)
	}
	only, _ := plain.Get("a-series.warden")
	if got := only.Art(); len(got) != 1 || got[0].Image != "assets/a-series/warden.svg" {
		t.Errorf("a character with no stage art has Art() %+v, want just its own", got)
	}
}

// TestAStageImageIsCheckedLikeTheCharactersAndSaysWhichStage is the parse-time
// half. The shape of an image path is this package's rule, and a stage's path
// gets the same rule as a character's — otherwise the one place a path is not
// checked is the one place it was added last.
func TestAStageImageIsCheckedLikeTheCharactersAndSaysWhichStage(t *testing.T) {
	cases := []struct {
		name    string
		image   string
		wantErr string
	}{
		{"a backslash", `assets\bloom.png`, "backslash"},
		{"an absolute path", "/assets/bloom.png", "absolute path"},
		{"a climb out of the directory", "assets/../../bloom.png", "climbs out"},
		{"no extension at all", "assets/bloom", "svg"},
	}
	for _, test := range cases {
		entry := baseCharacter()
		entry["stages"] = []map[string]any{
			{"name": "Sprout", "min_level": 1, "stats": table()},
			{"name": "Bloom", "min_level": 30, "stats": table(), "image": test.image},
		}
		_, err := parse(t, entry)
		if err == nil {
			t.Errorf("%s was accepted as a stage image", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.wantErr) {
			t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantErr)
		}
		// Which stage is the half a character-level message cannot give: a
		// character with four stages and one bad path is otherwise a hunt.
		if !strings.Contains(err.Error(), "Bloom") {
			t.Errorf("%s was refused with %q, want it to name the stage", test.name, err)
		}
	}
	// An absent stage image is not an empty one. ValidateImagePath refuses the
	// empty string, so a stage that says nothing must never reach it.
	entry := baseCharacter()
	entry["stages"] = []map[string]any{{"name": "Sprout", "min_level": 1, "stats": table()}}
	if _, err := parse(t, entry); err != nil {
		t.Errorf("a stage with no image was refused: %v", err)
	}
}

// TestPassivesAreCheckedOnACharacterAndOnAPreset is the cross-book half: a trait
// id is checked against the passive book at load, the way a kit is checked
// against the skill book.
//
// Both a character and a preset name traits, and both go through one resolver —
// so this covers the shared rule twice rather than the same rule twice.
func TestPassivesAreCheckedOnACharacterAndOnAPreset(t *testing.T) {
	entry := baseCharacter()
	entry["passives"] = []map[string]any{{"id": "endurance"}}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("a character holding a declared trait was refused: %v", err)
	}
	character, ok := book.Get("a-series.warden")
	if !ok {
		t.Fatal("the parsed character is not in the book")
	}
	// An unstated level resolves to one rather than staying zero, so a caller
	// asking "is this in force" never has to know which of the two it is looking
	// at.
	if !reflect.DeepEqual(character.Passives, []cast.Unlock{{ID: "endurance", AtLevel: 1}}) {
		t.Errorf("the character holds %+v, want the trait it named from level one", character.Passives)
	}

	// Absent is the ordinary case and stays absent rather than becoming empty:
	// the writer omits it, so a cast that names none keeps the bytes it had.
	plain, err := parse(t, baseCharacter())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	only, _ := plain.Get("a-series.warden")
	if only.Passives != nil {
		t.Errorf("a character naming no trait came back with %v", only.Passives)
	}

	for _, test := range []struct {
		name    string
		names   []map[string]any
		wantErr string
	}{
		{"an unknown trait", []map[string]any{{"id": "nobody-wrote-this"}}, "unknown passive"},
		{
			"the same trait twice",
			[]map[string]any{{"id": "endurance"}, {"id": "endurance", "at_level": 20}},
			"twice",
		},
		{
			"a level past the cap",
			[]map[string]any{{"id": "endurance", "at_level": progression.LevelCap + 1}},
			"outside 1..",
		},
		{
			"a negative level",
			[]map[string]any{{"id": "endurance", "at_level": -3}},
			"outside 1..",
		},
	} {
		broken := baseCharacter()
		broken["passives"] = test.names
		_, err := parse(t, broken)
		if err == nil {
			t.Errorf("%s was accepted on a character", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.wantErr) {
			t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantErr)
		}
	}

	// A preset suggesting a trait is where an archetype finally gains a
	// mechanical weight, so it is checked the same way.
	preset := baseArchetype()
	preset["passives"] = []map[string]any{{"id": "endurance", "at_level": 16}}
	presets, err := archetypes(t, preset)
	if err != nil {
		t.Fatalf("a preset suggesting a declared trait was refused: %v", err)
	}
	suggested, ok := presets.Get("sentinel")
	if !ok {
		t.Fatal("the parsed preset is not in the book")
	}
	if !reflect.DeepEqual(suggested.Passives, []cast.Unlock{{ID: "endurance", AtLevel: 16}}) {
		t.Errorf("the preset suggests %+v, want the trait it named", suggested.Passives)
	}
	broken := baseArchetype()
	broken["passives"] = []map[string]any{{"id": "nobody-wrote-this"}}
	if _, err := archetypes(t, broken); err == nil {
		t.Error("a preset suggesting an undeclared trait was accepted")
	}
}

// TestNamingAPassiveWithoutTheBookIsRefused keeps the check from being optional
// by accident. A caller that forgot to wire the book up would otherwise get a
// character whose traits were never verified, and it would load cleanly.
func TestNamingAPassiveWithoutTheBookIsRefused(t *testing.T) {
	deps := deps(t)
	deps.Passives = nil
	entry := baseCharacter()
	entry["passives"] = []map[string]any{{"id": "endurance"}}
	raw, err := json.Marshal(map[string]any{"characters": []map[string]any{entry}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := cast.ParseBook(raw, deps); err == nil {
		t.Error("a character naming a trait parsed with no passive book to check against")
	}
	// And a cast that names none still loads without one, which is what let the
	// field arrive without every caller being found.
	bare, err := json.Marshal(map[string]any{"characters": []map[string]any{baseCharacter()}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := cast.ParseBook(bare, deps); err != nil {
		t.Errorf("a cast naming no trait was refused without the book: %v", err)
	}
}
