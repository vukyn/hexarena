package draft_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// callersOwnRow is an entry no draft records, so finding it inside the record —
// or failing to find it in the caller's own slice — names an aliasing fault
// exactly rather than reporting two entries that merely differ. It is the same
// fixture internal/core/battle's cursor test uses, for the same reading.
var callersOwnRow = wire.DraftEntry{Seat: wire.Seat("the-callers-own-row"), Step: wire.StepBan}

// stateOf is everything a draft says about itself, as one line.
//
// It is the whole observable state on purpose, because it is what a replay has
// to reproduce: whose turn it is and which decision, what the pool has left,
// what each side has taken and in what loadout, who has still to arrange, the
// squads a finished draft fields, and whether it is done or abandoned. One
// string rather than a field-by-field comparison so that a divergence prints as
// a readable diff — and so that a field added to Pick without being added here
// is the only way this can quietly stop measuring, which is cheaper to notice
// than a comparison that silently skipped one.
//
// ⚠️ **It reads Squads as well as Picks, and that is the half the arrange phase
// added.** A comparison that stopped at the picks would call two drafts equal
// while their units stood in different cells — and the cells are worth nineteen
// points of win rate (TODO.md § "Ban and pick" (g)), so they are the last thing
// a mirror may be allowed to differ on.
func stateOf(drafting *draft.Draft) string {
	turn := "nothing due"
	if seat, step, due := drafting.Turn(); due {
		turn = fmt.Sprintf("%s %s", seat, step)
	}
	sides := []string{}
	for _, side := range drafting.Picks() {
		taken := []string{}
		for _, one := range side {
			taken = append(taken, fmt.Sprintf("%s@%d/%s[%s]{%s}",
				one.Character, one.Level, one.Stage,
				strings.Join(one.Skills, " "), strings.Join(one.Passives, " ")))
		}
		sides = append(sides, strings.Join(taken, ", "))
	}
	awaiting := []string{}
	for _, seat := range drafting.AwaitingArrangement() {
		awaiting = append(awaiting, string(seat))
	}
	fielded := []string{}
	for _, squad := range drafting.Squads() {
		standing := []string{}
		for _, unit := range squad.Units {
			standing = append(standing, fmt.Sprintf("%s@%s", unit.ID, unit.Slot))
		}
		fielded = append(fielded, fmt.Sprintf("%s(%s)", squad.ID, strings.Join(standing, " ")))
	}
	return fmt.Sprintf("turn %s | picked %v | done %v | cancelled %v | awaiting %v | pool %v | "+
		"host %s | guest %s | squads %s",
		turn, drafting.Picked(), drafting.Done(), drafting.Cancelled(), awaiting,
		candidateIDs(drafting), sides[0], sides[1], strings.Join(fielded, " vs "))
}

// TestTheRecordHoldsTheDecisionAndNothingDerived walks a whole draft and holds
// the record against what was decided, entry by entry.
//
// ⚠️ **The load-bearing assertion is the loadout's form**: the record keeps what
// was *named* — the empty string for "the furthest the cap reaches" — while the
// pick carries what it *resolved to*. An entry holding the resolved name would
// be a second statement of something the replay computes, and therefore the one
// place two peers could disagree. So this drives a loadout that names no form on
// a line that does not fork, which is the only state where the two differ, and
// asserts they differ.
func TestTheRecordHoldsTheDecisionAndNothingDerived(t *testing.T) {
	all := shippedCast(t)
	drafting := draftAtItsFirstPick(t, all)
	taken := firstCandidate(t, drafting)
	// The premise: a line that does not fork, so progression.Furthest has an
	// answer and "named" and "resolved" are two different strings.
	arms, err := taken.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("the forms %s reaches at the cap: %v", taken.ID, err)
	}
	if len(arms) != 1 {
		t.Fatalf("%s reaches %v at the cap, and this test needs a line that does not fork",
			taken.ID, progression.StageNames(arms))
	}
	skills := taken.SkillsAt(progression.LevelCap, progression.Furthest)
	if len(skills) < cast.SkillSlots {
		t.Fatalf("%s knows %d skills at the cap and a loadout brings %d",
			taken.ID, len(skills), cast.SkillSlots)
	}
	kit := skills[:cast.SkillSlots]

	if err := drafting.Pick(wire.SeatHost, taken.ID); err != nil {
		t.Fatalf("the host picks %s: %v", taken.ID, err)
	}
	if err := drafting.Loadout(wire.SeatHost, progression.Furthest, kit, nil); err != nil {
		t.Fatalf("the host's loadout for %s: %v", taken.ID, err)
	}

	record, cursor := drafting.Since(0)
	if cursor != len(record) {
		t.Errorf("Since(0) answered %d entries and a cursor of %d", len(record), cursor)
	}
	want := []wire.DraftEntry{
		{Seat: wire.SeatHost, Step: wire.StepBan},
		{Seat: wire.SeatGuest, Step: wire.StepBan},
		{Seat: wire.SeatHost, Step: wire.StepBan},
		{Seat: wire.SeatGuest, Step: wire.StepBan},
		{Seat: wire.SeatHost, Step: wire.StepPick, Character: taken.ID},
		{Seat: wire.SeatHost, Step: wire.StepLoadout, Skills: kit},
	}
	if len(record) != len(want) {
		t.Fatalf("four skipped bans, a pick and a loadout are six decisions and the record "+
			"holds %d: %+v", len(record), record)
	}
	for at, entry := range record {
		if entry.Seat != want[at].Seat || entry.Step != want[at].Step ||
			entry.Character != want[at].Character || entry.Stage != want[at].Stage ||
			!slices.Equal(entry.Skills, want[at].Skills) ||
			!slices.Equal(entry.Passives, want[at].Passives) {
			t.Errorf("entry %d is\n  %+v\nand the decision taken was\n  %+v", at, entry, want[at])
		}
	}
	// The two halves of the claim, stated apart: the record names no form and the
	// pick carries one.
	loadout := record[len(record)-1]
	if loadout.Stage != progression.Furthest {
		t.Errorf("the loadout named no form and the record kept %q, which is what the pick "+
			"resolved to rather than what was decided", loadout.Stage)
	}
	if got := drafting.Picks()[seatIndex(t, wire.SeatHost)][0].Stage; got != arms[0].Name {
		t.Errorf("the pick fielded %q and the cap reaches %q", got, arms[0].Name)
	}
	// A skipped ban is an entry that names nobody, and that absence is the whole
	// of how a skip is recorded — a replay reads it as SkipBan.
	if record[0].Character != "" {
		t.Errorf("a skipped ban recorded the character %q", record[0].Character)
	}
	t.Logf("%d entries; the loadout named %q and the pick fielded %q",
		len(record), loadout.Stage, arms[0].Name)
}

// TestAViewAndTheDraftRecordSurviveEachOthersAppends is battle.Since's
// load-bearing test brought onto this record, because the fault it holds off
// corrupts BOTH sides and neither side can see the other.
//
// A view into an append-only record shares the record's backing array. If that
// view carried the array's spare capacity, a caller's own append would write
// into the slot the next decision is about to be recorded in — so the draft would
// overwrite what the caller appended, and a caller appending after the draft had
// already recorded there would destroy an entry out of the record. Both
// directions are driven here, in that order, off one view.
//
// ⚠️ The sweep is over sixteen record lengths rather than one reading, because
// the fault is only *reachable* while the record's array has spare capacity and
// a caller cannot ask a slice about somebody else's capacity. `checked` counts
// the readings that could have seen anything at all.
func TestAViewAndTheDraftRecordSurviveEachOthersAppends(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format5v5, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 5v5 draft: %v", err)
	}

	checked := 0
	for {
		_, _, due := drafting.Turn()
		if !due {
			break
		}
		view, cursor := drafting.Since(0)
		before := cursor
		if len(view) != before {
			t.Fatalf("Since(0) answered %d entries against a cursor of %d", len(view), before)
		}
		// The property stated directly, which is the sharpest form of it: a
		// view's capacity is its length, so a caller's append cannot reach the
		// record's next slot however the runtime grew the array. Reported rather
		// than fatal, so the two corruption readings below still run and say
		// what the spare capacity actually costs.
		if cap(view) != len(view) {
			t.Errorf("at %d entries the view carries %d of spare capacity",
				before, cap(view)-len(view))
		}

		// One: the caller appends, then the draft records.
		early := append(view, callersOwnRow) //nolint:gocritic // the append is the measurement
		takeOneDecision(t, drafting, all)
		_, after := drafting.Since(0)
		if after == before {
			t.Fatalf("a decision was taken at %d entries and the record did not grow", before)
		}
		checked++
		if early[before].Seat != callersOwnRow.Seat {
			t.Errorf("at %d entries the draft's next record overwrote the caller's own row "+
				"with %+v", before, early[before])
		}

		// Two: the same view, appended to now the record has grown past it.
		late := append(view, callersOwnRow) //nolint:gocritic // the append is the measurement
		if late[before].Seat != callersOwnRow.Seat {
			t.Errorf("at %d entries a late append did not keep the caller's own row, got %+v",
				before, late[before])
		}
		record, _ := drafting.Since(0)
		for at, entry := range record {
			if entry.Seat == callersOwnRow.Seat {
				t.Errorf("the caller's own row reached the record at %d of %d, over the entry "+
					"that belonged there", at, len(record))
			}
		}
	}
	if checked == 0 {
		t.Fatal("the draft recorded nothing, so this fixture measured nothing")
	}
	t.Logf("%d readings, each with a decision recorded after it", checked)
}

// takeOneDecision advances a draft by exactly one decision, whichever kind is
// due. It is playOut's body with the loop taken off, for the tests that have to
// look at the record between two decisions.
func takeOneDecision(t *testing.T, drafting *draft.Draft, all []cast.Character) {
	t.Helper()
	seat, step, due := drafting.Turn()
	if !due {
		t.Fatal("the draft has nothing due, so there is no decision to take")
	}
	switch step {
	case wire.StepBan:
		if err := drafting.Ban(seat, firstCandidate(t, drafting).ID); err != nil {
			t.Fatalf("%s bans: %v", seat, err)
		}
	case wire.StepPick:
		if err := drafting.Pick(seat, firstCandidate(t, drafting).ID); err != nil {
			t.Fatalf("%s picks: %v", seat, err)
		}
	case wire.StepLoadout:
		side := drafting.Picks()[seatIndex(t, seat)]
		open := side[len(side)-1]
		form, skills, passives := legalLoadout(t, characterNamed(t, all, open.Character))
		if err := drafting.Loadout(seat, form, skills, passives); err != nil {
			t.Fatalf("%s's loadout for %s: %v", seat, open.Character, err)
		}
	default:
		t.Fatalf("the draft asked for a %q, which is not a decision a seat can make", step)
	}
}

// TestSinceRefusesACursorTheRecordCannotAnswer is battle.Since's panic, kept
// rather than softened.
//
// Answering an out-of-range cursor with an empty slice would make a consumer
// that has somehow got ahead of the draft look exactly like one that is up to
// date — the silent desync a cursor exists to prevent — and a cursor is a number
// Since handed the caller itself, so a bad one is a programming error rather
// than a runtime condition.
func TestSinceRefusesACursorTheRecordCannotAnswer(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	takeOneDecision(t, drafting, all)
	_, recorded := drafting.Since(0)
	if recorded == 0 {
		t.Fatal("one decision recorded nothing, so every reading below is of an empty record")
	}

	// The cursor at the end of the record is legal and answers nothing, which is
	// a consumer that is up to date.
	if fresh, cursor := drafting.Since(recorded); fresh != nil || cursor != recorded {
		t.Errorf("a cursor at the end of a record of %d answered %v and a cursor of %d",
			recorded, fresh, cursor)
	}
	for _, cursor := range []int{-1, recorded + 1, recorded + 100} {
		func() {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Errorf("a cursor of %d against a record of %d was answered rather than "+
						"refused", cursor, recorded)
				}
			}()
			drafting.Since(cursor)
		}()
	}
}

// replayCase is one whole draft to record and replay.
type replayCase struct {
	what   string
	format wire.Format
	first  wire.Seat
	// bans is read in the order the ban decisions come: true spends the slot,
	// false skips it. The mixture matters — a skip records an entry that names
	// nobody, which is a different thing for a replay to read.
	bans []bool
	// timeoutAfter cancels the draft after this many decisions rather than
	// playing it out, and is nought for a draft played to the end. A cancelled
	// draft has to replay as cancelled or a mirror would show a lobby that is
	// still waiting for a pick.
	timeoutAfter int
	// arrange is the order the two sides arrange in once the picking closes, and
	// is empty for a case that stops at the picking. ⚠️ It is a **case
	// parameter** because arrival order is the one thing the record deliberately
	// does not carry — the entries go in seats order — so a case arranging
	// guest-first is what proves the replay converges without it.
	arrange []wire.Seat
}

// TestARecordReplaysIntoTheSameDraft is the property the whole record exists
// for: **a draft is a pure function of the decisions taken**, so a client handed
// nothing but the decisions computes exactly the draft the server holds. It is
// the mirror trick that internal/room already rests on, one layer earlier.
//
// The comparison is against the original's state after **every** decision rather
// than only at the end, and that is deliberate: the remaining pool is only
// observable while a character decision is open, so a comparison taken at the
// end alone could not see the pool at all — and a replay that diverged in the
// middle and converged by the end would pass it.
//
// Two vacuity guards, because a walk that ran nothing passes every assertion in
// here: the record must not be empty, and the replay must have applied at least
// one decision. Both are fatal and both log their figure.
//
// ⚠️ **One comparison is deliberately relaxed, and it is the arrange phase's
// price.** The two arrangements are recorded in seats order rather than arrival
// order, so replaying the first entry puts the *host's* arrangement into the
// mirror however the two really arrived — and in a draft the guest arranged
// first, the original's state at that point says it was waiting on the host. The
// two states are equal again the moment the second entry is applied. So the
// comparison is skipped for a StepArrange that leaves the phase open, that skip
// is counted, and the states are compared by value at the end. That is the
// design being measured rather than an exception carved out of it: arrival order
// is a race, so a record carrying it would make two peers' records differ for a
// draft in which the same decisions were taken.
func TestARecordReplaysIntoTheSameDraft(t *testing.T) {
	all := shippedCast(t)
	for _, one := range []replayCase{
		{what: "a 3v3 from the host with every ban spent",
			format: wire.Format3v3, first: wire.SeatHost, bans: []bool{true, true, true, true}},
		{what: "a 3v3 from the guest with every ban skipped",
			format: wire.Format3v3, first: wire.SeatGuest, bans: []bool{false, false, false, false}},
		{what: "a 3v3 from the host with two bans spent and two skipped",
			format: wire.Format3v3, first: wire.SeatHost, bans: []bool{true, false, false, true}},
		{what: "a 5v5 from the host with every ban spent, which is the pool with nothing to spare",
			format: wire.Format5v5, first: wire.SeatHost, bans: spendEvery(wire.Format5v5)},
		{what: "a 3v3 abandoned by a timeout in the middle of the picking",
			format: wire.Format3v3, first: wire.SeatHost, bans: []bool{true, false, true, false},
			timeoutAfter: 7},
		{what: "a 3v3 played out and arranged, the host first",
			format: wire.Format3v3, first: wire.SeatHost, bans: []bool{true, false, true, false},
			arrange: []wire.Seat{wire.SeatHost, wire.SeatGuest}},
		{what: "a 3v3 played out and arranged, the GUEST first, which the record cannot say",
			format: wire.Format3v3, first: wire.SeatHost, bans: []bool{true, true, true, true},
			arrange: []wire.Seat{wire.SeatGuest, wire.SeatHost}},
		{what: "a 5v5 played out and arranged, five units on nine cells",
			format: wire.Format5v5, first: wire.SeatGuest, bans: spendEvery(wire.Format5v5),
			arrange: []wire.Seat{wire.SeatGuest, wire.SeatHost}},
	} {
		config := draft.Config{
			Format: one.format, Pool: draft.NewPool(all), First: one.first,
		}
		played, err := draft.New(config)
		if err != nil {
			t.Fatalf("%s: set up: %v", one.what, err)
		}

		// The original, with its state kept after every decision. states[0] is
		// the draft before anything was decided.
		states := []string{stateOf(played)}
		for {
			_, _, due := played.Turn()
			if !due {
				break
			}
			if one.timeoutAfter > 0 && len(states) > one.timeoutAfter {
				seat, _, _ := played.Turn()
				if err := played.TimedOut(seat); err != nil {
					t.Fatalf("%s: %s runs out of time: %v", one.what, seat, err)
				}
				states = append(states, stateOf(played))
				break
			}
			takeOneDecision(t, played, all)
			states = append(states, stateOf(played))
		}
		// The arrange phase, which Turn does not answer for, so it is driven from
		// outside the loop above rather than inside it.
		for _, seat := range one.arrange {
			arrangeSide(t, played, seat)
			states = append(states, stateOf(played))
		}
		// The case's own premise: a played-out draft closes its picking, one that
		// arranged is done, and an abandoned one is abandoned. Without this a walk
		// that stopped early would be replayed faithfully and prove nothing about
		// a whole draft.
		switch {
		case one.timeoutAfter > 0:
			if !played.Cancelled() {
				t.Fatalf("%s: the draft was meant to be abandoned and is not", one.what)
			}
		case len(one.arrange) > 0:
			if !played.Done() {
				t.Fatalf("%s: the draft was meant to be arranged and is not done", one.what)
			}
		default:
			if !played.Picked() {
				t.Fatalf("%s: the draft was meant to be played out and its picking is not over",
					one.what)
			}
			if played.Done() {
				t.Fatalf("%s: nobody arranged and the draft calls itself done", one.what)
			}
		}

		record, _ := played.Since(0)
		// Vacuity guard one.
		if len(record) == 0 {
			t.Fatalf("%s: the record is empty, so there is nothing to replay and this case "+
				"measured nothing", one.what)
		}
		if want := len(states) - 1; len(record) != want {
			t.Errorf("%s: %d decisions were taken and the record holds %d, so an entry is "+
				"missing or doubled", one.what, want, len(record))
		}

		// The replay: a fresh draft from the same Config, handed nothing but the
		// entries.
		mirror, err := draft.New(config)
		if err != nil {
			t.Fatalf("%s: set the mirror up: %v", one.what, err)
		}
		if got := stateOf(mirror); got != states[0] {
			t.Fatalf("%s: a fresh draft from the same Config is\n  %s\nand the original began\n"+
				"  %s", one.what, got, states[0])
		}
		applied, halfArranged := 0, 0
		for at, entry := range record {
			if err := mirror.Apply(entry); err != nil {
				t.Fatalf("%s: replaying entry %d (%+v): %v", one.what, at, entry, err)
			}
			applied++
			if at+1 >= len(states) {
				continue
			}
			// ⚠️ The one relaxation, and it is exactly one entry wide: the first of
			// the two arrangements. See this test's own comment.
			if entry.Step == wire.StepArrange && mirror.Arranging() {
				halfArranged++
				continue
			}
			if got := stateOf(mirror); got != states[at+1] {
				t.Fatalf("%s: after replaying entry %d (%+v) the mirror is\n  %s\nand the "+
					"original was\n  %s", one.what, at, entry, got, states[at+1])
			}
		}
		// Vacuity guard two.
		if applied == 0 {
			t.Fatalf("%s: the replay applied no decisions, so a walk that ran nothing would "+
				"have passed every comparison above", one.what)
		}
		// The relaxation is bounded rather than open: at most the first of the two
		// arrangements may be skipped, and a case with no arrangement skips none.
		// Without this the `continue` above could grow to swallow a real
		// divergence and nothing would say so.
		if want := min(len(one.arrange), 1); halfArranged != want {
			t.Errorf("%s: %d comparisons were skipped for a half-open arrange phase and %d is "+
				"the whole of what may be", one.what, halfArranged, want)
		}
		// And the end compared by value, which is what the skip above defers to.
		if got, want := stateOf(mirror), states[len(states)-1]; got != want {
			t.Errorf("%s: the whole record replayed to\n  %s\nand the original ended\n  %s",
				one.what, got, want)
		}
		t.Logf("%s: %d decisions, %d entries, %d replayed, %d comparison deferred",
			one.what, len(states)-1, len(record), applied, halfArranged)
	}
}
