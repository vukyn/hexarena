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
	second := m.squad.editing.Clone()
	second.ID = "khac"
	if err := m.lib.SaveSquad(second); err != nil {
		t.Fatalf("save a second squad: %v", err)
	}
	m.squad = m.squad.refresh(m.lib)
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
