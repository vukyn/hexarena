package draft

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/wire"
)

// Apply takes one recorded decision back through the decision it names, which is
// the whole of how a draft is mirrored: a client handed nothing but the entries
// out of a wire.Drafted computes exactly the draft the room holds, because a
// draft is a pure function of the decisions taken and its pool.
//
// ⚠️ **It lives here rather than beside each caller, and that was the plan
// before there was a second caller.** The switch was written in
// record_test.go — the replay proof needed it and nothing else did — with a
// comment saying that the day a client mirrored a draft there would be two
// copies and it would belong in this package. This is that day. A client holding
// the switch would be a second declaration of the draft's own sequence, which is
// the mistake cast.ChooseLoadout was pulled together out of three copies of. →
// CLAUDE.md § "One rule, one declaration".
//
// ⚠️ **Every refusal is the state machine's own**, unreworded: this function
// routes and does not judge. A decision out of turn, of the wrong step, naming a
// character out of the pool, an illegal loadout and a refused arrangement all
// come back in the words Ban, Pick, Loadout, Arrange and TimedOut already use —
// so a mirror that diverged is told why in the same sentence the room would use,
// and there is no second reading of the rules to drift from the first.
//
// ⚠️ **The arrange phase replays one entry at a time and still ends in the same
// place**, which is worth knowing before reading a cursor. The two arrangements
// are recorded together in seats order, so applying the first one puts the
// *host's* arrangement in however the two really arrived — a mirror is therefore
// briefly waiting on the seat the room was not, and the two agree again the
// moment the second is applied. The phase's *end* replays exactly; its middle is
// not a fact two peers could agree about, since arrival order is a race. →
// Draft.Arrange, and TestARecordReplaysIntoTheSameDraft, which defers exactly one
// comparison and counts the deferral.
//
// An unknown step is an error rather than a decision quietly dropped: a mirror
// that skipped a decision it did not recognise would carry on computing a draft
// that is no longer the room's, which is the silent desync every part of this
// package is arranged against.
func (d *Draft) Apply(entry wire.DraftEntry) error {
	switch entry.Step {
	case wire.StepBan:
		// A ban that names nobody is the skip, and that absence is the decision
		// rather than a missing field. → wire.DraftDecision.Character.
		if entry.Character == "" {
			return d.SkipBan(entry.Seat)
		}
		return d.Ban(entry.Seat, entry.Character)
	case wire.StepPick:
		return d.Pick(entry.Seat, entry.Character)
	case wire.StepLoadout:
		// The form as it was **named**, which is what the record keeps: the
		// resolution is this draft's to compute, and computing it here again from
		// the same input is what makes the mirror a mirror.
		return d.Loadout(entry.Seat, entry.Stage, entry.Skills, entry.Passives)
	case wire.StepArrange:
		return d.Arrange(entry.Seat, entry.Slots)
	case wire.StepTimeout:
		return d.TimedOut(entry.Seat)
	}
	return fmt.Errorf("this record holds a %q, which is not one of the %d decisions a draft "+
		"can be told: a mirror cannot compute a draft it cannot replay",
		entry.Step, len(wire.DraftSteps()))
}
