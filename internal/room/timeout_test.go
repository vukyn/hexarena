package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// openMatch seats two clients and returns the room with the first battle open,
// so the tests below start where they mean to.
func openMatch(t *testing.T, configuration room.Config) (*room.Room, *table) {
	t.Helper()
	dependencies := deps(t)
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies.Books, configuration.TurnCap)
	for _, one := range []struct {
		name  string
		squad []string
	}{
		{name: "Host", squad: []string{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"}},
		{name: "Guest", squad: []string{"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"}},
	} {
		squad := squadOf(t, dependencies.Characters, one.name+".squad", one.squad...)
		_, out, err := opened.Join(hello(t, squad, one.name))
		if err != nil {
			t.Fatalf("%s joins: %v", one.name, err)
		}
		clients.deliver(t, out)
	}
	if _, waiting := opened.Awaiting(); !waiting {
		t.Fatal("both seats are taken and the room is waiting on nobody")
	}
	return opened, clients
}

// answerFor plays one turn: the seat on turn either answers or is reported to
// have run out of time, depending on whose it is and what the caller wants.
func answerFor(t *testing.T, opened *room.Room, clients *table, timeOut wire.Seat) (wire.Seat, []room.Outbound) {
	t.Helper()
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatal("nobody is on turn")
	}
	var out []room.Outbound
	var err error
	if onTurn == timeOut {
		out, err = opened.TimedOut(onTurn)
	} else {
		out, err = opened.Deliver(onTurn, clients.at(onTurn).answer())
	}
	if err != nil {
		t.Fatalf("%s's turn: %v", onTurn, err)
	}
	clients.deliver(t, out)
	return onTurn, out
}

// TestThreeConsecutiveTimeoutsForfeitAndAFourthIsNotNeeded is the counting half
// of the clock, and it is counting rather than timing: the room is *told* an
// allowance ran out and never asks how long anything took.
//
// A disconnected client is not a slow one, which is what the limit is for.
// Ninety seconds a turn over a forty-turn battle is an hour of somebody sitting
// in front of a dead opponent.
func TestThreeConsecutiveTimeoutsForfeitAndAFourthIsNotNeeded(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	const victim = wire.SeatGuest

	missed, steps := 0, 0
	var last []room.Outbound
	for !opened.Finished() {
		onTurn, out := answerFor(t, opened, clients, victim)
		if onTurn == victim {
			missed++
			last = out
		}
		steps++
		if steps > 200 {
			t.Fatalf("after %d turns and %d timeouts the match is still running", steps, missed)
		}
	}
	if missed != room.TimeoutLimit {
		t.Fatalf("the match ended after %d timeouts, want exactly %d — a fourth is not needed",
			missed, room.TimeoutLimit)
	}
	// The forfeiting timeout sends nothing at all, which is a gap in the
	// protocol rather than a decision: the seven messages have no way to say
	// "the match is over and here is why". → the note on Room.forfeit.
	if len(last) != 0 {
		t.Errorf("the forfeiting timeout produced %d messages", len(last))
	}

	result := opened.Result()
	if result.Verdict != room.VerdictForfeited {
		t.Errorf("three timeouts gave the verdict %q, want %q", result.Verdict, room.VerdictForfeited)
	}
	if result.Forfeit != room.ForfeitTimedOut {
		t.Errorf("three timeouts were recorded as %q, want %q", result.Forfeit, room.ForfeitTimedOut)
	}
	if result.Loser != victim {
		t.Errorf("the forfeit was charged to %q, want %q", result.Loser, victim)
	}
	if result.Winner != wire.SeatHost {
		t.Errorf("the forfeit was awarded to %q, want the other seat", result.Winner)
	}
	// And a fourth timeout is not merely unnecessary, it is refused: the match
	// is over, so nobody is on turn.
	out, err := opened.TimedOut(victim)
	if err != nil {
		t.Fatalf("a fourth timeout: %v", err)
	}
	if got := onlyCodeFor(t, out, victim); got != wire.CodeNotYourTurn {
		t.Errorf("a timeout after the match answered %q, want %q", got, wire.CodeNotYourTurn)
	}
	t.Logf("forfeited after %d turns and %d timeouts", steps, missed)
}

// TestARealActionResetsTheTimeoutCount is the half that is easy to get wrong,
// and getting it wrong is not a crash: a counter that never reset would forfeit
// a merely slow player somewhere in the middle of a long match, having counted
// three misses spread across twenty minutes as though they were consecutive.
//
// The shape is two misses, one real answer, two more misses — five in total,
// which is well past the limit of three — and the match has to still be running.
// Then one more miss forfeits, which is what says the counter reset rather than
// stopped counting.
func TestARealActionResetsTheTimeoutCount(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	const victim = wire.SeatGuest

	// miss makes the victim's next prompt run out of time, letting the other
	// seat answer normally in between, and reports how many turns that took.
	miss := func(times int) {
		t.Helper()
		for missed := 0; missed < times; {
			if opened.Finished() {
				t.Fatalf("the match ended after %d of %d intended timeouts", missed, times)
			}
			if onTurn, _ := answerFor(t, opened, clients, victim); onTurn == victim {
				missed++
			}
		}
	}
	// answer makes the victim answer its own next prompt for real.
	answer := func() {
		t.Helper()
		for {
			if opened.Finished() {
				t.Fatal("the match ended before the victim could answer")
			}
			if onTurn, _ := answerFor(t, opened, clients, ""); onTurn == victim {
				return
			}
		}
	}

	miss(room.TimeoutLimit - 1)
	if opened.Finished() {
		t.Fatalf("%d timeouts forfeited a match; the limit is %d", room.TimeoutLimit-1, room.TimeoutLimit)
	}
	answer()
	miss(room.TimeoutLimit - 1)
	if opened.Finished() {
		t.Fatalf("%d timeouts either side of a real answer forfeited the match: the count did not reset",
			2*(room.TimeoutLimit-1))
	}
	// The counter still works, so the reset put it back to nought rather than
	// switching it off.
	miss(1)
	if !opened.Finished() {
		t.Fatalf("%d timeouts after a reset did not forfeit", room.TimeoutLimit)
	}
	if got := opened.Result().Forfeit; got != room.ForfeitTimedOut {
		t.Errorf("the forfeit was recorded as %q, want %q", got, room.ForfeitTimedOut)
	}
}

// TestASkippedPromptStartsNoClock is the third detail that follows from "the
// clock is not part of the battle": a unit that has already lost its action, to
// control or to a timed effect, is not being asked anything, so there is nothing
// for an allowance to run out on.
//
// It is a property of the room's own loop rather than a rule the transport has
// to remember — settle walks past a skipped prompt without ever leaving it open
// — and that is exactly why it needs a count to be measured against: a skipped
// turn produces no decision and therefore no message, so a run that happened to
// contain none would pass this test having observed nothing.
func TestASkippedPromptStartsNoClock(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))

	steps := 0
	for !opened.Finished() {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d turns nobody is on turn and the match is not over", steps)
		}
		// Whenever the room is waiting on a seat, the turn it is waiting on is
		// one somebody can actually take.
		pending := opened.Pending()
		if pending == nil {
			t.Fatalf("the room is waiting on %s with no turn open", onTurn)
		}
		if pending.Skipped {
			t.Fatalf("the room is waiting on %s to answer %q's turn, which is skipped (%s) — "+
				"an allowance would be running on a turn nobody is being asked about",
				onTurn, pending.Unit, pending.Reason)
		}
		answerFor(t, opened, clients, "")
		steps++
		if steps > 400 {
			t.Fatal("the match is not progressing")
		}
	}
	// The vacuity guard. Without skipped prompts in the run the loop above is a
	// loop that checked nothing.
	if opened.Skipped() == 0 {
		t.Fatalf("the match walked past no skipped prompts over %d turns, so this test measured nothing",
			steps)
	}
	t.Logf("%d turns, %d prompts walked past unaskable", steps, opened.Skipped())
}

// TestATimeoutOnNothingIsRefusedAndCountsNothing is the other half of the same
// claim, and it is the one that would let a forfeit in through the back door: a
// room that counted every TimedOut it was handed would forfeit a player whose
// transport reported a timeout on a turn the room was not asking about.
func TestATimeoutOnNothingIsRefusedAndCountsNothing(t *testing.T) {
	dependencies := deps(t)
	opened := newRoom(t, config(11, 1))
	squad := squadOf(t, dependencies.Characters, "one.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	seat, _, err := opened.Join(hello(t, squad, "Alone"))
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, waiting := opened.Awaiting(); waiting {
		t.Fatal("a room with one player is waiting on somebody to act")
	}
	// Well past the limit, so a room that counted these would have forfeited
	// several times over.
	for attempt := 0; attempt < 2*room.TimeoutLimit; attempt++ {
		out, err := opened.TimedOut(seat)
		if err != nil {
			t.Fatalf("timeout %d: %v", attempt, err)
		}
		if got := onlyCodeFor(t, out, seat); got != wire.CodeNotYourTurn {
			t.Fatalf("timeout %d answered %q, want %q", attempt, got, wire.CodeNotYourTurn)
		}
	}
	if opened.Finished() {
		t.Fatalf("%d timeouts on a room that was asking nobody anything gave the verdict %q",
			2*room.TimeoutLimit, opened.Result().Verdict)
	}
	if opened.Result().Forfeit != room.ForfeitNone {
		t.Errorf("a room that asked nobody anything recorded the forfeit %q", opened.Result().Forfeit)
	}
}

// TestATimeoutEntersTheBattleAsAPassAndNotAsATime is the reason none of this
// reaches the log: what a timeout puts into the battle is a decision, with a
// single constant reason, and never a reading of a clock. So --verify cannot
// tell a timed-out match from any other.
func TestATimeoutEntersTheBattleAsAPassAndNotAsATime(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatal("nobody is on turn")
	}
	out, err := opened.TimedOut(onTurn)
	if err != nil {
		t.Fatalf("timeout: %v", err)
	}
	var turns int
	for _, message := range out {
		turn, ok := message.Body.(wire.Turn)
		if !ok {
			t.Fatalf("a timeout produced a %s", message.Body.Kind())
		}
		turns++
		if !turn.Decision.Passed {
			t.Errorf("a timeout produced a decision that took %q", turn.Decision.Skill)
		}
		if turn.Decision.Reason != room.TimeoutReason {
			t.Errorf("a timeout recorded the reason %q, want the single constant %q",
				turn.Decision.Reason, room.TimeoutReason)
		}
		if _, aimed := turn.Decision.Aim.Offset(); aimed {
			t.Error("a timed-out turn aims somewhere")
		}
	}
	if turns != seatsInARoom {
		t.Errorf("a timeout was reported to %d clients, want %d", turns, seatsInARoom)
	}
	// The mirror applied the same decision from the same wire and agreed on the
	// digest, which is checked inside mirror.apply — so the timed-out turn is a
	// turn both engines produced identically.
	clients.deliver(t, out)
	if clients.host.compared != 1 || clients.guest.compared != 1 {
		t.Errorf("the timed-out turn was checked %d and %d times, want once each",
			clients.host.compared, clients.guest.compared)
	}
	// And the pass is in the record as an ordinary skipped turn: nothing about
	// it is a new kind of event.
	closing := lastEvent(t, clients.host.events, battle.TurnSkipped)
	if closing.Note != room.TimeoutReason {
		t.Errorf("the timed-out turn reads %q in the log, want %q", closing.Note, room.TimeoutReason)
	}
}

// seatsInARoom is two, restated here rather than exported from the package: how
// many seats a room has is the room's own business, and a test that read it off
// the package could not notice the package changing its mind.
const seatsInARoom = 2

// onlyCodeFor is onlyCode for a refusal addressed to a seat, which every refusal
// after the gate is.
func onlyCodeFor(t *testing.T, out []room.Outbound, seat wire.Seat) wire.Code {
	t.Helper()
	if len(out) != 1 {
		t.Fatalf("the room answered with %d messages, want one refusal", len(out))
	}
	refused, ok := out[0].Body.(wire.Refused)
	if !ok {
		t.Fatalf("the room answered with a %s, want a refusal", out[0].Body.Kind())
	}
	if out[0].To != seat {
		t.Errorf("the refusal was addressed to %q, want %q", out[0].To, seat)
	}
	return refused.Code
}
