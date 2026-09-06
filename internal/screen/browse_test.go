package screen

import (
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// What the cast browser draws. Its client half — the menu entry that opens it,
// where esc lands, and that p and ? really reach the two describers — stays in
// cmd/hexforge-tui, because a Back and a Raise are requests this package has no
// way to carry out.

// TestBrowsingResolvesAtTheChosenLevel is the reason the browser is more than a
// listing: a character is a curve, and the arrow keys walk it.
func TestBrowsingResolvesAtTheChosenLevel(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	b := NewBrowseScreen(lib)

	body, footer := b.View(c)
	if !strings.Contains(footer, "←/→") {
		t.Errorf("the footer does not offer the level keys: %q", footer)
	}
	if b.Level != progression.LevelCap {
		t.Errorf("the browser opens at level %d, want the cap", b.Level)
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
	b, _ = b.Update(c, press(t, "left"))
	if b.Level != progression.LevelCap-1 {
		t.Errorf("one step left went to level %d", b.Level)
	}
	body, _ = b.View(c)
	lower, _, err := character.Resolve(progression.LevelCap-1, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve one level down: %v", err)
	}
	if !strings.Contains(body, lower.String()) {
		t.Errorf("the detail pane did not follow the level:\n%s", body)
	}
	for range progression.LevelCap + 5 {
		b, _ = b.Update(c, press(t, "left"))
	}
	if b.Level != 1 {
		t.Errorf("walking off the bottom left the level at %d, want 1", b.Level)
	}

	// The origin filter narrows the list rather than hiding the fact that it
	// did: the count on screen names the filter.
	b, _ = b.Update(c, press(t, "f"))
	body, _ = b.View(c)
	if !strings.Contains(body, b.FilterName(c)) {
		t.Errorf("the filter in force is not named on screen:\n%s", body)
	}
	if b.FilterID() == "" {
		t.Fatal("pressing f did not narrow the list to a work")
	}
	for _, shown := range b.Rows() {
		if shown.Origin != b.FilterID() {
			t.Errorf("%s is shown under the %q filter", shown.ID, b.FilterID())
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
	c, lib := start(t, i18n.Vi)
	b := NewBrowseScreen(lib)

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
	for b.Rows()[b.Cursor].ID != subject.ID {
		before := b.Cursor
		b, _ = b.Update(c, press(t, "down"))
		if b.Cursor == before {
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
		b.Level = test.level
		body := b.Detail(c, subject)
		if !strings.Contains(body, test.want) {
			t.Errorf("at level %d the pane does not show %s:\n%s", test.level, test.want, body)
		}
		if other := subject.Image; test.want != other && strings.Contains(body, other) {
			t.Errorf("at level %d the pane still shows %s, which belongs to another form:\n%s",
				test.level, other, body)
		}
	}
}

// TestTheTraitRowFollowsTheLevelAndOnlyMarksAGateStillAhead is the feature as a
// reader meets it.
//
// A gate is printed while it is still in the future and not once it has been
// passed, so the mark reads as "not yet" rather than as a fact about the trait —
// and the row therefore *changes* as the level is walked, which is the one thing
// a level slider exists to show. Printing every gate at every level would say
// the same thing at level one and at the cap.
//
// Over the **shipped** books rather than the bench, because what it needs is a
// character somebody really gated a trait on.
func TestTheTraitRowFollowsTheLevelAndOnlyMarksAGateStillAhead(t *testing.T) {
	c, lib := startOverTheShippedBooks(t, i18n.Vi)
	c.Height = 44
	b := NewBrowseScreen(lib)

	// The character and the gate, taken from the data: one whose traits all come
	// in at level one has nothing to show here.
	var subject cast.Character
	gate, trait := 0, ""
	for _, candidate := range lib.Characters().All() {
		for _, entry := range candidate.Passives {
			if entry.AtLevel > gate {
				subject, gate, trait = candidate, entry.AtLevel, entry.ID
			}
		}
	}
	if gate <= 1 {
		t.Skip("no shipped character gates a trait above level one, so there is nothing to show")
	}

	b.Level = gate - 1
	locked := b.Detail(c, subject)
	if !strings.Contains(locked, trait+"@"+strconv.Itoa(gate)) {
		t.Errorf("one level below the gate the pane does not mark it:\n%s", locked)
	}

	b.Level = gate
	unlocked := b.Detail(c, subject)
	if strings.Contains(unlocked, trait+"@") {
		t.Errorf("at the gate the pane still marks it as ahead:\n%s", unlocked)
	}
	if !strings.Contains(unlocked, trait) {
		t.Errorf("at the gate the pane does not name the trait at all:\n%s", unlocked)
	}
	if locked == unlocked {
		t.Error("the pane drew the same thing either side of the gate")
	}

	// The names under the row are the traits actually in force. Glossing one the
	// unit does not have yet would read as one it has, which is the opposite of
	// what the mark above it says.
	held, err := lib.Passives().Lookup(trait)
	if err != nil {
		t.Fatalf("look up %s: %v", trait, err)
	}
	if held.Name == "" {
		t.Skip("the shipped trait has no authored name, so there is no gloss to place")
	}
	if strings.Contains(locked, held.Name) {
		t.Errorf("a trait the unit has not unlocked was glossed by name:\n%s", locked)
	}
	if !strings.Contains(unlocked, held.Name) {
		t.Errorf("a trait in force was not glossed by name:\n%s", unlocked)
	}
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
	const drawable = MinWidth - 1
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		width := c.DetailLabelWidth()
		// A detail row is two spaces of indent, the label column, and a space.
		indent := 2 + width + 1

		kits := make(map[string][]string)
		for _, character := range lib.Characters().All() {
			kits[character.ID] = cast.LearnedIDs(character.Skills)
		}
		for _, preset := range lib.Archetypes().All() {
			kits[preset.ID] = preset.Skills
		}
		for id, skills := range kits {
			glossed := lang.GlossedKit(lib.KitSkills(skills))
			if glossed == "" {
				continue
			}
			// Measured as it is *drawn*, not as it is built. A kit has no fixed
			// size, so the raw reading grows without bound -- six Vietnamese
			// names come to 84 cells -- and the row clips it. Holding the raw
			// string to the width would make every sixth skill an obstacle to
			// authoring rather than a layout bug.
			if drawn := indent + lipgloss.Width(Clip(glossed, drawable-indent)); drawn > drawable {
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

// TestTheDetailReserveIsTheTallestPaneInTheBook holds browseDetailRows against
// the book it was measured from, in both directions.
//
// Over rather than under is the defect the reserve exists to stop: a pane taller
// than its reserve is a pane the frame cuts from the bottom, and what goes first
// on a forking line is the form row -- the one row that says which arm the stats
// under it belong to. Under rather than over is slack: a reserve nobody reaches
// is rows the listing gave up for nothing, so the constant is held to the
// tallest exactly and a book that shrinks reddens this as loudly as one that
// grows.
//
// Measured at MinWidth, which is where the pane is tallest: every wrapped row in
// it -- the kit, its names, the bio, the stat line -- wraps hardest in the
// narrowest window this program draws in.
//
// ⚠️ **Every stage boundary, not just the cap.** The pane names the traits and
// the skills in force at the level being walked, so its height moves with the
// level: the tallest shipped pane is at level 16, nowhere near either end.
func TestTheDetailReserveIsTheTallestPaneInTheBook(t *testing.T) {
	tallest, who, atLevel, inLang := 0, "", 0, i18n.Vi
	for _, lang := range i18n.Langs() {
		c, lib := start(t, lang)
		c = atTheFloor(c)
		// The height plays no part in the pane's own height -- nothing in Detail
		// reads it -- so the floor here is about the width alone.
		b := NewBrowseScreen(lib)
		for _, character := range b.Rows() {
			for _, level := range levelsWorthWalking(character) {
				b.Level = level
				drawn := len(drawnLines(strings.TrimRight(b.Detail(c, character), "\n")))
				if drawn > browseDetailRows {
					t.Errorf("%s at level %d in %s draws a %d-row detail pane against a "+
						"reserve of %d, so the frame cuts the bottom of it",
						character.ID, level, lang, drawn, browseDetailRows)
				}
				if drawn > tallest {
					tallest, who, atLevel, inLang = drawn, character.ID, level, lang
				}
			}
		}
	}
	if tallest != browseDetailRows {
		t.Errorf("the tallest pane in the book is %s's at level %d in %s, %d rows against a "+
			"reserve of %d: the listing is giving up %d rows nothing draws",
			who, atLevel, inLang, tallest, browseDetailRows, browseDetailRows-tallest)
	}
}

// levelsWorthWalking is every level a character's pane can change at: the two
// ends, and each side of every stage boundary it declares.
//
// The whole range would be sixty renders a character a language, which is what
// the reserve exists to keep off the draw path in the first place -- so the test
// walks the levels the data says are interesting rather than all of them.
func levelsWorthWalking(character cast.Character) []int {
	levels := []int{1, progression.LevelCap}
	for _, stage := range character.Stages {
		if stage.MinLevel <= 1 {
			continue
		}
		levels = append(levels, stage.MinLevel-1, stage.MinLevel)
	}
	return levels
}

// TestTheCastListingLeavesTheDetailPaneItsRows is the property the reserve
// buys, and it is asserted on a book long enough to overrun the window.
//
// Before it, the listing drew every character it had and the pane took what was
// left, so the pane ran out at the bottom of a window that had held it the day
// before -- the frame said so in a line about the window being too short, and
// the row it had dropped was the form row.
func TestTheCastListingLeavesTheDetailPaneItsRows(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	b := NewBrowseScreen(lib)
	if len(b.Rows()) <= browseRoom(c) {
		t.Fatalf("the book holds %d characters against a listing of %d rows, so nothing "+
			"here is measured: this test needs a cast the window cannot hold",
			len(b.Rows()), browseRoom(c))
	}
	body, _ := b.View(c)
	if drawn := len(drawnLines(body)); drawn > bodyRoom(c) {
		t.Errorf("the browser draws %d lines into the %d the frame gives it, so the frame "+
			"cuts the bottom of the detail pane", drawn, bodyRoom(c))
	}
	// The last row of the pane, by name: a body that fits could still be one
	// that stopped early.
	if last := c.Lang.BudgetPierced(c.Lib.Budget(resolvedAtCursor(t, c, b))); last != "" &&
		!strings.Contains(body, last) {
		t.Errorf("the pane's last row is missing from a body that fits, so it is being " +
			"cut somewhere this test cannot see")
	}
}

func resolvedAtCursor(t *testing.T, c Context, b BrowseScreen) progression.Values {
	t.Helper()
	rows := b.Rows()
	character := rows[Clamp(b.Cursor, 0, len(rows)-1)]
	values, _, err := character.Resolve(b.Level, ChosenForm(character, b.Level, b.Form))
	if err != nil {
		t.Fatalf("resolve %s at level %d: %v", character.ID, b.Level, err)
	}
	return values
}

// TestTheCastListingWindowsAroundTheCursor is the other half: a listing that
// keeps its rows is worth nothing if the row under the cursor is off it.
func TestTheCastListingWindowsAroundTheCursor(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	b := NewBrowseScreen(lib)
	rows := b.Rows()
	if len(rows) <= browseRoom(c) {
		t.Fatalf("the book fits the listing whole, so no window is exercised")
	}
	for cursor := range rows {
		b.Cursor = cursor
		body, _ := b.View(c)
		if !strings.Contains(body, rows[cursor].ID) {
			t.Fatalf("the row under the cursor (%s, %d of %d) is not on the screen",
				rows[cursor].ID, cursor+1, len(rows))
		}
	}
}
