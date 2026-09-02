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
	draw "github.com/vukyn/hexarena/internal/screen"
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
	if m.play.Err != nil {
		t.Fatalf("the battle would not start: %v", m.play.Err)
	}
	return m
}

// TestTheFightRaisesABattleYouPlay is the wiring, and that the battle arrives
// already waiting on the player rather than on a key nobody knows to press.
func TestTheFightRaisesABattleYouPlay(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	if m.play.Fight == nil {
		t.Fatal("no battle was built")
	}
	if m.play.Pending == nil {
		t.Fatal("the battle opened without a turn for the player")
	}
	// The turn waiting is the player's own, which is the whole claim: every
	// other side's turn is taken on the way here.
	unit, known := m.play.Fight.Unit(m.play.Pending.Unit)
	if !known || unit.Side != m.play.Side {
		t.Fatalf("the battle is waiting on %q", m.play.Pending.Unit)
	}
	// And the cursor is on something that can actually be taken.
	if option := m.play.Pending.Options[m.play.Option]; !option.Available() {
		t.Errorf("the cursor opened on %q, which cannot be used: %s", option.Skill, option.Reason)
	}
	if back := key(t, m, "esc"); back.screen != screenFight {
		t.Errorf("esc left the battle for %v", back.screen)
	}
}

// TestAWayBackSurvivesTheScreenItRaised is the whole of what model.raisedOver is
// for, and it is the path no other test walks.
//
// A screen that was raised and then raises something itself has **two** doors
// open at once: its own, and the one it just opened. The battle is the first
// screen in this client with both — the fight opens it, and `?` opens a
// description over it — so a way back kept in one slot is a way back the raise
// overwrites, and the reader is quietly returned somewhere they have never been.
//
// ⚠️ **Depth one cannot see this, and depth one is what every other test walks.**
// TestTheFightRaisesABattleYouPlay above presses `fight → p → esc` and lands on
// the fight from a single slot, perfectly correctly; the defect needs the raise
// **in between**. Measured: collapsing the push to an assignment
// (`m.raisedOver, m.raisedFrom = m.raisedFrom, from` → `m.raisedFrom = from`)
// leaves the entire client suite green except this test.
//
// The two walks share one battle, which is safe for exactly the four keystrokes
// below: `?` reads the option under the cursor and `esc` leaves a screen, so
// neither steps the pointer the model does not copy.
func TestAWayBackSurvivesTheScreenItRaised(t *testing.T) {
	m, _, _ := start(t, i18n.En)
	m = atTheBattle(t, m)
	described := typeText(t, m, "?")
	if described.screen != screenBlurb {
		t.Fatalf("? opened %v rather than the description", described.screen)
	}
	returned := key(t, described, "esc")
	if returned.screen != screenPlay {
		t.Fatalf("esc left the description for %v rather than the battle it was raised "+
			"from", returned.screen)
	}
	if left := key(t, returned, "esc"); left.screen != screenFight {
		t.Errorf("after a description had been read, esc left the battle for %v — want "+
			"the fight that raised it, which is the door the raise displaced",
			left.screen)
	}
	// And the same key with nothing raised over the battle, so a failure above is
	// about the raise rather than about esc.
	if plain := key(t, m, "esc"); plain.screen != screenFight {
		t.Errorf("esc left the battle for %v with nothing raised over it", plain.screen)
	}
	// ⚠️ **And the rest of the way out, which is what makes two slots enough
	// rather than merely today's number.** This chain is catalogue → fight →
	// battle → description, which is **three** pushes and not two, so the third
	// one displaces the catalogue and nothing puts it back. It is sound only
	// because the fight answers esc by naming the catalogue **itself** instead
	// of following a way back — see model.raisedOver. Walking to the bottom is
	// what measures that: the day the fight's esc becomes a draw.Back, this leg
	// lands on the menu and two slots stop being enough.
	out := key(t, key(t, returned, "esc"), "esc")
	if out.screen != screenSquads {
		t.Errorf("leaving the fight after a battle and a description landed on %v, want "+
			"the catalogue — the chain is three raises deep and two slots hold it only "+
			"while the fight names its own door", out.screen)
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
	column := draw.PlayMarkerWidth + m.play.OptionWidth()
	if len(letters) < column+draw.PlayOptionGap {
		return "", ""
	}
	return strings.TrimRight(string(letters[draw.PlayMarkerWidth:column]), " "),
		string(letters[column+draw.PlayOptionGap:])
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
			theClippedRow(t, lang, drawable, aShippedBook, maySkip)
			measured++
		})
	}
	// ⚠️ **Reported rather than failed, because the clip is measured below
	// whatever the book says.** This was an error, and it fired the day the floor
	// moved to 120: the row went from 62 cells to 102 and neither language holds a
	// summary that long, so both skipped and the clip was drawn by nothing. The
	// error was right about the consequence and there is nothing a wording change
	// could do about the cause, so the coverage moved into a constructed case and
	// this line went back to being what it always described — a reading of how
	// close the book is to the row.
	t.Logf("languages whose own book reaches the clip: %d of %d", measured, len(i18n.Langs()))

	// And the constructed case, which clips at any floor.
	for _, lang := range i18n.Langs() {
		t.Run("a long id/"+lang.String(), func(t *testing.T) {
			theClippedRow(t, lang, drawable, aBookWithALongID, mustClip)
		})
	}
}

// The two ways this test gets a library, and whether a book that does not clip is
// an honest answer or a broken fixture.
const (
	maySkip  = true
	mustClip = false
)

// aShippedBook is the library as it ships, which is the reading worth taking:
// how near the widest summary the book actually holds comes to the row.
func aShippedBook(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	m, _, _ := start(t, lang)
	return m
}

// aBookWithALongID is the same library plus one skill whose id is long enough
// that the row it shares is narrower than the widest summary in the book.
//
// ⚠️ **The id is the dial rather than the summary, and it is the documented
// one**: widestIDInTheBook is what the row's room is measured from, and its own
// comment says a longer id in the book moves the room rather than the budget.
// Authoring a wordier *skill* would instead be authoring a summary the guard
// above — TestNoSummaryIsWiderThanARowCanHold — exists to refuse, so the two
// tests would be pulling in opposite directions.
//
// The length is derived from the widest summary the book holds, so the room
// lands under it by a stated margin rather than by a number that happens to work
// at this floor.
func aBookWithALongID(t *testing.T, lang i18n.Lang) model {
	t.Helper()
	const clearlyInside = 10
	plain, _, _ := start(t, lang)
	widest := lipgloss.Width(
		plain.lang.SummariseSkill(theWidestSummary(plain), plain.lib.Patterns()))

	dir := scratchData(t)
	length := minWidth - 1 - draw.PlayMarkerWidth - draw.PlayOptionGap - (widest - clearlyInside)
	if length < 1 {
		t.Fatalf("%s: the widest summary is %d cells, so no id shrinks the row under it",
			lang, widest)
	}
	appendSkills(t, dir, []string{strings.Repeat("i", length)}, nil)
	m, _, _ := startIn(t, lang, dir)
	return m
}

func theClippedRow(t *testing.T, lang i18n.Lang, drawable int,
	book func(*testing.T, i18n.Lang) model, mayNotClip bool) {
	t.Helper()
	m := book(t, lang)
	m = atTheBattle(t, m)
	wordiest, longest := theWidestSummary(m), theWidestID(m)
	room := drawable - draw.PlayMarkerWidth - lipgloss.Width(longest.ID) - draw.PlayOptionGap
	summary := m.lang.SummariseSkill(wordiest, m.lib.Patterns())
	if lipgloss.Width(summary) <= room {
		if !mayNotClip {
			t.Fatalf("%s: the constructed book does not clip either — the widest summary "+
				"is %q at %d cells and the row has %d, so the fixture is not building the "+
				"case it is named after", lang, wordiest.ID, lipgloss.Width(summary), room)
		}
		// Honest rather than contrived: no skill in the book is long enough to be
		// cut at the widest id column in this language, so there is no clip to
		// measure.
		t.Skipf("%s: the widest summary is %q at %d cells and the row has %d, "+
			"so nothing in the book clips", lang, wordiest.ID,
			lipgloss.Width(summary), room)
	}
	m.play.Pending = &battle.Prompt{
		Unit: m.play.Pending.Unit, Turn: m.play.Pending.Turn,
		Options: []battle.Option{
			{Skill: wordiest.ID, Aims: m.play.Pending.Options[0].Aims},
			{Skill: longest.ID, Aims: m.play.Pending.Options[0].Aims},
		},
	}
	m.play.Option = 0
	rows := strings.Split(strings.TrimRight(m.play.Choices(m.ctx()), "\n"), "\n")
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
		fought := len(m.play.Script)
		option := m.play.Pending.Options[m.play.Option]
		raised := typeText(t, m, "?")
		if raised.screen != screenBlurb || raised.raisedFrom != screenPlay {
			t.Fatalf("%s: ? opened %v from %v", lang, raised.screen, raised.raisedFrom)
		}
		// ⚠️ The battle is a pointer, so raising and leaving this must step no
		// turn: view reads the option and nothing else.
		if len(raised.play.Script) != fought || raised.play.Fight.Finished() {
			t.Errorf("%s: raising the description spent %d decisions",
				lang, len(raised.play.Script)-fought)
		}
		body := raised.screenContent()
		if !strings.Contains(body, option.Skill) &&
			!strings.Contains(body, raised.lang.GlossedSkill(
				mustSkill(t, raised, option.Skill))) {
			t.Errorf("%s: the description does not name %q:\n%s", lang, option.Skill, body)
		}
		// The same sentences the listing draws, rather than a second rendering.
		for _, line := range draw.SkillLines(raised.ctx(), mustSkill(t, raised, option.Skill)) {
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
		if len(back.play.Script) != fought || back.play.Seed != m.play.Seed {
			t.Errorf("%s: coming back rebuilt the battle: seed %d, %d decisions",
				lang, back.play.Seed, len(back.play.Script))
		}
		// ↑/↓ walks the option behind, so four of them can be read one after
		// another.
		walked := key(t, raised, "down")
		if len(m.play.Pending.Options) > 1 && walked.play.Option == raised.play.Option {
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
	aiming.play.Aiming = true
	raised := typeText(t, aiming, "?")
	if raised.screen != screenBlurb || raised.raisedFrom != screenPlay {
		t.Fatalf("? while aiming opened %v from %v", raised.screen, raised.raisedFrom)
	}
	if !strings.Contains(raised.screenContent(),
		aiming.play.Pending.Options[aiming.play.Option].Skill) &&
		strings.TrimSpace(raised.screenContent()) == "" {
		t.Error("the description while aiming is empty")
	}
	// And it leaves the aim where it was: coming back has to land on the second
	// question rather than on the first.
	if back := key(t, raised, "esc"); !back.play.Aiming {
		t.Error("coming back from the description dropped the aim")
	}
	// ↑/↓ does nothing here: the skill is settled, and walking the options would
	// change what is described out from under a half-taken decision.
	if walked := key(t, raised, "down"); walked.play.Option != raised.play.Option {
		t.Error("the description walked the option list while a skill was already chosen")
	}

	// Nothing pending is a key that does nothing.
	over := m
	for range draw.PlayTurnLimit {
		if over.play.Fight.Finished() || over.play.Err != nil {
			break
		}
		over = typeText(t, over, "a")
	}
	if !over.play.Fight.Finished() {
		t.Fatal("the battle never ended, so there is no promptless state to test")
	}
	if quiet := typeText(t, over, "?"); quiet.screen != screenPlay {
		t.Errorf("? on a finished battle opened %v", quiet.screen)
	}
}

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
		if m.play.Fight.Finished() {
			break
		}
		m = typeText(t, m, "a")
	}
	m = key(t, m, "ctrl+s")
	if m.play.Err != nil {
		t.Fatalf("the save was refused: %v", m.play.Err)
	}
	if len(m.play.Notes) == 0 {
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
	if log.Seed != m.play.Seed || len(log.Choices) != len(m.play.Script) {
		t.Errorf("the log holds seed %d and %d choices, want %d and %d",
			log.Seed, len(log.Choices), m.play.Seed, len(m.play.Script))
	}

	rerun, err := battle.New(lib.Books(), log.Seed, log.Roster)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	rerun.Begin()
	if _, _, err := rerun.Replay(log.Choices, draw.PlayTurnLimit, nil); err != nil {
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
	m.squad.Editing.ID = "../../escaped"
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	m = typeText(t, m, "f")
	m = typeText(t, m, "p")
	m = key(t, m, "ctrl+s")
	if m.play.Err != nil {
		t.Fatalf("the save was refused: %v", m.play.Err)
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
	m.squad.Editing.ID = id
	m.squad.IDInput.SetValue(id)
	characters := m.squad.Characters
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
	m.squad.Editing.Units = units
	m = withASquadSaved(t, m)
	m = key(t, m, "esc")
	// Point the catalogue at the squad just saved and both of the fight's sides
	// at it: the cursor because the catalogue may already hold somebody else's
	// squad, and the away chooser because it opens on the first squad on the
	// list rather than on the one under the cursor. A squad against a copy of
	// itself is the one pairing any catalogue is guaranteed to be able to field.
	m.squad.Cursor = squadIndex(t, m, id)
	m = typeText(t, m, "f")
	m.fight.away = m.fight.home
	m = typeText(t, m, "p")
	if m.screen != screenPlay {
		t.Fatalf("p opened %v rather than the battle", m.screen)
	}
	if m.play.Err != nil {
		t.Fatalf("a %d-a-side battle would not start: %v", side, m.play.Err)
	}
	if m.play.Fight == nil {
		t.Fatalf("a %d-a-side battle built nothing", side)
	}
	if want := side * 2; len(m.play.Fight.Units()) != want {
		t.Fatalf("a %d-a-side pairing fielded %d units, want %d",
			side, len(m.play.Fight.Units()), want)
	}
	return m
}

// squadIndex is where a squad of a given id sits in the catalogue.
func squadIndex(t *testing.T, m model, id string) int {
	t.Helper()
	for index, squad := range m.squad.Saved {
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
// it can. Measured at 120x24, where draw.PlayBodyRoom leaves the body twenty rows: the
// heading is one, tui.Board is a fixed ten, tui.Roster is one plus a row a unit,
// tui.Order is one, the log asks for draw.PlayLogWanted and the option list is one plus
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

// withAFullLog plays the engine's own turns until the history has grown past the
// rows the log asks for.
//
// ⚠️ **Without it the log is never the section that goes.** The opening board's
// log is one row a unit entering, so a 1v1 at the floor still holds all four of
// them and the section the priority drops first is never dropped — a sweep over
// the opening turn alone would report the log's place in the order as unmeasured
// while looking like it had measured it.
func withAFullLog(t *testing.T, m model) model {
	t.Helper()
	for range 20 {
		if len(m.play.LogRows(m.ctx())) >= draw.PlayLogWanted || m.play.Fight.Finished() {
			break
		}
		m = typeText(t, m, "a")
	}
	if rows := len(m.play.LogRows(m.ctx())); rows < draw.PlayLogWanted {
		t.Fatalf("the history came to %d rows of %d, so a squeezed log was not measured",
			rows, draw.PlayLogWanted)
	}
	if m.play.Pending == nil {
		t.Fatal("the battle stopped waiting on the player, so no option list is drawn")
	}
	return m
}

// withALongLog is withAFullLog played on far enough that the history runs past
// the frame at **every** height the sweep draws, which is what a scroll has to be
// measured against.
//
// ⚠️ **It has to be constructed and it is not the same fixture.** withAFullLog
// stops at the eight rows the log asks for, and eight rows fit the frame in a tall
// window — so on that fixture there is nothing above the frame, the keys correctly
// do nothing and every assertion about scrolling passes without exercising it.
// This fails rather than measuring nothing if the battle ends first.
func withALongLog(t *testing.T, m model, rows int) model {
	t.Helper()
	for range draw.PlayTurnLimit {
		if len(m.play.LogRows(m.ctx())) >= rows || m.play.Fight.Finished() ||
			m.play.Err != nil {
			break
		}
		m = typeText(t, m, "a")
	}
	if m.play.Err != nil {
		t.Fatalf("playing the battle out broke: %v", m.play.Err)
	}
	if got := len(m.play.LogRows(m.ctx())); got < rows {
		t.Fatalf("the battle finished with a history of %d rows against the %d this "+
			"needs, so nothing above the frame was constructed", got, rows)
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
	body, footer := m.play.View(m.ctx())
	lines := strings.Split(body, "\n")
	present := make(map[string]int, len(lines))
	for _, line := range lines {
		present[line]++
	}
	p := m.play
	var found drawn
	board := strings.Split(tui.Board(p.Fight, p.Tags), "\n")
	found.board = present[board[0]] > 0 && present[board[len(board)-1]] > 0
	// The header is not a unit, so the count starts past it.
	for _, row := range strings.Split(tui.Roster(p.Fight, p.Tags), "\n")[1:] {
		if present[row] > 0 {
			found.roster++
		}
	}
	found.order = present[m.style.Dim.Render(tui.Order(p.Fight.Queue(), p.Tags, 6))] > 0
	// Counted as a multiset and not by lookup: the log's rows are not distinct —
	// tui.Line opens a turn with a blank one — so asking whether each is on the
	// screen counts every blank row once per blank row there is. Over the **whole
	// history** rather than over the frame, because the frame is what is being
	// measured: counting the rows the screen decided to draw would be the test
	// asking the view what it had decided.
	for _, row := range p.LogRows(m.ctx()) {
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
		if len(line) > draw.PlayMarkerWidth && strings.HasPrefix(line[draw.PlayMarkerWidth:], id) {
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
			options := base.play.Pending.Options
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
					if index == m.play.Option && !strings.HasPrefix(row, "> ") {
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
			aiming.play.Aiming = true
			states["aiming"] = aiming
			states["saved"] = key(t, base, "ctrl+s")
			over, _, _ := start(t, lang)
			over = atABattleOf(t, over, side)
			for range draw.PlayTurnLimit {
				if over.play.Fight.Finished() || over.play.Err != nil {
					break
				}
				over = typeText(t, over, "a")
			}
			if !over.play.Fight.Finished() {
				t.Fatalf("%s %dv%d: the battle never ended, so the ending was not measured",
					lang, side, side)
			}
			states["over"] = over
			for name, state := range states {
				for _, height := range playHeights {
					m := state
					m.width, m.height = minWidth, height
					body, _ := m.play.View(m.ctx())
					if rows := len(strings.Split(body, "\n")); rows > draw.PlayBodyRoom(height) {
						t.Errorf("%s %dv%d %s h=%d: the body is %d rows against the %d it has",
							lang, side, side, name, height, rows, draw.PlayBodyRoom(height))
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

// # The log as a frame over the whole history
//
// Two defects, and they were separate. The section's allotment was clamped to the
// rows it asked for, so the body grew twenty rows to forty-two between a 120x24
// window and an 80x80 one and the log stood still at eight — a tall terminal
// bought the history nothing. And the rows past the frame were unreachable: the
// history came to three hundred rows, eight were drawn, and there was no key.
//
// aLongLog is the fixture the whole block below rests on. ⚠️ It has to be
// constructed: withAFullLog stops at the eight rows the section asks for, and eight
// rows fit the frame in any tall window — so on that fixture nothing is above the
// frame, every scroll key correctly does nothing, and every assertion here would
// pass without exercising a single line of it.
func aLongLog(t *testing.T, lang i18n.Lang, side int) model {
	t.Helper()
	base, _, _ := start(t, lang)
	base = withALongLog(t, atABattleOf(t, base, side), longLogRows)
	base.width, base.height = minWidth, longLogHeight
	return base
}

// longLogRows is how long "longer than any window draws" is. The body of an 80x48
// window is 44 rows, all of which the log could in principle be given, so this is
// comfortably past the largest frame the sweep asks for.
//
// longLogHeight is the window the fixture stands in, and it is not the floor: at
// 120x24 a three-a-side battle is given **no** log row at all — which is the budget
// working, and a fixture standing there would be pressing scroll keys at a section
// that is not on the screen.
const (
	longLogRows   = 120
	longLogHeight = 40
)

// TestTheSaveNoteOutranksTheBoard is where a write's own answer sits in the
// priority, and the arithmetic that says the other half of it cannot be reached.
//
// A save's notes are the answer to a keystroke pressed a moment ago and they name
// the file that was written, so they take rows before the board does. They are
// **not** reserved: a pair of notes runs to four rows or more, and reserving them
// could crowd out the option list, which may never be cut.
//
// ⚠️ **"Five rows" is what this said, and it was a number with a floor baked into
// it.** The second note is catalog wording wrapped at minWidth — three rows at a
// floor of 80 and two at 120 — so the pair came to five and now comes to four,
// with nothing here having changed. The first note carries the path it wrote to,
// which is free text and as long as the data directory is, so the total was never
// a constant in the first place. The catalog half is pinned by
// TestEveryFloorWrappedBlockTakesTheRowsItTakes; this comment stops quoting a
// total.
//
// ⚠️ **The branch that drops them is unreachable at any window the tool draws, so
// it is built at one the tool refuses.** At the floor the body has twenty rows,
// the heading and the option list reserve seven and the notice one, so twelve are
// left against the four or five the notes want — and even the aim list, the
// tallest tail a squad can field, leaves enough. A shorter window is what takes
// them, and m.tooSmall draws a message instead of this screen there, so the case
// is constructed rather than left to be rendered by nothing.
func TestTheSaveNoteOutranksTheBoard(t *testing.T) {
	const drawable = minWidth - 1
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		m = atABattleOf(t, m, hex.MaxTeamSize)
		m = key(t, m, "ctrl+s")
		if m.play.Err != nil {
			t.Fatalf("%s: the save failed, so no note was measured: %v", lang, m.play.Err)
		}
		notes := m.play.Wrote(m.ctx())
		if len(notes) == 0 {
			t.Fatalf("%s: the save left no note behind", lang)
		}
		for _, height := range playHeights {
			state := m
			state.width, state.height = minWidth, height
			body, _ := state.play.View(state.ctx())
			if !strings.Contains(body, notes[0]) {
				t.Errorf("%s h=%d: the save's own note is not on the screen:\n%s",
					lang, height, body)
			}
		}
		// The window that does take it, below the floor the tool draws at.
		short := m
		short.width, short.height = minWidth, minHeight-9
		body, _ := short.play.View(short.ctx())
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
