// ⚠️ The tests in this package are **in** package socket rather than beside it,
// which is the opposite of internal/room's choice and is a decision rather than
// laziness. The room is a state machine, so everything it does is observable
// through the messages it hands back and a test reaching inside would be
// measuring the implementation. A transport is not: two of the claims this
// package has to make cannot be produced from outside at all.
//
//   - **A late timeout.** "A timer that fires while an answer is in flight is
//     refused without dropping anybody" is a race, and driving it from outside
//     means sleeping and hoping. Calling the reporter the timer calls, at a
//     moment the test chose, is the same input with the timing made certain.
//   - **The count of them.** table.late leaves no other trace — nothing is sent
//     and nothing is closed — so a test asserting the path was taken has to read
//     it, exactly as internal/room's own tests read Room.Skipped.
//
// Everything else here goes over a real loopback listener through the exported
// surface, which is what the end-to-end test is for.
package socket

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// fixturePassword is named rather than written into a struct literal, so a
// secret scanner reading this file finds a constant with an obvious purpose
// instead of a bare string beside a field called Password.
//
// ⚠️ It is also what TestAWrongPasswordIsRefusedAndNeverPrinted greps the error
// sink for, so it has to be a string that could not appear for any other reason.
const fixturePassword = "the-cat-sat-on-the-mat"

// fixtureBuild is the build string this binary would announce. It is printed and
// never acted on, which is what internal/wire's version test pins.
const fixtureBuild = "socket-test"

// theWholeMatch is how long a test will wait for a match over a loopback socket
// before calling it hung. A bo3 of the shipped 3v3 is a hundred-odd decisions
// and runs in well under a second in process; the margin is for a loaded
// machine, and the point of the bound is that a transport that stops making
// progress fails the suite rather than hanging it.
const theWholeMatch = 60 * time.Second

// deps is the parsed data and the version a room is handed. The version carries
// the **real** data digest, because the gate's job is to compare a peer's
// against this binary's and a hand-made one would measure two invented values.
func deps(t *testing.T) room.Deps {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	version, err := wire.Local(fixtureBuild)
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	return room.Deps{Books: books, Characters: characters, Version: version}
}

// config is a room at 3v3 from one seed, with an allowance a test can shorten.
func config(matchSeed uint64, battles, allowance int) room.Config {
	return room.Config{
		Format:    wire.Format3v3,
		Battles:   battles,
		Allowance: allowance,
		Seed:      matchSeed,
		TurnCap:   room.DefaultTurnCap,
	}
}

// theThreeSlots are the formation cells the fixture squads stand in: one per
// rank, so a squad holds a front, a middle and a back.
var theThreeSlots = []hex.Offset{{Col: 0, Row: 1}, {Col: 1, Row: 1}, {Col: 2, Row: 1}}

// theFiveSlots are enough cells for the largest squad a room offers, which the
// one test that needs a 5v5 uses. hex.MaxTeamSize is five against nine
// formation slots, so any five distinct cells will do and these are the middle
// two ranks.
var theFiveSlots = []hex.Offset{
	{Col: 0, Row: 0}, {Col: 0, Row: 1}, {Col: 1, Row: 0}, {Col: 1, Row: 1}, {Col: 2, Row: 1},
}

// squadOf builds a legal 3v3 squad out of shipped characters, building every
// part of "legal" itself rather than borrowing a shipped squad — the shipped
// ones are two units, and a fixture out of internal/seed/data would move with
// the balance.
func squadOf(t *testing.T, characters *cast.Book, id string, wanted ...string) placement.Squad {
	t.Helper()
	return squadOn(t, characters, theThreeSlots, id, wanted...)
}

// squadOn is the same over a given set of cells, for the format that needs five.
func squadOn(t *testing.T, characters *cast.Book, slots []hex.Offset, id string, wanted ...string) placement.Squad {
	t.Helper()
	if len(wanted) > len(slots) {
		t.Fatalf("squadOn was asked for %d units and has %d slots", len(wanted), len(slots))
	}
	squad := placement.Squad{ID: id, Name: id}
	for index, name := range wanted {
		character, known := characters.Get(name)
		if !known {
			t.Fatalf("no character is called %q", name)
		}
		leaves, err := character.Stages.Leaves()
		if err != nil {
			t.Fatalf("the tips of %q: %v", name, err)
		}
		stage := leaves[0].Name
		squad.Units = append(squad.Units, placement.Placement{
			ID:        name + "@" + stage,
			Character: name,
			Level:     progression.LevelCap,
			Stage:     stage,
			Slot:      slots[index],
			Skills:    upTo(character.SkillsAt(progression.LevelCap, stage), cast.SkillSlots),
			Passives:  upTo(character.PassivesAt(progression.LevelCap, stage), cast.TraitSlots),
		})
	}
	return squad
}

func upTo(available []string, slots int) []string {
	if len(available) > slots {
		return available[:slots:slots]
	}
	return available[:len(available):len(available)]
}

// theHostSquad and theGuestSquad are two different 3v3 sides, and the difference
// is the measurement rather than variety: a mirror makes the two halves of a
// battle interchangeable, so nothing could see a transport that handed one
// client the other's side.
func theHostSquad(t *testing.T, characters *cast.Book) placement.Squad {
	// ⚠️ **Not pokemon.machop, and the reason is a property this squad is read
	// for.** TestASkippedPromptStartsNoClockOverASocket needs a match with skipped
	// prompts in it and fatals when there are none. Machop's learnset now opens on
	// a cooldownless self-aimed builder, so it is never out of options and never
	// skips — measured: the match came back with Skipped == 0 the day that kit
	// landed. Poliwag stands in as a unit whose kit can still run dry.
	return squadOf(t, characters, "host.squad",
		"pokemon.bulbasaur", "pokemon.poliwag", "pokemon.gastly")
}

func theGuestSquad(t *testing.T, characters *cast.Book) placement.Squad {
	return squadOf(t, characters, "guest.squad",
		"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa")
}

// hello is a peer announcing itself with the version this binary would.
func hello(t *testing.T, squad placement.Squad, name string, password wire.Password) wire.Hello {
	t.Helper()
	version, err := wire.Local(fixtureBuild)
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	return wire.Hello{Version: version, Squad: squad, Name: name, Password: password}
}

// listener is a server over a real loopback listener, with the address its room
// codes will carry.
type listener struct {
	server   *Server
	rooms    *room.Registry
	http     *httptest.Server
	at       netip.AddrPort
	timings  Timings
	failures *sink
	endings  chan ending
}

// ending is a match the transport saw finish, as the room's own last reading.
type ending struct {
	code    wire.RoomCode
	reading room.Reading
}

// listening starts a registry, a server over it and a loopback listener, and
// tears all three down in the order a shutdown takes: the sockets, then the
// rooms, then the wait that measures whether any goroutine was left behind.
func listening(t *testing.T, timings Timings) *listener {
	t.Helper()
	held := &listener{
		rooms:    room.NewRegistry(),
		timings:  timings,
		failures: newSink(),
		// Buffered so that a match ending never blocks the goroutine that
		// noticed, which is a connection's own.
		endings: make(chan ending, 8),
	}
	held.server = NewServer(held.rooms, Options{
		Timings: timings,
		Report:  held.failures.take,
		Finished: func(code wire.RoomCode, reading room.Reading) {
			held.endings <- ending{code: code, reading: reading}
		},
	})
	held.http = httptest.NewServer(held.server)
	at, err := netip.ParseAddrPort(held.http.Listener.Addr().String())
	if err != nil {
		t.Fatalf("read the listener's address: %v", err)
	}
	held.at = netip.AddrPortFrom(at.Addr().Unmap(), at.Port())
	t.Cleanup(func() {
		held.http.Close()
		held.rooms.CloseAll()
		held.rooms.Wait()
	})
	return held
}

// open starts a room behind this listener and hands back the code a player would
// paste.
func (l *listener) open(t *testing.T, configuration room.Config, dependencies room.Deps) wire.RoomCode {
	t.Helper()
	code, err := l.rooms.Open(l.at, configuration, dependencies)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	return code
}

// dial is a client joining by code, and fails the test on a refusal — the tests
// that are *about* a refusal call Dial themselves.
func (l *listener) dial(t *testing.T, code wire.RoomCode, joining wire.Hello, books battle.Books) *Client {
	t.Helper()
	client, err := Dial(context.Background(), code, joining, books, ClientOptions{Timings: l.timings})
	if err != nil {
		t.Fatalf("dial room %s: %v", code, err)
	}
	t.Cleanup(client.Close)
	return client
}

// finished is the reading of the next match to end, or a failure if none does.
func (l *listener) finished(t *testing.T) ending {
	t.Helper()
	select {
	case done := <-l.endings:
		return done
	case <-time.After(theWholeMatch):
		t.Fatalf("no match finished inside %s", theWholeMatch)
		return ending{}
	}
}

// emptied waits for the server to let go of every table and fails if it does
// not.
//
// ⚠️ It **polls**, and it is a poll for the same reason Server.Shutdown's last
// step is one: a table is released by the connection goroutine that held it, on
// its way out, and nothing signals that. A client returning from Play means that
// *client* is done; the server's own connection goroutine tears its table down a
// moment later when the socket finishes closing.
//
// ⚠️ This is deliberately **not** Shutdown, and the difference is what these tests
// measure. Shutdown makes the tables go away by closing them; this waits for a
// match that ended **by itself** to let go of them, which is the ordinary path and
// the one no shutdown is involved in. Calling Shutdown here would replace that
// measurement with a measurement of Shutdown, which has its own test.
//
// The race detector is what found this: without it the two happen to interleave
// the other way round on every run.
func (l *listener) emptied(t *testing.T) {
	t.Helper()
	for waited := 0; waited < 200; waited++ {
		if l.server.Tables() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("the server still holds %d tables two seconds after the match", l.server.Tables())
}

// tableFor is the transport's own record for a room, which the two tests that
// drive a timer by hand need. → the note at the head of this file.
func (l *listener) tableFor(t *testing.T, code wire.RoomCode) *table {
	t.Helper()
	l.server.mu.Lock()
	defer l.server.mu.Unlock()
	entry, held := l.server.tables[code]
	if !held {
		t.Fatalf("the server holds no connections for room %s", code)
	}
	return entry
}

// rating is a client answering off its **own** mirror of the battle, which is
// what makes a whole match play out over a socket with nobody typing.
//
// ⚠️ The battle is read at each call rather than captured, because a series
// fights several and the mirror builds a new one per wire.Start.
func rating(client *Client) battle.Chooser {
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		fight := client.Mirror().Battle()
		if fight == nil {
			return battle.Choice{}, false
		}
		return fight.Suggest(prompt)
	}
}

// played is one client's Play loop run in the background, so a test can start
// both and then wait for both.
type played struct {
	err  error
	done chan struct{}
}

func play(ctx context.Context, client *Client, choose battle.Chooser) *played {
	running := &played{done: make(chan struct{})}
	go func() {
		defer close(running.done)
		running.err = client.Play(ctx, choose)
	}()
	return running
}

// wait blocks until the loop returns and reports what it returned.
func (p *played) wait(t *testing.T, who string) error {
	t.Helper()
	select {
	case <-p.done:
		return p.err
	case <-time.After(theWholeMatch):
		t.Fatalf("%s was still playing after %s", who, theWholeMatch)
		return nil
	}
}

// sink collects the errors the transport reports, which is the only output this
// package has — there is no logger in it. → TestTheTransportCannotPrint.
type sink struct {
	said chan string
}

func newSink() *sink { return &sink{said: make(chan string, 64)} }

func (s *sink) take(err error) {
	if err == nil {
		return
	}
	select {
	case s.said <- err.Error():
	default:
	}
}

// everything is every error reported so far, drained.
func (s *sink) everything() []string {
	var out []string
	for {
		select {
		case said := <-s.said:
			out = append(out, said)
		default:
			return out
		}
	}
}

// stepped is a decision the fixture counted, for a test that wants to act at a
// chosen point in a match rather than at the end of one.
//
// It wraps a chooser and signals once, on the nth decision.
//
// ⚠️ **This note used to say that a test may not read a client's Mirror from
// another goroutine, and that has changed: a Mirror carries an RWMutex now, so
// every accessor and Mirror.Read are safe from anywhere.** What changed it was
// not the tests — it was the full-screen client, which draws a battle its own
// Play goroutine is stepping, and a lock was the only answer that did not turn
// the client into the thin one README.md refuses.
//
// The wrapped chooser stays, and for a reason the lock does not cover: it gives
// a test a **turn-ordered** signal rather than merely a safe one. "Read the
// mirror when the third decision is being taken" is a thing a poll cannot say —
// a poll can only report what it happened to see — and a fixture that slept
// instead would be a fixture that fails on a loaded machine.
func stepped(choose battle.Chooser, nth int) (battle.Chooser, <-chan struct{}) {
	reached := make(chan struct{})
	taken := 0
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		taken++
		if taken == nth {
			close(reached)
		}
		return choose(prompt)
	}, reached
}

// reachedOr fails the test when a fixture's signal never arrives.
func reachedOr(t *testing.T, reached <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-reached:
	case <-time.After(theWholeMatch):
		t.Fatalf("%s never happened inside %s", what, theWholeMatch)
	}
}

// side is a printable note for a client's own reading of a battle, for a log
// line that has to name what disagreed.
func side(one Fought) string {
	return fmt.Sprintf("battle %d on the %s side, outcome %s", one.Battle, one.Side, one.Outcome)
}
