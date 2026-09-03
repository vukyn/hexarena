package i18n

// The two protocol enums, turned into sentences.
//
// internal/wire sends a refusal and a closure as an **id** and nothing else,
// because a server that sent prose would be a server deciding what language its
// clients read in — and this client is Vietnamese-first with an English toggle.
// The far end is here. So this file is the far end, and it is the reason
// internal/wire must never import internal/i18n.
//
// ⚠️ **These take the enum's name rather than the typed value**, which is the
// house pattern Lang.StatusCategory already uses and is a decision rather than
// an accident: it keeps internal/wire out of this package's *production*
// imports entirely, so the direction of the dependency stays a rule nothing has
// to remember. What walks wire.CodeCount and wire.ClosureCount is
// protocol_test.go, where the import is test-only and a value added to either
// enum with no wording is a red test rather than a screen showing an id.

// Refusal words a wire.Code — why the room turned this client away.
//
// Held complete for the reason Lang.StatusCategory is: a code is a Go enum, not
// a data id, so a code with no wording is a gap in the catalog and fails a
// test, rather than falling through to its own name. The fall-through is kept
// anyway, for the one case a test cannot cover — a peer one version ahead
// sending a code this build has never heard of, which arrives as a name and
// not as a constant.
//
// The map is a lookup keyed by the name wire.Code already writes, so nothing
// here decides an order.
func (l Lang) Refusal(name string) string {
	worded := map[string]Key{
		"none":              RefusalNone,
		"protocol_mismatch": RefusalProtocolMismatch,
		"data_mismatch":     RefusalDataMismatch,
		"bad_password":      RefusalBadPassword,
		"room_unknown":      RefusalRoomUnknown,
		"room_full":         RefusalRoomFull,
		"squad_refused":     RefusalSquadRefused,
		"not_your_turn":     RefusalNotYourTurn,
		"illegal_action":    RefusalIllegalAction,
		"unknown_message":   RefusalUnknownMessage,
	}
	if key, known := worded[name]; known {
		return l.Text(key)
	}
	return name
}

// Seat words a wire.Seat — which of a room's two places this client took.
//
// ⚠️ **A seat is worded where an id would normally travel unworded, and the
// distinction is which sort of name it is.** The house rule leaves *data* ids
// alone — a character id, a skill id, a stat label — because those are the data
// files' own keys and a reader looks them up. A seat is neither: it is a Go
// constant with exactly two values, it goes on the waiting screen inside a
// sentence, and a bare "host" in a Vietnamese column is the leak the sweeps
// hunt for on every other screen.
//
// Keyed by the name wire.Seat already writes, like its two neighbours, so
// internal/wire stays out of this package's production imports.
func (l Lang) Seat(name string) string {
	worded := map[string]Key{
		"host":  SeatHost,
		"guest": SeatGuest,
	}
	if key, known := worded[name]; known {
		return l.Text(key)
	}
	return name
}

// Closure words a wire.Closure — why a match stopped for a reason the board
// cannot show.
//
// ⚠️ **Neither closure is a loss for anybody**, and both wordings say so,
// because that is the decision wire.ClosureLeft and wire.ClosureStopped are
// both written under: on a LAN between friends the enforcement of walking away
// is social, and nothing about the host's process ending is a fact about the
// board. A wording that read "you win" here would be this end inventing a
// verdict the room deliberately did not send.
//
// Keyed by the name wire.Closure already writes, like its neighbour above.
func (l Lang) Closure(name string) string {
	worded := map[string]Key{
		"none":    ClosedNone,
		"left":    ClosedLeft,
		"stopped": ClosedStopped,
	}
	if key, known := worded[name]; known {
		return l.Text(key)
	}
	return name
}
