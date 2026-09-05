package room_test

import (
	"reflect"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTwoFakeClientsDraftAndFightAWholeMatchInProcess is the test this step
// exists for, and it is TestTwoFakeClientsFightAWholeBo3InProcess with a draft in
// front of it: no sockets, no goroutines and no clock, at the speed of the
// engine. That test is the reason internal/room was built with no I/O; a drafting
// room that could not be driven the same way would be the least-tested code in
// the repository.
//
// ⚠️ **Both clients compute their own draft and step it through
// draft.Draft.Apply**, out of the entries on wire.Drafted and out of nothing
// else — never out of what they themselves sent. So the mirror is proven end to
// end rather than at one end: what the room recorded and what a client computed
// have to agree, and the handover is asserted **by value** — the two wire.Start
// rosters are the drafted squads, resolved, and the two clients then fight that
// battle to its finish.
//
// Six claims, each of them a thing that could be quietly wrong while a drafting
// match still ran to completion:
//
//  1. **Whose decision is due is agreed by two independent readings** — the
//     room's own draft and a draft each client computed from the record alone.
//     Asserted inside mirror.decide, so it fires on every decision of the draft,
//     and its sharpest form is the very first one: nothing on the wire says the
//     host bans first, so a client that computed the other seat is refused
//     immediately.
//  2. **The room's record and each client's replay are the same draft**, compared
//     as the two squads they produce rather than as a list of decisions.
//  3. **The rosters the battle was opened on are those squads, resolved**, home
//     enlisted first — which is the sixty-point fact, since roster order decides
//     a speed tie.
//  4. **Nothing downstream of the draft changed**: the battle then plays out
//     exactly as the undrafted one does, digest against digest on every turn.
//  5. **A drafted side cannot double up**, which is a property of the shared
//     exclusive pool rather than of any refusal — and it is the scope
//     squadIsFieldable's note is about.
//  6. **The picks are the pool's, and the bans are gone from it.**
func TestTwoFakeClientsDraftAndFightAWholeMatchInProcess(t *testing.T) {
	dependencies := deps(t)
	configuration := draftingConfig(11)
	opened := newRoom(t, configuration)
	// A replay limit generous enough for one decision plus whatever run of
	// skipped turns follows it. It is the client's own number: nothing on the wire
	// carries it.
	clients := newTable(t, dependencies, configuration.TurnCap)

	// ⚠️ Neither peer brings a squad, because the room drafts: the side it will
	// field is the side it is about to draft, and a hello that brought one is
	// refused with wire.CodeSquadUnwanted (→
	// TestADraftingRoomRefusesASquadAsUnwanted).
	seat, out, err := opened.Join(helloWithNoSquad(t, "Host"))
	if err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	if seat != wire.SeatHost {
		t.Fatalf("the first peer took the %q seat, want the host's", seat)
	}
	if len(out) != 1 {
		t.Fatalf("the first join produced %d messages, want only a welcome", len(out))
	}
	clients.deliver(t, out)
	if _, waiting := opened.Awaiting(); waiting {
		t.Error("a room with one player is waiting on a seat to decide")
	}

	seat, out, err = opened.Join(helloWithNoSquad(t, "Guest"))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	if seat != wire.SeatGuest {
		t.Fatalf("the second peer took the %q seat, want the guest's", seat)
	}
	// ⚠️ **Still one message**: opening a draft announces nothing at all, because
	// nothing has been decided and a room must not send a wire.Drafted carrying
	// none. And no wire.Start — the battle is eighteen decisions away.
	if len(out) != 1 {
		t.Fatalf("the second join produced %d messages, want only a welcome: a draft opens "+
			"with nothing to say", len(out))
	}
	clients.deliver(t, out)

	// Both clients were told the room drafts, which is the whole of what lets them
	// compute a draft at all — compared against the room's own reading of its
	// configuration rather than against the local value, for the reason the
	// allowance is.
	held := opened.Config()
	if held != configuration {
		t.Errorf("the room holds a configuration that is not the one it was opened with")
	}
	for _, client := range []*mirror{clients.host, clients.guest} {
		if !client.welcome.Drafts {
			t.Fatalf("%s was not told the room drafts, so it has no draft to mirror",
				client.seat)
		}
		if client.welcome.Drafts != held.Drafts {
			t.Errorf("%s was told drafts=%v and the room runs under %v",
				client.seat, client.welcome.Drafts, held.Drafts)
		}
		if client.drafting == nil {
			t.Fatalf("%s built no draft of its own", client.seat)
		}
	}

	// ⚠️ Claim 1's sharpest form: the host decides first and **nothing on the wire
	// said so**. Both peers derived it from Welcome.Drafts and Welcome.Seat.
	if onTurn, waiting := opened.Awaiting(); !waiting || onTurn != wire.SeatHost {
		t.Fatalf("the draft opened waiting on %q/%v, want the host", onTurn, waiting)
	}

	// The draft. Every decision is read off the deciding client's own draft, so
	// the whole thing is deterministic and nobody types.
	decisions := 0
	for {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d decisions the draft is waiting on nobody and no battle has "+
				"started", decisions)
		}
		answered, err := opened.Deliver(onTurn, clients.at(onTurn).decide())
		if err != nil {
			t.Fatalf("draft decision %d from %s: %v", decisions, onTurn, err)
		}
		clients.deliver(t, answered)
		decisions++
		if starts := startsIn(answered); starts > 0 {
			if starts != seatsInARoom {
				t.Fatalf("the draft finished and started %d battles, want one wire.Start a seat",
					starts)
			}
			break
		}
		// A backstop rather than an expectation: without it a state machine that
		// stopped progressing would hang the suite instead of failing it.
		if decisions > draftDecisions(wire.Format3v3) {
			t.Fatalf("the draft took more than %d decisions, so something is not progressing",
				decisions)
		}
	}
	records := len(clients.host.records)

	// The decision count is derived arithmetic and not a magic number: every ban
	// slot is one decision, every pick is two, and each side arranges once.
	if want := draftDecisions(wire.Format3v3); decisions != want {
		t.Errorf("the draft took %d decisions, want %d — %d ban slots, %d picks with a loadout "+
			"each, and one arrangement a side", decisions, want,
			2*draft.BansPerSide(wire.Format3v3), 2*draft.PicksPerSide(wire.Format3v3))
	}
	// ⚠️ Exactly one decision was answered with no message: the first arrangement,
	// which records nothing because an entry is public the moment it is sent.
	if want := decisions - 1; records != want {
		t.Errorf("the draft produced %d messages over %d decisions, want %d", records,
			decisions, want)
	}

	// Claim 2: the record and each client's replay are the same draft, compared as
	// the squads they produce. This is the mirror end to end — the room never sent
	// a squad, a pool or a turn, only the decisions.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if !client.drafting.Done() {
			t.Fatalf("%s's own draft is not finished after replaying %d entries",
				client.seat, client.applied)
		}
		if client.applied != decisions {
			t.Errorf("%s replayed %d entries over %d decisions", client.seat,
				client.applied, decisions)
		}
	}
	drafted := clients.host.drafting.Squads()
	if guests := clients.guest.drafting.Squads(); !reflect.DeepEqual(drafted, guests) {
		t.Fatal("the two clients replayed the same record into different squads, so a draft " +
			"is not a pure function of the decisions taken")
	}

	// Claim 3, by value: the roster both clients were started on is exactly the
	// two drafted squads resolved, home first.
	if len(clients.host.starts) != 1 {
		t.Fatalf("the host was started %d times in a bo1", len(clients.host.starts))
	}
	start := clients.host.starts[0]
	homeIndex := seatIndexOf(t, homeOf(t, clients))
	awayIndex := seatIndexOf(t, otherOf(t, homeOf(t, clients)))
	wanted, err := drafted[homeIndex].Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the drafted home squad: %v", err)
	}
	facing, err := drafted[awayIndex].Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the drafted away squad: %v", err)
	}
	wanted = append(wanted, facing...)
	if !reflect.DeepEqual(start.Roster, wanted) {
		t.Errorf("the battle was opened on a roster that is not the drafted pair resolved:\n"+
			"got  %v\nwant %v", rosterIDs(start.Roster), rosterIDs(wanted))
	}
	if !reflect.DeepEqual(clients.guest.starts[0].Roster, start.Roster) {
		t.Error("the two clients were started on different rosters")
	}

	// Claims 5 and 6: the pool is exclusive, so every unit on the board is a
	// different character — including across the two sides, since both spend out
	// of one pool — and nobody banned is on it.
	characters := map[string]int{}
	for _, side := range drafted {
		for _, unit := range side.Units {
			characters[unit.Character]++
		}
	}
	units := 2 * draft.PicksPerSide(wire.Format3v3)
	if len(characters) != units {
		t.Errorf("%d units were drafted out of %d distinct characters: the pool is exclusive, "+
			"so a drafted side cannot field the same character twice and neither can the pair "+
			"of them", units, len(characters))
	}
	pool := draft.NewPool(dependencies.Characters.All())
	banned := map[string]bool{}
	for _, entry := range clients.host.decided {
		if entry.Step == wire.StepBan && entry.Character != "" {
			banned[entry.Character] = true
		}
	}
	if len(banned) != 2*draft.BansPerSide(wire.Format3v3) {
		t.Errorf("the record holds %d bans naming somebody, want %d: this fixture spends "+
			"every slot, so a skip here would make the exclusion below measure less",
			len(banned), 2*draft.BansPerSide(wire.Format3v3))
	}
	for id := range banned {
		if !pool.Has(id) {
			t.Errorf("%q was banned and is not in the pool at all", id)
		}
		if characters[id] > 0 {
			t.Errorf("%q was banned and is on the board", id)
		}
	}

	// Claim 4: the battle, which is unchanged. This loop is
	// TestTwoFakeClientsFightAWholeBo3InProcess's loop, and that is the point of
	// it — nothing downstream of the draft knows a draft happened.
	steps := 0
	for !opened.Finished() {
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d turns the room is waiting on nobody and the match is not over",
				steps)
		}
		client := clients.at(onTurn)
		answered, err := opened.Deliver(onTurn, client.answer())
		if err != nil {
			t.Fatalf("turn %d from %s: %v", steps, onTurn, err)
		}
		clients.deliver(t, answered)
		steps++
		if steps > configuration.TurnCap {
			t.Fatalf("the battle took more than %d decisions, so something is not progressing",
				steps)
		}
	}

	played := opened.Played()
	if len(played) != 1 {
		t.Fatalf("a bo1 played %d battles", len(played))
	}
	one := played[0]
	if one.Turns <= 0 {
		t.Fatalf("the battle recorded %d turns", one.Turns)
	}
	// ⚠️ The vacuity guard that makes "it finished" mean something. A capped
	// battle is a hang detector firing rather than an ending: the engine concluded
	// nothing about it, so a run that reached the cap fought nobody to a finish
	// and every claim below it is about a battle that did not happen.
	if one.Capped {
		t.Fatalf("the battle hit the %d-turn cap, so this measures a battle that was stopped "+
			"rather than one that ended", configuration.TurnCap)
	}
	result := opened.Result()
	switch result.Verdict {
	case room.VerdictWon:
		if !result.Winner.Valid() || result.Winner != one.Winner {
			t.Errorf("the room declared %q the winner of a battle %q won",
				result.Winner, one.Winner)
		}
	case room.VerdictDrawn:
		if result.Winner.Valid() || one.Winner.Valid() {
			t.Errorf("a drawn match names %q, off a battle %q won", result.Winner, one.Winner)
		}
	default:
		t.Errorf("a drafted match played out to the end has the verdict %q", result.Verdict)
	}
	if result.Departed.Valid() {
		t.Errorf("a match played out to the end records %q as having gone away", result.Departed)
	}
	if one.Seed != configuration.SeedFor(1) {
		t.Errorf("the battle was fought from seed %d, want %d derived from the room's %d",
			one.Seed, configuration.SeedFor(1), configuration.Seed)
	}

	// Claim 4's own vacuity guard, the one the undrafted test uses: a 3v3 of the
	// shipped cast takes 34 to 55 decisions, so a run that compared a handful of
	// digests did not fight a battle and the check inside mirror.apply measured
	// almost nothing.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if client.compared != steps {
			t.Errorf("%s checked %d digests over %d decisions; every turn goes to both clients",
				client.seat, client.compared, steps)
		}
		if client.compared < 30 {
			t.Errorf("%s checked only %d digests, which is too few for a real battle",
				client.seat, client.compared)
		}
	}

	// And both mirrors of that battle are the same battle, which is what the whole
	// design rests on: same seed, same drafted roster, same events.
	hostDigest, err := wire.DigestEvents(clients.host.events)
	if err != nil {
		t.Fatalf("digest the host's events: %v", err)
	}
	guestDigest, err := wire.DigestEvents(clients.guest.events)
	if err != nil {
		t.Fatalf("digest the guest's events: %v", err)
	}
	if hostDigest != guestDigest {
		t.Errorf("the two clients' mirrors of the drafted battle differ: %s against %s",
			hostDigest.Short(), guestDigest.Short())
	}
	closing := lastEvent(t, clients.host.events, battle.Ended)
	if closing.Outcome != one.Outcome {
		t.Errorf("the host's own battle ended %q while the room recorded %q",
			closing.Outcome, one.Outcome)
	}

	t.Logf("a %s draft in %d decisions (%d records); %s drafted %v against %s's %v; "+
		"home %s, seed %d, %d turns, outcome %q, verdict %q, winner %q, %d prompts skipped",
		configuration.Format, decisions, records,
		wire.SeatHost, rosterIDs(nil, drafted[0].Units...),
		wire.SeatGuest, rosterIDs(nil, drafted[1].Units...),
		one.Home, one.Seed, one.Turns, one.Outcome, result.Verdict, result.Winner,
		opened.Skipped())
}

// homeOf is the seat the battle was fought home from, read off the side each
// client was given on its own wire.Start rather than off Config.HomeFor — a bo1's
// home is picked by the low bit of the derived seed, and asking the rule what the
// rule says would agree with any rule at all.
func homeOf(t *testing.T, clients *table) wire.Seat {
	t.Helper()
	for _, client := range []*mirror{clients.host, clients.guest} {
		if len(client.starts) == 0 {
			t.Fatalf("%s was never started, so nothing says which side it played", client.seat)
		}
		if client.starts[0].Side == hex.SideAlly {
			return client.seat
		}
	}
	t.Fatal("neither client was started on the ally side, so no seat was enlisted first")
	return ""
}

// rosterIDs is a roster or a squad as the ids it holds, so a mismatch reads as
// two lists of names rather than as two pages of resolved stat lines.
func rosterIDs(roster []battle.Roster, units ...placement.Placement) []string {
	out := make([]string, 0, len(roster)+len(units))
	for _, one := range roster {
		out = append(out, one.ID)
	}
	for _, one := range units {
		out = append(out, one.Character)
	}
	return out
}
