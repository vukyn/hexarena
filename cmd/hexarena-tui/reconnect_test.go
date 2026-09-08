package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	"sync"

	tea "charm.land/bubbletea/v2"

	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestAReconnectionIsRefusedWhenThereIsNothingToComeBackTo is the four ways this
// client decides not to try, and every one of them is a case where dialling again
// would be worse than stopping.
//
// ⚠️ **It asks the MIRROR rather than the error, and that is the decision.** A
// dropped socket, a cancelled context and a match played to its end all come back
// from Play as an error, so telling them apart by inspecting one would be this
// client deciding what a network failure looks like — a judgement it has no
// business making and no way to keep right.
func TestAReconnectionIsRefusedWhenThereIsNothingToComeBackTo(t *testing.T) {
	for _, one := range []struct {
		name  string
		setup func(*session) context.Context
	}{
		{
			name: "nothing was ever dialled",
			setup: func(s *session) context.Context {
				ctx := s.open()
				s.remember(nil)
				return ctx
			},
		},
		{
			name: "the player quit",
			setup: func(s *session) context.Context {
				ctx := s.open()
				s.remember(&redial{code: "AAAAAAAAAAAA"})
				s.leave()
				return ctx
			},
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			s := &session{}
			ctx := one.setup(s)
			// ⚠️ **The sender is what makes this discriminate.** Returning nil is
			// what a refusal and a ninety-second give-up both do, so a test
			// asserting only that would pass on a version that tried the whole
			// budget first — measured: with the cancellation check disabled, the
			// "player quit" arm still ended in nil, ninety seconds later. What
			// separates "did not try" from "tried and failed" is whether the
			// screen was ever told a reconnection had started.
			watching := &counting{}
			s.attach(watching)
			if back := s.reconnect(ctx, standing{token: "a-token"}); back != nil {
				t.Error("this client dialled again when there was nothing to come back to")
			}
			if watching.count() != 0 {
				t.Errorf("the screen was told a reconnection started %d time(s) when there "+
					"was nothing to come back to", watching.count())
			}
			if s.Reconnecting() {
				t.Error("the screen would say this client is reconnecting when it is not")
			}
		})
	}
}

// TestANewMatchForgetsTheLastOnesRedial.
//
// ⚠️ A redial kept across a match would be a reconnection to **somebody else's
// room**: the token is for a seat in the room this client has left, and the code
// names a match it is no longer in. open clears it for the reason it clears the
// clock and the pool — a player who leaves a room and joins another gets every
// guarantee back.
func TestANewMatchForgetsTheLastOnesRedial(t *testing.T) {
	s := &session{}
	s.open()
	s.remember(&redial{code: "AAAAAAAAAAAA"})
	s.setReconnecting(true)
	if s.redialling() == nil {
		t.Fatal("the redial was not kept in the first place")
	}
	s.open()
	if s.redialling() != nil {
		t.Error("a new match kept the last one's redial, so a reconnection would rejoin " +
			"a room this client has left")
	}
	if s.Reconnecting() {
		t.Error("a new match started out reconnecting")
	}
}

// TestARefusalEndsAReconnectionAndANetworkErrorDoesNot is the pair that decides
// how long this client keeps asking.
//
// A refusal is the **room's** answer: the seat is not there to take, because the
// window ran out or the match is over and the code names nothing. Retrying
// against that is asking a question that has been answered. A network error is
// the network's answer and says nothing about the seat, which is exactly the case
// the whole feature exists for.
//
// ⚠️ It is asserted on the classification rather than by running the loop,
// because the loop's other exit is a ninety-second budget and a test that waited
// for it would be measuring the constant.
func TestARefusalEndsAReconnectionAndANetworkErrorDoesNot(t *testing.T) {
	var refusal *socket.Refusal
	if !errors.As(error(&socket.Refusal{Code: wire.CodeRoomFull}), &refusal) {
		t.Fatal("a *socket.Refusal is not recognised as one, so the loop's exit is unreachable")
	}
	if refusal.Code != wire.CodeRoomFull {
		t.Errorf("the refusal carries %q", refusal.Code)
	}
	if errors.As(errors.New("dial tcp: connection refused"), &refusal) {
		t.Error("a network error is read as a refusal, so a client would stop trying the " +
			"moment the wifi went — which is the case this feature is for")
	}
}

// TestTheReconnectingLineReplacesTheWaitingOne.
//
// ⚠️ **A board that simply stopped moving is indistinguishable from the other
// player thinking**, and the two want opposite things from a reader: one is
// nothing to do about, and the other is somebody staring at a frozen screen
// deciding whether to quit — which, inside the window the seat is held for, is
// the one thing that actually loses the match.
//
// It replaces rather than joins the waiting line, because this screen has no row
// to give: → PlayScreen.heading, where the same budget put the countdown on the
// title row.
func TestTheReconnectingLineReplacesTheWaitingOne(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		live := aLiveBattleReconnecting(t, m)
		drawn := drawnBody(live)
		if !strings.Contains(drawn, live.text(i18n.PlayLiveReconnecting)) {
			t.Errorf("in %s a reconnecting battle says nothing about the connection:\n%s", lang, drawn)
		}
		if strings.Contains(drawn, live.text(i18n.PlayLiveWaiting)) {
			t.Errorf("in %s a reconnecting battle also says it is waiting on the other "+
				"player:\n%s", lang, drawn)
		}
	}
}

// counting is a sender that records how many redraws it was asked for, which is
// how a test observes something the session did on a goroutine of its own.
type counting struct {
	mu   sync.Mutex
	sent int
}

func (c *counting) Send(tea.Msg) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent++
}

func (c *counting) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sent
}

// TestADialRemembersWhatASecondOneWouldNeed.
//
// ⚠️ **Kept before the dial rather than after it**, which is what this asserts by
// asking before the command has been run: the window in which a first dial can
// fail is exactly where a client would otherwise be left with nothing to try
// again with.
func TestADialRemembersWhatASecondOneWouldNeed(t *testing.T) {
	held, _ := openARoom(t, 1)
	s := &session{}
	joining := wire.Hello{Version: held.deps.Version, Squad: held.squads[0], Name: "A"}
	_ = s.dial(held.code, joining, held.books, held.deps.Characters)
	kept := s.redialling()
	if kept == nil {
		t.Fatal("a dial kept nothing, so a dropped socket has nothing to dial again with")
	}
	if kept.code != held.code {
		t.Errorf("the redial names the room %q and the dial went to %q", kept.code, held.code)
	}
	if kept.hello.Name != joining.Name {
		t.Errorf("the redial carries the hello %q and the dial sent %q",
			kept.hello.Name, joining.Name)
	}
}

// TestAReconnectionTellsTheScreenItStartedAndThatItStopped.
//
// ⚠️ **The screen being told is the whole of what this feature adds for a
// player**, and it is the half nothing else here can see: the reconnection
// happens on the Play goroutine, so the only trace of it reaching a reader is the
// redraws it asks for. A version that tried and said nothing would look, from
// every other test, exactly like this one.
//
// The room is filled first so the reconnection is refused on its first attempt
// rather than spending the ninety-second budget: what is being measured is that
// the attempt was announced, not how long it lasts.
func TestAReconnectionTellsTheScreenItStartedAndThatItStopped(t *testing.T) {
	held, _ := openARoom(t, 1)
	ctx := context.Background()
	for index, name := range []string{"A", "B"} {
		joined, err := socket.Dial(ctx, held.code,
			wire.Hello{Version: held.deps.Version, Squad: held.squads[index], Name: name},
			held.books, socket.ClientOptions{})
		if err != nil {
			t.Fatalf("fill the room with %s: %v", name, err)
		}
		defer joined.Close()
	}

	watching := &counting{}
	s := &session{}
	matchCtx := s.open()
	s.attach(watching)
	s.remember(&redial{
		code:  held.code,
		hello: wire.Hello{Version: held.deps.Version, Squad: held.squads[0], Name: "A"},
		books: held.books,
	})
	before := watching.count()
	if back := s.reconnect(matchCtx, standing{token: "a-token"}); back != nil {
		t.Fatal("a full room gave a seat to a token it never issued")
	}
	if watching.count() <= before {
		t.Error("a reconnection asked for no redraw, so a player watching a frozen board " +
			"is never told the connection went")
	}
	if s.Reconnecting() {
		t.Error("the reconnection gave up and the screen would still say it was trying")
	}
}

// TestALiveDrawingCarriesTheReconnection is the one wire between the session's
// loop and the screen, and it is asserted on its own because nothing else can see
// it: every other reading a live drawing carries comes off the mirror, and this
// one does not.
//
// ⚠️ **A mirror knows nothing about a socket that closed under it.** It is a
// battle, and the battle did not change — the room held the seat and the board
// stayed exactly where it was. So this is the single field on a live drawing that
// comes from the client's own loop, and a mutation blanking it leaves every
// mirror-driven test green.
func TestALiveDrawingCarriesTheReconnection(t *testing.T) {
	if !liveOf(socket.Sight{}, draw.PlayClock{}, true).Reconnecting {
		t.Error("a reconnecting session draws a battle that says nothing about it")
	}
	if liveOf(socket.Sight{}, draw.PlayClock{}, false).Reconnecting {
		t.Error("a connected session draws a battle claiming to be reconnecting")
	}
}
