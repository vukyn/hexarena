package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/rng"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// The action value scale converts speed into a wait between turns:
// wait = scale / speed. A window of exactly one scale unit therefore gives a
// unit as many turns as it has speed, which is what makes the tables readable.
const (
	actionValueScale = 1_000_000
	coarseScale      = 10_000
	window           = actionValueScale
)

// Reference figures the scenarios are measured against.
const (
	referenceDefense  = 400
	singleTargetPower = 1800
	speedSkillPower   = 9600
	swiftAttack       = 760
	swiftSpeed        = 200
	slowAttack        = 480
	slowSpeed         = 80
	attackerAttack    = 800
	buffedPercent     = 500
	neutralAffinity   = combat.PermilleBase
	advantageAffinity = 1500
	stackedAffinity   = 2250
	resistedAffinity  = 667
	affinityBuff      = 300
	skillAccuracy     = 850
	poisonPower       = 500
)

func mustBounds(t *testing.T) modifier.Bounds {
	t.Helper()
	bounds, err := seed.ModifierBounds()
	if err != nil {
		t.Fatalf("load shipped modifier bounds: %v", err)
	}
	return bounds
}

func mustCeilings(t *testing.T) progression.Values {
	t.Helper()
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load shipped progression limits: %v", err)
	}
	return limits.Ceilings
}

func mustBook(t *testing.T) *pattern.Book {
	t.Helper()
	book, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load shipped pattern book: %v", err)
	}
	return book
}

func TestShippedModifierBounds(t *testing.T) {
	bounds := mustBounds(t)
	if got, want := bounds.Headroom, int64(3000); got != want {
		t.Errorf("headroom is %d, want %d", got, want)
	}
	if got, want := bounds.FloorFraction, int64(100); got != want {
		t.Errorf("floor_fraction is %d, want %d", got, want)
	}
	if got, want := bounds.MaxAffinityScale, int64(1000); got != want {
		t.Errorf("max_affinity_scale is %d, want %d", got, want)
	}
}

func TestShippedPatternBook(t *testing.T) {
	book := mustBook(t)
	if got, want := book.MaxTargets, 3; got != want {
		t.Errorf("max_targets is %d, want %d", got, want)
	}
	if got, want := book.SplashPower, 500; got != want {
		t.Errorf("splash_power is %d, want %d", got, want)
	}
	for _, shape := range book.Patterns() {
		if err := shape.Validate(book.MaxTargets); err != nil {
			t.Errorf("shipped shape %q does not validate: %v", shape.Name, err)
		}
	}
	// Every shape must be able to cover its full width somewhere on the enemy
	// half, otherwise it is a shape that can never do what it claims.
	for _, shape := range book.Patterns() {
		best := 0
		for _, centre := range hex.SideCells(hex.SideEnemy) {
			if covered := len(shape.Targets(centre)); covered > best {
				best = covered
			}
		}
		if best != shape.MaxTargets() {
			t.Errorf("shape %q claims %d cells but reaches at most %d anywhere on the enemy half",
				shape.Name, shape.MaxTargets(), best)
		}
	}
}

// TestSpeedBuffMatchesAttackBuffOnAttackScalingSkills is the reassuring half of
// the speed question: while a skill scales off attack, a percentage of speed and
// the same percentage of attack are worth the same sustained output, because one
// multiplies turns and the other multiplies damage.
func TestSpeedBuffMatchesAttackBuffOnAttackScalingSkills(t *testing.T) {
	rules, bounds, ceilings := mustRules(t), mustBounds(t), mustCeilings(t)
	base := slowProfile()
	plain := output(rules, base, base, singleTargetPower, progression.Attack, combat.CurrentStat)

	withSpeed := output(rules, buff(t, base, ceilings, bounds, modifier.Speed), base,
		singleTargetPower, progression.Attack, combat.CurrentStat)
	withAttack := output(rules, buff(t, base, ceilings, bounds, modifier.Attack), base,
		singleTargetPower, progression.Attack, combat.CurrentStat)

	for _, testCase := range []struct {
		name string
		got  int64
	}{{"speed buff", withSpeed}, {"attack buff", withAttack}} {
		// Saturation trims a +50% term to about +46%, and trims both the same
		// way, so the two must still agree.
		gain := testCase.got * 1000 / plain
		if gain < 1400 || gain > 1500 {
			t.Errorf("%s changed sustained output by %d per mille", testCase.name, gain)
		}
	}
	if difference := withSpeed - withAttack; difference > withAttack/50 || difference < -withAttack/50 {
		t.Errorf("a speed buff gave %d and an attack buff %d, want them within two percent", withSpeed, withAttack)
	}
}

// TestSpeedScalingCompoundsUnlessItReadsTheBaseStat is the warning half: once a
// skill reads speed as its damage stat, reading the buffed value makes a speed
// buff multiply turns and damage at once. Reading the base value keeps the buff
// worth what every other buff is worth.
func TestSpeedScalingCompoundsUnlessItReadsTheBaseStat(t *testing.T) {
	rules, bounds, ceilings := mustRules(t), mustBounds(t), mustCeilings(t)
	base := slowProfile()
	buffed := buff(t, base, ceilings, bounds, modifier.Speed)
	plain := output(rules, base, base, speedSkillPower, progression.Speed, combat.CurrentStat)

	compounded := output(rules, buffed, base, speedSkillPower, progression.Speed, combat.CurrentStat)
	contained := output(rules, buffed, base, speedSkillPower, progression.Speed, combat.BaseStat)

	compoundedGain := compounded * 1000 / plain
	containedGain := contained * 1000 / plain
	if compoundedGain <= containedGain {
		t.Fatalf("reading the current stat gained %d per mille and the base stat %d, want the current one to be larger",
			compoundedGain, containedGain)
	}
	// Reading the current stat multiplies turns and damage by the same factor,
	// so its gain is exactly the square of the gain from turns alone.
	squared := containedGain * containedGain / 1000
	if difference := compoundedGain - squared; difference < -20 || difference > 20 {
		t.Errorf("reading the current stat gained %d per mille, want the square of the base stat's %d, which is %d",
			compoundedGain, containedGain, squared)
	}
}

func slowProfile() progression.Values {
	return progression.Values{
		progression.HP: 4800, progression.Attack: slowAttack, progression.Defense: 400,
		progression.Speed: slowSpeed, progression.Accuracy: 60, progression.Dodge: 20,
	}
}

func swiftProfile() progression.Values {
	return progression.Values{
		progression.HP: 2200, progression.Attack: swiftAttack, progression.Defense: 250,
		progression.Speed: swiftSpeed, progression.Accuracy: 240, progression.Dodge: 150,
	}
}

func buff(t *testing.T, base, ceilings progression.Values, bounds modifier.Bounds, targets ...modifier.Target) progression.Values {
	t.Helper()
	var set modifier.Set
	for _, target := range targets {
		if err := set.Add(modifier.Modifier{Target: target, Mode: modifier.Percent, Amount: buffedPercent}); err != nil {
			t.Fatalf("add buff on %s: %v", target, err)
		}
	}
	return set.Stats(base, ceilings, bounds)
}

// output is total damage over one action-value window: as many turns as the
// unit's current speed, each dealing the skill's damage.
func output(rules combat.Rules, current, base progression.Values, power int, scaling progression.Kind, source combat.ScalingSource) int64 {
	turns := turnsInWindow(current[progression.Speed])
	value := combat.PickScaling(source, base[scaling], current[scaling])
	return turns * rules.Damage(value, referenceDefense, power, neutralAffinity)
}

func turnsInWindow(speed int64) int64 {
	wait := int64(actionValueScale) / speed
	if wait < 1 {
		wait = 1
	}
	return window / wait
}

// TestAreaShapesRarelyCatchTheirFullWidth is the measurement that decides how
// area skills are priced. A shape is a fixed set of cells, so how often it
// covers three occupied slots depends on the formation it is aimed at, not on
// the player's choice.
func TestAreaShapesRarelyCatchTheirFullWidth(t *testing.T) {
	book := mustBook(t)
	for _, shape := range book.Patterns() {
		if shape.MaxTargets() < 3 {
			continue
		}
		average, full := shapeCoverage(shape)
		if average >= 3000 {
			t.Errorf("shape %q averages %d per mille targets, which is its full width", shape.Name, average)
		}
		if full >= 500 {
			t.Errorf("shape %q covers its full width %d per mille of the time, want it uncommon", shape.Name, full)
		}
	}
}

// shapeCoverage averages a shape over every way five units can fill the nine
// enemy slots and every occupied cell it could be aimed at. It returns the mean
// number of units caught in parts per thousand, and how often all three land.
func shapeCoverage(shape pattern.Pattern) (average, full int64) {
	cells := hex.SideCells(hex.SideEnemy)
	aims, caught, fullHits := int64(0), int64(0), int64(0)
	// Each of the 512 subsets of nine slots, restricted to those holding a full
	// team of five.
	for mask := 0; mask < 1<<len(cells); mask++ {
		occupied := make(map[hex.Offset]bool, hex.MaxTeamSize)
		for i, cell := range cells {
			if mask&(1<<i) != 0 {
				occupied[cell] = true
			}
		}
		if len(occupied) != hex.MaxTeamSize {
			continue
		}
		for _, primary := range cells {
			if !occupied[primary] {
				continue
			}
			aims++
			hits := int64(0)
			for _, cell := range shape.Targets(primary) {
				if occupied[cell] {
					hits++
				}
			}
			caught += hits
			if int(hits) == shape.MaxTargets() {
				fullHits++
			}
		}
	}
	if aims == 0 {
		return 0, 0
	}
	return caught * 1000 / aims, fullHits * 1000 / aims
}

func TestScenariosGolden(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	got := scenarioReport(mustRules(t), chart, mustBounds(t), mustCeilings(t), mustBook(t))
	path := filepath.Join("testdata", "scenarios.golden")
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
		t.Errorf("scenario report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

func scenarioReport(rules combat.Rules, chart *element.Chart, bounds modifier.Bounds, ceilings progression.Values, book *pattern.Book) string {
	var b strings.Builder
	fmt.Fprintf(&b, "all damage figures are against %d defence unless stated\n", referenceDefense)
	fmt.Fprintf(&b, "buffs saturate towards %d per mille of a stat's ceiling and debuffs towards %d per mille of the stat,\n",
		bounds.Headroom, bounds.FloorFraction)
	b.WriteString("so neither limit is ever reached and each further term is worth less than the last\n\n")

	writeStatBuffScenario(&b, rules, bounds, ceilings)
	writeDebuffScenario(&b, rules, bounds, ceilings)
	writeAffinityScenario(&b, rules, chart, bounds)
	writeAccuracyScenario(&b, rules, ceilings)
	writeDodgeScenario(&b, rules, ceilings)
	writeBlockScenario(&b, rules)
	writeMultiStrikeScenario(&b, rules)
	writeAreaScenario(&b, rules, book)
	writeStatusScenario(&b, rules)
	writeTurnOrderScenario(&b, rules)
	writeTurnEconomyScenario(&b, rules, bounds, ceilings)
	writeSpeedScalingScenario(&b, rules, bounds, ceilings)
	writeActionValueScenario(&b)
	return b.String()
}

func writeStatBuffScenario(b *strings.Builder, rules combat.Rules, bounds modifier.Bounds, ceilings progression.Values) {
	b.WriteString("== attack buffs ==\n")
	fmt.Fprintf(b, "620 base attack, ceiling %d so the limit it approaches is %d\n",
		ceilings[progression.Attack], ceilings[progression.Attack]*bounds.Headroom/modifier.PercentBase)
	b.WriteString("terms                              raw   actual   absorbed   damage    gain\n")
	base := vanguardProfile()
	plain := rules.Damage(base[progression.Attack], referenceDefense, singleTargetPower, neutralAffinity)
	for _, row := range []struct {
		label     string
		modifiers []modifier.Modifier
	}{
		{"none", nil},
		{"+30%", percentTerms(modifier.Attack, 300)},
		{"+50%", percentTerms(modifier.Attack, 500)},
		{"+50% twice", percentTerms(modifier.Attack, 500, 500)},
		{"+50% four times", percentTerms(modifier.Attack, 500, 500, 500, 500)},
		{"+50% eight times", percentTerms(modifier.Attack, 500, 500, 500, 500, 500, 500, 500, 500)},
		{"+120 flat", []modifier.Modifier{{Target: modifier.Attack, Mode: modifier.Flat, Amount: 120}}},
		{"+120 flat then +50%", []modifier.Modifier{
			{Target: modifier.Attack, Mode: modifier.Flat, Amount: 120},
			{Target: modifier.Attack, Mode: modifier.Percent, Amount: 500},
		}},
	} {
		var set modifier.Set
		_ = set.AddAll(row.modifiers...)
		raw := set.Raw(progression.Attack, base[progression.Attack])
		actual := set.Stat(progression.Attack, base[progression.Attack], ceilings[progression.Attack], bounds)
		damage := rules.Damage(actual, referenceDefense, singleTargetPower, neutralAffinity)
		fmt.Fprintf(b, "%-28s%8d%9d%11d%9d%8s\n", row.label, raw, actual, raw-actual, damage, ratio(damage, plain))
	}
	b.WriteString("\n")
}

func writeDebuffScenario(b *strings.Builder, rules combat.Rules, bounds modifier.Bounds, ceilings progression.Values) {
	b.WriteString("== defence debuffs ==\n")
	fmt.Fprintf(b, "800 attack against 560 defence, which cannot be driven below %d\n",
		560*bounds.FloorFraction/modifier.PercentBase)
	b.WriteString("terms                              raw   actual   damage    gain\n")
	base := vanguardProfile()
	plain := rules.Damage(attackerAttack, base[progression.Defense], singleTargetPower, neutralAffinity)
	for _, row := range []struct {
		label     string
		modifiers []modifier.Modifier
	}{
		{"none", nil},
		{"-30%", percentTerms(modifier.Defense, -300)},
		{"-30% twice", percentTerms(modifier.Defense, -300, -300)},
		{"-30% four times", percentTerms(modifier.Defense, -300, -300, -300, -300)},
		{"-100%", percentTerms(modifier.Defense, -1000)},
		{"-300%", percentTerms(modifier.Defense, -3000)},
	} {
		var set modifier.Set
		_ = set.AddAll(row.modifiers...)
		raw := set.Raw(progression.Defense, base[progression.Defense])
		actual := set.Stat(progression.Defense, base[progression.Defense], ceilings[progression.Defense], bounds)
		damage := rules.Damage(attackerAttack, actual, singleTargetPower, neutralAffinity)
		fmt.Fprintf(b, "%-28s%8d%9d%9d%8s\n", row.label, raw, actual, damage, ratio(damage, plain))
	}
	b.WriteString("a term past -100% is safe to author, because it saturates instead of inverting\n\n")
}

func writeAffinityScenario(b *strings.Builder, rules combat.Rules, chart *element.Chart, bounds modifier.Bounds) {
	b.WriteString("== elemental effectiveness ==\n")
	fmt.Fprintf(b, "the term scales how far a matchup deviates from neutral, clamped to %+d per mille\n", bounds.MaxAffinityScale)
	b.WriteString("so it is worth more the worse the defender's matchup already is\n")
	b.WriteString("term       matchup           multiplier   shifted   damage   shifted   gain\n")
	neutral := chart.Multipliers().Neutral
	for _, row := range []struct {
		label      string
		amount     int64
		matchupTag string
		multiplier int
	}{
		{"+30%", 300, "weak twice", stackedAffinity},
		{"+30%", 300, "weak", advantageAffinity},
		{"+30%", 300, "neutral", neutral},
		{"+30%", 300, "resisted", resistedAffinity},
		{"+100%", 1000, "weak twice", stackedAffinity},
		{"+100%", 1000, "weak", advantageAffinity},
		{"-100%", -1000, "weak twice", stackedAffinity},
		{"-100%", -1000, "weak", advantageAffinity},
	} {
		var set modifier.Set
		_ = set.Add(modifier.Modifier{Target: modifier.Affinity, Mode: modifier.Percent, Amount: row.amount})
		shifted := set.Affinity(row.multiplier, neutral, bounds)
		plain := rules.Damage(attackerAttack, referenceDefense, singleTargetPower, row.multiplier)
		buffed := rules.Damage(attackerAttack, referenceDefense, singleTargetPower, shifted)
		fmt.Fprintf(b, "%-11s%-18s%11d%10d%9d%10d%7s\n",
			row.label, row.matchupTag, row.multiplier, shifted, plain, buffed, ratio(buffed, plain))
	}
	b.WriteString("\n")
}

func writeAccuracyScenario(b *strings.Builder, rules combat.Rules, ceilings progression.Values) {
	b.WriteString("== accuracy ==\n")
	b.WriteString("a skill's own accuracy is the floor and a certain hit is the limit it never reaches,\n")
	fmt.Fprintf(b, "so the accuracy stat, ceiling %d, closes the gap with diminishing returns\n", ceilings[progression.Accuracy])
	b.WriteString("skill accuracy ")
	for stat := int64(0); stat <= 300; stat += 60 {
		fmt.Fprintf(b, "%8d", stat)
	}
	b.WriteString("   <- accuracy stat\n")
	for _, accuracy := range []int{600, 700, 800, 850, 900, 950, 1000} {
		fmt.Fprintf(b, "%14d ", accuracy)
		for stat := int64(0); stat <= 300; stat += 60 {
			fmt.Fprintf(b, "%8d", rules.Chance(combat.Hit{SkillAccuracy: accuracy, AccuracyStat: stat}))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nan accuracy stat of a thousand times the ceiling still falls short of certainty\n")
	for _, accuracy := range []int{700, 900} {
		fmt.Fprintf(b, "  a %d skill with 300000 accuracy lands %d per mille of the time\n",
			accuracy, rules.Chance(combat.Hit{SkillAccuracy: accuracy, AccuracyStat: 300_000}))
	}
	fmt.Fprintf(b, "  a %d skill cannot miss at any accuracy, which is how a guaranteed effect is written\n", 1000)

	b.WriteString("\nrolled over 100000 attempts, 800 attack, 1800 per mille\n")
	b.WriteString("shape                   chance   landed   expected   measured\n")
	for _, row := range []struct {
		label   string
		strikes int
	}{
		{"single strike", 1},
		{"two strikes", 2},
		{"three strikes", 3},
	} {
		hit := combat.Hit{
			Scaling: attackerAttack, Multiplier: singleTargetPower / row.strikes, Strikes: row.strikes,
			Affinity: neutralAffinity, Defense: referenceDefense, SkillAccuracy: skillAccuracy, AccuracyStat: 120,
		}
		source := rng.New(0x5CE7A)
		const draws = 100_000
		total := int64(0)
		for i := 0; i < draws; i++ {
			attempts, _ := rules.Roll(hit, 0, source)
			total += combat.DamageDealt(attempts)
		}
		chance := rules.Chance(hit)
		expected := rules.Total(hit) * int64(chance) / combat.PermilleBase
		fmt.Fprintf(b, "%-24s%7d%9d%11d%11d\n",
			row.label, chance, rules.Total(hit), expected, total/draws)
	}
	b.WriteString("splitting a skill into strikes leaves the expected damage alone and cuts the variance,\n")
	b.WriteString("because every strike rolls on its own\n\n")
}

func writeMultiStrikeScenario(b *strings.Builder, rules combat.Rules) {
	b.WriteString("== multi strike, ignoring accuracy ==\n")
	b.WriteString("1800 per mille of total power split across strikes, 800 attack\n")
	b.WriteString("shape            per strike    total   lost to truncation\n")
	single := rules.Total(combat.Hit{
		Scaling: attackerAttack, Multiplier: singleTargetPower, Strikes: 1,
		Affinity: neutralAffinity, Defense: referenceDefense,
	})
	for _, strikes := range []int{1, 2, 3, 5, 6} {
		hit := combat.Hit{
			Scaling: attackerAttack, Multiplier: singleTargetPower / strikes, Strikes: strikes,
			Affinity: neutralAffinity, Defense: referenceDefense,
		}
		total := rules.Total(hit)
		fmt.Fprintf(b, "%2d x %-12d%11d%9d%21d\n",
			strikes, singleTargetPower/strikes, rules.Strike(hit), total, single-total)
	}
	b.WriteString("\n")
}

func writeAreaScenario(b *strings.Builder, rules combat.Rules, book *pattern.Book) {
	fmt.Fprintf(b, "== area shapes, limit %d targets, splash at %d per mille ==\n", book.MaxTargets, book.SplashPower)
	b.WriteString("aimed at every occupied slot across all ways five units fill the nine enemy slots\n")
	b.WriteString("shape             width   mean caught   full width   best aim   worst aim\n")
	shapes := book.Patterns()
	sort.SliceStable(shapes, func(i, j int) bool {
		return shapes[i].MaxTargets() < shapes[j].MaxTargets()
	})
	for _, shape := range shapes {
		mean, full := shapeCoverage(shape)
		best, worst := 0, 99
		for _, centre := range hex.SideCells(hex.SideEnemy) {
			covered := len(shape.Targets(centre))
			if covered > best {
				best = covered
			}
			if covered < worst {
				worst = covered
			}
		}
		fmt.Fprintf(b, "%-18s%6d%11d.%03d%13d%11d%12d\n",
			shape.Name, shape.MaxTargets(), mean/1000, mean%1000, full, best, worst)
	}
	b.WriteString("\nreach of each shape, per aiming cell, counting only cells on the enemy half\n")
	b.WriteString("shape       ")
	for _, centre := range hex.SideCells(hex.SideEnemy) {
		fmt.Fprintf(b, "%6s", centre)
	}
	b.WriteString("\n")
	for _, shape := range shapes {
		fmt.Fprintf(b, "%-12s", shape.Name)
		for _, centre := range hex.SideCells(hex.SideEnemy) {
			fmt.Fprintf(b, "%6d", len(shape.Targets(centre)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\ntotal output, primary at full power and splash at a share of it, 800 attack\n")
	// The widest shape's average is what an area skill is actually worth, since
	// covering the full width is the exception rather than the rule.
	widest, meanCaught := pattern.Pattern{}, int64(0)
	for _, shape := range shapes {
		if mean, _ := shapeCoverage(shape); mean > meanCaught {
			widest, meanCaught = shape, mean
		}
	}
	fmt.Fprintf(b, "expected uses the best shape %q, which averages %d.%03d targets caught\n",
		widest.Name, meanCaught/1000, meanCaught%1000)
	b.WriteString("power   splash share   1 caught   2 caught   3 caught   at 3   expected   expected gain\n")
	single := rules.Damage(attackerAttack, referenceDefense, singleTargetPower, neutralAffinity)
	for _, row := range []struct {
		power  int
		splash int
	}{
		{singleTargetPower, 1000},
		{singleTargetPower, book.SplashPower},
		{singleTargetPower, 350},
		{singleTargetPower * 7 / 10, book.SplashPower},
		{singleTargetPower * 6 / 10, book.SplashPower},
	} {
		primary := rules.Damage(attackerAttack, referenceDefense, row.power, neutralAffinity)
		splash := rules.Damage(attackerAttack, referenceDefense, row.power*row.splash/1000, neutralAffinity)
		expected := primary + splash*(meanCaught-1000)/1000
		fmt.Fprintf(b, "%5d%15d%11d%11d%11d%7s%11d%16s\n",
			row.power, row.splash, primary, primary+splash, primary+splash*2,
			ratio(primary+splash*2, single), expected, ratio(expected, single))
	}
	b.WriteString("\n")
}

func writeTurnEconomyScenario(b *strings.Builder, rules combat.Rules, bounds modifier.Bounds, ceilings progression.Values) {
	b.WriteString("== speed buffs while skills scale off attack ==\n")
	fmt.Fprintf(b, "one window of %d action value, so turns equal speed\n", window)
	b.WriteString("terms                         speed   turns   attack   per turn      total    gain\n")
	base := slowProfile()
	plain := output(rules, base, base, singleTargetPower, progression.Attack, combat.CurrentStat)
	for _, row := range []struct {
		label     string
		modifiers []modifier.Modifier
	}{
		{"unbuffed", nil},
		{"speed +50%", percentTerms(modifier.Speed, buffedPercent)},
		{"attack +50%", percentTerms(modifier.Attack, buffedPercent)},
		{"both +50%", append(percentTerms(modifier.Speed, buffedPercent), percentTerms(modifier.Attack, buffedPercent)...)},
		{"speed -30%", percentTerms(modifier.Speed, -300)},
	} {
		var set modifier.Set
		_ = set.AddAll(row.modifiers...)
		values := set.Stats(base, ceilings, bounds)
		turns := turnsInWindow(values[progression.Speed])
		perTurn := rules.Damage(values[progression.Attack], referenceDefense, singleTargetPower, neutralAffinity)
		total := turns * perTurn
		fmt.Fprintf(b, "%-30s%6d%8d%9d%11d%11d%8s\n",
			row.label, values[progression.Speed], turns, values[progression.Attack], perTurn, total, ratio(total, plain))
	}
	b.WriteString("a percentage of speed and the same percentage of attack are worth the same output:\n")
	b.WriteString("one multiplies turns, the other multiplies damage\n\n")
}

func writeSpeedScalingScenario(b *strings.Builder, rules combat.Rules, bounds modifier.Bounds, ceilings progression.Values) {
	b.WriteString("== speed as a damage stat ==\n")
	fmt.Fprintf(b, "an attack scaling skill at %d per mille against a speed scaling one at %d,\n",
		singleTargetPower, speedSkillPower)
	b.WriteString("chosen so a 150 speed, 800 attack duelist hits for the same amount with either\n\n")
	b.WriteString("unit                     speed  attack   turns   attack skill   speed skill\n")
	units := []struct {
		label  string
		values progression.Values
	}{{"swift", swiftProfile()}, {"slow", slowProfile()}}
	totals := make(map[string][2]int64)
	for _, unit := range units {
		byAttack := output(rules, unit.values, unit.values, singleTargetPower, progression.Attack, combat.CurrentStat)
		bySpeed := output(rules, unit.values, unit.values, speedSkillPower, progression.Speed, combat.CurrentStat)
		totals[unit.label] = [2]int64{byAttack, bySpeed}
		fmt.Fprintf(b, "%-24s%6d%8d%8d%15d%14d\n", unit.label,
			unit.values[progression.Speed], unit.values[progression.Attack],
			turnsInWindow(unit.values[progression.Speed]), byAttack, bySpeed)
	}
	fmt.Fprintf(b, "%-24s%6s%8s%8s%15s%14s\n", "swift over slow", "", "", "",
		ratio(totals["swift"][0], totals["slow"][0]), ratio(totals["swift"][1], totals["slow"][1]))

	b.WriteString("\nwhat a +50% speed buff is worth to the slow unit on a speed scaling skill\n")
	b.WriteString("scaling source                          turns   per turn      total    gain\n")
	base := slowProfile()
	var set modifier.Set
	_ = set.AddAll(percentTerms(modifier.Speed, buffedPercent)...)
	buffed := set.Stats(base, ceilings, bounds)
	plain := output(rules, base, base, speedSkillPower, progression.Speed, combat.CurrentStat)
	for _, source := range []combat.ScalingSource{combat.CurrentStat, combat.BaseStat} {
		turns := turnsInWindow(buffed[progression.Speed])
		value := combat.PickScaling(source, base[progression.Speed], buffed[progression.Speed])
		perTurn := rules.Damage(value, referenceDefense, speedSkillPower, neutralAffinity)
		total := turns * perTurn
		fmt.Fprintf(b, "%-38s%8d%11d%11d%8s\n", source, turns, perTurn, total, ratio(total, plain))
	}
	b.WriteString("a skill declares which it reads: the current stat accepts the compounding,\n")
	b.WriteString("the base stat lets the buff move the turn economy only\n\n")
}

func writeActionValueScenario(b *strings.Builder) {
	b.WriteString("== action value resolution ==\n")
	b.WriteString("the wait between turns is scale / speed, so a coarse scale makes nearby speeds identical\n")
	fmt.Fprintf(b, "speed   wait at %d   wait at %d\n", coarseScale, actionValueScale)
	for _, speed := range []int64{80, 110, 150, 180, 190, 196, 198, 199, 200} {
		fmt.Fprintf(b, "%5d%15d%17d\n", speed, coarseScale/speed, actionValueScale/speed)
	}
	coarse, fine := 0, 0
	for speed := int64(2); speed <= 200; speed++ {
		if coarseScale/speed == coarseScale/(speed-1) {
			coarse++
		}
		if actionValueScale/speed == actionValueScale/(speed-1) {
			fine++
		}
	}
	fmt.Fprintf(b, "\n%d of the 199 speeds from 2 to 200 are indistinguishable from the speed below at scale %d\n", coarse, coarseScale)
	fmt.Fprintf(b, "%d of them are indistinguishable at scale %d\n", fine, actionValueScale)
}

func vanguardProfile() progression.Values {
	return progression.Values{
		progression.HP: 3600, progression.Attack: 620, progression.Defense: 560,
		progression.Speed: 110, progression.Accuracy: 110, progression.Dodge: 60,
	}
}

func percentTerms(target modifier.Target, amounts ...int64) []modifier.Modifier {
	out := make([]modifier.Modifier, 0, len(amounts))
	for _, amount := range amounts {
		out = append(out, modifier.Modifier{Target: target, Mode: modifier.Percent, Amount: amount})
	}
	return out
}

func ratio(value, reference int64) string {
	if reference == 0 {
		return "-"
	}
	scaled := value * 100 / reference
	return fmt.Sprintf("%d.%02dx", scaled/100, scaled%100)
}

func writeDodgeScenario(b *strings.Builder, rules combat.Rules, ceilings progression.Values) {
	b.WriteString("== dodge ==\n")
	fmt.Fprintf(b, "dodge reopens the gap accuracy closed, towards a floor of %d it never reaches\n", rules.MinHitChance)
	fmt.Fprintf(b, "it is applied after accuracy rather than subtracted from it, so it bites even against\n")
	b.WriteString("an attack that was going to land almost every time\n")
	b.WriteString("chance before dodge ")
	for dodge := int64(0); dodge <= ceilings[progression.Dodge]; dodge += 30 {
		fmt.Fprintf(b, "%7d", dodge)
	}
	b.WriteString("   <- dodge stat\n")
	for _, before := range []struct {
		accuracy int
		stat     int64
	}{
		{600, 0}, {700, 0}, {850, 0}, {850, 300}, {950, 300}, {990, 300},
	} {
		plain := rules.Chance(combat.Hit{SkillAccuracy: before.accuracy, AccuracyStat: before.stat})
		fmt.Fprintf(b, "%19d ", plain)
		for dodge := int64(0); dodge <= ceilings[progression.Dodge]; dodge += 30 {
			hit := combat.Hit{SkillAccuracy: before.accuracy, AccuracyStat: before.stat, DodgeStat: dodge}
			fmt.Fprintf(b, "%7d", rules.Chance(hit))
		}
		b.WriteString("\n")
	}

	b.WriteString("\nwhat each stat is worth against an 850 per mille skill\n")
	b.WriteString("stat        points   chance   change   per point\n")
	baseline := rules.Chance(combat.Hit{SkillAccuracy: 850})
	for _, row := range []struct {
		label    string
		points   int64
		accuracy int64
		dodge    int64
	}{
		{"none", 0, 0, 0},
		{"accuracy", ceilings[progression.Accuracy], ceilings[progression.Accuracy], 0},
		{"dodge", ceilings[progression.Dodge], 0, ceilings[progression.Dodge]},
	} {
		chance := rules.Chance(combat.Hit{SkillAccuracy: 850, AccuracyStat: row.accuracy, DodgeStat: row.dodge})
		change := chance - baseline
		perPoint := int64(0)
		if row.points > 0 {
			perPoint = int64(change) * 100 / row.points
		}
		fmt.Fprintf(b, "%-12s%7d%9d%9d%9d.%02d\n",
			row.label, row.points, chance, change, perPoint/100, abs64(perPoint)%100)
	}
	fmt.Fprintf(b, "the dodge ceiling is %d against accuracy's %d, which is what pays for dodge working\n",
		ceilings[progression.Dodge], ceilings[progression.Accuracy])
	b.WriteString("against a wider gap and therefore being worth more per point\n")

	b.WriteString("\na guaranteed skill ignores dodge entirely, which is what leaves block a job to do\n")
	for _, dodge := range []int64{0, 150, 300000} {
		fmt.Fprintf(b, "  a 1000 skill against %6d dodge lands %d per mille of the time\n",
			dodge, rules.Chance(combat.Hit{SkillAccuracy: combat.PermilleBase, DodgeStat: dodge}))
	}
	b.WriteString("\n")
}

func writeBlockScenario(b *strings.Builder, rules combat.Rules) {
	b.WriteString("== block charges ==\n")
	b.WriteString("a charge cancels one strike that would otherwise have landed, guaranteed hits included,\n")
	b.WriteString("and is only spent on a strike that connects, so dodging wastes none\n")
	b.WriteString("a guaranteed skill splitting 1800 per mille of power, against one charge, 800 attack\n")
	b.WriteString("shape            per strike   blocked   struck    dealt   share stopped\n")
	source := rng.New(0xB10C)
	for _, strikes := range []int{1, 2, 3, 5, 6} {
		hit := combat.Hit{
			Scaling: attackerAttack, Multiplier: singleTargetPower / strikes, Strikes: strikes,
			Affinity: neutralAffinity, Defense: referenceDefense, SkillAccuracy: combat.PermilleBase,
		}
		attempts, _ := rules.Roll(hit, 1, source)
		unblocked := rules.Total(hit)
		dealt := combat.DamageDealt(attempts)
		stopped := int64(0)
		if unblocked > 0 {
			stopped = (unblocked - dealt) * 1000 / unblocked
		}
		fmt.Fprintf(b, "%2d x %-12d%11d%10d%9d%9d%12d.%d%%\n",
			strikes, singleTargetPower/strikes, rules.Strike(hit),
			combat.Count(attempts, combat.Blocked), combat.Count(attempts, combat.Struck),
			dealt, stopped/10, stopped%10)
	}
	b.WriteString("one charge erases a single heavy blow and is burned by a sixth of one,\n")
	b.WriteString("so multi strike is the answer to block just as block is the answer to a heavy hit\n")

	fmt.Fprintf(b, "\nseveral charges at once, capped at %d, against a guaranteed 3 x 600 skill\n", rules.MaxBlockCharges)
	b.WriteString("charges   blocked   struck    dealt   share stopped   prevented per charge\n")
	split := combat.Hit{
		Scaling: attackerAttack, Multiplier: singleTargetPower / 3, Strikes: 3,
		Affinity: neutralAffinity, Defense: referenceDefense, SkillAccuracy: combat.PermilleBase,
	}
	unblocked := rules.Total(split)
	for charges := 0; charges <= rules.MaxBlockCharges+1; charges++ {
		held, _ := rules.GrantBlocks(0, charges)
		attempts, _ := rules.Roll(split, held, source)
		dealt := combat.DamageDealt(attempts)
		stopped := (unblocked - dealt) * 1000 / unblocked
		perCharge := int64(0)
		if held > 0 {
			perCharge = (unblocked - dealt) / int64(held)
		}
		fmt.Fprintf(b, "%7d%10d%9d%9d%12d.%d%%%23d\n",
			charges, combat.Count(attempts, combat.Blocked), combat.Count(attempts, combat.Struck),
			dealt, stopped/10, stopped%10, perCharge)
	}
	b.WriteString("every charge is worth exactly as much as the first: charges are the one defence\n")
	b.WriteString("in the engine that does not saturate, which is why they are capped instead\n")

	fmt.Fprintf(b, "\nwhat a full stack of %d charges is worth against single heavy hits\n", rules.MaxBlockCharges)
	b.WriteString("incoming attacks   dealt without charges   dealt with them   share stopped\n")
	heavy := combat.Hit{
		Scaling: attackerAttack, Multiplier: singleTargetPower, Strikes: 1,
		Affinity: neutralAffinity, Defense: referenceDefense, SkillAccuracy: combat.PermilleBase,
	}
	perHit := rules.Strike(heavy)
	for incoming := 1; incoming <= 5; incoming++ {
		held := rules.MaxBlockCharges
		dealt := int64(0)
		for i := 0; i < incoming; i++ {
			attempts, left := rules.Roll(heavy, held, source)
			dealt += combat.DamageDealt(attempts)
			held = left
		}
		bare := perHit * int64(incoming)
		stopped := (bare - dealt) * 1000 / bare
		fmt.Fprintf(b, "%16d%24d%18d%12d.%d%%\n", incoming, bare, dealt, stopped/10, stopped%10)
	}
	b.WriteString("a full stack negates a whole round of heavy single hits, so it belongs on a long\n")
	b.WriteString("cooldown; against chip damage the same stack is worth almost nothing\n")

	b.WriteString("\nthe counter triangle\n")
	b.WriteString("  attacker choice        beaten by        beats\n")
	for _, row := range [][3]string{
		{"normal accuracy", "dodge", "block, which it wastes on misses"},
		{"guaranteed hit", "block", "dodge, which it ignores"},
		{"multi strike", "nothing directly", "dodge by averaging, block by burning charges"},
		{"single heavy hit", "block, dodge", "armour, since one big number beats the defence curve"},
	} {
		fmt.Fprintf(b, "  %-22s %-16s %s\n", row[0], row[1], row[2])
	}
	b.WriteString("\n")
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func mustStatuses(t *testing.T) *status.Book {
	t.Helper()
	book, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load shipped status book: %v", err)
	}
	return book
}

func TestShippedStatusBook(t *testing.T) {
	book := mustStatuses(t)
	rules := mustRules(t)
	blockKind, err := book.Lookup("block")
	if err != nil {
		t.Fatalf("the shipped statuses have no block: %v", err)
	}
	// Block charges are a shield status, so the two declarations of how many a
	// unit may hold have to agree or one of them is dead data.
	if got, want := blockKind.MaxStacks, rules.MaxBlockCharges; got != want {
		t.Errorf("the block status allows %d stacks but the damage rules cap charges at %d", got, want)
	}
	if blockKind.Category != status.Shield {
		t.Errorf("block is categorised as %s, want shield", blockKind.Category)
	}
	if blockKind.Duration < 1 {
		t.Errorf("block lasts %d turns, so charges would never expire", blockKind.Duration)
	}
	// Every declared category should be represented, otherwise a cleanse aimed
	// at one of them has nothing to work on and the category is untested.
	seen := make(map[status.Category]bool)
	for _, kind := range book.Kinds() {
		seen[kind.Category] = true
	}
	for _, category := range status.Categories() {
		if !seen[category] {
			t.Errorf("no shipped status is in the %s category", category)
		}
	}
}

// TestDamageOverTimeCannotBeDodgedOrBlocked is the role the damage model was
// chosen for. A tick is not rolled and not offered to a block charge, so a
// status is the answer to a target that has stacked both defences.
func TestDamageOverTimeCannotBeDodgedOrBlocked(t *testing.T) {
	book, rules := mustStatuses(t), mustRules(t)
	poison, err := book.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup poison: %v", err)
	}
	// A target with the dodge ceiling and a full stack of charges.
	evasive := combat.Hit{
		Scaling: attackerAttack, Multiplier: singleTargetPower, Strikes: 1,
		Affinity: neutralAffinity, Defense: referenceDefense,
		SkillAccuracy: skillAccuracy, DodgeStat: 150,
	}
	if chance := rules.Chance(evasive); chance >= combat.PermilleBase {
		if chance == combat.PermilleBase {
			t.Fatal("the evasive target can still be hit with certainty")
		}
	}
	tick := rules.Damage(attackerAttack, referenceDefense, poisonPower, neutralAffinity)
	var set status.Set
	set.Apply(poison, tick)
	damage, _, _ := set.Tick()
	if damage != tick {
		t.Errorf("the tick dealt %d against a dodging, shielded target, want the full %d", damage, tick)
	}
}

// TestTheBestCleanseLandsOnTheLastApplication records a result that runs against
// the obvious guess. Cleansing as early as possible is not best, because the
// attacker simply reapplies and rebuilds the stack; cleansing late is not best
// either, because the ticks have already been paid. The cheapest turn to cleanse
// is the one the attacker stops on, which means the defender has to read when
// that is. That is what makes the timing a decision rather than a reflex.
func TestTheBestCleanseLandsOnTheLastApplication(t *testing.T) {
	book, rules := mustStatuses(t), mustRules(t)
	poison, err := book.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup poison: %v", err)
	}
	tick := rules.Damage(attackerAttack, referenceDefense, poisonPower, neutralAffinity)
	const horizon = 8
	unchecked := poisonRamp(poison, tick, horizon, 0)

	// The last application is on turn MaxStacks and it lasts Duration turns, so
	// the final tick is on this turn; past it there is nothing left to cleanse.
	lastLiveTurn := poison.MaxStacks + poison.Duration - 1
	best, bestTurn := int64(-1), 0
	for cleanseTurn := 1; cleanseTurn <= lastLiveTurn; cleanseTurn++ {
		through := poisonRamp(poison, tick, horizon, cleanseTurn)
		if through >= unchecked {
			t.Errorf("cleansing on turn %d let through %d, no better than %d uncleansed",
				cleanseTurn, through, unchecked)
		}
		if best < 0 || through < best {
			best, bestTurn = through, cleanseTurn
		}
	}
	// A cleanse after the status has run out is a wasted action, not a smaller
	// saving, and the report should not imply otherwise.
	for cleanseTurn := lastLiveTurn + 1; cleanseTurn <= horizon; cleanseTurn++ {
		if through := poisonRamp(poison, tick, horizon, cleanseTurn); through != unchecked {
			t.Errorf("cleansing on turn %d, after the status ran out, changed the damage to %d from %d",
				cleanseTurn, through, unchecked)
		}
	}
	// The attacker applies one stack a turn until the status is full, so its
	// last application is on the turn the stack count reaches its cap.
	if bestTurn != poison.MaxStacks {
		t.Errorf("the cheapest cleanse is turn %d, want turn %d, the attacker's last application",
			bestTurn, poison.MaxStacks)
	}
	// Cleansing into a reapplication is measurably worse than waiting for it.
	if early := poisonRamp(poison, tick, horizon, 1); early <= best {
		t.Errorf("cleansing on the first turn let through %d, want more than the best turn's %d", early, best)
	}
	// And so is cleansing after the ticks have been paid.
	if late := poisonRamp(poison, tick, horizon, poison.MaxStacks+2); late <= best {
		t.Errorf("cleansing late let through %d, want more than the best turn's %d", late, best)
	}
}

// poisonRamp runs a victim's turns while an attacker applies one stack a turn
// for as long as the status allows, and returns the damage that got through. A
// cleanseTurn of zero means no cleanse.
func poisonRamp(poison status.Kind, tick int64, turns, cleanseTurn int) int64 {
	var set status.Set
	total := int64(0)
	for turn := 1; turn <= turns; turn++ {
		if turn <= poison.MaxStacks {
			set.Apply(poison, tick)
		}
		if turn == cleanseTurn {
			set.Cleanse([]status.Category{status.Dot}, poison.MaxStacks)
		}
		damage, _, _ := set.Tick()
		total += damage
	}
	return total
}

func writeStatusScenario(b *strings.Builder, rules combat.Rules) {
	book, err := seed.StatusBook()
	if err != nil {
		fmt.Fprintf(b, "== statuses ==\nunavailable: %v\n\n", err)
		return
	}
	poison, err := book.Lookup("poison")
	if err != nil {
		fmt.Fprintf(b, "== statuses ==\nunavailable: %v\n\n", err)
		return
	}

	fmt.Fprintf(b, "== damage over time, poison at %d per mille per stack per tick ==\n", poisonPower)
	fmt.Fprintf(b, "%d stacks at most, %d turns each, refreshed on every application\n",
		poison.MaxStacks, poison.Duration)
	b.WriteString("ticks go through the defence curve like any other damage and are never rolled,\n")
	b.WriteString("so a status cannot be dodged or blocked once it has landed\n\n")

	b.WriteString("the ramp against 400 defence, one stack applied per turn for three turns\n")
	b.WriteString("victim turn   stacks   tick   cumulative   as direct hits\n")
	tick := rules.Damage(attackerAttack, referenceDefense, poisonPower, neutralAffinity)
	direct := rules.Damage(attackerAttack, referenceDefense, singleTargetPower, neutralAffinity)
	var set status.Set
	cumulative := int64(0)
	for turn := 1; turn <= 7; turn++ {
		if turn <= poison.MaxStacks {
			set.Apply(poison, tick)
		}
		stacks := set.Stacks("poison")
		damage, _, _ := set.Tick()
		cumulative += damage
		fmt.Fprintf(b, "%11d%9d%7d%13d%17s\n", turn, stacks, damage, cumulative, ratio(cumulative, direct))
	}
	fmt.Fprintf(b, "\nthe three turns spent applying could have been three direct hits for %d,\n", direct*3)
	fmt.Fprintf(b, "so the poison is worth %s of the damage it gave up, in exchange for\n", ratio(cumulative, direct*3))
	b.WriteString("ignoring dodge and block and for setting up a skill that needs a status\n")

	b.WriteString("\nthe same ramp against each profile, showing the defence curve still applies\n")
	b.WriteString("profile                  def   tick at 3 stacks   total   three direct hits    ratio\n")
	for _, profile := range referenceProfiles() {
		defense := profile.values[progression.Defense]
		perStack := rules.Damage(attackerAttack, defense, poisonPower, neutralAffinity)
		hit := rules.Damage(attackerAttack, defense, singleTargetPower, neutralAffinity)
		total := int64(0)
		var run status.Set
		for turn := 1; turn <= 7; turn++ {
			if turn <= poison.MaxStacks {
				run.Apply(poison, perStack)
			}
			damage, _, _ := run.Tick()
			total += damage
		}
		fmt.Fprintf(b, "%-24s%4d%19d%8d%20d%9s\n",
			profile.name, defense, perStack*int64(poison.MaxStacks), total, hit*3, ratio(total, hit*3))
	}

	b.WriteString("\nwhen a cleanse lands, and what it saves\n")
	b.WriteString("cleanse on turn   damage through   saved   share saved\n")
	unchecked := poisonRamp(poison, tick, 8, 0)
	for cleanseTurn := 1; cleanseTurn <= 6; cleanseTurn++ {
		through := poisonRamp(poison, tick, 8, cleanseTurn)
		saved := unchecked - through
		fmt.Fprintf(b, "%15d%17d%8d%11d.%d%%\n",
			cleanseTurn, through, saved, saved*1000/unchecked/10, saved*1000/unchecked%10)
	}
	fmt.Fprintf(b, "%15s%17d%8d%11d.%d%%\n", "never", unchecked, 0, 0, 0)
	b.WriteString("the cheapest turn to cleanse is the attacker's last application, not the first:\n")
	b.WriteString("cleanse earlier and the stack is simply rebuilt, cleanse later and the ticks are\n")
	b.WriteString("already paid, so the defender has to read when the attacker stops\n")

	b.WriteString("\nwhat a detonate gives up, by the stack count it consumes\n")
	b.WriteString("stacks consumed   remaining ticks lost   a burst matching it needs power\n")
	for stacks := 1; stacks <= poison.MaxStacks; stacks++ {
		var run status.Set
		for i := 0; i < stacks; i++ {
			run.Apply(poison, tick)
		}
		// Consume reports the ticks the stacks still owed, which is exactly what
		// the detonate throws away — nothing here multiplies by the duration a
		// second time.
		_, forgone := run.Consume("poison")
		power := forgone * int64(referenceDefense+rules.DefenseConstant) * combat.PermilleBase /
			(attackerAttack * rules.DefenseConstant)
		fmt.Fprintf(b, "%15d%23d%33d\n", stacks, forgone, power)
	}
	b.WriteString("a detonate priced at that power is exactly break even, so it should sit below it\n")
	b.WriteString("and be paid for by arriving now instead of over three turns\n")

	b.WriteString("\ntotal damage does not depend on speed, because duration counts the victim's turns\n")
	b.WriteString("turns simulated   total\n")
	for _, turns := range []int{3, 10, 40} {
		var run status.Set
		run.Apply(poison, tick)
		total := int64(0)
		for turn := 0; turn < turns; turn++ {
			damage, _, _ := run.Tick()
			total += damage
		}
		fmt.Fprintf(b, "%15d%8d\n", turns, total)
	}
	b.WriteString("a hasted victim takes its ticks sooner, not more often\n\n")
}

// battleSpeeds is a five on five using the reference profiles' speeds on both
// sides, which is the turn order the rest of the design is reasoned against.
func battleSpeeds() []struct {
	id    string
	speed int64
} {
	type entry = struct {
		id    string
		speed int64
	}
	return []entry{
		{"ally.bulwark", 80}, {"ally.sentinel", 90}, {"ally.vanguard", 110},
		{"ally.duelist", 150}, {"ally.skirmisher", 200},
		{"foe.bulwark", 80}, {"foe.sentinel", 90}, {"foe.vanguard", 110},
		{"foe.duelist", 150}, {"foe.skirmisher", 200},
	}
}

func battleQueue(t *testing.T) *atb.Queue {
	t.Helper()
	queue := atb.New()
	for _, unit := range battleSpeeds() {
		if err := queue.Add(unit.id, unit.speed); err != nil {
			t.Fatalf("add %s: %v", unit.id, err)
		}
	}
	return queue
}

// TestTurnShareOverOneCycleMatchesSpeed is the check that the whole speed budget
// rests on: across a full cycle every unit takes as many turns as it has speed,
// so the 80 to 200 ceiling really is a two and a half times spread in tempo.
func TestTurnShareOverOneCycleMatchesSpeed(t *testing.T) {
	queue := battleQueue(t)
	for {
		preview := queue.Preview(1)
		if len(preview) == 0 || preview[0].At > atb.Scale {
			break
		}
		queue.Next()
	}
	for _, unit := range battleSpeeds() {
		turns := int64(queue.Turns(unit.id))
		if turns < unit.speed-1 || turns > unit.speed {
			t.Errorf("%s at speed %d took %d turns in a cycle, want about %d",
				unit.id, unit.speed, turns, unit.speed)
		}
	}
}

// TestASpeedBuffChangesTempoNotTheTurnAlreadyServed ties the queue's rule back
// to the buff layer: applying the shipped buff bounds to a unit's speed and
// rescheduling it moves its future turns without handing it a partial one.
func TestASpeedBuffChangesTempoNotTheTurnAlreadyServed(t *testing.T) {
	bounds, ceilings := mustBounds(t), mustCeilings(t)
	queue := battleQueue(t)
	// Run a while so the buffed unit is partway through a wait.
	for i := 0; i < 37; i++ {
		queue.Next()
	}
	const target = "ally.bulwark"
	base := slowProfile()
	buffed := buff(t, base, ceilings, bounds, modifier.Speed)

	pendingBefore := queue.Pending(target)
	turnsBefore := queue.Turns(target)
	if err := queue.Reschedule(target, buffed[progression.Speed]); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if got := queue.Turns(target); got != turnsBefore {
		t.Errorf("the buff changed the turn count from %d to %d", turnsBefore, got)
	}
	pendingAfter := queue.Pending(target)
	if pendingAfter >= pendingBefore {
		t.Errorf("after a speed buff there is %d pending, want less than %d", pendingAfter, pendingBefore)
	}
	// The served fraction is preserved, so the remaining wait shrinks by exactly
	// the ratio of the speeds.
	wantAfter := pendingBefore * base[progression.Speed] / buffed[progression.Speed]
	if pendingAfter != wantAfter {
		t.Errorf("there is %d pending, want %d, the old wait scaled by the speed ratio", pendingAfter, wantAfter)
	}
}

func writeTurnOrderScenario(b *strings.Builder, rules combat.Rules) {
	fmt.Fprintf(b, "== turn order, wait is %d over speed ==\n", atb.Scale)
	b.WriteString("five on five at the reference speeds, ties going to the faster unit\n")
	b.WriteString("speed   wait\n")
	shown := map[int64]bool{}
	for _, unit := range battleSpeeds() {
		if shown[unit.speed] {
			continue
		}
		shown[unit.speed] = true
		fmt.Fprintf(b, "%5d%7d\n", unit.speed, atb.Wait(unit.speed))
	}

	queue := atb.New()
	for _, unit := range battleSpeeds() {
		_ = queue.Add(unit.id, unit.speed)
	}
	b.WriteString("\nthe first twenty turns\n")
	b.WriteString("  #   at        unit               speed   its turn\n")
	for i := 1; i <= 20; i++ {
		turn, ok := queue.Next()
		if !ok {
			break
		}
		fmt.Fprintf(b, "%3d%7d   %-22s%5d%10d\n", i, turn.At, turn.ID, turn.Speed, turn.Number)
	}

	b.WriteString("\nturns taken across one full cycle\n")
	full := atb.New()
	for _, unit := range battleSpeeds() {
		_ = full.Add(unit.id, unit.speed)
	}
	for {
		preview := full.Preview(1)
		if len(preview) == 0 || preview[0].At > atb.Scale {
			break
		}
		full.Next()
	}
	b.WriteString("unit                     speed   turns\n")
	for _, unit := range battleSpeeds() {
		fmt.Fprintf(b, "%-24s%6d%8d\n", unit.id, unit.speed, full.Turns(unit.id))
	}

	b.WriteString("\na speed buff keeps the fraction of the wait already served\n")
	tempo := atb.New()
	for _, unit := range battleSpeeds() {
		_ = tempo.Add(unit.id, unit.speed)
	}
	for i := 0; i < 37; i++ {
		tempo.Next()
	}
	const target = "ally.bulwark"
	b.WriteString("                       speed   pending   next five turns\n")
	fmt.Fprintf(b, "%-22s%6d%10d   %s\n", "before", tempo.Speed(target), tempo.Pending(target), previewIDs(tempo, 5))
	_ = tempo.Reschedule(target, 117)
	fmt.Fprintf(b, "%-22s%6d%10d   %s\n", "after a +50% buff", tempo.Speed(target), tempo.Pending(target), previewIDs(tempo, 5))
	_ = tempo.Reschedule(target, 80)
	fmt.Fprintf(b, "%-22s%6d%10d   %s\n", "buff removed", tempo.Speed(target), tempo.Pending(target), previewIDs(tempo, 5))
	b.WriteString("removing the buff puts the pending value back, so alternating a buff and a debuff\n")
	b.WriteString("cannot stall a unit\n")

	b.WriteString("\na status ticks at the start of its holder's own turn\n")
	book, err := seed.StatusBook()
	if err != nil {
		fmt.Fprintf(b, "unavailable: %v\n\n", err)
		return
	}
	poison, err := book.Lookup("poison")
	if err != nil {
		fmt.Fprintf(b, "unavailable: %v\n\n", err)
		return
	}
	tick := rules.Damage(attackerAttack, referenceDefense, poisonPower, neutralAffinity)
	poisoned := atb.New()
	_ = poisoned.Add("ally.bulwark", 80)
	_ = poisoned.Add("foe.skirmisher", 200)
	var set status.Set
	set.Apply(poison, tick)
	set.Apply(poison, tick)
	set.Apply(poison, tick)
	b.WriteString("  #   at        unit               poison tick   stacks left\n")
	for i := 1; i <= 12; i++ {
		turn, ok := poisoned.Next()
		if !ok {
			break
		}
		damage := int64(0)
		if turn.ID == "ally.bulwark" {
			damage, _, _ = set.Tick()
		}
		fmt.Fprintf(b, "%3d%7d   %-22s%12d%14d\n", i, turn.At, turn.ID, damage, set.Stacks("poison"))
	}
	b.WriteString("the fast unit acts between the ticks but does not advance them, so the poison\n")
	b.WriteString("deals the same total however the tempo around it changes\n\n")
}

func previewIDs(queue *atb.Queue, count int) string {
	parts := make([]string, 0, count)
	for _, turn := range queue.Preview(count) {
		name := turn.ID
		if index := strings.IndexByte(name, '.'); index >= 0 {
			name = name[index+1:]
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, " ")
}
