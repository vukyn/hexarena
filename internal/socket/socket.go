// Package socket is the PvP transport: a WebSocket server around
// internal/room's registry, a dialling client, and the **mirror** that client
// needs in order to be a client at all.
//
// It is the one boundary the WebSocket dependency is allowed to cross.
// internal/wire is the format with no I/O in it and internal/room is a state
// machine with none either — both refuse `github.com/coder/websocket` by name in
// an AST walk — so everything that opens, reads, writes or closes a connection
// is here and nothing of it leaks back.
//
// # ⚠️ This package owns the clock, and it is the only clock in the PvP stack
//
// internal/room and internal/wire both refuse to import `time`, held
// mechanically, because *whoever owns the transport owns the countdown*: the
// room takes a timeout as an **input** and never asks what time it is. This
// package is the owner. So `time` appears here and nowhere else in the three,
// and every duration below is a decision this package makes rather than one it
// reads off the wire.
//
// What the room hands over is a number: Reading.Config.Allowance, **seconds as
// an int**, deliberately not a time.Duration (a Duration JSON-encodes as a
// nanosecond count, and the allowance rides on wire.Welcome). Turning that
// number into a timer is this package's whole share of the clock.
//
// # ⚠️ The mirror is here because no client can be thinner than one
//
// The mirror driver is filed in the design record under *The client* rather than
// under *The wire*, and it is in this package anyway because of one observation
// about the protocol: **nothing on the wire says whose turn it is.** wire.Turn
// carries a decision and a digest, and a client derives "I am being asked" from
// its own replayed battle — Mirror.Asking is that derivation, and it is the only
// one there is. That is deliberate (a "your turn" message would be a second
// declaration of state the mirror already computes), and it means an end-to-end
// test cannot exist without a mirror. Writing one as test code now and promoting
// it later would be writing it twice.
//
// The same argument reaches the *end* of a match. There is no series-standing
// message, so a client stops when its own arithmetic says the series is over —
// its own Ended events against wire.Welcome.Battles — which is why Mirror.Over
// re-derives a rule the room also has. Two peers agreeing because they compute
// the same thing from the same configuration is the mirror contract itself, the
// same shape wire.Welcome.TurnCap already takes.
//
// # What is deliberately NOT in here
//
// So a reader does not go looking for it:
//
//   - **Any TUI screen**, the lobby, the waiting and countdown drawing, and the
//     wordings. A wire.Code and a wire.Closure travel as ids precisely so the
//     wording lives at the client's far end, and nothing here words one — every
//     refusal comes out of this package as a *Refusal carrying the code, for a
//     screen to say in the player's own language. → TODO.md § The client.
//   - **The seat token and the rejoin.** Which is why CloseThreshold is what it
//     is: with no rejoin, a socket closing is a match ending, so that threshold
//     is currently guarding a whole match rather than a ping. → its own note.
//   - **The host binary.** Nothing here opens a listener, picks a port, decides
//     which address a room code should carry, reads a flag or prints a word: a
//     Server is an http.Handler, and the end-to-end test is what proves the shell
//     is wired up. That is cmd/hexarena-host, and it is built.
//     ⚠️ **Server.Shutdown is the one thing that crossed back**, because it could
//     not live out there: http.Server.Shutdown does not wait for hijacked
//     connections, and only this package holds the sockets that were hijacked.
//     It still opens nothing — the *listener* is the binary's, and closing it is
//     the binary's too.
//   - **Writing a finished match out as a battle.Log.** Another cursor over the
//     battle, which is why the room reads its own that way.
//   - **Spectators**, and **TLS** (→ README.md § Not in the first version).
//
// # A connection finds its room in the URL, not in a message
//
// No message body carries a wire.RoomCode and none should: the code is what a
// person **pastes to connect**, which is addressing rather than protocol
// content, and keeping it out is why widening the code to twelve characters
// moved no golden. So the code rides in the request path. → roomOf, for why the
// code is decoded and re-encoded before it is used as a key, and why an
// undecodable code is not refused here.
package socket

import (
	"time"

	"github.com/vukyn/hexarena/internal/wire"
)

// DefaultCloseThreshold is how long a silent, unresponsive peer is tolerated
// before the transport decides there is nobody there — at which point it tells
// the room, which ends the match as room.VerdictAbandoned.
//
// ⚠️ **It is not a ping timeout and it must not be picked as one.** There is no
// rejoin yet, so a socket closing is a *match* ending: this is the only dial
// standing between a wifi hiccup and losing a whole bo3, and the design record
// says so in as many words. → TODO.md, under the seat token and the rejoin,
// which is the item that turns this back into an ordinary liveness number.
//
// Sixty seconds, and the two things it was picked against:
//
//   - **Generous against a hiccup.** A LAN wifi roam or a switch reconvergence
//     is seconds, and TCP retransmission backoff rides out tens of seconds
//     without the socket noticing at all — so this is several times the worst
//     plausible blip. Note what it therefore does *not* govern: a peer whose
//     process exits sends a FIN, the read fails at once, and that is a real
//     departure with nothing to threshold. This number is only ever spent on a
//     peer that has gone **silent and unresponsive**, which is exactly the case
//     it has to be forgiving about.
//   - **Under the turn allowance**, which is ninety seconds
//     (room.DefaultAllowance). A machine that dies mid-turn is then noticed as a
//     departure *before* its allowance runs out, so the match ends as abandoned
//     rather than grinding out one timeout per turn until the board kills the
//     passing units — which is what would happen if this were the larger of the
//     two, and is the slow, confusing version of the same ending.
//
// It is configurable (Timings.CloseThreshold) because the two bounds above leave
// a range rather than a point, and because the number a host wants on a wired
// LAN is not the one a host wants over patchy wifi.
const DefaultCloseThreshold = 60 * time.Second

// DefaultKeepalive is how often the transport pings a peer to find out whether
// it is still there.
//
// It has to exist separately from the threshold because silence is not absence:
// a player thinking about a turn sends nothing for up to the whole allowance,
// so a read deadline would drop somebody for concentrating. A ping is answered
// by the *library* rather than by the player, which is what makes it a
// measurement of the machine rather than of the person.
const DefaultKeepalive = 15 * time.Second

// DefaultWrite is how long one message may take to go out before the peer it is
// addressed to counts as unreachable.
//
// It bounds the one place this transport holds a lock across I/O: a room's
// exchange is ordered by a mutex (→ table.exchange) and the writes happen under
// it, so a stuck peer would otherwise hold its own room still. Ten seconds is
// far past a LAN write of a few kilobytes and far short of the close threshold,
// so a genuinely wedged connection is reported by the threshold rather than by
// this.
const DefaultWrite = 10 * time.Second

// DefaultMessageLimit is the largest message either side will read.
//
// ⚠️ **This was written on a guess and the guess was wrong, so the figure is the
// measurement.** The reasoning was that wire.Start carries the whole resolved
// roster — every unit of both sides with its flat stat line, its kit and its
// traits — so a 5v5 start would approach the library's own 32 KiB default and a
// megabyte would be the safe answer. Measured
// (TestTheLargestStartFitsTheMessageLimit): the largest start a legal room can
// send is **2,911 bytes** over ten units. The library's default would have done
// perfectly well, and a megabyte was twenty times more allocation than a peer
// should be able to ask of this side for no reason at all.
//
// So it is 64 KiB: explicit rather than inherited, twenty-two times the largest
// message the protocol can actually produce — room for a wider format, longer
// ids or a field added to a roster entry — and still a bound. The test holds
// both ends of that, because a limit with no headroom and a limit with nothing
// but headroom are both worth failing on.
const DefaultMessageLimit = 64 << 10

// Timings is the transport's clock: every duration this package reads, in one
// place, because this package is the only one in the PvP stack that reads any.
//
// A zero field takes its default, so a caller that has no opinion passes the
// zero value and a test that needs a one-second allowance overrides one line.
type Timings struct {
	// CloseThreshold is how long a silent, unresponsive peer is tolerated.
	// → DefaultCloseThreshold, which is where what it guards is written down.
	CloseThreshold time.Duration
	// Keepalive is how often a peer is pinged.
	Keepalive time.Duration
	// Write is how long one outbound message may take.
	Write time.Duration
	// MessageLimit is the largest message that will be read, in bytes.
	MessageLimit int64
}

// withDefaults fills in whatever the caller had no opinion about.
func (t Timings) withDefaults() Timings {
	if t.CloseThreshold <= 0 {
		t.CloseThreshold = DefaultCloseThreshold
	}
	if t.Keepalive <= 0 {
		t.Keepalive = DefaultKeepalive
	}
	if t.Write <= 0 {
		t.Write = DefaultWrite
	}
	if t.MessageLimit <= 0 {
		t.MessageLimit = DefaultMessageLimit
	}
	return t
}

// allowanceOf is a room's per-prompt allowance as a duration, which is the one
// conversion in the repository from the protocol's seconds-as-an-int to a clock.
//
// It is a function rather than an expression at the call site because it is the
// seam the whole "the room reads no clock" arrangement turns on: there is
// exactly one place a wire.Welcome's number becomes a time.Duration, and it is
// this one.
func allowanceOf(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// roomPrefix is where a connection lands, and roomPattern is that path with the
// code as a wildcard. One declaration, read by the server's mux and by RoomPath,
// because a client dialling a path the server does not serve is a refusal with
// no code in it at all.
const (
	roomPrefix  = "/room/"
	roomPattern = roomPrefix + "{code}"
)

// RoomPath is the request path for one room code.
//
// ⚠️ The code goes in the **path** and not in a message, and that is a decision
// rather than a convenience: a code is what a person pastes to connect, so it is
// addressing rather than protocol content. Keeping it out of the eight messages
// is why widening it from ten characters to twelve moved no byte of
// messages.golden. → wire.RoomCode.
func RoomPath(code wire.RoomCode) string { return roomPrefix + string(code) }
