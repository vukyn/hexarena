package socket

import (
	"context"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTwoRealClientsFightAWholeBo3OverALoopbackListener is this package's gate,
// and it is the design record's own item: *one end-to-end test over a loopback
// listener, two real clients*.
//
// ⚠️ It asserts something real rather than "it finished", which is the failure
// mode an end-to-end test of this shape falls into. Four claims, each of which
// could be quietly wrong while a match still ran to completion:
//
//  1. **Each client was told its own seat and its own side.** Two facts and two
//     messages: the seat arrives once on wire.Welcome and holds for the match,
//     the side arrives on every wire.Start and *changes between battles*,
//     because a match is fought both ways round. A transport that handed one
//     client the other's side would draw the wrong half of the board from the
//     second battle on, and a mirror board could not see it — which is why the
//     two squads are different characters.
//  2. **The per-turn digests agreed on every turn.** That is the mirror's whole
//     promise: each client hashed the events its own engine produced and
//     compared them against the room's, and Mirror.Compared counts how many
//     times, so a run that checked a handful is a run that fought no match.
//  3. **The verdict is the one the board produced**, re-derived here from what
//     each client's own engine settled rather than read off the room's word for
//     it — the client learns each battle's outcome from its own Ended event and
//     there is deliberately no series-standing message.
//  4. **Nothing was reported.** The transport's error sink is its only output,
//     so an empty sink over a whole match is the claim that no write failed, no
//     message went to a seat with no connection, and no room refused an input.
func TestTwoRealClientsFightAWholeBo3OverALoopbackListener(t *testing.T) {
	dependencies := deps(t)
	configuration := config(11, 3, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	// The code carries the address, so this is the whole of what a player is
	// given — and it is the address the loopback listener is really on.
	at, err := code.AddrPort()
	if err != nil {
		t.Fatalf("the code the registry handed back does not decode: %v", err)
	}
	if at != held.at {
		t.Errorf("the code names %s and the listener is at %s", at, held.at)
	}

	host := held.dial(t, code, hello(t, theHostSquad(t, dependencies.Characters), "Host", ""), dependencies.Books)
	if host.Seat() != wire.SeatHost {
		t.Fatalf("the first client took the %q seat, want the host's", host.Seat())
	}
	// The host has to be reading before the second join, because that join is
	// what starts the match and its answer carries a wire.Start for both seats.
	ctx := context.Background()
	hostPlay := play(ctx, host, rating(host))

	guest := held.dial(t, code, hello(t, theGuestSquad(t, dependencies.Characters), "Guest", ""), dependencies.Books)
	if guest.Seat() != wire.SeatGuest {
		t.Fatalf("the second client took the %q seat, want the guest's", guest.Seat())
	}
	guestPlay := play(ctx, guest, rating(guest))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's match: %v", err)
	}
	done := held.finished(t)
	if done.code != code {
		t.Errorf("a match finished in room %s and this test opened %s", done.code, code)
	}
	result, played := done.reading.Result, done.reading.Played
	if !result.Verdict.Over() {
		t.Fatalf("the transport reported a finished match with the verdict %q", result.Verdict)
	}
	if len(played) == 0 {
		t.Fatal("the match finished having played no battles")
	}

	// Claim 1, the seat: it is on the welcome, it is this client's own, and it
	// does not change.
	for _, client := range []*Client{host, guest} {
		welcome, seated := client.Mirror().Welcome()
		if !seated {
			t.Errorf("%s played a whole match without a welcome", client.Seat())
			continue
		}
		if welcome.Seat != client.Seat() {
			t.Errorf("%s holds a welcome naming the %q seat", client.Seat(), welcome.Seat)
		}
		if welcome.Battles != configuration.Battles || welcome.Allowance != configuration.Allowance ||
			welcome.TurnCap != configuration.TurnCap || welcome.Format != configuration.Format {
			t.Errorf("%s was told a configuration that is not the room's: %+v", client.Seat(), welcome)
		}
	}

	// Claim 1, the side: each client played the half the room's own alternation
	// gives its seat, per battle, and the two clients were always on opposite
	// halves. The alternation is HomeFor's, and home is the sixty-point fact —
	// the home squad is enlisted first and therefore wins a speed tie.
	hostFought, guestFought := host.Mirror().Fought(), guest.Mirror().Fought()
	if len(hostFought) != len(played) || len(guestFought) != len(played) {
		t.Fatalf("the room played %d battles and the clients settled %d and %d",
			len(played), len(hostFought), len(guestFought))
	}
	for index := range played {
		home := configuration.HomeFor(index + 1)
		wantHost, wantGuest := hex.SideEnemy, hex.SideEnemy
		if home == wire.SeatHost {
			wantHost = hex.SideAlly
		} else {
			wantGuest = hex.SideAlly
		}
		if hostFought[index].Side != wantHost {
			t.Errorf("the host played %s, want the %s side (home is %s)",
				side(hostFought[index]), wantHost, home)
		}
		if guestFought[index].Side != wantGuest {
			t.Errorf("the guest played %s, want the %s side (home is %s)",
				side(guestFought[index]), wantGuest, home)
		}
		if played[index].Home != home {
			t.Errorf("battle %d was home to %s, want %s", index+1, played[index].Home, home)
		}
		if hostFought[index].Seed != played[index].Seed {
			t.Errorf("the host fought battle %d from seed %d and the room recorded %d",
				index+1, hostFought[index].Seed, played[index].Seed)
		}
	}
	// And the sides genuinely swapped at least once, or the claim above was
	// satisfied by a bo1 wearing a bo3's name.
	if len(hostFought) > 1 && hostFought[0].Side == hostFought[1].Side {
		t.Errorf("the host played the %s side in both of the first two battles; a match is fought both ways round",
			hostFought[0].Side)
	}

	// Claim 2: every turn was checked, by both clients, and there were enough of
	// them to have been a match. A bo3 of the shipped 3v3 takes 34 to 55
	// decisions a battle, and every turn goes to both clients — including the one
	// the client itself asked for — so the two counts are equal.
	if host.Mirror().Compared() != guest.Mirror().Compared() {
		t.Errorf("the host checked %d digests and the guest %d; every turn goes to both",
			host.Mirror().Compared(), guest.Mirror().Compared())
	}
	if compared := host.Mirror().Compared(); compared < 30 {
		t.Errorf("each client checked only %d digests, which is too few for a real match", compared)
	}

	// Claim 3: the verdict, re-derived from what the clients' own engines
	// settled. Each client knows which side it played per battle, so it knows
	// which battles it won without anything telling it.
	hostWon, guestWon := 0, 0
	for index := range hostFought {
		if hostFought[index].Mine() {
			hostWon++
		}
		if guestFought[index].Mine() {
			guestWon++
		}
		// The two clients' readings of one battle have to agree about the
		// outcome, since they are the same battle.
		if hostFought[index].Outcome != guestFought[index].Outcome {
			t.Errorf("battle %d ended %q for the host and %q for the guest",
				index+1, hostFought[index].Outcome, guestFought[index].Outcome)
		}
		if hostFought[index].Mine() && guestFought[index].Mine() {
			t.Errorf("both clients believe they won battle %d", index+1)
		}
	}
	switch result.Verdict {
	case room.VerdictWon:
		want := hostWon > guestWon
		if got := result.Winner == wire.SeatHost; got != want {
			t.Errorf("the room made %s the winner while the clients settled %d-%d to the host",
				result.Winner, hostWon, guestWon)
		}
	case room.VerdictDrawn:
		if hostWon != guestWon {
			t.Errorf("the room drew the match while the clients settled %d-%d", hostWon, guestWon)
		}
	default:
		t.Errorf("a match played out over a socket has the verdict %q", result.Verdict)
	}
	if result.Departed.Valid() {
		t.Errorf("a match played out to its end records %q as having gone away", result.Departed)
	}
	// The room's own standing and the clients' own count of their wins are two
	// independent readings of the series and have to agree.
	if want := result.Wins; want[0] != hostWon || want[1] != guestWon {
		t.Errorf("the room's standing is %v and the clients settled %d-%d", want, hostWon, guestWon)
	}

	// Claim 4: nothing was reported, and no client was refused anything.
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors over a whole match: %q", len(said), said)
	}
	for _, client := range []*Client{host, guest} {
		if refusals := client.Mirror().Refusals(); len(refusals) != 0 {
			t.Errorf("%s was refused %v over a match nothing went wrong in", client.Seat(), refusals)
		}
		if closure, closed := client.Mirror().Closure(); closed {
			t.Errorf("%s was told the match closed (%s); a match played out to its end sends nothing",
				client.Seat(), closure)
		}
	}
	// The transport lets its tables go with the connections, so a finished match
	// leaves the server holding nothing — the same claim room.Registry.Wait
	// makes about goroutines.
	held.emptied(t)

	t.Logf("bo%d over %d battles, %d turns checked by each client, standing %d-%d, verdict %q",
		configuration.Battles, len(played), host.Mirror().Compared(), hostWon, guestWon, result.Verdict)
}

// TestTheLargestStartFitsTheMessageLimit is what DefaultMessageLimit was set
// from, and it is the test that corrected the constant.
//
// wire.Start carries the **whole resolved roster**, because that is what a
// mirror builds its own battle from, so it is the one message in this protocol
// whose size is worth asking about. The guess was that a 5v5 would approach
// coder/websocket's own 32 KiB default; it comes to under three thousand bytes,
// and the constant moved from a megabyte to 64 KiB on that reading.
//
// ⚠️ It measures a **5v5**, which is hex.MaxTeamSize a side and the largest a
// legal room can produce, and it needs no room and no socket: the bytes are a
// property of the message.
//
// Both ends are held. A limit under what the protocol produces is a match that
// cannot start; a limit orders of magnitude over it is allocation a peer can ask
// for and nothing needs.
func TestTheLargestStartFitsTheMessageLimit(t *testing.T) {
	dependencies := deps(t)
	ally := squadOn(t, dependencies.Characters, theFiveSlots, "the.largest.ally",
		"pokemon.bulbasaur", "pokemon.charmander", "pokemon.squirtle", "pokemon.machop", "pokemon.mewtwo")
	enemy := squadOn(t, dependencies.Characters, theFiveSlots, "the.largest.enemy",
		"pokemon.gastly", "pokemon.cleffa", "pokemon.magnemite", "pokemon.poliwag", "pokemon.mew")
	roster, err := ally.Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the ally squad: %v", err)
	}
	facing, err := enemy.Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the enemy squad: %v", err)
	}
	roster = append(roster, facing...)
	if want := 2 * wire.Format5v5.Units(); len(roster) != want {
		t.Fatalf("the largest roster is %d units, want %d", len(roster), want)
	}
	raw, err := wire.Encode(wire.Start{Seed: 11, Roster: roster, Side: hex.SideAlly, Battle: 1})
	if err != nil {
		t.Fatalf("encode the largest start: %v", err)
	}
	largest := int64(len(raw))
	// The floor: a limit under what a room sends is a match nobody can start,
	// and four times over is the headroom a wider format or a longer id wants.
	if DefaultMessageLimit < 4*largest {
		t.Errorf("the largest start is %d bytes and the limit is %d, which leaves no headroom",
			largest, DefaultMessageLimit)
	}
	// The ceiling, which is the half the first version of this constant failed:
	// a megabyte was 360 times the largest message the protocol produces, which
	// is allocation a peer can ask for and nothing needs.
	if DefaultMessageLimit > 64*largest {
		t.Errorf("the largest start is %d bytes and the limit is %d, which is %d times more than anything needs",
			largest, DefaultMessageLimit, DefaultMessageLimit/largest)
	}
	t.Logf("the largest start a room can send is %d bytes over %d units; the limit is %d (%dx)",
		largest, len(roster), DefaultMessageLimit, DefaultMessageLimit/largest)
}
