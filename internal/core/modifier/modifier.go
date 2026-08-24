// Package modifier holds the buff and debuff layer: the temporary terms that
// sit between a unit's resolved stat line and the numbers a hit is calculated
// from.
//
// Two decisions here shape everything downstream.
//
// Flat terms apply before percentage terms, so a stat resolves as
// (base + flat) * (1000 + percent) / 1000. The alternative order lets a small
// flat buff be multiplied by a large percentage buff, which makes the two
// kinds of term impossible to price independently.
//
// Percentage terms of the same target add rather than compose. Three separate
// fifty-percent buffs come to +150%, not to 3.375x. Composition looks harmless
// on two buffs and explodes on four, and a battle where several supports stack
// on one carry is exactly where four happens.
//
// The summed terms are then saturated rather than clamped, so a stat approaches
// its limit without ever reaching it and each further buff is worth less than
// the last. See scale.Saturate. A consequence worth knowing: because a large
// term saturates rather than overflowing, a debuff can safely be authored well
// past a hundred percent, and a modest buff lands close to its face value.
//
// Everything is integer arithmetic on parts per thousand, so a battle replays
// identically from its seed.
package modifier

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// Target names what a modifier changes.
type Target uint8

const (
	HP       = Target(progression.HP)
	Attack   = Target(progression.Attack)
	Defense  = Target(progression.Defense)
	Speed    = Target(progression.Speed)
	Accuracy = Target(progression.Accuracy)
	Dodge    = Target(progression.Dodge)
	// Affinity is the effectiveness of an elemental advantage rather than a
	// stat, so it sits just past the stat range and shares the same indexing.
	Affinity = Target(progression.KindCount)
)

// TargetCount is the number of modifier targets.
const TargetCount = int(Affinity) + 1

var targetNames = [TargetCount]string{
	HP:       "hp",
	Attack:   "attack",
	Defense:  "defense",
	Speed:    "speed",
	Accuracy: "accuracy",
	Dodge:    "dodge",
	Affinity: "affinity",
}

func (t Target) String() string {
	if int(t) >= TargetCount {
		return fmt.Sprintf("target(%d)", uint8(t))
	}
	return targetNames[t]
}

// Valid reports whether the value is a declared target.
func (t Target) Valid() bool { return int(t) < TargetCount }

// IsStat reports whether the target is one of the four stats.
func (t Target) IsStat() bool { return int(t) < progression.KindCount }

// Stat returns the stat this target refers to. It is only meaningful when
// IsStat reports true.
func (t Target) Stat() progression.Kind { return progression.Kind(t) }

// ParseTarget resolves a target name as written in the data files.
func ParseTarget(name string) (Target, error) {
	for i, candidate := range targetNames {
		if candidate == name {
			return Target(i), nil
		}
	}
	return 0, fmt.Errorf("unknown modifier target %q", name)
}

// Targets returns every target in declaration order.
func Targets() []Target {
	out := make([]Target, 0, TargetCount)
	for i := 0; i < TargetCount; i++ {
		out = append(out, Target(i))
	}
	return out
}

// Mode is how a modifier's amount is read.
type Mode uint8

const (
	// Flat adds the amount to the value directly.
	Flat Mode = iota
	// Percent adds the amount, in parts per thousand, to the value's scale.
	// An amount of 300 is a thirty percent increase, -300 a thirty percent cut.
	Percent
)

var modeNames = []string{Flat: "flat", Percent: "percent"}

func (m Mode) String() string {
	if int(m) >= len(modeNames) {
		return fmt.Sprintf("mode(%d)", uint8(m))
	}
	return modeNames[m]
}

// ParseMode resolves a mode name as written in the data files.
func ParseMode(name string) (Mode, error) {
	for i, candidate := range modeNames {
		if candidate == name {
			return Mode(i), nil
		}
	}
	return 0, fmt.Errorf("unknown modifier mode %q", name)
}

// Modifier is one buff or debuff term. A negative amount is a debuff; there is
// no separate type for one, because every rule that bounds a buff has to bound
// a debuff by the same reasoning.
type Modifier struct {
	Target Target
	Mode   Mode
	Amount int64
}

// Validate rejects a term that cannot be applied.
func (m Modifier) Validate() error {
	if !m.Target.Valid() {
		return fmt.Errorf("modifier targets %s, which is not declared", m.Target)
	}
	if m.Target == Affinity && m.Mode != Percent {
		// The affinity term scales how far a matchup deviates from neutral, so
		// it is inherently proportional. A flat term would add the same amount
		// to a single weakness and to a stacked one, which makes it worth less
		// against the very matchups it is meant to reward.
		return fmt.Errorf("an affinity modifier must be a percentage, not %s", m.Mode)
	}
	if m.Amount == 0 {
		return fmt.Errorf("modifier on %s has no amount", m.Target)
	}
	return nil
}

func (m Modifier) String() string {
	sign := ""
	if m.Amount > 0 {
		sign = "+"
	}
	if m.Mode == Percent {
		return fmt.Sprintf("%s %s%d.%d%%", m.Target, sign, m.Amount/10, abs(m.Amount)%10)
	}

	return fmt.Sprintf("%s %s%d", m.Target, sign, m.Amount)
}

type modifierFile struct {
	Target string `json:"target"`
	Mode   string `json:"mode"`
	Amount int64  `json:"amount"`
}

// UnmarshalJSON reads a modifier written as {"target":..,"mode":..,"amount":..}.
func (m *Modifier) UnmarshalJSON(raw []byte) error {
	var file modifierFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("decode modifier: %w", err)
	}
	target, err := ParseTarget(file.Target)
	if err != nil {
		return err
	}
	mode, err := ParseMode(file.Mode)
	if err != nil {
		return err
	}
	built := Modifier{Target: target, Mode: mode, Amount: file.Amount}
	if err := built.Validate(); err != nil {
		return err
	}
	*m = built
	return nil
}

// MarshalJSON writes the modifier by name.
func (m Modifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(modifierFile{Target: m.Target.String(), Mode: m.Mode.String(), Amount: m.Amount})
}

// Bounds are the limits accumulated modifiers saturate towards.
type Bounds struct {
	// Headroom sets the value a buffed stat approaches, in parts per thousand
	// of its progression ceiling. It is never reached, so it is a limit rather
	// than a cap; the larger it is, the closer a single buff lands to its face
	// value and the more room a stack has before it flattens out.
	Headroom int64 `json:"headroom"`
	// FloorFraction sets the value a debuffed stat approaches, in parts per
	// thousand of the stat's unmodified value. It is never reached either, so a
	// stat can be crushed but never removed.
	FloorFraction int64 `json:"floor_fraction"`
	// MaxAffinityScale caps how far an elemental advantage can be scaled, in
	// parts per thousand, in either direction.
	//
	// This one is a hard clamp rather than a saturation, unlike every stat.
	// Saturating from zero would take a haircut off even a single term, because
	// the limit here is the same order of magnitude as the terms themselves,
	// whereas a stat's limit is far above its usual value.
	MaxAffinityScale int64 `json:"max_affinity_scale"`
}

// PercentBase is the denominator percentage terms are expressed against.
const PercentBase = scale.Base

// ParseBounds reads a bounds declaration. It never touches the filesystem.
func ParseBounds(raw []byte) (Bounds, error) {
	var bounds Bounds
	if err := json.Unmarshal(raw, &bounds); err != nil {
		return Bounds{}, fmt.Errorf("decode modifier bounds: %w", err)
	}
	if err := bounds.Validate(); err != nil {
		return Bounds{}, err
	}
	return bounds, nil
}

// Validate checks the bounds themselves are coherent.
func (b Bounds) Validate() error {
	switch {
	case b.Headroom <= PercentBase:
		return fmt.Errorf("headroom is %d, want more than %d so a buff has somewhere to go", b.Headroom, PercentBase)
	case b.FloorFraction <= 0:
		return fmt.Errorf("floor_fraction is %d, want a positive value", b.FloorFraction)
	case b.FloorFraction >= PercentBase:
		return fmt.Errorf("floor_fraction is %d, want less than %d so a debuff has somewhere to go", b.FloorFraction, PercentBase)
	case b.MaxAffinityScale < 0:
		return fmt.Errorf("max_affinity_scale is %d, want zero or more", b.MaxAffinityScale)
	}
	return nil
}

// Set is the accumulated modifiers on one unit. The zero value is an
// unmodified unit.
type Set struct {
	flat    [TargetCount]int64
	percent [TargetCount]int64
}

// Add accumulates one term. Terms of the same target and mode sum.
func (s *Set) Add(m Modifier) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if m.Mode == Percent {
		s.percent[m.Target] += m.Amount
		return nil
	}
	s.flat[m.Target] += m.Amount
	return nil
}

// AddAll accumulates several terms, stopping at the first invalid one.
func (s *Set) AddAll(modifiers ...Modifier) error {
	for _, m := range modifiers {
		if err := s.Add(m); err != nil {
			return err
		}
	}
	return nil
}

// Flat returns the summed flat term for a target.
func (s Set) Flat(target Target) int64 {
	if !target.Valid() {
		return 0
	}
	return s.flat[target]
}

// Percent returns the summed percentage term for a target, before clamping.
func (s Set) Percent(target Target) int64 {
	if !target.Valid() {
		return 0
	}
	return s.percent[target]
}

// Raw returns what a stat would be with no saturation: flat terms first, then
// the summed percentage. It exists so a report can show what the saturation
// actually absorbed.
func (s Set) Raw(kind progression.Kind, base int64) int64 {
	target := Target(kind)
	return (base + s.Flat(target)) * (PercentBase + s.Percent(target)) / PercentBase
}

// Stat resolves one stat. Flat terms apply first, then the summed percentage,
// and the whole change is then saturated towards the headroom limit above or
// the floor below, so the result approaches either limit without reaching it.
func (s Set) Stat(kind progression.Kind, base int64, ceiling int64, bounds Bounds) int64 {
	raw := s.Raw(kind, base)
	limit := ceiling * bounds.Headroom / PercentBase
	floor := base * bounds.FloorFraction / PercentBase
	value := scale.Saturate(base, raw-base, limit, floor)
	if value < 1 {
		return 1
	}
	return value
}

// Stats resolves every stat against the progression ceilings.
func (s Set) Stats(base progression.Values, ceilings progression.Values, bounds Bounds) progression.Values {
	var out progression.Values
	for _, kind := range progression.Kinds() {
		out[kind] = s.Stat(kind, base[kind], ceilings[kind], bounds)
	}
	return out
}

// Affinity scales an elemental multiplier.
//
// The term multiplies how far the matchup already deviates from neutral, so it
// is worth more the worse the defender's matchup is: the same buff adds more
// against a stacked weakness than against a single one. Adding a flat amount
// instead would do the opposite, since a fixed bonus is a smaller share of a
// bigger multiplier.
//
// It only moves a multiplier that is already an advantage. A term that lifted a
// neutral or resisted matchup would let a unit buff its way out of a bad
// matchup, which is the one thing the elemental chart exists to prevent, so the
// term rewards attacking into a weakness rather than papering over a
// resistance. A negative term can strip an advantage back to neutral but never
// past it, so a debuff cannot turn a strength into a weakness.
func (s Set) Affinity(multiplier, neutral int, bounds Bounds) int {
	if multiplier <= neutral {
		return multiplier
	}
	factor := clamp(s.Percent(Affinity), -bounds.MaxAffinityScale, bounds.MaxAffinityScale)
	deviation := int64(multiplier - neutral)
	shifted := int64(neutral) + deviation*(PercentBase+factor)/PercentBase
	if shifted < int64(neutral) {
		return neutral
	}
	return int(shifted)
}

func clamp(value, low, high int64) int64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
