package progression_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/progression"
)

func rules() combat.Rules {
	return combat.Rules{DefenseConstant: 300, MinimumDamage: 1, MinHitChance: 150}
}

func limits() progression.Limits {
	return progression.Limits{
		LevelCap: progression.LevelCap,
		Ceilings: progression.Values{
			progression.HP: 4800, progression.Attack: 800, progression.Defense: 800,
			progression.Speed: 200, progression.Accuracy: 300, progression.Dodge: 150,
		},
		MaxEffectiveHP: 11500,
	}
}

// table builds a stat table from the endpoints of each curve. Accuracy and
// dodge use fixed modest curves, since none of these cases turn on them.
func table(hpBase, hpMax, atkBase, atkMax, defBase, defMax, spdBase, spdMax int64) progression.Table {
	var out progression.Table
	out[progression.HP] = progression.Curve{Base: hpBase, Max: hpMax}
	out[progression.Attack] = progression.Curve{Base: atkBase, Max: atkMax}
	out[progression.Defense] = progression.Curve{Base: defBase, Max: defMax}
	out[progression.Speed] = progression.Curve{Base: spdBase, Max: spdMax}
	out[progression.Accuracy] = progression.Curve{Base: 20, Max: 120}
	out[progression.Dodge] = progression.Curve{Base: 10, Max: 60}
	return out
}

func TestCurveEndpointsAreExact(t *testing.T) {
	curve := progression.Curve{Base: 137, Max: 4800}
	if got := curve.At(1); got != 137 {
		t.Errorf("level 1 gives %d, want the base 137", got)
	}
	if got := curve.At(progression.LevelCap); got != 4800 {
		t.Errorf("level %d gives %d, want the max 4800", progression.LevelCap, got)
	}
	// Interpolation is a single integer division, so the midpoint is the exact
	// truncation of the real value rather than an accumulated per-level step.
	if got, want := curve.At(30), int64(137+(4800-137)*29/59); got != want {
		t.Errorf("level 30 gives %d, want %d", got, want)
	}
}

func TestCurveClampsOutsideTheLevelRange(t *testing.T) {
	curve := progression.Curve{Base: 100, Max: 500}
	for _, level := range []int{-40, 0, 1} {
		if got := curve.At(level); got != 100 {
			t.Errorf("level %d gives %d, want the base 100", level, got)
		}
	}
	for _, level := range []int{progression.LevelCap, progression.LevelCap + 1, 9999} {
		if got := curve.At(level); got != 500 {
			t.Errorf("level %d gives %d, want the max 500", level, got)
		}
	}
}

func TestCurveNeverDecreases(t *testing.T) {
	curve := progression.Curve{Base: 41, Max: 797}
	previous := int64(0)
	for level := 1; level <= progression.LevelCap; level++ {
		got := curve.At(level)
		if got < previous {
			t.Fatalf("stat fell from %d to %d at level %d", previous, got, level)
		}
		previous = got
	}
}

func TestFlatCurveStaysFlat(t *testing.T) {
	curve := progression.Curve{Base: 90, Max: 90}
	for level := 1; level <= progression.LevelCap; level++ {
		if got := curve.At(level); got != 90 {
			t.Fatalf("a flat curve gave %d at level %d", got, level)
		}
	}
}

func TestCurveValidateRejects(t *testing.T) {
	cases := []struct {
		name    string
		curve   progression.Curve
		wantErr string
	}{
		{"a base of zero", progression.Curve{Base: 0, Max: 100}, "want a positive value"},
		{"a negative base", progression.Curve{Base: -10, Max: 100}, "want a positive value"},
		{"a max below the base", progression.Curve{Base: 200, Max: 100}, "may not shrink"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.curve.Validate(progression.HP)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestValuesJSONRequiresEveryStat(t *testing.T) {
	var values progression.Values
	if err := json.Unmarshal([]byte(`{"hp":100,"attack":50,"defense":40,"speed":90,"accuracy":30,"dodge":15}`), &values); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if values[progression.HP] != 100 || values[progression.Speed] != 90 {
		t.Errorf("decoded %s, want hp 100 and spd 90", values)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var again progression.Values
	if err := json.Unmarshal(encoded, &again); err != nil {
		t.Fatalf("re-unmarshal %s: %v", encoded, err)
	}
	if again != values {
		t.Errorf("round trip of %s gave %s", values, again)
	}

	for _, raw := range []string{
		`{"attack":50,"defense":40,"speed":90,"accuracy":30,"dodge":15}`,
		`{"hp":100,"defense":40,"speed":90,"accuracy":30,"dodge":15}`,
		`{"hp":100,"attack":50,"speed":90,"accuracy":30,"dodge":15}`,
		`{"hp":100,"attack":50,"defense":40,"accuracy":30,"dodge":15}`,
		`{"hp":100,"attack":50,"defense":40,"speed":90,"dodge":15}`,
		`{"hp":100,"attack":50,"defense":40,"speed":90,"accuracy":30}`,
	} {
		var missing progression.Values
		err := json.Unmarshal([]byte(raw), &missing)
		if err == nil {
			t.Errorf("%s was accepted, want a rejection for the missing stat", raw)
			continue
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("%s gave %q, want a message about a missing stat", raw, err)
		}
	}
	if err := json.Unmarshal([]byte(`"fast"`), &progression.Values{}); err == nil {
		t.Error("a bare string was accepted as stat values")
	}
}

func TestTableJSONRequiresEveryCurve(t *testing.T) {
	raw := `{
	  "hp":      {"base": 320, "max": 4800},
	  "attack":  {"base":  40, "max":  500},
	  "defense": {"base":  30, "max":  400},
	  "speed":   {"base":  70, "max":  110},
	  "accuracy":{"base":  20, "max":  120},
	  "dodge":   {"base":  10, "max":   60}
	}`
	var decoded progression.Table
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := decoded.At(progression.LevelCap); got[progression.HP] != 4800 {
		t.Errorf("resolved %s at the cap, want hp 4800", got)
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var again progression.Table
	if err := json.Unmarshal(encoded, &again); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if again != decoded {
		t.Error("table round trip lost data")
	}
	var missing progression.Table
	if err := json.Unmarshal([]byte(`{"hp":{"base":1,"max":2}}`), &missing); err == nil {
		t.Error("a table missing three curves was accepted")
	}
}

func TestEffectiveHPCombinesHealthAndDefence(t *testing.T) {
	r := rules()
	cases := []struct {
		hp, defense, want int64
	}{
		// With no defence, effective health is just health.
		{4800, 0, 4800},
		// Defence equal to K doubles it.
		{4800, 300, 9600},
		// The ceilings together are the case the budget exists to forbid.
		{4800, 800, 17647},
		{3100, 800, 11397},
		{4800, 400, 11214},
		{2600, 320, 5383},
	}
	for _, testCase := range cases {
		values := progression.Values{progression.HP: testCase.hp, progression.Defense: testCase.defense}
		if got := progression.EffectiveHP(values, r); got != testCase.want {
			t.Errorf("%d health behind %d defence absorbs %d, want %d",
				testCase.hp, testCase.defense, got, testCase.want)
		}
	}
}

func TestCheckValuesRejects(t *testing.T) {
	l, r := limits(), rules()
	cases := []struct {
		name    string
		values  progression.Values
		wantErr string
	}{
		{"health over the ceiling",
			progression.Values{progression.HP: 5000, progression.Attack: 100, progression.Defense: 100, progression.Speed: 100},
			"hp is 5000, over the ceiling"},
		{"attack over the ceiling",
			progression.Values{progression.HP: 100, progression.Attack: 900, progression.Defense: 100, progression.Speed: 100},
			"attack is 900, over the ceiling"},
		{"speed over the ceiling",
			progression.Values{progression.HP: 100, progression.Attack: 100, progression.Defense: 100, progression.Speed: 400},
			"speed is 400, over the ceiling"},
		{"accuracy over the ceiling",
			progression.Values{progression.HP: 100, progression.Attack: 100, progression.Defense: 100, progression.Speed: 100, progression.Accuracy: 500},
			"accuracy is 500, over the ceiling"},
		{"both durability stats at their ceiling",
			progression.Values{progression.HP: 4800, progression.Attack: 100, progression.Defense: 800, progression.Speed: 100},
			"over the budget"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := l.CheckValues(testCase.values, r)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}

	// A unit may sit at the health ceiling as long as its defence pays for it.
	withinBudget := progression.Values{progression.HP: 4800, progression.Attack: 500, progression.Defense: 400, progression.Speed: 90}
	if err := l.CheckValues(withinBudget, r); err != nil {
		t.Errorf("%s was rejected: %v", withinBudget, err)
	}
	// And it may sit at the defence ceiling if it gives up health.
	alsoWithin := progression.Values{progression.HP: 3100, progression.Attack: 500, progression.Defense: 800, progression.Speed: 90}
	if err := l.CheckValues(alsoWithin, r); err != nil {
		t.Errorf("%s was rejected: %v", alsoWithin, err)
	}
}

func TestCheckTableNamesTheOffendingLevel(t *testing.T) {
	l, r := limits(), rules()
	good := table(320, 4800, 40, 500, 30, 400, 70, 110)
	if err := l.CheckTable(good, r); err != nil {
		t.Fatalf("a table inside the budget was rejected: %v", err)
	}
	over := table(320, 4800, 40, 500, 30, 800, 70, 110)
	err := l.CheckTable(over, r)
	if err == nil {
		t.Fatal("a table that ends over the budget was accepted")
	}
	if !strings.Contains(err.Error(), "at level") || !strings.Contains(err.Error(), "over the budget") {
		t.Errorf("error %q should name the level and the budget", err)
	}
	broken := table(0, 4800, 40, 500, 30, 400, 70, 110)
	if err := l.CheckTable(broken, r); err == nil {
		t.Fatal("a table with an invalid curve was accepted")
	}
}

func TestLimitsValidateRejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*progression.Limits)
		wantErr string
	}{
		{"a level cap the engine is not built for", func(l *progression.Limits) {
			l.LevelCap = 99
		}, "the engine is built for"},
		{"a ceiling of zero", func(l *progression.Limits) {
			l.Ceilings[progression.Attack] = 0
		}, "the attack ceiling is 0"},
		{"no effective health budget", func(l *progression.Limits) {
			l.MaxEffectiveHP = 0
		}, "max_effective_hp is 0"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			l := limits()
			testCase.mutate(&l)
			err := l.Validate()
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
	if _, err := progression.ParseLimits([]byte("{")); err == nil {
		t.Error("malformed JSON was accepted")
	}
}

func evolutionLine() progression.Line {
	return progression.Line{
		{Name: "seedling", MinLevel: 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
		{Name: "sapling", MinLevel: 20, Stats: table(320, 3000, 40, 340, 30, 260, 70, 100)},
		{Name: "elder", MinLevel: 45, Stats: table(320, 4800, 40, 500, 30, 400, 70, 110)},
	}
}

func TestLineResolvePicksTheReachedStage(t *testing.T) {
	line := evolutionLine()
	cases := []struct {
		level int
		stage string
	}{
		{1, "seedling"}, {19, "seedling"},
		{20, "sapling"}, {44, "sapling"},
		{45, "elder"}, {progression.LevelCap, "elder"},
	}
	for _, testCase := range cases {
		values, stage, err := line.Resolve(testCase.level)
		if err != nil {
			t.Errorf("level %d: %v", testCase.level, err)
			continue
		}
		if stage.Name != testCase.stage {
			t.Errorf("level %d resolved to %q, want %q", testCase.level, stage.Name, testCase.stage)
		}
		if values[progression.HP] <= 0 {
			t.Errorf("level %d resolved to %s", testCase.level, values)
		}
	}
	// Only the final stage reaches the health ceiling, which is the point of
	// evolving.
	atCap, _, err := line.Resolve(progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve at the cap: %v", err)
	}
	if got, want := atCap[progression.HP], int64(4800); got != want {
		t.Errorf("health at the cap is %d, want %d", got, want)
	}
}

func TestLineRejectsLevelsOutsideTheRange(t *testing.T) {
	line := evolutionLine()
	for _, level := range []int{0, -1, progression.LevelCap + 1} {
		if _, _, err := line.Resolve(level); err == nil {
			t.Errorf("level %d was accepted", level)
		}
	}
}

func TestLineValidateRejects(t *testing.T) {
	l, r := limits(), rules()
	cases := []struct {
		name    string
		line    progression.Line
		wantErr string
	}{
		{"an empty line", progression.Line{}, "at least one stage"},
		{"a first stage that does not start at level 1", progression.Line{
			{Name: "late", MinLevel: 5, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
		}, "want 1"},
		{"stages that do not advance", progression.Line{
			{Name: "one", MinLevel: 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
			{Name: "two", MinLevel: 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
		}, "not after the previous"},
		{"a stage past the level cap", progression.Line{
			{Name: "one", MinLevel: 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
			{Name: "two", MinLevel: progression.LevelCap + 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
		}, "past the cap"},
		{"a stage with no name", progression.Line{
			{MinLevel: 1, Stats: table(320, 1600, 40, 200, 30, 150, 70, 90)},
		}, "no name"},
		{"a stage over the stat budget", progression.Line{
			{Name: "one", MinLevel: 1, Stats: table(320, 4800, 40, 200, 30, 800, 70, 90)},
		}, "over the budget"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.line.Validate(l, r)
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
	if err := evolutionLine().Validate(l, r); err != nil {
		t.Errorf("the reference line was rejected: %v", err)
	}
}

func TestKindNames(t *testing.T) {
	want := []string{"hp", "attack", "defense", "speed", "accuracy", "dodge"}
	kinds := progression.Kinds()
	if len(kinds) != len(want) {
		t.Fatalf("there are %d stats, want %d", len(kinds), len(want))
	}
	for i, kind := range kinds {
		if kind.String() != want[i] {
			t.Errorf("stat %d is %q, want %q", i, kind, want[i])
		}
	}
	if got := progression.Kind(200).String(); !strings.Contains(got, "200") {
		t.Errorf("an unknown stat renders as %q", got)
	}
	var values progression.Values
	if got := values.Get(progression.Kind(200)); got != 0 {
		t.Errorf("Get for an unknown stat gave %d, want 0", got)
	}
}
