package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// thornBooks is the shared books with two traits that answer identically except
// for the stat they are priced against.
func thornBooks(t *testing.T, holder string) battle.Books {
	t.Helper()
	shared := books(t)
	answering, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"sharp","grants":[],"replies":{"power":500}},
	  {"id":"armoured","grants":[],"replies":{"power":500,"scaling":{"stat":"defense"}}},
	  {"id":"remembered","grants":[],"replies":{"power":500,"scaling":{"stat":"defense","source":"base"}}}
	]}`), passive.Deps{Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("passives: %v", err)
	}
	shared.Passives = answering
	_ = holder
	return shared
}

// answered is the damage one reply dealt, from a single exchange in which the
// holder is hit once.
func answered(t *testing.T, trait string, attack, defense int64) int64 {
	t.Helper()
	shared := thornBooks(t, trait)
	fight, err := battle.New(shared, 5, []battle.Roster{
		{ID: "holder", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(2400, attack, defense, 1),
			Skills: []string{"strike"}, Passives: []string{trait}},
		{ID: "biter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 300, 400, 200),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "biter" {
		t.Fatalf("the faster unit was %q, so the holder is not the one being hit", prompt.Unit)
	}
	if err := fight.Act("strike", hex.Place(hex.SideAlly, hex.Offset{Col: 2, Row: 1})); err != nil {
		t.Fatalf("strike: %v", err)
	}
	for _, event := range fight.Drain() {
		// The reply is the one carrying a trait rather than a skill.
		if event.Kind == battle.Damaged && event.Passive == trait {
			return event.Amount
		}
	}
	t.Fatalf("%s never answered", trait)
	return 0
}

// TestAReplyIsPricedOffTheStatItNames is the whole of the change.
//
// Both traits answer with the same share. The holder has more defence than
// attack, which is what an armoured unit is, so the one priced off defence must
// hit harder — and a reply that ignored the field would come back with the same
// number twice.
func TestAReplyIsPricedOffTheStatItNames(t *testing.T) {
	const attack, defense = 300, 800
	sharp := answered(t, "sharp", attack, defense)
	armoured := answered(t, "armoured", attack, defense)
	if sharp == 0 || armoured == 0 {
		t.Fatalf("a reply dealt nothing: sharp %d, armoured %d", sharp, armoured)
	}
	if armoured <= sharp {
		t.Errorf("a holder with %d defence and %d attack answered for %d off defence "+
			"and %d off attack, so the declared stat was not read", defense, attack, armoured, sharp)
	}
	// And the ratio is the stats' own, because a reply is priced the way
	// everything else in this engine is.
	if want := sharp * defense / attack; armoured < want-2 || armoured > want+2 {
		t.Errorf("answering off defence dealt %d where the stat ratio says %d", armoured, want)
	}
}

// TestAReplyStillAnswersOffAttackWhenItSaysNothing, because every trait written
// before the field existed said nothing and must not have moved.
func TestAReplyStillAnswersOffAttackWhenItSaysNothing(t *testing.T) {
	shared := thornBooks(t, "sharp")
	held, err := shared.Passives.Lookup("sharp")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if held.Replies.Scaling != skill.DefaultScaling() {
		t.Errorf("a reply that named no stat came out as %+v rather than the default",
			held.Replies.Scaling)
	}
}

// TestAReplyReadsTheModifiedStatUnlessItAsksForTheBase is the other half of the
// declaration, and it is the half that makes a stat trade work.
//
// A trait that trades speed for defence raises the defence a reply is priced
// against, so thorns and armour are bought with one number — but only if the
// reply reads the *current* value. Reading the base line is what "source":
// "base" is for, and a trait that wanted the untouched figure has to say so.
func TestAReplyReadsTheModifiedStatUnlessItAsksForTheBase(t *testing.T) {
	shared := books(t)
	answering, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"armoured","grants":[{"status":"toughened","stacks":2}],
	   "replies":{"power":500,"scaling":{"stat":"defense"}}},
	  {"id":"remembered","grants":[{"status":"toughened","stacks":2}],
	   "replies":{"power":500,"scaling":{"stat":"defense","source":"base"}}}
	]}`), passive.Deps{Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("passives: %v", err)
	}
	shared.Passives = answering

	dealt := func(trait string) int64 {
		t.Helper()
		fight, err := battle.New(shared, 5, []battle.Roster{
			{ID: "holder", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(2400, 300, 800, 1),
				Skills: []string{"strike"}, Passives: []string{trait}},
			{ID: "biter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4000, 300, 400, 200),
				Skills: []string{"strike"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		fight.Drain()
		if _, err := fight.Advance(); err != nil {
			t.Fatalf("advance: %v", err)
		}
		if err := fight.Act("strike", hex.Place(hex.SideAlly, hex.Offset{Col: 2, Row: 1})); err != nil {
			t.Fatalf("strike: %v", err)
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Damaged && event.Passive == trait {
				return event.Amount
			}
		}
		t.Fatalf("%s never answered", trait)
		return 0
	}

	current, base := dealt("armoured"), dealt("remembered")
	if current <= base {
		t.Errorf("with the holder's defence buffed, the reply reading the current value dealt %d "+
			"and the one reading the base line dealt %d, so the source is not read", current, base)
	}
}
