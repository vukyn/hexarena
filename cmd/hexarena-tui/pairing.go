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
// ⚠️ **This is the LOCAL pairing, and the network path does not go through it.**
// The away side here is the next side on the file, wrapping, so a catalogue
// holding one side fights it against a copy of itself — the pairing every
// fixture in internal/screen already opens on and the one the authoring tool's
// fight calls its control.
//
// # ⚠️ What this comment used to claim, and why three quarters of it was wrong
//
// It read: *"when the server arrives, what replaces it is this function and
// nothing else: `enter` still hands two squads to `Open`, `landSquad` still
// records which side the reader chose, and the battle screen never learns that
// the second one came off a socket."* The server has arrived — → session.go and
// lobby.go — and three of those four clauses are false:
//
//   - **The away side is never a `placement.Squad` on this client.** It arrives
//     already resolved as `[]battle.Roster` on a `wire.Start`, because
//     `Squad.Take` is the **server's** call at the gate — that is what makes a
//     squad checkable rather than trusted. This function's signature cannot
//     express it and should not try to.
//   - **`enter` does not hand two squads to `Open` on the PvP path.** A live
//     battle is `Attach`ed, not `Open`ed, and `Open` refuses outright while a
//     screen is live.
//   - **The battle screen does learn the difference**, in one field:
//     `draw.PlayScreen.Live`. It has to — a live screen may not step the battle,
//     because the mirror steps it from the turn that comes back.
//
// What survives, and is what this file is still for: `landSquad` / `m.taking`
// still records which side the reader chose, and that answer is now also what
// fills `wire.Hello.Squad` when a room is joined. The seam the server replaced
// was the **opponent**, not the pairing — and on this side of a match the
// opponent has no `placement.Squad` at all.
//
// So `pairing` stays exactly as it is and keeps the hot-seat battle exactly as
// it was. That is the point rather than an accident: a client that plays both
// has to be able to play the local one unchanged.
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
