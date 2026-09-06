package socket

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/websocket"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
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
	conn    *connection
	mirror  *Mirror
	code    wire.RoomCode
	stepped func()
	// drafting answers a draft decision, and is nil for a client that was never
	// meant to be in a room that drafts. → ClientOptions.Draft, and answer, which
	// is where a nil one becomes a loud failure rather than a wait.
	drafting DraftChooser
}

// ClientOptions is everything a caller may say about a dialled client, and it is
// a struct rather than a sixth positional parameter for the reason Options is
// one on the server side: a caller that wants the hook and not the clock should
// not have to write the clock down.
type ClientOptions struct {
	// Timings is the transport's clock. The zero value takes every default.
	Timings Timings

	// Stepped is called on the Play goroutine after every message this client
	// has taken in, so a renderer knows there is something new to draw.
	//
	// ⚠️ **It is the other half of a client's redraw and a chooser cannot be
	// it.** A chooser only fires on *this* client's turns, so a screen driven by
	// one alone would sit still for the whole of the opponent's turn and then
	// jump — and in a match that is up to a ninety-second allowance of a board
	// that looks frozen.
	//
	// ⚠️ **It is handed NOTHING**, and both halves of that are deliberate.
	// Passing the Mirror out is exactly what Mirror.Read exists to refuse, and
	// passing the battle out would be the pointer escaping the lock. And it is
	// called with **no lock held**: a caller that Sends into a program whose
	// Update is inside Mirror.Read would otherwise deadlock the two against each
	// other, which is the same ordering Mirror.Decide is written under.
	//
	// A nil hook is the ordinary case — a client with nothing drawing it, which
	// is what every test that plays a match out with a rating is.
	Stepped func()

	// Characters is the cast a draft's pool is drawn from, and it is here rather
	// than a sixth positional parameter on Dial for the argument this struct's own
	// comment makes: a caller with no interest in drafting should not have to
	// write it down, and Dial's signature stays put.
	//
	// ⚠️ **It is the book and not a draft.Pool**, deliberately. draft.NewPool is
	// this repository's single declaration of "the cast minus every character held
	// back", so a caller handing a pool in could hand one built from the whole
	// book — which would offer a held-back character the room refuses. →
	// Mirror.openDraft.
	//
	// A nil book is the ordinary case, and a welcome saying the room drafts
	// refuses a client without one rather than joining and then failing to compute
	// its pool.
	Characters *cast.Book

	// Draft is what answers a draft decision, and it is on this struct for the
	// same reason Characters is: Play's signature is untouched, and a client that
	// never drafts needs no change at all.
	//
	// ⚠️ **It is not battle.Chooser and could not be.** Play's chooser answers a
	// *battle* prompt — a skill and a cell off a board — and a draft decision is a
	// different question in a different vocabulary: a character out of a pool, a
	// form and a kit, a formation. → DraftChooser.
	//
	// ⚠️ **A nil one is not a client that quietly answers nothing.** A client
	// dialled without this that lands in a drafting room fails loudly the moment
	// it is asked, because a chooser that answered nothing would be a client
	// sitting for ever on a decision — the hang the whole timeout mechanism exists
	// to prevent, and the one #316 just fixed an instance of. The failure is at
	// the point of being **asked** rather than at the point of joining, so that a
	// caller with nothing to decide — a spectator, when there is one — can still
	// dial a room that drafts. → Client.answer.
	Draft DraftChooser
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
func Dial(ctx context.Context, code wire.RoomCode, hello wire.Hello, books battle.Books,
	options ClientOptions) (*Client, error) {
	at, which, err := code.Decode()
	if err != nil {
		return nil, fmt.Errorf("read the room code: %w", err)
	}
	canonical, err := wire.EncodeRoom(at, which)
	if err != nil {
		return nil, fmt.Errorf("read the room code: %w", err)
	}
	settings := options.Timings.withDefaults()
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
		conn:     newConnection(raw, settings),
		mirror:   NewMirror("", books, options.Characters),
		code:     canonical,
		stepped:  options.Stepped,
		drafting: options.Draft,
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
		c.mirror.takeSeat(message.Seat)
		return c.mirror.Receive(*message)
	case wire.Welcome:
		c.mirror.takeSeat(message.Seat)
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
// typing; a real client hands in the player's keystrokes instead. A **draft**
// decision is answered by ClientOptions.Draft, which is a different question and
// is why this signature did not have to grow one.
//
// ⚠️ **It answers before the first read as well as after every message, and that
// is what makes a drafting room joinable at all.** A room that drafts announces
// nothing when its draft opens — a wire.Drafted carries recorded decisions, none
// have been taken, and a room must not send one carrying none — so the host's
// first ban is due with no message on its way to prompt it. A loop that read
// first would sit there for ever, and it did.
func (c *Client) Play(ctx context.Context, choose battle.Chooser) error {
	watching, gone := context.WithCancel(ctx)
	defer gone()
	// ⚠️ The keepalive giving up has to be told apart from the caller giving up,
	// because both arrive as a cancelled context and only one of them is a
	// fault. A closed channel beside the cancel is what says which.
	silent := make(chan struct{})
	go c.keepalive(watching, func() { close(silent); gone() })
	// The opening decision, which in a battle is nobody's — no wire.Start has
	// arrived, so nothing is being asked and this is a no-op. → the ⚠️ above.
	if done, err := c.answer(watching, choose); done || err != nil {
		return err
	}
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
		// ⚠️ Outside Receive and therefore outside the write lock, which is what
		// lets the hook draw: a renderer told "there is something new" goes
		// straight to Mirror.Read, and a hook called from inside Receive would
		// have it waiting on the lock the caller is holding.
		if c.stepped != nil {
			c.stepped()
		}
		if c.mirror.Over() {
			c.Close()
			return nil
		}
		if done, err := c.answer(watching, choose); done || err != nil {
			return err
		}
	}
}

// answer is one decision, when this client is the one being asked for one, and
// it reports whether the loop should stop.
//
// ⚠️ **The draft is asked first, and the two are mutually exclusive rather than
// ordered.** A draft runs before the battle and the room begins one only when
// the draft is Done, so at most one of the two is ever open — the order is what
// reads naturally rather than a precedence rule.
//
// ⚠️ **A missing chooser fails here rather than at the join.** A nil one that
// quietly answered nothing would be a client sitting for ever on a decision, and
// the failure is placed at the point of being *asked* so that a caller with
// nothing to decide can still dial a room that drafts. → ClientOptions.Draft.
func (c *Client) answer(ctx context.Context, choose battle.Chooser) (bool, error) {
	if prompt, asking := c.mirror.DraftAsking(); asking {
		if c.drafting == nil {
			return false, fmt.Errorf("%s is being asked to %s and this client was dialled with "+
				"no draft chooser, so nothing here can answer it: a client that landed in a room "+
				"which drafts without one would sit on the decision for ever",
				c.Seat(), prompt.Due.Step)
		}
		decision, decided := c.mirror.DecideDraft(c.drafting)
		if !decided {
			return false, fmt.Errorf("%s was asked to %s and decided nothing: a draft has no "+
				"pass, so there is nothing to send in its place", c.Seat(), prompt.Due.Step)
		}
		return c.sending(ctx, decision)
	}
	if _, asking := c.mirror.Asking(); !asking {
		return false, nil
	}
	decision, decided := c.mirror.Decide(choose)
	if !decided {
		return false, fmt.Errorf("%s was asked to act and decided nothing", c.Seat())
	}
	return c.sending(ctx, decision)
}

// sending puts one decision on the wire, and reads the far end having gone as
// the ordinary ending it is rather than as a fault.
func (c *Client) sending(ctx context.Context, decision wire.Body) (bool, error) {
	if err := c.conn.send(ctx, decision); err != nil {
		if ended(err) {
			return true, nil
		}
		return false, fmt.Errorf("answer for %s: %w", c.Seat(), err)
	}
	return false, nil
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
