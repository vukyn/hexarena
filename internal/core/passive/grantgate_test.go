package passive_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
)

// The two-tier declaration every case here reads: an ungated grant carrying the
// first tier and a gated one carrying the difference, on one trait.
//
// ⚠️ **The trait has no `while` of its own, and that is the discriminating part
// rather than a detail.** A fixture that also carried a trait-level gate would
// pass with the per-grant field ignored entirely, because every grant on such a
// trait is gated either way. Only a trait gated *nowhere but on one grant* can
// tell a per-grant gate from nothing at all.
const twoTier = `[{"id":"bulwark","grants":[
	  {"status":"toughened","stacks":1},
	  {"status":"fortified","stacks":1,"while":{"above_health":700}}
	]}]`

// TestAGrantMayCarryAGateOfItsOwn is the shape arriving: the field survives the
// parse, and the effective gate over each grant is the one the design says.
//
// It reads GateOver rather than the fields, because GateOver is the answer every
// refusal, every renderer and the battle itself take — a test asserting on the
// fields would pass with that function answering the wrong gate.
func TestAGrantMayCarryAGateOfItsOwn(t *testing.T) {
	book, err := parse(t, twoTier)
	if err != nil {
		t.Fatalf("a two-tier trait was refused: %v", err)
	}
	held, err := book.Lookup("bulwark")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if held.While != nil {
		t.Fatal("the trait picked up a gate of its own, so this measures the whole-trait case")
	}
	if len(held.Grants) != 2 {
		t.Fatalf("the trait came back with %d grants, want two", len(held.Grants))
	}
	always, gated := held.Grants[0], held.Grants[1]
	if always.While != nil {
		t.Errorf("the ungated tier came back gated: %+v", always.While)
	}
	if gated.While == nil {
		t.Fatal("the gate on the second tier was dropped by the parse")
	}
	if gated.While.AboveHealth != 700 || gated.While.BelowHealth != 0 {
		t.Errorf("the grant's gate reads above %d / below %d, want above 700 and nothing below",
			gated.While.AboveHealth, gated.While.BelowHealth)
	}
	if !gated.While.AtTop() || gated.While.Threshold() != 700 {
		t.Errorf("the grant's gate reports top=%v threshold=%d, want true and 700",
			gated.While.AtTop(), gated.While.Threshold())
	}
	// The effective gate, which is the whole of what the rest of the engine asks.
	if got := held.GateOver(always); got != nil {
		t.Errorf("the ungated tier is behind %+v, want no gate at all", got)
	}
	if got := held.GateOver(gated); got == nil || got.AboveHealth != 700 {
		t.Errorf("the gated tier is behind %+v, want the grant's own gate above 700", got)
	}
	if !held.Gated() {
		t.Error("a trait with a gated grant reports that it gates nothing, which is the " +
			"answer that leaves the gate stuck at whatever enlistment put it at")
	}
}

// TestTheEffectiveGateOverAGrantFallsBackToTheTraits is the arm a naive move
// loses, and it is the reason GateOver exists rather than a field read.
//
// A grant with no clause of its own on a **trait-level** gated trait is a gated
// grant: the two refusals below it exist for exactly that case, and every
// renderer has to word it. An expression reading Grant.While alone answers "no
// gate" here — which compiles, parses, and turns both refusals off.
func TestTheEffectiveGateOverAGrantFallsBackToTheTraits(t *testing.T) {
	book, err := parse(t,
		`[{"id":"dug_in","while":{"below_health":333},"grants":[{"status":"toughened"}]}]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	held, err := book.Lookup("dug_in")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	grant := held.Grants[0]
	if grant.While != nil {
		t.Fatal("the grant carries a clause of its own, so the fallback is not what is measured")
	}
	gate := held.GateOver(grant)
	if gate == nil {
		t.Fatal("a grant under a trait-level gate reports no gate, which is the reading " +
			"that lets a health term and an absorbing pool through")
	}
	if gate.BelowHealth != 333 {
		t.Errorf("the effective gate reads below %d, want the trait's own 333", gate.BelowHealth)
	}
	if !held.Gated() {
		t.Error("a trait-level gated trait reports that it gates nothing")
	}
	// And an ungated trait is still ungated at both readings, or the fallback is
	// answering a gate where there is none.
	plain, err := parse(t, `[{"id":"hardy","grants":[{"status":"toughened"}]}]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bare, err := plain.Lookup("hardy")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if bare.GateOver(bare.Grants[0]) != nil || bare.Gated() {
		t.Error("an ungated trait's grant reports a gate")
	}
}

// TestAGrantThatRaisesHealthIsRefusedBehindEitherGate is the health rule, now
// that a gate has two places to hang.
//
// ⚠️ **Both arms, and the trait-level one is the arm a naive move loses.** The
// refusal used to read the trait's own field; if it moved to the grant's, a
// health-raising grant under a trait-level gate would be accepted and the whole
// rule would go quiet on the case it was written for — a holder healed into the
// room a gate opened, left standing above its own maximum when the gate shut.
//
// The reason is the same at either end and in either place: the gate closing
// calls Release, so the raise comes back off.
func TestAGrantThatRaisesHealthIsRefusedBehindEitherGate(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"gated by the trait",
			`[{"id":"odd","while":{"below_health":333},"grants":[{"status":"swollen"}]}]`,
		},
		{
			"gated by the grant itself",
			`[{"id":"odd","grants":[{"status":"swollen","while":{"below_health":333}}]}]`,
		},
		{
			"gated by the grant at the top of the bar",
			`[{"id":"odd","grants":[{"status":"swollen","while":{"above_health":900}}]}]`,
		},
		{
			// The health-raising grant is the *second* tier here, beside an
			// ungated one that is perfectly legal — so a refusal that only ever
			// looked at the first grant would let this through.
			"gated as the second tier of a two-tier trait",
			`[{"id":"odd","grants":[
			  {"status":"toughened"},
			  {"status":"swollen","while":{"above_health":900}}]}]`,
		},
	}
	ran := 0
	for _, test := range cases {
		ran++
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), "raises health") {
				t.Errorf("%s was refused with %q, want it to mention the health it raises",
					test.name, err)
			}
		})
	}
	if ran != len(cases) {
		t.Fatalf("ran %d rows, want %d", ran, len(cases))
	}
	// The control, or every row above passes with the term banned outright. An
	// ungated grant raises the maximum once, at enlistment, and never moves it
	// again — which is what a composition bonus does and what keeps this a rule
	// about the gate rather than about the term.
	if _, err := parse(t, `[{"id":"broad","grants":[
	  {"status":"toughened"},{"status":"swollen","stacks":2}]}]`); err != nil {
		t.Errorf("an ungated health grant beside another ungated grant was refused: %v", err)
	}
}

// TestAnAbsorbingPoolIsRefusedBehindEitherGate is the other refusal that came
// down with the gate, and the reason is the same at both ends.
//
// Hold runs the grant again every time a gate reopens, so a pool behind one
// comes back full each time its holder crosses the line — a wall with no cost,
// refilled by being hit. ⚠️ Both arms again, because a version reading the
// grant's own field would accept every pool under a trait-level gate.
func TestAnAbsorbingPoolIsRefusedBehindEitherGate(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"gated by the trait",
			`[{"id":"odd","while":{"below_health":333},
			  "grants":[{"status":"bastion","power":500,"scaling":"defense"}]}]`,
		},
		{
			"gated by the grant itself",
			`[{"id":"odd","grants":[
			  {"status":"bastion","power":500,"scaling":"defense","while":{"below_health":333}}]}]`,
		},
		{
			"gated by the grant at the top of the bar",
			`[{"id":"odd","grants":[
			  {"status":"bastion","power":500,"scaling":"defense","while":{"above_health":900}}]}]`,
		},
		{
			"gated as the second tier of a two-tier trait",
			`[{"id":"odd","grants":[
			  {"status":"toughened"},
			  {"status":"bastion","power":500,"scaling":"defense","while":{"above_health":900}}]}]`,
		},
	}
	ran := 0
	for _, test := range cases {
		ran++
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), "a pool is refilled") {
				t.Errorf("%s was refused with %q, want it to mention the refill", test.name, err)
			}
		})
	}
	if ran != len(cases) {
		t.Fatalf("ran %d rows, want %d", ran, len(cases))
	}
	// The control. An ungated pool runs once at enlistment, which is exactly
	// "puts a barrier up when the battle starts" — and without this row every
	// case above passes with a guard nobody may grant at all.
	if _, err := parse(t, `[{"id":"walled","grants":[
	  {"status":"toughened"},
	  {"status":"bastion","power":500,"scaling":"defense"}]}]`); err != nil {
		t.Errorf("an ungated pool beside an ungated grant was refused: %v", err)
	}
}

// TestAGrantsGateIsReadUnderTheSameRulesAsATraits is the rule sharing, said as a
// table rather than trusted to the fact that one function is called twice.
//
// A gate is one idea, so a grant's clause obeys a trait's rules: exactly one
// end, a share in parts per thousand, never a clause with nothing in it, and a
// field the book does not know is refused rather than dropped. Every row expects
// the *same* message fragment the trait-level table expects, which is the
// assertion that catches a second reading written beside the first — two
// expressions for one rule disagree eventually, and the refusal wording is where
// it shows first.
func TestAGrantsGateIsReadUnderTheSameRulesAsATraits(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
		// Whether the refusal is expected to name the grant. A strict-decode
		// refusal fires before resolve has looked at anything, so it names the
		// field and nothing else — the same answer an unknown field inside a
		// *trait's* clause gets, which is why it is a row here rather than an
		// omission.
		namesGrant bool
	}{
		{
			"a band on a grant",
			`[{"id":"odd","grants":[
			  {"status":"toughened","while":{"below_health":500,"above_health":200}}]}]`,
			"a band is two rules wearing one clause", true,
		},
		{
			"an empty clause on a grant",
			`[{"id":"odd","grants":[{"status":"toughened","while":{}}]}]`,
			"no threshold in it", true,
		},
		{
			"a nought at one end, which is the same empty clause written out",
			`[{"id":"odd","grants":[{"status":"toughened","while":{"below_health":0}}]}]`,
			"no threshold in it", true,
		},
		{
			"a share past a thousand on a grant",
			`[{"id":"odd","grants":[{"status":"toughened","while":{"below_health":1200}}]}]`,
			"parts per thousand", true,
		},
		{
			"a negative share on a grant",
			`[{"id":"odd","grants":[{"status":"toughened","while":{"above_health":-5}}]}]`,
			"parts per thousand", true,
		},
		{
			"an unknown field inside a grant's clause",
			`[{"id":"odd","grants":[{"status":"toughened","while":{"beside_health":500}}]}]`,
			"unknown field", false,
		},
		{
			// One gate over a grant, never two. A trait-level clause and a
			// grant-level one would be a conjunction, which is a band wearing two
			// clauses instead of one — and every screen words a gate as a single
			// sentence.
			"a gate on a grant of a trait that is already gated",
			`[{"id":"odd","while":{"below_health":333},
			  "grants":[{"status":"toughened","while":{"above_health":900}}]}]`,
			"behind a gate of its own on a trait that is already gated", true,
		},
	}
	ran := 0
	for _, test := range cases {
		ran++
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("%s was refused with %q, want it to mention %q",
					test.name, err, test.wantErr)
			}
			// The refusal names the grant, not just the trait: a trait may carry
			// several and an author reading "passive odd: is gated…" cannot tell
			// which clause was wrong.
			if !test.namesGrant {
				return
			}
			if !strings.Contains(err.Error(), `grants "toughened"`) {
				t.Errorf("%s was refused with %q, which never names the grant it is about",
					test.name, err)
			}
		})
	}
	if ran != len(cases) {
		t.Fatalf("ran %d rows, want %d", ran, len(cases))
	}
}

// TestAGrantsGateSurvivesTheFile is the round-trip, and it is the case that
// catches a Marshal that forgot the new field.
//
// A writer that does not know about it writes nothing, and the book reloads as a
// two-tier trait whose second tier is always on — silently, because dropping a
// field is not a parse error. The other half is the way a write can go wrong at
// the far end: a nought written at the end the gate is not at, which the parse
// refuses outright as a clause with no threshold in it.
func TestAGrantsGateSurvivesTheFile(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"plain","grants":[{"status":"toughened"}]},
	  {"id":"bulwark","grants":[
	    {"status":"toughened","stacks":1},
	    {"status":"fortified","stacks":1,"while":{"above_health":700}}]},
	  {"id":"lastditch","grants":[
	    {"status":"fortified","stacks":1,"while":{"below_health":250}}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	written := string(raw)
	for _, want := range []string{`"above_health": 700`, `"below_health": 250`} {
		if !strings.Contains(written, want) {
			t.Errorf("the rendering is missing %s:\n%s", want, written)
		}
	}
	// Neither end writes a nought at the other, which is what keeps the file
	// re-parsable at all.
	for _, unwanted := range []string{`"above_health": 0`, `"below_health": 0`} {
		if strings.Contains(written, unwanted) {
			t.Errorf("the rendering wrote %s, which the parse refuses:\n%s", unwanted, written)
		}
	}
	// Two gates written, one per gated grant — and none on the ungated grants,
	// so a book of ungated grants round-trips to the bytes it was authored as.
	if got := strings.Count(written, `"while"`); got != 2 {
		t.Errorf("the rendering wrote %d gates, want the 2 gated grants:\n%s", got, written)
	}
	reparsed, err := passive.ParseBook(raw, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v\n%s", err, written)
	}
	if !reflect.DeepEqual(reparsed.All(), book.All()) {
		t.Errorf("the trip through the file changed the book:\n%+v\n%+v",
			reparsed.All(), book.All())
	}
	// Named rather than left to DeepEqual, because the failure this exists to
	// catch is a *dropped* gate and two books that both lost it compare equal.
	back, err := reparsed.Lookup("bulwark")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(back.Grants) != 2 {
		t.Fatalf("the trait came back with %d grants", len(back.Grants))
	}
	if back.Grants[0].While != nil {
		t.Errorf("the ungated tier came back gated: %+v", back.Grants[0].While)
	}
	if back.Grants[1].While == nil {
		t.Fatal("the gate on the second tier did not survive the file")
	}
	if back.Grants[1].While.AboveHealth != 700 {
		t.Errorf("the gate came back as above %d, want 700", back.Grants[1].While.AboveHealth)
	}
	if back.While != nil {
		t.Errorf("the write put a gate on the trait itself: %+v", back.While)
	}
	// All hands out a copy of a grant's gate too. It is a pointer inside a
	// slice, so cloning the slice copies the pointer and not what it points at —
	// a caller editing what it was handed would edit the book through it.
	handed := book.All()
	for i := range handed {
		for j := range handed[i].Grants {
			if handed[i].Grants[j].While != nil {
				handed[i].Grants[j].While.AboveHealth = 1
				handed[i].Grants[j].While.BelowHealth = 1
			}
		}
	}
	for _, held := range book.All() {
		for _, grant := range held.Grants {
			if grant.While == nil {
				continue
			}
			if grant.While.AboveHealth == 1 || grant.While.BelowHealth == 1 {
				t.Errorf("editing the copy changed %q's own gate on %q", held.ID, grant.Status)
			}
		}
	}
}
