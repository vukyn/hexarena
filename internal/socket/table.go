package socket

import (
	"sync"
	"time"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// seatsPerTable is how many connections a table holds, and it is two because a
// room has two seats (room's own seatCount, which is unexported there). It is
// named so that the one place this package **walks** both — the shutdown telling
// each peer the host stopped — walks a fixed-size array rather than a slice
// literal whose length is a coincidence.
const seatsPerTable = 2

// table is one room's two connections, plus the timer on whichever seat is being
// asked something.
//
// ⚠️ It is two connections and not a map, which is the engine's own rule one
// layer out: a room has exactly two seats, so a map would be a collection whose
// iteration order could reach an output for no benefit at all. The seats are
// found by name and never walked.
type table struct {
	// exchange orders one whole exchange — ask the room, then write what it
	// answered — so the order messages reach a peer is the order the room
	// produced them in.
	//
	// ⚠️ **It is per room and must stay per room.** The registry's own point is
	// that N rooms do not serialise through one lock, so a server-wide lock here
	// would keep the letter of that while undoing it. What it costs is that a
	// stuck peer holds its own room still for as long as one write may take,
	// which is why Timings.Write exists and is far shorter than the close
	// threshold.
	//
	// ⚠️ Without it the *client* would be what kept the order straight, because a
	// mirror cannot answer a turn it has not received — and a peer that spammed
	// acts would break that. An invariant a well-behaved peer maintains is not an
	// invariant.
	exchange sync.Mutex

	// host and guest are the two seats' connections, guarded by exchange.
	host, guest *connection

	// allowance is the timer on the seat the room is waiting for.
	allowance allowance

	// holders is how many connections still refer to this table.
	// ⚠️ Guarded by the **Server's** mutex rather than by exchange: it is the
	// server's map that owns a table's lifetime, and a count guarded by the lock
	// the table's own users take would be a count a departing connection could
	// not read without waiting on an exchange it is not part of.
	holders int

	// late is how many timeouts fired after the seat had already answered, which
	// is a normal race rather than a fault.
	//
	// ⚠️ It is a count for the same reason Room.Skipped is one: "a late timeout is
	// refused without dropping anybody" is a claim about a path that leaves no
	// other trace — nothing is sent and nothing is closed — so without it a test
	// asserting it would pass on a run where no timer ever fired late.
	late int

	// ended is the one-shot behind the finished callback: both routes to an
	// ending (a last decision, a departure) hand the reading on, and a match ends
	// once.
	over sync.Once
}

// at is the connection in a seat, or nil for a seat nobody is holding.
func (t *table) at(seat wire.Seat) *connection {
	switch seat {
	case wire.SeatHost:
		return t.host
	case wire.SeatGuest:
		return t.guest
	}
	return nil
}

// seat records a connection as holding one.
func (t *table) seat(seat wire.Seat, peer *connection) {
	switch seat {
	case wire.SeatHost:
		t.host = peer
	case wire.SeatGuest:
		t.guest = peer
	}
}

// free gives a seat up, and **only if it is still this connection's**.
//
// A seat freed before the first battle is joinable again (that is what room.Left
// does then), so by the time a departing connection tidies up, a new one may
// already hold the seat — and a departure clearing its successor's entry would
// leave a seated peer unreachable.
func (t *table) free(seat wire.Seat, peer *connection) {
	if t.at(seat) == peer {
		t.seat(seat, nil)
	}
}

// lateTimeouts is how many timeouts fired after the seat had already answered.
// → the field, for why it is counted at all.
func (t *table) lateTimeouts() int {
	t.exchange.Lock()
	defer t.exchange.Unlock()
	return t.late
}

// ended hands a finished match's reading to the caller, once.
func (t *table) ended(code wire.RoomCode, reading room.Reading, tell func(wire.RoomCode, room.Reading)) {
	if tell == nil {
		return
	}
	t.over.Do(func() { tell(code, reading) })
}

// allowance is the timer on one seat's turn, and it is **the only clock in the
// PvP stack** — internal/room and internal/wire both refuse to import `time`,
// because whoever owns the transport owns the countdown.
//
// It is armed off room.Reading and nothing else: the seat is Reading.Awaiting,
// the length is Reading.Config.Allowance (seconds as an int), and a reading that
// is not waiting on anybody disarms it. So there is no state here about *whose*
// turn it is that could disagree with the room's.
type allowance struct {
	mu    sync.Mutex
	timer *time.Timer
	// generation is what makes a stale fire silent. A timer that has already
	// fired cannot be stopped, so its callback may still run after the seat has
	// answered; comparing the generation it was armed under against the current
	// one is what stops it reporting a seat that is no longer the one being
	// asked.
	//
	// ⚠️ It is **not** what makes a late timeout safe — room.TimedOut refusing a
	// seat it is not asking is what does that, and it has to, because the answer
	// and the fire genuinely race. This only keeps the common case quiet.
	generation uint64
}

// set arms the allowance for whichever seat the room is waiting on, replacing
// whatever was armed before.
//
// `only` narrows it: a re-arm after a refused timeout passes the seat it was
// reporting, so a report that raced with the *next* prompt cannot restart the
// clock on somebody else's turn. The zero Seat means "whoever the reading says".
func (a *allowance) set(reading room.Reading, only wire.Seat, fire func(wire.Seat)) {
	seat := reading.Awaiting
	waiting := reading.Waiting && seat.Valid()
	// ⚠️ **Leave whatever is armed alone, and return before touching it.** This
	// used to set `waiting = false` and fall through — which then bumped the
	// generation and stopped the live timer on its way to the "nothing to arm"
	// return. So a late timeout for a seat that had already answered **disarmed
	// the clock on the seat now being asked**: the only caller that passes a
	// narrowing seat is timedOut's refused path, and it does not go on to call
	// settled, so nothing armed one again. The seat on turn then had no
	// allowance at all, and a player who walked away from that turn hung the
	// match — which is the exact failure the timeout input exists to prevent.
	//
	// Returning early is what the doc above always claimed: `only` is here so a
	// report that raced with the next prompt **cannot restart** the clock on
	// somebody else's turn. It never meant "cannot leave it running".
	//
	// ⚠️ It has to return above the lock rather than inside it, because the
	// generation is what makes an armed callback live: bumping it and returning
	// would leave the timer in place and turn its fire into a no-op, which is
	// the same hang wearing a different shape and is what
	// TestALateTimeoutLeavesTheLiveAllowanceArmed checks by value.
	if waiting && only.Valid() && seat != only {
		return
	}
	length := Allowance(reading.Config.Allowance)

	a.mu.Lock()
	defer a.mu.Unlock()
	a.generation++
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
	if !waiting || length <= 0 || fire == nil {
		return
	}
	armed := a.generation
	a.timer = time.AfterFunc(length, func() {
		a.mu.Lock()
		stale := armed != a.generation
		a.mu.Unlock()
		if stale {
			return
		}
		fire(seat)
	})
}

// armed is whether a timer is in place and the generation it would fire under,
// which together are what "the clock on the seat being asked is live" means.
//
// ⚠️ It exists for a test and it takes **both**, because either alone can be
// true of a dead clock: a timer with a stale generation fires and returns
// without reporting anything, and a moved generation with no timer behind it is
// simply nothing armed. → set's own comment, and
// TestALateTimeoutLeavesTheLiveAllowanceArmed, which is the only thing in this
// package that can tell an armed clock from a disarmed one — a test that lets
// its clients answer never needs the clock at all, which is why the late-timeout
// test that already existed passed while this was broken.
func (a *allowance) armed() (bool, uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.timer != nil, a.generation
}

// stop disarms whatever is armed, for a table that is going away.
func (a *allowance) stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.generation++
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
}

// ticker is the keepalive's interval, wrapped so that the one place a
// time.Ticker appears is beside the one place a time.Timer does.
type ticker struct{ inner *time.Ticker }

func newTicker(every time.Duration) ticker { return ticker{inner: time.NewTicker(every)} }

func (t ticker) each() <-chan time.Time { return t.inner.C }

func (t ticker) stop() { t.inner.Stop() }
