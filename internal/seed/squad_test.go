package seed_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/seed"
)

// aSquad is one built out of the shipped cast, because what a placement is
// checked against is a real learnset: a fixture character with an invented kit
// would prove the parser and nothing about the rule under it.
//
// ⚠️ It lives here rather than beside internal/core/placement for the reason
// every core test does: a core package may not read a file, so the book to check
// a squad against is only available where the embed is.
func aSquad(t *testing.T) placement.Squad {
	t.Helper()
	return placement.Squad{
		ID:   "greens",
		Name: "đội cỏ",
		Units: []placement.Placement{{
			ID:        "venusaur",
			Character: "pokemon.bulbasaur",
			Level:     60,
			Slot:      hex.Offset{Col: 0, Row: 1},
			Skills:    []string{"razor_leaf", "sludge_bomb", "venoshock", "leech_seed"},
			Passives:  []string{"venom_blood"},
		}},
	}
}

// TestASquadTakesTheFieldAsEitherSide is what makes a squad worth having: it is
// not written down as somebody's enemy, so one can be measured against several
// opponents and against a copy of itself.
func TestASquadTakesTheFieldAsEitherSide(t *testing.T) {
	characters := castBook(t)
	squad := aSquad(t)
	ally, err := squad.Take(hex.SideAlly, characters)
	if err != nil {
		t.Fatalf("take the ally side: %v", err)
	}
	enemy, err := squad.Take(hex.SideEnemy, characters)
	if err != nil {
		t.Fatalf("take the enemy side: %v", err)
	}
	if ally[0].Side != hex.SideAlly || enemy[0].Side != hex.SideEnemy {
		t.Fatalf("the sides came back as %s and %s", ally[0].Side, enemy[0].Side)
	}
	// The ids are prefixed, which is the whole of how a mirror is readable: two
	// halves of one squad have to be told apart in a log, and nothing else in a
	// squad can do it.
	if ally[0].ID == enemy[0].ID {
		t.Errorf("both halves of a mirror are called %q", ally[0].ID)
	}
	if !strings.HasPrefix(ally[0].ID, "ally.") || !strings.HasPrefix(enemy[0].ID, "enemy.") {
		t.Errorf("the fielded ids are %q and %q", ally[0].ID, enemy[0].ID)
	}
	// And the stat line came from the character rather than from the file: a
	// placement names who it is, and nothing about a squad restates a number.
	if ally[0].Name == "" || ally[0].Stats != enemy[0].Stats {
		t.Errorf("the resolved unit is %+v", ally[0])
	}
}

// TestASquadIsRefusedForWhatABattleWouldRefuse is the division between the two
// checks: shape is refused while the file is read, and a loadout when it is
// asked to fight.
func TestASquadIsRefusedForWhatABattleWouldRefuse(t *testing.T) {
	characters := castBook(t)
	for _, testCase := range []struct {
		name   string
		change func(placement.Squad) placement.Squad
		want   string
	}{
		{"two units on one cell", func(s placement.Squad) placement.Squad {
			twin := s.Units[0].Clone()
			twin.ID = "twin"
			s.Units = append(s.Units, twin)
			return s
		}, "already is"},
		{"more than a side can field", func(s placement.Squad) placement.Squad {
			for i := 0; i < hex.MaxTeamSize; i++ {
				extra := s.Units[0].Clone()
				extra.ID = string(rune('a' + i))
				extra.Slot = hex.Offset{Col: i % hex.FormationCols, Row: i % hex.FormationRows}
				s.Units = append(s.Units, extra)
			}
			return s
		}, "a side can field"},
		{"a level nobody reaches", func(s placement.Squad) placement.Squad {
			s.Units[0].Level = 999
			return s
		}, "outside 1.."},
		{"a cell off the formation", func(s placement.Squad) placement.Squad {
			s.Units[0].Slot = hex.Offset{Col: hex.FormationCols, Row: 0}
			return s
		}, "not a cell"},
		{"nobody in it", func(s placement.Squad) placement.Squad {
			s.Units = nil
			return s
		}, "nothing to field"},
		{"a skill it has not learned", func(s placement.Squad) placement.Squad {
			s.Units[0].Skills = []string{"hydro_pump"}
			return s
		}, "has not learned"},
		{"nothing to swing", func(s placement.Squad) placement.Squad {
			s.Units[0].Skills = nil
			return s
		}, "chooses no skills"},
		{"a character nobody has written", func(s placement.Squad) placement.Squad {
			s.Units[0].Character = "pokemon.nobody"
			return s
		}, "unknown character"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			squad := testCase.change(aSquad(t).Clone())
			_, err := squad.Take(hex.SideAlly, characters)
			if err == nil {
				t.Fatalf("the squad was fielded, want a refusal mentioning %q", testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("refused with %q, want it to mention %q", err, testCase.want)
			}
		})
	}
}

// TestTheShippedCatalogueParses is the file itself: it holds whatever sides the
// author has built, none of them or several, and it has to read back either way.
// That is the whole contract — how many are in it is not one.
//
// ⚠️ This test used to insist the file was empty, and that clause was wrong from
// the day somebody built a squad. seed.Squads says the rule in its own words: the
// game boots from the copy go:embed baked in, so a squad built in the authoring
// tool reaches a battle at the next build, the same rule every other data file
// lives under. A squad shipping is the design working, not an accident to catch —
// and a test that fails when the tool is used is a test that gets deleted rather
// than read. What is worth holding is that the file stays readable, because an
// unreadable one takes the game down with it.
func TestTheShippedCatalogueParses(t *testing.T) {
	if _, err := seed.Squads(); err != nil {
		t.Fatalf("read the shipped catalogue: %v", err)
	}
}

func castBook(t *testing.T) *cast.Book {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	return book
}
