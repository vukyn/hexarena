package room_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTheGateRefusesEachThingForItsOwnCode is one case per code the gate can
// send, and each case is wrong about exactly one thing.
//
// One code per fault is the whole value of a code: the client holds the same
// books and both language books, so it can word "your squad is illegal" far
// better than the server could — but only if it is told which of the five it
// was. A gate that answered CodeSquadRefused to a bad password would be a client
// telling a player to fix a squad that was fine.
func TestTheGateRefusesEachThingForItsOwnCode(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")

	// Every case builds its own room, because a refusal must leave the room
	// exactly as open as it found it and a shared one would hide that.
	for _, one := range []struct {
		name string
		// password is the room's, empty for a room with none.
		password wire.Password
		// before is a peer that joins first, for the cases that need the room in
		// some state.
		before []wire.Hello
		hello  func(*testing.T) wire.Hello
		want   wire.Code
	}{
		{
			name: "a peer that cannot speak the protocol",
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Ahead")
				out.Protocol = wire.Protocol + 1
				return out
			},
			want: wire.CodeProtocolMismatch,
		},
		{
			name: "a peer whose data would not simulate the same battle",
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Edited")
				out.Data = mangled(out.Data)
				return out
			},
			want: wire.CodeDataMismatch,
		},
		{
			name:     "the room's password typed wrong",
			password: fixturePassword,
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Stranger")
				out.Password = fixturePassword + "!"
				return out
			},
			want: wire.CodeBadPassword,
		},
		{
			name:  "a third client with both seats taken",
			hello: func(t *testing.T) wire.Hello { return hello(t, legal, "Third") },
			before: []wire.Hello{
				hello(t, legal, "Host"),
				hello(t, squadOf(t, dependencies.Characters, "other.squad",
					"pokemon.charmander", "pokemon.squirtle", "pokemon.cleffa"), "Guest"),
			},
			want: wire.CodeRoomFull,
		},
		{
			name: "a squad of the wrong size for the format",
			hello: func(t *testing.T) wire.Hello {
				return hello(t, squadOf(t, dependencies.Characters, "pair.squad",
					"pokemon.bulbasaur", "pokemon.machop"), "Two")
			},
			want: wire.CodeSquadRefused,
		},
		{
			name: "a squad brought under the level cap",
			hello: func(t *testing.T) wire.Hello {
				squad := legal.Clone()
				squad.Units[1] = placeUnit(t, dependencies.Characters,
					"pokemon.machop", theThreeSlots[1], progression.LevelCap-1)
				return hello(t, squad, "Young")
			},
			want: wire.CodeSquadRefused,
		},
		{
			name: "a squad naming a character nobody has heard of",
			hello: func(t *testing.T) wire.Hello {
				squad := legal.Clone()
				squad.Units[0].Character = "pokemon.nonesuch"
				return hello(t, squad, "Invented")
			},
			want: wire.CodeSquadRefused,
		},
		{
			name: "a squad bringing a skill its unit has not learned",
			hello: func(t *testing.T) wire.Hello {
				squad := legal.Clone()
				squad.Units[0].Skills = []string{"hydro_pump", "hydro_pump", "hydro_pump", "hydro_pump"}
				return hello(t, squad, "Wishful")
			},
			want: wire.CodeSquadRefused,
		},
		{
			name: "two units of one squad standing on one cell",
			hello: func(t *testing.T) wire.Hello {
				squad := legal.Clone()
				squad.Units[1].Slot = squad.Units[0].Slot
				return hello(t, squad, "Stacked")
			},
			want: wire.CodeSquadRefused,
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			configuration := config(7, 1)
			configuration.Password = one.password
			opened := newRoom(t, configuration)
			for _, first := range one.before {
				if _, _, err := opened.Join(first); err != nil {
					t.Fatalf("seating a peer before the case: %v", err)
				}
			}
			seat, out, err := opened.Join(one.hello(t))
			if err != nil {
				t.Fatalf("join: %v", err)
			}
			if seat.Valid() {
				t.Errorf("a refused peer was given the %q seat", seat)
			}
			if got := onlyCode(t, out); got != one.want {
				t.Errorf("the gate answered %q, want %q", got, one.want)
			}
		})
	}
}

// TestTheGateRefusesInItsOwnOrder is the sharper half, and the reason it is a
// test of its own: a gate whose order is untested is a gate that reports
// whichever fault it happened to notice first.
//
// Each case is a peer wrong about **two** things, and the answer must be the
// earlier check's. Nothing about the code returned tells you the order was
// respected on its own — the same code comes back whether the gate checked in
// order or got lucky — so the pairs are what pins it, and the pairs are chosen
// to be adjacent so that the whole chain is covered link by link.
//
// ⚠️ The one link not restated here is protocol-before-digest inside
// wire.Version.Check, which internal/wire already pins with a peer wrong about
// both. It is asserted through the room anyway, because the room is what calls
// Check and a gate that read the two numbers itself would be a second reading of
// that order.
func TestTheGateRefusesInItsOwnOrder(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	illegal := legal.Clone()
	illegal.Units = illegal.Units[:2]

	for _, one := range []struct {
		name  string
		fill  bool
		hello func(*testing.T) wire.Hello
		want  wire.Code
		over  wire.Code
	}{
		{
			name: "protocol before data",
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Both")
				out.Protocol = wire.Protocol + 1
				out.Data = mangled(out.Data)
				return out
			},
			want: wire.CodeProtocolMismatch, over: wire.CodeDataMismatch,
		},
		{
			name: "data before the password",
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Both")
				out.Data = mangled(out.Data)
				out.Password = fixturePassword + "!"
				return out
			},
			want: wire.CodeDataMismatch, over: wire.CodeBadPassword,
		},
		{
			name: "the password before the seat",
			fill: true,
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, legal, "Both")
				out.Password = fixturePassword + "!"
				return out
			},
			want: wire.CodeBadPassword, over: wire.CodeRoomFull,
		},
		{
			name: "the seat before the squad",
			fill: true,
			hello: func(t *testing.T) wire.Hello {
				out := hello(t, illegal, "Both")
				out.Password = fixturePassword
				return out
			},
			want: wire.CodeRoomFull, over: wire.CodeSquadRefused,
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			configuration := config(7, 1)
			configuration.Password = fixturePassword
			opened := newRoom(t, configuration)
			if one.fill {
				for index, squad := range []placement.Squad{legal, legal} {
					seating := hello(t, squad, "Seated")
					seating.Password = fixturePassword
					// Two identical squads is a legal room: a match is fought
					// both ways round precisely because a mirror is the case
					// that reads worst.
					seating.Squad.ID = seating.Squad.ID + string(rune('a'+index))
					if _, _, err := opened.Join(seating); err != nil {
						t.Fatalf("filling the room: %v", err)
					}
				}
			}
			_, out, err := opened.Join(one.hello(t))
			if err != nil {
				t.Fatalf("join: %v", err)
			}
			got := onlyCode(t, out)
			if got == one.over {
				t.Fatalf("the gate answered %q, which is the LATER of the two faults: %q is checked first",
					got, one.want)
			}
			if got != one.want {
				t.Errorf("the gate answered %q, want %q", got, one.want)
			}
		})
	}
}

// TestOnlyALeafOfTheLineMayTakeTheField is the leaf rule at the gate, measured
// on **poliwag, which is the shipped fork**.
//
// ⚠️ This rule's interesting half was expected to be a latent branch, on the
// grounds recorded in CLAUDE.md that nothing shipped forks yet. That is now
// **stale**: poliwag ships as `Poliwag → Poliwhirl → (Poliwrath | Politoed)`, so
// both arms and an interior stage of a real fork are all reachable here. The
// forking *fixture* line is still measured, in
// TestALeafIsAFactAboutTheLineRatherThanTheLevel over in progression, because a
// fixture is where the rule can be pushed at a level nothing ships.
//
// ⚠️ **A stage above the cap does NOT tell IsLeaf apart from Furthest**, which
// an earlier draft of this comment claimed. `progression.Line.Validate` refuses
// a stage whose MinLevel is past the cap, so no legal line has one, and at the
// cap `Furthest(LevelCap)` is the tip of each arm by construction. Measured:
// substituting it inside `IsLeaf` reddens nothing here or in progression. What
// the predicate buys is the level that is no longer in the question — see the
// doc comment on `Line.Leaves`.
func TestOnlyALeafOfTheLineMayTakeTheField(t *testing.T) {
	dependencies := deps(t)
	const forking = "pokemon.poliwag"
	character, known := dependencies.Characters.Get(forking)
	if !known {
		t.Fatalf("no character is called %q", forking)
	}
	leaves, err := character.Stages.Leaves()
	if err != nil {
		t.Fatalf("the tips of %q: %v", forking, err)
	}
	// The vacuity guard, and it is the whole reason this test can measure the
	// interesting half at all: a line with one tip would let every case below
	// pass with the rule written as "the furthest form".
	if len(leaves) < 2 {
		t.Fatalf("%q has %d tip(s), so this test is not measuring a fork; the fixture line in "+
			"internal/core/progression is where the rule is held if this character stops forking",
			forking, len(leaves))
	}

	for _, one := range []struct {
		name  string
		stage string
		want  bool
	}{
		{name: "the first arm", stage: leaves[0].Name, want: true},
		{name: "the second arm", stage: leaves[1].Name, want: true},
		{name: "the form both arms grow out of", stage: "Poliwhirl", want: false},
		{name: "the root of the line", stage: "Poliwag", want: false},
		// Absent means "the furthest the level reaches", and on a line that
		// forks there is no such thing — so this is refused by Resolve's own
		// "name the one being fielded" rather than by a rule the gate writes.
		{name: "no stage named at all", stage: progression.Furthest, want: false},
	} {
		t.Run(one.name, func(t *testing.T) {
			squad := placement.Squad{ID: "fork.squad", Units: []placement.Placement{
				stagedUnit(t, dependencies.Characters, forking, one.stage, theThreeSlots[0], progression.LevelCap),
				placeUnit(t, dependencies.Characters, "pokemon.machop", theThreeSlots[1], progression.LevelCap),
				placeUnit(t, dependencies.Characters, "pokemon.gastly", theThreeSlots[2], progression.LevelCap),
			}}
			opened := newRoom(t, config(7, 1))
			seat, out, err := opened.Join(hello(t, squad, "Forked"))
			if err != nil {
				t.Fatalf("join: %v", err)
			}
			if one.want {
				if !seat.Valid() {
					t.Fatalf("fielding %q was refused with %q, and it is a tip of the line",
						one.stage, onlyCode(t, out))
				}
				return
			}
			if seat.Valid() {
				t.Fatalf("fielding %q was accepted, and something in the line grows out of it", one.stage)
			}
			if got := onlyCode(t, out); got != wire.CodeSquadRefused {
				t.Errorf("fielding %q was refused with %q, want %q", one.stage, got, wire.CodeSquadRefused)
			}
		})
	}
}

// TestOneSquadMayFieldTheSameCharacterTwice is a decision rather than an
// omission, and it is asserted so that nobody "tightens" it later.
//
// placement.Squad.Validate checks ids and slots and says nothing about
// characters; the squad builder will happily write two Charizards; and nothing
// in the engine cares, because a squad's ids are what tell its members apart and
// Take prefixes them with the side. A gate that refused it would refuse a player
// their own saved squad for a reason no screen has ever told them — and the
// screen that would have to start telling them does not exist. → the note on
// squadIsFieldable.
func TestOneSquadMayFieldTheSameCharacterTwice(t *testing.T) {
	dependencies := deps(t)
	const twice = "pokemon.charmander"
	squad := placement.Squad{ID: "doubled.squad", Units: []placement.Placement{
		twinUnit(t, dependencies.Characters, twice, "front", theThreeSlots[0]),
		twinUnit(t, dependencies.Characters, twice, "back", theThreeSlots[1]),
		placeUnit(t, dependencies.Characters, "pokemon.machop", theThreeSlots[2], progression.LevelCap),
	}}
	opened := newRoom(t, config(7, 1))
	seat, out, err := opened.Join(hello(t, squad, "Doubled"))
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if !seat.Valid() {
		t.Fatalf("a squad fielding %q twice was refused with %q", twice, onlyCode(t, out))
	}
	// And it reaches a battle rather than merely getting past the gate: the two
	// copies are distinguishable on the board, which is the property that makes
	// the decision safe.
	if _, _, err := opened.Join(hello(t, squad, "Also doubled")); err != nil {
		t.Fatalf("a mirror of a doubled squad cannot open a battle: %v", err)
	}
	if _, waiting := opened.Awaiting(); !waiting {
		t.Error("the match did not start with two doubled squads in the room")
	}
}

// twinUnit is one of two placements of the same character, told apart by an id
// suffix — which is what a squad has to tell its members apart by.
func twinUnit(t *testing.T, characters *cast.Book, name, suffix string, slot hex.Offset) placement.Placement {
	t.Helper()
	unit := placeUnit(t, characters, name, slot, progression.LevelCap)
	unit.ID = unit.ID + "." + suffix
	return unit
}

// TestALeavingPeerBeforeTheMatchFreesItsSeat is the one thing Left does that
// does not end a match: before the first battle there is no match to end, so the
// seat is freed and the room goes back to waiting.
//
// ⚠️ That is also why a reconnect window would sit **in front of** Left rather
// than inside it — the pre-match case frees the seat, so a rejoin cannot be
// something Left does. → TODO.md, under the seat token.
func TestALeavingPeerBeforeTheMatchFreesItsSeat(t *testing.T) {
	dependencies := deps(t)
	legal := squadOf(t, dependencies.Characters, "legal.squad",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	opened := newRoom(t, config(7, 1))
	seat, _, err := opened.Join(hello(t, legal, "Host"))
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := opened.Left(seat); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if opened.Finished() {
		t.Fatalf("a room nobody had played in was finished as %q", opened.Result().Verdict)
	}
	// The seat is free again, and it is the same seat: the host's place is the
	// first one a room hands out.
	retaken, out, err := opened.Join(hello(t, legal, "Host again"))
	if err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if retaken != seat {
		t.Errorf("the freed %q seat was handed out as %q, and the refusal was %q",
			seat, retaken, onlyCode(t, out))
	}
}

// mangled is a data digest that is not this binary's, built by flipping one byte
// of a real one rather than by inventing a value — so the case is "a peer whose
// data was edited" rather than "a peer that sent nonsense".
func mangled(real wire.Digest) wire.Digest {
	out := real
	out.Digest[0] ^= 0xFF
	return out
}

// onlyCode is the code of a refusal that is expected to be the room's whole
// answer. It insists on exactly one message, because a gate that also sent a
// welcome would be a gate that refused and seated the same peer.
func onlyCode(t *testing.T, out []room.Outbound) wire.Code {
	t.Helper()
	if len(out) != 1 {
		t.Fatalf("the gate answered with %d messages, want one refusal", len(out))
	}
	refused, ok := out[0].Body.(wire.Refused)
	if !ok {
		t.Fatalf("the gate answered with a %s, want a refusal", out[0].Body.Kind())
	}
	if out[0].To.Valid() {
		t.Errorf("a refusal at the gate names the %q seat, and refusing is what stops one being handed out",
			out[0].To)
	}
	if !refused.Code.Refuses() {
		t.Fatalf("the gate answered with the code %q, which is not a refusal", refused.Code)
	}
	return refused.Code
}
