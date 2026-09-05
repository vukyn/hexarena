package draft_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestApplyRoutesEveryStepThereIs is the totality guard on the mirror's switch,
// and it walks wire.DraftSteps rather than a table written here — the whole
// reason that function exists rather than a list each caller keeps.
//
// ⚠️ **What it is guarding against is a step Apply drops rather than refuses.**
// A sixth step added to the protocol would fall through the switch to its last
// line, and a mirror that answered an error there is doing the right thing — but
// only because the switch has a default arm at all. Deleting that arm makes the
// same case return nil, and a mirror carrying on as though a decision it did not
// understand had been applied is the silent desync every part of this package is
// arranged against. So this drives the draft to the state each step is due in,
// **applies an entry rather than calling the method**, and holds the set of
// steps it exercised against the declared list.
//
// ⚠️ **The skip is the sharp case and it is checked by its effect, not by its
// error.** A switch routing a characterless ban to Ban rather than SkipBan would
// hand Ban an empty id, which `offered` refuses — so an assertion on "no error"
// would catch that one by luck. What is asserted instead is that the pool did
// **not** shrink, which is what a skip *is*: bans are optional and a skip takes
// nought characters out.
func TestApplyRoutesEveryStepThereIs(t *testing.T) {
	all := shippedCast(t)
	config := draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	}
	mirror, err := draft.New(config)
	if err != nil {
		t.Fatalf("set the mirror up: %v", err)
	}
	routed := map[wire.DraftStep]int{}
	skipped := false
	for {
		seat, step, due := mirror.Turn()
		if !due {
			break
		}
		entry := wire.DraftEntry{Seat: seat, Step: step}
		switch step {
		case wire.StepBan:
			// The first ban of each pair is spent and the second skipped, so both
			// readings of one step are driven through Apply rather than only the
			// one that names somebody.
			if routed[wire.StepBan]%2 == 1 {
				before := len(mirror.Candidates())
				if err := mirror.Apply(entry); err != nil {
					t.Fatalf("%s skips a ban through Apply: %v", seat, err)
				}
				if after := len(mirror.Candidates()); after != before {
					t.Errorf("a skipped ban took the pool from %d candidates to %d, so Apply "+
						"routed a characterless ban to Ban rather than to SkipBan", before, after)
				}
				skipped = true
				routed[step]++
				continue
			}
			entry.Character = firstCandidate(t, mirror).ID
		case wire.StepPick:
			entry.Character = firstCandidate(t, mirror).ID
		case wire.StepLoadout:
			side := mirror.Picks()[seatIndex(t, seat)]
			open := side[len(side)-1]
			form, skills, passives := legalLoadout(t, characterNamed(t, all, open.Character))
			entry.Stage, entry.Skills, entry.Passives = form, skills, passives
		default:
			t.Fatalf("Turn asked for a %q, which this walk cannot build an entry for", step)
		}
		if err := mirror.Apply(entry); err != nil {
			t.Fatalf("%s takes a %s through Apply: %v", seat, step, err)
		}
		routed[step]++
	}
	if !mirror.Picked() {
		t.Fatalf("the walk applied %v and the picking is not over, so the states below are "+
			"not the ones this test is about", routed)
	}
	if !skipped {
		t.Error("no ban was skipped, so the one step with two readings was driven through " +
			"only one of them")
	}
	// The arrange phase, which Turn does not answer for, so it is driven from
	// outside the loop for the reason the replay proof drives it from outside its
	// own. Both sides, because the phase closes on the second.
	for _, seat := range mirror.AwaitingArrangement() {
		picks := mirror.Picks()[seatIndex(t, seat)]
		if err := mirror.Apply(wire.DraftEntry{
			Seat: seat, Step: wire.StepArrange, Slots: formationCells(len(picks)),
		}); err != nil {
			t.Fatalf("%s arranges through Apply: %v", seat, err)
		}
		routed[wire.StepArrange]++
	}
	if !mirror.Done() {
		t.Error("both sides arranged through Apply and the draft does not call itself done")
	}

	// The timeout cancels, so it needs a draft of its own — which is also the
	// only step whose entry is not a decision anybody made.
	expiring, err := draft.New(config)
	if err != nil {
		t.Fatalf("set the expiring draft up: %v", err)
	}
	seat, _, _ := expiring.Turn()
	if err := expiring.Apply(wire.DraftEntry{Seat: seat, Step: wire.StepTimeout}); err != nil {
		t.Fatalf("%s runs out of time through Apply: %v", seat, err)
	}
	if !expiring.Cancelled() {
		t.Error("a timeout applied through Apply did not cancel the draft, so that arm routes " +
			"somewhere else")
	}
	routed[wire.StepTimeout]++

	// The set, against the declared list rather than against a count written
	// here: a sixth step is a red test until this walk drives it.
	for _, step := range wire.DraftSteps() {
		if routed[step] == 0 {
			t.Errorf("no entry carrying %q was ever applied, so that arm of Apply is routed by "+
				"nothing in this suite", step)
		}
	}
	if got, want := len(routed), len(wire.DraftSteps()); got != want {
		t.Errorf("this walk applied %d of the %d steps the protocol declares: %v",
			got, want, routed)
	}
	t.Logf("%d steps routed: %v", len(routed), routed)
}

// TestApplyRefusesAStepNoDraftCanBeTold is the default arm, and it is the half a
// walk over the declared steps cannot reach: every step in wire.DraftSteps is
// routed, so the only way to reach the last line of Apply is a value the
// protocol does not declare — which is what a peer one version ahead sends.
//
// A dropped decision is the outcome to fear rather than the error. A mirror that
// answered nil here would carry on computing a draft that is no longer the
// room's, and the divergence would surface somewhere with no relation to its
// cause — the same argument wire.Kind.UnmarshalJSON refuses an unknown kind
// under.
func TestApplyRefusesAStepNoDraftCanBeTold(t *testing.T) {
	mirror, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(shippedCast(t)), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set the mirror up: %v", err)
	}
	before := stateOf(mirror)
	for _, step := range []wire.DraftStep{"", "surrender", "Ban", "trade"} {
		err := mirror.Apply(wire.DraftEntry{Seat: wire.SeatHost, Step: step})
		if err == nil {
			t.Errorf("a record holding the step %q was applied with no error at all", step)
			continue
		}
		// The refusal names the step, because a mirror that has diverged is a
		// thing somebody has to debug from one line of output.
		if !strings.Contains(err.Error(), string(step)) {
			t.Errorf("the refusal for the step %q does not name it: %v", step, err)
		}
	}
	if after := stateOf(mirror); after != before {
		t.Errorf("refusing an unknown step moved the draft\n  from %s\n    to %s", before, after)
	}
}

// TestAMirrorBuiltFromTheWireComputesTheSameDraft is the property step 3 exists
// for, taken through the **bytes** rather than through the values.
//
// ⚠️ **TestARecordReplaysIntoTheSameDraft is not this test and cannot stand in
// for it.** That one hands the mirror the record's own Go values, so it measures
// the state machine and says nothing about the format: a field dropped from
// wire.DraftDecision, a tag that does not survive a round trip, a skip whose
// absent character came back as something else — every one of those leaves it
// green. This one encodes the record into a wire.Drafted, decodes it on the
// other side, and replays **what came off the wire**, which is what a client
// actually holds.
//
// The comparison is by value at the end and the vacuity guards are the two the
// replay proof already found it needed: the record must not be empty and the
// replay must have applied something.
func TestAMirrorBuiltFromTheWireComputesTheSameDraft(t *testing.T) {
	all := shippedCast(t)
	config := draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatGuest,
	}
	played := draftedAndArranged(t, all, config.Format, config.First)
	record, _ := played.Since(0)
	if len(record) == 0 {
		t.Fatal("the record is empty, so there is nothing to put on the wire")
	}

	// One message, which is what a spectator joining at cursor nought is handed.
	raw, err := wire.Encode(&wire.Drafted{Decisions: record})
	if err != nil {
		t.Fatalf("encode the record: %v", err)
	}
	body, err := wire.Decode(raw)
	if err != nil {
		t.Fatalf("decode the record: %v", err)
	}
	batch, isBatch := body.(*wire.Drafted)
	if !isBatch {
		t.Fatalf("a drafted decoded as %T", body)
	}
	if got, want := len(batch.Decisions), len(record); got != want {
		t.Fatalf("%d decisions went onto the wire and %d came off", want, got)
	}

	mirror, err := draft.New(config)
	if err != nil {
		t.Fatalf("set the mirror up: %v", err)
	}
	for at, entry := range batch.Decisions {
		if err := mirror.Apply(entry); err != nil {
			t.Fatalf("replaying decision %d off the wire (%+v): %v", at, entry, err)
		}
	}
	if got, want := stateOf(mirror), stateOf(played); got != want {
		t.Errorf("the record replayed off the wire to\n  %s\nand the room holds\n  %s", got, want)
	}
	// And the squads, which is what the whole draft is for and the one output a
	// state string could agree about while the units disagreed.
	for index, squad := range mirror.Squads() {
		if err := squad.Validate(); err != nil {
			t.Errorf("the squad a wire-replayed draft fields for seat %d is not fieldable: %v",
				index, err)
		}
	}
	t.Logf("%d decisions, %d bytes on the wire, both squads fieldable", len(record), len(raw))
}
