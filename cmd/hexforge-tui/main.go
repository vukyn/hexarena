// Command hexforge-tui authors the cast in a full-screen terminal program.
//
// It is the other front-end over internal/forge. cmd/hexforge is flags and a
// line-oriented prompt, which is what a script and a pipe use; this one takes
// over the screen, so it can show a stat budget and a carry check updating as
// the author types rather than refusing an answer several questions later.
// Neither knows a rule of its own: the id check, the kit check, the element
// check and the budget all come from internal/forge, so the two cannot drift
// into disagreeing about what a legal character is.
//
// This one also speaks Vietnamese, and does so by default. Every sentence it
// shows comes from internal/i18n — there is no user-visible wording in this
// package's own source, which is what TestNoScreenHoldsItsOwnWording checks —
// and the facts behind those sentences are still internal/forge's. cmd/hexforge
// stays English on purpose: it is what a script reads.
//
// The screen is bubbletea, styled with lipgloss and built from bubbles. None of
// that reaches the engine: what has to replay identically in a year is
// internal/core, and it holds no state a terminal library could reach.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// programName is what this binary is called, in every language.
const programName = "hexforge-tui"

// options is one invocation: which data directory, and which language.
type options struct {
	dir  string
	lang i18n.Lang
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
	if err := run(chosen); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
		os.Exit(1)
	}
}

// parseOptions reads the flags and settles the language.
//
// The flag beats the environment variable, because the variable is a standing
// preference and the flag is this run. An unreadable value in either is an
// error naming where it came from and which spellings work, rather than a
// silent fall back to the default — somebody who typed "vn" would otherwise
// see a screen in the language they were trying to leave and have no idea why.
//
// The flag descriptions are worded before the strict check, since the flag
// package prints them while it is still parsing. They take the environment's
// language if it is usable and the default if it is not; the proper complaint
// follows a line later.
func parseOptions(arguments []string, environment string, out io.Writer) (options, error) {
	described := i18n.Prefer("", environment)
	set := flag.NewFlagSet(programName, flag.ContinueOnError)
	set.SetOutput(out)
	dir := set.String("data", forge.DefaultDataDir, described.Text(i18n.DataFlagUsage))
	chosen := set.String(i18n.FlagName, "", described.Text(i18n.LanguageFlagUsage))
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
	return options{dir: *dir, lang: lang}, nil
}

func run(chosen options) error {
	if !stdoutIsTerminal() {
		return errors.New(chosen.lang.Text(i18n.NotATerminal))
	}
	lib, err := forge.Load(chosen.dir)
	if err != nil {
		return err
	}
	program := tea.NewProgram(newModel(lib, chosen.lang), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

// stdoutIsTerminal reports whether there is a screen to take over.
//
// It is the same character-device test cmd/hexforge uses on stdin, and it has
// the same known limitation: /dev/null is a character device too, so a run with
// stdout redirected there looks like a terminal. That trade is deliberate. The
// case worth catching is a pipe or a file — someone running this expecting
// output they can read afterwards — and this catches those exactly. An ioctl
// through golang.org/x/term would also tell /dev/null apart from a terminal;
// that is the upgrade to make if the redirect case ever actually bites.
func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
