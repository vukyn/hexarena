package battle

import (
	"encoding/json"
	"fmt"
)

// Outcome is how a battle finished.
//
// It is a typed field on the closing event rather than a phrase in that event's
// note, because two readers have to switch on it rather than read it: a renderer
// deciding what to print, and a replay checking that the battle it re-ran ended
// the same way. An ending that can only be told apart by matching English is one
// neither of them can be trusted with — the same reason a side and a kind are
// names on the wire and values in the program.
type Outcome uint8

const (
	// Undecided is a battle still being fought. It is the zero value, so the
	// closing event is the only one that carries an outcome at all.
	Undecided Outcome = iota
	// Victory is one side left standing.
	Victory
	// Annihilation is both sides emptied, which a simultaneous kill can do:
	// nobody won, and nobody is left to go on.
	Annihilation
	// Stalemate is a battle nobody can act in and nothing pending will change:
	// units are alive on both sides and no skill any of them holds can be
	// aimed at anyone, now or ever, because nothing on this board moves.
	//
	// It is an outcome rather than an error. The turn limit is a backstop
	// against a runaway, and a deadlock is not one: it is a finished battle
	// whose result happens to be a draw.
	Stalemate
)

// OutcomeCount is the number of outcomes.
const OutcomeCount = int(Stalemate) + 1

var outcomeNames = [OutcomeCount]string{
	Undecided:    "undecided",
	Victory:      "victory",
	Annihilation: "annihilation",
	Stalemate:    "stalemate",
}

func (o Outcome) String() string {
	if int(o) >= OutcomeCount {
		return fmt.Sprintf("outcome(%d)", uint8(o))
	}
	return outcomeNames[o]
}

// Decided reports whether the outcome names a winner.
func (o Outcome) Decided() bool { return o == Victory }

// MarshalJSON writes the outcome by name, so a saved log does not depend on the
// order these constants are declared in.
func (o Outcome) MarshalJSON() ([]byte, error) { return json.Marshal(o.String()) }

// UnmarshalJSON reads an outcome written by name.
func (o *Outcome) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode outcome: %w", err)
	}
	for i, candidate := range outcomeNames {
		if candidate == name {
			*o = Outcome(i)
			return nil
		}
	}
	return fmt.Errorf("unknown outcome %q", name)
}
