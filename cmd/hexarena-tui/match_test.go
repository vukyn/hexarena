package main

import (
	"context"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/room"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # A match, end to end, over a real socket
//
// ⚠️ **The transport is deliberately NOT faked.** A faked *socket.Client would
// test the fake: the four things Play carries — divergence detection on the turn
// it happens, the unknown-message refusal, the keepalive-gave-up versus
// caller-cancelled distinction, and Mirror.Over — are precisely what a fake
// would not have, and they are the whole reason Play is called rather than
// reimplemented.
//
// **What IS faked is the sender**, because it has to be: the program cannot be
// built until the model is, the model cannot be built until the session is, and
// a headless test has no program at all. What a fakeSender cannot see is that a
// real *tea.Program was ever attached — that is held by
// `var _ sender = (*tea.Program)(nil)` in session.go and by the single
// sess.attach(program) line in run.

// fakeSender collects what a match sends the model, from whichever goroutine
// sends it.
//
// Unbounded rather than a buffered channel, and that is not laziness: a channel
// that dropped on a full buffer would turn "the client lost a redraw" into a
// test that still passed, which is exactly the class of silence this file is
// about.
type fakeSender struct {
	mu   sync.Mutex
	sent []tea.Msg
	wake chan struct{}
}

func newFakeSender() *fakeSender { return &fakeSender{wake: make(chan struct{}, 1)} }

func (f *fakeSender) Send(message tea.Msg) {
	f.mu.Lock()
	f.sent = append(f.sent, message)
	f.mu.Unlock()
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

// take is everything sent since the last call, drained.
func (f *fakeSender) take() []tea.Msg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

// awaits blocks until something is sent, and reports false on a timeout so a
// caller fails rather than hanging the suite.
func (f *fakeSender) awaits(within time.Duration) bool {
	select {
	case <-f.wake:
		return true
	case <-time.After(within):
		return false
	}
}

// theWholeMatch is how long this file waits for a bo3 over a loopback socket
// before calling it hung. A bo3 of a 3v3 is a hundred-odd decisions and runs in
// well under a second in process; the margin is for a loaded machine, and the
// point of the bound is that a client that stops making progress fails the suite
// rather than hanging it.
//
// ⚠️ **The bound has earned itself once, and the number is not what mattered.**
// It failed at 61.22s inside a loaded suite against work it does alone in 0.9s,
// and the temptation was to read a sixty-fold margin as too tight and widen it.
// It was not slowness: the client had gone silent on the battle screen with a
// prompt open and already answered, waiting out a whole allowance because the
// chooser had thrown that answer away.
// → TestAnAnswerPressedBeforeTheChooserAsksIsTakenRatherThanDropped, which holds
// the same thing deterministically and in a millisecond.
const theWholeMatch = 60 * time.Second

// aRoom is a registry, a server and a loopback listener with one room open on
// it, plus everything a peer needs to join.
type aRoom struct {
	code    wire.RoomCode
	books   battle.Books
	deps    room.Deps
	config  room.Config
	squads  []placement.Squad
	library *forge.Library
}

// openARoom starts a real room behind a real listener.
//
// ⚠️ **Both sides run on the EMBEDDED books**, which is not a convenience: the
// client builds its mirror from seed.Books() whatever --data says, because the
// digest at the gate is over the embedded files and anything else would make
// that promise a lie. So the room is handed the same books, and the squads are
// built out of the shipped cast rather than out of the fixture's injected one —
// a squad naming a character only the scratch directory has would be refused at
// the gate, correctly, and this test would be measuring that instead.
func openARoom(t *testing.T, battles int) (*aRoom, *forge.Library) {
	t.Helper()
	return openARoomAllowing(t, battles, room.DefaultAllowance)
}

// openARoomAllowing is the same room with the per-prompt allowance named, which
// is what the countdown and the chooser's third arm are both measured against: a
// test that waited out the default ninety seconds would not be run.
func openARoomAllowing(t *testing.T, battles, allowance int) (*aRoom, *forge.Library) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	dir := scratchData(t)
	library, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the embedded books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	// Two sides out of the shipped cast, around **different** characters: two
	// identical squads make the halves of a battle interchangeable, so nothing
	// could see a client drawing the wrong one.
	ours := aShippedSide(t, characters, "phe-ta", "pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	theirs := aShippedSide(t, characters, "phe-ho", "pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa")
	if err := library.SaveSquad(ours); err != nil {
		t.Fatalf("save the side this client brings: %v", err)
	}

	rooms := room.NewRegistry()
	server := socket.NewServer(rooms, socket.Options{})
	listening := httptest.NewServer(server)
	at, err := netip.ParseAddrPort(listening.Listener.Addr().String())
	if err != nil {
		t.Fatalf("read the listener's address: %v", err)
	}
	at = netip.AddrPortFrom(at.Addr().Unmap(), at.Port())
	config := room.Config{
		Format: wire.Format3v3, Battles: battles,
		Allowance: allowance, Seed: 11, TurnCap: room.DefaultTurnCap,
	}
	deps := room.Deps{Books: books, Characters: characters, Version: version}
	code, err := rooms.Open(at, config, deps)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	t.Cleanup(func() {
		listening.Close()
		rooms.CloseAll()
		rooms.Wait()
	})
	return &aRoom{
		code: code, books: books, deps: deps, config: config,
		squads: []placement.Squad{ours, theirs}, library: library,
	}, library
}

// aShippedSide is a legal 3v3 squad out of the shipped cast, every part of
// "legal" built here rather than borrowed: the shipped squads are two units and
// would move with the balance.
func aShippedSide(t *testing.T, characters *cast.Book, id string, wanted ...string) placement.Squad {
	t.Helper()
	slots := []hex.Offset{{Col: 0, Row: 1}, {Col: 1, Row: 1}, {Col: 2, Row: 1}}
	squad := placement.Squad{ID: id, Name: id}
	for index, name := range wanted {
		character, known := characters.Get(name)
		if !known {
			t.Fatalf("no shipped character is called %q", name)
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
			Skills:    upToSlots(character.SkillsAt(progression.LevelCap, stage), cast.SkillSlots),
			Passives:  upToSlots(character.PassivesAt(progression.LevelCap, stage), cast.TraitSlots),
		})
	}
	return squad
}

func upToSlots(available []string, slots int) []string {
	if len(available) > slots {
		return available[:slots:slots]
	}
	return available[:len(available):len(available)]
}

// joining is this client, wired to a fake sender, standing on the join screen
// with the room's code typed in.
func joining(t *testing.T, held *aRoom, library *forge.Library, lang i18n.Lang) (model, *fakeSender) {
	t.Helper()
	fake := newFakeSender()
	sess := newSession()
	sess.attach(fake)
	m := newModel(library, lang, sess)
	m.width, m.height = 120, 44
	m = m.enter(screenJoin)
	if len(m.join.Squads) == 0 {
		t.Fatal("the join screen found no side to bring")
	}
	m = typeText(t, m, string(held.code))
	return m, fake
}

// theOpponent is a plain socket.Client on the other seat, answering off its own
// mirror. It is what makes the match a match: this client cannot play both
// halves, and a room with one seat filled never starts.
func theOpponent(t *testing.T, held *aRoom) (*socket.Client, chan error) {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	client, err := socket.Dial(context.Background(), held.code, wire.Hello{
		Version: version, Squad: held.squads[1], Name: "Nam",
	}, held.books, socket.ClientOptions{})
	if err != nil {
		t.Fatalf("the opponent could not join: %v", err)
	}
	failed := make(chan error, 1)
	go func() {
		failed <- client.Play(context.Background(), func(prompt *battle.Prompt) (battle.Choice, bool) {
			fight := client.Mirror().Battle()
			if fight == nil {
				return battle.Choice{}, false
			}
			return fight.Suggest(prompt)
		})
	}()
	t.Cleanup(client.Close)
	return client, failed
}

// TestAJoinedMatchPlaysToItsEndOverALoopbackListener is the whole vertical.
//
// A real registry, a real server, a real listener; this client's session takes
// one seat and a plain socket.Client takes the other; the model is driven with
// the messages a match sends and the keystrokes a player presses, until it lands
// on the result.
//
// *Sees:* the chooser, the one-slot channel, the sender, Attach, the live
// footers, the routing, and the standing — over a real socket.
// *Cannot see:* that a real *tea.Program delivered the messages. → the note at
// the head of this file.
func TestAJoinedMatchPlaysToItsEndOverALoopbackListener(t *testing.T) {
	held, library := openARoom(t, 3)
	m, fake := joining(t, held, library, i18n.Vi)

	// The dial, through the key a player presses.
	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	if !m.join.Dialling {
		t.Fatal("the join screen is not calling anything")
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the dial was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	if m.screen != screenWaiting {
		t.Fatalf("a joined room landed on screen %v, want the waiting screen", m.screen)
	}
	if !m.waiting.Seat.Valid() {
		t.Fatalf("the waiting screen holds the seat %q, which is not one of the room's two",
			m.waiting.Seat)
	}
	if !m.waiting.Seated || m.waiting.Welcome.Battles != 3 {
		t.Fatalf("the welcome did not reach the waiting screen: %+v", m.waiting)
	}
	if !strings.Contains(drawnBody(m), string(held.code)) {
		t.Fatalf("the waiting screen does not name the room it joined:\n%s", drawnBody(m))
	}

	_, opponentFailed := theOpponent(t, held)

	// The driving loop: take whatever the match sent, feed it through the real
	// Update, and answer when the screen says the turn is this player's.
	answers := 0
	deadline := time.Now().Add(theWholeMatch)
	for m.screen != screenResult && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
		// A turn asks up to two questions — which skill, and then where — so
		// two presses at most, and the second bound is what stops a screen that
		// never answers turning this into a spin.
		for range 2 {
			if m.screen != screenBattle || !m.battle.Live ||
				m.battle.Pending == nil || m.battle.Answered {
				break
			}
			m = key(t, m, "enter")
		}
		if m.battle.Answered {
			answers++
		}
	}
	if m.screen != screenResult {
		t.Fatalf("the match did not reach the result inside %s; the client is on screen %v",
			theWholeMatch, m.screen)
	}
	if err := <-opponentFailed; err != nil {
		t.Fatalf("the opponent's loop: %v", err)
	}
	if answers == 0 {
		t.Fatal("this client answered no turn at all, so the chooser and the channel were " +
			"never exercised")
	}

	// What the mirror settled, read the way the client reads it.
	var fought []socket.Fought
	m.session.read(func(sight socket.Sight) { fought = append(fought, sight.Fought...) })
	if len(fought) < 2 {
		t.Fatalf("the match settled %d battles, so nothing below can say the side swapped",
			len(fought))
	}
	// ⚠️ **The side SWAPPED between battles, which is the assertion a bo1
	// wearing a bo3's name would otherwise satisfy.** A match is fought both
	// ways round — that is what makes a series measure the squads rather than
	// the slot — so two battles on one side is a defect and not a coincidence.
	if fought[0].Side == fought[1].Side {
		t.Errorf("battles 1 and 2 were both fought on the %s side, so the match did not "+
			"change ends and this is a bo1 played twice", fought[0].Side)
	}
	if fought[0].Seed == fought[1].Seed {
		t.Errorf("battles 1 and 2 ran from the same seed %d", fought[0].Seed)
	}
	if m.result.Err != nil {
		t.Errorf("the match ended with %v", m.result.Err)
	}

	// The per-turn digest really was checked, which is the whole reason a client
	// runs the engine at all.
	if checked := comparedBy(m); checked == 0 {
		t.Error("this client compared no digests, so the mirror never checked a turn " +
			"against the room's")
	} else {
		t.Logf("%d digests compared over %d battles, %d turns answered here",
			checked, len(fought), answers)
	}

	// And the standing drawn is the standing the mirror settled.
	mine, theirs := m.result.standing()
	wantMine, wantTheirs := 0, 0
	for _, one := range fought {
		switch {
		case !one.Decided:
		case one.Mine():
			wantMine++
		default:
			wantTheirs++
		}
	}
	if mine != wantMine || theirs != wantTheirs {
		t.Errorf("the result screen draws %d-%d over the mirror's %d-%d",
			mine, theirs, wantMine, wantTheirs)
	}
	if !strings.Contains(drawnBody(m), m.text(i18n.ResultStanding, mine, theirs)) {
		t.Errorf("the result screen does not draw the standing it holds:\n%s", drawnBody(m))
	}
}

// TestTheMenuWillNotOpenABattleOverALiveMatch is one of the two risks of drawing
// a live battle and a hot-seat one on **one** model field.
//
// ⚠️ A player cannot be in both at once, so one draw.PlayScreen serves both —
// but the menu still offers a battle, and opening one from there while a match
// is running would build a second engine over the top of the mirror's: the
// reader would be playing themselves while a person on another machine waited
// for a turn.
//
// *Sees:* the guard removed, as a hot-seat battle replacing the live one.
// *Cannot see:* the opponent noticing, which is a whole match away.
func TestTheMenuWillNotOpenABattleOverALiveMatch(t *testing.T) {
	held, library := openARoom(t, 1)
	m, fake := joining(t, held, library, i18n.Vi)

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command")
	}
	m = send(t, m, command())
	theOpponent(t, held)

	// Wait until the board is actually drawn, which is when the guard matters.
	deadline := time.Now().Add(theWholeMatch)
	for m.screen != screenBattle && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
	}
	if m.screen != screenBattle || !m.battle.Live || m.battle.Fight == nil {
		t.Fatalf("no live battle was reached inside %s (screen %v), so the guard is "+
			"pressed against nothing", theWholeMatch, m.screen)
	}
	engine := m.battle.Fight

	for _, target := range []screen{screenBattle, screenJoin} {
		after := m.enterUnlessInAMatch(target)
		if !after.battle.Live {
			t.Errorf("entering %v from the menu while a match is live turned the battle "+
				"local, so a hot-seat game was opened over the mirror's", target)
		}
		if after.battle.Fight != engine {
			t.Errorf("entering %v from the menu while a match is live replaced the engine",
				target)
		}
		if after.screen != screenBattle {
			t.Errorf("entering %v from the menu while a match is live landed on screen %v, "+
				"want the match the reader was trying to get back to", target, after.screen)
		}
	}
	// And the same entry with no match live really does open one, which is what
	// stops the guard from being a refusal that never lifts.
	m.session.leave()
	local := m.leaveMatch().enterUnlessInAMatch(screenBattle)
	if local.battle.Live {
		t.Error("the battle entry stayed refused after the match was left")
	}
	if local.battle.Fight == nil {
		t.Errorf("the battle entry opened nothing after the match was left: %v", local.battle.Err)
	}
}

// comparedBy is how many per-turn digests this client's mirror checked.
func comparedBy(m model) int {
	compared := 0
	m.session.mu.Lock()
	client := m.session.client
	m.session.mu.Unlock()
	if client != nil {
		compared = client.Mirror().Compared()
	}
	return compared
}

// TestQuittingMidTurnUnblocksTheChooser is the deadlock this design was asked to
// make impossible, driven to the exact moment it would happen.
//
// The chooser is blocked — matchAskingMsg has been sent and nothing has answered
// — and the only thing that can unblock it is the context being cancelled.
// session.leave is one of the two callers of that cancel; the other is run's own
// defer, which is what makes the guarantee structural rather than remembered.
//
// *Sees:* the ctx.Done arm being removed from the chooser, as a hang.
// *Cannot see:* the peer-died-mid-turn residual, which needs a clock and is
// deferred with the countdown. → session.go.
func TestQuittingMidTurnUnblocksTheChooser(t *testing.T) {
	held, library := openARoom(t, 1)
	m, fake := joining(t, held, library, i18n.Vi)

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command")
	}
	m = send(t, m, command())
	if m.screen != screenWaiting {
		t.Fatalf("the dial did not get past the gate; the client is on screen %v", m.screen)
	}
	theOpponent(t, held)

	// Wait for the chooser to be **blocked**, which is what matchAskingMsg says
	// and nothing else does: the message is sent immediately before the select.
	asked := false
	deadline := time.Now().Add(theWholeMatch)
	for !asked && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
			if _, asking := message.(matchAskingMsg); asking {
				asked = true
			}
		}
	}
	if !asked {
		t.Fatalf("no turn was ever asked of this client inside %s, so the chooser was "+
			"never blocked and this measures nothing", theWholeMatch)
	}

	m.session.leave()
	select {
	case <-m.session.finished():
	case <-time.After(10 * time.Second):
		t.Fatal("the Play goroutine was still blocked ten seconds after the player left, " +
			"which is the deadlock this design exists to have closed")
	}
	if err, done := m.session.outcome(); !done {
		t.Error("the session reports no outcome after its loop returned")
	} else if err != nil {
		t.Logf("the loop returned %v, which a cancelled match may do", err)
	}
}

// TestAKeystrokeIsNeverBlockedByAChooserThatHasGone is the reverse deadlock.
//
// An Update that waited for a chooser to take its answer would hang the whole
// program whenever nobody was asking — between turns, after the match, and on
// every keystroke that is not a decision. A dropped keystroke is the right
// answer there, because there is no turn for it to be about.
//
// *Sees:* an unbuffered or blocking send from Update.
// *Cannot see:* whether the dropped keystroke *should* have been dropped —
// that is internal/screen's Answered assertion.
func TestAKeystrokeIsNeverBlockedByAChooserThatHasGone(t *testing.T) {
	sess := newSession()
	sess.attach(newFakeSender())
	sess.open()
	sess.leave()

	// Three, not one: the first fills the one slot the buffer has, the second
	// hits the default, and the third is the second again on a channel nobody
	// will ever drain.
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		for range 3 {
			sess.answer(draw.PlayAnswer{Choice: battle.Choice{Skill: "razor_leaf"}, Acted: true},
				theTurnBeingAsked())
		}
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("answering a chooser that has gone blocked, which hangs the whole program " +
			"on every keystroke that is not a decision")
	}
}

// TestAStaleAnswerIsNotSpentOnTheNextTurn is the first half of the check at the
// top of the chooser.
//
// ⚠️ **A keystroke can land in the slot with nobody asking**, which is what the
// one-slot buffer is for and also what makes it dangerous: without the check,
// the answer pressed for a turn that has already gone — the player answered, the
// server timed the seat out and passed for it — would be spent on the next one,
// with nobody looking at the board it was taken on.
//
// The two turns are what makes it a measurement: the answer is pressed for
// A1's turn 1 and the chooser is asked about A1's turn 2, so a chooser that
// spends it is spending an answer taken on a different board.
//
// *Sees:* the check being removed, as a chooser that returns immediately with an
// answer nobody meant for this turn.
// *Cannot see:* the server's half of that story; what it drives is the client
// state the server's timeout leaves behind.
func TestAStaleAnswerIsNotSpentOnTheNextTurn(t *testing.T) {
	fake := newFakeSender()
	sess := newSession()
	sess.attach(fake)
	sess.open()
	t.Cleanup(sess.leave)

	gone := &battle.Prompt{Unit: "A1", Turn: 1}
	open := &battle.Prompt{Unit: "A1", Turn: 2}

	stale := draw.PlayAnswer{Choice: battle.Choice{Skill: "razor_leaf"}, Acted: true}
	sess.answer(stale, gone)
	answers, _ := sess.turn()
	if len(answers) != 1 {
		// ⚠️ This is also where an **unbuffered** answers channel is caught, and
		// the failure is worth naming in full: with no slot, a keystroke pressed
		// in the window between the chooser sending matchAskingMsg and reaching
		// its select hits the default in session.answer and is **lost** — a real
		// decision, on a real turn, silently dropped.
		t.Fatalf("a keystroke pressed with no chooser at the select was dropped rather "+
			"than buffered (%d held of the one slot): a lost decision on the client, and "+
			"the check below is then asked to drop nothing", len(answers))
	}

	taken := make(chan draw.PlayAnswer, 1)
	go func() {
		choice, acted := sess.choose(open)
		taken <- draw.PlayAnswer{Choice: choice, Acted: acted}
	}()
	select {
	case got := <-taken:
		t.Fatalf("the next turn's chooser returned %+v straight away, which is the answer "+
			"buffered for a turn that had already gone", got)
	case <-time.After(250 * time.Millisecond):
		// Still blocked, which is the whole point: the stale answer was dropped.
	}

	// And it is not blocked for ever — a fresh answer is taken, which is what
	// stops this passing on a chooser that returns nothing at all.
	fresh := draw.PlayAnswer{Choice: battle.Choice{Skill: "vine_whip"}, Acted: true}
	sess.answer(fresh, open)
	select {
	case got := <-taken:
		if got != fresh {
			t.Errorf("the chooser took %+v, want the answer pressed for this turn %+v", got, fresh)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the chooser never took the answer pressed for its own turn")
	}

	// And it said it was asking, which is the message the screen enables input
	// off — sent before the select, which is the window the one slot absorbs.
	asked := 0
	for _, message := range fake.take() {
		if _, asking := message.(matchAskingMsg); asking {
			asked++
		}
	}
	if asked == 0 {
		t.Error("the chooser sent no matchAskingMsg, so nothing would tell the screen it " +
			"was this player's turn")
	}
}

// TestAnAnswerPressedBeforeTheChooserAsksIsTakenRatherThanDropped is the other
// half, and it is the bug TODO.md filed as "a LAN test that fails under a loaded
// suite and passes alone".
//
// ⚠️ **The window this is about was believed not to exist.** The chooser's
// drain was written on the premise that nothing could be in the slot for the
// turn now opening, because the chooser had not sent matchAskingMsg yet. But
// "it is your turn" is socket.Mirror.Asking, which is true as soon as the room's
// batch is taken in — a message and a redraw *earlier* than this call — so a
// player answering off the board already in front of them lands in the slot
// first. The drain then ate a real decision; PlayScreen had recorded the turn as
// Answered and would not offer it again; and the match sat still until the
// allowance ran out at the far end.
//
// It cost a minute per occurrence, which is why the loopback match test failed
// at 61.22s against a 60s bound and passed in 0.9s alone.
//
// *Sees:* the check reverting to a bare drain, as a chooser that never returns.
// *Cannot see:* which of the two ends notices first — that is the allowance, and
// it is internal/room's.
func TestAnAnswerPressedBeforeTheChooserAsksIsTakenRatherThanDropped(t *testing.T) {
	fake := newFakeSender()
	sess := newSession()
	sess.attach(fake)
	sess.open()
	t.Cleanup(sess.leave)

	open := &battle.Prompt{Unit: "A1", Turn: 2}
	pressed := draw.PlayAnswer{Choice: battle.Choice{Skill: "vine_whip"}, Acted: true}
	// The whole of the race, made deterministic: the keystroke is in the slot
	// **before** the chooser is called at all.
	sess.answer(pressed, open)

	taken := make(chan draw.PlayAnswer, 1)
	go func() {
		choice, acted := sess.choose(open)
		taken <- draw.PlayAnswer{Choice: choice, Acted: acted}
	}()
	select {
	case got := <-taken:
		if got != pressed {
			t.Errorf("the chooser took %+v, want the answer pressed for this very turn %+v",
				got, pressed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the chooser threw away an answer pressed for the turn it is asking about " +
			"and is now waiting for one nobody will press: the screen has already recorded " +
			"this turn as answered, so the match stands still until the allowance runs out")
	}
}

// TestThisBinaryKnowsWhatItIs is the one impure line of the build string, and it
// asserts only what is true under every way of running the suite: something is
// said, and it is not blank.
//
// ⚠️ It cannot assert a **value** — `go test` and a stamped release produce
// different ones by design, which is the whole reason wire.BuildOf is pure and
// has its own table over there. What is *here* is that this binary reads its own
// stamp and its own build info, and that the string it produces is fit to go on
// a screen beside a data digest: a blank there reads as a bug in the printing
// rather than as a fact about the binary.
//
// It matters more than the same test on the host does, because this string is
// what a player reads off two machines when a room refuses them for a protocol
// or a data mismatch — the two refusals whose wording says "read the build line
// on each".
func TestThisBinaryKnowsWhatItIs(t *testing.T) {
	said := buildString()
	if strings.TrimSpace(said) == "" {
		t.Error("this binary announces an empty build string, which is the one of the " +
			"three version numbers a person is meant to read")
	}
	version, err := wire.Local(said)
	if err != nil {
		t.Fatalf("this binary cannot announce itself at a room's gate: %v", err)
	}
	if version.Build != said {
		t.Errorf("the version this client announces carries the build %q, not %q",
			version.Build, said)
	}
	t.Logf("this binary announces itself as %q, data %s", said, version.Data.Digest.Short())
}
