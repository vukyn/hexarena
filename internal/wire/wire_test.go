package wire

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestEveryMessageKindHasANameAndABody is the totality guard, and it walks
// KindCount rather than any of the tables it checks.
//
// That is the whole design of it. Ranging over kindNames would ask that table
// whether it holds what it holds, which is how a message ends up in the
// protocol with nothing measuring it — the shape cmd/hexarena-tui's screenCount
// was added for after five screens slipped into the authoring client. Here the
// three ways to add a kind and forget something are each a red line: no name (so
// it serialises as `kind(7)` and no peer can read it), no decoder entry (so it
// parses and then cannot be read), and no fixture (so nothing round-trips it and
// it is in no golden).
func TestEveryMessageKindHasANameAndABody(t *testing.T) {
	fixtures := messageFixtures(t)
	named := make(map[string]Kind, KindCount)
	for value := range KindCount {
		kind := Kind(value)
		name := kindNames[value]
		switch {
		case name == "":
			t.Errorf("kind %d has no name in kindNames, so it would serialise as %q and no peer could read it", value, kind.String())
		case strings.ContainsAny(name, " ABCDEFGHIJKLMNOPQRSTUVWXYZ"):
			t.Errorf("kind %d is named %q; the wire names are lower snake case", value, name)
		}
		if first, taken := named[name]; taken && name != "" {
			t.Errorf("kinds %d and %d are both named %q, so one of them is unreachable", first, value, name)
		}
		named[name] = kind
		if bodyForKind[value] == nil {
			t.Errorf("kind %s has no entry in bodyForKind, so it decodes to nothing", kind)
			continue
		}
		body := bodyForKind[value]()
		if body.Kind() != kind {
			t.Errorf("bodyForKind[%s] builds a %T, whose own Kind is %s", kind, body, body.Kind())
		}
		fixture, ok := fixtures[kind]
		if !ok {
			t.Errorf("kind %s has no entry in messageFixtures, so nothing round-trips it and it is in no golden", kind)
			continue
		}
		if fixture.Kind() != kind {
			t.Errorf("the fixture filed under %s is a %T, whose Kind is %s", kind, fixture, fixture.Kind())
		}
	}
	for kind := range fixtures {
		if int(kind) >= KindCount {
			t.Errorf("messageFixtures holds a fixture for kind %d, which this protocol does not declare", kind)
		}
	}
	if len(fixtures) != KindCount {
		t.Errorf("there are %d fixtures over %d kinds", len(fixtures), KindCount)
	}
	t.Logf("%d kinds, all named, decodable and fixtured", KindCount)
}

// TestEveryMessageRoundTripsThroughTheEnvelope is the property the format rests
// on: what one peer encoded is what the other decodes, field for field.
//
// It compares the decoded value against the fixture with reflect.DeepEqual
// rather than checking a field or two, because the failure this guards against
// is a field that does not survive the trip at all — a missing tag, an
// unexported field, an aim that came back as the ally back corner instead of
// absent. A spot check would pass on every one of those.
func TestEveryMessageRoundTripsThroughTheEnvelope(t *testing.T) {
	for kind, fixture := range messageFixtures(t) {
		t.Run(kind.String(), func(t *testing.T) {
			raw, err := Encode(fixture)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			// The kind travels by name, which is the rule this whole protocol
			// shares with a battle log: a number would silently reinterpret
			// itself the next time a constant was inserted.
			if !strings.Contains(string(raw), `"kind":"`+kind.String()+`"`) {
				t.Errorf("the envelope does not name the kind %q: %s", kind, raw)
			}
			decoded, err := Decode(raw)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded.Kind() != kind {
				t.Fatalf("decoded a %s as a %s", kind, decoded.Kind())
			}
			if !reflect.DeepEqual(decoded, fixture) {
				t.Errorf("the message did not survive the trip\n got %#v\nwant %#v", decoded, fixture)
			}
		})
	}
}

// TestAnUnknownKindIsARefusalRatherThanAZeroValue is the case a peer one version
// ahead produces, and the zero value is the outcome to fear rather than the
// error.
//
// An unknown kind decoding to zero would be read as a hello: the connection
// would carry on, every unreadable message would look like a join, and the
// divergence would surface somewhere with no relation to the cause. A panic
// would at least be loud; a clean error is what lets the room answer with a
// Refused carrying CodeUnknownMessage.
func TestAnUnknownKindIsARefusalRatherThanAZeroValue(t *testing.T) {
	cases := map[string]string{
		"a kind from a later version": `{"kind":"surrender","body":{}}`,
		"a kind written as a number":  `{"kind":3,"body":{}}`,
		"no kind at all":              `{"body":{}}`,
		"a renamed kind":              `{"kind":"Hello","body":{}}`,
		"not an envelope":             `["hello"]`,
		"trailing bytes":              `{"kind":"pass"}{"kind":"pass"}`,
		"a body that is not one":      `{"kind":"welcome","body":7}`,
		"a body-less welcome":         `{"kind":"welcome"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			body, err := Decode([]byte(raw))
			if err == nil {
				t.Fatalf("%s decoded to %#v with no error", raw, body)
			}
			if body != nil {
				t.Errorf("%s returned a usable-looking body alongside its error: %#v", raw, body)
			}
		})
	}
}

// TestAPassIsAnEnvelopeWithNoBodyAtAll is the one message whose emptiness is the
// design rather than a loss in transit, so it is measured rather than assumed:
// Decode accepts an absent body for a pass and for nothing else, which the
// body-less welcome above is the other half of.
func TestAPassIsAnEnvelopeWithNoBodyAtAll(t *testing.T) {
	raw, err := Encode(&Pass{})
	if err != nil {
		t.Fatalf("encode a pass: %v", err)
	}
	// An empty struct marshals to {}, and json.RawMessage("{}") is not empty, so
	// the encoded form carries two bytes of body. Both forms have to decode: the
	// one this package writes, and the barest thing a peer could send.
	for _, form := range []string{string(raw), `{"kind":"pass"}`} {
		decoded, err := Decode([]byte(form))
		if err != nil {
			t.Fatalf("decode %s: %v", form, err)
		}
		if !reflect.DeepEqual(decoded, &Pass{}) {
			t.Errorf("decode %s gave %#v", form, decoded)
		}
	}
	// And nothing crept onto it. A reason is the field this message exists
	// without — see Pass's own comment — so a field added here is a second
	// declaration of battle.NoActionReason and this is where it is caught.
	if fields := reflect.TypeFor[Pass]().NumField(); fields != 0 {
		t.Errorf("Pass carries %d fields; it is defined as carrying nothing at all, and a "+
			"reason on it would be a second declaration of battle.NoActionReason", fields)
	}
}

// TestActCarriesNoUnit is the other half of "the server knows whose turn it is",
// and it is written as a check on the type rather than on a value because the
// mistake it guards against is a field somebody adds in good faith.
//
// A unit on an act would be a second statement of a fact the server already
// owns, and the only thing it could add is a disagreement to resolve. The same
// reasoning keeps a whole battle.Decision off the client's side of the wire —
// Decision carries a unit, a turn number and a pass reason, all three of which
// are the server's to record — so an Act with a Decision in it fails here too.
func TestActCarriesNoUnit(t *testing.T) {
	banned := map[string]string{
		"unit":     "the server knows whose turn it is",
		"turn":     "the turn number is the server's count",
		"decision": "a whole Decision carries the unit, the turn and the pass reason",
		"reason":   "a pass reason is battle.NoActionReason and nothing else",
		"side":     "the side arrives on Start and does not change mid-battle",
		"seat":     "the seat arrives on Welcome and holds for the match",
	}
	structure := reflect.TypeFor[Act]()
	for index := range structure.NumField() {
		field := structure.Field(index)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if because, refused := banned[strings.ToLower(tag)]; refused {
			t.Errorf("Act carries %s (%s): %s", field.Name, tag, because)
		}
	}
	// Non-vacuity: the check above passes on an Act with no fields at all, and
	// an act does have two things to say.
	if structure.NumField() != 2 {
		t.Errorf("Act has %d fields, want the two it is defined as: a skill and an aim", structure.NumField())
	}
}

// TestEncodeRefusesWhatItCannotLabel covers the two ways a caller can hand
// Encode something it must not put on the wire. Both are errors rather than an
// envelope labelled with whatever sat at zero, which would be a hello.
func TestEncodeRefusesWhatItCannotLabel(t *testing.T) {
	if raw, err := Encode(nil); err == nil {
		t.Errorf("encoding no body gave %s", raw)
	}
	if raw, err := Encode(strayBody{}); err == nil {
		t.Errorf("encoding a body claiming kind %d gave %s", KindCount, raw)
	}
	// And an envelope built in Go around an undeclared kind is refused on the
	// way in too, which is the path Kind.UnmarshalJSON cannot cover.
	raw, err := json.Marshal(Envelope{Kind: Kind(KindCount), Body: json.RawMessage(`{}`)})
	if err != nil {
		// Marshalling it is expected to fail through Kind.MarshalJSON's own
		// fallback name, so both outcomes are fine; what matters is that no
		// path produces a body.
		return
	}
	if body, err := Decode(raw); err == nil {
		t.Errorf("an envelope naming kind %d decoded to %#v", KindCount, body)
	}
}

// strayBody is a body claiming a kind the protocol does not declare, which is
// what a half-finished new message looks like before its tables are filled in.
type strayBody struct{}

func (strayBody) Kind() Kind { return Kind(KindCount) }
