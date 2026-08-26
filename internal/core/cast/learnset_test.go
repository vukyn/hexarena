package cast_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// TestASkillIsLearnedAtALevelLikeATraitIs is the whole of "skills and traits are
// one mechanism": the same type, the same rules, the same "what is available at
// level N" function.
func TestASkillIsLearnedAtALevelLikeATraitIs(t *testing.T) {
	entry := baseCharacter()
	entry["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide", "at_level": 20},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, _ := book.Get("a-series.warden")
	if len(character.Skills) != 2 {
		t.Fatalf("the learnset holds %d entries, want 2", len(character.Skills))
	}
	// An unstated level normalises to one, so every caller can ask Unlocked
	// without first asking which of two spellings it is holding.
	if character.Skills[0].AtLevel != 1 {
		t.Errorf("an unstated level came out as %d, want 1", character.Skills[0].AtLevel)
	}
	if character.Skills[1].AtLevel != 20 {
		t.Errorf("the gate came out as %d, want 20", character.Skills[1].AtLevel)
	}
	if got := character.SkillsAt(19, progression.Furthest); strings.Join(got, " ") != "strike" {
		t.Errorf("at level 19 it knows %v, want only what it has learned", got)
	}
	if got := character.SkillsAt(20, progression.Furthest); strings.Join(got, " ") != "strike riptide" {
		t.Errorf("at level 20 it knows %v, want both", got)
	}
	// And the whole list regardless of level, which is what every check about a
	// restriction wants: a restriction is a property of the skill rather than of
	// when it is learned.
	if got := cast.LearnedIDs(character.Skills); strings.Join(got, " ") != "strike riptide" {
		t.Errorf("its learnset reads %v, want everything it ever learns", got)
	}
}

// TestACharacterLearnsSomethingAtLevelOne is a refusal about the character
// rather than about any placement of it.
//
// A character whose first skill arrives at level eight cannot be fielded at
// level one at all — it would have nothing to do on its turn. Refusing it where
// it is authored is what stops the author finding out by fielding it, since
// every other level would have been legal.
func TestACharacterLearnsSomethingAtLevelOne(t *testing.T) {
	entry := baseCharacter()
	entry["skills"] = []map[string]any{
		{"id": "strike", "at_level": 8},
		{"id": "riptide", "at_level": 20},
	}
	_, err := parse(t, entry)
	if err == nil {
		t.Fatal("a character that learns nothing at level 1 was accepted")
	}
	if !strings.Contains(err.Error(), "level 1") {
		t.Errorf("the refusal reads %q, want it to name the level nothing is known at", err)
	}
	// One entry at level one is enough, which is what makes the refusal about
	// having *something* rather than about the gates on the rest.
	entry["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide", "at_level": 20},
	}
	if _, err := parse(t, entry); err != nil {
		t.Errorf("a character that learns one skill at level 1 was refused: %v", err)
	}
}

// TestALearnsetRejections are the ways an entry can be wrong, and they are the
// ones a trait's entry is already refused for.
func TestALearnsetRejections(t *testing.T) {
	cases := []struct {
		name   string
		skills []map[string]any
		wantIn string
	}{
		{
			"a level past the cap",
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "at_level": progression.LevelCap + 1}},
			"outside 1..",
		},
		{
			"a negative level",
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "at_level": -3}},
			"outside 1..",
		},
		{
			"the same skill twice",
			[]map[string]any{{"id": "strike"}, {"id": "strike", "at_level": 20}},
			"twice",
		},
		{
			"an unknown skill",
			[]map[string]any{{"id": "strike"}, {"id": "meteor"}},
			"unknown skill",
		},
		{
			"an empty learnset",
			[]map[string]any{},
			"knows no skills",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			entry := baseCharacter()
			entry["skills"] = test.skills
			_, err := parse(t, entry)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantIn)
			}
		})
	}
}

// TestALearnsetSurvivesTheFile is the round trip an authoring tool depends on:
// hexforge writes a book by marshalling one it parsed, so a level the writer
// forgets is a level an author silently loses.
func TestALearnsetSurvivesTheFile(t *testing.T) {
	entry := baseCharacter()
	entry["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide", "at_level": 20},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// A gate of one is not written, because a parse normalises an unstated level
	// to one: writing it would be noise on the line that matters least, and
	// re-reading it gives the same book either way.
	if strings.Contains(string(raw), `"at_level": 1`) {
		t.Errorf("the written learnset states a gate of one:\n%s", raw)
	}
	again, err := cast.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("re-parse what was written: %v\n%s", err, raw)
	}
	back, _ := again.Get("a-series.warden")
	if len(back.Skills) != 2 || back.Skills[1].AtLevel != 20 {
		t.Errorf("the learnset came back as %+v", back.Skills)
	}
}

// TestAnEntryMayBeKeptForSomeForms is the second gate, and the one that makes
// giving up an evolution buy something.
//
// A threshold could only ever say "from this form onwards", so everything an
// early form knew a grown one knew too. A list can say "the bulb forms only",
// which is a move the grown form never gets — and that is the whole of why
// evolving is a decision rather than a formality.
func TestAnEntryMayBeKeptForSomeForms(t *testing.T) {
	entry := baseCharacter()
	entry["stages"] = []map[string]any{
		{"name": "Warden", "min_level": 1, "stats": table()},
		{"name": "Keeper", "min_level": 20, "stats": table()},
	}
	entry["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide", "stages": []string{"Warden"}},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	character, _ := book.Get("a-series.warden")
	if got := character.SkillsAt(30, "Warden"); strings.Join(got, " ") != "strike riptide" {
		t.Errorf("fielded as Warden it knows %v, want the skill kept for it", got)
	}
	if got := character.SkillsAt(30, "Keeper"); strings.Join(got, " ") != "strike" {
		t.Errorf("fielded as Keeper it knows %v, want the kept skill left behind", got)
	}
	// Both gates together, because a caller that asked only one would be wrong
	// in whichever direction it forgot.
	kept := character.Skills[1]
	if kept.Available(30, "Keeper") {
		t.Error("the level gate passed and the form gate did not, and it came out available")
	}
	if !kept.Held("Warden") || kept.Held("Keeper") {
		t.Errorf("the allowlist reads %v", kept.Stages)
	}
	// An entry with no list is held by every form, which is the ordinary case.
	if !character.Skills[0].Held("Keeper") {
		t.Error("an entry naming no forms is not held by all of them")
	}
}

// TestAStageAllowlistRejections are the ways one can be wrong, and each is a
// silence somebody would otherwise have to notice on their own.
func TestAStageAllowlistRejections(t *testing.T) {
	twoStages := []map[string]any{
		{"name": "Warden", "min_level": 1, "stats": table()},
		{"name": "Keeper", "min_level": 20, "stats": table()},
	}
	cases := []struct {
		name   string
		stages []map[string]any
		skills []map[string]any
		wantIn string
	}{
		{
			"a form the line does not have",
			twoStages,
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "stages": []string{"Nonesuch"}}},
			"its forms are",
		},
		{
			"one form named twice",
			twoStages,
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "stages": []string{"Warden", "Warden"}}},
			"twice",
		},
		{
			"every form, which is what naming none means",
			twoStages,
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "stages": []string{"Warden", "Keeper"}}},
			"every one of its forms",
		},
		{
			"a form on a character with one",
			[]map[string]any{{"name": "Warden", "min_level": 1, "stats": table()}},
			[]map[string]any{{"id": "strike"}, {"id": "riptide", "stages": []string{"Warden"}}},
			"every one of its forms",
		},
		{
			"nothing the first form can use at level one",
			twoStages,
			[]map[string]any{{"id": "strike", "stages": []string{"Keeper"}}, {"id": "riptide", "at_level": 20}},
			"nothing it could use at level 1",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			entry := baseCharacter()
			entry["stages"] = test.stages
			entry["skills"] = test.skills
			_, err := parse(t, entry)
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantIn)
			}
		})
	}
}

// TestAStageAllowlistSurvivesTheFile is the round trip, on the field most likely
// to be dropped: it is the newest, and it is the one an author cannot see the
// loss of until a unit is fielded.
func TestAStageAllowlistSurvivesTheFile(t *testing.T) {
	entry := baseCharacter()
	entry["stages"] = []map[string]any{
		{"name": "Warden", "min_level": 1, "stats": table()},
		{"name": "Keeper", "min_level": 20, "stats": table()},
	}
	entry["skills"] = []map[string]any{
		{"id": "strike"},
		{"id": "riptide", "at_level": 4, "stages": []string{"Warden"}},
	}
	book, err := parse(t, entry)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := cast.ParseBook(raw, deps(t))
	if err != nil {
		t.Fatalf("re-parse what was written: %v\n%s", err, raw)
	}
	back, _ := again.Get("a-series.warden")
	if len(back.Skills) != 2 {
		t.Fatalf("the learnset came back with %d entries", len(back.Skills))
	}
	if strings.Join(back.Skills[1].Stages, " ") != "Warden" {
		t.Errorf("the allowlist came back as %v", back.Skills[1].Stages)
	}
	if back.Skills[1].AtLevel != 4 {
		t.Errorf("the level came back as %d", back.Skills[1].AtLevel)
	}
	// An entry with no list writes none, so a character that keeps nothing for a
	// form round-trips to the bytes it was authored as.
	if strings.Contains(string(raw), `"stages": []`) {
		t.Errorf("an empty allowlist was written out:\n%s", raw)
	}
}
