package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// # The sweep, and why this client had to be born with one
//
// `TODO.md` records **five** separate occasions where a screen slipped out of
// cmd/hexforge-tui's `everyScreen` and silently lost all three of the properties
// a screen there is held to: that every line fits the declared floor, that it
// speaks the language it was asked for, and that no Vietnamese data name leaks
// onto an English row. A screen nothing renders has no width test, no
// translation test and no leak test, and it passes every one of them.
//
// So this client arrives with the sweep rather than after it, and with one thing
// the authoring tool's has never had: a **count** to be held against. A sweep is
// a map somebody remembered to write in; `screenCount` is what a walk can
// enumerate, so a view added to the enum without an entry here — or without a
// written reason for staying out — is a red test.
//
// ⚠️ **A count over screens is not a count over states**, and that is stated
// rather than papered over: a screen has as many interesting states as it has
// paths through View, and no constant can enumerate those. What is done about it
// is what internal/screen's golden does — every hand-built state asserts that it
// drew the line it exists for, so a state that has quietly stopped reaching its
// own branch fails loudly instead of recording an ordinary screen twice.

// everyScreen is each view this client draws, plus the states of one that share
// no line with each other.
//
// ⚠️ **Every battle state enters the screen for itself, and that is not
// tidiness.** draw.PlayScreen is the one screen holding something a model copy
// does not copy: the battle is a *battle.Battle, so `state := battle` shares the
// pointer and playing one of them out steps all of them. That is exactly what
// the authoring tool's sweep shipped — three states shared a battle, the
// finished one played it out, and both battle footers were over the window for a
// release with every width test passing.
func everyScreen(t *testing.T, m model) map[string]model {
	t.Helper()
	screens := map[string]model{
		"menu":     m.enter(screenMenu),
		"cast":     m.enter(screenCast),
		"skills":   m.enter(screenSkills),
		"traits":   m.enter(screenTraits),
		"species":  m.enter(screenSpecies),
		"works":    m.enter(screenWorks),
		"squads":   m.enter(screenSquads),
		"elements": widestElement(m.enter(screenElements)),
	}
	// The chart and the statuses reference are the two screens reached by a
	// keystroke rather than by the menu, and they are **raised** rather than
	// assigned: a screen this client could not actually get to is a screen the
	// sweep would be measuring on its own behalf.
	screens["chart"] = raisedFrom(t, m.enter(screenElements), "g", screenChart)
	screens["statuses"] = raisedFrom(t, m.enter(screenTraits), "?", screenStatuses)
	// The description screen in each of its three readings. It is one screen
	// branching on the kind of subject it was handed, and the three branches share
	// no line, so measuring one measures nothing about the other two.
	screens["a skill blurb"] = raisedFrom(t, m.enter(screenSkills), "?", screenBlurb)
	screens["a trait blurb"] = raisedFrom(t, atACharacterWithTraits(t, m), "?", screenBlurb)
	// The three states of the typed filter on the skill listing. Driven with the
	// keys a reader would press, not by writing the query onto the screen's own
	// field: the query is what decides which rows there are, so a hand-set field
	// would measure this test's idea of the filter rather than the one / opens.
	filtering := typeText(t, m.enter(screenSkills), "/")
	if !filtering.skills.Filtering {
		t.Fatal("/ did not open the skill filter, so its three states are drawn by nothing")
	}
	found := typeText(t, filtering, someSkillQuery)
	nothing := typeText(t, filtering, noSkillQuery)
	if rows, all := len(found.skills.Rows()), len(found.skills.Skills); rows < 2 || rows >= all {
		t.Fatalf("the query %q finds %d of %d skills, so the filtered listing is not a "+
			"narrowed one", someSkillQuery, rows, all)
	}
	if rows := len(nothing.skills.Rows()); rows != 0 {
		t.Fatalf("the query %q finds %d skills, so the empty result is drawn by nothing",
			noSkillQuery, rows)
	}
	screens["filtering skills"] = filtering
	screens["filtered skills"] = found
	screens["skills filtered to none"] = nothing
	// The squad catalogue with nothing on it, which is what a reader sees before
	// the authoring tool has been run: built by emptying the rows for the reason
	// internal/screen's golden empties a build's trait slot — nothing the fixture
	// ships need be deleted for a wording to be measured.
	empty := m.enter(screenSquads)
	empty.squads.Saved = nil
	if !strings.Contains(drawnBody(empty), empty.text(i18n.SquadsEmpty)) {
		t.Fatalf("the empty catalogue draws no line saying so, so this records an ordinary "+
			"listing twice:\n%s", drawnBody(empty))
	}
	screens["an empty squad catalogue"] = empty
	// The battle, in each of the states it draws. Every one of them enters the
	// screen for itself — see the note above.
	screens["a battle"] = withAFullLog(t, m.enter(screenBattle))
	screens["aiming"] = aiming(t, withAFullLog(t, m.enter(screenBattle)))
	screens["a battle over"] = playedOut(t, m.enter(screenBattle))
	screens["a saved battle"] = saved(t, withAFullLog(t, m.enter(screenBattle)))
	screens["a battle blurb"] = raisedFrom(t, withAFullLog(t, m.enter(screenBattle)), "?", screenBlurb)
	screens["a scrolled battle log"] = scrolledBack(t, m)
	screens["a squeezed battle"] = squeezed(t, withAFullLog(t, m.enter(screenBattle)))
	screens["a battle with no pairing"] = unpaired(t, m)
	return screens
}

// The two queries the filter states are driven with: one that narrows the book
// and one nothing answers to.
//
// A substring of an id rather than a name, so the fixture does not depend on
// which skills the book happens to have Vietnamese names for.
const (
	someSkillQuery = "a"
	noSkillQuery   = "zzqx"
)

// drawnBody is the framed screen, which is what a reader sees and what every
// assertion in this file is about.
func drawnBody(m model) string { return m.screenContent() }

// raisedFrom presses a raising key and asserts the client arrived, which is the
// half a sweep entry cannot state about itself: a raise that declined leaves the
// reader on the screen behind, and that screen renders perfectly well.
func raisedFrom(t *testing.T, m model, name string, want screen) model {
	t.Helper()
	raised := key(t, m, name)
	if raised.screen != want {
		t.Fatalf("%q on screen %v landed on %v, want %v", name, m.screen, raised.screen, want)
	}
	return raised
}

// atACharacterWithTraits puts the browser on the character carrying the most
// traits at the cap, which is the busiest the description screen ever draws.
//
// Looked up rather than named, for the reason widestElement is: which character
// that is is a fact about the book, and naming one would tie this record to
// content an author is free to change. A character carrying none draws the
// carries-nothing line, which is a different screen from the one this is for.
func atACharacterWithTraits(t *testing.T, m model) model {
	t.Helper()
	browser := m.enter(screenCast)
	browser.cast.Level = progression.LevelCap
	found, most := 0, 0
	for index, character := range browser.cast.Rows() {
		held := len(m.lib.KitPassives(
			character.PassivesAt(progression.LevelCap, progression.Furthest)))
		if held > most {
			found, most = index, held
		}
	}
	if most == 0 {
		t.Fatal("no character in the book carries a trait at the cap, so the trait blurb " +
			"records the carries-nothing line")
	}
	browser.cast.Cursor = found
	return browser
}

// widestElement puts the elements listing on the element whose description is
// longest: the rows are all one shape, so what varies is the pane below them.
func widestElement(m model) model {
	found, most := 0, 0
	for index, member := range element.All() {
		for _, line := range strings.Split(m.lang.DescribeElement(member, m.lib.Chart()), "\n") {
			if width := lipgloss.Width(line); width > most {
				found, most = index, width
			}
		}
	}
	m.elements.Cursor = found
	return m
}

// withAFullLog plays enough turns for the log to want more rows than the floor
// will give it.
//
// ⚠️ **Without it the log is never the section that goes.** The opening board's
// log is one row a unit entering, so the budget's last-in-line section is never
// dropped — a sweep over the opening turn alone would report the priority as
// measured while measuring nothing.
func withAFullLog(t *testing.T, m model) model {
	t.Helper()
	if m.battle.Fight == nil {
		t.Fatalf("the battle screen opened on no pairing: %v", m.battle.Err)
	}
	for range 40 {
		if len(m.battle.LogRows(m.ctx())) >= draw.PlayLogWanted || m.battle.Fight.Finished() {
			break
		}
		m = key(t, m, "a")
	}
	if rows := len(m.battle.LogRows(m.ctx())); rows < draw.PlayLogWanted {
		t.Fatalf("the history came to %d rows of %d, so a squeezed log was not measured",
			rows, draw.PlayLogWanted)
	}
	if m.battle.Pending == nil {
		t.Fatal("the battle stopped waiting on the player, so no option list is drawn")
	}
	return m
}

// aiming is the second question a turn asks: the cells the chosen skill may be
// pointed at, under the option list.
//
// The aim list is asserted rather than assumed: an option with nowhere to point
// records an ordinary battle screen twice.
func aiming(t *testing.T, m model) model {
	t.Helper()
	m.battle.Aiming = true
	option := m.battle.Pending.Options[m.battle.Option]
	if len(option.Aims) == 0 {
		t.Fatalf("the option %q has nowhere to point, so the aim list is empty", option.Skill)
	}
	if want := m.text(i18n.PlayAimAt, option.Skill); !strings.Contains(drawnBody(m), want) {
		t.Fatalf("the aiming battle draws no aim list:\n%s", drawnBody(m))
	}
	return m
}

// playedOut hands every turn to the engine until the battle finishes, which is
// the state that replaces the option list and takes the over footer with it.
func playedOut(t *testing.T, m model) model {
	t.Helper()
	for range draw.PlayTurnLimit {
		if m.battle.Fight.Finished() || m.battle.Err != nil {
			break
		}
		m = key(t, m, "a")
	}
	if m.battle.Err != nil {
		t.Fatalf("the battle broke: %v", m.battle.Err)
	}
	if !m.battle.Fight.Finished() {
		t.Fatal("the battle never ended")
	}
	said := false
	for _, ending := range []i18n.Key{
		i18n.PlayWon, i18n.PlayLost, i18n.PlayDrawn, i18n.PlayEmptied,
	} {
		if strings.Contains(drawnBody(m), m.text(ending)) {
			said = true
		}
	}
	if !said {
		t.Fatalf("the finished battle says nothing about how it ended:\n%s", drawnBody(m))
	}
	return m
}

// saved is the pair of notes a write leaves behind, drawn under the option list.
//
// ⚠️ **It really writes**, through the same forge.Library.SaveBattleLog a reader
// reaches with ctrl+s, into the scratch data directory the fixture arranged. The
// note names the file, so the golden's fixture hands the library a **relative**
// directory and noAbsolutePath is what says the recorded rows carry no machine
// in them.
func saved(t *testing.T, m model) model {
	t.Helper()
	written := key(t, m, "ctrl+s")
	if len(written.battle.Notes) == 0 {
		t.Fatalf("ctrl+s on the battle left no note (%v), so this records an ordinary "+
			"battle twice", written.battle.Err)
	}
	rows := written.battle.Wrote(written.ctx())
	if len(rows) == 0 {
		t.Fatal("the note renders no row")
	}
	if !strings.Contains(drawnBody(written), rows[0]) {
		t.Fatalf("the saved battle does not draw its own note:\n%s", drawnBody(written))
	}
	return written
}

// scrolledBack is the log walked back off its own tail, which is the one state
// that draws the position on the heading row.
//
// ⚠️ It has to be **built twice over**. A battle a few turns old has a history
// that fits its frame, so nothing is hidden and the position is correctly not
// drawn; and at the floor a three-a-side board is given no log row at all, so a
// fixture standing there would be pressing scroll keys at a section that is not
// on the screen. So the history is played out past any frame in a window the log
// has rows in, and the frame is walked back with the key a reader would press.
func scrolledBack(t *testing.T, m model) model {
	t.Helper()
	// A window the log has rows in, which the floor is not for three a side.
	tall := m
	tall.width, tall.height = minWidth, 40
	tall = tall.enter(screenBattle)
	for range draw.PlayTurnLimit {
		if len(tall.battle.LogRows(tall.ctx())) >= scrolledLogRows ||
			tall.battle.Fight.Finished() || tall.battle.Err != nil {
			break
		}
		tall = key(t, tall, "a")
	}
	if got := len(tall.battle.LogRows(tall.ctx())); got < scrolledLogRows {
		t.Fatalf("the battle finished with a history of %d rows against the %d this needs, "+
			"so nothing above the frame was constructed", got, scrolledLogRows)
	}
	for range 4 {
		tall = key(t, tall, "pgup")
	}
	if tall.battle.LogFollow || tall.battle.LogOffset == 0 {
		t.Fatalf("the log did not scroll back (following %v at %d), so the position is "+
			"drawn by nothing", tall.battle.LogFollow, tall.battle.LogOffset)
	}
	return tall
}

// scrolledLogRows is how long "longer than any window this suite draws" is: the
// body of a 160x60 window is 56 rows, all of which the log could be given.
const scrolledLogRows = 120

// squeezed is the battle in a window it cannot fit, which is a state of the
// screen rather than a screen of its own.
//
// It is here for the reason every other state is: the dim line naming what the
// window was too short for is wording like any other, and every other model in
// the sweep stands in a window tall enough to hold the whole screen.
func squeezed(t *testing.T, m model) model {
	t.Helper()
	m.width, m.height = minWidth, minHeight
	if !strings.Contains(drawnBody(m), m.text(i18n.PlayHidden, "")[:8]) {
		t.Fatalf("the squeezed battle names nothing it gave up, so the notice is drawn by "+
			"nothing:\n%s", drawnBody(m))
	}
	return m
}

// unpaired is the screen a reader reaches before any side has been built: no
// battle, and the line saying nothing has been built.
//
// It is built as a value rather than by emptying the catalogue and entering,
// because `enter` re-reads the catalogue off the library on the way in — which
// is the right thing for the client and the wrong thing for a fixture trying to
// take the rows away. What drives the client's own path instead is
// TestAnEmptyCatalogueOpensNoBattle.
func unpaired(t *testing.T, m model) model {
	t.Helper()
	m.battle = draw.NewPlayScreen().Open(m.ctx(), placement.Squad{}, placement.Squad{})
	m.screen = screenBattle
	if m.battle.Fight != nil || m.battle.Err != nil {
		t.Fatalf("two empty squads opened a battle (%v)", m.battle.Err)
	}
	if !strings.Contains(drawnBody(m), m.text(i18n.SquadsEmpty)) {
		t.Fatalf("the unpaired battle says nothing about a side having to be built:\n%s",
			drawnBody(m))
	}
	return m
}

// notSwept is every view this client can draw that the sweep deliberately does
// not register, with the reason it does not.
//
// ⚠️ **It is a list of decisions rather than a backlog**, and it is what makes
// the walk below able to fail: without it the walk would either pass on a screen
// nothing measures or would have to be relaxed into something that passes on any
// map whatsoever.
var notSwept = map[screen]string{
	screenPreview: "the art preview draws rasterised art, so what a sweep entry would " +
		"assert about it is an open question rather than an oversight — the same " +
		"decision cmd/hexforge-tui's everyScreen and internal/screen's golden both " +
		"take, and it is filed in TODO.md rather than settled here",
}

// TestEveryScreenThisClientDrawsIsSwept is the net TODO.md's five slipped
// screens asked for.
//
// ⚠️ **It walks screenCount rather than the sweep**, because the failure being
// guarded against is a view somebody added to the enum and did not register.
// Ranging over the map would ask it whether it holds what it holds — which is
// exactly what happened five times in the authoring tool, where a screen is
// registered by being remembered.
//
// Both halves are checked. A screen that is neither swept nor excused is a
// screen with no width test, no translation test and no leak test; and an
// excuse for a screen the sweep *does* register is an excuse nobody needs, which
// is how a stale exclusion outlives the reason for it.
func TestEveryScreenThisClientDrawsIsSwept(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	swept := make(map[screen]bool)
	entries := everyScreen(t, base)
	for _, m := range entries {
		swept[m.screen] = true
	}
	for value := range int(screenCount) {
		view := screen(value)
		reason, excused := notSwept[view]
		switch {
		case swept[view] && excused:
			t.Errorf("screen %d is both swept and excused as %q; an excuse for a screen the "+
				"sweep draws is one nobody needs", value, reason)
		case !swept[view] && !excused:
			t.Errorf("screen %d is drawn by nothing in everyScreen and has no entry in "+
				"notSwept, so it has no width test, no translation test and no leak test",
				value)
		}
	}
	for view := range notSwept {
		if view < 0 || view >= screenCount {
			t.Errorf("notSwept excuses screen %d, which this client does not declare", view)
		}
	}
	// And the sweep is doing more than one entry a screen, which is the other
	// half of what it is for: a screen has as many interesting states as it has
	// paths through View, and a sweep with one entry each would be a list of
	// screens wearing a sweep's name.
	if len(entries) <= len(swept) {
		t.Errorf("the sweep holds %d entries over %d screens, so no screen is registered in "+
			"more than one state", len(entries), len(swept))
	}
	t.Logf("%d entries over %d of this client's %d screens, %d excused",
		len(entries), len(swept), screenCount, len(notSwept))
}

// TestEveryWordingFitsTheMinimumWidth is the layout measurement, taken against
// the real screens in both languages rather than by eye.
//
// Vietnamese runs a fifth to a third longer than English for the same sentence,
// so this is the language the floor was decided by. Lines carrying free text
// from the data are skipped — a biography or a filesystem path has no length the
// program can promise, and frame clips those on purpose.
func TestEveryWordingFitsTheMinimumWidth(t *testing.T) {
	// The window's last column is left empty. A line that fills a terminal's
	// final cell wraps to the next row on some of them, and one wrapped line
	// pushes the footer off the bottom — the exact failure frame exists to
	// prevent.
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		base, lib, _ := start(t, lang)
		base.width, base.height = 200, 60
		free := append(freeText(lib), whoMayCarry(lang, lib)...)
		free = append(free, traitCarriers(lib)...)
		for name, m := range everyScreen(t, base) {
			m.width, m.height = 200, 60
			for _, line := range strings.Split(m.screenContent(), "\n") {
				if carriesFreeText(line, free) {
					continue
				}
				if width := lipgloss.Width(line); width > drawable {
					t.Errorf("the %s screen in %s draws a line %d cells wide, over the %d it has:\n%s",
						name, lang, width, drawable, line)
				}
			}
		}
		// The too-small screen is measured against something smaller still, since
		// it is only ever drawn in a window that is already too narrow.
		small := base
		small.width, small.height = 40, 10
		for _, line := range strings.Split(small.screenContent(), "\n") {
			if width := lipgloss.Width(line); width > 24 {
				t.Errorf("the too-small screen in %s draws %d cells:\n%s", lang, width, line)
			}
		}
	}
}

// TestEveryScreenRendersInBothLanguages walks the whole program twice.
//
// The markers are words that could only have come from a wording that was not
// translated — a screen still holding an English sentence in Vietnamese, or the
// reverse. Free text from the data files is skipped, because a biography is
// English in cast.json and will still be English on a Vietnamese screen: it is
// the author's prose, not the program's.
func TestEveryScreenRendersInBothLanguages(t *testing.T) {
	englishMarkers := []string{
		"MISSING", "quit", "back", "describe", "fight", "another battle",
		"look a skill up", "of the", "who may carry",
	}
	vietnameseMarkers := []string{
		"nhân vật", "chiêu", "thoát", "quay lại", "vào trận", "hiệu ứng", "chủng loài",
	}
	cases := []struct {
		lang    i18n.Lang
		unwant  []string
		mustSay []string
	}{
		{i18n.Vi, englishMarkers, vietnameseMarkers},
		{i18n.En, vietnameseMarkers, englishMarkers},
	}
	for _, test := range cases {
		base, lib, _ := start(t, test.lang)
		free := freeText(lib)
		spoken := make(map[string]bool)
		for name, m := range everyScreen(t, base) {
			drawn := m.screenContent()
			if strings.TrimSpace(drawn) == "" {
				t.Errorf("the %s screen drew nothing in %s", name, test.lang)
			}
			for _, line := range strings.Split(drawn, "\n") {
				// The footer names the other language in its own name, which is the
				// one place a word from it is meant to be there.
				line = strings.ReplaceAll(line, "tiếng Việt", "")
				if carriesFreeText(line, free) || partOfFreeText(line, free) {
					continue
				}
				for _, marker := range test.unwant {
					if strings.Contains(line, marker) {
						t.Errorf("the %s screen in %s still says %q:\n%s",
							name, test.lang, marker, line)
					}
				}
				for _, marker := range test.mustSay {
					if strings.Contains(line, marker) {
						spoken[marker] = true
					}
				}
			}
		}
		// And it really is the language asked for, rather than a screen that
		// happens to be free of the other one's words.
		if len(spoken) == 0 {
			t.Errorf("nothing on any screen reads like %s", test.lang)
		}
	}
}

// TestNoScreenLeaksAGlossIntoTheWrongLanguage is the third property the
// authoring tool's sweep carries, and the one whose failure is quietest: a
// Vietnamese name in an English column is a *wrong translation* where the bare
// id was the right answer, and it reads as a feature working.
//
// ⚠️ **Two lookups, not one, and collecting only the first is how a leak
// survived over there.** A compiled gloss comes out of i18n's own tables and is
// empty in English by construction, while a **data** name — a trait's, a
// species', a skill's — is a field on the declaration and is Vietnamese whoever
// asks. Reading such a field raw put "bền bỉ · máu độc" under an English traits
// row and nothing objected, because nothing knew the name existed.
//
// ⚠️ **Line by line, skipping authored prose.** What this hunts is a data name
// in a *column*. A name occurring inside a biography or a species' note is not
// that — it is the author's sentence, which is Vietnamese in both languages —
// and whole-screen matching cannot tell the two apart.
func TestNoScreenLeaksAGlossIntoTheWrongLanguage(t *testing.T) {
	vietnamese, lib, _ := start(t, i18n.Vi)
	// The positive half first, because the sweep below would pass on a client
	// that glossed nothing at all: an id on a Vietnamese screen arrives with its
	// name beside it.
	browser := vietnamese.enter(screenCast)
	body, _ := browser.cast.View(browser.ctx())
	glossed := 0
	for _, character := range browser.cast.Rows() {
		want := i18n.Vi.GlossedAffinity(character.Element)
		if want != "" && strings.Contains(body, want) {
			glossed++
		}
	}
	if glossed == 0 {
		t.Errorf("no row of the Vietnamese cast browser shows an element with its name, so "+
			"the sweep below proves nothing:\n%s", body)
	}

	english, _, _ := start(t, i18n.En)
	english.width, english.height = 200, 60
	var names []string
	for _, character := range lib.Characters().All() {
		names = append(names, i18n.Vi.Gloss(character.Archetype))
		for _, member := range character.Element.Elements() {
			names = append(names, i18n.Vi.Gloss(member.String()))
		}
		for _, entry := range character.Skills {
			names = append(names, i18n.Vi.Gloss(entry.ID))
		}
		for _, held := range lib.KitPassives(character.PassivesAt(1, progression.Furthest)) {
			names = append(names, held.Name)
		}
		for _, kind := range lib.KitSpecies(character.Species) {
			names = append(names, kind.Name)
		}
	}
	free := freeText(lib)
	for name, m := range everyScreen(t, english) {
		m.width, m.height = 200, 60
		drawn := m.screenContent()
		for _, line := range strings.Split(drawn, "\n") {
			if carriesFreeText(line, free) || partOfFreeText(line, free) {
				continue
			}
			for _, unwanted := range names {
				if unwanted != "" && strings.Contains(line, unwanted) {
					t.Errorf("the %s screen in English holds the gloss %q:\n%s", name, unwanted, line)
				}
			}
		}
	}
}
