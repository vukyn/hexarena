package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
)

// A finished PvP battle written out as a `battle.Log`, and the only claim worth
// making about one: **it re-runs**.
//
// The room composes the log and writes nothing — it has no filesystem — so what
// is asserted here is the value on BattleResult. Where it goes afterwards is
// `cmd/hexarena-host`'s question.

// TestEveryBattleOfAMatchReplaysFromItsOwnLog is `hexarena --replay --verify`
// applied to a room's record, and it is deliberately the same arithmetic that
// command uses rather than a weaker one: build a battle from the log's seed and
// the log's roster, replay the log's choices, and compare **every event in
// order** against the log's own.
//
// ⚠️ **From the log's roster and not from the squads**, for the reason the
// command gives: a placement is a choice now, so a log without one cannot be
// re-run at all and a log re-run against anything else is comparing two
// different fights.
func TestEveryBattleOfAMatchReplaysFromItsOwnLog(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3)
	opened := playedOut(t, dependencies, configuration)

	played := opened.Played()
	if len(played) == 0 {
		t.Fatal("the match recorded no battles, so there is no log to re-run")
	}
	for _, fought := range played {
		log := fought.Log
		if !log.Replayable() {
			t.Errorf("battle %d records no placement, so nothing could re-run it", fought.Battle)
			continue
		}
		if log.Seed != fought.Seed {
			t.Errorf("battle %d was fought from seed %d and its log says %d",
				fought.Battle, fought.Seed, log.Seed)
		}
		if len(log.Events) == 0 || len(log.Choices) == 0 {
			t.Errorf("battle %d logs %d events and %d choices",
				fought.Battle, len(log.Events), len(log.Choices))
			continue
		}
		rerun := replayed(t, dependencies, log, configuration.TurnCap)
		if len(rerun) != len(log.Events) {
			t.Errorf("battle %d logs %d events and re-running produced %d",
				fought.Battle, len(log.Events), len(rerun))
			continue
		}
		for i := range rerun {
			if rerun[i] != log.Events[i] {
				t.Errorf("battle %d, event %d differs from the log:\nlogged %+v\nre-ran %+v",
					fought.Battle, i, log.Events[i], rerun[i])
				break
			}
		}
		t.Logf("battle %d: %d events and %d choices re-ran exactly from seed %d%s",
			fought.Battle, len(log.Events), len(log.Choices), log.Seed,
			cappedIn(fought))
	}
}

// TestALoggedBattleStartsAtTheOpeningBoard is the half the round trip above
// cannot fail on and the room could still get wrong.
//
// ⚠️ **The room's own cursor does not start at nought.** It is set to Recorded()
// after the opening, because no wire.Turn carries the opening board and a mirror
// produces it by calling Begin itself. A log written from that cursor would
// re-run to a longer event list than it recorded and fail on the count — which
// is exactly what the test above would report, and it would report it as
// "the events differ" rather than as "the log starts in the wrong place". This
// says which.
func TestALoggedBattleStartsAtTheOpeningBoard(t *testing.T) {
	dependencies := deps(t)
	opened := playedOut(t, dependencies, config(11, 1))
	played := opened.Played()
	if len(played) != 1 {
		t.Fatalf("a bo1 played %d battles", len(played))
	}
	events := played[0].Log.Events
	if len(events) == 0 {
		t.Fatal("the battle logged no events at all")
	}
	// The opening board is what a battle emits before anybody is asked anything,
	// so the first event a log carries is one no decision produced.
	if first := events[0]; first.Turn != 0 {
		t.Errorf("the log opens on %+v, which belongs to turn %d: the record starts after "+
			"the opening board rather than at it", first, first.Turn)
	}
}

// TestAnAbandonedBattleIsLoggedByNobody holds the one battle that gets no log,
// and holds it as a property of the result rather than of the log: only a battle
// that CLOSED produces a BattleResult, and a departure closes nothing.
//
// A log of an interrupted fight would re-run past the point it stopped, because
// the engine concluded nothing about it and Replay would carry on with the
// fallback — so the honest record is no record.
func TestAnAbandonedBattleIsLoggedByNobody(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3)
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)
	seat(t, opened, clients, dependencies, "host.squad", "Host")
	seat(t, opened, clients, dependencies, "guest.squad", "Guest")

	// One turn taken, so the battle is genuinely under way and the room holds a
	// part-written record — the state a naive writer would flush.
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatal("the room is waiting on nobody, so no battle is under way")
	}
	answered, err := opened.Deliver(onTurn, clients.at(onTurn).answer())
	if err != nil {
		t.Fatalf("the first turn: %v", err)
	}
	clients.deliver(t, answered)

	if _, err := opened.Left(onTurn); err != nil {
		t.Fatalf("%s leaves: %v", onTurn, err)
	}
	if !opened.Finished() {
		t.Fatal("a departure left the match running")
	}
	if opened.Result().Verdict != room.VerdictAbandoned {
		t.Fatalf("the match ended %q rather than abandoned", opened.Result().Verdict)
	}
	if played := opened.Played(); len(played) != 0 {
		t.Errorf("an abandoned match recorded %d battle(s), and the one it interrupted "+
			"concluded nothing: %+v", len(played), played)
	}
}

// playedOut opens a room, seats two squads and plays the whole match out.
func playedOut(t *testing.T, dependencies room.Deps, configuration room.Config) *room.Room {
	t.Helper()
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)
	seat(t, opened, clients, dependencies, "host.squad", "Host")
	seat(t, opened, clients, dependencies, "guest.squad", "Guest")
	steps := 0
	for !opened.Finished() {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d turns the room is waiting on nobody and the match is not over",
				steps)
		}
		answered, err := opened.Deliver(onTurn, clients.at(onTurn).answer())
		if err != nil {
			t.Fatalf("turn %d from %s: %v", steps, onTurn, err)
		}
		clients.deliver(t, answered)
		steps++
		if steps > configuration.TurnCap*configuration.Battles {
			t.Fatalf("the match took more than %d decisions, so something is not progressing",
				steps)
		}
	}
	return opened
}

// seat joins one player and hands the room's messages to the fake clients.
func seat(t *testing.T, opened *room.Room, clients *table, dependencies room.Deps, id, name string) {
	t.Helper()
	var members []string
	switch id {
	case "host.squad":
		members = []string{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"}
	default:
		members = []string{"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"}
	}
	squad := squadOf(t, dependencies.Characters, id, members...)
	_, out, err := opened.Join(hello(t, squad, name))
	if err != nil {
		t.Fatalf("%s joins: %v", name, err)
	}
	clients.deliver(t, out)
}

// replayed re-runs a log the way `hexarena --verify` does and returns the events
// the re-run produced.
func replayed(t *testing.T, dependencies room.Deps, log battle.Log, limit int) []battle.Event {
	t.Helper()
	fight, err := battle.New(dependencies.Books, log.Seed, log.Roster)
	if err != nil {
		t.Fatalf("open the battle from its log: %v", err)
	}
	fight.Begin()
	if _, _, err := fight.Replay(log.Choices, limit, nil); err != nil {
		t.Fatalf("re-run the battle: %v", err)
	}
	return fight.Drain()
}

// cappedIn is a note for the log line, because a capped battle re-running is the
// interesting case rather than the ordinary one: its log has no Ended event.
func cappedIn(fought room.BattleResult) string {
	if fought.Capped {
		return " (capped, so the log carries no ending)"
	}
	return ""
}
