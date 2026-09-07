package room_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// transcript is everything the two **players** saw of one match, which is what a
// watcher must be unable to change.
//
// It deliberately holds no state of the watcher's beyond the counts: the claim
// under test is about the players, so anything the watcher was handed belongs in
// the tests that are about the record itself.
type transcript struct {
	// turns is every wire.Turn the host was sent, in order. The guest is sent
	// the same body by the same call, and its own mirror checks every digest
	// against the events its engine produced, so one seat's copy is the whole
	// of the comparison.
	turns   []wire.Turn
	result  room.Result
	played  []room.BattleResult
	skipped int
	// compared is how many digests the two clients checked between them, so a
	// run in which no mirror ever ran cannot pass for one in which they agreed.
	compared int
	// reads and handed are the watcher's, and they are here so a "with a
	// watcher" run that never actually read anything is a red test rather than
	// an identical one.
	reads, handed int
	// recorded is the whole record at the end of the match, read from nought.
	recorded []wire.Body
}

// playAMatch plays a bo3 from one seed with both clients answering off their own
// mirrors, and — when watching — reads the room's watcher record as it goes.
//
// ⚠️ **The reads are interleaved rather than taken at the end**, which is the
// whole point of the fixture: a read that only ever happened after the last turn
// could not starve anybody of anything. So there is one incremental read after
// every decision, and every third decision also takes a read from a **stale**
// cursor and a read from nought — the two shapes a real watcher produces, a
// reader that fell behind and one that joined halfway.
func playAMatch(t *testing.T, watching bool) transcript {
	t.Helper()
	configuration := config(11, 3)
	opened, clients := openMatch(t, configuration)

	out := transcript{}
	cursor, stale := 0, 0
	steps := 0
	for !opened.Finished() {
		_, answered := answerFor(t, opened, clients, "")
		for _, message := range answered {
			// One seat's copy: both are sent the same body by the same call.
			if turn, isTurn := message.Body.(wire.Turn); isTurn && message.To == wire.SeatHost {
				out.turns = append(out.turns, turn)
			}
		}
		steps++
		if watching {
			bodies, next := opened.Since(cursor)
			out.reads++
			out.handed += len(bodies)
			cursor = next
			if steps%3 == 0 {
				behind, _ := opened.Since(stale)
				whole, _ := opened.Since(0)
				out.reads += 2
				out.handed += len(behind) + len(whole)
				stale = cursor
			}
		}
		// A backstop rather than an expectation: without it a state machine
		// that stopped making progress would hang the suite instead of failing
		// it.
		if steps > 3*configuration.TurnCap {
			t.Fatalf("the match took more than %d decisions, so something is not progressing", steps)
		}
	}
	out.result = opened.Result()
	out.played = opened.Played()
	out.skipped = opened.Skipped()
	out.compared = clients.host.compared + clients.guest.compared
	out.recorded, _ = opened.Since(0)
	return out
}

// TestAMatchPlayedWithAWatcherReadingIsTheSameMatch is the one test this whole
// step exists to make possible, and nothing else in the suite can stand in for
// it.
//
// ⚠️ **A watcher threaded through the players' own structures would pass
// everything else.** The roster would still be legal, both peers would still
// agree on every digest — they would both have the watcher — and the match would
// still run to a verdict. What a third seat actually costs is the *order* the
// seats are visited in, which reaches the roster, and the roster's order decides
// which side wins a speed tie. So the only measurement that can see it is a
// comparison against a match played **without** one, from the same seed and with
// the same decisions, which is what this is.
//
// ⚠️ **The concrete bug it is a net for is one line away from being written.**
// resolved reads the battle with `events, next := r.fight.Since(r.cursor)` and
// then assigns `r.cursor = next`, so a watcher read implemented against
// **r.cursor** — the obvious move, since the room already holds a cursor —
// would take the players' own events out from under them. Measured: making
// Room.Since advance r.cursor reddens this test, and the failure is a digest
// divergence on the very first turn after the first read rather than anything
// about the watcher.
func TestAMatchPlayedWithAWatcherReadingIsTheSameMatch(t *testing.T) {
	watched := playAMatch(t, true)
	alone := playAMatch(t, false)

	// The run that was supposed to watch has to have watched, or the two runs
	// are the same run twice and this measures nothing.
	if watched.reads == 0 || watched.handed == 0 {
		t.Fatalf("the watched run took %d reads and was handed %d bodies, so nothing was watching",
			watched.reads, watched.handed)
	}
	if alone.reads != 0 {
		t.Fatalf("the unwatched run took %d reads", alone.reads)
	}
	if watched.compared == 0 || alone.compared == 0 {
		t.Fatalf("the clients checked %d and %d digests, so at least one match ran with no mirror",
			watched.compared, alone.compared)
	}

	if len(watched.turns) != len(alone.turns) {
		t.Fatalf("the watched match played %d turns and the unwatched one %d",
			len(watched.turns), len(alone.turns))
	}
	for index := range watched.turns {
		with, without := watched.turns[index], alone.turns[index]
		if with.Decision != without.Decision {
			t.Fatalf("turn %d differs: watched %+v, unwatched %+v", index, with.Decision, without.Decision)
		}
		if with.Events != without.Events {
			t.Fatalf("turn %d of %q digests %s with a watcher and %s without one",
				index, with.Decision.Unit, with.Events.Short(), without.Events.Short())
		}
	}
	if watched.result != alone.result {
		t.Errorf("the watched match ended %+v and the unwatched one %+v", watched.result, alone.result)
	}
	if len(watched.played) != len(alone.played) {
		t.Fatalf("the watched match played %d battles and the unwatched one %d",
			len(watched.played), len(alone.played))
	}
	for index := range watched.played {
		// DeepEqual rather than !=, because a BattleResult carries the battle's
		// log and a log is slices. ⚠️ That widens the claim rather than weakening
		// it: the two matches now have to produce the same **events** as well as
		// the same outcome, which is the strongest form of "a watcher changes
		// nothing" this test can make.
		if !reflect.DeepEqual(watched.played[index], alone.played[index]) {
			t.Errorf("battle %d differs: watched %+v, unwatched %+v",
				index+1, watched.played[index], alone.played[index])
		}
	}
	if watched.skipped != alone.skipped {
		t.Errorf("the watched match skipped %d prompts and the unwatched one %d",
			watched.skipped, alone.skipped)
	}
	t.Logf("%d turns, %d battles and %d skipped prompts, identical with and without a watcher "+
		"taking %d reads of %d bodies", len(watched.turns), len(watched.played), watched.skipped,
		watched.reads, watched.handed)
}

// TestAWatchersDeliverIsRefusedAndLeavesTheRoomWhereItWas is the other half of
// "a watcher is not a seat": it can be *sent* everything and can *say* nothing.
//
// ⚠️ **The state-untouched half is the one that matters.** A refusal that had
// already spent the turn would still look exactly like a refusal from the
// outside — one wire.Refused and nothing else — so the assertions here are about
// what did not move: the prompt is still the same prompt, the same seat is still
// on turn, no body reached the watcher's record, and the seat that really was on
// turn can still take the turn it was being asked for, with its mirror agreeing
// on the digest.
//
// The code is wire.CodeNotYourTurn because that is the closest true thing the
// ten codes can say — there is no "you are not playing in this room" among them
// — and Deliver's own comment says so.
func TestAWatchersDeliverIsRefusedAndLeavesTheRoomWhereItWas(t *testing.T) {
	opened, clients := openMatch(t, config(11, 1))

	// ⚠️ **The table is driven once with each seat on turn, and the test refuses
	// to pass until both have been.** Deliver refuses an unseated sender before
	// it looks at the prompt, and answerFrom refuses a seat that is not the one
	// being asked *after* — so a run in which only the seat the watcher is NOT
	// impersonating happened to be on turn is a run where the second guard hides
	// the first, and a sender routed to seat nought would sail through it.
	// Measured: with only one arrangement exercised, deleting Deliver's `seated`
	// test changed nothing.
	refusals, rounds := 0, 0
	onTurnSeen := map[wire.Seat]int{}
	for step := 0; step < 16; step++ {
		if onTurnSeen[wire.SeatHost] > 0 && onTurnSeen[wire.SeatGuest] > 0 {
			break
		}
		if opened.Finished() {
			break
		}
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d turns both seats are taken and the room is waiting on nobody", step)
		}
		prompt := opened.Pending()
		if prompt == nil {
			t.Fatal("the room is waiting on a seat with no prompt open")
		}
		before, _ := opened.Since(0)

		// ⚠️ **The Act is an otherwise LEGAL one, taken off the open prompt**, so
		// the refusal measured here is about the seat and not about the action:
		// an Act nobody could have taken would be refused as CodeIllegalAction by
		// a room that had never looked at who sent it.
		act := wire.Act{}
		for _, option := range prompt.Options {
			if len(option.Aims) > 0 {
				act = wire.Act{Skill: option.Skill, Aim: hex.At(option.Aims[0])}
				break
			}
		}
		if act.Skill == "" {
			t.Fatal("the open prompt offers no action to aim, so a legal Act cannot be built")
		}

		// The three messages a client sends once it is in, from a connection
		// that holds no seat. The zero Seat is what a watcher's connection
		// carries; the second is a seat this room never handed out, which
		// reaches the same refusal by the same test and would be the shape of a
		// stale reconnect.
		for _, from := range []wire.Seat{"", wire.Seat("watcher")} {
			for _, body := range []wire.Body{act, wire.Pass{}, wire.Decide{}} {
				out, err := opened.Deliver(from, body)
				if err != nil {
					t.Fatalf("a %s from %q: %v", body.Kind(), from, err)
				}
				if len(out) != 1 {
					t.Fatalf("a %s from %q was answered with %d messages, want one refusal",
						body.Kind(), from, len(out))
				}
				refused, isRefusal := out[0].Body.(wire.Refused)
				if !isRefusal {
					t.Fatalf("a %s from %q was answered with a %s, want a refusal",
						body.Kind(), from, out[0].Body.Kind())
				}
				if refused.Code != wire.CodeNotYourTurn {
					t.Errorf("a %s from %q was refused %q, want %q",
						body.Kind(), from, refused.Code, wire.CodeNotYourTurn)
				}
				refusals++

				// Nothing moved. The prompt is compared by identity as well as
				// by value: a battle that had advanced and come back to the same
				// unit would hold a different one.
				if still, open := opened.Awaiting(); !open || still != onTurn {
					t.Fatalf("after a %s from %q the room is waiting on %q (open %v), want %q",
						body.Kind(), from, still, open, onTurn)
				}
				if opened.Pending() != prompt {
					t.Fatalf("a %s from %q moved the open prompt", body.Kind(), from)
				}
				after, _ := opened.Since(0)
				if len(after) != len(before) {
					t.Fatalf("a %s from %q put %d bodies on the record",
						body.Kind(), from, len(after)-len(before))
				}
				if opened.Finished() {
					t.Fatalf("a %s from %q ended the match", body.Kind(), from)
				}
			}
		}
		rounds++
		onTurnSeen[onTurn]++

		// And the turn those refusals were aimed at is still there to be taken,
		// with the acting client's own engine agreeing on the digest — which is
		// the sharpest form of "the battle did not move", because a battle that
		// had quietly advanced would put the two out of step here.
		checked := clients.at(onTurn).compared
		answerFor(t, opened, clients, "")
		if clients.at(onTurn).compared <= checked {
			t.Fatalf("%s's turn produced no digest to compare after the refusals", onTurn)
		}
	}
	for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if onTurnSeen[seat] == 0 {
			t.Fatalf("the table was never driven with %s on turn, so half of it measures nothing "+
				"— the refusal a seat that is not being asked already gets would cover it", seat)
		}
	}
	if want := 6 * rounds; refusals != want {
		t.Fatalf("%d rounds drove %d refusals, want %d", rounds, refusals, want)
	}
	t.Logf("%d refusals over %d rounds (%d with the host on turn, %d with the guest), none of "+
		"which moved the prompt, the seat on turn or the record",
		refusals, rounds, onTurnSeen[wire.SeatHost], onTurnSeen[wire.SeatGuest])
}

// TestTheWatcherRecordIsWriteOnlyFromTheRoom is the guard that stops a watcher
// starting to change the battle it is watching, and it is mechanical because the
// way that happens is not a decision anybody takes: it is an innocent
// `if len(r.watched) > 0` appearing in one of the paths the room already runs.
//
// Two claims, both exhaustive over this package's own directory and both
// counted, because a walk that matched nothing would pass either way:
//
//  1. the record's field is named only inside the two functions that own it —
//     watch, which appends, and Since, which reads; and
//  2. the append is called from exactly the three places the room already emits
//     the corresponding body to the players, and from nowhere else.
//
// The second is what keeps the record honest: an append somewhere the two
// players are told nothing would be a watcher seeing a match they did not have.
func TestTheWatcherRecordIsWriteOnlyFromTheRoom(t *testing.T) {
	// The record's field may be named here, and nowhere else.
	owners := map[string]bool{"watch": true, "Since": true}
	// The append is made here, once each, and nowhere else — the three sites
	// where the room already sends the same body to the two seats.
	callers := map[string]int{"begin": 0, "resolved": 0, "abandon": 0}

	scanned, named, calls := 0, 0, 0
	for _, name := range packageSources(t) {
		scanned++
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || function.Body == nil {
				continue
			}
			within := function.Name.Name
			ast.Inspect(function.Body, func(node ast.Node) bool {
				selector, isSelector := node.(*ast.SelectorExpr)
				if !isSelector || selector.Sel == nil {
					return true
				}
				switch selector.Sel.Name {
				case "watched":
					named++
					if !owners[within] {
						t.Errorf("%s:%d reads or writes the watcher's record inside %s: "+
							"the room writes that record and never reads it, so a branch on it "+
							"anywhere else is a watcher changing the match it is watching",
							name, fileSet.Position(selector.Pos()).Line, within)
					}
				case "watch":
					// The declaration of the method is a FuncDecl rather than a
					// selector, so every hit here is a call to it.
					calls++
					if _, expected := callers[within]; !expected {
						t.Errorf("%s:%d appends to the watcher's record inside %s, which is not "+
							"one of the three sites the room emits the same body to the players",
							name, fileSet.Position(selector.Pos()).Line, within)
						return true
					}
					callers[within]++
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	if named == 0 {
		t.Fatal("the scan found no reference to the watcher's record at all, so it measures nothing")
	}
	for _, site := range []string{"begin", "resolved", "abandon"} {
		if callers[site] != 1 {
			t.Errorf("%s appends to the watcher's record %d times, want exactly one",
				site, callers[site])
		}
	}
	t.Logf("scanned %d source files: %d references to the record, %d appends, all accounted for",
		scanned, named, calls)
}

// TestAWatcherJoiningHalfwayIsHandedTheWholeMatch is why the read is a Since
// with a caller-held cursor rather than anything that empties: `Since(0)` is a
// watcher that wants everything, and joining halfway is exactly that.
//
// ⚠️ **The wire.Start is the load-bearing entry.** Without it a late watcher
// would hold a run of decisions and nothing to apply them to — the roster is
// built in a local inside begin and there is no Battle.Roster() accessor — so
// recording the body is what makes the mid-joiner work with no new engine
// surface at all.
func TestAWatcherJoiningHalfwayIsHandedTheWholeMatch(t *testing.T) {
	configuration := config(11, 1)
	opened, clients := openMatch(t, configuration)

	// A watcher that was there from the beginning, so its copy is what the late
	// one is compared against.
	fromTheStart, cursor := opened.Since(0)
	if len(fromTheStart) != 1 {
		t.Fatalf("a room with its first battle open recorded %d bodies, want the start alone",
			len(fromTheStart))
	}
	opening, isStart := fromTheStart[0].(wire.Start)
	if !isStart {
		t.Fatalf("the first recorded body is a %s, want a start", fromTheStart[0].Kind())
	}
	if opening.Battle != 1 {
		t.Errorf("the recorded start opens battle %d, want the first", opening.Battle)
	}
	if opening.Seed != configuration.SeedFor(1) {
		t.Errorf("the recorded start carries seed %d, want %d", opening.Seed, configuration.SeedFor(1))
	}
	if want := 2 * configuration.Format.Units(); len(opening.Roster) != want {
		t.Errorf("the recorded start carries %d units, want %d", len(opening.Roster), want)
	}
	// ⚠️ The side is the **host's**, decided rather than fallen out of a loop: a
	// watcher plays neither half, and the alternative — leaving it at its zero
	// value — reads as SideAlly to everything downstream.
	if want := clients.host.side; opening.Side != want {
		t.Errorf("the recorded start puts a watcher on the %s side, want the host's %s",
			opening.Side, want)
	}

	// Several turns, collected as the host was sent them.
	var sent []wire.Turn
	const turns = 6
	for step := 0; step < turns; step++ {
		if opened.Finished() {
			t.Fatalf("the battle ended after %d turns, so this fixture measures a shorter match "+
				"than it means to", step)
		}
		_, out := answerFor(t, opened, clients, "")
		for _, message := range out {
			if turn, isTurn := message.Body.(wire.Turn); isTurn && message.To == wire.SeatHost {
				sent = append(sent, turn)
			}
		}
	}
	if len(sent) != turns {
		t.Fatalf("%d decisions produced %d turns on the wire", turns, len(sent))
	}

	// The late watcher: one read, from nought, holding everything.
	whole, _ := opened.Since(0)
	if len(whole) != 1+turns {
		t.Fatalf("a watcher joining after %d turns was handed %d bodies, want %d",
			turns, len(whole), 1+turns)
	}
	if _, isStart := whole[0].(wire.Start); !isStart {
		t.Fatalf("the late watcher was handed a %s first, want the start", whole[0].Kind())
	}
	for index, body := range whole[1:] {
		turn, isTurn := body.(wire.Turn)
		if !isTurn {
			t.Fatalf("body %d of the record is a %s, want a turn", index+1, body.Kind())
		}
		if turn != sent[index] {
			t.Fatalf("turn %d of the record is %q's, and the host was sent %q's",
				index, turn.Decision.Unit, sent[index].Decision.Unit)
		}
	}

	// And the watcher that was there from the start reads only what is new,
	// which is the same run without the opening.
	since, _ := opened.Since(cursor)
	if len(since) != turns {
		t.Fatalf("the watcher that was there from the start read %d new bodies, want %d",
			len(since), turns)
	}
	t.Logf("a start and %d turns, read whole from nought and incrementally from %d", turns, cursor)
}

// TestTwoWatchersCursorsDoNotInterfere is the reason this is a Since and not a
// Drain, stated as a measurement: a single-consumer cursor that emptied what it
// read would let whichever watcher read first decide what the others never see.
func TestTwoWatchersCursorsDoNotInterfere(t *testing.T) {
	opened, clients := openMatch(t, config(11, 1))

	// One watcher keeps up; the other has not read since the room opened.
	keepingUp, _ := opened.Since(0)
	if len(keepingUp) != 1 {
		t.Fatalf("a room with its first battle open recorded %d bodies, want the start alone",
			len(keepingUp))
	}
	ahead := 1
	const behind = 0

	const turns = 5
	for step := 0; step < turns; step++ {
		if opened.Finished() {
			t.Fatalf("the battle ended after %d turns", step)
		}
		answerFor(t, opened, clients, "")
	}

	first, next := opened.Since(ahead)
	if len(first) != turns {
		t.Fatalf("the watcher at %d read %d bodies, want the %d turns it had not seen",
			ahead, len(first), turns)
	}
	ahead = next

	// The second watcher, reading after the first, is still owed everything.
	second, alsoNext := opened.Since(behind)
	if len(second) != 1+turns {
		t.Fatalf("the watcher at %d read %d bodies, want the start and %d turns",
			behind, len(second), turns)
	}
	if alsoNext != ahead {
		t.Errorf("the two watchers ended at cursors %d and %d, and the record has one end",
			alsoNext, ahead)
	}
	// The one that had kept up saw the same turns as the one that had not, in
	// the same order — the second's run is the first's with the opening in front.
	for index := range first {
		if first[index] != second[index+1] {
			t.Fatalf("body %d differs between the two watchers", index)
		}
	}
	// And neither read emptied anything: a third watcher joining now is owed
	// the whole match, exactly as the second was.
	third, _ := opened.Since(0)
	if len(third) != 1+turns {
		t.Fatalf("a third watcher was handed %d bodies after two others had read, want %d",
			len(third), 1+turns)
	}
	t.Logf("two cursors at %d and %d over one record of %d bodies, neither emptying it",
		ahead, behind, len(third))
}

// TestAnOutOfRangeWatcherCursorPanics holds the deliberate panic, for
// battle.Since's own reason: answering a cursor the record cannot answer with an
// empty slice would make a consumer that has got ahead of the room look exactly
// like one that is up to date, which is the silent desync a cursor exists to
// prevent. A cursor is a number Since handed the caller itself, so a bad one is
// a programming error rather than a runtime condition.
func TestAnOutOfRangeWatcherCursorPanics(t *testing.T) {
	opened, _ := openMatch(t, config(11, 1))
	_, end := opened.Since(0)
	if end == 0 {
		t.Fatal("the record is empty, so a cursor above it is the same cursor as one at it")
	}
	for _, cursor := range []int{end + 1, -1} {
		func() {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Errorf("a cursor of %d against a record of %d did not panic", cursor, end)
				}
			}()
			opened.Since(cursor)
		}()
	}
	// The cursor at the end is not out of range: it is a watcher that is up to
	// date, and it reads nothing.
	nothing, still := opened.Since(end)
	if len(nothing) != 0 || still != end {
		t.Errorf("a watcher at the end read %d bodies and moved to %d", len(nothing), still)
	}
	t.Logf("a record of %d bodies refuses %d and -1 and answers %d with nothing", end, end+1, end)
}

// TestADepartureReachesTheWatchersRecord is the ending a watcher cannot compute
// for itself. Every other ending it works out the way a player does — each
// battle's outcome off its own Ended event, the series length off the welcome,
// the cap by the same arithmetic — but a departure leaves no Ended for the
// battle in progress and no further start, so a watcher handed nothing would
// hang on a turn that is never coming.
//
// ⚠️ **The seat that stayed still gets its own Outbound, exactly as before.**
// The record is an addition and not a replacement: a watcher takes the closure
// off the record precisely so that Outbound never has to address anything but a
// seat, which is what keeps other() the other one.
func TestADepartureReachesTheWatchersRecord(t *testing.T) {
	opened, clients := openMatch(t, config(11, 1))
	for step := 0; step < 4; step++ {
		if opened.Finished() {
			t.Fatalf("the battle ended after %d turns", step)
		}
		answerFor(t, opened, clients, "")
	}
	before, _ := opened.Since(0)

	out, err := opened.Left(wire.SeatGuest)
	if err != nil {
		t.Fatalf("the guest leaves: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("a departure produced %d messages, want the one to the seat that stayed", len(out))
	}
	if out[0].To != wire.SeatHost {
		t.Errorf("a departure told %q, want the host, which is the seat that stayed", out[0].To)
	}
	closed, isClosed := out[0].Body.(wire.Closed)
	if !isClosed {
		t.Fatalf("a departure sent a %s, want a closure", out[0].Body.Kind())
	}
	if closed.Reason != wire.ClosureLeft {
		t.Errorf("a departure closed the room as %q, want %q", closed.Reason, wire.ClosureLeft)
	}

	after, _ := opened.Since(0)
	if len(after) != len(before)+1 {
		t.Fatalf("a departure put %d bodies on the record, want one", len(after)-len(before))
	}
	last, isClosure := after[len(after)-1].(wire.Closed)
	if !isClosure {
		t.Fatalf("the record ends with a %s, want the closure", after[len(after)-1].Kind())
	}
	if last != closed {
		t.Errorf("the record ends with %+v and the seat that stayed was sent %+v", last, closed)
	}
	t.Logf("a record of %d bodies ending in %q, and one Outbound to the seat that stayed",
		len(after), last.Reason)
}

// watchersOwnBody is a body a watcher appends to its own copy of the record,
// and it is one the room never records on this path: the room's only closure
// here is a departure's, so a ClosureStopped turning up **in** the record is the
// caller's own append having reached it.
var watchersOwnBody wire.Body = wire.Closed{Reason: wire.ClosureStopped}

// TestAWatchersViewAndTheRecordSurviveEachOthersAppends is the three-index
// slice, which is not tidiness: a view sharing the record's spare capacity
// corrupts **both** copies. The caller's append writes into the slot the room's
// next body is going to be recorded in, and that body then overwrites what the
// caller appended — measured for battle.Since and for draft.Since, and this is
// the third record with the same shape.
//
// ⚠️ **It is only reachable while cap > len**, and a caller cannot ask another
// package for its capacity, so the property is asserted directly *and* swept
// across the record's growth with a count: append's growth is amortised, so a
// single reading is a dice roll wearing an assertion.
func TestAWatchersViewAndTheRecordSurviveEachOthersAppends(t *testing.T) {
	opened, clients := openMatch(t, config(11, 1))

	checked := 0
	for step := 0; step < 12; step++ {
		if opened.Finished() {
			break
		}
		view, before := opened.Since(0)
		if len(view) != before {
			t.Fatalf("Since(0) answered %d bodies against a cursor of %d", len(view), before)
		}
		// The property stated directly, which is its sharpest form: a view's
		// capacity is its length, so a caller's append cannot reach the record's
		// next slot however the runtime grew the array. Reported rather than
		// fatal, so the two corruption readings below still run and say what the
		// spare capacity actually costs.
		if cap(view) != len(view) {
			t.Errorf("at %d bodies the view carries %d of spare capacity",
				before, cap(view)-len(view))
		}

		// One: the caller appends, then the room records.
		early := append(view, watchersOwnBody) //nolint:gocritic // the append is the measurement
		answerFor(t, opened, clients, "")
		_, after := opened.Since(0)
		if after == before {
			t.Fatalf("a decision was taken at %d bodies and the record did not grow", before)
		}
		checked++
		if kept, isClosure := early[before].(wire.Closed); !isClosure || kept != watchersOwnBody {
			t.Errorf("at %d bodies the room's next record overwrote the caller's own body with a %s",
				before, early[before].Kind())
		}

		// Two: the same view, appended to now the record has grown past it.
		late := append(view, watchersOwnBody) //nolint:gocritic // the append is the measurement
		if kept, isClosure := late[before].(wire.Closed); !isClosure || kept != watchersOwnBody {
			t.Errorf("at %d bodies a late append did not keep the caller's own body, got a %s",
				before, late[before].Kind())
		}
		record, _ := opened.Since(0)
		for at, body := range record {
			if closure, isClosure := body.(wire.Closed); isClosure && closure == watchersOwnBody {
				t.Errorf("the caller's own body reached the record at %d of %d, over the body "+
					"that belonged there", at, len(record))
			}
		}
	}
	if checked == 0 {
		t.Fatal("the room recorded nothing, so this fixture measured nothing")
	}
	t.Logf("%d readings, each with a body recorded after it", checked)
}
