package room

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/wire"
)

// Verdict is how a **match** ended, and it is deliberately not called an outcome.
//
// ⚠️ Nothing here is added to battle.Outcome and nothing ever should be. That
// enum is a core type read by --verify and by every renderer, and it answers a
// different question: how a *battle* ended. A departure is not a way a battle
// can end — the battle it interrupted was in a perfectly ordinary state when the
// room stopped asking about it — so it lives here, in the room's own record, and
// TestADepartureAddsNothingToTheBattlesOutcomes holds battle.OutcomeCount
// against the number it has always had.
//
// The two words are kept apart on purpose. A reader meeting `Outcome` twice in
// one file, once meaning the battle and once the match, is a reader one step
// from writing `battle.Outcome(result.Verdict)`.
type Verdict uint8

const (
	// VerdictUnfinished is a match still being played, and the zero value: a
	// room that has answered nothing yet has no verdict rather than a draw.
	VerdictUnfinished Verdict = iota
	// VerdictWon is a seat that took the series by the rule the room configured
	// — more battles than the rest of the series can still take back.
	VerdictWon
	// VerdictDrawn is every battle of the series fought with no seat ahead. It
	// needs no invented metric to break it: an odd series is offered precisely
	// so that a tie-break is never something the room has to make up, and a
	// series that drew every battle drew.
	VerdictDrawn
	// VerdictAbandoned is a match nobody played out, which is what a departure
	// leaves behind.
	//
	// ⚠️ **It is not a win, not a draw and not a forfeit.** Nobody is charged
	// with anything: leaving announces and nothing more, and a timeout does not
	// even reach here — it passes the turn and the match carries on. So Winner is
	// the zero Seat, Departed names who went away, and the standing is whatever
	// the battles already fought had made it.
	//
	// ⚠️ The concept this replaced was VerdictForfeited, with a Forfeit beside it
	// saying which of two routes had been taken. Both routes are gone: a
	// departure costs nothing (→ README.md § PvP over a LAN, where that cost is
	// stated) and three missed allowances cost nothing either, because a player
	// who walks away from the keyboard loses on the board — the opponent keeps
	// acting and kills the passing units — or, if both walk away, the turn cap
	// draws it. The forfeit was carrying nothing the board does not already
	// carry.
	//
	// Declared last, which is the rule this enum shares with the ones on the
	// wire even though nothing serialises it yet: the day it is written to a
	// file, appending will not reinterpret anything already written.
	VerdictAbandoned
)

// VerdictCount is the number of verdicts, so a test can walk them rather than
// range over the table of names and ask it whether it holds what it holds.
const VerdictCount = int(VerdictAbandoned) + 1

var verdictNames = [VerdictCount]string{
	VerdictUnfinished: "unfinished",
	VerdictWon:        "won",
	VerdictDrawn:      "drawn",
	VerdictAbandoned:  "abandoned",
}

func (v Verdict) String() string {
	if int(v) >= VerdictCount {
		return fmt.Sprintf("verdict(%d)", uint8(v))
	}
	return verdictNames[v]
}

// Over reports whether the match is finished, which is every verdict but the
// zero one.
func (v Verdict) Over() bool { return v != VerdictUnfinished && int(v) < VerdictCount }

// Result is the match, finished or not.
type Result struct {
	Verdict Verdict
	// Winner is the seat that took the match, and is the zero Seat for a draw,
	// for a match still being played **and for an abandoned one** — nobody wins
	// a match nobody played out. Seat.Valid is the question to ask.
	Winner wire.Seat
	// Departed is the seat that went away, and the zero Seat for a match that
	// was played out. It is the fact an abandoned match needs and cannot derive:
	// it is not simply "the other one" the day a room holds spectators.
	//
	// ⚠️ It was called Loser, beside a Forfeit saying why. Nobody loses by
	// leaving now, so the word had to go with the concept — a field called Loser
	// on a verdict where nobody lost is exactly the stale wording this
	// repository keeps a list of.
	Departed wire.Seat
	// Wins is how many battles each seat took, indexed the way the room indexes
	// its seats. Draws count for neither, which is why the two need not add up
	// to the number of battles played.
	Wins [seatCount]int
	// Battles is the battles actually fought, which is fewer than the series
	// length whenever a seat took an unbeatable lead early.
	Battles int
}

// BattleResult is one battle of the series as the room recorded it.
//
// It carries the engine's own battle.Outcome unchanged and adds nothing to it.
// What it adds *beside* it is the room's half of the story — which seat was
// home, what seed the battle was derived to, and whether the turn cap stopped it
// — none of which is a fact about how a battle ends.
type BattleResult struct {
	// Battle is which battle of the series this was, counting from one.
	Battle int
	// Home is the seat enlisted first, and therefore the one on hex.SideAlly.
	// It is the sixty-point fact: seq is assigned in roster order and is the
	// last tie-break in the turn queue, so the home seat wins a speed tie.
	Home wire.Seat
	// Seed is the seed this battle was fought from, derived from the room's one
	// seed and this battle's index. → Config.SeedFor.
	Seed uint64
	// Winner is the seat that won, and the zero Seat for any of the draws.
	Winner wire.Seat
	// Outcome is the engine's, exactly as the engine reported it, and it is
	// battle.Undecided for a battle the turn cap stopped — because the engine
	// did not conclude anything about that battle and a room writing an outcome
	// the engine never produced would be a second reading of how a battle ends.
	// Capped says so instead.
	Outcome battle.Outcome
	// Turns is how many turns the engine opened, skipped ones included.
	Turns int
	// Capped is the turn cap having stopped the battle rather than the engine
	// finishing it. It counts as a draw in the standing, which is what "the
	// outcome already carries the draws" means: the series needs to know that
	// neither seat won, and it needs no new way for a battle to end to know it.
	Capped bool
}

// Draw reports whether the battle went to neither seat, which every ending but
// a victory is: annihilation, stalemate, and the turn cap.
func (b BattleResult) Draw() bool { return !b.Winner.Valid() }
