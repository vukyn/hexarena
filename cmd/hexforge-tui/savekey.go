package main

import (
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

// saveKeyLabel is what a footer calls the save key.
//
// It is ctrl+s on every platform, including the one where ⌘S also works, and
// that is a rendering decision rather than a change of heart about ⌘S. ⌘ is
// East-Asian-Ambiguous width: lipgloss measures it as one cell, a good many
// terminals draw it as two, and the glyph then lands on top of whatever follows
// it — "⌘S" is drawn as the two characters overlapping, which is worse than not
// naming the key at all. Nothing inside a program can find out which of the two
// its terminal will do.
//
// Spacing it apart does not help, because the extra cell has to come from
// somewhere: the English character-form footer is 73 cells without the label,
// the smallest window is 80, and the last cell of a row is left empty so that
// writing it cannot wrap the line. Six cells for the label, and any spelling
// that names both keys in ASCII needs more than six.
//
// So the footer names the keystroke it can always deliver and always draw, and
// ⌘S is a Mac habit that turns out to work — MenuNote says so on the way in,
// and the README says why it might not. What the footer must never do is
// promise ⌘S alone, which is the one thing Terminal.app cannot keep.
func saveKeyLabel() string {
	return saveKeyControl
}
