package battle

import (
	"encoding/json"
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
	// Healed is a unit getting health back, from a regeneration ticking, a
	// skill restoring it, or a drain returning a share of damage dealt. The log
	// has to carry it: without one, a reader sees health go up with nothing to
	// account for it.
	Healed
	// PassiveHeld is a trait taking hold, and the permanent status it put on.
	// One per granted status: with the opening board for a trait that is simply
	// in force, and again mid-battle each time a gated one comes on.
	//
	// It is its own kind rather than a StatusApplied with an empty skill,
	// because the two are different facts: one is something a unit did to
	// another and rolled a chance for, the other is what a unit simply is. A
	// renderer that drew them the same way would tell a reader a trait had just
	// been inflicted.
	PassiveHeld
	// PassiveReleased is a gated trait letting go, and the permanent status that
	// came off with it.
	//
	// A trait coming and going changes a visible number, and the log is the only
	// contract a renderer has — so the way back needs a line exactly as the way
	// in does. Without it a reader watches a unit's damage fall with nothing
	// anywhere to account for it, which is the trap Pierce and Refused were
	// added for.
	//
	// Only a gated trait ever emits one. An ungated grant is in force from the
	// opening board until the holder dies, and death is already a line.
	PassiveReleased
	// Ended closes a battle, and says in its Outcome how. Side names the winner
	// and is meaningless unless the outcome is a Victory.
	//
	// Every way a battle can stop comes through this one kind: a log that simply
	// ran out of events could not say whether the fight was won, mutually lost
	// or deadlocked, and a replay comparing two such logs would be comparing two
	// stories rather than two battles.
	Ended
	// Summoned is a unit arriving mid-battle. Actor is whoever cast, Target is
	// the unit that arrived, and Name, Cell and Side describe it the way Started
	// describes a unit the roster placed — a renderer drawing a board needs the
	// same three facts either way, and needing them from two shapes would be two
	// code paths for one thing.
	Summoned
	// Left is a summoned unit going, which is not a death.
	//
	// A death is somebody being beaten and this is a copy running out of turns or
	// losing the unit it copied, so a log spelling them the same would report a
	// fight going worse than it went. The consequence is the same — the unit stops
	// taking turns and its side may be empty without it — and only the reading
	// differs, which is exactly what a kind is for.
	Left
)

// KindCount is the number of event kinds.
const KindCount = int(Left) + 1

var kindNames = [KindCount]string{
	Started:         "started",
	TurnBegan:       "turn_began",
	StatusTicked:    "status_ticked",
	StatusExpired:   "status_expired",
	SpeedChanged:    "speed_changed",
	TurnSkipped:     "turn_skipped",
	SkillUsed:       "skill_used",
	Amplified:       "amplified",
	StatusConsumed:  "status_consumed",
	Missed:          "missed",
	Blocked:         "blocked",
	Damaged:         "damaged",
	StatusApplied:   "status_applied",
	StatusResisted:  "status_resisted",
	StatusStripped:  "status_stripped",
	Died:            "died",
	Healed:          "healed",
	PassiveHeld:     "passive_held",
	PassiveReleased: "passive_released",
	Ended:           "ended",
	Summoned:        "summoned",
	Left:            "left",
}

func (k Kind) String() string {
	if int(k) >= KindCount {
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
	return kindNames[k]
}

// MarshalJSON writes the kind by name, so a saved log does not depend on the
// order these constants are declared in.
func (k Kind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }

// UnmarshalJSON reads a kind written by name.
func (k *Kind) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("decode event kind: %w", err)
	}
	for i, candidate := range kindNames {
		if candidate == name {
			*k = Kind(i)
			return nil
		}
	}
	return fmt.Errorf("unknown event kind %q", name)
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
	Kind Kind `json:"kind"`
	// At is the point on the action-value timeline.
	At int64 `json:"at"`
	// Turn is the acting unit's own turn number, the unit statuses and
	// cooldowns are counted in.
	Turn int `json:"turn,omitempty"`
	// Actor is the unit whose turn or action this is.
	Actor string `json:"actor,omitempty"`
	// Name is the actor's display name, carried on Started so a saved log can be
	// rendered without the roster it was fought with.
	Name string `json:"name,omitempty"`
	// Target is the unit on the receiving end, when there is one.
	Target string `json:"target,omitempty"`
	// Skill, Status name what was used or applied.
	Skill  string `json:"skill,omitempty"`
	Status string `json:"status,omitempty"`
	// Amount is damage, or a stat's new value on SpeedChanged.
	Amount int64 `json:"amount,omitempty"`
	// Before is the previous value, on SpeedChanged.
	Before int64 `json:"before,omitempty"`
	// Stacks is how many stacks were applied, consumed or stripped.
	Stacks int `json:"stacks,omitempty"`
	// Strike is which strike of a multi-strike skill this was, from one.
	Strike int `json:"strike,omitempty"`
	// Chance is the probability the roll was made against, in parts per
	// thousand.
	Chance int `json:"chance,omitempty"`
	// Multiplier is the elemental multiplier the damage was scaled by.
	Multiplier int `json:"multiplier,omitempty"`
	// Power is the skill power the hit resolved at, after any amplifier.
	Power int `json:"power,omitempty"`
	// Pierce is the share of the target's defence the hit ignored, in parts per
	// thousand, on Damaged. It is here because a pierced hit that logs like an
	// ordinary one leaves the log unable to explain its own figures: a reader
	// with the same stats and the same power cannot reproduce the damage, and
	// the log is the only contract a renderer has. Absent on every hit that
	// pierces nothing, which is every hit the shipped book can produce today.
	Pierce int `json:"pierce,omitempty"`
	// Remaining is the target's health after the event, or charges left after a
	// block.
	Remaining int64 `json:"remaining,omitempty"`
	// Cell and Side place a unit on the board.
	Cell hex.Offset `json:"cell,omitempty"`
	Side hex.Side   `json:"side,omitempty"`
	// Outcome is how the battle finished, on Ended and nowhere else. Undecided
	// is the zero value, so every other kind leaves it out.
	Outcome Outcome `json:"outcome,omitempty"`
	// Passive names the trait an event came from, and is the one field that
	// says a thing happened because of what a unit *is* rather than what it
	// did: on PassiveHeld and PassiveReleased, and on the Damaged,
	// StatusApplied and StatusResisted a reply produces. Skill is empty on
	// exactly those, and the two are never both set.
	Passive string `json:"passive,omitempty"`
	// Refused is the share of a status application's chance the target's traits
	// took off, in parts per thousand, on StatusApplied and StatusResisted.
	//
	// It is on the event because Chance alone cannot say why an application was
	// unlikely: a skill that inflicts on 400 and a skill that inflicts on 800
	// against a target refusing half are the same figure by the time it is
	// rolled. And the kind is called status_resisted whether the roll failed or
	// the target refused it outright, so without this the log gives a reader the
	// word "resisted" and no way to tell which of the two it means.
	Refused int `json:"refused,omitempty"`
	// Drained is the share of the damage a strike dealt that came back as
	// health, in parts per thousand, on the Healed a drain produces.
	//
	// It is on the event for the reason Pierce and Refused are: Amount alone
	// cannot say why. A drain of six hundred off a strike that dealt three
	// hundred and a drain of three hundred off one that dealt six hundred are the
	// same number by the time they reach a reader, and once a *trait* can drain
	// as well as a skill, the skill's own figure no longer accounts for it —
	// somebody reading leech_seed at 600 and a heal of 800 has nothing anywhere
	// to explain the difference.
	Drained int `json:"drained,omitempty"`
	// AmplifiedChance and AmplifiedEffect are the shares the *actor's* traits
	// added, in parts per thousand, on StatusApplied and StatusResisted: one to
	// the chance the roll was made against, one to the tick frozen on the stack.
	//
	// Two fields because they explain two different figures on the same event.
	// Amount already carries the frozen tick, so an amplified poison reads as 260
	// where the same skill and the same stats give 201, and Chance already carries
	// the rolled figure, which a reader cannot reproduce from the skill's own.
	// Netting either into Refused would say a number moved without saying which
	// side moved it — and the whole reason Refused exists is that a share is only
	// legible when the log names who took it.
	//
	// Absent whenever nothing amplified, which is every application the shipped
	// book made until a trait declared one.
	AmplifiedChance int `json:"amplified_chance,omitempty"`
	AmplifiedEffect int `json:"amplified_effect,omitempty"`
	// Note is a short reason, used only where a kind has one to give.
	Note string `json:"note,omitempty"`
}
