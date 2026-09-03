package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The three read-only views on a line that forks
//
// The defect these hold, as a user met it: pressing `p` on pokemon.poliwag at
// any level from 32 up drew nothing but
//
//	level 46 reaches [Poliwrath Politoed], which are alternatives: name the one being fielded
//
// in red, and the same character's detail pane ended at that line as well. The
// refusal is progression.Line.StageAt's and it is **right** — handing a browser
// whichever arm the file lists last is a wrong stat line, a wrong picture and a
// wrong trait list with nothing on screen saying so. What was missing was a way
// for a reader to name an arm, which is what these measure.
//
// ⚠️ **Nothing in the repository could see this before.** The art preview was
// only swept at all as of the change before this one, and the fixture it sweeps —
// naruto.naruto, and both clients' own fixture characters — is a line that does
// not fork, so every screen record and every sweep walked past the one shipped
// character that could not be drawn. The entries added beside these tests, in
// this package's golden and in both clients', are the other half of the fix.

// theShippedFork is the character whose evolution line forks, found rather than
// named.
//
// ⚠️ **Failing when there is none is the point.** A fork is a fact about the
// shipped data, so a helper that skipped when it found none would turn "the data
// changed" into "these tests measure nothing" without a word — which is the exact
// shape of the gap this file was written to close. The level is one the fork is
// open at, and it is deliberately **not** the cap: the cap is where every other
// entry sits, and a fork that only worked there would be a fork nothing walked
// into.
func theShippedFork(t *testing.T, lib *forge.Library) (cast.Character, int) {
	t.Helper()
	for _, character := range lib.Characters().All() {
		arms, err := character.FurthestAt(forkLevel)
		if err != nil || len(arms) < 2 {
			continue
		}
		return character, forkLevel
	}
	t.Fatalf("no shipped character's line forks at level %d, so nothing here measures "+
		"the case the art preview refused to draw", forkLevel)
	return cast.Character{}, 0
}

// forkLevel is a level the shipped fork is open at. Fourteen levels short of the
// cap, which is where the user met the refusal.
const forkLevel = 46

// TestAForkingLineIsPreviewedRatherThanRefused is the bug, and it is the one
// assertion in this file that fails on the code as it was: the preview resolved
// with progression.Furthest, took the refusal and drew it in red instead of a
// picture.
func TestAForkingLineIsPreviewedRatherThanRefused(t *testing.T) {
	c, lib := start(t, i18n.En)
	character, level := theShippedFork(t, lib)
	browser := browserOn(t, lib, character.ID, level)
	preview := NewPreviewScreen()
	preview.Subject = browser.Subject()
	drawn, _ := preview.View(c)

	// The refusal is asked for rather than quoted, so this reads the same wording
	// the screen would have drawn even after somebody rewords it.
	_, _, err := character.Resolve(level, progression.Furthest)
	if err == nil {
		t.Fatalf("%s resolves at level %d with nobody choosing, so this test is no "+
			"longer about a fork", character.ID, level)
	}
	if refusal := c.Lang.Error(err); strings.Contains(drawn, refusal) {
		t.Errorf("the art preview of %s at level %d draws the refusal instead of a "+
			"picture:\n%s", character.ID, level, drawn)
	}
	arms := FormArms(character, level)
	if !strings.Contains(drawn, arms[0].Name) {
		t.Errorf("the preview names no arm, so a reader cannot tell which form the "+
			"picture is:\n%s", drawn)
	}
	if !strings.Contains(drawn, character.StageArt(arms[0])) {
		t.Errorf("the preview does not name %s, the picture the first arm shows:\n%s",
			character.StageArt(arms[0]), drawn)
	}
	if !aRampRow(pictureRowOf(drawn)) {
		t.Errorf("the preview draws no row of art at all:\n%s", drawn)
	}
}

// TestAForkIsNeverPickedSilently is the property the whole shape rests on.
//
// StageAt refuses rather than take the last arm in the file, because a form
// chosen for somebody and not named is a stat line, a picture and a trait list
// that all belong to a character the reader did not ask about. So whatever these
// views settle on — including the arm a chooser has to open on before anybody
// has pressed anything — must be **on the screen**, on all three of them, in
// both languages.
func TestAForkIsNeverPickedSilently(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		c, lib := start(t, lang)
		character, level := theShippedFork(t, lib)
		browser := browserOn(t, lib, character.ID, level)
		// Once per arm, so the walk visits every one of them and not only the one a
		// chooser opens on.
		for range FormArms(character, level) {
			settled := ChosenForm(character, level, browser.Form)
			if settled == progression.Furthest {
				t.Fatalf("%s at level %d settled on no form at all", character.ID, level)
			}
			preview := NewPreviewScreen()
			preview.Subject = browser.Subject()
			previewBody, _ := preview.View(c)
			blurbBody, _ := BlurbScreen{Subject: browser.Subject()}.View(c)
			detail := browser.Detail(c, character)
			for name, drawn := range map[string]string{
				"the art preview":  previewBody,
				"the traits blurb": blurbBody,
				"the detail pane":  detail,
			} {
				if !strings.Contains(drawn, settled) {
					t.Errorf("in %v, %s reads %s as %s and never says so:\n%s",
						lang, name, character.ID, settled, drawn)
				}
			}
			browser = browser.CycleForm()
		}
	}
}

// TestTheFormKeyWalksEveryArmAndComesBack is the chooser: `s` steps to the next
// arm, every arm is reachable, and the walk is a cycle rather than a dead end at
// the last one.
func TestTheFormKeyWalksEveryArmAndComesBack(t *testing.T) {
	c, lib := start(t, i18n.En)
	character, level := theShippedFork(t, lib)
	browser := browserOn(t, lib, character.ID, level)
	arms := FormArms(character, level)

	seen := make([]string, 0, len(arms))
	for range arms {
		seen = append(seen, ChosenForm(character, level, browser.Form))
		next, action := browser.Update(c, press(t, "s"))
		if action.Kind != Stay {
			t.Fatalf("the form key asked for %v, and it navigates nowhere", action.Kind)
		}
		browser = next
	}
	for _, arm := range arms {
		if !contains(seen, arm.Name) {
			t.Errorf("walking the form key visited %v, which misses the arm %s",
				seen, arm.Name)
		}
	}
	if back := ChosenForm(character, level, browser.Form); back != seen[0] {
		t.Errorf("%d presses of the form key over %d arms landed on %s, want back on %s",
			len(arms), len(arms), back, seen[0])
	}
}

// TestTheThreeViewsReadOneForm is why the choice is carried on the Subject
// rather than settled in each describer.
//
// The picture, the traits and the stat line are three readings of one character,
// and a reader walks between them with two keys. Three settlings of the same
// fork is three things free to open on a different arm, so walking from the
// picture to the traits would change the form for a reason nothing on either
// screen said.
func TestTheThreeViewsReadOneForm(t *testing.T) {
	c, lib := start(t, i18n.En)
	character, level := theShippedFork(t, lib)
	browser := browserOn(t, lib, character.ID, level)
	// The second arm, so a disagreement shows up as the first one rather than as
	// nothing at all: every view opening on arms[0] would pass a comparison
	// between three copies of the same default.
	browser = browser.CycleForm()
	arms := FormArms(character, level)
	settled := ChosenForm(character, level, browser.Form)
	if settled == arms[0].Name {
		t.Fatalf("the form key left the first arm in front, so this compares defaults")
	}
	preview := NewPreviewScreen()
	preview.Subject = browser.Subject()
	previewBody, _ := preview.View(c)
	if !strings.Contains(previewBody, character.StageArt(armNamed(t, arms, settled))) {
		t.Errorf("the preview draws a picture that is not %s's:\n%s", settled, previewBody)
	}
	detail := browser.Detail(c, character)
	if !strings.Contains(detail, c.Text(i18n.StageInWords, settled)) {
		t.Errorf("the detail pane's stat row is not %s's:\n%s", settled, detail)
	}
	// The trait list is the reading with no stage name of its own in it, which is
	// why it is measured through what it actually draws: a trait gated on the
	// other arm must not be in front.
	blurbBody, _ := BlurbScreen{Subject: browser.Subject()}.View(c)
	for _, other := range arms {
		if other.Name == settled {
			continue
		}
		for _, gated := range gatedOn(character, level, other.Name, settled) {
			if strings.Contains(blurbBody, gated) {
				t.Errorf("the traits blurb reads %s as %s but carries %s, which only "+
					"%s holds:\n%s", character.ID, settled, gated, other.Name, blurbBody)
			}
		}
	}
}

// TestALineThatDoesNotForkIsExactlyWhatItWas is the other half of the promise,
// and the one the goldens are read against: eleven of the twelve shipped
// characters have one grown form, so none of them draws a form row, none of them
// answers the form key, and every Resolve behind them is the call it always made.
func TestALineThatDoesNotForkIsExactlyWhatItWas(t *testing.T) {
	c, lib := start(t, i18n.En)
	forked := 0
	for _, character := range lib.Characters().All() {
		if len(FormArms(character, progression.LevelCap)) > 1 {
			forked++
			continue
		}
		if row := FormRow(c, character, progression.LevelCap, ""); row != "" {
			t.Errorf("%s does not fork and still draws a form row: %q", character.ID, row)
		}
		if got := ChosenForm(character, progression.LevelCap, "Whatever"); got != progression.Furthest {
			t.Errorf("%s does not fork and reads as %q, want progression.Furthest",
				character.ID, got)
		}
		browser := browserOn(t, lib, character.ID, progression.LevelCap)
		before := browser.Subject()
		if after := browser.CycleForm().Subject(); after != before {
			t.Errorf("the form key moved %s, which has one form: %+v became %+v",
				character.ID, before, after)
		}
	}
	if forked == 0 {
		t.Fatal("no shipped character forks, so the paragraph above measures nothing")
	}
}

// TestTheFormRowIsPaidForOutOfThePicture is the row arithmetic, which is the
// half of this change a wording test cannot see.
//
// previewChrome was guessed once and was three rows short, and the frame
// answered by replacing the bottom row of the drawing with its "there was more
// than this" notice. A fork row written without being counted is that defect
// again, on exactly the character the row was added for — so the drawing gives
// the row back, and a forked preview and a linear one fill the same window to
// the same height.
func TestTheFormRowIsPaidForOutOfThePicture(t *testing.T) {
	c, lib := start(t, i18n.En)
	character, level := theShippedFork(t, lib)
	forked := NewPreviewScreen()
	forked.Subject = browserOn(t, lib, character.ID, level).Subject()
	forkedBody, _ := forked.View(c)

	linear, found := aLinearCharacterWithArt(t, c, lib)
	if !found {
		t.Skip("no shipped character has both a line that does not fork and art on disk")
	}
	plain := NewPreviewScreen()
	plain.Subject = browserOn(t, lib, linear.ID, level).Subject()
	plainBody, _ := plain.View(c)

	if forkedRows, plainRows := len(drawnLines(forkedBody)), len(drawnLines(plainBody)); forkedRows != plainRows {
		t.Errorf("in the same %dx%d window the forked preview is %d rows and the linear "+
			"one is %d: the fork row is not being paid for out of the drawing",
			c.Width, c.Height, forkedRows, plainRows)
	}
}

// browserOn is the cast browser with its cursor on a named character at a named
// level, which is the state every raise in this file is built from.
func browserOn(t *testing.T, lib *forge.Library, id string, level int) BrowseScreen {
	t.Helper()
	browser := NewBrowseScreen(lib)
	browser.Level = level
	for index, row := range browser.Rows() {
		if row.ID == id {
			browser.Cursor = index
			return browser
		}
	}
	t.Fatalf("the cast browser lists no %s", id)
	return browser
}

// aLinearCharacterWithArt is a character the preview can draw a picture of and
// whose line does not fork, which is what the row arithmetic is compared against.
func aLinearCharacterWithArt(t *testing.T, c Context, lib *forge.Library) (cast.Character, bool) {
	t.Helper()
	for _, character := range lib.Characters().All() {
		arms := FormArms(character, forkLevel)
		if len(arms) != 1 {
			continue
		}
		if c.Lib.ImageExists(character.StageArt(arms[0])) {
			return character, true
		}
	}
	return cast.Character{}, false
}

// armNamed is one of the arms by name, which a caller has already settled on.
func armNamed(t *testing.T, arms []progression.Stage, name string) progression.Stage {
	t.Helper()
	for _, arm := range arms {
		if arm.Name == name {
			return arm
		}
	}
	t.Fatalf("%s is not one of %v", name, progression.StageNames(arms))
	return progression.Stage{}
}

// gatedOn is the trait names only one arm carries: what the character holds as
// that arm, less what it holds as the arm in front.
//
// Names rather than ids, because the blurb draws the glossed name and an id it
// has a name for never reaches the screen.
func gatedOn(character cast.Character, level int, arm, inFront string) []string {
	held := map[string]bool{}
	for _, id := range character.PassivesAt(level, inFront) {
		held[id] = true
	}
	out := make([]string, 0, 2)
	for _, id := range character.PassivesAt(level, arm) {
		if !held[id] {
			out = append(out, id)
		}
	}
	return out
}

// pictureRowOf is a row of a drawn preview that ought to be art: the last
// non-empty one, which is inside the drawing at every window this fixture uses.
func pictureRowOf(body string) string {
	lines := drawnLines(body)
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) != "" {
			return lines[index]
		}
	}
	return ""
}

func contains(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}
