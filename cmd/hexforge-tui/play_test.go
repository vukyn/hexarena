package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
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

// atABattleOf is atTheBattle at a pairing of a given number of units a side.
//
// The squad is built by **looking the cast up** rather than by naming
// characters, which is the rule every fixture here follows: a test that names a
// character breaks the day somebody edits the cast for a reason that has nothing
// to do with it.
//
// ⚠️ **The characters repeat when the fixture cast is smaller than the squad, and
// that is deliberate.** What the height tests measure is **rows**, one a unit, so
// ten distinct characters would be ten readings of the same number; the fixture
// declares two. This is a count of rows and not a roster anybody would field.
func atABattleOf(t *testing.T, m model, side int) model {
	t.Helper()
	if side < 1 || side > hex.MaxTeamSize {
		t.Fatalf("a side of %d is outside the %d a squad may field", side, hex.MaxTeamSize)
	}
	m = menuTo(t, m, screenSquads)
	m.squad = someSquad(t, m)
	// Named after its size, so a catalogue may hold one of each without one
	// squad replacing another — Library.SaveSquad replaces by id.
	id := "do-thu-" + strconv.Itoa(side)
	m.squad.editing.ID = id
	m.squad.idInput.SetValue(id)
	characters := m.squad.characters
	if len(characters) == 0 {
		t.Fatal("the fixture cast is empty, so no squad can be built from it")
	}
	units := make([]placement.Placement, 0, side)
	for index, slot := range squadSlots(side) {
		character := characters[index%len(characters)]
		unit := placement.Placement{
			// The id is the slot rather than a word, so it is unique by
			// construction however many units the squad holds.
			ID:        slot.String(),
			Character: character.ID,
			Level:     progression.LevelCap,
			Slot:      slot,
		}
		known := character.SkillsAt(unit.Level, progression.Furthest)
		if len(known) > cast.SkillSlots {
			known = known[:cast.SkillSlots]
		}
		unit.Skills = known
		if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
			unit.Passives = traits[:cast.TraitSlots]
		}
		units = append(units, unit)
	}
	m.squad.editing.Units = units
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	// Point the catalogue at the squad just saved and both of the fight's sides
	// at it: the cursor because the catalogue may already hold somebody else's
	// squad, and the away chooser because it opens on the first squad on the
	// list rather than on the one under the cursor. A squad against a copy of
	// itself is the one pairing any catalogue is guaranteed to be able to field.
	m.squad.cursor = squadIndex(t, m, id)
	m = typeText(t, m, "f")
	m.fight.away = m.fight.home
	m = typeText(t, m, "p")
	if m.screen != screenPlay {
		t.Fatalf("p opened %v rather than the battle", m.screen)
	}
	if m.play.err != nil {
		t.Fatalf("a %d-a-side battle would not start: %v", side, m.play.err)
	}
	if m.play.fight == nil {
		t.Fatalf("a %d-a-side battle built nothing", side)
	}
	if want := side * 2; len(m.play.fight.Units()) != want {
		t.Fatalf("a %d-a-side pairing fielded %d units, want %d",
			side, len(m.play.fight.Units()), want)
	}
	return m
}

// squadIndex is where a squad of a given id sits in the catalogue.
func squadIndex(t *testing.T, m model, id string) int {
	t.Helper()
	for index, squad := range m.squad.saved {
		if squad.ID == id {
			return index
		}
	}
	t.Fatalf("the catalogue does not hold the squad %q it was just given", id)
	return 0
}

// squadSlots is where the units of a squad of n stand: the front rank first, so
// a range-one skill has somebody to point at whatever the size.
func squadSlots(n int) []hex.Offset {
	var out []hex.Offset
	for col := hex.FormationCols - 1; col >= 0 && len(out) < n; col-- {
		for row := range hex.FormationRows {
			if len(out) == n {
				break
			}
			out = append(out, hex.Offset{Col: col, Row: row})
		}
	}
	return out
}

// # The battle screen's height, and where the cut lands
//
// The screen cannot fit the window the tool declares, and nothing below pretends
// it can. Measured at 80x24, where playBodyRoom leaves the body twenty rows: the
// heading is one, tui.Board is a fixed ten, tui.Roster is one plus a row a unit,
// tui.Order is one, the log is up to playLogLines and the option list is one plus
// a row an option — so a 1v1 wants twenty of those rows before a single blank or
// log line, a 3v3 twenty-four and a 5v5 **twenty-eight**. A legal squad is up to
// hex.MaxTeamSize a side, so twenty-eight is the floor for one, and a summon puts
// units on the board past the five a squad brought.
//
// What was fixable is **where the cut lands**. frame cuts from the bottom and the
// option list was the last thing the body wrote, so the one thing a player has to
// see in order to act was the first thing thrown away. The bound that is true
// after the fix — and the one worth holding, in place of the old tripwire — is
// that the option list is never cut and the body never overruns its room.
//
// playHeights is the sweep and playSides the squads it is taken over: one, three
// and five a side, the last being the largest a squad may field.
var (
	playHeights = heightsFrom(minHeight, 48)
	playSides   = []int{1, 3, hex.MaxTeamSize}
)

func heightsFrom(low, high int) []int {
	var out []int
	for height := low; height <= high; height++ {
		out = append(out, height)
	}
	return out
}

// withAFullLog plays the engine's own turns until the log has filled to
// playLogLines rows.
//
// ⚠️ **Without it the log is never the section that goes.** The opening board's
// log is one row a unit entering, so a 1v1 at the floor still holds all four of
// them and the section the priority drops first is never dropped — a sweep over
// the opening turn alone would report the log's place in the order as unmeasured
// while looking like it had measured it.
func withAFullLog(t *testing.T, m model) model {
	t.Helper()
	for range 20 {
		if len(m.play.log(m, playLogLines)) >= playLogLines || m.play.fight.Finished() {
			break
		}
		m = typeText(t, m, "a")
	}
	if rows := len(m.play.log(m, playLogLines)); rows < playLogLines {
		t.Fatalf("the log filled to %d rows of %d, so a squeezed log was not measured",
			rows, playLogLines)
	}
	if m.play.pending == nil {
		t.Fatal("the battle stopped waiting on the player, so no option list is drawn")
	}
	return m
}

// drawn is which of the game client's drawings the body came back holding.
//
// ⚠️ **It is read out of the screen and never off the plan.** A test that asked
// the view what it had decided would be a test measuring itself, so this looks
// for the drawings themselves — tui.Board's own rows, tui.Roster's own rows,
// tui.Order's own line, the log's own rendered rows — among the rows the body
// handed back.
type drawn struct {
	board  bool
	roster int
	order  bool
	log    int
	notice string
}

func whatIsDrawn(m model) (drawn, string, string) {
	body, footer := m.play.view(m)
	lines := strings.Split(body, "\n")
	present := make(map[string]int, len(lines))
	for _, line := range lines {
		present[line]++
	}
	p := m.play
	var found drawn
	board := strings.Split(tui.Board(p.fight, p.tags), "\n")
	found.board = present[board[0]] > 0 && present[board[len(board)-1]] > 0
	// The header is not a unit, so the count starts past it.
	for _, row := range strings.Split(tui.Roster(p.fight, p.tags), "\n")[1:] {
		if present[row] > 0 {
			found.roster++
		}
	}
	found.order = present[m.style.dim.Render(tui.Order(p.fight.Queue(), p.tags, 6))] > 0
	// Counted as a multiset and not by lookup: the log's rows are not distinct —
	// tui.Line opens a turn with a blank one — so asking whether each is on the
	// screen counts every blank row once per blank row there is.
	for _, row := range p.log(m, playLogLines) {
		if present[row] > 0 {
			present[row]--
			found.log++
		}
	}
	// The notice is the row under the heading, matched on the wording's own
	// opening rather than on a word of it: a wrapped save note may perfectly well
	// begin with "the".
	if len(lines) > 1 && strings.HasPrefix(lines[1], noticeOpening(m)) {
		found.notice = lines[1]
	}
	return found, body, footer
}

// noticeOpening is everything the notice says before its list, which is what
// tells that row apart from every other row on the screen.
func noticeOpening(m model) string {
	const mark = "\x00"
	return strings.SplitN(m.text(i18n.PlayHidden, mark), mark, 2)[0]
}

// rowHolding is the row whose id column is the given id, cursor marker and all.
//
// The id column rather than the whole row, because "bolt" is a prefix of
// "arc_bolt" and a substring search would answer with the wrong row — and then
// check the marker on it, which is the shape of a test that passes for the wrong
// reason.
func rowHolding(body, id string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		if len(line) > markerWidth && strings.HasPrefix(line[markerWidth:], id) {
			return line, true
		}
	}
	return "", false
}

// TestTheOptionListSurvivesEveryWindowTheToolWillDraw is the bound that replaced
// the tripwire.
//
// Every option on its own row, the marker on the one under the cursor, and the
// footer still the last row of the framed screen — at every height from minHeight
// up, in both languages, for one, three and five units a side.
func TestTheOptionListSurvivesEveryWindowTheToolWillDraw(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			base, _, _ := start(t, lang)
			base = atABattleOf(t, base, side)
			options := base.play.pending.Options
			if len(options) == 0 {
				t.Fatalf("%s %dv%d: the opening turn offers nothing to draw", lang, side, side)
			}
			for _, height := range playHeights {
				m := base
				m.width, m.height = minWidth, height
				_, body, footer := whatIsDrawn(m)
				if strings.TrimSpace(footer) == "" {
					t.Fatalf("%s %dv%d h=%d: no footer, so nothing was measured",
						lang, side, side, height)
				}
				for index, option := range options {
					row, ok := rowHolding(body, option.Skill)
					if !ok {
						t.Errorf("%s %dv%d h=%d: %q is not on the screen:\n%s",
							lang, side, side, height, option.Skill, body)
						continue
					}
					if index == m.play.option && !strings.HasPrefix(row, "> ") {
						t.Errorf("%s %dv%d h=%d: the option under the cursor draws %q, "+
							"want the marker", lang, side, side, height, row)
					}
				}
				// And the framed screen still ends on the footer, which is where
				// the keys are: a screen whose keys have scrolled away is one
				// nobody can leave.
				screen := strings.Split(m.screenContent(), "\n")
				if last := screen[len(screen)-1]; !strings.Contains(last, strings.TrimSpace(footer)) {
					t.Errorf("%s %dv%d h=%d: the last row is %q, want the footer %q",
						lang, side, side, height, last, footer)
				}
			}
		}
	}
}

// TestTheAimListSurvivesEveryWindowToo is the same claim for the second question.
//
// Aiming is the taller of the two states — the option list is still drawn above
// the cells — so it is where a budget reserving only the options would show.
func TestTheAimListSurvivesEveryWindowToo(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			base, _, _ := start(t, lang)
			base = atABattleOf(t, base, side)
			base.play.aiming = true
			aims := base.play.pending.Options[base.play.option].Aims
			if len(aims) == 0 {
				t.Fatalf("%s %dv%d: the option under the cursor has nowhere to point",
					lang, side, side)
			}
			for _, height := range playHeights {
				m := base
				m.width, m.height = minWidth, height
				_, body, _ := whatIsDrawn(m)
				for index, cell := range aims {
					row, ok := rowHolding(body, cell.String())
					if !ok {
						t.Errorf("%s %dv%d h=%d: the cell %s is not on the screen:\n%s",
							lang, side, side, height, cell, body)
						continue
					}
					if index == m.play.aim && !strings.HasPrefix(row, "> ") {
						t.Errorf("%s %dv%d h=%d: the aim under the cursor draws %q, "+
							"want the marker", lang, side, side, height, row)
					}
				}
			}
		}
	}
}

// TestTheBattleScreenIsNeverTruncated is the cleanest statement of the fix.
//
// The body budgets itself against the room frame will give it, so frame never has
// to cut — and the Truncated marker, which is what a cut looks like, therefore
// never appears on this screen at any window the tool will draw.
//
// Four bodies rather than one, because they are four different heights: waiting
// on a skill, waiting on a cell, with a save's notes under the list, and over.
// ⚠️ Each state is entered for itself and the finished one is played out from its
// own model — playScreen holds a *battle.Battle, so a copy driven to its end
// steps the battle every other copy points at, which is the fixture defect this
// screen has already been bitten by once.
func TestTheBattleScreenIsNeverTruncated(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			base, _, _ := start(t, lang)
			base = withAFullLog(t, atABattleOf(t, base, side))
			states := map[string]model{"a turn": base}
			aiming := base
			aiming.play.aiming = true
			states["aiming"] = aiming
			states["saved"] = key(t, base, "ctrl+s")
			over, _, _ := start(t, lang)
			over = atABattleOf(t, over, side)
			for range playTurnLimit {
				if over.play.fight.Finished() || over.play.err != nil {
					break
				}
				over = typeText(t, over, "a")
			}
			if !over.play.fight.Finished() {
				t.Fatalf("%s %dv%d: the battle never ended, so the ending was not measured",
					lang, side, side)
			}
			states["over"] = over
			for name, state := range states {
				for _, height := range playHeights {
					m := state
					m.width, m.height = minWidth, height
					body, _ := m.play.view(m)
					if rows := len(strings.Split(body, "\n")); rows > playBodyRoom(height) {
						t.Errorf("%s %dv%d %s h=%d: the body is %d rows against the %d it has",
							lang, side, side, name, height, rows, playBodyRoom(height))
					}
					if drawn := m.screenContent(); strings.Contains(drawn, m.text(i18n.Truncated)) {
						t.Errorf("%s %dv%d %s h=%d: the screen was cut:\n%s",
							lang, side, side, name, height, drawn)
					}
				}
			}
		}
	}
}

// TestTheBattleScreenDropsInTheOrderItStates measures the priority rather than
// assuming it.
//
// Two universal claims over the whole sweep and two steps that have to exist in
// it:
//
//   - A roster row is given up only when the board is already gone. The board is
//     ten rows whose information is recoverable — the aim list prints the occupant
//     beside every cell it offers — while a roster row is a unit's health and
//     effects, which is what a turn is decided on.
//   - The log is drawn only where the order line is, which is the log going first
//     of the two.
//   - A height exists where the board is gone and the roster is whole.
//   - A one-row step exists where the log goes and the order line stays.
//
// ⚠️ **What disappears is not monotone in the height, and that is the priority
// working rather than a defect.** The board takes ten rows or none, so at the
// height where it still just fits it takes the rows the order line and the log
// would have had, and one row shorter it cannot fit at all and both come back.
// Only the *priority* is monotone: rows are offered to the roster, then the board,
// then the order line, then the log, at every height there is.
func TestTheBattleScreenDropsInTheOrderItStates(t *testing.T) {
	for _, side := range playSides {
		base, _, _ := start(t, i18n.Vi)
		base = withAFullLog(t, atABattleOf(t, base, side))
		units := len(base.play.fight.Units())
		boardGoneRosterWhole, logWentBeforeOrder := false, false
		previous := drawn{log: -1}
		for _, height := range playHeights {
			m := base
			m.width, m.height = minWidth, height
			found, body, _ := whatIsDrawn(m)
			if found.roster < units && found.board {
				t.Errorf("%dv%d h=%d: %d of %d units are drawn while the board still is:\n%s",
					side, side, height, found.roster, units, body)
			}
			if found.log > 0 && !found.order {
				t.Errorf("%dv%d h=%d: %d log rows are drawn with no order line:\n%s",
					side, side, height, found.log, body)
			}
			if !found.board && found.roster == units {
				boardGoneRosterWhole = true
			}
			// The sweep walks upwards, so the previous reading is the window one
			// row shorter.
			if previous.log == 0 && found.log > 0 && previous.order {
				logWentBeforeOrder = true
			}
			previous = found
		}
		if !boardGoneRosterWhole {
			t.Errorf("%dv%d: no height in the sweep drops the board and keeps the roster whole, "+
				"so the priority between those two went unmeasured", side, side)
		}
		if !logWentBeforeOrder {
			t.Errorf("%dv%d: no one-row step in the sweep takes the log and leaves the order "+
				"line, so the priority between those two went unmeasured", side, side)
		}
	}
}

// TestTheRosterIsClippedARowAtATime is the one section that compresses by degrees,
// and the case the fixture does not reach on its own.
//
// ⚠️ **It has to be constructed.** A 5-a-side pairing at the declared floor keeps
// its whole roster: the heading and a four-option list reserve seven of the twenty
// rows and the notice one, and the twelve left are exactly what the pane wants.
// Aiming is what takes them — the cells are reserved with the options — so the
// case is built by opening the second question, and this fails rather than passing
// quietly if that turns out to clip nothing.
func TestTheRosterIsClippedARowAtATime(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		base = atABattleOf(t, base, hex.MaxTeamSize)
		base.play.aiming = true
		units := len(base.play.fight.Units())
		clipped := 0
		for _, height := range playHeights {
			m := base
			m.width, m.height = minWidth, height
			found, body, _ := whatIsDrawn(m)
			if found.roster >= units || found.roster == 0 {
				continue
			}
			clipped++
			hidden := units - found.roster
			want := m.text(i18n.PlayHiddenUnits, hidden)
			if hidden == 1 {
				want = m.text(i18n.PlayHiddenUnitsOne)
			}
			if !strings.Contains(found.notice, want) {
				t.Errorf("%s h=%d: %d of %d units are drawn and the notice reads %q, want %q:\n%s",
					lang, height, found.roster, units, found.notice, want, body)
			}
		}
		if clipped == 0 {
			t.Errorf("%s: no height in the sweep clipped the roster, so the section that "+
				"compresses a row at a time went unmeasured", lang)
		}
	}
}

// TestTheLogIsCappedByRenderedLinesAndNotByEvents is the second defect the height
// work turned up, held.
//
// playLogLines was a cap on **events** and is a cap on **rows**. The two are not
// the same number: tui.Line opens a turn with a blank row of its own, so one event
// arrives as two rows, and the section with the loosest claim on the screen was
// the one whose stated budget did not hold.
//
// ⚠️ The multi-row event has to be **reached**, which the opening board does not
// do: it takes a turn or two before the log holds one. This builds that and fails
// if it cannot, rather than measuring a log every row of which is one event.
func TestTheLogIsCappedByRenderedLinesAndNotByEvents(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = withAFullLog(t, atABattleOf(t, m, 3))
	widest := 0
	for _, event := range m.play.events {
		if line := tui.Line(event, m.play.tags); line != "" {
			widest = max(widest, len(strings.Split(line, "\n")))
		}
	}
	if widest < 2 {
		t.Fatal("no event in the log renders to more than one row, so the cap this test " +
			"is about could not have been exceeded and nothing was measured")
	}
	// The old cap taken the old way: the last playLogLines events, rendered.
	events := m.play.events
	if len(events) > playLogLines {
		events = events[len(events)-playLogLines:]
	}
	byEvent := 0
	for _, event := range events {
		if line := tui.Line(event, m.play.tags); line != "" {
			byEvent += len(strings.Split(line, "\n"))
		}
	}
	if byEvent <= playLogLines {
		t.Fatalf("the last %d events render to %d rows, so the old cap happened to hold "+
			"here and this fixture measures nothing", playLogLines, byEvent)
	}
	t.Logf("the last %d events render to %d rows", playLogLines, byEvent)
	// And the cap that is one: never more rows than it was given, at every budget
	// from nothing up to the whole of it.
	for room := 0; room <= playLogLines; room++ {
		if rows := len(m.play.log(m, room)); rows > room {
			t.Errorf("a log budget of %d rows drew %d", room, rows)
		}
	}
	// The tail rather than the head, which is what a player has to read: every
	// budget that draws anything ends on the log's own last row.
	whole := m.play.log(m, playLogLines)
	if len(whole) == 0 {
		t.Fatal("the log renders nothing")
	}
	for room := 1; room <= playLogLines; room++ {
		rows := m.play.log(m, room)
		if len(rows) == 0 {
			continue
		}
		if rows[len(rows)-1] != whole[len(whole)-1] {
			t.Errorf("a budget of %d rows ends on %q, want the log's own last row %q",
				room, rows[len(rows)-1], whole[len(whole)-1])
		}
	}
}

// TestTheSaveNoteOutranksTheBoard is where a write's own answer sits in the
// priority, and the arithmetic that says the other half of it cannot be reached.
//
// A save's notes are the answer to a keystroke pressed a moment ago and they name
// the file that was written, so they take rows before the board does. They are
// **not** reserved: two notes wrap to as many as five rows, and reserving them
// could crowd out the option list, which may never be cut.
//
// ⚠️ **The branch that drops them is unreachable at any window the tool draws, so
// it is built at one the tool refuses.** At the floor the body has twenty rows,
// the heading and the option list reserve seven and the notice one, so twelve are
// left against the five the notes want — and even the aim list, the tallest tail a
// squad can field, leaves enough. A shorter window is what takes them, and
// m.tooSmall draws a message instead of this screen there, so the case is
// constructed rather than left to be rendered by nothing.
func TestTheSaveNoteOutranksTheBoard(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atABattleOf(t, m, hex.MaxTeamSize)
		m = key(t, m, "ctrl+s")
		if m.play.err != nil {
			t.Fatalf("%s: the save failed, so no note was measured: %v", lang, m.play.err)
		}
		notes := m.play.wrote(m)
		if len(notes) == 0 {
			t.Fatalf("%s: the save left no note behind", lang)
		}
		for _, height := range playHeights {
			state := m
			state.width, state.height = minWidth, height
			body, _ := state.play.view(state)
			if !strings.Contains(body, notes[0]) {
				t.Errorf("%s h=%d: the save's own note is not on the screen:\n%s",
					lang, height, body)
			}
		}
		// The window that does take it, below the floor the tool draws at.
		short := m
		short.width, short.height = minWidth, minHeight-9
		body, _ := short.play.view(short)
		if strings.Contains(body, notes[0]) {
			t.Fatalf("%s: neither the sweep nor a window below it drops the save note, so "+
				"the wording naming it is rendered by nothing:\n%s", lang, body)
		}
		notice := strings.Split(body, "\n")[1]
		if !strings.Contains(notice, short.text(i18n.PlayHiddenNote)) {
			t.Errorf("%s: the save note was dropped and the notice does not say so: %q",
				lang, notice)
		}
		// And the longest notice there is — every section named at once — still
		// fits the narrowest window the tool draws.
		if width := lipgloss.Width(notice); width > drawable {
			t.Errorf("%s: the fullest notice is %d cells wide, over the %d there are: %q",
				lang, width, drawable, notice)
		}
	}
}

// TestTheNoticeNamesWhatIsMissingAndNothingElse is the line itself.
//
// It exists because a screen silently missing its board reads as a broken screen.
// Three things: it appears when and only when a section is missing, it fits the
// narrowest window the tool draws — the wording sweep renders every screen at
// width 200, so the one row that is only ever drawn in a short window has to be
// measured here — and every section it can name is named by some height in the
// sweep, so none of those wordings is rendered by nothing.
//
// ⚠️ **A shorter log tail is not a missing section.** The log is a tail by design
// and playLogLines is its own cap, so a tail that came back two rows shorter is
// the section working; a tail with no rows at all is not, and that is the line the
// notice is drawn on.
func TestTheNoticeNamesWhatIsMissingAndNothingElse(t *testing.T) {
	const drawable = minWidth - 1
	nameable := []i18n.Key{i18n.PlayHiddenBoard, i18n.PlayHiddenOrder, i18n.PlayHiddenLog}
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			base, _, _ := start(t, lang)
			base = withAFullLog(t, atABattleOf(t, base, side))
			named := make(map[i18n.Key]bool)
			for _, height := range playHeights {
				m := base
				m.width, m.height = minWidth, height
				found, body, _ := whatIsDrawn(m)
				units := len(m.play.fight.Units())
				wanted := len(m.play.log(m, playLogLines))
				missing := !found.board || found.roster < units || !found.order ||
					(wanted > 0 && found.log == 0)
				switch {
				case missing && found.notice == "":
					t.Errorf("%s %dv%d h=%d: a section is missing and the screen does not "+
						"say which:\n%s", lang, side, side, height, body)
				case !missing && found.notice != "":
					t.Errorf("%s %dv%d h=%d: nothing is missing and the screen says "+
						"otherwise: %q", lang, side, side, height, found.notice)
				}
				if width := lipgloss.Width(found.notice); width > drawable {
					t.Errorf("%s %dv%d h=%d: the notice is %d cells wide, over the %d there "+
						"are: %q", lang, side, side, height, width, drawable, found.notice)
				}
				for _, key := range nameable {
					if found.notice != "" && strings.Contains(found.notice, m.lang.Text(key)) {
						named[key] = true
					}
				}
			}
			for _, key := range nameable {
				if !named[key] {
					t.Errorf("%s %dv%d: no height in the sweep hides what %q names, so that "+
						"wording is rendered by nothing", lang, side, side, base.lang.Text(key))
				}
			}
		}
	}
}

// TestTheBudgetSpendsWhatItSaysItDoes is the arithmetic on its own.
//
// Every other test here reads the screen, which is the right way round: a test
// that asked the view what it had decided would be measuring itself. This one is
// different in kind and is here for a reason the others cannot cover — the greedy
// walk has corners no board can reach. A window shorter than nine rows of purse
// takes the save's note, and one shorter still leaves the roster no room for its
// first unit, and both of those are below the height m.tooSmall draws a message
// at. Written out rather than left uncovered, so the walk cannot be re-ordered
// silently.
//
// The sizes are one 5-a-side board's: ten units, ten rows of board, a four-option
// list and a full log.
func TestTheBudgetSpendsWhatItSaysItDoes(t *testing.T) {
	squad := playSizes{tail: 5, board: 10, units: 10, log: 8}
	saved := playSizes{tail: 5, notes: 4, board: 10, units: 10, log: 8}
	aiming := playSizes{tail: 12, board: 10, units: 10, log: 8}
	cases := []struct {
		why   string
		room  int
		sizes playSizes
		want  playPlan
		names []i18n.Key
	}{{
		why:   "everything fits, so nothing is said",
		room:  40,
		sizes: squad,
		want:  playPlan{board: true, roster: 10, order: true, log: 8},
	}, {
		why: "one row short takes a row off the log's tail, and a shorter tail " +
			"is the section working rather than a section missing",
		room:  39,
		sizes: squad,
		want:  playPlan{board: true, roster: 10, order: true, log: 7},
	}, {
		why:   "the log goes before the order line",
		room:  32,
		sizes: squad,
		want:  playPlan{board: true, roster: 10, order: true, notice: true},
		names: []i18n.Key{i18n.PlayHiddenLog},
	}, {
		why: "the board goes before the roster's last row, and its ten rows " +
			"buy the order line and most of the log back",
		room:  29,
		sizes: squad,
		want:  playPlan{roster: 10, order: true, log: 6, notice: true},
		names: []i18n.Key{i18n.PlayHiddenBoard},
	}, {
		why:   "the aim list is reserved too, and the roster gives up rows for it",
		room:  20,
		sizes: aiming,
		want:  playPlan{roster: 3, notice: true},
		names: []i18n.Key{i18n.PlayHiddenBoard, i18n.PlayHiddenUnits,
			i18n.PlayHiddenOrder, i18n.PlayHiddenLog},
	}, {
		why: "a save's note outranks the board and everything under it, and is " +
			"given up only where the roster has nothing left either",
		room:  12,
		sizes: saved,
		want:  playPlan{roster: 2, notice: true},
		names: []i18n.Key{i18n.PlayHiddenBoard, i18n.PlayHiddenUnits,
			i18n.PlayHiddenOrder, i18n.PlayHiddenLog, i18n.PlayHiddenNote},
	}}
	for _, test := range cases {
		got := playFit(test.room, test.sizes)
		if got != test.want {
			t.Errorf("a purse of %d rows gives %+v, want %+v — %s",
				test.room, got, test.want, test.why)
		}
		if test.names != nil {
			if named := playHidden(got, test.sizes); !sameKeys(named, test.names) {
				t.Errorf("a purse of %d rows names %v, want %v — %s",
					test.room, named, test.names, test.why)
			}
		}
		// The invariant the priority states, at every row count there is: the
		// board is never drawn over a roster that is not.
		if got.board && got.roster == 0 {
			t.Errorf("a purse of %d rows draws the board with no unit under it", test.room)
		}
	}
}

func sameKeys(got, want []i18n.Key) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
