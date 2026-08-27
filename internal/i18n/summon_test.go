package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// summoner parses one summoning skill against the shipped shapes and statuses.
func summoner(t *testing.T, summons string) skill.Skill {
	t.Helper()
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	book, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"call","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":3,"target":"self",
	   "summons":`+summons+`}
	]}`), skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	call, err := book.Lookup("call")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	return call
}

// TestASummoningSkillSaysWhatItCallsUp is the description half of the feature.
//
// A skill whose whole effect is putting somebody on the board would otherwise
// describe itself with its costs and nothing else — a line saying how far it
// reaches and how long until it comes back, about a skill that does not reach
// anywhere. That is worse than no description: it reads as a skill that does
// nothing, which is exactly what the parser refuses to let one be.
func TestASummoningSkillSaysWhatItCallsUp(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	// The authored name is Vietnamese, so only Vietnamese prints it; English says
	// the word, which is the division Gloss makes everywhere else in these
	// descriptions.
	want := map[i18n.Lang]string{i18n.Vi: "phân thân", i18n.En: "copies"}
	for _, lang := range i18n.Langs() {
		described := lang.Describe(
			summoner(t, `{"count":2,"name":"phân thân","share":500,"skills":["jab"]}`), patterns)
		if !strings.Contains(described, want[lang]) {
			t.Errorf("%s: the description never names what arrives:\n%s", lang, described)
		}
		if !strings.Contains(described, "2") {
			t.Errorf("%s: the description never says how many arrive:\n%s", lang, described)
		}
	}
}

// TestASummonThatStaysAndOneThatDoesNotReadDifferently is the second wording,
// and the reason there are two rather than a clause tacked on.
//
// How long a copy stays is the difference between a skill worth a turn and one
// worth a turn every four, so a reader who cannot tell the two apart cannot
// decide anything.
func TestASummonThatStaysAndOneThatDoesNotReadDifferently(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		stays := lang.Describe(summoner(t, `{"name":"cóc","stats":{"hp":900,"attack":300,`+
			`"defense":200,"speed":40,"accuracy":0,"dodge":0},"skills":["jab"]}`), patterns)
		brief := lang.Describe(summoner(t, `{"name":"cóc","lasts":3,"stats":{"hp":900,`+
			`"attack":300,"defense":200,"speed":40,"accuracy":0,"dodge":0},"skills":["jab"]}`), patterns)
		if stays == brief {
			t.Errorf("%s: a summon that stays and one that lasts three turns read the same:\n%s",
				lang, stays)
		}
		if !strings.Contains(brief, "3") {
			t.Errorf("%s: the brief summon never says three:\n%s", lang, brief)
		}
	}
}

// TestASummonWithNoNameIsStillNamed is the fallback, and it is a word rather
// than an id on purpose: a summon with no name of its own is a copy of whoever
// cast it, and this layer holds the skill rather than the caster — so "copy" is
// the truest thing it can say.
func TestASummonWithNoNameIsStillNamed(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	described := i18n.Vi.Describe(summoner(t, `{"share":500,"skills":["jab"]}`), patterns)
	if !strings.Contains(described, "phân thân") {
		t.Errorf("an unnamed summon is described as %q", described)
	}
}
