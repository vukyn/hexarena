package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// A burrow was worth exactly nothing to the rating. What a turn is worth to
// price.go is damage done, health restored and statuses landed, and hiding is
// none of the three — it is worth the damage that does *not* arrive. `burrowed`
// is a buff carrying no modifier, so it reached `standing`, which read its terms,
// found none and returned nought: the skill worked, its own test passed, and the
// auto-battle never once chose it. Every squad measurement including it was
// measuring a unit one skill short.
//
// `hidden` prices it the way `taunting` prices a taunt — by what it does to the
// enemy's options rather than by the holder's danger — and the two tests below
// hold the two halves of that reading.

// hidingBench is one hider facing one attacker, with a second unit on the hider's
// side whose reachability the caller chooses by placing it.
//
// Both tests build their own battle rather than sharing one: a *battle.Battle is
// the one thing a copy does not copy, so a shared fixture would have the first
// test's decision decide the second's board.
func hidingBench(t *testing.T, hiderSpeed int64, allyCell hex.Offset,
	allyStats [4]int64) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 11, []battle.Roster{
		{ID: "hider", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, hiderSpeed),
			Skills: []string{"burrow", "jab"}},
		{ID: "ally", Side: hex.SideAlly, Slot: allyCell,
			Affinity: single("neutral"),
			Stats:    stats(allyStats[0], allyStats[1], allyStats[2], allyStats[3]),
			Skills:   []string{"jab"}},
		{ID: "hitter", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4000, 500, 300, 50),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// theTurnOf advances to a named unit and fails if anybody else is up, so a
// suggestion below is always the one the test means.
func theTurnOf(t *testing.T, fight *battle.Battle, who string) *battle.Prompt {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != who {
		t.Fatalf("%s is up and this test needs %s to be", prompt.Unit, who)
	}
	return prompt
}

// whoTheAttackerWouldHit is the premise, asked in the rating's own currency
// rather than through an internal helper: build the same board with the attacker
// going first and see which cell it actually chooses.
//
// ⚠️ Its own battle, deliberately. A premise read off the board the decision is
// then taken on would have the attacker's turn already spent by the time the
// hider is asked, and the decision under test would be about a different board.
func whoTheAttackerWouldHit(t *testing.T, allyCell hex.Offset, allyStats [4]int64) hex.Offset {
	t.Helper()
	fight := hidingBench(t, 10, allyCell, allyStats)
	prompt := theTurnOf(t, fight, "hitter")
	choice, acted := fight.Suggest(prompt)
	if !acted {
		t.Fatal("the attacker passed rather than swinging, so this board has no preference " +
			"to state and the decision below would be measured against nothing")
	}
	return choice.Aim
}

// TestSuggestCastsABurrowThatDeniesTheBestBlow is the defect, stated as a
// decision rather than as a number.
//
// The ally is out of the attacker's reach, so the hider is the only thing on the
// board it can hit: going under denies the whole blow, and a rating that priced
// hiding at nought would spend the turn on `jab` instead.
func TestSuggestCastsABurrowThatDeniesTheBestBlow(t *testing.T) {
	// The ally is placed on the hider's own back column, out of a range-1 jab
	// from the attacker's cell.
	const allyCell = 0
	away, soft := hex.Offset{Col: allyCell, Row: 1}, [4]int64{4000, 500, 300, 2}

	// The premise, held rather than assumed: the attacker really would come for
	// the hider. Without it, a board where nothing threatened anybody would pass
	// the assertion below for the wrong reason.
	fight := hidingBench(t, 60, away, soft)
	hider, _ := fight.Unit("hider")
	if aim := whoTheAttackerWouldHit(t, away, soft); aim != hider.Cell {
		t.Fatalf("the attacker would swing at %v rather than at the hider's %v, so this "+
			"board is the other test's", aim, hider.Cell)
	}

	prompt := theTurnOf(t, fight, "hider")
	choice, acted := fight.Suggest(prompt)
	if !acted {
		t.Fatal("the rating passed a turn it had a skill for")
	}
	if choice.Skill != "burrow" {
		t.Errorf("the rating chose %q over burrow on a board where hiding denies the only "+
			"blow there is: hiding is being priced at nothing", choice.Skill)
	}
}

// TestSuggestDeclinesABurrowThatDeniesNothing is the other half, and it is what
// stops the term becoming "always hide".
//
// What a burrow denies is the DIFFERENCE between the enemy's best blow on its
// holder and its best blow on anybody else — an attacker that was going to hit
// the ally anyway loses nothing when the hider vanishes. A term that priced the
// whole attack would pass the test above and fail this one, which is why both are
// here.
func TestSuggestDeclinesABurrowThatDeniesNothing(t *testing.T) {
	// The ally stands beside the attacker and is softer, so it is the better
	// target whether or not the hider is on the board.
	beside, frail := hex.Offset{Col: 2, Row: 0}, [4]int64{4000, 500, 1, 2}

	fight := hidingBench(t, 60, beside, frail)
	ally, _ := fight.Unit("ally")
	if aim := whoTheAttackerWouldHit(t, beside, frail); aim != ally.Cell {
		t.Fatalf("the attacker would swing at %v rather than at the ally's %v, so this "+
			"board is the other test's and the decline below would mean nothing",
			aim, ally.Cell)
	}

	prompt := theTurnOf(t, fight, "hider")
	choice, acted := fight.Suggest(prompt)
	if !acted {
		t.Fatal("the rating passed a turn it had a skill for")
	}
	if choice.Skill == "burrow" {
		t.Error("the rating hid from a blow that was never coming at it: what a burrow " +
			"denies is the difference between the enemy's best on its holder and its best " +
			"elsewhere, and here that difference is nought")
	}
}
