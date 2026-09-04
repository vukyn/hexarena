package screen

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The skill listing and the form over it: what each draws, how the cursor and
// the typed filter behave, and what the six lists do with an answer.
//
// The client's half stays in cmd/hexforge-tui — which menu entry reaches this
// screen, where esc lands, where the `?` raise arrives and whether a pick
// reaches the field it names. A keystroke is not a render and a render is not a
// landing; #218 measured the picker's cap and found the whole client suite green
// while two package tests went red, so neither side stands in for the other.

// The two queries the fixture is driven with, hardcoded because they are design
// facts about the data rather than layout: the shipped book calls its five
// dragon skills "long nộ", "long vũ", "long trảo", "cuồng nộ long" and "long
// xung", and **no id in either book holds "long"** — so the query exercises the
// *name* half of the match, and its first hit is the seventeenth skill declared
// rather than the first. `dragon` is the same five rows less one, reached
// through the *id* half, which is what tells the two halves apart.
//
// Nothing anywhere holds a z, in either language: a Vietnamese word cannot and
// no shipped or fixture id does.
//
// ⚠️ They are declared here **and** in cmd/hexforge-tui, copied rather than
// shared, for the reason firstWords is: a fixture is not code two packages may
// drift over, and the client's everyScreen registers three filter states built
// from the same queries.
const (
	someSkillQuery   = "long"
	someIDSkillQuery = "dragon"
	noSkillQuery     = "zzzz"
)

// typeInto presses every character of a query, one keystroke at a time, which is
// the only way a query is ever built.
func typeInto(t *testing.T, c Context, s SkillsScreen, typed string) SkillsScreen {
	t.Helper()
	for _, letter := range typed {
		s, _, _ = s.Update(c, press(t, string(letter)))
	}
	return s
}

// pressOn is one keystroke, keeping only the screen — the action is asserted
// where a test is about one.
func pressOn(t *testing.T, c Context, s SkillsScreen, name string) SkillsScreen {
	t.Helper()
	next, _, _ := s.Update(c, press(t, name))
	return next
}

// openTheFilter enters the listing and presses the key that opens the field.
func openTheFilter(t *testing.T, c Context) SkillsScreen {
	t.Helper()
	opened := pressOn(t, c, NewSkillsScreen(c), "/")
	if !opened.Filtering {
		t.Fatal("/ did not open the filter on the skill listing")
	}
	return opened
}

// filterTo opens the filter, types a query and hands the keyboard back to the
// rows with enter — which is the state every key that reads a row is asked about
// below.
func filterTo(t *testing.T, c Context, query string) SkillsScreen {
	t.Helper()
	filtered := pressOn(t, c, typeInto(t, c, openTheFilter(t, c), query), "enter")
	if filtered.Filtering {
		t.Fatal("enter did not hand the keyboard back to the listing")
	}
	if filtered.Query != query {
		t.Fatalf("the filter holds %q after typing %q", filtered.Query, query)
	}
	return filtered
}

// TestTheSkillFormsFieldsAreSharedBetweenCopies pins the one place "a screen is a
// value" is not true here.
//
// ⚠️ **Inputs is a slice, so two copies of this screen write through one backing
// array.** That was measured in #216 rather than reasoned about: dropping the
// client's write-back of the screen after a status pick left the inflicts entry
// filled in anyway — the landing had written through storage the copies share —
// while every flag beside it was lost, and every pre-existing status test stayed
// green.
//
// The behaviour is **relied on today** and is moved unchanged, so this test is
// what stops it changing by accident in either direction. It asserts three
// things, and the third is what makes the first two mean something:
//
//   - a write through one copy's Inputs is visible through another's, and the two
//     name the same address, so the sharing is real rather than a coincidence of
//     one value;
//   - a scalar beside it — Field — is **not** shared, which is what says the
//     screen is otherwise a value and that the first claim is about this field
//     rather than about copying;
//   - ResetForm hands back a screen with **fresh** storage, because that is the
//     one operation that must not write through to whatever copy the caller was
//     holding: a discarded draft that reached back into the model would be the
//     discard failing to discard.
//
// So it goes red if the sharing ever stops **or** starts, and whoever changes it
// later has to mean it.
func TestTheSkillFormsFieldsAreSharedBetweenCopies(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	original := NewSkillsScreen(c)
	copied := original

	const typed = "poison:250"
	copied.Inputs[SkillFieldInflicts].SetValue(typed)
	if got := original.Inputs[SkillFieldInflicts].Value(); got != typed {
		t.Errorf("a write through a copy left the original's inflicts field %q, want %q — "+
			"SkillsScreen.Inputs has stopped being shared between copies, which is a "+
			"behaviour change and not a tidy-up", got, typed)
	}
	if &original.Inputs[0] != &copied.Inputs[0] {
		t.Error("two copies of the screen no longer name the same backing array")
	}

	// And the field beside it is not shared, which is what makes the claim above
	// about this slice rather than about the screen.
	copied.Field = SkillFieldPower
	if original.Field == SkillFieldPower {
		t.Error("Field is shared between copies, so the screen has stopped being a value " +
			"in a way nothing here asked for")
	}

	// ResetForm is the one operation that must build fresh storage: it is what a
	// discarded draft goes through, and a discard that wrote through to the copy
	// the client is still holding would not have discarded anything.
	fresh := original.ResetForm(c)
	fresh.Inputs[SkillFieldInflicts].SetValue("burn:100")
	if got := original.Inputs[SkillFieldInflicts].Value(); got != typed {
		t.Errorf("ResetForm handed back storage the old screen still shares: writing "+
			"through the new one left the old field %q", got)
	}
}

// TestSkillRowDropsTheGlossColumnWhenItIsEmpty pins the rule itself, so it
// cannot be lost when the caller that measures the width changes.
func TestSkillRowDropsTheGlossColumnWhenItIsEmpty(t *testing.T) {
	with := skillRow(8, 6, 8, "strike", "đòn", "neutral", "1000x1", "anyone")
	without := skillRow(8, 0, 8, "strike", "đòn", "neutral", "1000x1", "anyone")
	if strings.Contains(without, "đòn") {
		t.Errorf("a zero gloss column still drew the gloss: %q", without)
	}
	if !strings.Contains(with, "đòn") {
		t.Errorf("a sized gloss column dropped the gloss: %q", with)
	}
	if lipgloss.Width(without) >= lipgloss.Width(with) {
		t.Errorf("dropping the column did not narrow the row: %d vs %d",
			lipgloss.Width(without), lipgloss.Width(with))
	}
}

// TestTheSkillFilterNarrowsByIDAndByName is the feature: a query finds a row by
// either of the two things a row is called, ignoring case and ignoring
// diacritics.
//
// The diacritic half is the reason the whole thing exists rather than a
// refinement of it. An author at a terminal with no Vietnamese input method
// cannot type "diệp", so a filter that only matched the letters as authored
// would leave the name column — the column this client added a whole feature to
// draw — unsearchable, and the filter would be an id filter with a misleading
// name.
func TestTheSkillFilterNarrowsByIDAndByName(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	all := len(NewSkillsScreen(c).Skills)

	for _, test := range []struct {
		typed string
		want  []string
		why   string
	}{
		// The name half, and the case that proves it is the name half: not one
		// of these five ids holds the query.
		{someSkillQuery, []string{
			"dragon_rage", "dragon_dance", "dragon_claw", "outrage", "dragon_drive",
		}, "the Vietnamese name"},
		// The id half, over four of the same five rows.
		{someIDSkillQuery, []string{
			"dragon_rage", "dragon_dance", "dragon_claw", "dragon_drive",
		}, "the id"},
		// The diacritics, dropped: "phi diệp" typed on an ASCII keyboard.
		{"diep", []string{"razor_leaf"}, "a name with its marks left off"},
		// đ is not a d with a mark on it, so this is the entry an NFD-and-strip
		// implementation would have needed by hand anyway.
		{"doc", []string{
			"poison_powder", "sludge_bomb", "venoshock", "toxic", "venom_fang",
		}, "đ folded to d"},
		{noSkillQuery, nil, "a query nothing answers to"},
	} {
		t.Run(test.typed, func(t *testing.T) {
			filtered := filterTo(t, c, test.typed)
			var got []string
			for _, row := range filtered.Rows() {
				got = append(got, row.ID)
			}
			if strings.Join(got, " ") != strings.Join(test.want, " ") {
				t.Errorf("%q finds %v through %s, want %v",
					test.typed, got, test.why, test.want)
			}
			if len(got) >= all {
				t.Errorf("%q left all %d skills on screen, so it narrowed nothing",
					test.typed, all)
			}
		})
	}

	// And nothing typed hides nothing, which is the value the two categorical
	// filters in this client keep at nought for the same reason.
	if opened := openTheFilter(t, c); len(opened.Rows()) != all {
		t.Errorf("an empty filter shows %d of %d skills", len(opened.Rows()), all)
	}
}

// TestEveryKeyThatReadsARowReadsTheFilteredOne is the defect this feature would
// otherwise ship, asserted from every key that can commit it.
//
// The cursor indexes the visible rows. `e` prefills the form, `?` raises the
// description and the damage row under the listing reads a preview — all three
// used to index s.Skills with that cursor, and with nothing typed the two lists
// are identical, so the whole existing suite would have gone on passing while an
// author edited the wrong skill.
//
// ⚠️ The fixture is what makes this discriminating, so it is asserted rather
// than assumed: the query has to leave a first row that is *not* the first skill
// in the book, or reading either list gives the same answer and the test proves
// nothing.
//
// ⚠️ **Where the `?` raise LANDS is not asserted here and cannot be.** This
// package can see the Subject an Action carries; whether the description screen
// is reached with it is the client's claim, and #205 measured that an off-by-one
// on a raise leaves this whole package green. See
// TestTheBlurbScreenDescribesTheSkillUnderTheCursor in cmd/hexforge-tui.
func TestEveryKeyThatReadsARowReadsTheFilteredOne(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	filtered := filterTo(t, c, someSkillQuery)
	rows := filtered.Rows()
	if len(rows) < 2 {
		t.Fatalf("%q leaves %d rows, so the cursor has nowhere to be wrong",
			someSkillQuery, len(rows))
	}
	want := rows[0]
	if unfiltered := filtered.Skills[0]; want.ID == unfiltered.ID {
		t.Fatalf("%q's first match is also the first skill in the book (%s), so "+
			"reading either list gives the same answer", someSkillQuery, want.ID)
	}
	if filtered.Cursor != 0 {
		t.Fatalf("the cursor sits at %d rather than on the first match", filtered.Cursor)
	}

	// e opens the form on the row under the marker.
	edited := pressOn(t, c, filtered, "e")
	if edited.Editing != want.ID {
		t.Errorf("e opened the form on %q, want the row under the cursor, %q",
			edited.Editing, want.ID)
	}
	// The query survives the trip: an author who narrowed the listing to find a
	// skill is looking at the same question when the form closes.
	if edited.Query != someSkillQuery {
		t.Errorf("opening the form dropped the filter, leaving %q", edited.Query)
	}

	// ? asks for the description of the row under the marker, and the subject it
	// carries is what says which row that is.
	_, action, _ := filtered.Update(c, press(t, "?"))
	if action.Kind != Raise || action.Target != Blurb {
		t.Fatalf("? asked for %v/%v rather than raising the description",
			action.Kind, action.Target)
	}
	if action.Subject.ID != want.ID {
		t.Errorf("? asked about %q, want the row under the cursor, %q",
			action.Subject.ID, want.ID)
	}
	// And the position it carries counts the narrowed list rather than the book,
	// so "1 / 5" does not read as "1 / 66".
	if action.Subject.At != 1 || action.Subject.Of != len(rows) {
		t.Errorf("? placed the skill at %d of %d, want 1 of %d",
			action.Subject.At, action.Subject.Of, len(rows))
	}

	// The listing itself draws the narrowed rows and nothing else.
	listing, _ := filtered.View(c)
	if dropped := filtered.Skills[0].ID; strings.Contains(listing, dropped) {
		t.Errorf("the filtered listing still draws %q:\n%s", dropped, listing)
	}
	for _, row := range rows {
		if !strings.Contains(listing, row.ID) {
			t.Errorf("the filtered listing does not draw %q:\n%s", row.ID, listing)
		}
	}
}

// TestEscapeClearsTheSkillFilterAndEnterKeepsIt covers the two ways out, which
// are two different answers rather than one with a shortcut.
//
// Escape is one key that undoes the whole thing, so there is never a listing
// narrowed by a query the author has just dismissed. Enter keeps the query and
// hands the keyboard back, which is the state the feature is for: the letters
// are commands again on the rows the query found.
//
// ⚠️ It asserts that escape here asks for **nothing** — a Stay — which is the
// half that says the filter closed rather than the screen. Where an escape on
// the *unfiltered* listing goes is the client's, since a Back names no screen
// this package has: TestEscapeLeavesTheSkillListingByGoingBack.
func TestEscapeClearsTheSkillFilterAndEnterKeepsIt(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	all := len(NewSkillsScreen(c).Skills)

	typed := typeInto(t, c, openTheFilter(t, c), someSkillQuery)
	cleared, action, _ := typed.Update(c, press(t, "esc"))
	if cleared.Query != "" || cleared.Filtering {
		t.Errorf("escape left the filter as %q (open: %v)", cleared.Query, cleared.Filtering)
	}
	if got := len(cleared.Rows()); got != all {
		t.Errorf("escape left %d of %d skills on screen", got, all)
	}
	if action.Kind != Stay {
		t.Errorf("escape on an open filter asked for %v, want nothing — it closed the "+
			"field, not the screen", action.Kind)
	}

	// Enter keeps it, and reopening the field picks the same query back up rather
	// than starting again — or the two keys would disagree about what enter did.
	kept := filterTo(t, c, someSkillQuery)
	if reopened := pressOn(t, c, kept, "/"); reopened.Query != someSkillQuery {
		t.Errorf("reopening the filter holds %q, want %q", reopened.Query, someSkillQuery)
	}
	// Backspace edits rather than clearing.
	shorter := pressOn(t, c, pressOn(t, c, kept, "/"), "backspace")
	if want := someSkillQuery[:len(someSkillQuery)-1]; shorter.Query != want {
		t.Errorf("backspace left %q, want %q", shorter.Query, want)
	}
}

// TestALetterIsTextWhileTheSkillFilterHasTheKeyboard is why the filter is a mode
// at all: every letter this screen has is already a command, so a field sharing
// the keyboard with them could take no query.
//
// q is the one worth spelling out — it asks for a quit from this screen — but a,
// e and ? are the same mistake with quieter consequences, and k and j are why
// the arrows are the arrows here rather than the vim pair.
func TestALetterIsTextWhileTheSkillFilterHasTheKeyboard(t *testing.T) {
	c, _ := start(t, i18n.Vi)

	// The contrast first, or the assertion under it proves nothing: q really does
	// ask to quit from this screen, which is why a field that shared the keyboard
	// with it could not take a query at all.
	if _, action, _ := NewSkillsScreen(c).Update(c, press(t, "q")); action.Kind != Quit {
		t.Fatal("q does not ask to quit from the unfiltered listing, so swallowing it " +
			"below measures nothing")
	}
	typing, action, _ := openTheFilter(t, c).Update(c, press(t, "q"))
	if action.Kind == Quit {
		t.Error("q asked to quit while the filter had the keyboard")
	}
	if typing.Query != "q" {
		t.Errorf("q left the query as %q", typing.Query)
	}

	// The rest of the screen's letters, straight into the field.
	typed := typeInto(t, c, openTheFilter(t, c), "aek j?")
	if typed.Query != "aek j?" {
		t.Errorf("the query reads %q, want every letter and the space", typed.Query)
	}
	if typed.Adding || typed.Editing != "" {
		t.Error("a letter typed into the filter opened the form")
	}

	// The arrows still walk the rows, because nothing else can — both of them,
	// since a filter that could only be walked one way would be a filter an
	// author has to reopen to correct an overshoot.
	walking := pressOn(t, c, pressOn(t, c, openTheFilter(t, c), "down"), "down")
	if walking.Cursor != 2 {
		t.Errorf("the down arrow left the cursor at %d", walking.Cursor)
	}
	if back := pressOn(t, c, walking, "up"); back.Cursor != 1 {
		t.Errorf("the up arrow left the cursor at %d", back.Cursor)
	}
	if !walking.Filtering {
		t.Error("the down arrow closed the filter")
	}
	if walking.Query != "" {
		t.Errorf("the arrows typed %q into the query", walking.Query)
	}
	// And j does not, which is the price of the mode and the thing to notice if
	// somebody "fixes" it.
	if vim := typeInto(t, c, openTheFilter(t, c), "j"); vim.Cursor != 0 {
		t.Errorf("j moved the cursor to %d instead of being typed", vim.Cursor)
	}
}

// TestAFilterThatFindsNothingRefusesTheKeysThatReadARow is the second half of
// the cursor defect: a query matching nothing arrives one keystroke at a time,
// so it is an ordinary state rather than an edge case, and the keys that index a
// row have to decline instead of reaching into an empty slice.
//
// `a` is deliberately not among them, and that is the point of listing it:
// adding indexes nothing, and a filter that has found nothing is exactly the
// moment an author wants to write the skill they were looking for.
func TestAFilterThatFindsNothingRefusesTheKeysThatReadARow(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	empty := filterTo(t, c, noSkillQuery)
	if len(empty.Rows()) != 0 {
		t.Fatalf("%q left %d rows", noSkillQuery, len(empty.Rows()))
	}
	if empty.Cursor != 0 {
		t.Errorf("the cursor sits at %d over an empty listing", empty.Cursor)
	}
	if _, held := empty.Selected(); held {
		t.Error("an empty listing reports a skill under its cursor")
	}

	if edited := pressOn(t, c, empty, "e"); edited.Editing != "" {
		t.Errorf("e opened the form on %q over an empty listing", edited.Editing)
	}
	if _, action, _ := empty.Update(c, press(t, "?")); action.Kind != Stay {
		t.Errorf("? asked for %v over an empty listing", action.Kind)
	}
	for _, name := range []string{"down", "up"} {
		if walked := pressOn(t, c, empty, name); walked.Cursor != 0 {
			t.Errorf("%s moved the cursor to %d over an empty listing", name, walked.Cursor)
		}
	}
	// The one key that still works, because the answer to "nothing matches" is
	// often to write it.
	if added := pressOn(t, c, empty, "a"); !added.Adding {
		t.Error("a would not open the new-skill form over an empty listing")
	}

	// ⚠️ And the clamp on the way *down* to nothing, which is the half a refusal
	// cannot see. Walk to the last of several matches, then type one more letter
	// and narrow to one: Selected clamps on the way out, so e and ? would still
	// act on a real skill — but the listing draws its marker by comparing the
	// cursor with the row's index, so a cursor left past the end draws a listing
	// with no marker at all while the damage row underneath describes a row
	// nothing is pointing at.
	narrowing := typeInto(t, c, openTheFilter(t, c), someSkillQuery)
	for range len(narrowing.Rows()) {
		narrowing = pressOn(t, c, narrowing, "down")
	}
	if narrowing.Cursor != len(narrowing.Rows())-1 {
		t.Fatalf("the cursor stopped at %d of %d matches, so it is not at the end "+
			"and the narrowing below measures nothing",
			narrowing.Cursor, len(narrowing.Rows()))
	}
	// " n" rather than a bare letter, so the space a Vietnamese name needs is
	// typed here as well: "long n" is "long nộ" and nothing else in the book.
	deeper := typeInto(t, c, narrowing, " n")
	if got := len(deeper.Rows()); got != 1 {
		t.Fatalf("%q leaves %d rows, want the one that makes the clamp measurable",
			deeper.Query, got)
	}
	if got, rows := deeper.Cursor, len(deeper.Rows()); got > max(rows-1, 0) {
		t.Errorf("narrowing to %d rows left the cursor at %d", rows, got)
	}
	body, _ := deeper.View(c)
	if marked, held := deeper.Selected(); held && !strings.Contains(body, "> "+marked.ID) {
		t.Errorf("the narrowed listing draws no marker on %q:\n%s", marked.ID, body)
	}

	// And the screen says so, in both languages, rather than drawing an empty box
	// — and it says the *right* thing, because a listing can be empty for two
	// reasons and only one of them is a keystroke to take back. The empty book is
	// unreachable through the shipped data, so it is reached by hand: a branch
	// nothing renders is a branch with no width test and no translation test.
	for _, lang := range i18n.Langs() {
		at, _ := start(t, lang)
		nothing := filterTo(t, at, noSkillQuery)
		body, _ := nothing.View(at)
		if want := lang.Text(i18n.SkillsFilterNothing); !strings.Contains(body, want) {
			t.Errorf("the %s listing does not say %q when nothing matches:\n%s",
				lang, want, body)
		}
		bookless := nothing
		bookless.Skills = nil
		body, _ = bookless.View(at)
		if want := lang.Text(i18n.NoneCatalogued); !strings.Contains(body, want) {
			t.Errorf("the %s listing over an empty book does not say %q:\n%s",
				lang, want, body)
		}
		if unwanted := lang.Text(i18n.SkillsFilterNothing); strings.Contains(body, unwanted) {
			t.Errorf("the %s listing blames the filter for an empty book:\n%s", lang, body)
		}
	}
}

// TestTheSkillFilterRowSaysWhatIsTypedAndWhatIsLeft covers the row itself, which
// is the only thing on screen that can say why the listing is short.
func TestTheSkillFilterRowSaysWhatIsTypedAndWhatIsLeft(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		// Nothing typed yet: the row says what to type, because a bare label
		// reads as a broken row.
		opened := openTheFilter(t, c)
		body, footer := opened.View(c)
		if want := lang.Text(i18n.SkillsFilterPrompt); !strings.Contains(body, want) {
			t.Errorf("the %s filter says nothing about what to type (%q):\n%s", lang, want, body)
		}
		// The footer is what says the field has the keyboard, since the row
		// cannot (colour is never information here, and there is no caret to
		// draw).
		if want := lang.Text(i18n.SkillsFilterFooter); footer != want {
			t.Errorf("the %s footer while filtering is %q, want %q", lang, footer, want)
		}

		filtered := filterTo(t, c, someSkillQuery)
		body, footer = filtered.View(c)
		want := lang.Say(i18n.SkillsFiltering, someSkillQuery,
			len(filtered.Rows()), len(filtered.Skills))
		if !strings.Contains(body, want) {
			t.Errorf("the %s filter row does not read %q:\n%s", lang, want, body)
		}
		// The keyboard is back on the rows, so the footer is the listing's again
		// — and it still names the key that reopens the field.
		if want := lang.Text(i18n.SkillsFooter); footer != want {
			t.Errorf("the %s footer after enter is %q, want %q", lang, footer, want)
		}
		// And the row is gone entirely when there is no filter, rather than drawn
		// empty: a row that says nothing is a row the listing has paid for.
		body, _ = NewSkillsScreen(c).View(c)
		if unwanted := lang.Text(i18n.SkillsFilterPrompt); strings.Contains(body, unwanted) {
			t.Errorf("the unfiltered %s listing still draws the filter row:\n%s", lang, body)
		}
	}
}

// TestTheSkillFootersNameTheFilterKeyAndFit holds the half a width sweep cannot:
// that the key the feature hangs on is still named after the next trim.
//
// The footer had to be trimmed to fit the new key and no key was given up, which
// is the same squeeze BrowseFooter and the three battle footers already took —
// the words after ↑/↓, esc and q are dropped, those being the keys whose meaning
// the screen itself shows. Measured rather than counted: a hand-count of one
// candidate came back six cells over.
func TestTheSkillFootersNameTheFilterKeyAndFit(t *testing.T) {
	const drawable = MinWidth - 1
	for _, lang := range i18n.Langs() {
		for _, key := range []i18n.Key{i18n.SkillsFooter, i18n.SkillsFilterFooter} {
			line := lang.Text(key)
			if width := lipgloss.Width(line); width > drawable {
				t.Errorf("the %s footer for key %d is %d cells, over the %d there are: %q",
					lang, key, width, drawable, line)
			} else {
				t.Logf("%s key %d: %d of %d cells", lang, key, width, drawable)
			}
		}
		// The listing has to name the key that opens the filter, and the filter
		// has to name both ways out of it: an unadvertised mode is a mode nobody
		// enters, and a mode nobody can see the exit of is worse.
		if !strings.Contains(lang.Text(i18n.SkillsFooter), "/ ") {
			t.Errorf("the %s listing footer no longer names the filter key: %q",
				lang, lang.Text(i18n.SkillsFooter))
		}
		for _, named := range []string{"enter", "esc"} {
			if !strings.Contains(lang.Text(i18n.SkillsFilterFooter), named) {
				t.Errorf("the %s filter footer does not name %s: %q",
					lang, named, lang.Text(i18n.SkillsFilterFooter))
			}
		}
	}
}

// TestATypedFilterIsBounded is the one thing between the filter row and the
// window: the query is author-typed text with no length of its own, so the field
// takes a bounded number of letters and the row clips what it draws.
func TestATypedFilterIsBounded(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	long := strings.Repeat("dragon", FilterLimit)
	typed := typeInto(t, c, openTheFilter(t, c), long)
	if got := len([]rune(typed.Query)); got != FilterLimit {
		t.Errorf("the field took %d letters of %d typed, want %d",
			got, len(long), FilterLimit)
	}
	// The row itself, at the floor and in the longer language: it is the one line
	// on this screen whose length an author decides.
	for _, lang := range i18n.Langs() {
		at, _ := start(t, lang)
		at = atTheFloor(at)
		row := strings.TrimRight(typed.filterRow(at, len(typed.Rows())), "\n")
		if width := lipgloss.Width(row) + 1; width > MinWidth {
			t.Errorf("a %s filter row of %d letters draws %d cells of %d: %q",
				lang, FilterLimit, width, MinWidth, row)
		} else {
			t.Logf("%s filter row at %d letters: %d of %d cells",
				lang, FilterLimit, lipgloss.Width(row), MinWidth-1)
		}
	}
}

// TestEverySkillsPickDestinationWritesItsOwnField is the six destinations that
// followed this screen, asserted as a set.
//
// ⚠️ **A count proves a destination is handled; it cannot prove it is handled
// right**, which #207, #214, #216 and #218 each measured again — so this walks
// SkillsPickCount *and* reads back which field moved, because five of the six
// differ from each other in nothing but that. Where a pick reaches this screen
// from is the client's claim; see TestEachAllowlistPickLandsInItsOwnField.
//
// ⚠️ The inflicts destination is the one whose answer survives an applier that
// applies nothing — Inputs is shared between copies, see
// TestTheSkillFormsFieldsAreSharedBetweenCopies — so it is asserted through
// Touched as well as through the field.
func TestEverySkillsPickDestinationWritesItsOwnField(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	lists := map[SkillsPick]func(SkillsScreen) []string{
		SkillsPickElements: func(s SkillsScreen) []string { return s.KeptElements },
		SkillsPickRoles:    func(s SkillsScreen) []string { return s.KeptRoles },
		SkillsPickWorlds:   func(s SkillsScreen) []string { return s.KeptWorlds },
		SkillsPickKinds:    func(s SkillsScreen) []string { return s.KeptKinds },
		SkillsPickWho:      func(s SkillsScreen) []string { return s.KeptWho },
	}
	// The walk: every destination but the zero has to be here, and the count is
	// what says so rather than a list somebody remembered to extend.
	if got, want := len(lists)+1, int(SkillsPickCount)-1; got != want {
		t.Fatalf("this table covers %d destinations of the %d declared", got, want)
	}

	const chosen = "fixture-id"
	for destination, read := range lists {
		landed, action := NewSkillsScreen(c).Picked(c, destination,
			PickAnswer{Chosen: []string{chosen}})
		if action.Kind != Stay {
			t.Errorf("destination %d asked for %v; a pick fills in a field and leaves the "+
				"reader on the form", destination, action.Kind)
		}
		if got := read(landed); len(got) != 1 || got[0] != chosen {
			t.Errorf("destination %d left its own field %v", destination, got)
		}
		if !landed.Touched {
			t.Errorf("destination %d left the form clean, so escaping it would throw the "+
				"answer away without asking", destination)
		}
		// And nowhere else. This is what a destination pointed at the wrong field
		// looks like, and nothing else here can see it.
		for other, otherRead := range lists {
			if other == destination {
				continue
			}
			if got := otherRead(landed); len(got) != 0 {
				t.Errorf("destination %d also wrote %v into destination %d's field",
					destination, got, other)
			}
		}
	}

	// The sixth is not a list: it composes into a text field through
	// forge.AddApplications, which is the same call the typed syntax goes
	// through.
	status := StatusOptions(c.Lib)
	if len(status) == 0 {
		t.Fatal("the status book is empty, so the inflicts destination cannot be measured")
	}
	inflicted, action := NewSkillsScreen(c).Picked(c, SkillsPickInflicts,
		PickAnswer{Chosen: []string{status[0].ID}, Typed: "250"})
	if action.Kind != Stay {
		t.Errorf("the inflicts destination asked for %v", action.Kind)
	}
	if got := inflicted.Inputs[SkillFieldInflicts].Value(); !strings.Contains(got, status[0].ID) {
		t.Errorf("choosing %q left the inflicts field %q", status[0].ID, got)
	}
	if !inflicted.Touched {
		t.Error("choosing a status left the form clean")
	}

	// And a destination this screen does not own lands nowhere rather than in
	// whichever field sorted first — the picker carries an `any`, so a value from
	// another client's vocabulary can arrive here.
	untouched, action := NewSkillsScreen(c).Picked(c, "not one of ours",
		PickAnswer{Chosen: []string{chosen}})
	if action.Kind != Stay || untouched.Touched {
		t.Error("a destination this screen does not own still marked the form")
	}
}

// TestTheDiscardQuestionNamesWhatIsBeingThrownAway is the Ask arm, pressed
// rather than constructed.
//
// Two questions reach one Confirmed and they differ only in what they say is
// being lost — a skill nobody has written yet, or a set of changes to one
// already in the book — so which key is carried is the whole of what a client
// can tell them apart by.
//
// ⚠️ A clean form asks nothing, and that is the arm worth the extra half of this
// test: a question raised over a form with nothing in it is a question an author
// has to dismiss on every escape.
func TestTheDiscardQuestionNamesWhatIsBeingThrownAway(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	listing := NewSkillsScreen(c)

	clean := pressOn(t, c, listing, "a")
	if !clean.Adding {
		t.Fatal("a did not open the new-skill form")
	}
	closed, action, _ := clean.Update(c, press(t, "esc"))
	if action.Kind != Stay {
		t.Errorf("escaping an untouched form asked %v rather than closing quietly", action.Kind)
	}
	if closed.Adding {
		t.Error("escaping an untouched form left it open")
	}

	for _, test := range []struct {
		name  string
		open  func() SkillsScreen
		want  i18n.Key
		other i18n.Key
	}{
		{"a new skill", func() SkillsScreen { return pressOn(t, c, listing, "a") },
			i18n.SkillFormDiscard, i18n.SkillFormEditDiscard},
		{"an edited one", func() SkillsScreen { return pressOn(t, c, listing, "e") },
			i18n.SkillFormEditDiscard, i18n.SkillFormDiscard},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Typed into rather than flagged: Touched is what decides whether a
			// question is worth asking, and setting it by hand would measure this
			// test's idea of a dirty form.
			dirty := typeInto(t, c, test.open(), "x")
			if !dirty.Touched {
				t.Fatal("typing into the form left it clean, so the question below is not reached")
			}
			_, action, _ := dirty.Update(c, press(t, "esc"))
			if action.Kind != Ask {
				t.Fatalf("escaping a dirty form asked for %v, want an Ask", action.Kind)
			}
			if action.Question != test.want {
				t.Errorf("the question is key %d, want %d — the two wordings differ in what "+
					"they say is being lost, so the wrong one names the wrong thing",
					action.Question, test.want)
			}
			if action.Question == test.other {
				t.Errorf("the question is the other form's")
			}
			// And a confirmed answer empties the form, which is what both
			// questions meant.
			answered, back := dirty.Confirmed(c, nil)
			if answered.FormInFront() {
				t.Error("confirming the discard left the form in front")
			}
			if answered.Touched {
				t.Error("confirming the discard left the form dirty")
			}
			if back.Kind != Stay {
				t.Errorf("confirming the discard asked for %v; the listing is where the "+
					"reader already is", back.Kind)
			}
		})
	}
}

// TestEveryListFieldOpensItsOwnPicker is the Pick arm, pressed rather than
// constructed, and it is what says the state handed over is the one the field
// asked for.
//
// ⚠️ The two status fields raise the **same** list and its answer lands in the
// inflicts field whichever of them opened it. That is what the field write has
// always done and it is preserved unchanged by the move — see OpenStatuses.
func TestEveryListFieldOpensItsOwnPicker(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	for _, test := range []struct {
		field int
		want  SkillsPick
	}{
		{SkillFieldKeptForElements, SkillsPickElements},
		{SkillFieldKeptForRoles, SkillsPickRoles},
		{SkillFieldKeptForCharacters, SkillsPickWho},
		{SkillFieldKeptForSpecies, SkillsPickKinds},
		{SkillFieldKeptForOrigins, SkillsPickWorlds},
		{SkillFieldInflicts, SkillsPickInflicts},
		{SkillFieldOnItself, SkillsPickInflicts},
	} {
		form := pressOn(t, c, NewSkillsScreen(c), "a")
		form.Field = test.field
		_, action, _ := form.Update(c, press(t, "space"))
		if action.Kind != Pick {
			t.Fatalf("space on field %d asked for %v, want a Pick", test.field, action.Kind)
		}
		if action.Picker == nil {
			t.Fatalf("the Pick on field %d carried no list", test.field)
		}
		if got := action.Picker.Into; got != test.want {
			t.Errorf("the list opened on field %d lands at %v, want %v",
				test.field, got, test.want)
		}
		if len(action.Picker.Options) == 0 {
			t.Errorf("the list opened on field %d offers no rows", test.field)
		}
	}

	// A field with no list behind it opens none, which is what says the branch
	// above is about these fields rather than about space.
	plain := pressOn(t, c, NewSkillsScreen(c), "a")
	plain.Field = SkillFieldPower
	if _, action, _ := plain.Update(c, press(t, "space")); action.Kind == Pick {
		t.Error("space on the power field opened a list")
	}
}

// TestTheAccuracyRowReadsAsAPercentage covers the reason the row exists: 850 is
// what the engine divides by, and nobody reads it as a chance. The parts per
// thousand stay on screen — the number written to the file has to be the number
// shown — with the percentage beside them.
func TestTheAccuracyRowReadsAsAPercentage(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	s := NewSkillsScreen(c)
	s.Adding = true
	s.Inputs[SkillFieldAccuracy].SetValue("850")
	if got := s.FieldValue(c, SkillFieldAccuracy, 16); !strings.Contains(got, "850") ||
		!strings.Contains(got, "85%") {
		t.Errorf("the accuracy row shows %q, want both 850 and 85%%", got)
	}

	// Power reads the same way, and a zero one says nothing a "0%" could add:
	// four shipped skills declare no power at all.
	s.Inputs[SkillFieldPower].SetValue("2200")
	if got := s.FieldValue(c, SkillFieldPower, 16); !strings.Contains(got, "220%") {
		t.Errorf("the power row shows %q, want 220%%", got)
	}
	s.Inputs[SkillFieldPower].SetValue("0")
	if got := s.percentHint(c, SkillFieldPower); got != "" {
		t.Errorf("a zero power produced the hint %q, want none", got)
	}

	// The chances in the inflicts field are parts per thousand too, but the
	// field holds the syntax ParseApplications reads, so the reading goes
	// beside it rather than into it.
	s.Inputs[SkillFieldInflicts].SetValue("weaken:800,blind:400")
	if got := s.FieldValue(c, SkillFieldInflicts, 16); !strings.Contains(got, "80%") ||
		!strings.Contains(got, "40%") {
		t.Errorf("the inflicts row shows %q, want both 80%% and 40%%", got)
	}
	// A list being typed is unparseable most of the time; that is not an error
	// to announce.
	for _, partial := range []string{"weaken", "weaken:", "weaken:8x"} {
		s.Inputs[SkillFieldInflicts].SetValue(partial)
		if got := s.chanceHint(c, 16); got != "" {
			t.Errorf("a value of %q produced the chance hint %q, want none", partial, got)
		}
	}

	// A half-typed number is the normal state of a text field, not an error to
	// announce, so the hint says nothing at all rather than guessing.
	for _, partial := range []string{"", "-", "8x"} {
		s.Inputs[SkillFieldAccuracy].SetValue(partial)
		if got := s.percentHint(c, SkillFieldAccuracy); got != "" {
			t.Errorf("a value of %q produced the hint %q, want none", partial, got)
		}
	}
}
