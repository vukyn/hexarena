package i18n

import (
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
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
	if opening := l.describeOpening(declared); opening != "" {
		lines = append(lines, opening)
	}
	lines = append(lines, l.describeExtras(declared)...)
	if condition := l.describeSelfCondition(declared); condition != "" {
		lines = append(lines, condition)
	}
	if condition := l.describeCondition(declared); condition != "" {
		lines = append(lines, condition)
	}
	lines = append(lines, l.describeCosts(declared, shapes))
	return strings.Join(lines, "\n")
}

// describeOpening is the damage sentence, or the aim alone when a skill deals
// none: a skill with no power is not a weaker attack, it is a different kind of
// action, and opening with "0% of attack" would file it as the first.
func (l Lang) describeOpening(declared skill.Skill) string {
	aim := l.describeAim(declared)
	if declared.Power <= 0 {
		// A skill that deals no damage still has to say what it reaches. Without
		// this a two-cell powder read exactly like a single-target one, because
		// every sentence after this one talks about "the target" and none of them
		// count how many there are.
		flavour := l.flavour(declared)
		if flavour != "" {
			return flavour + "."
		}
		if declared.Target == skill.Self {
			return ""
		}
		return l.Say(BlurbAims, aim)
	}
	stat := l.describeStat(declared.Scaling.Stat)
	damage := l.Say(BlurbOnce, share(declared.Power), stat)
	if declared.StrikeCount() > 1 {
		damage = l.Say(BlurbStrikes, declared.StrikeCount(),
			share(declared.Power), stat, share(declared.TotalPower()))
	}
	// An authored clause replaces the derived one it would otherwise open with,
	// and the figures are appended to it either way. Nothing derives "dây leo"
	// from vine_whip — the name is the one fact only a person holds — so this is
	// where prose is allowed in, and the digit ban at parse is what keeps it from
	// carrying a number that could go stale.
	sentence := l.Say(BlurbHits, aim, damage)
	if flavour := l.flavour(declared); flavour != "" {
		sentence = l.Say(BlurbFlavoured, flavour, damage)
	}
	if declared.Pierce > 0 {
		sentence += l.Say(BlurbPierces, share(declared.Pierce))
	}
	return sentence + "."
}

// flavour is a skill's authored opening clause in this language, or nothing.
//
// English has none: like a skill's name, the clause is authored once and in
// Vietnamese, so an English reader gets the derived opening rather than a
// Vietnamese sentence dropped into an English one. That is the same trade
// SkillName already makes, and it is why the field is not a translations table —
// a second file is a second thing to keep in step.
func (l Lang) flavour(declared skill.Skill) string {
	if l != Vi {
		return ""
	}
	return declared.Flavour
}

// traitFlavour is a trait's authored clause in this language, or nothing.
//
// English has none, the same trade skill flavour makes: the clause is authored
// once and in Vietnamese, so an English reader gets the derived lines rather than
// a Vietnamese sentence dropped into an English paragraph. A translations table
// would be a second file to keep in step, and the one it would be out of step
// with is the one an author actually edits.
func (l Lang) traitFlavour(held passive.Passive) string {
	if l != Vi {
		return ""
	}
	return held.Flavour
}

// describeAim is the side a skill reaches. How many cells it covers is on the
// cost line instead — see cellsCovered.
func (l Lang) describeAim(declared skill.Skill) string {
	return l.Text(map[skill.Side]Key{
		skill.Enemy: BlurbSideEnemy, skill.Ally: BlurbSideAlly,
		skill.Self: BlurbSideSelf, skill.All: BlurbSideAll,
	}[declared.Target])
}

// cells is how many of them a skill's shape covers, or one when the book cannot
// say. It is read by the cost line rather than by the opening sentence: a shape
// is a fact about reach, which is what that line is for, and keeping it there is
// what leaves the opening free for the authored clause. In the opening it also
// read badly twice over — "rắc phấn ru ngủ xuống chỗ đối phương đứng, nhắm 2 ô
// đối phương" says the target twice and the count once too late.
func cellsCovered(declared skill.Skill, shapes *pattern.Book) int {
	if shapes == nil {
		return 1
	}
	shape, err := shapes.Lookup(declared.Pattern)
	if err != nil {
		return 1
	}
	return shape.MaxTargets()
}

// describeExtras is everything a skill does that is not damage, one sentence
// each, in the order the fields are declared in.
func (l Lang) describeExtras(declared skill.Skill) []string {
	out := make([]string, 0, 4)
	if declared.Restores > 0 {
		out = append(out, l.Say(BlurbRestores,
			share(declared.Restores), l.describeStat(declared.Scaling.Stat)))
	}
	if declared.Drains > 0 {
		out = append(out, l.Say(BlurbDrains, share(declared.Drains)))
	}
	if declared.Summons.Summons() {
		out = append(out, l.describeSummon(declared.Summons))
	}
	// The verb turns on which side the skill can reach, because an application is
	// only an affliction when it lands on somebody the caster is fighting.
	//
	// Everything in the book aimed at an enemy until a team buff was written, and
	// nothing said the sentence had an assumption in it: the first ally-aimed
	// skill described its own buff as something inflicted on a friend. Enemy is
	// the one side that can be called hostile without asking anything else --
	// self and ally are plainly not, and a skill aimed at both halves is landing
	// on its own squad too, so the neutral verb is the only one true of every
	// target it catches.
	given := BlurbInflicts
	if declared.Target != skill.Enemy {
		given = BlurbGives
	}
	for _, application := range declared.Applies {
		out = append(out, l.Say(given,
			l.stacked(application.Status, application.Stacks), share(application.Chance)))
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

// describeSummon is what a skill puts on the board, in one sentence.
//
// The name is the summon's own where it has one and the skill's id where it does
// not, because a summon with no name is a copy of its caster and a caster's name
// is not a fact this layer holds — the skill is. A bare id in a Vietnamese
// sentence is the same fallback every other name in these descriptions takes.
//
// Nothing is said about the stat line. A share of a caster is a number a reader
// cannot check without knowing the caster, and a fixed line is six figures; the
// listing beside this carries both for anybody tuning them, and what a player
// wants from a sentence is what arrives and for how long.
func (l Lang) describeSummon(declared *skill.Summon) string {
	// The authored name is Vietnamese, like every other name in the data, so
	// English says "copy" rather than printing it. That is the division Gloss
	// makes — a data name is authored once and English shows what it can read —
	// and a summon has no id for English to fall back on, so the word is the
	// only thing left that is true in it.
	name := l.Text(BlurbSummonedCopy)
	if declared.Count > 1 {
		name = l.Text(BlurbSummonedCopies)
	}
	if l == Vi && declared.Name != "" {
		name = declared.Name
	}
	subject := l.Say(BlurbSummonedOne, name)
	if declared.Count > 1 {
		subject = l.Say(BlurbSummonedMany, declared.Count, name)
	}
	if declared.Lasts <= 0 {
		return l.Say(BlurbSummons, subject)
	}
	lasts := l.Say(BlurbStatusLasts, declared.Lasts)
	if declared.Lasts == 1 {
		lasts = l.Text(BlurbStatusLastsOne)
	}
	return l.Say(BlurbSummonsBriefly, subject, lasts)
}

// describeCondition is the amplifier, written as what the target must be rather
// than as a bonus figure: a player picking a skill is asking when it is worth
// using, and "+1000 power" answers a different question.
// describeCondition is the amplifier read against the target, written as what
// that target must be rather than as a bonus figure: a player picking a skill is
// asking when it is worth using, and "+1000 power" answers a different question.
func (l Lang) describeCondition(declared skill.Skill) string {
	return l.conditionSentence(declared, declared.Requires, BlurbAmplified)
}

// describeSelfCondition is the same sentence about the caster.
//
// A separate sentence rather than a second clause on the first, because they are
// two different bargains: one is about who is in front of you and the other is
// about what you are spending, and a skill carrying both is offering both.
func (l Lang) describeSelfCondition(declared skill.Skill) string {
	return l.conditionSentence(declared, declared.SelfRequires, BlurbSelfAmplified)
}

// conditionSentence writes one condition, whichever unit it reads.
//
// The clauses themselves say nothing about whose health or whose stacks are
// being counted -- "is carrying poison", "is at or below half health" -- so only
// the opening changes between the two. That is what makes one function honest
// here rather than a saving: the sentences really are the same sentence.
func (l Lang) conditionSentence(declared skill.Skill, condition *skill.Condition, opening Key) string {
	if condition == nil {
		return ""
	}
	clauses := make([]string, 0, 2)
	if condition.ReadsStatus() {
		clauses = append(clauses, l.Say(BlurbWhenCarrying,
			l.stacked(condition.Status, condition.MinStacks)))
	}
	if condition.ReadsHealth() {
		clauses = append(clauses, l.Say(BlurbWhenHurt, share(condition.BelowHealth)))
	}
	amplified := declared.Power + condition.BonusPower
	sentence := l.Say(opening, l.join(clauses),
		share(amplified*declared.StrikeCount()), l.describeStat(declared.Scaling.Stat))
	if condition.Consume {
		sentence += l.Say(BlurbConsumes, l.glossed(condition.Status))
	}
	return sentence + "."
}

// describeCosts is the line every skill has: how far it reaches, how often it
// connects, and how long it is gone for.
func (l Lang) describeCosts(declared skill.Skill, shapes *pattern.Book) string {
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
	if covered := cellsCovered(declared, shapes); covered > 1 {
		parts = append(parts, l.Say(BlurbCostCells, covered))
	}
	if declared.Power > 0 || len(declared.Applies) > 0 {
		parts = append(parts, l.Say(BlurbCostAccuracy, share(declared.Accuracy)))
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

// DescribeElement is where one element sits in the affinity chart: what it
// beats, what beats it, and what each of those is worth.
//
// Derived from the chart for the reason every description here is derived: the
// multipliers and the cycles are data, and the three figures in elements.json
// are the ones every damage number in the game is scaled by. An authored line
// reading "takes half again" survives advantage dropping from 1500 to 1300 with
// nothing to catch it.
//
// The elements inside it are bare ids, as they are in the skills listing and in
// WhoMaySummary: an id is what the data files hold and what an author types, and
// the reference this is printed under has the name in the column above.
//
// An inert element -- neutral, and nothing else today -- gets a sentence of its
// own rather than two empty ones. "Beats nothing" and "nothing beats it" as two
// lines reads as two separate facts about a broken entry; it is one fact, and it
// is the whole of what being inert means.
func (l Lang) DescribeElement(member element.Element, chart *element.Chart) string {
	if chart == nil || !member.Valid() {
		return ""
	}
	rates := chart.Multipliers()
	strong, weak := chart.Strengths(member), chart.Weaknesses(member)
	if len(strong) == 0 && len(weak) == 0 {
		return l.Say(BlurbElementInert, share(rates.Neutral))
	}
	lines := make([]string, 0, 2)
	if len(strong) > 0 {
		lines = append(lines, l.Say(BlurbElementStrong,
			l.JoinIDs(elementIDs(strong)), share(rates.Advantage)))
	}
	if len(weak) > 0 {
		lines = append(lines, l.Say(BlurbElementWeak,
			l.JoinIDs(elementIDs(weak)), share(rates.Disadvantage)))
	}
	return strings.Join(lines, "\n")
}

// elementIDs is a run of elements as the ids they are written with.
func elementIDs(members []element.Element) []string {
	out := make([]string, 0, len(members))
	for _, member := range members {
		out = append(out, member.String())
	}
	return out
}

// StatusesInSkill is the statuses a skill's description will name, in the order
// the sentences name them and with each id appearing once.
//
// The skill-side twin of StatusesNamed, and it exists for the same reason: a
// screen marking those names where they are printed must not find them by
// matching substrings against its own prose, which would style a word that
// happens to sit in an authored flavour clause.
//
// ⚠️ It has to agree with Describe about which statuses are *named*. The order
// is Describe's order -- applications, then what the caster gives itself, then
// the caster's own condition, then the target's -- because a caller marking a
// sentence walks the sentences in that order too.
//
// Strips names *categories* rather than statuses, so nothing from it belongs
// here: a category has its own glossary, and a caller wanting those asks
// StatusCategory for them.
func StatusesInSkill(declared skill.Skill) []string {
	named := make([]string, 0, 4)
	seen := make(map[string]bool, 4)
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		named = append(named, id)
	}
	for _, application := range declared.Applies {
		add(application.Status)
	}
	for _, application := range declared.SelfApplies {
		add(application.Status)
	}
	if declared.SelfRequires.ReadsStatus() {
		add(declared.SelfRequires.Status)
	}
	if declared.Requires.ReadsStatus() {
		add(declared.Requires.Status)
	}
	return named
}

// StatusesNamed is the statuses a trait's description will name, in the order
// the sentences name them and with each id appearing once.
//
// It exists so a screen can act on those names — mark them where they are
// printed, or offer to look one up — without reading the sentences back to find
// them. Reading them back would be substring matching against prose in two
// languages: it would style a name that happens to occur in a flavour clause,
// and miss one the glossary has no entry for, which prints as a bare id.
//
// ⚠️ It has to agree with DescribePassive about which statuses are *named*
// rather than merely held. A reply names its **first** application and no more,
// because that is all the one sentence a reply gets has room for — so a trait
// answering with two statuses holds two and names one.
// TestATraitNamesEveryStatusItsDescriptionNames holds the two together.
//
// Ids rather than names, and therefore no Lang: a caller that wants the name
// asks Gloss for it, and a caller that wants to jump to the status wants the id
// anyway.
func StatusesNamed(held passive.Passive) []string {
	named := make([]string, 0, 4)
	seen := make(map[string]bool, 4)
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		named = append(named, id)
	}
	for _, grant := range held.Grants {
		add(grant.Status)
	}
	for _, resistance := range held.Resists {
		add(resistance.Status)
	}
	for _, application := range held.Applies {
		add(application.Status)
	}
	for _, raise := range held.Amplifies {
		// Both shares name the same status, and a trait raising neither is
		// refused at parse, so there is nothing to guard here.
		add(raise.Status)
	}
	if held.Replies.Answers() && len(held.Replies.Applies) > 0 {
		add(held.Replies.Applies[0].Status)
	}
	return named
}

// DescribePassive is what a trait does. A trait is not chosen, so this answers
// "what is this unit carrying" rather than "should I use it".
//
// # The order the sentences come in
//
// Not the order the fields are declared in, which is what it was and which read
// backwards on the one trait that has two halves: venom_blood answered an
// attacker and then said it was immune to poison, when the fiction runs the
// other way — its blood is venom, so nothing poisons it, so whatever bites it is
// poisoned. Field order is an accident of when each half was built.
//
// So: what the holder **is** (grants, resists), then what its **own attacks** do
// (applies, amplifies, drains), then what attacking **it** costs (replies), then
// **when** any of it is true (while, which was already last and stays there
// because it qualifies everything above it).
//
// # Why there is a clause here at all
//
// Every line below is derived, which is right and is also why a trait read like
// a field dump: "always carries cứng đòn", "refuses bỏng outright". They say what
// it does and never what it is, so the mechanism arrives with nothing to hang it
// on — and the trait's own authored name, máu độc, was rendered nowhere in the
// sentences. Flavour is the one line allowed to say it, under the digit ban that
// keeps prose from going stale, exactly as a skill's is.
//
// It is a **lead line** rather than a replacement, which is where it differs
// from a skill's. A skill has one opening sentence for the clause to take over;
// a trait has between one and six lines and no opening among them, so a clause
// that replaced one would be replacing whichever happened to sort first.
func (l Lang) DescribePassive(held passive.Passive) string {
	lines := make([]string, 0, 4)
	if flavour := l.traitFlavour(held); flavour != "" {
		lines = append(lines, flavour+".")
	}
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
	for _, resistance := range held.Resists {
		if resistance.Amount >= scale.Base {
			lines = append(lines, l.Say(BlurbTraitImmune, l.glossed(resistance.Status)))
			continue
		}
		lines = append(lines, l.Say(BlurbTraitResists,
			share(resistance.Amount), l.glossed(resistance.Status)))
	}
	for _, application := range held.Applies {
		lines = append(lines, l.Say(BlurbTraitApplies,
			l.stacked(application.Status, application.Stacks), share(application.Chance)))
	}
	// Both shares, each as its own sentence, and the status named first in both
	// languages so one arg order serves both.
	for _, raise := range held.Amplifies {
		if raise.Effect > 0 {
			lines = append(lines, l.Say(BlurbTraitAmplifiesEffect,
				l.glossed(raise.Status), share(raise.Effect)))
		}
		if raise.Chance > 0 {
			lines = append(lines, l.Say(BlurbTraitAmplifiesChance,
				l.glossed(raise.Status), share(raise.Chance)))
		}
	}
	if held.Drains > 0 {
		lines = append(lines, l.Say(BlurbTraitDrains, share(held.Drains)))
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
			// The chance before the status, in both languages. A reply reads as
			// "and a 3% chance of poison" rather than "and poison, 3% of the
			// time" -- and once one language wants that order, both take it:
			// TestTheSameBlanksInEveryLanguage holds one arg order for the pair.
			lines = append(lines, l.Say(BlurbTraitReplyBoth,
				share(held.Replies.Power), l.describeStat(held.Replies.Scaling.Stat),
				share(first.Chance), l.stacked(first.Status, first.Stacks)))
		case held.Replies.Power > 0:
			// The stat is named rather than assumed. It was "attack" in the
			// wording itself while every reply was priced off attack, and a
			// sentence that says attack while the engine reads defence is the one
			// thing a derived description exists to make impossible.
			lines = append(lines, l.Say(BlurbTraitReplyDamage,
				share(held.Replies.Power), l.describeStat(held.Replies.Scaling.Stat)))
		default:
			first := held.Replies.Applies[0]
			lines = append(lines, l.Say(BlurbTraitReplyStatus,
				share(first.Chance), l.stacked(first.Status, first.Stacks)))
		}
	}
	if held.While != nil {
		lines = append(lines, l.Say(BlurbTraitWhile, share(held.While.BelowHealth)))
	}
	if len(lines) == 0 {
		return l.Text(BlurbTraitNone)
	}
	return strings.Join(lines, "\n")
}

// share renders a proportion in parts per thousand as a whole percent.
//
// # Rounded here, exact in the tables, and that split is the whole of it
//
// forge.Percent keeps a tenth where there is one — 25 parts per thousand reads
// as "2.5%" — and that is right for hexforge's tables, which an author reads to
// tune a number and where the tenth *is* what is being tuned. It is wrong inside
// a sentence: "và có 2.5% khả năng dính trúng độc" hands a player a precision
// they cannot act on, and the decimal point is the only mark in the line that is
// not a comma. So the sentence rounds and the table does not.
//
// This is a second renderer where there used to be one, and the history matters
// because the first split was a mistake. share truncated, on the argument that a
// fraction of a percent is a tuning detail whose exact figure is in the listing
// beside the sentence — and for a trait there was no such listing, so
// venom_blood's reply chance of 25 read as "2%". Truncation was replaced by
// forge.Percent for that reason, and the tenth it brought is what this rounds
// away again. The difference is that truncation lost a fifth of the value and
// rounding half away from zero does not: 25 becomes 3, not 2.
//
// What makes the rounding safe rather than lossy is a rule on the **data**:
// nothing is ever tuned by less than a percent, so nothing can land in the range
// where rounding would print "0%" — a share that small is one nobody feels
// across a battle, and a description of it reads as a feature that does not
// work. TestNoShippedShareIsUnderOnePercent holds it over every shipped skill,
// trait and status. Carrying a decimal place here to survive data nobody will
// author would be the renderer paying for a case the rule forbids.
func share(permille int) string {
	sign := ""
	if permille < 0 {
		sign, permille = "-", -permille
	}
	// Half away from zero, in integers: 25 becomes 3 rather than 2, so the
	// rounding never reads as the truncation this replaced.
	return sign + itoa((permille+5)/10) + "%"
}

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

// DescribeStatus is what a timed effect does, for somebody who has just read its
// name in a log and has nowhere else to look it up.
//
// It is the third of these and it answers a third question. Describe answers
// "should I use this", DescribePassive answers "what is this unit carrying", and
// this answers "what has just happened to me" — which is the one the game asked
// most often and had no answer to at all: the log said *resisted mire*, the unit
// table said *mire*, and nothing anywhere said that mire is a quarter off speed.
//
// Derived, for the reason the other two are. It is worth saying twice because a
// status is the layer most tempting to write a paragraph for: the ids read like
// words — burn, blind, stun — and a hand-written "burns for a couple of turns"
// survives the duration moving with nothing to catch it.
//
// # Why a life and not a tick
//
// Poison at 500 for three turns and burn at 800 for two are compared by what
// they cost over their lives, and the per-turn figures rank them the wrong way
// round: burn ticks harder and poison costs more. So both are stated, and the
// stacked total with them, because a percent per stack against a cap of three is
// a different number from the percent.
//
// One stack's life, and the cap's — not the ramp. skills.golden prints a third
// figure under the same words, the full ramp of a status reapplied every turn
// until it caps, and that one is an author's ceiling rather than a player's
// fact: reaching it needs a skill off cooldown every turn, which no shipped kit
// has. What is stated here assumes nothing about how often it is applied.
//
// # What it deliberately does not know
//
// The declared kind, not what a unit will feel. A tick scales off whoever
// applied it, an amplifier raises it and a resistance refuses it outright, and a
// stacked stat term saturates rather than adding up — so the figures here are
// the book's and not the log's. Saying so is the screen's job rather than this
// one's: BlurbStatusCaveat is printed once per reference, not fifteen times.
func (l Lang) DescribeStatus(kind status.Kind) string {
	lines := l.describeStatusEffect(kind)
	lines = append(lines, l.describeStatusCosts(kind))
	return strings.Join(lines, "\n")
}

// describeStatusEffect is what the status does, which is its category's doing
// plus whatever stat terms it carries.
//
// The terms are read outside the switch rather than under the two stat
// categories, because nothing stops a damage-over-time from carrying one and a
// description filed by category would silently drop it.
func (l Lang) describeStatusEffect(kind status.Kind) []string {
	out := make([]string, 0, 4)
	switch kind.Category {
	case status.Dot, status.Regen:
		ticks := BlurbStatusTicks
		if kind.Category == status.Regen {
			ticks = BlurbStatusHeals
		}
		out = append(out, l.Say(ticks, share(kind.TickPower)))
		// A permanent status can be neither of these — ParseBook refuses one —
		// so a life is always a number here rather than a nought standing in for
		// "never ends".
		life := kind.TickPower * kind.Duration
		if kind.MaxStacks > 1 {
			out = append(out, l.Say(BlurbStatusLifeCapped,
				l.lasts(kind), share(life), share(life*kind.MaxStacks)))
		} else {
			out = append(out, l.Say(BlurbStatusLife, l.lasts(kind), share(life)))
		}
	case status.Control:
		out = append(out, l.Text(BlurbStatusControls))
	case status.Taunt:
		out = append(out, l.Text(BlurbStatusTaunts))
	case status.Shield:
		out = append(out, l.Text(BlurbStatusShields))
	}
	for _, term := range kind.Modifiers {
		// "Per stack" only where there can be a second one. A status capped at
		// one stack is every permanent status a trait grants, and telling a
		// reader the rate of something that cannot happen twice reads as a
		// promise that it can.
		raises, lowers := BlurbStatusRaisesOnce, BlurbStatusLowersOnce
		if kind.MaxStacks > 1 {
			raises, lowers = BlurbStatusRaises, BlurbStatusLowers
		}
		moves := raises
		if term.Amount < 0 {
			moves = lowers
		}
		out = append(out, l.Say(moves, l.describeStat(term.Target.Stat()), statusAmount(term, 1)))
		if kind.MaxStacks > 1 {
			out = append(out, l.Say(BlurbStatusStacked,
				statusAmount(term, kind.MaxStacks)))
		}
	}
	if len(out) == 0 {
		// A buff with no terms, which is authorable and does nothing. Saying so
		// is the point: an empty description would read as this function failing
		// rather than as the status being empty.
		out = append(out, l.Text(BlurbStatusNothing))
	}
	return out
}

// describeStatusCosts is the line every status has, and it is the counterpart of
// describeCosts: what kind of thing it is, how long it stays, how many will
// layer.
//
// Permanent is a word rather than a duration. Snapshot.Permanent exists for the
// same reason — a permanent status carries no duration, and printing its zero
// reads as one about to expire.
func (l Lang) describeStatusCosts(kind status.Kind) string {
	parts := make([]string, 0, 3)
	parts = append(parts, l.StatusCategory(kind.Category.String()))
	switch {
	case kind.Permanent:
		parts = append(parts, l.Text(BlurbStatusAlways))
	default:
		parts = append(parts, l.lasts(kind))
	}
	if kind.MaxStacks == 1 {
		parts = append(parts, l.Text(BlurbStatusOneStack))
	} else {
		parts = append(parts, l.Say(BlurbStatusStacks, kind.MaxStacks))
	}
	return capitalise(strings.Join(parts, " · ") + ".")
}

// lasts is how long a status stays, as a phrase rather than a number, because
// the singular has a wording of its own. Two lines want it now — the cost line
// and the life line — and a second copy of the singular test is how the two
// would start disagreeing about what one turn is called.
//
// Permanent is not handled here: a permanent status has no duration to print,
// and the one caller that can meet one says so in its own words.
func (l Lang) lasts(kind status.Kind) string {
	if kind.Duration == 1 {
		return l.Text(BlurbStatusLastsOne)
	}
	return l.Say(BlurbStatusLasts, kind.Duration)
}

// statusAmount is how big a modifier term is, at a given number of stacks.
//
// Unsigned, because the sentence around it has already chosen between raising
// and lowering: a minus sign inside "lowers speed by -25%" is the same fact
// twice and reads as a double negative.
//
// A percentage and a flat term are printed differently rather than both as
// percentages, and that matters even though nothing shipped carries a flat one:
// a flat +50 rendered through share() would print as "5%", which is a wrong
// number rather than a missing one.
func statusAmount(term modifier.Modifier, stacks int) string {
	size := term.Amount * int64(stacks)
	if size < 0 {
		size = -size
	}
	if term.Mode == modifier.Percent {
		return share(int(size))
	}
	return itoa(int(size))
}
