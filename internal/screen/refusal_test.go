package screen

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The row a reader cannot act on
//
// An option the turn cannot take draws a reason where its summary would go, and
// that reason used to be battle.Option.Reason: an English sentence built inside
// internal/core, drawn unchanged on a Vietnamese screen. It was survivable while
// every refusal was a cooldown — a countdown explains itself in any language —
// and it stopped being survivable when a skill gated on the caster's own reserve
// arrived, because a cooldownless skill greyed out explains itself with nothing
// at all.
//
// What is asserted below is the row as it is drawn, not the helper's return
// value alone: the sentence has to survive the column the row cuts it into, and
// the id column is the thing standing next to it.

// aGlossedStatus is a status the library names in Vietnamese, with an id that is
// no part of that name.
//
// Looked up rather than named, which is this package's fixture rule: a test that
// wrote `heft` down breaks the day somebody edits statuses.json for a reason that
// has nothing to do with refusals. Both halves of the guard matter — an id whose
// gloss contains it (or which has no gloss at all) would make "the row does not
// hold the id" pass for a row that names the id and nothing else.
func aGlossedStatus(t *testing.T, c Context) (id, gloss string) {
	t.Helper()
	for _, kind := range c.Lib.Statuses().Kinds() {
		name := c.Lang.Gloss(kind.ID)
		if name == "" || name == kind.ID || strings.Contains(name, kind.ID) {
			continue
		}
		return kind.ID, name
	}
	t.Fatalf("no status in the library is glossed under a name that is not its own id, "+
		"so a row naming one by its gloss could not be told from a row naming it by "+
		"its id (%s)", c.Lang)
	return "", ""
}

// aRefusedRow draws a turn offering exactly the given blocked option and hands
// back what the slot beside the id holds.
//
// The option is built from one the battle really offered, so the id column is a
// real id at its real width, and the block fields are then written over it. Its
// Reason is left holding the engine's English on purpose: a helper that fell
// through to it would be drawing a sentence these tests can recognise.
func aRefusedRow(t *testing.T, c Context, blocked battle.Option) string {
	t.Helper()
	p := atTheBattle(t, c)
	prompt := *p.Pending
	prompt.Options = []battle.Option{blocked}
	p.Pending = &prompt
	rows := strings.Split(strings.TrimRight(p.Choices(c), "\n"), "\n")
	if len(rows) != 2 {
		t.Fatalf("one option drew %d rows, want a heading and a row:\n%s",
			len(rows), strings.Join(rows, "\n"))
	}
	_, tail := optionColumns(p, rows[1])
	if tail == "" {
		t.Fatalf("the refused row drew nothing beside its id:\n%s", rows[1])
	}
	return tail
}

// blockedOnFuel is the option a gate produces: no aims, the reserve it is short
// of, and the two counts.
func blockedOnFuel(t *testing.T, c Context, status string, need, held int) battle.Option {
	t.Helper()
	p := atTheBattle(t, c)
	option := p.Pending.Options[0]
	option.Aims = nil
	option.Blocked = battle.BlockFuel
	option.Status, option.Need, option.Held = status, need, held
	option.Reason = "needs " + strconv.Itoa(need) + " stacks of " + status +
		", holding " + strconv.Itoa(held)
	return option
}

// TestARefusedOptionNamesItsFuelByItsGlossAndNotItsID is the first of the three
// facts the fuel row carries: *of what*.
//
// A data id is what the option carries, because internal/core knows nothing else
// — and a Vietnamese screen showing `heft` has told its reader the name of a
// variable. Every other data name this package draws goes through the gloss, and
// this row is not the exception.
func TestARefusedOptionNamesItsFuelByItsGlossAndNotItsID(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	id, gloss := aGlossedStatus(t, c)
	tail := aRefusedRow(t, c, blockedOnFuel(t, c, id, 5, 2))
	if !strings.Contains(tail, gloss) {
		t.Errorf("the refused row draws %q, which does not name the reserve by its "+
			"Vietnamese name %q", tail, gloss)
	}
	if strings.Contains(tail, id) {
		t.Errorf("the refused row draws %q, which names the reserve by its data id "+
			"%q rather than by %q", tail, id, gloss)
	}
}

// TestARefusedOptionSaysHowMuchIsNeededAndHowMuchIsHeld is the other two facts,
// and the reason they are asserted together is that either one alone is worth
// nothing.
//
// ⚠️ **"The row holds a 5" is not a measurement.** A helper that printed the
// count needed twice would draw "needs 5, holding 5" and satisfy every assertion
// about the number 5 there is. So the row is compared against the wording filled
// in from the option itself, and then drawn a second time with a different amount
// held: the two rows have to differ, which is the whole of "Held reads the
// caster" said as something that can fail.
func TestARefusedOptionSaysHowMuchIsNeededAndHowMuchIsHeld(t *testing.T) {
	const need, held = 5, 2
	if need == held {
		t.Fatal("the two counts are the same, so a row printing one of them twice " +
			"would pass")
	}
	vietnamese, _ := start(t, i18n.Vi)
	fuel, _ := aGlossedStatus(t, vietnamese)
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		// English shows a data id as the data writes it — Gloss's own rule, and
		// the fallback the row takes for an unglossed status in either language —
		// so the name in the sentence is the gloss where there is one and the id
		// where there is not.
		name := c.Lang.Gloss(fuel)
		if name == "" {
			name = fuel
		}
		tail := aRefusedRow(t, c, blockedOnFuel(t, c, fuel, need, held))
		if want := c.Text(i18n.PlayBlockedFuel, need, name, held); tail != want {
			t.Errorf("%s: the refused row draws %q, want %q", lang, tail, want)
		}
		deeper := aRefusedRow(t, c, blockedOnFuel(t, c, fuel, need, held+1))
		if deeper == tail {
			t.Errorf("%s: a caster holding %d and a caster holding %d draw the same row "+
				"%q, so what is held is not read off the caster",
				lang, held, held+1, tail)
		}
	}
}

// TestARefusedCooldownStillReadsAsACountdown is the regression the fuel wording
// could quietly cause.
//
// Three of the four refusals were readable before this family existed, and a
// switch that sent them all through the newest wording would be a fix that broke
// what was working. The cooldown is the one worth naming: it is the refusal a
// player meets every turn, and its whole content is the number.
func TestARefusedCooldownStillReadsAsACountdown(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	p := atTheBattle(t, c)
	const turns = 3
	option := p.Pending.Options[0]
	option.Aims = nil
	option.Blocked, option.Turns = battle.BlockCooldown, turns
	option.Reason = "3 turns of cooldown left"
	tail := aRefusedRow(t, c, option)
	if want := c.Text(i18n.PlayBlockedCooldown, turns); tail != want {
		t.Errorf("a cooling skill draws %q, want the countdown %q", tail, want)
	}
	if !strings.Contains(tail, strconv.Itoa(turns)) {
		t.Errorf("a cooling skill draws %q, which does not say how many turns are left", tail)
	}
	if tail == option.Reason {
		t.Errorf("a cooling skill draws %q, which is the engine's English sentence "+
			"rather than anything this screen worded", tail)
	}
}

// TestOneTurnOfCooldownIsItsOwnWording is the count English cannot spell with a
// plural rule, and it is the row a reader meets most often of the four: a skill
// is one turn from ready on more turns than it is three.
//
// ⚠️ **Both languages, and not because Vietnamese needs it.** Vietnamese has no
// plural, so "còn hồi 1 lượt" is what the general wording would have printed
// anyway — but the key exists in both because every key does, and asserting it
// here is what stops a later reader "simplifying" the Vietnamese half away and
// leaving a format verb with no argument.
func TestOneTurnOfCooldownIsItsOwnWording(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		p := atTheBattle(t, c)
		option := p.Pending.Options[0]
		option.Aims = nil
		option.Blocked, option.Turns = battle.BlockCooldown, 1
		tail := aRefusedRow(t, c, option)
		if want := c.Text(i18n.PlayBlockedCooldownOne); tail != want {
			t.Errorf("%s: one turn of cooldown draws %q, want %q", lang, tail, want)
		}
		// The defect this closes, named so the assertion cannot be read as taste:
		// the counted wording spells the number into a plural noun and drew
		// "1 turns of cooldown left".
		//
		// ⚠️ **Asserted only where the two wordings differ, which is English
		// alone.** Vietnamese has no plural, so its two keys are byte-identical
		// and "the singular is not the plural" is unaskable there — asking it
		// anyway reddens a correct screen, which is what a first draft of this
		// test did. That the question has no Vietnamese half is the whole reason
		// this is two keys instead of a plural rule.
		counted := c.Text(i18n.PlayBlockedCooldown, 1)
		if counted != c.Text(i18n.PlayBlockedCooldownOne) && tail == counted {
			t.Errorf("%s: one turn of cooldown draws the counted wording %q", lang, counted)
		}
		// And the count above one is untouched.
		option.Turns = 2
		if tail := aRefusedRow(t, c, option); tail != c.Text(i18n.PlayBlockedCooldown, 2) {
			t.Errorf("%s: two turns of cooldown draws %q, which is not the counted wording", lang, tail)
		}
	}
}

// TestEveryRefusalDrawsAWordingOfItsOwn is the sweep behind the three above: one
// wording per battle.Block, each row drawing its own, no two of them the same
// sentence, and none of them the engine's English on a Vietnamese screen.
//
// ⚠️ **It walks the enum rather than the table.** A block added to internal/core
// and not worded here would draw the English sentence — the whole thing this
// change exists to stop — and a test that ranged over what it holds would be
// asking itself whether it holds what it holds.
//
// ⚠️ **Distinctness alone measured almost nothing, and it was measured.** With
// every block handed the same option, a mutation routing the cooldown through the
// fuel wording still drew two *different* rows, because the fuel arm glosses the
// status and the mutated one did not — the rows differed by an accident of the
// gloss rather than by saying different things. So each row is compared against
// the wording its own block owns, filled in from that same option: the mapping is
// restated, but the sentences come from the catalog, so a swapped arm, a doubled
// arm and an unworded block all fail.
func TestEveryRefusalDrawsAWordingOfItsOwn(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	fuel, gloss := aGlossedStatus(t, c)
	const turns, need, held = 3, 5, 2
	wording := map[battle.Block]string{
		battle.BlockUnknownSkill: c.Text(i18n.PlayBlockedUnknown),
		battle.BlockCooldown:     c.Text(i18n.PlayBlockedCooldown, turns),
		battle.BlockFuel:         c.Text(i18n.PlayBlockedFuel, need, gloss, held),
		battle.BlockNoReach:      c.Text(i18n.PlayBlockedNoReach),
	}
	p := atTheBattle(t, c)
	base := p.Pending.Options[0]
	base.Aims = nil
	base.Reason = "unknown skill"
	base.Turns, base.Status, base.Need, base.Held = turns, fuel, need, held
	drawn := make(map[string]battle.Block)
	for block := battle.BlockNone + 1; block <= battle.BlockNoReach; block++ {
		want, declared := wording[block]
		if !declared {
			t.Errorf("block %d has no wording of its own here, so whatever the row draws "+
				"for it is measured by nothing", block)
			continue
		}
		option := base
		option.Blocked = block
		tail := aRefusedRow(t, c, option)
		if tail != want {
			t.Errorf("block %d draws %q, want %q", block, tail, want)
		}
		if tail == option.Reason {
			t.Errorf("block %d draws %q, the engine's own English", block, tail)
		}
		if first, seen := drawn[tail]; seen {
			t.Errorf("block %d and block %d both draw %q, so the row cannot tell the "+
				"two refusals apart", first, block, tail)
			continue
		}
		drawn[tail] = block
	}
	if len(drawn) != int(battle.BlockNoReach) {
		t.Errorf("%d of the %d blocks drew a wording of their own",
			len(drawn), battle.BlockNoReach)
	}
	for block := range wording {
		if block <= battle.BlockNone || block > battle.BlockNoReach {
			t.Errorf("a wording is held for block %d, which internal/core does not declare",
				block)
		}
	}
}
