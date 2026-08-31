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
		Kind: draw.Raise, Target: draw.Statuses, Focus: firstStatusID(t, m),
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

// TestARaiseNobodyCanLandDeclinesTheWholeTrip is the staying-put the traits
// listing used to do for itself.
//
// A focus the raised screen cannot find must leave the reader where they are,
// rather than opening the reference on whatever its cursor happened to be on:
// that would answer a question nobody asked, and it would read as the jump
// working.
func TestARaiseNobodyCanLandDeclinesTheWholeTrip(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m.screen = screenPassives

	after, _ := m.navigate(screenPassives, draw.Action{
		Kind: draw.Raise, Target: draw.Statuses, Focus: "no_such_status",
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
