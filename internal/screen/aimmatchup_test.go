package screen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The aim list says the matchup
//
// The aim list drew a cell and whoever was standing on it, so the one fact that
// decides who to hit first — whether this element lands hard or is shrugged off
// — was the fact a player had to hold in their head against a chart on another
// screen.
//
// ⚠️ **Two things about what can measure this, and both are traps this package
// has already fallen into once each.**
//
// The first is the palette. Every golden and every test here runs under
// NO_COLOR, so Palette.Plain is true and Style.Good and Style.Bad render as the
// identity — a matchup encoded in colour alone would be invisible to the design
// record and to every assertion in the repository, and could break or vanish
// with the whole suite still green. So the mark is text and
// TestTheMatchupMarkSurvivesNoColour is the guard that says so.
//
// The second is the fixture. The cast battleCast picks opens on a neutral-element
// unit throwing neutral skills at a neutral, a grass and a ground defender, so
// **every row of the two aiming goldens is a 1000 and draws nothing** — a correct
// screen with the feature entirely absent from it, which is exactly the blindness
// memory/hexarena-a-golden-cannot-hold-a-keystroke-rule.md is about. Everything
// below therefore builds its own board out of characters chosen BY PROPERTY:
// whichever pair in the library actually produces the mark being asked about.

// The five values a composed multiplier can take against the shipped chart, and
// which of them the data reaches.
//
// ⚠️ **Written down here as a premise the tests below check rather than as a
// table they trust.** The chart is advantage 1500, neutral 1000, disadvantage
// 667 and MultiplierAgainst composes a dual defender's two lookups, so the set
// is {444, 667, 1000, 1500, 2250} — 444 and 2250 need a defender with two
// elements on the same side of the matchup, and only two shipped characters
// declare two elements at all. A test that asserted five states without
// checking the data can field them would be a test measuring a design document.
var everyMatchupMark = []string{
	matchupVeryResistMark, matchupResistMark, matchupWeakMark, matchupVeryWeakMark,
}

// TestTheChartReachesEveryMarkAndNoOthers is the premise, taken off the chart
// rather than off the constants above.
//
// It sweeps every attacker element against every affinity the type system allows
// — singles and every legal pair — and reports which marks come out. Five values
// and four marks is the claim the drawing rests on, and the arithmetic that
// produces it lives in a data file: a chart retuned to give the inert element a
// matchup, or given a fourth level, changes this and the screen has to know.
func TestTheChartReachesEveryMarkAndNoOthers(t *testing.T) {
	_, lib := start(t, i18n.En)
	chart := lib.Chart()
	seen := map[string]int{}
	values := map[int]bool{}
	for _, attacker := range element.All() {
		for _, primary := range element.All() {
			single, err := element.Single(primary)
			if err != nil {
				t.Fatalf("Single(%s): %v", primary, err)
			}
			affinities := []element.Affinity{single}
			for _, secondary := range element.All() {
				if pair, err := element.Dual(primary, secondary); err == nil {
					affinities = append(affinities, pair)
				}
			}
			for _, affinity := range affinities {
				values[chart.MultiplierAgainst(attacker, affinity)] = true
				seen[matchupMark(chart, attacker, affinity)]++
			}
		}
	}
	if len(values) != 5 {
		t.Errorf("the chart composes %d distinct multipliers %v, and the four marks plus "+
			"nothing were chosen for five", len(values), values)
	}
	// No zero anywhere, which is the whole of why there is no immunity mark. An
	// inert element is one in no cycle and no pair — the absence of a matchup —
	// and it reads as neutral, not as nothing getting through.
	if values[0] {
		t.Error("some pairing composes to a multiplier of nought, so a target can be " +
			"immune and the aim list has no mark that says so")
	}
	for _, mark := range append([]string{""}, everyMatchupMark...) {
		if seen[mark] == 0 {
			t.Errorf("no attacker and affinity in the whole chart draws %q, so that mark "+
				"is a state nothing can reach", mark)
		}
	}
	if len(seen) != len(everyMatchupMark)+1 {
		t.Errorf("the chart draws %d different marks and %d were declared (plus nothing)",
			len(seen), len(everyMatchupMark))
	}
	t.Logf("multipliers %v, marks %v", values, seen)
}

// aMarkedAim is a board on which the unit being asked carries one damaging skill
// that draws the wanted mark against the other side, with the aim list open and
// the cursor on the row that carries it.
//
// ⚠️ **Chosen by property, and the search is over attacker, skill and defender
// together.** Naming pokemon.magnemite and pokemon.lapras would work today and
// would go stale silently the day either is re-elemented — silently, because a
// pair whose matchup has become neutral draws a correct aim list with nothing on
// it, which is the same blindness the fixture cast already has. The search
// instead asks the library for whatever pair produces the mark, and the caller
// fails when there is none.
//
// The kit is the one skill rather than the character's first four, for two
// reasons. A placement takes only cast.SkillSlots and the skill wanted here is
// often not among the first of them; and a one-option turn is a list whose rows
// are all about the skill under test, so an assertion cannot pass by matching a
// mark some neighbouring option earned.
func aMarkedAim(t *testing.T, c Context, want string) (PlayScreen, string) {
	t.Helper()
	found := matchupTriples(c, want, func(skill.Skill) bool { return true })
	if len(found) == 0 {
		t.Fatalf("nothing in the library draws %q: no character's kit holds a damaging "+
			"enemy-aimed skill whose element earns that mark against any affinity the "+
			"cast declares, so the state is not reachable and asserting it would measure "+
			"nothing", want)
	}
	triple := found[0]
	p := aBoardOf(t, c, triple.attacker, triple.chosen, triple.defender, 1)
	option, index, ok := optionNamed(p, triple.chosen)
	if !ok {
		t.Fatalf("the turn in front does not offer %s, which is the only skill its unit "+
			"was given", triple.chosen)
	}
	p.Option, p.Aiming = index, true
	aim, standing := aimStandingOn(p, option, triple.defender)
	if !standing {
		t.Fatalf("%s can be pointed at %d cells and none of them holds the other side",
			triple.chosen, len(option.Aims))
	}
	p.Aim = aim
	return p, triple.chosen
}

// matchupTriple is an attacker, one damaging enemy-aimed skill out of its kit at
// the cap, and a defender whose affinity earns the wanted mark against it.
type matchupTriple struct {
	attacker cast.Character
	chosen   string
	defender cast.Character
}

// matchupTriples is every such combination the library holds, in cast order.
//
// ⚠️ **Every one of them and not the first.** A caller that needs a board with a
// particular shape on it — a splash cell with somebody standing on it, say —
// cannot tell from the triple alone whether it will get one, and the first
// candidate is chosen by nothing but where it happens to sit in the cast file.
// Handing back the whole list lets the caller keep asking, and lets its failure
// say "none of the N candidates" rather than "the one I tried".
func matchupTriples(c Context, want string, wanted func(skill.Skill) bool) []matchupTriple {
	chart := c.Lib.Chart()
	characters := c.Lib.Characters().All()
	var out []matchupTriple
	for _, candidate := range characters {
		for _, id := range candidate.SkillsAt(progression.LevelCap, progression.Furthest) {
			declared, err := c.Lib.Skills().Lookup(id)
			if err != nil || declared.Power == 0 || declared.Target.String() != "enemy" {
				continue
			}
			if !wanted(declared) {
				continue
			}
			for _, target := range characters {
				if matchupMark(chart, declared.Element, target.Element) == want {
					out = append(out, matchupTriple{candidate, id, target})
				}
			}
		}
	}
	return out
}

// aBoardOf opens a battle with one attacker carrying one named skill against a
// side of the given size, all of it the defender.
func aBoardOf(t *testing.T, c Context, attacker cast.Character, chosen string,
	defender cast.Character, side int) PlayScreen {
	t.Helper()
	facing := make([]cast.Character, 0, side)
	for range side {
		facing = append(facing, defender)
	}
	return aBoardFacing(t, c, attacker, chosen, facing)
}

// aBoardFacing is the same board with the other side named seat by seat, which
// is what lets one aim list hold more than one matchup.
func aBoardFacing(t *testing.T, c Context, attacker cast.Character, chosen string,
	facing []cast.Character) PlayScreen {
	t.Helper()
	side := len(facing)
	// Both sides the same size, so the attacking half is not simply outnumbered
	// and killed before its first turn — which is what a lone unit against three
	// at the cap does, and it presents as "opened without a turn for the player"
	// rather than as anything about a matchup.
	home := placement.Squad{ID: "do-cong", Name: "đội công"}
	for _, slot := range squadSlots(side) {
		home.Units = append(home.Units, placement.Placement{
			ID:        "c" + slot.String(),
			Character: attacker.ID,
			Level:     progression.LevelCap,
			Slot:      slot,
			Skills:    []string{chosen},
		})
	}
	away := placement.Squad{ID: "do-thu", Name: "đội thủ"}
	for index, slot := range squadSlots(side) {
		defender := facing[index]
		kit := defender.SkillsAt(progression.LevelCap, progression.Furthest)
		if len(kit) > cast.SkillSlots {
			kit = kit[:cast.SkillSlots]
		}
		away.Units = append(away.Units, placement.Placement{
			ID:        "d" + slot.String(),
			Character: defender.ID,
			Level:     progression.LevelCap,
			Slot:      slot,
			Skills:    kit,
		})
	}
	p := NewPlayScreen().Open(c, home, away)
	if p.Err != nil {
		t.Fatalf("%s against %d would not start: %v", attacker.ID, side, p.Err)
	}
	if p.Pending == nil {
		t.Fatalf("%s against %d opened without a turn for the player", attacker.ID, side)
	}
	return p
}

// aMixedAimList is a board whose one aim list draws several different marks at
// once, which is the picture worth recording: a reader deciding who to hit first
// is comparing rows, and a list where every row says the same thing is the one
// shape that cannot show the comparison.
//
// The other half is named seat by seat and chosen by property — the widest set
// of distinct answers any one skill in the library can produce against the cast
// — so the record is of the drawing rather than of a pairing somebody picked.
func aMixedAimList(t *testing.T, c Context, seats int) (PlayScreen, string, []string) {
	t.Helper()
	chart := c.Lib.Chart()
	characters := c.Lib.Characters().All()
	var bestAttacker cast.Character
	var bestSkill string
	var bestFacing []cast.Character
	var bestMarks []string
	for _, attacker := range characters {
		for _, id := range attacker.SkillsAt(progression.LevelCap, progression.Furthest) {
			declared, err := c.Lib.Skills().Lookup(id)
			if err != nil || declared.Power == 0 || declared.Target.String() != "enemy" {
				continue
			}
			var facing []cast.Character
			var marks []string
			taken := map[string]bool{}
			// One seat per distinct answer, hardest first, so a short side
			// spends its seats on the marks rather than on repeats.
			for _, want := range append(append([]string{}, everyMatchupMark...), "") {
				if len(facing) == seats {
					break
				}
				for _, target := range characters {
					if taken[want] || matchupMark(chart, declared.Element, target.Element) != want {
						continue
					}
					facing = append(facing, target)
					marks = append(marks, want)
					taken[want] = true
				}
			}
			if len(marks) > len(bestMarks) {
				bestAttacker, bestSkill, bestFacing, bestMarks = attacker, id, facing, marks
			}
		}
	}
	if len(bestMarks) < 2 {
		t.Fatalf("no skill in the library draws more than one answer against the cast, so "+
			"an aim list cannot compare two targets: best was %v", bestMarks)
	}
	p := aBoardFacing(t, c, bestAttacker, bestSkill, bestFacing)
	_, index, ok := optionNamed(p, bestSkill)
	if !ok {
		t.Fatalf("the turn in front does not offer %s", bestSkill)
	}
	p.Option, p.Aim, p.Aiming = index, 0, true
	return p, bestSkill, bestMarks
}

// optionNamed is the option for a skill id, and where it sits in the list.
func optionNamed(p PlayScreen, id string) (battle.Option, int, bool) {
	for index, option := range p.Pending.Options {
		if option.Skill == id {
			return option, index, true
		}
	}
	return battle.Option{}, 0, false
}

// aimStandingOn is the index of the first aim a unit of the given character is
// standing on.
func aimStandingOn(p PlayScreen, option battle.Option, defender cast.Character) (int, bool) {
	read := p.read()
	for index, aim := range option.Aims {
		if unit, standing := read.standing(aim); standing && unit.Affinity == defender.Element {
			return index, true
		}
	}
	return 0, false
}

// TestTheMatchupMarkSurvivesNoColour is the guard the whole shape of this
// feature was chosen for.
//
// ⚠️ **It asserts on the PLAIN rendering.** The goldens are taken under NO_COLOR
// and so is this, so a mark encoded in Style.Good and Style.Bad alone renders as
// the empty difference and every assertion about it would pass on a screen with
// nothing on it. The palette is asked rather than assumed — a test claiming to
// measure the plain path has to check it got one — and then the mark is looked
// for as bare text.
func TestTheMatchupMarkSurvivesNoColour(t *testing.T) {
	c, _ := start(t, i18n.En)
	if !c.Style.Plain {
		t.Fatal("this machine's palette is not the plain one, so every style below " +
			"renders as itself and the claim being measured is not the claim made")
	}
	for _, want := range everyMatchupMark {
		p, chosen := aMarkedAim(t, c, want)
		drawn := p.Choices(c)
		if lipgloss.Width(drawn) != lipgloss.Width(stripped(drawn)) {
			t.Fatalf("%s: the aim list carries escape codes under NO_COLOR", chosen)
		}
		row, ok := markedRow(drawn, p)
		if !ok {
			t.Fatalf("%s: the aim list has no row under the cursor:\n%s", chosen, drawn)
		}
		if got := trailingMark(row); got != want {
			t.Errorf("%s: the row under the cursor is %q, and it should end in the mark "+
				"%q — a mark that lives in the colour alone is a mark this rendering "+
				"cannot carry", chosen, row, want)
		}
	}
}

// stripped is the text with every escape sequence taken out, which is what makes
// "the plain rendering really was plain" an assertion rather than a hope.
func stripped(text string) string {
	var out strings.Builder
	for at := 0; at < len(text); at++ {
		if text[at] != 0x1b {
			out.WriteByte(text[at])
			continue
		}
		for at < len(text) && text[at] != 'm' {
			at++
		}
	}
	return out.String()
}

// markedRow is the aim row the cursor is on, which is the one row every
// assertion here is about.
func markedRow(drawn string, p PlayScreen) (string, bool) {
	want := "> "
	for _, line := range strings.Split(drawn, "\n") {
		if strings.HasPrefix(line, want) && !strings.Contains(line, p.Pending.Options[p.Option].Skill) {
			return strings.TrimPrefix(line, want), true
		}
	}
	return "", false
}

// trailingMark is the last whitespace-separated field of a row when it is one of
// the declared marks, and the empty string otherwise.
//
// ⚠️ **The whole field and not a substring.** "+" is a substring of "++" and "-"
// of "--", so strings.Contains would answer yes to the wrong mark for two of the
// four and the distinctness claim below would be vacuous.
func trailingMark(row string) string {
	fields := strings.Fields(row)
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	for _, mark := range everyMatchupMark {
		if last == mark {
			return mark
		}
	}
	return ""
}

// TestEveryMatchupTheDesignDistinguishesIsDrawnAndDistinct is guard two.
//
// ⚠️ **Which of the five states SHIPPED data reaches is not what this asserts,
// and the difference is the point.** Measured against the library these tests
// load — the shipped books with the fixture cast appended — all four marks are
// reachable, because two shipped characters and two fixture ones declare two
// elements. Measured against the **six shipped squads**, only the middle three
// values arise (667, 1000, 1500): no squad in squads.json fields either dual
// character, so a player picking a shipped side never sees "++" or "--" at all.
// That is why the boards below are built out of the cast rather than out of a
// squad, and why the two extreme marks would be invisible to any test that
// played a shipped pairing.
func TestEveryMatchupTheDesignDistinguishesIsDrawnAndDistinct(t *testing.T) {
	c, _ := start(t, i18n.En)
	drawnFor := map[string]string{}
	for _, want := range everyMatchupMark {
		p, chosen := aMarkedAim(t, c, want)
		drawn := p.Choices(c)
		row, ok := markedRow(drawn, p)
		if !ok {
			t.Fatalf("%s: no row under the cursor:\n%s", chosen, drawn)
		}
		if got := trailingMark(row); got != want {
			t.Errorf("%s: want the row to end in %q, got %q in %q", chosen, want, got, row)
		}
		if earlier, taken := drawnFor[want]; taken {
			t.Errorf("%q is drawn by both %s and %s, which cannot happen", want, earlier, chosen)
		}
		drawnFor[want] = chosen
		// And the legend is drawn, because a mark the screen never explains is a
		// symbol the reader has to guess at — the rule the splash mark already
		// follows on the same heading.
		if !strings.Contains(drawn, matchupLegend(c)) {
			t.Errorf("%s: the aim list draws %q and does not say what it means:\n%s",
				chosen, want, drawn)
		}
	}
	// Four marks, four different rows. Asserted as a set rather than pairwise so
	// that a mapping which collapsed two states onto one mark is a failure here
	// rather than a silent loss of half the answer a dual defender is for.
	if len(drawnFor) != len(everyMatchupMark) {
		t.Errorf("%d marks were drawn for %d states: %v",
			len(drawnFor), len(everyMatchupMark), drawnFor)
	}
	t.Logf("drawn by: %v", drawnFor)
}

// TestANeutralMatchupDrawsNoMarkAndNoLegend is the quiet case, and it is most of
// them: eleven elements and a chart of three cycles leave the ordinary pairing
// at neutral, so a mark on every row would be noise on the turns that have
// nothing to say.
func TestANeutralMatchupDrawsNoMarkAndNoLegend(t *testing.T) {
	c, _ := start(t, i18n.En)
	// The fixture battle is the case: its unit throws neutral skills at three
	// defenders and every row of it composes to the neutral multiplier.
	p := atABattleOf(t, c, 3)
	p.Aiming = true
	var measured int
	for index, option := range p.Pending.Options {
		declared, err := c.Lib.Skills().Lookup(option.Skill)
		if err != nil {
			continue
		}
		probe := p
		probe.Option, probe.Aim = index, 0
		read := probe.read()
		var marked bool
		for _, aim := range option.Aims {
			if probe.matchupOn(c, declared, read, aim) != "" {
				marked = true
			}
		}
		if marked {
			continue
		}
		measured++
		drawn := probe.Choices(c)
		for _, row := range aimRowsOf(c, drawn, probe) {
			if mark := trailingMark(row); mark != "" {
				t.Errorf("%s reaches nobody's weakness and the row %q ends in %q",
					option.Skill, row, mark)
			}
		}
		if strings.Contains(drawn, matchupLegend(c)) {
			t.Errorf("%s draws no mark and the heading explains four of them:\n%s",
				option.Skill, drawn)
		}
	}
	if measured == 0 {
		t.Fatal("every option this turn marks somebody, so the quiet case was not reached")
	}
}

// aimRowsOf is every row of the aim list, which is every line after the heading
// that names the skill being pointed.
func aimRowsOf(c Context, drawn string, p PlayScreen) []string {
	heading := c.Text(i18n.PlayAimAt, p.Pending.Options[p.Option].Skill)
	lines := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
	var rows []string
	var past bool
	for _, line := range lines {
		if strings.Contains(line, heading) {
			past = true
			continue
		}
		if past && strings.TrimSpace(line) != "" {
			rows = append(rows, strings.TrimSpace(line))
		}
	}
	return rows
}

// TestASkillWithNoPowerDrawsNoMatchup is guard three, and the ally half of it is
// the case that matters.
//
// ⚠️ **Forty-six of the hundred and fifty-one shipped skills deal no damage and
// still declare an element** — twenty-two aimed at the caster, fourteen at an
// ally, ten at an enemy. A matchup on one of those is a share of a quantity that
// does not exist, and on the ally-aimed fourteen it is worse than meaningless:
// a "++" beside a squadmate reads as a warning about a skill that is help.
func TestASkillWithNoPowerDrawsNoMatchup(t *testing.T) {
	c, _ := start(t, i18n.En)
	chart := c.Lib.Chart()
	var measured, allies int
	for _, declared := range c.Lib.Skills().Skills() {
		if declared.Power != 0 {
			continue
		}
		for _, character := range c.Lib.Characters().All() {
			if matchupMark(chart, declared.Element, character.Element) == "" {
				// A skill whose element earns nothing against this affinity
				// cannot tell a working guard from a deleted one.
				continue
			}
			measured++
			if declared.Target.String() == "ally" {
				allies++
			}
			read := playReading{units: []playUnit{{
				ID: "x", Name: "x", Cell: squadSlots(1)[0], Affinity: character.Element,
			}}}
			if mark := (PlayScreen{}).matchupOn(c, declared, read, squadSlots(1)[0]); mark != "" {
				t.Errorf("%s deals no damage and draws %q against %s (%s)",
					declared.ID, mark, character.ID, character.Element)
			}
			break
		}
	}
	if measured == 0 {
		t.Fatal("no powerless skill in the book has an element that earns a mark against " +
			"any affinity the cast declares, so this guard is vacuous")
	}
	if allies == 0 {
		t.Fatal("none of the powerless skills measured aims at an ally, so the case this " +
			"guard exists for was not reached")
	}
	t.Logf("%d powerless skills measured, %d of them ally-aimed", measured, allies)
}

// TestTheSplashRowsCarryTheMatchupMark is guard four: a splash cell is a unit
// this cast is about to hit, so it is as much a target as the aim itself.
//
// ⚠️ **A board of its own, for the same reason aimsplash_test needs one.** The
// only skill on the fixture board that covers more than a cell is thrown by a
// neutral unit, so the splash rows it draws are all quiet — a screen that dropped
// the mark from them would go on drawing a correct picture.
func TestTheSplashRowsCarryTheMatchupMark(t *testing.T) {
	c, _ := start(t, i18n.En)
	var drew int
	carried := map[string]int{}
	for _, want := range everyMatchupMark {
		candidates := matchupTriples(c, want, func(declared skill.Skill) bool {
			shape, err := c.Lib.LookupPattern(declared.Pattern)
			return err == nil && shape.MaxTargets() > 1
		})
		if len(candidates) == 0 {
			t.Errorf("no area skill in the library draws %q, so the splash half of that "+
				"state cannot be measured at all", want)
			continue
		}
		// ⚠️ **Every candidate until one lands, not the first.** A shape catches
		// a second CELL, and a cell only carries a mark when somebody is standing
		// on it — near an edge, or against a side that leaves the caught cell
		// empty, a perfectly good area skill draws a splash row with no occupant
		// and nothing to mark. Taking the first candidate left two of the four
		// marks unmeasured while the test passed on the other two.
		var landed bool
		for _, triple := range candidates {
			// Five a side, for the same reason: the fuller the other half, the
			// more of a footprint falls on somebody.
			p := aBoardOf(t, c, triple.attacker, triple.chosen, triple.defender, hex.MaxSquadSize)
			option, index, ok := optionNamed(p, triple.chosen)
			if !ok {
				continue
			}
			p.Option, p.Aiming = index, true
			for aim := range option.Aims {
				probe := p
				probe.Aim = aim
				read := probe.read()
				if _, standing := read.standing(option.Aims[aim]); !standing {
					continue
				}
				splash := probe.splashUnder(c, read, option)
				drawn := probe.Choices(c)
				for _, caught := range splash {
					if _, standing := read.standing(caught); !standing {
						continue
					}
					row, found := splashRowFor(drawn, caught.String())
					if !found {
						t.Fatalf("%s: the aim list draws no splash row for %s:\n%s",
							triple.chosen, caught, drawn)
					}
					drew++
					carried[want]++
					landed = true
					if got := trailingMark(row); got != want {
						t.Errorf("%s: the splash row %q ends in %q, want %q — a splash "+
							"cell is a unit this cast is about to hit",
							triple.chosen, row, got, want)
					}
				}
				if landed {
					break
				}
			}
			if landed {
				break
			}
		}
		if !landed {
			t.Errorf("none of the %d area candidates for %q caught an occupied second "+
				"cell, so that state has no splash row anywhere", len(candidates), want)
		}
	}
	if drew == 0 {
		t.Fatal("no splash row holding a unit was drawn at all, so this measures nothing")
	}
	t.Logf("%d splash rows carried a mark: %v", drew, carried)
}

// splashRowFor is the splash row naming a cell, found by the mark the shape
// diagram and this list share.
func splashRowFor(drawn, cell string) (string, bool) {
	for _, line := range strings.Split(drawn, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, shapeSplashMark+" ") && strings.Contains(trimmed, cell) {
			return trimmed, true
		}
	}
	return "", false
}

// TestAMarkedAimListFitsTheFloor is the width half, because the mark lengthens
// every row it lands on and the legend lengthens the heading.
//
// The floor is the narrowest window any screen is drawn in, and it is what this
// package measures prose against everywhere else. Both languages, because a
// wording is only as short as its longest translation.
func TestAMarkedAimListFitsTheFloor(t *testing.T) {
	const drawable = MinWidth - 1
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		for _, want := range everyMatchupMark {
			p, chosen := aMarkedAim(t, c, want)
			for _, line := range strings.Split(strings.TrimRight(p.Choices(c), "\n"), "\n") {
				if width := lipgloss.Width(line); width > drawable {
					t.Errorf("%s/%s: a row of the aim list is %d cells over the %d it "+
						"has:\n%s", lang, chosen, width, drawable, line)
				}
			}
			t.Logf("%s/%s heading: %d cells", lang, chosen,
				lipgloss.Width(strings.Split(p.Choices(c), "\n")[len(
					p.Pending.Options)+2]))
		}
		// ⚠️ **The widest heading is the one carrying BOTH legends**, and the
		// loop above cannot reach it: a one-a-side board with a single-cell
		// shape draws the matchup legend alone. An area skill against a side it
		// marks draws the splash legend beside it, which is the real floor
		// question and the case the golden records.
		mixed, chosen, _ := aMixedAimList(t, c, hex.FormationRows)
		drawn := mixed.Choices(c)
		if !strings.Contains(drawn, matchupLegend(c)) {
			t.Fatalf("%s: the mixed aim list explains no mark, so the widest heading was "+
				"not measured", lang)
		}
		for _, line := range strings.Split(strings.TrimRight(drawn, "\n"), "\n") {
			if width := lipgloss.Width(line); width > drawable {
				t.Errorf("%s/%s: a row of the mixed aim list is %d cells over the %d it "+
					"has:\n%s", lang, chosen, width, drawable, line)
			}
		}
		widest := 0
		for _, line := range strings.Split(drawn, "\n") {
			if width := lipgloss.Width(line); width > widest {
				widest = width
			}
		}
		t.Logf("%s: the widest row of the mixed list is %d of %d cells",
			lang, widest, drawable)
	}
}

// TestTheMatchupIsTakenFromTheReadingAndNotFromTheBattle is the layer half.
//
// A live screen may read the battle in Attach and nowhere else, because Attach
// is the one place the mirror's lock is held — so a mark worked out at draw time
// off p.Fight would rebuild the race readBattle exists to remove, and it would
// rebuild it silently, since the two answers agree on a battle nobody is
// stepping. This pins the mechanism rather than the timing: the affinity every
// mark is computed from comes off the reading, so a screen whose Fight pointer
// has been taken away still draws the same list.
func TestTheMatchupIsTakenFromTheReadingAndNotFromTheBattle(t *testing.T) {
	c, _ := start(t, i18n.En)
	for _, want := range everyMatchupMark {
		p, chosen := aMarkedAim(t, c, want)
		read := p.read()
		withTheBattle := p.choices(c, read)
		// The same reading, and no battle at all. Every read of p.Fight on this
		// path would answer differently; nothing else can.
		blind := p
		blind.Fight = nil
		if withoutIt := blind.choices(c, read); withoutIt != withTheBattle {
			t.Errorf("%s: the aim list changes when the battle pointer is taken away, so "+
				"something on the drawing path is reading it:\nwith:\n%s\nwithout:\n%s",
				chosen, withTheBattle, withoutIt)
		}
	}
}

// TestTheReadingCarriesEveryUnitsAffinity is the other end of the same rule: the
// reading has to hold what the drawing is about to ask for.
//
// ⚠️ It is a separate test because the one above passes for a bad reason if the
// reading carries nothing — two identical blank lists are equal. This asserts the
// affinities are really there and really are the battle's own.
func TestTheReadingCarriesEveryUnitsAffinity(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := atABattleOf(t, c, 3)
	read := p.read()
	if len(read.units) == 0 {
		t.Fatal("the reading holds no units, so it cannot carry an affinity")
	}
	for _, drawn := range read.units {
		fighting, known := p.Fight.Unit(drawn.ID)
		if !known {
			t.Fatalf("the reading holds %s and the battle does not", drawn.ID)
		}
		if drawn.Affinity != fighting.Affinity {
			t.Errorf("%s reads as %s and fights as %s",
				drawn.ID, drawn.Affinity, fighting.Affinity)
		}
		if drawn.Side != fighting.Side {
			t.Errorf("%s reads on side %v and fights on side %v",
				drawn.ID, drawn.Side, fighting.Side)
		}
	}
	t.Logf("%d units carried an affinity and a side", len(read.units))
}

// TestTheMarksAreTheOnesTheChartAsksFor ties the four constants to the three
// levels the chart declares, so the mapping cannot drift into a table of
// hardcoded multipliers.
func TestTheMarksAreTheOnesTheChartAsksFor(t *testing.T) {
	_, lib := start(t, i18n.En)
	chart := lib.Chart()
	levels := chart.Multipliers()
	// Written as the four figures the shipped chart composes, derived here from
	// the three it declares rather than typed in: a chart retuned moves these
	// with it, and a mark that stopped following would be a mark saying the
	// opposite of what the damage does.
	table := []struct {
		value int
		want  string
	}{
		{levels.Disadvantage * levels.Disadvantage / levels.Neutral, matchupVeryResistMark},
		{levels.Disadvantage, matchupResistMark},
		{levels.Neutral, ""},
		{levels.Advantage, matchupWeakMark},
		{levels.Advantage * levels.Advantage / levels.Neutral, matchupVeryWeakMark},
	}
	for _, row := range table {
		t.Run(strconv.Itoa(row.value), func(t *testing.T) {
			attacker, defender, found := somePairingAt(chart, row.value)
			if !found {
				t.Fatalf("no attacker and affinity compose to %d, so this row measures "+
					"nothing", row.value)
			}
			if got := matchupMark(chart, attacker, defender); got != row.want {
				t.Errorf("%s against %s composes to %d and draws %q, want %q",
					attacker, defender, row.value, got, row.want)
			}
		})
	}
	if _, _, found := somePairingAt(chart, 0); found {
		t.Error("some pairing composes to nought, which no mark covers")
	}
}

// somePairingAt is any attacker and affinity whose composed multiplier is the
// value asked for.
func somePairingAt(chart *element.Chart, value int) (element.Element, element.Affinity, bool) {
	for _, attacker := range element.All() {
		for _, primary := range element.All() {
			single, err := element.Single(primary)
			if err != nil {
				continue
			}
			if chart.MultiplierAgainst(attacker, single) == value {
				return attacker, single, true
			}
			for _, secondary := range element.All() {
				pair, err := element.Dual(primary, secondary)
				if err != nil {
					continue
				}
				if chart.MultiplierAgainst(attacker, pair) == value {
					return attacker, pair, true
				}
			}
		}
	}
	return 0, element.Affinity{}, false
}

// TestWhichMarksTheShippedDataReaches is the honest half of guard two: the
// boards above are built out of the library these tests load, which is the
// shipped books with the fixture cast appended, and that cast carries two
// dual-element characters of its own. What a player can actually see is a
// different question and it has three different answers.
//
// ⚠️ **It loads the shipped directory rather than the scratch one**, which is
// the only way to ask it. The claims it pins:
//
//   - Every one of the four marks is reachable from the shipped CAST: two
//     characters declare two elements, so an electric skill against either one
//     of them composes past a single advantage and a metal skill composes past a
//     single disadvantage.
//   - Only the middle two are reachable from the shipped SQUADS, because no side
//     in squads.json fields either dual character. So a player who only ever
//     picks a shipped pairing sees "+" and "-" and never the doubled pair — which
//     is a fact about squads.json rather than about this screen, and the reason
//     it is logged here rather than being treated as a defect.
func TestWhichMarksTheShippedDataReaches(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	chart := lib.Chart()
	fromCast := map[string]string{}
	for _, attacker := range lib.Characters().All() {
		for _, id := range attacker.SkillsAt(progression.LevelCap, progression.Furthest) {
			declared, err := lib.Skills().Lookup(id)
			if err != nil || declared.Power == 0 {
				continue
			}
			for _, defender := range lib.Characters().All() {
				if mark := matchupMark(chart, declared.Element, defender.Element); mark != "" {
					if _, taken := fromCast[mark]; !taken {
						fromCast[mark] = attacker.ID + " " + id + " → " + defender.ID
					}
				}
			}
		}
	}
	for _, mark := range everyMatchupMark {
		if fromCast[mark] == "" {
			t.Errorf("no shipped character can draw %q against any other, so that mark is "+
				"a state the shipped game never reaches", mark)
		}
	}
	t.Logf("from the shipped cast: %v", fromCast)

	fromSquads := map[string]string{}
	for _, squad := range lib.Squads() {
		for _, other := range lib.Squads() {
			for _, unit := range squad.Units {
				attacker, known := lib.Characters().Get(unit.Character)
				if !known {
					continue
				}
				kit := unit.Skills
				if len(kit) == 0 {
					kit = attacker.SkillsAt(unit.Level, progression.Furthest)
				}
				for _, id := range kit {
					declared, err := lib.Skills().Lookup(id)
					if err != nil || declared.Power == 0 {
						continue
					}
					for _, seat := range other.Units {
						defender, known := lib.Characters().Get(seat.Character)
						if !known {
							continue
						}
						if mark := matchupMark(chart, declared.Element, defender.Element); mark != "" {
							if _, taken := fromSquads[mark]; !taken {
								fromSquads[mark] = squad.ID + "/" + unit.ID + " " + id +
									" → " + defender.ID
							}
						}
					}
				}
			}
		}
	}
	if len(fromSquads) == 0 {
		t.Fatal("no shipped squad can mark any other, so a player picking a shipped " +
			"pairing never sees this feature at all")
	}
	t.Logf("from the %d shipped squads: %v", len(lib.Squads()), fromSquads)
	for _, doubled := range []string{matchupVeryWeakMark, matchupVeryResistMark} {
		if where, reached := fromSquads[doubled]; reached {
			t.Logf("⚠️ the shipped squads now reach %q (%s), which they did not when this "+
				"was written — the comment above is what to update", doubled, where)
		}
	}
}

// TestTheSplashFrameComesFromTheReading is the other draw-time read of the
// battle, and it predates the matchup mark: splashUnder resolved the caster's
// half by calling p.Fight.Unit while drawing, which on a live screen is a read
// of the mirror's battle outside the only lock there is.
//
// ⚠️ **Taking the battle away cannot measure it, and that is why this test
// exists instead.** The fallback when the battle is absent is hex.SideAlly, and
// the prompted unit is on the ally side in every battle this screen opens — so
// the old code and the new one agree on every board, and the obvious probe
// passes with the defect in place. What discriminates is mutating the READING:
// flip the side the reading reports and the cells have to move, which they can
// only do if the reading is what was consulted.
func TestTheSplashFrameComesFromTheReading(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := anAreaBattle(t, c)
	read := p.read()
	flipped := playReading{units: make([]playUnit, len(read.units))}
	copy(flipped.units, read.units)
	for index := range flipped.units {
		if flipped.units[index].ID == p.Pending.Unit {
			flipped.units[index].Side = hex.SideEnemy
			if read.units[index].Side == hex.SideEnemy {
				flipped.units[index].Side = hex.SideAlly
			}
		}
	}
	var measured int
	for option := range p.Pending.Options {
		for aim := range p.Pending.Options[option].Aims {
			probe := p
			probe.Option, probe.Aim, probe.Aiming = option, aim, true
			declared := probe.Pending.Options[option]
			asRead := probe.splashUnder(c, read, declared)
			asFlipped := probe.splashUnder(c, flipped, declared)
			if len(asRead) == 0 && len(asFlipped) == 0 {
				continue
			}
			measured++
			if slices.Equal(asRead, asFlipped) {
				// A shape that mirrors — `column` and `single` are the two that
				// do — catches the same cells in either frame and cannot tell
				// the two readings apart. That is not a failure, it is a shape
				// with nothing to say here.
				continue
			}
			return
		}
	}
	if measured == 0 {
		t.Fatal("no option this turn catches a second cell from any aim, so nothing here " +
			"resolves a shape at all")
	}
	t.Fatalf("every one of the %d footprints this turn is the same in both frames, so "+
		"flipping the reading's side changed nothing and this test cannot tell a "+
		"reading-fed resolution from a battle-fed one", measured)
}

// TestTheMarksFollowARetunedChart is what makes "the thresholds are read off the
// chart" a property rather than a sentence in a comment.
//
// ⚠️ **Every other test here asks the SHIPPED chart, and against the shipped
// chart a chart-derived reading and a hardcoded one agree by construction.**
// Replace `levels.Advantage` with 1500 and `levels.Disadvantage` with 667 in
// matchupMark and the whole package stays green — including
// TestTheMarksAreTheOnesTheChartAsksFor, which derives its table from the chart
// and then asks that same chart. The mapping is only observable where the two
// spellings must disagree, and the only way to get there is a chart with
// different numbers in it.
//
// `element.ParseChart` takes raw JSON, so a retuned chart is a **fixture**
// rather than a mock: the shipped file with three numbers swapped, so the
// cycles, the mutual pair and the inert element are all the real ones and
// Chart.Validate is really run over it.
//
// Two tunings rather than one, because each catches half the switch:
//
//   - **Wider** (2000/1000/500): a single advantage is 2000 and a single
//     disadvantage 500, so the hardcoded spelling reads both as doubled —
//     it catches the `+` and `-` arms.
//   - **Narrower** (1200/1000/850): a doubled advantage is 1440 and a doubled
//     disadvantage 722, so the hardcoded spelling reads both as single —
//     it catches the `++` and `--` arms.
//
// ⚠️ Every one of the four marks is reachable under **both** tunings; what
// differs is which of them a hardcoded reading gets wrong. So nothing here is
// padded to reach a state, and the discrimination net at the bottom is what says
// all four arms were really measured rather than merely drawn.
func TestTheMarksFollowARetunedChart(t *testing.T) {
	_, lib := start(t, i18n.En)
	shipped := lib.Chart().Multipliers()
	discriminated := map[string]string{}
	for _, tuning := range []struct {
		name                             string
		advantage, neutral, disadvantage int
	}{
		{"wider than the shipped chart", 2000, 1000, 500},
		{"narrower than the shipped chart", 1200, 1000, 850},
	} {
		t.Run(tuning.name, func(t *testing.T) {
			chart := aRetunedChart(t, tuning.advantage, tuning.neutral, tuning.disadvantage)
			levels := chart.Multipliers()
			if levels.Advantage == shipped.Advantage || levels.Disadvantage == shipped.Disadvantage {
				t.Fatalf("the retuned chart came back at %v, which is the shipped tuning: "+
					"the fixture did not retune anything and nothing below can disagree",
					levels)
			}
			// ⚠️ The five composed values of THIS chart, derived from its own
			// three levels. Asserting the MARKS and never the multipliers: a
			// check that Multipliers() came back different proves the fixture
			// works and says nothing about what matchupMark consulted.
			table := []struct {
				value int
				want  string
			}{
				{levels.Disadvantage * levels.Disadvantage / levels.Neutral, matchupVeryResistMark},
				{levels.Disadvantage, matchupResistMark},
				{levels.Neutral, ""},
				{levels.Advantage, matchupWeakMark},
				{levels.Advantage * levels.Advantage / levels.Neutral, matchupVeryWeakMark},
			}
			for _, row := range table {
				attacker, defender, found := somePairingAt(chart, row.value)
				if !found {
					t.Errorf("nothing in the retuned chart composes to %d, so the %q row "+
						"measures nothing", row.value, row.want)
					continue
				}
				got := matchupMark(chart, attacker, defender)
				if got != row.want {
					t.Errorf("under %v, %s against %s composes to %d and draws %q, want "+
						"%q — the mark is not following the chart it was handed",
						levels, attacker, defender, row.value, got, row.want)
				}
				// The vacuity half. A row where the shipped chart's own numbers
				// would have answered the same thing cannot tell a chart-derived
				// reading from a hardcoded one, however green it is.
				if atShippedLevels(row.value, shipped) != row.want {
					discriminated[row.want] = tuning.name
				}
			}
		})
	}
	// ⚠️ **All four arms, across the two tunings.** Without this the test passes
	// on a fixture that happens to agree with the shipped constants everywhere,
	// which is the exact failure mode it was written to remove.
	for _, mark := range everyMatchupMark {
		if discriminated[mark] == "" {
			t.Errorf("no retuned chart makes %q disagree with the shipped constants, so "+
				"that arm of the switch is drawn here and not measured", mark)
		}
	}
	t.Logf("arms caught by a retuning: %v", discriminated)
}

// atShippedLevels is the answer a matchupMark with the shipped chart's three
// numbers written into it would give.
//
// ⚠️ **It is the MUTATION, kept here on purpose and marked as such** — the same
// arrangement as narrowSwung in internal/core/combat's swung_test.go, which
// keeps the pre-fix expression verbatim so the test can say what it refuses. It
// exists only for the discrimination net above and must never be what the
// assertions compare against: that would pin the defect instead of the rule.
func atShippedLevels(multiplier int, shipped element.Multipliers) string {
	switch {
	case multiplier > shipped.Advantage:
		return matchupVeryWeakMark
	case multiplier > shipped.Neutral:
		return matchupWeakMark
	case multiplier < shipped.Disadvantage:
		return matchupVeryResistMark
	case multiplier < shipped.Neutral:
		return matchupResistMark
	}
	return ""
}

// aRetunedChart is the shipped element chart with its three multipliers
// replaced, parsed through the real parser.
//
// The rest of the file is carried through untouched as raw JSON rather than
// rebuilt, which is what keeps this a retuning and not a second chart: the
// cycles, the mutual pair and the inert list stay the shipped ones, so
// Chart.Validate's rules about classification and balance are enforced over
// real data and an element added to the game needs no change here.
func aRetunedChart(t *testing.T, advantage, neutral, disadvantage int) *element.Chart {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(shippedDataDir, "elements.json"))
	if err != nil {
		t.Fatalf("read the shipped element chart: %v", err)
	}
	var file map[string]json.RawMessage
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode the shipped element chart: %v", err)
	}
	if _, declared := file["multipliers"]; !declared {
		t.Fatal("the shipped element chart declares no multipliers, so there is nothing " +
			"to retune and this fixture is measuring a file it does not understand")
	}
	file["multipliers"] = json.RawMessage(fmt.Sprintf(
		`{"advantage":%d,"neutral":%d,"disadvantage":%d}`, advantage, neutral, disadvantage))
	out, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("re-encode the retuned chart: %v", err)
	}
	chart, err := element.ParseChart(out)
	if err != nil {
		t.Fatalf("the retuned chart %d/%d/%d will not parse: %v",
			advantage, neutral, disadvantage, err)
	}
	return chart
}
