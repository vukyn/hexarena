package forge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
)

// # A player's own squads, and why they are not the game's
//
// `internal/seed/data/squads.json` is the game's own data: the authoring tool
// writes it, go:embed bakes it into the binary, and it is part of what the data
// digest promises two peers agree on. A player has no business in it. What a
// player has instead is a file of their own, under the directory the platform
// keeps a program's configuration in, holding the sides they built to bring to a
// match.
//
// Both files are the same shape read by the same parser, and neither knows the
// other exists. What puts them side by side is SquadsOffered; what makes that
// safe is that a side is checked against the same books either way.
//
// ⚠️ **Reading a player's file is what this is, and WRITING one is deliberately
// not.** A client that could write it would be a client that authors, and this
// client not authoring is a measured decision rather than an omission —
// cmd/hexarena-tui/readonly_test.go holds four tests over
// screen.Context.Authoring, which is nought there. Nothing in this file writes
// and nothing added for it turns an authoring key on. The next item under
// TODO.md § *PvP over a LAN* is what lets the client build one, and this is what
// makes that a small change rather than a contradiction: today the only place to
// build a squad is the authoring tool, which edits the game's data, so a player
// cannot have a side of their own at all.
//
// ⚠️ **The authoring tool has no business here either**, which is the same
// argument pointing the other way. cmd/hexforge-tui never loads a player file,
// Library.squads stays the game's own, and this list is never folded into it —
// folding it in would put a player's side within reach of SaveSquad, which
// replaces by id, so a player's `s01` would be written over the game's.

// playerDir is the folder a player's own files live in, under whatever
// directory the platform keeps configuration in.
const playerDir = "hexarena"

// playerSquadsFile is a player's own squad catalogue, and it is deliberately
// the same name the game's own file has: the same shape read by the same
// parser, so a second spelling would only suggest a second format.
const playerSquadsFile = squadsFile

// PlayerSquadsPath is where a player's own squads live under a configuration
// directory.
//
// ⚠️ **The configuration directory is a PARAMETER and is not read here.** That
// is the whole reason this function exists rather than an os.UserConfigDir call
// at the site that wants the path: that call reads the environment and resolves
// differently on every platform — $XDG_CONFIG_HOME, ~/Library/Application
// Support, %AppData% — so anything driving it would be measuring the machine the
// test happened to run on rather than the code. The binary resolves it once at
// its own edge and hands the answer down as a value; everything under that takes
// a path. It is the same arrangement as handing GOOS in rather than reading it.
//
// An empty directory answers an empty path rather than a path relative to
// nowhere. os.UserConfigDir fails on a machine with no home, and the honest
// answer there is that there is no player file — not that there is one in
// whatever directory the program was started from.
func PlayerSquadsPath(configDir string) string {
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, playerDir, playerSquadsFile)
}

// PlayerSquads reads a player's own squad file and checks every side in it
// against these books.
//
// Three outcomes, and telling them apart is the whole of what this decides:
//
//   - **Absent is silent.** A player who has never built a side has no file, and
//     that is the ordinary case rather than a broken installation — the same
//     reading Load takes of a data directory with no squads.json in it. An empty
//     path answers the same way, because a machine whose configuration directory
//     could not be resolved has nowhere for the file to be.
//   - **Unreadable or malformed is refused, and the refusal names the path.**
//     This is the one that may not be swallowed. A player hand-edits this file,
//     so a missing comma is a thing that happens, and quietly showing them the
//     shipped sides instead would tell them their squads had vanished rather
//     than that their file has a typo in it. What the caller does with the
//     refusal is stop — the same treatment a --data directory that will not
//     parse already gets, which is what makes it unswallowable.
//   - **Illegal is refused on the SAME rule the game refuses one by**, which is
//     Squad.Take against the cast book: the call SaveSquad makes before it
//     writes and the call room's gate makes before it seats anybody. A
//     hand-edited file can name a character that does not exist, a stage off the
//     line, a skill the unit cannot carry or two units on one cell, and the cast
//     book already answers every one of those. A check written here would be a
//     second declaration of the legality rule.
//
// The whole file is refused rather than the offending side, for the reason
// placement.Parse refuses a whole file over one repeated id: a player told that
// four of their five sides loaded has been told their file works.
func (l *Library) PlayerSquads(path string) ([]placement.Squad, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	squads, err := placement.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, squad := range squads {
		if _, err := squad.Take(hex.SideAlly, l.characters); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return squads, nil
}

// SquadsOffered is the list a player picks from: the game's own sides with a
// player's own laid over the top.
//
// ⚠️ **Every id in the answer is unique, and that is correctness rather than
// tidiness.** A side is named by id wherever it is chosen — screen.Subject
// carries one and cmd/hexarena-tui turns it back into a row by walking the list
// for the first match — so two entries under one id is a reader pointing at the
// second and fielding the first. That is the failure this function exists to
// make impossible, and it is why the two lists are merged rather than
// concatenated.
//
// **A player's side wins the id it shares.** Three answers were defensible and
// this is the one chosen:
//
//   - *Refusing the clash* would be the loudest and is wrong for where the clash
//     comes from. The obvious way to start a squad file is to copy the shipped
//     one and change a number, which collides four times at once; and the game
//     may ship an `s05` next year that a player already has, so a refusal would
//     let the game's own data break a file it does not own.
//   - *Namespacing* keeps both, at the cost of showing a player an id they did
//     not write, in the file they wrote it in.
//   - *The player's winning* is what every configuration file already does — what
//     you wrote beats what shipped — and it costs only the shipped side of that
//     id, which is the one the player chose to replace.
//
// ⚠️ **The choice is visible on screen rather than only in this comment.** Every
// side out of a player's file is marked where it is drawn — the squad
// catalogue's rows and the join screen's chooser — through
// screen.Context.PlayerSquad. A shadowed id is therefore a row that says whose
// it is rather than a silent substitution, and that mark is what stops "the
// player's wins" being the silent collision no answer here may be.
//
// The game's order is kept and a shadowing side replaces its row **in place**,
// so the row a reader knew does not move the day they write a file; sides with
// new ids are appended in the order the player wrote them. Nothing here orders
// anything by ranging over a map.
func SquadsOffered(game, player []placement.Squad) []placement.Squad {
	at := make(map[string]int, len(player))
	for index, squad := range player {
		at[squad.ID] = index
	}
	laid := make([]bool, len(player))
	out := make([]placement.Squad, 0, len(game)+len(player))
	for _, squad := range game {
		if index, shadows := at[squad.ID]; shadows {
			out = append(out, player[index].Clone())
			laid[index] = true
			continue
		}
		out = append(out, squad.Clone())
	}
	for index, squad := range player {
		if laid[index] {
			continue
		}
		out = append(out, squad.Clone())
	}
	return out
}
