package wire

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestTheDraftDecisionIsDeclaredOnce is the shape decision as a measurement
// rather than as a paragraph.
//
// ⚠️ **The two directions need two bodies and only the seat differs**, so the
// live risk is somebody flattening one of them — writing the six decision fields
// out again beside a Seat — which compiles, round-trips, keeps the golden
// byte-identical and leaves two structs nothing holds in step. That is exactly
// the failure the repository's own note on the skill parse shape claims a compile
// error prevents, and (measured while this was written) it does **not**:
// skill.Skill.file() is a *keyed* literal, so a field added to either side of
// that pair is not a compile error anywhere. Here there is one struct instead,
// and this is what says so.
//
// It also asserts the embeds are **anonymous**, because that is what keeps both
// bodies serialising flat — a named field would nest them and move every line of
// the golden's two draft entries.
func TestTheDraftDecisionIsDeclaredOnce(t *testing.T) {
	decision := reflect.TypeFor[DraftDecision]()
	for _, one := range []struct {
		what  string
		outer reflect.Type
		at    int
	}{
		{"Decide", reflect.TypeFor[Decide](), 0},
		{"DraftEntry", reflect.TypeFor[DraftEntry](), 1},
	} {
		field := one.outer.Field(one.at)
		if field.Type != decision {
			t.Errorf("%s's field %d is a %s and not the one DraftDecision, so the decision is "+
				"declared twice and nothing holds the two in step", one.what, one.at, field.Type)
			continue
		}
		if !field.Anonymous {
			t.Errorf("%s embeds DraftDecision as the named field %q, so its body nests instead "+
				"of reading flat", one.what, field.Name)
		}
	}
	// The seat is the whole of the difference, stated by value: one field on the
	// record and none on the client's message.
	if got, want := reflect.TypeFor[Decide]().NumField(), 1; got != want {
		t.Errorf("Decide has %d fields, want the %d it is defined as: the decision and nothing "+
			"else", got, want)
	}
	if got, want := reflect.TypeFor[DraftEntry]().NumField(), 2; got != want {
		t.Errorf("DraftEntry has %d fields, want the %d it is defined as: the seat and the "+
			"decision", got, want)
	}
}

// TestADraftDecisionCarriesNoSeatAndNoDigest is the Act-shaped guard, and it is
// written against the type rather than a value because the mistake it catches is
// a field somebody adds in good faith.
//
// Two claims, and each has its own reason written on the code:
//
//   - **No seat on the client's message.** The room knows which connection
//     spoke, so a seat here would be a second statement of a fact one side owns
//     and the only thing it could add is a disagreement to resolve. → Decide.
//   - **No digest anywhere in the draft.** A reader who knows Turn will look for
//     one; a draft's state is a pure function of the decisions and the pool, and
//     the pool is already gated by the data digest at the join, so a per-decision
//     digest could catch nothing CodeDataMismatch does not already refuse. →
//     Drafted. This half is checked **by type** rather than by field name, since
//     the name is the easy half to get past.
func TestADraftDecisionCarriesNoSeatAndNoDigest(t *testing.T) {
	banned := map[string]string{
		"seat":  "the room knows which connection spoke, exactly as it knows whose turn it is",
		"side":  "a side is a fact about a battle and a draft has none yet",
		"turn":  "a draft counts decisions and the room is what counts them",
		"pool":  "the pool is computed from the decisions, and it is the mirror that computes it",
		"state": "a snapshot is the one place two peers could come to disagree",
	}
	structure := reflect.TypeFor[Decide]().Field(0).Type
	for index := range structure.NumField() {
		field := structure.Field(index)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if because, refused := banned[strings.ToLower(tag)]; refused {
			t.Errorf("a draft decision carries %s (%s): %s", field.Name, tag, because)
		}
	}
	// Non-vacuity: every check above passes on a decision with no fields at all,
	// and a decision has six things to say.
	if got, want := structure.NumField(), 6; got != want {
		t.Errorf("a draft decision has %d fields, want the %d it is defined as: a step, a "+
			"character, a form, two lists and an arrangement", got, want)
	}
	digest := reflect.TypeFor[EventDigest]()
	for _, one := range []struct {
		what string
		is   reflect.Type
	}{
		{"DraftDecision", structure},
		{"DraftEntry", reflect.TypeFor[DraftEntry]()},
		{"Drafted", reflect.TypeFor[Drafted]()},
	} {
		for index := range one.is.NumField() {
			if one.is.Field(index).Type == digest {
				t.Errorf("%s carries an EventDigest (%s): a draft's state is a pure function of "+
					"the decisions and the pool, and the pool is gated by the data digest at the "+
					"join", one.what, one.is.Field(index).Name)
			}
		}
	}
}

// TestEveryDraftStepTravels is the vocabulary's own walk, and it walks
// DraftSteps rather than a table written here — the shape KindCount and CodeCount
// take for the two enums that have a declaration order.
//
// A step is a named string, so there is nothing for an insertion to reinterpret
// and no names table to fall out of step (→ DraftStep, and Seat, which is a
// string for the same reason). What is still worth holding is that every step
// **survives the trip in both directions**: the client's message and the room's
// record carry the same declaration, so a step that travelled on one and not the
// other would mean the shape had been split.
func TestEveryDraftStepTravels(t *testing.T) {
	seen := map[DraftStep]bool{}
	for _, step := range DraftSteps() {
		if !step.Valid() {
			t.Errorf("%q is in DraftSteps and DraftStep.Valid refuses it", step)
		}
		if seen[step] {
			t.Errorf("%q is in DraftSteps twice, so one of the two is unreachable", step)
		}
		seen[step] = true
		name := string(step)
		if name == "" || strings.ContainsAny(name, " ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("the step %q is not lower snake case, which is what every name on this "+
				"wire is", name)
		}
		// ⚠️ Each reader is a **checked** assertion answering an error rather than
		// a bare one, because a body that came back as the wrong type is a real
		// failure this test has to be able to report: a body claiming another
		// kind's name decodes to that other kind's struct, and a panic here would
		// take every other test in the package down with it instead of naming the
		// one thing that moved.
		for _, one := range []struct {
			what string
			body Body
			back func(Body) (DraftStep, error)
		}{
			{"decide", &Decide{DraftDecision: DraftDecision{Step: step}},
				func(body Body) (DraftStep, error) {
					decided, ok := body.(*Decide)
					if !ok {
						return "", fmt.Errorf("it decoded as a %T", body)
					}
					return decided.Step, nil
				}},
			{"drafted", &Drafted{Decisions: []DraftEntry{
				{Seat: SeatHost, DraftDecision: DraftDecision{Step: step}},
			}}, func(body Body) (DraftStep, error) {
				batch, ok := body.(*Drafted)
				if !ok {
					return "", fmt.Errorf("it decoded as a %T", body)
				}
				if len(batch.Decisions) != 1 {
					return "", fmt.Errorf("it came back holding %d decisions", len(batch.Decisions))
				}
				return batch.Decisions[0].Step, nil
			}},
		} {
			raw, err := Encode(one.body)
			if err != nil {
				t.Errorf("encode a %s carrying %q: %v", one.what, step, err)
				continue
			}
			if !strings.Contains(string(raw), `"step":"`+name+`"`) {
				t.Errorf("a %s carrying %q does not name it: %s", one.what, step, raw)
			}
			decoded, err := Decode(raw)
			if err != nil {
				t.Errorf("decode a %s carrying %q: %v", one.what, step, err)
				continue
			}
			got, err := one.back(decoded)
			if err != nil {
				t.Errorf("a %s carrying %q did not come back as one: %v", one.what, step, err)
				continue
			}
			if got != step {
				t.Errorf("a %s carrying %q came back carrying %q", one.what, step, got)
			}
		}
	}
	if len(seen) != len(DraftSteps()) {
		t.Errorf("%d distinct steps over %d entries", len(seen), len(DraftSteps()))
	}
	// The zero step is not a step, which is the trap the whole enum-by-name rule
	// is about: an absent step must not quietly mean the first one.
	if DraftStep("").Valid() {
		t.Error("the zero DraftStep reports itself valid, so a decision with no step would " +
			"read as a ban")
	}
	if DraftStep("surrender").Valid() {
		t.Error("a step this protocol does not declare reports itself valid")
	}
	t.Logf("%d steps, all named and all travelling in both directions", len(DraftSteps()))
}

// TestADraftedBatchKeepsItsOrderAndItsAbsences pins the two things the batch
// exists for, on the fixture the golden records.
//
// **Order**, because the record is a sequence and a mirror replays it in order:
// the arrange phase records two entries in seats order and a batch that arrived
// shuffled would put the guest's arrangement on the host's picks.
//
// **Absence**, because a ban that names nobody is the skip and `omitempty` is
// what makes that absence the decision. A skip whose character came back as
// something other than empty would be a ban on whatever that was.
func TestADraftedBatchKeepsItsOrderAndItsAbsences(t *testing.T) {
	fixture, ok := messageFixtures(t)[KindDrafted].(*Drafted)
	if !ok {
		t.Fatal("the drafted fixture is not a *Drafted")
	}
	raw, err := Encode(fixture)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	batch, ok := decoded.(*Drafted)
	if !ok {
		t.Fatalf("a drafted decoded as %T", decoded)
	}
	if !reflect.DeepEqual(batch, fixture) {
		t.Fatalf("the batch did not survive the trip\n got %#v\nwant %#v", batch, fixture)
	}
	// Non-vacuity, and it is the whole point of the fixture being what it is: a
	// batch of one satisfies every claim below, and so does one with no skip and
	// no arrangement in it.
	if len(batch.Decisions) < 4 {
		t.Fatalf("the fixture batch holds %d decisions; a batch this test can measure order "+
			"and absence on needs several", len(batch.Decisions))
	}
	steps := make([]DraftStep, 0, len(batch.Decisions))
	for _, one := range batch.Decisions {
		steps = append(steps, one.Step)
	}
	if !slices.Contains(steps, StepArrange) {
		t.Error("the fixture batch holds no arrangement, which is the case a single-decision " +
			"message could not express and therefore the reason this is a batch")
	}
	skipped := 0
	for at, one := range batch.Decisions {
		if one.Step != StepBan || one.Character != "" {
			continue
		}
		skipped++
		// The absence read back off the bytes as well as off the value, because
		// omitempty is what this claim rests on and a value comparison cannot see
		// an empty string that was written out.
		var envelope Envelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("re-read the envelope: %v", err)
		}
		var shape struct {
			Decisions []map[string]json.RawMessage `json:"decisions"`
		}
		if err := json.Unmarshal(envelope.Body, &shape); err != nil {
			t.Fatalf("re-read the batch: %v", err)
		}
		if _, written := shape.Decisions[at]["character"]; written {
			t.Errorf("decision %d is a skipped ban and the bytes still carry a character "+
				"field, so the absence that IS the decision is not what travels", at)
		}
	}
	if skipped != 1 {
		t.Errorf("the fixture batch holds %d skipped bans and the record needs exactly one, "+
			"beside a spent ban, or the absence is measured against nothing", skipped)
	}
}

// TestAnEmptyLoadoutListTravelsAsAnAbsentOne is the omitempty claim
// DraftDecision.Skills makes, measured rather than assumed.
//
// It matters because the two are one decision at the far end —
// cast.ChooseLoadout reads neither an absent list nor an empty one as a request
// for anything — so a peer that sent `[]` must not be able to tell the
// difference on the way back. What would be a defect is the opposite: an empty
// list surviving as a *distinct* value, since then two encodings of one decision
// would exist and a record could hold either.
func TestAnEmptyLoadoutListTravelsAsAnAbsentOne(t *testing.T) {
	raw, err := Encode(&Decide{DraftDecision: DraftDecision{
		Step:     StepLoadout,
		Skills:   []string{},
		Passives: []string{},
	}})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, field := range []string{"skills", "passives"} {
		if strings.Contains(string(raw), `"`+field+`"`) {
			t.Errorf("an empty %s list was written out: %s", field, raw)
		}
	}
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	decision, ok := decoded.(*Decide)
	if !ok {
		t.Fatalf("a decide decoded as %T", decoded)
	}
	if decision.Skills != nil || decision.Passives != nil {
		t.Errorf("an empty loadout list came back as %#v, so one decision has two encodings",
			decision.DraftDecision)
	}
	// And a list with something in it is untouched, or the claim above would hold
	// for a decoder that dropped every list.
	full := &Decide{DraftDecision: DraftDecision{
		Step:     StepLoadout,
		Skills:   []string{"fixture.strike"},
		Passives: []string{"fixture.trait"},
	}}
	raw, err = Encode(full)
	if err != nil {
		t.Fatalf("encode a full loadout: %v", err)
	}
	decoded, err = Decode(raw)
	if err != nil {
		t.Fatalf("decode a full loadout: %v", err)
	}
	if !reflect.DeepEqual(decoded, full) {
		t.Errorf("a loadout that names something did not survive the trip: %#v", decoded)
	}
}

// TestAnArrangementCarriesTheBackCornerAsACell is the one coordinate in this
// protocol whose zero value is a real answer, and it is measured here for the
// reason wire.Act documents for its own Aim: hex.Offset{0,0} is the ally back
// corner, so a slot dropped by an omitempty would arrive as a cell somebody
// might genuinely have chosen rather than as an absence.
//
// The slice is what carries the absence — an arrangement is a side's *whole*
// arrangement, so no arrangement is no slice — and that is why the omitempty sits
// on the list and never on a cell.
func TestAnArrangementCarriesTheBackCornerAsACell(t *testing.T) {
	corner := &Decide{DraftDecision: DraftDecision{
		Step:  StepArrange,
		Slots: []hex.Offset{{Col: 0, Row: 0}, {Col: 1, Row: 0}},
	}}
	raw, err := Encode(corner)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(decoded, corner) {
		t.Fatalf("an arrangement holding the back corner came back as %#v", decoded)
	}
	// A side that has not arranged has no slice at all, which is the absence the
	// list expresses and the reason a cell needs none of its own.
	bare, err := Encode(&Decide{DraftDecision: DraftDecision{Step: StepBan}})
	if err != nil {
		t.Fatalf("encode a ban: %v", err)
	}
	if strings.Contains(string(bare), "slots") {
		t.Errorf("a ban was written with an arrangement on it: %s", bare)
	}
}
