package forge

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// shippedDataDir is the real data directory, relative to this package.
const shippedDataDir = "../seed/data"

// scratchData copies the shipped data into a temporary directory, so a test may
// write to it without touching the repository.
func scratchData(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	copyTree(t, shippedDataDir, target)
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
	removed := filepath.Join(dir, "assets", "example", "sprout.svg")
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
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Image: "assets/example/tester.png", Element: "wind/ground",
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
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	preset, known := lib.Archetypes().Get("duelist")
	if !known {
		t.Fatal("the duelist preset is not shipped")
	}
	character, err := Draft{
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Image: "assets/example/tester.svg", Element: "wind/ground",
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
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Image: "assets/example/tester.svg", Element: "wind/ground",
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
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	good := func() Draft {
		return Draft{
			ID: "example-film.tester", Name: "Tester", Origin: "example-film",
			Archetype: "duelist", Image: "assets/example/tester.svg", Element: "wind/ground",
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
		{"an id already in the cast", func(d *Draft) { d.ID = "example-anime.adept" }, "already in the cast"},
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
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = Draft{
		ID: "example-anime.mismatch", Name: "Mismatch", Origin: "example-anime",
		Archetype: "sentinel", Image: "assets/example/adept.svg", Element: "fire",
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
		"example-anime.adept": "assets/example-anime/adept.svg",
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
	for _, want := range []string{"assets/example/adept.svg", "assets/example/sprout.svg"} {
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
		Carrier{ID: "example-film.tester", Archetype: "duelist"})
	if !errors.As(refused, &byArchetype) {
		t.Fatalf("a duelist was allowed a skill kept for bulwark: %v", refused)
	}
	if byArchetype.Skill != "bulwark_oath" {
		t.Errorf("the refusal names %q", byArchetype.Skill)
	}
	if err := lib.ValidateKitFor("strike,bulwark_oath",
		Carrier{ID: "example-film.tester", Archetype: "bulwark"}); err != nil {
		t.Errorf("a bulwark was refused a skill kept for bulwark: %v", err)
	}

	// And the write's check, which is the parser's and is what actually holds:
	// even with the answer check skipped, the character cannot be written.
	_, err = Draft{
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Image: "assets/example/adept.svg", Element: "neutral",
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
