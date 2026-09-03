package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// TestAForkingLineIsShownAsARowPerArm is what `hexforge show` prints for the one
// shipped character whose line forks.
//
// A line that forks has no single furthest form, and Resolve refuses to pick one
// on purpose — taking whichever the file lists last would hand a reader the wrong
// form's stat line with nothing saying so. This print used to hand that refusal
// straight through as the row's own value and carry on, so `hexforge show
// pokemon.poliwag` ended on a sentence where the stats belong, and neither arm's
// numbers were reachable from the command at all.
//
// The screens answered the same refusal with a chooser. A one-shot print has
// nowhere to hold a choice, so it prints both arms — which is why this asserts on
// the *pair*: a page naming one arm and not the other would be the old defect
// wearing a stage name.
func TestAForkingLineIsShownAsARowPerArm(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	character, known := lib.Characters().Get(forkedCharacter)
	if !known {
		t.Fatalf("%s is not in the shipped cast, so this measures nothing", forkedCharacter)
	}
	arms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("furthest at the cap: %v", err)
	}
	if len(arms) < 2 {
		t.Fatalf("%s reaches %d form(s) at the cap, so it does not fork and this test is measuring an ordinary line",
			forkedCharacter, len(arms))
	}

	var page bytes.Buffer
	renderCharacter(&page, lib, character, progression.LevelCap)
	drawn := page.String()

	if strings.Contains(drawn, "which are alternatives") {
		t.Errorf("the page hands the reader the refusal where the stats belong:\n%s", drawn)
	}
	for _, arm := range arms {
		values, _, err := character.Resolve(progression.LevelCap, arm.Name)
		if err != nil {
			t.Fatalf("resolve %s: %v", arm.Name, err)
		}
		// The name and the numbers, because a page could name both arms off the
		// stages row alone — that row lists them and is printed either way.
		if !strings.Contains(drawn, values.String()) {
			t.Errorf("the page does not carry %s's stat line %q:\n%s", arm.Name, values, drawn)
		}
		if !strings.Contains(drawn, "stage \""+arm.Name+"\" shows") {
			t.Errorf("the page draws no stage row for %s:\n%s", arm.Name, drawn)
		}
	}
}

// TestAnOrdinaryLineIsStillOneRow is the other half: the change above must cost
// a line that does not fork nothing at all, because FurthestAt answers exactly
// one stage there and a second row would be a form nobody has.
func TestAnOrdinaryLineIsStillOneRow(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	for _, character := range lib.Characters().All() {
		arms, err := character.FurthestAt(progression.LevelCap)
		if err != nil || len(arms) != 1 {
			continue
		}
		var page bytes.Buffer
		renderCharacter(&page, lib, character, progression.LevelCap)
		if rows := strings.Count(page.String(), "stage \""); rows != 1 {
			t.Errorf("%s does not fork at the cap and draws %d stage rows, want 1:\n%s",
				character.ID, rows, page.String())
		}
	}
}

// forkedCharacter is named rather than searched for because the assertion above
// is about a shipped line, and a test that quietly found no fork would pass on
// nothing. If this character is ever unforked, the fatal above says so.
const forkedCharacter = "pokemon.poliwag"
