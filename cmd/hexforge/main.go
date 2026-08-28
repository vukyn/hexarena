// Command hexforge authors the cast the battles are fought with, from flags
// and prompts.
//
// It is a reader, a prompt and a writer, nothing more. Every rule about what a
// character may be lives in internal/core/cast, and the authoring logic that
// sequences those rules — loading the books, turning answers into a character,
// writing the file back — lives in internal/forge. This binary validates by
// calling those, rather than by knowing anything itself: the same relationship
// cmd/hexarena has with the event log. If a check belongs anywhere, it belongs
// in the parser, because the parser is what the game runs through at boot and
// this tool is not.
//
// It is flags and a line-oriented prompt on purpose, which is what makes it the
// front-end a script and a pipe use. cmd/hexforge-tui is the same authoring
// logic behind a full-screen program, and a full-screen program cannot run with
// stdin as a pipe. Neither front-end restates a rule the other has: both go
// through internal/forge.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vukyn/hexarena/internal/forge"
)

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
	"species":    runSpecies,
	"archetypes": runArchetypes,
	"passives":   runPassives,
	"statuses":   runStatuses,
	"skills":     runSkills,
	"cast":       runCast,
	"new":        runNew,
	"show":       runShow,
	"check":      runCheck,
	"spar":       runSpar,
	"weigh":      runWeigh,
}

func usage(out io.Writer) {
	fmt.Fprint(out, strings.TrimLeft(`
hexforge authors the cast the battles are fought with.

  hexforge origins                    the works the cast is borrowed from
  hexforge origins add <id> --title T --medium anime [--year N] [--note ...]
                                      add a work to the catalog
  hexforge species                    what a unit can be, and who is one
  hexforge species add <id> --name N [--note ...]
                                      add a kind of creature to the catalog
  hexforge archetypes                 the role presets, their curves and their kits
  hexforge passives                   the declared traits and what each grants
  hexforge statuses                   the timed effects, grouped, and what each does
  hexforge skills                     the declared skills and who may carry each
  hexforge skills add <id> --power N --accuracy N [--element E] [--target T]
                                      [--range N] [--pattern P] [--strikes N]
                                      [--cooldown N] [--applies s:c[:n],...]
                                      [--restrict-elements ...] [--restrict-archetypes ...]
                                      [--restrict-characters ...] [--restrict-species ...]
                                      [--restrict-origins ...]
                                      author a skill; flags prefill, the wizard
                                      asks for what is still missing. A skill is
                                      balance: the golden files will move
  hexforge skills edit <id> [same flags]
                                      change a skill already in the book. Only
                                      the fields named change: --cooldown 0 sets
                                      it to zero, no --cooldown leaves it, and
                                      --restrict-elements "" clears the list.
                                      The id itself cannot be edited, and an edit
                                      that would leave a character or a preset
                                      unable to carry the skill is refused
  hexforge cast                       the authored characters
  hexforge new [id]                   create a character; flags prefill, the
                                      wizard asks only for what is still missing.
                                      --species names what it is, which is what a
                                      skill kept for a lineage asks for
  hexforge show <id> [--level N]      resolve one character and show what it costs
  hexforge check                      parse the books from disk, verify the art
                                      is really there, report the stat budget
  hexforge spar <id> [--level N] [--seeds N]
                                      fight one character against every character
                                      in the book, both ways round, and report
                                      the win rates. A check says a character is
                                      legal; this says whether it belongs
  hexforge weigh <id> <skill> --field F --values a,b,c [--level N] [--seeds N]
                                      price ONE field on ONE skill. The carrier
                                      fights a copy of itself whose only
                                      difference is that field, so the deviation
                                      from an even split is the field's whole
                                      worth. A spar says whether a character
                                      belongs; this says what a number is worth.
                                      Read worth and turns together, and the
                                      band beside both. Every report carries its
                                      own control row, and one that is not
                                      exactly even refuses the whole report
  hexforge weigh --carriers all <skill> --field F --values a,b,c [--level N] [--seeds N]
                                      the same price taken once per carrier that
                                      brings the skill, as a table. Each row has
                                      its own control and its own refusal; a
                                      character that cannot bring it is absent
                                      rather than nought. There is NO headline
                                      figure and no average, deliberately: two
                                      rows were fought against two different
                                      opponents, so a carrier may be compared
                                      only to itself at another value

Every subcommand takes --data <dir> (default `+forge.DefaultDataDir+`), which is the
directory it reads and writes. Run any subcommand with -h for its own flags.

hexforge-tui is the same authoring in a full-screen program; this one is what a
script and a pipe use.

The game ships the embedded copies of these files, so rebuild after editing.
`, "\n"))
}

// dataFlag registers the one flag every subcommand shares.
func dataFlag(set *flag.FlagSet) *string {
	return set.String("data", forge.DefaultDataDir, "the data directory to read and write")
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
//
// It takes and parses in turns rather than once, so an operand may sit *between*
// two flags as well as before or after them. `hexforge weigh --carriers all
// razor_leaf --field crit` is the form that needs it: the selection comes first
// because it decides whether there is a character operand at all, so the skill
// lands in the middle. Taking the leading operands once and parsing the rest
// would stop at `razor_leaf` and hand `--field` back as a fifth operand.
func parseArgs(set *flag.FlagSet, args []string) ([]string, error) {
	operands := []string(nil)
	rest := args
	for len(rest) > 0 {
		split := len(rest)
		for i, arg := range rest {
			if strings.HasPrefix(arg, "-") && arg != "-" {
				split = i
				break
			}
		}
		operands = append(operands, rest[:split]...)
		if err := set.Parse(rest[split:]); err != nil {
			return nil, err
		}
		// Parse stops at the first operand it meets, so whatever it hands back
		// is the next round of the same split. It cannot fail to shrink: either
		// it consumed a flag, or there was none left to consume.
		rest = set.Args()
	}
	return operands, nil
}
