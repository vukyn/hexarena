package draft

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/wire"
)

// Arrange is one seat's whole arrangement: the cell each of its picks stands on,
// in pick order, so `slots[i]` is the cell for `Picks()[seat][i]` and a caller
// does not have to name anybody.
//
// ⚠️ **A side arranges in one call and not a slot at a time.** The phase is
// simultaneous and secret, so a slot-by-slot stream would need exactly the same
// buffer and would additionally give a peer a *partial* arrangement to be told
// about. One call is also what makes "append both together" fall out of the
// design rather than be arranged for: there is one moment per side at which
// anything could be recorded, and the second of the two is when both are.
//
// ⚠️ **Nothing reaches the record until both arrangements are in**, and then both
// go in at once, in **seats** order rather than arrival order. An entry is public
// the moment it is appended and a mirror replaying the record computes the state,
// so appending the first arrangement when it arrives *is* showing it to the other
// player — the one thing this phase exists to prevent. Arrival order is a race, so
// recording it would make two peers' records differ for a draft in which the same
// decisions were taken; the price is that the record cannot say who arranged
// first, which is a fact a mirror has no use for. → Draft.arranged, and
// wire.DraftDecision.Slots.
//
// ⚠️ **The legality of the arrangement is placement.Squad.Validate's answer and
// its words are kept**, behind a lead-in naming the seat. That function already
// refuses an empty squad, more than five, a blank or duplicated unit id, a level
// outside the range, a cell off the 3x3 and two units on one cell — so there is
// no duplicate-cell rule and no off-the-board rule written here. The lead-in is
// there for the reason Loadout's is: two arrangements are pending at once, so a
// sentence naming only the unit would not say whose arrangement was turned away.
// → CLAUDE.md § "One rule, one declaration".
func (d *Draft) Arrange(seat wire.Seat, slots []hex.Offset) error {
	index, err := d.dueToArrange(seat, len(slots))
	if err != nil {
		return err
	}
	if err := d.squadAt(index, slots).Validate(); err != nil {
		return fmt.Errorf("%s's arrangement: %w", seat, err)
	}
	// Cloned, so a caller that reuses or edits its own slice cannot reach into
	// the buffer — the same depth Pick.clone and Loadout's record hand out at.
	d.arranged[index] = slices.Clone(slots)
	if !d.arrangedBoth() {
		return nil
	}
	// Both in: the record catches up in one step, in seats order.
	for at, recording := range seats {
		d.record(wire.DraftEntry{
			Seat: recording, Step: wire.StepArrange, Slots: slices.Clone(d.arranged[at]),
		})
	}
	return nil
}

// Arranging reports whether the arrange phase is open: the picking is over and
// at least one side has still to arrange.
//
// It is the phase's own accessor rather than a step out of Turn, because two
// decisions are pending at once and Turn answers one. → Turn.
func (d *Draft) Arranging() bool { return d.Picked() && !d.arrangedBoth() }

// AwaitingArrangement is which seats have still to arrange, in the order a room
// hands seats out, and nil when the phase is not open at all — a draft still
// picking, one that is done, and one that was cancelled.
//
// ⚠️ **It is derived from the seats array and never from a map**, for the reason
// seatCount is an array: a map range would randomise the order, and here that
// order reaches an output directly — a screen naming who the phase is waiting on,
// and this package's own refusals.
func (d *Draft) AwaitingArrangement() []wire.Seat {
	if !d.Arranging() {
		return nil
	}
	out := make([]wire.Seat, 0, seatCount)
	for index, seat := range seats {
		if len(d.arranged[index]) == 0 {
			out = append(out, seat)
		}
	}
	return out
}

// Squads is what a whole draft produces: each side's picks as a fieldable squad,
// indexed by seat in the order a room hands seats out — **[0] is wire.SeatHost
// and [1] is wire.SeatGuest**, whichever of the two decided first.
//
// ⚠️ **It answers only once Done**, and until then it answers two squads with
// nobody in them. That is deliberately an *honestly incomplete* output rather
// than a plausible one: hex.Offset's zero value is a real cell, so a squad built
// out of unarranged picks would look authored and would then be refused at the
// moment it was fought, naming a cell nobody chose. A squad with no units is
// refused by Validate's own first line, by name. → Pick.
//
// ⚠️ **The units are ordered by SLOT, and that order decides who wins a speed
// tie.** atb.Queue.Add assigns seq in the order it is handed the roster and seq
// is the last tie-break in the turn order, so the slice order is worth up to
// sixty points of win rate in a mirror (CLAUDE.md § the layer rule, and
// wire.Start.Roster's own comment). → squadAt for which order and why that one.
func (d *Draft) Squads() [seatCount]placement.Squad {
	var out [seatCount]placement.Squad
	if !d.Done() {
		return out
	}
	for index := range seats {
		out[index] = d.squadAt(index, d.arranged[index])
	}
	return out
}

// squadAt is one side's picks and one side's cells as a placement.Squad, and it
// is the single place a Pick becomes a Placement — Arrange builds one to ask
// Validate about it and Squads builds one to hand out, so the squad that was
// checked and the squad that is fielded cannot differ.
//
// It reads slots as a parameter rather than off the buffer because Arrange has to
// check an arrangement *before* storing it, and a store-then-roll-back would be a
// second state to get wrong.
//
// ⚠️ **The unit id is the CHARACTER id, and the pool is what licenses it.** A
// placement.Placement.ID has to be unique within the squad; every ban and every
// pick takes a character out of one shared exclusive pool, so a side's picks are
// different characters by construction — and in fact so are both sides' together,
// since they spend out of the same pool. It also makes a log readable
// (`ally.pokemon.gible`), where `a1`/`a2` would not. The side prefix Squad.Take
// adds is still needed for its own reason — a squad fought against a copy of
// itself — and nothing here touches that. ⚠️ CLAUDE.md's "one squad may field the
// same character twice" is about a **saved** squad; both hold, and this is where
// the scope becomes load-bearing rather than descriptive. → Pick.
//
// ⚠️ **The units are ordered by slot, ROW-MAJOR: the Row is the outer loop and
// the Col is the inner**, so a side reads across its formation one row at a time.
// Pick order was the alternative and it is refused: the arrangement is the last
// decision taken and the only one that is *about the board*, so letting the board
// decide board order is the least surprising thing available — whereas pick order
// would hand a speed tie-break to a decision made for entirely different reasons,
// invisibly. Of the two axes the Row is the one the engine ignores everywhere
// else (reach is counted in ranks, so the Col already decides who is reached
// first and the Row decides nothing), which is why the tie-break keys on it: it
// does not compound the placement advantage the depth already carries.
// TestADraftedSquadIsOrderedByItsSlots holds the choice so it cannot drift.
//
// The sort is stable so that two units on one cell — which Validate refuses on
// the next line — keep their pick order rather than being ordered by whichever
// way the sort happened to fall.
func (d *Draft) squadAt(index int, slots []hex.Offset) placement.Squad {
	picks := d.taken[index]
	units := make([]placement.Placement, 0, len(picks))
	for at, one := range picks {
		units = append(units, placement.Placement{
			ID:        one.Character,
			Character: one.Character,
			Level:     one.Level,
			Stage:     one.Stage,
			Slot:      slots[at],
			Skills:    slices.Clone(one.Skills),
			Passives:  slices.Clone(one.Passives),
		})
	}
	slices.SortStableFunc(units, func(a, b placement.Placement) int {
		if a.Slot.Row != b.Slot.Row {
			return a.Slot.Row - b.Slot.Row
		}
		return a.Slot.Col - b.Slot.Col
	})
	// The seat is the whole of what names a drafted squad — there is nothing else
	// to call it, and it is what placement.Squad.Take puts in front of its own
	// refusals. No Name, which is display text nobody authored here.
	return placement.Squad{ID: string(seats[index]), Units: units}
}

// arrangedBoth reports whether both sides have arranged, which is the phase
// closing.
//
// It ranges the seats array rather than a map, and it reaches Done, Arranging and
// AwaitingArrangement — three outputs — so the ordering rule applies here as
// much as anywhere.
func (d *Draft) arrangedBoth() bool {
	for index := range seats {
		if len(d.arranged[index]) == 0 {
			return false
		}
	}
	return true
}

// dueToArrange is the whole of "may this seat arrange now", and every refusal it
// hands back is a sentence saying what cannot happen and why. It answers the
// seat's position, so a caller that got past it has one.
func (d *Draft) dueToArrange(seat wire.Seat, named int) (int, error) {
	switch {
	case d.Cancelled():
		return 0, fmt.Errorf("this draft was cancelled when %s ran out of time, so %q cannot "+
			"arrange: a draft that runs out of time is not resumed, it is played again from a "+
			"new room code", d.abandoned, seat)
	case !d.Picked():
		onTurn, open, _ := d.Turn()
		return 0, fmt.Errorf("the draft is waiting on %s to %s, so %q cannot arrange yet: both "+
			"sides arrange once every ban is spent and every pick has its loadout, and not "+
			"before", onTurn, open, seat)
	case d.arrangedBoth():
		return 0, fmt.Errorf("both sides have arranged and this draft is finished, so %q cannot "+
			"arrange again", seat)
	}
	index, seated := indexOf(seat)
	if !seated {
		return 0, fmt.Errorf("%q is not one of the two seats a room hands out, so it has no "+
			"arrangement to make in this draft", seat)
	}
	// Told apart from a seat that never arranged, because the two are different
	// mistakes: this one is a caller answering a question it has already
	// answered, and the advice is that an arrangement is made once.
	if len(d.arranged[index]) > 0 {
		return 0, fmt.Errorf("%s has already arranged, and an arrangement is made once: this "+
			"draft is waiting on %s", seat, wordSeats(d.AwaitingArrangement()))
	}
	if picks := len(d.taken[index]); named != picks {
		return 0, fmt.Errorf("%s drafted %d units and this arrangement names %d cells: a side "+
			"arranges its whole squad in one call, and slots[i] is the cell for its i-th pick",
			seat, picks, named)
	}
	return index, nil
}

// wordSeats writes a list of seats the way a sentence wants them, which is the
// same care characterCount takes over a count of one: "host and guest" rather
// than a bracketed slice a reader has to decode.
func wordSeats(list []wire.Seat) string {
	names := make([]string, 0, len(list))
	for _, seat := range list {
		names = append(names, string(seat))
	}
	switch len(names) {
	case 0:
		// Not reachable from any refusal here — every one of them is worded while
		// somebody is still owed — and it says so rather than answering an empty
		// string a sentence would swallow.
		return "nobody"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
