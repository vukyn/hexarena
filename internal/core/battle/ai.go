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
// It is deliberately shallow: it takes the option with the highest expected
// damage across everything the shape would catch, and falls back to the first
// usable non-damaging skill when nothing can be hurt. A summoning skill is the
// one exception to "damage this turn" — see summonWorth — because it is the only
// thing in the book that buys turns rather than spends one. That is enough to run a
// battle to its end without a player, which is what the tests and the enemy side
// need, and it is fully deterministic so a replay stays a replay.
//
// It reads no randomness and mutates nothing, so a client may call it to offer a
// hint without disturbing the battle's own sequence.
func (b *Battle) Suggest(prompt *Prompt) (Choice, bool) {
	if prompt == nil || prompt.Skipped {
		return Choice{}, false
	}
	actor, known := b.byID[prompt.Unit]
	if !known || actor.Dead {
		return Choice{}, false
	}
	best, bestValue, found := Choice{}, int64(-1), false
	fallback, hasFallback := Choice{}, false

	for _, option := range prompt.Options {
		if !option.Available() {
			continue
		}
		declared, err := b.books.Skills.Lookup(option.Skill)
		if err != nil {
			continue
		}
		// Before the power check, because a summoning skill has no power of its
		// own and would otherwise stay the fallback it used to be: every one of
		// them was reached only when nothing could be hurt, so the shipped
		// summoner never called anybody up while it had a kunai in reach.
		if declared.Summons.Summons() {
			// A cast worth nothing falls through to the fallback below rather
			// than being rated at nought, which is what it was before this
			// branch existed. The two differ on a board with no room: a rating
			// of nought is still a rating, so it would beat "no damaging option
			// at all" and take the turn ahead of a shield or a cleanse that
			// would have done something.
			if value := b.summonWorth(actor, declared); value > 0 {
				if value > bestValue {
					best, bestValue, found =
						Choice{Skill: option.Skill, Aim: option.Aims[0]}, value, true
				}
				continue
			}
		}
		if declared.Power == 0 {
			if !hasFallback {
				fallback, hasFallback = Choice{Skill: option.Skill, Aim: option.Aims[0]}, true
			}
			continue
		}
		if declared.Target != skill.Enemy {
			continue
		}
		for _, aim := range option.Aims {
			value := b.expected(actor, declared, aim)
			if value > bestValue {
				best, bestValue, found = Choice{Skill: option.Skill, Aim: aim}, value, true
			}
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
	// Read once, outside the loop, because that is when it is read for real. A
	// rating that asked per cell would still get the same answer today, and
	// would stop doing so the moment a condition read anything the loop changes.
	spent := declared.SelfBonus(conditionCaster(declared, actor))
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
		power := declared.PowerAgainst(conditionTarget(declared, target)) + spent
		if position > 0 {
			power = power * b.books.Patterns.SplashPower / 1000
		}
		targetStats := b.Stats(target)
		multiplier := b.books.Chart.MultiplierAgainst(declared.Element, target.Affinity)
		multiplier = actor.Statuses.Modifiers().Affinity(
			multiplier, b.books.Chart.Multipliers().Neutral, b.books.Bounds)
		hit := combat.Hit{
			Scaling: combat.PickScaling(declared.Scaling.Source,
				actor.Base[declared.Scaling.Stat], actorStats[declared.Scaling.Stat]),
			Multiplier:    power,
			Strikes:       declared.StrikeCount(),
			Affinity:      multiplier,
			Defense:       targetStats[progression.Defense],
			Pierce:        declared.Pierce,
			SkillAccuracy: declared.Accuracy,
			AccuracyStat:  actorStats[progression.Accuracy],
			DodgeStat:     targetStats[progression.Dodge],
		}
		landed := b.books.Rules.Total(hit) * int64(b.books.Rules.Chance(hit)) / combat.PermilleBase
		// Damage past a target's remaining health is wasted, so a finishing blow
		// is not rated above one that would kill twice over.
		if landed > target.HP {
			landed = target.HP
		}
		total += landed
	}
	return total
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
func (b *Battle) bestStrike(unit *Unit) int64 {
	best := int64(0)
	for _, id := range unit.Skills {
		declared, err := b.books.Skills.Lookup(id)
		if err != nil || declared.Power == 0 || declared.Target != skill.Enemy {
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

// RunToEnd plays the battle out with Suggest choosing for both sides, stopping at
// the turn limit so a runaway cannot hang a caller. It reports how many turns
// were taken, and the caller reads Finished to know whether that was the end or
// the limit.
//
// The limit is a backstop and nothing else. It used to be the only thing that
// noticed a battle nobody could act in, which made it stand in for an outcome
// the engine could not express; a deadlock now ends itself as a Stalemate, so
// reaching the limit means something genuinely endless is happening — two units
// buffing themselves at each other for ever — rather than a draw waiting to be
// recognised.
func (b *Battle) RunToEnd(maxTurns int) (int, error) {
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
		choice, ok := b.Suggest(prompt)
		if !ok {
			// Nothing is usable, which is a legal state: every skill is either
			// cooling down or out of reach.
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
