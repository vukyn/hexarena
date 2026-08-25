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
	"slices"
	"strings"

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
	// All aims at either half, and a shape aimed this way spreads across the
	// midline instead of stopping at it.
	//
	// It is what a skill that does not care whose side a unit is on needs: a
	// battlefield-wide cleanse, a haste on everybody, and — since nothing here
	// distinguishes a friendly target from a hostile one once a cell is chosen —
	// a damaging skill that hurts the caster's own squad as well. That last one
	// is the point of having the value rather than a flaw in it; resolveAgainst
	// has never asked which side a target is on, so an ally-aimed damaging skill
	// already did the same thing.
	//
	// Declared last on purpose. The value is serialised by name like every
	// other enum here, so appending cannot reinterpret a skill book or a saved
	// log — but the numbering is what SideCount and TargetNames are built from,
	// and putting a new value in the middle would reorder the chooser a
	// front-end offers.
	All
)

// SideCount is the number of targeting sides.
const SideCount = int(All) + 1

var sideNames = [SideCount]string{Enemy: "enemy", Ally: "ally", Self: "self", All: "all"}

func (s Side) String() string {
	if int(s) >= SideCount {
		return fmt.Sprintf("side(%d)", uint8(s))
	}
	return sideNames[s]
}

// Reaches reports whether a skill aimed by a unit on one side of the board may
// be pointed at a cell on another.
//
// The rule is relational rather than absolute — "the other side" depends on
// whose turn it is — and it lives here because it is one rule with two callers:
// the legal aims a unit is offered and the aim an action is checked against.
// Before, battle worked out an absolute side and then flipped it for an enemy
// caster, which is the same fact written as a special case.
//
// Self is not asked here and answers no. It reaches exactly the caster's own
// cell, which is a unit rather than a side, so nothing can decide it from two
// sides alone; battle answers that one before it asks this.
func (s Side) Reaches(from, at hex.Side) bool {
	switch s {
	case All:
		return true
	case Ally:
		return at == from
	case Self:
		return false
	default:
		return at != from
	}
}

// CrossesSides reports whether a shape aimed by this skill may spread onto the
// other half of the board.
//
// pattern.Targets drops a splash cell that lands on the far side of the midline,
// which is right for every skill that aims at one side: an area attack on the
// enemy front rank must not catch the caster's own. A skill aimed at both sides
// is the one case where that drop is wrong, and this is what says so — once,
// for the two places that walk a shape.
func (s Side) CrossesSides() bool { return s == All }

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

// Restriction is who a skill declares itself available to.
//
// Every list is an allowlist and every one is optional: a list that is absent
// restricts nothing, and a skill with no restriction at all is carried by
// whoever CanCarry allows. A list that is present and empty is a data error
// rather than "unrestricted" — an allowlist nobody satisfies is a mistake every
// time, and reading it as "no restriction" would turn that mistake into a skill
// silently available to everyone.
//
// # Why two of the three are plain strings
//
// Elements are parsed here because this package already knows what an element
// is. Archetypes and characters are not: internal/core/cast declares both, and
// cast imports this package, so a skill that named a cast type would be an
// import cycle. So they travel as the ids they were written with and are
// checked one layer up, by cast.ParseBook and cast.ParseArchetypes — exactly
// the way a skill's pattern and status names are checked by whoever holds those
// books rather than by whoever declares the skill.
//
// The layering has a consequence worth stating rather than working around:
// battle.Roster carries stats, skills, an affinity and a slot, and no archetype
// and no character identity, because both are resolved before a battle starts.
// So Elements is enforceable at battle load and the other two are not. They are
// authoring-time rules, and pushing either into the engine to "complete" the
// feature would put a fact into the replayable core that no replay needs. See
// CLAUDE.md, "What a restriction can enforce".
type Restriction struct {
	// Elements is the affinities allowed to carry the skill: a unit qualifies
	// by holding any one of them.
	//
	// Any rather than all, because a unit holds at most two elements and an
	// all-of rule of more than two could never be met. The list is what makes a
	// *neutral* skill restrictable at all — CanCarry lets every affinity carry
	// a neutral skill, so a neutral skill that should belong to two elements has
	// nowhere else to say so.
	Elements []element.Element
	// Archetypes is the role presets allowed to carry it, by id in the
	// archetype book.
	Archetypes []string
	// Characters is the characters allowed to carry it, by id in the cast book.
	// A list of one is a unique skill.
	Characters []string
}

// AllowsElement reports whether the element allowlist admits an affinity.
//
// The nil receiver is the unrestricted case and answers yes, so a caller never
// has to ask whether there is a restriction before asking what it says.
func (r *Restriction) AllowsElement(affinity element.Affinity) bool {
	if r == nil || len(r.Elements) == 0 {
		return true
	}
	for _, allowed := range r.Elements {
		if affinity.Has(allowed) {
			return true
		}
	}
	return false
}

// AllowsArchetype reports whether the preset allowlist admits an id.
func (r *Restriction) AllowsArchetype(id string) bool {
	return r == nil || len(r.Archetypes) == 0 || slices.Contains(r.Archetypes, id)
}

// AllowsCharacter reports whether the character allowlist admits an id.
func (r *Restriction) AllowsCharacter(id string) bool {
	return r == nil || len(r.Characters) == 0 || slices.Contains(r.Characters, id)
}

// NamesCharacters reports whether the skill belongs to named characters, which
// is what makes it unusable in a preset shared by every character built from
// it.
func (r *Restriction) NamesCharacters() bool { return r != nil && len(r.Characters) > 0 }

// ElementNames is the element allowlist written out, which is what a refusal
// and a listing want.
func (r *Restriction) ElementNames() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Elements))
	for _, allowed := range r.Elements {
		out = append(out, allowed.String())
	}
	return out
}

// Skill is one declared action.
type Skill struct {
	ID string
	// Name is the skill's authored display name, and this package knows nothing
	// about it beyond that it is text: not what language it is in, not how wide
	// it is on a screen, not whether anything shows it. Nothing in
	// internal/core reads it — no rule branches on it and no event carries it —
	// which is what lets it be here at all.
	//
	// # Why the name is on the declaration
	//
	// A skill's Vietnamese name used to be a compiled table in internal/i18n,
	// which meant a name could not be authored by the tool that authors the
	// skill. Putting it here rather than in a translations file beside the book
	// is the same judgement `Applies` and `Restrict` are here for: a skill and
	// its name are authored in one sitting, and a second file is a second thing
	// to keep in step — one that would go stale exactly when a skill was
	// renamed or removed.
	//
	// It is absent by default and absent is a real answer: a skill with no name
	// renders as its bare id, which is the rule a data id has always followed
	// when nothing has a name for it. internal/i18n prefers this over its own
	// table, so the compiled names remain the fallback for the skills that
	// shipped before this field existed. Being absent by default is also why no
	// golden moved when it arrived — see skillFile.
	Name    string
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
	// Restrict is who may carry the skill, or nil when anybody may.
	Restrict *Restriction
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

// skillFile is the shape a skill is written in, and therefore the shape it is
// read in: Skill.MarshalJSON builds one of these rather than carrying its own
// tags, so the writer cannot describe a field the parser does not read.
//
// The field order is the order the keys are written in — the nine core fields
// first, in the order the data files have always listed them, then the optional
// blocks. Every core field is written even at its zero value, so a skill's
// first line is always complete; each optional block is omitted when it is
// absent, so a skill that declares none reads exactly as it did before any of
// them existed.
type skillFile struct {
	ID string `json:"id"`
	// Name is written only when there is one, which is what kept every golden
	// still when it was added: the shipped book declares none, so the shipped
	// file round-trips byte for byte and the tables measured from it do not move.
	// It sits beside the id because that is where it reads — a skill and the name
	// it is called by, then the numbers.
	Name        string            `json:"name,omitempty"`
	Element     string            `json:"element"`
	Range       int               `json:"range"`
	Pattern     string            `json:"pattern"`
	Power       int               `json:"power"`
	Strikes     int               `json:"strikes"`
	Accuracy    int               `json:"accuracy"`
	Cooldown    int               `json:"cooldown"`
	Target      string            `json:"target"`
	Restrict    *restrictFile     `json:"restrict,omitempty"`
	Scaling     *scalingFile      `json:"scaling,omitempty"`
	Applies     []applicationFile `json:"applies,omitempty"`
	SelfApplies []applicationFile `json:"self_applies,omitempty"`
	Requires    *conditionFile    `json:"requires,omitempty"`
	Strips      *cleanseFile      `json:"strips,omitempty"`
}

type scalingFile struct {
	Stat   string `json:"stat"`
	Source string `json:"source"`
}

type conditionFile struct {
	Status     string `json:"status"`
	MinStacks  int    `json:"min_stacks"`
	BonusPower int    `json:"bonus_power"`
	Consume    bool   `json:"consume,omitempty"`
}

type cleanseFile struct {
	Categories []string `json:"categories"`
	Stacks     int      `json:"stacks"`
}

type restrictFile struct {
	Elements   []string `json:"elements,omitempty"`
	Archetypes []string `json:"archetypes,omitempty"`
	Characters []string `json:"characters,omitempty"`
}

type applicationFile struct {
	Status string `json:"status"`
	Chance int    `json:"chance"`
	Stacks int    `json:"stacks"`
}

type bookFile struct {
	Skills []skillFile `json:"skills"`
}

// marshalFile is the shape Book.Marshal writes. It holds Skill rather than
// skillFile because Skill.MarshalJSON already turns one into the other.
type marshalFile struct {
	Skills []Skill `json:"skills"`
}

// DefaultScaling is what a skill scales off when it says nothing: the caster's
// attack as it stands, which is what almost every skill wants.
//
// It is named once because three places need it — resolve, which fills it in,
// file, which leaves it out again, and an authoring tool building a skill from
// answers — and a second copy of the pair would let a written file disagree with
// the one it was read from.
//
// It is exported because the zero Scaling is *not* this: progression.HP is the
// zero stat, and a skill scaling off health is refused, so a Skill built in Go
// without setting this field is refused the moment it is written. That refusal
// is loud on purpose. Silently substituting the default here would make the
// field's zero value mean two different things depending on where the skill came
// from.
func DefaultScaling() Scaling {
	return Scaling{Stat: progression.Attack, Source: combat.CurrentStat}
}

// file is a resolved skill as the declaration it came from.
//
// Writing goes through the parse shape rather than through tags on Skill, which
// is what makes the write lossless by construction: the only fields that can be
// written are the fields the parser reads, and a field added to one is a
// compile error in the other until it is added there too.
//
// Every optional block is written when it is present and left out when it is
// not, and the one derived default — the scaling — is left out when it is the
// default, so a file that declared none reads back exactly as it was authored.
func (s Skill) file() skillFile {
	out := skillFile{
		ID: s.ID, Name: s.Name,
		Element: s.Element.String(), Range: s.Range, Pattern: s.Pattern,
		Power: s.Power, Strikes: s.Strikes, Accuracy: s.Accuracy,
		Cooldown: s.Cooldown, Target: s.Target.String(),
		Applies: applicationFiles(s.Applies), SelfApplies: applicationFiles(s.SelfApplies),
	}
	if s.Restrict != nil {
		out.Restrict = &restrictFile{
			Elements:   s.Restrict.ElementNames(),
			Archetypes: append([]string(nil), s.Restrict.Archetypes...),
			Characters: append([]string(nil), s.Restrict.Characters...),
		}
	}
	if s.Scaling != DefaultScaling() {
		out.Scaling = &scalingFile{
			Stat: s.Scaling.Stat.String(), Source: s.Scaling.Source.String(),
		}
	}
	if s.Requires != nil {
		out.Requires = &conditionFile{
			Status: s.Requires.Status, MinStacks: s.Requires.MinStacks,
			BonusPower: s.Requires.BonusPower, Consume: s.Requires.Consume,
		}
	}
	if s.Strips != nil {
		categories := make([]string, 0, len(s.Strips.Categories))
		for _, category := range s.Strips.Categories {
			categories = append(categories, category.String())
		}
		out.Strips = &cleanseFile{Categories: categories, Stacks: s.Strips.Stacks}
	}
	return out
}

func applicationFiles(applications []Application) []applicationFile {
	if len(applications) == 0 {
		return nil
	}
	out := make([]applicationFile, 0, len(applications))
	for _, application := range applications {
		out = append(out, applicationFile{
			Status: application.Status, Chance: application.Chance, Stacks: application.Stacks,
		})
	}
	return out
}

// MarshalJSON writes a skill as the declaration a parse would read back.
func (s Skill) MarshalJSON() ([]byte, error) { return json.Marshal(s.file()) }

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

	scaling := DefaultScaling()
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

	// The name is text and this package has no opinion about it beyond that: it
	// is trimmed, so that a name of nothing but spaces is the absent answer
	// rather than a name made of spaces, and it is not measured, not checked
	// against a character set and not compared with the id.
	name := strings.TrimSpace(declared.Name)

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

	restrict, err := resolveRestriction(declared.ID, declared.Restrict)
	if err != nil {
		return Skill{}, err
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
		ID: declared.ID, Name: name,
		Element: affinity, Range: declared.Range, Pattern: shape.Name,
		Power: declared.Power, Strikes: declared.Strikes, Accuracy: declared.Accuracy,
		Scaling: scaling, Applies: applies, SelfApplies: selfApplies,
		Requires: requires, Strips: strips, Restrict: restrict,
		Cooldown: declared.Cooldown, Target: target,
	}, nil
}

// resolveRestriction checks the half of a restriction this package can see.
//
// Element names are real, no list is present-but-empty, no entry is blank and
// no entry is repeated. What it deliberately does not check is whether an
// archetype or a character id exists, because the books that declare those are
// one layer up — see Restriction. cast.ParseArchetypes and cast.ParseBook make
// that check, which is the same division as a skill's pattern and status names.
func resolveRestriction(skillID string, declared *restrictFile) (*Restriction, error) {
	if declared == nil {
		return nil, nil
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("skill %q restricts "+format, append([]any{skillID}, args...)...)
	}
	if declared.Elements == nil && declared.Archetypes == nil && declared.Characters == nil {
		return nil, fail("nothing, because it names no lists; leave the block out to restrict nothing")
	}
	// A present-but-empty list is refused rather than read as "unrestricted".
	// The two readings differ by everything — one skill nobody may carry
	// against one skill everybody may — and the wrong one is silent.
	for _, list := range []struct {
		name    string
		entries []string
	}{
		{"elements", declared.Elements},
		{"archetypes", declared.Archetypes},
		{"characters", declared.Characters},
	} {
		if list.entries != nil && len(list.entries) == 0 {
			return nil, fail("its %s to an empty list, which nobody satisfies; leave the list out to restrict nothing",
				list.name)
		}
		seen := make(map[string]bool, len(list.entries))
		for _, entry := range list.entries {
			if entry == "" {
				return nil, fail("its %s with an empty name", list.name)
			}
			if seen[entry] {
				return nil, fail("its %s to %q twice", list.name, entry)
			}
			seen[entry] = true
		}
	}
	restriction := &Restriction{
		Archetypes: append([]string(nil), declared.Archetypes...),
		Characters: append([]string(nil), declared.Characters...),
	}
	for _, name := range declared.Elements {
		member, err := element.Parse(name)
		if err != nil {
			return nil, fail("its elements: %w", err)
		}
		restriction.Elements = append(restriction.Elements, member)
	}
	return restriction, nil
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

// Marshal writes the book as a data file: two-space indented JSON, in the order
// the skills were declared.
//
// Declaration order rather than sorted by id, which is where this differs from
// cast.Book.Marshal, and the difference is the point. A cast is a set looked up
// by id, so sorting it is free and makes an addition a one-block diff. A skill
// book's order is authored information — the shipped file reads basic attacks,
// then the elemental ones, then the utility skills, and skills.golden's table is
// that order — so sorting would shuffle a design record to buy the same
// one-block diff that appending already gives.
func (b *Book) Marshal() ([]byte, error) {
	out, err := json.MarshalIndent(marshalFile{Skills: b.Skills()}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return append(out, '\n'), nil
}

// Append returns a new book holding the declared skills plus the extra ones,
// validated exactly as a parse would validate them.
//
// It works by marshalling and re-parsing rather than by re-implementing the
// checks, which is what guarantees the bytes an authoring tool is about to write
// are bytes that load.
func (b *Book) Append(deps Deps, extra ...Skill) (*Book, error) {
	raw, err := json.Marshal(marshalFile{Skills: append(b.Skills(), extra...)})
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return ParseBook(raw, deps)
}

// Replace returns a new book with one skill's declaration changed, in the place
// that skill already had.
//
// Keeping the position is the whole of it, and it is the same reason Marshal
// keeps declaration order rather than sorting: the shipped skills.json is
// committed in the form Marshal writes, and its order is authored information —
// basic attacks, then the elemental ones, then utility, which is the order
// skills.golden's table reads in. Moving an edited skill to the end would
// rewrite every line of the file and every row of that table to change one
// number, which is exactly what committing the file in Marshal form was meant to
// avoid.
//
// It validates the way Append does, by marshalling and re-parsing rather than by
// re-implementing a check, so the bytes an authoring tool is about to write are
// bytes that load. A skill the book does not already hold is refused: this
// changes a declaration and never adds one, so an id that is not there is a
// caller's mistake rather than a new skill.
//
// The id itself cannot be changed through this, because the id is what the
// position is found by. Renaming a skill has to walk every kit and every
// restriction that names the old one, which is a different operation.
func (b *Book) Replace(deps Deps, edited Skill) (*Book, error) {
	skills := b.Skills()
	at := slices.IndexFunc(skills, func(current Skill) bool { return current.ID == edited.ID })
	if at < 0 {
		return nil, fmt.Errorf("unknown skill %q", edited.ID)
	}
	skills[at] = edited
	raw, err := json.Marshal(marshalFile{Skills: skills})
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return ParseBook(raw, deps)
}

// Lookup returns a skill by id.
func (b *Book) Lookup(id string) (Skill, error) {
	found, ok := b.byID[id]
	if !ok {
		return Skill{}, fmt.Errorf("unknown skill %q", id)
	}
	return found, nil
}

// CarryRefusal is why an affinity may not carry a skill, or that it may.
//
// It is a classification rather than a sentence for the same reason
// internal/forge returns values: three callers refuse a kit and each words it
// for its own reader — the engine for a log, the parser for a data file, the
// authoring tool in the author's language — and a sentence cannot be reworded
// after it is built without being taken apart again.
type CarryRefusal uint8

const (
	// CarryAllowed is a skill the affinity may use.
	CarryAllowed CarryRefusal = iota
	// CarryWrongElement is a skill of an element the unit does not have.
	CarryWrongElement
	// CarryElementRestricted is a skill whose element allowlist excludes the
	// unit. It is a separate answer from CarryWrongElement because the two need
	// different advice: one is fixed by changing the affinity to the skill's
	// element, and the other cannot be, because the skill's own element is
	// already shared.
	CarryElementRestricted
)

// WhyCannotCarry is the whole of which skills an affinity may carry.
//
// Two conditions, in the order they are worth reporting. A unit may only use a
// skill of an element it shares — a neutral skill is universal, which is what
// makes a second element worth carrying — and it must also satisfy whatever
// element allowlist the skill declares, which is the narrower rule a
// restriction adds. Nothing else here is enforceable: an archetype and a
// character identity do not reach the engine, so those two halves of a
// restriction are checked where a character is authored. See Restriction.
//
// This is the single declaration. battle.enlist refuses a roster entry that
// breaks it, cast.ParseBook refuses an authored character that breaks it and
// forge.CheckCarry brings the same answer forward to the moment an answer is
// typed — all three by calling this, because two callers wording one rule in
// their own words is how the two come to disagree, and the disagreement here
// would be a character that writes cleanly and then cannot enter a battle.
func WhyCannotCarry(affinity element.Affinity, carried Skill) CarryRefusal {
	if carried.Element != element.Neutral && !affinity.Has(carried.Element) {
		return CarryWrongElement
	}
	if !carried.Restrict.AllowsElement(affinity) {
		return CarryElementRestricted
	}
	return CarryAllowed
}

// CanCarry reports whether a unit of the given affinity may use this skill.
// It is WhyCannotCarry for a caller that only needs the yes or no.
func CanCarry(affinity element.Affinity, carried Skill) bool {
	return WhyCannotCarry(affinity, carried) == CarryAllowed
}

// Demands returns the distinct non-neutral elements a kit requires, in the
// order the skills were listed.
//
// It is what a kit says about the affinity that may hold it: a unit must have
// every element in this set, so a kit demanding more than two can never be
// carried at all, and a kit demanding two can only be carried by exactly that
// pair. Deriving it from the skills is the point — an authored hint would be
// free to drift from the kit it described.
func Demands(kit []Skill) []element.Element {
	out := make([]element.Element, 0, 2)
	for _, carried := range kit {
		if carried.Element == element.Neutral {
			continue
		}
		if !slices.Contains(out, carried.Element) {
			out = append(out, carried.Element)
		}
	}
	return out
}
