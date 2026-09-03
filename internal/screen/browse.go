package screen

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// BrowseScreen is the authored cast, filtered by origin, with one character
// resolved beside it.
//
// The level is the reason this screen is worth having over `hexforge cast`: a
// character is a curve rather than a stat line, and walking the level with the
// arrow keys is how the shape of that curve — and where it starts costing the
// budget — becomes visible.
//
// ⚠️ **It is the first moved screen that raises another one about a subject**,
// which is why it could move at all. Its whole coupling to a client was three
// writes of that client's own screen enum — esc to the menu, `p` to the art
// preview, `?` to the description — and Action already has the vocabulary for
// all three: Back, and two Raises naming Preview and Blurb. Nothing here asks
// where any of them lives.
type BrowseScreen struct {
	Characters []cast.Character
	// Origins is the catalogued work ids, and Filter indexes them from one.
	// Zero is the filter that hides nothing, and it is a number rather than a
	// name in the list because its name is a translated word while every other
	// entry is an id that is the same in every language.
	Origins []string
	Filter  int
	Cursor  int
	Level   int
	// Form is the arm of a forking evolution line the reader has asked for, by
	// stage name, and empty is "whatever the level resolves to".
	//
	// ⚠️ **It is read through ChosenForm and never straight**, because the cursor
	// and the level both move under it: a name chosen on one character means
	// nothing on the next, and below the level a fork opens at it means nothing
	// on the same one. Kept rather than settled on every key for the reason it is
	// kept at all — a reader who chose an arm, walked away and walked back should
	// find it still in front.
	Form string
}

// NewBrowseScreen is the cast listing filled from a library, opened at the level
// cap — the far end of every curve, which is where a character is comparable
// with another one.
func NewBrowseScreen(lib *forge.Library) BrowseScreen {
	return BrowseScreen{Level: progression.LevelCap}.Refresh(lib)
}

// Refresh re-reads the library, keeping the level and clamping the cursor —
// entering the screen after a write should show the new character rather than
// a stale list.
func (b BrowseScreen) Refresh(lib *forge.Library) BrowseScreen {
	b.Characters = lib.Characters().All()
	b.Origins = lib.OriginIDs()
	if b.Level == 0 {
		b.Level = progression.LevelCap
	}
	if b.Filter > len(b.Origins) {
		b.Filter = 0
	}
	b.Cursor = Clamp(b.Cursor, 0, len(b.Rows())-1)
	return b
}

// FilterID is the work in force, or empty when every work is shown.
func (b BrowseScreen) FilterID() string {
	if b.Filter <= 0 || b.Filter > len(b.Origins) {
		return ""
	}
	return b.Origins[b.Filter-1]
}

// FilterName is what the filter is called on screen: an id, or the translated
// word for "everything".
func (b BrowseScreen) FilterName(c Context) string {
	if id := b.FilterID(); id != "" {
		return id
	}
	return c.Text(i18n.BrowseAllOrigins)
}

// Rows is the characters the current filter leaves visible.
func (b BrowseScreen) Rows() []cast.Character {
	wanted := b.FilterID()
	if wanted == "" {
		return b.Characters
	}
	out := make([]cast.Character, 0, len(b.Characters))
	for _, character := range b.Characters {
		if character.Origin == wanted {
			out = append(out, character)
		}
	}
	return out
}

// Update reads one keystroke: the cursor and the level walk, the origin filter
// cycles, one of the two describers is raised about the row under the cursor, or
// the reader leaves.
func (b BrowseScreen) Update(_ Context, message tea.KeyPressMsg) (BrowseScreen, Action) {
	switch message.String() {
	case "q":
		return b, Action{Kind: Quit}
	case "esc":
		return b, Action{Kind: Back}
	case "up", "k":
		b.Cursor = Clamp(b.Cursor-1, 0, len(b.Rows())-1)
	case "down", "j":
		b.Cursor = Clamp(b.Cursor+1, 0, len(b.Rows())-1)
	case "left", "h":
		b.Level = Clamp(b.Level-1, 1, progression.LevelCap)
	case "right", "l":
		b.Level = Clamp(b.Level+1, 1, progression.LevelCap)
	case "home":
		b.Level = 1
	case "end":
		b.Level = progression.LevelCap
	case "f":
		b.Filter = (b.Filter + 1) % (len(b.Origins) + 1)
		b.Cursor = Clamp(b.Cursor, 0, len(b.Rows())-1)
	case "s":
		// The arm of a fork, on the character under the cursor. It does nothing
		// on the eleven shipped characters whose lines do not fork, and the
		// footer therefore does not name it: what names it is the row it walks,
		// which is drawn only while there is something to walk. See FormRow.
		return b.CycleForm(), Action{}
	case "p":
		// The preview keeps no cursor and no level of its own, so this screen
		// hands both over: the character under the cursor, at the level being
		// walked. ⚠️ It used to read them back off here instead, which is what
		// made the preview a screen that could not move.
		return b, Action{Kind: Raise, Target: Preview, Subject: b.Subject()}
	case "?":
		// The same arrangement, for the trait sentences. The row above prints a
		// trait's id and its name and stops there, which is the whole of what
		// this tool ever said about a trait — so an author moving virulence from
		// 300 to 200 watched a figure change and never read the line a player
		// gets. The describer is handed the same subject the preview takes, so
		// the traits it names are the ones in force at the level being walked.
		return b, Action{Kind: Raise, Target: Blurb, Subject: b.Subject()}
	}
	return b, Action{}
}

// Subject is what the two describers raised from here are about: the character
// under the cursor, read at the level being walked.
//
// Built here rather than in either describer because this screen is the one that
// knows what its list holds — the filter decides which characters are in it, so
// the position on the row is a fact about this screen and not about the cast.
//
// A filter that has left nothing gives a subject with Of at nought, which is how
// a raiser says there is nothing to describe without the describer reaching back
// to count. The level is carried either way: it is still the level being walked.
func (b BrowseScreen) Subject() Subject {
	rows := b.Rows()
	if len(rows) == 0 {
		return Subject{Kind: CharacterSubject, Level: b.Level}
	}
	at := Clamp(b.Cursor, 0, len(rows)-1)
	return Subject{
		Kind:  CharacterSubject,
		ID:    rows[at].ID,
		Level: b.Level,
		// Settled here rather than in either describer, for the reason the level
		// is carried at all: the two describers and the pane behind them must read
		// one form, and the screen with the cursor is the only one that can say
		// which. A describer settling its own would make walking from the picture
		// to the traits able to change the form.
		Stage: ChosenForm(rows[at], b.Level, b.Form),
		At:    at + 1,
		Of:    len(rows),
	}
}

// CycleForm steps to the next arm of the fork the character under the cursor has
// at the level being walked, and does nothing on a line that does not fork.
//
// Exported because the key that walks it is answered in three places: here, and
// in each client while one of the two describers is in front — the same
// arrangement the level already has, and for the same reason. A describer keeps
// no cursor and no level, so what a key pressed over it moves is the browser
// behind it, and the clients own that because they own which screen is in front.
func (b BrowseScreen) CycleForm() BrowseScreen {
	rows := b.Rows()
	if len(rows) == 0 {
		return b
	}
	b.Form = NextForm(rows[Clamp(b.Cursor, 0, len(rows)-1)], b.Level, b.Form)
	return b
}

// The two fixed columns of the listing. Both hold ids, which are the same in
// every language and as long as the data makes them, so these are constants
// where a column of labels would be measured. The element column after them
// takes no width of its own: it is last, so its gloss spends the row's slack
// rather than another column's.
const (
	browseIDWidth     = 24
	browseOriginWidth = 14
)

// View draws the listing, the character under the cursor resolved at the level
// being walked, and the footer.
func (b BrowseScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.BrowseFooter)
	rows := b.Rows()
	var out strings.Builder
	fmt.Fprintf(&out, "%s  %s\n\n",
		c.Style.Heading.Render(c.Text(i18n.BrowseHeading)),
		c.Style.Dim.Render(c.Text(i18n.BrowseShowing,
			b.FilterName(c), len(rows), len(b.Characters))))
	if len(rows) == 0 {
		out.WriteString("  " + c.Text(i18n.BrowseNothingHere) + "\n\n")
		if len(b.Characters) == 0 {
			out.WriteString("  " + c.Text(i18n.BrowseNothingAuthored) + "\n")
		} else {
			out.WriteString("  " + c.Text(i18n.BrowseNoneFromThisWork) + "\n")
		}
		return out.String(), footer
	}
	for i, character := range rows {
		marker := "  "
		// The element column is glossed here as well as in the detail pane
		// below, and it is the only list column that is: it is the last one on
		// the row, so the bracket eats the row's slack rather than another
		// column's. The worst case any pair of the eleven elements can make was
		// measured against both fixed columns at full width — 72 cells of the
		// 79 there are, on metal/electric. The two fixed columns take no gloss
		// for the same reason: an id past its column would push this one over.
		name := Pad(character.ID, browseIDWidth) + " " +
			Pad(character.Origin, browseOriginWidth) + " " +
			c.Lang.GlossedAffinity(character.Element)
		if i == b.Cursor {
			marker = "> "
			name = c.Style.Selected.Render(name)
		}
		out.WriteString(marker + name + "\n")
	}
	out.WriteString("\n")
	out.WriteString(b.Detail(c, rows[Clamp(b.Cursor, 0, len(rows)-1)]))
	return out.String(), footer
}

// Detail is one character resolved at the chosen level.
//
// Exported because the character it draws is a parameter rather than the row
// under the cursor: the tests that measure a form's art and a trait's gate walk
// the level over one named character, and reaching that through the listing
// would be measuring the cursor instead.
func (b BrowseScreen) Detail(c Context, character cast.Character) string {
	var out strings.Builder
	origin, known := c.Lib.Origins().Get(character.Origin)
	from := character.Origin
	if known {
		from = fmt.Sprintf("%s (%s, %s)", origin.Title, origin.Medium, character.Origin)
	}
	out.WriteString(c.Style.Heading.Render(character.ID+" — "+character.Name) + "\n")
	out.WriteString(c.Label(c.Text(i18n.LabelFrom), "%s", from))
	// The preset and the element read as ids with their names on the row under
	// them, which is the one convention this pane has rather than two.
	//
	// They used to bracket the name beside the id — "blighter (kẻ gieo độc)" —
	// while the kit, the species and the traits put theirs underneath, and the
	// split was invisible to a reader: it followed where the name came from, a
	// compiled table against a data file, which is a fact about this repository
	// and not about the character on screen. The list above still brackets,
	// because a table column has no second row to give.
	//
	// An unglossed id draws no second row rather than an empty one, so an English
	// screen is exactly what it was.
	out.WriteString(c.Label(c.Text(i18n.LabelPlaystyle), "%s", character.Archetype))
	if name := c.Lang.Gloss(character.Archetype); name != "" {
		out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, name))
	}
	out.WriteString(c.Label(c.Text(i18n.LabelElement), "%s", character.Element))
	if names := c.Lang.AffinityNames(character.Element); names != "" {
		out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, names))
	}
	// Wrapped, not clipped: nine ids are longer than any terminal, and half an
	// id is worse than a second row.
	out.WriteString(c.Wrapped(c.Text(i18n.LabelKit), c.DetailLabelWidth(),
		forge.UnlockSummaryAt(character.Skills, b.Level)))
	// The kit's names go under it rather than beside it: five skills glossed
	// inline is five brackets on one row, which does not fit. Nothing is drawn
	// at all when there is nothing to say — in English, or for a kit of skills
	// the table has no names for — rather than an empty row under a full one.
	if glossed := c.Lang.GlossedKit(c.Lib.KitSkills(cast.LearnedIDs(character.Skills))); glossed != "" {
		out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, glossed))
	}
	// Traits sit under the kit because they read as the other half of it: what
	// the character uses, then what it simply has. Drawn only when there are
	// any — a row saying "none" on every character in a cast that holds none is
	// a row that says nothing.
	out.WriteString(c.Label(c.Text(i18n.LabelStages), "%s", c.Lang.StageSummary(character)))
	if character.Bio != "" {
		out.WriteString(c.Wrapped(c.Text(i18n.LabelBiography), c.DetailLabelWidth(), character.Bio))
	}

	atLevel := c.Text(i18n.LabelAtLevel, b.Level)
	// The form, and not progression.Furthest: a line that forks reaches two grown
	// forms at one level and Resolve refuses to choose between them, which used to
	// end this pane at the row below with a refusal in it. ChosenForm is
	// progression.Furthest on every line that does not fork, so this call is the
	// one it always was for eleven of the twelve shipped characters.
	form := ChosenForm(character, b.Level, b.Form)
	values, stage, err := character.Resolve(b.Level, form)
	if err != nil {
		out.WriteString(c.Label(atLevel, "%s", c.Style.Bad.Render(c.Lang.Error(err))))
		return out.String()
	}
	// Which arm is in front, above the three rows it decides — the picture, the
	// traits and the stat line. Nothing at all on a line that does not fork.
	out.WriteString(FormRow(c, character, b.Level, b.Form))
	// The art row keeps its place in the block that says what the character is,
	// but the picture it names is the one the level resolved to. A character
	// whose forms have their own pictures has no single picture, so walking the
	// level with the arrow keys is what shows which one a form uses — the same
	// reason the level is worth walking for the stats. It is drawn after Resolve
	// rather than before it because there is no picture to name until the stage
	// is known, and a character that will not resolve has already returned.
	out.WriteString(c.Label(c.Text(i18n.LabelArt), "%s", artLine(c, character, stage)))
	// What it is, when it is anything: the axis a skill kept for a lineage reads,
	// and the one fact on this screen that the numbers cannot imply. Only when
	// there is one, on the same terms as the traits row under it.
	if len(character.Species) > 0 {
		out.WriteString(c.Wrapped(c.Text(i18n.FieldSpecies), c.DetailLabelWidth(),
			strings.Join(character.Species, " ")))
		if glossed := c.Lang.GlossedSpecies(c.Lib.KitSpecies(character.Species)); glossed != "" {
			out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, glossed))
		}
	}
	// Traits sit with the art rather than with the kit, because both answer "at
	// this level": a gate still ahead is printed and one already passed is not, so
	// walking the level is what shows a trait coming in. The names go under it the
	// way the kit's do, and only the ones actually in force are named — glossing a
	// trait the unit does not have yet would read as a trait it has.
	if len(character.Passives) > 0 {
		out.WriteString(c.Wrapped(c.Text(i18n.LabelTraits), c.DetailLabelWidth(),
			forge.UnlockSummaryAt(character.Passives, b.Level)))
		if glossed := c.Lang.GlossedPassives(
			c.Lib.KitPassives(character.PassivesAt(b.Level, form))); glossed != "" {
			out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, glossed))
		}
	}
	// Six stats and a stage name come to 88 cells at the floor, so this wraps
	// too. It was being clipped by the frame, and what fell off the end was the
	// stage — the one part of the row that says which form the numbers belong to.
	out.WriteString(c.Wrapped(atLevel, c.DetailLabelWidth(),
		fmt.Sprintf("%s   %s", values, c.Text(i18n.StageInWords, stage.Name))))
	budget := c.Lib.Budget(values)
	out.WriteString(c.Label(c.Text(i18n.LabelEffectiveHP), "%s", BudgetLine(c, budget)))
	// The pierced floor goes under the row rather than beside it, dim and with
	// no label of its own, which is the shape the kit's names above already use
	// for a second reading of the row before it. Lang.BudgetPierced records why
	// it cannot be a clause on the line itself.
	out.WriteString(c.WrappedIn("", c.DetailLabelWidth(), c.Style.Dim, c.Lang.BudgetPierced(budget)))
	return out.String()
}

// artLine is the picture the character shows at the resolved stage, and whether
// it is on disk.
//
// A function rather than a method now that it reads a Context: it wants the
// stage it was handed and nothing the screen keeps, and a method taking neither
// its receiver's cursor nor its level would be claiming a relationship it has
// not got.
func artLine(c Context, character cast.Character, stage progression.Stage) string {
	art := character.StageArt(stage)
	if c.Lib.ImageExists(art) {
		return c.Style.Good.Render(art + "  " + c.Text(i18n.ArtPresent))
	}
	return c.Style.Bad.Render(art + "  " + c.Text(i18n.ArtMissing))
}

// BudgetLine is the joint health-and-defence bound drawn as a meter and as
// numbers. The numbers are what makes it readable without colour, and being
// over the bound is said in words in both languages, because it is the state
// rather than the styling.
//
// Exported because the new-character form draws the same row and has not moved:
// a second copy would be a second answer to what "over the bound" looks like.
func BudgetLine(c Context, budget forge.Budget) string {
	meter := Bar(BudgetBarWidth, budget.Effective, budget.Max)
	if budget.Over() {
		return c.Style.Bad.Render(c.Lang.Budget(meter, budget))
	}
	return c.Style.Good.Render(c.Lang.Budget(meter, budget))
}
