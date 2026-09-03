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
	// CodeNotYourTurn is an act or a pass arriving from the seat that is not on
	// turn. The server knows whose turn it is — which is also why Act carries no
	// unit.
	CodeNotYourTurn
	// CodeIllegalAction is an act the prompt does not offer: a skill the unit
	// does not hold, one on cooldown, or an aim outside the cells the engine
	// listed. Act already refuses it, so this code is the engine's no travelling
	// back rather than a second reading of the rules.
	CodeIllegalAction
	// CodeUnknownMessage is a kind this peer does not know, which is what a
	// client one version ahead looks like. It is the answer Kind.UnmarshalJSON's
	// refusal turns into.
	//
	// Declared last, which is the rule this enum shares with Kind and with
	// battle.Kind: a code serialises by name, so appending cannot reinterpret a
	// refusal a peer already knows how to word.
	CodeUnknownMessage
)

// CodeCount is the number of codes, and it exists so a test can walk them rather
// than range over the table of names and ask it whether it holds what it holds.
//
// ⚠️ **Every code is worded now, and no screen draws one yet.** Lang.Refusal in
// internal/i18n carries a line per code in both books, and the walk over this
// count against those books is internal/i18n/protocol_test.go — it could never
// be here, because wire must not import internal/i18n (the whole point of a
// code is that the wording lives at the far end, and a protocol that imported
// the word book would be the server holding the sentences again). So this count
// is what is held here. What is still open is the *reader*: the pairing screen
// in cmd/hexarena-tui is what turns those wordings into something a player is
// shown, so until it lands the "shipped dead" shape is one step narrower —
// words exist, unread — rather than closed. → TODO.md § The client.
const CodeCount = int(CodeUnknownMessage) + 1

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
