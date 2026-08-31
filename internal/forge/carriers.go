package forge

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// This file is the whole-cast reading of one field, and the decision it is built
// around is the one the item in TODO.md said had to be made first: **what an
// average of prices taken against different opponents means**.
//
// # It means nothing, so there is no average and no headline
//
// A weighing is a price taken against a copy of *the carrier*, and that is the
// only reason it is a price at all: every other difference — stats, kit, form,
// placement, the queue's tie-break — has been made identical on both sides and
// cancelled, and what is left over is the field. Two carriers' figures are
// therefore taken against two different opponents, and the two numbers are not
// measurements of one quantity. Adding them and dividing produces a figure with
// no referent: it is not "what this field is worth", because there is no
// opponent it was taken against and no board it was taken on.
//
// This repository has been burnt by exactly that shape of number twice — the
// roster win rate, which is non-monotone in ally damage and reversed sign on a
// placement change, and the mirror-duel speed reading, which could not even
// order `swiftness` — and both times the fault was the same: a single figure
// quoted away from the conditions that produced it. So this report is **a table
// and nothing else**. Every row carries its own exactly-even control, every row
// is read down its own line, and the footer says in words that two rows may not
// be compared as magnitudes. A carrier may be compared only to itself, at
// another value.
//
// # Why it is not called a sweep
//
// `sweep` in weigh.go already means "the values to fight, control folded in",
// and a second meaning for the same word inside one package would make one of
// the two unreadable the first time somebody met it — the same reason WeighField
// is not called Field.

// CarriersRequest is one field on one skill, priced once for every character
// whose fielded kit brings it.
//
// There is no Character on it, and that absence is the whole difference from a
// WeighRequest: the carrier is not chosen, it is *discovered*, by asking each
// character in the book whether it brings the skill.
type CarriersRequest struct {
	Skill  string
	Field  WeighField
	Values []int
	Level  int
	// Seeds is how many battles each half of each row of each carrier is fought
	// over, so one carrier costs 2 × Seeds × len(values) and the table costs
	// that again per carrier.
	Seeds int
	// Stage is the form every carrier is weighed as, for the reason
	// WeighRequest carries one: a forking line has two stat lines and a price is
	// a fact about one. A carrier it does not apply to is unaffected.
	Stage string
}

// CarrierRow is one carrier's whole weighing, or the refusal that stands where
// its figures would be.
//
// A refusal is a field on the row rather than an error returned from the sweep,
// because a carrier that cannot be priced is a *fact about that carrier* and not
// a fault in the run. The workflow this replaces was seven invocations and a
// shell loop, two of which came back as refusals the author had to notice by
// eye; a sweep that stopped at the first one would be that workflow with fewer
// commands and the same blindness.
type CarrierRow struct {
	// Carrier is the character id. It is here rather than read off the report
	// because a refused row has no report to read it from.
	Carrier string
	// Report is the weighing, and is the zero value when Err is set.
	Report WeighReport
	Err    error
}

// Priced reports whether the row carries figures.
func (r CarrierRow) Priced() bool { return r.Err == nil }

// Leaked reports whether this row's refusal is the harness rather than the
// carrier.
//
// ⚠️ This is the one distinction a reader must not have to make by reading the
// sentence. Every other refusal says *this carrier cannot be priced here*, which
// is ordinary and expected across a whole cast. A control that did not come out
// exactly even says the measurement leaked — a variant in both kits, a side read
// backwards, a perturbed rng — and it is a claim about the run rather than about
// the carrier. Rendered the same way, it would sit in a column of dashes with
// thirty other dashes and be read as "another uninteresting one".
func (r CarrierRow) Leaked() bool {
	var uneven *UnevenControlError
	return errors.As(r.Err, &uneven)
}

// At is the weighing this row took at one swept value.
func (r CarrierRow) At(value int) (Weighing, bool) {
	if r.Err != nil {
		return Weighing{}, false
	}
	for _, row := range r.Report.Rows {
		if row.Value == value {
			return row, true
		}
	}
	return Weighing{}, false
}

// worthAt is the sort key: the row's worth at one value, and nought where there
// is none.
func (r CarrierRow) worthAt(value int) int {
	row, found := r.At(value)
	if !found {
		return 0
	}
	return row.Worth()
}

// rank groups the table before anything is compared inside a group.
//
// A harness refusal sorts above every priced row because it is the one line that
// says the run itself is suspect, and the rest of the refusals below them all
// because they are the rows with nothing to read. Neither group is a magnitude,
// so neither is sorted by one.
func (r CarrierRow) rank() int {
	switch {
	case r.Leaked():
		return 0
	case r.Priced():
		return 1
	default:
		return 2
	}
}

// CarrierSkipped is a character that is not in the table at all.
//
// Absent rather than a zero row, because a row of noughts and a carrier that
// never cast the skill are the same glyphs and opposite facts — the same reason
// Weigh refuses a row that landed nothing instead of pricing it at nought. The
// count and the reason go in the footer so the absence is still on the page.
type CarrierSkipped struct {
	Carrier string
	Why     *NotBroughtError
}

// CarriersReport is one field priced across the cast: a table, with no headline
// number, deliberately.
type CarriersReport struct {
	Skill string
	Field WeighField
	// Shipped is the value the book declares, which is the control column.
	Shipped int
	// Values are the swept values with the control folded in, in order. Every
	// row fought exactly these, so they are the table's columns.
	Values  []int
	Level   int
	Seeds   int
	Band    int
	Rows    []CarrierRow
	Skipped []CarrierSkipped
}

// Battles is what the table cost: one row is 2 × seeds × values, and there is
// one row per carrier.
//
// It is the cost of a *full* table. A row refused partway through fought fewer,
// because Weigh stops at the first value it cannot read — so this is what a
// reader should expect to pay before the run rather than an audit of what was
// paid, and it is printed before the table for that reason.
func (r CarriersReport) Battles() int { return 2 * r.Seeds * len(r.Values) * len(r.Rows) }

// Considered is how many characters the sweep asked, priced and skipped alike.
func (r CarriersReport) Considered() int { return len(r.Rows) + len(r.Skipped) }

// Largest is the biggest value swept, which is the column the table is sorted
// on.
func (r CarriersReport) Largest() int {
	if len(r.Values) == 0 {
		return 0
	}
	return r.Values[len(r.Values)-1]
}

// WeighCarriers prices one field on one skill once per carrier that brings it.
//
// # What it is
//
// len(Rows) independent weighings, each with its own exactly-even control, laid
// out in one table. It is not one measurement over many carriers: it is many
// measurements printed together, and the difference is the whole design — see
// the note at the top of this file for why there is no average and no headline
// figure.
//
// # Who is in it
//
// Every character whose *fielded kit* includes the skill at this level and form.
// Membership is decided by running the weighing and catching NotBroughtError,
// rather than by testing the kit here, so there is exactly one place that knows
// what "brings" means. A character that does not bring it is skipped — absent
// from the table, counted in the footer — because a zero row and a row that was
// never fought print identically.
//
// # What refuses the whole run, and what refuses one row
//
// Seeds, level, the skill's name and every swept *value* are refused whole: they
// are the request, and a request that cannot be answered has no rows to put the
// refusal on. A value in particular is a fact about the skill rather than about
// any carrier, so it is checked once before a die is rolled — printing the same
// parser sentence once per row would bury it in its own repetition. Anything a
// *battle* discovers refuses one row and leaves the others standing.
//
// A table with no rows at all is refused rather than printed empty, on the same
// principle every refusal here rests on: nought rows and nought carriers look
// the same on a page and mean different things.
func (l *Library) WeighCarriers(request CarriersRequest) (CarriersReport, error) {
	if request.Seeds < 1 {
		return CarriersReport{}, fmt.Errorf("a weighing over %d battles measures nothing", request.Seeds)
	}
	if request.Level < 1 || request.Level > progression.LevelCap {
		return CarriersReport{}, fmt.Errorf("level %d is outside 1..%d", request.Level, progression.LevelCap)
	}
	shipped, err := l.skills.Lookup(request.Skill)
	if err != nil {
		return CarriersReport{}, err
	}
	control := request.Field.of(shipped)
	report := CarriersReport{
		Skill: request.Skill, Field: request.Field, Shipped: control,
		Values: sweep(request.Values, control), Level: request.Level,
		Seeds: request.Seeds, Band: band(request.Seeds),
	}
	// The variants do not depend on the carrier — a value the parser refuses is
	// refused identically for every character in the book — so they are built
	// once, before a die is rolled, and a bad value comes back as one sentence
	// rather than as a column of the same sentence repeated per carrier. It
	// fails fast for free: this is the check that would otherwise be discovered
	// after the first row's battles had been fought.
	//
	// The refusal comes back whole and unreworded, which is the rule Weigh holds
	// too: every bound belongs to skill.resolve, and a second wording here would
	// be free to disagree with it.
	for _, value := range report.Values {
		if _, _, err := l.variantOf(shipped, request.Field, value); err != nil {
			return CarriersReport{}, err
		}
	}

	// cast.Book.All is declaration order, which is already deterministic — but
	// the table is sorted below whatever order arrives here, because
	// internal/core bans a map iteration that reaches an output and a report
	// that ordered itself by whatever it was handed would be the same fault one
	// layer up.
	for _, character := range l.characters.All() {
		// ⚠️ Membership is asked of every ARM before the weighing, because a
		// forking line cannot be resolved without one and the refusal that comes
		// back says nothing about whether the skill is even in the kit. Without
		// this a character that brings nothing lands in the table as a refused
		// row — which reads as "could not be priced" where the truth is "was
		// never a carrier", and those are the two things this table keeps apart.
		if skipped, brings := l.brings(character, request); !brings {
			report.Skipped = append(report.Skipped, skipped)
			continue
		}
		// One row a CHARACTER, and a forking line is a refused row rather than
		// two priced ones. A price is a fact about one stat line, so a carrier
		// with two of them has to be told apart by the caller — Stage on the
		// request is how, and the refusal names both arms.
		weighed, err := l.Weigh(WeighRequest{
			Character: character.ID, Skill: request.Skill, Field: request.Field,
			Values: request.Values, Level: request.Level, Seeds: request.Seeds,
			Stage: request.Stage,
		})
		var absent *NotBroughtError
		if errors.As(err, &absent) {
			report.Skipped = append(report.Skipped, CarrierSkipped{Carrier: character.ID, Why: absent})
			continue
		}
		report.Rows = append(report.Rows, CarrierRow{Carrier: character.ID, Report: weighed, Err: err})
	}
	if len(report.Rows) == 0 {
		return CarriersReport{}, fmt.Errorf(
			"no character brings %s at level %d, so there is nothing to price: all %d in the book were skipped",
			request.Skill, request.Level, len(report.Skipped))
	}
	report.order()
	return report, nil
}

// order puts the table in the sequence the footer states.
//
// Priced rows sort by worth at the largest swept value, largest first, because
// that is the end of the sweep an author is choosing from — the whole question
// the table answers is which carriers are worth authoring the value onto. Ties
// break by character id so two carriers that priced the same never swap places
// between runs, and the refusal groups sort by id alone because they have no
// magnitude to sort by.
func (r *CarriersReport) order() {
	largest := r.Largest()
	slices.SortStableFunc(r.Rows, func(a, b CarrierRow) int {
		if by := a.rank() - b.rank(); by != 0 {
			return by
		}
		if a.Priced() && b.Priced() {
			if by := b.worthAt(largest) - a.worthAt(largest); by != 0 {
				return by
			}
		}
		return strings.Compare(a.Carrier, b.Carrier)
	})
	slices.SortStableFunc(r.Skipped, func(a, b CarrierSkipped) int {
		return strings.Compare(a.Carrier, b.Carrier)
	})
}

// brings reports whether any form this character reaches at the request's level
// carries the skill, and the skip to file when none does.
//
// It reads the kit off duellist, which is the same reading Weigh makes, so the
// two cannot disagree about what "brings" means. A level or a line the arms
// cannot be worked out from is treated as bringing it, so the weighing below
// reports that fault in its own words rather than this having a second opinion.
func (l *Library) brings(character cast.Character, request CarriersRequest) (CarrierSkipped, bool) {
	arms, err := character.FurthestAt(request.Level)
	if err != nil {
		return CarrierSkipped{}, true
	}
	var last Duellist
	for _, arm := range arms {
		fielded, err := l.duellist(character, request.Level, arm.Name)
		if err != nil {
			return CarrierSkipped{}, true
		}
		if slices.Contains(fielded.Skills, request.Skill) {
			return CarrierSkipped{}, true
		}
		last = fielded
	}
	return CarrierSkipped{Carrier: character.ID, Why: &NotBroughtError{
		Carrier: character.ID, Skill: request.Skill, Level: last.Level,
		Stage: last.Stage, Brings: last.Skills,
	}}, false
}
