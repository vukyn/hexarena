package tui_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// mainRosterRow and mainRosterHeadingRow are the two format strings this table
// had on main, copied verbatim rather than referred to.
//
// ⚠️ **Referring to the package's own constants would measure nothing.** The
// claim this pair exists to hold is that the narrow table draws the same bytes it
// drew before the wide one arrived, and a copy that reads `rosterRow` agrees with
// whatever `rosterRow` becomes. So the literals are here, and an edit to the
// shipped ones is a red test that has to be answered rather than accepted.
//
// They are marked for deletion the day the narrow table is deliberately changed,
// exactly as combat's `narrowSwung` is: a frozen copy of a thing nobody intends
// to freeze forever.
const (
	mainRosterRow        = "%-17s%-26s%4d   %s\n"
	mainRosterHeadingRow = "%-17s%-26s%-4s   %s"
)

// mainRoster is tui.Roster as main drew it, written out whole.
//
// It calls HealthBar and Effects rather than copying those too, because neither
// is what this change touched and a copy of them would freeze two things this
// test is not about — but every line that *is* about the table's shape, the unit
// column and the trim included, is here rather than borrowed.
func mainRoster(lang i18n.Lang, fight *battle.Battle, tags map[string]string) string {
	var b strings.Builder
	heading := fmt.Sprintf(mainRosterHeadingRow,
		lang.Text(i18n.RosterHeadingUnit), "hp", "spd",
		lang.Text(i18n.RosterHeadingEffects))
	b.WriteString(heading + "\n")
	for _, unit := range fight.Units() {
		state := tui.HealthBar(unit.HP, fight.MaxHP(unit))
		if unit.Dead {
			state = "fallen"
		}
		stats := fight.Stats(unit)
		fmt.Fprintf(&b, mainRosterRow,
			fmt.Sprintf("%-2s %s", tags[unit.ID], unit.Name), state,
			stats[progression.Speed], tui.Effects(unit))
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// TestTheNarrowRosterDrawsWhatMainDrewToTheByte is the promise the whole shape of
// this change rests on.
//
// Option A was chosen precisely because nothing is cut: a player on a terminal at
// the floor keeps the table they had, with the effects column at its full sixty
// cells and the health bar untouched. That is a claim about *bytes*, and the way
// to hold it is against the old code rather than against a golden — a golden is
// accepted, and a golden accepted by the same change it is meant to police proves
// nothing.
//
// Both languages, because the heading is the one line a language changes, and
// over a battle that has been fought rather than an opening board, so that a
// fallen unit, a spent bar and an elided effects column are all in the sample.
//
// ⚠️ **What is frozen is the table's SHAPE, not its wording, and the difference
// now matters.** The copy above reads the heading out of the catalog exactly as
// the shipped code does, so a *wording* change moves both sides together and
// this stays green — which is right: dropping the parenthesis from the effects
// heading was asked for, and a test that went red on it would be freezing a
// decision nobody made here. What it does catch is a column moving, a width
// changing, a verb changing or a field arriving, in either the row or the
// heading, because those are the literals and they are copied rather than read.
// The heading's own wording is held by TestTheRosterHeadingNoLongerCarriesTheRule
// in internal/screen instead.
func TestTheNarrowRosterDrawsWhatMainDrewToTheByte(t *testing.T) {
	var opened string
	for _, one := range []struct {
		name  string
		turns int
	}{
		{name: "an opening board"},
		{name: "a battle underway", turns: 40},
	} {
		fight, tags := opening(t)
		// Some turns, so the second sample holds hurt bars and effects that have
		// counted down rather than ten full ones.
		for range one.turns {
			if fight.Finished() || !stepped(fight) {
				break
			}
		}
		for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
			want := mainRoster(lang, fight, tags)
			got := tui.Roster(lang, fight, tags)
			if got != want {
				t.Errorf("%v, %s: the narrow roster is no longer what main drew\n"+
					"main draws:\n%s\nit now draws:\n%s", lang, one.name, want, got)
			}
			if lang != i18n.En {
				continue
			}
			// The premise for the second case, held rather than assumed: if the
			// turns moved nothing, both rows of this table drew the same opening
			// board and the hurt, dead and counted-down states were never
			// compared at all.
			if one.turns == 0 {
				opened = got
				continue
			}
			if got == opened {
				t.Fatalf("%d turns moved nothing on the roster, so this compared the "+
					"opening board twice", one.turns)
			}
		}
	}
}

// stepped takes one turn however the battle in front will let it, and says
// whether it took one.
//
// The renderer's fixtures are boards rather than scripts, so this asks the engine
// for its own suggestion rather than deciding anything — what is wanted is a
// battle that has moved, not a battle that went a particular way.
func stepped(fight *battle.Battle) bool {
	prompt := fight.Pending()
	if prompt == nil {
		if _, err := fight.Advance(); err != nil {
			return false
		}
		return true
	}
	decision, ok := fight.Suggest(prompt)
	if !ok {
		return fight.Pass(battle.NoActionReason) == nil
	}
	return fight.Act(decision.Skill, decision.Aim) == nil
}

// TestTheWideRosterCarriesTheElementAttackAndDefenceOfEveryUnit is what the three
// columns are for: a player deciding who to hit first can read the matchup and
// the two numbers off the row rather than opening each unit.
//
// Asserted per unit against the engine's own answers — unit.Affinity and
// fight.Stats — rather than against an expected table, for the reason the effects
// test gives: a fixture spelling the rows out would go red for a bench re-kit, a
// rename or a stat retune, none of which is what this is about.
//
// ⚠️ **The dual-element case is the one worth naming**, because a single element
// is what a column sized for two would draw correctly while joining two of them
// wrongly, and `gro/met` is a string neither half of which is a row's own.
//
// ⚠️ The element is asserted as the **codes**, derived from the unit's affinity
// rather than written down: the column draws `gro/met` where it used to draw
// `ground/metal`, and a walk still looking for the spelled-out name would go red
// on a correct column. What it must not become is a copy of elementCell, which
// would agree with a broken one — so it joins the codes here and lets the
// separator be the only thing both sides share.
func TestTheWideRosterCarriesTheElementAttackAndDefenceOfEveryUnit(t *testing.T) {
	fight, tags := opening(t)
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		lines := strings.Split(tui.RosterWide(lang, fight, tags, nil), "\n")
		if got, want := len(lines)-1, len(fight.Units()); got != want {
			t.Fatalf("%v: the wide roster drew %d rows for %d units", lang, got, want)
		}
		if heading := lang.Text(i18n.RosterHeadingElement); !strings.Contains(lines[0], heading) {
			t.Errorf("%v: the heading line %q does not name the element column %q",
				lang, lines[0], heading)
		}
		duals := 0
		for index, unit := range fight.Units() {
			row := lines[index+1]
			stats := fight.Stats(unit)
			if unit.Affinity.IsDual() {
				duals++
			}
			codes := make([]string, 0, 2)
			for _, member := range unit.Affinity.Elements() {
				codes = append(codes, tui.ElementCode(member))
			}
			for _, want := range []string{
				strings.Join(codes, "/"),
				strconv.FormatInt(stats[progression.Attack], 10),
				strconv.FormatInt(stats[progression.Defense], 10),
			} {
				if !strings.Contains(row, want) {
					t.Errorf("%v: %s's row does not carry %q:\n%s", lang, unit.ID, want, row)
				}
			}
		}
		// The premise: the bench really does field a unit carrying two elements,
		// so the joined form was drawn rather than only the single one.
		if duals == 0 {
			t.Fatalf("%v: no unit on the bench carries two elements, so the dual case was "+
				"never drawn", lang)
		}
	}
}

// TestTheWideRosterIsTheNarrowOneWithThreeColumnsAdded holds the other half of
// "nothing is cut", which the byte-identity test above cannot see: that test says
// the narrow table is unchanged, and this says the wide one did not pay for its
// new columns out of the old ones.
//
// The effects column is the one at risk — it is the only capped column on the
// row, so it is the obvious place to take room from — and the health bar is the
// other thing a brief of this shape usually loses. Both are asserted to be drawn
// in the wide table exactly as the narrow table draws them.
func TestTheWideRosterIsTheNarrowOneWithThreeColumnsAdded(t *testing.T) {
	fight, tags := opening(t)
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		narrow := strings.Split(tui.Roster(lang, fight, tags), "\n")[1:]
		wide := strings.Split(tui.RosterWide(lang, fight, tags, nil), "\n")[1:]
		if len(narrow) != len(wide) {
			t.Fatalf("%v: the two tables drew %d and %d rows", lang, len(narrow), len(wide))
		}
		carried := 0
		for index, unit := range fight.Units() {
			effects := tui.Effects(unit)
			if strings.Contains(effects, "+") {
				t.Errorf("%v: %s's effects are elided in the fixture (%q), so the two rows "+
					"cannot be compared on it", lang, unit.ID, effects)
			}
			if effects != "-" {
				carried++
			}
			if !strings.Contains(wide[index], effects) {
				t.Errorf("%v: %s's wide row dropped effects the narrow row draws (%q):\n%s",
					lang, unit.ID, effects, wide[index])
			}
			bar := tui.HealthBar(unit.HP, fight.MaxHP(unit))
			if !unit.Dead && !strings.Contains(wide[index], bar) {
				t.Errorf("%v: %s's wide row dropped the health bar (%q):\n%s",
					lang, unit.ID, bar, wide[index])
			}
		}
		// The premise: somebody on the bench actually holds an effect, or the
		// effects half of this compared "-" against "-" ten times.
		if carried == 0 {
			t.Fatalf("%v: nobody on the bench holds an effect, so the effects column was "+
				"never compared", lang)
		}
	}
}
