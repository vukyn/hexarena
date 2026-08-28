package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/testfixture"
)

// shippedDataDir is the real data directory, relative to this package.
const shippedDataDir = "../seed/data"

// scratchData copies the shipped data into a temporary directory, so a test may
// write to it without touching the repository.
func scratchData(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	copyTree(t, shippedDataDir, target)
	// The fixture is what the tests name. Before this they named the characters
	// the repository shipped, so editing the real cast broke tests that had
	// nothing to do with it.
	if err := testfixture.Inject(target, func() (testfixture.Saver, error) {
		return Load(target)
	}); err != nil {
		t.Fatalf("inject the fixture: %v", err)
	}
	return target
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		t.Fatalf("read %s: %v", from, err)
	}
	for _, entry := range entries {
		source, destination := filepath.Join(from, entry.Name()), filepath.Join(to, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				t.Fatalf("create %s: %v", destination, err)
			}
			copyTree(t, source, destination)
			continue
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read %s: %v", source, err)
		}
		if err := os.WriteFile(destination, raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", destination, err)
		}
	}
}

// TestInspectPassesOnTheShippedData is the check that would have caught an
// example character pointing at art nobody committed.
func TestInspectPassesOnTheShippedData(t *testing.T) {
	report, err := Inspect(shippedDataDir)
	if err != nil {
		t.Fatalf("the shipped data does not load: %v", err)
	}
	if !report.OK() {
		t.Errorf("the shipped data has problems: %v", report.Problems)
	}
	if len(report.Rows) == 0 {
		t.Fatal("no characters were inspected, so nothing here is being exercised")
	}
	for _, row := range report.Rows {
		if len(row.Art) == 0 {
			t.Errorf("%s was inspected with no art at all", row.ID)
		}
		for _, art := range row.Art {
			if !art.Exists {
				t.Errorf("%s names art that is not there: %s (stage %q)", row.ID, art.Image, art.Stage)
			}
		}
		if row.Failure != nil {
			t.Errorf("%s does not resolve: %v", row.ID, row.Failure)
		}
		if row.Budget.Over() {
			t.Errorf("%s absorbs %d, over the budget", row.ID, row.Budget.Effective)
		}
	}
}

// TestInspectWarnsAboutAKitThatCannotCoverTheBoard is the authoring half of the
// draw: the case worth catching where an error is cheapest.
//
// It is a warning rather than a problem on purpose. A short-ranged character on
// a back column is a design an author may well mean, because the squad in front
// of it is what does the reaching — so failing the check would refuse a legal
// game, and saying nothing would leave the shape a battle nobody can act in is
// built out of entirely unremarked.
func TestInspectWarnsAboutAKitThatCannotCoverTheBoard(t *testing.T) {
	dir := scratchData(t)
	before, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect the copy: %v", err)
	}
	if len(before.Warnings) != 0 {
		t.Fatalf("the fixture already warns, so this measures nothing: %v", before.Warnings)
	}

	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// A skirmisher stands on the back column, three cells from the nearest
	// enemy. Give this one nothing that reaches past one.
	if preset, found := lib.Archetypes().Get("skirmisher"); !found || preset.Column != 0 {
		t.Fatal("the skirmisher preset is not on the back column, so this measures nothing")
	}
	character, err := Draft{
		ID: "fixture-game.stub", Name: "Stub", Origin: "fixture-game",
		Archetype: "skirmisher", Image: "assets/fixture/sprout.svg",
		Element: "grass", Bio: "Written by a test.", Skills: "venom_fang",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve a short-ranged skirmisher: %v", err)
	}
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save the shortened kit: %v", err)
	}

	after, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect after shortening the kit: %v", err)
	}
	// A warning is not a failure, and the check still passes.
	if !after.OK() {
		t.Errorf("a short kit failed the check: %v", after.Problems)
	}
	if len(after.Warnings) != 1 {
		t.Fatalf("%d warnings reported, want 1: %v", len(after.Warnings), after.Warnings)
	}
	short, isShort := after.Warnings[0].(*ShortReachWarning)
	if !isShort {
		t.Fatalf("the warning is a %T, want a *ShortReachWarning", after.Warnings[0])
	}
	if short.ID != character.ID || short.Range != 1 {
		t.Errorf("the warning reads %+v", short)
	}
	if !strings.Contains(short.Error(), character.ID) {
		t.Errorf("the warning %q does not name the character", short)
	}
}

// TestInspectNoticesMissingArt is the one question internal/core/cast is not
// allowed to ask, so it has to be asked here.
func TestInspectNoticesMissingArt(t *testing.T) {
	dir := scratchData(t)
	before, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect the copy: %v", err)
	}
	if !before.OK() {
		t.Fatalf("the copied data already has problems: %v", before.Problems)
	}

	// Take away exactly one file. The path shape is unchanged, so only a
	// program that reads the filesystem can notice.
	removed := filepath.Join(dir, "assets", "fixture", "sprout.svg")
	if err := os.Remove(removed); err != nil {
		t.Fatalf("remove %s: %v", removed, err)
	}
	after, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect after removing the art: %v", err)
	}
	if after.OK() {
		t.Fatal("the missing art was not noticed")
	}
	if len(after.Problems) != 1 {
		t.Errorf("%d problems reported, want 1: %v", len(after.Problems), after.Problems)
	}
	// The problem is a value rather than a sentence, so a front-end can word it
	// in whatever language it speaks. Its English is still the one the command
	// line prints, and both are checked here.
	missingArt, isMissingArt := after.Problems[0].(*MissingArtProblem)
	if !isMissingArt {
		t.Fatalf("the problem is a %T, want a *MissingArtProblem", after.Problems[0])
	}
	if !strings.HasSuffix(missingArt.Path, "sprout.svg") {
		t.Errorf("the problem names %q, want the missing file", missingArt.Path)
	}
	if !strings.Contains(missingArt.Error(), "sprout.svg") {
		t.Errorf("the problem reads %q, want it to name the missing file", missingArt)
	}
	missing, present := 0, 0
	for _, row := range after.Rows {
		missing += row.ArtMissing()
		present += len(row.Art) - row.ArtMissing()
	}
	pictures := 0
	for _, row := range before.Rows {
		pictures += len(row.Art)
	}
	if missing != 1 || present != pictures-1 {
		t.Errorf("%d pictures are missing and %d are not, want exactly one missing of %d",
			missing, present, pictures)
	}
	// The report is still a full report: taking away a file must not stop the
	// budget being tabulated for everyone else.
	if len(after.Rows) != len(before.Rows) {
		t.Errorf("the report covers %d characters, want %d", len(after.Rows), len(before.Rows))
	}
}

// TestInspectNoticesArtOnlyAGrownFormUses is the reason the report holds a list
// of pictures rather than one.
//
// Art a late stage uses is art nobody looks at until a character has grown, so a
// missing file there is precisely the one that surfaces in front of a player
// rather than in front of a check. The character's own picture is untouched
// here: a report that only asked about that would call this data fine.
func TestInspectNoticesArtOnlyAGrownFormUses(t *testing.T) {
	dir := scratchData(t)
	before, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect the copy: %v", err)
	}
	if !before.OK() {
		t.Fatalf("the copied data already has problems: %v", before.Problems)
	}
	// The bench's grown form owns this one, and nothing else names it.
	grown := filepath.Join(dir, "assets", "fixture", "bloom.svg")
	if err := os.Remove(grown); err != nil {
		t.Fatalf("remove %s: %v", grown, err)
	}
	after, err := Inspect(dir)
	if err != nil {
		t.Fatalf("inspect after removing the art: %v", err)
	}
	if after.OK() {
		t.Fatal("art that only a grown form uses went unnoticed")
	}
	if len(after.Problems) != 1 {
		t.Fatalf("%d problems reported, want 1: %v", len(after.Problems), after.Problems)
	}
	missing, isMissingArt := after.Problems[0].(*MissingArtProblem)
	if !isMissingArt {
		t.Fatalf("the problem is a %T, want a *MissingArtProblem", after.Problems[0])
	}
	// Which form is the half a character-level message cannot give, and it is
	// what tells an author where to look in cast.json.
	if missing.Stage != "Bloom" {
		t.Errorf("the problem names stage %q, want the grown form", missing.Stage)
	}
	if !strings.Contains(missing.Error(), "Bloom") {
		t.Errorf("the problem reads %q, want it to name the stage", missing)
	}
	// And the character's own picture is still fine, which is the point: one
	// row of the list is missing and the rest are not.
	for _, row := range after.Rows {
		if row.Art[0].Stage != "" {
			t.Errorf("%s lists %q first, want the character's own picture", row.ID, row.Art[0].Stage)
		}
		if !row.Art[0].Exists {
			t.Errorf("%s lost its own picture too, so this proves less than it looks", row.ID)
		}
	}
}

// TestWrittenCastIsStableAndReloads is what makes the tool safe to run twice:
// the bytes are a function of the content, and the content survives the trip
// through the file.
func TestWrittenCastIsStableAndReloads(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	first, err := lib.Characters().Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// The shipped file is already in this form, so writing it back is a no-op
	// on disk. That is the property that keeps a `new` diff to one block.
	onDisk, err := os.ReadFile(filepath.Join(dir, castFile))
	if err != nil {
		t.Fatalf("read the shipped cast: %v", err)
	}
	if string(onDisk) != string(first) {
		t.Error("the shipped cast.json is not in the form the tool writes, so the first write will churn the whole file")
	}

	character, err := Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/tester.png", Element: "wind/ground",
		Bio: "Written by a test.",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve a draft: %v", err)
	}
	if err := lib.SaveCharacter(character); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("reload after writing: %v", err)
	}
	returned, known := reloaded.Characters().Get(character.ID)
	if !known {
		t.Fatal("the written character is not in the reloaded book")
	}
	if !reflect.DeepEqual(returned, character) {
		t.Errorf("the trip through the file changed the character:\n%+v\n%+v", returned, character)
	}
	// Writing the reloaded book back has to produce the same bytes, or every
	// run of the tool would rewrite the file for no reason.
	againRaw, err := reloaded.Characters().Marshal()
	if err != nil {
		t.Fatalf("marshal the reloaded book: %v", err)
	}
	writtenRaw, err := os.ReadFile(filepath.Join(dir, castFile))
	if err != nil {
		t.Fatalf("read the written cast: %v", err)
	}
	if string(againRaw) != string(writtenRaw) {
		t.Error("marshalling the reloaded book does not reproduce the file it was read from")
	}
	// A saved character is in the library the caller still holds, so a second
	// save in one session sees the first.
	if _, known := lib.Characters().Get(character.ID); !known {
		t.Error("the library the character was saved through does not hold it")
	}
}

func TestResolveTakesTheArchetypeCurveAndKit(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	preset, known := lib.Archetypes().Get("duelist")
	if !known {
		t.Fatal("the duelist preset is not shipped")
	}
	character, err := Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/tester.svg", Element: "wind/ground",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(character.Stages) != 1 {
		t.Fatalf("the wizard produced %d stages, want 1", len(character.Stages))
	}
	if character.Stages[0].MinLevel != 1 {
		t.Errorf("the only stage starts at level %d, want 1", character.Stages[0].MinLevel)
	}
	if character.Stages[0].Name != "Tester" {
		t.Errorf("the only stage is named %q, want the character's name", character.Stages[0].Name)
	}
	if character.Stages[0].Stats != preset.Stats {
		t.Error("the stat curve is not the archetype's")
	}
	if !reflect.DeepEqual(cast.LearnedIDs(character.Skills), preset.Skills) {
		t.Errorf("the kit is %v, want the archetype's %v", character.Skills, preset.Skills)
	}

	// An override replaces exactly one curve and leaves the rest alone.
	overridden := Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/tester.svg", Element: "wind/ground",
		Skills: "strike, bolt",
	}
	overridden.Stats[progression.Speed] = "40:120"
	tuned, err := overridden.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve with an override: %v", err)
	}
	want := progression.Curve{Base: 40, Max: 120}
	if got := tuned.Stages[0].Stats[progression.Speed]; got != want {
		t.Errorf("the speed curve is %+v, want %+v", got, want)
	}
	if got := tuned.Stages[0].Stats[progression.HP]; got != preset.Stats[progression.HP] {
		t.Errorf("overriding speed also changed health to %+v", got)
	}
	if !reflect.DeepEqual(cast.LearnedIDs(tuned.Skills), []string{"strike", "bolt"}) {
		t.Errorf("the kit is %v, want the two skills that were named", tuned.Skills)
	}

	// Draft.Table is what a half-finished form shows a budget for, so it has to
	// agree with the table Resolve writes.
	table, err := overridden.Table(lib)
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	if table != tuned.Stages[0].Stats {
		t.Error("the table a form would show differs from the one the write produces")
	}
}

func TestResolveRejections(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	good := func() Draft {
		return Draft{
			ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
			Archetype: "duelist", Image: "assets/fixture/tester.svg", Element: "wind/ground",
		}
	}
	if _, err := good().Resolve(lib); err != nil {
		t.Fatalf("the base draft should resolve: %v", err)
	}
	cases := []struct {
		name   string
		change func(*Draft)
		wantIn string
	}{
		{"no id", func(d *Draft) { d.ID = "" }, "character id is empty"},
		{"non-slug id", func(d *Draft) { d.ID = "Example.Tester" }, "lowercase letters"},
		{"an id already in the cast", func(d *Draft) { d.ID = "fixture-anime.adept" }, "already in the cast"},
		{"no name", func(d *Draft) { d.Name = "  " }, "display name"},
		{"unknown origin", func(d *Draft) { d.Origin = "nowhere" }, "unknown origin"},
		{"unknown archetype", func(d *Draft) { d.Archetype = "berserker" }, "unknown archetype"},
		{"bad image extension", func(d *Draft) { d.Image = "assets/a.gif" }, "want .svg or .png"},
		{"absolute image", func(d *Draft) { d.Image = "/assets/a.svg" }, "absolute path"},
		{"unknown element", func(d *Draft) { d.Element = "plasma" }, "unknown element"},
		{"three elements", func(d *Draft) { d.Element = "fire/wind/ice" }, "want one or two"},
		{"an element pair the chart refuses", func(d *Draft) { d.Element = "water/fire" }, "counter each other"},
		{"unknown skill", func(d *Draft) { d.Skills = "strike,meteor" }, "unknown skill"},
		{"a duplicated skill", func(d *Draft) { d.Skills = "strike,strike" }, "twice"},
		{"an unreadable curve", func(d *Draft) { d.Stats[progression.HP] = "780" }, "want base:max"},
		{"a curve with an unreadable base", func(d *Draft) { d.Stats[progression.HP] = "x:2600" }, "unreadable base"},
		{"a curve that shrinks with level", func(d *Draft) { d.Stats[progression.HP] = "2600:780" }, "may not shrink"},
		{"a curve starting at nothing", func(d *Draft) { d.Stats[progression.HP] = "0:2600" }, "positive value"},
		{"a curve over its ceiling", func(d *Draft) { d.Stats[progression.Speed] = "60:260" }, "over the ceiling"},
		{
			// Health and defence multiply, so the joint bound is the one an
			// author is most likely to walk into without noticing.
			name: "a pair of curves over the joint durability budget",
			change: func(d *Draft) {
				d.Stats[progression.HP] = "1440:4800"
				d.Stats[progression.Defense] = "240:800"
			},
			wantIn: "over the budget",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := good()
			test.change(&candidate)
			_, err := candidate.Resolve(lib)
			if err == nil {
				t.Fatalf("the draft resolved, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}

	// The same refusals are available before the write, which is what a prompt
	// and a form apply to one answer at a time. They must not be a second
	// opinion: a kit the write refuses has to be refused here too.
	if err := lib.ValidateKit("strike,strike"); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("a duplicated skill gave %v, want a rejection mentioning it is named twice", err)
	}
}

// TestResolveRefusesAKitTheAffinityCannotCarry is the exact reproduction the
// coordinator reported: sentinel's kit is water, so a fire character built from
// it wrote cleanly and was then refused by battle.New.
func TestResolveRefusesAKitTheAffinityCannotCarry(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = Draft{
		ID: "fixture-anime.mismatch", Name: "Mismatch", Origin: "fixture-anime",
		Archetype: "sentinel", Image: "assets/fixture/adept.svg", Element: "fire",
	}.Resolve(lib)
	if err == nil {
		t.Fatal("a fire character carrying the sentinel kit was accepted")
	}
	for _, want := range []string{"riptide", "water", "fire"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection is %q, want it to mention %q", err, want)
		}
	}

	// And the same answer is available before the write, which is what a live
	// carry check in a form reports. skill.CanCarry is the single declaration;
	// CheckCarry only brings it forward.
	preset, _ := lib.Archetypes().Get("sentinel")
	kit, err := lib.LookupKit(preset.Skills)
	if err != nil {
		t.Fatalf("look the sentinel kit up: %v", err)
	}
	if err := lib.ValidateElement("fire", kit); err == nil {
		t.Error("the early check accepted fire for a water kit")
	} else if !strings.Contains(err.Error(), "riptide") {
		t.Errorf("the early rejection is %q, want it to name riptide", err)
	}
	if err := lib.ValidateElement("water/ice", kit); err != nil {
		t.Errorf("the early check refused an affinity that can carry the kit: %v", err)
	}
}

func TestSuggestedImageFollowsTheID(t *testing.T) {
	cases := map[string]string{
		"fixture-anime.adept": "assets/fixture-anime/adept.svg",
		"loner":               "assets/loner.svg",
		"":                    "",
	}
	// The map is a set of independent cases; nothing here reaches an ordered
	// output.
	for id, want := range cases {
		if got := SuggestedImage(id); got != want {
			t.Errorf("SuggestedImage(%q) is %q, want %q", id, got, want)
		}
	}
}

// TestArtFilesListsTheImagesOnDisk is the question internal/core/cast cannot
// ask, in the shape a picker needs it: not "is this one path there" but "which
// paths are".
//
// The tree is walked rather than listed because art is filed by origin, so the
// shipped assets folder is already two levels deep. Everything else here is
// what must not be offered: a file that is not an image, and a name this
// filesystem accepts but cast.ValidateImagePath refuses — a chooser handing
// back a value the write would reject is worse than a text field, which at
// least admits the author typed it.
func TestArtFilesListsTheImagesOnDisk(t *testing.T) {
	dir := scratchData(t)
	shipped, err := ArtFiles(dir)
	if err != nil {
		t.Fatalf("list the shipped art: %v", err)
	}
	if len(shipped) == 0 {
		t.Fatal("no art was found in the shipped data, so nothing here is being exercised")
	}
	// The two placeholders CLAUDE.md says not to delete are the fixed point:
	// anything else under assets belongs to whoever is authoring a cast.
	for _, want := range []string{"assets/fixture/adept.svg", "assets/fixture/sprout.svg"} {
		if !slices.Contains(shipped, want) {
			t.Errorf("the shipped art %q was not listed, only %v", want, shipped)
		}
	}

	deeper := filepath.Join(dir, "assets", "borrowed", "kanto")
	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatalf("create %s: %v", deeper, err)
	}
	writeFile(t, filepath.Join(deeper, "starter.png"), "not really a png")
	writeFile(t, filepath.Join(dir, "assets", "notes.txt"), "not art at all")

	art, err := ArtFiles(dir)
	if err != nil {
		t.Fatalf("list the art: %v", err)
	}
	if !slices.Contains(art, "assets/borrowed/kanto/starter.png") {
		t.Errorf("the walk did not reach three folders down: %v", art)
	}
	for _, unwanted := range []string{"assets/notes.txt", "notes.txt"} {
		if slices.Contains(art, unwanted) {
			t.Errorf("%q was offered as art: %v", unwanted, art)
		}
	}
	if len(art) != len(shipped)+1 {
		t.Errorf("%d paths were listed over the shipped %d, want exactly the one image added: %v",
			len(art), len(shipped), art)
	}
	// Every entry is a path a character may really name, and the order is the
	// one it will be shown in: sorted, because directory order is not a promise
	// any operating system makes and this list reaches a screen.
	for _, image := range art {
		if err := cast.ValidateImagePath(image); err != nil {
			t.Errorf("%q was offered but would be refused: %v", image, err)
		}
	}
	if !slices.IsSorted(art) {
		t.Errorf("the art is listed out of order: %v", art)
	}
}

// TestArtFilesSkipsANameTheParserRefuses is the other half of that rule, split
// out because it cannot run everywhere.
//
// A backslash in a filename is legal on a unix filesystem and refused by
// cast.ValidateImagePath, which is exactly the case worth having: a path this
// machine can hold and cast.json cannot mean the same thing twice. Windows has
// no such filename to make, so there is nothing to assert there.
func TestArtFilesSkipsANameTheParserRefuses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a backslash is not a legal filename here")
	}
	dir := scratchData(t)
	before, err := ArtFiles(dir)
	if err != nil {
		t.Fatalf("list the art: %v", err)
	}
	writeFile(t, filepath.Join(dir, "assets", `back\slash.svg`), "<svg/>")
	after, err := ArtFiles(dir)
	if err != nil {
		t.Fatalf("list the art after adding a refused name: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Errorf("a name the parser refuses was offered anyway:\nbefore %v\nafter  %v", before, after)
	}
}

// TestArtFilesTreatsAMissingAssetsFolderAsEmpty is the case a front-end has to
// survive rather than refuse.
//
// A data directory with no art is not broken — nothing has been drawn for it
// yet — so this is an empty list and not an error. A front-end that took an
// error here would have no form to show, which is a worse answer than a field
// somebody can type into.
func TestArtFilesTreatsAMissingAssetsFolderAsEmpty(t *testing.T) {
	dir := scratchData(t)
	assets := filepath.Join(dir, "assets")
	if err := os.RemoveAll(assets); err != nil {
		t.Fatalf("take away %s: %v", assets, err)
	}
	art, err := ArtFiles(dir)
	if err != nil {
		t.Fatalf("a missing assets folder is not an error, but it gave one: %v", err)
	}
	if len(art) != 0 {
		t.Errorf("%d paths were listed with no assets folder: %v", len(art), art)
	}
	// The books still load, which is what makes the empty list a state a form
	// has to handle rather than a failure it can decline to draw: whether art
	// exists is a check's question, not a parser's.
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("the books do not load without an assets folder: %v", err)
	}
	fromLibrary, err := lib.ArtFiles()
	if err != nil || len(fromLibrary) != 0 {
		t.Errorf("the library listed %v, %v; want nothing and no error", fromLibrary, err)
	}
	if lib.AssetsPath() != assets {
		t.Errorf("the library looks for art in %q, want %q", lib.AssetsPath(), assets)
	}
}

// writeFile drops one file into place, creating no folder: every caller here has
// already made the one it goes in.
func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadRejectsAMissingDirectory(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nothing-here")); err == nil {
		t.Error("a directory with no data files loaded")
	}
	if _, err := Load(""); err == nil {
		t.Error("an empty data directory loaded")
	}
}

// TestReplaceFileLeavesTheOldOneOnFailure is the reason a write goes through a
// temporary file: a truncated data file is a data file that stops the game
// booting.
func TestReplaceFileLeavesTheOldOneOnFailure(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, castFile))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// ⚠️ A **missing** folder used to be the way to make this fail, and is not
	// any more: a write may now name a folder of its own — a battle log lands in
	// one — so the writer creates what is not there. The failure has to come
	// from a folder that cannot be made at all, which is one under a file.
	lib.dir = filepath.Join(dir, castFile, "under-a-file")
	if err := lib.replaceFile(castFile, []byte("{}")); err == nil {
		t.Fatal("writing into a missing directory succeeded")
	}
	after, err := os.ReadFile(filepath.Join(dir, castFile))
	if err != nil {
		t.Fatalf("read after the failed write: %v", err)
	}
	if string(after) != string(before) {
		t.Error("a failed write changed the file it was replacing")
	}
}

// TestSaveOriginWritesTheCatalogTheBookWouldWrite covers the other write. The
// catalog is committed in exactly the form Marshal produces, so a tool-written
// entry and a hand-written one have to be indistinguishable — otherwise the
// next addition rewrites the whole file instead of one block.
func TestSaveOriginWritesTheCatalogTheBookWouldWrite(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	added := cast.Origin{
		ID: "example-play", Title: "Example Play", Medium: cast.Series,
		Year: 1999, Note: "Written by a test.",
	}
	if err := lib.SaveOrigin(added); err != nil {
		t.Fatalf("save an origin: %v", err)
	}
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	returned, known := reloaded.Origins().Get(added.ID)
	if !known {
		t.Fatal("the written work is not in the reloaded catalog")
	}
	if returned != added {
		t.Errorf("the trip through the file changed the work:\n%+v\n%+v", returned, added)
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, originsFile))
	if err != nil {
		t.Fatalf("read the written catalog: %v", err)
	}
	again, err := reloaded.Origins().Marshal()
	if err != nil {
		t.Fatalf("marshal the reloaded catalog: %v", err)
	}
	if string(again) != string(onDisk) {
		t.Error("marshalling the reloaded catalog does not reproduce the file it was read from")
	}

	// A second attempt at the same id is refused in words that describe what
	// the author actually did, rather than the parser's "declared twice".
	if err := lib.SaveOrigin(added); err == nil {
		t.Fatal("the same work was added twice")
	} else if !strings.Contains(err.Error(), "already in the catalog") {
		t.Errorf("the refusal is %q", err)
	}
}

// TestBudgetIsTheProgressionArithmetic pins the number a front-end draws a bar
// from to the engine's own function, so a bar can never be drawn from a second
// calculation of the same thing.
func TestBudgetIsTheProgressionArithmetic(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	preset, known := lib.Archetypes().Get("sentinel")
	if !known {
		t.Fatal("the sentinel preset is not shipped")
	}
	values := preset.Stats.At(progression.LevelCap)
	budget := lib.Budget(values)
	if want := progression.EffectiveHP(values, lib.Rules()); budget.Effective != want {
		t.Errorf("the budget reports %d absorbed, want %d", budget.Effective, want)
	}
	if want := lib.Limits().MaxEffectiveHP; budget.Max != want {
		t.Errorf("the budget's ceiling is %d, want %d", budget.Max, want)
	}
	if budget.Headroom != budget.Max-budget.Effective {
		t.Errorf("headroom is %d, want %d", budget.Headroom, budget.Max-budget.Effective)
	}
	if budget.Over() {
		t.Error("a shipped preset is over the budget, which ParseArchetypes should have refused")
	}
}

// TestCheckSkillAnswersEachRestrictionAndSkipsWhatIsUnanswered is the predicate
// both front-ends use while a form is half-filled.
//
// The skills are built here rather than parsed from a fixture because what is
// being checked is the predicate, not the parser: skill.ParseBook has its own
// tests for the same restrictions, and a fixture would only add a second thing
// that could be wrong.
func TestCheckSkillAnswersEachRestrictionAndSkipsWhatIsUnanswered(t *testing.T) {
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	water, err := element.Single(element.Water)
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	keptForFire := skill.Skill{
		ID: "oath", Element: element.Neutral,
		Restrict: &skill.Restriction{Elements: []element.Element{element.Fire}},
	}
	keptForBulwark := skill.Skill{
		ID: "wall", Element: element.Neutral,
		Restrict: &skill.Restriction{Archetypes: []string{"bulwark"}},
	}
	keptForSprout := skill.Skill{
		ID: "bloom", Element: element.Neutral,
		Restrict: &skill.Restriction{Characters: []string{"example.sprout"}},
	}
	keptForDragons := skill.Skill{
		ID: "roar", Element: element.Neutral,
		Restrict: &skill.Restriction{Species: []string{"dragon"}},
	}
	keptForAWork := skill.Skill{
		ID: "rasengan", Element: element.Neutral,
		Restrict: &skill.Restriction{Origins: []string{"naruto"}},
	}
	unrestricted := skill.Skill{ID: "strike", Element: element.Neutral}

	answered := Carrier{
		ID: "example.adept", Archetype: "duelist", Affinity: water, HasAffinity: true,
		Species: []string{"lizard"}, Origin: "pokemon",
	}
	var carry *CarryError
	if err := CheckSkill(answered, keptForFire); !errors.As(err, &carry) {
		t.Fatalf("a skill kept for fire was allowed to a water unit: %v", err)
	}
	if carry.Reason != skill.CarryElementRestricted {
		t.Errorf("the refusal is reason %d, want the restricted one", carry.Reason)
	}
	if got := carry.Allowed; len(got) != 1 || got[0] != "fire" {
		t.Errorf("the refusal carries the allowlist %v", got)
	}
	var byArchetype *ArchetypeRestrictedError
	if err := CheckSkill(answered, keptForBulwark); !errors.As(err, &byArchetype) {
		t.Fatalf("a skill kept for one preset was allowed to another: %v", err)
	}
	if byArchetype.Archetype != "duelist" || byArchetype.Skill != "wall" {
		t.Errorf("the refusal reads %+v", byArchetype)
	}
	var byCharacter *CharacterRestrictedError
	if err := CheckSkill(answered, keptForSprout); !errors.As(err, &byCharacter) {
		t.Fatalf("a skill kept for one character was allowed to another: %v", err)
	}
	var byLineage *SpeciesRestrictedError
	if err := CheckSkill(answered, keptForDragons); !errors.As(err, &byLineage) {
		t.Fatalf("a skill kept for a lineage was allowed to something else: %v", err)
	}
	var byWork *OriginRestrictedError
	if err := CheckSkill(answered, keptForAWork); !errors.As(err, &byWork) {
		t.Fatalf("a skill kept for one work was allowed to a character out of another: %v", err)
	}
	if byWork.Character != "example.adept" || byWork.Skill != "rasengan" ||
		len(byWork.Allowed) != 1 || byWork.Allowed[0] != "naruto" {
		t.Errorf("the refusal reads %+v", byWork)
	}
	if err := CheckSkill(answered, unrestricted); err != nil {
		t.Errorf("an unrestricted skill was refused: %v", err)
	}

	// Nothing answered yet: the kit may be filled in first, and none of the
	// five lists has anything to judge against. An empty origin is "not asked"
	// rather than "out of nowhere" — the same reading an empty species list
	// gets, and the reason a half-filled form does not refuse a skill the
	// finished one will accept.
	empty := Carrier{}
	for _, carried := range []skill.Skill{
		keptForFire, keptForBulwark, keptForSprout, keptForDragons, keptForAWork,
	} {
		if err := CheckSkill(empty, carried); err != nil {
			t.Errorf("%q was refused before anything was answered: %v", carried.ID, err)
		}
	}

	// And the other order: the element settled first admits what it allows.
	onlyFire := Carrier{Affinity: fire, HasAffinity: true}
	if err := CheckSkill(onlyFire, keptForFire); err != nil {
		t.Errorf("a fire unit was refused a skill kept for fire: %v", err)
	}
	if err := CheckKit(answered, []skill.Skill{unrestricted, keptForFire}); err == nil {
		t.Error("a kit holding one refused skill was accepted")
	}
	if err := CheckKit(answered, []skill.Skill{unrestricted}); err != nil {
		t.Errorf("a kit of one unrestricted skill was refused: %v", err)
	}
}

// TestADraftsCarrierLeavesAnUnreadableElementOut is what makes the two fill
// orders work: an element that is not an element yet is an ordinary state on a
// half-typed form, not a refusal, so it restricts nothing until it parses.
func TestADraftsCarrierLeavesAnUnreadableElementOut(t *testing.T) {
	half := Draft{ID: " example.adept ", Archetype: "duelist", Element: "fi"}
	who := half.Carrier()
	if who.ID != "example.adept" || who.Archetype != "duelist" {
		t.Errorf("the carrier reads %+v", who)
	}
	if who.HasAffinity {
		t.Error("a half-typed element was taken as an answer")
	}
	settled := Draft{Element: "fire/metal"}.Carrier()
	if !settled.HasAffinity || settled.Affinity.String() != "fire/metal" {
		t.Errorf("a settled element resolved to %+v", settled)
	}
}

// TestWrittenSkillsAreStableAndReloads is what makes skills.json safe for the
// tool to write: the bytes are a function of the content, the content survives
// the trip through the file, and the shipped file is already in the form the
// tool writes — so the first `skills add` is a one-block diff rather than a
// rewrite of all three hundred lines nobody can review.
func TestWrittenSkillsAreStableAndReloads(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	first, err := lib.Skills().Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	if string(onDisk) != string(first) {
		t.Error("the shipped skills.json is not in the form the tool writes, so the first write will churn the whole file")
	}

	built, err := SkillDraft{
		ID: "oath", Element: "neutral", Target: "enemy", Range: "1", Pattern: "single",
		Power: "1200", Strikes: "1", Accuracy: "900", Cooldown: "2",
		Applies: "burn:500", RestrictElements: "fire,metal",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve a draft: %v", err)
	}
	if err := lib.SaveSkill(built); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("reload after writing: %v", err)
	}
	returned, err := reloaded.Skills().Lookup(built.ID)
	if err != nil {
		t.Fatalf("the written skill is not in the reloaded book: %v", err)
	}
	if !reflect.DeepEqual(returned, built) {
		t.Errorf("the trip through the file changed the skill:\n%+v\n%+v", returned, built)
	}
	againRaw, err := reloaded.Skills().Marshal()
	if err != nil {
		t.Fatalf("marshal the reloaded book: %v", err)
	}
	writtenRaw, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the written skills: %v", err)
	}
	if string(againRaw) != string(writtenRaw) {
		t.Error("marshalling the reloaded book does not reproduce the file it was read from")
	}
	// A saved skill is in the library the caller still holds, so a character
	// authored in the same session can take it.
	if _, err := lib.Skills().Lookup(built.ID); err != nil {
		t.Errorf("the library the skill was saved through does not hold it: %v", err)
	}
	// And the write really did append rather than sort: the skill book's order
	// is authored, and skills.golden's table is that order.
	skills := reloaded.Skills().Skills()
	if last := skills[len(skills)-1]; last.ID != built.ID {
		t.Errorf("the new skill landed at %q rather than the end of the book", last.ID)
	}

	// The one id already in the book is refused in this package's words rather
	// than the parser's, which describes a file listing one id twice.
	var taken *SkillTakenError
	if err := lib.SaveSkill(built); !errors.As(err, &taken) {
		t.Errorf("saving a skill twice gave %v", err)
	}
}

// TestStageSummaryDrawsAForkAsAFork is the summary line's half of a line that
// forks, and it exists because the honest-looking answer is the wrong one.
//
// Joining every stage with an arrow is what the line always did and it is right
// for every shipped character. On a fork it reads as three forms in a row when
// the last two are alternatives — a table quietly saying something the file does
// not, which is the failure this whole feature is written against.
func TestStageSummaryDrawsAForkAsAFork(t *testing.T) {
	stage := func(name string, level int, after string) progression.Stage {
		return progression.Stage{Name: name, MinLevel: level, After: after}
	}
	for _, testCase := range []struct {
		name string
		line progression.Line
		want string
	}{
		{"a line that does not fork reads as it always did", progression.Line{
			stage("Bulbasaur", 1, ""), stage("Ivysaur", 16, ""), stage("Venusaur", 32, ""),
		}, "Bulbasaur@1 → Ivysaur@16 → Venusaur@32"},
		{"two arms share a bracket", progression.Line{
			stage("Eevee", 1, ""), stage("Vaporeon", 32, "Eevee"), stage("Jolteon", 32, "Eevee"),
		}, "Eevee@1 → (Vaporeon@32 | Jolteon@32)"},
		{"a stage past the fork stays on its own arm", progression.Line{
			stage("Eevee", 1, ""), stage("Vaporeon", 32, "Eevee"),
			stage("Jolteon", 32, "Eevee"), stage("Tempest", 48, "Jolteon"),
		}, "Eevee@1 → (Vaporeon@32 | Jolteon@32 → Tempest@48)"},
		{"an explicit line that happens not to fork keeps its arrows", progression.Line{
			stage("Eevee", 1, ""), stage("Vaporeon", 32, "Eevee"),
		}, "Eevee@1 → Vaporeon@32"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := StageSummary(cast.Character{Stages: testCase.line})
			if got != testCase.want {
				t.Errorf("the line reads %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestPreviewDamageIsTheEngineArithmetic pins the reference pair, because it is
// the whole reason the figure is worth showing: it has to be the same pair
// skills.golden's damage column is measured from, or an author reads two numbers
// for one skill and cannot tell which the design was made from.
func TestPreviewDamageIsTheEngineArithmetic(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	strike, err := lib.Skills().Lookup("strike")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	preview := lib.PreviewDamage(strike)
	if got, want := preview.Attack, lib.Limits().Ceilings[progression.Attack]; got != want {
		t.Errorf("the reference attack is %d, want the attack ceiling %d", got, want)
	}
	if got, want := preview.Defense, lib.Limits().Ceilings[progression.Defense]/2; got != want {
		t.Errorf("the reference defence is %d, want half the defence ceiling %d", got, want)
	}
	// The figures skills.golden's own table holds for these two rows.
	if preview.PerStrike != 342 || preview.Total != 342 {
		t.Errorf("strike previews as %d per strike and %d in all, want 342 and 342",
			preview.PerStrike, preview.Total)
	}
	// A multi-strike skill truncates once per strike, as a battle does, rather
	// than once over the total: three strikes of 600 are 615 and not 617.
	flurry, err := lib.Skills().Lookup("flurry")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := lib.PreviewDamage(flurry); got.PerStrike != 205 || got.Total != 615 {
		t.Errorf("flurry previews as %d x%d = %d, want 205 x3 = 615",
			got.PerStrike, got.Strikes, got.Total)
	}
	// A conditional skill reports both figures, because the form can edit the
	// power of a skill that has a condition even though it cannot author one.
	detonate, err := lib.Skills().Lookup("detonate")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := lib.PreviewDamage(detonate); got.Total != 308 || got.Amplified != 1062 {
		t.Errorf("detonate previews as %d and %d amplified, want 308 and 1062",
			got.Total, got.Amplified)
	}
	if lib.PreviewDamage(strike).Amplified != 0 {
		t.Error("a skill with no condition reports an amplified figure")
	}
}

// TestPreviewDamageReadsTheCastersOwnTerms is the caster half of the figure, and
// it exists because the preview did not have one.
//
// Three terms can raise a skill's power and only one of them reads the target.
// `self_requires` is a threshold on the caster and `self_gradient` is a curve off
// its wounds, so a preview that read `requires` alone showed `outrage` and
// `comeback` at their plain power — the two skills whose whole design is the term
// it was not reading — and an author editing either had no figure for it at all.
//
// The composition is combat.Swung, the expression the battle resolves through:
// the bonus is added to the power and the share is taken of the sum, so a skill
// declaring both is worth more than a reading that took the share of the declared
// power. The all-three case below is chosen so the two orders disagree, because a
// case where they agree would pass whichever way round the arithmetic went.
func TestPreviewDamageReadsTheCastersOwnTerms(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	damage := func(power int) int64 {
		reference := lib.Limits().Ceilings[progression.Defense] / 2
		return lib.Rules().Damage(lib.Limits().Ceilings[progression.Attack], reference, power, 1000)
	}

	// A gradient: worth its whole share at the bottom of the caster's health, so
	// a thousand per mille on a power of a thousand previews as two thousand.
	desperate, err := lib.Skills().Lookup("desperate")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got, want := lib.PreviewDamage(desperate).Amplified, damage(2000); got != want {
		t.Errorf("desperate previews %d with its gradient at the bottom, want %d", got, want)
	}

	// A threshold on the caster: a plain addition to the power, exactly as the
	// target's own condition is.
	reckless := desperate
	reckless.ID = "reckless_swing"
	reckless.SelfGradient = nil
	reckless.SelfRequires = &skill.Condition{BelowHealth: 400, BonusPower: 1200}
	if got, want := lib.PreviewDamage(reckless).Amplified, damage(2200); got != want {
		t.Errorf("a caster threshold previews %d, want %d", got, want)
	}

	// All three at once, and the figure has to be the engine's composition of
	// them rather than any other order. 1000 + 900 amplified by the target's
	// condition, + 500 for the caster's threshold, all of it doubled by a
	// gradient at the bottom: 4800. Taking the share of the declared power first
	// would give 3400, which is what this catches.
	everything := reckless
	everything.ID = "everything"
	everything.Requires = &skill.Condition{Status: "burn", MinStacks: 1, BonusPower: 900}
	everything.SelfRequires = &skill.Condition{BelowHealth: 400, BonusPower: 500}
	everything.SelfGradient = &skill.Gradient{AtEmpty: 1000}
	if got, want := lib.PreviewDamage(everything).Amplified, damage(4800); got != want {
		t.Errorf("a skill declaring all three previews %d, want %d", got, want)
	}

	// And a skill declaring nothing still reports nothing, so the figure keeps
	// meaning "this skill has a ceiling worth naming".
	plain := desperate
	plain.SelfGradient = nil
	if got := lib.PreviewDamage(plain).Amplified; got != 0 {
		t.Errorf("a skill with no term at all reports %d amplified", got)
	}
}

// TestAPiercingSkillIsAuthoredAndPreviewsAgainstTheArmourItLeaves is the
// authoring end of piercing: the answer reaches the skill, and the damage figure
// the form shows is measured against the defence the skill actually faces.
//
// The preview naming the pierced defence rather than the defender's is what
// keeps the line able to explain itself. An author who divides by the figure the
// row names has to arrive at the damage the row names beside it; showing the raw
// 400 next to damage computed against 160 would be a row that contradicts its
// own arithmetic.
func TestAPiercingSkillIsAuthoredAndPreviewsAgainstTheArmourItLeaves(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	answers := SkillDraft{
		ID: "sunder", Element: "metal", Target: "enemy", Range: "1", Pattern: "single",
		Power: "1000", Strikes: "1", Accuracy: "900", Cooldown: "2", Pierce: "600",
	}
	built, err := answers.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve a piercing draft: %v", err)
	}
	if built.Pierce != 600 {
		t.Fatalf("the draft answered 600 and the skill pierces %d", built.Pierce)
	}

	reference := lib.Limits().Ceilings[progression.Defense] / 2
	preview := lib.PreviewDamage(built)
	if want := combat.Pierced(reference, 600); preview.Defense != want {
		t.Errorf("the preview measures against %d defence, want the %d piercing leaves of %d",
			preview.Defense, want, reference)
	}
	if want := lib.Rules().Damage(preview.Attack, preview.Defense, built.Power, 1000); preview.PerStrike != want {
		t.Errorf("the preview says %d per strike but its own reference pair gives %d",
			preview.PerStrike, want)
	}
	// A skill that pierces nothing still reads against the plain reference, so
	// every skill authored before this existed previews as it always did.
	plain := built
	plain.Pierce = 0
	if got := lib.PreviewDamage(plain); got.Defense != reference {
		t.Errorf("an unpierced skill previews against %d defence, want the reference %d",
			got.Defense, reference)
	}

	// And the answer survives a save, a reload and being read back into the form
	// — which is the trip an edit takes.
	if err := lib.SaveSkill(built); err != nil {
		t.Fatalf("save: %v", err)
	}
	reloaded, err := Load(lib.Dir())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	returned, err := reloaded.Skills().Lookup(built.ID)
	if err != nil {
		t.Fatalf("the written skill is not in the reloaded book: %v", err)
	}
	if returned.Pierce != 600 {
		t.Errorf("the reloaded skill pierces %d, want 600", returned.Pierce)
	}
	if got := SkillAnswers(returned).Pierce; got != "600" {
		t.Errorf("the form reads the pierce back as %q, want \"600\"", got)
	}
	// Absent rather than a nought on a skill that does not pierce, so accepting
	// the form as it stands writes the file that was read.
	if got := SkillAnswers(plain).Pierce; got != "" {
		t.Errorf("an unpierced skill reads back as %q, want an empty answer", got)
	}
}

// TestAuthoringAgainstARestrictedSkillRefusesTheKitAndTheCharacter is the two
// halves of part one meeting the tool that writes them: a skill is authored with
// an allowlist, and the character form then refuses a kit holding it — first at
// the answer, by the same predicate the picker marks a row with, and then at the
// write, by cast.ParseBook.
func TestAuthoringAgainstARestrictedSkillRefusesTheKitAndTheCharacter(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	kept, err := SkillDraft{
		ID: "bulwark_oath", Element: "neutral", Target: "enemy", Range: "1",
		Pattern: "single", Power: "1000", Strikes: "1", Accuracy: "900", Cooldown: "0",
		RestrictArchetypes: "bulwark",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve a restricted skill: %v", err)
	}
	if err := lib.SaveSkill(kept); err != nil {
		t.Fatalf("save: %v", err)
	}

	// The answer's check, which is what a prompt applies as it is typed and what
	// the picker marks a row with.
	var byArchetype *ArchetypeRestrictedError
	refused := lib.ValidateKitFor("strike,bulwark_oath",
		Carrier{ID: "fixture-film.tester", Archetype: "duelist"})
	if !errors.As(refused, &byArchetype) {
		t.Fatalf("a duelist was allowed a skill kept for bulwark: %v", refused)
	}
	if byArchetype.Skill != "bulwark_oath" {
		t.Errorf("the refusal names %q", byArchetype.Skill)
	}
	if err := lib.ValidateKitFor("strike,bulwark_oath",
		Carrier{ID: "fixture-film.tester", Archetype: "bulwark"}); err != nil {
		t.Errorf("a bulwark was refused a skill kept for bulwark: %v", err)
	}

	// And the write's check, which is the parser's and is what actually holds:
	// even with the answer check skipped, the character cannot be written.
	_, err = Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/adept.svg", Element: "neutral",
		Skills: "strike,bulwark_oath",
	}.Resolve(lib)
	if err == nil {
		t.Fatal("a duelist carrying a skill kept for bulwark was written")
	}
	for _, want := range []string{"bulwark_oath", "bulwark"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the write's refusal %q does not mention %q", err, want)
		}
	}
}

// TestTheAllowlistValidatorsAcceptNothingAndRefuseTheUnknown covers the two
// checks the command line applies to `--restrict-archetypes` and
// `--restrict-characters`.
//
// The empty answer is the case that matters most, and it is the one an
// implementation gets wrong: an unanswered list is *unrestricted*, and it must
// not become the present-but-empty list skill.ParseBook refuses. Those are
// opposite meanings — one skill everybody may carry against one skill nobody
// may — and only one of them is what an author who pressed Enter meant.
func TestTheAllowlistValidatorsAcceptNothingAndRefuseTheUnknown(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, answer := range []string{"", "   ", ","} {
		if err := lib.ValidateRestrictedArchetypes(answer); err != nil {
			t.Errorf("the empty answer %q was refused for roles: %v", answer, err)
		}
		if err := lib.ValidateRestrictedCharacters(answer); err != nil {
			t.Errorf("the empty answer %q was refused for characters: %v", answer, err)
		}
	}
	// And an empty answer really does mean unrestricted rather than an empty
	// block, which the parser would refuse.
	restriction, err := SkillDraft{}.Restriction()
	if err != nil {
		t.Fatalf("an empty draft's restriction: %v", err)
	}
	if restriction != nil {
		t.Errorf("an unanswered restriction resolved to %+v, want none at all", restriction)
	}

	presets := lib.Archetypes().IDs()
	if len(presets) < 2 {
		t.Fatalf("the shipped data holds %d presets, and this needs two", len(presets))
	}
	if err := lib.ValidateRestrictedArchetypes(presets[0] + "," + presets[1]); err != nil {
		t.Errorf("two real presets were refused: %v", err)
	}
	var unknownArchetype *UnknownArchetypeError
	if err := lib.ValidateRestrictedArchetypes(presets[0] + ",nowhere"); !errors.As(err, &unknownArchetype) {
		t.Errorf("an unknown preset gave %v", err)
	}
	var twice *DuplicateEntryError
	if err := lib.ValidateRestrictedArchetypes(presets[0] + "," + presets[0]); !errors.As(err, &twice) {
		t.Errorf("a preset named twice gave %v", err)
	}

	cast := lib.CharacterIDs()
	if len(cast) == 0 {
		t.Fatal("the shipped data holds no characters, and this needs one")
	}
	if err := lib.ValidateRestrictedCharacters(cast[0]); err != nil {
		t.Errorf("a real character was refused: %v", err)
	}
	var unknownCharacter *UnknownCharacterError
	if err := lib.ValidateRestrictedCharacters("nobody.here"); !errors.As(err, &unknownCharacter) {
		t.Errorf("an unknown character gave %v", err)
	}
	if err := lib.ValidateRestrictedCharacters(cast[0] + "," + cast[0]); !errors.As(err, &twice) {
		t.Errorf("a character named twice gave %v", err)
	}
}

// The tests below are editing a skill that already ships, which is a different
// job from authoring one for the reason stated at the top of skill_edit.go:
// nobody carries a new skill, and shipped units carry an edited one.

// TestAnEditKeepsTheBlocksTheFormNeverAsksAbout is the losslessness that makes
// editing through the tool safe at all.
//
// The form authors nine fields, the statuses inflicted and the three allowlists.
// It never asks about requires, strips, scaling or self_applies, and all four
// exist in the shipped book — so an edit built onto an empty skill rather than
// onto the skill as it stands would silently delete a hand-authored condition, in
// a file the author was told they were only changing a power in.
//
// The four are asserted on the real data rather than on a fixture, for the same
// reason TestTheShippedSkillBookSurvivesBeingWritten is: a fixture proves the
// mechanism and the shipped file proves the game.
func TestAnEditKeepsTheBlocksTheFormNeverAsksAbout(t *testing.T) {
	cases := []struct {
		skill string
		holds func(skill.Skill) bool
		what  string
	}{
		{"detonate", func(s skill.Skill) bool { return s.Requires != nil }, "requires"},
		{"purify", func(s skill.Skill) bool { return s.Strips != nil }, "strips"},
		{"swift_edge", func(s skill.Skill) bool {
			return s.Scaling.Source == combat.BaseStat && s.Scaling.Stat == progression.Speed
		}, "scaling"},
		{"quickstep", func(s skill.Skill) bool { return len(s.SelfApplies) > 0 }, "self_applies"},
	}
	for _, test := range cases {
		t.Run(test.what, func(t *testing.T) {
			dir := scratchData(t)
			lib, err := Load(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			before, err := lib.Skills().Lookup(test.skill)
			if err != nil {
				t.Fatalf("look up %s: %v", test.skill, err)
			}
			if !test.holds(before) {
				t.Fatalf("the shipped %s no longer declares %s, so this proves nothing",
					test.skill, test.what)
			}

			// An edit that mentions the accuracy and nothing else.
			raised := "700"
			built, err := (SkillEdit{Accuracy: &raised}).Draft(before).ResolveEdit(lib, test.skill)
			if err != nil {
				t.Fatalf("resolve the edit: %v", err)
			}
			change, err := lib.EditSkill(built)
			if err != nil {
				t.Fatalf("edit: %v", err)
			}
			if change.After.Accuracy != 700 {
				t.Errorf("the accuracy became %d", change.After.Accuracy)
			}
			if !test.holds(change.After) {
				t.Errorf("the edit dropped %s: %+v", test.what, change.After)
			}

			// And the file, not only the value in hand: the block has to come back
			// off the disk.
			reloaded, err := Load(dir)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			written, err := reloaded.Skills().Lookup(test.skill)
			if err != nil {
				t.Fatalf("the edited skill is not in the reloaded book: %v", err)
			}
			if !reflect.DeepEqual(written, change.After) {
				t.Errorf("the trip through the file changed the skill:\n%+v\n%+v",
					written, change.After)
			}
			// Everything the edit did not name is what it was, which is the whole
			// claim: the accuracy is the only field that moved.
			expected := before
			expected.Accuracy = 700
			if !reflect.DeepEqual(written, expected) {
				t.Errorf("the edit changed more than the accuracy:\n%+v\n%+v",
					written, expected)
			}
		})
	}
}

// TestAnEditKeepsTheSkillInItsPlaceInTheFile is skill.Book.Replace's promise
// measured where it matters: on the shipped file, which is committed in the form
// Marshal writes precisely so an edit is a small diff.
func TestAnEditKeepsTheSkillInItsPlaceInTheFile(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	order := func(book *skill.Book) []string {
		out := make([]string, 0, len(book.Skills()))
		for _, current := range book.Skills() {
			out = append(out, current.ID)
		}
		return out
	}
	was := order(lib.Skills())

	// venom_fang is the fourth of nineteen, so a write that appended or sorted
	// would move it.
	current, err := lib.Skills().Lookup("venom_fang")
	if err != nil {
		t.Fatalf("look up venom_fang: %v", err)
	}
	power := "1300"
	built, err := (SkillEdit{Power: &power}).Draft(current).ResolveEdit(lib, "venom_fang")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := lib.EditSkill(built); err != nil {
		t.Fatalf("edit: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the written skills: %v", err)
	}
	if !slices.Equal(order(lib.Skills()), was) {
		t.Errorf("the declaration order became %v, want %v", order(lib.Skills()), was)
	}
	// The whole file rather than the entry: an assertion about the entry alone
	// would pass on a write that moved it to the end.
	oldLines, newLines := strings.Split(string(before), "\n"), strings.Split(string(after), "\n")
	if len(oldLines) != len(newLines) {
		t.Fatalf("the file went from %d lines to %d", len(oldLines), len(newLines))
	}
	moved := []string(nil)
	for i := range oldLines {
		if oldLines[i] != newLines[i] {
			moved = append(moved, strings.TrimSpace(newLines[i]))
		}
	}
	if len(moved) != 1 || moved[0] != `"power": 1300,` {
		t.Errorf("the write changed %v, want one power line", moved)
	}
}

// TestAnEditThatWouldOrphanAShippedCharacterIsRefused is the first of the two
// ways editing can break something an addition cannot.
//
// The case is built from the shipped data rather than a fixture, and it is the
// real one: fixture-anime.adept is water/ice and carries riptide, so keeping
// riptide for fire leaves the character it already ships in unable to carry it.
// What must not happen is a written file that then fails to load — so the refusal
// comes before the write, and it names who.
func TestAnEditThatWouldOrphanAShippedCharacterIsRefused(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the shipped skills: %v", err)
	}
	current, err := lib.Skills().Lookup("riptide")
	if err != nil {
		t.Fatalf("look up riptide: %v", err)
	}
	carrier, holds := lib.Characters().Get("fixture-anime.adept")
	if !holds || !slices.Contains(cast.LearnedIDs(carrier.Skills), "riptide") {
		t.Skip("the shipped cast no longer has a character carrying riptide")
	}

	fire := "fire"
	built, err := (SkillEdit{RestrictElements: &fire}).Draft(current).ResolveEdit(lib, "riptide")
	if err != nil {
		t.Fatalf("the skill itself is legal, so resolving it should pass: %v", err)
	}
	_, err = lib.EditSkill(built)
	var broken *SkillEditBreaksError
	if !errors.As(err, &broken) {
		t.Fatalf("keeping a carried skill for another element was accepted: %v", err)
	}
	if broken.ID != carrier.ID {
		t.Errorf("the refusal names %q, want %q", broken.ID, carrier.ID)
	}
	if broken.Carrier != BrokenCharacter {
		t.Errorf("the refusal is about %v, want a character", broken.Carrier)
	}
	if broken.Skill != "riptide" {
		t.Errorf("the refusal names the skill %q", broken.Skill)
	}
	// And why, from the same value skill.WhyCannotCarry hands the engine.
	var refused *CarryError
	if !errors.As(err, &refused) {
		t.Fatalf("the reason is %T, want a carry refusal", broken.Err)
	}
	if refused.Reason != skill.CarryElementRestricted {
		t.Errorf("the reason is %v, want the element allowlist", refused.Reason)
	}

	// Nothing was written, which is the point: a refused edit must not leave a
	// data directory the game no longer boots from.
	after, err := os.ReadFile(filepath.Join(dir, skillsFile))
	if err != nil {
		t.Fatalf("read the skills after the refusal: %v", err)
	}
	if string(after) != string(before) {
		t.Error("a refused edit still rewrote skills.json")
	}
	if held, err := lib.Skills().Lookup("riptide"); err != nil || held.Restrict != nil {
		t.Errorf("the library kept the refused edit: %+v", held)
	}
	if _, err := Load(dir); err != nil {
		t.Errorf("the data directory no longer loads: %v", err)
	}
}

// TestAnEditNarrowingASkillToAnotherWorkNamesItsCarrier is the same refusal on
// the axis a character carries rather than declares, and it is here because the
// walk that names the carrier was not looking at it.
//
// brokenPreset and brokenCharacter classify a refusal the re-parse has already
// made, so a walk that cannot see an axis does not let a bad edit through — it
// just blames nobody for it. The character walk was building a Carrier out of an
// id, a preset and an element, so a species or an origin refusal came back with
// the parser's words and an empty ID field. Both are handed over now, and this
// test is on the origin because that is the axis the walk was missing on the day
// it arrived.
func TestAnEditNarrowingASkillToAnotherWorkNamesItsCarrier(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	carrier, holds := lib.Characters().Get("fixture-anime.adept")
	if !holds || !slices.Contains(cast.LearnedIDs(carrier.Skills), "riptide") {
		t.Skip("the fixture cast no longer has a character carrying riptide")
	}
	current, err := lib.Skills().Lookup("riptide")
	if err != nil {
		t.Fatalf("look up riptide: %v", err)
	}
	elsewhere := "fixture-film"
	built, err := (SkillEdit{RestrictOrigins: &elsewhere}).Draft(current).ResolveEdit(lib, "riptide")
	if err != nil {
		t.Fatalf("the skill itself is legal, so resolving it should pass: %v", err)
	}
	_, err = lib.EditSkill(built)
	var broken *SkillEditBreaksError
	if !errors.As(err, &broken) {
		t.Fatalf("keeping a carried skill for another work was accepted: %v", err)
	}
	if broken.ID != carrier.ID || broken.Carrier != BrokenCharacter {
		t.Errorf("the refusal is about %v %q, want the character %q",
			broken.Carrier, broken.ID, carrier.ID)
	}
	var refused *OriginRestrictedError
	if !errors.As(err, &refused) {
		t.Fatalf("the reason is %T, want an origin refusal", broken.Err)
	}
	if len(refused.Allowed) != 1 || refused.Allowed[0] != elsewhere {
		t.Errorf("the refusal carries the allowlist %v", refused.Allowed)
	}
}

// TestAnEditThatWouldOrphanAnArchetypePresetIsRefused is the second way, and it
// is a different rule in a different place: a preset is the starting point for
// every character built from it, so a skill kept for named characters has no
// business in one — cast.resolveArchetype refuses it, and forge.CheckPresetKit
// brings that answer forward so the refusal can name the preset.
//
// The case is shipped data again: sever sits in bulwark's kit and nobody carries
// it, so keeping it for a named character breaks the preset and only the preset.
func TestAnEditThatWouldOrphanAnArchetypePresetIsRefused(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	current, err := lib.Skills().Lookup("sever")
	if err != nil {
		t.Fatalf("look up sever: %v", err)
	}
	preset, holds := lib.Archetypes().Get("bulwark")
	if !holds || !slices.Contains(preset.Skills, "sever") {
		t.Skip("the shipped presets no longer have bulwark carrying sever")
	}
	owner := lib.CharacterIDs()[0]

	built, err := (SkillEdit{RestrictCharacters: &owner}).Draft(current).ResolveEdit(lib, "sever")
	if err != nil {
		t.Fatalf("the skill itself is legal, so resolving it should pass: %v", err)
	}
	_, err = lib.EditSkill(built)
	var broken *SkillEditBreaksError
	if !errors.As(err, &broken) {
		t.Fatalf("giving a preset's skill to one character was accepted: %v", err)
	}
	if broken.Carrier != BrokenPreset || broken.ID != "bulwark" {
		t.Errorf("the refusal is about %v %q, want the bulwark preset", broken.Carrier, broken.ID)
	}
	var owned *PresetOwnedSkillError
	if !errors.As(err, &owned) {
		t.Fatalf("the reason is %T, want a preset-owned refusal", broken.Err)
	}
	if owned.Archetype != "bulwark" || owned.Skill != "sever" {
		t.Errorf("the reason names %q and %q", owned.Archetype, owned.Skill)
	}
	if _, err := Load(dir); err != nil {
		t.Errorf("the data directory no longer loads: %v", err)
	}
}

// TestAnAbsentFieldAndAnExplicitZeroAreDifferentAnswers is the trap a partial
// edit exists to close.
//
// "--cooldown 0" means make the cooldown zero and no --cooldown at all means
// leave it, and a field held as a plain string cannot tell the two apart. So the
// fields are pointers, and both halves are asserted: the zero has to land, and
// the silence has to leave everything alone.
func TestAnAbsentFieldAndAnExplicitZeroAreDifferentAnswers(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// riptide ships with a cooldown of three, so zero is a change and silence is
	// not.
	current, err := lib.Skills().Lookup("riptide")
	if err != nil {
		t.Fatalf("look up riptide: %v", err)
	}
	if current.Cooldown == 0 {
		t.Fatal("riptide ships with no cooldown, so this proves nothing")
	}

	zero := "0"
	cleared := (SkillEdit{Cooldown: &zero}).Draft(current)
	if cleared.Cooldown != "0" {
		t.Errorf("an explicit zero drafted as %q", cleared.Cooldown)
	}
	silent := (SkillEdit{}).Draft(current)
	if silent.Cooldown != strconv.Itoa(current.Cooldown) {
		t.Errorf("an absent field drafted as %q, want the cooldown it had", silent.Cooldown)
	}
	// And an edit naming nothing is a draft that reproduces the skill exactly,
	// which is what makes "leave it" mean leave it.
	unchanged, err := silent.ResolveEdit(lib, "riptide")
	if err != nil {
		t.Fatalf("resolve an edit that names nothing: %v", err)
	}
	if !reflect.DeepEqual(unchanged, current) {
		t.Errorf("an edit naming nothing changed the skill:\n%+v\n%+v", unchanged, current)
	}
	if (SkillEdit{}).Names() {
		t.Error("an edit naming nothing reports that it names something")
	}
	if !(SkillEdit{Cooldown: &zero}).Names() {
		t.Error("an edit naming a field reports that it names nothing")
	}

	// The zero really lands, through the write.
	built, err := cleared.ResolveEdit(lib, "riptide")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	change, err := lib.EditSkill(built)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if change.After.Cooldown != 0 {
		t.Errorf("the cooldown became %d, want zero", change.After.Cooldown)
	}
	if change.Before.Cooldown != current.Cooldown {
		t.Errorf("the change reports the old cooldown as %d", change.Before.Cooldown)
	}
}

// TestAnExplicitlyEmptyListClearsARestriction covers the only way the command
// line can take a restriction back off a skill.
//
// An empty string is a real answer here, and it means the opposite of silence: it
// clears the list, which puts a neutral skill back in the common pool. The
// distinction it must not fall into is the one skill.ParseBook refuses — a list
// present and empty, satisfied by nobody — and SkillDraft.Restriction is what
// keeps them apart.
func TestAnExplicitlyEmptyListClearsARestriction(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// A skill nobody carries, so narrowing it and widening it again breaks
	// nothing.
	kept, err := SkillDraft{
		ID: "oath", Element: "neutral", Target: "enemy", Range: "1", Pattern: "single",
		Power: "1000", Strikes: "1", Accuracy: "900", Cooldown: "0",
		RestrictElements: "fire,metal", RestrictArchetypes: "bulwark",
	}.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := lib.SaveSkill(kept); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Silence on the other two lists leaves them exactly as they were, which is
	// what makes clearing one list a change to one list.
	nothing := ""
	built, err := (SkillEdit{RestrictElements: &nothing}).Draft(kept).ResolveEdit(lib, "oath")
	if err != nil {
		t.Fatalf("resolve the clearing edit: %v", err)
	}
	if names := built.Restrict.ElementNames(); len(names) != 0 {
		t.Errorf("the element allowlist is still %v", names)
	}
	if built.Restrict == nil || !slices.Equal(built.Restrict.Archetypes, []string{"bulwark"}) {
		t.Errorf("clearing the elements also cleared the roles: %+v", built.Restrict)
	}

	// And clearing every list leaves no restrict block at all, rather than an
	// empty one the parser refuses.
	all := SkillEdit{
		RestrictElements: &nothing, RestrictArchetypes: &nothing, RestrictCharacters: &nothing,
	}
	bare, err := all.Draft(kept).ResolveEdit(lib, "oath")
	if err != nil {
		t.Fatalf("resolve the fully clearing edit: %v", err)
	}
	if bare.Restrict != nil {
		t.Errorf("clearing every list left %+v, want no restriction at all", bare.Restrict)
	}
	change, err := lib.EditSkill(bare)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !AnyoneMayCarry(change.After) {
		t.Errorf("the cleared skill is not back in the common pool: %s", WhoMaySummary(change.After))
	}
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	written, err := reloaded.Skills().Lookup("oath")
	if err != nil {
		t.Fatalf("the edited skill is not in the reloaded book: %v", err)
	}
	if written.Restrict != nil {
		t.Errorf("the file still holds a restriction: %+v", written.Restrict)
	}
}

// TestEditingTheIDIsRefused is the operation this deliberately does not attempt.
//
// A rename has to change every kit, every preset's kit and every
// restrict.characters list that names the old id. Half of that — moving the
// declaration and leaving the references — is a book that does not load, so the
// refusal says a rename is a separate operation rather than doing part of one.
func TestEditingTheIDIsRefused(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	current, err := lib.Skills().Lookup("sever")
	if err != nil {
		t.Fatalf("look up sever: %v", err)
	}
	renamed := "sunder"
	_, err = (SkillEdit{ID: &renamed}).Draft(current).ResolveEdit(lib, "sever")
	var rename *SkillRenameError
	if !errors.As(err, &rename) {
		t.Fatalf("a renamed id was accepted: %v", err)
	}
	if rename.From != "sever" || rename.To != "sunder" {
		t.Errorf("the refusal reads %q → %q", rename.From, rename.To)
	}
	// The same id is not a rename, so passing it changes nothing.
	same := "sever"
	if _, err := (SkillEdit{ID: &same}).Draft(current).ResolveEdit(lib, "sever"); err != nil {
		t.Errorf("naming the id it already has was refused: %v", err)
	}
	// And a skill the book does not hold is refused as unknown rather than
	// quietly added.
	var unknown *UnknownSkillError
	if _, err := (SkillEdit{}).Draft(current).ResolveEdit(lib, "nonesuch"); !errors.As(err, &unknown) {
		t.Errorf("editing a skill that does not exist gave %v", err)
	}
}

// TestTheBeforeAndAfterDamageIsPreviewDamage is what makes an edit reportable: a
// skill is balance, so the figure that matters after a write is what moved.
//
// Both halves have to be the same PreviewDamage the form shows before a write and
// skills.golden's column is measured from, or the two figures are not comparable
// with each other and neither is comparable with the table.
func TestTheBeforeAndAfterDamageIsPreviewDamage(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	current, err := lib.Skills().Lookup("venom_fang")
	if err != nil {
		t.Fatalf("look up venom_fang: %v", err)
	}
	wanted := lib.PreviewDamage(current)

	power := "1500"
	built, err := (SkillEdit{Power: &power}).Draft(current).ResolveEdit(lib, "venom_fang")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	expected := lib.PreviewDamage(built)
	change, err := lib.EditSkill(built)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if change.BeforeDamage != wanted {
		t.Errorf("the before is %+v, want %+v", change.BeforeDamage, wanted)
	}
	if change.AfterDamage != expected {
		t.Errorf("the after is %+v, want %+v", change.AfterDamage, expected)
	}
	if !change.MovesDamage() {
		t.Error("raising a power reports that the damage did not move")
	}

	// An edit that touches no number moves no damage, which is what leaves the
	// before-and-after line off a screen that has nothing to compare.
	side := "ally"
	still, err := lib.Skills().Lookup("war_cry")
	if err != nil {
		t.Fatalf("look up war_cry: %v", err)
	}
	quiet, err := (SkillEdit{Target: &side}).Draft(still).ResolveEdit(lib, "war_cry")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	unmoved, err := lib.EditSkill(quiet)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if unmoved.MovesDamage() {
		t.Errorf("changing who a skill aims at moved the damage: %s", unmoved.DamageSummary())
	}
	// The note still says the goldens moved, because the damage is not the only
	// thing that reaches one.
	kinds := make([]NoteKind, 0, 3)
	for _, note := range lib.EditSkillNoteFacts(unmoved) {
		kinds = append(kinds, note.Kind)
	}
	if !slices.Equal(kinds, []NoteKind{NoteEdited, NoteGoldensMove, NoteRebuild}) {
		t.Errorf("an edit reports %v", kinds)
	}
}

// TestEveryShippedSkillTakesABalanceEdit answers the question an author asks
// before touching the book at all: is any of it locked?
//
// None of it is. Every shipped skill takes a change to its power, because a
// power narrows nobody — the two ways an edit can orphan a carrier are the
// element and the restriction, and neither is what balancing a skill touches.
// That is worth a test rather than a claim: if a future skill were to arrive
// carried by somebody it could not be rebalanced for, this is where it would
// show up.
func TestEveryShippedSkillTakesABalanceEdit(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, current := range lib.Skills().Skills() {
		// One more than it had, so the edit is real for every skill including the
		// ones whose power is zero on purpose.
		raised := strconv.Itoa(current.Power + 1)
		built, err := (SkillEdit{Power: &raised}).Draft(current).ResolveEdit(lib, current.ID)
		if err != nil {
			t.Errorf("%s cannot be rebalanced: %v", current.ID, err)
			continue
		}
		if _, err := lib.EditSkill(built); err != nil {
			t.Errorf("%s cannot be rebalanced: %v", current.ID, err)
			continue
		}
		// An edit names one field and must carry **every** other one through
		// untouched. The whole skill is compared rather than a chosen field, and
		// that widening is the point of this block: it was written to watch one
		// field, the restriction, after the draft held three allowlists and the
		// fourth arrived with species — so every balance edit to a lineage skill
		// dropped its lineage. Watching one field caught that one and then missed
		// the next: `flavour` was wiped by the identical mechanism, silently, for
		// as long as the clause existed.
		//
		// The mechanism is worth naming, because a third field will find it.
		// SkillAnswers and resolveOnto are hand-maintained inverses with no
		// compile-time link between them: resolveOnto assigns a field, so a field
		// SkillAnswers forgets to read back is overwritten with a zero. Anything
		// resolveOnto assigns is at risk; anything it leaves alone rides through
		// on the base skill. Comparing the whole value is the only assertion that
		// does not have to be extended each time somebody adds a field.
		after, err := lib.Skills().Lookup(current.ID)
		if err != nil {
			t.Errorf("%s is gone after its own edit: %v", current.ID, err)
			continue
		}
		// The one field the edit was allowed to change, put back before the
		// comparison, so the comparison can be of everything else at once.
		expected := current
		expected.Power = after.Power
		if !reflect.DeepEqual(expected, after) {
			t.Errorf("%s changed something other than its power:\nbefore %+v\nafter  %+v",
				current.ID, expected, after)
		}
		if after.Power != current.Power+1 {
			t.Errorf("%s: the edit set power to %d, want %d",
				current.ID, after.Power, current.Power+1)
		}
		// Crit is nought on every shipped skill, so the whole-value comparison
		// above would pass just as happily with the field dropped from the
		// answers. Give the skill one and rebalance it again: this is the
		// assertion that an edit through either front-end does not silently zero
		// a skill's critical chance, which is the failure that would leave
		// skills.json able to carry the field and the tools unable to keep it.
		if after.Power == 0 {
			continue
		}
		const critted = 200
		crit := strconv.Itoa(critted)
		withCrit, err := (SkillEdit{Crit: &crit}).Draft(after).ResolveEdit(lib, current.ID)
		if err != nil {
			t.Errorf("%s cannot be given a critical chance: %v", current.ID, err)
			continue
		}
		if _, err := lib.EditSkill(withCrit); err != nil {
			t.Errorf("%s cannot be given a critical chance: %v", current.ID, err)
			continue
		}
		again := strconv.Itoa(withCrit.Power + 1)
		rebalanced, err := (SkillEdit{Power: &again}).Draft(withCrit).ResolveEdit(lib, current.ID)
		if err != nil {
			t.Errorf("%s cannot be rebalanced once it crits: %v", current.ID, err)
			continue
		}
		if _, err := lib.EditSkill(rebalanced); err != nil {
			t.Errorf("%s cannot be rebalanced once it crits: %v", current.ID, err)
			continue
		}
		final, err := lib.Skills().Lookup(current.ID)
		if err != nil {
			t.Errorf("%s is gone after its own edit: %v", current.ID, err)
			continue
		}
		if final.Crit != critted {
			t.Errorf("%s: a power edit left the critical chance at %d, want %d",
				current.ID, final.Crit, critted)
		}
	}
	if _, err := Load(dir); err != nil {
		t.Errorf("the data directory no longer loads after editing every skill: %v", err)
	}
}

// TestAnEditRefusedForNoOneCarriersFaultKeepsTheParsersWords is the third shape
// of an edit refusal, and the one a classification must not guess at.
//
// Whether a kit demands more than two elements is a rule about the kit as a
// whole rather than about one skill in it, so no per-carrier check can attribute
// it. The refusal therefore names nobody and shows what cast said, which is the
// same thing every other diagnostic from internal/core gets. Blaming a carrier
// here would name one that is not at fault.
func TestAnEditRefusedForNoOneCarriersFaultKeepsTheParsersWords(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// bolt is neutral and sits in skirmisher's kit beside a grass skill and an
	// electric one, so giving it a third element leaves that kit uncarryable.
	current, err := lib.Skills().Lookup("bolt")
	if err != nil {
		t.Fatalf("look up bolt: %v", err)
	}
	fire := "fire"
	built, err := (SkillEdit{Element: &fire}).Draft(current).ResolveEdit(lib, "bolt")
	if err != nil {
		t.Fatalf("the skill itself is legal, so resolving it should pass: %v", err)
	}
	_, err = lib.EditSkill(built)
	var broken *SkillEditBreaksError
	if !errors.As(err, &broken) {
		t.Fatalf("a kit demanding three elements was accepted: %v", err)
	}
	if broken.ID != "" {
		t.Errorf("the refusal blames %q, and no single carrier is at fault", broken.ID)
	}
	if !strings.Contains(err.Error(), "at most 2") {
		t.Errorf("the refusal does not keep the parser's own words: %v", err)
	}
	if _, err := Load(dir); err != nil {
		t.Errorf("the data directory no longer loads: %v", err)
	}
}

func TestPercentReadsPartsPerThousand(t *testing.T) {
	for _, test := range []struct {
		permille int
		want     string
	}{
		{1000, "100%"}, {960, "96%"}, {850, "85%"}, {250, "25%"}, {0, "0%"},
		// A tenth survives, so an authored 855 is not silently reported as 85%.
		{855, "85.5%"}, {1, "0.1%"},
		// Power exceeds the base routinely, and a debuff term can be negative.
		{2200, "220%"}, {-500, "-50%"},
	} {
		if got := Percent(test.permille); got != test.want {
			t.Errorf("Percent(%d) = %q, want %q", test.permille, got, test.want)
		}
	}
}

func TestDescribeApplicationsKeepsFormatParseable(t *testing.T) {
	applications := []skill.Application{
		{Status: "weaken", Chance: 800, Stacks: 1},
		{Status: "blind", Chance: 400, Stacks: 2},
	}
	// The reader's version carries the percentages.
	described := DescribeApplications(applications)
	for _, want := range []string{"weaken:800 (80%)", "blind:400:2 (40%)"} {
		if !strings.Contains(described, want) {
			t.Errorf("DescribeApplications = %q, missing %q", described, want)
		}
	}
	if got := ApplicationChances(applications); got != "80% · 40%" {
		t.Errorf("ApplicationChances = %q", got)
	}
	// The syntax version must stay exactly what ParseApplications reads back,
	// because it is what a prefilled form holds. This is the property a
	// percentage inside FormatApplications would have broken.
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	parsed, err := lib.ParseApplications(FormatApplications(applications))
	if err != nil {
		t.Fatalf("FormatApplications no longer round-trips: %v", err)
	}
	if len(parsed) != len(applications) {
		t.Fatalf("round trip gave %d applications, want %d", len(parsed), len(applications))
	}
	for i, want := range applications {
		if parsed[i] != want {
			t.Errorf("round trip %d = %+v, want %+v", i, parsed[i], want)
		}
	}
	if _, err := lib.ParseApplications(described); err == nil {
		t.Error("the reader's version parsed, so the two are not distinct after all")
	}
}

// TestTheShapeDiagramCellShowsTheMostOfEveryShape is the design record behind
// ShapeDiagramCell: it is a choice between two cells, each of which draws one of
// the nine shipped shapes short, and this is the arithmetic that made it.
//
// The claim in that function's doc comment is checked rather than asserted:
// {4,1} is the only cell on the board from which all six one-step directions
// stay on the board and on one side, and no cell at all draws all nine shapes in
// full. If a tenth shape or a wider board makes a better cell exist, this fails
// and the comment gets rewritten with it.
func TestTheShapeDiagramCellShowsTheMostOfEveryShape(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	chosen := ShapeDiagramCell()
	if !chosen.OnBoard() {
		t.Fatalf("the diagram's cell %v is off the board", chosen)
	}
	if chosen.Side() != hex.SideEnemy {
		t.Errorf("the diagram's cell %v is on the %s side, and a skill is aimed at the other one",
			chosen, chosen.Side())
	}

	// The property that picked it: every neighbour survives Targets' two drops.
	whole := func(cell hex.Offset) bool {
		for _, direction := range pattern.Directions() {
			step := cell.Cube().Add(direction.Step()).Offset()
			if !step.OnBoard() || step.Side() != cell.Side() {
				return false
			}
		}
		return true
	}
	if !whole(chosen) {
		t.Errorf("some one-step direction leaves the board or the side from %v", chosen)
	}
	for _, cell := range hex.Cells() {
		if cell == chosen || cell.Side() != chosen.Side() {
			continue
		}
		if whole(cell) {
			t.Errorf("%v also keeps all six directions, so the choice is no longer forced", cell)
		}
	}

	// The ally half has exactly one such cell too, and it is this one rotated,
	// which is why one drawing serves a skill aimed either way. Written down
	// because it is load-bearing: without it the diagram would need a cell per
	// side and the screen would have to know which side the skill aims at to
	// draw a shape.
	mirror := hex.Place(hex.SideEnemy, chosen)
	if !whole(mirror) {
		t.Errorf("the ally half's mirror %v does not keep all six directions", mirror)
	}
	for _, cell := range hex.Cells() {
		if cell == mirror || cell.Side() != mirror.Side() {
			continue
		}
		if whole(cell) {
			t.Errorf("%v keeps all six directions as well as the mirror %v", cell, mirror)
		}
	}

	// What that buys, shape by shape, and what it costs.
	short := make([]string, 0, 1)
	for _, name := range lib.PatternNames() {
		coverage, err := lib.ShapeCoverage(name, skill.Enemy.String())
		if err != nil {
			t.Fatalf("coverage of %s: %v", name, err)
		}
		if coverage.Primary != chosen {
			t.Errorf("%s is drawn from %v, want the one cell %v", name, coverage.Primary, chosen)
		}
		if !coverage.Whole() {
			short = append(short, name)
		}
	}
	// One shape draws short, and it is the two-step chain: its second
	// upper-right step leaves the board from the middle column. Named rather
	// than counted, because which one it is is the whole of the trade.
	if want := []string{"pierce"}; !slices.Equal(short, want) {
		t.Errorf("the shapes drawing short from %v are %v, want %v", chosen, short, want)
	}

	// Every shape covers the same number of cells from the mirror, so the one
	// drawing really is the other side's shape rotated rather than a different
	// shape.
	for _, name := range lib.PatternNames() {
		shape, err := lib.Patterns().Lookup(name)
		if err != nil {
			t.Fatalf("look up %s: %v", name, err)
		}
		here, there := len(shape.Targets(chosen)), len(shape.Targets(mirror))
		if here != there {
			t.Errorf("%s catches %d cells from %v and %d from its mirror %v",
				name, here, chosen, there, mirror)
		}
	}

	// And no cell does better, so the choice was which shape is short.
	for _, cell := range hex.Cells() {
		if cell.Side() != hex.SideEnemy {
			continue
		}
		all := true
		for _, name := range lib.PatternNames() {
			shape, err := lib.Patterns().Lookup(name)
			if err != nil {
				t.Fatalf("look up %s: %v", name, err)
			}
			if len(shape.Targets(cell)) != shape.MaxTargets() {
				all = false
				break
			}
		}
		if all {
			t.Errorf("%v draws every shape in full, so it is the cell to use", cell)
		}
	}
}

// TestTheSplashShareIsTheBookOwn keeps the figure the diagram's legend quotes
// tied to the pattern book rather than to a number typed on a screen.
func TestTheSplashShareIsTheBookOwn(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	if got, want := lib.SplashShare(), Percent(lib.Patterns().SplashPower); got != want {
		t.Errorf("the splash share reads %q against the book's %q", got, want)
	}
}

// TestTheDiagramCrossesTheMidlineOnlyForAnAllSidedSkill is the coupling between
// the drawing and the engine: a diagram that showed a cell the resolution would
// drop, or hid one it would hit, would be a chooser that lies about the shape it
// is offering.
//
// No shipped shape reaches across the midline from the diagram's cell — every
// splash chain there is one step long, and one step from the middle column stays
// on the enemy half — so this is measured against a shape authored here with a
// two-step chain, which does. It is the case a future shape brings with it, and
// the point of testing it now is that nothing on screen would say so.
func TestTheDiagramCrossesTheMidlineOnlyForAnAllSidedSkill(t *testing.T) {
	// The shipped book with one shape added, rather than a book of its own: the
	// shipped skills name the shipped shapes, so a replacement would not load.
	dir := scratchData(t)
	path := filepath.Join(dir, "patterns.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the pattern book: %v", err)
	}
	var book struct {
		MaxTargets  int `json:"max_targets"`
		SplashPower int `json:"splash_power"`
		Patterns    []struct {
			Name   string     `json:"name"`
			Splash [][]string `json:"splash"`
		} `json:"patterns"`
	}
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatalf("decode the pattern book: %v", err)
	}
	book.Patterns = append(book.Patterns, struct {
		Name   string     `json:"name"`
		Splash [][]string `json:"splash"`
	}{Name: "sweep", Splash: [][]string{{"lower_left"}, {"lower_left", "lower_left"}}})
	grown, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("encode the pattern book: %v", err)
	}
	if err := os.WriteFile(path, grown, 0o644); err != nil {
		t.Fatalf("write the pattern book: %v", err)
	}
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	stopped, err := lib.ShapeCoverage("sweep", skill.Enemy.String())
	if err != nil {
		t.Fatalf("coverage aimed at the enemy: %v", err)
	}
	crossed, err := lib.ShapeCoverage("sweep", skill.All.String())
	if err != nil {
		t.Fatalf("coverage aimed at both sides: %v", err)
	}
	if stopped.Covered() != 2 {
		t.Errorf("aimed at one side the shape covers %d cells, want 2 — the second "+
			"step leaves the enemy half", stopped.Covered())
	}
	if !crossed.Whole() {
		t.Errorf("aimed at both sides the shape covers %d of %d cells, want all of them",
			crossed.Covered(), crossed.Max)
	}
	// And the extra cell really is on the other half, which is the whole of the
	// difference.
	extra := crossed.Splash[len(crossed.Splash)-1]
	if extra.Side() == crossed.Primary.Side() {
		t.Errorf("the cell the crossing walk added, %v, is on the primary's own side", extra)
	}

	// Every other side stops at the midline, so admitting one did not move the
	// rest.
	for i := range skill.SideCount {
		side := skill.Side(i)
		if side == skill.All {
			continue
		}
		coverage, err := lib.ShapeCoverage("sweep", side.String())
		if err != nil {
			t.Fatalf("coverage aimed at %s: %v", side, err)
		}
		if coverage.Covered() != stopped.Covered() {
			t.Errorf("aimed at %s the shape covers %d cells against %d for the enemy",
				side, coverage.Covered(), stopped.Covered())
		}
	}

	// An answer that is not a side is a refusal rather than a silent default: a
	// diagram drawn from a misread answer is a diagram of the wrong shape.
	if _, err := lib.ShapeCoverage("sweep", "everyone"); err == nil {
		t.Error("an unknown targeting side was accepted")
	}
}

// TestAddApplicationsWritesWhatTheParserReads is the contract between the status
// picker and the field it writes into: the field is the record, so what is
// written there has to be something ParseApplications accepts.
//
// Every case is round-tripped through the parser rather than compared to a
// string typed here, because the string is not the point — a spelling this test
// agreed with and the parser did not would pass and ship a broken field.
func TestAddApplicationsWritesWhatTheParserReads(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	cases := []struct {
		name     string
		answer   string
		statuses []string
		chance   string
		want     string
	}{
		{"into an empty field", "", []string{"poison"}, "300", "poison:300"},
		{"several at one chance", "", []string{"poison", "burn"}, "500",
			"poison:500,burn:500"},
		{"onto what is already there", "poison:300", []string{"blind"}, "400",
			"poison:300,blind:400"},
		// A half-typed list ends in a comma, and joining onto one would leave an
		// empty entry the parser reads as a shape error.
		{"onto a trailing comma", "poison:300,", []string{"blind"}, "400",
			"poison:300,blind:400"},
		// A blank chance is the default rather than a nought: nought is a status
		// that can never land, which the skill book refuses over a field the
		// author never filled in.
		{"with no chance given", "", []string{"block"}, "",
			"block:" + DefaultApplicationChance},
		{"with nothing chosen", "poison:300", nil, "400", "poison:300"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := lib.AddApplications(test.answer, test.statuses, test.chance)
			if err != nil {
				t.Fatalf("refused: %v", err)
			}
			if got != test.want {
				t.Errorf("wrote %q, want %q", got, test.want)
			}
			applications, err := lib.ParseApplications(got)
			if err != nil {
				t.Fatalf("what was written does not parse: %v", err)
			}
			// And back out again unchanged, which is what makes the field
			// something the form can be prefilled from.
			if again := FormatApplications(applications); again != got {
				t.Errorf("the round trip turned %q into %q", got, again)
			}
		})
	}

	// A status the book does not declare is refused rather than written, because
	// a field holding one is a field the write refuses later with less to go on.
	if _, err := lib.AddApplications("", []string{"no_such_status"}, "300"); err == nil {
		t.Error("an unknown status was written into the field")
	}
	// A chance that is not a number is refused for the same reason. The picker's
	// own field takes digits only, so this is the answer a script could give.
	if _, err := lib.AddApplications("", []string{"poison"}, "half"); err == nil {
		t.Error("a chance that is not a number was accepted")
	}
	// The answer is handed back unchanged on a refusal, so a rejected addition
	// cannot lose what the author had already typed.
	held := "poison:300"
	if got, _ := lib.AddApplications(held, []string{"poison"}, "half"); got != held {
		t.Errorf("a refused addition left the field %q, want %q", got, held)
	}
}

// TestTheStatusBookIsOfferedInItsOwnOrder keeps the picker's rows tied to the
// data file rather than to a map, which would shuffle them between runs.
func TestTheStatusBookIsOfferedInItsOwnOrder(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	book := lib.StatusBook()
	ids := make([]string, 0, len(book))
	for _, kind := range book {
		ids = append(ids, kind.ID)
	}
	if !slices.Equal(ids, lib.StatusIDs()) {
		t.Errorf("the facts are in the order %v against the book's %v", ids, lib.StatusIDs())
	}
	// The facts a row shows are the status's own, not a guess: poison ticks and
	// a shield does not, which is the difference that changes what a skill
	// applying one is worth.
	for _, kind := range book {
		declared, err := lib.Statuses().Lookup(kind.ID)
		if err != nil {
			t.Fatalf("look up %s: %v", kind.ID, err)
		}
		if kind.Duration != declared.Duration || kind.MaxStacks != declared.MaxStacks {
			t.Errorf("%s reads as %d turns and %d stacks against the book's %d and %d",
				kind.ID, kind.Duration, kind.MaxStacks, declared.Duration, declared.MaxStacks)
		}
		if kind.Ticks != (declared.TickPower > 0) {
			t.Errorf("%s reads as ticking %v against a tick power of %d",
				kind.ID, kind.Ticks, declared.TickPower)
		}
	}
}

// TestArtImageFitsTheBoxItWasGiven is the contract a caller lays out against:
// the size asked for is the size returned, and the picture inside it keeps its
// own proportions rather than being stretched to the corners.
func TestArtImageFitsTheBoxItWasGiven(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	character, known := lib.Characters().Get("pokemon.bulbasaur")
	if !known {
		t.Skip("the shipped cast no longer holds the character this measures")
	}
	art := character.Image

	// A square box on square art fills it; the two are checked together so a
	// stretch cannot hide behind the box being the right size.
	square, err := lib.ArtImage(art, 40, 40)
	if err != nil {
		t.Fatalf("rasterise into a square: %v", err)
	}
	if got := square.Bounds(); got.Dx() != 40 || got.Dy() != 40 {
		t.Errorf("a 40x40 box came back %v", got)
	}

	// A box twice as wide as it is tall must leave the sides empty rather than
	// stretch the picture into them. Measured on the first column, which a
	// stretched drawing would paint and a fitted one cannot.
	wide, err := lib.ArtImage(art, 80, 40)
	if err != nil {
		t.Fatalf("rasterise into a wide box: %v", err)
	}
	if got := wide.Bounds(); got.Dx() != 80 || got.Dy() != 40 {
		t.Errorf("an 80x40 box came back %v", got)
	}
	painted := func(img *image.RGBA, x int) bool {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			if img.RGBAAt(x, y).A > 0 {
				return true
			}
		}
		return false
	}
	if painted(wide, 0) || painted(wide, 79) {
		t.Error("a wide box was filled to its edges, so the art was stretched rather than fitted")
	}
	if !painted(wide, 40) {
		t.Error("the middle of a wide box is empty, so nothing was drawn at all")
	}

	// The cap is what keeps a preview a preview: an enormous terminal must not
	// ask for an enormous raster.
	huge, err := lib.ArtImage(art, MaxArtPixels*4, MaxArtPixels*4)
	if err != nil {
		t.Fatalf("rasterise into an oversized box: %v", err)
	}
	if got := huge.Bounds(); got.Dx() > MaxArtPixels || got.Dy() > MaxArtPixels {
		t.Errorf("an oversized box came back %v, over the %d cap", got, MaxArtPixels)
	}
}

// TestArtImageRefusesWhatItCannotDraw covers the two answers that are not a
// picture, because a preview showing a Go error is worse than one saying it
// cannot open the file.
func TestArtImageRefusesWhatItCannotDraw(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	if _, err := lib.ArtImage("assets/nobody-drew-this.svg", 40, 40); err == nil {
		t.Error("art that is not on disk rasterised anyway")
	}
	if _, err := lib.ArtImage("assets/bulbasaur.svg", 0, 40); err == nil {
		t.Error("a box with no width rasterised anyway")
	}
}

// TestArtImageDrawsARaster is the .png half of what an authored path allows.
// Nothing ships as one, so without this the branch is only reachable by an
// author who tries it.
func TestArtImageDrawsARaster(t *testing.T) {
	dir := scratchData(t)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// A four-pixel picture, red and opaque, so what comes back is checkable
	// without depending on a resampler's exact weights.
	source := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := range 2 {
		for y := range 2 {
			source.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatalf("encode: %v", err)
	}
	written := filepath.Join(dir, "assets", "fixture", "raster.png")
	if err := os.WriteFile(written, encoded.Bytes(), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	drawn, err := lib.ArtImage("assets/fixture/raster.png", 16, 16)
	if err != nil {
		t.Fatalf("rasterise a png: %v", err)
	}
	if got := drawn.Bounds(); got.Dx() != 16 || got.Dy() != 16 {
		t.Errorf("a 16x16 box came back %v", got)
	}
	middle := drawn.RGBAAt(8, 8)
	if middle.A == 0 || middle.R <= middle.G || middle.R <= middle.B {
		t.Errorf("the middle of a red picture is %+v", middle)
	}
}

// TestTheShippedArtIsCutOutRatherThanFramed is the defect this catches, stated
// as the rule it broke.
//
// One of the three shipped pictures was traced from a source with no alpha, so
// it carried an opaque backdrop — its first path is a rectangle across the whole
// canvas. It inked 92 percent of the box it was drawn in where the other two
// inked around a third, and in the preview it showed as a filled rectangle
// behind the sprite while the others floated. Nothing said so: the file was well
// shaped, on disk, and drew without complaint.
//
// A unit stands on a board, so its art is a cut-out and a baked background is
// always wrong here.
//
// The measurement is the corners of the *inked* rectangle rather than of the box
// the picture was drawn in, and the first version of this test got that wrong: a
// square box letterboxes a 784x731 picture, so the box's corners were transparent
// margin and the framed asset passed. The inked rectangle needs no knowledge of
// the source's proportions and separates the two cases exactly — a frame's own
// corners are painted, while a creature's silhouette does not reach all four
// corners of its tightest rectangle.
func TestTheShippedArtIsCutOutRatherThanFramed(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Skip("the shipped cast is empty, so there is no art to measure")
	}
	const side = 64
	seen := 0
	for _, character := range characters {
		for _, art := range character.Art() {
			drawn, err := lib.ArtImage(art.Image, side, side)
			if err != nil {
				t.Errorf("%s: %v", art.Image, err)
				continue
			}
			inked, any := inkBounds(drawn)
			if !any {
				t.Errorf("%s drew nothing at all", art.Image)
				continue
			}
			seen++
			for _, corner := range []image.Point{
				{X: inked.Min.X, Y: inked.Min.Y},
				{X: inked.Max.X - 1, Y: inked.Min.Y},
				{X: inked.Min.X, Y: inked.Max.Y - 1},
				{X: inked.Max.X - 1, Y: inked.Max.Y - 1},
			} {
				if alpha := drawn.RGBAAt(corner.X, corner.Y).A; alpha > 8 {
					t.Errorf("%s has paint in the corner of what it drew, at %v (alpha %d): it carries a background rather than being cut out",
						art.Image, corner, alpha)
				}
			}
		}
	}
	if seen == 0 {
		t.Error("no picture was measured, so this asserts nothing")
	}
}

// inkBounds is the tightest rectangle holding every pixel with paint in it.
func inkBounds(drawn *image.RGBA) (image.Rectangle, bool) {
	bounds := drawn.Bounds()
	inked := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	any := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if drawn.RGBAAt(x, y).A <= 8 {
				continue
			}
			any = true
			inked.Min.X = min(inked.Min.X, x)
			inked.Min.Y = min(inked.Min.Y, y)
			inked.Max.X = max(inked.Max.X, x+1)
			inked.Max.Y = max(inked.Max.Y, y+1)
		}
	}
	return inked, any
}

// TestTraitCarriersReadsTheLearnsetFromTheOtherEnd is the answer a trait listing
// needs and the trait book cannot give.
//
// A trait is declared in passives.json knowing nothing about who takes it, and
// the edge lives on the character: a learnset is the character's fact. So this
// walks the cast rather than indexing off the trait, and the test that matters
// is that the two ends agree — every carrier it names has the trait in its own
// learnset, and every character whose learnset holds it is named.
func TestTraitCarriersReadsTheLearnsetFromTheOtherEnd(t *testing.T) {
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the scratch data: %v", err)
	}
	declared := lib.Passives().All()
	if len(declared) == 0 {
		t.Skip("no traits declared")
	}
	for _, held := range declared {
		carriers := lib.TraitCarriers(held.ID)
		named := make(map[string]TraitCarrier, len(carriers))
		for _, carrier := range carriers {
			if _, twice := named[carrier.Character]; twice {
				t.Errorf("%q names %q as a carrier twice", held.ID, carrier.Character)
			}
			named[carrier.Character] = carrier
		}
		for _, character := range lib.Characters().All() {
			holds := false
			for _, entry := range character.Passives {
				if entry.ID == held.ID {
					holds = true
					// The gates are carried through rather than re-read, so a
					// listing can print "endurance@16" without the learnset.
					if carrier, listed := named[character.ID]; !listed {
						t.Errorf("%q learns %q and TraitCarriers does not name it",
							character.ID, held.ID)
					} else if carrier.AtLevel != entry.AtLevel {
						t.Errorf("%q learns %q at level %d and TraitCarriers says %d",
							character.ID, held.ID, entry.AtLevel, carrier.AtLevel)
					}
					break
				}
			}
			if _, listed := named[character.ID]; listed && !holds {
				t.Errorf("TraitCarriers names %q as carrying %q, and its learnset does not",
					character.ID, held.ID)
			}
		}
	}
}

// TestTraitCarrierSummaryMarksTheGates is the row a listing prints: one token per
// carrier, with whatever gates the entry declares — the shape UnlockSummary gives
// a learnset, read from the other end.
func TestTraitCarrierSummaryMarksTheGates(t *testing.T) {
	for _, test := range []struct {
		name     string
		carriers []TraitCarrier
		want     string
	}{
		{"nobody", nil, ""},
		{"one from the start", []TraitCarrier{{Character: "a.b"}}, "a.b"},
		{"a level gate", []TraitCarrier{{Character: "a.b", AtLevel: 16}}, "a.b@16"},
		{"level one is not a gate", []TraitCarrier{{Character: "a.b", AtLevel: 1}}, "a.b"},
		{"a form gate", []TraitCarrier{{Character: "a.b", Stages: []string{"Grown"}}}, "a.b[Grown]"},
		{"both, and two of them",
			[]TraitCarrier{{Character: "a.b", AtLevel: 16, Stages: []string{"Grown"}}, {Character: "c.d"}},
			"a.b@16[Grown] c.d"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := TraitCarrierSummary(test.carriers); got != test.want {
				t.Errorf("TraitCarrierSummary = %q, want %q", got, test.want)
			}
		})
	}
}

// TestADataDirectoryWithNoCatalogueStillLoads is the one book a working directory
// is allowed to be missing, and it is worth a test because of the shape the
// failure would otherwise take: the whole tool refusing to start over a file that
// has nothing to do with the character being authored.
//
// A working directory is a copy somebody took, and every copy taken before
// builds.json existed is complete in every other way. So the absence reads as an
// older directory rather than a broken one, and what it reads as is an empty
// catalogue — which is exactly what a catalogue holding no builds says.
func TestADataDirectoryWithNoCatalogueStillLoads(t *testing.T) {
	dir := scratchData(t)
	if err := os.Remove(filepath.Join(dir, buildsFile)); err != nil {
		t.Fatalf("take the catalogue away: %v", err)
	}
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load a directory with no catalogue: %v", err)
	}
	if builds := lib.Builds(); len(builds) != 0 {
		t.Errorf("a directory with no catalogue holds %d builds", len(builds))
	}
	// And the per-character question answers the same way, because that is the one
	// a screen asks per row: a book left nil and reached through it would panic on
	// the first character drawn rather than on the load.
	for _, character := range lib.Characters().All() {
		if found := lib.BuildsOf(character.ID); len(found) != 0 {
			t.Errorf("%s has %d builds out of a directory with no catalogue",
				character.ID, len(found))
		}
	}
}

// TestTheShippedCatalogueIsReadThroughTheLibrary is the plumbing a screen stands
// on: the two questions a listing asks — every build, and one character's — have
// to agree, because the listing is built by asking the second once per character
// and is measured against the first.
func TestTheShippedCatalogueIsReadThroughTheLibrary(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	every := lib.Builds()
	if len(every) == 0 {
		t.Fatal("the shipped catalogue is empty, so nothing below proves anything")
	}
	grouped := 0
	for _, character := range lib.Characters().All() {
		for _, built := range lib.BuildsOf(character.ID) {
			if built.Character != character.ID {
				t.Errorf("%s's builds include %q, which is for %q",
					character.ID, built.ID, built.Character)
			}
			grouped++
		}
	}
	if grouped != len(every) {
		t.Errorf("the catalogue holds %d builds and the characters account for %d",
			len(every), grouped)
	}
}
