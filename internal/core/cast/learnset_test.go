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
	if got := character.SkillsAt(19); strings.Join(got, " ") != "strike" {
		t.Errorf("at level 19 it knows %v, want only what it has learned", got)
	}
	if got := character.SkillsAt(20); strings.Join(got, " ") != "strike riptide" {
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
