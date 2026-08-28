package combat_test

import (
	"math"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/rng"
)

// closedForm is the damage expression written out by hand, once, so every test
// below is checked against something other than the code it is testing.
//
// It is the same shape combat's package comment describes with one more
// multiplier folded into the numerator and one more PermilleBase under the line,
// and it truncates exactly once.
func closedForm(r combat.Rules, scaling, defense int64, multiplier, affinity, crit int) int64 {
	numerator := scaling * int64(multiplier) * int64(affinity) * int64(crit) * r.DefenseConstant
	denominator := int64(combat.PermilleBase) * int64(combat.PermilleBase) *
		int64(combat.PermilleBase) * (r.DefenseConstant + defense)
	damage := numerator / denominator
	if damage < r.MinimumDamage {
		return r.MinimumDamage
	}
	return damage
}

// TestACriticalIsOneTruncation is the whole arithmetic claim of the mechanic.
//
// A critical strike is the ordinary expression with the multiplier folded into
// the numerator, so it divides once. The obvious implementation — take Strike's
// answer and multiply it by 1250/1000 — divides twice, and the second division
// throws away up to a point of damage in a way that depends on where the
// defence curve happens to land. The last column of the table is what kills that
// mutation: those cases resolve to different figures, and the test says which
// one is right.
func TestACriticalIsOneTruncation(t *testing.T) {
	r := rules()
	cases := []struct {
		name       string
		defense    int64
		multiplier int
		affinity   int
		// twoStep is what Strike(h) * CriticalMultiplier / PermilleBase would
		// have come to. Where it differs from the answer, this case is load
		// bearing; where it does not, the case is still a check of the figure.
		twoStepDiffers bool
	}{
		{"a clean multiple of the base, both ways", 0, 1800, combat.PermilleBase, false},
		{"halved by defence equal to K", 300, 1800, combat.PermilleBase, false},
		{"both sides resisting, at no defence", 0, 1800, 444, true},
		{"a small skill into a resisted matchup", 0, 325, 444, true},
		{"one point of defence off the ceiling", 1, 1800, combat.PermilleBase, true},
		{"a stacked weakness against real armour", 400, 600, 2250, true},
		{"the defence ceiling", 800, 1800, 667, true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			hit := combat.Hit{
				Scaling: 800, Multiplier: testCase.multiplier,
				Affinity: testCase.affinity, Defense: testCase.defense,
				SkillAccuracy: combat.PermilleBase,
			}
			want := closedForm(r, 800, testCase.defense,
				testCase.multiplier, testCase.affinity, r.CriticalMultiplier)
			got := r.CriticalStrike(hit)
			if got != want {
				t.Fatalf("a critical strike dealt %d, want the %d one truncation gives", got, want)
			}
			twoStep := r.Strike(hit) * int64(r.CriticalMultiplier) / int64(combat.PermilleBase)
			if testCase.twoStepDiffers && got == twoStep {
				t.Errorf("multiplying Strike's answer afterwards also gives %d, so this case proves nothing;"+
					" pick one where the two truncations disagree", got)
			}
			if !testCase.twoStepDiffers && got != twoStep {
				t.Errorf("this case was recorded as one the two readings agree on, but they give %d and %d",
					got, twoStep)
			}
		})
	}
}

// TestAnOrdinaryStrikeIsUnchangedByTheCriticalFactor is the behaviour-preserving
// half, and the reason no battle golden moved.
//
// An ordinary hit passes PermilleBase as its critical multiplier, so it
// multiplies and divides by the same thousand: floor(1000a/1000b) == floor(a/b).
// The pre-change closed form is written out here with no critical term in it at
// all, and every point of the defence curve has to agree with it.
func TestAnOrdinaryStrikeIsUnchangedByTheCriticalFactor(t *testing.T) {
	r := rules()
	// The expression as it stood before a critical multiplier existed.
	before := func(attack, defense int64, skillMultiplier, affinityMultiplier int) int64 {
		if attack <= 0 || skillMultiplier <= 0 || affinityMultiplier <= 0 {
			return 0
		}
		if defense < 0 {
			defense = 0
		}
		numerator := attack * int64(skillMultiplier) * int64(affinityMultiplier) * r.DefenseConstant
		denominator := int64(combat.PermilleBase) * int64(combat.PermilleBase) * (r.DefenseConstant + defense)
		damage := numerator / denominator
		if damage < r.MinimumDamage {
			return r.MinimumDamage
		}
		return damage
	}
	for _, affinity := range []int{444, 667, combat.PermilleBase, 1500, 2250} {
		for _, multiplier := range []int{0, 175, 325, 600, 650, 1000, 1800, 2400} {
			for defense := int64(0); defense <= 1200; defense++ {
				for _, attack := range []int64{1, 200, 800, 2400} {
					want := before(attack, defense, multiplier, affinity)
					if got := r.Damage(attack, defense, multiplier, affinity); got != want {
						t.Fatalf("attack %d, defence %d, skill %d, affinity %d: damage is now %d, was %d",
							attack, defense, multiplier, affinity, got, want)
					}
				}
			}
		}
	}
}

// TestACriticalChanceOfZeroDrawsNothing is the property the whole change rests
// on: every shipped skill crits at nought, and rng.Source.Chance returns without
// drawing at nought, so the random stream a battle walks is bit for bit the one
// it walked before.
//
// It is measured against a source advanced by hand rather than against a
// recorded state, so it keeps saying something if splitmix64 is ever replaced.
func TestACriticalChanceOfZeroDrawsNothing(t *testing.T) {
	r := rules()
	for _, strikes := range []int{0, 1, 2, 3, 4} {
		hit := combat.Hit{
			Scaling: 800, Multiplier: 600, Strikes: strikes, Affinity: combat.PermilleBase,
			Defense: 400, SkillAccuracy: 850, AccuracyStat: 120, DodgeStat: 60,
		}
		rolled, byHand := rng.New(0xC0FFEE), rng.New(0xC0FFEE)
		if _, left := r.Roll(hit, 0, rolled); left != 0 {
			t.Fatalf("Roll returned %d charges from none", left)
		}
		// One accuracy roll per strike and nothing else. If a critical were
		// rolled anywhere — above the switch, on a miss, or on a skill that
		// cannot crit — the two sources would part company here.
		chance := r.Chance(hit)
		for i := 0; i < hit.StrikeCount(); i++ {
			byHand.Chance(chance)
		}
		if rolled.State() != byHand.State() {
			t.Errorf("%d strikes at no critical chance left the source at %d, want the %d that %d accuracy rolls give",
				hit.StrikeCount(), rolled.State(), byHand.State(), hit.StrikeCount())
		}
	}
}

// TestEveryStrikeRollsItsOwnCritical distinguishes a per-strike roll from a
// per-skill one, which is the only thing that can: a skill-level roll produces
// four flags that always agree, so mixed flags on one hit cannot happen under
// it.
func TestEveryStrikeRollsItsOwnCritical(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 400, Strikes: 4, Affinity: combat.PermilleBase,
		Defense: 200, Crit: 500, SkillAccuracy: combat.PermilleBase,
	}
	mixed := false
	for seed := uint64(1); seed <= 200 && !mixed; seed++ {
		attempts, _ := r.Roll(hit, 0, rng.New(seed))
		first := attempts[0].Critical
		for _, attempt := range attempts[1:] {
			if attempt.Critical != first {
				mixed = true
			}
		}
	}
	if !mixed {
		t.Error("200 four-strike hits at half a critical chance never produced a hit that crit on some strikes and not others;" +
			" that is what a skill-level roll looks like")
	}
}

// TestACriticalIsOnlyRolledOnAStrikeThatLands holds the roll inside the arm of
// the switch where the strike connects.
//
// A missed or blocked strike never happened, so there is nothing for it to have
// landed well — and a roll above the switch would draw on every miss, moving the
// stream for a strike that deals nothing. That is not a cosmetic difference: it
// is a battle that replays differently from the one before the mechanic arrived.
func TestACriticalIsOnlyRolledOnAStrikeThatLands(t *testing.T) {
	r := rules()
	// A skill that cannot connect, so every strike misses. The critical chances
	// span the two Chance shortcuts and the middle: 1000 and 0 return without
	// drawing whatever happens, so a middling value is what actually catches a
	// roll leaking onto a miss.
	for _, crit := range []int{200, 500, 999, combat.PermilleBase} {
		hit := combat.Hit{
			Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
			Defense: 400, Crit: crit, SkillAccuracy: 1, DodgeStat: 10_000,
		}
		rolled, byHand := rng.New(7), rng.New(7)
		attempts, _ := r.Roll(hit, 0, rolled)
		chance := r.Chance(hit)
		for i := 0; i < hit.StrikeCount(); i++ {
			byHand.Chance(chance)
		}
		if rolled.State() != byHand.State() {
			t.Errorf("at a critical chance of %d a hit that landed nothing drew more than its %d accuracy rolls",
				crit, hit.StrikeCount())
		}
		for _, attempt := range attempts {
			if attempt.Outcome == combat.Struck {
				t.Fatalf("the hit was built to miss and one strike landed; the test measures nothing")
			}
			if attempt.Critical {
				t.Errorf("a %s strike carries the critical flag", attempt.Outcome)
			}
		}
	}
	// The same against a target that blocks everything, which is the other
	// half: a block cancels a strike that would have landed, and a cancelled
	// strike did not land well either.
	blocking := combat.Rules{
		DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250,
		MinHitChance: 150, MaxBlockCharges: 3,
	}
	hit := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
		Defense: 400, Crit: 999, SkillAccuracy: combat.PermilleBase,
	}
	rolled, byHand := rng.New(21), rng.New(21)
	attempts, left := blocking.Roll(hit, 3, rolled)
	if left != 0 {
		t.Fatalf("three charges against three strikes left %d, want none", left)
	}
	chance := blocking.Chance(hit)
	for i := 0; i < hit.StrikeCount(); i++ {
		byHand.Chance(chance)
	}
	if rolled.State() != byHand.State() {
		t.Error("a fully blocked hit drew a critical for a strike a charge had already cancelled")
	}
	for _, attempt := range attempts {
		if attempt.Outcome != combat.Blocked {
			t.Fatalf("a strike came back %s against three charges; the test measures nothing", attempt.Outcome)
		}
		if attempt.Critical {
			t.Error("a blocked strike carries the critical flag")
		}
	}
}

// TestACriticalIsNotAFourthOutcome is the reason Critical is a flag beside the
// outcome rather than a Critted alongside Missed, Blocked and Struck.
//
// Count(attempts, Struck) is how the engine counts landings — a drain reads it,
// so does every tally — so a fourth outcome would silently stop counting the
// strikes that hit hardest, and nothing would report the change.
func TestACriticalIsNotAFourthOutcome(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
		Defense: 400, Crit: combat.PermilleBase, SkillAccuracy: combat.PermilleBase,
	}
	attempts, _ := r.Roll(hit, 0, rng.New(5))
	if got := combat.Count(attempts, combat.Struck); got != 3 {
		t.Fatalf("three certain critical strikes counted as %d struck, want 3", got)
	}
	for _, attempt := range attempts {
		if !attempt.Critical {
			t.Fatalf("a strike at a certain critical chance did not crit")
		}
	}
	want := r.CriticalStrike(hit) * 3
	if got := combat.DamageDealt(attempts); got != want {
		t.Errorf("three critical strikes dealt %d in total, want %d", got, want)
	}
	if plain := r.Strike(hit) * 3; want <= plain {
		t.Errorf("the critical total %d is no larger than the ordinary %d", want, plain)
	}
}

// TestTheExpectedStrikePricesACriticalWithoutRollingOne covers the function
// Suggest rates a hypothetical action through.
//
// It may not touch the battle's source and it may not keep a second copy of the
// resolving arithmetic, so it is composed from the same two functions Roll uses
// and takes no source at all — which is a fact about the signature, checked by
// the fact that this test never makes one.
func TestTheExpectedStrikePricesACriticalWithoutRollingOne(t *testing.T) {
	r := rules()
	base := combat.Hit{
		Scaling: 800, Multiplier: 600, Strikes: 3, Affinity: combat.PermilleBase,
		Defense: 400, SkillAccuracy: combat.PermilleBase,
	}

	never := base
	if got, want := r.ExpectedStrike(never), r.Strike(never); got != want {
		t.Errorf("a skill that cannot crit is priced at %d, want the %d an ordinary strike deals", got, want)
	}
	if got, want := r.Expected(never), r.Total(never); got != want {
		t.Errorf("a skill that cannot crit is priced at %d over its strikes, want Total's %d", got, want)
	}

	always := base
	always.Crit = combat.PermilleBase
	if got, want := r.ExpectedStrike(always), r.CriticalStrike(always); got != want {
		t.Errorf("a skill that always crits is priced at %d, want the %d a critical strike deals", got, want)
	}
	// Past certainty is clamped rather than extrapolated: skill.ParseBook
	// refuses a share over the base, but the clamp is what makes this function
	// safe for a caller that has not been through the parser.
	past := base
	past.Crit = 5000
	if got, want := r.ExpectedStrike(past), r.CriticalStrike(past); got != want {
		t.Errorf("a critical chance past certainty is priced at %d, want the %d certainty gives", got, want)
	}

	half := base
	half.Crit = 500
	ordinary, critical := r.Strike(half), r.CriticalStrike(half)
	got := r.ExpectedStrike(half)
	if got <= ordinary || got >= critical {
		t.Errorf("half a critical chance is priced at %d, want it strictly between %d and %d",
			got, ordinary, critical)
	}
	if want := r.ExpectedStrike(half) * 3; r.Expected(half) != want {
		t.Errorf("three strikes are priced at %d, want three times one strike's %d",
			r.Expected(half), r.ExpectedStrike(half))
	}
	// Total is deliberately the *deterministic* figure and must not have picked
	// up the weighting: it writes skills.golden's damage column.
	if r.Total(half) != ordinary*3 {
		t.Errorf("Total is %d on a critting skill, want the %d its ordinary strikes deal;"+
			" the golden column is a fixed figure, not an expected one", r.Total(half), ordinary*3)
	}
}

// TestTheDamageFormulaHasHeadroomForACritical checks the one cost of folding the
// multiplier into the numerator: the product grew by a factor of a thousand.
//
// The worst hit the bounds admit is built here rather than guessed at, and the
// margin is printed on failure so the next person to raise a ceiling can read
// how much room they have instead of re-deriving this.
func TestTheDamageFormulaHasHeadroomForACritical(t *testing.T) {
	r := rules()
	const (
		// The largest shipped stat ceiling is health at 4800, and a buffed stat
		// saturates towards ceiling * headroom, headroom being 3000 per mille.
		// A skill may scale off any of the six, so the largest is the bound.
		saturatedStat = int64(4800) * 3
		// The shipped book's largest power is 2400. Ten times that is a
		// deliberately generous allowance for every amplifier and gradient that
		// can stack on top of it — skill.ParseBook sets no ceiling on power, so
		// this is an allowance rather than a bound, and it is the number to
		// revisit if one ever gets near it.
		amplifiedPower = 2400 * 10
		// The chart's worst matchup is 2250, and modifier.Bounds clamps the
		// affinity term at 1000 per mille, which scales the deviation from
		// neutral by two: 1000 + (2250-1000)*2.
		clampedAffinity = 1000 + (2250-1000)*2
	)
	// Zero defence, because the denominator is smallest there and the quotient
	// largest. The numerator itself does not depend on it.
	numerator := saturatedStat * amplifiedPower * clampedAffinity *
		int64(r.CriticalMultiplier) * r.DefenseConstant
	if numerator <= 0 {
		t.Fatalf("the worst hit the bounds admit overflows int64 outright: %d", numerator)
	}
	margin := math.MaxInt64 / numerator
	if margin < 4 {
		t.Fatalf("the worst hit the bounds admit puts %d in the numerator, which clears math.MaxInt64 by only %dx;"+
			" folding the critical multiplier in costs a factor of %d, so raising a ceiling from here needs the"+
			" expression rearranged rather than the ceiling reconsidered",
			numerator, margin, combat.PermilleBase)
	}
	t.Logf("the worst hit the bounds admit puts %d in the numerator, %dx clear of math.MaxInt64", numerator, margin)
	// And the same hit resolves to something positive rather than a wrapped
	// negative, through the real function.
	worst := r.CriticalStrike(combat.Hit{
		Scaling: saturatedStat, Multiplier: amplifiedPower, Affinity: clampedAffinity,
		Defense: 0, SkillAccuracy: combat.PermilleBase,
	})
	if worst <= 0 {
		t.Errorf("the worst critical strike the bounds admit deals %d, want a positive figure", worst)
	}
}

// TestAHitAtTheDamageFloorIsUnchangedByACritical: the floor is applied to the
// one quotient, so a hit that resolves under it lands on it whether it crit or
// not. A critical is a multiplier on the expression, not a bypass of the floor.
func TestAHitAtTheDamageFloorIsUnchangedByACritical(t *testing.T) {
	r := rules()
	hit := combat.Hit{
		Scaling: 1, Multiplier: 1, Affinity: 1, Defense: 1200,
		SkillAccuracy: combat.PermilleBase,
	}
	ordinary, critical := r.Strike(hit), r.CriticalStrike(hit)
	if ordinary != r.MinimumDamage {
		t.Fatalf("the hit was built to resolve under the floor and dealt %d; the test measures nothing", ordinary)
	}
	if critical != r.MinimumDamage {
		t.Errorf("a critical strike at the damage floor dealt %d, want the floor's %d", critical, r.MinimumDamage)
	}
	// And a hit with no power behind it deals nothing at all, critical or not.
	nothing := hit
	nothing.Multiplier = 0
	if got := r.CriticalStrike(nothing); got != 0 {
		t.Errorf("a critical strike with no power dealt %d, want nothing", got)
	}
}

// TestValidateRefusesACriticalMultiplierThatIsNotCritical: the constant is
// required rather than tolerated.
//
// An absent value reads as nought, and a nought multiplier drives every critical
// strike to MinimumDamage — the mechanic silently inverted, with the book still
// loading and every ordinary hit still correct. A value of exactly the base is
// refused for the same reason with the sign the other way: a critical worth the
// same as an ordinary hit is a mechanic that does nothing.
func TestValidateRefusesACriticalMultiplierThatIsNotCritical(t *testing.T) {
	for _, multiplier := range []int{0, -1, 1, combat.PermilleBase} {
		r := rules()
		r.CriticalMultiplier = multiplier
		err := r.Validate()
		if err == nil {
			t.Errorf("a critical multiplier of %d was accepted", multiplier)
			continue
		}
		if !strings.Contains(err.Error(), "critical_multiplier") {
			t.Errorf("the refusal of %d does not name the field: %v", multiplier, err)
		}
	}
	// One over the base is the smallest critical that is really one.
	r := rules()
	r.CriticalMultiplier = combat.PermilleBase + 1
	if err := r.Validate(); err != nil {
		t.Errorf("a critical multiplier of %d was refused: %v", r.CriticalMultiplier, err)
	}
	// And it is refused at the parser too, where an absent key is the way it
	// actually goes wrong.
	_, err := combat.ParseRules([]byte(
		`{"defense_constant":300,"minimum_damage":1,"min_hit_chance":150,"max_block_charges":3}`))
	if err == nil {
		t.Fatal("a rules file with no critical_multiplier was accepted")
	}
	if !strings.Contains(err.Error(), "critical_multiplier") {
		t.Errorf("the refusal does not name the missing field: %v", err)
	}
}
