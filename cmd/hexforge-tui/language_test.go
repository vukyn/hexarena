package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	return map[string]model{
		"menu":       m.enter(screenMenu),
		"browse":     m.enter(screenBrowse),
		"form":       form,
		"origins":    m.enter(screenOrigins),
		"add a work": adding,
		"check":      m.enter(screenCheck),
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
			drawn := m.View()
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
	// A kit is a list of authored skill ids, so the rows that show one — the
	// archetype chooser and the kit field — are as long as the data makes them.
	// They are clipped like a biography rather than wrapped.
	for _, preset := range lib.Archetypes().All() {
		free = append(free, strings.Join(forge.PresetFacts(preset).Skills, " "))
	}
	return free
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
	m = author(t, m, "example-film.tester", "Tester", "example-film", "sentinel", "fire")
	before := m.form.draft()
	drawnBefore := m.View()

	m = send(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
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
	drawnAfter := m.View()
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
	m = send(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	if m.lang != i18n.Vi {
		t.Errorf("the toggle does not go back, it is at %q", m.lang)
	}
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving an edited form did not ask")
	}
	m = send(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	if m.guard == nil {
		t.Fatal("the toggle answered the pending question")
	}
	if !strings.Contains(m.View(), "discard the character") {
		t.Errorf("the pending question did not follow the language:\n%s", m.View())
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
	m = author(t, m, "example-anime.adept", "Duplicate", "example-anime", "sentinel", "water/ice")
	m = key(t, m, "ctrl+s")
	if m.form.err == nil {
		t.Fatal("a character with a taken id was written")
	}
	drawn := m.View()
	if want := `chưa ghi được: nhân vật "example-anime.adept" đã có trong dàn rồi`; !strings.Contains(drawn, want) {
		t.Errorf("the refusal on screen is not %q:\n%s", want, drawn)
	}

	// A curve nobody can level into, typed into the health row: progression
	// decides it is wrong, the catalog says why in Vietnamese.
	fresh, _, _ := start(t, i18n.Vi)
	fresh = author(t, fresh, "example-film.tester", "Tester", "example-film", "duelist", "wind/ground")
	fresh = key(t, fresh, "down") // biography
	fresh = key(t, fresh, "down") // the health curve
	fresh = retype(t, fresh, "900:400")
	fresh = key(t, fresh, "ctrl+s")
	if fresh.form.err == nil {
		t.Fatal("a curve that shrinks with level was written")
	}
	want := "hp kết thúc ở 400 nhưng bắt đầu từ 900; chỉ số không tụt khi lên cấp"
	if !strings.Contains(fresh.View(), want) {
		t.Errorf("the refusal on screen is not %q:\n%s", want, fresh.View())
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
		free := freeText(lib)
		for name, m := range everyScreen(t, base) {
			m.width, m.height = 200, 60
			for _, line := range strings.Split(m.View(), "\n") {
				if carriesFreeText(line, free) {
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
		for _, line := range strings.Split(small.View(), "\n") {
			if width := lipgloss.Width(line); width > 24 {
				t.Errorf("the too-small screen in %s draws %d cells:\n%s", lang, width, line)
			}
		}
	}
}

// TestEveryLabelFitsItsColumn holds the fixed columns the detail panes line up
// on. A label a cell too long pushes a whole pane out of alignment, and only in
// one language.
func TestEveryLabelFitsItsColumn(t *testing.T) {
	labels := []i18n.Key{
		i18n.LabelFrom, i18n.LabelTunedFrom, i18n.LabelElement, i18n.LabelKit,
		i18n.LabelArt, i18n.LabelStages, i18n.LabelBiography, i18n.LabelAbsorbs,
		i18n.LabelNote, i18n.LabelBudget, i18n.LabelCarries,
	}
	for _, lang := range i18n.Langs() {
		for _, key := range labels {
			if width := lipgloss.Width(lang.Text(key)); width > detailLabelWidth {
				t.Errorf("the %s label %q is %d cells, over the %d column",
					lang, lang.Text(key), width, detailLabelWidth)
			}
		}
		// The level label takes a number, and the widest one is the cap.
		widest := lang.Say(i18n.LabelAtLevel, 999)
		if width := lipgloss.Width(widest); width > detailLabelWidth {
			t.Errorf("the %s level label %q is %d cells", lang, widest, width)
		}
		for _, key := range []i18n.Key{i18n.MenuCast, i18n.MenuNewCharacter, i18n.MenuOrigins, i18n.MenuCheck} {
			if width := lipgloss.Width(lang.Text(key)); width > menuLabelWidth {
				t.Errorf("the %s menu label %q is %d cells", lang, lang.Text(key), width)
			}
		}
		for _, key := range []i18n.Key{i18n.ArtPresent, i18n.ArtMissing} {
			if width := lipgloss.Width(lang.Text(key)); width > checkArtWidth {
				t.Errorf("the %s art column holds %q at %d cells", lang, lang.Text(key), width)
			}
		}
	}
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
