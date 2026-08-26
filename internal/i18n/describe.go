package i18n

import (
	"strings"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Describe is what a skill does, in sentences, for somebody deciding whether to
// use it rather than somebody tuning it.
//
// # Why it is derived rather than authored
//
// Every number in it comes out of the skill itself. An authored description is
// free to drift the moment a value moves — a line reading "doubles" survives a
// bonus dropping from 1000 to 700 with nothing to catch it — and this engine
// already refused that trade once, in Archetype.Demands, which is derived from a
// kit precisely so it cannot describe a kit it no longer has. The cost is a
// uniform voice, which is the right thing to pay: a wrong number is worse than a
// flat sentence.
//
// # Why the shares rather than the damage
//
// Power is a share of the caster's scaling stat, so "100% of attack" is true
// wherever it is read and a damage figure is only true for one caster against
// one target. A player comparing two skills is comparing the shares; the figure
// they would actually deal changes with every buff between now and the strike.
//
// # Why it lives here
//
// It was written in internal/tui, beside the battle renderer that first showed
// it, and that was the wrong house: everything else in that package reads
// []Event, and this reads a skill. Here it sits beside SkillName and Gloss,
// which are the tables it borrows every data name from — one Vietnamese
// vocabulary rather than two — and it gets a second language for free, which is
// what the authoring tool needs, because that tool has a language toggle and a
// Vietnamese-only description would ignore it.
func (l Lang) Describe(declared skill.Skill, shapes *pattern.Book) string {
	lines := make([]string, 0, 4)
	if opening := l.describeOpening(declared, shapes); opening != "" {
		lines = append(lines, opening)
	}
	lines = append(lines, l.describeExtras(declared)...)
	if condition := l.describeCondition(declared); condition != "" {
		lines = append(lines, condition)
	}
	lines = append(lines, l.describeCosts(declared))
	return strings.Join(lines, "\n")
}

// describeOpening is the damage sentence, or the aim alone when a skill deals
// none: a skill with no power is not a weaker attack, it is a different kind of
// action, and opening with "0% of attack" would file it as the first.
func (l Lang) describeOpening(declared skill.Skill, shapes *pattern.Book) string {
	aim := l.describeAim(declared, shapes)
	if declared.Power <= 0 {
		// A skill that deals no damage still has to say what it reaches. Without
		// this a two-cell powder read exactly like a single-target one, because
		// every sentence after this one talks about "the target" and none of them
		// count how many there are.
		if declared.Target == skill.Self {
			return ""
		}
		return l.Say(BlurbAims, aim)
	}
	stat := l.describeStat(declared.Scaling.Stat)
	damage := l.Say(BlurbOnce, percent(declared.Power), stat)
	if declared.StrikeCount() > 1 {
		damage = l.Say(BlurbStrikes, declared.StrikeCount(),
			percent(declared.Power), stat, percent(declared.TotalPower()))
	}
	sentence := l.Say(BlurbHits, aim, damage)
	if declared.Pierce > 0 {
		sentence += l.Say(BlurbPierces, percent(declared.Pierce))
	}
	return sentence + "."
}

// describeAim is who the skill reaches: the side it aims at, and how many cells
// its shape covers when that is more than one.
func (l Lang) describeAim(declared skill.Skill, shapes *pattern.Book) string {
	side := l.Text(map[skill.Side]Key{
		skill.Enemy: BlurbSideEnemy, skill.Ally: BlurbSideAlly,
		skill.Self: BlurbSideSelf, skill.All: BlurbSideAll,
	}[declared.Target])
	cells := 1
	if shapes != nil {
		if shape, err := shapes.Lookup(declared.Pattern); err == nil {
			cells = shape.MaxTargets()
		}
	}
	if declared.Target == skill.Self || cells <= 1 {
		return side
	}
	return l.Say(BlurbCells, cells, side)
}

// describeExtras is everything a skill does that is not damage, one sentence
// each, in the order the fields are declared in.
func (l Lang) describeExtras(declared skill.Skill) []string {
	out := make([]string, 0, 4)
	if declared.Restores > 0 {
		out = append(out, l.Say(BlurbRestores,
			percent(declared.Restores), l.describeStat(declared.Scaling.Stat)))
	}
	if declared.Drains > 0 {
		out = append(out, l.Say(BlurbDrains, percent(declared.Drains)))
	}
	for _, application := range declared.Applies {
		out = append(out, l.Say(BlurbInflicts,
			l.stacked(application.Status, application.Stacks), percent(application.Chance)))
	}
	// One sentence for all of them rather than one each: a skill granting two
	// buffs grants them together, on the same turn, and two sentences read as two
	// separate things happening.
	if len(declared.SelfApplies) > 0 {
		names := make([]string, 0, len(declared.SelfApplies))
		for _, application := range declared.SelfApplies {
			names = append(names, l.stacked(application.Status, application.Stacks))
		}
		out = append(out, l.Say(BlurbSelfApplies, l.join(names)))
	}
	if declared.Strips != nil {
		names := make([]string, 0, len(declared.Strips.Categories))
		for _, category := range declared.Strips.Categories {
			names = append(names, l.glossed(category.String()))
		}
		if declared.Strips.Stacks == 1 {
			out = append(out, l.Say(BlurbStripsOne, l.join(names)))
		} else {
			out = append(out, l.Say(BlurbStrips, declared.Strips.Stacks, l.join(names)))
		}
	}
	return out
}

// describeCondition is the amplifier, written as what the target must be rather
// than as a bonus figure: a player picking a skill is asking when it is worth
// using, and "+1000 power" answers a different question.
func (l Lang) describeCondition(declared skill.Skill) string {
	if declared.Requires == nil {
		return ""
	}
	clauses := make([]string, 0, 2)
	if declared.Requires.ReadsStatus() {
		clauses = append(clauses, l.Say(BlurbWhenCarrying,
			l.stacked(declared.Requires.Status, declared.Requires.MinStacks)))
	}
	if declared.Requires.ReadsHealth() {
		clauses = append(clauses, l.Say(BlurbWhenHurt, percent(declared.Requires.BelowHealth)))
	}
	amplified := declared.PowerAgainst(declared.Requires.Satisfying())
	sentence := l.Say(BlurbAmplified, l.join(clauses),
		percent(amplified*declared.StrikeCount()), l.describeStat(declared.Scaling.Stat))
	if declared.Requires.Consume {
		sentence += l.Say(BlurbConsumes, l.glossed(declared.Requires.Status))
	}
	return sentence + "."
}

// describeCosts is the line every skill has: how far it reaches, how often it
// connects, and how long it is gone for.
func (l Lang) describeCosts(declared skill.Skill) string {
	parts := make([]string, 0, 3)
	if declared.Target == skill.Self {
		// A self-targeted skill has no range to state, and saying nothing at all
		// would leave the line opening with its cooldown as though the aim were
		// obvious. It is the one aim a reader cannot infer from the sentences
		// above, because those talk about what the caster receives.
		parts = append(parts, l.Text(BlurbCostSelf))
	} else {
		parts = append(parts, l.Say(BlurbCostRange, declared.Range))
	}
	if declared.Power > 0 || len(declared.Applies) > 0 {
		parts = append(parts, l.Say(BlurbCostAccuracy, percent(declared.Accuracy)))
	}
	if declared.Cooldown == 1 {
		// One turn is the common case and the one English gets wrong for free:
		// "1 turns" reads as a bug in the tool rather than as a cooldown. Two
		// keys rather than a plural rule, because Vietnamese has no plural and a
		// rule would make it pretend it does.
		parts = append(parts, l.Text(BlurbCostCooldownOne))
	} else if declared.Cooldown > 0 {
		parts = append(parts, l.Say(BlurbCostCooldown, declared.Cooldown))
	} else {
		parts = append(parts, l.Text(BlurbCostEveryTurn))
	}
	return capitalise(strings.Join(parts, " · ") + ".")
}

// DescribePassive is what a trait does. A trait is not chosen, so this answers
// "what is this unit carrying" rather than "should I use it".
func (l Lang) DescribePassive(held passive.Passive) string {
	lines := make([]string, 0, 3)
	// "Always" only where it is true. A gated trait comes and goes with its
	// holder's health, and the last line below says when — so opening with
	// "always carries" and closing with "only while under a third" would be two
	// sentences of the same paragraph contradicting each other, which is worse
	// than either of them alone.
	carries := BlurbTraitGrants
	if held.While != nil {
		carries = BlurbTraitGrantsGated
	}
	for _, grant := range held.Grants {
		lines = append(lines, l.Say(carries, l.stacked(grant.Status, grant.Stacks)))
	}
	for _, application := range held.Applies {
		lines = append(lines, l.Say(BlurbTraitApplies,
			l.stacked(application.Status, application.Stacks), percent(application.Chance)))
	}
	// One sentence for the whole reply rather than one per part: what a reader
	// wants is what it costs to attack this unit, and a damage line filed apart
	// from a status line leaves them to add it up.
	//
	// Three whole wordings rather than one built from fragments, because a
	// sentence assembled by joining clauses is a sentence neither language gets
	// to choose the shape of — and the blanks have to arrive in one order for
	// both, which TestTheSameBlanksInEveryLanguage enforces.
	if held.Replies.Answers() {
		switch {
		case held.Replies.Power > 0 && len(held.Replies.Applies) > 0:
			// The first application only. A trait answering with two statuses at
			// once is authorable and nothing shipped does it; a second sentence
			// for the rest is the change to make when something does.
			first := held.Replies.Applies[0]
			lines = append(lines, l.Say(BlurbTraitReplyBoth,
				percent(held.Replies.Power),
				l.stacked(first.Status, first.Stacks), percent(first.Chance)))
		case held.Replies.Power > 0:
			lines = append(lines, l.Say(BlurbTraitReplyDamage, percent(held.Replies.Power)))
		default:
			first := held.Replies.Applies[0]
			lines = append(lines, l.Say(BlurbTraitReplyStatus,
				l.stacked(first.Status, first.Stacks), percent(first.Chance)))
		}
	}
	for _, resistance := range held.Resists {
		if resistance.Amount >= scale.Base {
			lines = append(lines, l.Say(BlurbTraitImmune, l.glossed(resistance.Status)))
			continue
		}
		lines = append(lines, l.Say(BlurbTraitResists,
			percent(resistance.Amount), l.glossed(resistance.Status)))
	}
	if held.While != nil {
		lines = append(lines, l.Say(BlurbTraitWhile, percent(held.While.BelowHealth)))
	}
	if len(lines) == 0 {
		return l.Text(BlurbTraitNone)
	}
	return strings.Join(lines, "\n")
}

// percent turns a share in parts per thousand into whole percent. Truncation is
// deliberate: a share that is not a whole percent is a tuning detail, and the
// listing beside this carries the exact figure for anybody who wants it.
func percent(permille int) int { return permille / (scale.Base / 100) }

// glossed is a data id under its name in this language, falling back to the id.
// A miss is a bare id rather than a blank, the same answer every other listing
// gives, because a name nobody wrote is better read than guessed at.
func (l Lang) glossed(id string) string {
	if name := l.Gloss(id); name != "" {
		return name
	}
	return id
}

// stacked is a status under its name with its count, when the count is worth
// saying. One stack is the unstated default everywhere in the data, so writing
// "x1" would make the common case look like a special one.
func (l Lang) stacked(id string, stacks int) string {
	name := l.glossed(id)
	if stacks <= 1 {
		return name
	}
	return name + " x" + itoa(stacks)
}

// join is a list read out with the language's own conjunction, which is the one
// piece of grammar these sentences need and the one a format string cannot hold.
func (l Lang) join(parts []string) string {
	return strings.Join(parts, l.Text(BlurbAnd))
}

// describeStat is the stat a skill scales off. Nearly every skill scales off
// attack, so the name is looked up rather than assumed — a skill that scaled off
// defence and said "attack" would read as a typo in the data.
func (l Lang) describeStat(stat progression.Kind) string {
	key, known := map[progression.Kind]Key{
		progression.Attack: BlurbStatAttack, progression.Defense: BlurbStatDefense,
		progression.Speed: BlurbStatSpeed, progression.Accuracy: BlurbStatAccuracy,
		progression.Dodge: BlurbStatDodge,
	}[stat]
	if !known {
		return stat.String()
	}
	return l.Text(key)
}

func capitalise(line string) string {
	if line == "" {
		return line
	}
	runes := []rune(line)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
