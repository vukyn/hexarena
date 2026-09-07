package room_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// The gate's half of a watcher, and the registry's: step 2 gave the room a
// record nobody could reach, and these are the tests of the two things that
// reach it — a hello that is welcomed without taking a seat, and a read that
// crosses the room's goroutine.
//
// watch_test.go is the record itself and is deliberately separate: what is
// measured there is that a match played with a watcher **reading** is the same
// match, and what is measured here is that a match played with watchers
// **joining** is too. They are different mutations — one takes the players'
// cursor, the other takes a player's seat — and a single test could not fail for
// both reasons.

// TestAWatcherIsWelcomedIntoAFullRoom is the case the seat check would
// otherwise kill, and it is the whole point of the feature: the match worth
// watching is one already being played, and by then both seats are gone.
//
// The player arm is here rather than in gate_test.go on purpose. A test that
// only showed a watcher being welcomed would pass on a room that welcomed
// everybody, so the same room is shown refusing a *player* with
// wire.CodeRoomFull first — one room, two helloes, opposite answers, and the
// only difference between them is the flag.
func TestAWatcherIsWelcomedIntoAFullRoom(t *testing.T) {
	dependencies := deps(t)
	opened, _ := openMatch(t, watchable(config(11, 1)))
	if _, waiting := opened.Awaiting(); !waiting {
		t.Fatal("the room is not waiting on anybody, so the match is not under way and this measures nothing")
	}
	third := squadOf(t, dependencies.Characters, "third.squad",
		"pokemon.mew", "pokemon.poliwag", "pokemon.magnemite")

	// The same room, to a player.
	refused, turnedAway, err := opened.Join(hello(t, third, "Third"))
	if err != nil {
		t.Fatalf("a third player joins: %v", err)
	}
	if refused.Seat.Valid() || refused.Watching {
		t.Fatalf("a third player was admitted as %+v", refused)
	}
	if code := onlyCode(t, turnedAway); code != wire.CodeRoomFull {
		t.Fatalf("a third player was answered %q, want %q — without that this test cannot tell "+
			"a room that admits watchers from a room that admits anybody", code, wire.CodeRoomFull)
	}

	// The same room, to a watcher.
	admitted, out, err := opened.Join(watchingHello(t, "Watcher", third))
	if err != nil {
		t.Fatalf("a watcher joins: %v", err)
	}
	if !admitted.Watching {
		t.Fatalf("a watcher of a full room was answered %+v with %v: the seat check must not "+
			"reach a watcher, because the match worth watching is one already being played",
			admitted, out)
	}
	if admitted.Seat.Valid() {
		t.Errorf("a watcher was given the %q seat", admitted.Seat)
	}
	welcome := onlyWelcome(t, out)
	if !welcome.Watching() {
		t.Errorf("a watcher's welcome names the %q seat, so its own client would read it as a player's",
			welcome.Seat)
	}
	if out[0].To.Valid() {
		t.Errorf("a watcher's welcome is addressed to the %q seat, and a watcher has none — an "+
			"Outbound that names no seat is what Server.send answers to the connection it read from",
			out[0].To)
	}
	t.Logf("one room with both seats taken answered a player %q and welcomed a watcher with no seat",
		wire.CodeRoomFull)
}

// TestAWatchingJoinLeavesTheRoomExactlyWhereItWas is the snapshot half of "a
// watcher changes nothing": everything observable about a room, read before a
// watching join and again after it.
//
// ⚠️ **A refusal would pass this too**, which is why it is not the whole claim
// — the other half is TestAMatchPlayedWithWatchersJoiningIsTheSameMatch below,
// where the room has to go on being the same match rather than merely being
// unmoved for one call. What this one adds is the reading a whole-match
// comparison cannot take: the state *at the moment of the join*, so a watcher
// that spent a turn or moved the prompt is caught where it happened rather than
// as a divergence a hundred decisions later.
func TestAWatchingJoinLeavesTheRoomExactlyWhereItWas(t *testing.T) {
	dependencies := deps(t)
	opened, clients := openMatch(t, watchable(config(11, 3)))
	spare := squadOf(t, dependencies.Characters, "spare.squad",
		"pokemon.mew", "pokemon.poliwag", "pokemon.magnemite")

	// A few decisions first, so the room being compared is a room in the middle
	// of a battle rather than one that has only just opened.
	for step := 0; step < 5; step++ {
		answerFor(t, opened, clients, "")
	}

	before := stateOf(t, opened, spare)
	admitted, out, err := opened.Join(watchingHello(t, "Watcher", placement.Squad{}))
	if err != nil {
		t.Fatalf("a watcher joins: %v", err)
	}
	if !admitted.Watching {
		t.Fatalf("the watcher was not welcomed (%+v), so this compares a refusal", admitted)
	}
	if len(out) != 1 {
		t.Fatalf("welcoming a watcher produced %d messages, want the welcome alone: anything else "+
			"is the room telling the players something happened", len(out))
	}
	after := stateOf(t, opened, spare)

	if before != after {
		t.Fatalf("a watching join moved the room:\n before %+v\n after  %+v", before, after)
	}
	// The seat still on turn can still take the turn it was being asked for, and
	// its own mirror still agrees on the digest — which is the assertion that
	// says the prompt is not merely *reported* unmoved but really is.
	onTurn, answered := answerFor(t, opened, clients, "")
	if onTurn != before.awaiting {
		t.Errorf("the seat on turn after the watching join is %q and was %q", onTurn, before.awaiting)
	}
	if len(answered) == 0 {
		t.Error("the turn after a watching join produced nothing at all")
	}
	t.Logf("the room was identical either side of a watching join, still asking %q, with "+
		"%d bodies on the record and a third player still answered %q",
		before.awaiting, before.recorded, before.probe)
}

// roomState is everything about a room a test outside it can see, in one
// comparable value — so "nothing moved" is one comparison rather than eight that
// can each be forgotten.
//
// ⚠️ **The standing is not in it, and cannot be.** Room.Result is the zero
// Result until the match ends, so a per-battle tally is unreadable mid-match;
// what covers it is the whole-match comparison below, which compares the final
// Result and its Wins array. This one is the reading available *at* a join.
type roomState struct {
	awaiting wire.Seat
	waiting  bool
	finished bool
	result   room.Result
	played   int
	skipped  int
	recorded int
	// probe is what the gate answers a **player**, which is how the two seats
	// are read from outside a room that exposes no seat count. It is only
	// meaningful on a full room — on a room with a seat free the probe would
	// take it — so stateOf is used on full rooms and says so here rather than in
	// each caller.
	probe wire.Code
}

func stateOf(t *testing.T, opened *room.Room, spare placement.Squad) roomState {
	t.Helper()
	awaiting, waiting := opened.Awaiting()
	recorded, _ := opened.Since(0)
	admitted, out, err := opened.Join(hello(t, spare, "Probe"))
	if err != nil {
		t.Fatalf("probing the seats: %v", err)
	}
	if admitted.Seat.Valid() {
		t.Fatalf("the probe took the %q seat, so this room was not full and the reading is worthless",
			admitted.Seat)
	}
	return roomState{
		awaiting: awaiting,
		waiting:  waiting,
		finished: opened.Finished(),
		result:   opened.Result(),
		played:   len(opened.Played()),
		skipped:  opened.Skipped(),
		recorded: len(recorded),
		probe:    onlyCode(t, out),
	}
}

// TestAMatchPlayedWithWatchersJoiningIsTheSameMatch is the gate-level twin of
// TestAMatchPlayedWithAWatcherReadingIsTheSameMatch, and it is a second test
// rather than a second assertion in that one because the two mutations are
// different: there a watcher takes the players' **cursor**, here it takes a
// player's **seat**.
//
// ⚠️ **A watcher that filled a seat would pass almost everything else.** The
// roster would still be legal, both peers would still agree on every digest, and
// the match would still reach a verdict — it would simply be a different match,
// because the order the seats are visited in reaches the roster and the roster's
// order decides which side wins a speed tie. Only a comparison against the same
// seed played with no watcher anywhere near it can see that, which is this.
func TestAMatchPlayedWithWatchersJoiningIsTheSameMatch(t *testing.T) {
	joined := playWithWatchersJoining(t, true)
	alone := playWithWatchersJoining(t, false)

	if joined.welcomed == 0 {
		t.Fatal("the run that was supposed to admit watchers admitted none, so the two runs are the same run twice")
	}
	if alone.welcomed != 0 {
		t.Fatalf("the unwatched run admitted %d watchers", alone.welcomed)
	}
	if joined.compared == 0 || alone.compared == 0 {
		t.Fatalf("the clients checked %d and %d digests, so at least one match ran with no mirror",
			joined.compared, alone.compared)
	}

	if len(joined.turns) != len(alone.turns) {
		t.Fatalf("the watched match played %d turns and the unwatched one %d",
			len(joined.turns), len(alone.turns))
	}
	for index := range joined.turns {
		with, without := joined.turns[index], alone.turns[index]
		if with.Decision != without.Decision {
			t.Fatalf("turn %d differs: watched %+v, unwatched %+v", index, with.Decision, without.Decision)
		}
		if with.Events != without.Events {
			t.Fatalf("turn %d of %q digests %s with watchers and %s without any",
				index, with.Decision.Unit, with.Events.Short(), without.Events.Short())
		}
	}
	if joined.result != alone.result {
		t.Errorf("the watched match ended %+v and the unwatched one %+v", joined.result, alone.result)
	}
	if !reflect.DeepEqual(joined.played, alone.played) {
		t.Errorf("the watched match played %+v and the unwatched one %+v", joined.played, alone.played)
	}
	if joined.skipped != alone.skipped {
		t.Errorf("the watched match skipped %d prompts and the unwatched one %d",
			joined.skipped, alone.skipped)
	}
	t.Logf("%d turns, %d battles and %d skipped prompts, identical with %d watchers welcomed and with none",
		len(joined.turns), len(joined.played), joined.skipped, joined.welcomed)
}

// playedThrough is what the two runs of the comparison above are compared on.
type playedThrough struct {
	turns    []wire.Turn
	result   room.Result
	played   []room.BattleResult
	skipped  int
	welcomed int
	compared int
}

// playWithWatchersJoining plays a bo3 from one seed with both clients answering
// off their own mirrors, admitting a watcher before every single decision when
// asked to.
//
// ⚠️ **Before every decision rather than once at the start**, because a watcher
// joining a room that is between turns and a watcher joining a room that has an
// open prompt are different moments, and the second is the one where taking a
// seat or spending a turn would show.
func playWithWatchersJoining(t *testing.T, admitting bool) playedThrough {
	t.Helper()
	configuration := watchable(config(11, 3))
	opened, clients := openMatch(t, configuration)

	out := playedThrough{}
	steps := 0
	for !opened.Finished() {
		if admitting {
			admitted, answered, err := opened.Join(
				watchingHello(t, fmt.Sprintf("Watcher %d", steps), placement.Squad{}))
			if err != nil {
				t.Fatalf("a watcher joins before decision %d: %v", steps, err)
			}
			if !admitted.Watching || admitted.Seat.Valid() {
				t.Fatalf("the watcher before decision %d was admitted as %+v with %v",
					steps, admitted, answered)
			}
			out.welcomed++
		}
		_, answered := answerFor(t, opened, clients, "")
		for _, message := range answered {
			// One seat's copy: both are sent the same body by the same call.
			if turn, isTurn := message.Body.(wire.Turn); isTurn && message.To == wire.SeatHost {
				out.turns = append(out.turns, turn)
			}
		}
		steps++
		// A backstop rather than an expectation: without it a state machine that
		// stopped making progress would hang the suite instead of failing it.
		if steps > 3*configuration.TurnCap {
			t.Fatalf("the match took more than %d decisions, so something is not progressing", steps)
		}
	}
	out.result = opened.Result()
	out.played = opened.Played()
	out.skipped = opened.Skipped()
	out.compared = clients.host.compared + clients.guest.compared
	return out
}

// TestAWatchersWelcomeIsAPlayersWithTheSeatTakenOut is the shape of the answer,
// and it is asserted against a **player's** welcome from the same room rather
// than against a written-down list.
//
// ⚠️ **A list is what a zero Welcome passes.** Asserting "the seat is empty and
// Watching() is true" is satisfied by a welcome carrying nothing at all, and a
// watcher handed one would build a mirror with no turn cap — which is a client
// sitting on an open prompt for a battle the room stopped asking about. So the
// configuration is compared field by field against the room's own Config, and
// then the two welcomes are compared to each other with the seat put back: the
// only difference a watcher's welcome may have is the seat.
func TestAWatchersWelcomeIsAPlayersWithTheSeatTakenOut(t *testing.T) {
	dependencies := deps(t)
	for _, one := range []struct {
		name          string
		configuration room.Config
		joining       func(*testing.T) wire.Hello
	}{
		{
			name: "a room that takes squads",
			configuration: room.Config{
				Format: wire.Format3v3, Battles: 3, Allowance: 45, Seed: 11, TurnCap: 137,
				Watchable: true,
			},
			joining: func(t *testing.T) wire.Hello {
				return hello(t, squadOf(t, dependencies.Characters, "player.squad",
					"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"), "Player")
			},
		},
		{
			// A drafting room is here for one field, Drafts, which is the only
			// one that is false in every other room: a watcher told nothing
			// about a draft is a watcher that would draw a battle screen over a
			// ban and pick.
			name:          "a room that drafts",
			configuration: watchable(draftingConfig(11)),
			joining:       func(t *testing.T) wire.Hello { return helloWithNoSquad(t, "Player") },
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened := newRoom(t, one.configuration)
			admitted, out, err := opened.Join(watchingHello(t, "Watcher", placement.Squad{}))
			if err != nil {
				t.Fatalf("a watcher joins: %v", err)
			}
			if !admitted.Watching || admitted.Seat.Valid() {
				t.Fatalf("the watcher was admitted as %+v with %v", admitted, out)
			}
			watching := onlyWelcome(t, out)
			if !watching.Watching() {
				t.Errorf("the welcome names the %q seat, so Welcome.Watching reads it as a player's",
					watching.Seat)
			}

			// By value against the room's own configuration, so a zero Welcome
			// fails: every one of these is deliberately not a zero value.
			settings := opened.Config()
			if watching.Format != settings.Format {
				t.Errorf("the welcome says %q and the room is %q", watching.Format, settings.Format)
			}
			if watching.Battles != settings.Battles {
				t.Errorf("the welcome says %d battles and the room plays %d", watching.Battles, settings.Battles)
			}
			if watching.Allowance != settings.Allowance {
				t.Errorf("the welcome says an allowance of %d and the room gives %d",
					watching.Allowance, settings.Allowance)
			}
			if watching.TurnCap != settings.TurnCap {
				t.Errorf("the welcome says a turn cap of %d and the room caps at %d: a watcher "+
					"without it holds an open prompt on a battle the room has stopped asking about",
					watching.TurnCap, settings.TurnCap)
			}
			if watching.Drafts != settings.Drafts {
				t.Errorf("the welcome says drafts=%v and the room is %v", watching.Drafts, settings.Drafts)
			}

			// And against a player of the same room: the seat is the only
			// difference a watcher's welcome is allowed to have.
			seated, playerOut, err := opened.Join(one.joining(t))
			if err != nil {
				t.Fatalf("a player joins: %v", err)
			}
			if !seated.Seat.Valid() {
				t.Fatalf("the player was refused with %v", playerOut)
			}
			played := onlyWelcome(t, playerOut)
			if played.Seat != seated.Seat {
				t.Fatalf("the player's welcome names %q and it took %q", played.Seat, seated.Seat)
			}
			played.Seat = ""
			if watching != played {
				t.Errorf("with the seat taken out, the watcher's welcome is %+v and the player's %+v",
					watching, played)
			}
			t.Logf("%s: a watcher is told %+v, which is the player's welcome with the seat taken out",
				one.name, watching)
		})
	}
}

// TestAWatchingHelloIsRefusedForTheSameThingsAPlayersIs is the half of the gate
// a watcher does **not** skip, and each case is asserted twice — once as a
// player and once as a watcher — because the claim is not merely that a watcher
// is refused but that it is refused *identically*.
//
// The password is the case with an argument behind it rather than an ordering:
// a room's password is what keeps the strangers in the house off the board, and
// watching over somebody's shoulder is exactly what it is for. The version is
// the harder requirement — a watcher reads the same recorded bodies and needs
// the same books to make sense of them, so a peer that cannot speak the protocol
// or holds different data can no more watch than play.
func TestAWatchingHelloIsRefusedForTheSameThingsAPlayersIs(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")

	cases := 0
	for _, one := range []struct {
		name    string
		wrong   func(wire.Hello) wire.Hello
		want    wire.Code
		gateway bool
	}{
		{
			name: "a peer that cannot speak the protocol",
			wrong: func(out wire.Hello) wire.Hello {
				out.Protocol = wire.Protocol + 1
				return out
			},
			want: wire.CodeProtocolMismatch,
		},
		{
			name: "a peer whose data would not simulate the same battle",
			wrong: func(out wire.Hello) wire.Hello {
				out.Data = mangled(out.Data)
				return out
			},
			want: wire.CodeDataMismatch,
		},
		{
			name: "the room's password typed wrong",
			wrong: func(out wire.Hello) wire.Hello {
				out.Password = fixturePassword + "!"
				return out
			},
			want:    wire.CodeBadPassword,
			gateway: true,
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			answers := map[string]wire.Code{}
			for _, who := range []struct {
				what    string
				joining func(*testing.T) wire.Hello
			}{
				{"a player", func(t *testing.T) wire.Hello { return hello(t, legal, "Player") }},
				{"a watcher", func(t *testing.T) wire.Hello { return watchingHello(t, "Watcher", legal) }},
			} {
				// Watchable, so the fault under test is the ONLY thing wrong with
				// the watching hello. A room that refused every watcher would
				// leave the watcher arm carrying two faults, and the claim being
				// made — that the two are refused identically — would then rest on
				// the gate's ordering rather than on the refusal under test.
				configuration := watchable(config(7, 1))
				configuration.Password = fixturePassword
				opened := newRoom(t, configuration)
				joining := one.wrong(who.joining(t))
				if !one.gateway {
					// Everything but the password case types it correctly, so
					// the refusal under test is the only thing wrong.
					joining.Password = fixturePassword
				}
				admitted, out, err := opened.Join(joining)
				if err != nil {
					t.Fatalf("%s joins: %v", who.what, err)
				}
				if admitted.Watching {
					t.Errorf("%s was reported as watching while being refused — Admission.Watching "+
						"is what the room DID, not what the hello asked for, and a transport reading "+
						"it as the hello's flag would keep a connection it has just turned away",
						who.what)
				}
				if admitted.Seat.Valid() {
					t.Errorf("%s was given the %q seat while being refused", who.what, admitted.Seat)
				}
				answers[who.what] = onlyCode(t, out)
				cases++
			}
			if answers["a player"] != one.want {
				t.Errorf("a player was answered %q, want %q", answers["a player"], one.want)
			}
			if answers["a watcher"] != answers["a player"] {
				t.Errorf("a watcher was answered %q and a player %q for the same fault: a watcher "+
					"is a client of this room like any other and is turned away in the same words",
					answers["a watcher"], answers["a player"])
			}
		})
	}
	if cases != 6 {
		t.Fatalf("the sweep drove %d joins, want 6: three faults, each as a player and as a watcher", cases)
	}
}

// TestAWatchersSquadIsIgnoredWhateverItBrought is the claim that
// squadIsFieldable is not called on a watcher at all, and **the illegal squad is
// the half that measures it**: a legal one is welcomed either way, so a test
// with only that arm would pass with the five squad rules still running.
//
// The three arms are welcomed **identically**, compared as whole welcomes rather
// than one field each, because "ignored" means the squad reached nothing — not
// that it reached something harmless.
func TestAWatchersSquadIsIgnoredWhateverItBrought(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	// Wrong for the format's size and naming somebody who does not exist: two
	// different rules of the five, so this is not one refusal dodged.
	illegal := legal.Clone()
	illegal.Units = illegal.Units[:2]
	illegal.Units[0].Character = "pokemon.nonesuch"

	for _, kind := range []struct {
		name          string
		configuration room.Config
		// unwanted is what a **player** bringing a squad is answered here, which
		// is the arm that makes the drafting room worth testing at all.
		unwanted wire.Code
	}{
		{name: "a room that takes squads", configuration: watchable(config(11, 1)), unwanted: wire.CodeSquadRefused},
		{name: "a room that drafts", configuration: watchable(draftingConfig(11)), unwanted: wire.CodeSquadUnwanted},
	} {
		t.Run(kind.name, func(t *testing.T) {
			// The room really does refuse this squad from a player, or the
			// comparison below is between two squads it was never going to mind.
			player := newRoom(t, kind.configuration)
			refused, out, err := player.Join(hello(t, illegal, "Player"))
			if err != nil {
				t.Fatalf("a player joins: %v", err)
			}
			if refused.Seat.Valid() {
				t.Fatalf("this room seated a player bringing an illegal squad, so it refuses nothing")
			}
			if code := onlyCode(t, out); code != kind.unwanted {
				t.Fatalf("a player bringing a squad here was answered %q, want %q", code, kind.unwanted)
			}

			welcomes := map[string]wire.Welcome{}
			for _, one := range []struct {
				what  string
				squad placement.Squad
			}{
				{"no squad at all", placement.Squad{}},
				{"a legal squad", legal},
				{"an illegal squad", illegal},
			} {
				opened := newRoom(t, kind.configuration)
				admitted, out, err := opened.Join(watchingHello(t, "Watcher", one.squad))
				if err != nil {
					t.Fatalf("a watcher bringing %s joins: %v", one.what, err)
				}
				if !admitted.Watching {
					t.Fatalf("a watcher bringing %s was admitted as %+v with %v — a watcher expects "+
						"no side of its own, so nothing it brought can fail to appear and there is "+
						"nothing to refuse", one.what, admitted, out)
				}
				welcomes[one.what] = onlyWelcome(t, out)
			}
			for _, what := range []string{"a legal squad", "an illegal squad"} {
				if welcomes[what] != welcomes["no squad at all"] {
					t.Errorf("a watcher bringing %s was welcomed %+v and one bringing nothing %+v: "+
						"the squad is ignored, so it may not reach the answer",
						what, welcomes[what], welcomes["no squad at all"])
				}
			}
			t.Logf("%s: three watchers bringing nothing, a legal squad and an illegal one were "+
				"welcomed identically, where a player bringing one is answered %q",
				kind.name, kind.unwanted)
		})
	}
}

// TestWatchersDoNotStartTheMatch is the other thing the gate must not do with a
// watcher: the second **seat** is what opens the first battle, and a watcher
// takes none, so a room with one player and any number of watchers is still a
// room waiting.
//
// ⚠️ The record is asserted **empty** rather than merely short, because that is
// what makes "no battle started" checkable from outside: begin() is the only
// thing that records a wire.Start, so a record still at nought is the battle
// still unopened.
func TestWatchersDoNotStartTheMatch(t *testing.T) {
	dependencies := deps(t)
	configuration := watchable(config(11, 1))
	opened := newRoom(t, configuration)
	host := squadOf(t, dependencies.Characters, "host.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	guest := squadOf(t, dependencies.Characters, "guest.squad",
		"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa")

	seated, _, err := opened.Join(hello(t, host, "Host"))
	if err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	if !seated.Seat.Valid() {
		t.Fatal("the host was refused an empty room")
	}

	const watchers = 4
	for at := 0; at < watchers; at++ {
		admitted, out, err := opened.Join(
			watchingHello(t, fmt.Sprintf("Watcher %d", at), placement.Squad{}))
		if err != nil {
			t.Fatalf("watcher %d joins: %v", at, err)
		}
		if !admitted.Watching {
			t.Fatalf("watcher %d was admitted as %+v", at, admitted)
		}
		if len(out) != 1 {
			t.Fatalf("watcher %d was answered %d messages, want the welcome alone", at, len(out))
		}
		if _, isWelcome := out[0].Body.(wire.Welcome); !isWelcome {
			t.Fatalf("watcher %d was answered a %s", at, out[0].Body.Kind())
		}
		if onTurn, waiting := opened.Awaiting(); waiting {
			t.Fatalf("after %d watchers the room is waiting on %q: the second SEAT starts the "+
				"match, and a watcher takes none", at+1, onTurn)
		}
		recorded, cursor := opened.Since(0)
		if len(recorded) != 0 || cursor != 0 {
			t.Fatalf("after %d watchers the record holds %d bodies at cursor %d, and no battle "+
				"has begun", at+1, len(recorded), cursor)
		}
	}

	// And the second seat still does start it, so the test cannot pass on a room
	// that had stopped being able to begin at all.
	second, out, err := opened.Join(hello(t, guest, "Guest"))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	// ⚠️ Named before the "did it start" check below, because a watcher that
	// took the seat produces both failures and only this one says which: the
	// guest is turned away from a room that is full of watchers.
	if !second.Seat.Valid() {
		t.Fatalf("the guest was answered %q after %d watchers: the seat it was waiting for is gone, "+
			"so a watcher took one", onlyCode(t, out), watchers)
	}
	if _, waiting := opened.Awaiting(); !waiting {
		t.Fatal("the second seat did not start the match, so this test could not have seen a watcher that did")
	}
	starts := 0
	for _, message := range out {
		if _, isStart := message.Body.(wire.Start); isStart {
			starts++
		}
	}
	if starts != 2 {
		t.Errorf("the second seat produced %d wire.Starts, want one per seat", starts)
	}
	t.Logf("%d watchers left the room waiting with an empty record; the second seat opened the "+
		"battle and produced %d starts", watchers, starts)
}

// TestTheRegistryHandsOutAWatchersRead is the registry's half: Room.Since taken
// **inside** the room's goroutine and copied out, in the shape Registry.Read
// already has.
//
// Four claims, and each has a mutation that reddens it alone:
//
//  1. it answers what Room.Since answers, from nought and from a cursor in the
//     middle;
//  2. the answer is a **copy** — writing into the slice a watcher was handed does
//     not reach the record, which is what deleting the copy in watchedFrom would
//     break and nothing else here would;
//  3. an unknown code is *not known* rather than an empty read, which is the
//     distinction Answer.Known already draws; and
//  4. the read goes through the room's own goroutine, so it stops answering the
//     moment that goroutine does — a Close is what proves the read was never
//     reaching around it.
func TestTheRegistryHandsOutAWatchersRead(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	configuration := config(11, 1)
	code, err := registry.Open(theOneListener, configuration, dependencies)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	defer registry.CloseAll()

	// A watcher of a room nobody has joined: known, and empty. Those two are
	// different answers and this is the one place both are true at once.
	bodies, cursor, known := registry.Since(code, 0)
	if !known {
		t.Fatal("a room that is running was reported unknown")
	}
	if len(bodies) != 0 || cursor != 0 {
		t.Fatalf("a room nobody has joined handed out %d bodies at cursor %d", len(bodies), cursor)
	}

	clients := newTable(t, dependencies, configuration.TurnCap)
	host, guest := squadPair(t, dependencies, 0)
	for _, joining := range []struct {
		squad placement.Squad
		name  string
	}{{host, "Host"}, {guest, "Guest"}} {
		answered, err := registry.Join(code, hello(t, joining.squad, joining.name))
		if err != nil {
			t.Fatalf("%s joins: %v", joining.name, err)
		}
		if !answered.Known || !answered.Seat.Valid() {
			t.Fatalf("%s was not seated: %+v", joining.name, answered)
		}
		clients.deliver(t, answered.Out)
	}

	// The battle is open, so the record holds its wire.Start and nothing else.
	whole, cursor, known := registry.Since(code, 0)
	if !known || len(whole) != 1 {
		t.Fatalf("an open battle recorded %d bodies (known=%v), want the start alone", len(whole), known)
	}
	if _, isStart := whole[0].(wire.Start); !isStart {
		t.Fatalf("the first recorded body is a %s, want a start", whole[0].Kind())
	}

	// ⚠️ The copy, measured by writing into what the registry handed back. With
	// the copy in watchedFrom deleted this is the room's own record being
	// overwritten from another goroutine, and the re-read below finds the nil.
	whole[0] = nil

	// A watcher that keeps its cursor, decision by decision, against one that
	// reads the whole thing at the end.
	incremental := []wire.Body{}
	reading, readable := registry.Read(code)
	if !readable {
		t.Fatal("a room with a battle open is not readable")
	}
	if !reading.Waiting {
		t.Fatal("a room with a battle open is waiting on nobody")
	}
	seatOnTurn := reading.Awaiting
	for step := 0; step < 12; step++ {
		acted, actErr := registry.Deliver(code, seatOnTurn, clients.at(seatOnTurn).answer())
		if actErr != nil {
			t.Fatalf("decision %d from %s: %v", step, seatOnTurn, actErr)
		}
		if !acted.Known {
			t.Fatalf("the room went away after %d decisions", step)
		}
		clients.deliver(t, acted.Out)
		taken, next, ok := registry.Since(code, cursor)
		if !ok {
			t.Fatalf("the record was unreadable after decision %d", step)
		}
		incremental = append(incremental, taken...)
		cursor = next
		if acted.Reading.Finished {
			t.Fatalf("the match finished after %d decisions, and this test needs it running", step)
		}
		seatOnTurn = acted.Reading.Awaiting
	}
	if len(incremental) == 0 {
		t.Fatal("twelve decisions handed a watcher nothing, so the incremental read measures nothing")
	}

	// The whole record from nought is the start plus everything the incremental
	// reads were handed, in order — and the start is not the nil written above,
	// which is claim 2.
	again, next, ok := registry.Since(code, 0)
	if !ok {
		t.Fatal("the record was unreadable from nought")
	}
	if next != cursor {
		t.Errorf("a read from nought ends at cursor %d and the incremental reader is at %d", next, cursor)
	}
	if len(again) == 0 || again[0] == nil {
		t.Fatalf("the first recorded body came back %v: writing into a watcher's copy reached the "+
			"room's own record, so the answer is a view rather than a copy", again[0])
	}
	if _, isStart := again[0].(wire.Start); !isStart {
		t.Fatalf("the first recorded body is a %s, want the start", again[0].Kind())
	}
	if !reflect.DeepEqual(again[1:], incremental) {
		t.Errorf("a read from nought holds %d bodies after the start and the incremental reads were "+
			"handed %d, and they differ", len(again)-1, len(incremental))
	}

	// From a cursor in the middle, which is a watcher that fell behind.
	middle := len(again) / 2
	behind, endsAt, ok := registry.Since(code, middle)
	if !ok {
		t.Fatal("the record was unreadable from a cursor in the middle")
	}
	if endsAt != next {
		t.Errorf("a read from %d ends at cursor %d and a read from nought at %d", middle, endsAt, next)
	}
	if !reflect.DeepEqual(behind, again[middle:]) {
		t.Errorf("a read from %d handed %d bodies and the tail of the whole record is %d",
			middle, len(behind), len(again)-middle)
	}

	// A reader that is up to date: an empty read, and the cursor stands still.
	nothing, still, ok := registry.Since(code, next)
	if !ok || len(nothing) != 0 || still != next {
		t.Errorf("a watcher that is up to date was handed %d bodies at cursor %d (known=%v)",
			len(nothing), still, ok)
	}

	// An unknown code is not an empty read, which is the same distinction Read
	// and Answer.Known already draw.
	if _, _, known := registry.Since(codeFor(t, 99), 0); known {
		t.Error("a code no room is running under was reported as a readable record")
	}

	// And the read is the room's goroutine's: close the room and it stops
	// answering, rather than going on reading a record from outside.
	if !registry.Close(code) {
		t.Fatal("there was no room to close")
	}
	registry.Wait()
	if _, _, known := registry.Since(code, 0); known {
		t.Error("a closed room still answered a watcher's read, so the read was not going through its goroutine")
	}
	t.Logf("%d bodies recorded over 12 decisions, read whole, incrementally and from the middle; "+
		"a closed room and an unknown code both report nothing to read", len(again))
}

// TestManyWatchersReadOneRoomWhileItIsPlayed is what the race detector is here
// for: a watcher's read is a call into the room's goroutine like any other, so
// reads taken while a match is being driven must be ordered by the same channel
// the decisions are.
//
// ⚠️ It asserts a **count** rather than only running: a reader whose loop exited
// on its first answer would leave -race with nothing to interleave, and the run
// would look exactly like a clean one.
func TestManyWatchersReadOneRoomWhileItIsPlayed(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	configuration := config(11, 1)
	code, err := registry.Open(theOneListener, configuration, dependencies)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	defer registry.CloseAll()

	const readers = 3
	stop := make(chan struct{})
	handed := make([]int, readers)
	var readersDone sync.WaitGroup
	for at := 0; at < readers; at++ {
		readersDone.Add(1)
		go func(which int) {
			defer readersDone.Done()
			cursor := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				bodies, next, known := registry.Since(code, cursor)
				if !known {
					// The match ended and the room retired its entry, which is
					// the ordinary end of a watcher's read.
					return
				}
				handed[which] += len(bodies)
				cursor = next
			}
		}(at)
	}

	host, guest := squadPair(t, dependencies, 1)
	played := playMatch(t, throughTheRegistry{registry: registry, code: code},
		dependencies, configuration, host, guest)
	close(stop)
	readersDone.Wait()

	if !played.reading.Finished {
		t.Fatal("the match did not finish")
	}
	total := 0
	for which, count := range handed {
		if count == 0 {
			t.Errorf("reader %d was handed nothing across a whole match, so it interleaved with nothing", which)
		}
		total += count
	}
	t.Logf("%d watchers were handed %d bodies between them while %d decisions were played through "+
		"the same room", readers, total, played.steps)
}

// TestEveryInputAnswersWithWhatItRecorded is the reason the record read rides
// home on an input's own answer rather than on a Since taken afterwards, and it
// holds the measurement that forced it.
//
// ⚠️ **The exchange that records a match's last wire.Turn is the exchange that
// finishes the room**, and a finished room retires its own entry at once — so a
// consumer that answered its players and *then* asked for the record was asking a
// room that had already gone. This test asserts both halves: the whole stream is
// complete when it is accumulated off the answers, and the Since taken after the
// finishing decision reports **no room to read**. Without the second half a
// reader would take the first for tidiness rather than for necessity.
//
// It also holds the arithmetic every consumer of this needs: an answer's Cursor
// is where the record now stands, so the bodies it carries start exactly where
// the last answer left off. A consumer with its own cursor can check its place
// against that and never has to hand a cursor back to a room — which is what
// makes Room.Since's deliberate panic on an out-of-range cursor unreachable from
// a transport. → Registry.Since, and TODO.md's spectator step 4.
func TestEveryInputAnswersWithWhatItRecorded(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	configuration := config(11, 1)
	code, err := registry.Open(theOneListener, configuration, dependencies)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	defer func() { registry.CloseAll(); registry.Wait() }()

	clients := newTable(t, dependencies, configuration.TurnCap)
	host, guest := squadPair(t, dependencies, 0)
	cursor, carried := 0, []wire.Body{}
	// take accumulates one answer's recorded bodies and checks that they follow
	// on from the last one.
	take := func(what string, answered room.Answer) {
		t.Helper()
		if want := cursor + len(answered.Watched); answered.Cursor != want {
			t.Fatalf("%s carried %d bodies from cursor %d and says the record now reaches %d, "+
				"want %d: an answer's bodies have to be the ones between the two cursors",
				what, len(answered.Watched), cursor, answered.Cursor, want)
		}
		carried = append(carried, answered.Watched...)
		cursor = answered.Cursor
	}

	var answered room.Answer
	for _, joining := range []struct {
		squad placement.Squad
		name  string
	}{{host, "Host"}, {guest, "Guest"}} {
		answered, err = registry.Join(code, hello(t, joining.squad, joining.name))
		if err != nil {
			t.Fatalf("%s joins: %v", joining.name, err)
		}
		if !answered.Known || !answered.Seat.Valid() {
			t.Fatalf("%s was not seated: %+v", joining.name, answered)
		}
		clients.deliver(t, answered.Out)
		take(joining.name+"'s join", answered)
	}
	// The second seat's join is what opens the battle, so it is the input that
	// recorded the wire.Start — and the first join recorded nothing at all.
	if len(carried) != 1 {
		t.Fatalf("two joins recorded %d bodies, want the one wire.Start the second one opens",
			len(carried))
	}
	if _, isStart := carried[0].(wire.Start); !isStart {
		t.Fatalf("the body a join recorded is a %s, want a start", carried[0].Kind())
	}

	reading, decisions := answered.Reading, 0
	for !reading.Finished {
		if !reading.Waiting {
			t.Fatalf("after %d decisions the room waits on nobody and the match is not over", decisions)
		}
		answered, err = registry.Deliver(code, reading.Awaiting, clients.at(reading.Awaiting).answer())
		if err != nil {
			t.Fatalf("decision %d from %s: %v", decisions, reading.Awaiting, err)
		}
		if !answered.Known {
			t.Fatalf("the room went away after %d decisions with the match unfinished", decisions)
		}
		clients.deliver(t, answered.Out)
		take(fmt.Sprintf("decision %d", decisions), answered)
		reading = answered.Reading
		decisions++
		if decisions > configuration.Battles*configuration.TurnCap {
			t.Fatalf("the match took more than %d decisions, so something is not progressing", decisions)
		}
	}

	// A turn a decision, a start a battle, and the last of them is the one this
	// whole arrangement exists for.
	starts, turns := 0, 0
	for _, body := range carried {
		switch body.(type) {
		case wire.Start:
			starts++
		case wire.Turn:
			turns++
		default:
			t.Errorf("the record carries a %s, and it holds starts, turns and a closure", body.Kind())
		}
	}
	if turns != decisions {
		t.Errorf("%d decisions were played and the answers carried %d turns", decisions, turns)
	}
	if starts != len(reading.Played) {
		t.Errorf("%d battles were played and the answers carried %d starts", len(reading.Played), starts)
	}

	// ⚠️ And the half that says why: the room is **gone** by the time anybody
	// could ask it for that last turn. A reader that took a Since after each
	// decision would have every turn but the final one, which is the turn the
	// match ends on.
	if _, _, known := registry.Since(code, 0); known {
		t.Fatal("the room is still readable after the decision that finished the match, so the " +
			"reading below measures nothing and a Since taken afterwards would have done")
	}
	t.Logf("%d decisions and %d battles: the answers carried %d bodies (%d starts, %d turns), "+
		"the last of them from an input the room did not survive",
		decisions, len(reading.Played), len(carried), starts, turns)
}

// TestARoomTakesWatchersOnlyWhenItWasOpenedToThem is step 6's half at this
// layer: watching is **opt-in**, so the same hello that is welcomed by a room a
// host opened to spectators is turned away by one they did not.
//
// ⚠️ **The two arms differ by one field and nothing else** — the same seed, the
// same five decisions taken first, the same watching hello — so a gate that
// stopped reading Config.Watchable fails one of them whichever way it stopped
// reading it. A test with only the refusing arm would pass on a room that
// refused every watcher, and one with only the welcoming arm is the test that
// shipped in step 3.
//
// ⚠️ **The refusal is CodeWatchingClosed and specifically not CodeRoomFull**,
// which this room genuinely is: both seats are taken by the time a watcher
// arrives, so the nearest true-sounding refusal is available and is the wrong
// one. A spectator told the room is full has been told the one thing it can do
// nothing about — a full room is exactly the room worth watching. It is not
// CodeTooManyWatchers either, whose two books tell the reader to wait for one of
// the current watchers to leave; there are none, so that advice cannot come
// true. → wire.CodeWatchingClosed.
//
// ⚠️ **"The room is otherwise untouched" is the whole roomState either side of
// the join, not a field or two.** A refusal that spent the open prompt, moved
// the seat on turn or appended to the watcher's record would be a refusal that
// changed the match it refused, and the seat on turn answering afterwards is
// what says the prompt is really open rather than merely reported open.
func TestARoomTakesWatchersOnlyWhenItWasOpenedToThem(t *testing.T) {
	// ⚠️ Read off a zero Config rather than assumed: if watching were on by
	// default the refusing arm below would be measuring a room nobody can open.
	if (room.Config{}).Watchable {
		t.Fatal("a room takes watchers before any host asks for it, so watching is not opt-in " +
			"and the arm below that expects a refusal cannot be reached")
	}
	dependencies := deps(t)
	for _, one := range []struct {
		name     string
		opening  func(room.Config) room.Config
		welcomed bool
	}{
		{
			name:     "a room the host opened to spectators",
			opening:  watchable,
			welcomed: true,
		},
		{
			name:     "a room the host did not",
			opening:  func(configuration room.Config) room.Config { return configuration },
			welcomed: false,
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened, clients := openMatch(t, one.opening(config(11, 3)))
			spare := squadOf(t, dependencies.Characters, "spare.squad",
				"pokemon.mew", "pokemon.poliwag", "pokemon.magnemite")
			for step := 0; step < 5; step++ {
				answerFor(t, opened, clients, "")
			}

			before := stateOf(t, opened, spare)
			if before.probe != wire.CodeRoomFull {
				t.Fatalf("the room answers a player %q rather than %q, so it is not the full room "+
					"this test is about", before.probe, wire.CodeRoomFull)
			}
			admitted, out, err := opened.Join(watchingHello(t, "Watcher", placement.Squad{}))
			if err != nil {
				t.Fatalf("a watcher joins: %v", err)
			}
			if admitted.Seat.Valid() {
				t.Errorf("a watcher was given the %q seat", admitted.Seat)
			}
			if admitted.Watching != one.welcomed {
				t.Fatalf("the watcher was %s and it should have been %s: the only thing that "+
					"decides is room.Config.Watchable, and it is %t here",
					welcomedOrNot(admitted.Watching), welcomedOrNot(one.welcomed),
					opened.Config().Watchable)
			}
			if one.welcomed {
				if !onlyWelcome(t, out).Watching() {
					t.Error("the watcher's welcome names a seat, so its own client reads it as a player's")
				}
			} else if code := onlyCode(t, out); code != wire.CodeWatchingClosed {
				t.Errorf("a watcher of a room that takes none was answered %q, want %q: %q is "+
					"about the two seats and this client asked for neither, and %q tells the "+
					"reader to wait for a watcher to leave when there are none",
					code, wire.CodeWatchingClosed, wire.CodeRoomFull, wire.CodeTooManyWatchers)
			}

			if after := stateOf(t, opened, spare); before != after {
				t.Fatalf("the watching join moved the room:\n before %+v\n after  %+v", before, after)
			}
			onTurn, answered := answerFor(t, opened, clients, "")
			if onTurn != before.awaiting {
				t.Errorf("the seat on turn after the watching join is %q and was %q", onTurn, before.awaiting)
			}
			if len(answered) == 0 {
				t.Error("the turn after the watching join produced nothing at all")
			}
			t.Logf("Watchable=%t: the watcher was %s, and the room was identical either side of "+
				"it with %d bodies on the record",
				opened.Config().Watchable, welcomedOrNot(admitted.Watching), before.recorded)
		})
	}
}

// welcomedOrNot is one word for what the gate did with a watching hello, so the
// two messages above read as sentences rather than as booleans.
func welcomedOrNot(watching bool) string {
	if watching {
		return "welcomed"
	}
	return "turned away"
}
