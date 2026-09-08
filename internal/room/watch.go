package room

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/wire"
)

// The room's half of a **watcher**: an append-only record of the bodies somebody
// who is not playing has to be handed, and a read a consumer paces itself.
//
// ⚠️ **A watcher is not a third seat, and the reason is that a third seat would
// change who wins.** seatCount's own comment says why the seats are an array and
// not a map: the order the two seats are visited in reaches the roster, and the
// roster's order decides which side wins a speed tie. So a watcher threaded
// through seats, through the roster or through other() shifts that order, and the
// battle it came to watch is not the battle that would have been fought — while
// nothing looks wrong, because the roster is still legal and two peers who both
// have the watcher still agree on every digest. Only a comparison against a match
// played *without* one can see it, which is
// TestAMatchPlayedWithAWatcherReadingIsTheSameMatch.
//
// **So the payoff of this shape is what did NOT change**, and it is worth stating
// rather than leaving to be noticed: other() is still "the other one", seats is
// still the two seats a room hands out, the roster is composed exactly as it was
// and seatCount stays 2. The watcher takes its wire.Closed off this record rather
// than out of an Outbound, which is what keeps other() honest — TODO.md predicted
// that other() "stops being 'the other one' the day a room holds spectators", and
// it does not, because a watcher is a different kind of citizen rather than
// another seat.
//
// ⚠️ **The room writes this record and never reads it.** No behaviour of the room
// branches on it — not on its length, not on its contents — which is what makes
// "a watcher cannot change the match" a property of the code rather than a claim
// about it. TestTheWatcherRecordIsWriteOnlyFromTheRoom is an AST walk holding
// exactly that, because the way a watcher would *start* changing a battle is an
// innocent `if len(r.watched) > 0` appearing in one of the paths below.
//
// ⚠️ **The one thing the room does know about watching is whether it takes any,
// and that is configuration rather than state.** Config.Watchable is fixed
// before anybody joins, read in exactly one place — the gate, to answer one
// hello — and never written; it is not a count, a list or a cap, so everything
// below about the room keeping nothing is unchanged by it. → gate.go, and
// cmd/hexarena-host's -watch.
//
// ⚠️ **Outbound cannot address a watcher and must not try.** Outbound.To is a
// wire.Seat and Server.send reads an invalid To as "the connection this was read
// from", so a body addressed to a watcher would be delivered to whichever player
// happened to be talking. The record is the whole of the room's answer; who is
// watching, and how many, is the transport's to know.
//
// ⚠️ **Memory is bounded, and that is an answer rather than a hope.** An
// unbounded per-room buffer would be a real objection to this design. This one
// holds at most `Battles × (1 + TurnCap)` bodies plus one closure — one
// wire.Start a battle and one wire.Turn a turn, and the cap is what stops a
// battle opening turns for ever — and Config.Validate accepts only 1 or 3
// battles. A body is a decision and a digest, so a whole capped bo3 is thousands
// of small structs rather than a stream.
//
// ⚠️ **A watcher of a DRAFTING room sees nothing yet, and this is where that is
// written down rather than discovered.** draft.go emits wire.Drafted for every
// batch of decisions and a wire.Closed{ClosureDraftExpired} when the pick clock
// runs out, and neither is recorded here — deliberately, because watching a ban
// and pick is step 7 and belongs to TODO.md § *Ban and pick, and a spectator
// watching it* rather than to this one. What a watcher of a drafting room gets
// today is therefore an empty record until the draft closes, and then the whole
// battle from its wire.Start: the mid-joiner path, arriving late by a phase. Do
// not read the record's silence during a draft as a bug in it.

// Since returns the bodies recorded from cursor onward, and the cursor to pass
// next time.
//
// A watcher holds its own cursor and nothing else, which is the shape
// battle.Since and draft.Since already have and the third copy of it rather than
// a new idea. `Since(0)` is a watcher that wants everything, **which is exactly
// what joining halfway is**: the wire.Start of the battle in progress is in the
// record, so a late watcher is handed the roster and the seed it needs to build
// its own mirror and then every turn since — with no Battle.Roster() accessor and
// no second copy of the roster anywhere, because begin() builds it in a local and
// the record holds the body.
//
// It is a Since and **not a Drain** for internal/draft's own reason: a match owes
// its record to two players and however many watchers, so a single-consumer
// cursor that emptied what it read would let whichever of them read first decide
// what the others never see. Many cursors may coexist and none of them can starve
// another.
//
// ⚠️ **It panics on a cursor the record cannot answer**, deliberately rather than
// defensively and for battle.Since's own reason: answering an out-of-range cursor
// with an empty slice would make a consumer that has somehow got ahead of the
// room look exactly like one that is up to date, which is the silent desync a
// cursor exists to prevent — and a cursor is a number this method handed the
// caller itself, so a bad one is a programming error rather than a runtime
// condition.
//
// ⚠️ **The recorded wire.Start is built for a watcher, and its Side is the
// HOST's.** Everything else in it is the players' body unchanged — the same seed,
// the same roster in the same order, the same battle index — but Side is the half
// of the board *this client plays*, and a watcher plays neither. So it watches
// from the host's chair, which is a decision stated here rather than whatever
// happened to fall out of the loop: a watcher has to be given some side to draw
// the board from, the host is the seat a room hands out first, and the alternative
// — a wire.Start with no side — would be a hex.Side zero value that reads as
// SideAlly to everything downstream. → begin, which is the one place it is set.
//
// The slice is a three-index view, so its capacity is its length and a caller's
// own append has to reallocate rather than writing into the slot the next body is
// going to be recorded in. → battle.Since, where that was measured to corrupt
// both the caller's copy and the record.
func (r *Room) Since(cursor int) ([]wire.Body, int) {
	recorded := len(r.watched)
	if cursor < 0 || cursor > recorded {
		panic(fmt.Sprintf("room: Since called with a cursor of %d against a record of %d bodies",
			cursor, recorded))
	}
	if cursor == recorded {
		return nil, recorded
	}
	return r.watched[cursor:recorded:recorded], recorded
}

// Resume is the whole record as one seat would have received it, which is what a
// client that lost its socket needs in order to come back to the board it left.
//
// ⚠️ **The record cannot simply be replayed by a player, and that is the one
// thing about a rejoin that is not obvious.** A wire.Start carries the *side* its
// recipient plays, and the recorded one is taken from the host's chair on
// purpose — a watcher plays neither half, so it is given the seat a room hands
// out first. Handed unchanged to a returning **guest**, it would seat that
// client on the wrong half of its own board: every subsequent digest would
// disagree while both peers were fighting the same battle perfectly, which is
// the failure mode this file's Since already warns about from the other end.
// So each Start is re-sided here, for the seat asking.
//
// ⚠️ **The open prompt is NOT sent and does not need to be**, which retires a
// note that stood in TODO.md since before the record existed: a rejoin was said
// to want a copy of Room.Pending, because passing the room's own *battle.Prompt
// out of its goroutine is exactly the sharing the registry exists to prevent.
// A mirror does not need one. It calls Begin itself and advances to the same
// prompt after the last recorded turn — the same thing a watcher does, and the
// reason no wire.Turn carries the opening board either. What has to travel is the
// decisions; the prompt is derived at both ends from the same events.
//
// The bodies themselves are not deep-copied, for watchedFrom's reason: a
// wire.Body is a value the room recorded and never edits. The Starts are the
// exception and are **replaced** rather than edited, so the record keeps the one
// it holds.
func (r *Room) Resume(seat wire.Seat) []wire.Body {
	// Through Since rather than over the field, and TestTheWatcherRecordIsWrite
	// OnlyFromTheRoom is what insists: the record is named inside exactly two
	// functions, the one that appends and the one that reads, so a third reader
	// asks the second rather than becoming one. It also gets the bounded slice
	// for free.
	recorded, _ := r.Since(0)
	out := make([]wire.Body, 0, len(recorded))
	for _, body := range recorded {
		if opening, isStart := body.(wire.Start); isStart {
			opening.Side = r.sideOf(seat)
			out = append(out, opening)
			continue
		}
		out = append(out, body)
	}
	return out
}

// watch records one body, and it is the only way anything gets into the record.
//
// ⚠️ **Every call to it sits beside the room already emitting the same thing to
// the players**, which is the whole of why the record cannot drift from what was
// played: there are exactly three — the wire.Start of each battle in begin, each
// wire.Turn in resolved, and the wire.Closed of a departure in abandon — and each
// is a line away from the Outbound carrying the same body. A fourth append
// somewhere the players are told nothing would be a watcher seeing a match the
// two of them did not have.
//
// TestTheWatcherRecordIsWriteOnlyFromTheRoom counts those three by name.
func (r *Room) watch(body wire.Body) {
	r.watched = append(r.watched, body)
}
