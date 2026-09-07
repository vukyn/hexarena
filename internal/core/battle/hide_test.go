package battle_test

import (
	"testing"
)

// TestAHideIsPricedByTheEnemysTurnsRatherThanItsHolders is the correction the
// Diglett measurement asked for.
//
// A hiding status counts down on its HOLDER's turns, so turnsOf reports the
// holder's turns — and what a hide denies is the ENEMY's. Those are the same
// number only when the two are the same speed. A holder fast enough to take the
// whole window before the enemy moves once denies nothing at all, and was still
// charged the enemy's whole blow for every turn of it, which made hiding the
// biggest figure on the board and the only turn a fast hider ever took.
//
// Both arms are here because either alone proves nothing: an attack always
// preferred would pass the first with hiding priced at nought, and hiding always
// preferred would pass the second with the correction reverted.
func TestAHideIsPricedByTheEnemysTurnsRatherThanItsHolders(t *testing.T) {
	// Four times the enemy's speed, so the turns the hide lasts are over before
	// the enemy has taken one.
	fast := duel(t, []string{"burrow", "strike"}, []string{"strike"}, 200, 50)
	choice, ok := fast.Suggest(turnOf(t, fast, "a"))
	if !ok {
		t.Fatal("Suggest declined a turn holding a strike worth taking")
	}
	if choice.Skill != "strike" {
		t.Errorf("the fast holder chose %q; a window the enemy never acts in denies "+
			"nothing, so the strike is the turn", choice.Skill)
	}

	// The same board with the speeds swapped: the window now covers several of
	// the enemy's turns, and hiding is the turn.
	slow := duel(t, []string{"burrow", "strike"}, []string{"strike"}, 50, 200)
	choice, ok = slow.Suggest(turnOf(t, slow, "a"))
	if !ok {
		t.Fatal("Suggest declined a turn holding a hide worth taking")
	}
	if choice.Skill != "burrow" {
		t.Errorf("the slow holder chose %q; the enemy acts several times inside the "+
			"window, so hiding denies more than the strike deals", choice.Skill)
	}
}

// TestAHideDeniesAnOrdinaryTurnRatherThanTheHeaviestBlowInTheKit is the same
// correction turnWorth is already the written statement of, applied to the one
// term that had gone on reading the best attack in the kit.
//
// A denied turn is not another cast of the enemy's heaviest skill — that one is
// on cooldown most of the time — it is an ordinary turn of that enemy's. Pricing
// a hide against the heaviest blow over-charged it by the gap between the two,
// which on a kit holding one big skill and three small ones is most of the
// figure.
//
// Both arms hold the enemy's BEST blow fixed and move only the rest of its kit,
// so what is being read is the mean rather than the maximum: an enemy with
// nothing but its heaviest skill loses that skill's whole worth when it cannot
// aim, and hiding from it is still the turn.
func TestAHideDeniesAnOrdinaryTurnRatherThanTheHeaviestBlowInTheKit(t *testing.T) {
	// One skill, so an ordinary turn of this enemy's IS its heaviest blow.
	narrow := duel(t, []string{"burrow", "strike"}, []string{"strike"}, 100, 100)
	choice, ok := narrow.Suggest(turnOf(t, narrow, "a"))
	if !ok {
		t.Fatal("Suggest declined a turn holding a hide worth taking")
	}
	if choice.Skill != "burrow" {
		t.Errorf("the holder chose %q against an enemy whose every turn is its heaviest "+
			"blow; hiding denies all of it", choice.Skill)
	}

	// The same heaviest blow with three small skills beside it. Nothing about
	// what the enemy can do to this holder at its best has changed.
	broad := duel(t, []string{"burrow", "strike"},
		[]string{"strike", "jab", "clout", "daze"}, 100, 100)
	choice, ok = broad.Suggest(turnOf(t, broad, "a"))
	if !ok {
		t.Fatal("Suggest declined a turn holding a strike worth taking")
	}
	if choice.Skill != "strike" {
		t.Errorf("the holder chose %q against an enemy whose ordinary turn is a small "+
			"fraction of its best; a denied turn is worth the ordinary one", choice.Skill)
	}
}
