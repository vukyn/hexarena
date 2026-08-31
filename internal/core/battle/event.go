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
	// Spread is a skill covering a wider shape than the one it declares, because
	// the unit at the aim was carrying the status the skill spends.
	//
	// One per use, emitted before any cell resolves, and it names the carrier the
	// shape was chosen from. It has to exist: a reader watching a skill whose book
	// entry says "single" land on three units has nothing anywhere to account for
	// it, and the log is the only contract a renderer has. It is the same gap
	// Pierce and Refused were added to close, arriving as a kind rather than a
	// field because the fact is about the *use* rather than about one strike.
	//
	// It is not an Amplified. That one says a figure went up, and a spread moves
	// no figure at all — the power is what it always was and the shape is what
	// changed, so a log spelling them the same would have a reader hunting for a
	// bonus that is not there.
	//
	// Declared last, which is the rule for this enum: the kind serialises by name,
	// so appending cannot reinterpret a saved log, while slotting one in beside
	// Amplified would move KindCount and every table built from declaration order.
	Spread
)

// KindCount is the number of event kinds.
const KindCount = int(Spread) + 1

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
	Spread:          "spread",
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
	// Critical says the strike landed critically, on Damaged.
	//
	// Two things about it are worth writing down. --verify compares whole events
	// with ==, and a bool is comparable, so nothing there changes and no log
	// needs a reader that knows about this. And omitempty drops a false bool, so
	// every log written before this field existed is byte for byte what it was:
	// the format did not break a third time.
	//
	// It is a flag where Pierce, Refused, Drained and the two Amplified fields
	// are permille integers, and the difference is deliberate rather than an
	// inconsistency. Those five are shares that vary from cast to cast, so a
	// reader cannot reproduce the figure without being told which share applied.
	// This one multiplier is a single game-wide constant on combat.Rules, the
	// same on every critical strike ever logged — writing it onto each of them
	// would be the constant restated once per event, and a reader who wants it
	// reads the rules.
	Critical bool `json:"critical,omitempty"`
	// Remaining is the target's health after the event, or a stack count left
	// behind: charges after a block, and stacks after a consume that took only
	// some of them. The two spellings are one meaning — what is left of the thing
	// the event just spent — and a consume that takes the whole pile leaves nought,
	// which omitempty drops exactly as it always did.
	Remaining int64 `json:"remaining,omitempty"`
	// Cell and Side place a unit on the board. Most kinds place nobody, so the
	// cell is optional rather than a coordinate every event has to supply: see
	// hex.Cell for why the absence needs a field of its own.
	Cell hex.Cell `json:"cell,omitzero"`
	Side hex.Side `json:"side,omitempty"`
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
	// Reduced is the share of the healing that was taken off before it landed, in
	// parts per thousand, on Healed: what the receiving unit's own heal-cut
	// statuses cost it.
	//
	// It is on the event for the reason Pierce, Refused and Drained are, and this
	// one is the starkest of the four: without it a reader sees `heals 244` where
	// the book says nine hundred, and every figure they could check it against —
	// the skill's restores, the status's tick power, the drain share already on
	// this event — says the log is wrong. A log its own reader cannot reproduce is
	// the log lying.
	//
	// ⚠️ **A field of its own rather than a second meaning on Refused.** Refused
	// is a share of a status application's *chance*, signed, where a negative
	// means the target invited the status — two readings on one field is the
	// mistake this file keeps a list of, and a heal cut netted into it would be a
	// number about healing filed under a word about rolls.
	//
	// Positive means healing was lost. The share taken rather than the multiplier
	// applied, so nought means nothing happened and omitempty drops it — which is
	// why every log written before a heal-cut status existed is byte for byte what
	// it was.
	Reduced int `json:"reduced,omitempty"`
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
	// Gradient is the share the caster's own wounds added to its skill's power,
	// in parts per thousand, on SkillUsed.
	//
	// It is on the event for the reason Pierce, Refused and Drained are: the
	// numbers that follow cannot be reproduced without it. Power on the same
	// event carries what the skill *declares*, which is the figure a reader has
	// in front of them from the book — so a hurt caster's strike lands for more
	// than the book says and nothing anywhere accounts for the difference. It is
	// worse than a pierce in one respect: a pierce is a property of the skill and
	// the same on every cast, where this changes every time the caster is hit.
	//
	// The share added rather than the multiplier, so nought means nothing
	// happened and omitempty drops it — which is why every log written before a
	// skill declared a gradient is byte for byte what it was.
	//
	// It sits on SkillUsed rather than on the strike because it is read once per
	// use: a column that catches three units swings once, not three times, and an
	// event per target would say otherwise.
	Gradient int `json:"gradient,omitempty"`
	// Note is a short reason, used only where a kind has one to give.
	Note string `json:"note,omitempty"`
}
