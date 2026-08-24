package battle

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// Kind names what an event records.
type Kind uint8

const (
	// Started opens a battle. One is emitted per unit, carrying its side, cell
	// and health, so a replay can draw the opening board without being handed
	// the battle's own state.
	Started Kind = iota
	// TurnBegan opens a unit's turn.
	TurnBegan
	// StatusTicked is damage a timed effect dealt at the start of its holder's
	// turn. It carries the status so a log can name what is killing someone.
	StatusTicked
	// StatusExpired is a timed effect running out.
	StatusExpired
	// SpeedChanged is a unit's tempo moving, which reorders the queue.
	SpeedChanged
	// TurnSkipped is a unit losing its action, to control or to death.
	TurnSkipped
	// SkillUsed is a unit committing to an action.
	SkillUsed
	// Amplified is a skill's condition holding, so it lands at higher power.
	Amplified
	// StatusConsumed is a skill spending a status for a burst.
	StatusConsumed
	// Missed is a strike failing to connect.
	Missed
	// Blocked is a strike connecting and a charge cancelling it.
	Blocked
	// Damaged is a strike landing.
	Damaged
	// StatusApplied is a timed effect taking hold.
	StatusApplied
	// StatusResisted is an application failing its own chance.
	StatusResisted
	// StatusStripped is a cleanse or a dispel removing stacks.
	StatusStripped
	// Died is a unit reaching zero health.
	Died
	// Ended closes a battle.
	Ended
)

// KindCount is the number of event kinds.
const KindCount = int(Ended) + 1

var kindNames = [KindCount]string{
	Started:        "started",
	TurnBegan:      "turn_began",
	StatusTicked:   "status_ticked",
	StatusExpired:  "status_expired",
	SpeedChanged:   "speed_changed",
	TurnSkipped:    "turn_skipped",
	SkillUsed:      "skill_used",
	Amplified:      "amplified",
	StatusConsumed: "status_consumed",
	Missed:         "missed",
	Blocked:        "blocked",
	Damaged:        "damaged",
	StatusApplied:  "status_applied",
	StatusResisted: "status_resisted",
	StatusStripped: "status_stripped",
	Died:           "died",
	Ended:          "ended",
}

func (k Kind) String() string {
	if int(k) >= KindCount {
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
	return kindNames[k]
}

// Event is one thing that happened, flat enough to serialise and complete enough
// to render from without reading the battle's state.
//
// It is one struct with a discriminant rather than a set of types behind an
// interface. A battle log is written once and read by a terminal client, a
// graphical one and the tests, and every one of those readers wants the whole
// record in a form it can switch on and store. Which fields a kind uses is
// documented on the kind; the rest stay zero.
type Event struct {
	Kind Kind
	// At is the point on the action-value timeline.
	At int64
	// Turn is the acting unit's own turn number, the unit statuses and
	// cooldowns are counted in.
	Turn int
	// Actor is the unit whose turn or action this is.
	Actor string
	// Target is the unit on the receiving end, when there is one.
	Target string
	// Skill, Status name what was used or applied.
	Skill  string
	Status string
	// Amount is damage, or a stat's new value on SpeedChanged.
	Amount int64
	// Before is the previous value, on SpeedChanged.
	Before int64
	// Stacks is how many stacks were applied, consumed or stripped.
	Stacks int
	// Strike is which strike of a multi-strike skill this was, from one.
	Strike int
	// Chance is the probability the roll was made against, in parts per
	// thousand.
	Chance int
	// Multiplier is the elemental multiplier the damage was scaled by.
	Multiplier int
	// Power is the skill power the hit resolved at, after any amplifier.
	Power int
	// Remaining is the target's health after the event, or charges left after a
	// block.
	Remaining int64
	// Cell and Side place a unit on the board.
	Cell hex.Offset
	Side hex.Side
	// Note is a short reason, used only where a kind has one to give.
	Note string
}
