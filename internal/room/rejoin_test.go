package room_test

import (
	"fmt"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// countedTokens is a token source a test can predict, which is the whole reason
// room.Deps takes one instead of drawing its own randomness.
//
// ⚠️ The real source is wire.NewSeatToken and is unguessable; this is not, and
// must never be what a binary hands in. What it buys here is that a test can say
// "the host's token" rather than fishing one out of a welcome — and, in the
// failure arm below, hand over a token that was never issued.
func countedTokens() func() (wire.SeatToken, error) {
	issued := 0
	return func() (wire.SeatToken, error) {
		issued++
		return wire.SeatToken(fmt.Sprintf("token-%d", issued)), nil
	}
}

// tokenised is deps with a predictable token source on it.
func tokenised(t *testing.T) room.Deps {
	t.Helper()
	held := deps(t)
	held.Tokens = countedTokens()
	return held
}

// roomWithTokens is newRoom over a room that can issue seat tokens.
func roomWithTokens(t *testing.T, cfg room.Config) *room.Room {
	t.Helper()
	opened, err := room.New(cfg, tokenised(t))
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	return opened
}

// welcomeIn is the welcome out of a batch of outbound messages, and it fails
// rather than returning a zero one: every assertion below is about a field on it,
// so a missing welcome has to stop the test rather than compare zeroes.
func welcomeIn(t *testing.T, out []room.Outbound) wire.Welcome {
	t.Helper()
	for _, held := range out {
		if welcome, ok := held.Body.(wire.Welcome); ok {
			return welcome
		}
	}
	t.Fatalf("no welcome among %d message(s)", len(out))
	return wire.Welcome{}
}

// seatedIn joins a room and hands back the admission and the welcome together,
// which is what every test here starts from.
func seatedIn(t *testing.T, playing *room.Room, joining wire.Hello) (room.Admission, wire.Welcome) {
	t.Helper()
	admitted, out, err := playing.Join(joining)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if !admitted.Seat.Valid() {
		t.Fatalf("the join took no seat: %+v", admitted)
	}
	return admitted, welcomeIn(t, out)
}

// TestASeatIsIssuedATokenAndTheWelcomeCarriesIt.
//
// The welcome is the only place a token ever leaves the room, and it is
// addressed to the one connection that just took the seat. A token that stayed
// inside would be a rejoin nothing could perform.
func TestASeatIsIssuedATokenAndTheWelcomeCarriesIt(t *testing.T) {
	playing := roomWithTokens(t, config(7, 1))
	admitted, welcome := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
	if !welcome.Token.Set() {
		t.Error("a seated player was welcomed with no token, so it can never come back")
	}
	if !admitted.Rejoinable {
		t.Error("the admission does not say the seat can be come back to, so a transport " +
			"would hold nothing open for it")
	}
	if admitted.Rejoined {
		t.Error("a first join reports itself as a rejoin")
	}
}

// TestARoomWithNoTokenSourceSeatsExactlyAsBefore is the additive half, and it is
// what lets every existing caller keep working.
//
// room.Deps.Tokens is nil in every fixture that has not asked for one, so a room
// built by a caller that has never heard of rejoining issues no token, says the
// seat cannot be come back to, and behaves in every other respect as it did.
func TestARoomWithNoTokenSourceSeatsExactlyAsBefore(t *testing.T) {
	playing := newRoom(t, config(7, 1))
	admitted, welcome := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
	if welcome.Token.Set() {
		t.Error("a room with no token source issued one")
	}
	if admitted.Rejoinable {
		t.Error("a seat with no token says it can be come back to, and nothing could")
	}
}

// TestAHelloShowingASeatsTokenTakesThatSeatBack is the feature, and the room
// being full is the half that makes it one.
//
// ⚠️ A rejoining client's own seat is exactly what makes the room full, so this
// asserts both arms against the same board: the same hello without the token is
// refused as full, and with it takes the seat back.
func TestAHelloShowingASeatsTokenTakesThatSeatBack(t *testing.T) {
	playing := roomWithTokens(t, config(7, 1))
	_, first := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
	seatedIn(t, playing, hello(t, squadOf(t, characters(t), "away", "pokemon.squirtle", "pokemon.pichu", "pokemon.dratini"), "B"))

	// The control: the room really is full, so the rejoin below is not being
	// waved through by there happening to be space.
	stranger := hello(t, squadOf(t, characters(t), "third", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "C")
	admitted, out, err := playing.Join(stranger)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if admitted.Seat.Valid() {
		t.Fatalf("a third player took the seat %s in a full room", admitted.Seat)
	}
	if refused := onlyCode(t, out); refused != wire.CodeRoomFull {
		t.Fatalf("a third player was refused with %s, want %s", refused, wire.CodeRoomFull)
	}

	back := stranger
	back.Token = first.Token
	returned, welcome := seatedIn(t, playing, back)
	if returned.Seat != wire.SeatHost {
		t.Errorf("the token for %s took the seat %s", wire.SeatHost, returned.Seat)
	}
	if !returned.Rejoined {
		t.Error("taking a seat back does not report itself as a rejoin, so a transport " +
			"would treat it as a first join and start the match again")
	}
	if welcome.Token != first.Token {
		t.Errorf("the rejoin was welcomed with %v and the seat's token is %v: a client that "+
			"reconnected twice would be holding a token for a seat under a name the room "+
			"has forgotten", welcome.Token, first.Token)
	}
}

// TestARejoinDoesNotOpenTheMatchASecondTime.
//
// The second seat being taken is what starts the first battle, and a rejoin
// takes a seat that is already taken — so the gate must not run that branch
// again. It would deal a second opening battle into a room already playing one,
// and both peers would be handed a wire.Start for a board that is not the one in
// front of them.
//
// ⚠️ Asserted on what the gate SENDS, because that is where a second opening
// would appear: a rejoin answers with the welcome and nothing else.
func TestARejoinDoesNotOpenTheMatchASecondTime(t *testing.T) {
	playing := roomWithTokens(t, config(7, 1))
	_, first := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
	_, _, err := playing.Join(hello(t, squadOf(t, characters(t), "away", "pokemon.squirtle", "pokemon.pichu", "pokemon.dratini"), "B"))
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	back := hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A")
	back.Token = first.Token
	_, out, err := playing.Join(back)
	if err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if len(out) != 1 {
		kinds := make([]string, 0, len(out))
		for _, held := range out {
			kinds = append(kinds, held.Body.Kind().String())
		}
		t.Fatalf("a rejoin answered with %v, want the welcome and nothing else", kinds)
	}
	if _, ok := out[0].Body.(wire.Welcome); !ok {
		t.Errorf("a rejoin answered with a %s", out[0].Body.Kind())
	}
}

// TestAnEmptyTokenMatchesNoSeat is the trap this feature would otherwise walk
// into, and it is the case that is common rather than the case that is
// interesting.
//
// Every ordinary hello carries no token, and a room built with no Deps.Tokens
// holds none on either seat. A bare equality would match the first of those
// against the second and hand a stranger somebody's seat on the first join after
// it. → room.seatFor, where the Set check is.
func TestAnEmptyTokenMatchesNoSeat(t *testing.T) {
	for _, one := range []struct {
		name string
		open func(*testing.T, room.Config) *room.Room
	}{
		{"a room that issues tokens", roomWithTokens},
		{"a room that does not", newRoom},
	} {
		t.Run(one.name, func(t *testing.T) {
			playing := one.open(t, config(7, 1))
			first, _ := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
			second, _ := seatedIn(t, playing, hello(t, squadOf(t, characters(t), "away", "pokemon.squirtle", "pokemon.pichu", "pokemon.dratini"), "B"))
			if first.Seat == second.Seat {
				t.Fatalf("two joins with no token took the same seat %s", first.Seat)
			}
			if second.Rejoined {
				t.Error("an ordinary join with no token reported itself as a rejoin")
			}
		})
	}
}

// TestATokenNobodyIssuedTakesNoSeat is the other half of the gate, and it is the
// arm a constant-time compare exists for.
func TestATokenNobodyIssuedTakesNoSeat(t *testing.T) {
	playing := roomWithTokens(t, config(7, 1))
	seatedIn(t, playing, hello(t, squadOf(t, characters(t), "home", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "A"))
	seatedIn(t, playing, hello(t, squadOf(t, characters(t), "away", "pokemon.squirtle", "pokemon.pichu", "pokemon.dratini"), "B"))

	forged := hello(t, squadOf(t, characters(t), "third", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "C")
	forged.Token = "token-999"
	admitted, out, err := playing.Join(forged)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if admitted.Seat.Valid() || admitted.Rejoined {
		t.Fatalf("a token nobody issued took the seat %s (rejoined=%t)", admitted.Seat, admitted.Rejoined)
	}
	if refused := onlyCode(t, out); refused != wire.CodeRoomFull {
		t.Errorf("a forged token was refused with %s, want the ordinary %s", refused, wire.CodeRoomFull)
	}
}

// characters is the shipped cast book, which squadOf needs.
func characters(t *testing.T) *cast.Book {
	t.Helper()
	return deps(t).Characters
}
