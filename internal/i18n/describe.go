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
	if gradient := l.describeSelfGradient(declared); gradient != "" {
		lines = append(lines, gradient)
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
	// The chance and nothing else. What a critical strike is worth is a
	// game-wide constant living on combat.Rules, and naming it here would need
	// this package to import that one — which would put the rules book behind
	// every sentence a renderer draws, to print a figure identical on every
	// skill in the game.
	if declared.Crit > 0 {
		sentence += l.Say(BlurbCritical, share(declared.Crit))
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
		// A category is worded, never glossed: Gloss answers for a data id and
		// is empty outside Vietnamese on purpose, so an English sentence took
		// the enum spelling.
		for _, category := range declared.Strips.Categories {
			names = append(names, l.StatusCategoryNoun(category.String()))
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
// # The share is said and the fixed line is not
//
// They are not the same kind of number and were wrong to be treated as one. A
// share is a single figure that means the same thing wherever it is read — a
// copy at 40% of whoever made it is half as good as one at 80%, whichever
// character is holding the skill — which is the argument this whole file makes
// for printing power as a share of a stat rather than as damage. A fixed line is
// six figures nobody can compare without the caster in front of them, and it
// belongs where a stat line belongs.
//
// The share was left out on the reasoning that a listing beside this carries it.
// No listing does: neither hexforge nor its full-screen twin mentions a summon
// at all, so this sentence is the only place a summon is described, and what it
// left out was simply gone. A copy is the one thing a skill puts on the board
// whose strength is a choice the author made, and a description of it that
// cannot say how strong describes a different mechanic.
func (l Lang) describeSummon(declared *skill.Summon) string {
	subject := l.summonSubject(declared)
	// Exactly one of the two shares is ever set, and a summon with a fixed stat
	// line sets neither, so the sum is the share there is and nought is the
	// answer that there is none. Which of the two it was still has to be said:
	// they differ in whether buffing before the cast reaches the copy.
	if shared := declared.Share + declared.ShareOfBase; shared > 0 {
		of := l.Text(BlurbSummonedOfCurrent)
		if declared.ShareOfBase > 0 {
			of = l.Text(BlurbSummonedOfBase)
		}
		wording := BlurbSummonedShare
		if declared.Count > 1 {
			wording = BlurbSummonedShareEach
		}
		subject = l.Say(wording, subject, share(shared), of)
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

// summonSubject is what a summoning skill puts on the board, counted and named:
// "một phân thân", "2 copies", "một cóc".
//
// A function rather than the two copies it was about to become. The sentence
// describeSummon writes and the clause SummariseSkill writes are two readings of
// the same three decisions — copy or creature, one or several, authored name or
// the fallback word — and a second copy of them is a second thing that can call a
// toad a copy. That is not hypothetical: telling a creature from a copy by the
// stat spelling was itself a fix for the fallback saying the one thing about the
// shipped toad that is untrue.
//
// The authored name is Vietnamese, like every other name in the data, so English
// says "copy" rather than printing it. That is the division Gloss makes — a data
// name is authored once and English shows what it can read — and a summon has no
// id for English to fall back on, so the word is the only thing left that is
// true in it.
func (l Lang) summonSubject(declared *skill.Summon) string {
	one, many := BlurbSummonedCopy, BlurbSummonedCopies
	if declared.Stats != nil {
		one, many = BlurbSummonedCreature, BlurbSummonedCreatures
	}
	name := l.Text(one)
	if declared.Count > 1 {
		name = l.Text(many)
	}
	if l == Vi && declared.Name != "" {
		name = declared.Name
	}
	if declared.Count > 1 {
		return l.Say(BlurbSummonedMany, declared.Count, name)
	}
	return l.Say(BlurbSummonedOne, name)
}

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

// describeSelfGradient is the caster's health read as a slope rather than a
// line, and it is a sentence of its own rather than a clause on the condition
// above for the same reason that one is separate: it is a different bargain.
//
// ⚠️ It quotes the **bottom** of the curve and only the bottom. Every other
// figure in a description is what the skill does when its clause holds, and there
// is no moment at which this one holds — it is worth a little at a scratch and
// everything at a sliver, so the only number an author or a player can act on is
// the one at the end. The opening says which end it is; a reader who wants the
// middle has the word "càng" and the word "more" and a straight line between.
func (l Lang) describeSelfGradient(declared skill.Skill) string {
	if declared.SelfGradient == nil {
		return ""
	}
	atEmpty := declared.Power * (scale.Base + declared.SelfGradient.AtEmpty) / scale.Base
	return l.Say(BlurbSelfGradient,
		share(atEmpty*declared.StrikeCount()), l.describeStat(declared.Scaling.Stat)) + "."
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

// SummariseSkill is one skill in one line, for a list where four of them are
// compared at once rather than read one at a time.
//
// The battle screen offers a turn's options as a column of ids, and an id is a
// name rather than an answer: nothing in "venoshock" says it is the skill that
// doubles into a poison. This is that answer, on the option's own row, so all
// four can be weighed in one look and the screen spends no extra line on it.
//
// # Why a fourth describer, when a second reading of one set of facts drifts
//
// It is the rule of this house that two readings of the same numbers part
// company, and that rule is why forge.Percent and i18n.share are documented
// beside each other rather than merged. So the reason for a fourth has to be
// stated, and it is not brevity.
//
// Describe cannot be reduced to this, because in Vietnamese its opening sentence
// **fuses** the authored flavour with the damage figure: describeOpening builds
// BlurbFlavoured out of Skill.Flavour and the share together, so "dội một thứ
// dung dịch xanh lét, chạm vào đâu là khói bốc lên tới đó, 90% công" has no seam
// to cut. The flavour is not a separable clause that could be dropped to leave a
// compact line behind. English *would* separate cleanly, because English has no
// flavour and falls back to the derived opening — and a rule that works in one
// language and not the other is not the rule. Hence a distinct composition, in
// both languages, rather than a filter over an existing one.
//
// What stops the two disagreeing is not shared prose but a shared arithmetic:
// every figure here is read off the same field with the same helper Describe
// reads it with, and TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot
// walks every shipped skill in both languages and refuses a digit run this line
// prints that Describe's does not. The wordings deliberately differ; the numbers
// may not.
//
// # What it holds, and what it leaves to the question mark
//
// Derived only, nothing authored, and no figure that is not read off the skill.
// In order, omitting whatever the skill does not have: the damage as a share of
// its scaling stat, what it gives back (a restore, then a drain — the two of the
// three healing mechanisms a skill can carry), what it calls up, the statuses it
// puts on the other side and then on itself, each amplifier, its aim when that
// is the interesting fact, and its reach and cooldown last.
//
// Left out on purpose, because the full description is one keystroke away and
// this line has sixty-two cells at the floor: accuracy, piercing and a critical
// chance. Those change what a cast is worth at the margin; the clauses above are
// what a player is choosing between. The order is the order because the row
// **clips** rather than wraps, so what falls off the end has to be the least of
// it.
//
// ⚠️ A strip is **counted, never enumerated**, and that is the one clause where
// the sixty-two cells bind rather than merely press. purify names three
// categories; the enumerated clause is 79 cells in Vietnamese on its own, longer
// than the whole line has before the aim and the cooldown are appended, so it
// could only ever arrive trimmed. It also put the raw category enum spellings on
// an English screen, because the names it enumerated come from a Vietnamese-only
// gloss table. Describe keeps enumerating; its line has room.
func (l Lang) SummariseSkill(declared skill.Skill, shapes *pattern.Book) string {
	parts := make([]string, 0, 8)
	if declared.Power > 0 {
		// The total rather than the per-strike figure, because a row comparing
		// four skills is comparing what each one lands. Describe prints both on a
		// multi-strike skill and only this one on a single-strike, so the figure
		// is in its output either way.
		parts = append(parts, l.Say(SummaryDamage,
			share(declared.TotalPower()), l.describeStat(declared.Scaling.Stat)))
	}
	if declared.Restores > 0 {
		parts = append(parts, l.Say(SummaryRestores,
			share(declared.Restores), l.describeStat(declared.Scaling.Stat)))
	}
	if declared.Drains > 0 {
		parts = append(parts, l.Say(SummaryDrains, share(declared.Drains)))
	}
	if declared.Summons.Summons() {
		// Counted and named through summonSubject, which is the sentence's own
		// reading of the same three decisions rather than a second one of them.
		parts = append(parts, l.summonSubject(declared.Summons))
	}
	// The verb is not turned on the side here, as describeExtras turns it: there
	// is no verb. A clause of a name and a chance says which status and how often
	// and nothing about whose it is, which is what leaves room for the four
	// clauses beside it — and the two lists are told apart by SummarySelfApplies
	// naming the caster, which is the one place the distinction is load-bearing.
	for _, application := range declared.Applies {
		parts = append(parts, l.Say(SummaryStatus,
			l.stacked(application.Status, application.Stacks), share(application.Chance)))
	}
	if len(declared.SelfApplies) > 0 {
		names := make([]string, 0, len(declared.SelfApplies))
		for _, application := range declared.SelfApplies {
			names = append(names, l.stacked(application.Status, application.Stacks))
		}
		parts = append(parts, l.Say(SummarySelfApplies, l.join(names)))
	}
	// A strip is here because rapid_spin has nothing else in it — no power, no
	// application, no summon — so leaving it out made its whole row read
	// "itself · cd 3", which is the mistake describeSummon's own comment records.
	//
	// ⚠️ It is a **count and never the list Describe prints**, and that is a
	// measurement rather than a preference: purify strips three categories, and in
	// Vietnamese the enumerated clause is 79 cells on its own — longer than the
	// whole row has, before the aim and the cooldown are appended. A clause the
	// frame trims is still a clause, but one that can only ever arrive trimmed is
	// not a reading, and the enumeration also put the raw category **ids** on an
	// English screen, since the gloss table those names come from is Vietnamese
	// only. The full description keeps enumerating; that is its job.
	if declared.Strips != nil {
		parts = append(parts, l.Say(
			l.summariseStrips(declared.Strips), declared.Strips.Stacks))
	}
	if clause := l.summariseCondition(declared, declared.Requires, SummaryAmplified); clause != "" {
		parts = append(parts, clause)
	}
	if clause := l.summariseCondition(
		declared, declared.SelfRequires, SummarySelfAmplified); clause != "" {
		parts = append(parts, clause)
	}
	if declared.SelfGradient != nil {
		// The bottom of the curve, and only the bottom, for the reason
		// describeSelfGradient quotes it: there is no moment at which a gradient
		// "holds", so the end of the slope is the only figure anybody can act on.
		// The same expression, so the two cannot print different ends of it.
		atEmpty := declared.Power * (scale.Base + declared.SelfGradient.AtEmpty) / scale.Base
		parts = append(parts, l.Say(SummaryGradient, share(atEmpty*declared.StrikeCount())))
	}
	// The aim and the reach, in the one slot they share. A self-aimed skill has no
	// range to state — the same branch describeCosts takes, and for the same
	// reason: the aim is the one fact a reader cannot infer from the clauses above
	// it. An enemy is the ordinary answer and is left unsaid; an ally or both
	// sides is not, so those are named beside the range.
	if declared.Target == skill.Self {
		parts = append(parts, l.Text(BlurbCostSelf))
	} else {
		if declared.Target != skill.Enemy {
			parts = append(parts, l.describeAim(declared))
		}
		parts = append(parts, l.Say(BlurbCostRange, declared.Range))
	}
	if covered := cellsCovered(declared, shapes); covered > 1 {
		parts = append(parts, l.Say(BlurbCostCells, covered))
	}
	// A cooldown of nought is left unsaid rather than spelled "every turn". The
	// sentence has room to say a skill costs nothing to hold; a clause of sixty-two
	// cells does not, and "nothing said about a cooldown" is the honest reading of
	// a skill that has none.
	if declared.Cooldown > 0 {
		parts = append(parts, l.Say(SummaryCooldown, declared.Cooldown))
	}
	// The same middle dot the cost line is joined with, so the two readings of one
	// skill are punctuated alike.
	return strings.Join(parts, " · ")
}

// summariseStrips is which of the two counting wordings a strip gets.
//
// The claim is read off the categories the skill names rather than assumed, for
// the reason every figure in this file is read off its field. status.Category
// separates harmful from benign — Harmful exists to tell a cleanse from a dispel
// — so a skill naming nothing but harmful categories may be called a cleanse in
// as many words, and a skill naming a buff, a shield or a regen among them may
// not. An empty list cannot be authored (a strip with no category strips
// nothing), and would take the count with no claim, which is the right answer for
// it too.
func (l Lang) summariseStrips(strips *skill.Cleanse) Key {
	for _, category := range strips.Categories {
		if !category.Harmful() {
			return SummaryStrips
		}
	}
	if len(strips.Categories) == 0 {
		return SummaryStrips
	}
	return SummaryStripsHarmful
}

// summariseCondition is one amplifier as a clause: what has to be true, an
// arrow, and the share the skill lands at when it is.
//
// The arrow is doing the work a whole sentence does in Describe. What it may not
// do is say *whose* health or whose stacks are being counted — the clause inside
// it is silent about that, exactly as BlurbWhenCarrying and BlurbWhenHurt are —
// so the wording is a parameter and the caller passes the one that opens with the
// right subject. That is the same division conditionSentence makes, and it is the
// reason both callers exist: reading only Requires is a mistake this repository
// has already shipped once, in forge.PreviewDamage.
func (l Lang) summariseCondition(
	declared skill.Skill, condition *skill.Condition, wording Key) string {
	if condition == nil {
		return ""
	}
	clauses := make([]string, 0, 2)
	if condition.ReadsStatus() {
		clauses = append(clauses, l.stacked(condition.Status, condition.MinStacks))
	}
	if condition.ReadsHealth() {
		clauses = append(clauses, l.Say(SummaryHurt, share(condition.BelowHealth)))
	}
	// The same amplified figure conditionSentence prints, from the same
	// expression: a compact line quoting a different total from the sentence it
	// abbreviates would be the drift a fourth describer is on probation for.
	return l.Say(wording, l.join(clauses),
		share((declared.Power+condition.BonusPower)*declared.StrikeCount()))
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
		// A negative share is a vulnerability, and it gets its own sentence
		// rather than the resistance one with a minus in it: "refuses -30% of
		// any poison" is arithmetic on a screen, and what the trait does is the
		// opposite of refusing. The share is printed as its size, because the
		// sign is carried by the verb.
		if resistance.Amount < 0 {
			lines = append(lines, l.Say(BlurbTraitVulnerable,
				share(-resistance.Amount), l.glossed(resistance.Status)))
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
// Share is share for a caller outside this package, which is one screen: the
// chart drawing the three multipliers under the rings.
//
// Exported rather than the screen reaching for forge.Percent, which is the other
// permille-to-percent in this repository and rounds differently — it keeps a
// tenth, so 667 comes out "66.7%" there and "67%" here. Both are right for what
// they were written for; what is wrong is one figure spelt two ways on two
// screens a keystroke apart, and the elements reference already says 67%.
func Share(permille int) string { return share(permille) }

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
	return l.listed(parts, BlurbAnd)
}

// listed is the shape every list in this package is read out in: the last two
// items take the conjunction, everything before them takes a comma.
//
// It exists because the conjunction alone was put between *every* pair, so three
// items read "a and b and c" — which is not a sentence in either language. The
// two-item case is what hid it: every list the shipped data produces has one item
// or two, so no golden here has ever held a third and the defect was reachable
// only through the fixture's `purify` (three stripped categories) and through data
// nobody has authored yet.
//
// **Both languages take the same shape, and that was measured rather than
// assumed** — the open question this closes asked whether they would. English
// writes "a, b and c" and Vietnamese "a, b và c": both put the conjunction before
// the final item and a comma between the rest, and neither takes a serial comma
// ahead of the conjunction. So the comma is one key pair and the conjunction stays
// the caller's, which is what keeps a gloss list and an id list saying which they
// are while agreeing about the grammar.
//
// ⚠️ The caller owns the conjunction and this owns the comma, so a caller joining
// items that may **themselves contain a comma** would be building an ambiguous
// list. Every caller today joins short noun phrases — glossed status names, status
// categories, bare ids. The two that join *clauses* pass a slice of at most two
// (`ReadsStatus` and `ReadsHealth` are the only conditions there is), so they never
// reach the comma at all; if a third condition is ever added, that is the site to
// look at rather than this function.
func (l Lang) listed(parts []string, conjunction Key) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	last := len(parts) - 1
	return strings.Join(parts[:last], l.Text(ListComma)) +
		l.Text(conjunction) + parts[last]
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
