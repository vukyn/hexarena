// Command hexforge authors the cast the battles are fought with.
//
// It is a reader, a prompt and a writer, nothing more. Every rule about what a
// character may be lives in internal/core/cast, and this binary validates by
// calling those parsers rather than by knowing anything itself — the same
// relationship cmd/hexarena has with the event log. If a check belongs
// anywhere, it belongs in the parser, because the parser is what the game runs
// through at boot and this tool is not.
//
// It is the one place in the repository that reads the data directory rather
// than the embedded copy, and the one place that asks whether a file exists.
// internal/core may not touch the filesystem, and internal/seed only ever
// reads the copy the embed directive baked in, so a tool that has to write a
// data file and check that the art beside it is really there has to be a
// separate program.
//
// (This comment spells that directive "the embed directive" on purpose. A
// comment line beginning with its real spelling is read by the compiler as a
// directive, and in this repository a stray one would be a real trap.)
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// defaultDataDir is where the game's own data lives, relative to the module
// root. Every subcommand takes --data so a scratch copy can be edited without
// touching the shipped files.
const defaultDataDir = "internal/seed/data"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stdout)
		return
	}
	command, args := os.Args[1], os.Args[2:]
	switch command {
	case "help", "-h", "-help", "--help":
		usage(os.Stdout)
		return
	}
	run, known := commands[command]
	if !known {
		fmt.Fprintf(os.Stderr, "hexforge: unknown command %q\n\n", command)
		usage(os.Stderr)
		os.Exit(2)
	}
	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "hexforge: %v\n", err)
		os.Exit(1)
	}
}

// commands is the dispatch table. The standard library has no subcommand
// support, so argv is split by hand and each subcommand owns its own FlagSet.
var commands = map[string]func([]string) error{
	"origins":    runOrigins,
	"archetypes": runArchetypes,
	"cast":       runCast,
	"new":        runNew,
	"show":       runShow,
	"check":      runCheck,
}

func usage(out io.Writer) {
	fmt.Fprint(out, strings.TrimLeft(`
hexforge authors the cast the battles are fought with.

  hexforge origins                    the works the cast is borrowed from
  hexforge origins add <id> --title T --medium anime [--year N] [--note ...]
                                      add a work to the catalog
  hexforge archetypes                 the role presets, their curves and their kits
  hexforge cast                       the authored characters
  hexforge new [id]                   create a character; flags prefill, the
                                      wizard asks only for what is still missing
  hexforge show <id> [--level N]      resolve one character and show what it costs
  hexforge check                      parse the books from disk, verify the art
                                      is really there, report the stat budget

Every subcommand takes --data <dir> (default `+defaultDataDir+`), which is the
directory it reads and writes. Run any subcommand with -h for its own flags.

The game ships the embedded copies of these files, so rebuild after editing.
`, "\n"))
}

// dataFlag registers the one flag every subcommand shares.
func dataFlag(set *flag.FlagSet) *string {
	return set.String("data", defaultDataDir, "the data directory to read and write")
}

// newFlagSet builds a subcommand's flag set. ContinueOnError rather than
// ExitOnError so a bad flag comes back as an error and leaves main the single
// place that decides an exit code.
func newFlagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet("hexforge "+name, flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	return set
}

// parseArgs splits a subcommand's argv into its operands and its flags, and
// parses the flags.
//
// The split is needed because flag.Parse stops at the first argument that is
// not a flag, which would make `hexforge show some.id --level 30` quietly
// resolve at the default level instead: the flag would sit in Args() and never
// be read. Pulling the leading operands off first means an id may come before
// its flags, which is how the commands are documented and how anyone types
// them.
func parseArgs(set *flag.FlagSet, args []string) ([]string, error) {
	operands := args
	rest := []string(nil)
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") && arg != "-" {
			operands, rest = args[:i:i], args[i:]
			break
		}
	}
	if err := set.Parse(rest); err != nil {
		return nil, err
	}
	return append(operands, set.Args()...), nil
}
