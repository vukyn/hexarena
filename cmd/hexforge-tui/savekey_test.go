package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// TestTheCommandKeySavesWhereverControlSDoes is the whole reason this program is
// on bubbletea v2.
//
// A terminal cannot deliver the Command key over the classic escape sequences,
// so under v1 the keystroke did not exist as far as a program was concerned. The
// Kitty keyboard protocol carries it and v2 parses that protocol, which is what
// makes ModSuper something a screen can be handed at all.
//
// All three forms are driven, because three screens matching on their own
// spelling of one intent is three chances for one to be missed — and the one
// that gets missed is the one nobody presses in a test.
func TestTheCommandKeySavesWhereverControlSDoes(t *testing.T) {
	t.Run("the character form", func(t *testing.T) {
		m, _, dir := start(t, i18n.Vi)
		m = m.enter(screenNew)
		m = typeText(t, m, "fixture-film.commanded")
		m = key(t, m, "down")
		m = typeText(t, m, "Commanded")
		m = key(t, m, "down")
		m = chooseOrigin(t, m, "fixture-film")
		m = key(t, m, "down")
		m = chooseArchetype(t, m, "duelist")
		m = key(t, m, "down") // art
		m = key(t, m, "down") // species
		m = key(t, m, "down") // kit
		m = key(t, m, "down")
		m = typeText(t, m, "wind/ground")
		m = key(t, m, "super+s")

		if m.form.err != nil {
			t.Fatalf("the Command key refused the form: %v", m.form.err)
		}
		reloaded, err := forge.Load(dir)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if _, known := reloaded.Characters().Get("fixture-film.commanded"); !known {
			t.Error("the Command key did not write the character")
		}
	})

	t.Run("the origins form", func(t *testing.T) {
		m, _, dir := start(t, i18n.Vi)
		m = m.enter(screenOrigins)
		m = typeText(t, m, "a")
		m = typeText(t, m, "example-commanded")
		m = key(t, m, "down")
		m = typeText(t, m, "Example Commanded")
		m = key(t, m, "down")
		m = key(t, m, "down")
		m = typeText(t, m, "1999")
		m = key(t, m, "super+s")

		if m.origins.Err != nil {
			t.Fatalf("the Command key refused the work: %v", m.origins.Err)
		}
		if m.origins.Adding {
			t.Error("the add form is still open after the Command key wrote")
		}
		reloaded, err := forge.Load(dir)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if _, known := reloaded.Origins().Get("example-commanded"); !known {
			t.Error("the Command key did not write the work")
		}
	})

	t.Run("the skill form", func(t *testing.T) {
		m, lib, dir := start(t, i18n.Vi)
		before := len(lib.Skills().Skills())
		m = m.enter(screenSkills)
		m = typeText(t, m, "a")
		if !m.skills.Adding {
			t.Fatal("a did not open the skill form")
		}
		m = typeText(t, m, "commanded_strike")
		// A skill with no power and nothing else would be a wasted turn, and the
		// book refuses one — so this fills in enough to be a legal skill. What
		// is being measured is the keystroke, not the refusal.
		m = skillFormTo(t, m, draw.SkillFieldPower)
		m = typeText(t, m, "1200")
		m = skillFormTo(t, m, draw.SkillFieldAccuracy)
		m = typeText(t, m, "900")
		m = key(t, m, "super+s")

		if m.skills.Err != nil {
			t.Fatalf("the Command key refused the skill: %v", m.skills.Err)
		}
		reloaded, err := forge.Load(dir)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if _, err := reloaded.Skills().Lookup("commanded_strike"); err != nil {
			t.Errorf("the Command key did not write the skill: %v", err)
		}
		if got := len(reloaded.Skills().Skills()); got != before+1 {
			t.Errorf("the book holds %d skills, want %d", got, before+1)
		}
	})
}

// TestEverySaveFooterFitsTheSmallestWindow is the budget the save label is
// spelled against, and the reason it cannot grow to name a second key as well.
//
// The budget is minWidth-1 rather than minWidth for the reason the sub-screen
// layout tests use it: writing the last cell of a row is what makes a terminal
// wrap it, and a wrapped footer pushes itself off the bottom of a window that is
// exactly tall enough — which is the window this program promises to draw in.
func TestEverySaveFooterFitsTheSmallestWindow(t *testing.T) {
	const drawable = minWidth - 1
	footers := map[string]i18n.Key{
		"the character form": i18n.FormFooter,
		"the skill form":     i18n.SkillFormFooter,
		"the origins form":   i18n.OriginFormFooter,
	}
	for _, lang := range i18n.Langs() {
		for name, key := range footers {
			footer := lang.Say(key, saveKeyLabel())
			if width := lipgloss.Width(footer); width > drawable {
				t.Errorf("%s's %s footer draws %d cells, over the %d it has:\n%s",
					name, lang, width, drawable, footer)
			}
		}
	}
}

// TestEveryFormFooterNamesTheSaveKey is the pair to the label test: the three
// forms are the three screens that write, and a footer that stopped naming the
// key would leave an author with no way to find it.
func TestEveryFormFooterNamesTheSaveKey(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		footers := map[string]string{}

		m := base.enter(screenNew)
		_, footers["the character form"] = m.form.view(m)

		m = base.enter(screenOrigins)
		m = typeText(t, m, "a")
		_, footers["the origins form"] = m.origins.View(m.ctx())

		m = base.enter(screenSkills)
		m = typeText(t, m, "a")
		_, footers["the skill form"] = m.skills.View(m.ctx())

		for name, footer := range footers {
			if !strings.Contains(footer, saveKeyLabel()) {
				t.Errorf("%s's %s footer %q does not name %q",
					name, lang, footer, saveKeyLabel())
			}
		}
	}
}
