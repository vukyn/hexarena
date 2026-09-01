package screen

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/testfixture"
)

// A screen here is testable without a terminal and without a client: Update
// takes a key message and hands back a screen and an Action, View takes a
// Context and hands back a body and a footer. Everything below drives the real
// screens against a real library, so none of it needs a model, a program or a
// pty.
//
// ⚠️ **What that cannot reach is the other construction path.** The application
// arrives at these screens through cmd/hexforge-tui's `model.enter`, which
// refreshes against the library first and then draws inside `frame`. A screen
// built here and a screen entered there are two constructions, and a test can
// pass on one while the other is broken — so the client keeps `everyScreen` in
// its `language_test.go` (every one of these six reached through `m.enter`, in
// both languages, with a width, a translation and a leak test each) and
// `testdata/screens.golden` (the same screens as the application draws them,
// byte for byte). Neither may be dropped because the tests below exist.

const shippedDataDir = "../seed/data"

// scratchData copies the shipped data into a temporary directory and injects the
// fixture cast, so these tests name characters of their own rather than whatever
// the repository last shipped.
func scratchData(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	copyTree(t, shippedDataDir, target)
	// The squad catalogue is written by the authoring client and ships with
	// whatever its author last built. Nothing here reads it, and forge.Load takes
	// a missing file as an empty catalogue.
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

// start builds a Context over a scratch copy of the data, in the language asked
// for, sized to a terminal comfortably big enough for any of these screens.
//
// NO_COLOR is set for every test, exactly as the client's own fixture does: the
// styles then render as plain text, which is what lets an assertion look for a
// word rather than for a word wrapped in escape codes. That it works at all is
// the point of the palette — meaning never lives in colour here, in either
// language.
func start(t *testing.T, lang i18n.Lang) (Context, *forge.Library) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	dir := scratchData(t)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	return Context{
		Lib: lib, Lang: lang, Style: NewPalette(plainHere()),
		Width: 120, Height: 44,
	}, lib
}

// plainHere is the palette decision this machine's environment asks for, read
// the way a binary reads it. The fixture sets NO_COLOR, so it is true — and it
// is asked rather than written down so that a test claiming to measure the
// plain rendering can check it got one.
func plainHere() bool {
	return Plain(os.Getenv("NO_COLOR"), os.Getenv("TERM"), runtime.GOOS)
}

// atTheFloor is the same Context in the smallest window this program draws in.
func atTheFloor(c Context) Context {
	c.Width, c.Height = MinWidth, MinHeight
	return c
}

// bodyRoom is how many lines a client's frame leaves a screen's body: the window
// less the two the header pair takes and the two the blank and the footer do.
//
// ⚠️ It is the same `- 4` every Room helper in this package already spends, and
// it is a **mirror** of cmd/hexforge-tui's `frame` rather than its declaration —
// a screen package cannot see the frame that wraps it. What holds the real frame
// is that client's own `screens.golden`, which records these six screens at
// 120x24 as the application draws them, marker line and all.
func bodyRoom(c Context) int { return c.Height - 4 }

// drawnLines is a rendered body as the frame would split it, so a count here is
// the count the frame budgets against.
func drawnLines(body string) []string { return strings.Split(body, "\n") }

// press is a named key as a terminal delivers it.
//
// Only the keys a screen here answers **by itself**, which is why the list is
// short: q, g and p all ask for something a *client* has to carry out — a quit,
// a way back, a raise — so where they land is asserted in cmd/hexforge-tui,
// driven through the real model. An entry here for a key nothing presses is the
// shape that ships dead.
//
// The two arrows walk a cursor; left walks the cast browser's level, and f
// cycles its origin filter. f is printable, so it carries Text as well as Code —
// that is what makes String() report the letter rather than a key name.
//
// ⚠️ The picker is why esc, enter, space and ? are here too, and why they are
// not the exception to the paragraph above: it answers all four **itself** — esc
// and enter take the list down, space toggles a row, ? turns the reading pane on
// — and what a client does with the answer is a separate keystroke's worth of
// work in that package.
//
// ⚠️ Space is a named key rather than a rune, because that is how a terminal
// delivers it: bubbletea v2 turns a bare space into KeySpace, whose String is
// "space" rather than " ". Every `case " "` written the other way compiled fine
// and matched nothing.
func press(t *testing.T, name string) tea.KeyPressMsg {
	t.Helper()
	switch name {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "f":
		return tea.KeyPressMsg{Code: 'f', Text: "f"}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "?":
		return tea.KeyPressMsg{Code: '?', Text: "?"}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	// ⚠️ The two chords are here because the squad builder answers both
	// **itself** — ctrl+s writes the squad through internal/forge and ctrl+x
	// takes a member out — so they are not the exception to the paragraph above
	// either. A modified key carries no Text, which is what stops ctrl+x being
	// read as the letter x by the level field's digit filter.
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "ctrl+x":
		return tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}
	}
	// ⚠️ A single printable character is a **rune with its Text set**, and both
	// halves matter: String() reports the letter off Code, and the skill filter
	// reads Text, because that is the field that tells a typed letter from a
	// chord. A key built with one and not the other passes a test that only
	// switches on String() and types nothing.
	letters := []rune(name)
	if len(letters) == 1 {
		return tea.KeyPressMsg{Code: letters[0], Text: name}
	}
	t.Fatalf("no key named %q in the test helper", name)
	return tea.KeyPressMsg{}
}

// indexOfElement is where one element sits in the listing, which is
// element.All order rather than anything the screen chooses.
func indexOfElement(t *testing.T, want element.Element) int {
	t.Helper()
	for index, member := range element.All() {
		if member == want {
			return index
		}
	}
	t.Fatalf("no element %v", want)
	return 0
}

// firstWords is enough of a free-text value to recognise it by after the line
// it sits on has been wrapped or clipped. It is the client's own helper of the
// same name, copied rather than shared: a test fixture is not code two packages
// may drift over, and moving it would edit language_test.go.
func firstWords(text string) string {
	if len(text) > 20 {
		return text[:20]
	}
	return text
}
