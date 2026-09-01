package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The keystrokes that save, and the label a footer names them with. What is not
// here is where a save *lands* — three forms across two packages ask IsSaveKey,
// and driving each of them to a written file is cmd/hexforge-tui's
// TestTheCommandKeySavesWhereverControlSDoes.

// TestOnlyTheTwoSaveKeysSave guards the other side of isSaveKey: a keystroke
// that merely looks like one must not write the author's work.
//
// The bare letter is the case worth naming. A save key is the letter plus a
// modifier, and bubbletea v2 leaves Text empty whenever a modifier is held — so
// "s" typed into a field carries Text and no Mod, and must reach the field
// rather than the book.
func TestOnlyTheTwoSaveKeysSave(t *testing.T) {
	saves := []tea.KeyPressMsg{
		{Code: 's', Mod: tea.ModCtrl},
		{Code: 's', Mod: tea.ModSuper},
	}
	for _, press := range saves {
		if !IsSaveKey(press) {
			t.Errorf("%q does not save", press.String())
		}
	}
	others := []tea.KeyPressMsg{
		{Code: 's', Text: "s"},
		{Code: 's', Mod: tea.ModAlt},
		{Code: 'd', Mod: tea.ModCtrl},
		{Code: tea.KeyEnter},
		{Code: tea.KeyEscape},
	}
	for _, press := range others {
		if IsSaveKey(press) {
			t.Errorf("%q saves, and should not", press.String())
		}
	}
}

// TestTheSaveLabelAlwaysOffersTheKeyThatAlwaysWorks is the honesty check on the
// footer.
//
// ⌘S depends on the terminal speaking the Kitty keyboard protocol and on it not
// claiming the chord for its own Save dialog first — Terminal.app does exactly
// that. So whatever platform the footer is drawn on, it has to keep naming a
// control-S, because that is the keystroke this program can actually promise.
func TestTheSaveLabelAlwaysOffersTheKeyThatAlwaysWorks(t *testing.T) {
	label := SaveKeyLabel()
	if !strings.Contains(label, "^S") && !strings.Contains(label, saveKeyControl) {
		t.Errorf("the save label %q names no control-S, which is the key that always works", label)
	}
}

// TestTheSaveLabelIsDrawableEverywhere is the rendering half of it, and the
// reason the footer stopped naming ⌘S.
//
// ⌘ is East-Asian-Ambiguous width — measured as one cell, drawn as two by a good
// many terminals, which lands the glyph on top of the character after it. On
// those terminals "⌘S" is two overlapping characters rather than a key, and
// nothing inside the program can find out which sort of terminal is in front. So
// the label stays inside the characters every terminal draws at the width they
// were measured at.
//
// The assertion is on every letter rather than on ⌘ alone, because the next
// tempting symbol has exactly the same problem: ⌃, ⇧ and ⌥ are ambiguous too.
func TestTheSaveLabelIsDrawableEverywhere(t *testing.T) {
	for _, letter := range SaveKeyLabel() {
		if letter > 127 {
			t.Errorf("the save label %q carries %q, whose drawn width the program cannot know",
				SaveKeyLabel(), letter)
		}
	}
}
