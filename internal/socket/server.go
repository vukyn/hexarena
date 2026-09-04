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
	// Told **after** the messages went out rather than before, so a caller
	// printing a line for a join and a line for the match starting prints them in
	// the order the room produced them. The seat is the room's own answer, so a
	// refused join reaches no callback at all — the returns above are every path
	// that hands no seat out.
	if s.joined != nil {
		s.joined(code, answered.Seat, hello.Name)
	}
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
