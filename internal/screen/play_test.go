package screen

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// # The played battle
//
// The screen a game client needs most, and the last of the big ones to move. What
// is asserted here is what it **draws** and how its two cursors and its log frame
// move; where a key lands — the fight it escapes to, the description it raises —
// is asserted in cmd/hexforge-tui, driven through the real model, because a
// landing is a fact about a client's own screens standing next to each other.
//
// ⚠️ **Every state below builds its own battle, and that is not tidiness.**
// PlayScreen holds a `*battle.Battle`, which is the one thing on this model no
// copy copies: `state := p` shares the pointer, so playing one of them out steps
// the battle every other copy points at. That fixture has shipped twice in this
// repository — everyScreen's `finished` was a copy of `battle` driven to its end,
// so by the time anything drew `battle` its own `Finished()` was true, `View`
// returned at the game-over branch, and both footers were over the window with
// every width test passing. A package fixture that handed several states one
// battle would make the same mistake in a second place.
//
// What that costs is a battle built per state, which is why atABattleOf is a
// function rather than a table.

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

// aSquadOfSide is a squad of n units built by **looking the cast up** rather than
// by naming characters, which is the rule every fixture in this package follows:
// a test that names a character breaks the day somebody edits the cast for a
// reason that has nothing to do with it.
//
// ⚠️ **The characters repeat when the fixture cast is smaller than the squad, and
// that is deliberate.** What the height tests measure is **rows**, one a unit, so
// ten distinct characters would be ten readings of the same number. This is a
// count of rows and not a roster anybody would field.
func aSquadOfSide(t *testing.T, c Context, side int) placement.Squad {
	t.Helper()
	if side < 1 || side > hex.MaxTeamSize {
		t.Fatalf("a side of %d is outside the %d a squad may field", side, hex.MaxTeamSize)
	}
	characters := c.Lib.Characters().All()
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
		kit := character.SkillsAt(unit.Level, progression.Furthest)
		if len(kit) > cast.SkillSlots {
			kit = kit[:cast.SkillSlots]
		}
		unit.Skills = kit
		if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
			unit.Passives = traits[:cast.TraitSlots]
		}
		units = append(units, unit)
	}
	return placement.Squad{
		ID:    "do-thu-" + strconv.Itoa(side),
		Name:  "đội thử",
		Units: units,
	}
}

// atABattleOf opens the screen on a squad of n against a copy of itself, which is
// the one pairing any cast is guaranteed to be able to field.
//
// ⚠️ **It builds a fresh battle every call**, for the reason the file comment
// gives. Nothing here is saved to disk and nothing reads squads.json: the pairing
// is handed to Open as a value, which is the whole point of the two squads being
// a parameter.
func atABattleOf(t *testing.T, c Context, side int) PlayScreen {
	t.Helper()
	squad := aSquadOfSide(t, c, side)
	p := NewPlayScreen().Open(c, squad, squad.Clone())
	if p.Err != nil {
		t.Fatalf("a %d-a-side battle would not start: %v", side, p.Err)
	}
	if p.Fight == nil {
		t.Fatalf("a %d-a-side battle built nothing", side)
	}
	if want := side * 2; len(p.Fight.Units()) != want {
		t.Fatalf("a %d-a-side pairing fielded %d units, want %d",
			side, len(p.Fight.Units()), want)
	}
	if p.Pending == nil {
		t.Fatalf("a %d-a-side battle opened without a turn for the player", side)
	}
	return p
}

// atTheBattle is the one-a-side battle every test that does not care about the
// squad size starts from.
func atTheBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	return atABattleOf(t, c, 1)
}

// playing presses keys through the real Update and hands back the screen,
// dropping the actions.
//
// A key at a time, which is what a keyboard does, and every one of them goes
// through the switch a reader's keystroke goes through.
func playing(t *testing.T, c Context, p PlayScreen, keys ...string) PlayScreen {
	t.Helper()
	for _, name := range keys {
		p, _ = p.Update(c, press(t, name))
	}
	return p
}

// asking is one keystroke with the action it produced, for the handful of tests
// that are about what the screen asked for rather than about what it became.
func asking(t *testing.T, c Context, p PlayScreen, name string) (PlayScreen, Action) {
	t.Helper()
	return p.Update(c, press(t, name))
}

// inAWindow is the same Context at a height, which is what the budget spends.
func inAWindow(c Context, height int) Context {
	c.Width, c.Height = MinWidth, height
	return c
}

// TestATurnTakenMovesTheBattleOn is the loop: the player acts, the engine
// answers, and the next thing waiting is the player again.
func TestATurnTakenMovesTheBattleOn(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := atTheBattle(t, c)
	before, events := len(p.Script), len(p.Events)
	p = playing(t, c, p, "enter")
	if len(p.Script) <= before {
		t.Fatal("nothing was written down")
	}
	if len(p.Events) <= events {
		t.Error("a turn was taken and the battle recorded nothing")
	}
	// The opponent answered on the way back, so the script grew by more than the
	// one decision the player made.
	if len(p.Script) < before+2 && !p.Fight.Finished() {
		t.Errorf("the script holds %d decisions, want the player's and the engine's",
			len(p.Script))
	}
	if p.Pending == nil && !p.Fight.Finished() {
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
	c, _ := start(t, i18n.En)
	p := atTheBattle(t, c)
	opening := p.Script
	p = playing(t, c, p, "enter")
	if len(p.Script) <= len(opening) {
		t.Fatal("nothing to take back")
	}
	fought := len(p.Script)
	p = playing(t, c, p, "u")
	if len(p.Script) >= fought {
		t.Fatalf("undo left %d decisions of %d", len(p.Script), fought)
	}
	// What is left ends with somebody else's turn: undo cuts at the player's
	// last decision, so nothing of theirs survives it.
	for _, decision := range p.Script {
		unit, known := p.Fight.Unit(decision.Unit)
		if known && unit.Side == p.Side {
			t.Errorf("a turn of the player's own survived the undo: %+v", decision)
		}
	}
	// And the battle is waiting on the player again rather than on nobody.
	if p.Pending == nil && !p.Fight.Finished() {
		t.Error("after an undo the battle waits on nothing")
	}
	// Nothing to take back is not an error, it is a key that does nothing. Its
	// own battle, because this one has been played.
	fresh := playing(t, c, atTheBattle(t, c), "u")
	if fresh.Err != nil {
		t.Errorf("undo with nothing of the player's in the script reported %v", fresh.Err)
	}
}

// TestAnotherSeedIsAnotherBattle is what a player asks for when a pairing has
// been played once.
func TestAnotherSeedIsAnotherBattle(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := playing(t, c, atTheBattle(t, c), "enter")
	seed, fought := p.Seed, len(p.Script)
	p = playing(t, c, p, "n")
	if p.Seed != seed+1 {
		t.Errorf("n moved the seed to %d, want %d", p.Seed, seed+1)
	}
	if len(p.Script) >= fought {
		t.Errorf("the new battle kept %d decisions from the old one", len(p.Script))
	}
	if p.Err != nil {
		t.Errorf("the new battle would not start: %v", p.Err)
	}
	// And it is the same pairing: another seed is another arrangement of two
	// squads that were handed in once.
	if len(p.Fight.Units()) != 2 {
		t.Errorf("the new battle fielded %d units", len(p.Fight.Units()))
	}
}

// TestAimingIsAskedOnlyWhenItIsADecision is the second question: a skill with
// one legal cell does not ask it, because a question with one answer is not a
// decision.
func TestAimingIsAskedOnlyWhenItIsADecision(t *testing.T) {
	c, _ := start(t, i18n.En)
	// Two a side, so an enemy-aimed skill has two cells to choose between.
	p := atABattleOf(t, c, 2)
	found := false
	for index, option := range p.Pending.Options {
		if option.Available() && len(option.Aims) > 1 {
			p.Option = index
			found = true
			break
		}
	}
	if !found {
		t.Skip("no skill on the opening turn has two cells to choose between")
	}
	p = playing(t, c, p, "enter")
	if !p.Aiming {
		t.Fatal("a skill with two cells acted without asking where")
	}
	body, _ := p.View(c)
	if !strings.Contains(body, strings.Fields(c.Text(i18n.PlayAimAt, "x"))[0]) {
		t.Errorf("the aim list is not drawn:\n%s", body)
	}
	// esc backs out of the second question without spending the turn, and asks
	// for nothing of the client: the way out of the aim list is inside the screen.
	before := len(p.Script)
	backed, action := asking(t, c, p, "esc")
	if backed.Aiming {
		t.Error("esc did not leave the aim list")
	}
	if action != (Action{}) {
		t.Errorf("esc out of an aim asked for %+v, want nothing", action)
	}
	if len(backed.Script) != before {
		t.Error("backing out of an aim spent the turn")
	}
	// And once the aim is gone, esc is the way out of the screen.
	if _, action := asking(t, c, backed, "esc"); action.Kind != Back {
		t.Errorf("esc on a battle asked for %v, want a Back", action.Kind)
	}
}

// TestABattlePlayedOutEndsAndSaysHow is the far end of the screen, driven by
// the key that hands each turn to the engine.
func TestABattlePlayedOutEndsAndSaysHow(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := playedOut(t, c, atTheBattle(t, c))
	// It says which of the four endings it was, in words rather than in a code.
	body, _ := p.View(c)
	said := false
	for _, ending := range []i18n.Key{i18n.PlayWon, i18n.PlayLost, i18n.PlayDrawn, i18n.PlayEmptied} {
		if strings.Contains(body, c.Text(ending)) {
			said = true
		}
	}
	if !said {
		t.Errorf("the battle ended and the screen does not say how:\n%s", body)
	}
	// The script is the whole battle, both sides in it, so what was played can
	// be replayed.
	sides := map[hex.Side]int{}
	for _, decision := range p.Script {
		if unit, known := p.Fight.Unit(decision.Unit); known {
			sides[unit.Side]++
		}
	}
	if sides[hex.SideAlly] == 0 || sides[hex.SideEnemy] == 0 {
		t.Errorf("the script holds %v, want turns from both sides", sides)
	}
}

// playedOut hands every turn to the engine until the battle finishes, and fails
// rather than handing back a battle still running.
func playedOut(t *testing.T, c Context, p PlayScreen) PlayScreen {
	t.Helper()
	for range PlayTurnLimit {
		if p.Fight.Finished() || p.Err != nil {
			break
		}
		p = playing(t, c, p, "a")
	}
	if p.Err != nil {
		t.Fatalf("the battle broke: %v", p.Err)
	}
	if !p.Fight.Finished() {
		t.Fatal("the battle never ended")
	}
	return p
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
		c, _ := start(t, lang)
		p := atTheBattle(t, c)
		options := p.Pending.Options
		if len(options) == 0 {
			t.Fatalf("%s: the opening turn offers nothing to describe", lang)
		}
		for _, option := range options {
			// The wording is i18n's own, so this asks for that string rather than
			// for a clause of its own: a test naming one here would be the wording
			// living in two places, which is what the AST scan refuses.
			summary := c.Lang.SummariseSkill(
				mustSkill(t, c, option.Skill), c.Lib.Patterns())
			if strings.TrimSpace(summary) == "" {
				t.Errorf("%s: %q summarises as nothing", lang, option.Skill)
			}
		}
	}
}

// mustSkill is one skill out of the library in hand.
func mustSkill(t *testing.T, c Context, id string) skill.Skill {
	t.Helper()
	declared, err := c.Lib.Skills().Lookup(id)
	if err != nil {
		t.Fatalf("the book does not hold %q: %v", id, err)
	}
	return declared
}

// TestAnOptionRowCarriesAsMuchOfItsSummaryAsItHasRoomFor is the second half: the
// row.
//
// Three things, and the row count is the one that matters most. The screen is
// already eight rows past the window it declares, so a pane under the list would
// have been a pane nobody in a 120x24 terminal ever sees — which is why the answer
// is a line beside each option.
//
// The other two are what clipping actually promises. The slot holds a **prefix**
// of the summary, and the longest prefix the room allows: shorter would be the row
// throwing away cells it has, longer would be the row running past the window. It
// is read out of the row at the offset the constants give rather than searched
// for, because a clipped summary has no index to find.
func TestAnOptionRowCarriesAsMuchOfItsSummaryAsItHasRoomFor(t *testing.T) {
	const drawable = MinWidth - 1
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		p := atTheBattle(t, c)
		options := p.Pending.Options
		if len(options) == 0 {
			t.Fatalf("%s: the opening turn offers nothing to draw", lang)
		}
		drawn := p.Choices(c)
		rows := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
		// One heading and one row an option: exactly what it was before the
		// summary arrived.
		if len(rows) != len(options)+1 {
			t.Fatalf("%s: %d options draw %d rows, want one each under one heading:\n%s",
				lang, len(options), len(rows), drawn)
		}
		room := drawable - PlayMarkerWidth - p.OptionWidth() - PlayOptionGap
		for index, option := range options {
			row := rows[index+1]
			if width := lipgloss.Width(row); width > drawable {
				t.Errorf("%s: row %d is %d cells over the %d it has:\n%s",
					lang, index, width, drawable, row)
			}
			// ⚠️ **The marker, on the row under the cursor and on no other.**
			// It was held by the client alone until this screen moved: measured
			// under mutation, drawing it a row late (`index == p.Option+1`)
			// reddened both goldens and cmd/hexforge-tui's option-list sweep and
			// **nothing in this package** — the rest of this loop reads the id
			// column and the slot after it, both of which are unchanged by a
			// cursor pointing at the wrong row. A golden is a fine net and a
			// screen this package owns should not depend on another one's.
			if marked := strings.HasPrefix(row, "> "); marked != (index == p.Option) {
				t.Errorf("%s: row %d draws %q, and the cursor is on row %d",
					lang, index, row, p.Option)
			}
			// The id sits in the measured column, which is what makes every slot
			// after it start in the same place.
			named, tail := optionColumns(p, row)
			if named != option.Skill {
				t.Errorf("%s: row %d holds %q in the id column, want %q:\n%s",
					lang, index, named, option.Skill, row)
			}
			summary := c.Lang.SummariseSkill(
				mustSkill(t, c, option.Skill), c.Lib.Patterns())
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
func optionColumns(p PlayScreen, row string) (named, tail string) {
	letters := []rune(row)
	column := PlayMarkerWidth + p.OptionWidth()
	if len(letters) < column+PlayOptionGap {
		return "", ""
	}
	return strings.TrimRight(string(letters[PlayMarkerWidth:column]), " "),
		string(letters[column+PlayOptionGap:])
}

// widestIDInTheBook is the id column at its worst: the widest skill id anything
// may bring, so MinWidth - 1 less the marker, this and the gap is the least room a
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
	room := MinWidth - 1 - PlayMarkerWidth - widestIDInTheBook - PlayOptionGap
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		skills := c.Lib.Skills().Skills()
		if len(skills) == 0 {
			t.Fatalf("%s: the library holds no skills, so nothing was measured", lang)
		}
		worst, worstAt := 0, ""
		for _, declared := range skills {
			if width := lipgloss.Width(
				c.Lang.SummariseSkill(declared, c.Lib.Patterns())); width > worst {
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
	c, _ := start(t, i18n.En)
	p := atTheBattle(t, c)
	// Something has to go on cooldown first: every option is available on the
	// turn a battle opens, which is exactly why this is not covered by the tests
	// above.
	found := -1
	for range 40 {
		if p.Pending == nil || p.Err != nil {
			break
		}
		for index, option := range p.Pending.Options {
			if !option.Available() && option.Reason != "" {
				found = index
			}
		}
		if found >= 0 {
			break
		}
		p = playing(t, c, p, "a")
	}
	if found < 0 {
		t.Skip("nothing came off cooldown in forty turns, so no option is refused")
	}
	option := p.Pending.Options[found]
	rows := strings.Split(strings.TrimRight(p.Choices(c), "\n"), "\n")
	_, tail := optionColumns(p, rows[found+1])
	if tail == "" || !strings.HasPrefix(option.Reason, tail) {
		t.Errorf("the refused option draws %q beside it, which is no part of its "+
			"reason %q", tail, option.Reason)
	}
	summary := c.Lang.SummariseSkill(mustSkill(t, c, option.Skill), c.Lib.Patterns())
	if strings.TrimSpace(summary) == "" {
		t.Fatalf("%q summarises as nothing, so this proves nothing", option.Skill)
	}
	if strings.HasPrefix(summary, tail) {
		t.Errorf("the refused option draws %q, which is its summary rather than "+
			"its reason %q", tail, option.Reason)
	}
}

// TestTheQuestionMarkRaisesTheOptionUnderTheCursor is the half of that key this
// package can hold: what the screen **asks for**.
//
// Where the raise lands is the client's, and is asserted there — a describer is
// one of that client's own screens. What is asserted here is the Action, and in
// particular the Subject, because a raise carrying the wrong row is a keystroke
// that did everything right and answered a different question.
//
// ⚠️ **The cursor is walked first, so an off-by-one has somewhere to show.** A
// raise measured on the opening cursor names the first option, and `At: 1` is
// what an index of nought, an index of one and a hardcoded first row all produce.
func TestTheQuestionMarkRaisesTheOptionUnderTheCursor(t *testing.T) {
	c, _ := start(t, i18n.En)
	p := atTheBattle(t, c)
	if len(p.Pending.Options) < 3 {
		t.Fatalf("the opening turn offers %d options, so a walked cursor cannot be "+
			"told from the first row", len(p.Pending.Options))
	}
	walked := playing(t, c, p, "down", "down")
	if walked.Option == p.Option {
		t.Fatal("the cursor did not move, so the subject is read at the row it opened on")
	}
	raised, action := asking(t, c, walked, "?")
	if action.Kind != Raise || action.Target != Blurb {
		t.Fatalf("? asked for %v at %v, want a raise at the description", action.Kind, action.Target)
	}
	want := Subject{
		Kind: SkillSubject,
		ID:   walked.Pending.Options[walked.Option].Skill,
		At:   walked.Option + 1,
		Of:   len(walked.Pending.Options),
	}
	if action.Subject != want {
		t.Errorf("? raised %+v, want %+v", action.Subject, want)
	}
	// ⚠️ The battle is a pointer, so asking about an option must step no turn.
	if len(raised.Script) != len(walked.Script) || raised.Fight.Finished() {
		t.Errorf("raising the description spent %d decisions",
			len(raised.Script)-len(walked.Script))
	}
	// It works while aiming — the skill is chosen and the cell is not, so what
	// the skill does is still the live question.
	aiming := walked
	aiming.Aiming = true
	if _, action := asking(t, c, aiming, "?"); action.Kind != Raise {
		t.Errorf("? while aiming asked for %v", action.Kind)
	}
	// And a turn with nothing to take has nothing to describe.
	over := playedOut(t, c, atTheBattle(t, c))
	if _, action := asking(t, c, over, "?"); action.Kind != Stay {
		t.Errorf("? on a finished battle asked for %v", action.Kind)
	}
}

// TestTheBattleFootersNameTheDescriptionKeyAndFit is the defect this shipped
// with, measured rather than counted.
//
// ⚠️ Both battle footers were over the window and nothing said so: the fixture in
// the client's everyScreen handed the width sweep a battle that was already
// finished, so PlayFooter and PlayAimFooter were drawn by nothing in the suite and
// came to 82 and 83 cells against the 79 there were. The sweeps cover them now;
// this holds the other half, which a width test cannot — that the key the whole
// feature hangs on is still named after the next person trims a footer.
//
// ⚠️ **The scroll keys are named here too, and the aim footer carries them
// because the log is drawn while aiming.** Room had to be made rather than a key
// given up: the battle footer was 77 cells (vi) and 78 (en) of the 79 there were,
// so the words after ↑/↓, enter and ? are dropped — the three keys whose meaning
// the screen itself shows, which is the same judgement BrowseFooter and this
// footer's own esc already took. The widths below are logged rather than asserted
// against a number, because a hand-count of a candidate came back four cells
// wrong twice in a row.
func TestTheBattleFootersNameTheDescriptionKeyAndFit(t *testing.T) {
	const drawable = MinWidth - 1
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		footers := map[string]string{
			"battle": c.Text(i18n.PlayFooter, SaveKeyLabel()),
			"aim":    c.Text(i18n.PlayAimFooter),
			// The log is drawn on a finished battle as well, and reading back
			// through it is most of what is left to do there.
			"over": c.Text(i18n.PlayOverFooter, SaveKeyLabel()),
		}
		for name, footer := range footers {
			if width := lipgloss.Width(footer); width > drawable {
				t.Errorf("%s: the %s footer is %d cells over the %d it has: %q",
					lang, name, width, drawable, footer)
			}
			for _, named := range []string{scrollBackKey, scrollOnKey} {
				if !strings.Contains(footer, named) {
					t.Errorf("%s: the %s footer does not name %q: %q",
						lang, name, named, footer)
				}
			}
			t.Logf("%s: the %s footer is %d cells", lang, name, lipgloss.Width(footer))
		}
		// The two footers a turn is taken from also name the key the description
		// hangs on. The finished battle has no option under a cursor, so it has
		// nothing to describe and does not offer it.
		for _, name := range []string{"battle", "aim"} {
			if !strings.Contains(footers[name], describeKey) {
				t.Errorf("%s: the %s footer does not name %q: %q",
					lang, name, describeKey, footers[name])
			}
		}
	}
}

// describeKey is the keystroke that raises a description, and the two scroll keys
// are the pair that walks the log — named here so the test above is about the
// footer and not about a letter.
//
// They are [ and ] because ↑/↓ walk the options and may not be taken, and because
// this pair scrolls the trait description and the picker too: a second pair for one
// idea is the drift this repository keeps a list of. The footer spells them the way
// PickerReadingFooter does.
//
// ⚠️ **They are what the footer NAMES, which is no longer all that scrolls.**
// pgup/pgdown still walk every one of those frames and are not going away — the
// brackets are aliases for them, asserted site by site in the client's
// TestABracketScrollsWhereverAPageKeyDoes. What changed is which pair is
// advertised, and it is the brackets because a compact keyboard has no page keys
// at all.
const (
	describeKey   = "?"
	scrollBackKey = "["
	scrollOnKey   = "]"
)

// # The battle screen's height, and where the cut lands
//
// The screen cannot fit the window the tool declares, and nothing below pretends
// it can. Measured at 120x24, where PlayBodyRoom leaves the body twenty rows: the
// heading is one, tui.Board is a fixed ten, tui.Roster is one plus a row a unit,
// tui.Order is one, the log asks for PlayLogWanted and the option list is one plus
// a row an option — so a 1v1 wants twenty of those rows before a single blank or
// log line, a 3v3 twenty-four and a 5v5 **twenty-eight**. A legal squad is up to
// hex.MaxTeamSize a side, so twenty-eight is the floor for one, and a summon puts
// units on the board past the five a squad brought.
//
// What was fixable is **where the cut lands**. A client's frame cuts from the
// bottom and the option list was the last thing the body wrote, so the one thing a
// player has to see in order to act was the first thing thrown away.
//
// ⚠️ **The frame itself is not visible from here**, so the two claims that are
// about it — that the option list survives every window and that the Truncated
// marker never appears — stay in cmd/hexforge-tui, where the frame is. What is
// held below is the body's own budget.
//
// playHeights is the sweep and playSides the squads it is taken over: one, three
// and five a side, the last being the largest a squad may field.
var (
	playHeights = heightsFrom(MinHeight, 48)
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
func withAFullLog(t *testing.T, c Context, p PlayScreen) PlayScreen {
	t.Helper()
	for range 20 {
		if len(p.LogRows(c)) >= PlayLogWanted || p.Fight.Finished() {
			break
		}
		p = playing(t, c, p, "a")
	}
	if rows := len(p.LogRows(c)); rows < PlayLogWanted {
		t.Fatalf("the history came to %d rows of %d, so a squeezed log was not measured",
			rows, PlayLogWanted)
	}
	if p.Pending == nil {
		t.Fatal("the battle stopped waiting on the player, so no option list is drawn")
	}
	return p
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
func withALongLog(t *testing.T, c Context, p PlayScreen, rows int) PlayScreen {
	t.Helper()
	for range PlayTurnLimit {
		if len(p.LogRows(c)) >= rows || p.Fight.Finished() || p.Err != nil {
			break
		}
		p = playing(t, c, p, "a")
	}
	if p.Err != nil {
		t.Fatalf("playing the battle out broke: %v", p.Err)
	}
	if got := len(p.LogRows(c)); got < rows {
		t.Fatalf("the battle finished with a history of %d rows against the %d this "+
			"needs, so nothing above the frame was constructed", got, rows)
	}
	return p
}

// aLongLog is a three-a-side battle whose history runs past any frame the sweep
// draws, in a window the log actually has rows in.
//
// ⚠️ **Its own battle every call**, for the reason the file comment gives.
//
// longLogRows is how long "longer than any window draws" is: the body of an 80x48
// window is 44 rows, all of which the log could in principle be given.
//
// longLogHeight is the window the fixture stands in, and it is not the floor: at
// 120x24 a three-a-side battle is given **no** log row at all — which is the
// budget working, and a fixture standing there would be pressing scroll keys at a
// section that is not on the screen.
const (
	longLogRows   = 120
	longLogHeight = 40
)

func aLongLog(t *testing.T, lang i18n.Lang, side int) (Context, PlayScreen) {
	t.Helper()
	c, _ := start(t, lang)
	c = inAWindow(c, longLogHeight)
	return c, withALongLog(t, c, atABattleOf(t, c, side), longLogRows)
}

// theLogFrame is the history, how many rows of it the window in hand leaves, and
// where the frame starts — read the way the screen reads them, through the same
// drawings and the same playFit, because a test with its own arithmetic for this
// would agree with itself rather than with the screen.
func theLogFrame(c Context, p PlayScreen) (history []string, room, start int) {
	drawn := p.drawings(c)
	room = playFit(PlayBodyRoom(c.Height), drawn.sizes()).log
	return drawn.log, room, p.logStart(len(drawn.log), room)
}

// drawn is which of the game client's drawings the body came back holding.
//
// ⚠️ **It is read out of the screen and never off the plan.** A test that asked
// the view what it had decided would be a test measuring itself, so this looks
// for the drawings themselves — tui.Board's own rows, tui.Roster's own rows,
// tui.Order's own line, the log's own rendered rows — among the rows the body
// handed back.
type drawnSections struct {
	board  bool
	roster int
	order  bool
	log    int
	notice string
}

func whatIsDrawn(c Context, p PlayScreen) (drawnSections, string, string) {
	body, footer := p.View(c)
	lines := strings.Split(body, "\n")
	present := make(map[string]int, len(lines))
	for _, line := range lines {
		present[line]++
	}
	var found drawnSections
	board := strings.Split(tui.Board(p.Fight, p.Tags), "\n")
	found.board = present[board[0]] > 0 && present[board[len(board)-1]] > 0
	// The header is not a unit, so the count starts past it.
	for _, row := range strings.Split(tui.Roster(p.Fight, p.Tags), "\n")[1:] {
		if present[row] > 0 {
			found.roster++
		}
	}
	found.order = present[c.Style.Dim.Render(tui.Order(p.Fight.Queue(), p.Tags, 6))] > 0
	// Counted as a multiset and not by lookup: the log's rows are not distinct —
	// tui.Line opens a turn with a blank one — so asking whether each is on the
	// screen counts every blank row once per blank row there is. Over the **whole
	// history** rather than over the frame, because the frame is what is being
	// measured: counting the rows the screen decided to draw would be the test
	// asking the view what it had decided.
	for _, row := range p.LogRows(c) {
		if present[row] > 0 {
			present[row]--
			found.log++
		}
	}
	// The notice is the row under the heading, matched on the wording's own
	// opening rather than on a word of it: a wrapped save note may perfectly well
	// begin with "the".
	if len(lines) > 1 && strings.HasPrefix(lines[1], noticeOpening(c)) {
		found.notice = lines[1]
	}
	return found, body, footer
}

// noticeOpening is everything the notice says before its list, which is what
// tells that row apart from every other row on the screen.
func noticeOpening(c Context) string {
	const mark = "\x00"
	return strings.SplitN(c.Text(i18n.PlayHidden, mark), mark, 2)[0]
}

// rowHolding is the row whose id column is the given id, cursor marker and all.
//
// The id column rather than the whole row, because "bolt" is a prefix of
// "arc_bolt" and a substring search would answer with the wrong row — and then
// check the marker on it, which is the shape of a test that passes for the wrong
// reason.
func rowHolding(body, id string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		if len(line) > PlayMarkerWidth && strings.HasPrefix(line[PlayMarkerWidth:], id) {
			return line, true
		}
	}
	return "", false
}

// TestTheAimListSurvivesEveryWindowTheToolWillDraw is the bound that replaced the
// old tripwire, for the second of the two questions.
//
// Aiming is the taller of the two states — the option list is still drawn above
// the cells — so it is where a budget reserving only the options would show. Every
// cell on its own row and the marker on the one under the cursor, at every height
// from MinHeight up, in both languages, for one, three and five units a side.
func TestTheAimListSurvivesEveryWindowTheToolWillDraw(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			c, _ := start(t, lang)
			base := atABattleOf(t, c, side)
			base.Aiming = true
			aims := base.Pending.Options[base.Option].Aims
			if len(aims) == 0 {
				t.Fatalf("%s %dv%d: the option under the cursor has nowhere to point",
					lang, side, side)
			}
			for _, height := range playHeights {
				sized := inAWindow(c, height)
				_, body, _ := whatIsDrawn(sized, base)
				for index, cell := range aims {
					row, ok := rowHolding(body, cell.String())
					if !ok {
						t.Errorf("%s %dv%d h=%d: the cell %s is not on the screen:\n%s",
							lang, side, side, height, cell, body)
						continue
					}
					if index == base.Aim && !strings.HasPrefix(row, "> ") {
						t.Errorf("%s %dv%d h=%d: the aim under the cursor draws %q, "+
							"want the marker", lang, side, side, height, row)
					}
				}
			}
		}
	}
}

// TestTheBattleBodyNeverOutrunsItsRoom is the cleanest statement of the fix that
// this package can make.
//
// The body budgets itself against the room a frame will give it, so the frame
// never has to cut. Four bodies rather than one, because they are four different
// heights: waiting on a skill, waiting on a cell, with a save's notes under the
// list, and over.
//
// ⚠️ Each state is built for itself and the finished one from its own battle —
// PlayScreen holds a *battle.Battle, so a copy driven to its end steps the battle
// every other copy points at.
//
// ⚠️ **The saved state is a Notes value rather than a write.** Nothing in this
// package opens a file, so the note is the pair internal/forge would have handed
// back; what the real write produces is measured in the client, which has a
// scratch directory to write into.
func TestTheBattleBodyNeverOutrunsItsRoom(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			c, _ := start(t, lang)
			base := withAFullLog(t, c, atABattleOf(t, c, side))
			states := map[string]PlayScreen{"a turn": base}
			aiming := base
			aiming.Aiming = true
			states["aiming"] = aiming
			saved := base
			saved.Notes = aSaveNote()
			states["saved"] = saved
			states["over"] = playedOut(t, c, atABattleOf(t, c, side))
			for name, state := range states {
				for _, height := range playHeights {
					sized := inAWindow(c, height)
					body, _ := state.View(sized)
					if rows := len(strings.Split(body, "\n")); rows > PlayBodyRoom(height) {
						t.Errorf("%s %dv%d %s h=%d: the body is %d rows against the %d it has",
							lang, side, side, name, height, rows, PlayBodyRoom(height))
					}
				}
			}
		}
	}
}

// aSaveNote is the pair of notes a write leaves behind, as values.
//
// The path is **relative**, which is what keeps this package's golden and its
// noAbsolutePath walk honest — and it is also what a save really writes here,
// since forge.Library.SaveBattleLog builds its path off the directory it was
// loaded from.
func aSaveNote() []forge.Note {
	const written = "battles/do-thu-vs-do-thu-seed1.json"
	return []forge.Note{
		{Kind: forge.NoteWrote, ID: "do-thu-vs-do-thu-seed1.json", Path: written},
		{Kind: forge.NoteBattleVerify, Path: written},
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
		c, _ := start(t, i18n.Vi)
		base := withAFullLog(t, c, atABattleOf(t, c, side))
		units := len(base.Fight.Units())
		boardGoneRosterWhole, logWentBeforeOrder := false, false
		previous := drawnSections{log: -1}
		for _, height := range playHeights {
			sized := inAWindow(c, height)
			found, body, _ := whatIsDrawn(sized, base)
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
		c, _ := start(t, lang)
		base := atABattleOf(t, c, hex.MaxTeamSize)
		base.Aiming = true
		units := len(base.Fight.Units())
		clipped := 0
		for _, height := range playHeights {
			sized := inAWindow(c, height)
			found, body, _ := whatIsDrawn(sized, base)
			if found.roster >= units || found.roster == 0 {
				continue
			}
			clipped++
			hidden := units - found.roster
			want := sized.Text(i18n.PlayHiddenUnits, hidden)
			if hidden == 1 {
				want = sized.Text(i18n.PlayHiddenUnitsOne)
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
// The log's budget is spent in **rows** and not in events. The two are not the
// same number: tui.Line opens a turn with a blank row of its own, so one event
// arrives as two rows, and the section with the loosest claim on the screen was
// the one whose stated budget did not hold.
//
// ⚠️ The multi-row event has to be **reached**, which the opening board does not
// do: it takes a turn or two before the log holds one. This builds that and fails
// if it cannot, rather than measuring a log every row of which is one event.
func TestTheLogIsCappedByRenderedLinesAndNotByEvents(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	p := withAFullLog(t, c, atABattleOf(t, c, 3))
	widest := 0
	for _, event := range p.Events {
		if line := tui.Line(event, p.Tags, nil); line != "" {
			widest = max(widest, len(strings.Split(line, "\n")))
		}
	}
	if widest < 2 {
		t.Fatalf("no event in the history renders as more than one row, so a budget in "+
			"rows and a budget in events are the same number here (widest %d)", widest)
	}
	whole := p.LogRows(c)
	for _, room := range []int{1, 3, PlayLogWanted} {
		rows := p.logFrame(whole, room)
		if len(rows) != min(room, len(whole)) {
			t.Errorf("a budget of %d rows drew %d", room, len(rows))
		}
		if len(rows) == 0 {
			continue
		}
		if rows[len(rows)-1] != whole[len(whole)-1] {
			t.Errorf("a budget of %d rows ends on %q, want the log's own last row %q",
				room, rows[len(rows)-1], whole[len(whole)-1])
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

// TestTheLogGrowsWithTheWindow is the first defect.
//
// The rows the log is drawn are what the budget leaves it, so a taller window
// gives it more of them. Measured at the floor, in the middle and at a height
// nobody's laptop is short of, in both languages and at every squad size — and the
// two ends may not be equal, which is exactly what they were.
func TestTheLogGrowsWithTheWindow(t *testing.T) {
	heights := []int{MinHeight, 40, 80}
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			c, base := aLongLog(t, lang, side)
			rooms := make([]int, 0, len(heights))
			for _, height := range heights {
				sized := inAWindow(c, height)
				found, body, _ := whatIsDrawn(sized, base)
				_, room, _ := theLogFrame(sized, base)
				if found.log != room {
					t.Errorf("%s %dv%d h=%d: the budget gave the log %d rows and %d are "+
						"drawn:\n%s", lang, side, side, height, room, found.log, body)
				}
				rooms = append(rooms, room)
			}
			if rooms[0] >= rooms[len(rooms)-1] {
				t.Errorf("%s %dv%d: the log is %d rows at h=%d and %d at h=%d — a taller "+
					"window buys the history nothing, which is the defect",
					lang, side, side, rooms[0], heights[0],
					rooms[len(rooms)-1], heights[len(heights)-1])
			}
			t.Logf("%s %dv%d: %v rows at %v", lang, side, side, rooms, heights)
		}
	}
}

// TestEveryRowOfTheHistoryIsReachable is the second defect: two hundred and
// ninety-two rows of three hundred could not be got at by any means.
//
// Scrolled to the top the battle's first event is on screen; scrolled back down
// its newest is. Both ends are walked with a page count taken off the history
// rather than a number written here, so the test cannot pass by not going far
// enough.
func TestEveryRowOfTheHistoryIsReachable(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, base := aLongLog(t, lang, 3)
		history, room, _ := theLogFrame(c, base)
		if room <= 0 || len(history) <= room {
			t.Fatalf("%s: the frame holds %d rows of a %d-row history, so there is "+
				"nothing above it and nothing to reach", lang, room, len(history))
		}
		pages := len(history)/room + 2
		top := base
		for range pages {
			top = playing(t, c, top, "pgup")
		}
		if _, _, at := theLogFrame(c, top); at != 0 {
			t.Errorf("%s: scrolling to the top left the frame at row %d", lang, at)
		}
		body, _ := top.View(c)
		if !strings.Contains(body, history[0]) {
			t.Errorf("%s: the battle's first row %q is not on screen at the top:\n%s",
				lang, history[0], body)
		}
		bottom := top
		for range pages {
			bottom = playing(t, c, bottom, "pgdown")
		}
		body, _ = bottom.View(c)
		if !strings.Contains(body, history[len(history)-1]) {
			t.Errorf("%s: the newest row %q is not on screen at the bottom:\n%s",
				lang, history[len(history)-1], body)
		}
		// And coming back down is a reader asking for the newest rows, which is
		// the state rather than the number that happens to be the newest now.
		if !bottom.LogFollow {
			t.Errorf("%s: scrolling back to the bottom left the log at an offset "+
				"rather than following the tail", lang)
		}
	}
}

// TestFollowingTheTailIsNotAnOffset is the load-bearing claim, and the one a
// stored-offset implementation fails while passing everything else here.
//
// ⚠️ **The tail moves.** A reader at the newest rows is in a *state*, and if that
// state is stored as the offset which happened to be the newest, then the next
// event to arrive leaves them one frame behind with nothing saying so.
//
// ⚠️ **The event is appended rather than fought for, and that is the point.**
// Every turn this screen takes goes through record, which puts the reader back on
// the tail — so a test that played a turn would be measuring that reset and would
// pass against a stored offset. What has to be measured is an event arriving with
// no decision behind it, which is what a skipped turn and an engine's own turn do
// between the player's. The event is one the battle really emitted; a duplicate
// row is a row like any other to a frame.
func TestFollowingTheTailIsNotAnOffset(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, base := aLongLog(t, lang, 3)
		if !base.LogFollow {
			t.Fatalf("%s: the battle opened without following its own tail", lang)
		}
		// At the tail, and with rows above the frame, or nothing below measures.
		history, room, at := theLogFrame(c, base)
		if room <= 0 || len(history) <= room {
			t.Fatalf("%s: the whole history is on screen, so following it is not a "+
				"question this fixture asks", lang)
		}
		if at != len(history)-room {
			t.Fatalf("%s: following the tail starts the frame at row %d of %d",
				lang, at, len(history))
		}
		grown := base
		grown.Events = append(grown.Events, grown.Events[len(grown.Events)-1])
		after, room, at := theLogFrame(c, grown)
		if len(after) <= len(history) {
			t.Fatalf("%s: the appended event added no row, so nothing arrived", lang)
		}
		if at != len(after)-room {
			t.Errorf("%s: an event arrived and the frame stayed at row %d of %d — the "+
				"reader is looking at a stored offset rather than at the tail",
				lang, at, len(after))
		}
		body, _ := grown.View(c)
		if !strings.Contains(body, after[len(after)-1]) {
			t.Errorf("%s: the newest row is not on screen after an event arrived:\n%s",
				lang, body)
		}
	}
}

// TestActingReturnsTheLogToItsTail is the rule the blurb screen already follows:
// anything that changes the answer resets the offset into it.
//
// A player who scrolled back to read what happened and then took a turn would
// otherwise be looking at a frame from before their own decision, which is the one
// moment the log is certainly stale. Every way of spending a turn is measured,
// because they are four keys and one of them is the engine's.
func TestActingReturnsTheLogToItsTail(t *testing.T) {
	// The four ways a turn is spent: cast the option under the cursor, hand the
	// turn to the engine, pass it, and take one back.
	for _, spent := range []string{"enter", "a", "p", "u"} {
		c, base := aLongLog(t, i18n.Vi, 3)
		// Somebody's turn has to have been taken before undo has anything to take
		// back, and withALongLog has played dozens.
		scrolled := playing(t, c, base, "pgup")
		if scrolled.LogFollow {
			t.Fatalf("%q: pgup did not scroll back, so the reset is not being measured", spent)
		}
		acted := playing(t, c, scrolled, spent)
		if spent == "enter" && acted.Aiming {
			// A skill with more than one cell asks where before it is cast, and
			// opening that question is not spending the turn.
			acted = playing(t, c, acted, spent)
		}
		if acted.Err != nil {
			t.Fatalf("%q: taking the turn broke: %v", spent, acted.Err)
		}
		if len(acted.Script) == len(scrolled.Script) {
			t.Fatalf("%q spent no turn, so the reset it is about was not reached", spent)
		}
		if !acted.LogFollow {
			t.Errorf("%q left the log at offset %d rather than back on the tail",
				spent, acted.LogOffset)
		}
	}
	// And another seed is another battle, so it is another history.
	c, base := aLongLog(t, i18n.Vi, 3)
	fresh := playing(t, c, base, "pgup", "n")
	if !fresh.LogFollow || fresh.LogOffset != 0 {
		t.Errorf("another seed kept the old battle's offset: following %v at %d",
			fresh.LogFollow, fresh.LogOffset)
	}
}

// TestTheLogFrameSurvivesAnUndo is the shortening nothing else here can produce.
//
// ⚠️ **Undo makes the history shorter.** It cuts the script at the player's last
// decision and rebuilds the battle from the seed, so the events are rebuilt too and
// an offset kept across it can point past the end. The offset is therefore clamped
// wherever it is read and not only where it is written — and this constructs the
// case by writing an offset from the longer history onto the rebuilt one, which is
// the state a clamp only at the write would leave behind.
func TestTheLogFrameSurvivesAnUndo(t *testing.T) {
	c, base := aLongLog(t, i18n.Vi, 3)
	before, _, _ := theLogFrame(c, base)
	undone := playing(t, c, base, "u")
	if undone.Err != nil {
		t.Fatalf("undo broke: %v", undone.Err)
	}
	after, _, _ := theLogFrame(c, undone)
	if len(after) >= len(before) {
		t.Fatalf("undo left a history of %d rows against %d, so a shortening was not "+
			"measured", len(after), len(before))
	}
	// The offset the longer history could hold, carried onto the shorter one.
	stale := undone
	stale.LogFollow, stale.LogOffset = false, len(before)
	history, room, at := theLogFrame(c, stale)
	if at+room > len(history) {
		t.Errorf("an offset of %d over a %d-row history frames rows %d..%d",
			len(before), len(history), at, at+room)
	}
	if rows := len(stale.logFrame(history, room)); rows != min(room, len(history)) {
		t.Errorf("the frame drew %d rows of the %d it has", rows, room)
	}
	// And the body still fits the room it was budgeted against, which is what a
	// client's frame would otherwise have cut.
	body, _ := stale.View(c)
	if rows := len(strings.Split(body, "\n")); rows > PlayBodyRoom(c.Height) {
		t.Errorf("the stale offset drew %d rows against the %d there are",
			rows, PlayBodyRoom(c.Height))
	}
}

// TestTheLogScrollIsClampedAtBothEnds is the two edges and the case where the key
// has nothing to do.
//
// ⚠️ The third of those is a branch the long fixture cannot reach and the short one
// cannot miss, so both are here: a history that fits its frame has nothing above
// it, and the key must do nothing rather than framing rows that are not there.
func TestTheLogScrollIsClampedAtBothEnds(t *testing.T) {
	c, base := aLongLog(t, i18n.Vi, 3)
	history, room, _ := theLogFrame(c, base)
	pages := len(history)/room + 2
	top := base
	for range pages {
		top = playing(t, c, top, "pgup")
	}
	if again := playing(t, c, top, "pgup"); again.LogOffset != top.LogOffset {
		t.Errorf("pgup at the top moved the frame from %d to %d",
			top.LogOffset, again.LogOffset)
	}
	if again := playing(t, c, base, "pgdown"); again.LogOffset != base.LogOffset ||
		!again.LogFollow {
		t.Errorf("pgdown at the bottom moved the frame to %d (following %v)",
			again.LogOffset, again.LogFollow)
	}
	// A history that fits: the keys are quiet rather than helpful.
	plain, _ := start(t, i18n.Vi)
	plain = inAWindow(plain, 80)
	short := withAFullLog(t, plain, atABattleOf(t, plain, 1))
	rows, room, _ := theLogFrame(plain, short)
	if len(rows) > room {
		t.Fatalf("the short fixture holds %d rows in a frame of %d, so it is not the "+
			"case this is about", len(rows), room)
	}
	for _, pressed := range []string{"pgup", "pgdown"} {
		quiet := playing(t, plain, short, pressed)
		if !quiet.LogFollow || quiet.LogOffset != 0 {
			t.Errorf("%s with the whole history on screen left the log at %d (following %v)",
				pressed, quiet.LogOffset, quiet.LogFollow)
		}
	}
}

// TestTheLogScrollsWhileAiming is the state the keys are easiest to forget in and
// the one a width sweep could not see for a whole release.
//
// The log is drawn while a cell is being chosen, so it is still the thing being
// read — the same argument that put ? there.
func TestTheLogScrollsWhileAiming(t *testing.T) {
	c, base := aLongLog(t, i18n.Vi, 3)
	base.Aiming = true
	_, _, at := theLogFrame(c, base)
	scrolled := playing(t, c, base, "pgup")
	if !scrolled.Aiming {
		t.Fatal("pgup while aiming dropped the aim")
	}
	_, _, moved := theLogFrame(c, scrolled)
	if moved >= at {
		t.Errorf("pgup while aiming left the frame at row %d of %d", moved, at)
	}
}

// TestThePositionOnTheHeadingNamesTheRowsOnScreen is the indicator.
//
// Three things. It is on the heading row rather than on a row of its own, because
// this screen has no row to spare. It appears exactly when rows are hidden — not
// only when somebody has scrolled back, since the other half of the defect was that
// nothing said a history existed at all. And its numbers are checked against the
// rows the body actually drew rather than against a second calculation: the run of
// history rows it claims is on screen has to be on screen, contiguously.
func TestThePositionOnTheHeadingNamesTheRowsOnScreen(t *testing.T) {
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			c, base := aLongLog(t, lang, side)
			said, quiet := 0, 0
			for _, height := range playHeights {
				sized := inAWindow(c, height)
				history, room, _ := theLogFrame(sized, base)
				body, _ := base.View(sized)
				lines := strings.Split(body, "\n")
				hidden := room > 0 && len(history) > room
				position := sized.Text(i18n.PlayLogRange,
					base.logStart(len(history), room)+1,
					base.logStart(len(history), room)+room, len(history))
				switch {
				case hidden && !strings.Contains(lines[0], position):
					t.Errorf("%s %dv%d h=%d: %d rows of %d are drawn and the heading is "+
						"%q, want %q", lang, side, side, height, room, len(history),
						lines[0], position)
				case !hidden && strings.Contains(lines[0], onlyThePosition(sized)):
					t.Errorf("%s %dv%d h=%d: the whole log is on screen and the heading "+
						"still says where the frame is: %q", lang, side, side, height, lines[0])
				}
				if !hidden {
					quiet++
					continue
				}
				said++
				// The numbers, against the screen: the rows the heading names are the
				// rows the body drew, in that order and next to each other.
				first, last := numbersIn(t, lines[0], len(history))
				if last-first+1 != room {
					t.Errorf("%s %dv%d h=%d: the heading names rows %d..%d and the frame "+
						"holds %d", lang, side, side, height, first, last, room)
				}
				if !holdsRun(lines, history[first-1:last]) {
					t.Errorf("%s %dv%d h=%d: the heading names rows %d..%d and they are not "+
						"the rows on screen:\n%s", lang, side, side, height, first, last, body)
				}
			}
			if said == 0 {
				t.Errorf("%s %dv%d: no height in the sweep hid a row, so the position is "+
					"drawn by nothing", lang, side, side)
			}
			if quiet == 0 {
				t.Errorf("%s %dv%d: every height in the sweep hid a row, so the case the "+
					"position stays away from went unmeasured", lang, side, side)
			}
		}
	}
}

// onlyThePosition is everything the position says before its first number, which
// is what tells that clause apart from the rest of the heading row.
func onlyThePosition(c Context) string {
	const mark = "\x00"
	return strings.SplitN(c.Text(i18n.PlayLogRange, mark, mark, mark), mark, 2)[0]
}

// numbersIn reads the first and last row the heading claims to be showing.
//
// The heading's figures are the seed and then the position's three, in both
// languages, so the last three digit runs are the range and the total — read out of
// the row rather than filled into the wording, which would be the test agreeing
// with itself.
func numbersIn(t *testing.T, heading string, total int) (first, last int) {
	t.Helper()
	figures := digitsIn(heading)
	if len(figures) < 3 {
		t.Fatalf("the heading %q names no range", heading)
	}
	said := figures[len(figures)-3:]
	if said[2] != total {
		t.Fatalf("the heading %q counts a history of %d rows, and it has %d",
			heading, said[2], total)
	}
	return said[0], said[1]
}

// digitsIn is every run of digits in a line, in order, as numbers.
func digitsIn(text string) []int {
	var out []int
	for at := 0; at < len(text); {
		if text[at] < '0' || text[at] > '9' {
			at++
			continue
		}
		end := at
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
		}
		value, err := strconv.Atoi(text[at:end])
		if err == nil {
			out = append(out, value)
		}
		at = end
	}
	return out
}

// holdsRun says whether the lines hold the run next to each other and in order.
func holdsRun(lines, run []string) bool {
	for at := 0; at+len(run) <= len(lines); at++ {
		if slices.Equal(lines[at:at+len(run)], run) {
			return true
		}
	}
	return false
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
// ⚠️ **A shorter log frame is not a missing section.** The log is a frame over a
// history that is always longer than it, so a frame two rows shorter is the
// section working; a frame with no rows at all is not, and that is the line the
// notice is drawn on. Since the log now asks for the whole history, *some* of it
// is hidden in nearly every window — which is what the position on the heading row
// says, and it is a different statement from the notice's.
func TestTheNoticeNamesWhatIsMissingAndNothingElse(t *testing.T) {
	const drawable = MinWidth - 1
	nameable := []i18n.Key{i18n.PlayHiddenBoard, i18n.PlayHiddenOrder, i18n.PlayHiddenLog}
	for _, lang := range i18n.Langs() {
		for _, side := range playSides {
			c, _ := start(t, lang)
			base := withAFullLog(t, c, atABattleOf(t, c, side))
			named := make(map[i18n.Key]bool)
			for _, height := range playHeights {
				sized := inAWindow(c, height)
				found, body, _ := whatIsDrawn(sized, base)
				units := len(base.Fight.Units())
				wanted := len(base.LogRows(sized))
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
					if found.notice != "" && strings.Contains(found.notice, sized.Lang.Text(key)) {
						named[key] = true
					}
				}
			}
			for _, key := range nameable {
				if !named[key] {
					t.Errorf("%s %dv%d: no height in the sweep hides what %q names, so that "+
						"wording is rendered by nothing", lang, side, side, c.Lang.Text(key))
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
// first unit, and both of those are below the height a client draws a
// too-small message at. Written out rather than left uncovered, so the walk cannot
// be re-ordered silently.
//
// The sizes are one 5-a-side board's: ten units, ten rows of board, a four-option
// list and a full log.
func TestTheBudgetSpendsWhatItSaysItDoes(t *testing.T) {
	squad := playSizes{tail: 5, board: 10, units: 10, log: 8}
	saved := playSizes{tail: 5, notes: 4, board: 10, units: 10, log: 8}
	aiming := playSizes{tail: 12, board: 10, units: 10, log: 8}
	// The same board with a history longer than any window, which is what a
	// battle a dozen turns in actually has. It is a separate fixture because the
	// three above cannot reach the surplus: a history of eight rows is a history
	// with nothing left over to hand the log.
	history := playSizes{tail: 5, board: 10, units: 10, log: 300}
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
		why: "a long history takes every row nobody above it claimed, so a tall " +
			"window buys the log something: sixteen rows over the eight it asked for",
		room:  56,
		sizes: history,
		want:  playPlan{board: true, roster: 10, order: true, log: 24},
	}, {
		why: "and it takes them without anything above it losing a row: the same " +
			"purse gives the roster, the board and the order line exactly what the " +
			"eight-row history did",
		room:  29,
		sizes: history,
		want:  playPlan{roster: 10, order: true, log: 6, notice: true},
		names: []i18n.Key{i18n.PlayHiddenBoard},
	}, {
		why: "a purse with no room for the log's first row gives it none, however " +
			"long the history is",
		room:  32,
		sizes: history,
		want:  playPlan{board: true, roster: 10, order: true, notice: true},
		names: []i18n.Key{i18n.PlayHiddenLog},
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
