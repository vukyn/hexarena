package combat_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/rng"
)

func rules() combat.Rules {
	return combat.Rules{DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250, MinHitChance: 150, MaxBlockCharges: 3}
}

// TestDefenseCurveAnchors pins the three points that define the curve's shape.
// If any of them moves, every stat budget downstream is invalid.
func TestDefenseCurveAnchors(t *testing.T) {
	r := rules()
	const (
		attack = 800
		skill  = 1800
		plain  = combat.PermilleBase
	)
	cases := []struct {
		name    string
		defense int64
		want    int64
	}{
		{"no defence takes the full skill multiple", 0, attack * skill / plain},
		{"defence equal to K halves the damage", 300, attack * skill / plain / 2},
		{"defence at the level 60 ceiling", 800, 392},
		{"defence far past the ceiling still lets damage through", 5000, 81},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := r.Damage(attack, testCase.defense, skill, plain); got != testCase.want {
				t.Errorf("Damage(attack %d, defence %d) = %d, want %d",
					attack, testCase.defense, got, testCase.want)
			}
		})
	}
}

// TestAffinitySwingAtTheLevelCap is the sanity check behind the HP budget: at
// the level 60 ceiling of 800 attack against 800 defence, a stacked weakness
// has to be a serious hit without being an instant kill.
func TestAffinitySwingAtTheLevelCap(t *testing.T) {
	r := rules()
	cases := []struct {
		name     string
		affinity int
		want     int64
	}{
		{"both elements resist", 444, 174},
		{"neutral matchup", 1000, 392},
		{"single weakness", 1500, 589},
		{"stacked weakness", 2250, 883},
	}
	for _, testCase := range cases {
		got := r.Damage(800, 800, 1800, testCase.affinity)
		if got != testCase.want {
			t.Errorf("%s: damage %d, want %d", testCase.name, got, testCase.want)
		}
	}
}

func TestDamageFloorsAtMinimumButNotBelowZeroAttack(t *testing.T) {
	r := rules()
	if got := r.Damage(1, 100000, 1, 444); got != r.MinimumDamage {
		t.Errorf("a hit that rounds to nothing deals %d, want the floor %d", got, r.MinimumDamage)
	}
	for _, testCase := range []struct {
		name                                string
		attack                              int64
		skillMultiplier, affinityMultiplier int
	}{
		{"no attack", 0, 1800, 1000},
		{"negative attack", -50, 1800, 1000},
		{"no skill multiplier", 800, 0, 1000},
		{"no affinity multiplier", 800, 1800, 0},
	} {
		if got := r.Damage(testCase.attack, 100, testCase.skillMultiplier, testCase.affinityMultiplier); got != 0 {
			t.Errorf("%s: damage %d, want 0", testCase.name, got)
		}
	}
}

func TestNegativeDefenseIsTreatedAsNone(t *testing.T) {
	r := rules()
	if got, want := r.Damage(800, -500, 1800, 1000), r.Damage(800, 0, 1800, 1000); got != want {
		t.Errorf("negative defence dealt %d, want the same as no defence %d", got, want)
	}
	if got, want := r.DefenseReduction(-500), r.DefenseReduction(0); got != want {
		t.Errorf("negative defence reduction %d, want %d", got, want)
	}
}

func TestDefenseReductionMatchesTheCurve(t *testing.T) {
	r := rules()
	cases := []struct {
		defense int64
		want    int
	}{
		{0, 1000},
		{300, 500},
		{800, 272},
		{2700, 100},
	}
	for _, testCase := range cases {
		if got := r.DefenseReduction(testCase.defense); got != testCase.want {
			t.Errorf("DefenseReduction(%d) = %d, want %d", testCase.defense, got, testCase.want)
		}
	}
}

func TestDamageIsMonotonic(t *testing.T) {
	r := rules()
	previous := int64(1 << 62)
	for defense := int64(0); defense <= 2000; defense += 25 {
		got := r.Damage(800, defense, 1800, 1000)
		if got > previous {
			t.Fatalf("damage rose from %d to %d as defence reached %d", previous, got, defense)
		}
		previous = got
	}
	previous = 0
	for attack := int64(0); attack <= 2000; attack += 25 {
		got := r.Damage(attack, 400, 1800, 1000)
		if got < previous {
			t.Fatalf("damage fell from %d to %d as attack reached %d", previous, got, attack)
		}
		previous = got
	}
}

func TestParseRulesRejects(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"malformed json", "{", "decode combat rules"},
		{"zero defence constant", `{"defense_constant":0,"minimum_damage":1,"critical_multiplier":1250,"min_hit_chance":150,"max_block_charges":3}`, "want a positive value"},
		{"negative defence constant", `{"defense_constant":-300,"minimum_damage":1,"critical_multiplier":1250,"min_hit_chance":150,"max_block_charges":3}`, "want a positive value"},
		{"negative minimum damage", `{"defense_constant":300,"minimum_damage":-1,"critical_multiplier":1250,"min_hit_chance":150,"max_block_charges":3}`, "want zero or more"},
		{"no hit chance floor", `{"defense_constant":300,"minimum_damage":1,"critical_multiplier":1250,"min_hit_chance":0,"max_block_charges":3}`, "min_hit_chance"},
		{"a hit chance floor at certainty", `{"defense_constant":300,"minimum_damage":1,"critical_multiplier":1250,"min_hit_chance":1000,"max_block_charges":3}`, "somewhere to go"},
		{"no block charges allowed", `{"defense_constant":300,"minimum_damage":1,"critical_multiplier":1250,"min_hit_chance":150,"max_block_charges":0}`, "max_block_charges"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := combat.ParseRules([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestNoOverflowAtAbsurdInputs(t *testing.T) {
	r := rules()
	// Well past any reachable stat, to prove the intermediate product stays
	// inside int64 rather than wrapping negative.
	if got := r.Damage(1_000_000, 1_000_000, 10_000, 2250); got <= 0 {
		t.Errorf("damage at absurd inputs is %d, want a positive value", got)
	}
}

func TestHitChanceSaturatesTowardsCertainty(t *testing.T) {
	cases := []struct {
		name          string
		skillAccuracy int
		accuracyStat  int64
		want          int
	}{
		{"a reliable skill with no accuracy stat", 900, 0, 900},
		{"a reliable skill with some accuracy", 900, 100, 950},
		{"a reliable skill at the accuracy ceiling", 900, 300, 975},
		{"an unreliable skill with some accuracy", 700, 100, 775},
		{"an unreliable skill at the ceiling", 700, 300, 850},
		{"a skill that cannot miss", 1000, 0, 1000},
		{"a skill that cannot miss, unaffected by accuracy", 1000, 300, 1000},
		{"a negative accuracy stat is ignored", 900, -500, 900},
		{"a negative skill accuracy is treated as none", -100, 300, 230},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			hit := combat.Hit{SkillAccuracy: testCase.skillAccuracy, AccuracyStat: testCase.accuracyStat}
			if got := rules().Chance(hit); got != testCase.want {
				t.Errorf("chance is %d, want %d", got, testCase.want)
			}
		})
	}
}

// TestAccuracyNeverGuaranteesAnImperfectSkill is the point of saturating rather
// than clamping: stacking accuracy closes the gap to a certain hit without ever
// closing it, so an unreliable skill stays unreliable.
func TestAccuracyNeverGuaranteesAnImperfectSkill(t *testing.T) {
	for _, skillAccuracy := range []int{500, 700, 900, 990, 999} {
		for _, stat := range []int64{300, 3_000, 3_000_000, 1 << 40} {
			hit := combat.Hit{SkillAccuracy: skillAccuracy, AccuracyStat: stat}
			if got := rules().Chance(hit); got >= combat.PermilleBase {
				t.Errorf("a %d skill with %d accuracy reached %d, want short of certainty",
					skillAccuracy, stat, got)
			}
		}
	}
}

func TestAccuracyDiminishes(t *testing.T) {
	previous := 0
	previousGain := 1 << 30
	for stat := int64(0); stat <= 1200; stat += 100 {
		chance := rules().Chance(combat.Hit{SkillAccuracy: 700, AccuracyStat: stat})
		if chance < previous {
			t.Fatalf("chance fell from %d to %d at accuracy %d", previous, chance, stat)
		}
		if gain := chance - previous; stat > 0 {
			if gain > previousGain {
				t.Fatalf("the step at accuracy %d gained %d, more than the previous %d", stat, gain, previousGain)
			}
			previousGain = gain
		}
		previous = chance
	}
}

func TestRollLandsAtRoughlyTheStatedRate(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 1800, Strikes: 1, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: 800, AccuracyStat: 0,
	}
	source := rng.New(0x5EED)
	const draws = 100_000
	landed := 0
	for i := 0; i < draws; i++ {
		if attempts, _ := r.Roll(hit, 0, source); combat.DamageDealt(attempts) > 0 {
			landed++
		}
	}
	rate := landed * 1000 / draws
	if rate < 790 || rate > 810 {
		t.Errorf("an 800 per mille skill landed %d per mille of the time", rate)
	}
}

// TestMultiStrikeRollsEachStrikeSeparately is what gives a multi-strike skill a
// different shape from a single one at the same accuracy: the same expected
// damage, far less variance, and partial outcomes.
func TestMultiStrikeRollsEachStrikeSeparately(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: 700,
	}
	source := rng.New(11)
	outcomes := make(map[int]int)
	const draws = 20_000
	for i := 0; i < draws; i++ {
		attempts, _ := r.Roll(hit, 0, source)
		if len(attempts) != 3 {
			t.Fatalf("a three strike skill produced %d attempts", len(attempts))
		}
		outcomes[combat.Count(attempts, combat.Struck)]++
	}
	for count := 0; count <= 3; count++ {
		if outcomes[count] == 0 {
			t.Errorf("%d strikes landing never happened in %d draws", count, draws)
		}
	}
	// A whole miss is the cube of a single miss, so it should be rare.
	if share := outcomes[0] * 1000 / draws; share > 40 {
		t.Errorf("every strike missed %d per mille of the time, want about 27", share)
	}
}

func TestRollIsReproducible(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 4, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: 650,
	}
	first, second := rng.New(2024), rng.New(2024)
	for i := 0; i < 200; i++ {
		a, _ := r.Roll(hit, 0, first)
		b, _ := r.Roll(hit, 0, second)
		if len(a) != len(b) {
			t.Fatalf("draw %d produced %d and %d attempts", i, len(a), len(b))
		}
		for j := range a {
			if a[j] != b[j] {
				t.Fatalf("draw %d strike %d differs: %+v against %+v", i, j, a[j], b[j])
			}
		}
	}
}

func TestDamageDealtSumsOnlyTheStrikesThatStruck(t *testing.T) {
	attempts := []combat.Attempt{
		{Outcome: combat.Struck, Damage: 100},
		{Outcome: combat.Missed},
		{Outcome: combat.Blocked},
		{Outcome: combat.Struck, Damage: 250},
	}
	if got, want := combat.DamageDealt(attempts), int64(350); got != want {
		t.Errorf("DamageDealt is %d, want %d", got, want)
	}
	if got := combat.DamageDealt(nil); got != 0 {
		t.Errorf("DamageDealt of nothing is %d, want 0", got)
	}
	for outcome, want := range map[combat.Outcome]int{combat.Struck: 2, combat.Missed: 1, combat.Blocked: 1} {
		if got := combat.Count(attempts, outcome); got != want {
			t.Errorf("%s came up %d times, want %d", outcome, got, want)
		}
	}
}

func TestPickScaling(t *testing.T) {
	if got, want := combat.PickScaling(combat.BaseStat, 80, 120), int64(80); got != want {
		t.Errorf("BaseStat gave %d, want %d", got, want)
	}
	if got, want := combat.PickScaling(combat.CurrentStat, 80, 120), int64(120); got != want {
		t.Errorf("CurrentStat gave %d, want %d", got, want)
	}
	if got, want := combat.BaseStat.String(), "base"; got != want {
		t.Errorf("BaseStat renders as %q, want %q", got, want)
	}
	if got, want := combat.CurrentStat.String(), "current"; got != want {
		t.Errorf("CurrentStat renders as %q, want %q", got, want)
	}
}

// TestDodgeBitesEvenAgainstNearCertainAccuracy is the behaviour dodge exists
// for. If dodge were subtracted from accuracy instead of applied after it, a
// defender's dodge would cancel against a high-accuracy attacker and do nothing
// at all, which is exactly the abuse it is meant to answer.
func TestDodgeBitesEvenAgainstNearCertainAccuracy(t *testing.T) {
	r := rules()
	nearCertain := combat.Hit{SkillAccuracy: 950, AccuracyStat: 300}
	if got := r.Chance(nearCertain); got != 992 {
		t.Fatalf("the undodged chance is %d, want 992", got)
	}
	cases := []struct {
		dodge int64
		want  int
	}{
		{0, 992},
		{50, 945},
		{100, 903},
		{150, 865},
	}
	for _, testCase := range cases {
		hit := nearCertain
		hit.DodgeStat = testCase.dodge
		if got := r.Chance(hit); got != testCase.want {
			t.Errorf("%d dodge against a 992 chance gave %d, want %d", testCase.dodge, got, testCase.want)
		}
	}
}

func TestDodgeNeverMakesAUnitUntouchable(t *testing.T) {
	r := rules()
	for _, accuracy := range []int{300, 600, 850, 999} {
		for _, dodge := range []int64{150, 1_500, 1_500_000, 1 << 40} {
			hit := combat.Hit{SkillAccuracy: accuracy, DodgeStat: dodge}
			got := r.Chance(hit)
			if got <= r.MinHitChance {
				t.Errorf("a %d skill against %d dodge gave %d, the floor of %d must not be reached",
					accuracy, dodge, got, r.MinHitChance)
			}
			if got > accuracy {
				t.Errorf("a %d skill against %d dodge gave %d, dodge must not raise the chance",
					accuracy, dodge, got)
			}
		}
	}
}

// TestAGuaranteedSkillIgnoresDodge is what leaves block a job to do: if dodge
// could stop a guaranteed hit, nothing would be guaranteed and block would only
// duplicate dodge.
func TestAGuaranteedSkillIgnoresDodge(t *testing.T) {
	r := rules()
	for _, dodge := range []int64{0, 150, 1 << 30} {
		hit := combat.Hit{SkillAccuracy: combat.PermilleBase, DodgeStat: dodge}
		if got := r.Chance(hit); got != combat.PermilleBase {
			t.Errorf("a guaranteed skill against %d dodge gave %d, want certainty", dodge, got)
		}
	}
}

func TestAccuracyStillHelpsAgainstADodgyTarget(t *testing.T) {
	r := rules()
	base := combat.Hit{SkillAccuracy: 700, DodgeStat: 150}
	previous := 0
	for _, stat := range []int64{0, 75, 150, 225, 300} {
		hit := base
		hit.AccuracyStat = stat
		got := r.Chance(hit)
		if got <= previous {
			t.Errorf("accuracy %d gave %d, no better than the previous %d", stat, got, previous)
		}
		previous = got
	}
}

func TestDodgeIsIgnoredWhenTheChanceIsAlreadyAtTheFloor(t *testing.T) {
	r := rules()
	// A skill this unreliable is already below the floor, so dodge has nothing
	// left to take.
	hit := combat.Hit{SkillAccuracy: 100, DodgeStat: 150}
	if got, want := r.Chance(hit), 100; got != want {
		t.Errorf("chance is %d, want %d untouched", got, want)
	}
	if got := r.Chance(combat.Hit{SkillAccuracy: 700, DodgeStat: -500}); got != 700 {
		t.Errorf("a negative dodge changed the chance to %d", got)
	}
}

// TestABlockChargeCancelsOneStrike is the counterplay to a guaranteed hit, and
// the reason a multi-strike skill beats block: one charge erases a full-power
// blow but is burned by a third of one.
func TestABlockChargeCancelsOneStrike(t *testing.T) {
	r := rules()
	guaranteed := combat.Hit{
		Scaling: 800, Multiplier: 1800, Strikes: 1, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: combat.PermilleBase, DodgeStat: 150,
	}
	source := rng.New(1)
	attempts, left := r.Roll(guaranteed, 1, source)
	if len(attempts) != 1 || attempts[0].Outcome != combat.Blocked {
		t.Errorf("a blocked guaranteed hit resolved to %+v", attempts)
	}
	if left != 0 {
		t.Errorf("%d charges left, want the charge spent", left)
	}
	if got := combat.DamageDealt(attempts); got != 0 {
		t.Errorf("a blocked hit dealt %d, want nothing", got)
	}

	split := guaranteed
	split.Strikes, split.Multiplier = 3, 600
	attempts, left = r.Roll(split, 1, source)
	if got, want := combat.Count(attempts, combat.Blocked), 1; got != want {
		t.Errorf("%d of three strikes were blocked, want %d", got, want)
	}
	if got, want := combat.Count(attempts, combat.Struck), 2; got != want {
		t.Errorf("%d of three strikes struck, want %d", got, want)
	}
	if left != 0 {
		t.Errorf("%d charges left, want the single charge spent", left)
	}
	if got, want := combat.DamageDealt(attempts), r.Strike(split)*2; got != want {
		t.Errorf("a partly blocked multi strike dealt %d, want %d", got, want)
	}
}

func TestBlockChargesAreSpentOneAtATime(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 400, Strikes: 5, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: combat.PermilleBase,
	}
	source := rng.New(2)
	for _, testCase := range []struct{ charges, blocked, struck, left int }{
		{0, 0, 5, 0},
		{2, 2, 3, 0},
		{5, 5, 0, 0},
		{8, 5, 0, 3},
	} {
		attempts, left := r.Roll(hit, testCase.charges, source)
		if got := combat.Count(attempts, combat.Blocked); got != testCase.blocked {
			t.Errorf("%d charges blocked %d strikes, want %d", testCase.charges, got, testCase.blocked)
		}
		if got := combat.Count(attempts, combat.Struck); got != testCase.struck {
			t.Errorf("%d charges left %d strikes landing, want %d", testCase.charges, got, testCase.struck)
		}
		if left != testCase.left {
			t.Errorf("%d charges left %d over, want %d", testCase.charges, left, testCase.left)
		}
	}
}

// TestADodgedStrikeDoesNotSpendACharge keeps the two defences from competing for
// the same resource: a target that evades an attack should still be holding its
// charge when the next one arrives.
func TestADodgedStrikeDoesNotSpendACharge(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: 400, DodgeStat: 150,
	}
	source := rng.New(3)
	spentOnMisses := 0
	for i := 0; i < 5_000; i++ {
		attempts, left := r.Roll(hit, 3, source)
		spent := 3 - left
		if spent != combat.Count(attempts, combat.Blocked) {
			spentOnMisses++
		}
	}
	if spentOnMisses != 0 {
		t.Errorf("%d of 5000 rolls spent a charge on something other than a blocked strike", spentOnMisses)
	}
}

// TestGrantBlocksCapsAndReportsWaste keeps a stack of shielding effects from
// quietly pushing a unit past the cap, and lets the effect that overshoots know
// it did nothing.
func TestGrantBlocksCapsAndReportsWaste(t *testing.T) {
	r := rules()
	cases := []struct {
		held, granted, total, wasted int
	}{
		{0, 1, 1, 0},
		{0, 3, 3, 0},
		{0, 5, 3, 2},
		{2, 1, 3, 0},
		{2, 3, 3, 2},
		{3, 1, 3, 1},
		{3, 0, 3, 0},
		{-4, 2, 2, 0},
		{1, -2, 1, 0},
	}
	for _, testCase := range cases {
		total, wasted := r.GrantBlocks(testCase.held, testCase.granted)
		if total != testCase.total || wasted != testCase.wasted {
			t.Errorf("granting %d to %d held gave %d held and %d wasted, want %d and %d",
				testCase.granted, testCase.held, total, wasted, testCase.total, testCase.wasted)
		}
		if total > r.MaxBlockCharges {
			t.Errorf("granting %d to %d held exceeded the cap of %d", testCase.granted, testCase.held, r.MaxBlockCharges)
		}
	}
}

// TestChargesStackWithoutDiminishing records the property the cap exists for:
// unlike every other defence, each charge is worth exactly as much as the first.
func TestChargesStackWithoutDiminishing(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 1800, Strikes: 1, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: combat.PermilleBase,
	}
	source := rng.New(9)
	perStrike := r.Strike(hit)
	held := r.MaxBlockCharges
	for incoming := 0; incoming < r.MaxBlockCharges; incoming++ {
		attempts, left := r.Roll(hit, held, source)
		if got := combat.DamageDealt(attempts); got != 0 {
			t.Errorf("incoming attack %d dealt %d while charges remained", incoming, got)
		}
		if left != held-1 {
			t.Errorf("incoming attack %d left %d charges, want %d", incoming, left, held-1)
		}
		held = left
	}
	// The charges are gone, so the next attack lands in full.
	attempts, left := r.Roll(hit, held, source)
	if got := combat.DamageDealt(attempts); got != perStrike {
		t.Errorf("the attack after the charges ran out dealt %d, want %d", got, perStrike)
	}
	if left != 0 {
		t.Errorf("%d charges left, want none", left)
	}
}

// TestPiercingEndpointsAreExact pins the two ends of the ratio. Nought has to
// leave the defence untouched or every skill in the game changes, and a full
// thousand has to reach nought rather than one, or "ignores the armour" would be
// a near miss that still divides by something.
func TestPiercingEndpointsAreExact(t *testing.T) {
	for _, defense := range []int64{0, 1, 137, 400, 800, 100_000} {
		if got := combat.Pierced(defense, 0); got != defense {
			t.Errorf("Pierced(%d, 0) = %d, want the defence untouched", defense, got)
		}
		if got := combat.Pierced(defense, 1000); got != 0 {
			t.Errorf("Pierced(%d, 1000) = %d, want no defence left", defense, got)
		}
	}
	// Out of range in either direction is the nearer end rather than an error:
	// the bound belongs to skill validation, and this has to stay total.
	if got := combat.Pierced(400, -50); got != 400 {
		t.Errorf("Pierced(400, -50) = %d, want the defence untouched", got)
	}
	if got := combat.Pierced(400, 4000); got != 0 {
		t.Errorf("Pierced(400, 4000) = %d, want no defence left", got)
	}
}

// TestPiercingNeverRaisesTheDefenceItTakesFrom is the invariant a truncation
// could break: every step of the ratio has to leave no more armour standing
// than the step before it, and never more than there was to begin with.
func TestPiercingNeverRaisesTheDefenceItTakesFrom(t *testing.T) {
	const defense = 743
	previous := int64(defense)
	for pierce := 0; pierce <= 1000; pierce++ {
		got := combat.Pierced(defense, pierce)
		if got > previous {
			t.Fatalf("armour rose from %d to %d at %d pierced", previous, got, pierce)
		}
		if got > defense {
			t.Fatalf("piercing %d left %d of %d standing", pierce, got, defense)
		}
		previous = got
	}
}

// TestPiercingIsADialRatherThanASwitch is the reason the field is a ratio, in
// the numbers the design was chosen from.
//
// The two units sit within two percent of each other on the joint
// health-and-defence budget, which is exactly why armour needed an answer: they
// are equally durable against everything else in the game. A ratio walks the
// armoured one's advantage across the range in steps an author can aim at. A
// switch would offer only the last row, so an armour unit would be worthless
// against one skill and untouched by the next, with nothing in between — the
// same shape this engine rejects when it saturates a buff instead of clamping it.
func TestPiercingIsADialRatherThanASwitch(t *testing.T) {
	r := rules()
	// Health and defence of a sentinel and a bulwark at the cap.
	const (
		sentinelHealth, sentinelDefense = int64(3100), int64(800)
		bulwarkHealth, bulwarkDefense   = int64(4800), int64(400)
	)
	absorbs := func(health, defense int64, pierce int) int64 {
		reduction := int64(r.DefenseReduction(combat.Pierced(defense, pierce)))
		return health * 1000 / reduction
	}
	cases := []struct {
		pierce                  int
		sentinel, bulwark, edge int64
	}{
		{0, 11397, 11214, 983},
		{200, 9717, 9937, 1022},
		{400, 8072, 8648, 1071},
		{600, 6418, 7361, 1146},
		{1000, 3100, 4800, 1548},
	}
	for _, testCase := range cases {
		sentinel := absorbs(sentinelHealth, sentinelDefense, testCase.pierce)
		bulwark := absorbs(bulwarkHealth, bulwarkDefense, testCase.pierce)
		if sentinel != testCase.sentinel || bulwark != testCase.bulwark {
			t.Errorf("at %d pierced the two absorb %d and %d, want %d and %d",
				testCase.pierce, sentinel, bulwark, testCase.sentinel, testCase.bulwark)
		}
		// The edge in parts per thousand rather than as a ratio, because this
		// package deals in integers and a float here would be the only one in
		// the engine. 983 is the armoured unit slightly behind; 1548 is it half
		// again as durable.
		if edge := bulwark * 1000 / sentinel; edge != testCase.edge {
			t.Errorf("at %d pierced the bulwark's edge is %d per thousand, want %d",
				testCase.pierce, edge, testCase.edge)
		}
	}
}

// TestAPiercingStrikeHitsHarderThanAPlainOne checks the figure reaches the
// formula through Hit, which is the path a battle takes. Damage itself is
// deliberately unaware of piercing — see its comment.
func TestAPiercingStrikeHitsHarderThanAPlainOne(t *testing.T) {
	r := rules()
	base := combat.Hit{
		Scaling: 800, Multiplier: 1000, Affinity: 1000, Defense: 400, SkillAccuracy: 1000,
	}
	plain := r.Strike(base)
	pierced := base
	pierced.Pierce = 600
	got := r.Strike(pierced)
	if got <= plain {
		t.Fatalf("piercing 600 dealt %d, no more than the %d a plain strike dealt", got, plain)
	}
	// It has to be the same figure as the plain strike against the armour that
	// piercing leaves standing, or piercing is a second damage formula.
	want := r.Damage(800, combat.Pierced(400, 600), 1000, 1000)
	if got != want {
		t.Fatalf("piercing 600 dealt %d, want the %d a plain strike deals against %d defence",
			got, want, combat.Pierced(400, 600))
	}
}
