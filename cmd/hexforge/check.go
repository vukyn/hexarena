package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/vukyn/hexarena/internal/forge"
)

func runCheck(args []string) error {
	set := newFlagSet("check")
	dir := dataFlag(set)
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) > 0 {
		return fmt.Errorf("hexforge check takes no arguments, got %v", operands)
	}
	// forge.Inspect is Load and then Inspect, and this needs the reading rule in
	// between, so the two halves are spelled out. Library.Inspect already knows
	// what a library with no directory means — artToCheck asks for no pictures
	// rather than reporting sixty-six missing ones — which is what makes an
	// embedded check a report rather than a wall of false problems.
	lib, err := loadForReading("check", set, *dir)
	if err != nil {
		return err
	}
	report := lib.Inspect()
	renderReport(os.Stdout, report)
	if !report.OK() {
		return fmt.Errorf("%d problem(s) in %s", len(report.Problems), inspected(report))
	}
	return nil
}

// inspected is what the report calls the books it read.
//
// Report.Dir is empty for a library with no directory — that is the decision
// forge's own walk holds it to — and this is the one place that empty becomes
// words, so the header, the failure and the closing note cannot disagree about
// what was checked.
func inspected(r forge.Report) string {
	if r.Dir == "" {
		return embeddedBooks
	}
	return r.Dir
}

// renderReport is this front-end's drawing of forge.Report. The finding is in
// the package; only the columns are here.
func renderReport(out io.Writer, r forge.Report) {
	fmt.Fprintf(out, "checked %s: %d origins, %d archetypes, %d characters\n\n",
		inspected(r), r.Origins, r.Archetypes, len(r.Rows))
	if len(r.Rows) > 0 {
		rendered := newTable("character", "art", "stage at cap", "absorbs", "budget left", "stats at cap").
			rightAlign(3, 4)
		for _, row := range r.Rows {
			// One column for however many pictures a character has: "ok" while
			// they are all there, and the count when they are not, because
			// "MISSING" alone on a three-stage character does not say how much
			// is missing and the problem list below is where the names are.
			art := "ok"
			if missing := row.ArtMissing(); missing > 0 {
				art = fmt.Sprintf("MISSING %d/%d", missing, len(row.Art))
			}
			if row.Failure != nil {
				rendered.add(row.ID, art, "-", "-", "-", row.Failure.Error())
				continue
			}
			rendered.add(row.ID, art, row.Stage,
				strconv.FormatInt(row.Budget.Effective, 10),
				strconv.FormatInt(row.Budget.Headroom, 10),
				row.Values.String())
		}
		rendered.render(out)
		fmt.Fprintln(out)
	}
	renderHeld(out, r)
	if r.OK() {
		fmt.Fprintln(out, "no problems found")
	} else {
		for _, problem := range r.Problems {
			fmt.Fprintf(out, "problem: %s\n", problem)
		}
	}
	// Warnings print whether or not the check passed, because a passing check is
	// exactly when they matter: nothing else is going to say this.
	for _, warning := range r.Warnings {
		fmt.Fprintf(out, "warning: %s\n", warning)
	}
	// ⚠️ The embedded note says the art was not looked at, and that is the part
	// worth printing. A check with no directory silently stops asking the one
	// question only a program allowed to read the filesystem can answer, so a
	// clean report here is a narrower claim than a clean report over a directory
	// — and a reader who is not told that reads it as the wider one.
	if r.Dir == "" {
		fmt.Fprintf(out, "\nnote: there was no data directory to read, so this checked %s —\n"+
			"which is exactly what the game boots from, so nothing here needs a rebuild.\n"+
			"The art is NOT embedded and no picture was looked for: run this from the root of\n"+
			"a hexarena checkout, or with --data <dir>, to check that the art is really there.\n",
			embeddedBooks)
		return
	}
	fmt.Fprintf(out, "\nnote: this reads %s from disk. The game boots from the copies baked in by\n"+
		"go:embed, so an edit here needs a rebuild before it reaches a battle.\n", r.Dir)
}

// renderHeld is the same budget again, once per trait, because the table above
// it measures a line nobody fights on.
//
// A second table rather than more columns on the first: how many rows a
// character has here is how many traits its learnset reaches, which is not one
// and is not the same for every character. Squeezing that into the stat-line
// table would either repeat the character on every row of it or hide all but the
// first trait.
func renderHeld(out io.Writer, r forge.Report) {
	rendered := newTable("character", "trait", "absorbs", "budget left", "stats while held").
		rightAlign(2, 3)
	rows := 0
	for _, row := range r.Rows {
		for _, carried := range row.Traits {
			rendered.add(row.ID, carried.Trait,
				strconv.FormatInt(carried.Budget.Effective, 10),
				strconv.FormatInt(carried.Budget.Headroom, 10),
				carried.Values.String())
			rows++
		}
	}
	if rows == 0 {
		return
	}
	rendered.render(out)
	fmt.Fprintf(out, "\nthe budget is checked against the line above this table, not the one in it: a\n"+
		"trait is named on a placement and its grants go on at enlistment, after everything\n"+
		"that could have refused them. Only permanent grants are counted here — a timed buff\n"+
		"going over the bound is what a buff is for, and a gated one is off until it holds.\n\n")
}
