package socket

import (
	"context"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestAChessClockChargesTheTimeASeatWasWaitedOn is the arithmetic the whole
// feature is, and the thing it must not do is charge the allowance.
//
// ⚠️ **A player who answers in five seconds of a ninety-second allowance has
// spent five.** Charging the armed length instead would spend a whole match's
// budget in a handful of prompts, which is the obvious implementation and is the
// one this asserts against: the two seats between them are charged roughly the
// wall time the match took, not the allowance times the turns.
func TestAChessClockChargesTheTimeASeatWasWaitedOn(t *testing.T) {
	dependencies := deps(t)
	// A generous budget, so nothing runs out and what is measured is the
	// accounting rather than the enforcement.
	clocked := config(11, 1, room.DefaultAllowance)
	clocked.Budget = 3600
	held := listening(t, Timings{})
	code := held.open(t, clocked, dependencies)

	ctx := context.Background()
	started := time.Now()
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
	elapsed := time.Since(started)
	spent := done.spent
	total := spent[0] + spent[1]
	t.Logf("%d turns over %s: host %s, guest %s", done.reading.Played[0].Turns,
		elapsed.Round(time.Millisecond), spent[0].Round(time.Millisecond),
		spent[1].Round(time.Millisecond))

	if total > elapsed {
		t.Errorf("the two seats were charged %s between them and the match took %s: a clock "+
			"cannot spend time that did not pass", total, elapsed)
	}
	// ⚠️ The vacuity guard, and it is the one that matters here: two clients
	// rating their own turns answer in microseconds, so a run where nobody was
	// charged anything at all would satisfy every line above. What says the clock
	// ran is that both seats were charged something.
	for index, one := range spent {
		if one <= 0 {
			t.Errorf("seat %d was charged %s over a whole match, so the clock never ran",
				index, one)
		}
	}
	// And the number the obvious mistake would produce, named so a reader can see
	// what is being ruled out: the allowance times the turns taken.
	if wrong := Allowance(clocked.Allowance) * time.Duration(done.reading.Played[0].Turns); total >= wrong {
		t.Errorf("the seats were charged %s and the allowance times the turns is %s: the "+
			"clock is charging the time it armed rather than the time it waited", total, wrong)
	}
}

// TestASpentBudgetTimesOutEveryTurnRatherThanForfeiting is what running out
// means, and it is the same decision the per-turn allowance already took.
//
// ⚠️ **Nobody is declared to have lost on time.** A seat with nothing left times
// out the moment each prompt opens, so its units pass and the board kills them —
// which is where this game decides things. A forfeit would be a second way to
// lose a match, decided by a clock rather than by a battle, and the room has no
// verdict for it.
//
// The budget is one second and the allowance is the shortest the room accepts, so
// the clock is gone within a turn or two and the rest of the match is a side that
// cannot act.
func TestASpentBudgetTimesOutEveryTurnRatherThanForfeiting(t *testing.T) {
	dependencies := deps(t)
	starved := config(11, 1, 1)
	starved.Budget = 1
	held := listening(t, Timings{})
	code := held.open(t, starved, dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	// The host thinks for longer than its whole clock on every turn, so it spends
	// the budget and then has none.
	hostPlay := play(ctx, host, thinkingFirst(rating(host), 2*time.Second))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	result := done.reading.Result
	if !result.Verdict.Over() {
		t.Fatalf("a match against a spent clock ended %q", result.Verdict)
	}
	// ⚠️ **The verdict is an ordinary one.** Won or drawn, decided on the board —
	// never abandoned, which is what a forfeit would have to look like in a room
	// that has no verdict for losing on time.
	if result.Verdict == room.VerdictAbandoned {
		t.Errorf("a spent clock abandoned the match, which is a forfeit by another name: %+v",
			result)
	}
	t.Logf("verdict %q, %d–%d, %d turns", result.Verdict,
		result.Wins[0], result.Wins[1], done.reading.Played[0].Turns)
}

// TestARoomWithNoBudgetChargesNothing is the default, and it is what keeps every
// room that existed before this feature exactly as it was.
func TestARoomWithNoBudgetChargesNothing(t *testing.T) {
	plain := config(11, 1, room.DefaultAllowance)
	if plain.Budget != 0 {
		t.Fatalf("the fixture room already carries a budget of %d", plain.Budget)
	}
	if err := plain.Validate(); err != nil {
		t.Errorf("a room with no budget is refused: %v", err)
	}
	// A budget shorter than one turn's allowance is the configuration where the
	// first prompt spends the whole clock, and it is refused by name.
	tight := plain
	tight.Budget = plain.Allowance - 1
	if err := tight.Validate(); err == nil {
		t.Error("a budget shorter than the allowance was accepted, so the first prompt " +
			"would spend the whole clock")
	}
	negative := plain
	negative.Budget = -1
	if err := negative.Validate(); err == nil {
		t.Error("a negative budget was accepted")
	}
}

// TestTheWelcomeCarriesTheBudget, because a number the room enforces and never
// tells anybody about is a match lost to something invisible.
func TestTheWelcomeCarriesTheBudget(t *testing.T) {
	dependencies := deps(t)
	clocked := config(11, 1, room.DefaultAllowance)
	clocked.Budget = 600
	held := listening(t, Timings{})
	code := held.open(t, clocked, dependencies)

	client := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
		dependencies.Books)
	welcome, seated := client.Mirror().Welcome()
	if !seated {
		t.Fatal("the client was not seated")
	}
	if welcome.Budget != clocked.Budget {
		t.Errorf("the welcome carries a budget of %d and the room runs on %d",
			welcome.Budget, clocked.Budget)
	}
	if welcome.Allowance != clocked.Allowance {
		t.Errorf("the welcome's allowance moved: %d against %d",
			welcome.Allowance, clocked.Allowance)
	}
	_ = wire.SeatHost
}

// TestASeatIsNeverChargedMoreThanItsBudget is what the clamp buys, and it is the
// promise a chess clock makes: the clock runs out, it does not run over.
//
// ⚠️ **A timer armed for the allowance alone gives this away and nothing else
// notices.** Every other test here passes with the clamp deleted — measured —
// because they either have a budget generous enough never to bite, or an
// allowance already shorter than the budget. What separates the two is a player
// who keeps thinking: with the clamp, each prompt is armed for whatever is left
// and the total charged stops at the budget; without it, each prompt is armed for
// the whole allowance again and the seat spends the allowance times the turns.
//
// The margin is one allowance, because the charge is wall time and the last
// stretch is cut off by the timer rather than by the arithmetic.
func TestASeatIsNeverChargedMoreThanItsBudget(t *testing.T) {
	dependencies := deps(t)
	// The allowance is the shortest the room accepts and the budget is a couple
	// of them, so a player who never answers spends it inside a few prompts and
	// the test is over in seconds.
	starved := config(11, 1, 2)
	starved.Budget = 3
	held := listening(t, Timings{})
	code := held.open(t, starved, dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	// Longer than the whole budget, on every turn: this seat never answers in
	// time and its clock is the only thing that stops it.
	hostPlay := play(ctx, host, thinkingFirst(rating(host), 10*time.Second))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	budget, allowance := Allowance(starved.Budget), Allowance(starved.Allowance)
	ceiling := budget + allowance
	t.Logf("%d turns: host charged %s against a budget of %s", done.reading.Played[0].Turns,
		done.spent[0].Round(time.Millisecond), budget)
	if done.spent[0] > ceiling {
		t.Errorf("the host was charged %s against a budget of %s: a clock armed for the whole "+
			"allowance every prompt spends the allowance times the turns, which is what the "+
			"budget is meant to bound", done.spent[0], budget)
	}
}
