package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTwoFakeClientsFightAWholeBo3InProcess is the headline, and it is the test
// the whole package was shaped to make possible: the room is a state machine
// over messages with no I/O in it, so two clients drive a complete match with no
// sockets, no goroutines and no clock, at the speed of the engine.
//
// ⚠️ It asserts something real rather than "it finished", which is the failure
// mode an end-to-end test of this shape falls into. Four claims, and each of
// them is a thing that could be quietly wrong while a match still ran to
// completion:
//
//  1. **The events the room produced are the events each client produced**, per
//     turn, checked by comparing the digest on wire.Turn against the digest of
//     what the client's own engine emitted. That is the mirror's whole promise
//     and it is checked on every turn rather than at the end.
//  2. **The seat the room asks is the seat whose own battle says it is that
//     side's turn** — two independent readings of whose turn it is, one from the
//     room's seat map and one from the client's engine. Asserted inside
//     mirror.answer, so it fires on every turn of every battle.
//  3. **The series ended by the stated rule**, re-derived here from the battles
//     the room recorded rather than read off the verdict the room wrote.
//  4. **Each seat was home the alternation says it was**, because home is the
//     sixty-point fact: the home squad is enlisted first and therefore wins a
//     speed tie.
func TestTwoFakeClientsFightAWholeBo3InProcess(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3)
	opened := newRoom(t, configuration)
	// A replay limit generous enough for one decision plus whatever run of
	// skipped turns follows it. It is the client's own number: nothing on the
	// wire carries it.
	clients := newTable(t, dependencies, configuration.TurnCap)

	hostSquad := squadOf(t, dependencies.Characters, "host.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	guestSquad := squadOf(t, dependencies.Characters, "guest.squad",
		"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa")

	seat, out, err := opened.Join(hello(t, hostSquad, "Host"))
	if err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	if seat != wire.SeatHost {
		t.Fatalf("the first peer took the %q seat, want the host's", seat)
	}
	// One message and no battle: a room with one player in it is a room waiting.
	if len(out) != 1 {
		t.Fatalf("the first join produced %d messages, want only a welcome", len(out))
	}
	clients.deliver(t, out)
	if _, waiting := opened.Awaiting(); waiting {
		t.Error("a room with one player is waiting on a seat to act")
	}

	seat, out, err = opened.Join(hello(t, guestSquad, "Guest"))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	if seat != wire.SeatGuest {
		t.Fatalf("the second peer took the %q seat, want the guest's", seat)
	}
	clients.deliver(t, out)

	// Both clients know the series they are in, from wire.Welcome and nothing
	// else: there is deliberately no series-standing message. It is compared
	// against the room's **own** reading of its configuration rather than
	// against the local value, which is the stronger claim: the allowance a
	// client counts down and the allowance the transport reads off the room have
	// to be one number, or a countdown would disagree with the forfeit rule.
	held := opened.Config()
	if held != configuration {
		t.Errorf("the room holds a configuration that is not the one it was opened with")
	}
	for _, client := range []*mirror{clients.host, clients.guest} {
		if client.welcome.Battles != held.Battles {
			t.Errorf("%s was told the series is %d battles, want %d",
				client.seat, client.welcome.Battles, held.Battles)
		}
		if client.welcome.Allowance != held.Allowance {
			t.Errorf("%s was told the allowance is %ds, want %ds",
				client.seat, client.welcome.Allowance, held.Allowance)
		}
		if client.welcome.Format != held.Format {
			t.Errorf("%s was told the format is %s, want %s",
				client.seat, client.welcome.Format, held.Format)
		}
		// ⚠️ The turn cap is a room setting a client needs in order to behave
		// correctly, exactly as the allowance is: a capped battle emits no Ended
		// and no further start arrives, so a mirror that did not hold the cap
		// would sit on an open prompt for ever. It is compared against the
		// room's own reading of its configuration for the reason the allowance
		// is — the two have to be one number, or the two peers stop on different
		// turns.
		if client.welcome.TurnCap != held.TurnCap {
			t.Errorf("%s was told the turn cap is %d, want %d",
				client.seat, client.welcome.TurnCap, held.TurnCap)
		}
	}

	// The match. Every step is one decision, read off the acting client's own
	// mirror, so the whole thing is deterministic.
	steps := 0
	for !opened.Finished() {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d decisions the room is waiting on nobody and the match is not over", steps)
		}
		client := clients.at(onTurn)
		answered, err := opened.Deliver(onTurn, client.answer())
		if err != nil {
			t.Fatalf("decision %d from %s: %v", steps, onTurn, err)
		}
		clients.deliver(t, answered)
		steps++
		// Three battles of at most a cap each, plus room for the joins. A
		// backstop rather than an expectation: without it a state machine that
		// stopped making progress would hang the suite instead of failing it.
		if steps > 3*configuration.TurnCap {
			t.Fatalf("the match took more than %d decisions, so something is not progressing", steps)
		}
	}

	played := opened.Played()
	if len(played) == 0 {
		t.Fatal("the match finished having played no battles")
	}
	if len(played) > configuration.Battles {
		t.Fatalf("a bo%d played %d battles", configuration.Battles, len(played))
	}

	// Claim 3: the series ended by the rule, re-derived here. A seat holding
	// more than half the series ends it early; otherwise every battle is
	// fought.
	standing := map[wire.Seat]int{}
	for _, one := range played {
		if one.Winner.Valid() {
			standing[one.Winner]++
		}
	}
	leading := 0
	for _, won := range standing {
		if won > leading {
			leading = won
		}
	}
	decisive := configuration.Battles/2 + 1
	switch {
	case leading >= decisive:
		if len(played) != indexOfDecider(played, decisive) {
			t.Errorf("a seat reached %d wins at battle %d and the series ran to %d battles",
				decisive, indexOfDecider(played, decisive), len(played))
		}
	default:
		if len(played) != configuration.Battles {
			t.Errorf("no seat took %d battles and the series stopped after %d of %d",
				decisive, len(played), configuration.Battles)
		}
	}
	// And the verdict says the same thing the standing does.
	result := opened.Result()
	switch result.Verdict {
	case room.VerdictWon:
		if standing[result.Winner] != leading || leading < decisive {
			t.Errorf("the room declared %s the winner on a standing of %v", result.Winner, standing)
		}
	case room.VerdictDrawn:
		if result.Winner.Valid() {
			t.Errorf("a drawn match names %s as its winner", result.Winner)
		}
	default:
		t.Errorf("a match played out to the end has the verdict %q", result.Verdict)
	}
	if result.Departed.Valid() {
		t.Errorf("a match played out to the end records %q as having gone away", result.Departed)
	}

	// Claim 4: each battle was fought from the side the alternation gives it,
	// and from the seed the derivation gives it. The first two are written out
	// literally rather than asked of HomeFor, because a test that asked the rule
	// what the rule says would agree with any rule at all.
	if played[0].Home != wire.SeatHost {
		t.Errorf("battle 1 was home to %s, want the host", played[0].Home)
	}
	if len(played) > 1 && played[1].Home != wire.SeatGuest {
		t.Errorf("battle 2 was home to %s, want the guest — battles alternate", played[1].Home)
	}
	for index, one := range played {
		if want := configuration.HomeFor(index + 1); one.Home != want {
			t.Errorf("battle %d was home to %s, want %s", index+1, one.Home, want)
		}
		if want := configuration.SeedFor(index + 1); one.Seed != want {
			t.Errorf("battle %d was fought from seed %d, want %d derived from the room's %d",
				index+1, one.Seed, want, configuration.Seed)
		}
		if one.Turns <= 0 {
			t.Errorf("battle %d recorded %d turns", index+1, one.Turns)
		}
		if one.Capped {
			t.Errorf("battle %d hit the %d-turn cap, so this is not measuring a battle that ended itself",
				index+1, configuration.TurnCap)
		}
	}

	// Claim 1's vacuity guard. A match of the shipped 3v3 takes 34 to 55
	// decisions a battle, so a run that compared a handful of digests did not
	// fight a match and the assertion inside mirror.apply measured almost
	// nothing.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if client.compared != steps {
			t.Errorf("%s checked %d digests over %d decisions; every turn goes to both clients",
				client.seat, client.compared, steps)
		}
		if client.compared < 30 {
			t.Errorf("%s checked only %d digests, which is too few for a real match", client.seat, client.compared)
		}
		if len(client.starts) != len(played) {
			t.Errorf("%s was started %d times over %d battles", client.seat, len(client.starts), len(played))
		}
	}

	// Both mirrors of the last battle are the same battle, which is the property
	// the whole design rests on: same seed, same roster, same events.
	hostDigest, err := wire.DigestEvents(clients.host.events)
	if err != nil {
		t.Fatalf("digest the host's events: %v", err)
	}
	guestDigest, err := wire.DigestEvents(clients.guest.events)
	if err != nil {
		t.Fatalf("digest the guest's events: %v", err)
	}
	if hostDigest != guestDigest {
		t.Errorf("the two clients' mirrors of the last battle differ: %s against %s",
			hostDigest.Short(), guestDigest.Short())
	}

	// And the client learns the battle's outcome from its **own** Ended event
	// rather than from a standing message, so that outcome had better be the one
	// the room recorded.
	last := played[len(played)-1]
	closing := lastEvent(t, clients.host.events, battle.Ended)
	if closing.Outcome != last.Outcome {
		t.Errorf("the host's own battle ended %q while the room recorded %q",
			closing.Outcome, last.Outcome)
	}
	if closing.Side.Fights() != last.Winner.Valid() {
		t.Errorf("the host's own battle names %s the winner while the room recorded %q",
			closing.Side, last.Winner)
	}

	t.Logf("bo%d over %d battles in %d decisions; standing %v, verdict %q, %d prompts skipped",
		configuration.Battles, len(played), steps, standing, result.Verdict, opened.Skipped())
}

// indexOfDecider is the battle at which a seat first reached the winning number
// of wins, which is where the series should have stopped.
func indexOfDecider(played []room.BattleResult, decisive int) int {
	standing := map[wire.Seat]int{}
	for index, one := range played {
		if !one.Winner.Valid() {
			continue
		}
		standing[one.Winner]++
		if standing[one.Winner] >= decisive {
			return index + 1
		}
	}
	return len(played)
}

// lastEvent is the final event of a kind, for a test reading the closing line of
// a battle out of a client's own record.
func lastEvent(t *testing.T, events []battle.Event, kind battle.Kind) battle.Event {
	t.Helper()
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].Kind == kind {
			return events[index]
		}
	}
	t.Fatalf("no %s event in a run of %d", kind, len(events))
	return battle.Event{}
}
