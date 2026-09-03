package room

import (
	"fmt"
	"slices"
	"sync"

	"github.com/vukyn/hexarena/internal/wire"
)

// Registry is the many rooms one process runs, keyed by the code a player
// pastes. It is the concurrency around the room and **nothing else**: a room
// owns its battle in one goroutine and shares it with nothing, and the mutex
// that makes that true is this file's.
//
// ⚠️ **The mutex guards the map and nothing else.** It is taken to look a code
// up and released before anything is sent to the room it found, so N rooms in a
// process do not serialise through one lock. That is the whole point of the rule
// rather than a refinement of it: a mutex held across the send would keep the
// letter of "one goroutine per room" while making the process as slow as one
// room. There is exactly one place a request path locks — lookup — and exactly
// one place a request is sent — ask — and
// TestNoLockingFunctionSendsOnAChannel holds the two apart mechanically.
//
// # A request is a value, never a closure
//
// ⚠️ A `func(*Room)` on the channel is the tidy-looking design that **defeats
// the invariant**: it lets the caller capture the pointer and keep it, so the
// room's state is reachable from a goroutine that is not its own, and nothing
// about the code would look wrong. So what travels is a small discriminated
// value — request, with an inputKind and the arguments of the matching input —
// the goroutine switches on it and calls the method itself, and no *Room is
// reachable from anywhere but that goroutine. TestTheRegistryHandsOutNoRoom is
// the other half of it: nothing on this type's exported surface passes one out.
//
// The *data* is shared and that is a different thing. Deps holds the parsed
// books and the cast, every room reads the same copy, and none of it is written
// after it is parsed — what must not be shared is the *battle*.
//
// # The registry reads no clock either
//
// ⚠️ Whoever owns the transport owns the countdown, and the transport is the
// next item — so TimedOut is **forwarded** here exactly as it is taken there,
// and nothing in this file starts a timer, holds a deadline or asks what time it
// is. `time` stays unimported across the whole package and
// TestTheRoomReadsNoClock keeps holding it with this file in the directory it
// walks, which is one of the two reasons the registry lives in internal/room
// rather than in a package of its own. The other is that a *Room need not leave
// the package at all.
//
// # What is deliberately NOT in here
//
// So a reader does not go looking for it:
//
//   - **The WebSocket.** This is the concurrency and not the transport. Nothing
//     here opens a socket, and Open is handed the code rather than deriving one
//     — see the note on Open, which is where the design record's one collision
//     is written down.
//   - **The clock**, per above.
//   - **Writing a finished match out as a battle.Log.** Another cursor over the
//     battle, and the room already reads it that way so that a second consumer
//     costs nothing.
//   - **Spectators.** A third seat is a room change, not a registry one; a
//     watcher is a cursor.
type Registry struct {
	// mu guards rooms and live, and nothing else. ⚠️ It is never held across a
	// send on a room's inbox.
	mu sync.Mutex
	// idle is broadcast when a room's goroutine ends, and is what Wait waits on.
	// A condition variable rather than a channel because what is being waited
	// for is a *count* reaching nought, and it is guarded by the same mutex that
	// owns the count — one truth rather than a WaitGroup beside an int.
	idle *sync.Cond
	// rooms is every room reachable by code. A room removes its own entry when
	// its match ends, so a finished room stops being joinable without anybody
	// sweeping.
	rooms map[wire.RoomCode]*handle
	// live is how many room goroutines are owed an end.
	//
	// ⚠️ It is incremented in enrol — **before** the goroutine is started —
	// rather than by the goroutine itself, because a count the goroutine
	// increments would let Wait return in the window between the enrolment and
	// the first instruction of the goroutine, and report an empty process while
	// a room was starting up.
	//
	// It is not len(rooms): a room that has retired its entry but not yet
	// returned is still a goroutine, and "no goroutine is left behind" is
	// precisely the claim those two readings measure differently.
	live int
}

// NewRegistry is an empty registry.
func NewRegistry() *Registry {
	registry := &Registry{rooms: make(map[wire.RoomCode]*handle)}
	registry.idle = sync.NewCond(&registry.mu)
	return registry
}

// handle is a running room seen from outside its goroutine: three channels, and
// ⚠️ **no *Room**. That absence is the invariant rather than a description of
// it — a caller that holds a handle cannot reach the battle even by mistake.
type handle struct {
	// inbox is where requests go, and it is **never closed**. A send on a closed
	// channel panics and both closing it twice and closing it under a racing
	// sender are reachable here, so the goroutine's departure is announced by
	// closing done instead and every send selects on that.
	inbox chan request
	// quit asks the goroutine to stop, and is closed by whichever Close removed
	// the entry from the map — so exactly once, by construction, rather than by
	// agreement.
	quit chan struct{}
	// done is closed by the goroutine itself as it leaves, which is what makes a
	// send to a room that has just finished a refusal rather than a deadlock.
	done chan struct{}
}

// inputKind names which of the room's inputs a request carries. It exists so
// that a request can be a value: a kind and the arguments the kind needs.
type inputKind uint8

const (
	inputJoin inputKind = iota
	inputDeliver
	inputTimedOut
	inputLeft
	// inputRead is not one of the room's inputs at all — it is the readings a
	// transport needs (Awaiting, Result, Played) taken **inside** the room's
	// goroutine, because reading them from outside would be sharing the room.
	inputRead
)

// request is one call into a room.
//
// ⚠️ It is a **value with a kind**, deliberately, rather than the func(*Room)
// that suggests itself: a closure would let the caller keep the pointer it
// captured. → the package's note on Registry.
type request struct {
	kind inputKind
	// hello is inputJoin's argument.
	hello wire.Hello
	// seat is the argument of inputDeliver, inputTimedOut and inputLeft.
	seat wire.Seat
	// body is inputDeliver's message. An interface, but a message rather than a
	// function: nothing in wire closes over a room.
	body wire.Body
	// reply is where the answer goes, one channel per request, buffered so that
	// the room's goroutine never blocks answering.
	reply chan served
}

// served is what comes back down a request's reply channel: the answer and the
// error, kept apart the way a Go call keeps them apart.
type served struct {
	answer Answer
	err    error
}

// Answer is everything the registry says back to one input: the messages the
// room wants sent, the seat a Join handed out, and **the room as it stands after
// that input**.
//
// ⚠️ The Reading rides on the answer rather than being fetched afterwards, and
// that is forced rather than convenient. A room removes its own entry the moment
// its match ends — which is what keeps a finished room from lingering — so a
// transport that asked *afterwards* what the result was would be asking about a
// room that had already gone, and the result of every match would be
// unreachable. The answer to the input that ended the match is the only place it
// can be read.
//
// ⚠️ **wire.Closed does not change that**, and a reader will ask. That message
// is for the *peer*, which needs to know its opponent went away, and it is sent
// on one ending only: a match played out to its end sends nothing, because the
// client computes that ending itself. So the transport's own reading of any
// ending still has to travel here — the same division Known draws between a
// refusal for the peer and an answer for the transport.
type Answer struct {
	// Seat is the seat a Join handed out, and the zero Seat everywhere else —
	// including a refused join, exactly as Room.Join reports it.
	Seat wire.Seat
	// Out is every message the room wants sent, in order.
	Out []Outbound
	// Reading is the room after the input. It is the zero Reading when Known is
	// false, because there was no room to read.
	Reading Reading
	// Known is false when no room is running under that code, in which case Out
	// is the one wire.CodeRoomUnknown refusal.
	//
	// It is not a second copy of that refusal: the refusal is for the *peer*,
	// which reads a code and words it, and this is for the *transport*, which
	// has to decide whether to keep the connection at all. A transport reading
	// the code out of a message body to make that decision would be a transport
	// parsing its own output.
	Known bool
}

// Reading is what a transport reads off a running room, taken inside that room's
// own goroutine and copied out.
//
// ⚠️ **Pending is deliberately not in it.** Room.Pending hands back a
// *battle.Prompt, which is a pointer into the room's own state, and passing one
// out of the goroutine is exactly the sharing this file exists to prevent. What
// a rejoining client needs is the open prompt, so this is a real gap rather than
// an omission — it wants a copy whose slices are copied too, and the client that
// would read it is a later item. → TODO.md, under the seat token and the rejoin.
type Reading struct {
	// Config is what the room was opened with, which is where the allowance a
	// transport counts down comes from.
	Config Config
	// Awaiting is the seat whose answer is due and Waiting whether the room is
	// waiting on anybody at all — Room.Awaiting, which is what an allowance is
	// started on.
	Awaiting wire.Seat
	Waiting  bool
	// Finished and Result are the match.
	Finished bool
	Result   Result
	// Played is a **copy** of the battles the room recorded, so that nothing but
	// the room's goroutine holds a slice the room appends to.
	Played []BattleResult
	// Skipped is how many prompts the room walked past. → Room.Skipped for why
	// it is a reading rather than a debug counter.
	Skipped int
}

// Open starts a room under a code and returns nothing but an error, because
// ⚠️ **a *Room must not leave the goroutine that will own it** — the registry
// builds one and keeps it, and the code is the only handle on it anybody gets.
//
// # The code comes from the caller, and that is a decomposition
//
// The design record says a room code **carries its own address** — base32 of
// four address bytes and two port bytes, ten characters — and it also says one
// process runs **many rooms**. With one listener those two cannot both hold:
// every room would encode the same address and port, so the code would not
// identify a room. The reading that satisfies both without moving the wire
// format is **one listener per room**, so the code names that listener; the
// alternative is changing wire.RoomCode, which moves messages.golden and breaks
// the ten-character claim.
//
// ⚠️ **It is not decided here, and it cannot be**: allocating a port is I/O and
// the registry has none. So Open takes the code it is given, refuses a duplicate,
// and refuses one that does not decode — a code nothing can decode is a code no
// client can have typed. The decision lands with the socket. → TODO.md, under
// the WebSocket transport, and README.md § A room, and getting into one.
func (g *Registry) Open(code wire.RoomCode, config Config, deps Deps) error {
	if _, err := code.AddrPort(); err != nil {
		return fmt.Errorf("open a room: %w", err)
	}
	// The room is built — and its configuration and data validated — before the
	// map is touched, so a room that cannot run a match never occupies a code
	// and never starts a goroutine.
	opened, err := New(config, deps)
	if err != nil {
		return err
	}
	entry := &handle{
		inbox: make(chan request),
		quit:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	if err := g.enrol(code, entry); err != nil {
		return err
	}
	go g.serve(code, opened, entry)
	return nil
}

// Join is the gate, by code. → Room.Join.
func (g *Registry) Join(code wire.RoomCode, hello wire.Hello) (Answer, error) {
	return g.ask(code, request{kind: inputJoin, hello: hello})
}

// Deliver hands a room one message from a seated peer, by code. → Room.Deliver.
func (g *Registry) Deliver(code wire.RoomCode, from wire.Seat, body wire.Body) (Answer, error) {
	return g.ask(code, request{kind: inputDeliver, seat: from, body: body})
}

// TimedOut forwards the transport's report that an allowance ran out, by code.
//
// ⚠️ It **forwards** and does not measure. The registry no more reads a clock
// than the room does; the countdown belongs to whoever owns the connection.
// → Room.TimedOut.
func (g *Registry) TimedOut(code wire.RoomCode, seat wire.Seat) (Answer, error) {
	return g.ask(code, request{kind: inputTimedOut, seat: seat})
}

// Left forwards the transport's report that a peer went away, by code.
// → Room.Left.
func (g *Registry) Left(code wire.RoomCode, seat wire.Seat) (Answer, error) {
	return g.ask(code, request{kind: inputLeft, seat: seat})
}

// Read is the readings a transport needs, taken inside the room's goroutine,
// for a transport that wants to look at a room without giving it an input — the
// other peer's join is what starts the first allowance, and the seat waiting for
// it was handed no answer of its own.
//
// It reports false for a code no room is running under, which a **finished match
// also is**. The result of a match is therefore read off the Answer to the input
// that ended it and not from here. → the note on Answer.
func (g *Registry) Read(code wire.RoomCode) (Reading, bool) {
	answered, err := g.ask(code, request{kind: inputRead})
	if err != nil || !answered.Known {
		return Reading{}, false
	}
	return answered.Reading, true
}

// Close stops one room and reports whether there was one to stop. The room's
// goroutine ends; whatever it was in the middle of a match is not written
// anywhere, because nothing writes a match out yet.
func (g *Registry) Close(code wire.RoomCode) bool {
	entry, running := g.unenrol(code)
	if !running {
		return false
	}
	// Exactly-once by construction rather than by agreement: only the caller
	// that removed the entry from the map reaches this line.
	close(entry.quit)
	return true
}

// CloseAll stops every room and reports how many it stopped, which is what a
// server's shutdown does before it Waits.
func (g *Registry) CloseAll() int {
	closing := g.unenrolAll()
	for _, entry := range closing {
		close(entry.quit)
	}
	return len(closing)
}

// Wait blocks until every room's goroutine has ended.
//
// ⚠️ It waits for rooms to end **by themselves** and closes nothing, which is
// what makes it a measurement: a goroutine left behind by a finished match hangs
// this rather than being tidied up by the call that was supposed to observe it.
// A shutdown is therefore CloseAll then Wait, in that order and as two calls.
func (g *Registry) Wait() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for g.live > 0 {
		g.idle.Wait()
	}
}

// Count is how many rooms are reachable by code.
func (g *Registry) Count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.rooms)
}

// Running is how many room goroutines are owed an end, which is not Count: a
// room that has retired its entry but not yet returned is still a goroutine.
// → the note on the field.
func (g *Registry) Running() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.live
}

// Codes is every code a room is running under, **sorted**, because a map's
// iteration order must not reach an output — the engine's rule one layer out.
func (g *Registry) Codes() []wire.RoomCode {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]wire.RoomCode, 0, len(g.rooms))
	for code := range g.rooms {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}

// ask is the one path into a running room: look the code up under the mutex,
// **release it**, then send.
//
// ⚠️ The lock and the send are in two functions on purpose, and the split is not
// cosmetic. Holding the mutex across the send would serialise every room in the
// process through one lock — the letter of "one goroutine per room" with its
// point lost — and it is the mutation that measures this file, so
// TestNoLockingFunctionSendsOnAChannel refuses any function that both touches
// the mutex and sends.
//
// An unknown code is answered wire.CodeRoomUnknown, and so is a room that
// finished between the lookup and the send — an ordinary race rather than a
// fault. ⚠️ A send on a closed channel panics and a second close panics, and
// both are reachable here, so a room's inbox is never closed and the escape from
// a send nobody will read is its done channel.
func (g *Registry) ask(code wire.RoomCode, asked request) (Answer, error) {
	entry, running := g.lookup(code)
	if !running {
		return Answer{Out: roomUnknown()}, nil
	}
	asked.reply = make(chan served, 1)
	select {
	case entry.inbox <- asked:
	case <-entry.done:
		return Answer{Out: roomUnknown()}, nil
	}
	// Every request the goroutine takes off inbox is answered before it looks at
	// anything else, and reply is buffered and private to this call, so the room
	// cannot block writing it and this cannot block reading it.
	answered := <-asked.reply
	answered.answer.Known = true
	return answered.answer, answered.err
}

// serve is the room's one goroutine. **The *Room is a parameter of this function
// and is reachable from nowhere else**, which is what "shares it with nothing"
// means when it is enforced rather than asked for.
//
// It contains a send and no lock, and its teardown contains a lock and no send.
// That separation is the invariant, and it is why the retirement is a deferred
// call rather than a step inside the loop.
func (g *Registry) serve(code wire.RoomCode, playing *Room, entry *handle) {
	defer g.retired(code, entry)
	for {
		select {
		case <-entry.quit:
			return
		case asked := <-entry.inbox:
			asked.reply <- answerFrom(playing, asked)
			// A finished match is the room's own end: the entry goes and the
			// goroutine returns, so nothing has to sweep and a finished room
			// stops being joinable.
			if playing.Finished() {
				return
			}
		}
	}
}

// answerFrom is the switch, and it is the **only** place a *Room is called. A
// free function rather than a method so that the one thing holding a room is a
// parameter of a call made on the room's own goroutine.
func answerFrom(playing *Room, asked request) served {
	var answered served
	switch asked.kind {
	case inputJoin:
		answered.answer.Seat, answered.answer.Out, answered.err = playing.Join(asked.hello)
	case inputDeliver:
		answered.answer.Out, answered.err = playing.Deliver(asked.seat, asked.body)
	case inputTimedOut:
		answered.answer.Out, answered.err = playing.TimedOut(asked.seat)
	case inputLeft:
		answered.answer.Out, answered.err = playing.Left(asked.seat)
	case inputRead:
	default:
		// Unreachable while the five kinds above are the five that exist, and it
		// answers rather than panicking because a request the registry cannot
		// read is a bug in this file and not a reason to take a process down.
		answered.err = fmt.Errorf("the registry cannot read input kind %d", asked.kind)
		return answered
	}
	// Read on every answer, not only on inputRead: a match's result has to
	// travel back with the input that ended it. → the note on Answer.
	answered.answer.Reading = readingOf(playing)
	return answered
}

// readingOf copies the room's readings out. Everything it carries is a value or
// a copy, so nothing the transport holds is memory the room writes to.
func readingOf(playing *Room) Reading {
	seat, waiting := playing.Awaiting()
	played := playing.Played()
	return Reading{
		Config:   playing.Config(),
		Awaiting: seat,
		Waiting:  waiting,
		Finished: playing.Finished(),
		Result:   playing.Result(),
		Played:   append(make([]BattleResult, 0, len(played)), played...),
		Skipped:  playing.Skipped(),
	}
}

// enrol takes the code, and refuses a duplicate rather than replacing a running
// room: two people are already at a board under that code, and the host that
// asked for it can be told no.
func (g *Registry) enrol(code wire.RoomCode, entry *handle) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, running := g.rooms[code]; running {
		return fmt.Errorf("a room is already running under the code %q", string(code))
	}
	g.rooms[code] = entry
	g.live++
	return nil
}

// lookup is the **one place a request path locks**.
func (g *Registry) lookup(code wire.RoomCode) (*handle, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, running := g.rooms[code]
	return entry, running
}

// unenrol removes a code and hands the entry to the caller that removed it, so
// that closing quit is exactly-once without a flag saying whether it happened.
func (g *Registry) unenrol(code wire.RoomCode) (*handle, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, running := g.rooms[code]
	if running {
		delete(g.rooms, code)
	}
	return entry, running
}

// unenrolAll empties the map and hands every entry back.
//
// The map is iterated here, and the order does not reach an output: the caller
// closes each entry's quit and returns a count.
func (g *Registry) unenrolAll() []*handle {
	g.mu.Lock()
	defer g.mu.Unlock()
	taken := make([]*handle, 0, len(g.rooms))
	for code, entry := range g.rooms {
		delete(g.rooms, code)
		taken = append(taken, entry)
	}
	return taken
}

// retired is the goroutine's teardown: the entry goes, the count comes down, and
// done is closed so that anything mid-send is refused rather than left waiting.
//
// The entry is only removed if it is **still this one**, because a code freed by
// a finished match may already have been opened again — and a goroutine deleting
// its successor's entry would take a live room off the map.
//
// It touches the mutex and sends on nothing. close is not a send: it cannot
// block, so closing under the lock serialises nobody.
func (g *Registry) retired(code wire.RoomCode, entry *handle) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.rooms[code] == entry {
		delete(g.rooms, code)
	}
	close(entry.done)
	g.live--
	g.idle.Broadcast()
}

// roomUnknown is a code naming no room this process is running, and it is the
// **registry's** refusal: internal/room's gate documents wire.CodeRoomUnknown as
// this file's, and until this file existed nothing in the repository sent it at
// all — a code that shipped dead.
//
// It names no seat, for the reason a refusal at the gate names none: a seat is a
// place in a room, so a code that names no room cannot name one either, and the
// transport answers the connection it read the message from.
func roomUnknown() []Outbound {
	return []Outbound{{Body: wire.Refused{Code: wire.CodeRoomUnknown}}}
}
