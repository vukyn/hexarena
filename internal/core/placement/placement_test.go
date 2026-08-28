package placement_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
)

// aSquad is a squad in shape only. Nothing here resolves one against a cast
// book, because a core package may not read a file and the book lives where the
// embed does — internal/seed's squad_test is where a squad meets a real
// learnset.
func aSquad() placement.Squad {
	return placement.Squad{
		ID:   "greens",
		Name: "đội cỏ",
		Units: []placement.Placement{{
			ID:        "venusaur",
			Character: "pokemon.bulbasaur",
			Level:     60,
			Slot:      hex.Offset{Col: 0, Row: 1},
			Skills:    []string{"razor_leaf", "sludge_bomb"},
			Passives:  []string{"venom_blood"},
		}},
	}
}

// TestASquadIsRefusedForItsShape is the half of the checking that needs no cast
// book: how many are in it, where they stand, and whether they are somebody.
//
// It is a separate question from whether the loadout is legal, and deliberately
// so: a squad being built has members in every state of half-finished, and a
// file that refused to hold one could not be saved and come back to.
func TestASquadIsRefusedForItsShape(t *testing.T) {
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
		{"a cell off the board", func(s placement.Squad) placement.Squad {
			s.Units[0].Slot = hex.Offset{Col: 0, Row: hex.FormationRows}
			return s
		}, "not a cell"},
		{"nobody in it", func(s placement.Squad) placement.Squad {
			s.Units = nil
			return s
		}, "nothing to field"},
		{"two units of one name", func(s placement.Squad) placement.Squad {
			twin := s.Units[0].Clone()
			twin.Slot = hex.Offset{Col: 1, Row: 1}
			s.Units = append(s.Units, twin)
			return s
		}, "two units are called"},
		{"a member who is nobody", func(s placement.Squad) placement.Squad {
			s.Units[0].Character = ""
			return s
		}, "names no character"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.change(aSquad().Clone()).Validate()
			if err == nil {
				t.Fatalf("the squad was accepted, want a refusal mentioning %q", testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("refused with %q, want it to mention %q", err, testCase.want)
			}
		})
	}
}

// TestASquadHasNoSideOfItsOwn is the refusal that keeps a squad reusable: it is
// fielded as a side rather than written down as one.
func TestASquadHasNoSideOfItsOwn(t *testing.T) {
	if _, err := aSquad().Take(hex.SideNone, nil); err == nil {
		t.Error("a squad took the field on no side at all")
	}
}

// TestASquadSurvivesBeingWrittenBackOut is the round trip, and it matters for
// the reason every round trip here does: the builder rewrites the whole file on
// every save, so a field dropped on the way through is a squad that quietly
// becomes a different squad.
func TestASquadSurvivesBeingWrittenBackOut(t *testing.T) {
	raw, err := placement.Marshal([]placement.Squad{aSquad()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := placement.Parse(raw)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(back) != 1 {
		t.Fatalf("%d squads came back", len(back))
	}
	want, got := aSquad(), back[0]
	if got.ID != want.ID || got.Name != want.Name || len(got.Units) != len(want.Units) {
		t.Fatalf("the squad came back as %+v", got)
	}
	unit, wanted := got.Units[0], want.Units[0]
	if unit.ID != wanted.ID || unit.Character != wanted.Character ||
		unit.Level != wanted.Level || unit.Slot != wanted.Slot ||
		strings.Join(unit.Skills, " ") != strings.Join(wanted.Skills, " ") ||
		strings.Join(unit.Passives, " ") != strings.Join(wanted.Passives, " ") {
		t.Errorf("the member came back as %+v, want %+v", unit, wanted)
	}
	// Two squads of one id is refused at the read, because naming one of them
	// afterwards chooses neither.
	twinned, err := placement.Marshal([]placement.Squad{aSquad(), aSquad()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := placement.Parse(twinned); err == nil {
		t.Error("two squads of one id were read")
	}
	// An empty catalogue is a legal file rather than an absent one: the shipped
	// squads.json holds none, and it still has to read back.
	empty, err := placement.Marshal(nil)
	if err != nil {
		t.Fatalf("marshal nothing: %v", err)
	}
	if squads, err := placement.Parse(empty); err != nil || len(squads) != 0 {
		t.Errorf("an empty catalogue read back as %v, %v", squads, err)
	}
}

// TestAClonedSquadSharesNothing is what lets a builder hold one while the
// library holds another: a slice shared between them would have every keystroke
// reaching the file's idea of what is saved.
func TestAClonedSquadSharesNothing(t *testing.T) {
	original := aSquad()
	copied := original.Clone()
	copied.Units[0].Skills[0] = "changed"
	copied.Units[0].ID = "renamed"
	if original.Units[0].Skills[0] == "changed" || original.Units[0].ID == "renamed" {
		t.Errorf("editing the copy reached the original: %+v", original.Units[0])
	}
}
