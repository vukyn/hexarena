package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheSenderIsSafeToAttachWhileSomethingIsSending is the guard for a race the
// detector caught twice, days apart, and that neither sighting reproduced: about
// one run in ten of this package under `-race`, on
// TestTheCountdownReachesTheScreenOverASocket.
//
// `session.out` was written by attach and read by send with no synchronisation,
// on the strength of a comment saying it was "written once, before anything can
// send". That holds in `run`, where the attach precedes `program.Run()`. It does
// not hold in a test that attaches a sender to a session a Play goroutine is
// already running against, which is exactly what clock_test.go does.
//
// ⚠️ **This test is only a test under `-race`.** Without the detector both
// versions pass: an unsynchronised pointer write and read on arm64 does not
// produce a wrong value here, it produces an undefined program. That is why the
// package is on the Makefile's `-race` line, and why this test says so.
//
// *Sees:* either access leaving the mutex.
// *Cannot see:* the original failure. That one needed a socket, a Play goroutine
// and a timer to line up; this drives the two accesses directly, which is what
// makes it reproducible instead of one-in-ten.
func TestTheSenderIsSafeToAttachWhileSomethingIsSending(t *testing.T) {
	const rounds = 200
	sess := newSession()
	// Two counters, because they answer different questions: calls is how many
	// times send ran, and delivered is how many of those found a sender attached.
	var calls, delivered atomic.Int64
	var sending sync.WaitGroup
	sending.Add(1)
	stop := make(chan struct{})
	// ⚠️ The attaches wait for the loop to be genuinely running. Two hundred
	// attaches finish faster than a goroutine starts, so without this the writes
	// were all over before the first read and the detector saw one access.
	started := make(chan struct{})
	go func() {
		defer sending.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			sess.send(matchSteppedMsg{})
			if calls.Add(1) == 1 {
				close(started)
			}
		}
	}()
	<-started
	for range rounds {
		sess.attach(newCountingSender(&delivered))
	}
	// And let the loop run on past the last attach, so a read is still coming
	// while the writes stop rather than only before them.
	for delivered.Load() == 0 {
		runtime.Gosched()
	}
	close(stop)
	sending.Wait()
	// ⚠️ The vacuity guard. A send loop that never ran, or an attach that never
	// landed, would leave the detector nothing to observe and this test green for
	// the wrong reason.
	// ⚠️ The vacuity guard, and it is TWO claims. A send loop that never ran
	// leaves the detector nothing to observe; and sends that all ran before the
	// first attach landed would never have been concurrent with one, so the
	// interesting pair never met. `delivered > 0` is what says a send read a
	// sender that an attach had put there while the loop was going.
	if got := calls.Load(); got == 0 {
		t.Fatal("send never ran, so the detector had nothing to look at")
	}
	if got := delivered.Load(); got == 0 {
		t.Fatalf("%d sends ran and not one of them found an attached sender, so "+
			"the write and the read were never concurrent", calls.Load())
	}
	t.Logf("%d sends, %d of them through an attached sender, over %d attaches",
		calls.Load(), delivered.Load(), rounds)
}

// countingSender is a sender that only counts, so attaching one has no effect a
// test could mistake for the thing being measured.
type countingSender struct{ seen *atomic.Int64 }

func (c countingSender) Send(tea.Msg) { c.seen.Add(1) }

func newCountingSender(seen *atomic.Int64) sender { return countingSender{seen: seen} }
