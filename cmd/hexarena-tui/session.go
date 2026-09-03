package main

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # A blocking Play loop meeting a bubbletea event loop
//
// socket.Client.Play is a **blocking** message loop that asks a battle.Chooser
// for a decision when the turn is this client's, and bubbletea is a loop that
// hands one keystroke at a time to a model and takes a new model back. Neither
// can be driven by the other, so Play runs on its own goroutine and the two talk
// through exactly two channels of one item each: the chooser blocks on
// s.answers, and everything the loop wants drawn arrives at the model as a
// message through a sender.
//
// ⚠️ **Play is kept exactly as it is and nothing here reimplements it.** It
// carries four things a second loop would quietly lose: divergence detection on
// the turn it happens, the errUnreadable → wire.CodeUnknownMessage refusal, the
// difference between the keepalive giving up and the caller cancelling, and
// Mirror.Over. A client that read messages itself would be a fifth peer
// implementation of the protocol in a repository whose whole design is that
// there are two.
//
// ## The two arms of the chooser, and what cancels each
//
// choose blocks on a select with two arms and no third. Arm one is the player
// answering. Arm two is s.cancel, which has exactly two callers:
//
//  1. leave, from the model, on esc in a match or on quit.
//  2. **run's own defer** in main.go, which fires however program.Run returns —
//     a clean quit, ctrl+c, or an error. That defer is what makes "a player who
//     quits mid-turn leaves the chooser blocked for ever" impossible **by
//     construction** rather than by care: the process cannot leave run without
//     cancelling.
//
// ## ⚠️ A residual, named rather than overlooked
//
// A peer that dies **while this client is being asked** does not unblock the
// chooser. Play is inside Decide at that moment, not inside conn.read, so
// neither the read failing nor the keepalive giving up (which cancels Play's own
// internal watching context, not this one) can reach it. The goroutine sits
// until the player presses esc. Nothing a person can see is stuck — the UI stays
// responsive throughout and the board is drawn — and the third arm that would
// close it is a timer of Welcome.Allowance plus grace, which is a **clock**.
// That belongs with the countdown it shares one with, in the next step, and is
// deliberately not built here: CLAUDE.md's own warning is that a countdown moved
// into a fourth package is invisible to both existing clock bans.

// sender is what a session needs of a bubbletea program.
//
// It is an interface for two reasons rather than one. The knot: the program
// cannot be built until the model is, the model cannot be built until the
// session is, so the session cannot be handed a program at construction. And a
// headless test has no program at all — everything below is driven by an Update
// a test calls directly, so what a test needs is somewhere for the messages to
// land.
//
// ⚠️ **What a fake cannot see is that a real *tea.Program was ever attached.**
// That is held by the assertion below and by the single sess.attach(program)
// line in run.
type sender interface{ Send(tea.Msg) }

var _ sender = (*tea.Program)(nil)

// The three messages a match sends the model. All of them are **redraw
// triggers carrying no state**: the model re-reads the mirror under its lock,
// because a message carrying a *battle.Battle would be the pointer escaping
// that lock. It is internal/room's "a request is a VALUE, never a
// func(*Room)" pointed the other way.
type (
	// matchSteppedMsg is socket.ClientOptions.Stepped: a message arrived and
	// there is something new to draw. Sent on the Play goroutine.
	matchSteppedMsg struct{}
	// matchAskingMsg is the chooser having been called: it is this player's
	// turn. Sent on the Play goroutine, **before** the chooser reaches its
	// select — which is the window s.answers' one slot exists to absorb.
	matchAskingMsg struct{}
	// matchEndedMsg is Play having returned, sent from the goroutine that ran
	// it. Whatever it returned is readable through outcome by then.
	matchEndedMsg struct{}
)

// The two messages the dial itself answers with. These carry something, because
// a dial is a handover rather than a redraw: one of them carries the client the
// session is about to own, and the other the refusal a screen has to word.
type (
	matchJoinedMsg struct{ client *socket.Client }
	matchFailedMsg struct{ err error }
)

// session is one client's whole PvP side: the socket, the goroutine running
// Play, and the two channels between that goroutine and the model.
//
// ⚠️ **It is a pointer on the model and everything on it is guarded.** The model
// is a value copied on every keystroke, so a session copied with it would be a
// second set of channels; and three goroutines touch these fields — the model's,
// the Play loop's, and whichever one bubbletea runs a tea.Cmd on.
type session struct {
	// out is where a message goes. Written once, by attach, before anything can
	// send.
	out sender

	mu sync.Mutex
	// ctx is the match's own context and cancel is the ONLY thing that unblocks
	// a waiting chooser. → the two arms, above.
	ctx    context.Context
	cancel context.CancelFunc
	client *socket.Client
	// done is closed when Play has returned, and err is written before it is
	// closed and read after — which is what makes the pair race-free without
	// the reader holding anything.
	done chan struct{}
	err  error
	// left says this match has been left, so leaving twice is free.
	//
	// ⚠️ **A bool under the mutex rather than a sync.Once**, and the difference
	// is the second match: a Once cannot be re-armed, and a player who leaves a
	// room and joins another needs the guarantee back. open resets it.
	left bool

	// answers is the one decision in flight, and its capacity is **1** rather
	// than 0.
	//
	// ⚠️ **The pair that makes it correct is the ordering in choose.** The
	// chooser sends matchAskingMsg *before* it reaches its select, so a player
	// answering inside that window would hit the default on an unbuffered
	// channel and lose a real keystroke. One slot absorbs the window; the drain
	// at the top of every chooser call is what stops that slot outliving its
	// turn. A *second* press in the same turn hits the default and is dropped,
	// which is correct — one decision per turn, and the screen has already
	// stopped offering it.
	answers chan draw.PlayAnswer
}

func newSession() *session { return &session{} }

// attach names where a message goes.
//
// ⚠️ **A *tea.Program needs no guard around Send, and that is measured rather
// than assumed.** In charm.land/bubbletea/v2@v2.0.9, tea.go:
//
//	func (p *Program) Send(msg Msg) {
//	    select {
//	    case <-p.ctx.Done():
//	    case p.msgs <- msg:
//	    }
//	}
//
// Two facts follow. p.ctx is set in NewProgram — not in Run — and Run returning
// cancels it, so a Send **after the program has stopped** returns immediately
// and cannot strand the Play goroutine. And p.msgs is **unbuffered** with
// nothing reading it until Run begins, so a Send *before* Run blocks until it
// does. Nothing here sends before Run: the only senders are Stepped and the
// chooser, and both exist only after a Dial that an Update started.
func (s *session) attach(out sender) { s.out = out }

// send is one message to the model, and a session with nothing attached drops
// it.
//
// The nil check is for a test that is about the chooser rather than about the
// messages; every real path goes through attach in run.
func (s *session) send(message tea.Msg) {
	if s.out == nil {
		return
	}
	s.out.Send(message)
}

// dial is the command that joins a room.
//
// It is a tea.Cmd because a dial is a network round trip and the screen has to
// stay drawable while it happens — bubbletea runs one on its own goroutine and
// delivers what it returns as a message.
//
// ⚠️ **The match is armed before the dial rather than after it.** The context is
// what a dial is cancelled by, so a player who quits while a room is being
// called has to be able to reach it; arming afterwards would leave that window
// covered by nothing.
func (s *session) dial(code wire.RoomCode, hello wire.Hello, books battle.Books) tea.Cmd {
	ctx := s.open()
	return func() tea.Msg {
		client, err := socket.Dial(ctx, code, hello, books, socket.ClientOptions{
			Stepped: func() { s.send(matchSteppedMsg{}) },
		})
		if err != nil {
			return matchFailedMsg{err: err}
		}
		return matchJoinedMsg{client: client}
	}
}

// open arms a fresh match and hands back its context.
func (s *session) open() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx, s.cancel = ctx, cancel
	s.client, s.err, s.left = nil, nil, false
	s.done = make(chan struct{})
	s.answers = make(chan draw.PlayAnswer, 1)
	return ctx
}

// begin takes the dialled client and starts the loop.
func (s *session) begin(client *socket.Client) {
	s.mu.Lock()
	s.client = client
	done, ctx := s.done, s.ctx
	s.mu.Unlock()
	go func() {
		err := client.Play(ctx, s.choose)
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
		// Closed before the message is sent, so a model handling matchEndedMsg
		// finds the outcome already readable.
		close(done)
		s.send(matchEndedMsg{})
	}()
}

// choose is the chooser Play calls when the turn is this client's, and it is the
// whole of how a keystroke becomes a decision.
//
// → the two arms at the head of this file for what cancels each, and for the
// residual the second arm does not cover.
func (s *session) choose(*battle.Prompt) (battle.Choice, bool) {
	answers, ctx := s.turn()
	if answers == nil || ctx == nil {
		return battle.Choice{}, false
	}
	// ⚠️ **Drain first.** An answer buffered for a turn that has already gone —
	// the player answered, the server timed the seat out and passed for it — must
	// not be spent on the next one. Without this the keystroke pressed for the
	// turn the clock took would silently take the turn after it.
	select {
	case <-answers:
	default:
	}
	s.send(matchAskingMsg{})
	select {
	case answer := <-answers:
		return answer.Choice, answer.Acted
	case <-ctx.Done():
		// Pass, and nothing is spent by it. Mirror.Decide reads the false as a
		// wire.Pass, Play sends it on a cancelled context, conn.send fails,
		// ended() covers context.Canceled, and Play returns nil.
		return battle.Choice{}, false
	}
}

// answer is a keystroke on its way to the chooser, and it **never blocks**.
//
// ⚠️ That is the reverse deadlock closed: an Update that waited for a chooser to
// take its answer would hang the whole program whenever nobody was asking —
// between turns, after the match, and on every keystroke that is not a decision.
// A dropped keystroke is the right answer there, because there is no turn for it
// to be about.
func (s *session) answer(taken draw.PlayAnswer) {
	answers, _ := s.turn()
	if answers == nil {
		return
	}
	select {
	case answers <- taken:
	default:
	}
}

// turn is the current match's channel and context, read together.
func (s *session) turn() (chan draw.PlayAnswer, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.answers, s.ctx
}

// read runs fn under the mirror's read lock, which is the only safe way to look
// at a battle the Play goroutine is stepping. A session with no client reads
// nothing and says so.
func (s *session) read(fn func(socket.Sight)) bool {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return false
	}
	client.Mirror().Read(fn)
	return true
}

// live reports whether a match is joined, which is what the client's own guards
// against opening a second battle over the top of one are asked.
func (s *session) live() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil && !s.left
}

// outcome is what Play returned, and whether it has returned at all.
//
// The err is read only after the closed channel has been observed, which is what
// makes it safe without the caller knowing anything about the goroutine.
func (s *session) outcome() (error, bool) {
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done == nil {
		return nil, false
	}
	select {
	case <-done:
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.err, true
	default:
		return nil, false
	}
}

// finished is the channel a test waits on for Play to return. → outcome for the
// value.
func (s *session) finished() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// leave ends the match: it cancels the context — which is the only thing that
// unblocks a waiting chooser — and closes the socket, which is what the far end
// reads as an ordinary departure.
//
// ⚠️ **Leaving costs nothing and this client asks nothing before it.** Nobody
// forfeits: a departure announces and ends the match as abandoned, neither seat
// is charged with anything, and the enforcement of walking away from a losing
// board is social. A confirmation here would be this client inventing a cost the
// design refused. → README.md § Nobody forfeits.
//
// Idempotent, and safe from any goroutine. The cancel and the close are done
// **outside** the mutex, because Client.Close waits on a close handshake and a
// chooser reading s.ctx must not be behind it.
func (s *session) leave() {
	s.mu.Lock()
	if s.left || s.cancel == nil {
		s.mu.Unlock()
		return
	}
	s.left = true
	cancel, client := s.cancel, s.client
	s.mu.Unlock()
	cancel()
	if client != nil {
		client.Close()
	}
}
