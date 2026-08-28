package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// ranked is a board with the caster wherever a case wants it and the enemy
// holding whichever of its three formation columns the case names.
//
// The caster's own slot is a parameter because the whole point of the rule is
// that it does not matter; the enemy slots are, because they are the only thing
// that does.
func ranked(t *testing.T, casterSlot hex.Offset, skills []string, enemyCols ...int) *battle.Battle {
	t.Helper()
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: casterSlot,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200), Skills: skills},
	}
	for _, col := range enemyCols {
		roster = append(roster, battle.Roster{
			ID: enemyName(col), Side: hex.SideEnemy, Slot: hex.Offset{Col: col, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}})
	}
	fight := mustBattle(t, books(t), 5, roster)
	fight.Begin()
	fight.Drain()
	return fight
}

// enemyName is a stable id per formation column, so a failure names the column
// it is about rather than an index.
func enemyName(col int) string {
	return [...]string{"f.back", "f.mid", "f.front"}[col]
}

// aimsFor is the cells one skill may be pointed at on the opening turn.
func aimsFor(t *testing.T, fight *battle.Battle, skill string) []hex.Offset {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	for _, option := range prompt.Options {
		if option.Skill == skill {
			return option.Aims
		}
	}
	t.Fatalf("the prompt has no option %q", skill)
	return nil
}

// cols is the set of board columns a run of aims covers, which is the unit the
// rank rule is stated in.
func cols(aims []hex.Offset) map[int]bool {
	out := map[int]bool{}
	for _, aim := range aims {
		out[aim.Col] = true
	}
	return out
}

// TestRangeCountsOccupiedRanksFromTheFarSide is the rule itself.
//
// The enemy holds all three of its columns, so the ranks are board 3, 4 and 5
// and a range of N reaches exactly the first N of them.
func TestRangeCountsOccupiedRanksFromTheFarSide(t *testing.T) {
	for _, test := range []struct {
		skill string
		want  []int
	}{
		{"strike", []int{3}},    // range 1
		{"reach", []int{3, 4}},  // range 2
		{"lob", []int{3, 4, 5}}, // range 3
	} {
		fight := ranked(t, hex.Offset{Col: 0, Row: 0}, []string{test.skill}, 0, 1, 2)
		got := cols(aimsFor(t, fight, test.skill))
		if len(got) != len(test.want) {
			t.Errorf("%s reached columns %v, want %v", test.skill, got, test.want)
			continue
		}
		for _, col := range test.want {
			if !got[col] {
				t.Errorf("%s did not reach column %d (reached %v)", test.skill, col, got)
			}
		}
	}
}

// TestAnEmptyRankCostsNoRange is the half that makes a back-line unit playable:
// there is nobody in front to shoot past, so a range of one finds whoever is
// foremost however deep that is.
func TestAnEmptyRankCostsNoRange(t *testing.T) {
	// Only the enemy's own back column is held, which is board column 5.
	fight := ranked(t, hex.Offset{Col: 0, Row: 0}, []string{"strike"}, 0)
	got := cols(aimsFor(t, fight, "strike"))
	if len(got) != 1 || !got[5] {
		t.Errorf("a range-one skill reached %v, want the only occupied rank at column 5", got)
	}
}

// TestARankBlocksTheWholeRankBehindIt is the decision the board's shape rests on:
// blocking is by rank and not by file, so one survivor anywhere in front shields
// everybody behind it.
func TestARankBlocksTheWholeRankBehindIt(t *testing.T) {
	// The enemy front column holds one unit, at row 1; the middle column holds
	// another. A range of one must see only the front.
	fight := ranked(t, hex.Offset{Col: 0, Row: 0}, []string{"strike"}, 2, 1)
	got := cols(aimsFor(t, fight, "strike"))
	if got[4] {
		t.Error("a range-one skill reached past an occupied rank, so blocking is by file rather than by rank")
	}
	if len(got) != 1 || !got[3] {
		t.Errorf("a range-one skill reached %v, want the front rank alone", got)
	}
}

// TestTheCastersOwnColumnDoesNotChangeItsReach is the problem the whole rule was
// changed for: a unit does not move, and most skills declare a range of one, so
// reach measured from the caster's own cell made a back-line placement unusable.
func TestTheCastersOwnColumnDoesNotChangeItsReach(t *testing.T) {
	var first map[int]bool
	for col := range hex.FormationCols {
		fight := ranked(t, hex.Offset{Col: col, Row: 0}, []string{"strike"}, 0, 1, 2)
		got := cols(aimsFor(t, fight, "strike"))
		if len(got) == 0 {
			t.Fatalf("a range-one caster in its own column %d could aim at nothing", col)
		}
		if first == nil {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Errorf("column %d reached %v, want the same %v as the front", col, got, first)
		}
		for reached := range first {
			if !got[reached] {
				t.Errorf("column %d did not reach %d, which the front did", col, reached)
			}
		}
	}
}

// TestAnAllyAimedSkillIgnoresRangeEntirely is the second decision the rule turned
// on.
//
// Reach is read from the *far* side, so it has nothing to say about the half the
// caster is standing in: helping the squad you are in is not a question of
// distance. An ally-aimed skill therefore sees every living ally wherever both of
// them stand, and the range it declares is never consulted.
func TestAnAllyAimedSkillIgnoresRangeEntirely(t *testing.T) {
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"bless"}},
		// As far from the caster as its own half allows.
		{ID: "mate", Side: hex.SideAlly, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	reached := false
	for _, aim := range aimsFor(t, fight, "bless") {
		if aim == (hex.Offset{Col: 0, Row: 0}) {
			reached = true
		}
		if aim.Side() != hex.SideAlly {
			t.Errorf("an ally-aimed skill offered %v, which is not on its own side", aim)
		}
	}
	if !reached {
		t.Error("an ally-aimed skill could not reach the far corner of its own half")
	}
}

// TestATauntReachesThroughAnOccupiedRank is the third decision, and it is settled
// in aims rather than in the rank rule: the taunters are returned before reach is
// consulted at all.
//
// A taunt a front rank could wall off would be a status the front rank cancels,
// and taking the choice of enemy is the whole of what a taunt does.
func TestATauntReachesThroughAnOccupiedRank(t *testing.T) {
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"strike"}},
		// The wall, in the enemy's own front column.
		{ID: "f.front", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}},
		// The taunter, two ranks behind it.
		{ID: "f.back", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"jab"}},
	}
	fight := mustBattle(t, books(t), 5, roster)
	fight.Begin()
	fight.Drain()
	taunter := unitByID(t, fight, "f.back")

	// Without the taunt, a range-one skill sees the front rank alone.
	before := cols(aimsFor(t, fight, "strike"))
	if before[taunter.Cell.Col] {
		t.Fatalf("the back rank was reachable before the taunt, so this measures nothing (%v)", before)
	}

	// A second battle rather than a second Advance on this one: aimsFor advances,
	// and advancing again without acting is refused.
	taunted := mustBattle(t, books(t), 5, roster)
	taunted.Begin()
	taunted.Drain()
	carrying(t, taunted, "f.back", "taunting", 1)
	after := aimsFor(t, taunted, "strike")
	if len(after) != 1 || after[0] != taunter.Cell {
		t.Errorf("a taunt from behind a held rank produced %v, want the taunter alone at %v",
			after, taunter.Cell)
	}
}
