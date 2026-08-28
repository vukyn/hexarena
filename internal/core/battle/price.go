package battle

import (
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// The horizons a rating is allowed to pay a non-damaging effect for, in the
// holder's own turns. They are the design decisions of this file, the way
// summonHorizon was of the summon.
//
// Every one of them is capped rather than honest, and for one reason: the honest
// horizon for a buff that keeps being refreshed, or a regeneration on a unit
// nobody is attacking, is "the rest of the battle", which this rating cannot see
// and which would put such a skill above every attack in the book for ever. The
// direction of the error is chosen too — under-pricing costs a cast that was
// marginal anyway, while over-pricing costs a kill.
//
// buffHorizon matches the duration the shipped buffs carry, so the cap binds
// only on a permanent one. guardHorizon matches a block's, because a charge that
// has expired eats nothing. healHorizon is the shortest of the three and is the
// one to lower rather than raise: a heal is the only term here that can starve
// the gated layer, since a unit kept topped up never crosses the health line a
// trait like blaze waits for.
const (
	buffHorizon  = 3
	guardHorizon = 2
	healHorizon  = 2
	killHorizon  = 2
)

// pricing is one call to Suggest, and it exists to hold the answers to two
// questions a rating asks over and over: what a unit could do on its turn, and
// what could be done to it.
//
// Both are sweeps over the board, and both are asked once per option and per aim
// without the board changing in between, so they are worked out once per unit and
// kept. The maps are read by key and never iterated, which is the rule they have
// to obey to stay out of the replay's way: an order that reached an output would
// make a battle depend on Go's map seed.
type pricing struct {
	fight   *Battle
	strikes map[string]int64
	threats map[string]int64
}

func (b *Battle) newPricing() *pricing {
	return &pricing{
		fight:   b,
		strikes: make(map[string]int64),
		threats: make(map[string]int64),
	}
}

// rate is what an option is worth, in the one unit Suggest counts in: damage.
//
// The damage half is expected and is unchanged. What is added is the other half
// of the game — the statuses a skill applies, the health it restores, the stacks
// it takes off — each priced from the function that resolves it and each over one
// of the capped horizons above. A skill that does both is worth the sum, because
// a skill that hits and poisons does both things.
//
// An all-sided skill is rated by **both** halves of what it does, which is the
// only way it can be rated at all: expected skips a unit on the caster's own side
// rather than subtracting it, so the guard that used to refuse the whole skill and
// the subtraction that replaces it are two halves of one decision. Relaxing the
// guard without friendlyFire is exactly the opponent that bombs its own squad and
// reads it as a gain — that was the reason the refusal stood, and it is answered
// here rather than removed.
func (p *pricing) rate(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	total := int64(0)
	if aimedAtAnEnemy(declared) {
		total += p.fight.expected(actor, declared, aim)
		total += p.finished(actor, declared, aim)
	}
	if declared.Target == skill.All {
		total -= p.friendlyFire(actor, declared, aim)
	}
	// SelfApplies land on the caster whatever the skill is aimed at, which is how
	// a unit shields or braces itself, so they are priced outside the shape.
	total += p.granted(actor, actor, declared, declared.SelfApplies)
	// And what a unit does to itself is not always a gift. A skill whose cost is a
	// status on its own caster — the recoil on an all-out attack — is worth its
	// damage *minus* that, and a rating that read only the gain would spend every
	// turn on the most expensive thing in the kit and call it the best. The cost is
	// the same figure the gain would be if it landed on somebody else, which is why
	// it is the same function pointed at the caster.
	total -= p.inflictedOn(actor, actor, declared, declared.SelfApplies)

	shape, err := p.fight.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return total
	}
	for _, cell := range covers(shape, declared, aim) {
		target := p.fight.occupant(cell)
		if target == nil {
			continue
		}
		friendly := target.Side == actor.Side
		// An all-sided skill reaches both halves, so neither is skipped and each
		// is read by the branch that fits it — a benefit is a gain on one's own
		// side and a cost on the enemy's, and a harm is the other way round. The
		// two branches below already say that; what the skip did was keep one of
		// them from ever running.
		if declared.Target != skill.All && (declared.Target == skill.Enemy) == friendly {
			continue
		}
		if friendly {
			total += p.restored(actor, target, declared)
			total += p.granted(actor, target, declared, declared.Applies)
			total += p.cleansed(target, declared)
			// Symmetry with the caster's own cost above: a harmful status landing
			// on one's own side is a price, not a benefit, wherever it comes from.
			total -= p.inflictedOn(actor, target, declared, declared.Applies)
		} else {
			total += p.inflictedOn(actor, target, declared, declared.Applies)
			total += p.dispelled(target, declared)
			// And a benefit landing on an enemy is a price for the same reason.
			total -= p.granted(actor, target, declared, declared.Applies)
		}
	}
	return total
}

// finished is what a lethal hit is worth beyond the health it takes off: the
// turns of damage the unit will now never deal.
//
// It exists because the two halves of the rating had different horizons, and the
// asymmetry showed up the moment support was priced at all. Damage is clamped at
// the target's remaining health, so finishing a unit standing at forty rates
// forty — while a regeneration on a nearly-dead ally is paid for two turns of the
// worst attack on the board, and a shield for the strikes it eats. So an opponent
// with a kill in reach would heal instead, which is not a deeper opponent, it is a
// worse one.
//
// A kill is priced over its own horizon, kept beside the others: healing buys
// turns of survival and killing takes turns of attacking away, so the two are the
// same exchange read from opposite ends — but they are not the same number, and
// the measured roster is what says so.
//
// ⚠️ It is not a lookahead. Nothing here asks what the target would have done, only
// what it could do — bestStrike, the figure every other term is already priced
// against — and the horizon is the same capped constant. A term that tried to
// simulate the turns it prevents would be the unbounded reading every cap here
// refuses.
func (p *pricing) finished(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	if declared.Power == 0 {
		return 0
	}
	shape, err := p.fight.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return 0
	}
	total := int64(0)
	for position, cell := range covers(shape, declared, aim) {
		// The primary cell only. A splashed target takes a reduced share of the
		// power, so asking expected what the skill would do *aimed at that cell*
		// would credit a shape with a kill its edge could not land — and pricing
		// the edge properly means re-deriving the splash share here, which is the
		// second reading of the damage this file exists not to have. Conservative
		// in the direction the rest of the file is: a shape that really would
		// finish somebody on its edge is under-priced, and under-pricing costs a
		// cast that was marginal.
		if position > 0 {
			break
		}
		target := p.fight.occupant(cell)
		if target == nil || target.Side == actor.Side {
			continue
		}
		if p.fight.expected(actor, declared, cell) < target.HP {
			continue
		}
		total += p.strike(target) * killHorizon
	}
	return total
}

// friendlyFire is what an all-sided attack costs its own side: the damage it deals
// to the caster's own units, and the turns of attacking it takes away from any of
// them it would kill.
//
// It is expected and finished pointed the other way, and it is a separate function
// rather than a sign flipped inside those two because both are asked a great many
// other questions in this file — bestStrike, turnWorth, the hypothetical units —
// and every one of them means "what could this unit do to somebody else". A skill
// that hurts its own side is the only place the caster's half of the board is
// damage at all.
//
// ⚠️ The caster is not skipped. A shape can cover the cell its own caster stands
// in, and resolveAgainst has never asked whose side a target is on, so it really
// does take the hit — a rating that left itself out would prefer the skill that
// hurts nobody but itself.
//
// ⚠️ It is an under-estimate of a squad-killing cast, on purpose and in the same
// direction as every other cap here: the turns lost with an ally are priced at
// killHorizon, exactly as an enemy's are, and no term says that losing a unit
// loses the battle. What it is enough for is the decision it exists to make —
// a bomb is worth casting when the enemy half outweighs the own half, and not
// otherwise.
func (p *pricing) friendlyFire(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	if declared.Power == 0 {
		return 0
	}
	shape, err := p.fight.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	spent := declared.SelfBonus(conditionCaster(declared, actor))
	total := int64(0)
	for position, cell := range covers(shape, declared, aim) {
		target := p.fight.occupant(cell)
		if target == nil || target.Side != actor.Side {
			continue
		}
		dealt := p.fight.against(actor, actorStats, declared, target, position, spent)
		total += dealt
		// ⚠️ Every cell, splash included — where finished reads only the primary
		// one. The two are not inconsistent: finished asks expected what the skill
		// would do *aimed at* a cell, which over-states an edge, while the share
		// this loop already holds is the reduced one. So a shape that really would
		// finish an ally on its edge is priced for it, and the case is not
		// hypothetical: this wedge catches its own caster and the unit behind it
		// on splash and nothing else.
		if dealt >= target.HP {
			total += p.strike(target) * killHorizon
		}
	}
	return total
}

// landed is the chance an application would actually be rolled against, composed
// the way inflict composes it.
//
// It never rolls. The chance is a weight here and a coin there, which is the
// whole difference between rating a thing and doing it.
//
// ⚠️ amplify reads the *acting* unit and resist the *target*, and the two take the
// same Go type — handing them the wrong way round compiles, changes every price,
// and nothing anywhere would say so. Both are pure reads of a unit's traits, which
// is what makes them callable from here at all. The clamp comes last for the
// reason inflict clamps last: composing then clamping keeps the two sides
// interchangeable.
func (p *pricing) landed(actor, target *Unit, id string, declaredChance int) int64 {
	_, amplifiedChance := p.fight.amplify(actor, id)
	chance, _ := p.fight.resist(target, id, int(raise(int64(declaredChance), amplifiedChance)))
	return int64(min(chance, scale.Base))
}

// turnsOf is how many turns a status is paid for: its own duration, or the horizon
// when that is longer — and the horizon when it has no duration at all.
//
// ⚠️ A permanent status carries **zero** duration, not a large one, so the obvious
// `min(duration, horizon)` prices every trait-granted buff, every fortification and
// every permanent debuff at nothing. It is the same shape summonWorth already had
// to get right for a summon that never leaves, and it was wrong here first: the
// defence buff in the tests rated zero while its own arithmetic said seventy-two.
func turnsOf(kind status.Kind, horizon int64) int64 {
	if kind.Permanent || kind.Duration <= 0 {
		return horizon
	}
	if int64(kind.Duration) < horizon {
		return int64(kind.Duration)
	}
	return horizon
}

// strike is the most damage a unit could deal on one of its turns, kept because
// every buff, cleanse and dispel below is priced against it.
func (p *pricing) strike(unit *Unit) int64 {
	if held, known := p.strikes[unit.ID]; known {
		return held
	}
	value := p.fight.bestStrike(unit)
	p.strikes[unit.ID] = value
	return value
}

// threat is the most damage any living enemy of this unit could land on it in one
// turn: bestStrike pointed the other way.
//
// It is what makes a defensive term finite. Health that nothing can take off is
// not worth restoring, a shield against nobody eats nothing, and the unit with no
// enemy in reach is the case both of those collapse to — so the same figure
// answers "is this worth doing" and "is there anybody to do it to".
func (p *pricing) threat(unit *Unit) int64 {
	if held, known := p.threats[unit.ID]; known {
		return held
	}
	worst := int64(0)
	for _, other := range p.fight.units {
		if other.Dead || other.Side == unit.Side {
			continue
		}
		if value := p.fight.bestAgainst(other, unit); value > worst {
			worst = value
		}
	}
	p.threats[unit.ID] = worst
	return worst
}

// restored is what a skill's healing is worth on one recipient.
//
// Three clamps, and the third is the design rather than a safety net. The first
// two are the ones heal itself applies — a heal cannot exceed what it restores,
// and cannot exceed the room there is. The third is the horizon: health above
// what an enemy could actually take off cannot be banked, and without it a heal
// outranks a kill *by construction*, because expected clamps damage at the
// target's remaining health while a full bar of room has no such ceiling.
// Finishing a unit standing at two hundred rates two hundred; topping an ally up
// by two thousand would rate two thousand.
//
// It also answers two cases nothing else covers, for free: an ally at full health
// is worth nothing, and an ally nothing can reach is worth nothing.
func (p *pricing) restored(actor, target *Unit, declared skill.Skill) int64 {
	if declared.Restores <= 0 {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	// The same expression resolveAgainst uses, so the price and the heal cannot
	// disagree about what a restore is worth.
	restored := p.fight.books.Rules.Restore(
		combat.PickScaling(declared.Scaling.Source,
			actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat]),
		declared.Restores)
	return worthHealing(restored, target, p.threat(target))
}

// worthHealing is the three clamps, shared by a restore and a regeneration so the
// two cannot drift apart on the one term that keeps either honest.
func worthHealing(restored int64, target *Unit, threat int64) int64 {
	room := target.MaxHP() - target.HP
	if restored > room {
		restored = room
	}
	if ceiling := threat * healHorizon; restored > ceiling {
		restored = ceiling
	}
	if restored < 0 {
		return 0
	}
	return restored
}

// granted is what the statuses a skill puts on its own side are worth: a buff, a
// shield, a regeneration.
//
// Each is priced by what the recipient would be *with* it, built through
// status.Set.With so the cap, the duration refresh and the wasted stack are the
// ones Apply resolves. A status already at its cap is therefore worth nothing,
// which is also the term that stops two units buffing each other for ever.
func (p *pricing) granted(actor, target *Unit, declared skill.Skill,
	applications []skill.Application) int64 {
	total := int64(0)
	for _, application := range applications {
		kind, err := p.fight.books.Statuses.Lookup(application.Status)
		if err != nil {
			continue
		}
		chance := p.landed(actor, target, kind.ID, application.Chance)
		if chance == 0 {
			continue
		}
		value := int64(0)
		switch kind.Category {
		case status.Regen:
			// The expression inflict's Regen branch freezes onto the stack: the
			// caster's own scaling stat, no defence curve and no elemental
			// multiplier, times the turns the stack will tick for.
			tick := p.fight.books.Rules.Restore(
				fromSkill(declared).stat(p.fight, actor), kind.TickPower)
			ticks := turnsOf(kind, healHorizon) * int64(application.Stacks)
			value = worthHealing(tick*ticks, target, p.threat(target))
		case status.Shield:
			value = p.shielded(target, kind, application.Stacks)
		case status.Buff:
			value = p.standing(target, kind, application.Stacks)
		}
		if value <= 0 {
			continue
		}
		total += value * chance / scale.Base
	}
	return total
}

// shielded is what block charges are worth: one charge eats one whole strike, so
// the price is the strikes they would eat and nothing else.
//
// Counted rather than saturated, because a charge is a discrete resource with a
// hard cap — and the cap read is the status book's, through With, rather than the
// second declaration of the same number that lives in the combat rules and which
// the engine never reads.
func (p *pricing) shielded(target *Unit, kind status.Kind, stacks int) int64 {
	before := target.Statuses.Stacks(kind.ID)
	after := target.Statuses.With(kind, 0, stacks)
	gained := int64(after.Stacks(kind.ID) - before)
	if gained <= 0 {
		return 0
	}
	if horizon := turnsOf(kind, guardHorizon); gained > horizon {
		gained = horizon
	}
	return gained * p.strikeThreat(target)
}

// standing is what a stat change is worth to whoever holds it, in both
// directions at once: what it adds to the best attack they could make, and what
// it takes off the worst attack that could be made on them.
//
// One hypothetical unit yields both, so a status carrying an attack term and a
// defence term is read once per stat rather than once per role. Everything about
// the numbers themselves — the saturation, the ceilings, the floor — comes from
// modifier.Set.Stat through Battle.Stats, which is the only place any of it lives.
func (p *pricing) standing(target *Unit, kind status.Kind, stacks int) int64 {
	before := target.Statuses.Stacks(kind.ID)
	terms := target.Statuses.With(kind, 0, stacks)
	// A shortcut rather than the rule. The cap is enforced inside Apply, so a
	// status already at it produces an identical hypothetical and the arithmetic
	// below would come back nought anyway — which a mutation removing these three
	// lines confirms. They are here to skip the work, not to decide the answer.
	if terms.Stacks(kind.ID) == before {
		return 0
	}
	changed := p.fight.hypothetical(target, terms)
	gained := p.fight.bestStrike(changed) - p.strike(target)
	saved := p.threat(target) - p.threatAgainst(changed)
	worth := (gained + saved) * turnsOf(kind, buffHorizon)
	worth += p.tempo(target, changed, turnsOf(kind, buffHorizon))
	if worth <= 0 {
		return 0
	}
	return worth
}

// tempo is what a change of speed is worth: the turns it adds or takes away, in
// the damage those turns are worth.
//
// A wait is `atb.Scale / speed`, so a unit's turns over a stretch of the battle are
// proportional to its speed and a share added to the stat is that share added to
// its turns. Over a horizon of H of its own turns, a speed moving from `was` to
// `now` is worth `H × (now − was) / was` extra turns — and a turn is worth what the
// unit would do with it, which is the same `bestStrike` every other term here is
// priced against.
//
// ⚠️ **It reads the stat, not the queue.** That distinction is the whole of why this
// term is allowed to exist: the earlier work left tempo out on the grounds that
// pricing it meant reading `atb.Queue`, which is state a rating must not touch and
// an ordering a rating could disagree with. It does not. Speed *is* turn frequency
// by construction, so the arithmetic above is exact for the thing it claims —
// nothing here asks who acts next, only how often this unit acts.
//
// What it deliberately does not model: where in the order the extra turn falls, and
// therefore whether it arrives before the blow that would have killed its holder.
// That is the part that would need the queue, and a term claiming it would be
// claiming a reading it cannot make.
//
// Before this, `haste`, `quickstep`, `substitution` and the speed half of every
// stat trade were worth **nothing at all** to the opponent, and so was the recoil on
// `outrage` — a skill whose cost is a slow on its own caster was priced as though it
// had no cost.
func (p *pricing) tempo(before, after *Unit, horizon int64) int64 {
	was := p.fight.Stats(before)[progression.Speed]
	now := p.fight.Stats(after)[progression.Speed]
	if was <= 0 || now == was {
		return 0
	}
	// A debuff arrives here with `now` below `was`, so the whole term comes out
	// negative and the caller subtracts it.
	return p.turnWorth(before) * horizon * (now - was) / was
}

// turnWorth is what one of a unit's turns is worth on average, and it is the one
// place in this file that does *not* use the best attack in the kit.
//
// ⚠️ That is a correction rather than a preference, and the measurement is what
// said so. An extra turn is not another cast of the unit's heaviest skill — that
// one is on cooldown most of the time — it is an ordinary turn, and pricing tempo
// against the best strike over-charged every cost by the gap between the two.
// Charged that way, `outrage`'s recoil made the dragon build *avoid* its own best
// skill: its duel rate against the fire build fell from 26.6% to 20.0%, which is a
// rating playing worse while believing it had learned something.
//
// The mean over what the unit could point at somebody is the cheapest honest
// figure. It is still an over-estimate — a real turn can miss, or be spent
// guarding — and it is deliberately the same number in both directions, so the
// turns a haste buys and the turns a slow takes away are priced identically.
func (p *pricing) turnWorth(unit *Unit) int64 {
	total, counted := int64(0), int64(0)
	for _, id := range unit.Skills {
		declared, err := p.fight.books.Skills.Lookup(id)
		if err != nil || declared.Power == 0 || !aimedAtAnEnemy(declared) {
			continue
		}
		best := int64(0)
		for _, aim := range p.fight.aims(unit, declared) {
			if value := p.fight.expected(unit, declared, aim); value > best {
				best = value
			}
		}
		if best <= 0 {
			continue
		}
		total += best
		counted++
	}
	if counted == 0 {
		return 0
	}
	return total / counted
}

// cleansed is what taking a harmful status off one's own side is worth: exactly
// what that status would have done, which is the negative of the terms above.
func (p *pricing) cleansed(target *Unit, declared skill.Skill) int64 {
	if declared.Strips == nil {
		return 0
	}
	after := target.Statuses.Without(declared.Strips.Categories, declared.Strips.Stacks)
	// The ticks the removed stacks still owed. Pending walks stacks rather than
	// statuses, so a cleanse taking one of three poison stacks is priced at one
	// stack's remaining ticks rather than at all three.
	relief := target.Statuses.Pending() - after.Pending()
	// And what the removed debuff was costing, read as a stat change lifted.
	cleaned := p.fight.hypothetical(target, after)
	relief += p.fight.bestStrike(cleaned) - p.strike(target)
	relief += p.threat(target) - p.threatAgainst(cleaned)
	if relief <= 0 {
		return 0
	}
	return relief
}

// dispelled is cleansed pointed at somebody else: what an enemy loses when a
// buff or a shield comes off it.
func (p *pricing) dispelled(target *Unit, declared skill.Skill) int64 {
	if declared.Strips == nil {
		return 0
	}
	after := target.Statuses.Without(declared.Strips.Categories, declared.Strips.Stacks)
	stripped := p.fight.hypothetical(target, after)
	// Their attack, weakened, plus what they no longer survive.
	worth := p.strike(target) - p.fight.bestStrike(stripped)
	worth += p.threatAgainst(stripped) - p.threat(target)
	if worth <= 0 {
		return 0
	}
	return worth
}

// inflictedOn is what a harmful status is worth on whoever receives it: the
// damage it will
// tick for, or the turn a control effect takes away, or the attack a debuff
// blunts.
//
// It is read as a gain when the receiver is an enemy and subtracted as a cost
// when it is the caster or an ally, which is one function rather than two because
// the figure is the same either way and only the sign is a matter of who is
// standing there.
//
// This is the term that buys the detonate setup, and it buys it without a single
// line of lookahead. A skill of no power that lands a poison is now worth the
// poison, so it gets cast — and the skill that spends the poison is already rated
// correctly the turn it is available, because conditionTarget is deliberately the
// one builder Suggest and resolveAgainst share. A term that tried to price "this
// unlocks that" would double-count the status and would need a horizon over
// future turns, which is the unbounded reading every cap in this file exists to
// refuse.
//
// ⚠️ It is an upper bound and says so, the way summonWorth does. Pricing a burn's
// remaining ticks slightly over-counts when the caster's own detonate is about to
// consume them. The direction is the accepted one: over-pricing costs a kill,
// under-pricing costs a marginal cast.
func (p *pricing) inflictedOn(actor, target *Unit, declared skill.Skill,
	applications []skill.Application) int64 {
	total := int64(0)
	for _, application := range applications {
		kind, err := p.fight.books.Statuses.Lookup(application.Status)
		if err != nil {
			continue
		}
		chance := p.landed(actor, target, kind.ID, application.Chance)
		if chance == 0 {
			continue
		}
		value := int64(0)
		switch kind.Category {
		case status.Dot:
			before := target.Statuses.Pending()
			// The expression inflict's Dot branch freezes onto the stack, origin
			// and all: the skill's own scaling stat and its own element, the
			// target's full defence even when the skill pierces, and the acting
			// unit's amplifier.
			from := fromSkill(declared)
			amplifiedEffect, _ := p.fight.amplify(actor, kind.ID)
			tick := raise(p.fight.books.Rules.Damage(
				from.stat(p.fight, actor),
				p.fight.Stats(target)[progression.Defense],
				kind.TickPower,
				p.fight.books.Chart.MultiplierAgainst(from.Element, target.Affinity),
			), amplifiedEffect)
			after := target.Statuses.With(kind, tick, application.Stacks)
			value = after.Pending() - before
			// Clamped at what is left of the target, exactly as expected clamps a
			// strike: ticks past a unit's remaining health are never taken, so a
			// poison on somebody standing at a sliver is worth the sliver.
			//
			// Without this the term is the largest number in the rating by a wide
			// margin — three stacks over three turns against one hit — and the
			// opponent spends the battle re-poisoning a corpse-in-waiting instead
			// of finishing it. It is the same clamp for the same reason, and
			// leaving it out is what made the shipped roster read nineteen points
			// further apart than it does.
			if value > target.HP {
				value = target.HP
			}
		case status.Control:
			// A turn taken off somebody is a turn of theirs nobody has to answer,
			// so it is worth what they would have done with it.
			value = p.strike(target) * turnsOf(kind, buffHorizon)
		case status.StatDebuff:
			value = p.standingLost(target, kind, application.Stacks)
		}
		if value <= 0 {
			continue
		}
		total += value * chance / scale.Base
	}
	return total
}

// standingLost is standing read as harm: what a debuff takes off its holder is
// what it is worth to whoever applied it.
func (p *pricing) standingLost(target *Unit, kind status.Kind, stacks int) int64 {
	before := target.Statuses.Stacks(kind.ID)
	terms := target.Statuses.With(kind, 0, stacks)
	if terms.Stacks(kind.ID) == before {
		return 0
	}
	weakened := p.fight.hypothetical(target, terms)
	lost := (p.strike(target) - p.fight.bestStrike(weakened)) * turnsOf(kind, buffHorizon)
	lost += (p.threatAgainst(weakened) - p.threat(target)) * turnsOf(kind, buffHorizon)
	// A slow is the mirror of a haste: the turns it takes off somebody are turns
	// nobody has to answer, which is the same figure with the sign of who holds it.
	lost -= p.tempo(target, weakened, turnsOf(kind, buffHorizon))
	if lost <= 0 {
		return 0
	}
	return lost
}

// strikeThreat is threat for a single strike rather than a whole turn, which is
// what a block charge cancels.
func (p *pricing) strikeThreat(unit *Unit) int64 {
	strikes := p.worstStrikes(unit)
	if strikes < 1 {
		strikes = 1
	}
	return p.threat(unit) / int64(strikes)
}

// worstStrikes is how many strikes the heaviest attack aimed at this unit would
// come in, so a charge is priced against one of them rather than all.
func (p *pricing) worstStrikes(unit *Unit) int {
	strikes := 1
	for _, other := range p.fight.units {
		if other.Dead || other.Side == unit.Side {
			continue
		}
		for _, id := range other.Skills {
			declared, err := p.fight.books.Skills.Lookup(id)
			if err != nil || declared.Power == 0 || !aimedAtAnEnemy(declared) {
				continue
			}
			if count := declared.StrikeCount(); count > strikes {
				strikes = count
			}
		}
	}
	return strikes
}

// threatAgainst is threat computed against a hypothetical version of a unit,
// which is the half of a stat change a memoised figure cannot answer.
func (p *pricing) threatAgainst(unit *Unit) int64 {
	worst := int64(0)
	for _, other := range p.fight.units {
		if other.Dead || other.Side == unit.Side {
			continue
		}
		if value := p.fight.bestAgainst(other, unit); value > worst {
			worst = value
		}
	}
	return worst
}

// bestAgainst is the most damage one unit could do to another in a turn: the same
// arithmetic expected uses, restricted to a single victim.
//
// It takes the victim rather than its cell, which is what lets one function answer
// both halves of a stat change — what a unit has to fear as it stands, and what it
// would have to fear holding a buff nobody has given it yet. Reading the occupant
// of a cell instead is what the first version did, and it made the whole defensive
// half of the pricing silently dead: every hypothetical was handed straight back
// to the real board, so every stat change came out worth nothing.
func (b *Battle) bestAgainst(actor, victim *Unit) int64 {
	best := int64(0)
	for _, id := range actor.Skills {
		declared, err := b.books.Skills.Lookup(id)
		if err != nil || declared.Power == 0 || !aimedAtAnEnemy(declared) {
			continue
		}
		reaches := false
		for _, aim := range b.aims(actor, declared) {
			if aim == victim.Cell {
				reaches = true
				break
			}
		}
		if !reaches {
			continue
		}
		if value := b.against(actor, b.Stats(actor), declared, victim, 0,
			declared.SelfBonus(conditionCaster(declared, actor))); value > best {
			best = value
		}
	}
	return best
}

// hypothetical is a unit as it would stand holding a different set of statuses.
//
// It is summonWorth's copy generalised, and the generalisation is the point: that
// function already had to build a unit nobody enlisted in order to price a cast,
// and every term in this file needs the same thing for a unit already on the
// board.
//
// ⚠️ The statuses arrive as a set built by status.Set.With, which deep-copies.
// Handing this function the real unit's set and applying to it would refresh the
// durations of everything the unit already holds, from inside the one function in
// the engine that promises to mutate nothing — and no golden would say so.
func (b *Battle) hypothetical(unit *Unit, terms status.Set) *Unit {
	return &Unit{
		ID:       unit.ID,
		Side:     unit.Side,
		Cell:     unit.Cell,
		Affinity: unit.Affinity,
		Base:     unit.Base,
		HP:       unit.HP,
		Skills:   unit.Skills,
		Passives: unit.Passives,
		Statuses: terms,
	}
}
