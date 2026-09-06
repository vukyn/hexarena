package main

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/draft"
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
// ## The three arms of the chooser, and what ends each
//
// choose blocks on a select with three arms. Arm one is the player answering.
// Arm two is s.cancel, which has exactly two callers:
//
//  1. leave, from the model, on esc in a match or on quit.
//  2. **run's own defer** in main.go, which fires however program.Run returns —
//     a clean quit, ctrl+c, or an error. That defer is what makes "a player who
//     quits mid-turn leaves the chooser blocked for ever" impossible **by
//     construction** rather than by care: the process cannot leave run without
//     cancelling.
//
// ## ⚠️ The residual this file used to name is CLOSED, and arm three is how
//
// It read: a peer that dies **while this client is being asked** does not
// unblock the chooser. That diagnosis stands and is worth keeping, because it is
// what says why the other two arms cannot cover it — Play is inside Decide at
// that moment, not inside conn.read, so neither the read failing nor the
// keepalive giving up (which cancels Play's own internal watching context, not
// this one) can reach a chooser waiting on a keystroke. The goroutine sat until
// the player pressed esc.
//
// Arm three is a timer of Welcome.Allowance plus chooserGrace, after which the
// chooser **passes**. The grace is what makes this client the *second* to give
// up rather than the first — the room's own timer is the one that matters, and
// the argument for the number is at chooserGrace.
//
// ⚠️ **The race with the room's own timeout is already designed for.** An answer
// that arrives after the room has passed for that seat is refused by
// Room.TimedOut / Room.Deliver, which refuse a seat they are not asking, rather
// than being mis-applied to somebody's real turn.
//
// ⚠️ **It closes a second hole, which the original note did not see.** A player
// who simply never answers strands their own client just as thoroughly as a dead
// peer does: Play is inside Decide, so the room's pass for that seat arrives at
// a socket nobody is reading, and the board stops for the rest of the match.
// With arm three the chooser passes, Play sends it, and the queued messages are
// taken in behind it.
//
// The clock this needs is the one the countdown needs, which is why the two
// landed together and why both live in clock.go — the file the module-wide
// allowlist names for this package.

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
	mu sync.Mutex
	// out is where a message goes, and it is under the mutex like everything
	// else here. → attach, for the two race sightings that moved it in.
	out sender
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
	// clock is when this client saw the open turn open, which is what the
	// countdown on the battle screen is counted from and the one piece of state
	// here that a wall clock reaches. → clock.go, which is the whole of this
	// package's clock.
	clock matchClock
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
	// channel and lose a real keystroke. One slot absorbs the window; the check
	// at the top of every chooser call is what stops that slot outliving its
	// turn. A *second* press in the same turn hits the default and is dropped,
	// which is correct — one decision per turn, and the screen has already
	// stopped offering it.
	answers chan pressed

	// decisions is the draft's twin of answers, one slot and the same
	// arrangement. A channel of its own rather than a second kind on `pressed`,
	// because the two questions are asked by two different choosers on the same
	// goroutine and an answer for one must never be spent on the other: a battle
	// decision is a skill and a cell off a board, a draft decision is a character
	// out of a pool.
	decisions chan draftPressed
	// characters is the cast a drafting room's pool is drawn from and pool is
	// that pool, kept because the **screen** draws it and the mirror does not
	// hand it back on a reading.
	//
	// ⚠️ **Both come off the EMBEDDED books, whatever --data says**, which is
	// model.dialling's own rule one step further: wire.Version's digest is over
	// the embedded files, so a client whose pool came from an edited directory
	// would pass a gate promising the two peers draft the same cast and then ban
	// a character the room does not have.
	characters *cast.Book
	pooled     []cast.Character
}

// draftPressed is one draft decision with the decision it was given for.
//
// ⚠️ **The record length is the routing key and the step is not enough**, which
// is socket.DraftDue's own finding: a seat's two ban slots have the other seat's
// between them, so a ban given for the first and delivered while the second is
// open carries the same step and the same absent character — everything the wire
// has. The count is local and never travels.
type draftPressed struct {
	decision wire.DraftDecision
	recorded int
}

// about reports whether this was pressed for the decision now being asked.
func (p draftPressed) about(prompt socket.DraftPrompt) bool {
	return p.recorded == prompt.Due.Recorded && p.decision.Step == prompt.Due.Step
}

// pressed is one answer with the turn it was given for.
//
// ⚠️ **The turn is carried because the slot outlives nothing else.** A window is
// not a turn: a keystroke can land in the slot for the turn that is open, for a
// turn the clock has taken, or for no turn at all, and the three are only
// distinguishable if the answer says which turn it is about. → choose, whose
// whole top half is that question.
//
// ⚠️ **It is a pair beside draw.PlayAnswer rather than two more fields inside
// it.** PlayAnswer is battle.Chooser's return pair and its own doc says so —
// "a second vocabulary for a decision" is the mistake it was written to refuse —
// so the routing information is this client's, kept where the routing is done.
type pressed struct {
	answer draw.PlayAnswer
	// unit and turn are battle.Prompt's own two, copied rather than a pointer
	// kept: the prompt belongs to the mirror's battle and this travels between
	// goroutines.
	unit string
	turn int
}

// about reports whether this was pressed for the prompt now being asked.
//
// A nil prompt is answered false, which is the reading a chooser called with no
// prompt wants: nothing pressed can be about a turn that is not there.
func (p pressed) about(prompt *battle.Prompt) bool {
	return prompt != nil && p.unit == prompt.Unit && p.turn == prompt.Turn
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
//
// ⚠️ **It takes the mutex, and the comment on the field used to be the whole of
// the guarantee.** `out` was written here and read in `send` with no
// synchronisation at all, on the strength of "written once, before anything can
// send" — which is true of `run`, where the attach precedes `program.Run()`, and
// is **not** true of a test that attaches a sender to a session a Play goroutine
// is already running against. `cmd/hexarena-tui`'s `-race` line caught it twice,
// days apart, on `TestTheCountdownReachesTheScreenOverASocket`, and neither
// sighting reproduced: about one run in ten of the whole package under the
// detector. A field guarded by an ordering claim rather than by the mutex every
// other field on this struct uses is the one field that could do that.
func (s *session) attach(out sender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = out
}

// send is one message to the model, and a session with nothing attached drops
// it.
//
// The nil check is for a test that is about the chooser rather than about the
// messages; every real path goes through attach in run.
//
// ⚠️ **The lock is released before the Send**, and that ordering is the point
// rather than an optimisation: `tea.Program.Send` blocks until the update loop
// takes the message, an `Update` reaching this session takes the same mutex, and
// holding it across the Send would deadlock the two against each other. This is
// the same rule `Mirror.Decide` follows for the chooser, one layer out.
func (s *session) send(message tea.Msg) {
	s.mu.Lock()
	out := s.out
	s.mu.Unlock()
	if out == nil {
		return
	}
	out.Send(message)
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
// ⚠️ **The cast book goes in as an option and the pool is kept beside it.** A
// room that drafts refuses a client with no cast book at the **welcome** — it
// could not compute its own pool — so a client dialled without one can never
// take part; and draft.NewPool is this repository's single declaration of "the
// cast minus every character held back", so the pool is built through it here
// rather than being derived anywhere a screen can reach. → socket.ClientOptions.
func (s *session) dial(code wire.RoomCode, hello wire.Hello, books battle.Books,
	characters *cast.Book) tea.Cmd {
	ctx := s.drafting(characters)
	return func() tea.Msg {
		client, err := socket.Dial(ctx, code, hello, books, socket.ClientOptions{
			Characters: characters,
			// ⚠️ **A nil chooser is refused at the point of being ASKED rather
			// than at the join**, so this is not optional for a client that can
			// land in a drafting room: without it, Play returns naming what is
			// missing the moment the first ban comes round. → socket.Client.answer.
			Draft: s.chooseDraft,
			// ⚠️ **The stamp goes ahead of the redraw**, on the same goroutine
			// and in the same hook: this is the moment this client can honestly
			// say a turn opened for it, and a model told to redraw before the
			// moment was recorded would draw a countdown against the turn
			// before. → session.observed.
			Stepped: func() {
				s.observed()
				s.send(matchSteppedMsg{})
			},
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
	// A stamp from the last match would be a countdown against a turn that is
	// over, and it has to be cleared here for the reason left does: a player who
	// leaves a room and joins another gets every guarantee back.
	s.clock = matchClock{}
	s.done = make(chan struct{})
	s.answers = make(chan pressed, 1)
	s.decisions = make(chan draftPressed, 1)
	// A pool from the last match would be the cast of a room this client has
	// left, and it has to be cleared here for the reason the clock is: a player
	// who leaves a room and joins another gets every guarantee back.
	s.characters, s.pooled = nil, nil
	return ctx
}

// drafting is open with the cast a drafting room's pool comes out of.
//
// Two steps rather than a parameter on open, because open is what a test with
// nothing to draft calls and there is nothing there for it to hand over.
func (s *session) drafting(characters *cast.Book) context.Context {
	ctx := s.open()
	if characters == nil {
		return ctx
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.characters = characters
	s.pooled = draft.NewPool(characters.All()).All()
	return ctx
}

// pool is the cast a drafting room's decisions are taken out of, as the screen
// draws it.
func (s *session) pool() []cast.Character {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pooled
}

// seat is which side of the wire this client is, and the empty Seat before a
// welcome has arrived — which means "nobody" and must not quietly mean the host.
func (s *session) seat() wire.Seat {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return ""
	}
	return client.Seat()
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
// → the three arms at the head of this file for what ends each.
func (s *session) choose(prompt *battle.Prompt) (battle.Choice, bool) {
	answers, ctx := s.turn()
	if answers == nil || ctx == nil {
		return battle.Choice{}, false
	}
	// Armed before the slot is read and before the message rather than after
	// them, so the allowance covers the whole of the wait a player can see.
	expired, stop := s.waitOut()
	defer stop()
	// ⚠️ **Read the slot first, and ask what it is FOR.** An answer buffered for
	// a turn that has already gone — the player answered, the server timed the
	// seat out and passed for it — must not be spent on the next one; an answer
	// buffered for *this* turn must not be thrown away.
	//
	// ⚠️ **This used to be a bare drain, and the bare drain deadlocked the
	// client for a whole allowance.** The premise was that nothing could be in
	// the slot for the open turn yet, because the chooser had not asked. It is
	// wrong: "it is your turn" is socket.Mirror.Asking, which is true the moment
	// the room's batch is taken in — one message and one screen redraw *before*
	// Play gets round to calling this — so a player answering off the board they
	// can already see lands in the slot ahead of the chooser. The drain then ate
	// a real decision, the screen had already recorded the turn as Answered and
	// would not offer it again, and both ends sat there until the allowance ran
	// out. Measured: cmd/hexarena-tui's loopback match test failed at 61.22s
	// against a 60s bound with the client on the battle screen, live, prompt
	// open, answered, and nothing sent for the whole minute.
	select {
	case held := <-answers:
		if held.about(prompt) {
			// No matchAskingMsg: the screen asked and answered before this
			// call, so there is nobody to tell.
			return held.answer.Choice, held.answer.Acted
		}
	default:
	}
	s.send(matchAskingMsg{})
	select {
	case held := <-answers:
		// Not matched against the prompt, and the asymmetry is deliberate: past
		// this line the only thing that can fill the slot is a keystroke taken
		// after matchAskingMsg went out, and the screen offers a turn it has
		// recorded as Answered to nobody. → PlayScreen.Answered.
		return held.answer.Choice, held.answer.Acted
	case <-ctx.Done():
		// Pass, and nothing is spent by it. Mirror.Decide reads the false as a
		// wire.Pass, Play sends it on a cancelled context, conn.send fails,
		// ended() covers context.Canceled, and Play returns nil.
		return battle.Choice{}, false
	case <-expired:
		// The same pass, on a live connection this time, so it is really sent.
		// The room has almost certainly passed for this seat already and will
		// refuse it with wire.CodeNotYourTurn — which is drawn, and is the
		// honest thing to tell somebody whose answer was too late. What matters
		// is that Play is out of Decide and reading the socket again.
		return battle.Choice{}, false
	}
}

// answer is a keystroke on its way to the chooser, and it **never blocks**.
//
// ⚠️ That is the reverse deadlock closed: an Update that waited for a chooser to
// take its answer would hang the whole program whenever nobody was asking —
// between turns, after the match, and on every keystroke that is not a decision.
// A dropped keystroke is the right answer there, because there is no turn for it
// to be about — which is also why a nil prompt drops: an answer that cannot say
// which turn it is for is one the chooser could not route.
func (s *session) answer(taken draw.PlayAnswer, about *battle.Prompt) {
	answers, _ := s.turn()
	if answers == nil || about == nil {
		return
	}
	select {
	case answers <- pressed{answer: taken, unit: about.Unit, turn: about.Turn}:
	default:
	}
}

// turn is the current match's channel and context, read together.
func (s *session) turn() (chan pressed, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.answers, s.ctx
}

// turnDraft is the draft's pair of the same two.
func (s *session) turnDraft() (chan draftPressed, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.decisions, s.ctx
}

// chooseDraft is the chooser socket.Client.Play calls when a draft decision is
// this client's, and it is the whole of how a keystroke becomes a ban.
//
// It is `choose`'s three arms over a different question, and the two things that
// differ are both about a draft having **no pass**:
//
//   - ⚠️ **A false here ENDS THE MATCH**, where a battle's false is a pass.
//     socket.Client.answer says so in as many words — *"a draft has no pass, so
//     there is nothing to send in its place"* — and that is the design: a draft
//     whose allowance runs out is cancelled outright with no auto-pick, because a
//     defaulted pick would hand somebody a squad they did not choose and call it
//     theirs. So the expiry arm is a **loud ending** on purpose, and the wait is
//     deliberately the same allowance-plus-grace the battle's is.
//   - ⚠️ **A stale decision is DROPPED and the wait goes on**, where the battle's
//     second select takes whatever is in the slot. It has to be: returning false
//     for one would end the match over a keystroke that arrived late, and the
//     stale answer would otherwise be spent on the next decision — which is
//     socket.DraftDue's whole reason for existing. Hence the loop.
//
// ⚠️ **`For` is filled off the PROMPT and only for a decision that matches it**,
// which is what makes socket.Mirror.DecideDraft's own check meaningful: an answer
// that named a decision of its own would be this client asserting rather than
// reporting, and the refusal that check counts would never fire.
func (s *session) chooseDraft(prompt socket.DraftPrompt) (socket.DraftAnswer, bool) {
	decisions, ctx := s.turnDraft()
	if decisions == nil || ctx == nil {
		return socket.DraftAnswer{}, false
	}
	expired, stop := s.waitOut()
	defer stop()
	// ⚠️ **Read the slot first and ask what it is FOR**, which is choose's own
	// measured ordering: a player learns a decision has come to them from their
	// own mirror, a message and a redraw before Play gets round to asking, so a
	// decision taken off the screen already in front of them lands here ahead of
	// the chooser. A bare drain would eat it.
	select {
	case held := <-decisions:
		if held.about(prompt) {
			return socket.DraftAnswer{For: prompt.Due, Decision: held.decision}, true
		}
	default:
	}
	s.send(matchAskingMsg{})
	for {
		select {
		case held := <-decisions:
			if !held.about(prompt) {
				// A decision for a decision that has gone. Dropped rather than
				// applied and rather than reported: the reader's screen still shows
				// what is open, so the next press is the answer.
				continue
			}
			return socket.DraftAnswer{For: prompt.Due, Decision: held.decision}, true
		case <-ctx.Done():
			return socket.DraftAnswer{}, false
		case <-expired:
			return socket.DraftAnswer{}, false
		}
	}
}

// decide is a keystroke on its way to the draft chooser, and it **never blocks**
// for the reason answer does not: an Update that waited for a chooser to take its
// decision would hang the whole program whenever nobody was asking.
//
// ⚠️ **There is deliberately no "already decided this one" memo, here or on the
// screen**, and that is the one guard this client may not add. A room that drafts
// sends nothing when its second seat is taken, so the host's first ban goes into a
// one-player room, is refused, and — because a refusal leaves the decision open —
// has to be **taken again**. A real player's chooser blocks on a keystroke rather
// than spinning, so the retry is a second press; a memo would make that press do
// nothing and stall the match for good. → TODO.md § step 5a, and
// draw.DraftLive.Waiting, which is the line that tells the player to press again.
func (s *session) decide(decision wire.DraftDecision, recorded int) {
	decisions, _ := s.turnDraft()
	if decisions == nil {
		return
	}
	select {
	case decisions <- draftPressed{decision: decision, recorded: recorded}:
	default:
	}
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
