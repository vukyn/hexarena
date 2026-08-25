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
	addSkill := m.enter(screenSkills)
	addSkill.skills.adding = true
	// The two pickers, opened over the form that raises each. The kit's rows
	// carry a refusal each and an allowlist's do not, so both shapes of row are
	// measured.
	kit := form.openKit()
	allowlist := addSkill.openAllowlist(skillFieldKeptForCharacters)
	return map[string]model{
		"menu":             m.enter(screenMenu),
		"browse":           m.enter(screenBrowse),
		"form":             form,
		"origins":          m.enter(screenOrigins),
		"add a work":       adding,
		"skills":           m.enter(screenSkills),
		"add a skill":      addSkill,
		"kit picker":       kit,
		"allowlist picker": allowlist,
		"check":            m.enter(screenCheck),
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
	if want := `chưa ghi được: nhân vật "example-anime.adept" đã có trong danh sách rồi`; !strings.Contains(drawn, want) {
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

// TestALongArtPathStaysInsideItsRow is the width risk the art chooser brought
// with it, measured rather than assumed.
//
// The other two choosers show an id out of a book, which is short by
// construction: "example-anime", "bulwark". This one shows a filesystem path,
// and a path is as long as whoever filed the art made it —
// assets/example/sprout.svg is 24 cells before anybody nests one folder deeper.
// So the row shortens from the front and keeps the file name, and that is what
// is asserted here: in both languages, since the label column is measured per
// language and Vietnamese leaves three cells fewer for the value.
//
// Two shapes, because the shortening has two steps. A long folder loses the
// folder. A file name too long on its own loses its own front, so that the name
// and the extension are the part that survives.
func TestALongArtPathStaysInsideItsRow(t *testing.T) {
	const drawable = minWidth - 1
	folder := strings.Repeat("deep-folder-", 4) + "end"
	longName := strings.Repeat("very-long-name-", 4) + "end.svg"
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
		checks := []struct {
			label string
			want  string
		}{
			{m.text(i18n.LabelPlaystyle), i18n.Vi.Glossed(character.Archetype)},
			{m.text(i18n.LabelElement), i18n.Vi.GlossedAffinity(character.Element)},
		}
		for _, check := range checks {
			if check.want == "" {
				t.Errorf("%s has nothing glossed, so this proves nothing", character.ID)
				continue
			}
			row := paneRow(t, rows, check.label)
			if !strings.Contains(row, check.want) {
				t.Errorf("the %s row for %s is %q, want it to show %q",
					check.label, character.ID, row, check.want)
			}
		}
		// The kit's names are on the row under the kit's ids, in the same order.
		kit := i18n.Vi.GlossedKit(character.Skills)
		if kit == "" {
			t.Errorf("%s's kit is not glossed, so this proves nothing", character.ID)
		} else if under := rowUnder(t, rows, m.text(i18n.LabelKit)); !strings.Contains(under, kit) {
			t.Errorf("the row under %s's kit is %q, want it to show %q",
				character.ID, under, kit)
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
		"grass/electric (cỏ/điện)",
		"skirmisher (du kích)",
		"tia bắn · nanh độc · mục rữa · hồ quang",
	} {
		if !strings.Contains(sprout, want) {
			t.Errorf("the browser does not show %q:\n%s", want, sprout)
		}
	}

	// In English the same rows carry the ids alone. Every Vietnamese name of
	// every id the data holds is checked against every English screen, so a
	// gloss leaking into the wrong language is caught wherever it is drawn.
	english, _, _ := start(t, i18n.En)
	english.width, english.height = 200, 60
	var names []string
	for _, character := range lib.Characters().All() {
		names = append(names, i18n.Vi.Gloss(character.Archetype))
		for _, member := range character.Element.Elements() {
			names = append(names, i18n.Vi.Gloss(member.String()))
		}
		for _, id := range character.Skills {
			names = append(names, i18n.Vi.Gloss(id))
		}
	}
	for name, screen := range everyScreen(t, english) {
		screen.width, screen.height = 200, 60
		drawn := screen.View()
		for _, unwanted := range names {
			if unwanted != "" && strings.Contains(drawn, unwanted) {
				t.Errorf("the %s screen in English holds the gloss %q:\n%s", name, unwanted, drawn)
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
}

// paneRow is the detail row a label names, and rowUnder is the one after it.
func paneRow(t *testing.T, rows []string, label string) string {
	t.Helper()
	return rows[paneRowIndex(t, rows, label)]
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
			kits[character.ID] = character.Skills
		}
		for _, preset := range lib.Archetypes().All() {
			kits[preset.ID] = preset.Skills
		}
		for id, skills := range kits {
			glossed := lang.GlossedKit(skills)
			if glossed == "" {
				continue
			}
			if drawn := indent + lipgloss.Width(glossed); drawn > drawable {
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
			drawn := screen.View()
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
	taken := &forge.IDTakenError{ID: "example-anime.adept"}
	if got, want := i18n.Vi.Error(taken),
		`nhân vật "example-anime.adept" đã có trong danh sách rồi`; got != want {
		t.Errorf("the refusal reads %q, want %q", got, want)
	}
}
