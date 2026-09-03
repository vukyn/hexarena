package wire

import (
	"reflect"
	"strings"
	"testing"
)

// TestEveryRefusalCodeHasANameAndTravels is the code half of the totality guard,
// and it walks CodeCount for the reason the kind walk does: ranging over
// codeNames would ask that table whether it holds what it holds.
//
// ⚠️ What this test **cannot** measure is the wording, and saying so is better
// than implying otherwise. wire must not import internal/i18n — the whole point
// of a code is that the wording lives at the far end — so the count is what is
// held here. That wording now exists: TestEveryRefusalCodeIsWorded and its three
// neighbours in internal/i18n/protocol_test.go walk this count against both
// books, which is where they were always going to belong. What is still not
// measured anywhere is whether a **player** is ever shown one — no screen calls
// Lang.Refusal yet — so a code here is worded and unread. → TODO.md § The client.
func TestEveryRefusalCodeHasANameAndTravels(t *testing.T) {
	named := make(map[string]Code, CodeCount)
	for value := range CodeCount {
		code := Code(value)
		name := codeNames[value]
		switch {
		case name == "":
			t.Errorf("code %d has no name, so it would travel as %q and no client could word it", value, code.String())
			continue
		case strings.ContainsAny(name, " ABCDEFGHIJKLMNOPQRSTUVWXYZ"):
			t.Errorf("code %d is named %q; the wire names are lower snake case", value, name)
		}
		if first, taken := named[name]; taken {
			t.Errorf("codes %d and %d are both named %q, so one of them is unreachable", first, value, name)
		}
		named[name] = code
		// The round trip, per code, through the message that carries it. A code
		// that marshalled and did not come back is a refusal the far end reads
		// as something else.
		raw, err := Encode(&Refused{Code: code})
		if err != nil {
			t.Errorf("encode a refusal carrying %s: %v", code, err)
			continue
		}
		if !strings.Contains(string(raw), `"code":"`+name+`"`) {
			t.Errorf("a refusal carrying %s does not name it: %s", code, raw)
		}
		decoded, err := Decode(raw)
		if err != nil {
			t.Errorf("decode a refusal carrying %s: %v", code, err)
			continue
		}
		refused, isRefusal := decoded.(*Refused)
		if !isRefusal {
			t.Errorf("a refusal decoded as %T", decoded)
			continue
		}
		if refused.Code != code {
			t.Errorf("a refusal carrying %s came back carrying %s", code, refused.Code)
		}
	}
	if len(named) != CodeCount {
		t.Errorf("%d distinct names over %d codes", len(named), CodeCount)
	}
	t.Logf("%d codes, all named and all round-tripping", CodeCount)
}

// TestARefusalNeverCarriesProse is the design record's rule as a measurement:
// the server sends a code and the client words it, so a Refused has one field
// and no room for a sentence.
//
// A message with somewhere to put a sentence is a message that will eventually
// have one in it, and the sentence will be in one language.
func TestARefusalNeverCarriesProse(t *testing.T) {
	if fields := refusalFields(); fields != 1 {
		t.Errorf("Refused carries %d fields; it is defined as a code and nothing else, because a "+
			"server that sent prose would decide what language its clients read in", fields)
	}
	if Code(CodeCount).Refuses() {
		t.Error("a code this protocol does not declare reports itself as a refusal")
	}
	if CodeNone.Refuses() {
		t.Error("CodeNone reports itself as a refusal, so a Refused carrying nothing would look like one")
	}
	for value := 1; value < CodeCount; value++ {
		if !Code(value).Refuses() {
			t.Errorf("%s does not report itself as a refusal", Code(value))
		}
	}
}

// TestAnUnknownRefusalCodeIsRefusedRatherThanReadAsTheFirstOne is the same trap
// the unknown kind test covers, one layer in: a code from a later version
// decoding to zero would tell a client the join was fine and then go quiet.
func TestAnUnknownRefusalCodeIsRefusedRatherThanReadAsTheFirstOne(t *testing.T) {
	for _, raw := range []string{
		`{"kind":"refused","body":{"code":"seat_taken"}}`,
		`{"kind":"refused","body":{"code":4}}`,
		`{"kind":"refused","body":{"code":"None"}}`,
	} {
		if body, err := Decode([]byte(raw)); err == nil {
			t.Errorf("%s decoded to %#v with no error", raw, body)
		}
	}
}

// refusalFields is how many things a Refused carries, read off the type so that
// a field added in good faith reddens the test above rather than shipping.
func refusalFields() int { return reflect.TypeFor[Refused]().NumField() }
