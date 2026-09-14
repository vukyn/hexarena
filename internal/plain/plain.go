// Package plain makes somebody else's bytes fit to print on a terminal.
//
// A terminal reads what it is handed as instructions as well as text. A run
// beginning with ESC is a command, and the commands that matter here are not
// cosmetic: **OSC 52 writes the reader's clipboard** — the payload is pasted
// into a shell some minutes later, by the reader, on purpose — while OSC 0
// renames their window and CSI 2J clears their screen, which is enough to make
// output say whatever the writer of the bytes wanted it to say.
//
// Two strings in this module are written by somebody who is not the person
// reading them: a **name off the wire** (wire.Hello.Name, typed by a stranger on
// the network) and the **free text in a saved log** (an event's name, note or
// target, in a file that may have arrived from anywhere). Both pass through here
// before they reach a writer, so what the terminal is handed is text and nothing
// else.
//
// # One declaration, three callers
//
// internal/wire cleans a hello's name where it is decoded, internal/tui cleans a
// log's strings where they are rendered, and internal/screen's PasteText is this
// rule with a text field's own answer to a newline layered on it. A second
// predicate would be a second answer to "what does a pasted ESC become", and the
// two would come apart the first time either moved — which is the argument
// PasteText's own doc already makes about bubbles.
//
// # Why it is a package of its own
//
// The two callers that matter share no other layer: internal/wire is the
// protocol and internal/tui is a renderer, and neither may import the other. The
// predicate already lived in internal/screen, which cannot be the home for it
// because importing that package drags bubbletea into a host binary that draws
// no screen. So this imports **the standard library only**, and nothing under
// internal/core imports it — internal/core still imports nothing outside the
// standard library, which is what the replay contract rests on.
package plain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Text is text with everything a terminal would obey taken out of it.
//
// The predicate is unicode.IsControl rather than a hand-written range, which is
// the choice internal/screen's PasteText records and the reason it is worth
// repeating here: half a rule is how two sanitisers come apart. It also happens
// to be the complete answer rather than a lucky one — ESC (U+001B), BEL
// (U+0007), DEL (U+007F) and the **C1 controls** (U+0080–U+009F, which are the
// eight-bit spellings of the same commands, U+009D being OSC itself) are all
// category Cc, so one predicate covers both spellings of an escape rather than
// the seven-bit one a range would have been written for.
//
// A newline, a carriage return and a tab become a space rather than vanishing.
// They carry a word boundary, and a name that read "ab" where somebody typed
// "a<tab>b" would be a quieter kind of wrong than one that reads "a b". A
// carriage return is not a harmless blank either: it is how a line already
// printed is written over, which is the cheap way to forge a line without an
// escape sequence at all.
//
// A byte that is not UTF-8 decodes to utf8.RuneError and is dropped rather than
// written back: nobody typed that glyph, and a lone 0x9B — the one-byte CSI —
// arrives exactly that way.
//
// ⚠️ **This is a removal and not an escaping, so there is nothing to undo.** The
// bytes are gone rather than quoted, which is right for text on its way to a
// screen and wrong for text on its way into a file. Nothing here is a
// round-trip.
//
// ⚠️ **It does not bound the length**, on purpose: how long a string may be is a
// question about the field it is going in — a protocol's limit and a screen's
// column are different numbers — so the callers that need one say so themselves
// (→ wire.NameLength).
func Text(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	for _, letter := range text {
		switch {
		case letter == utf8.RuneError:
			// A byte that is not UTF-8 decodes to the replacement rune, and it is
			// dropped rather than drawn: nobody wrote that glyph.
		case letter == '\n' || letter == '\r' || letter == '\t':
			out.WriteRune(' ')
		case unicode.IsControl(letter):
			// Every other control character is dropped, escape included.
		default:
			out.WriteRune(letter)
		}
	}
	return out.String()
}

// Inert reports whether text is already what Text would return, which is what a
// test asserting "no escape reached the writer" actually wants to ask.
//
// ⚠️ It is here rather than written out at each call site because the question
// and the answer have to be the same rule: a test that looked for '\x1b' alone
// would pass on a string carrying U+009D, and one that compared against its own
// idea of a control character would be the second predicate this package exists
// to prevent.
func Inert(text string) bool { return Text(text) == text }
