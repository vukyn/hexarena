package modifier_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
)

func bounds() modifier.Bounds {
	return modifier.Bounds{Headroom: 3000, FloorFraction: 100, MaxAffinityScale: 1000}
}

func ceilings() progression.Values {
	return progression.Values{
		progression.HP: 4800, progression.Attack: 800, progression.Defense: 800,
		progression.Speed: 200, progression.Accuracy: 300, progression.Dodge: 150,
	}
}

func percent(target modifier.Target, amount int64) modifier.Modifier {
	return modifier.Modifier{Target: target, Mode: modifier.Percent, Amount: amount}
}

func flat(target modifier.Target, amount int64) modifier.Modifier {
	return modifier.Modifier{Target: target, Mode: modifier.Flat, Amount: amount}
}

func setOf(t *testing.T, modifiers ...modifier.Modifier) modifier.Set {
	t.Helper()
	var set modifier.Set
	if err := set.AddAll(modifiers...); err != nil {
		t.Fatalf("add modifiers: %v", err)
	}
	return set
}

func TestPercentTermsOfTheSameTargetAdd(t *testing.T) {
	set := setOf(t, percent(modifier.Attack, 500), percent(modifier.Attack, 500), percent(modifier.Attack, 500))
	if got, want := set.Percent(modifier.Attack), int64(1500); got != want {
		t.Errorf("three fifty percent buffs summed to %d, want %d", got, want)
	}
	// Adding gives a raw 2.5x; composing would have given 3.375x.
	if got, want := set.Raw(progression.Attack, 400), int64(1000); got != want {
		t.Errorf("the raw value is %d, want %d", got, want)
	}
}

func TestFlatAppliesBeforePercent(t *testing.T) {
	set := setOf(t, flat(modifier.Attack, 100), percent(modifier.Attack, 500))
	// (400 + 100) * 1.5 = 750, not 400 * 1.5 + 100 = 700.
	if got, want := set.Raw(progression.Attack, 400), int64(750); got != want {
		t.Errorf("the raw value is %d, want %d", got, want)
	}
	if got, want := set.Stat(progression.Attack, 400, 800, bounds()), int64(697); got != want {
		t.Errorf("the saturated value is %d, want %d", got, want)
	}
}

// TestBuffsSaturateTowardsTheHeadroomLimit is the behaviour the design asks for:
// each further buff is worth less than the last, and no stack of them reaches
// the limit.
func TestBuffsSaturateTowardsTheHeadroomLimit(t *testing.T) {
	b := bounds()
	const (
		base    = 400
		ceiling = 800
	)
	limit := ceiling * b.Headroom / modifier.PercentBase
	cases := []struct {
		name string
		term int64
		want int64
	}{
		{"nothing", 0, base},
		{"+50%", 500, 581},
		{"+150%", 1500, 861},
		{"+500%", 5000, 1400},
		{"+10000%", 100000, 2304},
	}
	previousGain := int64(1 << 40)
	for _, testCase := range cases {
		var set modifier.Set
		if testCase.term != 0 {
			set = setOf(t, percent(modifier.Attack, testCase.term))
		}
		got := set.Stat(progression.Attack, base, ceiling, b)
		if got != testCase.want {
			t.Errorf("%s gave %d, want %d", testCase.name, got, testCase.want)
		}
		if got >= limit {
			t.Errorf("%s reached %d, the limit of %d must never be touched", testCase.name, got, limit)
		}
		if gain := got - base; gain > 0 && testCase.term > 0 {
			perTerm := gain * modifier.PercentBase / testCase.term
			if perTerm > previousGain {
				t.Errorf("%s returned %d per mille of its term, more than the previous %d",
					testCase.name, perTerm, previousGain)
			}
			previousGain = perTerm
		}
	}
}

// TestDebuffsSaturateTowardsTheFloor is the same on the other side: a stat can
// be crushed but never removed, so a debuff can safely be authored well past a
// hundred percent.
func TestDebuffsSaturateTowardsTheFloor(t *testing.T) {
	b := bounds()
	const base = 800
	floor := base * b.FloorFraction / modifier.PercentBase
	cases := []struct {
		name string
		term int64
		want int64
	}{
		{"-30%", -300, 620},
		{"-60%", -600, 512},
		{"-75%", -750, 473},
		{"-300%", -3000, 247},
		{"-10000%", -100000, 87},
	}
	for _, testCase := range cases {
		set := setOf(t, percent(modifier.Defense, testCase.term))
		got := set.Stat(progression.Defense, base, 800, b)
		if got != testCase.want {
			t.Errorf("%s gave %d, want %d", testCase.name, got, testCase.want)
		}
		if got <= floor {
			t.Errorf("%s reached %d, the floor of %d must never be touched", testCase.name, got, floor)
		}
	}
}

func TestStatNeverDropsBelowOne(t *testing.T) {
	set := setOf(t, flat(modifier.Attack, -5000))
	if got := set.Stat(progression.Attack, 3, 800, bounds()); got < 1 {
		t.Errorf("a crushed stat is %d, want at least 1", got)
	}
}

func TestStatsResolvesEveryStatIndependently(t *testing.T) {
	set := setOf(t, percent(modifier.Speed, 500), percent(modifier.Defense, -300))
	base := progression.Values{
		progression.HP: 3600, progression.Attack: 620, progression.Defense: 560,
		progression.Speed: 110, progression.Accuracy: 60, progression.Dodge: 30,
	}
	want := progression.Values{
		progression.HP: 3600, progression.Attack: 620, progression.Defense: 434,
		progression.Speed: 159, progression.Accuracy: 60, progression.Dodge: 30,
	}
	if got := set.Stats(base, ceilings(), bounds()); got != want {
		t.Errorf("resolved %s, want %s", got, want)
	}
}

// TestAffinityIsWorthMoreAgainstAWorseMatchup is what the multiplicative form
// buys. The same term adds more, in both absolute and relative terms, against a
// stacked weakness than against a single one; a flat term would do the reverse.
func TestAffinityIsWorthMoreAgainstAWorseMatchup(t *testing.T) {
	const neutral = 1000
	b := bounds()
	buff := setOf(t, percent(modifier.Affinity, 300))
	single := buff.Affinity(1500, neutral, b)
	stacked := buff.Affinity(2250, neutral, b)
	if single != 1650 {
		t.Errorf("a single weakness became %d, want 1650", single)
	}
	if stacked != 2625 {
		t.Errorf("a stacked weakness became %d, want 2625", stacked)
	}
	singleGain := single * 1000 / 1500
	stackedGain := stacked * 1000 / 2250
	if stackedGain <= singleGain {
		t.Errorf("the term is worth %d per mille against a stacked weakness and %d against a single one, want more against the stacked one",
			stackedGain, singleGain)
	}
}

func TestAffinityLeavesANeutralOrResistedMatchupAlone(t *testing.T) {
	const neutral = 1000
	b := bounds()
	buff := setOf(t, percent(modifier.Affinity, 300))
	for _, multiplier := range []int{neutral, 667, 444} {
		if got := buff.Affinity(multiplier, neutral, b); got != multiplier {
			t.Errorf("a multiplier of %d became %d, want it untouched", multiplier, got)
		}
	}
}

func TestAffinityDebuffStripsButDoesNotInvert(t *testing.T) {
	const neutral = 1000
	b := bounds()
	stripped := setOf(t, percent(modifier.Affinity, -1000))
	for _, multiplier := range []int{1500, 2250} {
		if got := stripped.Affinity(multiplier, neutral, b); got != neutral {
			t.Errorf("a stripped %d became %d, want the neutral %d", multiplier, got, neutral)
		}
	}
	crushed := setOf(t, percent(modifier.Affinity, -100000))
	if got := crushed.Affinity(2250, neutral, b); got != neutral {
		t.Errorf("an over-stripped advantage became %d, want the neutral %d", got, neutral)
	}
}

func TestAffinityScaleIsClamped(t *testing.T) {
	const neutral = 1000
	b := bounds()
	stacked := setOf(t, percent(modifier.Affinity, 800), percent(modifier.Affinity, 800))
	if got, want := stacked.Percent(modifier.Affinity), int64(1600); got != want {
		t.Errorf("the raw affinity sum is %d, want %d", got, want)
	}
	// Clamped to +100%, so the deviation exactly doubles.
	if got, want := stacked.Affinity(1500, neutral, b), 2000; got != want {
		t.Errorf("the clamped result is %d, want %d", got, want)
	}
}

func TestModifierValidateRejects(t *testing.T) {
	cases := []struct {
		name     string
		modifier modifier.Modifier
		wantErr  string
	}{
		{"an undeclared target", flat(modifier.Target(99), 10), "not declared"},
		{"a flat affinity term", flat(modifier.Affinity, 300), "must be a percentage"},
		{"a term with no amount", percent(modifier.Attack, 0), "no amount"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.modifier.Validate()
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
			var set modifier.Set
			if err := set.Add(testCase.modifier); err == nil {
				t.Error("Add accepted an invalid modifier")
			}
		})
	}
}

func TestModifierJSONRoundTrip(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`{"target":"attack","mode":"percent","amount":500}`, "attack +50.0%"},
		{`{"target":"defense","mode":"percent","amount":-300}`, "defense -30.0%"},
		{`{"target":"speed","mode":"flat","amount":40}`, "speed +40"},
		{`{"target":"accuracy","mode":"flat","amount":25}`, "accuracy +25"},
		{`{"target":"dodge","mode":"percent","amount":400}`, "dodge +40.0%"},
		{`{"target":"affinity","mode":"percent","amount":300}`, "affinity +30.0%"},
	}
	for _, testCase := range cases {
		var decoded modifier.Modifier
		if err := json.Unmarshal([]byte(testCase.raw), &decoded); err != nil {
			t.Errorf("unmarshal %s: %v", testCase.raw, err)
			continue
		}
		if got := decoded.String(); got != testCase.want {
			t.Errorf("%s renders as %q, want %q", testCase.raw, got, testCase.want)
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			t.Errorf("marshal: %v", err)
			continue
		}
		var again modifier.Modifier
		if err := json.Unmarshal(encoded, &again); err != nil {
			t.Errorf("re-unmarshal %s: %v", encoded, err)
			continue
		}
		if again != decoded {
			t.Errorf("round trip of %v gave %v", decoded, again)
		}
	}
}

func TestModifierJSONRejects(t *testing.T) {
	for _, raw := range []string{
		`{"target":"luck","mode":"percent","amount":100}`,
		`{"target":"attack","mode":"multiply","amount":100}`,
		`{"target":"affinity","mode":"flat","amount":100}`,
		`{"target":"attack","mode":"percent","amount":0}`,
		`"attack"`,
	} {
		var decoded modifier.Modifier
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			t.Errorf("%s was accepted, want a rejection", raw)
		}
	}
}

func TestBoundsValidateRejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*modifier.Bounds)
		wantErr string
	}{
		{"headroom with no room", func(b *modifier.Bounds) { b.Headroom = modifier.PercentBase }, "somewhere to go"},
		{"no floor", func(b *modifier.Bounds) { b.FloorFraction = 0 }, "want a positive value"},
		{"a floor at the base itself", func(b *modifier.Bounds) { b.FloorFraction = modifier.PercentBase }, "somewhere to go"},
		{"a negative affinity scale", func(b *modifier.Bounds) { b.MaxAffinityScale = -1 }, "max_affinity_scale"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			b := bounds()
			testCase.mutate(&b)
			err := b.Validate()
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
	if _, err := modifier.ParseBounds([]byte("{")); err == nil {
		t.Error("malformed JSON was accepted")
	}
}

func TestTargetNamesLineUpWithStats(t *testing.T) {
	// The stat targets share numbering with progression.Kind, because a Set
	// indexes one array by both.
	for _, kind := range progression.Kinds() {
		target := modifier.Target(kind)
		if !target.IsStat() {
			t.Errorf("%s is a stat but its target does not report as one", kind)
		}
		if target.Stat() != kind {
			t.Errorf("the target for %s resolves back to %s", kind, target.Stat())
		}
		if target.String() != kind.String() {
			t.Errorf("target %q and stat %q disagree on their name", target, kind)
		}
	}
	if modifier.Affinity.IsStat() {
		t.Error("affinity reports itself as a stat")
	}
	if got, want := modifier.TargetCount, progression.KindCount+1; got != want {
		t.Errorf("there are %d targets, want the %d stats plus affinity", got, want)
	}
	for _, target := range modifier.Targets() {
		parsed, err := modifier.ParseTarget(target.String())
		if err != nil || parsed != target {
			t.Errorf("ParseTarget(%q) gave %v, %v", target, parsed, err)
		}
	}
	if _, err := modifier.ParseTarget("luck"); err == nil {
		t.Error("an unknown target name was accepted")
	}
	if _, err := modifier.ParseMode("multiply"); err == nil {
		t.Error("an unknown mode name was accepted")
	}
}

func TestZeroSetChangesNothing(t *testing.T) {
	var set modifier.Set
	base := progression.Values{
		progression.HP: 3600, progression.Attack: 620, progression.Defense: 560,
		progression.Speed: 110, progression.Accuracy: 60, progression.Dodge: 30,
	}
	if got := set.Stats(base, ceilings(), bounds()); got != base {
		t.Errorf("an empty set resolved %s to %s", base, got)
	}
	if got := set.Affinity(1500, 1000, bounds()); got != 1500 {
		t.Errorf("an empty set changed the affinity multiplier to %d", got)
	}
}
