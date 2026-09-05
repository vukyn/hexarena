package wire

import (
	"encoding/json"
	"fmt"
)

// Code is why a peer was turned away. It is the whole of a Refused, and it is a
// code rather than a sentence for the reason the design record gives: a server
// that sent prose would be a server deciding what language its clients read in,
// and this client is Vietnamese-first with an English toggle. The client words
// it; the id is the only thing that travels.
//
// A code says nothing about *which* battle or *whose* turn, because a refusal
// this protocol can produce is always about the join or about the last message,
// both of which the peer already knows. Anything a code would have to explain
// with a number is a message of its own or nothing at all.
type Code uint8

const (
	// CodeNone is no refusal, and it is the zero value on purpose — the same
	// choice hex.SideNone makes and for the same reason. Version.Check returns
	// it to mean "join accepted", and a Refused carrying it would be a refusal
	// that says nothing, so the room must never send one. Code.Refuses is the
	// question to ask.
	CodeNone Code = iota
	// CodeProtocolMismatch is a peer this one cannot talk to at all. It is
	// checked and answered *before* the digest: telling a peer its data is wrong
	// when it cannot even read the message saying so is telling it something it
	// has no way to act on, and the update it actually needs is the binary.
	CodeProtocolMismatch
	// CodeDataMismatch is a peer whose embedded data is not this data, so the
	// two would not simulate the same battle. Refused at the gate rather than
	// discovered on the eleventh minute of a battle that was never the same
	// battle twice — which the mirror's per-turn digest would catch eventually,
	// and eventually is the whole problem.
	CodeDataMismatch
	// CodeBadPassword is the room's password not matching. ⚠️ The password is
	// **not** security (see Password) and this code is not a security answer: it
	// says the same thing to a stranger in the house as to a friend who typed it
	// wrong, which is all a gate on a LAN is for.
	CodeBadPassword
	// CodeRoomUnknown is a code that names no room this process is running: the
	// address in it resolved, so something answered, and the room it names is
	// gone or was never here. A room code carries its own address (see
	// RoomCode), so this is what a restarted host looks like.
	CodeRoomUnknown
	// CodeRoomFull is both seats taken. Spectators are deliberately later work,
	// so today a third client is a full room rather than a watcher.
	CodeRoomFull
	// CodeSquadRefused is a squad that did not pass the gate: Squad.Validate,
	// then Take (which is already the loadout check), then the format's size and
	// the level and stage rules. One code covers all of them, and that is a
	// decision rather than laziness — the client holds the same books and the
	// same validator, so it can say precisely what is wrong with a squad it
	// built, in the player's own language, without the server spelling it.
	CodeSquadRefused
	// CodeNotYourTurn is an act, a pass **or a draft decision** arriving from the
	// seat that is not being asked. The server knows whose turn it is — which is
	// also why Act carries no unit and why Decide carries no seat.
	//
	// ⚠️ **The draft reuses this rather than getting a code of its own**, and the
	// reason is the discipline at the top of this file: it is the same fact about
	// the same thing, and what a player does about it is the same either way —
	// nothing, the screen already says whose decision is due. Two codes for one
	// situation is worth less than one.
	CodeNotYourTurn
	// CodeIllegalAction is an act the prompt does not offer: a skill the unit
	// does not hold, one on cooldown, or an aim outside the cells the engine
	// listed. Act already refuses it, so this code is the engine's no travelling
	// back rather than a second reading of the rules.
	//
	// ⚠️ **It covers the draft's four legality refusals too, and that is a
	// decision rather than laziness** — the same decision CodeSquadRefused makes
	// for the four ways a squad can be turned away, for the same reason. A draft
	// decision of the wrong step, one naming a character out of the pool or
	// already taken, an illegal loadout and a refused arrangement are all
	// decisions the open prompt did not offer, and the client holds the pool,
	// cast.ChooseLoadout and placement.Squad.Validate itself — so it can say
	// precisely which, in the player's own language, where a code could only ever
	// say "one of four". A client that offered any of the four was wrong about
	// rules it holds, which is what this code's wording says.
	CodeIllegalAction
	// CodeUnknownMessage is a kind this peer does not know, which is what a
	// client one version ahead looks like. It is the answer Kind.UnmarshalJSON's
	// refusal turns into.
	CodeUnknownMessage
	// CodeSquadUnwanted is a squad brought to a room that **drafts**: the two
	// sides ban and pick here, so the side a client built at home is not the side
	// it will field. → Welcome.Drafts.
	//
	// ⚠️ **No existing code can say this, which is why it is a new one.** Every
	// other refusal about a squad is CodeSquadRefused, and that code means one
	// thing — the squad is not *legal* — which this squad may perfectly well be.
	// Answering it would send a player to check the levels, forms and skills of a
	// squad with nothing wrong with it, and its wording says in as many words to
	// fix the squad and join again; the fix here is to bring **none**. A refusal
	// that misdirects is worse than a refusal that is merely blunt.
	//
	// ⚠️ It is a refusal rather than a silent drop for the reason Welcome.Drafts
	// gives: a squad quietly ignored is a player watching their side fail to
	// appear with nothing saying why. And it is answerable at the *gate*,
	// unlike the draft's own refusals, because it is a fact about the hello.
	//
	// Declared last, which is the rule this enum shares with Kind and with
	// battle.Kind: a code serialises by name, so appending cannot reinterpret a
	// refusal a peer already knows how to word. ⚠️ **The comment moves with the
	// last constant** — left on CodeUnknownMessage it would say something false
	// about this file.
	CodeSquadUnwanted
)

// CodeCount is the number of codes, and it exists so a test can walk them rather
// than range over the table of names and ask it whether it holds what it holds.
//
// ⚠️ **Every code is worded and every code but one is drawn.** Lang.Refusal in
// internal/i18n carries a line per code in both books, and the walk over this
// count against those books is internal/i18n/protocol_test.go — it could never
// be here, because wire must not import internal/i18n (the whole point of a
// code is that the wording lives at the far end, and a protocol that imported
// the word book would be the server holding the sentences again). So this count
// is what is held here. cmd/hexarena-tui's
// TestEveryRefusalIsShownAndEveryClosureIsShown is the other half: it produces
// every code out of a real room and reads its sentence back off a drawn screen.
//
// ⚠️ **The one exception is CodeSquadUnwanted, and it is a stated debt rather
// than an omission.** No room answers it yet, because a room that drafts is the
// next step — so that test names it as owed and holds the owed set at exactly
// one, which is what stops the exception growing quietly. → TODO.md § *The
// draft on the wire*.
const CodeCount = int(CodeSquadUnwanted) + 1

// codeNames is the wire form of every code, and it is the format: renaming an
// entry breaks every peer built before the rename.
var codeNames = [CodeCount]string{
	CodeNone:             "none",
	CodeProtocolMismatch: "protocol_mismatch",
	CodeDataMismatch:     "data_mismatch",
	CodeBadPassword:      "bad_password",
	CodeRoomUnknown:      "room_unknown",
	CodeRoomFull:         "room_full",
	CodeSquadRefused:     "squad_refused",
	CodeNotYourTurn:      "not_your_turn",
	CodeIllegalAction:    "illegal_action",
	CodeUnknownMessage:   "unknown_message",
	CodeSquadUnwanted:    "squad_unwanted",
}

func (c Code) String() string {
	if int(c) >= CodeCount {
		return fmt.Sprintf("code(%d)", uint8(c))
	}
	return codeNames[c]
}

// Refuses reports whether the code names an actual refusal, which is what a
// Refused has to carry and what Version.Check returns nothing of on a peer it
// accepts.
func (c Code) Refuses() bool { return c != CodeNone && int(c) < CodeCount }

// MarshalJSON writes the code by name, so the format does not depend on the
// order these constants are declared in.
func (c Code) MarshalJSON() ([]byte, error) { return json.Marshal(c.String()) }

// UnmarshalJSON reads a code written by name, and refuses one it does not know.
//
// A refusal a peer cannot read is worse than no refusal: it would be worded from
// whatever constant happened to sit at zero, so a client would tell its player
// the room was fine and then go quiet.
func (c *Code) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode refusal code: %w", err)
	}
	for i, candidate := range codeNames {
		if candidate == name {
			*c = Code(i)
			return nil
		}
	}
	return fmt.Errorf("unknown refusal code %q", name)
}
