package battle

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// Option is one action a unit may take this turn, with the cells it may aim at.
type Option struct {
	Skill string
	// Aims are the primary cells the skill may target. Empty means the skill is
	// known but cannot be used right now, and Reason says why.
	Aims   []hex.Offset
	Reason string
}

// Available reports whether the option can actually be chosen.
func (o Option) Available() bool { return len(o.Aims) > 0 }

// Prompt is whose turn it is and what they may do.
type Prompt struct {
	Unit string
	Turn int
	At   int64
	// Skipped means the unit lost its action, to control or to a timed effect
	// that killed it, and the caller should advance again without acting.
	Skipped bool
	Reason  string
	Options []Option
}

// Advance begins the next turn. It resolves everything that happens before a
// unit chooses: timed effects tick, a tempo change reorders the queue, cooldowns
// come down, and control is checked.
//
// The order is fixed and the control check comes before durations are spent.
// A stun that lasts one turn is applied on one turn and has to cost the next one;
// spending its duration first would expire it in the very turn it was meant to
// prevent, which would make one-turn control do nothing at all.
func (b *Battle) Advance() (*Prompt, error) {
	if b.finished {
		return nil, fmt.Errorf("the battle is over")
	}
	if b.awaiting {
		return nil, fmt.Errorf("unit %q has not acted yet", b.acting.ID)
	}
	turn, ok := b.queue.Next()
	if !ok {
		b.checkEnd()
		return nil, fmt.Errorf("no unit is left to act")
	}
	unit, known := b.byID[turn.ID]
	if !known || unit.Dead {
		return nil, fmt.Errorf("the queue offered %q, which is not fighting", turn.ID)
	}
	b.emit(Event{Kind: TurnBegan, At: turn.At, Turn: turn.Number, Actor: unit.ID, Amount: unit.HP})

	controlled := unit.Statuses.Has(stunStatus)
	b.tickStatuses(unit, turn)
	b.retuneAll(turn)
	if unit.Dead {
		b.emit(Event{Kind: TurnSkipped, At: turn.At, Turn: turn.Number, Actor: unit.ID, Note: "died"})
		return &Prompt{Unit: unit.ID, Turn: turn.Number, At: turn.At, Skipped: true, Reason: "died"}, nil
	}
	if controlled {
		b.spendCooldowns(unit)
		b.emit(Event{Kind: TurnSkipped, At: turn.At, Turn: turn.Number, Actor: unit.ID, Note: stunStatus})
		return &Prompt{Unit: unit.ID, Turn: turn.Number, At: turn.At, Skipped: true, Reason: stunStatus}, nil
	}

	b.acting, b.awaiting = unit, true
	b.prompt = &Prompt{
		Unit: unit.ID, Turn: turn.Number, At: turn.At, Options: b.options(unit),
	}
	return b.prompt, nil
}

// tickStatuses resolves the start-of-turn effects. The per-status damage is read
// from a snapshot taken before the tick, because a log that says only "you took
// 513" is much less use than one that names the poison.
func (b *Battle) tickStatuses(unit *Unit, turn atb.Turn) {
	before := unit.Statuses.Snapshot()
	damage, healing, expired := unit.Statuses.Tick()
	for _, entry := range before {
		if entry.TickAmount <= 0 {
			continue
		}
		// A regeneration reports itself as a heal rather than as a tick, so a
		// reader is never asked to work out from the status name whether a
		// number was taken or given.
		kind := StatusTicked
		if entry.Category == status.Regen {
			kind = Healed
		}
		b.emit(Event{
			Kind: kind, At: turn.At, Turn: turn.Number, Actor: unit.ID,
			Status: entry.ID, Amount: entry.TickAmount, Stacks: entry.Stacks,
		})
	}
	for _, id := range expired {
		b.emit(Event{
			Kind: StatusExpired, At: turn.At, Turn: turn.Number, Actor: unit.ID, Status: id,
		})
	}
	// Healing first: a regeneration that would carry a unit past a poison tick
	// should do so, rather than the order of two totals deciding who lives.
	if healing > 0 {
		b.heal(unit, healing, turn)
	}
	if damage > 0 {
		b.wound(unit, damage, turn)
	}
}

// heal gives health back, and refuses two things absolutely.
//
// A dead unit is not healed: wound calls kill the moment health reaches zero,
// and undoing that would leave a battle unable to end. And health never passes
// the maximum the unit was enlisted with, so a regeneration cannot become an
// uncapped shield. Both are silent — a heal that does nothing is not an error,
// it is a full-health unit standing in a healing wind.
func (b *Battle) heal(unit *Unit, amount int64, turn atb.Turn) {
	if amount <= 0 || unit.Dead || unit.HP >= unit.MaxHP() {
		return
	}
	if room := unit.MaxHP() - unit.HP; amount > room {
		amount = room
	}
	unit.HP += amount
	b.emit(Event{
		Kind: Healed, At: turn.At, Turn: turn.Number, Actor: unit.ID,
		Amount: amount, Remaining: unit.HP,
	})
}

func (b *Battle) wound(unit *Unit, damage int64, turn atb.Turn) {
	unit.HP -= damage
	if unit.HP <= 0 {
		b.kill(unit)
	}
	_ = turn
}

// retuneAll keeps the queue in step with every unit's current speed.
//
// It runs after anything that could change a stat rather than only at the start
// of the changed unit's own turn. A haste applied on a unit's turn has to shorten
// the wait it is about to serve; noticing it only when that wait had already
// elapsed would make the buff worthless for the very turn it was cast on, and a
// slow applied to an enemy would not delay them at all.
//
// Every unit is checked because a skill can change the tempo of anyone: the
// caster, its target, or a whole column caught by an area shape. There are at
// most ten units, so the sweep costs nothing worth avoiding.
func (b *Battle) retuneAll(turn atb.Turn) {
	for _, unit := range b.units {
		if unit.Dead {
			continue
		}
		current := b.Stats(unit)[progression.Speed]
		scheduled := b.queue.Speed(unit.ID)
		if current == scheduled || scheduled == 0 {
			continue
		}
		if err := b.queue.Reschedule(unit.ID, current); err != nil {
			continue
		}
		b.emit(Event{
			Kind: SpeedChanged, At: turn.At, Turn: turn.Number, Actor: unit.ID,
			Before: scheduled, Amount: current,
		})
	}
}

// spendCooldowns brings a unit's cooldowns down by the turn it just served.
//
// It runs when a turn ends rather than when one begins, so that the options a
// unit is offered and the action it is allowed to take always read the same
// numbers. Spending at the start would let a skill be used one turn early: the
// turn it was cast on would pay for part of its own cooldown.
func (b *Battle) spendCooldowns(unit *Unit) {
	for i := range unit.Cooldowns {
		if unit.Cooldowns[i] > 0 {
			unit.Cooldowns[i]--
		}
	}
}

// Pass ends the acting unit's turn without an action, which is what a unit with
// nothing usable does. The turn still counts against its cooldowns.
func (b *Battle) Pass(reason string) error {
	if !b.awaiting {
		return fmt.Errorf("no unit is waiting to act")
	}
	unit := b.acting
	b.awaiting, b.prompt = false, nil
	b.spendCooldowns(unit)
	if reason == "" {
		reason = "passed"
	}
	b.emit(Event{
		Kind: TurnSkipped, Turn: b.queue.Turns(unit.ID), Actor: unit.ID, Note: reason,
	})
	return nil
}

func (b *Battle) options(unit *Unit) []Option {
	out := make([]Option, 0, len(unit.Skills))
	for index, id := range unit.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			out = append(out, Option{Skill: id, Reason: "unknown skill"})
			continue
		}
		if unit.Cooldowns[index] > 0 {
			out = append(out, Option{
				Skill:  id,
				Reason: fmt.Sprintf("%d turns of cooldown left", unit.Cooldowns[index]),
			})
			continue
		}
		aims := b.aims(unit, known)
		option := Option{Skill: id, Aims: aims}
		if len(aims) == 0 {
			option.Reason = "nothing in reach"
		}
		out = append(out, option)
	}
	return out
}

// aims lists the cells a skill may be pointed at: within range, on a side the
// skill targets, and holding someone.
//
// The walk is over the whole board in hex.Cells' column-major order, filtered by
// skill.Side.Reaches, because a skill aimed at both halves has no single side to
// ask for. For a skill aimed at one half that is the same list in the same order
// as the side's own cells — hex.SideCells is Cells filtered — and the order is
// load-bearing: battle.Suggest keeps the first of two equally good aims, so
// reordering this would move the choices in a golden log without changing a rule.
func (b *Battle) aims(unit *Unit, known skill.Skill) []hex.Offset {
	if known.Target == skill.Self {
		return []hex.Offset{unit.Cell}
	}
	out := make([]hex.Offset, 0, hex.Cols*hex.Rows)
	for _, cell := range hex.Cells() {
		if !known.Target.Reaches(unit.Side, cell.Side()) {
			continue
		}
		if unit.Cell.DistanceTo(cell) > known.Range {
			continue
		}
		if b.occupant(cell) == nil {
			continue
		}
		out = append(out, cell)
	}
	return out
}

// covers is the cells a skill catches from an aim.
//
// One declaration for the two places that walk a shape — resolving an action and
// rating one — because they have to agree: an aim rated over cells the resolution
// would not touch is a hint that lies, and the mistake would only show on the
// skills aimed at both sides, which are the ones nothing else covers.
func covers(shape pattern.Pattern, known skill.Skill, aim hex.Offset) []hex.Offset {
	if known.Target.CrossesSides() {
		return shape.TargetsAcross(aim)
	}
	return shape.Targets(aim)
}

// Act resolves the acting unit's chosen skill against a chosen cell.
func (b *Battle) Act(skillID string, aim hex.Offset) error {
	if b.finished {
		return fmt.Errorf("the battle is over")
	}
	if !b.awaiting {
		return fmt.Errorf("no unit is waiting to act")
	}
	unit := b.acting
	index := -1
	for i, id := range unit.Skills {
		if id == skillID {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("unit %q does not know %q", unit.ID, skillID)
	}
	if unit.Cooldowns[index] > 0 {
		return fmt.Errorf("%q has %d turns of cooldown left", skillID, unit.Cooldowns[index])
	}
	known, err := b.books.Skills.Lookup(skillID)
	if err != nil {
		return err
	}
	legal := false
	for _, cell := range b.aims(unit, known) {
		if cell == aim {
			legal = true
			break
		}
	}
	if !legal {
		return fmt.Errorf("%q cannot be aimed at %s from %s", skillID, aim, unit.Cell)
	}

	// The turn being spent comes off every cooldown first, then the skill just
	// used starts its own, so a skill never pays part of its cooldown with the
	// turn it was cast on.
	b.spendCooldowns(unit)
	unit.Cooldowns[index] = known.Cooldown
	b.awaiting, b.prompt = false, nil
	turn := atb.Turn{At: b.queue.Now(), Number: b.queue.Turns(unit.ID)}
	b.emit(Event{
		Kind: SkillUsed, At: turn.At, Turn: turn.Number, Actor: unit.ID,
		Skill: known.ID, Cell: aim, Power: known.Power, Chance: known.Accuracy,
	})

	b.applyToSelf(unit, known, turn)
	if known.Target == skill.Self {
		b.strip(unit, unit, known, turn)
		b.retuneAll(turn)
		return nil
	}
	shape, err := b.books.Patterns.Lookup(known.Pattern)
	if err != nil {
		return err
	}
	for position, cell := range covers(shape, known, aim) {
		target := b.occupant(cell)
		if target == nil {
			continue
		}
		b.resolveAgainst(unit, target, known, shape.Name, position, turn)
		if b.finished {
			return nil
		}
	}
	b.retuneAll(turn)
	return nil
}

func (b *Battle) applyToSelf(unit *Unit, known skill.Skill, turn atb.Turn) {
	for _, application := range known.SelfApplies {
		b.inflict(unit, unit, known, application, turn)
	}
}

func (b *Battle) strip(actor, target *Unit, known skill.Skill, turn atb.Turn) {
	if known.Strips == nil {
		return
	}
	removed := target.Statuses.Cleanse(known.Strips.Categories, known.Strips.Stacks)
	if removed == 0 {
		return
	}
	b.emit(Event{
		Kind: StatusStripped, At: turn.At, Turn: turn.Number,
		Actor: actor.ID, Target: target.ID, Skill: known.ID, Stacks: removed,
	})
}

// resolveAgainst is one target's share of a skill: the cleanse, the amplifier,
// the strikes, then the statuses.
//
// A splash cell takes a reduced share of the power while the primary takes all of
// it, which is how an area skill trades focus for spread without being strictly
// better than a single-target one.
func (b *Battle) resolveAgainst(actor, target *Unit, known skill.Skill, shape string, position int, turn atb.Turn) {
	b.strip(actor, target, known, turn)

	power := known.Power
	if known.Requires != nil {
		stacks := target.Statuses.Stacks(known.Requires.Status)
		if known.Amplified(stacks) {
			power = known.PowerAgainst(stacks)
			b.emit(Event{
				Kind: Amplified, At: turn.At, Turn: turn.Number, Actor: actor.ID,
				Target: target.ID, Skill: known.ID, Status: known.Requires.Status,
				Stacks: stacks, Power: power,
			})
			if known.Requires.Consume {
				consumed, forgone := target.Statuses.Consume(known.Requires.Status)
				b.emit(Event{
					Kind: StatusConsumed, At: turn.At, Turn: turn.Number, Actor: actor.ID,
					Target: target.ID, Skill: known.ID, Status: known.Requires.Status,
					Stacks: consumed, Amount: forgone,
				})
			}
		}
	}
	if position > 0 {
		power = power * b.books.Patterns.SplashPower / scale.Base
	}

	actorStats := b.Stats(actor)
	targetStats := b.Stats(target)
	multiplier := b.books.Chart.MultiplierAgainst(known.Element, target.Affinity)
	multiplier = actor.Statuses.Modifiers().Affinity(multiplier, b.books.Chart.Multipliers().Neutral, b.books.Bounds)

	hit := combat.Hit{
		Scaling: combat.PickScaling(known.Scaling.Source,
			actor.Base[known.Scaling.Stat], actorStats[known.Scaling.Stat]),
		Multiplier:    power,
		Strikes:       known.StrikeCount(),
		Affinity:      multiplier,
		Defense:       targetStats[progression.Defense],
		Pierce:        known.Pierce,
		SkillAccuracy: known.Accuracy,
		AccuracyStat:  actorStats[progression.Accuracy],
		DodgeStat:     targetStats[progression.Dodge],
	}

	connected := false
	dealt := int64(0)
	if power > 0 {
		charges := target.Statuses.Stacks(blockStatus)
		attempts, left := b.books.Rules.Roll(hit, charges, b.source)
		if spent := charges - left; spent > 0 {
			target.Statuses.Remove(blockStatus, spent)
		}
		chance := b.books.Rules.Chance(hit)
		// Damage is taken off as each strike resolves rather than totalled and
		// applied at the end, so every event carries the health that was
		// actually left at that moment. A log where the second strike of a pair
		// reports more health than the first is worse than no log.
		for strike, attempt := range attempts {
			event := Event{
				At: turn.At, Turn: turn.Number, Actor: actor.ID, Target: target.ID,
				Skill: known.ID, Strike: strike + 1, Chance: chance,
				Multiplier: multiplier, Power: power, Pierce: known.Pierce,
				Remaining: target.HP,
			}
			switch attempt.Outcome {
			case combat.Missed:
				event.Kind = Missed
			case combat.Blocked:
				event.Kind = Blocked
				event.Remaining = int64(target.Statuses.Stacks(blockStatus))
			default:
				event.Kind = Damaged
				event.Amount = attempt.Damage
				dealt += attempt.Damage
				target.HP -= attempt.Damage
				if target.HP < 0 {
					target.HP = 0
				}
				event.Remaining = target.HP
				connected = true
			}
			b.emit(event)
			// A target that has fallen takes no further strikes; the rest of a
			// multi-strike skill is simply wasted on it.
			if target.HP <= 0 {
				break
			}
		}
	} else if b.source.Chance(b.books.Rules.Chance(hit)) {
		connected = true
	} else {
		b.emit(Event{
			Kind: Missed, At: turn.At, Turn: turn.Number, Actor: actor.ID,
			Target: target.ID, Skill: known.ID, Strike: 1,
			Chance: b.books.Rules.Chance(hit),
		})
	}

	if connected {
		for _, application := range known.Applies {
			b.inflict(actor, target, known, application, turn)
		}
	}
	// Restoring reads the caster's scaling stat and skips the defence curve
	// entirely: see combat.Rules.Restore. A splashed target gets the reduced
	// share, the same as damage, because a shape's edge is worth less wherever
	// it lands.
	if known.Restores > 0 {
		restore := known.Restores
		if position > 0 {
			restore = restore * b.books.Patterns.SplashPower / scale.Base
		}
		b.heal(target, b.books.Rules.Restore(
			combat.PickScaling(known.Scaling.Source,
				actor.Base[known.Scaling.Stat], actorStats[known.Scaling.Stat]),
			restore), turn)
	}
	// A drain takes its share of what was *dealt*, so a strike that missed or
	// was blocked returns nothing, and one that overkilled returns only the
	// damage that landed.
	if known.Drains > 0 && dealt > 0 {
		b.heal(actor, dealt*int64(known.Drains)/int64(scale.Base), turn)
	}
	if target.HP <= 0 {
		b.kill(target)
	}
	_ = shape
}

// inflict rolls one status application. The chance is the skill's own and nothing
// touches it: whether the hit landed and whether the status takes hold are two
// separate questions, and keeping them separate is what makes both legible.
func (b *Battle) inflict(actor, target *Unit, known skill.Skill, application skill.Application, turn atb.Turn) {
	kind, err := b.books.Statuses.Lookup(application.Status)
	if err != nil {
		return
	}
	if !b.source.Chance(application.Chance) {
		b.emit(Event{
			Kind: StatusResisted, At: turn.At, Turn: turn.Number, Actor: actor.ID,
			Target: target.ID, Skill: known.ID, Status: kind.ID, Chance: application.Chance,
		})
		return
	}
	tick := int64(0)
	if kind.Category == status.Dot {
		// Full defence, deliberately, even when the skill applying the status
		// pierces. A tick is computed once here and frozen on the stack for the
		// rest of its life, so piercing it would be worth as many pierced hits
		// as the stack has turns left — a far larger effect than the per-strike
		// ratio the author wrote. That is why Pierced is applied by Strike and
		// not folded into the defence a caller passes.
		tick = b.books.Rules.Damage(
			b.Stats(actor)[known.Scaling.Stat],
			b.Stats(target)[progression.Defense],
			kind.TickPower,
			b.books.Chart.MultiplierAgainst(known.Element, target.Affinity),
		)
	}
	applied := 0
	wasted := 0
	for i := 0; i < application.Stacks; i++ {
		added, full := target.Statuses.Apply(kind, tick)
		if added {
			applied++
		}
		if full {
			wasted++
		}
	}
	b.emit(Event{
		Kind: StatusApplied, At: turn.At, Turn: turn.Number, Actor: actor.ID,
		Target: target.ID, Skill: known.ID, Status: kind.ID,
		Stacks: applied, Amount: tick, Chance: application.Chance,
		Remaining: int64(target.Statuses.Stacks(kind.ID)),
		Note:      wastedNote(wasted),
	})
}

func wastedNote(wasted int) string {
	if wasted == 0 {
		return ""
	}
	return fmt.Sprintf("%d wasted at the cap", wasted)
}
