package battle

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// Log is a complete record of one battle: the seed its rolls came from, the
// choices that were made, and everything that happened.
//
// The seed and the choices together are enough to reproduce the battle from
// nothing, which is what makes a log verifiable rather than merely readable: a
// reader can re-run the engine and check that the events it produces are the
// ones the file claims. A log that could only be read would be a story about a
// battle; this one is the battle.
type Log struct {
	Seed uint64 `json:"seed"`
	// Roster is the placement the battle was fought with: every unit, resolved.
	//
	// It is here because a placement became a *choice*. While a roster came out
	// of the embedded data and nothing about it was decided, re-running a log
	// meant loading that data again and the two were the same battle by
	// construction. Now that a placement picks four skills out of nine and one
	// trait out of two, a log without it cannot be re-run at all — --verify
	// would be comparing two different fights and reporting the difference as
	// corruption.
	//
	// It carries the resolved form rather than the reference, which has the side
	// benefit the reference never had: a log is now readable across a data edit.
	// Retuning a stat curve or moving a skill's learn level no longer
	// invalidates every log written before it.
	Roster []Roster `json:"roster,omitempty"`
	// Choices are the actions taken, in order, including the ones the engine
	// picked for itself. Replaying them reproduces the fight exactly.
	Choices []Decision `json:"choices"`
	Events  []Event    `json:"events"`
}

// Replayable reports whether the log carries the placement its battle was fought
// with, which is what re-running it needs.
//
// A log written before placements were recorded renders perfectly — that reads
// the events and nothing else — and cannot be verified, because there is nothing
// to say which four of nine skills each unit brought. Saying so is better than
// re-running the shipped roster against it and calling the mismatch corruption.
func (l Log) Replayable() bool { return len(l.Roster) > 0 }

// Decision is one action as it was taken. A passed turn carries no skill, and
// so carries nowhere to point it either.
type Decision struct {
	Unit  string `json:"unit"`
	Turn  int    `json:"turn"`
	Skill string `json:"skill,omitempty"`
	// Aim is where the skill was pointed, and is absent on a turn that was given
	// up rather than spent — a pass aiming at the ally back corner is a fact the
	// log never observed. See hex.Cell.
	Aim hex.Cell `json:"aim,omitzero"`
	// Passed and Reason record a turn given up rather than spent. The reason is
	// part of the decision rather than something the caller supplies, because
	// two callers supplying different words for the same choice would make a
	// replay diverge from the log it is replaying.
	Passed bool   `json:"passed,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// PassReason is the note a passed turn records, defaulting for a decision that
// did not say.
func (d Decision) PassReason() string {
	if d.Reason == "" {
		return "passed"
	}
	return d.Reason
}

// MarshalLog encodes a log. It never touches the filesystem; the caller writes
// the bytes wherever they belong.
func MarshalLog(log Log) ([]byte, error) {
	raw, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode battle log: %w", err)
	}
	return append(raw, '\n'), nil
}

// ParseLog decodes a log and checks it is self-consistent enough to replay.
func ParseLog(raw []byte) (Log, error) {
	var log Log
	if err := json.Unmarshal(raw, &log); err != nil {
		return Log{}, fmt.Errorf("decode battle log: %w", err)
	}
	if len(log.Events) == 0 {
		return Log{}, fmt.Errorf("the log records no events")
	}
	for i, decision := range log.Choices {
		if decision.Unit == "" {
			return Log{}, fmt.Errorf("choice %d names no unit", i)
		}
		if decision.Passed && decision.Skill != "" {
			return Log{}, fmt.Errorf("choice %d both passes and uses %q", i, decision.Skill)
		}
		if !decision.Passed && decision.Skill == "" {
			return Log{}, fmt.Errorf("choice %d neither passes nor names a skill", i)
		}
		// A taken turn without an aim used to be unrepresentable: the coordinate
		// was always there, and a missing one read as the ally back corner. Now
		// that it can be absent, a replay would point the skill at that corner and
		// call whatever came out a faithful re-run.
		if _, aimed := decision.Aim.Offset(); !decision.Passed && !aimed {
			return Log{}, fmt.Errorf("choice %d uses %q but aims nowhere", i, decision.Skill)
		}
	}
	return log, nil
}

// Script is a sequence of decisions to be replayed in order.
type Script []Decision

// Replay plays a script back into a battle, using the fallback to decide any turn
// the script does not cover. The fallback is a Chooser — the type Suggest
// satisfies — so a log can be finished by whatever rating the caller is holding. It returns the decisions actually taken and, when it
// stopped because a turn needs a decision it has none for, the prompt it stopped
// on. That prompt is nil when the battle ended or the turn limit was reached.
//
// A skipped turn is not a decision. Control, a timed effect landing the last blow
// and a unit with nothing usable are all forced, so the replay walks through them
// on its own rather than expecting the script to record them. Requiring the script
// to carry them would mean a log that ends on a poison tick could not be replayed
// to its own conclusion.
//
// This is what an undo is built on. Rather than snapshotting and restoring a
// battle's state, a client drops the last decision and replays what is left:
// because the engine is deterministic, the result is the same state the battle was
// actually in, and nothing has to be deep copied for it to work. Handing back the
// pending prompt is what saves the caller from advancing a turn that has already
// begun.
func (b *Battle) Replay(script Script, limit int, fallback Chooser) (Script, *Prompt, error) {
	taken := Script{}
	for turns := 0; !b.finished && turns < limit; turns++ {
		// A turn already open is taken up rather than advanced past, so a replay
		// that stopped for want of a decision can be resumed by handing it more
		// script.
		prompt := b.Pending()
		if prompt == nil {
			opened, err := b.Advance()
			if err != nil {
				return taken, nil, err
			}
			prompt = opened
		}
		if prompt.Skipped {
			continue
		}
		var decision Decision
		switch {
		case len(script) > 0:
			decision, script = script[0], script[1:]
			if decision.Unit != prompt.Unit {
				return taken, nil, fmt.Errorf("the script expects %q to act but the battle offered %q",
					decision.Unit, prompt.Unit)
			}
		case fallback != nil:
			choice, ok := fallback(prompt)
			decision = Decision{
				Unit: prompt.Unit, Turn: prompt.Turn,
				Skill: choice.Skill, Passed: !ok,
			}
			if ok {
				decision.Aim = hex.At(choice.Aim)
			} else {
				decision.Reason = passReason(prompt)
			}
		default:
			// The script is spent and nothing decides for it, so the turn that
			// has just begun belongs to the caller.
			return taken, prompt, nil
		}
		if decision.Passed {
			if err := b.Pass(decision.PassReason()); err != nil {
				return taken, nil, err
			}
		} else {
			aim, aimed := decision.Aim.Offset()
			if !aimed {
				return taken, nil, fmt.Errorf("replaying %q for %q: the decision aims nowhere",
					decision.Skill, decision.Unit)
			}
			if err := b.Act(decision.Skill, aim); err != nil {
				return taken, nil, fmt.Errorf("replaying %q for %q: %w", decision.Skill, decision.Unit, err)
			}
		}
		taken = append(taken, decision)
	}
	return taken, nil, nil
}
