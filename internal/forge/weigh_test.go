package forge

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// weighSeeds is how many battles a test fights each row over.
//
// Small on purpose, and for a sharper reason than sparSeeds is. The properties
// asserted here are exact — a control row is even to the last part in a
// thousand, two sides of one weighing add to a whole — so a thousand seeds would
// only be a slower way to prove the same equality. The one property that is
// *not* exact, that more power reads above the control, is asserted with a
// change large enough that twenty-five seeds cannot miss it.
const weighSeeds = 25

// weighCarrier and weighSkill are the fixture character and the skill it carries
// that every test here prices. They are named once because a test that
// picked its own would be a test coupled to a kit somebody is free to edit.
const (
	weighCarrier = "fixture-anime.adept"
	weighSkill   = "strike"
)

// weighRequest is the request every test starts from, with the values it wants.
func weighRequest(field WeighField, values ...int) WeighRequest {
	return WeighRequest{
		Character: weighCarrier, Skill: weighSkill, Field: field,
		Values: values, Level: progression.LevelCap, Seeds: weighSeeds,
	}
}

// controlRow is the row a report took as its control.
func controlRow(t *testing.T, report WeighReport) Weighing {
	t.Helper()
	for _, row := range report.Rows {
		if row.Control {
			return row
		}
	}
	t.Fatalf("the report has no control row at all: %+v", report.Rows)
	return Weighing{}
}

// TestTheControlRowIsExactlyEven is the claim the whole instrument rests on.
//
// Both sides are the same character with the same stats, the same kit and the
// same placement, and the challenger's copy of one skill differs from the
// opponent's only in its id. There is nothing left for either side to win by, so
// anything other than an even split is the harness rather than the skill — and
// every figure taken beside it would be that same error plus a number.
func TestTheControlRowIsExactlyEven(t *testing.T) {
	lib := sparLibrary(t)
	report, err := lib.Weigh(weighRequest(WeighPower))
	if err != nil {
		t.Fatalf("weigh: %v", err)
	}
	control := controlRow(t, report)
	if control.Rate != scale.Base/2 {
		t.Errorf("the control came to %d rather than an even %d: %+v",
			control.Rate, scale.Base/2, control.Tally)
	}
	if control.Worth() != 0 {
		t.Errorf("the control is worth %d, and a skill against itself is worth nothing",
			control.Worth())
	}
	// The stronger statement, and the one that says *why* it is even: the halves
	// are the same battles with the kits swapped, so every win from one slot is a
	// loss from the other. A rate even by two errors cancelling passes above and
	// fails here.
	if control.Edge == 0 {
		t.Error("the first slot is worth nothing in this pairing, so the two halves are " +
			"cancelling nothing and an even control proves only that zero plus zero is zero")
	}
}

// TestAWeighingRefusesAReportWhoseControlIsNotEven is the refusal on its own,
// away from any battle.
//
// It is a check made on every run rather than a test made once, because it is a
// claim about the battles that were actually fought: a leak of the variant into
// both kits, a side read the wrong way round, a perturbed rng — each produces
// rows that look exactly like good rows and a control that is not even.
func TestAWeighingRefusesAReportWhoseControlIsNotEven(t *testing.T) {
	even := Weighing{Value: 1000, Control: true, Rate: scale.Base / 2,
		Tally: Tally{Wins: 10, Losses: 10}}
	if err := refuseUnevenControl(even, WeighPower, weighSkill); err != nil {
		t.Errorf("an even control was refused: %v", err)
	}
	for _, rate := range []int{0, 499, 501, scale.Base} {
		crooked := even
		crooked.Rate = rate
		err := refuseUnevenControl(crooked, WeighPower, weighSkill)
		if err == nil {
			t.Errorf("a control reading %d was accepted", rate)
			continue
		}
		for _, wanted := range []string{"control", weighSkill, "power"} {
			if !strings.Contains(err.Error(), wanted) {
				t.Errorf("the refusal of a control reading %d never says %q: %v", rate, wanted, err)
			}
		}
	}
}

// TestASynthesisedVariantDiffersInExactlyOneField.
//
// The claim a weighing makes is that the two sides differ in one number. This is
// that claim about the variant itself, asserted exhaustively rather than field by
// field: put the id and the one field back, and what comes out has to be the
// shipped skill in every other respect. A copy that quietly dropped a
// restriction, an element or an application would price a different skill and
// nothing on screen would say so.
func TestASynthesisedVariantDiffersInExactlyOneField(t *testing.T) {
	lib := sparLibrary(t)
	for _, test := range []struct {
		field WeighField
		value int
	}{
		{WeighPower, 1234},
		{WeighAccuracy, 700},
		{WeighCrit, 250},
		{WeighCooldown, 3},
	} {
		shipped, err := lib.Skills().Lookup(weighSkill)
		if err != nil {
			t.Fatalf("look %s up: %v", weighSkill, err)
		}
		_, book, err := lib.variantOf(shipped, test.field, test.value)
		if err != nil {
			t.Fatalf("synthesise %s at %s %d: %v", weighSkill, test.field, test.value, err)
		}
		// Read it back out of the book rather than trusting the struct handed in:
		// the variant only counts once it has been through the parser, which is
		// what the battle will read.
		built, err := book.Lookup(variantID(weighSkill, test.field, test.value))
		if err != nil {
			t.Fatalf("the variant is not in the book it came back in: %v", err)
		}
		if got := test.field.of(built); got != test.value {
			t.Errorf("%s was set to %d and reads %d", test.field, test.value, got)
		}
		restored := test.field.set(built, test.field.of(shipped))
		restored.ID = shipped.ID
		if !reflect.DeepEqual(restored, shipped) {
			t.Errorf("moving %s changed something else as well:\n variant %+v\n shipped %+v",
				test.field, restored, shipped)
		}
	}
}

// TestMoreOfAGoodFieldReadsAboveTheControl is the machinery cross-checked on a
// field whose answer is known before the battle is fought.
//
// It is power rather than crit deliberately. Crit is the field this instrument
// was built for, so measuring only crit would leave the whole apparatus checked
// against the one thing it is meant to discover — and a harness that reported
// every change as an improvement would pass. Power has an answer nobody needs an
// instrument for: more of it wins more often, and it kills sooner.
func TestMoreOfAGoodFieldReadsAboveTheControl(t *testing.T) {
	lib := sparLibrary(t)
	const stronger = 1300
	report, err := lib.Weigh(weighRequest(WeighPower, stronger))
	if err != nil {
		t.Fatalf("weigh: %v", err)
	}
	control := controlRow(t, report)
	if control.Value >= stronger {
		t.Fatalf("the control declares %d, which is not below the swept %d, so this test proves nothing",
			control.Value, stronger)
	}
	raised := report.Rows[len(report.Rows)-1]
	if raised.Value != stronger {
		t.Fatalf("the last row is %d rather than the swept %d", raised.Value, stronger)
	}
	if raised.Worth() <= report.Band {
		t.Errorf("raising power from %d to %d is worth %d, inside the band of %d — "+
			"more damage has to read as more",
			control.Value, stronger, raised.Worth(), report.Band)
	}
	if raised.Turns > control.Turns {
		t.Errorf("raising power from %d to %d took %d turns against the control's %d — "+
			"more damage has to kill sooner",
			control.Value, stronger, raised.Turns, control.Turns)
	}
}

// TestWeighingTheOpponentIsTheNegativeOfWeighingTheChallenger is the only check
// here that catches which side is being read.
//
// Everything else in this file would pass with the two sides swapped: the
// control is even either way, a band is a band either way, and a refusal refuses
// either way. Put the variant on the other unit and the same battles have to
// come back as the complement — so a harness reading the opponent's wins under
// the challenger's name reports every price with its sign reversed and this is
// what says so.
func TestWeighingTheOpponentIsTheNegativeOfWeighingTheChallenger(t *testing.T) {
	lib := sparLibrary(t)
	carrier, err := lib.duellist(mustCharacter(t, lib, weighCarrier), progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("field %s: %v", weighCarrier, err)
	}
	shipped, err := lib.Skills().Lookup(weighSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", weighSkill, err)
	}
	const value = 1300
	variant, book, err := lib.variantOf(shipped, WeighPower, value)
	if err != nil {
		t.Fatalf("synthesise: %v", err)
	}
	books := lib.Books()
	books.Skills = book

	carrying := carrier
	carrying.Skills = append([]string(nil), carrier.Skills...)
	for i, held := range carrying.Skills {
		if held == weighSkill {
			carrying.Skills[i] = variant.ID
		}
	}

	mine, err := duel(books, carrying, carrier, weighSeeds, false, variant.ID)
	if err != nil {
		t.Fatalf("weigh the challenger: %v", err)
	}
	theirs, err := duel(books, carrier, carrying, weighSeeds, false, variant.ID)
	if err != nil {
		t.Fatalf("weigh the opponent: %v", err)
	}
	if mine.Rate()+theirs.Rate() != scale.Base {
		t.Errorf("the variant reads %d on the challenger and %d on the opponent, which comes to %d rather than %d",
			mine.Rate(), theirs.Rate(), mine.Rate()+theirs.Rate(), scale.Base)
	}
	// And the strikes follow the kit rather than the slot. Counting the variant
	// while the *opponent* holds it has to come back empty: an event carries an
	// Actor and no Side at all, so a fold reading the wrong id would report the
	// other unit's attacks here and every other assertion would still pass.
	if mine.Strikes.Landed == 0 {
		t.Error("the challenger holds the variant and landed none of it")
	}
	if theirs.Strikes.Landed != 0 {
		t.Errorf("the opponent holds the variant and %d of its strikes were filed under the challenger",
			theirs.Strikes.Landed)
	}
}

// TestTheShippedBookIsUnchangedByAWeighing.
//
// A weighing invents a skill, and the whole reason it may is that the skill is
// never written down. If the book on disk moved, an author would be measuring
// against a book that no longer matches the file they are editing — and a golden
// would move for a measurement, which is the one thing the data files must never
// do.
func TestTheShippedBookIsUnchangedByAWeighing(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	before, err := lib.Skills().Marshal()
	if err != nil {
		t.Fatalf("marshal the book: %v", err)
	}
	directory := treeDigest(t, dir)

	if _, err := lib.Weigh(weighRequest(WeighPower, 1100)); err != nil {
		t.Fatalf("weigh: %v", err)
	}

	after, err := lib.Skills().Marshal()
	if err != nil {
		t.Fatalf("marshal the book again: %v", err)
	}
	if string(before) != string(after) {
		t.Error("the library's own skill book changed during a weighing, so the variant leaked out of it")
	}
	if again := treeDigest(t, dir); !reflect.DeepEqual(directory, again) {
		t.Errorf("the data directory moved during a weighing: was %v, now %v", directory, again)
	}
	// And the variant is not in it under any name.
	if _, found := lib.Skills().Lookup(variantID(weighSkill, WeighPower, 1100)); found == nil {
		t.Error("the variant is in the library's book after the weighing")
	}
}

// TestAWeighingRefusesASkillTheCarrierDoesNotBring.
//
// It would otherwise be measured, which is worse than refused: the variant would
// sit in the book, nobody would cast it, and the row would come back an even
// split — a price of nought printed against a skill that was never in the fight.
func TestAWeighingRefusesASkillTheCarrierDoesNotBring(t *testing.T) {
	lib := sparLibrary(t)
	carrier, err := lib.duellist(mustCharacter(t, lib, weighCarrier), progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("field %s: %v", weighCarrier, err)
	}
	absent := ""
	for _, declared := range lib.Skills().Skills() {
		if !containsString(carrier.Skills, declared.ID) {
			absent = declared.ID
			break
		}
	}
	if absent == "" {
		t.Skip("the carrier brings every skill in the book, so there is nothing it does not bring")
	}
	request := weighRequest(WeighPower, 1100)
	request.Skill = absent
	_, err = lib.Weigh(request)
	if err == nil {
		t.Fatalf("%s was measured on %s, which does not bring it", absent, weighCarrier)
	}
	if !strings.Contains(err.Error(), "does not bring") {
		t.Errorf("the refusal reads %q, which does not say the carrier does not bring it", err)
	}
}

// TestAValueTheParserRefusesComesBackInTheParsersOwnWords.
//
// Every bound a weighable field has is already enforced by skill.resolve, which
// is what the game boots through. A bound restated in this package would be a
// second copy of a rule free to disagree with the first — so the value goes to
// the parser and the parser's sentence comes back whole, unwrapped and
// unreworded. It is also the sentence the author has already seen from `hexforge
// skills edit`, which is worth more than a tidier one.
func TestAValueTheParserRefusesComesBackInTheParsersOwnWords(t *testing.T) {
	lib := sparLibrary(t)
	shipped, err := lib.Skills().Lookup(weighSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", weighSkill, err)
	}
	for _, test := range []struct {
		field WeighField
		value int
		says  string
	}{
		{WeighAccuracy, 5000, "want a share in parts per thousand"},
		{WeighCrit, -1, "want a share in parts per thousand"},
		{WeighPower, -10, "want zero or more"},
		{WeighCooldown, -1, "want zero or more"},
		{WeighRange, 99, "want between 1 and"},
	} {
		_, _, err := lib.variantOf(shipped, test.field, test.value)
		if err == nil {
			t.Errorf("%s %d was accepted", test.field, test.value)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s %d was refused with %q, which is not the parser's own %q",
				test.field, test.value, err, test.says)
		}
		// Unwrapped: no lead-in of this package's invention in front of it.
		if strings.Contains(err.Error(), "cannot be measured") {
			t.Errorf("%s %d came back reworded: %q", test.field, test.value, err)
		}
	}
}

// TestARowThatNeverLandedTheSkillIsRefusedRatherThanReportedAsEven.
//
// Worth nothing means *not rated*, never rated at nought, and the two are the
// same glyph in a column. A skill that was never cast — or cast and never landed
// — leaves both sides identical in every respect that matters, so the row comes
// back an even split and reads as "this field is worth nothing" when it means
// "nothing here measured this field".
func TestARowThatNeverLandedTheSkillIsRefusedRatherThanReportedAsEven(t *testing.T) {
	request := weighRequest(WeighPower, 1100)
	fought := Matchup{First: Tally{Wins: 10, Losses: 10}, Second: Tally{Wins: 10, Losses: 10}}
	for _, test := range []struct {
		name    string
		strikes Strikes
	}{
		{"never cast at all", Strikes{}},
		{"cast and never landed", Strikes{Cast: 40}},
	} {
		row := Weighing{Value: 1100, Rate: scale.Base / 2,
			Tally: Tally{Wins: 20, Losses: 20}, Strikes: test.strikes}
		err := refuseUnreadable(row, request, fought)
		if err == nil {
			t.Errorf("%s was reported as an even row", test.name)
			continue
		}
		if !strings.Contains(err.Error(), "nothing here prices") {
			t.Errorf("%s was refused with %q, which does not say nothing prices it", test.name, err)
		}
	}

	// And the mechanism with a mark of its own: a crit chance that never once
	// came up prices the noise, not the chance.
	crit := weighRequest(WeighCrit, 200)
	landed := Weighing{Value: 200, Rate: scale.Base / 2,
		Tally: Tally{Wins: 20, Losses: 20}, Strikes: Strikes{Cast: 40, Landed: 38}}
	if err := refuseUnreadable(landed, crit, fought); err == nil {
		t.Error("a crit chance that never fired was reported as a figure")
	}
	landed.Strikes.Critical = 1
	if err := refuseUnreadable(landed, crit, fought); err != nil {
		t.Errorf("a crit chance that fired was refused: %v", err)
	}
	// A crit of nought is the control, and has nothing to fire.
	zero := weighRequest(WeighCrit, 0)
	none := Weighing{Value: 0, Rate: scale.Base / 2,
		Tally: Tally{Wins: 20, Losses: 20}, Strikes: Strikes{Cast: 40, Landed: 38}}
	if err := refuseUnreadable(none, zero, fought); err != nil {
		t.Errorf("a control row with no crit to fire was refused: %v", err)
	}
}

// TestARowOfEndlessDuelsIsRefusedRatherThanCounted.
//
// A rate is taken over the battles that decided, so a row where most of them did
// not is a measurement of the minority that did — and whether a pairing resolves
// at all is exactly the thing a change to a damage number moves.
func TestARowOfEndlessDuelsIsRefusedRatherThanCounted(t *testing.T) {
	request := weighRequest(WeighPower, 1100)
	fought := Matchup{First: Tally{Wins: 10, Losses: 10}, Second: Tally{Wins: 10, Losses: 10}}
	for _, test := range []struct {
		name    string
		tally   Tally
		refused bool
	}{
		{"nothing endless", Tally{Wins: 20, Losses: 20}, false},
		{"a fifth exactly", Tally{Wins: 20, Losses: 20, Endless: 10}, false},
		{"more than a fifth", Tally{Wins: 20, Losses: 20, Endless: 11}, true},
		{"almost all of it", Tally{Wins: 2, Losses: 2, Endless: 96}, true},
	} {
		row := Weighing{Value: 1100, Rate: scale.Base / 2, Tally: test.tally,
			Strikes: Strikes{Cast: 40, Landed: 38}}
		err := refuseUnreadable(row, request, fought)
		if test.refused && err == nil {
			t.Errorf("%s was counted", test.name)
		}
		if !test.refused && err != nil {
			t.Errorf("%s was refused: %v", test.name, err)
		}
	}
}

// TestASaturatedHalfIsRefusedForHavingNoRoom.
//
// A half that already wins everything cannot show that the field was made
// larger, so the row prices the ceiling rather than the value. The refusal is on
// a *half* rather than on the total because the total of a saturated row is not
// extreme — one slot at the ceiling and the other short of it averages to
// something that reads like an ordinary strong figure.
func TestASaturatedHalfIsRefusedForHavingNoRoom(t *testing.T) {
	request := weighRequest(WeighPower, 1100)
	row := Weighing{Value: 1100, Rate: 900, Tally: Tally{Wins: 36, Losses: 4},
		Strikes: Strikes{Cast: 40, Landed: 38}}
	for _, test := range []struct {
		name    string
		fought  Matchup
		refused bool
	}{
		{"room in both halves",
			Matchup{First: Tally{Wins: 19, Losses: 1}, Second: Tally{Wins: 17, Losses: 3}}, false},
		{"the first slot at the ceiling",
			Matchup{First: Tally{Wins: 1000}, Second: Tally{Wins: 17, Losses: 3}}, true},
		{"the second slot on the floor",
			Matchup{First: Tally{Wins: 10, Losses: 10}, Second: Tally{Losses: 1000}}, true},
	} {
		err := refuseUnreadable(row, request, test.fought)
		if test.refused && err == nil {
			t.Errorf("%s was priced", test.name)
		}
		if !test.refused && err != nil {
			t.Errorf("%s was refused: %v", test.name, err)
		}
		if test.refused && err != nil && !strings.Contains(err.Error(), "saturated") {
			t.Errorf("%s was refused with %q, which does not say it is saturated", test.name, err)
		}
	}
}

// TestTheBandNarrowsAsTheSquareRootOfTheSeeds, which is what makes the default
// seed count an arithmetic answer rather than a taste.
//
// Four times the battles halve the band. If it narrowed faster the band would be
// a promise the measurement cannot keep, and if it narrowed slower nobody would
// ever be able to afford a figure.
func TestTheBandNarrowsAsTheSquareRootOfTheSeeds(t *testing.T) {
	for _, seeds := range []int{25, 100, 250, 1000, 2500, 10000} {
		wide, narrow := band(seeds), band(4*seeds)
		if narrow < 1 {
			t.Fatalf("the band over %d seeds is %d, and a band of nothing makes every wobble a finding",
				4*seeds, narrow)
		}
		// Rounded up at both ends, so the halving is exact to within one part.
		if gap := wide - 2*narrow; gap < -2 || gap > 2 {
			t.Errorf("the band over %d seeds is %d and over %d it is %d, which is not half",
				seeds, wide, 4*seeds, narrow)
		}
	}
	// A band is never zero, however many battles are fought.
	for _, seeds := range []int{1, 1000000, 1 << 30} {
		if got := band(seeds); got < 1 {
			t.Errorf("the band over %d seeds is %d", seeds, got)
		}
	}
	// The default is the figure the doc comment claims: two sigma at eight parts
	// per thousand, which the effects this was built for are two to three times.
	if got := band(10000); got != 8 {
		t.Errorf("the band over the default ten thousand seeds is %d rather than 8", got)
	}
	// And the square root underneath it is an integer one, because no float may
	// reach a figure this repository prints.
	for _, test := range []struct{ value, root int }{
		{0, 0}, {1, 1}, {2, 1}, {3, 1}, {4, 2}, {8, 2}, {9, 3},
		{20000, 141}, {1000000, 1000},
	} {
		if got := isqrt(test.value); got != test.root {
			t.Errorf("isqrt(%d) is %d, wanted %d", test.value, got, test.root)
		}
	}
}

// TestAWeighingIsRepeatable is the engine's determinism seen from the
// instrument: a price an author writes down stays true until the data moves.
func TestAWeighingIsRepeatable(t *testing.T) {
	lib := sparLibrary(t)
	request := weighRequest(WeighPower, 1100)
	first, err := lib.Weigh(request)
	if err != nil {
		t.Fatalf("weigh: %v", err)
	}
	again, err := lib.Weigh(request)
	if err != nil {
		t.Fatalf("weigh a second time: %v", err)
	}
	if !reflect.DeepEqual(first.Rows, again.Rows) {
		t.Errorf("two runs disagree:\n %+v\n %+v", first.Rows, again.Rows)
	}
	if first.Band != again.Band {
		t.Errorf("the band moved between runs: %d then %d", first.Band, again.Band)
	}
}

// TestTheSweepSaysWhetherItIsMonotone.
//
// A dial that is not monotone is not priced, whatever the figures beside it say:
// if more of a thing is sometimes worth less, the number against any one value
// is not that value's worth. This is the property the roster win rate failed —
// more ally damage lowered it — so it is reported on every sweep rather than
// left for a reader to notice.
//
// A step inside the band counts as no step, and that is not leniency. Every
// figure here is a measurement, so a genuinely ordered curve wobbles by less
// than its own band somewhere along it; demanding an exactly ordered series
// would report every real curve as unordered.
func TestTheSweepSaysWhetherItIsMonotone(t *testing.T) {
	for _, test := range []struct {
		name      string
		series    []int
		tolerance int
		want      bool
	}{
		{"rising", []int{-100, 0, 100, 300}, 8, true},
		{"falling", []int{300, 100, 0, -100}, 8, true},
		{"flat", []int{0, 0, 0}, 8, true},
		{"one row", []int{0}, 8, true},
		{"nothing at all", nil, 8, true},
		{"up then down, well outside the band", []int{0, 200, 100}, 8, false},
		{"down then up, well outside the band", []int{0, -200, -100}, 8, false},
		{"a wobble smaller than the band is not a turn", []int{0, 5, 2, 300}, 8, true},
		{"the same wobble with no band at all is one", []int{0, 5, 2, 300}, 0, false},
	} {
		if got := monotone(test.series, test.tolerance); got != test.want {
			t.Errorf("%s: monotone(%v, %d) is %v, wanted %v",
				test.name, test.series, test.tolerance, got, test.want)
		}
	}

	// And the report reads both columns, separately, because they can disagree
	// and the disagreement is the finding.
	report := WeighReport{Band: 8, Rows: []Weighing{
		{Value: 900, Rate: 400, Turns: 40},
		{Value: 1000, Rate: 500, Turns: 37},
		{Value: 1100, Rate: 600, Turns: 33},
	}}
	if !report.MonotoneWorth() || !report.MonotoneTurns() {
		t.Errorf("an ordered sweep reads worth=%v turns=%v",
			report.MonotoneWorth(), report.MonotoneTurns())
	}
	// The case this instrument exists for: worth flat inside the band while the
	// turns move. Worth is monotone (every step is noise) and turns are too, and
	// a reader who read only the first column would call the row nothing.
	quiet := WeighReport{Band: 8, Rows: []Weighing{
		{Value: 900, Rate: 502, Turns: 40},
		{Value: 1000, Rate: 500, Turns: 37},
		{Value: 1100, Rate: 503, Turns: 33},
	}}
	if !quiet.MonotoneWorth() {
		t.Error("a sweep whose worth only wobbles inside its band reads as unordered")
	}
	if !quiet.MonotoneTurns() {
		t.Error("a sweep whose turns fall throughout reads as unordered")
	}
	// And a real reversal is still a reversal.
	broken := WeighReport{Band: 8, Rows: []Weighing{
		{Value: 900, Rate: 400, Turns: 40},
		{Value: 1000, Rate: 600, Turns: 30},
		{Value: 1100, Rate: 450, Turns: 44},
	}}
	if broken.MonotoneWorth() || broken.MonotoneTurns() {
		t.Errorf("a reversed sweep reads worth=%v turns=%v",
			broken.MonotoneWorth(), broken.MonotoneTurns())
	}
}

// TestAVariantIdCannotCollideWithAShippedOne.
//
// A collision is the one way the variant stops being a variant: the challenger
// and the opponent would share the skill after all, or a real authored skill
// would quietly change under a name somebody chose. Neither can be seen in a
// column of figures, so it is refused at the name.
func TestAVariantIdCannotCollideWithAShippedOne(t *testing.T) {
	lib := sparLibrary(t)
	shipped, err := lib.Skills().Lookup(weighSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", weighSkill, err)
	}
	collider := shipped
	collider.ID = variantID(weighSkill, WeighPower, 1100)
	grown, err := lib.skills.Append(lib.SkillDeps(), collider)
	if err != nil {
		t.Fatalf("plant a colliding skill: %v", err)
	}
	lib.skills = grown

	if _, _, err := lib.variantOf(shipped, WeighPower, 1100); err == nil {
		t.Fatal("a variant was synthesised under a name the book already declares")
	} else if !strings.Contains(err.Error(), "already declares") {
		t.Errorf("the refusal reads %q, which does not say the name is taken", err)
	}
	// And the whole report refuses rather than fighting the rows that would have
	// worked.
	if _, err := lib.Weigh(weighRequest(WeighPower, 1100)); err == nil {
		t.Error("a sweep with a colliding row was reported")
	}
}

// TestTheControlIsTheSkillsOwnValueAndNotZero.
//
// The control has to be what the book declares, not the smallest value swept and
// not nought: a sweep read against zero would price the whole skill rather than
// the change, and a sweep read against its own lowest row would move its answer
// every time somebody added a row.
func TestTheControlIsTheSkillsOwnValueAndNotZero(t *testing.T) {
	lib := sparLibrary(t)
	shipped, err := lib.Skills().Lookup(weighSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", weighSkill, err)
	}
	declared := WeighPower.of(shipped)
	if declared == 0 {
		t.Fatalf("%s declares no power, so this test cannot tell a control from a zero", weighSkill)
	}
	report, err := lib.Weigh(weighRequest(WeighPower, 1100, 1200))
	if err != nil {
		t.Fatalf("weigh: %v", err)
	}
	if report.Shipped != declared {
		t.Errorf("the report calls %d the shipped value and the book declares %d",
			report.Shipped, declared)
	}
	control := controlRow(t, report)
	if control.Value != declared {
		t.Errorf("the control row is %d rather than the declared %d", control.Value, declared)
	}
	// Inserted even though nobody asked for it, deduped, and in order.
	if len(report.Rows) != 3 {
		t.Fatalf("a sweep of two values plus a control came to %d rows", len(report.Rows))
	}
	for i := 1; i < len(report.Rows); i++ {
		if report.Rows[i].Value <= report.Rows[i-1].Value {
			t.Errorf("the rows are not ascending: %d then %d",
				report.Rows[i-1].Value, report.Rows[i].Value)
		}
	}
	controls := 0
	for _, row := range report.Rows {
		if row.Control {
			controls++
		}
	}
	if controls != 1 {
		t.Errorf("the sweep has %d control rows", controls)
	}
}

// TestASweepAlwaysCarriesItsControlExactlyOnce is the list arithmetic on its
// own: a caller who names the control, names it twice, or names nothing at all
// gets the same one row.
func TestASweepAlwaysCarriesItsControlExactlyOnce(t *testing.T) {
	for _, test := range []struct {
		values  []int
		control int
		want    []int
	}{
		{nil, 600, []int{600}},
		{[]int{600}, 600, []int{600}},
		{[]int{600, 600}, 600, []int{600}},
		{[]int{700, 500}, 600, []int{500, 600, 700}},
		{[]int{700, 700, 500}, 600, []int{500, 600, 700}},
	} {
		got := sweep(test.values, test.control)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("sweep(%v, %d) is %v, wanted %v", test.values, test.control, got, test.want)
		}
	}
}

// TestEveryWeighableFieldIsOneBoundedNumber is the closed table asserted as a
// table: every member names itself, parses back, and reads a value off a skill.
func TestEveryWeighableFieldIsOneBoundedNumber(t *testing.T) {
	lib := sparLibrary(t)
	shipped, err := lib.Skills().Lookup(weighSkill)
	if err != nil {
		t.Fatalf("look %s up: %v", weighSkill, err)
	}
	names := FieldNames()
	if len(names) != weighFieldCount {
		t.Fatalf("%d names for %d fields", len(names), weighFieldCount)
	}
	for i := 1; i < len(names); i++ {
		if names[i] <= names[i-1] {
			t.Errorf("the field names are not sorted: %q then %q", names[i-1], names[i])
		}
	}
	for _, field := range WeighFields() {
		parsed, err := ParseWeighField(field.String())
		if err != nil {
			t.Errorf("%s does not parse back: %v", field, err)
			continue
		}
		if parsed != field {
			t.Errorf("%s parsed back as %s", field, parsed)
		}
		// set and of are each other's inverse, and set touches nothing else.
		moved := field.set(shipped, 7)
		if got := field.of(moved); got != 7 {
			t.Errorf("%s set to 7 reads %d", field, got)
		}
		restored := field.set(moved, field.of(shipped))
		if !reflect.DeepEqual(restored, shipped) {
			t.Errorf("%s does not put back what it took", field)
		}
	}
	if _, err := ParseWeighField("self_gradient"); err == nil {
		t.Error("self_gradient is weighable, and it is two numbers rather than one")
	}
	if _, err := ParseWeighField("applies"); err == nil {
		t.Error("applies is weighable, and changing it authors a different skill")
	}
}

// TestAWeighingRefusesWhatItCannotMeasure. Each of these would otherwise produce
// a report that looked like every other report and meant nothing.
func TestAWeighingRefusesWhatItCannotMeasure(t *testing.T) {
	lib := sparLibrary(t)
	for _, test := range []struct {
		name   string
		mutate func(*WeighRequest)
		says   string
	}{
		{"no battles at all", func(r *WeighRequest) { r.Seeds = 0 }, "measures nothing"},
		{"a negative count", func(r *WeighRequest) { r.Seeds = -1 }, "measures nothing"},
		{"below the first level", func(r *WeighRequest) { r.Level = 0 }, "outside"},
		{"past the cap", func(r *WeighRequest) { r.Level = progression.LevelCap + 1 }, "outside"},
		{"nobody by that name", func(r *WeighRequest) { r.Character = "nobody.at.all" }, "no character"},
		{"no skill by that name", func(r *WeighRequest) { r.Skill = "nothing_at_all" }, "unknown skill"},
	} {
		request := weighRequest(WeighPower, 1100)
		test.mutate(&request)
		_, err := lib.Weigh(request)
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s said %q, which does not mention %q", test.name, err, test.says)
		}
	}
}

// mustCharacter looks a character up or gives up.
func mustCharacter(t *testing.T, lib *Library, id string) cast.Character {
	t.Helper()
	character, known := lib.Characters().Get(id)
	if !known {
		t.Fatalf("no character is called %q", id)
	}
	return character
}

func containsString(list []string, wanted string) bool {
	for _, held := range list {
		if held == wanted {
			return true
		}
	}
	return false
}

// treeDigest is every file in a directory by content, which is how a test says
// "nothing here moved" without naming the files.
func treeDigest(t *testing.T, dir string) map[string]string {
	t.Helper()
	digests := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		digests[relative] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return digests
}
