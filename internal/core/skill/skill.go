// Package skill is where the rest of the engine meets: a skill names an element,
// a shape, a power, a scaling stat, the statuses it inflicts and the condition
// that makes it hit harder. Everything it names is validated against the book
// that declares it, so a typo in a data file fails at load rather than at the
// moment it would have mattered.
//
// Nothing here computes damage. A skill is a declaration; turning one into a
// combat.Hit is the battle layer's job, because only the battle knows the
// caster's current stats, the target's, and the state of the board.
package skill

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/status"
)

// Side is who a skill is aimed at.
type Side uint8

const (
	// Enemy aims across the battlefield.
	Enemy Side = iota
	// Ally aims at the caster's own half, the caster included.
	Ally
	// Self aims only at the caster.
	Self
)

// SideCount is the number of targeting sides.
const SideCount = int(Self) + 1

var sideNames = [SideCount]string{Enemy: "enemy", Ally: "ally", Self: "self"}

func (s Side) String() string {
	if int(s) >= SideCount {
		return fmt.Sprintf("side(%d)", uint8(s))
	}
	return sideNames[s]
}

// ParseSide resolves a targeting name as written in the data files.
func ParseSide(name string) (Side, error) {
	for i, candidate := range sideNames {
		if candidate == name {
			return Side(i), nil
		}
	}
	return 0, fmt.Errorf("unknown target side %q", name)
}

// Scaling names which stat a skill's damage comes from and which version of it
// to read.
type Scaling struct {
	Stat   progression.Kind
	Source combat.ScalingSource
}

// Application is a status a skill inflicts, with its own chance to take hold.
//
// The chance is fixed. It is not raised by accuracy, not lowered by dodge, and no
// stat touches it, so it stays a property of the skill rather than of whoever
// happens to be holding it. Landing the hit and inflicting the status are two
// separate questions: a skill that connects nine times in ten and poisons half
// the time poisons on 45 percent of casts, and both halves of that stay legible.
type Application struct {
	Status string
	Chance int
	Stacks int
}

// Condition is the amplifier that pays a skill off for arriving after a status.
type Condition struct {
	// Status and MinStacks are what the target must already be carrying.
	Status    string
	MinStacks int
	// BonusPower is added to the skill's power when the condition holds.
	BonusPower int
	// Consume removes the status, which is what a detonate does: the burst is
	// paid for by the ticks it throws away.
	Consume bool
}

// Cleanse is the statuses a skill strips.
type Cleanse struct {
	Categories []status.Category
	Stacks     int
}

// Skill is one declared action.
type Skill struct {
	ID      string
	Element element.Element
	// Range is the hex distance the skill reaches, measured on the shared board.
	Range int
	// Pattern is the shape it covers, by name in the pattern book.
	Pattern string
	// Power is the skill's strength per strike, in parts per thousand.
	Power int
	// Strikes is how many times it lands. A multi-strike skill is expected to
	// divide its power rather than repeat it at full strength.
	Strikes int
	// Accuracy is its own chance to connect, before the caster's accuracy stat
	// and the target's dodge. A full thousand cannot miss and cannot be dodged.
	Accuracy int
	Scaling  Scaling
	// Applies are the statuses inflicted on each target it hits.
	Applies []Application
	// SelfApplies are the statuses the caster gains, which is how a shield or a
	// self buff is written.
	SelfApplies []Application
	// Requires is the amplifier, if the skill has one.
	Requires *Condition
	// Strips is the cleanse or dispel the skill performs, if any.
	Strips *Cleanse
	// Cooldown is how many of the caster's own turns must pass before it can be
	// used again. Counting the caster's turns rather than cycles means a fast
	// unit really does get its skill back sooner, in step with everything else
	// that is timed.
	Cooldown int
	Target   Side
}

// TotalPower is the skill's power across every strike, which is the figure to
// compare two skills of different strike counts by.
func (s Skill) TotalPower() int { return s.Power * s.StrikeCount() }

// StrikeCount treats an unset count as one.
func (s Skill) StrikeCount() int {
	if s.Strikes < 1 {
		return 1
	}
	return s.Strikes
}

// PowerAgainst returns the power the skill lands with, given how many stacks of
// its required status the target carries.
func (s Skill) PowerAgainst(stacks int) int {
	if s.Requires == nil || stacks < s.Requires.MinStacks {
		return s.Power
	}
	return s.Power + s.Requires.BonusPower
}

// Amplified reports whether the condition holds for a target carrying the given
// number of stacks.
func (s Skill) Amplified(stacks int) bool {
	return s.Requires != nil && stacks >= s.Requires.MinStacks
}

// Guaranteed reports whether the skill cannot miss.
func (s Skill) Guaranteed() bool { return s.Accuracy >= scale.Base }

// Book is the declared skills.
type Book struct {
	skills []Skill
	byID   map[string]Skill
}

type skillFile struct {
	ID       string `json:"id"`
	Element  string `json:"element"`
	Range    int    `json:"range"`
	Pattern  string `json:"pattern"`
	Power    int    `json:"power"`
	Strikes  int    `json:"strikes"`
	Accuracy int    `json:"accuracy"`
	Scaling  *struct {
		Stat   string `json:"stat"`
		Source string `json:"source"`
	} `json:"scaling"`
	Applies     []applicationFile `json:"applies"`
	SelfApplies []applicationFile `json:"self_applies"`
	Requires    *struct {
		Status     string `json:"status"`
		MinStacks  int    `json:"min_stacks"`
		BonusPower int    `json:"bonus_power"`
		Consume    bool   `json:"consume"`
	} `json:"requires"`
	Strips *struct {
		Categories []string `json:"categories"`
		Stacks     int      `json:"stacks"`
	} `json:"strips"`
	Cooldown int    `json:"cooldown"`
	Target   string `json:"target"`
}

type applicationFile struct {
	Status string `json:"status"`
	Chance int    `json:"chance"`
	Stacks int    `json:"stacks"`
}

type bookFile struct {
	Skills []skillFile `json:"skills"`
}

// Deps are the books a skill's declarations are checked against. Validating here
// rather than at use is the whole point: a skill naming a shape or a status that
// does not exist is a data error, and a data error should stop the load.
type Deps struct {
	Patterns *pattern.Book
	Statuses *status.Book
}

// ParseBook reads a skill declaration and checks every name it uses. It never
// touches the filesystem.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if deps.Patterns == nil || deps.Statuses == nil {
		return nil, fmt.Errorf("skills cannot be validated without the pattern and status books")
	}
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode skill book: %w", err)
	}
	book := &Book{byID: make(map[string]Skill, len(file.Skills))}
	for _, declared := range file.Skills {
		built, err := resolve(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("skill %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.skills = append(book.skills, built)
	}
	if len(book.skills) == 0 {
		return nil, fmt.Errorf("the skill book is empty")
	}
	return book, nil
}

// maxRange is the longest distance on the board, so a skill cannot declare a
// reach that means nothing.
var maxRange = func() int {
	longest := 0
	for _, from := range hex.Cells() {
		for _, to := range hex.Cells() {
			if distance := from.DistanceTo(to); distance > longest {
				longest = distance
			}
		}
	}
	return longest
}()

func resolve(declared skillFile, deps Deps) (Skill, error) {
	if declared.ID == "" {
		return Skill{}, fmt.Errorf("a skill needs an id")
	}
	fail := func(format string, args ...any) (Skill, error) {
		return Skill{}, fmt.Errorf("skill %q: "+format, append([]any{declared.ID}, args...)...)
	}

	affinity, err := element.Parse(declared.Element)
	if err != nil {
		return fail("%w", err)
	}
	target, err := ParseSide(declared.Target)
	if err != nil {
		return fail("%w", err)
	}
	shape, err := deps.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return fail("%w", err)
	}

	// The order matters: a negative figure should say so rather than fall into
	// the broader "does nothing" complaint, which would send an author looking
	// in the wrong place.
	switch {
	case declared.Power < 0:
		return fail("has power %d, want zero or more", declared.Power)
	case declared.Power == 0 && len(declared.Applies) == 0 && len(declared.SelfApplies) == 0 && declared.Strips == nil:
		return fail("has no power and does nothing else, so it would be a wasted turn")
	case declared.Strikes < 0:
		return fail("has %d strikes, want zero or more", declared.Strikes)
	case declared.Accuracy < 0 || declared.Accuracy > scale.Base:
		return fail("has accuracy %d, want a share in parts per thousand", declared.Accuracy)
	case declared.Power > 0 && declared.Accuracy == 0:
		return fail("deals damage but can never connect")
	case declared.Cooldown < 0:
		return fail("has cooldown %d, want zero or more", declared.Cooldown)
	}

	// A self-targeted skill has nowhere to reach and no shape to spread over.
	if target == Self {
		if declared.Range != 0 {
			return fail("targets itself but declares a range of %d", declared.Range)
		}
		if shape.MaxTargets() != 1 {
			return fail("targets itself but uses the %q shape, which covers %d cells",
				shape.Name, shape.MaxTargets())
		}
	} else if declared.Range < 1 || declared.Range > maxRange {
		return fail("has range %d, want between 1 and %d", declared.Range, maxRange)
	}

	scaling := Scaling{Stat: progression.Attack, Source: combat.CurrentStat}
	if declared.Scaling != nil {
		stat, err := parseStat(declared.Scaling.Stat)
		if err != nil {
			return fail("%w", err)
		}
		if stat == progression.HP {
			return fail("scales off health, which would make damage grow as the caster is healed")
		}
		source := combat.CurrentStat
		switch declared.Scaling.Source {
		case "", "current":
		case "base":
			source = combat.BaseStat
		default:
			return fail("scales off the %q value, want \"base\" or \"current\"", declared.Scaling.Source)
		}
		scaling = Scaling{Stat: stat, Source: source}
	}

	applies, err := resolveApplications(declared.ID, "applies", declared.Applies, deps)
	if err != nil {
		return Skill{}, err
	}
	selfApplies, err := resolveApplications(declared.ID, "self_applies", declared.SelfApplies, deps)
	if err != nil {
		return Skill{}, err
	}

	var requires *Condition
	if declared.Requires != nil {
		kind, err := deps.Statuses.Lookup(declared.Requires.Status)
		if err != nil {
			return fail("condition: %w", err)
		}
		minStacks := declared.Requires.MinStacks
		if minStacks < 1 {
			minStacks = 1
		}
		if minStacks > kind.MaxStacks {
			return fail("condition needs %d stacks of %q, which caps at %d",
				minStacks, kind.ID, kind.MaxStacks)
		}
		if declared.Requires.BonusPower < 0 {
			return fail("condition adds %d power, want zero or more", declared.Requires.BonusPower)
		}
		if declared.Requires.Consume && declared.Requires.BonusPower == 0 {
			return fail("condition consumes %q for no bonus, which throws the status away for nothing", kind.ID)
		}
		requires = &Condition{
			Status: kind.ID, MinStacks: minStacks,
			BonusPower: declared.Requires.BonusPower, Consume: declared.Requires.Consume,
		}
	}

	var strips *Cleanse
	if declared.Strips != nil {
		if len(declared.Strips.Categories) == 0 {
			return fail("strips nothing, because it names no categories")
		}
		if declared.Strips.Stacks < 1 {
			return fail("strips %d stacks, want at least 1", declared.Strips.Stacks)
		}
		categories := make([]status.Category, 0, len(declared.Strips.Categories))
		seen := make(map[status.Category]bool, len(declared.Strips.Categories))
		for _, name := range declared.Strips.Categories {
			category, err := status.ParseCategory(name)
			if err != nil {
				return fail("strips: %w", err)
			}
			if seen[category] {
				return fail("strips the %s category twice", category)
			}
			seen[category] = true
			categories = append(categories, category)
		}
		strips = &Cleanse{Categories: categories, Stacks: declared.Strips.Stacks}
	}

	return Skill{
		ID: declared.ID, Element: affinity, Range: declared.Range, Pattern: shape.Name,
		Power: declared.Power, Strikes: declared.Strikes, Accuracy: declared.Accuracy,
		Scaling: scaling, Applies: applies, SelfApplies: selfApplies,
		Requires: requires, Strips: strips, Cooldown: declared.Cooldown, Target: target,
	}, nil
}

func resolveApplications(skillID, field string, declared []applicationFile, deps Deps) ([]Application, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	out := make([]Application, 0, len(declared))
	seen := make(map[string]bool, len(declared))
	for _, entry := range declared {
		kind, err := deps.Statuses.Lookup(entry.Status)
		if err != nil {
			return nil, fmt.Errorf("skill %q %s: %w", skillID, field, err)
		}
		if seen[kind.ID] {
			return nil, fmt.Errorf("skill %q %s: %q appears twice", skillID, field, kind.ID)
		}
		seen[kind.ID] = true
		if entry.Chance < 1 || entry.Chance > scale.Base {
			return nil, fmt.Errorf("skill %q %s: %q has a chance of %d, want between 1 and %d",
				skillID, field, kind.ID, entry.Chance, scale.Base)
		}
		stacks := entry.Stacks
		if stacks < 1 {
			stacks = 1
		}
		if stacks > kind.MaxStacks {
			return nil, fmt.Errorf("skill %q %s: applies %d stacks of %q, which caps at %d",
				skillID, field, stacks, kind.ID, kind.MaxStacks)
		}
		out = append(out, Application{Status: kind.ID, Chance: entry.Chance, Stacks: stacks})
	}
	return out, nil
}

func parseStat(name string) (progression.Kind, error) {
	for _, kind := range progression.Kinds() {
		if kind.String() == name {
			return kind, nil
		}
	}
	return 0, fmt.Errorf("unknown scaling stat %q", name)
}

// Skills returns every declared skill in declaration order.
func (b *Book) Skills() []Skill {
	out := make([]Skill, len(b.skills))
	copy(out, b.skills)
	return out
}

// Lookup returns a skill by id.
func (b *Book) Lookup(id string) (Skill, error) {
	found, ok := b.byID[id]
	if !ok {
		return Skill{}, fmt.Errorf("unknown skill %q", id)
	}
	return found, nil
}
