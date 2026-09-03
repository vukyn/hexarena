package socket

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/coder/websocket"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestAnUnknownCodeIsRefusedOverARealConnection is wire.CodeRoomUnknown reaching
// a peer, which is the code this transport exists to be able to send: the room's
// own gate documents it as *the registry's* refusal and says no room ever sends
// one, and until there was a transport nothing carried it to anybody.
//
// Four codes, and the fourth is the one this package adds:
//
//   - a code that decodes perfectly and names a room this process is not
//     running, which is what a **restarted host** looks like;
//   - a string that is not a code at all — refused by the same answer, because
//     it is handed to the registry as it stands and is the key of no room. → the
//     note on roomOf, for why this package produces no refusal of its own;
//   - the same, of the right length and out of the right alphabet, so the
//     refusal is not resting on a length check;
//   - ⚠️ the **real code in lower case**, which must be **accepted**. A code's
//     alphabet is upper-case only so the fold is total, which makes a
//     lower-case code a perfectly good one — but the registry keys its map on
//     the string and every key in it came out of wire.EncodeRoom, so without the
//     re-encoding in roomOf a player who typed their code in lower case would be
//     told the room is unknown *while the room sat right there*. That is the
//     worst shape a bug has, and it is the only one of these four that a
//     mutation could reach.
func TestAnUnknownCodeIsRefusedOverARealConnection(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, config(11, 1, room.DefaultAllowance), dependencies)

	// A code for a room behind the same address that nothing is running under.
	// 200 is far past the one room this test opened and inside the byte.
	stale, err := wire.EncodeRoom(held.at, 200)
	if err != nil {
		t.Fatalf("encode a stale code: %v", err)
	}
	if stale == code {
		t.Fatalf("the stale code %s is the code the registry handed out", stale)
	}

	for _, refused := range []struct {
		what string
		code string
	}{
		{"a code naming a room nothing is running", string(stale)},
		{"a string that is not a code", "not-a-code"},
		{"a string of the right length in the right alphabet", strings.Repeat("A", wire.RoomCodeLength)},
	} {
		answer := sayHello(t, held, refused.code,
			hello(t, theHostSquad(t, dependencies.Characters), "Stranger", ""))
		refusal, carried := answer.(wire.Refused)
		if !carried {
			t.Errorf("%s was answered with a %s, want a refusal", refused.what, answer.Kind())
			continue
		}
		if refusal.Code != wire.CodeRoomUnknown {
			t.Errorf("%s was refused with %q, want %q", refused.what, refusal.Code, wire.CodeRoomUnknown)
		}
	}

	// And the fourth: the code as a player might paste it, in the wrong case.
	typed := strings.ToLower(string(code))
	if typed == string(code) {
		t.Fatalf("the code %s has no lower-case form, so this measures nothing", code)
	}
	answer := sayHello(t, held, typed, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""))
	welcome, seated := answer.(wire.Welcome)
	if !seated {
		t.Fatalf("a code typed in lower case (%s) was answered with a %s, want a welcome", typed, answer.Kind())
	}
	if welcome.Seat != wire.SeatHost {
		t.Errorf("a lower-case code was welcomed into the %q seat", welcome.Seat)
	}
	t.Logf("three codes refused with %s; %q welcomed as %s", wire.CodeRoomUnknown, typed, welcome.Seat)
}

// TestAWrongPasswordIsRefusedAndNeverPrinted is the room password's two owed
// properties reaching a socket: the comparison is the room's (constant time,
// wire.Password.Equal) and the transport's share is that it **passes the hello
// through and never says a word about it**.
//
// ⚠️ The second half is measured over two shapes, because they fail differently.
// A hello that **decodes** is safe by the type — wire.Password redacts itself
// under every fmt verb, which is what internal/wire's own reflection test pins —
// but a hello that **does not decode** is bytes with no type left to do the
// redacting, and json's own errors quote what they choked on. So a malformed
// hello carrying the password is sent as well, and the sink is read for the
// password after both.
//
// The right password is tried too, so the room is refusing a password rather
// than refusing everybody.
func TestAWrongPasswordIsRefusedAndNeverPrinted(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	guarded := config(11, 1, room.DefaultAllowance)
	guarded.Password = fixturePassword
	code := held.open(t, guarded, dependencies)

	// The wrong password, through the client's own dial, which is where a screen
	// would meet the refusal.
	_, err := Dial(context.Background(), code,
		hello(t, theHostSquad(t, dependencies.Characters), "Stranger", "open-sesame"),
		dependencies.Books, ClientOptions{Timings: held.timings})
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("a wrong password was answered with %v, want a *Refusal", err)
	}
	if refusal.Code != wire.CodeBadPassword {
		t.Errorf("a wrong password was refused with %q, want %q", refusal.Code, wire.CodeBadPassword)
	}

	// A hello nothing can decode, whose bytes hold the password. It is the only
	// route by which the characters could reach an error at all.
	mangled := `{"kind":"hello","body":{"protocol":1,"password":"` + fixturePassword + `","data":42}}`
	answer := sayRaw(t, held, string(code), mangled)
	unreadable, carried := answer.(wire.Refused)
	if !carried || unreadable.Code != wire.CodeUnknownMessage {
		t.Errorf("a hello nothing can decode was answered %v, want %q", answer, wire.CodeUnknownMessage)
	}

	// Nothing the transport said may hold it.
	for _, said := range held.failures.everything() {
		if strings.Contains(said, fixturePassword) {
			t.Errorf("the transport reported a line holding the room's password: %q", said)
		}
	}

	// And the right password gets in, so the refusal above was about the
	// password and not about the room.
	client := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", fixturePassword),
		dependencies.Books)
	if client.Seat() != wire.SeatHost {
		t.Errorf("the right password took the %q seat", client.Seat())
	}
	// ⚠️ And the redaction holds on the value that did decode, which is the half
	// the type is responsible for: printing a whole hello may not spell it.
	joining := hello(t, theHostSquad(t, dependencies.Characters), "Host", fixturePassword)
	if printed := joining.Password.String(); strings.Contains(printed, fixturePassword) {
		t.Errorf("a password prints as %q", printed)
	}
}

// sayHello opens a real connection to a path, says one hello and reads the one
// answer, then closes.
//
// It hand-dials rather than going through Dial because two of its callers need a
// code Dial would refuse before it ever connected — and the claim is that the
// **server** refuses them, over a socket.
func sayHello(t *testing.T, held *listener, code string, joining wire.Hello) wire.Body {
	t.Helper()
	peer := rawDial(t, held, code)
	defer peer.drop()
	ctx := context.Background()
	if err := peer.send(ctx, joining); err != nil {
		t.Fatalf("say hello with code %q: %v", code, err)
	}
	answer, err := peer.read(ctx)
	if err != nil {
		t.Fatalf("read the answer to a hello with code %q: %v", code, err)
	}
	return valueOf(t, answer)
}

// sayRaw sends bytes this protocol may not even be able to decode, for the one
// claim that is about what happens to bytes rather than to a message.
func sayRaw(t *testing.T, held *listener, code, raw string) wire.Body {
	t.Helper()
	peer := rawDial(t, held, code)
	defer peer.drop()
	ctx := context.Background()
	if err := peer.conn.Write(ctx, websocket.MessageText, []byte(raw)); err != nil {
		t.Fatalf("write raw bytes: %v", err)
	}
	answer, err := peer.read(ctx)
	if err != nil {
		t.Fatalf("read the answer to raw bytes: %v", err)
	}
	return valueOf(t, answer)
}

// rawDial connects to a path without any of Dial's own checks.
func rawDial(t *testing.T, held *listener, code string) *connection {
	t.Helper()
	url := "ws://" + held.at.String() + roomPrefix + code
	raw, _, err := websocket.Dial(context.Background(), url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	return newConnection(raw, held.timings.withDefaults())
}

// valueOf turns wire.Decode's pointer into the value a test switches on, so a
// case list reads like the room's own.
func valueOf(t *testing.T, body wire.Body) wire.Body {
	t.Helper()
	switch message := body.(type) {
	case *wire.Refused:
		return *message
	case *wire.Welcome:
		return *message
	case *wire.Start:
		return *message
	case *wire.Turn:
		return *message
	case *wire.Closed:
		return *message
	}
	t.Fatalf("a server sent a %T, which is not one of the five", body)
	return nil
}
