package main

import (
	tea "charm.land/bubbletea/v2"

	draw "github.com/vukyn/hexarena/internal/screen"
)

// The keystrokes that write the work in front of the author live in
// internal/screen now, because the skill form went there and a second
// full-screen client will have forms of its own — three screens matching on
// their own spelling of one intent is three chances for one of them to be
// missed, and the package boundary is not a reason to make it four.
//
// The two forwarders below are here on the rule this package already follows for
// pad, clip, clamp and window: the character form and the origins form have not
// moved and still ask, the call sites read unchanged, and there is still exactly
// one body.

// isSaveKey reports whether a keystroke asks for the work to be written.
func isSaveKey(message tea.KeyPressMsg) bool { return draw.IsSaveKey(message) }

// saveKeyLabel is what a footer calls the save key.
func saveKeyLabel() string { return draw.SaveKeyLabel() }
