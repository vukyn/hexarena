package main

import (
	"github.com/vukyn/hexarena/internal/core/placement"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// # Which two squads a battle opens on, and why this file is the seam
//
// `draw.PlayScreen.Open` wants exactly two `placement.Squad` values and asks
// nobody where they came from — that is the whole of its contract with a client,
// and it is what #228 made it: the screen used to reach sideways into the
// authoring tool's fight for its pairing, which is precisely the thing that made
// it a screen only one client could draw.
//
// So a client owes it a pairing, and the two clients owe it different ones. In
// cmd/hexforge-tui a `draw.Fight` means *pick a second squad and measure the
// pairing* — two choosers, both arrangements, a win rate with a control. Here it
// means *take this side into a battle*, which is what a game is, and there is
// exactly one side named: the one the reader was pointing at.
//
// ⚠️ **The opponent is where the network work lands, and it is stubbed here
// rather than invented.** `README.md` § *PvP over a LAN* says what the real
// answer is — two people on a LAN, each bringing a squad they saved on their own
// machine, and a **server** that pairs them and resolves the battle — so the away
// side is not this client's to decide for long. Until a server sends one, this
// client picks the next side on the file, and a catalogue holding one side fights
// it against a copy of itself, which is the pairing every fixture in
// internal/screen already opens on and the one the authoring tool's fight calls
// its control.
//
// ⚠️ **`pairing` is the whole seam and it is one function on purpose.** When the
// server arrives, what replaces it is this function and nothing else: `enter`
// still hands two squads to `Open`, `landSquad` still records which side the
// reader chose, and the battle screen never learns that the second one came off
// a socket rather than off a file. A pairing read at three call sites would be
// three places to change and two of them would be found later.
//
// ⚠️ **What it may not become is a second reading of the same question.** The
// authoring tool's fight keeps a cache keyed on `home|away|seeds` because it runs
// thousands of battles; this runs one, on the way in, and the pairing is read
// where the battle is built. A client that read it again while drawing would be a
// redraw deciding who is fighting.

// pairing is the two squads the battle screen is opened on: the side the reader
// is taking in, and whoever it is being put against.
//
// Both come back empty when the catalogue is empty, which is not a sentinel and
// not an error: `Open` reads a squad with nobody in it as *no pairing* and says
// on screen that a side has to be built, which is the honest thing for a client
// whose catalogue is written by the other front-end. Branching here would put
// that refusal in two places.
func (m model) pairing() (home, away placement.Squad) {
	saved := m.squads.Saved
	if len(saved) == 0 {
		return placement.Squad{}, placement.Squad{}
	}
	// Clamped rather than trusted: `taking` is a row and the catalogue is
	// re-read on the way in, so a side deleted by the authoring tool between one
	// visit and the next would otherwise index past the end.
	at := draw.Clamp(m.taking, 0, len(saved)-1)
	// ⚠️ **The next side on the file, wrapping**, which is a local stand-in for
	// matchmaking and is written down as one. With one side saved that is the same
	// row, so the battle is a side against a copy of itself — and a copy is a real
	// opponent rather than a degenerate one, because `placement.Squad.Take`
	// prefixes the unit ids with the side it is fielded as, so the two halves are
	// units a log can tell apart.
	against := saved[(at+1)%len(saved)]
	return saved[at], against
}
