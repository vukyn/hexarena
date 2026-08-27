package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The column the two allies stand in, and the enemy that stands opposite.
//
// Rows 0 and 1 of the same formation column, which is what a column-shaped skill
// aimed at either of them covers: the aim cell plus the cell above and the cell
// below, and one of those two is off the board from row 0.
var (
	rallyCaller = hex.Offset{Col: 2, Row: 0}
	rallyBeside = hex.Offset{Col: 2, Row: 1}
	// The enemy's formation slot is chosen so that hex.Place puts it *adjacent
	// to the caller*, which is what makes "allies only" checkable at all.
	//
	// The first version of this test stood it opposite instead, and a mutation
	// found the hole: a rally that buffed both sides passed, because a column
	// never changes column and the enemy was two cells away in another one. The
	// skill's range and shape both excluded it whatever its targeting said, so
	// the test was asserting the geometry rather than the rule. From here the
	// enemy is one cell away and inside a column, so the only thing still
	// keeping the rally off it is the word "ally".
	rallyOpposed = hex.Offset{Col: 2, Row: 2}
)

// TestARallyReachesAWholeColumnOfAlliesAndNobodyElse is what the skill was
// authored for, asserted rather than assumed.
//
// Everything else in the shipped book aims at an enemy or at the caster alone,
// so this is the first skill whose whole point is that it lands on somebody the
// caster is not and did not attack. Four separate things have to be true for
// that to work, and none of them is checked anywhere else: an ally-aimed skill
// may be pointed at the caster's own cell, its shape may cover a second ally
// from there, an application on a zero-power skill still lands, and none of it
// crosses the midline.
func TestARallyReachesAWholeColumnOfAlliesAndNobodyElse(t *testing.T) {
	fight := aRallyingSquad(t)
	fight.Begin()

	prompt := theTurnOf(t, fight, "caller")
	aim, found := theAimAt(prompt, "rally", rallyCaller)
	if !found {
		t.Fatalf("an ally-aimed skill cannot be pointed at the caster's own cell %s, "+
			"so a rally can never include whoever called it", rallyCaller)
	}
	if err := fight.Act("rally", aim); err != nil {
		t.Fatalf("rally: %v", err)
	}

	rallied := map[string]int{}
	for _, event := range fight.Drain() {
		if event.Kind == battle.StatusApplied && event.Status == "fury" {
			rallied[event.Target] += event.Stacks
		}
	}
	// The caller and the ally beside it, and nothing else on the board. The
	// enemy is checked by name rather than by a count, so a rally that reached
	// across the midline fails with the name of who it hit.
	for _, who := range []string{"caller", "beside"} {
		if rallied[who] != 1 {
			t.Errorf("%s came out of the rally with %d stacks of fury, want 1", who, rallied[who])
		}
	}
	if stacks, hit := rallied["opposed"]; hit {
		t.Errorf("the rally put %d stacks of fury on the enemy", stacks)
	}

	// And the aim itself is refused, not merely unused. The enemy stands one
	// cell from the caller and inside a column, so a rally that aimed at
	// everybody could be pointed at it — an offer that never lands is still an
	// offer a player can take.
	enemyCell := hex.Place(hex.SideEnemy, rallyOpposed)
	if caller, known := fight.Unit("caller"); !known || caller.Cell.DistanceTo(enemyCell) != 1 {
		t.Fatalf("the enemy is not one cell from the caller, so this asserts nothing about targeting")
	}
	if aim, offered := theAimAt(prompt, "rally", enemyCell); offered {
		t.Errorf("a rally aimed at allies was offered the enemy's cell %s", aim)
	}
}

// TestARalliedAllyReallyHitsHarder is the other half, and it is a separate test
// because a status that is applied and does nothing would pass the one above.
//
// It reads the resolved attack rather than a damage figure: damage would also
// move if the defence curve or the affinity chart changed, and this is asking
// only whether the buff reached the stat.
func TestARalliedAllyReallyHitsHarder(t *testing.T) {
	fight := aRallyingSquad(t)
	fight.Begin()

	ally, known := fight.Unit("beside")
	if !known {
		t.Fatal("the ally is not in the battle")
	}
	before := fight.Stats(ally)[progression.Attack]

	prompt := theTurnOf(t, fight, "caller")
	aim, found := theAimAt(prompt, "rally", rallyCaller)
	if !found {
		t.Fatal("the caller cannot aim a rally at itself")
	}
	if err := fight.Act("rally", aim); err != nil {
		t.Fatalf("rally: %v", err)
	}

	after := fight.Stats(ally)[progression.Attack]
	if after <= before {
		t.Errorf("an ally rallied from %d attack to %d, so the buff landed and did nothing",
			before, after)
	}
}

// aRallyingSquad is two allies in one column against one enemy.
//
// Written out flat rather than taken from the shipped roster because no shipped
// character learns this skill yet: the skill exists so a team buff is writable,
// and putting it on a learnset is a balance decision with its own measuring to
// do. The stat lines are the bench ones, since nothing here reads a number.
func aRallyingSquad(t *testing.T) *battle.Battle {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	fight, err := battle.New(books, 5, []battle.Roster{
		{ID: "caller", Side: hex.SideAlly, Slot: rallyCaller,
			Affinity: mustAffinity(t, "fire"), Stats: benchStats(3000, 700, 300, 120),
			Skills: []string{"rally", "bite"}},
		{ID: "beside", Side: hex.SideAlly, Slot: rallyBeside,
			Affinity: mustAffinity(t, "water"), Stats: benchStats(3000, 600, 300, 90),
			Skills: []string{"bite"}},
		{ID: "opposed", Side: hex.SideEnemy, Slot: rallyOpposed,
			Affinity: mustAffinity(t, "metal"), Stats: benchStats(4000, 500, 300, 80),
			Skills: []string{"bite"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

// theTurnOf advances until the named unit is the one being asked, so a test
// about one unit's skill does not depend on who the queue happens to open with.
func theTurnOf(t *testing.T, fight *battle.Battle, id string) *battle.Prompt {
	t.Helper()
	for turn := 0; turn < 100; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if prompt.Skipped {
			continue
		}
		if prompt.Unit == id {
			return prompt
		}
		if err := fight.Pass("waiting"); err != nil {
			t.Fatalf("pass: %v", err)
		}
		fight.Drain()
	}
	t.Fatalf("%s was never offered a turn", id)
	return nil
}

// theAimAt finds a skill's offered aim at one cell, and reports whether it is
// offered at all — which for this skill is half of what is being asked.
func theAimAt(prompt *battle.Prompt, id string, cell hex.Offset) (hex.Offset, bool) {
	for _, option := range prompt.Options {
		if option.Skill != id || !option.Available() {
			continue
		}
		for _, aim := range option.Aims {
			if aim == cell {
				return aim, true
			}
		}
	}
	return hex.Offset{}, false
}
