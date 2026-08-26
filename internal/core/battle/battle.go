// Package battle is where the rest of the engine is assembled, and the only
// package that holds state.
//
// Everything it uses is pure: the geometry, the elemental chart, the damage
// formula, the buff layer, the timed effects, the turn order and the skill book
// all take numbers and return numbers. This package owns the numbers that
// change. That split is what makes a battle reproducible from a seed and a
// roster, and it is why the terminal client and a graphical one can be the same
// battle rendered twice rather than two implementations of it.
//
// The output is an event log. A caller drives the battle a turn at a time and
// drains the events; nothing about drawing a battle needs to read the state
// here, which is the property that keeps a renderer from quietly becoming a
// second copy of the rules.
package battle

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/rng"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// blockStatus is the shield whose stack count is a unit's block charges. Naming
// it once here is what keeps the charge mechanic and the timed-effect mechanic
// from drifting apart.
const blockStatus = "block"

// stunStatus is the control effect that costs a unit its action.
const stunStatus = "stun"

// Books are the parsed data a battle reads. They are never modified.
type Books struct {
	Rules    combat.Rules
	Chart    *element.Chart
	Bounds   modifier.Bounds
	Limits   progression.Limits
	Patterns *pattern.Book
	Statuses *status.Book
	Skills   *skill.Book
	// Passives is wanted only by a roster entry that names one. A battle whose
	// units hold no traits runs without it, which is what let the field arrive
	// without every existing caller having to be found.
	Passives *passive.Book
}

func (b Books) validate() error {
	switch {
	case b.Chart == nil:
		return fmt.Errorf("a battle needs an element chart")
	case b.Patterns == nil:
		return fmt.Errorf("a battle needs a pattern book")
	case b.Statuses == nil:
		return fmt.Errorf("a battle needs a status book")
	case b.Skills == nil:
		return fmt.Errorf("a battle needs a skill book")
	}
	if err := b.Rules.Validate(); err != nil {
		return err
	}
	if err := b.Bounds.Validate(); err != nil {
		return err
	}
	return b.Limits.Validate()
}

// Roster is one unit as it enters a battle, with its evolution and level already
// resolved into flat stats.
type Roster struct {
	ID   string
	Name string
	Side hex.Side
	// Slot is the formation slot the unit occupies, authored from its own side's
	// point of view. hex.Place maps it onto the shared board.
	Slot     hex.Offset
	Affinity element.Affinity
	Stats    progression.Values
	Skills   []string
	// Passives are the traits the unit holds, by id in the passive book.
	//
	// This is the first thing on a roster entry that is neither a stat nor a
	// skill, and it belongs here for the reason the archetype and the evolution
	// stage do not: those two are settled *before* a battle and leave nothing
	// behind but numbers, while a passive is in force during one and a replay has
	// to know about it. The test of what belongs on a Roster has always been
	// "does a replay read it", not "is it small".
	Passives []string
}

// Unit is a combatant's mutable state.
type Unit struct {
	ID       string
	Name     string
	Side     hex.Side
	Cell     hex.Offset
	Affinity element.Affinity
	// Base is the stat line before any timed effect. Statuses are applied to it
	// on demand rather than stored, so a debuff wearing off cannot leave a
	// stale number behind.
	Base progression.Values
	HP   int64
	Dead bool
	// Skills and Cooldowns run in step. A slice rather than a map, because the
	// order events come out in has to be the same every run.
	Skills    []string
	Cooldowns []int
	Statuses  status.Set
	// Passives are the ids the unit was enlisted with, kept so the log can say
	// which trait put a permanent status on. The statuses themselves are already
	// in Statuses; this is the provenance, and without it a reader sees a
	// permanent buff with nothing to account for it.
	Passives []string
}

// MaxHP is the health the unit started with.
func (u *Unit) MaxHP() int64 { return u.Base[progression.HP] }

// Battle is one fight in progress.
type Battle struct {
	books  Books
	source *rng.Source
	queue  *atb.Queue
	units  []*Unit
	byID   map[string]*Unit

	events   []Event
	acting   *Unit
	prompt   *Prompt
	awaiting bool
	finished bool
	winner   hex.Side
	decided  bool
	outcome  Outcome
}

// New sets up a battle. It validates the roster against every book, so a bad
// roster fails here rather than partway through a fight.
func New(books Books, seed uint64, roster []Roster) (*Battle, error) {
	if err := books.validate(); err != nil {
		return nil, err
	}
	if len(roster) == 0 {
		return nil, fmt.Errorf("a battle needs units")
	}
	fight := &Battle{
		books:  books,
		source: rng.New(seed),
		queue:  atb.New(),
		byID:   make(map[string]*Unit, len(roster)),
	}
	perSide := map[hex.Side]int{}
	occupied := map[hex.Offset]string{}
	for _, entry := range roster {
		unit, err := fight.enlist(entry, perSide, occupied)
		if err != nil {
			return nil, err
		}
		fight.units = append(fight.units, unit)
		fight.byID[unit.ID] = unit
	}
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		if perSide[side] == 0 {
			return nil, fmt.Errorf("no unit is on the %s side", side)
		}
	}
	// A unit that can aim at nobody from the slot it was placed in is refused
	// here, where the roster is still in front of whoever wrote it, rather than
	// discovered as a turn skipped four thousand times.
	//
	// It runs after the whole roster is enlisted because reach is a fact about
	// the board rather than about a unit: whether a range of three is enough
	// depends on where the other nine are standing.
	//
	// This is necessary and not sufficient, and the shape of the problem is why.
	// Nothing moves, so a unit's reach is fixed at enlistment — but the set of
	// cells worth reaching is not, because it shrinks every time somebody dies.
	// A roster that passes here can still deadlock later, which is what the
	// Stalemate outcome is for.
	for _, unit := range fight.units {
		if fight.canAimAtAnyone(unit) {
			continue
		}
		nearest := fight.nearestTargetable(unit)
		if nearest == 0 {
			return nil, fmt.Errorf("unit %q stands at %s knowing no skill it may aim at anybody on the board",
				unit.ID, unit.Cell)
		}
		return nil, fmt.Errorf("unit %q stands at %s, where no skill it knows can be aimed at anyone: "+
			"its longest range is %d and the nearest unit it may target is %d cells away",
			unit.ID, unit.Cell, fight.longestRange(unit), nearest)
	}
	return fight, nil
}

// canAimAtAnyone reports whether any skill the unit knows has a legal aim on the
// board as it currently stands.
//
// Cooldowns are deliberately not read. This asks what a unit can ever do, not
// what it can do this turn, and a cooldown always comes down.
func (b *Battle) canAimAtAnyone(unit *Unit) bool {
	for _, id := range unit.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			continue
		}
		if len(b.aims(unit, known)) > 0 {
			return true
		}
	}
	return false
}

// longestRange is the furthest any skill the unit knows can be pointed. It is
// only ever used to explain a refusal.
func (b *Battle) longestRange(unit *Unit) int {
	longest := 0
	for _, id := range unit.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			continue
		}
		if known.Range > longest {
			longest = known.Range
		}
	}
	return longest
}

// nearestTargetable is the distance to the closest unit that some skill this one
// knows is allowed to point at, with range ignored: the number an author has to
// compare a range against.
//
// The allowance is read rather than the distance alone, because a kit aimed only
// at the enemy is not helped by an ally standing beside it, and a refusal
// quoting that ally's distance would send the author looking in the wrong
// direction. Zero means there is nobody on the board any of its skills may
// target at any range, which is a different fault and says so.
//
// It is only ever used to explain a refusal.
func (b *Battle) nearestTargetable(unit *Unit) int {
	nearest := 0
	for _, id := range unit.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			continue
		}
		for _, other := range b.units {
			if other == unit || other.Dead {
				continue
			}
			if !known.Target.Reaches(unit.Side, other.Cell.Side()) {
				continue
			}
			if distance := unit.Cell.DistanceTo(other.Cell); nearest == 0 || distance < nearest {
				nearest = distance
			}
		}
	}
	return nearest
}

func (b *Battle) enlist(entry Roster, perSide map[hex.Side]int, occupied map[hex.Offset]string) (*Unit, error) {
	if entry.ID == "" {
		return nil, fmt.Errorf("a unit needs an id")
	}
	if _, clash := b.byID[entry.ID]; clash {
		return nil, fmt.Errorf("unit %q is listed twice", entry.ID)
	}
	// A roster entry that never said which half it fights on is refused here
	// rather than placed as an ally by default. hex.SideNone is the zero value,
	// so this is the field a caller is likeliest to have left out, and defaulting
	// it would put a unit on a side nobody chose.
	if !entry.Side.Fights() {
		return nil, fmt.Errorf("unit %q is on no side", entry.ID)
	}
	if entry.Slot.Col < 0 || entry.Slot.Col >= hex.FormationCols ||
		entry.Slot.Row < 0 || entry.Slot.Row >= hex.Rows {
		return nil, fmt.Errorf("unit %q sits at %s, which is not a formation slot", entry.ID, entry.Slot)
	}
	perSide[entry.Side]++
	if perSide[entry.Side] > hex.MaxTeamSize {
		return nil, fmt.Errorf("the %s side has more than %d units", entry.Side, hex.MaxTeamSize)
	}
	cell := hex.Place(entry.Side, entry.Slot)
	if holder, taken := occupied[cell]; taken {
		return nil, fmt.Errorf("unit %q and unit %q both sit at %s", entry.ID, holder, cell)
	}
	occupied[cell] = entry.ID

	if err := b.books.Chart.ValidateAffinity(entry.Affinity); err != nil {
		return nil, fmt.Errorf("unit %q: %w", entry.ID, err)
	}
	if err := b.books.Limits.CheckValues(entry.Stats, b.books.Rules); err != nil {
		return nil, fmt.Errorf("unit %q: %w", entry.ID, err)
	}
	if len(entry.Skills) == 0 {
		return nil, fmt.Errorf("unit %q has no skills", entry.ID)
	}
	seen := make(map[string]bool, len(entry.Skills))
	for _, id := range entry.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			return nil, fmt.Errorf("unit %q: %w", entry.ID, err)
		}
		if seen[known.ID] {
			return nil, fmt.Errorf("unit %q knows %q twice", entry.ID, known.ID)
		}
		seen[known.ID] = true
		// A unit may only carry a skill of an element it shares, or a neutral
		// one, and it must satisfy whatever element allowlist the skill
		// declares. The rule is declared once, in skill.WhyCannotCarry, so that
		// the layer where a character is authored refuses exactly what this
		// refuses.
		//
		// A skill's archetype and character allowlists are not checked here and
		// cannot be: a roster entry carries no archetype and no character
		// identity, because both are resolved before a battle starts. Those two
		// are authoring-time rules, enforced by cast.ParseBook. Adding either to
		// a roster entry to "complete" the rule would put a fact in the
		// replayable core that no replay reads.
		switch skill.WhyCannotCarry(entry.Affinity, known) {
		case skill.CarryWrongElement:
			return nil, fmt.Errorf("unit %q is %s but knows the %s skill %q",
				entry.ID, entry.Affinity, known.Element, known.ID)
		case skill.CarryElementRestricted:
			return nil, fmt.Errorf("unit %q is %s but knows %q, which only %s may carry",
				entry.ID, entry.Affinity, known.ID,
				strings.Join(known.Restrict.ElementNames(), " or "))
		}
	}

	unit := &Unit{
		ID: entry.ID, Name: entry.Name, Side: entry.Side, Cell: cell,
		Affinity: entry.Affinity, Base: entry.Stats, HP: entry.Stats[progression.HP],
		Skills: append([]string(nil), entry.Skills...), Cooldowns: make([]int, len(entry.Skills)),
	}
	if unit.Name == "" {
		unit.Name = unit.ID
	}
	if err := b.grant(unit, entry.Passives); err != nil {
		return nil, err
	}
	// The buffed speed, and the traits are on by the line above. That ordering is
	// the whole of this: a wait is 1_000_000/speed, so a trait touching speed has
	// to be in force before the first one is computed, or turn one is already
	// wrong for the rest of the battle. retuneAll exists because exactly this was
	// got wrong once with haste, and doing it here rather than retuning after the
	// fact means there is no wrong wait to correct and no SpeedChanged event
	// claiming a change that was true from the start.
	if err := b.queue.Add(unit.ID, b.Stats(unit)[progression.Speed]); err != nil {
		return nil, err
	}
	return unit, nil
}

// grant puts a unit's traits on it, before it has a place in the queue.
//
// The statuses are applied rather than the modifiers being read directly, which
// is the point of declaring a passive as statuses at all: the terms belong to the
// status, modifier.Set saturates every term of the same target together, and a
// trait therefore saturates *alongside* a temporary buff rather than composing
// with it. A passive that composed would be the one place in this game where
// stacking explodes.
//
// Nothing is emitted here. A battle has no log until Begin records the opening
// board, and an event describing a unit that has not been introduced yet is a
// log a renderer cannot draw — so the traits take effect here and are reported
// there, which is also the order a reader wants them in.
func (b *Battle) grant(unit *Unit, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if b.books.Passives == nil {
		return fmt.Errorf("unit %q holds passives, which needs the passive book", unit.ID)
	}
	for _, id := range ids {
		held, err := b.books.Passives.Lookup(id)
		if err != nil {
			return fmt.Errorf("unit %q: %w", unit.ID, err)
		}
		if slices.Contains(unit.Passives, held.ID) {
			return fmt.Errorf("unit %q holds the passive %q twice", unit.ID, held.ID)
		}
		unit.Passives = append(unit.Passives, held.ID)
		for _, grant := range held.Grants {
			kind, err := b.books.Statuses.Lookup(grant.Status)
			if err != nil {
				return fmt.Errorf("unit %q: passive %q: %w", unit.ID, held.ID, err)
			}
			for range grant.Stacks {
				// A tick amount of nought: a permanent status cannot be a
				// damage-over-time or a regeneration, which the status book
				// refuses, so there is nothing here to snapshot.
				unit.Statuses.Apply(kind, 0)
			}
		}
	}
	return nil
}

// Begin records the opening board. It takes no turn.
func (b *Battle) Begin() {
	for _, unit := range b.units {
		b.emit(Event{
			Kind: Started, Actor: unit.ID, Name: unit.Name, Side: unit.Side,
			Cell: unit.Cell, Amount: unit.HP, Note: unit.Affinity.String(),
		})
		// Each unit's traits directly after the unit itself, because that is the
		// order they read in and because a trait naming a unit the log has not
		// introduced is a line a renderer cannot place. They were put on in
		// enlist; this says so. Without it a reader sees a permanent buff on the
		// board with nothing anywhere to account for it — the same trap a silent
		// passive changing a damage figure would set.
		for _, id := range unit.Passives {
			held, err := b.books.Passives.Lookup(id)
			if err != nil {
				continue
			}
			for _, grant := range held.Grants {
				b.emit(Event{
					Kind: PassiveHeld, Actor: unit.ID, Passive: held.ID,
					Status: grant.Status, Stacks: grant.Stacks,
				})
			}
		}
	}
}

func (b *Battle) emit(event Event) {
	if event.At == 0 {
		event.At = b.queue.Now()
	}
	b.events = append(b.events, event)
}

// Drain returns the events recorded since the last call and clears them.
func (b *Battle) Drain() []Event {
	out := b.events
	b.events = nil
	return out
}

// Finished reports whether the battle is over.
func (b *Battle) Finished() bool { return b.finished }

// Winner reports which side won, and whether anyone did.
func (b *Battle) Winner() (hex.Side, bool) { return b.winner, b.decided }

// Outcome reports how the battle ended, and Undecided while it is still being
// fought. It is what separates the three ways a battle can finish without a
// winner from one another, which Winner alone cannot say.
func (b *Battle) Outcome() Outcome { return b.outcome }

// Unit returns a combatant by id.
func (b *Battle) Unit(id string) (*Unit, bool) {
	unit, ok := b.byID[id]
	return unit, ok
}

// Units returns every combatant, living and dead, in roster order.
func (b *Battle) Units() []*Unit { return b.units }

// Queue exposes the turn order, for a display that wants to show what is coming.
func (b *Battle) Queue() *atb.Queue { return b.queue }

// Books returns the data the battle was built from, for a client that needs to
// render a skill's declaration alongside the events.
func (b *Battle) Books() Books { return b.books }

// Pending returns the turn waiting for an action, or nil when the battle is
// between turns. It exists so a caller that lost track of an open turn can pick
// it up rather than advancing past it.
func (b *Battle) Pending() *Prompt {
	if !b.awaiting {
		return nil
	}
	return b.prompt
}

// Stats resolves a unit's current stat line: its base, with every active status's
// modifier terms applied.
func (b *Battle) Stats(unit *Unit) progression.Values {
	terms := unit.Statuses.Modifiers()
	return terms.Stats(unit.Base, b.books.Limits.Ceilings, b.books.Bounds)
}

func (b *Battle) living(side hex.Side) int {
	count := 0
	for _, unit := range b.units {
		if unit.Side == side && !unit.Dead {
			count++
		}
	}
	return count
}

func (b *Battle) occupant(cell hex.Offset) *Unit {
	for _, unit := range b.units {
		if !unit.Dead && unit.Cell == cell {
			return unit
		}
	}
	return nil
}

// checkEnd ends the battle when a side has been emptied. It is called the moment
// a unit falls, including in the middle of a skill still resolving, so it asks
// only the question that is safe to ask there: is anyone left.
func (b *Battle) checkEnd() {
	if b.finished {
		return
	}
	allies, enemies := b.living(hex.SideAlly), b.living(hex.SideEnemy)
	if allies > 0 && enemies > 0 {
		return
	}
	switch {
	case allies > 0:
		b.winner, b.decided, b.outcome = hex.SideAlly, true, Victory
	case enemies > 0:
		b.winner, b.decided, b.outcome = hex.SideEnemy, true, Victory
	default:
		b.outcome = Annihilation
	}
	b.end()
}

// settle is checkEnd plus the deadlock test, and runs when a turn has finished
// and the board is at rest.
//
// The two are separate because they may be asked at different moments. A side
// being emptied is true as soon as the last unit falls, wherever that happens;
// a deadlock is a statement about what can happen next, and asking it halfway
// through a skill that is still choosing targets would be reading a board that
// is not the board anyone will act from.
func (b *Battle) settle() {
	b.checkEnd()
	if b.finished || !b.frozen() {
		return
	}
	b.outcome = Stalemate
	b.end()
}

// end records the outcome the caller has already decided on.
func (b *Battle) end() {
	b.finished = true
	b.emit(Event{Kind: Ended, Side: b.winner, Outcome: b.outcome})
}

// frozen reports whether the battle can never change again: nobody can act, and
// nothing pending would give anyone something to do.
//
// It is a pure function of the state rather than a count of quiet turns, which
// is what a replay needs — a battle that draws on one machine has to draw on
// every other from the same seed, and a counter is one more thing two runs could
// disagree about.
//
// Two things have to be true of every living unit, and both are asked the
// pessimistic way round: the predicate has to be certain, because declaring a
// draw on a battle that would have resolved is worse than letting the turn limit
// catch a real one.
//
//   - Nothing timed is on it. A poisoned deadlock is not a deadlock: the poison
//     will kill somebody, and that ends the battle by emptying a side. So will a
//     stun wearing off, a debuff expiring or a shield running out. Anything with
//     a duration left to spend is a promise that the board is not final.
//   - No skill it knows has a legal aim, cooldowns ignored. A cooldown always
//     comes down, so a skill cooling down is not a reason a unit cannot act; a
//     skill with nothing in range is. Note that a self-targeting skill always has
//     an aim, so a unit that can still buff itself is not frozen — and it will
//     be holding a timed status the moment it does, which is the first clause
//     agreeing with the second.
//
// Cooldowns and control are what make a skipped turn ordinary. Neither is read
// here, which is exactly why an ordinary skipped turn cannot be mistaken for
// this.
func (b *Battle) frozen() bool {
	for _, unit := range b.units {
		if unit.Dead {
			continue
		}
		if unit.Statuses.Timed() {
			return false
		}
		if b.canAimAtAnyone(unit) {
			return false
		}
	}
	return true
}

func (b *Battle) kill(unit *Unit) {
	if unit.Dead {
		return
	}
	unit.Dead = true
	unit.HP = 0
	b.queue.Remove(unit.ID)
	b.emit(Event{Kind: Died, Actor: unit.ID, Cell: unit.Cell, Side: unit.Side})
	b.checkEnd()
}
