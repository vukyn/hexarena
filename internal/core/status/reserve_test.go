package status_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
)

// TestACounterIsTheTwoCategoriesThatAreBoundedByTheirOwnCap is the predicate
// asserted against the enum rather than against a list written a second time.
//
// Category.Counter is what three sites ask — the cap, the modifier refusal, and
// the spend pricing — and a list spelled out at each of them is a list that will
// disagree with itself the next time somebody appends. The names are asserted
// exactly so that appending a category cannot quietly join or leave the set.
func TestACounterIsTheTwoCategoriesThatAreBoundedByTheirOwnCap(t *testing.T) {
	counters := make([]string, 0, 2)
	for _, category := range status.Categories() {
		if category.Counter() {
			counters = append(counters, category.String())
		}
	}
	if got := strings.Join(counters, ","); got != "charge,reserve" {
		t.Errorf("the counters are %q, want \"charge,reserve\"", got)
	}
}

// TestTheTwoCountersSitOnOppositeSidesOfTheBoard is the whole of what stopped a
// reserve being a charge under another name.
//
// A charge is laid on an ENEMY and cashed by hitting them, so it is Harmful: the
// victim's own side washes it off, and that is its entire cost. `rinse` is a
// shipped cleanse a squad points at its own ally naming dot, stat_debuff and
// charge. A reserve is its holder's own fuel, so the same cleanse must leave it
// alone — folding the two together would make that skill a heal that empties the
// tank it was meant to help.
func TestTheTwoCountersSitOnOppositeSidesOfTheBoard(t *testing.T) {
	if !status.Charge.Harmful() {
		t.Error("Charge stopped being harmful, so a cleanse can no longer answer a conduit")
	}
	if status.Reserve.Harmful() {
		t.Error("Reserve reports itself harmful, so a cleanse aimed at debuffs would empty its holder's own tank")
	}
	// A charge is a rider: it goes onto a target because a blow arrived, and the
	// shield that ate the blow does not stop what was left on the target. Nothing
	// puts a reserve on anybody by hitting them, so the question is one nobody
	// asks about it.
	if !status.Charge.OutlastsAShield() {
		t.Error("Charge stopped reaching through a shield, which is the trade the conduit playstyle rests on")
	}
	if status.Reserve.OutlastsAShield() {
		t.Error("Reserve claims to reach through a shield, which is a rule about a rider and a reserve is never one")
	}
}

// TestACounterIsBoundedByItsOwnCapAndCarriesNoEffect is the pair of rules the
// category exists for, asserted for BOTH counters.
//
// The cap bounds a stack count rather than an effect: five stacks of a debuff at
// three hundred per mille each is a figure the stat budget was reasoned against,
// and a counter multiplies nothing at all — so it gets a ceiling of its own, and
// a modifier on one would be a term of any size at a stack count nothing in that
// budget answers.
func TestACounterIsBoundedByItsOwnCapAndCarriesNoEffect(t *testing.T) {
	for _, category := range []string{"charge", "reserve"} {
		// Over max_stacks and under max_counter_stacks: the whole point of the
		// second number.
		if _, err := status.ParseBook([]byte(`{
		  "max_stacks": 5, "max_duration": 6, "max_counter_stacks": 40,
		  "kinds": [{"id": "tally", "category": "` + category + `", "max_stacks": 40, "duration": 4}]
		}`)); err != nil {
			t.Errorf("a %s of forty stacks under a counter cap of forty was refused: %v", category, err)
		}
		// And over the counter cap it is refused by the number it is actually
		// bounded by, named in the message so the reader is not sent to the
		// wrong one.
		_, err := status.ParseBook([]byte(`{
		  "max_stacks": 5, "max_duration": 6, "max_counter_stacks": 40,
		  "kinds": [{"id": "tally", "category": "` + category + `", "max_stacks": 41, "duration": 4}]
		}`))
		if err == nil {
			t.Errorf("a %s of forty-one stacks passed a counter cap of forty", category)
		} else if !strings.Contains(err.Error(), "max_counter_stacks") {
			t.Errorf("the %s refusal says %q, which sends the reader to the wrong cap", category, err)
		}
		// A counter that changes a stat is bounded by nothing and does something
		// anyway.
		_, err = status.ParseBook([]byte(`{
		  "max_stacks": 5, "max_duration": 6, "max_counter_stacks": 40,
		  "kinds": [{"id": "tally", "category": "` + category + `", "max_stacks": 40, "duration": 4,
		   "modifiers": [{"target": "attack", "mode": "percent", "amount": 100}]}]
		}`))
		if err == nil {
			t.Errorf("a %s carrying a stat term was accepted", category)
		} else if !strings.Contains(err.Error(), "may not also change a stat") {
			t.Errorf("the %s modifier refusal says %q", category, err)
		}
	}
}
