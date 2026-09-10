package forge

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
)

// A library built from the copy the binary embeds, which is what a game client
// wants: nothing to edit, nowhere to write, and the same books every peer on the
// same build is fighting from.
//
// Two claims, and they are different in kind. The first is that the *books* are
// the same however they were read — a parse is a parse. The second is the one
// this whole shape exists for: an embedded library has **no directory**, and
// `filepath.Join("", "cast.json")` is `"cast.json"`, so the sixteen places that
// join a data directory onto a name would have quietly become reads from, and
// writes into, whatever directory the player happened to be standing in. That is
// worse than the refusal it replaced, because a refusal names the problem and a
// relative path finds a stranger's file.

// bookReadings is one row per data file forge.Load reads, and what that file
// became once it was parsed.
//
// A reading rather than a count, because a count is the cheap half: two books
// with the same number of entries and different contents would agree on a count
// and disagree about the game. So each row is compared whole with
// reflect.DeepEqual — a misread field, a truncated list, a file read out of the
// wrong place all move it — and then checked for holding *something*, which is
// the guard the two optional books need. builds.json and squads.json are
// tolerated as absent by the loader (an older data directory, and a directory
// nobody has saved a squad in), so a wrongly rooted embedded filesystem would
// come back not-exist and be read as an author's empty catalogue: green tests
// over a library missing two books.
var bookReadings = []struct {
	file string
	read func(*Library) any
}{
	{combatFile, func(l *Library) any { return l.Rules() }},
	{elementsFile, func(l *Library) any { return l.Chart().Multipliers() }},
	{limitsFile, func(l *Library) any { return l.Limits() }},
	{modifiersFile, func(l *Library) any { return l.Bounds() }},
	{patternsFile, func(l *Library) any { return l.Patterns().Patterns() }},
	{statusesFile, func(l *Library) any { return l.Statuses().Kinds() }},
	{passivesFile, func(l *Library) any { return l.Passives().All() }},
	{skillsFile, func(l *Library) any { return l.Skills().Skills() }},
	{originsFile, func(l *Library) any { return l.Origins().All() }},
	{speciesFile, func(l *Library) any { return l.Species().All() }},
	{archetypesFile, func(l *Library) any { return l.Archetypes().All() }},
	{castFile, func(l *Library) any { return l.Characters().All() }},
	{buildsFile, func(l *Library) any { return l.Builds() }},
	{squadsFile, func(l *Library) any { return l.Squads() }},
	{bonusesFile, func(l *Library) any { return l.Bonuses().All() }},
}

// booksALibraryHolds is how many data files a library is built out of.
//
// It is written down rather than derived so that a sixteenth book arrives with a
// row above: the loader would read it, the digest would cover it, and nothing
// else here would notice it had never been compared. ⚠️ It is **fifteen against
// the sixteen the embed names** — `roster.json` is the placement the game boots
// from rather than a book a character is validated against, and internal/seed
// reads it on its own.
const booksALibraryHolds = 15

// TestTheEmbeddedLibraryHoldsTheSameBooksAsTheDirectoryItWasReadFrom is the
// positive claim: two readings of one set of bytes come to the same books.
//
// *Sees:* a filesystem rooted at the wrong directory, a name list that has
// drifted, a book skipped, a book read out of the wrong file, an optional book
// silently absent.
// *Cannot see:* anything about the *paths* an embedded library hands out — that
// is the test below, and it is the half that carries the risk.
func TestTheEmbeddedLibraryHoldsTheSameBooksAsTheDirectoryItWasReadFrom(t *testing.T) {
	if len(bookReadings) != booksALibraryHolds {
		t.Fatalf("%d readings for %d books: a book with no row is a book this test does "+
			"not compare", len(bookReadings), booksALibraryHolds)
	}
	onDisk, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load %s: %v", shippedDataDir, err)
	}
	embedded, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	for _, reading := range bookReadings {
		fromDirectory, fromBinary := reading.read(onDisk), reading.read(embedded)
		if !reflect.DeepEqual(fromDirectory, fromBinary) {
			t.Errorf("%s parses differently out of the binary than out of %s, so the two "+
				"readings are not one set of bytes", reading.file, shippedDataDir)
		}
		if !holdsSomething(fromBinary) {
			t.Errorf("the embedded %s came back empty. For the two optional books that is "+
				"what a wrongly rooted filesystem looks like: a not-exist error read as "+
				"an author's empty catalogue", reading.file)
		}
		t.Logf("%-18s %s", reading.file, sizeOf(fromBinary))
	}
	// The whole library, once the one field that is *meant* to differ is taken
	// out of the comparison. The rows above name the file a difference is in,
	// which is what a reader needs; this catches a field no row thought to read.
	onDisk.home = dataHome{}
	if !reflect.DeepEqual(onDisk, embedded) {
		t.Error("the two libraries differ in a field no reading above covers")
	}
}

// holdsSomething reports whether a parsed book holds anything at all.
//
// A slice or a map is asked for its length, because an empty one is not the zero
// value and would pass an IsZero check; everything else — a rules struct, a
// bounds struct, the affinity matrix — is asked whether it is still all zeroes,
// which is what a file nothing was read out of leaves behind.
func holdsSomething(reading any) bool {
	value := reflect.ValueOf(reading)
	switch value.Kind() {
	case reflect.Slice, reflect.Map:
		return value.Len() > 0
	default:
		return !value.IsZero()
	}
}

// sizeOf is what a reading is worth logging: entries where there are entries to
// count, and otherwise that it is one record.
func sizeOf(reading any) string {
	value := reflect.ValueOf(reading)
	switch value.Kind() {
	case reflect.Slice, reflect.Map:
		return strconv.Itoa(value.Len()) + " entries"
	default:
		return "one record"
	}
}

// aHomeDecision is what one function in this package does when the library it is
// asked of has no data directory.
type aHomeDecision struct {
	// decision is the answer, in a few words, for a reader of the failure below.
	decision string
	// check asserts that answer against a real embedded library. onDisk is beside
	// it so a check can show its own claim is measured — "hands out no art" is
	// worth nothing from a fixture that has no art either way.
	check func(t *testing.T, embedded, onDisk *Library)
}

// homeDecisions is every function in this package that reaches Library.home, and
// what each of them does when there is no directory.
//
// ⚠️ **The keys are checked against the source, both ways.** The walk below
// parses this package and holds this map equal to the set of functions that
// really reach the field, so a seventeenth consumer added without a decision is
// a red test in the commit that adds it, and a row left behind after a consumer
// stops reaching the field is red too. That bijection is the point: an earlier
// version of this file asserted a handful of accessors return an error, which is
// exactly the shape that passes while a *different* accessor leaks.
//
// The two shapes of answer are deliberate and are not interchangeable:
//
//   - An accessor that returns a bare string answers **""**. Its signature
//     cannot carry a refusal, and "" is the one string that cannot be opened,
//     written or walked by accident — the same reading PlayerSquadsPath takes of
//     a machine with no configuration directory.
//   - Anything that can refuse **does**, with ErrNoDataDirectory. Every write in
//     the package funnels through replaceFile, so the loud half of the decision
//     is where it counts: nothing is written to nowhere in silence.
var homeDecisions = map[string]aHomeDecision{
	"(*Library).Dir": {
		decision: `"" — read from nowhere`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			if got := embedded.Dir(); got != "" {
				t.Errorf("Dir() is %q, want the empty path", got)
			}
			if onDisk.Dir() != shippedDataDir {
				t.Errorf("Dir() over a directory is %q, so this row is not measuring the "+
					"difference it claims", onDisk.Dir())
			}
		},
	},
	"(*Library).MatchesEmbeddedData": {
		decision: "true — it IS the embedded copy",
		check: func(t *testing.T, embedded, onDisk *Library) {
			same, err := embedded.MatchesEmbeddedData()
			if err != nil {
				t.Fatalf("MatchesEmbeddedData(): %v", err)
			}
			if !same {
				t.Error("the embedded library reports it differs from the embedded data, so " +
					"a client would draw 'your edits will not reach the battle' over data " +
					"nobody can edit")
			}
		},
	},
	"(*Library).CastPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "CastPath", embedded.CastPath(), castFile)
		},
	},
	"(*Library).OriginsPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "OriginsPath", embedded.OriginsPath(), originsFile)
		},
	},
	"(*Library).SpeciesPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "SpeciesPath", embedded.SpeciesPath(), speciesFile)
		},
	},
	"(*Library).SkillsPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "SkillsPath", embedded.SkillsPath(), skillsFile)
		},
	},
	"(*Library).SquadsPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "SquadsPath", embedded.SquadsPath(), squadsFile)
		},
	},
	"(*Library).BattlesPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "BattlesPath", embedded.BattlesPath(), battlesDir)
		},
	},
	"(*Library).AssetsPath": {
		decision: `""`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			refusesAPath(t, "AssetsPath", embedded.AssetsPath(), assetsDir)
		},
	},
	"(*Library).ImagePath": {
		decision: `"" — the art is not embedded`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			// An authored path, exactly as it is written in cast.json: relative and
			// slash separated. It is the input that produced the leak — joined onto
			// an empty directory it comes back unchanged, and ImageExists then
			// stats it in the working directory.
			const authored = "assets/fixture/adept.svg"
			refusesAPath(t, "ImagePath", embedded.ImagePath(authored), authored)
			if embedded.ImageExists(authored) {
				t.Error("ImageExists says the embedded library has art on disk")
			}
		},
	},
	"(*Library).ArtFiles": {
		decision: "refuses — a walk of `assets` in the working directory is not this library's art",
		check: func(t *testing.T, embedded, onDisk *Library) {
			art, err := embedded.ArtFiles()
			if !errors.Is(err, ErrNoDataDirectory) {
				t.Errorf("ArtFiles() gave %d files and error %v, want ErrNoDataDirectory: "+
					"joined onto an empty directory the root of that walk is the "+
					"relative name `assets`", len(art), err)
			}
			if len(art) != 0 {
				t.Errorf("ArtFiles() handed back %d paths from nowhere, starting %q",
					len(art), art[0])
			}
			// The same claim measured: over a directory it really does find art, so
			// the refusal above is a refusal rather than this library having none.
			found, err := onDisk.ArtFiles()
			if err != nil || len(found) == 0 {
				t.Errorf("ArtFiles() over %s found %d files (err %v), so the refusal above "+
					"is not measured against anything", shippedDataDir, len(found), err)
			}
		},
	},
	"(*Library).SaveBattleLog": {
		decision: "refuses",
		check: func(t *testing.T, embedded, onDisk *Library) {
			// One roster entry and nothing else: Log.Replayable is len(Roster) > 0
			// and it is checked before the path is resolved, so an empty log would
			// be refused for the wrong reason and this row would measure nothing.
			log := battle.Log{Seed: 7, Roster: []battle.Roster{{}}}
			path, err := embedded.SaveBattleLog("home", "away", 7, log)
			if !errors.Is(err, ErrNoDataDirectory) {
				t.Errorf("SaveBattleLog() gave %q and error %v, want ErrNoDataDirectory", path, err)
			}
			if path != "" {
				t.Errorf("SaveBattleLog() reported it wrote to %q", path)
			}
		},
	},
	"(*Library).replaceFile": {
		decision: "refuses — every write in the package funnels here",
		check: func(t *testing.T, embedded, onDisk *Library) {
			// ⚠️ The assertion is not only the error: it is that the file is not
			// there afterwards. This is the defect in its original form — a write
			// through an empty directory lands in the process's working directory,
			// which under `go test` is the package's own source folder.
			const name = "a-file-no-test-may-leave-behind.json"
			if err := embedded.replaceFile(name, []byte("{}")); !errors.Is(err, ErrNoDataDirectory) {
				t.Errorf("replaceFile() refused with %v, want ErrNoDataDirectory", err)
			}
			if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s exists in the working directory after a write to a library "+
					"with no data directory: the write landed in the source tree", name)
				_ = os.Remove(name)
			}
		},
	},
	"(*Library).recheckCarriers": {
		decision: "refuses — the books it re-reads are on disk, and there are none",
		check: func(t *testing.T, embedded, onDisk *Library) {
			carried := embedded.Skills().Skills()
			if len(carried) == 0 {
				t.Fatal("the embedded skill book is empty, so this row measures nothing")
			}
			id := carried[0].ID
			_, _, err := embedded.recheckCarriers(embedded.Skills(), id)
			if !errors.Is(err, ErrNoDataDirectory) {
				t.Errorf("recheckCarriers() refused with %v, want ErrNoDataDirectory", err)
			}
		},
	},
	"(*Library).artToCheck": {
		decision: "no art — whether a picture is on disk is not asked",
		check: func(t *testing.T, embedded, onDisk *Library) {
			everybody := embedded.Characters().All()
			if len(everybody) == 0 {
				t.Fatal("the embedded cast is empty, so this row measures nothing")
			}
			if pictures := everybody[0].Art(); len(pictures) == 0 {
				t.Fatalf("%s declares no art, so this row measures nothing", everybody[0].ID)
			}
			if asked := embedded.artToCheck(everybody[0]); len(asked) != 0 {
				t.Errorf("artToCheck offered %d pictures to look for with nowhere to look",
					len(asked))
			}
			if asked := onDisk.artToCheck(everybody[0]); len(asked) == 0 {
				t.Error("artToCheck offers nothing over a real directory either, so the " +
					"claim above is not measured")
			}
		},
	},
	"(*Library).Inspect": {
		decision: `Dir "" and no art problems`,
		check: func(t *testing.T, embedded, onDisk *Library) {
			report := embedded.Inspect()
			if report.Dir != "" {
				t.Errorf("the report names %q as the directory it inspected", report.Dir)
			}
			for _, problem := range report.Problems {
				missing, isArt := problem.(*MissingArtProblem)
				if !isArt {
					continue
				}
				t.Errorf("the report calls %s's %q missing, and the art is not embedded: "+
					"every character in the cast would carry one of these",
					missing.ID, missing.Image)
			}
			for _, row := range report.Rows {
				if len(row.Art) != 0 {
					t.Errorf("%s's row lists %d pictures a library with no directory could "+
						"not have looked at", row.ID, len(row.Art))
				}
			}
		},
	},
}

// refusesAPath is the assertion every string-returning path accessor owes: the
// empty path, and specifically **not** the bare name a join onto an empty
// directory would have produced.
//
// Naming the leaked value is what makes the failure readable — `want ""` says a
// path is wrong, and `got "cast.json"` says which defect it is.
func refusesAPath(t *testing.T, accessor, got, wouldLeak string) {
	t.Helper()
	if got == wouldLeak {
		t.Errorf("%s() answered %q, which is the bare relative name: it would read from, "+
			"or write into, whatever directory the player is standing in", accessor, got)
		return
	}
	if got != "" {
		t.Errorf("%s() answered %q, want the empty path", accessor, got)
	}
}

// directoryCallers is the short list of functions allowed past join to the raw
// directory string, and why each needs it.
//
// join is the only expression that builds a path under a data directory, and
// dataHome.directory is the only way round it. Something handing the directory
// to os.DirFS, or to a package function that takes one, needs the string itself
// rather than a path under it — and owes its own check for the string being
// empty, which is what these three have and what a fourth would have to write.
var directoryCallers = map[string]string{
	"(*Library).Dir": "two clients draw it in a header line, uncleaned, and \"\" draws as nothing",
	"(*Library).MatchesEmbeddedData": "os.DirFS takes a directory; guarded by known() above it, " +
		"since os.DirFS(\"\") errors rather than resolving to the working directory",
	"(*Library).ArtFiles": "the package function walks a directory; it refuses an empty one itself",
	"(*Library).Inspect":  "the report says which directory was inspected, and \"\" says none was",
}

// TestEveryLibraryAccessorThatReachesTheDataDirectoryDecidesWhatNoDirectoryMeans
// is the guard on the sharp edge of a library with no directory.
//
// It is two tests wearing one name on purpose, because either half alone is the
// shape that passes while the defect ships:
//
//   - The **walk** parses this package and holds homeDecisions equal, both ways,
//     to the set of functions that really reach Library.home. A table asserting
//     a handful of accessors is exactly what lets a different accessor leak, and
//     nothing about a new accessor makes anybody open this file — so the source
//     is what decides which cases exist.
//   - The **checks** run every row against a real embedded library. A walk on its
//     own proves only that somebody wrote a row down.
//
// ⚠️ The walk logs its case count and refuses to run on none. A walk whose
// predicate has stopped matching agrees with every claim there is.
//
// *Sees:* a bare relative path out of any accessor; a new consumer of the field
// with no decision; a decision that stops being true; a write landing in the
// source tree.
// *Cannot see:* a path built from something other than this field — PlayerSquadsPath
// takes a configuration directory and answers "" to an empty one on its own terms.
func TestEveryLibraryAccessorThatReachesTheDataDirectoryDecidesWhatNoDirectoryMeans(t *testing.T) {
	reached, calls, scanned := walkForTheDataHome(t)
	if scanned == 0 {
		t.Fatal("the walk read no source files, so it measures nothing")
	}
	if len(reached) == 0 {
		t.Fatalf("the walk read %d files and found nothing reaching Library.home, and "+
			"sixteen functions do: it is measuring nothing", scanned)
	}
	t.Logf("%d source files, %d functions reaching the data directory, %d of them past join",
		scanned, len(reached), len(calls))

	for _, name := range slices.Sorted(maps.Keys(reached)) {
		decided, hasDecision := homeDecisions[name]
		if !hasDecision {
			t.Errorf("%s (%s) reaches the data directory and no decision is written down "+
				"for it. With no directory, filepath.Join gives it a bare relative name — "+
				"add a row to homeDecisions saying what it answers instead",
				name, reached[name])
			continue
		}
		t.Logf("  %-34s %s", name, decided.decision)
	}
	for _, name := range slices.Sorted(maps.Keys(homeDecisions)) {
		if _, reaches := reached[name]; !reaches {
			t.Errorf("homeDecisions has a row for %s and nothing of that name reaches the "+
				"data directory any more: the row is stale and its check is measuring a "+
				"function that has moved on", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(calls)) {
		because, allowed := directoryCallers[name]
		if !allowed {
			t.Errorf("%s (%s) takes the raw directory past join, which is the one way to "+
				"build a path without the emptiness check. If it genuinely needs the "+
				"string, add it to directoryCallers with the check it makes instead",
				name, calls[name])
			continue
		}
		t.Logf("  past join: %-24s %s", name, because)
	}
	for name := range directoryCallers {
		if _, calls := calls[name]; !calls {
			t.Errorf("directoryCallers excuses %s and it no longer reads the raw directory", name)
		}
	}

	embedded, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	onDisk, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load %s: %v", shippedDataDir, err)
	}
	for _, name := range slices.Sorted(maps.Keys(homeDecisions)) {
		t.Run(name, func(t *testing.T) {
			homeDecisions[name].check(t, embedded, onDisk)
		})
	}
}

// walkForTheDataHome parses this package and reports which functions reach
// Library.home, which of those go past join to the raw directory string, and how
// many files it read.
//
// It walks this package's own directory rather than the module, because dataHome
// is unexported: nothing outside this package can reach the field at all, which
// is the property that makes the walk exhaustive rather than a sample. Test files
// are skipped — a fixture setting the field is not a consumer of it.
//
// ⚠️ It matches a **selector expression** and not the identifier: the field's
// name appears as a key in `&Library{home: home}` and as a declaration in the
// struct type, and an identifier match would count both — a row for loadBooks
// that has no decision to make, and a walk that cannot be made to fail.
func walkForTheDataHome(t *testing.T) (reached, calls map[string]string, scanned int) {
	t.Helper()
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list this package's sources: %v", err)
	}
	reached, calls = map[string]string{}, map[string]string{}
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
			name := qualifiedName(function)
			ast.Inspect(function.Body, func(node ast.Node) bool {
				selector, isSelector := node.(*ast.SelectorExpr)
				if !isSelector {
					return true
				}
				switch selector.Sel.Name {
				case "home":
					reached[name] = path
				case "directory":
					calls[name] = path
				}
				return true
			})
		}
	}
	return reached, calls, scanned
}

// qualifiedName is a function's name with its receiver type, so that a method
// and a package function of the same name cannot share a row.
func qualifiedName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	owner := ""
	ast.Inspect(function.Recv.List[0].Type, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.StarExpr:
			owner = "*"
			return true
		case *ast.Ident:
			owner += typed.Name
			return false
		}
		return true
	})
	return "(" + owner + ")." + function.Name.Name
}
