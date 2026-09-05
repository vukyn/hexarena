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
// There are **three of those** and the enum is shaped for a fourth: a value
// added here is an entry in a table rather than a new message kind, which is the
// whole reason the reason is a field and not a Kind. ClosureStopped and
// ClosureDraftExpired are what the second and third cost — one constant and one
// name each, and no new message either time — which is the evidence for the
// shape rather than a claim about it.
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
	ClosureLeft
	// ClosureStopped is the host having stopped serving: the process is going
	// away, so the match is over wherever it had got to.
	//
	// ⚠️ **It is a different fact from ClosureLeft and sending that one instead
	// would be a lie.** A departure is a judgement about a *peer* — the thing
	// that owns the connection decided there was nobody at the other end — and
	// this is the thing that owns the connection deciding to stop. A player told
	// "your opponent left" while their opponent was sitting right there would go
	// looking for a network fault that does not exist. The other way round is no
	// better: sending nothing at all leaves a player staring at a socket that
	// died for no stated reason, which is the one thing a Closed exists to
	// prevent.
	//
	// ⚠️ Like ClosureLeft it is **not a loss for anybody**. Nothing about the
	// host's process ending is a fact about the board, so the match has no
	// winner and the room's own verdict for it is room.VerdictAbandoned — the
	// same one a departure produces, because "nobody played this out" is the
	// same statement either way.
	ClosureStopped
	// ClosureDraftExpired is an allowance running out during the ban and pick:
	// **a draft that runs out of time is not resumed**, the room closes, and the
	// match is played again from a new code.
	//
	// ⚠️ **That is the one place the design does not follow "a timeout announces
	// and passes"**, and the reason is that there is nothing honest to pass with:
	// a side that never picked has no squad to fight with, and a defaulted pick
	// or a defaulted formation would hand somebody a side they did not choose and
	// call it theirs — where placement alone is worth nineteen points of win rate
	// (27.6% → 47.3% ally on the shipped roster). → TODO.md § "Ban and pick" (c).
	//
	// ⚠️ **It is a closure and not a refusal, which is what makes it belong
	// here.** A Refused answers one message and the match carries on; this ends
	// the match, and a mirror that was only refused would sit holding an open
	// draft decision waiting for a decision nobody is coming to take — the exact
	// hang Closed exists to prevent.
	//
	// ⚠️ Like both closures above it is **not a loss for anybody**. Nobody
	// drafted a squad, so there is no board to have a verdict about.
	//
	// Declared last, which is the rule this enum shares with Kind, Code and
	// battle.Kind: a closure serialises by name, so appending cannot
	// reinterpret an ending a peer already knows how to word. ⚠️ **The comment
	// moves with the last constant** — left on ClosureStopped it would say
	// something false about this file.
	ClosureDraftExpired
)

// ClosureCount is the number of closures, and it exists so a test can walk them
// rather than range over the table of names and ask it whether it holds what it
// holds.
//
// ⚠️ **Every ending is worded and every ending is drawn** — the same state
// CodeCount is in, and worded in the same place for the same reason: wire must
// not import internal/i18n, because the whole point of sending an id is that the
// wording lives at the far end. Lang.Closure is that end,
// internal/i18n/protocol_test.go is the walk over this count against both books,
// and cmd/hexarena-tui's TestEveryRefusalIsShownAndEveryClosureIsShown is what
// reads each sentence back off the result screen.
//
// ⚠️ Note the asymmetry with CodeCount, which is worth knowing rather than
// smoothing over: a closure needs no producer to be *drawable*, because the
// result screen words whatever closure it is handed — so ClosureDraftExpired is
// read by a player from the day it exists, while CodeSquadUnwanted is owed a
// room. What neither test can see is whether a room ever *sends* this one. →
// TODO.md § *The draft on the wire*.
const ClosureCount = int(ClosureDraftExpired) + 1

// closureNames is the wire form of every closure, and it is the format:
// renaming an entry breaks every peer built before the rename.
var closureNames = [ClosureCount]string{
	ClosureNone:         "none",
	ClosureLeft:         "left",
	ClosureStopped:      "stopped",
	ClosureDraftExpired: "draft_expired",
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
