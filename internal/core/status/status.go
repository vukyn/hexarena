// Package status holds the timed effects that sit on a unit: damage over time,
// stat debuffs, control, buffs and shields.
//
// This is where duration lives, so anything that should wear off is modelled as
// a status rather than growing its own bookkeeping. Block charges are a shield
// status whose stack count is the charge count, which is how they gained an
// expiry without a second mechanism.
//
// Three decisions shape the rest.
//
// Duration is counted in the holder's own turns and a status ticks at the start
// of one. That makes the total damage of a damage-over-time independent of
// speed: a fast victim takes its three ticks sooner, not more often. Counting in
// rounds instead would punish a unit for being hasted, which would put speed at
// war with itself.
//
// Every stack keeps its own snapshotted tick damage, taken when it was applied.
// The unit that applied it may be dead by the time it resolves, and two
// different attackers may stack the same poison on one target, so a single
// number per status could not stay honest. At a cap of three stacks the cost of
// keeping them separately is nothing.
//
// Stacks are a discrete resource, so they are bounded by a hard cap rather than
// the saturation the engine uses for continuous values. Ticks themselves are not
// rolled: a status that has landed cannot be dodged or blocked, which is the
// whole role damage over time plays against those two defences. Cleanse is what
// answers it.
package status

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/vukyn/hexarena/internal/core/modifier"
)

// Category groups statuses so a cleanse can name what it removes without
// listing every kind, and so a cleanse aimed at debuffs cannot strip the
// holder's own buffs.
type Category uint8

const (
	// Dot deals its snapshotted damage at the start of the holder's turn.
	Dot Category = iota
	// StatDebuff carries modifier terms applied while it lasts.
	StatDebuff
	// Control prevents or restricts the holder's action.
	Control
	// Buff is a beneficial timed effect.
	Buff
	// Shield absorbs incoming damage; its stack count is the charge count.
	Shield
)

// CategoryCount is the number of categories.
const CategoryCount = int(Shield) + 1

var categoryNames = [CategoryCount]string{
	Dot:        "dot",
	StatDebuff: "stat_debuff",
	Control:    "control",
	Buff:       "buff",
	Shield:     "shield",
}

func (c Category) String() string {
	if int(c) >= CategoryCount {
		return fmt.Sprintf("category(%d)", uint8(c))
	}
	return categoryNames[c]
}

// Valid reports whether the value is a declared category.
func (c Category) Valid() bool { return int(c) < CategoryCount }

// Harmful reports whether the category is something the holder wants removed.
// It is what separates a cleanse from a dispel.
func (c Category) Harmful() bool {
	switch c {
	case Dot, StatDebuff, Control:
		return true
	default:
		return false
	}
}

// ParseCategory resolves a category name as written in the data files.
func ParseCategory(name string) (Category, error) {
	for i, candidate := range categoryNames {
		if candidate == name {
			return Category(i), nil
		}
	}
	return 0, fmt.Errorf("unknown status category %q", name)
}

// Categories returns every category in declaration order.
func Categories() []Category {
	out := make([]Category, 0, CategoryCount)
	for i := 0; i < CategoryCount; i++ {
		out = append(out, Category(i))
	}
	return out
}

// Kind is a declared status type.
type Kind struct {
	ID       string
	Category Category
	// MaxStacks is how many times the status can be layered on one unit.
	MaxStacks int
	// Duration is how many of the holder's turns a freshly applied stack lasts.
	Duration int
	// TickPower is what one stack deals per tick, in parts per thousand of the
	// applier's scaling stat, run through the usual damage formula at the moment
	// it is applied.
	//
	// It belongs to the status rather than to the skill for the same reason the
	// modifier terms do: poison should tick for the same thing however it was
	// inflicted. What the skill contributes is the attack behind it, which is
	// why two attackers stacking one poison produce stacks of different weight.
	TickPower int
	// Modifiers are the stat terms one stack contributes while it lasts.
	//
	// They belong to the status rather than to the skill that applies it, so
	// that every skill inflicting the same debuff inflicts the same thing. A
	// skill carrying its own terms would let two skills apply a status called
	// "weaken" that weakened by different amounts, which is a bug nobody would
	// see until they were compared.
	Modifiers []modifier.Modifier
}

// Stack is one application of a status.
type Stack struct {
	// TickDamage is the damage this stack deals per tick, snapshotted when it
	// was applied. It is zero for a status that does not deal damage.
	TickDamage int64
	// Remaining is how many of the holder's turns are left.
	Remaining int
}

type entry struct {
	kind   Kind
	stacks []Stack
}

// Set is the statuses active on one unit. The zero value is a clean unit.
//
// Entries are held in a slice in order of first application rather than a map,
// because a tick has to resolve in the same order every time for a battle to
// replay from its seed.
type Set struct {
	entries []entry
}

func (s *Set) find(id string) int {
	for i := range s.entries {
		if s.entries[i].kind.ID == id {
			return i
		}
	}
	return -1
}

// Apply layers one stack of a status onto the unit and refreshes the duration
// of every stack already there, which is what makes sustained pressure worth
// more than a single application.
//
// It reports whether a stack was added and, when the status was already at its
// cap, that the application was wasted. A caller that wants to tell the player
// its effect did nothing needs that distinction; silently succeeding would hide
// it.
func (s *Set) Apply(kind Kind, tickDamage int64) (added, wasted bool) {
	if tickDamage < 0 {
		tickDamage = 0
	}
	index := s.find(kind.ID)
	if index < 0 {
		s.entries = append(s.entries, entry{
			kind:   kind,
			stacks: []Stack{{TickDamage: tickDamage, Remaining: kind.Duration}},
		})
		return true, false
	}
	target := &s.entries[index]
	// The kind may have been redeclared between applications; the newest
	// declaration wins, so a data change takes effect without a reload.
	target.kind = kind
	for i := range target.stacks {
		target.stacks[i].Remaining = kind.Duration
	}
	if len(target.stacks) >= kind.MaxStacks {
		return false, true
	}
	target.stacks = append(target.stacks, Stack{TickDamage: tickDamage, Remaining: kind.Duration})
	return true, false
}

// Tick resolves the start of the holder's turn: it totals the damage every
// damage-dealing stack owes, then spends one turn of every stack's duration and
// drops whatever ran out.
//
// Damage is totalled before durations are spent, so a status with one turn left
// still deals its final tick. It returns the ids of statuses that ran out
// entirely, in the order they were applied.
func (s *Set) Tick() (damage int64, expired []string) {
	for i := range s.entries {
		for _, stack := range s.entries[i].stacks {
			damage += stack.TickDamage
		}
	}
	kept := s.entries[:0]
	for i := range s.entries {
		current := s.entries[i]
		alive := current.stacks[:0]
		for _, stack := range current.stacks {
			stack.Remaining--
			if stack.Remaining > 0 {
				alive = append(alive, stack)
			}
		}
		current.stacks = alive
		if len(current.stacks) == 0 {
			expired = append(expired, current.kind.ID)
			continue
		}
		kept = append(kept, current)
	}
	s.entries = kept
	return damage, expired
}

// Stacks returns how many stacks of a status the unit carries. It is what a
// skill's condition is tested against.
func (s *Set) Stacks(id string) int {
	if index := s.find(id); index >= 0 {
		return len(s.entries[index].stacks)
	}
	return 0
}

// Has reports whether the unit carries the status at all.
func (s *Set) Has(id string) bool { return s.Stacks(id) > 0 }

// Modifiers returns the stat terms every active stack contributes, accumulated.
//
// Stacks contribute their terms once each, so three stacks of a debuff are three
// times the term before the modifier package saturates the total. That is what
// makes stacking a debuff worth doing and still bounded.
func (s *Set) Modifiers() modifier.Set {
	var out modifier.Set
	for i := range s.entries {
		for range s.entries[i].stacks {
			for _, term := range s.entries[i].kind.Modifiers {
				// The terms were validated when the book was parsed, so a
				// failure here cannot come from data.
				_ = out.Add(term)
			}
		}
	}
	return out
}

// TickDamage returns what the status currently deals per tick.
func (s *Set) TickDamage(id string) int64 {
	index := s.find(id)
	if index < 0 {
		return 0
	}
	total := int64(0)
	for _, stack := range s.entries[index].stacks {
		total += stack.TickDamage
	}
	return total
}

// Remaining returns the longest duration left on any stack of a status.
func (s *Set) Remaining(id string) int {
	index := s.find(id)
	if index < 0 {
		return 0
	}
	longest := 0
	for _, stack := range s.entries[index].stacks {
		if stack.Remaining > longest {
			longest = stack.Remaining
		}
	}
	return longest
}

// Active returns the ids of every status on the unit, in the order they were
// applied.
func (s *Set) Active() []string {
	out := make([]string, 0, len(s.entries))
	for i := range s.entries {
		out = append(out, s.entries[i].kind.ID)
	}
	return out
}

// CountIn returns how many stacks the unit carries across a category.
func (s *Set) CountIn(category Category) int {
	total := 0
	for i := range s.entries {
		if s.entries[i].kind.Category == category {
			total += len(s.entries[i].stacks)
		}
	}
	return total
}

// Remove takes up to count stacks off one status and reports what went, both as
// a stack count and as the per-tick damage that stopped.
//
// The heaviest stacks go first. A cleanse that removed the weakest would leave a
// player worse off than not cleansing at all in the case they care about, and
// picking by application order would make the outcome depend on bookkeeping
// nobody can see.
func (s *Set) Remove(id string, count int) (removed int, damage int64) {
	index := s.find(id)
	if index < 0 || count <= 0 {
		return 0, 0
	}
	target := &s.entries[index]
	order := make([]int, len(target.stacks))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return target.stacks[order[a]].TickDamage > target.stacks[order[b]].TickDamage
	})
	if count > len(order) {
		count = len(order)
	}
	doomed := make(map[int]bool, count)
	for i := 0; i < count; i++ {
		doomed[order[i]] = true
		damage += target.stacks[order[i]].TickDamage
	}
	kept := make([]Stack, 0, len(target.stacks)-count)
	for i, stack := range target.stacks {
		if !doomed[i] {
			kept = append(kept, stack)
		}
	}
	target.stacks = kept
	removed = count
	if len(target.stacks) == 0 {
		s.entries = append(s.entries[:index], s.entries[index+1:]...)
	}
	return removed, damage
}

// Consume removes every stack of a status and reports what it was worth. It is
// what a skill that detonates a status calls, so the burst can be priced off
// the damage it gave up.
func (s *Set) Consume(id string) (stacks int, damage int64) {
	return s.Remove(id, s.Stacks(id))
}

// Cleanse removes up to count stacks from the given categories, working through
// the statuses in the order they were applied. It returns how many stacks went.
//
// Taking categories rather than ids is what keeps a cleanse from stripping the
// holder's own buffs, and what lets one effect answer a whole class of statuses
// without naming each one.
func (s *Set) Cleanse(categories []Category, count int) int {
	if count <= 0 || len(categories) == 0 {
		return 0
	}
	wanted := [CategoryCount]bool{}
	for _, category := range categories {
		if category.Valid() {
			wanted[category] = true
		}
	}
	removed := 0
	for removed < count {
		target := ""
		for i := range s.entries {
			if wanted[s.entries[i].kind.Category] {
				target = s.entries[i].kind.ID
				break
			}
		}
		if target == "" {
			break
		}
		took, _ := s.Remove(target, count-removed)
		if took == 0 {
			break
		}
		removed += took
	}
	return removed
}

// Snapshot is a readable view of one status, for logs and reports.
type Snapshot struct {
	ID         string
	Category   Category
	Stacks     int
	TickDamage int64
	Remaining  int
}

// Snapshot returns every active status, in the order they were applied.
func (s *Set) Snapshot() []Snapshot {
	out := make([]Snapshot, 0, len(s.entries))
	for i := range s.entries {
		current := s.entries[i]
		total, longest := int64(0), 0
		for _, stack := range current.stacks {
			total += stack.TickDamage
			if stack.Remaining > longest {
				longest = stack.Remaining
			}
		}
		out = append(out, Snapshot{
			ID: current.kind.ID, Category: current.kind.Category,
			Stacks: len(current.stacks), TickDamage: total, Remaining: longest,
		})
	}
	return out
}

// Book is the declared statuses plus the limits they obey.
type Book struct {
	// MaxStacks and MaxDuration bound every declared kind, so one status cannot
	// be authored far outside the range the rest are balanced against.
	MaxStacks   int
	MaxDuration int
	kinds       []Kind
	byID        map[string]Kind
}

type bookFile struct {
	MaxStacks   int `json:"max_stacks"`
	MaxDuration int `json:"max_duration"`
	Kinds       []struct {
		ID        string              `json:"id"`
		Category  string              `json:"category"`
		MaxStacks int                 `json:"max_stacks"`
		Duration  int                 `json:"duration"`
		TickPower int                 `json:"tick_power"`
		Modifiers []modifier.Modifier `json:"modifiers"`
	} `json:"kinds"`
}

// ParseBook reads a status declaration. It never touches the filesystem.
func ParseBook(raw []byte) (*Book, error) {
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode status book: %w", err)
	}
	if file.MaxStacks < 1 {
		return nil, fmt.Errorf("max_stacks is %d, want at least 1", file.MaxStacks)
	}
	if file.MaxDuration < 1 {
		return nil, fmt.Errorf("max_duration is %d, want at least 1", file.MaxDuration)
	}
	book := &Book{
		MaxStacks:   file.MaxStacks,
		MaxDuration: file.MaxDuration,
		byID:        make(map[string]Kind, len(file.Kinds)),
	}
	for _, declared := range file.Kinds {
		if declared.ID == "" {
			return nil, fmt.Errorf("a status needs an id")
		}
		category, err := ParseCategory(declared.Category)
		if err != nil {
			return nil, fmt.Errorf("status %q: %w", declared.ID, err)
		}
		switch {
		case declared.MaxStacks < 1:
			return nil, fmt.Errorf("status %q allows %d stacks, want at least 1", declared.ID, declared.MaxStacks)
		case declared.MaxStacks > book.MaxStacks:
			return nil, fmt.Errorf("status %q allows %d stacks, over the limit of %d",
				declared.ID, declared.MaxStacks, book.MaxStacks)
		case declared.Duration < 1:
			return nil, fmt.Errorf("status %q lasts %d turns, want at least 1", declared.ID, declared.Duration)
		case declared.Duration > book.MaxDuration:
			return nil, fmt.Errorf("status %q lasts %d turns, over the limit of %d",
				declared.ID, declared.Duration, book.MaxDuration)
		}
		switch {
		case category == Dot && declared.TickPower < 1:
			return nil, fmt.Errorf("status %q deals damage over time but has no tick_power", declared.ID)
		case category != Dot && declared.TickPower != 0:
			return nil, fmt.Errorf("status %q is %s but declares tick_power %d, which only a dot ticks with",
				declared.ID, category, declared.TickPower)
		}
		if _, clash := book.byID[declared.ID]; clash {
			return nil, fmt.Errorf("status %q is declared twice", declared.ID)
		}
		for i, term := range declared.Modifiers {
			if err := term.Validate(); err != nil {
				return nil, fmt.Errorf("status %q modifier %d: %w", declared.ID, i, err)
			}
			if term.Target == modifier.Affinity {
				// An affinity term shifts a matchup rather than a stat, and a
				// timed effect has no matchup of its own to shift.
				return nil, fmt.Errorf("status %q modifier %d targets affinity, which a status cannot carry",
					declared.ID, i)
			}
		}
		kind := Kind{
			ID: declared.ID, Category: category,
			MaxStacks: declared.MaxStacks, Duration: declared.Duration,
			TickPower: declared.TickPower, Modifiers: declared.Modifiers,
		}
		book.byID[kind.ID] = kind
		book.kinds = append(book.kinds, kind)
	}
	if len(book.kinds) == 0 {
		return nil, fmt.Errorf("the status book is empty")
	}
	return book, nil
}

// Kinds returns every declared status in declaration order.
func (b *Book) Kinds() []Kind {
	out := make([]Kind, len(b.kinds))
	copy(out, b.kinds)
	return out
}

// Lookup returns a status by id.
func (b *Book) Lookup(id string) (Kind, error) {
	kind, ok := b.byID[id]
	if !ok {
		return Kind{}, fmt.Errorf("unknown status %q", id)
	}
	return kind, nil
}
