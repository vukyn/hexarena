package passive_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// TestAReplyIsCarriedThroughTheFile is the declaration surviving a round trip,
// which is what an authoring tool depends on: hexforge writes a book by
// marshalling one it parsed, so a field the writer forgets is a field an author
// silently loses.
func TestAReplyIsCarriedThroughTheFile(t *testing.T) {
	book, err := parse(t, `[
	  {"id":"thorns","replies":{"power":300,"applies":[{"status":"poison","chance":400,"stacks":2}]}},
	  {"id":"scratch","replies":{"power":120}},
	  {"id":"caustic","replies":{"applies":[{"status":"poison","chance":1000}]}}
	]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	thorns, err := book.Lookup("thorns")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if thorns.Replies == nil {
		t.Fatal("the reply was dropped")
	}
	if thorns.Replies.Power != 300 {
		t.Errorf("the reply answers for %d, want 300", thorns.Replies.Power)
	}
	if len(thorns.Replies.Applies) != 1 {
		t.Fatalf("the reply applies %d statuses, want 1", len(thorns.Replies.Applies))
	}
	if got := thorns.Replies.Applies[0]; got.Status != "poison" || got.Chance != 400 || got.Stacks != 2 {
		t.Errorf("the reply's application reads %+v, want poison at 400 for 2 stacks", got)
	}
	// Damage alone and a status alone are both whole answers, which is why the
	// two halves are separate fields rather than one required pair.
	scratch, _ := book.Lookup("scratch")
	if scratch.Replies == nil || len(scratch.Replies.Applies) != 0 || scratch.Replies.Power != 120 {
		t.Errorf("a damage-only reply reads %+v", scratch.Replies)
	}
	caustic, _ := book.Lookup("caustic")
	if caustic.Replies == nil || caustic.Replies.Power != 0 || len(caustic.Replies.Applies) != 1 {
		t.Errorf("a status-only reply reads %+v", caustic.Replies)
	}

	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := parse(t, strings.TrimSuffix(
		strings.TrimPrefix(string(raw), "{\n  \"passives\": "), "\n}\n"))
	if err != nil {
		t.Fatalf("reparse what was written: %v\n%s", err, raw)
	}
	back, err := again.Lookup("thorns")
	if err != nil {
		t.Fatalf("lookup after a round trip: %v", err)
	}
	if back.Replies == nil || back.Replies.Power != thorns.Replies.Power ||
		len(back.Replies.Applies) != len(thorns.Replies.Applies) {
		t.Errorf("the reply came back as %+v, want %+v", back.Replies, thorns.Replies)
	}
}

// TestReplyRejections are the declarations refused rather than accepted and
// quietly ignored.
//
// The three status rules are the same three a rider obeys, and that is the
// point: a rider and a reply differ in when they fire and in nothing else, so
// they read one another's checks. A copy of them here that had drifted would be
// the bug this shares a function to prevent.
func TestReplyRejections(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			"a reply that does nothing",
			`[{"id":"odd","replies":{}}]`,
			"no damage and no status",
		},
		{
			"a reply that heals what attacked it",
			`[{"id":"odd","replies":{"power":-100}}]`,
			"cannot heal what attacked it",
		},
		{
			"a reply applying a permanent status",
			`[{"id":"odd","replies":{"applies":[{"status":"toughened","chance":500}]}}]`,
			"permanent",
		},
		{
			"a reply applying an unknown status",
			`[{"id":"odd","replies":{"applies":[{"status":"nonesuch","chance":500}]}}]`,
			"nonesuch",
		},
		{
			"a reply with a chance past a thousand",
			`[{"id":"odd","replies":{"applies":[{"status":"poison","chance":1200}]}}]`,
			"parts per thousand",
		},
		{
			"a reply with a chance of nought",
			`[{"id":"odd","replies":{"applies":[{"status":"poison","chance":0}]}}]`,
			"parts per thousand",
		},
		{
			"a reply naming one status twice",
			`[{"id":"odd","replies":{"applies":[{"status":"poison","chance":300},{"status":"poison","chance":400}]}}]`,
			"twice",
		},
		{
			"a reply over the stack cap",
			`[{"id":"odd","replies":{"applies":[{"status":"poison","chance":300,"stacks":9}]}}]`,
			"caps at",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parse(t, test.body)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("%s was refused with %q, want it to mention %q",
					test.name, err, test.wantErr)
			}
		})
	}
}

// TestATraitThatOnlyRepliesIsWorthHolding is the "changes nothing" refusal
// knowing about the new field.
//
// A trait that granted nothing, resisted nothing and added nothing used to be
// refused as a trait with no effect, and a reply is a fourth way to have one —
// so the check had to learn about it or a perfectly good trait would have been
// turned away.
func TestATraitThatOnlyRepliesIsWorthHolding(t *testing.T) {
	if _, err := parse(t, `[{"id":"thorns","replies":{"power":300}}]`); err != nil {
		t.Errorf("a trait whose only effect is a reply was refused: %v", err)
	}
	if _, err := parse(t, `[{"id":"nothing"}]`); err == nil {
		t.Error("a trait that does nothing at all was accepted")
	}
}

// TestAllHandsOutACopyOfTheReply is the same rule the condition follows: a
// caller editing what it was handed must not be editing the book.
//
// The reply is a pointer holding a slice, so it is two copies rather than one,
// and the slice is the half that is easy to forget.
func TestAllHandsOutACopyOfTheReply(t *testing.T) {
	book, err := parse(t,
		`[{"id":"thorns","replies":{"power":300,"applies":[{"status":"poison","chance":400}]}}]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	handed := book.All()
	if len(handed) != 1 || handed[0].Replies == nil {
		t.Fatalf("the book handed out %+v", handed)
	}
	handed[0].Replies.Power = 1
	handed[0].Replies.Applies[0].Chance = 1

	again, err := book.Lookup("thorns")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if again.Replies.Power != 300 {
		t.Errorf("editing the copy changed the book's power to %d", again.Replies.Power)
	}
	if again.Replies.Applies[0].Chance != 400 {
		t.Errorf("editing the copy changed the book's chance to %d",
			again.Replies.Applies[0].Chance)
	}
}

// TestAReplyScalingSurvivesBeingWrittenBack is the round trip every authored
// field has to make.
//
// hexforge reads the book and writes it back, so a field the writer drops is a
// field an author loses by opening the tool -- and this one would be lost
// quietly, because the reply still works and simply answers off the wrong stat.
// A mutation that dropped it passed every other test in the package.
func TestAReplyScalingSurvivesBeingWrittenBack(t *testing.T) {
	book, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"armoured","grants":[],"replies":{"power":500,"scaling":{"stat":"defense"}}},
	  {"id":"remembered","grants":[],"replies":{"power":500,"scaling":{"stat":"speed","source":"base"}}},
	  {"id":"sharp","grants":[],"replies":{"power":500}}
	]}`), passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	written, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := passive.ParseBook(written, passive.Deps{Statuses: statuses(t)})
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	for _, want := range []struct {
		id      string
		scaling skill.Scaling
	}{
		{"armoured", skill.Scaling{Stat: progression.Defense, Source: combat.CurrentStat}},
		{"remembered", skill.Scaling{Stat: progression.Speed, Source: combat.BaseStat}},
		{"sharp", skill.DefaultScaling()},
	} {
		held, err := again.Lookup(want.id)
		if err != nil {
			t.Fatalf("lookup %s: %v", want.id, err)
		}
		if held.Replies.Scaling != want.scaling {
			t.Errorf("%s came back scaling off %+v, wanted %+v",
				want.id, held.Replies.Scaling, want.scaling)
		}
	}
	// The default is written as nothing, so a book of traits that all take it
	// reads exactly as it did before the field existed.
	if strings.Contains(string(written), `"scaling"`) != true {
		t.Fatal("the two declared scalings were not written at all")
	}
	if strings.Count(string(written), `"scaling"`) != 2 {
		t.Errorf("the default scaling was written out as well: %s", written)
	}
}

// TestAReplyWithNoDamageMayNotNameAStat, because there is nothing for the stat
// to price. A trait that answered with a status alone and named defence would be
// an author waiting for a change that never arrives.
func TestAReplyWithNoDamageMayNotNameAStat(t *testing.T) {
	_, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"quiet","grants":[],
	   "replies":{"applies":[{"status":"poison","chance":100}],"scaling":{"stat":"defense"}}}
	]}`), passive.Deps{Statuses: statuses(t)})
	if err == nil {
		t.Fatal("a reply with no damage was allowed to name a stat to scale it off")
	}
	if !strings.Contains(err.Error(), "no damage") {
		t.Errorf("the refusal says %q, which does not say why", err)
	}
}
