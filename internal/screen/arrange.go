package screen

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The arrange phase, and why it is a screen of its own
//
// It is the last decision of a draft and the only one that is about the **board**
// rather than about the pool, which is the whole of the argument: DraftScreen is a
// cursor over a list, this is a cursor over a 3x3, and the two share no key. A mode
// on that screen would also have quietly deleted an assertion this repository
// wanted kept — *no keystroke on the draft screen can take an arrangement* — which
// is what stops the arrange keys being reachable by accident from the wrong screen.
// TestNoKeystrokeOnTheDraftScreenTakesAnArrangement is that assertion, and it is in
// this package rather than in a client so that it holds wherever the screen is
// drawn.
//
// ⚠️ **It reuses DraftLive rather than declaring a third vocabulary.** Everything
// this screen needs is already on that reading — this side's picks in pick order,
// whether the phase is open, whether this reader is one of the two still being
// asked, the seats, the latest refusal — and a second shape mapped by the same
// client would be a second walk to hold total. What it deliberately does **not**
// carry is either side's cells: the phase is simultaneous and secret by design, so
// the opponent's arrangement is not on any reading a client can take, and this
// screen cannot draw it even by mistake. → socket.DraftSight.
//
// ⚠️ **The whole board is drawn and the reason is not symmetry.** Placement in this
// game is purely defensive — reach is counted in occupied ranks, so a unit's own
// column decides who is reached first and its row decides nothing — and the
// measured value of the decision is rank depth: the shipped roster's aces moved
// from the front column to the back read **27.6% → 47.3%** ally over 4000 seeds,
// and splitting the screening pair read **31.1%** against the adjacent pair's
// 47.3%. A player who cannot see which rank a cell is cannot take the decision
// those numbers are about, so the far half is drawn to say which end of the grid
// is met first, and every row carries RankLabel beside its coordinate.
//
// ⚠️ **There is no countdown here and that is a decision.** The phase's allowance
// is **not** one allowance for the phase: Server.settled re-arms off the reading
// after every batch, so whichever side arranges first hands its opponent a fresh
// full allowance and the worst case is about twice one. A single number on a screen
// with two decisions open is the wrong reading drawn in figures, and two readings
// of this were written into an earlier brief and both were wrong. So the fact is
// worded — i18n.ArrangeAllowance — and TestTheArrangeScreenWordsTheAllowanceRather
// ThanCountingItDown is what stops a countdown arriving later.

// The board's two marks. Both are **ASCII and exactly two bytes**, which is a
// requirement of the drawing rather than a style: hex.Render slices its label with
// len and text[:2], so a multi-byte rune would be cut in half and put broken UTF-8
// on the screen. → shapeAimMark, which is the same constraint one screen over.
const (
	// arrangeCursorMark is the cell enter would put the pick in hand on.
	arrangeCursorMark = "[]"
	// arrangeTheirsMark is a cell of the other half. It says the cell is theirs
	// and nothing about what stands on it, because nothing here knows: the phase
	// is secret and the reading carries no opponent's cells at all.
	arrangeTheirsMark = "::"
)

// ArrangeScreen is one side placing its drafted picks: the whole board, this
// side's picks in pick order, and the cell each has been put on.
//
// ⚠️ **The cells are filled in pick order and the pick in hand is the first
// without one**, which is what makes Slots a plain prefix rather than a slice with
// holes in it. hex.Offset's zero value is a **real cell** — draft.Squads carries
// the same warning — so a slot standing for "not placed yet" would be a legal
// looking answer, and an absence has to be declared rather than detected. What it
// costs is that a player who wants to move the second pick takes the third back
// first; what it buys is that there is no second state to get wrong.
type ArrangeScreen struct {
	// Live is the reading in hand, replaced whole on every Attach. It is the draft
	// screen's own reading — see the head of this file for why there is no second
	// one.
	Live DraftLive
	// Arranging says a reading has been handed over at all. Nought is a screen
	// nothing has attached to, which draws nothing rather than an empty formation.
	Arranging bool
	// Cursor is the cell enter would place on, in **authoring** coordinates: column
	// 0 is this side's own back column and column 2 the one that meets the other
	// half, which is what FormationSlots and hex.Place are both written in.
	//
	// ⚠️ **It never rests on a cell already taken**, which is the squad builder's
	// own answer to the same question: a chooser that stops on an occupied cell is
	// a chooser that can be left in a state the save has to refuse, so the movement
	// steps over one instead of letting a later refusal explain it.
	Cursor hex.Offset
	// Slots is the cell each pick stands on, in pick order — `Slots[i]` is the cell
	// for `Picks()[i]`, which is exactly what draft.Arrange takes.
	Slots []hex.Offset
	// Sent is this screen having handed its arrangement over, and it is **local
	// state that gates nothing**.
	//
	// ⚠️ **Nothing on any reading can say this, which is the finding rather than a
	// shortcut.** The record takes both arrangements in **one** step — the first to
	// arrive is deliberately not recorded, because appending it would be showing it
	// to the other player — so a client that has arranged sees exactly what it saw
	// before: its own seat still owed, and no message at all until the phase
	// closes. Without this the send is a keystroke after which nothing on the screen
	// changes, for up to a whole allowance, which reads as a program that has hung.
	//
	// ⚠️ **It is not the "already answered" memo step 5a refused, and the
	// difference is that it stops nothing.** That memo would have gated the send,
	// and the retry-on-refusal loop is the only thing that ever gets a decision the
	// room turned away through — so the keys stay live here, the footer goes on
	// naming them, and a refusal drawn beside this line is an arrangement to send
	// again. → i18n.ArrangeSent, and draw.DraftLive.Waiting, which is the same
	// gap one phase earlier.
	Sent bool
	// Err is what placement.Squad.Validate said about the arrangement as it stands,
	// and it is what blocks the send.
	//
	// ⚠️ **That function is the single legality rule and its words are kept.** It
	// already refuses an empty squad, more than five, a blank or duplicated id, a
	// cell off the 3x3 and two units on one cell, so nothing here restates any of
	// them — and a refusal drawn here is the answer the room would give rather than
	// a guess at it. Blocking rather than sending is the loadout's own reason: a
	// refused draft decision is left open and re-sent, so an illegal one is an
	// unbounded hot loop.
	Err error
}

// NewArrangeScreen is the screen before a reading has been handed to it.
func NewArrangeScreen() ArrangeScreen { return ArrangeScreen{} }

// Attach points this screen at a drafting room.
//
// It is called on every redraw, like every other Attach here, so it is cheap and
// idempotent. The one thing it resets is the arrangement itself, and only when the
// picks it was built for have changed: a cell belongs to the pick it was chosen
// for, so carrying cells into another set of picks would be an arrangement about
// somebody else.
func (a ArrangeScreen) Attach(_ Context, live DraftLive) ArrangeScreen {
	previous, attached := a.Picks(), a.Arranging
	a.Live = live
	a.Arranging = true
	if !attached {
		// ⚠️ **The opening cell is named rather than left to the zero value**, which
		// would be `0,0` — this side's own **back** corner. hex.Offset's zero is a
		// real cell, so a cursor that started there would be a decision nobody took
		// that happened to land on the rank the measurements say is worth the most.
		// FormationSlots is front first, which is the order a formation is read:
		// the rank that meets the other side, because that is the order an attack
		// arrives in.
		a.Cursor = FormationSlots()[0]
	}
	if !slices.Equal(pickIDs(previous), pickIDs(a.Picks())) {
		a.Slots, a.Err, a.Sent = nil, nil, false
	}
	if len(a.Slots) > len(a.Picks()) {
		a.Slots = a.Slots[:len(a.Picks())]
	}
	a.Cursor = a.settleCursor(a.Cursor)
	return a
}

// pickIDs is a side's picks as the ids that identify them, which is what says one
// set of picks is the set an arrangement was built for.
func pickIDs(picks []DraftPick) []string {
	out := make([]string, 0, len(picks))
	for _, pick := range picks {
		out = append(out, pick.Character)
	}
	return out
}

// Picks is this reader's own side's picks, in the order they were taken — which is
// the order draft.Arrange reads the cells in.
func (a ArrangeScreen) Picks() []DraftPick { return a.Live.picksAt(a.Live.mine()) }

// Placing reports whether there is an arrangement for this reader to take: the
// phase is open and this side has not answered it.
//
// ⚠️ **It is Live.Yours and never `OnTurn == You`.** The phase has two decisions
// open at once, so the reading reports no seat on turn and this reader may still be
// one of the two being asked — which is draft.Draft.Turn's own note about never
// widening for it. Yours goes false the moment this side's arrangement is recorded,
// which is what makes Sent below a state rather than a flag of this screen's own.
func (a ArrangeScreen) Placing() bool {
	return a.Arranging && a.Live.Arranging && a.Live.Yours
}

// Waiting reports that there is nothing left for this reader to do: either the
// arrangement has gone, or the phase has closed and the board has not arrived yet
// — which is the one frame between the record taking both arrangements and the
// wire.Start that follows it.
//
// It is what makes the screen say **why** it is waiting. The phase is simultaneous
// and secret by design, so a screen that only said it was waiting would read as a
// match that had stalled.
func (a ArrangeScreen) Waiting() bool { return a.Arranging && (a.Sent || !a.Placing()) }

// InHand is the pick waiting for a cell, and whether there is one.
func (a ArrangeScreen) InHand() (DraftPick, bool) {
	picks := a.Picks()
	if len(a.Slots) >= len(picks) {
		return DraftPick{}, false
	}
	return picks[len(a.Slots)], true
}

// Whole reports every pick having a cell, which is the one state the send is
// offered in.
func (a ArrangeScreen) Whole() bool {
	return len(a.Picks()) > 0 && len(a.Slots) == len(a.Picks())
}

// taken reports whether some pick already stands on a cell.
func (a ArrangeScreen) taken(cell hex.Offset) bool { return slices.Contains(a.Slots, cell) }

// Update reads one keystroke.
func (a ArrangeScreen) Update(_ Context, message tea.KeyPressMsg) (ArrangeScreen, DraftResult) {
	if message.String() == "esc" {
		// What leaving a drafting room costs is the client's to decide, exactly as
		// it is on the draft screen: there is no screen behind this one, because
		// the room it was joined from no longer exists.
		return a, DraftResult{Action: Action{Kind: Back}}
	}
	if !a.Placing() {
		// Every key below is about a decision this reader does not have, and the
		// waiting footer names none of them. → i18n.ArrangeWaitingFooter.
		return a, DraftResult{}
	}
	switch message.String() {
	case "up", "k":
		a.Cursor = a.stepped(a.Cursor, 0, -1)
	case "down", "j":
		a.Cursor = a.stepped(a.Cursor, 0, 1)
	case "left", "h":
		a.Cursor = a.stepped(a.Cursor, -1, 0)
	case "right", "l":
		a.Cursor = a.stepped(a.Cursor, 1, 0)
	case "u":
		// ⚠️ **`u` rather than backspace, and it is the battle screen's own key for
		// the same idea**: that screen undoes a turn with it, this one takes a
		// placement back, and one letter for one idea is the rule two vocabularies
		// for one thing keeps costing here. Backspace was written first and is what
		// a hand reaches for — it is not offered, because an alias no footer names
		// is a key nobody presses and a footer naming both spends the cells this
		// screen has not got.
		return a.takeBack(), DraftResult{}
	case "enter":
		if _, holding := a.InHand(); holding {
			return a.place(), DraftResult{}
		}
		return a.send()
	}
	return a, DraftResult{}
}

// place puts the pick in hand on the cell under the cursor and moves the cursor
// off it, so the next pick is never offered a cell that has just been taken.
func (a ArrangeScreen) place() ArrangeScreen {
	if _, holding := a.InHand(); !holding {
		return a
	}
	a.Slots = append(slices.Clone(a.Slots), a.Cursor)
	a.Cursor = a.settleCursor(a.Cursor)
	return a.judge()
}

// takeBack undoes the last placement and puts the cursor on the cell it freed,
// which is where a player who is changing their mind is looking.
//
// ⚠️ **Without it a misplaced unit could only be answered by letting the allowance
// run out**, and a draft that runs out of time is cancelled outright rather than
// resumed — so the cost of no undo here is the whole match. → draft.Draft.TimedOut.
func (a ArrangeScreen) takeBack() ArrangeScreen {
	if len(a.Slots) == 0 {
		return a
	}
	freed := a.Slots[len(a.Slots)-1]
	a.Slots = slices.Clone(a.Slots[:len(a.Slots)-1])
	a.Cursor = freed
	return a.judge()
}

// send hands the whole arrangement over, or refuses it in the words the room
// would. → Err, which is where the single-rule argument is.
func (a ArrangeScreen) send() (ArrangeScreen, DraftResult) {
	if !a.Whole() {
		return a, DraftResult{}
	}
	a = a.judge()
	if a.Err != nil {
		return a, DraftResult{}
	}
	a.Sent = true
	return a, DraftResult{Decided: true, Decision: DraftDecision{
		Step: DraftStepArrange,
		// Cloned, so the screen's own slice cannot be reached through the decision
		// — the same depth draft.Arrange hands out at.
		Slots: slices.Clone(a.Slots),
	}}
}

// judge asks the legality rule about the arrangement as it stands, and only once
// every pick has a cell.
//
// ⚠️ **A part-built arrangement is not judged, and that is not laziness.**
// placement.Squad.Validate is asked about a whole squad — its first line refuses
// one with nobody in it — so putting a half-placed side to it would draw a refusal
// about a state the player is halfway through creating, which reads as the program
// objecting to a keystroke it just accepted.
func (a ArrangeScreen) judge() ArrangeScreen {
	if !a.Whole() {
		a.Err = nil
		return a
	}
	a.Err = a.Squad().Validate()
	return a
}

// Squad is the arrangement as the value the legality rule is asked about, and it is
// the same shape draft.squadAt builds: the unit id is the character id, because a
// draft's pool is exclusive and every pick is therefore a different character.
//
// ⚠️ **The level is progression.LevelCap and is not read off the reading**, which
// is DraftPick's own decision: every drafted unit fights at the cap, so a level on
// the screen would be a number a reader wonders about — and Validate does ask for
// one.
func (a ArrangeScreen) Squad() placement.Squad {
	picks := a.Picks()
	units := make([]placement.Placement, 0, len(picks))
	for at, pick := range picks {
		unit := placement.Placement{
			ID:        pick.Character,
			Character: pick.Character,
			Level:     progression.LevelCap,
			Stage:     pick.Stage,
			Skills:    pick.Skills,
			Passives:  pick.Passives,
		}
		if at < len(a.Slots) {
			unit.Slot = a.Slots[at]
		}
		units = append(units, unit)
	}
	return placement.Squad{ID: a.Live.You, Units: units}
}

// settleCursor is a cell that is free, starting from the one asked for: the cell
// itself when nobody stands on it, and otherwise the first free one in the order a
// formation is read.
func (a ArrangeScreen) settleCursor(from hex.Offset) hex.Offset {
	slots := FormationSlots()
	if slices.Contains(slots, from) && !a.taken(from) {
		return from
	}
	for _, slot := range slots {
		if !a.taken(slot) {
			return slot
		}
	}
	// Every cell taken, which needs more picks than a side may field — Validate
	// refuses that squad by name. The cursor stays where it is rather than
	// answering a cell off the grid.
	return from
}

// stepped is the cursor moved one cell, **over** anything already standing there.
//
// ⚠️ **Stepping over is the squad builder's own answer and it is copied on
// purpose**: two units on one cell is not a formation, and a cursor that stops on
// one is a cursor that can be left in a state the send has to refuse. Wrapping in
// each axis rather than stopping at the edge is the same chooser's behaviour: a
// 3x3 is small enough that walking off one side and back on the other is quicker
// than turning round.
func (a ArrangeScreen) stepped(from hex.Offset, byCol, byRow int) hex.Offset {
	steps := hex.FormationRows
	if byCol != 0 {
		steps = hex.FormationCols
	}
	next := from
	for range steps {
		next = hex.Offset{
			Col: (next.Col + byCol + hex.FormationCols) % hex.FormationCols,
			Row: (next.Row + byRow + hex.FormationRows) % hex.FormationRows,
		}
		if !a.taken(next) {
			return next
		}
	}
	// A whole row or column taken, so there is nowhere to step to. The cursor
	// stays where it is, which is a free cell by construction.
	return from
}

// View draws the screen and its footer.
func (a ArrangeScreen) View(c Context) (string, string) {
	if !a.Arranging {
		return "", c.Text(i18n.ArrangeWaitingFooter)
	}
	lines := a.head(c, a.roomForWhy(c))
	lines = append(lines, strings.Split(a.board(c), "\n")...)
	lines = append(lines, "  "+c.Style.Dim.Render(c.Text(i18n.ArrangeLegend,
		arrangeCursorMark, arrangeTheirsMark)), "")
	lines = append(lines, a.rows(c, a.room(c, len(lines)))...)
	lines = append(lines, a.status(c)...)
	return strings.Join(lines, "\n"), a.footer(c)
}

// room is what the list of picks has left once everything else is drawn.
//
// ⚠️ **The board is never what gives way**, which is this screen's whole priority:
// it is ten rows at every window and a formation with the far half missing cannot
// say which rank a cell is, so the list of picks is what a short window takes from
// — the same answer poolRows gives, and the same floor of one row.
//
// The `- 4` is the two rows a client's header pair takes and the two the blank and
// the footer do. It is a **mirror** of the clients' frame rather than its
// declaration, exactly as every other Room helper here is.
func (a ArrangeScreen) room(c Context, spent int) int {
	room := c.Height - 4 - spent - len(a.status(c))
	if room < 1 {
		return 1
	}
	return room
}

// roomForWhy reports whether the window can hold the note saying why the whole
// board is drawn, **without taking a row off the list of picks**.
//
// ⚠️ **It is measured rather than guessed, and the floor is where it goes.** At
// 120x24 a frame leaves twenty rows and this screen spends every one of them on
// the heading, the notices, the board, the legend, the picks and the line saying
// what enter would do — so the note is what a taller window buys. It is dropped
// **silently**, unlike the battle screen's sections: what it carries is the
// reasoning behind the decision, while what the decision needs — which rank each
// cell is — is on every row and on the board at every window. A screen missing its
// board reads as broken; a screen missing an explanation reads as a screen.
func (a ArrangeScreen) roomForWhy(c Context) bool {
	spent := len(a.head(c, false)) + len(strings.Split(a.board(c), "\n")) +
		1 + 1 + len(a.Picks()) + len(a.status(c))
	return c.Height-4-spent >= len(WrapWords(c.Text(i18n.ArrangeDepth), MinWidth-3))
}

// head is the heading, the notices and whatever the room last refused — every line
// above the board.
func (a ArrangeScreen) head(c Context, why bool) []string {
	lines := []string{
		c.Style.Heading.Render(c.Text(i18n.ArrangeHeading)) + "  " +
			c.Style.Dim.Render(c.Text(i18n.ArrangePlaced, len(a.Slots), len(a.Picks()))),
		"",
	}
	prose := func(style lipgloss.Style, text string) {
		for _, line := range WrapWords(text, MinWidth-3) {
			lines = append(lines, "  "+style.Render(line))
		}
	}
	if a.Waiting() {
		prose(c.Style.Dim, c.Text(i18n.ArrangeSent))
	} else {
		prose(c.Style.Dim, c.Text(i18n.DraftArrangingNow))
	}
	// ⚠️ **Drawn in both states**, because the reading it corrects is available in
	// both: a player who has just sent is watching a screen that looks stopped, and
	// a player still placing is the one deciding how long to spend.
	prose(c.Style.Dim, c.Text(i18n.ArrangeAllowance))
	if why {
		// ⚠️ **The measured reason the far half is drawn at all**: placement here is
		// purely defensive and its whole value is rank depth — 27.6% → 47.3% ally
		// over 4000 seeds for moving a roster's aces to the back column, and 31.1%
		// against 47.3% for splitting the screening pair. → roomForWhy, for why it
		// is the first thing a short window gives up.
		prose(c.Style.Dim, c.Text(i18n.ArrangeDepth))
	}
	if a.Live.Refusal != "" {
		lines = append(lines, "  "+c.Style.Bad.Render(c.Text(i18n.DraftRefused)))
		prose(c.Style.Bad, c.Lang.Refusal(a.Live.Refusal))
	}
	if a.Err != nil {
		// Lang.Error is the house rule for a diagnostic that is not this program's
		// to word: the lead-in is the reader's language and internal/core's own
		// English follows it.
		prose(c.Style.Bad, c.Lang.Error(a.Err))
	}
	return append(lines, "")
}

// board is the whole battlefield with this side's picks on it, drawn through the
// one function that draws a formation anywhere in this repository.
//
// ⚠️ **This side is always the ally half**, whichever half the room ends up
// fielding it as. A formation is authored the same way by both sides — hex.Place
// maps the enemy's through a 180 degree rotation — so "your nine on the left,
// theirs on the right" is the arrangement a player already knows from the squad
// builder, and the coordinates drawn beside each row are the ones the decision
// travels in.
func (a ArrangeScreen) board(c Context) string {
	numbers := make(map[hex.Offset]string, len(a.Slots))
	for at, slot := range a.Slots {
		numbers[hex.Place(hex.SideAlly, slot)] = pickMark(at)
	}
	cursor := hex.Place(hex.SideAlly, a.Cursor)
	drawn := hex.Render(func(cell hex.Offset) string {
		if cell.Side() != hex.SideAlly {
			return arrangeTheirsMark
		}
		if mark, standing := numbers[cell]; standing {
			return mark
		}
		if a.Placing() && cell == cursor {
			return arrangeCursorMark
		}
		return ""
	})
	rows := strings.Split(drawn, "\n")
	for at, row := range rows {
		rows[at] = "  " + c.Style.Dim.Render(row)
	}
	return strings.Join(rows, "\n")
}

// pickMark is what one placed pick is drawn as on the board: its position in pick
// order, which is the number the row under the board carries.
//
// A digit is safe by construction — hex.MaxTeamSize is five — and it is a number
// rather than a letter of the id because two cells cannot hold a name and the ids
// in a drafted squad share no prefix worth cutting to.
func pickMark(at int) string { return string(rune('1' + at)) }

// rows is one line per pick: where it stands, and which rank that is.
//
// The rank is drawn beside the coordinate for slotLabel's own reason — the
// coordinate is what the decision travels in and the rank is the half that means
// something, because reach is counted in ranks and a coordinate does not say which
// end of the grid is met first.
func (a ArrangeScreen) rows(c Context, room int) []string {
	picks := a.Picks()
	if len(picks) == 0 {
		return []string{"  " + c.Style.Dim.Render(c.Text(i18n.DraftNoPicks))}
	}
	from, to := Window(len(picks), len(a.Slots), room)
	out := make([]string, 0, to-from)
	for at := from; at < to; at++ {
		marker := "  "
		if at == len(a.Slots) {
			marker = "> "
		}
		line := pickMark(at) + " " + Pad(picks[at].Character, draftIDWidth) + " " + a.cellLabel(c, at)
		if at == len(a.Slots) && a.Placing() {
			line = c.Style.Selected.Render(line)
		}
		out = append(out, marker+strings.TrimRight(line, " "))
	}
	return out
}

// cellLabel is where one pick stands, said as the cell and as the rank, or that it
// has no cell yet — an absence declared rather than a blank column.
func (a ArrangeScreen) cellLabel(c Context, at int) string {
	if at >= len(a.Slots) {
		return c.Style.Dim.Render(c.Text(i18n.ArrangeUnplaced))
	}
	slot := a.Slots[at]
	rank := RankLabel(c, slot)
	if rank == "" {
		return slot.String()
	}
	return slot.String() + "  " + c.Style.Dim.Render(rank)
}

// status is the line under the list: what enter would do, and where the cursor is.
//
// ⚠️ **One line, no blank over it, and both halves of that are the floor rather
// than taste.** At 120x24 a frame leaves this screen twenty rows and it spends
// nineteen of them before the picks: the heading and a blank, four rows of notice
// (the arranging sentence wraps to two in English), the blank the head ends on,
// ten rows of board and legend, and a blank. Three picks and this line come to
// exactly twenty. A second sentence, or a blank over this one, is a pick row the
// window then drops.
func (a ArrangeScreen) status(c Context) []string {
	if !a.Placing() {
		return nil
	}
	if pick, holding := a.InHand(); holding {
		return []string{"  " + c.Style.Emphasis.Render(c.Text(i18n.ArrangeInHand, pick.Character)) +
			c.Style.Dim.Render("  ·  "+c.Text(i18n.ArrangeCursorAt, a.Cursor.String(), RankLabel(c, a.Cursor)))}
	}
	if a.Err != nil {
		// ⚠️ **The line may not say enter sends when enter does not**, which is
		// sendValue's own rule one screen over: the reason is drawn above with every
		// other diagnostic on this screen, and a row that announced the send here
		// would be the program promising a keystroke it refuses.
		return []string{"  " + c.Style.Bad.Render(Ellipsis)}
	}
	return []string{"  " + c.Style.Emphasis.Render(c.Text(i18n.ArrangeSendReady))}
}

// footer is the keys this reading actually offers.
//
// ⚠️ **Three footers, and the waiting one is the load-bearing half.** A footer
// naming a key on a decision that has already been sent would be the program
// promising something it does not do, and here the promise is a whole match: a
// player who believes they are still arranging finds out when the battle opens
// with their squad somewhere else.
func (a ArrangeScreen) footer(c Context) string {
	switch {
	case !a.Placing():
		return c.Text(i18n.ArrangeWaitingFooter)
	case a.Whole():
		return c.Text(i18n.ArrangeSendFooter)
	}
	return c.Text(i18n.ArrangeFooter)
}
