package socket

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The transport's half of a spectator
//
// A watcher is a connection the room welcomed with **no seat** (→
// room.Admission.Watching), which the transport keeps and writes the match to
// off the room's own append-only record. Everything below is over a real
// loopback listener with real clients, because every claim here is about the
// wire: what a connection is handed, in what order, and what happens to it when
// its room goes away.
//
// ⚠️ **A watcher is not a seat and none of these tests may make it look like
// one.** The order the two seats are visited in reaches the roster and the
// roster's order decides who wins a speed tie, so the claim
// TestAMatchWithWatchersAttachedIsTheSameMatch holds is the load-bearing one:
// everything else here could pass on a transport that had quietly changed the
// match it was showing.

// watchingHello is a peer asking to watch rather than to play. It brings no
// squad — a watcher's squad is ignored rather than refused, which is
// internal/room's own test — and it announces the version this binary would.
func watchingHello(t *testing.T, name string) wire.Hello {
	t.Helper()
	joining := hello(t, placement.Squad{}, name, "")
	joining.Watch = true
	return joining
}

// spectator is a watching connection and everything it has been handed, read on
// its own goroutine the way a real client's Play loop would read it.
//
// ⚠️ It reads the **bodies** rather than driving a Mirror, deliberately: the
// claims here are about what reaches the socket and in what order, and a mirror
// would fold a run of turns into a battle that no longer says which of them
// arrived, or whether one arrived twice. A watching client is step 5.
type spectator struct {
	*Client
	name string

	mu   sync.Mutex
	got  []wire.Body
	err  error
	done chan struct{}
}

// watch dials a watcher and starts reading. It fails the test on a refusal — the
// test that is *about* a refusal calls Dial itself.
func (l *listener) watch(t *testing.T, code wire.RoomCode, name string, dependencies room.Deps) *spectator {
	t.Helper()
	client, err := Dial(context.Background(), code, watchingHello(t, name), dependencies.Books,
		ClientOptions{Timings: l.timings})
	if err != nil {
		t.Fatalf("%s dials room %s to watch: %v", name, code, err)
	}
	if seat := client.Seat(); seat.Valid() {
		t.Fatalf("%s was welcomed into the %q seat, and a watcher takes none", name, seat)
	}
	watching := &spectator{Client: client, name: name, done: make(chan struct{})}
	go watching.read()
	t.Cleanup(func() { client.Close(); <-watching.done })
	return watching
}

// read is the loop, and it ends when the connection does — which is what the
// transport letting go of a watcher looks like from this end.
func (w *spectator) read() {
	defer close(w.done)
	for {
		body, err := w.conn.read(context.Background())
		if err != nil {
			w.mu.Lock()
			w.err = err
			w.mu.Unlock()
			return
		}
		w.mu.Lock()
		w.got = append(w.got, body)
		w.mu.Unlock()
	}
}

// bodies is everything handed to this watcher so far, copied.
func (w *spectator) bodies() []wire.Body {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append(make([]wire.Body, 0, len(w.got)), w.got...)
}

// ended waits for the transport to let this watcher go, which is what the end of
// a match does to every one of them.
func (w *spectator) ended(t *testing.T) {
	t.Helper()
	select {
	case <-w.done:
	case <-time.After(theWholeMatch):
		t.Fatalf("%s was still being written to %s after the match ended", w.name, theWholeMatch)
	}
}

// awaits waits until this watcher has been handed at least n bodies.
//
// ⚠️ It polls, for the reason listener.emptied does: what it is waiting for is
// another goroutine's write to a socket, and nothing signals that. The bound is
// what turns a transport that stopped writing into a failure rather than a hang.
func (w *spectator) awaits(t *testing.T, n int, what string) []wire.Body {
	t.Helper()
	for waited := 0; waited < 600; waited++ {
		if held := w.bodies(); len(held) >= n {
			return held
		}
		time.Sleep(10 * time.Millisecond)
	}
	held := w.bodies()
	t.Fatalf("%s holds %d bodies after six seconds and %s needs %d", w.name, len(held), what, n)
	return held
}

// counted is the shape of one watcher's stream: how many battles opened and how
// many turns were played, which are the two independent numbers a match can be
// checked against.
type counted struct {
	starts, turns, other int
}

func shapeOf(t *testing.T, who string, bodies []wire.Body) counted {
	t.Helper()
	var out counted
	for at, body := range bodies {
		switch body.(type) {
		case wire.Start, *wire.Start:
			out.starts++
		case wire.Turn, *wire.Turn:
			out.turns++
		default:
			out.other++
			t.Errorf("%s was handed a %s at position %d, and a watcher's record is starts, "+
				"turns and a closure", who, body.Kind(), at)
		}
	}
	return out
}

// decisionsIn is the decisions a stream carries, in order, which is what two
// watchers of one match have to agree on down to the last one.
func decisionsIn(bodies []wire.Body) []string {
	out := make([]string, 0, len(bodies))
	for _, body := range bodies {
		switch turn := body.(type) {
		case wire.Turn:
			out = append(out, describeDecision(turn.Decision))
		case *wire.Turn:
			out = append(out, describeDecision(turn.Decision))
		case wire.Start:
			out = append(out, fmt.Sprintf("start battle %d seed %d", turn.Battle, turn.Seed))
		case *wire.Start:
			out = append(out, fmt.Sprintf("start battle %d seed %d", turn.Battle, turn.Seed))
		default:
			out = append(out, body.Kind().String())
		}
	}
	return out
}

// startAt is the wire.Start a stream holds at one position, in either of the two
// forms a decoded body arrives in: wire.Decode hands back a pointer and the room
// records a value, and a watcher reads the decoded one.
func startAt(bodies []wire.Body, at int) (wire.Start, bool) {
	if at >= len(bodies) {
		return wire.Start{}, false
	}
	switch start := bodies[at].(type) {
	case wire.Start:
		return start, true
	case *wire.Start:
		return *start, true
	}
	return wire.Start{}, false
}

func describeDecision(decision battle.Decision) string {
	return fmt.Sprintf("%s turn %d %s at %v", decision.Unit, decision.Turn, decision.Skill, decision.Aim)
}

// sameStream fails when two watchers of one match were handed different things.
func sameStream(t *testing.T, first, second *spectator) {
	t.Helper()
	one, two := decisionsIn(first.bodies()), decisionsIn(second.bodies())
	if len(one) != len(two) {
		t.Errorf("%s was handed %d bodies and %s %d, so one of them was sent something twice "+
			"or missed something", first.name, len(one), second.name, len(two))
	}
	for at := 0; at < len(one) && at < len(two); at++ {
		if one[at] != two[at] {
			t.Fatalf("at body %d %s holds %q and %s holds %q", at, first.name, one[at], second.name, two[at])
		}
	}
}

// TestAWatcherIsHandedTheStartAndEveryTurnOfAWholeMatch is step four's headline:
// a real spectator over a real listener, watching a bo3 from the first message
// to the last.
//
// ⚠️ **The counts come from the players, not from this test's own arithmetic.**
// Mirror.Compared is how many wire.Turn bodies each client checked a digest for,
// and the room's own reading says how many battles were played — so "a start a
// battle and a turn a turn" is checked against two numbers produced by somebody
// else. A test that asserted only "some bodies arrived, in a plausible order"
// would pass on a transport that dropped every other turn.
//
// A **bo3** rather than a bo1, because the record holds one wire.Start per
// battle and a bo1 cannot tell a start-per-battle from a start-per-match.
func TestAWatcherIsHandedTheStartAndEveryTurnOfAWholeMatch(t *testing.T) {
	dependencies := deps(t)
	configuration := watchable(config(11, 3, room.DefaultAllowance))
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	watching := held.watch(t, code, "the.watcher", dependencies)
	if tables := held.server.Tables(); tables != 1 {
		t.Fatalf("a watcher alone left the server holding %d tables, want one: a watcher is a "+
			"connection like any other and holds the table it is on", tables)
	}

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	ctx := context.Background()
	hostPlay := play(ctx, host, rating(host))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	// The match ended, so the transport has let go of every cursor it was
	// holding — which from this end is the connection closing.
	watching.ended(t)

	bodies := watching.bodies()
	shape := shapeOf(t, watching.name, bodies)
	played, compared := len(done.reading.Played), host.Mirror().Compared()
	if shape.starts != played {
		t.Errorf("the watcher was handed %d starts and the room played %d battles", shape.starts, played)
	}
	if shape.turns != compared {
		t.Errorf("the watcher was handed %d turns and each client checked %d digests: every turn "+
			"the players were sent is a turn the record holds", shape.turns, compared)
	}
	if len(bodies) != played+compared {
		t.Errorf("the watcher holds %d bodies and the match was %d starts and %d turns",
			len(bodies), played, compared)
	}
	// In order, and the order is what a mirror needs: a battle's start before any
	// of its turns, or the client has decisions and nothing to apply them to.
	if len(bodies) == 0 {
		t.Fatal("the watcher was handed nothing at all")
	}
	opening, isStart := startAt(bodies, 0)
	if !isStart {
		t.Fatalf("the first body a watcher is handed is a %s, want the start of the first battle",
			bodies[0].Kind())
	}
	if opening.Battle != 1 || opening.Seed != configuration.SeedFor(1) {
		t.Errorf("the first start opens battle %d from seed %d, want battle 1 from seed %d",
			opening.Battle, opening.Seed, configuration.SeedFor(1))
	}
	// ⚠️ The side is the **host's**, which is the room's own decision (a watcher
	// plays neither half, and a start with no side reads as SideAlly downstream).
	// It is read off the host's own mirror rather than written down here.
	if want := host.Mirror().Fought()[0].Side; opening.Side != want {
		t.Errorf("the recorded start puts a watcher on the %s side and the host played %s",
			opening.Side, want)
	}
	battles := 0
	for at := range bodies {
		start, isStart := startAt(bodies, at)
		if !isStart {
			continue
		}
		battles++
		if start.Battle != battles {
			t.Errorf("body %d opens battle %d and it is number %d in the stream of starts",
				at, start.Battle, battles)
		}
		if _, next := startAt(bodies, at+1); next {
			t.Errorf("two starts in a row at body %d, so a battle was opened with no turns in it", at)
		}
	}
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors over a watched match: %q", len(said), said)
	}
	held.emptied(t)
	t.Logf("a watcher of a bo%d was handed %d bodies: %d starts and %d turns, against %d battles "+
		"played and %d digests checked by each client",
		configuration.Battles, len(bodies), shape.starts, shape.turns, played, compared)
}

// TestAWatcherJoiningHalfwayIsHandedTheWholeMatchAndThenKeepsUp is the ordering
// claim, and it is the one most likely to be got wrong: a watcher that arrives
// mid-match has to be handed the record **from nought** before it can be handed
// anything new, and then exactly the bodies that follow.
//
// ⚠️ **What makes it hold is a lock rather than an arrangement of calls.** The
// catch-up read, the write of it and the connection being added to the table all
// happen under the table's exchange lock, and every body recorded after that
// point is produced by an exchange that has to take the same lock. So there is no
// window for a body to be recorded between the read and the join, which is where
// a skip would come from, and none for a body to be both caught up and forwarded,
// which is where a duplicate would.
//
// The measurement is that the late watcher's stream is **the same stream** as the
// one that was there from the first message — same length, same bodies, same
// order — against a match that had genuinely started, which is what the digest
// count asserted at the moment it joins is for.
func TestAWatcherJoiningHalfwayIsHandedTheWholeMatchAndThenKeepsUp(t *testing.T) {
	dependencies := deps(t)
	configuration := watchable(config(11, 1, room.DefaultAllowance))
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	early := held.watch(t, code, "the.early.watcher", dependencies)

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	ctx := context.Background()
	choose, halfway := stepped(rating(host), 12)
	hostPlay := play(ctx, host, choose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, halfway, "the host's twelfth decision")
	// The match is genuinely under way, which is the premise the whole test rests
	// on: a late watcher that joined an empty record would measure nothing.
	if compared := host.Mirror().Compared(); compared < 12 {
		t.Fatalf("the host had checked %d digests when the late watcher joined, so there was "+
			"nothing for it to catch up on", compared)
	}
	behind := host.Mirror().Compared()
	late := held.watch(t, code, "the.late.watcher", dependencies)
	// Its catch-up is the whole record and arrives before anything new, so the
	// bodies it holds by the time it holds any at all already outnumber the turns
	// that had been played when it dialled.
	caught := late.awaits(t, behind, "the record it joined halfway through")
	if _, isStart := startAt(caught, 0); !isStart {
		t.Fatalf("the first body a late watcher is handed is a %s, and without the start it has "+
			"decisions and no battle to apply them to", caught[0].Kind())
	}

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	early.ended(t)
	late.ended(t)

	played, compared := len(done.reading.Played), host.Mirror().Compared()
	for _, watching := range []*spectator{early, late} {
		bodies := watching.bodies()
		shape := shapeOf(t, watching.name, bodies)
		if shape.starts != played || shape.turns != compared {
			t.Errorf("%s holds %d starts and %d turns; the match was %d battles and %d turns",
				watching.name, shape.starts, shape.turns, played, compared)
		}
	}
	// Nothing twice and nothing skipped: the two streams are equal body for body,
	// and one of them was there from the beginning.
	sameStream(t, early, late)
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors: %q", len(said), said)
	}
	t.Logf("a watcher joining after %d turns was handed the same %d bodies as one that was there "+
		"from the first message (%d starts, %d turns)",
		behind, len(late.bodies()), played, compared)
}

// TestTwoWatchersAtDifferentPositionsEachGetEverythingTheyAreOwed is why the
// room's read is a Since with a caller-held cursor and **not** a Drain.
//
// Three watchers join at three different points of one match. A read that
// emptied what it handed out would let whichever of them read first decide what
// the others never see — so the measurement is that all three end up holding the
// same complete stream, having started from three different places in it.
func TestTwoWatchersAtDifferentPositionsEachGetEverythingTheyAreOwed(t *testing.T) {
	dependencies := deps(t)
	configuration := watchable(config(11, 1, room.DefaultAllowance))
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	first := held.watch(t, code, "the.first.watcher", dependencies)

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	ctx := context.Background()
	choose, early := stepped(rating(host), 5)
	choose, late := stepped(choose, 15)
	hostPlay := play(ctx, host, choose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, early, "the host's fifth decision")
	atFive := host.Mirror().Compared()
	second := held.watch(t, code, "the.second.watcher", dependencies)
	reachedOr(t, late, "the host's fifteenth decision")
	atFifteen := host.Mirror().Compared()
	third := held.watch(t, code, "the.third.watcher", dependencies)

	if atFive == 0 || atFifteen <= atFive {
		t.Fatalf("the two late watchers joined at %d and %d turns played, so they did not join "+
			"at different positions and this measures one case twice", atFive, atFifteen)
	}

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	watchers := []*spectator{first, second, third}
	for _, watching := range watchers {
		watching.ended(t)
	}
	played, compared := len(done.reading.Played), host.Mirror().Compared()
	for _, watching := range watchers {
		shape := shapeOf(t, watching.name, watching.bodies())
		if shape.starts != played || shape.turns != compared {
			t.Errorf("%s holds %d starts and %d turns; the match was %d battles and %d turns",
				watching.name, shape.starts, shape.turns, played, compared)
		}
	}
	sameStream(t, first, second)
	sameStream(t, first, third)
	t.Logf("three watchers joined at 0, %d and %d turns played and each holds the same %d bodies",
		atFive, atFifteen, len(first.bodies()))
}

// TestNoAllowanceIsEverArmedForAWatcher is the claim that a watcher cannot touch
// the clock, and it is measured with allowance.armed because **nothing else in
// this package can see a clock at all**: a test whose clients answer never needs
// one, which is how a broken allowance shipped once already.
//
// ⚠️ **The case that matters is a watcher arriving while a seat is being asked**,
// which is where the damage would be. Every exchange a seat has re-arms the clock
// off the room's own reading; if a watcher's join or a watcher's message took the
// same path, each one would restart the allowance of the player on turn — so a
// spectator could keep a stalling opponent alive indefinitely, and a suite whose
// clients answer would never see it. Both readings are taken by value, the
// generation included, because a re-arm is a *new* timer on the *same* seat and
// only the generation tells the two apart.
func TestNoAllowanceIsEverArmedForAWatcher(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	// A long allowance: this test is about the clock being armed, never about it
	// firing.
	code := held.open(t, watchable(config(11, 1, 600)), dependencies)

	// Nobody plays: both seats are dialled and neither answers, so the room sits
	// waiting on the host with its allowance armed.
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	entry := held.tableFor(t, code)
	// ⚠️ Waited for rather than read straight off, because Dial returns as soon
	// as it has read the welcome and the clock is armed a line later on the
	// server's own goroutine. Reading here would be reading a race and would
	// usually see nothing armed at all.
	generation := awaitArmed(t, entry)
	reading, known := held.rooms.Read(code)
	if !known || !reading.Waiting {
		t.Fatalf("the room is not waiting on anybody (known=%v), so there is no seat's clock to "+
			"leave alone", known)
	}
	asking := reading.Awaiting

	// Three watchers attach while that seat is being asked.
	watchers := make([]*spectator, 0, 3)
	for at := range 3 {
		watchers = append(watchers, held.watch(t, code, fmt.Sprintf("watcher.%d", at), dependencies))
		nowArmed, now := entry.allowance.armed()
		if !nowArmed || now != generation {
			t.Errorf("after %d watchers attached the clock is armed=%v at generation %d, and the "+
				"seat being asked had generation %d: a watcher joining re-armed the allowance of "+
				"the player on turn", at+1, nowArmed, now, generation)
		}
	}
	// And a watcher that talks. An act from a watcher is refused by the room, and
	// the refusal must not carry a fresh allowance back with it.
	watching := watchers[0]
	if err := watching.conn.send(context.Background(), wire.Act{Skill: "razor_leaf"}); err != nil {
		t.Fatalf("a watcher sends an act: %v", err)
	}
	refusedWith(t, watching, wire.CodeNotYourTurn)
	nowArmed, now := entry.allowance.armed()
	if !nowArmed || now != generation {
		t.Errorf("after a watcher sent an act the clock is armed=%v at generation %d, want the "+
			"seat's own %d: a message from somebody watching restarted the turn's allowance",
			nowArmed, now, generation)
	}
	// And a watcher **leaving** while a seat is being asked, which is the third
	// moment this could happen at and the one no other test here can see.
	//
	// ⚠️ It is what makes the branch in Server.parted measurable at all. A
	// watcher must not reach room.Left — that call tells the room a *seat* went
	// away — and room.Left happens to ignore a report from no seat, so reporting
	// one changes nothing about the room and nothing about either player's
	// stream. What it does change is the clock: the transport re-arms the
	// allowance after every departure it reports, so a watcher routed through
	// that path restarts the turn of whoever is being asked. Measured: with the
	// branch removed, the whole suite stayed green except for this reading.
	watchers[1].Close()
	for waited := 0; waited < 600 && entry.watching() != 2; waited++ {
		time.Sleep(10 * time.Millisecond)
	}
	if carrying := entry.watching(); carrying != 2 {
		t.Errorf("the table carries %d watchers six seconds after one of three left", carrying)
	}
	leftArmed, afterLeaving := entry.allowance.armed()
	if !leftArmed || afterLeaving != generation {
		t.Errorf("after a watcher left, the clock is armed=%v at generation %d, want the seat's "+
			"own %d: a watcher going away was reported as a seat going away",
			leftArmed, afterLeaving, generation)
	}

	// The seat being asked has not moved either, which is the other half of "the
	// clock is still the seat's".
	after, known := held.rooms.Read(code)
	if !known || !after.Waiting || after.Awaiting != asking {
		t.Errorf("the room was waiting on %s and is now waiting on %s (waiting=%v, known=%v)",
			asking, after.Awaiting, after.Waiting, known)
	}
	t.Logf("three watchers attached and one sent an act while %s was being asked; the clock stayed "+
		"armed at generation %d throughout", asking, generation)
}

// awaitArmed waits for the clock on the seat being asked to be armed and hands
// back the generation it is armed under, which is what a later reading has to
// match for the clock to be the *same* one rather than a fresh one on the same
// seat.
func awaitArmed(t *testing.T, entry *table) uint64 {
	t.Helper()
	for waited := 0; waited < 600; waited++ {
		if armed, generation := entry.allowance.armed(); armed {
			return generation
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no clock is armed on a room with both seats taken and a turn open, so this test " +
		"cannot tell an untouched clock from a restarted one")
	return 0
}

// refusedWith waits for a refusal carrying one code to reach a watcher.
func refusedWith(t *testing.T, watching *spectator, code wire.Code) {
	t.Helper()
	for waited := 0; waited < 600; waited++ {
		for _, body := range watching.bodies() {
			switch refused := body.(type) {
			case wire.Refused:
				if refused.Code == code {
					return
				}
			case *wire.Refused:
				if refused.Code == code {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s was never refused with %q; it holds %v", watching.name, code, decisionsIn(watching.bodies()))
}

// TestTheWatcherOverTheCapIsRefusedAndTheRestAreUndisturbed holds the bound and
// the refusal that carries it.
//
// ⚠️ **The refusal is the transport's own and it has a code of its own**, which
// is a decision rather than a spare constant: wire.CodeRoomFull is worded in both
// books as two players already having the board and then advises waiting for
// their match to end or opening a room of your own — nonsense advice for somebody
// who came to watch *this* match. → wire.CodeTooManyWatchers.
//
// The second half is what stops the cap being implemented by dropping somebody:
// the watchers already attached are still there afterwards and still being
// written to, measured by starting the match and watching every one of them
// receive its opening.
func TestTheWatcherOverTheCapIsRefusedAndTheRestAreUndisturbed(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, watchable(config(11, 1, room.DefaultAllowance)), dependencies)

	watchers := make([]*spectator, 0, MaxWatchers)
	for at := range MaxWatchers {
		watchers = append(watchers, held.watch(t, code, fmt.Sprintf("watcher.%d", at), dependencies))
	}
	entry := held.tableFor(t, code)
	if carrying := entry.watching(); carrying != MaxWatchers {
		t.Fatalf("the table carries %d watchers after %d joined, so the cap below is measured "+
			"against the wrong number", carrying, MaxWatchers)
	}

	// The one over the cap. Dial itself is what answers, because a refused join
	// is the end of the conversation and the code is the whole of what a client
	// needs in order to say why.
	_, err := Dial(context.Background(), code, watchingHello(t, "the.unlucky.watcher"),
		dependencies.Books, ClientOptions{Timings: held.timings})
	if err == nil {
		t.Fatal("a ninth watcher was welcomed into a room that carries eight")
	}
	refusal, refused := err.(*Refusal)
	if !refused {
		t.Fatalf("a watcher over the cap was turned away with %v, want a *Refusal carrying a code", err)
	}
	if refusal.Code != wire.CodeTooManyWatchers {
		t.Errorf("a watcher over the cap was refused with %q, want %q: %q is worded about the two "+
			"seats and advises opening a room of your own, which is not what somebody who came to "+
			"watch this match can do",
			refusal.Code, wire.CodeTooManyWatchers, wire.CodeRoomFull)
	}
	if carrying := entry.watching(); carrying != MaxWatchers {
		t.Errorf("the table carries %d watchers after one was refused, want the %d that were "+
			"already there", carrying, MaxWatchers)
	}

	// Undisturbed: every one of them is still connected and still being written
	// to, which the opening of the match is what proves.
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	for _, watching := range watchers {
		bodies := watching.awaits(t, 1, "the start of the match it was already watching")
		if _, isStart := startAt(bodies, 0); !isStart {
			t.Errorf("%s was handed a %s when the match opened", watching.name, bodies[0].Kind())
		}
	}
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors while refusing one watcher: %q", len(said), said)
	}
	t.Logf("%d watchers attached, the next was refused %q, and all %d were handed the opening",
		MaxWatchers, refusal.Code, MaxWatchers)
}

// TestAMatchWithWatchersAttachedIsTheSameMatch is the transport's twin of
// internal/room's TestAMatchPlayedWithAWatcherReadingIsTheSameMatch, and it is
// the only test here that could see the failure this whole design exists to
// avoid.
//
// ⚠️ **A watcher threaded through the seats would pass everything else.** The
// roster would still be legal, both players would still agree on every digest,
// and the match would still end — it would simply be a *different* match, because
// the order the seats are visited in reaches the roster and the roster's order
// decides which side wins a speed tie. Only a comparison against the same match
// played with nobody watching can see it.
//
// What is compared: every decision both clients took, in the order they took
// them; each battle's seed, side and outcome as each client's own engine settled
// it; how many digests were checked; and the room's own result, standing and
// skipped-prompt count. Two runs of one seed with one rating differ in none of
// those unless the battle itself differed.
func TestAMatchWithWatchersAttachedIsTheSameMatch(t *testing.T) {
	alone := aWatchedMatch(t, 0)
	watched := aWatchedMatch(t, 3)

	if len(alone.decisions) == 0 {
		t.Fatal("the unwatched match took no decisions, so the comparison below measures nothing")
	}
	if len(alone.decisions) != len(watched.decisions) {
		t.Fatalf("the match took %d decisions with nobody watching and %d with three watchers "+
			"attached", len(alone.decisions), len(watched.decisions))
	}
	for at := range alone.decisions {
		if alone.decisions[at] != watched.decisions[at] {
			t.Fatalf("decision %d was %q with nobody watching and %q with three watchers attached",
				at, alone.decisions[at], watched.decisions[at])
		}
	}
	if alone.compared != watched.compared {
		t.Errorf("each client checked %d digests with nobody watching and %d with three watchers",
			alone.compared, watched.compared)
	}
	if len(alone.fought) != len(watched.fought) {
		t.Fatalf("%d battles were fought with nobody watching and %d with three watchers",
			len(alone.fought), len(watched.fought))
	}
	for at := range alone.fought {
		one, two := alone.fought[at], watched.fought[at]
		if one.Seed != two.Seed || one.Side != two.Side || one.Outcome != two.Outcome {
			t.Errorf("battle %d was %s with nobody watching and %s with three watchers",
				at+1, side(one), side(two))
		}
	}
	if alone.result != watched.result {
		t.Errorf("the match ended %+v with nobody watching and %+v with three watchers",
			alone.result, watched.result)
	}
	if alone.skipped != watched.skipped {
		t.Errorf("%d prompts were skipped with nobody watching and %d with three watchers",
			alone.skipped, watched.skipped)
	}
	t.Logf("the same %d decisions, the same %d digests and the same result (%+v) with nobody "+
		"watching and with three watchers attached", len(alone.decisions), alone.compared, alone.result)
}

// playedOut is one run of a match, as everything about it that two runs have to
// agree on.
type playedOut struct {
	decisions []string
	fought    []Fought
	compared  int
	result    room.Result
	skipped   int
}

// aWatchedMatch plays one whole match with a given number of watchers attached
// before it starts, and hands back what it came to.
//
// ⚠️ The decisions are recorded through a **shared** list under one mutex rather
// than one list per client, because the order matters and the two clients take
// their turns strictly one after another: what is being compared between runs is
// the sequence of the whole match, and two separate lists would compare each half
// against itself and miss a match whose halves swapped.
func aWatchedMatch(t *testing.T, watchers int) playedOut {
	t.Helper()
	dependencies := deps(t)
	configuration := watchable(config(11, 1, room.DefaultAllowance))
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	for at := range watchers {
		held.watch(t, code, fmt.Sprintf("watcher.%d", at), dependencies)
	}

	var taken struct {
		sync.Mutex
		decisions []string
	}
	recording := func(who string, choose battle.Chooser) battle.Chooser {
		return func(prompt *battle.Prompt) (battle.Choice, bool) {
			choice, decided := choose(prompt)
			taken.Lock()
			taken.decisions = append(taken.decisions,
				fmt.Sprintf("%s %s turn %d %s at %v", who, prompt.Unit, prompt.Turn, choice.Skill, choice.Aim))
			taken.Unlock()
			return choice, decided
		}
	}

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	ctx := context.Background()
	hostPlay := play(ctx, host, recording("host", rating(host)))
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, recording("guest", rating(guest)))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match with %d watchers: %v", watchers, err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match with %d watchers: %v", watchers, err)
	}
	done := held.finished(t)
	taken.Lock()
	defer taken.Unlock()
	return playedOut{
		decisions: append([]string(nil), taken.decisions...),
		fought:    host.Mirror().Fought(),
		compared:  host.Mirror().Compared(),
		result:    done.reading.Result,
		skipped:   done.reading.Skipped,
	}
}

// TestAWatchersActIsRefusedAndTheMatchIsWhereItWas holds that a message from
// somebody watching goes to the **room** and is refused there.
//
// ⚠️ **The transport must not declare that rule a second time.** room.Deliver
// already refuses a peer with no seat, with wire.CodeNotYourTurn, and it leaves
// the prompt open and the battle where it was — internal/room has the test for
// the refusal itself. What is measured here is that the message reaches that rule
// at all, that the answer comes back down the watcher's own socket, and that
// nothing about the match moved: the same seat is still being asked, the record
// did not grow, and the two players were sent nothing.
func TestAWatchersActIsRefusedAndTheMatchIsWhereItWas(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, watchable(config(11, 1, 600)), dependencies)

	watching := held.watch(t, code, "the.talkative.watcher", dependencies)
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)

	opening := watching.awaits(t, 1, "the start of the match")
	before, known := held.rooms.Read(code)
	if !known {
		t.Fatal("the room went away before the watcher said anything")
	}

	// Three ways of taking a turn, none of which is this connection's to take.
	for _, sent := range []wire.Body{
		wire.Act{Skill: "razor_leaf"},
		wire.Pass{},
		wire.Decide{},
	} {
		if err := watching.conn.send(context.Background(), sent); err != nil {
			t.Fatalf("a watcher sends a %s: %v", sent.Kind(), err)
		}
		refusedWith(t, watching, wire.CodeNotYourTurn)
	}

	after, known := held.rooms.Read(code)
	if !known {
		t.Fatal("a watcher's act ended the room")
	}
	if after.Awaiting != before.Awaiting || after.Waiting != before.Waiting {
		t.Errorf("the room was waiting on %s (waiting=%v) and after three messages from a watcher "+
			"it waits on %s (waiting=%v)", before.Awaiting, before.Waiting, after.Awaiting, after.Waiting)
	}
	if after.Finished != before.Finished || after.Skipped != before.Skipped {
		t.Errorf("the room's reading moved: finished %v→%v, skipped %d→%d",
			before.Finished, after.Finished, before.Skipped, after.Skipped)
	}
	// The record did not grow either, which is what "the watcher saw nothing
	// happen" means: it holds the opening it already had, plus its own refusals.
	bodies := watching.bodies()
	shape := shapeOf(t, watching.name, dropRefusals(bodies))
	if shape.starts != len(opening) || shape.turns != 0 {
		t.Errorf("the watcher holds %d starts and %d turns after sending three messages, want the "+
			"%d it opened with and no turns", shape.starts, shape.turns, len(opening))
	}
	t.Logf("three messages from a watcher were each refused %q and the room is still asking %s",
		wire.CodeNotYourTurn, after.Awaiting)
}

// dropRefusals is a stream with the answers to this connection's own messages
// taken out, leaving the record it was handed.
func dropRefusals(bodies []wire.Body) []wire.Body {
	out := make([]wire.Body, 0, len(bodies))
	for _, body := range bodies {
		switch body.(type) {
		case wire.Refused, *wire.Refused:
		default:
			out = append(out, body)
		}
	}
	return out
}

// TestAShutdownTellsEveryWatcher is the other half of the shutdown's promise: a
// socket that simply dies leaves a person staring at a dead connection, and that
// is as true of somebody watching as of somebody playing.
func TestAShutdownTellsEveryWatcher(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, watchable(config(11, 1, room.DefaultAllowance)), dependencies)

	watchers := []*spectator{
		held.watch(t, code, "watcher.0", dependencies),
		held.watch(t, code, "watcher.1", dependencies),
	}
	held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	for _, watching := range watchers {
		watching.awaits(t, 1, "the start of the match it is watching")
	}

	ctx, cancel := context.WithTimeout(context.Background(), theWholeMatch)
	defer cancel()
	if err := held.server.Shutdown(ctx); err != nil {
		t.Fatalf("shut the server down: %v", err)
	}
	for _, watching := range watchers {
		watching.ended(t)
		told := false
		for _, body := range watching.bodies() {
			switch closed := body.(type) {
			case wire.Closed:
				told = told || closed.Reason == wire.ClosureStopped
			case *wire.Closed:
				told = told || closed.Reason == wire.ClosureStopped
			}
		}
		if !told {
			t.Errorf("%s was never told why the match stopped; it holds %v",
				watching.name, decisionsIn(watching.bodies()))
		}
	}
	if tables := held.server.Tables(); tables != 0 {
		t.Errorf("the server still holds %d tables after a shutdown that returned", tables)
	}
}

// TestAWatcherLeavingEndsNothing is the claim a spectator's whole design rests
// on from the players' side: somebody watching is not a participant, so their
// leaving frees no seat and ends no match.
//
// ⚠️ **A watcher must not reach room.Left**, which is the thing this would catch:
// that call tells the room a *seat* went away, and with no rejoin a seat going
// away ends the match. So the watcher here leaves in the middle of a battle, and
// what is measured is that the match carries on to a real ending afterwards — a
// verdict from the board rather than room.VerdictAbandoned.
func TestAWatcherLeavingEndsNothing(t *testing.T) {
	dependencies := deps(t)
	configuration := watchable(config(11, 1, room.DefaultAllowance))
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	watching := held.watch(t, code, "the.watcher.who.leaves", dependencies)
	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	ctx := context.Background()
	choose, halfway := stepped(rating(host), 8)
	hostPlay := play(ctx, host, choose)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	guestPlay := play(ctx, guest, rating(guest))

	reachedOr(t, halfway, "the host's eighth decision")
	watching.Close()
	watching.ended(t)
	// The table let go of the watcher and is still holding the two players.
	entry := held.tableFor(t, code)
	for waited := 0; waited < 200 && entry.watching() != 0; waited++ {
		time.Sleep(10 * time.Millisecond)
	}
	if carrying := entry.watching(); carrying != 0 {
		t.Errorf("the table still carries %d watchers two seconds after the only one left", carrying)
	}

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	// The ending is the board's, not a departure's — which is what a watcher
	// reaching room.Left would have produced.
	if done.reading.Result.Departed.Valid() {
		t.Errorf("the match recorded %q as having gone away, and the only connection that left "+
			"was watching", done.reading.Result.Departed)
	}
	switch done.reading.Result.Verdict {
	case room.VerdictWon, room.VerdictDrawn:
	default:
		t.Errorf("a match somebody stopped watching halfway through ended %q",
			done.reading.Result.Verdict)
	}
	for _, client := range []*Client{host, guest} {
		if closure, closed := client.Mirror().Closure(); closed {
			t.Errorf("%s was told the match closed (%s) because somebody stopped watching",
				client.Seat(), closure)
		}
	}
	t.Logf("a watcher left after 8 decisions and the match ran on to %q over %d battles",
		done.reading.Result.Verdict, len(done.reading.Played))
}

// TestATableIsStillTwoSeatsAndAFixedWalk is the guard on the two of the three
// twos this step is on the other side of.
//
// ⚠️ **Watchers are a collection and seats are not, and the whole design depends
// on that staying true.** seatCount's comment one layer out says why: the order
// the two seats are visited in reaches the roster, and the roster's order decides
// which side wins a speed tie — so the shutdown walking a fixed-size array of two
// named fields is the shape that cannot be widened by accident. A watcher added
// to that walk would be a third seat wearing a spectator's name.
//
// Three claims, all mechanical: the constant is two; **every** place this
// package walks a literal walks an **array** — a composite literal with a
// length — rather than a slice literal whose length is a coincidence; and there
// are exactly two such places, the shutdown telling both peers and the draft's
// seat index, both of which are the same two seats under the same rule.
//
// ⚠️ The count is asserted rather than only the shapes, because the way this
// gets broken is a **third** walk appearing: somebody wanting to tell "everyone
// at the table" writes one over a slice built from the seats and the watchers,
// and that is precisely the third-citizen change the two constants refuse. A
// watcher is walked as a collection of its own, from its own field, and is not
// one of these.
//
// There was no test holding any of it before this step, which is why this is a
// new one rather than an extension: `seatsPerTable` had two readers — that walk
// and a fixture's arithmetic — and nothing that would fail if it moved.
func TestATableIsStillTwoSeatsAndAFixedWalk(t *testing.T) {
	if seatsPerTable != 2 {
		t.Errorf("a table holds %d seats: a room has exactly two, and a watcher is not one of "+
			"them — → room's own seatCount", seatsPerTable)
	}
	if MaxWatchers < 2 {
		t.Errorf("the watcher cap is %d, which is not a cap on watching so much as its absence",
			MaxWatchers)
	}

	within := map[string]int{}
	scanned := 0
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
			ast.Inspect(function.Body, func(node ast.Node) bool {
				ranged, isRange := node.(*ast.RangeStmt)
				if !isRange {
					return true
				}
				literal, isLiteral := ranged.X.(*ast.CompositeLit)
				if !isLiteral {
					return true
				}
				array, isArray := literal.Type.(*ast.ArrayType)
				if !isArray {
					return true
				}
				if array.Len == nil {
					t.Errorf("%s:%d walks a slice literal inside %s, and the two seats are walked "+
						"as a fixed-size array so that the length is a declaration rather than a "+
						"coincidence", name, fileSet.Position(ranged.Pos()).Line, function.Name.Name)
					return true
				}
				within[function.Name.Name]++
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("the walk read no source files, so it measures nothing")
	}
	if within["stopping"] != 1 {
		t.Errorf("the shutdown takes %d fixed-size walks over a literal, want the one over the "+
			"two seats: a room has exactly two and the order they are told in is an output",
			within["stopping"])
	}
	if len(within) != 2 {
		t.Errorf("%d functions walk a literal of seats (%v), want the two that do: the shutdown "+
			"and the draft's seat index. A third is somebody walking everybody at the table, "+
			"which is a watcher becoming a seat", len(within), within)
	}
	t.Logf("scanned %d source files: %v, every one of them a fixed-size array", scanned, within)
}
