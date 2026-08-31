package battle

import (
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// NoActionReason is the note recorded when a unit has nothing it can use. It is
// named so that every path which passes for that reason records the same words,
// and a replay cannot diverge from the log over a turn of phrase.
const NoActionReason = "nothing usable"

// summonHorizon is how many of a summoned unit's own turns a rating is allowed
// to pay for.
//
// A summon converts one of the caster's turns into several of somebody else's,
// and that conversion is the whole of what makes one worth casting — so the
// number of turns belongs in the price, and leaving it out is what kept Suggest
// from ever casting one. But the honest horizon for a summon that stays is "the
// rest of the battle", which this rating cannot see and which would put such a
// skill above every attack in the book for ever.
//
// So the horizon is capped rather than read: a summon is priced for its own
// `lasts` when that is shorter, and for this many turns when it is longer or
// when the summon never leaves. The cap is deliberately low. Over-pricing a
// summon costs a kill — Suggest would call up a body instead of finishing a unit
// standing at a sliver of health — and under-pricing it costs a cast that was
// only marginal anyway.
const summonHorizon = 4

// Choice is an action picked for a unit.
type Choice struct {
	Skill string
	Aim   hex.Offset
}

// Suggest picks an action for the acting unit.
//
// It takes the option worth the most, in one unit: damage. Everything that is not
// damage is *priced* in damage rather than left to a fallback — a poison by the
// ticks it owes, a buff by what it adds to an attack and takes off one coming in,
// a guard by the strikes it eats, a heal by the health an enemy could otherwise
// have taken off, a cleanse by the harm it lifts, a summon by the turns it buys.
// See price.go for one rule per job and the capped horizon each is paid over.
//
// It is still shallow, and deliberately: one turn deep, no search, no memory of
// what it did last turn. What changed is that the whole timed-effect layer is now
// played rather than merely present — before this, a skill of no power was reached
// only when nothing could be hurt, so the opponent never poisoned anybody it could
// hit instead, never guarded and never healed on purpose.
//
// The fallback survives for what a price cannot reach: a skill worth nothing to
// anybody standing where they are (a taunt, a shield on a unit already at its cap)
// still gets taken when there is nothing better, exactly as it did.
//
// **Holding a skill for a later turn** is read as narrowly as a rating this
// shallow can honestly read it: not as waiting — that would need to know what next
// turn is worth, which is the lookahead this file does not have — but as refusing
// to *spend* a scarce turn on what a plentiful one buys. Damage is clamped at a
// target's remaining health, so the heaviest skill in a kit and the filler beside
// it are worth exactly the same against a unit standing at a sliver, and before
// this the tie went to whichever came first in the kit: the nuke was burnt on five
// points of health and cooled down for three turns for it. The tie now goes to the
// skill that will be there again next turn.
//
// It is a tie-break rather than a discount for two reasons. A discount would price
// scarcity, which means guessing at the turns being given up, and every guess of
// that shape in this file has been wrong in the direction of playing worse. And a
// tie is where the whole of the waste is: an option worth strictly more is worth
// more *now*, which is the only tense a one-turn-deep rating has.
//
// It reads no randomness and mutates nothing, so a client may call it to offer a
// hint without disturbing the battle's own sequence. Every price obeys that too:
// the chance an application would be rolled against is read as a weight, and a
// hypothetical unit is built beside the real one rather than out of it.
func (b *Battle) Suggest(prompt *Prompt) (Choice, bool) {
	if prompt == nil || prompt.Skipped {
		return Choice{}, false
	}
	actor, known := b.byID[prompt.Unit]
	if !known || actor.Dead {
		return Choice{}, false
	}
	best, bestValue, bestCooldown, found := Choice{}, int64(-1), 0, false
	fallback, hasFallback := Choice{}, false
	prices := b.newPricing()

	// take is the whole of the decision, and it is a pair rather than a number:
	// what an option is worth first, and only on a tie what it costs to have spent
	// it. See holding a skill for a later turn, below.
	take := func(choice Choice, value int64, cooldown int) {
		switch {
		case value > bestValue:
		case value == bestValue && cooldown < bestCooldown:
		default:
			return
		}
		best, bestValue, bestCooldown, found = choice, value, cooldown, true
	}

	for _, option := range prompt.Options {
		if !option.Available() {
			continue
		}
		declared, err := b.books.Skills.Lookup(option.Skill)
		if err != nil {
			continue
		}
		// Before the shape, because a summoning skill has no power of its own and
		// would otherwise stay the fallback it used to be: every one of them was
		// reached only when nothing could be hurt, so the shipped summoner never
		// called anybody up while it had a kunai in reach.
		if declared.Summons.Summons() {
			// A cast worth nothing falls through to the fallback below rather
			// than being rated at nought, which is what it was before this
			// branch existed. The two differ on a board with no room: a rating
			// of nought is still a rating, so it would beat "no damaging option
			// at all" and take the turn ahead of a shield or a cleanse that
			// would have done something. Every priced job below is written the
			// same way for the same reason.
			if value := b.summonWorth(actor, declared); value > 0 {
				take(Choice{Skill: option.Skill, Aim: option.Aims[0]}, value,
					declared.Cooldown)
				continue
			}
		}
		rated, priced, bestPriced := false, false, int64(0)
		for _, aim := range option.Aims {
			value := prices.rate(actor, declared, aim)
			// The best this option manages anywhere, kept even when it is a loss,
			// because a loss is the one thing the fallback below must not pick up.
			if !priced || value > bestPriced {
				bestPriced, priced = value, true
			}
			if value <= 0 {
				continue
			}
			rated = true
			take(Choice{Skill: option.Skill, Aim: aim}, value, declared.Cooldown)
		}
		// Worth nothing to anybody it could reach: the fallback, on the same terms
		// as before. The first such skill in kit order is kept, and it is taken
		// only if nothing at all was worth doing.
		//
		// ⚠️ Worth nothing and worth **less than nothing** are different, and
		// lumping them together is what made this rating cast a skill it had just
		// priced as a loss. rate subtracts friendly fire and the recoil a skill
		// puts on its own caster, so a negative total is reachable and means
		// something precise: taking this turn leaves the board worse than not
		// taking it. A taunt nobody is standing near is worth nothing and is a
		// perfectly good thing to do with a turn nobody else wants; outrage
		// against a target on a sliver is worth *less* than nothing, because the
		// damage clamps at what is left of the target while the recoil does not.
		if bestPriced < 0 {
			continue
		}
		if !rated && !hasFallback {
			fallback, hasFallback = Choice{Skill: option.Skill, Aim: option.Aims[0]}, true
		}
	}
	if found {
		return best, true
	}
	if hasFallback {
		return fallback, true
	}
	return Choice{}, false
}

// expected is the damage a skill would deal on average against a cell, summed
// over everything its shape catches and weighted by the chance each strike
// connects. It never rolls, so calling it costs nothing and changes nothing.
func (b *Battle) expected(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	shape, err := b.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return 0
	}
	actorStats := b.Stats(actor)
	// Read once, outside the loop, because that is when it is read for real.
	// ⚠️ This used to say a per-cell reading would get the same answer anyway and
	// would stop doing so the moment a caster's term read something the loop
	// changes. That moment has arrived: the gradient reads the caster's health,
	// and a draining skill heals its caster inside the loop.
	brought := swingOf(declared, actor)
	total := int64(0)
	for position, cell := range covers(shape, declared, aim) {
		target := b.occupant(cell)
		// A unit on the caster's own side is skipped rather than counted as a
		// negative: this is "expected damage", and the one skill that can hurt
		// an ally — an area skill aimed at both halves — is one Suggest never
		// reaches, because it only rates a skill aimed at the enemy. Both facts
		// are recorded here because relaxing either alone would produce an
		// opponent that happily bombs its own squad and rates it as a gain.
		if target == nil || target.Side == actor.Side {
			continue
		}
		total += b.against(actor, actorStats, declared, target, position, brought)
	}
	return total
}

// against is one target's share of what a skill would do to it, and it takes the
// unit rather than the cell for one reason: a rating asks the same question about
// a unit that is *not* on the board — the same unit holding a buff it has not been
// given, so that a stat change can be priced by the difference it would make.
//
// Reading the occupant of a cell instead is what the first version of this did,
// and the whole defensive half of the pricing was silently dead: every
// hypothetical was handed straight back to the real board and every stat change
// came out worth nothing.
func (b *Battle) against(actor *Unit, actorStats progression.Values, declared skill.Skill,
	target *Unit, position int, brought swing) int64 {
	hit := b.hitAgainst(actor, actorStats, declared, target, position, brought)
	// Expected rather than Total: a critical is a chance, and this file weights
	// chances rather than rolling them. It returns Total exactly whenever the
	// skill cannot crit, which is every skill in the book today.
	landed := b.books.Rules.Expected(hit) * int64(b.books.Rules.Chance(hit)) / combat.PermilleBase
	// Damage past a target's remaining health is wasted, so a finishing blow is not
	// rated above one that would kill twice over.
	if landed > target.HP {
		landed = target.HP
	}
	return landed
}

// hitAgainst is the combat.Hit one target would be struck with, and it is a
// function of its own so that the *chance* the blow connects has one reading
// rather than two.
//
// against wants the damage, and it weights the damage by combat.Rules.Chance.
// The reply price wants the same chance on its own — a reply answers a blow that
// landed, so a blow that misses provokes none — and building a second Hit to ask
// for it would be exactly the second copy of the resolving arithmetic price.go
// exists not to have: the two would drift the day a pierce, a gradient or an
// affinity modifier moved, and the rating would charge for a reply to a hit it
// no longer thinks it lands.
func (b *Battle) hitAgainst(actor *Unit, actorStats progression.Values, declared skill.Skill,
	target *Unit, position int, brought swing) combat.Hit {
	power := brought.applied(declared.PowerAgainst(conditionTarget(declared, target)))
	if position > 0 {
		power = power * b.books.Patterns.SplashPower / 1000
	}
	targetStats := b.Stats(target)
	multiplier := b.books.Chart.MultiplierAgainst(declared.Element, target.Affinity)
	multiplier = actor.Statuses.Modifiers().Affinity(
		multiplier, b.books.Chart.Multipliers().Neutral, b.books.Bounds)
	return combat.Hit{
		Scaling: combat.PickScaling(declared.Scaling.Source,
			actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat]),
		Multiplier:    power,
		Strikes:       declared.StrikeCount(),
		Affinity:      multiplier,
		Defense:       targetStats[progression.Defense],
		Pierce:        declared.Pierce,
		Crit:          declared.Crit,
		SkillAccuracy: declared.Accuracy,
		AccuracyStat:  actorStats[progression.Accuracy],
		DodgeStat:     targetStats[progression.Dodge],
	}
}

// summonWorth is what a cast is worth in the one unit the rest of Suggest counts
// in: damage.
//
// A summon deals none this turn, so the price is the damage the copies would
// deal over the turns they are given — their own best attack, from the cell each
// would actually stand in, against whoever is standing there to be hit, times
// the capped horizon. Everything it reads comes from the functions that do the
// real thing: summonPlaces puts the copies down, summonStats gives them their
// line, summonAffinity gives them their elements, and expected rates the attack.
// A second reading of any of those would let the rating prefer a cast for
// something the cast does not produce.
//
// ⚠️ It is an upper bound and says so. The turns are what the skill promises,
// not what the board will grant: a copy can be killed on the turn it arrives,
// and a bound one leaves the moment its summoner does. A shallow rating cannot
// know either, so it pays the promise — which is why the horizon above is capped
// low rather than honest.
//
// The hypothetical unit is built here and never enlisted, so nothing is mutated
// and no id is spent. Deliberately not through enlist: enlist appends to the
// battle, and a rating that put a unit on the board to find out what it was
// worth would be a rating a client could not call for a hint.
func (b *Battle) summonWorth(caster *Unit, declared skill.Skill) int64 {
	stats, err := b.summonStats(caster, declared.Summons)
	if err != nil {
		return 0
	}
	affinity, err := b.summonAffinity(caster, declared.Summons)
	if err != nil {
		return 0
	}
	turns := int64(summonHorizon)
	if lasts := int64(declared.Summons.Lasts); lasts > 0 && lasts < turns {
		turns = lasts
	}
	// One slot per copy and each a different one, because summonPlaces walks the
	// free list once rather than answering the same question per copy. Nothing
	// inside the loop has to book a cell against the next iteration — a pair of
	// lines here that did was dead, and read as though it were load-bearing.
	perSide, occupied := b.census()
	total := int64(0)
	for _, slot := range b.summonPlaces(caster, declared.Summons, perSide, occupied) {
		copied := &Unit{
			Side: caster.Side, Cell: hex.Place(caster.Side, slot),
			Affinity: affinity, Base: stats, HP: stats[progression.HP],
			Skills: declared.Summons.Skills,
		}
		total += b.bestStrike(copied) * turns
	}
	return total
}

// bestStrike is the most damage a unit could do on one turn with the kit it
// holds: Suggest's own rating, restricted to one unit's attacks.
//
// It walks the skill list rather than b.options, because options reads the unit's
// cooldowns by index and a unit that has never acted has none — and because a
// summon arriving with everything off cooldown is not an approximation, it is
// what enlist gives it.
//
// An all-sided attack counts, and it counts as exactly what it would do to the
// other side: expected skips a unit on the caster's own half, so the figure that
// comes back is the harm and not the cost. Leaving it out is what made a unit
// whose only attack was all-sided read as threatening nobody — so a heal on the
// ally it was about to hit was worth nothing, and a shield against it ate
// nothing. Nothing shipped is all-sided, so this is a blind spot closed rather
// than a number moved.
func (b *Battle) bestStrike(unit *Unit) int64 {
	best := int64(0)
	for _, id := range unit.Skills {
		declared, err := b.books.Skills.Lookup(id)
		if err != nil || declared.Power == 0 || !aimedAtAnEnemy(declared) {
			continue
		}
		for _, aim := range b.aims(unit, declared) {
			if value := b.expected(unit, declared, aim); value > best {
				best = value
			}
		}
	}
	return best
}

// aimedAtAnEnemy reports whether a skill can hurt the other side at all, which is
// the question both bestStrike and turnWorth are really asking. It is one function
// because the two must agree: a skill counted as an attack in one and not the
// other would make a turn worth less than the threat it poses.
func aimedAtAnEnemy(declared skill.Skill) bool {
	return declared.Target == skill.Enemy || declared.Target == skill.All
}

func requiredStatus(declared skill.Skill) string {
	if declared.Requires == nil {
		return ""
	}
	return declared.Requires.Status
}

func spentStatus(declared skill.Skill) string {
	if declared.SelfRequires == nil {
		return ""
	}
	return declared.SelfRequires.Status
}

// conditionCaster is conditionTarget for the unit doing the acting, and it is a
// second function for exactly one reason: it counts a different status.
//
// A skill may read one status of its target and another of itself, so a single
// builder taking a unit would have to be told which condition it was reading and
// would be the same mistake conditionTarget exists to prevent -- a reading that
// does not match the resolution. Two functions, each naming its own condition,
// cannot be handed the wrong one.
func conditionCaster(declared skill.Skill, actor *Unit) skill.Target {
	return skill.Target{
		Stacks:  actor.Statuses.Stacks(spentStatus(declared)),
		Health:  actor.HP,
		Maximum: actor.MaxHP(),
	}
}

// swing is everything the caster's own state puts into a use of its own skill:
// the bonus a threshold adds, and the share a gradient multiplies in.
//
// One struct rather than two parameters, because the two are ints that sit next
// to each other in every signature they travel through -- and a caller that
// handed the bonus where the share goes would compile, would silently divide the
// power by a thousand, and would look like a balance change rather than a bug.
// Naming them at the one place they are built is what makes that unwritable.
type swing struct {
	// Bonus is added to the power, from Skill.SelfBonus.
	Bonus int
	// Share is added to the multiplier, in parts per thousand, from
	// Skill.SelfScale. Nought is a skill with no gradient, or a caster at full
	// health.
	Share int
}

// applied is the power a skill lands at once the caster's own terms are in.
//
// The arithmetic is combat.Swung, and it is there rather than here because the
// authoring preview measures an unwritten skill through the same two terms: a
// figure an author reads before a write and the blow the engine lands afterwards
// have to come out of one expression. What stays here is the *reading* -- which
// unit is asked, and once per use rather than once per target.
//
// PermilleBase is added inside Swung rather than returned by combat.Gradient, so
// that nought means "nothing happened" everywhere else -- in the struct above, in
// the event log, and in the report tables.
func (s swing) applied(power int) int {
	return combat.Swung(power, s.Bonus, s.Share)
}

// swingOf reads both terms at once, and is the single reading for the same
// reason conditionTarget is: Suggest rates a skill by the power it would land
// and the battle then lands it, so a rating built from a different reading than
// the resolution would make the opponent prefer a skill for a bonus it does not
// get, with nothing anywhere reporting the disagreement.
//
// ⚠️ Read once per *use*, never once per target. A gradient asks how hurt the
// caster is, and a draining skill heals its caster inside the loop that walks a
// shape -- so a reading taken per cell would have the second cell of a column
// swing softer than the first, for a difference written on no skill. That is the
// same trap Battle.spend records, arriving through a field that has no status to
// consume.
func swingOf(declared skill.Skill, actor *Unit) swing {
	return swing{
		Bonus: declared.SelfBonus(conditionCaster(declared, actor)),
		Share: declared.SelfScale(actor.HP, actor.MaxHP()),
	}
}

// conditionTarget is what a skill's condition is allowed to read about a unit.
//
// It is one function rather than a literal at each call site because the two
// callers must agree exactly: Suggest rates a skill by the power it would land,
// and resolveAgainst then lands it. A rating built from a different reading than
// the resolution would make the opponent prefer a skill for a bonus it does not
// get, and nothing would report the disagreement.
func conditionTarget(declared skill.Skill, target *Unit) skill.Target {
	return skill.Target{
		Stacks:  target.Statuses.Stacks(requiredStatus(declared)),
		Health:  target.HP,
		Maximum: target.MaxHP(),
	}
}

// Chooser picks an action for whoever is prompted. Battle.Suggest is one.
//
// It is called once per open turn and reports false when there is nothing it can
// take, which is a legal answer: the turn is passed for it. Like Suggest, a
// Chooser must **mutate nothing and draw no randomness** — so the same battle,
// driven by the same Chooser from the same seed, replays exactly. A Chooser that
// rolled would advance the battle's own source and every draw after it would come
// out differently, which is the one way a decision here can break the replay
// contract the whole engine rests on.
//
// The type exists so that two ratings can be fought against each other rather
// than only run: see internal/forge.Bout.
type Chooser func(*Prompt) (Choice, bool)

// RunToEnd plays the battle out with Suggest choosing for both sides. It is
// RunToEndWith fixed to the rating the game ships, which is what every caller
// wanting "play this out" means.
func (b *Battle) RunToEnd(maxTurns int) (int, error) { return b.RunToEndWith(maxTurns, b.Suggest) }

// RunToEndWith plays the battle out with the given Chooser deciding every open
// turn, stopping at the turn limit so a runaway cannot hang a caller. It reports
// how many turns were taken, and the caller reads Finished to know whether that
// was the end or the limit.
//
// The limit is a backstop and nothing else. It used to be the only thing that
// noticed a battle nobody could act in, which made it stand in for an outcome
// the engine could not express; a deadlock now ends itself as a Stalemate, so
// reaching the limit means something genuinely endless is happening — two units
// buffing themselves at each other for ever — rather than a draw waiting to be
// recognised.
//
// One Chooser decides for both sides, because a Chooser is handed the prompt and
// not a side. A caller wanting a rating per side composes one that reads the
// prompted unit's side and delegates; keeping that out of here is what stops the
// engine from having an opinion about who is playing.
func (b *Battle) RunToEndWith(maxTurns int, choose Chooser) (int, error) {
	taken := 0
	for !b.finished && taken < maxTurns {
		prompt, err := b.Advance()
		if err != nil {
			return taken, err
		}
		taken++
		if prompt.Skipped {
			continue
		}
		choice, ok := choose(prompt)
		if !ok {
			// The Chooser took nothing, which is a legal answer: every skill is
			// cooling down or out of reach, or the rating declined what was on
			// offer. Either way the turn is spent and its cooldowns come down.
			if err := b.Pass(NoActionReason); err != nil {
				return taken, err
			}
			continue
		}
		if err := b.Act(choice.Skill, choice.Aim); err != nil {
			return taken, err
		}
	}
	return taken, nil
}
