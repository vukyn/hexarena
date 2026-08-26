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

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/testfixture"
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
	// The fixture is what the tests name. Before this they named the characters
	// the repository shipped, so editing the real cast broke tests that had
	// nothing to do with it.
	if err := testfixture.Inject(target, func() (testfixture.Saver, error) {
		return forge.Load(target)
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

// TestFillRePromptsOnABadAnswer is the behaviour that keeps the wizard usable:
// a wrong answer costs one line, not the whole session.
//
// The answers are in prompt order, and two of the orderings are deliberate: the
// kit comes before the element because the kit decides which elements are legal,
// and what the character *is* comes before the kit because a skill kept for a
// lineage is one of the things the kit check reads.
func TestFillRePromptsOnABadAnswer(t *testing.T) {
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	script := strings.Join([]string{
		"Example.Bad",          // rejected: not a slug
		"fixture-film.tester",  // accepted
		"Tester",               // name
		"nowhere",              // rejected: unknown origin
		"fixture-film",         // accepted
		"berserker",            // rejected: unknown archetype
		"duelist",              // accepted
		"",                     // art: take the suggested default
		"dragoon",              // rejected: not a species in the catalog
		"lizard",               // accepted: what it is
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
	if character.ID != "fixture-film.tester" || character.Name != "Tester" {
		t.Errorf("collected %s / %s", character.ID, character.Name)
	}
	if character.Element.String() != "wind/ground" {
		t.Errorf("the element is %s, want the answer that could carry the kit", character.Element)
	}
	if character.Image != forge.SuggestedImage("fixture-film.tester") {
		t.Errorf("the art is %q, want the suggested default %q",
			character.Image, forge.SuggestedImage("fixture-film.tester"))
	}
	if character.Bio != "A tester." {
		t.Errorf("the biography is %q", character.Bio)
	}
	if !reflect.DeepEqual(character.Species, []string{"lizard"}) {
		t.Errorf("what it is came out as %v, want the answer that was accepted", character.Species)
	}
	preset, _ := lib.Archetypes().Get("duelist")
	if character.Stages[0].Stats != preset.Stats {
		t.Error("pressing Enter through the curves did not take the preset")
	}
	if !reflect.DeepEqual(cast.LearnedIDs(character.Skills), preset.Skills) {
		t.Errorf("pressing Enter through the kit gave %v", character.Skills)
	}
}

// TestElementPromptFollowsTheKitNotThePreset is why the kit is asked first.
//
// A substituted kit demands something different from the preset's, and it is the
// substituted kit that the write will be checked against, so it has to be the
// one the prompt checks too.
func TestElementPromptFollowsTheKitNotThePreset(t *testing.T) {
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// The duelist preset demands wind. Substituting a water kit moves the
	// demand to water, so "wind" must now be refused and "water" accepted.
	given := forge.Draft{
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
		Archetype: "duelist", Image: "assets/fixture/tester.svg",
		Skills: "strike,riptide",
	}
	script := strings.Join([]string{
		"",            // species: nothing in particular
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
	lib, err := forge.Load(scratchData(t))
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
		ID: "fixture-film.tester", Name: "Tester", Origin: "fixture-film",
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
	if !reflect.DeepEqual(cast.LearnedIDs(character.Skills), preset.Skills) {
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
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Answered up to the element, then the input simply stops. The two bare
	// newlines in the middle are the art and the species; the third is the kit.
	script := "fixture-film.tester\nTester\nfixture-film\nduelist\n\n\n\nwind/ground\n"
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
	if !reflect.DeepEqual(cast.LearnedIDs(character.Skills), preset.Skills) {
		t.Errorf("the kit after the input ended is %v", character.Skills)
	}

	// And when the input stops before a field that has no default, the error
	// still names the flag rather than reporting a bare EOF.
	short := &prompter{in: bufio.NewReader(strings.NewReader("fixture-film.tester\n")), out: io.Discard, interactive: true}
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
		"--id", "fixture-anime.piped", "--name", "Piped",
		"--origin", "fixture-anime", "--archetype", "sentinel",
		"--element", "water/ice", "--image", "assets/fixture/piped.svg", "--yes")
	if err != nil {
		t.Fatalf("a flags-only run through a pipe failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "wrote fixture-anime.piped") {
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
	written, known := lib.Characters().Get("fixture-anime.piped")
	if !known {
		t.Fatal("the character was not written")
	}
	preset, _ := lib.Archetypes().Get("sentinel")
	if !reflect.DeepEqual(cast.LearnedIDs(written.Skills), preset.Skills) {
		t.Errorf("the written kit is %v, want the preset's %v", written.Skills, preset.Skills)
	}
	if written.Stages[0].Stats != preset.Stats {
		t.Error("the written curves are not the preset's")
	}

	// A field with no default fails, naming its flag, instead of hanging or
	// reporting an EOF nobody can act on.
	output, err = run("new", "--data", dir,
		"--id", "fixture-anime.nameless", "--origin", "fixture-anime",
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
		"--id", "fixture-anime.mismatch", "--name", "Mismatch",
		"--origin", "fixture-anime", "--archetype", "sentinel",
		"--element", "fire", "--image", "assets/fixture/adept.svg", "--yes")
	if err == nil {
		t.Fatalf("a fire character carrying the sentinel kit was written:\n%s", output)
	}
	if !strings.Contains(output, "riptide") {
		t.Errorf("the failure does not name the skill it cannot carry:\n%s", output)
	}

	// Without --yes there is nobody to confirm to, so it says so rather than
	// writing on somebody's behalf.
	output, err = run("new", "--data", dir,
		"--id", "fixture-anime.unconfirmed", "--name", "Unconfirmed",
		"--origin", "fixture-anime", "--archetype", "sentinel",
		"--element", "water/ice", "--image", "assets/fixture/adept.svg")
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
	removed := filepath.Join(dir, "assets", "fixture", "sprout.svg")
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
		"oath — neutral, enemy", "range 1, single", "1200 (120%) x1, accuracy 900 (90%), cooldown 2",
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

// TestSkillsEditRunsEndToEndThroughAPipe is the other half of
// TestSkillsAddRunsEndToEndThroughAPipe, and every case below is a difference
// between the two rather than a repeat.
//
// The one that matters most is the absent flag against the explicit zero: on
// `add` an absent flag is a question or a default, and here it means leave the
// field alone, so the two have to be told apart by which flags were really given
// rather than by comparing a value against "". Both directions are asserted,
// because getting it wrong in either is silent — a cleared cooldown that did not
// clear, or a cooldown nobody mentioned that went to zero.
func TestSkillsEditRunsEndToEndThroughAPipe(t *testing.T) {
	binary := buildHexforge(t)
	dir := scratchData(t)

	run := func(args ...string) (string, error) {
		command := exec.Command(binary, args...)
		command.Stdin = strings.NewReader("")
		output, err := command.CombinedOutput()
		return string(output), err
	}
	held := func(id string) skill.Skill {
		t.Helper()
		lib, err := forge.Load(dir)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		found, err := lib.Skills().Lookup(id)
		if err != nil {
			t.Fatalf("look up %s: %v", id, err)
		}
		return found
	}

	// riptide ships at 900 power over two strikes with a cooldown of three, and
	// nothing below relies on those numbers except as the values an edit that
	// does not name them has to leave alone.
	before := held("riptide")
	if before.Cooldown == 0 || before.Accuracy == 0 {
		t.Fatalf("riptide ships as %+v, which makes the zero cases meaningless", before)
	}

	// An explicit zero lands.
	output, err := run("skills", "edit", "riptide", "--data", dir, "--cooldown", "0", "--yes")
	if err != nil {
		t.Fatalf("an explicit zero was refused: %v\n%s", err, output)
	}
	for _, want := range []string{"edited riptide", "make golden"} {
		if !strings.Contains(output, want) {
			t.Errorf("the run does not report %q:\n%s", want, output)
		}
	}
	cleared := held("riptide")
	if cleared.Cooldown != 0 {
		t.Errorf("--cooldown 0 left the cooldown at %d", cleared.Cooldown)
	}
	// And every field the edit did not name is untouched, which is the other half
	// of the same rule.
	expected := before
	expected.Cooldown = 0
	if !reflect.DeepEqual(cleared, expected) {
		t.Errorf("--cooldown 0 changed more than the cooldown:\n%+v\n%+v", cleared, expected)
	}

	// A power change reports the before and the after, because a skill is balance
	// and what moved is the point of editing through the tool.
	output, err = run("skills", "edit", "riptide", "--data", dir, "--power", "1100", "--yes")
	if err != nil {
		t.Fatalf("a power edit failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "damage 308 → 377 per strike") {
		t.Errorf("the run does not show the damage before and after:\n%s", output)
	}
	if raised := held("riptide"); raised.Power != 1100 || raised.Cooldown != 0 {
		t.Errorf("the second edit produced %+v", raised)
	}

	// An explicitly empty list clears a restriction, which is the only way this
	// front-end can widen a skill again. The skill is one nobody carries, so
	// narrowing it in the first place is legal.
	if _, err := run("skills", "add", "oath", "--data", dir,
		"--power", "1000", "--accuracy", "900", "--restrict-elements", "fire,metal", "--yes"); err != nil {
		t.Fatalf("author a restricted skill: %v", err)
	}
	if got := forge.WhoMaySummary(held("oath")); got != "kept for fire or metal" {
		t.Fatalf("the authored restriction reads %q", got)
	}
	output, err = run("skills", "edit", "oath", "--data", dir, "--restrict-elements", "", "--yes")
	if err != nil {
		t.Fatalf("clearing a list was refused: %v\n%s", err, output)
	}
	if got := forge.WhoMaySummary(held("oath")); got != "anyone" {
		t.Errorf("the cleared skill reads %q, want the common pool", got)
	}

	// An edit naming nothing is refused rather than rewriting the file with the
	// same bytes, and the refusal shows what naming something looks like.
	output, err = run("skills", "edit", "oath", "--data", dir, "--yes")
	if err == nil {
		t.Fatalf("an edit naming no field was accepted:\n%s", output)
	}
	if !strings.Contains(output, "--power") {
		t.Errorf("the refusal does not show how to name a field:\n%s", output)
	}

	// The id is not editable, and the refusal says a rename is its own job.
	output, err = run("skills", "edit", "oath", "--data", dir, "--id", "vow", "--yes")
	if err == nil {
		t.Fatalf("a renamed id was written:\n%s", output)
	}
	if !strings.Contains(output, "separate operation") {
		t.Errorf("the refusal does not say a rename is a separate operation:\n%s", output)
	}

	// An edit that would leave a shipped character unable to carry the skill is
	// refused before anything is written, and it names the character.
	output, err = run("skills", "edit", "riptide", "--data", dir,
		"--restrict-elements", "fire", "--yes")
	if err == nil {
		t.Fatalf("an edit that orphans a character was written:\n%s", output)
	}
	for _, want := range []string{"fixture-anime.adept", "unable to carry it"} {
		if !strings.Contains(output, want) {
			t.Errorf("the refusal does not mention %q:\n%s", want, output)
		}
	}
	if refused := held("riptide"); refused.Restrict != nil {
		t.Errorf("the refused edit was written anyway: %+v", refused.Restrict)
	}

	// The same for an archetype preset, which is a different rule in a different
	// place: a preset is shared by every character built from it.
	output, err = run("skills", "edit", "sever", "--data", dir,
		"--restrict-characters", "fixture-anime.adept", "--yes")
	if err == nil {
		t.Fatalf("an edit that orphans a preset was written:\n%s", output)
	}
	for _, want := range []string{"bulwark preset", "unable to carry it"} {
		if !strings.Contains(output, want) {
			t.Errorf("the refusal does not mention %q:\n%s", want, output)
		}
	}

	// And the whole book still lists, which is the property every refusal above
	// exists to protect.
	if _, err := run("skills", "--data", dir); err != nil {
		t.Errorf("the book does not list after the edits: %v", err)
	}
}

// TestSkillsNameIsAuthoredAndEditedThroughFlags is the command line's half of a
// skill carrying its own display name: it used to be a compiled table in
// internal/i18n, so a script could not set one at all.
//
// It also covers the trap `--cooldown 0` covers for a number: an absent flag and
// an empty one are different answers, and for a name the empty one is how a name
// is taken back off.
// TestEverySkillsEditFlagReachesTheEdit is a bug that shipped, and the shape of
// test that would have stopped it.
//
// The flags were bound to the edit's fields by two maps keyed by the same names,
// and three of them — self-applies, restores and drains — were added to one map
// and not the other. A missing key reads back as a nil answer, which is the exact
// shape "the flag was not given" has, so the command answered "nothing to change"
// and exited zero-effect. It had done that since the day those flags were added,
// and nothing noticed, because every test named a flag that happened to work.
//
// So this names none of them. It reads the flag list out of the command's own
// help, which is the one list that cannot drift from what the command accepts,
// and asserts that every flag it offers does something. A flag whose value is
// rejected downstream still passes: what is being tested is that the answer
// arrived, not that it was any good.
func TestEverySkillsEditFlagReachesTheEdit(t *testing.T) {
	binary := buildHexforge(t)
	dir := scratchData(t)

	help := func() string {
		command := exec.Command(binary, "skills", "edit", "-h")
		output, _ := command.CombinedOutput()
		return string(output)
	}()

	// The two flags that are not fields of the edit: one names the directory and
	// one is the confirmation. Every other flag the command offers has to land.
	notAnEdit := map[string]bool{"data": true, "yes": true}

	flags := make([]string, 0, 20)
	for _, line := range strings.Split(help, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		name := strings.TrimPrefix(strings.Fields(trimmed)[0], "-")
		if !notAnEdit[name] {
			flags = append(flags, name)
		}
	}
	if len(flags) < 15 {
		t.Fatalf("only %d editable flags found in the help, so this is testing nothing:\n%s",
			len(flags), help)
	}

	for _, name := range flags {
		command := exec.Command(binary, "skills", "edit", "riptide",
			"--data", dir, "--"+name, "1", "--yes")
		command.Stdin = strings.NewReader("")
		output, _ := command.CombinedOutput()
		if strings.Contains(string(output), "nothing to change") {
			t.Errorf("--%s was given and the command still says nothing to change:\n%s",
				name, output)
		}
	}
}

func TestSkillsNameIsAuthoredAndEditedThroughFlags(t *testing.T) {
	binary := buildHexforge(t)
	dir := scratchData(t)

	run := func(args ...string) (string, error) {
		command := exec.Command(binary, args...)
		command.Stdin = strings.NewReader("")
		output, err := command.CombinedOutput()
		return string(output), err
	}
	held := func(id string) skill.Skill {
		t.Helper()
		lib, err := forge.Load(dir)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		found, err := lib.Skills().Lookup(id)
		if err != nil {
			t.Fatalf("the book does not hold %s: %v", id, err)
		}
		return found
	}

	output, err := run("skills", "add", "tidal_hymn", "--data", dir,
		"--name", "khúc thủy triều", "--power", "900", "--accuracy", "900", "--yes")
	if err != nil {
		t.Fatalf("authoring a named skill failed: %v\n%s", err, output)
	}
	// The name is in what the run reports, so a script can read back what it set.
	if !strings.Contains(output, "tidal_hymn (khúc thủy triều)") {
		t.Errorf("the run does not report the name it wrote:\n%s", output)
	}
	if got, want := held("tidal_hymn").Name, "khúc thủy triều"; got != want {
		t.Errorf("the written skill is named %q, want %q", got, want)
	}

	// A skill authored without one has no name, which is the default and the
	// state every shipped skill is in.
	if _, err := run("skills", "add", "oath", "--data", dir,
		"--power", "900", "--accuracy", "900", "--yes"); err != nil {
		t.Fatalf("authoring an unnamed skill failed: %v", err)
	}
	if got := held("oath").Name; got != "" {
		t.Errorf("a skill authored with no --name is called %q, want nothing", got)
	}

	// An edit sets one on a skill that shipped without, and the compiled table
	// stops being what that skill reads as. Nothing else about it moves.
	before := held("riptide")
	if before.Name != "" {
		t.Fatalf("riptide already carries a name, so this measures nothing")
	}
	if _, err := run("skills", "edit", "riptide", "--data", dir,
		"--name", "sóng dữ", "--yes"); err != nil {
		t.Fatalf("naming a shipped skill failed: %v", err)
	}
	named := held("riptide")
	if got, want := named.Name, "sóng dữ"; got != want {
		t.Errorf("the edited skill is named %q, want %q", got, want)
	}
	expected := before
	expected.Name = "sóng dữ"
	if !reflect.DeepEqual(named, expected) {
		t.Errorf("--name changed more than the name:\n%+v\n%+v", named, expected)
	}

	// An explicitly empty --name takes it back off, the way an empty allowlist
	// clears a restriction. An absent flag would have left it alone, which is
	// the difference this pair is here to hold.
	if _, err := run("skills", "edit", "riptide", "--data", dir,
		"--power", "1100", "--yes"); err != nil {
		t.Fatalf("an edit naming no name failed: %v", err)
	}
	if got := held("riptide").Name; got != "sóng dữ" {
		t.Errorf("an edit that did not name --name left the name %q", got)
	}
	if _, err := run("skills", "edit", "riptide", "--data", dir, "--name", "", "--yes"); err != nil {
		t.Fatalf("clearing a name was refused: %v", err)
	}
	if got := held("riptide").Name; got != "" {
		t.Errorf("--name \"\" left the name %q, want it cleared", got)
	}
}
