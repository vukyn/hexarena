package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheSquadCatalogueRaisesTheFight is the wiring: the fight is reached from
// where a squad is under a cursor, not from the menu, so pressing the key is the
// only way to find out that it is.
func TestTheSquadCatalogueRaisesTheFight(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	// Nothing built yet, so the key does nothing rather than opening a screen
	// with no subject.
	m = typeText(t, m, "f")
	if m.screen != screenSquads {
		t.Fatalf("f opened %v with nothing to fight", m.screen)
	}
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	// Back out to the catalogue, which is where the fight is raised from: the
	// builder leaves the squad open, and f in a name field is a letter.
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	if m.screen != screenFight {
		t.Fatalf("f opened %v", m.screen)
	}
	// esc goes back where it came from, which is the catalogue and not the menu.
	if back := key(t, m, "esc"); back.screen != screenSquads {
		t.Errorf("esc left the fight for %v", back.screen)
	}
}

// TestAFightAgainstACopyIsTheControl is the figure the screen exists to draw,
// and the mirror is the one pairing whose answer is known in advance.
func TestAFightAgainstACopyIsTheControl(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	// Back out to the catalogue, which is where the fight is raised from: the
	// builder leaves the squad open, and f in a name field is a letter.
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	report, failure, ok := m.fight.report(m)
	if !ok || failure != nil {
		t.Fatalf("the run did not happen: %v", failure)
	}
	if !report.Mirror() {
		t.Fatal("one squad against itself is not reporting as a mirror")
	}
	if report.Rate() != 500 {
		t.Errorf("the control reads %d per mille, want 500 exactly", report.Rate())
	}
	body := m.screenContent()
	// The control says it is one, and the caution says what the figure is not.
	for _, line := range []i18n.Key{i18n.FightControl, i18n.FightRate, i18n.FightBySide} {
		if !strings.Contains(body, strings.Fields(m.text(line))[0]) {
			t.Errorf("the screen does not draw %v:\n%s", line, body)
		}
	}
}

// TestTheFightWalksTheCatalogueAndTheSeedCount is the two things this screen
// lets somebody change, and that both are cached rather than re-fought on every
// redraw.
func TestTheFightWalksTheCatalogueAndTheSeedCount(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	// A second squad, so the opponent chooser has somewhere to walk to.
	second := m.squad.Editing.Clone()
	second.ID = "khac"
	if err := m.lib.SaveSquad(second); err != nil {
		t.Fatalf("save a second squad: %v", err)
	}
	m.squad = m.squad.Refresh(m.ctx())
	m = key(t, m, "esc")
	m = typeText(t, m, "f")

	was := m.fight.away
	m = key(t, m, "right")
	if m.fight.away == was {
		t.Fatalf("the opponent did not move off %d", was)
	}
	home, away, ok := m.fight.sides(m)
	if !ok || home.ID == away.ID {
		t.Errorf("the pairing is %q against %q", home.ID, away.ID)
	}

	seeds := m.fight.seeds
	m = typeText(t, m, "+")
	if m.fight.seeds != seeds*2 {
		t.Errorf("+ moved the count to %d, want %d", m.fight.seeds, seeds*2)
	}
	m = typeText(t, m, "-")
	m = typeText(t, m, "-")
	if m.fight.seeds != seeds/2 {
		t.Errorf("- moved the count to %d, want %d", m.fight.seeds, seeds/2)
	}
	// The floor and the ceiling hold, so a key held down cannot run a hundred
	// thousand battles or none at all.
	for range 10 {
		m = typeText(t, m, "-")
	}
	if m.fight.seeds != fightMinSeeds {
		t.Errorf("the count fell to %d, past the floor of %d", m.fight.seeds, fightMinSeeds)
	}
	for range 10 {
		m = typeText(t, m, "+")
	}
	if m.fight.seeds != fightMaxSeeds {
		t.Errorf("the count rose to %d, past the ceiling of %d", m.fight.seeds, fightMaxSeeds)
	}
}

// twoSquadsSaved puts two squads in the catalogue and hands back a model sitting
// on the listing with the cursor on the first of them.
//
// Two, because every claim about which side is which needs a second squad to be
// wrong about: one squad makes home and away the same row, and a screen reading
// the wrong index would look exactly right.
func twoSquadsSaved(t *testing.T, m model) model {
	t.Helper()
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	second := m.squad.Editing.Clone()
	second.ID = "khac"
	if err := m.lib.SaveSquad(second); err != nil {
		t.Fatalf("save the second squad: %v", err)
	}
	m.squad = m.squad.Refresh(m.ctx())
	// Back out to the catalogue: the builder leaves the squad open, and f in a
	// name field is a letter.
	m = key(t, m, "esc")
	if m.screen != screenSquads {
		t.Fatalf("esc left the builder for %v", m.screen)
	}
	if len(m.squad.Saved) != 2 {
		t.Fatalf("the catalogue holds %d squads, want the two just saved", len(m.squad.Saved))
	}
	return m
}

// homeAndAway reads the pairing off the drawn heading, which is the only place
// the screen says which way round it is.
//
// The render rather than the fields, because the fields are what a test would be
// asserting about itself: home is an index, and an index read into the wrong list
// draws the wrong id while comparing equal to what was set.
func homeAndAway(t *testing.T, m model) (home, away string) {
	t.Helper()
	for _, line := range strings.Split(m.screenContent(), "\n") {
		before, after, found := strings.Cut(line, m.text(i18n.FightAgainst))
		if !found || !strings.Contains(before, m.text(i18n.FightHeading)) {
			continue
		}
		home = strings.TrimSpace(
			strings.TrimPrefix(strings.TrimSpace(before), m.text(i18n.FightHeading)))
		away = strings.Trim(strings.TrimSpace(after), "<> ")
		return home, away
	}
	t.Fatalf("the fight draws no heading:\n%s", m.screenContent())
	return "", ""
}

// TestTheCatalogueStillFightsTheSquadUnderItsCursor is the compatibility claim:
// the fight carries its own home side now, and f from the catalogue has to go on
// meaning exactly what it meant when the screen read that cursor directly.
//
// The cursor is put on the second squad rather than the first, because the first
// is what an unseeded home index would give and the two would be indistinguishable.
func TestTheCatalogueStillFightsTheSquadUnderItsCursor(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = twoSquadsSaved(t, m)
	m = key(t, m, "down")
	wanted := m.squad.Saved[m.squad.Cursor].ID
	m = typeText(t, m, "f")
	if m.screen != screenFight {
		t.Fatalf("f opened %v", m.screen)
	}
	home, _ := homeAndAway(t, m)
	if home != wanted {
		t.Errorf("the fight draws %q as the home side, want %q, which was under the cursor",
			home, wanted)
	}
}

// TestTheFightWalksItsOwnHomeSide is the half of the pairing that used to belong
// to another screen: both are chosen here now, so both have to move here.
func TestTheFightWalksItsOwnHomeSide(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = twoSquadsSaved(t, m)
	m = typeText(t, m, "f")
	first, away := homeAndAway(t, m)

	m = key(t, m, "down")
	second, stillAway := homeAndAway(t, m)
	if second == first {
		t.Fatalf("down left the home side on %q", first)
	}
	if stillAway != away {
		t.Errorf("down moved the opponent from %q to %q", away, stillAway)
	}
	// And back, and by the letter the rest of this client accepts beside the
	// arrow, so the two do not disagree about which way each goes.
	if back, _ := homeAndAway(t, key(t, m, "up")); back != first {
		t.Errorf("up left the home side on %q, want %q", back, first)
	}
	if jumped, _ := homeAndAway(t, typeText(t, m, "k")); jumped != first {
		t.Errorf("k left the home side on %q, want %q", jumped, first)
	}
	if jumped, _ := homeAndAway(t, typeText(t, key(t, m, "up"), "j")); jumped != second {
		t.Errorf("j left the home side on %q, want %q", jumped, second)
	}
}

// TestTheMenuReachesTheFight is the door this whole change is about: a player who
// has built a side finds a battle on the menu rather than behind the catalogue.
//
// Driven from a fresh model through the menu, so the claim is that the row
// exists, lands on the fight, and draws a real pairing over a library holding two
// squads — a row that opened onto "nothing built yet" goes nowhere useful.
//
// ⚠️ It does NOT discriminate the catalogue refresh in enter(screenFight), which
// was expected to be what made this work: newSquadScreen already refreshes at
// construction, so a launch has the saved list before any screen is entered, and
// deleting that line passes this test and every other. The refresh is kept for a
// different reason, written where it sits.
func TestTheMenuReachesTheFight(t *testing.T) {
	m, _, dir := start(t, i18n.En)
	m = twoSquadsSaved(t, m)
	// A model built from scratch over the same directory, which is what a launch
	// after the squads were written is.
	fresh, _, _ := startIn(t, i18n.En, dir)
	fresh = menuTo(t, fresh, screenFight)
	if fresh.screen != screenFight {
		t.Fatalf("the menu opened %v", fresh.screen)
	}
	home, away := homeAndAway(t, fresh)
	saved := fresh.squad.Saved
	if len(saved) != 2 {
		t.Fatalf("the fight sees %d squads, want the two on the file", len(saved))
	}
	if home != saved[0].ID || away != saved[0].ID {
		t.Errorf("the fight opened on %q against %q, want the catalogue's first squad both ways",
			home, away)
	}
}

// TestMovingHomeDoesNotRedrawThePreviousPairing is the cache, which is keyed by
// the pair: a report is kept because a run costs real battles and an arrow key
// repeats, and a key that forgot the home side would hand back the last pair's
// numbers under the new pair's heading.
func TestMovingHomeDoesNotRedrawThePreviousPairing(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = twoSquadsSaved(t, m)
	m = typeText(t, m, "f")
	first, _, ok := m.fight.report(m)
	if !ok {
		t.Fatal("the first run did not happen")
	}
	if !first.Mirror() {
		t.Fatalf("the opening pairing is %q against %q, want the control",
			first.Home.ID, first.Away.ID)
	}
	// The control says so on screen, which is the visible half of the same claim.
	if !strings.Contains(m.screenContent(), m.text(i18n.FightControl)) {
		t.Errorf("the mirror does not draw its control line:\n%s", m.screenContent())
	}

	m = key(t, m, "down")
	home, away := homeAndAway(t, m)
	moved, _, ok := m.fight.report(m)
	if !ok {
		t.Fatal("the second run did not happen")
	}
	if moved.Home.ID != home || moved.Away.ID != away {
		t.Errorf("the report is %q against %q under a heading reading %q against %q",
			moved.Home.ID, moved.Away.ID, home, away)
	}
	if moved.Mirror() {
		t.Errorf("moving the home side left the control report in front: %q against %q",
			moved.Home.ID, moved.Away.ID)
	}
	if strings.Contains(m.screenContent(), m.text(i18n.FightControl)) {
		t.Errorf("the control line survived the pairing it was about:\n%s", m.screenContent())
	}
}

// TestTheFightWithNothingBuiltSaysSo is a state that could not be reached before:
// f on an empty catalogue did nothing, and the menu row is what opens it.
//
// Both languages, because the line is the whole screen when it is drawn.
func TestTheFightWithNothingBuiltSaysSo(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = menuTo(t, m, screenFight)
		if m.screen != screenFight {
			t.Fatalf("the menu opened %v in %s", m.screen, lang)
		}
		body := m.screenContent()
		if !strings.Contains(body, m.text(i18n.FightNoSquads)) {
			t.Errorf("the empty fight in %s does not say a side has to be built:\n%s", lang, body)
		}
		// And not the catalogue's line, which offers an n this screen has not got.
		if strings.Contains(body, m.text(i18n.SquadsEmpty)) {
			t.Errorf("the empty fight in %s borrows the catalogue's wording:\n%s", lang, body)
		}
	}
}

// TestASideMayBeFoughtAgainstItselfFromEitherChooser is the mirror, which is a
// real measurement here rather than a mistake to refuse: it is the one pairing
// whose answer is known in advance, so it is what says the swap still cancels.
//
// Reached by walking the two choosers onto the same row, which is the way a
// screen with two of them can be asked the question at all.
func TestASideMayBeFoughtAgainstItselfFromEitherChooser(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = twoSquadsSaved(t, m)
	m = typeText(t, m, "f")
	// Apart first, then together again from the other side: the away chooser
	// moves, and the home one is walked onto it.
	m = key(t, m, "right")
	if home, away := homeAndAway(t, m); home == away {
		t.Fatalf("the opponent did not move off %q", home)
	}
	m = key(t, m, "down")
	home, away := homeAndAway(t, m)
	if home != away {
		t.Fatalf("the two choosers did not meet: %q against %q", home, away)
	}
	report, failure, ok := m.fight.report(m)
	if !ok || failure != nil {
		t.Fatalf("a squad against itself was refused: %v", failure)
	}
	if !report.Mirror() || report.Rate() != 500 {
		t.Errorf("the mirror reads %d per mille (mirror %v), want 500 exactly",
			report.Rate(), report.Mirror())
	}
}
