package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// # What this tool may do without a data directory, and how that stays decided
//
// `forge.DefaultDataDir` is a **relative** path, so an installed hexforge run
// anywhere but a checkout used to die on
// `read internal/seed/data/combat.json: no such file or directory` — all
// fourteen subcommands, on a binary carrying every one of those files. The game
// client was fixed first (→ forge.LoadForReading); hexforge is the harder half,
// because it writes as well, and `go:embed` is read-only.
//
// So the answer is per **code path**. `origins` lists and `origins add` appends,
// under one subcommand name, dispatched by a single `args[0] == "add"`. A table
// keyed on the subcommand name would have to give both paths one answer and
// would be wrong whichever it chose.
//
// The walk below is what makes that survive a fifteenth subcommand: it parses
// this package, finds every function that registers `--data`, and holds that set
// **equal** to the decision table. A path with no row is red, a row with no path
// is red, and a row whose declared decision disagrees with the helper the source
// really calls is red. Without it, a subcommand added later silently gets an
// embedded library and its writes go nowhere — or a writer is quietly allowed
// one and refuses eleven prompts later, in forge.ErrNoDataDirectory's words
// rather than in words that name the way out.

// dataDecision is one code path's declared answer to having no data directory,
// and the run that measures it.
type dataDecision struct {
	// writes is the declaration held against the source: a writing path must
	// call loadForWriting and a reading one loadForReading, and the walk checks
	// which it really calls rather than trusting this field.
	writes bool
	// why is the sentence the row exists for, logged so the table reads as a
	// decision record rather than as a list of names.
	why string
	// argv is a whole hexforge command line, without --data, cheap enough to run
	// several times. It must be one that reaches the load: a run refused for a
	// missing operand measures the flag parser and not this.
	argv []string
	// wants are markers the reading run's output must carry. They are things
	// only the books can supply, so their presence is the evidence the embedded
	// copy really answered — "no error" alone is what a program that printed an
	// empty table would also give.
	wants []string
}

// dataDecisions is every code path in this package that reaches a data
// directory, and what each one does when there is none.
//
// It is held bijective with the source by the walk below, so this is a record
// rather than a sample.
var dataDecisions = map[string]dataDecision{
	"loadForListing": {
		why: "the bare listings only print; every one of them is worth having " +
			"from a clean install and none can write",
		argv:  []string{"origins"},
		wants: []string{"pokemon", "works, media:"},
	},
	"runShow": {
		why:   "resolving one character and pricing it reads and writes nothing",
		argv:  []string{"show", "pokemon.abra"},
		wants: []string{"pokemon.abra", "Abra"},
	},
	"runCheck": {
		why: "parsing the books is a read; the art half is dropped rather than " +
			"failed, and the report says so",
		argv: []string{"check"},
		// The closing note is the assertion that matters here: a clean check with
		// no directory is a narrower claim than a clean check over one, and a
		// reader not told that reads it as the wider one.
		wants: []string{
			"checked " + embeddedBooks,
			"no problems found",
			"The art is NOT embedded",
		},
	},
	"runSpar": {
		why:   "a duel is fought in memory and reported to the screen",
		argv:  []string{"spar", "pokemon.abra", "--seeds", "1"},
		wants: []string{"pokemon.abra", "seeds"},
	},
	"runCensus": {
		why:   "counting a build's own casts is fought in memory too",
		argv:  []string{"census", "bulbasaur.poison", "--seeds", "1"},
		wants: []string{"bulbasaur.poison", "razor_leaf"},
	},
	"runWeigh": {
		why: "pricing a field fights a copy of the carrier; nothing is written",
		// ⚠️ The values are chosen to leave the row unsaturated — a saturated row
		// is refused before anything is drawn, and the refusal would still name
		// the skill, so a weaker assertion could not tell the two apart. If a
		// balance change saturates this pair, pick another; the claim being made
		// is about the data directory, not about ember.
		argv: []string{"weigh", "pokemon.charmander", "ember",
			"--field", "power", "--values", "60,70", "--seeds", "25"},
		wants: []string{"weighing ember power", "(control)"},
	},
	"runOriginsAdd": {
		writes: true,
		why:    "it appends to origins.json",
		argv:   []string{"origins", "add", "fixture.work", "--title", "T", "--medium", "anime"},
	},
	"runSpeciesAdd": {
		writes: true,
		why:    "it appends to species.json",
		argv:   []string{"species", "add", "construct", "--name", "Construct"},
	},
	"runSkillsAdd": {
		writes: true,
		why:    "it appends to skills.json",
		argv:   []string{"skills", "add", "oath", "--power", "1200", "--accuracy", "900"},
	},
	"runSkillsEdit": {
		writes: true,
		why:    "it rewrites an entry in skills.json",
		argv:   []string{"skills", "edit", "ember", "--power", "1100"},
	},
	"runNew": {
		writes: true,
		why:    "it appends to cast.json",
		argv: []string{"new", "pokemon.someone", "--name", "Someone", "--origin", "pokemon",
			"--archetype", "sentinel", "--element", "water/ice", "--yes"},
	},
}

// listingSubcommands is every subcommand name that goes through loadForListing,
// mapped to the function that dispatches it.
//
// loadForListing is one function and therefore one row above, but it is eight
// subcommands, and a ninth would arrive with no run of its own. The walk holds
// this map equal to the callers of loadForListing for the same reason it holds
// dataDecisions equal to the callers of dataFlag.
var listingSubcommands = map[string]string{
	"origins":    "runOrigins",
	"species":    "runSpecies",
	"statuses":   "runStatuses",
	"archetypes": "runArchetypes",
	"passives":   "runPassives",
	"skills":     "runSkills",
	"cast":       "runCast",
	"builds":     "runBuilds",
}

// loadHelpers are the two functions allowed to call into forge for a library.
//
// Everything else goes through one of them, which is what makes the reads/writes
// declaration above checkable from the source at all: if a subcommand could call
// forge.Load itself, the walk could see that it reaches a directory and never
// see which of the two answers it gives.
var loadHelpers = map[string]bool{"loadForReading": true, "loadForWriting": true}

// TestEveryHexforgeCodePathThatReachesTheDataDirectorySaysWhetherItReadsOrWrites
// is the guard the whole change rests on.
//
// It is two tests wearing one name, deliberately, because either half alone is
// the shape that passes while the defect ships:
//
//   - The **walk** parses this package and holds dataDecisions equal, both ways,
//     to the set of functions that register --data. A table asserting a handful
//     of subcommands is exactly what lets the fifteenth one leak, and nothing
//     about adding a subcommand makes anybody open this file — so the source is
//     what decides which cases exist. It also reads which load helper each one
//     calls, so a row that *claims* to read while calling loadForWriting is red.
//   - The **runs** put every row through the real binary, standing in a directory
//     with nothing under it. A walk on its own proves only that somebody wrote a
//     row down.
//
// ⚠️ It logs its case count and refuses to run on none. A walk whose predicate
// has stopped matching agrees with every claim there is.
//
// *Sees:* a new subcommand with no declared decision; a stale row; a writer
// handed the embedded copy; a reader denied it; a refusal that does not name the
// way out; a load that skipped both helpers.
// *Cannot see:* whether the sentence a row prints is the *right* sentence — the
// wants are markers, and the wording tests below are what hold the refusals.
func TestEveryHexforgeCodePathThatReachesTheDataDirectorySaysWhetherItReadsOrWrites(t *testing.T) {
	reaches, callsHelper, callsForge, callsListing, scanned := walkForTheDataFlag(t)
	if scanned == 0 {
		t.Fatal("the walk read no source files, so it measures nothing")
	}
	if len(reaches) == 0 {
		t.Fatalf("the walk read %d files and found nothing registering --data, and "+
			"eleven functions do: it is measuring nothing", scanned)
	}
	reading, writing := 0, 0
	for _, decided := range dataDecisions {
		if decided.writes {
			writing++
			continue
		}
		reading++
	}
	t.Logf("%d source files, %d code paths reaching the data directory, "+
		"%d declared reading and %d declared writing, %d listing subcommands",
		scanned, len(reaches), reading, writing, len(listingSubcommands))

	for _, name := range slices.Sorted(maps.Keys(reaches)) {
		decided, hasDecision := dataDecisions[name]
		if !hasDecision {
			t.Errorf("%s (%s) registers --data and no decision is written down for it. "+
				"With no directory it either has to answer from the embedded books or "+
				"refuse in words naming the way out — add a row to dataDecisions saying "+
				"which, and call loadForReading or loadForWriting accordingly",
				name, reaches[name])
			continue
		}
		want := "loadForReading"
		if decided.writes {
			want = "loadForWriting"
		}
		if got := callsHelper[name]; got != want {
			t.Errorf("dataDecisions says %s %s, so it should call %s; the source calls %q",
				name, map[bool]string{true: "writes", false: "only reads"}[decided.writes],
				want, got)
			continue
		}
		t.Logf("  %-16s %-14s %s", name, want, decided.why)
	}
	for _, name := range slices.Sorted(maps.Keys(dataDecisions)) {
		if _, reaches := reaches[name]; !reaches {
			t.Errorf("dataDecisions has a row for %s and nothing of that name registers "+
				"--data any more: the row is stale and its run measures a code path that "+
				"has moved on", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(callsForge)) {
		if loadHelpers[name] {
			continue
		}
		t.Errorf("%s calls %s directly, going round loadForReading and "+
			"loadForWriting. That is the one way a code path can reach the data "+
			"directory without declaring what it does there", name, callsForge[name])
	}
	// The listing helper is one row and eight subcommands, so its callers get the
	// same bijection: a ninth listing arrives with a name in this map or red.
	dispatchers := slices.Collect(maps.Values(listingSubcommands))
	for _, name := range slices.Sorted(maps.Keys(callsListing)) {
		if !slices.Contains(dispatchers, name) {
			t.Errorf("%s calls loadForListing and listingSubcommands does not name it: "+
				"add the subcommand it serves, so it is run from a clean directory too",
				name)
		}
	}
	for subcommand, function := range listingSubcommands {
		if _, dispatches := commands[subcommand]; !dispatches {
			t.Errorf("listingSubcommands names %q and the dispatch table has no such "+
				"subcommand", subcommand)
		}
		if _, calls := callsListing[function]; !calls {
			t.Errorf("listingSubcommands says %s serves %q and it no longer calls "+
				"loadForListing", function, subcommand)
		}
	}

	binary := buildHexforge(t)
	for _, name := range slices.Sorted(maps.Keys(dataDecisions)) {
		decided := dataDecisions[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if decided.writes {
				refusesWithNowhereToWrite(t, binary, decided)
				return
			}
			answersFromTheEmbeddedBooks(t, binary, decided)
		})
	}
}

// answersFromTheEmbeddedBooks is a reading path's whole obligation: it works from
// a clean install, and a directory somebody *named* is still never replaced.
func answersFromTheEmbeddedBooks(t *testing.T, binary string, decided dataDecision) {
	t.Helper()
	if len(decided.wants) == 0 {
		t.Fatal("a reading row with nothing to look for asserts only that the run " +
			"did not error, which a program printing an empty table also passes")
	}
	output, err := runFromNowhere(t, binary, decided.argv...)
	if err != nil {
		t.Fatalf("`hexforge %s` failed with no data directory reachable: %v\n%s",
			strings.Join(decided.argv, " "), err, output)
	}
	for _, want := range decided.wants {
		if !strings.Contains(output, want) {
			t.Errorf("the run said nothing about %q, so it may not have read the books "+
				"at all:\n%s", want, output)
		}
	}

	// ⚠️ A named directory is never second-guessed. Handing back different data
	// is not an answer to what was asked, and the refusal has to name the way out
	// rather than the first book it failed to open.
	named := append(slices.Clone(decided.argv), "--data", filepath.Join(t.TempDir(), "nope"))
	refused, err := runFromNowhere(t, binary, named...)
	if err == nil {
		t.Fatalf("`hexforge %s` succeeded against a --data that is not there:\n%s",
			strings.Join(named, " "), refused)
	}
	namesTheWayOut(t, refused, "--data")
	if !strings.Contains(refused, "leave --data off") {
		t.Errorf("the refusal does not say the flag can be dropped, which is the one "+
			"thing a reader can do that a writer cannot:\n%s", refused)
	}
}

// refusesWithNowhereToWrite is a writing path's whole obligation.
func refusesWithNowhereToWrite(t *testing.T, binary string, decided dataDecision) {
	t.Helper()
	output, err := runFromNowhere(t, binary, decided.argv...)
	if err == nil {
		t.Fatalf("`hexforge %s` succeeded with no data directory: it either wrote "+
			"somewhere nobody asked for, or it did nothing and said it had:\n%s",
			strings.Join(decided.argv, " "), output)
	}
	if strings.Contains(output, "no such file or directory") {
		t.Errorf("the refusal is the bare read error, which names a book nobody typed "+
			"and offers nothing to do:\n%s", output)
	}
	namesTheWayOut(t, output, "--data", "checkout")
}

// namesTheWayOut is the assertion a refusal owes beyond being a refusal: it says
// what to do, not only what failed.
func namesTheWayOut(t *testing.T, output string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Errorf("the refusal does not mention %q, so it says what failed and not "+
				"what to do about it:\n%s", want, output)
		}
	}
}

// runFromNowhere runs the binary standing in an empty directory, which is what a
// clean `go install` leaves somebody in.
//
// The working directory is the whole fixture: forge.DefaultDataDir is relative,
// so a scratch directory with nothing under it is the only thing needed to make
// it name nothing. Stdin is a real pipe with nothing in it, because every writing
// path here would otherwise prompt — and a prompt that reads EOF fails for the
// wrong reason, which is exactly the confusion these refusals exist to remove.
func runFromNowhere(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = t.TempDir()
	command.Stdin = strings.NewReader("")
	output, err := command.CombinedOutput()
	return string(output), err
}

// walkForTheDataFlag parses this package and reports, per function: which ones
// register --data, which load helper each calls, which call into forge for a
// library directly, which call loadForListing, and how many files it read.
//
// It walks this package's own directory because these are unexported functions
// in package main — nothing outside can reach them, which is the property that
// makes the walk exhaustive rather than a sample. Test files are skipped: a test
// registering a flag is not a code path a user can reach.
//
// ⚠️ The predicate is **registering the flag**, not calling forge. That is the
// wider net of the two and deliberately so: a subcommand that takes --data has
// declared it touches a data directory, whatever it then does with the string,
// and a subcommand that reached one *without* taking --data would be a worse
// defect than the one this test was written for.
func walkForTheDataFlag(t *testing.T) (reaches, callsHelper, callsForge, callsListing map[string]string, scanned int) {
	t.Helper()
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list this package's sources: %v", err)
	}
	reaches = map[string]string{}
	callsHelper = map[string]string{}
	callsForge = map[string]string{}
	callsListing = map[string]string{}
	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declared := range file.Decls {
			function, isFunction := declared.(*ast.FuncDecl)
			if !isFunction || function.Body == nil {
				continue
			}
			name := function.Name.Name
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, isCall := node.(*ast.CallExpr)
				if !isCall {
					return true
				}
				switch called := call.Fun.(type) {
				case *ast.Ident:
					switch called.Name {
					case "dataFlag":
						reaches[name] = path
					case "loadForReading", "loadForWriting":
						callsHelper[name] = called.Name
					case "loadForListing":
						callsListing[name] = path
					}
				case *ast.SelectorExpr:
					// forge.Load, forge.LoadEmbedded, forge.LoadForReading and
					// forge.Inspect are the four ways to a library, and Inspect is
					// the one the original survey of this package missed: it is not
					// spelled Load and it reaches a directory all the same.
					pkg, isPkg := called.X.(*ast.Ident)
					if !isPkg || pkg.Name != "forge" {
						return true
					}
					switch called.Sel.Name {
					case "Load", "LoadEmbedded", "LoadForReading", "Inspect":
						callsForge[name] = "forge." + called.Sel.Name + " in " + path
					}
				}
				return true
			})
		}
	}
	return reaches, callsHelper, callsForge, callsListing, scanned
}

// TestEveryListingSubcommandListsFromACleanInstall is the eight names behind the
// one loadForListing row, run for real.
//
// The walk above holds the map equal to the source; this holds each name to
// printing something. They are one function and eight flag sets, and a listing
// that takes an operand it should not, or that renders off a book that is not
// loaded, would pass the walk and fail here.
func TestEveryListingSubcommandListsFromACleanInstall(t *testing.T) {
	binary := buildHexforge(t)
	names := slices.Sorted(maps.Keys(listingSubcommands))
	t.Logf("%d listing subcommands: %s", len(names), strings.Join(names, " "))
	if len(names) == 0 {
		t.Fatal("no listing subcommands, so this test measures nothing")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			output, err := runFromNowhere(t, binary, name)
			if err != nil {
				t.Fatalf("`hexforge %s` failed with no data directory: %v\n%s", name, err, output)
			}
			// A listing that found nothing prints its "add one with" line, which is
			// a real answer and not what an unloaded book gives — so the assertion
			// is that *something* came out, and the per-listing content is held by
			// the row above and by the rendering tests.
			if strings.TrimSpace(output) == "" {
				t.Errorf("`hexforge %s` printed nothing at all", name)
			}
			if strings.Contains(output, "no such file or directory") {
				t.Errorf("`hexforge %s` still reports a missing file:\n%s", name, output)
			}
		})
	}
}

// TestANamedDataDirectoryThatIsBrokenIsRefusedRatherThanQuietlyReplaced is the
// trap the reading rule is really about.
//
// The two-line version of the rule — "load, and take the embedded copy if that
// returned an error" — passes every clean-install test in this file and quietly
// changes what an author's own trailing comma means: instead of a refusal naming
// the book that will not parse, hexforge lists the baked-in books and says
// nothing, and the author spends the evening looking for an edit they can see in
// the file. So the fallback keys on the directory being **absent**, never on the
// load failing, and this is what measures that.
func TestANamedDataDirectoryThatIsBrokenIsRefusedRatherThanQuietlyReplaced(t *testing.T) {
	binary := buildHexforge(t)

	// Named, so the first line of the rule applies and there was never a question.
	broken := scratchData(t)
	breakABook(t, broken)
	output, err := runFromNowhere(t, binary, "origins", "--data", broken)
	if err == nil {
		t.Fatalf("a named directory with an unparseable book was replaced by the "+
			"embedded copy:\n%s", output)
	}
	brokenBook(t, output)

	// ⚠️ The harder half: **not** named, and the directory is there. The default
	// is relative, so the fixture is a working directory with a broken
	// `internal/seed/data` under it — an author standing in their own checkout.
	// Nothing distinguishes this run from the clean-install one except that the
	// stat succeeds, which is the whole of the rule; keyed on the load failing
	// instead, this run would list the baked-in works and exit nought.
	root := t.TempDir()
	nested := filepath.Join(root, "internal", "seed", "data")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("build a checkout-shaped scratch directory: %v", err)
	}
	copyTree(t, shippedDataDir, nested)
	breakABook(t, nested)
	command := exec.Command(binary, "origins")
	command.Dir = root
	command.Stdin = strings.NewReader("")
	unnamed, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("a data directory that is there and will not parse was quietly "+
			"replaced by the embedded copy:\n%s", unnamed)
	}
	brokenBook(t, string(unnamed))
}

// brokenBook is the refusal a data directory that will not parse produces.
//
// ⚠️ It matches the **parse** wording and not the file name, and that is a fact
// about internal/forge rather than a looser assertion: a read error is wrapped
// with the path it failed on, a decode error is not, so `combat.json` does not
// appear anywhere in it. Matching the path would make this test pass only while
// the load is failing for the wrong reason.
func brokenBook(t *testing.T, output string) {
	t.Helper()
	if !strings.Contains(output, "decode combat rules") {
		t.Errorf("the refusal is not the parse failure the fixture arranged:\n%s", output)
	}
}

// breakABook makes one of a data directory's files unparseable, which is the
// state a trailing comma leaves an author in.
func breakABook(t *testing.T, dir string) {
	t.Helper()
	combat := filepath.Join(dir, "combat.json")
	if err := os.WriteFile(combat, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("break %s: %v", combat, err)
	}
}

// TestACheckWithNoDirectoryMakesTheNarrowerClaimInWords is the honesty half of
// letting check run from a clean install.
//
// A library with no directory asks for no art at all — forge.artToCheck returns
// nothing rather than sixty-six missing pictures — so a clean report off the
// embedded books is a **narrower** claim than a clean report over a directory,
// and a reader not told that reads it as the wider one. The two reports are
// drawn side by side here because the assertion is the difference between them:
// asserting only the embedded sentence would pass with both branches printing
// it, which is the mutation that quietly tells an author with a real checkout
// that their art was never looked at.
func TestACheckWithNoDirectoryMakesTheNarrowerClaimInWords(t *testing.T) {
	embedded, err := forge.LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	onDisk, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load %s: %v", shippedDataDir, err)
	}

	var withoutADirectory, withOne strings.Builder
	renderReport(&withoutADirectory, embedded.Inspect())
	renderReport(&withOne, onDisk.Inspect())

	if !strings.Contains(withoutADirectory.String(), "The art is NOT embedded") {
		t.Errorf("the embedded report does not say the art was not looked for:\n%s",
			withoutADirectory.String())
	}
	if strings.Contains(withOne.String(), "The art is NOT embedded") {
		t.Errorf("the report over %s says the art was not looked for, and it was:\n%s",
			shippedDataDir, withOne.String())
	}
	// The header is the other half: Report.Dir is empty for the embedded copy and
	// interpolating it raw gives `checked : 25 origins`, which reads as a bug.
	if !strings.Contains(withoutADirectory.String(), "checked "+embeddedBooks) {
		t.Errorf("the embedded report does not name what it checked:\n%s",
			withoutADirectory.String())
	}
	if !strings.Contains(withOne.String(), "checked "+shippedDataDir) {
		t.Errorf("the report over a directory does not name it:\n%s", withOne.String())
	}
}
