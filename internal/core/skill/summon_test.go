package skill_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// summoning is a book of one summoning skill plus the skill it summons, built
// from a fragment so a test can say only what it is about.
func summoning(t *testing.T, summons string) (*skill.Book, error) {
	t.Helper()
	return skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"call","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "summons":`+summons+`}
	]}`), deps(t))
}

// TestASummonMustSayWhatTheUnitIsMadeOf is the first of the two ways the three
// stat spellings can be got wrong, and it is the silent one: a summon with no
// stat line at all would put a unit of nothing on the board.
func TestASummonMustSayWhatTheUnitIsMadeOf(t *testing.T) {
	_, err := summoning(t, `{"count":1,"skills":["jab"]}`)
	if err == nil {
		t.Fatal("a summon with no stat line was accepted")
	}
	if !strings.Contains(err.Error(), "share") {
		t.Errorf("the refusal is %q and does not name the field to fill in", err)
	}
}

// TestASummonMayNotSayItTwice is the other way: two spellings of one thing, and
// no rule anywhere about which of them wins.
func TestASummonMayNotSayItTwice(t *testing.T) {
	for _, pair := range []string{
		`{"share":500,"share_of_base":500,"skills":["jab"]}`,
		`{"share":500,"stats":{"hp":10,"attack":10,"defense":10,"speed":10,"accuracy":0,"dodge":0},"skills":["jab"]}`,
	} {
		if _, err := summoning(t, pair); err == nil {
			t.Errorf("%s was accepted with two stat lines", pair)
		}
	}
}

// TestASummonedUnitMustKnowSomething keeps a skill from putting a unit down that
// can only stand there. The board has room for five and a slot spent on
// scenery is a slot a real unit cannot have.
func TestASummonedUnitMustKnowSomething(t *testing.T) {
	if _, err := summoning(t, `{"share":500,"skills":[]}`); err == nil {
		t.Error("a summon of a unit knowing no skill was accepted")
	}
}

// TestASummonMayNotSummon is the rule that cannot be checked while one skill is
// being read, and the reason ParseBook has a second pass.
//
// Without it a single cast is unbounded. The board would stop it in practice,
// and "it runs out of room" is not a rule anybody can read off the file.
func TestASummonMayNotSummon(t *testing.T) {
	_, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"inner","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "summons":{"share":500,"skills":["jab"]}},
	  {"id":"outer","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "summons":{"share":500,"skills":["inner"]}}
	]}`), deps(t))
	if err == nil {
		t.Fatal("a summon of a summoner was accepted")
	}
	if !strings.Contains(err.Error(), "outer") || !strings.Contains(err.Error(), "inner") {
		t.Errorf("the refusal is %q and does not name both skills", err)
	}
}

// TestASummonIsRefusedForwardsAsWellAsBackwards is the half a single pass would
// miss: the skill being summoned is declared *after* the one summoning it, so a
// check made while reading the first entry has nothing to look up yet.
func TestASummonIsRefusedForwardsAsWellAsBackwards(t *testing.T) {
	_, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":60,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"outer","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "summons":{"share":500,"skills":["later"]}},
	  {"id":"later","element":"neutral","range":0,"pattern":"single",
	   "power":0,"strikes":0,"accuracy":1000,"cooldown":0,"target":"self",
	   "summons":{"share":500,"skills":["jab"]}}
	]}`), deps(t))
	if err == nil {
		t.Fatal("a summon of a summoner declared below it was accepted")
	}
}

// TestASummonNamesASkillTheBookHas catches the typo, which is the mistake an
// author actually makes.
func TestASummonNamesASkillTheBookHas(t *testing.T) {
	if _, err := summoning(t, `{"share":500,"skills":["jib"]}`); err == nil {
		t.Error("a summon naming a skill the book has no entry for was accepted")
	}
}

// TestASummonsCountIsBounded stops an author asking for more units than a side
// can ever hold, which the board would answer with silence.
func TestASummonsCountIsBounded(t *testing.T) {
	if _, err := summoning(t, `{"count":99,"share":500,"skills":["jab"]}`); err == nil {
		t.Error("a summon of ninety-nine units was accepted")
	}
	book, err := summoning(t, `{"share":500,"skills":["jab"]}`)
	if err != nil {
		t.Fatalf("a summon with no count: %v", err)
	}
	call, err := book.Lookup("call")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if call.Summons.Count != 1 {
		t.Errorf("a summon with no count declared %d, want the one it reads as",
			call.Summons.Count)
	}
}

// TestASummoningSkillIsNotAWastedTurn is the guard that already refused every
// skill with no power and nothing else, meeting the first effect it did not know
// about. It refused the fixture skills the moment they were written.
func TestASummoningSkillIsNotAWastedTurn(t *testing.T) {
	if _, err := summoning(t, `{"share":500,"skills":["jab"]}`); err != nil {
		t.Errorf("a skill whose whole effect is a summon was called a wasted turn: %v", err)
	}
}

// TestASummonRoundTripsThroughTheBook is what the authoring tool needs: the
// bytes it is about to write are bytes that load, and every field survives.
func TestASummonRoundTripsThroughTheBook(t *testing.T) {
	book, err := summoning(t,
		`{"count":2,"name":"copy","share_of_base":400,"element":"water",`+
			`"skills":["jab"],"lasts":3,"bound":true}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := skill.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("re-parse what marshal wrote: %v\n%s", err, raw)
	}
	before, _ := book.Lookup("call")
	after, _ := again.Lookup("call")
	if !reflect.DeepEqual(before.Summons, after.Summons) {
		t.Errorf("the summon came back as %+v, want %+v", *after.Summons, *before.Summons)
	}
}
