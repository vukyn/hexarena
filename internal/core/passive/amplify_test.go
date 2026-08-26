package passive_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
)

func TestAmplificationsParseAndReadBack(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"virulent","name":"độc lực","grants":[],
	   "amplifies":[{"status":"poison","effect":300,"chance":200}]},
	  {"id":"caustic","grants":[],"amplifies":[{"status":"poison","effect":300}]},
	  {"id":"insidious","grants":[],"amplifies":[{"status":"weaken","chance":500}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// A trait that only amplifies is a trait, on the same terms a trait that only
	// resists is one: making somebody else's turn worse is a thing to do.
	virulent, err := book.Lookup("virulent")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !reflect.DeepEqual(virulent.Amplifies,
		[]passive.Amplification{{Status: "poison", Effect: 300, Chance: 200}}) {
		t.Errorf("the amplification came back as %+v", virulent.Amplifies)
	}

	// Boosts is what the engine asks, so it is pinned directly: both shares from
	// one call, and nought for a status the trait says nothing about.
	effect, chance := virulent.Boosts("poison")
	if effect != 300 || chance != 200 {
		t.Errorf("Boosts(poison) = %d, %d, want the declared 300, 200", effect, chance)
	}
	if effect, chance := virulent.Boosts("weaken"); effect != 0 || chance != 0 {
		t.Errorf("Boosts(weaken) = %d, %d on a trait that says nothing about it", effect, chance)
	}

	// Either share alone, which is the whole reason there are two: a trait may
	// want a stronger tick without a better chance, and the other way round.
	caustic, err := book.Lookup("caustic")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if effect, chance := caustic.Boosts("poison"); effect != 300 || chance != 0 {
		t.Errorf("an effect-only trait reads %d, %d", effect, chance)
	}
	insidious, err := book.Lookup("insidious")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if effect, chance := insidious.Boosts("weaken"); effect != 0 || chance != 500 {
		t.Errorf("a chance-only trait reads %d, %d", effect, chance)
	}
}

func TestAmplificationRejections(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		wantErr string
	}{
		{"an unknown status",
			`[{"id":"a","grants":[],"amplifies":[{"status":"nothing","chance":200}]}]`,
			`unknown status "nothing"`},
		{"neither share",
			`[{"id":"a","grants":[],"amplifies":[{"status":"poison"}]}]`,
			"amplifies \"poison\" by nothing"},
		// The one refusal that is about the *kind* of status rather than the
		// number: a status with no tick has no effect to raise, so the field
		// would be an author waiting for a change that never comes.
		{"an effect on a status that does not tick",
			`[{"id":"a","grants":[],"amplifies":[{"status":"weaken","effect":300}]}]`,
			"only a damage-over-time has a tick this could raise"},
		// A regeneration looks like it should pass — it declares a tick_power and
		// heals from a frozen amount the way a poison damages from one — and it is
		// refused because battle.inflict computes a tick for a Dot and nothing
		// else. See the note on the guard: an applied regeneration freezes nought
		// and heals nothing, which is a bug older than this field.
		{"an effect on a regeneration",
			`[{"id":"a","grants":[],"amplifies":[{"status":"regrowth","effect":300}]}]`,
			"only a damage-over-time has a tick this could raise"},
		{"a chance past the base",
			`[{"id":"a","grants":[],"amplifies":[{"status":"poison","chance":1001}]}]`,
			"amplifies the chance of \"poison\" by 1001"},
		{"an effect past the base",
			`[{"id":"a","grants":[],"amplifies":[{"status":"poison","effect":2000}]}]`,
			"amplifies the effect of \"poison\" by 2000"},
		{"a negative share",
			`[{"id":"a","grants":[],"amplifies":[{"status":"poison","chance":-200}]}]`,
			"amplifies the chance of \"poison\" by -200"},
		{"the same status twice",
			`[{"id":"a","grants":[],"amplifies":[{"status":"poison","chance":200},{"status":"poison","effect":300}]}]`,
			"amplifies \"poison\" twice"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("the refusal is %q, want it to mention %q", err, test.wantErr)
			}
		})
	}
}

// TestAmplifyingAHelpfulStatusIsAllowed is the asymmetry with resistance, and it
// is deliberate rather than an oversight.
//
// A resistance is refused on anything the holder's own side puts on it, because
// refusing your own help is nonsense. Making that help *better* is exactly as
// sensible as making a poison worse, so nothing here asks whether the status is
// harmful — the narrower refusal above (an effect on something with no tick) is
// what carries the real constraint.
func TestAmplifyingAHelpfulStatusIsAllowed(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"encouraging","grants":[],"amplifies":[{"status":"haste","chance":300}]}
	]`)
	if err != nil {
		t.Fatalf("a trait amplifying a buff was refused: %v", err)
	}
	held, err := book.Lookup("encouraging")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if _, chance := held.Boosts("haste"); chance != 300 {
		t.Errorf("the share came back as %d", chance)
	}
	// A regeneration's *chance* may be raised for the same reason a buff's may,
	// which is what shows the harmful split is genuinely absent rather than
	// hiding behind the tick guard above.
	if _, err := parse(t, `[
	  {"id":"nurturing","grants":[],"amplifies":[{"status":"regrowth","chance":300}]}
	]`); err != nil {
		t.Errorf("a trait amplifying a regeneration's chance was refused: %v", err)
	}
}

// TestATraitThatOnlyAmplifiesIsNotEmpty is the "holding it would change nothing"
// guard, which had to learn about the fourth job.
func TestATraitThatOnlyAmplifiesIsNotEmpty(t *testing.T) {
	if _, err := parse(t, `[{"id":"a","grants":[]}]`); err == nil {
		t.Error("a trait that does nothing at all was accepted")
	}
	if _, err := parse(t,
		`[{"id":"a","grants":[],"amplifies":[{"status":"poison","chance":200}]}]`); err != nil {
		t.Errorf("a trait that only amplifies was called empty: %v", err)
	}
}

// TestAmplificationsSurviveTheFileAndAreOmittedWhenThereAreNone is the
// round-trip: what Marshal writes has to parse back to the same trait, and a
// trait amplifying nothing must not gain an empty list it was not authored with.
func TestAmplificationsSurviveTheFileAndAreOmittedWhenThereAreNone(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"virulent","grants":[],"amplifies":[{"status":"poison","effect":300,"chance":200}]},
	  {"id":"plain","grants":[{"status":"fleet"}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"amplifies": []`) {
		t.Errorf("a trait amplifying nothing was written with an empty list:\n%s", raw)
	}
	again, err := passive.ParseBook(raw, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("what Marshal wrote does not parse: %v", err)
	}
	before, err := book.Lookup("virulent")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	after, err := again.Lookup("virulent")
	if err != nil {
		t.Fatalf("lookup after the round trip: %v", err)
	}
	if !reflect.DeepEqual(before.Amplifies, after.Amplifies) {
		t.Errorf("the trip through the file turned %+v into %+v",
			before.Amplifies, after.Amplifies)
	}
}
