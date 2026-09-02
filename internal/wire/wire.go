// Package wire is the PvP protocol: the seven messages two peers exchange, the
// three version numbers they check before they start, and the codes a refusal is
// reported in. It is the format and nothing else — no room, no socket, no I/O,
// not one read of a clock.
//
// Everything here is stdlib over types that already exist. That is the whole
// design, and it is the reason this package is small: a squad is already a wire
// format (placement.Squad is a *reference*, so a client cannot send an inflated
// stat line), a resolved unit is already tagged for JSON (battle.Roster), an aim
// is already a hex.Cell with its own absence, a taken turn is already a
// battle.Decision, and the data fingerprint is already seed.Digest. A second
// declaration of any of them would drift from the first, which is the mistake
// this repository has recorded more times than any other.
//
// Three rules come straight off the design record (README.md § PvP over a LAN)
// and every type below is shaped by them:
//
//   - **Kinds and codes serialise by name, never by number.** This is already
//     hard law for event kinds, sides and outcomes: a saved log — or here, a
//     peer one commit older — would silently reinterpret itself the next time a
//     constant was inserted. Renaming a kind or a code is a wire break, so the
//     names *are* the format.
//   - **Nothing on the wire is prose.** The server sends a Code and the client
//     words it. A server that sent sentences would be a server deciding what
//     language its clients read in, and the client is Vietnamese-first.
//   - **The clock is not part of the battle.** The turn allowance is room
//     configuration and rides on Welcome; no battle-carrying message holds a
//     reading of a clock. This package imports `time` not at all, and
//     TestTheProtocolCannotReadAClock holds it mechanically.
//
// ⚠️ Why this imports internal/seed. The data digest must be *the same type* on
// both sides of a comparison — a room's gate that compared two strings would
// accept a peer whose digest had been mangled into the same characters by some
// other path, and would compile if somebody handed it an event digest by
// mistake. seed.Digest is that type, seed does not and cannot import wire (it is
// data over internal/core, and this is a protocol over both), so there is no
// cycle. The cost is that a binary importing wire embeds the fifteen data files
// — and both peers already import seed, because neither can *compute* its own
// digest, resolve a squad or draw a description without it. So the cost is zero
// in practice and the alternative was a second digest type.
package wire

import (
	"encoding/json"
	"fmt"
)

// Kind names which of the seven messages an envelope carries.
//
// Three go from a client to a server and four come back. There are deliberately
// no others: see the note on Turn for the series-standing message that is
// missing on purpose.
type Kind uint8

const (
	// KindHello is the first thing a client says: the three version numbers, the
	// squad it brings, the room's password and a name to show the other player.
	// It is the only message with a gate in front of it — see Version.Check.
	KindHello Kind = iota
	// KindAct is a client spending its turn. A skill and an aim, and nothing
	// else; see Act for why it names no unit.
	KindAct
	// KindPass is a client giving its turn up. It carries nothing at all; see
	// Pass, which is where the reason would have gone and why it does not.
	KindPass
	// KindWelcome is the room accepting a client: the room's configuration and
	// which of the two seats this client took.
	KindWelcome
	// KindRefused is the room turning a client away, as a Code and nothing else.
	KindRefused
	// KindStart opens a battle of the series: the seed, the roster both peers
	// build their engine from, which side this client plays and which battle
	// this is.
	KindStart
	// KindTurn is one resolved turn: the decision that was taken and the digest
	// of the events it produced, which is what makes a divergence loud on the
	// turn it happens rather than a board that quietly drifts.
	//
	// Declared last, which is the rule this enum shares with battle.Kind: a kind
	// serialises by name, so appending cannot reinterpret anything already
	// written, while slotting one in beside KindAct would move KindCount and
	// every table built from declaration order.
	KindTurn
)

// KindCount is the number of message kinds.
//
// It exists so a test can walk the kinds rather than range over a table and ask
// it whether it holds what it holds. A kind added without a name, without a
// decoder or without a golden entry is then a red test rather than a message
// nothing measures — the shape cmd/hexarena-tui's screenCount established after
// five screens slipped into the authoring client unmeasured.
const KindCount = int(KindTurn) + 1

// kindNames is the wire form of every kind. It is the format: renaming an entry
// here breaks every peer built before the rename, which is why a rename is a
// protocol bump and not a tidy-up.
var kindNames = [KindCount]string{
	KindHello:   "hello",
	KindAct:     "act",
	KindPass:    "pass",
	KindWelcome: "welcome",
	KindRefused: "refused",
	KindStart:   "start",
	KindTurn:    "turn",
}

func (k Kind) String() string {
	if int(k) >= KindCount {
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
	return kindNames[k]
}

// MarshalJSON writes the kind by name, so the format does not depend on the
// order these constants are declared in.
func (k Kind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }

// UnmarshalJSON reads a kind written by name, and refuses one it does not know.
//
// This is where an unknown message becomes a clean refusal rather than a parse
// error or a zero value: a peer one version ahead sends a kind this one has
// never heard of, and the answer is a Refused carrying CodeUnknownMessage. A
// zero value would have been the worst of the three — the connection would have
// carried on, reading every unknown message as a hello.
func (k *Kind) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode message kind: %w", err)
	}
	for i, candidate := range kindNames {
		if candidate == name {
			*k = Kind(i)
			return nil
		}
	}
	return fmt.Errorf("unknown message kind %q", name)
}

// Body is one of the seven message bodies, and the interface exists for exactly
// one reason: it makes the pairing of a kind with a body a property of the type
// rather than of whoever remembered to set the field. Encode never takes a kind.
type Body interface {
	// Kind is which envelope this body travels in.
	Kind() Kind
}

// Envelope is what actually crosses the connection: a named kind and the body,
// still raw.
//
// One connection carries all seven, so the reader has to know what it is holding
// before it can decode it — which is the same discriminant-plus-payload shape
// battle.Event takes for the same reason, one line at a time from a single
// stream. The body stays json.RawMessage here so that decoding is one pass over
// the bytes and an unknown kind never reaches a struct.
type Envelope struct {
	Kind Kind            `json:"kind"`
	Body json.RawMessage `json:"body,omitempty"`
}

// UnmarshalJSON reads an envelope, and it exists for one case that the field's
// own decoder cannot see: an envelope that names **no** kind at all.
//
// Kind.UnmarshalJSON refuses a name it does not know, but it is never called for
// a field that is not there — so `{"body":{}}` would leave the kind at its zero
// value and every unlabelled message in the protocol would read as a hello.
// That is the trap hex.SideNone exists to avoid, and the two cases take opposite
// answers on purpose: a side genuinely has a "nobody" (a draw has no winner), so
// it is represented, while an envelope with no kind is not a message this format
// has — nothing sends one — so it is refused rather than given a name. Which
// also keeps the enum at the seven messages the design record pins.
func (e *Envelope) UnmarshalJSON(raw []byte) error {
	var shape struct {
		Kind *Kind           `json:"kind"`
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		return err
	}
	if shape.Kind == nil {
		return fmt.Errorf("the envelope names no kind")
	}
	e.Kind, e.Body = *shape.Kind, shape.Body
	return nil
}

// bodyForKind builds an empty body per kind, and it is the table Decode
// dispatches on. Every entry must be non-nil; TestEveryMessageKindHasANameAndABody
// walks KindCount to say so, because a nil entry is a kind that parses and then
// cannot be read.
var bodyForKind = [KindCount]func() Body{
	KindHello:   func() Body { return new(Hello) },
	KindAct:     func() Body { return new(Act) },
	KindPass:    func() Body { return new(Pass) },
	KindWelcome: func() Body { return new(Welcome) },
	KindRefused: func() Body { return new(Refused) },
	KindStart:   func() Body { return new(Start) },
	KindTurn:    func() Body { return new(Turn) },
}

// Encode wraps a body in its own envelope and returns the bytes that travel.
//
// The kind comes from the body rather than from an argument, so a message
// labelled as something it is not is unrepresentable rather than a bug to be
// found on the far end.
func Encode(body Body) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("encode message: no body")
	}
	kind := body.Kind()
	if int(kind) >= KindCount {
		return nil, fmt.Errorf("encode message: %s is not a kind this protocol declares", kind)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode %s body: %w", kind, err)
	}
	out, err := json.Marshal(Envelope{Kind: kind, Body: raw})
	if err != nil {
		return nil, fmt.Errorf("encode %s envelope: %w", kind, err)
	}
	return out, nil
}

// Decode reads an envelope and its body, returning a pointer to one of the seven
// structs. The caller switches on the type, or asks the result for its Kind.
//
// Every failure is an error and never a usable-looking zero value: an unknown
// kind, a body that does not fit the kind it claims, and trailing bytes after
// the envelope are each a refusal. A peer that guessed would be a peer that
// fights a different battle for a while before anybody noticed.
func Decode(raw []byte) (Body, error) {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode message envelope: %w", err)
	}
	// Unreachable through UnmarshalJSON, which refuses an unknown name before
	// this line — but a caller can also build an Envelope in Go, and a table
	// entry can be forgotten.
	if int(envelope.Kind) >= KindCount || bodyForKind[envelope.Kind] == nil {
		return nil, fmt.Errorf("decode message: no body is declared for kind %s", envelope.Kind)
	}
	body := bodyForKind[envelope.Kind]()
	if len(envelope.Body) == 0 {
		// Pass carries nothing, so its body is legitimately absent. Any other
		// kind arriving empty is a message that lost its contents in transit.
		if envelope.Kind != KindPass {
			return nil, fmt.Errorf("decode %s: the envelope carries no body", envelope.Kind)
		}
		return body, nil
	}
	if err := json.Unmarshal(envelope.Body, body); err != nil {
		return nil, fmt.Errorf("decode %s body: %w", envelope.Kind, err)
	}
	return body, nil
}
