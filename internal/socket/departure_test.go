package socket

import (
	"context"
	"errors"
	"testing"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestADepartureClosesTheRoomAndTellsThePeerStillThere is the **one ending a
// mirror cannot reach on its own**, and it is therefore the one that needs a
// message.
//
// Every other ending a client computes: each battle's outcome from its own Ended
// event, the series from wire.Welcome.Battles, a capped battle from
// wire.Welcome.TurnCap by the same arithmetic the room uses. A departure is
// different — there is no Ended for the battle in progress, because the engine
// concluded nothing about it, and no further wire.Start — so a peer handed
// nothing would sit on its own open prompt waiting for a turn that is never
// coming. Hence wire.Closed{ClosureLeft}, addressed to the seat **still there**
// and to nothing else.
//
// ⚠️ This is the only test in the package that can measure that arriving,
// because it is the only thing in the protocol a mirror cannot produce for
// itself.
//
// Three claims:
//
//  1. the peer still there is told, **over the socket**, with ClosureLeft;
//  2. the transport's own reading of the ending is room.VerdictAbandoned naming
//     the seat that went — which is not a win, not a draw and not a forfeit;
//  3. both loops end, so a departure is not a hang.
//
// The host leaves **mid-battle**, on the fifth decision, which is where a
// departure is interesting: a bo3 of the shipped 3v3 takes thirty-odd decisions
// a battle, so nothing has been concluded yet.
func TestADepartureClosesTheRoomAndTellsThePeerStillThere(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	ctx := context.Background()
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	hostChoose, fifth := stepped(rating(host), 5)
	hostPlay := play(ctx, host, hostChoose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	// ⚠️ The signal comes out of the host's own chooser rather than from a test
	// reading its Mirror, which belongs to that client's loop: a reading from
	// here would be a race, and the detector would be right about it.
	reachedOr(t, fifth, "the host's fifth decision")
	host.Close()

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's own loop after it left: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's loop after the host left: %v", err)
	}

	// Claim 1: the closure arrived, over the socket, and it is the one there is.
	closure, closed := guest.Mirror().Closure()
	if !closed {
		t.Fatalf("the guest was never told the match closed, so it would have sat on its own open prompt")
	}
	if closure != wire.ClosureLeft {
		t.Errorf("the guest was told the match closed because %q, want %q", closure, wire.ClosureLeft)
	}
	// And the seat that left is told nothing, because the transport has already
	// decided there is nobody there.
	if _, told := host.Mirror().Closure(); told {
		t.Error("the seat that left was sent a closure, which is a message to nobody")
	}

	// Claim 2: the transport's own reading, which rides on the answer to the
	// input that ended the match — a room retires its entry the moment its match
	// ends, so there is nowhere else to read it from.
	done := held.finished(t)
	if done.code != code {
		t.Errorf("a match finished in room %s and this test opened %s", done.code, code)
	}
	if done.reading.Result.Verdict != room.VerdictAbandoned {
		t.Errorf("a departure ended the match %q, want %q",
			done.reading.Result.Verdict, room.VerdictAbandoned)
	}
	if done.reading.Result.Departed != wire.SeatHost {
		t.Errorf("the match records %q as having gone away, want the host",
			done.reading.Result.Departed)
	}
	// ⚠️ Nobody is charged with anything, which is the stated cost: a losing
	// player can leave for free and on a LAN the enforcement is social.
	if done.reading.Result.Winner.Valid() {
		t.Errorf("an abandoned match names %q as its winner", done.reading.Result.Winner)
	}
	// The battle in progress is not among the ones played — the engine concluded
	// nothing about it — so a departure mid-battle records no battle at all.
	if played := len(done.reading.Played); played != 0 {
		t.Errorf("a departure in the first battle recorded %d battles", played)
	}
	// And the guest's own engine agrees it settled nothing.
	if settled := guest.Mirror().Fought(); len(settled) != 0 {
		t.Errorf("the guest's own engine settled %d battles before the host left", len(settled))
	}

	// Claim 3: nothing is left holding anything.
	held.emptied(t)
	t.Logf("the host left after 5 decisions; the guest was told %q and the match ended %q",
		closure, done.reading.Result.Verdict)
}

// TestAJoinerRefusedForAFullRoomIsToldSo is the third seat, which is what a
// spectator looks like until spectators exist.
//
// It is here rather than beside the other refusals because it needs a room with
// **both seats taken over real connections**, which is a transport fact: the
// gate checks the seat before the squad, so a stranger with a legal squad and a
// full room is told the room is full rather than anything about its squad.
func TestAJoinerRefusedForAFullRoomIsToldSo(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, config(11, 1, room.DefaultAllowance), dependencies)

	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)

	_, err := Dial(context.Background(), code,
		hello(t, theHostSquad(t, dependencies.Characters), "Third", ""),
		dependencies.Books, ClientOptions{Timings: held.timings})
	refusal := refusalOf(t, err)
	if refusal.Code != wire.CodeRoomFull {
		t.Errorf("a third client was refused with %q, want %q", refusal.Code, wire.CodeRoomFull)
	}
}

// refusalOf reads the code off a refusal, which is all a client is told and all
// a screen needs — nothing in this package words one.
func refusalOf(t *testing.T, err error) *Refusal {
	t.Helper()
	var refusal *Refusal
	if err == nil {
		t.Fatal("a client that should have been refused got in")
	}
	if !errors.As(err, &refusal) {
		t.Fatalf("a client was turned away with %v, which is not a *Refusal", err)
	}
	return refusal
}
