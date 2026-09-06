package screen

import (
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The aim list used to open only on `len(option.Aims) > 1`, so a skill with one
// legal cell committed the turn on the keystroke that chose the skill. The rule
// read well — a question with one answer is not a decision — and it is wrong in
// the one case that matters: the aim list is where a skill's footprint is drawn,
// so skipping it because the CHOICE was forced also skips the READING, and the
// turn is gone by the time anybody notices.
//
// ⚠️ **The goldens do not measure this and cannot.** Every fixture that draws the
// aim list sets `p.Aiming = true` by assignment rather than by pressing a key —
// screens_golden_test.aBattleAiming, and the same shape in both TUI suites — so a
// golden holds the aiming *state* and says nothing about the path to it. The
// TODO entry predicted this change would move "every play fixture that casts a
// single-aim skill"; it moved none, and that is not the change being small, it is
// the goldens being blind to it. What holds the rule is here and in
// play_test.takingATurn.

// TestTheAimListOpensForASkillWithOneLegalCell is the reversal, stated on the
// case it was reversed for.
//
// It fails rather than skips when the fixture battle offers no single-aim option,
// because that is the state in which this test measures nothing while staying
// green — the option list is built from the shipped roster and a balance change
// could quietly take the case away.
func TestTheAimListOpensForASkillWithOneLegalCell(t *testing.T) {
	c, _ := start(t, i18n.En)
	base := atTheBattle(t, c)
	var measured int
	for index, option := range base.Pending.Options {
		if !option.Available() || len(option.Aims) != 1 {
			continue
		}
		measured++
		p := base
		p.Option = index
		chosen := playing(t, c, p, "enter")
		if !chosen.Aiming {
			t.Errorf("%s has one legal cell and choosing it did not open the aim list, "+
				"so the turn was spent on the keystroke that picked the skill", option.Skill)
		}
		if len(chosen.Script) != len(base.Script) {
			t.Errorf("%s spent the turn while opening the aim list: the script grew from %d to %d",
				option.Skill, len(base.Script), len(chosen.Script))
		}
		// And the second keystroke is what casts it, so the extra question costs
		// exactly one press and does not strand the player in the list.
		cast := playing(t, c, chosen, "enter")
		if len(cast.Script) <= len(base.Script) {
			t.Errorf("%s: answering the aim list spent no turn", option.Skill)
		}
	}
	if measured == 0 {
		t.Fatal("no option in the fixture battle has exactly one legal cell, so the case " +
			"this test exists for was not reached")
	}
}

// TestALiveAimListOpensForASkillWithOneLegalCell is the same rule on the other
// path, and it exists because the mutation proved nothing held it.
//
// PlayScreen.answer is the live twin of choose — the same two questions, answered
// with an Action instead of with a call into the engine — and reverting *it* alone
// to "one cell does not ask" reddened not one test in the repository. The
// neighbouring live test presses enter in a `for action.Kind == Stay &&
// struck.Aiming` loop, which is exactly the shape that tolerates either answer:
// it asserts a decision arrives eventually and says nothing about how many
// questions were asked on the way. A loop written to be robust against a rule is
// a loop that cannot measure it.
func TestALiveAimListOpensForASkillWithOneLegalCell(t *testing.T) {
	c, _ := start(t, i18n.En)
	// One a side, not the three the neighbouring live tests use: a single-aim
	// option is the whole subject here, and on a three-a-side board every skill
	// the opener offers can be pointed at more than one body.
	fight, prompt := aBattleNobodyHereDrives(t, c, 1)
	base := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt, Seed: 7})
	if !base.Live || base.Pending == nil {
		t.Fatal("the screen did not attach to the open turn")
	}
	var measured int
	for index, option := range base.Pending.Options {
		if !option.Available() || len(option.Aims) != 1 {
			continue
		}
		measured++
		p := base
		p.Option = index
		chosen, action := asking(t, c, p, "enter")
		if action.Kind != Stay {
			t.Errorf("%s has one legal cell and choosing it asked for a %v; the first "+
				"keystroke opens the aim list, it does not answer the turn",
				option.Skill, action.Kind)
		}
		if !chosen.Aiming {
			t.Errorf("%s has one legal cell and choosing it did not open the aim list",
				option.Skill)
		}
		answered, action := asking(t, c, chosen, "enter")
		if action.Kind != Answer || !action.Answer.Acted {
			t.Errorf("%s: answering the aim list asked for a %v, want an Answer that acted",
				option.Skill, action.Kind)
		}
		if !answered.Answered {
			t.Errorf("%s: the turn was answered and the screen did not mark it", option.Skill)
		}
	}
	if measured == 0 {
		t.Fatal("no option on the live board has exactly one legal cell, so the case this " +
			"test exists for was not reached")
	}
}
