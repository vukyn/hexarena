package i18n

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/seed"
)

// A gating condition is the one condition here that is not an amplifier, and
// until it existed every sentence in describe.go could open the same way and be
// true. It is read *before* the skill is offered — skill.Condition.Gates says so
// and battle.options acts on it — so "while this unit is carrying five stacks"
// describes a moment that never happens: a caster short of the five is not
// casting at all.
//
// Two describers say it and both said it wrong, in different ways. The sentence
// called a gate an amplifier; the compact line, having no figure to quote, fell
// through to the wording that ends in "spreads" — the one thing a caster's own
// condition is forbidden to do.

// beforeTheBlank is the fixed part of a wording, up to its first blank. The clauses
// after it come from the data, so the opening is the whole of what a sentence can
// be asserted to start with.
func beforeTheBlank(text string) string {
	if at := strings.Index(text, "%"); at >= 0 {
		return text[:at]
	}
	return text
}

// gatedSkills is every shipped skill whose cast is gated, and it fails rather
// than skipping when there are none.
//
// ⚠️ A test over a shape nothing ships measures nothing, and it does it silently
// for as long as nobody looks. The gate arrived with a skill; if that skill
// leaves the data, this says so instead of going green.
func gatedSkills(t *testing.T) []skill.Skill {
	t.Helper()
	book, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	var out []skill.Skill
	for _, declared := range book.Skills() {
		if declared.SelfRequires.GatesCast() {
			out = append(out, declared)
		}
	}
	if len(out) == 0 {
		t.Fatal("no shipped skill gates its cast, so nothing below measures the wording " +
			"a gate takes")
	}
	return out
}

// TestAGatedSkillIsNotDescribedAsAnAmplifier holds the sentence.
//
// The assertion is on the **opening** and by prefix rather than by search,
// because the opening is the whole difference: the clause after it — "is carrying
// at least 5 stacks of X" — is the same string either way, which is exactly why
// the amplifier's version of this sentence read as plausibly as it did.
func TestAGatedSkillIsNotDescribedAsAnAmplifier(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	for _, lang := range Langs() {
		gate := beforeTheBlank(lang.Text(BlurbSelfGatedShape))
		amplifier := beforeTheBlank(lang.Text(BlurbSelfAmplifiedShape))
		if gate == "" || gate == amplifier {
			t.Fatalf("%s opens a gate with %q and an amplifier with %q, so no sentence "+
				"below can be told from the other", lang, gate, amplifier)
		}
		for _, declared := range gatedSkills(t) {
			var gated, amplified bool
			for _, line := range strings.Split(lang.Describe(declared, shapes), "\n") {
				gated = gated || strings.HasPrefix(line, gate)
				amplified = amplified || strings.HasPrefix(line, amplifier)
			}
			if !gated {
				t.Errorf("%s describes the gated %s without a line opening %q:\n%s",
					lang, declared.ID, gate, lang.Describe(declared, shapes))
			}
			if amplified {
				t.Errorf("%s describes the gated %s with a line opening %q, which says the "+
					"fuel makes the skill stronger rather than possible:\n%s",
					lang, declared.ID, amplifier, lang.Describe(declared, shapes))
			}
		}
	}
}

// TestAGatedSkillsCompactLineDoesNotSayItSpreads holds the other describer.
//
// The one-line summary is where the older wording was not merely imprecise: with
// no bonus figure to quote it took the shape wording, and the shape wording for a
// caster-side condition ends in the word for a chain. A gate that spends the
// caster's own reserve cannot travel anywhere.
func TestAGatedSkillsCompactLineDoesNotSayItSpreads(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	for _, lang := range Langs() {
		gate := beforeTheBlank(lang.Text(SummarySelfGatedShape))
		spreads := lang.Text(SummarySelfAmplifiedShape)
		tail := spreads[strings.LastIndex(spreads, "%s")+len("%s"):]
		if strings.TrimSpace(tail) == "" {
			t.Fatalf("%s ends its shape wording %q with nothing, so a summary carrying it "+
				"cannot be recognised", lang, spreads)
		}
		for _, declared := range gatedSkills(t) {
			line := lang.SummariseSkill(declared, shapes)
			if !strings.Contains(line, gate) {
				t.Errorf("%s summarises the gated %s as %q, which does not say the fuel is "+
					"what lets it be cast (want a clause opening %q)",
					lang, declared.ID, line, gate)
			}
			if strings.Contains(line, tail) {
				t.Errorf("%s summarises the gated %s as %q, which ends a clause in %q — the "+
					"caster's own condition may not chain", lang, declared.ID, line, tail)
			}
		}
	}
}
