// Command hexforge-tui authors the cast in a full-screen terminal program.
//
// It is the other front-end over internal/forge. cmd/hexforge is flags and a
// line-oriented prompt, which is what a script and a pipe use; this one takes
// over the screen, so it can show a stat budget and a carry check updating as
// the author types rather than refusing an answer several questions later.
// Neither knows a rule of its own: the id check, the kit check, the element
// check, the budget and the wording of every refusal all come from
// internal/forge, so the two cannot drift into disagreeing about what a legal
// character is.
//
// The screen is bubbletea, styled with lipgloss and built from bubbles. None of
// that reaches the engine: what has to replay identically in a year is
// internal/core, and it holds no state a terminal library could reach.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vukyn/hexarena/internal/forge"
)

func main() {
	dir := flag.String("data", forge.DefaultDataDir, "the data directory to read and write")
	flag.Parse()
	if arguments := flag.Args(); len(arguments) > 0 {
		fmt.Fprintf(os.Stderr, "hexforge-tui: takes no arguments, got %v\n", arguments)
		os.Exit(2)
	}
	if err := run(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "hexforge-tui: %v\n", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	if !stdoutIsTerminal() {
		return fmt.Errorf(
			"stdout is not a terminal, and a full-screen program would write control codes into it.\n" +
				"Use hexforge instead: it authors the same cast through the same checks, takes flags\n" +
				"and reads a pipe, and `hexforge check` prints what this program's check screen shows")
	}
	lib, err := forge.Load(dir)
	if err != nil {
		return err
	}
	program := tea.NewProgram(newModel(lib), tea.WithAltScreen())
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
