package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// TestEveryRaiseTargetNamesAScreenInThisClient is what stops a raise from
// silently doing nothing.
//
// A screen asks for a draw.Target and this client's map turns it into one of its
// own views. A target with no entry is a keystroke that reads as broken: the
// screen did everything right, the reader pressed the key the footer names, and
// nothing happens — the same shape as a screen slipping out of everyScreen,
// which this repository has recorded five times.
//
// ⚠️ It walks screen.TargetCount rather than the map, because the failure being
// guarded against is a target somebody added over there and did not list here.
// Ranging over raiseTargets would ask the map whether it holds what it holds.
func TestEveryRaiseTargetNamesAScreenInThisClient(t *testing.T) {
	for value := 1; value < draw.TargetCount; value++ {
		target := draw.Target(value)
		if _, known := raiseTargets[target]; !known {
			t.Errorf("draw.Target %v (%d) names no screen in this client, so a raise carrying it does nothing",
				target, value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the package does not declare is a target this client would answer to
	// and no screen could ask for.
	if got, want := len(raiseTargets), draw.TargetCount-1; got != want {
		t.Errorf("raiseTargets holds %d entries against the %d targets declared besides NoTarget",
			got, want)
	}
	// NoTarget is what every action that is not a raise carries, so a screen for
	// it would be a screen reachable by every Quit and every Back.
	if _, known := raiseTargets[draw.NoTarget]; known {
		t.Error("draw.NoTarget names a screen, and it is what a non-raise action carries")
	}
}

// TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack is the one-slot memory,
// asserted as a pair because either half alone passes for the wrong reason.
//
// Remembering without forgetting leaves a reader who arrived through a trait and
// then came back through the menu being sent to the trait; forgetting without
// remembering is the hard-coded menu this step exists to remove.
//
// It goes through navigate rather than through a screen's keys so that it says
// what it is about — the per-screen paths are driven with real keystrokes in
// chart_test.go and trait_status_test.go.
func TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)

	raised, _ := m.navigate(screenPassives, draw.Action{
		Kind: draw.Raise, Target: draw.Statuses,
		Subject: draw.Subject{Kind: draw.StatusSubject, ID: firstStatusID(t, m)},
	})
	after, ok := raised.(model)
	if !ok {
		t.Fatalf("navigate returned %T, want the model", raised)
	}
	if after.screen != screenStatuses {
		t.Fatalf("a raise of the statuses reference landed on screen %v", after.screen)
	}
	if after.raisedFrom != screenPassives {
		t.Errorf("the raise remembered screen %v as the way back, want the traits listing",
			after.raisedFrom)
	}

	back, _ := after.navigate(screenStatuses, draw.Action{Kind: draw.Back})
	returned := back.(model)
	if returned.screen != screenPassives {
		t.Fatalf("back from a raised screen went to screen %v", returned.screen)
	}
	if returned.raisedFrom != screenMenu {
		t.Errorf("the way back survived being used: it still reads screen %v", returned.raisedFrom)
	}

	// And a screen nobody raised goes to the menu, which is what the four
	// listings with no raiser above them have always done.
	plain, _ := m.navigate(screenBuilds, draw.Action{Kind: draw.Back})
	if got := plain.(model).screen; got != screenMenu {
		t.Errorf("back from a screen reached through the menu went to screen %v", got)
	}
}

// TestEverySubjectKindIsAppliedByThisClient is the other half of a total raise,
// and it is the one #203 could not write.
//
// A screen names what a raise is about with a draw.Subject and this client's
// applier puts it where it belongs. A kind with no entry is a keystroke that
// reads as broken in a quieter way than a missing target does: the screen opens,
// draws nothing, and looks like a screen with nothing on it rather than like a
// raise that went wrong.
//
// ⚠️ That is exactly what `Action.Focus` was — one undeclared case. It answered
// only the statuses reference, every other target reported not-found, and a
// declined raise is silent by design, so a subject aimed anywhere else was
// indistinguishable from a key nobody pressed. Nothing counted the cases, so
// nothing could walk them.
//
// ⚠️ It walks screen.SubjectKindCount rather than the map, because the failure
// being guarded against is a kind somebody added over there and did not list
// here. Ranging over subjects would ask the map whether it holds what it holds.
func TestEverySubjectKindIsAppliedByThisClient(t *testing.T) {
	for value := 1; value < draw.SubjectKindCount; value++ {
		kind := draw.SubjectKind(value)
		if _, known := subjects[kind]; !known {
			t.Errorf("draw.SubjectKind %v (%d) is applied by nothing in this client, so a raise "+
				"carrying it hands the describer nothing and draws an empty screen", kind, value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the package does not declare is a subject this client would apply and
	// no screen could ask for.
	if got, want := len(subjects), draw.SubjectKindCount-1; got != want {
		t.Errorf("subjects holds %d entries against the %d kinds declared besides NoSubject",
			got, want)
	}
	// NoSubject is what every raise about nothing carries, so an applier for it
	// would be one every Quit and every Back could reach.
	if _, known := subjects[draw.NoSubject]; known {
		t.Error("draw.NoSubject is applied by an entry, and it is what a raise about nothing carries")
	}
}

// TestEveryScreenThatAsksAnswersItsOwnQuestion is the third totality walk in
// this file, and it guards the quietest of the three failures.
//
// A screen raises a guard, the reader reads the question and presses y, and the
// question comes down. If the asking screen has no entry in confirmedBy, that is
// **all** that happens: the work is not discarded, the squad is not deleted, and
// nothing on screen says the answer went nowhere — a keystroke that reads as
// having worked. A missing target draws no screen and a missing subject applier
// draws an empty one; this one draws the screen the reader was already looking
// at.
//
// ⚠️ It walks guardAskers rather than the map, for the reason the two above walk
// their counts: the failure being guarded against is a screen that grew a
// question and no answer, and ranging over confirmedBy would ask the map whether
// it holds what it holds.
//
// ⚠️ **And it proves presence, not effect.** A dispatch entry that exists and
// does nothing passes this completely — the same blind spot #207 measured on an
// applier table. What holds the other half is one behaviour test per confirm:
// TestLeavingAnEditedFormAsksFirst in tui_test.go for the character form, and
// the four in guard_test.go for the other four.
func TestEveryScreenThatAsksAnswersItsOwnQuestion(t *testing.T) {
	for _, asked := range guardAskers {
		if _, known := confirmedBy[asked]; !known {
			t.Errorf("screen %v raises a guard and answers none, so a confirmed y on it "+
				"takes the question down and does nothing", asked)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// screen that never asks is an answer to a question nobody can raise.
	if got, want := len(confirmedBy), len(guardAskers); got != want {
		t.Errorf("confirmedBy holds %d entries against the %d screens that ask", got, want)
	}
}

// TestEveryPickDestinationLandsSomewhere is the fourth totality walk in this
// file, and the widest of the four: ten cases against the guard's four.
//
// A picker is closed with enter, the list comes down, and the reader believes
// they chose. If the destination has no entry in pickedInto that is **all** that
// happens — the field is unchanged and nothing on screen says the answer went
// nowhere. It is confirmedBy's failure exactly, and it is easier to make: a
// screen grows one guard and it is noticed, while the skill form alone raises
// six pickers that differ in nothing but which field they fill.
//
// ⚠️ It walks pickDestCount rather than the map, for the reason the three walks
// above walk theirs — and here the count is doing more than a list would.
// Adding a destination is adding a constant, and a constant added above
// pickDestCount enters this walk without anybody remembering to enrol it, which
// is a failure guardAskers' hand-written list can still have.
//
// ⚠️ **And it proves presence, not effect.** An entry that exists and writes the
// wrong field passes this completely, which is the #207 shape and the one #214
// measured on the guard. What holds the other half is one behaviour test per
// destination, driven through the real keys: TestEachAllowlistPickLandsInItsOwnField
// and TestTheCharacterFormsTwoPicksLandInTheirOwnFields in picked_test.go for
// seven of the ten, and the four named in their doc comment for the rest.
func TestEveryPickDestinationLandsSomewhere(t *testing.T) {
	for value := 1; value < int(pickDestCount); value++ {
		into := pickDest(value)
		if _, known := pickedInto[into]; !known {
			t.Errorf("pickDest %d lands nowhere in this client, so a picker closed with "+
				"enter on it takes the list down and writes nothing", value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the enum does not declare is a landing no picker could reach.
	if got, want := len(pickedInto), int(pickDestCount)-1; got != want {
		t.Errorf("pickedInto holds %d entries against the %d destinations declared besides pickNowhere",
			got, want)
	}
	// pickNowhere is the zero value, which a pickState built by hand carries, so
	// a landing for it would be one every un-destined picker fell into.
	if _, known := pickedInto[pickNowhere]; known {
		t.Error("pickNowhere lands somewhere, and it is what a picker with no destination carries")
	}
}

// TestARaiseAboutNothingStillArrives is the case the loop above cannot state: a
// Raise carrying no subject is ordinary rather than a subject nobody applied.
//
// The elements listing opening the chart is a whole screen rather than a thing on
// one, so it names none — and if applySubject treated the zero kind as unknown,
// that raise would decline and the g key would silently stop working.
func TestARaiseAboutNothingStillArrives(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	after, _ := m.navigate(screenElements, draw.Action{Kind: draw.Raise, Target: draw.Chart})
	if got := after.(model).screen; got != screenChart {
		t.Errorf("a raise carrying no subject landed on screen %v, want the chart", got)
	}
}

// TestARaiseNobodyCanLandDeclinesTheWholeTrip is the staying-put the traits
// listing used to do for itself.
//
// A subject the raised screen cannot find must leave the reader where they are,
// rather than opening the reference on whatever its cursor happened to be on:
// that would answer a question nobody asked, and it would read as the jump
// working.
func TestARaiseNobodyCanLandDeclinesTheWholeTrip(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m.screen = screenPassives

	after, _ := m.navigate(screenPassives, draw.Action{
		Kind: draw.Raise, Target: draw.Statuses,
		Subject: draw.Subject{Kind: draw.StatusSubject, ID: "no_such_status"},
	})
	stayed := after.(model)
	if stayed.screen != screenPassives {
		t.Errorf("a raise carrying an unfindable id moved to screen %v", stayed.screen)
	}
	if stayed.raisedFrom != screenMenu {
		t.Errorf("a declined raise wrote screen %v as a way back", stayed.raisedFrom)
	}
}

// firstStatusID is a status the reference really holds, so a focus that fails is
// a fault rather than a fixture.
func firstStatusID(t *testing.T, m model) string {
	t.Helper()
	for _, row := range m.statuses.Rows {
		if !row.Heading {
			return row.Kind.ID
		}
	}
	t.Fatal("the statuses reference holds no status, so a focus measures nothing")
	return ""
}
