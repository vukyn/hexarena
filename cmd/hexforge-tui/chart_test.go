package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
)

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

// TestTheChartScreenDrawsEveryDeclaredRelation is the property that makes the
// picture worth trusting: a reader who learns the rings has learnt the chart,
// which is only true if every edge is in one of them.
//
// Measured against the resolved edges rather than against the rings, and that is
// the whole of the test. The rings are what was *declared* and the matrix is what
// the game *plays*, so a ring dropped from the picture — or an edge added by a
// mutual pair nobody drew — is a chart that reads complete and is not.
func TestTheChartScreenDrawsEveryDeclaredRelation(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = 200, 60
		m = openChart(t, m)
		drawn := m.screenContent()
		// Whitespace collapsed before scanning, because a ring too long for a
		// line continues under itself: the cross ring's closing bracket sits on
		// the row below the arrow that points at it, and the edge is drawn even
		// though the two are not adjacent in the bytes. A mark only ever appears
		// inside a chain, so joining the lines cannot invent an edge.
		flat := strings.Join(strings.Fields(drawn), " ")

		chart := m.lib.Chart()
		for _, attacker := range element.All() {
			for _, defender := range chart.Strengths(attacker) {
				if !chartShows(flat, attacker, defender) {
					t.Errorf("%s: the chart never shows %v beating %v:\n%s",
						lang, attacker, defender, drawn)
				}
			}
		}
	}
}

// chartShows reports whether the picture puts one element immediately before
// another, in a ring or across a pair.
//
// Adjacency in the drawn text rather than a lookup, because that is what a reader
// does: the claim being tested is that the *picture* carries the edge, and a test
// that asked the chart again would only prove the chart agrees with itself.
func chartShows(drawn string, attacker, defender element.Element) bool {
	for _, mark := range []string{beatsMark, mutualMark} {
		if strings.Contains(drawn, attacker.String()+mark+defender.String()) {
			return true
		}
		// A ring closes on the name it opened with, in brackets.
		if strings.Contains(drawn, attacker.String()+mark+"("+defender.String()+")") {
			return true
		}
	}
	// A pair is drawn once, from whichever side was declared first, and it is
	// true both ways.
	return strings.Contains(drawn, defender.String()+mutualMark+attacker.String())
}

// TestTheChartNamesTheInertElement is the row a picture of edges cannot have.
//
// neutral is in no ring and no pair, so nothing about it is drawn by the arrows —
// and a reference that simply left it out would read as a chart that had lost an
// element rather than one that has an element doing nothing.
func TestTheChartNamesTheInertElement(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = openChart(t, m)
		inert := m.lib.Chart().Inert()
		if len(inert) == 0 {
			t.Fatal("no shipped element is inert, so this measures nothing")
		}
		drawn := m.screenContent()
		if !strings.Contains(drawn, m.text(i18n.ChartInert)) {
			t.Errorf("%s: the chart does not label its inert row:\n%s", lang, drawn)
		}
		for _, member := range inert {
			if !strings.Contains(drawn, member.String()) {
				t.Errorf("%s: the chart never names the inert %v:\n%s", lang, member, drawn)
			}
		}
	}
}

// TestTheChartSpellsItsRatesLikeTheDescriptionsDo is one figure, one spelling.
//
// There are two permille-to-percent conversions in this repository and they round
// differently: forge.Percent keeps a tenth, i18n.Share does not. The chart and an
// element's own description sit one keystroke apart and print the same three
// numbers, so "66.7%" on one and "67%" on the other is the kind of difference a
// reader spends a minute deciding is not a difference.
func TestTheChartSpellsItsRatesLikeTheDescriptionsDo(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	rates := m.lib.Chart().Multipliers()
	want := m.text(i18n.ChartRates, i18n.Share(rates.Advantage),
		i18n.Share(rates.Neutral), i18n.Share(rates.Disadvantage))

	chart := openChart(t, m).screenContent()
	if !strings.Contains(chart, want) {
		t.Errorf("the chart does not print its rates as %q:\n%s", want, chart)
	}
	// And the spelling really is the one the descriptions use, checked against a
	// description rather than against the same call this screen made.
	described := m.lang.DescribeElement(element.Water, m.lib.Chart())
	if !strings.Contains(described, i18n.Share(rates.Advantage)) {
		t.Errorf("water's description does not spell the advantage %q:\n%s",
			i18n.Share(rates.Advantage), described)
	}
}

// TestEscFromTheChartGoesBackToTheListing is the other half of one keystroke.
func TestEscFromTheChartGoesBackToTheListing(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenElements)
	m.elements.cursor = indexOfElement(t, element.Dark)
	back := key(t, openChart(t, m), "esc")
	if back.screen != screenElements {
		t.Fatalf("esc from the chart went to screen %v", back.screen)
	}
	// And the listing is where it was left: a reader who stepped out to look at
	// the shape has not finished with the row they were on.
	if back.elements.cursor != indexOfElement(t, element.Dark) {
		t.Errorf("the elements cursor moved to %d across the trip", back.elements.cursor)
	}
}

// TestTheChartReadsWithoutColour is the palette's own rule, applied to the one
// screen in this program that colours data.
//
// Colour here is decoration: it makes a ring easier to follow with the eye and it
// carries nothing. The tests all run with NO_COLOR set, so what this really pins
// is the shape of the claim — every relation is in the text, so the plain
// rendering is the whole chart rather than a hint that something is missing.
func TestTheChartReadsWithoutColour(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	if !plainTerminal() {
		t.Fatal("the tests are meant to run with NO_COLOR set")
	}
	body, _ := openChart(t, m).chart.view(m)
	if lipgloss.Width(body) == 0 {
		t.Fatal("the chart drew nothing")
	}
	for _, member := range element.All() {
		if !strings.Contains(body, member.String()) {
			t.Errorf("the plain chart never names %v:\n%s", member, body)
		}
	}
	// Every style is the identity under NO_COLOR, so nothing drawn may carry an
	// escape: a screen that wrote one anyway would be one the rule had stopped
	// covering.
	if strings.Contains(body, "\x1b[") {
		t.Errorf("the chart writes escape codes under NO_COLOR:\n%q", body)
	}
}

// TestEveryElementHasAStyle is the completeness the array buys.
//
// The colours are indexed by the enum, so a twelfth element added tomorrow gets a
// zero value rather than a missing key — which draws plain and looks deliberate.
// Only neutral is meant to be undecorated.
func TestEveryElementHasAStyle(t *testing.T) {
	for _, member := range element.All() {
		if member == element.Neutral {
			continue
		}
		if elementColours[member] == "" {
			t.Errorf("%v has no colour, so it draws like the inert element", member)
		}
	}
}
