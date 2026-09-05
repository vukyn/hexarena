package room

import (
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/wire"
)

// Join is the gate. It reports the seat the peer took and everything the room
// says back; on a refusal the seat is the zero Seat and the one message is a
// wire.Refused carrying a wire.Code.
//
// ⚠️ On a refusal the Outbound names **no seat**, because refusing is precisely
// what stops one being handed out — the transport answers the connection it read
// the hello from. Seat.Valid is the question to ask about the first return.
//
// # The order, which is part of the answer
//
// Five checks, and the order is pinned by TestTheGateRefusesInItsOwnOrder
// because a gate whose order is untested is a gate that reports whichever fault
// it happened to notice first. A peer wrong about two things has to be told
// about the earlier one:
//
//  1. **The version**, through wire.Version.Check, which is itself protocol
//     before digest. A peer that cannot speak the protocol must not be told its
//     data is wrong: it may not be able to read the refusal, and the update it
//     needs is the binary. That ordering is tested in internal/wire and is not
//     restated here.
//  2. **The password**, in constant time, through wire.Password.Equal. Before
//     the seat, so that a stranger with the wrong password learns nothing about
//     how full the room is.
//  3. **The seat.** Before the squad, because a squad check on a full room is
//     work done to reach an answer that was already decided — and because
//     "your squad is illegal" is a worse thing to tell somebody who was never
//     getting in than "the room is full".
//  4. **The squad**, which is five rules under one code (see squadRefused) — or,
//     in a room that **drafts**, the opposite question under a different code:
//     there a squad is *unwanted* rather than illegal, and squadIsFieldable is
//     not consulted at all. → the branch below, and squadIsFieldable's own note.
//  5. Nothing else. wire.CodeRoomUnknown is a *registry's* refusal — a code
//     naming no room this process is running — and the registry of many rooms
//     is its own TODO.md item, so no room ever sends it.
func (r *Room) Join(hello wire.Hello) (wire.Seat, []Outbound, error) {
	if code := r.deps.Version.Check(hello.Version); code.Refuses() {
		return "", r.refuseConnection(code), nil
	}
	if r.config.Password.Set() && !r.config.Password.Equal(hello.Password) {
		return "", r.refuseConnection(wire.CodeBadPassword), nil
	}
	index, free := r.freeSeat()
	if !free {
		return "", r.refuseConnection(wire.CodeRoomFull), nil
	}
	// ⚠️ **In a room that drafts a squad is UNWANTED, not illegal, and the two
	// branches are exclusive rather than one after the other.** The two sides ban
	// and pick out of a shared pool here, so the side a client built at home is
	// not the side it will field — and the squad may be perfectly legal, which is
	// why running squadIsFieldable on it would be answering a question nobody
	// asked. What a player has to *do* is the whole difference: CodeSquadRefused
	// says fix the squad and join again, and the fix here is to bring **none**. A
	// refusal that misdirects is worse than one that is merely blunt.
	// → wire.CodeSquadUnwanted, and Welcome.Drafts on why a hello cannot know
	// this before it is sent.
	if r.config.Drafts {
		if broughtASquad(hello.Squad) {
			return "", r.refuseConnection(wire.CodeSquadUnwanted), nil
		}
	} else if !r.squadIsFieldable(hello.Squad) {
		return "", r.refuseConnection(wire.CodeSquadRefused), nil
	}
	seat := seats[index]
	// The squad is the empty one in a drafting room, and the draft fills both in
	// itself once it is Done. → draftAdvanced.
	r.seated[index] = peer{taken: true, name: hello.Name, squad: hello.Squad.Clone()}
	// Every room setting a client needs in order to behave correctly, and the
	// cap is one of those rather than an extra: a mirror that did not know it
	// would sit holding an open prompt on a battle the room had stopped asking
	// about. → wire.Welcome.TurnCap.
	out := []Outbound{{To: seat, Body: wire.Welcome{
		Format:    r.config.Format,
		Battles:   r.config.Battles,
		Allowance: r.config.Allowance,
		TurnCap:   r.config.TurnCap,
		Drafts:    r.config.Drafts,
		Seat:      seat,
	}}}
	// The second peer to be seated starts the match, which is the one place a
	// join produces more than an answer to itself.
	if _, stillFree := r.freeSeat(); !stillFree {
		opening, err := r.bothTaken()
		if err != nil {
			return seat, out, err
		}
		out = append(out, opening...)
	}
	return seat, out, nil
}

// bothTaken is what the second peer sitting down starts.
//
// ⚠️ **A room that drafts opens its draft here instead of calling begin(), and
// opening one sends NOTHING AT ALL.** The draft was built in New and its first
// ban is due the moment the second seat is taken, so there is no state to
// announce: a wire.Drafted carries *recorded decisions*, none have been taken,
// and a room must not send one carrying none (→ wire.Drafted.Decisions). What a
// client needs in order to draw the opening ban it already holds — Welcome.Drafts
// says a draft is coming and Welcome.Seat says which side it is on, and the host
// bans first — so both peers compute the same open decision out of the same two
// facts. → New, where that constant is stated once.
func (r *Room) bothTaken() ([]Outbound, error) {
	if r.config.Drafts {
		return nil, nil
	}
	return r.begin()
}

// broughtASquad reports whether a hello named a side at all, which is the whole
// of what a drafting room turns away.
//
// ⚠️ **It reads the units and not the id**, and that is a decision: a squad is a
// side to field, so a client that filled in a name and no members brought
// nobody — placement.Squad.Validate refuses that shape by its own first line, so
// there was never a squad there to be unwanted. What CodeSquadUnwanted exists to
// prevent is a player watching the side they spent an evening building fail to
// appear, and an empty squad is not that side.
func broughtASquad(squad placement.Squad) bool { return len(squad.Units) > 0 }

// freeSeat is the first seat nobody holds, in the order a room hands them out,
// so the peer that opened the room is the host.
func (r *Room) freeSeat() (int, bool) {
	for index := range r.seated {
		if !r.seated[index].taken {
			return index, true
		}
	}
	return 0, false
}

// squadIsFieldable is the squad half of the gate: five rules, all of them under
// wire.CodeSquadRefused, and the one code is a decision rather than laziness.
//
// ⚠️ **It is NOT consulted in a room that drafts, and that is the point rather
// than a saved call.** There the question is not whether the squad is legal —
// it may well be — but that nobody wanted one: the sides ban and pick out of a
// shared pool, so the squad a client built at home is not the squad it will
// field. Join answers wire.CodeSquadUnwanted before reaching here, and running
// these five rules on the way would risk telling a player their perfectly legal
// squad was wrong about its levels or its forms, which is the misdirection that
// code was added to avoid. → Join.
//
// ⚠️ **And that room is why the doubling-up rule below has a SCOPE.** CLAUDE.md's
// *"one squad may field the same character twice"* is decided yes and holds
// here — a **saved** squad is what this gate sees. A **drafted** squad cannot
// double up at all, and not because anything refuses it: every ban and every pick
// takes a character out of one shared *exclusive* pool, so a side's picks are
// different characters by construction, and so are both sides' together. Both
// statements hold, and this is where the scope became load-bearing rather than
// descriptive. → internal/draft's Squads, which says the same thing from the
// other end.
// The client holds the same books and the same validator, so it can say
// precisely what is wrong with a squad it built, in the player's own language,
// without the server spelling it — and a server that spelled it would be a
// server deciding what language its clients read in.
//
// In order, and each of them is here because the one before it cannot see it:
//
//  1. placement.Squad.Validate — the ids, the slots, the level bounds. It is
//     deliberately lenient about a half-finished unit, because a squad being
//     built has to be savable, so it is the floor rather than the gate.
//  2. **The format's size.** A 3v3 room takes squads of three. Validate only
//     knows hex.MaxTeamSize, which is five, so a three-unit room would take a
//     five-unit squad without this.
//  3. **Level 60.** PvP is fought at the cap; every balance figure in the
//     repository was read there, and a squad brought under-levelled is a player
//     giving away a match rather than a choice worth offering.
//  4. **A leaf of the line.** Fully grown, both arms of a fork accepted, an
//     interior form refused. → the note on leafStage, which is the half of this
//     gate that is easy to get subtly wrong.
//  5. placement.Squad.Take, which **is** the loadout check: four skills out of
//     what the level and form unlocked, one trait out of the two the placement
//     allows, through cast.ChooseLoadout. Last because it is the only one that
//     needs the cast book resolved, and because it is the only one that could
//     fail for a reason the four above have already ruled out.
//
// ⚠️ **One squad MAY field the same character twice, and this gate allows it on
// purpose.** placement.Squad.Validate checks ids and slots and says nothing
// about characters, the squad builder will happily write two Charizards, and
// nothing in the engine cares — a squad's ids are what tell its members apart
// and Take prefixes them with the side, so even a mirror of a mirror stays
// readable in a log. A gate that refused it would refuse a player their own
// saved squad for a reason no screen has ever told them, and the screen that
// would have to start telling them does not exist. The measurement that argues
// the other way — that two copies of the same character is the strongest squad
// available — has not been taken, and refusing a shape on a hunch is what this
// repository does not do.
func (r *Room) squadIsFieldable(squad placement.Squad) bool {
	if err := squad.Validate(); err != nil {
		return false
	}
	if len(squad.Units) != r.config.Format.Units() {
		return false
	}
	for _, unit := range squad.Units {
		if unit.Level != progression.LevelCap {
			return false
		}
		if !r.leafStage(unit) {
			return false
		}
	}
	// The side is hex.SideAlly here and it is not a claim about which half this
	// squad will fight from: a match is fought both ways round, so Take is
	// called again per battle with the side that battle assigns. What Take
	// checks does not depend on the side at all — the side only prefixes the
	// resolved ids and fills Roster.Side — so either answers the legality
	// question, and this one is discarded.
	if _, err := squad.Take(hex.SideAlly, r.deps.Characters); err != nil {
		return false
	}
	return true
}

// leafStage reports whether a placement fields a form with nothing after it.
//
// ⚠️ **A leaf is not progression.Furthest and it is not StageAt.** Furthest is
// every tip a *level* has reached, so at the cap it agrees with this by
// coincidence and would start disagreeing the day a stage was authored above the
// cap; StageAt refuses a fork outright rather than reporting both arms, so a
// gate written on it would refuse a legal Poliwrath for having a sibling.
// progression.Line.IsLeaf is the predicate this wants, and it was added there
// rather than here because "is anything after this form" is a fact about a line
// and a second copy of it in this package is the drift this repository keeps a
// list of.
//
// The stage is **resolved first**, which is what makes a placement that names no
// stage work: an absent stage means the furthest the level reaches, so on a line
// that does not fork it resolves to the single leaf and is accepted, and on a
// line that forks Resolve refuses it with "name the one being fielded" — which
// is the right refusal and not one this gate has to write.
func (r *Room) leafStage(unit placement.Placement) bool {
	character, known := r.deps.Characters.Get(unit.Character)
	if !known {
		return false
	}
	_, stage, err := character.Resolve(unit.Level, unit.Stage)
	if err != nil {
		return false
	}
	leaf, err := character.Stages.IsLeaf(stage.Name)
	if err != nil {
		return false
	}
	return leaf
}
