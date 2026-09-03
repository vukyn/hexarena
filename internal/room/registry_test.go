package room_test

import (
	"fmt"
	"net/netip"
	"reflect"
	"sync"
	"testing"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// theOneListener is the address **every** room in these fixtures is opened
// behind, and the room byte is what changes per room — allocated by the
// registry, which is the shape the socket will actually have.
//
// ⚠️ **These fixtures used to be a port per room**, and that was the design
// record's open collision written down as its likely answer: a code carried four
// address bytes and two port bytes, so with one listener every room in a process
// encoded the same ten characters and the code named the process rather than the
// room. It is decided the other way — one listener, and wire.RoomCode carries a
// seventh byte for the room — so a test opens N rooms behind one address and
// keeps the N codes Open hands back. → the note on Registry.Open, and README.md
// § A room, and getting into one.
//
// Nothing is bound: the registry has no I/O, and a code is an address in a form
// a person can retype.
var theOneListener = netip.AddrPortFrom(netip.AddrFrom4([4]byte{127, 0, 0, 1}), 7100)

// codeFor is the code for one room behind theOneListener, for the two questions
// that need a code **without** opening a room under it: the stale one no process
// is running, and the order Open allocates in. Every test that opens a room uses
// the code Open returned instead, because the registry is what picks the byte.
func codeFor(t *testing.T, room uint8) wire.RoomCode {
	t.Helper()
	code, err := wire.EncodeRoom(theOneListener, room)
	if err != nil {
		t.Fatalf("encode the code of room %d: %v", room, err)
	}
	return code
}

// theSquadsInPlay is one pair of 3v3 squads per room, and they are **different
// squads with different seeds** for one reason: a registry that crossed two
// rooms' wires would pass every test in which the rooms are alike, because the
// result it handed back would be a result those inputs could have produced.
var theSquadsInPlay = [][2][3]string{
	{
		{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"},
		{"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"},
	},
	{
		{"pokemon.charmander", "pokemon.gastly", "pokemon.poliwag"},
		{"pokemon.bulbasaur", "pokemon.cleffa", "pokemon.machop"},
	},
	{
		{"pokemon.mew", "pokemon.squirtle", "pokemon.magnemite"},
		{"pokemon.mewtwo", "pokemon.poliwag", "pokemon.gastly"},
	},
	{
		{"pokemon.cleffa", "pokemon.magnemite", "pokemon.bulbasaur"},
		{"pokemon.mew", "pokemon.machop", "pokemon.charmander"},
	},
}

// board is the inputs one match needs, so that the **same driver** plays a match
// through a registry and through a bare room.
//
// That is what makes "the registry does not cross wires" measurable rather than
// asserted: the reference result is the same two squads and the same seed played
// alone, single-threaded, with no registry anywhere near it, and a crossed wire
// is a concurrent result that does not equal it.
type board interface {
	join(hello wire.Hello) (room.Answer, error)
	deliver(seat wire.Seat, body wire.Body) (room.Answer, error)
}

// aloneOn plays a bare *room.Room, and builds the Answer the registry would have
// built. It reads the room's accessors from the test's own goroutine, which is
// exactly what nothing may do to a room the registry owns — and is perfectly
// safe here, because this room is in no registry and has no goroutine.
type aloneOn struct{ playing *room.Room }

func (b aloneOn) join(hello wire.Hello) (room.Answer, error) {
	seat, out, err := b.playing.Join(hello)
	return room.Answer{Seat: seat, Out: out, Reading: b.read(), Known: true}, err
}

func (b aloneOn) deliver(seat wire.Seat, body wire.Body) (room.Answer, error) {
	out, err := b.playing.Deliver(seat, body)
	return room.Answer{Out: out, Reading: b.read(), Known: true}, err
}

func (b aloneOn) read() room.Reading {
	seat, waiting := b.playing.Awaiting()
	played := b.playing.Played()
	return room.Reading{
		Config:   b.playing.Config(),
		Awaiting: seat,
		Waiting:  waiting,
		Finished: b.playing.Finished(),
		Result:   b.playing.Result(),
		Played:   append(make([]room.BattleResult, 0, len(played)), played...),
		Skipped:  b.playing.Skipped(),
	}
}

// throughTheRegistry plays a room the registry owns, by code.
type throughTheRegistry struct {
	registry *room.Registry
	code     wire.RoomCode
}

func (b throughTheRegistry) join(hello wire.Hello) (room.Answer, error) {
	return b.registry.Join(b.code, hello)
}

func (b throughTheRegistry) deliver(seat wire.Seat, body wire.Body) (room.Answer, error) {
	return b.registry.Deliver(b.code, seat, body)
}

// outcome is what a driven match came to, which is what two runs of one pairing
// are compared on.
type outcome struct {
	reading room.Reading
	// steps is the decisions the match took.
	steps int
	// compared is how many digests the host's mirror checked, which is the
	// vacuity guard: a run that compared a handful did not fight a match.
	compared int
}

// squadPair builds the two squads of one room.
func squadPair(t *testing.T, dependencies room.Deps, index int) (placement.Squad, placement.Squad) {
	t.Helper()
	pair := theSquadsInPlay[index%len(theSquadsInPlay)]
	return squadOf(t, dependencies.Characters, fmt.Sprintf("host.%d", index), pair[0][:]...),
		squadOf(t, dependencies.Characters, fmt.Sprintf("guest.%d", index), pair[1][:]...)
}

// playMatch drives a whole match through a board with two fake clients, each
// answering off its own mirror of the battle through battle.Suggest.
//
// ⚠️ It reads whose turn it is off the **Answer** rather than by asking again,
// which is the shape the protocol forces: a room removes its own entry the moment
// its match ends, so the result of the last decision is only readable on the
// answer to the decision that ended it.
func playMatch(t *testing.T, on board, dependencies room.Deps, configuration room.Config, host, guest placement.Squad) outcome {
	t.Helper()
	clients := newTable(t, dependencies.Books, configuration.TurnCap)

	answered, err := on.join(hello(t, host, "Host"))
	if err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	if !answered.Known {
		t.Fatal("the host joined a room the registry does not know")
	}
	if answered.Seat != wire.SeatHost {
		t.Fatalf("the first peer took the %q seat, want the host's", answered.Seat)
	}
	clients.deliver(t, answered.Out)

	answered, err = on.join(hello(t, guest, "Guest"))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	if answered.Seat != wire.SeatGuest {
		t.Fatalf("the second peer took the %q seat, want the guest's", answered.Seat)
	}
	clients.deliver(t, answered.Out)

	reading, steps := answered.Reading, 0
	for !reading.Finished {
		if !reading.Waiting {
			t.Fatalf("after %d decisions the room waits on nobody and the match is not over", steps)
		}
		client := clients.at(reading.Awaiting)
		if client == nil {
			t.Fatalf("the room waits on %q, which is not a seat", reading.Awaiting)
		}
		answered, err = on.deliver(reading.Awaiting, client.answer())
		if err != nil {
			t.Fatalf("decision %d from %s: %v", steps, reading.Awaiting, err)
		}
		if !answered.Known {
			t.Fatalf("the room went away after %d decisions with the match unfinished", steps)
		}
		clients.deliver(t, answered.Out)
		reading = answered.Reading
		steps++
		// A backstop rather than an expectation: without it a room that stopped
		// making progress would hang the suite instead of failing it.
		if steps > configuration.Battles*configuration.TurnCap {
			t.Fatalf("the match took more than %d decisions, so something is not progressing", steps)
		}
	}
	return outcome{reading: reading, steps: steps, compared: clients.host.compared}
}

// TestManyRoomsPlayWholeMatchesAtOnce is the headline of the registry and the
// test the race detector exists for: several rooms in one process, each driven to
// the end of its match by two fake clients answering through battle.Suggest, all
// at the same time.
//
// The rooms are driven by **parallel subtests** rather than by goroutines of this
// test's own, and that is not a style choice: a t.Fatalf outside the goroutine
// running the test does not fail the suite properly, so every mirror's assertion
// — and there is one on every turn of every battle — would turn a failure into a
// hang. A parallel subtest has its own *testing.T on its own goroutine, so the
// fixtures assert exactly as they do single-threaded.
//
// `make check` runs this package with -race for it. Measured at ~3s against a
// gate of about a minute, which is what makes it affordable — and a race test
// nobody runs is not a net.
func TestManyRoomsPlayWholeMatchesAtOnce(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	const rooms = 4

	codes := make([]wire.RoomCode, rooms)
	for index := range rooms {
		configuration := config(uint64(11+index), 3)
		code, err := registry.Open(theOneListener, configuration, dependencies)
		if err != nil {
			t.Fatalf("open room %d: %v", index, err)
		}
		codes[index] = code
	}
	// The thing the seventh byte exists for, asserted before a turn is fought:
	// four rooms behind **one** address, four distinct codes, each naming that
	// address. Before the widening this loop could not have been written — every
	// room behind one listener encoded the same string.
	seen := make(map[wire.RoomCode]int, rooms)
	for index, code := range codes {
		at, err := code.AddrPort()
		if err != nil {
			t.Fatalf("room %d's code %q does not decode: %v", index, code, err)
		}
		if at != theOneListener {
			t.Errorf("room %d's code names %s, want %s", index, at, theOneListener)
		}
		if first, taken := seen[code]; taken {
			t.Fatalf("rooms %d and %d were both handed the code %q", first, index, code)
		}
		seen[code] = index
	}
	if got := registry.Count(); got != rooms {
		t.Fatalf("the registry holds %d rooms, want %d", got, rooms)
	}
	if got := registry.Running(); got != rooms {
		t.Fatalf("%d room goroutines are owed an end, want %d", got, rooms)
	}
	if got := registry.Codes(); len(got) != rooms {
		t.Fatalf("the registry names %d codes, want %d", len(got), rooms)
	}

	// A reading of a room nobody has joined, which is also the one place Read is
	// exercised on a live room: the transport asks this to start an allowance on
	// a seat that was handed no answer of its own.
	waiting, known := registry.Read(codes[0])
	if !known {
		t.Fatal("the registry does not know a room it just opened")
	}
	if waiting.Waiting || waiting.Finished {
		t.Errorf("a room nobody has joined is waiting on %q / finished %v", waiting.Awaiting, waiting.Finished)
	}
	if waiting.Config != config(11, 3) {
		t.Error("the room holds a configuration that is not the one it was opened with")
	}

	results := make([]outcome, rooms)
	t.Run("at once", func(t *testing.T) {
		for index := range rooms {
			t.Run(fmt.Sprintf("room %d", index), func(t *testing.T) {
				t.Parallel()
				host, guest := squadPair(t, dependencies, index)
				// Distinct elements of one slice, written from distinct
				// goroutines: race-free, and the race detector is what says so.
				results[index] = playMatch(t,
					throughTheRegistry{registry: registry, code: codes[index]},
					dependencies, config(uint64(11+index), 3), host, guest)
			})
		}
	})

	for index, result := range results {
		if !result.reading.Finished {
			t.Errorf("room %d did not finish its match", index)
			continue
		}
		if !result.reading.Result.Verdict.Over() {
			t.Errorf("room %d finished with the verdict %q", index, result.reading.Result.Verdict)
		}
		if result.reading.Result.Departed.Valid() {
			t.Errorf("room %d records %q as having gone away on a match played out",
				index, result.reading.Result.Departed)
		}
		played := len(result.reading.Played)
		if played == 0 || played > 3 {
			t.Errorf("room %d played %d battles of a bo3", index, played)
		}
		// The vacuity guard. A 3v3 of the shipped cast takes 34 to 55 decisions a
		// battle, so a run that compared a handful of digests did not fight a
		// match and every assertion inside the mirror measured nothing.
		if result.compared != result.steps {
			t.Errorf("room %d's host checked %d digests over %d decisions; every turn goes to both clients",
				index, result.compared, result.steps)
		}
		if result.compared < 30 {
			t.Errorf("room %d's host checked only %d digests, which is too few for a real match", index, result.compared)
		}
	}

	// Every match is over, so every goroutine has ended by itself. Wait closes
	// nothing — → TestAFinishedRoomLeavesNoGoroutineBehind, which is where that
	// is the claim rather than the tidying up.
	registry.Wait()
	t.Logf("%d rooms behind %s played %d, %d, %d and %d decisions at once, under the codes %v",
		rooms, theOneListener, results[0].steps, results[1].steps, results[2].steps, results[3].steps, codes)
}

// TestTheRegistryDoesNotCrossWires is the claim a single-room test cannot make:
// with N rooms running at once, each room's result is the one **its own** inputs
// produced.
//
// The reference is the same pairing played **alone** — same squads, same seed,
// through a bare room with no registry and no goroutine — which is available as a
// reference precisely because a battle is a pure function of its seed and the
// decisions taken. So a registry that handed room 2's answer to room 1 would fail
// this while passing every test in which one room runs at a time, and a registry
// that quietly shared one room between two codes would fail it too.
//
// ⚠️ The four rooms are behind **one address** and differ only in their room
// byte, which is what makes the claim worth more than it used to be: base32
// spends five bits a character, so the first six bytes fill nine characters and
// the four codes share **their first nine characters** exactly, differing only
// in the last three. A lookup that compared a prefix, or a byte read at the
// wrong offset, would hand one room's answer to another.
//
// It is bo1 rather than bo3 to keep the eight matches cheap; what is being
// measured is which room an answer came from, and that does not need a series.
func TestTheRegistryDoesNotCrossWires(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	const rooms = 4

	codes := make([]wire.RoomCode, rooms)
	for index := range rooms {
		code, err := registry.Open(theOneListener, config(uint64(101+index), 1), dependencies)
		if err != nil {
			t.Fatalf("open room %d: %v", index, err)
		}
		codes[index] = code
	}

	together := make([]outcome, rooms)
	t.Run("at once", func(t *testing.T) {
		for index := range rooms {
			t.Run(fmt.Sprintf("room %d", index), func(t *testing.T) {
				t.Parallel()
				host, guest := squadPair(t, dependencies, index)
				together[index] = playMatch(t,
					throughTheRegistry{registry: registry, code: codes[index]},
					dependencies, config(uint64(101+index), 1), host, guest)
			})
		}
	})
	registry.Wait()

	for index := range rooms {
		configuration := config(uint64(101+index), 1)
		host, guest := squadPair(t, dependencies, index)
		alone := playMatch(t, aloneOn{playing: newRoom(t, configuration)}, dependencies, configuration, host, guest)

		if together[index].steps != alone.steps {
			t.Errorf("room %d took %d decisions through the registry and %d alone",
				index, together[index].steps, alone.steps)
		}
		if together[index].reading.Result != alone.reading.Result {
			t.Errorf("room %d came to %+v through the registry and %+v alone",
				index, together[index].reading.Result, alone.reading.Result)
		}
		if !reflect.DeepEqual(together[index].reading.Played, alone.reading.Played) {
			t.Errorf("room %d recorded %+v through the registry and %+v alone",
				index, together[index].reading.Played, alone.reading.Played)
		}
		if together[index].reading.Skipped != alone.reading.Skipped {
			t.Errorf("room %d walked past %d prompts through the registry and %d alone",
				index, together[index].reading.Skipped, alone.reading.Skipped)
		}
	}

	// And the four results are not four copies of one match, or the comparison
	// above would be satisfied by a registry that ran the same room four times.
	for index := 1; index < rooms; index++ {
		if reflect.DeepEqual(together[index].reading.Played, together[0].reading.Played) {
			t.Errorf("rooms 0 and %d recorded the same battles, so this test cannot tell them apart", index)
		}
	}
	t.Logf("%d rooms each matched the same pairing played alone, over %d, %d, %d and %d decisions",
		rooms, together[0].steps, together[1].steps, together[2].steps, together[3].steps)
}

// TestAFinishedRoomLeavesNoGoroutineBehind measures the two halves of a room
// ending rather than asserting the code looks right: the goroutine ends, and the
// entry goes.
//
// ⚠️ **Wait closes nothing**, which is what makes it the measurement. A room left
// behind hangs this test — the suite's own timeout is the failure — where a
// shutdown call would have tidied the leak up and reported success. Count and
// Running are then read *after* Wait rather than after the last decision, because
// a room retires in its own teardown: reading them a moment earlier would be
// racing the thing being measured rather than measuring it.
func TestAFinishedRoomLeavesNoGoroutineBehind(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	const rooms = 3

	codes := make([]wire.RoomCode, rooms)
	for index := range rooms {
		code, err := registry.Open(theOneListener, config(uint64(201+index), 1), dependencies)
		if err != nil {
			t.Fatalf("open room %d: %v", index, err)
		}
		codes[index] = code
	}
	// The before reading, without which "nought afterwards" would pass on a
	// registry that never started a goroutine at all.
	if got := registry.Running(); got != rooms {
		t.Fatalf("%d room goroutines before the matches, want %d", got, rooms)
	}
	if got := registry.Count(); got != rooms {
		t.Fatalf("%d rooms on the map before the matches, want %d", got, rooms)
	}

	for index := range rooms {
		configuration := config(uint64(201+index), 1)
		host, guest := squadPair(t, dependencies, index)
		played := playMatch(t, throughTheRegistry{registry: registry, code: codes[index]},
			dependencies, configuration, host, guest)
		if !played.reading.Finished {
			t.Fatalf("room %d did not finish", index)
		}
	}

	registry.Wait()
	if got := registry.Running(); got != 0 {
		t.Errorf("%d room goroutines outlived their matches", got)
	}
	if got := registry.Count(); got != 0 {
		t.Errorf("%d finished rooms are still reachable by code", got)
	}
	if got := registry.Codes(); len(got) != 0 {
		t.Errorf("the registry still names the codes %v", got)
	}
	// And nothing was closed to make that true: with every room already gone,
	// there is nothing left for a shutdown to close.
	if closed := registry.CloseAll(); closed != 0 {
		t.Errorf("CloseAll closed %d rooms after every match had ended, so the rooms did not retire themselves", closed)
	}
	t.Logf("%d rooms finished, retired their entries and ended their goroutines with nothing closing them", rooms)
}

// TestAMessageInFlightToAFinishedRoomIsRefused is the panic that is reachable
// here and is not taken: a send on a closed channel panics and a second close
// panics, and a room's own retirement races every peer still talking to it.
//
// Three shapes of it, because they are three different windows:
//
//  1. **After the fact.** Every input on a code whose match has ended answers
//     wire.CodeRoomUnknown, including Left — which is the ordinary case rather
//     than a fault, since a transport notices a socket closing after the room
//     that closed it has gone.
//  2. **A room closed under a live peer.** Close removes the entry and stops the
//     goroutine mid-match; the peer's next message is refused.
//  3. **Genuinely in flight.** Eight goroutines hammer a room with messages it
//     always refuses while the match is played out under them, so sends are
//     landing across the retirement rather than after it. They must all be
//     answered or refused, and none may panic.
//
// The third is what the race detector is pointed at, and the messages are chosen
// to change nothing: a body no seated peer sends is answered
// wire.CodeUnknownMessage, and a seat nobody holds is answered
// wire.CodeNotYourTurn — neither takes a turn, so the driver's own match is
// unaffected by however many arrive. (It used to read "neither resets a miss
// count nor takes a turn"; there is no miss count any more, and taking a turn is
// the whole of what a refusal now protects.)
func TestAMessageInFlightToAFinishedRoomIsRefused(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	code, err := registry.Open(theOneListener, config(303, 1), dependencies)
	if err != nil {
		t.Fatalf("open the room: %v", err)
	}
	// (2)'s room is opened **here, beside the first**, and that is a consequence
	// of the registry allocating the byte. It used to be opened further down, and
	// with a caller-supplied code that was free; now the byte is the lowest one
	// going, so a room opened after the first has retired would be handed **the
	// first room's code back** — and whether it was would depend on how far that
	// retirement had got, so the two codes would be equal on some runs and not
	// others. A test whose two subjects are sometimes one subject measures
	// something different every time it is run.
	//
	// Wait below is unaffected, which is the thing to check before moving it: it
	// waits for every goroutine in the process, and this room is closed before
	// the call.
	closedCode, err := registry.Open(theOneListener, config(304, 1), dependencies)
	if err != nil {
		t.Fatalf("open the room that gets closed: %v", err)
	}
	if closedCode == code {
		t.Fatalf("both rooms were opened under %q, so this test has one subject and not two", code)
	}

	// (3) the noise, running across the whole match and its ending.
	stop := make(chan struct{})
	noise := sync.WaitGroup{}
	refusals := make([]int, 8)
	for at := range refusals {
		noise.Add(1)
		go func() {
			defer noise.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				// A body a seated peer never sends, from a seat nobody holds:
				// refused either way, and state-free whichever refusal it gets.
				answered, err := registry.Deliver(code, wire.Seat("spectator"), wire.Welcome{})
				if err != nil {
					t.Errorf("a message in flight came back with an error: %v", err)
					return
				}
				if len(answered.Out) != 1 {
					t.Errorf("a refusal came back as %d messages, want one", len(answered.Out))
					return
				}
				if _, refused := answered.Out[0].Body.(wire.Refused); !refused {
					t.Errorf("a refusal came back as a %T", answered.Out[0].Body)
					return
				}
				refusals[at]++
			}
		}()
	}

	host, guest := squadPair(t, dependencies, 0)
	played := playMatch(t, throughTheRegistry{registry: registry, code: code}, dependencies, config(303, 1), host, guest)
	if !played.reading.Finished {
		t.Fatal("the match did not finish")
	}
	close(stop)
	noise.Wait()
	landed := 0
	for _, count := range refusals {
		landed += count
	}
	if landed == 0 {
		t.Error("no message landed while the match was played, so nothing was ever in flight")
	}

	// (2) a room closed under a live peer, mid-match. → the note beside its Open
	// for why it is opened at the top now.
	answered, err := registry.Join(closedCode, hello(t, host, "Host"))
	if err != nil || !answered.Known {
		t.Fatalf("the host joins the room that gets closed: %v / known %v", err, answered.Known)
	}
	if !registry.Close(closedCode) {
		t.Fatal("closing a running room reported there was none")
	}
	// A second close is the double-close panic, and it must simply report that
	// there was nothing left to close.
	if registry.Close(closedCode) {
		t.Error("closing the same room twice reported there was one to close both times")
	}
	answered, err = registry.Deliver(closedCode, wire.SeatHost, wire.Pass{})
	if err != nil {
		t.Errorf("a message to a closed room errored: %v", err)
	}
	if answered.Known {
		t.Error("a message to a closed room was answered by a room the registry claims to know")
	}
	assertRoomUnknown(t, "a message to a closed room", answered.Out)

	// (1) after the fact. The wait is what makes this a claim about a room that
	// has **retired** rather than one that is about to: a room replies to the
	// decision that ends its match and only then leaves, so an input sent in
	// that window is legitimately answered by the room, and asserting a refusal
	// without waiting would be racing the thing being measured.
	registry.Wait()
	for _, refused := range []struct {
		what   string
		answer func() (room.Answer, error)
	}{
		{"a join", func() (room.Answer, error) { return registry.Join(code, hello(t, host, "Late")) }},
		{"an act", func() (room.Answer, error) { return registry.Deliver(code, wire.SeatHost, wire.Pass{}) }},
		{"a timeout", func() (room.Answer, error) { return registry.TimedOut(code, wire.SeatHost) }},
		{"a departure", func() (room.Answer, error) { return registry.Left(code, wire.SeatHost) }},
	} {
		answered, err := refused.answer()
		if err != nil {
			t.Errorf("%s to a finished room errored: %v", refused.what, err)
			continue
		}
		if answered.Known {
			t.Errorf("%s to a finished room was answered by a room the registry claims to know", refused.what)
		}
		assertRoomUnknown(t, refused.what+" to a finished room", answered.Out)
	}

	if got := registry.Count(); got != 0 {
		t.Errorf("%d rooms are still reachable by code after one finished and one was closed", got)
	}
	if got := registry.Running(); got != 0 {
		t.Errorf("%d room goroutines are still running", got)
	}
	t.Logf("%d messages landed in flight and were refused; a finished room, a closed room and a second close all answered rather than panicking", landed)
}

// TestAnUnknownCodeAnswersRoomUnknown is the first thing in the repository that
// sends wire.CodeRoomUnknown.
//
// ⚠️ The code has shipped since internal/wire landed and **nothing sent it**:
// internal/room's gate documents it at gate.go as *the registry's* refusal and
// says no room ever sends one, so until the registry existed it was a code no
// peer could ever be shown. Which makes this test the one that closes it, and the
// reason the commit says so.
//
// It names **no seat**, for the reason a refusal at the gate names none: a seat
// is a place in a room, so a code naming no room cannot name a seat in one, and
// the transport answers the connection it read the message from.
func TestAnUnknownCodeAnswersRoomUnknown(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	// A code that decodes perfectly and names a room this process is not running,
	// which is what a restarted host looks like to a client with an old code.
	stale := codeFor(t, 99)
	host, _ := squadPair(t, dependencies, 0)

	for _, unknown := range []struct {
		what   string
		answer func() (room.Answer, error)
	}{
		{"a join", func() (room.Answer, error) { return registry.Join(stale, hello(t, host, "Host")) }},
		{"an act", func() (room.Answer, error) {
			return registry.Deliver(stale, wire.SeatHost, wire.Act{Skill: "razor_leaf"})
		}},
		{"a pass", func() (room.Answer, error) { return registry.Deliver(stale, wire.SeatGuest, wire.Pass{}) }},
		{"a timeout", func() (room.Answer, error) { return registry.TimedOut(stale, wire.SeatHost) }},
		{"a departure", func() (room.Answer, error) { return registry.Left(stale, wire.SeatGuest) }},
	} {
		answered, err := unknown.answer()
		if err != nil {
			t.Errorf("%s to an unknown code errored: %v", unknown.what, err)
			continue
		}
		if answered.Known {
			t.Errorf("%s to an unknown code reports a room the registry knows", unknown.what)
		}
		if answered.Seat.Valid() {
			t.Errorf("%s to an unknown code handed out the %q seat", unknown.what, answered.Seat)
		}
		if !reflect.DeepEqual(answered.Reading, room.Reading{}) {
			t.Errorf("%s to an unknown code came back with a reading of a room that does not exist", unknown.what)
		}
		assertRoomUnknown(t, unknown.what+" to an unknown code", answered.Out)
	}

	// A reading is not a message, so it reports that there is nothing to read
	// rather than answering a code to nobody.
	if _, known := registry.Read(stale); known {
		t.Error("the registry read a room it is not running")
	}
	if got := registry.Count(); got != 0 {
		t.Errorf("refusing five messages left %d rooms on the map", got)
	}
	t.Log("every input on an unknown code answers wire.CodeRoomUnknown, naming no seat — the first thing in the repository that sends it")
}

// TestTheRegistryAllocatesTheLowestFreeRoomByte is what replaced
// TestADuplicateCodeIsRefused, and the replacement is the point rather than a
// tidy-up: with the registry picking the byte, **a duplicate code cannot be
// asked for**, so the refusal that test held is unreachable and a test asserting
// it would measure nothing. Impossible by construction beats refused.
//
// What that test *also* held survives here, because it was the half that
// mattered: a second room opened behind the same address leaves the first
// **untouched and still playable**. It used to be untouched because the second
// open was refused; it is untouched now because the second open gets a code of
// its own.
//
// The lowest free byte rather than a counter is what makes the code a test can
// name: a closed room gives its byte back, so the next open takes the gap.
func TestTheRegistryAllocatesTheLowestFreeRoomByte(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	code, err := registry.Open(theOneListener, config(401, 1), dependencies)
	if err != nil {
		t.Fatalf("open the room: %v", err)
	}
	if want := codeFor(t, 0); code != want {
		t.Fatalf("the first room opened under %q, want %q — room nought", code, want)
	}
	host, guest := squadPair(t, dependencies, 0)

	// Somebody is already at the board.
	answered, err := registry.Join(code, hello(t, host, "Host"))
	if err != nil || answered.Seat != wire.SeatHost {
		t.Fatalf("the host joins: %v / seat %q", err, answered.Seat)
	}

	// A second room behind the same address, with a different configuration so
	// that a registry which had somehow replaced the first would be visible in
	// what that room says about itself.
	second, err := registry.Open(theOneListener, config(999, 3), dependencies)
	if err != nil {
		t.Fatalf("open a second room behind %s: %v", theOneListener, err)
	}
	if want := codeFor(t, 1); second != want {
		t.Errorf("the second room opened under %q, want %q — the lowest free byte", second, want)
	}
	if got := registry.Count(); got != 2 {
		t.Errorf("the registry holds %d rooms after two opens, want 2", got)
	}
	if got := registry.Running(); got != 2 {
		t.Errorf("%d room goroutines after two opens, want 2", got)
	}

	// The room that was already there is still the one that was already there,
	// and the match it was in the middle of runs to its end.
	reading, known := registry.Read(code)
	if !known {
		t.Fatal("the first room is gone after a second was opened behind the same address")
	}
	if reading.Config != config(401, 1) {
		t.Error("the first room's configuration is not the one it was opened with")
	}
	answered, err = registry.Join(code, hello(t, guest, "Guest"))
	if err != nil || answered.Seat != wire.SeatGuest {
		t.Fatalf("the guest joins the room that was already running: %v / seat %q", err, answered.Seat)
	}
	if !answered.Reading.Waiting {
		t.Error("the match did not start when the second peer sat down")
	}

	// Two more, so there is a byte in the middle to give back.
	third, err := registry.Open(theOneListener, config(402, 1), dependencies)
	if err != nil {
		t.Fatalf("open a third room: %v", err)
	}
	if want := codeFor(t, 2); third != want {
		t.Errorf("the third room opened under %q, want %q", third, want)
	}

	// A closed room gives its byte back, and the next open takes **that** gap
	// rather than the next number up — which is the whole of what "lowest free"
	// buys, and the one claim a counter would fail.
	if !registry.Close(second) {
		t.Fatal("closing the second room reported there was none")
	}
	reopened, err := registry.Open(theOneListener, config(403, 1), dependencies)
	if err != nil {
		t.Fatalf("reopen into the gap: %v", err)
	}
	if reopened != second {
		t.Errorf("the room after the gap opened under %q, want %q — the byte the closed room gave back", reopened, second)
	}

	if closed := registry.CloseAll(); closed != 3 {
		t.Errorf("CloseAll closed %d rooms, want 3", closed)
	}
	registry.Wait()
	if got := registry.Running(); got != 0 {
		t.Errorf("%d room goroutines survived the shutdown", got)
	}
	t.Logf("rooms behind %s took the bytes 0, 1, 2 in order and reused 1 after it was closed (%q)", theOneListener, reopened)
}

// TestA257thRoomBehindOneAddressIsRefused is the bound the room byte is: eight
// bits, so 256 rooms behind one address and no more.
//
// ⚠️ It is an **error rather than a wire.Code**, and that division is the point.
// A joiner is told a room is unknown, refused at the gate or sent away by a
// password; a host that has 256 rooms running and asks for another is not a
// joiner and there is nothing to tell a peer — the caller of Open is the host's
// own process, so it takes the answer as a Go error.
//
// 256 rooms is far past what a LAN wants, which is why this is a shape check
// rather than a limit anybody will meet. Each room is a struct and an idle
// goroutine until somebody joins, so the sweep is cheap.
func TestA257thRoomBehindOneAddressIsRefused(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	for index := range wire.RoomsPerProcess {
		// #nosec G115 -- index is a loop over 0..255.
		want := codeFor(t, uint8(index))
		code, err := registry.Open(theOneListener, config(uint64(500+index), 1), dependencies)
		if err != nil {
			t.Fatalf("open room %d of %d: %v", index, wire.RoomsPerProcess, err)
		}
		if code != want {
			t.Fatalf("room %d opened under %q, want %q", index, code, want)
		}
	}
	if got := registry.Count(); got != wire.RoomsPerProcess {
		t.Fatalf("the registry holds %d rooms, want %d", got, wire.RoomsPerProcess)
	}

	overflowing, refusal := registry.Open(theOneListener, config(999, 1), dependencies)
	if refusal == nil {
		t.Fatalf("a %dth room opened behind %s, under %q", wire.RoomsPerProcess+1, theOneListener, overflowing)
	}
	if overflowing != "" {
		t.Errorf("a refused open handed back the code %q as well as its error", overflowing)
	}
	// The refusal costs the process nothing: no entry, and no goroutine owed an
	// end. A registry that enrolled and then refused would leak one of each.
	if got := registry.Count(); got != wire.RoomsPerProcess {
		t.Errorf("the registry holds %d rooms after a refused open, want %d", got, wire.RoomsPerProcess)
	}
	if got := registry.Running(); got != wire.RoomsPerProcess {
		t.Errorf("%d room goroutines after a refused open, want %d", got, wire.RoomsPerProcess)
	}

	// And a room closing makes room for one, which says the refusal is about the
	// bytes in use rather than a count that only ever goes up.
	if !registry.Close(codeFor(t, 7)) {
		t.Fatal("closing room 7 reported there was none")
	}
	reopened, reopenErr := registry.Open(theOneListener, config(998, 1), dependencies)
	if reopenErr != nil {
		t.Fatalf("open a room into the freed byte: %v", reopenErr)
	}
	if want := codeFor(t, 7); reopened != want {
		t.Errorf("the room after the gap opened under %q, want %q", reopened, want)
	}

	if closed := registry.CloseAll(); closed != wire.RoomsPerProcess {
		t.Errorf("CloseAll closed %d rooms, want %d", closed, wire.RoomsPerProcess)
	}
	registry.Wait()
	t.Logf("%d rooms fit behind %s and the next one is refused as an error: %v",
		wire.RoomsPerProcess, theOneListener, refusal)
}

// TestARoomOnlyOpensAtAnAddressACodeCanCarry is what is left of the malformed
// half of the deleted duplicate test. A caller cannot hand in a malformed code
// any more — there is no code parameter — so the refusal moved to the **address**,
// which is the one thing about a code the caller still chooses.
//
// It is wire.EncodeRoom's refusal rather than a second copy of it: sixteen bytes
// of IPv6 plus a port and a room is thirty-one base32 characters and not a code
// anybody retypes. → the note on EncodeRoom, and the v4-mapped case, which must
// **not** be refused because it is what a listener reports for an ordinary v4
// socket.
func TestARoomOnlyOpensAtAnAddressACodeCanCarry(t *testing.T) {
	dependencies := deps(t)
	registry := room.NewRegistry()
	for _, refused := range []netip.AddrPort{
		netip.MustParseAddrPort("[::1]:7777"),
		netip.MustParseAddrPort("[fe80::1]:7777"),
		{},
	} {
		code, err := registry.Open(refused, config(601, 1), dependencies)
		if err == nil {
			t.Errorf("a room opened at %s, under %q", refused, code)
		}
		if code != "" {
			t.Errorf("opening at %s handed back the code %q alongside its error", refused, code)
		}
	}
	// Refused before anything was built, so no entry and no goroutine.
	if got := registry.Count(); got != 0 {
		t.Errorf("%d rooms on the map after three refused addresses", got)
	}
	if got := registry.Running(); got != 0 {
		t.Errorf("%d room goroutines after three refused addresses", got)
	}

	mapped := netip.AddrPortFrom(netip.MustParseAddr("::ffff:127.0.0.1"), 7100)
	code, err := registry.Open(mapped, config(602, 1), dependencies)
	if err != nil {
		t.Fatalf("a v4-mapped address is an ordinary v4 socket and was refused: %v", err)
	}
	// Unwrapped, so it is the same code the plain v4 address would have given —
	// a room a host reached one way and a joiner the other has to be one room.
	if want := codeFor(t, 0); code != want {
		t.Errorf("a room opened at %s took the code %q, want %q", mapped, code, want)
	}
	if closed := registry.CloseAll(); closed != 1 {
		t.Errorf("CloseAll closed %d rooms, want 1", closed)
	}
	registry.Wait()
	t.Log("an address no code can carry is refused before a room is built; a v4-mapped one is unwrapped and takes the plain address's code")
}

// assertRoomUnknown is the one refusal the registry sends, checked the same way
// everywhere: one message, no seat, wire.CodeRoomUnknown.
func assertRoomUnknown(t *testing.T, what string, out []room.Outbound) {
	t.Helper()
	if len(out) != 1 {
		t.Errorf("%s came back as %d messages, want one refusal", what, len(out))
		return
	}
	if out[0].To.Valid() {
		t.Errorf("%s was addressed to the %q seat; a code naming no room names no seat either", what, out[0].To)
	}
	refused, ok := out[0].Body.(wire.Refused)
	if !ok {
		t.Errorf("%s came back as a %T, want a wire.Refused", what, out[0].Body)
		return
	}
	if refused.Code != wire.CodeRoomUnknown {
		t.Errorf("%s was refused with %q, want %q", what, refused.Code, wire.CodeRoomUnknown)
	}
}

// TestConcurrentOpensAtOneAddressNeverShareAByte is the net for the one thing
// enrol does that reading it cannot prove: it finds a free room byte and takes
// it under **one** hold of the mutex.
//
// ⚠️ This test exists because splitting that into two holds — find under the
// first, occupy under the second — **survived the entire suite, including
// `-race` three times over.** It is a *logic* race and not a data race: both
// holds are properly locked, so the detector has nothing to report, and every
// other Open in this file is sequential, so nothing ever drove two of them into
// the window. Reading enrol tells you the hold has to be single; only this says
// so.
//
// What the two-hold version does when the window is hit is exact rather than
// vague, which is what makes the assertions sharp: two callers find the same
// free byte, and the second `g.rooms[code] = entry` **overwrites** the first —
// so the map holds one entry where live counted two. Count and Running
// therefore disagree, one room's goroutine is orphaned, and two callers hold the
// same code. Each of those is asserted.
//
// ⚠️ Honest about what it is: the *window* is probabilistic and the *assertion*
// is not. With one hold this passes every time by construction, so it never
// cries wolf; with two it fails often rather than always, which is why it opens
// several rounds rather than one. A regression here is caught quickly, not
// necessarily on the first run.
func TestConcurrentOpensAtOneAddressNeverShareAByte(t *testing.T) {
	const rounds, atOnce = 20, 8
	dependencies := deps(t)
	for round := range rounds {
		registry := room.NewRegistry()
		start := make(chan struct{})
		codes := make([]wire.RoomCode, atOnce)
		failures := make([]error, atOnce)
		var racing sync.WaitGroup
		for index := range atOnce {
			racing.Add(1)
			go func() {
				defer racing.Done()
				<-start
				codes[index], failures[index] = registry.Open(
					theOneListener, config(uint64(600+index), 1), dependencies)
			}()
		}
		close(start)
		racing.Wait()

		seen := make(map[wire.RoomCode]int, atOnce)
		for index, code := range codes {
			if failures[index] != nil {
				t.Fatalf("round %d: opening room %d at one address: %v", round, index, failures[index])
			}
			seen[code]++
		}
		if len(seen) != atOnce {
			t.Fatalf("round %d: %d callers opened a room at %s and hold %d distinct codes, want %d — two of them were handed the same room byte",
				round, atOnce, theOneListener, len(seen), atOnce)
		}
		if got := registry.Count(); got != atOnce {
			t.Fatalf("round %d: %d rooms opened and the registry holds %d, want %d — an entry was overwritten",
				round, atOnce, got, atOnce)
		}
		if got := registry.Running(); got != atOnce {
			t.Fatalf("round %d: %d rooms opened and %d goroutines are owed an end, want %d",
				round, atOnce, got, atOnce)
		}
		registry.CloseAll()
		registry.Wait()
	}
}
