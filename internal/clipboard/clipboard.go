// Package clip is the system clipboard, read for one keystroke.
//
// # Two routes into a text field, and this is the second one
//
// A terminal that speaks bracketed paste intercepts ⌘V, ctrl+shift+V and
// right-click-paste itself and injects the clipboard as a bracketed paste, which
// arrives as a tea.PasteMsg carrying the text. Nothing has to read a clipboard
// for any of those — the terminal already did.
//
// Plain ctrl+v is the one a terminal does **not** intercept. It arrives as an
// ordinary keystroke, so the program has to read the clipboard itself, and this
// package is the whole of that. What it produces is a **tea.PasteMsg**, the same
// message the terminal's own paste delivers, so the two routes converge on one
// insert rather than being two ways for text to reach a field.
//
// ⚠️ **bubbles' own textinput.Paste cannot be used for this, and it was measured
// rather than assumed.** textinput's default key map already binds ctrl+v, and a
// ctrl+v that reaches a focused field returns textinput.Paste as its command —
// which really does read the clipboard. But the message that command produces is
// `textinput.pasteMsg`, an **unexported** type: a model outside that package
// cannot name it in a type switch, so it arrives at Update, matches nothing and
// dies, and the field it came from never sees it. Measured end to end: ctrl+v on
// the join screen shells out to pbpaste today and inserts nothing. The only way
// to route it would be for a client to forward every message it cannot name into
// whatever field is focused, which is a much wider door than this feature needs.
// So the clipboard is read here and the answer is said in a message both clients
// can name.
//
// # ⚠️ This is not the clipboard cmd/hexarena-host refuses, and the distinction matters
//
// cmd/hexarena-host's doc comment declines a clipboard on purpose — "it needs
// pbcopy/xclip/wl-copy and is a per-platform external dependency" — and that
// refusal stands. It is about a **non-interactive CLI** shelling out to put the
// room code *out* on a machine nobody may be sitting at, where a missing helper
// would be a feature that silently did not happen on somebody's server.
//
// This is a **text field in a full-screen client offering a paste key** to a
// person who is at the keyboard: the dependency is already in the module (bubbles
// pulls github.com/atotto/clipboard for the very command described above), the
// component already implements the insert, and a helper that is missing costs
// exactly the keystroke — ⌘V still works, because that route never reaches here.
// Different question, different answer.
//
// # What a missing helper costs
//
// On darwin the reader shells out to pbpaste, which ships with macOS. On unix it
// wants xsel, xclip, wl-paste or termux-clipboard-get, none of which is
// guaranteed; atotto/clipboard probes for them **at init** and ReadAll then
// returns an error immediately rather than blocking, so the failure is a fast
// error and never a hang. Paste turns it into no message at all, which is the
// quietest outcome there is: bubbletea skips a nil message, so nothing is
// inserted, nothing is drawn, and no screen grows a diagnostic about a keystroke
// somebody pressed by habit. That is deliberate — a game screen is not the place
// to explain a Linux packaging decision, and there is no wording here for exactly
// that reason.
package clipboard

import (
	tea "charm.land/bubbletea/v2"
	// Aliased because the package this one is named after is the thing it wraps.
	// The name belongs here rather than to the dependency: what a caller wants is
	// "the clipboard", and which library reads it is this file's business.
	system "github.com/atotto/clipboard"
)

// Read is what a paste reads.
//
// A variable rather than a call so a suite can hand in a clipboard of its own.
// A test that drove the real one would fail on a machine with no helper and pass
// for the wrong reason on a machine whose clipboard happened to hold something —
// neither of which measures the wiring. What a suite that replaces it **cannot**
// see is that the real reader is still the one wired in, so TestReadIsTheSystemClipboard
// holds that separately, by identity.
var Read = system.ReadAll

// Paste is the command ctrl+v runs: read the clipboard, say it as the same
// message a terminal's own paste says.
//
// Nothing on failure and nothing on an empty clipboard, and they are one arm
// deliberately: both come to "there was nothing to paste", and a caller that
// told them apart would be a caller with a sentence to write about each.
func Paste() tea.Msg {
	text, err := Read()
	if err != nil || text == "" {
		return nil
	}
	return tea.PasteMsg{Content: text}
}
