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

// TestASummonSaysHowStrongItArrives is the figure the sentence used to leave out.
//
// A copy is the only thing a skill puts on the board whose strength is a number
// the author chose, and it was the one number the description did not carry: two
// copies at a tenth of their caster and two at four fifths read identically, so
// a reader deciding whether the skill was worth a turn was deciding without it.
// The share was left out on the reasoning that a listing beside the sentence
// carries it, and no listing does — neither hexforge nor its full-screen twin
// mentions a summon at all.
func TestASummonSaysHowStrongItArrives(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		described := lang.Describe(
			summoner(t, `{"count":2,"name":"phân thân","share":400,"skills":["jab"]}`), patterns)
		if !strings.Contains(described, "40%") {
			t.Errorf("%s: a copy at 400 parts per thousand never says 40%%:\n%s", lang, described)
		}
	}
}

// TestTwoSummonsAtDifferentSharesReadDifferently is the same rule stated as the
// thing that would break: a figure printed but never varying is a figure copied
// into the wording, and this is a description whose whole claim is that it is
// derived.
func TestTwoSummonsAtDifferentSharesReadDifferently(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		weak := lang.Describe(
			summoner(t, `{"count":2,"name":"phân thân","share":200,"skills":["jab"]}`), patterns)
		strong := lang.Describe(
			summoner(t, `{"count":2,"name":"phân thân","share":800,"skills":["jab"]}`), patterns)
		if weak == strong {
			t.Errorf("%s: a copy at a fifth and one at four fifths read the same:\n%s",
				lang, weak)
		}
	}
}

// TestAShareOfBaseReadsDifferentlyFromAShare is the distinction the two spellings
// exist for, and it is one a player acts on: a share of the caster as it stands
// pays for buffing before the cast, and a share of its base ignores every buff on
// the board. One wording for both would describe whichever it was not written for
// as the other.
func TestAShareOfBaseReadsDifferentlyFromAShare(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		now := lang.Describe(
			summoner(t, `{"name":"phân thân","share":400,"skills":["jab"]}`), patterns)
		base := lang.Describe(
			summoner(t, `{"name":"phân thân","share_of_base":400,"skills":["jab"]}`), patterns)
		if now == base {
			t.Errorf("%s: a share of the caster and a share of its base read the same:\n%s",
				lang, now)
		}
	}
}

// TestASummonOnAFixedLineClaimsNoShare is the other half: six figures nobody can
// compare without the caster in front of them stay out, and a summon that has no
// share must not be handed one. A percent sign in a toad's sentence would be a
// number invented by the renderer.
func TestASummonOnAFixedLineClaimsNoShare(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		described := lang.Describe(summoner(t, `{"name":"cóc","stats":{"hp":900,"attack":300,`+
			`"defense":200,"speed":40,"accuracy":0,"dodge":0},"skills":["jab"]}`), patterns)
		if strings.Contains(described, "%") {
			t.Errorf("%s: a summon on a fixed stat line is described with a share:\n%s",
				lang, described)
		}
	}
}

// TestACreatureIsNotDescribedAsACopy is the fallback telling the truth.
//
// English does not print an authored Vietnamese name, so it falls back to a word
// — and the word was "copy" for everything, which made the shipped toad read as
// "calls up a copy". A toad is not a copy of the ninja who called it, and the
// engine already draws that line: a copy is written as a share of its caster and
// a creature as a stat line of its own.
func TestACreatureIsNotDescribedAsACopy(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	clone := i18n.En.Describe(summoner(t, `{"share":400,"skills":["jab"]}`), patterns)
	beast := i18n.En.Describe(summoner(t, `{"stats":{"hp":900,"attack":300,`+
		`"defense":200,"speed":40,"accuracy":0,"dodge":0},"skills":["jab"]}`), patterns)
	if !strings.Contains(clone, "copy") {
		t.Errorf("a summon made of a share of its caster is not called a copy: %q", clone)
	}
	if strings.Contains(beast, "copy") {
		t.Errorf("a summon on a stat line of its own is called a copy: %q", beast)
	}
}

// holds reports whether described carries every literal of a wording, which is
// how a test asks "was this wording the one used" without knowing the language.
//
// A format string is its blanks and the words between them; the blanks are filled
// with things this test does not want to predict, and the words between them are
// the wording itself. Splitting on the verb and asking for the pieces is the only
// check here that stays true when a translation is reworded, which is a thing
// these tables are for.
func holds(described, wording string) bool {
	for _, literal := range strings.Split(wording, "%s") {
		if literal == "" {
			continue
		}
		if !strings.Contains(described, literal) {
			return false
		}
	}
	return true
}

// TestOneCopyAndSeveralSayTheShareDifferently is the reason there are two share
// wordings rather than one.
//
// Several copies each carry the share and one copy simply has it, and a language
// decides for itself whether that needs a word — Vietnamese wants "mỗi bên" and
// English wants "each". Handing the singular wording to a pair reads as a pair
// that carries 40% between them, which is half of what arrives.
//
// Comparing the two descriptions would not catch it: they differ in their count
// whatever wording the share takes, so the difference has to be asked about the
// share clause itself.
func TestOneCopyAndSeveralSayTheShareDifferently(t *testing.T) {
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("shapes: %v", err)
	}
	for _, lang := range i18n.Langs() {
		alone := lang.Describe(
			summoner(t, `{"count":1,"name":"phân thân","share":400,"skills":["jab"]}`), patterns)
		pair := lang.Describe(
			summoner(t, `{"count":2,"name":"phân thân","share":400,"skills":["jab"]}`), patterns)
		if !holds(pair, lang.Text(i18n.BlurbSummonedShareEach)) {
			t.Errorf("%s: two copies are described with the wording written for one:\n%s",
				lang, pair)
		}
		if holds(alone, lang.Text(i18n.BlurbSummonedShareEach)) {
			t.Errorf("%s: one copy is described with the wording written for several:\n%s",
				lang, alone)
		}
	}
}
