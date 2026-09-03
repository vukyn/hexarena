package socket

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/websocket"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/wire"
)

// Client is a dialled connection and the Mirror behind it.
//
// It is the whole of what a screen would sit on top of: Dial gets past the gate,
// Mirror is the battle, and Play is the message loop. Nothing here draws
// anything and nothing here words a refusal — a wire.Code travels as an id
// precisely so the sentence lives at this end, in the player's own language.
// → the package comment on what is deliberately out of scope.
type Client struct {
	conn   *connection
	mirror *Mirror
	code   wire.RoomCode
}

// Refusal is the room turning this client away, and it carries the code and
// nothing else — for the reason wire.Refused does.
//
// It is a typed error so that a caller can word it: `errors.As` gives the code,
// and the caller's own language book gives the sentence.
type Refusal struct {
	Code wire.Code
}

func (r *Refusal) Error() string { return fmt.Sprintf("the room refused this client: %s", r.Code) }

// Dial connects to the room a code names and gets past its gate.
//
// The **code carries the address**, so there is nothing else to be told: it
// decodes to an IPv4 address, a port and which of the rooms behind them is
// meant, and the path is where the code itself rides (→ RoomPath). A player who
// typed it in lower case is fine — the code is re-encoded into its canonical
// form, exactly as the server does, so the path names the key the registry holds.
//
// On a refusal it returns a *Refusal and closes the connection: a refused join
// is the end of the conversation, and the code is everything the client needs in
// order to say why.
func Dial(ctx context.Context, code wire.RoomCode, hello wire.Hello, books battle.Books, timings Timings) (*Client, error) {
	at, which, err := code.Decode()
	if err != nil {
		return nil, fmt.Errorf("read the room code: %w", err)
	}
	canonical, err := wire.EncodeRoom(at, which)
	if err != nil {
		return nil, fmt.Errorf("read the room code: %w", err)
	}
	settings := timings.withDefaults()
	// ⚠️ Plain ws and not wss, which is a decision rather than an omission: the
	// design record refuses TLS on a LAN, because a self-signed certificate
	// implying security would be worse than saying there is none. → README.md
	// § Not in the first version.
	url := "ws://" + at.String() + RoomPath(canonical)
	raw, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("dial room %s: %w", canonical, err)
	}
	client := &Client{
		conn:   newConnection(raw, settings),
		mirror: NewMirror("", books),
		code:   canonical,
	}
	if err := client.handshake(ctx, hello); err != nil {
		client.conn.drop()
		return nil, err
	}
	return client, nil
}

// handshake sends the hello and reads the one answer to it.
func (c *Client) handshake(ctx context.Context, hello wire.Hello) error {
	if err := c.conn.send(ctx, hello); err != nil {
		return fmt.Errorf("say hello to room %s: %w", c.code, err)
	}
	body, err := c.conn.read(ctx)
	if err != nil {
		return fmt.Errorf("read the room's answer: %w", err)
	}
	switch message := body.(type) {
	case *wire.Refused:
		return &Refusal{Code: message.Code}
	case wire.Refused:
		return &Refusal{Code: message.Code}
	case *wire.Welcome:
		// ⚠️ The seat comes off the welcome and is not decided here. A client
		// cannot know which of the two places it took — the room hands them out
		// in the order it seats people — and it is the one fact that holds for
		// the whole match, where the *side* changes between battles.
		c.mirror.seat = message.Seat
		return c.mirror.Receive(*message)
	case wire.Welcome:
		c.mirror.seat = message.Seat
		return c.mirror.Receive(message)
	}
	if body == nil {
		return fmt.Errorf("room %s answered a hello with nothing", c.code)
	}
	return fmt.Errorf("room %s answered a hello with a %s", c.code, body.Kind())
}

// Code is the canonical code this client is connected under.
func (c *Client) Code() wire.RoomCode { return c.code }

// Seat is which of the room's two places this client took, for the match.
func (c *Client) Seat() wire.Seat { return c.mirror.Seat() }

// Mirror is this client's own battle. → Mirror, on why a client cannot be
// thinner than one.
func (c *Client) Mirror() *Mirror { return c.mirror }

// Close ends the connection with a normal closure, which is what the far end
// reads as an ordinary departure rather than a fault.
func (c *Client) Close() { c.conn.bye(websocket.StatusNormalClosure, "leaving") }

// Play is the client's message loop: read, step the mirror, and answer when the
// mirror says the turn is this client's.
//
// It returns when the match is over by this client's **own** arithmetic (→
// Mirror.Over — there is no series-standing message, so the client computes the
// ending), when the room closes the match for a reason the board cannot show, or
// when the connection ends in the ordinary way. A divergence is an error, on the
// turn it happens.
//
// The chooser is what decides a turn. `(*battle.Battle).Suggest` is the shipped
// rating and satisfies it, which is what lets a whole match play out with nobody
// typing; a real client hands in the player's keystrokes instead.
func (c *Client) Play(ctx context.Context, choose battle.Chooser) error {
	watching, gone := context.WithCancel(ctx)
	defer gone()
	// ⚠️ The keepalive giving up has to be told apart from the caller giving up,
	// because both arrive as a cancelled context and only one of them is a
	// fault. A closed channel beside the cancel is what says which.
	silent := make(chan struct{})
	go c.keepalive(watching, func() { close(silent); gone() })
	for {
		body, err := c.conn.read(watching)
		switch {
		case err == nil:
		case errors.Is(err, errUnreadable):
			// A server one version ahead. The protocol's answer is a refusal
			// carrying wire.CodeUnknownMessage, and the connection carries on.
			if err := c.conn.refuse(watching, wire.CodeUnknownMessage); err != nil {
				return err
			}
			continue
		case ended(err):
			select {
			case <-silent:
				return fmt.Errorf("room %s answered no ping for %s, so this client gave up on it",
					c.code, c.conn.timings.CloseThreshold)
			default:
			}
			// The far end closed. Every ordinary ending closes both sockets, so
			// this is the normal way out rather than a fault.
			return nil
		default:
			return fmt.Errorf("read from room %s: %w", c.code, err)
		}
		if err := c.mirror.Receive(body); err != nil {
			return err
		}
		if c.mirror.Over() {
			c.Close()
			return nil
		}
		if _, asking := c.mirror.Asking(); !asking {
			continue
		}
		answer, decided := c.mirror.Decide(choose)
		if !decided {
			return fmt.Errorf("%s was asked to act and decided nothing", c.Seat())
		}
		if err := c.conn.send(watching, answer); err != nil {
			if ended(err) {
				return nil
			}
			return fmt.Errorf("answer for %s: %w", c.Seat(), err)
		}
	}
}

// keepalive is the client's half of the close threshold, and it exists for the
// case the server's own cannot cover: a host that goes silent without closing
// its socket would otherwise leave this client waiting on a turn that is never
// coming.
//
// It ends the read rather than reporting anything — there is no room at this end
// to be told a peer has gone, so the match simply ends locally, which is what
// Play returning means.
func (c *Client) keepalive(ctx context.Context, gone func()) {
	beat := newTicker(c.conn.timings.Keepalive)
	defer beat.stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-beat.each():
		}
		if err := c.conn.ping(ctx); err != nil {
			gone()
			return
		}
	}
}
