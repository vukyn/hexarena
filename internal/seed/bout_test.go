package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/seed"
)

// boutSeeds is how many seeds the head-to-head below is fought over.
//
// Modest on purpose, because it runs in `make check`: 500 seeds is 1000 battles
// per arrangement pair and a two-sigma band of about 32 parts per thousand, which
// is wide enough to be honest about and narrow enough to catch a rating that has
// stopped being better than picking the first thing it can. The figure quoted in
// CLAUDE.md and README.md is taken by hand at 10,000 seeds; this is the guard, not
// the measurement.
const boutSeeds = 500

// shippedBoard is the books and the roster the game ships, which is the board
// every other figure in this repository is quoted on.
func shippedBoard(t *testing.T) (battle.Books, []battle.Roster) {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("assemble the books: %v", err)
	}
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("read the roster: %v", err)
	}
	return books, roster
}

// TestTheRatingBeatsPickingTheFirstThingItCan is the first true claim anybody has
// made about Suggest.
//
// Every figure quoted about the opponent until now was a **roster** win rate, and
// a roster win rate cannot see a rating at all: both sides use Suggest, so a change
// that helps both leaves the rate where it was and what moves is whichever squad's
// kit had more to gain. The only way to ask whether a rating is any good is to
// fight it against a different one, which is what forge.Bout does — and what it
// fights against here is the frozen ruler, Suggest as it was before any pricing
// landed.
//
// The bar is the band and not nought. A rate inside `500 ± Band` is a wobble, so
// clearing it is the claim: over the shipped roster, fought from both ends, the
// priced rating wins more than the unpriced one by more than the measurement's own
// width.
func TestTheRatingBeatsPickingTheFirstThingItCan(t *testing.T) {
	books, roster := shippedBoard(t)
	report, err := forge.Bout(books, roster, forge.Suggesting, forge.FirstUsable, boutSeeds)
	if err != nil {
		t.Fatalf("fight the rating against the ruler: %v", err)
	}
	t.Logf("Suggest against FirstUsable over %d seeds (%d battles): %s, band ±%d‰, "+
		"median %d turns, %+v",
		report.Seeds, report.Challenger.Battles(), forge.PercentInColumn(report.Rate()),
		report.Band, report.Turns, report.Challenger)
	if floor := scale.Base/2 + report.Band; report.Rate() <= floor {
		t.Errorf("Suggest comes to %d against a rating that takes the first thing it can, "+
			"which does not clear the even split plus the band (%d): the whole pricing "+
			"layer is either not helping or not being reached",
			report.Rate(), floor)
	}
}

// TestATurnIsGivenUpOnlyForAReasonThatIsWrittenDown is what
// TestNothingWaitsOnPurpose became, and the change of name is the change of
// claim.
//
// It used to hold that a turn is skipped for exactly three reasons and every one
// of them is FORCED — a unit died to a timed effect before it could act, control
// took its turn, or it had nothing it could use — and its own comment said that
// if a waiting rule were ever added it would pass a turn on purpose, carry a note
// of its own, and be named here the day it appeared. That day has arrived.
//
// ⚠️ **The pass that now exists is not the waiting TODO.md decided against**, and
// keeping the two apart is the point of this comment. That entry is about passing
// to get a skill back sooner, and it is still arithmetically empty: spendCooldowns
// runs on a pass and on an act alike, so the skill comes back on the same turn
// either way. This is the opposite question — whether to START a cooldown on a
// skill that buys nothing — and there the pass is strictly better, because an act
// pays its own cooldown and the pass does not.
//
// So the note set is asserted rather than the count, and DeclinedReason has to
// actually appear: a rule nothing on the shipped roster reaches is a rule this
// test is not measuring.
func TestATurnIsGivenUpOnlyForAReasonThatIsWrittenDown(t *testing.T) {
	// "died" and the stun are emitted by Advance; the other two are what RunToEnd
	// passes with — one for a unit with no move, one for a unit that had one and
	// would not take it. A unit whose summon ran out leaves rather than skipping,
	// and emits no TurnSkipped at all.
	written := map[string]bool{
		"died": true, "stun": true,
		battle.NoActionReason: true, battle.DeclinedReason: true,
	}

	skipped := map[string]int{}
	for seedValue := uint64(0); seedValue < 200; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: assemble: %v", seedValue, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		for _, event := range fight.Drain() {
			if event.Kind != battle.TurnSkipped {
				continue
			}
			skipped[event.Note]++
			if !written[event.Note] {
				t.Fatalf("seed %d: %s skipped a turn for %q, which is not one of the "+
					"reasons this engine records: a pass nobody can name is a rating "+
					"quietly getting worse — see CLAUDE.md § Rating an action",
					seedValue, event.Actor, event.Note)
			}
		}
	}
	// A run in which nothing was ever skipped would pass the loop above without
	// having looked at anything.
	if len(skipped) == 0 {
		t.Error("no turn was skipped at all over 200 battles, so nothing was checked")
	}
	// And the new one has to be reached, or this test is holding a claim about a
	// branch the shipped roster never takes.
	if skipped[battle.DeclinedReason] == 0 {
		t.Errorf("no turn was declined over 200 battles: the rule that refuses to "+
			"spend a cooldown on nothing is either unreachable on the shipped roster "+
			"or gone, and %v is all that happened", skipped)
	}
	t.Logf("skipped turns by reason: %v", skipped)
}
