package screen

import (
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The four marks an aim row can carry, and they are TEXT rather than a colour.
//
// The goldens in this package are taken under NO_COLOR, so Palette.Plain is true
// and every style renders as the identity — a matchup drawn in green and red
// alone would be a feature the design record cannot see, and one that could
// break or vanish with every test in the repository still green. Colour is the
// reinforcement, applied over these; the marks are the answer.
//
// ASCII, for the reason shapeSplashMark is: a glyph that measures one cell and
// draws two puts every row after it out by a column on somebody else's terminal,
// and this repository has been bitten by that. They are also deliberately not
// the two marks the shape diagram already uses — "##" is the cell a skill is
// aimed at and ".." is a cell that only catches the splash share, both of which
// appear on the very rows these sit at the end of.
//
// Two levels each way and not one, because a defender with two elements composes
// its two multipliers: a blow both halves are weak to lands at a little over
// twice what a neutral one does, and one both halves resist at a little under a
// half. Those are four genuinely different decisions to take with a turn, so
// collapsing them to "weak" and "resists" would throw away the half of the
// answer that a dual-element target is the whole reason to ask.
const (
	matchupVeryWeakMark   = "++"
	matchupWeakMark       = "+"
	matchupResistMark     = "-"
	matchupVeryResistMark = "--"
)

// matchupMark is the mark a defender's affinity earns against one skill, and the
// empty string when there is nothing worth saying.
//
// ⚠️ **The thresholds are read off the CHART and not written down here.** The
// shipped chart is advantage 1500, neutral 1000, disadvantage 667, so the five
// values a composition can produce are 444, 667, 1000, 1500 and 2250 — but those
// are balance numbers that live in elements.json, and a screen holding a second
// copy of them is a screen that draws the wrong mark the day somebody retunes
// the chart. What is asked instead is structural: past a single advantage is
// doubly weak, above neutral is weak, and the two mirrors of that below.
//
// There is no fifth mark for an immunity because the chart has no zero: inert
// means the element is in no cycle and no pair, which is the absence of a
// matchup rather than a multiplier of nought, so an inert defender reads as
// neutral and draws nothing. A mark for a state the data cannot reach would be a
// mark nobody could ever check.
func matchupMark(chart *element.Chart, attacker element.Element, defender element.Affinity) string {
	if chart == nil {
		return ""
	}
	levels := chart.Multipliers()
	switch multiplier := chart.MultiplierAgainst(attacker, defender); {
	case multiplier > levels.Advantage:
		return matchupVeryWeakMark
	case multiplier > levels.Neutral:
		return matchupWeakMark
	case multiplier < levels.Disadvantage:
		return matchupVeryResistMark
	case multiplier < levels.Neutral:
		return matchupResistMark
	}
	return ""
}

// matchupOn is the mark for whoever is standing on a cell, styled.
//
// ⚠️ **A skill with no power draws nothing, whoever is standing there.** Forty-six
// of the hundred and fifty-one shipped skills deal no damage and still declare an
// element — a sleep powder is grass — so a matchup on one is a number with no
// quantity behind it for it to be a share of. Twenty-two of those aim at the
// caster and fourteen at an ally, where the same mark would not merely be
// meaningless but wrong: it would read as a warning about a skill that is help.
//
// The mark is a fact about the BLOW and not about the player's fortune, which is
// what decides the colour on the two shipped all-sided skills — discharge and
// hail catch both halves of the board, so a "+" can land on the caster's own
// side. Style.Good there says this blow lands harder on that unit, which is what
// it says everywhere else and what the reader needs to know before spending the
// turn. Under NO_COLOR the two cases are the same row, which is the case the
// design record holds.
func (p PlayScreen) matchupOn(c Context, declared skill.Skill, read playReading, cell hex.Offset) string {
	if declared.Power == 0 {
		return ""
	}
	unit, standing := read.standing(cell)
	if !standing {
		return ""
	}
	mark := matchupMark(c.Lib.Chart(), declared.Element, unit.Affinity)
	switch mark {
	case matchupVeryWeakMark, matchupWeakMark:
		return c.Style.Good.Render(mark)
	case matchupResistMark, matchupVeryResistMark:
		return c.Style.Bad.Render(mark)
	}
	return ""
}

// matchupLegend is what the marks mean, in the reader's own language.
//
// It is drawn where the marks are and only there, which is the rule the splash
// legend already follows on the same heading: a line explaining a mark the screen
// is not currently drawing is a line spent on nothing, and on a screen that
// cannot fit a five-a-side pairing at its own floor there is no such line to
// spend.
func matchupLegend(c Context) string {
	return c.Text(i18n.PlayAimMatchup, matchupVeryWeakMark, matchupWeakMark,
		matchupResistMark, matchupVeryResistMark)
}
