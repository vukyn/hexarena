package socket

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// theLoopbackAddress is the address a room in these in-process tests is opened
// behind. Nothing is bound: a code is arithmetic over an address, and the
// registry has no I/O.
var theLoopbackAddress = netip.AddrPortFrom(netip.AddrFrom4([4]byte{127, 0, 0, 1}), 7100)

// TestADivergenceIsLoudOnTheTurnItHappens is the mirror's whole promise made
// mechanical, and it fails the other way round from most tests here: the thing
// being measured is that a **failure** is reported, on time.
//
// ⚠️ It drives a bare room.Registry rather than a socket, deliberately. The
// claim is about the Mirror's digest comparison and nothing else, so a
// connection in the middle would be a slower run of the same assertion with a
// second thing able to break it. Every other test in this file goes over a real
// listener because the claim there genuinely involves one.
//
// ⚠️ **The divergence is a real one**, not a doctored hash. One client is handed
// a decision that names the same unit and a **different legal skill**, so its
// engine resolves a different turn and produces different events — which is the
// thing the digest exists to catch. Flipping a byte of the digest would measure
// the comparison and say nothing about whether two battles that had genuinely
// parted company would be noticed.
//
// The claim is *on the turn it happens*: every turn before the doctored one is
// applied without complaint, and the error names that turn.
func TestADivergenceIsLoudOnTheTurnItHappens(t *testing.T) {
	dependencies := deps(t)
	// A bo1, because one battle is enough to part company in and a series would
	// only add turns.
	configuration := config(11, 1, room.DefaultAllowance)
	rooms := room.NewRegistry()
	t.Cleanup(func() {
		rooms.CloseAll()
		rooms.Wait()
	})
	code, err := rooms.Open(theLoopbackAddress, configuration, dependencies)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}

	host := NewMirror(wire.SeatHost, dependencies.Books)
	guest := NewMirror(wire.SeatGuest, dependencies.Books)
	mirrors := map[wire.Seat]*Mirror{wire.SeatHost: host, wire.SeatGuest: guest}

	// Both clients join, and the second join is what opens the first battle.
	answered, err := rooms.Join(code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""))
	if err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	deliverTo(t, mirrors, answered.Out)
	answered, err = rooms.Join(code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	deliverTo(t, mirrors, answered.Out)

	// ⚠️ The third turn, not the first: what is being held is that the check
	// fires **on the turn it happens** rather than at the end, so there have to
	// be clean turns in front of it for the failure to be distinguishable from
	// "this never worked".
	const divergeOn = 3
	clean, reading := 0, answered.Reading
	var diverged *Divergence
	for reading.Waiting && diverged == nil {
		acting := mirrors[reading.Awaiting]
		if acting == nil {
			t.Fatalf("the room waits on %q, which is not a seat", reading.Awaiting)
		}
		answer, decided := acting.Decide(dueRating(acting))
		if !decided {
			t.Fatalf("%s was asked to act and decided nothing", reading.Awaiting)
		}
		answered, err = rooms.Deliver(code, reading.Awaiting, answer)
		if err != nil {
			t.Fatalf("decision %d from %s: %v", clean+1, reading.Awaiting, err)
		}
		reading = answered.Reading

		for _, message := range answered.Out {
			turn, carried := message.Body.(wire.Turn)
			if !carried {
				continue
			}
			// The guest always gets the honest turn, so a failure below is one
			// client's and not the protocol's.
			if message.To == wire.SeatGuest {
				if err := guest.Receive(turn); err != nil {
					t.Fatalf("the guest applied an honest turn and refused it: %v", err)
				}
				continue
			}
			doctored, twisted := twist(host, turn, clean+1 >= divergeOn)
			err := host.Receive(doctored)
			switch {
			case !twisted && err != nil:
				t.Fatalf("the host applied an honest turn %d and refused it: %v", turn.Decision.Turn, err)
			case !twisted:
				clean++
			case err == nil:
				t.Fatalf("the host was handed %q's turn %d with %q substituted for %q and reported nothing",
					doctored.Decision.Unit, doctored.Decision.Turn, doctored.Decision.Skill, turn.Decision.Skill)
			case !errors.As(err, &diverged):
				t.Fatalf("the host refused a doctored turn with %v, which is not a *Divergence", err)
			default:
				// Recorded for the assertions below, against the honest turn.
				assertDivergence(t, diverged, turn, clean)
			}
		}
	}
	if diverged == nil {
		t.Fatalf("the match ran to a verdict over %d clean turns and no divergence was ever reported", clean)
	}
}

// assertDivergence is what the report has to say, and every clause of it is
// something a bug report needs: which turn, whose, and the two digests.
func assertDivergence(t *testing.T, diverged *Divergence, honest wire.Turn, clean int) {
	t.Helper()
	if clean < 2 {
		t.Errorf("the divergence was reported after only %d clean turns, so it does not show the check fires late as well as early", clean)
	}
	if diverged.Turn != honest.Decision.Turn {
		t.Errorf("the divergence names turn %d and it happened on turn %d", diverged.Turn, honest.Decision.Turn)
	}
	if diverged.Unit != honest.Decision.Unit {
		t.Errorf("the divergence names %q and the turn was %q's", diverged.Unit, honest.Decision.Unit)
	}
	if diverged.Seat != wire.SeatHost {
		t.Errorf("the divergence names the %q seat and the host is what diverged", diverged.Seat)
	}
	if diverged.Room != honest.Events {
		t.Errorf("the divergence reports the room's digest as %s and the turn carried %s",
			diverged.Room.Short(), honest.Events.Short())
	}
	if diverged.Client == honest.Events {
		t.Error("the divergence reports the same digest for both sides, so nothing was compared")
	}
	if diverged.Battle != 1 {
		t.Errorf("the divergence names battle %d of a bo1", diverged.Battle)
	}
	t.Logf("caught on turn %d of battle %d after %d clean turns: %v",
		diverged.Turn, diverged.Battle, clean, diverged)
}

// twist substitutes a different **legal** skill for the one that was taken, on
// the same unit, so that the client's own engine resolves a genuinely different
// turn.
//
// It reads the legal moves off the mirror's own open prompt, which is the prompt
// for exactly this decision — so the substituted skill is one the engine will
// accept, and the divergence is a board that parted company rather than a replay
// that failed.
func twist(client *Mirror, turn wire.Turn, wanted bool) (wire.Turn, bool) {
	if !wanted || turn.Decision.Passed {
		return turn, false
	}
	fight := client.Battle()
	if fight == nil {
		return turn, false
	}
	prompt := fight.Pending()
	if prompt == nil || prompt.Unit != turn.Decision.Unit {
		return turn, false
	}
	for _, option := range prompt.Options {
		if option.Skill == turn.Decision.Skill || !option.Available() {
			continue
		}
		doctored := turn
		doctored.Decision.Skill = option.Skill
		doctored.Decision.Aim = hex.At(option.Aims[0])
		return doctored, true
	}
	return turn, false
}

// deliverTo hands every outbound message to the mirror it names.
func deliverTo(t *testing.T, mirrors map[wire.Seat]*Mirror, out []room.Outbound) {
	t.Helper()
	for _, message := range out {
		client, seated := mirrors[message.To]
		if !seated {
			t.Fatalf("the room addressed a %s to %q, which is not a seat", message.Body.Kind(), message.To)
		}
		if err := client.Receive(message.Body); err != nil {
			t.Fatalf("%s receives a %s: %v", message.To, message.Body.Kind(), err)
		}
	}
}

// dueRating is a mirror answering off its own battle, for the in-process driver.
func dueRating(client *Mirror) battle.Chooser {
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		fight := client.Battle()
		if fight == nil {
			return battle.Choice{}, false
		}
		return fight.Suggest(prompt)
	}
}
