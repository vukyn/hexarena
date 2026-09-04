package main

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The client's clock, measured
//
// Two features share one, and each half is measured where it can fail. The
// arithmetic is a pure function of a reading and two moments, so it needs no
// wall clock at all; the chooser's third arm is a timer, so one test really does
// wait for it — over a **one-second** allowance, which is what
// openARoomAllowing is for.
//
// ⚠️ **The bounds on that test are two-sided on purpose.** An arm that never
// fires and an arm that fires at once both leave a chooser that "returned a
// pass", and only the lower bound tells them apart. → the vacuity guard below,
// which is the other half: a third arm that swallowed the first two would answer
// every one of these and be a client that ignored its player.

// TestTheCountdownCountsTheSeatOnTurn is the whole of the arithmetic.
//
// ⚠️ **Which seat is which is DERIVED rather than assumed.** The battle decides
// who moves first, so the fixture reads the side off the unit the open prompt
// names and then plays this client's seat from **both** ends of the wire: once
// as the side being asked, once as the other. A test that only ever looked at
// its own turn would pass on a client that drew one clock twice, which is the
// defect this feature is most likely to have.
//
// *Sees:* the two counts swapped, the idle seat counted down, a countdown drawn
// with no allowance, and the rounding.
// *Cannot see:* that the moment handed in is the moment a turn really opened —
// that is the stamp, and TestTheCountdownReachesTheScreenOverASocket is what
// drives it.
func TestTheCountdownCountsTheSeatOnTurn(t *testing.T) {
	held, _ := openARoom(t, 1)
	fight, prompt := aBattleNobodyDrives(t, held)
	unit, known := fight.Unit(prompt.Unit)
	if !known {
		t.Fatalf("the open turn names %q, which is on no side of the board", prompt.Unit)
	}
	const allowance = 90
	opened := time.Now()

	// The seat being asked is this client's, and then the other one's.
	asked := socket.Sight{
		Fight: fight, Side: unit.Side, Welcome: wire.Welcome{Allowance: allowance},
	}
	watching := asked
	watching.Side = otherSide(unit.Side)

	clock := clockOf(asked, opened, opened.Add(18*time.Second))
	if clock.Waiting != draw.PlayClockYou {
		t.Errorf("the seat this client plays is being asked and the clock waits on %v",
			clock.Waiting)
	}
	if clock.Yours != 72 || clock.Theirs != allowance {
		t.Errorf("18 seconds into a %d-second allowance the clocks read %d and %d, want "+
			"72 and %d — only the seat on turn counts down",
			allowance, clock.Yours, clock.Theirs, allowance)
	}

	clock = clockOf(watching, opened, opened.Add(18*time.Second))
	if clock.Waiting != draw.PlayClockThem {
		t.Errorf("the other seat is being asked and the clock waits on %v", clock.Waiting)
	}
	if clock.Yours != allowance || clock.Theirs != 72 {
		t.Errorf("watching the other player think, the clocks read %d and %d, want %d "+
			"and 72", clock.Yours, clock.Theirs, allowance)
	}

	// A room that runs no clock of its own is not one a client counts down for:
	// it arms no timer, so a countdown here would be the client inventing a
	// deadline nobody is enforcing.
	none := asked
	none.Welcome.Allowance = 0
	if clock := clockOf(none, opened, opened.Add(time.Second)); clock.Waiting != draw.PlayClockNobody {
		t.Errorf("a room with no allowance draws a clock waiting on %v", clock.Waiting)
	}
	// And a turn nobody was ever seen opening. The zero moment is what a session
	// holds before its first message, and a count from it would be a countdown
	// against 1 January year one.
	if clock := clockOf(asked, time.Time{}, opened); clock.Waiting != draw.PlayClockNobody {
		t.Errorf("a turn this client never saw open draws a clock waiting on %v", clock.Waiting)
	}
}

// TestACountdownRoundsUpAndStopsAtNought is the seconds arithmetic on its own.
//
// Rounded up, so a turn that has just opened shows the whole allowance rather
// than one second less than it, and every number is on the screen for a whole
// second rather than the first one being a flicker.
func TestACountdownRoundsUpAndStopsAtNought(t *testing.T) {
	cases := []struct {
		allowance int
		since     time.Duration
		want      int
	}{
		{90, 0, 90},
		{90, time.Millisecond, 90},
		{90, 999 * time.Millisecond, 90},
		{90, time.Second, 89},
		{90, 89*time.Second + 500*time.Millisecond, 1},
		{90, 90 * time.Second, 0},
		{90, 5 * time.Minute, 0},
		{1, 0, 1},
		{0, 0, 0},
	}
	for _, one := range cases {
		if got := remaining(one.allowance, one.since); got != one.want {
			t.Errorf("%s into a %d-second allowance leaves %d seconds, want %d",
				one.since, one.allowance, got, one.want)
		}
	}
}

// TestThePatienceIsTheAllowanceAndAGrace pins the two numbers the third arm is
// built out of.
//
// ⚠️ **Against literals rather than against the constants themselves**, which is
// the rule this repository keeps for a design number: reading chooserGrace and
// adding it to the allowance would agree with any grace at all, including
// nought, and nought is the value that turns this client into the *first* to
// give up rather than the second.
func TestThePatienceIsTheAllowanceAndAGrace(t *testing.T) {
	cases := map[int]time.Duration{
		90: 92 * time.Second,
		1:  3 * time.Second,
		0:  0,
		-1: 0,
	}
	for allowance, want := range cases {
		if got := patienceFor(allowance); got != want {
			t.Errorf("a %d-second allowance is waited out in %s, want %s", allowance, got, want)
		}
	}
	// The grace is what makes this client second, so it has to be real and it
	// has to be small against the match: the whole point of a grace of two
	// seconds is that a player waiting on a peer that has died waits the
	// allowance and a moment, not the allowance twice.
	if chooserGrace <= 0 {
		t.Errorf("the grace is %s, so this client races the room for who passes first",
			chooserGrace)
	}
	if chooserGrace >= socket.Allowance(room2minimum) {
		t.Errorf("the grace is %s, which is not small against an allowance", chooserGrace)
	}
}

// room2minimum is the shortest allowance a room is worth running, in seconds,
// and it is here only to give the grace something to be small against.
const room2minimum = 10

// TestTheThirdArmPassesWhenTheAllowanceHasRunOut is the residual the lobby left
// open, driven.
//
// A real room, a real socket, this client seated on it — and **no peer at all**,
// which is the shape a peer that dies mid-prompt leaves behind: nothing will
// arrive, and Play is inside Decide rather than inside conn.read, so neither the
// read failing nor the keepalive giving up can reach the chooser.
//
// ⚠️ **Both bounds are asserted.** Under the upper one the arm exists at all;
// under the lower one it waits the allowance out rather than firing at once —
// and an arm that fired at once would pass every turn of every match while
// looking, from here, exactly like this test passing.
//
// *Sees:* the arm deleted (the wait times out), the arm fired early, the arm
// answering something other than a pass.
// *Cannot see:* the room's own timeout racing it, because there is no battle
// here to time out. → internal/socket's own timeout tests for that half.
func TestTheThirdArmPassesWhenTheAllowanceHasRunOut(t *testing.T) {
	const allowance = 1
	held, _ := openARoomAllowing(t, 1, allowance)
	sess := aSeatedSession(t, held)

	type answer struct {
		choice battle.Choice
		acted  bool
		took   time.Duration
	}
	answered := make(chan answer, 1)
	go func() {
		began := time.Now()
		choice, acted := sess.choose(theTurnBeingAsked())
		answered <- answer{choice: choice, acted: acted, took: time.Since(began)}
	}()

	waited := time.Second + chooserGrace
	select {
	case got := <-answered:
		if got.acted {
			t.Errorf("the chooser acted on %q with nobody to answer it; a client that "+
				"invents a decision is worse than one that waits", got.choice.Skill)
		}
		if got.choice != (battle.Choice{}) {
			t.Errorf("the chooser passed and spent %+v doing it", got.choice)
		}
		if got.took < waited {
			t.Errorf("the chooser gave up after %s, before the room's own %s allowance "+
				"and grace were out — this client has to be the second to give up, not "+
				"the first", got.took, waited)
		}
		t.Logf("the chooser passed after %s, on a %ds allowance plus %s of grace",
			got.took, allowance, chooserGrace)
	case <-time.After(waited + 10*time.Second):
		t.Fatalf("the chooser was still blocked %s after the allowance ran out, so a peer "+
			"that dies while this client is being asked still strands it", waited)
	}
}

// TestTheThirdArmDoesNotSwallowTheOtherTwo is the vacuity guard on the test
// above, and it is the half a green run of that one cannot give.
//
// ⚠️ **A chooser that always passed after three seconds would satisfy every
// assertion up there.** So both of the arms that were already here are driven
// with the timer armed behind them: the player's answer comes back as the
// player's answer, and a cancelled match comes back **promptly** rather than at
// the end of an allowance nobody is waiting for.
//
// *Sees:* arm one dropped, arm two dropped, and either of them answered by the
// timer instead — the promptness bound is what says which arm replied.
func TestTheThirdArmDoesNotSwallowTheOtherTwo(t *testing.T) {
	const allowance = 1
	waited := time.Second + chooserGrace

	t.Run("the player answers", func(t *testing.T) {
		held, _ := openARoomAllowing(t, 1, allowance)
		sess := aSeatedSession(t, held)
		// The answer is sent from the sender itself, on the "it is your turn"
		// message the chooser raises before it reaches its select. That is the
		// window the one-slot channel exists for, and answering inside it is
		// what a player pressing a key at that moment does.
		taken := draw.PlayAnswer{Choice: battle.Choice{Skill: "strike"}, Acted: true}
		sess.attach(senderThatAnswers(sess, taken))

		choice, acted, took := chosenBy(t, sess, waited)
		if !acted || choice.Skill != taken.Choice.Skill {
			t.Errorf("the chooser came back with %q/%v, want the answer the player gave "+
				"(%q) — the timer answered a turn the player had", choice.Skill, acted,
				taken.Choice.Skill)
		}
		if took >= waited {
			t.Errorf("the player's answer took %s to come back, which is the whole "+
				"allowance: it was the timer that replied, not the player", took)
		}
	})

	t.Run("the match is left", func(t *testing.T) {
		held, _ := openARoomAllowing(t, 1, allowance)
		sess := aSeatedSession(t, held)
		sess.attach(senderThatLeaves(sess))

		choice, acted, took := chosenBy(t, sess, waited)
		if acted || choice != (battle.Choice{}) {
			t.Errorf("leaving a match answered %q/%v, want a pass that spends nothing",
				choice.Skill, acted)
		}
		if took >= waited {
			t.Errorf("a left match unblocked the chooser after %s, which is the whole "+
				"allowance: the cancel did not reach it and the timer did", took)
		}
	})
}

// TestTheCountdownReachesTheScreenOverASocket is the chain the two tests above
// each hold one link of: a real match, a real mirror, the stamp taken on the
// Play goroutine, and a clock drawn on the battle screen.
//
// ⚠️ **It reads the drawn body rather than the field**, because a countdown the
// client computed and did not draw is the same to a player as no countdown at
// all — and the field alone would pass on a screen that dropped it.
//
// *Sees:* liveOf dropping the clock, the stamp never being taken (a turn nobody
// saw open draws nothing), and the wiring into Attach.
// *Cannot see:* the number being right to the second, which is the arithmetic
// test above, and which nothing over a real socket can bound anyway.
func TestTheCountdownReachesTheScreenOverASocket(t *testing.T) {
	held, library := openARoom(t, 1)
	m, fake := joining(t, held, library, i18n.Vi)
	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	m = send(t, m, command())
	theOpponent(t, held)

	deadline := time.Now().Add(theWholeMatch)
	for m.screen != screenBattle && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
	}
	if m.screen != screenBattle {
		t.Fatalf("the board was never drawn inside %s; the client is on screen %v",
			theWholeMatch, m.screen)
	}
	clock := m.battle.Clock
	if clock.Waiting == draw.PlayClockNobody {
		t.Fatalf("a turn is open on the board and the screen was handed no clock: %+v", clock)
	}
	// One of the two is counting down and the other is holding the whole
	// allowance, whichever way round the first turn fell.
	whole, running := held.config.Allowance, clock.Theirs
	if clock.Waiting == draw.PlayClockYou {
		running = clock.Yours
	}
	if running <= 0 || running > whole {
		t.Errorf("the seat on turn has %d seconds of a %d-second allowance left",
			running, whole)
	}
	if clock.Yours != whole && clock.Theirs != whole {
		t.Errorf("neither clock holds the whole %d-second allowance (%d and %d), so both "+
			"seats are being counted down at once", whole, clock.Yours, clock.Theirs)
	}
	body := drawnBody(m)
	drawn := m.text(i18n.PlayClockYours, "", "")
	if clock.Waiting == draw.PlayClockThem {
		drawn = m.text(i18n.PlayClockTheirs, "", "")
	}
	if lead, _, _ := strings.Cut(drawn, " "); !strings.Contains(body, lead) {
		t.Errorf("the board draws no countdown at all:\n%s", body)
	}
	t.Logf("the board drew %v with %d and %d seconds", clock.Waiting, clock.Yours, clock.Theirs)
}

// TestTheCountdownMovesOnATickAndTheTickReArms is the half of the feature that
// is not the arithmetic: a number that only moved when a message arrived would
// be still for exactly the wait it exists to fill, because during the other
// player's turn nothing arrives by definition.
//
// ⚠️ **This one really does wait a second, and the seam was left alone on
// purpose.** The moment could have been injected into the model, but the only
// way to do that is to hand the client a second answer to what time it is —
// which is what the whole feature is arranged to avoid. One real second buys the
// claim honestly, and it is the only wall-clock wait in this file besides the
// third arm's own.
//
// *Sees:* the tick not re-armed (one move and then a still clock), the tick not
// redrawing, and the countdown recomputed from the wrong moment.
// *Cannot see:* that bubbletea really delivers it — tea.Tick is the library's,
// and what is driven here is the model's answer to one.
func TestTheCountdownMovesOnATickAndTheTickReArms(t *testing.T) {
	held, library := openARoomAllowing(t, 1, 30)
	m, fake := joining(t, held, library, i18n.Vi)
	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	// The join is what arms the first tick, rather than the first turn: a clock
	// that started when somebody had already waited is a clock that is late.
	joined, armed := m.Update(command())
	m = joined.(model)
	if armed == nil {
		t.Fatal("joining a room armed no tick, so nothing will redraw between messages")
	}
	theOpponent(t, held)

	// ⚠️ **Driven to THIS client's own turn, and that is what makes the second
	// that passes a measurement.** While the room is waiting on this seat
	// nothing else arrives, so the open turn cannot change under the sleep — on
	// any earlier turn the opponent is still answering, and a turn that changed
	// hands mid-sleep would open a fresh allowance and read as a clock that
	// never moved.
	deadline := time.Now().Add(theWholeMatch)
	for m.battle.Pending == nil && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
	}
	if m.screen != screenBattle || m.battle.Pending == nil {
		t.Fatalf("this client was never asked a turn inside %s; it is on screen %v",
			theWholeMatch, m.screen)
	}
	before := m.battle.Clock
	if before.Waiting != draw.PlayClockYou {
		t.Fatalf("this client holds the open turn and the clock waits on %v: %+v",
			before.Waiting, before)
	}
	time.Sleep(1100 * time.Millisecond)
	ticked, again := m.Update(clockTickMsg{})
	m = ticked.(model)
	if again == nil {
		t.Fatal("a tick asked for no next tick, so the countdown moves exactly once")
	}
	after := m.battle.Clock
	if after.Yours >= before.Yours {
		t.Errorf("a second passed on this client's own turn and its clock went from %d "+
			"to %d: nothing is counting down between messages", before.Yours, after.Yours)
	}
	if after.Theirs != before.Theirs {
		t.Errorf("the seat that is not being asked went from %d to %d; only the seat on "+
			"turn counts down", before.Theirs, after.Theirs)
	}
	t.Logf("the clocks went from %d/%d to %d/%d over a second",
		before.Yours, before.Theirs, after.Yours, after.Theirs)
}

// aSeatedSession is a session holding a real client on a real room, dialled the
// way the model dials one.
//
// ⚠️ **Nobody else joins, and that is the fixture.** A room with one seat filled
// starts no battle, so Play sits reading a socket nothing will arrive on — which
// is exactly the state a peer that has died leaves behind, and the state a
// chooser called here is blocked in.
func aSeatedSession(t *testing.T, held *aRoom) *session {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	sess := newSession()
	sess.attach(newFakeSender())
	// The real dial, so the real Stepped hook is the one wired up.
	message := sess.dial(held.code, wire.Hello{
		Version: version, Squad: held.squads[0], Name: "Lan",
	}, held.books)()
	joined, seated := message.(matchJoinedMsg)
	if !seated {
		t.Fatalf("the dial was turned away: %v", message)
	}
	sess.begin(joined.client)
	t.Cleanup(sess.leave)
	return sess
}

// chosenBy runs the chooser and hands back what it answered and how long it
// took, failing rather than hanging when nothing answers at all.
func chosenBy(t *testing.T, sess *session, waited time.Duration) (battle.Choice, bool, time.Duration) {
	t.Helper()
	type answer struct {
		choice battle.Choice
		acted  bool
		took   time.Duration
	}
	answered := make(chan answer, 1)
	go func() {
		began := time.Now()
		choice, acted := sess.choose(theTurnBeingAsked())
		answered <- answer{choice: choice, acted: acted, took: time.Since(began)}
	}()
	select {
	case got := <-answered:
		return got.choice, got.acted, got.took
	case <-time.After(waited + 10*time.Second):
		t.Fatalf("the chooser answered nothing at all inside %s", waited+10*time.Second)
		return battle.Choice{}, false, 0
	}
}

// senderThatAnswers is a sender that presses a key the moment the chooser says
// it is this player's turn.
func senderThatAnswers(sess *session, taken draw.PlayAnswer) sender {
	return senderFunc(func(message tea.Msg) {
		if _, asking := message.(matchAskingMsg); asking {
			sess.answer(taken, theTurnBeingAsked())
		}
	})
}

// theTurnBeingAsked is the prompt these chooser tests are about, and every one
// of them uses this one so that an answer and the chooser waiting for it agree
// on which turn is open. → session.pressed, which is why they have to.
//
// It is a fresh value per call rather than a shared pointer on purpose: what
// routes an answer is the (unit, turn) pair, and a test that happened to hand
// the same pointer to both halves could not tell that from identity.
func theTurnBeingAsked() *battle.Prompt { return &battle.Prompt{Unit: "A1", Turn: 1} }

// senderThatLeaves is a sender that leaves the match the moment the chooser says
// it is this player's turn, which is esc pressed mid-prompt.
func senderThatLeaves(sess *session) sender {
	return senderFunc(func(message tea.Msg) {
		if _, asking := message.(matchAskingMsg); asking {
			go sess.leave()
		}
	})
}

// senderFunc is a sender that is one function. → the head of match_test.go for
// why the sender is the one thing here that is faked.
type senderFunc func(tea.Msg)

func (f senderFunc) Send(message tea.Msg) { f(message) }

// aBattleNobodyDrives is a battle out of the shipped cast, stopped on its first
// open turn — the shape a mirror hands over, built here because the arithmetic
// under test needs a board and a prompt rather than a socket.
func aBattleNobodyDrives(t *testing.T, held *aRoom) (*battle.Battle, *battle.Prompt) {
	t.Helper()
	ours, err := held.squads[0].Take(hex.SideAlly, held.deps.Characters)
	if err != nil {
		t.Fatalf("field this client's side: %v", err)
	}
	theirs, err := held.squads[1].Take(hex.SideEnemy, held.deps.Characters)
	if err != nil {
		t.Fatalf("field the other side: %v", err)
	}
	fight, err := battle.New(held.books, held.config.Seed, append(ours, theirs...))
	if err != nil {
		t.Fatalf("open a battle: %v", err)
	}
	fight.Begin()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("open the first turn: %v", err)
	}
	for prompt != nil && prompt.Skipped {
		if err := fight.Pass(prompt.Reason); err != nil {
			t.Fatalf("skip a turn nobody may take: %v", err)
		}
		if prompt, err = fight.Advance(); err != nil {
			t.Fatalf("open the next turn: %v", err)
		}
	}
	if prompt == nil {
		t.Fatal("the battle opened with no turn for anybody")
	}
	return fight, prompt
}

// otherSide is the half of the board this client is not on.
func otherSide(side hex.Side) hex.Side {
	if side == hex.SideAlly {
		return hex.SideEnemy
	}
	return hex.SideAlly
}
