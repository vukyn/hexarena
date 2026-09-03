package socket

import (
	"fmt"
	"sync"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/wire"
)

// Mirror is a client's own battle, driven by the five messages a server sends.
//
// It is a **mirror** in the sense the design record means: it builds its own
// *battle.Battle from the seed and roster on wire.Start and steps it with the
// decisions on wire.Turn, so it computes the state by computing the battle
// rather than by being told it. Nothing about the board, the health, the
// statuses or the queue ever crosses the wire, because none of it has to.
//
// # ⚠️ Why this is production code and not a test fixture
//
// **Nothing on the wire says whose turn it is.** wire.Turn carries a decision
// and a digest; Asking is where "I am being asked" comes from, and it is derived
// from this battle. So a client that was not a mirror could not know when to
// enable input, and the thinnest possible client is therefore exactly this. An
// end-to-end test needs one too, so writing it as a fixture and promoting it
// later would be writing it twice. → the package comment.
//
// # The digest is what makes it a check rather than a hope
//
// Every wire.Turn carries the digest of the events the *server's* battle
// produced. This one digests the events its own battle produced for the same
// decision and compares, every turn, so a divergence is loud on the turn it
// happens with two digests to compare — rather than a board that quietly
// drifts. A mismatch is a *Divergence naming the turn.
//
// ⚠️ The order the events are read in is what keeps the two equal, and it is
// easy to get wrong: Replay applies the decision, walks through whatever is
// forced after it, and stops on the prompt it cannot decide, so the run holds
// the decision, then every skipped turn, then the next turn's opening. That is
// the same order room.resolved reads its cursor in, and reading a step earlier
// on either side makes every digest disagree while both peers fight the same
// battle perfectly.
//
// # It is safe to DRAW while Play is stepping it, and that is new
//
// ⚠️ **This comment used to end "it is not safe for concurrent use and needs not
// to be: one client, one connection, one goroutine reading it", and that stopped
// being true the moment a terminal drew one.** A full-screen client runs Play on
// its own goroutine and redraws on bubbletea's, so the battle Play is stepping
// is the battle the screen is reading — which is exactly the concurrent use that
// sentence said would never happen. The answer is a lock rather than a copy: a
// *battle.Battle is a pointer and copying one is not a thing this repository can
// do, and handing a renderer the events instead would be the thin client
// README.md § *The client runs the engine too* refuses.
//
// So: Receive takes the write lock and every reading accessor takes the read
// lock. Read is the only safe way for a renderer on another goroutine to look at
// more than one of those readings at once — a screen that asked for the battle,
// then the prompt, then the side would get three readings of three different
// moments.
//
// ⚠️ **Decide releases the lock before it calls the chooser**, and that is the
// one ordering here that must not be got wrong. → Decide, which carries the
// measurement of *why* — the reason is a writer queuing, not the two readers,
// and getting that wrong once made a test that could not see the mistake.
type Mirror struct {
	// mu orders a Play goroutine stepping this mirror against a renderer drawing
	// it. → the paragraph above for why it exists at all.
	//
	// It is an RWMutex rather than a Mutex because drawing is a read and there
	// may be more than one reader — View and Update both take it in the game
	// client, and neither excludes the other.
	mu sync.RWMutex

	seat  wire.Seat
	books battle.Books

	welcome wire.Welcome
	seated  bool

	// The battle in progress, and this client's own cursor into its record.
	fight  *battle.Battle
	prompt *battle.Prompt
	cursor int
	index  int
	side   hex.Side
	seed   uint64

	// turns is how many turns this client's own battle has opened, skipped ones
	// included, and capped is that count having passed the cap wire.Welcome
	// named.
	//
	// ⚠️ This is the client doing the arithmetic the cap exists for. A capped
	// battle emits no Ended — the engine concluded nothing about it — and no
	// further wire.Start arrives, so without the cap a client would sit holding
	// the prompt it stopped on for ever. Given it, the client stops on the same
	// turn the room did, because it counts the same thing: every Advance emits
	// exactly one battle.TurnBegan.
	turns  int
	capped bool

	// events is everything this client's battle has produced in the battle in
	// progress, which is what a renderer would be drawing.
	events []battle.Event
	// fought is the series as this client's own engine settled it.
	fought []Fought
	// compared is how many digests this client has checked, so a caller can say
	// the check happened rather than hoping it did — the same argument
	// Room.Skipped is exposed under.
	compared int

	refusals []wire.Code
	closure  wire.Closure
}

// Fought is one battle of the series as a client's own engine settled it, which
// is the only place a client learns an outcome from: there is deliberately no
// series-standing message, because a standing would be a second declaration of
// something the mirror computes.
type Fought struct {
	// Battle is which battle of the series this was, counting from one.
	Battle int
	// Side is the half of the board this client played, which changes between
	// battles because a match is fought both ways round.
	Side hex.Side
	// Seed is the seed this battle was fought from.
	Seed uint64
	// Outcome is the engine's own, and is battle.Undecided for a battle this
	// client stopped at the turn cap — the engine concluded nothing about that
	// one, exactly as the room records it.
	Outcome battle.Outcome
	// Winner is the side that won and Decided whether anybody did.
	Winner  hex.Side
	Decided bool
	// Capped is the turn cap having stopped the battle.
	Capped bool
	// Turns is how many turns this client's battle opened, skipped included.
	Turns int
}

// Mine reports whether this client won the battle.
func (f Fought) Mine() bool { return f.Decided && f.Winner == f.Side }

// NewMirror is a client's battle-to-be: it holds the books and knows which seat
// it is, and everything else arrives on a message.
func NewMirror(seat wire.Seat, books battle.Books) *Mirror {
	return &Mirror{seat: seat, books: books}
}

// Seat is which of the room's two places this client took.
//
// It takes the read lock like every other accessor even though the seat is
// written once, at the handshake, before Play exists: a rule that holds for
// "the fields that change" is a rule somebody has to classify a field under
// every time one is added, and the classification is what goes wrong.
func (m *Mirror) Seat() wire.Seat {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.seat
}

// Welcome is the room's configuration as this client was told it, and Seated
// whether it has been told.
func (m *Mirror) Welcome() (wire.Welcome, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.welcome, m.seated
}

// Side is the half of the board this client plays in the battle in progress.
func (m *Mirror) Side() hex.Side {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.side
}

// Battle is this client's own battle, for a renderer or a rating. Nil between
// battles and before the first one.
//
// ⚠️ **The lock is around the pointer and not around the battle.** A caller that
// keeps what this returns and reads it later is reading a battle Play may be
// stepping, which is precisely what Read exists to do instead. This accessor is
// for a caller on the Play goroutine itself — the chooser is the one that
// matters, since Play calls it and nothing else is running.
func (m *Mirror) Battle() *battle.Battle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fight
}

// Events is everything this client's battle has produced in the battle in
// progress, which is what a renderer draws.
func (m *Mirror) Events() []battle.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.events[:len(m.events):len(m.events)]
}

// Fought is every battle of the series this client has settled, in order.
func (m *Mirror) Fought() []Fought {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fought[:len(m.fought):len(m.fought)]
}

// Compared is how many per-turn digests this client has checked. → the field.
func (m *Mirror) Compared() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.compared
}

// Refusals is every wire.Code this client has been sent, in order, for a screen
// to word. Nothing here words one: the whole point of a code is that the
// sentence lives at this end, in the player's own language.
func (m *Mirror) Refusals() []wire.Code {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.refusals[:len(m.refusals):len(m.refusals)]
}

// Closure is the room having closed the match for a reason the board cannot
// show, and whether it has.
//
// ⚠️ This is the one ending a mirror cannot reach on its own: a departure leaves
// no Ended for the battle in progress and no further Start, so a client handed
// nothing would hang on its own open prompt. Every other ending is computed
// here. → wire.Closed.
func (m *Mirror) Closure() (wire.Closure, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closure, m.closure.Closes()
}

// Capped reports whether this client's battle in progress stopped at the turn
// cap the room named on wire.Welcome.
func (m *Mirror) Capped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capped
}

// Over reports whether the match is finished, by this client's own arithmetic.
//
// ⚠️ **It re-derives a rule the room also has, and that is the design rather
// than a duplication to be tidied away.** There is no series-standing message —
// a standing would be a second declaration of something the client computes, and
// the one place two peers could disagree about who is winning while both of
// their battles agreed. So the client learns each battle's outcome from its own
// Ended event, knows the series length from wire.Welcome.Battles, and stops when
// a side is past taking back or every battle has been fought. Two peers agree
// because they compute the same thing from the same configuration, which is the
// mirror contract itself — the same shape wire.Welcome.TurnCap takes.
//
// A closure ends it too, because that is the ending this arithmetic cannot see.
func (m *Mirror) Over() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.over()
}

// over is Over with the lock already held, so that Read can answer it without
// taking the read lock a second time.
//
// ⚠️ **A second RLock inside one is not free in Go**, and that is why every
// reading here comes in a locked and an unlocked spelling: sync.RWMutex blocks a
// read that arrives while a writer is waiting, so a reader that re-entered could
// sit behind a writer that is sitting behind the reader.
func (m *Mirror) over() bool {
	if m.closure.Closes() {
		return true
	}
	if !m.seated || m.welcome.Battles <= 0 {
		return false
	}
	if len(m.fought) >= m.welcome.Battles {
		return true
	}
	mine, theirs := 0, 0
	for _, one := range m.fought {
		switch {
		case !one.Decided:
		case one.Mine():
			mine++
		default:
			theirs++
		}
	}
	decisive := m.welcome.Battles / 2
	return mine > decisive || theirs > decisive
}

// Asking is the open prompt when it is **this** client's to answer, and it is
// the whole of how a client knows its turn has come.
//
// ⚠️ Nothing on the wire says so. The room knows whose turn it is — which is
// why wire.Act carries no unit — and it says nothing about it, because a "your
// turn" message would be a second declaration of a fact this battle already
// holds. So the derivation is: the battle stopped on a prompt, and the unit it
// names is on the side this client plays.
//
// It is false past the turn cap, because the room stops asking on the turn this
// client stops at.
func (m *Mirror) Asking() (*battle.Prompt, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.asking()
}

// asking is Asking with the lock already held. → over, for why the pair exists.
func (m *Mirror) asking() (*battle.Prompt, bool) {
	if m.fight == nil || m.prompt == nil || m.capped {
		return nil, false
	}
	unit, known := m.fight.Unit(m.prompt.Unit)
	if !known || unit.Side != m.side {
		return nil, false
	}
	return m.prompt, true
}

// Sight is a consistent reading of this mirror, valid only for the duration of
// the call it is handed to.
//
// It is one struct rather than a dozen accessors because a screen needs several
// of them to agree: a renderer that asked for the battle, then the prompt, then
// the side would get three readings taken at three different moments, and the
// prompt would name a unit on a board that had moved under it.
type Sight struct {
	// Fight is the battle in progress, nil between battles and before the first.
	Fight *battle.Battle
	// Asking is the open prompt when it is this client's to answer, and nil
	// otherwise — which is the whole of how a client knows its turn has come.
	Asking *battle.Prompt
	// Side is the half of the board this client plays in the battle in progress,
	// and Seed and Index which battle of the series it is.
	Side  hex.Side
	Seed  uint64
	Index int
	// Seated is the room's configuration having arrived, Over the match being
	// finished by this client's own arithmetic.
	Seated bool
	Over   bool
	// Welcome is the room's configuration, meaningful only when Seated.
	Welcome wire.Welcome
	// Closure is why the match stopped for a reason the board cannot show, and
	// is ClosureNone when nothing did.
	Closure wire.Closure
	// Fought is the series as this client's own engine settled it, and Refusals
	// every code the room has sent, in order.
	Fought   []Fought
	Refusals []wire.Code
}

// Read runs fn under this mirror's read lock, which is the only safe way for a
// renderer on another goroutine to look at a battle Play is stepping.
//
// ⚠️ **Nothing fn is handed may outlive fn.** That is the rule Registry.Read is
// written under, and here it is held by this comment and by nothing else: a
// *battle.Battle is handed over on purpose — a renderer computes the board by
// computing the battle, which is the whole design — so no type walk and no race
// detector can refuse one. A caller that keeps the pointer and reads it after fn
// returns is reading a battle the Play goroutine may be stepping, and the
// detector only fires if it actually does.
//
// Several readers may be inside this at once, which is what makes an Update and
// a View both able to take it in a bubbletea client.
func (m *Mirror) Read(fn func(Sight)) {
	if fn == nil {
		return
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	prompt, _ := m.asking()
	fn(Sight{
		Fight:    m.fight,
		Asking:   prompt,
		Side:     m.side,
		Seed:     m.seed,
		Index:    m.index,
		Seated:   m.seated,
		Over:     m.over(),
		Welcome:  m.welcome,
		Closure:  m.closure,
		Fought:   m.fought[:len(m.fought):len(m.fought)],
		Refusals: m.refusals[:len(m.refusals):len(m.refusals)],
	})
}

// takeSeat records the seat the room handed out, which the handshake reads off
// the welcome before the mirror has been told anything else.
//
// It is a method with the lock rather than a field write for the reason every
// accessor above takes one: a rule that holds for "the fields that matter" needs
// somebody to classify each new field, and the classification is what goes
// wrong.
func (m *Mirror) takeSeat(seat wire.Seat) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seat = seat
}

// Decide is the message this client sends for the open prompt, read off its own
// battle by the given chooser.
//
// ⚠️ It **applies nothing**. A mirror steps its battle from the wire.Turn that
// comes back rather than from its own input, so that the events it produces come
// out of the same call on both sides — which is why the room sends every turn to
// both clients including the one that asked for it. Deciding locally and
// applying locally would be two paths into one battle.
//
// It reports false when there is no prompt to answer.
//
// ⚠️ **The read lock is taken to find the prompt and RELEASED before the chooser
// is called, and that ordering is load-bearing rather than tidy.** In a
// full-screen client the chooser *is the player*: it blocks until a keystroke
// arrives, and the keystroke is handled by a goroutine that needs the read lock
// in order to draw.
//
// ⚠️ **Why a held lock is fatal is NOT "two readers cannot both be in", and that
// was measured after being got wrong.** sync.RWMutex admits several readers, so
// a chooser that only ever reads would sit happily beside a renderer that only
// ever reads, and holding the lock across choose passes a test built on that
// prediction. What actually breaks is a **writer arriving while the chooser
// waits**: Go queues a waiting writer ahead of new readers, so the next Receive
// blocks behind the held read lock and the renderer then blocks behind the
// writer — and the renderer is what the player is waiting to see before they can
// answer. TestDecideDoesNotHoldTheLockAcrossTheChooser builds exactly that three
// way and is what says so; the failure it catches is a hang, bounded so it
// reports rather than hangs the suite.
//
// The prompt handed to the chooser outlives the lock, which is safe for the one
// reason Play is what calls this: nothing steps the battle between the read and
// the answer, because the goroutine that would is the one waiting here.
func (m *Mirror) Decide(choose battle.Chooser) (wire.Body, bool) {
	m.mu.RLock()
	prompt, asking := m.asking()
	m.mu.RUnlock()
	if !asking || choose == nil {
		return nil, false
	}
	choice, ok := choose(prompt)
	if !ok {
		// ⚠️ No reason travels with it. A passed turn's wording lives on
		// battle.Decision and battle.NoActionReason is the single declaration of
		// it, so wire.Pass carries nothing at all and the server records the
		// reason because the server writes the log.
		return wire.Pass{}, true
	}
	return wire.Act{Skill: choice.Skill, Aim: hex.At(choice.Aim)}, true
}

// Receive is the client's whole message loop: one of the five server-bound
// messages in, and an error out when this client can no longer claim to be
// fighting the same battle.
//
// ⚠️ It takes both a body and a pointer to one, because the two producers in
// this repository hand over different things: room.Outbound carries values, and
// wire.Decode hands back pointers so that an unknown kind never reaches a
// struct. Room.Deliver takes both for the same reason.
//
// ⚠️ **It takes the write lock for the whole of one message**, so a renderer on
// another goroutine never sees a battle half stepped: the events, the prompt,
// the cursor and the fought list all move together or not at all. Every private
// step below therefore runs with the lock already held and must not take it
// again.
func (m *Mirror) Receive(body wire.Body) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch message := body.(type) {
	case *wire.Welcome:
		return m.welcomed(*message)
	case wire.Welcome:
		return m.welcomed(message)
	case *wire.Refused:
		m.refused(*message)
		return nil
	case wire.Refused:
		m.refused(message)
		return nil
	case *wire.Start:
		return m.open(*message)
	case wire.Start:
		return m.open(message)
	case *wire.Turn:
		return m.apply(*message)
	case wire.Turn:
		return m.apply(message)
	case *wire.Closed:
		return m.closed(*message)
	case wire.Closed:
		return m.closed(message)
	}
	if body == nil {
		return fmt.Errorf("%s was sent nothing at all", m.seat)
	}
	return fmt.Errorf("%s was sent a %s, which no server sends", m.seat, body.Kind())
}

// welcomed takes the room's configuration, and refuses a welcome addressed to
// another seat: a client that read one for the other would count somebody
// else's allowance down.
func (m *Mirror) welcomed(welcome wire.Welcome) error {
	if welcome.Seat != m.seat {
		return fmt.Errorf("%s was welcomed into the %q seat", m.seat, welcome.Seat)
	}
	if welcome.TurnCap <= 0 {
		// Without it a capped battle is invisible here: no Ended, no further
		// Start, and an open prompt nobody will ever answer.
		return fmt.Errorf("%s was welcomed with a turn cap of %d, so it could not stop where the room does",
			m.seat, welcome.TurnCap)
	}
	m.welcome, m.seated = welcome, true
	return nil
}

// refused records a code for a screen to word. It is not an error: a refusal is
// the room answering a message, and the connection carries on — a peer one
// version ahead meeting wire.CodeUnknownMessage is the case that shape exists
// for.
func (m *Mirror) refused(refusal wire.Refused) {
	m.refusals = append(m.refusals, refusal.Code)
}

// closed records the one ending this client cannot compute.
func (m *Mirror) closed(closed wire.Closed) error {
	if !closed.Reason.Closes() {
		return fmt.Errorf("%s was told the match closed for no reason", m.seat)
	}
	m.closure = closed.Reason
	m.fight, m.prompt = nil, nil
	return nil
}

// open builds this client's own battle from the seed and the roster, and walks
// it to the first turn that needs a decision — exactly what the room does, which
// is why the cursor starts *after* the opening board rather than at nought.
func (m *Mirror) open(start wire.Start) error {
	if !m.seated {
		return fmt.Errorf("%s was started before it was welcomed", m.seat)
	}
	if !start.Side.Fights() {
		return fmt.Errorf("%s was started on the %s side of battle %d", m.seat, start.Side, start.Battle)
	}
	fight, err := battle.New(m.books, start.Seed, start.Roster)
	if err != nil {
		return fmt.Errorf("%s builds its mirror of battle %d: %w", m.seat, start.Battle, err)
	}
	fight.Begin()
	// A nil script with a nil fallback walks to the first turn this client is
	// not allowed to decide, which is every turn: the room decides nothing for
	// it either, so both stop in the same place.
	_, prompt, err := fight.Replay(nil, m.limit(), nil)
	if err != nil {
		return fmt.Errorf("%s opens battle %d: %w", m.seat, start.Battle, err)
	}
	m.fight, m.prompt = fight, prompt
	m.side, m.index, m.seed = start.Side, start.Battle, start.Seed
	m.cursor, m.events = fight.Recorded(), nil
	m.turns, m.capped = 0, false
	// ⚠️ The opening's own turns are counted off the **whole record** and not
	// off the cursor. The cursor starts after the opening board, because the
	// first digest exchanged covers the first decision rather than the first
	// decision plus the board — so a client counting only what arrives on a
	// wire.Turn would sit one turn behind the cap for the whole battle.
	opening, _ := fight.Since(0)
	m.count(opening)
	return nil
}

// apply takes one decision and checks the digest.
func (m *Mirror) apply(turn wire.Turn) error {
	if m.fight == nil {
		return fmt.Errorf("%s was sent %q's turn with no battle open", m.seat, turn.Decision.Unit)
	}
	_, prompt, err := m.fight.Replay(battle.Script{turn.Decision}, m.limit(), nil)
	if err != nil {
		return fmt.Errorf("%s applies %q's turn %d: %w", m.seat, turn.Decision.Unit, turn.Decision.Turn, err)
	}
	m.prompt = prompt
	events, next := m.fight.Since(m.cursor)
	m.cursor = next
	m.events = append(m.events, events...)
	m.count(events)
	digest, err := wire.DigestEvents(events)
	if err != nil {
		return fmt.Errorf("%s digests %q's turn %d: %w", m.seat, turn.Decision.Unit, turn.Decision.Turn, err)
	}
	m.compared++
	if digest != turn.Events {
		return &Divergence{
			Seat: m.seat, Battle: m.index,
			Unit: turn.Decision.Unit, Turn: turn.Decision.Turn,
			Room: turn.Events, Client: digest,
		}
	}
	if m.fight.Finished() || m.capped {
		m.settled()
	}
	return nil
}

// settled records the battle this client's own engine has just finished, which
// is where a client learns an outcome.
func (m *Mirror) settled() {
	one := Fought{
		Battle: m.index, Side: m.side, Seed: m.seed,
		Outcome: m.fight.Outcome(), Turns: m.turns, Capped: m.capped,
	}
	one.Winner, one.Decided = m.fight.Winner()
	m.fought = append(m.fought, one)
	m.prompt = nil
}

// count adds a run of events' turns to this client's own tally and stops it at
// the cap the room named.
func (m *Mirror) count(events []battle.Event) {
	for _, event := range events {
		if event.Kind == battle.TurnBegan {
			m.turns++
		}
	}
	if m.welcome.TurnCap > 0 && m.turns > m.welcome.TurnCap {
		m.capped = true
	}
}

// limit is how many turns one Replay may walk through: a decision plus however
// many skipped turns follow it.
//
// It comes off wire.Welcome.TurnCap rather than being a number of this package's
// own, which is the cheapest possible reading of "the client stops where the
// room stops": no battle can open more turns than the cap, so a limit of the cap
// cannot cut a run of skipped turns short, and there is nothing here for a caller
// to get wrong.
func (m *Mirror) limit() int {
	if m.welcome.TurnCap > 0 {
		return m.welcome.TurnCap
	}
	return 1
}

// Divergence is two peers no longer fighting the same battle, reported on the
// turn it happens with both digests to compare.
//
// It is a type rather than a sentence because the turn is the whole point: a
// caller that can name the turn can name the decision, and the two digests are
// what a bug report needs. → the note on Mirror.
type Divergence struct {
	// Seat is the client that noticed, Battle which battle of the series it was
	// fighting, and Unit and Turn the decision the two disagreed about.
	//
	// ⚠️ Turn is battle.Decision.Turn, which is **the unit's own count of its
	// turns** and not a position in the battle. That is the number the log
	// records and the number a decision is identified by, so it is the right one
	// to carry — but a reader who takes it for "the nth turn of the battle" will
	// read `A1 turn 5` before `E1 turn 4` and think the report is wrong.
	Seat   wire.Seat
	Battle int
	Unit   string
	Turn   int
	// Room and Client are the two digests, the server's first.
	Room   wire.EventDigest
	Client wire.EventDigest
}

func (d *Divergence) Error() string {
	return fmt.Sprintf("%s diverged on %q's turn %d of battle %d: the room's events digest %s, its own %s",
		d.Seat, d.Unit, d.Turn, d.Battle, d.Room.Short(), d.Client.Short())
}
