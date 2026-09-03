package socket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

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
}

// NewServer wraps a registry. The registry keeps owning the rooms; this owns the
// connections and the clock.
func NewServer(rooms *room.Registry, options Options) *Server {
	server := &Server{
		rooms:    rooms,
		timings:  options.Timings.withDefaults(),
		report:   options.Report,
		finished: options.Finished,
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

	seat, seated := s.join(ctx, code, entry, peer, hello)
	if !seated {
		return
	}
	// Whatever route this connection leaves by — a read error, a closed peer, a
	// finished match — the room is told once, here.
	defer s.left(code, entry, peer, seat)
	go s.keepalive(ctx, gone, peer)
	s.pump(ctx, code, entry, peer, seat)
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

// join is the gate, by code, and the seating that follows it.
//
// The seat is recorded **before** the batch goes out, because the second peer's
// join is answered with a wire.Start for *both* seats and the first one's
// connection has to be findable by then.
func (s *Server) join(ctx context.Context, code wire.RoomCode, entry *table, peer *connection, hello wire.Hello) (wire.Seat, bool) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	answered, err := s.rooms.Join(code, hello)
	if err != nil {
		s.failed(fmt.Errorf("join room %s: %w", code, err))
		return "", false
	}
	if !answered.Known || !answered.Seat.Valid() {
		// An unknown code, or the room's own refusal. Either way the one message
		// names no seat, so it goes to the connection it was read from and the
		// connection ends.
		s.send(ctx, entry, peer, answered.Out)
		return "", false
	}
	entry.seat(answered.Seat, peer)
	s.send(ctx, entry, peer, answered.Out)
	s.settled(ctx, code, entry, answered)
	return answered.Seat, true
}

// pump is the connection's read loop: one message in, the room's answer out.
func (s *Server) pump(ctx context.Context, code wire.RoomCode, entry *table, peer *connection, seat wire.Seat) {
	for {
		body, err := peer.read(ctx)
		switch {
		case err == nil:
		case errors.Is(err, errUnreadable):
			s.failed(peer.refuse(ctx, wire.CodeUnknownMessage))
			continue
		default:
			if !ended(err) {
				s.failed(fmt.Errorf("read from %s of room %s: %w", seat, code, err))
			}
			return
		}
		if over := s.deliver(ctx, code, entry, peer, seat, body); over {
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
func (s *Server) deliver(ctx context.Context, code wire.RoomCode, entry *table, peer *connection, seat wire.Seat, body wire.Body) bool {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	answered, err := s.rooms.Deliver(code, seat, body)
	if err != nil {
		s.failed(fmt.Errorf("deliver a %s from %s of room %s: %w", body.Kind(), seat, code, err))
		return true
	}
	if !answered.Known {
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
	s.settled(ctx, code, entry, answered)
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
		// on.
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
}

// left tells the room a peer went away, which with no rejoin is a match ending.
//
// The seat is freed **before** the room is told, so the wire.Closed the room
// addresses to the other seat cannot be delivered to the connection that has
// just gone — and so that a seat freed before the first battle (which is what
// room.Left does then) is free here too, for the next joiner.
func (s *Server) left(code wire.RoomCode, entry *table, peer *connection, seat wire.Seat) {
	entry.exchange.Lock()
	defer entry.exchange.Unlock()
	entry.free(seat, peer)
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
		return
	}
	// A departure's own context is the connection's, and that has been cancelled
	// by now — so the peer still there is written to on a fresh one, bounded by
	// the write timeout like every other message.
	s.send(context.Background(), entry, nil, answered.Out)
	s.settled(context.Background(), code, entry, answered)
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
