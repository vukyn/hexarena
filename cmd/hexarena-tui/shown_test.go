package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/room"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The ten refusals and the three closures, read by a person
//
// ⚠️ **They were worded and unread**, which TODO.md records as one step narrower
// than "shipped dead" rather than closed: internal/i18n has held all thirteen
// sentences in both languages since #255, and nothing drew any of them. This is
// the far end of that, and it is the only test in the repository that can see
// the gap.
//
// ⚠️ **TestNoKeyIsOrphaned cannot see it and must not be quoted as if it
// could.** It counts an identifier named anywhere in the module, and
// internal/i18n/protocol.go's own lookup names all thirteen — so it passed for
// as long as the words had no reader at all. Nor can internal/i18n's four
// protocol tests: they hold the catalog complete, distinct and free of enum
// spellings, which says nothing about a screen.

// TestEveryRefusalIsShownAndEveryClosureIsShown walks both enums by **count**
// and puts each value on the screen it is reachable on.
//
// ⚠️ **Which screen a refusal belongs to is DERIVED TWICE, not declared**, and
// that is the half that makes this able to fail. A join screen will happily word
// any code handed to it and so will a live battle, so a table pairing codes with
// screens could be permuted freely and every drawing assertion would still pass.
// So both sets are **produced out of a real room.Room**: six refusals a gate can
// answer a *join* with, and three a room can answer a seated peer with. The two
// are then held **disjoint and total** against wire.CodeCount, and the walk has a
// default arm — so a code moved from one set to the other lands in neither and
// this goes red naming it.
//
// What it sees: a wording nobody draws, a new enum value with nowhere to appear,
// a code no room produces at all, and a code moved between the two sets.
// ⚠️ What it cannot see: the two *arms of this test* swapped over — both screens
// word whatever they are handed, and no assertion here can tell them apart once
// the sets themselves are right. What stands in for that is that neither screen
// chooses its own code: the join screen is fed by a *socket.Refusal out of Dial
// and the battle screen by socket.Mirror.Refusals, and neither producer is
// reachable from the other's path.
// It also cannot see whether the sentence is the *right* sentence. That is
// internal/i18n's own four tests, and past those a reader.
func TestEveryRefusalIsShownAndEveryClosureIsShown(t *testing.T) {
	gate := theGateAnswers(t)
	inMatch := theRoomRefusesInAMatch(t)
	if len(gate) == 0 || len(inMatch) == 0 {
		t.Fatal("a real room produced no refusals, so every code below would be checked " +
			"on the same screen and this measures nothing")
	}
	// Disjoint and total, which is what makes the default arm below able to fire:
	// a refusal is answered either at the gate or in a match and never both, so a
	// code in the wrong set is a code in neither.
	for code := range gate {
		if inMatch[code] {
			t.Errorf("the %q refusal is produced both at the gate and in a match, so "+
				"neither set says which screen reads it", code)
		}
	}
	if got, want := len(gate)+len(inMatch), wire.CodeCount-1; got != want {
		t.Errorf("a real room produces %d of the %d refusals; the rest are drawn by "+
			"nothing and reachable from nothing", got, want)
	}
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		base.width, base.height = 200, 60

		for value := range wire.CodeCount {
			code := wire.Code(value)
			// ⚠️ CodeNone is excused **by name and with the reason**: it is the
			// "join" answer, so a screen that drew it would be telling a player
			// they had been turned away at the moment they got in.
			if code == wire.CodeNone {
				continue
			}
			drawn := ""
			where := ""
			switch {
			case gate[code]:
				where = "the join screen"
				drawn = drawnBody(aJoinRefusedWith(t, base, code))
			case inMatch[code]:
				where = "a live battle"
				drawn = drawnBody(aLiveBattleRefusedWith(t, base, code))
			default:
				t.Errorf("nothing a real room does produces the %q refusal, so there is no "+
					"screen it is reachable on and no player will ever read its wording", code)
				continue
			}
			worded := lang.Refusal(code.String())
			if worded == code.String() {
				t.Fatalf("%v leaves the %q refusal at its own id, so the assertion below "+
					"would pass on a screen drawing the raw name", lang, code)
			}
			opening := draw.WrapWords(worded, draw.MinWidth-3)[0]
			if !strings.Contains(drawn, opening) {
				t.Errorf("%s in %s does not read the %q refusal out to the player:\n%s",
					where, lang, code, drawn)
			}
		}

		for value := range wire.ClosureCount {
			closure := wire.Closure(value)
			// ⚠️ ClosureNone is excused by name too, and for the same shape of
			// reason: Closure.Closes is the guard, and a match that stopped for
			// no reason did not stop.
			if closure == wire.ClosureNone {
				continue
			}
			worded := lang.Closure(closure.String())
			if worded == closure.String() {
				t.Fatalf("%v leaves the %q closure at its own id", lang, closure)
			}
			drawn := drawnBody(aMatchClosedBy(base, closure))
			opening := draw.WrapWords(worded, draw.MinWidth-3)[0]
			if !strings.Contains(drawn, opening) {
				t.Errorf("the result screen in %s does not read the %q closure out to the "+
					"player:\n%s", lang, closure, drawn)
			}
		}
	}
	t.Logf("%d of the %d refusals are answers at the gate; the other %d are only reachable "+
		"during a match", len(gate), wire.CodeCount-1, len(inMatch))
}

// theRoomRefusesInAMatch is every wire.Code a **seated** peer can be answered
// with, produced out of a real room rather than listed.
//
// ⚠️ **These three are only reachable during a match**, which is the whole
// reason the battle screen has anywhere to draw a refusal at all: a refusal does
// not end a connection, so a room answers one and the match carries on. Nothing
// on a lobby screen could ever show them.
func theRoomRefusesInAMatch(t *testing.T) map[wire.Code]bool {
	t.Helper()
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	answers := map[wire.Code]bool{}
	// 1. A move from the seat nobody is asking. It is refused rather than
	//    applied, which is what stops one player spending the other's turn.
	playing, awaiting := aSeatedRoom(t, characters)
	answers[refusalIn(t, deliveredBy(t, playing, otherSeat(awaiting),
		wire.Act{Skill: "razor_leaf"}))] = true
	// 2. A move that was never on offer, from the seat that IS being asked.
	playing, awaiting = aSeatedRoom(t, characters)
	answers[refusalIn(t, deliveredBy(t, playing, awaiting,
		wire.Act{Skill: "khong-co-chieu-nay"}))] = true
	// 3. A message the room does not take from a peer at all, which is what two
	//    different builds look like from this end.
	playing, awaiting = aSeatedRoom(t, characters)
	answers[refusalIn(t, deliveredBy(t, playing, awaiting, wire.Welcome{}))] = true

	delete(answers, wire.CodeNone)
	return answers
}

// aSeatedRoom is a room with both seats filled and a turn open, plus the seat
// that turn belongs to.
func aSeatedRoom(t *testing.T, characters *cast.Book) (*room.Room, wire.Seat) {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the embedded books: %v", err)
	}
	opened, err := room.New(room.Config{
		Format: wire.Format3v3, Battles: 1,
		Allowance: room.DefaultAllowance, Seed: 11, TurnCap: room.DefaultTurnCap,
	}, room.Deps{Books: books, Characters: characters, Version: version})
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	sides := [][]string{
		{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"},
		{"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"},
	}
	for index, side := range sides {
		squad := aShippedSide(t, characters, "phe-"+string(rune('a'+index)), side...)
		if _, _, err := opened.Join(wire.Hello{
			Version: version, Squad: squad, Name: squad.ID,
		}); err != nil {
			t.Fatalf("seat a peer: %v", err)
		}
	}
	awaiting, asking := opened.Awaiting()
	if !asking {
		t.Fatal("a room with both seats filled is asking nobody, so no in-match refusal " +
			"can be produced")
	}
	return opened, awaiting
}

func otherSeat(seat wire.Seat) wire.Seat {
	if seat == wire.SeatHost {
		return wire.SeatGuest
	}
	return wire.SeatHost
}

func deliveredBy(t *testing.T, playing *room.Room, from wire.Seat, body wire.Body) []room.Outbound {
	t.Helper()
	out, err := playing.Deliver(from, body)
	if err != nil {
		t.Fatalf("a message from %s was refused with an error rather than a code: %v", from, err)
	}
	return out
}

// aJoinRefusedWith is the join screen after a dial came back with a refusal,
// through the very path a failed Dial takes.
func aJoinRefusedWith(t *testing.T, m model, code wire.Code) model {
	t.Helper()
	joining := m.enter(screenJoin)
	joining.join = joining.join.Failed(&socket.Refusal{Code: code})
	if joining.join.Refused != code.String() {
		t.Fatalf("a *socket.Refusal carrying %q left the screen holding %q",
			code, joining.join.Refused)
	}
	return joining
}

// aLiveBattleRefusedWith is the battle screen after a refusal arrived
// mid-match, through the very path a mirror takes: a real socket.Mirror is fed
// the wire.Refused, and what the screen is handed comes out of Mirror.Read.
//
// ⚠️ **The mirror is real rather than a value built here**, because the claim is
// that this client reads the *latest* refusal off a mirror — and a hand-built
// socket.Sight would be this test agreeing with itself about what a mirror holds.
func aLiveBattleRefusedWith(t *testing.T, m model, code wire.Code) model {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the embedded books: %v", err)
	}
	mirror := socket.NewMirror(wire.SeatHost, books)
	// An earlier refusal as well, so "the latest" is a claim with two candidates
	// rather than one.
	if err := mirror.Receive(wire.Refused{Code: wire.CodeIllegalAction}); err != nil {
		t.Fatalf("the mirror refused a refusal: %v", err)
	}
	if err := mirror.Receive(wire.Refused{Code: code}); err != nil {
		t.Fatalf("the mirror refused a refusal: %v", err)
	}

	live := withAFullLog(t, m.enter(screenBattle))
	if live.battle.Fight == nil {
		t.Fatalf("the battle opened on nothing: %v", live.battle.Err)
	}
	engine, prompt := live.battle.Fight, live.battle.Pending
	var attached draw.PlayScreen
	mirror.Read(func(sight socket.Sight) {
		carried := liveOf(sight)
		carried.Fight, carried.Asking = engine, prompt
		attached = draw.NewPlayScreen().Attach(live.ctx(), carried)
	})
	if attached.LiveRefusal != code.String() {
		t.Fatalf("the mirror was sent %q last and the screen holds %q, so the client is "+
			"not reading the latest refusal", code, attached.LiveRefusal)
	}
	live.battle = attached
	live.screen = screenBattle
	return live
}

// aMatchClosedBy is the result screen for a match the room stopped.
func aMatchClosedBy(m model, closure wire.Closure) model {
	m.screen = screenResult
	m.result = resultScreen{
		Fought:  []socket.Fought{{Battle: 1, Decided: false}},
		Closure: closure.String(),
	}
	return m
}

// theGateAnswers is every wire.Code a **join** can be turned away with,
// produced out of a real room rather than listed.
//
// ⚠️ **This is the whole reason the test above can fail.** Both screens will
// word any code handed to them, so a declared table pairing codes with screens
// could be permuted freely and nothing would notice. Six real refusals out of a
// real room.Room and a real room.Registry is a set nobody wrote down.
//
// What it cannot see: a seventh way the gate could refuse that this file does
// not think to provoke. The count is logged for that reason — a gate that grew
// one would leave that code being checked on the battle screen, which is a
// failure of *this* helper rather than of the claim.
func theGateAnswers(t *testing.T) map[wire.Code]bool {
	t.Helper()
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
	deps := room.Deps{Books: books, Characters: characters, Version: version}
	config := func() room.Config {
		return room.Config{
			Format: wire.Format3v3, Battles: 1,
			Allowance: room.DefaultAllowance, Seed: 11, TurnCap: room.DefaultTurnCap,
		}
	}
	good := aShippedSide(t, characters, "phe-tot",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	hello := func() wire.Hello {
		return wire.Hello{Version: version, Squad: good.Clone(), Name: "An"}
	}

	answers := map[wire.Code]bool{}
	// A room with a password, joined five different wrong ways plus one right
	// one. Each case is a *separate* room where the refusal would otherwise
	// change the room's state.
	guarded := config()
	guarded.Password = "mot-hai-ba"

	// 1. The protocol.
	ahead := hello()
	ahead.Protocol = wire.Protocol + 1
	answers[refusedBy(t, config(), deps, ahead)] = true
	// 2. The data digest.
	stale := hello()
	stale.Data = wire.Digest{Digest: seed.Digest{0xff}}
	answers[refusedBy(t, config(), deps, stale)] = true
	// 3. The password.
	wrong := hello()
	wrong.Password = "khong-dung"
	answers[refusedBy(t, guarded, deps, wrong)] = true
	// 4. The squad, which is the one refusal about what a player brought.
	unfieldable := hello()
	unfieldable.Squad = placement.Squad{ID: "trong", Name: "trong"}
	answers[refusedBy(t, config(), deps, unfieldable)] = true
	// 5. Both seats taken.
	full, err := room.New(config(), deps)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	for range 2 {
		if _, _, err := full.Join(hello()); err != nil {
			t.Fatalf("a good hello was refused with an error: %v", err)
		}
	}
	answers[refusalIn(t, joinedWith(t, full, hello()))] = true
	// 6. A code no room answers to, which is the registry's own refusal and the
	//    one that does not come out of a Room at all.
	rooms := room.NewRegistry()
	t.Cleanup(func() { rooms.CloseAll(); rooms.Wait() })
	unknown, err := rooms.Join("AAAAAAAAAAAA", hello())
	if err != nil {
		t.Fatalf("joining an unknown room reported an error rather than a refusal: %v", err)
	}
	answers[refusalIn(t, unknown.Out)] = true

	delete(answers, wire.CodeNone)
	if len(answers) != 6 {
		named := make([]string, 0, len(answers))
		for code := range answers {
			named = append(named, code.String())
		}
		slices.Sort(named)
		t.Fatalf("six ways of being turned away at a gate produced %d distinct codes (%v), "+
			"so two of the cases above are answering with the same refusal and one code is "+
			"being checked on the wrong screen", len(answers), named)
	}
	return answers
}

// refusedBy opens a room, offers it one hello, and hands back the code it was
// turned away with.
func refusedBy(t *testing.T, config room.Config, deps room.Deps, hello wire.Hello) wire.Code {
	t.Helper()
	opened, err := room.New(config, deps)
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	return refusalIn(t, joinedWith(t, opened, hello))
}

func joinedWith(t *testing.T, opened *room.Room, hello wire.Hello) []room.Outbound {
	t.Helper()
	_, out, err := opened.Join(hello)
	if err != nil {
		t.Fatalf("a hello was refused with an error rather than a code: %v", err)
	}
	return out
}

// refusalIn is the one wire.Refused in what a room answered with.
func refusalIn(t *testing.T, out []room.Outbound) wire.Code {
	t.Helper()
	for _, message := range out {
		switch refused := message.Body.(type) {
		case wire.Refused:
			return refused.Code
		case *wire.Refused:
			return refused.Code
		}
	}
	t.Fatalf("a room answered %d messages and none of them was a refusal", len(out))
	return wire.CodeNone
}
