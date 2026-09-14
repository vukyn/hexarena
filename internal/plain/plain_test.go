package plain_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/plain"
	"github.com/vukyn/hexarena/internal/screen"
)

// osc52 is the payload the whole package exists for, written out in bytes rather
// than described.
//
// ⚠️ **It is the one that is not cosmetic.** ESC ] 52 ; c ; <base64> BEL tells
// the terminal to put <base64> on the reader's clipboard, and the reader pastes
// it into a shell some minutes later believing it is what they copied. It is 22
// bytes here, comfortably inside the 32 runes cmd/hexarena-host used to allow,
// which is why a length bound on its own refused nothing worth refusing.
const osc52 = "\x1b]52;c;cm0gLXJmIH4=\x07"

// eightBit is the same instruction in its C1 spelling: U+009D is OSC and U+009C
// is the string terminator, so a terminal reading eight-bit controls obeys this
// exactly as it obeys osc52 — and a sanitiser written as `!= '\x1b'` lets it
// through.
const eightBit = "\u009d52;c;cm0gLXJmIH4=\u009c"

func TestEveryControlCharacterIsTakenOut(t *testing.T) {
	for _, each := range []struct {
		what  string
		given string
		want  string
	}{
		{"an OSC 52 clipboard write", "Bảo" + osc52, "Bảo]52;c;cm0gLXJmIH4="},
		{"the same thing spelled eight-bit", "Bảo" + eightBit, "Bảo52;c;cm0gLXJmIH4="},
		{"a window title and a cleared screen", "\x1b]0;owned\x07\x1b[2Jx", "]0;owned[2Jx"},
		{"a carriage return, which overwrites a line already printed", "a\rb", "a b"},
		{"a newline, which is a line the writer did not get", "a\nb", "a b"},
		{"a tab", "a\tb", "a b"},
		{"a DEL", "a\x7fb", "ab"},
		{"a lone CSI byte, which is not UTF-8 at all", "a\x9bb", "ab"},
		{"an ordinary name", "Bảo Nguyễn", "Bảo Nguyễn"},
		{"an empty string", "", ""},
	} {
		t.Run(each.what, func(t *testing.T) {
			if got := plain.Text(each.given); got != each.want {
				t.Errorf("plain.Text(%q) = %q, want %q", each.given, got, each.want)
			}
		})
	}
}

// TestAnHonestStringComesBackByteForByte is the property every golden in this
// module rests on. inert is applied to nine fields of every event rendered, so if
// plain.Text touched anything a real log holds, every testdata file would have to
// be re-accepted — and the change would be invisible in this package.
func TestAnHonestStringComesBackByteForByte(t *testing.T) {
	for _, honest := range []string{
		"Bảo", "wind.dancer", "ember-2", "A1", "E5", "ally", "crit x1.5",
		"Nguyễn Thị Ánh", "skill.oath_of_ash", "left 3 charges", "σ ± 12‰", "—",
	} {
		if got := plain.Text(honest); got != honest {
			t.Errorf("plain.Text(%q) = %q: an honest string was altered, which every golden would notice", honest, got)
		}
		if !plain.Inert(honest) {
			t.Errorf("plain.Inert(%q) is false for a string plain.Text does not change", honest)
		}
	}
}

// TestInertAgreesWithText holds the two halves of this package to one answer. A
// test elsewhere asks "did an escape reach the writer" through Inert, so an Inert
// that were laxer than Text would make every one of those tests pass while
// proving nothing.
func TestInertAgreesWithText(t *testing.T) {
	for _, each := range []string{"", "Bảo", osc52, eightBit, "a\rb", "a\x7fb", "a\x9bb", "plain"} {
		if plain.Inert(each) != (plain.Text(each) == each) {
			t.Errorf("plain.Inert(%q) disagrees with plain.Text on the same string", each)
		}
	}
	if plain.Inert(osc52) {
		t.Error("plain.Inert says an OSC 52 is already fit to print")
	}
}

// TestTheScreensPasteIsThisSameRule holds the move that made this package: the
// predicate used to live in internal/screen and now there is one of it. If
// somebody re-inlines a loop into PasteText, this fails — which is the whole
// reason the delegation is worth a test rather than a comment.
//
// ⚠️ It does **not** replace internal/screen's own test against a real bubbles
// textinput. That one is the claim that a paste behaves like a text field; this
// one is the claim that a terminal and a text field are currently given the same
// answer. Both have to hold, and if bubbles ever parts them it is that test that
// must fail rather than this one.
func TestTheScreensPasteIsThisSameRule(t *testing.T) {
	for _, each := range []string{"Bảo" + osc52, "a\nb", "a\tb", "a\x9bb", "ordinary"} {
		if screen.PasteText(each) != plain.Text(each) {
			t.Errorf("screen.PasteText(%q) = %q but plain.Text = %q: two answers to one question",
				each, screen.PasteText(each), plain.Text(each))
		}
	}
}

// TestNoEscapeSurvivesAnyPrefixOrSuffix is the crude sweep the table above is
// not: it drops an escape at every position of a plausible name and asserts the
// result carries no control byte at all, so a fix that happened to handle a
// leading ESC and not a trailing one is caught.
func TestNoEscapeSurvivesAnyPrefixOrSuffix(t *testing.T) {
	host := "Bảo Nguyễn"
	for cut := 0; cut <= len(host); cut++ {
		if cut > 0 && host[cut-1] >= 0x80 && cut < len(host) && host[cut]&0xc0 == 0x80 {
			continue // never split a rune: that is a different test
		}
		for _, payload := range []string{osc52, eightBit, "\x1b[2J", "\r"} {
			got := plain.Text(host[:cut] + payload + host[cut:])
			for _, letter := range got {
				if letter < 0x20 || (letter >= 0x7f && letter <= 0x9f) {
					t.Fatalf("plain.Text left %U in %q", letter, got)
				}
			}
			if strings.ContainsRune(got, 0x1b) {
				t.Fatalf("plain.Text left an ESC in %q", got)
			}
		}
	}
}
