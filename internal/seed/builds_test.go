package seed_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/seed"
)

// mustBuilds is the shipped catalogue, or a stop.
func mustBuilds(t *testing.T) *cast.BuildBook {
	t.Helper()
	book, err := seed.Builds()
	if err != nil {
		t.Fatalf("load the shipped builds: %v", err)
	}
	return book
}

// TestTheShippedBuildsAreTheOnesTheTestsMeasure is the join between the two
// halves of this work, and the reason the catalogue is data at all.
//
// The kits below were measured before they were shipped: fireBuild/dragonBuild in
// dragon_test.go, tankBuild/semiBuild in squirtle_test.go, poisonBuild/
// sustainBuild in bulbasaur_test.go. Those vars are the design record — hardcoded
// on purpose, so a change to the data cannot quietly rewrite what the claim was
// about. This test is what stops the catalogue a player reads and the kits the
// suite measures from being two different games.
//
// It compares the trait too. A kit is half a build: ballast belongs to the
// attacking Squirtle and endurance to the standing one, and swapping those two
// changes which build survives without touching a single skill.
func TestTheShippedBuildsAreTheOnesTheTestsMeasure(t *testing.T) {
	book := mustBuilds(t)
	for _, want := range []struct {
		id     string
		skills []string
		trait  string
	}{
		{"bulbasaur.poison", poisonBuild, "virulence"},
		{"bulbasaur.parasite", sustainBuild, "blood_thirst"},
		{"charmander.scorch", fireBuild, "blaze"},
		{"charmander.dragon", dragonBuild, "reckless"},
		{"squirtle.fortress", tankBuild, "thorns"},
		{"squirtle.ram", semiBuild, "ballast"},
		{"poliwag.flurry", flurryBuild, "blood_thirst"},
		{"poliwag.riptide", riptideBuild, "spiteful"},
		{"poliwag.chorus", chorusBuild, "composure"},
		{"machop.gamble", gambleBuild, "berserk"},
		{"machop.sure", sureBuild, "unyielding"},
		{"cleffa.mend", mendBuild, "composure"},
		{"cleffa.hex", hexBuild, "elusive"},
	} {
		build, known := book.Get(want.id)
		if !known {
			t.Errorf("the catalogue holds no build %q, which the suite measures", want.id)
			continue
		}
		if !slices.Equal(build.Skills, want.skills) {
			t.Errorf("the build %q ships %v and the suite measures %v",
				want.id, build.Skills, want.skills)
		}
		if !slices.Equal(build.Passives, []string{want.trait}) {
			t.Errorf("the build %q ships the trait %v and the suite measures %q",
				want.id, build.Passives, want.trait)
		}
	}
}

// TestEveryShippedBuildFillsItsSlotsAndSaysWhyItExists is the shape claim: a
// build is a full loadout with a name and a reason, because those are the only
// two things it adds over the learnset it draws from.
//
// The parser already refuses an overfull kit and an unlearned skill, so what is
// left to assert here is the half a parser cannot judge — that nobody shipped a
// direction with three skills, no trait, or an empty intent, all of which are
// legal files and useless entries.
func TestEveryShippedBuildFillsItsSlotsAndSaysWhyItExists(t *testing.T) {
	for _, build := range mustBuilds(t).All() {
		if len(build.Skills) != cast.SkillSlots {
			t.Errorf("the build %q spends %d of %d skill slots",
				build.ID, len(build.Skills), cast.SkillSlots)
		}
		if len(build.Passives) != cast.TraitSlots {
			t.Errorf("the build %q fills %d of %d trait slots, and a direction that "+
				"declines its trait is throwing away the choice",
				build.ID, len(build.Passives), cast.TraitSlots)
		}
		if build.Intent == "" {
			t.Errorf("the build %q says nothing about why it exists", build.ID)
		}
	}
}

// TestABuildIsACatalogueOfChoicesRatherThanOfKits is the claim that keeps the
// catalogue honest: a character listed there has at least two directions.
//
// One build for a character is not a build, it is that character's kit — nothing
// is being chosen, and a screen offering a single option is telling a player they
// have a decision they do not have. A character with none is the honest case
// (Naruto today): its learnset has no second direction yet, and inventing one to
// fill the row would be the lie this test is against.
func TestABuildIsACatalogueOfChoicesRatherThanOfKits(t *testing.T) {
	book := mustBuilds(t)
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	listed := 0
	for _, character := range characters.All() {
		builds := book.Of(character.ID)
		if len(builds) == 0 {
			continue
		}
		listed++
		if len(builds) < 2 {
			t.Errorf("%s has one build (%q), which is a kit rather than a choice",
				character.ID, builds[0].ID)
		}
	}
	if listed == 0 {
		t.Fatal("no character has a build, so the catalogue says nothing")
	}
	if listed*2 > book.Count() {
		t.Errorf("%d characters are listed and the catalogue holds %d builds",
			listed, book.Count())
	}
}
