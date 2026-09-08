package socket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// Server is the WebSocket side of a room.Registry: an http.Handler that turns a
// connection into a seat and carries messages between the two.
//
// It opens no listener and picks no port — a caller mounts it — because the host
// binary is where the flag and output decisions live. → the package comment.
//
// # What it holds that the registry deliberately does not
//
// The registry has no sockets, so a second map is unavoidable: the room's
// answers are addressed **per seat** (room.Outbound names one), and only
// something holding both connections can deliver them. That is what a table is,
// and it is the whole of this type's own state.
type Server struct {
	rooms   *room.Registry
	timings Timings
	// report is where an error goes that no peer can be told about — a write that
	// failed, a room that refused an input. It is the caller's, and it is
	// **errors only**: nothing here logs a message, and nothing here may ever be
	// handed a wire.Hello. → TestTheTransportOwnsTheClockAndPrintsNothing.
	report func(error)
	// finished is called once per match that ends, with the room's own last
	// reading. It exists because a room retires its entry the moment its match
	// ends, so a caller asking *afterwards* would be asking about a room that had
	// already gone — the result travels on the answer to the input that ended it,
	// and this is where that answer is handed on. → room.Answer.
	finished func(wire.RoomCode, room.Reading)
	// joined is called once per seat a room hands out, with the name the peer
	// announced. It exists because a join leaves no other trace a caller can
	// reach: room.Reading carries no seat occupancy, so a host binary wanting to
	// print a line as each player arrives could otherwise only poll for the
	// *match* starting, which is one line for two people. → Options.Joined.
	joined func(wire.RoomCode, wire.Seat, string)

	mux *http.ServeMux

	// mu guards tables, and nothing else.
	mu     sync.Mutex
	tables map[wire.RoomCode]*table
}

// Options is what a Server is handed rather than what it decides.
type Options struct {
	// Timings is the transport's clock. The zero value takes every default.
	Timings Timings
	// Report is where an error nobody can be told about goes. Nil discards.
	Report func(error)
	// Finished is called once per match that ends, with the room's last reading.
	// Nil ignores it. → the field on Server for why it is a callback.
	Finished func(wire.RoomCode, room.Reading)
	// Joined is called once per seat a room hands out, with the seat and the name
	// the peer announced. Nil ignores it.
	//
	// ⚠️ **The name is the peer's own and is not checked for anything.** It is a
	// string a stranger on the network chose, so a caller printing it is printing
	// somebody else's bytes; nothing here trims, folds or bounds it, because
	// wire.Hello is the format and this is the transport handing a field on
	// unchanged. A caller that draws it owes it the same treatment as any other
	// text it did not write.
	//
	// ⚠️ It is called **under the room's exchange lock**, like Finished, so a
	// callback that blocks holds its own room's next message. Printing a line is
	// what it is for.
	Joined func(wire.RoomCode, wire.Seat, string)
}

// NewServer wraps a registry. The registry keeps owning the rooms; this owns the
// connections and the clock.
func NewServer(rooms *room.Registry, options Options) *Server {
	server := &Server{
		rooms:    rooms,
		timings:  options.Timings.withDefaults(),
		report:   options.Report,
		finished: options.Finished,
		joined:   options.Joined,
		tables:   make(map[wire.RoomCode]*table),
	}
	server.mux = http.NewServeMux()
	server.mux.HandleFunc(roomPattern, server.serve)
	return server
}

// ServeHTTP is the handler. Every path but roomPattern is a 404, which is what a
// client dialling a server that is not this one should get.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// failed hands an error to the caller's sink, if there is one.
func (s *Server) failed(err error) {
	if err == nil || s.report == nil {
		return
	}
	s.report(err)
}

// serve is one connection, from the upgrade to the departure.
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	// The default AcceptOptions are deliberate: origin verification is left on,
	// which costs a Go client nothing (it sends no Origin header) and is the one
	// protection a browser-borne client would want. Compression is off, which is
	// the library's default and right for a LAN.
	raw, err := websocket.Accept(w, r, nil)
	if err != nil {
		// Accept has already written a response, so there is no peer left to
		// refuse in the protocol's own terms.
		s.failed(fmt.Errorf("accept a connection: %w", err))
		return
	}
	peer := newConnection(raw, s.timings)
	defer peer.drop()

	// ⚠️ Not r.Context(): net/http may cancel a hijacked request's context, and
	// the library's own documentation warns against reading it after Accept. This
	// one is cancelled by the departure below and by the keepalive giving up.
	ctx, gone := context.WithCancel(context.Background())
	defer gone()

	code := roomOf(r.PathValue("code"))
	hello, read := s.firstHello(ctx, peer)
	if !read {
		return
	}
	entry := s.claim(code)
	defer s.release(code, entry)

	admitted, kept := s.join(ctx, gone, code, entry, peer, hello)
	if !kept {
		return
	}
	// Whatever route this connection leaves by — a read error, a closed peer, a
	// finished match — it is given back once, here: a seat to the room, and a
	// watcher's place to the table that was holding it.
	defer s.parted(code, entry, peer, admitted)
	go s.keepalive(ctx, gone, peer)
	s.pump(ctx, code, entry, peer, admitted)
}

// parted is the one departure every connection takes, whichever kind it was.
//
// ⚠️ **A watcher must not reach `left`**, and it is a branch here rather than a
// guard inside that function because the two are different events. `left` tells
// the *room* that a seat went away, which with no rejoin ends the match; a
// watcher took no seat, so there is nothing to tell the room — it never knew one
// was there — and what goes back is a place at the table and a cursor. Passing a
// watcher's empty seat to room.Left would be reporting a departure the room
// would have to decide what to do with.
func (s *Server) parted(code wire.RoomCode, entry *table, peer *connection, admitted room.Admission) {
	if admitted.Watching {
		entry.unwatch(peer)
		return
	}
	s.left(code, entry, peer, admitted.Seat)
}

// speaking is what to call a connection in a message reported to the caller: the
// seat it took, or that it is one of the people watching.
//
// A watcher has no name here on purpose. wire.Hello.Name is a string a stranger
// chose and this transport hands it to the Joined callback unchanged rather than
// putting it into its own errors.
func speaking(admitted room.Admission) string {
	if admitted.Watching {
		return "a watcher"
	}
	return string(admitted.Seat)
}

// roomOf is the map key a pasted code names, and the reason it decodes rather
// than passing the characters through is a **correct-looking refusal**.
//
// wire.RoomCode.Decode upper-cases before it decodes, because the base32
// alphabet is upper-case only and the fold is therefore total — so a player who
// typed their code in lower case has a perfectly good code. The registry keys
// its map on the string, though, and every key in it came out of
// wire.EncodeRoom, so a lower-case code would look up a key that is not there
// and the player would be told the room is unknown *while the room sat right
// there*. Decoding and re-encoding is what turns what was typed into the key.
//
// ⚠️ **An undecodable code is NOT refused here**, and that is deliberate. It is
// handed to the registry as it stands, where it is the key of no room and
// answers wire.CodeRoomUnknown — which is *the registry's own refusal*, the one
// declaration of it in the repository. Producing a second one here would be this
// package spelling a refusal the layer below it owns, and the answer would be
// identical. The registry's map cannot contain a string EncodeRoom never
// produced, so the answer is right by construction rather than by agreement.
func roomOf(pasted string) wire.RoomCode {
	code := wire.RoomCode(pasted)
	at, which, err := code.Decode()
	if err != nil {
		return code
	}
	canonical, err := wire.EncodeRoom(at, which)
	if err != nil {
		return code
	}
	return canonical
}

// firstHello reads the one message a connection may open with.
//
// ⚠️ **Nothing about the hello is reported, ever.** A hello carries the room's
// password in the clear — wire.Password redacts itself under every fmt verb, so
// a decoded one is safe, and an *undecodable* one is bytes with no type left to
// do the redacting. → errUnreadable, which is why an unreadable message is a
// sentinel and a byte count rather than the decoder's own error.
func (s *Server) firstHello(ctx context.Context, peer *connection) (wire.Hello, bool) {
	body, err := peer.read(ctx)
	switch {
	case err == nil:
	case errors.Is(err, errUnreadable):
		// A peer one version ahead, or a mangled frame. Answered in the
		// protocol's own terms; the bytes are not reported.
		s.failed(peer.refuse(ctx, wire.CodeUnknownMessage))
		return wire.Hello{}, false
	default:
		// A peer that went away before it said anything is not a fault.
		if !ended(err) {
			s.failed(fmt.Errorf("read a hello: %w", err))
		}
		return wire.Hello{}, false
	}
	switch message := body.(type) {
	case *wire.Hello:
		return *message, true
	case wire.Hello:
		return message, true
	}
	// A kind this protocol has that does not belong here — an act before a join,
	// or a peer speaking the server's own direction.
	s.failed(peer.refuse(ctx, wire.CodeUnknownMessage))
	return wire.Hello{}, false
}

// join is the gate, by code, and the seating — or the watching — that follows
// it.
//
// The seat is recorded **before** the batch goes out, because the second peer's
// join is answered with a wire.Start for *both* seats and the first one's
// connection has to be findable by then.
//
// ⚠️ **What it branches on is room.Admission.Watching and never an empty seat.**
// A refusal and a welcomed watcher both leave the seat empty — those are the
// three answers a wire.Seat cannot hold, which is why Admission exists — so
// `!Seat.Valid()` would close the connection of everybody who came to watch. Nor
// may it read the welcome back out of answered.Out: that is a transport parsing
// its own output, the mistake Answer.Known exists to prevent, one message down.
func (s *Server) join(ctx context.Context, stop func(), code wire.RoomCode, entry *table,
	peer *connection, hello wire.Hello) (room.Admission, bool) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	answered, err := s.rooms.Join(code, hello)
	if err != nil {
		s.failed(fmt.Errorf("join room %s: %w", code, err))
		return room.Admission{}, false
	}
	if !answered.Known || (!answered.Watching && !answered.Seat.Valid()) {
		// An unknown code, or the room's own refusal. Either way the one message
		// names no seat, so it goes to the connection it was read from and the
		// connection ends.
		s.send(ctx, entry, peer, answered.Out)
		return room.Admission{}, false
	}
	if answered.Watching {
		return s.watched(ctx, stop, code, entry, peer, answered)
	}
	// ⚠️ **The window is stopped before anything is written, and stopping it is
	// what makes this a rejoin rather than a race.** The room has already given
	// the seat back — it matched a token, which is the only way Rejoined is true —
	// so the timer still running is one that would tell the room this client left
	// a match it is sitting in. The caller holds exchange, which is what lets a
	// stopped timer and a re-seated connection be one act.
	if index, seated := indexOfSeat(answered.Seat); seated {
		entry.rejoinable[index] = answered.Rejoinable
	}
	if answered.Rejoined {
		entry.release(answered.Seat)
	}
	entry.seat(answered.Seat, peer)
	s.send(ctx, entry, peer, answered.Out)
	// ⚠️ **After the welcome and before anything else**, which is the order a
	// mirror was built for: it takes its seat off the welcome and then applies
	// bodies, so a record arriving first would be replayed by a mirror that does
	// not yet know which half it plays. Same order the watcher path uses, and for
	// the same reason.
	if len(answered.Resumed) > 0 {
		if err := peer.send(ctx, answered.Resumed...); err != nil {
			if !ended(err) {
				s.failed(fmt.Errorf("hand %s the %d recorded bodies of room %s: %w",
					answered.Seat, len(answered.Resumed), code, err))
			}
			return room.Admission{}, false
		}
	}
	s.settled(ctx, code, entry, answered)
	// The second seat's join is what opens the first battle, so it is an exchange
	// that records — the wire.Start every watcher already attached is owed.
	s.forward(ctx, entry, answered)
	// Told **after** the messages went out rather than before, so a caller
	// printing a line for a join and a line for the match starting prints them in
	// the order the room produced them. The seat is the room's own answer, so a
	// refused join reaches no callback at all — the returns above are every path
	// that hands no seat out, and a watcher is not a seat being handed out.
	if s.joined != nil {
		s.joined(code, answered.Seat, hello.Name)
	}
	return answered.Admission, true
}

// watched is a welcomed watcher: the cap, the welcome, everything the match has
// recorded so far, and a place at the table.
//
// The caller holds exchange, which is what makes the order below a property
// rather than a hope. Three things happen under one lock — the room is asked for
// its whole record, that record goes down this socket, and the connection is
// added to the table — and every body recorded after this point is produced by
// an exchange that has to take the same lock. So a watcher joining mid-match is
// handed the record from nought **before** it can be handed anything new, and
// the first fan-out that reaches it starts exactly where its catch-up stopped:
// nothing twice, nothing skipped, and not by arranging the calls carefully but
// because no other body can be recorded while this runs.
//
// ⚠️ **The cap is checked after the room has answered, and the welcome it
// produced is then dropped.** The gate's order is version, password, and only
// then anything about how full anything is — so a watcher on a stale build hears
// about the build rather than about a queue — and a welcome followed by a
// refusal would be two answers to one hello. → wire.CodeTooManyWatchers.
func (s *Server) watched(ctx context.Context, stop func(), code wire.RoomCode, entry *table,
	peer *connection, answered room.Answer) (room.Admission, bool) {
	if entry.full() {
		// The transport's own refusal, because the cap is the transport's own
		// fact: the room keeps no count of watchers and could not answer this.
		// It is still a code and not a sentence — nothing here words a refusal.
		s.failed(peer.refuse(ctx, wire.CodeTooManyWatchers))
		return room.Admission{}, false
	}
	s.send(ctx, entry, peer, answered.Out)
	// Nought, always: the one cursor this transport ever hands a room. → the
	// watcher's own cursor, and room.Answer.Watched, for why a cursor of its own
	// could reach a room that had been reissued the code and panic on that room's
	// goroutine.
	recorded, cursor, known := s.rooms.Since(code, 0)
	if !known {
		// The match ended between the welcome and the read, which is the same
		// ordinary race a seated peer's deliver reads as "the room has gone".
		// There is nothing to say about it: this connection has no match to be
		// told the result of, and the room is not there to be asked again.
		peer.bye(websocket.StatusNormalClosure, "the match is over")
		return room.Admission{}, false
	}
	if len(recorded) > 0 {
		if err := peer.send(ctx, recorded...); err != nil {
			if !ended(err) {
				s.failed(fmt.Errorf("hand a watcher the %d recorded bodies of room %s: %w",
					len(recorded), code, err))
			}
			return room.Admission{}, false
		}
	}
	entry.admit(peer, cursor, stop)
	return answered.Admission, true
}

// forward hands every watching connection the bodies an exchange recorded.
//
// ⚠️ **The bodies come off the answer to the input that recorded them, not from
// a read taken afterwards, and that is the one thing about this design that is
// not obvious.** A room retires its own entry the moment its match ends, so the
// exchange that records a match's last wire.Turn is the exchange that takes the
// record away: a transport that answered its players and then asked for the
// record was asking a room that had already gone. Measured before it was
// changed — every run of a whole match handed a reader 62 of its 63 turns, with
// no race to lose, because retiring beats a socket write every time. →
// room.Answer.Watched.
//
// The caller holds exchange, so the order bodies reach a watcher is the order the
// room produced them in — the same guarantee the two seats get, from the same
// lock.
//
// ⚠️ **A watcher whose cursor is not where the record says it should be is ended
// rather than caught up**, and that is what stands in for the range guard the
// room deliberately does not have. Room.Since panics on a cursor its record
// cannot answer, on the room's own goroutine, because answering an out-of-range
// cursor with an empty read would make a consumer that has got ahead look exactly
// like one that is up to date. The same reasoning applies one layer out with a
// different remedy: this transport never hands a room a cursor, so it can neither
// panic nor be caught up quietly — a watcher out of step is a watcher reading a
// match this table is no longer serving, and the honest answer is to end it.
func (s *Server) forward(ctx context.Context, entry *table, answered room.Answer) {
	if len(answered.Watched) == 0 || len(entry.watchers) == 0 {
		return
	}
	from := answered.Cursor - len(answered.Watched)
	// Filtered in place, which is safe because the write index never runs ahead
	// of the read one: whoever is ended here is dropped from the table now, and
	// its own departure finds nothing left to give back.
	kept := entry.watchers[:0]
	for _, watching := range entry.watchers {
		if watching.cursor != from {
			s.failed(fmt.Errorf("a watcher stands at %d of a record that reaches %d, so it is "+
				"reading a match this table is no longer serving", watching.cursor, from))
			watching.stop()
			continue
		}
		if err := watching.peer.send(ctx, answered.Watched...); err != nil {
			if !ended(err) {
				s.failed(fmt.Errorf("hand a watcher %d recorded bodies: %w", len(answered.Watched), err))
			}
			watching.stop()
			continue
		}
		watching.cursor = answered.Cursor
		kept = append(kept, watching)
	}
	entry.watchers = kept
}

// endWatching lets go of every watcher on a table, which is what a match ending
// does to the cursors it leaves behind.
//
// ⚠️ **A cursor must not outlive its room**, and this is where that is paid: a
// finished room gives its code back to the lowest-free-byte allocator, and a
// table is keyed by code and outlives its match by however long the sockets take
// to close — so a watcher left attached could be handed a *fresh* room's bodies
// at a cursor that was never about it. Ending them here is the first of the two
// answers to that; the second is forward's check, which catches it if this is
// ever missed on a path nobody thought of.
//
// It cancels rather than closing, so no socket is written to under the exchange
// lock: each watcher's own goroutine finds its read unblocked, tidies its
// connection up and gives its place back through the departure that runs behind
// every connection. Whatever was written to it before this line was written
// while the room was still there, which is the whole final turn of the match.
//
// ⚠️ **Measured: deleting the call on the *finished* path alone reddens
// nothing**, and that is worth knowing before somebody deletes it on purpose.
// Both players leave the moment a match ends, and their own departures reach
// here through `left` finding the room unknown — a few milliseconds later. What
// the finished path buys is that the drop is **prompt and its own** rather than
// a consequence of two sockets closing, and the case where the difference is
// real is a match ended by a *timeout* with both peers still connected and
// silent: nothing closes those sockets, so nothing would reach `left` either.
// Making this whole function a no-op does redden — the whole-match test waits
// out its bound — so what is untested is the promptness, not the drop.
func (s *Server) endWatching(entry *table) {
	for _, watching := range entry.watchers {
		watching.stop()
	}
	entry.watchers = nil
}

// pump is the connection's read loop: one message in, the room's answer out.
//
// ⚠️ **A watcher reads the same loop, and what it sends goes to the room like
// anything else.** room.Deliver already refuses a peer with no seat, with
// wire.CodeNotYourTurn, and that refusal leaves the prompt open and the battle
// where it was — so a transport that dropped a watcher's act itself, or answered
// it here, would be declaring a rule the room owns for a second time.
func (s *Server) pump(ctx context.Context, code wire.RoomCode, entry *table, peer *connection,
	admitted room.Admission) {
	for {
		body, err := peer.read(ctx)
		switch {
		case err == nil:
		case errors.Is(err, errUnreadable):
			s.failed(peer.refuse(ctx, wire.CodeUnknownMessage))
			continue
		default:
			if !ended(err) {
				s.failed(fmt.Errorf("read from %s of room %s: %w", speaking(admitted), code, err))
			}
			return
		}
		if over := s.deliver(ctx, code, entry, peer, admitted, body); over {
			// The match ended, so this connection's work is done: the peer has
			// computed the same ending from its own battle, and the close frame
			// is what says so on the wire.
			peer.bye(websocket.StatusNormalClosure, "the match is over")
			return
		}
	}
}

// deliver hands the room one message and reports whether the match ended with
// it.
//
// ⚠️ **A watcher's message reaches the room and nothing else here changes**, with
// one exception that is the whole of what "a watcher cannot disturb the match"
// means at this layer: the allowance is **not** re-armed for it. Every exchange a
// *seat* has re-arms the clock on whoever the room is waiting for, which is
// right — the room's own reading is what the clock is set from — but a watcher's
// exchange changes nothing about the room, so re-arming would hand the seat on
// turn a fresh allowance for somebody else's message. A spectator sending an act
// a turn could then keep a stalling player alive for ever, and nothing in a
// suite whose clients answer would see it. → allowance.armed.
func (s *Server) deliver(ctx context.Context, code wire.RoomCode, entry *table, peer *connection,
	admitted room.Admission, body wire.Body) bool {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	answered, err := s.rooms.Deliver(code, admitted.Seat, body)
	if err != nil {
		s.failed(fmt.Errorf("deliver a %s from %s of room %s: %w",
			body.Kind(), speaking(admitted), code, err))
		s.endWatching(entry)
		return true
	}
	if !answered.Known {
		// The room has gone, so every cursor this table holds is about a record
		// that no longer exists. → endWatching.
		s.endWatching(entry)
		// ⚠️ **Not forwarded**, and this is the same division `left` draws.
		// wire.CodeRoomUnknown is the registry's refusal for a **joiner** — a
		// code naming no room this process is running — and this peer was
		// *seated*: its room existed and has ended. Telling it the room is
		// unknown would be saying something untrue about a match it just played,
		// and it needs no telling either way, because it computes an ordinary
		// ending itself and is sent wire.Closed for the one it cannot.
		return true
	}
	s.send(ctx, entry, peer, answered.Out)
	if !admitted.Watching {
		s.settled(ctx, code, entry, answered)
	}
	s.forward(ctx, entry, answered)
	if answered.Reading.Finished {
		s.endWatching(entry)
	}
	return answered.Reading.Finished
}

// timedOut is what an allowance running out becomes: the transport reporting it,
// exactly as room.TimedOut takes it.
//
// ⚠️ **A timer that fires while an answer is in flight is normal, not an error.**
// The room refuses a timeout for a seat it is not asking, with
// wire.CodeNotYourTurn, so a late report is already harmless — and this must not
// treat that refusal as a reason to close anything. Getting it wrong drops a
// player for answering quickly.
//
// The refusal is also **not forwarded**. The transport owns the timeout, so it
// owns the answer to it: a wire.Refused the client never provoked would be a
// refusal of a question it never asked. Anything else in the answer is the
// room's real business and goes out as usual.
func (s *Server) timedOut(ctx context.Context, code wire.RoomCode, entry *table, seat wire.Seat) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	answered, err := s.rooms.TimedOut(code, seat)
	if err != nil {
		s.failed(fmt.Errorf("report %s's allowance in room %s: %w", seat, code, err))
		return
	}
	if !answered.Known {
		// The room went away between the timer being armed and it firing, which
		// is an ordinary race: a match that ended is a match nobody is waiting
		// on. Its watchers' cursors go with it. → endWatching.
		s.endWatching(entry)
		return
	}
	if refusedAlone(answered.Out, wire.CodeNotYourTurn) {
		entry.late++
		entry.allowance.set(answered.Reading, seat, s.reporter(ctx, code, entry))
		return
	}
	// There is no connection to answer a message that names no seat here: a
	// timeout is the transport's own input, so nothing in the answer may be
	// addressed to "whoever asked".
	s.send(ctx, entry, nil, answered.Out)
	s.settled(ctx, code, entry, answered)
	// A timeout resolves a turn like a decision does, so it records like one.
	s.forward(ctx, entry, answered)
	if answered.Reading.Finished {
		s.endWatching(entry)
	}
}

// left is a peer's socket closing, and it is now two different things.
//
// ⚠️ **A seat that can be come back to is HELD rather than reported**, which is
// the whole of the rejoin: the transport cannot tell a wifi hiccup from somebody
// walking away — a socket closing is a socket closing — so with no window every
// blip killed a match. The room is not told, so the board, the series and the
// clock stay exactly as they were and a client that returns needs nothing
// rebuilt; what is given up is the connection, and if the allowance runs out
// meanwhile the room passes that turn as it would for anybody thinking too long.
//
// The seat is freed **before** either branch, so the wire.Closed the room
// addresses to the other seat cannot be delivered to the connection that has just
// gone — and so that a seat freed before the first battle (which is what
// room.Left does then) is free here too, for the next joiner.
func (s *Server) left(code wire.RoomCode, entry *table, peer *connection, seat wire.Seat) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	entry.free(seat, peer)
	if entry.hold(seat, s.timings.RejoinWindow, func() { s.windowClosed(code, entry, seat) }) {
		return
	}
	s.abandon(code, entry, seat)
}

// windowClosed is the reconnect window running out: the client did not come back,
// so the room hears what it would have heard at once before rejoins existed.
//
// It takes exchange itself, because it runs on the timer's own goroutine rather
// than on the departing connection's.
func (s *Server) windowClosed(code wire.RoomCode, entry *table, seat wire.Seat) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	// ⚠️ **The race is decided here rather than in Stop.** A hello that arrived
	// while this timer was already firing released the window and re-seated the
	// connection, and telling the room that seat left would end a match somebody
	// is sitting in. `held` is false in exactly that case, because release clears
	// the slot under the same lock this is holding.
	if !entry.held(seat) {
		return
	}
	entry.release(seat)
	s.abandon(code, entry, seat)
}

// abandon tells the room a peer went away, which is a match ending. The caller
// holds exchange.
func (s *Server) abandon(code wire.RoomCode, entry *table, seat wire.Seat) {
	answered, err := s.rooms.Left(code, seat)
	if err != nil {
		s.failed(fmt.Errorf("report that %s left room %s: %w", seat, code, err))
		return
	}
	if !answered.Known {
		// The room had already gone, which every ordinary ending reaches: the
		// match finished, the room retired its entry, and this is the socket
		// closing behind it. The wire.CodeRoomUnknown in that answer is for a
		// *joiner* and must not be forwarded to a peer that was seated.
		s.endWatching(entry)
		return
	}
	// A departure's own context is the connection's, and that has been cancelled
	// by now — so the peer still there is written to on a fresh one, bounded by
	// the write timeout like every other message.
	s.send(context.Background(), entry, nil, answered.Out)
	s.settled(context.Background(), code, entry, answered)
	// ⚠️ **A departure is the one ending a watcher cannot compute**, which is why
	// the room records its wire.Closed{ClosureLeft} beside the one it addresses
	// to the seat still there: a match played out to its end needs no message,
	// and this one has no Ended event and no further wire.Start behind it.
	s.forward(context.Background(), entry, answered)
	if answered.Reading.Finished {
		s.endWatching(entry)
	}
}

// settled is what happens after every batch: the allowance is re-armed off the
// room's own reading, and a match that ended is handed to the caller.
//
// ⚠️ **The allowance is started on the seat the room says it is Awaiting**, which
// is the whole of the room's side of the clock — and it is never set across a
// Skipped prompt, because the room walks past those itself. That is a property of
// the state machine rather than a rule this has to remember; it is measured
// rather than assumed (→ TestASkippedPromptStartsNoClockOverASocket).
func (s *Server) settled(ctx context.Context, code wire.RoomCode, entry *table, answered room.Answer) {
	entry.allowance.set(answered.Reading, "", s.reporter(ctx, code, entry))
	if answered.Reading.Finished {
		entry.ended(code, answered.Reading, s.finished)
	}
}

// reporter is the callback a timer fires: the seat whose allowance ran out.
//
// ⚠️ The context is **not** the one the timer will fire under. A connection's
// context dies with the connection, and the seat whose allowance runs out may be
// the one that has gone quiet — so the write goes out on a background context,
// bounded per message by the write timeout. The connection's own ctx is still
// what the *keepalive* and the reads use.
func (s *Server) reporter(_ context.Context, code wire.RoomCode, entry *table) func(wire.Seat) {
	return func(seat wire.Seat) {
		s.timedOut(context.Background(), code, entry, seat)
	}
}

// keepalive is the liveness half of the close threshold: a ping every interval,
// each bounded by the threshold, and a peer that does not answer is a peer that
// has gone.
//
// ⚠️ This is the only thing in the package that decides a peer is absent without
// the socket saying so, and with no rejoin that decision **ends a match**.
// → DefaultCloseThreshold, where what it is guarding is written down.
func (s *Server) keepalive(ctx context.Context, gone func(), peer *connection) {
	ticker := newTicker(s.timings.Keepalive)
	defer ticker.stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.each():
		}
		if err := peer.ping(ctx); err != nil {
			// Cancelling the context closes the connection, which unblocks the
			// read this connection's pump is sitting in — so the departure is
			// reported by the one deferred call that reports every departure.
			gone()
			return
		}
	}
}

// send hands each of the room's messages to the connection it names.
//
// A message naming **no seat** is a refusal at the gate, which goes to the
// connection the hello was read from — `from`, which is nil on the paths where
// there is no such connection (a timeout, a departure) precisely so that a
// message with nowhere to go is reported rather than delivered to whoever
// happened to be at hand.
func (s *Server) send(ctx context.Context, entry *table, from *connection, out []room.Outbound) {
	for _, message := range out {
		target := from
		if message.To.Valid() {
			target = entry.at(message.To)
		}
		if target == nil {
			if message.To.Valid() {
				// The seat's connection has gone — a departure's own peer, which
				// the room does not address, so this is a real gap rather than
				// the ordinary case.
				s.failed(fmt.Errorf("a %s for %s has no connection", message.Body.Kind(), message.To))
				continue
			}
			s.failed(fmt.Errorf("a %s names no seat and no connection asked for it", message.Body.Kind()))
			continue
		}
		if err := target.send(ctx, message.Body); err != nil && !ended(err) {
			s.failed(fmt.Errorf("send a %s to %s: %w", message.Body.Kind(), message.To, err))
		}
	}
}

// refusedAlone reports whether a batch is exactly one wire.Refused carrying the
// given code, which is the shape room.TimedOut answers a seat it is not asking.
//
// It is a precise discriminator rather than a search: the room answers a refused
// timeout with one Outbound and a resolved one with a wire.Turn to each seat, so
// the two cannot be confused.
func refusedAlone(out []room.Outbound, code wire.Code) bool {
	if len(out) != 1 {
		return false
	}
	switch refusal := out[0].Body.(type) {
	case *wire.Refused:
		return refusal.Code == code
	case wire.Refused:
		return refusal.Code == code
	}
	return false
}

// claim is the table for a code, made if there is none, with this connection's
// reference counted.
func (s *Server) claim(code wire.RoomCode) *table {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, running := s.tables[code]
	if !running {
		entry = &table{}
		s.tables[code] = entry
	}
	entry.holders++
	return entry
}

// release gives this connection's reference back, and the table goes with the
// last one — along with any timer still armed on it, because a timer outliving
// the room it reports to is a goroutine nobody will collect.
func (s *Server) release(code wire.RoomCode, entry *table) {
	s.mu.Lock()
	last := false
	entry.holders--
	if entry.holders <= 0 && s.tables[code] == entry {
		delete(s.tables, code)
		last = true
	}
	s.mu.Unlock()
	if last {
		entry.allowance.stop()
	}
}

// Tables is how many rooms this server is holding connections for, which is not
// the registry's Count: a table outlives the match by however long the two
// sockets take to close, and a room with nobody connected has no table at all.
//
// It exists so a shutdown can be measured rather than assumed — the same
// argument room.Registry.Running is exposed under.
func (s *Server) Tables() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tables)
}

// settlingPoll is how often Shutdown asks whether the last table has gone.
//
// ⚠️ It is a poll and there is genuinely nothing to wait on, which is the gap
// this whole function was filed under. A table is released by the connection
// goroutine that held it, on its way out; nothing signals that, and a channel
// closed by "the last connection" would be a second lifetime beside the holder
// count that already owns one. The registry's own half of the shutdown is *not*
// polled — Registry.Wait is a proper condition variable — so this is the one
// approximation in the call, and it is bounded by the caller's context rather
// than by a count.
//
// Ten milliseconds is what internal/socket's own end-to-end fixture already
// polls Tables at, so a shutdown notices a settled server about as fast as a
// test does, and a hundred of them is one second of a bound that is measured in
// seconds.
const settlingPoll = 10 * time.Millisecond

// Shutdown stops serving: it tells every connected peer why, stops every room,
// and returns when nothing is left running.
//
// ⚠️ **This exists because http.Server.Shutdown cannot do it.** That call waits
// for connections it can see finish a *request*, and a WebSocket is
// **hijacked** — net/http has handed the connection over and stopped counting
// it — so shutting the http server down leaves every socket open and every room
// goroutine alive. A caller that only closed its listener would exit with a
// match still being played inside it.
//
// # It is four steps and they are in this order for reasons, not for tidiness
//
//  1. **Tell every connected peer**, with wire.ClosureStopped and then a close
//     frame. The message goes first because it is the only thing that says
//     *why*: a socket that simply dies leaves a player staring at a dead
//     connection, and a client's mirror reads a Closed as an ending it could not
//     have computed. The close frame goes after because a shutdown must not
//     depend on the peer's cooperation — a wedged client that never reads the
//     Closed still has its socket taken away, so the tables below empty on this
//     side's own initiative.
//  2. **room.Registry.CloseAll**, which stops the rooms.
//  3. **room.Registry.Wait**, bounded by ctx. ⚠️ Two calls rather than one, for
//     the reason Wait's own comment gives: Wait **closes nothing**, so it is a
//     measurement rather than a tidy-up, and a goroutine left behind hangs it
//     instead of being quietly collected. Merging the two would lose exactly
//     that property.
//  4. **Both readings at nought.** Tables and Running measure different things —
//     a table outlives its match by however long two sockets take to close, and
//     a room that has retired its entry but not returned is still a goroutine —
//     so a shutdown that checked one of them would return with the other still
//     going. They are exposed for this and are used for this.
//
// ⚠️ **CloseAll runs even on a context that is already done.** Only the *waiting*
// is bounded: closing is the point of the call, and a shutdown that skipped it
// because it was out of time would leave behind exactly what it was asked to
// stop. So an expired context gives up on the wait, not on the work — and the
// error names what was still running, because a caller printing "gave up" with
// no numbers tells its user nothing.
//
// It is not safe for two callers at once, and it does not need to be: a binary
// has one shutdown, reached from the match ending or from a signal, and running
// both would only send a second Closed down a socket that had already gone.
func (s *Server) Shutdown(ctx context.Context) error {
	s.stopping(ctx)
	s.rooms.CloseAll()
	if err := s.waited(ctx); err != nil {
		return err
	}
	return s.settling(ctx)
}

// stopping is step one: every peer this server is still holding is told the host
// has stopped, and then has its socket closed.
//
// The table's own exchange lock is taken per room, which is what makes reading
// the two connections safe against the goroutines that seat and free them — and
// it is per room rather than server-wide for the reason table.exchange carries,
// so a stuck peer delays its own room's notice and no other's.
func (s *Server) stopping(ctx context.Context) {
	for _, held := range s.held() {
		entry := held.table
		entry.exchange.Lock()
		// ⚠️ **Every reconnect window is stopped first**, and stopping them is
		// not tidiness. A window that fired after this point would call
		// Registry.Left on a room CloseAll has already retired, and would do it
		// on a timer's goroutine after Shutdown had returned — so the one thing
		// this function promises, that nothing is left running, would be false
		// for up to a whole window. There is also nothing left to hold a seat
		// FOR: the host is stopping, so the match this seat belongs to is over
		// whatever its client does next.
		entry.releaseAll()
		// Two connections in a fixed order rather than a walk over a collection:
		// a room has exactly two seats, and the order they are told in is an
		// output. → table, which is two fields for the same reason.
		for _, seated := range [seatsPerTable]struct {
			seat wire.Seat
			peer *connection
		}{{wire.SeatHost, entry.host}, {wire.SeatGuest, entry.guest}} {
			if seated.peer == nil {
				continue
			}
			if err := seated.peer.send(ctx, wire.Closed{Reason: wire.ClosureStopped}); err != nil && !ended(err) {
				s.failed(fmt.Errorf("tell %s of room %s the host stopped: %w", seated.seat, held.code, err))
			}
			// ⚠️ **drop and not bye, and the difference is five seconds a socket.**
			// bye is the close *handshake*: it writes a close frame and waits for
			// the peer's answer, and the library gives that wait five seconds. A
			// peer that is not reading — which is exactly the peer a shutdown has
			// to be robust against — never answers, so a graceful close costs the
			// full five seconds per connection and buys nothing: the wire.Closed
			// above has already said why, at the application level, and it is
			// already flushed to the socket before this line. Measured on the
			// four-connection shutdown test: 20.0s with bye, 0.2s with drop.
			//
			// What it costs is that a peer which read the Closed and then sat
			// there sees a reset rather than a close frame. A hexarena client does
			// not: Mirror.Over goes true on a Closed and Client.Play returns
			// before this end's socket is ever read again.
			seated.peer.drop()
		}
		// And everybody watching, after the two playing and in the order they
		// arrived. It is a walk over a collection where the seats are a
		// fixed-size array, which is the difference between the two: a room has
		// exactly two seats and any number of watchers up to MaxWatchers, so
		// there is no pair of names to walk here and no count to state twice.
		//
		// They are told the same thing for the same reason — a socket that simply
		// dies leaves a person staring at a dead connection — and dropped the same
		// way, because a shutdown must not depend on the peer's cooperation.
		for _, watching := range entry.watchers {
			if err := watching.peer.send(ctx, wire.Closed{Reason: wire.ClosureStopped}); err != nil && !ended(err) {
				s.failed(fmt.Errorf("tell a watcher of room %s the host stopped: %w", held.code, err))
			}
			watching.peer.drop()
		}
		entry.watchers = nil
		entry.exchange.Unlock()
	}
}

// held is every table this server holds, with its code, taken under the server's
// own mutex and copied out — so the notify above walks a snapshot rather than the
// live map, which connections are adding to and deleting from as it runs.
//
// ⚠️ It is **sorted by code**, because a map's iteration order must not reach an
// output and the order two rooms' players are told in is one. The engine's rule,
// one layer out; room.Registry.Codes sorts for the same reason.
func (s *Server) held() []codedTable {
	s.mu.Lock()
	out := make([]codedTable, 0, len(s.tables))
	for code, entry := range s.tables {
		out = append(out, codedTable{code: code, table: entry})
	}
	s.mu.Unlock()
	slices.SortFunc(out, func(a, b codedTable) int { return strings.Compare(string(a.code), string(b.code)) })
	return out
}

// codedTable is one entry of that snapshot: the map's key beside its value,
// because a table does not know its own code.
type codedTable struct {
	code  wire.RoomCode
	table *table
}

// waited is step three: room.Registry.Wait, bounded by the caller's context.
//
// ⚠️ Wait takes no context and cannot, because what it waits on is a
// sync.Cond — so the bound is a select against a goroutine, and on the losing
// path **that goroutine is still there**. It is not a leak that grows: it is
// blocked on the same condition the rooms will broadcast when they end, so it
// ends when they do, and CloseAll has already asked them to. What it costs is
// that a caller giving up here does not get its memory back until the wedged
// room does, which is the honest state of a process that is about to exit
// anyway.
func (s *Server) waited(ctx context.Context) error {
	settled := make(chan struct{})
	go func() {
		defer close(settled)
		s.rooms.Wait()
	}()
	select {
	case <-settled:
		return nil
	case <-ctx.Done():
		// ⚠️ **The reading decides, not the channel and not the select's own
		// choice.** A select over two ready cases picks one at random, and both
		// of these are ready whenever the last room ends around the moment the
		// bound does — so the same shutdown of the same server answered nil or
		// a refusal by coin flip, and the refusal it wrote on that path named
		// **nought** rooms and nought connections, which is the useless message
		// gaveUp exists to avoid. Asking the registry is also deterministic
		// where <-settled is not: the goroutine above may not have been
		// scheduled yet even when Wait would return at once.
		running := s.rooms.Running()
		if running == 0 {
			return nil
		}
		return s.gaveUp(ctx, running, s.Tables())
	}
}

// settling is step four: wait for the last table to go.
// ⚠️ **The two counts are read once per turn of the loop, and the same pair
// both decides and is reported.** Reading them again inside the refusal would
// re-open the window this loop exists to close: the last table can go between
// the test and the giving-up branch, and a refusal naming nought of everything
// still running says only that a clock ran out. There is no branch to get
// wrong here because there is only one reading.
func (s *Server) settling(ctx context.Context) error {
	for {
		rooms, tables := s.rooms.Running(), s.Tables()
		if rooms == 0 && tables == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return s.gaveUp(ctx, rooms, tables)
		case <-time.After(settlingPoll):
		}
	}
}

// gaveUp is the one refusal this shutdown has, and it **names both readings**
// rather than only the context's error: "the shutdown timed out" tells a host
// nothing it can act on, and "two rooms and one connection are still running"
// tells it whether the match is stuck or a socket is.
//
// The context's own error is wrapped rather than replaced, so a caller can still
// ask errors.Is whether it was a deadline or a cancellation — which are a bound
// being hit and a second ctrl-c, and read very differently.
//
// ⚠️ **The counts are passed in rather than read here, and that is the whole
// point of the parameters.** Both callers refuse *because of* a reading; taking
// a second one on the way to the message would let the refusal disagree with
// the decision that produced it, and the disagreement it produced in practice
// was "0 room(s) and 0 connected room(s) still running" — a give-up naming
// nothing to act on, which is the exact failure this function was written to
// avoid.
func (s *Server) gaveUp(ctx context.Context, rooms, tables int) error {
	return fmt.Errorf("stop serving: %d room(s) and %d connected room(s) still running: %w",
		rooms, tables, ctx.Err())
}
