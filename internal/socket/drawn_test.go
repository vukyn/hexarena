package socket

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// # Three claims about a Mirror being drawn while it is stepped
//
// The full-screen client runs Play on its own goroutine and redraws on
// bubbletea's, so it is the first thing in this repository that *draws* a battle
// another goroutine is stepping. What that needs of this package is a lock, a
// consistent reading under it, and a hook saying there is something new — and
// each of the three has a way of being wrong that the other two cannot see.

// TestAMirrorIsSafeToDrawWhileItIsStepped is the lock, over a real match.
//
// A second goroutine calls Mirror.Read in a tight loop for the whole of a bo1
// while Play steps it. Under -race an unsynchronised read of the battle, the
// prompt, the events or the fought list is a failure; without it this is a
// smoke test that the two do not deadlock, which is worth having on its own.
//
// ⚠️ **What it cannot see is a Sight field escaping the callback.** The
// detector fires on a concurrent *use*, and this reads everything inside fn —
// so a caller that kept the *battle.Battle and read it afterwards would pass
// here. That rule is held by Mirror.Read's own doc comment and by nothing else,
// exactly as room.request's closure argument is.
//
// ⚠️ **The reader is counted.** A loop that never ran, or that ran three times
// before the match finished, would pass a test that only asserted no race
// happened — which is the vacuity this repository has recorded more than once.
func TestAMirrorIsSafeToDrawWhileItIsStepped(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, config(11, 1, room.DefaultAllowance), dependencies)

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
		dependencies.Books)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""),
		dependencies.Books)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hostPlaying := play(ctx, host, rating(host))
	guestPlaying := play(ctx, guest, rating(guest))

	// The reader runs until the host's loop returns, which is the whole match.
	var reads atomic.Int64
	var seen atomic.Int64
	drawing := make(chan struct{})
	go func() {
		defer close(drawing)
		for {
			select {
			case <-hostPlaying.done:
				return
			default:
			}
			host.Mirror().Read(func(sight Sight) {
				reads.Add(1)
				if sight.Fight == nil {
					return
				}
				// Everything a screen would touch, inside the callback: the
				// board, the queue, the history and the prompt.
				_ = sight.Fight.Units()
				_ = sight.Fight.Queue()
				_, _ = sight.Fight.Since(0)
				if sight.Asking != nil {
					seen.Add(1)
					_ = sight.Asking.Options
				}
			})
		}
	}()

	if err := hostPlaying.wait(t, "the host"); err != nil {
		t.Fatalf("the host's loop: %v", err)
	}
	if err := guestPlaying.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's loop: %v", err)
	}
	<-drawing

	if reads.Load() == 0 {
		t.Fatal("the reader never ran, so nothing was drawn beside a match and this " +
			"measures no concurrency at all")
	}
	if seen.Load() == 0 {
		t.Error("the reader never caught the mirror with a prompt open, so the one field a " +
			"screen enables input off was read by nothing")
	}
	if fought := host.Mirror().Fought(); len(fought) == 0 {
		t.Fatal("the match settled no battle, so the reader ran beside nothing")
	}
	t.Logf("%d readings beside a whole match, %d of them with a prompt open",
		reads.Load(), seen.Load())
}

// TestDecideDoesNotHoldTheLockAcrossTheChooser is the one ordering that must not
// be got wrong, and it took two goes to write something that could see it.
//
// ⚠️ **The obvious test does NOT catch it, and the reason is worth writing
// down.** A first version let a chooser call Mirror.Read on a real match and
// asserted no hang — and holding the read lock across choose passes that
// perfectly, because sync.RWMutex admits several readers and the only writer in
// a client is Receive, which runs on the goroutine that is blocked in the
// chooser. So the deadlock the plan predicted "on the very first turn" cannot
// happen with nobody writing, and a test built on that prediction measured
// nothing. **Measured**: inverting the release passes it.
//
// What makes the hold fatal is a **writer arriving while the chooser waits**.
// Go's RWMutex queues a waiting writer ahead of new readers, so:
//
//	Play      RLock (inside Decide) ─── blocked in choose, holding it
//	somebody  Lock  ────────────────── waits for the reader
//	renderer  RLock ────────────────── waits BEHIND the writer
//
// and the renderer is what the chooser is waiting for, because a player answers
// what they can see. That is the shape below: a mirror at an open turn, a
// chooser that blocks, a Receive from another goroutine, and a Read that has to
// come back. It is bounded rather than left to hang, so the failure is a message
// naming which of the three is stuck.
//
// *Sees:* the read lock held across the chooser.
// *Cannot see:* a lock held too long anywhere that is not the chooser, which is
// harmless — nothing else in this package calls out to a caller.
func TestDecideDoesNotHoldTheLockAcrossTheChooser(t *testing.T) {
	mirror := aMirrorAtAnOpenTurn(t)

	inChooser := make(chan struct{})
	release := make(chan struct{})
	decided := make(chan struct{})
	go func() {
		defer close(decided)
		mirror.Decide(func(*battle.Prompt) (battle.Choice, bool) {
			close(inChooser)
			<-release
			return battle.Choice{}, false
		})
	}()
	<-inChooser

	// A writer, which is what turns a held read lock into a queue.
	written := make(chan struct{})
	go func() {
		defer close(written)
		if err := mirror.Receive(wire.Refused{Code: wire.CodeNotYourTurn}); err != nil {
			t.Errorf("the mirror refused a refusal: %v", err)
		}
	}()

	// And the renderer, which is what the player is waiting to see.
	drawn := make(chan struct{})
	go func() {
		defer close(drawn)
		mirror.Read(func(Sight) {})
	}()

	const bounded = 5 * time.Second
	select {
	case <-drawn:
	case <-time.After(bounded):
		close(release)
		t.Fatalf("a renderer could not read the mirror within %s while a chooser was "+
			"waiting for the player, which is the client deadlocked against itself: the "+
			"read lock is being held across the chooser", bounded)
	}
	select {
	case <-written:
	case <-time.After(bounded):
		close(release)
		t.Fatalf("a message could not be taken in within %s while a chooser was waiting", bounded)
	}
	close(release)
	select {
	case <-decided:
	case <-time.After(bounded):
		t.Fatalf("the chooser never returned within %s", bounded)
	}
	// And the writer really did write, so the queue above was a real one rather
	// than a Receive that declined.
	if refusals := mirror.Refusals(); len(refusals) != 1 {
		t.Errorf("the mirror recorded %d refusals, want the one the writer sent — without "+
			"it no writer was ever queued and this measures nothing", len(refusals))
	}
}

// aMirrorAtAnOpenTurn is a mirror welcomed, started and stopped on a turn of its
// own side, built out of messages rather than over a socket: what the test above
// is about is the lock, and a listener would put a network's timing into it.
//
// ⚠️ **Which SIDE is searched for, and the seed is not.** A mirror whose first
// prompt belongs to the other half of the board has no turn to be asked about,
// so the chooser would never be reached and the whole test would pass on
// nothing — and the obvious fix, walking seeds, was tried and **measured to do
// nothing**: twenty seeds all opened on the same side, because the opening turn
// order comes off the speeds and the seed only moves the rolls. What actually
// varies is which half a seat plays, which is a fact about the seat rather than
// about the battle, and it is exactly what wire.Start.Side carries.
func aMirrorAtAnOpenTurn(t *testing.T) *Mirror {
	t.Helper()
	dependencies := deps(t)
	home, err := theHostSquad(t, dependencies.Characters).Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the home side: %v", err)
	}
	away, err := theGuestSquad(t, dependencies.Characters).Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the away side: %v", err)
	}
	roster := append(home, away...)
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		mirror := NewMirror(wire.SeatHost, dependencies.Books, dependencies.Characters)
		welcome := wire.Welcome{
			Format: wire.Format3v3, Battles: 1, Allowance: room.DefaultAllowance,
			TurnCap: room.DefaultTurnCap, Seat: wire.SeatHost,
		}
		if err := mirror.Receive(welcome); err != nil {
			t.Fatalf("welcome the mirror: %v", err)
		}
		if err := mirror.Receive(wire.Start{
			Seed: 11, Roster: roster, Side: side, Battle: 1,
		}); err != nil {
			t.Fatalf("start the mirror on the %s side: %v", side, err)
		}
		if _, asking := mirror.Asking(); asking {
			return mirror
		}
	}
	t.Fatal("neither half of the board opened on a turn this mirror is asked about, so " +
		"there is no chooser to block")
	return nil
}

// TestSteppedIsCalledForEveryMessageAndHoldsNoLock is the redraw hook.
//
// Two claims in one test because neither is worth a fixture on its own:
//
//   - The hook is called with **no lock held**. It calls Mirror.Read, which is
//     what a renderer does the moment it is told there is something new; if the
//     hook were called from inside Receive this would deadlock on the write lock
//     the caller is still holding, and the failure is a hang.
//   - It is called for **every** message Play takes in and not only for a turn.
//     The count is held against what the mirror itself recorded: one start a
//     battle, one per turn compared, and a closed if there was one. A hook wired
//     only to wire.Turn would come back short by the start, which is the message
//     a screen has to see in order to draw a board at all.
//
// *Cannot see:* whether the client actually redraws — that is the client's own
// test, and no hook in this package can state it.
func TestSteppedIsCalledForEveryMessageAndHoldsNoLock(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, config(11, 1, room.DefaultAllowance), dependencies)

	var steps atomic.Int64
	var inside atomic.Int64
	// ⚠️ The hook needs the client and the client needs the hook, which is the
	// same knot the game client's sender is tied with. It is untied the same
	// way: the closure reads a variable the dial below fills in, and nothing
	// calls the hook until Play starts — which is after the assignment and on a
	// goroutine started after it, so there is a happens-before rather than a
	// race the detector would have to be lucky to catch.
	var host *Client
	notify := func() {
		steps.Add(1)
		// A renderer's first move on being told there is something new. Called
		// from inside Receive this would deadlock on the write lock.
		host.Mirror().Read(func(Sight) { inside.Add(1) })
	}

	dialled, err := Dial(context.Background(), code,
		hello(t, theHostSquad(t, dependencies.Characters), "Host", ""),
		dependencies.Books, ClientOptions{Timings: held.timings, Stepped: notify})
	if err != nil {
		t.Fatalf("dial room %s: %v", code, err)
	}
	host = dialled
	t.Cleanup(host.Close)
	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""),
		dependencies.Books)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hostPlaying := play(ctx, host, rating(host))
	guestPlaying := play(ctx, guest, rating(guest))
	if err := hostPlaying.wait(t, "the host"); err != nil {
		t.Fatalf("the host's loop: %v", err)
	}
	if err := guestPlaying.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's loop: %v", err)
	}

	compared := host.Mirror().Compared()
	fought := len(host.Mirror().Fought())
	if compared == 0 || fought == 0 {
		t.Fatalf("the match compared %d digests over %d battles, so the count below is "+
			"held against nothing", compared, fought)
	}
	// One start a battle, one per turn compared. A closed would add one, and a
	// bo1 played to its own end sends none.
	//
	// ⚠️ **The welcome is deliberately NOT in this sum, and that was measured
	// rather than reasoned about.** It arrives during the handshake — Dial reads
	// it and hands it to Receive itself — which is before Play exists and
	// therefore before there is a hook to call. A screen must not wait for a
	// step in order to learn the room's format: it has the welcome the moment
	// Dial returns.
	want := fought + compared
	if got := int(steps.Load()); got != want {
		t.Errorf("the hook fired %d times over a match of %d turns in %d battles, want %d "+
			"— one start a battle and one a turn, the welcome having arrived before Play",
			got, compared, fought, want)
	}
	if inside.Load() != steps.Load() {
		t.Errorf("the hook read the mirror %d times against %d calls, so a reading was "+
			"refused or a call was not counted", inside.Load(), steps.Load())
	}
}

// TestReadRefusesNothingAndTakesNoLockTwice is the small half of Mirror.Read:
// that a nil callback is an ordinary no-op rather than a panic, and that a
// second reader may be inside it at the same time.
//
// A read lock that had been written as a plain Mutex would pass every test above
// — they read from one goroutine at a time in practice — and would then serialise
// a client's Update against its own View. This is the one that fails on it.
func TestReadRefusesNothingAndTakesNoLockTwice(t *testing.T) {
	mirror := NewMirror(wire.SeatHost, battle.Books{}, nil)
	mirror.Read(nil)

	inside := make(chan struct{})
	second := make(chan struct{})
	var once sync.WaitGroup
	once.Add(1)
	go func() {
		defer once.Done()
		mirror.Read(func(Sight) {
			close(inside)
			// Held open until the other reader has got in, which a mutual
			// exclusion would make impossible.
			<-second
		})
	}()
	<-inside
	mirror.Read(func(sight Sight) {
		if sight.Seated {
			t.Error("a mirror nobody welcomed reports itself seated")
		}
	})
	close(second)
	once.Wait()
}
