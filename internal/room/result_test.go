package room_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// theBattlesOutcomes is how many ways a battle can end, restated here as a
// literal on purpose.
//
// ⚠️ Reading battle.OutcomeCount and comparing it to itself would agree with any
// number at all. Four is the number, it has been four since the outcome became a
// typed field, and the day somebody adds a fifth *for the room's sake* this line
// is the red test that asks them not to. A departure, a dropped socket and a
// refused join are results of the **match**; a battle.Outcome is read by
// --verify and by every renderer, and a dropped socket is not a way a battle can
// end.
const theBattlesOutcomes = 4

// TestADepartureAddsNothingToTheBattlesOutcomes drives a match to the one route
// that ends it early, so that route is not the dead branch this repository has
// recorded several times, and checks that the core enum is exactly where it was.
//
// ⚠️ **It used to have two cases and now has one**, and the arithmetic of that is
// worth reading rather than the diff. The two were the two routes a forfeit had:
// three missed allowances, and a peer that went away. Neither forfeits now —
// nothing does — so the timeout case moved out to
// TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit, where the claim is
// that the match ends the *ordinary* way. What survives here unchanged is the
// claim worth keeping: whatever the room does about a match, **nothing is added
// to battle.Outcome**, and OutcomeCount against a literal is still the way to
// hold it.
func TestADepartureAddsNothingToTheBattlesOutcomes(t *testing.T) {
	if battle.OutcomeCount != theBattlesOutcomes {
		t.Fatalf("battle.OutcomeCount is %d and was %d: if a way for a *match* to end has been "+
			"added to the enum a *battle* ends by, it belongs in room.Verdict instead",
			battle.OutcomeCount, theBattlesOutcomes)
	}

	opened, clients := openMatch(t, config(11, 3))
	const victim = wire.SeatGuest
	// One real turn first, so the departure lands in a match that was genuinely
	// under way rather than in the opening — which is a different case, answered
	// by TestALeavingPeerBeforeTheMatchFreesItsSeat.
	answerFor(t, opened, clients, "")
	out, err := opened.Left(victim)
	if err != nil {
		t.Fatalf("leave: %v", err)
	}
	clients.deliver(t, out)

	if !opened.Finished() {
		t.Fatal("the match is not over")
	}
	result := opened.Result()
	if result.Verdict != room.VerdictAbandoned {
		t.Errorf("the verdict is %q, want %q", result.Verdict, room.VerdictAbandoned)
	}
	// Nobody wins and nobody loses, which is the whole of what changed: the
	// match was not played out, so it is not awarded.
	if result.Winner.Valid() {
		t.Errorf("an abandoned match names %q as its winner", result.Winner)
	}
	if result.Departed != victim {
		t.Errorf("the departure was recorded as %q, want %q", result.Departed, victim)
	}
	// Whatever battle it interrupted is **not** recorded as having ended: a
	// departure is a result of the match, so the battle simply stops being asked
	// about.
	for index, one := range opened.Played() {
		if one.Outcome == battle.Undecided && !one.Capped {
			t.Errorf("battle %d was recorded as undecided and uncapped, so a departure wrote a battle result",
				index+1)
		}
		if int(one.Outcome) >= battle.OutcomeCount {
			t.Errorf("battle %d was recorded with the outcome %q, which is not one of the %d",
				index+1, one.Outcome, battle.OutcomeCount)
		}
	}
	// The room is waiting on nobody, so a transport that kept reading would not
	// be handed a turn.
	if seat, waiting := opened.Awaiting(); waiting {
		t.Errorf("an abandoned match is still waiting on %q", seat)
	}
}

// TestALeavingPeerIsAnnouncedToTheOneStillThere is the message half, and it is
// the one ending in the protocol that needs one.
//
// ⚠️ A mirror cannot reach this state on its own, which is the whole argument for
// the message existing at all. There is no Ended for the battle in progress —
// the engine concluded nothing about it — and no further Start, so a client
// handed nothing would sit on its own open prompt waiting for a turn that is
// never coming. Every *other* ending it computes: a battle's outcome from its own
// Ended, the series length from Welcome.Battles, the turn cap from
// Welcome.TurnCap.
//
// Four claims, and the addressing is not a detail: the message goes to the seat
// that is **still there** and to nothing else.
func TestALeavingPeerIsAnnouncedToTheOneStillThere(t *testing.T) {
	opened, clients := openMatch(t, config(11, 3))
	const victim = wire.SeatGuest
	answerFor(t, opened, clients, "")

	out, err := opened.Left(victim)
	if err != nil {
		t.Fatalf("leave: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("a departure produced %d messages, want one for the seat still there", len(out))
	}
	if out[0].To != wire.SeatHost {
		t.Errorf("the room told %q the match had ended, want the seat that did not leave", out[0].To)
	}
	closed, isClosed := out[0].Body.(wire.Closed)
	if !isClosed {
		t.Fatalf("a departure produced a %s", out[0].Body.Kind())
	}
	if closed.Reason != wire.ClosureLeft {
		t.Errorf("the room closed with the reason %q, want %q", closed.Reason, wire.ClosureLeft)
	}
	// It is a reason a client can word rather than a value meaning "no reason":
	// a Closed carrying ClosureNone is a room declining to say why.
	if !closed.Reason.Closes() {
		t.Errorf("the reason %q does not report itself as an ending", closed.Reason)
	}
	// And it survives the trip, because the far end is where it is worded.
	raw, err := wire.Encode(out[0].Body)
	if err != nil {
		t.Fatalf("encode the closure: %v", err)
	}
	decoded, err := wire.Decode(raw)
	if err != nil {
		t.Fatalf("decode the closure: %v", err)
	}
	if got, ok := decoded.(*wire.Closed); !ok || got.Reason != wire.ClosureLeft {
		t.Errorf("the closure crossed as %#v", decoded)
	}

	clients.deliver(t, out)
	if got := clients.host.closures; len(got) != 1 || got[0] != wire.ClosureLeft {
		t.Errorf("the host recorded the closures %v", got)
	}
	// The peer that went away is told nothing, and that is a decision rather
	// than an omission: the transport has already decided there is nobody there,
	// so a message addressed to it is a message to nobody.
	if got := clients.guest.closures; len(got) != 0 {
		t.Errorf("the seat that left was told %v", got)
	}
	// The remaining client is holding a battle with no Ended in it, which is
	// exactly the state the message exists to explain — so the state is asserted
	// rather than assumed.
	if clients.host.fight == nil {
		t.Fatal("the host has no battle open")
	}
	if clients.host.fight.Finished() {
		t.Error("the host's own battle reports itself finished; a departure interrupts a battle " +
			"rather than ending one, so there is no Ended for it to have read")
	}
	if clients.host.prompt == nil {
		t.Error("the host is not holding an open prompt, so this run does not measure the case " +
			"the message exists for")
	}
}

// TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas is the backstop against a
// runaway, so a stalemate nothing on the board can resolve does not hold two
// people at a table for ever.
//
// ⚠️ It needs no new battle.Outcome, and the room does **not** stamp one on the
// battle it stopped. The engine did not conclude anything about that battle, and
// a room writing an outcome the engine never produced would be a second reading
// of how a battle ends — the mistake this repository has recorded most often —
// and the eventual log would fail its own --verify. So the outcome stays
// Undecided, BattleResult.Capped says what happened, and the series counts it
// the way it counts every other draw. That is what "the outcome already carries
// the draws" buys: the standing needs to know that neither seat won, and it
// needs no new way for a battle to end to know it.
func TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas(t *testing.T) {
	// A cap far below any real battle, so the cap is certainly what stops it.
	// The shipped 3v3 runs to dozens of decisions.
	configuration := config(11, 1)
	configuration.TurnCap = 6
	opened, clients := openMatch(t, configuration)

	for steps := 0; !opened.Finished(); steps++ {
		if steps > 4*configuration.TurnCap {
			t.Fatalf("the %d-turn cap did not stop the battle in %d turns", configuration.TurnCap, steps)
		}
		answerFor(t, opened, clients, "")
	}

	played := opened.Played()
	if len(played) != 1 {
		t.Fatalf("a bo1 played %d battles", len(played))
	}
	capped := played[0]
	if !capped.Capped {
		t.Fatalf("the battle ended as %q rather than at the cap", capped.Outcome)
	}
	if !capped.Draw() {
		t.Errorf("a capped battle went to %q", capped.Winner)
	}
	if capped.Outcome != battle.Undecided {
		t.Errorf("a capped battle was recorded as %q; the engine concluded nothing about it, so "+
			"the room must not write an outcome on its behalf", capped.Outcome)
	}
	if capped.Turns <= configuration.TurnCap {
		t.Errorf("the battle opened %d turns against a cap of %d", capped.Turns, configuration.TurnCap)
	}
	// The whole series drew, which is the verdict a bo1 of one drawn battle has
	// and needs no invented tie-break to reach.
	result := opened.Result()
	if result.Verdict != room.VerdictDrawn {
		t.Errorf("a bo1 whose only battle drew has the verdict %q, want %q", result.Verdict, room.VerdictDrawn)
	}
	if result.Winner.Valid() {
		t.Errorf("a drawn match names %q as its winner", result.Winner)
	}
	if result.Wins != [2]int{0, 0} {
		t.Errorf("a drawn match records the standing %v", result.Wins)
	}
	// And the enum is where it was.
	if battle.OutcomeCount != theBattlesOutcomes {
		t.Errorf("battle.OutcomeCount is %d, want %d", battle.OutcomeCount, theBattlesOutcomes)
	}
	// The clients followed the capped battle turn for turn, which is the half of
	// this that is not obvious: the room stops asking on a boundary a mirror also
	// stops at, so no digest disagreed on the way there.
	//
	// ⚠️ **And each of them stopped on the same turn, by its own arithmetic.**
	// That is what Welcome.TurnCap buys and it is the whole reason a capped
	// battle needs no message and no Ended: the client is a mirror, so given the
	// cap it counts the same turns off its own battle and reaches the same
	// boundary. Without the cap on the welcome every one of these clients would
	// be holding an open prompt on a battle the room had stopped asking about,
	// with nothing in the protocol ever telling it otherwise.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if client.compared == 0 {
			t.Errorf("%s checked no digests in a capped battle", client.seat)
		}
		if client.welcome.TurnCap != configuration.TurnCap {
			t.Errorf("%s was told the cap is %d, want %d",
				client.seat, client.welcome.TurnCap, configuration.TurnCap)
		}
		if !client.capped {
			t.Errorf("%s has not stopped at the cap after %d turns of a battle the room capped at %d",
				client.seat, client.turns, capped.Turns)
		}
		if client.turns != capped.Turns {
			t.Errorf("%s counted %d turns where the room counted %d; the two peers stop on "+
				"different turns", client.seat, client.turns, capped.Turns)
		}
	}
	t.Logf("capped at %d turns with the outcome %q; %d digests agreed on the way",
		capped.Turns, capped.Outcome, clients.host.compared)
}

// TestEveryVerdictIsNamed is the totality guard this enum shares with every
// other enum in this repository: a walk of the count rather than a range over
// the table of names, because ranging over a table and asking it whether it
// holds what it holds is what let five screens slip into the authoring client
// unmeasured.
//
// ⚠️ **It used to walk two enums and now walks one.** room.Forfeit is gone —
// none · timed_out · left, three values pricing two routes to a forfeit, neither
// of which exists — and with it went the clause that held ForfeitCount against a
// literal 3. What replaced that value is room.VerdictAbandoned, a fourth verdict
// rather than a second enum, which is why the count below is still four and why
// the numeric fallback is still "verdict(4)".
func TestEveryVerdictIsNamed(t *testing.T) {
	for value := 0; value < room.VerdictCount; value++ {
		verdict := room.Verdict(value)
		name := verdict.String()
		// A value with no entry in the names table falls through to the numeric
		// form, which is exactly what an unnamed verdict looks like.
		if name == "" || strings.HasPrefix(name, "verdict(") {
			t.Errorf("verdict %d has no name, it stringifies as %q", value, name)
		}
		if want := value != 0; verdict.Over() != want {
			t.Errorf("verdict %q reports Over as %v, want %v", name, verdict.Over(), want)
		}
	}
	if got := room.Verdict(room.VerdictCount).String(); got != "verdict(4)" {
		t.Errorf("a verdict past the count stringifies as %q", got)
	}
	// Every verdict but the zero one is reached by a test in this package, which
	// is what keeps VerdictCount from counting a branch nothing takes: won and
	// drawn by TestTwoFakeClientsFightAWholeBo3InProcess and
	// TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas, abandoned by
	// TestADepartureAddsNothingToTheBattlesOutcomes. A fifth needs a case of its
	// own before it is counted here.
	if room.VerdictCount != 4 {
		t.Errorf("there are %d verdicts; the three that end a match are each driven by a test, so "+
			"a fourth needs one before it is counted", room.VerdictCount)
	}
}
