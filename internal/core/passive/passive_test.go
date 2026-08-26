package passive_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/status"
)

func statuses(t *testing.T) *status.Book {
	t.Helper()
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "weaken", "category": "stat_debuff", "max_stacks": 3, "duration": 3,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": -300}]},
	    {"id": "stun", "category": "control", "max_stacks": 1, "duration": 1},
	    {"id": "block", "category": "shield", "max_stacks": 3, "duration": 2},
	    {"id": "regrowth", "category": "regen", "max_stacks": 3, "duration": 3, "tick_power": 400},
	    {"id": "haste", "category": "buff", "max_stacks": 2, "duration": 3,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 300}]},
	    {"id": "fleet", "category": "buff", "max_stacks": 1, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 500}]},
	    {"id": "toughened", "category": "buff", "max_stacks": 3, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 200}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	return book
}

func parse(t *testing.T, body string) (*passive.Book, error) {
	t.Helper()
	return passive.ParseBook([]byte(`{"passives":`+body+`}`), passive.Deps{Statuses: statuses(t)})
}

func TestParseBookAcceptsWhatItShould(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"swift","name":"nhanh nhẹn","grants":[{"status":"fleet"}]},
	  {"id":"hardy","grants":[{"status":"toughened","stacks":3},{"status":"fleet","stacks":1}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := book.IDs(); !reflect.DeepEqual(got, []string{"swift", "hardy"}) {
		t.Errorf("the book holds %v, want declaration order", got)
	}
	swift, err := book.Lookup("swift")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if swift.Name != "nhanh nhẹn" {
		t.Errorf("the name came back as %q", swift.Name)
	}
	// An unstated stack count is one, the way an unstated strike count is.
	if len(swift.Grants) != 1 || swift.Grants[0].Stacks != 1 {
		t.Errorf("an unstated stack count resolved to %+v, want one", swift.Grants)
	}
	if got := swift.StatusIDs(); !reflect.DeepEqual(got, []string{"fleet"}) {
		t.Errorf("StatusIDs is %v", got)
	}
	if _, err := book.Lookup("nobody-wrote-this"); err == nil {
		t.Error("an undeclared passive was looked up without complaint")
	}
	// An empty book is a book, unlike the status book: a game with no traits is
	// the state this shipped in, and refusing it would mean the file could not
	// exist before the first trait was authored.
	if _, err := parse(t, `[]`); err != nil {
		t.Errorf("an empty book was refused: %v", err)
	}
}

func TestParseBookRejects(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"no id", `[{"grants":[{"status":"fleet"}]}]`, "needs an id"},
		{"nothing granted", `[{"id":"idle","grants":[]}]`, "grants nothing"},
		{"no grants block at all", `[{"id":"idle"}]`, "grants nothing"},
		{"an unknown status", `[{"id":"odd","grants":[{"status":"glow"}]}]`, "unknown status"},
		{
			"a timed status",
			`[{"id":"quick","grants":[{"status":"haste"}]}]`,
			"which is timed",
		},
		{
			"more stacks than the status allows",
			`[{"id":"tough","grants":[{"status":"toughened","stacks":9}]}]`,
			"caps at 3",
		},
		{
			"the same status twice",
			`[{"id":"double","grants":[{"status":"fleet"},{"status":"fleet"}]}]`,
			"twice",
		},
		{
			"the same passive twice",
			`[{"id":"swift","grants":[{"status":"fleet"}]},{"id":"swift","grants":[{"status":"fleet"}]}]`,
			"declared twice",
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
	// The status book is not optional. A passive naming a status is exactly what
	// cannot be checked without one, so parsing anyway would accept a book whose
	// grants are all unverified.
	if _, err := passive.ParseBook([]byte(`{"passives":[]}`), passive.Deps{}); err == nil {
		t.Error("a passive book parsed with no status book to check against")
	}
}

// TestATimedStatusIsRefusedForTheReasonItIsRefused states the failure the rule
// prevents rather than only that the rule fires.
//
// A passive is granted once, when the unit is enlisted. A timed status put on
// that way counts down on the holder's own turns and nothing ever reapplies it,
// so the trait would be true for three turns and quietly false for the rest of
// the battle — and nothing on screen would say when it stopped.
func TestATimedStatusIsRefusedForTheReasonItIsRefused(t *testing.T) {
	_, err := parse(t, `[{"id":"quick","grants":[{"status":"haste"}]}]`)
	if err == nil {
		t.Fatal("a trait granting a timed status was accepted")
	}
	for _, want := range []string{"haste", "timed", "granted only once"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal reads %q, want it to mention %q", err, want)
		}
	}
}

func TestMarshalIsLosslessAndKeepsDeclarationOrder(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"hardy","grants":[{"status":"toughened","stacks":2}]},
	  {"id":"swift","name":"nhanh nhẹn","grants":[{"status":"fleet","stacks":1}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	first, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, err := passive.ParseBook(first, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v\n%s", err, first)
	}
	if !reflect.DeepEqual(reparsed.All(), book.All()) {
		t.Errorf("the trip through the file changed the book:\n%+v\n%+v", reparsed.All(), book.All())
	}
	again, err := reparsed.Marshal()
	if err != nil {
		t.Fatalf("marshal the round trip: %v", err)
	}
	if string(again) != string(first) {
		t.Errorf("a second write produced different bytes:\n%s\n%s", again, first)
	}
	// Declaration order, not sorted: the file is a design record read top to
	// bottom, and sorting would shuffle it to buy a diff appending already gives.
	if got := reparsed.IDs(); !reflect.DeepEqual(got, []string{"hardy", "swift"}) {
		t.Errorf("the round trip reordered the book to %v", got)
	}
	// A passive with no name writes no name, which is what keeps a book that
	// names none round-tripping to the bytes it was authored as.
	if strings.Contains(string(first), `"name": ""`) {
		t.Errorf("an absent name was written as an empty one:\n%s", first)
	}
}

// TestAllHandsOutACopy is the same guard every other book here carries: a caller
// editing what it was handed must not edit the book.
func TestAllHandsOutACopy(t *testing.T) {
	book, err := parse(t, `[{"id":"hardy","grants":[{"status":"toughened","stacks":2}]}]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	handed := book.All()
	handed[0].ID = "vandalised"
	handed[0].Grants[0].Stacks = 99
	again := book.All()
	if again[0].ID != "hardy" || again[0].Grants[0].Stacks != 2 {
		t.Errorf("editing the copy changed the book: %+v", again[0])
	}
}
