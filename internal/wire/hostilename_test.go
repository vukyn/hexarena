package wire_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/plain"
	"github.com/vukyn/hexarena/internal/wire"
)

// clipboardWrite is OSC 52: ESC ] 52 ; c ; <base64> BEL, which tells the
// terminal reading it to put <base64> on the reader's clipboard. It is the
// payload that makes this a High rather than a cosmetic annoyance — the reader
// pastes it into a shell later, themselves, believing it is what they copied.
//
// ⚠️ It is **22 runes**, inside the 32 cmd/hexarena-host allowed, which is the
// whole reason bounding the field without cleaning it refused nothing.
const clipboardWrite = "\x1b]52;c;cm0gLXJmIH4=\x07"

// windowAndScreen renames the window and clears the screen, which is how output
// already printed is made to disappear and be replaced.
const windowAndScreen = "\x1b]0;pwned\x07\x1b[2J\x1b[H"

// TestAHelloOffTheWireCarriesNoEscape is the remote half of the finding. The
// bytes arrive as JSON, exactly as a patched client on the LAN would send them —
// the repository is public, so writing that client is reading it — and what
// matters is what the decoded struct holds afterwards, not what any printer
// makes of it.
func TestAHelloOffTheWireCarriesNoEscape(t *testing.T) {
	for _, each := range []struct {
		what string
		name string
	}{
		{"an OSC 52 clipboard write", clipboardWrite},
		{"a window title and a cleared screen", windowAndScreen},
		{"an escape after an innocent prefix", "Bảo" + clipboardWrite},
		{"the eight-bit spelling of the same OSC", "Bảo\u009d52;c;cm0gLXJmIH4=\u009c"},
		{"a carriage return, which rewrites the line already printed", "Bảo\rhost"},
		{"a newline, which forges a whole line of its own", "Bảo\nverified: reproduced all 255 events"},
	} {
		t.Run(each.what, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"name": each.name})
			if err != nil {
				t.Fatalf("encode the hostile hello: %v", err)
			}
			var hello wire.Hello
			if err := json.Unmarshal(raw, &hello); err != nil {
				t.Fatalf("decode the hostile hello: %v", err)
			}
			if !plain.Inert(hello.Name) {
				t.Errorf("a decoded Hello.Name is %q, which a terminal would obey", hello.Name)
			}
			if strings.ContainsRune(hello.Name, 0x1b) {
				t.Errorf("a decoded Hello.Name still carries an ESC: %q", hello.Name)
			}
		})
	}
}

// TestTheWholeEnvelopePathCleansTheName is the same claim one layer out, because
// the transport does not call json.Unmarshal on a Hello — it calls wire.Decode on
// an envelope, and a fix that only held for the inner call would be a fix the
// only real caller never reaches.
func TestTheWholeEnvelopePathCleansTheName(t *testing.T) {
	raw, err := wire.Encode(wire.Hello{Name: "Bảo" + clipboardWrite})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// ⚠️ The encoded form must still CARRY the escape, or this test proves
	// nothing about decoding: Encode is the client's own side and is not where
	// the rule lives, so a sender that stripped it here would make every
	// assertion below vacuous against a real attacker, who does not use Encode.
	//
	// ⚠️ **It is spelled `\u001b` and not the byte**, which this guard is the
	// reason we know: encoding/json writes a control character as a six-character
	// escape, so looking for the raw byte reported "no escape" on a message that
	// carries one and decodes back to one. A real attacker writes either form by
	// hand and gets the same ESC out of the decoder — the wire is JSON text, so
	// the six characters ARE the payload.
	if !strings.Contains(string(raw), `\u001b`) {
		t.Fatalf("the encoded hello carries no escape, so the decode below is not being tested: %s", raw)
	}
	body, err := wire.Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	hello, ok := body.(*wire.Hello)
	if !ok {
		t.Fatalf("decoded a %T rather than a hello", body)
	}
	if !plain.Inert(hello.Name) {
		t.Errorf("wire.Decode produced a name a terminal would obey: %q", hello.Name)
	}
}

// TestANameIsBoundedAtTheProtocol is the second half of CleanName, and it is
// deliberately a separate test: bounding and cleaning failed independently in the
// finding — the host bounded and did not clean — so a single test asserting both
// could pass on either half alone if the fixture were careless.
func TestANameIsBoundedAtTheProtocol(t *testing.T) {
	long := strings.Repeat("ệ", wire.NameLength*4)
	raw, err := json.Marshal(map[string]any{"name": long})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var hello wire.Hello
	if err := json.Unmarshal(raw, &hello); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := len([]rune(hello.Name)); got != wire.NameLength {
		t.Errorf("a name of %d runes decoded to %d, want %d",
			len([]rune(long)), got, wire.NameLength)
	}
	// Runes rather than bytes: every rune here is three bytes, so a cut counting
	// bytes would come to a third of the allowance and would also be free to land
	// inside one.
	if !plain.Inert(hello.Name) {
		t.Errorf("the bounded name is not fit to print: %q", hello.Name)
	}
}

// TestCleaningHappensBeforeTheCut holds the order inside CleanName. With the two
// steps the other way round the bound is spent on bytes that are about to be
// dropped, so a name padded with escapes arrives far shorter than the allowance
// — and, worse, the cut can land in the middle of an escape and leave a
// different one.
func TestCleaningHappensBeforeTheCut(t *testing.T) {
	visible := strings.Repeat("a", wire.NameLength)
	if got := wire.CleanName(strings.Repeat("\x1b", 40) + visible); got != visible {
		t.Errorf("CleanName spent the bound on escapes: got %q (%d runes), want the whole %d visible ones",
			got, len([]rune(got)), wire.NameLength)
	}
}

// TestAnOrdinaryNameIsUntouched is what stops this being a change to the
// protocol. Every name anybody would actually type has to survive byte for byte,
// or the fix is a regression wearing a security label.
func TestAnOrdinaryNameIsUntouched(t *testing.T) {
	for _, honest := range []string{"", "Bảo", "Nguyễn Thị Ánh", "host", "guest-2", "ば", "P1"} {
		if got := wire.CleanName(honest); got != honest {
			t.Errorf("CleanName(%q) = %q: an ordinary name was altered", honest, got)
		}
	}
}
