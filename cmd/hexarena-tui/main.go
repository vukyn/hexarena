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
	dir  string
	lang i18n.Lang
	// version is the ask that is answered instead of taking over the screen:
	// print what this binary is and exit.
	version bool
}

func main() {
	chosen, err := parseOptions(os.Args[1:], os.Getenv(i18n.EnvVar), os.Stderr)
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
func parseOptions(arguments []string, environment string, out io.Writer) (options, error) {
	described := i18n.Prefer("", environment)
	set := flag.NewFlagSet(programName, flag.ContinueOnError)
	set.SetOutput(out)
	dir := set.String("data", forge.DefaultDataDir, described.Text(i18n.DataFlagUsage))
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
	return options{dir: *dir, lang: lang, version: *version}, nil
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
	// what it is for. It is also ahead of forge.Load, so a --data directory that
	// does not exist is not a reason a binary cannot say what it is.
	//
	// It is not in parseOptions, where flag.ErrHelp is answered, for two
	// reasons: that function writes to **stderr** and a version is output rather
	// than a diagnostic, and it is the one part of this file with no side
	// effects at all — printing from it would be the first.
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
	lib, err := forge.Load(chosen.dir)
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
	program := tea.NewProgram(newModel(lib, chosen.lang, sess))
	sess.attach(program)
	defer sess.leave()
	_, err = program.Run()
	return err
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
