package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
	body, _ := m.blurb.view(m)
	if want := i18n.Vi.Describe(first, lib.Patterns()); !strings.Contains(body, firstLine(want)) {
		t.Errorf("the description screen does not carry the skill's own sentences:\n%s", body)
	}

	// Moving here moves the listing's cursor, which is the one thing borrowed.
	m = send(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	second := m.skills.skills[m.skills.cursor]
	if second.ID == first.ID {
		t.Fatal("the cursor did not move, so the next assertion proves nothing")
	}
	body, _ = m.blurb.view(m)
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
