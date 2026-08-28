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

// TestTheDrawnLoopIsTheDeclaredRing is the property that makes the picture worth
// trusting: what a reader follows round the loop has to be the ring the chart is
// declared with, in that order.
//
// It reads the drawing back rather than asking the chart again — a test that
// asked the chart twice would only prove the chart agrees with itself. Following
// the picture is what a reader does, so following the picture is what is checked:
// along the top row left to right, then along the return leg **right to left**,
// which is the way its arrows point.
//
// Two of the edges in a split ring are drawn by the rules at the corners rather
// than by any adjacent pair of names — the last name on the top row reaches the
// first name of the return leg down the right-hand edge, and the last name of the
// return leg reaches the first name of all up the left. Those are exactly the
// edges a text-adjacency test cannot see and a reader can, which is why this
// walks the order instead of scanning for pairs.
func TestTheDrawnLoopIsTheDeclaredRing(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	cycles := m.lib.Chart().Cycles()
	if len(cycles) == 0 {
		t.Fatal("the shipped chart declares no ring, so this measures nothing")
	}
	split := 0
	for _, cycle := range cycles {
		lines := ringLines(elementIDs(cycle.Chain), chartRoom())
		if len(lines) > 2 {
			split++
		}
		got := walkLoop(t, lines)
		want := elementIDs(cycle.Chain)
		if len(got) != len(want) {
			t.Errorf("%s: the drawing walks %v, want the ring %v", cycle.Name, got, want)
			continue
		}
		for index := range want {
			if got[index] != want[index] {
				t.Errorf("%s: the drawing walks %v, want the ring %v", cycle.Name, got, want)
				break
			}
		}
	}
	// And at least one ring really is split over two legs, or the harder half of
	// walkLoop — the reversed return leg — was never exercised.
	if split == 0 {
		t.Error("no shipped ring is drawn over two legs, so the return leg is untested")
	}
}

// walkLoop reads a drawn ring back into the order a reader follows it in.
//
// The top leg reads left to right and the return leg right to left, which is the
// way each one's arrows point. Anything that is not an element id — the rules,
// the corners, the marks — is skipped, so the walk is the names alone.
func walkLoop(t *testing.T, lines []string) []string {
	t.Helper()
	if len(lines) == 0 {
		t.Fatal("the ring drew nothing")
	}
	out := namesIn(lines[0])
	switch len(lines) {
	case 2:
		// One leg and the rule that closes it under.
		return out
	case 3:
		// Two legs and the connector between their ends.
		return append(out, reversed(namesIn(lines[2]))...)
	}
	t.Fatalf("a ring drew %d lines, want one leg or two", len(lines))
	return nil
}

// namesIn is every element id on one line, in the order it is written.
func namesIn(line string) []string {
	out := make([]string, 0, 8)
	for _, word := range strings.Fields(line) {
		word = strings.Trim(word, ",'`|<->-()")
		if _, err := element.Parse(word); err == nil {
			out = append(out, word)
		}
	}
	return out
}

// TestEveryRingClosesAtTheFloor is the rule an author has to be held to when a
// twelfth element is added: **the drawing has to still be a drawing**.
//
// The picture is generated, so adding an element cannot make it *wrong* — it
// redraws itself, which is the whole reason it is allowed to be a picture at all.
// What adding an element can do is make it not fit, and then ringLines falls back
// to a plain chain and the chart quietly stops being a loop. That is the failure
// this holds shut, at the floor, which is the width every reader gets.
//
// It also checks the box actually closes: every line the same width, the opening
// corner above the closing one, and each member drawn exactly once. A loop that
// is one cell out is a loop nobody trusts.
func TestEveryRingClosesAtTheFloor(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	cycles := m.lib.Chart().Cycles()
	if len(cycles) == 0 {
		t.Fatal("the shipped chart declares no ring, so this measures nothing")
	}
	for _, cycle := range cycles {
		names := elementIDs(cycle.Chain)
		lines := ringLines(names, chartRoom())
		if len(lines) == 1 {
			t.Errorf("%s fell back to a plain chain at %d columns; it no longer draws as a loop: %q",
				cycle.Name, chartRoom(), lines[0])
			continue
		}
		width := len(lines[0])
		if width > chartRoom() {
			t.Errorf("%s is %d columns wide, over the %d the floor gives it",
				cycle.Name, width, chartRoom())
		}
		for index, line := range lines {
			if len(line) != width {
				t.Errorf("%s: line %d is %d cells and the first is %d, so the box does not close:\n%s",
					cycle.Name, index, len(line), width, strings.Join(lines, "\n"))
			}
		}
		if opened, closed := lines[0][0], lines[len(lines)-1][0]; opened != ',' || (closed != '`' && closed != '\'') {
			t.Errorf("%s opens with %q and closes with %q:\n%s",
				cycle.Name, string(opened), string(closed), strings.Join(lines, "\n"))
		}
		drawn := strings.Join(lines, "\n")
		for _, name := range names {
			if strings.Count(drawn, name) != 1 {
				t.Errorf("%s draws %q %d times, want once:\n%s",
					cycle.Name, name, strings.Count(drawn, name), drawn)
			}
		}
	}
}

// TestARingTooWideFallsBackToAChain is the escape hatch, and it is here so that
// the fallback is a decision rather than a surprise.
//
// A ring wide enough to beat two legs cannot be drawn as a loop a reader can
// follow — a third leg would run left to right again and the arrows would stop
// meaning anything. What it must not do is draw a broken box: it drops to the
// plain chain, which is honest about being a list, and still names every member.
func TestARingTooWideFallsBackToAChain(t *testing.T) {
	huge := make([]string, 0, 8)
	for _, letter := range "abcdefgh" {
		huge = append(huge, strings.Repeat(string(letter), 20))
	}
	lines := ringLines(huge, chartRoom())
	if len(lines) != 1 {
		t.Fatalf("a ring too wide for two legs drew %d lines, want the chain fallback:\n%s",
			len(lines), strings.Join(lines, "\n"))
	}
	for _, name := range huge {
		if !strings.Contains(lines[0], name) {
			t.Errorf("the fallback drops %q: %s", name, lines[0])
		}
	}
	// And it still says the ring closes, which is the one thing the box was
	// carrying that a list does not.
	if !strings.Contains(lines[0], "("+huge[0]+")") {
		t.Errorf("the fallback does not close on its first member: %s", lines[0])
	}
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
