package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// defaultWeighSeeds is what a row costs when nobody says: ten thousand seeds
// each way, which is twenty thousand battles and about seven and a half seconds.
//
// The figure comes from the band and not from taste. Twenty thousand battles put
// two sigma at eight parts per thousand — forge.band is the arithmetic — and the
// effects this instrument was built to price were measured at seventeen to
// twenty-four parts per thousand, two to three times that band. A smaller
// default would report a real effect as noise; a larger one would buy resolution
// nothing needs, at a cost paid on every run.
const defaultWeighSeeds = 10000

func runWeigh(args []string) error {
	set := newFlagSet("weigh")
	dir := dataFlag(set)
	level := set.Int("level", progression.LevelCap,
		"the level the carrier is fielded at; the cap is what the stat budget is written for")
	seeds := set.Int("seeds", defaultWeighSeeds,
		"how many battles each row is fought over from each slot")
	field := set.String("field", "",
		"which one number to move: "+strings.Join(forge.FieldNames(), ", "))
	// Required, with no default. A default range would be the tool guessing at
	// what is worth trying, and a guess printed in a table reads exactly like a
	// finding — the one thing this instrument exists to stop.
	values := set.String("values", "",
		"the values to sweep, comma separated. The skill's own value is always added as the control")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 2 {
		return fmt.Errorf("usage: hexforge weigh <character> <skill> --field F --values a,b,c [--level N] [--seeds N]")
	}
	if strings.TrimSpace(*field) == "" {
		return fmt.Errorf("--field is required: name one of %s", strings.Join(forge.FieldNames(), ", "))
	}
	weighed, err := forge.ParseWeighField(strings.TrimSpace(*field))
	if err != nil {
		return err
	}
	if strings.TrimSpace(*values) == "" {
		return fmt.Errorf("--values is required: a sweep with no values to try is the control row alone, " +
			"and which values are worth trying is the author's question rather than this tool's")
	}
	swept, err := parseValues(*values)
	if err != nil {
		return err
	}
	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	report, err := lib.Weigh(forge.WeighRequest{
		Character: operands[0], Skill: operands[1], Field: weighed,
		Values: swept, Level: *level, Seeds: *seeds,
	})
	if err != nil {
		return err
	}
	renderWeigh(os.Stdout, report)
	return nil
}

// parseValues reads the comma-separated sweep.
func parseValues(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("the value list %q has an empty entry in it", raw)
		}
		value, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number; every field a weighing moves is one integer", trimmed)
		}
		values = append(values, value)
	}
	return values, nil
}

// renderWeigh is this front-end's drawing of forge.WeighReport. The measurement
// is in the package; only the columns are here.
//
// ⚠️ worth and turns are co-equal headline columns and are drawn next to each
// other for that reason. A rate is lumpy at the kill threshold — a little more
// damage buys whether the last strike lands this turn or next, which is discrete
// per seed — so a real effect can sit inside the band. The median turn count has
// no such boundary: more damage kills sooner, every time. A reader who reads only
// worth will call a real effect noise, which is the mistake this whole file was
// written after.
func renderWeigh(out io.Writer, report forge.WeighReport) {
	carrier := report.Carrier
	fmt.Fprintf(out, "%s — %s at level %d as %s, %s\n",
		carrier.ID, carrier.Name, carrier.Level, carrier.Stage, carrier.Affinity)
	fmt.Fprintf(out, "weighing %s %s against a copy of itself; the book declares %d\n",
		report.Skill, report.Field, report.Shipped)
	fmt.Fprintf(out, "brings %s", strings.Join(carrier.Skills, " "))
	if len(carrier.Passives) > 0 {
		fmt.Fprintf(out, " and %s", strings.Join(carrier.Passives, " "))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)

	rendered := newTable(
		"value", "worth", "±", "turns", "first move",
		"won", "lost", "drawn", "endless", "cast", "landed", "crit").
		rightAlign(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	for _, row := range report.Rows {
		value := strconv.Itoa(row.Value)
		if row.Control {
			value += " (control)"
		}
		rendered.add(value,
			signed(row.Worth()),
			"±"+forge.PercentInColumn(report.Band),
			strconv.Itoa(row.Turns),
			signed(row.Edge),
			strconv.Itoa(row.Tally.Wins), strconv.Itoa(row.Tally.Losses),
			strconv.Itoa(row.Tally.Draws), strconv.Itoa(row.Tally.Endless),
			strconv.Itoa(row.Strikes.Cast), strconv.Itoa(row.Strikes.Landed),
			strconv.Itoa(row.Strikes.Critical))
	}
	rendered.render(out)

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d rows, %d seeds from each slot, %d battles in all; the band is ±%s\n",
		len(report.Rows), report.Seeds, report.Battles(), forge.PercentInColumn(report.Band))
	fmt.Fprintf(out, "worth: %s (a step inside the band counts as no step)\n",
		monotonically(report.MonotoneWorth()))
	fmt.Fprintf(out, "turns: %s\n", monotonically(report.MonotoneTurns()))
	fmt.Fprint(out, "\nnote: worth and turns are the two headline columns and are read together. A row\n"+
		"whose worth sits inside the band while its turns moved is a real effect: a rate is\n"+
		"lumpy at the kill threshold, and the median turn count is the reading taken inside\n"+
		"one battle, which has no win boundary to be lumpy at.\n")
	fmt.Fprintf(out, "\nthis is a price on %s as %s carries it at level %d against a copy of itself,\n"+
		"in parts per thousand. It is not a win rate, it says nothing about how the roster\n"+
		"will do, and it does not carry across a data change — a change to the cast or to a\n"+
		"placement can reverse the sign of a figure taken here.\n",
		report.Skill, carrier.ID, carrier.Level)
}

// monotonically words whether a column only ever moved one way.
func monotonically(ordered bool) string {
	if ordered {
		return "only ever moves one way across the sweep"
	}
	return "⚠️ does NOT only move one way across the sweep, so no row on it is a price"
}
