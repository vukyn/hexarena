package main

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// drawnCarriers is a table built by hand rather than fought.
//
// No battle is run here, for the reason drawnReport gives: what these tests are
// about is what the page says, and a page assembled from a real sweep would take
// minutes to produce and would change its figures every time the data moved — so
// the assertions would have to be loosened until they stopped asserting
// anything. It carries one of every kind of row on purpose: two that priced, one
// whose sweep reversed, one the harness refused and one refused for an ordinary
// reason.
func drawnCarriers() forge.CarriersReport {
	priced := func(id string, worths ...int) forge.CarrierRow {
		weighed := forge.WeighReport{
			Skill: "strike", Field: forge.WeighCrit, Shipped: 0, Seeds: 2000, Band: 16,
		}
		for at, worth := range worths {
			value := []int{0, 200, 400}[at]
			weighed.Rows = append(weighed.Rows, forge.Weighing{
				Value: value, Control: value == 0, Rate: 500 + worth, Turns: 40 - 2*at,
				Tally: forge.Tally{Wins: 2000 + worth*4, Losses: 2000 - worth*4},
			})
		}
		return forge.CarrierRow{Carrier: id, Report: weighed}
	}
	return forge.CarriersReport{
		Skill: "strike", Field: forge.WeighCrit, Shipped: 0,
		Values: []int{0, 200, 400}, Level: 60, Seeds: 2000, Band: 16,
		Rows: []forge.CarrierRow{
			priced("fixture-anime.adept", 0, 86, 140),
			priced("fixture-game.sprout", 0, 2, 6),
			// A sweep that goes up and comes back down prices nothing, whatever
			// the figures beside it look like.
			priced("fixture-anime.wanderer", 0, 62, 1),
			{Carrier: "fixture-anime.leak", Err: &forge.UnevenControlError{
				Skill: "strike", Field: forge.WeighCrit, Value: 0, Rate: 512,
				Tally: forge.Tally{Wins: 2048, Losses: 1952},
			}},
			{Carrier: "fixture-game.mute", Err: errors.New(
				"nothing here prices strike at crit 200: it was cast 40 time(s) and landed none, " +
					"so the row is the absence of a measurement rather than a measurement of nothing")},
		},
		Skipped: []forge.CarrierSkipped{
			{Carrier: "pokemon.charmander", Why: &forge.NotBroughtError{
				Carrier: "pokemon.charmander", Skill: "strike", Level: 60,
				Stage: "Charizard", Brings: []string{"ember", "fire_spin"}}},
			{Carrier: "pokemon.squirtle", Why: &forge.NotBroughtError{
				Carrier: "pokemon.squirtle", Skill: "strike", Level: 60,
				Stage: "Blastoise", Brings: []string{"water_gun", "bubble"}}},
		},
	}
}

// TestRenderCarriersNamesEverythingBehindTheFigures.
//
// The same claim TestRenderWeighNamesEverythingBehindTheFigures makes about one
// weighing, made about a table: every figure on the page is a number somebody
// will author balance data from, and it is only worth authoring from if what
// produced it is on the page. A table adds two things a single report does not
// have — which carriers are *in* it and in what order they sit — and both are
// lines somebody could drop without the columns looking any different.
func TestRenderCarriersNamesEverythingBehindTheFigures(t *testing.T) {
	report := drawnCarriers()
	var drawn strings.Builder
	renderCarriers(&drawn, report)
	page := drawn.String()

	for _, wanted := range []string{
		"strike",
		"crit",
		"level 60",
		"0 (control)",
		"+0.0%",
		"+8.6%",
		"+14.0%",
		"±1.6%",
		"2000 seeds from each slot",
		"battles in all",
		"worth/turns",
		"ties by character id",
	} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the table never says %q:\n%s", wanted, page)
		}
	}
	// Every carrier the sweep looked at is named, priced, refused or skipped —
	// a character that is on none of the three lists has vanished, and a reader
	// counting the cast against the table would never know which.
	for _, row := range report.Rows {
		if !strings.Contains(page, row.Carrier) {
			t.Errorf("the table gives figures without naming the carrier %q:\n%s", row.Carrier, page)
		}
	}
	for _, absent := range report.Skipped {
		if !strings.Contains(page, absent.Carrier) {
			t.Errorf("%q was skipped and the page never says so:\n%s", absent.Carrier, page)
		}
	}
	if !strings.Contains(page, "skipped 2 of 7") {
		t.Errorf("the footer never counts the skipped against the whole cast:\n%s", page)
	}
	// The value the book declares is the control column and reads exactly even.
	// A table whose control column drifted would be a table of prices taken
	// against something other than the shipped skill.
	header := strings.Split(page, "\n")[4]
	for _, column := range []string{"0 (control)", "200", "400", "±", "note"} {
		if !strings.Contains(header, column) {
			t.Errorf("the header %q has no %q column", header, column)
		}
	}
}

// TestTheCarrierFooterRefusesTheCrossCarrierComparison is the sentence this
// whole report is shaped around, and the reason it has no headline number.
//
// A weighing is a price taken against a copy of its own carrier, so two rows
// were fought against two different opponents and are not two readings of one
// quantity. An average of them is a figure with no referent — no opponent it was
// taken against, no board it was taken on — and a figure with no referent
// printed at the top of a table is what a reader quotes. This repository has
// twice been burnt by exactly that: the roster win rate, which is non-monotone
// in ally damage and reversed sign on a placement change, and the mirror-duel
// speed reading, which could not even order swiftness.
func TestTheCarrierFooterRefusesTheCrossCarrierComparison(t *testing.T) {
	var drawn strings.Builder
	renderCarriers(&drawn, drawnCarriers())
	page := drawn.String()
	for _, wanted := range []string{
		"NOT comparable to each other",
		"against a copy of that same carrier",
		"two different opponents",
		"compared only to itself",
		"no headline number and no average",
		"a number with no referent",
		"no figure here is a win rate",
		"does not carry across a data change",
	} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the footer never says %q:\n%s", wanted, page)
		}
	}
	// And there is no headline: nothing on the page averages the rows or totals
	// them. The strongest thing that can be asserted about an absence is that
	// the words which would introduce one are not there.
	for _, forbidden := range []string{"average worth", "mean worth", "overall worth", "across the cast:"} {
		if strings.Contains(page, forbidden) {
			t.Errorf("the page carries %q, and this report has no headline figure:\n%s", forbidden, page)
		}
	}
}

// TestAHarnessRefusalIsMarkedApartFromAnOrdinaryOne.
//
// Across a whole cast most refusals are ordinary — this carrier cannot be priced
// here — and exactly one kind is not: a control that did not come out exactly
// even says the *measurement* leaked, which is a claim about the run rather than
// about the carrier. Drawn the same way they would be the same dash in the same
// column, and the loudest line on the page would be indistinguishable from the
// dullest. This is the failure the sweep was written to stop: the workflow it
// replaces had refusals somebody had to notice by eye.
func TestAHarnessRefusalIsMarkedApartFromAnOrdinaryOne(t *testing.T) {
	var drawn strings.Builder
	renderCarriers(&drawn, drawnCarriers())
	page := drawn.String()

	leaked, ordinary := "", ""
	for _, line := range strings.Split(page, "\n") {
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "fixture-anime.leak"):
			if leaked == "" {
				leaked = line
			}
		case strings.HasPrefix(strings.TrimSpace(line), "fixture-game.mute"):
			if ordinary == "" {
				ordinary = line
			}
		}
	}
	if leaked == "" || ordinary == "" {
		t.Fatalf("one of the two refused rows is not on the page:\n%s", page)
	}
	if !strings.Contains(leaked, "HARNESS") {
		t.Errorf("the leaked row %q is not marked as the harness", leaked)
	}
	if strings.Contains(ordinary, "HARNESS") {
		t.Errorf("an ordinary refusal %q is marked as the harness", ordinary)
	}
	// Both are still rows: a refused carrier stays in the table, because a
	// carrier that cannot be priced is a fact about that carrier.
	for _, refused := range []string{leaked, ordinary} {
		if !strings.Contains(refused, "—") {
			t.Errorf("the refused row %q has no dash where its figures would be", refused)
		}
		if strings.Contains(refused, "0.0%") {
			t.Errorf("the refused row %q prints a figure, and a nought is not a refusal", refused)
		}
	}
	// And the sentence itself is on the page, under the table, where a sentence
	// can be read.
	if !strings.Contains(page, "landed none") {
		t.Errorf("the ordinary refusal is marked and never spelled out:\n%s", page)
	}
	if !strings.Contains(page, "rather than an even 500") {
		t.Errorf("the harness refusal is marked and never spelled out:\n%s", page)
	}
	if !strings.Contains(page, "before believing any other row on this page") {
		t.Errorf("nothing tells the reader what a leaked control costs the rest of the table:\n%s", page)
	}
}

// TestARowThatDoesNotOnlyMoveOneWayIsMarkedOnItsOwnLine.
//
// A dial that is not monotone is not priced, whatever the figures beside it say
// — and on a table the figures are all a reader sees. renderWeigh puts that in a
// footer because it has one sweep to report on; here every row is its own sweep,
// so the answer belongs on the row.
func TestARowThatDoesNotOnlyMoveOneWayIsMarkedOnItsOwnLine(t *testing.T) {
	var drawn strings.Builder
	renderCarriers(&drawn, drawnCarriers())
	for _, line := range strings.Split(drawn.String(), "\n") {
		if !strings.HasPrefix(line, "fixture-anime.wanderer") {
			continue
		}
		if !strings.Contains(line, "not ordered") {
			t.Errorf("a sweep that reverses is drawn as a price: %q", line)
		}
		return
	}
	t.Errorf("the reversing row is not on the page:\n%s", drawn.String())
}

// TestATableCostsFewerSeedsByDefaultThanOneWeighing, because the table
// multiplies a weighing by the cast and the default that makes one question
// quick makes a table something nobody runs twice.
func TestATableCostsFewerSeedsByDefaultThanOneWeighing(t *testing.T) {
	if defaultCarrierSeeds >= defaultWeighSeeds {
		t.Errorf("a table defaults to %d seeds per carrier and one weighing to %d",
			defaultCarrierSeeds, defaultWeighSeeds)
	}
	if defaultCarrierSeeds < 1 {
		t.Errorf("a table defaults to %d seeds, and a sweep over none measures nothing",
			defaultCarrierSeeds)
	}
}

// TestWeighAcrossCarriersRefusesAnArgumentListItCannotAnswer.
//
// Each of these would otherwise produce a table, and a table is what an author
// reads as an answer. --carriers in particular takes a name rather than a bool:
// a mistyped selection that silently priced the whole cast would be a run
// nobody asked for, paid for once per carrier.
func TestWeighAcrossCarriersRefusesAnArgumentListItCannotAnswer(t *testing.T) {
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
		{"a character as well as a skill", []string{"--data", dir, "--carriers", "all", who, what,
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"no skill at all", []string{"--data", dir, "--carriers", "all",
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"a selection nobody declared", []string{"--data", dir, "--carriers", "everyone", what,
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"no field", []string{"--data", dir, "--carriers", "all", what,
			"--values", "1100", "--seeds", seeds}},
		{"no values", []string{"--data", dir, "--carriers", "all", what,
			"--field", "power", "--seeds", seeds}},
		{"no skill by that name", []string{"--data", dir, "--carriers", "all", "nothing_at_all",
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"a skill nobody brings", []string{"--data", dir, "--carriers", "all", "solar_beam",
			"--field", "power", "--values", "1100", "--seeds", seeds}},
		{"a value the parser refuses", []string{"--data", dir, "--carriers", "all", what,
			"--field", "accuracy", "--values", "5000", "--seeds", seeds}},
		{"no battles at all", []string{"--data", dir, "--carriers", "all", what,
			"--field", "power", "--values", "1100", "--seeds", "0"}},
	} {
		if err := runWeigh(test.args); err == nil {
			t.Errorf("%s was accepted: hexforge weigh %v", test.name, test.args)
		}
	}
}

// TestTheSkillMaySitBetweenTheFlags is the argv shape --carriers forces.
//
// The selection has to come first, because it decides whether there is a
// character operand at all — so the skill lands between two flags, which is a
// form flag.Parse alone cannot read and parseArgs now can. Without it the skill
// would be taken as an operand and every flag after it handed back as four more.
func TestTheSkillMaySitBetweenTheFlags(t *testing.T) {
	dir := scratchData(t)
	if err := runWeigh([]string{"--data", dir, "--carriers", "all", "strike",
		"--field", "power", "--values", "1100", "--seeds", "2"}); err != nil {
		t.Fatalf("a skill between two flags was refused: %v", err)
	}
}

// TestParseArgsReadsAnOperandBetweenTwoFlags is the same claim taken directly,
// because runWeigh above would also pass if the flags were merely tolerated
// rather than read.
func TestParseArgsReadsAnOperandBetweenTwoFlags(t *testing.T) {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	first := set.String("carriers", "", "")
	second := set.Int("level", 60, "")
	operands, err := parseArgs(set, []string{"--carriers", "all", "razor_leaf", "--level", "30"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(operands) != 1 || operands[0] != "razor_leaf" {
		t.Errorf("the operands are %v, want [razor_leaf]", operands)
	}
	if *first != "all" || *second != 30 {
		t.Errorf("the flags read %q and %d, want all and 30", *first, *second)
	}
}

// TestWasSetTellsANumberTypedFromANumberDefaulted.
//
// It is the whole of what chooses between defaultWeighSeeds and
// defaultCarrierSeeds, and flag has no other way to ask: a --seeds of 10000
// typed by hand and a --seeds nobody typed are the same int. Getting it wrong
// would silently overrule an author who named a count.
func TestWasSetTellsANumberTypedFromANumberDefaulted(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		given bool
	}{
		{"nobody said", []string{"razor_leaf"}, false},
		{"a number typed", []string{"razor_leaf", "--seeds", "50"}, true},
		{"the default typed out", []string{"razor_leaf", "--seeds", "10000"}, true},
	} {
		set := flag.NewFlagSet("test", flag.ContinueOnError)
		set.SetOutput(io.Discard)
		set.Int("seeds", defaultWeighSeeds, "")
		if _, err := parseArgs(set, test.args); err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got := wasSet(set, "seeds"); got != test.given {
			t.Errorf("%s reads given=%v, want %v", test.name, got, test.given)
		}
	}
}
