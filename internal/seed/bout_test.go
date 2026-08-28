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

// TestNothingWaitsOnPurpose is the executable half of "waiting is arithmetically
// empty in this engine".
//
// A turn is skipped for exactly three reasons, and every one of them is forced: a
// unit died to a timed effect before it could act, control took its turn, or it had
// nothing it could use. None of those is a decision. If a waiting rule is ever
// added, it will pass a turn on purpose and that pass will carry a note of its own
// — a fourth reason — so this test is what names it the day it appears.
//
// It plays the shipped roster rather than a fixture because the claim is about the
// opponent the game ships, and it is here rather than in internal/core/battle for
// the layer rule: a test in that package may not reach for the seed data.
func TestNothingWaitsOnPurpose(t *testing.T) {
	// The three notes a skipped turn may carry. "died" and the stun are emitted by
	// Advance; NoActionReason is what RunToEnd passes with when a unit has nothing
	// usable. A unit whose summon ran out leaves rather than skipping, and emits no
	// TurnSkipped at all.
	forced := map[string]bool{"died": true, "stun": true, battle.NoActionReason: true}

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
			if !forced[event.Note] {
				t.Fatalf("seed %d: %s skipped a turn for %q, which is not one of the "+
					"three forced reasons: a turn given up on purpose is waiting, and "+
					"waiting is decided against — see CLAUDE.md § Rating an action",
					seedValue, event.Actor, event.Note)
			}
		}
	}
	// A run in which nothing was ever skipped would pass the loop above without
	// having looked at anything.
	if len(skipped) == 0 {
		t.Error("no turn was skipped at all over 200 battles, so nothing was checked")
	}
	t.Logf("skipped turns by reason: %v", skipped)
}
