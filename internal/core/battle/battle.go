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

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
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
	awaiting bool
	finished bool
	winner   hex.Side
	decided  bool
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
	return fight, nil
}

func (b *Battle) enlist(entry Roster, perSide map[hex.Side]int, occupied map[hex.Offset]string) (*Unit, error) {
	if entry.ID == "" {
		return nil, fmt.Errorf("a unit needs an id")
	}
	if _, clash := b.byID[entry.ID]; clash {
		return nil, fmt.Errorf("unit %q is listed twice", entry.ID)
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
		// one. That constraint is what makes a second element worth having:
		// it buys a second line of skills rather than a better multiplier.
		if known.Element != element.Neutral && !entry.Affinity.Has(known.Element) {
			return nil, fmt.Errorf("unit %q is %s but knows the %s skill %q",
				entry.ID, entry.Affinity, known.Element, known.ID)
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
	if err := b.queue.Add(unit.ID, entry.Stats[progression.Speed]); err != nil {
		return nil, err
	}
	return unit, nil
}

// Begin records the opening board. It takes no turn.
func (b *Battle) Begin() {
	for _, unit := range b.units {
		b.emit(Event{
			Kind: Started, Actor: unit.ID, Side: unit.Side, Cell: unit.Cell,
			Amount: unit.HP, Note: unit.Affinity.String(),
		})
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

func (b *Battle) checkEnd() {
	if b.finished {
		return
	}
	allies, enemies := b.living(hex.SideAlly), b.living(hex.SideEnemy)
	if allies > 0 && enemies > 0 {
		return
	}
	b.finished = true
	switch {
	case allies > 0:
		b.winner, b.decided = hex.SideAlly, true
	case enemies > 0:
		b.winner, b.decided = hex.SideEnemy, true
	}
	note := "draw"
	if b.decided {
		note = b.winner.String()
	}
	b.emit(Event{Kind: Ended, Side: b.winner, Note: note})
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
