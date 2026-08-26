package i18n_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
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

	var b strings.Builder
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		fmt.Fprintf(&b, "== %s ==\n\n", lang)
		for _, declared := range skills.Skills() {
			fmt.Fprintf(&b, "%s\n%s\n\n", declared.ID, lang.Describe(declared, shapes))
		}
		for _, held := range passives.All() {
			fmt.Fprintf(&b, "[trait] %s\n%s\n\n", held.ID, lang.DescribePassive(held))
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
			if held.Replies.Answers() {
				want++
			}
			if held.Drains > 0 {
				want++
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
