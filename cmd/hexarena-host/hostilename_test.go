package main

import (
	"context"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/plain"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// hostileName is the payload of the remote finding, as a name a peer sends.
//
// ⚠️ **OSC 52 is the one that matters.** ESC ] 52 ; c ; <base64> BEL puts
// <base64> on the clipboard of whoever is reading the terminal, and they paste it
// into a shell later, themselves. The other two are the theatre around it: OSC 0
// renames the window and CSI 2J clears the screen, which together let a line the
// host already read be replaced with one the peer wrote.
//
// ⚠️ **It fits in the old allowance with room to spare**, which is the finding:
// playerName cut at 32 runes and stripped nothing, so the cut was not a defence
// and never had been.
const hostileName = "\x1b]0;pwned\x07\x1b]52;c;cm0gLXJmIH4=\x07\x1b[2J"

// TestANameOffTheWireReachesThisScreenAsTextAndNothingElse is the remote High,
// end to end, over a real listener and a real client.
//
// ⚠️ **It asserts the BYTES on the host's paper, not a rendered string.** The
// temptation is to call playerName on a hostile string and check the answer,
// which tests a function rather than a path: the name reaching this screen does
// not come off the seat the room stored — internal/socket hands Options.Joined
// the hello's own field — so a fix applied only to the room would leave this test
// green through a function nobody's bytes travel through. What is measured here
// is everything that was actually written to the host's stdout while a peer with
// a hostile name joined.
//
// ⚠️ **No password is set**, deliberately: the room in this test is the default
// one, so the attack needs nothing but the address — which -browse hands out over
// mDNS to anybody on the segment.
func TestANameOffTheWireReachesThisScreenAsTextAndNothingElse(t *testing.T) {
	out, errs := newPaper(), newPaper()
	held := hosting(t, aRoom(), dialableAt, out.on, errs.on)
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	joining := wire.Hello{
		Version: dependencies.Version,
		Squad:   squadOf(t, dependencies.Characters, "joiner"),
		Name:    hostileName,
	}
	ctx, cancel := context.WithTimeout(context.Background(), theWholeShutdown)
	defer cancel()
	client, err := socket.Dial(ctx, held.code, joining, dependencies.Books, socket.ClientOptions{})
	if err != nil {
		t.Fatalf("join the room this host opened: %v", err)
	}
	defer client.Close()

	// Waited for rather than read, for the reason eventually records: Joined
	// fires on a connection's own goroutine after the welcome has gone out. The
	// thing waited on is the ordinary half of the line, which survives the
	// cleaning, so the wait ends when the join has been announced.
	said := out.eventually(t, "joined as host")
	if !strings.Contains(said, "joined as host") {
		t.Fatalf("the join was never announced, so nothing was measured:\n%q", said)
	}
	for _, page := range []struct {
		what string
		said string
	}{{"stdout", said}, {"stderr", errs.said()}} {
		// plain.Inert rather than a hand-written search for ESC: the whole point
		// of the package is that "what a terminal obeys" has one definition, and
		// a test that asked the question its own way would pass on the C1
		// spelling. The explicit ESC check below is kept beside it as the
		// human-legible half, not as the assertion that carries the weight.
		//
		// ⚠️ The newline the host writes at the end of its line is exactly what a
		// naive "no control bytes" sweep would trip on, which is why the
		// predicate is the one that turns a newline into a space rather than one
		// that bans it: a line ending is the writer's byte, not the peer's.
		if !plain.Inert(strings.ReplaceAll(page.said, "\n", " ")) {
			t.Errorf("%s carries bytes a terminal would obey: %q", page.what, page.said)
		}
		if strings.Contains(page.said, "\x1b") {
			t.Errorf("%s carries a raw ESC: %q", page.what, page.said)
		}
	}
}

// TestTheBoundIsTheProtocolsAndNotThisScreensAlone holds the second half of the
// old bug. The host cut at 32 runes and that cut is still here, but the figure is
// now the protocol's, so the two cannot drift into disagreement — and a name
// arriving over a socket is already within it, which is what makes this screen's
// cut a fallback rather than the defence.
func TestTheBoundIsTheProtocolsAndNotThisScreensAlone(t *testing.T) {
	long := strings.Repeat("ệ", wire.NameLength*3)
	// Handed over directly, NOT decoded: this is the path a hello built in Go
	// takes, which is the only one where the screen's own cut still does work.
	if got := len([]rune(playerName(long))); got != wire.NameLength+1 {
		t.Errorf("playerName kept %d runes of %d, want %d plus the ellipsis",
			got, len([]rune(long)), wire.NameLength)
	}
	// And through the protocol, where the cut has already happened, so no
	// ellipsis is added and the two bounds agree by construction.
	if got := playerName(wire.CleanName(long)); len([]rune(got)) != wire.NameLength {
		t.Errorf("a decoded name of %d runes drew as %d: the two bounds disagree",
			wire.NameLength, len([]rune(got)))
	}
}

// TestAnEmptyNameStillReadsAsSomebody is the one behaviour of playerName that is
// not about bytes, kept because the cleaning can now produce an empty string out
// of a name that was not empty — a peer whose whole name was an escape sequence
// arrives here as "". Before the strip that was unreachable.
func TestAnEmptyNameStillReadsAsSomebody(t *testing.T) {
	if got := playerName(wire.CleanName("\x1b\x1b\x07")); got != "somebody" {
		t.Errorf("a name that cleaned away entirely drew as %q, want %q", got, "somebody")
	}
}
