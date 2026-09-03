package socket

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/wire"
)

// theWholeShutdown is how long a test will wait for a shutdown over loopback
// sockets before calling it hung. Everything it waits on is in-process — four
// sockets closing and two room goroutines returning, measured at 0.01s — so this
// is a margin for a loaded machine rather than a measurement of anything.
const theWholeShutdown = 30 * time.Second

// TestShutdownTellsEveryPeerAndLetsGoOfEveryRoom is the behavioural half of the
// gap TODO.md filed: http.Server.Shutdown does not wait for hijacked
// connections, so this is what a host binary calls instead.
//
// # What it can see
//
//   - **The notify.** Every one of the four connected peers is read until a
//     wire.Closed arrives, and the reason has to be ClosureStopped. Deleting the
//     send from Server.stopping turns each of those reads into a close error and
//     reddens this by Fatalf — measured, not assumed.
//   - **CloseAll.** ⚠️ This one had to be **built to be visible**, and the first
//     version of this test could not see it at all: deleting `CloseAll` left the
//     test green, because with no rejoin a socket closing *ends a match*, so the
//     notify's own dropped connections retire both rooms without anybody asking
//     them to. What CloseAll is really for is a room **nobody is connected to** —
//     opened, code printed, still waiting for its first player — and there was no
//     such room in the fixture. There is one now (`lonely`), and deleting
//     CloseAll leaves it Running for ever, so the settling poll never converges
//     and Shutdown returns the context's error. Measured both ways round.
//   - **Both readings, as genuinely different numbers.** Three rooms and two
//     tables: `Tables` counts rooms with connections and `Running` counts room
//     goroutines, and a fixture where the two happened to be equal could not tell
//     a shutdown that checked one from a shutdown that checked both.
//
// # What it CANNOT see, stated rather than smoothed over
//
// ⚠️ **It cannot see Registry.Wait blocking.** Step four polls until Tables and
// Running are both nought, and that poll converges whether or not step three
// waited first — so deleting the Wait leaves this test green. That is not an
// oversight in the test, it is the shape of the thing: Wait's contribution is
// *when* Shutdown returns, not what is true when it does, and a timing assertion
// on a test runner is a flaky test wearing a correctness claim. The claim is held
// structurally instead, by TestShutdownClosesEveryRoomAndThenMeasuresThem, which
// is deterministic. Neither test alone is enough.
//
// # Why more than one room
//
// ⚠️ **Server.Tables counts rooms and not connections**, so two peers in one room
// is `Tables() == 1` — indistinguishable from a fixture that connected nobody and
// left a table behind. The three codes are asserted **distinct** first:
// internal/room's registry test shipped a bug this month where two rooms were
// sometimes the same room, and a fixture that opened one room three times would
// pass every assertion below.
func TestShutdownTellsEveryPeerAndLetsGoOfEveryRoom(t *testing.T) {
	held := listening(t, Timings{})
	dependencies := deps(t)
	characters := dependencies.Characters

	first := held.open(t, config(7, 1, 60), dependencies)
	second := held.open(t, config(8, 1, 60), dependencies)
	// The room nobody joined, and it is the only thing in this test that measures
	// CloseAll. → the note above.
	lonely := held.open(t, config(9, 1, 60), dependencies)
	for _, pair := range [][2]wire.RoomCode{{first, second}, {first, lonely}, {second, lonely}} {
		if pair[0] == pair[1] {
			t.Fatalf("two rooms opened under the code %s, so this test would measure one room twice", pair[0])
		}
	}

	// Four connections over two of the three rooms, and deliberately **not**
	// playing: a match that ran to its end would tear its own table down, and this
	// is measuring what a shutdown does to a match still in progress.
	joined := []*Client{
		held.dial(t, first, hello(t, theHostSquad(t, characters), "first.host", ""), dependencies.Books),
		held.dial(t, first, hello(t, theGuestSquad(t, characters), "first.guest", ""), dependencies.Books),
		held.dial(t, second, hello(t, theHostSquad(t, characters), "second.host", ""), dependencies.Books),
		held.dial(t, second, hello(t, theGuestSquad(t, characters), "second.guest", ""), dependencies.Books),
	}

	// Non-vacuity, before anything is shut down. A shutdown with nothing
	// connected is a shutdown that measures nothing, and it is the exact shape of
	// pass this repository keeps a list of. The two numbers are asserted
	// **exactly**, and they differ: three rooms behind two tables.
	if tables := held.server.Tables(); tables != 2 {
		t.Fatalf("the server holds %d tables before the shutdown, want 2: nothing connected, so this test measures nothing", tables)
	}
	if running := held.rooms.Running(); running != 3 {
		t.Fatalf("%d rooms are running before the shutdown, want 3 (two joined and one nobody joined): "+
			"without the third, nothing here measures CloseAll", running)
	}

	ctx, cancel := context.WithTimeout(context.Background(), theWholeShutdown)
	defer cancel()
	if err := held.server.Shutdown(ctx); err != nil {
		t.Fatalf("shut the server down: %v", err)
	}

	if tables := held.server.Tables(); tables != 0 {
		t.Errorf("the server still holds %d tables after a shutdown that returned", tables)
	}
	if running := held.rooms.Running(); running != 0 {
		t.Errorf("%d room goroutines are still owed an end after a shutdown that returned", running)
	}
	if count := held.rooms.Count(); count != 0 {
		t.Errorf("%d rooms are still reachable by code after a shutdown that returned", count)
	}

	// And every one of the four was told, with the reason that is true.
	for _, client := range joined {
		if reason := toldWhyItStopped(t, client); reason != wire.ClosureStopped {
			t.Errorf("%s was told the match closed because %q, want %q", client.Seat(), reason, wire.ClosureStopped)
		}
	}
	if failures := held.failures.everything(); len(failures) != 0 {
		t.Errorf("the shutdown reported %d errors: %v", len(failures), failures)
	}
}

// toldWhyItStopped reads one client's messages until the room's own wire.Closed
// arrives, and hands back the reason it carried.
//
// ⚠️ The messages already in flight when the shutdown began are read **past**
// rather than asserted on. Each of these clients has an unread wire.Start
// waiting — Dial reads the one wire.Welcome and no more — and this is measuring
// what the shutdown added, not what the join sent; the join has its own tests.
//
// The bound is a message count rather than a deadline, so a server that streamed
// for ever fails the test instead of hanging it, and it is Fatalf rather than
// Errorf because the return value is read straight afterwards.
func toldWhyItStopped(t *testing.T, client *Client) wire.Closure {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), theWholeShutdown)
	defer cancel()
	const patience = 8
	for read := 0; read < patience; read++ {
		body, err := client.conn.read(ctx)
		if err != nil {
			t.Fatalf("%s read %d messages and then an error rather than a closed: %v", client.Seat(), read, err)
		}
		switch closed := body.(type) {
		case *wire.Closed:
			return closed.Reason
		case wire.Closed:
			return closed.Reason
		}
	}
	t.Fatalf("%s was sent %d messages after the shutdown and none of them was a closed", client.Seat(), patience)
	return wire.ClosureNone
}

// TestShutdownGivesUpAndNamesWhatItWasWaitingFor holds the bound, and holds that
// what it reports is usable.
//
// A context that is **already** done is what a real bound hitting looks like
// from inside the function, with none of the timing that would make the test
// flaky. What it pins is the division the doc comment draws: the *waiting* is
// bounded and the *work* is not, so the rooms are closed even on the path that
// gives up — which the second shutdown proves by finishing immediately.
//
// ⚠️ It also pins the message. A shutdown that reported only "context deadline
// exceeded" would tell a host nothing it could act on; the numbers are what say
// whether a match is stuck or a socket is.
func TestShutdownGivesUpAndNamesWhatItWasWaitingFor(t *testing.T) {
	held := listening(t, Timings{})
	dependencies := deps(t)
	code := held.open(t, config(9, 1, 60), dependencies)
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "the.host", ""), dependencies.Books)
	if tables := held.server.Tables(); tables != 1 {
		t.Fatalf("the server holds %d tables, want 1: nothing connected, so this test measures nothing", tables)
	}

	expired, cancel := context.WithCancel(context.Background())
	cancel()
	err := held.server.Shutdown(expired)
	if err == nil {
		t.Fatal("a shutdown on a context that was already done reported no error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("the refusal does not carry the context's own error, so errors.Is cannot ask which bound it was: %v", err)
	}
	// The one thing a host can act on. "1 room" is this room; the connected count
	// is whichever of the two the shutdown had got to, so only the room count is
	// asserted by value.
	if !strings.Contains(err.Error(), "1 room(s)") && !strings.Contains(err.Error(), "0 room(s)") {
		t.Errorf("the refusal names no room count, so it says only that a bound was hit: %v", err)
	}

	// ⚠️ The work was done anyway. This is the assertion that separates "gave up
	// on waiting" from "gave up", and it is the whole reason CloseAll is outside
	// the bound.
	ctx, done := context.WithTimeout(context.Background(), theWholeShutdown)
	defer done()
	if err := held.server.Shutdown(ctx); err != nil {
		t.Fatalf("the second shutdown, on a live context: %v", err)
	}
	if running := held.rooms.Running(); running != 0 {
		t.Errorf("%d room goroutines are still owed an end", running)
	}
}

// TestShutdownClosesEveryRoomAndThenMeasuresThem is the structural half, and it
// exists because the behavioural test above cannot see one of the four steps.
//
// ⚠️ **Registry.Wait closes nothing** — its own comment says so, and that is what
// makes it a measurement rather than a tidy-up: a goroutine left behind hangs it
// instead of being quietly collected. So a shutdown is CloseAll **then** Wait, in
// that order and as two calls, and the order is the claim. Merging them, or
// dropping the Wait because a poll happens to converge without it, loses exactly
// the property the pair was built to have.
//
// ⚠️ **What this test can see is that the calls are written, in that order.** It
// cannot see that Wait blocked, and nothing can without asserting on timing. That
// is the honest limit of it, and it is why this is beside the behavioural test
// rather than instead of it.
func TestShutdownClosesEveryRoomAndThenMeasuresThem(t *testing.T) {
	// Every call, in source order, per function of this package's own source.
	calls := map[string][]string{}
	scanned := 0
	for _, name := range packageSources(t) {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		scanned++
		for _, declared := range file.Decls {
			function, isFunction := declared.(*ast.FuncDecl)
			if !isFunction || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				called, isCall := node.(*ast.CallExpr)
				if !isCall {
					return true
				}
				switch target := called.Fun.(type) {
				case *ast.SelectorExpr:
					calls[function.Name.Name] = append(calls[function.Name.Name], target.Sel.Name)
				case *ast.Ident:
					calls[function.Name.Name] = append(calls[function.Name.Name], target.Name)
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	shutdown, written := calls["Shutdown"]
	if !written {
		t.Fatal("this package declares no Shutdown, which is the whole of what this test measures")
	}
	// The four steps, in order, allowing anything else between them — what is
	// pinned is the sequence and not the absence of other work.
	for _, step := range []struct {
		call    string
		because string
	}{
		{"stopping", "a peer whose socket dies with no wire.Closed is a player staring at a dead connection"},
		{"CloseAll", "Registry.Wait closes nothing, so something has to ask the rooms to stop"},
		{"waited", "CloseAll asks and Wait measures; without the measurement nothing observes a goroutine left behind"},
		{"settling", "a table outlives its match, so Tables reaching nought is a separate reading from Running"},
	} {
		at := indexOfCall(shutdown, step.call)
		if at < 0 {
			t.Errorf("Shutdown never calls %s: %s", step.call, step.because)
			continue
		}
		shutdown = shutdown[at+1:]
	}
	// And the measurement is really the registry's, one hop down. Inlining Wait's
	// job into the poll would leave the order above intact and lose the claim.
	if indexOfCall(calls["waited"], "Wait") < 0 {
		t.Error("Shutdown's bounded step does not call Registry.Wait, so nothing waits for a room goroutine to end; " +
			"the settling poll converges whether or not one was left behind")
	}
	if indexOfCall(calls["stopping"], "drop") < 0 {
		t.Error("the notify never closes a socket, so a shutdown depends on every peer choosing to leave")
	}
	t.Logf("scanned %d source files; Shutdown calls %d distinct steps", scanned, len(calls["Shutdown"]))
}

// indexOfCall is where a name first appears in a run of calls, or -1.
func indexOfCall(calls []string, wanted string) int {
	for at, name := range calls {
		if name == wanted {
			return at
		}
	}
	return -1
}

// TestBothPlayersAreAnnouncedAsTheyJoin holds Options.Joined, which exists
// because a join leaves **no other trace a caller can reach**: room.Reading
// carries the configuration, the seat being awaited and the result, and no seat
// occupancy at all — so a host binary wanting a line per player could otherwise
// only poll for the match starting, which is one line for two people.
//
// It asserts the seats *and* the names, in order, because a callback that fired
// twice with the same seat, or handed the guest's name to the host, would pass
// any assertion that only counted them.
func TestBothPlayersAreAnnouncedAsTheyJoin(t *testing.T) {
	held := listening(t, Timings{})
	dependencies := deps(t)
	arrived := make(chan string, 4)
	held.server.joined = func(code wire.RoomCode, seat wire.Seat, name string) {
		arrived <- string(code) + " " + string(seat) + " " + name
	}

	code := held.open(t, config(11, 1, 60), dependencies)
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "ha", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "khach", ""), dependencies.Books)

	want := []string{
		string(code) + " " + string(wire.SeatHost) + " ha",
		string(code) + " " + string(wire.SeatGuest) + " khach",
	}
	for _, wanted := range want {
		select {
		case got := <-arrived:
			if got != wanted {
				t.Errorf("a join was announced as %q, want %q", got, wanted)
			}
		case <-time.After(theWholeShutdown):
			t.Fatalf("no join was announced; %q never arrived", wanted)
		}
	}
	// A refused join hands out no seat, so it must reach no callback: a host
	// printing "somebody joined" for a stranger who was turned away would be
	// printing the opposite of what happened.
	if _, err := Dial(context.Background(), code,
		hello(t, theGuestSquad(t, dependencies.Characters), "third", ""), dependencies.Books, Timings{}); err == nil {
		t.Fatal("a third client joined a room with two seats")
	}
	select {
	case got := <-arrived:
		t.Errorf("a refused join was announced as %q", got)
	case <-time.After(200 * time.Millisecond):
	}
}
