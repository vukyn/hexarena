package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/forge"
)

// drawnReport is a report built by hand rather than fought.
//
// No battle is run here on purpose. What these tests are about is what the page
// says, and a page assembled from a real sweep would take seconds to produce and
// would change its figures every time the data moved — so the assertions would
// have to be loosened until they stopped asserting anything.
func drawnReport() forge.WeighReport {
	affinity, _ := element.Single(element.Grass)
	return forge.WeighReport{
		Carrier: forge.Duellist{
			ID: "fixture-anime.adept", Name: "Example Adept", Level: 60,
			Stage: "Example Adept", Affinity: affinity,
			Skills:   []string{"strike", "riptide", "guard_wall", "purify"},
			Passives: []string{"endurance"},
		},
		Skill: "strike", Field: forge.WeighCrit, Shipped: 0,
		Seeds: 10000, Band: 8,
		Rows: []forge.Weighing{
			{Value: 0, Control: true, Rate: 500, Turns: 37, Edge: 327,
				Tally:   forge.Tally{Wins: 10000, Losses: 10000},
				Strikes: forge.Strikes{Cast: 125991, Landed: 226151}},
			{Value: 200, Rate: 586, Turns: 35, Edge: 293,
				Tally:   forge.Tally{Wins: 11737, Losses: 8263},
				Strikes: forge.Strikes{Cast: 124133, Landed: 222747, Critical: 44714}},
		},
	}
}

// TestRenderWeighNamesEverythingBehindTheFigures.
//
// A price is a number somebody will author balance data from, and it is only
// worth authoring from if what produced it is on the page: which character, at
// what level, in what form, carrying which kit, which skill and which field were
// moved, what the book declares for it, over how many battles, and which row is
// the control. Each is asserted rather than eyeballed, because each is a line
// somebody could drop without the table looking any different.
func TestRenderWeighNamesEverythingBehindTheFigures(t *testing.T) {
	report := drawnReport()
	var drawn strings.Builder
	renderWeigh(&drawn, report)
	page := drawn.String()

	for _, wanted := range []string{
		"fixture-anime.adept",
		"Example Adept",
		"level 60",
		"strike",
		"crit",
		"(control)",
		"first move",
		"turns",
		"landed",
		"10000 seeds from each slot",
		"40000 battles in all",
	} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the report never says %q:\n%s", wanted, page)
		}
	}
	for _, fielded := range report.Carrier.Skills {
		if !strings.Contains(page, fielded) {
			t.Errorf("the report gives figures without naming the fielded skill %q:\n%s", fielded, page)
		}
	}
	for _, trait := range report.Carrier.Passives {
		if !strings.Contains(page, trait) {
			t.Errorf("the report gives figures without naming the fielded trait %q:\n%s", trait, page)
		}
	}
	// worth and turns are co-equal headline columns, so the footer has to read
	// both. A page reporting only whether the rate was ordered would send a
	// reader back to the one column that cannot see a lumpy effect.
	for _, wanted := range []string{"worth:", "turns:"} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the footer never reports %q on its own:\n%s", wanted, page)
		}
	}
	// And they are drawn beside each other rather than at opposite ends of a
	// wide table, because a reader who has to look for the second column will
	// read the first alone.
	header := strings.SplitN(page, "\n", 6)[4]
	worth, turns := strings.Index(header, "worth"), strings.Index(header, "turns")
	if worth < 0 || turns < 0 || turns < worth {
		t.Fatalf("the header %q does not put worth before turns", header)
	}
	if between := strings.Fields(header[worth:turns]); len(between) > 2 {
		t.Errorf("%d columns sit between worth and turns in %q, so they are not adjacent",
			len(between)-1, header)
	}
}

// TestEveryRateIsPrintedWithItsBand.
//
// A worth on its own is a number an author will act on, and half of these
// numbers are inside the noise. The band is what says which half, so a row
// without one is a row that reads as a finding whatever it is.
func TestEveryRateIsPrintedWithItsBand(t *testing.T) {
	report := drawnReport()
	var drawn strings.Builder
	renderWeigh(&drawn, report)
	lines := strings.Split(drawn.String(), "\n")

	rows := 0
	for _, line := range lines {
		// The rows are the lines carrying a signed worth, which is what the band
		// belongs to.
		if !strings.Contains(line, "+0.0%") && !strings.Contains(line, "+8.6%") {
			continue
		}
		rows++
		if !strings.Contains(line, "±0.8%") {
			t.Errorf("a row prints a worth with no band beside it: %q", line)
		}
	}
	if rows != len(report.Rows) {
		t.Errorf("%d of %d rows were found on the page", rows, len(report.Rows))
	}
	// And the band is stated once in words as well, because a column of ± signs
	// does not say what they are two of.
	if !strings.Contains(drawn.String(), "the band is ±0.8%") {
		t.Errorf("the footer never says what the band is:\n%s", drawn.String())
	}
}

// TestTheFooterRefusesTheRosterReading is the sentence this whole instrument
// exists because of.
//
// The measurement it replaced was a roster win rate, and the roster win rate is
// not monotone in ally damage: giving the ally more damage lowered it by about
// as much as adding a crit chance did, and the same change read positive before
// an unrelated placement move and negative after. So a figure taken here has to
// arrive with what it is not attached to it — a reader who carries it back to
// the roster is making exactly the mistake that produced the wrong answer the
// first time.
func TestTheFooterRefusesTheRosterReading(t *testing.T) {
	var drawn strings.Builder
	renderWeigh(&drawn, drawnReport())
	page := drawn.String()
	for _, wanted := range []string{
		"against a copy of itself",
		"not a win rate",
		"roster",
		"does not carry across a data change",
	} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the footer never says %q:\n%s", wanted, page)
		}
	}
}

// TestAnUnorderedSweepSaysSoRatherThanPrintingTheRows.
//
// A dial that is not monotone is not priced, whatever the figures beside it say.
// The rows still print — a reader wants to see what happened — but the footer has
// to refuse them, in both columns and separately, because the two can disagree.
func TestAnUnorderedSweepSaysSoRatherThanPrintingTheRows(t *testing.T) {
	report := drawnReport()
	report.Rows = append(report.Rows, forge.Weighing{
		Value: 400, Rate: 450, Turns: 44, Edge: 261,
		Tally:   forge.Tally{Wins: 9000, Losses: 11000},
		Strikes: forge.Strikes{Cast: 122483, Landed: 219775, Critical: 88253},
	})
	var drawn strings.Builder
	renderWeigh(&drawn, report)
	page := drawn.String()
	if !strings.Contains(page, "does NOT only move one way") {
		t.Errorf("a sweep that reverses is reported as ordered:\n%s", page)
	}
	if !strings.Contains(page, "400") {
		t.Errorf("the reversing row was not printed:\n%s", page)
	}
}

// TestWeighRefusesAnArgumentListItCannotAnswer.
//
// Every one of these would otherwise produce a table, and a table is what an
// author reads as an answer. --values in particular is required with no default:
// a default range would be the tool guessing at which values are worth trying,
// and a guess printed in a column is indistinguishable from a finding.
func TestWeighRefusesAnArgumentListItCannotAnswer(t *testing.T) {
	dir := scratchData(t)
	const (
		who   = "fixture-anime.adept"
		what  = "strike"
		seeds = "2"
	)
	for _, test := range []struct {
		name string
		args []string
	}{
		{"no operands at all", []string{"--data", dir}},
		{"a character and no skill", []string{"--data", dir, who,
			"--field", "power", "--values", "1100"}},
		{"three operands", []string{"--data", dir, who, what, "extra",
			"--field", "power", "--values", "1100"}},
		{"no field", []string{"--data", dir, who, what, "--values", "1100"}},
		{"a field nobody declared", []string{"--data", dir, who, what,
			"--field", "self_gradient", "--values", "1100"}},
		{"no values", []string{"--data", dir, who, what, "--field", "power"}},
		{"an empty value list", []string{"--data", dir, who, what,
			"--field", "power", "--values", ""}},
		{"a value that is not a number", []string{"--data", dir, who, what,
			"--field", "power", "--values", "1100,lots"}},
		{"an empty entry in the list", []string{"--data", dir, who, what,
			"--field", "power", "--values", "1100,,1200"}},
		{"no battles at all", []string{"--data", dir, who, what,
			"--field", "power", "--values", "1100", "--seeds", "0"}},
		{"nobody by that name", []string{"--data", dir, "nobody.at.all", what,
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"no skill by that name", []string{"--data", dir, who, "nothing_at_all",
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"a skill the carrier does not bring", []string{"--data", dir, who, "bolt",
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"a value the parser refuses", []string{"--data", dir, who, what,
			"--field", "accuracy", "--values", "5000", "--seeds", seeds}},
	} {
		if err := runWeigh(test.args); err == nil {
			t.Errorf("%s was accepted: hexforge weigh %v", test.name, test.args)
		}
	}
}

// TestAValueListIsReadAsWrittenOrRefused, because a sweep silently dropping an
// entry would report a curve with a hole in it and no sign of one.
func TestAValueListIsReadAsWrittenOrRefused(t *testing.T) {
	got, err := parseValues(" 100, 200 ,300")
	if err != nil {
		t.Fatalf("a spaced list was refused: %v", err)
	}
	if len(got) != 3 || got[0] != 100 || got[1] != 200 || got[2] != 300 {
		t.Errorf("the list read as %v", got)
	}
	if _, err := parseValues("100,"); err == nil {
		t.Error("a trailing comma was read as a two-value list")
	}
	if _, err := parseValues("100,two"); err == nil {
		t.Error("a word was read as a number")
	}
}
