package tui_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

// The ids the gloss sweep drives its lines with, taken from the SHIPPED books.
//
// ⚠️ **A fixture skill would prove nothing here, and that is the whole trap this
// file exists to avoid.** i18n's compiled skillGloss holds the nineteen ids that
// shipped before skill.Skill carried a name — strike, creeping_rot, purify and
// the rest — and **not one of them is in skills.json any more**; only
// internal/testfixture still reaches them. So a line glossed with `creeping_rot`
// takes the id-table path and leaves the authored-name path that all 43 shipped
// skills take completely unexercised. venoshock is a shipped skill and
// TestAShippedSkillIsGlossedFromItsAuthoredName asserts which path names it.
//
// The three are also checked against each other for nesting: the sweep counts
// occurrences of an id in a line, so an id inside another id would make the count
// lie.
const (
	sweepSkill   = "venoshock"
	sweepStatus  = "poison"
	sweepPassive = "venom_blood"
)

func sweepGlosses(t *testing.T) map[string]string {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	glosses := i18n.Vi.LogGlosses(
		books.Skills.Skills(), books.Statuses.Kinds(), books.Passives.All())
	for _, id := range []string{sweepSkill, sweepStatus, sweepPassive} {
		if glosses[id] == "" {
			t.Fatalf("the shipped books gloss no %q, so the sweep would measure a bare id", id)
		}
	}
	ids := []string{sweepSkill, sweepStatus, sweepPassive}
	for _, one := range ids {
		for _, other := range ids {
			if one != other && strings.Contains(other, one) {
				t.Fatalf("the id %q sits inside %q, so counting occurrences of it cannot be trusted",
					one, other)
			}
		}
	}
	return glosses
}

// glossArm is one branch of tui.Line that prints a data id, and how many of them
// it is expected to name.
//
// ⚠️ **This table IS the coverage check, and it is a table of ARMS rather than of
// kinds** — status_resisted alone has four branches and damaged has two, so a
// sweep over battle.KindCount would render one arm of each and report full
// coverage. Every entry with want > 0 is an arm the change is *for*; every entry
// with want 0 is an arm that must stay bare, so a gloss leaking into a line that
// names no data id fails here too.
//
// **Adding a branch to tui.Line that prints event.Skill, event.Status or
// event.Passive means adding a row here.** If it is not added, the count check at
// the foot of the sweep fails and names how many arms went unmeasured — which is
// the half an assertion cannot do on its own, because a table nobody extended
// passes completely.
type glossArm struct {
	name  string
	event battle.Event
	// want is how many of the three glossable fields this arm prints.
	want int
}

func sweepArms() []glossArm {
	base := battle.Event{
		At: 5000, Turn: 3, Actor: "a", Target: "f",
		Skill: sweepSkill, Status: sweepStatus, Passive: sweepPassive,
		Amount: 617, Before: 100, Stacks: 2, Strike: 1, Chance: 900,
		Multiplier: 1500, Power: 1800, Remaining: 2400,
		Cell: hex.At(hex.Offset{Col: 3, Row: 1}), Side: hex.SideAlly, Note: "ally",
	}
	with := func(kind battle.Kind, edit func(*battle.Event)) battle.Event {
		event := base
		event.Kind = kind
		if edit != nil {
			edit(&event)
		}
		return event
	}
	noPassive := func(event *battle.Event) { event.Passive = "" }
	return []glossArm{
		// The arms that gloss. Together these are every print of event.Skill,
		// event.Status and event.Passive in tui.Line.
		{"status_ticked", with(battle.StatusTicked, nil), 1},
		{"healed/regen", with(battle.Healed, nil), 1},
		{"status_expired", with(battle.StatusExpired, nil), 1},
		{"skill_used", with(battle.SkillUsed, nil), 1},
		{"amplified", with(battle.Amplified, nil), 2},
		{"status_consumed", with(battle.StatusConsumed, nil), 1},
		{"status_applied", with(battle.StatusApplied, noPassive), 1},
		{"status_applied/trait", with(battle.StatusApplied, nil), 2},
		{"status_resisted/immune", with(battle.StatusResisted, func(e *battle.Event) {
			e.Refused = 1000
		}), 1},
		{"status_resisted/shrugs off", with(battle.StatusResisted, func(e *battle.Event) {
			e.Refused = 400
		}), 1},
		{"status_resisted/wide open", with(battle.StatusResisted, func(e *battle.Event) {
			e.Refused = -200
		}), 1},
		{"status_resisted/resists", with(battle.StatusResisted, func(e *battle.Event) {
			e.Refused = 0
		}), 1},
		{"passive_held", with(battle.PassiveHeld, nil), 2},
		{"passive_released", with(battle.PassiveReleased, nil), 2},
		{"damaged/reply", with(battle.Damaged, nil), 1},
		{"summoned", with(battle.Summoned, nil), 1},

		// The arms that must stay bare. A line naming no data id may not grow one.
		{"started", with(battle.Started, nil), 0},
		{"turn_began", with(battle.TurnBegan, nil), 0},
		{"speed_changed", with(battle.SpeedChanged, nil), 0},
		{"turn_skipped", with(battle.TurnSkipped, nil), 0},
		{"missed", with(battle.Missed, nil), 0},
		{"blocked", with(battle.Blocked, nil), 0},
		{"damaged/strike", with(battle.Damaged, noPassive), 0},
		{"healed/drain", with(battle.Healed, func(e *battle.Event) {
			e.Status, e.Drained = "", 400
		}), 0},
		{"healed/plain", with(battle.Healed, func(e *battle.Event) {
			e.Status, e.Drained = "", 0
		}), 0},
		{"status_stripped", with(battle.StatusStripped, nil), 0},
		{"died", with(battle.Died, nil), 0},
		{"left", with(battle.Left, nil), 0},
		{"ended", with(battle.Ended, nil), 0},
	}
}

// glossingArms is how many rows of sweepArms name at least one data id.
//
// The number is written down rather than counted off the table, so extending the
// table is not the same act as claiming the extension was measured: a row added
// with the wrong want raises the counted total and this constant is what
// disagrees.
const glossingArms = 16

// TestEveryGlossableIDOnALineIsNamed is the sweep TestEveryEventKindRenders is
// for coverage: every arm of tui.Line that prints a skill, a status or a trait id
// must print it with its name beside it, at **every** occurrence.
func TestEveryGlossableIDOnALineIsNamed(t *testing.T) {
	glosses := sweepGlosses(t)
	tags := map[string]string{"a": "A1", "f": "E1"}
	glossed := map[string]string{}
	for _, id := range []string{sweepSkill, sweepStatus, sweepPassive} {
		glossed[id] = i18n.GlossBracket(id, glosses[id])
	}

	named, kinds := 0, map[battle.Kind]bool{}
	var unmeasured []string
	for _, arm := range sweepArms() {
		t.Run(arm.name, func(t *testing.T) {
			line := tui.Line(arm.event, tags, glosses)
			if strings.TrimSpace(line) == "" {
				t.Fatalf("%s renders as nothing", arm.name)
			}
			found := 0
			for id, want := range glossed {
				bare := strings.Count(line, id)
				if bare == 0 {
					continue
				}
				found += bare
				if with := strings.Count(line, want); with != bare {
					t.Errorf("%s prints %s %d times and names it %d:\n%s",
						arm.name, id, bare, with, line)
				}
			}
			if found != arm.want {
				t.Errorf("%s names %d data ids, want %d:\n%s", arm.name, found, arm.want, line)
			}
		})
		if arm.want == 0 {
			continue
		}
		line := tui.Line(arm.event, tags, glosses)
		printed := 0
		for id := range glossed {
			printed += strings.Count(line, id)
		}
		if printed == 0 {
			unmeasured = append(unmeasured, arm.name)
			continue
		}
		named++
		kinds[arm.event.Kind] = true
	}

	// ⚠️ The half an assertion cannot see: an arm whose branch was never taken
	// passes every check above, because a line naming no id trivially names every
	// id it prints. So the arms that DID print one are counted, and the count is
	// held against the number written down.
	if named != glossingArms {
		sort.Strings(unmeasured)
		t.Errorf("%d of the %d glossing arms printed a data id; %d went unmeasured: %v\n"+
			"either an arm lost its gloss, or a row was added to sweepArms without "+
			"glossingArms being raised to match",
			named, glossingArms, len(unmeasured), unmeasured)
	}
	// The kinds are counted too, because the arms are unevenly spread over them —
	// four of them are status_resisted — and a kind falling out entirely is a
	// different defect from an arm doing so.
	const glossingKinds = 12
	if len(kinds) != glossingKinds {
		t.Errorf("%d event kinds printed a data id, want %d", len(kinds), glossingKinds)
	}
}

// TestALineWithNoGlossesIsTheLineItAlwaysWas is the English contract, and it is
// the cheapest of these tests to hold.
//
// A nil map is what English is (a data id is shown exactly as the data writes it,
// which is what i18n.Lang.LogGlosses returns nil to say) and what a replay drawn
// without books is. So a nil map must put **no** name on **any** line, and it must
// be indistinguishable from an empty one — a caller that has no names and a caller
// that has an empty map are the same caller.
//
// testdata/opening.golden is the other half and is the byte-for-byte proof: it
// renders a real battle's log through tui.Log with no glosses, and it did not move
// when this landed.
func TestALineWithNoGlossesIsTheLineItAlwaysWas(t *testing.T) {
	glosses := sweepGlosses(t)
	tags := map[string]string{"a": "A1", "f": "E1"}
	for _, arm := range sweepArms() {
		t.Run(arm.name, func(t *testing.T) {
			bare := tui.Line(arm.event, tags, nil)
			if empty := tui.Line(arm.event, tags, map[string]string{}); empty != bare {
				t.Errorf("an empty map renders differently from a nil one:\nnil   %q\nempty %q",
					bare, empty)
			}
			for _, id := range []string{sweepSkill, sweepStatus, sweepPassive} {
				if name := glosses[id]; name != "" && strings.Contains(bare, name) {
					t.Errorf("%s put the name %q on a glossless line:\n%s", arm.name, name, bare)
				}
			}
			// The bare id is still there, which is the other half: glossless is the
			// data's own name for itself, not a field dropped.
			for _, id := range []string{sweepSkill, sweepStatus, sweepPassive} {
				withNames := tui.Line(arm.event, tags, glosses)
				if strings.Contains(withNames, id) && !strings.Contains(bare, id) {
					t.Errorf("%s prints %s when glossed and not when bare:\n%s", arm.name, id, bare)
				}
			}
		})
	}
}

// TestAShippedSkillIsGlossedFromItsAuthoredName is the assertion that makes the
// rest of this file about the shipped game rather than about the fixtures.
//
// The mutation it exists for: build LogGlosses off i18n's compiled id tables
// instead of the authored names. Every other test here still passes — statuses are
// glossed by a table, so a status line reads correctly — and **no skill on any log
// line is named at all**, because skillGloss's nineteen ids and skills.json's
// forty-three do not intersect. So this asserts the negative as well as the
// positive: the name is on the line, and Gloss does not know it.
func TestAShippedSkillIsGlossedFromItsAuthoredName(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	declared, err := books.Skills.Lookup(sweepSkill)
	if err != nil {
		t.Fatalf("the shipped book has no %s: %v", sweepSkill, err)
	}
	authored := i18n.Vi.SkillName(declared)
	if authored == "" || authored == declared.ID {
		t.Fatalf("%s carries no authored name, so this test measures nothing", sweepSkill)
	}
	if fromTable := i18n.Vi.Gloss(declared.ID); fromTable != "" {
		t.Fatalf("%s is in a compiled gloss table (%q), so it is not the authored-name "+
			"path this test is about; pick a shipped skill that is not", sweepSkill, fromTable)
	}

	glosses := i18n.Vi.LogGlosses(
		books.Skills.Skills(), books.Statuses.Kinds(), books.Passives.All())
	line := tui.Line(battle.Event{
		Kind: battle.SkillUsed, Actor: "a", Skill: declared.ID,
		Cell: hex.At(hex.Offset{Col: 3, Row: 1}),
	}, map[string]string{"a": "A1"}, glosses)
	if want := i18n.GlossBracket(declared.ID, authored); !strings.Contains(line, want) {
		t.Errorf("a shipped skill rendered as %q, want %q in it", line, want)
	}

	// And the whole shipped book, so this cannot be one lucky id: every skill the
	// game ships must reach the log with a name.
	for _, one := range books.Skills.Skills() {
		if glosses[one.ID] == "" {
			t.Errorf("the shipped skill %s reaches the log with no name", one.ID)
		}
	}
	for _, one := range books.Passives.All() {
		if glosses[one.ID] == "" {
			t.Errorf("the shipped trait %s reaches the log with no name", one.ID)
		}
	}
	for _, kind := range books.Statuses.Kinds() {
		if glosses[kind.ID] == "" {
			t.Errorf("the shipped status %s reaches the log with no name", kind.ID)
		}
	}
}

// knownWide is the arms whose glossed row is over the window, each named with its
// measured width so a breach cannot get quietly worse and cannot be quietly fixed.
//
// ⚠️ **It is empty, and keeping it empty is the point.** It had one entry:
// `amplified` is the only arm printing two glossed ids in one clause, and with
// dragon_drive (the longest shipped skill id at 12 cells, and one that amplifies)
// against expose it measured 82. That was a row which fit before glossing and did
// not after, on a screen #162 measured as having no spare rows — so the arm gave
// up the word `is` instead, and the widest reachable row is 79 of the 79 there
// are. Zero margin is deliberate rather than lucky: the bound below is recomputed
// from the books every run, so the day a longer skill name is authored this test
// goes red rather than the log quietly wrapping.
var knownWide = map[string]int{}

// TestNoGlossedLogRowOutgrowsTheWindow measures every reachable combination the
// shipped books can put on a glossing arm, against the 79 cells the client draws.
//
// Reachable rather than worst-case: pairing the longest skill name with the
// longest status name and the longest trait name in one event gives 102 cells and
// describes no event any battle emits — a skill is amplified by **its own**
// condition's status, a trait names the status it grants, and `wide open` needs a
// negative refusal no shipped trait declares. A bound nothing can reach is a bound
// nobody will fix.
func TestNoGlossedLogRowOutgrowsTheWindow(t *testing.T) {
	// The client leaves the last column empty so a full row cannot wrap, and the
	// log rows are drawn two cells in.
	const drawable = 79
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	glosses := i18n.Vi.LogGlosses(
		books.Skills.Skills(), books.Statuses.Kinds(), books.Passives.All())
	tags := map[string]string{"a": "A1", "f": "E1"}
	measure := func(label string, event battle.Event) {
		for _, row := range strings.Split(tui.Line(event, tags, glosses), "\n") {
			cells := lipgloss.Width("  " + row)
			if declared, known := knownWide[label]; known {
				if cells != declared {
					t.Errorf("%s measures %d cells, and is written down as %d: "+
						"update knownWide with the reason, or take the breach off it",
						label, cells, declared)
				}
				continue
			}
			if cells > drawable {
				t.Errorf("%s measures %d cells of the %d there are:\n%s",
					label, cells, drawable, row)
			}
		}
	}

	// amplified: a skill beside the status of its own condition, at that status's
	// stack cap and the power the skill amplifies to.
	for _, one := range books.Skills.Skills() {
		if one.Requires == nil || one.Requires.Status == "" {
			continue
		}
		if one.Requires.BonusPower == 0 {
			// A condition paid for in shape rather than in power emits no
			// amplified row at all — the arm exists to say a figure moved, and
			// this one moves none, so Act skips it and the Spread row below is
			// what that skill logs instead. Measuring it here would put the
			// widest row in the book on an event no battle can produce, which is
			// exactly the "reachable rather than worst-case" rule this test opens
			// with. It is not hypothetical: `electro_ball` beside a counter
			// capped at 999 measures 82 of the 79 there are.
			continue
		}
		stacks := 1
		if kind, err := books.Statuses.Lookup(one.Requires.Status); err == nil {
			stacks = kind.MaxStacks
		}
		measure(fmt.Sprintf("amplified/%s/%s", one.ID, one.Requires.Status), battle.Event{
			Kind: battle.Amplified, Actor: "a", Target: "f",
			Skill: one.ID, Status: one.Requires.Status, Stacks: stacks,
			Power: one.Power + one.Requires.BonusPower,
		})
	}
	// spread: the row the arm above skips, and the other place two glossed ids
	// share one clause. Every skill whose discharge can travel draws one.
	for _, one := range books.Skills.Skills() {
		if !one.Requires.ChainsOn() {
			continue
		}
		measure(fmt.Sprintf("spread/%s/%s", one.ID, one.Requires.Status), battle.Event{
			Kind: battle.Spread, Actor: "a", Target: "f",
			Skill: one.ID, Status: one.Requires.Status,
			Cell: hex.At(hex.Offset{Col: 3, Row: 1}),
		})
	}
	// skill_used and summoned: every skill.
	for _, one := range books.Skills.Skills() {
		measure("skill_used/"+one.ID, battle.Event{
			Kind: battle.SkillUsed, Actor: "a", Skill: one.ID, Gradient: 800,
			Cell: hex.At(hex.Offset{Col: 3, Row: 1}),
		})
		if one.Summons == nil {
			continue
		}
		measure("summoned/"+one.ID, battle.Event{
			Kind: battle.Summoned, Actor: "a", Skill: one.ID,
			Target: "enemy.2,2#1", Amount: 4800,
			Cell: hex.At(hex.Offset{Col: 3, Row: 1}),
		})
	}
	// The status-only arms, over every declared status.
	for _, kind := range books.Statuses.Kinds() {
		for _, arm := range []struct {
			name  string
			event battle.Event
		}{
			{"status_ticked", battle.Event{Kind: battle.StatusTicked, Amount: 4800, Stacks: 5}},
			{"healed", battle.Event{Kind: battle.Healed, Amount: 4800, Stacks: 5}},
			{"status_expired", battle.Event{Kind: battle.StatusExpired}},
			{"status_consumed", battle.Event{Kind: battle.StatusConsumed, Stacks: 5, Amount: 4800}},
			{"status_applied", battle.Event{Kind: battle.StatusApplied, Stacks: 5, Remaining: 6}},
			{"resisted/immune", battle.Event{Kind: battle.StatusResisted, Refused: 1000}},
			{"resisted/shrugs", battle.Event{Kind: battle.StatusResisted, Refused: 400, Chance: 1000}},
			{"resisted/plain", battle.Event{Kind: battle.StatusResisted, Chance: 1000}},
		} {
			event := arm.event
			event.Actor, event.Target, event.Status = "a", "f", kind.ID
			measure(arm.name+"/"+kind.ID, event)
		}
	}
	// The trait arms, each beside the statuses that trait actually names.
	for _, held := range books.Passives.All() {
		if held.Replies != nil && held.Replies.Power > 0 {
			// A reply is neutral and always lands, so it carries no affinity note.
			measure("damaged/reply/"+held.ID, battle.Event{
				Kind: battle.Damaged, Actor: "a", Target: "f", Passive: held.ID,
				Amount: 4800, Multiplier: 1000, Remaining: 4800,
			})
		}
		applied := make([]string, 0, 4)
		for _, application := range held.Applies {
			applied = append(applied, application.Status)
		}
		if held.Replies != nil {
			for _, application := range held.Replies.Applies {
				applied = append(applied, application.Status)
			}
		}
		for _, id := range applied {
			stacks, remaining := 1, 6
			if kind, err := books.Statuses.Lookup(id); err == nil {
				stacks, remaining = kind.MaxStacks, kind.Duration
			}
			measure(fmt.Sprintf("status_applied/%s/%s", held.ID, id), battle.Event{
				Kind: battle.StatusApplied, Actor: "a", Target: "f",
				Status: id, Passive: held.ID, Stacks: stacks, Remaining: int64(remaining),
			})
		}
		for _, grant := range held.Grants {
			for _, kind := range []battle.Kind{battle.PassiveHeld, battle.PassiveReleased} {
				measure(fmt.Sprintf("%s/%s/%s", kind, held.ID, grant.Status), battle.Event{
					Kind: kind, Actor: "a", Passive: held.ID,
					Status: grant.Status, Stacks: 1,
				})
			}
		}
	}
}

// TestNoGlossNestsInsideABracketOfItsOwnKind is why the gloss is written with
// angle brackets rather than round ones.
//
// A gloss is drawn in places that already carry a parenthetical of their own —
// `status_applied` names the trait a status came from as `(virulence)` — so a
// round gloss inside it read `(virulence (độc lực))`, a bracket inside a bracket
// for the reader to unpick. The gloss is the **inner** thing wherever the two
// meet, so the gloss is what changes shape, and it changed everywhere because
// i18n.GlossBracket is its one definition.
//
// ⚠️ It counts depth rather than looking for "((", which the nesting it exists to
// catch does not contain: the two brackets sat five words apart.
//
// ⚠️ The second half is the guard against a vacuous pass. A sweep that finds no
// nesting because no arm puts a gloss inside a parenthetical any more is a sweep
// measuring nothing, and it would go quiet the moment somebody reworded the arm
// this test was written for — so it fails unless some arm still does.
func TestNoGlossNestsInsideABracketOfItsOwnKind(t *testing.T) {
	glosses := sweepGlosses(t)
	tags := map[string]string{"a": "A1", "f": "E1"}
	pairs := []struct{ open, close rune }{{'(', ')'}, {'<', '>'}, {'[', ']'}}
	glossInsideAParenthetical := ""
	for _, arm := range sweepArms() {
		line := tui.Line(arm.event, tags, glosses)
		for _, pair := range pairs {
			depth, deepest := 0, 0
			for _, letter := range line {
				switch letter {
				case pair.open:
					depth++
					deepest = max(deepest, depth)
				case pair.close:
					depth--
				}
			}
			if deepest > 1 {
				t.Errorf("%s nests %c%c %d deep, so a reader has to unpick which "+
					"bracket closes what:\n%s", arm.name, pair.open, pair.close, deepest, line)
			}
		}
		// The case the test exists for: a gloss opening while a round bracket is
		// still open. That is the shape that used to nest.
		round := 0
		for _, letter := range line {
			switch letter {
			case '(':
				round++
			case ')':
				round--
			case '<':
				if round > 0 {
					glossInsideAParenthetical = arm.name
				}
			}
		}
	}
	if glossInsideAParenthetical == "" {
		t.Error("no arm draws a gloss inside a parenthetical any more, so this sweep " +
			"proves nothing: find the arm that replaced status_applied/trait and say " +
			"so here, or take the test off with the reason")
	}
}
