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

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/forge"
)

// shippedDataDir is the real data directory, relative to this package.
const shippedDataDir = "../../internal/seed/data"

// scratchData copies the shipped data into a temporary directory, so a test may
// write to it without touching the repository. internal/forge's tests carry
// their own copy of this: it is test scaffolding rather than a rule, and
// exporting it would put test-only code in the package the game links.
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

// TestFillRePromptsOnABadAnswer is the behaviour that keeps the wizard usable:
// a wrong answer costs one line, not the whole session.
//
// The answers are in prompt order, and the kit comes before the element on
// purpose — the kit is what decides which elements are legal.
func TestFillRePromptsOnABadAnswer(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
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
	filled, err := fill(forge.Draft{}, lib, prompt)
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	character, err := filled.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve what the wizard collected: %v", err)
	}
	if character.ID != "example-film.tester" || character.Name != "Tester" {
		t.Errorf("collected %s / %s", character.ID, character.Name)
	}
	if character.Element.String() != "wind/ground" {
		t.Errorf("the element is %s, want the answer that could carry the kit", character.Element)
	}
	if character.Image != forge.SuggestedImage("example-film.tester") {
		t.Errorf("the art is %q, want the suggested default %q",
			character.Image, forge.SuggestedImage("example-film.tester"))
	}
	if character.Bio != "A tester." {
		t.Errorf("the biography is %q", character.Bio)
	}
	preset, _ := lib.Archetypes().Get("duelist")
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
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// The duelist preset demands wind. Substituting a water kit moves the
	// demand to water, so "wind" must now be refused and "water" accepted.
	given := forge.Draft{
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Image: "assets/example/tester.svg",
		Skills: "strike,riptide",
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
	character, err := filled.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if character.Element.String() != "water/ice" {
		t.Errorf("the element is %s, want the answer the substituted kit allows", character.Element)
	}

	// And the same rule refuses it at the write, which is the guarantee the
	// prompt is only bringing forward: skill.CanCarry is the single declaration.
	mismatched := given
	mismatched.Element = "wind/ground"
	if _, err := mismatched.Resolve(lib); err == nil {
		t.Fatal("a character was resolved carrying a water skill without water")
	} else if !strings.Contains(err.Error(), "riptide") {
		t.Errorf("the rejection is %q, want it to name riptide", err)
	}
}

// TestFillWithoutATerminalTakesEveryDefault is the promise the wizard makes:
// it asks only for what is still missing, and a preset-supplied value is not
// missing. Without this, the only way through a scripted run was to pipe the
// right number of blank lines and hope.
func TestFillWithoutATerminalTakesEveryDefault(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	unattended := func() *prompter {
		return &prompter{in: bufio.NewReader(strings.NewReader("")), out: io.Discard, interactive: false}
	}

	// Nothing given: the first field with no default is named, by its flag.
	if _, err := fill(forge.Draft{}, lib, unattended()); err == nil {
		t.Fatal("a draft with nothing in it was accepted without a terminal")
	} else if !strings.Contains(err.Error(), "--id") {
		t.Errorf("the error is %q, want it to name --id", err)
	}

	// Every field with no default is required, and each one says so by name.
	full := forge.Draft{
		ID: "example-film.tester", Name: "Tester", Origin: "example-film",
		Archetype: "duelist", Element: "wind/ground",
	}
	for _, missing := range []struct {
		flag  string
		clear func(*forge.Draft)
	}{
		{"name", func(d *forge.Draft) { d.Name = "" }},
		{"origin", func(d *forge.Draft) { d.Origin = "" }},
		{"archetype", func(d *forge.Draft) { d.Archetype = "" }},
		{"element", func(d *forge.Draft) { d.Element = "" }},
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
	character, err := filled.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	preset, _ := lib.Archetypes().Get("duelist")
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
		t.Errorf("the kit is %v, want the preset's %v", character.Skills, preset.Skills)
	}
	if character.Stages[0].Stats != preset.Stats {
		t.Error("the curves are not the preset's")
	}
	if character.Image != forge.SuggestedImage(full.ID) {
		t.Errorf("the art is %q, want the suggested default %q", character.Image, forge.SuggestedImage(full.ID))
	}
	if character.Bio != "" {
		t.Errorf("the biography is %q, want it left empty", character.Bio)
	}

	// A bad flag is reported rather than re-asked, and names the flag.
	broken := full
	broken.Element = "plasma"
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
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Answered up to the element, then the input simply stops.
	script := "example-film.tester\nTester\nexample-film\nduelist\n\n\nwind/ground\n"
	prompt := &prompter{in: bufio.NewReader(strings.NewReader(script)), out: io.Discard, interactive: true}
	filled, err := fill(forge.Draft{}, lib, prompt)
	if err != nil {
		t.Fatalf("fill stopped when the input ended: %v", err)
	}
	if prompt.interactive {
		t.Error("the prompter still thinks somebody is answering")
	}
	character, err := filled.Resolve(lib)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	preset, _ := lib.Archetypes().Get("duelist")
	if character.Stages[0].Stats != preset.Stats {
		t.Error("the curves after the input ended are not the preset's")
	}
	if !reflect.DeepEqual(character.Skills, preset.Skills) {
		t.Errorf("the kit after the input ended is %v", character.Skills)
	}

	// And when the input stops before a field that has no default, the error
	// still names the flag rather than reporting a bare EOF.
	short := &prompter{in: bufio.NewReader(strings.NewReader("example-film.tester\n")), out: io.Discard, interactive: true}
	if _, err := fill(forge.Draft{}, lib, short); err == nil {
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

	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	written, known := lib.Characters().Get("example-anime.piped")
	if !known {
		t.Fatal("the character was not written")
	}
	preset, _ := lib.Archetypes().Get("sentinel")
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

// TestRenderReportSurvivesAMissingRow is the one part of check that stayed in
// this front-end: forge.Inspect finds the problem, and these columns draw it.
func TestRenderReportSurvivesAMissingRow(t *testing.T) {
	dir := scratchData(t)
	removed := filepath.Join(dir, "assets", "example", "sprout.svg")
	if err := os.Remove(removed); err != nil {
		t.Fatalf("remove %s: %v", removed, err)
	}
	report, err := forge.Inspect(dir)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	var drawn strings.Builder
	renderReport(&drawn, report)
	// The state is named, not merely coloured or left blank.
	if !strings.Contains(drawn.String(), "MISSING") {
		t.Errorf("the row with no art does not say so:\n%s", drawn.String())
	}
	if !strings.Contains(drawn.String(), "sprout.svg") {
		t.Errorf("the problem list does not name the file:\n%s", drawn.String())
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

// TestSkillsAddRunsEndToEndThroughAPipe is the authoring flow a script uses,
// from argv to the file and back.
//
// It matters more here than for a character, because a skill is balance: the
// figures a run reports are the ones the golden tables will move to, so a script
// that writes one has to be able to read what it just did.
func TestSkillsAddRunsEndToEndThroughAPipe(t *testing.T) {
	binary := buildHexforge(t)
	dir := scratchData(t)

	run := func(args ...string) (string, error) {
		command := exec.Command(binary, args...)
		command.Stdin = strings.NewReader("")
		output, err := command.CombinedOutput()
		return string(output), err
	}

	output, err := run("skills", "add", "oath", "--data", dir,
		"--power", "1200", "--accuracy", "900", "--cooldown", "2",
		"--applies", "burn:500", "--restrict-elements", "fire,metal", "--yes")
	if err != nil {
		t.Fatalf("a flags-only run through a pipe failed: %v\n%s", err, output)
	}
	for _, want := range []string{
		// Every default a skill takes, and the two figures it does not.
		"oath — neutral, enemy", "range 1, single", "1200 x1, accuracy 900, cooldown 2",
		"burn:500", "kept for fire or metal",
		// The damage against the reference pair skills.golden is measured from,
		// which is the number this subcommand exists to show before a write.
		"411 per strike", "800 attack and 400 defence",
		"wrote oath", "make golden",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("the run does not report %q:\n%s", want, output)
		}
	}

	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	written, err := lib.Skills().Lookup("oath")
	if err != nil {
		t.Fatalf("the skill was not written: %v", err)
	}
	if written.Element != element.Neutral || written.Power != 1200 || written.Cooldown != 2 {
		t.Errorf("the written skill is %+v", written)
	}
	if got := forge.WhoMaySummary(written); got != "kept for fire or metal" {
		t.Errorf("the written restriction reads %q", got)
	}
	// Written at the end of the book rather than sorted into it, because a skill
	// book's order is authored: skills.golden's table is that order.
	if last := lib.Skills().Skills()[len(lib.Skills().Skills())-1]; last.ID != "oath" {
		t.Errorf("the new skill landed at %q rather than the end of the book", last.ID)
	}

	// A field with no default fails naming its flag, instead of writing a
	// balance number nobody chose.
	output, err = run("skills", "add", "powerless", "--data", dir, "--accuracy", "900", "--yes")
	if err == nil {
		t.Fatalf("a run with no --power succeeded:\n%s", output)
	}
	if !strings.Contains(output, "--power") {
		t.Errorf("the failure does not name --power:\n%s", output)
	}

	// An id already in the book is refused in the words internal/forge owns,
	// rather than by the parser complaining about a file listing one id twice.
	output, err = run("skills", "add", "strike", "--data", dir,
		"--power", "1000", "--accuracy", "900", "--yes")
	if err == nil {
		t.Fatalf("an id already in the book was written:\n%s", output)
	}
	if !strings.Contains(output, "already in the book") {
		t.Errorf("the failure does not say the id is taken:\n%s", output)
	}

	// An allowlist naming somebody who does not exist is refused up front. The
	// skill book cannot make that check — cast imports skill, so the reverse
	// would be an import cycle — and letting it be written would defer the
	// refusal to the first character that tried to carry it.
	output, err = run("skills", "add", "phantom", "--data", dir,
		"--power", "1000", "--accuracy", "900", "--restrict-characters", "nobody.here", "--yes")
	if err == nil {
		t.Fatalf("a skill kept for a character nobody has authored was written:\n%s", output)
	}
	if !strings.Contains(output, "nobody.here") {
		t.Errorf("the failure does not name the character:\n%s", output)
	}

	// And the whole book still loads, which is the property the write rests on.
	if _, err := run("skills", "--data", dir); err != nil {
		t.Errorf("the book does not list after a write: %v", err)
	}
}
