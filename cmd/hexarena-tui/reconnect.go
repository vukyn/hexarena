package main

import (
	"context"
	"errors"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// standing is what the client that just stopped playing says about the match it
// was in: the seat it can prove it held, and the two ways there can be nothing to
// come back to.
//
// ⚠️ **It is a value taken off the client rather than the client itself, and the
// reason is what reconnect must not do.** A dropped socket, a cancelled context
// and a match played to its end all come back from Play as an error, so a
// reconnection that inspected one would be this client deciding what a network
// failure looks like. The mirror already knows the two facts that matter, so they
// are read once, here, and handed over as answers — which also makes the decision
// testable without a live socket to drop.
type standing struct {
	token  wire.SeatToken
	over   bool
	closed bool
}

// standingOf reads those three off a client.
func standingOf(client *socket.Client) standing {
	_, closed := client.Mirror().Closure()
	return standing{token: client.Token(), over: client.Mirror().Over(), closed: closed}
}

// redial is what a second Dial to the same room needs. → the session field.
type redial struct {
	code       wire.RoomCode
	hello      wire.Hello
	books      battle.Books
	characters *cast.Book
}

// reconnect takes the seat back after a socket closed under a live match, and
// hands back the client that holds it — or nil, which means this match is over
// and the caller should report what Play returned.
//
// ⚠️ **It decides whether to try at all by asking the MIRROR rather than by
// reading the error.** A dropped socket, a cancelled context and a match played
// to its end all come back from Play as an error, and telling them apart by
// inspecting one would be this client deciding what a network failure looks
// like. The mirror already knows the two facts that matter — the room said the
// match closed, and the series is decided — and either of them means there is
// nothing to come back to.
func (s *session) reconnect(ctx context.Context, standing standing) *socket.Client {
	// ⚠️ **The two cheap refusals come before the client is touched at all**, and
	// the order is load-bearing rather than tidy: a session that never dialled
	// holds no client to ask, and a session whose player has quit is one where
	// asking is pointless. Reading the token first would be reaching into
	// something that may not be there to answer.
	held := s.redialling()
	if held == nil {
		// Nothing was ever dialled, so there is no room to dial again.
		return nil
	}
	if ctx.Err() != nil {
		// The player quit or left. The socket closing is the consequence rather
		// than the cause, and dialling again would rejoin a room somebody has
		// just walked out of.
		return nil
	}
	if !standing.token.Set() {
		// A room that issues no token cannot be come back to. → room.Deps.Tokens.
		return nil
	}
	if standing.over || standing.closed {
		return nil
	}

	s.setReconnecting(true)
	defer s.setReconnecting(false)

	joining := held.hello
	joining.Token = standing.token
	budget := newRetries()
	for {
		if ctx.Err() != nil {
			return nil
		}
		back, err := socket.Dial(ctx, held.code, joining, held.books, socket.ClientOptions{
			Characters: held.characters,
			Draft:      s.chooseDraft,
			Stepped: func() {
				s.observed()
				s.send(matchSteppedMsg{})
			},
		})
		if err == nil {
			return back
		}
		// ⚠️ **A refusal is the room's answer and ends this**, where a network
		// error is the network's and does not. The room refusing means the seat
		// is not there to take — the window ran out, or the match is over and the
		// code names nothing — and retrying against that is asking a question
		// that has been answered.
		var refused *socket.Refusal
		if errors.As(err, &refused) {
			return nil
		}
		if budget.spent() || !budget.pause(ctx) {
			return nil
		}
	}
}

// redialling is what a second dial needs, read under the mutex.
func (s *session) redialling() *redial {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.redial
}

// setReconnecting records whether this client is trying to take its seat back,
// and asks for a redraw so the screen can say so.
func (s *session) setReconnecting(trying bool) {
	s.mu.Lock()
	s.reconnecting = trying
	s.mu.Unlock()
	s.send(matchSteppedMsg{})
}

// Reconnecting is whether this client has lost its socket mid-match and is
// trying to take its seat back, which is the one thing a screen has to be able
// to say about it: a board that simply stopped moving is indistinguishable from
// the other player thinking.
func (s *session) Reconnecting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconnecting
}
