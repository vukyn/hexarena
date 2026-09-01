package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheBlurbScreenDescribesTheSkillUnderTheCursor is the whole contract: the
// screen keeps nothing of its own, so what it draws has to follow the listing it
// was raised from. A screen with its own cursor would drift from the one behind
// it and describe a skill the author is not looking at.
func TestTheBlurbScreenDescribesTheSkillUnderTheCursor(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	if len(m.skills.skills) < 2 {
		t.Fatalf("the fixture holds %d skills, and this needs two to move between",
			len(m.skills.skills))
	}
	m = send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.screen != screenBlurb {
		t.Fatalf("? from the listing landed on screen %d, want the description", m.screen)
	}

	first := m.skills.skills[m.skills.cursor]
	body, _ := m.blurb.View(m.ctx())
	if want := i18n.Vi.Describe(first, lib.Patterns()); !strings.Contains(body, firstLine(want)) {
		t.Errorf("the description screen does not carry the skill's own sentences:\n%s", body)
	}

	// Moving here moves the listing's cursor, which is the one thing borrowed.
	m = send(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	second := m.skills.skills[m.skills.cursor]
	if second.ID == first.ID {
		t.Fatal("the cursor did not move, so the next assertion proves nothing")
	}
	body, _ = m.blurb.View(m.ctx())
	if !strings.Contains(body, second.ID) {
		t.Errorf("after moving, the screen still does not name %q:\n%s", second.ID, body)
	}

	// Escape and ? both go back, and the listing is where they go back to.
	back := send(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if back.screen != screenSkills {
		t.Errorf("esc landed on screen %d, want the listing", back.screen)
	}
	if back.skills.cursor != m.skills.cursor {
		t.Errorf("going back moved the listing's cursor to %d, want %d",
			back.skills.cursor, m.skills.cursor)
	}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}

// TestTheBrowserDescribesTheTraitsItIsShowing is the other half of the same
// contract, and the reason the screen now branches instead of being copied.
//
// The detail pane behind it prints a trait's id and its name and stops, which
// was the whole of what this tool ever said about a trait: DescribePassive had
// one caller and it was the battle prompt. So an author moving virulence from
// 300 to 200 watched a figure change in a table and never read the line a player
// gets — the exact drift the skill half of this screen exists to prevent.
func TestTheBrowserDescribesTheTraitsItIsShowing(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenBrowse)
	// The first character carrying a trait at the cap, since a fixture may lead
	// with one that has none.
	rows := m.browse.Rows()
	found := -1
	for index, character := range rows {
		if len(lib.KitPassives(character.PassivesAt(progression.LevelCap, progression.Furthest))) > 0 {
			found = index
			break
		}
	}
	if found < 0 {
		t.Skip("no character in the fixture carries a trait, so there is nothing to describe")
	}
	m.browse.Cursor = found
	m.browse.Level = progression.LevelCap

	m = send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.screen != screenBlurb {
		t.Fatalf("? from the browser landed on screen %d, want the description", m.screen)
	}
	if m.blurb.from != screenBrowse {
		t.Fatalf("the description screen thinks it was raised from screen %d", m.blurb.from)
	}
	body, _ := m.blurb.View(m.ctx())
	held := lib.KitPassives(rows[found].PassivesAt(progression.LevelCap, progression.Furthest))
	for _, one := range held {
		if !strings.Contains(body, one.ID) {
			t.Errorf("the screen does not name the trait %q it is carrying:\n%s", one.ID, body)
		}
	}
	if !strings.Contains(body, firstLine(i18n.Vi.DescribePassive(held[0]))) {
		t.Errorf("the screen does not carry the trait's own sentences:\n%s", body)
	}

	// esc goes back to the browser rather than to the skill listing, which is
	// the whole reason `from` is kept.
	back := send(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if back.screen != screenBrowse {
		t.Errorf("esc landed on screen %d, want the browser", back.screen)
	}
}

// TestTheTraitScreenFollowsTheLevelRatherThanTheBook is what makes it a reading
// of *this unit* rather than of the catalog.
//
// A trait comes in at a level, so "what is this carrying" has no answer without
// one — and a screen that described every declared trait would be describing
// traits the character has not learned. Walking the level here walks it on the
// browser behind, the same borrow previewScreen makes.
func TestTheTraitScreenFollowsTheLevelRatherThanTheBook(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenBrowse)
	rows := m.browse.Rows()
	found := -1
	for index, character := range rows {
		early := len(lib.KitPassives(character.PassivesAt(1, progression.Furthest)))
		late := len(lib.KitPassives(character.PassivesAt(progression.LevelCap, progression.Furthest)))
		if late > early {
			found = index
			break
		}
	}
	if found < 0 {
		t.Skip("no character in the fixture learns a trait after level one")
	}
	m.browse.Cursor, m.browse.Level = found, progression.LevelCap
	m = send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	atCap, _ := m.blurb.View(m.ctx())

	// Driven with the key rather than by writing the level onto the browser,
	// because the describer is handed its subject now: home is what a reader
	// presses, and it walks the browser behind and re-pushes in one step. Writing
	// the field would leave the description reading the level it was raised at,
	// which is a fixture measuring nothing rather than a screen misbehaving.
	m = send(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	atOne, _ := m.blurb.View(m.ctx())
	if atCap == atOne {
		t.Errorf("the screen reads the same at level 1 and at the cap:\n%s", atOne)
	}
	late := lib.KitPassives(rows[found].PassivesAt(progression.LevelCap, progression.Furthest))
	early := lib.KitPassives(rows[found].PassivesAt(1, progression.Furthest))
	for _, one := range late {
		if containsPassive(early, one.ID) {
			continue
		}
		if strings.Contains(atOne, one.ID) {
			t.Errorf("%q is not learned at level 1 and the screen describes it anyway:\n%s",
				one.ID, atOne)
		}
	}
}

func containsPassive(held []passive.Passive, id string) bool {
	for _, one := range held {
		if one.ID == id {
			return true
		}
	}
	return false
}

// TestTheTraitScreenScrollsRatherThanBeingCut is why this screen keeps an
// offset at all.
//
// Five traits at the level cap wrap to more lines than a 120-by-24
// window holds, and that window is the declared floor rather than an unusual
// case. Letting the frame cut it would mean the one screen built for reading a
// trait cannot finish reading one — so the last trait has to be reachable, and
// the position line has to be there saying so.
func TestTheTraitScreenScrollsRatherThanBeingCut(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, lib, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenBrowse)
		rows := m.browse.Rows()
		found, most := -1, 0
		for index, character := range rows {
			if held := len(lib.KitPassives(
				character.PassivesAt(progression.LevelCap, progression.Furthest))); held > most {
				found, most = index, held
			}
		}
		if most < 2 {
			t.Skip("no character in the fixture carries two traits, so nothing here overflows")
		}
		m.browse.Cursor, m.browse.Level = found, progression.LevelCap
		m = send(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
		if drawn := m.screenContent(); strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: the trait screen is cut by the frame instead of scrolling:\n%s", lang, drawn)
		}
		// Far past the end, which has to clamp rather than run off it.
		for range 60 {
			m = send(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
		}
		drawn := m.screenContent()
		if strings.Contains(drawn, m.text(i18n.Truncated)) {
			t.Errorf("%s: scrolled to the end and the frame still cuts it:\n%s", lang, drawn)
		}
		held := lib.KitPassives(rows[found].PassivesAt(progression.LevelCap, progression.Furthest))
		if last := held[len(held)-1]; !strings.Contains(drawn, last.ID) {
			t.Errorf("%s: the last trait %q cannot be scrolled to:\n%s", lang, last.ID, drawn)
		}
	}
}
