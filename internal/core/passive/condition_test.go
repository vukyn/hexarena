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
			"a share of nought",
			`[{"id":"odd","while":{"below_health":0},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			"a share past a thousand",
			`[{"id":"odd","while":{"below_health":1200},"applies":[{"status":"poison","chance":300}]}]`,
			"parts per thousand",
		},
		{
			"a gate on a trait that grants",
			`[{"id":"odd","while":{"below_health":500},"grants":[{"status":"toughened"}]}]`,
			"cannot be taken back",
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

// TestAGatedGrantIsRefusedRatherThanIgnored is the one refusal here that is
// about a missing mechanism rather than a bad number.
//
// A grant is applied once, when the unit is enlisted, and the status it puts on
// is permanent precisely so nothing can take it off. A condition on one would
// have to add and remove that status as health crossed the line — an engine door
// into a permanent status, an event for the trait coming and going, and a retune
// each time. Accepting the declaration would ship a trait whose gate was
// silently ignored, which is worse than not being able to write it.
func TestAGatedGrantIsRefusedRatherThanIgnored(t *testing.T) {
	_, err := parse(t,
		`[{"id":"overgrow","while":{"below_health":333},"grants":[{"status":"toughened"}]}]`)
	if err == nil {
		t.Fatal("a gated grant was accepted, so its gate would be ignored at runtime")
	}
	for _, want := range []string{"toughened", "applied once", "gate would be ignored"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal reads %q, want it to mention %q", err, want)
		}
	}
	// The two halves that *can* be gated are accepted alongside each other.
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
