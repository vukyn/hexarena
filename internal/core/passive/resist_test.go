package passive_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
)

func TestResistancesParseAndReadBack(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"clean_blood","name":"máu sạch","grants":[],"resists":[{"status":"poison","amount":1000}]},
	  {"id":"stalwart","grants":[{"status":"toughened","stacks":2}],
	   "resists":[{"status":"poison","amount":400},{"status":"weaken","amount":250}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// A trait that only resists is a trait: refusing a status is a thing to do,
	// and demanding a grant alongside it would mean inventing a stat change
	// nobody wanted just to make the entry legal.
	clean, err := book.Lookup("clean_blood")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(clean.Grants) != 0 {
		t.Errorf("a resistance-only trait came back granting %+v", clean.Grants)
	}
	if !reflect.DeepEqual(clean.Resists,
		[]passive.Resistance{{Status: "poison", Amount: 1000}}) {
		t.Errorf("the resistance came back as %+v", clean.Resists)
	}

	// Refuses is what the engine asks, so it is worth pinning directly: the
	// named status, and nought for everything else.
	if got := clean.Refuses("poison"); got != 1000 {
		t.Errorf("Refuses(poison) = %d, want the declared thousand", got)
	}
	if got := clean.Refuses("weaken"); got != 0 {
		t.Errorf("Refuses(weaken) = %d on a trait that says nothing about it", got)
	}
	stalwart, err := book.Lookup("stalwart")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := stalwart.Refuses("weaken"); got != 250 {
		t.Errorf("Refuses(weaken) = %d, want 250", got)
	}
	if len(stalwart.Grants) != 1 {
		t.Errorf("a trait that both grants and resists lost its grant: %+v", stalwart.Grants)
	}
}

func TestResistanceRejections(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			"an unknown status",
			`[{"id":"odd","resists":[{"status":"glow","amount":500}]}]`,
			"unknown status",
		},
		{
			"a buff, which the holder's own side puts on it",
			`[{"id":"sour","resists":[{"status":"haste","amount":500}]}]`,
			"nothing to refuse",
		},
		{
			"a permanent buff, which is what a trait grants",
			`[{"id":"sour","resists":[{"status":"toughened","amount":500}]}]`,
			"nothing to refuse",
		},
		{
			"a share of nought",
			`[{"id":"idle","resists":[{"status":"poison","amount":0}]}]`,
			"parts per thousand",
		},
		{
			"a share past a thousand",
			`[{"id":"much","resists":[{"status":"poison","amount":1500}]}]`,
			"parts per thousand",
		},
		{
			"a negative share",
			`[{"id":"odd","resists":[{"status":"poison","amount":-100}]}]`,
			"parts per thousand",
		},
		{
			"the same status twice",
			`[{"id":"twice","resists":[
			  {"status":"poison","amount":400},{"status":"poison","amount":400}]}]`,
			"twice",
		},
		{
			"a trait that does nothing at all",
			`[{"id":"idle","grants":[],"resists":[]}]`,
			"holding it would change nothing",
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

// TestOnlyAHarmfulStatusCanBeResisted states the rule as the thing it prevents.
//
// A trait resisting a buff would be refusing its own side's help, and one
// resisting a shield or a regeneration the same. Harmful is the split the cleanse
// already leans on, so the two cannot end up disagreeing about which categories
// are an attack.
func TestOnlyAHarmfulStatusCanBeResisted(t *testing.T) {
	for _, harmful := range []string{"poison", "weaken", "stun"} {
		body := `[{"id":"x","resists":[{"status":"` + harmful + `","amount":500}]}]`
		if _, err := parse(t, body); err != nil {
			t.Errorf("resisting the harmful %q was refused: %v", harmful, err)
		}
	}
	for _, benign := range []string{"haste", "block", "regrowth"} {
		body := `[{"id":"x","resists":[{"status":"` + benign + `","amount":500}]}]`
		if _, err := parse(t, body); err == nil {
			t.Errorf("resisting the harmless %q was accepted", benign)
		}
	}
}

func TestResistancesSurviveTheFileAndAreOmittedWhenThereAreNone(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"plain","grants":[{"status":"toughened"}]},
	  {"id":"clean_blood","grants":[],"resists":[{"status":"poison","amount":1000}]}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"amount": 1000`) {
		t.Errorf("the resistance was not written:\n%s", raw)
	}
	// A trait that resists nothing writes no resists block, so a book from before
	// resistances existed round-trips to the bytes it was authored as.
	if strings.Count(string(raw), `"resists"`) != 1 {
		t.Errorf("a trait resisting nothing still wrote the block:\n%s", raw)
	}
	reparsed, err := passive.ParseBook(raw, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("the rendering does not parse back: %v\n%s", err, raw)
	}
	if !reflect.DeepEqual(reparsed.All(), book.All()) {
		t.Errorf("the trip through the file changed the book:\n%+v\n%+v",
			reparsed.All(), book.All())
	}
	// All hands out a copy of the resistances too, not just of the grants.
	handed := book.All()
	for i := range handed {
		for j := range handed[i].Resists {
			handed[i].Resists[j].Amount = 1
		}
	}
	if book.All()[1].Resists[0].Amount != 1000 {
		t.Error("editing the copy changed the book's resistances")
	}
}
