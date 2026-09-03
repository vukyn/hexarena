package wire

import (
	"encoding/json"
	"fmt"
)

// Closure is why a room closed a match, and it is the whole of a Closed.
//
// It is an enum rather than a sentence for the reason a Code is: a server that
// sent prose would be a server deciding what language its clients read in, and
// this client is Vietnamese-first with an English toggle. The client words it;
// the id is the only thing that travels.
//
// ⚠️ **A match played out to its end closes nothing.** The client is a mirror:
// it learns each battle's outcome from its own Ended event and it knows the
// series length from Welcome.Battles, so a room that announced "you won" would
// be declaring a second time what the client has already computed — the mistake
// the missing series-standing message exists to avoid. What this enum names is
// the *other* kind of ending: a match stopped for a reason the board cannot
// show, which a mirror has no way to reach on its own.
//
// There is **one of those today** and the enum is shaped for a second: a value
// added here is an entry in a table rather than a new message kind, which is
// the whole reason the reason is a field and not a Kind.
type Closure uint8

const (
	// ClosureNone is no closure at all, and it is the zero value for exactly the
	// reason CodeNone is: an absent field must not read as a real reason. A
	// Closed carrying it would be a room saying the match is over and declining
	// to say why, so no room may send one — Closure.Closes is the question to
	// ask.
	ClosureNone Closure = iota
	// ClosureLeft is the opponent having gone away.
	//
	// ⚠️ Whether a peer has really gone or is merely slow is the **transport's**
	// judgement and never the room's: the room is told, exactly as it is told
	// about a timeout, and a reconnect window sits in front of that report
	// rather than inside the room. So this is not "the opponent quit" — it is
	// "the thing that owns the connection has decided there is nobody there".
	//
	// ⚠️ It is **not a loss for the leaver and not a win for the one still
	// there**, and that is a decision rather than an omission: on a LAN between
	// friends the enforcement of walking away is social. → README.md § PvP over
	// a LAN, where the cost is written down.
	//
	// Declared last, which is the rule this enum shares with Kind, Code and
	// battle.Kind: a closure serialises by name, so appending cannot
	// reinterpret an ending a peer already knows how to word.
	ClosureLeft
)

// ClosureCount is the number of closures, and it exists so a test can walk them
// rather than range over the table of names and ask it whether it holds what it
// holds.
//
// ⚠️ Nothing words these yet, the same gap CodeCount carries and for the same
// reason: wire must not import internal/i18n, because the whole point of
// sending an id is that the wording lives at the far end. "Opponent left" is
// already on the wordings list. → TODO.md § The client.
const ClosureCount = int(ClosureLeft) + 1

// closureNames is the wire form of every closure, and it is the format:
// renaming an entry breaks every peer built before the rename.
var closureNames = [ClosureCount]string{
	ClosureNone: "none",
	ClosureLeft: "left",
}

func (c Closure) String() string {
	if int(c) >= ClosureCount {
		return fmt.Sprintf("closure(%d)", uint8(c))
	}
	return closureNames[c]
}

// Closes reports whether the closure names an actual ending, which is what a
// Closed has to carry.
func (c Closure) Closes() bool { return c != ClosureNone && int(c) < ClosureCount }

// MarshalJSON writes the closure by name, so the format does not depend on the
// order these constants are declared in.
func (c Closure) MarshalJSON() ([]byte, error) { return json.Marshal(c.String()) }

// UnmarshalJSON reads a closure written by name, and refuses one it does not
// know.
//
// A reason a peer cannot read is worse than no reason: it would be worded from
// whatever constant happened to sit at zero, so a client would tell its player
// nothing had happened and then go quiet on a match that was over.
func (c *Closure) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode closure: %w", err)
	}
	for i, candidate := range closureNames {
		if candidate == name {
			*c = Closure(i)
			return nil
		}
	}
	return fmt.Errorf("unknown closure %q", name)
}
