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
	report, err := forge.Inspect(*dir)
	if err != nil {
		return err
	}
	renderReport(os.Stdout, report)
	if !report.OK() {
		return fmt.Errorf("%d problem(s) in %s", len(report.Problems), report.Dir)
	}
	return nil
}

// renderReport is this front-end's drawing of forge.Report. The finding is in
// the package; only the columns are here.
func renderReport(out io.Writer, r forge.Report) {
	fmt.Fprintf(out, "checked %s: %d origins, %d archetypes, %d characters\n\n",
		r.Dir, r.Origins, r.Archetypes, len(r.Rows))
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
	if r.OK() {
		fmt.Fprintln(out, "no problems found")
	} else {
		for _, problem := range r.Problems {
			fmt.Fprintf(out, "problem: %s\n", problem)
		}
	}
	fmt.Fprintf(out, "\nnote: this reads %s from disk. The game boots from the copies baked in by\n"+
		"go:embed, so an edit here needs a rebuild before it reaches a battle.\n", r.Dir)
}
