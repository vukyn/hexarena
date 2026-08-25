package forge

import (
	"encoding/json"
	"errors"
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
		if !row.ImageExists {
			t.Errorf("%s names art that is not there: %s", row.ID, row.Image)
		}
		if row.Failure != nil {
			t.Errorf("%s does not resolve: %v", row.ID, row.Failure)
		}
		if row.Budget.Over() {
			t.Errorf("%s absorbs %d, over the budget", row.ID, row.Budget.Effective)
		}
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
		if row.ImageExists {
			present++
		} else {
			missing++
		}
	}
	if missing != 1 || present != len(after.Rows)-1 {
		t.Errorf("%d characters are missing art and %d are not, want exactly one missing",
			missing, present)
	}
	// The report is still a full report: taking away a file must not stop the
	// budget being tabulated for everyone else.
	if len(after.Rows) != len(before.Rows) {
		t.Errorf("the report covers %d characters, want %d", len(after.Rows), len(before.Rows))
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
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
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
	if !reflect.DeepEqual(tuned.Skills, []string{"strike", "bolt"}) {
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
	// A directory where the temp file has to go cannot be written into, so the
	// replacement fails before the target is touched.
	lib.dir = filepath.Join(dir, "does-not-exist")
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
	lib, err := Load(shippedDataDir)
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
	unrestricted := skill.Skill{ID: "strike", Element: element.Neutral}

	answered := Carrier{
		ID: "example.adept", Archetype: "duelist", Affinity: water, HasAffinity: true,
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
	if err := CheckSkill(answered, unrestricted); err != nil {
		t.Errorf("an unrestricted skill was refused: %v", err)
	}

	// Nothing answered yet: the kit may be filled in first, and none of the
	// three lists has anything to judge against.
	empty := Carrier{}
	for _, carried := range []skill.Skill{keptForFire, keptForBulwark, keptForSprout} {
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

// TestPreviewDamageIsTheEngineArithmetic pins the reference pair, because it is
// the whole reason the figure is worth showing: it has to be the same pair
// skills.golden's damage column is measured from, or an author reads two numbers
// for one skill and cannot tell which the design was made from.
func TestPreviewDamageIsTheEngineArithmetic(t *testing.T) {
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
	if !holds || !slices.Contains(carrier.Skills, "riptide") {
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
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
	lib, err := Load(shippedDataDir)
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
