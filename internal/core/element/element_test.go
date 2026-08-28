package element_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
)

type cycleDecl struct {
	Name  string   `json:"name"`
	Chain []string `json:"chain"`
}

type fixture struct {
	Multipliers element.Multipliers `json:"multipliers"`
	Cycles      []cycleDecl         `json:"cycles"`
	Mutual      [][]string          `json:"mutual"`
	Inert       []string            `json:"inert"`
}

// valid returns a chart declaration that parses, so each negative test can
// change exactly one thing and prove that change is what breaks it.
func valid() fixture {
	return fixture{
		Multipliers: element.Multipliers{Advantage: 1500, Neutral: 1000, Disadvantage: 667},
		Cycles: []cycleDecl{
			{Name: "organic", Chain: []string{"water", "fire", "grass"}},
		},
		Mutual: [][]string{{"light", "dark"}},
	}
}

// encode fills the inert list with every element the fixture does not mention,
// because a chart has to classify all of them, then marshals the result.
func (f fixture) encode(t *testing.T) []byte {
	t.Helper()
	mentioned := make(map[string]bool)
	for _, cycle := range f.Cycles {
		for _, name := range cycle.Chain {
			mentioned[name] = true
		}
	}
	for _, pair := range f.Mutual {
		for _, name := range pair {
			mentioned[name] = true
		}
	}
	for _, name := range f.Inert {
		mentioned[name] = true
	}
	for _, member := range element.All() {
		if !mentioned[member.String()] {
			f.Inert = append(f.Inert, member.String())
		}
	}
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

func TestParseChartAcceptsTheBaseFixture(t *testing.T) {
	if _, err := element.ParseChart(valid().encode(t)); err != nil {
		t.Fatalf("base fixture should parse: %v", err)
	}
}

func TestParseChartRejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*fixture)
		wantErr string
	}{
		{"unknown element in a cycle", func(f *fixture) {
			f.Cycles[0].Chain[1] = "plasma"
		}, "unknown element"},
		{"cycle shorter than three", func(f *fixture) {
			f.Cycles[0].Chain = []string{"water", "fire"}
		}, "at least 3"},
		{"element listed twice in a row", func(f *fixture) {
			f.Cycles[0].Chain = []string{"water", "water", "fire", "grass"}
		}, "beats itself"},
		{"reciprocal relation outside a mutual pair", func(f *fixture) {
			f.Cycles[0].Chain = []string{"water", "fire", "water", "grass"}
		}, "not declared a mutual pair"},
		{"the same relation declared twice", func(f *fixture) {
			f.Cycles = append(f.Cycles, cycleDecl{Name: "copy", Chain: []string{"water", "fire", "grass"}})
		}, "more than once"},
		{"mutual pair with three members", func(f *fixture) {
			f.Mutual = [][]string{{"light", "dark", "water"}}
		}, "want 2"},
		{"element both cycled and inert", func(f *fixture) {
			f.Inert = []string{"water"}
		}, "want exactly 1"},
		{"element left unclassified", func(f *fixture) {
			f.Inert = []string{"__skip__"}
		}, "unknown element"},
		{"advantage not above neutral", func(f *fixture) {
			f.Multipliers.Advantage = 900
		}, "does not exceed neutral"},
		{"disadvantage not below neutral", func(f *fixture) {
			f.Multipliers.Disadvantage = 1200
		}, "not between 0 and neutral"},
		{"neutral multiplier of zero", func(f *fixture) {
			f.Multipliers.Neutral = 0
		}, "want a positive value"},
		{"cycles of different lengths leave uneven degrees", func(f *fixture) {
			f.Cycles = []cycleDecl{
				{Name: "three", Chain: []string{"water", "fire", "grass"}},
				{Name: "overlap", Chain: []string{"water", "ground", "ice"}},
			}
		}, "the cycles are uneven"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			declaration := valid()
			testCase.mutate(&declaration)
			_, err := element.ParseChart(declaration.encode(t))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestParseChartRejectsMalformedJSON(t *testing.T) {
	if _, err := element.ParseChart([]byte("{")); err == nil {
		t.Fatal("want an error for truncated JSON, got none")
	}
}

func TestRelationsAreConsistentInBothDirections(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	multipliers := chart.Multipliers()
	for _, attacker := range element.All() {
		for _, defender := range element.All() {
			forward := chart.Multiplier(attacker, defender)
			backward := chart.Multiplier(defender, attacker)
			switch forward {
			case multipliers.Advantage:
				// Only a mutual pair may be strong in both directions.
				if backward != multipliers.Disadvantage && backward != multipliers.Advantage {
					t.Errorf("%s beats %s but the reverse multiplier is %d", attacker, defender, backward)
				}
			case multipliers.Disadvantage:
				if backward != multipliers.Advantage {
					t.Errorf("%s is weak to %s but the reverse multiplier is %d", attacker, defender, backward)
				}
			case multipliers.Neutral:
				if backward != multipliers.Neutral {
					t.Errorf("%s is neutral against %s but the reverse multiplier is %d", attacker, defender, backward)
				}
			default:
				t.Errorf("%s against %s produced unexpected multiplier %d", attacker, defender, forward)
			}
		}
	}
}

func TestSelfMatchupIsAlwaysNeutral(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, member := range element.All() {
		if got, want := chart.Multiplier(member, member), chart.Multipliers().Neutral; got != want {
			t.Errorf("%s against itself: %d, want %d", member, got, want)
		}
	}
}

func TestScaleUsesIntegerMath(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		attacker, defender element.Element
		damage, want       int64
	}{
		{element.Water, element.Fire, 1000, 1500},
		{element.Fire, element.Water, 1000, 667},
		{element.Water, element.Ice, 1000, 1000},
		{element.Light, element.Dark, 1000, 1500},
		{element.Dark, element.Light, 1000, 1500},
		{element.Neutral, element.Water, 1000, 1000},
		// Integer division truncates rather than rounding.
		{element.Water, element.Fire, 1, 1},
		{element.Fire, element.Water, 1, 0},
		{element.Fire, element.Water, 3, 2},
	}
	for _, testCase := range cases {
		got := chart.Scale(testCase.damage, testCase.attacker, testCase.defender)
		if got != testCase.want {
			t.Errorf("Scale(%d, %s, %s) = %d, want %d",
				testCase.damage, testCase.attacker, testCase.defender, got, testCase.want)
		}
	}
}

func TestUnknownElementValueFallsBackToNeutral(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rogue := element.Element(element.Count + 7)
	if rogue.Valid() {
		t.Fatalf("%v should not report itself as valid", rogue)
	}
	if got, want := chart.Multiplier(rogue, element.Water), chart.Multipliers().Neutral; got != want {
		t.Errorf("multiplier for an out-of-range element is %d, want the neutral %d", got, want)
	}
	if got := chart.Strengths(rogue); got != nil {
		t.Errorf("strengths for an out-of-range element is %v, want nil", got)
	}
}

func TestParseAndStringAgreeForEveryElement(t *testing.T) {
	for _, member := range element.All() {
		parsed, err := element.Parse(member.String())
		if err != nil {
			t.Errorf("Parse(%q): %v", member.String(), err)
			continue
		}
		if parsed != member {
			t.Errorf("Parse(%q) = %s, want %s", member.String(), parsed, member)
		}
	}
	if _, err := element.Parse("plasma"); err == nil {
		t.Error("want an error for an unknown name, got none")
	}
}

func mustChart(t *testing.T) *element.Chart {
	t.Helper()
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse fixture chart: %v", err)
	}
	return chart
}

func TestSingleAndDualConstructors(t *testing.T) {
	single, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("Single(fire): %v", err)
	}
	if single.IsDual() {
		t.Error("a single affinity reports itself as dual")
	}
	if got, want := single.String(), "fire"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if _, dual := single.Secondary(); dual {
		t.Error("a single affinity reports a secondary element")
	}

	pair, err := element.Dual(element.Fire, element.Wind)
	if err != nil {
		t.Fatalf("Dual(fire, wind): %v", err)
	}
	if !pair.IsDual() {
		t.Error("a dual affinity does not report itself as dual")
	}
	if got, want := pair.String(), "fire/wind"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if !pair.Has(element.Wind) || !pair.Has(element.Fire) || pair.Has(element.Water) {
		t.Errorf("Has is wrong for %s", pair)
	}
	if got := pair.Elements(); len(got) != 2 || got[0] != element.Fire || got[1] != element.Wind {
		t.Errorf("Elements() = %v, want [fire wind]", got)
	}

	// The zero value has to be usable, because a unit with no declared element
	// is the common case.
	var zero element.Affinity
	if zero.IsDual() || zero.Primary() != element.Neutral {
		t.Errorf("the zero affinity is %s, want a single neutral", zero)
	}
}

func TestDualConstructorRejects(t *testing.T) {
	cases := []struct {
		name               string
		primary, secondary element.Element
		wantErr            string
	}{
		{"the same element twice", element.Fire, element.Fire, "listed twice"},
		{"neutral as the second element", element.Fire, element.Neutral, "pairs with neutral"},
		{"neutral as the first element", element.Neutral, element.Fire, "pairs with neutral"},
		{"an undeclared element", element.Fire, element.Element(99), "undeclared element"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := element.Dual(testCase.primary, testCase.secondary)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
	if _, err := element.Single(element.Element(99)); err == nil {
		t.Error("Single accepted an undeclared element")
	}
}

func TestAffinityJSONRoundTrip(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`"fire"`, "fire"},
		{`"neutral"`, "neutral"},
		{`["fire"]`, "fire"},
		{`["fire","wind"]`, "fire/wind"},
	}
	for _, testCase := range cases {
		var affinity element.Affinity
		if err := json.Unmarshal([]byte(testCase.raw), &affinity); err != nil {
			t.Errorf("unmarshal %s: %v", testCase.raw, err)
			continue
		}
		if got := affinity.String(); got != testCase.want {
			t.Errorf("unmarshal %s gave %s, want %s", testCase.raw, got, testCase.want)
			continue
		}
		encoded, err := json.Marshal(affinity)
		if err != nil {
			t.Errorf("marshal %s: %v", affinity, err)
			continue
		}
		var again element.Affinity
		if err := json.Unmarshal(encoded, &again); err != nil {
			t.Errorf("re-unmarshal %s: %v", encoded, err)
			continue
		}
		if again != affinity {
			t.Errorf("round trip of %s produced %s", affinity, again)
		}
	}
}

func TestAffinityJSONRejects(t *testing.T) {
	cases := []struct {
		raw     string
		wantErr string
	}{
		{`"plasma"`, "unknown element"},
		{`[]`, "want 1 or 2"},
		{`["fire","wind","water"]`, "want 1 or 2"},
		{`["fire","fire"]`, "listed twice"},
		{`["fire","neutral"]`, "pairs with neutral"},
		{`["fire","plasma"]`, "unknown element"},
		{`17`, "must be an element name or an array"},
		{`{"primary":"fire"}`, "must be an element name or an array"},
	}
	for _, testCase := range cases {
		var affinity element.Affinity
		err := json.Unmarshal([]byte(testCase.raw), &affinity)
		if err == nil {
			t.Errorf("%s: want an error mentioning %q, got none", testCase.raw, testCase.wantErr)
			continue
		}
		if !strings.Contains(err.Error(), testCase.wantErr) {
			t.Errorf("%s: error %q does not mention %q", testCase.raw, err, testCase.wantErr)
		}
	}
}

func TestValidateAffinityRejectsPairsThatCounterEachOther(t *testing.T) {
	chart := mustChart(t)
	// The fixture cycle is water > fire > grass > water, so every pair drawn
	// from it has a relation and none of them may share a unit.
	related := [][2]element.Element{
		{element.Water, element.Fire},
		{element.Fire, element.Grass},
		{element.Grass, element.Water},
		{element.Light, element.Dark},
	}
	for _, pair := range related {
		affinity, err := element.Dual(pair[0], pair[1])
		if err != nil {
			t.Fatalf("Dual(%s, %s): %v", pair[0], pair[1], err)
		}
		if err := chart.ValidateAffinity(affinity); err == nil {
			t.Errorf("%s was accepted, want a rejection", affinity)
		}
		if !chart.Related(pair[0], pair[1]) || !chart.Related(pair[1], pair[0]) {
			t.Errorf("Related is false for %s and %s", pair[0], pair[1])
		}
	}

	// An inert element has no relations, so it pairs with anything.
	affinity, err := element.Dual(element.Water, element.Ice)
	if err != nil {
		t.Fatalf("Dual(water, ice): %v", err)
	}
	if err := chart.ValidateAffinity(affinity); err != nil {
		t.Errorf("%s was rejected: %v", affinity, err)
	}
	for _, single := range element.All() {
		affinity, err := element.Single(single)
		if err != nil {
			t.Fatalf("Single(%s): %v", single, err)
		}
		if err := chart.ValidateAffinity(affinity); err != nil {
			t.Errorf("single affinity %s was rejected: %v", affinity, err)
		}
	}
}

func TestLegalPairsAreSymmetricAndUnordered(t *testing.T) {
	chart := mustChart(t)
	seen := make(map[string]bool)
	for _, pair := range chart.LegalPairs() {
		if !pair.IsDual() {
			t.Errorf("LegalPairs returned the single affinity %s", pair)
		}
		if err := chart.ValidateAffinity(pair); err != nil {
			t.Errorf("LegalPairs returned %s, which does not validate: %v", pair, err)
		}
		secondary, _ := pair.Secondary()
		if pair.Primary() >= secondary {
			t.Errorf("%s is not in declaration order, so the reverse could be listed too", pair)
		}
		if seen[pair.String()] {
			t.Errorf("%s is listed twice", pair)
		}
		seen[pair.String()] = true
		if pair.Has(element.Neutral) {
			t.Errorf("%s includes neutral", pair)
		}
	}
}

// TestMultiplierAgainstComposesTheTwoLookups guards the stacking arithmetic:
// a dual defender's multiplier is the product of its two, scaled back down by
// the neutral base.
func TestMultiplierAgainstComposesTheTwoLookups(t *testing.T) {
	chart := mustChart(t)
	base := chart.Multipliers().Neutral
	for _, attacker := range element.All() {
		for _, primary := range element.All() {
			single, err := element.Single(primary)
			if err != nil {
				t.Fatalf("Single(%s): %v", primary, err)
			}
			if got, want := chart.MultiplierAgainst(attacker, single), chart.Multiplier(attacker, primary); got != want {
				t.Errorf("%s against %s: %d, want the single lookup %d", attacker, single, got, want)
			}
			for _, secondary := range element.All() {
				pair, err := element.Dual(primary, secondary)
				if err != nil {
					continue
				}
				want := chart.Multiplier(attacker, primary) * chart.Multiplier(attacker, secondary) / base
				if got := chart.MultiplierAgainst(attacker, pair); got != want {
					t.Errorf("%s against %s: %d, want %d", attacker, pair, got, want)
				}
			}
		}
	}
}

func TestWeaknessAndResistanceCancel(t *testing.T) {
	chart := mustChart(t)
	// Water beats fire and grass beats water, so a grass attack is strong
	// against the water half and weak against the fire half.
	pair, err := element.Dual(element.Water, element.Fire)
	if err != nil {
		t.Fatalf("Dual(water, fire): %v", err)
	}
	if got, want := chart.MultiplierAgainst(element.Grass, pair), chart.Multipliers().Neutral; got != want {
		t.Errorf("grass against %s: %d, want the neutral %d", pair, got, want)
	}
	if got, want := chart.ScaleAgainst(1000, element.Grass, pair), int64(1000); got != want {
		t.Errorf("ScaleAgainst gave %d, want %d", got, want)
	}
}

// TestTheDeclaredRingsAgreeWithTheResolvedEdges is the anti-drift test the
// retained declaration exists behind.
//
// Cycles and MutualPairs hand back what the author wrote; Strengths answers from
// the matrix those were walked into. Two readings of one thing drift, and here
// the drift is silent in both directions: a ring kept but not walked would draw
// an edge the game does not play, and an edge walked but not kept would be
// missing from every picture of the chart.
func TestTheDeclaredRingsAgreeWithTheResolvedEdges(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	declared := map[[2]element.Element]bool{}
	cycles := chart.Cycles()
	if len(cycles) == 0 {
		t.Fatal("the fixture declares no cycle, so this measures nothing")
	}
	for _, cycle := range cycles {
		for index, attacker := range cycle.Chain {
			declared[[2]element.Element{attacker, cycle.Chain[(index+1)%len(cycle.Chain)]}] = true
		}
	}
	pairs := chart.MutualPairs()
	if len(pairs) == 0 {
		t.Fatal("the fixture declares no mutual pair, so half of this measures nothing")
	}
	for _, pair := range pairs {
		declared[[2]element.Element{pair[0], pair[1]}] = true
		declared[[2]element.Element{pair[1], pair[0]}] = true
	}

	resolved := map[[2]element.Element]bool{}
	for _, attacker := range element.All() {
		for _, defender := range chart.Strengths(attacker) {
			resolved[[2]element.Element{attacker, defender}] = true
		}
	}
	for edge := range declared {
		if !resolved[edge] {
			t.Errorf("%v > %v is declared and not resolved", edge[0], edge[1])
		}
	}
	for edge := range resolved {
		if !declared[edge] {
			t.Errorf("%v > %v is resolved and in no declaration", edge[0], edge[1])
		}
	}
}

// TestTheDeclarationIsHandedOutAsCopies is what stops a reference screen editing
// the chart the game is playing.
//
// The chart is loaded once and shared, so a caller holding the live slices could
// sort a ring for display and reorder the ring every later reader sees.
func TestTheDeclarationIsHandedOutAsCopies(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	first := chart.Cycles()
	if len(first) == 0 || len(first[0].Chain) < 2 {
		t.Fatal("the fixture has no ring to scribble on")
	}
	opened := first[0].Chain[0]
	first[0].Name = "scribbled"
	first[0].Chain[0], first[0].Chain[1] = first[0].Chain[1], first[0].Chain[0]
	pairs := chart.MutualPairs()
	if len(pairs) == 0 {
		t.Fatal("the fixture has no pair to scribble on")
	}
	scribbled := [2]element.Element{element.Fire, element.Fire}
	pairs[0] = scribbled

	again := chart.Cycles()
	if again[0].Name == "scribbled" {
		t.Error("a caller renamed a ring through the chart's own slice")
	}
	if again[0].Chain[0] != opened {
		t.Error("a caller reordered a ring through the chart's own slice")
	}
	if chart.MutualPairs()[0] == scribbled {
		t.Error("a caller rewrote a pair through the chart's own slice")
	}
}

// TestInertIsReadFromTheEdgesRatherThanTheList is the difference between what a
// chart says about itself and what it does.
//
// The "inert" list in the file is a declaration; being inert in play is having no
// edges. The two cannot disagree the other way round — addEdge and Validate
// refuse an element that is listed inert and given edges — so what is left to
// hold is that every element the chart gives no edges is reported as one.
func TestInertIsReadFromTheEdgesRatherThanTheList(t *testing.T) {
	chart, err := element.ParseChart(valid().encode(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	reported := map[element.Element]bool{}
	for _, member := range chart.Inert() {
		reported[member] = true
	}
	if len(reported) == 0 {
		t.Fatal("the fixture leaves no element edgeless, so this measures nothing")
	}
	for _, member := range element.All() {
		edgeless := len(chart.Strengths(member)) == 0 && len(chart.Weaknesses(member)) == 0
		if edgeless != reported[member] {
			t.Errorf("%v: edgeless=%v but Inert reports %v", member, edgeless, reported[member])
		}
	}
}
