package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheBonusListingSaysWhoReceivesEachOne is the claim the screen exists for.
//
// Every other reference in this package describes something a player can also
// see on the board: a status is on a roster line, a trait is on a character, a
// skill is on a unit's own list. A composition bonus is not — it is decided from
// the squad before the first turn and leaves only a permanent buff behind, which
// is exactly what a trait leaves. So the two facts that cannot be recovered by
// looking are the ones this has to draw: **which threshold** paid, and **who**
// receives it.
func TestTheBonusListingSaysWhoReceivesEachOne(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	body, footer := NewBonusesScreen(lib).View(c)
	declared := lib.Bonuses().All()
	if len(declared) == 0 {
		t.Fatal("the shipped book declares no bonus, so this screen is measured against nothing")
	}
	for _, bonus := range declared {
		if !strings.Contains(body, bonus.ID) {
			t.Errorf("the listing does not name the bonus %q", bonus.ID)
		}
	}
	// The scope in the data's own word, on the row, so two bonuses granting one
	// status can be told apart at a glance.
	if !strings.Contains(body, declared[0].Scope.String()) {
		t.Errorf("the row does not say who %q goes to:\n%s", declared[0].ID, body)
	}
	// And every rung of the one under the cursor, with what it grants.
	for _, rung := range declared[0].Rungs {
		for _, grant := range rung.Grants {
			if !strings.Contains(body, grant.Status) {
				t.Errorf("the description does not name what the rung at %d grants:\n%s", rung.At, body)
			}
		}
	}
	if footer == "" {
		t.Error("the screen draws no footer")
	}
}

// TestTheBonusCaveatIsDrawnOnceRatherThanPerRow holds the placement decision:
// the thing that is true of every bonus — counted once, never recounted, several
// live at a time — sits at the foot, because a warning repeated under every row
// is a warning nobody finishes reading.
func TestTheBonusCaveatIsDrawnOnceRatherThanPerRow(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	body, _ := NewBonusesScreen(lib).View(c)
	caveat := c.Text(i18n.BlurbBonusCaveat)
	if got := strings.Count(body, caveat); got != 1 {
		t.Errorf("the caveat is drawn %d times, wanted once:\n%s", got, body)
	}
}

// TestTheBonusRoomLeavesTheWidestDescriptionItsRows is the layout rule, and it
// is asserted against a **book that grows** rather than against the shipped one.
//
// The rungs are the part expected to change — two today, four when 5v5 opens —
// so a room computed from anything but the rungs in hand is a number that stops
// being right the day one is added, and the listing would then be drawn over the
// description under it.
func TestTheBonusRoomLeavesTheWidestDescriptionItsRows(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	small := []composition.Bonus{{ID: "one", Rungs: []composition.Rung{
		{At: 2, Grants: []composition.Grant{{Status: "kinship", Stacks: 1}}},
	}}}
	big := []composition.Bonus{{ID: "one", Rungs: []composition.Rung{
		{At: 2, Grants: []composition.Grant{{Status: "kinship", Stacks: 1}}},
		{At: 3, Grants: []composition.Grant{{Status: "kinship", Stacks: 2}}},
		{At: 4, Grants: []composition.Grant{{Status: "kinship", Stacks: 3}}},
		{At: 5, Grants: []composition.Grant{{Status: "kinship", Stacks: 4}}},
	}}}
	narrow, wide := bonusesRoom(c, small), bonusesRoom(c, big)
	if wide >= narrow {
		t.Errorf("a four-rung book left the listing %d rows and a one-rung book %d: "+
			"the room does not follow the rungs, so a longer description would be drawn over",
			wide, narrow)
	}
	if wide != narrow-3 {
		t.Errorf("three more rungs cost %d rows, wanted 3", narrow-wide)
	}
}

// TestTheBonusCursorStaysInsideTheBook is the ordinary guard, and it is here
// because this listing has no headings to step over: the cursor is clamped at
// both ends rather than settled onto the nearest row, so the two ends are the
// whole of what can go wrong.
func TestTheBonusCursorStaysInsideTheBook(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	s := NewBonusesScreen(lib)
	for range len(s.Rows) + 3 {
		s, _ = s.Update(c, press(t, "down"))
	}
	if s.Cursor != len(s.Rows)-1 {
		t.Errorf("walking off the end left the cursor at %d of %d rows", s.Cursor, len(s.Rows))
	}
	for range len(s.Rows) + 3 {
		s, _ = s.Update(c, press(t, "up"))
	}
	if s.Cursor != 0 {
		t.Errorf("walking off the top left the cursor at %d", s.Cursor)
	}
	if _, action := s.Update(c, press(t, "esc")); action.Kind != Back {
		t.Errorf("esc gave %v, wanted Back", action.Kind)
	}
	if _, action := s.Update(c, press(t, "q")); action.Kind != Quit {
		t.Errorf("q gave %v, wanted Quit", action.Kind)
	}
}

// TestAnEmptyBonusBookSaysSo is the state no shipped data can reach, and it is
// drawn rather than left as a blank screen for the reason every other empty
// listing here says so: a reference that renders nothing reads as broken.
func TestAnEmptyBonusBookSaysSo(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	body, footer := BonusesScreen{}.View(c)
	if !strings.Contains(body, c.Text(i18n.BonusesEmpty)) {
		t.Errorf("an empty book drew no line saying so:\n%s", body)
	}
	if footer == "" {
		t.Error("an empty book drew no footer")
	}
}
