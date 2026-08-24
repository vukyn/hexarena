// Package atb decides who acts next.
//
// Every unit waits Scale/speed action value between its turns, and the unit with
// the smallest pending timestamp goes first. A window of exactly Scale action
// value therefore gives a unit as many turns as it has speed, which is what
// makes speed legible: a hundred speed is a hundred turns per cycle, and doubling
// it doubles them.
//
// Scale is a million rather than the ten thousand a first pass would reach for.
// At ten thousand the integer wait collapses: fifty of the speeds from two to two
// hundred are indistinguishable from the speed one point below them, and the
// collisions are worst at the top of the range where the difference matters most.
// A million has none.
//
// A speed change keeps the fraction of the wait a unit has already served rather
// than restarting it. Restarting would make a speed buff worth a partial turn on
// its own and, worse, make repeatedly buffing and debuffing a way to stall a unit
// indefinitely.
//
// The queue is a sorted slice, not a heap. A battle holds at most a handful of
// units, so the ordering cost is irrelevant, and an explicit comparator is much
// harder to get subtly wrong than a heap invariant. Order never depends on map
// iteration, so a battle replays from its seed exactly.
package atb

import (
	"fmt"
	"sort"
)

// Scale is the action value a unit of one speed waits between turns, and the
// length of one cycle.
const Scale int64 = 1_000_000

// Wait returns how much action value a unit of the given speed waits between
// its turns. Speed below one is treated as one, and the wait never falls below
// one, so no unit can act infinitely often.
func Wait(speed int64) int64 {
	if speed < 1 {
		speed = 1
	}
	wait := Scale / speed
	if wait < 1 {
		return 1
	}
	return wait
}

// Turn is one unit's scheduled action.
type Turn struct {
	// ID is the unit that acts.
	ID string
	// Speed is the speed it acted at.
	Speed int64
	// At is the point on the timeline the turn happens.
	At int64
	// Number is how many turns this unit has taken, counting this one. Status
	// durations are measured in these.
	Number int
	// Cycle is At divided by Scale, a legible round number for logs.
	Cycle int
}

type entry struct {
	id    string
	speed int64
	next  int64
	turns int
	// seq is the order the unit joined, used as the final tie-break so the
	// ordering is total and never depends on anything unobservable.
	seq int
}

// Queue is the turn order of one battle. The zero value is an empty queue.
type Queue struct {
	now     int64
	entries []entry
	joined  int
}

// New returns an empty queue.
func New() *Queue { return &Queue{} }

func (q *Queue) find(id string) int {
	for i := range q.entries {
		if q.entries[i].id == id {
			return i
		}
	}
	return -1
}

// Add puts a unit in the queue, scheduled one full wait from the current
// instant. Its first turn therefore comes sooner the faster it is.
func (q *Queue) Add(id string, speed int64) error {
	if id == "" {
		return fmt.Errorf("a unit needs an id")
	}
	if q.find(id) >= 0 {
		return fmt.Errorf("unit %q is already in the queue", id)
	}
	if speed < 1 {
		speed = 1
	}
	q.joined++
	q.entries = append(q.entries, entry{
		id: id, speed: speed, next: q.now + Wait(speed), seq: q.joined,
	})
	return nil
}

// Remove takes a unit out, which is what a death does. It reports whether the
// unit was there.
func (q *Queue) Remove(id string) bool {
	index := q.find(id)
	if index < 0 {
		return false
	}
	q.entries = append(q.entries[:index], q.entries[index+1:]...)
	return true
}

// Len is how many units are in the queue.
func (q *Queue) Len() int { return len(q.entries) }

// Has reports whether a unit is in the queue.
func (q *Queue) Has(id string) bool { return q.find(id) >= 0 }

// Now is the current position on the timeline: the instant of the turn most
// recently taken.
func (q *Queue) Now() int64 { return q.now }

// Cycle is Now divided by Scale, a legible round number for logs.
func (q *Queue) Cycle() int { return int(q.now / Scale) }

// Turns is how many turns a unit has taken.
func (q *Queue) Turns(id string) int {
	if index := q.find(id); index >= 0 {
		return q.entries[index].turns
	}
	return 0
}

// Speed is the speed a unit is currently scheduled at.
func (q *Queue) Speed(id string) int64 {
	if index := q.find(id); index >= 0 {
		return q.entries[index].speed
	}
	return 0
}

// Pending is how much action value is left before a unit's next turn.
func (q *Queue) Pending(id string) int64 {
	index := q.find(id)
	if index < 0 {
		return 0
	}
	remaining := q.entries[index].next - q.now
	if remaining < 0 {
		return 0
	}
	return remaining
}

// order sorts the entries into turn order: soonest first, then the faster unit,
// then the one that joined earlier. The last two rules exist only to make the
// ordering total; without them two units scheduled at the same instant would
// resolve in whatever order the slice happened to hold.
func (q *Queue) order() {
	sort.SliceStable(q.entries, func(a, b int) bool {
		left, right := q.entries[a], q.entries[b]
		if left.next != right.next {
			return left.next < right.next
		}
		if left.speed != right.speed {
			return left.speed > right.speed
		}
		return left.seq < right.seq
	})
}

// Next advances to the next turn and reports it, or reports false when the
// queue is empty. The unit that acted is immediately rescheduled a full wait
// later, at whatever speed it currently has.
func (q *Queue) Next() (Turn, bool) {
	if len(q.entries) == 0 {
		return Turn{}, false
	}
	q.order()
	acting := &q.entries[0]
	q.now = acting.next
	acting.turns++
	acting.next = q.now + Wait(acting.speed)
	return Turn{
		ID: acting.id, Speed: acting.speed, At: q.now,
		Number: acting.turns, Cycle: int(q.now / Scale),
	}, true
}

// Preview reports the next turns without taking them, which is what a turn
// order display reads. It works on a copy, so the queue is untouched.
func (q *Queue) Preview(count int) []Turn {
	if count <= 0 || len(q.entries) == 0 {
		return nil
	}
	clone := q.clone()
	out := make([]Turn, 0, count)
	for i := 0; i < count; i++ {
		turn, ok := clone.Next()
		if !ok {
			break
		}
		out = append(out, turn)
	}
	return out
}

func (q *Queue) clone() *Queue {
	copied := make([]entry, len(q.entries))
	copy(copied, q.entries)
	return &Queue{now: q.now, entries: copied, joined: q.joined}
}

// Reschedule changes a unit's speed, keeping the fraction of its wait already
// served.
//
// A unit halfway to its turn stays halfway to it: the remaining action value is
// scaled by the ratio of the old speed to the new one. Recomputing a whole fresh
// wait instead would hand a speed buff a partial turn for free, and would let a
// unit be stalled indefinitely by alternating a buff and a debuff on it.
//
// The scaling truncates, so each change loses up to one action value when the
// speeds do not divide evenly: a buff and its removal together leave the unit up
// to two further along than it started. The direction is deliberate. Drifting
// forwards by a hair cannot be used to stall a unit, which is the failure that
// would actually matter, and eating a wait worth thousands two at a time would
// take more buff and debuff pairs than a battle contains.
// TestRescheduleDriftIsBounded pins the size so a later change to the formula
// cannot quietly widen it.
func (q *Queue) Reschedule(id string, speed int64) error {
	index := q.find(id)
	if index < 0 {
		return fmt.Errorf("unit %q is not in the queue", id)
	}
	if speed < 1 {
		speed = 1
	}
	target := &q.entries[index]
	if target.speed == speed {
		return nil
	}
	remaining := target.next - q.now
	if remaining < 0 {
		remaining = 0
	}
	target.next = q.now + remaining*target.speed/speed
	target.speed = speed
	return nil
}

// GrantTurn schedules a unit to act at the current instant, which is how an
// extra action is given out. It reports whether the unit was there.
func (q *Queue) GrantTurn(id string) bool {
	index := q.find(id)
	if index < 0 {
		return false
	}
	q.entries[index].next = q.now
	return true
}

// Delay pushes a unit's next turn back by a share of one full wait, in parts per
// thousand, and reports the action value added. It is the counterpart to
// GrantTurn for effects that cost a target its tempo without touching its speed.
func (q *Queue) Delay(id string, permille int64) int64 {
	index := q.find(id)
	if index < 0 || permille <= 0 {
		return 0
	}
	added := Wait(q.entries[index].speed) * permille / 1000
	if added < 1 {
		added = 1
	}
	q.entries[index].next += added
	return added
}

// Standing is a unit's place in the queue, for logs and reports.
type Standing struct {
	ID      string
	Speed   int64
	Pending int64
	Turns   int
}

// Standings reports every unit in turn order.
func (q *Queue) Standings() []Standing {
	q.order()
	out := make([]Standing, 0, len(q.entries))
	for i := range q.entries {
		current := q.entries[i]
		pending := current.next - q.now
		if pending < 0 {
			pending = 0
		}
		out = append(out, Standing{
			ID: current.id, Speed: current.speed, Pending: pending, Turns: current.turns,
		})
	}
	return out
}
