package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/forge"
)

// defaultCarrierSeeds is what a whole-cast table costs when nobody says.
//
// It is a fifth of defaultWeighSeeds and the arithmetic is why. A single
// weighing is 2 × seeds × values battles; a table is that again per carrier, so
// the default that makes one question take seven and a half seconds makes a
// four-carrier table take half a minute and a twenty-carrier one take two and a
// half. Two thousand seeds put two sigma at ±1.6%, which is the band the first
// crit chances were actually authored at — `razor_leaf` read +8.4% and `kunai`
// −1.7% against it, five times the band and inside it respectively — so it
// resolves the effects this instrument exists to find while keeping a table a
// thing somebody will run.
//
// A row that lands close to the band is then re-taken on its own with `weigh`
// and the full ten thousand. That is the intended shape of the workflow: the
// table says which carriers are worth a second look, and the single-carrier tool
// takes it.
const defaultCarrierSeeds = 2000

// renderCarriers draws forge.CarriersReport. The measurement is in the package;
// only the columns are here.
//
// ⚠️ There is no headline number on this page and there must never be one. Every
// cell in a row is a price taken against a copy of *that row's* carrier, so two
// rows were fought against two different opponents and are not two readings of
// one quantity. An average of them would be a figure with no referent, and a
// figure with no referent printed at the top of a table is what a reader
// quotes — which is exactly how the roster win rate came to be believed.
//
// ⚠️ worth and turns share a cell, drawn as `worth/turns`, for the reason
// renderWeigh draws them in adjacent columns: a rate is lumpy at the kill
// threshold and can sit inside the band while a real effect moves the median
// turn count, so a reader who reads only worth will call a real effect noise. On
// a table this wide they cannot be adjacent columns without the page wrapping,
// and a wrapped page is a page where the second column is not read at all.
func renderCarriers(out io.Writer, report forge.CarriersReport) {
	fmt.Fprintf(out, "%s %s, priced once per carrier that brings it, at level %d\n",
		report.Skill, report.Field, report.Level)
	fmt.Fprintf(out, "every row is that carrier against a copy of itself; the book declares %s %d\n",
		report.Field, report.Shipped)
	fmt.Fprintf(out, "%d carrier(s) over %d value(s), %d seeds from each slot: %d battles in all\n",
		len(report.Rows), len(report.Values), report.Seeds, report.Battles())
	fmt.Fprintln(out)

	renderCarrierTable(out, report)

	fmt.Fprintln(out)
	renderCarrierRefusals(out, report)
	renderCarrierFooter(out, report)
}

// renderCarrierTable is one line per carrier: a cell per swept value, the band,
// and whatever has to be said about the line as a whole.
func renderCarrierTable(out io.Writer, report forge.CarriersReport) {
	header := make([]string, 0, len(report.Values)+3)
	header = append(header, "carrier")
	for _, value := range report.Values {
		label := strconv.Itoa(value)
		if value == report.Shipped {
			label += " (control)"
		}
		header = append(header, label)
	}
	header = append(header, "±", "note")
	rendered := newTable(header...)
	// Every column but the carrier's name and the note reads better against the
	// right-hand edge, which is what lets a reader run an eye down one value.
	for column := 1; column <= len(report.Values)+1; column++ {
		rendered.rightAlign(column)
	}

	for _, row := range report.Rows {
		cells := make([]string, 0, len(report.Values)+3)
		cells = append(cells, row.Carrier)
		for _, value := range report.Values {
			cells = append(cells, carrierCell(row, value))
		}
		cells = append(cells, "±"+forge.PercentInColumn(report.Band), carrierNote(row))
		rendered.add(cells...)
	}
	rendered.render(out)
}

// carrierCell is one carrier's reading at one value: worth and turns together,
// or a dash where the row was refused.
//
// A dash rather than a blank, and never a nought. A refused row printed as 0.0%
// would read as "this field is worth nothing to this carrier", which is the
// opposite of what a refusal says — the same distinction Weigh makes when it
// refuses a row that landed nothing instead of pricing it at nought.
func carrierCell(row forge.CarrierRow, value int) string {
	weighed, found := row.At(value)
	if !found {
		return "—"
	}
	return signed(weighed.Worth()) + "/" + strconv.Itoa(weighed.Turns)
}

// carrierNote is what has to be said about a whole line.
//
// A refusal is marked here and spelled out under the table, because a refusal
// sentence is a sentence and a table cell is not the place for one. The harness
// marking is deliberately the loudest thing on the page: it does not say this
// carrier is uninteresting, it says the run leaked.
//
// A priced row is marked when its own sweep did not only move one way, because a
// dial that is not monotone is not priced whatever the figures beside it say —
// and on a table the figures are all a reader sees. Worth and turns are reported
// separately, as they are in renderWeigh, because they can disagree and the
// disagreement is the finding.
func carrierNote(row forge.CarrierRow) string {
	switch {
	case row.Leaked():
		return "⚠️ HARNESS — see below"
	case !row.Priced():
		return "refused — see below"
	}
	unordered := make([]string, 0, 2)
	if !row.Report.MonotoneWorth() {
		unordered = append(unordered, "worth")
	}
	if !row.Report.MonotoneTurns() {
		unordered = append(unordered, "turns")
	}
	if len(unordered) == 0 {
		return ""
	}
	return "⚠️ " + strings.Join(unordered, " and ") + " not ordered, so this row prices nothing"
}

// renderCarrierRefusals spells out every row that has no figures, in the words
// the measurement refused it in.
func renderCarrierRefusals(out io.Writer, report forge.CarriersReport) {
	refused := 0
	for _, row := range report.Rows {
		if row.Priced() {
			continue
		}
		if refused == 0 {
			fmt.Fprintln(out, "refused rows — the table keeps them, because a carrier that cannot be")
			fmt.Fprintln(out, "priced is a fact about that carrier and not a fault in the run:")
		}
		refused++
		if row.Leaked() {
			fmt.Fprintf(out, "  ⚠️ %s — %v\n", row.Carrier, row.Err)
			fmt.Fprintln(out, "     This one is the HARNESS and not the carrier. A control row is even by")
			fmt.Fprintln(out, "     construction, so anything else means the measurement leaked: fix it")
			fmt.Fprintln(out, "     before believing any other row on this page.")
			continue
		}
		fmt.Fprintf(out, "  %s — %v\n", row.Carrier, row.Err)
	}
	if refused > 0 {
		fmt.Fprintln(out)
	}
}

// renderCarrierFooter is what the table means, and what it does not.
func renderCarrierFooter(out io.Writer, report forge.CarriersReport) {
	fmt.Fprintf(out, "%d carrier(s), %d seeds from each slot, %d battles in all; the band is ±%s\n",
		len(report.Rows), report.Seeds, report.Battles(), forge.PercentInColumn(report.Band))
	fmt.Fprintf(out, "each cell is worth/turns at that value; the %d (control) column is the value the "+
		"book declares and reads +0.0%% by construction\n", report.Shipped)
	fmt.Fprintf(out, "sorted by worth at %s %d, largest first, ties by character id; a row the harness "+
		"refused sorts above every priced row and every other refusal below them all\n",
		report.Field, report.Largest())
	renderCarrierSkipped(out, report)

	fmt.Fprint(out, "\nnote: worth and turns are the two headline columns and are read together. A row\n"+
		"whose worth sits inside the band while its turns moved is a real effect: a rate is\n"+
		"lumpy at the kill threshold, and the median turn count is the reading taken inside\n"+
		"one battle, which has no win boundary to be lumpy at.\n")

	fmt.Fprintf(out, "\n⚠️ these figures are NOT comparable to each other. Every row is a price on %s as\n"+
		"THAT row's carrier brings it, taken against a copy of that same carrier — so two rows\n"+
		"were fought against two different opponents and are not two readings of one quantity.\n"+
		"A carrier may be compared only to itself, at another value, along its own row. That is\n"+
		"why this report has no headline number and no average: an average of prices taken\n"+
		"against different opponents is a number with no referent.\n", report.Skill)
	fmt.Fprint(out, "\nno figure here is a win rate, it says nothing about how the roster will do, and it\n"+
		"does not carry across a data change — a change to the cast or to a placement can\n"+
		"reverse the sign of a figure taken here.\n")
}

// renderCarrierSkipped says who is not in the table and why.
//
// The count is on the page whether or not anybody was skipped, because "three of
// four characters cannot bring this" and "this is the whole cast" are different
// readings of the same table and nothing else on it tells them apart.
func renderCarrierSkipped(out io.Writer, report forge.CarriersReport) {
	if len(report.Skipped) == 0 {
		fmt.Fprintf(out, "nobody was skipped: every one of the %d character(s) in the book brings %s at level %d\n",
			report.Considered(), report.Skill, report.Level)
		return
	}
	names := make([]string, 0, len(report.Skipped))
	for _, skipped := range report.Skipped {
		names = append(names, skipped.Carrier)
	}
	fmt.Fprintf(out, "skipped %d of %d character(s), all for the same reason — they do not bring %s at\n"+
		"level %d in the form that level reaches, so they are absent rather than a row of noughts: %s\n",
		len(report.Skipped), report.Considered(), report.Skill, report.Level, strings.Join(names, ", "))
}
