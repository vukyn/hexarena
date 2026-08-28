package forge

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// squadOf builds a squad of one, out of whatever the fixture cast is, at a
// level, and saves it.
//
// One member rather than five because what these tests are about is the
// arrangement — which side a squad stands on, and how the two halves are folded
// — and a squad of five would take five times as long to say the same thing.
func squadOf(t *testing.T, lib *Library, id string, level int) placement.Squad {
	t.Helper()
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Fatal("the fixture cast is empty")
	}
	character := characters[0]
	_, stage, err := character.Resolve(level, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve %s at level %d: %v", character.ID, level, err)
	}
	skills := character.SkillsAt(level, stage.Name)
	if len(skills) > cast.SkillSlots {
		skills = skills[:cast.SkillSlots]
	}
	squad := placement.Squad{
		ID: id,
		Units: []placement.Placement{{
			ID:        "one",
			Character: character.ID,
			Level:     level,
			Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: hex.FormationRows / 2},
			Skills:    skills,
		}},
	}
	if err := lib.SaveSquad(squad); err != nil {
		t.Fatalf("save %q: %v", id, err)
	}
	return squad
}

// TestASquadAgainstACopyOfItselfIsExactlyEven is the control, and it is what
// says the swap is doing its job.
//
// The engine enlists in roster order and breaks a tie in the turn queue by it,
// so the squad listed first has an advantage that has nothing to do with what is
// in it. Fighting one arrangement would report that advantage as though it
// belonged to the squad — a mirror read **58.8%** the last time this repository
// measured one without swapping. Two arrangements over the same seeds put the
// same squad on both ends of it, so a mirror is even to the point rather than
// approximately: every battle it wins as the ally it loses as the enemy.
func TestASquadAgainstACopyOfItselfIsExactlyEven(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	squadOf(t, lib, "twin", progression.LevelCap)
	report, err := lib.FightSquads("twin", "twin", 12)
	if err != nil {
		t.Fatalf("fight: %v", err)
	}
	if !report.Mirror() {
		t.Error("a squad fought against itself does not report as a mirror")
	}
	if rate := report.Rate(); rate != 500 {
		t.Errorf("a mirror reads %d.%d%%, want exactly 50.0%%: the swap is not cancelling "+
			"what it is there to cancel", rate/10, rate%10)
	}
	// Both halves are counted, and over the same seeds.
	if fought := report.Total().Battles(); fought != 24 {
		t.Errorf("12 seeds came to %d battles, want both arrangements of each", fought)
	}
	if report.AsAlly.Battles() != report.AsEnemy.Battles() {
		t.Errorf("the halves are %d and %d battles", report.AsAlly.Battles(), report.AsEnemy.Battles())
	}
}

// TestAStrongerSquadWinsMore is the other half of the same claim: a measurement
// that could not tell two squads apart would pass the mirror test by being
// broken in a symmetric way.
func TestAStrongerSquadWinsMore(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	squadOf(t, lib, "grown", progression.LevelCap)
	squadOf(t, lib, "young", 1)
	report, err := lib.FightSquads("grown", "young", 12)
	if err != nil {
		t.Fatalf("fight: %v", err)
	}
	if report.Rate() <= 500 {
		t.Errorf("a squad at the cap reads %d against one at level 1, want more than half",
			report.Rate())
	}
	// And read from the other end it is the complement, because a rate has a
	// subject and the report says which.
	other, err := lib.FightSquads("young", "grown", 12)
	if err != nil {
		t.Fatalf("fight the other way: %v", err)
	}
	if report.Rate()+other.Rate() != 1000 {
		t.Errorf("the two readings are %d and %d, which do not add to a thousand",
			report.Rate(), other.Rate())
	}
	if report.Turns <= 0 {
		t.Error("no battle finished, so nothing was measured")
	}
}

// TestAFightIsRefusedRatherThanReportedEmpty covers the ways a run cannot start.
func TestAFightIsRefusedRatherThanReportedEmpty(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	squadOf(t, lib, "only", progression.LevelCap)
	for _, testCase := range []struct {
		name       string
		home, away string
		seeds      int
	}{
		{"no seeds at all", "only", "only", 0},
		{"a home squad nobody has built", "missing", "only", 4},
		{"an opponent nobody has built", "only", "missing", 4},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := lib.FightSquads(testCase.home, testCase.away, testCase.seeds); err == nil {
				t.Error("the run started")
			}
		})
	}
}
