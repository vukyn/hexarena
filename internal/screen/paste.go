package screen

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// # Where a pasted string goes, and where it stops
//
// A paste is not a run of keystrokes and must not be turned into one. A terminal
// with bracketed paste on — which is every terminal these clients run under,
// because bubbletea v2's renderer enables the mode unless a View turns it off and
// nothing here does — sends the whole clipboard as **one** tea.PasteMsg carrying
// the text, precisely so that a program can tell text somebody pasted from text
// somebody typed. Replaying it as key presses would put the pasted characters
// through every screen's key switch, where a `/` would open the skill filter, a
// `q` would quit and an `n` would start a squad; the whole point of the message
// is that none of that happens.
//
// ⚠️ **The message existed and nothing read it**, which is the same family as the
// three bubbletea v2 changes recorded in CLAUDE.md: v2 moved a message shape, the
// component under the field already handled the new one, and both clients' Update
// switched on tea.KeyPressMsg and their own messages and nothing else — so every
// paste in either client died in the model. It compiled, it drew, and pasting did
// nothing.
//
// # The one rule: a paste lands where a typed letter would land
//
// That sentence decides everything else here, including the two cases that must
// do nothing. A typed letter reaches `Inputs[Field]` only when that field has the
// keyboard, so a paste reaches it under exactly the same condition, and the
// condition is asked in exactly one place per screen: each screen's `Pasting`.
//
// Two states make it a real question rather than a formality, and both are the
// **ordinary** state of a screen rather than a corner:
//
//   - **A form's field is focused while the form is not in front.** OriginsScreen
//     and SkillsScreen focus `Inputs[0]` in ResetForm, which runs at
//     construction; SquadsScreen's level field is a NumberField, which focuses
//     itself. So a client sitting on the works listing, the skill listing or the
//     squad catalogue has a focused text field the reader cannot see. A paste
//     routed at "the focused field" and nothing else would quietly fill it, and
//     the author would find a half-typed origin waiting for them the next time
//     they pressed `a`. The **mode** check is what stops that — `o.Adding`,
//     `s.FormInFront()`, `s.Mode` — and it is load-bearing, not decorative.
//   - **The form is in front and the keyboard is on something that is not a
//     field.** The origin form's medium row, the skill form's six choosers and
//     five allowlists, a member's character and cell: `moveTo` and `moveField`
//     blur the field they leave and focus the next one only when it is a field,
//     so `Focused()` is already the exact answer and no screen restates which of
//     its rows are text.
//
// `focusedField` is those two halves put together for the two screens that hold a
// slice of fields, and the three that hold named ones ask `Focused()` on the one
// their own key path would have written to.
//
// ⚠️ **bubbles agrees, and that is a backstop rather than the rule.**
// textinput.Update returns immediately when the field is not focused — measured,
// and pinned by TestATextFieldRefusesAPasteWhenItIsNotFocused — so a paste handed
// to a blurred field is dropped by the component even if a screen got its own
// answer wrong. The rule is still written here, because a screen picking the
// wrong *field* is a mistake bubbles cannot see: the one it is handed is focused,
// it is simply not the one the reader is looking at.
//
// # What the clients add on top
//
// Nothing about which field. Each client answers tea.PasteMsg with a route that
// names the screens it can have in front and calls `Paste` on them; every other
// screen falls through to nothing at all. cmd/hexarena-tui's route reaches the
// lobby's two fields and the skill filter and no third thing — the forms in this
// package open behind Context.Authoring, which is nought there, so `FormInFront`
// can never be true on that client and the arm ordering below is what says so.

// PasteInto is the one place a pasted string reaches a text field.
//
// It hands the field the same message a terminal delivers, which is what keeps
// the two routes — the terminal's own bracketed paste and the ctrl+v this program
// answers itself — landing on identical code. The command that comes back is the
// cursor's blink and has to be returned for the reason every three-return Update
// in this package returns one.
//
// ⚠️ **A newline becomes a space, and a tab too — measured, not assumed.**
// textinput sanitises what it inserts through a rune sanitizer built with
// ReplaceNewlines(" ") and ReplaceTabs(" "), so `"7QK4M2XZ9BTF\n"` lands as
// thirteen characters ending in a space and `"…\r\n"` as fourteen ending in two,
// while an escape or any other control character is dropped and invalid UTF-8 is
// discarded. Nothing here re-does that work: a caller that needs a clean value
// trims it where the value is read, which is what joinScreen.submit has always
// done and why its comment already says a pasted code arrives with a newline more
// often than not. The behaviour is pinned by
// TestATextFieldTurnsANewlineInAPasteIntoASpace so a bubbles upgrade that changed
// it would say so here rather than in a room code that would not decode.
//
// ⚠️ **The command is NOT a signal that anything was pasted, and reading it as
// one is a mistake this shipped with for one commit.** It is the cursor's blink,
// and a field on a plain terminal has no virtual cursor — NewInput turns it off
// under NO_COLOR — so a paste that landed perfectly well hands back **nil** on
// every machine the suite runs on. A caller that needs to know whether the value
// moved compares the value, which is what every arm calling this does.
func PasteInto(field *textinput.Model, text string) tea.Cmd {
	updated, command := field.Update(tea.PasteMsg{Content: text})
	*field = updated
	return command
}

// PasteDigits is PasteInto for a number field, which the level and the chance
// are.
//
// ⚠️ **A paste that is not all digits is refused whole rather than stripped.**
// Refused, because these two fields' typed path is NumberKey-gated and a paste is
// the only other way in: a field that took letters from one route and refused
// them from the other would be numeric only by accident, and the value would sit
// there looking accepted while strconv.Atoi quietly ignored it — which is exactly
// the failure NumberKey's own comment records having been written the other way
// round first. Whole, because stripping "1,200" down to "1200" would enter a
// number nobody pasted, and a level that silently became something else is worse
// than a keystroke that did nothing.
//
// An empty paste is refused too: there is nothing in it to be a digit.
//
// ⚠️ **A nil answer does not mean refused.** It carries PasteInto's nil for the
// reason written there, so the two are indistinguishable and a caller that has to
// tell them apart reads the field's value either side. Both callers do.
func PasteDigits(field *textinput.Model, text string) tea.Cmd {
	if !allDigits(text) {
		return nil
	}
	return PasteInto(field, text)
}

func allDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, letter := range text {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return true
}

// PasteText is a pasted string made fit for a field that is **not** a
// textinput — which in this package is the skill listing's typed filter, a plain
// string with a backspace for the reasons SkillsScreen.Query records.
//
// ⚠️ **It is the same rule bubbles applies and is held to it by a test rather
// than by this comment.** A second sanitiser is a second answer to "what does a
// pasted newline become", and the two would drift the first time either moved;
// TestTheFilterSanitisesAPasteExactlyAsATextFieldDoes feeds the same strings to
// this and to a real textinput and holds the answers equal, so a bubbles upgrade
// that changed the rule fails here instead of leaving one field behaving unlike
// every other.
func PasteText(text string) string {
	var out strings.Builder
	for _, letter := range text {
		switch {
		case letter == utf8.RuneError:
			// A byte that is not UTF-8 decodes to the replacement rune, and it is
			// dropped rather than drawn: nobody pasted that glyph.
		case letter == '\n' || letter == '\r' || letter == '\t':
			out.WriteRune(' ')
		case unicode.IsControl(letter):
			// Every other control character is dropped, escape included. It is
			// unicode.IsControl rather than a range because that is the predicate
			// bubbles uses, and half a rule is how the two would come apart.
		default:
			out.WriteRune(letter)
		}
	}
	return out.String()
}

// focusedField is the field a form's keyboard is on, and nil when it is on
// something that is not a field or when the form has not been built at all.
//
// The nil is a real answer rather than a guard against a bug: a chooser row and
// an allowlist row are both rows a form's cursor sits on with every field
// blurred, and a screen assembled by hand — which is how both clients' sweeps
// reach the form states — has no Inputs until ResetForm has run.
//
// ⚠️ **The pointer is into the screen's own slice, which two copies of a screen
// share.** That is the sharing SkillsScreen.Inputs documents and
// TestTheSkillFormsFieldsAreSharedBetweenCopies pins; it is what lets Paste take
// a value receiver here and still write where the client will read.
func focusedField(inputs []textinput.Model, field int) *textinput.Model {
	if field < 0 || field >= len(inputs) || !inputs[field].Focused() {
		return nil
	}
	return &inputs[field]
}
