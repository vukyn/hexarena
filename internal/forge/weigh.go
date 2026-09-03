package forge

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// WeighField is the one number a weighing moves.
//
// It is not called Field because this package already has one: forge.Field
// names which authored answer internal/core/cast refused, and the two have
// nothing to do with each other. The longer name is the cost of keeping both
// legible.
//
// It is a closed table rather than a name looked up by reflection, and what is
// *not* in it is the design rather than an omission.
//
// Every member is one bounded integer whose bound skill.resolve already
// enforces, so a value out of range comes back in the parser's own words and
// this package never restates a rule the parser holds.
//
// ⚠️ `self_gradient` is in the table, and it is the one member whose *off* state
// is not a number. skill.Gradient has exactly one field, AtEmpty — its own doc
// says why there is no second one: the top of the curve is not a choice — so a
// sweep of it is one line rather than a surface, and MonotoneWorth keeps exactly
// the meaning it has on every other field here: one field, one sweep, one answer
// to whether more of this is sometimes worth less. What is different is the
// bottom of that line. resolveGradient refuses AtEmpty below one, so a skill
// that declares no gradient carries a nil pointer where every other field is off
// at a nought a crit is legal at. `of` reads that absence as nought — nil-safe
// the way Gradient.Share is — and `set` hands the nought straight back to the
// parser rather than translating it.
//
// ⚠️ The honest consequence, stated here rather than worked around: this field
// prices **how much** a gradient is worth and never **whether to have one**. A
// sweep may not contain the control row of a skill that declares none, because
// that control is a nought and the parser refuses it — so no report here has a
// row for "this skill without a gradient at all", and none can be read as one.
// Mapping the nought back to nil inside `set` would buy that row and would be
// this package holding a second copy of a bound skill.resolve owns, which is the
// one thing `set` exists not to do. See TODO.md.
//
// ⚠️ Nothing shipped carries it either: `comeback` is the only skill in the book
// that declares a gradient and no character fields it, so `--carriers all` on it
// is an empty table of skips. That is the state of the data rather than a fault
// in the field, and the fix is a character bringing it, not a change here.
//
// Every non-scalar is left out for a different reason: `applies`, `strips`,
// `summons`, `restriction`, `element`, `target` and `pattern` are not dials.
// Changing one authors a *different skill*, so the deviation measured against
// the control would be the worth of that other skill rather than the price of a
// value, and the whole claim a weighing makes is that the two sides differ in
// one number.
type WeighField uint8

const (
	// WeighAccuracy through WeighStrikes are declared in the order their names
	// sort, so WeighFields is declaration order and the two cannot drift apart.
	WeighAccuracy WeighField = iota
	WeighCooldown
	WeighCrit
	WeighDrains
	WeighPierce
	WeighPower
	WeighRange
	WeighSelfGradient
	WeighStrikes
)

// weighFieldCount is how many fields can be weighed.
const weighFieldCount = int(WeighStrikes) + 1

var weighFieldNames = [weighFieldCount]string{
	WeighAccuracy:     "accuracy",
	WeighCooldown:     "cooldown",
	WeighCrit:         "crit",
	WeighDrains:       "drains",
	WeighPierce:       "pierce",
	WeighPower:        "power",
	WeighRange:        "range",
	WeighSelfGradient: "self_gradient",
	WeighStrikes:      "strikes",
}

// String is the field's name as it is typed and printed.
func (f WeighField) String() string {
	if int(f) >= weighFieldCount {
		return fmt.Sprintf("field(%d)", uint8(f))
	}
	return weighFieldNames[f]
}

// WeighFields is every weighable field, sorted by name.
func WeighFields() []WeighField {
	all := make([]WeighField, 0, weighFieldCount)
	for field := 0; field < weighFieldCount; field++ {
		all = append(all, WeighField(field))
	}
	slices.SortFunc(all, func(a, b WeighField) int { return strings.Compare(a.String(), b.String()) })
	return all
}

// FieldNames is every weighable field's name, sorted, which is what a flag's
// help line and a refusal both want.
func FieldNames() []string {
	names := make([]string, 0, weighFieldCount)
	for _, field := range WeighFields() {
		names = append(names, field.String())
	}
	return names
}

// ParseWeighField resolves a field by name.
func ParseWeighField(name string) (WeighField, error) {
	for _, field := range WeighFields() {
		if field.String() == name {
			return field, nil
		}
	}
	return 0, fmt.Errorf("no field is called %q; there are %s",
		name, strings.Join(FieldNames(), ", "))
}

// of is the value the shipped skill declares for this field, which is the
// weighing's control.
func (f WeighField) of(declared skill.Skill) int {
	switch f {
	case WeighAccuracy:
		return declared.Accuracy
	case WeighCooldown:
		return declared.Cooldown
	case WeighCrit:
		return declared.Crit
	case WeighDrains:
		return declared.Drains
	case WeighPierce:
		return declared.Pierce
	case WeighPower:
		return declared.Power
	case WeighRange:
		return declared.Range
	case WeighSelfGradient:
		// Nil-safe for the reason skill.Gradient.Share is: what a skill without
		// a gradient adds is nought, and that is an answer rather than a state
		// worth branching on at every call site. It is also the nought a sweep
		// of such a skill takes as its control row and is refused for — see the
		// type's doc.
		if declared.SelfGradient == nil {
			return 0
		}
		return declared.SelfGradient.AtEmpty
	default:
		return declared.Strikes
	}
}

// set is the one place in this package where a field's name becomes a struct
// field, and it checks nothing.
//
// Checking nothing is the point rather than an oversight. Every bound worth
// enforcing is already enforced by skill.resolve, which the variant goes through
// on its way back into a book — so a bound restated here would be a second copy
// of a rule, free to disagree with the one the game runs through. An
// out-of-range value is therefore refused by the parser, in the parser's own
// words, which is also the wording an author has already seen from `hexforge
// skills edit`.
func (f WeighField) set(declared skill.Skill, value int) skill.Skill {
	switch f {
	case WeighAccuracy:
		declared.Accuracy = value
	case WeighCooldown:
		declared.Cooldown = value
	case WeighCrit:
		declared.Crit = value
	case WeighDrains:
		declared.Drains = value
	case WeighPierce:
		declared.Pierce = value
	case WeighPower:
		declared.Power = value
	case WeighRange:
		declared.Range = value
	case WeighSelfGradient:
		// A fresh pointer rather than an assignment through the old one:
		// declared is a shallow copy, so writing AtEmpty in place would move the
		// gradient of the skill the book holds — both twins at once, which is a
		// weighing of nothing.
		//
		// Nought is passed through like any other value. A gradient of nought is
		// what the parser refuses, in the parser's own words, and mapping it to
		// nil here is the second copy of that bound this function exists not to
		// hold.
		declared.SelfGradient = &skill.Gradient{AtEmpty: value}
	default:
		declared.Strikes = value
	}
	return declared
}

// Mechanism is one thing a skill does when it works, and therefore one kind of
// evidence that the skill was in the fight at all.
//
// It is a closed table for the reason WeighField is, and it exists because that
// evidence used to be one thing: a count of landed damaging strikes. That is the
// right proof for a skill whose mechanism *is* damage and no proof at all for
// one whose power is nought — poison_powder lands nothing however well it works,
// so every buff, debuff, heal, cleanse and summon in the book came back "cast N
// time(s) and landed none", which was true and was not the question. The refusal
// was right and its evidence was mis-specified; this is the evidence.
//
// ⚠️ A mechanism is not a WeighField and the two must not be read as a pair.
// A field is a dial an author moves; a mechanism is what the skill does when the
// dial is where it is. `cooldown` moves no mechanism of its own — it decides how
// often the skill is cast at all — and what proves a cooldown weighing measured
// anything is whatever that skill does taking hold at least once.
type Mechanism uint8

const (
	// Striking through Summoning are declared in the order they are drawn, so a
	// report's columns are declaration order and the two cannot drift apart.
	Striking Mechanism = iota
	Applying
	Restoring
	Cleansing
	Summoning
)

// mechanismCount is how many mechanisms a skill can have.
const mechanismCount = int(Summoning) + 1

// mechanismNames is the column each mechanism is drawn under: what happened,
// counted.
var mechanismNames = [mechanismCount]string{
	Striking:  "landed",
	Applying:  "applied",
	Restoring: "healed",
	Cleansing: "cleansed",
	Summoning: "summoned",
}

// mechanismNothings is how a refusal says this mechanism was never observed.
//
// ⚠️ Striking's is "landed none" rather than "landed no strike", and the exact
// words are load-bearing: that is the sentence a damaging skill has always been
// refused with, it is quoted in CLAUDE.md, and widening the guard must not
// reword the refusal it already gave.
var mechanismNothings = [mechanismCount]string{
	Striking:  "landed none",
	Applying:  "applied no status",
	Restoring: "restored no health",
	Cleansing: "cleansed nothing",
	Summoning: "summoned nothing",
}

// String is the mechanism's name as it is printed.
func (m Mechanism) String() string {
	if int(m) >= mechanismCount {
		return fmt.Sprintf("mechanism(%d)", uint8(m))
	}
	return mechanismNames[m]
}

// Nothing is how this mechanism reads when the log recorded none of it.
func (m Mechanism) Nothing() string {
	if int(m) >= mechanismCount {
		return fmt.Sprintf("did no mechanism(%d)", uint8(m))
	}
	return mechanismNothings[m]
}

// Count is how often the log recorded this mechanism over a row.
func (m Mechanism) Count(strikes Strikes, effects Effects) int {
	switch m {
	case Striking:
		return strikes.Landed
	case Applying:
		return effects.Applied
	case Restoring:
		return effects.Restored
	case Cleansing:
		return effects.Stripped
	default:
		return effects.Summoned
	}
}

// Mechanisms is what a skill declares it does, in declaration order.
//
// ⚠️ It reads the skill **as fought** rather than as shipped, and that matters
// on the one field that can change what a skill is: a power swept up from nought
// makes a support skill damaging for those rows, and a power swept down to
// nought takes the damage away. A row checked against the shipped skill's
// mechanism would be checked against a mechanism that row did not have.
//
// A skill that declares none at all comes back empty, and Weigh refuses such a
// row in so many words. That is not a gap to be filled with a default: a skill
// which strikes for nothing, applies nothing, restores nothing, cleanses nothing
// and summons nothing does nothing, and there is no field on it whose worth
// could be anything but the noise.
func Mechanisms(declared skill.Skill) []Mechanism {
	found := make([]Mechanism, 0, mechanismCount)
	if declared.Power > 0 {
		found = append(found, Striking)
	}
	if len(declared.Applies) > 0 || len(declared.SelfApplies) > 0 {
		found = append(found, Applying)
	}
	// Both places a heal can be written. A reserve-paid heal carries no flat
	// `restores` at all — the whole payout is the condition's rate times the
	// stacks the spend takes — so a reading of that field alone would report such
	// a skill as declaring no mechanism, and Weigh refuses a row that declares
	// none in so many words.
	if declared.Restores > 0 ||
		(declared.SelfRequires != nil && declared.SelfRequires.StackRestore > 0) {
		found = append(found, Restoring)
	}
	if declared.Strips != nil {
		found = append(found, Cleansing)
	}
	if declared.Summons != nil {
		found = append(found, Summoning)
	}
	return found
}

// mechanismsOver is every mechanism any row of one sweep could show.
//
// The union across the swept variants rather than the shipped skill's own, for
// the reason Mechanisms reads the variant: one field can change what a skill is,
// so a header taken off the control alone would leave the other rows' work in a
// column that is not on the page.
func mechanismsOver(shipped skill.Skill, field WeighField, values []int) []Mechanism {
	var seen [mechanismCount]bool
	for _, value := range values {
		for _, worked := range Mechanisms(field.set(shipped, value)) {
			seen[worked] = true
		}
	}
	all := make([]Mechanism, 0, mechanismCount)
	for worked := 0; worked < mechanismCount; worked++ {
		if seen[worked] {
			all = append(all, Mechanism(worked))
		}
	}
	return all
}

// fired reports whether the mechanism this FIELD turns on was observed at all.
//
// It is a narrower question than whether the skill worked, which is what
// Mechanisms answers, and the two are asked separately because they fail
// separately: a crit chance can go unrolled across a battle in which every
// strike landed.
//
// Only crit has an answer worth asking for here, and that is a fact about the
// event log rather than a shortcut. A critical strike is the one thing on the
// field list with a mark of its own on an event, so it is the one thing that can
// be absent while the skill is plainly working — a crit chance of two hundred
// that never once came up would price the chance at whatever the noise happened
// to be.
//
// For every other field the skill's own mechanism is the whole of the answer. A
// power, a pierce or a drain that resolved has already been applied by the time
// the log says Damaged, and there is no second thing left to have failed to
// happen; a cooldown, an accuracy or a range is spent on getting the skill cast
// at all, so what proves it measured something is that the skill did its own
// work — which for a damaging skill is a landed strike and for a support skill
// is a status taking hold, a heal landing, a cleanse or a summon. That list is
// Mechanisms, and refuseUnreadable is where it is checked.
func (f WeighField) fired(value int, strikes Strikes) bool {
	if f != WeighCrit || value == 0 {
		return true
	}
	return strikes.Critical > 0
}

// variantID is what the in-memory copy of a skill is called.
//
// The separator is a tilde rather than an at sign because `@` is already spoken
// for: UnlockSummary writes `poison_powder@8` for "at level 8", and a second
// meaning for the same punctuation would make one of the two unreadable the
// first time somebody met it. It is spelled to be read — `razor_leaf~crit=200`
// says which skill, which field and which value — because it is what a refusal
// names and what the counting is attributed to.
func variantID(id string, field WeighField, value int) string {
	return fmt.Sprintf("%s~%s=%d", id, field, value)
}

// WeighRequest is one price to be taken: one field, on one skill, as one
// character carries it.
type WeighRequest struct {
	Character string
	Skill     string
	Field     WeighField
	// Values are the values to sweep. The skill's own value is always added, so
	// a sweep that forgot the control still has one.
	Values []int
	Level  int
	// Seeds is how many battles each half of each row is fought over, so a row
	// costs twice this many.
	Seeds int
	// Stage is the form the carrier is weighed as, and is only ever needed on a
	// line that forks — absent means the furthest the level reaches, which is
	// what every weighing meant before any line had two arms. A forking carrier
	// with none is refused naming both, because a price is a fact about one
	// stat line and two arms are two of them.
	Stage string
}

// Weighing is one row: what the field was set to, and what that was worth.
type Weighing struct {
	Value int
	// Control marks the row where the field is left at the value the book
	// declares. Both sides are then the same skill under two ids, so it must
	// read exactly even — and a report whose control does not is refused whole.
	Control bool
	// Rate is the challenger's balanced share in parts per thousand, both slots
	// added up.
	Rate int
	// Turns is the median length of the duels that finished.
	Turns int
	// Edge is what moving first was worth in this row, in parts per thousand.
	Edge    int
	Tally   Tally
	Strikes Strikes
	// Effects is everything the skill did that was not a strike, which for a
	// skill of nought power is everything it did at all.
	Effects Effects
}

// Worth is the deviation from an even split, signed, in parts per thousand.
//
// This is the whole of what a weighing reports. The two sides are the same unit,
// the same stats, the same kit, the same placement and the same level, fought
// from both slots so the queue's tie-break cancels — so everything that could
// make the figure anything other than the price of the one field has been made
// identical on purpose, and what is left over is that field.
func (w Weighing) Worth() int { return w.Rate - scale.Base/2 }

// WeighReport is a sweep of one field, and everything needed to read it.
type WeighReport struct {
	Carrier Duellist
	Skill   string
	Field   WeighField
	// Shipped is the value the book declares, which is the control row's value.
	Shipped int
	Seeds   int
	// Band is the two-sigma width of a row's rate in parts per thousand: the
	// figure a difference has to clear before it is a finding rather than noise.
	Band int
	// Mechanisms is what the swept skill does, which is what a row has to have
	// done to be believed and what the drawing gives a column each. It is the
	// union over the swept values rather than the control's own: see
	// mechanismsOver.
	Mechanisms []Mechanism
	Rows       []Weighing
}

// Battles is how many duels the whole report was taken from.
func (r WeighReport) Battles() int { return 2 * r.Seeds * len(r.Rows) }

// MonotoneWorth reports whether worth only ever moves one way across the sweep,
// counting a step inside the band as no step at all.
//
// A dial that is not monotone is not priced, whatever the numbers say: if more
// of a thing is sometimes worth less, then the figure beside any one value is
// not the value's worth. This is the check the roster win rate failed — more ally
// damage lowered it — and it is on the report so that it is read every time
// rather than remembered.
func (r WeighReport) MonotoneWorth() bool { return monotone(r.series(worthOf), r.Band) }

// MonotoneTurns reports the same about the median turn count.
//
// It is a separate answer from MonotoneWorth and is reported separately, because
// the two can disagree and the disagreement is the finding: a row whose worth
// sits inside the band while its turns moved has a real effect that the rate was
// too lumpy to see.
func (r WeighReport) MonotoneTurns() bool { return monotone(r.series(turnsOf), 0) }

func worthOf(row Weighing) int { return row.Worth() }
func turnsOf(row Weighing) int { return row.Turns }

func (r WeighReport) series(read func(Weighing) int) []int {
	out := make([]int, 0, len(r.Rows))
	for _, row := range r.Rows {
		out = append(out, read(row))
	}
	return out
}

// monotone reports whether a series only ever moves one way, treating a step no
// larger than the tolerance as no move.
//
// The tolerance is what makes the answer mean anything at a finite number of
// seeds. Every figure here is a measurement, so a perfectly monotone curve will
// wobble by less than its own band somewhere along it — demanding an exactly
// ordered series would report every real curve as unordered, and reporting a
// series ordered only outside its band is the same claim the band itself makes.
func monotone(series []int, tolerance int) bool {
	direction := 0
	for i := 1; i < len(series); i++ {
		step := series[i] - series[i-1]
		if step <= tolerance && step >= -tolerance {
			continue
		}
		way := 1
		if step < 0 {
			way = -1
		}
		if direction != 0 && way != direction {
			return false
		}
		direction = way
	}
	return true
}

// endlessShare is the share of a row's battles that may end in no result before
// the row is refused.
//
// A fifth, because a rate is taken over the battles that decided and an endless
// battle is left out of it: past this many, the figure on screen is a
// measurement of the minority that finished, and whether a pairing resolves at
// all is exactly the thing a small change to a damage number moves.
const endlessShare = 5

// saturated is the rate at which one half of a row has no room left to move, in
// parts per thousand. Ninety-nine per cent of the battles going one way means a
// field made larger cannot show it, so the row cannot price anything above it.
const saturated = 990

// NotBroughtError is a skill the carrier does not field.
//
// It is a type rather than a sentence for one reason, and it is not
// translation: a sweep over the whole cast has to decide which characters are
// *in* the table, and "brings the skill" is exactly this refusal read as a
// question. A sweep that answered it for itself would hold a second copy of the
// kit rule — free to disagree with the one a single weighing enforces — so the
// membership test is this error, matched with errors.As.
//
// The wording is what Weigh printed before the type existed, unchanged.
type NotBroughtError struct {
	Carrier string
	Skill   string
	Level   int
	Stage   string
	// Brings is the kit the carrier actually fielded, which is the half of the
	// sentence that tells an author what to weigh instead.
	Brings []string
}

func (e *NotBroughtError) Error() string {
	return fmt.Sprintf("%s does not bring %s at level %d as %s; it brings %s",
		e.Carrier, e.Skill, e.Level, e.Stage, strings.Join(e.Brings, " "))
}

// UnevenControlError is a control row that did not come out exactly even.
//
// It is a type for the reason above and one more: across a table of carriers
// every other refusal says the *carrier* is uninteresting, and this one says the
// *harness* leaked. Those must not print as the same kind of line, and a reader
// asked to tell them apart by reading the sentence will one day not.
type UnevenControlError struct {
	Skill string
	Field WeighField
	Value int
	Rate  int
	Tally Tally
}

func (e *UnevenControlError) Error() string {
	return fmt.Sprintf(
		"the control row (%s %d, the value %s declares) came to %d rather than an even %d, "+
			"so the two sides are not the same fight and no other row on this sweep means anything: %+v",
		e.Field, e.Value, e.Skill, e.Rate, scale.Base/2, e.Tally)
}

// Weigh prices one field on one skill, as one character carries it, against a
// copy of itself.
//
// # What it is
//
// Both sides are the same character, at the same level, in the same form, with
// the same stats, the same trait and the same four skills — and the challenger's
// copy of the one skill has the one field moved. Every duel is fought from both
// slots and added, so the turn queue's tie-break cancels the way it does in a
// spar. What is left when everything else has been made identical is the price
// of that field, and it is read as a deviation from an even split.
//
// # What it is not
//
// It is not a roster win rate, and the reason that matters is that a roster win
// rate was tried first and does not work. Giving the ally more damage *lowers*
// its share against the shipped roster by about as much as adding a crit chance
// does, so the rate is not monotone in ally damage and a change measured through
// it is priced by whatever the placement happens to be — the same change read
// +2.4 points before a placement change and −2.1 after. A price taken here is a
// price on this carrier at this level against itself, and it does not transfer.
//
// ⚠️ The two twins share one *rng.Source, so a change on one side re-scrambles
// every draw after it. That is roll drift, and it is why the variance is what it
// is; it biases nothing, because both arrangements fight the same seeds. Do not
// try to fix it by seeding the two units separately — there is no such seam, and
// making one would be a change to internal/core in order to make a measurement
// prettier.
func (l *Library) Weigh(request WeighRequest) (WeighReport, error) {
	if request.Seeds < 1 {
		return WeighReport{}, fmt.Errorf("a weighing over %d battles measures nothing", request.Seeds)
	}
	if request.Level < 1 || request.Level > progression.LevelCap {
		return WeighReport{}, fmt.Errorf("level %d is outside 1..%d", request.Level, progression.LevelCap)
	}
	character, known := l.characters.Get(request.Character)
	if !known {
		return WeighReport{}, fmt.Errorf("no character is called %q", request.Character)
	}
	carrier, err := l.duellist(character, request.Level, request.Stage)
	if err != nil {
		return WeighReport{}, err
	}
	shipped, err := l.skills.Lookup(request.Skill)
	if err != nil {
		return WeighReport{}, err
	}
	// A skill the carrier does not bring is refused rather than measured,
	// because it would be measured: the variant would sit in the book, nobody
	// would cast it, and the row would come back an even split — a price of
	// nought on a skill that was never in the fight.
	if !slices.Contains(carrier.Skills, request.Skill) {
		return WeighReport{}, &NotBroughtError{
			Carrier: carrier.ID, Skill: request.Skill, Level: carrier.Level,
			Stage: carrier.Stage, Brings: carrier.Skills,
		}
	}

	control := request.Field.of(shipped)
	values := sweep(request.Values, control)
	report := WeighReport{
		Carrier: carrier, Skill: request.Skill, Field: request.Field,
		Shipped: control, Seeds: request.Seeds, Band: band(request.Seeds),
		Mechanisms: mechanismsOver(shipped, request.Field, values),
	}

	// The control is fought first and checked before anything else is believed.
	// A harness that leaked the variant into both kits, attributed the wrong
	// side, or perturbed the rng would produce rows that look exactly like these
	// ones, and this is the row where that shows.
	controlRow, err := l.weighOne(carrier, shipped, request, control)
	if err != nil {
		return WeighReport{}, err
	}
	controlRow.Control = true
	if err := refuseUnevenControl(controlRow, request.Field, request.Skill); err != nil {
		return WeighReport{}, err
	}

	for _, value := range values {
		if value == control {
			report.Rows = append(report.Rows, controlRow)
			continue
		}
		row, err := l.weighOne(carrier, shipped, request, value)
		if err != nil {
			return WeighReport{}, err
		}
		report.Rows = append(report.Rows, row)
	}
	return report, nil
}

// sweep is the values to fight, with the skill's own value always among them,
// deduplicated and in order.
//
// The control is inserted rather than expected because a sweep without one is a
// column of numbers with nothing to read them against, and an author asked to
// remember to include it will one day not.
func sweep(values []int, control int) []int {
	all := append(append([]int(nil), values...), control)
	slices.Sort(all)
	return slices.Compact(all)
}

// refuseUnevenControl is the check every report makes on itself before a single
// figure of it is believed.
//
// The control row fights the skill against the same skill under another id, so
// it is even by construction — not by luck, and not to within a band. A harness
// that leaked the variant into both kits, attributed the wrong side, perturbed
// the rng, or swapped the two halves would produce rows that look exactly like
// good ones, and this is the row where each of those shows as an inequality.
//
// It is a check on every run rather than a test somewhere else because it is a
// claim about *this* report: a test proves the code was right once, and this
// proves the twenty thousand battles just fought were the twenty thousand
// battles intended.
func refuseUnevenControl(control Weighing, field WeighField, skillID string) error {
	if control.Rate == scale.Base/2 {
		return nil
	}
	return &UnevenControlError{
		Skill: skillID, Field: field, Value: control.Value,
		Rate: control.Rate, Tally: control.Tally,
	}
}

// variantOf is the in-memory copy of a skill with one field moved, and the book
// it lives in.
//
// ⚠️ Append rather than Replace, and the book comes back rather than going onto
// the library. Replace would change the skill for *both* sides, which is a
// weighing of nothing; assigning to l.skills would leave the tool holding an
// edited book nobody wrote to disk, so every later command in the same process
// would report on a skill that does not exist.
//
// The variant keeps the original's element and its restriction, which is what
// lets both twins past battle.enlist's carry check — a variant that dropped
// either would be refused for one side and not the other, and the row would be
// an error rather than a price.
//
// A parser refusal comes back whole and unreworded. It is the same sentence
// `hexforge skills edit` gives for the same number, and rewording it here would
// be this package holding an opinion about a bound it does not own.
func (l *Library) variantOf(shipped skill.Skill, field WeighField, value int) (skill.Skill, *skill.Book, error) {
	id := variantID(shipped.ID, field, value)
	// A collision would mean the challenger and the opponent share a skill after
	// all, or that a real skill quietly changed under a name somebody authored.
	if _, clash := l.skills.Lookup(id); clash == nil {
		return skill.Skill{}, nil, fmt.Errorf(
			"the variant would be called %q, which is a skill the book already declares", id)
	}
	variant := field.set(shipped, value)
	variant.ID = id
	grown, err := l.skills.Append(l.SkillDeps(), variant)
	if err != nil {
		return skill.Skill{}, nil, err
	}
	return variant, grown, nil
}

// weighOne fights one row and refuses it if it cannot be read.
func (l *Library) weighOne(carrier Duellist, shipped skill.Skill, request WeighRequest, value int) (Weighing, error) {
	variant, variantBook, err := l.variantOf(shipped, request.Field, value)
	if err != nil {
		return Weighing{}, err
	}
	id := variant.ID
	books := l.Books()
	books.Skills = variantBook

	// The swap is in place, so the kit's order — which is the character's own
	// stated preference — is the same on both sides.
	challenger := carrier
	challenger.Skills = slices.Clone(carrier.Skills)
	challenger.Skills[slices.Index(challenger.Skills, request.Skill)] = id

	fought, err := duel(books, challenger, carrier, request.Seeds, false, id)
	if err != nil {
		return Weighing{}, fmt.Errorf("%s at %s %d cannot be measured: %w",
			request.Skill, request.Field, value, err)
	}
	row := Weighing{
		Value: value, Rate: fought.Rate(), Turns: fought.Turns,
		Edge: fought.Edge(), Tally: fought.Total(), Strikes: fought.Strikes,
		Effects: fought.Effects,
	}
	// The VARIANT rather than the shipped skill, because the variant is what was
	// fought: a power swept to nought is a skill that lands nothing by design,
	// and checking it against the shipped skill's mechanism would refuse it for
	// failing to do something this row had taken away from it.
	return row, refuseUnreadable(row, request, variant, fought)
}

// refuseUnreadable is every way a row can look like a figure and be none.
//
// Each of these would otherwise print as an ordinary number in an ordinary
// column. That is the whole danger: a price of nought and the absence of a price
// are the same glyph, and only the row itself knows which it is.
//
// fielded is the skill **as this row fought it**, and it is here rather than
// derived because the first refusal below asks what the skill does. See
// Mechanisms.
func refuseUnreadable(row Weighing, request WeighRequest, fielded skill.Skill, fought Matchup) error {
	at := fmt.Sprintf("%s at %s %d", request.Skill, request.Field, row.Value)
	worked := Mechanisms(fielded)
	switch {
	case len(worked) == 0:
		// A skill with no mechanism at all is refused before any counting,
		// because there is nothing to count. It is not the same refusal as the
		// one below and must not read like it: that one says the skill did not
		// work, and this one says there is nothing this skill could have done.
		return fmt.Errorf("nothing here prices %s: it strikes for nothing, applies nothing, "+
			"restores nothing, cleanses nothing and summons nothing, so there is no mechanism "+
			"for %s to be the price of",
			at, request.Field)
	case !anyFired(worked, row):
		// Worth nothing means *not rated*, never rated at nought. A skill that
		// never did the thing it does has an even row beside it and prices
		// nothing at all.
		return fmt.Errorf("nothing here prices %s: it was cast %d time(s) and %s, "+
			"so the row is the absence of a measurement rather than a measurement of nothing",
			at, row.Strikes.Cast, nothingHappened(worked))
	case !request.Field.fired(row.Value, row.Strikes):
		return fmt.Errorf("nothing here prices %s: the mechanism never fired across %d landed strike(s), "+
			"so what the row reports is the noise and not the field",
			at, row.Strikes.Landed)

	case row.Tally.Endless*endlessShare > row.Tally.Battles():
		return fmt.Errorf("%s left %d of %d battle(s) undecided, which is more than a fifth: "+
			"the rate is taken over what finished, and whether this pairing finishes is itself moving",
			at, row.Tally.Endless, row.Tally.Battles())
	case fought.First.Rate() >= saturated || fought.Second.Rate() <= scale.Base-saturated:
		return fmt.Errorf("%s is saturated — one slot wins %s of what it decides and the other %s — "+
			"so there is no room left for the field to move and the row cannot price it",
			at, PercentInColumn(fought.First.Rate()), PercentInColumn(fought.Second.Rate()))
	}
	return nil
}

// anyFired reports whether the log recorded the skill doing any one of the
// things it declares it does.
//
// Any rather than all, and the difference is a design. A skill that both applies
// a status and restores health did something when either landed — the one that
// did not land is a fact about the fight rather than about the instrument, and
// demanding both would refuse a row that measured a real difference.
func anyFired(worked []Mechanism, row Weighing) bool {
	for _, mechanism := range worked {
		if mechanism.Count(row.Strikes, row.Effects) > 0 {
			return true
		}
	}
	return false
}

// nothingHappened is how a refusal says the skill did none of its own work,
// naming every mechanism it has rather than only the first.
//
// Every one of them, because the reader's next question is which of the things
// this skill does failed to happen, and a sentence naming one of three would
// send them to look at the wrong half of the skill.
func nothingHappened(worked []Mechanism) string {
	said := make([]string, 0, len(worked))
	for _, mechanism := range worked {
		said = append(said, mechanism.Nothing())
	}
	return strings.Join(said, " and ")
}

// band is the two-sigma width of a rate measured over one row, in parts per
// thousand.
//
// A row is two halves of the given seeds, and a rate near even over N battles
// has a standard deviation of 500/√N parts per thousand, so two of them is
// 1000/√N. Integer arithmetic throughout and rounded up: the house style is
// permille everywhere because internal/core has no floats in it, and a band
// rounded down would be a band that lets a difference through.
//
// It is never zero, because ceiling division of a positive numerator cannot be.
// A band of zero would mean every wobble is a finding.
func band(seeds int) int {
	root := isqrt(2 * seeds)
	if root < 1 {
		root = 1
	}
	return (scale.Base + root - 1) / root
}

// isqrt is the integer square root, by Newton's method.
func isqrt(value int) int {
	if value < 2 {
		if value < 0 {
			return 0
		}
		return value
	}
	root, next := value, (value+1)/2
	for next < root {
		root, next = next, (next+value/next)/2
	}
	return root
}
