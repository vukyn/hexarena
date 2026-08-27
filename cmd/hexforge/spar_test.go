package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// TestRenderSparNamesEverythingBehindTheFigures.
//
// A win rate is a number somebody will act on, and it is only worth acting on if
// what produced it is on the page: which kit was fielded, at what level, over how
// many battles, and which row is the control rather than an opponent. Each of
// those is asserted here rather than eyeballed, because each of them is a line
// somebody could drop without the table looking any different.
func TestRenderSparNamesEverythingBehindTheFigures(t *testing.T) {
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	const seeds = 5
	report, err := lib.Spar("fixture-anime.adept", progression.LevelCap, seeds)
	if err != nil {
		t.Fatalf("spar: %v", err)
	}
	var drawn strings.Builder
	renderSpar(&drawn, report)
	page := drawn.String()

	for _, wanted := range []string{
		"fixture-anime.adept",
		report.Challenger.Stage,
		"(control)",
		"first move",
	} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the report never says %q:\n%s", wanted, page)
		}
	}
	for _, fielded := range report.Challenger.Skills {
		if !strings.Contains(page, fielded) {
			t.Errorf("the report gives figures without naming the fielded skill %q:\n%s", fielded, page)
		}
	}
	// The number of battles is stated, and it is the number that was actually
	// fought: a reader who multiplies the seeds by the rows has to get it.
	if !strings.Contains(page, "5 seeds from each slot") {
		t.Errorf("the report does not say how many seeds it used:\n%s", page)
	}
}

// TestARateKeepsTheDecimalThatMatters.
//
// forge.Percent drops a trailing zero, which is right in a sentence and wrong in
// a column: a rate is the number an author tunes against, half a point either
// side of even is the difference between a character that belongs and one that
// does not, and a column of 50%, 100.0%, 39% cannot be read down the page.
func TestARateKeepsTheDecimalThatMatters(t *testing.T) {
	for _, test := range []struct {
		permille int
		want     string
	}{
		{0, "0.0%"},
		{1, "0.1%"},
		{495, "49.5%"},
		{500, "50.0%"},
		{505, "50.5%"},
		{1000, "100.0%"},
		{-440, "-44.0%"},
	} {
		if got := forge.PercentInColumn(test.permille); got != test.want {
			t.Errorf("%d parts per thousand printed as %q, wanted %q", test.permille, got, test.want)
		}
	}
	// And the sibling still shortens, which is what makes them two functions.
	if got := forge.Percent(500); got != "50%" {
		t.Errorf("a sentence's percentage printed as %q rather than the short form", got)
	}
}

// TestAnEdgePrintsItsDirection. The first slot being worth four points and the
// second slot being worth four points are the same figure and opposite findings.
func TestAnEdgePrintsItsDirection(t *testing.T) {
	if got := signed(440); got != "+44.0%" {
		t.Errorf("an advantage to the first slot printed as %q", got)
	}
	if got := signed(-440); got != "-44.0%" {
		t.Errorf("an advantage to the second slot printed as %q", got)
	}
	if got := signed(0); got != "+0.0%" {
		t.Errorf("a pairing the slot does not decide printed as %q", got)
	}
}

// TestSparRefusesAnArgumentListItCannotAnswer, because a subcommand that took
// two ids and measured the first would be measuring something nobody asked for.
func TestSparRefusesAnArgumentListItCannotAnswer(t *testing.T) {
	dir := scratchData(t)
	for _, args := range [][]string{
		{"--data", dir},
		{"--data", dir, "fixture-anime.adept", "fixture-game.sprout"},
		{"--data", dir, "nobody.at.all"},
		{"--data", dir, "fixture-anime.adept", "--seeds", "0"},
	} {
		if err := runSpar(args); err == nil {
			t.Errorf("hexforge spar %v was accepted", args)
		}
	}
}
