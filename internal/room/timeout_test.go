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
	clients := newTable(t, dependencies, configuration.TurnCap)
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

// ⚠️ **Two tests were deleted here rather than left to rot, and this note is
// what they leave behind.** TestThreeConsecutiveTimeoutsForfeitAndAFourthIsNotNeeded
// and TestARealActionResetsTheTimeoutCount were both good tests of a mechanism
// that no longer exists: the per-seat tally of consecutive missed allowances,
// room.TimeoutLimit, and the branch that ended the match on the third. A timeout
// announces and passes the turn now, and nothing counts. Rewriting either would
// have left a test whose name described a rule and whose body measured nothing —
// the shape this repository has recorded as "shipped dead" pointed at a suite.
//
// What replaced them is not a smaller version of the same claim but the opposite
// one, measured on the board rather than on a counter:
// TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit is what the
// forfeit was for, and TestWhenNobodyAnswersTheTurnCapDrawsIt is the case where
// there is nobody left to punish. Between them they say the thing the counting
// was buying was already carried by the board.

// TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit is the measurement
// that made the forfeit unnecessary, and it is the whole argument for deleting
// it: a player who walks away from the keyboard **loses anyway**, because the
// opponent keeps acting and kills units that only ever pass.
//
// So the claim is stronger than "the match does not end early". It is that the
// match ends the ordinary way — a verdict of won, on the board, with a real
// winner and a standing the battles produced — while one seat answered nothing
// at all. Nobody is charged with anything and no room.Closed is sent.
func TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	const victim = wire.SeatGuest

	missed, steps := 0, 0
	for !opened.Finished() {
		onTurn, _ := answerFor(t, opened, clients, victim)
		if onTurn == victim {
			missed++
		}
		steps++
		if steps > 3*room.DefaultTurnCap {
			t.Fatalf("after %d turns and %d timeouts the match is still running", steps, missed)
		}
	}
	// The vacuity guard, and it is the sharpest thing here: with no tally there
	// is no bound on how many timeouts a match can take, so a run that happened
	// to contain three would have measured the mechanism that was deleted.
	if missed < 4 {
		t.Fatalf("the victim ran out of time %d times, which is inside the limit the deleted rule "+
			"used to have — this run cannot tell the new behaviour from the old", missed)
	}

	result := opened.Result()
	if result.Verdict != room.VerdictWon {
		t.Fatalf("a match one seat never answered ended as %q, want %q on the board",
			result.Verdict, room.VerdictWon)
	}
	if result.Winner != wire.SeatHost {
		t.Errorf("the match went to %q, want the seat that was answering", result.Winner)
	}
	if result.Departed.Valid() {
		t.Errorf("a timed-out match recorded %q as having gone away; a timeout is not a departure",
			result.Departed)
	}
	// And the standing is battles actually fought rather than a walkover.
	if got := len(opened.Played()); got == 0 {
		t.Error("the match ended having played no battles")
	}
	for _, client := range []*mirror{clients.host, clients.guest} {
		if len(client.closures) != 0 {
			t.Errorf("%s was told the room closed (%v); a timeout announces through the decision it "+
				"passes and needs no message of its own", client.seat, client.closures)
		}
	}
	t.Logf("won on the board after %d turns, %d of them the victim's allowance running out; standing %v",
		steps, missed, result.Wins)
}

// TestWhenNobodyAnswersTheTurnCapDrawsIt is the other half of the same argument,
// and it is the case a forfeit could never have handled well: with **both** seats
// timing out there is nobody to award the match to.
//
// The board answers it without a new rule. Neither side deals damage, so the
// battle cannot end itself, and the turn cap stops it as the draw the outcome
// already carries. That is what "the forfeit was carrying nothing the board does
// not already carry" means in the one case where the board has to carry it alone.
func TestWhenNobodyAnswersTheTurnCapDrawsIt(t *testing.T) {
	// A cap far below any real battle, so the cap is certainly what stops it.
	configuration := config(11, 1)
	configuration.TurnCap = 6
	opened, clients := openMatch(t, configuration)

	timeouts, steps := 0, 0
	for !opened.Finished() {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d turns nobody is on turn and the match is not over", steps)
		}
		out, err := opened.TimedOut(onTurn)
		if err != nil {
			t.Fatalf("%s's allowance running out: %v", onTurn, err)
		}
		clients.deliver(t, out)
		timeouts++
		steps++
		if steps > 4*configuration.TurnCap {
			t.Fatalf("the %d-turn cap did not stop a battle nobody was playing in %d turns",
				configuration.TurnCap, steps)
		}
	}
	if timeouts != steps {
		t.Fatalf("%d of %d turns were timeouts; this measures a battle nobody answered", timeouts, steps)
	}

	played := opened.Played()
	if len(played) != 1 {
		t.Fatalf("a bo1 played %d battles", len(played))
	}
	if !played[0].Capped {
		t.Fatalf("the battle ended as %q rather than at the cap", played[0].Outcome)
	}
	result := opened.Result()
	if result.Verdict != room.VerdictDrawn {
		t.Errorf("a battle nobody answered ended as %q, want %q — with both seats away there is "+
			"nobody to award it to", result.Verdict, room.VerdictDrawn)
	}
	if result.Departed.Valid() {
		t.Errorf("a capped match recorded %q as having gone away", result.Departed)
	}
	t.Logf("capped at %d turns, every one of them an allowance running out", played[0].Turns)
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

// TestATimeoutOnNothingIsRefusedAndSpendsNobodysTurn is the guard that used to
// be about the tally and is now about the turn, which is a **sharper** claim
// rather than a weaker one.
//
// With three-in-a-row forfeiting, the risk was a transport reporting a spurious
// timeout ending somebody's match through the back door. With nothing counted,
// the risk is worse in kind: a timeout the room accepted from the wrong seat
// would **spend the other player's turn for them** — a real decision entering
// the battle, in the log, off a report about a seat the room was not asking.
//
// Two shapes, because the two ways a spurious report can arrive are different:
// nobody is on turn at all, and somebody else is.
func TestATimeoutOnNothingIsRefusedAndSpendsNobodysTurn(t *testing.T) {
	t.Run("a room asking nobody anything", func(t *testing.T) {
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
		for attempt := 0; attempt < 6; attempt++ {
			out, err := opened.TimedOut(seat)
			if err != nil {
				t.Fatalf("timeout %d: %v", attempt, err)
			}
			if got := onlyCodeFor(t, out, seat); got != wire.CodeNotYourTurn {
				t.Fatalf("timeout %d answered %q, want %q", attempt, got, wire.CodeNotYourTurn)
			}
		}
		if opened.Finished() {
			t.Fatalf("timeouts on a room that was asking nobody anything gave the verdict %q",
				opened.Result().Verdict)
		}
	})

	t.Run("the seat that is not on turn", func(t *testing.T) {
		opened, clients := openMatch(t, config(11, 3))
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatal("nobody is on turn")
		}
		// The turn as it stands, so the refusal can be shown to have left it
		// exactly where it was.
		before := opened.Pending()
		if before == nil {
			t.Fatal("the room is waiting on a seat with no turn open")
		}
		other := wire.SeatHost
		if onTurn == other {
			other = wire.SeatGuest
		}
		out, err := opened.TimedOut(other)
		if err != nil {
			t.Fatalf("a timeout from the seat that is not on turn: %v", err)
		}
		if got := onlyCodeFor(t, out, other); got != wire.CodeNotYourTurn {
			t.Errorf("it answered %q, want %q", got, wire.CodeNotYourTurn)
		}
		after, stillWaiting := opened.Awaiting()
		if !stillWaiting || after != onTurn {
			t.Fatalf("the room is now waiting on %q (%v), want %q — a spurious timeout spent a turn",
				after, stillWaiting, onTurn)
		}
		if now := opened.Pending(); now == nil || now.Unit != before.Unit || now.Turn != before.Turn {
			t.Errorf("the open turn moved from %q's turn %d; a spurious timeout entered the battle",
				before.Unit, before.Turn)
		}
		// And the match is untouched: nothing went out to either client.
		for _, client := range []*mirror{clients.host, clients.guest} {
			if client.compared != 0 {
				t.Errorf("%s applied %d turns off a refused timeout", client.seat, client.compared)
			}
		}
		if opened.Finished() {
			t.Errorf("a timeout from the wrong seat ended the match as %q", opened.Result().Verdict)
		}
	})
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

// TestATimeoutTellsTheMirrorWithNoMessageOfItsOwn is the claim the departure
// message's absence rests on, and it is checked end to end through the encoder
// rather than read off the struct the room built.
//
// The reasoning is that a timeout needs no message because one already travels:
// the pass carries room.TimeoutReason, the reason is part of the
// battle.Decision, and the decision rides on wire.Turn. But battle.Decision
// tags that field `json:"reason,omitempty"`, which is exactly the sort of
// declaration a claim like this can be wrong about — so the message the room
// produced is **encoded and decoded**, the way it would cross a socket, and the
// reason is read out of the far end.
//
// Two things are asserted together, because either alone would be satisfied by
// the wrong design: the reason arrives, and **no kind but wire.Turn is sent**. A
// timeout that also announced itself would be a second declaration of a fact the
// decision already carries.
func TestATimeoutTellsTheMirrorWithNoMessageOfItsOwn(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatal("nobody is on turn")
	}
	out, err := opened.TimedOut(onTurn)
	if err != nil {
		t.Fatalf("timeout: %v", err)
	}
	if len(out) != seatsInARoom {
		t.Fatalf("a timeout produced %d messages, want one turn a seat", len(out))
	}
	told := 0
	for _, message := range out {
		if message.Body.Kind() != wire.KindTurn {
			t.Fatalf("a timeout produced a %s; the decision is the whole of what it says",
				message.Body.Kind())
		}
		raw, err := wire.Encode(message.Body)
		if err != nil {
			t.Fatalf("encode the turn for %s: %v", message.To, err)
		}
		decoded, err := wire.Decode(raw)
		if err != nil {
			t.Fatalf("decode the turn for %s: %v", message.To, err)
		}
		turn, ok := decoded.(*wire.Turn)
		if !ok {
			t.Fatalf("a turn decoded as %T", decoded)
		}
		if !turn.Decision.Passed {
			t.Errorf("%s was sent a decision that took %q rather than a pass",
				message.To, turn.Decision.Skill)
		}
		if turn.Decision.Reason != room.TimeoutReason {
			t.Errorf("%s reads the reason %q off the wire, want %q — the omitempty tag dropped it, "+
				"so a timeout is invisible to a mirror and does need a message of its own",
				message.To, turn.Decision.Reason, room.TimeoutReason)
			continue
		}
		told++
	}
	if told != seatsInARoom {
		t.Errorf("%d of %d clients learn of the timeout from the decision they were handed",
			told, seatsInARoom)
	}
	// And nothing else went out, now or afterwards: the match carries on.
	clients.deliver(t, out)
	for _, client := range []*mirror{clients.host, clients.guest} {
		if len(client.closures) != 0 {
			t.Errorf("%s was told the room closed (%v) on a timeout", client.seat, client.closures)
		}
	}
	if opened.Finished() {
		t.Errorf("one timeout ended the match as %q", opened.Result().Verdict)
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
