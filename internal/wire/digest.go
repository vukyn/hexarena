package wire

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
)

// EventDigest is the fingerprint of a run of events — in practice the events one
// turn produced, which is what a Turn carries.
//
// It is a **different type** from the data digest on purpose, and not because
// two hashes want two names. The two answer different questions and confusing
// them would be a check that passed while measuring nothing: the data digest
// asks "will these two peers simulate the same battle" once, at the gate, and
// this one asks "did they, this turn" every turn. Sharing seed.Digest between
// them would make comparing a data digest to an event digest compile.
type EventDigest [32]byte

// String is the whole digest, sixty-four lowercase hex characters, which is the
// form that travels and the form a divergence gets reported in.
func (d EventDigest) String() string { return hex.EncodeToString(d[:]) }

// Short is the first twelve hex characters, for a person reading two screens
// side by side. ⚠️ Never compare on this, for the reason seed.Digest.Short says:
// forty-eight bits is plenty to tell two turns apart at a glance and nowhere
// near enough to be the check that decides whether two peers are still fighting
// the same battle.
func (d EventDigest) Short() string { return d.String()[:12] }

// MarshalJSON writes the digest as hex rather than as thirty-two numbers.
func (d EventDigest) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON reads it back, and a malformed one is an error rather than a
// zero digest: two peers agreeing on the digest of nothing is the failure this
// check exists to catch.
func (d *EventDigest) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return fmt.Errorf("decode event digest: %w", err)
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return fmt.Errorf("decode event digest: %w", err)
	}
	if len(decoded) != len(d) {
		return fmt.Errorf("decode event digest: %d bytes, want %d", len(decoded), len(d))
	}
	copy(d[:], decoded)
	return nil
}

// DigestEvents is the fingerprint of a run of events, and it lives **here**
// rather than in either peer because both of them have to compute it identically
// — two implementations of it is the drift this repository bans everywhere else,
// and the one place it would show up is a divergence report that was itself the
// divergence.
//
// Each event is **framed** the way seed.digest frames a file: its byte length as
// a fixed-width big-endian integer, then its bytes. The bytes are what
// json.Marshal writes off battle.Event's own tags, so the shape of an event is
// still declared in exactly one place — a field added there enters this digest
// with no edit here, which is the property TestTheEventDigestReadsEveryFieldOfAnEvent
// holds by walking the struct rather than by naming fields.
//
// Two things about the framing were decided rather than copied, and this is the
// second user of the idea seed introduced, so both are written down:
//
//   - **The name half of seed's framing has no analogue here and is left out.**
//     There, the name catches a *rename* — the same bytes read under another
//     file name — and does nearly all the work, because a file's name is not in
//     the file. An event's identity *is* in its bytes: `kind` is a field, and it
//     is the first thing marshalled. So a name prefix here would be a second
//     copy of something already inside the frame, which is the one kind of
//     redundancy this repository consistently refuses.
//   - **The length half is kept, and its justification is stronger here than it
//     was there** — though it still has no test that isolates it, and that is
//     worth saying rather than implying. A stream of marshalled events is
//     self-delimiting *because of a property of the encoder*: every event begins
//     `{"kind":"` and json.Marshal escapes every quote inside a string, so no
//     free-text Note can forge that sequence and no two different runs of events
//     can concatenate to the same bytes. Which means the collision seed could
//     write down (a boundary moved across a copy of the next file's own name)
//     cannot be built here at all. The length is what makes the framing
//     unambiguous without resting on the escaping, for eight bytes an event.
//
// ⚠️ Not shared with seed.digest, deliberately. The two framings are different —
// that one walks an fs.FS and prefixes a name, this one walks a slice and
// prefixes nothing — so a shared helper would be the union of both with each
// caller passing a piece it does not use, and it would have to live in a third
// package that both internal/seed and internal/wire import, since internal/core
// may not. What is genuinely common is sha256 plus a big-endian length, and both
// of those are already the standard library. Restating six lines is cheaper than
// a package, and the difference between the two framings is a fact worth having
// visible at both sites.
//
// An error rather than a swallowed one for the reason seed's digest returns one:
// a partial digest is worse than none, because two peers that both skipped the
// same unencodable event would agree and carry on.
func DigestEvents(events []battle.Event) (EventDigest, error) {
	hasher := sha256.New()
	var length [8]byte
	for index, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			return EventDigest{}, fmt.Errorf("digest event %d (%s): %w", index, event.Kind, err)
		}
		binary.BigEndian.PutUint64(length[:], uint64(len(raw)))
		hasher.Write(length[:])
		hasher.Write(raw)
	}
	var out EventDigest
	copy(out[:], hasher.Sum(nil))
	return out, nil
}
