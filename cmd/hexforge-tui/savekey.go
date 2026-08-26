package main

import (
	"runtime"

	tea "charm.land/bubbletea/v2"
)

// The keystrokes that write the work in front of the author.
//
// Two rather than one, and the second is the whole reason this program is on
// bubbletea v2. A terminal cannot deliver the Command key over the classic
// escape sequences — there is no encoding for it, so under v1 the modifier did
// not exist as far as a program was concerned. The Kitty keyboard protocol does
// carry it, v2 parses that protocol, and a Command key arrives as ModSuper.
//
// ⚠️ That does not make ⌘S universally available, and nothing here can. Three
// things have to be true at once: the terminal has to speak the protocol
// (kitty, Ghostty, WezTerm, foot, and iTerm2 once CSI u is on — Terminal.app
// never), it has to be willing to pass ⌘S through rather than opening its own
// Save dialog, and on Linux and Windows a window manager may claim the Super
// key before the terminal sees it. So ctrl+s stays the binding that always
// works, and this is the one that works where it can.
const (
	saveKeyControl = "ctrl+s"
	saveKeyCommand = "super+s"
)

// saveKeyMacLabel is how the pair is written on a Mac, and the space in the
// middle is the whole of it.
//
// It was "⌘S/^S": four symbols and a slash with nothing between them, which on
// screen reads as one smear rather than as two keystrokes. The footer around it
// separates one key from the next with a space, so a key that contains no space
// has nothing to separate it from itself. The space costs nothing — the label is
// five cells either way.
//
// A slash with spaces around it would read better still, and does not fit. The
// English character-form footer is 73 cells without the label, the smallest
// window this program will draw in is 80, and the last cell of a line is left
// empty so that writing it cannot wrap the row — six cells, and "⌘S / ^S" is
// seven. That budget is guarded by TestEverySaveFooterFitsTheSmallestWindow
// rather than left as a number in a comment.
const saveKeyMacLabel = "⌘S ^S"

// isSaveKey reports whether a keystroke asks for the work to be written.
//
// One declaration for all three forms, for the reason a passed turn's reason
// lives in one place: three screens matching on their own spelling of the same
// intent is three chances for one of them to be missed when a fourth keystroke
// is added, and the one that gets missed is the one nobody presses in a test.
func isSaveKey(message tea.KeyPressMsg) bool {
	switch message.String() {
	case saveKeyControl, saveKeyCommand:
		return true
	}
	return false
}

// saveKeyLabel is what a footer calls the save key, which depends on the
// keyboard in front of it rather than on the language being read.
//
// On macOS both are offered and ⌘S is named first, because it is the one a Mac
// keyboard reaches for and the one that needs discovering; ^S follows because it
// is the one that always works, and a footer promising only ⌘S would be a
// promise this program cannot keep on Terminal.app. Everywhere else there is
// nothing to choose between, so the footer says the single true thing.
//
// It is a keystroke rather than a sentence, which is why it lives here and not
// in internal/i18n: ⌘ and ^ are what is printed on the key in every language,
// and a catalogue entry per platform per language would be four ways to say one
// symbol.
func saveKeyLabel() string {
	return saveKeyLabelFor(runtime.GOOS)
}

// saveKeyLabelFor is saveKeyLabel with the platform passed in, so that the width
// of a label this machine will never draw is still measurable here. The off-Mac
// label is the longer of the two, so a test that only ever saw the host's would
// be measuring the easy case on whichever platform it happened to run.
func saveKeyLabelFor(goos string) string {
	if goos == "darwin" {
		return saveKeyMacLabel
	}
	return saveKeyControl
}
