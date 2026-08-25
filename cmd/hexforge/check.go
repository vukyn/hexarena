package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// characterCheck is what check found out about one character.
type characterCheck struct {
	id    string
	image string
	// imageExists is the question internal/core/cast is not allowed to ask.
	// The parser has already agreed the path is well shaped; only a program
	// that may read the filesystem can say whether the art is really there.
	imageExists bool
	stage       string
	values      progression.Values
	effective   int64
	headroom    int64
	// failure is set when the character will not resolve at the level cap,
	// which the parser cannot catch on its own because a line only has to be
	// valid, not reachable at every level.
	failure error
}

// checkReport is the whole result, assembled without printing anything.
//
// Separating the finding from the rendering is what makes check testable: a
// test asks the report whether the missing art was noticed instead of scraping
// it back out of stdout.
type checkReport struct {
	dir        string
	origins    int
	archetypes int
	rows       []characterCheck
	// problems are the reasons check exits non-zero, worded for a person.
	problems []string
}

func (r checkReport) ok() bool { return len(r.problems) == 0 }

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
	report, err := inspect(*dir)
	if err != nil {
		return err
	}
	report.render(os.Stdout)
	if !report.ok() {
		return fmt.Errorf("%d problem(s) in %s", len(report.problems), report.dir)
	}
	return nil
}

// inspect parses the three books from disk and asks the filesystem the one
// question the parsers may not.
//
// A book that will not parse comes back as an error rather than a report: there
// is nothing to tabulate if the data does not load, and the parser's own
// message already says exactly what is wrong.
func inspect(dir string) (checkReport, error) {
	lib, err := load(dir)
	if err != nil {
		return checkReport{}, err
	}
	report := checkReport{
		dir:        lib.dir,
		origins:    len(lib.origins.All()),
		archetypes: len(lib.archetypes.All()),
	}
	// The presets need no row of their own: ParseArchetypes has already checked
	// every one against the stat budget and the skill book, so reaching here
	// means they all passed. `hexforge archetypes` is where their numbers are
	// read.
	for _, character := range lib.characters.All() {
		row := characterCheck{
			id:          character.ID,
			image:       character.Image,
			imageExists: lib.imageExists(character.Image),
		}
		if !row.imageExists {
			report.problems = append(report.problems,
				fmt.Sprintf("character %s names the art %s, which is not at %s",
					character.ID, character.Image, lib.imagePath(character.Image)))
		}
		// Both ends of the line are resolved: the first level a character can
		// exist at and the last, which is where the stat budget bites.
		if _, _, err := character.Resolve(1); err != nil {
			row.failure = err
		}
		values, stage, err := character.Resolve(progression.LevelCap)
		if err != nil {
			row.failure = err
		} else {
			row.stage = stage.Name
			row.values = values
			row.effective = progression.EffectiveHP(values, lib.rules)
			row.headroom = lib.limits.MaxEffectiveHP - row.effective
		}
		if row.failure != nil {
			report.problems = append(report.problems,
				fmt.Sprintf("character %s does not resolve: %v", character.ID, row.failure))
		}
		report.rows = append(report.rows, row)
	}
	return report, nil
}

func (r checkReport) render(out io.Writer) {
	fmt.Fprintf(out, "checked %s: %d origins, %d archetypes, %d characters\n\n",
		r.dir, r.origins, r.archetypes, len(r.rows))
	if len(r.rows) > 0 {
		rendered := newTable("character", "art", "stage at cap", "absorbs", "budget left", "stats at cap").
			rightAlign(3, 4)
		for _, row := range r.rows {
			art := "ok"
			if !row.imageExists {
				art = "MISSING"
			}
			if row.failure != nil {
				rendered.add(row.id, art, "-", "-", "-", row.failure.Error())
				continue
			}
			rendered.add(row.id, art, row.stage,
				strconv.FormatInt(row.effective, 10),
				strconv.FormatInt(row.headroom, 10),
				row.values.String())
		}
		rendered.render(out)
		fmt.Fprintln(out)
	}
	if r.ok() {
		fmt.Fprintln(out, "no problems found")
	} else {
		for _, problem := range r.problems {
			fmt.Fprintf(out, "problem: %s\n", problem)
		}
	}
	fmt.Fprintf(out, "\nnote: this reads %s from disk. The game boots from the copies baked in by\n"+
		"go:embed, so an edit here needs a rebuild before it reaches a battle.\n", r.dir)
}
