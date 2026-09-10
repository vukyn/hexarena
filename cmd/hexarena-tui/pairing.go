package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/i18n"
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
// means *take two sides into a battle*, which is what a game is.
//
// # The chooser, and why it is this client's own
//
// Both halves are named now, one cursor each: `model.taking` is the side the
// reader commands and `model.against` is who it is put against, and
// `screenPairing` — `updatePairing` and `viewPairing`, below — is the screen
// that moves them. Before it, the away side was *the next row on the file,
// wrapping*, which was written down as a local stand-in for matchmaking and had
// the property that nobody could choose it.
//
// ⚠️ **It is not cmd/hexforge-tui's `fightScreen` moved somewhere shared, and
// that is a decision rather than duplication.** That screen answers a different
// question with the same two cursors: it *measures* a pairing — a seed count on
// the same ladder the spar uses, a rate, a record, both arrangements, and a
// squad against itself as its **control**. This one settles who turns up. One
// shape over both would have to carry the seeds and the rate into a client that
// has nothing to do with either, and would put a measurement screen's wording in
// front of somebody about to play one battle. What it does follow is that
// screen's one structural decision, and the comment there says why: **the two
// cursors are indices into the catalogue, not ids**, because the catalogue is
// what they walk and an id would need looking up on every keystroke to find out
// where in it a cursor is.
//
// ⚠️ **`m.against` is a standing answer and not a screen's field.** The squad
// catalogue's `f` raises a battle about the side under its cursor — it names a
// home side and says nothing at all about the other half — so the half it does
// not name has to be an answer the reader gave earlier and can give again. That
// is also why `f` still opens the battle directly rather than landing on the
// chooser: the way back this client keeps is two slots deep, and putting a
// screen between the catalogue and the battle makes the chain
// catalogue → chooser → battle → description, which is three pushes and loses
// the catalogue. → `model.raisedOver`, which says what the answer is on the day
// that is wanted.
//
// # ⚠️ This is the LOCAL pairing, and the network path does not go through it
//
// A PvP squad is chosen on the join screen, out of that screen's own chooser
// (`joinScreen.Squad` / `joinScreen.Chosen`), and `model.dialling` is what puts
// it in `wire.Hello.Squad`. Neither `taking` nor `against` is read anywhere on
// that path, and a keystroke on this screen cannot reach it.
//
// ⚠️ **An earlier version of this comment said the opposite** — *"`landSquad` /
// `m.taking` ... is now also what fills `wire.Hello.Squad` when a room is
// joined"* — and it was false when it was written or stopped being true shortly
// after: the join screen has carried its own chooser since it grew the **bring
// none** position, which a drafting room needs and which a row of the squad
// catalogue cannot express. Checked rather than reasoned about, because the
// claim decides whether a change here is a change to a match:
// `TestThePairingChooserDoesNotReachThePvPSquad` is what says so now.
//
// # ⚠️ What the rest of that comment used to claim, and why it was wrong
//
// It read: *"when the server arrives, what replaces it is this function and
// nothing else: `enter` still hands two squads to `Open`, `landSquad` still
// records which side the reader chose, and the battle screen never learns that
// the second one came off a socket."* The server has arrived — → session.go and
// lobby.go — and none of those clauses survived:
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
// The seam the server replaced was the **opponent**, not the pairing — and on
// this side of a match the opponent has no `placement.Squad` at all. So the
// hot-seat battle stays a thing this client answers on its own, which is the
// point rather than an accident: a client that plays both has to be able to play
// the local one unchanged.
//
// ⚠️ **What this may not become is a second reading of the same question.** The
// authoring tool's fight keeps a cache keyed on `home|away|seeds` because it runs
// thousands of battles; this runs one, on the way in, and the pairing is read
// where the battle is built. A client that read it again while drawing would be a
// redraw deciding who is fighting.

// awayOpensOn is the row the other half of the pairing starts on.
//
// ⚠️ **The second one rather than the first**, and the reason is what a reader
// who has chosen nothing gets: nought on both cursors is a side against a copy
// of itself for anybody with more than one side saved, which is a pairing this
// client would have started offering without anybody deciding it should. One
// row along is what the away side has always been for a reader arriving at the
// first row of the catalogue, so a chooser nobody has touched opens on the
// battle this client has always opened on. It is clamped where it is read, so a
// catalogue holding one side lands back on it.
const awayOpensOn = 1

// pairing is the two squads the battle screen is opened on: the side the reader
// is taking in, and the side it is being put against.
//
// Both come back empty when the catalogue is empty, which is not a sentinel and
// not an error: `Open` reads a squad with nobody in it as *no pairing* and says
// on screen that a side has to be built, which is the honest thing for a client
// whose catalogue is written by the other front-end. Branching here would put
// that refusal in two places — and the chooser above does not phrase one either,
// for the same reason: it draws no rows when there are none, and the reader who
// presses on lands on the one sentence there is.
func (m model) pairing() (home, away placement.Squad) {
	saved := m.squads.Saved
	if len(saved) == 0 {
		return placement.Squad{}, placement.Squad{}
	}
	// Clamped rather than trusted, and both of them: a cursor is a row, the
	// catalogue is re-read on the way in, and a side deleted by the authoring
	// tool between one visit and the next would otherwise index past the end.
	// This is the one place either cursor is turned into a squad, so it is the
	// one place that can be sure the list it is indexing is the list in hand.
	return saved[draw.Clamp(m.taking, 0, len(saved)-1)],
		saved[draw.Clamp(m.against, 0, len(saved)-1)]
}

// updatePairing routes one keystroke on the chooser.
//
// The two cursors take different keys — up and down for the side being
// commanded, left and right for the one across the board — which is the
// arrangement cmd/hexforge-tui's fight uses and the reason it can offer two
// choosers on one screen at all.
func (m model) updatePairing(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := len(m.squads.Saved)
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		return m.goBack(), nil
	case "up":
		m.taking = pairingStep(m.taking, -1, rows)
	case "down":
		m.taking = pairingStep(m.taking, 1, rows)
	case "left":
		m.against = pairingStep(m.against, -1, rows)
	case "right":
		m.against = pairingStep(m.against, 1, rows)
	case "enter":
		// The way back is recorded here because this is the raiser: a screen may
		// not name its own door, and the battle's esc is a draw.Back.
		//
		// ⚠️ **An empty catalogue goes through as well.** What is on the other
		// side is `Open` reading two squads with nobody in them as no pairing and
		// saying a side has to be built, which is the only place that sentence
		// lives; a guard here would be the second one.
		return m.raisedBy(screenPairing).enter(screenBattle), nil
	}
	return m, nil
}

// pairingStep moves one cursor a row and wraps, which is what a chooser drawn
// between arrows promises.
//
// It clamps before it steps rather than after: a cursor left past the end by a
// side the authoring tool deleted has to come back onto the list before it can
// be moved along it, or the first arrow key after a deletion lands somewhere
// nobody pressed for.
func pairingStep(at, by, rows int) int {
	if rows == 0 {
		return at
	}
	return (draw.Clamp(at, 0, rows-1) + by + rows) % rows
}

// viewPairing draws the two choosers and the footer.
func (m model) viewPairing() (string, string) {
	c := m.ctx()
	footer := c.Text(i18n.PairingFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.PairingHeading)) + "\n\n")
	// Wrapped against the floor rather than against the window in hand, which is
	// the prose half of this repository's width rule: a sentence gets the same
	// shape on every terminal, a data cell takes the room there is.
	for _, line := range draw.WrapWords(c.Text(i18n.PairingHint), draw.MinWidth-3) {
		out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
	}
	if len(m.squads.Saved) == 0 {
		// ⚠️ **No rows and no sentence about there being none.** A chooser over an
		// empty catalogue chooses nothing, and what that means for the reader —
		// that a side has to be built — is `Open`'s to say, one keystroke away.
		// Saying it here as well would be the refusal in two places, which is the
		// thing pairing above is written not to do.
		return strings.TrimRight(out.String(), "\n"), footer
	}
	home, away := m.pairing()
	out.WriteString("\n")
	width := pairingLabelWidth(c)
	out.WriteString("  " + draw.Pad(c.Text(i18n.PairingHome), width) + " " +
		pairingChoice(c, home) + "\n")
	out.WriteString("  " + draw.Pad(c.Text(i18n.PairingAway), width) + " " +
		pairingChoice(c, away) + "\n")
	if home.ID == away.ID {
		out.WriteString("\n")
		for _, line := range draw.WrapWords(c.Text(i18n.PairingSameSide), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
		}
	}
	return strings.TrimRight(out.String(), "\n"), footer
}

// pairingChoice is one chooser row: the side name between arrows, and its id
// beside it. Both are free text an author wrote, so neither is measured.
//
// Whose side it is goes beside the id for the reason the join screen's chooser
// says: a player's own side wins the id it shares with a shipped one
// (forge.SquadsOffered), so a row showing the id alone would offer two different
// squads under one spelling with nothing to tell them apart.
func pairingChoice(c draw.Context, squad placement.Squad) string {
	id := squad.ID
	if c.PlayerSquad(squad.ID) {
		id += " " + c.Text(i18n.SquadMine)
	}
	return fmt.Sprintf(draw.ChoiceFormat, squad.Name, c.Style.Dim.Render(id))
}

// pairingLabelWidth is the column the two labels sit in, measured over the
// language in front for the reason every other label column here is measured:
// one number for two languages is only right for both by luck.
func pairingLabelWidth(c draw.Context) int {
	widest := 0
	for _, key := range []i18n.Key{i18n.PairingHome, i18n.PairingAway} {
		if width := lipgloss.Width(c.Text(key)); width > widest {
			widest = width
		}
	}
	return widest
}
