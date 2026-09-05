package room

import (
	"slices"

	"github.com/vukyn/hexarena/internal/wire"
)

// The room's half of the ban and pick: routing a decision into the draft,
// answering the transport's clock while one runs, and handing the two squads it
// produces to begin().
//
// ⚠️ **Nothing downstream of the draft knows a draft happened, and that is the
// promise the whole design was built on.** What a finished draft produces is two
// squads; wire.Start already carries a roster; so begin() is called **unchanged**
// and a wire.Start is a wire.Start. The room's draft state is confined to this
// file and to the two fields on Room.
//
// ⚠️ **The state machine itself is not here and must not be restated here.**
// internal/draft owns whose decision is due, what the pool has left and every
// refusal about it; this file decides which wire.Code a refusal travels as, which
// is the one thing that package cannot know. draft.Draft.Apply is the single
// declaration of the draft's own sequence and this file is one of its two
// callers — a mirroring client is the other — which is what makes the mirror a
// mirror rather than two switches that agree today.

// draftOpen reports whether this room's ban and pick is the thing it is waiting
// on: a room that drafts, with both seats taken, whose draft is neither finished
// nor cancelled.
//
// ⚠️ **Both seats taken is part of the question and not a redundancy.** The draft
// is built in New — so that one this room could never finish is refused before a
// code is handed out — which means it exists, and is already due its first ban,
// while the room still holds one player. A room with one player is waiting on
// nobody (→ Awaiting), so what opens the draft is the second seat rather than the
// draft's own construction.
//
// A draft that is **Done** is closed here: the battle has begun, so the room is
// waiting on a prompt and not on a decision. A **cancelled** one is closed
// because the room is over.
//
// ⚠️ **Finished is asked as well as Cancelled, and the two are not the same
// question.** A draft cancels itself only on its own timeout; a peer *leaving*
// mid-draft ends the match through abandon and leaves the draft neither done nor
// cancelled — so without this the room would go on reporting an open decision
// after its match was over, and Awaiting promises to be false once it is. What
// that cost is not hypothetical: a transport arms its allowance off Awaiting, so
// a finished room would hand it a seat to start a countdown on.
func (r *Room) draftOpen() bool {
	if r.drafting == nil || !r.bothSeatsTaken() || r.Finished() {
		return false
	}
	return !r.drafting.Done() && !r.drafting.Cancelled()
}

// bothSeatsTaken reports whether the room is full, which is the state a match —
// and a draft — runs in.
func (r *Room) bothSeatsTaken() bool {
	for index := range r.seated {
		if !r.seated[index].taken {
			return false
		}
	}
	return true
}

// draftAwaiting is the room's side of the clock while the ban and pick runs: the
// seat whose decision is due, and false when the draft is not what the room is
// waiting on.
//
// # ⚠️ THE ARRANGE PHASE HAS TWO DECISIONS PENDING AND A READING HOLDS ONE SEAT
//
// Reading carries one Awaiting, internal/socket arms exactly one timer off it,
// and the arrange phase has **both** seats pending at once by design
// (→ draft.Draft.Arrange, and draft.Draft.Turn on why Turn never widened for it).
// Neither Reading nor the transport is widened here: a two-timer transport is a
// far larger change than one phase is worth.
//
// So the phase's clock is **serialised** — the seat answered is the *first in
// draft.AwaitingArrangement()*, a slice derived from the seats array precisely so
// that it reaches an output deterministically, and Waiting is true. One allowance
// is armed on it; when it fires, TimedOut on that seat cancels the draft, which
// is what a draft timeout does for any seat.
//
// ⚠️ **Two things that costs, written down rather than discovered, because a
// reader will expect otherwise** — both measured in
// TestTheArrangePhaseSerialisesItsAllowance:
//
//  1. **The countdown RESTARTS when either side arranges**, so the phase's worst
//     case is about twice an allowance rather than one. That is a fact about the
//     transport rather than about this function: Server.settled re-arms off the
//     reading after **every** batch, so whichever arrangement arrives first is an
//     exchange, and the seat still owed is then given a fresh full allowance from
//     that moment. A side that arranged promptly therefore hands its opponent
//     more time and not less — which is not what "one allowance per decision"
//     would give, and is not what "one allowance covers the phase" would give
//     either.
//  2. **While NEITHER side has arranged the seat named is the host**, because
//     that is the order the seats array is in and neither seat is more owed than
//     the other. Once one side *has* arranged the name is exact —
//     AwaitingArrangement holds only seats that have not — so this is a wording
//     matter confined to the both-silent case. It is harmless there for a stated
//     reason: a draft timeout cancels the whole room and the outcome blames
//     nobody, so nothing is decided by which seat the closure was reported
//     against. → draftTimedOut.
func (r *Room) draftAwaiting() (wire.Seat, bool) {
	if onTurn, _, due := r.drafting.Turn(); due {
		return onTurn, true
	}
	// Turn answers nothing during the arrange phase — it answers one seat and the
	// phase has two — so reaching here with anybody still owed *is* the phase.
	// → the note above for what taking the first of them costs.
	if pending := r.drafting.AwaitingArrangement(); len(pending) > 0 {
		return pending[0], true
	}
	return "", false
}

// decideFrom is a seat taking its draft decision.
//
// ⚠️ **THE SEAT COMES FROM THE CONNECTION AND NEVER FROM THE MESSAGE, and that is
// structural rather than remembered.** A reader looking for where the seat is
// read will look here, and the answer is `index` — which Deliver took off its
// `from` parameter — assembled with the decision the peer sent in the one
// wire.DraftEntry literal below. There is nothing in the message to be tempted
// by: wire.Decide embeds a wire.DraftDecision and a DraftDecision has **no seat
// field at all**, which is why that shape exists rather than a DraftEntry with
// the seat left blank — a field a sender is asked not to fill in is a field
// somebody fills in. → wire.Decide.
//
// ⚠️ **It routes through draft.Draft.Apply rather than switching on the step.**
// Apply is this repository's one declaration of the draft's sequence and a
// mirroring client is its other caller, so the room and the client take the same
// decision through the same call — which is what makes a client's computed draft
// comparable to the room's at all. A switch here would be the second copy Apply
// was moved out of internal/draft's record_test.go to prevent.
func (r *Room) decideFrom(index int, decision wire.DraftDecision) ([]Outbound, error) {
	seat := seats[index]
	if !r.draftOpen() {
		// The closest true thing the ten codes can say, and it is the same answer
		// answerFrom gives a wire.Act that arrives while a draft is still running:
		// the room is not asking this peer for this. A room that never drafts at
		// all lands here too, which is right — nobody in it is ever asked to
		// decide.
		return r.refuse(seat, wire.CodeNotYourTurn), nil
	}
	// ⚠️ **A timeout is the TRANSPORT's input and never a peer's message**, and
	// this is the one step Apply routes that a client must not be able to reach:
	// draft.TimedOut cancels the whole room, so a peer allowed to send a
	// StepTimeout could close a draft it did not like by claiming its own clock
	// had run out. It is wire.CodeIllegalAction rather than an unknown-message
	// refusal because the *kind* is one this room takes — what the open decision
	// did not offer is this step. → TimedOut, which is where a real one arrives,
	// and wire.StepTimeout, which says it is not a decision anybody made.
	if decision.Step == wire.StepTimeout {
		return r.refuse(seat, wire.CodeIllegalAction), nil
	}
	if !r.draftAsking(seat) {
		return r.refuse(seat, wire.CodeNotYourTurn), nil
	}
	if err := r.drafting.Apply(wire.DraftEntry{Seat: seat, DraftDecision: decision}); err != nil {
		// ⚠️ **Every refusal left here is a legality one, and the draft's own
		// sentence is deliberately dropped.** The four are a decision of the wrong
		// step, one naming a character out of the pool or already taken, an
		// illegal loadout and a refused arrangement — plus a step this protocol
		// does not have — and they all travel as wire.CodeIllegalAction, which is
		// the decision wire.CodeSquadRefused already takes for the four ways a
		// squad is turned away. The client holds the pool, cast.ChooseLoadout and
		// placement.Squad.Validate itself, so it can say precisely which, in the
		// player's own language, where a code could only ever say "one of five" —
		// and a server that spelled it would be a server deciding what language
		// its clients read in. → wire.CodeIllegalAction.
		return r.refuse(seat, wire.CodeIllegalAction), nil
	}
	return r.draftAdvanced()
}

// draftAsking is whether the draft is waiting on this seat, asked **before** the
// decision is applied so that "nobody is asking you" and "what you asked for is
// illegal" come back as two different codes. internal/draft answers both in one
// vocabulary — sentences — and this package has two codes.
//
// ⚠️ **The arrange phase is asked about separately because Turn answers one seat
// and the phase has two**, which is draft.Turn's own note: Arranging is the
// phase's accessor and AwaitingArrangement is who it is still waiting on. A seat
// that has already arranged is not being asked anything, so it is refused here
// rather than by Arrange — an arrangement is made once.
func (r *Room) draftAsking(seat wire.Seat) bool {
	if r.drafting.Arranging() {
		return slices.Contains(r.drafting.AwaitingArrangement(), seat)
	}
	onTurn, _, due := r.drafting.Turn()
	return due && onTurn == seat
}

// draftAdvanced is what the room says once a decision has gone in: the entries
// the draft recorded since this room last read its record, and — only once the
// draft is Done — the two drafted squads seated and the match begun.
//
// ⚠️ **A wire.Drafted is never sent empty, and the arrange phase is why that
// guard is here rather than obviously unreachable.** A ban, a skip, a pick and a
// loadout each record exactly one entry, so the read is never empty for them; the
// **first** of the two arrangements records **nothing**, because nothing reaches
// the record until both are in — an entry is public the moment it is sent, so
// appending the first arrangement when it arrives *is* showing it to the other
// player (→ draft.Draft.Arrange). That decision is therefore answered with no
// message at all, which is the honest thing: a room sending `{"decisions":[]}`
// would be a room saying nothing happened. → wire.Drafted.Decisions.
//
// ⚠️ **begin() is called UNCHANGED and only on Done().** Picked() is the wrong
// question and would be a live bug rather than a slow one: draft.Squads answers
// two squads with **nobody in them** until both sides have arranged, deliberately
// — hex.Offset's zero value is a real cell, so an honestly empty squad beats a
// plausible one — so a room that began on Picked would hand begin() an empty
// roster and fail at the moment it fielded it.
//
// The squads are indexed by seat in the order this package hands seats out, which
// is the same array r.seated is indexed by, and draft.Squads says so at its own
// end. So the two are copied across positionally rather than by name.
func (r *Room) draftAdvanced() ([]Outbound, error) {
	entries, next := r.drafting.Since(r.draftCursor)
	r.draftCursor = next
	var out []Outbound
	if len(entries) > 0 {
		out = r.both(wire.Drafted{Decisions: entries})
	}
	if !r.drafting.Done() {
		return out, nil
	}
	drafted := r.drafting.Squads()
	for index := range r.seated {
		r.seated[index].squad = drafted[index]
	}
	opening, err := r.begin()
	if err != nil {
		return out, err
	}
	return append(out, opening...), nil
}

// draftTimedOut is an allowance running out during the ban and pick, and it
// **closes the room** rather than passing anything.
//
// ⚠️ **This is the one place the design does not follow "a timeout announces and
// passes"**, and the reason is that there is nothing honest to pass with: a side
// that never picked has no squad to fight with, and a defaulted pick or a
// defaulted formation would hand somebody a side they did not choose and call it
// theirs — where placement alone is worth nineteen points of win rate.
// → wire.ClosureDraftExpired, which carries that argument, and draft.TimedOut,
// which is where the draft itself refuses to invent one.
//
// ⚠️ **Both seats are told, unlike a departure.** abandon writes to the other
// seat only, because the transport has already decided nobody is at the one that
// left; here both peers are still connected and both are holding an open draft
// decision — in the arrange phase, literally both at once — so a message to one
// of them would leave the other waiting for a decision nobody is coming to take.
//
// ⚠️ **The verdict is VerdictAbandoned with no Departed**, which is the same
// verdict a departure and a stopped host produce, for the reason that closure
// gives: nobody drafted a squad, so there is no board to have a verdict about and
// nobody is charged with anything. Departed is left the zero Seat because nobody
// went away — so "abandoned, naming nobody" is what a transport reads a cancelled
// draft as, and that absence is the only thing telling the two endings apart in
// the room's own record today.
func (r *Room) draftTimedOut(seat wire.Seat) ([]Outbound, error) {
	if err := r.drafting.TimedOut(seat); err != nil {
		// A timeout for a seat the draft is not asking, which is an ordinary race
		// rather than a fault: a timer that has already fired cannot be stopped,
		// so one may arrive just after its seat answered — and in the arrange
		// phase, just after that seat arranged. It is wire.CodeNotYourTurn for the
		// reason the battle's is, and the choice is load-bearing rather than
		// tasteful: internal/socket reads **exactly that code, alone** as "this
		// report was late", counts it and re-arms, instead of dropping anybody.
		return r.refuse(seat, wire.CodeNotYourTurn), nil
	}
	r.result = Result{Verdict: VerdictAbandoned}
	return r.both(wire.Closed{Reason: wire.ClosureDraftExpired}), nil
}
