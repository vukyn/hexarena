// Package combat holds the damage formula and the balance constants it reads.
//
// Damage is a ratio against defence rather than a subtraction:
//
//	damage = attack * skillMultiplier * affinityMultiplier * K / (K + defence)
//
// K is DefenseConstant, and the shape of the curve follows from it: a defender
// whose defence equals K takes half damage, one with no defence takes the full
// figure, and the curve flattens from there instead of ever reaching zero.
// Subtraction was rejected because attack - defence produces a breakpoint where
// a small stat change flips a hit between full damage and none, and that
// breakpoint moves every time the level cap changes.
//
// Both multipliers are integers in parts per thousand and the whole expression
// resolves with a single integer division at the end, so truncation happens
// once and the result is identical on every platform.
//
// The numerator that division reads is built in 128 bits. Five factors
// multiplied together pass what an int64 holds long before any one of them is
// an unreasonable number to author, and the obvious repair — dividing earlier —
// is refused precisely because it would truncate twice. Widening the
// intermediate is what lets the division stay single; below the point where the
// narrower arithmetic overflowed, the answer is bit for bit the one it gave.
package combat

import (
	"encoding/json"
	"fmt"
	"math"
	"math/bits"

	"github.com/vukyn/hexarena/internal/core/rng"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// PermilleBase is the denominator every multiplier in this package is
// expressed against. The element chart's neutral multiplier uses the same base.
const PermilleBase = scale.Base

// Rules holds the tunable constants of the damage formula.
type Rules struct {
	// DefenseConstant is the defence value at which a defender takes half
	// damage. Its size sets how long defence keeps mattering: too small and
	// defence saturates early, too large and defence scales close to linearly
	// and tanks run away with the late game.
	DefenseConstant int64 `json:"defense_constant"`
	// MinimumDamage is the floor a connecting hit deals, so a heavily
	// resisted attack against a heavily armoured target still moves the
	// battle forward instead of stalling it.
	MinimumDamage int64 `json:"minimum_damage"`
	// CriticalMultiplier is what a critical strike multiplies the whole damage
	// expression by, in parts per thousand. It is one game-wide constant rather
	// than a figure a skill declares: how *often* a skill crits is what makes
	// one skill different from the next, and a second per-skill number would
	// only let two skills disagree about what the word means.
	//
	// Validate requires it rather than tolerating an absent one. A missing
	// value reads as nought, which drives every critical strike straight to
	// MinimumDamage — the mechanic silently inverted, with the book still
	// loading and every ordinary hit still correct.
	CriticalMultiplier int `json:"critical_multiplier"`
	// MinHitChance is the limit dodge drives a hit chance towards without ever
	// reaching it, in parts per thousand. Without a floor a dodge-stacked unit
	// would eventually be untouchable, which is not a defensive stat but an
	// immunity.
	MinHitChance int `json:"min_hit_chance"`
	// MaxBlockCharges is the most charges a unit may hold at once.
	//
	// Charges are the one defence in the engine that does not saturate: three
	// of them cancel three strikes outright, with no diminishing return, so a
	// stack of them can erase a whole round of incoming damage rather than
	// reduce it. A hard cap is the right bound for a discrete resource, the same
	// reason the affinity scale is clamped rather than saturated.
	MaxBlockCharges int `json:"max_block_charges"`
}

// ParseRules reads a rules declaration. It never touches the filesystem; the
// caller supplies the bytes.
func ParseRules(raw []byte) (Rules, error) {
	var rules Rules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return Rules{}, fmt.Errorf("decode combat rules: %w", err)
	}
	if err := rules.Validate(); err != nil {
		return Rules{}, err
	}
	return rules, nil
}

// Validate rejects constants that would make the formula meaningless.
func (r Rules) Validate() error {
	switch {
	case r.DefenseConstant <= 0:
		return fmt.Errorf("defense_constant is %d, want a positive value", r.DefenseConstant)
	case r.MinimumDamage < 0:
		return fmt.Errorf("minimum_damage is %d, want zero or more", r.MinimumDamage)
	case r.CriticalMultiplier <= PermilleBase:
		return fmt.Errorf("critical_multiplier is %d, want more than %d so a critical hit is bigger than an ordinary one", r.CriticalMultiplier, PermilleBase)
	case r.MinHitChance <= 0:
		return fmt.Errorf("min_hit_chance is %d, want a positive value", r.MinHitChance)
	case r.MinHitChance >= scale.Base:
		return fmt.Errorf("min_hit_chance is %d, want less than %d so dodge has somewhere to go", r.MinHitChance, scale.Base)
	case r.MaxBlockCharges < 1:
		return fmt.Errorf("max_block_charges is %d, want at least 1", r.MaxBlockCharges)
	}
	return nil
}

// GrantBlocks adds charges to what a unit already holds, capped at
// MaxBlockCharges, and reports how many were wasted on the cap.
//
// Granting goes through here rather than being a plain addition so that a
// stack of shielding effects cannot quietly push a unit past the cap, and so
// the effect that overshoots can say so instead of silently doing nothing.
func (r Rules) GrantBlocks(held, granted int) (total, wasted int) {
	if held < 0 {
		held = 0
	}
	if granted < 0 {
		granted = 0
	}
	total = held + granted
	if total > r.MaxBlockCharges {
		return r.MaxBlockCharges, total - r.MaxBlockCharges
	}
	return total, 0
}

// Damage returns the damage a hit deals. skillMultiplier and
// affinityMultiplier are in parts per thousand; a skill multiplier of 1800 is
// a 1.8x hit and an affinity multiplier of 2250 is a stacked weakness.
//
// A hit that resolves below MinimumDamage is raised to it, but an attack with
// no power behind it deals nothing at all.
//
// defense is the defence as it applies to this hit, piercing already taken off.
// Piercing is Pierced's job rather than a fifth parameter here: what this
// signature refuses is a fifth positional *non-multiplier*. attack and defense
// are not interchangeable and a mis-ordered pair passes silently at every
// caller; Hit is the struct that exists precisely to carry an attack whose terms
// are all settled. A caller that reaches this directly rather than through
// Strike is asking for the raw curve, and a damage-over-time tick is exactly
// that — see turn.go.
//
// The private damage below is allowed the fifth argument the public one is not,
// because a critical multiplier is a *multiplier*: the three of them commute, so
// a mis-order among them cannot change a figure, and it is unexported so no
// caller outside this package can mis-order them anyway. It exists so the
// package resolves through exactly one expression, and so a critical strike
// truncates once rather than multiplying an already-floored result.
func (r Rules) Damage(attack, defense int64, skillMultiplier, affinityMultiplier int) int64 {
	return r.damage(attack, defense, skillMultiplier, affinityMultiplier, PermilleBase)
}

// damage is the one damage expression this package has. critMultiplier folds
// into the numerator with a matching PermilleBase in the denominator, so an
// ordinary hit passes PermilleBase and multiplies and divides by the same
// thousand: floor(1000a/1000b) == floor(a/b), exactly. That identity is why
// adding the mechanic moved no damage figure anywhere.
//
// ⚠️ **The numerator is a product of five and does not fit an int64.** It used
// to be written as one int64 expression and wrapped silently: measured at the
// attack ceiling against half the defence ceiling, a power of ninety million
// came to four and a half million — a large, plausible, wrong figure, not a
// visibly broken one — while a power of a hundred and twenty million came to
// MinimumDamage off a wrapped numerator. Nothing refuses such a power, because
// skill.Validate bounds power below and not above, so a skill like that parses,
// saves and fights. And the wrapped expression is *not monotone in power*, so
// no reading taken off a single figure could have caught it.
//
// The repair may not be to divide earlier. Two truncations break the identity
// above and move every damage figure in the game, which is the one thing the
// single division exists to prevent. So the intermediate got wider instead —
// see wide — and the division stayed single.
//
// Every factor is positive by the time the product is built, which is what
// makes the unsigned arithmetic safe to read: the early return refuses a
// non-positive attack or multiplier and a negative defence is clamped.
// DefenseConstant joins that guard as the fifth factor. It is the only one that
// comes from data rather than from a caller, Validate already refuses a
// non-positive one, and so this is unreachable through every loading path; it
// is written down so the conversion below cannot be handed a negative by a
// Rules built by hand.
func (r Rules) damage(attack, defense int64, skillMultiplier, affinityMultiplier, critMultiplier int) int64 {
	if attack <= 0 || skillMultiplier <= 0 || affinityMultiplier <= 0 || critMultiplier <= 0 ||
		r.DefenseConstant <= 0 {
		return 0
	}
	if defense < 0 {
		defense = 0
	}
	numerator := widen(uint64(attack), uint64(skillMultiplier)).
		times(uint64(affinityMultiplier)).
		times(uint64(critMultiplier)).
		times(uint64(r.DefenseConstant))
	// The denominator stays a plain int64, and it has room to. It is a thousand
	// cubed times K plus defence, so it overflows only past a K plus defence of
	// 9,223,372,037; defence saturates at three times its progression ceiling of
	// 800, which puts the largest reachable value of that sum at 2,699 with the
	// shipped constant of 300. Three million times the room it needs.
	denominator := int64(PermilleBase) * int64(PermilleBase) * int64(PermilleBase) * (r.DefenseConstant + defense)
	damage := numerator.over(uint64(denominator))
	if damage < r.MinimumDamage {
		return r.MinimumDamage
	}
	return damage
}

// wide is an unsigned 128-bit intermediate together with whether it is still
// exact. It exists for the two products in this package that do not fit an
// int64 — the damage numerator and Swung — and for nothing else.
//
// exact goes false the moment a product passes 128 bits and never goes back, so
// one overflow anywhere in a chain is still an overflow when the chain ends.
// Nothing the game can author comes near it — the numerator's shipped factors
// would need a power past 1.9e26, and Swung's single widening cannot lose
// exactness at all, since widen returns the exact product of a pair — and it is
// carried anyway, because a silent wrap at 128 bits is the same defect as the
// one at 64, only rarer and therefore harder to find the next time.
type wide struct {
	high, low uint64
	exact     bool
}

// widen returns the exact 128-bit product of two 64-bit values.
func widen(a, b uint64) wide {
	high, low := bits.Mul64(a, b)
	return wide{high: high, low: low, exact: true}
}

// times multiplies by a further 64-bit factor.
//
// bits.Mul64 widens a *pair*, and the numerator is a product of five, so the
// chain needs this 128x64 step to continue past the first multiplication. The
// low word carries the whole result and the high word's product must land
// entirely in the upper half; anything above that, or a carry out of it, is the
// value no longer fitting.
func (w wide) times(factor uint64) wide {
	upperHigh, upperLow := bits.Mul64(w.high, factor)
	high, low := bits.Mul64(w.low, factor)
	high, carry := bits.Add64(high, upperLow, 0)
	return wide{high: high, low: low, exact: w.exact && upperHigh == 0 && carry == 0}
}

// over divides by a 64-bit denominator, saturating rather than panicking on a
// quotient that will not fit.
//
// ⚠️ **The guard is not optional.** bits.Div64 panics on both of the two things
// that can go wrong here — a divisor of nought, and a quotient that would pass
// 64 bits — and a panic inside the damage formula is strictly worse than the
// wrap it replaces: it takes the battle down rather than printing a wrong
// number.
//
// One comparison answers both, and that is arithmetic rather than luck.
// Div64's precondition is that the high word is below the divisor, and a high
// word is unsigned, so it is never below nought: a divisor of nought fails
// `high >= divisor` for every value there is. A separate `denominator == 0`
// clause was written first and a mutation deleting it survived the whole suite,
// which is what said so. Do not put it back.
//
// The second check is a different question rather than a repeat. A high word
// below the divisor puts the quotient inside 64 bits, but not necessarily
// inside a *signed* one.
//
// What a quotient past the type should produce is math.MaxInt64, and the
// argument is that this is a bound on the *type* and not on the design. The
// largest effective health the progression limits allow is eleven and a half
// thousand, so every input that reaches here already kills everything it can
// touch and pinning it changes no figure any battle could show; damage larger
// than any health is not a wrong answer, where a panic is. It is also the only
// choice that keeps damage non-decreasing in power, which is the property the
// wrap actually broke.
//
// It is deliberately *not* a ceiling on what may be authored. An implementation
// limit must not set a design bound — that number is the game's to choose.
func (w wide) over(denominator uint64) int64 {
	if !w.exact || w.high >= denominator {
		return math.MaxInt64
	}
	quotient, _ := bits.Div64(w.high, w.low, denominator)
	if quotient > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(quotient)
}

// Restore returns how much health a multiplier of a stat gives back.
//
// It does not divide by the defence curve, and damage over time does. That
// asymmetry is the design rather than an oversight: defence turns away what is
// coming *at* a unit and has nothing to do with what is helping it, so a
// well-armoured unit is no harder to heal than a bare one. Adding the division
// for symmetry's sake would make armour quietly reduce its own side's support.
//
// Like everything else here it is integer parts per thousand, truncated once.
func (r Rules) Restore(stat int64, multiplier int) int64 {
	if stat <= 0 || multiplier <= 0 {
		return 0
	}
	return stat * int64(multiplier) / int64(PermilleBase)
}

// Pierced returns the defence a hit resolves against once a piercing ratio has
// been taken off it, in parts per thousand of defence ignored.
//
// It is a ratio rather than a switch, and that is the whole design. Measured
// against a 3100/800 unit and a 4800/400 one, both of which sit at the same
// joint budget, a ratio walks the armoured unit's edge from 0.98x to 1.55x
// across its range while a switch jumps straight to the far end — making an
// armour unit worthless against one skill and unaffected by the next, with
// nothing in between. A hard cap on a continuous quantity is the shape this
// engine has rejected everywhere else, which is why buffs saturate.
//
// Only the ignored share is truncated, so full piercing reaches exactly zero
// defence and no piercing leaves the defence exactly as it was.
func Pierced(defense int64, pierce int) int64 {
	if defense <= 0 {
		return 0
	}
	if pierce <= 0 {
		return defense
	}
	if pierce >= PermilleBase {
		return 0
	}
	return defense * int64(PermilleBase-pierce) / int64(PermilleBase)
}

// DefenseReduction returns the share of damage that gets through a given
// defence, in parts per thousand. It exists for tuning and for showing the
// curve; Damage does not call it, because folding it in would introduce a
// second truncation.
func (r Rules) DefenseReduction(defense int64) int {
	if defense < 0 {
		defense = 0
	}
	return int(r.DefenseConstant * int64(PermilleBase) / (r.DefenseConstant + defense))
}

// Hit describes one attack after stats, buffs and the elemental chart have all
// been settled. It deliberately carries plain numbers rather than references to
// a unit or a skill, so this package stays free of every layer above it.
type Hit struct {
	// Scaling is the attacker's scaling stat, already buffed. Most skills
	// scale off attack, but a skill may declare a different stat; the layer
	// that builds the Hit picks which one, and the multiplier is what
	// normalises between stats of different magnitudes.
	Scaling int64
	// Multiplier is the skill's power for a single strike, in parts per
	// thousand.
	Multiplier int
	// Strikes is how many times the skill lands. Zero and one both mean a
	// single strike. A multi-strike skill is expected to divide its power
	// across strikes rather than repeat it at full power.
	Strikes int
	// Affinity is the elemental multiplier, in parts per thousand.
	Affinity int
	// Defense is the target's defence, already buffed and not yet pierced.
	Defense int64
	// Pierce is the share of that defence the skill ignores, in parts per
	// thousand. Zero is every skill that does not pierce, which is why adding
	// the field moved no golden file.
	//
	// It sits on the hit rather than being folded into Defense by the caller so
	// that the figure survives to the event log: a pierced hit that logs like an
	// ordinary one leaves the log unable to explain its own numbers, which is
	// the same trap a silent passive would set.
	Pierce int
	// Crit is the chance each strike lands critically, in parts per thousand,
	// and it is the *skill's* own figure. No stat moves it: progression.Values
	// is a fixed-size array behind a schema of six required pointers, and a
	// seventh kind is not something a stat line may grow — so a skill that crits
	// crits at the same rate in every hand that carries it, and how often is the
	// thing that distinguishes one skill from the next.
	//
	// Zero is every skill in the book today, which is why adding it moved no
	// battle golden and no saved log: rng.Source.Chance returns without drawing
	// at a chance of nought, so the stream is untouched. Pierce used the same
	// trick, for the same reason.
	Crit int
	// SkillAccuracy is the skill's own chance to connect, in parts per
	// thousand. Zero means the skill never lands, so it must be set.
	SkillAccuracy int
	// AccuracyStat is the attacker's accuracy stat, which closes the gap
	// between the skill's own accuracy and a certain hit.
	AccuracyStat int64
	// DodgeStat is the target's dodge stat, which reopens that gap.
	DodgeStat int64
}

// ScalingSource selects which version of the attacker's scaling stat a skill
// reads.
//
// The distinction exists because a stat that feeds both the turn order and the
// damage would otherwise be worth its square when buffed: a fifty percent speed
// buff would grant fifty percent more turns each dealing fifty percent more
// damage. A skill that reads BaseStat lets a buff move the turn economy while
// leaving the damage term alone, which keeps such a buff worth the same as any
// other. A skill that reads CurrentStat accepts the compounding on purpose.
type ScalingSource uint8

const (
	// CurrentStat reads the stat after buffs and debuffs.
	CurrentStat ScalingSource = iota
	// BaseStat reads the stat before them.
	BaseStat
)

func (s ScalingSource) String() string {
	if s == BaseStat {
		return "base"
	}
	return "current"
}

// PickScaling returns whichever version of a stat the source names.
func PickScaling(source ScalingSource, base, current int64) int64 {
	if source == BaseStat {
		return base
	}
	return current
}

// Chance returns the probability the hit connects, in parts per thousand.
//
// The skill's own accuracy is where it starts. The attacker's accuracy stat
// closes the gap towards a certain hit; the target's dodge stat then reopens it
// towards MinHitChance. Both sides saturate, so neither can be stacked into an
// absolute: no amount of accuracy makes an unreliable skill certain, and no
// amount of dodge makes a unit untouchable.
//
// Dodge is applied after accuracy rather than subtracted from it, because a
// contest of the two raw stats would cancel out and a defender with high dodge
// would gain nothing at all against an attacker with high accuracy. Applying it
// second means dodge always bites, even against an attack that was going to
// land ninety-nine times in a hundred. The cost of that choice is that dodge
// works against a wider gap than accuracy does and is therefore worth more per
// point; the dodge ceiling is set lower to pay for it.
//
// A skill that declares full accuracy cannot miss and cannot be dodged. That is
// the explicit way to write an effect that must land, and it is what gives a
// block charge its purpose.
func (r Rules) Chance(h Hit) int {
	if h.SkillAccuracy >= scale.Base {
		return scale.Base
	}
	accuracy := h.SkillAccuracy
	if accuracy < 0 {
		accuracy = 0
	}
	miss := int64(scale.Base - accuracy)
	stat := h.AccuracyStat
	if stat < 0 {
		stat = 0
	}
	landed := int64(accuracy) + scale.Saturate(0, stat, miss, 0)

	dodge := h.DodgeStat
	if dodge <= 0 {
		return int(landed)
	}
	floor := int64(r.MinHitChance)
	if landed <= floor {
		return int(landed)
	}
	return int(scale.Saturate(landed, -dodge, scale.Base, floor))
}

// Outcome is what became of one strike.
type Outcome uint8

const (
	// Missed means the strike did not connect.
	Missed Outcome = iota
	// Blocked means it connected and a block charge cancelled it.
	Blocked
	// Struck means it connected and dealt its damage.
	Struck
)

func (o Outcome) String() string {
	switch o {
	case Missed:
		return "missed"
	case Blocked:
		return "blocked"
	case Struck:
		return "struck"
	default:
		return "unknown"
	}
}

// Attempt is the outcome of one strike.
type Attempt struct {
	Outcome Outcome
	Damage  int64
	// Critical says this strike landed critically. It is a flag beside the
	// outcome rather than a fourth Outcome, and that is deliberate:
	// Count(attempts, Struck) is how the engine counts landings — drains,
	// on-hit riders and every tally read it — so a Critted outcome would
	// silently change what that counts, and every one of those callers would
	// start missing exactly the strikes that hit hardest. A critical strike is
	// a strike that landed well, not a different thing from a strike.
	Critical bool
}

// Roll resolves every strike of a hit against its chance to connect, spending
// the target's block charges on whichever strikes get through. It returns the
// per-strike outcomes and the charges left over.
//
// Each strike rolls on its own rather than the skill rolling once, so a
// multi-strike skill can land partially. That is what gives accuracy a
// different shape on a multi-strike skill than on a single one: the expected
// damage is the same, but the variance is far lower. An area skill rolls once
// per target for the same reason, by resolving each target as its own hit.
//
// A charge is only spent on a strike that would otherwise have landed, so a
// target that dodges wastes none. One charge cancels one strike, which is what
// makes block the answer to a single heavy hit and multi-strike the answer to
// block: the same charge that erases a full-power blow is burned by a third of
// one.
func (r Rules) Roll(h Hit, blocks int, source *rng.Source) (attempts []Attempt, blocksLeft int) {
	chance := r.Chance(h)
	// Both figures a strike can come to, resolved once outside the loop. There
	// are only two because the critical multiplier is a game-wide constant; a
	// per-skill one would have made this a per-strike computation for nothing.
	damage := r.Strike(h)
	critical := r.CriticalStrike(h)
	count := h.StrikeCount()
	out := make([]Attempt, 0, count)
	remaining := blocks
	for i := 0; i < count; i++ {
		switch {
		case !source.Chance(chance):
			out = append(out, Attempt{Outcome: Missed})
		case remaining > 0:
			remaining--
			out = append(out, Attempt{Outcome: Blocked})
		default:
			// The critical is rolled here rather than above the switch, and per
			// strike rather than per skill. A missed or blocked strike never
			// happened, so there is nothing for it to have landed well; rolling
			// above the switch would draw on every miss and move the stream for
			// a strike that deals nothing, which is a battle that replays
			// differently from the one before this line existed.
			if source.Chance(h.Crit) {
				out = append(out, Attempt{Outcome: Struck, Damage: critical, Critical: true})
				continue
			}
			out = append(out, Attempt{Outcome: Struck, Damage: damage})
		}
	}
	return out, remaining
}

// DamageDealt returns how much damage a set of attempts actually dealt.
func DamageDealt(attempts []Attempt) int64 {
	total := int64(0)
	for _, attempt := range attempts {
		if attempt.Outcome == Struck {
			total += attempt.Damage
		}
	}
	return total
}

// Count returns how many attempts ended in the given outcome.
func Count(attempts []Attempt, outcome Outcome) int {
	total := 0
	for _, attempt := range attempts {
		if attempt.Outcome == outcome {
			total++
		}
	}
	return total
}

// Strikes returns how many times the hit lands, treating an unset count as one.
func (h Hit) StrikeCount() int {
	if h.Strikes < 1 {
		return 1
	}
	return h.Strikes
}

// Strike returns the damage of a single strike, against whatever defence the
// hit's piercing leaves standing.
func (r Rules) Strike(h Hit) int64 { return r.strike(h, PermilleBase) }

// CriticalStrike returns the damage of a single strike that landed critically.
//
// It is the same expression with the game-wide multiplier folded into it rather
// than Strike's answer multiplied afterwards. Multiplying afterwards would be a
// second truncation, so a critical hit would sometimes come to one point less
// than the formula says — and the shortfall would depend on the defence curve,
// which is the one thing this package resolves in a single division precisely so
// nobody has to reason about that.
func (r Rules) CriticalStrike(h Hit) int64 { return r.strike(h, r.CriticalMultiplier) }

func (r Rules) strike(h Hit, critMultiplier int) int64 {
	return r.damage(h.Scaling, Pierced(h.Defense, h.Pierce), h.Multiplier, h.Affinity, critMultiplier)
}

// Resolve returns the damage of each strike in order.
//
// Every strike is computed and truncated on its own rather than by dividing a
// single total, because a strike is the unit that on-hit effects and shields
// react to. The cost is that a multi-strike skill loses up to one point of
// damage per strike to truncation, which is why splitting power across strikes
// is very slightly worse than concentrating it.
func (r Rules) Resolve(h Hit) []int64 {
	count := h.StrikeCount()
	out := make([]int64, 0, count)
	strike := r.Strike(h)
	for i := 0; i < count; i++ {
		out = append(out, strike)
	}
	return out
}

// Total returns the damage of every strike combined, ignoring the chance of a
// critical.
//
// ⚠️ It stays because its remaining caller is internal/seed's skillReport, which
// writes skills.golden's damage column, and that column is a *deterministic*
// figure the design record is read against — a number that moved with an
// expected value would stop being the thing an author compares two skills by.
// Anything rating a hypothetical action wants Expected instead. Do not delete
// this as dead weight: doing so takes the golden column with it.
func (r Rules) Total(h Hit) int64 {
	return r.Strike(h) * int64(h.StrikeCount())
}

// ExpectedStrike returns what one strike is worth before it is rolled, with the
// chance of a critical priced in.
//
// It exists so a rating can charge for a critical without drawing one: Suggest
// may not touch the battle's source, and it may not keep a second copy of the
// resolving arithmetic either, so this is composed from the same two functions
// Roll resolves through.
//
// A skill that cannot crit takes the early return rather than the weighting,
// which is what makes this bit-identical to Strike for every skill in the book
// today — and therefore why the opponent's choices did not move.
func (r Rules) ExpectedStrike(h Hit) int64 {
	if h.Crit <= 0 {
		return r.Strike(h)
	}
	chance := int64(h.Crit)
	if chance > PermilleBase {
		chance = PermilleBase
	}
	ordinary := r.Strike(h)
	critical := r.CriticalStrike(h)
	return (ordinary*(int64(PermilleBase)-chance) + critical*chance) / int64(PermilleBase)
}

// Expected returns what every strike of a hit is worth before any is rolled.
//
// Because ExpectedStrike returns early at a chance of nought, this is bit for
// bit Total for every shipped skill.
func (r Rules) Expected(h Hit) int64 {
	return r.ExpectedStrike(h) * int64(h.StrikeCount())
}

// Gradient returns the share a hurt caster adds to its own skill's power, in
// parts per thousand: nought at full health, atEmpty as it approaches nothing,
// and a straight line between the two.
//
// It is the smooth half of an idea whose stepped half already exists.
// `self_requires` asks a *threshold* -- at or below forty per cent, take a fixed
// bonus -- and a threshold is the right shape for a skill that becomes a
// different skill once a line is crossed. This is the other shape: the move that
// grows with the wound rather than switching at one, where every point of health
// lost is worth the same as the last and there is no line for either side to play
// around.
//
// ⚠️ **A multiplier rather than a bonus, and that is the whole reason it is
// arithmetic here rather than a second Condition in skill.** A bonus is a number
// added to power and would have to be added to *something* -- the declared power,
// which is not what a skill lands at once a detonate has amplified it. A share of
// whatever power the skill arrived at means a caster swinging harder swings
// harder at the power it actually has, and the two terms compose instead of
// arguing about which goes first. A Condition could not express it either way,
// because a Condition answers yes or no and there is no yes or no here.
//
// ⚠️ **It returns the share added, not the multiplier**, which is what makes
// nought mean "nothing happened" — the same shape as Pierce, Refused and Drained,
// all of which are the share that moved a number and are absent from the log when
// none did. The caller adds PermilleBase itself, in one place.
//
// A maximum of nought is answered with nought rather than divided by: something
// with no maximum is not something that is hurt.
//
// ⚠️ The `=` half of the full-health guard is redundant with the arithmetic below
// and always will be — at exactly full health the missing share is nought, so the
// last line already answers nought — which means mutating it to `>` survives the
// whole suite. It is kept anyway because the `>` half is *not* redundant: a unit
// somehow past its own bar would otherwise earn a negative share and hand a hurt
// skill less power than a healthy one. Written down so the next person to mutate
// this stops looking for the missing test.
func Gradient(health, maximum int64, atEmpty int) int {
	if maximum <= 0 || atEmpty <= 0 || health >= maximum {
		return 0
	}
	if health <= 0 {
		return atEmpty
	}
	// One division, at the end, like everything else in this package: the share
	// of health missing and the share of the bonus earned are the same fraction,
	// so they are multiplied before they are divided.
	return int(int64(atEmpty) * (maximum - health) / maximum)
}

// Swung is the power a skill lands at once the caster's own terms are in: the
// bonus a threshold on the caster adds, and the share a gradient multiplies in.
//
// ⚠️ **The bonus first and the share second, and the order is a design rather
// than an accident:** a caster swinging harder swings harder at the power it
// actually has, so a detonate that arrives at three thousand four hundred is what
// the wound is a share of. The other order would make the gradient a share of the
// declared power and quietly worth less on exactly the skills it should be worth
// most on.
//
// It lives here, beside Gradient, because two callers have to agree on it
// exactly and neither may own it. The battle resolves a hit through it and the
// authoring preview measures an unwritten skill through it, so an author reading
// a figure before a write and the engine landing the blow afterwards are reading
// one arithmetic rather than two that agree today.
//
// ⚠️ **The product is built in 128 bits, for the reason the damage numerator is,
// and it matters more here.** This figure *becomes* `skillMultiplier` in
// Rules.damage, so it is upstream of the widening that protects that expression
// and the widening there cannot reach it. Written in one `int` it passed
// math.MaxInt64 at a power around 9.2e15 and wrapped, which handed damage a
// negative multiplier — refused by that function's first line — so an enormous
// power came back dealing MinimumDamage.
//
// ⚠️ **The widening is reused rather than a cheaper overflow check, and the
// reason is that the divisor is one of the multiplicands' own scale.** A skill
// with no bonus and no share is `power * 1000 / 1000`, which is `power` for
// every power the type holds. A guard that saturated whenever the *product* left
// an int64 would answer math.MaxInt64 to that — refusing a figure it was handed
// and could have returned untouched, three orders of magnitude below where the
// quotient stops fitting. Widening answers `power`, so the identity survives and
// the only inputs that saturate are the ones whose answer genuinely does not fit.
//
// A quotient past the type saturates at math.MaxInt64, which is a bound on the
// **type** and not on the design, exactly as it is in the numerator — and the two
// compose: a saturated multiplier hands damage a factor that saturates its own
// division in turn, so the blow lands at the widest figure an int64 holds against
// a bar whose largest reachable value is eleven and a half thousand. It kills
// whatever it touches, which is what a power that large asked for. What it must
// not do is wrap, because a wrap is the one answer that comes back *smaller*.
//
// ⚠️ **The three clamps refuse a negative rather than preserving what one used
// to produce, and that is the one input whose answer moves.** None is reachable:
// power is Skill.Power plus a Condition's BonusPower and Validate refuses a
// negative of either, the bonus is that same field read off the caster, and
// Gradient cannot return a negative share — its full-health guard exists
// precisely so a unit somehow past its own bar does not earn one. They are
// written down anyway, because the arithmetic below is unsigned and a Skill built
// by hand must not be able to reach it; and there is no answer to preserve, since
// a negative power, a bonus that is a penalty and a wound that weakens are three
// things no field expresses.
func Swung(power, bonus, share int) int {
	if power < 0 {
		power = 0
	}
	if bonus < 0 {
		bonus = 0
	}
	if share < 0 {
		share = 0
	}
	// Both sums are exact in 64 unsigned bits and neither needs a guard: two
	// non-negative int64 values cannot reach 2^64 between them, so only the
	// multiplication ever had room to wrap.
	swung := uint64(power) + uint64(bonus)
	multiplier := uint64(PermilleBase) + uint64(share)
	return int(widen(swung, multiplier).over(PermilleBase))
}
