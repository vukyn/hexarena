// Package composition prices what a squad shares.
//
// A composition bonus is a grant a side receives for fielding several units
// that have something in common — today one thing, the element they carry — and
// it is a **drafting** decision rather than a tactic: the count is taken once,
// from the roster a battle opens with, and never again. Nothing that happens on
// the board can take one away or hand one over, so focusing the odd unit out
// does not strip a bonus and a summon does not earn one.
//
// The package is a pure function of what it is handed, like every core package
// but battle. It counts, it decides which rung a count reaches, and it says who
// receives what. Applying the grant is the caller's, because a status set is
// state and this has none.
//
// ⚠️ **The count is walked in slice order and never over a map.** Ordering a
// result by ranging over a map is the one thing this repository's layer rule
// names outright: Go randomises that order and a battle would stop replaying
// from its seed. Values are collected in the order they are first met, which is
// a fact about the roster rather than about a hash.
package composition

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
)

// Axis is the thing a bonus counts.
//
// It is an enum rather than a string so an unknown axis is refused at parse
// instead of counting nothing at run time — a bonus whose axis nobody implements
// would otherwise load, draw, and quietly never fire.
type Axis int

const (
	// AxisNone is the zero value and is never a legal axis, so a bonus that
	// forgot to declare one is refused rather than defaulting into whichever
	// axis happens to be first.
	AxisNone Axis = iota
	// AxisElement counts the elements the fielded units carry.
	AxisElement
	// AxisColumn counts the formation column the fielded units stand in.
	//
	// ⚠️ **It is where a unit STANDS, not the column its archetype prefers**, and
	// that distinction decided the whole shape of this axis. The roadmap named the
	// archetype's column as the best-shaped candidate on reachability alone; the
	// engine cannot see it. Roster carries no archetype on purpose — "settled
	// before a battle and leaves nothing behind but numbers" — so counting the
	// preset would have meant widening the roster, the wire and every log, to say
	// something weaker than what the slot already says.
	//
	// The slot is the better half of the same idea anyway. An archetype's column
	// is where a character *wants* to stand; the slot is where the player *put*
	// it, which is the decision a composition bonus is supposed to reward.
	//
	// ⚠️ **Rungs above three are unreachable here and always will be.** A column
	// holds hex.FormationRows units and that is a property of the board rather
	// than of the format, so this is not the "wait for 5v5" case the element axis
	// has: a fourth rung on a column would be dead on every board this game has.
	AxisColumn
)

// String is the name an axis is declared under.
func (a Axis) String() string {
	switch a {
	case AxisElement:
		return "element"
	case AxisColumn:
		return "column"
	}
	return "none"
}

// ParseAxis reads an axis by name.
func ParseAxis(name string) (Axis, error) {
	switch name {
	case "element":
		return AxisElement, nil
	case "column":
		return AxisColumn, nil
	}
	return AxisNone, fmt.Errorf("no axis is called %q", name)
}

// Scope is who receives a bonus that has fired.
//
// Both kinds ship, and a bonus declares which it is. They answer different
// questions: a squad-wide grant says the side is better arranged, a
// sharers-only grant says these units in particular back one another up, and a
// screen has to be able to tell a reader which of the two it is looking at.
type Scope int

const (
	// ScopeNone is the zero value, refused for the reason AxisNone is.
	ScopeNone Scope = iota
	// ScopeSquad hands the grant to every unit on the side.
	ScopeSquad
	// ScopeSharers hands it only to the units that carry the counted value.
	ScopeSharers
)

// String is the name a scope is declared under.
func (s Scope) String() string {
	switch s {
	case ScopeSquad:
		return "squad"
	case ScopeSharers:
		return "sharers"
	}
	return "none"
}

// ParseScope reads a scope by name.
func ParseScope(name string) (Scope, error) {
	switch name {
	case "squad":
		return ScopeSquad, nil
	case "sharers":
		return ScopeSharers, nil
	}
	return ScopeNone, fmt.Errorf("no scope is called %q", name)
}

// MinimumRung is the smallest count that can be a threshold.
//
// Two, because one unit shares nothing with anybody: a rung of one would fire
// for every squad in the game and is a change to the base numbers wearing a
// threshold, which is the trap the origin axis is held out of the data for.
const MinimumRung = 2

// Grant is one status a rung hands out.
//
// It is deliberately the same shape a trait's grant is, minus the quantity: a
// trait can grant a pool whose depth is a share of its holder's stat line, and
// a composition bonus may not, because the units receiving one do not share a
// stat line. A bonus that wants to be worth more says so with another stack or
// another status.
type Grant struct {
	Status string
	Stacks int
}

// Rung is one threshold and what reaching it gives.
type Rung struct {
	// At is how many units must share the value.
	At int
	// Grants is what each receiving unit is held. Never empty: a rung that
	// grants nothing is a row in a table that a screen would draw and a battle
	// would ignore.
	Grants []Grant
}

// Bonus is one rule about what a squad shares.
type Bonus struct {
	ID string
	// Name is the authored display name, opaque text to this package exactly as
	// a skill's and a trait's are. Absent is a real answer and renders as the id.
	Name  string
	Axis  Axis
	Scope Scope
	// Value is the one value on the axis this bonus counts, or empty for all of
	// them.
	//
	// ⚠️ **Empty is a real answer and it is the older one.** `same_element` counts
	// whichever element a side happens to share and hands out the same grant for
	// any of them; a bonus naming a value is the narrower shape — "three of WATER
	// heal better" — and the two are different designs rather than one with a
	// default. A table of per-value bonuses says what each tribe is FOR, where the
	// unnamed form says only that sharing is worth something.
	//
	// It is checked against the axis at parse: an element that is not on the chart
	// is a bonus that could never fire, and a bonus that can never fire loads,
	// draws and is silently dead — the one failure this package's Axis enum exists
	// to prevent.
	Value string
	// Rungs are ascending by At, with no two the same. A count reaches at most
	// one of them — the highest it satisfies — because rungs are a ladder rather
	// than a set: three units sharing an element are not also two units sharing
	// it a second time, and a cumulative reading would make the top rung's real
	// value the sum of a table nobody wrote down.
	Rungs []Rung
}

// Reached is the rung a count arrives at, and whether it arrived at one.
func (b Bonus) Reached(count int) (Rung, bool) {
	best, found := Rung{}, false
	for _, rung := range b.Rungs {
		if count >= rung.At {
			best, found = rung, true
		}
	}
	return best, found
}

// Top is the highest threshold a bonus declares, which is what a reference
// screen measures its column against.
func (b Bonus) Top() int {
	top := 0
	for _, rung := range b.Rungs {
		top = max(top, rung.At)
	}
	return top
}

// Member is one fielded unit, as much of it as counting needs.
//
// An id, an affinity and a column, and nothing else: a bonus is settled before
// the first turn, where the roster is still a slice of facts rather than a board,
// and handing this package a battle unit would tie a counting rule to a state
// machine it must never read.
type Member struct {
	ID       string
	Affinity element.Affinity
	// Column is the formation column the unit was placed in, from its own side's
	// point of view — the authored slot rather than the cell on the shared board,
	// so the two sides count the same shape as the same shape.
	Column int
}

// Award is one unit's share of one bonus that fired.
//
// It carries the value and the count as well as the grants, because a log line
// and a screen both have to be able to say *why* a unit is carrying something —
// "three of water" is the whole explanation, and a grant on its own is a buff
// with no account of where it came from.
type Award struct {
	Unit   string
	Bonus  string
	Value  string
	Count  int
	Scope  Scope
	Grants []Grant
}

// Book is the declared bonuses, in declaration order.
type Book struct {
	bonuses []Bonus
}

// All is every bonus, in the order it was declared.
func (b *Book) All() []Bonus {
	if b == nil {
		return nil
	}
	return slices.Clone(b.bonuses)
}

// Lookup finds a bonus by id.
func (b *Book) Lookup(id string) (Bonus, error) {
	if b != nil {
		for _, held := range b.bonuses {
			if held.ID == id {
				return held, nil
			}
		}
	}
	return Bonus{}, fmt.Errorf("no bonus is called %q", id)
}

// Without is the same book with the named bonuses taken out, and it is the
// pricing instrument rather than a convenience.
//
// Nothing about a bonus can be measured until it can be turned **off**: the two
// obvious controls are both already known to measure something else — swapping
// a member measures the member, and putting the same bonus on both sides
// cancels it exactly. What is left is the same squad, the same members and the
// same seeds with one bonus gone, which is what this hands a caller.
//
// ⚠️ **A set rather than a switch.** Bonuses stack, so a global off measures
// *the system* and can never price one rung; taking one out leaves the board the
// other bonuses actually make, which is the board the rung is played on.
//
// An id nobody declares is not an error. A caller naming one is asking for a
// book without it, and a book without it is what comes back.
func (b *Book) Without(ids ...string) *Book {
	if b == nil {
		return nil
	}
	kept := make([]Bonus, 0, len(b.bonuses))
	for _, held := range b.bonuses {
		if slices.Contains(ids, held.ID) {
			continue
		}
		kept = append(kept, held)
	}
	return &Book{bonuses: kept}
}

// Awards is what one side receives for what it brought.
//
// The members are one side's units in roster order, and every list this returns
// is ordered by that slice and by the book's own declaration order — never by a
// map walk. A nil book awards nothing, which is what lets a battle run without
// the file.
//
// ⚠️ **An inert element forms no tribe.** Sharing the element that has no
// strengths and no weaknesses is sharing the absence of one, and a bonus for it
// would be handed to any pair of unaligned characters for standing next to each
// other. Which elements those are is read off the chart rather than named here,
// so a chart that gives the inert element a matchup makes it count without this
// file being edited.
func (b *Book) Awards(chart *element.Chart, members []Member) []Award {
	if b == nil || chart == nil || len(members) == 0 {
		return nil
	}
	byElement := elementTallies(chart, members)
	byColumn := columnTallies(members)
	var awards []Award
	for _, held := range b.bonuses {
		counted := byElement
		if held.Axis == AxisColumn {
			counted = byColumn
		}
		for _, value := range counted {
			// A bonus naming a value ignores every other tally, which is what makes
			// a per-element table possible: eight bonuses walk the same tallies and
			// each one answers for its own tribe.
			if held.Value != "" && value.value != held.Value {
				continue
			}
			rung, reached := held.Reached(value.count)
			if !reached {
				continue
			}
			receiving := value.sharers
			if held.Scope == ScopeSquad {
				receiving = make([]string, 0, len(members))
				for _, member := range members {
					receiving = append(receiving, member.ID)
				}
			}
			for _, unit := range receiving {
				awards = append(awards, Award{
					Unit: unit, Bonus: held.ID, Value: value.value,
					Count: value.count, Scope: held.Scope, Grants: slices.Clone(rung.Grants),
				})
			}
		}
	}
	return awards
}

// tally is one value of one axis: what was shared, how many shared it, and who.
//
// The value is a string because that is what an Award carries and what a log
// line prints, and because the two axes have nothing else in common — an element
// and a column are not the same kind of thing and a shared numeric type would be
// a coincidence rather than a meaning.
type tally struct {
	value   string
	count   int
	sharers []string
}

// counted adds one member's value to a running list, in first-appearance order.
//
// A slice rather than a map because both the order and the membership reach the
// result, which is the rule internal/core is written under: a map's iteration
// order may not decide an output.
func counted(tallies []tally, value, member string) []tally {
	at := slices.IndexFunc(tallies, func(t tally) bool { return t.value == value })
	if at < 0 {
		return append(tallies, tally{value: value, count: 1, sharers: []string{member}})
	}
	tallies[at].count++
	tallies[at].sharers = append(tallies[at].sharers, member)
	return tallies
}

// elementTallies counts the elements a side brought.
//
// ⚠️ **An inert element forms no tribe.** Sharing the element that has no
// strengths and no weaknesses is sharing the absence of one, and a bonus for it
// would be handed to any pair of unaligned characters for standing next to each
// other. Which elements those are is read off the chart rather than named here,
// so a chart that gives the inert element a matchup makes it count without this
// file being edited.
func elementTallies(chart *element.Chart, members []Member) []tally {
	inert := chart.Inert()
	var tallies []tally
	for _, member := range members {
		for _, carried := range member.Affinity.Elements() {
			if slices.Contains(inert, carried) {
				continue
			}
			tallies = counted(tallies, carried.String(), member.ID)
		}
	}
	return tallies
}

// columnTallies counts the formation columns a side stood in.
//
// ⚠️ **There is no inert column.** Every element bonus has to skip the element
// that means nothing, and the mirror of that rule here would be a column that
// means nothing — there is none: standing at the back is as much a decision as
// standing at the front, and a squad that stacks the back rank has made the same
// kind of choice as one that stacks the front. The asymmetry between the two
// axes is real and is why they are two functions rather than one with a flag.
//
// One value per member rather than one per element: a unit stands in exactly one
// column, where it may carry two elements, so the counts here can never exceed
// the squad size and the element ones can.
func columnTallies(members []Member) []tally {
	var tallies []tally
	for _, member := range members {
		tallies = counted(tallies, columnValue(member.Column), member.ID)
	}
	return tallies
}

// columnValue is the name a column is counted and logged under.
//
// Named rather than the bare number, because the value reaches a log line and a
// screen through the same slot an element's name does: "three of 2" is a sentence
// nobody can read, and a reader who cannot tell which axis a bonus counted cannot
// reproduce it from the data.
func columnValue(column int) string { return "column" + strconv.Itoa(column) }

// Deps are the books a bonus is checked against.
//
// The statuses, because a grant names one; the chart, because a rung counts
// elements. Both are required at parse for the reason a skill's pattern and
// status names are: a bonus naming something that does not exist fails at load
// rather than at the moment it would have mattered.
type Deps struct {
	Statuses *status.Book
	Chart    *element.Chart
}

type bonusFile struct {
	ID    string     `json:"id"`
	Name  string     `json:"name,omitempty"`
	Axis  string     `json:"axis"`
	Value string     `json:"value,omitempty"`
	Scope string     `json:"scope"`
	Rungs []rungFile `json:"rungs"`
}

type rungFile struct {
	At     int         `json:"at"`
	Grants []grantFile `json:"grants"`
}

type grantFile struct {
	Status string `json:"status"`
	Stacks int    `json:"stacks"`
}

type file struct {
	Bonuses []bonusFile `json:"bonuses"`
}

// ParseBook reads the declared bonuses and checks every name they use.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if deps.Statuses == nil {
		return nil, fmt.Errorf("bonuses grant statuses, which cannot be checked without the status book")
	}
	if deps.Chart == nil {
		return nil, fmt.Errorf("bonuses count elements, which cannot be checked without the element chart")
	}
	var decoded file
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode bonuses: %w", err)
	}
	book := &Book{bonuses: make([]Bonus, 0, len(decoded.Bonuses))}
	for _, entry := range decoded.Bonuses {
		parsed, err := parseBonus(entry, deps)
		if err != nil {
			return nil, err
		}
		for _, already := range book.bonuses {
			if already.ID == parsed.ID {
				return nil, fmt.Errorf("two bonuses are called %q, so naming one of them chooses neither", parsed.ID)
			}
		}
		book.bonuses = append(book.bonuses, parsed)
	}
	return book, nil
}

func parseBonus(entry bonusFile, deps Deps) (Bonus, error) {
	if strings.TrimSpace(entry.ID) == "" {
		return Bonus{}, fmt.Errorf("a bonus needs an id")
	}
	axis, err := ParseAxis(entry.Axis)
	if err != nil {
		return Bonus{}, fmt.Errorf("bonus %q: %w", entry.ID, err)
	}
	scope, err := ParseScope(entry.Scope)
	if err != nil {
		return Bonus{}, fmt.Errorf("bonus %q: %w", entry.ID, err)
	}
	if len(entry.Rungs) == 0 {
		return Bonus{}, fmt.Errorf("bonus %q declares no rung, so nothing can reach it", entry.ID)
	}
	// A named value is checked against its axis, because a bonus that names a
	// value the axis can never produce loads, draws and never fires — the exact
	// failure the Axis enum exists to prevent, one level down.
	value := strings.TrimSpace(entry.Value)
	if value != "" {
		switch axis {
		case AxisElement:
			member, err := element.Parse(value)
			if err != nil {
				return Bonus{}, fmt.Errorf("bonus %q counts %q, which is no element: %w",
					entry.ID, value, err)
			}
			// An inert element forms no tribe — Awards skips it — so a bonus named
			// after one could never fire, and saying so at parse is cheaper than a
			// bonus nobody notices is dead.
			if slices.Contains(deps.Chart.Inert(), member) {
				return Bonus{}, fmt.Errorf("bonus %q counts %q, which is inert: sharing the "+
					"element with no matchup is sharing the absence of one, and Awards "+
					"never tallies it", entry.ID, value)
			}
			value = member.String()
		case AxisColumn:
			if !strings.HasPrefix(value, "column") {
				return Bonus{}, fmt.Errorf("bonus %q counts %q, which is no column: a column "+
					"is written as column0, column1 and so on", entry.ID, value)
			}
		}
	}
	parsed := Bonus{ID: entry.ID, Name: entry.Name, Axis: axis, Value: value, Scope: scope}
	for _, rung := range entry.Rungs {
		if rung.At < MinimumRung {
			return Bonus{}, fmt.Errorf(
				"bonus %q has a rung at %d: a threshold starts at %d, because one unit shares nothing with anybody",
				entry.ID, rung.At, MinimumRung)
		}
		if rung.At > hex.MaxTeamSize {
			return Bonus{}, fmt.Errorf("bonus %q has a rung at %d, which no side of %d can reach",
				entry.ID, rung.At, hex.MaxTeamSize)
		}
		if len(parsed.Rungs) > 0 && rung.At <= parsed.Rungs[len(parsed.Rungs)-1].At {
			return Bonus{}, fmt.Errorf("bonus %q lists its rungs out of order at %d; a ladder is written from the bottom",
				entry.ID, rung.At)
		}
		if len(rung.Grants) == 0 {
			return Bonus{}, fmt.Errorf("bonus %q grants nothing at %d", entry.ID, rung.At)
		}
		held := Rung{At: rung.At}
		for _, grant := range rung.Grants {
			kind, err := deps.Statuses.Lookup(grant.Status)
			if err != nil {
				return Bonus{}, fmt.Errorf("bonus %q at %d: %w", entry.ID, rung.At, err)
			}
			// ⚠️ Permanent for the reason a trait's grant is: a bonus is settled
			// once, before the first turn, and nothing ever applies it again. A
			// timed status granted here would count down, expire, and leave a
			// squad that had built for a threshold with nothing to show for it,
			// with the log's only mention of it several turns behind.
			if !kind.Permanent {
				return Bonus{}, fmt.Errorf(
					"bonus %q at %d grants %q, which is not permanent: a bonus is applied once and never again",
					entry.ID, rung.At, grant.Status)
			}
			if grant.Stacks < 1 {
				return Bonus{}, fmt.Errorf("bonus %q at %d grants %q %d times",
					entry.ID, rung.At, grant.Status, grant.Stacks)
			}
			for _, already := range held.Grants {
				if already.Status == grant.Status {
					return Bonus{}, fmt.Errorf("bonus %q at %d grants %q twice; say it once with the stacks it wants",
						entry.ID, rung.At, grant.Status)
				}
			}
			held.Grants = append(held.Grants, Grant{Status: grant.Status, Stacks: grant.Stacks})
		}
		parsed.Rungs = append(parsed.Rungs, held)
	}
	return parsed, nil
}
