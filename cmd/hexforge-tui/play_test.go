package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// atTheBattle is a model with a squad saved, the fight raised over it and the
// battle opened: the state every test below starts from.
func atTheBattle(t *testing.T, m model) model {
	t.Helper()
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	if m.screen != screenPlay {
		t.Fatalf("p opened %v rather than the battle", m.screen)
	}
	if m.play.err != nil {
		t.Fatalf("the battle would not start: %v", m.play.err)
	}
	return m
}

// TestTheFightRaisesABattleYouPlay is the wiring, and that the battle arrives
// already waiting on the player rather than on a key nobody knows to press.
func TestTheFightRaisesABattleYouPlay(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	if m.play.fight == nil {
		t.Fatal("no battle was built")
	}
	if m.play.pending == nil {
		t.Fatal("the battle opened without a turn for the player")
	}
	// The turn waiting is the player's own, which is the whole claim: every
	// other side's turn is taken on the way here.
	unit, known := m.play.fight.Unit(m.play.pending.Unit)
	if !known || unit.Side != m.play.side {
		t.Fatalf("the battle is waiting on %q", m.play.pending.Unit)
	}
	// And the cursor is on something that can actually be taken.
	if option := m.play.pending.Options[m.play.option]; !option.Available() {
		t.Errorf("the cursor opened on %q, which cannot be used: %s", option.Skill, option.Reason)
	}
	if back := key(t, m, "esc"); back.screen != screenFight {
		t.Errorf("esc left the battle for %v", back.screen)
	}
}

// TestATurnTakenMovesTheBattleOn is the loop: the player acts, the engine
// answers, and the next thing waiting is the player again.
func TestATurnTakenMovesTheBattleOn(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	before := len(m.play.script)
	events := len(m.play.events)
	m = key(t, m, "enter")
	if len(m.play.script) <= before {
		t.Fatal("nothing was written down")
	}
	if len(m.play.events) <= events {
		t.Error("a turn was taken and the battle recorded nothing")
	}
	// The opponent answered on the way back, so the script grew by more than the
	// one decision the player made.
	if len(m.play.script) < before+2 && !m.play.fight.Finished() {
		t.Errorf("the script holds %d decisions, want the player's and the engine's",
			len(m.play.script))
	}
	if m.play.pending == nil && !m.play.fight.Finished() {
		t.Error("the battle stopped without a turn for the player and without ending")
	}
}

// TestUndoTakesBackYourOwnTurnAndNotTheEngines is the one thing a hand-played
// battle needs that a simulation does not.
//
// It works because a battle is a pure function of its seed and the decisions
// taken: undo is not an unwinding, it is a shorter list replayed. That is the
// same property --verify rests on, which is why this is worth asserting here
// rather than trusting.
func TestUndoTakesBackYourOwnTurnAndNotTheEngines(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	opening := m.play.script
	m = key(t, m, "enter")
	if len(m.play.script) <= len(opening) {
		t.Fatal("nothing to take back")
	}
	fought := len(m.play.script)
	m = typeText(t, m, "u")
	if len(m.play.script) >= fought {
		t.Fatalf("undo left %d decisions of %d", len(m.play.script), fought)
	}
	// What is left ends with somebody else's turn: undo cuts at the player's
	// last decision, so nothing of theirs survives it.
	for _, decision := range m.play.script {
		unit, known := m.play.fight.Unit(decision.Unit)
		if known && unit.Side == m.play.side {
			t.Errorf("a turn of the player's own survived the undo: %+v", decision)
		}
	}
	// And the battle is waiting on the player again rather than on nobody.
	if m.play.pending == nil && !m.play.fight.Finished() {
		t.Error("after an undo the battle waits on nothing")
	}
	// Nothing to take back is not an error, it is a key that does nothing.
	fresh, _, _ := start(t, i18n.En)
	fresh = atTheBattle(t, fresh)
	fresh = typeText(t, fresh, "u")
	if fresh.play.err != nil {
		t.Errorf("undo with nothing of the player's in the script reported %v", fresh.play.err)
	}
}

// TestAnotherSeedIsAnotherBattle is what a player asks for when a pairing has
// been played once.
func TestAnotherSeedIsAnotherBattle(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	m = key(t, m, "enter")
	seed := m.play.seed
	fought := len(m.play.script)
	m = typeText(t, m, "n")
	if m.play.seed != seed+1 {
		t.Errorf("n moved the seed to %d, want %d", m.play.seed, seed+1)
	}
	if len(m.play.script) >= fought {
		t.Errorf("the new battle kept %d decisions from the old one", len(m.play.script))
	}
	if m.play.err != nil {
		t.Errorf("the new battle would not start: %v", m.play.err)
	}
}

// TestAimingIsAskedOnlyWhenItIsADecision is the second question: a skill with
// one legal cell does not ask it, because a question with one answer is not a
// decision.
func TestAimingIsAskedOnlyWhenItIsADecision(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	// A second member, so an enemy-aimed skill has two cells to choose between.
	second := m.squad.editing.Units[0].Clone()
	second.ID = "hai"
	second.Slot = hex.Offset{Col: hex.FormationCols - 1, Row: 0}
	m.squad.editing.Units = append(m.squad.editing.Units, second)
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	if m.play.pending == nil {
		t.Fatal("the battle opened without a turn for the player")
	}
	// Walk to a skill with more than one cell, if the opening turn has one.
	found := false
	for index, option := range m.play.pending.Options {
		if option.Available() && len(option.Aims) > 1 {
			m.play.option = index
			found = true
			break
		}
	}
	if !found {
		t.Skip("no skill on the opening turn has two cells to choose between")
	}
	m = key(t, m, "enter")
	if !m.play.aiming {
		t.Fatal("a skill with two cells acted without asking where")
	}
	body := m.screenContent()
	if !strings.Contains(body, strings.Fields(m.text(i18n.PlayAimAt, "x"))[0]) {
		t.Errorf("the aim list is not drawn:\n%s", body)
	}
	// esc backs out of the second question without spending the turn.
	before := len(m.play.script)
	m = key(t, m, "esc")
	if m.play.aiming {
		t.Error("esc did not leave the aim list")
	}
	if m.screen != screenPlay {
		t.Errorf("esc left the battle for %v", m.screen)
	}
	if len(m.play.script) != before {
		t.Error("backing out of an aim spent the turn")
	}
}

// TestABattlePlayedOutEndsAndSaysHow is the far end of the screen, driven by
// the key that hands each turn to the engine.
func TestABattlePlayedOutEndsAndSaysHow(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	for range playTurnLimit {
		if m.play.fight.Finished() || m.play.err != nil {
			break
		}
		m = typeText(t, m, "a")
	}
	if m.play.err != nil {
		t.Fatalf("the battle broke: %v", m.play.err)
	}
	if !m.play.fight.Finished() {
		t.Fatal("the battle never ended")
	}
	// It says which of the four endings it was, in words rather than in a code.
	body := m.screenContent()
	said := false
	for _, ending := range []i18n.Key{i18n.PlayWon, i18n.PlayLost, i18n.PlayDrawn, i18n.PlayEmptied} {
		if strings.Contains(body, m.text(ending)) {
			said = true
		}
	}
	if !said {
		t.Errorf("the battle ended and the screen does not say how:\n%s", body)
	}
	// The script is the whole battle, both sides in it, so what was played can
	// be replayed.
	sides := map[hex.Side]int{}
	for _, decision := range m.play.script {
		if unit, known := m.play.fight.Unit(decision.Unit); known {
			sides[unit.Side]++
		}
	}
	if sides[hex.SideAlly] == 0 || sides[hex.SideEnemy] == 0 {
		t.Errorf("the script holds %v, want turns from both sides", sides)
	}
	var _ battle.Script = m.play.script
}

// playBodyLines is how many lines the battle screen's body takes on the turn it
// opens, in both languages.
//
// # A tripwire, and deliberately not a bound
//
// The screen does **not** fit the declared minimum window and this does not
// pretend it does. Measured at 80x24: frame gives the body m.height - 2 rows less
// the two the header takes, so twenty survive of the twenty-eight below and the
// whole option list is cut with the Truncated marker. The list first appears at
// h >= 30 and the screen is un-truncated from h >= 32; once the log tail fills to
// its eight lines the body is 34 and wants h >= 38.
//
// Asserting that it fits would ship red, which is the mistake the reckless work
// already made once — a floor raised on the strength of a fix that never landed.
// So this is the shape TestNoTraitCarriesACharacterFarPastTheBudget uses: the
// measurement is stated, the fact that it is filed rather than fixed is stated,
// and the assertion stops it getting worse.
//
// Which rows the screen should give up to fit 24 is a layout question and its own
// piece of work — playLogLines is eight of them, and the board, the roster and the
// order line are the game client's own drawing rather than this screen's. Filed in
// TODO.md.
const playBodyLines = 28

// TestTheBattleScreenIsNoTallerThanItAlreadyWas is that tripwire.
//
// The turn it opens rather than a turn deep into a battle, because that state is a
// pure function of the fixture and the seed and so is the only figure two runs
// cannot disagree about. The log tail is what grows it afterwards, and it grows it
// by a known amount.
func TestTheBattleScreenIsNoTallerThanItAlreadyWas(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atTheBattle(t, m)
		m.width, m.height = minWidth, minHeight
		body, footer := m.play.view(m)
		if strings.TrimSpace(footer) == "" {
			t.Fatalf("%s: the battle draws no footer, so nothing was measured", lang)
		}
		lines := len(strings.Split(body, "\n"))
		if lines > playBodyLines {
			t.Errorf("%s: the battle screen's body is %d lines against the %d it was, "+
				"and it already needs a taller window than the declared minimum: "+
				"anything added to it now is a row the smallest window loses",
				lang, lines, playBodyLines)
		}
		t.Logf("%s: %d body lines at %dx%d", lang, lines, minWidth, minHeight)
	}
}

// TestEveryOptionIsSummarised is the first half of the claim: the turn's options
// each have a compact line, asked of the function that composes it rather than of
// the row that draws it.
//
// Split from the row test on purpose. What SummariseSkill returns and what fits
// on a row are two different questions, and asserting them together made the
// second one silently assert the first: a row is **clipped**, so "the full summary
// appears in the row" is a claim the design does not make and cannot keep. purify
// is where that showed — it strips three categories and its Vietnamese line came
// to 104 cells against 65 of room.
func TestEveryOptionIsSummarised(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atTheBattle(t, m)
		options := m.play.pending.Options
		if len(options) == 0 {
			t.Fatalf("%s: the opening turn offers nothing to describe", lang)
		}
		for _, option := range options {
			// The wording is i18n's own, so this asks for that string rather than
			// for a clause of its own: a test naming one here would be the wording
			// living in two places, which is what the AST scan refuses.
			summary := m.lang.SummariseSkill(
				mustSkill(t, m, option.Skill), m.lib.Patterns())
			if strings.TrimSpace(summary) == "" {
				t.Errorf("%s: %q summarises as nothing", lang, option.Skill)
			}
		}
	}
}

// TestAnOptionRowCarriesAsMuchOfItsSummaryAsItHasRoomFor is the second half: the
// row.
//
// Three things, and the row count is the one that matters most. The screen is
// already eight rows past the window it declares, so a pane under the list would
// have been a pane nobody in an 80x24 terminal ever sees — which is why the answer
// is a line beside each option.
//
// The other two are what clipping actually promises. The slot holds a **prefix**
// of the summary, and the longest prefix the room allows: shorter would be the row
// throwing away cells it has, longer would be the row running past the window. It
// is read out of the row at the offset the constants give rather than searched
// for, because a clipped summary has no index to find — which is the second
// failure purify produced, and it was a consequence of the first rather than a
// separate defect.
func TestAnOptionRowCarriesAsMuchOfItsSummaryAsItHasRoomFor(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atTheBattle(t, m)
		options := m.play.pending.Options
		if len(options) == 0 {
			t.Fatalf("%s: the opening turn offers nothing to draw", lang)
		}
		drawn := m.play.choices(m)
		rows := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
		// One heading and one row an option: exactly what it was before the
		// summary arrived.
		if len(rows) != len(options)+1 {
			t.Fatalf("%s: %d options draw %d rows, want one each under one heading:\n%s",
				lang, len(options), len(rows), drawn)
		}
		room := drawable - markerWidth - m.play.optionWidth() - optionGap
		for index, option := range options {
			row := rows[index+1]
			if width := lipgloss.Width(row); width > drawable {
				t.Errorf("%s: row %d is %d cells over the %d it has:\n%s",
					lang, index, width, drawable, row)
			}
			// The id sits in the measured column, which is what makes every slot
			// after it start in the same place.
			named, tail := optionColumns(m, row)
			if named != option.Skill {
				t.Errorf("%s: row %d holds %q in the id column, want %q:\n%s",
					lang, index, named, option.Skill, row)
			}
			summary := m.lang.SummariseSkill(
				mustSkill(t, m, option.Skill), m.lib.Patterns())
			if !strings.HasPrefix(summary, tail) || tail == "" {
				t.Errorf("%s: %q draws %q beside it, which is no part of %q",
					lang, option.Skill, tail, summary)
				continue
			}
			if tail != summary && lipgloss.Width(tail) != room {
				t.Errorf("%s: %q draws %d of the %d cells its row has and its "+
					"summary is %d: a clip has to take the whole room",
					lang, option.Skill, lipgloss.Width(tail), room,
					lipgloss.Width(summary))
			}
		}
	}
}

// optionColumns splits an option row into the id it names and whatever the slot
// after it holds.
//
// It cuts at the offset the row is built from rather than searching for either
// half: the point of the test above is that the column is measured, and a split
// that looked for the id would pass on a row whose column had drifted. Cut by
// rune, which is cut by cell here — every wording measures one cell per letter
// and an id is ASCII, both of which are held by tests of their own.
func optionColumns(m model, row string) (named, tail string) {
	letters := []rune(row)
	column := markerWidth + m.play.optionWidth()
	if len(letters) < column+optionGap {
		return "", ""
	}
	return strings.TrimRight(string(letters[markerWidth:column]), " "),
		string(letters[column+optionGap:])
}

// widestIDInTheBook is the id column at its worst: the widest skill id anything
// may bring, so minWidth - 1 less the marker, this and the gap is the least room a
// summary ever gets. Thirteen is poison_powder, and a longer id in the book moves
// this rather than the budget.
//
// summaryOvershoot is how far past that room a summary may run before this is a
// finding, in cells.
//
// ⚠️ Being over is allowed, because the row clips and the clause order is built
// for it — reach and cooldown are last precisely because the end is what goes.
// outrage is over: 62 cells in Vietnamese and 65 in English, against 62, because
// it is the one skill carrying damage, a self-applied status *and* a caster-side
// amplifier. What is refused is a clause that can only ever arrive trimmed.
const (
	widestIDInTheBook = 13
	summaryOvershoot  = 4
)

// TestNoSummaryIsWiderThanARowCanHold is the guard that was missing, and purify
// is why it exists.
//
// It strips three categories, and the summary enumerated them: 79 cells for that
// clause alone in Vietnamese, before the aim and the cooldown were appended, on a
// line that has 62 at worst — so the aim, the range and the cooldown could never
// be drawn at all. Nothing said so. The golden records the text and not its width,
// and the row test cannot help: a clip is the design, so an unboundedly long
// summary clips to a legal row and every assertion passes.
//
// So the width is measured here, over every skill the library holds rather than
// over the shipped book — purify is a fixture skill, which is exactly how it
// stayed invisible while describe.golden looked fine.
func TestNoSummaryIsWiderThanARowCanHold(t *testing.T) {
	room := minWidth - 1 - markerWidth - widestIDInTheBook - optionGap
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		skills := m.lib.Skills().Skills()
		if len(skills) == 0 {
			t.Fatalf("%s: the library holds no skills, so nothing was measured", lang)
		}
		worst, worstAt := 0, ""
		for _, declared := range skills {
			if width := lipgloss.Width(
				m.lang.SummariseSkill(declared, m.lib.Patterns())); width > worst {
				worst, worstAt = width, declared.ID
			}
		}
		if worst > room+summaryOvershoot {
			t.Errorf("%s: %q summarises to %d cells against the %d a row has at "+
				"the widest id, past the %d this allows: a clause that can only "+
				"arrive trimmed is not a reading",
				lang, worstAt, worst, room, room+summaryOvershoot)
		}
		t.Logf("%s: the widest is %q at %d cells, against %d of room",
			lang, worstAt, worst, room)
	}
}

// TestAClippedRowKeepsTheLongestPrefixItHasRoomFor is the clip itself, measured
// where it actually happens.
//
// ⚠️ It is a test of its own because the turn atTheBattle opens **cannot reach
// it**. Those four options come to at most 44 cells against 65 of room, so nothing
// clips, and the prefix assertion in the row test above is satisfied trivially by
// every one of them — a clip mutated to take one cell less than the room passed the
// whole suite. An assertion no fixture can fire is the defect this branch already
// paid for once in everyScreen.
//
// So the prompt is built rather than played: the widest-summary skill in the book
// beside the widest-id skill, which is the narrowest room and the longest line at
// the same time. Both are looked up rather than named, for the reason
// widestElementRow and inTheWidestRank are — which skill is widest is a fact about
// a language, and naming one would measure English and skip Vietnamese.
//
// A prompt is a value the engine hands over, so building one measures the drawing
// and asks nothing of the engine. The unit and the aims are the real ones.
//
// ⚠️ One subtest a language, because whether the book holds a line long enough to
// clip is a fact about the language and only Vietnamese lacks one. t.Skip is
// runtime.Goexit, so skipping inside a plain loop over the two would abandon the
// whole test at the first language and never measure the second — an unreachable
// assertion arrived at from the other direction.
//
// ⚠️ **And at least one language has to actually reach it.** A skip is honest
// while some other language still measures the clip; a test where every language
// skips has gone quiet, and nobody reads a PASS to find out. Vietnamese skips
// today and English does not — outrage is 62 cells there against 62 of room, and
// 65 in English — so if a wording change ever puts Vietnamese inside the budget
// as well, this fails rather than reporting two skips and a pass. It clears itself
// the other way too: the day a Vietnamese line grows past the row, the skip stops
// firing on its own.
func TestAClippedRowKeepsTheLongestPrefixItHasRoomFor(t *testing.T) {
	const drawable = minWidth - 1
	measured := 0
	for _, lang := range i18n.Langs() {
		t.Run(lang.String(), func(t *testing.T) {
			// theClippedRow skips by way of runtime.Goexit, so this line is
			// reached only by a language that had a clip to measure.
			theClippedRow(t, lang, drawable)
			measured++
		})
	}
	if measured == 0 {
		t.Error("no language holds a summary long enough to be clipped, so the clip " +
			"itself is drawn by nothing here — the row still clips, and nothing measures it")
	}
}

func theClippedRow(t *testing.T, lang i18n.Lang, drawable int) {
	t.Helper()
	m, _, _ := start(t, lang)
	m = atTheBattle(t, m)
	wordiest, longest := theWidestSummary(m), theWidestID(m)
	room := drawable - markerWidth - lipgloss.Width(longest.ID) - optionGap
	summary := m.lang.SummariseSkill(wordiest, m.lib.Patterns())
	if lipgloss.Width(summary) <= room {
		// Honest rather than contrived: no skill in the book is long enough to be
		// cut at the widest id column in this language, so there is no clip to
		// measure. Vietnamese is here today — outrage comes to exactly the 62
		// cells the row has.
		t.Skipf("%s: the widest summary is %q at %d cells and the row has %d, "+
			"so nothing in the book clips", lang, wordiest.ID,
			lipgloss.Width(summary), room)
	}
	m.play.pending = &battle.Prompt{
		Unit: m.play.pending.Unit, Turn: m.play.pending.Turn,
		Options: []battle.Option{
			{Skill: wordiest.ID, Aims: m.play.pending.Options[0].Aims},
			{Skill: longest.ID, Aims: m.play.pending.Options[0].Aims},
		},
	}
	m.play.option = 0
	rows := strings.Split(strings.TrimRight(m.play.choices(m), "\n"), "\n")
	if len(rows) != 3 {
		t.Fatalf("%s: two options drew %d rows", lang, len(rows))
	}
	row := rows[1]
	if width := lipgloss.Width(row); width > drawable {
		t.Errorf("%s: the clipped row is %d cells over the %d it has:\n%s",
			lang, width, drawable, row)
	}
	_, tail := optionColumns(m, row)
	if !strings.HasPrefix(summary, tail) {
		t.Errorf("%s: the row draws %q, which is no part of %q", lang, tail, summary)
	}
	// The whole room and not a cell less: a clip that stopped short would be the
	// row throwing away space it has, and nothing else would say so.
	if lipgloss.Width(tail) != room {
		t.Errorf("%s: %q was cut to %d cells of the %d its row has (summary is %d)",
			lang, wordiest.ID, lipgloss.Width(tail), room, lipgloss.Width(summary))
	}
	// And it really was cut, or this measured a row that fit.
	if tail == summary {
		t.Errorf("%s: %q was not clipped at all", lang, wordiest.ID)
	}
}

// theWidestSummary is the skill whose compact line takes the most cells in the
// language in front, and theWidestID the one with the longest id.
//
// Looked up rather than named for the reason every other widest-thing helper here
// is: both answers are facts about the data and one of them is a fact about the
// language too.
func theWidestSummary(m model) skill.Skill {
	var found skill.Skill
	most := -1
	for _, declared := range m.lib.Skills().Skills() {
		if width := lipgloss.Width(
			m.lang.SummariseSkill(declared, m.lib.Patterns())); width > most {
			found, most = declared, width
		}
	}
	return found
}

func theWidestID(m model) skill.Skill {
	var found skill.Skill
	most := -1
	for _, declared := range m.lib.Skills().Skills() {
		if width := lipgloss.Width(declared.ID); width > most {
			found, most = declared, width
		}
	}
	return found
}

// TestAnUnavailableOptionKeepsItsReasonAndNotItsSummary is the one row where the
// two answers compete for one slot.
//
// ⚠️ The reason wins, and that is not an oversight to be tidied later. Why a
// skill cannot be cast is the live question the moment a cursor steps over it,
// and what the skill does is one keystroke away on the description screen.
//
// Read out of the slot rather than searched for, for the reason the tests above
// are: "the summary is not in the row" passes for free on a row where the summary
// was merely clipped, which would make this assert nothing at all.
func TestAnUnavailableOptionKeepsItsReasonAndNotItsSummary(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	// Something has to go on cooldown first: every option is available on the
	// turn a battle opens, which is exactly why this is not covered by the tests
	// above.
	found := -1
	for range 40 {
		if m.play.pending == nil || m.play.err != nil {
			break
		}
		for index, option := range m.play.pending.Options {
			if !option.Available() && option.Reason != "" {
				found = index
			}
		}
		if found >= 0 {
			break
		}
		m = typeText(t, m, "a")
	}
	if found < 0 {
		t.Skip("nothing came off cooldown in forty turns, so no option is refused")
	}
	option := m.play.pending.Options[found]
	rows := strings.Split(strings.TrimRight(m.play.choices(m), "\n"), "\n")
	_, tail := optionColumns(m, rows[found+1])
	if tail == "" || !strings.HasPrefix(option.Reason, tail) {
		t.Errorf("the refused option draws %q beside it, which is no part of its "+
			"reason %q", tail, option.Reason)
	}
	summary := m.lang.SummariseSkill(mustSkill(t, m, option.Skill), m.lib.Patterns())
	if strings.TrimSpace(summary) == "" {
		t.Fatalf("%q summarises as nothing, so this proves nothing", option.Skill)
	}
	if strings.HasPrefix(summary, tail) {
		t.Errorf("the refused option draws %q, which is its summary rather than "+
			"its reason %q", tail, option.Reason)
	}
}

// TestTheQuestionMarkDescribesTheOptionInFront is the long form of the line
// beside the cursor.
//
// It reuses the description screen the skill listing and the cast browser raise,
// so what a player reads while choosing is the same paragraph an author reads
// while tuning. What is asserted here is the wiring and the three ways it must
// not misbehave.
func TestTheQuestionMarkDescribesTheOptionInFront(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atTheBattle(t, m)
		fought := len(m.play.script)
		option := m.play.pending.Options[m.play.option]
		raised := typeText(t, m, "?")
		if raised.screen != screenBlurb || raised.blurb.from != screenPlay {
			t.Fatalf("%s: ? opened %v from %v", lang, raised.screen, raised.blurb.from)
		}
		// ⚠️ The battle is a pointer, so raising and leaving this must step no
		// turn: view reads the option and nothing else.
		if len(raised.play.script) != fought || raised.play.fight.Finished() {
			t.Errorf("%s: raising the description spent %d decisions",
				lang, len(raised.play.script)-fought)
		}
		body := raised.screenContent()
		if !strings.Contains(body, option.Skill) &&
			!strings.Contains(body, raised.lang.GlossedSkill(
				mustSkill(t, raised, option.Skill))) {
			t.Errorf("%s: the description does not name %q:\n%s", lang, option.Skill, body)
		}
		// The same sentences the listing draws, rather than a second rendering.
		for _, line := range skillLines(raised, mustSkill(t, raised, option.Skill)) {
			if !strings.Contains(body, strings.TrimSpace(line)) {
				t.Errorf("%s: the description is missing the listing's line %q:\n%s",
					lang, line, body)
			}
		}
		// esc comes back to the battle, and comes back to the battle that was
		// being played rather than to a fresh one.
		back := key(t, raised, "esc")
		if back.screen != screenPlay {
			t.Errorf("%s: esc left the description for %v", lang, back.screen)
		}
		if len(back.play.script) != fought || back.play.seed != m.play.seed {
			t.Errorf("%s: coming back rebuilt the battle: seed %d, %d decisions",
				lang, back.play.seed, len(back.play.script))
		}
		// ↑/↓ walks the option behind, so four of them can be read one after
		// another.
		walked := key(t, raised, "down")
		if len(m.play.pending.Options) > 1 && walked.play.option == raised.play.option {
			t.Errorf("%s: the cursor behind the description did not move", lang)
		}
	}
}

// TestTheQuestionMarkWorksWhileAimingAndNotWithoutAPrompt is the two edges of
// that key.
//
// ⚠️ Aiming is the state where it is most wanted and the easiest to forget: the
// skill is chosen and the cell is not, so "what does this actually do" is still
// the open question — and it is the state the width sweep could not see either.
// With no prompt there is nothing to describe, and the key does nothing rather
// than opening an empty screen.
func TestTheQuestionMarkWorksWhileAimingAndNotWithoutAPrompt(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	aiming := m
	aiming.play.aiming = true
	raised := typeText(t, aiming, "?")
	if raised.screen != screenBlurb || raised.blurb.from != screenPlay {
		t.Fatalf("? while aiming opened %v from %v", raised.screen, raised.blurb.from)
	}
	if !strings.Contains(raised.screenContent(),
		aiming.play.pending.Options[aiming.play.option].Skill) &&
		strings.TrimSpace(raised.screenContent()) == "" {
		t.Error("the description while aiming is empty")
	}
	// And it leaves the aim where it was: coming back has to land on the second
	// question rather than on the first.
	if back := key(t, raised, "esc"); !back.play.aiming {
		t.Error("coming back from the description dropped the aim")
	}
	// ↑/↓ does nothing here: the skill is settled, and walking the options would
	// change what is described out from under a half-taken decision.
	if walked := key(t, raised, "down"); walked.play.option != raised.play.option {
		t.Error("the description walked the option list while a skill was already chosen")
	}

	// Nothing pending is a key that does nothing.
	over := m
	for range playTurnLimit {
		if over.play.fight.Finished() || over.play.err != nil {
			break
		}
		over = typeText(t, over, "a")
	}
	if !over.play.fight.Finished() {
		t.Fatal("the battle never ended, so there is no promptless state to test")
	}
	if quiet := typeText(t, over, "?"); quiet.screen != screenPlay {
		t.Errorf("? on a finished battle opened %v", quiet.screen)
	}
}

// TestTheBattleFootersNameTheDescriptionKeyAndFit is the defect this shipped
// with, measured rather than counted.
//
// ⚠️ Both battle footers were over the window and nothing said so: the fixture
// in everyScreen handed the width sweep a battle that was already finished, so
// PlayFooter and PlayAimFooter were drawn by nothing in the suite and came to 82
// and 83 cells against the 79 there are. The sweep covers them now; this holds
// the other half, which a width test cannot — that the key the whole feature
// hangs on is still named after the next person trims a footer.
func TestTheBattleFootersNameTheDescriptionKeyAndFit(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		footers := map[string]string{
			"battle": m.text(i18n.PlayFooter, saveKeyLabel()),
			"aim":    m.text(i18n.PlayAimFooter),
		}
		for name, footer := range footers {
			if width := lipgloss.Width(footer); width > drawable {
				t.Errorf("%s: the %s footer is %d cells over the %d it has: %q",
					lang, name, width, drawable, footer)
			}
			if !strings.Contains(footer, describeKey) {
				t.Errorf("%s: the %s footer does not name %q: %q",
					lang, name, describeKey, footer)
			}
			t.Logf("%s: the %s footer is %d cells", lang, name, lipgloss.Width(footer))
		}
	}
}

// describeKey is the keystroke that raises a description, named here so the test
// above is about the footer and not about a letter.
const describeKey = "?"

// mustSkill is one skill out of the library in hand.
func mustSkill(t *testing.T, m model, id string) skill.Skill {
	t.Helper()
	declared, err := m.lib.Skills().Lookup(id)
	if err != nil {
		t.Fatalf("the book does not hold %q: %v", id, err)
	}
	return declared
}

// ⚠️ The screen as a save leaves it is deliberately **not** in everyScreen. Its
// first line names the file's absolute path, which under a test's temporary
// directory is longer than any window — so the width sweep would be measuring
// where the test happened to run rather than anything anybody wrote. The wording
// itself is held by the two-language key tests, and that a save says something
// at all is held below.
//
// TestASavedBattleReplaysExactly is the whole point of writing one out: a log
// that could not be re-run would be a picture of a battle rather than a record
// of it.
//
// It re-runs the log the way `hexarena --verify` does — build from the log's own
// roster and seed, replay its choices, compare every event — and against this
// library's books rather than the embedded copy, which is the one difference
// between the two and the reason the save carries a rebuild note.
func TestASavedBattleReplaysExactly(t *testing.T) {
	m, lib, dir := start(t, i18n.En)
	m = atTheBattle(t, m)
	// A few turns in, so the script has both sides in it and the save is not
	// recording an opening board.
	for range 6 {
		if m.play.fight.Finished() {
			break
		}
		m = typeText(t, m, "a")
	}
	m = key(t, m, "ctrl+s")
	if m.play.err != nil {
		t.Fatalf("the save was refused: %v", m.play.err)
	}
	if len(m.play.notes) == 0 {
		t.Fatal("a write said nothing")
	}
	// It landed in the battles folder, under a name built from the pairing.
	written, err := filepath.Glob(filepath.Join(dir, "battles", "*.json"))
	if err != nil || len(written) != 1 {
		t.Fatalf("the battles folder holds %v (%v)", written, err)
	}
	raw, err := os.ReadFile(written[0])
	if err != nil {
		t.Fatalf("read it back: %v", err)
	}
	log, err := battle.ParseLog(raw)
	if err != nil {
		t.Fatalf("parse the log: %v", err)
	}
	if !log.Replayable() {
		t.Fatal("the log records no placement, so nothing could re-run it")
	}
	if log.Seed != m.play.seed || len(log.Choices) != len(m.play.script) {
		t.Errorf("the log holds seed %d and %d choices, want %d and %d",
			log.Seed, len(log.Choices), m.play.seed, len(m.play.script))
	}

	rerun, err := battle.New(lib.Books(), log.Seed, log.Roster)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	rerun.Begin()
	if _, _, err := rerun.Replay(log.Choices, playTurnLimit, nil); err != nil {
		t.Fatalf("re-running the battle: %v", err)
	}
	produced := rerun.Drain()
	if len(produced) != len(log.Events) {
		t.Fatalf("the log records %d events but re-running produced %d",
			len(log.Events), len(produced))
	}
	for index := range produced {
		if produced[index] != log.Events[index] {
			t.Fatalf("event %d differs from the log:\nlogged  %+v\nre-ran  %+v",
				index, log.Events[index], produced[index])
		}
	}

	// Saving the same battle again writes over itself rather than leaving two
	// copies of one thing: the pairing and the seed are what a battle is.
	m = key(t, m, "ctrl+s")
	again, err := filepath.Glob(filepath.Join(dir, "battles", "*.json"))
	if err != nil || len(again) != 1 {
		t.Errorf("saving twice left %v", again)
	}
}

// TestASquadNameCannotClimbOutOfTheBattlesFolder is the one thing a file name
// built from author-typed text has to be checked for.
func TestASquadNameCannotClimbOutOfTheBattlesFolder(t *testing.T) {
	m, _, dir := start(t, i18n.En)
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	m.squad.editing.ID = "../../escaped"
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	m = key(t, m, "ctrl+s")
	if m.play.err != nil {
		t.Fatalf("the save was refused: %v", m.play.err)
	}
	written, err := filepath.Glob(filepath.Join(dir, "battles", "*.json"))
	if err != nil || len(written) != 1 {
		t.Fatalf("the battles folder holds %v (%v)", written, err)
	}
	if strings.Contains(filepath.Base(written[0]), "..") {
		t.Errorf("the log landed at %q", written[0])
	}
}
