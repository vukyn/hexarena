package main

import (
	"time"

	tea "charm.land/bubbletea/v2"

	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
)

// # The client's clock
//
// This is the **fourth package in the module to read a clock**, and the whole of
// this one's reading is in this file. internal/room and internal/wire refuse
// `time` outright, internal/socket owns the countdown a room's allowance is
// actually enforced by, and cmd/hexarena-host has one for its shutdown. Each of
// those bans is an AST walk over its own package directory, which is precisely
// what a fourth package escapes — so
// internal/socket's TestEveryClockInTheModuleIsOnTheAllowlist walks the whole
// module and holds every clock reader against a written list with a reason. This
// file is one entry on it, and the reason it is one file rather than three is
// that a list of "which file may read a clock" is only worth keeping if the
// answer stays small enough to read.
//
// ## Two things that share a clock, and one that must not have one
//
//  1. **The countdown drawn on a live battle**, so a player can see the other
//     one thinking rather than watching a screen that looks frozen for up to a
//     whole allowance.
//  2. **The chooser's third arm**, which gives up on a prompt nothing can
//     answer. → session.choose.
//
// And internal/screen, which draws the countdown, has no clock and must not gain
// one: it is handed two counts of seconds and draws them. → draw.PlayClock,
// which carries that argument where the field is.
//
// ## ⚠️ Nothing about this goes on the wire, and TODO.md used to ask for it to
//
// The item this landed under called for "a remaining duration on the wire". It
// is not needed, and the mirror is why: both peers apply the same wire.Turn and
// open the same prompt, so **both clients already know, locally, the moment a
// turn opened and whose it is** — and Welcome.Allowance is already known to
// both. So each client counts down for whichever seat is on turn and draws both
// clocks, with no new message kind and no change to any existing one.
//
// The reasoning that item gave for a *duration rather than a deadline* — two
// machines on a LAN have no reason to agree what time it is — is exactly right,
// and it is why a count from a **locally observed event** is the correct shape
// rather than a compromise: nothing here reads a timestamp anybody else wrote.
//
// The cost is that the two displays drift by the network hop and by when each
// client got round to processing the event: tens of milliseconds on a LAN
// against a ninety-second allowance. That is affordable because **the display is
// advisory and the room's timer is authoritative** — a client whose countdown is
// wrong still learns the real outcome, because a timeout arrives as a pass event
// like any other and the board says what happened.

// chooserGrace is how long past the room's own allowance this client waits
// before giving up on a prompt itself.
//
// ⚠️ **The client has to be the SECOND to give up, and the grace is the margin
// for that.** The room arms its timer when it produces the turn; this client
// starts counting when it receives one, which is a network hop later — so this
// client is already behind by construction, and what the grace covers is the
// remainder: two machines' monotonic clocks running at slightly different rates
// (tens of parts per million, so ~10ms over a ninety-second allowance), a timer
// firing coarsely on a loaded machine, and a room whose own arming was delayed.
// Two seconds is three orders of magnitude of headroom over the drift.
//
// The other end of the bound is the player waiting on a peer that has died: they
// wait the allowance plus this, so it has to be small against the allowance
// (two seconds is ~2% of the default ninety) and small against
// socket.DefaultCloseThreshold, which is the sixty seconds after which a silent
// peer ends the match anyway. A grace of nought would make this client race the
// room for who passes first, and a grace of a minute would make it useless.
const chooserGrace = 2 * time.Second

// clockRate is how often a live battle redraws itself so the countdown moves.
//
// ⚠️ **Without it the countdown would only move when a message arrived**, which
// is the opposite of what it is for: the whole point is the screen during the
// *other* player's turn, when by definition nothing is arriving. A second is the
// resolution the drawing has — draw.PlayClock counts in seconds — so a faster
// tick would redraw the same bytes and a slower one would skip numbers.
const clockRate = time.Second

// clockTickMsg is the countdown asking for a redraw. It carries nothing, like
// every other message this client's match sends: the model re-reads the mirror.
type clockTickMsg struct{}

// clockTick is one tick, re-armed by whoever handles it. bubbletea has no
// repeating timer — a Cmd fires once — so the re-arm is the model's, which is
// also what lets it stop ticking when the match is over.
func clockTick() tea.Cmd {
	return tea.Tick(clockRate, func(time.Time) tea.Msg { return clockTickMsg{} })
}

// turnKey is which open turn a countdown is being counted against.
//
// ⚠️ **No *battle.Battle in it, deliberately.** This is read inside
// socket.Mirror.Read and kept afterwards, and nothing a Sight hands over may
// outlive the callback. The battle within the series is named by its index
// instead, which is a number.
type turnKey struct {
	battle int
	unit   string
	turn   int
}

// matchClock is when this client saw the open turn open. One per match, reset by
// session.open with everything else.
type matchClock struct {
	open turnKey
	at   time.Time
}

// openTurnOf is the turn the mirror is stopped on, whichever side it belongs to.
//
// ⚠️ **socket.Sight.Asking is NOT what this asks**, and that is the difference
// between one clock and two: Asking is nil on the other player's turn, which is
// exactly the turn this feature exists to draw. So the prompt comes off the
// battle itself and the side is read off the unit it names.
//
// Past the cap there is no turn: the room stops asking on the turn this client
// stops at, so a counted-down clock there would be counting something nobody is
// waiting for.
func openTurnOf(sight socket.Sight) turnKey {
	if sight.Fight == nil || sight.Capped {
		return turnKey{}
	}
	prompt := sight.Fight.Pending()
	if prompt == nil {
		return turnKey{}
	}
	return turnKey{battle: sight.Index, unit: prompt.Unit, turn: prompt.Turn}
}

// clockOf is the whole countdown, as a pure function of a reading, the moment
// the turn was observed and the moment now.
//
// It reads no clock itself — `now` is a parameter — so the arithmetic is
// measurable without waiting for one, and the single call to time.Now is the
// caller's.
//
// ⚠️ **Only the seat on turn counts down.** The allowance is per prompt, not per
// match: a room runs no chess clock, so the player who is not being asked has
// the whole of it waiting for them and that is what their number says.
func clockOf(sight socket.Sight, opened, now time.Time) draw.PlayClock {
	open := openTurnOf(sight)
	if open == (turnKey{}) || opened.IsZero() || sight.Welcome.Allowance <= 0 {
		return draw.PlayClock{}
	}
	unit, known := sight.Fight.Unit(open.unit)
	if !known {
		return draw.PlayClock{}
	}
	whole := sight.Welcome.Allowance
	left := remaining(whole, now.Sub(opened))
	if unit.Side == sight.Side {
		return draw.PlayClock{Waiting: draw.PlayClockYou, Yours: left, Theirs: whole}
	}
	return draw.PlayClock{Waiting: draw.PlayClockThem, Yours: whole, Theirs: left}
}

// remaining is how many seconds are left of an allowance opened `since` ago.
//
// Rounded **up**, so a turn that has just opened shows the whole allowance
// rather than a second less than it, and every number is on the screen for a
// whole second. Nothing below nought: an allowance that has run out has none
// left, and what happens next is the room's to say.
func remaining(allowance int, since time.Duration) int {
	left := socket.Allowance(allowance) - since
	if left <= 0 {
		return 0
	}
	return int((left + time.Second - 1) / time.Second)
}

// observed stamps the moment this client saw the open turn change.
//
// Called from the hook that fires on every message the match takes in, on the
// Play goroutine, which is the earliest this client can honestly say a turn
// opened for it. It is the pair below with the clock read, so that the whole of
// this package's reading of one is in this file and the stamping is measurable
// without waiting for a real second to pass.
func (s *session) observed() { s.observedAt(time.Now()) }

// observedAt is observed against a given moment.
//
// ⚠️ **The lock order is the mirror's read lock, then the session's**, which is
// the order every other pair here takes: session.read releases s.mu *before* it
// asks the mirror for anything, so nothing in this client holds the session's
// lock while reaching for the mirror's and there is no cycle to have.
func (s *session) observedAt(now time.Time) {
	var open turnKey
	if !s.read(func(sight socket.Sight) { open = openTurnOf(sight) }) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock.open != open {
		s.clock = matchClock{open: open, at: now}
	}
}

// countdown is what the battle screen draws, for the reading in hand.
//
// ⚠️ **A reading whose turn is not the one that was stamped draws nothing.** The
// stamp is taken on the Play goroutine before the redraw is asked for, so the
// two agree on every real path; if they ever did not, this client would not know
// when that turn opened, and a countdown from a moment it made up would be worse
// than none.
func (s *session) countdown(sight socket.Sight) draw.PlayClock {
	s.mu.Lock()
	clock := s.clock
	s.mu.Unlock()
	if clock.open != openTurnOf(sight) {
		return draw.PlayClock{}
	}
	return clockOf(sight, clock.at, time.Now())
}

// waitOut is the chooser's third arm: the channel it gives up on, and the stop
// that goes with it.
//
// ⚠️ **A room with no allowance gets a nil channel, which is an arm that is not
// there.** A receive on nil blocks for ever, so a select over one is the two-arm
// select this used to be — and that is the right reading, because a room whose
// allowance is nought arms no timer of its own either (socket.Allowance answers
// nought and the transport starts nothing), so a client giving up on such a
// prompt would be the *first* to give up rather than the second.
func (s *session) waitOut() (<-chan time.Time, func()) {
	length := s.patience()
	if length <= 0 {
		return nil, func() {}
	}
	timer := time.NewTimer(length)
	return timer.C, func() { timer.Stop() }
}

// patience is how long this client waits on one prompt, off the room's own
// configuration. A session with no client, or one whose welcome has not arrived,
// waits on nothing — there is no allowance to be second to.
func (s *session) patience() time.Duration {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return 0
	}
	welcome, seated := client.Mirror().Welcome()
	if !seated {
		return 0
	}
	return patienceFor(welcome.Allowance)
}

// patienceFor is the allowance plus the grace, and the conversion is
// socket.Allowance because that is the one place in the module a wire.Welcome's
// seconds become a duration. A client repeating that arithmetic would be a
// second declaration of what the protocol's number means.
func patienceFor(allowance int) time.Duration {
	length := socket.Allowance(allowance)
	if length <= 0 {
		return 0
	}
	return length + chooserGrace
}
