package socket

import (
	"context"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// tokenised is deps with a real token source on it, which is what makes a seat
// rejoinable at all.
//
// ⚠️ The fixtures' own `deps` deliberately has none, so every test written
// before rejoining existed still measures the behaviour it was written for: a
// socket closing ends the match at once. That is not an oversight in those tests
// — it is the arrangement working, and this is the one line that opts in.
func tokenised(t *testing.T) room.Deps {
	t.Helper()
	held := deps(t)
	held.Tokens = wire.NewSeatToken
	return held
}

// theRejoinWindow is short enough that a test which waits for it to run out does
// not spend a minute doing so, and long enough that a machine under load does not
// close it while a reconnecting client is still dialling.
const theRejoinWindow = 750 * time.Millisecond

// TestASeatIsHeldWhenItsSocketClosesAndTheMatchDoesNotEnd is the whole point of
// the feature, stated as the thing that used to happen and now does not.
//
// Before this, a socket closing was a match ending: the transport cannot tell a
// wifi hiccup from somebody walking away, so every blip cost a whole series. Here
// the host's socket closes mid-match and the guest is told **nothing**, because
// as far as the room is concerned nobody has gone anywhere.
func TestASeatIsHeldWhenItsSocketClosesAndTheMatchDoesNotEnd(t *testing.T) {
	dependencies := tokenised(t)
	held := listening(t, Timings{RejoinWindow: time.Minute})
	code := held.open(t, config(11, 3, room.DefaultAllowance), dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()
	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's own loop after its socket closed: %v", err)
	}

	// The guest is still playing, and the only way to say so from out here is to
	// wait long enough that a closure would have arrived and then find none.
	time.Sleep(theRejoinWindow)
	if closure, closed := guest.Mirror().Closure(); closed {
		t.Errorf("the guest was told the match closed because %q: a seat that can be come "+
			"back to must not end the match when its socket goes", closure)
	}
	select {
	case done := <-held.endings:
		t.Fatalf("a match ended while a seat was being held: %+v", done.reading.Result)
	default:
	}
	guest.Close()
	_ = guestPlay.wait(t, "the guest")
}

// TestTheWindowRunningOutEndsTheMatchTheWayADepartureAlwaysDid is the other end
// of the same arrangement, and it is what stops a held seat being a room that
// never closes.
//
// ⚠️ **The verdict has to be the same one**, because nothing about the match
// changed: a client that did not come back is a client that left, and a second
// verdict for "left slowly" would be a distinction no player asked for and every
// reader of a result would have to learn.
func TestTheWindowRunningOutEndsTheMatchTheWayADepartureAlwaysDid(t *testing.T) {
	dependencies := tokenised(t)
	held := listening(t, Timings{RejoinWindow: theRejoinWindow})
	code := held.open(t, config(11, 3, room.DefaultAllowance), dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()
	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's own loop after its socket closed: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's loop after the window ran out: %v", err)
	}

	closure, closed := guest.Mirror().Closure()
	if !closed {
		t.Fatalf("the window ran out and the guest was never told, so it sat on an open prompt")
	}
	if closure != wire.ClosureLeft {
		t.Errorf("the guest was told the match closed because %q, want %q", closure, wire.ClosureLeft)
	}
	done := held.finished(t)
	if done.reading.Result.Verdict != room.VerdictAbandoned {
		t.Errorf("the window running out ended the match %q, want %q",
			done.reading.Result.Verdict, room.VerdictAbandoned)
	}
	if done.reading.Result.Departed != wire.SeatHost {
		t.Errorf("the match records %q as having gone away, want the host", done.reading.Result.Departed)
	}
}

// TestASeatWithNoTokenIsNotHeldAtAll is the guard that keeps a held seat from
// costing the other player a minute for nothing.
//
// A room whose caller supplied no token source seats nobody who can prove they
// are that seat, so there is nobody the window could be waiting for. The whole
// existing suite rides on this — every test written before rejoining uses the
// plain `deps` — and this is the one that says so on purpose rather than by
// passing.
func TestASeatWithNoTokenIsNotHeldAtAll(t *testing.T) {
	dependencies := deps(t)
	if dependencies.Tokens != nil {
		t.Fatal("the fixtures now issue tokens, so this test measures nothing")
	}
	held := listening(t, Timings{RejoinWindow: time.Minute})
	code := held.open(t, config(11, 3, room.DefaultAllowance), dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()
	_ = hostPlay.wait(t, "the host")
	// A minute-long window is set above and the match must end anyway, which is
	// the whole assertion: the window is not being waited on.
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's loop after the host left: %v", err)
	}
	done := held.finished(t)
	if done.reading.Result.Verdict != room.VerdictAbandoned {
		t.Errorf("a seat with no token was held instead of reported: %q",
			done.reading.Result.Verdict)
	}
}

// TestAClientThatComesBackTakesItsSeatAndTheMatchCarriesOn is the round trip,
// and it is the one test here that exercises the whole arrangement at once: the
// room issues a token, the transport holds the seat, and a second connection
// showing that token is seated as the same player rather than refused as a third.
//
// ⚠️ **The control is the third client**, which is what makes the rejoin mean
// something. A room holding a seat is still a full room to anybody else, so a
// hello with no token has to be turned away at the same moment the one with a
// token is let in — otherwise this would be measuring an empty seat.
func TestAClientThatComesBackTakesItsSeatAndTheMatchCarriesOn(t *testing.T) {
	dependencies := tokenised(t)
	held := listening(t, Timings{RejoinWindow: time.Minute})
	code := held.open(t, config(11, 3, room.DefaultAllowance), dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	token := host.Token()
	if !token.Set() {
		t.Fatal("the host was seated with no token, so there is nothing to come back with")
	}
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()
	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's own loop after its socket closed: %v", err)
	}

	// The control: still full to a stranger.
	_, err := Dial(ctx, code, hello(t, theHostSquad(t, dependencies.Characters), "Stranger", ""),
		dependencies.Books, ClientOptions{Timings: held.timings})
	if refusal := refusalOf(t, err); refusal.Code != wire.CodeRoomFull {
		t.Fatalf("a stranger was answered %q while a seat was held, want %q",
			refusal.Code, wire.CodeRoomFull)
	}

	// And the rejoin.
	returning := hello(t, theHostSquad(t, dependencies.Characters), "Host", "")
	returning.Token = token
	back := held.dial(t, code, returning, dependencies.Books)
	if back.Seat() != wire.SeatHost {
		t.Errorf("the returning client took the seat %q, want %q", back.Seat(), wire.SeatHost)
	}
	if back.Token() != token {
		t.Errorf("the returning client was welcomed with a different token, so a second " +
			"reconnection would show one the room has forgotten")
	}
	// The guest was never told anything, because nothing happened to the room.
	if closure, closed := guest.Mirror().Closure(); closed {
		t.Errorf("the guest was told the match closed because %q", closure)
	}
	// ⚠️ **The window has to be STOPPED and not merely outlived**, which a
	// minute-long one cannot show: a test that ended here would pass with the
	// release deleted, because the timer would simply never fire inside it —
	// measured. So the same round trip is taken again over a short window and the
	// clock is allowed to run past it.
	back.Close()
	guest.Close()
	_ = guestPlay.wait(t, "the guest")
}

// TestARejoinStopsTheWindowRatherThanOutlivingIt is the half the test above
// cannot hold.
//
// A window that is never stopped fires anyway, and telling the room a seat left
// would end a match the returning client is sitting in. With a window shorter
// than the wait below, a release that did not happen is a match that ends.
func TestARejoinStopsTheWindowRatherThanOutlivingIt(t *testing.T) {
	dependencies := tokenised(t)
	held := listening(t, Timings{RejoinWindow: theRejoinWindow})
	code := held.open(t, config(11, 3, room.DefaultAllowance), dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	token := host.Token()
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()
	_ = hostPlay.wait(t, "the host")

	returning := hello(t, theHostSquad(t, dependencies.Characters), "Host", "")
	returning.Token = token
	back := held.dial(t, code, returning, dependencies.Books)
	if back.Seat() != wire.SeatHost {
		t.Fatalf("the returning client took the seat %q", back.Seat())
	}
	// Past the window the departure would have fired in.
	time.Sleep(2 * theRejoinWindow)
	if closure, closed := guest.Mirror().Closure(); closed {
		t.Errorf("the match closed because %q after a client came back: the window was "+
			"outlived rather than stopped", closure)
	}
	select {
	case done := <-held.endings:
		t.Errorf("a match ended after its seat was taken back: %+v", done.reading.Result)
	default:
	}
	back.Close()
	guest.Close()
	_ = guestPlay.wait(t, "the guest")
}
