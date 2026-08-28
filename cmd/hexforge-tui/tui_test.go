package main

import (
	"crypto/rand"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/testfixture"
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
	// The fixture is what the tests name. Before this they named the characters
	// the repository shipped, so editing the real cast broke tests that had
	// nothing to do with it.
	if err := testfixture.Inject(target, func() (testfixture.Saver, error) {
		return forge.Load(target)
	}); err != nil {
		t.Fatalf("inject the fixture: %v", err)
	}
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
//
// A key is a code and a set of modifiers rather than a type, which is what
// bubbletea v2 delivers. A modified key carries no Text, so nothing here has to
// say separately that ctrl+s is not the letter s.
func key(t *testing.T, m model, name string) model {
	t.Helper()
	presses := map[string]tea.KeyPressMsg{
		"down": {Code: tea.KeyDown}, "up": {Code: tea.KeyUp},
		"left": {Code: tea.KeyLeft}, "right": {Code: tea.KeyRight},
		"enter": {Code: tea.KeyEnter}, "esc": {Code: tea.KeyEscape},
		"tab":    {Code: tea.KeyTab},
		"ctrl+s": {Code: 's', Mod: tea.ModCtrl},
		// The Command key, which only a terminal speaking the Kitty keyboard
		// protocol ever reports. The whole point of being on v2 is that this
		// keystroke can exist at all.
		"super+s": {Code: 's', Mod: tea.ModSuper},
		// Space is a named key here rather than a rune, because that is how a
		// terminal delivers it: bubbletea turns a bare space into KeySpace,
		// whose String is " ", which is what the screens match on.
		"space": {Code: tea.KeySpace, Text: " "},
	}
	press, known := presses[name]
	if !known {
		t.Fatalf("no key named %q in the test helper", name)
	}
	return send(t, m, press)
}

// typeText sends one rune per message, which is what a keyboard does.
//
// Sending a whole word in one message would be a lie in a way that matters: a
// key carrying several characters stringifies to that word, so "up" typed in one
// go would be routed as the up arrow rather than as two letters.
//
// Code and Text both carry the letter, which is what a terminal reports for a
// printable key: Code is which key, Text is what it produced.
func typeText(t *testing.T, m model, text string) model {
	t.Helper()
	for _, letter := range text {
		m = send(t, m, tea.KeyPressMsg{Code: letter, Text: string(letter)})
	}
	return m
}

// retype clears the focused text field and types something else into it.
func retype(t *testing.T, m model, text string) model {
	t.Helper()
	for range len(m.form.inputs[m.form.cursor].Value()) {
		m = send(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
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
const exampleArt = "assets/fixture/adept.svg"

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
	m = key(t, m, "down") // species: leave it nothing in particular
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
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "duelist", "wind/ground")

	fromTheForm, err := m.form.draft().Resolve(lib)
	if err != nil {
		t.Fatalf("the form's draft does not resolve: %v", err)
	}

	// The same run on the command line: five flags, everything else taken from
	// the preset and from the id.
	fromTheFlags, err := forge.Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
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
	// keeping: forge.SuggestedImage derives assets/fixture/sprout.svg from
	// "fixture.sprout", and that is deliberately not the entry the chooser
	// started on, so following it is a move this can see.
	suggested := forge.SuggestedImage("fixture.sprout")
	if suggested == onDisk[0] || !lib.ImageExists(suggested) {
		t.Fatalf("the suggestion for that id is %q, so this test is not asking what it means to",
			suggested)
	}
	m = typeText(t, m, "fixture.sprout")
	if got := m.form.draft().Image; got != suggested {
		t.Errorf("the art is %q, want the suggestion %q, which is on disk", got, suggested)
	}

	// An id whose art is not there. The suggestion means nothing now, so the
	// selection is left where it was.
	m.form = newFormScreen(lib)
	m = m.enter(screenNew)
	m = typeText(t, m, "fixture-film.tester")
	if lib.ImageExists(forge.SuggestedImage("fixture-film.tester")) {
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

	m = typeText(t, m, "fixture-film.tester")
	m = key(t, m, "down")
	m = typeText(t, m, "Tester")
	m = key(t, m, "down")
	m = chooseOrigin(t, m, "fixture-film")
	m = key(t, m, "down")
	m = chooseArchetype(t, m, "duelist")
	m = key(t, m, "down")
	// The suggestion still fills the field, exactly as it did before there was
	// anything to choose from.
	if got, want := m.form.draft().Image, forge.SuggestedImage("fixture-film.tester"); got != want {
		t.Errorf("the art field holds %q, want the suggestion %q", got, want)
	}
	typed := "assets/fixture-film/tester.png"
	m = retype(t, m, typed)
	m = key(t, m, "down") // species
	m = key(t, m, "down") // kit
	m = key(t, m, "down")
	m = typeText(t, m, "wind/ground")
	m = key(t, m, "ctrl+s")

	if m.form.err != nil {
		t.Fatalf("a form with no art on disk could not be completed: %v", m.form.err)
	}
	written, known := lib.Characters().Get("fixture-film.tester")
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
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "sentinel", "fire")
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
		m = send(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
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
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "duelist", "wind/ground")

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

	// The fully-pierced floor is on screen too, because the bound above it
	// measures damage that does not pierce: a row showing only that figure
	// quotes the best case as though it were the only one. It is the raw health,
	// which is a different number from either of the two checked above.
	//
	// The whole worded row rather than the figure alone: the figure is the raw
	// health, which the health curve's own field is already showing a few rows
	// up, so a bare number would have passed with this row deleted. It did.
	if row := i18n.Vi.BudgetPierced(lib.Budget(values)); !strings.Contains(body, row) {
		t.Errorf("the fully-pierced floor %q is not on screen:\n%s", row, body)
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
	m = typeText(t, m, "fixture-film.tester")
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("leaving an edited form did not ask")
	}
	if !strings.Contains(m.screenContent(), "[y/N]") {
		t.Errorf("the pending question is not on screen:\n%s", m.screenContent())
	}

	// Anything but a yes keeps the work, including the screen and the text.
	m = typeText(t, m, "n")
	if m.guard != nil {
		t.Error("the question is still pending after an answer")
	}
	if m.screen != screenNew {
		t.Error("declining the question left the form anyway")
	}
	if got := m.form.draft().ID; got != "fixture-film.tester" {
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
	drawn := m.screenContent()
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
	if !strings.Contains(m.screenContent(), m.text(i18n.FormHeading)) {
		t.Errorf("the form did not come back after a resize:\n%s", m.screenContent())
	}
}

// TestSavingWritesThroughTheLibrary is the round trip: the form's character
// reaches cast.json, the confirmation is the one hexforge prints, and the
// screens that list the cast see it without a restart.
func TestSavingWritesThroughTheLibrary(t *testing.T) {
	m, lib, dir := start(t, i18n.Vi)
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "duelist", "wind/ground")
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
	if !strings.Contains(joined, "đã ghi fixture-film.tester") {
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
	if _, known := reloaded.Characters().Get("fixture-film.tester"); !known {
		t.Error("the character is not in the reloaded cast")
	}
	// The library the program holds knows too, so the browser and the check
	// screen see it on the next visit.
	if _, known := lib.Characters().Get("fixture-film.tester"); !known {
		t.Error("the open library does not hold the character it just wrote")
	}
	m = m.enter(screenBrowse)
	body, _ := m.browse.view(m)
	if !strings.Contains(body, "fixture-film.tester") {
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
			if err := os.Remove(filepath.Join(dir, "assets", "fixture", "sprout.svg")); err != nil {
				t.Fatalf("remove the art: %v", err)
			}
			m = m.enter(screenCheck)
			drawn := m.screenContent()
			if strings.Contains(drawn, "\x1b[") {
				t.Errorf("NO_COLOR is set and the screen still carries escape codes:\n%q", drawn)
			}
			// The art preview is the one screen that draws in colour on purpose,
			// so it is the one that could break this promise. Its monochrome
			// path has to be a drawing rather than a blank: a preview with the
			// colour taken out and nothing put back is an empty box.
			plain := m.enter(screenBrowse)
			plain.screen = screenPreview
			picture := plain.screenContent()
			if strings.Contains(picture, "\x1b[") {
				t.Errorf("NO_COLOR is set and the preview still carries escape codes:\n%q", picture)
			}
			if !strings.ContainsAny(picture, "=+*#%@") {
				t.Errorf("the monochrome preview drew no ink at all:\n%s", picture)
			}
			for _, want := range []string{test.failed, test.missing, "sprout.svg"} {
				if !strings.Contains(drawn, want) {
					t.Errorf("the check screen does not say %q:\n%s", want, drawn)
				}
			}
			// The missing-art row is the whole reason this screen exists, so it
			// is asserted as the sentence it draws rather than as a word.
			if test.lang == i18n.Vi &&
				!strings.Contains(drawn, "nhân vật fixture-game.sprout dùng ảnh assets/fixture/sprout.svg") {
				t.Errorf("the missing art is not reported in Vietnamese:\n%s", drawn)
			}

			// A passing check says so in words as well.
			fresh, _, _ := start(t, test.lang)
			fresh = fresh.enter(screenCheck)
			if !strings.Contains(fresh.screenContent(), test.works) {
				t.Errorf("a clean check does not say it passed:\n%s", fresh.screenContent())
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
	values, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
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
	lower, _, err := character.Resolve(progression.LevelCap-1, progression.Furthest)
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

// TestBrowsingShowsTheArtOfTheFormItResolvedTo is the per-stage art feature as a
// reader meets it: the art row is under the level and follows it, so walking the
// arrow keys is what shows which picture a form uses.
//
// The bench's grown form owns a picture of its own and its young form does not,
// which is both halves in one character: below the boundary the row shows the
// character's picture, at or above it the form's.
func TestBrowsingShowsTheArtOfTheFormItResolvedTo(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenBrowse)

	// The character with more than one picture, whichever row it is on.
	var subject cast.Character
	for _, candidate := range lib.Characters().All() {
		if len(candidate.Art()) > 1 {
			subject = candidate
			break
		}
	}
	if subject.ID == "" {
		t.Fatal("no character in the bench has art of its own per stage, so this tests nothing")
	}
	for m.browse.rows()[m.browse.cursor].ID != subject.ID {
		before := m.browse.cursor
		m = key(t, m, "down")
		if m.browse.cursor == before {
			t.Fatalf("walked to the end of the list without reaching %s", subject.ID)
		}
	}

	// The boundary the pictures change at, taken from the character rather than
	// written down here: a bench that moves its stage must not silently turn
	// this into a test of one level twice.
	grown := subject.Stages[len(subject.Stages)-1]
	if grown.Image == "" || grown.MinLevel <= 1 {
		t.Fatalf("the bench's grown form is %+v, which cannot show a change", grown)
	}
	cases := []struct {
		level int
		want  string
	}{
		{grown.MinLevel - 1, subject.Image},
		{grown.MinLevel, grown.Image},
		{progression.LevelCap, grown.Image},
	}
	for _, test := range cases {
		m.browse.level = test.level
		body := m.browse.detail(m, subject)
		if !strings.Contains(body, test.want) {
			t.Errorf("at level %d the pane does not show %s:\n%s", test.level, test.want, body)
		}
		if other := subject.Image; test.want != other && strings.Contains(body, other) {
			t.Errorf("at level %d the pane still shows %s, which belongs to another form:\n%s",
				test.level, other, body)
		}
	}
}

// TestThePreviewDrawsTheFormTheLevelResolvedTo is the art preview end to end:
// raised from the browser with p, drawing the picture of the form the level
// lands in, and walking the level walks the picture.
//
// It is drawn from the shipped data rather than the bench, because the bench's
// art is a flat placeholder and a flat picture cannot tell a working drawing
// from a broken one.
func TestThePreviewDrawsTheFormTheLevelResolvedTo(t *testing.T) {
	// The monochrome drawing, so the assertions are about characters rather
	// than about escape codes. Both paths draw the same grid.
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	m := newModel(lib, i18n.Vi)
	m.width, m.height = 92, 44
	m = m.enter(screenBrowse)

	character := m.browse.rows()[0]
	if len(character.Art()) < 2 {
		t.Skip("the shipped cast no longer has a character whose forms differ")
	}
	grown := character.Stages[len(character.Stages)-1]

	// p opens it, and the browser keeps its place: the preview has no cursor of
	// its own, so anything it lost would have to be found again on the way back.
	before := m.browse
	next, _ := m.browse.update(m, tea.KeyPressMsg{Code: 'p', Text: "p"})
	m = next.(model)
	if m.screen != screenPreview {
		t.Fatalf("p left the program on screen %d", m.screen)
	}
	if m.browse.cursor != before.cursor || m.browse.level != before.level {
		t.Error("opening the preview moved the browser underneath it")
	}

	// The young form at level one, the grown one at its own threshold, and the
	// picture on screen changes with it.
	m.browse.level = 1
	young, footer := m.preview.view(m)
	if !strings.Contains(young, character.Image) {
		t.Errorf("the preview does not name the young form's art:\n%s", young)
	}
	if !strings.Contains(footer, "←/→") {
		t.Errorf("the footer does not offer the level keys: %q", footer)
	}
	m.browse.level = grown.MinLevel
	old, _ := m.preview.view(m)
	if !strings.Contains(old, grown.Image) {
		t.Errorf("at level %d the preview does not name %s:\n%s", grown.MinLevel, grown.Image, old)
	}
	if young == old {
		t.Error("the two forms drew the same picture, so the level is not reaching the raster")
	}

	// A drawing, not a blank rectangle: the ramp has to actually be in it, and
	// the picture has to be as tall as the room it was given.
	if !strings.ContainsAny(old, "=+*#%@") {
		t.Errorf("the preview drew no ink at all:\n%s", old)
	}
	if lines := strings.Count(old, "\n"); lines < 10 {
		t.Errorf("the preview drew %d lines, want a picture rather than a caption", lines)
	}

	// esc goes back, and p from inside is the same door.
	back, _ := m.preview.update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if back.(model).screen != screenBrowse {
		t.Error("esc did not return to the browser")
	}
}

// TestThePreviewRasterisesOncePerFileAndSize is the cache and its invalidation.
//
// Two earlier versions of this proved nothing, and both are worth naming.
// Counting entries in the map fails to notice a preview with no cache at all,
// because that writes the same key every time and the map is the same size
// either way. Removing the picture then proved the cache but froze the wrong
// behaviour: a drawing that survives its file being deleted is a tool telling
// somebody the data directory still holds something it does not. So the cache is
// keyed on what the file *is* — its size and modification time — and this
// measures a hit by making the bytes unreadable without changing either.
func TestThePreviewRasterisesOncePerFileAndSize(t *testing.T) {
	m, _, dir := start(t, i18n.Vi)
	m = m.enter(screenBrowse)
	character := m.browse.rows()[clamp(m.browse.cursor, 0, len(m.browse.rows())-1)]
	art := filepath.Join(dir, character.Image)

	first, _ := m.preview.view(m)
	if !strings.Contains(first, character.Image) {
		t.Fatalf("the first look drew nothing:\n%s", first)
	}

	// Unreadable, but the same size and the same modification time, so the key
	// is unchanged and only a cache can answer.
	info, err := os.Stat(art)
	if err != nil {
		t.Fatalf("stat the art: %v", err)
	}
	if err := os.Chmod(art, 0); err != nil {
		t.Fatalf("chmod the art: %v", err)
	}
	if _, err := os.ReadFile(art); err == nil {
		t.Skip("this user can read a file with no permissions, so nothing here is measured")
	}
	again, _ := m.preview.view(m)
	if again != first {
		t.Errorf("the second look at the same file and size went back to disk:\n%s", again)
	}
	if err := os.Chmod(art, 0o644); err != nil {
		t.Fatalf("restore the art: %v", err)
	}

	// A different size is a different key, so it is drawn again — from a file
	// that is readable once more.
	m.width -= 12
	resized, _ := m.preview.view(m)
	if resized == first {
		t.Error("a resize returned the drawing made at the old size")
	}

	// And redrawing the art outside the program invalidates what was cached,
	// rather than being ignored until a restart. Written a byte shorter, so the
	// stamp differs even where a filesystem's clock is coarse.
	m.width += 12
	shorter := make([]byte, info.Size()-1)
	if _, err := rand.Read(shorter); err != nil {
		t.Fatalf("make some bytes: %v", err)
	}
	if err := os.WriteFile(art, shorter, 0o644); err != nil {
		t.Fatalf("rewrite the art: %v", err)
	}
	changed, _ := m.preview.view(m)
	if changed == first {
		t.Error("the art was rewritten and the preview kept the old drawing")
	}
}

// TestThePreviewFitsTheWindowItWasGiven is the row arithmetic, which was wrong
// when this shipped.
//
// previewChrome was guessed at five and the true cost is eight, so the picture
// came out three rows too tall at every window size and the frame replaced the
// bottom row of the drawing with its "there was more than this" notice — on a
// screen with visible space above and below the sprite. The three rows a guess
// misses are the ones nobody writes on purpose: the empty string Split leaves
// after the last newline, and the blank and the footer the frame keeps at the
// bottom.
//
// Two things are asserted, because either alone has a cheap wrong answer. The
// notice must never appear — and a picture pinned to a small constant would also
// never trigger it, so the drawing must grow with the window too.
func TestThePreviewFitsTheWindowItWasGiven(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	previous := 0
	for _, height := range []int{minHeight, 27, 30, 33, 40, 44, 60} {
		m := newModel(lib, i18n.Vi)
		m.width, m.height = 92, height
		m = m.enter(screenBrowse)
		m.screen = screenPreview

		framed := m.screenContent()
		if notice := m.text(i18n.Truncated); strings.Contains(framed, notice) {
			t.Errorf("at %d rows the preview is cut off:\n%s", height, framed)
		}
		// The footer has to be the last line, which is the reason the frame cuts
		// at all: a screen whose keys have scrolled away is one nobody can leave.
		lines := strings.Split(framed, "\n")
		if want := m.text(i18n.PreviewFooter); !strings.Contains(lines[len(lines)-1], want) {
			t.Errorf("at %d rows the last line is %q, want the footer", height, lines[len(lines)-1])
		}

		body, _ := m.preview.view(m)
		drawn := strings.Count(body, "\n")
		if drawn <= previous {
			t.Errorf("at %d rows the picture is %d lines, no more than the %d it had in a shorter window",
				height, drawn, previous)
		}
		previous = drawn
	}
}

// TestQuitKeysWorkFromEveryScreen covers the promise the footers make. ctrl+c
// has to work even with a question pending, or a modal can trap somebody.
func TestQuitKeysWorkFromEveryScreen(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	for _, target := range []screen{screenMenu, screenBrowse, screenOrigins, screenCheck, screenPreview, screenBlurb} {
		m := base.enter(target)
		if _, command := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}); !quits(command) {
			t.Errorf("q did not quit from screen %d", target)
		}
		if _, command := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); !quits(command) {
			t.Errorf("ctrl+c did not quit from screen %d", target)
		}
	}

	// On the form, q is a letter: it lands in the field rather than ending the
	// session. ctrl+c is the quit there, which is what the footer says.
	m := base.enter(screenNew)
	m = send(t, m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if got := m.form.draft().ID; got != "q" {
		t.Errorf("the id field holds %q, want the letter that was typed", got)
	}
	if m.screen != screenNew {
		t.Error("q left the form")
	}
	if _, command := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); !quits(command) {
		t.Error("ctrl+c did not quit the form")
	}
	// And with a question pending it still quits, rather than being eaten as an
	// answer: a modal nobody can escape has to be killed from another terminal.
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("the guard did not fire")
	}
	if _, command := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); !quits(command) {
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
	if !strings.Contains(m.screenContent(), "chưa thêm được") {
		t.Errorf("the refusal is not on screen:\n%s", m.screenContent())
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
		{screenNew, []string{"↑/↓", "←/→", saveKeyLabel(), "esc", "ctrl+c"}},
		{screenOrigins, []string{"a", "esc", "q"}},
		// Both of the skill list's own keys, because a key nobody is told about
		// is a key nobody presses: editing a shipped skill is the whole reason
		// this screen is more than a table.
		{screenSkills, []string{"a", "e", "esc", "q"}},
		{screenCheck, []string{"r", "esc", "q"}},
	}
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		for _, test := range cases {
			m := base.enter(test.target)
			drawn := m.screenContent()
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

// formCursorTo walks the new-character form to a field with the keys an author
// would press, rather than setting the cursor behind the screen's back.
func formCursorTo(t *testing.T, m model, field int) model {
	t.Helper()
	for m.form.cursor != field {
		m = key(t, m, "down")
	}
	return m
}

// pickTo moves an open picker's cursor onto an id.
//
// It rewinds to the top first, because the list clamps at both ends the way
// every other list in this program does: pressing down from below the target
// would never reach it.
func pickTo(t *testing.T, m model, id string) model {
	t.Helper()
	if m.picker == nil {
		t.Fatal("no picker is open")
	}
	for range len(m.picker.options) {
		m = key(t, m, "up")
	}
	// The visible rows rather than every row: a picker may be filtered, and the
	// cursor indexes what is on screen.
	for range len(m.picker.options) {
		rows := m.picker.visible()
		if len(rows) > 0 && rows[clamp(m.picker.cursor, 0, len(rows)-1)].id == id {
			return m
		}
		m = key(t, m, "down")
	}
	t.Fatalf("the picker never reached %q", id)
	return m
}

// clearKit takes every chosen skill back out, which is what a kit chosen from
// scratch starts from.
func clearKit(t *testing.T, m model) model {
	t.Helper()
	for len(m.picker.chosen) > 0 {
		m = pickTo(t, m, m.picker.chosen[0])
		m = key(t, m, "space")
	}
	return m
}

// TestTheKitIsChosenFromTheBookAndKeepsItsOrder is the field turning from a
// comma separated list somebody types into a list somebody chooses.
//
// The order is the property worth asserting: a kit is not a set, and the order
// the skills were chosen in is the order cast.json records.
func TestTheKitIsChosenFromTheBookAndKeepsItsOrder(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	m = formCursorTo(t, m, fieldKit)
	if !m.form.choiceField(fieldKit) {
		t.Fatal("the kit is still a text field")
	}
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the kit row did not open the list")
	}
	if got, want := len(m.picker.options), len(lib.Skills().Skills()); got != want {
		t.Errorf("the list offers %d skills of %d; every one has to be offered, "+
			"because a hidden skill reads as a skill that does not exist", got, want)
	}
	m = clearKit(t, m)
	// Chosen in an order that is not the book's, so that the answer cannot pass
	// by accident.
	for _, id := range []string{"bolt", "strike", "flurry"} {
		m = pickTo(t, m, id)
		m = key(t, m, "space")
	}
	m = key(t, m, "enter")
	if m.picker != nil {
		t.Fatal("enter did not close the list")
	}
	if got, want := m.form.draft().Skills, "bolt,strike,flurry"; got != want {
		t.Errorf("the kit is %q, want %q", got, want)
	}
	// Escape leaves the answer alone, which is what makes the list safe to open
	// just to read it.
	m = key(t, m, "space")
	m = clearKit(t, m)
	m = key(t, m, "esc")
	if got, want := m.form.draft().Skills, "bolt,strike,flurry"; got != want {
		t.Errorf("after escaping the list the kit is %q, want %q", got, want)
	}
}

// TestTheKitListSaysWhatThisCharacterCannotTakeAndWhy is the half of the picker
// that could have been done by hiding rows instead, and must not be.
//
// The expected sentence is built from forge.CheckSkill here rather than written
// out, because the property is that the reason on screen is the *same predicate*
// the write applies. A hand-written string would pass while the screen invented
// its own explanation.
func TestTheKitListSaysWhatThisCharacterCannotTakeAndWhy(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = author(t, m, "fixture-film.tester", "Tester", "fixture-film", "duelist", "fire")
	m = formCursorTo(t, m, fieldKit)
	m = key(t, m, "space")
	m = pickTo(t, m, "venom_fang")

	grass, err := lib.Skills().Lookup("venom_fang")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	refusal := forge.CheckSkill(m.form.draft().Carrier(), grass)
	if refusal == nil {
		t.Fatal("a fire character is allowed a grass skill")
	}
	drawn := m.screenContent()
	if !strings.Contains(drawn, i18n.Vi.Error(refusal)) {
		t.Errorf("the list does not say why the skill is unavailable:\n%s", drawn)
	}
	if !strings.Contains(drawn, "venom_fang") {
		t.Errorf("the unavailable skill is hidden rather than marked:\n%s", drawn)
	}
	// Marked, and refused: the row cannot be taken in, and nothing about the
	// answer changes when it is tried.
	before := strings.Join(m.picker.chosen, ",")
	m = key(t, m, "space")
	if got := strings.Join(m.picker.chosen, ","); got != before {
		t.Errorf("an unavailable skill was chosen: %q became %q", before, got)
	}
	// A skill the character may take is still choosable, so the mark is about
	// the skill and not about the list being read-only.
	m = pickTo(t, m, "ember_lance")
	m = key(t, m, "space")
	if !slices.Contains(m.picker.chosen, "ember_lance") {
		t.Error("a fire character could not choose a fire skill")
	}
}

// TestEitherOrderWorksBetweenTheKitAndTheElement is the conflict the two
// front-ends resolve differently, resolved here in both directions.
//
// At a prompt the kit has to come first, because an answer once given is given.
// On a form both are on screen, so whichever is filled in first constrains the
// other and the second re-checks against the first — and neither silently drops
// the other's answer.
func TestEitherOrderWorksBetweenTheKitAndTheElement(t *testing.T) {
	// Nothing answered: nothing is unavailable, because an unanswered element
	// restricts nothing. Without this the kit could only be filled in second.
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	m = formCursorTo(t, m, fieldKit)
	m = key(t, m, "space")
	for _, option := range m.picker.options {
		if option.refusal != nil {
			t.Errorf("%q is unavailable before an element was answered: %v",
				option.id, option.refusal)
		}
	}
	m = clearKit(t, m)
	// The kit first: a wind skill, chosen with no element in hand.
	m = pickTo(t, m, "gale_slash")
	m = key(t, m, "space")
	m = key(t, m, "enter")
	// Then the element, which the kit now constrains. The carry line refuses it
	// and names the skill, rather than the kit quietly losing the entry.
	m = formCursorTo(t, m, fieldElement)
	m = typeText(t, m, "fire")
	body, _ := m.form.view(m)
	if !strings.Contains(body, "gale_slash") {
		t.Errorf("the carry line does not name the skill the element cannot take:\n%s", body)
	}
	if strings.Contains(m.form.draft().Skills, "gale_slash") != true {
		t.Error("choosing an element dropped a skill from the kit behind the author's back")
	}

	// The other way round: element first, and the same skill is then marked in
	// the list by the same predicate.
	m = formCursorTo(t, m, fieldKit)
	m = key(t, m, "space")
	m = pickTo(t, m, "gale_slash")
	wind, err := lib.Skills().Lookup("gale_slash")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if forge.CheckSkill(m.form.draft().Carrier(), wind) == nil {
		t.Fatal("a fire character is allowed a wind skill")
	}
	if m.picker.options[m.picker.cursor].refusal == nil {
		t.Error("the list does not mark a skill the settled element cannot take")
	}
	// And taking it back out always works, which is how the conflict is fixed.
	m = key(t, m, "space")
	if slices.Contains(m.picker.chosen, "gale_slash") {
		t.Error("an unavailable skill could not be taken back out")
	}
}

// skillFormTo walks the new-skill form to a field with the keys an author would
// press.
func skillFormTo(t *testing.T, m model, field int) model {
	t.Helper()
	for m.skills.field != field {
		m = key(t, m, "down")
	}
	return m
}

// cycleSkillChooserTo steps a chooser until it reads the wanted value, or says
// the value was never offered.
func cycleSkillChooserTo(t *testing.T, m model, want string, read func(model) string) model {
	t.Helper()
	for range 24 {
		if read(m) == want {
			return m
		}
		m = key(t, m, "right")
	}
	t.Fatalf("the chooser never reached %q", want)
	return m
}

// TestTheSkillFormProducesTheSkillTheCommandLineProduces is the same property
// the character form has, for the book the character form draws its kit from.
//
// The same answers given as flags and as keystrokes have to arrive at the same
// skill.Skill, because both go through forge.SkillDraft.Resolve. If this fails,
// one of the two front-ends has started deciding something for itself.
func TestTheSkillFormProducesTheSkillTheCommandLineProduces(t *testing.T) {
	m, lib, dir := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	if !m.skills.adding {
		t.Fatal("a did not open the new-skill form")
	}

	m = typeText(t, m, "oath")
	m = skillFormTo(t, m, skillFieldName)
	m = typeText(t, m, "lời thề")
	m = skillFormTo(t, m, skillFieldElement)
	m = cycleSkillChooserTo(t, m, "fire", func(m model) string { return m.skills.draft(m).Element })
	m = skillFormTo(t, m, skillFieldPower)
	m = typeText(t, m, "1200")
	m = skillFormTo(t, m, skillFieldAccuracy)
	m = typeText(t, m, "900")
	m = skillFormTo(t, m, skillFieldInflicts)
	m = typeText(t, m, "burn:500")
	// The allowlist is chosen from the book rather than typed, so a name that
	// does not exist cannot be given.
	m = skillFormTo(t, m, skillFieldKeptForRoles)
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the role allowlist did not open the list")
	}
	m = pickTo(t, m, "bulwark")
	m = key(t, m, "space")
	m = key(t, m, "enter")

	fromTheForm, err := m.skills.draft(m).Resolve(lib)
	if err != nil {
		t.Fatalf("the form's draft does not resolve: %v", err)
	}
	fromTheFlags, err := forge.SkillDraft{
		ID: "oath", Name: "lời thề",
		Element: "fire", Target: "enemy", Range: "1", Pattern: "single",
		Power: "1200", Strikes: "1", Accuracy: "900", Cooldown: "0",
		Applies: "burn:500", RestrictArchetypes: "bulwark",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("the flag-only draft does not resolve: %v", err)
	}
	if !reflect.DeepEqual(fromTheForm, fromTheFlags) {
		t.Errorf("the two front-ends produced different skills:\nform:  %+v\nflags: %+v",
			fromTheForm, fromTheFlags)
	}

	// The damage is on screen before the write, and it is the engine's own
	// figure rather than a formula retyped here: 800 attack against 400
	// defence at 1200 per mille, which is the reference skills.golden's own
	// damage column is measured from.
	preview, err := m.lib.PreviewDraft(m.skills.draft(m))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.PerStrike != 411 {
		t.Errorf("a 1200 power previews as %d, want 411", preview.PerStrike)
	}
	body, _ := m.skills.view(m)
	if !strings.Contains(body, i18n.Vi.Damage(preview)) {
		t.Errorf("the form does not show the damage:\n%s", body)
	}

	m = key(t, m, "ctrl+s")
	if m.skills.err != nil {
		t.Fatalf("the write was refused: %v", m.skills.err)
	}
	if m.skills.adding {
		t.Error("the form is still open after a write")
	}
	if m.skills.added == nil || m.skills.added.ID != "oath" {
		t.Errorf("the screen does not report what it wrote: %+v", m.skills.added)
	}
	// Written through the library the screen holds, so the listing behind it
	// already has the new skill.
	if _, err := m.lib.Skills().Lookup("oath"); err != nil {
		t.Errorf("the library does not hold the written skill: %v", err)
	}
	reloaded, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	written, err := reloaded.Skills().Lookup("oath")
	if err != nil {
		t.Fatalf("the file does not hold the written skill: %v", err)
	}
	if !reflect.DeepEqual(written, fromTheForm) {
		t.Errorf("the trip through the file changed the skill:\n%+v\n%+v", written, fromTheForm)
	}
	// And the skill is now offered to a kit, which is why authoring one from
	// this program is worth having at all.
	kit := kitOptions(reloaded, forge.Carrier{})
	if !slices.ContainsFunc(kit, func(option pickOption) bool { return option.id == "oath" }) {
		t.Error("the written skill is not offered to a kit")
	}
}

// TestASavedSkillSaysTheGoldensHaveMoved is the note that makes a skill
// different from a character: a power is balance, so the measured tables the
// design was read from have changed, and reading that diff is the next step
// rather than an afterthought.
func TestASavedSkillSaysTheGoldensHaveMoved(t *testing.T) {
	_, lib, _ := start(t, i18n.Vi)
	built, err := forge.SkillDraft{
		ID: "oath", Element: "neutral", Target: "enemy", Range: "1", Pattern: "single",
		Power: "1000", Strikes: "1", Accuracy: "900", Cooldown: "0",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	kinds := make([]forge.NoteKind, 0, 3)
	for _, note := range lib.SaveSkillNoteFacts(built) {
		kinds = append(kinds, note.Kind)
	}
	want := []forge.NoteKind{forge.NoteWrote, forge.NoteGoldensMove, forge.NoteRebuild}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("a saved skill reports %v, want %v", kinds, want)
	}
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		line := lang.Note(forge.Note{Kind: forge.NoteGoldensMove})
		if !strings.Contains(line, "make golden") {
			t.Errorf("the %s note does not say how to accept the move: %q", lang, line)
		}
	}
}

// skillListTo moves the skill listing's cursor onto an id, with the keys an
// author would press.
func skillListTo(t *testing.T, m model, id string) model {
	t.Helper()
	for range len(m.skills.skills) {
		m = key(t, m, "up")
	}
	for range len(m.skills.skills) {
		if m.skills.skills[m.skills.cursor].ID == id {
			return m
		}
		m = key(t, m, "down")
	}
	t.Fatalf("the listing never reached %q", id)
	return m
}

// TestEditingASkillOpensThePrefilledFormAndReplaces is the property that makes
// one form serve both jobs.
//
// Three things are asserted and each is a way this can go wrong quietly. The
// form has to open on the skill under the cursor with its own values in it, or an
// author accepting a field as it stands changes it. The write has to replace
// rather than append, or the book grows a duplicate id and the file reorders. And
// the listing behind it has to show the new value, because a screen that reports
// a write it did not make is worse than one that reports nothing.
func TestEditingASkillOpensThePrefilledFormAndReplaces(t *testing.T) {
	m, lib, dir := start(t, i18n.Vi)
	before, err := os.ReadFile(filepath.Join(dir, "skills.json"))
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	original, err := lib.Skills().Lookup("venom_fang")
	if err != nil {
		t.Fatalf("look up venom_fang: %v", err)
	}
	count := len(lib.Skills().Skills())

	m = m.enter(screenSkills)
	m = skillListTo(t, m, "venom_fang")
	m = typeText(t, m, "e")
	if m.skills.editing != "venom_fang" {
		t.Fatalf("e opened the form on %q", m.skills.editing)
	}
	if m.skills.adding {
		t.Error("editing a skill also reports itself as adding one")
	}
	// Prefilled from the book, and prefilled exactly: the draft the form would
	// write with nothing touched is the skill as it stands.
	if drafted := m.skills.draft(m); drafted != forge.SkillAnswers(original) {
		t.Errorf("the form opened with\n%+v\nwant\n%+v", drafted, forge.SkillAnswers(original))
	}
	// The screen says which of the two jobs it is doing.
	body, _ := m.skills.view(m)
	if !strings.Contains(body, i18n.Vi.Text(i18n.SkillFormEditHeading)) {
		t.Errorf("the form does not say it is editing:\n%s", body)
	}
	if strings.Contains(body, i18n.Vi.Text(i18n.SkillFormHeading)) {
		t.Errorf("the form still says it is authoring a new skill:\n%s", body)
	}

	// Change one number and save.
	m = skillFormTo(t, m, skillFieldPower)
	for range len(m.skills.inputs[skillFieldPower].Value()) {
		m = send(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	m = typeText(t, m, "1500")
	m = key(t, m, "ctrl+s")
	if m.skills.err != nil {
		t.Fatalf("the edit was refused: %v", m.skills.err)
	}
	if m.skills.editing != "" {
		t.Error("the form is still open after a write")
	}
	if m.skills.edited == nil || m.skills.edited.After.Power != 1500 {
		t.Fatalf("the screen does not report what it changed: %+v", m.skills.edited)
	}
	if m.skills.edited.Before.Power != original.Power {
		t.Errorf("the change reports the old power as %d", m.skills.edited.Before.Power)
	}

	// A replace and not an append: the book is the same length, in the same
	// order, and the file differs in one line.
	if got := len(m.lib.Skills().Skills()); got != count {
		t.Errorf("the book holds %d skills after an edit, want %d", got, count)
	}
	after, err := os.ReadFile(filepath.Join(dir, "skills.json"))
	if err != nil {
		t.Fatalf("read the written skills: %v", err)
	}
	oldLines, newLines := strings.Split(string(before), "\n"), strings.Split(string(after), "\n")
	if len(oldLines) != len(newLines) {
		t.Fatalf("the file went from %d lines to %d", len(oldLines), len(newLines))
	}
	moved := []string(nil)
	for i := range oldLines {
		if oldLines[i] != newLines[i] {
			moved = append(moved, strings.TrimSpace(newLines[i]))
		}
	}
	if len(moved) != 1 || moved[0] != `"power": 1500,` {
		t.Errorf("the write changed %v, want one power line", moved)
	}

	// The listing shows the new value, and the two lines an edit leaves: what
	// changed, and what the damage moved from and to.
	body, _ = m.skills.view(m)
	if !strings.Contains(body, "1500x1") {
		t.Errorf("the listing does not show the new power:\n%s", body)
	}
	if want := i18n.Vi.Say(i18n.SkillEdited, "venom_fang", m.lib.SkillsPath()); !strings.Contains(body, want) {
		t.Errorf("the listing does not say %q:\n%s", want, body)
	}
	if !strings.Contains(body, i18n.Vi.DamageMoved(*m.skills.edited)) {
		t.Errorf("the listing does not show the damage before and after:\n%s", body)
	}
	// And the figures are the library's own, not the screen's.
	if m.skills.edited.BeforeDamage != lib.PreviewDamage(original) {
		t.Errorf("the before is %+v", m.skills.edited.BeforeDamage)
	}
	if m.skills.edited.AfterDamage != lib.PreviewDamage(m.skills.edited.After) {
		t.Errorf("the after is %+v", m.skills.edited.AfterDamage)
	}
}

// TestAnEditTheDataCannotSurviveIsRefusedOnScreen is the refusal the form has to
// show rather than write, in the author's own language.
//
// The wording is internal/i18n's from internal/forge's facts, which is what lets
// the screen name the character without holding a rule: the refusal it draws
// carries the carrier's id, and the screen only decides where to put it.
func TestAnEditTheDataCannotSurviveIsRefusedOnScreen(t *testing.T) {
	m, _, dir := start(t, i18n.Vi)
	before, err := os.ReadFile(filepath.Join(dir, "skills.json"))
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	m = m.enter(screenSkills)
	m = skillListTo(t, m, "riptide")
	m = typeText(t, m, "e")

	// Keep a skill a shipped character carries for an element that character does
	// not have.
	m = skillFormTo(t, m, skillFieldKeptForElements)
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the element allowlist did not open the list")
	}
	m = pickTo(t, m, "fire")
	m = key(t, m, "space")
	m = key(t, m, "enter")
	m = key(t, m, "ctrl+s")

	var broken *forge.SkillEditBreaksError
	if !errors.As(m.skills.err, &broken) {
		t.Fatalf("the screen accepted an edit that orphans a character: %v", m.skills.err)
	}
	if m.skills.editing != "riptide" {
		t.Error("a refused edit closed the form, losing what was typed")
	}
	if after, err := os.ReadFile(filepath.Join(dir, "skills.json")); err != nil {
		t.Fatalf("read the skills after the refusal: %v", err)
	} else if string(after) != string(before) {
		t.Error("a refused edit still rewrote skills.json")
	}

	body, _ := m.skills.view(m)
	// The character is named on screen, in the language in front.
	if !strings.Contains(body, broken.ID) {
		t.Errorf("the refusal does not name who would break:\n%s", body)
	}
	if !strings.Contains(body, i18n.Vi.Error(m.skills.err)) {
		t.Errorf("the refusal on screen is not the one internal/i18n words:\n%s", body)
	}
}

// TestTheSkillListingFitsTheSmallestWindowAfterAnEdit is skillsRoom's arithmetic,
// measured rather than counted by hand — because counting it by hand is how it
// went wrong before, and because an edit spends a line an addition does not.
//
// Two things make the case the busiest one this screen has, and both are
// necessary: the cursor sits on a skill with a condition, so the amplified damage
// row is drawn, and an edit has been reported, so both of the lines a write
// leaves are drawn. On any lesser state the screen has a line spare and a reserve
// that is one short still fits, which is a measurement that proves nothing.
func TestTheSkillListingFitsTheSmallestWindowAfterAnEdit(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		// The worst case, and the only one that measures anything: the cursor on a
		// skill that has a condition, so the amplified row is drawn as well. On a
		// skill without one the screen spends a line less and a reserve that is one
		// short still fits, which is a measurement that proves nothing.
		m = skillListTo(t, m, "detonate")
		if selected := m.skills.skills[m.skills.cursor]; selected.Requires == nil {
			t.Fatalf("%s no longer has a condition, so this measures the wrong case",
				selected.ID)
		}
		m.skills.edited = someSkillChange(t, m)
		drawn := m.screenContent()
		if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
			strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
			t.Errorf("the %s skill listing is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		// And the two lines an edit leaves are both really on screen, which is
		// what makes the reserve worth spending.
		for _, want := range []string{
			m.lang.Say(i18n.SkillEdited, m.skills.edited.After.ID, m.lib.SkillsPath()),
			m.lang.DamageMoved(*m.skills.edited),
		} {
			// The path is clipped to the window like any other free text, so the
			// assertion is on the front of the line rather than all of it.
			if head := want; !strings.Contains(drawn, head[:min(len(head), 20)]) {
				t.Errorf("the %s listing does not show %q:\n%s", lang, want, drawn)
			}
		}
	}
}

// TestTheShapeChooserDrawsWhatTheShapeCatches is item three of the form's
// legibility: "< pierce >" names a step chain and says nothing about which cells
// it covers, so the chooser draws them.
//
// What is asserted is the composition rather than the picture: the expectation
// here walks pattern.Targets itself, from the pattern book, and the drawn board
// has to agree cell for cell. So a diagram that marked a plausible-looking set
// of cells that a battle would not actually hit fails, which is the only failure
// worth catching — a shape chooser that lies is worse than one that says nothing.
func TestTheShapeChooserDrawsWhatTheShapeCatches(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	primary := forge.ShapeDiagramCell()
	for _, name := range lib.PatternNames() {
		shape, err := lib.Patterns().Lookup(name)
		if err != nil {
			t.Fatalf("look up %s: %v", name, err)
		}
		caught := shape.Targets(primary)
		if len(caught) == 0 || caught[0] != primary {
			t.Fatalf("%s catches %v from %v, want the primary first", name, caught, primary)
		}
		want := hex.Render(func(cell hex.Offset) string {
			for position, target := range caught {
				if target != cell {
					continue
				}
				if position == 0 {
					return shapeAimMark
				}
				return shapeSplashMark
			}
			return ""
		})

		coverage, err := lib.ShapeCoverage(name, defaultSkillTarget)
		if err != nil {
			t.Fatalf("coverage of %s: %v", name, err)
		}
		if coverage.Covered() != len(caught) {
			t.Errorf("%s reports %d cells covered against Targets' %d",
				name, coverage.Covered(), len(caught))
		}
		if got := shapeBoard(coverage); got != want {
			t.Errorf("the board drawn for %s is not the cells Targets returns:\n%s\nwant:\n%s",
				name, got, want)
		}
		// The two marks are different characters, not two colours: the tests run
		// with NO_COLOR set, so what is counted here is what a monochrome
		// terminal shows.
		board := shapeBoard(coverage)
		if got := strings.Count(board, shapeAimMark); got != 1 {
			t.Errorf("%s marks the aim %d times, want once:\n%s", name, got, board)
		}
		if got, want := strings.Count(board, shapeSplashMark), len(caught)-1; got != want {
			t.Errorf("%s marks %d splash cells over %d caught:\n%s", name, got, want, board)
		}
		if shapeAimMark == shapeSplashMark {
			t.Error("the aim and a splash cell are the same mark, so the board needs colour")
		}

		// And the drawn screen really holds that board, rather than the board
		// being a function nothing calls.
		screen := m.enter(screenSkills)
		screen.skills.adding = true
		screen.skills.field = skillFieldShape
		screen.skills.shapeIndex = indexOf(lib.PatternNames(), name)
		screen.skills.shapeDrawn = true
		body, _ := screen.skills.view(screen)
		for _, line := range strings.Split(want, "\n") {
			if !strings.Contains(body, line) {
				t.Errorf("the %s diagram does not draw %q:\n%s", name, line, body)
			}
		}
	}
}

// TestTheShapeDiagramOpensFromTheChooserAndFollowsIt is the interaction: the key
// that opens it, the keys that leave it, and that it draws the shape the field
// behind it holds rather than a copy of it.
func TestTheShapeDiagramOpensFromTheChooserAndFollowsIt(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = skillFormTo(t, m, skillFieldShape)
	// Space is free on a chooser — a chooser is stepped with the arrows — and it
	// is the same key the three allowlists open their picker with.
	m = key(t, m, "space")
	if !m.skills.shapeDrawn {
		t.Fatal("space on the shape field did not open the diagram")
	}
	opened := m.skills.draft(m).Pattern

	// The arrows on the diagram are the chooser's own, so the field behind it
	// moves with the drawing and there is nothing to apply when it closes.
	m = key(t, m, "right")
	moved := m.skills.draft(m).Pattern
	if moved == opened {
		t.Errorf("the diagram's right arrow did not change the shape, still %q", opened)
	}
	body, _ := m.skills.view(m)
	if !strings.Contains(body, moved) {
		t.Errorf("the diagram draws %q while the field holds %q:\n%s", opened, moved, body)
	}
	m = key(t, m, "left")
	if back := m.skills.draft(m).Pattern; back != opened {
		t.Errorf("stepping back landed on %q, want %q", back, opened)
	}

	// Nothing else on the form is reachable while it is over it: enter would
	// otherwise move to the next field and escape would leave the form.
	m = key(t, m, "enter")
	if m.skills.shapeDrawn {
		t.Error("enter did not close the diagram")
	}
	if m.skills.field != skillFieldShape {
		t.Errorf("closing the diagram moved the cursor to field %d", m.skills.field)
	}
	m = key(t, m, "space")
	m = key(t, m, "esc")
	if m.skills.shapeDrawn {
		t.Error("escape did not close the diagram")
	}
	if !m.skills.adding {
		t.Error("escape on the diagram left the form as well")
	}
	if m.guard != nil {
		t.Error("escape on the diagram raised the discard question")
	}

	// The other two choosers have nothing to draw, so space on them does
	// nothing rather than opening an empty board.
	for _, field := range []int{skillFieldElement, skillFieldTarget} {
		m = skillFormTo(t, m, field)
		m = key(t, m, "space")
		if m.skills.shapeDrawn {
			t.Errorf("space on field %d opened the shape diagram", field)
		}
	}
	_ = lib
}

// TestTheMenuFitsTheSmallestWindow is the screen everybody arrives on, measured
// like the sub-screens are.
//
// The note underneath the menu is the part that grows: it is the only place a
// keystroke that is not in a footer gets explained, so it collects a line every
// time something needs saying, and the menu has a whole window to lose before
// anyone notices. Both languages, because the Vietnamese note is the longer one
// and it is the default.
//
// The measurement is the note's own lines rather than the drawn screen's: the
// header carries the data directory, which is a filesystem path of any length
// and is deliberately clipped to the window like any other free text. A note
// line is written by hand and cannot be clipped without losing a word.
func TestTheMenuFitsTheSmallestWindow(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		drawn := m.screenContent()
		if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
			strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
			t.Errorf("the %s menu is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		for _, line := range strings.Split(m.text(i18n.MenuNote), "\n") {
			if width := lipgloss.Width(line); width > drawable {
				t.Errorf("a line of the %s menu note draws %d cells, over the %d it has:\n%s",
					lang, width, drawable, line)
			}
		}
	}
}

// TestTheShapeDiagramFitsTheSmallestWindow is why it is a sub-screen: the board
// alone is eight lines and the form it would have gone under spends nineteen of
// its twenty.
func TestTheShapeDiagramFitsTheSmallestWindow(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		base, lib, _ := start(t, lang)
		base.width, base.height = minWidth, minHeight
		for _, name := range lib.PatternNames() {
			m := base.enter(screenSkills)
			m.skills.adding = true
			m.skills.field = skillFieldShape
			m.skills.shapeIndex = indexOf(lib.PatternNames(), name)
			m.skills.shapeDrawn = true
			drawn := m.screenContent()
			if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
				strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
				t.Errorf("the %s diagram for %s is truncated at %dx%d:\n%s",
					lang, name, minWidth, minHeight, drawn)
			}
			body, footer := m.skills.view(m)
			for _, line := range append(strings.Split(body, "\n"), footer) {
				if width := lipgloss.Width(line); width > drawable {
					t.Errorf("the %s diagram for %s draws %d cells, over the %d it has:\n%s",
						lang, name, width, drawable, line)
				}
			}
		}
	}
}

// TestTheStatusFieldIsPickedRatherThanRemembered is item five: the field takes
// "status:chance" in parts per thousand and nothing on screen said so, so the
// statuses are chosen out of the book and the syntax is written for the author.
//
// The property that matters is not that a picker exists but that what it writes
// parses: forge.ParseApplications is the only parser and it has to accept what
// the screen produced, or the shortcut has produced a field the write refuses.
func TestTheStatusFieldIsPickedRatherThanRemembered(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = skillFormTo(t, m, skillFieldInflicts)
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the inflicts field did not open the status list")
	}
	// Every declared status is offered, in the book's own order.
	offered := make([]string, 0, len(m.picker.options))
	for _, option := range m.picker.options {
		offered = append(offered, option.id)
	}
	if !slices.Equal(offered, lib.StatusIDs()) {
		t.Errorf("the picker offers %v, want the status book's %v", offered, lib.StatusIDs())
	}

	// Pick a poison, type a chance, done.
	m = pickTo(t, m, "poison")
	m = key(t, m, "space")
	m = typeText(t, m, "300")
	m = key(t, m, "enter")
	if m.picker != nil {
		t.Fatal("enter did not close the picker")
	}
	if m.skills.err != nil {
		t.Fatalf("the picker's answer was refused: %v", m.skills.err)
	}
	written := m.skills.inputs[skillFieldInflicts].Value()
	if written != "poison:300" {
		t.Errorf("the field holds %q, want %q", written, "poison:300")
	}
	// The parser accepts it, which is the whole point.
	applications, err := lib.ParseApplications(written)
	if err != nil {
		t.Fatalf("what the picker wrote does not parse: %v", err)
	}
	if len(applications) != 1 || applications[0].Status != "poison" ||
		applications[0].Chance != 300 {
		t.Errorf("the entry parsed as %+v, want one poison at 300", applications)
	}

	// A second trip appends rather than replacing, so two statuses at two
	// chances is two trips — which is what one chance for a batch costs.
	m = key(t, m, "space")
	m = pickTo(t, m, "blind")
	m = key(t, m, "space")
	m = typeText(t, m, "500")
	m = key(t, m, "enter")
	if got, want := m.skills.inputs[skillFieldInflicts].Value(), "poison:300,blind:500"; got != want {
		t.Errorf("the second trip left the field %q, want %q", got, want)
	}
	if _, err := lib.ParseApplications(m.skills.inputs[skillFieldInflicts].Value()); err != nil {
		t.Fatalf("the appended list does not parse: %v", err)
	}

	// And the skill that comes out of the form really carries them.
	m = skillFormTo(t, m, skillFieldPower)
	m = typeText(t, m, "800")
	m = skillFormTo(t, m, skillFieldAccuracy)
	m = typeText(t, m, "900")
	m = skillFormTo(t, m, skillFieldID)
	m = typeText(t, m, "hex_bite")
	built, err := m.skills.draft(m).Resolve(m.lib)
	if err != nil {
		t.Fatalf("the draft does not resolve: %v", err)
	}
	if len(built.Applies) != 2 {
		t.Fatalf("the skill inflicts %+v, want two statuses", built.Applies)
	}
	if built.Applies[0].Status != "poison" || built.Applies[0].Chance != 300 ||
		built.Applies[1].Status != "blind" || built.Applies[1].Chance != 500 {
		t.Errorf("the skill inflicts %+v, want poison at 300 and blind at 500", built.Applies)
	}
}

// TestTheChanceFieldTakesDigitsAndDefaultsToCertain covers the two ways the
// chance is answered and the one way it cannot be.
//
// The default is a placeholder rather than a value, and that is the fix for a
// real fault: a four-digit default in a field limited to four characters refuses
// the next keystroke, so the first thing an author had to do was delete an answer
// they never gave.
func TestTheChanceFieldTakesDigitsAndDefaultsToCertain(t *testing.T) {
	m, lib, _ := start(t, i18n.En)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = skillFormTo(t, m, skillFieldInflicts)

	// Nothing typed: the default is written, and it is on screen as the
	// placeholder before enter is pressed.
	m = key(t, m, "space")
	if got := m.picker.typed.Value(); got != "" {
		t.Errorf("the chance field opens holding %q, want it empty so typing works", got)
	}
	if got := m.picker.typed.Placeholder; got != forge.DefaultApplicationChance {
		t.Errorf("the chance field shows %q as its default, want %q",
			got, forge.DefaultApplicationChance)
	}
	body, _ := m.picker.view(m)
	if !strings.Contains(body, forge.DefaultApplicationChance) {
		t.Errorf("the default chance is not on screen:\n%s", body)
	}
	m = pickTo(t, m, "block")
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if got, want := m.skills.inputs[skillFieldInflicts].Value(),
		"block:"+forge.DefaultApplicationChance; got != want {
		t.Errorf("enter with nothing typed wrote %q, want %q", got, want)
	}

	// Letters are refused by the field, and the movement keys stay movement:
	// k and j are how a picker's cursor moves, so they must not become text.
	m.skills.inputs[skillFieldInflicts].SetValue("")
	m = key(t, m, "space")
	before := m.picker.cursor
	m = typeText(t, m, "j")
	if m.picker.cursor == before {
		t.Error("j did not move the picker's cursor, so it was typed into the field")
	}
	m = typeText(t, m, "abc")
	if got := m.picker.typed.Value(); got != "" {
		t.Errorf("the chance field took %q, want digits only", got)
	}
	m = typeText(t, m, "7")
	if got := m.picker.typed.Value(); got != "7" {
		t.Errorf("the chance field holds %q after a digit, want %q", got, "7")
	}

	// Escape throws the trip away, including the chance.
	m = key(t, m, "esc")
	if m.picker != nil {
		t.Fatal("escape did not close the picker")
	}
	if got := m.skills.inputs[skillFieldInflicts].Value(); got != "" {
		t.Errorf("escape still wrote %q", got)
	}
	_ = lib
}

// TestTypingTheStatusSyntaxProducesWhatThePickerDoes is the rule the picker must
// not break: the field is the record, and a script writes the same thing by hand.
func TestTypingTheStatusSyntaxProducesWhatThePickerDoes(t *testing.T) {
	// Two models built from scratch rather than one forked in two. A model holds
	// its text fields in a slice, so two copies of one model write into the same
	// fields — which is harmless in the program, where there is one model, and a
	// silent wrong answer in a test that compares two paths.
	openForm := func() model {
		m, _, _ := start(t, i18n.Vi)
		m = m.enter(screenSkills)
		m = typeText(t, m, "a")
		m = typeText(t, m, "oath")
		m = skillFormTo(t, m, skillFieldPower)
		m = typeText(t, m, "800")
		m = skillFormTo(t, m, skillFieldAccuracy)
		m = typeText(t, m, "900")
		return skillFormTo(t, m, skillFieldInflicts)
	}
	byHand := typeText(t, openForm(), "poison:300")

	byPicker := key(t, openForm(), "space")
	byPicker = pickTo(t, byPicker, "poison")
	byPicker = key(t, byPicker, "space")
	byPicker = typeText(t, byPicker, "300")
	byPicker = key(t, byPicker, "enter")

	fromHand, err := byHand.skills.draft(byHand).Resolve(byHand.lib)
	if err != nil {
		t.Fatalf("the typed draft does not resolve: %v", err)
	}
	fromPicker, err := byPicker.skills.draft(byPicker).Resolve(byPicker.lib)
	if err != nil {
		t.Fatalf("the picked draft does not resolve: %v", err)
	}
	if !reflect.DeepEqual(fromHand, fromPicker) {
		t.Errorf("typing and picking produced different skills:\ntyped:  %+v\npicked: %+v",
			fromHand, fromPicker)
	}
}

// TestTheStatusPickerFitsTheSmallestWindow is the layout, measured the way the
// other sub-screens are: the field it carries costs two lines the other four
// pickers do not spend, and the status book is twelve rows.
func TestTheStatusPickerFitsTheSmallestWindow(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		m.skills.adding = true
		m.skills.field = skillFieldInflicts
		m = m.openStatuses()
		// Something chosen and something typed, which is the busiest it gets.
		m = pickTo(t, m, "poison")
		m = key(t, m, "space")
		m = typeText(t, m, "300")
		drawn := m.screenContent()
		if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
			strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
			t.Errorf("the %s status picker is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, drawn)
		}
		body, footer := m.picker.view(m)
		for _, line := range append(strings.Split(body, "\n"), footer) {
			if width := lipgloss.Width(line); width > drawable {
				t.Errorf("the %s status picker draws %d cells, over the %d it has:\n%s",
					lang, width, drawable, line)
			}
		}
		// The chance row and its reading are both really on screen.
		if !strings.Contains(body, lang.Text(i18n.PickerChance)) {
			t.Errorf("the %s picker does not name the chance field:\n%s", lang, body)
		}
		if !strings.Contains(body, forge.Percent(300)) {
			t.Errorf("the %s picker does not read the chance back as a percentage:\n%s",
				lang, body)
		}
	}
}

// TestTheCharacterAllowlistNarrowsByOrigin is item six: the list of characters
// grows with the cast, so it gets a filter, and the filter is the cast browser's
// — same key, same cycle, same axis — rather than a second interaction for the
// same job.
func TestTheCharacterAllowlistNarrowsByOrigin(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = skillFormTo(t, m, skillFieldKeptForCharacters)
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the character allowlist did not open the list")
	}
	if got, want := len(m.picker.visible()), len(lib.CharacterIDs()); got != want {
		t.Fatalf("the picker opens showing %d of %d characters, want all of them", got, want)
	}
	if !slices.Equal(m.picker.groups, lib.OriginIDs()) {
		t.Errorf("the filter cycles %v, want the catalogued works %v",
			m.picker.groups, lib.OriginIDs())
	}

	// Choose somebody before filtering, so the filter can be shown not to lose
	// an answer.
	first := m.picker.visible()[0].id
	m = key(t, m, "space")

	// f narrows to one work, and the rows really are that work's.
	seen := make(map[string]int, len(m.picker.groups))
	for range len(m.picker.groups) {
		m = typeText(t, m, "f")
		group := m.picker.group()
		if group == "" {
			t.Fatalf("stepping the filter landed back on everything too early")
		}
		for _, row := range m.picker.visible() {
			character, known := lib.Characters().Get(row.id)
			if !known {
				t.Fatalf("the picker offers %q, which the cast book does not hold", row.id)
			}
			if character.Origin != group {
				t.Errorf("filtering by %q left %q, which is from %q",
					group, row.id, character.Origin)
			}
		}
		seen[group] = len(m.picker.visible())
		// Whatever was chosen stays chosen, whether the filter shows it or not:
		// narrowing is a way to find a row, never a way to lose one.
		if !slices.Contains(m.picker.chosen, first) {
			t.Errorf("filtering by %q dropped %q from the answer: %v",
				group, first, m.picker.chosen)
		}
	}
	// One more step wraps back to everything, exactly as the browser's does.
	m = typeText(t, m, "f")
	if got := m.picker.group(); got != "" {
		t.Errorf("the filter wrapped to %q, want everything", got)
	}
	if got, want := len(m.picker.visible()), len(lib.CharacterIDs()); got != want {
		t.Errorf("wrapping around shows %d of %d characters", got, want)
	}
	// The filtered counts add up to the whole list, so nothing was hidden from
	// every group at once.
	total := 0
	for _, count := range seen {
		total += count
	}
	if total != len(lib.CharacterIDs()) {
		t.Errorf("the works account for %d characters of %d", total, len(lib.CharacterIDs()))
	}

	// And the answer survives the trip.
	m = key(t, m, "enter")
	if !slices.Contains(m.skills.keptWho, first) {
		t.Errorf("the allowlist came back as %v, want it to hold %q", m.skills.keptWho, first)
	}
}

// TestOnlyTheCharacterListHasAFilter keeps f a letter nothing listens for on the
// pickers with nothing to narrow — eleven elements and five role presets are
// fixed lists, and a key that sometimes does nothing is worse than one that never
// appears.
func TestOnlyTheCharacterListHasAFilter(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	for _, field := range []int{skillFieldKeptForElements, skillFieldKeptForRoles} {
		m = skillFormTo(t, m, field)
		m = key(t, m, "space")
		if m.picker == nil {
			t.Fatalf("space on field %d did not open a list", field)
		}
		if len(m.picker.groups) != 0 {
			t.Errorf("field %d's picker carries a filter: %v", field, m.picker.groups)
		}
		before := len(m.picker.visible())
		m = typeText(t, m, "f")
		if got := len(m.picker.visible()); got != before {
			t.Errorf("f narrowed field %d's list from %d to %d", field, before, got)
		}
		if got := m.picker.footer; got != i18n.PickerFooter {
			t.Errorf("field %d's picker offers a filter key in its footer", field)
		}
		m = key(t, m, "esc")
	}
	// The kit picker on the other form is the fourth without one, and the status
	// picker the fifth.
	kit := m.enter(screenNew).openKit()
	if len(kit.picker.groups) != 0 {
		t.Errorf("the kit picker carries a filter: %v", kit.picker.groups)
	}
}

// TestAWorkWithNoCastSaysSo covers the state the filter can reach that the list
// cannot: every work is catalogued whether or not anybody has been borrowed from
// it, so a filter can land on an empty one.
func TestAWorkWithNoCastSaysSo(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, lib, _ := start(t, lang)
		m.width, m.height = minWidth, minHeight
		m = m.enter(screenSkills)
		m.skills.adding = true
		m.skills.field = skillFieldKeptForCharacters
		m = m.openAllowlist(skillFieldKeptForCharacters)

		empty := ""
		for _, origin := range lib.OriginIDs() {
			used := false
			for _, character := range lib.Characters().All() {
				if character.Origin == origin {
					used = true
					break
				}
			}
			if !used {
				empty = origin
				break
			}
		}
		if empty == "" {
			t.Skip("every catalogued work has a character, so this state is unreachable")
		}
		for m.picker.group() != empty {
			m = typeText(t, m, "f")
		}
		body, _ := m.picker.view(m)
		if !strings.Contains(body, lang.Text(i18n.PickerNothingInGroup)) {
			t.Errorf("the %s picker says nothing about an empty work:\n%s", lang, body)
		}
		// Space on nothing does nothing rather than reaching past the end of the
		// list.
		before := append([]string(nil), m.picker.chosen...)
		m = key(t, m, "space")
		if !slices.Equal(m.picker.chosen, before) {
			t.Errorf("space on an empty list chose %v", m.picker.chosen)
		}
		if strings.Contains(m.screenContent(), lang.Text(i18n.Truncated)) {
			t.Errorf("the %s filtered picker is truncated at %dx%d:\n%s",
				lang, minWidth, minHeight, m.screenContent())
		}
	}
}

// TestTheFormAuthorsASkillsVietnameseName is item seven from the front: the name
// used to be a compiled table, so it could not be authored by the tool that
// authors the skill, and now it is a field on the declaration.
//
// The end of the chain is what is asserted: a name typed on the form reaches the
// file, comes back out of it, and is what the listing draws in the column that
// used to be fed by the table.
func TestTheFormAuthorsASkillsVietnameseName(t *testing.T) {
	m, lib, dir := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = typeText(t, m, "tidal_hymn")
	m = skillFormTo(t, m, skillFieldName)
	m = typeText(t, m, "khúc thủy triều")
	m = skillFormTo(t, m, skillFieldPower)
	m = typeText(t, m, "800")
	m = skillFormTo(t, m, skillFieldAccuracy)
	m = typeText(t, m, "900")

	// The name is the second field, where it is authored, and the row is called
	// what the listing's own column is called.
	if got, want := skillFieldName, skillFieldID+1; got != want {
		t.Errorf("the name is field %d, want it straight after the id at %d", got, want)
	}
	if got, want := skillFieldLabel(m, skillFieldName), m.text(i18n.ColumnGloss); got != want {
		t.Errorf("the name row is called %q and the listing's column %q", got, want)
	}

	m = key(t, m, "ctrl+s")
	if m.skills.err != nil {
		t.Fatalf("the write was refused: %v", m.skills.err)
	}
	written, err := m.lib.Skills().Lookup("tidal_hymn")
	if err != nil {
		t.Fatalf("the library does not hold the written skill: %v", err)
	}
	if got, want := written.Name, "khúc thủy triều"; got != want {
		t.Errorf("the written skill is named %q, want %q", got, want)
	}
	// Off the disk, because the file is where a name now lives.
	reloaded, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	fromFile, err := reloaded.Skills().Lookup("tidal_hymn")
	if err != nil {
		t.Fatalf("the file does not hold the written skill: %v", err)
	}
	if fromFile.Name != written.Name {
		t.Errorf("the trip through the file left the name %q, want %q",
			fromFile.Name, written.Name)
	}
	// And it is what the listing draws, in the column the compiled table used to
	// be the only source for.
	listing := m.enter(screenSkills)
	listing.skills = listing.skills.refresh(m.lib)
	// Scrolled to the skill rather than reading the first screenful: the list
	// scrolls, so which rows are drawn depends on how many skills the book
	// holds, and asserting on the top of it made authoring a skill anywhere
	// break this.
	listing = skillListTo(t, listing, "tidal_hymn")
	body, _ := listing.skills.view(listing)
	if !strings.Contains(body, "khúc thủy triều") {
		t.Errorf("the listing does not show the authored name:\n%s", body)
	}
	_ = lib
}

// TestAnAuthoredNameOverridesTheCompiledOneOnScreen is the precedence rule where
// an author sees it: editing a shipped skill's name replaces what the compiled
// table says for it, rather than fighting it.
func TestAnAuthoredNameOverridesTheCompiledOneOnScreen(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = skillListTo(t, m, "strike")
	compiled := i18n.Vi.SkillName(m.skills.skills[m.skills.cursor])
	if compiled == "" {
		t.Fatal("strike has no compiled name, so this measures nothing")
	}

	m = typeText(t, m, "e")
	if m.skills.editing != "strike" {
		t.Fatalf("e opened the form on %q, want strike", m.skills.editing)
	}
	// The form opens holding the skill's *authored* name, which is empty for a
	// shipped skill: prefilling the compiled one would turn opening the form
	// into a write that moved a name out of Go and into the data file.
	if got := m.skills.inputs[skillFieldName].Value(); got != "" {
		t.Errorf("the name field opened holding %q, want it empty for a shipped skill", got)
	}
	m = skillFormTo(t, m, skillFieldName)
	m = typeText(t, m, "cú đánh")
	m = key(t, m, "ctrl+s")
	if m.skills.err != nil {
		t.Fatalf("the edit was refused: %v", m.skills.err)
	}

	edited, err := m.lib.Skills().Lookup("strike")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got, want := i18n.Vi.SkillName(edited), "cú đánh"; got != want {
		t.Errorf("the skill now reads %q, want the authored %q", got, want)
	}
	if i18n.Vi.SkillName(edited) == compiled {
		t.Error("the compiled name won over the authored one")
	}
	body, _ := m.skills.view(m)
	if !strings.Contains(body, "cú đánh") {
		t.Errorf("the listing still shows the compiled name:\n%s", body)
	}
	if strings.Contains(body, compiled) {
		t.Errorf("the listing shows both names at once:\n%s", body)
	}
}

// TestATransparentCellIsLeftAlone is the one property of the drawing that is not
// about how it looks: a cell with nothing in either half must be a plain space
// with no styling at all.
//
// Anything else paints the terminal's own background over with whatever colour
// this program guessed it to be, which turns a transparent margin into a
// rectangle — and the two new pictures in the shipped cast have transparent
// margins, so the wrong answer here would be visible on the first look.
func TestATransparentCellIsLeftAlone(t *testing.T) {
	clear := color.RGBA{}
	// Just under the floor: a pixel this faint is coverage from anti-aliasing
	// rather than ink, and drawing it would thicken every edge in the picture.
	faint := color.RGBA{R: 10, G: 10, B: 10, A: alphaFloor - 1}
	solid := color.RGBA{R: 200, G: 40, B: 40, A: 255}

	for _, test := range []struct {
		name       string
		top, below color.RGBA
	}{
		{"both empty", clear, clear},
		{"both under the floor", faint, faint},
		{"one empty, one under the floor", clear, faint},
	} {
		if got := blockCell(ink(test.top), ink(test.below), newPalette()); got != " " {
			t.Errorf("%s rendered as %q in colour, want a bare space", test.name, got)
		}
		if got := rampCell(ink(test.top), ink(test.below)); got != " " {
			t.Errorf("%s rendered as %q in monochrome, want a bare space", test.name, got)
		}
	}

	// And a painted cell is never a space, in either drawing: a pixel that reads
	// as nothing turns a filled shape into a hole.
	for _, test := range []struct {
		name       string
		top, below color.RGBA
	}{
		{"the top half", solid, clear},
		{"the bottom half", clear, solid},
		{"both halves", solid, solid},
	} {
		if got := blockCell(ink(test.top), ink(test.below), newPalette()); strings.TrimSpace(got) == "" {
			t.Errorf("%s rendered as %q in colour, want ink", test.name, got)
		}
		if got := rampCell(ink(test.top), ink(test.below)); got == " " {
			t.Errorf("%s rendered as a space in monochrome, want ink", test.name)
		}
	}

	// A fully white pixel is the case that would read as nothing on the ramp, so
	// it is the one worth naming: white ink is still ink.
	white := ink(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if got := rampCell(white, white); got == " " {
		t.Error("a white pixel drew as a space, so a pale shape would come out hollow")
	}
}

// TestAWideWindowShowsTheWholeRestrictionColumn is the column that was being cut
// on a terminal with room to spare.
//
// The listing's last cell was clipped to minWidth, which is the floor a window
// has to clear rather than a ceiling on what one may spend, so the species cell
// reached a hundred-column terminal cut short of the species — a row that has
// stopped saying which species it is for, which is the one thing that column
// exists to say.
//
// The cell is not quoted here on purpose. It was "để dành cho loài dragon" when
// this was written and is "chủng loài dragon" now, and a shorter wording is not
// a fix: it moves the width the clip bites at without moving the clip.
func TestAWideWindowShowsTheWholeRestrictionColumn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m.width, m.height = 160, 60
		m = m.enter(screenSkills)
		m = skillListTo(t, m, "dragon_claw")
		restricted := m.skills.skills[m.skills.cursor]
		summary := m.lang.WhoMaySummary(restricted)
		if !strings.Contains(summary, "dragon") {
			t.Fatalf("%s no longer names a species (%q), so this measures nothing",
				restricted.ID, summary)
		}
		drawn := m.screenContent()
		if !strings.Contains(drawn, summary) {
			t.Errorf("the %s listing never draws %q whole at 160 columns:\n%s",
				lang, summary, drawn)
		}
		// And the width really is what widened it: the same row at the floor has
		// nowhere to put the tail and clips. Only worth asserting for a language
		// whose row does not fit the floor anyway -- English says "kept for a
		// dragon" in half the cells, so there it would be measuring nothing.
		if !rowOverflowsTheFloor(drawn, restricted.ID) {
			continue
		}
		narrow := m
		narrow.width, narrow.height = minWidth, minHeight
		if strings.Contains(narrow.screenContent(), summary) {
			t.Errorf("the %s listing draws %q whole at the floor too, so the "+
				"width it was given is not what widened it", lang, summary)
		}
	}
}

// rowOverflowsTheFloor reports whether the listing row for one id is longer than
// the minimum window, which is what decides whether clipping at the floor has
// anything to cut.
func rowOverflowsTheFloor(drawn, id string) bool {
	for _, line := range strings.Split(drawn, "\n") {
		if strings.Contains(line, id) && lipgloss.Width(line) > minWidth-1 {
			return true
		}
	}
	return false
}
