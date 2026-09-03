package wire

import (
	"reflect"
	"strings"
	"testing"
)

// TestEveryClosureHasANameAndTravels is the closure half of the totality guard,
// and it walks ClosureCount for the reason the kind and code walks do: ranging
// over closureNames would ask that table whether it holds what it holds, which
// is how a value ends up in the protocol with nothing measuring it.
//
// ⚠️ What this test **cannot** measure is the same thing the code walk cannot:
// nothing words these yet. wire must not import internal/i18n, because the whole
// point of sending an id is that the wording lives at the far end — so the count
// is what is held here and the gap is written into TODO.md § The client, beside
// the codes' own entry, rather than left silent.
func TestEveryClosureHasANameAndTravels(t *testing.T) {
	named := make(map[string]Closure, ClosureCount)
	for value := range ClosureCount {
		closure := Closure(value)
		name := closureNames[value]
		switch {
		case name == "":
			t.Errorf("closure %d has no name, so it would travel as %q and no client could word it",
				value, closure.String())
			continue
		case strings.ContainsAny(name, " ABCDEFGHIJKLMNOPQRSTUVWXYZ"):
			t.Errorf("closure %d is named %q; the wire names are lower snake case", value, name)
		}
		if first, taken := named[name]; taken {
			t.Errorf("closures %d and %d are both named %q, so one of them is unreachable", first, value, name)
		}
		named[name] = closure
		// The round trip, per closure, through the message that carries it. One
		// that marshalled and did not come back is a match ending the far end
		// reads as something else.
		raw, err := Encode(&Closed{Reason: closure})
		if err != nil {
			t.Errorf("encode a closed carrying %s: %v", closure, err)
			continue
		}
		if !strings.Contains(string(raw), `"reason":"`+name+`"`) {
			t.Errorf("a closed carrying %s does not name it: %s", closure, raw)
		}
		decoded, err := Decode(raw)
		if err != nil {
			t.Errorf("decode a closed carrying %s: %v", closure, err)
			continue
		}
		closed, isClosed := decoded.(*Closed)
		if !isClosed {
			t.Errorf("a closed decoded as %T", decoded)
			continue
		}
		if closed.Reason != closure {
			t.Errorf("a closed carrying %s came back carrying %s", closure, closed.Reason)
		}
	}
	if len(named) != ClosureCount {
		t.Errorf("%d distinct names over %d closures", len(named), ClosureCount)
	}
	t.Logf("%d closures, all named and all round-tripping", ClosureCount)
}

// TestAClosedNeverCarriesProse is the design record's rule as a measurement, one
// message along from the refusal it shares it with: the server sends an id and
// the client words it, so a Closed has one field and no room for a sentence.
//
// A message with somewhere to put a sentence is a message that will eventually
// have one in it, and the sentence will be in one language. The other half is
// that a Closed must carry no *battle* fact either — an outcome, a winner, a
// standing — because every one of those is something the client computes from
// its own battle, and a second declaration of one is the mistake the missing
// series-standing message exists to avoid.
func TestAClosedNeverCarriesProse(t *testing.T) {
	structure := reflect.TypeFor[Closed]()
	if fields := structure.NumField(); fields != 1 {
		t.Errorf("Closed carries %d fields; it is defined as a closure and nothing else, because a "+
			"server that sent prose would decide what language its clients read in", fields)
	}
	banned := map[string]string{
		"message":     "the client words the closure from its own books",
		"reason_text": "the reason travels as an id, not as a sentence",
		"outcome":     "how a battle ended is battle.Outcome, read off the client's own Ended event",
		"winner":      "the client computes the standing from its own battles",
		"wins":        "the client computes the standing from its own battles",
		"battle":      "which battle it was is on Start",
		"seat":        "the seat arrives on Welcome and holds for the match",
	}
	for index := range structure.NumField() {
		field := structure.Field(index)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if because, refused := banned[strings.ToLower(tag)]; refused {
			t.Errorf("Closed carries %s (%s): %s", field.Name, tag, because)
		}
	}
	if Closure(ClosureCount).Closes() {
		t.Error("a closure this protocol does not declare reports itself as an ending")
	}
	if ClosureNone.Closes() {
		t.Error("ClosureNone reports itself as an ending, so a Closed carrying nothing would look like one")
	}
	for value := 1; value < ClosureCount; value++ {
		if !Closure(value).Closes() {
			t.Errorf("%s does not report itself as an ending", Closure(value))
		}
	}
}

// TestAnUnknownClosureIsRefusedRatherThanReadAsTheFirstOne is the trap the
// unknown-kind and unknown-code tests cover, one message along: a closure from a
// later version decoding to zero would be read as ClosureNone, so a client would
// be told the match was over and given no reason it could word — which is worse
// than not being told at all, because the screen would then say nothing.
func TestAnUnknownClosureIsRefusedRatherThanReadAsTheFirstOne(t *testing.T) {
	for _, raw := range []string{
		`{"kind":"closed","body":{"reason":"host_shut_the_room"}}`,
		`{"kind":"closed","body":{"reason":1}}`,
		`{"kind":"closed","body":{"reason":"Left"}}`,
	} {
		if body, err := Decode([]byte(raw)); err == nil {
			t.Errorf("%s decoded to %#v with no error", raw, body)
		}
	}
}
