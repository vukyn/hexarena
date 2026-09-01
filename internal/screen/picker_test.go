package screen

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The picker's own two claims: what it draws, and what a keystroke does to the
// answer it is holding.
//
// ⚠️ **Where you land is not in here, and that is the split rather than a gap.**
// A finished pick hands a client a destination and a set of ids, and which field
// of which of *its* screens that fills is that client's claim — asserted in
// cmd/hexforge-tui/picked_test.go, driven through the real model. What is here
// is the half a client cannot see going wrong: the column, the cursor, the
// filter and the cap.
//
// Every state below is **built here** rather than raised through a screen, for
// the reason the golden's hand-built states are: the three screens that raise a
// picker have not moved, and a package that reached for them would be measuring
// a client. What that costs is that a raise carrying the wrong options is
// invisible from here — which is why the raise sites keep their own tests over
// there.

// The picker's detail column, and the one thing it has to be able to do: go
// away.
//
// A trait's name is authored in passives.json in Vietnamese only, so
// Lang.PassiveName answers nothing in English by design — an English reader gets
// the id, which in English is the name. The picker drew that as a padded id and
// a blank cell after it, and on the cursor row the selection bar was painted
// across the whole empty width. The trait listing already drops its gloss column
// outright there, which is the house rule a table follows, and these hold the
// picker to it.
//
// Every assertion below is on the text View actually draws, not on a field: a
// column is a thing on a screen, and a helper answering "is there a column"
// could agree with itself while the rows disagreed with both.
const (
	// A row's marker and state cells as the picker writes them, for a row
	// carrying neither a position in the answer nor a refusal: two cells of
	// marker, then the five `%2s %s ` leaves blank.
	plainRowCells = "       "
	// The same row with the cursor on it. The selection is a style — escape
	// codes — and every test here runs under NO_COLOR, so what is asserted is
	// the *cell* the bar covers rather than the bar.
	selectedRowCells = ">      "
)

// showingEveryRow is the window in hand made tall enough to draw the whole list,
// so a claim about "the row for x" is a claim about a row that is on screen.
//
// The scroll is not what any of these measure and a windowed list would make
// each of them depend on where the cursor happened to be. Room is the only thing
// the height reaches here.
func showingEveryRow(c Context, p *PickState) Context {
	c.Height = len(p.Options) + 4 + 12
	return c
}

// theCarrier is a shipped character as forge.CheckSkill reads one, which is what
// makes a kit picker's rows carry refusals at all.
//
// Every fact is answered, because an unanswered one restricts nothing — a
// carrier built empty gives a list where every row is available, which is the
// one shape these tests must not be handed.
func theCarrier(character cast.Character) forge.Carrier {
	return forge.Carrier{
		ID: character.ID, Archetype: character.Archetype,
		Affinity: character.Element, HasAffinity: true,
		Species: character.Species, Origin: character.Origin,
	}
}

// aTraitHolder is whichever character in the book learns the most traits at the
// cap, and false when none learns two.
//
// Looked up rather than named, which is the rule this package's fixtures keep:
// the fixture cast is data, and a test resting on which entry happens to hold a
// trait measures nothing the day that entry changes.
func aTraitHolder(lib *forge.Library) (cast.Character, bool) {
	found, most := cast.Character{}, 0
	for _, character := range lib.Characters().All() {
		if held := len(character.PassivesAt(
			progression.LevelCap, progression.Furthest)); held > most {
			found, most = character, held
		}
	}
	return found, most >= 2
}

// aDeepLearner is whichever character learns more skills than a placement may
// bring, which is the only sort a slot cap can be measured on: a picker with
// four rows and four slots can say nothing about a fifth.
func aDeepLearner(lib *forge.Library) (cast.Character, bool) {
	found, most := cast.Character{}, 0
	for _, character := range lib.Characters().All() {
		if known := len(character.SkillsAt(
			progression.LevelCap, progression.Furthest)); known > most {
			found, most = character, known
		}
	}
	return found, most > cast.SkillSlots
}

// aTraitPicker is the squad builder's trait picker as that screen raises one: a
// member's learned traits, one slot, the first already spoken for.
func aTraitPicker(t *testing.T, lang i18n.Lang) (Context, *PickState) {
	t.Helper()
	c, lib := start(t, lang)
	holder, holds := aTraitHolder(lib)
	if !holds {
		t.Skip("nothing in the fixture book learns two traits")
	}
	held := holder.PassivesAt(progression.LevelCap, progression.Furthest)
	picker := (&PickState{
		Title: i18n.SquadPickPassives, Hint: i18n.SquadTraitHint,
		Kind: PickPassives, Slots: cast.TraitSlots,
		Options: IDOptions(held), Chosen: append([]string(nil), held[:cast.TraitSlots]...),
	}).Raise()
	if len(picker.Visible()) < 2 {
		t.Fatalf("the trait picker lists %d rows, too few to have an unchosen one",
			len(picker.Visible()))
	}
	return showingEveryRow(c, picker), picker
}

// aKitPicker is the character form's kit picker: the whole skill book, with
// forge.CheckSkill's answer on every row.
func aKitPicker(t *testing.T, lang i18n.Lang) (Context, *PickState) {
	t.Helper()
	c, lib := start(t, lang)
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Fatal("the fixture book holds no characters to build a carrier from")
	}
	picker := (&PickState{
		Title: i18n.PickerKitTitle, Kind: PickSkills,
		Options: KitOptions(lib, theCarrier(characters[0])),
	}).Raise()
	if len(picker.Visible()) < 2 {
		t.Fatal("the kit picker lists fewer than two rows")
	}
	return showingEveryRow(c, picker), picker
}

// anAllowlist is a restriction's character list, which is a picker with no slots
// at all: an author names as many characters as the skill is kept for. It is
// also the one picker with a filter.
func anAllowlist(t *testing.T, lang i18n.Lang) (Context, *PickState) {
	t.Helper()
	c, lib := start(t, lang)
	picker := (&PickState{
		Title: i18n.PickerCharactersTitle, Hint: i18n.PickerAllowlistHint,
		Footer: i18n.PickerFilterFooter, Kind: PickCharacters,
		Options: CharacterOptions(lib), Groups: lib.OriginIDs(),
	}).Raise()
	if picker.Slots != 0 {
		t.Fatalf("the allowlist picker carries %d slot(s), and an allowlist has none",
			picker.Slots)
	}
	return showingEveryRow(c, picker), picker
}

// aSpeciesPicker is the species allowlist, whose detail cell is the one that
// comes out of a data file rather than a compiled gloss.
func aSpeciesPicker(t *testing.T, lang i18n.Lang) (Context, *PickState) {
	t.Helper()
	c, lib := start(t, lang)
	kinds := lib.Species().IDs()
	if len(kinds) == 0 {
		t.Skip("the fixture book declares no species")
	}
	picker := (&PickState{
		Title: i18n.PickerSpeciesTitle, Hint: i18n.PickerAllowlistHint,
		Kind: PickSpecies, Options: IDOptions(kinds),
	}).Raise()
	return showingEveryRow(c, picker), picker
}

// plainRowID is the first row of the picker in hand carrying neither a position
// in the answer nor the cursor, which is the row whose state cell is blank — so
// what precedes its id on the line is exactly plainRowCells.
func plainRowID(t *testing.T, p *PickState) string {
	t.Helper()
	for index, option := range p.Visible() {
		if index == p.Cursor || slices.Contains(p.Chosen, option.ID) {
			continue
		}
		if option.Refusal != nil {
			continue
		}
		return option.ID
	}
	t.Fatal("every row of the picker is chosen, refused or under the cursor")
	return ""
}

// lineStartingWith is the one drawn line beginning with a prefix, and a failure
// when there is none. It is how a claim about the *rest* of a row is made
// without restating how the front of the row is built.
func lineStartingWith(t *testing.T, drawn, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(drawn, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("no line of the screen starts with %q:\n%s", prefix, drawn)
	return ""
}

// pointAt walks the cursor onto a row with the key an author would press, which
// is what keeps the walk honest: writing Cursor directly would measure the draw
// and skip everything the arrow does on the way, filter included.
func pointAt(t *testing.T, p *PickState, id string) *PickState {
	t.Helper()
	for range len(p.Options) + 1 {
		rows := p.Visible()
		if len(rows) > 0 && rows[Clamp(p.Cursor, 0, len(rows)-1)].ID == id {
			return p
		}
		next, result := p.Update(Context{}, press(t, "down"))
		if next == nil {
			t.Fatalf("walking to %q closed the picker (answered: %v)", id, result.Answered)
		}
		p = next
	}
	t.Fatalf("the picker never reached %q", id)
	return p
}

// toggled is space on the row under the cursor, through Update rather than
// through Toggle: what the cap is about is a keystroke, and the arm that reaches
// the cap is one a direct call skips.
func toggled(t *testing.T, p *PickState) *PickState {
	t.Helper()
	next, result := p.Update(Context{}, press(t, "space"))
	if next == nil {
		t.Fatalf("space closed the picker (answered: %v)", result.Answered)
	}
	return next
}

// An English trait row is the bare id and nothing else: no padding out to the
// widest id, and no separator after it.
func TestTheTraitPickerDrawsNoDetailColumnInEnglish(t *testing.T) {
	c, picker := aTraitPicker(t, i18n.En)
	id := plainRowID(t, picker)
	drawn, _ := picker.View(c)

	want := plainRowCells + id
	if !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the English trait picker does not draw %q as its own whole line:\n%s",
			want, drawn)
	}
	// And the shape it used to draw is gone. The exact line above already says
	// so; this names the defect, so a failure reads as "the column came back"
	// rather than as a line that moved.
	if padded := Pad(id, picker.IDColumn()); strings.Contains(drawn, padded) {
		t.Errorf("the English trait picker still pads %q out to a column of %d:\n%s",
			id, picker.IDColumn(), drawn)
	}
}

// The same picker in Vietnamese keeps the column, so the collapse is
// language-blind in effect without being a language check in code.
func TestTheTraitPickerKeepsItsDetailColumnInVietnamese(t *testing.T) {
	c, picker := aTraitPicker(t, i18n.Vi)
	id := plainRowID(t, picker)
	held, err := c.Lib.Passives().Lookup(id)
	if err != nil {
		t.Fatalf("look %s up in the trait book: %v", id, err)
	}
	name := c.Lang.PassiveName(held)
	if name == "" {
		t.Fatalf("the trait %s carries no Vietnamese name, so this row has no column to keep", id)
	}

	want := plainRowCells + Pad(id, picker.IDColumn()) + " " + name
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the Vietnamese trait picker does not draw %q as its own whole line:\n%s",
			want, drawn)
	}
}

// A picker whose rows all carry a detail keeps its column in both languages. The
// kit's is forge.WhoMaySummary, which answers in both — an unrestricted skill
// says "anybody" rather than nothing — so this is the case the collapse must not
// reach.
func TestTheKitPickerKeepsItsDetailColumnInBothLanguages(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, picker := aKitPicker(t, lang)
		id := plainRowID(t, picker)
		// The whole row is not asserted, because a summary long enough to be
		// clipped would make the expected text a second copy of Clip. What the
		// column *is* — the id padded, then the separator, then something — is
		// the claim, and it is the claim the collapse would break.
		prefix := plainRowCells + Pad(id, picker.IDColumn()) + " "
		drawn, _ := picker.View(c)
		line := lineStartingWith(t, drawn, prefix)
		if len(line) <= len(prefix) {
			t.Errorf("the kit picker's row for %s in %s carries an empty detail cell: %q",
				id, lang, line)
		}
	}
}

// The kit picker's rows carry a refusal each, which is what the mark and the
// sentence under the list are drawn from — and what plainRowID skips past.
//
// ⚠️ A carrier answering nothing restricts nothing, so a fixture built with an
// empty one hands every test above a list where every row is available: the
// mark, the dimmed id and the refusal row are then drawn by nothing, and every
// assertion still passes. This is the fixture saying it reached the state it
// exists for.
func TestTheKitPickerRefusesSomeRowsAndOffersOthers(t *testing.T) {
	c, picker := aKitPicker(t, i18n.Vi)
	refused, offered := 0, 0
	for _, option := range picker.Options {
		if option.Refusal != nil {
			refused++
			continue
		}
		offered++
	}
	if refused == 0 || offered == 0 {
		t.Fatalf("the kit picker refuses %d of %d rows, so it measures a list with "+
			"only one sort of row on it", refused, len(picker.Options))
	}
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "!") {
		t.Errorf("no row of the kit picker carries the unavailable mark:\n%s", drawn)
	}
}

// The collapse is all or nothing. A list where some rows have a detail and some
// do not keeps its column on every row, empty cells included — a column dropped
// per row is a ragged table rather than no table.
func TestAPickerWithSomeDetailsKeepsTheColumnOnEveryRow(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	const (
		named = "fire"
		bare  = "khong-co-ten"
	)
	if c.Lang.Gloss(named) == "" {
		t.Fatalf("%s has no Vietnamese gloss, so this list has no detail at all", named)
	}
	if c.Lang.Gloss(bare) != "" {
		t.Fatalf("%s glosses to %q, so this list has no empty row", bare, c.Lang.Gloss(bare))
	}
	picker := (&PickState{
		Title: i18n.PickerElementsTitle, Hint: i18n.PickerAllowlistHint,
		Kind:    PickElements,
		Options: []PickOption{{ID: named}, {ID: bare}},
	}).Raise()

	// The empty row keeps its padding and its separator, so the two rows' detail
	// cells start in the same column.
	want := plainRowCells + Pad(bare, picker.IDColumn()) + " "
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the row for %s lost the column the row above it keeps; wanted %q:\n%s",
			bare, want, drawn)
	}
}

// A selected row with no column after it: the bar covers the bare id, which is
// what the species and trait listings already draw when they drop theirs.
// Padding is what gives a following cell a column to start at, and with no cell
// after it there is nothing left for the width to be for.
func TestTheSelectedRowOfAColumnlessPickerIsTheBareID(t *testing.T) {
	c, picker := aTraitPicker(t, i18n.En)
	id := plainRowID(t, picker)
	picker = pointAt(t, picker, id)

	want := selectedRowCells + id
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "\n"+want+"\n") {
		t.Errorf("the selected row of the English trait picker is not %q:\n%s", want, drawn)
	}
}

// speciesRowFront is the marker and state cells of one row of a species picker,
// which is plainRowCells for every row but the one under the cursor.
func speciesRowFront(p *PickState, index int) string {
	if index == p.Cursor {
		return selectedRowCells
	}
	return plainRowCells
}

// The species detail cell goes through Lang.SpeciesName, so an English row is
// the id and nothing else. Reading kind.Name raw drew "dragon  rồng" on an
// English screen — a leak rather than a translation.
func TestTheSpeciesPickerDrawsNoVietnameseNameInEnglish(t *testing.T) {
	c, picker := aSpeciesPicker(t, i18n.En)
	drawn, _ := picker.View(c)
	for index, option := range picker.Visible() {
		kind, known := c.Lib.Species().Get(option.ID)
		if !known {
			t.Fatalf("the picker offers %s, which the species book does not know", option.ID)
		}
		want := speciesRowFront(picker, index) + option.ID
		if !strings.Contains(drawn, "\n"+want+"\n") {
			t.Errorf("the English species picker does not draw %q as its own whole line:\n%s",
				want, drawn)
		}
		if name := kind.Name; name != "" && strings.Contains(drawn, name) {
			t.Errorf("the English species picker leaks %s's authored name %q:\n%s",
				option.ID, name, drawn)
		}
	}
}

// The same book in Vietnamese keeps the column, filled from the declaration
// rather than from a compiled gloss.
func TestTheSpeciesPickerKeepsTheAuthoredNameInVietnamese(t *testing.T) {
	c, picker := aSpeciesPicker(t, i18n.Vi)
	drawn, _ := picker.View(c)
	for index, option := range picker.Visible() {
		kind, known := c.Lib.Species().Get(option.ID)
		if !known {
			t.Fatalf("the picker offers %s, which the species book does not know", option.ID)
		}
		name := c.Lang.SpeciesName(kind)
		if name == "" {
			t.Fatalf("the species %s carries no authored name, so this row has no column to keep",
				option.ID)
		}
		want := speciesRowFront(picker, index) + Pad(option.ID, picker.IDColumn()) + " " + name
		if !strings.Contains(drawn, "\n"+want+"\n") {
			t.Errorf("the Vietnamese species picker does not draw %q as its own whole line:\n%s",
				want, drawn)
		}
	}
}

// With nothing in any English cell the whole column goes, which is the behaviour
// #164 built and this screen reaches for free — asserted rather than assumed,
// since a list where every detail is empty is exactly the case a per-row collapse
// would also pass.
func TestTheEnglishSpeciesPickerDropsItsDetailColumn(t *testing.T) {
	c, picker := aSpeciesPicker(t, i18n.En)
	rows := picker.Visible()
	if column := picker.detailColumn(c, rows); column != 0 {
		t.Errorf("the English species picker keeps a detail column of %d", column)
	}
	drawn, _ := picker.View(c)
	for index, option := range rows {
		// No padding: the id is not widened to the column the ids would share.
		padded := speciesRowFront(picker, index) + Pad(option.ID, picker.IDColumn())
		if picker.IDColumn() > len(option.ID) && strings.Contains(drawn, padded) {
			t.Errorf("the English species picker still pads %q out to a column of %d:\n%s",
				option.ID, picker.IDColumn(), drawn)
		}
		// And no separator: the line stops at the id.
		front := speciesRowFront(picker, index)
		if strings.Contains(drawn, "\n"+front+option.ID+" ") {
			t.Errorf("the English species picker draws a separator after %q:\n%s",
				option.ID, drawn)
		}
	}
	// The Vietnamese picker over the same book keeps its column, so the collapse
	// is a reading of the rows rather than a check on the language.
	vi, viPicker := aSpeciesPicker(t, i18n.Vi)
	if column := viPicker.detailColumn(vi, viPicker.Visible()); column == 0 {
		t.Error("the Vietnamese species picker dropped its detail column too")
	}
}

// The loadout limit as the picker enforces it.
//
// It used to be asked only after enter, by cast.ChooseFrom through the squad
// builder's own refusal, so an author picked six skills, learned the answer was
// wrong from a red line under the form, and had to reopen the list to give two
// back. The cap is in Toggle now — and the two checks it sits on top of are
// still there, because a loadout hand-edited into squads.json never came through
// a keystroke at all.

// aFullSquadKit is the squad builder's kit picker over a member whose slots are
// already spoken for, which is the state the cap is about.
func aFullSquadKit(t *testing.T, lang i18n.Lang) (Context, *PickState) {
	t.Helper()
	c, lib := start(t, lang)
	deep, learns := aDeepLearner(lib)
	if !learns {
		t.Skip("no character in the book learns more skills than a placement may bring")
	}
	known := deep.SkillsAt(progression.LevelCap, progression.Furthest)
	picker := (&PickState{
		Title: i18n.SquadPickSkills, Hint: i18n.SquadKitHint,
		Kind: PickSkills, Slots: cast.SkillSlots,
		Options: IDOptions(known), Chosen: append([]string(nil), known[:cast.SkillSlots]...),
	}).Raise()
	if len(picker.Chosen) != cast.SkillSlots {
		t.Fatalf("the picker opened with %d chosen, want the %d slots full",
			len(picker.Chosen), cast.SkillSlots)
	}
	if len(picker.Options) <= cast.SkillSlots {
		t.Fatalf("the picker lists %d rows, so there is no row past the slots",
			len(picker.Options))
	}
	return showingEveryRow(c, picker), picker
}

// spare is the first row of the picker in hand that is neither chosen nor
// refused, which is the row a full list has to turn away.
func spare(t *testing.T, p *PickState) string {
	t.Helper()
	for _, option := range p.Visible() {
		if option.Refusal == nil && !slices.Contains(p.Chosen, option.ID) {
			return option.ID
		}
	}
	t.Fatal("every row is either chosen or refused, so nothing is left to refuse for slots")
	return ""
}

// TestAFullKitRefusesAFifthSkill is the cap itself: with the slots spoken for,
// space on a row that has nothing else wrong with it does nothing.
func TestAFullKitRefusesAFifthSkill(t *testing.T) {
	_, picker := aFullSquadKit(t, i18n.Vi)
	fifth := spare(t, picker)
	picker = toggled(t, pointAt(t, picker, fifth))
	if len(picker.Chosen) != cast.SkillSlots {
		t.Errorf("a full kit took a fifth skill: %d chosen, want %d",
			len(picker.Chosen), cast.SkillSlots)
	}
	if slices.Contains(picker.Chosen, fifth) {
		t.Errorf("the answer holds %q, which arrived past the slots", fifth)
	}
}

// TestAFullKitStillTakesARowBackOut is the branch order, which is the whole
// reason taking one out is the first thing Toggle does.
//
// A cap written ahead of it would freeze an over-full loadout solid, and an
// over-full loadout — hand-edited into squads.json — is exactly the thing an
// author opens this picker to fix. Asserted as a round trip rather than as one
// removal, because the removal alone would pass on a picker that had simply
// stopped taking rows at all.
func TestAFullKitStillTakesARowBackOut(t *testing.T) {
	_, picker := aFullSquadKit(t, i18n.Vi)
	held := picker.Chosen[0]
	picker = toggled(t, pointAt(t, picker, held))
	if len(picker.Chosen) != cast.SkillSlots-1 {
		t.Fatalf("taking a row out of a full kit left %d chosen, want %d",
			len(picker.Chosen), cast.SkillSlots-1)
	}
	if slices.Contains(picker.Chosen, held) {
		t.Errorf("the answer still holds %q after it was taken out", held)
	}
	fifth := spare(t, picker)
	picker = toggled(t, pointAt(t, picker, fifth))
	if !slices.Contains(picker.Chosen, fifth) {
		t.Errorf("the room just made would not take %q", fifth)
	}
	if len(picker.Chosen) != cast.SkillSlots {
		t.Errorf("the kit holds %d, want the slots full again at %d",
			len(picker.Chosen), cast.SkillSlots)
	}
}

// TestTheTraitPickerHoldsOneTraitAtATime is the same claim at the value most
// likely to be special-cased wrong.
//
// One slot is where a cap is tempting to write as a swap — space on a second row
// replacing the first — and a swap would make space two verbs at once, worded
// one way on this picker and another on the kit picker beside it.
func TestTheTraitPickerHoldsOneTraitAtATime(t *testing.T) {
	_, picker := aTraitPicker(t, i18n.Vi)
	if len(picker.Options) <= cast.TraitSlots {
		t.Skip("the character learns no more traits than it may hold")
	}
	if len(picker.Chosen) != cast.TraitSlots {
		t.Fatalf("the picker opened with %d chosen, want the %d slot(s) full",
			len(picker.Chosen), cast.TraitSlots)
	}
	held := picker.Chosen[0]
	second := spare(t, picker)
	picker = toggled(t, pointAt(t, picker, second))
	if len(picker.Chosen) != cast.TraitSlots {
		t.Errorf("the trait list holds %d, want %d", len(picker.Chosen), cast.TraitSlots)
	}
	if slices.Contains(picker.Chosen, second) {
		t.Errorf("%q arrived past the trait slot", second)
	}
	if !slices.Contains(picker.Chosen, held) {
		t.Errorf("%q was swapped out by the row that should have been refused", held)
	}
}

// TestAnUncappedPickerTakesMoreThanASquadLoadout is what stops the cap leaking
// into the pickers that must stay uncapped.
//
// Every picker but the squad builder's two is one of these — the five
// restriction allowlists, the statuses a skill inflicts, and the character
// form's own kit — and each takes as many rows as an author names.
func TestAnUncappedPickerTakesMoreThanASquadLoadout(t *testing.T) {
	_, picker := anAllowlist(t, i18n.Vi)
	rows := picker.Visible()
	if len(rows) <= cast.SkillSlots {
		t.Skipf("the allowlist lists %d rows, which cannot pass a loadout of %d",
			len(rows), cast.SkillSlots)
	}
	wanted := make([]string, 0, cast.SkillSlots+1)
	for _, option := range rows[:cast.SkillSlots+1] {
		wanted = append(wanted, option.ID)
		picker = toggled(t, pointAt(t, picker, option.ID))
	}
	if !slices.Equal(picker.Chosen, wanted) {
		t.Errorf("an uncapped picker kept %v of the %v it was given", picker.Chosen, wanted)
	}
}

// TestTheSlotCounterReplacesTheListPosition is the counter, read off the render
// rather than off the field: a picker holding the right number and drawing the
// old wording is exactly the build this is for.
//
// The two are asserted against each other because a list position says nothing
// about what binds — four of nineteen rows is not four of four slots — and both
// languages are rendered because the counter is wording.
func TestTheSlotCounterReplacesTheListPosition(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, capped := aFullSquadKit(t, lang)
		content, _ := capped.View(c)
		slotted := c.Text(i18n.ChoiceSlots, len(capped.Chosen), cast.SkillSlots)
		if !strings.Contains(content, slotted) {
			t.Errorf("the %s kit picker draws no slot counter %q", lang, slotted)
		}
		walked := c.Text(i18n.ChoicePosition, len(capped.Chosen), len(capped.Options))
		if strings.Contains(content, walked) {
			t.Errorf("the %s kit picker still draws the list position %q", lang, walked)
		}

		// And the other way on a picker with no slots, so the counter that was
		// there before is the counter those still get. Every count the slot
		// wording could have been drawn with is refused, because which number a
		// leaked cap would print is exactly what the test cannot know.
		open, list := anAllowlist(t, lang)
		content, _ = list.View(open)
		walked = open.Text(i18n.ChoicePosition, len(list.Chosen), len(list.Options))
		if !strings.Contains(content, walked) {
			t.Errorf("the %s allowlist draws no list position %q", lang, walked)
		}
		for count := range len(list.Options) + 1 {
			slotted = open.Text(i18n.ChoiceSlots, len(list.Chosen), count)
			if strings.Contains(content, slotted) {
				t.Errorf("the %s allowlist draws a slot counter %q, and it has no slots",
					lang, slotted)
			}
		}
	}
}

// TestAnOverFullLoadoutOpensAndCanBeFixed is the state the cap cannot produce
// and the counter is still drawn for: a loadout past its slots, hand-edited into
// squads.json, opened here to be brought back inside them.
//
// The refusing style it is drawn in cannot be asserted — every test here runs
// under NO_COLOR, which is the palette's own rule that meaning never lives in
// colour — so what is measured is the reading and the way out.
func TestAnOverFullLoadoutOpensAndCanBeFixed(t *testing.T) {
	c, picker := aFullSquadKit(t, i18n.Vi)
	picker.Chosen = append(picker.Chosen, spare(t, picker))
	over := len(picker.Chosen)
	if drawn, _ := picker.View(c); !strings.Contains(drawn,
		c.Text(i18n.ChoiceSlots, over, cast.SkillSlots)) {
		t.Errorf("an over-full kit draws no counter reading %d of %d:\n%s",
			over, cast.SkillSlots, drawn)
	}
	picker = toggled(t, pointAt(t, picker, picker.Chosen[0]))
	if len(picker.Chosen) != over-1 {
		t.Errorf("an over-full kit would not give a row back: %d chosen, want %d",
			len(picker.Chosen), over-1)
	}
}

// The filter, which is the one thing only the character allowlist has: f steps
// through the works and back to everything, and nothing chosen is lost on the
// way — the summary under the list is the whole answer and is drawn from Chosen
// rather than from the rows on screen.
func TestTheFilterNarrowsTheListAndKeepsTheAnswer(t *testing.T) {
	c, picker := anAllowlist(t, i18n.Vi)
	if len(picker.Groups) < 2 {
		t.Skipf("the fixture book has %d works, too few to cycle a filter through",
			len(picker.Groups))
	}
	whole := len(picker.Visible())
	first := picker.Visible()[0].ID
	picker = toggled(t, pointAt(t, picker, first))

	seen := map[string]int{}
	for step := range len(picker.Groups) {
		next, _ := picker.Update(c, press(t, "f"))
		picker = next
		group := picker.Group()
		if group == "" {
			t.Fatalf("f landed back on everything after %d of %d works",
				len(seen), len(picker.Groups))
		}
		// ⚠️ **Which group, not merely some group.** Filter counts from one so
		// that nought can be the entry that hides nothing, and every reading of
		// it — the rows, the line above them, the name in that line — goes
		// through Group(), so a filter off by one narrows the list *and* labels
		// it, consistently and wrongly. Measured: an off-by-one there passed
		// every other assertion in this function, including the row-membership
		// check two lines down, and reddened the two goldens alone.
		if want := picker.Groups[step]; group != want {
			t.Fatalf("press %d of f lands on %q, want the %q the book declares there",
				step+1, group, want)
		}
		for _, row := range picker.Visible() {
			if row.Group != group {
				t.Fatalf("the %q filter shows %q, which is out of %q",
					group, row.ID, row.Group)
			}
		}
		seen[group] = len(picker.Visible())
		if !slices.Contains(picker.Chosen, first) {
			t.Fatalf("filtering to %q lost %q from the answer: %v", group, first, picker.Chosen)
		}
	}
	// One more press comes back to everything, which is the entry the filter is
	// numbered from one to leave room for.
	next, _ := picker.Update(c, press(t, "f"))
	picker = next
	if group := picker.Group(); group != "" {
		t.Errorf("the cycle did not come back to everything: it is on %q", group)
	}
	if got := len(picker.Visible()); got != whole {
		t.Errorf("the unfiltered list is %d rows, want the %d it opened with", got, whole)
	}
	narrowed := 0
	for _, count := range seen {
		narrowed += count
	}
	if narrowed != whole {
		t.Errorf("the works hold %d rows between them against the %d in the whole list",
			narrowed, whole)
	}
}
