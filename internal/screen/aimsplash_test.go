package screen

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The aim list printed one cell and its occupant, and the summary beside the
// skill printed a count — "3 ô". Neither said WHICH cells an area skill was about
// to reach, so the one thing worth knowing before spending a turn was the one
// thing a player had to hold in their head.
//
// ⚠️ **The fixture battle cannot see this and neither can any golden.** The cast
// that fixture picks brings four `single` skills, so every existing test drives a
// screen on which the new rows are correctly absent. Twenty-one area skills ship
// in builds.json, so the feature is reachable in a real battle and invisible to
// the suite as it stood — which is the blindness memory/hexarena-a-golden-cannot
// -hold-a-keystroke-rule.md is about, arriving through the other door. Everything
// below therefore builds its own battle out of a character chosen BY PROPERTY.

// anAreaCaster is a shipped character whose furthest kit carries a skill covering
// more than one cell.
//
// By property and not by name, which is the reading #328 gave the battle fixture:
// a named character is a fixture that goes stale the day its learnset is
// rebalanced, and it goes stale silently, because a kit of `single` skills draws
// a correct screen with nothing on it.
func anAreaCaster(t *testing.T, c Context) cast.Character {
	t.Helper()
	characters := c.Lib.Characters().All()
	for _, character := range characters {
		for _, id := range character.SkillsAt(progression.LevelCap, progression.Furthest) {
			declared, err := c.Lib.Skills().Lookup(id)
			if err != nil {
				continue
			}
			shape, err := c.Lib.LookupPattern(declared.Pattern)
			if err != nil {
				continue
			}
			if shape.MaxTargets() > 1 {
				return character
			}
		}
	}
	t.Fatal("no shipped character carries a skill covering more than one cell, so the " +
		"splash rows below are drawn for a state the game cannot reach")
	return cast.Character{}
}

// anAreaSquad is aSquadOfSide with every seat filled by that character, so
// whichever unit the turn order asks is one that can cast an area skill.
func anAreaSquad(t *testing.T, c Context, side int) placement.Squad {
	t.Helper()
	character := anAreaCaster(t, c)
	units := make([]placement.Placement, 0, side)
	for _, slot := range squadSlots(side) {
		unit := placement.Placement{
			ID:        slot.String(),
			Character: character.ID,
			Level:     progression.LevelCap,
			Slot:      slot,
		}
		kit := character.SkillsAt(unit.Level, progression.Furthest)
		if len(kit) > cast.SkillSlots {
			kit = kit[:cast.SkillSlots]
		}
		unit.Skills = kit
		if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
			unit.Passives = traits[:cast.TraitSlots]
		}
		units = append(units, unit)
	}
	return placement.Squad{
		ID: "do-lan-" + strconv.Itoa(side), Name: "đội lan", Units: units,
	}
}

// aimingAtAnArea is that battle with the cursor parked on an option and an aim
// whose shape really does catch a second cell from there.
//
// The search is over both lists rather than over the options alone, because a
// shape near an edge loses cells: `pierce` aimed at the far column catches one
// cell and is not a defect. Picking the first option and hoping is how a test
// starts passing on a screen with nothing on it.
func aimingAtAnArea(t *testing.T, c Context) (PlayScreen, []hex.Offset) {
	t.Helper()
	return aimingAtAnAreaFrom(t, c, anAreaBattle(t, c))
}

// anAreaBattle is the board those squads make, opened and waiting on a turn.
func anAreaBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	squad := anAreaSquad(t, c, 3)
	p := NewPlayScreen().Open(c, squad, squad.Clone())
	if p.Err != nil {
		t.Fatalf("the area battle would not start: %v", p.Err)
	}
	if p.Pending == nil {
		t.Fatal("the area battle opened without a turn for the player")
	}
	return p
}

// aimingAtAnAreaFrom parks the cursor on a splash-catching aim of a battle the
// caller has already driven, which is what lets the golden take the same state
// with a log behind it.
func aimingAtAnAreaFrom(t *testing.T, c Context, p PlayScreen) (PlayScreen, []hex.Offset) {
	t.Helper()
	if p.Pending == nil {
		t.Fatal("the area battle is between turns, so there is no aim list to park on")
	}
	for option := range p.Pending.Options {
		for aim := range p.Pending.Options[option].Aims {
			probe := p
			probe.Option, probe.Aim, probe.Aiming = option, aim, true
			if splash := probe.splashUnder(c, probe.Pending.Options[option]); len(splash) > 0 {
				return probe, splash
			}
		}
	}
	t.Fatalf("no option this turn offers catches a second cell from any of its aims, so "+
		"the rows below are measured on a board that has none: unit %s", p.Pending.Unit)
	return p, nil
}

// TestTheAimListNamesTheCellsAnAreaSkillAlsoCatches is the whole item.
func TestTheAimListNamesTheCellsAnAreaSkillAlsoCatches(t *testing.T) {
	c, _ := start(t, i18n.En)
	p, splash := aimingAtAnArea(t, c)
	drawn := p.Choices(c)

	for _, cell := range splash {
		if !strings.Contains(drawn, cell.String()) {
			t.Errorf("the aim list does not name %s, which the shape catches:\n%s",
				cell, drawn)
		}
		// And whoever is standing there, which is the half a count could never
		// carry: "3 ô" is three cells and says nothing about whether any of them
		// holds an ally.
		if held := p.occupant(p.read(), cell); held != "" && !strings.Contains(drawn, held) {
			t.Errorf("the aim list names the cell %s but not %q standing in it:\n%s",
				cell, held, drawn)
		}
	}
	// The mark is explained where it is used, and only there.
	legend := c.Text(i18n.PlayAimSplash, shapeSplashMark, c.Lib.SplashShare())
	if !strings.Contains(drawn, legend) {
		t.Errorf("the aim list draws splash rows and does not say what %q means:\n%s",
			shapeSplashMark, drawn)
	}
}

// TestTheSplashIsResolvedFromTheAimAndNotFromTheDiagram is the trap the TODO
// entry named: the drawing already existed on the authoring screen and could not
// be reused, because it walks from forge.ShapeDiagramCell — a fixed cell chosen
// so most shapes draw in full. A battle is asking about this aim on this board.
func TestTheSplashIsResolvedFromTheAimAndNotFromTheDiagram(t *testing.T) {
	c, _ := start(t, i18n.En)
	p, splash := aimingAtAnArea(t, c)
	option := p.Pending.Options[p.Option]
	aim := option.Aims[p.Aim]
	if slices.Contains(splash, aim) {
		t.Errorf("the aim %s is drawn as a splash cell of itself: %v", aim, splash)
	}

	// The diagram's answer, asked exactly as the authoring screen asks it — the
	// shape by name and the skill's own target side — and it is a DIFFERENT set
	// of cells, which is what makes reusing it wrong rather than merely indirect.
	declared, err := c.Lib.Skills().Lookup(option.Skill)
	if err != nil {
		t.Fatalf("the shipped skill %q is not in the book: %v", option.Skill, err)
	}
	if aim == forge.ShapeDiagramCell() {
		t.Fatalf("the cursor is on the diagram's own cell %s, so the two answers agree "+
			"by accident and this comparison measures nothing", aim)
	}
	diagram, err := c.Lib.ShapeCoverage(declared.Pattern, declared.Target.String())
	if err != nil {
		t.Fatalf("the shipped shape %q has no diagram: %v", declared.Pattern, err)
	}
	if slices.Equal(diagram.Splash, splash) {
		t.Errorf("the aim list draws %v, which is exactly what the diagram draws from %s: "+
			"the footprint is not being resolved from the cell under the cursor",
			splash, diagram.Primary)
	}

	// And it follows the cursor. Two aims of one option are two different
	// footprints, which is the property a fixed cell cannot have and the reason
	// this is a second entry point rather than a second caller of the first.
	var moved bool
	for other := range option.Aims {
		if other == p.Aim {
			continue
		}
		elsewhere := p
		elsewhere.Aim = other
		if !slices.Equal(elsewhere.splashUnder(c, option), splash) {
			moved = true
			break
		}
	}
	if len(option.Aims) > 1 && !moved {
		t.Errorf("%s catches the same cells %v from every one of its %d aims, so the rows "+
			"are not reading the cursor", option.Skill, splash, len(option.Aims))
	}
}

// TestASingleCellShapeDrawsNoSplashRows is the other half, and it is what stops
// the rows becoming noise on the ninety per cent of turns that have none.
func TestASingleCellShapeDrawsNoSplashRows(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := atABattleOf(t, c, 3)
	p.Aiming = true
	var measured int
	for index, option := range p.Pending.Options {
		coverage, err := c.Lib.AimCoverage(option.Skill, option.Aims[0])
		if err != nil || coverage.Max > 1 {
			continue
		}
		measured++
		probe := p
		probe.Option, probe.Aim = index, 0
		drawn := probe.Choices(c)
		if strings.Contains(drawn, shapeSplashMark) {
			t.Errorf("%s catches one cell and the aim list draws a splash mark:\n%s",
				option.Skill, drawn)
		}
		if strings.Contains(drawn, c.Text(i18n.PlayAimSplash,
			shapeSplashMark, c.Lib.SplashShare())) {
			t.Errorf("%s catches one cell and the aim list explains a mark it never draws",
				option.Skill)
		}
	}
	if measured == 0 {
		t.Fatal("no option this turn covers a single cell, so the quiet case was not reached")
	}
}
