package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// atSpar is the model with the spar in front, over the character the check
// screen's cursor is on.
func atSpar(t *testing.T, m model, row int) model {
	t.Helper()
	m = m.enter(screenCheck)
	m.check.cursor = row
	m = m.enter(screenSpar)
	if m.screen != screenSpar {
		t.Fatalf("the spar did not come up: screen %d", m.screen)
	}
	return m
}

// TestTheSparFightsWhoeverTheCheckScreenHasUnderItsCursor.
//
// The screen owns no cursor over the cast, so the character it measures is
// whichever the screen behind it was on. A spar that always fought the first row
// would look right on the first row and be silently wrong everywhere else.
func TestTheSparFightsWhoeverTheCheckScreenHasUnderItsCursor(t *testing.T) {
	base, lib, _ := start(t, i18n.Vi)
	if len(lib.Characters().All()) < 2 {
		t.Fatal("one character in the book measures nothing here")
	}
	// The CHECK SCREEN's rows, not the cast — the cursor indexes those, and the
	// two stopped being the same list the day a line forked: Inspect reports one
	// row an arm, so a forking character owns two of them and walking the cast
	// put the cursor on the wrong character from the fork onwards.
	rows := base.enter(screenCheck).check.report.Rows
	if len(rows) < 2 {
		t.Fatal("one row on the check screen measures nothing here")
	}
	arms := 0
	for row, want := range rows {
		m := atSpar(t, base, row)
		m.spar.seeds = 5
		report, failure, ok := m.spar.report(m)
		if !ok || failure != nil {
			t.Fatalf("row %d (%s as %s) produced no report: %v", row, want.ID, want.Stage, failure)
		}
		if report.Challenger.ID != want.ID {
			t.Errorf("the cursor is on %s and the spar fought %s", want.ID, report.Challenger.ID)
		}
		// And the row's FORM is fought, which is the half a forking line adds:
		// two rows share an id and differ only here.
		if report.Challenger.Stage != want.Stage {
			t.Errorf("the cursor is on %s as %s and the spar fought it as %s",
				want.ID, want.Stage, report.Challenger.Stage)
		}
		if row > 0 && rows[row-1].ID == want.ID {
			arms++
		}
	}
	if arms == 0 {
		t.Log("no character in the book forks, so the form half of this is not being exercised")
	}
}

// TestTheSparDrawsTheKitItFought is the promise a win rate depends on. The
// figures are only readable beside the four skills that produced them.
func TestTheSparDrawsTheKitItFought(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		m := atSpar(t, base, 0)
		m.spar.seeds = 5
		report, _, _ := m.spar.report(m)
		body, _ := m.spar.view(m)
		for _, fielded := range report.Challenger.Skills {
			if !strings.Contains(body, fielded) {
				t.Errorf("in %s the screen shows figures without naming %q:\n%s", lang, fielded, body)
			}
		}
		if !strings.Contains(body, report.Challenger.Stage) {
			t.Errorf("in %s the screen never says which form was fielded:\n%s", lang, body)
		}
	}
}

// TestTheSparMarksTheControlRow. Every other row is an opponent and this one is
// the character against itself; a reader who took it for a matchup would read an
// even split as a fact about the cast.
func TestTheSparMarksTheControlRow(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		m := atSpar(t, base, 0)
		m.spar.seeds = 5
		report, _, _ := m.spar.report(m)
		body, _ := m.spar.view(m)
		// The parenthesised form, not the bare word: the note under the table
		// explains what the control row is, and counting that as a row would let
		// a screen that marked nothing at all pass.
		mark := "(" + m.text(i18n.SparControl) + ")"
		marked := 0
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, mark) {
				marked++
			}
		}
		if marked != 1 {
			t.Errorf("in %s exactly one row is the control and %d are marked:\n%s", lang, marked, body)
		}
		if !strings.Contains(body, report.Challenger.ID) {
			t.Errorf("in %s the control row does not name the character:\n%s", lang, body)
		}
	}
}

// TestTheSparRefightsWhenTheQuestionChanges.
//
// The level and the seed count are both cached into the key, because both change
// the answer. A cache keyed on the character alone would keep showing the first
// level's figures under a different level's heading, which is the one failure a
// screen like this must not have.
func TestTheSparRefightsWhenTheQuestionChanges(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	m := atSpar(t, base, 0)
	m.spar.seeds = 5

	atCap, _, _ := m.spar.report(m)
	if atCap.Challenger.Level != progression.LevelCap {
		t.Fatalf("the spar opened at level %d rather than the cap", atCap.Challenger.Level)
	}
	lower := m
	lower.spar.level = 1
	atOne, _, _ := lower.spar.report(lower)
	if atOne.Challenger.Level != 1 {
		t.Errorf("walking the level down still reports level %d", atOne.Challenger.Level)
	}
	if atOne.Challenger.Stats == atCap.Challenger.Stats {
		t.Error("level one and the cap field the same stat line, so the level key measures nothing")
	}

	more := m
	more.spar.seeds = 10
	if again, _, _ := more.spar.report(more); again.Seeds != 10 {
		t.Errorf("doubling the battles still reports %d seeds", again.Seeds)
	}
}

// TestEnteringTheSparForgetsWhatItFought, because the two screens that write to
// the library are one Escape away. A report is derived from the whole book, so a
// cache that survived a write would show a character fighting opponents that
// have since changed.
func TestEnteringTheSparForgetsWhatItFought(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	m := atSpar(t, base, 0)
	m.spar.seeds = 5
	if _, _, ok := m.spar.report(m); !ok {
		t.Fatal("no report to cache")
	}
	if len(m.spar.fought) == 0 {
		t.Fatal("the report was not cached, so the test below proves nothing")
	}
	if entered := m.enter(screenSpar); len(entered.spar.fought) != 0 {
		t.Errorf("entering the spar again kept %d cached report(s)", len(entered.spar.fought))
	}
}

// TestTheSparKeysGoWhereTheFooterSaysTheyDo.
func TestTheSparKeysGoWhereTheFooterSaysTheyDo(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	m := atSpar(t, base, 0)

	back := send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if back.screen != screenCheck {
		t.Errorf("escape from the spar landed on screen %d rather than the check", back.screen)
	}

	fewer := send(t, m, tea.KeyPressMsg{Code: '-', Text: "-"})
	if fewer.spar.seeds >= m.spar.seeds {
		t.Errorf("fewer battles went from %d to %d", m.spar.seeds, fewer.spar.seeds)
	}
	more := send(t, m, tea.KeyPressMsg{Code: '+', Text: "+"})
	if more.spar.seeds <= m.spar.seeds {
		t.Errorf("more battles went from %d to %d", m.spar.seeds, more.spar.seeds)
	}

	// Both ends are bounded, because a spar over no battles measures nothing and
	// one over a hundred thousand is a program that has stopped responding.
	floored := m
	for range 12 {
		floored = send(t, floored, tea.KeyPressMsg{Code: '-', Text: "-"})
	}
	if floored.spar.seeds != sparMinSeeds {
		t.Errorf("halving without end settled on %d rather than %d", floored.spar.seeds, sparMinSeeds)
	}
	ceilinged := m
	for range 12 {
		ceilinged = send(t, ceilinged, tea.KeyPressMsg{Code: '+', Text: "+"})
	}
	if ceilinged.spar.seeds != sparMaxSeeds {
		t.Errorf("doubling without end settled on %d rather than %d", ceilinged.spar.seeds, sparMaxSeeds)
	}

	lowered := send(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	if lowered.spar.level != 1 {
		t.Errorf("home settled on level %d", lowered.spar.level)
	}
	raised := send(t, lowered, tea.KeyPressMsg{Code: tea.KeyEnd})
	if raised.spar.level != progression.LevelCap {
		t.Errorf("end settled on level %d", raised.spar.level)
	}
	// The level does not run off either end of the line, because a spar at level
	// zero is not a spar at all.
	if stepped := send(t, lowered, tea.KeyPressMsg{Code: tea.KeyLeft}); stepped.spar.level != 1 {
		t.Errorf("stepping below level one reached %d", stepped.spar.level)
	}
	if stepped := send(t, raised, tea.KeyPressMsg{Code: tea.KeyRight}); stepped.spar.level != progression.LevelCap {
		t.Errorf("stepping past the cap reached %d", stepped.spar.level)
	}
}

// TestTheSparSaysNothingWhenThereIsNothingToFight, rather than drawing a table
// of no rows under a heading that promises one.
func TestTheSparSaysNothingWhenThereIsNothingToFight(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	m := atSpar(t, base, 0)
	m.check.report.Rows = nil
	body, _ := m.spar.view(m)
	if !strings.Contains(body, m.text(i18n.CheckNothingToCheck)) {
		t.Errorf("an empty book draws:\n%s", body)
	}
}
