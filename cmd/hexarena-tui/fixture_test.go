package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/testfixture"
)

// A bubbletea model is testable without a terminal: Update takes a message and
// returns a model, and View returns a string. Everything below drives the real
// model with real key messages and reads the real screen, so none of it needs a
// pty, a subprocess or a timing assumption.
//
// ⚠️ **This is a second fixture beside cmd/hexforge-tui's and not a shared
// one.** A test helper is not code two suites may drift over — and the two
// fixtures want opposite things anyway: that one authors a character through the
// form and this one cannot open a form at all, so what it arranges instead is a
// squad catalogue with two sides on it, which is what this client's own screens
// need in order to draw anything at all.

const shippedDataDir = "../../internal/seed/data"

// scratchData copies the shipped data into a temporary directory and injects the
// fixture cast, so these tests name characters of their own rather than whatever
// the repository last shipped.
//
// squads.json is cleared for the reason the authoring client's fixture clears
// it: that file is the author's own working document and ships with whatever
// they last built, so a suite that read it would be measuring somebody's saved
// sides. This client's catalogue is then filled by twoSidesSaved, in a shape the
// tests can make an assertion about.
func scratchData(t *testing.T) string {
	t.Helper()
	return scratchDataFrom(t, shippedDataDir)
}

// scratchDataFrom is scratchData reading the shipped books from a directory the
// caller names.
//
// ⚠️ **The golden's fixture needs it because that one changes the working
// directory**, so `shippedDataDir` — a relative path — would resolve somewhere
// else on the second language's turn. It resolves the source once, before the
// move, and hands it down. The authoring client's golden solves the same problem
// by copying the shipped tree to wherever the constant lands from the new
// directory; naming the source is the same fix with nothing to keep in step.
func scratchDataFrom(t *testing.T, shipped string) string {
	t.Helper()
	target := t.TempDir()
	copyTree(t, shipped, target)
	if err := os.Remove(filepath.Join(target, "squads.json")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("clear the squad catalogue: %v", err)
	}
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
// for, sized to a terminal comfortably big enough for any of these screens, with
// two sides in the squad catalogue.
//
// NO_COLOR is set for every test, exactly as both other suites do: the styles
// then render as plain text, which is what lets an assertion look for a word
// rather than for a word wrapped in escape codes. That it works at all is the
// point of the palette — meaning never lives in colour here, in either language.
func start(t *testing.T, lang i18n.Lang) (model, *forge.Library, string) {
	t.Helper()
	return startIn(t, lang, scratchData(t))
}

// startIn is start over a data directory the test has already arranged, and it
// is what the golden's relative-directory fixture needs.
func startIn(t *testing.T, lang i18n.Lang, dir string) (model, *forge.Library, string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	twoSidesSaved(t, lib)
	m := newModel(lib, lang, newSession())
	m.width, m.height = 120, 44
	return m, lib, dir
}

// startEmpty is start with the squad catalogue left as scratchData leaves it:
// nothing on it.
//
// It is what two states need and neither of them is reachable any other way —
// the listing with no rows, and a battle with no pairing to open on, which is
// the arm draw.PlayScreen.Open takes when a client hands it two empty squads.
func startEmpty(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	dir := scratchData(t)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	if len(lib.Squads()) != 0 {
		t.Fatalf("the scratch catalogue holds %d sides, so the empty states are drawn by "+
			"nothing", len(lib.Squads()))
	}
	m := newModel(lib, lang, newSession())
	m.width, m.height = 120, 44
	return m
}

// sideSize is how many units each fixture side brings.
//
// ⚠️ **Three rather than one, and it is not for realism.** At the declared
// 120x24 floor a battle's body has twenty rows, and a one-a-side board, roster,
// order line and option list come to exactly twenty — so nothing is ever
// dropped, the notice naming what the window was too short for is drawn by
// nothing, and every state registered in the sweep would measure the roomy
// screen twice. Three a side is 24 rows against 20, which is where the budget
// starts deciding, and it is the size the same fixture in internal/screen is
// built at for the same reason.
const sideSize = 3

// twoSidesSaved writes two sides into the catalogue, built around two different
// characters.
//
// ⚠️ **Two, and around characters that differ, and both halves are load-bearing.**
// Two because a catalogue with one row measures no id column — the column is
// sized over the rows *and* the header — and because this client's pairing takes
// the *next* side on the file as the opponent, so one row makes every battle a
// side against a copy of itself. Different characters because two identical
// sides make the two halves of a battle interchangeable: the board, the roster
// and the order line would draw the same thing whichever way round they were
// fielded, and a client that opened the battle with the sides swapped would look
// exactly like one that did not.
func twoSidesSaved(t *testing.T, lib *forge.Library) []placement.Squad {
	t.Helper()
	characters := lib.Characters().All()
	built := make([]placement.Squad, 0, 2)
	for _, character := range characters {
		squad := aSideOf(t, character, len(built))
		if err := lib.SaveSquad(squad); err != nil {
			// A character the fixture cannot field is ordinary — a learnset the
			// slots cannot fill, art nobody wrote — so the next one is tried
			// rather than the fixture failing on the first.
			continue
		}
		built = append(built, squad)
		if len(built) == 2 {
			// ⚠️ Asserted rather than assumed, because it is the whole reason the
			// loop walks the book instead of taking the first character twice: two
			// sides of the same character make the halves of a battle
			// interchangeable, and a client that opened one with the sides swapped
			// would draw exactly what a correct one draws.
			if built[0].Units[0].Character == built[1].Units[0].Character {
				t.Fatalf("both fixture sides field %q, so the two halves of a battle are "+
					"indistinguishable", built[0].Units[0].Character)
			}
			return built
		}
	}
	t.Fatalf("only %d of the %d characters in the book could be fielded, so the fixture "+
		"has no two sides to put against each other", len(built), len(characters))
	return nil
}

// aSideOf is one side of sideSize units, all of the same character, in distinct
// cells.
//
// The same character throughout because what the fixture needs is two sides that
// differ from *each other*, not a squad with a story; distinct cells because
// placement refuses two units in one, and distinct ids because a squad's own ids
// have to tell its units apart before Take prefixes them with a side.
func aSideOf(t *testing.T, character cast.Character, which int) placement.Squad {
	t.Helper()
	units := make([]placement.Placement, 0, sideSize)
	for index := range sideSize {
		unit := placement.Placement{
			ID:        "don-vi-" + strconv.Itoa(index),
			Character: character.ID,
			Level:     progression.LevelCap,
			Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: index},
		}
		kit := character.SkillsAt(unit.Level, progression.Furthest)
		if len(kit) > cast.SkillSlots {
			kit = kit[:cast.SkillSlots]
		}
		unit.Skills = kit
		if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
			unit.Passives = traits[:cast.TraitSlots]
		}
		units = append(units, unit)
	}
	return placement.Squad{
		ID:    "phe-" + strconv.Itoa(which),
		Name:  "đội thử " + strconv.Itoa(which),
		Units: units,
	}
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
	return send(t, m, press(t, name))
}

// press is what each name sends.
//
// ⚠️ **Each entry sends what its name says**, which is the vacuity the authoring
// client's TestABracketIsTheKeystrokeItLooksLike exists to catch: a helper
// mapping "[" to a KeyPgUp would satisfy every alias assertion while proving
// nothing at all. The single-rune fall-through at the bottom is what makes a
// printable key the keystroke it looks like.
//
// A modified key carries no Text, which is what stops ctrl+s being read as the
// letter s. Space is a named key rather than a rune because that is how a
// terminal delivers it: bubbletea v2 turns a bare space into KeySpace, whose
// String is "space" rather than " ", and every `case " "` written the other way
// compiled fine and matched nothing.
func press(t *testing.T, name string) tea.KeyPressMsg {
	t.Helper()
	if press, known := namedKeys[name]; known {
		return press
	}
	letters := []rune(name)
	if len(letters) == 1 {
		return tea.KeyPressMsg{Code: letters[0], Text: name}
	}
	t.Fatalf("no key named %q in the test helper", name)
	return tea.KeyPressMsg{}
}

var namedKeys = map[string]tea.KeyPressMsg{
	"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown},
	"left": {Code: tea.KeyLeft}, "right": {Code: tea.KeyRight},
	"enter": {Code: tea.KeyEnter}, "esc": {Code: tea.KeyEscape},
	"tab":   {Code: tea.KeyTab},
	"space": {Code: tea.KeySpace, Text: " "},
	"pgup":  {Code: tea.KeyPgUp}, "pgdown": {Code: tea.KeyPgDown},
	"home": {Code: tea.KeyHome}, "end": {Code: tea.KeyEnd},
	"backspace": {Code: tea.KeyBackspace},
	"ctrl+s":    {Code: 's', Mod: tea.ModCtrl},
	"ctrl+x":    {Code: 'x', Mod: tea.ModCtrl},
	"ctrl+l":    {Code: 'l', Mod: tea.ModCtrl},
}

// everyKeyPressed is every keystroke this suite can send, which is what the
// sweeps that press *everything* walk.
//
// It is the named keys plus every printable character a screen in
// internal/screen switches on, so a sweep over it reaches every arm of every
// Update those screens have — including the ones this client is supposed to
// ignore.
func everyKeyPressed() []string {
	named := make([]string, 0, len(namedKeys))
	for name := range namedKeys {
		named = append(named, name)
	}
	// Sorted, because a sweep whose order comes off a map range is a sweep whose
	// failures arrive in a different order every run — the same discipline
	// internal/core states about map iteration reaching an output.
	sortStrings(named)
	for _, letter := range "abcdefghijklmnopqrstuvwxyz0123456789/?[]+-" {
		named = append(named, string(letter))
	}
	return named
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

// typeText sends one rune per message, which is what a keyboard does.
//
// Sending a whole word in one message would be a lie in a way that matters: a
// key carrying several characters stringifies to that word, so "up" typed in one
// go would be routed as the up arrow rather than as two letters.
func typeText(t *testing.T, m model, text string) model {
	t.Helper()
	for _, letter := range text {
		m = send(t, m, tea.KeyPressMsg{Code: letter, Text: string(letter)})
	}
	return m
}

// quits reports whether a command is tea.Quit.
func quits(command tea.Cmd) bool {
	if command == nil {
		return false
	}
	_, isQuit := command().(tea.QuitMsg)
	return isQuit
}

// freeText is the authored **prose** on screen: biographies, notes, and the
// directory the books were read from — text that takes a whole line, and whose
// line therefore has no length the program can promise.
//
// A width or a translation sweep may not measure any of it. A biography is
// English in cast.json and will still be English on a Vietnamese screen — it is
// the author's prose, not the program's — and a path is as long as whoever filed
// the art made it, which is exactly why frame clips rather than promises.
//
// ⚠️ **Prose only. A name is in freeNames and exempts its own cell instead.**
// The two used to be one list, and because carriesFreeText exempts the *line* a
// value sits on, a row like `dạng < Poliwrath > 1/2, bấm s để đổi` was skipped
// whole: the stage name in the middle of it bought the program's wording around
// it an exemption it had not earned. i18n.FormChoice could be lengthened to 155
// cells with both clients' width sweeps still green. Prose keeps the line
// exemption because a paragraph really is the whole row; a name gives up only the
// cells it occupies, and the wording either side of it stays measured.
//
// It is the authoring client's list of the same name with the entries this
// client cannot draw taken out, and it is a copy for the reason the fixture
// itself is: reaching across would edit that package.
func freeText(lib *forge.Library) []string {
	free := []string{lib.Dir()}
	for _, character := range lib.Characters().All() {
		if character.Bio != "" {
			free = append(free, character.Bio)
		}
	}
	for _, origin := range lib.Origins().All() {
		if origin.Note != "" {
			free = append(free, origin.Note)
		}
	}
	for _, kind := range lib.Species().All() {
		if kind.Note != "" {
			free = append(free, kind.Note)
		}
	}
	return free
}

// freeNames is every authored **name** on screen: a character's, a form's, an
// origin's title, and the name and id of a side.
//
// freeText's other half, and the one difference that matters is what it exempts.
// A name is a **cell inside a measured row** — `dạng < Poliwrath > 1/2, bấm s để
// đổi` is one authored word with the program's own wording either side of it — so
// withoutNames takes the name out and the sweep measures what is left. A row
// carrying a long authored name then costs exactly that name's width and keeps
// its promise about everything else on it.
//
// **Longest first**, which withoutNames relies on: `Mew` is a shipped stage name
// and a substring of `Mewtwo`, and stripping the short one first would leave
// `two` behind to be measured as though the program had written it.
//
// ⚠️ **A name can be a substring of ordinary wording, and that costs measuring
// power rather than correctness.** Nothing here knows *where* on the row a name
// was drawn, so a name that also occurs inside a sentence is taken out of the
// sentence too. The error only ever runs one way: the remainder is shorter than
// what was drawn, so the strip can **hide** a breach and can never invent one.
//
// **Measured rather than argued**, over every line of every screen of both
// clients in both languages. The shortest shipped name is `Mew`, three cells, and
// no name is taken out of more lines than the rows that name it. The most any one
// line gives up is 33 cells — the stage-summary row, which is four form names on
// one line — and 39 cells of program wording are still measured on it. And the
// figure that decides the question: the strip takes **no** line anywhere from over
// the floor to under it, so it is at present hiding nothing at all.
//
// So no minimum length is imposed, and a floor would cost more than it bought:
// `Mew` left out of the strip means the Mewtwo pane is measured with its own name
// on the row, which is a **failure invented** out of authored data — the one
// direction this must not go. If a future book names a form after a common word,
// the number to watch is that rescued-line count, not the name's length.
func freeNames(lib *forge.Library) []string {
	var names []string
	for _, character := range lib.Characters().All() {
		names = append(names, character.Name)
		for _, stage := range forge.StageFacts(character) {
			names = append(names, stage.Name)
		}
	}
	for _, origin := range lib.Origins().All() {
		names = append(names, origin.Title)
	}
	// A side's own name and id are authored in squads.json — by the other
	// front-end, which is the whole point of this client reading the file — so a
	// catalogue row and a battle's own file name are as long as whoever built the
	// side made them.
	for _, squad := range lib.Squads() {
		names = append(names, squad.Name, squad.ID)
	}
	slices.SortFunc(names, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	return names
}

// withoutNames is the line with the authored names taken out of it, so that what
// is measured is the program's own wording.
//
// names must be longest first — freeNames is its only source and sorts there,
// because the order is a property of the list rather than of any one call.
func withoutNames(line string, names []string) string {
	for _, name := range names {
		if name == "" {
			continue
		}
		line = strings.ReplaceAll(line, name, "")
	}
	return line
}

// whoMayCarry is the restriction column of the skills listing, for every skill
// in the book.
//
// It is free text for the width measurement and not for the language one, which
// is why it is a second list rather than part of freeText: the wording around
// the data ("hệ ...", "chủng loài ...") is the program's and has to be
// translated, but the data it wraps — an element, a species, a work — has no
// length the program can promise.
func whoMayCarry(lang i18n.Lang, lib *forge.Library) []string {
	out := make([]string, 0, len(lib.Skills().Skills()))
	for _, declared := range lib.Skills().Skills() {
		if summary := lang.WhoMaySummary(declared); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

// traitCarriers is the "who carries it" column of the traits listing.
//
// whoMayCarry's twin, exempt for the same reason: the cell is character ids the
// book named, and a listing that clipped it to the floor would hide which
// characters hold a trait on a terminal with columns to spare.
func traitCarriers(lib *forge.Library) []string {
	out := make([]string, 0, len(lib.Passives().All()))
	for _, held := range lib.Passives().All() {
		if summary := forge.TraitCarrierSummary(lib.TraitCarriers(held.ID)); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

// kitGlosses is the dim row of Vietnamese skill names under the kit on the cast
// browser's detail pane, for every character in the book.
//
// whoMayCarry's third twin, exempt for the same reason: the cell is the names
// the book gave a character's own skills, so it is as long as whoever wrote the
// kit made it — sixteen of them on pokemon.poliwag — and no wording of the
// program's is on the row at all. It is a separate list from the other two
// because the three cells are built by different callers, and a single list of
// "data cells" is a list nobody would remember to add to.
//
// ⚠️ **This row is wrapped rather than clipped, which is why the floor is the
// wrong question to ask it.** WrappedIn spends UsableWidth() - 1 - 2 - width - 1,
// so at the sweep's 200 columns the row fills 199 of them by construction and a
// floor assertion over it would fail on data doing exactly what it should.
// What holds that arithmetic instead is internal/screen's
// TestAWrappedRowLeavesTheWindowsLastColumnEmpty, which measures the room rather
// than the wording. Nothing else on the pane is exempted: the neighbouring dim
// rows — the archetype's name, the element's, the species', the traits' and the
// pierced-floor reading — are all still measured against the floor.
//
// It is empty in English, where GlossedKit draws nothing at all, so the English
// sweep gives up nothing for it.
func kitGlosses(lang i18n.Lang, lib *forge.Library) []string {
	out := make([]string, 0, len(lib.Characters().All()))
	for _, character := range lib.Characters().All() {
		if glossed := lang.GlossedKit(lib.KitSkills(cast.LearnedIDs(character.Skills))); glossed != "" {
			out = append(out, glossed)
		}
	}
	return out
}

// kitIDs is the pair of id rows on the cast browser's detail pane — the kit and
// the traits — for every character in the book, at every level either of them
// changes at.
//
// kitGlosses' twin one row up, and exempt for the same reason: the cell is the
// authored ids of a character's own learnset, sixteen of them on
// pokemon.poliwag, and no length the program can promise is on the row at all. It
// is wrapped like the glossed row under it, so the floor is the wrong question to
// ask it for the same arithmetic reason recorded there.
//
// ⚠️ **It exists so that the row is exempt by kind rather than by accident.** The
// forked pane's kit row is 181 cells and used to pass the width sweep because it
// happens to contain `submission[Poliwrath]` and every form name used to exempt
// its whole line — an exemption that would have gone the day a form was renamed,
// which is exactly the failure traitCarriers records having lived through. The
// traits row goes in beside it because it is the same call on the other
// learnset, not because it is long today: a column exempted by length is a column
// waiting for the next row.
//
// **Levels, because UnlockSummaryAt is read from one.** The summary prints a gate
// only while it is still ahead, so the string changes as the level is walked, and
// carriesFreeText recognises a value by its opening. It can only change at a
// level some entry unlocks at, so those levels and level one are the whole set —
// enumerated rather than sampled, so a fixture that walks to a new level does not
// quietly stop matching.
func kitIDs(lib *forge.Library) []string {
	var out []string
	for _, character := range lib.Characters().All() {
		learnsets := [][]cast.Unlock{character.Skills, character.Passives}
		levels := []int{1}
		for _, learnset := range learnsets {
			for _, entry := range learnset {
				if !slices.Contains(levels, entry.AtLevel) {
					levels = append(levels, entry.AtLevel)
				}
			}
		}
		for _, level := range levels {
			for _, learnset := range learnsets {
				if summary := forge.UnlockSummaryAt(learnset, level); summary != "" {
					out = append(out, summary)
				}
			}
		}
	}
	return out
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

// partOfFreeText reports whether a line is one *wrapped* line of authored prose.
//
// carriesFreeText recognises a value by its opening, which is enough for text
// that is clipped. It cannot see the second line of text that is **wrapped**, so
// the test here is containment the other way round: a wrapped line is a run of
// the original's own words, so the original contains it. Short lines are refused
// because a two-word one would be contained by half the prose in the book.
func partOfFreeText(line string, free []string) bool {
	const enoughToBeSure = 12
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < enoughToBeSure {
		return false
	}
	for _, text := range free {
		if strings.Contains(text, trimmed) {
			return true
		}
	}
	return false
}

// firstWords is enough of a free-text value to recognise it by after the line it
// sits on has been clipped to the window.
func firstWords(text string) string {
	if len(text) > 20 {
		return text[:20]
	}
	return text
}
