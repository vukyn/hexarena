package screen

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/i18n"
)

// What the works catalogue draws, and how its cursor, its two modes and its five
// fields behave.
//
// The client's half stays in cmd/hexforge-tui — which menu entry reaches this
// screen, where esc lands, whether q ends the program, and the whole end-to-end
// write. ⚠️ **Nothing here writes a file**, which is the rule this package's
// squad tests already state: `save` reaches a real directory through
// internal/forge, and a suite where some tests write and others do not is one
// edit away from a golden that authored a work into the repository. So every
// state below is a **value** — `Origins` and `Counts` are fields, exactly as the
// squad builder's `Saved` is.

// pressOrigins sends one named key and hands back the screen and what it asked
// the client for.
func pressOrigins(t *testing.T, c Context, o OriginsScreen, name string) (OriginsScreen, Action) {
	t.Helper()
	next, action, _ := o.Update(c, press(t, name))
	return next, action
}

// onOrigins is pressOrigins for a test that is not asking about the action.
func onOrigins(t *testing.T, c Context, o OriginsScreen, name string) OriginsScreen {
	t.Helper()
	next, _ := pressOrigins(t, c, o, name)
	return next
}

// typeIntoOrigins sends one rune per message, which is what a keyboard does.
func typeIntoOrigins(t *testing.T, c Context, o OriginsScreen, text string) OriginsScreen {
	t.Helper()
	for _, letter := range text {
		o = onOrigins(t, c, o, string(letter))
	}
	return o
}

// TestEscFromTheWorksCatalogueAsksTheClientToGoBack is the one key on this
// screen that used to name a view of the client that drew it.
//
// It wrote `m.screen = screenMenu`, which is true today and is a fact about
// which client is in front rather than about the catalogue — the same argument
// the chart's own way back and the squad catalogue's were converted under. `q` is
// checked beside it because the two are not the same answer here: a quit ends
// the session and a step back does not, and a screen that returned one for the
// other would read as working from the menu.
func TestEscFromTheWorksCatalogueAsksTheClientToGoBack(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	if _, action := pressOrigins(t, c, NewOriginsScreen(c), "esc"); action.Kind != Back {
		t.Errorf("esc on the catalogue asked for %v, want a step back", action.Kind)
	}
	if _, action := pressOrigins(t, c, NewOriginsScreen(c), "q"); action.Kind != Quit {
		t.Errorf("q on the catalogue asked for %v, want the session to end", action.Kind)
	}
}

// TestTheWorkDiscardQuestionNamesWhatIsBeingThrownAway is the half of escaping
// the add-a-work form that belongs to this screen: whether it asks at all, and
// in which words.
//
// ⚠️ **Both arms, because one alone measures nothing.** A form nobody has typed
// into closes on the first escape — asking about an empty form is a question with
// no content — so a screen that asked unconditionally and a screen that never
// asked both pass a test that drives only one of them.
//
// The question carries **no subject**: this screen has one question and it is
// about the screen itself, which is why it passes nil where the squad builder
// passes a SquadsAsk. That is asserted rather than left implicit — an About the
// client kept and handed back would reach a Confirmed that ignores it, so a
// value appearing there would be silent.
func TestTheWorkDiscardQuestionNamesWhatIsBeingThrownAway(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	untouched := onOrigins(t, c, NewOriginsScreen(c), "a")
	if !untouched.Adding {
		t.Fatal("a did not open the add-a-work form")
	}
	closed, action := pressOrigins(t, c, untouched, "esc")
	if action.Kind != Stay {
		t.Errorf("escaping an untyped form asked for %v, want nothing of the client", action.Kind)
	}
	if closed.Adding {
		t.Error("escaping an untyped form left it open")
	}

	typed := typeIntoOrigins(t, c, untouched, "fixture-work")
	if !typed.Touched {
		t.Fatal("typing an id left the form reading as untouched, so escape asks nothing")
	}
	still, asked := pressOrigins(t, c, typed, "esc")
	if asked.Kind != Ask {
		t.Fatalf("escaping a typed-into form asked for %v rather than a question", asked.Kind)
	}
	if asked.Question != i18n.OriginFormDiscard {
		t.Errorf("the question asked was %v", asked.Question)
	}
	if asked.About != nil {
		t.Errorf("the question is about %v; this screen names nothing", asked.About)
	}
	if !still.Adding {
		t.Error("asking took the form down; a question is drawn over what is in front")
	}

	// And the answer empties it, which is the other end of the same key.
	answered, after := still.Confirmed(c, nil)
	if after.Kind != Stay {
		t.Errorf("confirming the discard asked for %v, want to stay on the listing", after.Kind)
	}
	if answered.Adding || answered.Touched {
		t.Error("the form that came back is still open or still claims changes")
	}
	if got := answered.Inputs[OriginFieldID].Value(); got != "" {
		t.Errorf("the id field still reads %q, want the typed work thrown away", got)
	}
}

// TestTheWorksCursorWalksOneRowAtATimeAndStopsAtTheEnds is the cursor, and it is
// driven from the **middle** of the catalogue rather than from an end.
//
// ⚠️ That is the whole design of it. A cursor sitting on the last row cannot tell
// `+1` from correct, because the clamp catches both — #223 measured exactly that
// on the squad catalogue — so the walk starts on the second row of at least three
// and asserts each step lands on the row it names, in both vocabularies.
func TestTheWorksCursorWalksOneRowAtATimeAndStopsAtTheEnds(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	o := NewOriginsScreen(c)
	if len(o.Origins) < 3 {
		t.Skipf("the catalog holds %d works, too few to walk from the middle", len(o.Origins))
	}
	for _, keys := range [][2]string{{"up", "down"}, {"k", "j"}} {
		back, forward := keys[0], keys[1]
		middle := o
		middle.Cursor = 1
		if got := onOrigins(t, c, middle, back).Cursor; got != 0 {
			t.Errorf("%q off row 1 landed on row %d, want 0", back, got)
		}
		if got := onOrigins(t, c, middle, forward).Cursor; got != 2 {
			t.Errorf("%q off row 1 landed on row %d, want 2", forward, got)
		}
		top := o
		top.Cursor = 0
		if got := onOrigins(t, c, top, back).Cursor; got != 0 {
			t.Errorf("%q off the first row landed on row %d, want to stay", back, got)
		}
		bottom := o
		bottom.Cursor = len(o.Origins) - 1
		if got := onOrigins(t, c, bottom, forward).Cursor; got != bottom.Cursor {
			t.Errorf("%q off the last row landed on row %d, want to stay", forward, got)
		}
	}
}

// TestTheAddFormReachesEveryFieldInOrderAndWrapsRound is the field walk, and
// what it is for is a step that **skips** one.
//
// ⚠️ A form driven down its rows and checked only for where it ends up passes a
// cycle that steps by two, because the count wraps: five fields stepped two at a
// time still visits five positions. So every stop is named, in order, and the
// round trip back to the first field is what says the wrap is a wrap rather than
// a clamp. Both directions and every spelling of each, because the three keys
// that move forward are three cases in one switch.
func TestTheAddFormReachesEveryFieldInOrderAndWrapsRound(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	opened := onOrigins(t, c, NewOriginsScreen(c), "a")
	if opened.Field != OriginFieldID {
		t.Fatalf("the form opened on field %d, want the id", opened.Field)
	}
	for _, forward := range []string{"down", "tab", "enter"} {
		o := opened
		for step := 1; step <= OriginFieldCount; step++ {
			o = onOrigins(t, c, o, forward)
			if want := step % OriginFieldCount; o.Field != want {
				t.Fatalf("%d presses of %q left the cursor on field %d, want %d",
					step, forward, o.Field, want)
			}
		}
	}
	for _, back := range []string{"up", "shift+tab"} {
		o := opened
		for step := 1; step <= OriginFieldCount; step++ {
			o = onOrigins(t, c, o, back)
			if want := (OriginFieldCount - step%OriginFieldCount) % OriginFieldCount; o.Field != want {
				t.Fatalf("%d presses of %q left the cursor on field %d, want %d",
					step, back, o.Field, want)
			}
		}
	}
}

// TestTheMediumChooserStepsOneNameAtATimeAndWrapsBothWays is the one field that
// is not a text field: a medium is a value out of a closed list, so typing one is
// only a way to get it wrong.
//
// It is stepped from the middle for the reason the cursor above is, and the wrap
// is asserted in both directions because the two expressions are different — one
// adds the length before taking the remainder and the other does not.
func TestTheMediumChooserStepsOneNameAtATimeAndWrapsBothWays(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	names := cast.MediumNames()
	if len(names) < 3 {
		t.Skipf("there are %d media, too few to step from the middle", len(names))
	}
	form := onOrigins(t, c, NewOriginsScreen(c), "a")
	for form.Field != OriginFieldMedium {
		form = onOrigins(t, c, form, "down")
	}
	middle := form
	middle.MediumIndex = 1
	if got := onOrigins(t, c, middle, "left").MediumIndex; got != 0 {
		t.Errorf("left off medium 1 landed on %d, want 0", got)
	}
	if got := onOrigins(t, c, middle, "right").MediumIndex; got != 2 {
		t.Errorf("right off medium 1 landed on %d, want 2", got)
	}
	first := form
	first.MediumIndex = 0
	if got := onOrigins(t, c, first, "left").MediumIndex; got != len(names)-1 {
		t.Errorf("left off the first medium landed on %d, want the last (%d)", got, len(names)-1)
	}
	last := form
	last.MediumIndex = len(names) - 1
	if got := onOrigins(t, c, last, "right").MediumIndex; got != 0 {
		t.Errorf("right off the last medium landed on %d, want the first", got)
	}
	// Stepping it counts as typing into the form, which is what makes escape ask.
	if !onOrigins(t, c, first, "right").Touched {
		t.Error("stepping the medium left the form reading as untouched")
	}
}

// TestAnUndatedWorkPrintsNoYearRatherThanAZero is the year nobody recorded.
//
// cast.Origin.Year is optional and zero means "not recorded", so three places
// render the absence: this row leaves the cell blank, cmd/hexforge leaves it
// empty, and the golden report writes "undated". All three were exercised by one
// shipped origin — Pokémon, undated until its year was filled in — and by nothing
// else, because every work the books ship carries a year. So an undated one is
// authored here instead.
//
// ⚠️ The assertion is the blank and NOT the alignment, and the first version of
// this test had that backwards. A row is id, medium, year, count and title at
// fixed widths, so it reads as though a missing guard would shift every column
// after it — but Pad gives the cell its five cells whatever is in it, so removing
// the guard entirely left the table perfectly aligned and printed "anime 0"
// beside "0 nhân vật". Both mutations passed. What the guard is for is the word,
// so that is what is measured: the cell a dated row puts its year in has nothing
// in it on an undated one.
//
// ⚠️ **The two works are values rather than a write**, which is the one thing
// that changed when the screen moved. It used to drive the form and press
// ctrl+s; nothing in this package opens a file, and what the row draws is a
// render either way. The write half of it stayed in cmd/hexforge-tui as
// TestAWorkWithNoYearIsWrittenUndated.
func TestAnUndatedWorkPrintsNoYearRatherThanAZero(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	medium, err := cast.ParseMedium(cast.MediumNames()[0])
	if err != nil {
		t.Fatalf("the first medium does not parse: %v", err)
	}
	dated := cast.Origin{ID: "fixture-dated", Title: "Fixture Dated", Medium: medium, Year: 1999}
	blankYear := cast.Origin{ID: "fixture-blank", Title: "Fixture Blank", Medium: medium}
	o := NewOriginsScreen(c)
	o.Origins = []cast.Origin{dated, blankYear}
	o.Counts = map[string]int{dated.ID: 0, blankYear.ID: 0}

	body, _ := o.View(c)
	row := func(id string) string {
		t.Helper()
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, id) {
				return line
			}
		}
		t.Fatalf("the listing has no row for %s:\n%s", id, body)
		return ""
	}
	// Where the year lives, read off the row that has one rather than declared
	// here: the column widths are origins.go's and a second copy of them would be
	// the thing that drifts.
	written := strconv.Itoa(dated.Year)
	at := strings.Index(row(dated.ID), written)
	if at < 0 {
		t.Fatalf("the dated row does not draw its year:\n%s", row(dated.ID))
	}
	blank := row(blankYear.ID)
	if cell := blank[at : at+len(written)]; strings.TrimSpace(cell) != "" {
		t.Errorf("an undated row draws %q where a year goes:\n%s\n%s",
			cell, row(dated.ID), blank)
	}
}
