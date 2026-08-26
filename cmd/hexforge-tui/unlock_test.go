package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheTraitRowFollowsTheLevelAndOnlyMarksAGateStillAhead is the feature as a
// reader meets it.
//
// A gate is printed while it is still in the future and not once it has been
// passed, so the mark reads as "not yet" rather than as a fact about the trait —
// and the row therefore *changes* as the level is walked, which is the one thing
// a level slider exists to show. Printing every gate at every level would say
// the same thing at level one and at the cap.
func TestTheTraitRowFollowsTheLevelAndOnlyMarksAGateStillAhead(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	m := newModel(lib, i18n.Vi)
	m.width, m.height = 100, 44
	m = m.enter(screenBrowse)

	// The character and the gate, taken from the data: one whose traits all come
	// in at level one has nothing to show here.
	var subject cast.Character
	gate, trait := 0, ""
	for _, candidate := range lib.Characters().All() {
		for _, entry := range candidate.Passives {
			if entry.AtLevel > gate {
				subject, gate, trait = candidate, entry.AtLevel, entry.ID
			}
		}
	}
	if gate <= 1 {
		t.Skip("no shipped character gates a trait above level one, so there is nothing to show")
	}

	m.browse.level = gate - 1
	locked := m.browse.detail(m, subject)
	if !strings.Contains(locked, trait+"@"+strconv.Itoa(gate)) {
		t.Errorf("one level below the gate the pane does not mark it:\n%s", locked)
	}

	m.browse.level = gate
	unlocked := m.browse.detail(m, subject)
	if strings.Contains(unlocked, trait+"@") {
		t.Errorf("at the gate the pane still marks it as ahead:\n%s", unlocked)
	}
	if !strings.Contains(unlocked, trait) {
		t.Errorf("at the gate the pane does not name the trait at all:\n%s", unlocked)
	}
	if locked == unlocked {
		t.Error("the pane drew the same thing either side of the gate")
	}

	// The names under the row are the traits actually in force. Glossing one the
	// unit does not have yet would read as one it has, which is the opposite of
	// what the mark above it says.
	held, err := lib.Passives().Lookup(trait)
	if err != nil {
		t.Fatalf("look up %s: %v", trait, err)
	}
	if held.Name == "" {
		t.Skip("the shipped trait has no authored name, so there is no gloss to place")
	}
	if strings.Contains(locked, held.Name) {
		t.Errorf("a trait the unit has not unlocked was glossed by name:\n%s", locked)
	}
	if !strings.Contains(unlocked, held.Name) {
		t.Errorf("a trait in force was not glossed by name:\n%s", unlocked)
	}
}
