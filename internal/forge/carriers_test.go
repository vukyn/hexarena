package forge

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// carriersRequest is the sweep every test here starts from, with the values it
// wants. It is weighRequest with the character taken off, which is the whole
// difference between the two instruments.
func carriersRequest(field WeighField, values ...int) CarriersRequest {
	return CarriersRequest{
		Skill: weighSkill, Field: field, Values: values,
		Level: progression.LevelCap, Seeds: weighSeeds,
	}
}

// bringers is who the fixture cast actually fields the skill with, worked out
// the long way round.
//
// It reads the kit off each duellist rather than asking WeighCarriers, on
// purpose: a test that decided membership the way the code does would agree with
// it however wrong both were.
func bringers(t *testing.T, lib *Library, skillID string, level int) []string {
	t.Helper()
	brought := []string(nil)
	for _, character := range lib.Characters().All() {
		fielded, err := lib.duellist(character, level)
		if err != nil {
			t.Fatalf("field %s at level %d: %v", character.ID, level, err)
		}
		if slices.Contains(fielded.Skills, skillID) {
			brought = append(brought, character.ID)
		}
	}
	slices.Sort(brought)
	return brought
}

// carriersOf is the ids the table put a row against, in the order it put them.
func carriersOf(report CarriersReport) []string {
	out := make([]string, 0, len(report.Rows))
	for _, row := range report.Rows {
		out = append(out, row.Carrier)
	}
	return out
}

// twinOf saves a second copy of a character under a new id, so a sweep has more
// than one row to keep apart.
//
// The fixture cast has no skill two characters share — nor does the shipped one,
// which is a fact worth knowing and is why this exists. A twin is the smallest
// thing that makes "the other rows are still standing" a statement about
// something.
func twinOf(t *testing.T, lib *Library, id, twin string) {
	t.Helper()
	character, known := lib.Characters().Get(id)
	if !known {
		t.Fatalf("no character is called %q", id)
	}
	character.ID = twin
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save %s as %s: %v", id, twin, err)
	}
}

// TestTheTableHoldsExactlyTheCarriersThatBringTheSkill.
//
// Membership is the first decision this instrument makes and the one that can go
// wrong silently. A character that cannot bring the skill must be *absent*: a row
// of noughts against it would be read as "the field is worth nothing to this
// one", which is the opposite of what a character that never casts the skill
// says — the same distinction Weigh makes when it refuses a row that landed
// nothing rather than pricing it at nought.
func TestTheTableHoldsExactlyTheCarriersThatBringTheSkill(t *testing.T) {
	lib := sparLibrary(t)
	report, err := lib.WeighCarriers(carriersRequest(WeighPower, 1100))
	if err != nil {
		t.Fatalf("weigh carriers: %v", err)
	}
	wanted := bringers(t, lib, weighSkill, progression.LevelCap)
	got := slices.Clone(carriersOf(report))
	slices.Sort(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("the table priced %v, and %v bring %s", got, wanted, weighSkill)
	}

	// And everybody else is in the skipped list rather than nowhere: an absence
	// nothing counts is an absence a reader cannot see.
	skipped := make([]string, 0, len(report.Skipped))
	for _, absent := range report.Skipped {
		skipped = append(skipped, absent.Carrier)
		if slices.Contains(got, absent.Carrier) {
			t.Errorf("%s is both priced and skipped", absent.Carrier)
		}
		if absent.Why == nil {
			t.Errorf("%s was skipped with no reason attached", absent.Carrier)
			continue
		}
		if !strings.Contains(absent.Why.Error(), "does not bring") {
			t.Errorf("%s was skipped for %q, which does not say it cannot bring the skill",
				absent.Carrier, absent.Why)
		}
	}
	if counted := report.Considered(); counted != len(lib.Characters().All()) {
		t.Errorf("the sweep considered %d characters and the book holds %d",
			counted, len(lib.Characters().All()))
	}
	for _, character := range lib.Characters().All() {
		if !slices.Contains(got, character.ID) && !slices.Contains(skipped, character.ID) {
			t.Errorf("%s is neither priced nor skipped, so it vanished", character.ID)
		}
	}
}

// TestOneCarrierWeighedAloneAndInTheTableAgreeRowForRow.
//
// The table is many weighings printed together and nothing else, so a row of it
// has to be the *same measurement* the single-carrier tool takes. Anything else
// would mean the sweep is quietly a second instrument — different seeds,
// different control, different fold — reported under the same name.
func TestOneCarrierWeighedAloneAndInTheTableAgreeRowForRow(t *testing.T) {
	lib := sparLibrary(t)
	alone, err := lib.Weigh(weighRequest(WeighPower, 1100, 1300))
	if err != nil {
		t.Fatalf("weigh %s alone: %v", weighCarrier, err)
	}
	table, err := lib.WeighCarriers(carriersRequest(WeighPower, 1100, 1300))
	if err != nil {
		t.Fatalf("weigh carriers: %v", err)
	}
	var found *CarrierRow
	for i := range table.Rows {
		if table.Rows[i].Carrier == weighCarrier {
			found = &table.Rows[i]
		}
	}
	if found == nil {
		t.Fatalf("the table has no row for %s: %v", weighCarrier, carriersOf(table))
	}
	if found.Err != nil {
		t.Fatalf("the table refused %s, which weighs on its own: %v", weighCarrier, found.Err)
	}
	if !reflect.DeepEqual(found.Report, alone) {
		t.Errorf("the row and the single weighing differ:\n row: %+v\nalone: %+v",
			found.Report.Rows, alone.Rows)
	}
	// The columns the two share have to agree too, or the same figures would be
	// read against two different controls.
	if table.Shipped != alone.Shipped || table.Band != alone.Band || table.Seeds != alone.Seeds {
		t.Errorf("the table declares shipped %d band %d seeds %d and the weighing %d %d %d",
			table.Shipped, table.Band, table.Seeds, alone.Shipped, alone.Band, alone.Seeds)
	}
}

// forkedTwin saves a copy of a character whose evolution line forks, which is a
// character that cannot be fielded with nobody choosing an arm.
//
// It is the refusal a test can *ask for*. Everything a battle refuses a row for
// — nothing landed, a mechanism that never fired, a saturated half — needs a
// carrier the fixture cast does not have, and everything the parser refuses is
// now caught once for the whole sweep. A fork is legal data, refused by
// progression.Furthest by design, and refused for one carrier and not the other.
func forkedTwin(t *testing.T, lib *Library, id, twin string) {
	t.Helper()
	character, known := lib.Characters().Get(id)
	if !known {
		t.Fatalf("no character is called %q", id)
	}
	root := character.Stages[0]
	character.ID = twin
	character.Stages = progression.Line{
		root,
		{Name: "Twin Vine", MinLevel: 30, After: root.Name, Stats: root.Stats},
		{Name: "Twin Fang", MinLevel: 30, After: root.Name, Stats: root.Stats},
	}
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save %s as a forked %s: %v", id, twin, err)
	}
}

// TestARefusedRowLeavesEveryOtherRowStanding.
//
// This is the whole reason the sweep exists rather than a shell loop. The
// workflow it replaces was seven invocations, two of which came back as refusals
// somebody had to notice by eye — so a table that stopped at the first refusal
// would be that workflow with fewer commands and the same blindness.
//
// The refused carrier is deliberately the one that sorts *first*, so a sweep
// that gave up on the first refusal would produce an empty table rather than a
// short one and could not be mistaken for a narrow pass.
func TestARefusedRowLeavesEveryOtherRowStanding(t *testing.T) {
	lib := sparLibrary(t)
	// "fixture-anime.a-fork" sorts before "fixture-anime.adept".
	const forked = "fixture-anime.a-fork"
	forkedTwin(t, lib, weighCarrier, forked)

	report, err := lib.WeighCarriers(carriersRequest(WeighPower, 1100))
	if err != nil {
		t.Fatalf("a sweep with a refused row came back as one error: %v", err)
	}
	wanted := []string{forked, weighCarrier}
	got := slices.Clone(carriersOf(report))
	slices.Sort(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Fatalf("the table holds %v, and %v bring %s", got, wanted, weighSkill)
	}
	rows := map[string]CarrierRow{}
	for _, row := range report.Rows {
		rows[row.Carrier] = row
	}
	if refused := rows[forked]; refused.Priced() {
		t.Errorf("%s priced a field, and it cannot be fielded at all", forked)
	} else if refused.Leaked() {
		t.Errorf("%s reads as a harness refusal, and it is an ordinary one: %v", forked, refused.Err)
	}
	standing := rows[weighCarrier]
	if !standing.Priced() {
		t.Fatalf("the refusal above took %s down with it: %v", weighCarrier, standing.Err)
	}
	if len(standing.Report.Rows) != len(report.Values) {
		t.Errorf("%s priced %d of the %d values swept", weighCarrier,
			len(standing.Report.Rows), len(report.Values))
	}
	// And the surviving row still carries its own exactly-even control, because
	// a row that stands beside a refusal has to be as trustworthy as one that
	// stands alone.
	control, found := standing.At(report.Shipped)
	if !found || control.Rate != scale.Base/2 {
		t.Errorf("%s has no even control beside the refusal: %+v", weighCarrier, standing.Report.Rows)
	}
}

// TestAValueTheParserRefusesRefusesTheWholeSweepOnce.
//
// A value is a fact about the *skill*, not about any carrier: the variant is
// built from the book and every character in the cast would be refused the same
// number for the same reason. Refusing it per row would print one sentence per
// carrier and bury it in its own repetition — and it would print it only after
// the first carrier's battles had been fought for nothing.
func TestAValueTheParserRefusesRefusesTheWholeSweepOnce(t *testing.T) {
	lib := sparLibrary(t)
	if _, err := lib.WeighCarriers(carriersRequest(WeighAccuracy, 5000)); err == nil {
		t.Fatal("an accuracy of 5000 was swept")
	}
	// Nought is the other shape of the same claim: legal to type, refused by the
	// parser because the skill it would author does nothing at all.
	_, err := lib.WeighCarriers(carriersRequest(WeighPower, 0))
	if err == nil {
		t.Fatal("a power of nought was swept")
	}
	if !strings.Contains(err.Error(), weighSkill) {
		t.Errorf("the refusal %q never names the skill whose value it refused", err)
	}
}

// TestAControlThatIsNotEvenRefusesItsRowLoudly.
//
// Every other refusal says *this carrier cannot be priced here*, which is
// ordinary across a whole cast. A control that did not come out exactly even
// says the measurement leaked — a variant in both kits, a side read backwards, a
// perturbed rng — and it is a claim about the run rather than about the carrier.
// On a table the two would otherwise be the same dash in the same column, and
// the loudest thing on the page would be indistinguishable from the dullest.
func TestAControlThatIsNotEvenRefusesItsRowLoudly(t *testing.T) {
	leaked := CarrierRow{Carrier: "a.leak", Err: &UnevenControlError{
		Skill: weighSkill, Field: WeighPower, Value: 1200, Rate: 512,
		Tally: Tally{Wins: 26, Losses: 24},
	}}
	dull := CarrierRow{Carrier: "b.dull", Err: fmt.Errorf(
		"nothing here prices %s at power 0: it was cast 40 time(s) and landed none", weighSkill)}
	wrapped := CarrierRow{Carrier: "c.wrapped", Err: fmt.Errorf("weighing c.wrapped: %w",
		&UnevenControlError{Skill: weighSkill, Field: WeighPower, Rate: 1})}

	for _, test := range []struct {
		row  CarrierRow
		leak bool
	}{{leaked, true}, {dull, false}, {wrapped, true}} {
		if test.row.Priced() {
			t.Errorf("%s reads as priced and carries a refusal", test.row.Carrier)
		}
		if got := test.row.Leaked(); got != test.leak {
			t.Errorf("%s reads leaked=%v, want %v: %v", test.row.Carrier, got, test.leak, test.row.Err)
		}
	}
	// And a priced row is neither, which is the case the two are told apart from.
	priced := CarrierRow{Carrier: "d.fine"}
	if !priced.Priced() || priced.Leaked() {
		t.Errorf("a row with no refusal reads priced=%v leaked=%v", priced.Priced(), priced.Leaked())
	}
	// The refusal has to name the field and the skill, because on a table it is
	// read a long way from the heading that says which two they were.
	for _, wanted := range []string{"control", weighSkill, "power"} {
		if !strings.Contains(leaked.Err.Error(), wanted) {
			t.Errorf("the harness refusal never says %q: %v", wanted, leaked.Err)
		}
	}
}

// TestTheTableIsOrderedByWorthAndNotByWhateverArrivedFirst.
//
// internal/core bans a map iteration that reaches an output, and a report that
// printed itself in whatever order it was handed would be the same fault one
// layer up: the same data would render two ways and a diff between two runs
// would be noise. The order is stated in the footer, so it has to be the order.
func TestTheTableIsOrderedByWorthAndNotByWhateverArrivedFirst(t *testing.T) {
	priced := func(id string, worth int) CarrierRow {
		return CarrierRow{Carrier: id, Report: WeighReport{Rows: []Weighing{
			{Value: 100, Control: true, Rate: scale.Base / 2},
			{Value: 400, Rate: scale.Base/2 + worth},
		}}}
	}
	scrambled := []CarrierRow{
		priced("e.small", 4),
		{Carrier: "d.refused", Err: fmt.Errorf("landed none")},
		priced("b.large", 90),
		{Carrier: "a.leak", Err: &UnevenControlError{Rate: 512}},
		priced("c.large", 90),
	}
	// Two carriers priced the same at the largest value, so the tie-break is the
	// only thing that can order them.
	wanted := []string{"a.leak", "b.large", "c.large", "e.small", "d.refused"}

	for _, order := range [][]int{{0, 1, 2, 3, 4}, {4, 3, 2, 1, 0}, {2, 0, 4, 1, 3}} {
		report := CarriersReport{Values: []int{100, 400}, Shipped: 100}
		for _, at := range order {
			report.Rows = append(report.Rows, scrambled[at])
		}
		report.Skipped = []CarrierSkipped{{Carrier: "z.absent"}, {Carrier: "y.absent"}}
		report.order()
		if got := carriersOf(report); !reflect.DeepEqual(got, wanted) {
			t.Errorf("rows fed in as %v came out %v, want %v", order, got, wanted)
		}
		if report.Skipped[0].Carrier != "y.absent" {
			t.Errorf("the skipped list is in %v rather than by id", report.Skipped)
		}
	}
}

// TestTheSameSweepOrdersTheSameWayTwice is the claim above taken against the
// real thing rather than against a hand-built report, because the fixture is
// what a reader will actually diff two runs of.
func TestTheSameSweepOrdersTheSameWayTwice(t *testing.T) {
	lib := sparLibrary(t)
	twinOf(t, lib, weighCarrier, "fixture-anime.twin")
	first, err := lib.WeighCarriers(carriersRequest(WeighPower, 1100))
	if err != nil {
		t.Fatalf("weigh carriers: %v", err)
	}
	second, err := lib.WeighCarriers(carriersRequest(WeighPower, 1100))
	if err != nil {
		t.Fatalf("weigh carriers again: %v", err)
	}
	if !reflect.DeepEqual(carriersOf(first), carriersOf(second)) {
		t.Errorf("two runs ordered the table %v and %v", carriersOf(first), carriersOf(second))
	}
	if len(first.Rows) < 2 {
		t.Fatalf("the twin did not make a second row: %v", carriersOf(first))
	}
}

// TestTheTableCountsWhatItCostBeforeItIsFought.
//
// A sweep multiplies a weighing by the cast, and the multiplication is the thing
// somebody has to be told before they wait for it rather than after.
func TestTheTableCountsWhatItCostBeforeItIsFought(t *testing.T) {
	report := CarriersReport{
		Seeds: 2000, Values: []int{0, 100, 200, 400},
		Rows: []CarrierRow{{Carrier: "a"}, {Carrier: "b"}, {Carrier: "c"}},
	}
	if got, want := report.Battles(), 2*2000*4*3; got != want {
		t.Errorf("the table counts %d battles, and it is 2 x seeds x values x carriers = %d", got, want)
	}
	if got := report.Largest(); got != 400 {
		t.Errorf("the sort column is %d rather than the largest value swept", got)
	}
	if got := (CarriersReport{}).Largest(); got != 0 {
		t.Errorf("an empty sweep has a largest value of %d", got)
	}
}

// TestASweepRefusesWhatItCannotAnswerAtAll.
//
// The split is the design: what the *request* got wrong is refused whole,
// because there are no rows to put the refusal on, and what a *battle*
// discovered refuses one row. A sweep nobody carries is in the first group —
// a table with no rows in it and a table where the field is worth nothing are the
// same empty page.
func TestASweepRefusesWhatItCannotAnswerAtAll(t *testing.T) {
	lib := sparLibrary(t)
	for _, test := range []struct {
		name    string
		request CarriersRequest
		says    string
	}{
		{"no battles at all", CarriersRequest{Skill: weighSkill, Field: WeighPower, Values: []int{1100},
			Level: progression.LevelCap, Seeds: 0}, "measures nothing"},
		{"a level nobody reaches", CarriersRequest{Skill: weighSkill, Field: WeighPower, Values: []int{1100},
			Level: progression.LevelCap + 1, Seeds: weighSeeds}, "outside"},
		{"a level below one", CarriersRequest{Skill: weighSkill, Field: WeighPower, Values: []int{1100},
			Level: 0, Seeds: weighSeeds}, "outside"},
		{"no skill by that name", CarriersRequest{Skill: "nothing_at_all", Field: WeighPower, Values: []int{1100},
			Level: progression.LevelCap, Seeds: weighSeeds}, "nothing_at_all"},
		{"a skill nobody brings", CarriersRequest{Skill: "solar_beam", Field: WeighPower, Values: []int{1100},
			Level: progression.LevelCap, Seeds: weighSeeds}, "no character brings"},
	} {
		_, err := lib.WeighCarriers(test.request)
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s was refused for %q, which never says %q", test.name, err, test.says)
		}
	}
}

// TestASkillTheCarrierDoesNotBringIsATypeAndNotASentence.
//
// The sweep decides who is in the table by catching this refusal rather than by
// testing the kit itself, so that there is exactly one place that knows what
// "brings" means. A refusal that stopped being this type — reworded, wrapped
// into something else — would turn every carrier into a refused row instead of
// an absent one, and the table would fill up with dashes.
func TestASkillTheCarrierDoesNotBringIsATypeAndNotASentence(t *testing.T) {
	lib := sparLibrary(t)
	request := weighRequest(WeighPower, 1100)
	request.Character = weighCarrier
	request.Skill = "solar_beam"
	_, err := lib.Weigh(request)
	var absent *NotBroughtError
	if !errors.As(err, &absent) {
		t.Fatalf("a skill the carrier does not bring came back as %T: %v", err, err)
	}
	if absent.Carrier != weighCarrier || absent.Skill != "solar_beam" {
		t.Errorf("the refusal names %s and %s", absent.Carrier, absent.Skill)
	}
	if len(absent.Brings) == 0 {
		t.Error("the refusal does not say what the carrier brings instead")
	}
	if !strings.Contains(absent.Error(), "does not bring") {
		t.Errorf("the wording changed: %v", absent)
	}
}
