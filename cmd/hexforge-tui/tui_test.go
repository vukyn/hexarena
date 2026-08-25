package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// A bubbletea model is testable without a terminal: Update takes a message and
// returns a model, and View returns a string. Everything below drives the real
// model with real key messages and reads the real screen, so none of it needs a
// pty, a subprocess or a timing assumption — and the one property that matters
// most, that this front-end and cmd/hexforge produce the same character from
// the same answers, is asserted directly rather than inferred.

const shippedDataDir = "../../internal/seed/data"

// scratchData copies the shipped data into a temporary directory, so a test may
// write to it without touching the repository.
func scratchData(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	copyTree(t, shippedDataDir, target)
	return target
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		t.Fatalf("read %s: %v", from, err)
	}
	for _, entry := range entries {
		source, destination := filepath.Join(from, entry.Name()), filepath.Join(to, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				t.Fatalf("create %s: %v", destination, err)
			}
			copyTree(t, source, destination)
			continue
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read %s: %v", source, err)
		}
		if err := os.WriteFile(destination, raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", destination, err)
		}
	}
}

// start builds a model over a scratch copy of the data, in the language asked
// for, sized to a terminal that is comfortably big enough.
//
// NO_COLOR is set for every test: the styles then render as plain text, which
// is what lets an assertion look for a word rather than for a word wrapped in
// escape codes. That it works at all is the point of the palette — meaning
// never lives in colour here, in either language.
//
// The language is a parameter rather than the default because most of what is
// asserted below is a state rather than a sentence, and the tests that are
// about the wording say which language they mean.
func start(t *testing.T, lang i18n.Lang) (model, *forge.Library, string) {
	t.Helper()
	return startIn(t, lang, scratchData(t))
}

// startIn is start over a data directory the test has already arranged.
//
// The art picker is what needs it: the form reads the assets folder when it is
// built, so a test about art that is or is not there has to put it there, or
// take it away, before the model exists.
func startIn(t *testing.T, lang i18n.Lang, dir string) (model, *forge.Library, string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	m := newModel(lib, lang)
	m.width, m.height = 120, 44
	return m, lib, dir
}

// send pushes one message through the model and hands back the concrete type.
func send(t *testing.T, m model, message tea.Msg) model {
	t.Helper()
	next, _ := m.Update(message)
	typed, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T, want the model", next)
	}
	return typed
}

// key sends a named key: "down", "esc", "ctrl+s" and so on.
func key(t *testing.T, m model, name string) model {
	t.Helper()
	types := map[string]tea.KeyType{
		"down": tea.KeyDown, "up": tea.KeyUp, "left": tea.KeyLeft, "right": tea.KeyRight,
		"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "ctrl+s": tea.KeyCtrlS,
	}
	kind, known := types[name]
	if !known {
		t.Fatalf("no key named %q in the test helper", name)
	}
	return send(t, m, tea.KeyMsg{Type: kind})
}

// typeText sends one rune per message, which is what a keyboard does.
//
// Sending a whole word in one message would be a lie in a way that matters: a
// multi-rune KeyMsg stringifies to that word, so "up" typed in one go would be
// routed as the up arrow rather than as two letters.
func typeText(t *testing.T, m model, text string) model {
	t.Helper()
	for _, letter := range text {
		m = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{letter}})
	}
	return m
}

// retype clears the focused text field and types something else into it.
func retype(t *testing.T, m model, text string) model {
	t.Helper()
	for range len(m.form.inputs[m.form.cursor].Value()) {
		m = send(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	return typeText(t, m, text)
}

// quits reports whether a command is tea.Quit.
//
// The command is run, which is safe for the screens this asks about: theirs is
// either nil or the quit. It is deliberately not asked of the form, where a
// text field returns a blink timer that would make the test sleep.
func quits(command tea.Cmd) bool {
	if command == nil {
		return false
	}
	_, isQuit := command().(tea.QuitMsg)
	return isQuit
}

// chooseOrigin and chooseArchetype step a chooser to a named entry. The cursor
// has to already be on that field.
func chooseOrigin(t *testing.T, m model, id string) model {
	t.Helper()
	for range len(m.form.origins) {
		if m.form.draft().Origin == id {
			return m
		}
		m = key(t, m, "right")
	}
	t.Fatalf("no origin %q to choose", id)
	return m
}

func chooseArchetype(t *testing.T, m model, id string) model {
	t.Helper()
	for range len(m.form.archetypes) {
		if m.form.draft().Archetype == id {
			return m
		}
		m = key(t, m, "right")
	}
	t.Fatalf("no archetype %q to choose", id)
	return m
}

// exampleArt is one of the two placeholder images the shipped data ships with,
// which CLAUDE.md says not to delete. The art field is a chooser over what is
// really on disk, so a test that wants a known answer has to name a file that
// is really there — and picking one of these two rather than whatever sorts
// first keeps these tests out of the way of whoever is authoring a real cast.
const exampleArt = "assets/example/adept.svg"

// chooseArt steps the art chooser to a named path. The cursor has to already be
// on that field, and the path has to be one internal/forge found on disk.
func chooseArt(t *testing.T, m model, image string) model {
	t.Helper()
	for range len(m.form.art) {
		if m.form.draft().Image == image {
			return m
		}
		m = key(t, m, "right")
	}
	t.Fatalf("no art %q to choose among %v", image, m.form.art)
	return m
}

// author drives the form from the menu to a completed draft, in the order the
// fields are walked.
func author(t *testing.T, m model, id, name, origin, archetype, element string) model {
	t.Helper()
	m = m.enter(screenNew)
	m = typeText(t, m, id)
	m = key(t, m, "down")
	m = typeText(t, m, name)
	m = key(t, m, "down")
	m = chooseOrigin(t, m, origin)
	m = key(t, m, "down")
	m = chooseArchetype(t, m, archetype)
	m = key(t, m, "down") // art: chosen from what is on disk
	m = chooseArt(t, m, exampleArt)
	m = key(t, m, "down") // kit: keep the preset's
	m = key(t, m, "down")
	m = typeText(t, m, element)
	return m
}

// TestTheFormProducesTheCharacterTheCommandLineProduces is the property that
// keeps two front-ends honest.
//
// The same answers given as flags and given as keystrokes have to arrive at the
// same cast.Character, byte for byte, because they go through the same
// forge.Draft.Resolve. If this ever fails, one of the two has started deciding
// something for itself — which is exactly the drift internal/forge exists to
// prevent.
func TestTheFormProducesTheCharacterTheCommandLineProduces(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = author(t, m, "example-film.tester", "Tester", "example-film", "duelist", "wind/ground")

	fromTheForm, err := m.form.draft().Resolve(lib)
	if err != nil {
		t.Fatalf("the form's draft does not resolve: %v", err)
	}

	// The same run on the command line: five flags, everything else taken from
	// the preset and from the id.
	fromTheFlags, err := forge.Draft{
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Element: "wind/ground",
		Image: exampleArt,
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("the flag-only draft does not resolve: %v", err)
	}
	if !reflect.DeepEqual(fromTheForm, fromTheFlags) {
		t.Errorf("the two front-ends produced different characters:\nform:  %+v\nflags: %+v",
			fromTheForm, fromTheFlags)
	}

	// And the form really did fill the fields it fills rather than resolving to
	// the same thing by leaving them empty.
	draft := m.form.draft()
	if draft.Image != exampleArt {
		t.Errorf("the art path is %q, want the one that was chosen", draft.Image)
	}
	// The chooser cannot offer art that is not there, which is the whole of why
	// it is a chooser: on the command line this field is free text and a check
	// is what finds out.
	if !lib.ImageExists(draft.Image) {
		t.Errorf("the art chooser produced %q, which is not on disk", draft.Image)
	}
	preset, _ := lib.Archetypes().Get("duelist")
	if draft.Skills != strings.Join(preset.Skills, ",") {
		t.Errorf("the kit field holds %q, want the preset's kit", draft.Skills)
	}
	if draft.Stats[progression.HP] != forge.FormatCurve(preset.Stats[progression.HP]) {
		t.Errorf("the health curve field holds %q, want the preset's",
			draft.Stats[progression.HP])
	}
}

// TestTheArtFieldPicksFromWhatIsOnDisk is the field turning from a path
// somebody types into a path somebody chooses.
//
// The list is internal/forge's, because whether a file is there is the one
// question internal/core is not allowed to ask. What is asserted here is the
// choosing: where the selection starts, that it starts on the id's suggestion
// when that art really exists, and that the value it lands on is one a write
// will accept.
func TestTheArtFieldPicksFromWhatIsOnDisk(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	onDisk, err := lib.ArtFiles()
	if err != nil {
		t.Fatalf("list the art: %v", err)
	}
	if len(onDisk) < 2 {
		t.Fatalf("the data holds %d images, and stepping through a list needs two", len(onDisk))
	}

	m = m.enter(screenNew)
	if !m.form.choiceField(fieldImage) {
		t.Fatal("the art field is not a chooser with art on disk")
	}
	// Nothing typed, so nothing suggested: the first entry is where a chooser
	// has to be, since an empty art field is a character that will not write.
	if got := m.form.draft().Image; got != onDisk[0] {
		t.Errorf("the art starts at %q, want the first entry %q", got, onDisk[0])
	}

	// An id whose art is filed where the id says it would be. The suggestion
	// then names something real, and honouring it is what makes it worth
	// keeping: forge.SuggestedImage derives assets/example/sprout.svg from
	// "example.sprout", and that is deliberately not the entry the chooser
	// started on, so following it is a move this can see.
	suggested := forge.SuggestedImage("example.sprout")
	if suggested == onDisk[0] || !lib.ImageExists(suggested) {
		t.Fatalf("the suggestion for that id is %q, so this test is not asking what it means to",
			suggested)
	}
	m = typeText(t, m, "example.sprout")
	if got := m.form.draft().Image; got != suggested {
		t.Errorf("the art is %q, want the suggestion %q, which is on disk", got, suggested)
	}

	// An id whose art is not there. The suggestion means nothing now, so the
	// selection is left where it was.
	m.form = newFormScreen(lib)
	m = m.enter(screenNew)
	m = typeText(t, m, "example-film.tester")
	if lib.ImageExists(forge.SuggestedImage("example-film.tester")) {
		t.Fatal("that id's art exists, so this test is not asking what it means to")
	}
	if got := m.form.draft().Image; got != onDisk[0] {
		t.Errorf("the art is %q, want the first entry %q when the suggestion is not there", got, onDisk[0])
	}

	// And it steps, in both directions, and every value it offers is one the
	// parser will take.
	for range fieldImage {
		m = key(t, m, "down")
	}
	seen := make(map[string]bool)
	for range len(onDisk) {
		image := m.form.draft().Image
		if err := cast.ValidateImagePath(image); err != nil {
			t.Errorf("the chooser offered %q, which a write would refuse: %v", image, err)
		}
		if !lib.ImageExists(image) {
			t.Errorf("the chooser offered %q, which is not on disk", image)
		}
		seen[image] = true
		m = key(t, m, "right")
	}
	if len(seen) != len(onDisk) {
		t.Errorf("stepping right reached %d of the %d images", len(seen), len(onDisk))
	}
	// Right all the way round is back at the start, and left from there wraps
	// to the end, so neither end of the list is a dead stop.
	if got := m.form.draft().Image; got != onDisk[0] {
		t.Errorf("stepping right through the whole list ended at %q, want %q", got, onDisk[0])
	}
	m = key(t, m, "left")
	if got, want := m.form.draft().Image, onDisk[len(onDisk)-1]; got != want {
		t.Errorf("stepping left from the first entry gives %q, want the last %q", got, want)
	}

	// A choice made by hand survives an edit to the id, on the same terms a
	// typed path always did: what somebody set themselves is not overwritten
	// under them.
	chosen := m.form.draft().Image
	for range fieldImage {
		m = key(t, m, "up")
	}
	m = typeText(t, m, "-again")
	if got := m.form.draft().Image; got != chosen {
		t.Errorf("editing the id moved the art from %q to %q", chosen, got)
	}
}

// TestAFormWithNoArtOnDiskIsStillCompletable is the fallback, and it is the
// reason the art field is not a chooser unconditionally.
//
// A folder with nothing in it must not be able to stop a character being
// authored. So with no art to offer the field is the text field it always was,
// the form says where it looked rather than showing an empty chooser, and the
// write goes through — carrying the warning that the art is not there yet,
// which is the same one cmd/hexforge prints and the one the picker made
// unreachable by construction.
func TestAFormWithNoArtOnDiskIsStillCompletable(t *testing.T) {
	dir := scratchData(t)
	if err := os.RemoveAll(filepath.Join(dir, "assets")); err != nil {
		t.Fatalf("take away the art: %v", err)
	}
	m, lib, _ := startIn(t, i18n.Vi, dir)
	m = m.enter(screenNew)
	if len(m.form.art) != 0 {
		t.Fatalf("the form found art after it was taken away: %v", m.form.art)
	}
	if m.form.choiceField(fieldImage) {
		t.Fatal("the art field is a chooser with nothing to choose")
	}
	body, _ := m.form.view(m)
	if !strings.Contains(body, lib.AssetsPath()) {
		t.Errorf("the form does not name the folder it found no art in:\n%s", body)
	}

	m = typeText(t, m, "example-film.tester")
	m = key(t, m, "down")
	m = typeText(t, m, "Tester")
	m = key(t, m, "down")
	m = chooseOrigin(t, m, "example-film")
	m = key(t, m, "down")
	m = chooseArchetype(t, m, "duelist")
	m = key(t, m, "down")
	// The suggestion still fills the field, exactly as it did before there was
	// anything to choose from.
	if got, want := m.form.draft().Image, forge.SuggestedImage("example-film.tester"); got != want {
		t.Errorf("the art field holds %q, want the suggestion %q", got, want)
	}
	typed := "assets/example-film/tester.png"
	m = retype(t, m, typed)
	m = key(t, m, "down")
	m = key(t, m, "down")
	m = typeText(t, m, "wind/ground")
	m = key(t, m, "ctrl+s")

	if m.form.err != nil {
		t.Fatalf("a form with no art on disk could not be completed: %v", m.form.err)
	}
	written, known := lib.Characters().Get("example-film.tester")
	if !known {
		t.Fatal("the character was not written")
	}
	if written.Image != typed {
		t.Errorf("the character names the art %q, want the typed %q", written.Image, typed)
	}
	// And the write says the art is not there yet, which is the whole reason a
	// field somebody can type into is still allowed to produce one.
	joined := strings.Join(m.lang.Notes(m.form.notes), "\n")
	if !strings.Contains(joined, "chưa có file") {
		t.Errorf("the confirmation does not warn that the art is missing:\n%s", joined)
	}
}

// TestTheCarryCheckFlipsWithTheElement is the check that used to arrive too
// late: a character carrying a skill of an element it does not share wrote
// cleanly and was refused by battle.New.
//
// Here the answer is on screen before anything is written, and it flips the
// moment the element changes — while naming the skill that cannot be carried,
// which is what tells the author what to do about it.
func TestTheCarryCheckFlipsWithTheElement(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	// The sentinel kit demands water, so fire cannot carry it.
	m = author(t, m, "example-film.tester", "Tester", "example-film", "sentinel", "fire")
	refused := m.form.carryLine(m, m.form.draft())
	// The whole sentence, in the language in front. It names the affinity, the
	// skill and what that skill is, because "no" on its own leaves an author
	// with nothing to change.
	if want := `KHÔNG — hệ fire không mang được chiêu "riptide" (hệ water)`; refused != want {
		t.Errorf("fire against a water kit reads\n %q\nwant\n %q", refused, want)
	}
	// The whole screen says it too, not just the helper.
	body, _ := m.form.view(m)
	if !strings.Contains(body, "riptide") {
		t.Errorf("the form does not show the carry failure:\n%s", body)
	}

	// Delete "fire" and type an affinity that carries it: the same check flips
	// without anything else changing.
	for range len("fire") {
		m = send(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m = typeText(t, m, "water/ice")
	accepted := m.form.carryLine(m, m.form.draft())
	if want := "ĐƯỢC — water/ice mang được mọi chiêu trong bộ"; accepted != want {
		t.Errorf("water/ice against a water kit reads\n %q\nwant\n %q", accepted, want)
	}

	// The two answers agree with the one the write would give, which is the
	// only reason showing them early is safe.
	if _, err := m.form.draft().Resolve(m.lib); err != nil {
		t.Errorf("the draft the check accepted does not resolve: %v", err)
	}
}

// TestTheBudgetBarShowsProgressionEffectiveHP pins the number on screen to the
// engine's own arithmetic. A budget meter drawn from a second calculation would
// be worse than no meter at all.
func TestTheBudgetBarShowsProgressionEffectiveHP(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = author(t, m, "example-film.tester", "Tester", "example-film", "duelist", "wind/ground")

	table, err := m.form.draft().Table(lib)
	if err != nil {
		t.Fatalf("the draft's table: %v", err)
	}
	values := table.At(progression.LevelCap)
	want := progression.EffectiveHP(values, lib.Rules())

	body, _ := m.form.view(m)
	if !strings.Contains(body, strconv.FormatInt(want, 10)) {
		t.Errorf("the budget line does not show %d:\n%s", want, body)
	}
	if !strings.Contains(body, strconv.FormatInt(lib.Limits().MaxEffectiveHP, 10)) {
		t.Errorf("the budget line does not show the ceiling of %d:\n%s",
			lib.Limits().MaxEffectiveHP, body)
	}

	// Health and defence multiply, so the joint bound is the one an author
	// walks into without noticing. Pushing both to their own ceilings breaks it
	// while breaking neither ceiling — which is exactly why the meter exists —
	// and it has to change the wording, not only the colour.
	m = key(t, m, "down") // bio
	m = key(t, m, "down") // health curve
	m = retype(t, m, "1440:4800")
	m = key(t, m, "down") // attack curve
	m = key(t, m, "down") // defence curve
	m = retype(t, m, "240:800")
	body, _ = m.form.view(m)
	if !strings.Contains(body, "VƯỢT HẠN MỨC") {
		t.Errorf("a stat line over the joint bound is not called out:\n%s", body)
	}
	// And the write agrees, so the warning is not a second opinion.
	if _, err := m.form.draft().Resolve(lib); err == nil {
		t.Error("a draft the screen called over budget resolved anyway")
	}
}

// TestLeavingAnEditedFormAsksFirst is the unsaved-changes guard. An Escape is
// one keystroke away from every other key on the form, so it must not be able
// to silently throw away an hour of tuning.
func TestLeavingAnEditedFormAsksFirst(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)

	// An untouched form has nothing to lose and leaves without a question.
	m = m.enter(screenNew)
	m = key(t, m, "esc")
	if m.guard != nil {
		t.Error("leaving an untouched form asked a question")
	}
	if m.screen != screenMenu {
		t.Error("leaving an untouched form did not go back")
	}

	// One keystroke later, it does ask, and the question is on screen.
	m = m.enter(screenNew)
	m = typeText(t, m, "example-film.tester")
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving an edited form did not ask")
	}
	if !strings.Contains(m.View(), "[y/N]") {
		t.Errorf("the pending question is not on screen:\n%s", m.View())
	}

	// Anything but a yes keeps the work, including the screen and the text.
	m = typeText(t, m, "n")
	if m.guard != nil {
		t.Error("the question is still pending after an answer")
	}
	if m.screen != screenNew {
		t.Error("declining the question left the form anyway")
	}
	if got := m.form.draft().ID; got != "example-film.tester" {
		t.Errorf("the id is now %q, want the typed one back", got)
	}

	// A yes leaves, and the form that comes back is empty rather than the old
	// one waiting to be saved by accident.
	m = key(t, m, "esc")
	m = typeText(t, m, "y")
	if m.screen != screenMenu {
		t.Error("confirming the question did not leave the form")
	}
	if got := m.form.draft().ID; got != "" {
		t.Errorf("the discarded form still holds %q", got)
	}
}

// TestATerminalTooSmallSaysSo covers the case where drawing anyway would be
// worse than drawing nothing: an author cannot tell a mangled form from a wrong
// one.
func TestATerminalTooSmallSaysSo(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	drawn := m.View()
	// It is drawn in the language in front, and it names both sizes: the person
	// who cannot read this screen is exactly the one who needs it.
	for _, want := range []string{"quá nhỏ", "80x24", "40x10", "hexforge"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the fallback does not mention %q:\n%s", want, drawn)
		}
	}
	if strings.Contains(drawn, m.text(i18n.FormHeading)) {
		t.Errorf("the form was drawn into a window that cannot hold it:\n%s", drawn)
	}
	// Growing the window brings the form back rather than needing a restart.
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 44})
	if !strings.Contains(m.View(), m.text(i18n.FormHeading)) {
		t.Errorf("the form did not come back after a resize:\n%s", m.View())
	}
}

// TestSavingWritesThroughTheLibrary is the round trip: the form's character
// reaches cast.json, the confirmation is the one hexforge prints, and the
// screens that list the cast see it without a restart.
func TestSavingWritesThroughTheLibrary(t *testing.T) {
	m, lib, dir := start(t, i18n.Vi)
	m = author(t, m, "example-film.tester", "Tester", "example-film", "duelist", "wind/ground")
	m = key(t, m, "ctrl+s")

	if m.form.err != nil {
		t.Fatalf("the save was refused: %v", m.form.err)
	}
	if len(m.form.notes) == 0 {
		t.Fatal("the save reported nothing")
	}
	// The notes are kept as facts and worded on the way to the screen, so this
	// asks the screen rather than the field.
	joined := strings.Join(m.lang.Notes(m.form.notes), "\n")
	if !strings.Contains(joined, "đã ghi example-film.tester") {
		t.Errorf("the confirmation does not name the write:\n%s", joined)
	}
	// The art was chosen from what is on disk, so there is nothing to warn
	// about. That warning is not gone — it is what the form says when there is
	// no art to choose from and a path was typed instead, which
	// TestAFormWithNoArtOnDiskIsStillCompletable asserts.
	if strings.Contains(joined, "chưa có file") {
		t.Errorf("the confirmation warns about art the picker took off the disk:\n%s", joined)
	}

	// A fresh form is waiting, so ctrl+s twice cannot write twice.
	if got := m.form.draft().ID; got != "" {
		t.Errorf("the form still holds %q after the save", got)
	}

	// It is on disk, and it loads back.
	reloaded, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, known := reloaded.Characters().Get("example-film.tester"); !known {
		t.Error("the character is not in the reloaded cast")
	}
	// The library the program holds knows too, so the browser and the check
	// screen see it on the next visit.
	if _, known := lib.Characters().Get("example-film.tester"); !known {
		t.Error("the open library does not hold the character it just wrote")
	}
	m = m.enter(screenBrowse)
	body, _ := m.browse.view(m)
	if !strings.Contains(body, "example-film.tester") {
		t.Errorf("the cast browser does not show the new character:\n%s", body)
	}
}

// TestEveryStateIsReadableWithoutColour is the contract the palette makes.
//
// With NO_COLOR set nothing renders an escape code, and every state that
// matters — missing art, a failed check, a refused kit — is still a word on
// screen. A tool whose only report of "this is broken" is a red cell is a tool
// that lies on a monochrome terminal and in a pasted transcript.
func TestEveryStateIsReadableWithoutColour(t *testing.T) {
	// Both languages, because a state that is a word in one and a colour in the
	// other is a screen that lies to half its readers.
	cases := []struct {
		lang                   i18n.Lang
		failed, missing, works string
	}{
		{i18n.Vi, "KHÔNG ĐẠT", "THIẾU", "ĐẠT"},
		{i18n.En, "FAILED", "MISSING", "PASSED"},
	}
	for _, test := range cases {
		t.Run(test.lang.String(), func(t *testing.T) {
			m, _, dir := start(t, test.lang)
			// Take one file away, which is the one problem only a program
			// allowed to read the filesystem can find.
			if err := os.Remove(filepath.Join(dir, "assets", "example", "sprout.svg")); err != nil {
				t.Fatalf("remove the art: %v", err)
			}
			m = m.enter(screenCheck)
			drawn := m.View()
			if strings.Contains(drawn, "\x1b[") {
				t.Errorf("NO_COLOR is set and the screen still carries escape codes:\n%q", drawn)
			}
			for _, want := range []string{test.failed, test.missing, "sprout.svg"} {
				if !strings.Contains(drawn, want) {
					t.Errorf("the check screen does not say %q:\n%s", want, drawn)
				}
			}
			// The missing-art row is the whole reason this screen exists, so it
			// is asserted as the sentence it draws rather than as a word.
			if test.lang == i18n.Vi &&
				!strings.Contains(drawn, "nhân vật example-game.sprout dùng ảnh assets/example/sprout.svg") {
				t.Errorf("the missing art is not reported in Vietnamese:\n%s", drawn)
			}

			// A passing check says so in words as well.
			fresh, _, _ := start(t, test.lang)
			fresh = fresh.enter(screenCheck)
			if !strings.Contains(fresh.View(), test.works) {
				t.Errorf("a clean check does not say it passed:\n%s", fresh.View())
			}
		})
	}
}

// TestBrowsingResolvesAtTheChosenLevel is the reason the browser is more than a
// listing: a character is a curve, and the arrow keys walk it.
func TestBrowsingResolvesAtTheChosenLevel(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenBrowse)

	body, footer := m.browse.view(m)
	if !strings.Contains(footer, "←/→") {
		t.Errorf("the footer does not offer the level keys: %q", footer)
	}
	if m.browse.level != progression.LevelCap {
		t.Errorf("the browser opens at level %d, want the cap", m.browse.level)
	}
	character := lib.Characters().All()[0]
	values, _, err := character.Resolve(progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve at the cap: %v", err)
	}
	if !strings.Contains(body, values.String()) {
		t.Errorf("the detail pane does not show the stat line at the cap:\n%s", body)
	}

	// Walking left one level shows a different, lower line, and the level never
	// runs off either end of its range.
	m = key(t, m, "left")
	if m.browse.level != progression.LevelCap-1 {
		t.Errorf("one step left went to level %d", m.browse.level)
	}
	body, _ = m.browse.view(m)
	lower, _, err := character.Resolve(progression.LevelCap - 1)
	if err != nil {
		t.Fatalf("resolve one level down: %v", err)
	}
	if !strings.Contains(body, lower.String()) {
		t.Errorf("the detail pane did not follow the level:\n%s", body)
	}
	for range progression.LevelCap + 5 {
		m = key(t, m, "left")
	}
	if m.browse.level != 1 {
		t.Errorf("walking off the bottom left the level at %d, want 1", m.browse.level)
	}

	// The origin filter narrows the list rather than hiding the fact that it
	// did: the count on screen names the filter.
	m = typeText(t, m, "f")
	body, _ = m.browse.view(m)
	if !strings.Contains(body, m.browse.filterName(m)) {
		t.Errorf("the filter in force is not named on screen:\n%s", body)
	}
	if m.browse.filterID() == "" {
		t.Fatal("pressing f did not narrow the list to a work")
	}
	for _, shown := range m.browse.rows() {
		if shown.Origin != m.browse.filterID() {
			t.Errorf("%s is shown under the %q filter", shown.ID, m.browse.filterID())
		}
	}
}

// TestQuitKeysWorkFromEveryScreen covers the promise the footers make. ctrl+c
// has to work even with a question pending, or a modal can trap somebody.
func TestQuitKeysWorkFromEveryScreen(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	for _, target := range []screen{screenMenu, screenBrowse, screenOrigins, screenCheck} {
		m := base.enter(target)
		if _, command := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); !quits(command) {
			t.Errorf("q did not quit from screen %d", target)
		}
		if _, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(command) {
			t.Errorf("ctrl+c did not quit from screen %d", target)
		}
	}

	// On the form, q is a letter: it lands in the field rather than ending the
	// session. ctrl+c is the quit there, which is what the footer says.
	m := base.enter(screenNew)
	m = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if got := m.form.draft().ID; got != "q" {
		t.Errorf("the id field holds %q, want the letter that was typed", got)
	}
	if m.screen != screenNew {
		t.Error("q left the form")
	}
	if _, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(command) {
		t.Error("ctrl+c did not quit the form")
	}
	// And with a question pending it still quits, rather than being eaten as an
	// answer: a modal nobody can escape has to be killed from another terminal.
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("the guard did not fire")
	}
	if _, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(command) {
		t.Error("ctrl+c did not quit with a question pending")
	}
}

// TestAddingAWorkWritesTheCatalog covers the origins screen, which exists so an
// author who finds the work they want missing does not have to leave.
func TestAddingAWorkWritesTheCatalog(t *testing.T) {
	m, _, dir := start(t, i18n.Vi)
	m = m.enter(screenOrigins)
	m = typeText(t, m, "a")
	if !m.origins.adding {
		t.Fatal("a did not open the add form")
	}
	m = typeText(t, m, "example-play")
	m = key(t, m, "down")
	m = typeText(t, m, "Example Play")
	m = key(t, m, "down") // medium: keep the first
	m = key(t, m, "down")
	m = typeText(t, m, "1999")
	m = key(t, m, "down")
	m = typeText(t, m, "Added by a test.")
	m = key(t, m, "ctrl+s")
	if m.origins.err != nil {
		t.Fatalf("the work was refused: %v", m.origins.err)
	}
	if m.origins.adding {
		t.Error("the add form is still open after a successful write")
	}

	reloaded, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	added, known := reloaded.Origins().Get("example-play")
	if !known {
		t.Fatal("the work is not in the reloaded catalog")
	}
	if added.Title != "Example Play" || added.Year != 1999 {
		t.Errorf("the written work is %+v", added)
	}

	// A second attempt at the same id is refused, in the words the catalog
	// itself uses.
	m = typeText(t, m, "a")
	m = typeText(t, m, "example-play")
	m = key(t, m, "down")
	m = typeText(t, m, "Duplicate")
	m = key(t, m, "ctrl+s")
	if m.origins.err == nil {
		t.Fatal("a duplicate id was accepted")
	}
	// The refusal is the catalog's own, as a value, so the screen says it in the
	// language in front while cmd/hexforge keeps its English.
	var taken *forge.OriginTakenError
	if !errors.As(m.origins.err, &taken) {
		t.Fatalf("the refusal is a %T, want a *forge.OriginTakenError", m.origins.err)
	}
	if want := `nguồn "example-play" đã có trong danh mục rồi`; m.lang.Error(taken) != want {
		t.Errorf("the refusal reads\n %q\nwant\n %q", m.lang.Error(taken), want)
	}
	if !strings.Contains(m.View(), "chưa thêm được") {
		t.Errorf("the refusal is not on screen:\n%s", m.View())
	}
}

// TestTheFooterNamesTheKeysOfTheScreenInFront is what makes the program
// navigable without a manual, and it is easy to break by adding a key and
// forgetting the footer.
//
// The keys themselves are what is asserted, in both languages: a chord is the
// same on every keyboard, so it is the part of a footer that cannot be
// translated away. The language toggle is checked on every screen for the same
// reason it works on every screen — a toggle nobody can find is a toggle
// nobody uses.
func TestTheFooterNamesTheKeysOfTheScreenInFront(t *testing.T) {
	cases := []struct {
		target screen
		wants  []string
	}{
		{screenMenu, []string{"enter", "q"}},
		{screenBrowse, []string{"↑/↓", "←/→", "f", "esc", "q"}},
		{screenNew, []string{"↑/↓", "←/→", "ctrl+s", "esc", "ctrl+c"}},
		{screenOrigins, []string{"a", "esc", "q"}},
		{screenCheck, []string{"r", "esc", "q"}},
	}
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		for _, test := range cases {
			m := base.enter(test.target)
			drawn := m.View()
			lines := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
			footer := lines[len(lines)-1]
			for _, want := range append(test.wants, "ctrl+l") {
				if !strings.Contains(footer, want) {
					t.Errorf("screen %d's %s footer %q does not offer %q",
						test.target, lang, footer, want)
				}
			}
		}
	}
}
