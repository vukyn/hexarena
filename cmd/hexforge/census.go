package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
)

func runCensus(args []string) error {
	set := newFlagSet("census")
	dir := dataFlag(set)
	against := set.String("against", "",
		"one authored squad to stand in and fight, instead of walking them until a board plays every slot")
	seeds := set.Int("seeds", forge.CensusSeeds,
		"how many battles each of the two arrangements is fought over on a board")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge census <build> [--against SQUAD] [--seeds N]")
	}
	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	build, known := lib.Build(operands[0])
	if !known {
		return fmt.Errorf("no build is called %q; hexforge builds lists them", operands[0])
	}
	if *against != "" {
		census, err := lib.Census(build, *against, *seeds)
		if err != nil {
			return err
		}
		writeCensusHead(os.Stdout, build)
		fmt.Fprintf(os.Stdout, "%d seeds an arrangement, %d battles, one named board\n\n",
			census.Seeds, census.Battles)
		renderCensusBoard(os.Stdout, census)
		fmt.Fprintf(os.Stdout, "\nnote: one board is one matchup, and this is NOT the catalogue rule. A slot\n"+
			"silent here may fire against another squad — run without --against for the walk\n"+
			"that is the rule a build has to pass.\n")
		return nil
	}
	walk, err := lib.CensusWalk(build, *seeds)
	if err != nil {
		return err
	}
	renderCensusWalk(os.Stdout, walk)
	return nil
}

// renderCensusWalk draws the rule: every board the build was measured on, and the
// verdict underneath.
//
// ⚠️ **The verdict is about the walk and never about the last board.** Stopping
// early means the last board printed is the one that left nothing silent, so a
// reader taking the verdict off the bottom table would read a pass as a fact
// about that squad. It is a fact about the set.
func renderCensusWalk(out io.Writer, walk forge.CensusWalk) {
	writeCensusHead(out, walk.Build)
	if len(walk.Taken) > 0 {
		first := walk.Taken[0]
		fmt.Fprintf(out, "%d seeds an arrangement, %d battles a board, %d board(s) walked\n",
			first.Seeds, first.Battles, len(walk.Taken))
	}
	for _, census := range walk.Taken {
		fmt.Fprintln(out)
		renderCensusBoard(out, census)
	}
	fmt.Fprintln(out)
	switch {
	case walk.Plays() && walk.Boards() == 1:
		fmt.Fprintf(out, "plays every slot it names, on the first board it was stood on\n")
	case walk.Plays():
		last := walk.Taken[len(walk.Taken)-1]
		fmt.Fprintf(out, "plays every slot it names; it took %d board(s) to find one, and %s is\n"+
			"the board that did — the ones above it each left something silent\n",
			walk.Boards(), last.Against)
	default:
		fmt.Fprintf(out, "never cast %s against any of the %d authored squad(s): a build may not\n"+
			"name a skill it does not play\n", strings.Join(walk.Silent, ", "), walk.Boards())
	}
	fmt.Fprintf(out, "\nnote: the subject stands in the squad's first slot against that squad intact,\n"+
		"so the other six units cancel and the only difference on the board is its own four\n"+
		"slots. Both arrangements are fought over the same seeds, because the roster order\n"+
		"breaks a tie in the turn queue.\n")
}

// renderCensusBoard is one census: the kit in its own order, counts beside it.
func renderCensusBoard(out io.Writer, census forge.CastCensus) {
	fmt.Fprintf(out, "against %s — %d battles\n", census.Against, census.Battles)
	silent := census.Silent()
	rendered := newTable("skill", "casts", "").rightAlign(1)
	for _, counted := range census.Casts {
		mark := ""
		if slices.Contains(silent, counted.Skill) {
			mark = "silent"
		}
		rendered.add(counted.Skill, strconv.Itoa(counted.Cast), mark)
	}
	rendered.render(out)
}

// writeCensusHead names the build and its kit, once, above the boards.
//
// ⚠️ What the reading COST is not printed here, and the two callers are why: a
// walk paid for several boards and a named board paid for one, and one sentence
// covering both would have to be vague about which. The seeds and the battle
// count are read off a census that was actually taken rather than off the flag,
// so what is printed is what was fought — the rule CastCensus.Against follows,
// one level up: a figure nobody can re-take is not a measurement.
func writeCensusHead(out io.Writer, build cast.Build) {
	fmt.Fprintf(out, "%s — %s", build.ID, build.Character)
	if build.Name != "" {
		fmt.Fprintf(out, ", %q", build.Name)
	}
	if build.Stage != "" {
		fmt.Fprintf(out, " as %s", build.Stage)
	}
	fmt.Fprintf(out, "\nbrings %s", strings.Join(build.Skills, " "))
	if len(build.Passives) > 0 {
		fmt.Fprintf(out, " and %s", strings.Join(build.Passives, " "))
	}
	fmt.Fprintln(out)
}
