package main

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/i18n"
)

// # A redraw against the goroutine stepping the match
//
// This client is the first thing in the repository to draw a battle another
// goroutine is stepping, and the two sides of that are a lock apart:
// socket.Mirror.Read hands a reading out under the mirror's read lock and its
// own doc says the reading is valid only for the duration of the call, while
// Client.Play takes the write lock every time a wire.Turn arrives and replays it
// into the same battle.
//
// ⚠️ **The overlap is FORCED here rather than waited for**, and that is the whole
// point of the test. The unsynchronised access is there by construction, so
// whether the detector sees it is only a question of whether the two goroutines
// happen to be inside it at the same moment — which under a full suite is a
// coin flip and alone is close to never. So one goroutine redraws in a tight
// loop for as long as the match is applying turns, and the vacuity net below
// counts both halves: a run where nothing was drawn, or where no turn was
// applied while it was being drawn, measures nothing and fails rather than
// passing quietly.
//
// It needs `-race` to say anything at all. Without the detector the reads and
// the writes both succeed and the test only proves the client can be drawn from
// a second goroutine at all, which is why `cmd/hexarena-tui` is in `make
// check`'s `-race` line.

// The floor under the overlap: how many redraws and how many applied turns a run
// has to have managed before its silence means anything.
//
// ⚠️ **They are a vacuity floor and not a target.** The window is the whole
// battle, so a run clears both by an order of magnitude — measured, a 3v3 bo1
// draws about two hundred times against about ninety applied turns — and the
// numbers are here so that a run which drew nothing, or drew against a battle
// nobody was stepping, fails instead of passing quietly.
const (
	redrawsWanted = 20
	turnsWanted   = 3
)

// TestARedrawDoesNotRaceTheGoroutineSteppingTheMatch draws a live battle from a
// second goroutine while the match steps it.
//
// *Sees:* every unsynchronised read of the mirror's battle on the drawing path
// — the board, the roster, the order line, the ending and the turn in front.
// *Cannot see:* anything at all without `-race`, and nothing about a battle this
// screen owns itself, which is single-goroutine by construction.
func TestARedrawDoesNotRaceTheGoroutineSteppingTheMatch(t *testing.T) {
	held, library := openARoom(t, 1)
	m, fake := joining(t, held, library, i18n.Vi)

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the dial was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	if m.screen != screenWaiting {
		t.Fatalf("a joined room landed on screen %v, want the waiting screen", m.screen)
	}

	_, opponentFailed := theOpponent(t, held)

	// The renderer's own copy of the model. A model is a value, so this is
	// exactly what bubbletea draws from — and the battle in it is a pointer, so
	// the copy shares the mirror's battle with the goroutine stepping it. That
	// sharing is the bug, and copying the model is how the test gets hold of it
	// without reaching into anything private.
	var renderer model
	var drawn atomic.Int64
	var body atomic.Pointer[string]
	stop := make(chan struct{})
	drawing := make(chan struct{})
	started := false
	comparedAtStart := 0

	deadline := time.Now().Add(theWholeMatch)
	for m.screen != screenResult && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
		if !started && m.screen == screenBattle && m.battle.Live && m.battle.Fight != nil {
			started = true
			comparedAtStart = comparedBy(m)
			renderer = m
			go func() {
				defer close(drawing)
				for {
					select {
					case <-stop:
						return
					default:
					}
					// The shipped drawing path, from the same entry bubbletea
					// calls: model.parts, into draw.PlayScreen.View.
					text, _ := renderer.parts()
					body.Store(&text)
					drawn.Add(1)
				}
			}()
		}
		// The driver from the whole-match test, and it holds nothing here either:
		// two presses answer a turn's two questions, and the point is to keep the
		// room sending turns while the goroutine above draws them.
		for range 2 {
			if m.screen != screenBattle || !m.battle.Live ||
				m.battle.Pending == nil || m.battle.Answered {
				break
			}
			m = key(t, m, "enter")
		}
	}

	if !started {
		t.Fatalf("no battle ever reached the screen inside %s, so nothing was drawn against "+
			"a stepping match; the client is on screen %v", theWholeMatch, m.screen)
	}
	close(stop)
	<-drawing

	// The vacuity net. Each of these three is a way for this test to pass while
	// measuring nothing at all.
	redraws := drawn.Load()
	if redraws < redrawsWanted {
		t.Fatalf("the second goroutine drew %d times, under the %d this run needs before "+
			"its silence says anything", redraws, redrawsWanted)
	}
	if turns := comparedBy(m) - comparedAtStart; turns < turnsWanted {
		t.Fatalf("%d turns were applied during %d redraws, under the %d this run needs: "+
			"nothing was writing to the battle while it was being read",
			turns, redraws, turnsWanted)
	}
	last := body.Load()
	if last == nil || *last == "" {
		t.Fatal("every redraw came back empty, so the drawing path was never reached")
	}
	// And the drawing really did read the battle rather than an error line or a
	// waiting notice: a unit tag is drawn by the board and the roster, which are
	// two of the reads this test is about.
	tag := ""
	for _, drawnTag := range renderer.battle.Tags {
		tag = drawnTag
		break
	}
	if tag == "" {
		t.Fatal("the battle on screen has no unit tags, so the board and the roster drew " +
			"no unit")
	}
	if !strings.Contains(*last, tag) {
		t.Fatalf("the redrawn body names no unit (looked for the tag %q), so it drew "+
			"something other than the board:\n%s", tag, *last)
	}
	t.Logf("%d redraws against %d applied turns", redraws, comparedBy(m)-comparedAtStart)

	if m.screen != screenResult {
		t.Fatalf("the match did not reach the result inside %s; the client is on screen %v",
			theWholeMatch, m.screen)
	}
	if err := <-opponentFailed; err != nil {
		t.Fatalf("the opponent's loop: %v", err)
	}
}
