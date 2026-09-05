package draft

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/wire"
)

// Entry is one thing that happened to a draft, as it was decided.
//
// ⚠️ **An entry carries the decision and nothing derived from it, and that is
// what makes the mirror work.** A client replays the decisions and computes the
// state — the remaining pool, both sides' picks, whose turn it is — exactly as it
// does for a battle. An entry holding "the pool after this ban", or the form a
// loadout *resolved* to rather than the form it *named*, would be a second
// statement of something the replay already computes, and therefore the one
// place two peers could come to disagree. There is no snapshot in here for the
// same reason there is no clock in this package.
//
// The record is replayed by taking each entry back through the decision it
// names: a StepBan with a character is Ban and one without is SkipBan, a
// StepPick is Pick, a StepLoadout is Loadout and a StepTimeout is TimedOut.
// TestARecordReplaysIntoTheSameDraft is what does that today; the day a **second**
// caller needs it — a client mirroring a draft, which is step 3 — that switch
// belongs in this package rather than beside each caller, for the reason
// cast.ChooseLoadout was pulled together.
type Entry struct {
	// Seat is who decided. On a StepTimeout it is the seat whose allowance ran
	// out, which is the seat that was being asked.
	Seat wire.Seat
	// Step is which of the four things this entry is.
	Step Step
	// Character is the id banned or picked.
	//
	// ⚠️ **Empty on a StepBan is the skip**, and it is the whole of how a skip is
	// recorded: a ban that names nobody is a ban slot spent on nobody, and there
	// is no third state, so a `Skipped` flag beside this would be a second
	// statement of one fact and two fields that could disagree. Empty on a
	// StepLoadout and a StepTimeout because neither names a character — the
	// loadout's is the pick's, one entry earlier.
	Character string
	// Stage is the form a loadout **named**, which is progression.Furthest — the
	// empty string — for "the furthest the cap reaches" and a form's name on a
	// line that forks.
	//
	// ⚠️ It is what was named and never what it resolved to. Pick.Stage is the
	// resolved one, and it is resolved again by the replay from this.
	Stage string
	// Skills and Passives are the loadout as it was named, before
	// cast.ChooseLoadout was asked about it. Empty on every other step.
	Skills   []string
	Passives []string
}

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
// ⚠️ **The copy is of the slice, not of the entries.** An Entry carries two
// slices of its own and they stay shared, which is the depth Pool.All hands out
// at as well; what this stops is a caller changing the record's shape. Recording
// clones what a caller named (→ Loadout), so what is shared here is a list
// nobody outside this package still holds a reference to.
func (d *Draft) Since(cursor int) ([]Entry, int) {
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
func (d *Draft) record(entry Entry) {
	d.entries = append(d.entries, entry)
}
