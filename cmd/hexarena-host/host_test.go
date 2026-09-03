package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// fixturePassword is named rather than written into a struct literal, so a
// secret scanner reading this file finds a constant with an obvious purpose
// instead of a bare string beside a field called Password. It is also what
// TestARoomPasswordIsNeverPrintedByTheHost greps every printed byte for, so it
// has to be a string that could not appear for any other reason.
const fixturePassword = "the-cat-sat-on-the-mat"

// theWholeShutdown is how long a test waits for this binary to stop before
// calling it hung. Everything it waits on is local; the margin is for a loaded
// machine.
const theWholeShutdown = 30 * time.Second

// documented is the address a test advertises when nothing is going to dial it:
// 192.0.2.7, out of TEST-NET-1, which no real host may use.
//
// ⚠️ It is fixed and deliberately **not** this machine's. What a room code
// carries is a decision this binary makes, and a fixture that read the real
// address would make every assertion about a code depend on whatever network the
// suite happened to run on.
const documented = "192.0.2.7"

// dialable is the address a test advertises when a client is going to connect,
// and it is loopback on purpose.
//
// ⚠️ 127.0.0.1 is exactly what pick refuses, and handing it to open anyway is not
// a contradiction: the loopback rule is the **picker's**, about which address a
// human should be given to retype, and open takes an address rather than choosing
// one. What these tests need is a code a client in this process can actually
// dial — advertising a documentation address gave a code that decoded fine and
// then hung until the dial timed out, measured at 30 seconds a test. Which is
// itself the argument for the picker's rule, from the other end.
const dialableAt = "127.0.0.1"

// paper is a buffer and the lock this binary writes to it through.
//
// ⚠️ It is a lock and not a plain bytes.Buffer, because the transport writes to
// this binary's stdout from **a connection's own goroutine** — socket.Options.Joined
// fires there — while the test reads it. Measured: the race detector caught
// exactly that on the join test's first run, which is why ./cmd/hexarena-host is
// on the -race line of `make check` and why screen exists at all.
type paper struct {
	on   *screen
	held *bytes.Buffer
}

func newPaper() *paper {
	held := &bytes.Buffer{}
	return &paper{on: newScreen(held), held: held}
}

// said is everything written so far, read under the same lock the writes take.
func (p *paper) said() string {
	var out string
	p.on.withLock(func(io.Writer) { out = p.held.String() })
	return out
}

// hosting is the binary's own open, over a given advertised address and a port a
// test chose, torn down whatever the test does.
func hosting(t *testing.T, chosen settings, advertised string, out, errs io.Writer) *hosted {
	t.Helper()
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	held, err := open(chosen, netip.MustParseAddr(advertised), dependencies, out, errs)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	t.Cleanup(func() {
		if err := held.stop(); err != nil {
			t.Errorf("stop the host: %v", err)
		}
	})
	return held
}

// aRoom is the flags a test hosts under: an ephemeral port, so nothing in the
// suite fights the default or another session on this machine for 13579.
func aRoom() settings {
	return settings{port: 0, format: int(wire.Format3v3), battles: 1,
		allowance: room.DefaultAllowance, turns: room.DefaultTurnCap, seed: 11}
}

// TestTheCodeCarriesThePortThatWasActuallyBound is the ordering requirement, and
// the ⚠️ **-port 0 is the measurement rather than a convenience**.
//
// A room code carries the port, and room.Registry.Open takes the address the code
// will name — so the port has to be known before the room exists, which means
// binding the listener first. Get it the other way round and the code carries
// whatever was in the flag.
//
// ⚠️ **At the default port that bug is invisible.** With -port 13579, opening the
// room before binding still produces a code carrying 13579, which still works, so
// a test run at the default passes whether or not the order is right and measures
// nothing at all — the exact shape of vacuous pass this repository keeps a record
// of. So this drives -port 0, where the flag's value and the bound port differ by
// construction, and asserts the code's port is the bound one **and is not
// nought**. Reversing the two calls in open reddens it; verified by doing so.
func TestTheCodeCarriesThePortThatWasActuallyBound(t *testing.T) {
	out, errs := newPaper(), newPaper()
	held := hosting(t, aRoom(), documented, out.on, errs.on)

	bound, ok := held.listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the listener reported a %T as its address", held.listener.Addr())
	}
	if bound.Port == 0 {
		t.Fatal("the listener reports port 0, so this test cannot tell a bound port from a flag's value")
	}
	at, which, err := held.code.Decode()
	if err != nil {
		t.Fatalf("decode the code this host printed (%s): %v", held.code, err)
	}
	if at.Port() == 0 {
		t.Errorf("the code %s carries port 0, so a player pasting it would dial nowhere: "+
			"the room was opened before the listener was bound", held.code)
	}
	// #nosec G115 -- a TCP port is a uint16 and the field is an int holding one.
	if at.Port() != uint16(bound.Port) {
		t.Errorf("the code %s carries port %d and the listener is on %d", held.code, at.Port(), bound.Port)
	}
	if at.Addr().String() != documented {
		t.Errorf("the code %s carries the address %s, want the advertised one", held.code, at.Addr())
	}
	if which != 0 {
		t.Errorf("the first room of a process is room %d, want 0", which)
	}
}

// TestAPortAlreadyInUseSaysSoAndSaysWhatToDo holds the cost of a fixed default.
//
// ⚠️ With an ephemeral port this failure could not happen; with DefaultPort it is
// ordinary — a second host on one machine, or the last match's process still
// running. The syscall's own words are `listen tcp :13579: bind: address already
// in use`, which names neither which of those it is nor anything the host can do,
// so the message is this binary's.
//
// The port is one the test bound itself rather than DefaultPort: the suite must
// not fight a real host, or another session on this machine, for 13579.
func TestAPortAlreadyInUseSaysSoAndSaysWhatToDo(t *testing.T) {
	taken, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("bind a port to hold: %v", err)
	}
	defer func() { _ = taken.Close() }()
	bound, ok := taken.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the listener reported a %T as its address", taken.Addr())
	}

	held, err := listen(bound.Port)
	if err == nil {
		_ = held.Close()
		t.Fatal("a port another listener is holding was bound a second time")
	}
	for _, wanted := range []string{
		fmt.Sprintf("%d", bound.Port), // which port
		"already in use",              // what happened, in words rather than errno
		"-port",                       // and what to do about it
	} {
		if !strings.Contains(err.Error(), wanted) {
			t.Errorf("the refusal does not mention %q, so a host reading it cannot act on it: %v", wanted, err)
		}
	}
	// And the ordinary failure is still passed through rather than swallowed: a
	// port nobody may bind is a different problem and must not read as this one.
	if _, err := listen(-1); err == nil {
		t.Error("port -1 was accepted")
	} else if strings.Contains(err.Error(), "already in use") {
		t.Errorf("an out-of-range port is reported as a port in use: %v", err)
	}
}

// TestARoomPasswordIsNeverPrintedByTheHost is this binary's half of the promise
// internal/socket already keeps, and it is here because the transport's test
// cannot see it: internal/socket reaches no writer at all (log, log/slog and os
// are import-banned there), and this binary is the thing that prints.
//
// It drives a **real** password through every writing path there is — the flag,
// the banner, the error sink, a refused join over a real socket — and greps the
// bytes. It also renders every value that holds one under %v, %+v and %#v,
// because the redaction that makes those safe is wire.Password's own String and
// GoString, and a field promoted to a plain string somewhere would pass every
// other test in the repository.
func TestARoomPasswordIsNeverPrintedByTheHost(t *testing.T) {
	out, errs := newPaper(), newPaper()
	chosen := aRoom()
	chosen.password = wire.Password(fixturePassword)
	held := hosting(t, chosen, dialableAt, out.on, errs.on)
	banner(held, "was told by -advertise", out.on)

	// A joiner with the wrong password, over a real socket. The room refuses at
	// the gate — before the squad is even looked at — and this is the path where
	// the bytes of a hello are nearest to a writer.
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	joining := wire.Hello{
		Version:  dependencies.Version,
		Squad:    squadOf(t, dependencies.Characters, "joiner"),
		Name:     "stranger",
		Password: wire.Password(fixturePassword + "-wrong"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), theWholeShutdown)
	defer cancel()
	client, err := socket.Dial(ctx, held.code, joining, dependencies.Books, socket.ClientOptions{})
	if err == nil {
		client.Close()
		t.Fatal("a wrong password got past the gate")
	}
	fmt.Fprintf(errs.on, "hexarena-host: %v\n", err)

	// A hello carrying the **right** one, formatted every way a careless line
	// might format it. Nothing sends this; it is the shape a log line takes.
	joining.Password = wire.Password(fixturePassword)
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		fmt.Fprintf(out.on, format+"\n", joining)
		fmt.Fprintf(out.on, format+"\n", chosen)
		fmt.Fprintf(out.on, format+"\n", held.config)
		fmt.Fprintf(out.on, format+"\n", chosen.password)
	}

	// The flag path, end to end: a run that fails its configuration check with a
	// password on the command line. Everything it says goes through the two
	// writers and the returned error.
	ranOut, ranErrs := newPaper(), newPaper()
	runErr := run([]string{"-password", fixturePassword, "-battles", "2"}, ranOut.on, ranErrs.on)
	if runErr == nil {
		t.Error("a series of two battles was accepted; room.Config.Validate refuses a bo2 by name")
	}

	printed := map[string]string{
		"stdout":               out.said(),
		"stderr":               errs.said(),
		"stdout of a bad run":  ranOut.said(),
		"stderr of a bad run":  ranErrs.said(),
		"the error a run gave": fmt.Sprint(runErr),
	}
	for where, said := range printed {
		if strings.Contains(said, fixturePassword) {
			t.Errorf("the room password reached %s: %s", where, said)
		}
	}
	// ⚠️ Non-vacuity, and it is the assertion this test would otherwise be
	// missing: an empty buffer passes a grep, so a fixture that quietly stopped
	// writing would leave this test green for ever.
	//
	// It is only over the three that **do** carry output. The two run buffers are
	// deliberately not among them: `run` returns its refusal rather than printing
	// it — main is what writes it to stderr — so those two are legitimately empty,
	// and the error string is what carries that path's bytes. Measured: asserting
	// on all five failed on exactly those two.
	for _, where := range []string{"stdout", "stderr", "the error a run gave"} {
		if strings.TrimSpace(printed[where]) == "" {
			t.Errorf("nothing was written to %s, so the grep over it measured nothing", where)
		}
	}
	// And the banner says a password is set, which is what a player who cannot
	// get in needs — the redaction must not turn into silence.
	if !strings.Contains(out.said(), "set (players will need it)") {
		t.Error("the banner does not say a password is set, so a refused player has nothing to go on")
	}
}

// TestAJoinIsAnnouncedAndAFinishedMatchIsReported drives the two things this
// binary prints that nothing else can: the line per player, and the result.
//
// ⚠️ The join half goes over a **real socket**, because it is the wiring between
// socket.Options.Joined and this binary's writer that is being measured, and a
// hand-called callback would measure the fmt.Fprintf and not the wiring. The
// result half is driven from a hand-made room.Reading instead, deliberately:
// playing a whole match here would be a second copy of internal/socket's
// end-to-end test, and what is this binary's own is the *wording* of a reading —
// which a made-up reading exercises exactly, and for the endings a real 3v3 would
// not happen to produce.
func TestAJoinIsAnnouncedAndAFinishedMatchIsReported(t *testing.T) {
	out, errs := newPaper(), newPaper()
	held := hosting(t, aRoom(), dialableAt, out.on, errs.on)
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	joining := wire.Hello{
		Version: dependencies.Version,
		Squad:   squadOf(t, dependencies.Characters, "joiner"),
		Name:    "Bảo",
	}
	ctx, cancel := context.WithTimeout(context.Background(), theWholeShutdown)
	defer cancel()
	client, err := socket.Dial(ctx, held.code, joining, dependencies.Books, socket.ClientOptions{})
	if err != nil {
		t.Fatalf("join the room this host opened: %v", err)
	}
	defer client.Close()
	if said := out.said(); !strings.Contains(said, "Bảo joined as host") {
		t.Errorf("a player joined and the host printed %q", said)
	}

	for _, each := range []struct {
		what    string
		reading room.Reading
		wanted  []string
	}{
		{
			what: "a series somebody won",
			reading: room.Reading{Finished: true, Result: room.Result{
				Verdict: room.VerdictWon, Winner: wire.SeatGuest, Wins: [2]int{1, 2}, Battles: 3}},
			wanted: []string{"won", "3 battle(s)", "guest took it, 1–2"},
		},
		{
			what: "a series nobody won",
			reading: room.Reading{Finished: true, Result: room.Result{
				Verdict: room.VerdictDrawn, Wins: [2]int{0, 0}, Battles: 1}},
			wanted: []string{"drawn", "nobody took it, 0–0"},
		},
		{
			// ⚠️ Not a loss and not a forfeit. Nobody wins a match nobody played
			// out, so this line must not read as a win for the one still there.
			what: "a match somebody walked away from",
			reading: room.Reading{Finished: true, Result: room.Result{
				Verdict: room.VerdictAbandoned, Departed: wire.SeatHost, Battles: 1}},
			wanted: []string{"abandoned", "host went away, so nobody took it"},
		},
		{
			what: "a battle the turn cap stopped",
			reading: room.Reading{Finished: true,
				Result: room.Result{Verdict: room.VerdictDrawn, Battles: 1},
				Played: []room.BattleResult{{Battle: 1, Home: wire.SeatHost, Seed: 99, Capped: true}}},
			wanted: []string{"battle 1:", "nobody took it", "seed 99", "stopped at the turn cap"},
		},
	} {
		t.Run(each.what, func(t *testing.T) {
			var said bytes.Buffer
			report(each.reading, &said)
			for _, wanted := range each.wanted {
				if !strings.Contains(said.String(), wanted) {
					t.Errorf("the result does not say %q:\n%s", wanted, said.String())
				}
			}
			if strings.Contains(said.String(), " ,") || strings.Contains(said.String(), "  took") {
				t.Errorf("the result has a gap where a seat should be, so a zero Seat reached the screen:\n%s", said.String())
			}
		})
	}
}

// TestTheHostStopsWhenTheMatchDoes holds the first of the two ways this process
// ends: the match finishing prints the result and shuts everything down, with no
// signal and nobody typing.
//
// ⚠️ It asserts the **listener is released**, by binding the same port again
// afterwards. A shutdown that printed a result and left the port held would pass
// any assertion about what was written, and would be the failure a host meets the
// next time they start one — which, at a fixed default port, is immediately.
func TestTheHostStopsWhenTheMatchDoes(t *testing.T) {
	out, errs := newPaper(), newPaper()
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	held, err := open(aRoom(), netip.MustParseAddr(documented), dependencies, out.on, errs.on)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	bound, ok := held.listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the listener reported a %T as its address", held.listener.Addr())
	}

	held.finished <- room.Reading{Finished: true, Result: room.Result{
		Verdict: room.VerdictWon, Winner: wire.SeatHost, Wins: [2]int{1, 0}, Battles: 1}}

	done := make(chan error, 1)
	go func() { done <- held.serve(out.on, errs.on) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v; a match ending is not a failure", err)
		}
	case <-time.After(theWholeShutdown):
		t.Fatalf("the host was still serving %s after the match ended", theWholeShutdown)
	}
	if said := out.said(); !strings.Contains(said, "host took it") {
		t.Errorf("the match ended and the host printed %q", said)
	}
	if said := errs.said(); strings.TrimSpace(said) != "" {
		t.Errorf("a clean shutdown wrote to stderr: %s", said)
	}
	if running := held.rooms.Running(); running != 0 {
		t.Errorf("%d room goroutines are still owed an end after the host stopped", running)
	}

	// The port is free again. This is the assertion a "did it print the right
	// thing" test cannot make, and it is the one a host notices.
	again, err := net.Listen("tcp", ":"+fmt.Sprint(bound.Port))
	if err != nil {
		t.Fatalf("the host stopped and did not let go of port %d: %v", bound.Port, err)
	}
	_ = again.Close()
}

// TestTheConfigurationIsRefusedInTheRoomsOwnWords holds that this binary adds no
// second wording of a rule it does not own.
//
// ⚠️ "a series of 2 battles is even, and an even series has to invent a rule for
// a 1–1" is room.Config.Validate explaining a **design decision**, and a
// paraphrase here would be a second place to keep that true. So the check is that
// the refusal arrives unchanged.
func TestTheConfigurationIsRefusedInTheRoomsOwnWords(t *testing.T) {
	for _, each := range []struct {
		what     string
		flags    []string
		fragment string
	}{
		{"a best of two", []string{"-battles", "2"}, "an even series has to invent a rule"},
		{"a format nobody plays", []string{"-format", "4"}, "not a format this game offers"},
		{"no time to think", []string{"-allowance", "0"}, "gives a player no turn to take"},
		{"a cap of nothing", []string{"-turns", "0"}, "ends every battle before it starts"},
	} {
		t.Run(each.what, func(t *testing.T) {
			out, errs := newPaper(), newPaper()
			err := run(append(each.flags, "-advertise", documented), out.on, errs.on)
			if err == nil {
				t.Fatalf("%s was accepted", each.what)
			}
			if !strings.Contains(err.Error(), each.fragment) {
				t.Errorf("the refusal is not the room's own words (%q): %v", each.fragment, err)
			}
		})
	}
}

// TestABadFlagIsReportedOnceAndHelpIsNotAFailure holds two things about the
// command line that are easy to get wrong in opposite directions.
//
// ⚠️ **`flag` with ContinueOnError prints the error AND the usage itself** before
// it hands the error back, so a main that prints the returned error puts the same
// sentence once above a screen of usage and once below it. Measured:
// `hexarena-host -nonsense` said "flag provided but not defined" at both ends.
// The fix is a sentinel, and this is what keeps it — a plain grep for the count,
// because "it reads better now" is not a test.
//
// The other direction is `-h`, which `flag` reports as an error and which is not
// one: a host asking what the flags are got what they asked for, and exiting 1 on
// it breaks any script that checks.
func TestABadFlagIsReportedOnceAndHelpIsNotAFailure(t *testing.T) {
	out, errs := newPaper(), newPaper()
	err := run([]string{"-nonsense"}, out.on, errs.on)
	if err == nil {
		t.Fatal("a flag this binary does not have was accepted")
	}
	if !errors.Is(err, errSaid) {
		t.Errorf("a parse failure is not marked as already reported, so main would print it a second time: %v", err)
	}
	const complaint = "flag provided but not defined"
	if said := strings.Count(errs.said(), complaint); said != 1 {
		t.Errorf("%q appears %d times in what the user was shown, want 1:\n%s", complaint, said, errs.said())
	}
	// And the usage really was printed, or the count above would be measuring an
	// empty buffer.
	if !strings.Contains(errs.said(), "-advertise") {
		t.Error("the usage was not printed beside the complaint, so this test counted nothing")
	}

	asked, askedErrs := newPaper(), newPaper()
	if err := run([]string{"-h"}, asked.on, askedErrs.on); err != nil {
		t.Errorf("-h reported %v; asking what the flags are is not a failure", err)
	}
	if !strings.Contains(askedErrs.said(), "visible to every other process") {
		t.Error("the -h text does not say a password given as a flag is visible in ps, " +
			"which is the one thing a reader of that flag has to be told")
	}
}

// TestASeedIsDrawnAndPrinted holds the one number a match cannot be reproduced
// without. ⚠️ -seed 0 means "draw one", so a drawn seed that was not printed
// would be a match nobody could replay — and the drawing is in main rather than
// in the engine, because everything under internal/core is a pure function of its
// inputs and drawing a number out of the air is what it may not do.
func TestASeedIsDrawnAndPrinted(t *testing.T) {
	first, err := drawSeed()
	if err != nil {
		t.Fatalf("draw a seed: %v", err)
	}
	second, err := drawSeed()
	if err != nil {
		t.Fatalf("draw a second seed: %v", err)
	}
	if first == second {
		t.Errorf("two drawn seeds are both %d, so every match this binary hosts would be the same match", first)
	}
	if first == 0 || second == 0 {
		t.Error("a drawn seed is 0, which is the value that means 'draw one'")
	}

	out, errs := newPaper(), newPaper()
	chosen := aRoom()
	chosen.seed = first
	held := hosting(t, chosen, documented, out.on, errs.on)
	banner(held, "was told by -advertise", out.on)
	if !strings.Contains(out.said(), fmt.Sprint(first)) {
		t.Errorf("the banner does not carry the seed %d, so the match could not be replayed:\n%s", first, out.said())
	}
}

// squadOf is a legal 3v3 out of shipped characters.
//
// ⚠️ It builds every part of "legal" itself rather than borrowing a squad out of
// internal/seed/data: the shipped ones are two units, and a fixture read from the
// data would move with the balance. The same reason internal/socket's own fixture
// builds one.
func squadOf(t *testing.T, characters *cast.Book, id string) placement.Squad {
	t.Helper()
	slots := []hex.Offset{{Col: 0, Row: 1}, {Col: 1, Row: 1}, {Col: 2, Row: 1}}
	squad := placement.Squad{ID: id, Name: id}
	for index, name := range []string{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"} {
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

// mustLoad keeps go vet quiet about an unused import when a test is commented
// out during development, and is a real assertion besides: this binary cannot
// start at all if the embedded data will not parse.
func TestTheEmbeddedDataLoads(t *testing.T) {
	if _, err := seed.Books(); err != nil {
		t.Fatalf("load the game data: %v", err)
	}
}
