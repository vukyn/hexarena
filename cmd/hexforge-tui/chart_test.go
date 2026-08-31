package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

// What is left here is the client's half of the chart: the one key that reaches
// it and the one that leaves. Everything about what it *draws* — the loop, the
// inert row, the rates, the plain rendering — moved to internal/screen with the
// screen, because a drawing is not something this binary decides.

// openChart is the g the elements listing offers, which is the only door to this
// screen. Driven rather than assigned: a view with no key into it is a view
// nobody can reach, and assigning m.screen would pass either way.
func openChart(t *testing.T, m model) model {
	t.Helper()
	m = m.enter(screenElements)
	m = send(t, m, tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.screen != screenChart {
		t.Fatalf("g on the elements listing left the reader on screen %v", m.screen)
	}
	return m
}

// TestEscFromTheChartGoesBackToTheListing is the other half of one keystroke.
//
// The screen itself only asks for draw.Back; where that lands is this client's
// one-slot raisedFrom, so the claim is the client's and stays here.
func TestEscFromTheChartGoesBackToTheListing(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenElements)
	m.elements.Cursor = indexOfElement(t, element.Dark)
	back := key(t, openChart(t, m), "esc")
	if back.screen != screenElements {
		t.Fatalf("esc from the chart went to screen %v", back.screen)
	}
	// And the listing is where it was left: a reader who stepped out to look at
	// the shape has not finished with the row they were on.
	if back.elements.Cursor != indexOfElement(t, element.Dark) {
		t.Errorf("the elements cursor moved to %d across the trip", back.elements.Cursor)
	}
}
