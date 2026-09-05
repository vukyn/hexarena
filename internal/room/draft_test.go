package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The five assertions the room's draft half OWES, and why they exist
//
// ⚠️ **internal/wire's golden is hand-written, so it pins the FORMAT and can
// never see the PRODUCER.** Its own doc comment carries the measurement: taking
// one line out of gate.go so the room stops filling in TurnCap leaves that
// package — golden included — entirely green, because the golden's welcome is a
// fixture's and not a room's. So a field added to a message needs two things, an
// entry there saying it travels and an assertion here saying it is filled in, and
// each of the five tests named below is that gap closed for one of them:
//
//  1. Welcome.Drafts is really set for a drafting room **and really false
//     otherwise** — TestAWelcomeSaysWhetherTheRoomDrafts. Both halves, because a
//     field hardcoded true passes the first alone.
//  2. Drafted.Decisions is what draft.Since answered, in that order, and is never
//     sent empty — TestEveryDraftedCarriesTheRecordInOrderAndIsNeverEmpty.
//  3. A Decide is routed with the seat taken from the **connection** and never
//     from the message — TestADecideIsRoutedByTheConnectionAndNeverByTheMessage.
//  4. wire.CodeSquadUnwanted answers a hello that brought a squad to a drafting
//     room — TestADraftingRoomRefusesASquadAsUnwanted.
//  5. wire.ClosureDraftExpired is sent when the draft's timeout cancels —
//     TestADraftTimeoutClosesTheRoomAsExpired.

// pickingDecisions is how many decisions the ban and pick takes before the
// arrange phase opens, **derived** rather than written down: every ban slot is
// one decision, spent or skipped, and every pick is two — the character, then its
// loadout. At 3v3 that is four and twelve.
func pickingDecisions(format wire.Format) int {
	return 2*draft.BansPerSide(format) + 4*draft.PicksPerSide(format)
}

// draftDecisions is the whole draft: the picking, plus one arrangement a side.
func draftDecisions(format wire.Format) int { return pickingDecisions(format) + seatsInARoom }

// aDraftingRoom is a room that drafts with both seats taken, so its draft is
// open and waiting on the host's first ban, plus the two clients mirroring it.
//
// ⚠️ It asserts that opening the draft **said nothing** — one welcome each and no
// wire.Drafted — because that is the design rather than an accident: a Drafted
// carries recorded decisions, none have been taken, and a room must not send one
// carrying none.
func aDraftingRoom(t *testing.T) (*room.Room, *table) {
	t.Helper()
	dependencies := deps(t)
	configuration := draftingConfig(11)
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)
	for _, name := range []string{"Host", "Guest"} {
		_, out, err := opened.Join(helloWithNoSquad(t, name))
		if err != nil {
			t.Fatalf("%s joins a drafting room: %v", name, err)
		}
		if len(out) != 1 {
			t.Fatalf("%s's join produced %d messages, want only a welcome: opening a draft "+
				"announces nothing, because nothing has been decided yet", name, len(out))
		}
		clients.deliver(t, out)
	}
	if onTurn, waiting := opened.Awaiting(); !waiting || onTurn != wire.SeatHost {
		t.Fatalf("a drafting room with both seats taken is waiting on %q/%v, want the host: "+
			"the host bans first and nothing on the wire says so", onTurn, waiting)
	}
	return opened, clients
}

// oneDecision hands the room the decision the seat it is waiting on would make,
// read off that client's own draft, and reports the seat, the body it sent and
// everything the room said back.
func oneDecision(t *testing.T, opened *room.Room, clients *table) (wire.Seat, wire.Decide, []room.Outbound) {
	t.Helper()
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatalf("the room is waiting on nobody and its draft is not finished")
	}
	deciding, ok := clients.at(onTurn).decide().(wire.Decide)
	if !ok {
		t.Fatalf("%s answered a draft decision with something that is not a wire.Decide", onTurn)
	}
	answered, err := opened.Deliver(onTurn, deciding)
	if err != nil {
		t.Fatalf("a decision from %s: %v", onTurn, err)
	}
	clients.deliver(t, answered)
	return onTurn, deciding, answered
}

// draftUntilArranging drives the ban and pick to the point where both sides have
// to arrange, which is the phase every clock assertion below is about, and
// reports what the room said on the way.
//
// The bound is the derived decision count and not a round number: a state machine
// that stopped progressing would otherwise hang the suite instead of failing it.
func draftUntilArranging(t *testing.T, opened *room.Room, clients *table) []room.Outbound {
	t.Helper()
	var said []room.Outbound
	for at := range pickingDecisions(wire.Format3v3) {
		_, _, answered := oneDecision(t, opened, clients)
		said = append(said, answered...)
		if clients.host.drafting.Arranging() {
			if at+1 != pickingDecisions(wire.Format3v3) {
				t.Fatalf("the arrange phase opened after %d decisions and the picking takes %d",
					at+1, pickingDecisions(wire.Format3v3))
			}
			return said
		}
	}
	t.Fatalf("the picking took %d decisions and the arrange phase never opened",
		pickingDecisions(wire.Format3v3))
	return nil
}

// arrangementOf is one seat's whole arrangement as the message it sends, built
// from that client's own draft.
func arrangementOf(t *testing.T, clients *table, seat wire.Seat) wire.Decide {
	t.Helper()
	picks := clients.at(seat).drafting.Picks()[seatIndexOf(t, seat)]
	return wire.Decide{DraftDecision: wire.DraftDecision{
		Step:  wire.StepArrange,
		Slots: firstCells(len(picks)),
	}}
}

// onlyWelcome is the welcome a join is expected to be the whole of, for a test
// reading a field off the message a real room built.
func onlyWelcome(t *testing.T, out []room.Outbound) wire.Welcome {
	t.Helper()
	if len(out) != 1 {
		t.Fatalf("the gate answered with %d messages, want one welcome", len(out))
	}
	welcome, ok := out[0].Body.(wire.Welcome)
	if !ok {
		t.Fatalf("the gate answered with a %s, want a welcome", out[0].Body.Kind())
	}
	return welcome
}

// draftedIn is the one wire.Drafted in a batch, and the assertion that there is
// exactly one and that it is addressed to both seats.
func draftedIn(t *testing.T, out []room.Outbound) []wire.DraftEntry {
	t.Helper()
	seen := map[wire.Seat]wire.Drafted{}
	for _, message := range out {
		drafted, ok := message.Body.(wire.Drafted)
		if !ok {
			continue
		}
		if _, twice := seen[message.To]; twice {
			t.Fatalf("the room sent %q two draft records for one decision", message.To)
		}
		seen[message.To] = drafted
	}
	if len(seen) != seatsInARoom {
		t.Fatalf("a recorded decision reached %d of the %d seats; every decision is public "+
			"the moment it is recorded", len(seen), seatsInARoom)
	}
	return seen[wire.SeatHost].Decisions
}

// startsIn is how many wire.Start messages a batch carries, which is what "the
// battle began" looks like from outside the room.
func startsIn(out []room.Outbound) int {
	starts := 0
	for _, message := range out {
		if _, ok := message.Body.(wire.Start); ok {
			starts++
		}
	}
	return starts
}

// closureIn is the one wire.Closed addressed to a seat, and whether there was
// one at all.
func closureIn(out []room.Outbound, seat wire.Seat) (wire.Closure, bool) {
	for _, message := range out {
		closed, ok := message.Body.(wire.Closed)
		if ok && message.To == seat {
			return closed.Reason, true
		}
	}
	return wire.ClosureNone, false
}

// TestAWelcomeSaysWhetherTheRoomDrafts is owed assertion 1, and it is **both
// halves on purpose**: a Drafts field hardcoded true passes the drafting case
// alone, and one hardcoded false passes the ordinary case alone, so either half
// by itself measures a constant.
//
// The value is compared against the **room's own reading of its configuration**
// rather than against the local literal, which is the stronger claim and the one
// TestTwoFakeClientsFightAWholeBo3InProcess already makes for the allowance and
// the cap: what a client was told and what the room is running under have to be
// one fact.
func TestAWelcomeSaysWhetherTheRoomDrafts(t *testing.T) {
	dependencies := deps(t)
	brought := squadOf(t, dependencies.Characters, "brought.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")

	for _, one := range []struct {
		name          string
		configuration room.Config
		joining       wire.Hello
	}{
		{"a room that drafts", draftingConfig(11), helloWithNoSquad(t, "An")},
		{"a room that does not", config(11, 1), hello(t, brought, "An")},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened := newRoom(t, one.configuration)
			seat, out, err := opened.Join(one.joining)
			if err != nil {
				t.Fatalf("join: %v", err)
			}
			if !seat.Valid() {
				t.Fatalf("the hello was refused with %v", out)
			}
			welcome := onlyWelcome(t, out)
			if welcome.Drafts != one.configuration.Drafts {
				t.Errorf("%s welcomed a client with drafts=%v, want %v",
					one.name, welcome.Drafts, one.configuration.Drafts)
			}
			if welcome.Drafts != opened.Config().Drafts {
				t.Errorf("the welcome says drafts=%v and the room is running under %v, "+
					"which have to be one fact", welcome.Drafts, opened.Config().Drafts)
			}
		})
	}
}

// TestEveryDraftedCarriesTheRecordInOrderAndIsNeverEmpty is owed assertion 2, and
// the **guest arranges first** on purpose: that is the one decision in a draft
// whose arrival order differs from its record order, so it is the only thing that
// can tell "in that order" apart from "in some order".
//
// Three claims, and each of them is a thing that could be quietly wrong while a
// whole draft still played out:
//
//  1. **Every entry the room recorded reached both clients, in record order.**
//     Compared against the sequence this test drove, entry by entry, with the
//     arrangements written out as the record orders them rather than as they
//     arrived.
//  2. **The two arrangements arrive together and in seats order**, host then
//     guest, though the guest's went in first — nothing reaches the record until
//     both are in, because an entry is public the moment it is sent.
//  3. **No wire.Drafted is ever empty.** Held twice: the fixture fatals on an
//     empty batch for every test in this file, and the message count here is
//     exactly one fewer than the decision count, which names *which* decision
//     answered with nothing.
func TestEveryDraftedCarriesTheRecordInOrderAndIsNeverEmpty(t *testing.T) {
	opened, clients := aDraftingRoom(t)

	// The picking, recording what was driven so the record can be compared
	// against it rather than against itself.
	var wanted []wire.DraftEntry
	for range pickingDecisions(wire.Format3v3) {
		onTurn, sent, answered := oneDecision(t, opened, clients)
		entries := draftedIn(t, answered)
		if len(entries) != 1 {
			t.Fatalf("%s's %s was answered with %d recorded decisions, want one",
				onTurn, sent.Step, len(entries))
		}
		wanted = append(wanted, wire.DraftEntry{Seat: onTurn, DraftDecision: sent.DraftDecision})
	}
	if !clients.host.drafting.Arranging() {
		t.Fatalf("after %d decisions the arrange phase is not open", len(wanted))
	}

	// The guest arranges first, and is answered with **nothing at all**.
	guestArranging := arrangementOf(t, clients, wire.SeatGuest)
	answered, err := opened.Deliver(wire.SeatGuest, guestArranging)
	if err != nil {
		t.Fatalf("the guest arranges: %v", err)
	}
	if len(answered) != 0 {
		t.Fatalf("the first arrangement was answered with %d messages; nothing reaches the "+
			"record until both are in, and a room must not send a wire.Drafted carrying none",
			len(answered))
	}
	clients.deliver(t, answered)

	// Then the host, which closes the phase and puts **both** in at once.
	hostArranging := arrangementOf(t, clients, wire.SeatHost)
	answered, err = opened.Deliver(wire.SeatHost, hostArranging)
	if err != nil {
		t.Fatalf("the host arranges: %v", err)
	}
	pair := draftedIn(t, answered)
	if len(pair) != seatsInARoom {
		t.Fatalf("closing the arrange phase recorded %d decisions, want both arrangements",
			len(pair))
	}
	// Claim 2, which is the sharp one: recorded host-first though the guest
	// arranged first.
	for at, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if pair[at].Seat != seat {
			t.Errorf("the arrange phase recorded %q at position %d, want %q: the pair goes in "+
				"in seats order and not in arrival order, and the guest arranged first here",
				pair[at].Seat, at, seat)
		}
		if pair[at].Step != wire.StepArrange {
			t.Errorf("position %d of the arrange pair is a %q", at, pair[at].Step)
		}
	}
	wanted = append(wanted,
		wire.DraftEntry{Seat: wire.SeatHost, DraftDecision: hostArranging.DraftDecision},
		wire.DraftEntry{Seat: wire.SeatGuest, DraftDecision: guestArranging.DraftDecision})
	clients.deliver(t, answered)

	// Claim 1, per client: what each was sent is the record, in order.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if len(client.decided) != len(wanted) {
			t.Fatalf("%s was sent %d recorded decisions over a draft of %d",
				client.seat, len(client.decided), len(wanted))
		}
		for at, entry := range client.decided {
			if entry.Seat != wanted[at].Seat || entry.Step != wanted[at].Step ||
				entry.Character != wanted[at].Character {
				t.Errorf("%s's entry %d is %s's %q of %q, want %s's %q of %q",
					client.seat, at, entry.Seat, entry.Step, entry.Character,
					wanted[at].Seat, wanted[at].Step, wanted[at].Character)
			}
		}
		// Claim 3's other half. Every decision but one produced a message; the
		// one is the first arrangement, which recorded nothing.
		if want := draftDecisions(wire.Format3v3) - 1; len(client.records) != want {
			t.Errorf("%s was sent %d draft records over %d decisions, want %d: exactly one "+
				"decision — the first arrangement — is answered with no message at all",
				client.seat, len(client.records), draftDecisions(wire.Format3v3), want)
		}
		if client.applied != len(wanted) {
			t.Errorf("%s replayed %d of the %d recorded decisions",
				client.seat, client.applied, len(wanted))
		}
	}
	t.Logf("%d decisions recorded as %d entries in %d messages; the arrange pair went in "+
		"host-first having arrived guest-first",
		draftDecisions(wire.Format3v3), len(wanted), len(clients.host.records))
}

// TestADecideIsRoutedByTheConnectionAndNeverByTheMessage is owed assertion 3.
//
// ⚠️ **The mutation this is written against cannot be compiled, and that is the
// finding rather than a gap.** wire.Decide embeds a wire.DraftDecision, which has
// no seat field at all, so "take the seat from the message" is not an expression
// that exists — the enforcement is structural. What *is* compilable is the room
// deriving the seat from something other than the connection (a fixed seat, the
// seat that went first), and that is what the two halves below catch.
//
// The second half is the sharper one and it is only available in the arrange
// phase: **both** seats are being asked at once there, so the identical message
// value can be sent from each connection in turn — and the two entries it records
// differ only in a seat that came from nowhere but the connection.
func TestADecideIsRoutedByTheConnectionAndNeverByTheMessage(t *testing.T) {
	opened, clients := aDraftingRoom(t)

	// Half one, in the picking: the draft is waiting on the host, so the host's
	// own decision sent from the guest's connection is refused and moves nothing.
	hostBanning, ok := clients.host.decide().(wire.Decide)
	if !ok {
		t.Fatal("the host answered its ban with something that is not a wire.Decide")
	}
	answered, err := opened.Deliver(wire.SeatGuest, hostBanning)
	if err != nil {
		t.Fatalf("the guest sends the host's ban: %v", err)
	}
	if code := onlyCodeFor(t, answered, wire.SeatGuest); code != wire.CodeNotYourTurn {
		t.Errorf("the host's own ban sent from the guest's connection was answered %q, want %q",
			code, wire.CodeNotYourTurn)
	}
	if onTurn, waiting := opened.Awaiting(); !waiting || onTurn != wire.SeatHost {
		t.Errorf("after a refused decision the room is waiting on %q/%v, want the host still",
			onTurn, waiting)
	}
	// The same body from the host's connection goes in, as the host's.
	answered, err = opened.Deliver(wire.SeatHost, hostBanning)
	if err != nil {
		t.Fatalf("the host sends its own ban: %v", err)
	}
	recorded := draftedIn(t, answered)
	if len(recorded) != 1 || recorded[0].Seat != wire.SeatHost {
		t.Fatalf("the host's ban recorded %v, want one entry naming the host", recorded)
	}
	clients.deliver(t, answered)

	// Half two, in the arrange phase, where both seats are asked at once.
	for range pickingDecisions(wire.Format3v3) - 1 {
		oneDecision(t, opened, clients)
	}
	if !clients.host.drafting.Arranging() {
		t.Fatalf("the arrange phase did not open after the picking")
	}
	// ⚠️ One message VALUE, sent from two connections. Both sides drafted three
	// units and both arrange on the same cells, so the two bodies are equal by
	// construction — the entries the room records may therefore differ in the
	// seat and in nothing else, and the seat is not in the body.
	shared := arrangementOf(t, clients, wire.SeatGuest)
	if other := arrangementOf(t, clients, wire.SeatHost); len(other.Slots) != len(shared.Slots) {
		t.Fatalf("the two sides arrange %d and %d cells, so one message cannot stand for both",
			len(other.Slots), len(shared.Slots))
	}
	if answered, err = opened.Deliver(wire.SeatGuest, shared); err != nil {
		t.Fatalf("the guest arranges: %v", err)
	}
	clients.deliver(t, answered)
	if answered, err = opened.Deliver(wire.SeatHost, shared); err != nil {
		t.Fatalf("the host arranges: %v", err)
	}
	pair := draftedIn(t, answered)
	if len(pair) != seatsInARoom {
		t.Fatalf("the arrange phase closed with %d entries", len(pair))
	}
	for at, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if pair[at].Seat != seat {
			t.Errorf("one message value sent from two connections recorded %q at position %d, "+
				"want %q: the seat comes from the connection and the message has none",
				pair[at].Seat, at, seat)
		}
	}
}

// TestADraftingRoomRefusesASquadAsUnwanted is owed assertion 4, and it is the
// producer wire.CodeSquadUnwanted was declared without: cmd/hexarena-tui's
// TestEveryRefusalIsShownAndEveryClosureIsShown derives which screen draws each
// code out of a real room.Room, so until a room answered this one it sat in that
// test's `owed` set. This is what let that entry be deleted.
//
// ⚠️ **The legal-squad case is the point.** A squad that would be accepted
// unchanged by a room that does not draft is refused here — so the refusal is
// about the *room* and not about the squad, which is the whole reason the code is
// new rather than CodeSquadRefused. And an **illegal** squad is refused with the
// same code, which is what says squadIsFieldable is not consulted on this path: a
// player is told to bring none rather than sent to fix levels and forms that were
// never the question.
func TestADraftingRoomRefusesASquadAsUnwanted(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	illegal := legal.Clone()
	illegal.Units = illegal.Units[:2]

	// The same legal squad, in a room that does not draft, is accepted — which is
	// what makes the refusal below a fact about the room.
	ordinary := newRoom(t, config(11, 1))
	if seat, out, err := ordinary.Join(hello(t, legal, "An")); err != nil || !seat.Valid() {
		t.Fatalf("a room that does not draft refused the fixture squad (%v): %v", out, err)
	}

	for _, one := range []struct {
		name  string
		squad placement.Squad
	}{
		{"a legal squad", legal},
		{"an illegal squad", illegal},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened := newRoom(t, draftingConfig(11))
			seat, out, err := opened.Join(hello(t, one.squad, "An"))
			if err != nil {
				t.Fatalf("join: %v", err)
			}
			if seat.Valid() {
				t.Fatalf("a drafting room seated %q for a hello that brought a squad", seat)
			}
			if code := onlyCode(t, out); code != wire.CodeSquadUnwanted {
				t.Errorf("a drafting room answered %s with %q, want %q: the fix is to bring "+
					"none, and %q tells a player to fix the squad instead",
					one.name, code, wire.CodeSquadUnwanted, wire.CodeSquadRefused)
			}
			// The refusal left the room joinable: refusing is what stops a seat
			// being handed out.
			if _, out, err := opened.Join(helloWithNoSquad(t, "An")); err != nil {
				t.Fatalf("joining with no squad after a refusal: %v", err)
			} else if welcome := onlyWelcome(t, out); !welcome.Drafts {
				t.Error("the second attempt was welcomed into a room that does not draft")
			}
		})
	}

	// ⚠️ The order is unchanged by any of this: the seat is still checked before
	// the squad, so a full drafting room says it is full rather than saying the
	// squad it was never going to look at is unwanted.
	full := newRoom(t, draftingConfig(11))
	for range seatsInARoom {
		if _, _, err := full.Join(helloWithNoSquad(t, "Seated")); err != nil {
			t.Fatalf("filling a drafting room: %v", err)
		}
	}
	_, out, err := full.Join(hello(t, legal, "Late"))
	if err != nil {
		t.Fatalf("joining a full drafting room: %v", err)
	}
	if code := onlyCode(t, out); code != wire.CodeRoomFull {
		t.Errorf("a full drafting room answered a hello with a squad %q, want %q: the seat is "+
			"checked before the squad and a squad check on a full room is work done to reach "+
			"an answer already decided", code, wire.CodeRoomFull)
	}
}

// TestADraftTimeoutClosesTheRoomAsExpired is owed assertion 5.
//
// ⚠️ **A draft timeout does not pass anything — it closes the room**, which is the
// one place the design does not follow "a timeout announces and passes": a side
// that never picked has no squad to fight with. So the claims are that **both**
// peers are told (both are holding an open decision, and in the arrange phase
// literally both at once), that the verdict is the one an ending nobody is charged
// with produces, and that no battle ever started.
//
// The refused half is measured too, and it is not tidiness: internal/socket reads
// wire.CodeNotYourTurn **alone** as "this report was late", counts it and re-arms
// rather than dropping anybody, so a spurious timeout answered with any other code
// would drop a player for arranging quickly.
func TestADraftTimeoutClosesTheRoomAsExpired(t *testing.T) {
	t.Run("while picking", func(t *testing.T) {
		opened, clients := aDraftingRoom(t)
		for range 3 {
			oneDecision(t, opened, clients)
		}
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatal("the draft is waiting on nobody four decisions in")
		}

		// A timeout for the seat nobody is waiting on is refused and closes
		// nothing.
		idle, err := opened.TimedOut(otherOf(t, onTurn))
		if err != nil {
			t.Fatalf("a timeout for the seat not on turn: %v", err)
		}
		if code := onlyCodeFor(t, idle, otherOf(t, onTurn)); code != wire.CodeNotYourTurn {
			t.Errorf("a timeout for the seat not being asked was answered %q, want %q, which "+
				"is the code the transport reads as a late report", code, wire.CodeNotYourTurn)
		}
		if opened.Finished() {
			t.Fatal("a refused timeout ended the match")
		}

		closed, err := opened.TimedOut(onTurn)
		if err != nil {
			t.Fatalf("%s's allowance runs out: %v", onTurn, err)
		}
		assertDraftExpired(t, opened, clients, closed)
	})

	t.Run("while arranging", func(t *testing.T) {
		opened, clients := aDraftingRoom(t)
		draftUntilArranging(t, opened, clients)

		// The host arranges, so it has nothing of its own left to run out — and a
		// timeout naming it is refused for that reason rather than cancelling on
		// behalf of a seat nobody is waiting on.
		answered, err := opened.Deliver(wire.SeatHost, arrangementOf(t, clients, wire.SeatHost))
		if err != nil {
			t.Fatalf("the host arranges: %v", err)
		}
		clients.deliver(t, answered)
		refused, err := opened.TimedOut(wire.SeatHost)
		if err != nil {
			t.Fatalf("a timeout for the seat that has arranged: %v", err)
		}
		if code := onlyCodeFor(t, refused, wire.SeatHost); code != wire.CodeNotYourTurn {
			t.Errorf("a timeout for a seat that had already arranged was answered %q, want %q",
				code, wire.CodeNotYourTurn)
		}
		if opened.Finished() {
			t.Fatal("a timeout for a seat that had already arranged cancelled the draft")
		}

		closed, err := opened.TimedOut(wire.SeatGuest)
		if err != nil {
			t.Fatalf("the guest's allowance runs out while arranging: %v", err)
		}
		assertDraftExpired(t, opened, clients, closed)
	})
}

// assertDraftExpired is the whole of what a cancelled draft leaves behind, so the
// two cases above hold one claim rather than two copies of it.
func assertDraftExpired(t *testing.T, opened *room.Room, clients *table, closed []room.Outbound) {
	t.Helper()
	if startsIn(closed) != 0 {
		t.Errorf("a cancelled draft started %d battles", startsIn(closed))
	}
	for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		reason, told := closureIn(closed, seat)
		if !told {
			t.Errorf("%s was not told the draft expired; both peers are holding an open draft "+
				"decision, so one of them left unanswered waits for a decision nobody is "+
				"coming to take", seat)
			continue
		}
		if reason != wire.ClosureDraftExpired {
			t.Errorf("%s was told the room closed because %q, want %q",
				seat, reason, wire.ClosureDraftExpired)
		}
	}
	clients.deliver(t, closed)
	for _, client := range []*mirror{clients.host, clients.guest} {
		if len(client.starts) != 0 {
			t.Errorf("%s was started on a battle in a draft that expired", client.seat)
		}
		if len(client.closures) != 1 || client.closures[0] != wire.ClosureDraftExpired {
			t.Errorf("%s read the closures %v", client.seat, client.closures)
		}
	}
	if !opened.Finished() {
		t.Fatal("a cancelled draft left the room unfinished, so the registry would keep it " +
			"joinable and its goroutine alive")
	}
	result := opened.Result()
	if result.Verdict != room.VerdictAbandoned {
		t.Errorf("a cancelled draft ended %q, want %q: nobody drafted a squad, so there is no "+
			"board to have a verdict about", result.Verdict, room.VerdictAbandoned)
	}
	if result.Winner.Valid() {
		t.Errorf("a cancelled draft names %q as its winner", result.Winner)
	}
	// ⚠️ Departed is the zero Seat, and that absence is what tells a cancelled
	// draft apart from a departure in the room's own record. → draftTimedOut.
	if result.Departed.Valid() {
		t.Errorf("a cancelled draft records %q as having gone away, and nobody did",
			result.Departed)
	}
	if len(opened.Played()) != 0 {
		t.Errorf("a cancelled draft recorded %d battles", len(opened.Played()))
	}
}

// TestTheArrangePhaseSerialisesItsAllowance is the phase's clock, which is the
// awkward half of hosting a draft: room.Reading carries **one** Awaiting,
// internal/socket arms exactly one timer off it, and the arrange phase has
// **both** seats pending at once by design.
//
// Neither of those is widened. The seat answered is the first in
// draft.AwaitingArrangement — array-derived, so it reaches this output
// deterministically — and what that costs is written on draftAwaiting and
// measured here:
//
//   - while **neither** side has arranged the seat named is the host, because
//     that is seats order and neither is more owed than the other;
//   - once one side has arranged the name is **exact**, whichever side it was, so
//     the seat a timeout would blame is the slow one in every case that has a slow
//     one.
//
// ⚠️ Both orders are driven, and that is what makes this able to fail: answering
// a fixed seat throughout passes the host-first case and fails the guest-first
// one, and vice versa.
func TestTheArrangePhaseSerialisesItsAllowance(t *testing.T) {
	for _, one := range []struct {
		name          string
		arrangesFirst wire.Seat
	}{
		{"the host arranges first", wire.SeatHost},
		{"the guest arranges first", wire.SeatGuest},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened, clients := aDraftingRoom(t)
			draftUntilArranging(t, opened, clients)

			// Both pending: the phase is answered with the first seat still to
			// arrange, which is the host by seats order and not by anything
			// about the two sides.
			onTurn, waiting := opened.Awaiting()
			if !waiting {
				t.Fatal("the arrange phase is waiting on nobody, so no allowance is armed and " +
					"a draft nobody answers hangs for ever")
			}
			if onTurn != wire.SeatHost {
				t.Errorf("with both sides still to arrange the room is waiting on %q, want the "+
					"host: the phase is serialised onto the first seat in seats order", onTurn)
			}

			answered, err := opened.Deliver(one.arrangesFirst,
				arrangementOf(t, clients, one.arrangesFirst))
			if err != nil {
				t.Fatalf("%s arranges: %v", one.arrangesFirst, err)
			}
			clients.deliver(t, answered)

			// One side left, and the seat named is now exactly that side.
			owed := otherOf(t, one.arrangesFirst)
			onTurn, waiting = opened.Awaiting()
			if !waiting {
				t.Fatalf("with %s still to arrange the room is waiting on nobody", owed)
			}
			if onTurn != owed {
				t.Errorf("%s arranged and the room is waiting on %q, want %q: once one side "+
					"has answered the seat named is the one still owed", one.arrangesFirst,
					onTurn, owed)
			}

			// And the phase closing hands the clock over to the battle.
			answered, err = opened.Deliver(owed, arrangementOf(t, clients, owed))
			if err != nil {
				t.Fatalf("%s arranges: %v", owed, err)
			}
			if startsIn(answered) != seatsInARoom {
				t.Fatalf("closing the arrange phase started %d battles, want one wire.Start a "+
					"seat", startsIn(answered))
			}
			clients.deliver(t, answered)
			if _, waiting = opened.Awaiting(); !waiting {
				t.Error("the battle opened waiting on nobody")
			}
			if clients.host.fight == nil || clients.guest.fight == nil {
				t.Error("a client was not started on the battle the draft produced")
			}
		})
	}
}

// TestADraftingRoomStartsNoBattleUntilBothSidesHaveArranged is begin() being
// called on Done() and not on Picked(), which is a live bug rather than a slow
// one: draft.Squads answers two squads with **nobody in them** until both sides
// have arranged, on purpose — hex.Offset's zero value is a real cell, so an
// honestly empty squad beats a plausible one — so a room that began on Picked
// would hand begin() an empty roster.
//
// It counts rather than spot-checks, because "no battle yet" is a claim about
// every decision of the draft and not only the last.
func TestADraftingRoomStartsNoBattleUntilBothSidesHaveArranged(t *testing.T) {
	opened, clients := aDraftingRoom(t)

	for at := range pickingDecisions(wire.Format3v3) {
		onTurn, sent, answered := oneDecision(t, opened, clients)
		if starts := startsIn(answered); starts != 0 {
			t.Fatalf("%s's %s (decision %d of the picking) started %d battles, and the draft "+
				"is not finished", onTurn, sent.Step, at+1, starts)
		}
	}
	answered, err := opened.Deliver(wire.SeatHost, arrangementOf(t, clients, wire.SeatHost))
	if err != nil {
		t.Fatalf("the host arranges: %v", err)
	}
	if starts := startsIn(answered); starts != 0 {
		t.Fatalf("the first arrangement started %d battles; the draft is Picked and not Done, "+
			"and Squads answers two empty squads until both sides have arranged", starts)
	}
	clients.deliver(t, answered)
	if len(opened.Played()) != 0 || opened.Finished() {
		t.Fatal("a room mid-draft has played a battle")
	}

	answered, err = opened.Deliver(wire.SeatGuest, arrangementOf(t, clients, wire.SeatGuest))
	if err != nil {
		t.Fatalf("the guest arranges: %v", err)
	}
	if starts := startsIn(answered); starts != seatsInARoom {
		t.Fatalf("the draft finished and started %d battles, want one wire.Start a seat", starts)
	}
	clients.deliver(t, answered)

	// ⚠️ **The record and the starts arrive in one batch and in that order**, so a
	// client applies the two arrangements before it is asked to build a battle
	// out of them.
	recorded, started := -1, -1
	for at, message := range answered {
		switch message.Body.(type) {
		case wire.Drafted:
			if recorded < 0 {
				recorded = at
			}
		case wire.Start:
			if started < 0 {
				started = at
			}
		}
	}
	if recorded < 0 || started < 0 || recorded > started {
		t.Errorf("the closing batch put its first wire.Start at %d and its first wire.Drafted "+
			"at %d: a client has to be told the arrangements before it is started on the "+
			"battle they produced", started, recorded)
	}
}

// TestAPeerLeavingMidDraftClosesTheRoom is the departure path, **checked rather
// than assumed** — and the check found it wrong.
//
// ⚠️ A draft runs with no battle open and nothing played, which is exactly the
// shape Room.Left's pre-match arm matched: the seat was freed and the room went
// back to waiting for a joiner, with the departed side's bans and picks still in
// the draft and the peer still there holding an open decision nobody was coming
// to take. The ending itself is the existing one — ClosureLeft, VerdictAbandoned,
// nobody charged with anything.
func TestAPeerLeavingMidDraftClosesTheRoom(t *testing.T) {
	opened, clients := aDraftingRoom(t)
	for range 5 {
		oneDecision(t, opened, clients)
	}

	out, err := opened.Left(wire.SeatGuest)
	if err != nil {
		t.Fatalf("the guest leaves mid-draft: %v", err)
	}
	if reason, told := closureIn(out, wire.SeatHost); !told || reason != wire.ClosureLeft {
		t.Errorf("the host was told %q/%v when its opponent left mid-draft, want %q",
			reason, told, wire.ClosureLeft)
	}
	if _, told := closureIn(out, wire.SeatGuest); told {
		t.Error("the room wrote to the seat that had already gone")
	}
	if !opened.Finished() {
		t.Fatal("a peer leaving mid-draft left the room unfinished, so it stays joinable with " +
			"the departed side's picks still in its draft and the other peer waiting for a " +
			"decision nobody is coming to take")
	}
	result := opened.Result()
	if result.Verdict != room.VerdictAbandoned {
		t.Errorf("a departure mid-draft ended %q, want %q", result.Verdict, room.VerdictAbandoned)
	}
	if result.Departed != wire.SeatGuest {
		t.Errorf("the departure names %q, want the guest", result.Departed)
	}
	if result.Winner.Valid() {
		t.Errorf("a departure mid-draft names %q the winner; leaving costs nothing",
			result.Winner)
	}
	// ⚠️ **The room is waiting on nobody**, and this is the assertion that found a
	// real bug rather than confirming one. A departure ends the match through
	// abandon and leaves the draft neither Done nor Cancelled, so a draft-open
	// test written on those two alone goes on answering an open decision after the
	// match is over — and a transport arms its allowance off exactly this, so it
	// would start a countdown on a finished room. → draftOpen.
	if onTurn, waiting := opened.Awaiting(); waiting {
		t.Errorf("a room whose match ended mid-draft is waiting on %q; Awaiting is false once "+
			"the match is over, and a transport starts its allowance on it", onTurn)
	}
	// The same, from the two inputs a transport could still send.
	if refused, err := opened.TimedOut(wire.SeatHost); err != nil {
		t.Fatalf("a timeout for an abandoned drafting room: %v", err)
	} else if code := onlyCodeFor(t, refused, wire.SeatHost); code != wire.CodeNotYourTurn {
		t.Errorf("a timeout after a mid-draft departure was answered %q, want %q",
			code, wire.CodeNotYourTurn)
	}
	stale := wire.Decide{DraftDecision: wire.DraftDecision{
		Step: wire.StepBan, Character: "pokemon.pichu",
	}}
	if answered, err := opened.Deliver(wire.SeatHost, stale); err != nil {
		t.Fatalf("a decision for an abandoned drafting room: %v", err)
	} else if code := onlyCodeFor(t, answered, wire.SeatHost); code != wire.CodeNotYourTurn {
		t.Errorf("a decision after a mid-draft departure was answered %q, want %q",
			code, wire.CodeNotYourTurn)
	}
	// The room is over, so the seat is not free for a new joiner to walk into a
	// half-played draft.
	if _, refused, err := opened.Join(helloWithNoSquad(t, "Late")); err != nil {
		t.Fatalf("joining an abandoned drafting room: %v", err)
	} else if code := onlyCode(t, refused); code != wire.CodeRoomFull {
		t.Errorf("a joiner arriving after a mid-draft departure was answered %q, want %q",
			code, wire.CodeRoomFull)
	}
}

// TestARoomThatCouldNotFinishItsDraftIsRefusedWhenItOpens is the check
// Config.Validate cannot make: draft.Fits needs the pool, the pool needs the cast
// book, and Validate is a method on the configuration with no Deps.
//
// ⚠️ **Both halves are asserted, and the first is what makes this about New.**
// The configuration is *valid* — Validate says so by name — and New refuses it
// anyway, which is what "a room that could not finish its draft fails when it is
// opened rather than when somebody joins" means. The same configuration without
// Drafts opens fine, so the refusal is the draft's.
//
// The fixture is a cast book with **nobody in it** rather than a subset, and that
// is honesty rather than laziness: the shipped pool is nineteen characters and
// seats every format the game offers with room to spare — nine to spare at 3v3 and
// three at 5v5, measured — so a pool too small cannot be built out of the shipped
// data at all.
func TestARoomThatCouldNotFinishItsDraftIsRefusedWhenItOpens(t *testing.T) {
	dependencies := deps(t)
	dependencies.Characters = &cast.Book{}

	configuration := draftingConfig(11)
	if err := configuration.Validate(); err != nil {
		t.Fatalf("the configuration itself is refused (%v), so this measures Validate and not "+
			"New", err)
	}
	if _, err := room.New(configuration, dependencies); err == nil {
		t.Error("a room that drafts opened over a cast nobody can be picked out of, so its " +
			"draft would have run dry partway through somebody's ban and pick")
	} else {
		t.Logf("refused at New: %v", err)
	}

	// The same data without a draft is a perfectly ordinary room: what is being
	// refused is the draft and not the books.
	ordinary := configuration
	ordinary.Drafts = false
	if _, err := room.New(ordinary, dependencies); err != nil {
		t.Errorf("a room that does not draft was refused the same data: %v", err)
	}

	// And the shipped cast is comfortable at both formats, which is the other
	// half of "measured": a refusal nothing can trip is a refusal nobody has
	// checked the sign of.
	pool := draft.NewPool(deps(t).Characters.All())
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		if err := draft.Fits(pool.Len(), format); err != nil {
			t.Errorf("the shipped pool of %d cannot seat %s: %v", pool.Len(), format, err)
		}
	}
	t.Logf("the shipped pool holds %d: %d to spare at 3v3 and %d at 5v5", pool.Len(),
		draft.Slack(pool.Len(), wire.Format3v3), draft.Slack(pool.Len(), wire.Format5v5))
}

// TestADraftingRoomIsABo1 is Config.Validate's own refusal, and it is a design
// decision rather than a bounds check — the same kind as its refusal of a bo2.
//
// "A ban lasts the match, and the first cut is bo1 only" is what was settled. What
// a draft means across a *series* is three different games — three drafts, one
// draft carried across all three battles, or a draft a battle with the previous
// winner banning first — so a room that accepted the configuration would be
// choosing the second of them silently, which is exactly the decision that is
// deliberately a later item.
func TestADraftingRoomIsABo1(t *testing.T) {
	dependencies := deps(t)
	drafting := draftingConfig(11)
	drafting.Battles = 3
	if err := drafting.Validate(); err == nil {
		t.Error("a drafting bo3 was accepted, which picks one of the three games a draft " +
			"across a series could be without anybody deciding")
	}
	if _, err := room.New(drafting, dependencies); err == nil {
		t.Error("a drafting bo3 opened a room")
	}
	// The bo3 itself is untouched: what is refused is drafting one.
	plain := drafting
	plain.Drafts = false
	if err := plain.Validate(); err != nil {
		t.Errorf("an ordinary bo3 is refused: %v", err)
	}
}

// TestAPeerCannotCancelADraftByClaimingItsOwnClockRanOut is the one step
// draft.Draft.Apply routes that a client must not be able to reach.
//
// ⚠️ A wire.StepTimeout cancels the **whole room**, and Apply would happily route
// one — so a peer allowed to send it could close a draft it did not like by
// asserting its own allowance had run out. A timeout is the transport's input and
// never a peer's message.
func TestAPeerCannotCancelADraftByClaimingItsOwnClockRanOut(t *testing.T) {
	opened, _ := aDraftingRoom(t)

	claimed := wire.Decide{DraftDecision: wire.DraftDecision{Step: wire.StepTimeout}}
	answered, err := opened.Deliver(wire.SeatHost, claimed)
	if err != nil {
		t.Fatalf("a peer sends a timeout as a decision: %v", err)
	}
	if code := onlyCodeFor(t, answered, wire.SeatHost); code != wire.CodeIllegalAction {
		t.Errorf("a peer claiming its own clock ran out was answered %q, want %q",
			code, wire.CodeIllegalAction)
	}
	if opened.Finished() {
		t.Fatal("a peer closed the room by sending a timeout as its draft decision")
	}
	if onTurn, waiting := opened.Awaiting(); !waiting || onTurn != wire.SeatHost {
		t.Errorf("after the refusal the room is waiting on %q/%v, want the host's first ban "+
			"still open", onTurn, waiting)
	}
	// The seat nobody is asking gets the other refusal, which is the ordering the
	// room applies: not-your-turn is asked after the step, so a decision from the
	// wrong seat carrying a timeout is still a timeout being refused.
	answered, err = opened.Deliver(wire.SeatGuest, claimed)
	if err != nil {
		t.Fatalf("the seat off turn sends a timeout as a decision: %v", err)
	}
	if code := onlyCodeFor(t, answered, wire.SeatGuest); code != wire.CodeIllegalAction {
		t.Errorf("a timeout sent as a decision from the seat off turn was answered %q, want %q",
			code, wire.CodeIllegalAction)
	}
	if opened.Finished() {
		t.Fatal("the seat nobody was asking closed the room with a timeout decision")
	}
	// And a decision the draft has no step for is a legality refusal rather than
	// an error out of Deliver.
	unknown := wire.Decide{DraftDecision: wire.DraftDecision{Step: wire.DraftStep("khong-co")}}
	answered, err = opened.Deliver(wire.SeatHost, unknown)
	if err != nil {
		t.Fatalf("a decision naming a step this protocol has not: %v", err)
	}
	if code := onlyCodeFor(t, answered, wire.SeatHost); code != wire.CodeIllegalAction {
		t.Errorf("a decision naming an unknown step was answered %q, want %q",
			code, wire.CodeIllegalAction)
	}
}

// TestADecideOutsideADraftIsRefused is the other side of the routing: a room that
// does not draft, and a drafting room with one peer in it, both answer a
// wire.Decide with the closest true thing the ten codes have — nobody in them is
// being asked to decide.
//
// It is the same answer answerFrom gives a wire.Act that arrives while a draft is
// still running, which is the symmetry worth keeping: the room is not asking this
// peer for this.
//
// ⚠️ **The lone-peer case is the branch draftOpen's "both seats taken" clause
// exists for, and it is a real hazard rather than tidiness.** The draft is built
// in New — so that one this room could never finish is refused before a code is
// handed out — which means it exists, and is already due its first ban, while the
// room holds one player. Without that clause the host could spend its whole ban
// allowance before an opponent had arrived.
func TestADecideOutsideADraftIsRefused(t *testing.T) {
	t.Run("a drafting room with one peer in it", func(t *testing.T) {
		opened := newRoom(t, draftingConfig(11))
		seat, out, err := opened.Join(helloWithNoSquad(t, "Alone"))
		if err != nil || seat != wire.SeatHost {
			t.Fatalf("the host joins a drafting room: %q, %v, %v", seat, out, err)
		}
		if onTurn, waiting := opened.Awaiting(); waiting {
			t.Errorf("a drafting room with one player is waiting on %q; the second seat is "+
				"what opens the draft, not the draft's own construction", onTurn)
		}
		early := wire.Decide{DraftDecision: wire.DraftDecision{
			Step: wire.StepBan, Character: "pokemon.pichu",
		}}
		answered, err := opened.Deliver(wire.SeatHost, early)
		if err != nil {
			t.Fatalf("a ban before the opponent arrived: %v", err)
		}
		if code := onlyCodeFor(t, answered, wire.SeatHost); code != wire.CodeNotYourTurn {
			t.Errorf("a ban sent before an opponent arrived was answered %q, want %q",
				code, wire.CodeNotYourTurn)
		}
		// And the seat that arrives second finds a draft nobody has spent
		// anything out of: the host still owes the first ban.
		if _, out, err = opened.Join(helloWithNoSquad(t, "Guest")); err != nil {
			t.Fatalf("the guest joins: %v", err)
		}
		if onTurn, waiting := opened.Awaiting(); !waiting || onTurn != wire.SeatHost {
			t.Errorf("with both seats taken the room is waiting on %q/%v, want the host's "+
				"first ban", onTurn, waiting)
		}
	})

	dependencies := deps(t)
	configuration := config(11, 1)
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)
	for index, side := range [][]string{
		{"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly"},
		{"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"},
	} {
		squad := squadOf(t, dependencies.Characters, "phe-"+string(rune('a'+index)), side...)
		_, out, err := opened.Join(hello(t, squad, squad.ID))
		if err != nil {
			t.Fatalf("seat a peer: %v", err)
		}
		clients.deliver(t, out)
	}
	onTurn, waiting := opened.Awaiting()
	if !waiting {
		t.Fatal("a room with both seats taken is asking nobody")
	}

	deciding := wire.Decide{DraftDecision: wire.DraftDecision{
		Step: wire.StepBan, Character: "pokemon.pichu",
	}}
	answered, err := opened.Deliver(onTurn, deciding)
	if err != nil {
		t.Fatalf("a decision in a room that does not draft: %v", err)
	}
	if code := onlyCodeFor(t, answered, onTurn); code != wire.CodeNotYourTurn {
		t.Errorf("a room that does not draft answered a decision %q, want %q",
			code, wire.CodeNotYourTurn)
	}
	if _, stillWaiting := opened.Awaiting(); !stillWaiting {
		t.Error("a refused decision spent the open prompt")
	}
}

// otherOf is the seat that is not this one, for a test that has one and wants the
// other.
func otherOf(t *testing.T, seat wire.Seat) wire.Seat {
	t.Helper()
	switch seat {
	case wire.SeatHost:
		return wire.SeatGuest
	case wire.SeatGuest:
		return wire.SeatHost
	}
	t.Fatalf("%q is not one of the two seats a room hands out", seat)
	return ""
}

// firstCellsAreLegal is the fixture's own claim rather than the room's, and it is
// here because every arrangement above is built by firstCells: if those cells were
// not a legal formation every test in this file would fail for a reason that has
// nothing to do with the room.
func TestFirstCellsAreALegalArrangement(t *testing.T) {
	for _, units := range []int{
		draft.PicksPerSide(wire.Format3v3), draft.PicksPerSide(wire.Format5v5),
	} {
		cells := firstCells(units)
		if len(cells) != units {
			t.Fatalf("firstCells(%d) named %d cells", units, len(cells))
		}
		seen := map[hex.Offset]bool{}
		for _, cell := range cells {
			if seen[cell] {
				t.Errorf("firstCells(%d) puts two units on %v", units, cell)
			}
			seen[cell] = true
			if cell.Col < 0 || cell.Col >= hex.FormationCols ||
				cell.Row < 0 || cell.Row >= hex.FormationRows {
				t.Errorf("firstCells(%d) names %v, which is off a %dx%d formation",
					units, cell, hex.FormationCols, hex.FormationRows)
			}
		}
	}
}
