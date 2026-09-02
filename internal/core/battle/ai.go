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

// DeclinedReason is the note recorded when a unit HAD something it could use and
// the rating refused it, which is the one pass in this engine that is a decision.
//
// ⚠️ It is a second constant rather than a reuse of the one above, because the
// two are different facts about the board and a log that spelled them the same
// would be telling a reader a unit was helpless when it was choosing. `TestNothing
// WaitsOnPurpose` counts passes by reason and was written to name this the day it
// appeared; a rule that arrived without its own note would have slipped past it.
const DeclinedReason = "nothing worth doing"

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
	fallback, fallbackCooldown, hasFallback := Choice{}, 0, false
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
		// Worth nothing to anybody it could reach: the fallback. It is taken only
		// if nothing at all was worth doing, and which one is taken goes through
		// the same tie-break every rated option goes through — the cheapest to
		// have spent, with kit order deciding a tie, exactly as `take` reads it.
		//
		// ⚠️ **This arm kept "the first in kit order" long after `take` stopped
		// doing so, and it is the same mistake one branch over.** The tie-break
		// exists because two options worth the same are not the same to spend:
		// `TestAScarceSkillIsNotSpentOnWhatACommonOneBuys` was written for a
		// `clout` burnt on ten points of health while the `jab` beside it would
		// have done. Options worth *nothing* are the sharpest case of that, not an
		// exception to it — a `rapid_spin` at three turns of cooldown, cast on a
		// board with nothing to strip, is a cleanse gone for three turns bought
		// with nothing, while a cooldownless option in the same kit costs the turn
		// and no more. Measured before it was written: with two skills both worth
		// nought, kit `[spin, dust]` cast `spin` and kit `[dust, spin]` cast
		// `dust` — kit order was the whole of the decision.
		//
		// ⚠️ It is NOT a pass. Whether a unit should decline a turn rather than
		// spend a cooldown for nothing is a separate question and a larger one,
		// because every gap in what this file prices is an *under*-price: worth
		// nothing to the rating is not yet worth nothing. See TODO.md.
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
		if !rated && (!hasFallback || declared.Cooldown < fallbackCooldown) {
			fallback, fallbackCooldown, hasFallback =
				Choice{Skill: option.Skill, Aim: option.Aims[0]}, declared.Cooldown, true
		}
	}
	if found {
		return best, true
	}
	// Nothing was worth anything to anybody it could reach. Casting the cheapest
	// of those buys nought either way, so the only thing left to decide is what it
	// COSTS — and a skill on a cooldown is gone for turns afterwards, bought with
	// nothing.
	//
	// ⚠️ **Returning no choice is a pass, and the mechanism already existed**:
	// RunToEndWith turns `(Choice{}, false)` into Battle.Pass, on the same terms as
	// a unit with nothing it can use. This is a rating rule, not a new verb.
	//
	// ⚠️ **It waited for the pricing, and that was the right order.** TODO.md held
	// this behind six gaps — a repeating skill read at its floor, a drain worth
	// nothing, a pool nothing discounted a blow into, a discharge priced nowhere,
	// a taunt and a heal cut with no arm at all — because every one of them was an
	// *under*-price, and "worth nought to the rating" was not yet "worth nought".
	// With all six closed it is much nearer, which is what makes declining a turn
	// safe rather than merely cheap.
	//
	// ⚠️ A cooldownless option is still cast. It costs the turn and no more, and
	// what this file cannot see is always something rather than nothing — the
	// shipped `taunt` was worth nought here until an hour ago. The shape of the
	// error is chosen: refusing a free cast could throw away a real effect, while
	// refusing a priced one only declines a turn nobody had a use for.
	if hasFallback && fallbackCooldown == 0 {
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
	// The two halves are taken apart rather than multiplied straight through,
	// because a wall of charges needs both: what ONE strike comes to, and how many
	// strikes connect. Rules.Expected is those two multiplied, so this is the same
	// reading and not a second one.
	perStrike := b.books.Rules.ExpectedStrike(hit)
	connecting := int64(hit.ExpectedStrikes()) * int64(b.books.Rules.Chance(hit)) /
		combat.PermilleBase
	landed := b.pastAWall(target, declared, perStrike, connecting,
		combat.Scaled(perStrike, int(connecting)))
	// Damage past a target's remaining health is wasted, so a finishing blow is not
	// rated above one that would kill twice over.
	if landed > target.HP {
		landed = target.HP
	}
	return landed
}

// pastAWall is what is left of a blow after a wall of block charges on the target
// has cancelled what it can, and it is the half of a guard that #229 measured,
// wrote, and deliberately left out.
//
// A charge cancels a STRIKE whole, which is the whole of `warden`'s trade: a wall
// answers one heavy blow and multi-strike answers a wall.
//
// ⚠️ **A wall deeper than the blow needs no clamp**, and the first cut of this
// carried one anyway — the charge count trimmed to the strikes that would have
// connected, on the reasoning that Rules.Roll spends nothing on a miss. True, and
// arithmetically dead: subtracting more than the blow drives it under nought and
// the guard below already answers nought. The miss is priced where it belongs, in
// `connecting`, which is the strike count weighted by the chance to land. A clause
// no mutation can break is a clause that is not doing anything.
//
// ⚠️ **It is charged on every cast, and a charge is only paid for once.** A wall
// of three cancels three strikes over the whole battle, so a rating one turn deep
// that discounts by the whole wall every turn is pricing the same loss again and
// again. That over-count is real and it is accepted, because the alternative is a
// dial and there is no setting on it: measured, every discount small enough to
// leave the shipped balance claims standing reads INSIDE the measurement's own
// band against the frozen ruler, and every discount large enough to clear that
// band moves them. See TODO.md for the sweep.
func (b *Battle) pastAWall(target *Unit, declared skill.Skill,
	perStrike, connecting, damage int64) int64 {
	if declared.Unblockable || damage <= 0 {
		return b.pastAPool(target, declared, damage)
	}
	if charges := int64(target.Statuses.Stacks(blockStatus)); charges > 0 {
		// A wall this deep is unreachable and the product is not: perStrike is
		// what ExpectedStrike answered, which saturates, and a saturated figure
		// times a stack count is where that turns back into a wrap. What comes
		// out here is subtracted from a non-negative damage, so a saturated wall
		// drives the blow under nought and the guard below answers it.
		damage -= combat.Repeated(perStrike, int(charges))
		if damage <= 0 {
			return 0
		}
	}
	return b.pastAPool(target, declared, damage)
}

// pastAPool is what is left of a blow after an absorbing pool on the target has
// taken its share, and it is half of a gap this rating had: `shielded` and
// `guarded` paid to PUT a guard up, and nothing discounted a blow INTO one.
//
// Measured before it was written: offered two identical enemies, one of them the
// softer target and carrying a pool of a hundred thousand, the rating aimed at
// the softer one every time — and could not land a point of it.
//
// A pool eats damage and is indifferent to how it arrives, so what it takes over
// a volley is simply the smaller of the pool and the damage. That is the whole of
// what combat.Absorb comes to across a volley rather than a second copy of it:
// the function spreads the same total over the strikes so each one's line can
// report its share, and nothing here needs the shares.
//
// ⚠️ An unblockable skill is offered nothing, exactly as resolveAgainst offers it
// nothing — and *offered*, not emptied: what the target is carrying is untouched.
//
// ⚠️ It is deliberately not a per-attacker reservation. Two allies swinging at one
// barrier both read it whole, so the second is under-priced, which is the
// direction this file errs in everywhere.
//
// ⚠️⚠️ **The other half — a wall of BLOCK CHARGES — is deliberately not here, and
// it is a measurement rather than an oversight.** A charge cancels a strike
// whole, so the arithmetic is easy and it was written; what it does is the
// problem. On a board thick with `withdraw` it is a large improvement — 990 per
// mille against the frozen ruler with every battle decided, where the rating
// without it cannot finish 212 of 800 — but on the ordinary squad boards the
// design record is quoted on it reads **identically** against that ruler (668
// either way) while moving squad rates by up to a hundred and eighty per mille,
// which broke three balance claims at once and FLIPPED one of them.
// Redistributing which kit wins without playing measurably better is a balance
// change wearing a rating fix, and it needs those claims re-derived rather than
// re-baselined. The pool half has no such history: nothing shipped fields a deep
// pool, so there was nothing to disturb, and the three claims do not move.
// See TODO.md.
func (b *Battle) pastAPool(target *Unit, declared skill.Skill, damage int64) int64 {
	if declared.Unblockable || damage <= 0 {
		return damage
	}
	if pool := target.Statuses.PoolIn(absorbCategory); pool > 0 {
		if pool >= damage {
			return 0
		}
		damage -= pool
	}
	return damage
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
		power = int(combat.Scaled(int64(power), b.books.Patterns.SplashPower))
	}
	targetStats := b.Stats(target)
	multiplier := b.books.Chart.MultiplierAgainst(declared.Element, target.Affinity)
	multiplier = actor.Statuses.Modifiers().Affinity(
		multiplier, b.books.Chart.Multipliers().Neutral, b.books.Bounds)
	return combat.Hit{
		Scaling: combat.PickScaling(declared.Scaling.Source,
			actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat]),
		Multiplier: power,
		Strikes:    declared.StrikeCount(),
		// ⚠️ **Carried, because a Hit that leaves them out is a Hit that repeats
		// nothing.** Rules.Expected reads the count through Hit.ExpectedStrikes,
		// and a zero Repeat makes that answer the floor — so the rating priced a
		// repeating skill at the strikes it is guaranteed and none of the tail,
		// however long the tail. Fixing Expected alone would have moved nothing:
		// the two halves are one change.
		Repeat:     declared.Repeat,
		MaxStrikes: declared.MaxStrikes,
		Affinity:   multiplier,
		Defense:    targetStats[progression.Defense],
		Pierce:     declared.Pierce,
		// The actor's own conversion, read here for the reason this whole
		// function exists: a rating that built its hit without it would price
		// every blow a converting unit throws as smaller than the one it lands,
		// and against a wall — the exact board the trait is for — it would prefer
		// anything else.
		Convert:       b.converts(actor),
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

// passReason tells the two kinds of pass apart by asking what was on offer, which
// is the one place both facts are in hand: the Chooser has already answered, and
// the prompt still says what it was answering about.
//
// Read off Option.Available rather than off the option count, because a prompt
// lists every skill a unit knows and marks the ones it cannot use — a unit with
// four skills all cooling down is helpless and its prompt is not empty.
func passReason(prompt *Prompt) string {
	for _, option := range prompt.Options {
		if option.Available() {
			return DeclinedReason
		}
	}
	return NoActionReason
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
			// offer. Either way the turn is spent and its cooldowns come down —
			// but the two are written down apart, because one is a unit with no
			// move and the other is a unit that had one and would not take it.
			if err := b.Pass(passReason(prompt)); err != nil {
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
