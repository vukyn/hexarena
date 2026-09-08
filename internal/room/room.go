// Package room is a PvP match as a **state machine over messages**, with no I/O
// of its own: messages and prompts in, messages and decisions out.
//
// It speaks internal/wire and declares no message of its own. It owns one
// *battle.Battle at a time, the series that battle is one of, and the gate a
// peer gets past to sit down. A Room opens no socket, starts no goroutine, takes no mutex,
// and — see below — reads no clock. Two fake clients therefore drive a whole
// match in-process, at the speed of the engine, which is the reason it was built
// this way round: a server with the transport in the middle of the state machine
// would make this the least-tested code in a repository whose whole method is
// measurement.
//
// ⚠️ **Registry, in this package, is the one thing here that does hold a
// goroutine and a mutex**, and the two are kept apart by receiver rather than by
// file: TestNoRoomMethodTouchesTheMutex refuses a mutex, a channel or a goroutine
// on any method of Room, wherever it is written. → Registry, and the note there
// on why it is not a package of its own.
//
// # The room reads no clock, and that is the load-bearing shape
//
// ⚠️ **A timeout is an INPUT, not a reading.** The room never asks what time it
// is. Whoever owns the transport counts the allowance down and calls TimedOut to
// say it ran out, and the room applies a Pass with a single constant reason
// (TimeoutReason). So `time` is not imported anywhere in this package, and
// TestTheRoomReadsNoClock holds that mechanically with an AST walk over this
// directory.
//
// ⚠️ **A timeout is announced and nothing more — it does not count towards
// anything.** It used to: three of a seat's own allowances running out in a row
// forfeited the match, and TimeoutLimit, the per-seat tally and that branch are
// all gone. What passing the turn buys is that the match *progresses*, which is
// the whole point of the input — without it a room waits forever on somebody who
// never answers. What the counting bought was nothing the board does not already
// carry: a player who walks away from the keyboard loses on the board, because
// the opponent keeps acting and kills the passing units, and if both walk away
// the turn cap draws it.
//
// The per-turn allowance is *configuration* the room carries and hands to
// clients on wire.Welcome. The room does not count it down and has no opinion
// about how long it has been.
//
// ⚠️ internal/wire's own clock test says in its comment that a room "does need a
// clock". That was the expectation when the protocol landed and it turned out to
// be wrong: handing the timeout in as an input costs nothing and buys a package
// with no clock in it, so this one inherits wire's ban rather than escaping it.
//
// # What is deliberately NOT in here
//
// So a reader does not go looking for it:
//
//   - **The registry of many rooms, and the one-goroutine-per-room rule.** Not
//     in a *Room*, which is what this list is about: concurrency did not belong
//     in the same commit as "this has no I/O", so it landed in the one after it,
//     as Registry. A room owning its battle in one goroutine and sharing it with
//     nothing is a property of the thing that *holds* rooms, and the mutex is
//     the registry's. **Nothing on a Room is safe for concurrent use and nothing
//     on it needs to be** — the registry is what makes that true rather than
//     what makes it a problem.
//   - **The WebSocket**, and everything that makes a peer a peer: a connection,
//     a seat token, a rejoin. wire.CodeRoomUnknown is the **registry's** refusal
//     and no room ever sends it.
//   - **Writing the finished match out as a battle.Log.** The room holds every
//     decision the engine took only through the engine; a log writer is another
//     cursor over the record, which is exactly why Since exists.
//   - **Seating a spectator.** The *reading* half is here — Since, an
//     append-only record of the bodies a watcher is owed, and watch.go for the
//     whole of it — and the half that is not is everything that decides somebody
//     is watching: the registry's Watch, the transport holding a connection with
//     no seat, and a client that draws a match it is not playing. ⚠️ A watcher is
//     **not** a third seat and must never become one, because the order the two
//     seats are visited in reaches the roster and the roster decides a speed tie.
//   - **The contested-speed-group alternation.** → Config.HomeFor, which says
//     what is implemented and what is deferred and why.
//
// # Reading the battle
//
// The battle is read through Battle.Since and a cursor, and ⚠️ **Drain is never
// called in this package** — TestNothingHereDrainsTheBattle holds that with the
// same walk the clock test uses. Drain empties its consumer's cursor into the
// battle itself, and a room with two players, a log to write and a watcher is
// exactly the multi-consumer case it cannot serve. There is one cursor into the
// battle, the one that turns a turn's events into the digest on wire.Turn; the
// point of reading it this way is that the other consumers need no change here
// and cannot disturb it.
//
// ⚠️ **A watcher does NOT get a second cursor into the battle**, and that is
// worth being exact about because it is the obvious move. It reads the room's
// own record of *bodies* (→ watch.go), not the battle's record of events: what a
// client is owed is the decision and the digest, and rebuilding a
// battle.Decision out of battle.Events would be a second derivation of a
// recorded string. A watcher read written against r.cursor would take the
// players' own events out from under them —
// TestAMatchPlayedWithAWatcherReadingIsTheSameMatch is the net for it.
//
// # What a client is handed, and what it is not
//
// A client is a **mirror**: it holds its own battle built from the seed and
// roster on wire.Start, and applies the decisions on wire.Turn. So the room
// hands over *decisions and digests*, never events — the client computes the
// events by computing the battle, and the digest is what makes that a check
// rather than a hope. A divergence is loud on the turn it happens.
package room

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// TimeoutReason is the note recorded on a turn the room passed because an
// allowance ran out, and it is a **single constant** for the reason
// battle.Decision's own documentation gives: two callers supplying different
// words for the same choice would make a replay diverge from the log it is
// replaying.
//
// It travels on wire.Turn inside the decision, so the mirror records the same
// string rather than inventing one. ⚠️ It is not glossed yet — internal/tui
// prints an event's Note raw, so today it reads "timeout" in both languages.
// → TODO.md, under the client's wordings.
const TimeoutReason = "timeout"

// Deps is what a room is handed rather than what it decides: the parsed data, and
// the version this binary announces.
type Deps struct {
	// Books is the parsed data the battle reads.
	Books battle.Books
	// Characters is the cast book, which is what resolves a squad — a placement
	// is a reference and Take is what turns it into units.
	Characters *cast.Book
	// Version is what this binary announces, and the gate checks a peer against
	// it. It is passed in rather than computed here because a build string is
	// stamped at build time and read by the binary's own main; wire.Local is
	// what assembles one.
	Version wire.Version
	// Tokens makes a seat token, and is what a rejoin rests on.
	//
	// ⚠️ **It is injected rather than called for here, because a room has no
	// randomness of its own and is not getting any.** Everything else this
	// package does is a state machine over messages — no clock, no rand, no I/O —
	// which is what lets the room be driven a message at a time in a test and
	// replayed from a log. A `crypto/rand` read inside Join would put the first
	// unrepeatable thing into it.
	//
	// ⚠️ **Nil is a supported answer and means no rejoin.** The room seats
	// clients exactly as before and issues no token, so a caller that has not
	// thought about tokens gets the behaviour it had rather than a panic or a
	// half-feature. cmd/hexarena-host supplies one; a test that does not care
	// leaves it nil.
	Tokens func() (wire.SeatToken, error)
}

// Outbound is one message the room wants sent, and the seat it is for.
//
// Every recipient gets its own Outbound rather than one message addressed to the
// room, because the two seats' messages genuinely differ: a wire.Welcome names
// the seat it went to, and a wire.Start carries the *side* this client plays,
// which is the opposite half of the board for the other one. A "send to
// everybody" shape would only fit the messages that happen to be identical.
type Outbound struct {
	// To is the seat this message is for, and is the zero Seat on a refusal at
	// the gate — refusing is what stops a seat being handed out, so there is no
	// seat to name and the transport answers the connection it read the hello
	// from.
	To   wire.Seat
	Body wire.Body
}

// peer is one seated client: what it brought, and nothing else.
//
// ⚠️ There is no tally of missed allowances on it. There was, and the branch it
// fed — three in a row forfeiting the match — is gone; a timeout announces and
// passes the turn. → the package comment.
type peer struct {
	taken bool
	name  string
	squad placement.Squad
	// token is what this seat's client shows to take it back, and is empty when
	// the room was built with no way to make one. → Deps.Tokens.
	//
	// ⚠️ It is compared in constant time and never logged, never drawn and never
	// recorded: it is on the peer beside the squad because a seat is what it
	// belongs to, and it leaves the room only in the welcome addressed to the
	// client that owns it.
	token wire.SeatToken
}

// Room is one match: a series of battles between two seats.
//
// It is not safe for concurrent use, deliberately — see the package comment.
type Room struct {
	config Config
	deps   Deps

	seated [seatCount]peer

	// The series.
	played   []BattleResult
	standing [seatCount]int
	result   Result

	// The battle in progress. fight is nil before the first battle and after the
	// match ends.
	fight  *battle.Battle
	index  int
	home   wire.Seat
	seed   uint64
	turns  int
	capped bool
	// cursor is this room's one read position in the battle's record. → the
	// package comment on Drain.
	cursor int

	// record is the battle being written out as a battle.Log, and logCursor is
	// its own read position — the second consumer the package comment says Since
	// exists for.
	//
	// ⚠️ **It starts at nought and `cursor` does not**, which is the whole reason
	// this is a separate position rather than a second use of the same one. The
	// room's cursor is set to Recorded() after the opening board, because no
	// wire.Turn carries the opening and a mirror produces it itself; a log has to
	// carry it, because `--verify` re-runs from the seed and compares **every**
	// event from the first.
	//
	// ⚠️ **Where it stops is load-bearing for a capped battle.** A capped log has
	// no Ended event and it still verifies — but only when the record includes
	// the capped turn's own turn_began, which is where this stands, because
	// settle advanced into that turn before deciding not to ask about it and
	// Replay advances into it too. Stopping one event earlier reads 43 recorded
	// against 44 re-run and fails on the count. A "tidier" stop is wrong.
	record    battle.Log
	logCursor int

	// prompt is the open turn and onTurn the seat whose answer is due, which is
	// -1 whenever nobody is being asked anything.
	prompt *battle.Prompt
	onTurn int

	// skipped is how many prompts the room walked past because the unit had
	// already lost its action.
	//
	// It is exposed (Skipped) for one reason and it is not a debug counter: "a
	// Skipped prompt starts no clock" is a claim about a loop that leaves no
	// other trace — a skipped turn produces no decision and therefore no
	// message — so without a count the claim would be held by nothing, and a
	// test asserting it would pass on a battle that happened to contain no
	// skipped turns at all. The same shape as the scan counts in the two AST
	// walks and as cmd/hexarena-tui's screenCount.
	skipped int

	// watched is the append-only record of every body a watcher has to be
	// handed, and the room **writes it and never reads it** — no behaviour here
	// branches on it, which is what makes a watcher unable to change the match
	// it is watching. → watch.go, which is the whole of the room's watcher half,
	// and Since, which is the only read.
	watched []wire.Body

	// drafting is the ban and pick this room runs before its battle, and nil in
	// a room that does not draft. → draft.go, which is the whole of the room's
	// draft half.
	drafting *draft.Draft
	// draftCursor is this room's one read position in the draft's record, the
	// same shape cursor above is for the battle's.
	draftCursor int
}

// New sets a room up. It validates the configuration and the data, so a room
// that cannot run a match fails here rather than when somebody joins it.
//
// ⚠️ **A room that drafts builds its draft here, and that is where the pool is
// checked against the format** — which is the one part of the configuration
// Config.Validate cannot see, because draft.Fits needs a pool, a pool needs the
// cast book, and Validate has no Deps. So the whole question "could this room
// ever finish its draft" is answered by draft.New in one call: the pool seats the
// format, every character in it has an id, and no id is in it twice. A room whose
// draft could not finish therefore fails **before a code is handed out** rather
// than halfway through somebody's ban and pick, which is the arrangement
// internal/draft's package comment rests on.
func New(config Config, deps Deps) (*Room, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if deps.Characters == nil {
		return nil, fmt.Errorf("a room cannot resolve a squad without the cast book")
	}
	opened := &Room{config: config, deps: deps, onTurn: -1}
	if config.Drafts {
		drafting, err := newDraft(config, deps)
		if err != nil {
			return nil, fmt.Errorf("a room that drafts: %w", err)
		}
		opened.drafting = drafting
	}
	return opened, nil
}

// Config is the room's configuration, for a caller that has to hand a client
// something it was set up with.
func (r *Room) Config() Config { return r.config }

// Awaiting is the seat whose answer is due and whether the room is waiting on
// anybody at all.
//
// ⚠️ **This is what a transport starts its allowance on, and it is the whole of
// the room's side of the clock.** It is never set across a skipped prompt,
// because the room walks past those itself, so "a Skipped prompt starts no
// clock" is a property of this state machine rather than a rule the transport
// has to remember. It is also false between battles and once the match is over.
//
// ⚠️ **In a room that drafts this answers the DRAFT's open decision first**, and
// the arrange phase — which has two decisions pending against this one seat — is
// serialised into it rather than widening what a reading can hold.
// → draftAwaiting, which carries that decision and the two things it costs.
func (r *Room) Awaiting() (wire.Seat, bool) {
	if r.draftOpen() {
		return r.draftAwaiting()
	}
	if r.onTurn < 0 || r.prompt == nil {
		return "", false
	}
	return seats[r.onTurn], true
}

// Pending is the open turn: whose unit is acting and what it may do, which is
// what a server hands a client that lost track of it. Nil when nothing is open.
func (r *Room) Pending() *battle.Prompt { return r.prompt }

// Result is the match. Its Verdict is VerdictUnfinished until the match ends.
func (r *Room) Result() Result { return r.result }

// Finished reports whether the match is over, by any of the routes.
func (r *Room) Finished() bool { return r.result.Verdict.Over() }

// Played is every battle of the series the room has recorded, in order.
func (r *Room) Played() []BattleResult { return r.played[:len(r.played):len(r.played)] }

// Skipped is how many prompts the room walked past because the unit had already
// lost its action. → the field's own comment for why it is exposed.
func (r *Room) Skipped() int { return r.skipped }

// Deliver hands the room one message from a seated peer and returns everything
// the room says back.
//
// It takes wire.Act, wire.Pass and wire.Decide, which are the three messages a
// client sends once it is in. Anything else — including a wire.Hello, which goes
// to Join, and any of the five server-bound bodies, which is a peer speaking the
// wrong direction — is answered with wire.CodeUnknownMessage, because that is
// what this protocol's ten codes have for a message that does not belong here.
//
// ⚠️ **`from` is where the seat comes from and it is the only place any of the
// three read one.** wire.Act carries no unit, wire.Pass carries nothing at all
// and wire.Decide carries no seat — all three by the same design — so the seat is
// a fact about the connection the transport read the message off, and this
// parameter is its one statement. → decideFrom, where that is load-bearing
// rather than incidental.
func (r *Room) Deliver(from wire.Seat, body wire.Body) ([]Outbound, error) {
	index, seated := indexOf(from)
	// A peer with no seat is not on turn, which is the closest true thing the
	// ten codes can say: there is no "you are not in this room" among them, and
	// the registry that would own such a refusal is a later item.
	if !seated || !r.seated[index].taken {
		return r.refuse(from, wire.CodeNotYourTurn), nil
	}
	switch message := body.(type) {
	case *wire.Act:
		return r.answerFrom(index, *message)
	case wire.Act:
		return r.answerFrom(index, message)
	case *wire.Pass:
		return r.passFrom(index)
	case wire.Pass:
		return r.passFrom(index)
	case *wire.Decide:
		return r.decideFrom(index, message.DraftDecision)
	case wire.Decide:
		return r.decideFrom(index, message.DraftDecision)
	}
	return r.refuse(from, wire.CodeUnknownMessage), nil
}

// answerFrom is a seat spending its turn.
func (r *Room) answerFrom(index int, act wire.Act) ([]Outbound, error) {
	if r.onTurn != index || r.prompt == nil {
		return r.refuse(seats[index], wire.CodeNotYourTurn), nil
	}
	aim, aimed := act.Aim.Offset()
	if !aimed {
		return r.refuse(seats[index], wire.CodeIllegalAction), nil
	}
	prompt := r.prompt
	if err := r.fight.Act(act.Skill, aim); err != nil {
		// The engine already refused it — a skill the unit does not hold, one on
		// cooldown, an aim outside the cells it listed — so this code is the
		// engine's no travelling back rather than a second reading of the rules.
		//
		// The turn stays open: an illegal action is not an answer, so the seat is
		// still the one being asked.
		return r.refuse(seats[index], wire.CodeIllegalAction), nil
	}
	return r.resolved(battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Skill: act.Skill, Aim: act.Aim,
	})
}

// passFrom is a seat giving its turn up.
//
// The reason is left **empty** rather than worded here, which is the one-source
// rule taken to its end: battle.Pass supplies "passed" for a decision that did
// not say, so the room adds no second spelling of it and the mirror — which
// applies the decision through Decision.PassReason — cannot disagree.
func (r *Room) passFrom(index int) ([]Outbound, error) {
	if r.onTurn != index || r.prompt == nil {
		return r.refuse(seats[index], wire.CodeNotYourTurn), nil
	}
	prompt := r.prompt
	if err := r.fight.Pass(""); err != nil {
		return nil, fmt.Errorf("pass %q: %w", prompt.Unit, err)
	}
	return r.resolved(battle.Decision{Unit: prompt.Unit, Turn: prompt.Turn, Passed: true})
}

// TimedOut tells the room that the allowance for the open prompt ran out.
//
// ⚠️ It is an **input**. The room does not know how long anything took and does
// not ask; the transport owns the countdown and this is it reporting the result.
// What enters the battle is a Pass with TimeoutReason, never a timestamp and
// never a duration, so a PvP log stays exactly as verifiable as one from a
// battle nobody was waiting on and --verify cannot tell a timed-out match from
// any other.
//
// ⚠️ **It announces and nothing more: nothing is counted and nothing is
// forfeited.** Three in a row used to end the match, and that is gone —
// → the package comment for what the counting was buying and why the board
// already carries it. What survives is the pass, which is what makes the match
// progress rather than wait forever on somebody who never answers.
//
// ⚠️ **It needs no message of its own, and that is worth being exact about.**
// The pass carries TimeoutReason, that reason is part of the battle.Decision,
// and the decision goes out on wire.Turn — where Decision.Reason is tagged
// `json:"reason,omitempty"`. So the mirror is *already told* that the turn was
// lost to a clock, by the one declaration of it that travels, and a second
// message would be a second spelling of a fact both peers already hold.
// TestATimeoutTellsTheMirrorWithNoMessageOfItsOwn is what measures that rather
// than asserting it here.
//
// A timeout for a seat that is not being asked anything is **refused**, and the
// refusal matters more than it did: with no tally to protect, what it protects
// is the turn itself — a transport reporting a spurious timeout must not spend
// the answer of a seat the room is not asking. It is also what makes a Skipped
// prompt untimeoutable: the room never leaves one open, so there is never an
// allowance to run out on one.
// ⚠️ **A timeout during the ban and pick does NOT pass anything — it closes the
// room**, which is the one place the paragraphs above do not hold, and the reason
// is that there is nothing honest to pass with: a side that never picked has no
// squad to fight with. → draftTimedOut, and wire.ClosureDraftExpired.
func (r *Room) TimedOut(seat wire.Seat) ([]Outbound, error) {
	index, seated := indexOf(seat)
	if !seated || !r.seated[index].taken {
		return r.refuse(seat, wire.CodeNotYourTurn), nil
	}
	if r.draftOpen() {
		return r.draftTimedOut(seat)
	}
	if r.onTurn != index || r.prompt == nil {
		return r.refuse(seat, wire.CodeNotYourTurn), nil
	}
	prompt := r.prompt
	if err := r.fight.Pass(TimeoutReason); err != nil {
		return nil, fmt.Errorf("pass %q on a timeout: %w", prompt.Unit, err)
	}
	return r.resolved(battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Passed: true, Reason: TimeoutReason,
	})
}

// Left tells the room that a peer went away.
//
// ⚠️ Whether a peer has really gone or is merely slow is the **transport's**
// judgement and not the room's, exactly as a timeout is. A reconnect window
// would sit in front of this call rather than inside it — the design record
// holds the seat for a rejoin, and the seat token that makes a rejoin possible
// is its own TODO.md item — so this is the terminal case: the transport has
// decided.
//
// ⚠️ **Leaving announces and costs nothing.** The match ends as
// VerdictAbandoned, which is not a win, not a draw and not a forfeit: the seat
// that went away is recorded and neither seat is charged with anything. A player
// who is losing can therefore leave at no cost, and on a LAN between friends the
// enforcement of that is social — a stated cost rather than a gap.
// → README.md § PvP over a LAN.
//
// Before the match starts there is no match to end, so the seat is simply freed
// and the room goes back to waiting for a second player. A reconnect window
// therefore sits **in front of** this call and not inside it — → TODO.md, under
// the seat token.
//
// ⚠️ **A draft in progress IS a match in progress here, and that branch had to be
// added rather than being already right.** A draft runs with no battle open and
// nothing played, so the pre-match arm below matched it: the seat was freed, the
// room went back to waiting for a joiner — with the departed side's bans and
// picks still in the draft, and the peer still there holding an open decision
// nobody was coming to take. That is exactly the hang abandon's own comment says
// a wire.Closed exists to prevent. The ending itself is unchanged: the existing
// ClosureLeft path, VerdictAbandoned, nobody charged with anything.
func (r *Room) Left(seat wire.Seat) ([]Outbound, error) {
	index, seated := indexOf(seat)
	if !seated || !r.seated[index].taken || r.Finished() {
		return nil, nil
	}
	if r.draftOpen() {
		return r.abandon(seat), nil
	}
	if r.fight == nil && len(r.played) == 0 {
		r.seated[index] = peer{}
		return nil, nil
	}
	return r.abandon(seat), nil
}

// resolved carries the battle from a turn just spent to whatever comes next: the
// digest of everything that decision produced goes to both seats, and a battle
// that ended takes the series forward.
//
// ⚠️ The order here is what keeps the mirror's digest equal to the room's, and
// it is the one piece of this file that is easy to get wrong. The room advances
// to the next open turn **before** reading its cursor, because that is exactly
// what the mirror does: Replay with one decision and a nil fallback applies it,
// walks through whatever is forced after it, and stops on the prompt it cannot
// decide. So both event runs contain the decision, then every skipped turn, then
// the next turn's opening — and reading the cursor a step earlier would make
// every digest disagree while both peers were fighting the same battle
// perfectly.
func (r *Room) resolved(decision battle.Decision) ([]Outbound, error) {
	r.prompt, r.onTurn = nil, -1
	if err := r.settle(); err != nil {
		return nil, err
	}
	events, next := r.fight.Since(r.cursor)
	r.cursor = next
	// The log's own position, read here rather than off `events` above: the two
	// cursors stand in different places, so the slice one of them just took is
	// not the slice the other is owed. → the record field.
	r.record.Choices = append(r.record.Choices, decision)
	r.transcribe()
	digest, err := wire.DigestEvents(events)
	if err != nil {
		return nil, fmt.Errorf("digest the events of %q's turn: %w", decision.Unit, err)
	}
	turn := wire.Turn{Decision: decision, Events: digest}
	out := r.both(turn)
	// The watcher's record takes the same body the two players were just sent,
	// on the line the room sends it, so the two cannot come apart. ⚠️ It is
	// recorded rather than derived from the battle for the reason the wire
	// carries a decision and not events: reconstructing a battle.Decision out of
	// battle.Events would be a second derivation of a recorded string — a pass
	// reason is battle.NoActionReason and the room's TimeoutReason is the room's
	// own — and it is the digest a mirror checks against, which nothing outside
	// this call can produce. → watch.
	r.watch(turn)
	if !r.fight.Finished() && !r.capped {
		return out, nil
	}
	more, err := r.close()
	if err != nil {
		return out, err
	}
	return append(out, more...), nil
}

// settle carries the battle to the next turn that needs an answer, or to its
// end.
//
// A skipped prompt is walked past rather than reported: the unit has already
// lost its action, to control or to a timed effect, and nobody is being asked
// anything — which is what makes "a Skipped prompt starts no clock" a property
// of this loop.
//
// ⚠️ The turn cap is checked **after** the skipped test and not before it, and
// the reason is the mirror again. A mirror stops only at a turn it is asked to
// decide, so that is the only boundary the room may stop at either; capping in
// the middle of a run of skipped turns would leave the room's event run one
// short of the mirror's and report a divergence that was not one. Skipped turns
// still count towards the cap — a turn is a turn — they just cannot be the turn
// the cap bites on.
func (r *Room) settle() error {
	for !r.fight.Finished() {
		prompt, err := r.fight.Advance()
		if err != nil {
			return fmt.Errorf("advance battle %d of %d: %w", r.index, r.config.Battles, err)
		}
		r.turns++
		if prompt.Skipped {
			r.skipped++
			continue
		}
		if r.turns > r.config.TurnCap {
			r.capped = true
			return nil
		}
		r.prompt, r.onTurn = prompt, r.seatIndexOn(prompt)
		return nil
	}
	return nil
}

// seatIndexOn is which seat has to answer a prompt, read off the side of the
// unit it names.
func (r *Room) seatIndexOn(prompt *battle.Prompt) int {
	unit, known := r.fight.Unit(prompt.Unit)
	if !known {
		return -1
	}
	index, seated := indexOf(r.seatOnSide(unit.Side))
	if !seated {
		return -1
	}
	return index
}

// begin opens the next battle of the series: the seed derived from the room's
// one seed, the home seat enlisted first, and a wire.Start to each seat naming
// its own side.
func (r *Room) begin() ([]Outbound, error) {
	r.index++
	r.home = r.config.HomeFor(r.index)
	r.seed = r.config.SeedFor(r.index)
	r.turns, r.capped, r.cursor = 0, false, 0
	// A fresh Log rather than slices truncated to nought, so the battle just
	// finished keeps the arrays it was handed out on: BattleResult carries the
	// log by value and Reading copies the slice of results, and a room appending
	// into an array a caller is reading is the one thing that copy exists to
	// prevent.
	r.record, r.logCursor = battle.Log{Seed: r.seed}, 0

	// Home first, which is the sixty-point line: atb.Queue.Add assigns seq in
	// the order battle.New is handed its roster and seq is the last tie-break in
	// the turn order, so the slice order decides which side wins a speed tie.
	// This is the same append forge.FightSquads does and for the same reason.
	away := other(r.home)
	homeIndex, _ := indexOf(r.home)
	awayIndex, _ := indexOf(away)
	roster, err := r.seated[homeIndex].squad.Take(hex.SideAlly, r.deps.Characters)
	if err != nil {
		return nil, fmt.Errorf("field the %s squad: %w", r.home, err)
	}
	facing, err := r.seated[awayIndex].squad.Take(hex.SideEnemy, r.deps.Characters)
	if err != nil {
		return nil, fmt.Errorf("field the %s squad: %w", away, err)
	}
	roster = append(roster, facing...)
	// And then the lead of each contested speed group is alternated over that
	// slice, which is the refinement the append above is only half of: home
	// first decides *which* side a tie falls to, and this decides that they do
	// not all fall to the same one. → alternateContested.
	roster, err = alternateContested(r.deps.Books, r.seed, roster)
	if err != nil {
		return nil, fmt.Errorf("order battle %d of %d: %w", r.index, r.config.Battles, err)
	}

	fight, err := battle.New(r.deps.Books, r.seed, roster)
	if err != nil {
		return nil, fmt.Errorf("open battle %d of %d: %w", r.index, r.config.Battles, err)
	}
	r.fight = fight
	r.record.Roster = roster
	fight.Begin()

	out := make([]Outbound, 0, seatCount)
	for _, seat := range seats {
		out = append(out, Outbound{To: seat, Body: wire.Start{
			Seed:   r.seed,
			Roster: roster,
			Side:   r.sideOf(seat),
			Battle: r.index,
		}})
	}
	// A watcher gets the same wire.Start, from the **host's** chair: Side is the
	// half of the board a client plays and a watcher plays neither, so it is
	// given the seat a room hands out first rather than whatever fell out of the
	// loop above. Everything else is identical to the players' — the same seed,
	// the same roster in the same order, the same battle index — and recording
	// the whole body is what lets a watcher joining halfway build its own mirror
	// with no Battle.Roster() accessor and no second copy of that slice.
	// → Since, which carries the argument for the side.
	r.watch(wire.Start{
		Seed:   r.seed,
		Roster: roster,
		Side:   r.sideOf(wire.SeatHost),
		Battle: r.index,
	})
	// The opening board and the first turn's beginning are events, and no
	// wire.Turn carries them: a mirror produces them itself by calling Begin and
	// advancing to the same prompt. So the cursor starts *after* them, which is
	// what Recorded is for, and the first digest exchanged covers the first
	// decision rather than the first decision plus the opening.
	if err := r.settle(); err != nil {
		return out, err
	}
	r.cursor = r.fight.Recorded()
	r.transcribe()
	if r.capped {
		more, err := r.close()
		if err != nil {
			return out, err
		}
		out = append(out, more...)
	}
	return out, nil
}

// transcribe takes whatever the battle has produced since the log last read it.
//
// It is called wherever the room's own cursor moves and nowhere else, which is
// what keeps the two positions describing the same moment: the room reads after
// settle has carried the battle as far as it goes, so the log stops on the same
// event — including, for a capped battle, the capped turn's own turn_began.
//
// A room with no battle in front of it transcribes nothing rather than being a
// state the callers have to check for: close clears the fight before the series
// moves on, and abandon clears it without a result at all.
func (r *Room) transcribe() {
	if r.fight == nil {
		return
	}
	events, next := r.fight.Since(r.logCursor)
	r.record.Events = append(r.record.Events, events...)
	r.logCursor = next
}

// close records the battle that has just ended, moves the series on, and opens
// the next battle or finishes the match.
func (r *Room) close() ([]Outbound, error) {
	result := BattleResult{
		Battle: r.index, Home: r.home, Seed: r.seed,
		Outcome: r.fight.Outcome(), Turns: r.turns, Capped: r.capped,
		Log: r.record,
	}
	if side, decided := r.fight.Winner(); decided {
		result.Winner = r.seatOnSide(side)
	}
	r.played = append(r.played, result)
	if index, seated := indexOf(result.Winner); seated {
		r.standing[index]++
	}
	r.fight, r.prompt, r.onTurn = nil, nil, -1
	if !r.seriesOver() {
		// ⚠️ **A drafting series bans and picks again rather than beginning**,
		// and it sends nothing while it does — the same shape bothTaken has when
		// it opens the first one. → redraft, where the pool's reset is argued.
		if r.config.Drafts {
			return nil, r.redraft()
		}
		return r.begin()
	}
	leader := r.leader()
	r.result = Result{
		Winner: leader, Wins: r.standing, Battles: len(r.played),
		Verdict: VerdictDrawn,
	}
	if leader.Valid() {
		r.result.Verdict = VerdictWon
	}
	return nil, nil
}

// abandon ends the match without finishing the battle it interrupted, and tells
// the seat that is still there.
//
// ⚠️ **This is the one ending that needs a message**, and the reason is that it
// is the one a mirror cannot reach on its own. Every other ending a client
// computes: it learns each battle's outcome from its own Ended event and the
// series length from Welcome.Battles, and it stops at the turn cap by the same
// arithmetic the room does (→ wire.Welcome.TurnCap). A departure is different —
// there is no Ended for the battle in progress, because the engine concluded
// nothing about it, and no further Start — so a peer handed nothing would hang on
// its own open prompt waiting for a turn that is never coming.
//
// It is addressed to the **other** seat only. The transport has already decided
// there is nobody at the seat that left, so a message to it is a message to
// nobody; Outbound names a seat precisely so the two can be handed different
// things. That other seat is necessarily taken: a match only starts once both
// seats are, and nothing frees one mid-match — the pre-match departure is
// answered in Left, above, before this is reached.
func (r *Room) abandon(departed wire.Seat) []Outbound {
	r.result = Result{
		Verdict:  VerdictAbandoned,
		Departed: departed,
		Wins:     r.standing,
		Battles:  len(r.played),
	}
	r.fight, r.prompt, r.onTurn = nil, nil, -1
	closed := wire.Closed{Reason: wire.ClosureLeft}
	// ⚠️ **A watcher takes this off the record rather than out of an Outbound,
	// and that is what keeps other() honest.** The message below is addressed to
	// one seat because the transport has already decided nobody is at the other;
	// a watcher is not a seat at all, so widening the addressing to reach it
	// would be exactly the third-citizen change seatCount refuses. It needs the
	// body for the reason the remaining player does — there is no Ended for the
	// battle this interrupted and no further Start, so a mirror handed nothing
	// hangs on its own open prompt. → watch.go.
	r.watch(closed)
	return []Outbound{{To: other(departed), Body: closed}}
}

// refuse is one wire.Refused for one seat.
func (r *Room) refuse(seat wire.Seat, code wire.Code) []Outbound {
	return []Outbound{{To: seat, Body: wire.Refused{Code: code}}}
}

// refuseConnection is a refusal at the gate, where there is no seat to name.
func (r *Room) refuseConnection(code wire.Code) []Outbound {
	return []Outbound{{Body: wire.Refused{Code: code}}}
}

// both is one body addressed to each seat. Every turn goes to both clients,
// including the one the client itself asked for: a mirror applies its own
// decision from the wire rather than from its own input, so that the events it
// produces come out of the same call on both sides.
func (r *Room) both(body wire.Body) []Outbound {
	out := make([]Outbound, 0, seatCount)
	for _, seat := range seats {
		out = append(out, Outbound{To: seat, Body: body})
	}
	return out
}
