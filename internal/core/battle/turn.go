package battle

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/passive"
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

	// Before the tick, because a summon whose time is up is not on the board to
	// be poisoned: ticking it first would log damage to a unit that is about to
	// be told it was never going to act, and could kill it — turning a copy that
	// ran out into a copy that was beaten.
	// Before the tick, because a summon whose time is up is not on the board to
	// be poisoned: ticking it first would log damage to a unit that is about to
	// be told it was never going to act, and could kill it — turning a copy that
	// ran out into a copy that was beaten.
	if b.expired(unit) {
		b.dismiss(unit, "out of turns")
		b.settle()
		return &Prompt{Unit: unit.ID, Turn: turn.Number, At: turn.At, Skipped: true, Reason: "left"}, nil
	}
	controlled := unit.Statuses.Has(stunStatus)
	b.tickStatuses(unit, turn)
	b.retuneAll(turn)
	if unit.Dead {
		b.emit(Event{Kind: TurnSkipped, At: turn.At, Turn: turn.Number, Actor: unit.ID, Note: "died"})
		// A turn that a timed effect took is still a turn that ended, and the
		// death it ended with may have been the one that froze the board.
		b.settle()
		return &Prompt{Unit: unit.ID, Turn: turn.Number, At: turn.At, Skipped: true, Reason: "died"}, nil
	}
	if controlled {
		b.spendCooldowns(unit)
		b.emit(Event{Kind: TurnSkipped, At: turn.At, Turn: turn.Number, Actor: unit.ID, Note: stunStatus})
		// Control was read before durations were spent, so the stun that costs
		// this turn may have expired during the tick that just ran. This is
		// therefore the turn on which the last timed thing on the board can go.
		b.settle()
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
	// The healing total is deliberately dropped. status.Set.Tick still computes
	// it and the status package still tests it — it is a correct thing for a Set
	// to be able to report — but this caller cannot use it: healing is resolved
	// per entry below so that each regeneration can name itself, and adding the
	// total back on top is exactly the double-count this shape exists to avoid.
	damage, _, expired := unit.Statuses.Tick()
	// Healing is applied here, entry by entry, while damage is applied once
	// below from the total. The asymmetry is not an oversight: heal emits its
	// own event and wound does not, so a regeneration resolved from the total
	// would either lose the name of the status that healed — the one thing this
	// loop exists to record — or say it twice, once here and once from heal.
	// Per entry, each regeneration produces exactly one Healed carrying its own
	// name, its own amount, and the health it left behind.
	//
	// It is also the more truthful arithmetic. heal stops at full health, so two
	// regenerations worth more than the room between them have the second one
	// clamped, which a single total would hide behind one number.
	//
	// Healing runs before the damage below, deliberately: a regeneration that
	// would carry a unit past a poison tick should do so, rather than the order
	// of two totals deciding who lives.
	for _, entry := range before {
		if entry.TickAmount <= 0 {
			continue
		}
		// A regeneration reports itself as a heal rather than as a tick, so a
		// reader is never asked to work out from the status name whether a
		// number was taken or given.
		if entry.Category == status.Regen {
			b.heal(unit, entry.TickAmount, turn, entry.ID)
			continue
		}
		b.emit(Event{
			Kind: StatusTicked, At: turn.At, Turn: turn.Number, Actor: unit.ID,
			Status: entry.ID, Amount: entry.TickAmount, Stacks: entry.Stacks,
		})
	}
	for _, id := range expired {
		b.emit(Event{
			Kind: StatusExpired, At: turn.At, Turn: turn.Number, Actor: unit.ID, Status: id,
		})
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
// The status id is what healed, and it is empty for a skill restoring health
// directly: a reader of a Healed wants to know whether a number came from a
// regeneration still on the board or from a cast that is already over, and the
// event has no other way to tell them apart.
func (b *Battle) heal(unit *Unit, amount int64, turn atb.Turn, from string) {
	if amount <= 0 || unit.Dead || unit.HP >= unit.MaxHP() {
		return
	}
	amount, reduced := healingFor(unit, amount)
	// Nothing landed, which is not the same question as the guard above and is
	// deliberately not a second floor: healingFor promises a non-negative amount,
	// so this asks only whether the cut took all of it. Written == rather than <=
	// for exactly that reason — two floors for one invariant is a guard a mutation
	// can delete for free, which is what happened to the reply drain's damage > 0.
	if amount == 0 {
		return
	}
	if room := unit.MaxHP() - unit.HP; amount > room {
		amount = room
	}
	unit.HP += amount
	b.emit(Event{
		Kind: Healed, At: turn.At, Turn: turn.Number, Actor: unit.ID,
		Status: from, Amount: amount, Remaining: unit.HP, Reduced: reduced,
	})
	b.reconsider(unit, turn)
}

// healingFor is what a unit actually receives out of an amount aimed at it: the
// share its own statuses take off, and the share they took, for the event.
//
// # One definition, two callers, and they are all the callers there are
//
// heal and drain are the only two functions in this engine that raise a unit's
// health — every restores, every regeneration tick and every drain, from a skill
// or from a trait, comes through one of them — so a cut declared here reaches all
// five healing sources with nothing to keep in step. drain is separate from heal
// only because its event carries Drained; writing the arithmetic twice would be
// two answers to one question, and the one that got edited would be the one the
// tests happened to reach.
//
// # ⚠️ Reduce BEFORE the cap, never after
//
// Both callers cap the amount at the room left to full afterwards, and the order
// is the whole mechanic. Capping first hands this function a number that is
// already the room rather than the heal, so on a nearly-full unit the cut would
// come off something that was going to be thrown away anyway and the debuff would
// be invisible exactly where a sustain build lives. Reduce, then cap.
//
// # ⚠️ Floored at nought, because a heal may never become damage
//
// Set.HealShare accumulates per stack and does not bound itself, so authored data
// can reach past total negation — a max_stacks of three at -400 a stack asks for
// -1200, and nothing refuses it, because the bound is here. The multiplier
// is floored at nought here rather than the share being clamped there, because
// this is where the amount is and a negative amount handed back would be
// subtracted by a caller that only knows how to add. The floor is what makes
// "cut" the whole of what the category can do.
func healingFor(unit *Unit, amount int64) (landed int64, reduced int) {
	share := unit.Statuses.HealShare()
	if share == 0 {
		return amount, 0
	}
	landing := scale.Base + share
	if landing < 0 {
		landing = 0
	}
	return amount * int64(landing) / int64(scale.Base), scale.Base - landing
}

func (b *Battle) wound(unit *Unit, damage int64, turn atb.Turn) {
	unit.HP -= damage
	if unit.HP <= 0 {
		b.kill(unit)
		return
	}
	b.reconsider(unit, turn)
}

// reconsider turns the holder's gated traits on and off as its health crosses
// the line they were authored against.
//
// It is called from every place health moves and from nowhere else, and there
// are three rather than the two a reading of this file suggests: heal, wound,
// and the strike loop in resolveAgainst, which subtracts from a target directly
// rather than going through wound. That third one is the one that matters —
// almost all the damage in a battle is dealt there, and a version of this that
// hooked only the two named functions would leave a gate that opened for a
// poison tick and never for a sword.
//
// It takes the unit whose health moved rather than sweeping all ten, because a
// gate reads its own holder and nobody else's: the other nine would be the same
// answer computed to be discarded, on every point of damage in the battle.
//
// Both directions emit. A trait coming on or going off changes a number a reader
// can see, and the log is the only contract a renderer has, so a grant that
// arrived silently would be a damage figure that moved with nothing to account
// for it. And both directions retune: a gated trait touching speed reorders the
// queue, and a wait computed against a speed the unit no longer has is wrong for
// the rest of the battle.
//
// A unit at nought health is skipped, and the test is health rather than the
// Dead flag because the strike loop leaves a target at zero for the rest of the
// skill and kills it afterwards — so a flag-only guard would announce a trait
// coming on to a unit whose died line is two events away.
func (b *Battle) reconsider(unit *Unit, turn atb.Turn) {
	if unit.Dead || unit.HP <= 0 || len(unit.Passives) == 0 || b.books.Passives == nil {
		return
	}
	changed := false
	for _, id := range unit.Passives {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			continue
		}
		// An ungated trait never moves, and asking is cheaper than the two loops
		// below. A trait that grants nothing has nothing to hold either way: its
		// gate is read live, at the site that reads it, which is what lets a
		// resistance stop protecting a healed unit without anything here.
		if held.While == nil || len(held.Grants) == 0 {
			continue
		}
		wanted := b.inForce(unit, held)
		for _, grant := range held.Grants {
			if wanted == unit.Statuses.Has(grant.Status) {
				continue
			}
			if wanted {
				kind, err := b.books.Statuses.Lookup(grant.Status)
				if err != nil {
					continue
				}
				unit.Statuses.Hold(kind, grant.Stacks)
				b.emit(Event{
					Kind: PassiveHeld, At: turn.At, Turn: turn.Number,
					Actor: unit.ID, Passive: held.ID,
					Status: grant.Status, Stacks: grant.Stacks,
				})
			} else {
				released := unit.Statuses.Release(grant.Status)
				b.emit(Event{
					Kind: PassiveReleased, At: turn.At, Turn: turn.Number,
					Actor: unit.ID, Passive: held.ID,
					Status: grant.Status, Stacks: released,
				})
			}
			changed = true
		}
	}
	if changed {
		b.retuneAll(turn)
	}
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
	// This is the turn a deadlock is made of, so it is the one that has to ask.
	// A unit passing for want of anything to use is the symptom the whole
	// Stalemate outcome exists to catch, and the board is at rest here.
	b.settle()
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
	// A taunt narrows a hostile skill to the units taunting, and only a hostile
	// one: a taunted unit may still shield itself, cleanse itself and help its
	// own side, because what a taunt takes is the choice of *enemy* and not the
	// turn. A skill aimed at both halves is left alone too — it is not a choice
	// of enemy in the first place.
	if known.Target == skill.Enemy {
		if forced := b.taunters(unit); len(forced) > 0 {
			return forced
		}
	}
	reachable := b.reachableRanks(unit, known)
	out := make([]hex.Offset, 0, hex.Cols*hex.Rows)
	for _, cell := range hex.Cells() {
		if !known.Target.Reaches(unit.Side, cell.Side()) {
			continue
		}
		if b.occupant(cell) == nil {
			continue
		}
		// A cell on the caster's own half costs no range at all: helping the
		// squad you are standing in is not a question of reach. That covers the
		// ally-aimed skills and the own half of an all-sided one, and it is why
		// only the opposing half is filtered below.
		if cell.Side() != unit.Side && !reachable[cell] {
			continue
		}
		out = append(out, cell)
	}
	return out
}

// reachableRanks is the set of opposing cells a skill may be pointed at, by
// **rank** rather than by distance.
//
// # Why not distance
//
// A unit never moves, and most skills declare a range of one, so measuring from
// the caster's own cell meant a unit placed at the back could not use its own
// kit — the range it needed was a fact about where the author had put it rather
// than about the skill. Reach is now read from the *far* side: how many of the
// enemy's ranks an attack can get through, counted from their frontline.
//
// # The rule
//
// Range N reaches the first N **occupied** ranks. An empty rank costs nothing —
// there is nobody there to shoot past — so a range of one finds the enemy's
// foremost survivors wherever they are standing.
//
// Blocking is by the whole rank: one unit anywhere in a rank shields everybody
// behind it. That is the decision that gives the board its shape — killing the
// front rank is what opens the one behind — and it is deliberately not a
// per-file rule, which would let a single gap expose a whole column.
//
// ⚠️ A taunt is not filtered by any of this, and that is settled above rather
// than here: `aims` returns the taunters before this is called, so a taunter in
// the back rank drags an attack through everything in front of it. Taking the
// choice of enemy is the whole of what a taunt does, and a taunt that could be
// walled off would be a status the front rank cancels.
func (b *Battle) reachableRanks(unit *Unit, known skill.Skill) map[hex.Offset]bool {
	out := make(map[hex.Offset]bool, hex.FormationCols*hex.Rows)
	if known.Range <= 0 {
		return out
	}
	spent := 0
	for _, rank := range hex.Ranks(unit.Side.Opposing()) {
		held := make([]hex.Offset, 0, hex.Rows)
		for _, cell := range rank {
			if b.occupant(cell) != nil {
				held = append(held, cell)
			}
		}
		if len(held) == 0 {
			continue
		}
		spent++
		if spent > known.Range {
			break
		}
		for _, cell := range held {
			out[cell] = true
		}
	}
	return out
}

// taunters is the cells holding somebody taunting this unit, in the board's own
// order, and empty when nobody is.
//
// Range is not read, which is the whole of what makes a taunt a taunt here.
// Nothing on this board moves, so a taunt that could be answered by standing far
// enough away would be ignored by exactly the long-ranged attackers a tank most
// needs to pull off its own back column — and a tank nobody can be made to
// attack is furniture. So the taunter is reachable from anywhere, and a skill of
// range one aimed at a taunter four cells away lands: the damage formula has
// never read distance, only the legality of the aim did.
//
// Several taunters are not an ambiguity, only a smaller list: every one of them
// is a legal aim and nobody else is.
func (b *Battle) taunters(unit *Unit) []hex.Offset {
	out := make([]hex.Offset, 0, hex.MaxTeamSize)
	for _, cell := range hex.Cells() {
		if !skill.Enemy.Reaches(unit.Side, cell.Side()) {
			continue
		}
		other := b.occupant(cell)
		if other == nil || other.Statuses.Stacks(tauntStatus) == 0 {
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
	// Read before the use is announced, because the announcement carries it: a
	// gradient that moved the power and left no trace in the log would be the trap
	// Pierce, Refused and Drained each record -- a number a reader cannot account
	// for from the skill and the stats alone.
	brought := swingOf(known, unit)
	b.emit(Event{
		Kind: SkillUsed, At: turn.At, Turn: turn.Number, Actor: unit.ID,
		Skill: known.ID, Cell: hex.At(aim), Power: known.Power, Chance: known.Accuracy,
		Gradient: brought.Share,
	})

	// Before applyToSelf, deliberately: a skill that both grants a status and
	// spends one would otherwise pay itself, and "hits harder while furied"
	// would hold for the skill that just applied the fury.
	b.spend(unit, known, brought, turn)
	b.applyToSelf(unit, known, turn)
	// After the self-applies, so a skill that buffs itself and then copies
	// itself copies the buffed line. That ordering is the only reason the two
	// are not interchangeable, and it is the reading an author expects.
	b.summon(unit, known, turn)
	if known.Target == skill.Self {
		b.strip(unit, unit, known, turn)
		// The restore the shape walk below would have paid out. A self-aimed
		// skill reaches exactly one unit — the caster — and never a splashed
		// one, so it is position nought.
		b.restore(unit, unit, known, 0, turn)
		b.retuneAll(turn)
		b.settle()
		return nil
	}
	shape, err := b.books.Patterns.Lookup(known.Pattern)
	if err != nil {
		return err
	}
	var bitten []*Unit
	for position, cell := range covers(shape, known, aim) {
		target := b.occupant(cell)
		if target == nil {
			continue
		}
		// Every target takes the whole skill before anybody answers it. A reply
		// resolved here, in the middle of the loop, could kill the actor while
		// it still had cells to hit — and "what happens to the rest of the
		// skill" is a question with no good answer, so the shape of this loop is
		// what stops it being asked.
		if dealt := b.resolveAgainst(unit, target, known, shape.Name, position, brought, turn); dealt > 0 {
			bitten = append(bitten, target)
		}
		if b.finished {
			return nil
		}
	}
	b.answer(unit, bitten, turn)
	b.retuneAll(turn)
	b.settle()
	return nil
}

// answer is what the units a skill just hurt cost the unit that hurt them.
//
// # When it runs, and why here rather than anywhere else
//
// After the whole skill, once, per holder. Three of the four rules this feature
// was designed under are simply where this call sits: a reply answers a *use* of
// a skill rather than a strike, so a trait's worth cannot scale with somebody
// else's strike count; the holder takes every strike first, so a striker never
// dies partway through its own turn; and a reply never triggers a reply, which
// is closed by the shape of the code rather than by a depth counter — the list
// is built from the skill loop above and a reply is not in it, so there is
// nothing here for a second one to answer.
//
// # What does not answer
//
// A holder the skill killed does not, the way a dead unit cannot be healed.
// Neither does one whose trait is gated shut, nor the actor itself: a skill that
// caught its own caster is still not somebody attacking it.
func (b *Battle) answer(actor *Unit, bitten []*Unit, turn atb.Turn) {
	if len(bitten) == 0 || b.books.Passives == nil {
		return
	}
	for _, holder := range bitten {
		if holder.Dead || holder == actor {
			continue
		}
		for _, id := range holder.Passives {
			held, err := b.books.Passives.Lookup(id)
			if err != nil {
				continue
			}
			if !held.Replies.Answers() || !b.inForce(holder, held) {
				continue
			}
			b.reply(holder, actor, held, turn)
			if actor.Dead {
				// The attacker is gone, so anybody still holding a reply is
				// answering nothing. Returning rather than breaking is the
				// difference between a corpse taking one more hit and taking
				// several.
				return
			}
		}
	}
}

// reply resolves one trait's answer against the unit that attacked its holder.
//
// The damage goes through combat.Rules and the statuses through inflict, which
// is the whole of "not a second damage path": a reply is priced by the same
// curve as a skill, refused by the same resistances, and written to the log as
// the same kinds — so a replay reads it without knowing it was a trait, and
// --verify re-runs it from the seed.
//
// It may kill. Damage gets no exemption for arriving out of turn, so a battle
// can end on a turn nobody took; the caller's settle is what notices.
func (b *Battle) reply(holder, attacker *Unit, held passive.Passive, turn atb.Turn) {
	from := fromTrait(held)
	if held.Replies.Power > 0 {
		damage, multiplier := b.replyDamage(holder, attacker, held)
		attacker.HP -= damage
		if attacker.HP < 0 {
			attacker.HP = 0
		}
		b.emit(Event{
			Kind: Damaged, At: turn.At, Turn: turn.Number, Actor: holder.ID,
			Target: attacker.ID, Passive: held.ID, Strike: 1,
			Power: held.Replies.Power, Multiplier: multiplier,
			Amount: damage, Remaining: attacker.HP,
		})
		// A reply drains like anything else its holder does. The trait's own
		// share only — a reply has no skill, so there is no skill share to add
		// to it — but the same cap and the same conservation rule.
		//
		// This was missing, and the description was the thing that said so:
		// "mọi đòn của nó hút lại 25% sát thương gây ra" / "everything it does
		// takes back 25%". A reply is an đòn. The two jobs sit on one Passive,
		// so one trait can hold both today — the placement's single trait slot
		// stops a unit carrying a replier and a drainer, and stops nothing at
		// all about a trait that is both.
		//
		// Before the kill rather than after, which is the skill path's answer to
		// the same question: resolveAgainst drains from what it dealt whether or
		// not the target fell, so a reply that finishes somebody has to pay out
		// too. Draining after the return would make lethal damage the one kind
		// that is worth nothing to take back.
		//
		// No guard on the damage, unlike the skill path. There, dealt is summed
		// across strikes that may all have missed and can be nought with a skill
		// behind it; here the branch this sits in has already said the reply has
		// power, and drain refuses a share of nothing on its own. A guard that
		// cannot fire is one a reader has to work out the meaning of.
		if drained := drainShare(b.lifesteal(holder)); drained > 0 {
			b.drain(holder, damage, drained, turn)
		}
		if attacker.HP <= 0 {
			b.kill(attacker)
			return
		}
		b.reconsider(attacker, turn)
	}
	for _, application := range held.Replies.Applies {
		b.inflict(holder, attacker, from, application, turn)
	}
}

// spend reads the caster's own condition and reports what it adds to the skill's
// power, consuming the status if the condition says to.
//
// Once per use. That is the whole reason it is here rather than inside
// resolveAgainst, which runs once per cell a shape covers: a condition consumed
// per target would charge a column three times and a single-target skill once,
// and the difference would be written on neither skill.
//
// The events are the ones a target-side condition already emits, with the actor
// as its own target. A reader can tell them apart by that, and a third event kind
// would have been a third thing for every renderer to learn for no new fact.
func (b *Battle) spend(unit *Unit, known skill.Skill, brought swing, turn atb.Turn) {
	if known.SelfRequires == nil {
		return
	}
	against := conditionCaster(known, unit)
	if !known.SelfAmplified(against) {
		return
	}
	b.emit(Event{
		Kind: Amplified, At: turn.At, Turn: turn.Number, Actor: unit.ID,
		Target: unit.ID, Skill: known.ID, Status: known.SelfRequires.Status,
		Stacks: against.Stacks, Power: known.Power + brought.Bonus,
	})
	if known.SelfRequires.Consume {
		consumed, forgone := unit.Statuses.Consume(known.SelfRequires.Status)
		b.emit(Event{
			Kind: StatusConsumed, At: turn.At, Turn: turn.Number, Actor: unit.ID,
			Target: unit.ID, Skill: known.ID, Status: known.SelfRequires.Status,
			Stacks: consumed, Amount: forgone,
		})
		// A consumed stat change is a stat change gone, so the queue has to be
		// told before the next turn is picked -- the same sweep a status wearing
		// off gets. Act retunes at the end of a use, and this is a use that has
		// not finished yet.
		b.retuneAll(turn)
	}
}

func (b *Battle) applyToSelf(unit *Unit, known skill.Skill, turn atb.Turn) {
	for _, application := range known.SelfApplies {
		b.inflict(unit, unit, fromSkill(known), application, turn)
	}
}

// restore is a skill's `restores` paid out to one unit it reached.
//
// It reads the caster's scaling stat and skips the defence curve entirely: see
// combat.Rules.Restore. A splashed target gets the reduced share, the same as
// damage, because a shape's edge is worth less wherever it lands — and a
// self-aimed skill is never splashed, so it is always position nought.
//
// ⚠️ **This is a method because it has two callers, and the second one was
// missing.** It used to sit inline in resolveAgainst, which Act returns before
// for a Target: Self skill — so `synthesis`, whose whole body is a 900 restore
// on itself, healed nothing, and `withdraw` paid out its block and dropped its
// 500. Both are self-aimed, and both were the only shipped skills that declare
// `restores` at all, so the field did nothing anywhere.
//
// ⚠️ **The rating could see it and the engine could not**, which is the shape
// the file's own rule names: pricing.restored prices a restore off
// combat.Rules.Restore, so Suggest chose synthesis on a hurt caster expecting
// up to nine hundred health and got none. That is "a price built from a second
// reading lets the opponent prefer a skill for something the skill does not
// do", except the second reading was the honest one.
func (b *Battle) restore(actor, target *Unit, known skill.Skill, position int, turn atb.Turn) {
	if known.Restores <= 0 {
		return
	}
	restore := known.Restores
	if position > 0 {
		restore = restore * b.books.Patterns.SplashPower / scale.Base
	}
	b.heal(target, b.books.Rules.Restore(
		combat.PickScaling(known.Scaling.Source,
			actor.Base[known.Scaling.Stat], b.Stats(actor)[known.Scaling.Stat]),
		restore), turn, "")
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
func (b *Battle) resolveAgainst(actor, target *Unit, known skill.Skill, shape string,
	position int, brought swing, turn atb.Turn) (dealt int64) {
	b.strip(actor, target, known, turn)

	power := known.Power
	if known.Requires != nil {
		against := conditionTarget(known, target)
		stacks := against.Stacks
		if known.Amplified(against) {
			power = known.PowerAgainst(against)
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
	// The caster's own terms land before the splash share is taken, the same as
	// the target's: they are part of what the skill hits for, and a shape's edge
	// is worth less however the power was arrived at.
	power = brought.applied(power)
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
		Crit:          known.Crit,
		SkillAccuracy: known.Accuracy,
		AccuracyStat:  actorStats[progression.Accuracy],
		DodgeStat:     targetStats[progression.Dodge],
	}

	connected := false
	// Carried beside connected rather than folded into it: a blow that was eaten
	// by a shield arrived, which a blow that missed did not, and the two deliver
	// different things. See the rider block below.
	blocked := false
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
				blocked = true
			default:
				event.Kind = Damaged
				event.Amount = attempt.Damage
				event.Critical = attempt.Critical
				dealt += attempt.Damage
				target.HP -= attempt.Damage
				if target.HP < 0 {
					target.HP = 0
				}
				event.Remaining = target.HP
				connected = true
			}
			b.emit(event)
			// The gate is re-read here rather than once when the skill is
			// finished, because this is where the health moved. A trait that
			// came on after the first strike of three is in force for the other
			// two, which is what "in force while its holder is hurt" says — and
			// waiting until the end would make the same trait worth less against
			// a three-strike skill than against a single one, for no reason a
			// reader could find on either skill.
			if event.Kind == Damaged {
				b.reconsider(target, turn)
			}
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

	// A blow the shield ate still arrived, so its riders are FILTERED rather than
	// cancelled with it: a shield stops the blow and the wear, but not the
	// contamination — status.Category.OutlastsAShield is the whole of that rule
	// and carries the reading. A strike that MISSED reaches neither arm, because
	// nothing touched the target at all; that is why blocked is carried beside
	// connected instead of widening it.
	//
	// throughAShield only when nothing landed. A multi-strike skill with one
	// strike eaten and one through has connected, and the blow it got through is
	// what the riders ride on.
	if connected || blocked {
		throughAShield := !connected
		for _, application := range known.Applies {
			if throughAShield && !b.outlastsAShield(application) {
				continue
			}
			b.inflict(actor, target, fromSkill(known), application, turn)
		}
		// The actor's traits contribute to the same list rather than to a second
		// pass of their own, so a trait's rider goes through the same roll, the
		// same resistance and the same event as a skill's own application. A
		// separate path would be a second place for all three to be got wrong.
		// The shield filter is the same call for the same reason: a trait's rider
		// surviving a block on a different rule from a skill's own application
		// would be a difference no reader could find on either.
		//
		// Only on a skill with power. resolveAgainst deliberately never asks
		// which side a target is on, so "already dealing damage to it" is the
		// available way to say the skill is hostile — and an ally-aimed damaging
		// skill is already an attack on whoever it catches. A block can only
		// happen inside the power > 0 branch above, so the guard is already true
		// on the blocked arm and is kept for the arm where it is not.
		if power > 0 {
			for _, application := range b.riders(actor) {
				if throughAShield && !b.outlastsAShield(application) {
					continue
				}
				b.inflict(actor, target, fromSkill(known), application, turn)
			}
		}
	}
	b.restore(actor, target, known, position, turn)
	// A drain takes its share of what was *dealt*, so a strike that missed or
	// was blocked returns nothing, and one that overkilled returns only the
	// damage that landed.
	//
	// The skill's share and the caster's traits are added rather than composed,
	// and the total is capped at the base. That cap is not the hard cap this
	// engine rejects elsewhere: a buff ceiling bounds how good a number may get,
	// where this bounds a *conservation* — health taken back cannot exceed damage
	// dealt, which is the same invariant skill.resolve enforces on a single
	// share. Saturating instead would be worse than either: it would take a
	// trait's four hundred and pay out two hundred and eighty-five on a skill
	// that drains nothing, so a trait would be worth less than it says.
	if drained := drainShare(known.Drains + b.lifesteal(actor)); drained > 0 && dealt > 0 {
		b.drain(actor, dealt, drained, turn)
	}
	if target.HP <= 0 {
		b.kill(target)
	}
	_ = shape
	return dealt
}

// lifesteal is the share of its damage the actor's traits take back, added
// across every trait in force.
//
// Added rather than multiplied because a share of the damage dealt is not a
// chance: two resistances compose by what each lets through, and two drains
// simply both drain. The sum is bounded by its caller.
func (b *Battle) lifesteal(actor *Unit) int {
	if len(actor.Passives) == 0 || b.books.Passives == nil {
		return 0
	}
	total := 0
	for _, id := range actor.Passives {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			continue
		}
		if held.Drains == 0 || !b.inForce(actor, held) {
			continue
		}
		total += held.Drains
	}
	return total
}

// drainShare bounds a total drain at the base. See the note at its call site for
// why this cap is a conservation rule rather than a ceiling.
func drainShare(total int) int {
	if total > scale.Base {
		return scale.Base
	}
	return total
}

// drain heals the actor for its share of what a strike dealt, and says on the
// event what the share was.
//
// A drain is a Healed like any other, so the kind stays what a renderer already
// knows; what it carries is the one number that makes the heal reproducible now
// that the skill's own figure is no longer the whole of it.
func (b *Battle) drain(actor *Unit, dealt int64, share int, turn atb.Turn) {
	amount := dealt * int64(share) / int64(scale.Base)
	if amount <= 0 || actor.Dead || actor.HP >= actor.MaxHP() {
		return
	}
	// The same cut heal takes, from the same definition and in the same place:
	// before the amount is capped at the room left. A drain is healing however it
	// was earned, so a festering attacker takes back less of what it dealt.
	amount, reduced := healingFor(actor, amount)
	// == rather than <= for the reason heal's own copy of this guard states: the
	// floor lives in healingFor and a second one here would be a guard a mutation
	// deletes for free.
	if amount == 0 {
		return
	}
	if room := actor.MaxHP() - actor.HP; amount > room {
		amount = room
	}
	actor.HP += amount
	b.emit(Event{
		Kind: Healed, At: turn.At, Turn: turn.Number, Actor: actor.ID,
		Amount: amount, Remaining: actor.HP, Drained: share, Reduced: reduced,
	})
	b.reconsider(actor, turn)
}

// inflict rolls one status application.
//
// The chance starts as the skill's own — whether the hit landed and whether the
// status takes hold are two separate questions, and keeping them separate is what
// makes both legible. The one thing that touches it is the *target's* traits: a
// resistance is the only reason an application's chance is not the skill's own,
// and it is applied here rather than at status.Set.Apply because this is where the
// roll happens. Apply has no dice, so a resistance living there could only refuse
// outright — a hard cap on a continuous quantity, which this engine has chosen
// against everywhere else.
// origin is where an effect came from, and it exists because a trait can now put
// a status on somebody without a skill being involved.
//
// inflict only ever wanted three things out of the skill it used to be handed:
// what to name in the log, which element to price a tick against, and which of
// the actor's stats that tick scales off. Naming those three is what lets a
// reply go through the very same function rather than a copy of it with the
// skill parts taken out — which is how the two would have started disagreeing
// about resistances, or about what a status_applied event says.
type origin struct {
	// Skill and Passive name the source, and exactly one of them is set. An
	// event carries whichever it was, so a reader is never asked to work out
	// from an empty field which kind of thing happened.
	Skill   string
	Passive string
	Element element.Element
	// Scaling is which stat prices this, and whether it reads the base line or
	// the modified one.
	//
	// The whole declaration rather than just the stat, because the source is
	// half of it: a skill that says "base" and then had its poison tick read the
	// current value would be two answers to one question, and the only reason
	// nothing has noticed is that no shipped skill declares a source at all.
	Scaling skill.Scaling
}

// fromSkill is the origin of anything a skill does.
func fromSkill(known skill.Skill) origin {
	return origin{Skill: known.ID, Element: known.Element, Scaling: known.Scaling}
}

// stat is what this origin is priced against, read off the unit it belongs to.
func (o origin) stat(b *Battle, unit *Unit) int64 {
	return combat.PickScaling(o.Scaling.Source,
		unit.Base[o.Scaling.Stat], b.Stats(unit)[o.Scaling.Stat])
}

// fromTrait is the origin of anything a trait does on its own account.
//
// Neutral, always: the elemental chart prices what one creature threw at
// another, and a trait reading it would make a fire creature's blood weak to
// water for a reason written nowhere on the trait.
//
// The stat is the trait's own now. It used to be attack, on the grounds that a
// trait had nowhere to say otherwise — and that turned out to be the wrong
// default rather than a missing field: a trait that answers whoever hit it
// belongs to a unit built to be hit, which is an armoured unit and not a sharp
// one, so pricing every reply off attack made thorns worth least to exactly the
// character thorns are for.
// replyDamage is what one trait's answer takes off the unit that attacked its
// holder, and the elemental multiplier it was priced with.
//
// Neutral against whatever the attacker is, and the figure is returned rather
// than recomputed by a caller wanting it: a log that carried the damage but not
// the multiplier it was priced with would be a record that cannot account for
// its own number — and "it is always neutral" is a fact about today's rule
// rather than about this battle.
//
// It is a function of its own because the rating has to charge an option for the
// reply it provokes, and CLAUDE.md § Rating an action allows exactly one reading
// of any figure the resolution uses. reply lands this number and
// pricing.replied charges for it; a second expression in price.go would let the
// opponent decline an attack over a blow the trait does not actually strike.
//
// ⚠️ It mutates nothing, which is what makes it callable from a rating at all.
func (b *Battle) replyDamage(holder, attacker *Unit, held passive.Passive) (damage int64, multiplier int) {
	from := fromTrait(held)
	multiplier = b.books.Chart.MultiplierAgainst(from.Element, attacker.Affinity)
	return b.books.Rules.Damage(
		from.stat(b, holder),
		b.Stats(attacker)[progression.Defense],
		held.Replies.Power,
		multiplier,
	), multiplier
}

func fromTrait(held passive.Passive) origin {
	scaling := skill.DefaultScaling()
	if held.Replies != nil {
		scaling = held.Replies.Scaling
	}
	return origin{Passive: held.ID, Element: element.Neutral, Scaling: scaling}
}

func (b *Battle) inflict(actor, target *Unit, from origin, application skill.Application, turn atb.Turn) {
	kind, err := b.books.Statuses.Lookup(application.Status)
	if err != nil {
		return
	}
	// The actor's side first and the target's second, which reads in the order
	// the fiction does — a trait sharpens what is thrown, armour refuses what
	// arrives — and is arithmetically the same either way round, which is the
	// point of both composing by multiplication.
	amplifiedEffect, amplifiedChance := b.amplify(actor, kind.ID)
	chance, refused := b.resist(target, kind.ID,
		int(raise(int64(application.Chance), amplifiedChance)))
	// Clamped last, and only here. A probability cannot exceed one, so an
	// amplifier on an application that already lands every time is worth
	// nothing — but clamping *before* the target's share would make the order
	// the two sides compose in matter: a certain application amplified then
	// halved is five hundred, while halved then amplified is six. Composing
	// first and clamping the result keeps the two interchangeable, which is the
	// property both sides were written around.
	chance = min(chance, scale.Base)
	// Chance is the figure that was actually rolled and Refused says why it is
	// not the skill's own. Without the second, a reader of the log cannot tell an
	// application that rolled badly from one the target refused — and the kind is
	// called status_resisted either way, which is exactly the confusion this
	// field exists to end.
	if !b.source.Chance(chance) {
		b.emit(Event{
			Kind: StatusResisted, At: turn.At, Turn: turn.Number, Actor: actor.ID,
			Target: target.ID, Skill: from.Skill, Passive: from.Passive,
			Status: kind.ID, Chance: chance, Refused: refused,
			AmplifiedChance: amplifiedChance,
		})
		return
	}
	// A ticking status is worth something only if a tick is computed here, where
	// it is frozen onto the stack. Two categories tick; everything else keeps the
	// zero, and a stack carrying zero is skipped by the whole downstream path.
	tick := int64(0)
	switch kind.Category {
	case status.Dot:
		// Full defence, deliberately, even when the skill applying the status
		// pierces. A tick is computed once here and frozen on the stack for the
		// rest of its life, so piercing it would be worth as many pierced hits
		// as the stack has turns left — a far larger effect than the per-strike
		// ratio the author wrote. That is why Pierced is applied by Strike and
		// not folded into the defence a caller passes.
		tick = raise(b.books.Rules.Damage(
			from.stat(b, actor),
			b.Stats(target)[progression.Defense],
			kind.TickPower,
			b.books.Chart.MultiplierAgainst(from.Element, target.Affinity),
		), amplifiedEffect)
	case status.Regen:
		// Restore rather than Damage, which drops two things and keeps one.
		//
		// No defence curve: combat.Rules.Restore records why — armour turns away
		// what is coming at a unit and has nothing to do with what is helping it,
		// so dividing here would make a unit's own armour quietly weaken its own
		// regeneration. And no elemental multiplier: the chart prices what one
		// creature threw at another, and a grass creature healing itself is not
		// throwing anything, so reading the chart would make aqua_ring worth more
		// or less depending on who cast it on whom.
		//
		// What is kept is the actor's scaling stat and the freeze. Both are the
		// same as a damage-over-time's, and the freeze is the point: two casters
		// stacking one regeneration each contribute what their own attack was
		// worth at the moment they cast, which is what status.Regen already
		// documents and what nothing was honouring.
		//
		// Not amplified, and that is a decision rather than an omission. See
		// passive.Amplification: a trait's effect share is described to a player
		// in the words of harm, and a share that heals under a sentence reading
		// "ticks harder" would be a description that lies — which is the one
		// thing every derived description in this engine exists to prevent.
		tick = b.books.Rules.Restore(from.stat(b, actor), kind.TickPower)
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
		Target: target.ID, Skill: from.Skill, Passive: from.Passive, Status: kind.ID,
		Stacks: applied, Amount: tick, Chance: chance, Refused: refused,
		AmplifiedChance: amplifiedChance, AmplifiedEffect: amplifiedEffect,
		Remaining: int64(target.Statuses.Stacks(kind.ID)),
		Note:      wastedNote(wasted),
	})
}

// riders is the applications a unit's traits add to what its damaging skills
// inflict, in the order the traits were enlisted with.
//
// Declaration order, because these reach the log: a set ordered by anything a
// map decided would stop a battle replaying from its seed.
func (b *Battle) riders(actor *Unit) []skill.Application {
	if len(actor.Passives) == 0 || b.books.Passives == nil {
		return nil
	}
	var out []skill.Application
	for _, id := range actor.Passives {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			continue
		}
		if !b.inForce(actor, held) {
			continue
		}
		out = append(out, held.Applies...)
	}
	return out
}

// outlastsAShield reports whether an application would still land on a target
// whose shield ate the strike carrying it.
//
// The rule itself is status.Category.OutlastsAShield; this is only the lookup in
// front of it. An unknown status answers no, which is what inflict does with the
// same lookup a line later — so a rider naming a status the book does not hold is
// dropped in one place rather than reaching the roll through this one.
//
// ⚠️ That error branch is UNREACHABLE and is reported rather than tested: both
// skill.ParseBook and passive.ParseBook check every status name against the
// status book, so a parsed rider cannot name one the book lacks, and a mutation
// returning true there survives the whole suite. It is kept because the
// alternative — reading Category off a zero Kind — is Dot, so a missing status
// would silently be the one thing that goes through a shield.
func (b *Battle) outlastsAShield(application skill.Application) bool {
	kind, err := b.books.Statuses.Lookup(application.Status)
	if err != nil {
		return false
	}
	return kind.Category.OutlastsAShield()
}

// inForce reports whether a trait's gated half applies to its holder right now.
//
// A trait with no condition is always in force, so this is the only question a
// caller has to ask. Health is read live rather than snapshotted: a trait that
// turns on when its holder is hurt has to turn on during the turn it was hurt,
// not on the next one.
func (b *Battle) inForce(unit *Unit, held passive.Passive) bool {
	return held.While.Holds(unit.HP, unit.MaxHP())
}

// resist takes whatever the target's traits refuse off an application's chance,
// and reports the share refused.
//
// Sources compose by multiplying what each lets through, which is what a chance
// does anyway: two resistances of six hundred leave sixteen percent rather than
// none, so stacking diminishes for free and no saturation helper is needed. A
// single declared full thousand still reaches zero, which is the whole point of
// the absolute being available to an author and never to a stack.
//
// One resistance is exact: surviving comes back as scale.Base minus the amount
// with nothing lost, so the chance takes a single truncation. That is the common
// case, and the one worth being exact in.
func (b *Battle) resist(target *Unit, statusID string, chance int) (effective, refused int) {
	if chance <= 0 || len(target.Passives) == 0 || b.books.Passives == nil {
		return chance, 0
	}
	surviving := scale.Base
	for _, id := range target.Passives {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			continue
		}
		amount := held.Refuses(statusID)
		// Nought only. A **negative** share is a vulnerability and has to reach
		// the multiply below — skipping it here is how this silently did nothing
		// for the whole time a negative was refused at parse and this line went
		// unread.
		if amount == 0 {
			continue
		}
		// The gate is read here rather than at enlistment, so a trait that only
		// protects a hurt unit stops protecting it the moment it is healed back.
		if !b.inForce(target, held) {
			continue
		}
		surviving = surviving * (scale.Base - amount) / scale.Base
	}
	// Only the exact base is nothing happening. It used to be `>=`, which was
	// right while a share could only refuse — and is the line that would have
	// dropped a vulnerability on the floor, because a trait that invites a status
	// leaves *more* than the base surviving and would have taken this door out
	// with the chance untouched.
	if surviving == scale.Base {
		return chance, 0
	}
	// Refused is signed, and that is the decision this feature turned on. It is
	// the share the target took off the chance, so a share it *added* is a
	// negative — and the event it lands on is already named for the application
	// failing rather than for a resistance existing (a unit with no traits at all
	// emits it with a nought). Reading "refused -300" as "invited 30%" is the
	// same sentence the field already carries, in the other direction.
	//
	// The alternative was a second field, which would have been two names for one
	// number and left every reader to check both.
	return chance * surviving / scale.Base, scale.Base - surviving
}

// amplify is what the actor's traits add to a status they are inflicting: a
// share on the tick and a share on the chance.
//
// ⚠️ **This is the one place a trait reads the unit that is *acting* rather than
// the unit being acted on.** resist walks the target's passives a few lines
// above; this walks the actor's, and both run inside inflict. Every other job a
// trait has is about its holder, so the two being neighbours is worth noticing
// before adding a third: the parameter is not interchangeable, and passing the
// wrong unit would compile and would silently give a target the attacker's
// amplifier.
//
// Composed by multiplying what each trait raises, which is the arithmetic resist
// composes with read in the other direction. That is what makes the order the two
// sides are applied in irrelevant — the chance is multiplied by everything
// raising it and everything lowering it — and it is why neither function has to
// know the other exists.
func (b *Battle) amplify(actor *Unit, statusID string) (effect, chance int) {
	if len(actor.Passives) == 0 || b.books.Passives == nil {
		return 0, 0
	}
	raisedEffect, raisedChance := scale.Base, scale.Base
	for _, id := range actor.Passives {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			continue
		}
		byEffect, byChance := held.Boosts(statusID)
		if byEffect == 0 && byChance == 0 {
			continue
		}
		// The gate is read here for the same reason resist reads it: a trait
		// that only sharpens a hurt holder stops sharpening it the moment it is
		// healed back.
		if !b.inForce(actor, held) {
			continue
		}
		raisedEffect = raisedEffect * (scale.Base + byEffect) / scale.Base
		raisedChance = raisedChance * (scale.Base + byChance) / scale.Base
	}
	return raisedEffect - scale.Base, raisedChance - scale.Base
}

// raise applies an amplifier's share to a figure. A share of nought is the
// figure itself, so the common case takes no arithmetic at all rather than a
// multiplication and a division that cancel.
func raise(value int64, share int) int64 {
	if share <= 0 {
		return value
	}
	return value * int64(scale.Base+share) / int64(scale.Base)
}

func wastedNote(wasted int) string {
	if wasted == 0 {
		return ""
	}
	return fmt.Sprintf("%d wasted at the cap", wasted)
}
