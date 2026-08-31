package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/seed"
)

// chargeSeeds is how many seeds each kit is fought over, both ways round. The
// readings this was written against sit on three hundred battles a kit.
const chargeSeeds = 150

// accumulating and bursting are the two kits under comparison, and they are the
// design record for what the counter was added to make possible.
//
// The first spends turns putting a counter on and then cashes it one stack at a
// time; the second brings the four heaviest single blows the learnset offers.
// Same character, same stats, same trait, same squad around it, same opposition,
// same seeds — the four skills are the whole of the difference.
var (
	accumulating = []string{"charge_beam", "magnetise", "electro_ball", "spark"}
	bursting     = []string{"zap_cannon", "thunderbolt", "flash_cannon", "discharge"}
)

// TestAccumulatingIsAWayOfFightingRatherThanASlowerOne is what the charge
// mechanic was built for, held as a measurement rather than as an intention.
//
// Two things have to be true at once and neither is enough alone. The damage has
// to arrive **differently** — more pieces, each smaller — or the counter is
// flavour on a kit that fights like every other kit. And it has to arrive well
// enough to be worth choosing: a playstyle that reads a third of the win rate of
// the one beside it is not an option, it is a trap with a story attached.
//
// ⚠️ **Both halves were false when this was first run, and the fixing of them is
// the whole content of the feature.** The first cut read 6 per mille against 366
// — not slower, simply losing — and the cause was in the rating rather than in
// the data: a pile of counters was priced at one stack times its height, so three
// stacks over two cells looked like three whole strikes and the opponent spent
// its opening turn on a skill that dealt no damage. A consumer cashes one stack a
// cast and a battle is about fifteen turns long, so the stacks near the top of a
// pile are speculative; halving each one against the one below it took the same
// kit from 6 to 110 without a number in the data moving. The rest came from the
// skills themselves, which were priced as though the shape were free — see
// TestASpreadConsumerIsBoundedByTheShapeItBuys for what the shape is actually
// worth, which is much less than it looks.
func TestAccumulatingIsAWayOfFightingRatherThanASlowerOne(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	opposition := aSquadOf("plain", aThirdMember("pokemon.machop",
		"rock_throw", "body_slam", "cross_chop", "vital_throw"))

	slow := readKit(t, books, characters, accumulating, opposition)
	fast := readKit(t, books, characters, bursting, opposition)
	t.Logf("accumulating: %d per mille, %d hits of %d each, %d stacks put on, %d spread",
		slow.rate(), slow.hits, slow.each(), slow.charged, slow.spread)
	t.Logf("bursting:     %d per mille, %d hits of %d each",
		fast.rate(), fast.hits, fast.each())

	// The counter has to have been used as a counter. Without this the whole
	// comparison could hold with the engine never charging anybody — two kits of
	// ordinary skills, one of them weaker — and the test would be measuring the
	// power numbers rather than the mechanic.
	if slow.spread == 0 || slow.charged == 0 {
		t.Fatalf("the accumulating kit put %d stacks on and spread %d times, so nothing here measured the counter at all",
			slow.charged, slow.spread)
	}

	if slow.hits <= fast.hits {
		t.Errorf("the accumulating kit landed %d blows against the bursting kit's %d: "+
			"the damage is not arriving in more pieces, so the counter is flavour on a kit that fights the same way",
			slow.hits, fast.hits)
	}
	if slow.each() >= fast.each() {
		t.Errorf("the accumulating kit hits for %d a blow against the bursting kit's %d: "+
			"the pieces are not smaller, so there is no trade here to choose between",
			slow.each(), fast.each())
	}
	// A floor rather than parity. What is held is that accumulating is a way of
	// fighting somebody chose, not that it is the same way with different words —
	// and the figure moves with every character in the squad around it, so a
	// tighter band would be a reading rather than a claim. Three fifths is far
	// under the 293 against 366 it was written at and far over the 6 it started at.
	if floor := fast.rate() * 3 / 5; slow.rate() < floor {
		t.Errorf("the accumulating kit reads %d per mille against the bursting kit's %d, under the %d held: "+
			"a playstyle this far behind the one beside it is a trap rather than an option",
			slow.rate(), fast.rate(), floor)
	}
}

// kitReading is one kit's record and the shape of the damage it dealt.
//
// The hits counted are the third member's own, not the squad's: the two fixed
// carriers beside it fight identically in both squads, and folding their blows in
// would dilute exactly the difference this is measuring.
type kitReading struct {
	wins, losses     int
	hits             int
	damage           int64
	charged, spread  int
	endless, decided int
}

func (r kitReading) rate() int {
	if r.decided == 0 {
		return 0
	}
	return r.wins * 1000 / r.decided
}

func (r kitReading) each() int64 {
	if r.hits == 0 {
		return 0
	}
	return r.damage / int64(r.hits)
}

// readKit fights one kit against the opposition from both slots over the same
// seeds, and reads the log rather than only the result.
func readKit(t *testing.T, books battle.Books, characters *cast.Book,
	kit []string, away placement.Squad) kitReading {
	t.Helper()
	home := aSquadOf("probe", aThirdMember("pokemon.magnemite", kit...))
	var out kitReading
	for n := 1; n <= chargeSeeds; n++ {
		for _, swapped := range []bool{false, true} {
			first, second := home, away
			mine, subject := hex.SideAlly, "ally.third"
			if swapped {
				first, second = away, home
				mine, subject = hex.SideEnemy, "enemy.third"
			}
			ally, err := first.Take(hex.SideAlly, characters)
			if err != nil {
				t.Fatalf("field %s: %v", first.ID, err)
			}
			foe, err := second.Take(hex.SideEnemy, characters)
			if err != nil {
				t.Fatalf("field %s: %v", second.ID, err)
			}
			fight, err := battle.New(books, uint64(n), append(ally, foe...))
			if err != nil {
				t.Fatalf("seed %d: %v", n, err)
			}
			if _, err := fight.RunToEnd(menderTurnLimit); err != nil {
				t.Fatalf("seed %d: %v", n, err)
			}
			if !fight.Finished() {
				out.endless++
				continue
			}
			for _, event := range fight.Drain() {
				if event.Actor != subject {
					continue
				}
				switch event.Kind {
				case battle.Damaged:
					out.hits++
					out.damage += event.Amount
				case battle.StatusApplied:
					if event.Status == "charge" {
						out.charged += event.Stacks
					}
				case battle.Spread:
					out.spread++
				}
			}
			winner, decided := fight.Winner()
			if !decided {
				continue
			}
			out.decided++
			if winner == mine {
				out.wins++
			} else {
				out.losses++
			}
		}
	}
	if out.endless > 0 {
		t.Fatalf("%d of %d battles never finished, so a rate over the rest reads a different question",
			out.endless, chargeSeeds*2)
	}
	return out
}
