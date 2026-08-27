package i18n_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// TestDescribeEveryShippedSkillGolden is the design record for the sentences,
// and the reason the descriptions are derived rather than authored: a number
// moving in skills.json moves a line here, and the diff is what says how the
// change reads to a player.
//
// It covers every shipped skill and trait in **both** languages rather than a
// chosen few in one. A sampled golden would leave the next skill's phrasing
// unmeasured, and phrasing is exactly what this file exists to hold still; one
// language would leave the other free to fall behind, which is the failure a
// second language has every time it is added to something already written.
func TestDescribeEveryShippedSkillGolden(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped passives: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}

	var b strings.Builder
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		fmt.Fprintf(&b, "== %s ==\n\n", lang)
		for _, declared := range skills.Skills() {
			fmt.Fprintf(&b, "%s\n%s\n\n", declared.ID, lang.Describe(declared, shapes))
		}
		for _, held := range passives.All() {
			fmt.Fprintf(&b, "[trait] %s\n%s\n\n", held.ID, lang.DescribePassive(held))
		}
		// Grouped, as every reference draws them: the order is the record too,
		// because a status moving between categories changes what a cleanse
		// naming that category strips.
		for _, group := range statuses.Grouped() {
			for _, kind := range group.Kinds {
				fmt.Fprintf(&b, "[status] %s\n%s\n\n", kind.ID, lang.DescribeStatus(kind))
			}
		}
	}

	got := b.String()
	path := filepath.Join("testdata", "describe.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/i18n -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the descriptions differ from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// TestEveryTraitDescriptionSaysEveryThingItDoes is the property a golden cannot
// hold: a golden freezes whatever was generated, including a trait the writer
// has nothing to say about.
//
// A trait accumulates halves — a grant, a resistance, a rider, a gate, a reply —
// and each was added by someone editing this file. The failure mode is not a bad
// sentence but a missing one: a shipped trait whose declaration says four things
// and whose description says three, which reads as a complete answer and is not.
// So this counts what the declaration promises rather than reading the prose.
func TestEveryTraitDescriptionSaysEveryThingItDoes(t *testing.T) {
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped passives: %v", err)
	}
	for _, lang := range i18n.Langs() {
		for _, held := range passives.All() {
			description := lang.DescribePassive(held)
			if strings.TrimSpace(description) == "" {
				t.Errorf("%s: %q has no description at all", lang, held.ID)
				continue
			}
			lines := len(strings.Split(strings.TrimSpace(description), "\n"))
			want := len(held.Grants) + len(held.Resists) + len(held.Applies)
			// The authored clause is a line of its own, and only in the language
			// it was authored in: an English reader gets the derived lines, the
			// same trade a skill's flavour makes.
			if held.Flavour != "" && lang == i18n.Vi {
				want++
			}
			if held.Replies.Answers() {
				want++
			}
			if held.Drains > 0 {
				want++
			}
			// An amplification is one entry and up to two lines, because the two
			// shares promise different things and a trait may carry either alone
			// — so the count is of shares rather than of entries.
			for _, raise := range held.Amplifies {
				if raise.Effect > 0 {
					want++
				}
				if raise.Chance > 0 {
					want++
				}
			}
			if held.While != nil {
				want++
			}
			if lines != want {
				t.Errorf("%s: %q declares %d things and is described in %d lines:\n%s",
					lang, held.ID, want, lines, description)
			}
		}
	}
}

// TestAGatedTraitIsNotDescribedAsAlways is the sentence a gate made wrong.
//
// A trait's grants read as "always carries" because until a grant could be gated
// that was simply true. A gated one closes with the line saying when it is in
// force, so the two sentences would have argued with each other — and a reader
// deciding whether to field the trait would have to work out which the engine
// meant.
func TestAGatedTraitIsNotDescribedAsAlways(t *testing.T) {
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped passives: %v", err)
	}
	// The wordings are matched by their opening clause rather than by a literal,
	// so this reads the catalog the same way the description does: a translation
	// reworded tomorrow keeps the test honest, where a copied English string
	// would quietly stop matching and pass.
	opening := func(text string) string {
		return strings.TrimSpace(strings.SplitN(text, "%", 2)[0])
	}
	gated := 0
	for _, held := range passives.All() {
		if held.While == nil || len(held.Grants) == 0 {
			continue
		}
		gated++
		for _, lang := range i18n.Langs() {
			description := lang.DescribePassive(held)
			if !strings.Contains(description, opening(lang.Text(i18n.BlurbTraitWhile))) {
				t.Errorf("%s: %q is gated and its description never says when: %q",
					lang, held.ID, description)
			}
			if strings.Contains(description, opening(lang.Text(i18n.BlurbTraitGrants))) {
				t.Errorf("%s: %q is gated and still says it always applies: %q",
					lang, held.ID, description)
			}
		}
	}
	if gated == 0 {
		t.Skip("no shipped trait both grants and is gated, so there is nothing here to contradict")
	}
}

// TestEverySkillDescriptionSaysWhatItDoes is the property a golden cannot hold:
// a golden freezes whatever was generated, including a skill the generator has
// nothing to say about. This asserts that every shipped skill, in every
// language, produces a description with an aim, a cost line, and no leftover id.
func TestEverySkillDescriptionSaysWhatItDoes(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		for _, declared := range skills.Skills() {
			description := lang.Describe(declared, shapes)
			if strings.TrimSpace(description) == "" {
				t.Errorf("%s: %q has no description at all", lang, declared.ID)
				continue
			}
			lines := strings.Split(description, "\n")
			if len(lines) < 2 {
				t.Errorf("%s: %q is described in one line, with no cost line under it: %q",
					lang, declared.ID, description)
			}
			// The last line is always the costs, and it always names the aim: a
			// range for anything that reaches, the self word for anything that
			// does not. Lowered before matching, because the line is capitalised
			// as a sentence and a test matching the capital would pass for the
			// wrong reason the day it stopped opening with this clause.
			costs := strings.ToLower(lines[len(lines)-1])
			if declared.Target == skill.Self {
				if !strings.Contains(costs, strings.ToLower(lang.Text(i18n.BlurbCostSelf))) {
					t.Errorf("%s: %q targets itself and its cost line does not say so: %q",
						lang, declared.ID, costs)
				}
			} else if !strings.Contains(costs, strings.ToLower(rangeWord(lang))) {
				t.Errorf("%s: %q reaches out and its cost line states no range: %q",
					lang, declared.ID, costs)
			}
			// Vietnamese only, and that is the whole point of the gloss tables
			// rather than an exemption: an id *is* the English name, so "poison"
			// in an English sentence is the reading working. The same word in a
			// Vietnamese one is a name nobody wrote.
			if lang != i18n.Vi {
				continue
			}
			for _, application := range declared.Applies {
				if strings.Contains(description, application.Status) {
					t.Errorf("%s: %q describes the status %q by its id, so it has no name in this language",
						lang, declared.ID, application.Status)
				}
			}
		}
	}
}

// rangeWord is the fixed half of the range clause, which is what a cost line has
// to contain. Taken from the wording rather than written twice: a test holding
// its own copy of a sentence is the drift it was meant to catch.
func rangeWord(lang i18n.Lang) string {
	word, _, _ := strings.Cut(lang.Say(i18n.BlurbCostRange, 0), " 0")
	return word
}

// TestEveryStatusDescriptionSaysWhatItIs is the property the golden cannot hold:
// a golden freezes whatever came out, a status the writer had nothing to say
// about included.
//
// Three things every one of them has to carry, and each is a way the reference
// has already been wrong somewhere else in this program. It has to say something
// beyond its cost line, or a category nobody wrote a sentence for reads as a
// status that does nothing. It has to close on the category, the duration and
// the stacks, because those are the three facts a player is looking one up for.
// And a permanent one must never print a duration: Snapshot.Permanent exists
// because a zero there reads as something about to expire.
func TestEveryStatusDescriptionSaysWhatItIs(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	for _, lang := range i18n.Langs() {
		for _, kind := range statuses.Kinds() {
			description := lang.DescribeStatus(kind)
			lines := strings.Split(description, "\n")
			if len(lines) < 2 {
				t.Errorf("%s: %q is described in one line, so it says nothing the cost line does not: %q",
					lang, kind.ID, description)
				continue
			}
			// Lowered before matching, because the line is capitalised as a
			// sentence and a test matching the capital would pass for the wrong
			// reason the day it stopped opening with the category.
			costs := strings.ToLower(lines[len(lines)-1])
			if !strings.Contains(costs, strings.ToLower(lang.StatusCategory(kind.Category.String()))) {
				t.Errorf("%s: %q closes without naming its category: %q", lang, kind.ID, costs)
			}
			if kind.Permanent {
				if !strings.Contains(costs, strings.ToLower(lang.Text(i18n.BlurbStatusAlways))) {
					t.Errorf("%s: %q is permanent and its cost line does not say so: %q",
						lang, kind.ID, costs)
				}
				continue
			}
			want := lang.Say(i18n.BlurbStatusLasts, kind.Duration)
			if kind.Duration == 1 {
				want = lang.Text(i18n.BlurbStatusLastsOne)
			}
			if !strings.Contains(costs, strings.ToLower(want)) {
				t.Errorf("%s: %q lasts %d turns and its cost line does not say so: %q",
					lang, kind.ID, kind.Duration, costs)
			}
		}
	}
}

// TestAPermanentStatusIsNeverGivenARate is the sentence a permanent status made
// wrong, and it is the mirror of TestAGatedTraitIsNotDescribedAsAlways.
//
// A stat term used to read "raises defence by 15% per stack" whatever carried
// it. On toughened and kindled — the two permanent statuses a trait grants, and
// the two a reader is least able to guess at — there can never be a second
// stack, so a rate is a promise of something the engine refuses: status.Set caps
// at MaxStacks, and Hold stops at the same cap.
func TestAPermanentStatusIsNeverGivenARate(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	// Matched by the fixed half of the wording rather than by a literal, so a
	// translation reworded tomorrow keeps this honest: whatever the per-stack
	// wording says after its last blank is the clause the other one does not
	// have.
	rate := func(lang i18n.Lang) string {
		text := lang.Text(i18n.BlurbStatusRaises)
		last := strings.LastIndex(text, "%s")
		if last < 0 {
			return ""
		}
		return strings.TrimSpace(text[last+len("%s"):])
	}
	single := 0
	for _, kind := range statuses.Kinds() {
		if kind.MaxStacks > 1 || len(kind.Modifiers) == 0 {
			continue
		}
		single++
		for _, lang := range i18n.Langs() {
			suffix := rate(lang)
			if suffix == "" {
				t.Fatalf("%s: the two stat wordings share no ending, so this test cannot tell them apart", lang)
			}
			if strings.Contains(lang.DescribeStatus(kind), suffix) {
				t.Errorf("%s: %q allows one stack and is described at a rate per stack:\n%s",
					lang, kind.ID, lang.DescribeStatus(kind))
			}
		}
	}
	if single == 0 {
		t.Skip("no shipped status carries a stat term and caps at one stack")
	}
}

// TestAStatusIsDescribedOverItsLifeAndNotOnlyPerTurn is the comparison the
// per-turn figure gets backwards.
//
// Poison ticks for 50% and burn for 80%, so a reference printing the tick alone
// has a reader rating burn the heavier of the two. Over their lives poison is
// 150% to burn's 160% for one stack and 450% to 320% at their caps, which is the
// other way round at the only place it matters. So the life is asserted to be
// there and to be the right number, in both languages.
func TestAStatusIsDescribedOverItsLifeAndNotOnlyPerTurn(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	ticking := 0
	for _, kind := range statuses.Kinds() {
		if kind.TickPower == 0 {
			continue
		}
		ticking++
		life := kind.TickPower * kind.Duration
		for _, lang := range i18n.Langs() {
			description := lang.DescribeStatus(kind)
			want := lang.Say(i18n.BlurbStatusLife, forge.Percent(life))
			if kind.MaxStacks > 1 {
				want = lang.Say(i18n.BlurbStatusLifeCapped,
					forge.Percent(life), kind.MaxStacks, forge.Percent(life*kind.MaxStacks))
			}
			if !strings.Contains(description, want) {
				t.Errorf("%s: %q is not described over its life; wanted %q in:\n%s",
					lang, kind.ID, want, description)
			}
		}
	}
	if ticking == 0 {
		t.Skip("no shipped status ticks, so nothing here has a life to state")
	}
}

// TestATraitsSharesArePrintedExactly is the bug traits found and skills never
// could.
//
// A share used to be truncated to whole percent, which is lossless for a skill
// and lossy for a trait: a skill is priced in hundreds of parts per thousand and
// a trait in tens, so venom_blood's reply chance of 25 printed as "2%" when it is
// 2.5, and anything under ten would have printed "0%" — a feature reading as one
// that does not work. The argument for truncating was that the listing beside the
// sentence carries the exact figure, and for a trait there is no such listing:
// hexforge passives has no column for a reply or for a drain.
//
// So every share a trait declares has to appear in its description as the figure
// itself. An immunity is the one exception, and it is a wording rather than a
// figure: a full share reads "refuses it outright", which is the fact rather than
// the number.
func TestATraitsSharesArePrintedExactly(t *testing.T) {
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped passives: %v", err)
	}
	for _, held := range passives.All() {
		shares := make([]int, 0, 6)
		if held.Replies.Answers() {
			if held.Replies.Power > 0 {
				shares = append(shares, held.Replies.Power)
			}
			for _, application := range held.Replies.Applies {
				shares = append(shares, application.Chance)
			}
		}
		for _, application := range held.Applies {
			shares = append(shares, application.Chance)
		}
		for _, resistance := range held.Resists {
			if resistance.Amount < scale.Base {
				shares = append(shares, resistance.Amount)
			}
		}
		if held.Drains > 0 {
			shares = append(shares, held.Drains)
		}
		for _, raise := range held.Amplifies {
			if raise.Effect > 0 {
				shares = append(shares, raise.Effect)
			}
			if raise.Chance > 0 {
				shares = append(shares, raise.Chance)
			}
		}
		if held.While != nil {
			shares = append(shares, held.While.BelowHealth)
		}
		for _, lang := range i18n.Langs() {
			description := lang.DescribePassive(held)
			for _, permille := range shares {
				if !strings.Contains(description, forge.Percent(permille)) {
					t.Errorf("%s: %q declares %d parts per thousand and its description never says %q:\n%s",
						lang, held.ID, permille, forge.Percent(permille), description)
				}
			}
		}
	}
}
