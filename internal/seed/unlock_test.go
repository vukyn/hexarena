package seed_test

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestARosterReferenceBringsOnlyTheTraitsItsLevelHasUnlocked is where the gate
// is actually applied: the one place a character and a level meet.
//
// Measured on the shipped roster, which fields the same character at three
// levels — so one battle covers both sides of the gate, and the shipped data is
// the fixture rather than a fixture standing in for it.
// TestARosterReferenceBringsOnlyWhatItsLevelHasUnlocked is the gate, read the way
// a slot made it readable.
//
// It used to check that a unit whose level had unlocked a trait was carrying it,
// which was true while a placement brought every trait it had. A placement now
// **chooses** one, so an unlocked trait may quite deliberately be left behind —
// and the property that survives is the one that was always the point: nothing
// is brought that the level has not unlocked.
func TestARosterReferenceBringsOnlyWhatItsLevelHasUnlocked(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	raw, err := seed.RosterFile()
	if err != nil {
		t.Fatalf("read the roster: %v", err)
	}
	var file struct {
		Units []struct {
			ID        string `json:"id"`
			Character string `json:"character"`
			Level     int    `json:"level"`
		} `json:"units"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode the roster: %v", err)
	}
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}
	if len(roster) != len(file.Units) {
		t.Fatalf("the roster resolved to %d units from %d entries", len(roster), len(file.Units))
	}

	// The gate is only doing something if somebody on this roster is standing
	// below one, so that is measured rather than assumed.
	gated := 0
	for i, unit := range file.Units {
		if unit.Character == "" {
			continue
		}
		character, known := characters.Get(unit.Character)
		if !known {
			t.Errorf("%s references the unknown character %q", unit.ID, unit.Character)
			continue
		}
		available := character.PassivesAt(unit.Level, progression.Furthest)
		if len(available) < len(character.Passives) {
			gated++
		}
		for _, held := range roster[i].Passives {
			if !slices.Contains(available, held) {
				t.Errorf("%s brings %q, which %s has not unlocked at level %d",
					unit.ID, held, character.ID, unit.Level)
			}
		}
		// And the same rule for the kit, which is the half that is new.
		learned := character.SkillsAt(unit.Level, progression.Furthest)
		for _, carried := range roster[i].Skills {
			if !slices.Contains(learned, carried) {
				t.Errorf("%s brings %q, which %s has not learned at level %d",
					unit.ID, carried, character.ID, unit.Level)
			}
		}
	}
	if gated == 0 {
		t.Error("every unit on the shipped roster has unlocked everything, so no gate is being exercised")
	}
}

// TestAFlatRosterEntryStatesItsOwnTraits is the other form, and the reason the
// gate is not applied to it: a flat entry writes out its numbers instead of
// naming a character, so it has no level for a gate to be measured against.
func TestAFlatRosterEntryStatesItsOwnTraits(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	flat := `{"units":[
	  {"id":"a","name":"A","side":"ally","slot":[2,1],"element":["grass"],
	   "stats":{"hp":900,"attack":150,"defense":130,"speed":30,"accuracy":40,"dodge":15},
	   "skills":["vine_whip"],"passives":["endurance"]},
	  {"id":"f","name":"F","side":"enemy","slot":[2,1],"element":["grass"],
	   "stats":{"hp":900,"attack":150,"defense":130,"speed":30,"accuracy":40,"dodge":15},
	   "skills":["vine_whip"]}
	]}`
	roster, err := seed.ParseRoster([]byte(flat), characters)
	if err != nil {
		t.Fatalf("parse a flat roster: %v", err)
	}
	if len(roster) != 2 {
		t.Fatalf("the roster came back with %d units", len(roster))
	}
	if len(roster[0].Passives) != 1 || roster[0].Passives[0] != "endurance" {
		t.Errorf("the flat entry brought %v, want the trait it wrote out", roster[0].Passives)
	}
	if len(roster[1].Passives) != 0 {
		t.Errorf("an entry naming no trait brought %v", roster[1].Passives)
	}
}

// TestAReferenceChoosesItsTraitRatherThanRestatingIt is the rule that replaced
// the one this used to hold.
//
// `passives` on a reference used to be a restatement of the character sheet and
// was refused as one. It is now the **trait slot**: a placement picks one of the
// traits the character has unlocked at its level, which is a decision rather
// than a second copy of a fact. What is still refused is naming a trait the
// character does not have there, or naming more than the slot holds.
func TestAReferenceChoosesItsTraitRatherThanRestatingIt(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	// A character with at least two traits at the cap, because a slot that has
	// nothing to choose between measures nothing.
	var chooser cast.Character
	for _, character := range characters.All() {
		if len(character.PassivesAt(progression.LevelCap, progression.Furthest)) >= 2 {
			chooser = character
			break
		}
	}
	if chooser.ID == "" {
		t.Skip("no shipped character holds two traits, so a trait slot has nothing to choose between")
	}
	held := chooser.PassivesAt(progression.LevelCap, progression.Furthest)
	place := func(passives string) error {
		raw := `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + chooser.ID +
			`","level":` + strconv.Itoa(progression.LevelCap) +
			`,"skills":` + loadoutOf(t, chooser, progression.LevelCap) +
			`,"passives":` + passives + `}]}`
		_, err := seed.ParseRoster([]byte(raw), characters)
		return err
	}

	if err := place(`["` + held[0] + `"]`); err != nil {
		t.Errorf("choosing one of its own traits was refused: %v", err)
	}
	if err := place(`["` + held[0] + `","` + held[1] + `"]`); err == nil {
		t.Error("a placement brought two traits and the slot holds one")
	} else if !strings.Contains(err.Error(), "slot") {
		t.Errorf("the refusal reads %q, want it to say the slot is full", err)
	}
	if err := place(`["nonesuch"]`); err == nil {
		t.Error("a placement brought a trait the character does not have")
	} else if !strings.Contains(err.Error(), "not learned") {
		t.Errorf("the refusal reads %q, want it to say the character has not learned it", err)
	}
}

// TestAPlacementMustChooseItsLoadout is the half of a slot that is easy to leave
// out: not what it refuses, but that it insists on being asked.
//
// A default would be this parser quietly picking four of nine on an author's
// behalf and never saying which — and the whole point of a slot is that somebody
// decided. The refusal lists what was available, because an author who has just
// been told "no" wants the list rather than a second trip to cast.json.
func TestAPlacementMustChooseItsLoadout(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	named := characters.All()
	if len(named) == 0 {
		t.Skip("the shipped cast is empty")
	}
	character := named[0]
	raw := `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + character.ID +
		`","level":` + strconv.Itoa(progression.LevelCap) + `}]}`
	_, err = seed.ParseRoster([]byte(raw), characters)
	if err == nil {
		t.Fatal("a placement that chose no skills was accepted")
	}
	if !strings.Contains(err.Error(), "chooses no skill") {
		t.Errorf("the refusal reads %q, want it to say nothing was chosen", err)
	}
	for _, known := range character.SkillsAt(progression.LevelCap, progression.Furthest) {
		if !strings.Contains(err.Error(), known) {
			t.Errorf("the refusal does not offer %q, which the character knows: %v", known, err)
		}
	}
}

// TestAPlacementCannotBringMoreThanItsSlots is the cap itself, in both lists.
func TestAPlacementCannotBringMoreThanItsSlots(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	var deep cast.Character
	for _, character := range characters.All() {
		if len(character.SkillsAt(progression.LevelCap, progression.Furthest)) > cast.SkillSlots {
			deep = character
			break
		}
	}
	if deep.ID == "" {
		t.Skip("no shipped character learns more than a placement may bring")
	}
	known := deep.SkillsAt(progression.LevelCap, progression.Furthest)
	tooMany, err := json.Marshal(known[:cast.SkillSlots+1])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + deep.ID +
		`","level":` + strconv.Itoa(progression.LevelCap) + `,"skills":` + string(tooMany) + `}]}`
	if _, err := seed.ParseRoster([]byte(raw), characters); err == nil {
		t.Errorf("a placement brought %d skills and there are %d slots",
			cast.SkillSlots+1, cast.SkillSlots)
	} else if !strings.Contains(err.Error(), "slot") {
		t.Errorf("the refusal reads %q, want it to say the slots are full", err)
	}

	// The same skill twice is a placement that has wasted a slot rather than
	// filled two, and it reads as a typo.
	doubled := `["` + known[0] + `","` + known[0] + `"]`
	raw = `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + deep.ID +
		`","level":` + strconv.Itoa(progression.LevelCap) + `,"skills":` + doubled + `}]}`
	if _, err := seed.ParseRoster([]byte(raw), characters); err == nil {
		t.Error("a placement brought the same skill twice")
	} else if !strings.Contains(err.Error(), "twice") {
		t.Errorf("the refusal reads %q, want it to say the skill was named twice", err)
	}
}

// TestAPlacementCannotBringWhatItsLevelHasNotLearned is the learnset doing its
// job: a young unit is not a grown one with fewer slots, it is one that has
// fewer things to put in them.
func TestAPlacementCannotBringWhatItsLevelHasNotLearned(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	// A character and a level where something is still ahead of it.
	var late cast.Character
	var gate int
	for _, character := range characters.All() {
		for _, entry := range character.Skills {
			if entry.AtLevel > gate {
				late, gate = character, entry.AtLevel
			}
		}
	}
	if gate <= 1 {
		t.Skip("no shipped character gates a skill above level one")
	}
	ahead := ""
	for _, entry := range late.Skills {
		if entry.AtLevel == gate {
			ahead = entry.ID
		}
	}
	below := gate - 1
	raw := `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + late.ID +
		`","level":` + strconv.Itoa(below) + `,"skills":["` + ahead + `"]}]}`
	_, err = seed.ParseRoster([]byte(raw), characters)
	if err == nil {
		t.Fatalf("a level %d placement brought %q, which is learned at %d", below, ahead, gate)
	}
	if !strings.Contains(err.Error(), "not learned") {
		t.Errorf("the refusal reads %q, want it to say the level has not learned it", err)
	}
	// And the same placement at the level it learns it is fine, which is what
	// makes the refusal about the level rather than about the skill.
	raw = `{"units":[{"id":"a","side":"ally","slot":[2,1],"character":"` + late.ID +
		`","level":` + strconv.Itoa(gate) + `,"skills":["` + ahead + `"]}]}`
	if _, err := seed.ParseRoster([]byte(raw), characters); err != nil {
		t.Errorf("the same placement at level %d was refused: %v", gate, err)
	}
}
