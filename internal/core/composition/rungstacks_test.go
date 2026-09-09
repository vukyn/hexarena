package composition_test

import (
	"fmt"
	"strings"
	"testing"
)

// TestARungMayNotGrantMoreStacksThanTheStatusHolds is the other half of the
// ceiling rungceiling_test.go holds, one layer down.
//
// **A rung nobody reaches is refused; so is a rung everybody reaches for
// nothing.** parseBonus already checks that the status exists, that it is
// permanent and that the count is at least one, and it used to stop there. A
// grant is applied through status.Set.Hold, which loops Apply, and Apply refuses
// a stack past the kind's cap — so a rung declaring three stacks of a max-two
// status parsed, loaded, drew its row on the reference screen, fired on the
// board, emitted its bonus_held, and handed out exactly what the rung below it
// did.
//
// ⚠️ **That is the harder failure of the two, and it is why this test exists
// beside the other one rather than inside it.** A rung above hex.MaxSquadSize is
// visible to anything that walks the book against the cast —
// TestEveryElementBonusRungIsReachable does exactly that. A rung whose grant is
// clamped away is invisible to every such walk: the count reaches it, the rung
// fires, the log says so, and only the stack column on a status snapshot would
// have told anybody. It is DAT-002 decision 6's vacuous row wearing the clothes
// of a working one.
//
// Three clauses, and the third is the one a regression shows up in:
//
//   - a grant AT the kind's cap parses. It is the shape every shipped rung has —
//     the two-rung element bonuses grant one stack then two of a max-two status.
//   - a grant one above the cap is refused, and the refusal says the rest is
//     clamped away rather than naming a number with no reason beside it.
//   - ⚠️ **the bound is the KIND's cap and not the book's ceiling.** The fixture
//     book allows five, so a guard reading the book's number would let three
//     stacks of a max-two kind through and both of the first two clauses would
//     stay green. resolve is capped at one against the same book ceiling of five,
//     which is what makes that difference measurable here.
func TestARungMayNotGrantMoreStacksThanTheStatusHolds(t *testing.T) {
	// One rung at the smallest legal threshold, granting one status however many
	// times the case is about.
	book := func(status string, stacks int) string {
		return fmt.Sprintf(`{"bonuses": [
		  {"id": "kin", "name": "đồng hệ", "axis": "element", "scope": "sharers", "rungs": [
		    {"at": 2, "grants": [{"status": %q, "stacks": %d}]}
		  ]}
		]}`, status, stacks)
	}

	for _, allowed := range []struct {
		status string
		stacks int
	}{
		// kinship holds two, which is the shape the shipped table is written in.
		{"kinship", 1},
		{"kinship", 2},
		// resolve holds one, and one is therefore all a rung may ask of it.
		{"resolve", 1},
	} {
		if _, err := parse(t, book(allowed.status, allowed.stacks)); err != nil {
			t.Errorf("a rung granting %s %d times was refused, and the status holds that many: %v",
				allowed.status, allowed.stacks, err)
		}
	}

	for _, refused := range []struct {
		status string
		stacks int
		holds  int
	}{
		{"kinship", 3, 2},
		// ⚠️ resolve at two is the clause that separates the kind's cap from the
		// book's: the fixture book allows five, so a guard reading the ceiling
		// instead of the kind would accept this and nothing else here would move.
		{"resolve", 2, 1},
		{"resolve", 5, 1},
	} {
		_, err := parse(t, book(refused.status, refused.stacks))
		if err == nil {
			t.Errorf("a rung granting %s %d times parsed and the status holds %d: "+
				"the rest is clamped away by Set.Hold, so the rung fires and changes nothing",
				refused.status, refused.stacks, refused.holds)
			continue
		}
		if !strings.Contains(err.Error(), "clamped away") {
			t.Errorf("the refusal of %s at %d stacks does not say what it prevents: %v",
				refused.status, refused.stacks, err)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("%d is all that status holds", refused.holds)) {
			t.Errorf("the refusal of %s at %d stacks does not name the cap it read (%d): %v",
				refused.status, refused.stacks, refused.holds, err)
		}
	}
}
