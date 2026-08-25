package forge

import (
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// This file is the authoring side of a skill, on exactly the terms the rest of
// this package holds a character: the answers arrive as the strings a flag, a
// prompt or a text field produced, Resolve turns them into a skill.Skill, and
// the last word belongs to skill.ParseBook, which the candidate is appended
// through so nothing can be produced here that would not load.
//
// # What the form authors and what it does not
//
// The nine core fields, the restriction and the statuses a skill inflicts. Not
// requires, strips, scaling or self_applies: each of those is a composite worth
// several questions on its own, and a wizard that asked twelve of them would be
// worse than an edit to the file. Those four survive a save untouched, which is
// what Skill.MarshalJSON going through the parse shape guarantees and what
// TestTheShippedSkillBookSurvivesBeingWritten measures.

// SkillDraft is every answer that makes a skill, held as text until Resolve.
//
// Text rather than numbers for the same reason Draft holds text: a flag-only
// invocation, a prompted one and a form driven by keystrokes have to turn into
// the same skill, and they only do if they meet at one parse. The alternative —
// typed flags — would leave the command line refusing a bad power in the flag
// package's words and the form refusing it in this package's, which is one rule
// worded twice.
type SkillDraft struct {
	ID       string
	Element  string
	Target   string
	Range    string
	Pattern  string
	Power    string
	Strikes  string
	Accuracy string
	Cooldown string
	// Applies is the statuses the skill inflicts, comma separated, each written
	// "status:chance" or "status:chance:stacks".
	Applies string
	// The three allowlists, comma separated. An empty answer is an absent list,
	// which is what makes the common pool the default shape: a skill authored
	// with nothing filled in here is written with no restrict block at all,
	// rather than with an empty one, which would be the error case.
	RestrictElements   string
	RestrictArchetypes string
	RestrictCharacters string
}

// Resolve turns a draft into a skill, or says which answer is wrong.
//
// The candidate is appended to a copy of the book, which validates it exactly as
// loading the file would — every bound on a power, an accuracy or a stack count
// is skill.ParseBook's, and none of them is restated here.
func (d SkillDraft) Resolve(lib *Library) (skill.Skill, error) {
	if err := lib.ValidateNewSkillID(d.ID); err != nil {
		return skill.Skill{}, err
	}
	member, err := ParseElement(d.Element)
	if err != nil {
		return skill.Skill{}, err
	}
	target, err := ParseTarget(d.Target)
	if err != nil {
		return skill.Skill{}, err
	}
	shape, err := lib.LookupPattern(d.Pattern)
	if err != nil {
		return skill.Skill{}, err
	}
	numbers := map[string]int{}
	for _, field := range []struct {
		name   string
		answer string
	}{
		{"range", d.Range}, {"power", d.Power}, {"strikes", d.Strikes},
		{"accuracy", d.Accuracy}, {"cooldown", d.Cooldown},
	} {
		// A map written and read by key, never ranged over into an output: the
		// fields below are named one at a time.
		value, err := ParseNumber(field.answer)
		if err != nil {
			return skill.Skill{}, err
		}
		numbers[field.name] = value
	}
	applies, err := lib.ParseApplications(d.Applies)
	if err != nil {
		return skill.Skill{}, err
	}
	restrict, err := d.Restriction()
	if err != nil {
		return skill.Skill{}, err
	}

	built := skill.Skill{
		ID: strings.TrimSpace(d.ID), Element: member, Target: target,
		Range: numbers["range"], Pattern: shape.Name,
		Power: numbers["power"], Strikes: numbers["strikes"],
		Accuracy: numbers["accuracy"], Cooldown: numbers["cooldown"],
		// The default has to be set rather than left zero: the zero stat is
		// health, and a skill scaling off health is refused. See
		// skill.DefaultScaling.
		Scaling: skill.DefaultScaling(),
		Applies: applies, Restrict: restrict,
	}
	grown, err := lib.skills.Append(lib.SkillDeps(), built)
	if err != nil {
		return skill.Skill{}, err
	}
	// What comes back is the parser's skill and not the one built above, even
	// though the two describe the same thing. Append marshals and re-parses, so
	// this is the value a reload will produce — down to an omitted list being
	// nil rather than an empty slice, which is the difference that made the
	// round-trip comparison in TestTheSkillFormProducesTheSkillTheCommandLine
	// Produces fail on two skills that printed identically. Returning the parsed
	// one means "what Resolve produced" and "what the file holds" cannot differ
	// in a way only reflect.DeepEqual can see.
	return grown.Lookup(built.ID)
}

// Restriction is the three allowlists a draft names, or nil when it names none.
//
// Nil rather than an empty block is the whole of the common pool being the
// default: skill.ParseBook refuses a restrict block that restricts nothing, so a
// form that always wrote one would refuse every ordinary skill.
func (d SkillDraft) Restriction() (*skill.Restriction, error) {
	elements := SplitList(d.RestrictElements)
	archetypes := SplitList(d.RestrictArchetypes)
	characters := SplitList(d.RestrictCharacters)
	if len(elements) == 0 && len(archetypes) == 0 && len(characters) == 0 {
		return nil, nil
	}
	restriction := &skill.Restriction{Archetypes: archetypes, Characters: characters}
	for _, name := range elements {
		member, err := ParseElement(name)
		if err != nil {
			return nil, err
		}
		restriction.Elements = append(restriction.Elements, member)
	}
	return restriction, nil
}

// ParseElement reads one element name, which is what a chooser and a
// restriction list both hand over.
//
// It is ParseAffinity's single-element half rather than a second reading of a
// name: an affinity is one element or two, and a skill's element is always one.
func ParseElement(raw string) (element.Element, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return element.Neutral, &MissingElementError{}
	}
	member, err := element.Parse(name)
	if err != nil {
		return element.Neutral, &UnknownElementError{Name: name, Err: err}
	}
	return member, nil
}

// ParseTarget reads a targeting side by name, which is what the data files
// write and what --target takes.
func ParseTarget(raw string) (skill.Side, error) {
	name := strings.TrimSpace(raw)
	side, err := skill.ParseSide(name)
	if err != nil {
		return 0, &UnknownTargetError{Name: name, Err: err}
	}
	return side, nil
}

// ParseNumber reads a whole number. An empty answer is zero, which is what
// every one of a skill's numbers means when it is left out of the file.
func ParseNumber(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, &NumberError{Raw: trimmed, Err: err}
	}
	return value, nil
}

// ParseApplications reads the statuses a skill inflicts, written
// "status:chance" or "status:chance:stacks" and separated by commas.
//
// Whether a chance is a legal share and whether a stack count fits the status is
// skill.ParseBook's judgement and is not repeated here. What this owns is the
// shape of the answer and whether the status exists, which is the part a form
// can refuse as it is typed.
func (l *Library) ParseApplications(answer string) ([]skill.Application, error) {
	entries := SplitList(answer)
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]skill.Application, 0, len(entries))
	for _, entry := range entries {
		parts := strings.Split(entry, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, &ApplicationShapeError{Raw: entry}
		}
		id := strings.TrimSpace(parts[0])
		kind, err := l.statuses.Lookup(id)
		if err != nil {
			return nil, &UnknownStatusError{ID: id, Err: err}
		}
		chance, err := ParseNumber(parts[1])
		if err != nil {
			return nil, err
		}
		stacks := 1
		if len(parts) == 3 {
			if stacks, err = ParseNumber(parts[2]); err != nil {
				return nil, err
			}
		}
		out = append(out, skill.Application{Status: kind.ID, Chance: chance, Stacks: stacks})
	}
	return out, nil
}

// FormatApplications is ParseApplications' inverse, which is what a prefilled
// field and a listing both need, so that accepting a field as it stands
// reproduces what was in it.
func FormatApplications(applications []skill.Application) string {
	parts := make([]string, 0, len(applications))
	for _, application := range applications {
		written := application.Status + ":" + strconv.Itoa(application.Chance)
		if application.Stacks > 1 {
			written += ":" + strconv.Itoa(application.Stacks)
		}
		parts = append(parts, written)
	}
	return strings.Join(parts, ",")
}

// ValidateNewSkillID rejects a skill with no id or one the book already holds.
//
// It checks no shape beyond that, and the omission is deliberate:
// skill.ParseBook asks only that an id is not empty, and the shipped ids are
// written with underscores, so a slug rule imposed here would refuse a name the
// game accepts.
func (l *Library) ValidateNewSkillID(id string) error {
	if strings.TrimSpace(id) == "" {
		return &MissingSkillIDError{}
	}
	if _, known := l.skills.Lookup(strings.TrimSpace(id)); known == nil {
		return &SkillTakenError{ID: strings.TrimSpace(id)}
	}
	return nil
}

// LookupPattern resolves a shape name against the pattern book.
func (l *Library) LookupPattern(name string) (pattern.Pattern, error) {
	shape, err := l.patterns.Lookup(strings.TrimSpace(name))
	if err != nil {
		return shape, &UnknownPatternError{Name: strings.TrimSpace(name), Err: err}
	}
	return shape, nil
}

// PatternNames is every declared shape, in the book's own order, which is what
// a chooser offers.
func (l *Library) PatternNames() []string {
	shapes := l.patterns.Patterns()
	out := make([]string, 0, len(shapes))
	for _, shape := range shapes {
		out = append(out, shape.Name)
	}
	return out
}

// ElementNames is every declared element's name, in the order the enum declares
// them, which is what a chooser and an allowlist picker offer.
func ElementNames() []string {
	members := element.All()
	out := make([]string, 0, len(members))
	for _, member := range members {
		out = append(out, member.String())
	}
	return out
}

// TargetNames is the three sides a skill can aim at, by the names the data files
// write and --target takes.
func TargetNames() []string {
	out := make([]string, 0, skill.SideCount)
	for i := range skill.SideCount {
		out = append(out, skill.Side(i).String())
	}
	return out
}

// CharacterIDs is the authored cast's ids, in the book's own order, which is
// what the character allowlist is chosen from.
func (l *Library) CharacterIDs() []string {
	characters := l.characters.All()
	out := make([]string, 0, len(characters))
	for _, character := range characters {
		out = append(out, character.ID)
	}
	return out
}

// StatusIDs is every declared status, in the book's own order, which is what a
// hint beside the applies field lists.
func (l *Library) StatusIDs() []string {
	kinds := l.statuses.Kinds()
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, kind.ID)
	}
	return out
}

// SkillPreview is what a skill would do, measured before it is written.
//
// A skill is balance rather than content, so the figure that matters is not
// whether the answers parse but what they are worth against a reference target.
// Without it an author picks a power, saves, and finds out at the next test run
// that a dozen golden numbers moved.
type SkillPreview struct {
	// Attack and Defense are the reference figures the damage is measured
	// against, carried so a front-end can name them rather than assume them.
	Attack, Defense int64
	// PerStrike is one strike's damage and Total is every strike's, which is
	// the figure to compare two skills of different strike counts by.
	PerStrike, Total int64
	Strikes          int
	// Amplified is Total with the skill's condition holding, or zero when the
	// skill has no condition. The authoring form cannot write a condition, but
	// it can edit the power of a skill that has one.
	Amplified int64
}

// PreviewDamage measures a skill against a reference attacker and defender.
//
// The two figures are the shipped progression ceilings: the attack ceiling
// exactly, and half the defence ceiling. That is not a round number picked to
// look reasonable — it is the pair skills.golden's own damage column is computed
// from (800 attack, 400 defence), so the number this shows before a write is the
// number the golden table will show after one. A different reference would give
// an author two damage figures for one skill and no way to tell which was the
// one the design was read from.
//
// Half the defence ceiling rather than all of it because the ceiling is the
// armour end of the budget, which progression.Limits already bounds jointly with
// health: measuring every skill against a defender nobody can legally be would
// flatten the differences between skills, which is the one thing this figure is
// for.
//
// A neutral matchup, and accuracy left out of it, for the same reason: it is
// what the golden's column does. The skill's own accuracy is a field on the form
// directly above this line, so an author reads the two together.
func (l *Library) PreviewDamage(built skill.Skill) SkillPreview {
	attack := l.limits.Ceilings[progression.Attack]
	defense := l.limits.Ceilings[progression.Defense] / 2
	preview := SkillPreview{
		Attack: attack, Defense: defense, Strikes: built.StrikeCount(),
	}
	// Per strike and then multiplied, rather than the total power in one call:
	// truncation happens once per strike in a battle, and a preview that
	// truncated once would read a few points high on a multi-strike skill.
	preview.PerStrike = l.rules.Damage(attack, defense, built.Power, combat.PermilleBase)
	preview.Total = preview.PerStrike * int64(preview.Strikes)
	if built.Requires != nil {
		amplified := l.rules.Damage(attack, defense,
			built.PowerAgainst(built.Requires.MinStacks), combat.PermilleBase)
		preview.Amplified = amplified * int64(preview.Strikes)
	}
	return preview
}

// CarryFacts is who may take a skill, as the facts rather than as a sentence.
//
// Four things can narrow a skill and they compose: its own element gates it to
// affinities holding that element, and each of the three allowlists gates it
// further. Nothing narrowing it at all is the common pool, which is what Anyone
// says, and it is worth its own field rather than a caller checking for four
// empty ones — "which skills can anybody take" is the question an author asks
// most often and the one the picker answers at a glance.
type CarryFacts struct {
	// Anyone is true when nothing narrows the skill.
	Anyone bool
	// Element is the skill's own element, empty when it is neutral and
	// therefore gates nobody.
	Element string
	// Elements, Archetypes and Characters are the allowlists it declares.
	Elements   []string
	Archetypes []string
	Characters []string
}

// WhoMayCarry reads a skill's gates without wording any of them.
func WhoMayCarry(carried skill.Skill) CarryFacts {
	facts := CarryFacts{
		Elements:   carried.Restrict.ElementNames(),
		Archetypes: append([]string(nil), restrictedArchetypes(carried)...),
		Characters: append([]string(nil), restrictedCharacters(carried)...),
	}
	if carried.Element != element.Neutral {
		facts.Element = carried.Element.String()
	}
	facts.Anyone = AnyoneMayCarry(carried)
	return facts
}

func restrictedArchetypes(carried skill.Skill) []string {
	if carried.Restrict == nil {
		return nil
	}
	return carried.Restrict.Archetypes
}

func restrictedCharacters(carried skill.Skill) []string {
	if carried.Restrict == nil {
		return nil
	}
	return carried.Restrict.Characters
}

// WhoMaySummary is WhoMayCarry as the English cmd/hexforge prints.
func WhoMaySummary(carried skill.Skill) string {
	facts := WhoMayCarry(carried)
	if facts.Anyone {
		return "anyone"
	}
	parts := make([]string, 0, 4)
	if facts.Element != "" {
		parts = append(parts, facts.Element+" units")
	}
	if len(facts.Elements) > 0 {
		parts = append(parts, "kept for "+strings.Join(facts.Elements, " or "))
	}
	if len(facts.Archetypes) > 0 {
		parts = append(parts, "kept for the "+strings.Join(facts.Archetypes, " or ")+" role")
	}
	if len(facts.Characters) > 0 {
		parts = append(parts, "belongs to "+strings.Join(facts.Characters, " or "))
	}
	return strings.Join(parts, ", ")
}

// PreviewDraft is the damage a half-finished draft is worth.
//
// Only the power and the strike count reach the figure, which is why this works
// on a form nobody has named yet: the two are the balance, and the rest of the
// answers decide who may use it and when. The refusal is ParseNumber's, so a
// power that is not a number reads the same here as it does at the write.
func (l *Library) PreviewDraft(d SkillDraft) (SkillPreview, error) {
	power, err := ParseNumber(d.Power)
	if err != nil {
		return SkillPreview{}, err
	}
	strikes, err := ParseNumber(d.Strikes)
	if err != nil {
		return SkillPreview{}, err
	}
	return l.PreviewDamage(skill.Skill{Power: power, Strikes: strikes}), nil
}

// AnyoneMayCarry reports whether every character may take a skill: it is the
// common pool, and it is the shape a skill has by default.
//
// Two conditions, and both are already the rule rather than a new one:
// skill.CanCarry demands a shared element only of a skill that has one, so a
// neutral skill is carryable by any affinity, and an absent restriction admits
// everybody. It lives here rather than being spelled out in a front-end so that
// "which skills can anybody take" has one answer for the picker, the listing and
// the prompt.
func AnyoneMayCarry(carried skill.Skill) bool {
	return carried.Element == element.Neutral && carried.Restrict == nil
}
