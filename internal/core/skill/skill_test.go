package skill_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

func deps(t *testing.T) skill.Deps {
	t.Helper()
	patterns, err := pattern.ParseBook([]byte(`{
	  "max_targets": 3, "splash_power": 500,
	  "patterns": [
	    {"name": "single", "splash": []},
	    {"name": "column", "splash": [["up"], ["down"]]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "burn", "category": "dot", "max_stacks": 2, "duration": 2, "tick_power": 800},
	    {"id": "weaken", "category": "stat_debuff", "max_stacks": 3, "duration": 3,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": -300}]},
	    {"id": "block", "category": "shield", "max_stacks": 3, "duration": 2}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	return skill.Deps{Patterns: patterns, Statuses: statuses}
}

// base is a declaration that parses, so each negative case can change exactly
// one thing and prove that change is what breaks it.
func base() map[string]any {
	return map[string]any{
		"id": "ember_lance", "element": "fire", "range": 2, "pattern": "single",
		"power": 1800, "strikes": 1, "accuracy": 900, "cooldown": 2, "target": "enemy",
	}
}

func parse(t *testing.T, declarations ...map[string]any) (*skill.Book, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"skills": declarations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return skill.ParseBook(raw, deps(t))
}

func TestParseBookAcceptsTheBaseDeclaration(t *testing.T) {
	book, err := parse(t, base())
	if err != nil {
		t.Fatalf("the base declaration should parse: %v", err)
	}
	found, err := book.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found.Element != element.Fire || found.Range != 2 || found.Power != 1800 {
		t.Errorf("resolved %+v", found)
	}
	// Scaling defaults to the caster's current attack, which is what almost
	// every skill wants.
	if found.Scaling.Stat != progression.Attack || found.Scaling.Source != combat.CurrentStat {
		t.Errorf("the default scaling is %+v, want current attack", found.Scaling)
	}
	if found.Guaranteed() {
		t.Error("a 900 accuracy skill reports itself as guaranteed")
	}
	if _, err := book.Lookup("nowhere"); err == nil {
		t.Error("an unknown id was accepted")
	}
}

func TestParseBookRejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{"no id", func(s map[string]any) { s["id"] = "" }, "needs an id"},
		{"an unknown element", func(s map[string]any) { s["element"] = "plasma" }, "unknown element"},
		{"an unknown target side", func(s map[string]any) { s["target"] = "everyone" }, "unknown target side"},
		{"an unknown shape", func(s map[string]any) { s["pattern"] = "spiral" }, "unknown pattern"},
		{"negative power", func(s map[string]any) { s["power"] = -100 }, "want zero or more"},
		{"negative strikes", func(s map[string]any) { s["strikes"] = -2 }, "want zero or more"},
		{"accuracy past certainty", func(s map[string]any) { s["accuracy"] = 1200 }, "parts per thousand"},
		{"negative accuracy", func(s map[string]any) { s["accuracy"] = -5 }, "parts per thousand"},
		{"damage that can never land", func(s map[string]any) { s["accuracy"] = 0 }, "can never connect"},
		{"negative cooldown", func(s map[string]any) { s["cooldown"] = -1 }, "want zero or more"},
		{"range of zero on an enemy skill", func(s map[string]any) { s["range"] = 0 }, "want between 1 and"},
		{"range past the board", func(s map[string]any) { s["range"] = 9 }, "want between 1 and"},
		{"a skill that does nothing at all", func(s map[string]any) {
			s["power"] = 0
			s["accuracy"] = 1000
		}, "wasted turn"},
		{"a self skill with a range", func(s map[string]any) {
			s["target"] = "self"
			s["range"] = 2
		}, "declares a range"},
		{"a self skill with an area shape", func(s map[string]any) {
			s["target"] = "self"
			s["range"] = 0
			s["pattern"] = "column"
		}, "covers 3 cells"},
		{"scaling off health", func(s map[string]any) {
			s["scaling"] = map[string]any{"stat": "hp"}
		}, "scales off health"},
		{"an unknown scaling stat", func(s map[string]any) {
			s["scaling"] = map[string]any{"stat": "luck"}
		}, "unknown scaling stat"},
		{"an unknown scaling source", func(s map[string]any) {
			s["scaling"] = map[string]any{"stat": "speed", "source": "future"}
		}, `want "base" or "current"`},
		{"an unknown status applied", func(s map[string]any) {
			s["applies"] = []any{map[string]any{"status": "curse", "chance": 500}}
		}, "unknown status"},
		{"a status applied with no chance", func(s map[string]any) {
			s["applies"] = []any{map[string]any{"status": "burn", "chance": 0}}
		}, "want between 1 and"},
		{"a status applied past certainty", func(s map[string]any) {
			s["applies"] = []any{map[string]any{"status": "burn", "chance": 1400}}
		}, "want between 1 and"},
		{"the same status applied twice", func(s map[string]any) {
			s["applies"] = []any{
				map[string]any{"status": "burn", "chance": 500},
				map[string]any{"status": "burn", "chance": 300},
			}
		}, "appears twice"},
		{"more stacks than the status allows", func(s map[string]any) {
			s["applies"] = []any{map[string]any{"status": "burn", "chance": 500, "stacks": 4}}
		}, "caps at 2"},
		{"a condition on an unknown status", func(s map[string]any) {
			s["requires"] = map[string]any{"status": "curse", "min_stacks": 1, "bonus_power": 400}
		}, "unknown status"},
		{"a condition needing more stacks than exist", func(s map[string]any) {
			s["requires"] = map[string]any{"status": "burn", "min_stacks": 5, "bonus_power": 400}
		}, "caps at 2"},
		{"a condition with a negative bonus", func(s map[string]any) {
			s["requires"] = map[string]any{"status": "burn", "min_stacks": 1, "bonus_power": -200}
		}, "want zero or more"},
		{"a consume for no bonus", func(s map[string]any) {
			s["requires"] = map[string]any{"status": "burn", "min_stacks": 1, "bonus_power": 0, "consume": true}
		}, "for nothing"},
		{"a cleanse naming no categories", func(s map[string]any) {
			s["strips"] = map[string]any{"categories": []string{}, "stacks": 2}
		}, "names no categories"},
		{"a cleanse of no stacks", func(s map[string]any) {
			s["strips"] = map[string]any{"categories": []string{"dot"}, "stacks": 0}
		}, "want at least 1"},
		{"a cleanse of an unknown category", func(s map[string]any) {
			s["strips"] = map[string]any{"categories": []string{"hex"}, "stacks": 2}
		}, "unknown status category"},
		{"a cleanse naming a category twice", func(s map[string]any) {
			s["strips"] = map[string]any{"categories": []string{"dot", "dot"}, "stacks": 2}
		}, "twice"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			declaration := base()
			testCase.mutate(declaration)
			_, err := parse(t, declaration)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestParseBookStructuralRejects(t *testing.T) {
	if _, err := skill.ParseBook([]byte("{"), deps(t)); err == nil {
		t.Error("malformed JSON was accepted")
	}
	if _, err := parse(t); err == nil {
		t.Error("an empty book was accepted")
	}
	if _, err := parse(t, base(), base()); err == nil {
		t.Error("a duplicate id was accepted")
	}
	raw, err := json.Marshal(map[string]any{"skills": []map[string]any{base()}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, missing := range []skill.Deps{
		{},
		{Patterns: deps(t).Patterns},
		{Statuses: deps(t).Statuses},
	} {
		if _, err := skill.ParseBook(raw, missing); err == nil {
			t.Error("a book was validated without both dependency books")
		}
	}
}

func TestSelfAndAllySkillsAreAccepted(t *testing.T) {
	shield := base()
	shield["id"] = "guard_wall"
	shield["target"] = "self"
	shield["range"] = 0
	shield["power"] = 0
	shield["strikes"] = 0
	shield["accuracy"] = 1000
	shield["self_applies"] = []any{map[string]any{"status": "block", "chance": 1000, "stacks": 2}}

	cleanser := base()
	cleanser["id"] = "purify"
	cleanser["target"] = "ally"
	cleanser["power"] = 0
	cleanser["strikes"] = 0
	cleanser["accuracy"] = 1000
	cleanser["strips"] = map[string]any{"categories": []string{"dot", "stat_debuff"}, "stacks": 3}

	book, err := parse(t, shield, cleanser)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	guard, err := book.Lookup("guard_wall")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if guard.Target != skill.Self || !guard.Guaranteed() {
		t.Errorf("resolved %+v", guard)
	}
	if len(guard.SelfApplies) != 1 || guard.SelfApplies[0].Stacks != 2 {
		t.Errorf("the shield applies %+v", guard.SelfApplies)
	}
	purify, err := book.Lookup("purify")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if purify.Strips == nil || len(purify.Strips.Categories) != 2 || purify.Strips.Stacks != 3 {
		t.Errorf("the cleanse is %+v", purify.Strips)
	}
	if purify.Target != skill.Ally {
		t.Errorf("the cleanse targets %s, want ally", purify.Target)
	}
}

// TestApplicationChanceIsFixed is the decision that keeps landing a hit and
// inflicting a status as two separate questions. Nothing in the engine reads or
// alters the chance, so it stays a property of the skill.
func TestApplicationChanceIsFixed(t *testing.T) {
	declaration := base()
	declaration["applies"] = []any{map[string]any{"status": "burn", "chance": 700}}
	book, err := parse(t, declaration)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found, err := book.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(found.Applies) != 1 {
		t.Fatalf("the skill applies %+v", found.Applies)
	}
	if got, want := found.Applies[0].Chance, 700; got != want {
		t.Errorf("the chance is %d, want %d", got, want)
	}
	if got, want := found.Applies[0].Stacks, 1; got != want {
		t.Errorf("an unset stack count became %d, want %d", got, want)
	}
	// A 900 accuracy skill with a 700 chance inflicts the status on 63 percent
	// of casts, and the two figures never merge into one.
	if found.Accuracy == found.Applies[0].Chance {
		t.Error("the accuracy and the application chance are the same number, which hides the distinction")
	}
}

func TestPowerAgainstAppliesTheCondition(t *testing.T) {
	declaration := base()
	declaration["requires"] = map[string]any{"status": "poison", "min_stacks": 2, "bonus_power": 500}
	book, err := parse(t, declaration)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found, err := book.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	cases := []struct {
		stacks    int
		want      int
		amplified bool
	}{
		{0, 1800, false},
		{1, 1800, false},
		{2, 2300, true},
		{3, 2300, true},
	}
	for _, testCase := range cases {
		if got := found.PowerAgainst(testCase.stacks); got != testCase.want {
			t.Errorf("against %d stacks the power is %d, want %d", testCase.stacks, got, testCase.want)
		}
		if got := found.Amplified(testCase.stacks); got != testCase.amplified {
			t.Errorf("against %d stacks amplified is %v, want %v", testCase.stacks, got, testCase.amplified)
		}
	}

	// A skill with no condition never amplifies.
	plain, err := parse(t, base())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	unconditional, err := plain.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	for _, stacks := range []int{0, 3, 99} {
		if unconditional.Amplified(stacks) {
			t.Errorf("a skill with no condition amplified against %d stacks", stacks)
		}
		if got := unconditional.PowerAgainst(stacks); got != 1800 {
			t.Errorf("a skill with no condition landed at %d power", got)
		}
	}
}

func TestMinStacksDefaultsToOne(t *testing.T) {
	declaration := base()
	declaration["requires"] = map[string]any{"status": "poison", "bonus_power": 400}
	book, err := parse(t, declaration)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found, _ := book.Lookup("ember_lance")
	if found.Requires == nil || found.Requires.MinStacks != 1 {
		t.Errorf("the condition is %+v, want one stack", found.Requires)
	}
}

func TestTotalPowerComparesAcrossStrikeCounts(t *testing.T) {
	single := base()
	single["power"] = 1800
	single["strikes"] = 1
	triple := base()
	triple["id"] = "flurry"
	triple["power"] = 600
	triple["strikes"] = 3
	unset := base()
	unset["id"] = "unset"
	unset["power"] = 1800
	unset["strikes"] = 0

	book, err := parse(t, single, triple, unset)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, testCase := range []struct {
		id            string
		want, strikes int
	}{
		{"ember_lance", 1800, 1},
		{"flurry", 1800, 3},
		{"unset", 1800, 1},
	} {
		found, err := book.Lookup(testCase.id)
		if err != nil {
			t.Fatalf("lookup %s: %v", testCase.id, err)
		}
		if got := found.TotalPower(); got != testCase.want {
			t.Errorf("%s totals %d power, want %d", testCase.id, got, testCase.want)
		}
		if got := found.StrikeCount(); got != testCase.strikes {
			t.Errorf("%s has %d strikes, want %d", testCase.id, got, testCase.strikes)
		}
	}
}

func TestSpeedScalingOffTheBaseStatIsAccepted(t *testing.T) {
	declaration := base()
	declaration["id"] = "swift_edge"
	declaration["power"] = 9600
	declaration["scaling"] = map[string]any{"stat": "speed", "source": "base"}
	book, err := parse(t, declaration)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found, _ := book.Lookup("swift_edge")
	if found.Scaling.Stat != progression.Speed || found.Scaling.Source != combat.BaseStat {
		t.Errorf("the scaling is %+v, want base speed", found.Scaling)
	}
}

func TestSkillsReturnsACopy(t *testing.T) {
	book, err := parse(t, base())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	book.Skills()[0] = skill.Skill{ID: "tampered"}
	if got := book.Skills()[0].ID; got != "ember_lance" {
		t.Errorf("the book was modified through its own accessor, first skill is now %q", got)
	}
}

func TestSideNames(t *testing.T) {
	for _, testCase := range []struct {
		side skill.Side
		name string
	}{{skill.Enemy, "enemy"}, {skill.Ally, "ally"}, {skill.Self, "self"}} {
		if got := testCase.side.String(); got != testCase.name {
			t.Errorf("%d renders as %q, want %q", testCase.side, got, testCase.name)
		}
		parsed, err := skill.ParseSide(testCase.name)
		if err != nil || parsed != testCase.side {
			t.Errorf("ParseSide(%q) gave %v, %v", testCase.name, parsed, err)
		}
	}
	if _, err := skill.ParseSide("everyone"); err == nil {
		t.Error("an unknown side name was accepted")
	}
	if got := skill.Side(9).String(); !strings.Contains(got, "9") {
		t.Errorf("an undeclared side renders as %q", got)
	}
}

// TestCanCarry is the single declaration of a rule two packages apply:
// battle.enlist refuses a roster entry that breaks it and cast.ParseBook
// refuses an authored character that breaks it. If either one grew its own copy
// of the predicate, a character could write cleanly and then be refused at load.
func TestCanCarry(t *testing.T) {
	book := mustBook(t)
	strike, err := book.Lookup("strike")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	ember, err := book.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	water, err := element.Single(element.Water)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	dual, err := element.Dual(element.Water, element.Ice)
	if err != nil {
		t.Fatalf("dual: %v", err)
	}
	neutral, err := element.Single(element.Neutral)
	if err != nil {
		t.Fatalf("single: %v", err)
	}

	// A neutral skill is universal, which is why every kit can start with one.
	for _, affinity := range []element.Affinity{fire, water, dual, neutral} {
		if !skill.CanCarry(affinity, strike) {
			t.Errorf("%s cannot carry the neutral skill %q", affinity, strike.ID)
		}
	}
	if !skill.CanCarry(fire, ember) {
		t.Errorf("fire cannot carry %q", ember.ID)
	}
	if skill.CanCarry(water, ember) {
		t.Errorf("water carries the fire skill %q", ember.ID)
	}
	if skill.CanCarry(neutral, ember) {
		t.Errorf("a neutral unit carries the fire skill %q", ember.ID)
	}
	// A second element buys a second line of skills, which is the whole reason
	// to carry one.
	fireAndMetal, err := element.Dual(element.Fire, element.Metal)
	if err != nil {
		t.Fatalf("dual: %v", err)
	}
	if !skill.CanCarry(fireAndMetal, ember) {
		t.Errorf("%s cannot carry %q", fireAndMetal, ember.ID)
	}
}

func TestDemands(t *testing.T) {
	book := mustBook(t)
	kit := func(ids ...string) []skill.Skill {
		t.Helper()
		out := make([]skill.Skill, 0, len(ids))
		for _, id := range ids {
			found, err := book.Lookup(id)
			if err != nil {
				t.Fatalf("lookup %q: %v", id, err)
			}
			out = append(out, found)
		}
		return out
	}
	cases := []struct {
		name string
		ids  []string
		want []element.Element
	}{
		{"an all-neutral kit demands nothing", []string{"strike"}, nil},
		{"one element", []string{"strike", "ember_lance"}, []element.Element{element.Fire}},
		{
			"a repeated element is demanded once",
			[]string{"ember_lance", "cinder_burst"},
			[]element.Element{element.Fire},
		},
		{
			// Listing order, not element declaration order: nothing here may
			// depend on a map's iteration.
			"two elements in the order the skills were listed",
			[]string{"riptide", "ember_lance"},
			[]element.Element{element.Water, element.Fire},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := skill.Demands(kit(test.ids...))
			if len(got) != len(test.want) {
				t.Fatalf("demands %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Errorf("demands %v, want %v", got, test.want)
				}
			}
		})
	}
}

// mustBook is a small book with a neutral, a fire, a second fire and a water
// skill, which is enough to exercise both halves of the carry rule.
func mustBook(t *testing.T) *skill.Book {
	t.Helper()
	book, err := parse(t,
		merge(base(), map[string]any{"id": "strike", "element": "neutral", "power": 1000}),
		base(),
		merge(base(), map[string]any{"id": "cinder_burst"}),
		merge(base(), map[string]any{"id": "riptide", "element": "water"}),
	)
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	return book
}

func merge(into, extra map[string]any) map[string]any {
	for key, value := range extra {
		into[key] = value
	}
	return into
}

// TestARestrictionParsesTheHalfThisPackageCanSee is the layering of a
// restriction, checked from the outside: element names are resolved here, and
// archetype and character names are carried as they were written because the
// books that declare them are one layer up and importing them would be a cycle.
func TestARestrictionParsesTheHalfThisPackageCanSee(t *testing.T) {
	book, err := parse(t, merge(base(), map[string]any{
		"restrict": map[string]any{
			"elements":   []string{"fire", "metal"},
			"archetypes": []string{"bulwark"},
			"characters": []string{"example.adept"},
		},
	}))
	if err != nil {
		t.Fatalf("a full restriction should parse: %v", err)
	}
	found, err := book.Lookup("ember_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found.Restrict == nil {
		t.Fatal("the restriction was dropped")
	}
	if got := found.Restrict.ElementNames(); len(got) != 2 || got[0] != "fire" || got[1] != "metal" {
		t.Errorf("the element allowlist resolved to %v", got)
	}
	if !found.Restrict.AllowsArchetype("bulwark") || found.Restrict.AllowsArchetype("duelist") {
		t.Errorf("the archetype allowlist %v admits the wrong presets", found.Restrict.Archetypes)
	}
	if !found.Restrict.AllowsCharacter("example.adept") || found.Restrict.AllowsCharacter("example.other") {
		t.Errorf("the character allowlist %v admits the wrong characters", found.Restrict.Characters)
	}
	if !found.Restrict.NamesCharacters() {
		t.Error("a skill naming one character does not report itself as belonging to somebody")
	}
}

// TestAnAbsentRestrictionRestrictsNothing covers the nil receiver every caller
// relies on, so that nobody has to ask whether there is a restriction before
// asking what it says.
func TestAnAbsentRestrictionRestrictsNothing(t *testing.T) {
	book := mustBook(t)
	strike, err := book.Lookup("strike")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if strike.Restrict != nil {
		t.Fatalf("an undeclared restriction resolved to %+v", strike.Restrict)
	}
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	if !strike.Restrict.AllowsElement(fire) || !strike.Restrict.AllowsArchetype("anything") ||
		!strike.Restrict.AllowsCharacter("anyone") {
		t.Error("an absent restriction refused somebody")
	}
	if strike.Restrict.NamesCharacters() || strike.Restrict.ElementNames() != nil {
		t.Error("an absent restriction reported a list")
	}
}

// TestARestrictionRejects is the shape of the refusals, and the empty list is
// the one that matters most: an allowlist nobody satisfies is a mistake every
// time, and reading it as "unrestricted" would turn one skill nobody may carry
// into one skill everybody may.
func TestARestrictionRejects(t *testing.T) {
	cases := []struct {
		name   string
		block  map[string]any
		expect string
	}{
		{"an empty character list", map[string]any{"characters": []string{}}, "empty list"},
		{"an empty element list", map[string]any{"elements": []string{}}, "empty list"},
		{"an empty archetype list", map[string]any{"archetypes": []string{}}, "empty list"},
		{"a block with no lists at all", map[string]any{}, "names no lists"},
		{"an element that is not one", map[string]any{"elements": []string{"custard"}}, "custard"},
		{"a repeated element", map[string]any{"elements": []string{"fire", "fire"}}, "twice"},
		{"a repeated character", map[string]any{"characters": []string{"a", "a"}}, "twice"},
		{"a blank name", map[string]any{"archetypes": []string{""}}, "empty name"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, merge(base(), map[string]any{"restrict": test.block}))
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.expect) {
				t.Errorf("the refusal was %q, want it to mention %q", err, test.expect)
			}
			if !strings.Contains(err.Error(), "ember_lance") {
				t.Errorf("the refusal %q does not name the skill", err)
			}
		})
	}
}

// TestWhyCannotCarryTellsTheTwoElementRefusalsApart is why the answer is a
// classification rather than a boolean: one of the two is fixed by taking the
// skill's element and the other cannot be, because the skill's element is
// already shared.
func TestWhyCannotCarryTellsTheTwoElementRefusalsApart(t *testing.T) {
	book, err := parse(t,
		// A neutral skill kept for two elements. Neutral is the case a
		// restriction exists for: CanCarry lets every affinity carry a neutral
		// skill, so there is nowhere else to say who it belongs to.
		merge(base(), map[string]any{
			"id": "oath", "element": "neutral",
			"restrict": map[string]any{"elements": []string{"fire", "metal"}},
		}),
		// A fire skill kept for fire units carrying metal as well.
		merge(base(), map[string]any{
			"id":       "forge_lance",
			"restrict": map[string]any{"elements": []string{"metal"}},
		}),
	)
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	oath, err := book.Lookup("oath")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	lance, err := book.Lookup("forge_lance")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	water, err := element.Single(element.Water)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	fireAndMetal, err := element.Dual(element.Fire, element.Metal)
	if err != nil {
		t.Fatalf("dual: %v", err)
	}
	for _, test := range []struct {
		name     string
		affinity element.Affinity
		carried  skill.Skill
		want     skill.CarryRefusal
	}{
		{"fire takes a neutral skill kept for fire", fire, oath, skill.CarryAllowed},
		{"water is refused the same skill", water, oath, skill.CarryElementRestricted},
		{"fire alone is refused a fire skill kept for metal", fire, lance, skill.CarryElementRestricted},
		{"fire and metal takes it", fireAndMetal, lance, skill.CarryAllowed},
		{"water is refused it for its element first", water, lance, skill.CarryWrongElement},
	} {
		if got := skill.WhyCannotCarry(test.affinity, test.carried); got != test.want {
			t.Errorf("%s: got refusal %d, want %d", test.name, got, test.want)
		}
		if got, want := skill.CanCarry(test.affinity, test.carried), test.want == skill.CarryAllowed; got != want {
			t.Errorf("%s: CanCarry said %v", test.name, got)
		}
	}
}

// TestMarshalIsLosslessForEveryBlockASkillCanDeclare is the property an
// authoring tool rests on. It writes the whole book back on every addition, and
// a form that authors nine fields must not quietly drop the four blocks it does
// not ask about.
//
// The comparison is over the resolved skills rather than the bytes, because the
// parser normalises a few things — an omitted stack count is one stack — and the
// question being asked is whether the *content* survives, not whether the file
// is written the way it was typed.
func TestMarshalIsLosslessForEveryBlockASkillCanDeclare(t *testing.T) {
	declarations := []map[string]any{
		// Nine core fields and nothing else, which is the common case.
		merge(base(), map[string]any{"id": "plain"}),
		// Every optional block at once, so a block cannot be dropped by
		// another block being present.
		merge(base(), map[string]any{
			"id": "everything", "element": "fire", "strikes": 3, "power": 600,
			"scaling":      map[string]any{"stat": "speed", "source": "base"},
			"applies":      []map[string]any{{"status": "poison", "chance": 650, "stacks": 2}},
			"self_applies": []map[string]any{{"status": "block", "chance": 1000, "stacks": 2}},
			"requires": map[string]any{
				"status": "burn", "min_stacks": 2, "bonus_power": 400, "consume": true,
			},
			"strips":   map[string]any{"categories": []string{"dot", "stat_debuff"}, "stacks": 2},
			"restrict": map[string]any{"elements": []string{"fire", "metal"}},
		}),
		// A restriction with all three lists, and a condition that does not
		// consume, so the false half of that flag is covered too.
		merge(base(), map[string]any{
			"id": "kept", "requires": map[string]any{"status": "poison", "bonus_power": 500},
			"restrict": map[string]any{
				"elements":   []string{"fire"},
				"archetypes": []string{"bulwark", "sentinel"},
				"characters": []string{"a.one"},
			},
		}),
	}
	book, err := parse(t, declarations...)
	if err != nil {
		t.Fatalf("the fixtures should parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, err := skill.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("the marshalled book does not parse: %v\n%s", err, raw)
	}
	if !reflect.DeepEqual(reparsed.Skills(), book.Skills()) {
		t.Errorf("the trip through the file changed the book:\n%+v\n%+v",
			reparsed.Skills(), book.Skills())
	}
	// And again, so that the bytes are a function of the content rather than of
	// how many times they have been written.
	again, err := reparsed.Marshal()
	if err != nil {
		t.Fatalf("marshal the reparsed book: %v", err)
	}
	if string(again) != string(raw) {
		t.Errorf("a second write produced different bytes:\n%s\n%s", again, raw)
	}
	// The order is the order it was declared in, because skills.golden's table
	// is that order and it is a design record rather than a listing.
	for i, want := range []string{"plain", "everything", "kept"} {
		if got := reparsed.Skills()[i].ID; got != want {
			t.Errorf("skill %d came back as %q, want %q", i, got, want)
		}
	}
}

// TestAppendValidatesLikeAParse is what keeps a written book a loadable one: the
// addition goes through the parser rather than past it.
func TestAppendValidatesLikeAParse(t *testing.T) {
	book := mustBook(t)
	grown, err := book.Append(deps(t), skill.Skill{
		ID: "oath", Element: element.Neutral, Range: 1, Pattern: "single",
		Power: 1000, Strikes: 1, Accuracy: 950, Target: skill.Enemy,
		Scaling:  skill.Scaling{Stat: progression.Attack, Source: combat.CurrentStat},
		Restrict: &skill.Restriction{Elements: []element.Element{element.Fire}},
	})
	if err != nil {
		t.Fatalf("append a legal skill: %v", err)
	}
	if got := len(grown.Skills()); got != len(book.Skills())+1 {
		t.Errorf("the grown book holds %d skills", got)
	}
	if len(book.Skills()) != 4 {
		t.Error("Append changed the book it was called on")
	}
	// A skill the parser would refuse never becomes a book, so it can never
	// become a file either.
	if _, err := book.Append(deps(t), skill.Skill{ID: "hollow"}); err == nil {
		t.Error("a skill with no power and no effect was appended")
	}
	if _, err := book.Append(deps(t), skill.Skill{
		ID: "strike", Element: element.Neutral, Range: 1, Pattern: "single",
		Power: 1000, Strikes: 1, Accuracy: 950, Target: skill.Enemy,
	}); err == nil {
		t.Error("an id already in the book was appended")
	}
}
