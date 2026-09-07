package forge

import (
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// A shape's name says nothing about which cells it covers. "pierce" and
// "wedge_left" are the authored names of two step chains, and an author choosing
// between them from a chooser is choosing between two words.
//
// So a front-end draws the shape, and what it draws is pattern.Pattern.Targets —
// the same function battle.Act walks to decide who is hit. This file is the half
// of that a screen must not decide for itself: which cell the illustration is
// aimed at, and what the shape catches from there. The glyphs are the screen's.

// ShapeDiagramCell is the cell a shape is illustrated from.
//
// It is the middle of the enemy formation, and it is the only cell on that half
// where all six of the one-step directions stay on the board *and* on one side —
// which matters because Targets drops a cell that does either, so any other
// primary would draw some shape smaller than it is and blame the shape for the
// aim. Eight of the nine shipped shapes therefore draw in full.
//
// The ninth is "pierce", whose second upper-right step leaves the board from
// here, so it draws two cells of its three. That is not a rendering artefact and
// must not be smoothed over: a two-step chain aimed near an edge really does
// lose its far cell, and Covered against Max is what says so on screen.
//
// The alternative was the enemy frontline, {3, 1}, which draws pierce in full
// and collapses "wedge_left" to the primary alone, because both of its cells sit
// behind the frontline on the caster's own half. That is worse: a shape that
// draws identically to "single" reads as a shape that does nothing, where a
// shape drawing two cells of three reads as what it is. No cell draws all nine
// in full — pierce needs a primary at column 3 and wedge_left needs one at
// column 4 or beyond — so this is a choice between which one is short, not a
// choice about whether any is.
//
// One drawing serves a skill aimed either way. The ally half's own middle,
// {1, 1}, is this cell under hex.Place's 180 degree rotation and has the same
// property, and every shipped shape covers the same number of cells from it — so
// an ally-aimed skill's shape is this picture mirrored, not a different picture.
// That is a consequence of the rotation existing rather than a coincidence, and
// TestTheShapeDiagramCellShowsTheMostOfEveryShape measures it, because it is the
// reason this is one cell and not one per side.
func ShapeDiagramCell() hex.Offset { return hex.Offset{Col: 4, Row: 1} }

// ShapeCoverage is what a shape catches from the diagram's cell: values, for a
// screen to draw.
//
// Splash is separate from Primary rather than being the tail of one list,
// because the two are not worth the same — the primary takes a skill's whole
// power and a splash cell takes the pattern book's share of it — and a renderer
// that has to know which end of a slice is which would be a renderer deciding
// it.
type ShapeCoverage struct {
	Shape string
	// AimedAt is the side the skill carrying this shape aims at, because it
	// changes what the shape covers: a skill aimed at both halves spreads across
	// the midline where every other one stops at it.
	AimedAt skill.Side
	Primary hex.Offset
	Splash  []hex.Offset
	// Max is how many cells the shape covers when nothing is dropped, so a
	// caller can say that this aim loses one rather than only how many it got.
	Max int
}

// Covered is how many cells the shape really catches from here.
func (c ShapeCoverage) Covered() int { return 1 + len(c.Splash) }

// Whole reports whether the shape draws in full from this cell.
func (c ShapeCoverage) Whole() bool { return c.Covered() == c.Max }

// ShapeCoverage resolves a shape and a targeting side by name and walks the
// shape from the diagram's cell.
//
// The answers arrive as the strings a chooser produced, like every other answer
// in this package, so a screen hands over its draft rather than parsing on its
// own behalf.
//
// The walk is pattern's own and nothing else, so what a screen draws is what a
// battle would hit — including the midline, which a skill aimed at both halves
// crosses and every other one stops at. Which of the two walks that is comes
// from skill.Side.CrossesSides, the same declaration battle uses, so a diagram
// cannot promise a cell the resolution would drop. Targets returns the primary
// first and that is the contract this relies on — see its doc comment.
func (l *Library) ShapeCoverage(shape, target string) (ShapeCoverage, error) {
	found, err := l.LookupPattern(shape)
	if err != nil {
		return ShapeCoverage{}, err
	}
	aimedAt, err := ParseTarget(target)
	if err != nil {
		return ShapeCoverage{}, err
	}
	primary := ShapeDiagramCell()
	cells := found.Targets(primary)
	if aimedAt.CrossesSides() {
		cells = found.TargetsAcross(primary)
	}
	coverage := ShapeCoverage{
		Shape: found.Name, AimedAt: aimedAt, Primary: primary, Max: found.MaxTargets(),
	}
	if len(cells) > 0 {
		coverage.Splash = append(coverage.Splash, cells[1:]...)
	}
	return coverage, nil
}

// AimCoverage is what a skill catches when it is aimed at one cell of a real
// board — the same values ShapeCoverage carries, resolved from the aim rather
// than from the diagram's fixed cell.
//
// ⚠️ **It is a second entry point and not a second walk.** ShapeCoverage answers
// "what is this *shape*", which is a question about a shape and is therefore
// asked from one chosen cell; this answers "what does this *aim* catch", which is
// a question about a board and can only be asked from the cell under the cursor.
// A screen drawing the diagram's footprint over a battle would be drawing a
// picture of a different board — the shape loses its far cell near an edge, and
// which edge depends entirely on where the player is pointing.
//
// The side comes off the skill rather than being passed in, because the skill is
// what decides it: battle.covers reads known.Target.CrossesSides() and so does
// this, so the cells drawn are the cells the resolution would walk. Same
// predicate, three callers, one declaration — skill.Side.CrossesSides.
//
// ⚠️ It answers about CELLS and not about damage. A cell in the result may hold
// a unit the resolution then refuses — a burrowed target takes nothing off a
// splash it is standing in — so a caller may say "this reaches these cells" and
// may not say "these will be hit".
func (l *Library) AimCoverage(id string, aim hex.Offset) (ShapeCoverage, error) {
	declared, err := l.skills.Lookup(id)
	if err != nil {
		return ShapeCoverage{}, err
	}
	shape, err := l.LookupPattern(declared.Pattern)
	if err != nil {
		return ShapeCoverage{}, err
	}
	cells := shape.Targets(aim)
	if declared.Target.CrossesSides() {
		cells = shape.TargetsAcross(aim)
	}
	coverage := ShapeCoverage{
		Shape: shape.Name, AimedAt: declared.Target, Primary: aim, Max: shape.MaxTargets(),
	}
	if len(cells) > 0 {
		coverage.Splash = append(coverage.Splash, cells[1:]...)
	}
	return coverage, nil
}

// SplashShare is the share of a skill's power a splash cell takes, as a reader
// thinks of it. The primary always takes the whole.
func (l *Library) SplashShare() string { return Percent(l.patterns.SplashPower) }
