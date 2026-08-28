package forge

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// ArtReport is one of a character's pictures and whether it is really there.
type ArtReport struct {
	// Stage is the form the picture belongs to, empty for the character's own.
	Stage string
	Image string
	// Exists is the question internal/core/cast is not allowed to ask. The
	// parser has already agreed the path is well shaped; only a program that
	// may read the filesystem can say whether the art is really there.
	Exists bool
}

// CharacterReport is what an inspection found out about one character.
type CharacterReport struct {
	ID string
	// Art is every distinct picture the character can show, its own first and
	// then one per stage that names a different one.
	//
	// A list rather than the single image it used to be, because art a grown
	// form uses is art nobody looks at until the character has grown — so a
	// missing file there is precisely the one that surfaces late, in front of a
	// player rather than in front of a check.
	Art    []ArtReport
	Stage  string
	Values progression.Values
	Budget Budget
	// Traits is what the same character costs once a trait is on it, one entry
	// per trait its learnset can reach at the cap.
	//
	// One row per trait rather than one number, because a trait is chosen on a
	// placement and not on a character: which of them is carried is a decision
	// nobody has made yet at this point, so the honest answer is all of them.
	Traits []TraitBudget
	// Failure is set when the character will not resolve at the level cap,
	// which the parser cannot catch on its own because a line only has to be
	// valid, not reachable at every level.
	Failure error
}

// TraitBudget is what one character costs while one trait is held.
//
// It exists because the budget on the row above it is measured against a line
// nobody fights on. See Library.Held for the whole of why that is, and why the
// answer here is a figure rather than a refusal.
type TraitBudget struct {
	Trait  string
	Values progression.Values
	Budget Budget
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
	// Warnings are things worth saying that are not reasons to fail.
	//
	// The distinction is the whole reason there are two lists. A short-ranged
	// character on a back column is a design an author may well mean — the rest
	// of the squad stands in front of it, and it is the squad rather than the
	// character that has to reach — so refusing it here would refuse a legal
	// game. Saying nothing at all is the other mistake: that is the shape a
	// battle nobody can act in is built out of, and the moment to notice it is
	// while the character is still in front of whoever wrote it.
	Warnings []Warning
}

// Warning is something a check noticed that is not a reason to fail.
//
// It is a closed set of causes with the same shape as Problem, and for the same
// reason: cmd/hexforge-tui says these in the author's language, and a sentence
// cannot be translated once it has been built.
type Warning interface {
	error
	warning()
}

// ShortReachWarning is a character whose kit can only ever hit the enemy's front
// rank.
//
// ⚠️ It used to mean something else: a unit whose range could not span the cells
// between it and the nearest enemy, which battle.New refused outright. Reach is
// counted in ranks now, so distance can no longer strand anybody and that
// refusal is gone — see CLAUDE.md § Invariants.
//
// What is left is worth saying at authoring time all the same. A kit that stops
// at the first rank can do nothing at all until the enemy's front line is dead,
// so the character is a passenger for the opening of every battle it is in. That
// is a legal thing to build and sometimes a deliberate one, which is why it is a
// warning rather than a refusal.
type ShortReachWarning struct {
	ID        string
	Archetype string
	// Range is the deepest the character's kit can be pointed, in ranks.
	Range int
}

func (w *ShortReachWarning) Error() string {
	return fmt.Sprintf("character %s is a %s whose kit reaches only rank %d: "+
		"it can do nothing until the enemy's front line is dead",
		w.ID, w.Archetype, w.Range)
}

func (w *ShortReachWarning) warning() {}

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
	ID string
	// Stage is the form whose art is missing, empty when it is the character's
	// own. It is on the problem rather than folded into the sentence because
	// cmd/hexforge-tui says these in the author's language, and "which stage"
	// is a fact rather than a phrase.
	Stage string
	Image string
	Path  string
}

func (p *MissingArtProblem) Error() string {
	if p.Stage != "" {
		return fmt.Sprintf("character %s names the art %s for its %s stage, which is not at %s",
			p.ID, p.Image, p.Stage, p.Path)
	}
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

// ArtMissing is how many of the character's pictures are not on disk.
func (r CharacterReport) ArtMissing() int {
	missing := 0
	for _, art := range r.Art {
		if !art.Exists {
			missing++
		}
	}
	return missing
}

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
		row := CharacterReport{ID: character.ID}
		for _, art := range character.Art() {
			exists := l.ImageExists(art.Image)
			row.Art = append(row.Art, ArtReport{
				Stage: art.Stage, Image: art.Image, Exists: exists,
			})
			if !exists {
				report.Problems = append(report.Problems, &MissingArtProblem{
					ID: character.ID, Stage: art.Stage, Image: art.Image,
					Path: l.ImagePath(art.Image),
				})
			}
		}
		// Both ends of the line are resolved: the first level a character can
		// exist at and the last, which is where the stat budget bites.
		if _, _, err := character.Resolve(1, progression.Furthest); err != nil {
			row.Failure = err
		}
		values, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
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
		if row.Failure == nil {
			row.Traits = l.heldBudgets(character, stage.Name, values)
		}
		if short := l.shortReach(character); short != nil {
			report.Warnings = append(report.Warnings, short)
		}
		report.Rows = append(report.Rows, row)
	}
	return report
}

// heldBudgets is one row per trait the character can reach at the cap.
//
// A trait it cannot reach is left out rather than shown at nought: the question
// is what this character can be fielded as, and a trait no placement of it could
// name is not one of the answers.
func (l *Library) heldBudgets(character cast.Character, stage string,
	values progression.Values) []TraitBudget {
	var out []TraitBudget
	for _, trait := range character.PassivesAt(progression.LevelCap, stage) {
		held, err := l.Held(values, []string{trait})
		if err != nil {
			continue
		}
		out = append(out, TraitBudget{Trait: trait, Values: held, Budget: l.Budget(held)})
	}
	return out
}

// shortReach measures a character's kit against the column its archetype puts it
// in, and reports the one case that cannot act alone.
//
// The archetype is where the column comes from because that is the only place a
// character says anything about where it stands: a roster slot is a placement
// and belongs to a battle, while an archetype is the role, and a role is exactly
// the claim "this belongs on the front" or "this belongs at the back".
//
// A character whose archetype has gone missing, or whose kit holds nothing this
// library can look up, is not warned about: the first is a problem the parser
// already refuses and the second would be a warning about nothing.
func (l *Library) shortReach(character cast.Character) *ShortReachWarning {
	preset, known := l.archetypes.Get(character.Archetype)
	if !known {
		return nil
	}
	deepest, counted := 0, 0
	for _, entry := range character.Skills {
		carried, err := l.skills.Lookup(entry.ID)
		if err != nil {
			continue
		}
		if carried.Target != skill.Enemy {
			continue
		}
		counted++
		if carried.Range > deepest {
			deepest = carried.Range
		}
	}
	// A kit with nothing aimed at an enemy is a different fault and is not this
	// one: a healer that never attacks is not short of reach.
	if counted == 0 || deepest > 1 {
		return nil
	}
	return &ShortReachWarning{ID: character.ID, Archetype: preset.ID, Range: deepest}
}
