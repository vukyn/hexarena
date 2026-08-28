package forge

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
)

// SquadReport is what two squads came to over a number of seeds.
//
// It is read from the **home** squad throughout, the way a SparReport is read
// from its challenger: a rate needs a subject, and a report that reported both
// would be two numbers that always add to a thousand.
type SquadReport struct {
	Home, Away placement.Squad
	// Seeds is how many battles each half was fought over, so a report counts
	// twice this many.
	Seeds int
	// AsAlly and AsEnemy are the home squad's tallies from each half of the
	// swap, kept apart rather than folded because the difference between them is
	// a finding in its own right: it is what the side itself is worth, and on
	// this board that is not nothing.
	AsAlly, AsEnemy Tally
	// Turns is the median length of a battle that finished, in turns.
	Turns int
}

// Total is the home squad's record across both halves.
func (r SquadReport) Total() Tally { return r.AsAlly.add(r.AsEnemy) }

// Rate is the home squad's share in parts per thousand.
func (r SquadReport) Rate() int { return r.Total().Rate() }

// Mirror reports whether a squad was fought against itself, which is a control
// rather than a matchup: it reads exactly even by construction, and a figure
// that did not would mean the swap was not cancelling what it is there to
// cancel.
func (r SquadReport) Mirror() bool { return r.Home.ID == r.Away.ID }

// FightSquads measures one squad against another over seeds 1..n, both ways
// round.
//
// **Both ways round is not a refinement, it is the measurement.** The engine
// enlists in roster order and breaks a tie in the turn queue by it, and the two
// halves of the board are not interchangeable — a squad placed as the ally is
// listed first and moves first on a tie. Fighting one arrangement would report
// that advantage as though it belonged to the squad. So each seed is fought
// twice, with the squads swapped, and both halves run the *same* seeds: halves
// fought over different seeds cancel nothing, they are two measurements of two
// different things.
//
// A refusal comes back as an error rather than as a report with a hole in it.
// Every seed would refuse for the same reason — a roster is checked before a die
// is rolled — so fighting the rest would be a slower way to be told once.
func (l *Library) FightSquads(home, away string, seeds int) (SquadReport, error) {
	if seeds < 1 {
		return SquadReport{}, fmt.Errorf("a run over %d battles measures nothing", seeds)
	}
	first, err := l.squad(home)
	if err != nil {
		return SquadReport{}, err
	}
	second, err := l.squad(away)
	if err != nil {
		return SquadReport{}, err
	}
	report := SquadReport{Home: first, Away: second, Seeds: seeds}
	books := l.Books()
	lengths := make([]int, 0, 2*seeds)
	for _, arrangement := range []struct {
		ally, enemy placement.Squad
		// mine is the side the home squad stands on in this arrangement, because
		// a Result is always read from the home squad.
		mine hex.Side
		into *Tally
	}{
		{first, second, hex.SideAlly, &report.AsAlly},
		{second, first, hex.SideEnemy, &report.AsEnemy},
	} {
		roster, err := arrangement.ally.Take(hex.SideAlly, l.characters)
		if err != nil {
			return SquadReport{}, err
		}
		facing, err := arrangement.enemy.Take(hex.SideEnemy, l.characters)
		if err != nil {
			return SquadReport{}, err
		}
		roster = append(roster, facing...)
		for seed := 1; seed <= seeds; seed++ {
			result, turns, _, err := fight(books, roster, arrangement.mine, uint64(seed))
			if err != nil {
				return SquadReport{}, err
			}
			arrangement.into.count(result)
			if result != Endless {
				lengths = append(lengths, turns)
			}
		}
	}
	report.Turns = median(lengths)
	return report, nil
}

func (l *Library) squad(id string) (placement.Squad, error) {
	for _, held := range l.squads {
		if held.ID == id {
			return held.Clone(), nil
		}
	}
	return placement.Squad{}, fmt.Errorf("no squad is called %q", id)
}
