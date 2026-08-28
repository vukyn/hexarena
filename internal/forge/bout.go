package forge

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// Brain builds a Chooser for one battle, because a Chooser closes over the
// battle it decides in.
//
// battle.Suggest is a method on a *Battle, so "a rating" is not a value until a
// battle exists to bind it to. A Brain is that binding made a type, which is what
// lets a caller name two ratings before either has a board to play on.
type Brain func(*battle.Battle) battle.Chooser

// Suggesting is the rating the game ships, as a Brain.
func Suggesting(fight *battle.Battle) battle.Chooser { return fight.Suggest }

// FirstUsable takes the first option it can use, in kit order, and aims it at the
// first cell offered. It is what Suggest was before any pricing landed.
//
// ⚠️ **This is a ruler, not an opponent, and it may never be improved.** Every
// claim made about a rating in this repository is of the shape "Suggest beats
// FirstUsable by X permille over N seeds", and X is only comparable across the
// repository's history if the thing it is measured against never moves. A better
// FirstUsable would silently re-scale every figure ever quoted against it, and
// nothing would report the disagreement — the old numbers would simply become
// wrong. If a second, stronger baseline is wanted, add one beside this; do not
// sharpen this one.
func FirstUsable(*battle.Battle) battle.Chooser {
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		for _, option := range prompt.Options {
			if !option.Available() {
				continue
			}
			return battle.Choice{Skill: option.Skill, Aim: option.Aims[0]}, true
		}
		return battle.Choice{}, false
	}
}

// BoutReport is what a head-to-head came to.
//
// Challenger and Incumbent are the two sides of the same battles, so one's wins
// are the other's losses and a reader may check the report against itself. Band
// is the two-sigma width of the rate at this many seeds: a Challenger rate inside
// `500 ± Band` is not a finding.
type BoutReport struct {
	Seeds int
	Band  int

	Challenger Tally
	Incumbent  Tally

	// Turns is the median length of a decided battle, and it belongs beside the
	// rate rather than under it. A rating that wins no more often but finishes
	// sooner has still improved, and two ratings that have both improved make a
	// battle longer — so a rate that did not move over turns that did is a
	// result, not a null one.
	Turns int

	// Roster is the board the figure was taken on, carried so that a reader is
	// never left to guess which one it was.
	Roster []battle.Roster
}

// Rate is the challenger's share in parts per thousand.
func (r BoutReport) Rate() int { return r.Challenger.Rate() }

// Bout fights two ratings against each other over the given roster and reports
// the challenger's share.
//
// # What it is
//
// Every seed is fought twice: once with the challenger driving the ally squad and
// the incumbent driving the enemy, and once with the two exchanged. Both halves
// run the same seeds, and the challenger's wins are counted over all 2N battles.
// What is left when the board has been fought from both ends is the difference
// between the two ratings, and it is read as a deviation from an even split.
//
// # The control is exact, not statistical
//
// When the two Brains are the same, the two arrangements are the *same battle* —
// identical seed, identical roster, identical decisions — so they produce the same
// winner, and the challenger holds the winning side in exactly one of the two.
// Over N seeds that is N wins and N losses: **exactly 500‰, by construction**. A
// draw is counted half to each side and lands in both arrangements, so it does not
// disturb it either.
//
// So Bout runs that control before it measures anything and refuses to print a
// figure unless it comes out exactly even. ⚠️ It is asserted **at exactly 500‰ and
// never within a band** — a band is what a leak would hide behind, and a harness
// that is allowed to be nearly even is a harness that cannot tell a real edge from
// its own bookkeeping.
//
// # Why the roster does not have to be symmetric
//
// The board's asymmetry is fought from both ends and cancels exactly — that is the
// whole content of the paragraph above. So this measures on the **shipped** roster,
// which is the board every other figure in this repository is quoted on, rather
// than on a mirror built to be fair.
//
// ⚠️ CLAUDE.md bans a mirror roster, and that ban is about a **data** change — it
// says the shipped cast may not be flattened into two copies of itself to make a
// number easier to read. It does not apply here: nothing is authored, the roster
// is a parameter, and the mirror is in the *procedure* rather than in the data.
//
// # What it accepts
//
// ⚠️ **Roll drift.** The two arrangements share one *rng.Source per battle, so the
// first decision the two ratings disagree on re-scrambles every draw after it. It
// **biases nothing** — both arrangements fight the same seeds — and fixing it would
// mean an internal/core change to make a measurement prettier. This is the same
// note Weigh carries; see CLAUDE.md § Pricing one number.
//
// The roster is a parameter rather than read from internal/seed, so that this
// package keeps not importing that one: a tool that measured the embedded copy
// would keep reporting on the cast as it was at the last build.
func Bout(books battle.Books, roster []battle.Roster, challenger, incumbent Brain, seeds int) (BoutReport, error) {
	if seeds < 1 {
		return BoutReport{}, fmt.Errorf("a bout needs at least one seed, not %d", seeds)
	}
	if challenger == nil || incumbent == nil {
		return BoutReport{}, fmt.Errorf("a bout needs two ratings, and one of them is missing")
	}

	control, err := headToHead(books, roster, challenger, challenger, seeds)
	if err != nil {
		return BoutReport{}, fmt.Errorf("the control: %w", err)
	}
	if err := control.decisive("the control"); err != nil {
		return BoutReport{}, err
	}
	if rate := control.challenger.Rate(); rate != scale.Base/2 {
		return BoutReport{}, fmt.Errorf("the control comes to %d rather than an exactly even %d "+
			"(%+v): a rating against itself is the same battle fought from both ends, so anything "+
			"else is the harness and not the ratings, and no figure it prints could be trusted",
			rate, scale.Base/2, control.challenger)
	}

	fought, err := headToHead(books, roster, challenger, incumbent, seeds)
	if err != nil {
		return BoutReport{}, err
	}
	if err := fought.decisive("the bout"); err != nil {
		return BoutReport{}, err
	}
	return BoutReport{
		Seeds: seeds, Band: band(seeds),
		Challenger: fought.challenger, Incumbent: fought.incumbent,
		Turns: median(fought.lengths), Roster: roster,
	}, nil
}

// sides is one head-to-head's counts, from both ends.
type sides struct {
	challenger Tally
	incumbent  Tally
	// lengths holds every decided battle, endless ones left out, so the median
	// is a length and not a mixture of a length and a limit.
	lengths []int
}

// decisive refuses a run too much of which never resolved.
//
// Two better opponents stalling each other is a real risk of any change to a
// rating — a squad that has learnt to guard and heal is a squad that can hold a
// board for ever — and it must not be filed as a draw. It is the same fifth Weigh
// refuses a row at, because "the rate is taken over what finished" is the same
// sentence in both tools.
func (s sides) decisive(what string) error {
	total := s.challenger.Battles()
	if s.challenger.Endless*endlessShare > total {
		return fmt.Errorf("%s left %d of %d battle(s) undecided, which is more than a fifth: "+
			"the rate is taken over what finished, and two ratings stalling each other is "+
			"exactly what a change to one of them can cause",
			what, s.challenger.Endless, total)
	}
	return nil
}

// headToHead fights every seed from both ends and counts the challenger.
func headToHead(books battle.Books, roster []battle.Roster, challenger, incumbent Brain, seeds int) (sides, error) {
	counted := sides{lengths: make([]int, 0, 2*seeds)}
	// The challenger drove the ally squad, then drove the enemy one. Both halves
	// run the same seeds: a half fought over different seeds from the other would
	// not cancel anything, it would be two measurements of two different things.
	for _, arrangement := range []struct {
		allied, enemy Brain
		// mine is the side the challenger is driving in this arrangement,
		// because a Result is always read from the challenger.
		mine hex.Side
	}{
		{challenger, incumbent, hex.SideAlly},
		{incumbent, challenger, hex.SideEnemy},
	} {
		for seed := 1; seed <= seeds; seed++ {
			result, turns, err := clash(books, roster, uint64(seed), arrangement.allied,
				arrangement.enemy, arrangement.mine)
			if err != nil {
				// The first refusal ends the run. Every seed would refuse for the
				// same reason — a roster is checked before a die is rolled — so
				// fighting the rest would be a slower way to be told once.
				return sides{}, err
			}
			counted.challenger.count(result)
			counted.incumbent.count(opposite(result))
			if result != Endless {
				counted.lengths = append(counted.lengths, turns)
			}
		}
	}
	return counted, nil
}

// clash runs one battle with a rating per side and reports how it ended for the
// given side and how long it took.
//
// It is the counterpart of fight, and the difference is the whole point of this
// file: fight plays both sides with the rating the game ships, so it measures the
// data, while this plays a different rating on each side, so it measures the
// rating.
func clash(books battle.Books, roster []battle.Roster, seed uint64,
	allied, enemy Brain, mine hex.Side) (Result, int, error) {
	fought, err := battle.New(books, seed, roster)
	if err != nil {
		return Drawn, 0, err
	}
	fought.Begin()
	turns, err := fought.RunToEndWith(sparTurnLimit, sided(fought, allied(fought), enemy(fought)))
	if err != nil {
		return Drawn, turns, err
	}
	return outcome(fought, mine), turns, nil
}

// sided is one Chooser per side, dispatched on whoever was prompted.
//
// The engine hands a Chooser a prompt and not a side, deliberately, so this is
// where a bout puts the two ratings on opposite ends of the board. It reads the
// unit rather than the roster because a summon is on neither: a copy put down
// mid-battle has to be decided for by whichever rating called it up, and looking
// its id up in the roster would find nothing.
//
// A unit the battle does not know is not a state that can happen — the prompt
// came from the battle — but it declines rather than reaching into a nil, so a
// bug of that shape becomes a passed turn instead of a panic.
func sided(fight *battle.Battle, allied, enemy battle.Chooser) battle.Chooser {
	return func(prompt *battle.Prompt) (battle.Choice, bool) {
		unit, known := fight.Unit(prompt.Unit)
		if !known {
			return battle.Choice{}, false
		}
		if unit.Side == hex.SideAlly {
			return allied(prompt)
		}
		return enemy(prompt)
	}
}

// outcome reads how a finished battle went for one side.
//
// It is one function rather than a copy in each caller because "which of these two
// was better" has to be the same question in a spar and in a bout: a second
// reading would be free to call a mutual kill a draw in one tool and a loss in the
// other, and the two reports would disagree about a battle they both watched.
func outcome(fought *battle.Battle, mine hex.Side) Result {
	if !fought.Finished() {
		return Endless
	}
	winner, decided := fought.Winner()
	switch {
	case !decided:
		return Drawn
	case winner == mine:
		return Won
	default:
		return Lost
	}
}

// opposite is the same result read from the other side of the board. A draw and
// an endless battle are the same answer to both.
func opposite(result Result) Result {
	switch result {
	case Won:
		return Lost
	case Lost:
		return Won
	default:
		return result
	}
}
