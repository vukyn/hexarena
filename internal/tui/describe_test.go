package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

// TestDescribeEveryShippedSkillGolden is the design record for the sentences,
// and the reason the descriptions are derived rather than authored: a number
// moving in skills.json moves a line here, and the diff is what says how the
// change reads to a player.
//
// It covers every shipped skill and trait rather than a chosen few. A sampled
// golden would leave the next skill's phrasing unmeasured, and phrasing is
// exactly what this file exists to hold still.
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
	b.WriteString("what a skill does, as the battle prompt says it\n\n")
	for _, declared := range skills.Skills() {
		fmt.Fprintf(&b, "%s\n%s\n\n", declared.ID, tui.Describe(declared, shapes))
	}
	b.WriteString("traits\n\n")
	for _, held := range passives.All() {
		fmt.Fprintf(&b, "%s\n%s\n\n", held.ID, tui.DescribePassive(held))
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
		t.Fatalf("read golden (run: go test ./internal/tui -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the descriptions differ from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// TestEverySkillDescriptionSaysWhatItDoes is the property a golden cannot hold:
// a golden freezes whatever was generated, including a skill the generator has
// nothing to say about. This asserts that every shipped skill produces a
// description with an aim, a cost line, and no leftover placeholder.
func TestEverySkillDescriptionSaysWhatItDoes(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	for _, declared := range skills.Skills() {
		description := tui.Describe(declared, shapes)
		if strings.TrimSpace(description) == "" {
			t.Errorf("%q has no description at all", declared.ID)
			continue
		}
		lines := strings.Split(description, "\n")
		if len(lines) < 2 {
			t.Errorf("%q is described in one line, with no cost line under it: %q",
				declared.ID, description)
		}
		// The last line is always the costs, and it always names the aim: a
		// range for anything that reaches, "bản thân" for anything that does not.
		// Lowered before matching: the line is capitalised as a sentence, and a
		// test that matched the capital would pass for the wrong reason the day
		// the line stopped opening with this clause.
		costs := strings.ToLower(lines[len(lines)-1])
		if declared.Target == skill.Self {
			if !strings.Contains(costs, "bản thân") {
				t.Errorf("%q targets itself and its cost line does not say so: %q", declared.ID, costs)
			}
		} else if !strings.Contains(costs, "tầm") {
			t.Errorf("%q reaches out and its cost line states no range: %q", declared.ID, costs)
		}
		// An id leaking into the prose means a gloss is missing, which reads as
		// English dropped into the middle of a Vietnamese sentence.
		for _, application := range declared.Applies {
			if strings.Contains(description, application.Status) {
				t.Errorf("%q describes the status %q by its id, so it has no Vietnamese name",
					declared.ID, application.Status)
			}
		}
	}
}
