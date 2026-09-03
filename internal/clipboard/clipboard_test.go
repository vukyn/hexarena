package clipboard

import (
	"errors"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	system "github.com/atotto/clipboard"
)

// TestReadIsTheSystemClipboard is the half no injected reader can see.
//
// Both clients' suites replace Read with a function of their own, which is what
// keeps them off a machine's real clipboard — and it means every one of those
// tests would still pass if Read had been left pointing at something that never
// reads a clipboard at all. This is the assertion that the wiring is real, and it
// is made by **identity** rather than by calling: calling would shell out to
// pbpaste, which is the thing those tests avoid.
//
// *Sees:* Read repointed at a stub, a wrapper, or a platform-specific helper that
// is not the library's own entry point.
// *Cannot see:* whether atotto/clipboard works on the machine this runs on. It
// cannot, and that is the point — the failure mode on a machine with no helper is
// a fast error, which the route turns into no message at all.
func TestReadIsTheSystemClipboard(t *testing.T) {
	want := reflect.ValueOf(system.ReadAll).Pointer()
	if got := reflect.ValueOf(Read).Pointer(); got != want {
		t.Errorf("Read is not github.com/atotto/clipboard.ReadAll, so every test that "+
			"injects a reader is measuring a wire that goes nowhere (got %#x, want %#x)",
			got, want)
	}
}

// TestPasteSaysWhatTheClipboardHeld is the whole of the command's happy path.
//
// ⚠️ **The message has to be tea.PasteMsg and nothing else**, because that is the
// one type both clients name in their Update switch. bubbles' own textinput.Paste
// answers with an unexported type instead, which is exactly why it could not be
// used and why this assertion is on the concrete type rather than on "some
// message".
func TestPasteSaysWhatTheClipboardHeld(t *testing.T) {
	hand(t, func() (string, error) { return "7QK4M2XZ9BTF", nil })
	message := Paste()
	pasted, named := message.(tea.PasteMsg)
	if !named {
		t.Fatalf("Paste produced a %T, which neither client can name", message)
	}
	if pasted.Content != "7QK4M2XZ9BTF" {
		t.Errorf("the message carries %q, want what the clipboard held", pasted.Content)
	}
}

// TestPasteSaysNothingWhenThereIsNothingToSay is the Linux box with no xclip, and
// the empty clipboard beside it.
//
// Both come back as no message at all rather than as an empty one: bubbletea
// skips a nil message, so nothing is inserted, nothing is redrawn, and no screen
// grows a diagnostic about a keystroke somebody pressed out of habit.
//
// *Sees:* an error turned into a message a client would route; an empty read
// turned into a redraw for nothing.
// *Cannot see:* that the real reader fails fast rather than blocking on a machine
// with no helper. atotto/clipboard probes at init, which is recorded in the
// package comment rather than measured here — a test that proved it would have to
// uninstall xclip.
func TestPasteSaysNothingWhenThereIsNothingToSay(t *testing.T) {
	t.Run("a reader that fails", func(t *testing.T) {
		hand(t, func() (string, error) {
			return "", errors.New("no clipboard utilities available")
		})
		if message := Paste(); message != nil {
			t.Errorf("a failed read produced %#v, want no message at all", message)
		}
	})

	t.Run("a reader that fails and still returns text", func(t *testing.T) {
		// The error wins. A helper that printed a diagnostic on stdout and exited
		// non-zero would otherwise have its diagnostic pasted into a room code.
		hand(t, func() (string, error) {
			return "xclip: cannot open display", errors.New("exit status 1")
		})
		if message := Paste(); message != nil {
			t.Errorf("a failed read produced %#v even though it returned text", message)
		}
	})

	t.Run("an empty clipboard", func(t *testing.T) {
		hand(t, func() (string, error) { return "", nil })
		if message := Paste(); message != nil {
			t.Errorf("an empty clipboard produced %#v, want no message at all", message)
		}
	})
}

// hand replaces the reader for one test and puts the real one back.
func hand(t *testing.T, reader func() (string, error)) {
	t.Helper()
	restore := Read
	t.Cleanup(func() { Read = restore })
	Read = reader
}
