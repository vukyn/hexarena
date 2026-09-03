package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/coder/websocket"

	"github.com/vukyn/hexarena/internal/wire"
)

// errUnreadable is a message this peer could not decode, and it is a sentinel
// rather than the decoder's own error **on purpose**.
//
// ⚠️ The bytes of an unreadable message are never reported anywhere, and this is
// why: the first message on a connection is a wire.Hello, a hello carries the
// room's password in the clear, and json's own decode errors can quote what they
// choked on. wire.Password redacts itself under every fmt verb, which covers a
// hello that *decoded* — but a hello that did not decode is still bytes, and
// there is no type left to do the redacting. So the rule is one rule with no
// exceptions: an unreadable message is reported as a byte count and a kind that
// could not be read, never as content. → TestAWrongPasswordIsRefusedAndNeverPrinted.
//
// A peer that sends one is answered wire.CodeUnknownMessage and the connection
// carries on, which is what a peer one version ahead looks like.
var errUnreadable = errors.New("the message could not be read")

// connection is one peer's socket: the reads, the writes, and the two rules
// about them.
//
// It is used by both sides — a server's accepted connection and a client's
// dialled one are the same thing from here — because the protocol is symmetric
// in everything but which of the eight messages each end sends.
type connection struct {
	conn    *websocket.Conn
	timings Timings

	// writing orders the writes on **this** connection, and it is a different
	// job from the room's own exchange lock: that one orders whole batches
	// across the two seats (→ table.exchange), and this one is what makes a
	// close from the connection's own goroutine safe against a dispatch from
	// somebody else's. The two are always taken in that order and never the
	// reverse, so there is no cycle to deadlock on.
	writing sync.Mutex
}

// newConnection wraps an accepted or dialled socket, and sets the read limit
// before anything is read off it — before, because a limit set after the first
// read is a limit the first message never met. → DefaultMessageLimit, and the
// measurement it was corrected by.
func newConnection(conn *websocket.Conn, timings Timings) *connection {
	conn.SetReadLimit(timings.MessageLimit)
	return &connection{conn: conn, timings: timings}
}

// read is the next message, decoded.
//
// An unreadable message is errUnreadable and nothing else — no wrapped decoder
// error, no bytes. → the sentinel's own note, which is where the password lives
// in that decision.
func (c *connection) read(ctx context.Context) (wire.Body, error) {
	kind, raw, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if kind != websocket.MessageText {
		return nil, fmt.Errorf("%w: %d bytes arrived as %s and this protocol is JSON text", errUnreadable, len(raw), kind)
	}
	body, err := wire.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %d bytes", errUnreadable, len(raw))
	}
	return body, nil
}

// send writes a run of messages in order, each bounded by the write timeout.
//
// The whole run is under this connection's own lock, so two callers cannot
// interleave their messages — a wire.Start and the wire.Turn that follows it
// arriving the other way round would be a client building its battle after the
// first decision had already been applied to it.
func (c *connection) send(ctx context.Context, bodies ...wire.Body) error {
	c.writing.Lock()
	defer c.writing.Unlock()
	for _, body := range bodies {
		raw, err := wire.Encode(body)
		if err != nil {
			return fmt.Errorf("encode a %s: %w", body.Kind(), err)
		}
		written, cancel := context.WithTimeout(ctx, c.timings.Write)
		err = c.conn.Write(written, websocket.MessageText, raw)
		cancel()
		if err != nil {
			return fmt.Errorf("write a %s: %w", body.Kind(), err)
		}
	}
	return nil
}

// refuse is one wire.Refused and the code it carries, which is every refusal
// this transport ever sends — nothing here words one.
func (c *connection) refuse(ctx context.Context, code wire.Code) error {
	return c.send(ctx, wire.Refused{Code: code})
}

// ping asks whether the peer is still there, and is bounded by the close
// threshold rather than by the write timeout: a pong is answered by the far
// side's *library* rather than by its player, so what this measures is the
// machine. → DefaultCloseThreshold.
func (c *connection) ping(ctx context.Context) error {
	asked, cancel := context.WithTimeout(ctx, c.timings.CloseThreshold)
	defer cancel()
	return c.conn.Ping(asked)
}

// bye closes the connection with a status, and is safe to call more than once —
// the library makes a second Close a no-op, which matters because a connection
// is closed both by the goroutine that owns it and by the deferred tidy-up
// behind it.
//
// The reason is a fixed string per call site rather than a formatted one: the
// protocol's rule is that nothing on the wire is prose, and a close reason is
// still something a peer might print.
func (c *connection) bye(status websocket.StatusCode, reason string) {
	c.writing.Lock()
	defer c.writing.Unlock()
	_ = c.conn.Close(status, reason)
}

// drop closes the connection without a handshake, for the paths that are already
// leaving — a peer that has gone, or a tidy-up behind a real close.
func (c *connection) drop() { _ = c.conn.CloseNow() }

// ⚠️ There is no read deadline anywhere in this file, and its absence is a
// decision. A player thinking about a turn sends nothing for up to the whole
// allowance, so a deadline on a read would drop somebody for concentrating —
// which is why liveness is a ping (ping, above) and not a silence.

// ended reports whether an error is a connection closing in the ordinary way
// rather than something worth reporting.
//
// A match that ends closes both sockets, so "the far end went away cleanly" is
// the **normal** end of every loop in this package and must not be reported as a
// fault. Three shapes count:
//
//   - a close handshake with a normal status, which is the far end leaving;
//   - net.ErrClosed, which is **this** end leaving — a client that calls Close
//     and then finds its own in-flight read or write refused. ⚠️ This one was
//     missing, and the departure test is what found it: a client that closed its
//     own connection reported `use of closed network connection` as a failure of
//     the match it had just left;
//   - context.Canceled, which is this side deciding to stop.
//
// ⚠️ context.DeadlineExceeded is deliberately **not** among them. The only
// deadline this package sets is Timings.Write, so a write that exceeds one is a
// peer that has stopped reading — which is exactly the sort of thing the error
// sink exists to carry, and swallowing it would lose the one signal that a
// connection is wedged rather than gone.
func ended(err error) bool {
	if err == nil {
		return true
	}
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway:
		return true
	}
	return errors.Is(err, net.ErrClosed) || errors.Is(err, context.Canceled)
}
