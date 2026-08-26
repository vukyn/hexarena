package seed_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestARosterReferenceBringsOnlyTheTraitsItsLevelHasUnlocked is where the gate
// is actually applied: the one place a character and a level meet.
//
// Measured on the shipped roster, which fields the same character at three
// levels — so one battle covers both sides of the gate, and the shipped data is
// the fixture rather than a fixture standing in for it.
func TestARosterReferenceBringsOnlyTheTraitsItsLevelHasUnlocked(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}

	// The gate to measure against, taken from the data rather than written down:
	// a character whose traits are all unlocked at level one has nothing to say
	// here, and would let this pass without measuring anything.
	gate, trait := 0, ""
	for _, character := range characters.All() {
		for _, entry := range character.Passives {
			if entry.AtLevel > gate {
				gate, trait = entry.AtLevel, entry.ID
			}
		}
	}
	if gate <= 1 {
		t.Skip("no shipped character gates a trait above level one, so there is nothing to measure")
	}

	below, above := 0, 0
	for _, unit := range roster {
		holds := false
		for _, id := range unit.Passives {
			if id == trait {
				holds = true
			}
		}
		// The roster entry carries no level of its own once it is resolved, so
		// the two cases are told apart by whether the trait is there at all —
		// and both have to occur, or the roster is not exercising the gate.
		if holds {
			above++
			continue
		}
		below++
	}
	if above == 0 {
		t.Errorf("no unit on the shipped roster holds %q, so the gate is refusing everybody", trait)
	}
	if below == 0 {
		t.Errorf("every unit on the shipped roster holds %q, so the gate at level %d is doing nothing",
			trait, gate)
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

// TestAReferenceMayNotRestateItsTraits keeps the rule the rest of the reference
// form already follows: a character reference is the single source for what it
// resolves, and restating half of it is rejected rather than resolved by
// precedence.
func TestAReferenceMayNotRestateItsTraits(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	named := characters.All()
	if len(named) == 0 {
		t.Skip("the shipped cast is empty")
	}
	mixed := `{"units":[
	  {"id":"a","side":"ally","slot":[2,1],"character":"` + named[0].ID + `","level":` +
		strconv.Itoa(progression.LevelCap) + `,"passives":["endurance"]}
	]}`
	_, err = seed.ParseRoster([]byte(mixed), characters)
	if err == nil {
		t.Fatal("an entry both referencing a character and restating its traits was accepted")
	}
	if !strings.Contains(err.Error(), "passives") {
		t.Errorf("the refusal reads %q, want it to name the field that was restated", err)
	}
}
