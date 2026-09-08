package passive_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/skill"
)

func TestAddedApplicationsParseAndReadBack(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"venomous","grants":[],"applies":[{"status":"poison","chance":300,"stacks":2}]},
	  {"id":"plain","grants":[],"applies":[{"status":"weaken","chance":1000}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	venomous, err := book.Lookup("venomous")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !reflect.DeepEqual(venomous.Applies,
		[]skill.Application{{Status: "poison", Chance: 300, Stacks: 2}}) {
		t.Errorf("the application came back as %+v", venomous.Applies)
	}
	// An unstated stack count is one, the way it is everywhere else here.
	plain, err := book.Lookup("plain")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if plain.Applies[0].Stacks != 1 {
		t.Errorf("an unstated stack count resolved to %d", plain.Applies[0].Stacks)
	}
	// Adding is enough on its own: a trait whose whole job is a rider does not
	// have to invent a stat change to be a legal entry.
	if len(plain.Grants) != 0 || len(plain.Resists) != 0 {
		t.Errorf("an applies-only trait came back with %+v / %+v", plain.Grants, plain.Resists)
	}
}

func TestAddedApplicationRejections(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			"an unknown status",
			`[{"id":"odd","applies":[{"status":"glow","chance":300}]}]`,
			"unknown status",
		},
		{
			"a chance of nought",
			`[{"id":"idle","applies":[{"status":"poison","chance":0}]}]`,
			"parts per thousand",
		},
		{
			"a chance past a thousand",
			`[{"id":"much","applies":[{"status":"poison","chance":1200}]}]`,
			"parts per thousand",
		},
		{
			"more stacks than the status allows",
			`[{"id":"deep","applies":[{"status":"poison","chance":300,"stacks":9}]}]`,
			"caps at",
		},
		{
			"the same status twice",
			`[{"id":"twice","applies":[
			  {"status":"poison","chance":300},{"status":"poison","chance":300}]}]`,
			"twice",
		},
		{
			"a permanent status, which is what a trait grants rather than inflicts",
			`[{"id":"odd","applies":[{"status":"toughened","chance":300}]}]`,
			"nothing could ever take it off",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantErr)
			}
		})
	}
}

// TestAConditionIsReadAtOrUnderItsThreshold pins the boundary, which is the only
// thing a one-term condition can get wrong.
//
// A share in parts per thousand is not a fraction: 333 of 3000 is 999, not 1000,
// so "a third" written as 333 is a hair under a third and the point at a third
// exactly is *above* the gate. That is worth knowing before authoring a threshold
// and is why this measures a share that divides cleanly.
func TestAConditionIsReadAtOrUnderItsThreshold(t *testing.T) {
	half := &passive.Condition{BelowHealth: 500}
	cases := []struct {
		health, maximum int64
		want            bool
	}{
		{1500, 3000, true},  // exactly half, and at counts
		{1501, 3000, false}, // a point above it
		{1499, 3000, true},
		{0, 3000, true},
		{3000, 3000, false},
		// A unit with no maximum is not a hurt unit, and the alternative is
		// dividing by nought.
		{0, 0, false},
	}
	for _, test := range cases {
		if got := half.Holds(test.health, test.maximum); got != test.want {
			t.Errorf("Holds(%d, %d) = %v, want %v", test.health, test.maximum, got, test.want)
		}
	}
	// And the share really is a share rather than a fraction: 333 is a hair under
	// a third, so a third exactly does not pass a gate written that way.
	if third := (&passive.Condition{BelowHealth: 333}); third.Holds(1000, 3000) {
		t.Error("333 per thousand admitted a health of exactly one third")
	}
	// No condition is always in force, so a caller never has to check for nil
	// before asking.
	var none *passive.Condition
	if !none.Holds(3000, 3000) || !none.Holds(0, 3000) {
		t.Error("a trait with no condition is not always in force")
	}
}

func TestConditionRejections(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			// ⚠️ **This row's expectation changed when the second end landed, and
			// it was changed deliberately rather than discovered.** A share of
			// nought used to be a share written wrong — "want a share in parts per
			// thousand" — because there was one term and nought was not a legal
			// value of it. With two ends, nought at one of them means "not this
			// end", so a clause with nought at both is asking about nothing rather
			// than asking badly, and the refusal has to say which.
			"a clause with nothing in it",
			`[{"id":"odd","while":{"below_health":0},"applies":[{"status":"poison","chance":300}]}]`,
			"no threshold in it",
		},
		{
			"an empty clause, written as an empty object",
			`[{"id":"odd","while":{},"applies":[{"status":"poison","chance":300}]}]`,
			"no threshold in it",
		},
		{
			"a share past a thousand",
			`[{"id":"odd","while":{"below_health":1200},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			"a negative share, which is still a share written wrong",
			`[{"id":"odd","while":{"below_health":-5},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			"a share past a thousand at the top of the bar",
			`[{"id":"odd","while":{"above_health":1200},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			"a negative share at the top of the bar",
			`[{"id":"odd","while":{"above_health":-5},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			// A band is two rules wearing one clause, and every screen words a
			// gate as one clause — so this is refused rather than resolved by
			// precedence, which would silently ignore the half an author had just
			// written.
			"both ends at once, which is a band",
			`[{"id":"odd","while":{"below_health":500,"above_health":200},
			  "applies":[{"status":"poison","chance":300}]}]`,
			"a band is two rules wearing one clause",
		},
		{
			// The same refusal with the two ends swapped, because a band the
			// wrong way round — in force above a half and below a fifth, which is
			// nowhere at all — must not be let through as "obviously empty, so
			// harmless".
			"both ends at once with no overlap between them",
			`[{"id":"odd","while":{"below_health":200,"above_health":500},
			  "applies":[{"status":"poison","chance":300}]}]`,
			"a band is two rules wearing one clause",
		},
		{
			"an unknown field inside the clause",
			`[{"id":"odd","while":{"beside_health":500},"applies":[{"status":"poison","chance":300}]}]`,
			"unknown field",
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
				t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantErr)
			}
		})
	}
	// The count, so a table that lost its rows to an editing accident cannot pass
	// by refusing nothing.
	if ran != len(cases) {
		t.Fatalf("ran %d rows, want %d", ran, len(cases))
	}
}

// TestAGateAtTheTopOfTheBarParsesAndReadsBack is the arrival of the second end.
//
// It reads the whole of what the term is: the threshold survives the parse, the
// gate reports which end it is at, Threshold answers the figure whichever end
// that is, and Holds is *at or over* rather than strictly over — so this gate and
// one written below the same number cover the bar between them with nobody
// falling through.
func TestAGateAtTheTopOfTheBarParsesAndReadsBack(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"unbowed","while":{"above_health":900},"applies":[{"status":"poison","chance":500}]},
	  {"id":"cornered","while":{"below_health":900},"applies":[{"status":"poison","chance":500}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fresh, err := book.Lookup("unbowed")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if fresh.While == nil {
		t.Fatal("the gate was dropped by the parse")
	}
	if fresh.While.AboveHealth != 900 {
		t.Errorf("the gate reads above %d, want 900", fresh.While.AboveHealth)
	}
	if fresh.While.BelowHealth != 0 {
		t.Errorf("a gate at the top of the bar also filled in below %d",
			fresh.While.BelowHealth)
	}
	if !fresh.While.AtTop() {
		t.Error("a gate written above a threshold does not report itself at the top of the bar")
	}
	if got := fresh.While.Threshold(); got != 900 {
		t.Errorf("Threshold answered %d, want 900", got)
	}
	// At or over, and the boundary is where the two ends meet: at 2700 of 3000
	// both this gate and the one written below 900 hold.
	hurt, err := book.Lookup("cornered")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	cases := []struct {
		health      int64
		fresh, hurt bool
	}{
		{3000, true, false},
		{2701, true, false},
		{2700, true, true}, // the shared point, and it is shared on purpose
		{2699, false, true},
		{0, false, true},
	}
	rows := 0
	for _, test := range cases {
		rows++
		if got := fresh.While.Holds(test.health, 3000); got != test.fresh {
			t.Errorf("at %d of 3000 the upper gate answered %v, want %v",
				test.health, got, test.fresh)
		}
		if got := hurt.While.Holds(test.health, 3000); got != test.hurt {
			t.Errorf("at %d of 3000 the lower gate answered %v, want %v",
				test.health, got, test.hurt)
		}
		if !test.fresh && !test.hurt {
			t.Errorf("at %d of 3000 neither gate holds, so a pair either side of "+
				"900 leaves a hole in the bar", test.health)
		}
	}
	if rows != len(cases) {
		t.Fatalf("walked %d rows, want %d", rows, len(cases))
	}
	// And the older end is unchanged by the arrival of the new one: it still
	// reports itself at the bottom of the bar and still answers its own figure.
	if hurt.While.AtTop() {
		t.Error("a gate written below a threshold reports itself at the top of the bar")
	}
	if got := hurt.While.Threshold(); got != 900 {
		t.Errorf("the lower gate's Threshold answered %d, want 900", got)
	}
}

// TestAGateAtTheTopOfTheBarSurvivesTheFile is the round-trip, and it is the case
// that catches a Marshal that forgot the new term.
//
// Parse, write, parse again: a writer that only knows below_health writes nothing
// at all for a gate at the top of the bar, and the book reloads as one that is
// always in force — silently, because dropping a field is not a parse error. The
// second half is the other way the write can go wrong: writing a nought at the
// end the gate is not at, which the parse now refuses outright as a clause with
// no threshold in it.
func TestAGateAtTheTopOfTheBarSurvivesTheFile(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"plain","grants":[{"status":"toughened"}]},
	  {"id":"unbowed","while":{"above_health":900},"applies":[{"status":"poison","chance":500}]},
	  {"id":"cornered","while":{"below_health":333},"applies":[{"status":"poison","chance":500}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	written := string(raw)
	for _, want := range []string{`"above_health": 900`, `"below_health": 333`} {
		if !strings.Contains(written, want) {
			t.Errorf("the rendering is missing %s:\n%s", want, written)
		}
	}
	// Neither end writes a nought at the other, which is what keeps the file
	// re-parsable at all and what keeps a book of gates at the bottom of the bar
	// — every gate shipped today — round-tripping to the bytes it was authored
	// as.
	for _, unwanted := range []string{`"above_health": 0`, `"below_health": 0`} {
		if strings.Contains(written, unwanted) {
			t.Errorf("the rendering wrote %s, which the parse refuses:\n%s", unwanted, written)
		}
	}
	if got := strings.Count(written, `"while"`); got != 2 {
		t.Errorf("the rendering wrote %d gates, want 2:\n%s", got, written)
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
	back, err := reparsed.Lookup("unbowed")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if back.While == nil {
		t.Fatal("the gate at the top of the bar did not survive the file")
	}
	if back.While.AboveHealth != 900 {
		t.Errorf("the gate came back as above %d, want 900", back.While.AboveHealth)
	}
}

// TestAGatedGrantIsCarriedThrough is the declaration that used to be refused.
//
// A grant behind a gate needed a mechanism rather than a term — an engine door
// into a permanent status, an event each way, a retune each time — so the parse
// layer refused it rather than accepting a gate it would then ignore. The
// mechanism is built, and what is checked here is that the two halves survive
// together as declared: refusing this was never about the numbers.
func TestAGatedGrantIsCarriedThrough(t *testing.T) {
	book, err := parse(t,
		`[{"id":"overgrow","while":{"below_health":333},"grants":[{"status":"toughened","stacks":1}]}]`)
	if err != nil {
		t.Fatalf("a gated grant was refused: %v", err)
	}
	held, err := book.Lookup("overgrow")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if held.While == nil {
		t.Fatal("the gate was dropped, which is the failure the refusal existed to prevent")
	}
	if held.While.BelowHealth != 333 {
		t.Errorf("the gate is at %d, want 333", held.While.BelowHealth)
	}
	if len(held.Grants) != 1 || held.Grants[0].Status != "toughened" {
		t.Errorf("the grant reads %+v, want one of toughened", held.Grants)
	}
	// A gated grant is still a grant, so the status it names still has to be one
	// nothing in the game can dispel. A timed one would wear off on the holder's
	// own turns with nothing to put it back, and the gate does not change that.
	if _, err := parse(t,
		`[{"id":"odd","while":{"below_health":333},"grants":[{"status":"poison"}]}]`); err == nil {
		t.Error("a gated grant of a timed status was accepted")
	}
	// The two halves that were always gateable are still accepted alongside
	// each other, with or without a grant beside them.
	if _, err := parse(t, `[{"id":"cornered","while":{"below_health":333},
	  "applies":[{"status":"poison","chance":500}],
	  "resists":[{"status":"weaken","amount":700}]}]`); err != nil {
		t.Errorf("a gated trait with no grant was refused: %v", err)
	}
}

func TestTheGateAndTheRidersSurviveTheFile(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"plain","grants":[{"status":"toughened"}]},
	  {"id":"cornered","grants":[],"while":{"below_health":333},
	   "applies":[{"status":"poison","chance":500,"stacks":2}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"below_health": 333`, `"chance": 500`, `"stacks": 2`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the rendering is missing %s:\n%s", want, raw)
		}
	}
	// A trait with neither writes neither block, so a book from before these
	// existed round-trips to the bytes it was authored as.
	if strings.Count(string(raw), `"while"`) != 1 || strings.Count(string(raw), `"applies"`) != 1 {
		t.Errorf("an ungated trait with no riders still wrote the blocks:\n%s", raw)
	}
	reparsed, err := passive.ParseBook(raw, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v\n%s", err, raw)
	}
	if !reflect.DeepEqual(reparsed.All(), book.All()) {
		t.Errorf("the trip through the file changed the book:\n%+v\n%+v",
			reparsed.All(), book.All())
	}
	// All hands out a copy of the condition too. It is a pointer, so a caller
	// editing what it was handed would otherwise edit the book through it.
	handed := book.All()
	for i := range handed {
		if handed[i].While != nil {
			handed[i].While.BelowHealth = 1
		}
	}
	for _, held := range book.All() {
		if held.While != nil && held.While.BelowHealth == 1 {
			t.Error("editing the copy changed the book's condition")
		}
	}
}
