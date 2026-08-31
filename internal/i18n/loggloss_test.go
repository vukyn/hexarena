package i18n

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/seed"
)

// TestNoLogGlossCollidesAcrossKinds is the loud half of LogGlosses.
//
// ⚠️ **TestNoIDIsGlossedTwice does not cover this and could not.** It holds the
// compiled tables in gloss.go disjoint from each other; this map is built over
// three *data* books that have no shared namespace, so an id can be a skill and a
// status at once with every table above still disjoint. That is the same shape of
// gap the category gloss had — it was checked within a table and `taunt` collided
// across one — and `taunt` is a shipped skill id with `taunting` a shipped status
// id, so the near miss is real.
//
// LogGlosses leaves a collision out rather than picking, because it is asked while
// a screen is drawn and a data id with no name is this package's declared normal.
// A collision is therefore silent in the client and has to be caught here.
func TestNoLogGlossCollidesAcrossKinds(t *testing.T) {
	skillBook, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}
	statusBook, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	passiveBook, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load passives: %v", err)
	}
	collisions := LogGlossCollisions(
		skillBook.Skills(), statusBook.Kinds(), passiveBook.All())
	if len(collisions) != 0 {
		t.Errorf("these ids are declared by more than one of skills.json, statuses.json "+
			"and passives.json, so the battle log cannot say which one it means and drops "+
			"the name entirely: %s", strings.Join(collisions, ", "))
	}
}

// TestACollidingIDIsLeftOutRatherThanPickedBetween is the behaviour the shipped
// books cannot show, because they collide on nothing.
//
// It matters which way this fails: a wrong name is worse than a missing one, and
// nothing on screen distinguishes them. So the map is asserted to hold neither of
// the two candidates rather than to hold "the right one" — there is no right one.
func TestACollidingIDIsLeftOutRatherThanPickedBetween(t *testing.T) {
	const shared = "taunt"
	statusBook, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	skillBook, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}
	declared, err := skillBook.Lookup(shared)
	if err != nil {
		t.Fatalf("the shipped book has no %s, so this test measures nothing: %v", shared, err)
	}
	if Vi.SkillName(declared) == "" {
		t.Fatalf("%s carries no name, so a collision over it would be invisible", shared)
	}
	// A status with the skill's id. Built by hand: statuses.json declares none, and
	// a data file that did would be a change to the game rather than to this test.
	twin := statusBook.Kinds()[0]
	twin.ID = shared
	glosses := Vi.LogGlosses(skillBook.Skills(), append(statusBook.Kinds(), twin), nil)
	if name, named := glosses[shared]; named {
		t.Errorf("a colliding id was glossed as %q; it must be left out, because "+
			"nothing here can tell which kind the event printing it meant", name)
	}
	// Every other id still answers: a collision costs one name and not the map.
	if glosses["poison"] == "" {
		t.Error("a collision took the rest of the map with it")
	}
	collisions := LogGlossCollisions(skillBook.Skills(), append(statusBook.Kinds(), twin), nil)
	if len(collisions) != 1 || collisions[0] != shared {
		t.Errorf("the collisions came back as %v, want [%s]", collisions, shared)
	}
}

// TestEnglishGetsNoLogGlossesAtAll is the English contract at its source: the nil
// is the answer, not an absence of one.
func TestEnglishGetsNoLogGlossesAtAll(t *testing.T) {
	skillBook, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}
	statusBook, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	passiveBook, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load passives: %v", err)
	}
	if got := En.LogGlosses(skillBook.Skills(), statusBook.Kinds(), passiveBook.All()); got != nil {
		t.Errorf("English came back with %d names, want nil", len(got))
	}
}

// TestTheLogGlossesNameEveryShippedID is what the whole mechanism rests on, and
// it is the counts as well as the coverage.
//
// ⚠️ **The counts are the finding this change was built on.** Lang.Gloss cannot
// name a skill: skillGloss holds the nineteen ids that shipped before
// skill.Skill.Name existed and **none of them is in skills.json**, so a log
// glossed through Gloss would name no skill at all. This asserts the three
// accessors each cover their own book, and logs the split so the next reader does
// not have to re-derive it.
func TestTheLogGlossesNameEveryShippedID(t *testing.T) {
	skillBook, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}
	statusBook, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	passiveBook, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load passives: %v", err)
	}
	glosses := Vi.LogGlosses(skillBook.Skills(), statusBook.Kinds(), passiveBook.All())

	inTable := 0
	for _, one := range skillBook.Skills() {
		if glosses[one.ID] == "" {
			t.Errorf("the shipped skill %s has no name for the log", one.ID)
		}
		if skillGloss[one.ID] != "" {
			inTable++
		}
	}
	if inTable != 0 {
		t.Errorf("%d shipped skills are in skillGloss; that table is the pre-name fallback "+
			"and the log must not come to depend on it", inTable)
	}
	for _, kind := range statusBook.Kinds() {
		if glosses[kind.ID] == "" {
			t.Errorf("the shipped status %s has no name for the log", kind.ID)
		}
	}
	for _, one := range passiveBook.All() {
		if glosses[one.ID] == "" {
			t.Errorf("the shipped trait %s has no name for the log", one.ID)
		}
	}
	t.Logf("glossed for the log: %d skills, %d statuses, %d traits (%d names in all)",
		len(skillBook.Skills()), len(statusBook.Kinds()), len(passiveBook.All()), len(glosses))
}
