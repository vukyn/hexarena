package forge

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// CharacterReport is what an inspection found out about one character.
type CharacterReport struct {
	ID    string
	Image string
	// ImageExists is the question internal/core/cast is not allowed to ask.
	// The parser has already agreed the path is well shaped; only a program
	// that may read the filesystem can say whether the art is really there.
	ImageExists bool
	Stage       string
	Values      progression.Values
	Budget      Budget
	// Failure is set when the character will not resolve at the level cap,
	// which the parser cannot catch on its own because a line only has to be
	// valid, not reachable at every level.
	Failure error
}

// Report is the whole result of an inspection, assembled without printing
// anything.
//
// Separating the finding from the rendering is what makes the check testable
// and what lets two front-ends draw it differently: a test asks the report
// whether the missing art was noticed instead of scraping it back out of a
// screen, and a full-screen client lays the same rows out its own way.
type Report struct {
	Dir        string
	Origins    int
	Archetypes int
	Rows       []CharacterReport
	// Problems are the reasons a check fails. Each is a value carrying what it
	// is about, and its Error method is the English cmd/hexforge prints, so a
	// front-end may draw either the sentence or the facts.
	Problems []Problem
}

// Problem is one reason a check fails.
//
// It is an interface over a closed set of causes rather than a string, because
// cmd/hexforge-tui says these in the author's language and a sentence cannot be
// translated once it has been built. The unexported method is what closes the
// set: only this package can add a cause, so a front-end switching over them
// has a knowable list — and it still needs a default, which prints the cause's
// own English rather than nothing.
type Problem interface {
	error
	problem()
}

// MissingArtProblem is a character naming art that is not on disk. This is the
// one question internal/core is not allowed to ask.
type MissingArtProblem struct {
	ID    string
	Image string
	Path  string
}

func (p *MissingArtProblem) Error() string {
	return fmt.Sprintf("character %s names the art %s, which is not at %s", p.ID, p.Image, p.Path)
}

func (p *MissingArtProblem) problem() {}

// ResolveProblem is a character whose evolution line will not resolve at one of
// the levels a check tries.
type ResolveProblem struct {
	ID  string
	Err error
}

func (p *ResolveProblem) Error() string {
	return fmt.Sprintf("character %s does not resolve: %v", p.ID, p.Err)
}

func (p *ResolveProblem) Unwrap() error { return p.Err }
func (p *ResolveProblem) problem()      {}

// OK reports whether the data directory is in a state worth shipping.
func (r Report) OK() bool { return len(r.Problems) == 0 }

// Inspect parses the books from disk and asks the filesystem the one question
// the parsers may not.
//
// A book that will not parse comes back as an error rather than a report: there
// is nothing to tabulate if the data does not load, and the parser's own
// message already says exactly what is wrong.
func Inspect(dir string) (Report, error) {
	lib, err := Load(dir)
	if err != nil {
		return Report{}, err
	}
	return lib.Inspect(), nil
}

// Inspect is Inspect over an already-loaded library, which is what a
// long-running front-end has.
func (l *Library) Inspect() Report {
	report := Report{
		Dir:        l.dir,
		Origins:    len(l.origins.All()),
		Archetypes: len(l.archetypes.All()),
	}
	// The presets need no row of their own: ParseArchetypes has already checked
	// every one against the stat budget and the skill book, so reaching here
	// means they all passed. The archetype listing is where their numbers are
	// read.
	for _, character := range l.characters.All() {
		row := CharacterReport{
			ID:          character.ID,
			Image:       character.Image,
			ImageExists: l.ImageExists(character.Image),
		}
		if !row.ImageExists {
			report.Problems = append(report.Problems, &MissingArtProblem{
				ID: character.ID, Image: character.Image,
				Path: l.ImagePath(character.Image),
			})
		}
		// Both ends of the line are resolved: the first level a character can
		// exist at and the last, which is where the stat budget bites.
		if _, _, err := character.Resolve(1); err != nil {
			row.Failure = err
		}
		values, stage, err := character.Resolve(progression.LevelCap)
		if err != nil {
			row.Failure = err
		} else {
			row.Stage = stage.Name
			row.Values = values
			row.Budget = l.Budget(values)
		}
		if row.Failure != nil {
			report.Problems = append(report.Problems,
				&ResolveProblem{ID: character.ID, Err: row.Failure})
		}
		report.Rows = append(report.Rows, row)
	}
	return report
}
