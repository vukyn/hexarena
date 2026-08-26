package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/testfixture"
)

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
		for _, level := range []int{1, progression.LevelCap} {
			values, stage, err := character.Resolve(level)
			if err != nil {
				t.Errorf("%s at level %d: %v", character.ID, level, err)
				continue
			}
			if stage.Name == "" {
				t.Errorf("%s at level %d landed in an unnamed stage", character.ID, level)
			}
			if err := limits.CheckValues(values, rules); err != nil {
				t.Errorf("%s at level %d: %v", character.ID, level, err)
			}
		}
		// Every stage boundary resolves to the stage that declares it, which is
		// the property the whole staged design rests on.
		for _, stage := range character.Stages {
			_, reached, err := character.Resolve(stage.MinLevel)
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
		for _, id := range character.Skills {
			if _, err := skills.Lookup(id); err != nil {
				t.Errorf("%s: %v", character.ID, err)
			}
		}
		if err := cast.ValidateImagePath(character.Image); err != nil {
			t.Errorf("%s: %v", character.ID, err)
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
		for _, id := range character.Skills {
			known, err := skills.Lookup(id)
			if err != nil {
				t.Fatalf("%s: %v", character.ID, err)
			}
			if !skill.CanCarry(character.Element, known) {
				t.Errorf("%s is %s and carries %s, which is %s: battle.New will refuse it",
					character.ID, character.Element, id, known.Element)
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
		if !sameStrings(character.Skills, preset.Skills) {
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
func referenceRoster(t *testing.T, characters *cast.Book) ([]byte, cast.Character) {
	t.Helper()
	held := characters.All()
	if len(held) == 0 {
		t.Skip("the shipped cast is empty, so there is no reference to place")
	}
	first := held[0]
	// One on each side, because a battle needs an opponent and a cast of one is
	// the smallest a shipped book is allowed to be.
	return []byte(fmt.Sprintf(`{
  "units": [
    {"id": "ally.one", "character": %q, "level": 60, "side": "ally", "slot": [2, 1]},
    {"id": "foe.one", "character": %q, "level": 60, "side": "enemy", "slot": [2, 1]}
  ]
}`, first.ID, first.ID)), first
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
	wanted, _, err := adept.Resolve(progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve %s: %v", adept.ID, err)
	}
	if roster[0].Name != adept.Name {
		t.Errorf("the placement is named %q, want the character's %q", roster[0].Name, adept.Name)
	}
	if roster[0].Stats != wanted {
		t.Errorf("the placement resolved to %s, the character resolves to %s", roster[0].Stats, wanted)
	}
	if roster[0].Affinity.String() != adept.Element.String() {
		t.Errorf("the placement is %s, the character is %s", roster[0].Affinity, adept.Element)
	}
	if strings.Join(roster[0].Skills, " ") != strings.Join(adept.Skills, " ") {
		t.Errorf("the placement carries %v, the character knows %v", roster[0].Skills, adept.Skills)
	}
	// A reference resolves the whole evolution line, not the last stage. The
	// level comes from the character's own line rather than a number written
	// here, so the property survives the cast being re-authored; a single-stage
	// cast simply has nothing to prove.
	if len(adept.Stages) > 1 {
		second := adept.Stages[1]
		staged := []byte(fmt.Sprintf(`{
  "units": [
    {"id": "ally.one", "character": %q, "level": %d, "side": "ally", "slot": [2, 1]}
  ]
}`, adept.ID, second.MinLevel))
		placed, err := seed.ParseRoster(staged, characters)
		if err != nil {
			t.Fatalf("parse at the stage boundary: %v", err)
		}
		wantedLate, stage, err := adept.Resolve(second.MinLevel)
		if err != nil {
			t.Fatalf("resolve at level %d: %v", second.MinLevel, err)
		}
		if stage.Name != second.Name {
			t.Fatalf("level %d lands in stage %q, want %q", second.MinLevel, stage.Name, second.Name)
		}
		if placed[0].Stats != wantedLate {
			t.Errorf("the placement resolved to %s, want %s", placed[0].Stats, wantedLate)
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
			wantIn: "restates [element skills]",
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
		fmt.Fprintf(&b, "  from     %s\n", title)
		fmt.Fprintf(&b, "  preset   %s\n", character.Archetype)
		fmt.Fprintf(&b, "  element  %s\n", character.Element)
		fmt.Fprintf(&b, "  art      %s\n", character.Image)
		fmt.Fprintf(&b, "  kit      %s\n", strings.Join(character.Skills, " "))
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
