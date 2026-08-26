package main

import (
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
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
		m = key(t, m, "down")
		m = key(t, m, "down")
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

		if m.origins.err != nil {
			t.Fatalf("the Command key refused the work: %v", m.origins.err)
		}
		if m.origins.adding {
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
		if !m.skills.adding {
			t.Fatal("a did not open the skill form")
		}
		m = typeText(t, m, "commanded_strike")
		// A skill with no power and nothing else would be a wasted turn, and the
		// book refuses one — so this fills in enough to be a legal skill. What
		// is being measured is the keystroke, not the refusal.
		m = skillFormTo(t, m, skillFieldPower)
		m = typeText(t, m, "1200")
		m = skillFormTo(t, m, skillFieldAccuracy)
		m = typeText(t, m, "900")
		m = key(t, m, "super+s")

		if m.skills.err != nil {
			t.Fatalf("the Command key refused the skill: %v", m.skills.err)
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

// TestOnlyTheTwoSaveKeysSave guards the other side of isSaveKey: a keystroke
// that merely looks like one must not write the author's work.
//
// The bare letter is the case worth naming. A save key is the letter plus a
// modifier, and bubbletea v2 leaves Text empty whenever a modifier is held — so
// "s" typed into a field carries Text and no Mod, and must reach the field
// rather than the book.
func TestOnlyTheTwoSaveKeysSave(t *testing.T) {
	saves := []tea.KeyPressMsg{
		{Code: 's', Mod: tea.ModCtrl},
		{Code: 's', Mod: tea.ModSuper},
	}
	for _, press := range saves {
		if !isSaveKey(press) {
			t.Errorf("%q does not save", press.String())
		}
	}
	others := []tea.KeyPressMsg{
		{Code: 's', Text: "s"},
		{Code: 's', Mod: tea.ModAlt},
		{Code: 'd', Mod: tea.ModCtrl},
		{Code: tea.KeyEnter},
		{Code: tea.KeyEscape},
	}
	for _, press := range others {
		if isSaveKey(press) {
			t.Errorf("%q saves, and should not", press.String())
		}
	}
}

// TestTheSaveLabelAlwaysOffersTheKeyThatAlwaysWorks is the honesty check on the
// footer.
//
// ⌘S depends on the terminal speaking the Kitty keyboard protocol and on it not
// claiming the chord for its own Save dialog first — Terminal.app does exactly
// that. So whatever platform the footer is drawn on, it has to keep naming a
// control-S, because that is the keystroke this program can actually promise.
func TestTheSaveLabelAlwaysOffersTheKeyThatAlwaysWorks(t *testing.T) {
	label := saveKeyLabel()
	if !strings.Contains(label, "^S") && !strings.Contains(label, saveKeyControl) {
		t.Errorf("the save label %q names no control-S, which is the key that always works", label)
	}
	if runtime.GOOS == "darwin" && !strings.Contains(label, "⌘") {
		t.Errorf("the save label %q offers no Command key on a Mac", label)
	}
	if runtime.GOOS != "darwin" && strings.Contains(label, "⌘") {
		t.Errorf("the save label %q offers a Command key off a Mac", label)
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
		_, footers["the origins form"] = m.origins.view(m)

		m = base.enter(screenSkills)
		m = typeText(t, m, "a")
		_, footers["the skill form"] = m.skills.view(m)

		for name, footer := range footers {
			if !strings.Contains(footer, saveKeyLabel()) {
				t.Errorf("%s's %s footer %q does not name %q",
					name, lang, footer, saveKeyLabel())
			}
		}
	}
}
