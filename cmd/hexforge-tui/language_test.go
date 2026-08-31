package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// Everything about the two languages, asserted against the real model: the
// screens are rendered by driving it, not by calling the catalog and hoping a
// screen asks for the same key.

// everyScreen is each view in the order the menu offers them, plus the two
// forms, which are states of a screen rather than screens of their own.
func everyScreen(t *testing.T, m model) map[string]model {
	t.Helper()
	adding := m.enter(screenOrigins)
	adding.origins.adding = true
	form := m.enter(screenNew)
	addSkill := m.enter(screenSkills)
	addSkill.skills.adding = true
	// The same form over a skill that already exists, which is the widest it ever
	// draws: every field is prefilled from the book rather than empty.
	editSkill := m.enter(screenSkills)
	editSkill.skills = editSkill.skills.prefill(m.lib, editSkill.skills.skills[0])
	// And the listing as an edit leaves it: two lines rather than one, the second
	// of which is the damage before and after.
	editedSkill := m.enter(screenSkills)
	editedSkill.skills.edited = someSkillChange(t, m)
	// The typed filter on the skill listing, in each of the three states it
	// draws: the field just opened with nothing in it, a query that has found
	// several rows, and a query that has found none. Three rather than one
	// because they share no line — an empty field says what to type, a full one
	// says how much of the book is left, and a query that found nothing says so
	// where the rows would have been.
	//
	// ⚠️ Driven with the keys an author would press, not by writing the query
	// onto the screen's own field. The query is what decides which rows there
	// are, so a hand-set field would measure this test's idea of the filter
	// rather than the one / opens — and this is the fixture that has twice
	// measured a screen's early exit instead of the screen (playScreen's shared
	// battle, and plainTerminal above it).
	filterOpen := typeText(t, m.enter(screenSkills), "/")
	if !filterOpen.skills.filtering {
		t.Fatal("/ did not open the skill filter, so its three states are drawn by " +
			"nothing in the sweep")
	}
	filterFound := typeText(t, filterOpen, someSkillQuery)
	filterNothing := typeText(t, filterOpen, noSkillQuery)
	// And the fixture's own discrimination, which no assertion downstream can
	// see: a query that has quietly stopped matching turns the "found" state into
	// a second copy of the "nothing" state, and both would still render.
	if found, all := len(filterFound.skills.rows()), len(filterFound.skills.skills); found < 2 || found >= all {
		t.Fatalf("the query %q finds %d of %d skills, so the filtered listing is not "+
			"a narrowed one", someSkillQuery, found, all)
	}
	if found := len(filterNothing.skills.rows()); found != 0 {
		t.Fatalf("the query %q finds %d skills, so the empty result is drawn by nothing",
			noSkillQuery, found)
	}
	// The shape diagram, which is a state of the skill form rather than a screen
	// of its own. The shape it draws is the widest the board ever is, because the
	// board is a fixed size — what varies is the line under it, so this is here
	// for the same reason every other state is: to be measured in both languages.
	shape := m.enter(screenSkills)
	shape.skills.adding = true
	shape.skills.field = skillFieldShape
	shape.skills.shapeIndex = indexOf(m.lib.PatternNames(), "pierce")
	shape.skills.shapeDrawn = true
	// The two pickers, opened over the form that raises each. The kit's rows
	// carry a refusal each and an allowlist's do not, so both shapes of row are
	// measured.
	kit := form.openKit()
	allowlist := addSkill.openAllowlist(skillFieldKeptForCharacters)
	// The status picker is the fifth, and the only one that collects a number as
	// well as a set, so its extra row is measured with the rest.
	statuses := addSkill.openStatuses()
	statuses.picker.chosen = []string{"poison"}
	statuses.picker.typed.SetValue("300")
	// And the character allowlist with its filter narrowed, which is a line the
	// unfiltered picker does not draw.
	filtered := addSkill.openAllowlist(skillFieldKeptForCharacters)
	filtered.picker.nextFilter()
	// The species picker, opened over the character form's species field the way
	// every other picker here is opened over the form that raises it — a
	// hand-built pickState would measure this test's idea of the screen rather
	// than the one a keystroke reaches.
	//
	// ⚠️ It was missing, and the English sweep below is built to catch exactly
	// what was wrong with it: the detail cell read the species book's own name
	// raw, so an English row drew "dragon  rồng". Every line of every screen in
	// this map is checked against every data name and nothing objected, because
	// this screen was rendered by nothing in the suite.
	speciesPick := form.openSpecies()
	// And the origins allowlist, which is here for width and wording rather than
	// for a defect: its branch reads work.Title raw and rightly so — a work's
	// title is a proper noun and there is no Lang accessor being gone round — but
	// its own title line, i18n.PickerOriginsTitle, was rendered by nothing in
	// this suite, so neither language's spelling of it had ever been measured
	// against the smallest window.
	//
	// The other two unregistered pickers buy nothing and are left out: the
	// species and origins *allowlists* differ from the two above only in a hint
	// line the character allowlist already draws, so every line of them is
	// already measured somewhere in this map.
	originsPick := addSkill.openAllowlist(skillFieldKeptForOrigins)
	// The spar, which is the only screen that runs battles to draw itself. Its
	// widest state is the one with a row per character, which is what entering it
	// over a checked library gives.
	spar := m.enter(screenCheck).enter(screenSpar)
	// The description screen in both of its shapes. It is one screen branching on
	// which screen raised it, and the two branches share no line, so measuring
	// one measures nothing about the other.
	skillBlurb := m.enter(screenSkills)
	skillBlurb.blurb.from = screenSkills
	skillBlurb.screen = screenBlurb
	traitBlurb := m.enter(screenBrowse)
	traitBlurb.browse.cursor = widestTraitRow(traitBlurb)
	traitBlurb.browse.level = progression.LevelCap
	traitBlurb.blurb.from = screenBrowse
	traitBlurb.screen = screenBlurb
	// The affinity chart, on the element whose description is longest: the rows
	// are all one shape, so what varies is the pane below them.
	elements := m.enter(screenElements)
	elements.elements.cursor = widestElementRow(elements)
	// The species reference twice. The shipped cast claims every kind, so the
	// row with an empty members cell and the line that explains it are drawn
	// only by a book that has one — and they are wording like any other, so they
	// are measured rather than left to the first unclaimed kind somebody writes.
	// The chart, whose widest line is the longest ring — a fixed thing the data
	// decides, so entering it is the whole of its worst case.
	graph := m.enter(screenElements)
	graph.screen = screenChart
	species := m.enter(screenSpecies)
	unclaimed := m.enter(screenSpecies)
	unclaimed.species = withNobodyClaiming(unclaimed.species)
	// The build catalogue twice. Every shipped build spends its trait slot, so the
	// row that says a slot was deliberately left empty is drawn only by a
	// catalogue that has one — and it is wording like any other, so it is measured
	// here rather than left to the first traitless build somebody writes.
	builds := m.enter(screenBuilds)
	traitless := m.enter(screenBuilds)
	traitless.builds = withNoTraitTaken(t, traitless.builds)
	// The squad builder in each of its three depths, plus the two pickers it
	// raises. The fixture's catalogue starts empty, so the listing with a squad in it and
	// the two views under it are drawn only by a squad built here — and every
	// line of them is wording, so they are measured rather than left to whoever
	// builds the first one.
	squadEmpty := m.enter(screenSquads)
	building := squadEmpty
	building.squad = someSquad(t, building)
	member := building
	member.squad = member.squad.editUnit(0)
	// And the same member standing in the rank whose name is longest. The
	// fixture puts it in the front rank, so the other two ranks' words are drawn
	// by nothing here — and a wording nothing renders is a wording no width test
	// measures. The cell is written on the unit under edit rather than committed,
	// which is also the state the live grid is for: the picture follows the
	// arrows, so this is the widest slot row beside the moved mark.
	deepest := member
	deepest.squad = inTheWidestRank(deepest)
	// And the same member holding a character the cast has taken out of the
	// builder's list, which is the one state that draws the held-back line: the
	// chooser goes on offering a character already chosen, so the screen has to
	// say why one nothing else offers is on the list. Built by hand rather than
	// left to the shipped data, for the reason the traitless build below is —
	// nothing the fixture ships is held back, so the wording would be measured
	// by nothing.
	heldBack := member
	heldBack.squad = withAHeldBackMember(t, heldBack.squad)
	// ⚠️ A state registered here that does not actually draw the line it exists
	// for is a state every sweep below measures nothing about, and it passes
	// them all. This repository has shipped that fixture twice — plainTerminal's
	// early return and the played battle's finished board — so the fixture says
	// so itself rather than trusting that it reached the branch.
	if !strings.Contains(heldBack.screenContent(), heldBack.text(i18n.SquadHeldBack)) {
		t.Fatalf("the held-back member state draws no held-back line, so every sweep over it measures nothing:\n%s",
			heldBack.screenContent())
	}
	skillPick := member.openSquadSkills()
	traitPick := member.openSquadPassives()
	// And each picker with a description in front of its list, which is the
	// picker's other state and shares no line with the list. The trait one needs
	// a member that actually learns a trait: the fixture cast declares none, so
	// every test that had opened that picker had opened an empty one — which is
	// how it shipped drawing an error where each row's detail belongs.
	skillReading := reading(skillPick)
	traitHolder, holds := aTraitHolder(building)
	traitRows := traitHolder.openSquadPassives()
	traitReading := reading(traitRows)
	// The fight, which is the only screen here that runs battles to draw itself.
	// It needs a squad the library has actually been told about, because the run
	// looks the pair up there rather than on the screen — so this one is saved
	// rather than held. Fought against itself, which is the control and the one
	// pairing a fixture cast is guaranteed to be able to field.
	fight := withASquadSaved(t, building).enter(screenFight)
	// And the same screen with nothing to fight, which is a different line.
	noSquads := m.enter(screenFight)
	// The battle, in each of the four states it draws: waiting on a skill,
	// waiting on a cell, describing the option in front, and over. The board and
	// the roster in it are the game client's own drawing rather than this one's,
	// and they are measured here because this is the screen that has to fit them
	// beside a menu.
	//
	// ⚠️ **Each state enters the screen for itself, and that is not tidiness.**
	// playScreen is the one screen holding something the model does not copy: the
	// battle is a *battle.Battle, so `state := battle` shares the pointer and
	// playing one of them out steps all of them. That is exactly what used to
	// happen here — `finished` was a copy of `battle` driven to its end, so by the
	// time anything drew `battle` its own p.fight.Finished() was true, play.view
	// returned at the game-over branch, and PlayFooter, PlayAimFooter, the option
	// rows and the whole aim block were drawn by **nothing in this suite**. Every
	// width and translation test passed over a screen it never rendered.
	//
	// This is the second fixture in this repository whose early return made the
	// interesting branch unreachable — plainTerminal was the first, where every
	// test set NO_COLOR and returned above the branch. A fixture that reaches a
	// screen's early exit measures the exit.
	battle := fight.enter(screenPlay)
	aiming := fight.enter(screenPlay)
	aiming.play.aiming = true
	// The description of the option under the battle's cursor, which is a state of
	// the blurb rather than a screen of its own — and its own battle for the
	// reason above, even though nothing here steps it.
	playBlurb := fight.enter(screenPlay)
	playBlurb.blurb.from = screenPlay
	playBlurb.screen = screenBlurb
	finished := fight.enter(screenPlay)
	for range 200 {
		if finished.play.fight == nil || finished.play.fight.Finished() {
			break
		}
		finished = typeText(t, finished, "a")
	}
	// And the battle in the window it cannot fit, which is a state of the screen
	// rather than a screen of its own and is here for the reason every other
	// state is. Its body is budgeted against the room frame will give it, so at
	// the declared floor the board, the order line and the log are dropped and
	// one dim line says so — a wording like any other, and one no other state
	// here can render, since every model above is measured in a window tall
	// enough to hold the whole screen.
	//
	// Five a side, because that is the largest squad there is and the roster is
	// what grows with it; aiming as well, because the cells are reserved with the
	// options and are what push the roster into being clipped a row at a time,
	// which is a different clause of the same line.
	squeezed := atABattleOf(t, m, hex.MaxTeamSize)
	squeezed.width, squeezed.height = minWidth, minHeight
	squeezedAim := squeezed
	squeezedAim.play.aiming = true
	// And the battle with its log scrolled back, which is a state of the screen
	// rather than a screen of its own and is here for the reason every other state
	// is: it is the only one that draws the log's position on the heading row, and
	// that row is a wording like any other.
	//
	// ⚠️ It has to be **built**, twice over. A battle a few turns old has a history
	// that fits its frame, so nothing is hidden and the position is correctly not
	// drawn; and every model above stands in a window where the log has room, so
	// none of them can reach it either. The history is played out and the frame
	// scrolled back with the key a reader would press — its own battle, because
	// playScreen holds a pointer the model does not copy.
	scrolled := aLongLog(t, m.lang, 3)
	scrolled = key(t, scrolled, "pgup")
	if scrolled.play.logFollow {
		t.Fatal("the scrolled battle did not scroll, so the log's position is drawn " +
			"by nothing in the sweep")
	}
	screens := map[string]model{
		"shape diagram":            shape,
		"spar":                     spar,
		"menu":                     m.enter(screenMenu),
		"browse":                   m.enter(screenBrowse),
		"form":                     form,
		"origins":                  m.enter(screenOrigins),
		"add a work":               adding,
		"skills":                   m.enter(screenSkills),
		"statuses":                 m.enter(screenStatuses),
		"traits":                   m.enter(screenPassives),
		"add a skill":              addSkill,
		"edit a skill":             editSkill,
		"edited a skill":           editedSkill,
		"filtering skills":         filterOpen,
		"filtered skills":          filterFound,
		"skills filtered to none":  filterNothing,
		"kit picker":               kit,
		"allowlist picker":         allowlist,
		"status picker":            statuses,
		"filtered picker":          filtered,
		"species picker":           speciesPick,
		"origins picker":           originsPick,
		"skill blurb":              skillBlurb,
		"trait blurb":              traitBlurb,
		"check":                    m.enter(screenCheck),
		"elements":                 elements,
		"chart":                    graph,
		"species":                  species,
		"unclaimed kind":           unclaimed,
		"builds":                   builds,
		"squads":                   squadEmpty,
		"a squad":                  building,
		"a squad member":           member,
		"a deep member":            deepest,
		"a held-back member":       heldBack,
		"a squad kit":              skillPick,
		"a squad trait":            traitPick,
		"reading a skill":          skillReading,
		"a fight":                  fight,
		"nothing to fight":         noSquads,
		"a battle":                 battle,
		"aiming":                   aiming,
		"a squeezed battle":        squeezed,
		"a squeezed battle aiming": squeezedAim,
		"a scrolled battle log":    scrolled,
		"a battle blurb":           playBlurb,
		"a battle over":            finished,
		"traitless build":          traitless,
	}
	// The two states a trait picker only has once something in the book learns
	// one. Skipped rather than faked when nothing does: a picker over invented
	// rows would measure wording against a trait nobody could reach.
	if holds {
		screens["a squad trait held"] = traitRows
		screens["reading a trait"] = traitReading
	}
	return screens
}

// withAHeldBackMember takes the character the member under edit already holds
// out of the builder's list, without touching the member.
//
// It edits the screen's own copy of the cast rather than the library, which is
// the same shape withNoTraitTaken uses and for the same reason: the state wanted
// is one the shipped data does not have, and authoring it here keeps the fixture
// from depending on which character cast.json happens to hide. The slice is
// copied because everyScreen hands out models sharing this one's backing array.
func withAHeldBackMember(t *testing.T, s squadScreen) squadScreen {
	t.Helper()
	characters := append([]cast.Character(nil), s.characters...)
	held := false
	for index := range characters {
		if characters[index].ID == s.unit.Character {
			characters[index].Hidden, held = true, true
		}
	}
	if !held {
		t.Fatalf("the fixture member names %q, which is not in the cast the screen holds", s.unit.Character)
	}
	s.characters = characters
	return s
}

// aTraitHolder is the squad in hand with its member pointed at whichever
// character in the book learns the most traits, and false when none does.
//
// It looks the character up rather than naming one, which is the rule the
// fixture exists to keep — and it is needed at all because the fixture cast
// learns no traits, so every screen that had opened the squad trait picker had
// opened an empty one and nothing measured a row of it.
func aTraitHolder(m model) (model, bool) {
	s := m.squad
	found, most := -1, 0
	for index, character := range s.characters {
		if held := len(character.PassivesAt(
			progression.LevelCap, progression.Furthest)); held > most {
			found, most = index, held
		}
	}
	if found < 0 || len(s.editing.Units) == 0 {
		return m, false
	}
	character := s.characters[found]
	unit := s.editing.Units[0]
	unit.Character, unit.Level, unit.Stage = character.ID, progression.LevelCap, ""
	known := character.SkillsAt(unit.Level, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}
	unit.Skills = known
	unit.Passives = character.PassivesAt(unit.Level, progression.Furthest)[:cast.TraitSlots]
	s.editing.Units = []placement.Placement{unit}
	m.squad = s.editUnit(0)
	return m, true
}

// reading is the picker in hand with a description in front of its list.
//
// The pickState is cloned rather than flipped in place, because m.picker is a
// pointer and every model copied off this one shares it: setting the flag on the
// picker itself would put the entry beside this one into a state it is not here
// to measure.
func reading(m model) model {
	clone := *m.picker
	clone.reading = true
	m.picker = &clone
	return m
}

// inTheWidestRank is the member under edit moved into the rank whose wording
// takes the most cells, which is the widest the slot row ever draws.
//
// The rank is looked up rather than named, for the reason widestElementRow and
// widestTraitRow are: which of the three words is longest is a fact about a
// language, so naming one would measure English and skip Vietnamese.
func inTheWidestRank(m model) squadScreen {
	s := m.squad
	found, most := s.unit.Slot, 0
	for _, slot := range formationSlots() {
		if width := lipgloss.Width(m.rankLabel(slot)); width > most {
			found, most = slot, width
		}
	}
	s.unit.Slot = found
	return s
}

// widestElementRow is the element whose chart entry takes the most cells, which
// is the busiest the description pane under the listing ever draws.
func widestElementRow(m model) int {
	found, most := 0, 0
	for index, member := range element.All() {
		for _, line := range strings.Split(m.lang.DescribeElement(member, m.lib.Chart()), "\n") {
			if width := lipgloss.Width(line); width > most {
				found, most = index, width
			}
		}
	}
	return found
}

// withNobodyClaiming is the species reference as a book with an unclaimed kind
// draws it: the kind under the cursor counted down to nobody, which is what puts
// the "nobody is one" line on screen.
//
// A copy of the map rather than a write into it, because the screen it came from
// is one of the models this helper hands back and a shared map would clear that
// one's count too.
func withNobodyClaiming(s speciesScreen) speciesScreen {
	claimed := make(map[string]int, len(s.claimed))
	for id, count := range s.claimed {
		claimed[id] = count
	}
	if len(s.kinds) > 0 {
		claimed[s.kinds[clamp(s.cursor, 0, len(s.kinds)-1)].ID] = 0
	}
	s.claimed = claimed
	return s
}

// withNoTraitTaken is the build catalogue as a build that spends no trait slot
// draws it: the build under the cursor with its trait taken off, which is what
// puts the "takes no trait" row on screen.
//
// The rows are copied rather than written into, because the screen it came from is
// one of the models everyScreen hands back and a shared slice would empty that
// one's build too.
//
// A build with no trait is legal — cast.ParseBuilds insists on the kit and not on
// the trait, because a unit with no skills cannot act while a unit with no trait
// is an ordinary one — and nothing shipped is one. That is exactly the case a
// state built by hand is for: the alternative is wording no test ever renders.
// someSquad is a squad built by hand, because the fixture's catalogue starts empty.
//
// It is the widest state the builder draws: an id and a name typed, somebody in
// the front rank, and a full kit — every one of which is a line the empty
// listing never renders and every one of which is wording.
func someSquad(t *testing.T, m model) squadScreen {
	t.Helper()
	s := m.squad.begin()
	s.editing.ID = "do-thu"
	s.editing.Name = "đội thử"
	s.idInput.SetValue(s.editing.ID)
	s.nameInput.SetValue(s.editing.Name)
	if len(s.characters) == 0 {
		t.Fatal("the fixture cast is empty, so no squad can be built from it")
	}
	character := s.characters[0]
	unit := placement.Placement{
		ID:        "mot",
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: 1},
	}
	known := character.SkillsAt(unit.Level, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}
	unit.Skills = known
	if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
		unit.Passives = traits[:cast.TraitSlots]
	}
	s.editing.Units = []placement.Placement{unit}
	s.unit = unit
	return s
}

// withASquadSaved writes the squad in hand into the library and hands back a
// model that can see it, which is what a fight needs: the run looks its pair up
// in the catalogue rather than taking them from the screen.
func withASquadSaved(t *testing.T, m model) model {
	t.Helper()
	if err := m.lib.SaveSquad(m.squad.editing); err != nil {
		t.Fatalf("save the fixture squad: %v", err)
	}
	// The squad in hand is now the squad on the file, so the baseline the
	// discard guard compares against moves with it — which is what the save key
	// does for a squad built through the keyboard. Without this the screen would
	// hold a squad differing from nothing and still be asked about discarding
	// it, and every fixture that backs out to the catalogue would stop there.
	m.squad.baseline = m.squad.editing.Clone()
	m.squad = m.squad.refresh(m.lib)
	return m
}

func withNoTraitTaken(t *testing.T, b buildsScreen) buildsScreen {
	t.Helper()
	rows := append([]buildRow(nil), b.rows...)
	found := false
	for index, row := range rows {
		if !row.build() {
			continue
		}
		rows[index].built.Passives = nil
		b.cursor, found = index, true
		break
	}
	if !found {
		t.Fatal("the catalogue holds no build, so there is no trait to take off")
	}
	b.rows = rows
	return b
}

// widestTraitRow is the browser row carrying the most traits, which is the
// busiest the description screen ever draws — the row a width test wants and the
// row a shorter fixture would not have.
func widestTraitRow(m model) int {
	found, most := 0, 0
	for index, character := range m.browse.rows() {
		held := len(m.lib.KitPassives(
			character.PassivesAt(progression.LevelCap, progression.Furthest)))
		if held > most {
			found, most = index, held
		}
	}
	return found
}

// someSkillChange is a written edit as the screens receive one, built without
// writing anything: the figures are the library's own PreviewDamage, so the
// before-and-after line is as wide here as it is after a real edit.
func someSkillChange(t *testing.T, m model) *forge.SkillChange {
	t.Helper()
	skills := m.lib.Skills().Skills()
	if len(skills) == 0 {
		t.Fatal("the shipped book holds no skills")
	}
	before := skills[0]
	after := before
	after.Power = before.Power*2 + 1000
	return &forge.SkillChange{
		Before: before, After: after,
		BeforeDamage: m.lib.PreviewDamage(before),
		AfterDamage:  m.lib.PreviewDamage(after),
	}
}

// TestEveryScreenRendersInBothLanguages walks the whole program twice.
//
// The markers are words that could only have come from a wording that was not
// translated — a screen still holding an English sentence in Vietnamese, or the
// reverse. Free text from the data files is skipped, because a biography is
// English in cast.json and will still be English on a Vietnamese screen: it is
// the author's prose, not the program's.
func TestEveryScreenRendersInBothLanguages(t *testing.T) {
	englishMarkers := []string{
		"MISSING", "PASSED", "FAILED", "quit", "back", "budget",
		"archetype", "characters", "cannot", "of the",
	}
	vietnameseMarkers := []string{
		"nhân vật", "hạn mức", "chiêu", "thoát", "quay lại", "kiểm tra", "giai đoạn",
	}
	cases := []struct {
		lang    i18n.Lang
		unwant  []string
		mustSay []string
	}{
		{i18n.Vi, englishMarkers, vietnameseMarkers},
		{i18n.En, vietnameseMarkers, englishMarkers},
	}
	for _, test := range cases {
		base, lib, _ := start(t, test.lang)
		free := freeText(lib)
		spoken := make(map[string]bool)
		for name, m := range everyScreen(t, base) {
			drawn := m.screenContent()
			if strings.TrimSpace(drawn) == "" {
				t.Errorf("the %s screen drew nothing in %s", name, test.lang)
			}
			for _, line := range strings.Split(drawn, "\n") {
				// The footer names the other language in its own name, which is
				// the one place a word from it is meant to be there.
				line = strings.ReplaceAll(line, "tiếng Việt", "")
				if carriesFreeText(line, free) {
					continue
				}
				for _, marker := range test.unwant {
					if strings.Contains(line, marker) {
						t.Errorf("the %s screen in %s still says %q:\n%s",
							name, test.lang, marker, line)
					}
				}
				for _, marker := range test.mustSay {
					if strings.Contains(line, marker) {
						spoken[marker] = true
					}
				}
			}
		}
		// And it really is the language asked for, rather than a screen that
		// happens to be free of the other one's words.
		if len(spoken) == 0 {
			t.Errorf("nothing on any screen reads like %s", test.lang)
		}
	}
}

// freeText is everything on screen that belongs to the data rather than to the
// program: biographies, notes, and the directory the books were read from.
func freeText(lib *forge.Library) []string {
	free := []string{lib.Dir()}
	for _, character := range lib.Characters().All() {
		if character.Bio != "" {
			free = append(free, character.Bio)
		}
		free = append(free, character.Name)
		for _, stage := range forge.StageFacts(character) {
			free = append(free, stage.Name)
		}
	}
	for _, origin := range lib.Origins().All() {
		if origin.Note != "" {
			free = append(free, origin.Note)
		}
		free = append(free, origin.Title)
	}
	// A species' note is authored prose like a biography or an origin's, and the
	// species reference prints it whole in both languages.
	//
	// Both languages, unlike the *name* beside it, and the two are not the same
	// call. A name is a translation of an id: printing the Vietnamese one on an
	// English screen is a wrong translation, and the bare id is the right answer,
	// which is why SpeciesName is empty in English. A note has no id to fall back
	// to, so dropping it leaves nothing — the trade an origin's note already
	// takes.
	for _, kind := range lib.Species().All() {
		if kind.Note != "" {
			free = append(free, kind.Note)
		}
	}
	// A build's name and its intent are authored in the catalogue, and the intent
	// is printed in both languages for the reason a species' note is: a name has an
	// id to fall back to and prose has nothing, so dropping it would leave the row
	// empty rather than untranslated. Both go in, because the Vietnamese screen
	// draws the name as well.
	for _, built := range lib.Builds() {
		free = append(free, built.Name)
		if built.Intent != "" {
			free = append(free, built.Intent)
		}
	}
	// A kit is a list of authored skill ids, so the rows that show one — the
	// archetype chooser and the kit field — are as long as the data makes them.
	// They are clipped like a biography rather than wrapped.
	for _, preset := range lib.Archetypes().All() {
		free = append(free, strings.Join(forge.PresetFacts(preset).Skills, " "))
	}
	return free
}

// whoMayCarry is the restriction column of the skills listing, for every skill
// in the book.
//
// It is free text for the width measurement and not for the language one, which
// is why it is a second list rather than part of freeText: the wording around
// the data ("hệ ...", "chủng loài ...") is the program's and has to be
// translated, but the data it wraps -- an element and a species, both named by
// the book -- has no length the program can promise. The kit rows above are
// exempted for exactly that reason; this column reached the same shape once a
// species restriction was authored.
func whoMayCarry(lang i18n.Lang, lib *forge.Library) []string {
	out := make([]string, 0, len(lib.Skills().Skills()))
	for _, declared := range lib.Skills().Skills() {
		if summary := lang.WhoMaySummary(declared); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

// traitCarriers is the "who carries it" column of the traits listing, for every
// trait in the book.
//
// whoMayCarry's twin, and exempt for the same reason: the cell is character ids
// the book named, and a listing that clipped it to the floor would hide which
// characters hold a trait on a terminal with columns to spare. It is a separate
// list from that one because the two columns are built by different callers, and
// a single list of "data cells" is a list nobody would remember to add to.
//
// ⚠️ It went missing for one change and one character. The column fitted the
// floor while two characters carried the busiest trait, so the skills column was
// exempted and this one was not noticed; a third carrier arrived with the cast's
// second origin and the line went to 88 cells. A column exempted by length
// rather than by kind is a column waiting for the next row.
func traitCarriers(lib *forge.Library) []string {
	out := make([]string, 0, len(lib.Passives().All()))
	for _, held := range lib.Passives().All() {
		if summary := forge.TraitCarrierSummary(lib.TraitCarriers(held.ID)); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

// pickerDetails is the detail column of whichever picker a screen is holding
// open, for every row in it.
//
// The third data column, exempt for the reason the other two are: the cell is a
// gloss, a species name, a work's title or a restriction, it is clipped rather
// than wrapped, and it spends the window rather than the floor, so measuring it
// against the floor asks a question the row does not answer. What holds it at
// the floor instead is clip itself — at eighty columns the cell is cut to fit by
// construction, which is the promise minWidth makes.
//
// ⚠️ It calls the screen's own detail rather than rebuilding one. A second
// reading would have to know the five kinds apart, and a data column enumerated
// by a copy of the thing that draws it is a column that goes quietly wrong the
// day a sixth kind is added — the same drift TestATraitNamesEveryStatusItsDescriptionNames
// exists to catch elsewhere. It is also what makes the entry honest: the status
// row is the one that overflowed when this column was widened, and it overflowed
// because its cell composes catalog wording around the book's figures, which is
// exactly whoMayCarry's shape.
func pickerDetails(m model) []string {
	if m.picker == nil {
		return nil
	}
	out := make([]string, 0, len(m.picker.options))
	for _, option := range m.picker.options {
		if detail := m.picker.detail(m, option.id); detail != "" {
			out = append(out, detail)
		}
	}
	return out
}

// carriesFreeText reports whether a line is showing authored text, which is not
// the program's to translate or to keep inside a column.
func carriesFreeText(line string, free []string) bool {
	for _, text := range free {
		if text != "" && strings.Contains(line, firstWords(text)) {
			return true
		}
	}
	return false
}

// partOfFreeText reports whether a line is one *wrapped* line of authored prose.
//
// carriesFreeText recognises a value by its opening, which is enough for text
// that is clipped — a biography on one row, cut at the window. It cannot see the
// second line of text that is **wrapped**, and the species note is the first
// authored prose in the tool that wraps: its opening is on one line and the rest
// on the next, so a name occurring in the tail was read as a name the program had
// looked up.
//
// The test is containment the other way round: a wrapped line is a run of the
// original's own words, so the original contains it. Short lines are refused
// because a two-word one would be contained by half the prose in the book, and
// the point of this is to exempt prose rather than to exempt everything narrow.
func partOfFreeText(line string, free []string) bool {
	const enoughToBeSure = 12
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < enoughToBeSure {
		return false
	}
	for _, text := range free {
		if strings.Contains(text, trimmed) {
			return true
		}
	}
	return false
}

// firstWords is enough of a free-text value to recognise it by after the line
// it sits on has been clipped to the window.
func firstWords(text string) string {
	if len(text) > 20 {
		return text[:20]
	}
	return text
}

// TestTheLanguageToggleKeepsWhatWasTyped is the property that makes the toggle
// worth having mid-form: comparing the two wordings must not cost the work.
func TestTheLanguageToggleKeepsWhatWasTyped(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "sentinel", "fire")
	before := m.form.draft()
	drawnBefore := m.screenContent()

	m = send(t, m, tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	if m.lang != i18n.En {
		t.Fatalf("ctrl+l left the language at %q", m.lang)
	}
	if m.screen != screenNew {
		t.Error("the toggle left the form")
	}
	after := m.form.draft()
	if after != before {
		t.Errorf("the toggle changed the draft:\nbefore %+v\nafter  %+v", before, after)
	}
	if m.form.cursor == 0 {
		t.Error("the toggle moved the cursor back to the first field")
	}
	drawnAfter := m.screenContent()
	if drawnAfter == drawnBefore {
		t.Error("the screen did not change language")
	}
	// The live carry check is re-worded rather than left in the old language,
	// because it is held as a fact and not as a sentence.
	if !strings.Contains(drawnAfter, `fire cannot carry the skill "riptide"`) {
		t.Errorf("the carry refusal did not follow the language:\n%s", drawnAfter)
	}

	// Back again, from a different screen, and with a question pending: the
	// toggle works everywhere ctrl+c does.
	m = send(t, m, tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	if m.lang != i18n.Vi {
		t.Errorf("the toggle does not go back, it is at %q", m.lang)
	}
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving an edited form did not ask")
	}
	m = send(t, m, tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	if m.guard == nil {
		t.Fatal("the toggle answered the pending question")
	}
	if !strings.Contains(m.screenContent(), "discard the character") {
		t.Errorf("the pending question did not follow the language:\n%s", m.screenContent())
	}
	if got := m.form.draft().ID; got != before.ID {
		t.Errorf("the id is now %q, want %q", got, before.ID)
	}
}

// TestTheLanguageComesFromTheFlagThenTheEnvironment covers how a run picks its
// language, including both ways of getting it wrong.
func TestTheLanguageComesFromTheFlagThenTheEnvironment(t *testing.T) {
	cases := []struct {
		name        string
		arguments   []string
		environment string
		want        i18n.Lang
	}{
		{"nothing given", nil, "", i18n.Vi},
		{"the environment alone", nil, "en", i18n.En},
		{"the flag alone", []string{"--lang", "en"}, "", i18n.En},
		{"the flag over the environment", []string{"--lang", "vi"}, "en", i18n.Vi},
		{"the flag over the environment, the other way", []string{"--lang", "en"}, "vi", i18n.En},
	}
	for _, test := range cases {
		got, err := parseOptions(test.arguments, test.environment, os.Stderr)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		if got.lang != test.want {
			t.Errorf("%s: the language is %q, want %q", test.name, got.lang, test.want)
		}
		if got.dir != forge.DefaultDataDir {
			t.Errorf("%s: the data directory is %q", test.name, got.dir)
		}
	}

	// An unusable value is refused rather than quietly replaced, and the
	// refusal says where it came from and what would have worked.
	refusals := []struct {
		name        string
		arguments   []string
		environment string
		wants       []string
	}{
		{"an unknown flag value", []string{"--lang", "vn"}, "", []string{"--lang", "vn", "vi", "en"}},
		{"an unknown environment value", nil, "english", []string{i18n.EnvVar, "english", "vi", "en"}},
		{"a bad flag with a good environment", []string{"--lang", "vn"}, "en", []string{"--lang", "vn"}},
	}
	for _, test := range refusals {
		_, err := parseOptions(test.arguments, test.environment, os.Stderr)
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		for _, want := range test.wants {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: the refusal %q does not mention %q", test.name, err, want)
			}
		}
	}

	// An argument that is not a flag is refused in the language that was asked
	// for, since by then it is known.
	_, err := parseOptions([]string{"nonsense"}, "", os.Stderr)
	if err == nil || !strings.Contains(err.Error(), "không nhận tham số") {
		t.Errorf("a stray argument gave %v, want the refusal in Vietnamese", err)
	}
	_, err = parseOptions([]string{"nonsense"}, "en", os.Stderr)
	if err == nil || !strings.Contains(err.Error(), "takes no arguments") {
		t.Errorf("a stray argument gave %v, want the refusal in English", err)
	}
}

// TestARefusedWriteIsWordedInTheLanguageInFront covers a per-field validation
// refusal end to end: the form's, not the catalog's, and reached by typing.
func TestARefusedWriteIsWordedInTheLanguageInFront(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	// A character that already exists. The refusal comes back from
	// forge.Draft.Resolve as a *forge.IDTakenError.
	m = author(t, m, "fixture-anime.adept", "Duplicate", "fixture-anime", "sentinel", "water/ice")
	m = key(t, m, "ctrl+s")
	if m.form.err == nil {
		t.Fatal("a character with a taken id was written")
	}
	drawn := m.screenContent()
	if want := `chưa ghi được: nhân vật "fixture-anime.adept" đã có trong danh sách rồi`; !strings.Contains(drawn, want) {
		t.Errorf("the refusal on screen is not %q:\n%s", want, drawn)
	}

	// A curve nobody can level into, typed into the health row: progression
	// decides it is wrong, the catalog says why in Vietnamese.
	fresh, _, _ := start(t, i18n.Vi)
	fresh = author(t, fresh, "fixture-film.tester", "Tester", "fixture-film", "duelist", "wind/ground")
	fresh = key(t, fresh, "down") // biography
	fresh = key(t, fresh, "down") // the health curve
	fresh = retype(t, fresh, "900:400")
	fresh = key(t, fresh, "ctrl+s")
	if fresh.form.err == nil {
		t.Fatal("a curve that shrinks with level was written")
	}
	want := "hp kết thúc ở 400 nhưng bắt đầu từ 900; chỉ số không tụt khi lên cấp"
	if !strings.Contains(fresh.screenContent(), want) {
		t.Errorf("the refusal on screen is not %q:\n%s", want, fresh.screenContent())
	}
}

// TestEveryWordingFitsTheMinimumWidth is the layout measurement, taken against
// the real screens in both languages rather than by eye.
//
// Vietnamese is the longer language, so this is where the minimum window width
// was decided: at 72 the busiest footers ran past the edge and were cut, which
// hides exactly the keys somebody stuck on a screen needs. Lines carrying free
// text from the data are skipped — a biography or a filesystem path has no
// length the program can promise, and frame clips those on purpose.
func TestEveryWordingFitsTheMinimumWidth(t *testing.T) {
	// The window's last column is left empty. A line that fills a terminal's
	// final cell wraps to the next row on some of them, and one wrapped line
	// pushes the footer off the bottom — the exact failure frame exists to
	// prevent.
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		base, lib, _ := start(t, lang)
		base.width, base.height = 200, 60
		free := append(freeText(lib), whoMayCarry(lang, lib)...)
		free = append(free, traitCarriers(lib)...)
		for name, m := range everyScreen(t, base) {
			m.width, m.height = 200, 60
			// The picker's detail column is per-screen, so it joins the two
			// listing columns here rather than in free: which cells exist
			// depends on which picker this screen has open.
			exempt := append(append([]string{}, free...), pickerDetails(m)...)
			for _, line := range strings.Split(m.screenContent(), "\n") {
				if carriesFreeText(line, exempt) {
					continue
				}
				if width := lipgloss.Width(line); width > drawable {
					t.Errorf("the %s screen in %s draws a line %d cells wide, over the %d it has:\n%s",
						name, lang, width, drawable, line)
				}
			}
		}
		// The too-small screen is measured against something smaller still,
		// since it is only ever drawn in a window that is already too narrow.
		small := base
		small.width, small.height = 40, 10
		for _, line := range strings.Split(small.screenContent(), "\n") {
			if width := lipgloss.Width(line); width > 24 {
				t.Errorf("the too-small screen in %s draws %d cells:\n%s", lang, width, line)
			}
		}
	}
}

// TestALongArtPathStaysInsideItsRow is the width risk the art chooser brought
// with it, measured rather than assumed.
//
// The other two choosers show an id out of a book, which is short by
// construction: "fixture-anime", "bulwark". This one shows a filesystem path,
// and a path is as long as whoever filed the art made it —
// assets/fixture/sprout.svg is 24 cells before anybody nests one folder deeper.
// So the row shortens from the front and keeps the file name, and that is what
// is asserted here: in both languages, since the label column is measured per
// language and Vietnamese leaves three cells fewer for the value.
//
// Two shapes, because the shortening has two steps. A long folder loses the
// folder. A file name too long on its own loses its own front, so that the name
// and the extension are the part that survives.
//
// ⚠️ **Asked at the floor, explicitly, and that is the point of it.** artRoom
// measures the window in hand — a path is data and data uses the window — so the
// row this draws is as wide as the terminal it is drawn on, and a test that took
// whatever width start happened to hand it would be measuring the fixture rather
// than the promise. Eighty is the width the program promises to draw in, so
// eighty is where "a path never runs past the edge" has to be true. The other
// half — that a wide window is actually spent, rather than the path being cut to
// seventy-nine beside a hundred empty columns — is
// aPathPartLongerThanTheFloor repeats a stem until the segment it builds is
// wider than the whole floor, so a path holding it cannot fit the art row at any
// window the tool draws at.
//
// The whole floor rather than the row's share of it: the row's arithmetic —
// marker, label column, the brackets — is what the test is measuring, and a
// fixture reproducing it would be that arithmetic written out a second time.
func aPathPartLongerThanTheFloor(stem, tail string) string {
	for repeat := 3; ; repeat++ {
		if built := strings.Repeat(stem, repeat) + tail; lipgloss.Width(built) > minWidth {
			return built
		}
	}
}

// TestAWideWindowWidensTheDataCells in width_rule_test.go.
func TestALongArtPathStaysInsideItsRow(t *testing.T) {
	const drawable = minWidth - 1
	// ⚠️ Both are repeated until the path alone clears the whole floor, rather
	// than a literal four each. Four cleared a floor of 80 and not one of 120, so
	// the row held the whole path, nothing was shortened, and the two assertions
	// below were about a value the screen was never going to cut.
	folder := aPathPartLongerThanTheFloor("deep-folder-", "end")
	longName := aPathPartLongerThanTheFloor("very-long-name-", "end.svg")
	for _, lang := range i18n.Langs() {
		dir := scratchData(t)
		if err := os.MkdirAll(filepath.Join(dir, "assets", folder), 0o755); err != nil {
			t.Fatalf("create the folder: %v", err)
		}
		for _, name := range []string{"hero.svg", longName} {
			if err := os.WriteFile(
				filepath.Join(dir, "assets", folder, name), []byte("<svg/>"), 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}
		base, _, _ := startIn(t, lang, dir)
		base.width, base.height = minWidth, minHeight

		cases := []struct {
			art   string
			shows []string
		}{
			// The folder goes and the file name stays whole.
			{"assets/" + folder + "/hero.svg", []string{ellipsis + "/hero.svg"}},
			// Nothing but the tail of the name fits, and the tail is the half
			// worth keeping: it is the end that holds the extension.
			{"assets/" + folder + "/" + longName, []string{ellipsis, "end.svg"}},
		}
		for _, test := range cases {
			m := base.enter(screenNew)
			for range fieldImage {
				m = key(t, m, "down")
			}
			m = chooseArt(t, m, test.art)
			row := strings.TrimRight(m.form.row(m, fieldImage, formLabelWidth(m)), "\n")
			if width := lipgloss.Width(row); width > drawable {
				t.Errorf("the %s art row is %d cells wide, over the %d it has:\n%s",
					lang, width, drawable, row)
			}
			for _, want := range test.shows {
				if !strings.Contains(row, want) {
					t.Errorf("the %s art row does not show %q:\n%s", lang, want, row)
				}
			}
			if strings.Contains(row, folder) {
				t.Errorf("the %s art row kept the whole folder, so nothing was shortened:\n%s",
					lang, row)
			}
			// The row is what was shortened, not the answer: a form that wrote
			// the path it drew would write a path no file has.
			if got := m.form.draft().Image; got != test.art {
				t.Errorf("the draft holds %q, want the whole path %q", got, test.art)
			}
		}
	}
}

// TestEveryLabelFitsItsFixedColumn holds the columns that are still a constant.
//
// The detail panes' and the menu's are not: they are measured from the labels
// being drawn, which is what TestTheDetailPanesMeasureTheirLabelColumn asserts
// and what a constant could not survive once "effective hp" and "nguồn tham
// khảo" existed. What is left here is the check screen's art cell, which holds
// one of two known words in each language rather than a label that can be
// reworded into anything.
func TestEveryLabelFitsItsFixedColumn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, key := range []i18n.Key{i18n.ArtPresent, i18n.ArtMissing} {
			if width := lipgloss.Width(lang.Text(key)); width > checkArtWidth {
				t.Errorf("the %s art column holds %q at %d cells", lang, lang.Text(key), width)
			}
		}
	}
	// The form's own column already measures itself, and its summary rows are
	// told that width rather than assuming the detail panes'. Both of those
	// labels have to fit it, or the two rows under the stats sit out of line.
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		width := formLabelWidth(m)
		for _, key := range []i18n.Key{i18n.LabelBudget, i18n.LabelCarries} {
			if measured := lipgloss.Width(lang.Text(key)); measured >= width {
				t.Errorf("the %s label %q is %d cells against the form's %d column",
					lang, lang.Text(key), measured, width)
			}
		}
	}
}

// TestTheDetailPanesMeasureTheirLabelColumn is the alignment property, asserted
// by reading the drawn screen rather than the number behind it.
//
// Every row of a pane puts its value in the same column, in each language, and
// the two languages land on different columns — which is the whole point of
// measuring: 11 was right for both only until it was right for neither. The kit
// gloss line is in the block being measured, so it is held to the same column
// as the ids above it.
func TestTheDetailPanesMeasureTheirLabelColumn(t *testing.T) {
	columns := make(map[i18n.Lang]int)
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		browse := m.enter(screenBrowse)
		body, _ := browse.browse.view(browse)
		rows := detailRows(t, body)
		if len(rows) < 6 {
			t.Fatalf("the %s detail pane drew %d rows:\n%s", lang, len(rows), body)
		}
		found := make(map[int][]string)
		for _, row := range rows {
			at := valueColumn(row)
			if at < 0 {
				t.Errorf("the %s pane drew a row with no value column:\n%q", lang, row)
				continue
			}
			found[at] = append(found[at], row)
		}
		if len(found) != 1 {
			t.Errorf("the %s pane starts its values in %d different columns:\n%s",
				lang, len(found), strings.Join(rows, "\n"))
		}
		for at := range found {
			columns[lang] = at
		}

		// The origins pane shares the column, which is what makes the panes line
		// up with each other rather than each with itself.
		note, _ := m.enter(screenOrigins).origins.view(m)
		for _, line := range strings.Split(note, "\n") {
			if !strings.Contains(line, lang.Text(i18n.LabelNote)) {
				continue
			}
			if at := valueColumn(line); at != columns[lang] {
				t.Errorf("the %s note row puts its value at %d, the browser at %d:\n%q",
					lang, at, columns[lang], line)
			}
		}
	}
	if columns[i18n.Vi] == columns[i18n.En] {
		t.Errorf("both languages put their values at column %d, so the width is not measured",
			columns[i18n.Vi])
	}
}

// detailRows is the "name  value" block a character's detail pane draws, which
// is everything after its heading. The heading is the one line in the pane that
// starts in the first column, so the rows are what follows the last of those.
func detailRows(t *testing.T, body string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	start := -1
	for i, line := range lines {
		if line != "" && !strings.HasPrefix(line, " ") {
			start = i
		}
	}
	if start < 0 {
		t.Fatalf("no detail heading in:\n%s", body)
	}
	var rows []string
	for _, line := range lines[start+1:] {
		if strings.TrimSpace(line) != "" {
			rows = append(rows, line)
		}
	}
	return rows
}

// valueColumn is the cell a row's value starts in, or -1 when the line is not
// one of these rows.
//
// A label may hold a space of its own — "nguồn tham khảo", "effective hp",
// "cấp 20" — so the value is not the second word. It is what follows the
// padding, and padding is two spaces or more: the widest label is padded to one
// past itself and then separated by another. A row that carries on from the one
// above has no label at all, so its whole prefix is that padding.
func valueColumn(line string) int {
	runes := []rune(line)
	if len(runes) < 4 || runes[0] != ' ' || runes[1] != ' ' {
		return -1
	}
	for at := 2; at+1 < len(runes); at++ {
		if runes[at] != ' ' || runes[at+1] != ' ' {
			continue
		}
		for at < len(runes) && runes[at] == ' ' {
			at++
		}
		if at >= len(runes) {
			return -1
		}
		return at
	}
	return -1
}

// TestNoScreenHoldsItsOwnWording is the rule that keeps the two languages
// honest: a sentence written here would exist in one of them only, and nothing
// would notice.
//
// The scan is over this package's own source. A string is treated as something
// a person would read when it holds two words of three letters or more with a
// space between them, or when it is a shouted word — those are the two shapes
// every line this program used to draw had. Format skeletons, key names and
// import paths have neither, which is why they can stay.
func TestNoScreenHoldsItsOwnWording(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		excused := environmentNames(file)
		ast.Inspect(file, func(node ast.Node) bool {
			literal, isLiteral := node.(*ast.BasicLit)
			if !isLiteral || literal.Kind != token.STRING || excused[literal.Pos()] {
				return true
			}
			text, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			if reason := readsLikeProse(text); reason != "" {
				t.Errorf("%s holds %q, which %s — put it in internal/i18n", name, text, reason)
			}
			return true
		})
	}
}

// environmentNames is the literals handed to os.Getenv.
//
// NO_COLOR and TERM are shouted words that nobody reads off a screen: they are
// the names of variables, and recognising them by where they are used rather
// than by a list means a new one needs no maintenance here.
func environmentNames(file *ast.File) map[token.Pos]bool {
	excused := make(map[token.Pos]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		selector, isSelector := call.Fun.(*ast.SelectorExpr)
		if !isSelector || selector.Sel.Name != "Getenv" {
			return true
		}
		for _, argument := range call.Args {
			if literal, isLiteral := argument.(*ast.BasicLit); isLiteral {
				excused[literal.Pos()] = true
			}
		}
		return true
	})
	return excused
}

// readsLikeProse says why a literal looks like something drawn for a person,
// or returns empty when it does not.
func readsLikeProse(text string) string {
	if strings.Contains(text, " ") && len(words(text)) >= 2 {
		return "reads like a sentence"
	}
	for _, word := range words(text) {
		if len(word) >= 4 && word == strings.ToUpper(word) {
			return "reads like a state shouted at the author"
		}
	}
	return ""
}

// words is the runs of three or more letters in a string, which is what tells
// prose apart from a format skeleton like "%-24s %-8s %s".
func words(text string) []string {
	var found []string
	current := strings.Builder{}
	flush := func() {
		if current.Len() >= 3 {
			found = append(found, current.String())
		}
		current.Reset()
	}
	for _, letter := range text {
		if unicode.IsLetter(letter) {
			current.WriteRune(letter)
			continue
		}
		flush()
	}
	flush()
	return found
}

// TestTheScreensGlossEveryDataName is the feature end to end: an id from a data
// file arrives on a Vietnamese screen with its Vietnamese name beside it, and on
// an English screen exactly as the file writes it.
//
// The expected strings come from internal/i18n rather than from a list here,
// because the point being asserted is that the screen asks for the gloss at all
// — a hand-kept list would pass while the browser drew the bare id. The one
// literal below is the format itself, which is the thing that has to be stable.
//
// ⚠️ **A stage name is deliberately not in this sweep, and must not be added.**
// This asserts that an id arrives *with its gloss*; a stage name has none, by
// decision — it is an identifier drawn exactly as the data writes it, in both
// languages, and there is no Lang.StageName to ask. Adding one here would assert
// the opposite of what was decided. What a stage name owes instead is that it can
// *be* an identifier, and that is refused upstream at the parser by
// progression.ValidateStageName, so a name this sweep could have caught never
// reaches a screen at all. That is why one shipped stage name was a Vietnamese
// phrase for as long as it was: the gap was never in this test's collection, it
// was that nothing anywhere said what a stage name may look like.
func TestTheScreensGlossEveryDataName(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	browse := m.enter(screenBrowse)
	for index, character := range browse.browse.rows() {
		browse.browse.cursor = index
		body, _ := browse.browse.view(browse)

		// Asserted against the row that is supposed to carry each gloss, not
		// against the screen: the element is glossed twice — in the list and in
		// the pane — so a whole-screen search passes with either one of them
		// gone. That was not a hypothetical; it let a mutation through.
		rows := detailRows(t, body)
		// The id is on the labelled row and the name is on the row under it, which
		// is the pane's one convention: the preset and the element are checked the
		// way the kit already was. Both halves are asserted, because a row holding
		// the id alone is what an unglossed preset looks like and would otherwise
		// pass.
		checks := []struct {
			label string
			id    string
			name  string
		}{
			{m.text(i18n.LabelPlaystyle), character.Archetype, i18n.Vi.Gloss(character.Archetype)},
			{m.text(i18n.LabelElement), character.Element.String(),
				i18n.Vi.AffinityNames(character.Element)},
		}
		for _, check := range checks {
			if check.name == "" {
				t.Errorf("%s has nothing glossed, so this proves nothing", character.ID)
				continue
			}
			if row := paneRow(t, rows, check.label); !strings.Contains(row, check.id) {
				t.Errorf("the %s row for %s is %q, want it to show %q",
					check.label, character.ID, row, check.id)
			}
			if under := rowUnder(t, rows, check.label); !strings.Contains(under, check.name) {
				t.Errorf("the row under %s's %s is %q, want it to show %q",
					character.ID, check.label, under, check.name)
			}
		}
		// The kit's names are under the kit's ids, in the same order.
		//
		// Searched for rather than taken as the very next row: the ids wrap over
		// as many rows as they need, and how many that is depends on the cast —
		// a tenth skill or a mark saying which forms may hold one both add a row
		// without changing what the screen owes. A long kit is clipped there, so
		// what is owed is the beginning of the reading rather than all of it;
		// that every skill is glossed at all is GlossedKit's property and is
		// checked on the function.
		kit := i18n.Vi.GlossedKit(lib.KitSkills(cast.LearnedIDs(character.Skills)))
		if kit == "" {
			t.Errorf("%s's kit is not glossed, so this proves nothing", character.ID)
		} else {
			opening := kit
			if runes := []rune(kit); len(runes) > 12 {
				opening = string(runes[:12])
			}
			if !rowsBelow(t, rows, m.text(i18n.LabelKit), opening) {
				t.Errorf("no row under %s's kit begins %q, in:\n%s",
					character.ID, opening, strings.Join(rows, "\n"))
			}
		}
		for _, carried := range lib.KitSkills(cast.LearnedIDs(character.Skills)) {
			if name := i18n.Vi.SkillName(carried); name == "" {
				t.Errorf("%s carries %s, which has no Vietnamese name", character.ID, carried.ID)
			}
		}
		// The element is glossed in the list as well as in the pane, and the
		// list row is the one that had to be measured to fit.
		row := listRow(t, body, character.ID)
		if want := i18n.Vi.GlossedAffinity(character.Element); !strings.Contains(row, want) {
			t.Errorf("the list row for %s does not show %q:\n%q", character.ID, want, row)
		}
	}

	// The shipped cast holds the pair this was asked for, in the format it was
	// asked for. Everything above would pass if the format changed; this would
	// not.
	browse.browse.cursor = 1
	sprout, _ := browse.browse.view(browse)
	for _, want := range []string{
		// The list still brackets, because a table column has no row underneath
		// to put a name on. Everything in the pane reads as an id with its name
		// dimmed under it, and the pair below is what that looks like for a dual
		// affinity: one row of ids, one row of names, positionally the same.
		"grass/electric <cỏ/điện>",
		"grass/electric",
		"cỏ/điện",
		"skirmisher",
		"du kích",
		"tia bắn · nanh độc · mục rữa · hồ quang",
	} {
		if !strings.Contains(sprout, want) {
			t.Errorf("the browser does not show %q:\n%s", want, sprout)
		}
	}

	// In English the same rows carry the ids alone. Every Vietnamese name of
	// every id the data holds is checked against every English screen, so a
	// gloss leaking into the wrong language is caught wherever it is drawn.
	//
	// ⚠️ The two halves below are not the same lookup, and collecting only the
	// first is how a leak survived: a compiled gloss comes out of i18n's own
	// tables and is empty in English by construction, while a **data** name —
	// a trait's, a species', a skill's — is a field on the declaration and is
	// Vietnamese whoever asks. Reading such a field raw put "bền bỉ · máu độc"
	// under an English traits row, and nothing here objected, because nothing
	// here knew the name existed.
	english, _, _ := start(t, i18n.En)
	english.width, english.height = 200, 60
	var names []string
	for _, character := range lib.Characters().All() {
		names = append(names, i18n.Vi.Gloss(character.Archetype))
		for _, member := range character.Element.Elements() {
			names = append(names, i18n.Vi.Gloss(member.String()))
		}
		for _, entry := range character.Skills {
			names = append(names, i18n.Vi.Gloss(entry.ID))
		}
		for _, held := range lib.KitPassives(character.PassivesAt(1, progression.Furthest)) {
			names = append(names, held.Name)
		}
		for _, kind := range lib.KitSpecies(character.Species) {
			names = append(names, kind.Name)
		}
	}
	// Line by line, skipping the lines that carry authored prose. What this is
	// hunting is a data name in a **column** — a row that resolved an id to its
	// Vietnamese word for an English reader, which is a wrong translation where
	// the bare id was the right answer. A name occurring *inside* authored prose
	// is not that: a species' note reads "chất rồng" and a biography may name a
	// trait, and neither is a lookup the program performed. Whole-screen matching
	// could not tell the two apart, so the first Vietnamese note put on screen
	// failed here for the wrong reason.
	free := freeText(lib)
	for name, screen := range everyScreen(t, english) {
		screen.width, screen.height = 200, 60
		drawn := screen.screenContent()
		for _, line := range strings.Split(drawn, "\n") {
			if carriesFreeText(line, free) || partOfFreeText(line, free) {
				continue
			}
			for _, unwanted := range names {
				if unwanted != "" && strings.Contains(line, unwanted) {
					t.Errorf("the %s screen in English holds the gloss %q:\n%s",
						name, unwanted, drawn)
				}
			}
		}
	}
	englishBrowse := english.enter(screenBrowse)
	body, _ := englishBrowse.browse.view(englishBrowse)
	for _, want := range []string{"playstyle     sentinel", "element       water/ice"} {
		if !strings.Contains(body, want) {
			t.Errorf("the English browser does not draw %q:\n%s", want, body)
		}
	}
	// Walked per character rather than left on the first, and that is the whole
	// of why this bites: the cursor opens on a fixture that has no traits and no
	// species, so a data name leaking into either row was drawn on a pane nobody
	// asked to see. Every character is asked, and only the ones that have
	// something to leak prove anything — which is what the count below insists
	// on.
	asked := 0
	for index, character := range englishBrowse.browse.rows() {
		leaks := make([]string, 0, 4)
		for _, held := range lib.KitPassives(character.PassivesAt(1, progression.Furthest)) {
			leaks = append(leaks, held.Name)
		}
		for _, kind := range lib.KitSpecies(character.Species) {
			leaks = append(leaks, kind.Name)
		}
		if len(leaks) == 0 {
			continue
		}
		asked++
		englishBrowse.browse.cursor = index
		drawn, _ := englishBrowse.browse.view(englishBrowse)
		for _, unwanted := range leaks {
			if unwanted != "" && strings.Contains(drawn, unwanted) {
				t.Errorf("the English browser holds the data name %q for %s:\n%s",
					unwanted, character.ID, drawn)
			}
		}
	}
	if asked == 0 {
		t.Error("no character in the cast has a trait or a species, so the English pane proves nothing")
	}
}

// paneRow is the detail row a label names, and rowUnder is the one after it.
func paneRow(t *testing.T, rows []string, label string) string {
	t.Helper()
	return rows[paneRowIndex(t, rows, label)]
}

// rowsBelow reports whether any row after the labelled one contains the text.
//
// The pane wraps a long value over several rows, so "under" is a region rather
// than a single line — and which line a reading lands on is a property of the
// data's length, which is exactly what a layout test should not be pinned to.
func rowsBelow(t *testing.T, rows []string, label, want string) bool {
	t.Helper()
	at := paneRowIndex(t, rows, label)
	for _, row := range rows[at+1:] {
		if strings.Contains(row, want) {
			return true
		}
	}
	return false
}

func rowUnder(t *testing.T, rows []string, label string) string {
	t.Helper()
	at := paneRowIndex(t, rows, label)
	if at+1 >= len(rows) {
		t.Fatalf("nothing follows the %q row in:\n%s", label, strings.Join(rows, "\n"))
	}
	return rows[at+1]
}

func paneRowIndex(t *testing.T, rows []string, label string) int {
	t.Helper()
	for i, row := range rows {
		if strings.HasPrefix(row, "  "+label+" ") {
			return i
		}
	}
	t.Fatalf("no %q row in:\n%s", label, strings.Join(rows, "\n"))
	return -1
}

// listRow is the browser's own row for a character, as opposed to the detail
// pane below, which names it again.
func listRow(t *testing.T, body, id string) string {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimLeft(line, "> ")
		if strings.HasPrefix(trimmed, id+" ") {
			return line
		}
	}
	t.Fatalf("no list row for %s in:\n%s", id, body)
	return ""
}

// TestEveryGlossFitsItsRow is the width measurement for the rows a gloss
// lengthened, taken over every character and every preset rather than over the
// one character the browser happens to open on.
//
// The kit is the row that decides this. Five skills is the longest kit the
// presets ship — the duelist's — and five Vietnamese names of two or three
// words each is what pushed the gloss onto its own line instead of into five
// brackets beside the ids.
func TestEveryGlossFitsItsRow(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, lib, _ := start(t, lang)
		width := detailLabelWidth(m)
		// A detail row is two spaces of indent, the label column, and a space.
		indent := 2 + width + 1

		kits := make(map[string][]string)
		for _, character := range lib.Characters().All() {
			kits[character.ID] = cast.LearnedIDs(character.Skills)
		}
		for _, preset := range lib.Archetypes().All() {
			kits[preset.ID] = preset.Skills
		}
		for id, skills := range kits {
			glossed := lang.GlossedKit(lib.KitSkills(skills))
			if glossed == "" {
				continue
			}
			// Measured as it is *drawn*, not as it is built. A kit has no fixed
			// size, so the raw reading grows without bound -- six Vietnamese
			// names come to 84 cells -- and the row clips it. Holding the raw
			// string to the width would make every sixth skill an obstacle to
			// authoring rather than a layout bug.
			if drawn := indent + lipgloss.Width(clip(glossed, drawable-indent)); drawn > drawable {
				t.Errorf("%s's kit gloss in %s draws %d cells, over the %d there are: %q",
					id, lang, drawn, drawable, glossed)
			}
		}

		for _, character := range lib.Characters().All() {
			for _, row := range []string{
				lang.Glossed(character.Archetype),
				lang.GlossedAffinity(character.Element),
			} {
				if drawn := indent + lipgloss.Width(row); drawn > drawable {
					t.Errorf("%s draws %q at %d cells in %s, over the %d there are",
						character.ID, row, drawn, lang, drawable)
				}
			}
			// The list row is the tighter of the two: two fixed columns come
			// before the element, and the gloss has what is left.
			list := 2 + browseIDWidth + 1 + browseOriginWidth + 1 +
				lipgloss.Width(lang.GlossedAffinity(character.Element))
			if list > drawable {
				t.Errorf("%s's list row in %s draws %d cells, over the %d there are",
					character.ID, lang, list, drawable)
			}
		}
	}
}

// saysWord reports whether a screen says a word, rather than merely holding its
// letters somewhere inside a longer one.
func saysWord(text, word string) bool {
	boundary := regexp.MustCompile(`(^|[^\p{L}])` + regexp.QuoteMeta(word) + `($|[^\p{L}])`)
	return boundary.MatchString(text)
}

// TestTheRenamedLabelsSayTheNewThing holds the four wordings that were changed,
// and holds the old ones gone.
//
// Both halves matter. A screen still drawing the old label would be caught by
// the first, and a rename applied to the catalog but not to the screen that asks
// for it would be caught by the second.
func TestTheRenamedLabelsSayTheNewThing(t *testing.T) {
	cases := []struct {
		lang    i18n.Lang
		nowSays []string
		gone    []string
	}{
		{i18n.Vi,
			[]string{"danh sách nhân vật", "nguồn tham khảo", "lối chơi", "máu quy đổi"},
			// "dàn" is the whole word that "danh sách" replaced. It is matched as
			// a word and not as a substring, and that is not fussiness: "để dành
			// cho", which is how a restricted skill says who it is kept for,
			// holds those three letters with that tone and is a perfectly
			// ordinary thing to say. A substring test would have banned it.
			[]string{"dàn", "dựa trên", "chịu được"}},
		{i18n.En,
			[]string{"playstyle", "effective hp"},
			[]string{"tuned from", "absorbs"}},
	}
	for _, test := range cases {
		base, _, _ := start(t, test.lang)
		base.width, base.height = 200, 60
		said := make(map[string]bool)
		for name, screen := range everyScreen(t, base) {
			screen.width, screen.height = 200, 60
			drawn := screen.screenContent()
			for _, unwanted := range test.gone {
				if saysWord(drawn, unwanted) {
					t.Errorf("the %s screen in %s still says %q:\n%s",
						name, test.lang, unwanted, drawn)
				}
			}
			for _, wanted := range test.nowSays {
				if strings.Contains(drawn, wanted) {
					said[wanted] = true
				}
			}
		}
		for _, wanted := range test.nowSays {
			if !said[wanted] {
				t.Errorf("no screen in %s says %q", test.lang, wanted)
			}
		}
	}

	// The refusal is worded from the same term, so the two cannot part company.
	taken := &forge.IDTakenError{ID: "fixture-anime.adept"}
	if got, want := i18n.Vi.Error(taken),
		`nhân vật "fixture-anime.adept" đã có trong danh sách rồi`; got != want {
		t.Errorf("the refusal reads %q, want %q", got, want)
	}
}

// wholeSkillList is the listing rendered from the top and from the bottom.
//
// The view is a window around the cursor, so a single render only ever shows
// the skills near it. The glossed skills this test names are fixture ones,
// which sort after every shipped skill: reading one render made the assertion
// depend on how many skills happened to ship above them, and it broke the day
// another character's kit landed. What is being measured is the gloss column,
// not the length of the book.
func wholeSkillList(m model) string {
	entered := m.enter(screenSkills)
	top, _ := entered.skills.view(entered)
	tail := entered.skills
	tail.cursor = len(tail.skills) - 1
	bottom, _ := tail.view(entered)
	return top + "\n" + bottom
}

// TestTheSkillListNamesSkillsInVietnamese covers the translated-name column and,
// more importantly, that it disappears rather than emptying when nothing is
// glossed. A column of blanks reads as missing data; no column reads as a column
// that does not apply, which is the truth in English.
func TestTheSkillListNamesSkillsInVietnamese(t *testing.T) {
	vietnamese, _, _ := start(t, i18n.Vi)
	body := wholeSkillList(vietnamese)
	for _, want := range []string{i18n.Vi.Text(i18n.ColumnGloss), "đòn đánh", "cắt lìa"} {
		if !strings.Contains(body, want) {
			t.Errorf("the Vietnamese skill list is missing %q", want)
		}
	}

	english, _, _ := start(t, i18n.En)
	body = wholeSkillList(english)
	for _, unwanted := range []string{i18n.En.Text(i18n.ColumnGloss), "đòn đánh", "cắt lìa"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("the English skill list should not carry %q", unwanted)
		}
	}
	// That the column is *gone* rather than blank is pinned by
	// TestSkillRowDropsTheGlossColumnWhenItIsEmpty, which measures the row
	// instead of guessing at runs of spaces — the id column pads too, so a
	// space-run heuristic here flags correct output.
}

// TestSkillRowDropsTheGlossColumnWhenItIsEmpty pins the rule itself, so it
// cannot be lost when the caller that measures the width changes.
func TestSkillRowDropsTheGlossColumnWhenItIsEmpty(t *testing.T) {
	with := skillRow(8, 6, 8, "strike", "đòn", "neutral", "1000x1", "anyone")
	without := skillRow(8, 0, 8, "strike", "đòn", "neutral", "1000x1", "anyone")
	if strings.Contains(without, "đòn") {
		t.Errorf("a zero gloss column still drew the gloss: %q", without)
	}
	if !strings.Contains(with, "đòn") {
		t.Errorf("a sized gloss column dropped the gloss: %q", with)
	}
	if lipgloss.Width(without) >= lipgloss.Width(with) {
		t.Errorf("dropping the column did not narrow the row: %d vs %d",
			lipgloss.Width(without), lipgloss.Width(with))
	}
}

// TestTheAccuracyRowReadsAsAPercentage covers the reason the row exists: 850 is
// what the engine divides by, and nobody reads it as a chance. The parts per
// thousand stay on screen — the number written to the file has to be the number
// shown — with the percentage beside them.
func TestTheAccuracyRowReadsAsAPercentage(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	screen := m.enter(screenSkills)
	screen.skills.adding = true
	screen.skills.inputs[skillFieldAccuracy].SetValue("850")
	if got := screen.skills.value(screen, skillFieldAccuracy, 16); !strings.Contains(got, "850") ||
		!strings.Contains(got, "85%") {
		t.Errorf("the accuracy row shows %q, want both 850 and 85%%", got)
	}

	// Power reads the same way, and a zero one says nothing a "0%" could add:
	// four shipped skills declare no power at all.
	screen.skills.inputs[skillFieldPower].SetValue("2200")
	if got := screen.skills.value(screen, skillFieldPower, 16); !strings.Contains(got, "220%") {
		t.Errorf("the power row shows %q, want 220%%", got)
	}
	screen.skills.inputs[skillFieldPower].SetValue("0")
	if got := screen.skills.percentHint(screen, skillFieldPower); got != "" {
		t.Errorf("a zero power produced the hint %q, want none", got)
	}

	// The chances in the inflicts field are parts per thousand too, but the
	// field holds the syntax ParseApplications reads, so the reading goes
	// beside it rather than into it.
	screen.skills.inputs[skillFieldInflicts].SetValue("weaken:800,blind:400")
	if got := screen.skills.value(screen, skillFieldInflicts, 16); !strings.Contains(got, "80%") ||
		!strings.Contains(got, "40%") {
		t.Errorf("the inflicts row shows %q, want both 80%% and 40%%", got)
	}
	// A list being typed is unparseable most of the time; that is not an error
	// to announce.
	for _, partial := range []string{"weaken", "weaken:", "weaken:8x"} {
		screen.skills.inputs[skillFieldInflicts].SetValue(partial)
		if got := screen.skills.chanceHint(screen, 16); got != "" {
			t.Errorf("a value of %q produced the chance hint %q, want none", partial, got)
		}
	}

	// A half-typed number is the normal state of a text field, not an error to
	// announce, so the hint says nothing at all rather than guessing.
	for _, partial := range []string{"", "-", "8x"} {
		screen.skills.inputs[skillFieldAccuracy].SetValue(partial)
		if got := screen.skills.percentHint(screen, skillFieldAccuracy); got != "" {
			t.Errorf("a value of %q produced the hint %q, want none", partial, got)
		}
	}
}

// TestEveryFieldOfTheSkillFormHasHelp is the reason the static footnote went:
// fourteen fields and one sentence about two of them explained nothing about
// the twelve nobody could guess.
//
// Three properties, and the third is the one a wording test alone would miss:
// every field draws a line, the line changes as the cursor moves, and it is
// really on the drawn screen rather than only in the catalog.
func TestEveryFieldOfTheSkillFormHasHelp(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		base.width, base.height = minWidth, minHeight
		m := base.enter(screenSkills)
		m.skills.adding = true

		seen := make(map[string]int, skillFieldCount)
		for field := range skillFieldCount {
			m = skillFormTo(t, m, field)
			help := skillFieldHelp(m, field)
			if strings.TrimSpace(help) == "" {
				t.Errorf("field %d has no help line in %s", field, lang)
				continue
			}
			// The help is a sentence in a screen-wide row, so it is measured
			// against the whole drawable width rather than against a column.
			if width := lipgloss.Width("  " + help); width > drawable {
				t.Errorf("the %s help for field %d is %d cells wide, over the %d it has:\n%s",
					lang, field, width, drawable, help)
			}
			if before, clash := seen[help]; clash {
				t.Errorf("fields %d and %d share the %s help line %q",
					before, field, lang, help)
			}
			seen[help] = field
			body, _ := m.skills.view(m)
			if !strings.Contains(body, help) {
				t.Errorf("the %s form does not draw the help for field %d:\n%s",
					lang, field, body)
			}
			// A help line that says nothing but the label again is the footnote
			// back: it has to be longer than the name of the field it explains.
			if label := skillFieldLabel(m, field); lipgloss.Width(help) <= lipgloss.Width(label) {
				t.Errorf("the %s help for %q is no longer than its own label: %q",
					lang, label, help)
			}
		}
		if len(seen) != skillFieldCount {
			t.Errorf("%s drew %d distinct help lines over %d fields",
				lang, len(seen), skillFieldCount)
		}
	}
}

// TestTheSkillFormFitsTheSmallestWindow is the form's half of what
// TestTheSkillListingFitsTheSmallestWindowAfterAnEdit measures for the listing.
//
// The busiest state, because that is the only one that measures anything: every
// field prefilled from a skill already in the book, and a refused write under
// them, which is the extra line an error costs. The form spent nineteen of the
// twenty body lines a 120x24 window has before the help line existed and it
// still spends nineteen, because the help replaced the footnote rather than
// joining it — there is no spare line here, which is why the shape diagram is a
// sub-screen and not a pane.
func TestTheSkillFormFitsTheSmallestWindow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		m.skills = m.skills.prefill(m.lib, m.skills.skills[0])
		m.skills.err = &forge.MissingSkillIDError{}
		drawn := m.screenContent()
		if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
			strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
			t.Errorf("the %s skill form is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		// And the count itself, so that a line added to this screen fails here
		// with the arithmetic rather than only through the truncation marker.
		body, _ := m.skills.view(m)
		if lines, room := len(strings.Split(body, "\n")), minHeight-4; lines > room {
			t.Errorf("the %s skill form draws %d body lines against the %d it has",
				lang, lines, room)
		}
	}
}

// TestTheListingHeaderLinesUpWithItsRows is what the power column's width is
// for, and it is measured on the drawn screen rather than on the number behind
// it.
//
// The header names its columns with the labels the form authored them with, so
// renaming a field renames a column here — and the power column was a constant 8
// while its header was the word "power". A 17-cell "damage multiplier" in an
// 8-cell column does not overflow anything, because pad only widens: it pushes
// the last column's header nine cells right of the rows it names, which is a
// table that lies about which numbers are under which word.
func TestTheListingHeaderLinesUpWithItsRows(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		body, _ := m.skills.view(m)
		lines := strings.Split(body, "\n")

		// The comparable row is an unrestricted one, because its last cell is
		// short enough to be drawn whole: every other cell in that column can
		// end in an ellipsis, and an offset measured to a clip is measured to
		// nothing. Which page holds one is data, not layout — the first page did
		// until every skill in the book was given the work it comes out of — so
		// the cursor walks the listing until a screen has both.
		header, row := "", ""
		for step := 0; step <= len(m.skills.skills); step++ {
			for _, line := range lines {
				switch {
				case strings.Contains(line, lang.Text(i18n.ColumnWhoMayCarry)):
					header = line
				// Any row but the cursor's, so the marker is the same two cells
				// as the header's and the offsets are comparable.
				case row == "" && strings.HasPrefix(line, "  ") &&
					strings.Contains(line, lang.Text(i18n.WhoAnyone)):
					row = line
				}
			}
			if header != "" && row != "" {
				break
			}
			m = key(t, m, "down")
			body, _ = m.skills.view(m)
			lines = strings.Split(body, "\n")
		}
		if header == "" || row == "" {
			t.Fatalf("the %s listing never draws a header and an unrestricted row together:\n%s",
				lang, body)
		}
		// Cell offsets, not byte offsets: a Vietnamese header is multi-byte and
		// strings.Index would compare two different units.
		cellAt := func(line, needle string) int {
			return lipgloss.Width(line[:strings.Index(line, needle)])
		}
		if got, want := cellAt(header, lang.Text(i18n.ColumnWhoMayCarry)),
			cellAt(row, lang.Text(i18n.WhoAnyone)); got != want {
			t.Errorf("the %s header's last column starts at %d and its rows' at %d:\n%s\n%s",
				lang, got, want, header, row)
		}
		// And the power column really is named after the field that authored it,
		// which is the coupling the width exists to keep affordable.
		if !strings.Contains(header, lang.Text(i18n.SkillFieldPower)) {
			t.Errorf("the %s header does not name the power column %q:\n%s",
				lang, lang.Text(i18n.SkillFieldPower), header)
		}
		for _, line := range lines {
			if width := lipgloss.Width(line); width > drawable {
				t.Errorf("the %s listing draws %d cells, over the %d it has:\n%s",
					lang, width, drawable, line)
			}
		}
	}
}

// TestEveryFormRowFitsTheWindowAtItsBusiest is the row arithmetic, measured
// against the widest content each row can hold rather than against the empty
// form every other test happens to render.
//
// This is what caught fieldValueRoom being a cell short. The two rows with a
// part that has no length of its own — the chances beside the inflicts field and
// the ids of an allowlist — computed what was left from the text field's
// declared Width, and a bubbles text field draws a cell more than that for its
// cursor. Both rows came to exactly 80 cells in an 80-column window: inside
// frame's clip, so nothing looked wrong, and over the edge of the terminals that
// wrap a line filling the final cell.
func TestEveryFormRowFitsTheWindowAtItsBusiest(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		m.skills.adding = true
		// The busiest each row can be: a full-width id, three statuses read out
		// beside the field, and three long ids in every allowlist.
		m.skills.inputs[skillFieldID].SetValue(strings.Repeat("a", 40))
		// A power high enough that the damage row under the fields carries
		// four-digit figures, which is the case that row is measured on: it is
		// the widest fixed row on this screen and it grows with the number of
		// digits in its own numbers, so an empty power — the only state every
		// other test renders it in — measures nothing. The shipped book tops out
		// at 2200, which is three digits; this is past it on purpose.
		//
		// It has a real ceiling above this: five-digit damage, which needs a
		// power around 15000, runs the row two cells over and is clipped by
		// frame. Nothing shipped is within a factor of six of that.
		m.skills.inputs[skillFieldPower].SetValue("3000")
		m.skills.inputs[skillFieldAccuracy].SetValue("1000")
		m.skills.inputs[skillFieldInflicts].SetValue("poison:300,burn:500,weaken:1000")
		long := []string{"example-hero", "example-sprout", "example-adept"}
		m.skills.keptElements, m.skills.keptRoles, m.skills.keptWho = long, long, long

		width := skillLabelWidth(m)
		for field := range skillFieldCount {
			m.skills.field = field
			row := "  " + pad(skillFieldLabel(m, field), width) + " " +
				m.skills.value(m, field, width)
			if measured := lipgloss.Width(row); measured > drawable {
				t.Errorf("the %s row for field %d is %d cells wide, over the %d it has:\n%s",
					lang, field, measured, drawable, row)
			}
		}
		// And the whole screen in that state, since the rows are not the only
		// lines on it.
		body, _ := m.skills.view(m)
		for _, line := range strings.Split(body, "\n") {
			if measured := lipgloss.Width(line); measured > drawable {
				t.Errorf("the %s form draws %d cells, over the %d it has:\n%s",
					lang, measured, drawable, line)
			}
		}
	}
}

// TestAnAllowlistPickerSaysWhatAnEmptyListMeans covers a hint that used to be
// borrowed from the kit. The kit's hint talks about the order of a selection and
// what this character cannot take; on an allowlist neither is true, and what the
// author needs to know instead is that leaving it empty lets anyone carry it.
func TestAnAllowlistPickerSaysWhatAnEmptyListMeans(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		m, _, _ := start(t, lang)
		for _, field := range []int{
			skillFieldKeptForElements, skillFieldKeptForRoles, skillFieldKeptForCharacters,
		} {
			opened := m.enter(screenSkills).openAllowlist(field)
			body, _ := opened.picker.view(opened)
			if want := lang.Text(i18n.PickerAllowlistHint); !strings.Contains(body, want) {
				t.Errorf("%v field %d is missing the allowlist hint %q", lang, field, want)
			}
			if unwanted := lang.Text(i18n.PickerHint); strings.Contains(body, unwanted) {
				t.Errorf("%v field %d still borrows the kit's hint", lang, field)
			}
		}
	}
}

// TestTheSquadPickersSayWhatTheirOwnListsAre is the allowlist's defect one
// screen further on, and it was wrong in the same shape for the same reason.
//
// Both builder pickers borrowed the form's kit hint, which names a ! for a row
// the character cannot take. The form is choosing out of the whole skill book,
// so that mark is real there; the builder's rows come out of the learnset and
// carry no refusal at all, so the hint named a mark neither list can draw. The
// trait half was wrong twice over — it called a trait a skill, and there is one
// slot, so an order says nothing about it.
//
// The refusal is asserted beside the wording rather than left to it. A hint is a
// sentence and a sentence can be made true by editing it; what says the sentence
// stays true is that the rows behind it hold nothing to mark.
func TestTheSquadPickersSayWhatTheirOwnListsAre(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenSquads)
		m.squad = someSquad(t, m)
		member := m
		member.squad = member.squad.editUnit(0)
		holder, holds := aTraitHolder(m)
		if !holds {
			t.Skip("no character in the book learns a trait, so the trait list has no rows")
		}
		for _, list := range []struct {
			what   string
			opened model
			hint   i18n.Key
		}{
			{"kit", member.openSquadSkills(), i18n.SquadKitHint},
			{"trait", holder.openSquadPassives(), i18n.SquadTraitHint},
		} {
			if list.opened.picker == nil || len(list.opened.picker.options) == 0 {
				t.Fatalf("the %s %s field raised no picker with rows in it", lang, list.what)
			}
			body, _ := list.opened.picker.view(list.opened)
			if want := lang.Text(list.hint); !strings.Contains(body, want) {
				t.Errorf("the %s %s picker is missing its own hint %q:\n%s",
					lang, list.what, want, body)
			}
			if unwanted := lang.Text(i18n.PickerHint); strings.Contains(body, unwanted) {
				t.Errorf("the %s %s picker still borrows the form's kit hint", lang, list.what)
			}
			for _, option := range list.opened.picker.options {
				if option.refusal != nil {
					t.Errorf("the %s %s picker refuses %s, so the mark its hint no longer names is reachable",
						lang, list.what, option.id)
				}
			}
		}
	}
}

// TestTheFormScrollsToTheFieldTheCursorIsOn covers the window rather than the
// fields: the form outgrew a 120x24 window when healing added three answers, so
// what has to hold is that tabbing to the last one brings it into view.
func TestTheFormScrollsToTheFieldTheCursorIsOn(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		m, _, _ := start(t, lang)
		form := m.enter(screenSkills)
		form.skills.adding = true
		// Every field, not a chosen three. The help line under the form is the
		// *focused* field's, so a wording is only measured on the state that
		// focuses it — a field left out of this loop is a wording nothing in the
		// suite has ever drawn at the minimum width.
		for field := 0; field < skillFieldCount; field++ {
			form.skills.field = field
			body, _ := form.skills.view(form)
			label := skillFieldLabel(form, field)
			if !strings.Contains(body, label) {
				t.Errorf("%v: the cursor is on %q and the form does not draw it:\n%s",
					lang, label, body)
			}
			for _, line := range strings.Split(body, "\n") {
				if lipgloss.Width(line) >= minWidth {
					t.Errorf("%v: a row is %d cells, over the %d there are: %q",
						lang, lipgloss.Width(line), minWidth-1, line)
				}
			}
			if got := len(strings.Split(body, "\n")); got > form.height-4 {
				t.Errorf("%v: the body is %d rows, over the %d it has",
					lang, got, form.height-4)
			}
		}
	}
}

// TestNoDetailRowIsCutOff measures every row the character pane draws, at the
// floor and at a wider window.
//
// It exists because three of its rows have no bound on their length -- a kit, its
// reading, and a biography -- and they were being clipped by the frame, which
// takes the tail off silently. A row cut at the window is indistinguishable from
// a row that ends there.
func TestNoDetailRowIsCutOff(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		// ⚠️ Every width here has to be one the tool actually draws at. The list
		// read {minWidth, 100, 140} while the floor was 80; at a floor of 120 the
		// hundred is *under* it, so usableWidth stood the floor in for it and the
		// rows were correctly 120 cells wide in a window m.tooSmall refuses to
		// draw at all. The failure was the fixture naming a window, not the row.
		for _, width := range []int{minWidth, minWidth + 20, minWidth + 60} {
			m, lib, _ := start(t, lang)
			m = send(t, m, tea.WindowSizeMsg{Width: width, Height: 40})
			browse := m.enter(screenBrowse)
			for index := range lib.Characters().All() {
				browse.browse.cursor = index
				body, _ := browse.browse.view(browse)
				for _, line := range strings.Split(body, "\n") {
					if drawn := lipgloss.Width(line); drawn > width {
						t.Errorf("%v at %d cells draws a %d-cell row: %q",
							lang, width, drawn, line)
					}
				}
			}
		}
	}
}

// TestAWrappedRowLinesUpUnderItsValue is the property that makes wrapping worth
// having over clipping: the tail has to read as a continuation rather than as a
// new row with a missing label.
func TestAWrappedRowLinesUpUnderItsValue(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = send(t, m, tea.WindowSizeMsg{Width: minWidth, Height: 40})
	const width = 10
	drawn := m.wrapped("nhãn", width, strings.Repeat("một hai ba bốn năm ", 8))
	lines := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("nothing wrapped: %q", drawn)
	}
	// Measured as an indent rather than by finding a word: a continuation starts
	// with whatever word happened to fall there, and the styles put escape codes
	// in front of the text, so a byte index is not a column.
	indent := func(line string) int {
		bare := ansi.Strip(line)
		return lipgloss.Width(bare) - lipgloss.Width(strings.TrimLeft(bare, " "))
	}
	// The first line's indent is the body's own two: its label sits there. Every
	// continuation is indented past the label column instead, which is the column
	// the value started in — two, plus the label width, plus the space between.
	want := 2 + width + 1
	for i, line := range lines[1:] {
		if got := indent(line); got != want {
			t.Errorf("continuation %d is indented %d, want the value column %d: %q",
				i, got, want, line)
		}
	}
	// A word longer than the room keeps its own line rather than being halved.
	long := m.wrapped("nhãn", width, "một "+strings.Repeat("x", 200))
	if !strings.Contains(long, strings.Repeat("x", 200)) {
		t.Error("a word longer than the room was cut instead of overflowing")
	}
}

// TestAStyledWrappedRowDoesNotEatTheRowBelow is a regression, and the shape of
// the bug is worth keeping: styling a whole wrapped block instead of each of its
// lines made lipgloss treat it as a box, pad every line out to the widest and
// swallow the trailing newline — so the row after it was appended to the end of a
// field of spaces and vanished from the screen, taking its own styling with it.
//
// Asserted structurally rather than by looking for escape codes, because
// lipgloss disables colour when the output is not a terminal and a test never
// has one: what has to hold is that every row is still drawn, on its own line.
func TestAStyledWrappedRowDoesNotEatTheRowBelow(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = send(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	browse := m.enter(screenBrowse)
	body, _ := browse.browse.view(browse)
	rows := strings.Split(body, "\n")

	// The kit's reading is the styled wrapped row; the art row is what used to
	// disappear behind it.
	for _, label := range []i18n.Key{
		i18n.LabelFrom, i18n.LabelPlaystyle, i18n.LabelElement, i18n.LabelKit,
		i18n.LabelArt, i18n.LabelStages, i18n.LabelBiography, i18n.LabelEffectiveHP,
	} {
		want := i18n.Vi.Text(label)
		found := false
		for _, row := range rows {
			if strings.HasPrefix(strings.TrimSpace(row), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the pane has no row for %q:\n%s", want, body)
		}
	}
	// And no row is a run of spaces: that is what the swallowed newline left
	// behind, and it reads as a gap rather than as a bug.
	for i, row := range rows {
		if row != "" && strings.TrimSpace(row) == "" {
			t.Errorf("row %d is %d cells of nothing", i, lipgloss.Width(row))
		}
	}
	_ = lib
}
