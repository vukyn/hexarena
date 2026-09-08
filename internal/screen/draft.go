package screen

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The ban and pick, and why its vocabulary is declared twice
//
// ⚠️ **This package cannot take a socket.DraftSight and never will.** That value
// holds a wire.Seat, a wire.DraftStep and a draft.Pick — and internal/draft
// imports internal/wire, for Format and Seat, so pulling either package in here
// would drag the protocol into the package two clients share. It is the same
// constraint that keeps the lobby's three screens in cmd/hexarena-tui, and
// socket.DraftSight's own comment states it from the other side.
//
// So this screen declares **DraftLive**, the client maps a reading into it, and
// the arrangement is PlayLive's exactly: PlayScreen.Attach takes a value the
// client built inside socket.Mirror.Read, and PlayLive.Refusal is a *name*
// rather than a wire.Code for the same reason i18n.Lang.Refusal(string) exists.
//
// Three options were weighed and the cost of each is worth having written down,
// because the third one changes the answer the day it happens.
//
//  1. **Here, with a vocabulary of its own** — what is written. The cost is a
//     second spelling of the step, and there is no compiler trick that would
//     catch a missing arm: a keyed literal gives no error, which step 3 measured
//     on skillFile. So it is held by a **walk** instead —
//     cmd/hexarena-tui's mapper is proved to cover every wire.DraftSteps() entry
//     and every value below DraftStepCount, in both directions, so a sixth step
//     on the protocol is a red test rather than a screen that quietly draws
//     nothing. Both wire.DraftStep and wire.Seat are named **string** types, so
//     each conversion is one line; what costs is the declaration.
//  2. **In cmd/hexarena-tui, beside the lobby's three.** Refused. That comment's
//     own ground for accepting one golden instead of two is that *"there is no
//     data column and no drawing on any of the three — a heading, some prose,
//     two fields and a code"*, and a draft screen is a pool of nineteen
//     characters with a marked column, which is precisely a data column. CLAUDE.md
//     records measuring **twice** that a one-cell column widening is invisible to
//     one of the two goldens and caught by the other.
//  3. **Move Format, Seat and DraftStep down into a package both could import.**
//     Deferred at step 3 and still deferred — six packages, and a refactor of six
//     may not ride inside a screen. ⚠️ **But it would not, on its own, delete the
//     second vocabulary, and the plan this arrived under said it would.** Moving
//     those three (and DraftEntry and DraftDecision with them, which is the same
//     move) would let a DraftLive carry a draft.Pick and the moved Seat — it
//     would **not** let this screen take a socket.DraftSight, because that value
//     is declared in internal/socket, which imports internal/wire for a dozen
//     other reasons. What would delete the cost is that plus moving the
//     *reading* — DraftSight, DraftPrompt, DraftDue — down into internal/draft,
//     which already owns Picks, Candidates and Squads and would then be
//     importable here. That is the change to write down, and it is two changes
//     rather than one.
//
// # What this screen does not decide
//
// The state machine is internal/draft's and nothing here restates it. Whose
// decision is due, what the pool has left and every refusal about it arrive on
// the reading; what this file owns is a cursor, a loadout under construction and
// the drawing.
//
// The one rule it does ask is **cast.ChooseLoadout**, through cast.ChooseFrom,
// and it asks the same call draft.Draft.Loadout does — which is what makes the
// refusal drawn as a kit is chosen the same answer the room would give. ⚠️ It
// also **blocks the send** rather than letting an illegal loadout go out, and
// that is not politeness: a refused draft decision leaves the decision open, so
// a client that sent one would re-send it, and the retry-on-refusal loop is an
// unbounded hot loop for a decision that is genuinely illegal rather than merely
// early. → TODO.md § step 5a, finding (3).

// DraftStep is which kind of decision a draft is asking for, in this package's
// own vocabulary.
//
// ⚠️ **It is an enum and not a name, and that is the one place this file departs
// from PlayLive.Refusal's precedent on purpose.** A refusal is only ever
// *worded* — it crosses as a string and goes straight to i18n.Lang.Refusal — and
// so does a seat here, for the same reason. A step is **branched on**: a ban, a
// pick and a loadout draw different lines, offer different keys and name
// different footers. Branching on an unchecked string would mean writing the
// protocol's own spellings out as literals in this package, where nothing would
// notice them drifting: "ban" is one short word with no space in it, so
// TestNoScreenHoldsItsOwnWording would pass on all five.
//
// The zero value is nothing due, which is the safe half a forgotten declaration
// falls into: a screen that quietly went to DraftStepNone draws no decision,
// where one that quietly went to DraftStepBan would offer a ban nobody asked for.
type DraftStep uint8

const (
	// DraftStepNone is no decision due: between decisions, before the first, on
	// a draft that is over and during the arrange phase, which is asked about
	// through DraftLive.Arranging because two decisions are open at once.
	DraftStepNone DraftStep = iota
	// DraftStepBan is a side taking a character out of the pool for the match, and
	// it is optional — a ban that names nobody is a slot spent on nobody, which
	// is the skip.
	DraftStepBan
	// DraftStepPick is a side taking a character for itself, the first of the two
	// decisions a pick is made of.
	DraftStepPick
	// DraftStepLoadout is the second: the form, the skills and the trait the
	// character just picked will field.
	DraftStepLoadout
	// DraftStepArrange is a side putting its picks on its own formation. This screen
	// draws that the phase is **open** and nothing else about it; the formation
	// is the arrange screen's.
	DraftStepArrange
	// DraftStepTimeout is nobody's decision: an allowance having run out, which
	// cancels the whole draft. It is in the vocabulary because the protocol's is,
	// and a mapper that had nowhere to put it would be a mapper with a hole.
	DraftStepTimeout
)

// DraftStepCount is how many DraftStep values are declared, DraftStepNone
// included.
//
// ⚠️ It exists so a client can prove its mapping is **total in both
// directions** — every step the protocol declares reaches one of these, and
// every one of these is reached by some step. A one-way walk would pass on a
// value nothing ever produces, which is a screen arm that can never be drawn,
// and this repository has recorded the mirror-image failure (a target with no
// screen) five times.
const DraftStepCount = int(DraftStepTimeout) + 1

var draftStepNames = [DraftStepCount]string{
	DraftStepNone:    "none",
	DraftStepBan:     "ban",
	DraftStepPick:    "pick",
	DraftStepLoadout: "loadout",
	DraftStepArrange: "arrange",
	DraftStepTimeout: "timeout",
}

// String names a DraftStep for a diagnostic, and nothing draws it — the same
// shape, and the same reason, as Kind.String and Target.String.
func (s DraftStep) String() string {
	if int(s) >= DraftStepCount {
		return fmt.Sprintf("draftstep(%d)", uint8(s))
	}
	return draftStepNames[s]
}

// DraftPick is one side's pick as this screen draws it: the character, the form
// it will field and the kit it will bring.
//
// It carries no level, unlike draft.Pick, and that is deliberate rather than an
// omission: every drafted unit fights at progression.LevelCap and nothing on
// this screen draws a level, so a field holding one would be a number a reader
// wonders about. → draft.Pick.Level, which is where the constant is argued for.
type DraftPick struct {
	Character string
	// Stage is the form this pick fields, **as resolved** — empty until the
	// loadout has been taken.
	Stage string
	// Skills and Passives are the loadout. Both are empty while the pick's
	// second decision is still open, which is a state the reading reports rather
	// than one to detect: a pick with no skills is a character out of the pool
	// whose loadout has not been chosen.
	Skills   []string
	Passives []string
}

// Settled reports whether this pick's loadout has been taken.
func (p DraftPick) Settled() bool { return len(p.Skills) > 0 }

// DraftLive is a drafting room as one reading of it, and it is the draft's twin
// of PlayLive.
//
// ⚠️ **It is a value taken under the mirror's read lock and it does not outlive
// the call.** Unlike PlayLive it holds no pointer at all — a draft's state is the
// pool, the picks and whose decision is due, which is exactly what this carries,
// so there is no computation a renderer has to be handed an engine for. →
// socket.DraftSight, which says why Sight.Fight beside it is the one deliberate
// exception.
type DraftLive struct {
	// Pool is every character the draft may ban and pick, **in the cast book's
	// own declaration order**.
	//
	// ⚠️ **Do not sort it, and do not sort it here either.** draft.NewPool's
	// comment carries the argument: five other listings in this repository are in
	// declaration order — the browser, the builds screen, `hexforge list`, the
	// squad builder's chooser and the restriction picker — so a draft screen that
	// sorted would lay the cast out differently from every screen the player has
	// just come from. ⚠️ The shipped cast happens to be in id order today, so
	// sorting it would be a no-op and a test taken against the shipped books
	// would measure nothing.
	//
	// It is **handed in** rather than read off Context.Lib, and that is the same
	// decision PlayScreen.Home/Away is: the pool is a parameter of the draft. The
	// client builds it with draft.NewPool over the **embedded** cast — the one the
	// data digest at the gate promises both peers share — where Context.Lib is
	// whatever --data pointed at, which may differ and would then offer a
	// character the room refuses.
	Pool []cast.Character
	// Candidates is what the **open** decision may choose from, by id, in the
	// pool's own order — draft.Draft.Candidates' answer, unchanged.
	//
	// ⚠️ It is what may be chosen and **not** what is left in the pool. A loadout
	// chooses a form and a kit rather than a character, so this is empty during
	// one while nineteen characters are still nobody's; what is left is derived
	// from Bans and Picks instead, which is right in every state.
	Candidates []string
	// Picks and Bans are both sides' decisions, indexed by Seats.
	Picks [2][]DraftPick
	// Bans is the characters each side took out, in the order they were taken. A
	// **skipped** ban is nowhere in it, because a skip names nobody. →
	// draft.Draft.Bans.
	Bans [2][]string
	// Seats are the two seats in the order Picks and Bans are indexed, and the
	// order is **handed in rather than declared here**: which of the two the room
	// hands out first is the protocol's business, and a screen re-deriving it
	// would be a fourth copy of three lines internal/draft, internal/room and
	// internal/socket each already keep for their own stated reasons.
	//
	// They are names for the reason PlayLive.Refusal is one: a seat is only ever
	// worded, through i18n.Lang.Seat, and never branched on.
	Seats [2]string
	// You is this reader's own seat and OnTurn the seat whose decision is due,
	// empty when none is.
	//
	// ⚠️ **A You that names neither seat reads as neither side's**, and that is
	// the safe half rather than an accident: every mark on the pool then says the
	// *opponent* took it. The other way round, a forgotten seat would tell a
	// player they had banned characters they never touched.
	You    string
	OnTurn string
	// Step is which kind of decision is due, DraftStepNone when none is.
	Step DraftStep
	// Yours is the open decision being **this** reader's to take, which is
	// socket.DraftSight.Asked exactly.
	//
	// ⚠️ **It is not `OnTurn == You` and must not be replaced by it.** The arrange
	// phase has two decisions open at once, so the reading reports no seat on turn
	// and this reader may still be one of the two being asked — which is
	// draft.Draft.Turn's own note about never widening for it.
	Yours bool
	// Subject is the character a loadout is owed for: the pick one decision
	// earlier, which the reading derives rather than storing. Empty on every
	// other step.
	Subject string
	// Recorded is how many decisions the draft has recorded, and it is the whole
	// of how this screen knows a draft has begun at all. → Waiting, and
	// socket.DraftDue.Recorded, which is the same number used as an answer's
	// routing key.
	Recorded int
	// Units and BanSlots are the format's two numbers: how many characters a
	// side picks, and how many it may ban.
	//
	// ⚠️ **Handed in, from draft.PicksPerSide and draft.BansPerSide, and never
	// worked out here.** Those are the single declarations of both, and the
	// arithmetic over them has moved three times in two days — a screen carrying
	// its own copy would be a number that silently stopped being right. Note what
	// is *not* here: how many candidates the last pick sees. That is
	// `draft.Slack + 1` and this screen never needs it, because "there is only
	// one left" is a fact about the list in hand. → OnlyOne.
	Units    int
	BanSlots int
	// Arranging is the arrange phase being open, and Cancelled the draft having
	// been abandoned — which today happens for exactly one reason, an allowance
	// running out.
	Arranging bool
	Cancelled bool
	// Refusal is the **name** of the latest protocol refusal this client has been
	// sent, empty when there has been none. → PlayScreen.LiveRefusal for why it
	// is a name and never a typed value.
	Refusal string
	// Clock is what is left of the open decision's allowance on both sides,
	// **handed in already counted**.
	//
	// ⚠️ **It is PlayClock and not a second countdown type.** A draft's allowance
	// and a battle's are one number the room carries and one transport counts
	// down, and PlayClock's whole argument — that this package reads no clock and
	// is handed two counts of seconds — is about the boundary rather than about a
	// battle. A DraftClock would be a second declaration of the same idea and a
	// second pair of wordings for one row.
	Clock PlayClock
}

// side is where a seat's own decisions are indexed, and -1 for anything that is
// not one of the two — including the empty seat, which means "nobody" and must
// not quietly mean the first one.
func (l DraftLive) side(seat string) int {
	if seat == "" {
		return -1
	}
	for index, candidate := range l.Seats {
		if candidate == seat {
			return index
		}
	}
	return -1
}

// mine and theirs are this reader's own side and the other one, each -1 when the
// reading names no seat for it.
func (l DraftLive) mine() int { return l.side(l.You) }

func (l DraftLive) theirs() int {
	switch l.mine() {
	case 0:
		return 1
	case 1:
		return 0
	}
	return -1
}

// picksAt is one side's picks, and nothing for a side that is not one of the two.
func (l DraftLive) picksAt(index int) []DraftPick {
	if index < 0 || index >= len(l.Picks) {
		return nil
	}
	return l.Picks[index]
}

func (l DraftLive) bansAt(index int) []string {
	if index < 0 || index >= len(l.Bans) {
		return nil
	}
	return l.Bans[index]
}

// Waiting reports that the draft has recorded nothing, which is the state a
// player has to be told about.
//
// ⚠️ **It cannot mean "the room is not full", and that is the finding rather
// than a weakness of this reading.** A room that drafts sends **nothing at all**
// when its second seat is taken — internal/room's bothTaken answers no message,
// and an empty wire.Drafted is refused by design — so from a client's side a
// draft with an empty record is either a room still waiting for its second
// player or a room that filled a moment ago and has been asked nothing, and the
// two are indistinguishable on the wire.
//
// What is *not* indistinguishable is what a decision costs in each. Before the
// second seat is taken the room refuses a decision (draftOpen asks both seats
// taken) and the refusal leaves the decision open, so the client re-sends — five
// refusals were measured before the second client arrived — and what a real
// player, whose chooser blocks on a keystroke rather than spinning, gets instead
// is a ban that was **thrown away**. So this is drawn whenever the record is
// empty and the wording says what is true of both readings. → i18n.DraftNotBegun,
// and TODO.md § step 5a, which carries the protocol gap and the two candidate
// fixes for it, neither of which is a client fix.
func (l DraftLive) Waiting() bool {
	return l.Recorded == 0 && !l.Cancelled && !l.Arranging
}

// OnlyOne is the character the open decision has no choice about, and whether
// there is one.
//
// ⚠️ **The rule is the list in hand and never a figure.** With every ban spent
// the final pick of a draft sees exactly `draft.Slack + 1` candidates, and that
// expression has answered 1, 2 and 4 within two days as the pool moved from
// sixteen characters to nineteen — so a screen holding the count, or a test
// asserting it, is a number that stops being true without anything saying so.
// Asking whether the list has one entry is the same question with nothing
// written down, and it is right for every step and every format.
//
// A draft whose decision has one candidate must **say so** rather than present a
// list of one: the two look identical on screen and only one of them is a
// choice.
func (l DraftLive) OnlyOne() (string, bool) {
	if len(l.Candidates) != 1 {
		return "", false
	}
	return l.Candidates[0], true
}

// DraftPickInto is where one of the loadout's two lists lands, in the shape
// SquadsPick already has: a destination names a field of *this* screen, so it is
// declared beside it. → PickState.Into.
type DraftPickInto uint8

const (
	// DraftPickNothing is the zero value: a destination that names no field.
	DraftPickNothing DraftPickInto = iota
	// DraftPickKit is the skills a drafted unit brings, and DraftPickTrait the
	// one trait it holds.
	DraftPickKit
	DraftPickTrait
	// DraftPickIntoCount is the count a client's dispatch is held total against.
	DraftPickIntoCount
)

// DraftField is which row of the loadout has the cursor.
//
// ⚠️ **Four rows for three fields, and the fourth is why `enter` needs no
// explaining.** On every other screen in this program enter means "do what is
// under the cursor", and on a loadout there are two things it could mean: open
// the list on this row, or send the whole decision. Making the send a **row**
// keeps one meaning for one key and lets the footer name it once — where a key
// that meant two things would need a footer explaining which, and a footer that
// explains a key is a key nobody has understood.
type DraftField uint8

const (
	// DraftForm is the form the pick will field, a chooser walked with the
	// arrows.
	DraftForm DraftField = iota
	// DraftKit and DraftTrait each open a list.
	DraftKit
	DraftTrait
	// DraftSend is the row that takes the decision.
	DraftSend
	// DraftFieldCount is how many rows there are, and is what the cursor wraps
	// against.
	DraftFieldCount
)

// DraftDecision is a draft decision as this screen took it.
//
// ⚠️ **A shape of its own for the reason DraftLive is one**, and the same walk
// holds it: the client maps this into a wire.DraftDecision, and its Step has to
// be one of the five the protocol declares.
//
// ⚠️ **The form is what was NAMED and never what it resolved to.** An empty
// Stage is progression.Furthest — "the furthest the level cap reaches" — which
// is what a line that does not fork means, and the resolution is the room's and
// the mirror's to compute. A decision carrying a resolved form would be a second
// statement of something both peers already work out, which is the one place two
// peers can disagree. → wire.DraftDecision.Stage.
type DraftDecision struct {
	Step DraftStep
	// Character is the id banned or picked, and **empty on a ban is the skip** —
	// a slot spent on nobody, with no third state and no flag beside it.
	Character string
	Stage     string
	Skills    []string
	Passives  []string
	// Slots is one side's whole arrangement, in **pick order**: `Slots[i]` is the
	// cell for that side's i-th pick, which is what draft.Arrange takes and the
	// only order it can be read in.
	//
	// ⚠️ **A side arranges in one call rather than a cell at a time**, which is
	// why this is a list on one decision and not one decision per unit: the phase
	// is simultaneous and secret, so a cell-by-cell stream would need the same
	// buffer and would additionally hand a peer a partial arrangement to be told
	// about. Empty on every other step. → ArrangeScreen, and wire.DraftDecision.
	Slots []hex.Offset
}

// DraftResult is what a keystroke did to the draft screen, beside the screen
// itself.
//
// ⚠️ **The pair is not (itself, Action), and the reason is PickResult's exactly:
// an Action cannot carry an answer.** A draft decision is a step, a character, a
// form, four skills and a trait; Action holds a Kind, a Target and a Subject, and
// a Subject is one id — so a loadout would have to be encoded into a field built
// to name one thing, which is the shape this repository has twice paid for.
//
// ⚠️ **And Kind did not grow an eighth value, which was the alternative.**
// draw.Answer is *"a decision this screen has taken on a battle it does not
// drive"* and reusing it for a draft would be a second vocabulary inside one
// field; a Kind of its own would bill cmd/hexforge-tui for an arm about a screen
// it does not draw and can never reach. A result type is the answer the picker
// already established for the same problem.
type DraftResult struct {
	// Action is what the screen wants the client to do next — a Back, or a Pick
	// putting one of the loadout's two lists in front.
	Action Action
	// Decided reports a decision having been taken, and Decision is it.
	//
	// ⚠️ Carried **beside** the decision rather than read off it, for
	// PickResult.Answered's own reason: a zero Decision is a legal-looking value
	// (a skipped ban names nobody), so absence has to be declared rather than
	// detected.
	Decided  bool
	Decision DraftDecision
}

// DraftScreen is the ban and pick: the pool with what is gone marked, both
// sides' picks, whose decision is due, and the loadout that finishes a pick.
//
// It is drawn by cmd/hexarena-tui alone today — cmd/hexforge-tui has no room to
// join — and it lives here anyway, for the reason at the head of this file: it is
// a data column, and a data column held by one golden rather than two is a
// column whose width nothing measures on the client that stops drawing it.
type DraftScreen struct {
	// Live is the reading in hand, replaced whole on every Attach.
	Live DraftLive
	// Drafting says a reading has been handed over at all. Nought is a screen
	// nothing has attached to, which draws nothing rather than an empty draft.
	Drafting bool
	// Cursor indexes Live.Candidates, so it can only ever point at something the
	// open decision may actually name.
	//
	// ⚠️ **It indexes the candidates and not the pool rows**, which is what makes
	// a character already gone unselectable by construction rather than by a
	// refusal at the keystroke. The pool is drawn whole; the marker is put on the
	// row whose id the cursor names.
	Cursor int
	// Field is which row of the loadout has the cursor, and Stage, Skills and
	// Passives are the loadout under construction — this screen's own state,
	// which nothing on the wire has yet been told about.
	Field    DraftField
	Stage    string
	Skills   []string
	Passives []string
	// Err is what cast.ChooseFrom said about the kit as it was chosen, and it is
	// what blocks the send. → the file comment on why blocking is not politeness.
	Err error
}

// NewDraftScreen is the screen before a reading has been handed to it.
func NewDraftScreen() DraftScreen { return DraftScreen{} }

// Attach points this screen at a drafting room.
//
// It is called on **every** redraw rather than only when something changed —
// the client reads its mirror under a lock and hands over whatever it found — so
// it is cheap and idempotent, exactly as PlayScreen.Attach is. Unlike that one it
// is idempotent about everything, because a draft has no event cursor: the whole
// state arrives on every reading.
//
// What it keeps and what it throws away:
//
//   - The cursor follows the **character** rather than the row. The candidate
//     list shrinks under a reader — the opponent bans while they are walking
//     it — and every row below the one that went moves up by one, so a kept index
//     is a cursor that jumps to a neighbour every time the other player decides
//     anything. When the character under the cursor is the one that went, the row
//     number is kept, which lands on whatever is now next.
//   - The **loadout is thrown away when the pick it is owed for changes**, which
//     is the one thing here that resets. A form and a kit belong to the character
//     they were chosen for, so carrying them into the next pick would offer
//     skills the new character has never learned — and cast.ChooseLoadout would
//     refuse them, one decision later, with the reader unable to see why.
func (d DraftScreen) Attach(_ Context, live DraftLive) DraftScreen {
	previous, attached := d.Live, d.Drafting
	d.Drafting = true
	d.Live = live
	on := ""
	if attached && d.Cursor >= 0 && d.Cursor < len(previous.Candidates) {
		on = previous.Candidates[d.Cursor]
	}
	if at := slices.Index(live.Candidates, on); at >= 0 {
		d.Cursor = at
	}
	d.Cursor = Clamp(d.Cursor, 0, len(live.Candidates)-1)
	if previous.Subject != live.Subject {
		d.Field, d.Stage, d.Skills, d.Passives, d.Err = 0, "", nil, nil, nil
	}
	return d
}

// Choosing reports whether the loadout editor is in front rather than the pool:
// a loadout is due and it is this reader's.
//
// A loadout the **other** player owes draws the pool, because that is what there
// is to look at while they choose — and because nothing about their form and kit
// is public until the decision is recorded.
func (d DraftScreen) Choosing() bool {
	return d.Live.Yours && d.Live.Step == DraftStepLoadout
}

// Deciding reports whether there is a decision on the pool for this reader to
// take, which is what decides whether enter does anything and which footer is
// drawn.
func (d DraftScreen) Deciding() bool {
	return d.Live.Yours && (d.Live.Step == DraftStepBan || d.Live.Step == DraftStepPick)
}

// Chosen is the character under the cursor, and whether there is one.
func (d DraftScreen) Chosen() (string, bool) {
	if d.Cursor < 0 || d.Cursor >= len(d.Live.Candidates) {
		return "", false
	}
	return d.Live.Candidates[d.Cursor], true
}

// Character is the pool entry the loadout is being chosen for, and whether the
// pool holds one.
//
// ⚠️ **Read off the pool and never off Context.Lib**, which is the same rule
// DraftLive.Pool is handed in under: the legality of a kit has to be judged
// against the cast the *room* is drafting from, and --data may point at another
// one. A subject the pool does not hold is a reading nothing can act on, which
// the editor says rather than guessing at.
func (d DraftScreen) Character() (cast.Character, bool) {
	for _, character := range d.Live.Pool {
		if character.ID == d.Live.Subject {
			return character, true
		}
	}
	return cast.Character{}, false
}

// Update reads one keystroke.
func (d DraftScreen) Update(c Context, message tea.KeyPressMsg) (DraftScreen, DraftResult) {
	if message.String() == "esc" {
		// What leaving a drafting room costs is the client's to decide, exactly as
		// leaving a live battle is: a screen may not name its own way back, and
		// there is no screen behind this one — the room it was joined from no
		// longer exists.
		return d, DraftResult{Action: Action{Kind: Back}}
	}
	if d.Choosing() {
		return d.updateLoadout(c, message)
	}
	return d.updatePool(message)
}

// updatePool is the pool in front: a cursor over the candidates, and the one
// decision the open step allows.
func (d DraftScreen) updatePool(message tea.KeyPressMsg) (DraftScreen, DraftResult) {
	switch message.String() {
	case "up", "k":
		d.Cursor = Clamp(d.Cursor-1, 0, len(d.Live.Candidates)-1)
	case "down", "j":
		d.Cursor = Clamp(d.Cursor+1, 0, len(d.Live.Candidates)-1)
	case "enter":
		if !d.Deciding() {
			// A key the footer does not name on this reading, which is the whole
			// of the reading footer's promise. → i18n.DraftReadingFooter.
			return d, DraftResult{}
		}
		chosen, have := d.Chosen()
		if !have {
			return d, DraftResult{}
		}
		return d, DraftResult{Decided: true, Decision: DraftDecision{
			Step: d.Live.Step, Character: chosen,
		}}
	case "s":
		// ⚠️ **A skip is a decision somebody takes and not one anything invents.**
		// Bans are optional, and a skipped slot leaves the character in the pool
		// because it names nobody — so this sends a ban with no character, which
		// is the whole of how a skip travels. Offered on a ban and nowhere else,
		// which is what the two footers say.
		if d.Live.Yours && d.Live.Step == DraftStepBan {
			return d, DraftResult{Decided: true, Decision: DraftDecision{Step: DraftStepBan}}
		}
	}
	return d, DraftResult{}
}

// updateLoadout is the four-row editor: the form, the two lists and the send.
func (d DraftScreen) updateLoadout(c Context, message tea.KeyPressMsg) (DraftScreen, DraftResult) {
	switch message.String() {
	case "up":
		d.Field = (d.Field + DraftFieldCount - 1) % DraftFieldCount
	case "down":
		d.Field = (d.Field + 1) % DraftFieldCount
	case "left":
		d = d.cycleForm(-1)
	case "right":
		d = d.cycleForm(1)
	case "enter":
		switch d.Field {
		case DraftKit:
			if picker := d.OpenSkills(); picker != nil {
				return d, DraftResult{Action: Action{Kind: Pick, Picker: picker}}
			}
		case DraftTrait:
			if picker := d.OpenPassives(); picker != nil {
				return d, DraftResult{Action: Action{Kind: Pick, Picker: picker}}
			}
		case DraftSend:
			return d.send(c)
		}
	}
	return d, DraftResult{}
}

// send takes the loadout, or refuses it in the same words the room would.
//
// ⚠️ **The refusal is cast.ChooseLoadout's and the send is blocked on it.** That
// function is this repository's single declaration of "may this unit bring that
// skill" and draft.Draft.Loadout asks the same one, so a refusal drawn here is
// the answer the room would give rather than a guess at it — and sending a
// decision the room refuses would leave the decision open and be re-sent, which
// is the unbounded loop at the head of this file.
func (d DraftScreen) send(c Context) (DraftScreen, DraftResult) {
	d = d.judge()
	if d.Err != nil {
		return d, DraftResult{}
	}
	return d, DraftResult{Decided: true, Decision: DraftDecision{
		Step:     DraftStepLoadout,
		Stage:    d.Stage,
		Skills:   append([]string(nil), d.Skills...),
		Passives: append([]string(nil), d.Passives...),
	}}
}

// judge asks the loadout rule about what has been chosen so far, and is called
// as the kit is chosen as well as at the send — which is the point of asking it
// at all rather than letting the room answer.
func (d DraftScreen) judge() DraftScreen {
	character, known := d.Character()
	if !known {
		return d
	}
	form, err := d.form(character)
	if err != nil {
		d.Err = err
		return d
	}
	_, _, err = cast.ChooseLoadout(character.ID, d.Skills, d.Passives,
		character, progression.LevelCap, form)
	d.Err = err
	return d
}

// form is the form the loadout has named, resolved, or progression's own refusal
// for a line that forks and has not been told which arm.
//
// ⚠️ **The resolution is asked here so the arms can be offered**, and it is the
// same call draft.Draft.Loadout makes: an unnamed form on a forking line has no
// answer, and pokemon.poliwag is in the shipped pool — it reaches both of its
// grown forms at the cap. A screen that did not ask would offer a loadout the
// room refuses with a message about a level, which reads as a bug in the game.
func (d DraftScreen) form(character cast.Character) (string, error) {
	_, stage, err := character.Resolve(progression.LevelCap, d.Stage)
	if err != nil {
		return "", err
	}
	return stage.Name, nil
}

// FormChoices is the forms this pick may field: the empty string, which is "the
// furthest the cap reaches", and then every arm the line offers by name.
//
// It is the squad builder's StageChoices over a drafted pick, and the empty
// first entry is there for the same reason: a line that does not fork has one
// end and naming it would be a decision nobody has to take, while a line that
// forks has to be told which arm. A character the pool does not hold offers
// nothing.
func (d DraftScreen) FormChoices() []string {
	character, known := d.Character()
	if !known {
		return nil
	}
	stages, err := character.StagesAt(progression.LevelCap)
	if err != nil {
		return []string{progression.Furthest}
	}
	return append([]string{progression.Furthest}, progression.StageNames(stages)...)
}

// cycleForm walks the form chooser and drops a kit the new form no longer
// offers.
//
// ⚠️ **The kit is emptied rather than carried**, which is the same answer the
// squad builder gives when a member's character changes: a skill belongs to the
// form it was learned as, so carrying names the other arm has never heard of
// into the send would be a refusal one keystroke away from where it was caused.
func (d DraftScreen) cycleForm(by int) DraftScreen {
	forms := d.FormChoices()
	if len(forms) == 0 {
		return d
	}
	at := 0
	for index, form := range forms {
		if form == d.Stage {
			at = index
		}
	}
	next := forms[(at+by+len(forms))%len(forms)]
	if next == d.Stage {
		return d
	}
	d.Stage = next
	d.Skills, d.Passives, d.Err = nil, nil, nil
	return d
}

// OpenSkills and OpenPassives are the loadout's two lists.
//
// ⚠️ **They are PickState, which is the whole reason this screen needed no list
// of its own.** The multi-select already is "several out of a list, keeping the
// order", the squad builder's two are the prior art, and the two hints and the
// two slot counts below are that screen's — because a drafted loadout and a
// saved member's are the same question asked of the same books.
//
// The options are the **learnset**, so no row can carry a refusal, which is why
// each names its own hint rather than taking the picker's default.
//
// Nil is a pick the pool does not hold, or a form that does not resolve: there is
// no learnset to offer, and the key that raises the list declines rather than
// putting an empty one in front.
func (d DraftScreen) OpenSkills() *PickState {
	character, form, ok := d.learnset()
	if !ok {
		return nil
	}
	return &PickState{
		Title:   i18n.SquadPickSkills,
		Hint:    i18n.SquadKitHint,
		Kind:    PickSkills,
		Slots:   cast.SkillSlots,
		Options: IDOptions(character.SkillsAt(progression.LevelCap, form)),
		Chosen:  append([]string(nil), d.Skills...),
		Into:    DraftPickKit,
	}
}

func (d DraftScreen) OpenPassives() *PickState {
	character, form, ok := d.learnset()
	if !ok {
		return nil
	}
	return &PickState{
		Title:   i18n.SquadPickPassives,
		Hint:    i18n.SquadTraitHint,
		Kind:    PickPassives,
		Slots:   cast.TraitSlots,
		Options: IDOptions(character.PassivesAt(progression.LevelCap, form)),
		Chosen:  append([]string(nil), d.Passives...),
		Into:    DraftPickTrait,
	}
}

// learnset is the character and the resolved form its lists are read at.
func (d DraftScreen) learnset() (cast.Character, string, bool) {
	character, known := d.Character()
	if !known {
		return cast.Character{}, "", false
	}
	form, err := d.form(character)
	if err != nil {
		return cast.Character{}, "", false
	}
	return character, form, true
}

// Picked is what the loadout's two lists write back, judged by the rule the send
// uses.
//
// ⚠️ **The destination arrives as an `any`** — see SkillsScreen.Picked for why
// there is more than one destination vocabulary to tell apart — and a
// destination this screen does not own is left alone rather than guessed at.
//
// ⚠️ **The refusal is taken here as well as at the send, which is the point.**
// An author picking five skills for four slots should be told at the list rather
// than at the decision: the picker's own Slots cap stops an over-full answer
// being *built*, and this is the rule underneath it, asked through the same call
// the room will make.
func (d DraftScreen) Picked(_ Context, into any, answer PickAnswer) (DraftScreen, Action) {
	switch into {
	case DraftPickKit:
		d.Skills = answer.Chosen
	case DraftPickTrait:
		d.Passives = answer.Chosen
	default:
		return d, Action{}
	}
	// Both stay on the editor, so both hand back the zero action.
	return d.judge(), Action{}
}

// The pool listing's fixed column, which is browse's own: an id is the same in
// every language and as long as the data makes it, so it is a constant where a
// column of wording would be measured.
const draftIDWidth = 24

// draftStateWidth is the column the marks sit in, measured over the five
// wordings in the language in front.
//
// Measured rather than declared, for menuLabelWidth's reason: "they banned" is
// 11 cells and "bên kia cấm" is 11 too, but "you banned" is 10 against "bạn
// cấm" at 8, so one number for two languages is only right for both by luck.
func draftStateWidth(c Context) int {
	widest := 0
	for _, key := range []i18n.Key{
		i18n.DraftStateOpen, i18n.DraftStateYouBanned, i18n.DraftStateTheyBanned,
		i18n.DraftStateYouPicked, i18n.DraftStateTheyPicked,
	} {
		if width := lipgloss.Width(c.Text(key)); width > widest {
			widest = width
		}
	}
	return widest
}

// stateOf is the mark one row of the pool carries.
func (d DraftScreen) stateOf(id string) i18n.Key {
	live := d.Live
	mine, theirs := live.mine(), live.theirs()
	switch {
	case pickedBy(live.picksAt(mine), id):
		return i18n.DraftStateYouPicked
	case pickedBy(live.picksAt(theirs), id):
		return i18n.DraftStateTheyPicked
	case slices.Contains(live.bansAt(mine), id):
		return i18n.DraftStateYouBanned
	case slices.Contains(live.bansAt(theirs), id):
		return i18n.DraftStateTheyBanned
	}
	return i18n.DraftStateOpen
}

func pickedBy(picks []DraftPick, id string) bool {
	return slices.ContainsFunc(picks, func(one DraftPick) bool { return one.Character == id })
}

// Left is how many of the pool nobody has taken.
//
// ⚠️ **Derived from the marks and not from the candidate list**, and the
// difference is a whole decision: Candidates is what the *open* decision may
// choose from, which is empty during a loadout — so a count taken off it would
// tell a player nought characters were left while nineteen were still nobody's.
func (d DraftScreen) Left() int {
	left := 0
	for _, character := range d.Live.Pool {
		if d.stateOf(character.ID) == i18n.DraftStateOpen {
			left++
		}
	}
	return left
}

// View draws the screen and its footer.
func (d DraftScreen) View(c Context) (string, string) {
	if !d.Drafting {
		return "", c.Text(i18n.DraftReadingFooter)
	}
	head := d.head(c)
	tail := d.tail(c)
	var out strings.Builder
	out.WriteString(strings.Join(head, "\n") + "\n")
	if d.Choosing() {
		out.WriteString(strings.Join(d.loadoutRows(c), "\n") + "\n")
	} else {
		out.WriteString(strings.Join(d.poolRows(c, d.room(c, head, tail)), "\n") + "\n")
	}
	out.WriteString("\n" + strings.Join(tail, "\n"))
	return out.String(), d.footer(c)
}

// room is what the pool listing has left once everything around it is drawn.
//
// ⚠️ **Both blocks are counted rather than reserved at a worst case**, which is
// the opposite of what bonusesRoom does, and the reason is that they only ever
// grow the way the reader made them grow: the notices are the state of the draft
// and the side blocks grow by a row each time somebody picks. Reserving for the
// worst case — ten picks and two ban rows at five a side — would leave the
// listing nothing at all at the floor from the very first decision, and by the
// time both blocks really are that tall the picking is over and the pool is not
// what anybody is looking at.
//
// The floor of three is bonusesRoom's, for the same reason: a listing squeezed
// past that is a column of nothing.
// The `- 4` is the same one every Room helper in this package already spends —
// the two rows a client's header pair takes and the two the blank and the footer
// do — and it is a **mirror** of the clients' frame rather than its declaration:
// a screen package cannot see the frame that wraps it. What holds the real frame
// is each client's own screens.golden.
func (d DraftScreen) room(c Context, head, tail []string) int {
	room := c.Height - 4 - len(head) - len(tail) - 1 // the blank over the tail
	if room < 3 {
		return 3
	}
	return room
}

// head is the heading, the notices, whose decision is due and whatever the room
// last refused — every line above the listing, built as a block so that the
// listing's budget is a count rather than an estimate.
func (d DraftScreen) head(c Context) []string {
	live := d.Live
	lines := []string{
		c.Style.Heading.Render(c.Text(i18n.DraftHeading)) + "  " +
			c.Style.Dim.Render(c.Text(i18n.DraftPoolLeft, d.Left(), len(live.Pool))),
		"",
	}
	prose := func(style lipgloss.Style, text string) {
		for _, line := range WrapWords(text, MinWidth-3) {
			lines = append(lines, "  "+style.Render(line))
		}
	}
	switch {
	case live.Cancelled:
		// ⚠️ The **closure's** own wording, which is the sentence the result
		// screen draws for this same event. Wording it twice would let the screen
		// a player is standing on and the screen they land on afterwards disagree
		// about what happened. → i18n.ClosedDraftExpired.
		prose(c.Style.Bad, c.Text(i18n.ClosedDraftExpired))
	case live.Arranging:
		prose(c.Style.Dim, c.Text(i18n.DraftArrangingNow))
	case live.Waiting():
		prose(c.Style.Bad, c.Text(i18n.DraftNotBegun))
	}
	if turn := d.turnLine(c); turn != "" {
		lines = append(lines, "  "+turn)
	}
	if live.Refusal != "" {
		lines = append(lines, "  "+c.Style.Bad.Render(c.Text(i18n.DraftRefused)))
		prose(c.Style.Bad, c.Lang.Refusal(live.Refusal))
	}
	if only, one := live.OnlyOne(); one {
		prose(c.Style.Dim, c.Text(i18n.DraftOnlyOne, only))
	}
	if d.Err != nil {
		// Lang.Error is the house rule for a diagnostic that is not this
		// program's to word: the lead-in is the reader's language and
		// internal/core's own English follows it.
		prose(c.Style.Bad, c.Lang.Error(d.Err))
	}
	return append(lines, "")
}

// turnLine is whose decision is due and which kind, with the countdown beside
// it, and empty when nothing is due.
func (d DraftScreen) turnLine(c Context) string {
	live := d.Live
	said := ""
	switch {
	case live.Yours:
		switch live.Step {
		case DraftStepBan:
			said = c.Style.Emphasis.Render(c.Text(i18n.DraftYourBan))
		case DraftStepPick:
			said = c.Style.Emphasis.Render(c.Text(i18n.DraftYourPick))
		case DraftStepLoadout:
			said = c.Style.Emphasis.Render(c.Text(i18n.DraftYourLoadout, live.Subject))
		}
	case live.OnTurn != "":
		them := c.Lang.Seat(live.OnTurn)
		switch live.Step {
		case DraftStepBan:
			said = c.Text(i18n.DraftTheirBan, them)
		case DraftStepPick:
			said = c.Text(i18n.DraftTheirPick, them)
		case DraftStepLoadout:
			said = c.Text(i18n.DraftTheirLoadout, them)
		}
	}
	if said == "" {
		return ""
	}
	if clock := d.clockLine(c); clock != "" {
		return said + c.Style.Dim.Render("  ·  "+clock)
	}
	return said
}

// clockLine is the two countdowns, and empty when no decision is being counted.
//
// It is PlayClock's own pair of wordings, which take the reader's own number
// first in both languages: a number that changes position when the turn changes
// is a number nobody can watch.
func (d DraftScreen) clockLine(c Context) string {
	clock := d.Live.Clock
	switch clock.Waiting {
	case PlayClockYou:
		return c.Text(i18n.PlayClockYours, playClock(clock.Yours), playClock(clock.Theirs))
	case PlayClockThem:
		return c.Text(i18n.PlayClockTheirs, playClock(clock.Yours), playClock(clock.Theirs))
	}
	return ""
}

// poolRows is the pool in the book's declaration order, each row marked, with
// the cursor on whatever the open decision would name.
func (d DraftScreen) poolRows(c Context, room int) []string {
	live := d.Live
	if len(live.Pool) == 0 {
		return []string{"  " + c.Text(i18n.DraftPoolLeft, 0, 0)}
	}
	// The window is taken over the **pool** row the cursor names rather than over
	// the cursor itself: the cursor indexes the candidates, which is a shorter
	// list than the rows being drawn, so scrolling by it would put the window
	// somewhere near the top for ever.
	at := 0
	if chosen, have := d.Chosen(); have {
		at = slices.IndexFunc(live.Pool, func(one cast.Character) bool {
			return one.ID == chosen
		})
		at = Clamp(at, 0, len(live.Pool)-1)
	}
	width := draftStateWidth(c)
	from, to := Window(len(live.Pool), at, room)
	rows := make([]string, 0, to-from)
	for index := from; index < to; index++ {
		character := live.Pool[index]
		// The character's element, not a form's, for the reason the cast
		// listing's own row keeps it: a pool row has no level and therefore no
		// form to resolve, and the glossed affinity is the last column, so a
		// line that gained an element on evolution would widen the one column
		// with nothing behind it to give way. A ban is a decision about the
		// character rather than about a form, which is what the column is for.
		line := Pad(character.ID, draftIDWidth) + " " +
			Pad(c.Text(d.stateOf(character.ID)), width) + " " +
			c.Lang.GlossedAffinity(character.Element)
		marker := "  "
		if chosen, have := d.Chosen(); have && chosen == character.ID {
			// The marker is the selection and the style only agrees with it,
			// which is Palette's rule: this row is the one enter names.
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		rows = append(rows, marker+strings.TrimRight(line, " "))
	}
	return rows
}

// loadoutRows is the four-row editor: the form, the two lists and the send.
func (d DraftScreen) loadoutRows(c Context) []string {
	width := draftLabelWidth(c)
	rows := []string{
		d.row(c, width, DraftForm, i18n.SquadFieldStage, d.formValue(c)),
		d.row(c, width, DraftKit, i18n.SquadFieldSkills,
			d.listValue(c, d.Skills, cast.SkillSlots)),
		d.row(c, width, DraftTrait, i18n.SquadFieldPassives,
			d.listValue(c, d.Passives, cast.TraitSlots)),
		d.row(c, width, DraftSend, i18n.DraftFieldSend, d.sendValue(c)),
	}
	// ⚠️ **The fork's own line, and it is the one diagnostic on this screen that
	// is drawn without anything having gone wrong yet.** An unnamed form on a
	// forking line has no answer — progression refuses to pick an arm — so both
	// lists above show only what every arm learns and the send is refused. A
	// player who cannot see that is looking at a shortened learnset with nothing
	// saying why, which is the defect the squad builder shipped and then fixed.
	//
	// It is the squad builder's shape and **not** its wording: SquadForkArms ends
	// on *"the save refuses"*, and a draft has no save. → i18n.DraftForkArms.
	if arms := d.unnamedArms(); len(arms) > 1 {
		rows = append(rows, "", "  "+c.Style.Bad.Render(c.Text(i18n.DraftForkArms,
			strings.Join(progression.StageNames(arms), " / "))))
	}
	return rows
}

// unnamedArms is the arms this pick's line reaches with no form named, and
// nothing once one is — which is what makes the line above appear exactly while
// it is true.
func (d DraftScreen) unnamedArms() []progression.Stage {
	if d.Stage != "" {
		return nil
	}
	character, known := d.Character()
	if !known {
		return nil
	}
	arms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		return nil
	}
	return arms
}

// draftLabelWidth is the column the editor's four row names sit in, measured
// over the language in front for the reason draftStateWidth is.
func draftLabelWidth(c Context) int {
	widest := 0
	for _, key := range []i18n.Key{
		i18n.SquadFieldStage, i18n.SquadFieldSkills, i18n.SquadFieldPassives,
		i18n.DraftFieldSend,
	} {
		if width := lipgloss.Width(c.Text(key)); width > widest {
			widest = width
		}
	}
	return widest
}

// row is one editor row, drawn the way every labelled row in these clients is.
func (d DraftScreen) row(c Context, width int, field DraftField,
	label i18n.Key, value string) string {
	marker := "  "
	if d.Field == field {
		marker = "> "
	}
	return marker + c.Style.Label.Render(Pad(c.Text(label), width)) + " " + value
}

// formValue is the form chooser's row: the form between arrows and where it sits
// in the list.
func (d DraftScreen) formValue(c Context) string {
	forms := d.FormChoices()
	if len(forms) == 0 {
		return c.Style.Dim.Render(Ellipsis)
	}
	at := 0
	for index, form := range forms {
		if form == d.Stage {
			at = index
		}
	}
	return fmt.Sprintf(ChoiceFormat, d.formLabel(c),
		c.Style.Dim.Render(fmt.Sprintf("%d/%d", at+1, len(forms))))
}

// formLabel is what the form row calls the form in hand: the name when one was
// chosen, and otherwise the squad builder's own two words for a line that has
// one end and a line that has two.
func (d DraftScreen) formLabel(c Context) string {
	if d.Stage != "" {
		return d.Stage
	}
	if len(d.FormChoices()) > 2 {
		// More than "furthest" and one arm, so the line forks and an unnamed form
		// has no answer. → i18n.SquadForkUnnamed, and the refusal above this row.
		return c.Text(i18n.SquadForkUnnamed)
	}
	return c.Text(i18n.SquadFurthest)
}

// listValue is a chosen list with how full its slots are, so an unfinished kit
// says so before the send does. It is the squad builder's row, in the same two
// wordings.
func (d DraftScreen) listValue(c Context, chosen []string, slots int) string {
	if len(chosen) == 0 {
		return c.Style.Dim.Render(c.Text(i18n.SquadNothingChosen, slots))
	}
	return fmt.Sprintf("%s %s", strings.Join(chosen, " "),
		c.Style.Dim.Render(fmt.Sprintf("%d/%d", len(chosen), slots)))
}

// sendValue is what the send row says: that enter takes the decision, or nothing
// when the loadout is refused — the reason itself is drawn above the rows, where
// every other diagnostic on this screen is.
func (d DraftScreen) sendValue(c Context) string {
	if !d.sendable() {
		return c.Style.Bad.Render(Ellipsis)
	}
	return c.Style.Dim.Render(c.Text(i18n.DraftSendReady))
}

// sendable reports whether enter on the send row would take the decision.
//
// ⚠️ **It asks the FORM as well as Err, and that is not belt and braces.** Err
// holds what the kit choice produced, and a form is refused before any kit has
// been chosen — an unnamed arm on a forking line has no answer at all — so a row
// reading off Err alone announces that enter sends this loadout on exactly the
// screen where enter cannot. The reason is drawn above the rows either way: the
// kit's as a diagnostic, the fork's as the arms line under them.
func (d DraftScreen) sendable() bool {
	if d.Err != nil {
		return false
	}
	character, known := d.Character()
	if !known {
		return false
	}
	_, err := d.form(character)
	return err == nil
}

// tail is the two sides' own blocks, this reader's first.
//
// ⚠️ **The reader's own side is always first**, which is PlayClock's rule for the
// same reason: a block that changed position with the seat is a block nobody can
// find twice.
func (d DraftScreen) tail(c Context) []string {
	live := d.Live
	mine, theirs := live.mine(), live.theirs()
	lines := d.sideBlock(c, c.Text(i18n.DraftYourSide,
		len(live.picksAt(mine)), live.Units), mine)
	them := c.Lang.Seat(other(live.Seats, live.You))
	return append(lines, d.sideBlock(c, c.Text(i18n.DraftTheirSide,
		them, len(live.picksAt(theirs)), live.Units), theirs)...)
}

// other is the seat that is not this reader's, which is what names the opposite
// block. An empty You leaves the second seat named, which is the reading a
// screen with no seat of its own falls into.
func other(seats [2]string, you string) string {
	if seats[0] == you {
		return seats[1]
	}
	return seats[0]
}

// sideBlock is one side: a heading, a row per pick, and what it took out of the
// pool.
func (d DraftScreen) sideBlock(c Context, heading string, index int) []string {
	live := d.Live
	lines := []string{"  " + c.Style.Label.Render(heading)}
	picks := live.picksAt(index)
	if len(picks) == 0 {
		lines = append(lines, "    "+c.Style.Dim.Render(c.Text(i18n.DraftNoPicks)))
	}
	for _, pick := range picks {
		lines = append(lines, "    "+Pad(pick.Character, draftIDWidth)+" "+d.pickValue(c, pick))
	}
	bans := live.bansAt(index)
	if len(bans) == 0 {
		lines = append(lines, "    "+c.Style.Dim.Render(c.Text(i18n.DraftBannedNobody)))
		return lines
	}
	return append(lines, "    "+c.Style.Dim.Render(
		c.Text(i18n.DraftBanned, strings.Join(bans, " · "))))
}

// pickValue is what one pick will field, or that its loadout is still open.
func (d DraftScreen) pickValue(c Context, pick DraftPick) string {
	if !pick.Settled() {
		return c.Style.Dim.Render(c.Text(i18n.DraftLoadoutOpen))
	}
	return pick.Stage + "  " + c.Style.Dim.Render(c.Text(i18n.SquadLoadoutCount,
		len(pick.Skills), cast.SkillSlots, len(pick.Passives), cast.TraitSlots))
}

// footer is the keys this reading actually offers.
//
// ⚠️ **Four footers, and the reading one is the load-bearing half.** A footer
// naming a decision key on a decision that is not this player's would be the
// program promising something it does not do — the rule picker.go states and
// Context.Footer exists to keep — and here the promise is a whole match: a player
// who believes they have banned somebody finds out one allowance later.
func (d DraftScreen) footer(c Context) string {
	switch {
	case d.Choosing():
		return c.Text(i18n.DraftLoadoutFooter)
	case d.Live.Yours && d.Live.Step == DraftStepBan:
		return c.Text(i18n.DraftBanFooter)
	case d.Live.Yours && d.Live.Step == DraftStepPick:
		return c.Text(i18n.DraftPickFooter)
	}
	return c.Text(i18n.DraftReadingFooter)
}
