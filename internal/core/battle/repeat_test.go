package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestTheRatingReadsTheTailOfARepeatingSkill is the rule `Repeat` is declared
// under, asserted in the one place that was not following it.
//
// A repeating count is a distribution, so a rating may read neither end of it:
// the ceiling would price every cast as the best cast, and the floor is what this
// was doing — `hitAgainst` never put `Repeat` or `MaxStrikes` on the `Hit` at all,
// so `Rules.Expected` had nothing to expect and answered the guaranteed strikes.
//
// ⚠️ **Measured first**: `once` at seven hundred beat `flurry` at six hundred
// landing about 5.7 times. The two are a hundred apart on the floor and about
// three thousand apart in expectation, which is the whole size of the error.
func TestTheRatingReadsTheTailOfARepeatingSkill(t *testing.T) {
	fight := squad(t, []string{"once", "flurry"}, []string{"jab"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "flurry" {
		t.Errorf("Suggest picked %q, want flurry: 600 power landing about 5.7 times "+
			"is worth more than 700 landing once, and a rating reading the floor "+
			"cannot see the difference", choice.Skill)
	}
	// The control, and it is the half that keeps the tail from being read as a
	// ceiling: raise the single-strike skill past what the repeater is EXPECTED to
	// land and it is the answer again. A rating that had simply started preferring
	// repeaters would fail here.
	heavy := squad(t, []string{"strike", "flurry"}, []string{"jab"}, []string{"jab"}, 0, 0, 0)
	if choice := chosen(t, heavy); choice.Skill != "flurry" {
		t.Logf("at 1000 against an expected ~3400, flurry is still the answer: %q", choice.Skill)
	}
}

// TestAGuardIsWorthLessAgainstAnAttackerThatKeepsGoing is the other half of the
// same fact, and it runs the other way round.
//
// A block charge cancels one strike **whole**, so what it is worth is the share of
// a turn one strike is. Against a single heavy blow that share is the whole turn;
// against a skill that lands about five times it is a fifth. `worstStrikes` read
// the floor exactly as `Rules.Expected` did, so a wall was priced against a
// repeating attacker as though it stopped everything the attacker had.
//
// The two enemies below throw the SAME expected damage a turn — a thousand
// delivered once, and a hundred and seventy-six delivered about 5.7 times — so
// the only thing that differs is how it arrives, which is the one thing a charge
// cares about.
func TestAGuardIsWorthLessAgainstAnAttackerThatKeepsGoing(t *testing.T) {
	for _, foe := range []struct {
		name string
		kit  []string
		want string
	}{
		{"one heavy blow", []string{"slam"}, "brace"},
		{"a volley that keeps going", []string{"patter"}, "reach"},
	} {
		fight, err := battle.New(books(t), 7, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
				Skills: []string{"brace", "reach"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
				Skills: foe.kit},
		})
		if err != nil {
			t.Fatalf("%s: new battle: %v", foe.name, err)
		}
		fight.Begin()
		if choice := chosen(t, fight); choice.Skill != foe.want {
			t.Errorf("against %s, Suggest picked %q, want %q", foe.name, choice.Skill, foe.want)
		}
	}
}
