package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// defaultSparSeeds is what a run costs when nobody says. It is the same figure
// the full-screen client starts at, so the two front-ends answer the same
// question by default rather than two questions that look alike.
const defaultSparSeeds = 100

func runSpar(args []string) error {
	set := newFlagSet("spar")
	dir := dataFlag(set)
	level := set.Int("level", progression.LevelCap,
		"the level both sides are fielded at; the cap is what the stat budget is written for")
	seeds := set.Int("seeds", defaultSparSeeds,
		"how many battles each pairing is fought over from each slot")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge spar <id> [--level N] [--seeds N]")
	}
	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	report, err := lib.Spar(operands[0], *level, *seeds)
	if err != nil {
		return err
	}
	renderSpar(os.Stdout, report)
	return nil
}

// renderSpar is this front-end's drawing of forge.SparReport. The measurement is
// in the package; only the columns are here.
func renderSpar(out io.Writer, report forge.SparReport) {
	challenger := report.Challenger
	fmt.Fprintf(out, "%s — %s at level %d as %s, %s\n",
		challenger.ID, challenger.Name, challenger.Level, challenger.Stage, challenger.Affinity)
	fmt.Fprintf(out, "brings %s", strings.Join(challenger.Skills, " "))
	if len(challenger.Passives) > 0 {
		fmt.Fprintf(out, " and %s", strings.Join(challenger.Passives, " "))
	}
	fmt.Fprintf(out, "\n%d rows, %d seeds from each slot, %d battles in all\n\n",
		len(report.Matchups), report.Seeds, 2*report.Seeds*len(report.Matchups))

	rendered := newTable("opponent", "rate", "won", "lost", "drawn", "turns", "first move").
		rightAlign(1, 2, 3, 4, 5, 6)
	for _, matchup := range report.Matchups {
		name := matchup.Against.ID
		if matchup.Mirror {
			name += " (control)"
		}
		total := matchup.Total()
		rendered.add(name, forge.PercentInColumn(matchup.Rate()),
			strconv.Itoa(total.Wins), strconv.Itoa(total.Losses), strconv.Itoa(total.Draws),
			strconv.Itoa(matchup.Turns), signed(matchup.Edge()))
	}
	rendered.render(out)

	fmt.Fprintln(out)
	if report.Opponents() == 0 {
		fmt.Fprintln(out, "nobody else is in the book yet, so there is only the control above")
	} else {
		fmt.Fprintf(out, "overall %s against %d opponent(s)\n", forge.PercentInColumn(report.Rate()), report.Opponents())
	}
	if endless := endlessDuels(report); endless > 0 {
		fmt.Fprintf(out, "%d battle(s) never ended and are counted as no result\n", endless)
	}
	fmt.Fprintf(out, "\nnote: both sides bring the first %d skills and %d trait their learnset\n"+
		"declares. Every pairing is fought from both slots and added up, because the turn\n"+
		"queue breaks a tie by placement — the control row shows what a slot alone is worth.\n",
		cast.SkillSlots, cast.TraitSlots)
}

// endlessDuels is how many battles in the whole report hit the turn limit.
func endlessDuels(report forge.SparReport) int {
	endless := 0
	for _, matchup := range report.Matchups {
		endless += matchup.Total().Endless
	}
	return endless
}

// signed is a difference drawn with its direction, which a bare figure cannot
// carry: an advantage to the first slot and an advantage to the second are the
// same number and opposite findings. forge.PercentInColumn writes the minus; the
// plus is here because a column of differences needs both signs or neither.
func signed(permille int) string {
	if permille < 0 {
		return forge.PercentInColumn(permille)
	}
	return "+" + forge.PercentInColumn(permille)
}
