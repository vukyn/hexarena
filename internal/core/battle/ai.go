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

// Choice is an action picked for a unit.
type Choice struct {
	Skill string
	Aim   hex.Offset
}

// Suggest picks an action for the acting unit.
//
// It is deliberately shallow: it takes the option with the highest expected
// damage across everything the shape would catch, and falls back to the first
// usable non-damaging skill when nothing can be hurt. That is enough to run a
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
	total := int64(0)
	for position, cell := range shape.Targets(aim) {
		target := b.occupant(cell)
		if target == nil || target.Side == actor.Side {
			continue
		}
		power := declared.PowerAgainst(target.Statuses.Stacks(requiredStatus(declared)))
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

func requiredStatus(declared skill.Skill) string {
	if declared.Requires == nil {
		return ""
	}
	return declared.Requires.Status
}

// RunToEnd plays the battle out with Suggest choosing for both sides, stopping at
// the turn limit so a stalemate cannot hang a caller. It reports how many turns
// were taken.
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
