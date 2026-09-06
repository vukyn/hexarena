package status_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
)

// A status book used to accept any field at all and keep the ones it knew. That
// cost a debugging session: `stoked` was authored with a `name` and a `flavour`
// beside its numbers — exactly the shape a skill and a trait are written in — the
// book parsed clean, and the status reached the battle log as a bare id. Nothing
// between the file and the screen said a word.
//
// Both halves are fixed and they are separate fixes, which is why there are two
// tests here. The field is real now, so the author's `name` means something; and
// an unknown field is refused, so the next author who guesses a field name that
// does NOT exist gets a sentence rather than silence.

// TestAStatusMayCarryItsOwnName is the first half.
func TestAStatusMayCarryItsOwnName(t *testing.T) {
	book, err := status.ParseBook([]byte(`{"max_stacks":5,"max_duration":6,"kinds":[
	  {"id":"scorching","name":"  cháy sém  ","category":"dot","max_stacks":3,
	   "duration":3,"tick_power":500},
	  {"id":"plain","category":"buff","max_stacks":1,"duration":2}
	]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	named, err := book.Lookup("scorching")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	// Trimmed, the way a skill's name is: a name of nothing but spaces is the
	// absent answer rather than a name made of spaces, and this package has no
	// other opinion about the text.
	if named.Name != "cháy sém" {
		t.Errorf("the authored name is %q, want it trimmed to %q", named.Name, "cháy sém")
	}
	unnamed, err := book.Lookup("plain")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if unnamed.Name != "" {
		t.Errorf("a status declaring no name carries %q", unnamed.Name)
	}
}

// TestAFieldTheStatusBookDoesNotKnowIsRefused is the second half, and it is the
// one that would have caught the original defect.
//
// `flavour` is in the table by name rather than as a stand-in for any unknown
// field, because it is the exact field that was written and lost — a skill and a
// trait both take one, so it is the guess an author makes twice.
func TestAFieldTheStatusBookDoesNotKnowIsRefused(t *testing.T) {
	cases := []struct{ name, raw string }{
		{"a flavour clause on a kind", `{"max_stacks":5,"max_duration":6,"kinds":[
		  {"id":"x","category":"dot","max_stacks":1,"duration":1,"tick_power":500,
		   "flavour":"thiêu đốt"}]}`},
		{"a misspelt field on a kind", `{"max_stacks":5,"max_duration":6,"kinds":[
		  {"id":"x","category":"dot","max_stacks":1,"duration":1,"tick_powers":500}]}`},
		{"an unknown field on the book", `{"max_stacks":5,"max_duration":6,"max_charges":9,
		  "kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":1,"tick_power":500}]}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := status.ParseBook([]byte(testCase.raw))
			if err == nil {
				t.Fatal("the book accepted a field it does not know, so the author's line went nowhere and nothing said so")
			}
			if !strings.Contains(err.Error(), "unknown field") {
				t.Errorf("the refusal is %q, which does not name the field as unknown", err)
			}
		})
	}
}
