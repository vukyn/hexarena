package main

import (
	"flag"
	"fmt"

	"github.com/vukyn/hexarena/internal/forge"
)

// # Where this tool's books come from, and why the answer is per code path
//
// `forge.DefaultDataDir` is a **relative** path, so before this every one of
// these subcommands only worked from inside a checkout. Installed from the
// module proxy and run anywhere else, all fourteen died on the same line —
//
//	hexforge: read internal/seed/data/combat.json: open internal/seed/data/combat.json: no such file or directory
//
// — on a binary carrying a copy of every one of those files. cmd/hexarena-tui
// was fixed for the same reason (→ forge.LoadForReading); this is the other
// half, and it is not the same fix, because **hexforge writes**.
//
// So the decision is per **code path** and not per subcommand. `origins` lists
// the catalogue and `origins add` appends to it — one name, two paths, and the
// dispatch between them is a single `args[0] == "add"` inside runOrigins. A
// reading path takes the embedded copy when there is no directory; a writing
// path has nowhere to put its answer and must say so in words that name the way
// out. loadForReading and loadForWriting below are those two answers, and
// TestEveryHexforgeCodePathThatReachesTheDataDirectorySaysWhetherItReadsOrWrites
// is what makes a fifteenth subcommand arrive with one of them rather than
// with a silent third behaviour.
//
// ⚠️ **A home directory like `~/.hexforge` was asked for and refused**, and the
// refusal is a measurement rather than a taste: nothing in the game ever plays
// from the --data directory. cmd/hexarena-tui builds its mirror from
// seed.Books() whatever --data says (model.go:781), and cmd/hexarena and
// cmd/hexarena-host declare no --data flag at all — both take their books from
// seed.Books() outright. So the data a battle is fought on is
// `internal/seed/data` compiled in, tracked in git. A home default would send an
// author to edit a copy that reaches neither a battle nor the repository — a
// worse failure than the one above, because it looks like it worked. forge.PlayerSquadsPath is the one thing that legitimately
// lives under os.UserConfigDir, and it is the player's own squads, which the game
// really does read. → TODO.md § FRG-004.

// loadForReading is the books for a subcommand that only ever prints.
//
// The rule is forge.LoadForReading and its three lines are written down there,
// which is also why they are not written down here: cmd/hexarena-tui follows the
// same rule and two commands cannot import each other.
//
// What this adds is the **wording** of the one refusal that rule leaves bare. A
// --data that names a directory which is not there is never second-guessed — the
// caller asked for that directory — but `forge.Load` can only answer with the
// read error on whichever book it reached first, which names a file the caller
// never typed and offers nothing to do about it. So the absence is probed here,
// before the load, and refused in a sentence. A directory that *is* there and
// will not parse still comes back through forge.Load untouched, because a
// trailing comma in skills.json is an author's problem and naming the book is
// the useful answer.
func loadForReading(name string, set *flag.FlagSet, dir string) (*forge.Library, error) {
	given := wasSet(set, "data")
	if given && forge.DataDirectoryIsAbsent(dir) {
		return nil, fmt.Errorf("%s reads a data directory and there is none at %q: "+
			"pass --data <dir> naming one that exists, or leave --data off to read the "+
			"copy of the books embedded in this binary", name, dir)
	}
	return forge.LoadForReading(dir, given)
}

// loadForWriting is the books for a subcommand that will write one of them back,
// and it always requires a real directory.
//
// It may not fall back to the embedded copy for the reason go:embed is read-only:
// there would be nowhere to put the answer. The refusal happens **here**, on the
// absence probe, rather than being left to the write at the end of the run: a
// wizard that prompts for eleven fields and then says it cannot save is a worse
// program than one that says so first, and forge.ErrNoDataDirectory — which is
// what a written-to embedded library answers — would arrive after the prompting.
//
// The message names both ways out, because they are different situations. Someone
// with a checkout somewhere wants `--data <dir>`; someone standing in a checkout
// who ran the binary from the wrong subdirectory wants to know the default is
// relative to the module root.
func loadForWriting(name string, set *flag.FlagSet, dir string) (*forge.Library, error) {
	if forge.DataDirectoryIsAbsent(dir) {
		return nil, fmt.Errorf("%s writes to a data directory and there is none at %q: "+
			"the copy of the books embedded in this binary cannot be written to, so pass "+
			"--data <dir> naming a directory that exists, or run hexforge from the root of "+
			"a hexarena checkout, where %s is the game's own",
			name, dir, forge.DefaultDataDir)
	}
	return forge.Load(dir)
}

// booksAt is what a message calls the place a library came from.
//
// Library.Dir is a display string and is empty for the embedded copy, so a
// sentence that interpolates it raw reads `no character "x" in ; list them
// with…`. This is the one place that empty is turned into words, so the two
// front-ends' habit of one declaration per sentence holds for this one too.
func booksAt(lib *forge.Library) string {
	if !lib.HasDataDirectory() {
		return embeddedBooks
	}
	return lib.Dir()
}

// embeddedBooks is what the copy baked in by go:embed is called wherever a
// message would otherwise name a directory. One spelling, because it appears in
// a listing refusal and in the check report and those must not drift.
const embeddedBooks = "the copy of the books embedded in this binary"
