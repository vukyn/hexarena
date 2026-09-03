package socket

import (
	"context"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestATimeoutPassesTheTurnOverARealSocket is the clock end to end: the
// transport counts an allowance down, the room takes the report as an **input**
// and applies a Pass with room.TimeoutReason, and the mirror is told by the one
// declaration of it that travels — Decision.Reason on wire.Turn, which reaches
// the client's own battle as the note on a TurnSkipped.
//
// ⚠️ **A timeout needs no message of its own**, and this is what says so over a
// socket: nothing here reads a message kind. What is looked for is the *reason*
// arriving inside a decision the client applied to its own engine.
//
// The allowance is one second, which is the shortest room.Config.Validate
// accepts, and both clients deliberately think for longer than that on their
// first turn — so whichever the room asks first loses that turn to the clock.
// After that they answer at once and the match plays out, which is the other
// half of the claim: a timeout **announces and passes the turn**, and nothing is
// forfeited.
func TestATimeoutPassesTheTurnOverARealSocket(t *testing.T) {
	dependencies := deps(t)
	// A bo1, so the mirror's own event record covers the whole match — it starts
	// afresh per battle.
	hurried := config(11, 1, 1)
	held := listening(t, Timings{})
	code := held.open(t, hurried, dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostPlay := play(ctx, host, thinkingFirst(rating(host), 3*time.Second))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, thinkingFirst(rating(guest), 3*time.Second))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)

	// The turn was passed and the reason travelled, into each client's own
	// battle. Both clients see it, because every turn goes to both.
	for _, client := range []*Client{host, guest} {
		if timeouts := timedOutTurns(client.Mirror().Events()); timeouts == 0 {
			t.Errorf("%s's own battle records no turn lost to the clock over a %ds allowance",
				client.Seat(), hurried.Allowance)
		}
	}
	// And the match still ended the ordinary way: a timeout announces and passes
	// the turn, so nobody forfeits.
	switch done.reading.Result.Verdict {
	case room.VerdictWon, room.VerdictDrawn:
	default:
		t.Errorf("a match with timeouts in it ended %q; a timeout costs nothing",
			done.reading.Result.Verdict)
	}
	if done.reading.Result.Departed.Valid() {
		t.Errorf("a timeout recorded %q as having gone away", done.reading.Result.Departed)
	}
	t.Logf("%d turns lost to a %ds allowance; verdict %q over %d battles",
		timedOutTurns(host.Mirror().Events()), hurried.Allowance,
		done.reading.Result.Verdict, len(done.reading.Played))
}

// TestALateTimeoutIsRefusedWithoutDroppingAnybody is the shape that drops a
// player for answering **quickly** if it is got wrong.
//
// A timer armed for one seat can fire after that seat has already answered and
// the room has moved on — the answer and the fire genuinely race, and no amount
// of stopping the timer closes that window. room.TimedOut refuses a seat it is
// not asking, with wire.CodeNotYourTurn, so the report is already harmless; what
// this holds is the transport's side of it:
//
//   - it is **not** a reason to close anything;
//   - the refusal is **not forwarded**, because the transport owns the timeout
//     and therefore owns the answer to it — a wire.Refused the client never
//     provoked would be a refusal of a question it never asked;
//   - the clock carries on, so the match finishes normally.
//
// ⚠️ The timer is fired **by hand**, at a moment this test chose, and the reason
// is at the head of fixtures_test.go: driving the race from outside means
// sleeping and hoping. The match is held still at a prompt — the awaited client
// is blocked inside its own chooser — so the seat that is *not* being asked is
// certain, and that is exactly the input a late timer produces.
func TestALateTimeoutIsRefusedWithoutDroppingAnybody(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 1, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	ctx := context.Background()
	resume := make(chan struct{})
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostChoose, hostPaused := paused(rating(host), resume)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestChoose, guestPaused := paused(rating(guest), resume)
	guestPlay := play(ctx, guest, guestChoose)

	// Whichever seat the room asks first stops there, so the board is still.
	select {
	case <-hostPaused:
	case <-guestPaused:
	case <-time.After(theWholeMatch):
		t.Fatalf("neither client was asked to act inside %s", theWholeMatch)
	}
	reading, running := held.rooms.Read(code)
	if !running || !reading.Waiting {
		t.Fatalf("the room is not waiting on anybody with a client paused at its prompt")
	}
	entry := held.tableFor(t, code)
	// The seat the room is **not** asking, which is what a timer for the seat
	// that has just answered looks like from the room's side.
	late := otherSeat(reading.Awaiting)
	if before := entry.lateTimeouts(); before != 0 {
		t.Fatalf("the table already counted %d late timeouts before one was fired", before)
	}
	held.server.timedOut(ctx, code, entry, late)
	if counted := entry.lateTimeouts(); counted != 1 {
		t.Errorf("a timeout for %s while the room waits on %s counted %d, want 1",
			late, reading.Awaiting, counted)
	}

	// Nobody was dropped: the match plays out from where it was held.
	close(resume)
	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match after a late timeout: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match after a late timeout: %v", err)
	}
	done := held.finished(t)
	switch done.reading.Result.Verdict {
	case room.VerdictWon, room.VerdictDrawn:
	default:
		t.Errorf("a late timeout left the match ending %q", done.reading.Result.Verdict)
	}
	// And the refusal never reached a client, which is the half that would
	// otherwise show up on somebody's screen as a refusal they never provoked.
	for _, client := range []*Client{host, guest} {
		if refusals := client.Mirror().Refusals(); len(refusals) != 0 {
			t.Errorf("%s was refused %v; a late timeout is the transport's own business",
				client.Seat(), refusals)
		}
	}
	// Nor did the turn move: the seat that was being asked is the seat that
	// answered, so the timeout spent nobody's turn.
	if timeouts := timedOutTurns(host.Mirror().Events()); timeouts != 0 {
		t.Errorf("a refused timeout still put %d passes into the battle", timeouts)
	}
	t.Logf("one late timeout for %s while the room waited on %s: refused, %q over %d battles",
		late, reading.Awaiting, done.reading.Result.Verdict, len(done.reading.Played))
}

// TestASkippedPromptStartsNoClockOverASocket is the room's own property seen
// from the transport, and it is here because the transport is what would break
// it: an allowance is started on room.Reading.Awaiting, so "a Skipped prompt
// starts no clock" holds only while the room never leaves one open.
//
// ⚠️ It is **verified rather than assumed**, which is what the brief asked for:
// the match is played with an allowance of one second and a rating that answers
// at once, so any prompt the room left open across a skipped turn would time out
// — and Reading.Skipped says a real match had skipped turns in it, so the claim
// is not vacuous.
func TestASkippedPromptStartsNoClockOverASocket(t *testing.T) {
	dependencies := deps(t)
	hurried := config(11, 1, 1)
	held := listening(t, Timings{})
	code := held.open(t, hurried, dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostPlay := play(ctx, host, rating(host))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	if done.reading.Skipped == 0 {
		t.Fatal("the match had no skipped prompts in it, so this measures nothing")
	}
	if timeouts := timedOutTurns(host.Mirror().Events()); timeouts != 0 {
		t.Errorf("%d turns were lost to the clock over %d skipped prompts, with both clients answering at once",
			timeouts, done.reading.Skipped)
	}
	t.Logf("%d prompts skipped over %d battles and no clock ran out",
		done.reading.Skipped, len(done.reading.Played))
}

// timedOutTurns is how many turns in a client's own battle were passed because
// an allowance ran out, read off the note battle.Pass records — which is
// room.TimeoutReason travelling inside the decision.
func timedOutTurns(events []battle.Event) int {
	lost := 0
	for _, event := range events {
		if event.Kind == battle.TurnSkipped && event.Note == room.TimeoutReason {
			lost++
		}
	}
	return lost
}

// thinkingFirst is a chooser that takes its time over its **first** decision and
// answers at once after that, which is a player who looked away once.
func thinkingFirst(choose battle.Chooser, waiting time.Duration) battle.Chooser {
	first := true
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		if first {
			first = false
			time.Sleep(waiting)
		}
		return choose(prompt)
	}
}

// paused is a chooser that stops on its first decision and waits to be let go,
// so a test can hold a match still at a prompt.
//
// It signals through a channel rather than by having the test read the client's
// Mirror, which belongs to the client's own loop — a reading from another
// goroutine is a race the detector would find and would be right about.
func paused(choose battle.Chooser, resume <-chan struct{}) (battle.Chooser, <-chan struct{}) {
	reached := make(chan struct{})
	first := true
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		if first {
			first = false
			close(reached)
			<-resume
		}
		return choose(prompt)
	}, reached
}

// otherSeat is the seat that is not this one, for a test naming the seat the
// room is not asking.
func otherSeat(seat wire.Seat) wire.Seat {
	if seat == wire.SeatHost {
		return wire.SeatGuest
	}
	return wire.SeatHost
}
