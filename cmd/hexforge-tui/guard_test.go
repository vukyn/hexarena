package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// The behaviour half of the guard, which is the half a totality walk cannot see.
//
// TestEveryScreenThatAsksAnswersItsOwnQuestion in navigate_test.go proves every
// asking screen has a dispatch entry; an entry that exists and does nothing
// passes it, which is exactly what #207 measured on an applier table. So each of
// the five confirms is driven here with the keys a reader would press and
// checked for what it actually did — four of them, and the fifth is
// TestLeavingAnEditedFormAsksFirst in tui_test.go, which has held the character
// form's discard (and its walk back to the menu) since the guard was written.
//
// Every one of these presses `y` through the real key path rather than calling
// answerGuard, because the dispatch is the thing being measured and a direct
// call would go round it.

// TestDiscardingAHalfTypedWorkEmptiesTheFormAndStays is the origins form's
// confirm, which is the plainest of the five: it resets its own screen and asks
// the client for nothing.
func TestDiscardingAHalfTypedWorkEmptiesTheFormAndStays(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = menuTo(t, m, screenOrigins)
	m = typeText(t, m, "a")
	if !m.origins.Adding {
		t.Fatal("a did not open the add-a-work form")
	}
	m = typeText(t, m, "fixture.work")
	if !m.origins.Touched {
		t.Fatal("typing an id left the form reading as untouched, so escape asks nothing")
	}

	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving a typed-into work form did not ask")
	}
	if m.guard.question != i18n.OriginFormDiscard {
		t.Errorf("the question asked was %v", m.guard.question)
	}

	m = typeText(t, m, "y")
	if m.guard != nil {
		t.Error("the question is still pending after a yes")
	}
	if m.origins.Adding {
		t.Error("confirming the discard left the form open")
	}
	if m.origins.Touched {
		t.Error("the form that came back still claims changes")
	}
	if got := m.origins.Inputs[draw.OriginFieldID].Value(); got != "" {
		t.Errorf("the id field still reads %q, want the typed work thrown away", got)
	}
	// It stays on the works listing: this confirm hands back the zero action, so
	// the client is asked to do nothing.
	if m.screen != screenOrigins {
		t.Errorf("discarding the work form left for screen %v", m.screen)
	}
}

// TestDiscardingAHalfWrittenSkillEmptiesTheFormAndStays is the skill form's
// confirm.
//
// It is asked over an **edit** rather than over a fresh skill, because that is
// the arm carrying the second wording (SkillFormEditDiscard) and the one where
// the form has something in it that came off the book — so a confirm that reset
// nothing would leave the prefilled fields standing and look like it had worked.
func TestDiscardingAHalfWrittenSkillEmptiesTheFormAndStays(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = menuTo(t, m, screenSkills)
	m = typeText(t, m, "e")
	if !m.skills.FormInFront() {
		t.Fatal("e did not open the skill form over the row under the cursor")
	}
	if m.skills.Editing == "" {
		t.Fatal("the form opened over no skill, so the edit wording is not the one asked")
	}
	m = typeText(t, m, "x")
	if !m.skills.Touched {
		t.Fatal("typing left the skill form reading as untouched, so escape asks nothing")
	}

	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving a typed-into skill form did not ask")
	}
	if m.guard.question != i18n.SkillFormEditDiscard {
		t.Errorf("the question asked was %v, want the one about losing an edit", m.guard.question)
	}

	m = typeText(t, m, "y")
	if m.guard != nil {
		t.Error("the question is still pending after a yes")
	}
	if m.skills.FormInFront() {
		t.Errorf("confirming the discard left the form open over %q", m.skills.Editing)
	}
	if m.skills.Touched {
		t.Error("the form that came back still claims changes")
	}
	if got := m.skills.Inputs[draw.SkillFieldID].Value(); got != "" {
		t.Errorf("the id field still reads %q, want the prefilled edit thrown away", got)
	}
	if m.screen != screenSkills {
		t.Errorf("discarding the skill form left for screen %v", m.screen)
	}
}

// TestDiscardingASquadUnderEditPutsTheFileBackAndReturnsToTheCatalogue is the
// first of the squad builder's two confirms, and the one told apart by
// guardsTheSquadInHand.
//
// What it has to get right is that discarding **restores** rather than only
// leaving: the squad in hand goes back to the one on the file, so re-opening it
// shows the file's version rather than the edit that was supposedly thrown away.
func TestDiscardingASquadUnderEditPutsTheFileBackAndReturnsToTheCatalogue(t *testing.T) {
	m := aSavedSquad(t)
	saved := m.squad.Baseline.Clone()
	m = typeText(t, m, "x")
	if !m.squad.Dirty() {
		t.Fatal("typing into the name left the squad reading as the one on the file")
	}

	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving an edited squad did not ask")
	}
	if m.guard.question != i18n.SquadDiscard {
		t.Errorf("the question asked was %v", m.guard.question)
	}

	m = typeText(t, m, "y")
	if m.guard != nil {
		t.Error("the question is still pending after a yes")
	}
	if m.squad.Mode != draw.SquadList {
		t.Errorf("confirming the discard left the builder in %v", m.squad.Mode)
	}
	if !m.squad.Editing.Equal(saved) {
		t.Errorf("the squad in hand is %+v, want the one on the file back", m.squad.Editing)
	}
	if m.squad.Dirty() {
		t.Error("the squad in hand still differs from the one written down")
	}
	// And the file itself was not touched: this question discards an edit, it
	// does not delete anything.
	if got := len(m.lib.Squads()); got != 1 {
		t.Errorf("the catalogue holds %d squads after discarding an edit, want the one saved", got)
	}
}

// TestDeletingASquadTakesTheOneUnderTheCursorAndNoOther is the second of the
// builder's confirms, and the only one of the five that spends the id the
// question carried.
//
// ⚠️ **Two squads, cursor on the first**, which is what makes this an assertion
// rather than a count. With one squad on the file any id at all deletes it, so a
// confirm reading the wrong row — the one after the cursor, the last one, the
// first one regardless — passes a one-squad fixture completely. The catalogue is
// checked from the library rather than from the screen, because the screen is
// refreshed from the library and would agree with it either way.
func TestDeletingASquadTakesTheOneUnderTheCursorAndNoOther(t *testing.T) {
	m := aCatalogueOfTwoSquads(t)
	wanted, spared := m.squad.Saved[0].ID, m.squad.Saved[1].ID
	if wanted == spared {
		t.Fatalf("both fixture squads are called %q, so nothing here can tell them apart", wanted)
	}
	m.squad.Cursor = 0

	m = typeText(t, m, "d")
	if m.guard == nil {
		t.Fatal("d on the catalogue deleted without asking")
	}
	if m.guard.question != i18n.SquadDiscardSaved {
		t.Errorf("the question asked was %v", m.guard.question)
	}
	// The id travels with the question rather than being read back off the
	// cursor when the answer arrives, and this is what says so.
	//
	// ⚠️ **The pending subject is an `any` now**, because the screen that asks is
	// in internal/screen and the vocabulary telling its two questions apart went
	// with it. This client carries the value and never opens it, so the assertion
	// has to say which type it expected — a guard filed under some other screen's
	// vocabulary would fail the assertion here rather than silently comparing
	// unequal.
	about, carried := m.guard.about.(draw.SquadsAsk)
	if !carried {
		t.Fatalf("the pending question is about a %T, want a draw.SquadsAsk", m.guard.about)
	}
	if want := (draw.SquadsAsk{Kind: draw.SquadsAskSavedSquad, ID: wanted}); about != want {
		t.Errorf("the pending question is about %v, want %v", about, want)
	}

	m = typeText(t, m, "y")
	if m.guard != nil {
		t.Error("the question is still pending after a yes")
	}
	if m.squad.Err != nil {
		t.Fatalf("the delete was refused: %v", m.squad.Err)
	}
	held := m.lib.Squads()
	if len(held) != 1 {
		t.Fatalf("the catalogue holds %d squads after one delete", len(held))
	}
	if held[0].ID != spared {
		t.Errorf("the squad left on the file is %q, want %q — the wrong one was deleted",
			held[0].ID, spared)
	}
	// And the screen was refreshed rather than left showing a squad that is gone.
	if got := len(m.squad.Saved); got != 1 {
		t.Errorf("the catalogue on screen still lists %d squads", got)
	}
	if m.screen != screenSquads {
		t.Errorf("deleting a squad left for screen %v", m.screen)
	}
}

// aCatalogueOfTwoSquads is the fixture the delete needs: two squads on the file
// and the builder sitting on the catalogue that lists them.
//
// The second is the first under another id, because what is being told apart is
// which row was taken and not what either row holds.
func aCatalogueOfTwoSquads(t *testing.T) model {
	t.Helper()
	m := aSavedSquad(t)
	second := m.squad.Editing.Clone()
	second.ID, second.Name = "do-hai", "đội hai"
	if err := m.lib.SaveSquad(second); err != nil {
		t.Fatalf("save the second fixture squad: %v", err)
	}
	m = key(t, m, "esc")
	if m.guard != nil {
		t.Fatalf("backing out of an unedited squad raised %v", m.guard.question)
	}
	if m.squad.Mode != draw.SquadList {
		t.Fatalf("escape from the fixture squad landed in %v", m.squad.Mode)
	}
	m.squad = m.squad.Refresh(m.ctx())
	if got := len(m.squad.Saved); got != 2 {
		t.Fatalf("the fixture catalogue holds %d squads, want two", got)
	}
	return m
}
