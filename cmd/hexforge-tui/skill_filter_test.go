package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The typed filter on the skill listing. Two things are being held down here and
// only one of them is the feature.
//
// The feature is that a query narrows the rows, by id or by the Vietnamese name,
// without a Vietnamese keyboard. The other thing is the defect the feature
// introduces: skillsScreen.cursor counts the rows the filter left, so every read
// of it that reached into the book instead would act on a different skill than
// the one under the marker — and the two agree exactly while nothing is typed,
// which is every other test in this package. So the tests below that matter most
// are the ones that filter to something whose first match is *not* the first
// skill in the book and then press the keys that read a row.

// The two queries the fixture is driven with, hardcoded because they are design
// facts about the data rather than layout: the shipped book calls its five dragon
// skills "long nộ", "long vũ", "long trảo", "cuồng nộ long" and "long xung",
// and **no id in either book holds "long"** — so the query exercises the *name*
// half of the match, and its first hit is the seventeenth skill declared rather
// than the first. `dragon` is the same five rows less one, reached through the
// *id* half, which is what tells the two halves apart.
//
// Nothing anywhere holds a z, in either language: a Vietnamese word cannot and no
// shipped or fixture id does.
const (
	someSkillQuery   = "long"
	someIDSkillQuery = "dragon"
	noSkillQuery     = "zzzz"
)

// openSkillFilter enters the listing and presses the key that opens the filter.
func openSkillFilter(t *testing.T, m model) model {
	t.Helper()
	opened := typeText(t, m.enter(screenSkills), "/")
	if !opened.skills.filtering {
		t.Fatal("/ did not open the filter on the skill listing")
	}
	return opened
}

// filterSkillsTo opens the filter, types a query and hands the keyboard back to
// the rows with enter — which is the state every key that reads a row is asked
// about below.
func filterSkillsTo(t *testing.T, m model, query string) model {
	t.Helper()
	filtered := key(t, typeText(t, openSkillFilter(t, m), query), "enter")
	if filtered.skills.filtering {
		t.Fatal("enter did not hand the keyboard back to the listing")
	}
	if filtered.skills.query != query {
		t.Fatalf("the filter holds %q after typing %q", filtered.skills.query, query)
	}
	return filtered
}

// TestTheSkillFilterNarrowsByIDAndByName is the feature: a query finds a row by
// either of the two things a row is called, ignoring case and ignoring
// diacritics.
//
// The diacritic half is the reason the whole thing exists rather than a
// refinement of it. An author at a terminal with no Vietnamese input method
// cannot type "diệp", so a filter that only matched the letters as authored would
// leave the name column — the column this client added a whole feature to draw —
// unsearchable, and the filter would be an id filter with a misleading name.
func TestTheSkillFilterNarrowsByIDAndByName(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	all := len(m.enter(screenSkills).skills.skills)

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
		{"DIEP", []string{"razor_leaf"}, "the same query shouted"},
		// đ is not a d with a mark on it, so this is the entry an NFD-and-strip
		// implementation would have needed by hand anyway.
		{"doc", []string{
			"poison_powder", "sludge_bomb", "venoshock", "venom_fang",
		}, "đ folded to d"},
		{"ĐỘC", []string{
			"poison_powder", "sludge_bomb", "venoshock", "venom_fang",
		}, "the accented letters in upper case"},
		{noSkillQuery, nil, "a query nothing answers to"},
	} {
		t.Run(test.typed, func(t *testing.T) {
			filtered := filterSkillsTo(t, m, test.typed)
			var got []string
			for _, row := range filtered.skills.rows() {
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
	if opened := openSkillFilter(t, m); len(opened.skills.rows()) != all {
		t.Errorf("an empty filter shows %d of %d skills", len(opened.skills.rows()), all)
	}
}

// TestEveryKeyThatReadsARowReadsTheFilteredOne is the defect this feature would
// otherwise ship, asserted from every key that can commit it.
//
// The cursor indexes the visible rows. `e` prefills the form, `?` raises the
// description and the damage row under the listing reads a preview — all three
// used to index s.skills with that cursor, and with nothing typed the two lists
// are identical, so the whole existing suite would have gone on passing while an
// author edited the wrong skill.
//
// ⚠️ The fixture is what makes this discriminating, so it is asserted rather than
// assumed: the query has to leave a first row that is *not* the first skill in
// the book, or reading either list gives the same answer and the test proves
// nothing.
func TestEveryKeyThatReadsARowReadsTheFilteredOne(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	filtered := filterSkillsTo(t, m, someSkillQuery)
	rows := filtered.skills.rows()
	if len(rows) < 2 {
		t.Fatalf("%q leaves %d rows, so the cursor has nowhere to be wrong",
			someSkillQuery, len(rows))
	}
	want := rows[0]
	if unfiltered := filtered.skills.skills[0]; want.ID == unfiltered.ID {
		t.Fatalf("%q's first match is also the first skill in the book (%s), so "+
			"reading either list gives the same answer", someSkillQuery, want.ID)
	}
	if filtered.skills.cursor != 0 {
		t.Fatalf("the cursor sits at %d rather than on the first match",
			filtered.skills.cursor)
	}

	// e opens the form on the row under the marker.
	edited := typeText(t, filtered, "e")
	if edited.skills.editing != want.ID {
		t.Errorf("e opened the form on %q, want the row under the cursor, %q",
			edited.skills.editing, want.ID)
	}
	// The query survives the trip: an author who narrowed the listing to find a
	// skill is looking at the same question when the form closes.
	if edited.skills.query != someSkillQuery {
		t.Errorf("opening the form dropped the filter, leaving %q", edited.skills.query)
	}

	// ? describes the row under the marker.
	described := typeText(t, filtered, "?")
	if described.screen != screenBlurb {
		t.Fatalf("? left the screen at %v rather than raising the description", described.screen)
	}
	body, _ := described.blurb.View(described.ctx())
	if named := described.lang.GlossedSkill(want); !strings.Contains(body, named) {
		t.Errorf("the description does not name %q:\n%s", named, body)
	}
	if wrong := described.lang.GlossedSkill(filtered.skills.skills[0]); strings.Contains(body, wrong) {
		t.Errorf("the description names %q, which is the book's first skill and not "+
			"the filtered one:\n%s", wrong, body)
	}
	// And the position beside the name counts the narrowed list rather than the
	// book, so "1 / 5" does not read as "1 / 66".
	if position := described.text(i18n.ChoicePosition, 1, len(rows)); !strings.Contains(body, position) {
		t.Errorf("the description does not place the skill in the narrowed list (%q):\n%s",
			position, body)
	}

	// The listing itself draws the narrowed rows and nothing else.
	listing, _ := filtered.skills.view(filtered)
	if dropped := filtered.skills.skills[0].ID; strings.Contains(listing, dropped) {
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
// hands the keyboard back, which is the state the feature is for: the letters are
// commands again on the rows the query found.
func TestEscapeClearsTheSkillFilterAndEnterKeepsIt(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	all := len(m.enter(screenSkills).skills.skills)

	cleared := key(t, typeText(t, openSkillFilter(t, m), someSkillQuery), "esc")
	if cleared.skills.query != "" || cleared.skills.filtering {
		t.Errorf("escape left the filter as %q (open: %v)",
			cleared.skills.query, cleared.skills.filtering)
	}
	if got := len(cleared.skills.rows()); got != all {
		t.Errorf("escape left %d of %d skills on screen", got, all)
	}
	// And it is the filter that escape closed, not the screen: a second escape is
	// what leaves.
	if cleared.screen != screenSkills {
		t.Errorf("escape left the listing for %v while the filter was open", cleared.screen)
	}
	if left := key(t, cleared, "esc"); left.screen != screenMenu {
		t.Errorf("escape on the unfiltered listing went to %v", left.screen)
	}

	// Enter keeps it, and reopening the field picks the same query back up rather
	// than starting again — or the two keys would disagree about what enter did.
	kept := filterSkillsTo(t, m, someSkillQuery)
	if reopened := typeText(t, kept, "/"); reopened.skills.query != someSkillQuery {
		t.Errorf("reopening the filter holds %q, want %q",
			reopened.skills.query, someSkillQuery)
	}
	// Backspace edits rather than clearing.
	shorter := key(t, typeText(t, kept, "/"), "backspace")
	if want := someSkillQuery[:len(someSkillQuery)-1]; shorter.skills.query != want {
		t.Errorf("backspace left %q, want %q", shorter.skills.query, want)
	}
}

// TestALetterIsTextWhileTheSkillFilterHasTheKeyboard is why the filter is a mode
// at all: every letter this screen has is already a command, so a field sharing
// the keyboard with them could take no query.
//
// q is the one worth spelling out — it quits the program from this screen — but
// a, e and ? are the same mistake with quieter consequences, and k and j are why
// the arrows are the arrows here rather than the vim pair.
func TestALetterIsTextWhileTheSkillFilterHasTheKeyboard(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	opened := openSkillFilter(t, m)

	// The contrast first, or the assertion under it proves nothing: q really does
	// quit from this screen, which is why a field that shared the keyboard with it
	// could not take a query at all.
	if _, command := m.enter(screenSkills).Update(
		tea.KeyPressMsg{Code: 'q', Text: "q"}); !quits(command) {
		t.Fatal("q does not quit from the unfiltered listing, so swallowing it " +
			"below measures nothing")
	}
	next, command := opened.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if quits(command) {
		t.Error("q quit the program while the filter had the keyboard")
	}
	typing, isModel := next.(model)
	if !isModel {
		t.Fatalf("Update returned %T", next)
	}
	if typing.skills.query != "q" {
		t.Errorf("q left the query as %q", typing.skills.query)
	}

	// The rest of the screen's letters, straight into the field.
	typed := typeText(t, opened, "aek j?")
	if typed.skills.query != "aek j?" {
		t.Errorf("the query reads %q, want every letter and the space", typed.skills.query)
	}
	if typed.skills.adding || typed.skills.editing != "" {
		t.Error("a letter typed into the filter opened the form")
	}
	if typed.screen != screenSkills {
		t.Errorf("a letter typed into the filter left for %v", typed.screen)
	}

	// The arrows still walk the rows, because nothing else can — both of them,
	// since a filter that could only be walked one way would be a filter an
	// author has to reopen to correct an overshoot.
	walking := key(t, openSkillFilter(t, m), "down")
	walking = key(t, walking, "down")
	if walking.skills.cursor != 2 {
		t.Errorf("the down arrow left the cursor at %d", walking.skills.cursor)
	}
	if back := key(t, walking, "up"); back.skills.cursor != 1 {
		t.Errorf("the up arrow left the cursor at %d", back.skills.cursor)
	}
	if !walking.skills.filtering {
		t.Error("the down arrow closed the filter")
	}
	if walking.skills.query != "" {
		t.Errorf("the arrows typed %q into the query", walking.skills.query)
	}
	// And k does not, which is the price of the mode and the thing to notice if
	// somebody "fixes" it.
	if vim := typeText(t, openSkillFilter(t, m), "j"); vim.skills.cursor != 0 {
		t.Errorf("j moved the cursor to %d instead of being typed", vim.skills.cursor)
	}
}

// TestAFilterThatFindsNothingRefusesTheKeysThatReadARow is the second half of the
// cursor defect: a query matching nothing arrives one keystroke at a time, so it
// is an ordinary state rather than an edge case, and the keys that index a row
// have to decline instead of reaching into an empty slice.
//
// `a` is deliberately not among them, and that is the point of listing it: adding
// indexes nothing, and a filter that has found nothing is exactly the moment an
// author wants to write the skill they were looking for.
func TestAFilterThatFindsNothingRefusesTheKeysThatReadARow(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	empty := filterSkillsTo(t, m, noSkillQuery)
	if len(empty.skills.rows()) != 0 {
		t.Fatalf("%q left %d rows", noSkillQuery, len(empty.skills.rows()))
	}
	if empty.skills.cursor != 0 {
		t.Errorf("the cursor sits at %d over an empty listing", empty.skills.cursor)
	}
	if _, held := empty.skills.selected(); held {
		t.Error("an empty listing reports a skill under its cursor")
	}

	if edited := typeText(t, empty, "e"); edited.skills.editing != "" {
		t.Errorf("e opened the form on %q over an empty listing", edited.skills.editing)
	}
	if described := typeText(t, empty, "?"); described.screen != screenSkills {
		t.Errorf("? raised %v over an empty listing", described.screen)
	}
	for _, name := range []string{"down", "up"} {
		if walked := key(t, empty, name); walked.skills.cursor != 0 {
			t.Errorf("%s moved the cursor to %d over an empty listing",
				name, walked.skills.cursor)
		}
	}
	// The one key that still works, because the answer to "nothing matches" is
	// often to write it.
	if added := typeText(t, empty, "a"); !added.skills.adding {
		t.Error("a would not open the new-skill form over an empty listing")
	}

	// ⚠️ And the clamp on the way *down* to nothing, which is the half a refusal
	// cannot see. Walk to the last of several matches, then type one more letter
	// and narrow to one: selected() clamps on the way out, so e and ? would still
	// act on a real skill — but the listing draws its marker by comparing the
	// cursor with the row's index, so a cursor left past the end draws a listing
	// with no marker at all while the damage row underneath describes a row
	// nothing is pointing at.
	narrowing := openSkillFilter(t, m)
	narrowing = typeText(t, narrowing, someSkillQuery)
	for range len(narrowing.skills.rows()) {
		narrowing = key(t, narrowing, "down")
	}
	if narrowing.skills.cursor != len(narrowing.skills.rows())-1 {
		t.Fatalf("the cursor stopped at %d of %d matches, so it is not at the end "+
			"and the narrowing below measures nothing",
			narrowing.skills.cursor, len(narrowing.skills.rows()))
	}
	// " n" rather than a bare letter, so the space a Vietnamese name needs is
	// typed here as well: "long n" is "long nộ" and nothing else in the book.
	deeper := typeText(t, narrowing, " n")
	if got := len(deeper.skills.rows()); got != 1 {
		t.Fatalf("%q leaves %d rows, want the one that makes the clamp measurable",
			deeper.skills.query, got)
	}
	if got, rows := deeper.skills.cursor, len(deeper.skills.rows()); got > max(rows-1, 0) {
		t.Errorf("narrowing to %d rows left the cursor at %d", rows, got)
	}
	body, _ := deeper.skills.view(deeper)
	if marked, held := deeper.skills.selected(); held &&
		!strings.Contains(body, "> "+marked.ID) {
		t.Errorf("the narrowed listing draws no marker on %q:\n%s", marked.ID, body)
	}

	// And the screen says so, in both languages, rather than drawing an empty box
	// — and it says the *right* thing, because a listing can be empty for two
	// reasons and only one of them is a keystroke to take back. The empty book is
	// unreachable through the shipped data, so it is reached by hand: a branch
	// nothing renders is a branch with no width test and no translation test.
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		nothing := filterSkillsTo(t, base, noSkillQuery)
		body, _ := nothing.skills.view(base)
		if want := lang.Text(i18n.SkillsFilterNothing); !strings.Contains(body, want) {
			t.Errorf("the %s listing does not say %q when nothing matches:\n%s",
				lang, want, body)
		}
		bookless := nothing
		bookless.skills.skills = nil
		body, _ = bookless.skills.view(base)
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
		m, _, _ := start(t, lang)
		// Nothing typed yet: the row says what to type, because a bare label reads
		// as a broken row.
		opened := openSkillFilter(t, m)
		body, footer := opened.skills.view(opened)
		if want := lang.Text(i18n.SkillsFilterPrompt); !strings.Contains(body, want) {
			t.Errorf("the %s filter says nothing about what to type (%q):\n%s", lang, want, body)
		}
		// The footer is what says the field has the keyboard, since the row cannot
		// (colour is never information here, and there is no caret to draw).
		if want := lang.Text(i18n.SkillsFilterFooter); footer != want {
			t.Errorf("the %s footer while filtering is %q, want %q", lang, footer, want)
		}

		filtered := filterSkillsTo(t, m, someSkillQuery)
		body, footer = filtered.skills.view(filtered)
		want := lang.Say(i18n.SkillsFiltering, someSkillQuery,
			len(filtered.skills.rows()), len(filtered.skills.skills))
		if !strings.Contains(body, want) {
			t.Errorf("the %s filter row does not read %q:\n%s", lang, want, body)
		}
		// The keyboard is back on the rows, so the footer is the listing's again —
		// and it still names the key that reopens the field.
		if want := lang.Text(i18n.SkillsFooter); footer != want {
			t.Errorf("the %s footer after enter is %q, want %q", lang, footer, want)
		}
		// And the row is gone entirely when there is no filter, rather than drawn
		// empty: a row that says nothing is a row the listing has paid for.
		plain := m.enter(screenSkills)
		body, _ = plain.skills.view(plain)
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
	const drawable = minWidth - 1
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
		// The listing has to name the key that opens the filter, and the filter has
		// to name both ways out of it: an unadvertised mode is a mode nobody enters,
		// and a mode nobody can see the exit of is worse.
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

// TestTheSkillListingFitsTheSmallestWindowWhileFiltering is the other half of
// skillsRoom's arithmetic: the filter row is the tenth line the reserve pays for,
// and the reserve is what stops the listing overflowing its budget.
//
// The busiest filtered state is the field open with **nothing** typed, not a
// query in it: an empty query hides nothing, so the listing is as long as the
// window allows and the row above it costs the same either way. The cursor sits
// on a skill with a condition and an edit has been reported, which is what the
// unfiltered measurement (TestTheSkillListingFitsTheSmallestWindowAfterAnEdit)
// found to be the two lines that make the case worth measuring.
// Height only. The lines this screen can draw past the window are the two that
// carry the data directory's path — the header's and the write note's — and those
// are free text that frame clips on purpose; the width sweep in language_test.go
// is what measures the rest, with the filter's three states registered in
// everyScreen so it can see them at all.
func TestTheSkillListingFitsTheSmallestWindowWhileFiltering(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, query := range []string{"", someSkillQuery, noSkillQuery} {
			m, _, _ := start(t, lang)
			m.width, m.height = minWidth, minHeight
			m = m.enter(screenSkills)
			m = skillListTo(t, m, "detonate")
			if selected, _ := m.skills.selected(); selected.Requires == nil {
				t.Fatalf("%s no longer has a condition, so this measures the wrong case",
					selected.ID)
			}
			m = typeText(t, openSkillFilter(t, m), query)
			m.skills.edited = someSkillChange(t, m)
			drawn := m.screenContent()
			if strings.Contains(drawn, i18n.Vi.Text(i18n.Truncated)) ||
				strings.Contains(drawn, i18n.En.Text(i18n.Truncated)) {
				t.Errorf("the %s listing filtered by %q is truncated at %dx%d:\n%s",
					lang, query, minWidth, minHeight, drawn)
			}
		}
	}
}

// TestATypedFilterIsBounded is the one thing between the filter row and the
// window: the query is author-typed text with no length of its own, so the field
// takes a bounded number of letters and the row clips what it draws.
func TestATypedFilterIsBounded(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m.width, m.height = minWidth, minHeight
	long := strings.Repeat("dragon", filterLimit)
	typed := typeText(t, openSkillFilter(t, m), long)
	if got := len([]rune(typed.skills.query)); got != filterLimit {
		t.Errorf("the field took %d letters of %d typed, want %d",
			got, len(long), filterLimit)
	}
	// The row itself, at the floor and in the longer language: it is the one line
	// on this screen whose length an author decides.
	for _, lang := range i18n.Langs() {
		at := m
		at.lang = lang
		row := strings.TrimRight(
			typed.skills.filterRow(at, len(typed.skills.rows())), "\n")
		if width := lipgloss.Width(row) + 1; width > minWidth {
			t.Errorf("a %s filter row of %d letters draws %d cells of %d: %q",
				lang, filterLimit, width, minWidth, row)
		} else {
			t.Logf("%s filter row at %d letters: %d of %d cells",
				lang, filterLimit, lipgloss.Width(row), minWidth-1)
		}
	}
}
