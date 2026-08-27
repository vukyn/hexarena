package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
)

// provoke gives the attacker its turn against the trait's holder and returns
// what the whole exchange logged: the strike, the answer, and anything the
// answer paid back.
func provoke(t *testing.T, fight *battle.Battle, holderHP int64) []battle.Event {
	t.Helper()
	fight.Begin()
	fight.Drain()
	unitByID(t, fight, "a").HP = holderHP
	// The foe is slower, so the holder acts first and passes: what is being
	// measured is what happens on somebody else's turn.
	for range 4 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			t.Fatal("the battle ended before the attacker acted")
		}
		if prompt.Unit == "a" {
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
			fight.Drain()
			continue
		}
		if err := fight.Act("strike", unitByID(t, fight, "a").Cell); err != nil {
			t.Fatalf("strike: %v", err)
		}
		return fight.Drain()
	}
	t.Fatal("the attacker never got a turn")
	return nil
}

// drains is every heal a drain produced, which the log tells apart from any
// other heal by the share it carries.
func drains(events []battle.Event) []battle.Event {
	out := make([]battle.Event, 0, len(events))
	for _, event := range events {
		if event.Kind == battle.Healed && event.Drained > 0 {
			out = append(out, event)
		}
	}
	return out
}

// TestAReplyDrainsForItsHolder is the job that was rendering in the description
// and not in the engine.
//
// A trait's drain is described as "mọi đòn của nó hút lại 25% sát thương gây
// ra" — everything it does takes back a share — and a reply is one of the things
// it does. resolveAgainst drained and reply did not, so a trait holding both
// jobs promised a share of a reply it never paid.
//
// One trait holding both is the case that matters: a placement brings a single
// trait, so no unit can carry a replier and a drainer separately, and nothing
// stops a trait being both.
func TestAReplyDrainsForItsHolder(t *testing.T) {
	// The control first, so the figure below is a measurement rather than a
	// number: the same reply from a trait that does not drain pays nothing.
	plain := provoke(t, answering(t, "spiked", "", []string{"strike"}, []string{"strike"}), 1000)
	if paid := drains(plain); len(paid) != 0 {
		t.Fatalf("a trait that only replies drained anyway: %+v", paid)
	}

	events := provoke(t, answering(t, "barbed", "", []string{"strike"}, []string{"strike"}), 1000)
	answered := replies(events)
	if len(answered) != 1 {
		t.Fatalf("the trait answered %d times, want once", len(answered))
	}
	paid := drains(events)
	if len(paid) != 1 {
		t.Fatalf("a reply worth %d paid back %d times, want once",
			answered[0].Amount, len(paid))
	}
	if want := answered[0].Amount * 250 / 1000; paid[0].Amount != want {
		t.Errorf("the reply dealt %d and paid back %d, want %d — a quarter of it",
			answered[0].Amount, paid[0].Amount, want)
	}
	// Whose heal it is, which is the half a reply gets wrong most easily: the
	// holder took the health back, not the unit its reply hit.
	if paid[0].Actor != "a" {
		t.Errorf("the drain healed %q, want the trait's holder", paid[0].Actor)
	}
	if paid[0].Drained != 250 {
		t.Errorf("the heal says a share of %d, want the trait's own 250", paid[0].Drained)
	}
}

// TestAReplyWithNoDamageDrainsNothing is the same conservation rule the skill
// path keeps: a drain is a share of damage dealt, so a reply that deals none
// returns none however much its trait drains.
//
// What enforces it is the branch rather than a guard — a status-only reply has
// no power and never reaches the damage half at all. A damage > 0 check beside
// the drain was written first and removed: a mutation deleting it survived
// every test here, because inside that branch there is always damage.
func TestAReplyWithNoDamageDrainsNothing(t *testing.T) {
	events := provoke(t, answering(t, "venom_barb", "", []string{"strike"}, []string{"strike"}), 1000)
	if len(replies(events)) != 0 {
		t.Fatal("a status-only reply dealt damage, so this test measures nothing")
	}
	applied := find(events, battle.StatusApplied)
	if len(applied) != 1 {
		t.Fatalf("the status-only reply applied %d statuses, want one", len(applied))
	}
	if paid := drains(events); len(paid) != 0 {
		t.Errorf("a reply that dealt nothing paid back %+v", paid)
	}
}

// TestAReplyThatKillsStillDrains is the ordering, and it follows the skill path
// rather than being decided here.
//
// resolveAgainst drains from what it dealt whether or not the target fell, so
// lethal damage is worth taking back like any other. A drain placed after the
// kill would make the one blow that matters most the one blow worth nothing.
func TestAReplyThatKillsStillDrains(t *testing.T) {
	fight := answering(t, "barbed", "", []string{"strike"}, []string{"strike"})
	fight.Begin()
	fight.Drain()
	// The holder is hurt so a drain has somewhere to go, and the attacker is
	// left with less health than the reply is worth.
	unitByID(t, fight, "a").HP = 1000
	unitByID(t, fight, "f").HP = 1

	events := []battle.Event{}
	for range 4 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if prompt.Unit == "a" {
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
			fight.Drain()
			continue
		}
		if err := fight.Act("strike", unitByID(t, fight, "a").Cell); err != nil {
			t.Fatalf("strike: %v", err)
		}
		events = fight.Drain()
		break
	}
	if len(find(events, battle.Died)) != 1 {
		t.Fatalf("the reply did not kill, so this test measures nothing:\n%+v", events)
	}
	if paid := drains(events); len(paid) != 1 {
		t.Errorf("a lethal reply paid back %d times, want once", len(paid))
	}
}

// TestAReplysDrainNeverPassesFullHealth is drain's own bound, reached through the
// new path.
//
// The refusal drain makes for a full unit cannot be provoked here and the test
// says so rather than pretending: a reply answers a strike, so by the time it
// pays out its holder has just been hit and there is always room. What is left
// to assert is the bound itself — the payout is capped by the room, never by the
// share alone.
func TestAReplysDrainNeverPassesFullHealth(t *testing.T) {
	fight := answering(t, "barbed", "", []string{"strike"}, []string{"strike"})
	events := provoke(t, fight, 3000)
	if len(replies(events)) != 1 {
		t.Fatal("the trait did not answer, so this test measures nothing")
	}
	holder := unitByID(t, fight, "a")
	if holder.HP >= holder.MaxHP() {
		t.Fatalf("the holder is on %d of %d health, so it was never hurt",
			holder.HP, holder.MaxHP())
	}
	// It was hurt by the strike it answered, so there IS room — the point is
	// that the room is what bounds the payout, and the assertion below is that
	// the log never claims more than landed.
	for _, paid := range drains(events) {
		if paid.Remaining > holder.MaxHP() {
			t.Errorf("a drain left the holder on %d of %d", paid.Remaining, holder.MaxHP())
		}
	}
}

// TestAGatedTraitDrainsOnItsReplyOnlyWhileInForce is the gate covering the whole
// trait, seen from the new path: a gate takes the reply and the drain together,
// because a trait wanting one gated half is two traits.
func TestAGatedTraitDrainsOnItsReplyOnlyWhileInForce(t *testing.T) {
	// cornered_spikes replies and does not drain, so the ungated pairing above
	// is what this contrasts with: what is measured here is that a trait out of
	// force answers nothing at all, drain included.
	fight := answering(t, "cornered_spikes", "", []string{"strike"}, []string{"strike"})
	events := provoke(t, fight, 3000)
	if len(replies(events)) != 0 {
		t.Error("a gated trait answered while its holder was at full health")
	}
	if paid := drains(events); len(paid) != 0 {
		t.Errorf("a gated trait paid back %+v while out of force", paid)
	}
}
