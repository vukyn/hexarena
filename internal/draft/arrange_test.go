package draft_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// formationCells is n distinct cells of one side's 3x3, which is *an*
// arrangement and not a designed one.
//
// ⚠️ **It walks the grid column by column, and at three units that is the same
// list row-major order produces** — three cells of one column come back exactly
// as they went in — so this fixture cannot tell slot order from pick order and
// is deliberately not what the ordering claim is measured on.
// TestADraftedSquadIsOrderedByItsSlots names its own cells for that reason. At
// five the two orders do differ, which is what makes the 5v5 replay case worth
// having.
func formationCells(n int) []hex.Offset {
	out := make([]hex.Offset, 0, n)
	for at := range n {
		out = append(out, hex.Offset{Col: at / hex.FormationRows, Row: at % hex.FormationRows})
	}
	return out
}

// arrangeSide puts one seat's picks on the first cells of its formation, which is
// the cheapest legal arrangement and the one every test that is not about the
// cells themselves wants.
func arrangeSide(t *testing.T, drafting *draft.Draft, seat wire.Seat) {
	t.Helper()
	picks := drafting.Picks()[seatIndex(t, seat)]
	if err := drafting.Arrange(seat, formationCells(len(picks))); err != nil {
		t.Fatalf("%s arranges its %d picks: %v", seat, len(picks), err)
	}
}

// arrangeBothSides closes the arrange phase, host first.
func arrangeBothSides(t *testing.T, drafting *draft.Draft) {
	t.Helper()
	for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		arrangeSide(t, drafting, seat)
	}
}

// draftedAndArranged is a whole draft of the given format, played out and
// arranged by both sides, which is the state Squads answers in.
func draftedAndArranged(t *testing.T, all []cast.Character, format wire.Format,
	first wire.Seat) *draft.Draft {
	t.Helper()
	drafting, err := draft.New(draft.Config{
		Format: format, Pool: draft.NewPool(all), First: first,
	})
	if err != nil {
		t.Fatalf("set up a %s draft: %v", format, err)
	}
	playOut(t, drafting, all, spendEvery(format))
	arrangeBothSides(t, drafting)
	if !drafting.Done() {
		t.Fatalf("a %s draft was played out and arranged and is not done", format)
	}
	return drafting
}

// TestBothSidesArrangeAtOnceAndTurnNeverSaysSo is the phase's shape, which is the
// whole reason it is a phase rather than more state machine.
//
// Three claims, and each is one clause of TODO.md § "The arrange phase":
// **Turn does not widen** — it answers nothing at all while two arrangements are
// pending, and the phase is asked about through Arranging and
// AwaitingArrangement; **nothing reaches the record until both are in**, so a
// mirror never sees a half-open arrange; and **Squads answers only once the phase
// has closed**, honestly empty until then rather than plausibly wrong.
func TestBothSidesArrangeAtOnceAndTurnNeverSaysSo(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	// Before the picking closes there is no phase to be in, and the accessors say
	// so rather than answering a state nobody is in yet.
	if drafting.Arranging() {
		t.Error("a draft on its first ban has the arrange phase open")
	}
	if got := drafting.AwaitingArrangement(); got != nil {
		t.Errorf("a draft on its first ban is waiting on %v to arrange", got)
	}
	playOut(t, drafting, all, spendEvery(wire.Format3v3))

	// One: Turn answers nothing, and the phase's own accessors answer everything.
	if seat, step, due := drafting.Turn(); due {
		t.Errorf("the picking is over and Turn is still asking %s for a %s, which is the one "+
			"thing this phase is a phase in order not to do", seat, step)
	}
	if !drafting.Arranging() {
		t.Fatal("the picking is over and the arrange phase is not open")
	}
	if got, want := drafting.AwaitingArrangement(),
		[]wire.Seat{wire.SeatHost, wire.SeatGuest}; !slices.Equal(got, want) {
		t.Errorf("both sides have still to arrange and the draft is waiting on %v, want %v "+
			"— in the order a room hands seats out", got, want)
	}

	// Two: the record does not move for the first arrangement.
	_, before := drafting.Since(0)
	arrangeSide(t, drafting, wire.SeatHost)
	if _, after := drafting.Since(0); after != before {
		t.Errorf("the host arranged and the record grew from %d entries to %d: an entry is "+
			"public the moment it is appended, so that is the host's board shown to the guest",
			before, after)
	}
	if got, want := drafting.AwaitingArrangement(), []wire.Seat{wire.SeatGuest}; !slices.Equal(got, want) {
		t.Errorf("the host has arranged and the draft is waiting on %v, want %v", got, want)
	}
	if drafting.Done() {
		t.Error("one side has arranged and the draft calls itself done")
	}
	// Squads answers nothing yet, and what it answers is refused by name rather
	// than fielded as a plausible squad.
	for index, squad := range drafting.Squads() {
		if len(squad.Units) != 0 {
			t.Errorf("side %d has %d units in it with the phase still open", index, len(squad.Units))
		}
		if err := squad.Validate(); err == nil {
			t.Errorf("side %d's half-drafted squad was accepted by Validate", index)
		}
	}

	// Three: both in, and the record catches up with exactly two entries, in
	// seats order.
	arrangeSide(t, drafting, wire.SeatGuest)
	if !drafting.Done() {
		t.Fatal("both sides have arranged and the draft is not done")
	}
	if drafting.Arranging() {
		t.Error("both sides have arranged and the phase is still open")
	}
	if got := drafting.AwaitingArrangement(); got != nil {
		t.Errorf("both sides have arranged and the draft is waiting on %v", got)
	}
	fresh, _ := drafting.Since(before)
	if len(fresh) != seatCount() {
		t.Fatalf("closing the phase recorded %d entries and one arrangement a seat is %d: %+v",
			len(fresh), seatCount(), fresh)
	}
	for at, entry := range fresh {
		if entry.Seat != seats()[at] {
			t.Errorf("entry %d of the pair is %s's and the record goes in seats order, so it "+
				"should be %s's", at, entry.Seat, seats()[at])
		}
		if entry.Step != draft.StepArrange {
			t.Errorf("entry %d of the pair is a %q", at, entry.Step)
		}
		if want := draft.PicksPerSide(wire.Format3v3); len(entry.Slots) != want {
			t.Errorf("%s's arrangement records %d cells and a 3v3 fields %d",
				entry.Seat, len(entry.Slots), want)
		}
	}
	t.Logf("the phase closed at entry %d with two arrangements: %v and %v",
		before, fresh[0].Slots, fresh[1].Slots)
}

// seats is the order a room hands seats out, restated here because the package's
// own array is unexported. A test reading it as a literal is right: the order is
// a decision, so there is nothing to derive it from.
func seats() []wire.Seat { return []wire.Seat{wire.SeatHost, wire.SeatGuest} }

// TestAwaitingArrangementIsInSeatsOrderEveryTime is the map ban held where it
// reaches an output, and it reads the same state **sixty-four times** for a
// measured reason rather than out of caution.
//
// ⚠️ **A draft has two seats, so a map range over them comes out in the right
// order half the time**: with the loop replaced by a `map[wire.Seat]int` range,
// the whole package suite reddened in 17 of 20 runs and passed in 3 — a test that
// has to be re-run to be believed, which is the shape CLAUDE.md § *Mistakes
// already made here* keeps a note about. Sixty-four readings of one unchanging
// state turn that coin flip into a certainty, and it is cheap because the state
// is built once.
func TestAwaitingArrangementIsInSeatsOrderEveryTime(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		// From the GUEST, so the order this answers in cannot be the order the
		// draft happened to hand decisions out in.
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatGuest,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	playOut(t, drafting, all, spendEvery(wire.Format3v3))
	// The premise: both seats are in the answer, because a one-element list has
	// no order to get wrong.
	if got := drafting.AwaitingArrangement(); len(got) != seatCount() {
		t.Fatalf("the phase is waiting on %v, and this test needs both seats in the answer", got)
	}
	const readings = 64
	for at := range readings {
		if got := drafting.AwaitingArrangement(); !slices.Equal(got, seats()) {
			t.Fatalf("reading %d of %d answered %v, and the order a room hands seats out is %v: "+
				"this order reaches a screen and this package's own refusals, so it may not come "+
				"out of a map", at+1, readings, got, seats())
		}
	}
	t.Logf("%d readings, all of them %v", readings, seats())
}

// seatCount is how many seats a draft has, restated for the same reason seats is.
func seatCount() int { return len(seats()) }

// TestADraftedSquadIsOrderedByItsSlots holds the ordering decision, and it is
// held because the consequence is invisible: atb.Queue.Add assigns seq in the
// order it is handed the roster and seq is the last tie-break in the turn order,
// so this slice order is worth up to sixty points of win rate in a mirror.
//
// The order is **row-major** — the Row is the outer loop and the Col the inner —
// and pick order is the alternative that was refused, because it would hand a
// speed tie-break to a decision made for entirely different reasons.
//
// ⚠️ The fixture arranges its picks so that pick order and slot order **disagree**
// and both are asserted, since a squad arranged into the order it was picked in
// would pass an ordering test that measured nothing.
func TestADraftedSquadIsOrderedByItsSlots(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	playOut(t, drafting, all, spendEvery(wire.Format3v3))

	// Three cells whose row-major order is the reverse of the order they are
	// named in, so the sort has to move every one of them.
	cells := []hex.Offset{{Col: 2, Row: 2}, {Col: 1, Row: 1}, {Col: 0, Row: 0}}
	for _, seat := range seats() {
		if err := drafting.Arrange(seat, cells); err != nil {
			t.Fatalf("%s arranges onto %v: %v", seat, cells, err)
		}
	}
	picks := drafting.Picks()
	for index, squad := range drafting.Squads() {
		inPickOrder := []string{}
		for _, one := range picks[index] {
			inPickOrder = append(inPickOrder, one.Character)
		}
		inSlotOrder := []string{}
		standing := []string{}
		for _, unit := range squad.Units {
			inSlotOrder = append(inSlotOrder, unit.Character)
			standing = append(standing, unit.Slot.String())
		}
		// The premise, so a fixture whose two orders agreed could not pass.
		if slices.Equal(inPickOrder, inSlotOrder) {
			t.Fatalf("side %d comes back in the order it picked (%v), so this fixture cannot "+
				"tell slot order from pick order", index, inPickOrder)
		}
		if want := []string{"0,0", "1,1", "2,2"}; !slices.Equal(standing, want) {
			t.Errorf("side %d stands in the order %v and row-major over %v is %v",
				index, standing, cells, want)
		}
		// And the relation stated directly, over every neighbouring pair, so a
		// wider formation is covered by the same claim.
		for at := 1; at < len(squad.Units); at++ {
			previous, next := squad.Units[at-1].Slot, squad.Units[at].Slot
			if previous.Row > next.Row || (previous.Row == next.Row && previous.Col >= next.Col) {
				t.Errorf("side %d has %s before %s, which is not row-major",
					index, previous, next)
			}
		}
		t.Logf("side %d picked %v and stands %v", index, inPickOrder, standing)
	}
}

// TestADraftedSquadIsNamedByTheCharactersItTook is the unit-id decision and the
// reason it is safe: the pool is exclusive, so a character id is unique within a
// side by construction — and across the whole battle, since both sides spend out
// of one pool.
//
// ⚠️ The side prefix Squad.Take adds is untouched by this and still earns its
// keep for its own reason, so what is asserted is both halves: the ids inside a
// squad are the characters, and the ids on the board carry the side.
func TestADraftedSquadIsNamedByTheCharactersItTook(t *testing.T) {
	all := shippedCast(t)
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("parse the embedded cast: %v", err)
	}
	drafting := draftedAndArranged(t, all, wire.Format3v3, wire.SeatHost)
	squads := drafting.Squads()
	picks := drafting.Picks()

	everyID := []string{}
	for index, squad := range squads {
		for _, unit := range squad.Units {
			if unit.ID != unit.Character {
				t.Errorf("side %d fields %q as %q, and a drafted unit is named by its character",
					index, unit.Character, unit.ID)
			}
			everyID = append(everyID, unit.ID)
		}
		// Every pick reached the squad, so the id claim is about all of them.
		if len(squad.Units) != len(picks[index]) {
			t.Errorf("side %d took %d picks and fields %d units",
				len(picks[index]), index, len(squad.Units))
		}
	}
	// Unique across BOTH sides, which is what the exclusive pool buys and is
	// stronger than the per-squad uniqueness Placement.ID needs.
	for at, one := range everyID {
		if slices.Contains(everyID[at+1:], one) {
			t.Errorf("%s is fielded twice in one battle, and one shared exclusive pool cannot "+
				"do that", one)
		}
	}
	// The side prefix, read off what Take produces rather than off the squad.
	roster, err := squads[0].Take(hex.SideAlly, characters)
	if err != nil {
		t.Fatalf("field the host's squad: %v", err)
	}
	for _, unit := range roster {
		if !strings.HasPrefix(unit.ID, hex.SideAlly.String()+".") {
			t.Errorf("%q took the field without its side in front of it, and a squad fought "+
				"against a copy of itself needs two halves a log can tell apart", unit.ID)
		}
	}
	t.Logf("one battle fields %v, each of them once", everyID)
}

// TestTheRecordDoesNotSayWhoArrangedFirst is the price of recording in seats
// order, held as a property rather than left as a remark: two drafts in which the
// same decisions were taken leave the **same** record, whichever side arranged
// first.
//
// Arrival order is a race — it is a fact about two clients' latency and nothing
// about the game — so a record carrying it would make two peers disagree about a
// draft they both watched.
func TestTheRecordDoesNotSayWhoArrangedFirst(t *testing.T) {
	all := shippedCast(t)
	records := map[string][]draft.Entry{}
	for _, order := range [][]wire.Seat{
		{wire.SeatHost, wire.SeatGuest},
		{wire.SeatGuest, wire.SeatHost},
	} {
		drafting, err := draft.New(draft.Config{
			Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
		})
		if err != nil {
			t.Fatalf("set up a 3v3 draft: %v", err)
		}
		playOut(t, drafting, all, spendEvery(wire.Format3v3))
		for _, seat := range order {
			arrangeSide(t, drafting, seat)
		}
		record, _ := drafting.Since(0)
		records[fmt.Sprintf("%s then %s", order[0], order[1])] = record
	}
	hostFirst := records["host then guest"]
	guestFirst := records["guest then host"]
	if len(hostFirst) != len(guestFirst) {
		t.Fatalf("arranging host-first left %d entries and guest-first left %d",
			len(hostFirst), len(guestFirst))
	}
	// The premise: there are arrangements in there at all.
	arrangements := 0
	for at := range hostFirst {
		mine, theirs := hostFirst[at], guestFirst[at]
		if mine.Step == draft.StepArrange {
			arrangements++
		}
		if mine.Seat != theirs.Seat || mine.Step != theirs.Step ||
			mine.Character != theirs.Character || !slices.Equal(mine.Slots, theirs.Slots) {
			t.Errorf("entry %d is\n  %+v\nwhen the host arranged first and\n  %+v\nwhen the "+
				"guest did", at, mine, theirs)
		}
	}
	if arrangements != seatCount() {
		t.Fatalf("the records hold %d arrangements between them, so this comparison is not "+
			"about the arrange phase at all", arrangements)
	}
	t.Logf("%d entries, identical either way round, two of them arrangements", len(hostFirst))
}

// TestATimeoutInTheArrangePhaseDiscardsWhatItHeld is decision (c) reaching the
// last phase: a timeout here cancels the whole draft, and the arrangement one
// side had already made is **thrown away rather than recorded**.
//
// ⚠️ Appending it on the way out would leak exactly what the buffer exists to
// hide, to a draft nobody is going to play. So what is asserted is the record's
// contents and not only the draft's state: one timeout entry, no arrangement.
func TestATimeoutInTheArrangePhaseDiscardsWhatItHeld(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	playOut(t, drafting, all, spendEvery(wire.Format3v3))
	arrangeSide(t, drafting, wire.SeatHost)
	// The premise: something is being held. Without this the test would pass
	// against a phase that never buffered anything.
	if got, want := drafting.AwaitingArrangement(), []wire.Seat{wire.SeatGuest}; !slices.Equal(got, want) {
		t.Fatalf("the host has arranged and the draft is waiting on %v, want %v", got, want)
	}
	_, before := drafting.Since(0)

	if err := drafting.TimedOut(wire.SeatGuest); err != nil {
		t.Fatalf("the guest's allowance runs out while arranging: %v", err)
	}
	if !drafting.Cancelled() {
		t.Error("the guest ran out of time in the arrange phase and the draft is not cancelled")
	}
	if drafting.Done() || drafting.Arranging() {
		t.Errorf("a cancelled draft reports done %v and arranging %v",
			drafting.Done(), drafting.Arranging())
	}
	if got := drafting.AwaitingArrangement(); got != nil {
		t.Errorf("a cancelled draft is waiting on %v to arrange", got)
	}
	fresh, _ := drafting.Since(before)
	if len(fresh) != 1 || fresh[0].Step != draft.StepTimeout || fresh[0].Seat != wire.SeatGuest {
		t.Fatalf("the timeout recorded %+v, and it owes exactly one entry: the guest's timeout",
			fresh)
	}
	record, _ := drafting.Since(0)
	for at, entry := range record {
		if entry.Step == draft.StepArrange {
			t.Errorf("entry %d of %d is an arrangement (%v), and the buffered one is discarded "+
				"rather than recorded — appending it leaks the board it was hiding",
				at, len(record), entry.Slots)
		}
	}
	// And the squads, so the discard is a fact about the state as well as about
	// the record: nothing can be fielded out of a cancelled draft.
	for index, squad := range drafting.Squads() {
		if len(squad.Units) != 0 {
			t.Errorf("side %d has %d units to field out of a cancelled draft",
				index, len(squad.Units))
		}
	}
	t.Logf("%d entries, the last of them the timeout, and no arrangement among them",
		len(record))
}

// TestAnIllegalArrangementKeepsPlacementsOwnWords is the arrange phase's half of
// "one rule, one declaration": the draft adds a **buffer** and not a second set
// of formation rules.
//
// ⚠️ The assertion is that the sentence is placement.Squad.Validate's, produced by
// calling it directly on the same squad. A phase writing its own duplicate-cell
// or off-the-board refusal would be a second declaration of a rule that already
// has one, and the two would drift the day a formation changed shape.
func TestAnIllegalArrangementKeepsPlacementsOwnWords(t *testing.T) {
	all := shippedCast(t)
	for _, one := range []struct {
		what  string
		cells []hex.Offset
	}{
		{"two units on one cell",
			[]hex.Offset{{Col: 0, Row: 0}, {Col: 0, Row: 0}, {Col: 1, Row: 0}}},
		{"a cell off the side's own three columns",
			[]hex.Offset{{Col: hex.FormationCols, Row: 0}, {Col: 0, Row: 1}, {Col: 1, Row: 0}}},
		{"a cell off the board altogether",
			[]hex.Offset{{Col: 0, Row: hex.FormationRows}, {Col: 0, Row: 1}, {Col: 1, Row: 0}}},
		{"a negative cell",
			[]hex.Offset{{Col: -1, Row: 0}, {Col: 0, Row: 1}, {Col: 1, Row: 0}}},
	} {
		t.Run(one.what, func(t *testing.T) {
			drafting, err := draft.New(draft.Config{
				Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
			})
			if err != nil {
				t.Fatalf("set up a 3v3 draft: %v", err)
			}
			playOut(t, drafting, all, spendEvery(wire.Format3v3))

			refused := drafting.Arrange(wire.SeatHost, one.cells)
			if refused == nil {
				t.Fatalf("%v was accepted as an arrangement", one.cells)
			}
			// The same squad, built by hand out of the same picks and the same
			// cells, asked the same question — so what is compared is the wording
			// and not a paraphrase of it. Row-major order is applied here too,
			// because the sentence names the unit the walk reaches second.
			byHand := placement.Squad{ID: string(wire.SeatHost)}
			for at, pick := range drafting.Picks()[seatIndex(t, wire.SeatHost)] {
				byHand.Units = append(byHand.Units, placement.Placement{
					ID: pick.Character, Character: pick.Character,
					Level: pick.Level, Stage: pick.Stage, Slot: one.cells[at],
					Skills: pick.Skills, Passives: pick.Passives,
				})
			}
			slices.SortStableFunc(byHand.Units, func(a, b placement.Placement) int {
				if a.Slot.Row != b.Slot.Row {
					return a.Slot.Row - b.Slot.Row
				}
				return a.Slot.Col - b.Slot.Col
			})
			placements := byHand.Validate()
			if placements == nil {
				t.Fatalf("placement accepts the squad %v builds, so this case is not about one "+
					"of its refusals at all", one.cells)
			}
			if got, want := refused.Error(), "host's arrangement: "+placements.Error(); got != want {
				t.Errorf("the arrangement is refused with\n  %q\nand placement's own words "+
					"behind a lead-in are\n  %q", got, want)
			}
			// The arrangement was refused, so nothing was buffered: a side that
			// named an illegal board is still owed one.
			if got := drafting.AwaitingArrangement(); len(got) != seatCount() {
				t.Errorf("an arrangement was refused and the draft is waiting on %v", got)
			}
			t.Logf("refused with %q", refused)
		})
	}
}

// TestADraftedPairFightsAWholeBattle is the point of the whole step, and it is a
// battle rather than a shape assertion for one reason: everything before the
// arrange phase produced values **nothing could fight with**, so a test that
// checked Squads' shape would not know whether the draft's output is usable.
//
// It drives a whole draft, arranges both sides, fields each squad through
// placement.Squad.Take and hands the pair to battle.New — the same three calls
// internal/room's begin makes, in the same order and with the home side enlisted
// first, because that append is the sixty-point line.
//
// ⚠️ **Two vacuity guards, because "it finished" is a claim a battle that never
// started also satisfies.** The roster has to hold every unit both sides drafted,
// and the battle has to end **on its own** rather than at RunToEndWith's backstop
// — the limit is a hang detector and a run that reaches it has measured nothing
// about whether the squads were fieldable. The outcome and the turn count are
// logged by value rather than asserted, since which side wins is a balance
// reading and not this package's contract.
func TestADraftedPairFightsAWholeBattle(t *testing.T) {
	all := shippedCast(t)
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("parse the embedded books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("parse the embedded cast: %v", err)
	}
	// The backstop, and it is internal/forge's spar limit rather than a smaller
	// guess: reaching it means something genuinely endless is happening, which is
	// a finding rather than a slow test.
	const turnLimit = 4000

	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		drafting := draftedAndArranged(t, all, format, wire.SeatHost)
		squads := drafting.Squads()

		// Home first, which is the order internal/room and forge.FightSquads both
		// append in: atb.Queue.Add assigns seq in the order battle.New is handed
		// its roster and seq is the last tie-break in the turn order.
		roster := make([]battle.Roster, 0, 2*draft.PicksPerSide(format))
		for index, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
			taken, err := squads[index].Take(side, characters)
			if err != nil {
				t.Fatalf("%s: field the %s squad on the %s side: %v",
					format, squads[index].ID, side, err)
			}
			roster = append(roster, taken...)
		}
		// Vacuity guard one: the board holds the whole draft.
		if want := 2 * draft.PicksPerSide(format); len(roster) != want {
			t.Fatalf("%s: a drafted board holds %d units and this one holds %d",
				format, want, len(roster))
		}

		fought, err := battle.New(books, 11, roster)
		if err != nil {
			t.Fatalf("%s: open a battle on two drafted squads: %v", format, err)
		}
		fought.Begin()
		turns, err := fought.RunToEndWith(turnLimit, fought.Suggest)
		if err != nil {
			t.Fatalf("%s: play the battle out: %v", format, err)
		}
		// Vacuity guard two: it ended, and it ended on its own.
		if !fought.Finished() {
			t.Fatalf("%s: the battle was still going after %d turns, which is the backstop "+
				"rather than an ending", format, turns)
		}
		if turns >= turnLimit {
			t.Errorf("%s: the battle took %d turns of a %d-turn backstop, so it finished at the "+
				"limit rather than by being won", format, turns, turnLimit)
		}
		if fought.Outcome() == battle.Undecided {
			t.Errorf("%s: the battle reports itself finished with no outcome", format)
		}
		winner := "nobody"
		if side, decided := fought.Winner(); decided {
			winner = side.String()
		}
		names := []string{}
		for _, unit := range roster {
			names = append(names, unit.ID)
		}
		t.Logf("%s: %v fought to %q in %d turns, won by %s",
			format, names, fought.Outcome(), turns, winner)
	}
}
