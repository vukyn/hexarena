package i18n_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/testfixture"
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
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the shipped chart: %v", err)
	}

	var b strings.Builder
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		fmt.Fprintf(&b, "== %s ==\n\n", lang)
		for _, declared := range skills.Skills() {
			// The two readings of one skill, one under the other. The compact
			// line is a fourth describer, so the golden is where the two are
			// compared by eye: a balance change has to move both, and a diff
			// where only one of them moved is the drift the containment test
			// below cannot see (it holds the figures, not the wordings).
			fmt.Fprintf(&b, "%s\n%s\n[one line] %s\n\n", declared.ID,
				lang.Describe(declared, shapes), lang.SummariseSkill(declared, shapes))
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
		// The chart last, and every member of it including the inert one. It is
		// the design record with the widest blast radius in the file: an edge
		// moving here rescales every damage figure above it at once, and the
		// diff is the only place that shows which way round.
		for _, member := range element.All() {
			fmt.Fprintf(&b, "[element] %s\n%s\n\n", member, lang.DescribeElement(member, chart))
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
			want := len(held.Grants) + len(held.Resists) + len(held.Applies) + len(held.Renews)
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
			if held.Converts > 0 {
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

// TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot is what a fourth
// describer is on probation for.
//
// The standing rule of this package is that a second reading of one set of facts
// drifts from the first, and Lang.SummariseSkill is a second reading of every
// number Lang.Describe reads. Its doc comment says why it cannot be Describe with
// the prose dropped — in Vietnamese the authored flavour and the damage figure
// are fused into one sentence with no seam to cut — and this is the other half of
// that bargain: the two are allowed to word a skill differently and are not
// allowed to disagree about a figure.
//
// So it compares digit runs rather than words. Every run the compact line prints
// has to appear in the sentences for the same skill, in the same language, over
// every shipped skill. One way round only: Describe carries figures the compact
// line leaves out on purpose (accuracy, pierce, a critical chance, the
// per-strike share of a volley), and demanding those back would be demanding the
// compact line be the long one.
//
// What it catches is the failure that matters: a power, a chance or a cooldown
// read off a different field, or through a different rounding, than the sentence
// beside it. A player comparing four rows and then pressing ? has to get the same
// numbers twice.
func TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot(t *testing.T) {
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
			summary := lang.SummariseSkill(declared, shapes)
			if strings.TrimSpace(summary) == "" {
				t.Errorf("%s: %q has no one-line summary at all", lang, declared.ID)
				continue
			}
			described := digitRuns(lang.Describe(declared, shapes))
			for _, run := range digitRuns(summary) {
				if !slices.Contains(described, run) {
					t.Errorf("%s: %q summarises as %q, whose figure %q is nowhere in "+
						"its description — the two readings disagree",
						lang, declared.ID, summary, run)
				}
			}
		}
	}
}

// digitRuns is every unbroken run of digits in a string, in the order they are
// printed.
//
// A run rather than a character, because 130 and 30 are different figures and a
// per-character comparison would call the first one contained in the second. A
// slice rather than a set, so a failure names the clauses in the order a reader
// meets them: this package holds the same line about map iteration reaching an
// output that internal/core does.
func digitRuns(text string) []string {
	var found []string
	current := strings.Builder{}
	flush := func() {
		if current.Len() > 0 {
			found = append(found, current.String())
		}
		current.Reset()
	}
	for _, letter := range text {
		if letter >= '0' && letter <= '9' {
			current.WriteRune(letter)
			continue
		}
		flush()
	}
	flush()
	return found
}

// TestAStripIsCalledHarmfulOnlyWhenEveryCategoryItNamesIs is the claim the second
// strips wording exists to make, and the one thing a width measurement cannot
// check: the tripwire executes the branch, it never reads what came out.
//
// The compact line counts a strip rather than enumerating it — three categories
// cannot fit a row — so the only thing left to say about it is *which kind*. That
// is read off status.Category.Harmful, the function that separates a cleanse from
// a dispel, and it is a claim: "gỡ 3 hiệu ứng xấu" on a skill that strips a shield
// is a description of a mechanic the skill does not have. Five of the eight
// categories are harmful and three are not, so the wrong answer is reachable
// rather than theoretical.
//
// ⚠️ **Both branches have to be reached or this proves nothing**, which is the
// whole shape of the test rather than a nicety: a bare loop over whatever the book
// holds would pass with a table covering one side, and a table silently covering
// one side is exactly how purify shipped invisible. Today the shipped rapid_spin
// and the fixture purify are all-harmful and the fixture unmake strips a buff and
// a shield, so both are live — and the day somebody empties one, this says so
// instead of going quiet.
//
// Over the books a loaded library holds rather than a named list, for the same
// reason: unmake is a fixture skill, so a test reading only the shipped book would
// have no benign case at all, and one naming ids by hand would not notice a shipped
// dispel being authored.
func TestAStripIsCalledHarmfulOnlyWhenEveryCategoryItNamesIs(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	skills := everySkillDeclared(t, shapes)
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		claimed, counted := 0, 0
		for _, declared := range skills {
			if declared.Strips == nil {
				continue
			}
			if len(declared.Strips.Categories) == 0 {
				t.Errorf("%s: %q strips nothing at all, so it names no kind",
					lang, declared.ID)
				continue
			}
			harmful := true
			for _, category := range declared.Strips.Categories {
				if !category.Harmful() {
					harmful = false
				}
			}
			// Both clauses rendered with this skill's own count, asked of the
			// catalog rather than written out here: a test holding its own copy of
			// a sentence is the drift it exists to catch.
			harmfulWording := lang.Say(i18n.SummaryStripsHarmful, declared.Strips.Stacks)
			countWording := lang.Say(i18n.SummaryStrips, declared.Strips.Stacks)
			if harmfulWording == countWording {
				t.Fatalf("%s words both strip clauses %q, so they cannot be told apart",
					lang, countWording)
			}
			summary := lang.SummariseSkill(declared, shapes)
			// The harmful one is tested first because the counting one is a
			// **prefix** of it in both languages — "gỡ 2 hiệu ứng" inside
			// "gỡ 2 hiệu ứng xấu", "strips 2" inside "strips 2 harmful" — so
			// asking about the count first would be true of every strip there is.
			said := strings.Contains(summary, harmfulWording)
			if said {
				claimed++
			} else {
				counted++
			}
			if said != harmful {
				t.Errorf("%s: %q strips %v and summarises as %q — it %s them harmful "+
					"and %s of them is",
					lang, declared.ID, declared.Strips.Categories, summary,
					map[bool]string{true: "calls", false: "does not call"}[said],
					map[bool]string{true: "every one", false: "not every one"}[harmful])
			}
			if !said && !strings.Contains(summary, countWording) {
				t.Errorf("%s: %q summarises as %q, which is neither of the two strip "+
					"clauses", lang, declared.ID, summary)
			}
		}
		if claimed == 0 {
			t.Errorf("%s: nothing in the books strips only harmful categories, so the "+
				"claim %q was never made and this measured nothing",
				lang, lang.Say(i18n.SummaryStripsHarmful, 1))
		}
		if counted == 0 {
			t.Errorf("%s: everything in the books strips only harmful categories, so "+
				"the plain count %q was never reached and a wrong claim would pass here",
				lang, lang.Say(i18n.SummaryStrips, 1))
		}
		t.Logf("%s: %d skills call a strip harmful and %d only count it",
			lang, claimed, counted)
	}
}

// TestNoEnglishStripsClauseNamesACategoryEnum is the regression, and it is about
// a **bare word** rather than a substring: "shielding" holds "shield" and is the
// wording working, while "shield" alone is the Go enum on a player-facing line.
//
// `rapid_spin` is the shipped skill that reached it — it read "Strips 1 stack of
// stat_debuff and dot." — because the clause used to word each category with
// Gloss, which answers in Vietnamese only by design. It is named here rather
// than left to the sweep so the regression is checked by name, and the sweep is
// beside it so the next stripping skill authored is checked too.
func TestNoEnglishStripsClauseNamesACategoryEnum(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	shipped, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	spelt := func(t *testing.T, declared skill.Skill) {
		t.Helper()
		if declared.Strips == nil {
			t.Fatalf("%q strips nothing, so its description measures none of this",
				declared.ID)
		}
		description := i18n.En.Describe(declared, shapes)
		for category := status.Category(0); int(category) < status.CategoryCount; category++ {
			name := category.String()
			for _, word := range bareWords(description) {
				if word == name {
					t.Errorf("the English %q reads %q, which names the %q enum",
						declared.ID, description, name)
					break
				}
			}
		}
	}
	named, err := shipped.Lookup("rapid_spin")
	if err != nil {
		t.Fatalf("look up rapid_spin: %v", err)
	}
	spelt(t, named)
	swept := 0
	for _, declared := range shipped.Skills() {
		if declared.Strips == nil {
			continue
		}
		spelt(t, declared)
		swept++
	}
	if swept == 0 {
		t.Fatal("nothing shipped strips anything, so the sweep measured nothing")
	}
}

// TestTheStripsClauseReadsInBothOfItsFrames pins the English wording of a
// category as a whole sentence, in the two frames the clause has.
//
// The literals are the point. A category noun has to survive "1 stack of" and "2
// stacks of" alike — an article or a plural reads wrong in one of them — and a
// table of seven words cannot say that about itself, so the two worst-reading
// combinations are written out here rather than derived. A wording change that
// breaks either frame fails on the sentence a player would have read.
//
// The two skills are built here rather than taken from the books because no
// shipped skill strips more than one stack, so the second frame has no shipped
// carrier at all.
func TestTheStripsClauseReadsInBothOfItsFrames(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	declared, err := skill.ParseBook([]byte(`{"skills":[
      {"id":"one_stack","element":"neutral","range":0,"pattern":"single",
       "power":0,"strikes":0,"accuracy":1000,"cooldown":3,"target":"self",
       "strips":{"categories":["stat_debuff","dot"],"stacks":1}},
      {"id":"two_stacks","element":"neutral","range":1,"pattern":"single",
       "power":0,"strikes":0,"accuracy":1000,"cooldown":3,"target":"enemy",
       "strips":{"categories":["shield","regen"],"stacks":2}}
    ]}`), skill.Deps{Patterns: shapes, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the two strips: %v", err)
	}
	for _, want := range []struct{ id, sentence string }{
		{"one_stack", "Strips 1 stack of stat reduction and damage over time."},
		{"two_stacks", "Strips 2 stacks of shielding and healing over time."},
	} {
		carried, err := declared.Lookup(want.id)
		if err != nil {
			t.Fatalf("look up %s: %v", want.id, err)
		}
		if got := i18n.En.Describe(carried, shapes); !strings.Contains(got, want.sentence) {
			t.Errorf("the English %s reads %q, which does not hold %q",
				want.id, got, want.sentence)
		}
	}
}

// TestAConditionsStackCountReadsAsAFloorRatherThanAnAmount is the ambiguity that
// sat unexercised for as long as no shipped condition asked for more than one.
//
// The two counts in this package's sentences read the same and mean opposite
// things. An APPLICATION's is exact — "puts charge x2 on" is two stacks and no
// more — while a CONDITION's is a floor: the skill wants *at least* that many and
// is happy with a pile. Rendering both as "x2" was harmless while every shipped
// condition asked for one, because a threshold of one renders as nothing at all.
//
// `overload` is the first to ask for two, and it is the skill where the ambiguity
// bites hardest: it would have read "carrying charge x2" and then "spends every
// stack it had", leaving a reader no way to tell whether the two was the
// requirement or the payment. One skill here declares both counts over the same
// status at the same number, which is the only shape that can catch a renderer
// spelling them alike.
func TestAConditionsStackCountReadsAsAFloorRatherThanAnAmount(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	declared, err := skill.ParseBook([]byte(`{"skills":[
      {"id":"both_counts","element":"neutral","range":1,"pattern":"single",
       "power":1000,"strikes":1,"accuracy":1000,"cooldown":3,"target":"enemy",
       "applies":[{"status":"poison","chance":1000,"stacks":3}],
       "requires":{"status":"poison","min_stacks":3,"bonus_power":500}}
    ]}`), skill.Deps{Patterns: shapes, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the probe: %v", err)
	}
	carried, err := declared.Lookup("both_counts")
	if err != nil {
		t.Fatalf("look up the probe: %v", err)
	}
	for _, lang := range i18n.Langs() {
		described := lang.Describe(carried, shapes)
		applied, required := "", ""
		for _, line := range strings.Split(described, "\n") {
			switch {
			case strings.Contains(line, lang.Say(i18n.BlurbAtLeast, 3, lang.Gloss("poison"))):
				required = line
			case strings.Contains(line, lang.Gloss("poison")+" x3"),
				strings.Contains(line, "poison x3"):
				applied = line
			}
		}
		if applied == "" {
			t.Errorf("%s: nothing in %q reads as three stacks being put on, so this measures no exact count at all",
				lang, described)
		}
		if required == "" {
			t.Errorf("%s: nothing in %q reads as three stacks being *required*, so the floor is still spelled like an amount:\n%s",
				lang, described, described)
		}
		if applied != "" && applied == required {
			t.Errorf("%s: the same line says what is put on and what is asked for:\n%s", lang, applied)
		}
	}
}

// bareWords splits prose into the words a reader sees, keeping the underscore
// because an enum spelling holds one — splitting on it would look for
// "stat_debuff" and never find it, while finding "stat" everywhere.
func bareWords(text string) []string {
	return strings.FieldsFunc(text, func(letter rune) bool {
		return !unicode.IsLetter(letter) && !unicode.IsDigit(letter) && letter != '_'
	})
}

// everySkillDeclared is the shipped book plus the fixture's, which is the set a
// loaded library holds.
//
// The fixture is parsed exactly the way testfixture.Inject parses it — through
// the skill book's own parser, against the shipped pattern and status books — so
// this is the same set of skills, without a temporary directory and a third copy
// of a data-copying helper to keep in step.
func everySkillDeclared(t *testing.T, shapes *pattern.Book) []skill.Skill {
	t.Helper()
	shipped, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	added, err := skill.ParseBook([]byte(`{"skills":`+testfixture.Skills+`}`),
		skill.Deps{Patterns: shapes, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the fixture skills: %v", err)
	}
	return append(shipped.Skills(), added.Skills()...)
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
			// The duration is spelled the same way here as the line under it
			// spells it, singular included: a life stated over "1 lượt" and a
			// cost line saying "1 turn" would be two names for one number.
			lasts := lang.Say(i18n.BlurbStatusLasts, kind.Duration)
			if kind.Duration == 1 {
				lasts = lang.Text(i18n.BlurbStatusLastsOne)
			}
			want := lang.Say(i18n.BlurbStatusLife, lasts, shareOf(life))
			if kind.MaxStacks > 1 {
				want = lang.Say(i18n.BlurbStatusLifeCapped,
					lasts, shareOf(life), shareOf(life*kind.MaxStacks))
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

// TestATraitsSharesAreRoundedAndNotTruncated is the bug traits found and skills
// never could, kept after the answer to it changed.
//
// A share was once truncated, which is lossless for a skill — priced in hundreds
// of parts per thousand — and lossy for a trait, priced in tens: venom_blood's
// reply chance of 25 printed as "2%" for 2.5, a fifth of the value gone. The
// sentence rounds now rather than printing the tenth, so what is asserted is the
// rounded figure and, in the next test, that nothing rounds to nothing. Rounding
// half away from zero is what keeps it from being the old truncation under a new
// name: 25 becomes 3.
//
// An immunity is the one exception, and it is a wording rather than a figure: a
// full share reads "miễn nhiễm", which is the fact rather than the number.
func TestATraitsSharesAreRoundedAndNotTruncated(t *testing.T) {
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
				// The magnitude, because a share's SIGN is carried by the verb
				// rather than by the figure: a resistance of -300 prints as
				// "Tăng 30% khả năng dính choáng" / "Takes 30% more". Rounding
				// the signed value gives -29% — Go truncates a negative
				// quotient towards zero — which is a figure no description has
				// ever printed.
				//
				// ⚠️ This went unnoticed until 2026-08-31 because vulnerability
				// is the one half of `resists` with no shipped user, so `shares`
				// had never held a negative. The first trait to declare one made
				// every language fail here at once.
				magnitude := permille
				if magnitude < 0 {
					magnitude = -magnitude
				}
				rounded := fmt.Sprintf("%d%%", (magnitude+5)/10)
				if !strings.Contains(description, rounded) {
					t.Errorf("%s: %q declares %d parts per thousand and its description never says %q:\n%s",
						lang, held.ID, permille, rounded, description)
				}
				// And never the truncation this replaced, which is a different
				// number whenever the tenth is five or more.
				if truncated := fmt.Sprintf("%d%%", magnitude/10); truncated != rounded &&
					strings.Contains(description, truncated) {
					t.Errorf("%s: %q declares %d parts per thousand and its description says %q, the truncation:\n%s",
						lang, held.ID, permille, truncated, description)
				}
			}
		}
	}
}

// shareOf is how a description writes a proportion: rounded to a whole percent,
// half away from zero. Written here rather than imported because i18n.share is
// unexported, and written as arithmetic rather than as a literal so a test
// asserting a figure cannot quietly assert the wrong one.
func shareOf(permille int) string {
	if permille < 0 {
		return "-" + shareOf(-permille)
	}
	return fmt.Sprintf("%d%%", (permille+5)/10)
}

// TestAListOfThreeTakesCommasAndOneConjunction is the grammar of every list this
// package reads out, asserted at each length rather than at the one the shipped
// data happens to produce.
//
// The defect it closes put the conjunction between *every* pair, so three items
// read "a and b and c". What hid it is that no list the shipped books produce has
// a third item — both goldens hold zero of them — so the golden could not have
// moved and this table is the only thing standing between the rule and a
// regression to it.
//
// Both joiners are in it. They keep separate conjunction keys on purpose (one is
// prose, one is punctuation between untranslated ids) and that is exactly why the
// comma rule has to be asserted through both: identical wordings that are declared
// twice are the drift this repository keeps a list of.
func TestAListOfThreeTakesCommasAndOneConjunction(t *testing.T) {
	for _, lang := range i18n.Langs() {
		and, comma := lang.Text(i18n.BlurbAnd), lang.Text(i18n.ListComma)
		ids := lang.Text(i18n.ElementJoiner)
		for _, c := range []struct {
			name  string
			parts []string
			gloss string
			id    string
		}{
			{"nothing", nil, "", ""},
			{"one item", []string{"a"}, "a", "a"},
			{"two take the conjunction alone", []string{"a", "b"},
				"a" + and + "b", "a" + ids + "b"},
			{"three take one conjunction and one comma", []string{"a", "b", "c"},
				"a" + comma + "b" + and + "c", "a" + comma + "b" + ids + "c"},
			{"four take one conjunction and two commas", []string{"a", "b", "c", "d"},
				"a" + comma + "b" + comma + "c" + and + "d",
				"a" + comma + "b" + comma + "c" + ids + "d"},
		} {
			t.Run(lang.String()+"/"+c.name, func(t *testing.T) {
				if got := lang.JoinIDs(c.parts); got != c.id {
					t.Errorf("JoinIDs(%q) = %q, want %q", c.parts, got, c.id)
				}
			})
		}
		// The gloss joiner is unexported, so it is reached through the sentence
		// that uses it rather than called directly — a strip names its categories
		// through `join` and takes a count, which is the shortest route to it.
		if strings.Count(lang.Text(i18n.BlurbStrips), "%") < 2 {
			t.Fatalf("%v: BlurbStrips no longer takes a count and a list, so this "+
				"test is no longer reaching the gloss joiner", lang)
		}
	}
}

// TestNoDescriptionReadsAConjunctionTwice is the same rule measured end to end,
// over every skill the library holds rather than over a slice this test wrote.
//
// ⚠️ It fails when **nothing** in the books produces a list of three, because then
// it is asserting the absence of a shape that could not have appeared and measures
// nothing. Today the only such list is the fixture's `purify`, which strips three
// categories — no shipped skill has one — so this guard is one authored cleanse
// away from being the only reader of that branch, and it should say so rather than
// pass in silence.
func TestNoDescriptionReadsAConjunctionTwice(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	for _, lang := range i18n.Langs() {
		and, comma := lang.Text(i18n.BlurbAnd), lang.Text(i18n.ListComma)
		lists := 0
		for _, declared := range everySkillDeclared(t, shapes) {
			for _, line := range strings.Split(lang.Describe(declared, shapes), "\n") {
				if strings.Count(line, and) > 1 {
					t.Errorf("%v: %q reads the conjunction twice in one line:\n  %s",
						lang, declared.ID, line)
				}
				if strings.Contains(line, comma) && strings.Contains(line, and) {
					lists++
				}
			}
		}
		if lists == 0 {
			t.Errorf("%v: nothing in the books reads out a list of three, so a "+
				"conjunction between every pair would have passed here", lang)
		}
	}
}
