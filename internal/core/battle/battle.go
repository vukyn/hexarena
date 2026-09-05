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
// reads the events; nothing about drawing a battle needs to read the state
// here, which is the property that keeps a renderer from quietly becoming a
// second copy of the rules.
//
// That log is append-only and is kept for the battle's whole life, so a battle
// has as many consumers as anybody wants: each holds a cursor of its own (see
// Battle.Since), and Drain is one such consumer with its cursor kept by the
// battle. Two players, a spectator who joined halfway and the log writer are
// four cursors over one record rather than four copies of a battle.
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

// absorbCategory is the guard whose pool a strike is eaten by. Named as a
// CATEGORY rather than as an id, unlike the block above, and the difference is
// not an inconsistency: block charges are counted, and a count belongs to one
// kind, while a pool is a quantity that arriving damage does not sort by name.
// Two kinds of barrier on one unit are one pool to whatever hits it.
const absorbCategory = status.Absorb

// stunStatus is the control effect that costs a unit its action.
const stunStatus = "stun"

// tauntStatus is the control effect that costs a unit its choice of target.
//
// It sits on the unit doing the taunting rather than on the unit being taunted,
// and that is the design rather than a preference. A taunt held by its victim
// would have to remember *who* taunted it, and a status.Stack deliberately does
// not remember who applied it — which is what keeps a stack worth the same after
// its author has died. Held by the taunter it needs no such memory: "who must I
// attack" is answered by looking at the board, and a taunter who dies takes the
// answer with it on the same turn, with nothing to clean up.
const tauntStatus = "taunting"

// burrowStatus is the status that takes a unit off the list of things anything
// else may aim at, and refuses every status somebody else tries to put on it.
//
// # Why the engine knows this one by name
//
// It is the second status the engine reads by id rather than by category, after
// the taunt above, and for the same reason: both change *who may be aimed at*,
// which is a question `aims` answers before any category is consulted. A taunt
// narrows the list to the units holding it; this removes the units holding it
// from the list. They are mirror images and they belong in one place.
//
// # What it does NOT stop
//
// Splash. `aims` is the cells a skill may be pointed at and `covers` is the
// cells the shape then catches, and only the first is touched — so a unit
// underground is caught by a blast aimed at somebody standing next to it, which
// is the whole of what makes the hiding a decision rather than a wall. It also
// does not stop the holder acting, healing itself or being finished off by a
// status already on it when it went under.
//
// ⚠️ **A board where every survivor is hidden ends as a Stalemate rather than
// hanging.** Battle.settle asks whether anybody can act at all and calls it when
// nobody can, so a hiding that leaves the other side nothing to aim at is a draw
// and not a hung turn limit. That is why this needs no rule of its own about
// being the last one standing.
const burrowStatus = "burrowed"

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
// The tags are here because a roster is written to a log, and a log is a file
// somebody reads. Without them the placement would come out in Go's field names
// while every other file this repository writes is snake case — the same record
// spelled two ways depending on which layer produced it.
type Roster struct {
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
	Side hex.Side `json:"side"`
	// Slot is the formation slot the unit occupies, authored from its own side's
	// point of view. hex.Place maps it onto the shared board.
	Slot     hex.Offset         `json:"slot"`
	Affinity element.Affinity   `json:"element"`
	Stats    progression.Values `json:"stats"`
	Skills   []string           `json:"skills"`
	// Passives are the traits the unit holds, by id in the passive book.
	//
	// This is the first thing on a roster entry that is neither a stat nor a
	// skill, and it belongs here for the reason the archetype and the evolution
	// stage do not: those two are settled *before* a battle and leave nothing
	// behind but numbers, while a passive is in force during one and a replay has
	// to know about it. The test of what belongs on a Roster has always been
	// "does a replay read it", not "is it small".
	Passives []string `json:"passives,omitempty"`
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
	// Summoner is the unit that put this one on the board, and is empty for one
	// the roster placed. Summoned counts what this unit has put down, so two
	// casts cannot name their clones the same thing while the first lot is still
	// standing. Leaves is how many of its own turns a summon has left, and is
	// minus one for one that stays; Bound says it goes when its summoner does.
	//
	// All four are read by a replay, which is the test for what belongs on a
	// Unit — but none of them is on Roster, because a summon is derived from the
	// caster and the skill rather than placed. A log carrying them would be a log
	// that could disagree with the engine about who is standing there.
	Summoner string
	Summoned int
	Leaves   int
	Bound    bool
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

	// events is the append-only record of everything this battle has emitted,
	// kept for the battle's whole life, and drained is how far Drain has read
	// into it. A cursor rather than a truncation is what lets a room hand the
	// same battle to two players, a spectator and a log at once — see Since.
	events   []Event
	drained  int
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
	// ⚠️ A reach guard stood here, refusing a unit that could aim at nobody from
	// the slot it was placed in. It is gone because it can no longer fire, not
	// because it stopped mattering.
	//
	// Reach is counted in ranks from the far side now, so a range of one finds
	// the enemy's foremost survivor wherever it stands, and both sides are known
	// to be occupied by the checks above. Every unit holding an offensive skill
	// can therefore always aim at somebody, and one holding none can always aim
	// at itself. No placement is left for the guard to catch, and a branch that
	// cannot be reached is worse than no branch: it reads as a protection
	// somebody is relying on.
	//
	// The half of the problem that survives is not about distance at all — a
	// board can still freeze because no kit holds anything to throw — and that
	// is what the Stalemate outcome now answers for on its own.
	return fight, nil
}

// ⚠️ canAimAtAnyone, longestRange and nearestTargetable were deleted here along
// with the reach guard in New. All three existed only to phrase its refusal —
// "its longest range is 3 and the nearest unit it may target is 5 cells away" —
// and that sentence cannot be true of any board now that reach is counted in
// ranks rather than in cells. They are named here so that a reader looking for
// them learns where they went instead of assuming the file lost them.

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
//
// A gated trait is a different matter: it is off at full health, so enlistment
// puts nothing on and reconsider is what turns it on, mid-battle, with an event
// of its own.
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
		// A gate is read here rather than assumed open, and at enlistment a unit
		// is at full health — so a trait gated on being hurt starts off, and
		// turns on the first time it is. Checking the gate rather than
		// special-casing enlistment is what keeps one rule: a trait is on when
		// its condition holds, from the first moment to the last.
		if !b.inForce(unit, held) {
			continue
		}
		if err := b.hold(unit, held); err != nil {
			return err
		}
	}
	return nil
}

// hold puts one trait's grants on its holder.
//
// Hold rather than Apply, and the difference is the gate: a permanent status is
// refused by Remove so that nothing in the game can dispel a trait, which leaves
// the trait's own gate as the only thing that may take it back. Hold and Release
// are that door, and they work on nothing else.
func (b *Battle) hold(unit *Unit, held passive.Passive) error {
	for _, grant := range held.Grants {
		kind, err := b.books.Statuses.Lookup(grant.Status)
		if err != nil {
			return fmt.Errorf("unit %q: passive %q: %w", unit.ID, held.ID, err)
		}
		unit.Statuses.Hold(kind, b.granted(unit, grant), grant.Stacks)
	}
	return nil
}

// granted is how big a trait's grant is, and nought for every grant that carries
// no quantity — which is all of them but a guard holding a pool.
//
// It reads the holder's **base** line rather than its buffed one. Grants go on
// one trait after another at enlistment, so a buffed reading would make the
// answer depend on which trait was applied first: a unit with endurance and a
// defence-scaled barrier would get a different barrier in one declaration order
// than in another, with nothing on either trait saying so. Base is the only
// reading that is a fact about the unit rather than about the loop.
//
// Through Rules.Restore, which is the same expression a regeneration and an
// inflicted barrier are built from, so a granted guard and a cast one cannot
// disagree about what a share of a stat comes to.
func (b *Battle) granted(unit *Unit, grant passive.Grant) int64 {
	if grant.Power <= 0 {
		return 0
	}
	return b.books.Rules.Restore(unit.Base[grant.Scaling], grant.Power)
}

// Begin records the opening board. It takes no turn.
func (b *Battle) Begin() {
	for _, unit := range b.units {
		b.emit(Event{
			Kind: Started, Actor: unit.ID, Name: unit.Name, Side: unit.Side,
			Cell: hex.At(unit.Cell), Amount: unit.HP, Note: unit.Affinity.String(),
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
			// Only what is actually on the unit. A gated trait is off at full
			// health, and announcing a grant the opening board does not show
			// would be the log describing a different unit from the one being
			// drawn beside it.
			if !b.inForce(unit, held) {
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

// Drain returns the events recorded since the last call.
//
// It is one consumer of the record — the battle's own — holding a cursor the
// battle keeps for it, which is why it may sit beside any number of Since
// readers without disturbing them or being disturbed. It no longer empties
// anything, and that is the only thing about it that changed: it still answers
// exactly the events since the last call, and still answers **nil** when there
// are none.
//
// ⚠️ The nil is kept because keeping it is free, and **not** because anything
// reads it. Measured: returning an empty non-nil slice instead reddens three
// tests, all three of them new here, and **none** of the several hundred
// existing Drain callers — they all take a length or a range. So the rule is
// held by its own tests and by nothing else, which is worth knowing before
// somebody decides it is load-bearing and builds on it.
func (b *Battle) Drain() []Event {
	out, next := b.Since(b.drained)
	b.drained = next
	return out
}

// Since returns the events recorded from cursor onward, and the cursor to pass
// next time. A consumer holds its own cursor and nothing else, so a room's two
// players, its spectators and its log each read the whole battle at their own
// pace: Since(0) is a consumer that wants everything, which is what a spectator
// joining a battle already in progress is.
//
// It panics on a cursor the record cannot answer, and that is deliberate rather
// than defensive. Answering an out-of-range cursor with an empty slice would
// make a consumer that has somehow got ahead of the battle look exactly like one
// that is up to date, which is the silent desync a cursor exists to prevent; and
// a cursor is a number this method handed the caller itself, so a bad one is a
// programming error rather than a runtime condition — the same reading rng.Intn
// takes of a non-positive bound.
func (b *Battle) Since(cursor int) ([]Event, int) {
	recorded := len(b.events)
	if cursor < 0 || cursor > recorded {
		panic(fmt.Sprintf("battle: Since called with a cursor of %d against a record of %d events", cursor, recorded))
	}
	if cursor == recorded {
		return nil, recorded
	}
	// Three-index, so the view's capacity is its length and a caller's own
	// append has to reallocate. Sharing the record's spare capacity corrupts
	// BOTH sides: the caller's append writes into the slot the next emit is
	// going to use, and that emit then overwrites what the caller appended.
	// Measured in TestAViewAndTheRecordSurviveEachOthersAppends; a copy would
	// answer as well and cost more.
	return b.events[cursor:recorded:recorded], recorded
}

// Recorded reports how many events the record holds, which is the cursor a
// consumer joining now would start from if it wanted only what happens next.
func (b *Battle) Recorded() int { return len(b.events) }

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
//   - Nothing timed is on it **that could change how the battle ends**. A poisoned
//     deadlock is not a deadlock: the poison will kill somebody, and that ends the
//     battle by emptying a side. A taunt is the other one — it narrows who a unit
//     may aim at, so a unit with no legal aim today may have one the moment the
//     taunt expires.
//
//     ⚠️ Everything else timed is deliberately *not* a reason to keep the board
//     open, and reading "anything with a duration" was a real hole rather than a
//     conservative choice. A regeneration, a buff and a shield cannot kill anybody
//     and cannot make anybody reachable, so a unit refreshing its own regeneration
//     every turn held a frozen board open **for ever**: the draw was never
//     declared, and the battle ran the four-thousand-turn limit out instead of
//     ending. That is what a draft roster using slot `1,2` hit in 5 seeds of 4000 —
//     two survivors past every range in the cast, one of them tending its own
//     garden. It is not even reported as a draw, which is the part that matters: a
//     replay of it says nothing at all about what happened.
//
//     A stat debuff and a control effect are in the same class: neither changes
//     what a unit can aim at, and a stunned unit is already "not frozen" by the
//     clause below, because an aim it cannot take this turn is still an aim. A
//     taunt is the same, for a reason worth knowing — see outcomeChanging.
//
//   - No skill it knows can be aimed at an **enemy**, cooldowns ignored. A cooldown
//     always comes down, so a skill cooling down is not a reason a unit cannot act;
//     a skill with nothing in range is.
//
//     ⚠️ "At an enemy" rather than "at anyone", and that was the other half of the
//     same hole. A self-targeting skill *always* has an aim — its caster — so a
//     unit holding one could never be frozen, whatever else was true of the board.
//     Two survivors past every range in the cast, each able to tend its own
//     garden, therefore never drew: clause one saw a regeneration ticking and
//     clause two saw a legal aim, and each was agreeing with the other about the
//     wrong thing. Buffing yourself for ever is not something happening; it is the
//     shape a deadlock takes when the units in it have support skills.
//
//     An ally-aimed skill is out for the same reason and one more: the only way
//     healing an ally changes an outcome is by outlasting something that is
//     killing it, and that something is a damage-over-time, which the clause above
//     already keeps the board open for.
//
// Cooldowns and control are what make a skipped turn ordinary. Neither is read
// here, which is exactly why an ordinary skipped turn cannot be mistaken for
// this.
// outcomeChanging is the timed effects that can still decide a battle nobody can
// act in, and there is exactly one: damage over time, because it kills, and a kill
// empties a side.
//
// A taunt looks like it belongs here — it decides who may be aimed at, so a unit
// with no legal aim while one is in force has an aim again when it expires — and it
// is deliberately absent, because the clause below already covers it: `aims` offers
// a taunted unit its taunter **whether or not the taunter is in reach**, so a
// taunted unit always has an aim and is never frozen. A second entry saying the
// same thing would be a claim no test could reach, and this file has no room for
// one.
var outcomeChanging = []status.Category{status.Dot}

func (b *Battle) frozen() bool {
	for _, unit := range b.units {
		if unit.Dead {
			continue
		}
		if unit.Statuses.TimedIn(outcomeChanging) {
			return false
		}
		if b.canAimAtAnEnemy(unit) {
			return false
		}
	}
	return true
}

// canAimAtAnEnemy is canAimAtAnyone restricted to the aims that can change how a
// battle ends: an enemy-aimed skill, or an all-sided one, with somebody in reach.
//
// It is a second function rather than a flag on the first because the two callers
// are asking different questions. Placement asks "can this unit ever act", where a
// support-only unit is a legal thing to field; the deadlock predicate asks "can
// anything still change", where it is not.
func (b *Battle) canAimAtAnEnemy(unit *Unit) bool {
	for _, id := range unit.Skills {
		known, err := b.books.Skills.Lookup(id)
		if err != nil {
			continue
		}
		// A summoning skill counts wherever it is aimed, and that is the third
		// thing that can change an outcome from a self-aimed cast: it puts a unit
		// on the board that the board did not have, and the new one may reach what
		// its summoner cannot. Leaving it out made a caster holding nothing but a
		// summon read as deadlocked the moment its escort fell.
		if known.Summons.Summons() {
			return true
		}
		if known.Target == skill.Self || known.Target == skill.Ally {
			continue
		}
		if len(b.aims(unit, known)) > 0 {
			return true
		}
	}
	return false
}

func (b *Battle) kill(unit *Unit) {
	if unit.Dead {
		return
	}
	unit.Dead = true
	unit.HP = 0
	b.queue.Remove(unit.ID)
	b.emit(Event{Kind: Died, Actor: unit.ID, Cell: hex.At(unit.Cell), Side: unit.Side})
	// Before the end is checked, not after. A bound summon going with its
	// summoner can be what empties a side, and checking first would declare a
	// battle still running that the very next line ends — two answers to one
	// question, in the wrong order.
	b.dismissBound(unit)
	b.checkEnd()
}
