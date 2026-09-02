package wire

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/seed"
)

// Protocol is the version of this format. It moves when the messages change and
// for no other reason: the wire changes independently of the game, which is the
// whole argument for it being a number of its own rather than a byte of the
// build string.
//
// What moves it: a kind or a code renamed or removed, a field renamed, a field's
// meaning changed. What does not: a new balance number, a new character, a new
// screen, or a code appended (a peer that meets an unknown code refuses cleanly
// and says so, which is a worse conversation than a good one and a much better
// one than a wrong one).
const Protocol = 1

// Digest carries a seed.Digest across the wire in the form seed's own
// documentation names as the one that travels: the sixty-four lowercase hex
// characters of Digest.String.
//
// ⚠️ It is not a second digest and it derives nothing. It *holds* a seed.Digest,
// so comparing a peer's against this binary's is an equality of that one type
// and the compiler checks it — which is the property the room's gate rests on,
// and the reason the field is not a string. Wrapping rather than re-declaring is
// also why the value can only be built out of a real seed.Digest: there is no
// path here that produces one from anything else.
//
// The wrapper exists purely for the encoding. seed.Digest is [32]byte and would
// otherwise ride as thirty-two numbers, which is a golden nobody can read and a
// diff nobody can review.
type Digest struct {
	seed.Digest
}

// MarshalJSON writes the digest as hex, through the String seed already
// declares — so there is one statement anywhere of what a digest looks like
// written down.
func (d Digest) MarshalJSON() ([]byte, error) { return json.Marshal(d.Digest.String()) }

// UnmarshalJSON reads a digest back out of those hex characters.
//
// This is a decoding of the declared form and not a second derivation of the
// digest: nothing here reads the data, and a value that arrives malformed is an
// error rather than a zero digest, because two peers agreeing on a digest of
// nothing is the one outcome this gate must never produce.
func (d *Digest) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return fmt.Errorf("decode data digest: %w", err)
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return fmt.Errorf("decode data digest: %w", err)
	}
	var out seed.Digest
	if len(decoded) != len(out) {
		return fmt.Errorf("decode data digest: %d bytes, want %d", len(decoded), len(out))
	}
	copy(out[:], decoded)
	d.Digest = out
	return nil
}

// Version is the three numbers a peer announces, and they answer three different
// questions — which is why there are three of them and not one. → README.md
// § Three version numbers, because there are three questions.
//
// It is embedded in Hello rather than nested inside it, so the three ride at the
// top level of a hello's body: they are the first thing a reader of the format
// should see, and a peer that cannot decode the rest of a hello can still be
// told which of the three it got wrong.
type Version struct {
	// Protocol answers "can these two talk at all". On a mismatch: refuse.
	Protocol int `json:"protocol"`
	// Build answers "what does the human need to update", and it is **printed
	// and never acted on**. Nothing in this package branches on it, and
	// TestTheThreeVersionNumbersAreActedOnDifferently asserts that by joining
	// two peers whose builds disagree. It is a free-form string because it is
	// for a person: a commit, a tag, whatever the binary knows about itself.
	Build string `json:"build"`
	// Data answers "will these two simulate the same battle", and it is the one
	// a version string cannot stand in for: editing a power in skills.json
	// changes every battle without moving a semver by one character. On a
	// mismatch: refuse at the gate.
	Data Digest `json:"data"`
}

// Check reports how this binary answers a peer announcing itself as v, and
// CodeNone means join. It is one function rather than three checks at three call
// sites, because the *order* of the two refusals is part of the answer.
//
// ⚠️ Protocol is checked before the data. A peer that cannot speak the protocol
// must not be told its data is wrong: it may not even be able to read the
// refusal, the digest it would be arguing about is a field this version of the
// format may not have in the same place, and the update it actually needs is the
// binary. TestTheThreeVersionNumbersAreActedOnDifferently pins the order with a
// peer that is wrong about both.
//
// Build is deliberately absent from the body of this function. It is the only
// one of the three with nothing to decide.
func (v Version) Check(against Version) Code {
	if v.Protocol != against.Protocol {
		return CodeProtocolMismatch
	}
	if v.Data.Digest != against.Data.Digest {
		return CodeDataMismatch
	}
	return CodeNone
}

// Local is the version this binary announces: the protocol constant above, the
// build string the caller was handed, and the digest of the data it embeds.
//
// The build string is a parameter because wire has no way to know it — a version
// is stamped at build time and read by the binary's own main — and inventing one
// here would be this package guessing at something it is not the source of.
func Local(build string) (Version, error) {
	digest, err := seed.DataDigest()
	if err != nil {
		return Version{}, fmt.Errorf("read the local data digest: %w", err)
	}
	return Version{Protocol: Protocol, Build: build, Data: Digest{Digest: digest}}, nil
}
