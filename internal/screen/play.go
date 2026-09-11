package screen

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// PlayScreen is a battle fought by hand against the opponent the engine plays.
//
// It is raised from the fight for the reason the fight is raised from the
// catalogue: that is where a pairing is already chosen, and a battle wants two
// squads before it wants anything else. What the simulation answers over two
// hundred battles, this answers once — and the two are worth having side by
// side, because a rate says which squad is better and playing one says why.
//
// ⚠️ **This is the one screen holding something the model does not copy.** Every
// other screen is a value the model carries, so a field written while drawing is
// thrown away with the copy; a battle is a pointer, and a mutation reaches every
// copy of the model there is. So the battle is stepped in update and **never**
// touched in view — view reads it and draws, and that division is what keeps a
// redraw from playing a turn.
type PlayScreen struct {
	// Seed is which battle this is. Walking it is how a player asks for another
	// arrangement of the same pairing, and it is the same number the log carries,
	// so a battle played here can be replayed by the game client.
	Seed uint64
	// Side is the half the player is fighting on: the home squad's, always, so
	// that the squad under the catalogue's cursor is the one being played rather
	// than the one being played against.
	Side hex.Side

	// Home and Away are the pairing this screen was opened on, and they are
	// **handed in** rather than fetched.
	//
	// ⚠️ This is the one thing about the move that was design rather than
	// mechanics. The screen used to ask the client's own fight screen — a screen
	// that has not moved and is not going to — which two squads it was between,
	// twice: once to field the roster when a battle starts, and once to name the
	// log file a save writes. Both are the same question, and it is a question
	// about *opening* this screen rather than something to reach sideways for. So
	// the client, which owns the fight, answers it once and passes the answer to
	// Open; the same fix #207 made for the two describers, which used to read
	// three other screens and are now handed a Subject.
	//
	// Both are kept rather than reduced to what each caller wants, because the
	// two callers want different halves of them: fielding needs the units and the
	// save needs the ids, and undo and another seed both re-field from here.
	Home, Away placement.Squad

	// Fight is the battle in hand, and nil is a screen with no pairing to open
	// one on.
	//
	// ⚠️ **On a LIVE screen this pointer is the mirror's own, and the drawing
	// path may not touch it.** Another goroutine steps that battle every time a
	// turn arrives, so a read taken while drawing is a read with no lock on it —
	// which is a data race whatever it reads, and the detector caught the board
	// and the roster doing exactly that. Live mode may dereference it in Attach
	// and nowhere else: Attach runs inside socket.Mirror.Read, under the lock, so
	// that is the one moment the battle is safe to ask anything. Everything a
	// draw needs is taken there and carried in the reading field below.
	//
	// A **local** battle is this screen's own — nothing else holds it and nothing
	// else steps it — so the local path reads it freely, as it always has. → read.
	Fight *battle.Battle
	// Roster is what the battle was built from, kept because a log records it:
	// a log carrying the resolved placement is what makes it re-runnable across
	// a data edit, and asking the battle for it afterwards would be asking for
	// the units as they are now rather than as they were placed.
	Roster []battle.Roster
	// Tags and Names are what a drawing calls each unit, read off the board once
	// when the battle is built.
	Tags  map[string]string
	Names map[string]string
	// Events is every event the battle has emitted, which is the whole history
	// the log is a frame over.
	Events []battle.Event
	// Script is every decision taken, the player's and the engine's alike. It is
	// what undo shortens and what the whole battle is rebuilt from, which is why
	// a rebuild needs nothing else: a battle is a pure function of its seed and
	// the decisions taken.
	Script battle.Script
	// Pending is the turn waiting on the player. Nil means the battle is between
	// turns, which is where the engine's own units act.
	Pending *battle.Prompt

	// Option and Aim are the two questions a turn asks, and Aiming says which of
	// them is in front.
	Option int
	Aim    int
	Aiming bool

	// LogFollow and LogOffset are where the log's frame sits, and they are two
	// fields rather than one for the one reason that decides this whole feature.
	//
	// ⚠️ **Following the tail is a state, not an offset value, because the tail
	// moves.** The reader is normally looking at the newest rows; an offset
	// storing "the newest rows" would be a number that means something different
	// every time an event arrives, so every turn taken would silently shift what
	// is under the reader. So LogOffset counts rows **from the start of the
	// history** — which is exactly what the position on the heading row states —
	// and following is carried **beside** it rather than encoded into it.
	//
	// This is the rule Suggest's abandoned queue tie-break paid for: Queue.Pending
	// answers 0 for a unit it has never heard of and 0 is *soonest*, so absence
	// had to be declared rather than detected. A sentinel offset meaning "the
	// tail" would be that mistake again — and it would read as working, because
	// the sentinel is a legal offset on the turn it is written.
	//
	// LogOffset is meaningless while LogFollow holds, and it is clamped against
	// the history's current length wherever it is read rather than only where it
	// is written: undo rebuilds the battle from a shortened script, so the
	// history it rebuilds is shorter and an offset kept across it can point past
	// the end.
	LogFollow bool
	LogOffset int

	// Err is what went wrong building or stepping the battle, drawn in place of
	// the screen.
	Err error
	// Notes are what a write left behind, held as facts rather than as a
	// sentence so a language toggle redraws them in the other one.
	Notes []forge.Note

	// Live says somebody else is driving this battle, and it is the whole of
	// what a PvP battle is on this screen: the engine is a socket.Mirror's, the
	// decisions are the server's, and this screen renders and chooses.
	//
	// ⚠️ **One battle exists per client and it is the mirror's.** A screen that
	// also called Fight.Act would step the same battle twice and diverge from
	// the room on turn one — the mirror applies on the wire.Turn that comes
	// back, deliberately, which is why the room sends every turn to both clients
	// including the one that asked. So every key that spends a turn is guarded
	// on this field, at the key, once.
	//
	// Nought is the local reading, which is the half a forgotten declaration has
	// to fall into: a local screen that quietly went live would stop driving its
	// own battle and sit for ever, while a live screen that quietly went local
	// would play the opponent's turns for them.
	Live bool
	// Watching says this client is a **spectator** of the live battle rather than
	// one of the two people in it, and it is a third reading of this screen rather
	// than a fourteenth screen.
	//
	// ⚠️ **A mode and not a screen of its own, which is the opposite of the
	// decision ArrangeScreen took, and the two differ in what they SHARE.** The
	// arrangement is a cursor over a 3x3 where the draft is a cursor over a list:
	// no key, no section and no budget in common, so a mode there would have been
	// two screens in one type. A spectator shares **everything** — the board, the
	// roster, the queue line, the log and the frame the log is read through, the
	// heading, the notice and the whole priority in playFit — and differs only by
	// **absence**. Two copies of that budget is the defect this screen's own row
	// arithmetic was written to stop, and the second copy is the one that would
	// silently stop matching.
	//
	// ⚠️ **What makes "a watcher is never asked" hold is NOT this field, and that
	// is deliberate.** It is Pending being nil, which is socket.Mirror.Asking
	// being false for a watching mirror at the one derivation there is — so every
	// key that spends a turn is already behind the `p.Pending == nil` return in
	// Update, which is a guard that was there before spectators existed and is
	// exercised on every turn of every match. A second guard reading this field
	// would be a guard a mutation could delete for free, which is the mistake
	// recorded on battle.healingFor's single floor.
	//
	// So what this field decides is only what is **said**: three wordings that
	// address the reader as one of the two players are replaced by three that do
	// not. That is Context.Footer's rule, and here the difference is a whole
	// match — → View, the tail in drawings, and clocks.
	//
	// Nought is the reading a forgotten declaration falls into and it is the safe
	// half: a spectator that quietly went back to being a player would be offered
	// a turn it can never answer, and a player that quietly went to watching would
	// be told to press keys that are not there.
	Watching bool
	// Reconnecting is this client having lost its socket mid-match and being in
	// the middle of taking its seat back. → PlayLive.Reconnecting, where the
	// argument for drawing it at all is, and waiting, which is the row it takes.
	Reconnecting bool
	// Cursor is this screen's own read position in a battle it does not own.
	//
	// ⚠️ **Live mode may not call Drain.** Drain is Since(b.drained) over an
	// append-only record and it *writes* b.drained — and a live battle is read
	// under a lock that admits several readers, so a read that moved the mirror's
	// own cursor would be a write smuggled through one. Two consumers of one
	// battle is exactly what the record-and-cursor work bought; this is the
	// second cursor. Local mode keeps Drain untouched, which is what keeps the
	// hot-seat goldens still.
	Cursor int
	// Answered is the decision already sent for the open prompt, so a screen
	// that has answered stops offering the same turn again — and stops drawing
	// an option list whose keys it would ignore.
	//
	// Cleared by Attach when the prompt names a different (unit, turn), which is
	// the pair a decision is identified by everywhere else in this repository.
	// ⚠️ Not the prompt pointer: a mirror hands back a pointer into its own
	// battle and two prompts for one turn are the same turn however they are
	// allocated.
	Answered     bool
	answeredUnit string
	answeredTurn int
	// LiveRefusal is the **name** of the latest protocol refusal this client has
	// been sent mid-match, drawn where a save's note goes and empty when there
	// has been none.
	//
	// ⚠️ **A name and never a typed value.** i18n.Lang.Refusal takes a string
	// for exactly this reason: a wire.Code on a field here would pull the
	// protocol into the package two clients share, and this package must not
	// know a socket exists. The client turns the enum into its own id and this
	// screen hands it to the language book.
	//
	// ⚠️ It is the one place three of the ten refusals can ever be read —
	// not-your-turn, illegal-action and unknown-message are only reachable
	// **during** a match, because a refusal does not end a connection.
	LiveRefusal string
	// Clock is what is left of the open turn's allowance on both sides, **handed
	// in already counted**. → PlayClock.
	Clock PlayClock

	// reading is what a drawing needs of a live battle, taken while the mirror's
	// read lock was held.
	//
	// ⚠️ **It is the reason a redraw of a match reads no battle at all**, and it
	// is only meaningful while Live: a local screen owns its battle and reads it
	// as it draws, so nothing is stored for it. → read, which is the one place
	// that choice is made, and the Fight field for what went wrong without it.
	//
	// Unexported because it is not a knob. Every live screen in the repository is
	// built by Attach, which is where the value comes from, and a caller able to
	// set it could hand this screen a board its battle never held.
	reading playReading
}

// PlayClock is the two countdowns a live battle draws, in **seconds**.
//
// ⚠️ **This package reads no clock and this type is how it stays that way.** It
// is the same arrangement one layer down: internal/room never asks what time it
// is either — `Allowance` is a number the room *carries* and hands to its
// clients, and whoever owns the transport counts it down and reports what
// happened. A screen is one layer further out and gets the same treatment. It is
// handed two counts of seconds and draws them; it does not know what a second
// is, when this turn opened, or which machine's clock said so.
//
// So there is no time.Duration here on purpose, for wire.Welcome.Allowance's own
// reason: seconds as an int, because the number crosses a boundary and a
// Duration is a count of nanoseconds that means nothing on the other side of
// one.
type PlayClock struct {
	// Waiting is whose answer the open turn is waiting for, and **nought is
	// nobody** — which is the reading a screen handed a zero value falls into,
	// and the safe one: no turn is being counted, so no clock is drawn.
	Waiting PlayClockSeat
	// Yours and Theirs are the seconds left, this screen's reader first.
	//
	// ⚠️ **Only one of them is counting down.** The allowance is per prompt
	// rather than per match — a chess clock is not what a room runs — so the
	// player who is not being asked has the whole of it waiting for them, and
	// that is what their number says.
	Yours, Theirs int
}

// PlayClockSeat is which side of the wire the open turn belongs to.
type PlayClockSeat int

const (
	// PlayClockNobody is no turn open to count: between turns, before the first
	// one, and on a battle that is over.
	PlayClockNobody PlayClockSeat = iota
	// PlayClockYou is this player being asked, PlayClockThem the other one.
	PlayClockYou
	PlayClockThem
)

// PlayTurnLimit is where a battle is abandoned, and it is cmd/hexarena's number
// deliberately: a battle this screen calls endless and a battle the game calls
// endless have to be the same battle.
const PlayTurnLimit = 4000

// NewPlayScreen is the screen before a pairing has been handed to it: no
// battle, the first seed, and the player on the home side.
func NewPlayScreen() PlayScreen {
	// Following, because the newest rows are what a battle nobody has scrolled is
	// showing, and because the alternative would be an offset into a history that
	// does not exist yet.
	return PlayScreen{Seed: 1, Side: hex.SideAlly, LogFollow: true}
}

// Open enters the screen on a pairing and plays it up to the player's first
// decision.
//
// The two squads are the whole of what this screen needs from the client, and
// they are a **parameter of opening** rather than something to fetch — see Home
// and Away for why that is the design and not a signature preference. The seed
// and the side are kept, so re-opening the same pairing walks on from the seed
// the reader had reached.
//
// ⚠️ **A squad with nobody in it is not a pairing**, which is what a client with
// an empty catalogue has to hand over, and the screen then says nothing has been
// built rather than reporting the refusal fielding one would give. That is a
// reading of the value rather than a sentinel: placement.Squad.Validate already
// refuses an empty squad, so nothing that could be fielded arrives this way.
// ⚠️ **A live screen refuses to be opened**, and that is the guard the two risks
// of this feature meet at: a client whose menu still offers a battle while a
// match is running would build a second one over the mirror's, and the reader
// would be playing a hot-seat game against a copy of themselves while a person
// on another machine waited. Attach is the live entrance and Open is the local
// one; neither is the other's fallback.
func (p PlayScreen) Open(c Context, home, away placement.Squad) PlayScreen {
	if p.Live {
		return p
	}
	p.Home, p.Away = home, away
	return p.begin(c)
}

// Attach points this screen at a battle it does not drive.
//
// It is the live counterpart of Open and it is called on **every** redraw, not
// only when something changed: the client reads its mirror under a lock and
// hands over whatever it found, so this has to be cheap and idempotent. The one
// thing it is not idempotent about is the event cursor, which is the point — the
// run since the last call is what a renderer has not drawn yet.
//
// What it resets and when:
//
//   - A **different battle pointer** is the next battle of the series, so the
//     cursor, the history, the option cursor and the aim all start again, and
//     the tags and names are read off the new board. The seed and the side come
//     off the message rather than being kept, because both change between
//     battles of a match.
//   - A **different (unit, turn)** on the prompt is a new turn, so Answered is
//     cleared and the option cursor opens on the first option the unit may
//     actually take.
//   - Anything else is the same turn seen again, and nothing moves.
func (p PlayScreen) Attach(c Context, live PlayLive) PlayScreen {
	p.Live = true
	if live.Fight != p.Fight {
		p.Fight = live.Fight
		p.Cursor, p.Events = 0, nil
		p.Option, p.Aim, p.Aiming = 0, 0, false
		p.Answered, p.answeredUnit, p.answeredTurn = false, "", 0
		p.LogFollow, p.LogOffset = true, 0
		p.Pending = nil
		p.Tags, p.Names = nil, nil
		if live.Fight != nil {
			p.Tags = tui.Tags(live.Fight.Units())
			p.Names = tui.Names(live.Fight.Units())
		}
	}
	p.Side, p.Seed = live.Side, live.Seed
	// Taken on every reading rather than only when the battle changes, like the
	// refusal and the clock below: it is a fact about the connection this reading
	// came off, and a screen that kept an earlier answer would be a screen whose
	// mode outlived the match it was told about.
	p.Watching = live.Watching
	p.Reconnecting = live.Reconnecting
	p.LiveRefusal = live.Refusal
	// Taken on every reading like the refusal is, because it is a reading rather
	// than a state: the client counts it down and this is called on every redraw.
	p.Clock = live.Clock
	if p.Fight != nil {
		// Since rather than Drain: a pure read, so this is safe under the read
		// lock the caller is holding. → the Cursor field.
		events, next := p.Fight.Since(p.Cursor)
		p.Cursor = next
		p.Events = append(p.Events, events...)
	}
	// ⚠️ **Everything a draw will read off the battle is read HERE**, because
	// here is the only place a live screen is holding the lock. It is taken on
	// every call rather than only when the battle pointer changes, for the reason
	// the cursor is: the battle the mirror hands over is being stepped, so a
	// reading kept from an earlier call would draw a board several turns stale.
	// The comment above this block already knew the rule and said it about one
	// call; it is the whole path. → the reading field.
	p.reading = readBattle(p.Fight, p.Tags)
	opened := live.Asking
	switch {
	case opened == nil:
		p.Pending = nil
		p.Aiming = false
	case p.Answered && opened.Unit == p.answeredUnit && opened.Turn == p.answeredTurn:
		// The same turn, already answered. The prompt is kept so the screen
		// still knows whose turn it was, and the tail says it is waiting.
		p.Pending = opened
	case p.Pending == nil || opened.Unit != p.Pending.Unit || opened.Turn != p.Pending.Turn:
		p.Pending = opened
		p.Answered, p.answeredUnit, p.answeredTurn = false, "", 0
		p.Option = p.firstAvailable(opened)
		p.Aim, p.Aiming = 0, false
	default:
		p.Pending = opened
	}
	return p
}

// PlayLive is a battle somebody else is driving, as one reading of it.
//
// ⚠️ **It is a value taken under the mirror's read lock and it does not outlive
// the call.** The *battle.Battle in it is the mirror's own — a client computes
// the board by computing the battle, which is the whole design and the reason no
// type walk can refuse the pointer — so the caller's job is to build one of
// these inside socket.Mirror.Read and hand it straight here.
//
// ⚠️ **There is deliberately no index of the battle within the series.** The
// plan this arrived under carried one; nothing on this screen reads it, the
// heading already names the seed and a seed changes between battles of a match,
// and a field a screen does not draw is a field the next reader wonders about.
// The series standing is the *result* screen's, off socket.Mirror.Fought.
type PlayLive struct {
	// Fight is the mirror's battle, nil between battles and before the first.
	Fight *battle.Battle
	// Asking is the open prompt when it is this player's to answer, and nil
	// otherwise — which is socket.Mirror.Asking exactly, because that derivation
	// and PlayScreen.Pending are the same fact.
	Asking *battle.Prompt
	// Side is the half of the board this player is on in this battle, and Seed
	// which battle it is. Both change between battles of a match, which is why
	// they arrive on every reading rather than being set once.
	Side hex.Side
	Seed uint64
	// Watching is this client watching the match rather than playing in it, which
	// is socket.Sight.Watching and therefore wire.Welcome.Watching.
	//
	// ⚠️ **Side above is then the HOST's half rather than this client's**, because
	// a spectator plays neither and the wire.Start on the room's record carries the
	// host's. So the board a watcher reads is the board the host reads, and the two
	// halves are the same `A` and `E` on both screens — which is what lets the
	// clocks be worded off those letters. → PlayScreen.Watching.
	Watching bool
	// Refusal is the name of the latest protocol refusal, empty when there has
	// been none. → PlayScreen.LiveRefusal for why it is a name.
	Refusal string
	// Clock is the two countdowns, **already computed by the client**, and a
	// zero value is a turn nobody is being asked about. → PlayClock for why the
	// arithmetic is not done here.
	Clock PlayClock
	// Reconnecting is this client having lost its socket mid-match and being in
	// the middle of taking its seat back.
	//
	// ⚠️ **A board that simply stopped moving is indistinguishable from the other
	// player thinking**, and the two want opposite things from a reader: one is
	// nothing to do about, and the other is somebody staring at a frozen screen
	// wondering whether to quit — which, inside the window the seat is held for,
	// is the one thing that would actually lose the match. So it is drawn, on the
	// row the waiting line already has rather than on one of its own, because
	// this screen has no row to give. → PlayScreen.waiting.
	Reconnecting bool
}

// playReading is a battle as a drawing needs it: one value, taken at one moment.
//
// ⚠️ **This is socket.DraftSight's decision arriving one screen later.** That
// type is a snapshot and never a *draft.Draft, and its own comment names
// Sight.Fight beside it as the one deliberate exception — a renderer computes
// the board by computing the battle, so the pointer had to reach this screen.
// It still does, and Attach still dereferences it; what changed is that the
// pointer stops there. A draw reads this instead, so there is nothing left on
// the drawing goroutine that could be racing the goroutine stepping the match.
//
// It holds the three sections **already rendered** rather than the units they
// were rendered from, because rendering them is the read: tui.Board, tui.Roster
// and tui.Order each walk the units, the statuses and the queue, and a value
// carrying those to be walked later would have moved the race rather than
// removed it. The units are carried as well because two rows are assembled from
// them here — whose turn it is, and who is standing on an aim.
type playReading struct {
	// board, roster and order are tui.Board, tui.Roster and tui.Order, unstyled:
	// the palette is applied where the section is placed, exactly as before.
	board, roster, order string
	// finished, outcome and winner are how the battle stands, which is the
	// question the turn in front is budgeted around. → drawings, ending.
	finished bool
	outcome  battle.Outcome
	winner   hex.Side
	// units is every unit on the board, in the battle's own order, flattened to
	// the four facts a drawing asks about one.
	units []playUnit
}

// playUnit is one unit as a drawing needs it.
//
// ⚠️ **Side and Affinity are here because a drawing may not go and ask.** Both
// are facts a live screen would otherwise have to read off the battle at the
// moment it draws, which is the one thing the reading exists to stop — the
// mirror's lock is held in Attach and nowhere else. Affinity is what the aim
// list marks a target's matchup from, and Side is which half a shape is walked
// in; the second used to be read through p.Fight.Unit inside splashUnder, on
// the drawing path, which is exactly the race readBattle was written to remove.
//
// They are captured for every unit rather than for the few an aim list happens
// to name, for the reason the rest of this struct is: the reading is taken once
// per redraw and a selective one would need to know what the draw was going to
// ask before the draw asked it.
type playUnit struct {
	ID, Name string
	Cell     hex.Offset
	Dead     bool
	Side     hex.Side
	Affinity element.Affinity
}

// readBattle takes a reading. A nil battle reads as the zero value, which is
// the screen with nothing to draw.
//
// ⚠️ **Whoever calls this has to be allowed to read the battle.** For a live
// screen that is Attach and only Attach, under the mirror's read lock; for a
// local one it is any time, because nothing else holds the battle. → read.
func readBattle(fight *battle.Battle, tags map[string]string) playReading {
	if fight == nil {
		return playReading{}
	}
	winner, _ := fight.Winner()
	read := playReading{
		board:    tui.Board(fight, tags),
		roster:   tui.Roster(fight, tags),
		order:    tui.Order(fight.Queue(), tags, 6),
		finished: fight.Finished(),
		outcome:  fight.Outcome(),
		winner:   winner,
	}
	units := fight.Units()
	read.units = make([]playUnit, 0, len(units))
	for _, unit := range units {
		read.units = append(read.units, playUnit{
			ID: unit.ID, Name: unit.Name, Cell: unit.Cell, Dead: unit.Dead,
			Side: unit.Side, Affinity: unit.Affinity,
		})
	}
	return read
}

// read is the battle as this screen is allowed to see it while drawing.
//
// The whole division is these three lines: a live screen has the reading Attach
// took under the mirror's lock and may not go back to the battle for more, and a
// local screen owns its battle and reads it on the spot. Taking a local reading
// lazily rather than storing one is deliberate — a stored reading would have to
// be refreshed at every site that steps the local battle, and a site missed
// there draws a stale board with every test still green.
func (p PlayScreen) read() playReading {
	if p.Live {
		return p.reading
	}
	return readBattle(p.Fight, p.Tags)
}

// unit is the unit an id names, and whether the board has one at all.
func (r playReading) unit(id string) (playUnit, bool) {
	for _, unit := range r.units {
		if unit.ID == id {
			return unit, true
		}
	}
	return playUnit{}, false
}

// begin builds the battle from the pairing in front and runs it up to the
// player's first decision.
func (p PlayScreen) begin(c Context) PlayScreen {
	if p.Live {
		return p
	}
	p.Fight, p.Tags, p.Names = nil, nil, nil
	p.Roster = nil
	p.Events, p.Script, p.Pending = nil, nil, nil
	p.Option, p.Aim, p.Aiming = 0, 0, false
	p.LogFollow, p.LogOffset = true, 0
	p.Err, p.Notes = nil, nil

	if len(p.Home.Units) == 0 || len(p.Away.Units) == 0 {
		return p
	}
	roster, err := p.Home.Take(hex.SideAlly, c.Lib.Characters())
	if err != nil {
		p.Err = err
		return p
	}
	facing, err := p.Away.Take(hex.SideEnemy, c.Lib.Characters())
	if err != nil {
		p.Err = err
		return p
	}
	placed := append(roster, facing...)
	fight, err := battle.New(c.Lib.Books(), p.Seed, placed)
	if err != nil {
		p.Err = err
		return p
	}
	fight.Begin()
	p.Fight = fight
	p.Roster = placed
	p.Tags = tui.Tags(fight.Units())
	p.Names = tui.Names(fight.Units())
	p.collect()
	return p.run()
}

// collect drains whatever the battle has recorded since it was last asked.
//
// The event log is the only contract a reader of a battle has, and this screen
// is a reader like any other: a screen that summed its own strikes would be a
// second place where what a battle did is decided.
func (p *PlayScreen) collect() {
	if p.Fight == nil {
		return
	}
	p.Events = append(p.Events, p.Fight.Drain()...)
}

// run takes every turn that is not the player's, and stops on the one that is.
//
// A skipped turn is stepped past rather than shown: a unit that lost its action
// to control has no decision in it, and a screen that stopped there would ask a
// question with no answers.
func (p PlayScreen) run() PlayScreen {
	if p.Fight == nil {
		return p
	}
	for steps := 0; steps < PlayTurnLimit; steps++ {
		if p.Fight.Finished() {
			p.Pending = nil
			return p
		}
		prompt := p.Pending
		p.Pending = nil
		if prompt == nil {
			opened, err := p.Fight.Advance()
			if err != nil {
				p.Err = err
				return p
			}
			prompt = opened
		}
		p.collect()
		if prompt.Skipped {
			continue
		}
		unit, known := p.Fight.Unit(prompt.Unit)
		if !known {
			// A turn offered to somebody who is not fighting is stepped past
			// rather than reported. It cannot happen — the queue holds units the
			// battle enlisted — and a screen is the wrong place to discover that
			// it did; the step limit above is what stops this being a loop.
			continue
		}
		if unit.Side == p.Side {
			p.Pending = prompt
			p.Option = p.firstAvailable(prompt)
			p.Aim, p.Aiming = 0, false
			return p
		}
		if err := p.engineOrder(prompt); err != nil {
			p.Err = err
			return p
		}
	}
	return p
}

// firstAvailable is the option the cursor starts on: the first the unit may
// actually take, rather than the first declared. A cursor that opened on a skill
// on cooldown would be one press from a refusal on most turns.
func (p PlayScreen) firstAvailable(prompt *battle.Prompt) int {
	for index, option := range prompt.Options {
		if option.Available() {
			return index
		}
	}
	return 0
}

// engineOrder takes the turn the engine would take, which is how the opponent
// plays and how the "let it pick" key answers.
//
// It lands in the script as a decision like the player's own, because the script
// is what the whole battle is rebuilt from: a half of it that was not written
// down would replay as a different battle.
func (p *PlayScreen) engineOrder(prompt *battle.Prompt) error {
	choice, acted := p.Fight.Suggest(prompt)
	if !acted {
		return p.skip(prompt, battle.NoActionReason)
	}
	return p.take(prompt, choice.Skill, choice.Aim)
}

// take and skip are the two things a turn can be spent on, and they are two
// methods rather than one taking a decision so that a decision with a skill and
// no aim — which the engine would refuse and nothing here should be able to
// build — cannot be written at all.
func (p *PlayScreen) take(prompt *battle.Prompt, skill string, aim hex.Offset) error {
	if err := p.Fight.Act(skill, aim); err != nil {
		return err
	}
	return p.record(battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Skill: skill, Aim: hex.At(aim),
	})
}

func (p *PlayScreen) skip(prompt *battle.Prompt, reason string) error {
	decision := battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Passed: true, Reason: reason,
	}
	if err := p.Fight.Pass(decision.PassReason()); err != nil {
		return err
	}
	return p.record(decision)
}

func (p *PlayScreen) record(decision battle.Decision) error {
	p.Script = append(p.Script, decision)
	// A turn taken puts the reader back on the live tail, and this is the one
	// place every turn goes through — the player's, the engine's, the one the
	// "let it pick" key hands over and the pass. Somebody who scrolled back to
	// read what happened and then acted would otherwise be reading a frame from
	// before their own decision, which is the one moment the log is certainly
	// stale. Undo and another seed reset it too, through begin.
	p.LogFollow, p.LogOffset = true, 0
	p.collect()
	return nil
}

// rewind rebuilds the battle from the seed and a shortened script.
//
// It is the whole of undo, and it works because a battle is a pure function of
// its seed and the decisions taken: there is no state to unwind, only a shorter
// list to replay. That is the same property the log's --verify rests on.
func (p PlayScreen) rewind(c Context, script battle.Script) PlayScreen {
	fresh := NewPlayScreen()
	fresh.Seed, fresh.Side = p.Seed, p.Side
	// The pairing goes with it: a rebuild is the same two squads at the same
	// seed, which is the whole of why a shorter script replays as the same
	// battle.
	fresh.Home, fresh.Away = p.Home, p.Away
	fresh = fresh.begin(c)
	if fresh.Fight == nil || fresh.Err != nil {
		return fresh
	}
	replayed, pending, err := fresh.Fight.Replay(script, PlayTurnLimit, nil)
	if err != nil {
		fresh.Err = err
		return fresh
	}
	fresh.Script = replayed
	fresh.Pending = pending
	fresh.Events = fresh.Fight.Drain()
	return fresh.run()
}

// undo drops the player's last decision and everything that followed it.
func (p PlayScreen) undo(c Context) PlayScreen {
	cut := -1
	for index := len(p.Script) - 1; index >= 0; index-- {
		unit, known := p.Fight.Unit(p.Script[index].Unit)
		if known && unit.Side == p.Side {
			cut = index
			break
		}
	}
	if cut < 0 {
		return p
	}
	shortened := make(battle.Script, cut)
	copy(shortened, p.Script[:cut])
	return p.rewind(c, shortened)
}

// Update routes one keystroke.
//
// ⚠️ **Two returns and not three.** A screen with a text field on it has to hand
// back the field's own command — the cursor blink — which is why the skill form
// answers with three; this screen has no field on it at all, so there is nothing
// to carry and an Action says the whole of what it wants.
//
// # The live guards
//
// Six keys are answered differently, or not at all, when Live. Each guard is on
// the key it turns off and there is exactly **one** of them per key, which is
// the rule the "which guard masks" measurement in this repository asked for: a
// second guard further down would mean deleting the first leaves every test
// green.
//
//	n         another seed      ignored — a match's seeds are the room's
//	u         undo              ignored — the opponent has already seen the
//	                            events it would take back
//	ctrl+s    save the log      ignored — Home and Away are empty in a match,
//	                            and the room is what writes a match's log
//	a         let it pick       ignored — it is a decision, and the decision is
//	                            the player's
//	enter/    take the turn     answered rather than taken: an Action, because
//	space                       Mirror.Decide applies nothing and a local Act
//	                            would step the battle twice
//	p         pass              the same, with Acted false
//
// esc, ?, ↑/↓ and the log's own keys are unchanged: reading and scrolling touch
// no turn, and a Back is still the client's to interpret.
//
// # A watcher, and why it needed no seventh guard
//
// ⚠️ **A spectator presses every key here and changes nothing, and it is the
// `p.Pending == nil` return below that says so rather than a guard of its own.**
// A watching mirror is never asking — socket.Mirror.asking refuses one at the one
// derivation there is — so Pending is nil on every turn of the match including
// the host's, and ↑/↓, enter, space, `?`, `a` and `p` all fall past that return
// without reaching the switch under it. The six live guards above cover the rest:
// `n`, `u` and the save key are behind `!p.Live`, and a watcher is Live.
//
// A seventh guard reading PlayScreen.Watching would be a second declaration of
// one invariant, and the note on battle.healingFor's single floor is what says
// what that costs: with two floors, deleting either reddens nothing. What
// Watching decides is the **wording** — → View, waiting, clocks.
func (p PlayScreen) Update(c Context, message tea.KeyPressMsg) (PlayScreen, Action) {
	// Saving is asked before the switch because it answers to more than one
	// keystroke; IsSaveKey is the single declaration of which.
	//
	// ⚠️ The second half of the guard is the library, and it is **one**
	// condition rather than a second guard further down, for the reason the six
	// above are one each: with two, deleting either leaves every test green. A
	// library built from the embedded copy has nowhere to write a log, so
	// SaveBattleLog refuses and the whole of what a press could achieve is
	// forge.ErrNoDataDirectory drawn as a red line — a sentence about how the
	// binary was built, shown to somebody who pressed a key the footer offered
	// them. The footer does not offer it in that state either; the two have to
	// agree, or the ignored key is the program going quiet on a promise.
	if IsSaveKey(message) {
		if p.Live || !c.Lib.HasDataDirectory() {
			return p, Action{}
		}
		return p.save(c), Action{}
	}
	switch message.String() {
	case "esc":
		if p.Aiming {
			p.Aiming = false
			return p, Action{}
		}
		// One step back towards whoever raised this, which the client is what
		// remembers: a screen may not name its own way out even while it has
		// only one raiser. In a match the client turns it into leaving the
		// match, which costs nothing by design — nobody forfeits.
		return p, Action{Kind: Back}
	case "n":
		if !p.Live {
			p.Seed++
			p = p.begin(c)
		}
	case "u":
		if p.Fight != nil && !p.Live {
			p = p.undo(c)
		}
	// The log's own keys, and they are answered **here** rather than under the
	// guard below: the log is drawn between turns and on a battle that has
	// finished, which is exactly when reading back through it is the only thing
	// left to do, and it is drawn while aiming as well — the same reason ? works
	// there. ↑/↓ are the option list's and may not be taken, so this is the pair
	// that already scrolls in this client (the trait description and the picker),
	// rather than a second vocabulary for one idea.
	//
	// [ and ] are aliases for that pair and not a second vocabulary either: they
	// are the same one idea reached by keys every keyboard has. A compact board
	// has no PgUp and no PgDn — a laptop reaches them through a modifier and a
	// sixty-percent board through a layer, and neither is discoverable from a
	// footer naming them — so on such a keyboard the whole log below the frame
	// was as unreachable as it was before it scrolled at all. The brackets are
	// added at **every** site the pair works (here, the trait description and the
	// picker's reading pane), because one site aliased is exactly the second
	// vocabulary the paragraph above refuses. Back is [ and forward is ], which
	// is the direction ↑/↓ reads in and the order the footers print them in.
	case "pgup", "[":
		p = p.scrollLog(c, -1)
	case "pgdown", "]":
		p = p.scrollLog(c, 1)
	}
	if p.Pending == nil || (p.Live && p.Answered) {
		// A live screen that has answered is between turns as far as the player
		// is concerned: the decision has gone, and offering the same one again
		// is a key the screen would have to ignore. → the Answered field.
		return p, Action{}
	}
	switch message.String() {
	case "up", "k":
		p = p.Move(-1)
	case "down", "j":
		p = p.Move(1)
	case "?":
		// The full description of the option under the cursor, on the screen the
		// rest of this tool already raises with this key. It is free here and it
		// costs no turn: the blurb reads the option and nothing else, so raising
		// it and leaving it cannot step the battle.
		//
		// Reached with a prompt open, which is what the guard above this switch
		// buys, and it works while aiming too — the skill is chosen and the cell
		// is not, so "what does this do" is still the live question. An empty
		// option list is refused rather than indexed: a turn with nothing to take
		// is a turn with nothing to describe.
		if len(p.Pending.Options) > 0 {
			return p, Action{Kind: Raise, Target: Blurb, Subject: p.Subject()}
		}
	case "enter", "space":
		if p.Live {
			return p.answer(c)
		}
		p = p.choose(c)
	case "a":
		// The engine's own answer, taken as the player's: somebody who wants to
		// see what it would do here should not have to guess it, and it lands in
		// the script as a decision like any other.
		if p.Live {
			break
		}
		if err := p.engineOrder(p.Pending); err != nil {
			p.Err = err
		} else {
			p.Pending = nil
			p = p.run()
		}
	case "p":
		if p.Live {
			// A pass carries no reason across the wire and none is invented
			// here: battle.NoActionReason is the single declaration of one and
			// the server is what records it, because the server writes the log.
			return p.answering(), Action{Kind: Answer, Answer: PlayAnswer{Acted: false}}
		}
		if err := p.skip(p.Pending, ""); err != nil {
			p.Err = err
		} else {
			p.Pending = nil
			p = p.run()
		}
	}
	return p, Action{}
}

// answer is choose on a battle this screen does not drive: the same two
// questions, answered with an Action instead of with a call into the engine.
//
// It is a second method rather than a branch inside choose because the two
// differ in what they *do* rather than in what they ask — choose steps the
// battle and this one may not — and a single method with the step behind a flag
// would put the one line that must never run on a live screen inside the one
// that always runs.
func (p PlayScreen) answer(c Context) (PlayScreen, Action) {
	option := p.Pending.Options[Clamp(p.Option, 0, len(p.Pending.Options)-1)]
	if !option.Available() || len(option.Aims) == 0 {
		return p, Action{}
	}
	if !p.Aiming {
		p.Aiming, p.Aim = true, 0
		return p, Action{}
	}
	aim := option.Aims[Clamp(p.Aim, 0, len(option.Aims)-1)]
	return p.answering(), Action{Kind: Answer, Answer: PlayAnswer{
		Choice: battle.Choice{Skill: option.Skill, Aim: aim}, Acted: true,
	}}
}

// answering records that this turn has been answered, which is what stops the
// screen offering it a second time. Both a strike and a pass go through it, for
// the reason record is the one place every local turn goes through.
func (p PlayScreen) answering() PlayScreen {
	p.Answered, p.Aiming = true, false
	p.answeredUnit, p.answeredTurn = p.Pending.Unit, p.Pending.Turn
	// A turn taken puts the reader back on the live tail, exactly as record
	// does locally: somebody who scrolled back to read what happened and then
	// acted would otherwise be reading a frame from before their own decision.
	p.LogFollow, p.LogOffset = true, 0
	return p
}

// move walks whichever of the two lists is in front.
// subject is what the description screen raised from here is about: the option
// under the cursor, named by its skill id and by where it sits in this unit's
// offered list.
//
// ⚠️ **It names the skill and hands over nothing of the battle**, which is what
// keeps the promise above the ? key: PlayScreen is the one screen holding
// something the model does not copy, so a describer given a battle to read is a
// redraw that could step a turn. The id is looked up in the library instead,
// because a battle carries a unit's resolved kit and the sentences are about the
// declared skill.
//
// Nothing pending and no options are both ordinary answers rather than errors — a
// turn nobody is being asked about is a turn with nothing to describe — and they
// come back as a subject with Of at nought, which is how the describer is told
// so without looking.
func (p PlayScreen) Subject() Subject {
	if p.Pending == nil || len(p.Pending.Options) == 0 {
		return Subject{Kind: SkillSubject}
	}
	at := Clamp(p.Option, 0, len(p.Pending.Options)-1)
	return Subject{
		Kind: SkillSubject,
		ID:   p.Pending.Options[at].Skill,
		At:   at + 1,
		Of:   len(p.Pending.Options),
	}
}

func (p PlayScreen) Move(by int) PlayScreen {
	if p.Aiming {
		aims := p.Pending.Options[p.Option].Aims
		p.Aim = (p.Aim + by + len(aims)) % len(aims)
		return p
	}
	// Unavailable options are stepped over rather than hidden: a player deciding
	// what to do needs to know a skill exists and is two turns away, and a cursor
	// that could rest on one would be a cursor whose enter does nothing.
	for step := 1; step <= len(p.Pending.Options); step++ {
		next := (p.Option + by*step + len(p.Pending.Options)*len(p.Pending.Options)) % len(p.Pending.Options)
		if p.Pending.Options[next].Available() {
			p.Option = next
			return p
		}
	}
	return p
}

// choose answers whichever question is in front: which skill, and then where.
//
// ⚠️ **The aim list opens every time, including for a skill with one legal
// cell**, and that is a decision reversed rather than an oversight. The rule was
// "a question with one answer is not a decision" — written on this screen and on
// the CLI's chooseAim — and it reads well until the one answer is the thing the
// player wanted to look at before committing. The aim list is where a skill's
// footprint is drawn: which cells the shape catches, who is standing in them,
// what the matchup does there. Skipping it because the *choice* was forced also
// skips the *reading*, and the keystroke that picked the skill has already spent
// the turn by the time anybody notices.
//
// The cost is one keystroke on a forced aim and it is paid deliberately: a turn
// is the one thing in this game that cannot be taken back on a live board, so
// the screen would rather ask twice than commit once unread.
//
// ⚠️ **cmd/hexarena deliberately does NOT follow.** Its own comment says what it
// is — a prompt loop driven from a script or a pipe — and an extra required line
// per single-aim skill is a change to the input format of every script written
// against it, in exchange for a footprint a piped caller does not read. The two
// clients disagree here on purpose, and this is the note that says so.
func (p PlayScreen) choose(c Context) PlayScreen {
	option := p.Pending.Options[Clamp(p.Option, 0, len(p.Pending.Options)-1)]
	if !option.Available() {
		return p
	}
	if !p.Aiming {
		p.Aiming, p.Aim = true, 0
		return p
	}
	aim := option.Aims[Clamp(p.Aim, 0, len(option.Aims)-1)]
	if err := p.take(p.Pending, option.Skill, aim); err != nil {
		p.Err = err
		return p
	}
	p.Pending, p.Aiming = nil, false
	return p.run()
}

// save writes the battle out where the game client can replay it.
//
// It may be pressed at any point rather than only at the end, and that is not a
// concession: a battle stopped halfway is a battle, its script is consistent,
// and re-running it reproduces exactly the half that was played. What a log
// records is what happened, not what finished.
func (p PlayScreen) save(c Context) PlayScreen {
	if p.Fight == nil {
		return p
	}
	path, err := c.Lib.SaveBattleLog(p.Home.ID, p.Away.ID, p.Seed, battle.Log{
		Seed: p.Seed, Roster: p.Roster, Choices: p.Script, Events: p.Events,
	})
	if err != nil {
		p.Err, p.Notes = err, nil
		return p
	}
	p.Err = nil
	// The second note is the whole reason the first one is worth having, and it
	// carries the rebuild warning itself: --verify re-runs against the copy
	// baked into the game binary, so a log written after an edit nobody rebuilt
	// will not verify, and the mismatch would read as corruption.
	//
	// It names the file relative to the data directory, because what goes after
	// --replay is a path somebody has to type and the absolute one is mostly the
	// part they are already standing in.
	//
	// ⚠️ **The empty first argument is unreachable, measured rather than
	// assumed, and it needs no guard.** A library with no directory refuses in
	// SaveBattleLog above and this function has already returned; the key that
	// gets here is not even offered in that state. And if it were reached, the
	// `err == nil` is what carries it: measured, `filepath.Rel("", p)` **errors**
	// for a rooted p — which is what SaveBattleLog returns whenever a directory
	// was rooted — so `relative` keeps the whole path, and it hands back a
	// relative p unchanged, which is already the answer. The one input it
	// mangles is `Rel("", "")`, which comes back ".", and path is "" only when
	// err was non-nil, which returned above.
	// → TestTheReplayNoteIsNeverBuiltWithoutADataDirectory.
	relative := path
	if shortened, err := filepath.Rel(c.Lib.Dir(), path); err == nil {
		relative = shortened
	}
	p.Notes = []forge.Note{
		{Kind: forge.NoteWrote, ID: filepath.Base(path), Path: path},
		{Kind: forge.NoteBattleVerify, Path: relative},
	}
	return p
}

// PlayLogWanted is how many rendered rows the log asks the budget for.
//
// It is what the player has to read: what happened since they last chose. A
// screen that grew a line per event would push the board off the top by the
// third turn, which is why the section asks for a few rows rather than for all
// of them.
//
// ⚠️ **It is a floor of intent and no longer a ceiling**, and it was the ceiling
// that was the defect. `playFit` clamped the log's allotment to this number, so
// the body grew 20 → 42 rows between a 120x24 window and an 80x80 one and the log
// stood still at eight — a tall terminal bought the history nothing. The log now
// takes whatever rows nobody above it in the priority claimed, and this is only
// what it asks for **first**, which is what still makes "everything fits" a
// question with an answer: a window that gives the log its eight rows is a window
// with nothing missing, and one that gives it forty is the same window with room
// to spare.
//
// ⚠️ **It is not a floor in the priority**, and it cannot become one. The log is
// last precisely because it is history rather than state, so a guaranteed eight
// rows would have to be taken off the roster, the board or the order line — and
// every drop height in CLAUDE.md would move. Growing the log may only ever spend
// rows nobody else claimed.
//
// ⚠️ **Rendered rows, not events**, and it used to be events. The two are not
// the same number: tui.Line opens a turn with a blank row of its own, so one
// event arrives as two rows, and eight events measured **eleven rows** in a
// battle a few turns deep.
//
// Renamed off playLogLines because the number no longer says how many lines the
// screen keeps — it says how many it asks for.
const PlayLogWanted = 8

// PlayBodyRoom is how many rows the body may write before frame cuts it.
//
// frame gives the whole screen c.Height - 2 rows, spends the first two on the
// header and the blank under it, and puts the footer below whatever padding is
// left — so the body's own purse is four rows short of the window. Reading it
// off frame's arithmetic rather than writing the number down is the point: a
// second copy of "how many rows are there" is how a screen comes to disagree
// with the frame drawn around it.
func PlayBodyRoom(height int) int { return height - 4 }

// The battle screen budgets its own body, and the reason is that it cannot fit.
//
// Measured at the declared 120x24 floor, where the body's purse is twenty rows:
// the heading is one, tui.Board is a fixed ten, tui.Roster is one plus a row a
// unit, tui.Order is one, the log asks for PlayLogWanted, and the option list is
// one plus a row an option. A legal squad is up to hex.MaxSquadSize a side, so
// **28 rows is the floor for a 5-a-side pairing** before a single blank or log
// line — and a summon puts units on the board past the five the squad brought,
// up to the nine formation slots a side, which is board + roster = 29 on its
// own. There is no arrangement of these sections that fits twenty rows.
//
// So the deliverable was never "make it fit". frame cuts from the **bottom** and
// the option list was the last thing the body wrote, so the one thing the player
// has to see in order to act was the first thing thrown away. That is fixable at
// any content height, and it is the actual defect.
//
// The heading and the turn in front are therefore **reserved**: never dropped,
// never cut. A battle screen that cannot show the moves is not a battle screen.
// Everything else takes what is left, in this order:
//
//  1. The save's own note. It is the answer to a keystroke pressed a moment ago
//     and it names the file that was written, so it outranks the board — and it
//     is **not** reserved, because a pair of notes runs to four or more rows and
//     reserving them could crowd out the option list. See the note under this
//     list for why that count is not written down as a constant.
//  2. tui.Roster, clipped a row at a time. It carries the health and the effects
//     a turn is decided on, and it is the one section that compresses by degrees
//     rather than all at once.
//  3. tui.Board, dropped **whole**, because ten rows of ASCII art have no half.
//     What it says is recoverable: the aim list already prints the occupant
//     beside every cell it offers (PlayScreen.occupant), so the question the
//     board answers is answered again where the player is pointing.
//  4. tui.Order, one row, ahead of the log.
//  5. The log, which asks for PlayLogWanted rendered rows and then takes every
//     row nobody above it claimed. It is last because it is history rather than
//     state, and the two-part answer is what lets a tall window buy the history
//     something without any of the four sections above losing a row.
//
// ⚠️ **How many rows the save note takes is not a fixed number, and two comments
// here used to say it was — disagreeing with each other.** This one read "four"
// and play_test.go read "five" about the same pair. The second note is catalog
// wording wrapped at MinWidth, so it is two rows at a floor of 120 and was three
// at the old 80; the *first* names the file it wrote and therefore carries a
// path, which is free text as long as whoever chose the data directory made it.
// The catalog half moves with the floor, the path half moves with the
// filesystem, and only the catalog half is worth pinning —
// TestEveryFloorWrappedBlockTakesTheRowsItTakes holds it at two.
//
// The log is also the one section a reader can move: it is a frame over the whole
// history rather than a fixed tail, and pgup/pgdown walk it — as do [ and ], the
// aliases the footer advertises, because a compact keyboard has neither page key.
// Following the tail is a state and not an offset — see the fields on PlayScreen
// for why that is the decision the rest of it hangs off.
//
// ⚠️ Nothing here may touch the battle. The plan is computed while drawing, and
// this screen is the one holding a pointer the model does not copy.
//
// playSizes is what each section would spend if it were drawn whole. Every one
// of them is measured off what is actually on the board rather than assumed,
// which is what makes a summoned unit cost a row the way a placed one does.
type playSizes struct {
	// tail is the turn in front: the option list, or the ending once the battle
	// is over. Reserved, so it is not a section the plan chooses about.
	tail int
	// notes is what a save left behind, already wrapped.
	notes int
	// board is tui.Board's rows, and units is tui.Roster's rows without its
	// header — the header goes with the first unit or not at all, because a
	// column heading over nothing is a row spent on nothing.
	board int
	units int
	// log is how many rendered rows the **whole history** comes to, which is
	// what the section would spend if it were drawn whole, the way every other
	// field here is. It used to be the tail capped at eight, and a section
	// reporting its cap as its size is a section that can never be given more.
	log int
}

// playPlan is how much of each section the body draws.
type playPlan struct {
	notes bool
	// board is drawn whole or not at all; roster is a count of units.
	board  bool
	roster int
	order  bool
	// log is how many of the log's rendered lines survive, counted from its end.
	log int
	// notice is the one dim line naming what is not shown and why. It is a row
	// like any other and is budgeted for before the sections are allotted.
	notice bool
}

// playFit is the whole of the arithmetic above.
//
// Two passes rather than one. The first asks whether everything fits, because a
// screen with nothing missing has nothing to say; only when something has to go
// is a row spent saying so, and the second pass allots what is left of the
// smaller purse. If that pass turns out to have nothing worth naming — a log
// tail one line shorter is the tail it always is, not something hidden — the
// first plan is kept and the log gets the row back.
func playFit(room int, sizes playSizes) playPlan {
	left := room - 1 - blockRows(sizes.tail)
	plan, whole := playTake(left, sizes)
	if whole {
		return plan
	}
	squeezed, _ := playTake(left-1, sizes)
	squeezed.notice = true
	if len(playHidden(squeezed, sizes)) == 0 {
		return plan
	}
	return squeezed
}

// playTake is the greedy walk down the priority list, and whole says whether
// every section got all of what it wanted.
func playTake(left int, sizes playSizes) (playPlan, bool) {
	var plan playPlan
	whole := true
	if sizes.notes > 0 {
		if cost := blockRows(sizes.notes); cost <= left {
			plan.notes, left = true, left-cost
		} else {
			whole = false
		}
	}
	// The roster and the board are one pane sharing one blank above them, the
	// way the game client draws them, and the roster is allotted first so the
	// roster pays for it: the blank and the column heading go with the first
	// unit and with no unit at all, because a heading over nothing is a row
	// spent on nothing.
	if sizes.units > 0 {
		if rows := left - 2; rows > 0 {
			plan.roster = min(sizes.units, rows)
			left -= 2 + plan.roster
		}
		if plan.roster < sizes.units {
			whole = false
		}
	}
	// The board goes whole or not at all — ten rows of drawing have no half —
	// and it is never drawn over a roster that is not: the picture without the
	// health is the wrong half of the pane to keep, and it is what the priority
	// already says, since the board is dropped before the roster's last row.
	if sizes.board > 0 {
		if plan.roster > 0 && sizes.board <= left {
			plan.board, left = true, left-sizes.board
		} else {
			whole = false
		}
	}
	if cost := blockRows(1); cost <= left {
		plan.order, left = true, left-cost
	} else {
		whole = false
	}
	// The log asks for PlayLogWanted rows and is answered in two parts, which is
	// what lets it grow without moving anything above it. First its own ask, so
	// that "everything fits" still means something: a window that gives it those
	// rows has nothing missing. Then whatever nobody claimed, because a tall
	// terminal ought to buy the history something and the log is the only section
	// on this screen with more to show than it is ever given.
	if sizes.log > 0 {
		wanted := min(sizes.log, PlayLogWanted)
		if rows := left - 1; rows > 0 {
			plan.log = min(wanted, rows)
			left -= 1 + plan.log
		}
		if plan.log < wanted {
			whole = false
		}
		// The surplus, and only the surplus: the log is last in the priority, so
		// every row still in hand here is a row nobody above it wanted. It cannot
		// take more rows than the history has, or the frame would be padded with
		// nothing.
		if plan.log > 0 && left > 0 {
			spare := min(left, sizes.log-plan.log)
			plan.log += spare
			left -= spare
		}
	}
	return plan, whole
}

// blockRows is what a section of n rows costs: the rows, plus the blank row that
// separates it from whatever is above. A section of nothing costs nothing.
//
// Named in full rather than `block` for the reason drawnRows is: another screen
// in this package already uses that word for a local.
func blockRows(rows int) int {
	if rows <= 0 {
		return 0
	}
	return rows + 1
}

// playHidden is what the notice names, in the order the screen would have drawn
// the sections it is talking about.
//
// The keys rather than the sentence, because the line is composed in one place
// and because the count is what decides whether there is a line at all. A log
// that came back a row or two shorter is **not** in here: the log is a tail by
// design, so a shorter tail is the section working rather than a section
// missing, and naming it would put a notice on nearly every window.
func playHidden(plan playPlan, sizes playSizes) []i18n.Key {
	var hidden []i18n.Key
	if sizes.board > 0 && !plan.board {
		hidden = append(hidden, i18n.PlayHiddenBoard)
	}
	if sizes.units-plan.roster > 0 {
		hidden = append(hidden, i18n.PlayHiddenUnits)
	}
	if !plan.order {
		hidden = append(hidden, i18n.PlayHiddenOrder)
	}
	if sizes.log > 0 && plan.log == 0 {
		hidden = append(hidden, i18n.PlayHiddenLog)
	}
	if sizes.notes > 0 && !plan.notes {
		hidden = append(hidden, i18n.PlayHiddenNote)
	}
	return hidden
}

// hiddenSeparator is what the notice's list is joined with. Punctuation rather
// than a wording: both languages point a list with a comma, and the two ASCII
// cells are the same in either.
const hiddenSeparator = ", "

// notice is the one line saying what is not shown and why.
func (p PlayScreen) notice(c Context, plan playPlan, sizes playSizes) string {
	hidden := playHidden(plan, sizes)
	parts := make([]string, 0, len(hidden))
	for _, key := range hidden {
		// The unit count is the only entry carrying a number, and English needs
		// the singular where Vietnamese does not — hence the second key rather
		// than a plural rule.
		if key != i18n.PlayHiddenUnits {
			parts = append(parts, c.Text(key))
			continue
		}
		if left := sizes.units - plan.roster; left == 1 {
			parts = append(parts, c.Text(i18n.PlayHiddenUnitsOne))
		} else {
			parts = append(parts, c.Text(i18n.PlayHiddenUnits, left))
		}
	}
	return c.Style.Dim.Render(
		c.Text(i18n.PlayHidden, strings.Join(parts, hiddenSeparator)))
}

// playDrawn is every section of this screen drawn whole, before the budget says
// how much of each survives.
//
// It exists so that there is **one** reading of how many rows the log has and how
// many of them the window leaves it: the view that draws the frame and the keys
// that move it both ask this, and a key that scrolled by a different number of
// rows than the screen shows would step over lines nobody ever saw.
//
// ⚠️ Nothing here touches the battle. It is read while drawing and it is read on
// a keystroke, and this is the one screen holding a pointer the model does not
// copy.
type playDrawn struct {
	// tail is the turn in front, and over says the battle has finished — which
	// the footer needs and the sizes do not.
	tail []string
	over bool

	board  []string
	roster []string
	order  string
	// log is the **whole history**, rendered. The frame is a window into it.
	log   []string
	notes []string
}

// drawings measures every section against the board as it stands.
//
// ⚠️ **One reading, taken once, and nothing here asks the battle anything.**
// This function is called while drawing and again on a keystroke, and on a live
// screen the battle underneath it is being stepped by another goroutine. → read.
func (p PlayScreen) drawings(c Context) playDrawn {
	var drawn playDrawn
	read := p.read()
	// The turn in front, read before anything else because it is what the rest of
	// the screen is budgeted around. A finished battle first, because its ending
	// is the answer to the question a prompt would have asked; then the prompt.
	// With neither — between turns, where the engine's own units act — there is no
	// question on the screen and nothing to reserve room for.
	switch {
	case read.finished:
		drawn.tail = []string{c.Style.Emphasis.Render(p.ending(c, read))}
		drawn.over = true
	case p.Live && (p.Pending == nil || p.Answered):
		// ⚠️ **One drawn row that the local screen does not have, and it is not
		// decoration.** Locally an empty tail means "the engine's own units
		// act", which resolves in microseconds; in a match it is the other
		// player thinking, for up to the whole allowance, and a screen with
		// nothing where the moves go reads as frozen.
		//
		// It covers the answered turn as well as the empty one, because from
		// this side of the wire they are one state: the decision has gone and
		// the board is waiting on the other end. → the Answered field.
		drawn.tail = []string{c.Style.Dim.Render(c.Text(p.waiting()))}
	case p.Pending != nil:
		drawn.tail = drawnRows(p.choices(c, read))
	}
	drawn.board = drawnRows(read.board)
	drawn.roster = drawnRows(read.roster)
	drawn.order = c.Style.Dim.Render(read.order)
	drawn.log = p.LogRows(c)
	drawn.notes = p.Wrote(c)
	// ⚠️ **A live battle's notes slot is the refusal**, and the two cannot both
	// be there: a live screen never saves — Home and Away are empty in a match
	// and the room is what writes a match's log — so Wrote is nil on that path
	// and this is not a section being displaced. It is the notes slot rather
	// than a reserved row because a refusal is the answer to something that just
	// happened, exactly as a save's note is, and because the budget already
	// knows how to drop it when the window is too short.
	if refused := p.refusalRows(c); len(refused) > 0 {
		drawn.notes = refused
	}
	return drawn
}

// refusalRows is the protocol refusal a live battle was sent, worded by the
// language book off the name the client handed over.
func (p PlayScreen) refusalRows(c Context) []string {
	if !p.Live || p.LiveRefusal == "" {
		return nil
	}
	var out []string
	// Wrapped against MinWidth rather than the window in hand, for the reason
	// Wrote is: measuring the real terminal gives one sentence two shapes and
	// leaves the width sweep nothing to hold.
	for _, line := range WrapWords(c.Lang.Refusal(p.LiveRefusal), MinWidth-1) {
		out = append(out, c.Style.Bad.Render(line))
	}
	return out
}

// sizes is what each section would spend if it were drawn whole.
func (d playDrawn) sizes() playSizes {
	return playSizes{
		tail:  len(d.tail),
		notes: len(d.notes),
		board: len(d.board),
		// The header is not a unit, and tui.Roster always draws one.
		units: max(len(d.roster)-1, 0),
		log:   len(d.log),
	}
}

// ⚠️ **The three live footers are second wordings and not the local ones with
// clauses deleted.** Context.Footer already states why for the authoring pair:
// dropping a clause out of a rendered line leaves the separators either side of
// it and nothing measures what is left. A live battle names neither `u`, `n` nor
// the save key and gains nothing, so they are shorter than the local three —
// which is a reason to measure them rather than a reason not to.
func (p PlayScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.PlayFooter, SaveKeyLabel())
	// ⚠️ **An offer that can only fail is worse than no offer**, so the two
	// local footers that name the save key have a second wording for a library
	// with nowhere to write. See the guard in Update, which is what makes this a
	// withdrawn offer rather than a footer disagreeing with the keyboard.
	if !c.Lib.HasDataDirectory() {
		footer = c.Text(i18n.PlayNoSaveFooter)
	}
	if p.Live {
		footer = c.Text(i18n.PlayLiveFooter)
	}
	// ⚠️ **A watcher's footer names the scroll keys and the way out, and nothing
	// else, because nothing else is there.** The live footer names ↑/↓, enter, ?
	// and p — four keys a watching screen ignores, every one of them behind the
	// `p.Pending == nil` return in Update — and Context.Footer's own rule is that
	// a footer naming a key the screen ignores is the program promising something
	// it does not do. Here what it would be promising is a turn of somebody else's
	// match.
	if p.Watching {
		footer = c.Text(i18n.PlayWatchFooter)
	}
	if p.Aiming {
		footer = c.Text(i18n.PlayAimFooter)
		if p.Live {
			footer = c.Text(i18n.PlayLiveAimFooter)
		}
	}
	if p.Err != nil {
		return p.heading(c, "") + "\n\n  " + c.Style.Bad.Render(c.Lang.Error(p.Err)), footer
	}
	if p.Fight == nil {
		// A live screen with no battle is between battles of a match, not a
		// client with an empty catalogue: saying a side has to be built there
		// would be this screen answering a question nobody asked.
		if p.Live {
			return p.heading(c, "") + "\n\n  " +
				c.Style.Dim.Render(c.Text(p.waiting())), footer
		}
		return p.heading(c, "") + "\n\n  " + c.Text(i18n.SquadsEmpty), footer
	}

	drawn := p.drawings(c)
	if drawn.over {
		footer = c.Text(i18n.PlayOverFooter, SaveKeyLabel())
		if !c.Lib.HasDataDirectory() {
			footer = c.Text(i18n.PlayOverNoSaveFooter)
		}
		if p.Live {
			footer = c.Text(i18n.PlayLiveOverFooter)
		}
	}
	sizes := drawn.sizes()
	plan := playFit(PlayBodyRoom(c.Height), sizes)
	log := p.logFrame(drawn.log, plan.log)

	body := []string{p.heading(c, p.logPosition(c, len(drawn.log), plan.log))}
	if plan.notice {
		body = append(body, p.notice(c, plan, sizes))
	}
	if plan.board || plan.roster > 0 {
		body = append(body, "")
		if plan.board {
			body = append(body, drawn.board...)
		}
		if plan.roster > 0 {
			body = append(body, drawn.roster[:plan.roster+1]...)
		}
	}
	if plan.order {
		body = append(body, "", drawn.order)
	}
	if len(log) > 0 {
		body = append(body, "")
		body = append(body, log...)
	}
	if len(drawn.tail) > 0 {
		body = append(body, "")
		body = append(body, drawn.tail...)
	}
	if plan.notes {
		body = append(body, "")
		body = append(body, drawn.notes...)
	}
	return strings.Join(body, "\n"), footer
}

// waiting is the line a live screen draws where a player's option list goes,
// which is the one row live mode adds to the drawing.
//
// ⚠️ **One declaration for the two sites that draw it.** It is drawn between
// battles (the whole body, with no battle to show) and between turns (the tail
// row), and the two sentences it can be have to agree: a screen that said
// *"waiting on the other player"* in one place and *"watching"* in the other
// would be telling the reader they were in the match on one screen and not on the
// next.
//
// A watcher is waiting on **both** of them, which is why this is a different
// sentence rather than the live one with a word swapped.
func (p PlayScreen) waiting() i18n.Key {
	// ⚠️ **Before the watching branch**, because a watcher can lose its socket
	// too and "watching — the two of them are deciding" would then be describing
	// a match this client is no longer connected to.
	if p.Reconnecting {
		return i18n.PlayLiveReconnecting
	}
	if p.Watching {
		return i18n.PlayWatchWaiting
	}
	return i18n.PlayLiveWaiting
}

// heading is the screen's title row, and the log's position in the history when
// there is one.
//
// ⚠️ **The position goes here rather than on a row of its own.** A row of its own
// would cost what the budget below spent a whole feature proving this screen has
// not got, and the title is about seventeen cells of the seventy-nine there are.
func (p PlayScreen) heading(c Context, position string) string {
	row := c.Style.Heading.Render(c.Text(i18n.PlayHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.PlaySeed, p.Seed))
	if clocks := p.clocks(c); clocks != "" {
		row += "  " + c.Style.Emphasis.Render(clocks)
	}
	if position == "" {
		return row
	}
	return row + "  " + c.Style.Dim.Render(position)
}

// clocks is the countdown, and nothing at all when no turn is being counted.
//
// ⚠️ **It goes on the heading row rather than on a row of its own, and that is
// the budget below rather than a layout preference.** A live 3v3 already asks
// for more rows than the floor has; the log is what pays for every row anything
// else takes, so a clock row would cost the reader a line of history *and* move
// the frame the history is read through — three lines moved to draw one. The
// heading is the one place on this screen a reading is free, which is the
// argument the log's own position is already here under, and the clock is about
// eighteen cells of the seventy-nine there are.
//
// ⚠️ **Live only, and a battle nobody is being asked about draws none.** The
// countdown is the room's allowance running out; a local battle has no room, no
// allowance and nobody waiting, and a live battle between turns has no open turn
// to count. → PlayClock.Waiting, whose nought is that reading.
// ⚠️ **A WATCHER gets the same two numbers under different words**, and the
// numbers needed no change at all: the client counts down for whichever side the
// open turn belongs to and PlayClockYou is *the half this screen is drawn from*,
// which on a watching mirror is the host's — so `Yours` is the ally half's clock
// and `Theirs` the enemy half's, already, and only the label was a lie. The
// wording names the halves `A` and `E`, which are the labels tui.Tags puts on
// every row of the board and the roster the reader is looking at; a spectator has
// no `you`, and the seat words are on neither.
func (p PlayScreen) clocks(c Context) string {
	if !p.Live {
		return ""
	}
	yours, theirs := playClock(p.Clock.Yours), playClock(p.Clock.Theirs)
	switch {
	case p.Clock.Waiting == PlayClockYou && p.Watching:
		return c.Text(i18n.PlayWatchTurnAlly, yours, theirs)
	case p.Clock.Waiting == PlayClockThem && p.Watching:
		return c.Text(i18n.PlayWatchTurnEnemy, yours, theirs)
	case p.Clock.Waiting == PlayClockYou:
		return c.Text(i18n.PlayClockYours, yours, theirs)
	case p.Clock.Waiting == PlayClockThem:
		return c.Text(i18n.PlayClockTheirs, yours, theirs)
	}
	return ""
}

// playClock is a count of seconds as a clock reads it.
//
// The minutes are not padded and the seconds always are, which is what makes the
// number the same width for a whole minute — a countdown that changed width as
// it ran would shift the log's position along the row beside it every second.
// Nothing below nought exists: a turn whose allowance has run out has none left,
// and the room has already been told.
func playClock(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// logPosition is where the frame sits in the whole history, and nothing when the
// whole of it is on screen.
//
// ⚠️ **Shown whenever rows are hidden, not only while scrolled back.** Half of
// the defect this answers is that nothing on the screen said a history existed:
// eight rows of three hundred were drawn and the other two hundred and ninety-two
// were unreachable by any means. A reader who cannot see that there are three
// hundred rows will not go looking for the key that reaches them.
//
// Nothing is said when the log is not drawn at all — the notice under the heading
// already names it as a section the window is too short for, and a range for a
// frame nobody can see would be a position in a thing that is not there.
func (p PlayScreen) logPosition(c Context, total, room int) string {
	if room <= 0 || total <= room {
		return ""
	}
	start := p.logStart(total, room)
	return c.Text(i18n.PlayLogRange, start+1, start+room, total)
}

// scrollLog moves the log's frame by whole pages, and does nothing at all when
// the history already fits the frame.
//
// A page rather than a row, because the history runs to hundreds of rows and a
// key that had to be pressed two hundred times to reach the opening board is a
// key nobody presses twice.
//
// ⚠️ **Reaching the bottom asks to follow again**, and that is not the sentinel
// the field comments refuse. A reader who scrolls down to the newest row is
// saying they want the newest row, which is a state; storing the number that
// happens to be the newest row today is the thing that goes wrong the moment the
// next event arrives. So the offset goes back to nothing there — nought is also a
// perfectly ordinary offset, meaning the top of the history, and the flag beside
// it is what tells the two apart. That is the whole argument for two fields.
func (p PlayScreen) scrollLog(c Context, pages int) PlayScreen {
	if p.Fight == nil || p.Err != nil {
		return p
	}
	drawn := p.drawings(c)
	room := playFit(PlayBodyRoom(c.Height), drawn.sizes()).log
	total := len(drawn.log)
	if room <= 0 || total <= room {
		// Nothing above the frame, so nothing to scroll to.
		return p
	}
	tail := total - room
	offset := Clamp(p.logStart(total, room)+pages*room, 0, tail)
	if offset == tail {
		p.LogFollow, p.LogOffset = true, 0
		return p
	}
	p.LogFollow, p.LogOffset = false, offset
	return p
}

// drawnRows splits a drawing into the rows it occupies, dropping the empty one a
// trailing newline leaves behind.
//
// That last empty string is the miscount (*pickState).room had to be corrected
// for: frame splits the body on newlines, so a section ending in one is a
// section a row longer than it looks.
//
// Named for what it returns rather than `rows`, which half the screens in this
// package already use as a local: a package function shadowed in most of the
// files that could call it is one nobody reaches for.
func drawnRows(drawn string) []string {
	trimmed := strings.TrimRight(drawn, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// logRows is the whole history, rendered.
//
// ⚠️ **Every event, and it used to be the last few.** The old reading walked from
// the end and stopped at the budget, which meant the rows past the frame were not
// merely off screen — they did not exist, so no key could have reached them and
// the frame had nothing to be a window into. p.Events holds every event a battle
// has emitted (collect appends and never trims), so the history was always there
// and it was the view that threw it away.
//
// Counted in **rendered rows** and not in events, because tui.Line opens a turn
// with a blank row and one event therefore arrives as two rows. Each event is
// rendered exactly as it was before there was a budget — the indent goes on the
// front of whatever tui.Line returned, once, so a turn's blank row still reads as
// a blank row — and the rows are then counted rather than the events.
//
// Nothing here reads the battle: the event log is the only contract a reader of
// one has.
func (p PlayScreen) LogRows(c Context) []string {
	// ⚠️ Built **once**, here, and not once a row. This renders the whole history
	// now rather than the last few rows, so a per-row build would walk the skill,
	// status and passive books again for every event a battle has emitted. It is
	// not on the screen beside p.Tags either, for one reason: ctrl+l toggles the
	// language and this is read every draw, while p.Tags is written once a battle.
	glosses := c.Lang.LogGlosses(
		c.Lib.Skills().Skills(), c.Lib.Statuses().Kinds(), c.Lib.Passives().All(), c.Lib.Bonuses().All())
	var lines []string
	for _, event := range p.Events {
		line := tui.Line(event, p.Tags, glosses)
		if line == "" {
			continue
		}
		lines = append(lines, drawnRows("  "+c.Style.Dim.Render(line))...)
	}
	return lines
}

// logStart is the first row of the history the frame shows.
//
// ⚠️ **Clamped against the total every time it is read**, not only where the
// offset is written. Undo is a shorter script replayed, so the history is rebuilt
// shorter than the one the offset was taken in, and an offset carried across it
// points past the end.
func (p PlayScreen) logStart(total, room int) int {
	if room <= 0 {
		return 0
	}
	tail := max(total-room, 0)
	if p.LogFollow {
		return tail
	}
	return Clamp(p.LogOffset, 0, tail)
}

// logFrame is the rows of the history that are on screen: the tail while the
// reader is following it, and whichever page they scrolled back to otherwise.
func (p PlayScreen) logFrame(rows []string, room int) []string {
	if room <= 0 {
		return nil
	}
	if len(rows) <= room {
		return rows
	}
	start := p.logStart(len(rows), room)
	return rows[start : start+room]
}

// choices is the turn in front: whose it is, what they may do, and where it may
// be pointed once a skill is picked.
func (p PlayScreen) Choices(c Context) string { return p.choices(c, p.read()) }

// choices is Choices over a reading already taken, which is what the drawing
// path has: one reading serves the whole screen, and a second one taken here
// would render the board, the roster and the order line again to answer a
// question about the turn in front.
func (p PlayScreen) choices(c Context, read playReading) string {
	unit, known := read.unit(p.Pending.Unit)
	if !known {
		return ""
	}
	var out strings.Builder
	out.WriteString(c.Style.Label.Render(c.Text(i18n.PlayYourTurn,
		p.Tags[unit.ID], unit.Name, p.Pending.Turn)) + "\n")
	// The id column is measured over the options this turn offers rather than
	// fixed, for the reason menuLabelWidth and every detail pane measure theirs.
	// Over the options and not over the book: the widest id in the game is
	// thirteen cells and this unit may be bringing four short ones, so a
	// book-wide column would spend the summary's room on a skill nobody here can
	// cast.
	width := p.OptionWidth()
	// Clipped to the floor rather than to the window in hand, which is what the
	// caution line and the trait sentences do and for the same reason: measuring
	// the real terminal gives one line two shapes and leaves the width sweep
	// nothing to hold. Below the floor no screen is drawn at all, so the floor is
	// also the narrowest room this row ever really has.
	clip := lipgloss.NewStyle().MaxWidth(max(MinWidth-1-PlayMarkerWidth-width-PlayOptionGap, 0))
	for index, option := range p.Pending.Options {
		marker := "  "
		// ⚠️ An unavailable option keeps its reason and **drops its summary**.
		// The row has one slot and the two answer different questions: the reason
		// is why this cannot be cast, which is the live question the moment a
		// cursor steps over it, and what the skill does is a ? away. Do not
		// "fix" this by drawing both — the second one would be the half that got
		// clipped.
		tail := p.summarise(c, option.Skill)
		if !option.Available() {
			tail = OptionRefusal(c, option)
		}
		line := option.Skill
		if tail != "" {
			line += strings.Repeat(" ",
				width-lipgloss.Width(option.Skill)+PlayOptionGap) + clip.Render(tail)
		}
		switch {
		case !option.Available():
			line = c.Style.Dim.Render(line)
		case index == p.Option && !p.Aiming:
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}
	if !p.Aiming {
		return out.String()
	}
	option := p.Pending.Options[Clamp(p.Option, 0, len(p.Pending.Options)-1)]
	// The splash the aim under the cursor also reaches, worked out before the
	// heading because the heading is where the mark is explained and there is
	// nothing to explain when the shape catches one cell.
	splash := p.splashUnder(c, read, option)
	// The skill behind the option, looked up once for the whole list rather than
	// per row: every mark below is a fact about this one skill's element and its
	// power, and a lookup a row would be the same answer fetched up to nine
	// times. A skill the book cannot find marks nothing, which is the reading
	// summarise gives the same miss — the options come out of a battle built
	// from this library, so it is unreachable, and an aim row is the wrong place
	// to say so.
	declared, _ := c.Lib.Skills().Lookup(option.Skill)
	heading := c.Text(i18n.PlayAimAt, option.Skill)
	if len(splash) > 0 {
		heading += "  " + c.Style.Dim.Render(
			c.Text(i18n.PlayAimSplash, shapeSplashMark, c.Lib.SplashShare()))
	}
	// The matchup legend, on the same heading and on the same rule as the splash
	// one: drawn when a mark below is drawn, and never otherwise. Worked out by
	// walking the rows this list is about to print rather than by asking whether
	// the skill has an element — most turns are neutral against everybody, and a
	// legend for four marks none of which appear is a legend about nothing.
	if p.marksAMatchup(c, declared, read, option, splash) {
		heading += "  " + c.Style.Dim.Render(matchupLegend(c))
	}
	out.WriteString("\n" + c.Style.Label.Render(heading) + "\n")
	for index, cell := range option.Aims {
		marker := "  "
		line := cell.String()
		if held := p.occupant(read, cell); held != "" {
			line += "  " + held
		}
		// After the occupant rather than before it, because the mark is about
		// whoever is standing there: a mark on a row naming nobody would be a
		// claim about an empty cell.
		if mark := p.matchupOn(c, declared, read, cell); mark != "" {
			line += "  " + mark
		}
		if index == p.Aim {
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
		// Under the aim they belong to rather than in a block of their own: a
		// splash cell is a fact about ONE aim, and a list printed beside the
		// others would be read as another cell the cursor could move to.
		if index != p.Aim {
			continue
		}
		for _, caught := range splash {
			row := shapeSplashMark + " " + caught.String()
			if held := p.occupant(read, caught); held != "" {
				row += "  " + held
			}
			// The same mark on the same rule: a splash cell is a unit this cast
			// is about to hit, so the matchup is as much a fact about it as
			// about the cell the cursor is on. A row that carried the occupant
			// and not the mark would be the one place a reader had to work it
			// out for themselves.
			if mark := p.matchupOn(c, declared, read, caught); mark != "" {
				row += "  " + mark
			}
			out.WriteString("    " + c.Style.Dim.Render(row) + "\n")
		}
	}
	return out.String()
}

// marksAMatchup says whether any row this aim list is about to draw carries a
// mark, which is what decides whether the heading explains them.
//
// It walks the same cells in the same order as the list itself rather than
// asking a cheaper question — "does this skill have an element" would draw the
// legend on every neutral turn, and "is anybody dual" would miss the ordinary
// single weakness that is most of what a reader sees. The cost is one pass over
// at most nine cells.
func (p PlayScreen) marksAMatchup(c Context, declared skill.Skill, read playReading,
	option battle.Option, splash []hex.Offset) bool {
	for _, cell := range option.Aims {
		if p.matchupOn(c, declared, read, cell) != "" {
			return true
		}
	}
	// The splash rows belong to the aim under the cursor alone, which is the
	// only one whose caught cells are drawn, so they are the only ones that can
	// put a mark on the screen.
	for _, caught := range splash {
		if p.matchupOn(c, declared, read, caught) != "" {
			return true
		}
	}
	return false
}

// splashUnder is every cell but the primary that the option under the cursor
// catches from the aim the cursor is on.
//
// ⚠️ **It is resolved from the aim and it has to be.** The shape diagram on the
// authoring screen walks from forge.ShapeDiagramCell, a fixed cell chosen so
// eight of the nine shipped shapes draw in full; a battle is asking about *this*
// aim on *this* board, where a two-step chain pointed near an edge really does
// lose its far cell. Drawing the diagram's footprint here would be a picture of
// a different board, and it would promise a cell the resolution drops.
//
// Nothing at all for a single-cell shape, which is most of them: the rows exist
// to answer "who else does this reach", and a shape that reaches nobody else has
// no answer to draw.
//
// A skill the book cannot find catches nothing rather than reporting an error,
// which is the reading summarise gives the same miss two hundred lines down: the
// options come out of a battle built from this library, so it is unreachable,
// and an aim row is the wrong place to say so.
func (p PlayScreen) splashUnder(c Context, read playReading, option battle.Option) []hex.Offset {
	if len(option.Aims) == 0 {
		return nil
	}
	aim := option.Aims[Clamp(p.Aim, 0, len(option.Aims)-1)]
	// The frame the shape is walked in is the CASTER's half, so it is read off
	// the unit being prompted rather than off p.Side: the two are the same in
	// every battle this screen opens, and reading the one that is true by
	// definition rather than the one that happens to agree is what stops this
	// from drawing the wrong cells the day it is not. → package pattern's doc.
	//
	// ⚠️ **Off the READING and not off the battle.** This used to call
	// p.Fight.Unit(p.Pending.Unit) right here, which is a read of the mirror's
	// battle taken while drawing, on a live screen, outside the only lock there
	// is — the defect readBattle exists to remove, rebuilt in the one place the
	// rule was not being looked at. The reading already carries every unit's
	// side; taking it from there costs nothing and cannot race.
	caster := hex.SideAlly
	if p.Pending != nil {
		if unit, known := read.unit(p.Pending.Unit); known {
			caster = unit.Side
		}
	}
	coverage, err := c.Lib.AimCoverage(option.Skill, aim, caster)
	if err != nil {
		return nil
	}
	return coverage.Splash
}

// The two fixed columns a row spends before its summary: the cursor marker, and
// the gap between the id column and whatever follows it.
const (
	PlayMarkerWidth = 2
	PlayOptionGap   = 2
)

// optionWidth is the id column, measured over the turn's own options.
func (p PlayScreen) OptionWidth() int {
	width := 0
	for _, option := range p.Pending.Options {
		if drawn := lipgloss.Width(option.Skill); drawn > width {
			width = drawn
		}
	}
	return width
}

// summarise is what a skill does, in the one line that fits beside its id.
//
// The whole reason the list is worth reading: an id is a name, and nothing in
// "venoshock" says it is the skill that doubles into a poison. It is
// i18n.Lang.SummariseSkill rather than anything assembled here — this screen may
// hold no wording of its own, and the compact line is tied to the full
// description by a test in that package rather than by a screen's good
// intentions.
//
// A skill the book cannot find summarises as nothing rather than as an error.
// The options come out of the battle, which was built from the same library, so
// a miss is not reachable; and a row is the wrong place to report that it was.
func (p PlayScreen) summarise(c Context, id string) string {
	declared, err := c.Lib.Skills().Lookup(id)
	if err != nil {
		return ""
	}
	return c.Lang.SummariseSkill(declared, c.Lib.Patterns(), c.Lib.Statuses())
}

// OptionRefusal is why an option cannot be taken, in the reader's own language.
//
// **One place, and it is exported so that it stays one.** battle.Option carries a
// Reason of its own and every renderer in this repository used to print it, but
// that sentence is built inside internal/core, which may not import internal/i18n
// — so it is English wherever it is drawn, including on a Vietnamese screen. What
// the engine hands over instead is the enum and the three counts behind the
// sentence, and turning those back into words is a switch: a switch copied into
// the next client is the second answer that drifts, so there is one here and
// callers ask it.
//
// ⚠️ **A greyed row with a cooldown of nought is what this is really for.** While
// every refusal was a cooldown the row explained itself — it said three and the
// reader waited three — and an untranslated countdown is a small thing. A skill
// gated on the caster's own reserve has no cooldown to count down, so the row was
// a skill the reader could see, could not use, and was told nothing about. The
// fuel wording carries all three of its facts for that reason: how much is
// wanted, of what, and how much is actually held.
//
// The status is named through its **gloss** — the Vietnamese name the reference
// screens and the battle log put beside a data id — falling back to the id, which
// is what English gets and what an unglossed status gets in either language. That
// fallback is this package's rule for every data id it draws and not a special
// case here.
//
// An option blocked for a reason this does not know keeps the engine's own
// sentence. That is the honest degrade: English is worse than Vietnamese and both
// are better than a blank column where the reason was.
func OptionRefusal(c Context, option battle.Option) string {
	switch option.Blocked {
	case battle.BlockUnknownSkill:
		return c.Text(i18n.PlayBlockedUnknown)
	case battle.BlockCooldown:
		if option.Turns == 1 {
			// One turn is its own wording rather than a plural rule: English
			// needs the singular and Vietnamese does not.
			return c.Text(i18n.PlayBlockedCooldownOne)
		}
		return c.Text(i18n.PlayBlockedCooldown, option.Turns)
	case battle.BlockFuel:
		// The gloss alone, not the id with the gloss beside it: this row's own
		// first column is already a bare id, and a second one inside the sentence
		// would spend the width that carries the two counts.
		name := option.Status
		if kind, err := c.Lib.Statuses().Lookup(option.Status); err == nil {
			if named := c.Lang.StatusName(kind); named != "" {
				name = named
			}
		}
		return c.Text(i18n.PlayBlockedFuel, option.Need, name, option.Held)
	case battle.BlockSpent:
		// The allowance alone, without the status behind it. The counter is
		// bookkeeping — a reader never chose to hold it and cannot spend it on
		// anything else — so naming it would spend the row's width on a word that
		// answers no question.
		return c.Text(i18n.PlayBlockedSpent, option.Need)
	case battle.BlockNoReach:
		return c.Text(i18n.PlayBlockedNoReach)
	}
	return option.Reason
}

// occupant is the tag and name standing on a cell, so an aim reads as somebody
// rather than as a coordinate.
func (p PlayScreen) occupant(read playReading, cell hex.Offset) string {
	unit, standing := read.standing(cell)
	if !standing {
		return ""
	}
	return p.Tags[unit.ID] + " " + unit.Name
}

// standing is the live unit a cell holds, and whether it holds one at all.
//
// Separate from occupant because two questions are asked about the same cell and
// only one of them is about words: the aim row wants a tag and a name, and the
// matchup mark wants the unit's affinity. Walking the reading twice would be two
// answers to "who is there", and the day they disagreed the row would name one
// unit and mark another.
func (r playReading) standing(cell hex.Offset) (playUnit, bool) {
	for _, unit := range r.units {
		if unit.Dead || unit.Cell != cell {
			continue
		}
		return unit, true
	}
	return playUnit{}, false
}

// ending is how the battle finished, in the words the game client uses for it.
func (p PlayScreen) ending(c Context, read playReading) string {
	switch read.outcome {
	case battle.Victory:
		if read.winner == p.Side {
			return c.Text(i18n.PlayWon)
		}
		return c.Text(i18n.PlayLost)
	case battle.Stalemate:
		return c.Text(i18n.PlayDrawn)
	default:
		return c.Text(i18n.PlayEmptied)
	}
}

// wrote is the line a save leaves behind, in the shape every other write in this
// client reports itself.
// It hands back the rows rather than a block, because the budget above counts
// them: a section that reported itself as one string would have to be measured
// twice, once to place it and once to draw it.
func (p PlayScreen) Wrote(c Context) []string {
	if len(p.Notes) == 0 {
		return nil
	}
	var out []string
	for index, note := range c.Lang.Notes(p.Notes) {
		style := c.Style.Dim
		if index == 0 {
			style = c.Style.Good
		}
		// Wrapped against MinWidth rather than the window in hand, for the
		// reason the fight's caution is: measuring the real terminal would give
		// one sentence two shapes and leave the width sweep nothing to hold.
		for _, line := range WrapWords(note, MinWidth-1) {
			out = append(out, style.Render(line))
		}
	}
	return out
}
