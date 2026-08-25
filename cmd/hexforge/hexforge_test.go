package main

import (
	"bufio"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// shippedDataDir is the real data directory, relative to this package.
const shippedDataDir = "../../internal/seed/data"

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
	report, err := inspect(shippedDataDir)
	if err != nil {
		t.Fatalf("the shipped data does not load: %v", err)
	}
	if !report.ok() {
		t.Errorf("the shipped data has problems: %v", report.problems)
	}
	if len(report.rows) == 0 {
		t.Fatal("no characters were inspected, so nothing here is being exercised")
	}
	for _, row := range report.rows {
		if !row.imageExists {
			t.Errorf("%s names art that is not there: %s", row.id, row.image)
		}
		if row.failure != nil {
			t.Errorf("%s does not resolve: %v", row.id, row.failure)
		}
		if row.headroom < 0 {
			t.Errorf("%s absorbs %d, over the budget", row.id, row.effective)
		}
	}
}

// TestInspectNoticesMissingArt is the one question internal/core/cast is not
// allowed to ask, so it has to be asked here.
func TestInspectNoticesMissingArt(t *testing.T) {
	dir := scratchData(t)
	before, err := inspect(dir)
	if err != nil {
		t.Fatalf("inspect the copy: %v", err)
	}
	if !before.ok() {
		t.Fatalf("the copied data already has problems: %v", before.problems)
	}

	// Take away exactly one file. The path shape is unchanged, so only a
	// program that reads the filesystem can notice.
	removed := filepath.Join(dir, "assets", "example", "sprout.svg")
	if err := os.Remove(removed); err != nil {
		t.Fatalf("remove %s: %v", removed, err)
	}
	after, err := inspect(dir)
	if err != nil {
		t.Fatalf("inspect after removing the art: %v", err)
	}
	if after.ok() {
		t.Fatal("the missing art was not noticed")
	}
	if len(after.problems) != 1 {
		t.Errorf("%d problems reported, want 1: %v", len(after.problems), after.problems)
	}
	if !strings.Contains(after.problems[0], "sprout.svg") {
		t.Errorf("the problem is %q, want it to name the missing file", after.problems[0])
	}
	missing, present := 0, 0
	for _, row := range after.rows {
		if row.imageExists {
			present++
		} else {
			missing++
		}
	}
	if missing != 1 || present != len(after.rows)-1 {
		t.Errorf("%d characters are missing art and %d are not, want exactly one missing",
			missing, present)
	}
	// The report is still a full report: taking away a file must not stop the
	// budget being tabulated for everyone else.
	if len(after.rows) != len(before.rows) {
		t.Errorf("the report covers %d characters, want %d", len(after.rows), len(before.rows))
	}
	// And it renders without panicking on the row that has no art.
	after.render(io.Discard)
}

// TestWrittenCastIsStableAndReloads is what makes the tool safe to run twice:
// the bytes are a function of the content, and the content survives the trip
// through the file.
func TestWrittenCastIsStableAndReloads(t *testing.T) {
	dir := scratchData(t)
	lib, err := load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	first, err := lib.characters.Marshal()
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

	character, err := draft{
		id: "example-film.tester", name: "Tester", origin: "example-film",
		archetype: "duelist", image: "assets/example/tester.png", element: "wind/ground",
		bio: "Written by a test.",
	}.resolve(lib)
	if err != nil {
		t.Fatalf("resolve a draft: %v", err)
	}
	grown, err := lib.characters.Append(lib.castDeps(), character)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := lib.writeCast(grown); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded, err := load(dir)
	if err != nil {
		t.Fatalf("reload after writing: %v", err)
	}
	returned, known := reloaded.characters.Get(character.ID)
	if !known {
		t.Fatal("the written character is not in the reloaded book")
	}
	if !reflect.DeepEqual(returned, character) {
		t.Errorf("the trip through the file changed the character:\n%+v\n%+v", returned, character)
	}
	// Writing the reloaded book back has to produce the same bytes, or every
	// run of the tool would rewrite the file for no reason.
	againRaw, err := reloaded.characters.Marshal()
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
}

func TestResolveTakesTheArchetypeCurveAndKit(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	preset, known := lib.archetypes.Get("duelist")
	if !known {
		t.Fatal("the duelist preset is not shipped")
	}
	character, err := draft{
		id: "example-film.tester", name: "Tester", origin: "example-film",
		archetype: "duelist", image: "assets/example/tester.svg", element: "wind/ground",
	}.resolve(lib)
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
	overridden := draft{
		id: "example-film.tester", name: "Tester", origin: "example-film",
		archetype: "duelist", image: "assets/example/tester.svg", element: "wind/ground",
		skills: "strike, bolt",
	}
	overridden.stats[progression.Speed] = "40:120"
	tuned, err := overridden.resolve(lib)
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
}

func TestResolveRejections(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	good := func() draft {
		return draft{
			id: "example-film.tester", name: "Tester", origin: "example-film",
			archetype: "duelist", image: "assets/example/tester.svg", element: "wind/ground",
		}
	}
	if _, err := good().resolve(lib); err != nil {
		t.Fatalf("the base draft should resolve: %v", err)
	}
	cases := []struct {
		name   string
		change func(*draft)
		wantIn string
	}{
		{"no id", func(d *draft) { d.id = "" }, "character id is empty"},
		{"non-slug id", func(d *draft) { d.id = "Example.Tester" }, "lowercase letters"},
		{"an id already in the cast", func(d *draft) { d.id = "example-anime.adept" }, "already in the cast"},
		{"no name", func(d *draft) { d.name = "  " }, "display name"},
		{"unknown origin", func(d *draft) { d.origin = "nowhere" }, "unknown origin"},
		{"unknown archetype", func(d *draft) { d.archetype = "berserker" }, "unknown archetype"},
		{"bad image extension", func(d *draft) { d.image = "assets/a.gif" }, "want .svg or .png"},
		{"absolute image", func(d *draft) { d.image = "/assets/a.svg" }, "absolute path"},
		{"unknown element", func(d *draft) { d.element = "plasma" }, "unknown element"},
		{"three elements", func(d *draft) { d.element = "fire/wind/ice" }, "want one or two"},
		{"an element pair the chart refuses", func(d *draft) { d.element = "water/fire" }, "counter each other"},
		{"unknown skill", func(d *draft) { d.skills = "strike,meteor" }, "unknown skill"},
		{"a duplicated skill", func(d *draft) { d.skills = "strike,strike" }, "twice"},
		{"an unreadable curve", func(d *draft) { d.stats[progression.HP] = "780" }, "want base:max"},
		{"a curve with an unreadable base", func(d *draft) { d.stats[progression.HP] = "x:2600" }, "unreadable base"},
		{"a curve that shrinks with level", func(d *draft) { d.stats[progression.HP] = "2600:780" }, "may not shrink"},
		{"a curve starting at nothing", func(d *draft) { d.stats[progression.HP] = "0:2600" }, "positive value"},
		{"a curve over its ceiling", func(d *draft) { d.stats[progression.Speed] = "60:260" }, "over the ceiling"},
		{
			// Health and defence multiply, so the joint bound is the one an
			// author is most likely to walk into without noticing.
			name: "a pair of curves over the joint durability budget",
			change: func(d *draft) {
				d.stats[progression.HP] = "1440:4800"
				d.stats[progression.Defense] = "240:800"
			},
			wantIn: "over the budget",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := good()
			test.change(&candidate)
			_, err := candidate.resolve(lib)
			if err == nil {
				t.Fatalf("the draft resolved, want a rejection mentioning %q", test.wantIn)
			}
			if !strings.Contains(err.Error(), test.wantIn) {
				t.Errorf("the rejection is %q, want it to mention %q", err, test.wantIn)
			}
		})
	}
}

// TestFillRePromptsOnABadAnswer is the behaviour that keeps the wizard usable:
// a wrong answer costs one line, not the whole session.
//
// The answers are in prompt order, and the kit comes before the element on
// purpose — the kit is what decides which elements are legal.
func TestFillRePromptsOnABadAnswer(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	script := strings.Join([]string{
		"Example.Bad",          // rejected: not a slug
		"example-film.tester",  // accepted
		"Tester",               // name
		"nowhere",              // rejected: unknown origin
		"example-film",         // accepted
		"berserker",            // rejected: unknown archetype
		"duelist",              // accepted
		"",                     // art: take the suggested default
		"",                     // kit: take the preset's
		"water/ice",            // rejected: a legal pair that cannot carry gale_slash
		"water/fire",           // rejected: the chart refuses the pair itself
		"wind/ground",          // accepted
		"", "", "", "", "", "", // the six curves: take the preset
		"A tester.", // biography
	}, "\n") + "\n"
	prompt := &prompter{in: bufio.NewReader(strings.NewReader(script)), out: io.Discard, interactive: true}
	filled, err := fill(draft{}, lib, prompt)
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	character, err := filled.resolve(lib)
	if err != nil {
		t.Fatalf("resolve what the wizard collected: %v", err)
	}
	if character.ID != "example-film.tester" || character.Name != "Tester" {
		t.Errorf("collected %s / %s", character.ID, character.Name)
	}
	if character.Element.String() != "wind/ground" {
		t.Errorf("the element is %s, want the answer that could carry the kit", character.Element)
	}
	if character.Image != suggestedImage("example-film.tester") {
		t.Errorf("the art is %q, want the suggested default %q",
			character.Image, suggestedImage("example-film.tester"))
	}
	if character.Bio != "A tester." {
		t.Errorf("the biography is %q", character.Bio)
	}
	preset, _ := lib.archetypes.Get("duelist")
	if character.Stages[0].Stats != preset.Stats {
		t.Error("pressing Enter through the curves did not take the preset")
	}
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
		t.Errorf("pressing Enter through the kit gave %v", character.Skills)
	}
}

// TestElementPromptFollowsTheKitNotThePreset is why the kit is asked first.
//
// A substituted kit demands something different from the preset's, and it is the
// substituted kit that the write will be checked against, so it has to be the
// one the prompt checks too.
func TestElementPromptFollowsTheKitNotThePreset(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// The duelist preset demands wind. Substituting a water kit moves the
	// demand to water, so "wind" must now be refused and "water" accepted.
	given := draft{
		id: "example-film.tester", name: "Tester", origin: "example-film",
		archetype: "duelist", image: "assets/example/tester.svg",
		skills: "strike,riptide",
	}
	script := strings.Join([]string{
		"wind/ground", // rejected: the substituted kit needs water
		"water/ice",   // accepted
		"", "", "", "", "", "",
		"",
	}, "\n") + "\n"
	prompt := &prompter{in: bufio.NewReader(strings.NewReader(script)), out: io.Discard, interactive: true}
	filled, err := fill(given, lib, prompt)
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	character, err := filled.resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if character.Element.String() != "water/ice" {
		t.Errorf("the element is %s, want the answer the substituted kit allows", character.Element)
	}

	// And the same rule refuses it at the write, which is the guarantee the
	// prompt is only bringing forward: skill.CanCarry is the single declaration.
	mismatched := given
	mismatched.element = "wind/ground"
	if _, err := mismatched.resolve(lib); err == nil {
		t.Fatal("a character was resolved carrying a water skill without water")
	} else if !strings.Contains(err.Error(), "riptide") {
		t.Errorf("the rejection is %q, want it to name riptide", err)
	}
}

// TestResolveRefusesAKitTheAffinityCannotCarry is the exact reproduction the
// coordinator reported: sentinel's kit is water, so a fire character built from
// it wrote cleanly and was then refused by battle.New.
func TestResolveRefusesAKitTheAffinityCannotCarry(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = draft{
		id: "example-anime.mismatch", name: "Mismatch", origin: "example-anime",
		archetype: "sentinel", image: "assets/example/adept.svg", element: "fire",
	}.resolve(lib)
	if err == nil {
		t.Fatal("a fire character carrying the sentinel kit was accepted")
	}
	for _, want := range []string{"riptide", "water", "fire"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection is %q, want it to mention %q", err, want)
		}
	}
}

// TestFillWithoutATerminalTakesEveryDefault is the promise the wizard makes:
// it asks only for what is still missing, and a preset-supplied value is not
// missing. Without this, the only way through a scripted run was to pipe the
// right number of blank lines and hope.
func TestFillWithoutATerminalTakesEveryDefault(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	unattended := func() *prompter {
		return &prompter{in: bufio.NewReader(strings.NewReader("")), out: io.Discard, interactive: false}
	}

	// Nothing given: the first field with no default is named, by its flag.
	if _, err := fill(draft{}, lib, unattended()); err == nil {
		t.Fatal("a draft with nothing in it was accepted without a terminal")
	} else if !strings.Contains(err.Error(), "--id") {
		t.Errorf("the error is %q, want it to name --id", err)
	}

	// Every field with no default is required, and each one says so by name.
	full := draft{
		id: "example-film.tester", name: "Tester", origin: "example-film",
		archetype: "duelist", element: "wind/ground",
	}
	for _, missing := range []struct {
		flag  string
		clear func(*draft)
	}{
		{"name", func(d *draft) { d.name = "" }},
		{"origin", func(d *draft) { d.origin = "" }},
		{"archetype", func(d *draft) { d.archetype = "" }},
		{"element", func(d *draft) { d.element = "" }},
	} {
		t.Run("missing "+missing.flag, func(t *testing.T) {
			candidate := full
			missing.clear(&candidate)
			_, err := fill(candidate, lib, unattended())
			if err == nil {
				t.Fatalf("a draft with no --%s was accepted without a terminal", missing.flag)
			}
			if !strings.Contains(err.Error(), "--"+missing.flag) {
				t.Errorf("the error is %q, want it to name --%s", err, missing.flag)
			}
		})
	}

	// With those in, the kit, the six curves, the art and the biography all come
	// from their defaults and nothing is asked.
	filled, err := fill(full, lib, unattended())
	if err != nil {
		t.Fatalf("a fully flagged draft was refused without a terminal: %v", err)
	}
	character, err := filled.resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	preset, _ := lib.archetypes.Get("duelist")
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
		t.Errorf("the kit is %v, want the preset's %v", character.Skills, preset.Skills)
	}
	if character.Stages[0].Stats != preset.Stats {
		t.Error("the curves are not the preset's")
	}
	if character.Image != suggestedImage(full.id) {
		t.Errorf("the art is %q, want the suggested default %q", character.Image, suggestedImage(full.id))
	}
	if character.Bio != "" {
		t.Errorf("the biography is %q, want it left empty", character.Bio)
	}

	// A bad flag is reported rather than re-asked, and names the flag.
	broken := full
	broken.element = "plasma"
	if _, err := fill(broken, lib, unattended()); err == nil || !strings.Contains(err.Error(), "--element") {
		t.Errorf("a bad --element gave %v, want an error naming the flag", err)
	}
}

// TestFillFallsBackWhenStdinEnds is the reported bug at the unit level.
//
// os.Stdin.Stat cannot tell a terminal from /dev/null, so a run with stdin
// closed looks interactive and then hits EOF on the first prompt. Failing there
// would abandon a session whose remaining fields all had perfectly good
// defaults, so EOF turns the rest of the session unattended.
func TestFillFallsBackWhenStdinEnds(t *testing.T) {
	lib, err := load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Answered up to the element, then the input simply stops.
	script := "example-film.tester\nTester\nexample-film\nduelist\n\n\nwind/ground\n"
	prompt := &prompter{in: bufio.NewReader(strings.NewReader(script)), out: io.Discard, interactive: true}
	filled, err := fill(draft{}, lib, prompt)
	if err != nil {
		t.Fatalf("fill stopped when the input ended: %v", err)
	}
	if prompt.interactive {
		t.Error("the prompter still thinks somebody is answering")
	}
	character, err := filled.resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	preset, _ := lib.archetypes.Get("duelist")
	if character.Stages[0].Stats != preset.Stats {
		t.Error("the curves after the input ended are not the preset's")
	}
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
		t.Errorf("the kit after the input ended is %v", character.Skills)
	}

	// And when the input stops before a field that has no default, the error
	// still names the flag rather than reporting a bare EOF.
	short := &prompter{in: bufio.NewReader(strings.NewReader("example-film.tester\n")), out: io.Discard, interactive: true}
	if _, err := fill(draft{}, lib, short); err == nil {
		t.Fatal("a session that ended before the name was accepted")
	} else if !strings.Contains(err.Error(), "--name") {
		t.Errorf("the error is %q, want it to name --name", err)
	}
}

// TestNewRunsEndToEndThroughAPipe is the reported bug at the level it was
// reported: the binary, a real pipe on stdin, flags only.
//
// exec.Command hands the child an os.Pipe whenever Stdin is not an *os.File, so
// this is the genuine article rather than a redirect from /dev/null — which is a
// character device and therefore indistinguishable from a terminal.
func TestNewRunsEndToEndThroughAPipe(t *testing.T) {
	binary := buildHexforge(t)
	dir := scratchData(t)

	run := func(args ...string) (string, error) {
		command := exec.Command(binary, args...)
		command.Stdin = strings.NewReader("")
		output, err := command.CombinedOutput()
		return string(output), err
	}

	// The exact invocation that used to stop at "kit [strike,riptide,...]: EOF".
	output, err := run("new", "--data", dir,
		"--id", "example-anime.piped", "--name", "Piped",
		"--origin", "example-anime", "--archetype", "sentinel",
		"--element", "water/ice", "--image", "assets/example/piped.svg", "--yes")
	if err != nil {
		t.Fatalf("a flags-only run through a pipe failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "wrote example-anime.piped") {
		t.Errorf("the run did not report a write:\n%s", output)
	}
	// The art does not exist yet, and the write says so rather than pretending
	// the character is finished.
	if !strings.Contains(output, "piped.svg is not there yet") {
		t.Errorf("the write did not warn about the missing art:\n%s", output)
	}

	lib, err := load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	written, known := lib.characters.Get("example-anime.piped")
	if !known {
		t.Fatal("the character was not written")
	}
	preset, _ := lib.archetypes.Get("sentinel")
	if !reflect.DeepEqual(written.Skills, preset.Skills) {
		t.Errorf("the written kit is %v, want the preset's %v", written.Skills, preset.Skills)
	}
	if written.Stages[0].Stats != preset.Stats {
		t.Error("the written curves are not the preset's")
	}

	// A field with no default fails, naming its flag, instead of hanging or
	// reporting an EOF nobody can act on.
	output, err = run("new", "--data", dir,
		"--id", "example-anime.nameless", "--origin", "example-anime",
		"--archetype", "sentinel", "--element", "water/ice", "--yes")
	if err == nil {
		t.Fatalf("a run with no --name succeeded:\n%s", output)
	}
	if !strings.Contains(output, "--name") {
		t.Errorf("the failure does not name --name:\n%s", output)
	}

	// The kit-versus-element rule fires here too, where it is authored, rather
	// than later in battle.New.
	output, err = run("new", "--data", dir,
		"--id", "example-anime.mismatch", "--name", "Mismatch",
		"--origin", "example-anime", "--archetype", "sentinel",
		"--element", "fire", "--image", "assets/example/adept.svg", "--yes")
	if err == nil {
		t.Fatalf("a fire character carrying the sentinel kit was written:\n%s", output)
	}
	if !strings.Contains(output, "riptide") {
		t.Errorf("the failure does not name the skill it cannot carry:\n%s", output)
	}

	// Without --yes there is nobody to confirm to, so it says so rather than
	// writing on somebody's behalf.
	output, err = run("new", "--data", dir,
		"--id", "example-anime.unconfirmed", "--name", "Unconfirmed",
		"--origin", "example-anime", "--archetype", "sentinel",
		"--element", "water/ice", "--image", "assets/example/adept.svg")
	if err == nil {
		t.Fatalf("an unconfirmed run wrote anyway:\n%s", output)
	}
	if !strings.Contains(output, "--yes") {
		t.Errorf("the failure does not suggest --yes:\n%s", output)
	}

	// check on the resulting directory reports the one thing that is now wrong:
	// the character that was written names art nobody has drawn. Note the path
	// shape was fine, so only a program allowed to read the filesystem can say
	// this.
	output, err = run("check", "--data", dir)
	if err == nil {
		t.Fatalf("check passed with art missing:\n%s", output)
	}
	if !strings.Contains(output, "MISSING") {
		t.Errorf("check did not flag the missing art:\n%s", output)
	}
}

// buildHexforge builds the binary under test once, so the end-to-end run
// exercises the real program rather than a re-entry into the test binary.
func buildHexforge(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "hexforge")
	build := exec.Command("go", "build", "-o", binary, "./cmd/hexforge")
	build.Dir = "../.."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build hexforge: %v\n%s", err, output)
	}
	return binary
}

// TestParseArgsAllowsFlagsAfterAnOperand covers the reason parseArgs exists:
// flag.Parse stops at the first operand, which would have made a level silently
// ignored.
func TestParseArgsAllowsFlagsAfterAnOperand(t *testing.T) {
	cases := []struct {
		name         string
		args         []string
		wantOperands []string
		wantLevel    int
	}{
		{"flag after the operand", []string{"some.id", "--level", "30"}, []string{"some.id"}, 30},
		{"flag before the operand", []string{"--level", "30", "some.id"}, []string{"some.id"}, 30},
		{"no flag at all", []string{"some.id"}, []string{"some.id"}, 60},
		{"no operand at all", []string{"--level", "7"}, []string{}, 7},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			set := flag.NewFlagSet("test", flag.ContinueOnError)
			set.SetOutput(io.Discard)
			level := set.Int("level", 60, "")
			operands, err := parseArgs(set, test.args)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !reflect.DeepEqual(operands, test.wantOperands) && len(operands) != len(test.wantOperands) {
				t.Errorf("operands are %v, want %v", operands, test.wantOperands)
			}
			if *level != test.wantLevel {
				t.Errorf("level is %d, want %d", *level, test.wantLevel)
			}
		})
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
		if got := suggestedImage(id); got != want {
			t.Errorf("suggestedImage(%q) is %q, want %q", id, got, want)
		}
	}
}

func TestLoadRejectsAMissingDirectory(t *testing.T) {
	if _, err := load(filepath.Join(t.TempDir(), "nothing-here")); err == nil {
		t.Error("a directory with no data files loaded")
	}
	if _, err := load(""); err == nil {
		t.Error("an empty data directory loaded")
	}
}

// TestReplaceFileLeavesTheOldOneOnFailure is the reason a write goes through a
// temporary file: a truncated data file is a data file that stops the game
// booting.
func TestReplaceFileLeavesTheOldOneOnFailure(t *testing.T) {
	dir := scratchData(t)
	lib, err := load(dir)
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
