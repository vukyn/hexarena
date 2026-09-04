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
//
// There are two costs of acting subtracted here and not one. friendlyFire is what
// a skill does to the caster's own side; replied is what the units it hurts do
// back. They are the same shape for the same reason — a rating that cannot see a
// cost prefers the option that carries it — and the second was missing for as
// long as this file did not mention passive.Replies at all.
func (p *pricing) rate(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	total := int64(0)
	from := fromSkill(declared)
	if aimedAtAnEnemy(declared) {
		dealt := p.fight.expected(actor, declared, aim)
		total += dealt
		total += p.finished(actor, declared, aim)
		// And what the charge already on the board deals when this cast sets it
		// off, which nothing here charged for at all.
		total += p.discharged(actor, declared, aim)
		// And the health the blow takes back, which nothing here charged for
		// until 2026-09-02: a plain hit rated above the same hit returning nine
		// tenths of itself as healing.
		total += p.drained(actor, declared, dealt)
	}
	if declared.Target == skill.All {
		total -= p.friendlyFire(actor, declared, aim)
	}
	// And what the units it hurts hurt back. It is friendlyFire's other half — a
	// cost of acting rather than a gain from it — and it is subtracted for the
	// same reason: a rating that cannot see it prefers the attack that answers
	// itself.
	total -= p.replied(actor, declared, aim)
	// The health the skill charges its own caster, subtracted for every skill
	// that has one rather than for an all-sided one.
	//
	// ⚠️ **It lived inside friendlyFire first, and that arm only runs when the
	// target side is All.** So a single-target skill that cost a quarter of its
	// caster's health was charged nothing at all: measured, a Magnezone holding
	// one cast it three times a battle, handed over seven tenths of itself and
	// lost 120 of 120 duels — against 69-51 for the same kit without it. A cost
	// filed in a branch that does not run is a cost nobody pays.
	//
	// ⚠️ Priced as the health itself rather than as the turns it costs. That
	// under-states it on a thin frame, and the alternative is worse: charging it
	// at the caster's own worth needs a horizon nothing here has, and a rating
	// that guessed at one would decline a skill for a turn that never came.
	// Health is the unit the rest of this file counts in, and it is what the
	// caster actually hands over.
	total -= p.spentHealth(actor, declared)
	// And the other thing a skill charges its own caster: the counter it cashes
	// in. Subtracted here rather than anywhere nearer the gain, because the gain
	// is inside expected and the two are separate readings on purpose — see
	// spentCounter.
	total -= p.spentCounter(actor, declared)
	// SelfApplies land on the caster whatever the skill is aimed at, which is how
	// a unit shields or braces itself, so they are priced outside the shape.
	total += p.granted(actor, actor, from, declared.SelfApplies)
	// And what a unit does to itself is not always a gift. A skill whose cost is a
	// status on its own caster — the recoil on an all-out attack — is worth its
	// damage *minus* that, and a rating that read only the gain would spend every
	// turn on the most expensive thing in the kit and call it the best. The cost is
	// the same figure the gain would be if it landed on somebody else, which is why
	// it is the same function pointed at the caster.
	total -= p.inflictedOn(actor, actor, from, declared.SelfApplies)

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
		// A condition's own riders are paid only where it holds, and the rating
		// has the target in hand -- so this asks the same question the resolution
		// asks rather than an estimate of it. Folded into one list because every
		// arm below wants the same answer about all of them, and a second pass
		// would be a second place for the friendly and the hostile readings to
		// part company.
		riders := declared.Applies
		if declared.Requires.AppliesOnHold() &&
			declared.Amplified(conditionTarget(declared, target)) {
			riders = append(append([]skill.Application(nil), riders...), declared.Requires.Applies...)
		}
		if friendly {
			total += p.restored(actor, target, declared)
			total += p.granted(actor, target, from, riders)
			total += p.cleansed(target, declared)
			// Symmetry with the caster's own cost above: a harmful status landing
			// on one's own side is a price, not a benefit, wherever it comes from.
			total -= p.inflictedOn(actor, target, from, riders)
		} else {
			total += p.inflictedOn(actor, target, from, riders)
			total += p.dispelled(target, declared)
			// And a benefit landing on an enemy is a price for the same reason.
			total -= p.granted(actor, target, from, riders)
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
	total := int64(0)
	shape, err := p.fight.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	// Read once, outside the loop, exactly as expected does: the caster is the
	// same caster for every cell its own bomb catches, and a gradient asks how
	// hurt it is.
	brought := swingOf(declared, actor)
	for position, cell := range covers(shape, declared, aim) {
		target := p.fight.occupant(cell)
		if target == nil || target.Side != actor.Side {
			continue
		}
		dealt := p.fight.against(actor, actorStats, declared, target, position, brought)
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

// spentHealth is what a skill ASKS of its caster, which is deliberately not what
// a caster on its last legs would actually hand over.
//
// ⚠️ **The floor Battle.spendHealth applies is left out of the price on purpose,
// and the first version included it.** That version read the clamped figure — the
// health there is, rather than the health asked for — so the skill got *cheaper*
// exactly as casting it got more fatal: a unit at two hundred was charged a
// hundred and ninety nine instead of seven hundred and fifty, and the rating
// therefore liked it best in the one position where it should refuse. Measured
// that way, a Magnezone cast it eighty-eight times and did it almost entirely
// while nearly dead.
//
// So this is the one figure in the file that is deliberately not the one the
// resolving function pays. The rule the file is written under — read the
// resolving function, never a second copy — is about arithmetic that could drift;
// this is the same arithmetic with one clamp left off, and the clamp is what a
// unit can afford rather than what the skill costs. A unit that cannot afford it
// should decline, which is what charging the full ask makes it do.
func (p *pricing) spentHealth(actor *Unit, declared skill.Skill) int64 {
	if declared.Cost <= 0 {
		return 0
	}
	return actor.MaxHP() * int64(declared.Cost) / scale.Base
}

// discharged is what the charge already sitting on the board deals when this cast
// sets it off, and it is the CASH-IN of the conduit playstyle.
//
// ⚠️ **It was priced nowhere at all.** `ArcPower` appeared in exactly one place in
// this file — `spendable`, which values a stack to decide whether *laying one
// down* is worth a turn — and the turn that spends it was rated on the skill's own
// power and nothing else. So a conduit's whole payload was a free rider: the
// rating chose `electro_ball` because six hundred and forty times two is a decent
// blow, and the discharge happened to come with it.
//
// That was invisible while nothing else moved, and it stopped being invisible the
// moment the rating learned to read a guard: **an arc is not stopped by one** —
// the single thing a conduit has over an ordinary attack, and the reason a counter
// is worth laying down in front of a wall — so a rating that starts avoiding a
// wall stops firing arcs into the one board they exist for, and the playstyle
// reads as a trap. See TODO.md.
//
// The model is the resolving loop read once per carrier rather than once per
// strike:
//
//   - The chain is `chainFrom`, the same function the discharge walks, so the aim
//     gates it identically — a chain from an uncharged unit is empty however much
//     charge is standing beside it.
//   - `Takes` says how many stacks one strike spends, and the arc fires once per
//     strike that did not MISS — a blocked strike discharges, because the guard
//     stopped the blow and not what was already on the target. So the round count
//     is the expected strikes weighted by the chance to connect, and it is capped
//     at the rounds the carrier actually has stacks for.
//   - The damage is the resolving expression itself, with the same scaling, the
//     same defence and the same chart lookup — and deliberately NOT the caster's
//     own affinity modifier, which `hitAgainst` applies and the discharge does
//     not.
//
// ⚠️ The aimed carrier's arc is clamped at what the skill's own blow will leave of
// it. Without that a conduit aimed at a unit on a sliver is worth its remaining
// health twice over, which is the same double-count `expected`'s own clamp exists
// to refuse.
func (p *pricing) discharged(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	if !declared.Requires.Arcs() {
		return 0
	}
	chain := p.fight.chainFrom(actor, declared, aim)
	if len(chain) == 0 {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	brought := swingOf(declared, actor)
	from := fromSkill(declared)
	scaling := combat.PickScaling(declared.Scaling.Source,
		actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat])
	// Rolled once for the cast and against the unit it was AIMED at, because that
	// is where combat.Roll reads its accuracy — the units further down a chain are
	// never rolled against at all.
	rounds := int64(declared.ExpectedStrikes()) *
		int64(p.fight.books.Rules.Chance(p.fight.hitAgainst(
			actor, actorStats, declared, chain[0], 0, brought))) / scale.Base
	total := int64(0)
	for at, unit := range chain {
		stacks := unit.Statuses.Stacks(declared.Requires.Status)
		take := declared.Requires.Takes(stacks)
		if take <= 0 {
			continue
		}
		amount := p.fight.books.Rules.Damage(scaling,
			p.fight.Stats(unit)[progression.Defense],
			declared.Requires.ArcPower*take,
			p.fight.books.Chart.MultiplierAgainst(from.Element, unit.Affinity))
		if amount <= 0 {
			continue
		}
		// The rounds this carrier has stacks for, in the same per mille the strike
		// count comes in.
		spent := rounds
		if capacity := int64(stacks/take) * scale.Base; spent > capacity {
			spent = capacity
		}
		value := amount * spent / scale.Base
		room := unit.HP
		if at == 0 {
			room -= p.fight.against(actor, actorStats, declared, unit, 0, brought)
		}
		if value > room {
			value = room
		}
		if value > 0 {
			total += value
		}
	}
	return total
}

// drained is the health a skill takes back out of the damage it deals, priced as
// healing on the caster.
//
// ⚠️ **Four shipped things were worth nothing here** — `leech_seed`,
// `dream_eater`, `blood_thirst` and `last_gasp` — and the measurement was blunt:
// offered a plain hit and the same hit returning nine tenths of itself, the
// rating took the plain one.
//
// A drain is the exact mirror of `restores`: health arriving on a unit that has
// room for it. So it goes through worthHealing and collects the same three
// clamps, the horizon included — the one that stops a heal outranking a kill by
// construction, because expected clamps damage at what is left of a target while
// a full bar of room has no such ceiling.
//
// The share comes from drainShare over the skill's own figure plus the caster's
// traits, which is the expression resolveAgainst pays it with, cap and all: a
// share of damage dealt is not a chance, so two drains simply both drain and the
// sum is bounded rather than saturated. Reading only `declared.Drains` would have
// priced `leech_seed` and left `blood_thirst` at nought, which is the shape of
// half a fix.
//
// ⚠️ It reads damage **dealt** rather than damage rolled, which is what the field
// means: a strike that missed drains nothing, and `expected` is already the
// figure with the chance to hit weighted into it.
//
// ⚠️ Not clamped a second time against the damage. worthHealing's room clamp is
// what bounds it, and drainShare has already bounded the share at the base — the
// conservation the resolving side enforces is that health taken back cannot
// exceed damage dealt, and a share of at most one of the damage cannot.
func (p *pricing) drained(actor *Unit, declared skill.Skill, dealt int64) int64 {
	share := drainShare(declared.Drains + p.fight.lifesteal(actor))
	if share <= 0 || dealt <= 0 {
		return 0
	}
	return worthHealing(dealt*int64(share)/scale.Base, actor, p.threat(actor))
}

// replied is what an attack costs its own caster in answers: the damage the
// units it hurts strike back with, the statuses they put on it, and the turns it
// loses if one of them finishes it.
//
// It is the exact mirror of friendlyFire — a cost of acting, subtracted by rate,
// priced from the functions that resolve it — and it exists because price.go did
// not mention passive.Replies at all. Attacking a venom_blood or a thorns holder
// was charged nothing for the blow that comes back, which is the same blind spot
// friendlyFire was written to close on the other side of the board.
//
// # Whose reply, and when
//
// Battle.answer runs over every unit the skill actually damaged, and it never
// asks which side they are on — so an all-sided or multi-target skill provokes
// several answers, and an ally caught by the shape answers its own caster. This
// walks the same cells in the same order. Three units are skipped for the three
// reasons answer skips them: the caster itself (a skill that caught its own
// caster is still not somebody attacking it), a holder the blow would kill (the
// dead do not answer), and a holder whose gate is shut.
//
// # Once per cast, never once per strike
//
// answer is called after the whole skill loop, once per holder, and its own
// comment says why: a reply answers a *use* of a skill rather than a strike, so
// a trait's worth cannot scale with somebody else's strike count. So a
// three-strike skill is charged for exactly one answer per holder, and a rating
// that multiplied by StrikeCount would decline the triple and take the single
// for a difference the engine does not resolve.
//
// # The chance
//
// Two of them, and both are weighted rather than rolled. A reply answers a blow
// that *landed*, so the whole per-holder charge is weighted by the chance the
// attack connects — read from combat.Rules.Chance through the one Hit
// Battle.hitAgainst builds, which is the same Hit against weights the damage
// with. And the statuses a reply applies go through inflictedOn, which composes
// the holder's amplifier with the attacker's resistance in landed exactly as
// inflict does — which is what makes venom_blood's poison cost nothing at all
// against a target that refuses poison.
//
// ⚠️ The connect chance is one strike's, not "at least one of them", so a
// multi-strike skill is under-charged by the gap. That is the accepted direction
// and it is deliberately not corrected here: the correction is a second
// expression over StrikeCount, and combat exposes no helper for it.
//
// # The horizon
//
// None on the damage: a reply is health taken off *now* rather than over a
// stretch of turns, so it is charged at its face value clamped at what the
// caster has left. The status half has no horizon here either — it goes through
// inflictedOn, which already prices an inflicted status over the horizons this
// file declares, rather than through a second copy of them.
//
// # A dead attacker
//
// A reply may kill, and Battle.reply gives damage no exemption for arriving out
// of turn. So the caster's health is tracked down the walk the way reply spends
// it, and when an answer would take the last of it two things follow, both read
// off the resolving code rather than assumed:
//
//   - the caster's own death is charged the way friendlyFire charges an ally's,
//     at strike × killHorizon — the turns of attacking it will now never take;
//   - nothing after that is charged at all, because reply returns before its own
//     statuses land and answer returns before the next holder gets a turn.
//     Under-charging what a corpse would still have suffered is the accepted
//     direction, and it is also simply what happens.
//
// Getting this the other way round is the failure worth naming: a rating blind
// to it walks its own unit into a reply that kills it, and a rating that charged
// the death without the return would decline every attack on a board with two
// repliers on it.
//
// # Why it cannot double-count
//
// Nothing else in this file reads passive.Replies — expected, finished,
// friendlyFire, threat, bestAgainst and turnWorth are all built out of the skill
// book, and a reply is not a skill. TestNoOtherTermChargesForAReply holds that,
// by pricing an option against a bare target and against a replier and showing
// the whole of the difference is this term.
func (p *pricing) replied(actor *Unit, declared skill.Skill, aim hex.Offset) int64 {
	// A skill of no power bites nobody, and Act returns before the shape for a
	// self-aimed one — so neither reaches answer, and neither is charged.
	if declared.Power == 0 || declared.Target == skill.Self || p.fight.books.Passives == nil {
		return 0
	}
	shape, err := p.fight.books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	// Read once, outside the loop, exactly as expected and friendlyFire do.
	brought := swingOf(declared, actor)
	// What the caster has left as the answers land, spent down the walk the way
	// reply spends it. In expectation rather than in health: every other figure
	// in this file is an expectation, and a reply that only half arrives should
	// only half kill.
	remaining := actor.HP
	total := int64(0)
	for position, cell := range covers(shape, declared, aim) {
		holder := p.fight.occupant(cell)
		// answer skips the caster by identity, and so does this: a shape that
		// covers the cell it was cast from hurts its caster, but nobody is
		// attacking the caster.
		if holder == nil || holder == actor {
			continue
		}
		dealt := p.fight.against(actor, actorStats, declared, holder, position, brought)
		// Not bitten, so not in answer's list. And a holder the blow finishes does
		// not answer, which is the same reading finished and friendlyFire take of
		// the same figure.
		if dealt <= 0 || dealt >= holder.HP {
			continue
		}
		connected := int64(p.fight.books.Rules.Chance(
			p.fight.hitAgainst(actor, actorStats, declared, holder, position, brought)))
		if connected <= 0 {
			continue
		}
		for _, id := range holder.Passives {
			held, err := p.fight.books.Passives.Lookup(id)
			if err != nil || !held.Replies.Answers() {
				continue
			}
			// ⚠️ The gate is read on the holder as the board stands, while answer
			// reads it on the holder as this very blow left it. The two can only
			// differ one way: passive.Condition carries nothing but BelowHealth, so
			// a gate can turn *on* as its holder is hurt and can never turn off. So
			// this misses a trait the attack itself wakes up and over-charges for
			// none, which is the direction every cap in this file errs in — and it
			// costs no hypothetical unit to be exact about.
			if !p.fight.inForce(holder, held) {
				continue
			}
			cost, lethal := int64(0), false
			if held.Replies.Power > 0 {
				damage, _ := p.fight.replyDamage(holder, actor, held)
				if damage >= remaining {
					damage, lethal = remaining, true
				}
				cost += damage
				remaining -= damage * connected / scale.Base
			}
			if lethal {
				// The turns the caster will now never take, at the same horizon
				// friendlyFire charges for an ally it kills. reply returns before
				// its own statuses land, so they are not added.
				cost += p.strike(actor) * killHorizon
			} else {
				from := fromTrait(held)
				// The harm the answer puts on the caster, and — for symmetry with
				// rate's own two branches — anything it hands the caster read back
				// off the bill. Both through the terms that already price a status.
				cost += p.inflictedOn(holder, actor, from, held.Replies.Applies)
				cost -= p.granted(holder, actor, from, held.Replies.Applies)
			}
			// ⚠️ Clamped at nought per answer, and this is the sign guard. rate
			// *subtracts* what comes back, so a negative charge here would make an
			// attack more attractive for being answered — the opponent hunting
			// venom_blood holders, which reads as a plausible strategy rather than
			// as a bug. A reply worth less than nothing to its holder is worth
			// nothing to the unit it answers.
			if cost > 0 {
				total += cost * connected / scale.Base
			}
			if lethal {
				// answer returns the moment the attacker dies rather than breaking,
				// so no further holder answers and no further trait of this one
				// does either.
				return total
			}
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
//
// ⚠️ **Both halves of the figure, through swingOf rather than a reading of its
// own.** A heal paid for out of a reserve carries no flat `restores` at all --
// the whole payout is the condition's rate times the stacks the spend takes --
// so a price read off that field alone would rate every such skill at nought and
// Suggest would never cast one. swingOf is the function Battle.restore is handed
// its half from, so the price and the payout come out of one expression rather
// than two that agree today.
func (p *pricing) restored(actor, target *Unit, declared skill.Skill) int64 {
	brought := swingOf(declared, actor)
	total := declared.Restores + brought.Restore
	if total <= 0 {
		return 0
	}
	actorStats := p.fight.Stats(actor)
	// The same expression resolveAgainst uses, so the price and the heal cannot
	// disagree about what a restore is worth.
	restored := p.fight.books.Rules.Restore(
		combat.PickScaling(declared.Scaling.Source,
			actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat]),
		total)
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

// granted is what the statuses an application puts on its recipient are worth: a
// buff, a shield, a regeneration.
//
// Each is priced by what the recipient would be *with* it, built through
// status.Set.With so the cap, the duration refresh and the wasted stack are the
// ones Apply resolves. A status already at its cap is therefore worth nothing,
// which is also the term that stops two units buffing each other for ever.
//
// ⚠️ It takes an origin rather than the skill, because inflict does: a skill and
// a trait's reply both reach inflict, and everything this function needs to know
// about the source — which element prices it and which of the actor's stats it
// scales off — is what origin was extracted to name. Handing it the skill made a
// reply impossible to price without a second copy of the Regen expression below.
func (p *pricing) granted(actor, target *Unit, from origin,
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
				from.stat(p.fight, actor), kind.TickPower)
			ticks := turnsOf(kind, healHorizon) * int64(application.Stacks)
			value = worthHealing(tick*ticks, target, p.threat(target))
		case status.Shield:
			value = p.shielded(target, kind, application.Stacks)
		case status.Absorb:
			value = p.guarded(actor, target, from, kind, application.Stacks)
		case status.Buff:
			value = p.standing(target, kind, application.Stacks)
		case status.Taunt:
			value = p.taunting(target, kind)
		case status.Reserve:
			// A reserve reaches this branch and a charge reaches inflictedOn's,
			// which is the whole of the difference between them written where the
			// rating can see it: one is a gift to its holder and the other a mark
			// on a victim. Both go through charged, because what either is worth
			// is the same question — what somebody can do with it.
			value = p.charged(actor, target, kind, application.Stacks)
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

// guarded is what an absorbing pool is worth: the damage it will eat, bounded by
// the damage there is to eat.
//
// The pool itself comes from the expression inflict freezes onto the stack — the
// caster's own scaling stat through Rules.Restore — for the reason every price in
// this file reads the resolving function rather than a second copy of it. The
// bound is the one shielded uses and for the same argument: a guard is only worth
// what can actually be thrown at it before it runs out, so it is capped at the
// worst attack on the board times the turns the stack will live, and a barrier
// nobody can reach is worth nothing without a case saying so.
//
// ⚠️ It is NOT clamped at the holder's health the way a heal is clamped at the
// room there is. A heal above the maximum is thrown away by heal itself; a pool
// larger than the holder's remaining health is not wasted at all — it is the
// difference between dying and not.
func (p *pricing) guarded(actor, target *Unit, from origin, kind status.Kind, stacks int) int64 {
	pool := p.fight.books.Rules.Restore(from.stat(p.fight, actor), kind.PoolPower)
	if pool <= 0 {
		return 0
	}
	// ⚠️ Through With rather than multiplied by the stack count, and the first
	// version did the arithmetic instead. Apply refuses a stack past the cap, so
	// a barrier already at it gains nothing — and a price that multiplied would
	// have rated putting one up on a unit that already had one, every turn,
	// forever. That is the same term that stops two units buffing each other for
	// ever, and it is exactly what shielded and standing read the same way.
	before := target.Statuses.PoolIn(status.Absorb)
	after := target.Statuses.With(kind, pool, stacks)
	gained := after.PoolIn(status.Absorb) - before
	if gained <= 0 {
		return 0
	}
	if ceiling := p.threat(target) * turnsOf(kind, guardHorizon); gained > ceiling {
		gained = ceiling
	}
	if gained < 0 {
		return 0
	}
	return gained
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
// buff, a shield or a regeneration comes off it.
//
// ⚠️ **There are three things a strip can take and this priced one of them.** A
// stat buff moves a number, so the hypothetical below reads it for free — but a
// shield and a regeneration move no stat at all, so both came back nought and an
// opponent handed a dispel would never have cast one. Measured before the two
// terms were added: an actor holding a strip and a ten-power poke chose the poke
// against an enemy carrying three block charges, three regeneration stacks, and
// both at once.
//
// The two terms are the inverses of the ones that price *putting* each on, and
// they are read from those functions rather than written again — which is the
// rule this file exists under, and here it is also the only way the price and
// the effect can agree about what a charge or a tick was worth.
func (p *pricing) dispelled(target *Unit, declared skill.Skill) int64 {
	if declared.Strips == nil {
		return 0
	}
	after := target.Statuses.Without(declared.Strips.Categories, declared.Strips.Stacks)
	stripped := p.fight.hypothetical(target, after)
	// Their attack, weakened, plus what they no longer survive.
	worth := p.strike(target) - p.fight.bestStrike(stripped)
	worth += p.threatAgainst(stripped) - p.threat(target)
	worth += p.unguarded(target, after)
	worth += p.unbarriered(target, after)
	worth += p.unfuelled(target, after)
	worth += p.undone(target, after, declared.Strips.Categories)
	if worth <= 0 {
		return 0
	}
	return worth
}

// unguarded is what taking block charges off an enemy is worth: the strikes they
// would have eaten, which is exactly what shielded pays for putting them on.
//
// Clamped at the guard horizon for shielded's reason rather than as a safety
// net: a charge that outlives the horizon was never going to be spent inside it,
// and counting it would make a dispel worth more than the shield it removes.
func (p *pricing) unguarded(target *Unit, after status.Set) int64 {
	taken := int64(target.Statuses.CountIn(status.Shield) - after.CountIn(status.Shield))
	if taken <= 0 {
		return 0
	}
	if taken > guardHorizon {
		taken = guardHorizon
	}
	return taken * p.strikeThreat(target)
}

// unbarriered is the pool a removed absorb still had, which is damage the enemy
// will now take after all.
//
// The mirror of unguarded one function up, and separate from it because the two
// count different things: a charge is worth a whole strike and a pool is worth
// exactly itself. Bounded by the same argument — what the board can actually
// throw before the stack expires — read through the horizon rather than through
// the stack's own duration, because a strip does not know which stack it took.
func (p *pricing) unbarriered(target *Unit, after status.Set) int64 {
	taken := target.Statuses.PoolIn(status.Absorb) - after.PoolIn(status.Absorb)
	if taken <= 0 {
		return 0
	}
	if ceiling := p.threat(target) * guardHorizon; taken > ceiling {
		taken = ceiling
	}
	return taken
}

// undone is the healing a removed regeneration still owed, run through the same
// three clamps a heal is: health above the room there is cannot be banked, and
// health above what this side could take off cannot be denied.
//
// ⚠️ It is restricted to strips that name nothing harmful, and the restriction is
// the whole of what keeps the sign honest. Pending totals every stack's frozen
// tick without asking whether it heals or hurts, so a strip naming `dot` would be
// taking a poison OFF an enemy and the same difference would report that as a
// gain. Nothing shipped does that; the guard is here because the arithmetic
// cannot tell, not because the data currently would.
func (p *pricing) undone(target *Unit, after status.Set, categories []status.Category) int64 {
	for _, category := range categories {
		if category.Harmful() {
			return 0
		}
	}
	denied := target.Statuses.Pending() - after.Pending()
	if denied <= 0 {
		return 0
	}
	return worthHealing(denied, target, p.threat(target))
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
// ⚠️ It takes an origin rather than the skill, for the reason granted does: a
// trait's reply reaches inflict through the very same function, so the only way
// to price one without a second copy of the Dot expression below is to name the
// three things inflict actually reads off its source.
func (p *pricing) inflictedOn(actor, target *Unit, from origin,
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
			// and all: the source's own scaling stat and its own element, the
			// target's full defence even when the skill pierces, and the acting
			// unit's amplifier.
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
		case status.Charge:
			value = p.charged(actor, target, kind, application.Stacks)
		case status.HealCut:
			value = p.uncured(target, kind, application.Stacks)
		}
		if value <= 0 {
			continue
		}
		total += value * chance / scale.Base
	}
	return total
}

// taunting is what a taunt is worth to WHOEVER HOLDS IT, and where it sits is the
// whole of why this is a `granted` case and not an `inflictedOn` one.
//
// ⚠️ **The status goes on the unit doing the taunting, not on the unit being
// taunted** — `battle.tauntStatus` says so in as many words, and the shipped
// `taunt` is a self-aimed skill whose whole body is `self_applies`. The first cut
// of this priced it in `inflictedOn`, which is where a HARMFUL self-application is
// charged as a cost, so a unit that taunted was billed its own best strike three
// times over for doing it. That is worse than the nothing it was worth before.
//
// What it buys is the aim it takes off every enemy at once: each of them wanted
// its best cast and is left with whatever it can do to the holder. A taunt is not
// a Control and pricing it as one would be wrong the other way — a stunned unit
// does not act, a taunted one acts and is simply not allowed to pick.
//
// ⚠️ **An enemy the holder was already the best target of contributes nothing**,
// and that is the honest answer rather than a missing case: a taunt that changes
// nobody's mind changes nothing. It is also what keeps the term from paying for
// the taunt a squishy unit should not be casting — the value is the denial, and
// what it costs to stand in the way is not modelled, so the board where this
// would over-price is the board where the difference is nought anyway.
func (p *pricing) taunting(holder *Unit, kind status.Kind) int64 {
	denied := int64(0)
	for _, other := range p.fight.units {
		if other.Dead || other.Side == holder.Side {
			continue
		}
		best := p.strike(other)
		if narrowed := p.fight.bestAgainst(other, holder); best > narrowed {
			denied += best - narrowed
		}
	}
	return denied * turnsOf(kind, buffHorizon)
}

// uncured is what cutting somebody's healing denies them, priced as the healing
// itself.
//
// ⚠️ **It was worth nothing too**, and the consequence was written down when
// `fester` shipped: the opponent never aimed one at a healer on purpose. The cut
// is read through `healingFor`, which is the expression the battle pays a heal
// with — the floor at total negation included — so the two cannot disagree about
// how much of a heal a stack takes.
//
// ⚠️ **Only the regeneration already ticking on the target is counted**, and that
// is deliberately less than the truth: a cut applies to every heal the holder
// receives, including a restore an ally has not cast yet and a drain nobody has
// rolled. Those need a lookahead this file does not have and will not grow, so
// what is priced is the healing that is visibly owed. Under-pricing costs a
// marginal cast, which is the direction every cap here errs in.
//
// Clamped by worthHealing for the reason `undone` is: health above the room there
// is cannot be denied, and neither can health above what this side could take off.
func (p *pricing) uncured(target *Unit, kind status.Kind, stacks int) int64 {
	owed := target.Statuses.PendingIn(status.Regen)
	if owed <= 0 {
		return 0
	}
	before, _ := healingFor(target, owed)
	after, _ := healingFor(
		p.fight.hypothetical(target, target.Statuses.With(kind, 0, stacks)), owed)
	denied := before - after
	if denied <= 0 {
		return 0
	}
	return worthHealing(denied, target, p.threat(target))
}

// charged is what a counter is worth to whoever puts it on, which is not a
// question about the counter at all.
//
// A charge changes nothing on its holder — no stat, no tick, no turn — so read
// the way every other branch here reads its category, it is worth exactly
// nothing, and a rating that said so would never spend a turn putting one on and
// the whole playstyle would be unreachable by the opponent. What it is worth is
// what somebody can *do* with it, so that is what is asked.
//
// ⚠️ **It is nought when nobody on the caster's side carries a skill that spends
// it, and that is the honest answer rather than a missing case.** A charge on a
// squad with no consumer is a turn thrown away, and a rating that valued it
// anyway would have a unit carefully electrifying a board it has no way to cash.
func (p *pricing) charged(actor, target *Unit, kind status.Kind, stacks int) int64 {
	before := target.Statuses.Stacks(kind.ID)
	after := target.Statuses.With(kind, 0, stacks)
	gained := int64(after.Stacks(kind.ID) - before)
	if gained <= 0 {
		return 0
	}
	// ⚠️ **A pile is worth far less than a stack times its height, and pricing it
	// that way is a measured mistake rather than a theoretical one.** A consumer
	// cashes one stack per cast, so the second stack needs one more turn to go
	// right than the first, the third needs two, and a battle is roughly fifteen
	// turns long. Valued linearly, one cast of a three-stack charge over two cells
	// read as three whole strikes of value, and a rating handed that figure spends
	// its opening turn on a skill that deals no damage — which against a striking
	// squad is simply losing. The kit measured 6 per mille against 113 for the same
	// kit with that skill taken out.
	//
	// So each stack is worth half the one before it. The sum can never pass twice
	// the first stack however high the pile goes, which is the shape of the real
	// thing: the top of a pile is speculative, and no amount of charging is worth
	// two turns of hitting somebody.
	// The halving starts from what is already there, not from nothing: a second
	// cast onto a target that is charged already is adding to the top of the pile,
	// and it is the *height* that makes a stack speculative rather than the cast
	// that put it there. Without this a charger is worth its full opening every
	// turn, which is the same mistake one step further along.
	return p.pile(actor, target, kind, before, int(gained))
}

// pile is what a run of counter stacks is worth, and it is the one place the two
// counters are priced by different arithmetic.
//
// ⚠️ **The halving above is right for a charge and wrong for a reserve, and the
// reason is a sentence from that comment.** "A consumer cashes one stack per
// cast, so the second stack needs one more turn to go right" is exact for a
// conduit: one blow, one charge. A reserve spender cashes a WHOLE RUN at once, so
// the second stack does not need another turn — it goes off in the same cast as
// the first, and pricing it at half is pricing a mechanic as if it were the one
// it was built to be the opposite of.
//
// So a reserve is flat up to what its holder can actually cash in one go, and
// speculative — the same halving — above that. The flat region is bounded by the
// KIT rather than by a horizon: stacks past the deepest spend the holder owns
// buy nothing this cast, and a second cast is exactly the speculation the halving
// was written for. A holder with no spender has a capacity of nought, so every
// stack is speculative and spendable has already answered nought anyway.
func (p *pricing) pile(actor, target *Unit, kind status.Kind, under, count int) int64 {
	worth := p.spendable(actor, target, kind)
	if kind.Category != status.Reserve {
		return pileWorth(worth, under, count)
	}
	flat := 0
	if room := p.capacity(target, kind.ID) - under; room > 0 {
		flat = min(room, count)
	}
	return int64(flat)*worth + pileWorth(worth, under+flat, count-flat)
}

// capacity is the most of one reserve the unit could spend in a single cast,
// read off the skills it actually carries.
//
// Through Takes, so it is the same figure the spend removes — including the
// ceiling a scaling payment stops at. A spender that takes "all of them" for a
// flat bonus has no ceiling of its own, and the status's cap is what bounds it
// there, which is the honest answer: what it could spend really is everything.
//
// A GATED spender is the sharpest case and needs no special arm: it names an
// exact consume_stacks, so Takes returns that count whatever the tank holds, and
// the capacity of a kit carrying two of them is the deeper one's price. That is
// the same figure selfSpendable divides by, read through the same function, which
// is what keeps the flat region of a pile and the worth of one stack in step.
func (p *pricing) capacity(holder *Unit, id string) int {
	most := 0
	for _, held := range holder.Skills {
		declared, err := p.fight.books.Skills.Lookup(held)
		if err != nil || declared.SelfRequires == nil {
			continue
		}
		spends := declared.SelfRequires
		if !spends.Consume || spends.Status != id {
			continue
		}
		kind, err := p.fight.books.Statuses.Lookup(id)
		if err != nil {
			continue
		}
		if takes := spends.Takes(kind.MaxStacks); takes > most {
			most = takes
		}
	}
	return most
}

// pileWorth is what count stacks sitting on top of a pile already `under` high
// are worth, at one stack's value halving with every stack beneath it.
//
// One function because two callers ask and they must agree: charged asks what
// adding to the top is worth, and spentCounter asks what taking the top off
// gives up. A grant and a spend priced by two copies of this series would drift
// the first time either was edited, and the drift would read as a rating that
// liked charging more than cashing in — or the other way, which is worse.
func pileWorth(perStack int64, under, count int) int64 {
	total := int64(0)
	for range under {
		perStack /= 2
		if perStack == 0 {
			return 0
		}
	}
	for range count {
		if perStack == 0 {
			break
		}
		total += perStack
		perStack /= 2
	}
	return total
}

// spendable is what one stack of a counter buys its side, priced as the share of
// a turn the consumer's gain represents.
//
// The gain is the two things a consume can be paid in: the bonus power a
// detonate adds, and the arc power a conduit discharges — each read straight off
// the skill that would spend it, and each already the *net* figure, because a
// conduit damps its own blow to pay for the discharge. Turning that into a figure
// on the same scale as every other term here is the share it is of the whole
// cast, times what that unit's turn is already worth. It is an estimate, and it
// is bounded by construction: a stack can never be worth a whole turn however
// large the payment, which is the property that keeps a rating from preferring to
// charge for ever over ever cashing it in.
//
// ⚠️ **A conduit's gain is read per strike and multiplied by the expected count,
// because that is what a stack actually buys.** One blow spends one charge, so a
// skill that lands twice on average discharges twice — and reading the floor
// would price a repeating conduit at half what it does.
// ⚠️ **Who may spend it is a property of the category, not of the caller**, and
// getting that wrong would make one of the two counters worth nothing. A charge
// is on an ENEMY, so anybody on the charger's side who can cash it counts —
// laying one down for a squadmate to discharge is the whole of what a support
// charger does. A reserve is on its HOLDER and buys the holder's own skills
// through self_requires, so a squadmate's kit is irrelevant to it: reading the
// side there would price fuel by what somebody else could have done with it.
//
// So the search is over the holder alone for a reserve and over the caster's
// side for a charge, and the holder is passed in rather than inferred, because
// for a charge the two are on opposite halves of the board.
func (p *pricing) spendable(actor, holder *Unit, kind status.Kind) int64 {
	if kind.Category == status.Reserve {
		return p.selfSpendable(holder, kind.ID)
	}
	best := int64(0)
	for _, mate := range p.fight.units {
		if mate.Dead || mate.Side != actor.Side {
			continue
		}
		for _, held := range mate.Skills {
			declared, err := p.fight.books.Skills.Lookup(held)
			if err != nil || declared.Power == 0 {
				continue
			}
			if declared.Requires == nil || !declared.Requires.Consume ||
				declared.Requires.Status != kind.ID {
				continue
			}
			gain := int64(declared.Requires.BonusPower)
			if declared.Requires.Arcs() {
				// A conduit adds its arc to a blow it was going to land anyway, so
				// what one stack is worth is the arc and nothing subtracted: the
				// price was paid on the turn that laid the stack down, which is a
				// turn this rating already charged for when it rated that skill.
				gain += int64(declared.Requires.ArcPower)
			}
			if gain <= 0 {
				continue
			}
			if value := p.strike(mate) * gain / (int64(declared.Power) + gain); value > best {
				best = value
			}
		}
	}
	return best
}

// selfSpendable is spendable's reserve half: what one stack of fuel buys the unit
// standing on it, read off that unit's own kit.
//
// ⚠️ **The gain is divided by the stacks a cast actually takes, which the charge
// half does not do**, and the difference is real rather than an inconsistency. A
// detonate spends a pile for a flat bonus and the pile is whatever happened to be
// there, so there is no honest divisor; a reserve spender names its price. A flat
// bonus bought with twenty stacks is worth a twentieth of itself per stack, and
// pricing it otherwise would have a unit hoarding to twenty to buy what one stack
// was already rated as buying.
//
// A per-stack payment needs no divisor at all: it *is* the per-stack figure, and
// that is the shape "spend all of it" is written in.
//
// ⚠️ **A reserve buys health as well as damage, and this used to be blind to
// half of it.** The loop skipped every skill of no power, which is exactly the
// shape a reserve-paid heal has -- so a unit whose only spender was a heal valued
// its whole tank at nought: it never thought banking was worth a turn, cashing in
// cost it nothing, and a dispel aimed at it was free. The guard is now on the
// damage ARM rather than on the search, because it is the damage expression that
// needs a power: a share of a cast is meaningless when the cast is not a blow.
func (p *pricing) selfSpendable(holder *Unit, id string) int64 {
	best := int64(0)
	for _, held := range holder.Skills {
		declared, err := p.fight.books.Skills.Lookup(held)
		if err != nil {
			continue
		}
		spends := declared.SelfRequires
		if spends == nil || !spends.Consume || spends.Status != id {
			continue
		}
		// The health arm, and it needs no share-of-a-cast: a per-stack restore IS
		// the per-stack figure, in the same units every other term here counts in,
		// through the expression restored prices a heal with. Clamped against the
		// holder rather than against a guess, and the clamp is exact here because
		// the skill is aimed at its caster -- the unit whose stack this is.
		if spends.ScalesRestore() {
			healed := p.fight.books.Rules.Restore(
				combat.PickScaling(declared.Scaling.Source,
					holder.Base[declared.Scaling.Stat],
					p.fight.Stats(holder)[declared.Scaling.Stat]),
				spends.StackRestore)
			if value := worthHealing(healed, holder, p.threat(holder)); value > best {
				best = value
			}
			continue
		}
		// The gated arm, and it is the one payment here that is not a share of a
		// cast. Every other arm prices a stack as the SHARE of a cast the condition
		// bought -- strike * gain/(power+gain), where gain is what holding the fuel
		// added to a blow that was going to land anyway. A gate's share is the
		// WHOLE cast: the skill does not exist without the fuel, and the flat power
		// on its own face is its entire figure rather than a bonus on top of one.
		// So the numerator is the cast and the divisor is what the cast costs.
		//
		// ⚠️ **Without this arm the term is nought, and nothing else notices.** A
		// gated spender carries neither StackPower nor BonusPower -- that is the
		// shape, not an omission -- so `gain` below comes out nought, the loop moves
		// on, and the grant that fills the tank is priced at exactly nothing.
		// Measured at 0 on the tree carrying the parser change and not this one.
		// Suggest would never spend a turn charging, so the gate would never open
		// and the spender would ship unreachable: the same dead-branch shape as the
		// heal arm above, one currency along.
		//
		// ⚠️ **It reads p.strike(holder) rather than this skill's own expected
		// damage**, because that is the aim-free approximation every arm in this
		// function already makes -- there is no aim here to read one against, and
		// asking for one would make a stack's worth depend on who happens to be
		// standing where. It therefore OVER-prices the fuel whenever the holder
		// carries a filler heavier than the gated blow, and what checks that is a
		// battle rather than an assertion here: TestTheRatingChargesUntilTheGateOpens
		// holds the loop closing against a filler worth taking, and the auto-battle
		// counts on a shipped kit are what would show it charging too eagerly.
		if spends.GatesCast() {
			// The stacks one cast hands over. A gated spender names its price
			// exactly -- see capacity, which reads the same figure through the same
			// function -- and the floor is there for the "all of them" spelling.
			takes := spends.Takes(spends.MinStacks)
			if takes < 1 {
				takes = 1
			}
			if value := p.strike(holder) / int64(takes); value > best {
				best = value
			}
			continue
		}
		if declared.Power == 0 {
			continue
		}
		gain := int64(spends.StackPower)
		if gain == 0 {
			// The stacks one cast hands over: the count it names, or the floor it
			// needs when it names none and takes everything.
			perCast := spends.ConsumeStacks
			if perCast < spends.MinStacks {
				perCast = spends.MinStacks
			}
			if perCast < 1 {
				perCast = 1
			}
			gain = int64(spends.BonusPower) / int64(perCast)
		}
		if gain <= 0 {
			continue
		}
		if value := p.strike(holder) * gain / (int64(declared.Power) + gain); value > best {
			best = value
		}
	}
	return best
}

// spentCounter is what a skill gives up by cashing its caster's own reserve in,
// and it is the cost half of a spend that the gain half was already priced for.
//
// ⚠️ **Without it the rating spends everything the moment it may.** SelfBonus is
// read inside expected, so the damage a spend buys is counted; the stacks it
// hands over were counted by nothing, so emptying a full reserve for a small
// bonus rated identically to spending one stack for it. That is the shape
// Skill.Cost shipped in — a price filed where the rating never looks — one field
// along.
//
// Priced through the same series the grant is, from the same per-stack figure, so
// a stack cannot be worth one thing going on and another coming off. It is the
// TOP of the pile that goes: a consume takes the speculative stacks, which are
// the cheap ones, so a spend costs less than the charging turns nominally bought.
// That is the honest reading rather than a discount — the alternative prices a
// cash-in at more than the hoard, and a rating handed that never cashes in.
func (p *pricing) spentCounter(actor *Unit, declared skill.Skill) int64 {
	spends := declared.SelfRequires
	if spends == nil || !spends.Consume {
		return 0
	}
	kind, err := p.fight.books.Statuses.Lookup(spends.Status)
	if err != nil {
		return 0
	}
	against := conditionCaster(declared, actor)
	if !declared.SelfAmplified(against) {
		return 0
	}
	taken := spends.Takes(against.Stacks)
	if taken <= 0 {
		return 0
	}
	return pileWorth(p.spendable(actor, actor, kind), against.Stacks-taken, taken)
}

// unfuelled is the reserve a dispel takes off an enemy, which is the casts it
// will now never make.
//
// The third of the strip terms, beside unguarded and unbarriered, and it exists
// for the reason they do: Without removes the stacks, and every other reading in
// dispelled is a stat or a tick, so a counter that changes neither would come off
// an enemy for a price of nothing. A dispel is the only answer a squad has to a
// unit hoarding fuel, and a rating that could not see the answer would never
// play it.
//
// ⚠️ Read from the HOLDER's side of the arithmetic — spendable asks what the
// holder's own kit could do with it — because a reserve is worth what its owner
// can spend, and the unit doing the stripping has no skill that names it.
func (p *pricing) unfuelled(target *Unit, after status.Set) int64 {
	total := int64(0)
	for _, id := range target.Statuses.Active() {
		kind, err := p.fight.books.Statuses.Lookup(id)
		if err != nil || kind.Category != status.Reserve {
			continue
		}
		taken := target.Statuses.Stacks(id) - after.Stacks(id)
		if taken <= 0 {
			continue
		}
		total += pileWorth(p.spendable(target, target, kind),
			target.Statuses.Stacks(id)-taken, taken)
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
//
// The division is by a count in parts per thousand, so it is written out rather
// than left as a plain divide: see worstStrikes for why the count is not a whole
// number.
func (p *pricing) strikeThreat(unit *Unit) int64 {
	strikes := p.worstStrikes(unit)
	if strikes < scale.Base {
		strikes = scale.Base
	}
	return p.threat(unit) * scale.Base / int64(strikes)
}

// worstStrikes is how many strikes the heaviest attack aimed at this unit would
// come in, in parts per thousand, so a charge is priced against one of them
// rather than all.
//
// ⚠️ **In per mille, and through ExpectedStrikes, because a repeating attacker
// makes a charge worth LESS and the floor could not say so.** A charge cancels
// one strike whole, so what it is worth is the share of a turn one strike is —
// and against a skill that lands about five times that share is a fifth, not a
// half. Reading StrikeCount here priced a guard against the strikes an attacker
// is guaranteed and none of the tail, which is the same floor Rules.Expected was
// reading and the other half of one fact.
//
// A whole number cannot carry it: rounding a count of 3,120 down to 3 inflates
// every charge priced against it, and this figure is a DIVISOR, so the rounding
// error runs the wrong way — over-pricing a guard costs a kill where under-pricing
// costs a marginal cast.
func (p *pricing) worstStrikes(unit *Unit) int {
	strikes := scale.Base
	for _, other := range p.fight.units {
		if other.Dead || other.Side == unit.Side {
			continue
		}
		for _, id := range other.Skills {
			declared, err := p.fight.books.Skills.Lookup(id)
			if err != nil || declared.Power == 0 || !aimedAtAnEnemy(declared) {
				continue
			}
			if count := declared.ExpectedStrikes(); count > strikes {
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
			swingOf(declared, actor)); value > best {
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
