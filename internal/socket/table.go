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

// MaxWatchers is how many watching connections one table carries, and it is
// **not** a second seat count: it bounds a collection rather than naming one.
//
// Eight, and the two things it was picked against. A room is a handful of people
// in one place on one LAN, so four times the two seats is more than a group
// watching a match ever needs — and the cost of the number is paid inside the
// room's exchange lock, where every watcher is written to in turn, so the worst
// case a stuck spectator can impose on the two people actually playing is this
// many Timings.Write. Uncapped is the real objection: a table's watchers are
// connections a stranger on the LAN opens, and nothing else in this transport
// would stop one machine opening thousands.
//
// It is exported because the cap is a fact about the transport that its callers
// need: a host printing how many people are watching, a client wording
// wire.CodeTooManyWatchers, and the test that fills a table up.
//
// ⚠️ **It stayed here when the host's flag landed, and that is a decision.**
// Whether a room takes watchers at all is the room's own configuration
// (room.Config.Watchable, chosen by cmd/hexarena-host's -watch, refused at the
// gate with wire.CodeWatchingClosed); how many may watch it at once is this,
// and the two are different questions with different answers. Moving this onto
// room.Config would put a number about watchers on a struct whose whole point is
// that the room counts none of them — and the cost this bounds is paid here,
// inside the room's exchange lock, over connections only this package holds.
const MaxWatchers = 8

// watcher is one connection reading the room's record: a socket, its own place
// in that record, and the way to end it.
//
// ⚠️ **It is deliberately not a seat and lives beside the two rather than among
// them.** The order the seats are visited in reaches the roster and the roster's
// order decides which side wins a speed tie, so a watcher threaded through the
// same fields would change who wins the match it came to watch — and nothing
// would look wrong, because the roster would still be legal and both players
// would still agree on every digest. → room/watch.go, which says the same thing
// from the room's end, and seatsPerTable above.
type watcher struct {
	// peer is the socket, and it is written to under the table's exchange lock
	// like a seat's.
	peer *connection
	// cursor is where this connection has reached in the room's record.
	//
	// ⚠️ It is a **check** rather than a read position, and the difference is
	// what makes an out-of-range cursor unreachable: the bodies are handed to
	// the transport by the answer to the input that recorded them, so this is
	// compared against where the room says the record now stands and is never
	// passed back into a room. → room.Answer.Watched, and Server.forward.
	cursor int
	// stop ends this connection's own goroutine, which is how a watcher is let
	// go of without holding a lock across a close: cancelling the context its
	// read is sitting in unblocks that read, and everything else — the socket,
	// the table entry, the cursor — is tidied up by the departure that already
	// runs behind every connection.
	stop func()
}

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

	// watchers is every connection reading this room's record, in the order they
	// arrived, guarded by exchange and bounded by MaxWatchers.
	//
	// ⚠️ **The same lock as the seats, and that is the point rather than
	// convenience.** exchange orders one whole exchange — ask the room, then
	// write what it answered — so the order bodies reach a watcher is the order
	// the room produced them in, which is exactly the guarantee the two seats
	// get. A lock of its own would let two exchanges' fan-outs interleave and
	// deliver one watcher turn six before turn five, and no client could tell,
	// because a mirror applies whatever it is handed. The cost is the one
	// exchange already carries — a stuck peer holds its own room still for up to
	// Timings.Write — with MaxWatchers bounding how many times over.
	//
	// ⚠️ It is a **slice and not a map**, for the reason the seats are two named
	// fields: a map's iteration order must not reach an output, and the order
	// several people are written to is one.
	watchers []*watcher

	// allowance is the timer on the seat the room is waiting for.
	allowance allowance

	// holding is the reconnect window on each seat: the timer that will tell the
	// room the client is not coming back, and nil for a seat that is not being
	// held. Guarded by exchange, like the seats themselves.
	//
	// ⚠️ **A held seat is still TAKEN in the room**, which is the whole
	// arrangement: the transport keeps the room ignorant of a socket closing, so
	// the match, the board and the clock are exactly where they were and a
	// returning client needs nothing rebuilt. What the transport has given up is
	// the seat's *connection*, so nothing is written to it and nothing is read
	// from it — and if the allowance runs out meanwhile, the room passes the turn
	// as it would for anybody thinking too long.
	holding [seatsPerTable]*time.Timer
	// rejoinable is whether each seat can be come back to at all, which is the
	// room's answer rather than this package's. → room.Admission.Rejoinable.
	//
	// ⚠️ Without it a room that issues no tokens would hold a seat open for a
	// minute for a client that has no way to prove it is that client — a minute
	// of the other player's evening spent on nothing.
	rejoinable [seatsPerTable]bool

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

// indexOfSeat is a seat's place in the fixed-size arrays above, and whether it is
// a seat at all. A watcher has none.
func indexOfSeat(seat wire.Seat) (int, bool) {
	switch seat {
	case wire.SeatHost:
		return 0, true
	case wire.SeatGuest:
		return 1, true
	}
	return 0, false
}

// hold starts the reconnect window on a seat, and reports whether one was
// started. The caller holds exchange.
//
// It refuses to start a second window on one seat: a seat can only be left once
// while it is empty, and a stray second call would leave the first timer running
// with nothing to stop it.
func (t *table) hold(seat wire.Seat, window time.Duration, expired func()) bool {
	index, ok := indexOfSeat(seat)
	if !ok || !t.rejoinable[index] || t.holding[index] != nil || window <= 0 {
		return false
	}
	t.holding[index] = time.AfterFunc(window, expired)
	return true
}

// release stops the reconnect window on a seat, and reports whether one was
// running. The caller holds exchange.
//
// ⚠️ **A false from Stop is not a failure here and must not be treated as one.**
// It means the timer had already fired, which is the race this whole arrangement
// has: the window expired while the returning client's hello was in flight. What
// happens then is decided by the room rather than here — the expiry has already
// told the room the seat left, so the hello that arrives a moment later finds a
// seat with no token to match and is refused as an ordinary full room, or seated
// as a new player if the seat was freed. Either is correct; what would not be is
// this pretending the window was still open.
func (t *table) release(seat wire.Seat) bool {
	index, ok := indexOfSeat(seat)
	if !ok || t.holding[index] == nil {
		return false
	}
	running := t.holding[index].Stop()
	t.holding[index] = nil
	return running
}

// releaseAll stops every window, for a table that is going away. The caller holds
// exchange.
func (t *table) releaseAll() {
	for index := range t.holding {
		if t.holding[index] != nil {
			t.holding[index].Stop()
			t.holding[index] = nil
		}
	}
}

// held reports whether a seat is being kept for a client that may come back. The
// caller holds exchange.
func (t *table) held(seat wire.Seat) bool {
	index, ok := indexOfSeat(seat)
	return ok && t.holding[index] != nil
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

// full is whether this table already carries as many watchers as it may, and it
// is the **one** declaration of that bound.
//
// ⚠️ **admit deliberately does not check it again, and that is a measurement
// rather than a preference.** The bound has to be read *before* the welcome goes
// out — a welcome followed by a refusal is two answers to one hello — so the
// refusal is the caller's, and a second check here would be a guard a mutation
// deletes for free: with one in each place, turning this one into "make room by
// dropping the oldest watcher" left the whole suite green, because the caller's
// refusal never let the mutated line run. One bound, one place that states it,
// and a test that can see it move.
//
// The caller holds exchange.
func (t *table) full() bool { return len(t.watchers) >= MaxWatchers }

// admit records a connection as watching.
//
// The caller holds exchange and has already asked full. The cursor is where the
// record stood when this connection was handed everything recorded so far, so
// the first fan-out that follows can check that it is handing this watcher the
// very next bodies.
func (t *table) admit(peer *connection, cursor int, stop func()) {
	t.watchers = append(t.watchers, &watcher{peer: peer, cursor: cursor, stop: stop})
}

// unwatch gives a watching connection's place back, and takes the lock itself
// because it is called from the departure behind every connection.
//
// It is safe to call for a connection that is no longer there, which is the
// ordinary case rather than a guard: a match that ends lets go of every watcher
// on its way out, and each of those connections then runs this on its way out
// too.
func (t *table) unwatch(peer *connection) {
	t.exchange.Lock()
	defer t.exchange.Unlock()
	for at, watching := range t.watchers {
		if watching.peer == peer {
			t.watchers = append(t.watchers[:at], t.watchers[at+1:]...)
			return
		}
	}
}

// watching is how many connections are reading this room's record.
func (t *table) watching() int {
	t.exchange.Lock()
	defer t.exchange.Unlock()
	return len(t.watchers)
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
	// spent is how long each seat has been kept waiting across the whole match,
	// and armedSeat with armedAt is the stretch currently being counted.
	//
	// ⚠️ **A seat is charged for the time it was ASKED, not for the length the
	// timer was armed with**, which is the whole difference between a chess clock
	// and a per-turn allowance. A player who answers in five seconds of a ninety
	// second allowance has spent five, and charging the armed length would spend
	// a whole match's budget in a handful of prompt turns.
	//
	// ⚠️ They live on this struct rather than on the table because this is what
	// already knows when a clock started: `set` is the one place a seat begins
	// and stops being waited on, so the charge has exactly one site and cannot
	// drift from the arming.
	spent     [seatsPerTable]time.Duration
	armedSeat wire.Seat
	armedAt   time.Time
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
	// Whatever was being waited on stops being waited on here, so this is where
	// it is charged — before anything decides what to arm next, so a seat asked
	// twice running is charged for the first stretch rather than having it
	// overwritten.
	a.charge()
	if !waiting || length <= 0 || fire == nil {
		return
	}
	// ⚠️ **The shorter of the two, which is what makes the budget a budget.** The
	// allowance bounds one turn and the budget bounds the match; a clock armed
	// for the allowance alone would let a player with four seconds left hold a
	// prompt for ninety. A budget already spent leaves nothing to arm for, and
	// that case is a timeout the moment the prompt opens rather than a hang —
	// which is what "running out is not a forfeit" means in practice, because the
	// room then passes that turn like any other timeout.
	if budget := Allowance(reading.Config.Budget); budget > 0 {
		if left := budget - a.spentBy(seat); left < length {
			length = left
		}
		if length <= 0 {
			// Nothing left at all. Arming a zero timer would be a fire on the
			// next tick anyway; a tiny one keeps every path through this function
			// the same shape and keeps the fire on the timer's goroutine rather
			// than on this one, which is the ordering everything below assumes.
			length = time.Millisecond
		}
	}
	a.armedSeat, a.armedAt = seat, time.Now()
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

// charge adds the stretch just ended to the seat that was being waited on. The
// caller holds the mutex.
//
// A zero armedAt is nothing being waited on, which is every call before the first
// prompt and every call after a match ends.
func (a *allowance) charge() {
	index, seated := indexOfSeat(a.armedSeat)
	if !seated || a.armedAt.IsZero() {
		a.armedSeat, a.armedAt = "", time.Time{}
		return
	}
	a.spent[index] += time.Since(a.armedAt)
	a.armedSeat, a.armedAt = "", time.Time{}
}

// spentBy is how long one seat has been kept waiting so far. The caller holds
// the mutex.
func (a *allowance) spentBy(seat wire.Seat) time.Duration {
	index, seated := indexOfSeat(seat)
	if !seated {
		return 0
	}
	return a.spent[index]
}

// Spent is how long each seat has been waited on across the match, for a caller
// that has to report a clock. Seats are in the order the room hands them out.
func (a *allowance) Spent() [seatsPerTable]time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.spent
	// The stretch in flight is included, because a reader asking "how much has
	// this seat used" while it is being asked wants the answer that includes now.
	if index, seated := indexOfSeat(a.armedSeat); seated && !a.armedAt.IsZero() {
		out[index] += time.Since(a.armedAt)
	}
	return out
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
	a.charge()
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
