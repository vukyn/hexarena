package seed_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// TestShippedElementChartMatchesTheDesign freezes the affinity table the roster
// is balanced around. The chart itself is data, so without this the shipped
// relations could drift from the design with every test still passing.
func TestShippedElementChartMatchesTheDesign(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	design := []struct {
		member     element.Element
		strengths  []element.Element
		weaknesses []element.Element
	}{
		{element.Water, []element.Element{element.Fire, element.Metal}, []element.Element{element.Ground, element.Electric}},
		{element.Fire, []element.Element{element.Grass, element.Ice}, []element.Element{element.Water, element.Wind}},
		{element.Grass, []element.Element{element.Ground, element.Wind}, []element.Element{element.Fire, element.Metal}},
		{element.Ground, []element.Element{element.Water, element.Electric}, []element.Element{element.Grass, element.Ice}},
		{element.Ice, []element.Element{element.Ground, element.Metal}, []element.Element{element.Fire, element.Electric}},
		{element.Metal, []element.Element{element.Grass, element.Wind}, []element.Element{element.Water, element.Ice}},
		{element.Wind, []element.Element{element.Fire, element.Electric}, []element.Element{element.Grass, element.Metal}},
		{element.Electric, []element.Element{element.Water, element.Ice}, []element.Element{element.Ground, element.Wind}},
		{element.Light, []element.Element{element.Dark}, []element.Element{element.Dark}},
		{element.Dark, []element.Element{element.Light}, []element.Element{element.Light}},
		{element.Neutral, nil, nil},
	}
	if len(design) != element.Count {
		t.Fatalf("the design table covers %d elements, the package declares %d", len(design), element.Count)
	}
	for _, row := range design {
		if got := chart.Strengths(row.member); !sameElements(got, row.strengths) {
			t.Errorf("%s is strong against %v, design says %v", row.member, got, row.strengths)
		}
		if got := chart.Weaknesses(row.member); !sameElements(got, row.weaknesses) {
			t.Errorf("%s is weak to %v, design says %v", row.member, got, row.weaknesses)
		}
	}
}

// TestEveryCycledElementHasTwoOfEach is the balance property the two-cycle plus
// cross-cycle layout exists to produce: no cycled element carries more or fewer
// matchups than any other.
func TestEveryCycledElementHasTwoOfEach(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	cycled := []element.Element{
		element.Water, element.Fire, element.Grass, element.Ground,
		element.Ice, element.Metal, element.Wind, element.Electric,
	}
	for _, member := range cycled {
		if got := len(chart.Strengths(member)); got != 2 {
			t.Errorf("%s has %d strengths, want 2", member, got)
		}
		if got := len(chart.Weaknesses(member)); got != 2 {
			t.Errorf("%s has %d weaknesses, want 2", member, got)
		}
	}
}

func TestShippedMultipliers(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	got := chart.Multipliers()
	want := element.Multipliers{Advantage: 1500, Neutral: 1000, Disadvantage: 667}
	if got != want {
		t.Errorf("multipliers are %+v, want %+v", got, want)
	}
}

func TestElementChartGolden(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	got := chartReport(chart)
	path := filepath.Join("testdata", "elements.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("element report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

func chartReport(chart *element.Chart) string {
	var b strings.Builder
	multipliers := chart.Multipliers()
	fmt.Fprintf(&b, "== multipliers, parts per thousand ==\nadvantage %d  neutral %d  disadvantage %d\n\n",
		multipliers.Advantage, multipliers.Neutral, multipliers.Disadvantage)

	b.WriteString("== relations ==\n")
	for _, member := range element.All() {
		fmt.Fprintf(&b, "%-9s beats %-20s weak to %s\n",
			member, joinElements(chart.Strengths(member)), joinElements(chart.Weaknesses(member)))
	}

	b.WriteString("\n== multiplier matrix, attacker down, defender across ==\n")
	b.WriteString("         ")
	for _, defender := range element.All() {
		fmt.Fprintf(&b, "%9s", defender)
	}
	b.WriteString("\n")
	for _, attacker := range element.All() {
		fmt.Fprintf(&b, "%-9s", attacker)
		for _, defender := range element.All() {
			fmt.Fprintf(&b, "%9d", chart.Multiplier(attacker, defender))
		}
		b.WriteString("\n")
	}

	// A base that is not a multiple of the neutral multiplier, so the integer
	// truncation in Scale is visible in the snapshot.
	const sampleDamage = 250
	fmt.Fprintf(&b, "\n== %d damage scaled ==\n", sampleDamage)
	b.WriteString("         ")
	for _, defender := range element.All() {
		fmt.Fprintf(&b, "%9s", defender)
	}
	b.WriteString("\n")
	for _, attacker := range element.All() {
		fmt.Fprintf(&b, "%-9s", attacker)
		for _, defender := range element.All() {
			fmt.Fprintf(&b, "%9d", chart.Scale(sampleDamage, attacker, defender))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func joinElements(members []element.Element) string {
	if len(members) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(members))
	for _, member := range members {
		parts = append(parts, member.String())
	}
	return strings.Join(parts, ", ")
}

func sameElements(got, want []element.Element) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestShippedLegalPairs freezes which dual combinations the chart allows. The
// pairing rule is what keeps a unit from being its own counter, and the exact
// list is what the roster gets designed against.
func TestShippedLegalPairs(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	cycledPairs := []string{
		"water/grass", "water/ice", "water/wind",
		"fire/ground", "fire/metal", "fire/electric",
		"grass/ice", "grass/electric",
		"ground/metal", "ground/wind",
		"ice/wind",
		"metal/electric",
	}
	got := make(map[string]bool)
	for _, pair := range chart.LegalPairs() {
		got[pair.String()] = true
	}
	// Twelve pairs among the eight cycled elements, plus light and dark each
	// pairing with all eight; light with dark is a mutual relation and so is
	// excluded, and neutral never appears in a dual at all.
	if want := len(cycledPairs) + 16; len(got) != want {
		t.Errorf("chart allows %d dual combinations, want %d", len(got), want)
	}
	for _, pair := range cycledPairs {
		if !got[pair] {
			t.Errorf("%s should be a legal pair but is not allowed", pair)
		}
	}
	for _, forbidden := range []string{
		"water/fire", "water/ground", "water/electric", "water/metal",
		"light/dark", "fire/ice", "grass/wind", "ice/metal",
	} {
		if got[forbidden] {
			t.Errorf("%s counters itself and should not be a legal pair", forbidden)
		}
	}
	if got["light/water"] || !got["water/light"] {
		t.Error("pairs should be listed in element declaration order")
	}
}

// TestShippedDualStacking freezes where the extremes of the affinity range
// land. These are the matchups the difficulty of an encounter is tuned with,
// so they must not move silently.
func TestShippedDualStacking(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	multipliers := chart.Multipliers()
	stacked := multipliers.Advantage * multipliers.Advantage / multipliers.Neutral
	resisted := multipliers.Disadvantage * multipliers.Disadvantage / multipliers.Neutral
	if stacked != 2250 || resisted != 444 {
		t.Fatalf("stacked multipliers are %d and %d, want 2250 and 444", stacked, resisted)
	}

	cases := []struct {
		primary, secondary element.Element
		// doubleWeak and doubleResist are Neutral when the pair has none.
		doubleWeak, doubleResist element.Element
	}{
		{element.Water, element.Grass, element.Neutral, element.Neutral},
		{element.Water, element.Ice, element.Electric, element.Metal},
		{element.Water, element.Wind, element.Neutral, element.Fire},
		{element.Fire, element.Ground, element.Neutral, element.Neutral},
		{element.Fire, element.Metal, element.Water, element.Grass},
		{element.Fire, element.Electric, element.Wind, element.Ice},
		{element.Grass, element.Ice, element.Fire, element.Ground},
		{element.Grass, element.Electric, element.Neutral, element.Neutral},
		{element.Ground, element.Metal, element.Ice, element.Neutral},
		{element.Ground, element.Wind, element.Grass, element.Electric},
		{element.Ice, element.Wind, element.Neutral, element.Neutral},
		{element.Metal, element.Electric, element.Neutral, element.Neutral},
	}
	for _, testCase := range cases {
		pair, err := element.Dual(testCase.primary, testCase.secondary)
		if err != nil {
			t.Fatalf("Dual(%s, %s): %v", testCase.primary, testCase.secondary, err)
		}
		t.Run(pair.String(), func(t *testing.T) {
			if err := chart.ValidateAffinity(pair); err != nil {
				t.Fatalf("%s is not a legal pair: %v", pair, err)
			}
			for _, attacker := range element.All() {
				got := chart.MultiplierAgainst(attacker, pair)
				switch {
				case attacker == testCase.doubleWeak && attacker != element.Neutral:
					if got != stacked {
						t.Errorf("%s against %s: %d, want the stacked weakness %d", attacker, pair, got, stacked)
					}
				case attacker == testCase.doubleResist && attacker != element.Neutral:
					if got != resisted {
						t.Errorf("%s against %s: %d, want the stacked resistance %d", attacker, pair, got, resisted)
					}
				default:
					if got == stacked {
						t.Errorf("%s against %s stacks to %d, the design lists no double weakness there", attacker, pair, got)
					}
					if got == resisted {
						t.Errorf("%s against %s stacks to %d, the design lists no double resistance there", attacker, pair, got)
					}
				}
			}
		})
	}
}

func TestShippedCombatRules(t *testing.T) {
	rules, err := seed.CombatRules()
	if err != nil {
		t.Fatalf("load shipped combat rules: %v", err)
	}
	if got, want := rules.DefenseConstant, int64(300); got != want {
		t.Errorf("defense_constant is %d, want %d", got, want)
	}
	if got, want := rules.MinimumDamage, int64(1); got != want {
		t.Errorf("minimum_damage is %d, want %d", got, want)
	}
	if got, want := rules.MinHitChance, 150; got != want {
		t.Errorf("min_hit_chance is %d, want %d", got, want)
	}
	// At the level 60 ceiling of 800 defence a hit keeps roughly 27% of its
	// damage, which is what the stat budget assumes.
	if got, want := rules.DefenseReduction(800), 272; got != want {
		t.Errorf("damage getting through 800 defence is %d per thousand, want %d", got, want)
	}
}

// TestPermilleBasesAgree keeps the two packages on the same denominator. The
// element chart hands its multiplier straight to the damage formula, so a
// mismatch would scale every hit by a factor of a thousand.
func TestPermilleBasesAgree(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	if got, want := chart.Multipliers().Neutral, combat.PermilleBase; got != want {
		t.Errorf("the chart's neutral multiplier is %d, the damage formula's base is %d", got, want)
	}
}

func TestCombatGolden(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	rules, err := seed.CombatRules()
	if err != nil {
		t.Fatalf("load shipped combat rules: %v", err)
	}
	got := combatReport(chart, rules)
	path := filepath.Join("testdata", "combat.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("combat report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

func combatReport(chart *element.Chart, rules combat.Rules) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== damage formula ==\nattack * skill * affinity * K / (K + defence),  K = %d, floor %d\n",
		rules.DefenseConstant, rules.MinimumDamage)
	fmt.Fprintf(&b, "multipliers are parts per thousand, base %d\n\n", combat.PermilleBase)

	b.WriteString("== defence curve, share of damage getting through ==\n")
	b.WriteString("defence  per mille\n")
	for defense := int64(0); defense <= 1200; defense += 100 {
		fmt.Fprintf(&b, "%7d  %9d\n", defense, rules.DefenseReduction(defense))
	}

	b.WriteString("\n== damage at the level 60 ceiling, attack 800, skill 1800 ==\n")
	b.WriteString("affinity           ")
	for defense := int64(0); defense <= 800; defense += 200 {
		fmt.Fprintf(&b, "%7d", defense)
	}
	b.WriteString("   <- defence\n")
	for _, row := range []struct {
		name       string
		multiplier int
	}{
		{"both resist  444", 444},
		{"resisted     667", 667},
		{"neutral     1000", combat.PermilleBase},
		{"weak        1500", 1500},
		{"both weak   2250", 2250},
	} {
		fmt.Fprintf(&b, "%-19s", row.name)
		for defense := int64(0); defense <= 800; defense += 200 {
			fmt.Fprintf(&b, "%7d", rules.Damage(800, defense, 1800, row.multiplier))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n== dual defence, attacker across, multiplier per mille ==\n")
	pairs := chart.LegalPairs()
	b.WriteString("                 ")
	for _, attacker := range element.All() {
		fmt.Fprintf(&b, "%9s", attacker)
	}
	b.WriteString("\n")
	for _, pair := range pairs {
		fmt.Fprintf(&b, "%-17s", pair)
		for _, attacker := range element.All() {
			fmt.Fprintf(&b, "%9d", chart.MultiplierAgainst(attacker, pair))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\n%d legal dual combinations\n", len(pairs))
	return b.String()
}

func TestShippedProgressionLimits(t *testing.T) {
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load shipped progression limits: %v", err)
	}
	if got, want := limits.LevelCap, progression.LevelCap; got != want {
		t.Errorf("level_cap is %d, want %d", got, want)
	}
	expected := progression.Values{
		progression.HP: 4800, progression.Attack: 800, progression.Defense: 800,
		progression.Speed: 200, progression.Accuracy: 300, progression.Dodge: 150,
	}
	if limits.Ceilings != expected {
		t.Errorf("ceilings are %s, want %s", limits.Ceilings, expected)
	}
	if got, want := limits.MaxEffectiveHP, int64(11500); got != want {
		t.Errorf("max_effective_hp is %d, want %d", got, want)
	}
	// The two durability ceilings together must be outside the budget,
	// otherwise the budget is not constraining anything.
	both := progression.Values{progression.HP: 4800, progression.Defense: 800}
	if err := limits.CheckValues(both, mustRules(t)); err == nil {
		t.Error("a unit at both durability ceilings passed the budget")
	}
}

// referenceProfiles are level 60 stat lines used to reason about battle length.
// They are not shipped units; they exist so the golden report shows what the
// budget actually buys.
func referenceProfiles() []struct {
	name   string
	values progression.Values
} {
	profile := func(hp, attack, defense, speed, accuracy, dodge int64) progression.Values {
		return progression.Values{
			progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
			progression.Speed: speed, progression.Accuracy: accuracy, progression.Dodge: dodge,
		}
	}
	return []struct {
		name   string
		values progression.Values
	}{
		{"bulwark, max health", profile(4800, 480, 400, 80, 60, 20)},
		{"sentinel, max defence", profile(3100, 500, 800, 90, 80, 30)},
		{"vanguard, balanced", profile(3600, 620, 560, 110, 110, 60)},
		{"duelist, max attack", profile(2600, 800, 320, 150, 180, 90)},
		{"skirmisher, max speed", profile(2200, 760, 250, 200, 240, 150)},
	}
}

func TestReferenceProfilesFitTheBudget(t *testing.T) {
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load shipped progression limits: %v", err)
	}
	rules := mustRules(t)
	for _, profile := range referenceProfiles() {
		if err := limits.CheckValues(profile.values, rules); err != nil {
			t.Errorf("%s (%s) does not fit the budget: %v", profile.name, profile.values, err)
		}
	}
}

func mustRules(t *testing.T) combat.Rules {
	t.Helper()
	rules, err := seed.CombatRules()
	if err != nil {
		t.Fatalf("load shipped combat rules: %v", err)
	}
	return rules
}

func TestProgressionGolden(t *testing.T) {
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load shipped progression limits: %v", err)
	}
	got := progressionReport(limits, mustRules(t))
	path := filepath.Join("testdata", "progression.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("progression report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

func progressionReport(limits progression.Limits, rules combat.Rules) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== budget ==\nlevel cap %d, ceilings %s, effective health budget %d\n",
		limits.LevelCap, limits.Ceilings, limits.MaxEffectiveHP)
	b.WriteString("stats grow linearly: base at level 1, max at the level cap\n")

	b.WriteString("\n== how much defence each health total can afford ==\n")
	b.WriteString("   health  max defence  effective health\n")
	for health := int64(2000); health <= limits.Ceilings[progression.HP]; health += 400 {
		affordable := int64(-1)
		for defense := int64(0); defense <= limits.Ceilings[progression.Defense]; defense++ {
			values := progression.Values{progression.HP: health, progression.Defense: defense}
			if limits.CheckValues(values, rules) != nil {
				break
			}
			affordable = defense
		}
		values := progression.Values{progression.HP: health, progression.Defense: affordable}
		fmt.Fprintf(&b, "%9d  %11d  %16d\n", health, affordable, progression.EffectiveHP(values, rules))
	}

	b.WriteString("\n== hits to kill, struck by 800 attack at a 1800 skill multiplier ==\n")
	tiers := []struct {
		name       string
		multiplier int
	}{
		{"resisted twice", 444},
		{"resisted", 667},
		{"neutral", combat.PermilleBase},
		{"weak", 1500},
		{"weak twice", 2250},
	}
	fmt.Fprintf(&b, "%-22s", "profile")
	for _, tier := range tiers {
		fmt.Fprintf(&b, "%15s", tier.name)
	}
	b.WriteString("\n")
	for _, profile := range referenceProfiles() {
		fmt.Fprintf(&b, "%-22s", profile.name)
		for _, tier := range tiers {
			damage := rules.Damage(800, profile.values[progression.Defense], 1800, tier.multiplier)
			hits := (profile.values[progression.HP] + damage - 1) / damage
			fmt.Fprintf(&b, "%15d", hits)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n== reference profiles at the level cap ==\n")
	for _, profile := range referenceProfiles() {
		fmt.Fprintf(&b, "%-22s %s, absorbs %d\n",
			profile.name, profile.values, progression.EffectiveHP(profile.values, rules))
	}

	b.WriteString("\n== a sample curve, health 320 to 4800 ==\n")
	curve := progression.Curve{Base: 320, Max: 4800}
	b.WriteString("level ")
	for level := 1; level <= progression.LevelCap; level += 10 {
		fmt.Fprintf(&b, "%8d", level)
	}
	fmt.Fprintf(&b, "%8d\n", progression.LevelCap)
	b.WriteString("value ")
	for level := 1; level <= progression.LevelCap; level += 10 {
		fmt.Fprintf(&b, "%8d", curve.At(level))
	}
	fmt.Fprintf(&b, "%8d\n", curve.At(progression.LevelCap))
	return b.String()
}
