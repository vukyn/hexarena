package wire

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestTheEventDigestReadsEveryFieldOfAnEvent is the load-bearing test of this
// file, and the reason it walks the struct by reflection instead of naming
// fields is that the failure it guards against is **a digest over a subset**.
//
// A hand-written test over two or three fields passes on a digest that marshals
// only `kind` and `at` — and such a digest would agree on two turns that
// differed in everything a reader cares about, which is precisely the
// divergence the per-turn comparison exists to catch. battle.Event has
// twenty-odd fields and gains one every time the engine learns to say something
// new; this walk covers the next one for free.
//
// It is also why DigestEvents marshals the event through battle.Event's own json
// tags rather than listing fields: the shape of an event is declared once, in
// the package that owns it, and a field added there enters this digest with no
// edit here.
func TestTheEventDigestReadsEveryFieldOfAnEvent(t *testing.T) {
	base := battle.Event{}
	before := mustDigestEvents(t, []battle.Event{base})
	structure := reflect.TypeOf(base)
	walked := 0
	for index := range structure.NumField() {
		field := structure.Field(index)
		t.Run(field.Name, func(t *testing.T) {
			mutated := base
			value := reflect.ValueOf(&mutated).Elem().Field(index)
			if !moveField(value, field) {
				t.Fatalf("battle.Event gained a %s field (%s) and this walk does not know how to "+
					"change one, so the digest's sensitivity to it is unmeasured", field.Type, field.Name)
			}
			after := mustDigestEvents(t, []battle.Event{mutated})
			if after == before {
				t.Fatalf("%s moved and the digest did not: two peers would agree about a turn they "+
					"disagree about", field.Name)
			}
			walked++
		})
	}
	// Non-vacuity, the way seed's walk states it: a walk over no fields would
	// pass this whether or not the digest read anything.
	if walked != structure.NumField() {
		t.Fatalf("walked %d of battle.Event's %d fields", walked, structure.NumField())
	}
	t.Logf("every one of battle.Event's %d fields moves the digest", structure.NumField())
}

// moveField sets a field to something that is not its zero value, and reports
// whether it knew how. A false answer is a red test above rather than a skip:
// a field this walk cannot vary is a field the digest is unmeasured on, and the
// two are indistinguishable from the outside.
//
// The two named types are here rather than under their reflect.Kind because
// hex.Cell's absence lives in an unexported field — hex.At is the only thing
// that can build a present one — and because a Kind, a Side and an Outcome all
// serialise by name, so a value past the end of their tables would move the
// digest for the wrong reason.
func moveField(value reflect.Value, field reflect.StructField) bool {
	switch field.Type {
	case reflect.TypeFor[hex.Cell]():
		value.Set(reflect.ValueOf(hex.At(hex.Offset{Col: 3, Row: 2})))
		return true
	case reflect.TypeFor[hex.Side]():
		value.Set(reflect.ValueOf(hex.SideEnemy))
		return true
	case reflect.TypeFor[battle.Outcome]():
		value.Set(reflect.ValueOf(battle.Victory))
		return true
	case reflect.TypeFor[battle.Kind]():
		value.Set(reflect.ValueOf(battle.Damaged))
		return true
	}
	switch field.Type.Kind() {
	case reflect.String:
		value.SetString("moved")
		return true
	case reflect.Bool:
		value.SetBool(true)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(7)
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(7)
		return true
	}
	return false
}

// TestTwoIdenticalRunsOfEventsAgreeAndOrderMatters is the other half of what the
// mirror rests on, and the two halves fail in opposite directions.
//
// Agreement is what makes a matching digest mean "the same turn happened":
// without it every turn would look like a divergence and the check would be
// switched off within a day. Order mattering is what makes a *mismatching*
// digest mean something: a digest over a set rather than a sequence would agree
// about a turn whose events resolved in a different order, which is exactly the
// class of divergence this engine's determinism rules exist to prevent — the
// map-iteration ban, `Set.SpendPool` draining oldest first, a tick resolving in
// application order.
func TestTwoIdenticalRunsOfEventsAgreeAndOrderMatters(t *testing.T) {
	events := fixtureEvents()
	first := mustDigestEvents(t, events)
	second := mustDigestEvents(t, fixtureEvents())
	if first != second {
		t.Fatalf("two identical runs digested differently: %s then %s", first.Short(), second.Short())
	}
	// A fresh slice with the same values, to say the digest reads the values
	// rather than the backing array.
	copied := append([]battle.Event(nil), events...)
	if got := mustDigestEvents(t, copied); got != first {
		t.Errorf("a copy of the same events digested as %s", got.Short())
	}
	reversed := []battle.Event{events[1], events[0]}
	if got := mustDigestEvents(t, reversed); got == first {
		t.Error("the same events in the other order digested the same: the digest reads a set rather than a sequence")
	}
	// And a run is not its own prefix, which is what the length prefix and the
	// walk together buy: a turn that emitted one more event is a different turn.
	if got := mustDigestEvents(t, events[:1]); got == first {
		t.Error("a prefix of the events digested the same as the whole run")
	}
	// The empty run is a real answer rather than an error — a turn control took
	// emits nothing a client has to compare — and it is not the zero digest,
	// because sha256 of nothing is a value.
	empty := mustDigestEvents(t, nil)
	if empty == first {
		t.Error("no events digested the same as two events")
	}
	if empty == (EventDigest{}) {
		t.Error("the digest of no events is the zero value, so an uninitialised field would look like an empty turn")
	}
}

// TestAnEventDigestSurvivesTheWireAsHex is the encoding, and the malformed cases
// are errors rather than a zero digest for the same reason the data digest's are:
// two peers agreeing on the digest of nothing is the failure this check exists
// to catch, and a zero value is what an absent field decodes to.
func TestAnEventDigestSurvivesTheWireAsHex(t *testing.T) {
	digest := mustDigestEvents(t, fixtureEvents())
	raw, err := digest.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `"` + digest.String() + `"`; string(raw) != want {
		t.Errorf("a digest marshalled as %s, want %s", raw, want)
	}
	if len(digest.String()) != 64 {
		t.Errorf("String is %d characters, want 64", len(digest.String()))
	}
	if strings.ToLower(digest.String()) != digest.String() {
		t.Errorf("String is not lowercase hex: %s", digest)
	}
	if digest.Short() != digest.String()[:12] {
		t.Errorf("Short %s is not the first twelve of %s", digest.Short(), digest)
	}
	var back EventDigest
	if err := back.UnmarshalJSON(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != digest {
		t.Errorf("a digest came back as %s", back.Short())
	}
	for _, broken := range []string{`"nope"`, `"00"`, `""`, `7`, `null`, `"` + digest.String() + `ff"`} {
		var out EventDigest
		if err := out.UnmarshalJSON([]byte(broken)); err == nil {
			t.Errorf("%s decoded to %s with no error", broken, out.Short())
		}
	}
}

// TestTheDataDigestAndTheEventDigestAreDifferentTypes is a short test about a
// decision rather than about behaviour, and it is written down because the
// alternative compiles.
//
// Both are thirty-two bytes of sha256 and both print as sixty-four hex
// characters, so sharing one type would be the obvious tidy-up — and it would
// make comparing a *data* digest against an *event* digest a legal expression.
// The two answer different questions (will these peers simulate the same battle,
// once; did they, this turn) so a check that confused them would pass while
// measuring nothing at all.
func TestTheDataDigestAndTheEventDigestAreDifferentTypes(t *testing.T) {
	if reflect.TypeFor[EventDigest]() == reflect.TypeFor[Digest]() {
		t.Fatal("the event digest and the data digest are one type, so comparing one against the other compiles")
	}
	if reflect.TypeFor[Turn]().Field(1).Type != reflect.TypeFor[EventDigest]() {
		t.Error("Turn's second field is not an EventDigest")
	}
	if got, _ := reflect.TypeFor[Version]().FieldByName("Data"); got.Type != reflect.TypeFor[Digest]() {
		t.Error("Version.Data is not a wire.Digest")
	}
}

// mustDigestEvents is the shorthand every case above uses. A digest over a
// hand-written slice has no reason to fail, and a failure is the test's own bug.
func mustDigestEvents(t *testing.T, events []battle.Event) EventDigest {
	t.Helper()
	digest, err := DigestEvents(events)
	if err != nil {
		t.Fatalf("digest %d events: %v", len(events), err)
	}
	return digest
}
