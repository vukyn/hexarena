package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestASkillDescriptionNamesItsCriticalChanceAndNotTheMultiplier is the clause,
// in both languages.
//
// It names the chance and only the chance. What a critical strike is worth is one
// game-wide constant on combat.Rules, so putting it in the sentence would restate
// on every critting skill a number a reader can look up once — and it would need
// this package to import the rules book to say it, which is the import the layer
// rule is there to prevent.
func TestASkillDescriptionNamesItsCriticalChanceAndNotTheMultiplier(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	// A shipped damaging skill, so everything but the one field is the real
	// thing and the clause is the only difference between the two readings.
	base, err := skills.Lookup("vine_whip")
	if err != nil {
		t.Fatalf("look up a shipped skill: %v", err)
	}
	if base.Crit != 0 {
		t.Fatalf("%s already crits, so the pair below differs by more than the clause", base.ID)
	}
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		// The clause without the figure in it, so the test does not carry a
		// second copy of a wording it is checking.
		clause, _, _ := strings.Cut(strings.TrimPrefix(lang.Say(i18n.BlurbCritical, "20%"), ", "), "20%")
		clause = strings.TrimSpace(clause)
		if clause == "" {
			// The Vietnamese wording opens with the share, so take the tail
			// instead; either way this is the fixed half of the sentence.
			_, clause, _ = strings.Cut(lang.Say(i18n.BlurbCritical, "20%"), "20%")
			clause = strings.TrimSpace(clause)
		}
		if clause == "" {
			t.Fatalf("%s: the critical clause is nothing but its figure", lang)
		}

		silent := lang.Describe(base, shapes)
		if strings.Contains(silent, clause) {
			t.Errorf("%s: a skill that cannot crit is described as %q: %s", lang, clause, silent)
		}

		critting := base
		critting.Crit = 200
		described := lang.Describe(critting, shapes)
		if !strings.Contains(described, clause) {
			t.Errorf("%s: a skill that crits is not described as doing so: %s", lang, described)
		}
		if !strings.Contains(described, "20%") {
			t.Errorf("%s: the clause does not carry the rounded share: %s", lang, described)
		}
		// Never the multiplier. 1250, 125% and 1.25 are all the same mistake:
		// a figure identical on every skill in the game, restated per skill.
		for _, forbidden := range []string{"1250", "125%", "1.25", "1,25"} {
			if strings.Contains(described, forbidden) {
				t.Errorf("%s: the description names the critical multiplier (%q), which is a"+
					" constant every skill shares: %s", lang, forbidden, described)
			}
		}
		// The clause sits after the piercing one, so a skill that does both reads
		// in the order the fields are written in.
		both := base
		both.Pierce = 400
		both.Crit = 200
		line := lang.Describe(both, shapes)
		pierceClause, _, _ := strings.Cut(strings.TrimPrefix(lang.Say(i18n.BlurbPierces, "40%"), ", "), " ")
		if at, to := strings.Index(line, pierceClause), strings.Index(line, clause); at < 0 || to < 0 || at > to {
			t.Errorf("%s: the piercing clause is at %d and the critical one at %d, want them in that order: %s",
				lang, at, to, line)
		}
	}
}

// TestTheCriticalWordingsAreWordedInBothLanguages is a spot check of the three
// new keys against the property the whole table already has, so a failure here
// says which key rather than only that one is missing.
func TestTheCriticalWordingsAreWordedInBothLanguages(t *testing.T) {
	for _, key := range []i18n.Key{i18n.BlurbCritical, i18n.SkillFieldCrit, i18n.SkillHelpCrit} {
		for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
			if strings.TrimSpace(lang.Text(key)) == "" {
				t.Errorf("%s has no wording for %v", lang, key)
			}
		}
		if i18n.Vi.Text(key) == i18n.En.Text(key) {
			t.Errorf("%v is worded identically in both languages, which is a table entry that was copied", key)
		}
	}
}
