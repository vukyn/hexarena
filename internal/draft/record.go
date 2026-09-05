package draft

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/wire"
)

// A draft's record is a run of **wire.DraftEntry**, and that type is not
// declared here.
//
// ⚠️ **It was, until the record went on the wire.** The record is exactly what a
// mirror is handed — a draft is a pure function of the decisions taken, so the
// decisions are the whole of what a room owes a client — which makes the record
// a protocol shape. internal/wire may not import this package (Config.Format and
// every seat here are wire's, so it would be a cycle), so the shape is declared
// there and named from here, and there is no local alias for the reason there is
// none for Seat: one spelling, and it is the protocol's. The payoff is that
// Since's answer is *already* what wire.Drafted carries, with nothing to convert
// and nothing to hold in step. → wire.DraftEntry, wire.DraftDecision, and
// TODO.md § *The draft on the wire*.
//
// ⚠️ **An entry carries the decision and nothing derived from it, and that is
// what makes the mirror work.** A client replays the decisions and computes the
// state — the remaining pool, both sides' picks, whose turn it is — exactly as it
// does for a battle. An entry holding "the pool after this ban", or the form a
// loadout *resolved* to rather than the form it *named*, would be a second
// statement of something the replay already computes, and therefore the one
// place two peers could come to disagree. There is no snapshot in here for the
// same reason there is no clock in this package. → Apply, which is the switch
// that takes a record back through the decisions it names.

// Since returns the entries recorded from cursor onward, and the cursor to pass
// next time.
//
// A consumer holds its own cursor and nothing else, which is the shape
// battle.Since already has and the reason it is copied rather than a Drain:
// **a draft owes its record to two players and a spectator**, so a
// single-consumer cursor that emptied what it read would let whichever of them
// read first decide what the others never see. `Since(0)` is a consumer that
// wants everything, which is what a spectator joining a draft in progress is.
//
// It **panics** on a cursor the record cannot answer, deliberately rather than
// defensively and for battle.Since's own reason: answering an out-of-range
// cursor with an empty slice would make a consumer that has somehow got ahead of
// the draft look exactly like one that is up to date, which is the silent desync
// a cursor exists to prevent — and a cursor is a number this method handed the
// caller itself, so a bad one is a programming error rather than a runtime
// condition.
//
// ⚠️ **The copy is of the slice, not of the entries.** An entry carries two
// slices of its own and they stay shared, which is the depth Pool.All hands out
// at as well; what this stops is a caller changing the record's shape. Recording
// clones what a caller named (→ Loadout), so what is shared here is a list
// nobody outside this package still holds a reference to.
func (d *Draft) Since(cursor int) ([]wire.DraftEntry, int) {
	recorded := len(d.entries)
	if cursor < 0 || cursor > recorded {
		panic(fmt.Sprintf("draft: Since called with a cursor of %d against a record of %d entries",
			cursor, recorded))
	}
	if cursor == recorded {
		return nil, recorded
	}
	// Three-index, so the view's capacity is its length and a caller's own
	// append has to reallocate. Sharing the record's spare capacity corrupts
	// BOTH sides: the caller's append writes into the slot the next decision is
	// going to be recorded in, and that decision then overwrites what the caller
	// appended. Measured for the battle's record in
	// TestAViewAndTheRecordSurviveEachOthersAppends and for this one in
	// TestAViewAndTheDraftRecordSurviveEachOthersAppends; a copy would answer as
	// well and cost more.
	return d.entries[cursor:recorded:recorded], recorded
}

// record appends one entry, which is the only way anything gets into the record.
func (d *Draft) record(entry wire.DraftEntry) {
	d.entries = append(d.entries, entry)
}
