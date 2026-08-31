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
	"slices"
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
	// Regen restores its snapshotted amount at the start of the holder's turn.
	// It is Dot with the sign the other way, and it shares the machinery: the
	// amount is frozen when the stack is applied, so two casters stacking one
	// regeneration each contribute what their own attack was worth.
	Regen
	// Taunt takes the holder's choice of enemy rather than its turn.
	//
	// Its own category rather than a second Control, because the two do opposite
	// things to a turn: a stunned unit does not act, and a taunted one acts and
	// is simply not allowed to pick. A category exists so a cleanse can name a
	// class without listing its members, and "strips a control" that took a
	// taunt off along with a stun would be a cleanse nobody could aim.
	//
	// Declared last on purpose. The value serialises by name like every other
	// enum here, so appending cannot reinterpret a saved book or log — but
	// CategoryCount and the order the grouped reference prints in are built from
	// the declaration order, and slotting one in beside Control would move every
	// table below it for no rule.
	Taunt
)

// CategoryCount is the number of categories.
const CategoryCount = int(Taunt) + 1

var categoryNames = [CategoryCount]string{
	Dot:        "dot",
	StatDebuff: "stat_debuff",
	Control:    "control",
	Buff:       "buff",
	Shield:     "shield",
	Regen:      "regen",
	Taunt:      "taunt",
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
	case Dot, StatDebuff, Control, Taunt:
		return true
	default:
		return false
	}
}

// OutlastsAShield reports whether a rider of this category still lands when the
// strike carrying it was eaten by a shield.
//
// The reading is one sentence: **a shield stops the blow and the wear, but not
// the contamination.** Fire still burns you through a shield and poison still
// gets on you, because both are something left on the target rather than
// something done to it. A stat the blow never bent and a turn it never took are
// stopped with the strike — there is nothing left over for them to be about.
//
// ⚠️ A **missed** strike is a different question and this is not asked about
// one. A block means the blow arrived and was stopped; a miss means nothing
// touched the target, so a miss delivers nothing, this category included. That
// distinction is the whole justification for the rule and collapsing the two
// deletes it.
//
// ⚠️ **It is one case on purpose, and letting StatDebuff through as well was
// measured and rejected.** With mire unstoppable — 25% off speed a stack, two
// stacks — pokemon.squirtle against itself stops resolving: 0 of 20 duels
// finished inside spar's 4000-turn limit (Endless 40 of 40 across both
// arrangements) against 20 of 20 finishing with a kill today, mire applications
// went 373 → 12875, and nothing was anywhere near dying — every unit sat at 45%
// health or better when the limit was hit and the lowest any was driven to at any
// point was 29%. That breaks TestABothWaysMirrorIsExactlyEven, which is a
// **fairness invariant** — a character duelling an identical copy of itself
// comes to exactly 500‰ — rather than a balance number. So do not "complete"
// this predicate by adding a second case, and do not fold it into Harmful,
// which is Dot|StatDebuff|Control|Taunt and answers what a cleanse may strip.
func (c Category) OutlastsAShield() bool { return c == Dot }

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
	// A permanent status declares none.
	Duration int
	// Permanent means a stack never counts down and never expires, which is what
	// a passive's granted status needs: a passive is what a unit *is* rather than
	// something that happened to it.
	//
	// It is a flag rather than a duration of nought, because nought would make an
	// absent or mistyped duration silently permanent — and the fields around it
	// already refuse their own zero for that reason. A permanent status is
	// therefore two declarations that have to agree, and ParseBook checks they
	// do.
	Permanent bool
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
	// TickAmount is what this stack ticks for, snapshotted when it
	// was applied. It is zero for a status that does not deal damage.
	TickAmount int64
	// Remaining is how many of the holder's turns are left.
	Remaining int
}

// worth is everything this stack still owes: its frozen tick for each turn it
// has left. It is what a removal reports and what Pending totals, said once so
// the two cannot drift.
func (s Stack) worth() int64 { return s.TickAmount * int64(s.Remaining) }

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
func (s *Set) Apply(kind Kind, tickAmount int64) (added, wasted bool) {
	if tickAmount < 0 {
		tickAmount = 0
	}
	index := s.find(kind.ID)
	if index < 0 {
		s.entries = append(s.entries, entry{
			kind:   kind,
			stacks: []Stack{{TickAmount: tickAmount, Remaining: kind.Duration}},
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
	target.stacks = append(target.stacks, Stack{TickAmount: tickAmount, Remaining: kind.Duration})
	return true, false
}

// Tick resolves the start of the holder's turn: it totals what every ticking
// stack owes, then spends one turn of every stack's duration and drops whatever
// ran out.
//
// Damage and healing come back as two unsigned totals rather than one signed
// one, and that is deliberate. The caller subtracts damage through the same path
// that kills a unit at zero, so a negative arriving there would subtract a
// negative and could bring a corpse back — the one bug this shape makes
// impossible to write.
//
// Totals are taken before durations are spent, so a status with one turn left
// still ticks a final time. It returns the ids of statuses that ran out
// entirely, in the order they were applied.
func (s *Set) Tick() (damage, healing int64, expired []string) {
	for i := range s.entries {
		for _, stack := range s.entries[i].stacks {
			if s.entries[i].kind.Category == Regen {
				healing += stack.TickAmount
				continue
			}
			damage += stack.TickAmount
		}
	}
	kept := s.entries[:0]
	for i := range s.entries {
		current := s.entries[i]
		// A permanent status is not timed, so nothing about it is spent here and
		// it can never reach the expiry list. Skipping the whole entry rather
		// than guarding the decrement is what keeps that true of both halves.
		if current.kind.Permanent {
			kept = append(kept, current)
			continue
		}
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
	return damage, healing, expired
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

// Timed reports whether the unit holds anything that will change on its own:
// a stack with a duration still to spend. A permanent status does not count,
// because nothing about one is ever spent — Tick skips the whole entry.
//
// It is the question a caller asks before deciding a battle can no longer
// change. Snapshot could be read for the same answer, but only by re-deriving
// which kinds are permanent, and a caller deriving that would be a second copy
// of the rule Tick already enforces.
func (s *Set) Timed() bool {
	for i := range s.entries {
		if !s.entries[i].kind.Permanent && len(s.entries[i].stacks) > 0 {
			return true
		}
	}
	return false
}

// TimedIn is Timed narrowed to a set of categories: does the unit hold a stack
// with a duration still to spend, in one of these kinds of effect.
//
// It exists because "will anything change on its own" and "can anything change
// how this battle *ends*" are different questions, and a deadlock predicate wants
// the second. A regeneration ticking away on a unit nobody can reach changes a
// number for ever and changes nothing else; a poison on the same unit will kill
// it, which empties a side.
//
// Categories rather than ids, and a list rather than one, because that is how
// Cleanse already asks the same shape of question — a caller naming ids would be
// naming today's data instead of the rule.
func (s *Set) TimedIn(categories []Category) bool {
	wanted := [CategoryCount]bool{}
	for _, category := range categories {
		if category.Valid() {
			wanted[category] = true
		}
	}
	for i := range s.entries {
		entry := s.entries[i]
		if entry.kind.Permanent || len(entry.stacks) == 0 {
			continue
		}
		if wanted[entry.kind.Category] {
			return true
		}
	}
	return false
}

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

// TickAmount returns what the status currently deals per tick.
func (s *Set) TickAmount(id string) int64 {
	index := s.find(id)
	if index < 0 {
		return 0
	}
	total := int64(0)
	for _, stack := range s.entries[index].stacks {
		total += stack.TickAmount
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

// Hold puts a permanent status on the unit, and is one half of the door a
// passive's gate needs.
//
// Apply would do the same thing, and that is exactly why this exists: Remove
// refuses a permanent status so that nothing in the game can dispel a trait, and
// a gated trait has to be able to take its own grant back. A pair that only
// works on permanent statuses says which door is which — Apply/Remove is what
// the game does to a unit, Hold/Release is what a unit's own traits do to
// themselves, and a cleanse reaching the second would be a bug that reads as a
// typo rather than as a rule being broken.
//
// It refuses a timed status for the same reason Remove refuses a permanent one:
// a trait that granted a timed status would wear off on its holder's own turns
// with nothing to put it back, and the parse layer already says so. Both halves
// of the pair refusing the other's kind is what keeps either from becoming a
// second Apply.
//
// Stacks below one are nothing to hold. It reports how many stacks went on,
// which is nought when the status is timed or already at its cap.
func (s *Set) Hold(kind Kind, stacks int) int {
	if !kind.Permanent || stacks < 1 {
		return 0
	}
	held := 0
	for range stacks {
		// A tick amount of nought: a permanent status can be neither a
		// damage-over-time nor a regeneration, which ParseBook refuses, so there
		// is nothing here to snapshot.
		if added, _ := s.Apply(kind, 0); !added {
			break
		}
		held++
	}
	return held
}

// Release takes a permanent status back off, every stack of it, and is the other
// half of the door.
//
// Every stack rather than a count, because a grant is a fact about the unit
// rather than an accumulation: a trait that granted three stacks and gave back
// one would leave two stacks of a trait that is no longer in force. It reports
// how many went, which is nought when the status is timed or absent.
//
// A timed status is refused here so that this cannot be used as a cleanse that
// ignores a resistance. Remove is the way to take a timed status off, and it is
// the way that reports the damage that stopped.
func (s *Set) Release(id string) int {
	index := s.find(id)
	if index < 0 || !s.entries[index].kind.Permanent {
		return 0
	}
	released := len(s.entries[index].stacks)
	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return released
}

// Remove takes up to count stacks off one status and reports what went, both as
// a stack count and as the damage those stacks still owed: each one's frozen
// tick times the turns it had left, which is the same figure Pending prices a
// whole set at.
//
// The turns left rather than one tick of them. A stack is not worth a tick, it
// is worth every tick it was still going to take, and a detonate that reported
// the smaller number said a burn with two turns on it cost half what it did —
// which is exactly backwards for the decision the figure exists to inform,
// because the longer a status has left the *less* a consume is worth taking.
//
// The heaviest stacks go first. A cleanse that removed the weakest would leave a
// player worse off than not cleansing at all in the case they care about, and
// picking by application order would make the outcome depend on bookkeeping
// nobody can see. Heaviest is measured by the same worth that is reported, so
// the two cannot disagree: today every stack of one status carries the same
// duration, because Apply refreshes all of them, so this orders exactly as the
// tick alone did — and it keeps ordering by what a stack is worth if that ever
// stops being true.
func (s *Set) Remove(id string, count int) (removed int, damage int64) {
	index := s.find(id)
	if index < 0 || count <= 0 {
		return 0, 0
	}
	target := &s.entries[index]
	// A permanent status is refused here, which is what keeps a passive from
	// being dispelled. Cleanse and Consume both come through Remove, so saying it
	// once covers both — and it has to be said, because a passive is granted only
	// when the unit is enlisted: a dispel that took one off would turn it off for
	// the rest of the battle with no way back, which is a far larger effect than
	// stripping a buff somebody cast.
	if target.kind.Permanent {
		return 0, 0
	}
	order := make([]int, len(target.stacks))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return target.stacks[order[a]].worth() > target.stacks[order[b]].worth()
	})
	if count > len(order) {
		count = len(order)
	}
	doomed := make(map[int]bool, count)
	for i := 0; i < count; i++ {
		doomed[order[i]] = true
		damage += target.stacks[order[i]].worth()
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

// Consume removes every stack of a status and reports what it was worth over
// the turns it had left. It is what a skill that detonates a status calls, so
// the burst can be priced off the damage it gave up.
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
	TickAmount int64
	// Remaining is the longest turn count left across the stacks, and it is
	// meaningless when Permanent is set: a permanent status carries no duration,
	// so a renderer reading Remaining alone would draw "0 turns left" beside
	// something that never runs out.
	Remaining int
	Permanent bool
}

// Snapshot returns every active status, in the order they were applied.
func (s *Set) Snapshot() []Snapshot {
	out := make([]Snapshot, 0, len(s.entries))
	for i := range s.entries {
		current := s.entries[i]
		total, longest := int64(0), 0
		for _, stack := range current.stacks {
			total += stack.TickAmount
			if stack.Remaining > longest {
				longest = stack.Remaining
			}
		}
		out = append(out, Snapshot{
			ID: current.kind.ID, Category: current.kind.Category,
			Stacks: len(current.stacks), TickAmount: total, Remaining: longest,
			Permanent: current.kind.Permanent,
		})
	}
	return out
}

// With is a copy of this set with one application layered on, for a caller that
// wants to know what a status would do before anything does it.
//
// It exists for Suggest, which may not mutate: the rating prices a buff, a guard
// or a regeneration by building the unit that would hold it and reading the
// numbers off that, so it needs the set the application *would* produce. The
// application itself goes through Apply, so the cap, the duration refresh and
// the wasted-stack rule are the ones that resolve for real rather than a second
// reading of them.
//
// ⚠️ The copy is deep, and it has to be. A Set holds its entries in a slice and
// each entry holds its stacks in another, so a value copy shares both arrays --
// and Apply writes *through* them, refreshing every existing stack's Remaining.
// A shallow copy here would therefore have Suggest quietly refresh the real
// unit's durations every time it rated a status, which is a mutation in the one
// function that promises not to, and no existing test would have failed.
func (s *Set) With(kind Kind, tickAmount int64, stacks int) Set {
	copied := Set{}
	if len(s.entries) > 0 {
		copied.entries = make([]entry, len(s.entries))
		for i := range s.entries {
			copied.entries[i] = entry{
				kind:   s.entries[i].kind,
				stacks: slices.Clone(s.entries[i].stacks),
			}
		}
	}
	for range stacks {
		copied.Apply(kind, tickAmount)
	}
	return copied
}

// Without is With pointed the other way: a copy of this set with stacks taken
// off, which is what prices a cleanse.
//
// Same deep copy and the same reason, and the removal goes through Cleanse so
// the stacks it takes are the ones a cleanse would actually take -- a price that
// chose its own stacks could name a figure the skill never delivers.
func (s *Set) Without(categories []Category, count int) Set {
	copied := s.With(Kind{}, 0, 0)
	copied.Cleanse(categories, count)
	return copied
}

// Pending is what every ticking stack in this set still owes over the turns it
// has left: the frozen tick amount times the remaining duration, per stack.
//
// Per stack rather than per status, because TickAmount totals a status's stacks
// while Remaining reports the longest of them — multiplying those two would
// charge every stack for the longest one's duration. A cleanse is priced by this
// figure, and a price that over-counted would have the opponent cleansing ahead
// of finishing a unit off.
func (s *Set) Pending() int64 {
	total := int64(0)
	for i := range s.entries {
		for _, stack := range s.entries[i].stacks {
			total += stack.worth()
		}
	}
	return total
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
		Permanent bool                `json:"permanent,omitempty"`
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
		case declared.Permanent && declared.Duration != 0:
			return nil, fmt.Errorf("status %q is permanent and also lasts %d turns, which are two different answers",
				declared.ID, declared.Duration)
		case !declared.Permanent && declared.Duration < 1:
			return nil, fmt.Errorf("status %q lasts %d turns, want at least 1", declared.ID, declared.Duration)
		case declared.Duration > book.MaxDuration:
			return nil, fmt.Errorf("status %q lasts %d turns, over the limit of %d",
				declared.ID, declared.Duration, book.MaxDuration)
		}
		// A permanent status that ticks is a unit losing health for the whole
		// battle with nothing able to stop it, or gaining it forever, and one is
		// as broken as the other. The two ticking categories are refused
		// together for the same reason they share the tick_power requirement.
		if declared.Permanent && (category == Dot || category == Regen) {
			return nil, fmt.Errorf("status %q is a permanent %s, which would tick for the whole battle",
				declared.ID, category)
		}
		// Both ticking categories need a power and nothing else may carry one.
		// Regen is Dot with the sign the other way, so it earns the same
		// requirement rather than an exception.
		ticks := category == Dot || category == Regen
		switch {
		case ticks && declared.TickPower < 1:
			return nil, fmt.Errorf("status %q ticks but has no tick_power", declared.ID)
		case !ticks && declared.TickPower != 0:
			return nil, fmt.Errorf("status %q is %s but declares tick_power %d, which only a ticking status uses",
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
			if term.Target == modifier.HP {
				// A health term does nothing, and has always done nothing:
				// Unit.MaxHP reads the *base* line, and nothing in the engine
				// reads the modified health at all. So the status would apply,
				// show up in the log and change no number anybody can see.
				//
				// It is refused rather than fixed because fixing it is a design
				// question, not an oversight: raising a maximum mid-battle has to
				// decide whether current health follows it up, and lowering one
				// has to decide what happens to a unit already above the new
				// maximum. Neither answer is obvious and neither is needed yet.
				// A passive is what makes this reachable — "more health" is the
				// most obvious trait anybody would write — so the refusal is
				// what stops it being written and silently doing nothing.
				return nil, fmt.Errorf("status %q modifier %d targets health, which nothing in the engine reads: health comes from the stat line and does not move during a battle",
					declared.ID, i)
			}
		}
		kind := Kind{
			ID: declared.ID, Category: category,
			MaxStacks: declared.MaxStacks, Duration: declared.Duration,
			Permanent: declared.Permanent,
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

// Group is every declared status of one category, in declaration order.
type Group struct {
	Category Category
	Kinds    []Kind
}

// Grouped files the declared statuses under their categories.
//
// It exists because a cleanse names a category rather than a status — rapid_spin
// strips a stat_debuff and a dot — so a reader who cannot see which statuses
// those are cannot tell what the skill removes. Every front-end that lists the
// statuses needs the same grouping, and a grouping worked out twice is two
// answers to "which category is this in" waiting to disagree.
//
// Category order is the enum's, kind order is the book's, and a category nothing
// is declared in is left out rather than printed empty: an empty heading reads as
// a listing that failed to load its rows.
func (b *Book) Grouped() []Group {
	out := make([]Group, 0, CategoryCount)
	for _, category := range Categories() {
		var kinds []Kind
		for _, kind := range b.kinds {
			if kind.Category == category {
				kinds = append(kinds, kind)
			}
		}
		if len(kinds) == 0 {
			continue
		}
		out = append(out, Group{Category: category, Kinds: kinds})
	}
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
