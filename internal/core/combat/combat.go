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
package combat

import (
	"encoding/json"
	"fmt"

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
func (r Rules) Damage(attack, defense int64, skillMultiplier, affinityMultiplier int) int64 {
	if attack <= 0 || skillMultiplier <= 0 || affinityMultiplier <= 0 {
		return 0
	}
	if defense < 0 {
		defense = 0
	}
	numerator := attack * int64(skillMultiplier) * int64(affinityMultiplier) * r.DefenseConstant
	denominator := int64(PermilleBase) * int64(PermilleBase) * (r.DefenseConstant + defense)
	damage := numerator / denominator
	if damage < r.MinimumDamage {
		return r.MinimumDamage
	}
	return damage
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
	// Defense is the target's defence, already buffed.
	Defense int64
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
	damage := r.Strike(h)
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

// Strike returns the damage of a single strike.
func (r Rules) Strike(h Hit) int64 {
	return r.Damage(h.Scaling, h.Defense, h.Multiplier, h.Affinity)
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

// Total returns the damage of every strike combined.
func (r Rules) Total(h Hit) int64 {
	return r.Strike(h) * int64(h.StrikeCount())
}
