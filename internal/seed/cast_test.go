package seed_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/testfixture"
)

func mustSpecies(t *testing.T) *cast.SpeciesBook {
	t.Helper()
	book, err := seed.Species()
	if err != nil {
		t.Fatalf("load shipped species: %v", err)
	}
	return book
}

func mustOrigins(t *testing.T) *cast.OriginBook {
	t.Helper()
	book, err := seed.Origins()
	if err != nil {
		t.Fatalf("load shipped origins: %v", err)
	}
	return book
}

func mustArchetypes(t *testing.T) *cast.ArchetypeBook {
	t.Helper()
	book, err := seed.Archetypes()
	if err != nil {
		t.Fatalf("load shipped archetypes: %v", err)
	}
	return book
}

func mustCast(t *testing.T) *cast.Book {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load shipped cast: %v", err)
	}
	return book
}

func mustLimits(t *testing.T) progression.Limits {
	t.Helper()
	limits, err := seed.ProgressionLimits()
	if err != nil {
		t.Fatalf("load shipped progression limits: %v", err)
	}
	return limits
}

// TestShippedArchetypesMatchTheReferenceProfiles ties the presets to the design
// figures the stat budget was reasoned about with, deliberately hardcoded here
// the way TestShippedProgressionLimits hardcodes the ceilings.
//
// Without it a preset could drift away from the profile it is named after with
// every test still passing, and the profile is what the golden report's
// hits-to-kill table was read from.
func TestShippedArchetypesMatchTheReferenceProfiles(t *testing.T) {
	design := []struct {
		id     string
		column int
		cap    progression.Values
	}{
		{"blighter", 1, profile(3800, 620, 520, 100, 140, 50)},
		{"scorcher", 0, profile(3100, 700, 400, 140, 150, 70)},
		{"warden", 0, profile(3600, 460, 640, 85, 135, 38)},
		{"summoner", 1, profile(3200, 540, 380, 120, 155, 62)},
		{"bruiser", 0, profile(3700, 660, 560, 90, 140, 40)},
		{"slugger", 0, profile(3300, 760, 460, 85, 210, 28)},
		{"mender", 0, profile(4200, 420, 380, 115, 155, 55)},
	}
	book := mustArchetypes(t)
	if got, want := len(book.All()), len(design); got != want {
		t.Fatalf("the shipped data declares %d presets, the design table lists %d", got, want)
	}
	for _, row := range design {
		archetype, known := book.Get(row.id)
		if !known {
			t.Errorf("no %q preset is shipped", row.id)
			continue
		}
		if archetype.Column != row.column {
			t.Errorf("%s sits in column %d, the design says %d", row.id, archetype.Column, row.column)
		}
		if got := archetype.Stats.At(progression.LevelCap); got != row.cap {
			t.Errorf("%s at the cap is %s, the design says %s", row.id, got, row.cap)
		}
	}
}

func profile(hp, attack, defense, speed, accuracy, dodge int64) progression.Values {
	return progression.Values{
		progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
		progression.Speed: speed, progression.Accuracy: accuracy, progression.Dodge: dodge,
	}
}

// TestEveryShippedArchetypeFitsTheBudget re-checks what ParseArchetypes already
// enforces, on purpose: a preset that does not fit hands every author a stat
// line that fails later, and this is the test that says so by name.
func TestEveryShippedArchetypeFitsTheBudget(t *testing.T) {
	limits, rules := mustLimits(t), mustRules(t)
	for _, archetype := range mustArchetypes(t).All() {
		if err := limits.CheckTable(archetype.Stats, rules); err != nil {
			t.Errorf("preset %s does not fit the budget: %v", archetype.ID, err)
		}
		// A level 1 line a quarter to a third of the cap is what keeps an
		// early battle legible; a base at the cap would make levelling pointless.
		base, capped := archetype.Stats.At(1), archetype.Stats.At(progression.LevelCap)
		for _, kind := range progression.Kinds() {
			if base[kind] <= 0 {
				t.Errorf("preset %s starts %s at %d", archetype.ID, kind, base[kind])
			}
			if base[kind] > capped[kind] {
				t.Errorf("preset %s has %s shrinking from %d to %d",
					archetype.ID, kind, base[kind], capped[kind])
			}
		}
	}
}

// TestEveryShippedCharacterResolves walks both ends of every evolution line,
// which is the pair of levels a placement is most likely to ask for.
func TestEveryShippedCharacterResolves(t *testing.T) {
	limits, rules := mustLimits(t), mustRules(t)
	characters := mustCast(t).All()
	if len(characters) == 0 {
		t.Fatal("no characters are shipped, so nothing here is being exercised")
	}
	for _, character := range characters {
		// Every ARM, not the one furthest form. A line that forks has two grown
		// ends and the budget bites on each, so asking for "the" furthest is a
		// refusal there rather than a pick — which is exactly what this used to
		// do, and it could not be reached until a line forked.
		for _, level := range []int{1, progression.LevelCap} {
			forms, err := character.FurthestAt(level)
			if err != nil {
				t.Errorf("%s at level %d: %v", character.ID, level, err)
				continue
			}
			for _, form := range forms {
				values, stage, err := character.Resolve(level, form.Name)
				if err != nil {
					t.Errorf("%s at level %d as %s: %v", character.ID, level, form.Name, err)
					continue
				}
				if stage.Name == "" {
					t.Errorf("%s at level %d landed in an unnamed stage", character.ID, level)
				}
				if err := limits.CheckValues(values, rules); err != nil {
					t.Errorf("%s at level %d as %s: %v", character.ID, level, stage.Name, err)
				}
			}
		}
		// Every stage boundary resolves to the stage that declares it, which is
		// the property the whole staged design rests on. Asked BY NAME, because
		// on a forking line the boundary is shared: Poliwrath and Politoed both
		// begin at the same level, and "whichever is furthest there" is two
		// answers.
		for _, stage := range character.Stages {
			_, reached, err := character.Resolve(stage.MinLevel, stage.Name)
			if err != nil {
				t.Errorf("%s at level %d: %v", character.ID, stage.MinLevel, err)
				continue
			}
			if reached.Name != stage.Name {
				t.Errorf("%s at level %d landed in %q, want %q",
					character.ID, stage.MinLevel, reached.Name, stage.Name)
			}
		}
	}
}

// TestEveryShippedCharacterNamesSomethingReal re-states what ParseBook enforces,
// so that a change loosening the parser fails here rather than shipping.
func TestEveryShippedCharacterNamesSomethingReal(t *testing.T) {
	origins, archetypes, characters := mustOrigins(t), mustArchetypes(t), mustCast(t)
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load shipped skills: %v", err)
	}
	for _, character := range characters.All() {
		if _, known := origins.Get(character.Origin); !known {
			t.Errorf("%s comes from the unknown origin %q", character.ID, character.Origin)
		}
		if _, known := archetypes.Get(character.Archetype); !known {
			t.Errorf("%s was tuned from the unknown archetype %q", character.ID, character.Archetype)
		}
		for _, entry := range character.Skills {
			if _, err := skills.Lookup(entry.ID); err != nil {
				t.Errorf("%s: %v", character.ID, err)
			}
		}
		if err := cast.ValidateImagePath(character.Image); err != nil {
			t.Errorf("%s: %v", character.ID, err)
		}
	}
}

// TestEveryShippedRestrictionNamesSomethingReal checks the names inside a
// skill's allowlists, for every skill in the book rather than only the carried
// ones.
//
// The parsers deliberately check less than this. cast.checkCharacterRestrictions
// only looks at skills somebody carries, because a unique skill cannot be
// written before the character it names and that character cannot be written
// before its kit -- checking every skill would deadlock authoring. The shipped
// data has no such excuse: everything it names exists, so a typo in an allowlist
// nobody carries yet is a typo, and without this it would sit unread until the
// day somebody picked the skill up and got refused for being the wrong thing.
func TestEveryShippedRestrictionNamesSomethingReal(t *testing.T) {
	archetypes, characters := mustArchetypes(t), mustCast(t)
	species, origins := mustSpecies(t), mustOrigins(t)
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load shipped skills: %v", err)
	}
	for _, carried := range skills.Skills() {
		if carried.Restrict == nil {
			continue
		}
		for _, id := range carried.Restrict.Archetypes {
			if _, known := archetypes.Get(id); !known {
				t.Errorf("%s is kept for the archetype %q, which no preset has", carried.ID, id)
			}
		}
		for _, id := range carried.Restrict.Characters {
			if _, known := characters.Get(id); !known {
				t.Errorf("%s is kept for the character %q, which the cast does not hold", carried.ID, id)
			}
		}
		for _, id := range carried.Restrict.SpeciesNames() {
			if _, known := species.Get(id); !known {
				t.Errorf("%s is kept for the species %q, which the catalog does not declare", carried.ID, id)
			}
		}
		for _, id := range carried.Restrict.OriginNames() {
			if _, known := origins.Get(id); !known {
				t.Errorf("%s is kept for the origin %q, which the catalog does not declare", carried.ID, id)
			}
		}
	}
}

// TestEveryShippedArchetypeKitIsCarryableAtAll covers the half of the rule
// ParseArchetypes cannot reach.
//
// The parser rejects a kit demanding more than two elements, because no affinity
// can hold three. What it cannot check is whether the two it demands are allowed
// to sit together: that needs the element chart, and a preset is validated
// without one. So a kit demanding exactly two is checked here, against the
// derived Demands rather than a second copy of the derivation.
func TestEveryShippedArchetypeKitIsCarryableAtAll(t *testing.T) {
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load shipped element chart: %v", err)
	}
	for _, archetype := range mustArchetypes(t).All() {
		switch len(archetype.Demands) {
		case 0, 1:
			// Anything from a single element upwards carries this kit.
		case 2:
			pair, err := element.Dual(archetype.Demands[0], archetype.Demands[1])
			if err != nil {
				t.Errorf("preset %s demands %v, which cannot be one affinity: %v",
					archetype.ID, archetype.DemandNames(), err)
				continue
			}
			if err := chart.ValidateAffinity(pair); err != nil {
				t.Errorf("preset %s demands %v, which the chart refuses as an affinity: %v",
					archetype.ID, archetype.DemandNames(), err)
			}
		default:
			t.Errorf("preset %s demands %d elements and still parsed, which ParseArchetypes should have refused",
				archetype.ID, len(archetype.Demands))
		}
	}
}

// TestEveryShippedCharacterMayCarryItsKit re-checks through skill.CanCarry what
// cast.ParseBook already applied, so a change loosening the parser fails here
// rather than shipping a character battle.New will refuse.
func TestEveryShippedCharacterMayCarryItsKit(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load shipped skills: %v", err)
	}
	for _, character := range mustCast(t).All() {
		for _, entry := range character.Skills {
			known, err := skills.Lookup(entry.ID)
			if err != nil {
				t.Fatalf("%s: %v", character.ID, err)
			}
			if !skill.CanCarry(character.Element, known) {
				t.Errorf("%s is %s and carries %s, which is %s: battle.New will refuse it",
					character.ID, character.Element, entry.ID, known.Element)
			}
		}
	}
}

// TestEveryShippedCharacterCoversItsPresetDemand is the property that makes the
// wizard's element prompt honest: a character sitting on a preset's kit must
// carry everything that kit demands.
func TestEveryShippedCharacterCoversItsPresetDemand(t *testing.T) {
	archetypes, characters := mustArchetypes(t), mustCast(t)
	for _, character := range characters.All() {
		preset, known := archetypes.Get(character.Archetype)
		if !known {
			t.Errorf("%s names the unknown preset %q", character.ID, character.Archetype)
			continue
		}
		if !sameStrings(cast.LearnedIDs(character.Skills), preset.Skills) {
			// A substituted kit is allowed; only the kit it actually carries
			// binds it, and the test above covers that.
			continue
		}
		for _, member := range preset.Demands {
			if !character.Element.Has(member) {
				t.Errorf("%s carries the %s preset's kit unchanged but is %s, missing %s",
					character.ID, preset.ID, character.Element, member)
			}
		}
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// TestShippedOriginsAreUsable checks the catalog itself, which nothing else
// reaches: an origin nobody has borrowed from is fine, an origin nobody could
// borrow from is not.
func TestShippedOriginsAreUsable(t *testing.T) {
	origins := mustOrigins(t).All()
	if len(origins) == 0 {
		t.Fatal("the origin catalog is empty, so an author has nothing to point at")
	}
	seen := make(map[string]bool, len(origins))
	for _, origin := range origins {
		if origin.Title == "" {
			t.Errorf("origin %s has no title", origin.ID)
		}
		if !origin.Medium.Valid() {
			t.Errorf("origin %s has the unusable medium %s", origin.ID, origin.Medium)
		}
		if seen[origin.ID] {
			t.Errorf("origin %s appears twice", origin.ID)
		}
		seen[origin.ID] = true
	}
}

// referenceRoster places a shipped character by reference, which is the form an
// authored cast is placed with.
//
// It names no character: it asks the shipped book for one. Naming a row coupled
// this test to content the author is free to change, and editing the cast broke
// tests that had nothing to do with it.
// loadoutOf is the first few skills a character has learned by a level, as the
// JSON a placement writes. A test that only wants a roster to parse should not
// have to know which four to bring — but it does have to bring some, because a
// placement is a choice and this file is where that rule is exercised.
func loadoutOf(t *testing.T, character cast.Character, level int) string {
	t.Helper()
	known := character.SkillsAt(level, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}
	raw, err := json.Marshal(known)
	if err != nil {
		t.Fatalf("marshal a loadout: %v", err)
	}
	return string(raw)
}

func referenceRoster(t *testing.T, characters *cast.Book) ([]byte, cast.Character) {
	t.Helper()
	held := characters.All()
	if len(held) == 0 {
		t.Skip("the shipped cast is empty, so there is no reference to place")
	}
	first := held[0]
	// One on each side, because a battle needs an opponent and a cast of one is
	// the smallest a shipped book is allowed to be.
	loadout := loadoutOf(t, first, progression.LevelCap)
	return []byte(fmt.Sprintf(`{
  "units": [
    {"id": "ally.one", "character": %q, "level": 60, "side": "ally", "slot": [2, 1], "skills": %s},
    {"id": "foe.one", "character": %q, "level": 60, "side": "enemy", "slot": [2, 1], "skills": %s}
  ]
}`, first.ID, loadout, first.ID, loadout)), first
}

func TestParseRosterResolvesACharacterReference(t *testing.T) {
	characters := mustCast(t)
	raw, adept := referenceRoster(t, characters)
	roster, err := seed.ParseRoster(raw, characters)
	if err != nil {
		t.Fatalf("parse the reference roster: %v", err)
	}
	if len(roster) != 2 {
		t.Fatalf("the roster resolved to %d units, want 2", len(roster))
	}
	wanted, fielded, err := adept.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve %s: %v", adept.ID, err)
	}
	// The form fielded, not the character. The stat line beside the name is the
	// form's, so a placement named for the character reads as a first form with
	// a last form's numbers.
	if roster[0].Name != fielded.Name {
		t.Errorf("the placement is named %q, want the fielded form's %q", roster[0].Name, fielded.Name)
	}
	// A line whose last form is named after the character proves nothing here:
	// the two names it could have picked are the same string. Said out loud so a
	// re-authored cast fails here rather than quietly turning the check above
	// into a tautology.
	if len(adept.Stages) > 1 && fielded.Name == adept.Name {
		t.Errorf("%s reaches %q at the cap, which is its own name, so naming the form and naming "+
			"the character cannot be told apart", adept.ID, fielded.Name)
	}
	if roster[0].Stats != wanted {
		t.Errorf("the placement resolved to %s, the character resolves to %s", roster[0].Stats, wanted)
	}
	if roster[0].Affinity.String() != adept.Element.String() {
		t.Errorf("the placement is %s, the character is %s", roster[0].Affinity, adept.Element)
	}
	// A placement chooses from what the character knows rather than carrying all
	// of it, so what is checked is that every chosen skill is one the character
	// has — the loadout itself is the placement's decision, not the sheet's.
	for _, carried := range roster[0].Skills {
		if !slices.Contains(cast.LearnedIDs(adept.Skills), carried) {
			t.Errorf("the placement carries %q, which %s never learns", carried, adept.ID)
		}
	}
	// A reference resolves the whole evolution line, not the last stage. The
	// level comes from the character's own line rather than a number written
	// here, so the property survives the cast being re-authored; a single-stage
	// cast simply has nothing to prove.
	if len(adept.Stages) > 1 {
		second := adept.Stages[1]
		staged := []byte(fmt.Sprintf(`{
  "units": [
    {"id": "ally.one", "character": %q, "level": %d, "side": "ally", "slot": [2, 1], "skills": %s}
  ]
}`, adept.ID, second.MinLevel, loadoutOf(t, adept, second.MinLevel)))
		placed, err := seed.ParseRoster(staged, characters)
		if err != nil {
			t.Fatalf("parse at the stage boundary: %v", err)
		}
		wantedLate, stage, err := adept.Resolve(second.MinLevel, progression.Furthest)
		if err != nil {
			t.Fatalf("resolve at level %d: %v", second.MinLevel, err)
		}
		if stage.Name != second.Name {
			t.Fatalf("level %d lands in stage %q, want %q", second.MinLevel, stage.Name, second.Name)
		}
		if placed[0].Stats != wantedLate {
			t.Errorf("the placement resolved to %s, want %s", placed[0].Stats, wantedLate)
		}
		// And the name follows the form down the line rather than staying on the
		// character: the same character placed at two levels is two names.
		if placed[0].Name != stage.Name {
			t.Errorf("the placement at level %d is named %q, want the form it fields, %q",
				second.MinLevel, placed[0].Name, stage.Name)
		}
		if placed[0].Name == roster[0].Name {
			t.Errorf("%s is named %q at level %d and at the cap, so the name is not reading the form",
				adept.ID, placed[0].Name, second.MinLevel)
		}
	}
	// A referenced roster has to build a real battle, not merely parse: the
	// engine applies its own rules about affinities, slots and kits.
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	if _, err := battle.New(books, 11, roster); err != nil {
		t.Fatalf("a referenced roster did not make a battle: %v", err)
	}
}

func TestParseRosterRejections(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		wantIn string
	}{
		{
			// Two sources for one number is how the two drift apart, so the
			// mixture is refused rather than resolved by precedence.
			name: "a character reference that also restates its stats",
			raw: `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 60,
			       "side": "ally", "slot": [2, 1],
			       "stats": {"hp": 1, "attack": 1, "defense": 1, "speed": 1, "accuracy": 1, "dodge": 1}}]}`,
			wantIn: "restates [stats]",
		},
		{
			name: "a character reference that also restates its name",
			raw: `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 60,
			       "side": "ally", "slot": [2, 1], "name": "Something Else"}]}`,
			wantIn: "restates [name]",
		},
		{
			name: "a character reference that also restates its element and skills",
			raw: `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 60,
			       "side": "ally", "slot": [2, 1], "element": "fire", "skills": ["strike"]}]}`,
			wantIn: "restates [element]",
		},
		{
			name:   "a character reference with no level",
			raw:    `{"units": [{"id": "a", "character": "fixture-anime.adept", "side": "ally", "slot": [2, 1]}]}`,
			wantIn: "gives no level",
		},
		{
			name: "a character reference past the level cap",
			raw: `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 61,
			       "side": "ally", "slot": [2, 1]}]}`,
			wantIn: "outside 1..60",
		},
		{
			name: "a character reference at level zero",
			raw: `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 0,
			       "side": "ally", "slot": [2, 1]}]}`,
			wantIn: "outside 1..60",
		},
		{
			name:   "an unknown character",
			raw:    `{"units": [{"id": "a", "character": "nobody", "level": 10, "side": "ally", "slot": [2, 1]}]}`,
			wantIn: "unknown character",
		},
		{
			name:   "a level on an inline entry, which is already resolved",
			raw:    `{"units": [{"id": "a", "level": 10, "side": "ally", "slot": [2, 1], "name": "A"}]}`,
			wantIn: "gives a level but no character",
		},
		{
			name:   "an inline entry with nothing in it",
			raw:    `{"units": [{"id": "a", "side": "ally", "slot": [2, 1]}]}`,
			wantIn: "needs a name, an element and a stat line",
		},
		{
			name:   "an unknown side",
			raw:    `{"units": [{"id": "a", "character": "fixture-anime.adept", "level": 10, "side": "neither", "slot": [2, 1]}]}`,
			wantIn: "unknown side",
		},
	}
	characters := mustCast(t)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := seed.ParseRoster([]byte(test.raw), characters)
			if err == nil {
				t.Fatalf("the roster parsed, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

// TestTheInlineRosterFormStillWorks records that a roster may still spell its
// units out rather than referencing the cast.
//
// The shipped roster stopped using it: with a real cast, a placement says which
// character stands there. The form has to keep working anyway -- it is what the
// bench uses, and the bench is what the engine tests fight on -- so this reads
// the bench rather than the shipped file it used to read.
func TestTheInlineRosterFormStillWorks(t *testing.T) {
	characters := mustCast(t)
	roster, err := seed.ParseRoster([]byte(testfixture.Roster), characters)
	if err != nil {
		t.Fatalf("parse the inline roster: %v", err)
	}
	if len(roster) == 0 {
		t.Fatal("the inline roster resolved to nothing")
	}
	for _, unit := range roster {
		if unit.Name == "" {
			t.Errorf("unit %s resolved without a name", unit.ID)
		}
		if len(unit.Skills) == 0 {
			t.Errorf("unit %s resolved without a kit", unit.ID)
		}
	}
}

func TestSpeciesGolden(t *testing.T) {
	compareGolden(t, "species.golden", speciesReport(mustSpecies(t), mustCast(t)))
}

func TestOriginsGolden(t *testing.T) {
	compareGolden(t, "origins.golden", originsReport(mustOrigins(t), mustCast(t)))
}

func TestArchetypesGolden(t *testing.T) {
	compareGolden(t, "archetypes.golden",
		archetypesReport(mustArchetypes(t), mustLimits(t), mustRules(t)))
}

func TestCastGolden(t *testing.T) {
	compareGolden(t, "cast.golden",
		castReport(mustCast(t), mustOrigins(t), mustLimits(t), mustRules(t)))
}

// compareGolden is the same accept-or-compare dance the older golden tests
// spell out inline; the three new tables share it rather than repeating it.
func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// speciesReport is the catalog with its carriers under it, which is the pairing
// that makes the table worth reading: a species nobody is is not an error, and a
// skill kept for one nobody is, is unplayable -- and only this report shows both
// halves at once.
func speciesReport(species *cast.SpeciesBook, characters *cast.Book) string {
	var b strings.Builder
	b.WriteString("== what a unit can be ==\n")
	for _, kind := range species.All() {
		fmt.Fprintf(&b, "%-16s %s\n", kind.ID, kind.Name)
		are := characters.OfSpecies(kind.ID)
		if len(are) == 0 {
			b.WriteString("                 nobody is one yet\n")
		}
		for _, character := range are {
			fmt.Fprintf(&b, "                 %s (%s)\n", character.ID, character.Name)
		}
		if kind.Note != "" {
			fmt.Fprintf(&b, "                 note: %s\n", kind.Note)
		}
	}
	fmt.Fprintf(&b, "\n%d kinds declared\n", len(species.All()))
	return b.String()
}

func originsReport(origins *cast.OriginBook, characters *cast.Book) string {
	var b strings.Builder
	b.WriteString("== the works the cast is borrowed from ==\n")
	for _, origin := range origins.All() {
		year := "undated"
		if origin.Year != 0 {
			year = fmt.Sprintf("%d", origin.Year)
		}
		fmt.Fprintf(&b, "%-16s %-7s %-8s %s\n", origin.ID, origin.Medium, year, origin.Title)
		borrowed := characters.OfOrigin(origin.ID)
		if len(borrowed) == 0 {
			b.WriteString("                 nobody borrowed from it yet\n")
		}
		for _, character := range borrowed {
			fmt.Fprintf(&b, "                 %s (%s)\n", character.ID, character.Name)
		}
		if origin.Note != "" {
			fmt.Fprintf(&b, "                 note: %s\n", origin.Note)
		}
	}
	fmt.Fprintf(&b, "\n%d works, %d media declared: %s\n",
		len(origins.All()), cast.MediumCount, strings.Join(cast.MediumNames(), " "))
	return b.String()
}

func archetypesReport(archetypes *cast.ArchetypeBook, limits progression.Limits, rules combat.Rules) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== role presets ==\ncurves read base at level 1 to max at level %d\n\n",
		progression.LevelCap)
	for _, archetype := range archetypes.All() {
		fmt.Fprintf(&b, "%s, formation column %d\n  %s\n", archetype.ID, archetype.Column, archetype.Role)
		// The demand is derived from the kit by skill.Demands, never authored.
		// It belongs in the record because it is a design constraint on every
		// character built from the preset, not a property of the preset alone.
		demanded := "any element carries this kit"
		if names := archetype.DemandNames(); len(names) > 0 {
			demanded = "a character built from it must carry " + strings.Join(names, " and ")
		}
		fmt.Fprintf(&b, "  demands  %s\n", demanded)
		curves := make([]string, 0, progression.KindCount)
		for _, kind := range progression.Kinds() {
			curve := archetype.Stats[kind]
			curves = append(curves, fmt.Sprintf("%s %d->%d", kind, curve.Base, curve.Max))
		}
		fmt.Fprintf(&b, "  %s\n", strings.Join(curves, "  "))
		for _, level := range []int{1, 30, progression.LevelCap} {
			values := archetype.Stats.At(level)
			effective := progression.EffectiveHP(values, rules)
			fmt.Fprintf(&b, "  level %-2d %s\n", level, values)
			fmt.Fprintf(&b, "           absorbs %d of the %d budget, %d to spare\n",
				effective, limits.MaxEffectiveHP, limits.MaxEffectiveHP-effective)
		}
		fmt.Fprintf(&b, "  kit      %s\n\n", strings.Join(archetype.Skills, " "))
	}
	fmt.Fprintf(&b, "%d presets\n", len(archetypes.All()))
	return b.String()
}

func castReport(characters *cast.Book, origins *cast.OriginBook, limits progression.Limits, rules combat.Rules) string {
	var b strings.Builder
	b.WriteString("== the authored cast ==\n\n")
	for _, character := range characters.All() {
		title := character.Origin
		if origin, known := origins.Get(character.Origin); known {
			title = fmt.Sprintf("%s (%s)", origin.Title, origin.Medium)
		}
		fmt.Fprintf(&b, "%s — %s\n", character.ID, character.Name)
		// Printed only where it is set, and first, because it qualifies
		// everything under it: a record that cannot show a character has been
		// taken out of the authoring lists is a record that reads as though
		// every character in it were on offer. The other way round from the
		// species line below — an absent species is a real answer about the
		// character and has to be worded, while an absent flag is the ordinary
		// case and saying "offered" on every one of them is noise. What that
		// buys is a diff: the day somebody hides a character, the design record
		// moves by exactly one line.
		if character.Hidden {
			fmt.Fprintf(&b, "  hidden   held back from the lists an author builds a side out of\n")
		}
		fmt.Fprintf(&b, "  from     %s\n", title)
		fmt.Fprintf(&b, "  preset   %s\n", character.Archetype)
		fmt.Fprintf(&b, "  element  %s\n", character.Element)
		// Printed even when there is none, and worded rather than left blank: a
		// character that is nothing in particular is a real answer -- it is what
		// a skill kept for a lineage refuses -- and a blank field reads as a
		// record that forgot to ask.
		kinds := "nothing in particular"
		if len(character.Species) > 0 {
			kinds = strings.Join(character.Species, " ")
		}
		fmt.Fprintf(&b, "  species  %s\n", kinds)
		fmt.Fprintf(&b, "  art      %s\n", character.Image)
		fmt.Fprintf(&b, "  kit      %s\n", forge.UnlockSummary(character.Skills))
		for i, stage := range character.Stages {
			// A stage is reported over the levels it actually owns, not up to
			// the cap: a stage that is superseded at 30 never reaches level 60,
			// and printing what its curve would have said there invites a
			// comparison nobody will ever play.
			last := progression.LevelCap
			if i+1 < len(character.Stages) {
				last = character.Stages[i+1].MinLevel - 1
			}
			// The art each form shows, resolved rather than declared: a stage
			// that names none shows the character's, and a record that printed
			// the empty field instead would read as "this form has no picture".
			fmt.Fprintf(&b, "  stage %q owns levels %d to %d, showing %s\n",
				stage.Name, stage.MinLevel, last, character.StageArt(stage))
			levels := []int{stage.MinLevel, last}
			if stage.MinLevel == last {
				levels = levels[:1]
			}
			for _, level := range levels {
				values := stage.Stats.At(level)
				effective := progression.EffectiveHP(values, rules)
				fmt.Fprintf(&b, "    level %-2d %s\n", level, values)
				fmt.Fprintf(&b, "             absorbs %d of the %d budget, %d to spare\n",
					effective, limits.MaxEffectiveHP, limits.MaxEffectiveHP-effective)
			}
		}
		b.WriteString("\n")
	}
	// "1 characters" was in this record for as long as there was one character
	// in it. The plural is a word rather than a number, so it is chosen here.
	if count := len(characters.All()); count == 1 {
		b.WriteString("1 character\n")
	} else {
		fmt.Fprintf(&b, "%d characters\n", count)
	}
	return b.String()
}

// TestBulbasaurCanBeBuiltTwoWays is the scenario the trait work was for, asserted
// rather than described: at the cap the character offers traits from two
// different builds, and a placement carries exactly one, so bringing the poison
// answer means not bringing the sustain and the other way round.
//
// It measures the *choice* rather than any one trait, because a choice is what a
// slot is. Before the slot existed a character brought everything it had, and
// every trait added made one unit better rather than two units different — this
// fails the day that comes back, whether by the slot count moving or by a
// direction losing its last entry.
func TestBulbasaurCanBeBuiltTwoWays(t *testing.T) {
	book := mustCast(t)
	character, known := book.Get("pokemon.bulbasaur")
	if !known {
		t.Fatal("the shipped cast holds no bulbasaur")
	}
	available := character.PassivesAt(progression.LevelCap, progression.Furthest)

	builds := map[string][]string{
		"poison":    {"venom_blood", "virulence"},
		"sustain":   {"blood_thirst", "last_gasp"},
		"endurance": {"endurance"},
	}
	for name, wanted := range builds {
		found := 0
		for _, id := range wanted {
			if slices.Contains(available, id) {
				found++
			}
		}
		if found == 0 {
			t.Errorf("nothing in the %s build is available at the cap; it holds %v", name, available)
		}
	}

	// One slot, so the builds are alternatives rather than a list. If this ever
	// grows, every trait above becomes an addition and the scenario stops being
	// a choice.
	if cast.TraitSlots != 1 {
		t.Errorf("a placement carries %d traits, and two builds are only a choice at one",
			cast.TraitSlots)
	}
	if len(available) < 2 {
		t.Errorf("%d traits are available at the cap, so the one slot decides nothing",
			len(available))
	}
}

// TestABiographyCarriesNoFigure closes, for a character's prose, the hole the
// digit ban already closes for a skill's.
//
// Every number about a character is derived and drawn on the pane the biography
// sits in: the stages row prints where each form starts, the effective-hp row
// prints what the stat line is worth, and the level the browser is walking is on
// screen the whole time. A figure written into the prose says the same thing
// twice until the day it stops being true — and the shipped biographies were
// doing exactly that, each ending "Ivysaur từ cấp 16, Venusaur từ cấp 32", which
// one edit to a stage's min_level turns into a lie nothing would have caught.
func TestABiographyCarriesNoFigure(t *testing.T) {
	checked := 0
	for _, character := range mustCast(t).All() {
		if character.Bio == "" {
			continue
		}
		checked++
		for _, letter := range character.Bio {
			if unicode.IsDigit(letter) {
				t.Errorf("%s: its biography writes the figure %q, and every figure about a character is derived beside it",
					character.ID, string(letter))
				break
			}
		}
	}
	if checked == 0 {
		t.Fatal("no shipped character has a biography, so this measures nothing")
	}
}

// TestABiographyNamesNoLaterForm is the other half of the same restatement, and
// the half a digit ban cannot see.
//
// The stages row already lists every form and where it starts. A biography that
// walks the evolution line in prose is that row again, in worse form and without
// its numbers — and it goes stale the moment a form is renamed or a fourth added.
//
// The **first** form is free, and deliberately: it is what the character is
// called before anything happens, so "Charmander giấu được mọi thứ trừ tâm
// trạng" is the creature's name rather than a list of what it turns into.
func TestABiographyNamesNoLaterForm(t *testing.T) {
	checked := 0
	for _, character := range mustCast(t).All() {
		if character.Bio == "" || len(character.Stages) < 2 {
			continue
		}
		checked++
		for _, stage := range character.Stages[1:] {
			if strings.Contains(character.Bio, stage.Name) {
				t.Errorf("%s: its biography names the later form %q, which the stages row beside it already lists",
					character.ID, stage.Name)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no shipped character has both a biography and a second form, so this measures nothing")
	}
}
