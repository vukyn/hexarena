// Command hexarena-tui plays the game in a full-screen terminal program.
//
// It is the second front-end over internal/screen, and it is the thing eleven
// screens were moved into that package for. cmd/hexarena is flags and a
// line-oriented prompt — which is what a script, a pipe and `--replay --verify`
// use, and which is the verification contract nothing here may touch; this one
// takes over the screen, so a player can walk the cast, read what a skill does
// and fight a battle with a board drawn beside the moves.
//
// ⚠️ **It draws the same screens as cmd/hexforge-tui and authors none of them.**
// Three of the catalogues it offers are screens that write a file in the
// authoring tool — the skill listing writes `skills.json`, the works catalogue
// writes `origins.json`, the squad catalogue's two depths under it write
// `squads.json` — and this client offers none of those keys and names none of
// them in a footer. That is one answer in one place: `screen.Context.Authoring`
// is nought here, which is the read-only reading, and this package never sets
// it. See `readonly_test.go` for the two measurements that hold it.
//
// It speaks Vietnamese by default, like the authoring tool and for the same
// reason. Every sentence it shows comes from internal/i18n — there is no
// user-visible wording in this package's own source, which is what
// TestNoScreenHoldsItsOwnWording checks, and it is this package's **own** copy
// of that walker because the one in internal/screen reads its own directory
// only.
//
// ⚠️ **-version is the one thing it prints that is not in either language, and
// it is not an exception to the rule above.** What it writes is
// wire.Version.Report — one function both binaries answer that flag with, whose
// two labels are `protocol` and `data`, the same ones cmd/hexarena-host prints
// and the same ones both version refusals tell a player to read. So there is no
// sentence here to translate; the flag's own *description* is a sentence and
// comes from internal/i18n like every other. → i18n.VersionFlagUsage.
//
// The screen is bubbletea, styled with lipgloss. None of that reaches the
// engine: what has to replay identically in a year is internal/core, and it
// holds no state a terminal library could reach.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/wire"
)

// programName is what this binary is called, in every language.
const programName = "hexarena-tui"

// options is one invocation: which data directory, which language, and whether
// the ask is for a screen at all.
type options struct {
	dir string
	// dataGiven is whether --data was really typed, as against left at
	// forge.DefaultDataDir.
	//
	// ⚠️ **The string cannot answer that question and this is the only field
	// that can.** A player who types the default path by hand and a player who
	// types nothing hand `dir` the same value, and the two mean opposite things
	// to loadLibrary: one named a directory and is owed a refusal when it is not
	// there, the other named nothing and is owed the copy in the binary. It is
	// filled from flag.FlagSet.Visit, which is the only thing that knows.
	dataGiven bool
	// squads is the player's **own** squad file — a different file from the
	// game's own squads.json, which lives in dir and which a player has no
	// business in. Empty means there is none to read, which is both what a
	// machine with no resolvable configuration directory answers and what
	// `--squads ""` asks for. → forge.PlayerSquads.
	squads string
	lang   i18n.Lang
	// version is the ask that is answered instead of taking over the screen:
	// print what this binary is and exit.
	version bool
}

func main() {
	chosen, err := parseOptions(os.Args[1:], os.Getenv(i18n.EnvVar), playerSquadsPath(), os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
		os.Exit(2)
	}
	if err := run(chosen, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
		os.Exit(1)
	}
}

// parseOptions reads the flags and settles the language.
//
// The flag beats the environment variable, because the variable is a standing
// preference and the flag is this run. An unreadable value in either is an
// error naming where it came from and which spellings work, rather than a
// silent fall back to the default — somebody who typed "vn" would otherwise see
// a screen in the language they were trying to leave and have no idea why.
//
// It is cmd/hexforge-tui's own arrangement, down to the descriptions being
// worded before the strict check, because the flag package prints them while it
// is still parsing. Two front-ends reading `HEXARENA_LANG` differently would be
// two answers to a standing preference.
func parseOptions(arguments []string, environment, playerSquads string, out io.Writer) (options, error) {
	described := i18n.Prefer("", environment)
	set := flag.NewFlagSet(programName, flag.ContinueOnError)
	set.SetOutput(out)
	dir := set.String("data", forge.DefaultDataDir, described.Text(i18n.DataFlagUsage))
	// The default is a **parameter** rather than something worked out here, and
	// that is the one decision this flag carries. os.UserConfigDir reads the
	// environment and resolves differently on every platform, so a function that
	// called it would answer whatever machine it ran on — main resolves it once
	// and hands the answer in, exactly as it already hands the language variable
	// in rather than reading it here. → playerSquadsPath and forge.PlayerSquadsPath.
	squads := set.String("squads", playerSquads, described.Text(i18n.SquadsFlagUsage))
	chosen := set.String(i18n.FlagName, "", described.Text(i18n.LanguageFlagUsage))
	version := set.Bool("version", false, described.Text(i18n.VersionFlagUsage))
	if err := set.Parse(arguments); err != nil {
		return options{}, err
	}
	lang, err := i18n.Resolve(*chosen, environment)
	if err != nil {
		return options{}, err
	}
	if operands := set.Args(); len(operands) > 0 {
		return options{}, errors.New(lang.Say(i18n.NoArguments, operands))
	}
	return options{
		dir:       *dir,
		dataGiven: wasSet(set, "data"),
		squads:    *squads,
		lang:      lang,
		version:   *version,
	}, nil
}

// wasSet reports whether a flag was given on the command line, as against left
// at its default.
//
// flag has no other way to ask: a --data of `internal/seed/data` typed by hand
// and a --data nobody typed are the same string. It is cmd/hexforge/weigh.go's
// own function, name and all, and a copy rather than a shared helper for the
// reason the wording walker in this package is one — a command's flags are its
// own, and four lines of flag plumbing is not a dependency between two binaries.
// cmd/hexforge/skills.go reads the same Visit for the same reason at greater
// length.
//
// The walk is over the flags that were given, so nothing about the order it
// visits in reaches the answer.
func wasSet(set *flag.FlagSet, name string) bool {
	given := false
	set.Visit(func(flagged *flag.Flag) {
		if flagged.Name == name {
			given = true
		}
	})
	return given
}

// playerSquadsPath is the default a player's own squad file is looked for at,
// and it is the one line of this program that asks the operating system where a
// player's configuration lives.
//
// ⚠️ **This is the edge, and it is deliberately the whole of it.** os.UserConfigDir
// reads the environment ($XDG_CONFIG_HOME, ~/Library/Application Support,
// %AppData%), so a test that drove anything below it would be measuring the
// platform it happened to run on rather than this program — which is the mistake
// memory/windows-sets-no-term.md is about, and the reason the path is resolved
// once here and taken as a value everywhere else. Nothing under this function
// calls it, and the flag can name a file without it.
//
// A machine with no resolvable configuration directory answers the empty path,
// which forge.PlayerSquads reads as "there is no player file" — the same silent
// answer an absent file gets, because a directory that does not exist cannot be
// hiding one.
func playerSquadsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return forge.PlayerSquadsPath(dir)
}

// run is the invocation, with the writer -version answers through handed in —
// which is what makes that one path testable, since everything below it takes
// over a terminal instead of writing to anything.
func run(chosen options, out io.Writer) error {
	// ⚠️ **Ahead of the terminal check, and that is the whole placement
	// decision.** The check below refuses a pipe because a full-screen program
	// would pour control codes into one; -version writes three plain lines and
	// takes over nothing, so it is not what that check is about. A version a
	// script cannot read because the answer was "stdout is not a terminal" would
	// be a machine-readable version with the machines left out, which is half of
	// what it is for. It is also ahead of the books, so a --data directory that
	// does not exist is not a reason a binary cannot say what it is.
	//
	// It is not in parseOptions, where flag.ErrHelp is answered, for two
	// reasons: that function writes to **stderr** and a version is output rather
	// than a diagnostic, and it is the one part of this file with no side
	// effects at all — printing from it would be the first.
	//
	// ⚠️ It is ahead of loadLibrary as well, which is why every -version case in
	// version_flag_test.go names a directory that is not there **and** says the
	// flag was given: without dataGiven those cases would take the embedded copy
	// and stop measuring the placement they are about.
	if chosen.version {
		version, err := wire.Local(buildString())
		if err != nil {
			return err
		}
		fmt.Fprint(out, version.Report(programName))
		return nil
	}
	if !stdoutIsTerminal() {
		return errors.New(chosen.lang.Text(i18n.GameNotATerminal))
	}
	lib, err := loadLibrary(chosen)
	if err != nil {
		return err
	}
	// ⚠️ **A malformed player file stops the program here, and that placement is
	// the answer to "where is it said".** The file is hand-edited, so a missing
	// comma is a thing that happens, and the one outcome that may not follow is
	// the player's squads quietly not being there — a player who was shown the
	// shipped sides and nothing else would read it as their work having
	// vanished. Refusing before a screen exists is what makes it unswallowable:
	// there is no drawing for the sentence to be missed on and no screen the
	// reader might not visit. It is also exactly what a --data directory that
	// will not parse already gets, three lines up.
	//
	// An **absent** file is silent and answers no squads at all, because a player
	// who has never built a side is the ordinary case rather than a broken one.
	player, err := lib.PlayerSquads(chosen.squads)
	if err != nil {
		return err
	}
	// No alternate-screen option here: bubbletea v2 asks for it on the view the
	// model returns, so it is model.View that says so.
	//
	// ⚠️ **The three lines below are one guarantee and have to stay together.**
	// A session's chooser blocks on a channel that Update feeds, and the only
	// other thing that can unblock it is its own context being cancelled — so
	// "a player who quits mid-turn leaves the Play goroutine blocked for ever"
	// is closed by this defer rather than by anybody remembering to leave a
	// match. It fires however Run returns: a clean quit, ctrl+c, or an error.
	// The process cannot leave this function without cancelling.
	//
	// The order is the knot the sender interface exists to untie: the program
	// cannot be built until the model is, and the model cannot be built until
	// the session is, so the session learns where to send **after** both exist.
	sess := newSession()
	program := tea.NewProgram(newModel(lib, chosen.lang, sess, player))
	sess.attach(program)
	defer sess.leave()
	_, err = program.Run()
	return err
}

// loadLibrary is where this client's books come from, and it is the one thing
// about this binary a player installing it from the module proxy notices.
//
// The rule is three lines, and the middle one is why the other two are not one:
//
//	--data given                the directory, always
//	--data not given, it exists the directory, exactly as before
//	--data not given, it is not the copy the binary embeds
//
// An installed binary is the third line. `forge.DefaultDataDir` is a **relative**
// path — it is where the data sits inside a checkout — so away from one it names
// a directory in whatever the player happened to be standing in, and this client
// died on it with `read internal/seed/data/combat.json: no such file or
// directory`. Nothing about a battle needed that directory: model.go builds its
// mirror from seed.Books() whatever --data says, because the digest at a room's
// gate is over the embedded files.
//
// ⚠️ **The fallback keys on the directory being ABSENT and never on the load
// failing.** "Load, and take the embedded copy if that returned an error" reads
// the same from here and is a different program: an author who leaves a trailing
// comma in skills.json would be handed the baked-in books and told nothing, and
// would spend the evening wondering why an edit they can see in the file does
// not reach the screen. A directory that exists and will not parse must refuse
// exactly as it did before this function existed. It is the distinction
// testfixture.RequireSharedArt is built on — a guard keyed on the *result* of
// the thing it guards deletes itself, so key on the *probe*.
//
// ⚠️ **Absent means absent**, rather than "the stat did not succeed". A stat
// that fails for any other reason is not a missing data directory, and
// answering that with the embedded copy would swallow the one error naming the
// real problem. Only fs.ErrNotExist takes the third line — and that is measured
// rather than asserted here, by
// TestADataDirectoryPathBlockedByAFileIsRefusedRatherThanQuietlyReplaced, which
// puts a plain file where a path component should be a directory: every stat
// below it then fails with ENOTDIR, which is neither present nor absent.
// Widening this to `err != nil` compiles and passes every other test in the
// package.
//
// ⚠️ **A named directory is never second-guessed**, which is the first line.
// A player who typed `--data /nope` gets a refusal naming it; handing them
// different data is not an answer to what they asked.
//
// Why the second line is not "always embed": `make play-tui` passes no --data
// and is run from the module root by an author who wants the cast browser to
// show the file they just edited. Embedding regardless would take that away
// without saying so, and the join screen's warning that those edits will not
// reach a battle is drawn off forge.MatchesEmbeddedData, which needs a
// directory to compare.
func loadLibrary(chosen options) (*forge.Library, error) {
	if chosen.dataGiven || !dataDirectoryIsAbsent(chosen.dir) {
		return forge.Load(chosen.dir)
	}
	return forge.LoadEmbedded()
}

// dataDirectoryIsAbsent reports whether there is nothing at all at dir.
//
// Anything that is there — a directory, or a file sitting where one should be —
// is not absent, and is left to forge.Load to read and to refuse in its own
// words. This function's whole job is to be narrower than "the load failed".
func dataDirectoryIsAbsent(dir string) bool {
	_, err := os.Stat(dir)
	return errors.Is(err, fs.ErrNotExist)
}

// build is the version string this binary announces, stamped by a release:
//
//	go build -ldflags "-X main.build=v0.4.0" ./cmd/hexarena-tui
//
// It is one of the three numbers a peer is told at a room's gate and the only
// one with nothing to decide — printed by a host and read by a person working
// out which of two machines to update. → wire.Version.Build.
var build string

// buildString is wire.BuildOf over this process, which is the one impure line of
// it. The derivation lives in internal/wire because two binaries need the same
// three-step fallback and a second spelling of one is how two peers come to
// disagree about what they are; the **stamp** stays here, because a linker
// writes into a binary's own variable.
func buildString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return wire.BuildOf(build, info)
}

// stdoutIsTerminal reports whether there is a screen to take over.
//
// It is the same character-device test cmd/hexforge and cmd/hexforge-tui use,
// with the same known limitation: /dev/null is a character device too, so a run
// with stdout redirected there looks like a terminal. The case worth catching is
// a pipe or a file — somebody expecting output they can read afterwards — and
// this catches those exactly.
func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
