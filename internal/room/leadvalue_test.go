package room_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
)

// TestAlternatingTheLeadIsWorthLessToTheSideThatGetsIt is the measurement the
// alternation was deferred for want of, taken against the ordering it replaced.
//
// # What is measured, and why it is a gap rather than a rate
//
// ⚠️ **A one-way mirror rate is not a measurement here.** A squad against a copy
// of itself fought one way and its own reverse ought to sum to a thousand per
// mille, and above one unit a side it does not: the shipped s03, s04 and s05
// read 1119, 1153 and 1057 over 2000 seeds each. So neither arm on its own says
// what the order is worth. The **difference** between the two arms does, because
// whatever residual makes them fail to sum cancels out of a subtraction.
//
// (The residual is not unexplained any more, and it is not the turn order: the
// splash of a pattern is walked in absolute board directions while hex.Place
// puts the enemy down under a 180 degree rotation, so an authored formation does
// not catch the same neighbours on the two halves. → TODO.md, its own item.)
//
// So the figure is the gap between the same battles with the lead handed to the
// one side and to the other. Enlisting home's whole squad first, that gap was
// ±129.5, ±242.4, ±54.5, ±121.3 and ±54.0 per mille on the five shipped squads
// over 2000 seeds; alternating the lead it is ±76.4, ±89.4, ±42.8, ±93.3 and
// ±26.5 — smaller on every one of the five, and 45% smaller on the mean. At five
// a side, on squads composed for the size: ±46.0 became ±2.5 on one and ±125.7
// became ±69.3 on the other.
//
// This test is the small, fast statement of that: one squad, few enough seeds to
// run beside the rest of the package, asserting the direction rather than the
// figure. The table above is the record; what has to keep holding is that the
// alternation is not worth *more* to whoever leads than taking every tie was.
func TestAlternatingTheLeadIsWorthLessToTheSideThatGetsIt(t *testing.T) {
	dependencies := deps(t)
	squad := squadOf(t, dependencies.Characters, "mirror",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	alternated := openingRoster(t, squad, squad.Clone(), config(11, 1))

	ally, err := squad.Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the ally half: %v", err)
	}
	enemy, err := squad.Clone().Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the enemy half: %v", err)
	}
	homeWhole := append(append([]battle.Roster{}, ally...), enemy...)
	if sides(alternated) == sides(homeWhole) {
		t.Fatal("the room enlisted the two halves in the order it used to, so " +
			"there is nothing here to compare")
	}

	const seeds = 150
	whole := gapOverSeeds(t, dependencies, homeWhole, seeds)
	taking := gapOverSeeds(t, dependencies, alternated, seeds)
	t.Logf("over %d seeds: %s reads a gap of %d‰, %s reads %d‰",
		seeds, sides(homeWhole), whole, sides(alternated), taking)
	if taking >= whole {
		t.Errorf("alternating the lead is worth %d‰ to the side that leads first "+
			"and enlisting home's squad whole is worth %d‰: the alternation has to "+
			"be worth less than the ordering it replaced, or it is buying nothing",
			taking, whole)
	}
}

// gapOverSeeds is what the given roster order is worth to the half listed first,
// in parts per thousand, read as the difference between that order and its own
// reflection over the same seeds.
//
// Both arms are fought over the *same* seeds, because two arms over different
// seeds cancel nothing — the same rule forge.FightSquads is built on.
func gapOverSeeds(t *testing.T, dependencies room.Deps, roster []battle.Roster, seeds int) int {
	t.Helper()
	forward := winsForAlly(t, dependencies, roster, seeds)
	backward := winsForAlly(t, dependencies, reflected(t, roster), seeds)
	return forward - backward
}

// reflected is a mirror's roster with each unit replaced by its opposite number:
// the same order, played by the other side. It is how the second arm of the swap
// is built for an order that interleaves the halves, which cannot be produced by
// listing one squad before the other.
func reflected(t *testing.T, roster []battle.Roster) []battle.Roster {
	t.Helper()
	byID := make(map[string]battle.Roster, len(roster))
	for _, entry := range roster {
		byID[entry.ID] = entry
	}
	out := make([]battle.Roster, 0, len(roster))
	for _, entry := range roster {
		opposite, known := byID[opposingID(entry.ID)]
		if !known {
			t.Fatalf("unit %q has no opposite number, so this roster is not a mirror",
				entry.ID)
		}
		out = append(out, opposite)
	}
	return out
}

// opposingID is a unit id with its half exchanged. The ids a squad is fielded
// under carry the side, which is what makes a mirror's two halves nameable from
// each other at all.
func opposingID(id string) string {
	if rest, cut := strings.CutPrefix(id, "ally."); cut {
		return "enemy." + rest
	}
	if rest, cut := strings.CutPrefix(id, "enemy."); cut {
		return "ally." + rest
	}
	return id
}

// winsForAlly is the ally half's share of the battles that resolved, in parts
// per thousand. A battle that never resolved is left out of both halves of the
// ratio rather than counted as half a win: it measured nothing.
func winsForAlly(t *testing.T, dependencies room.Deps, roster []battle.Roster, seeds int) int {
	t.Helper()
	wins, decided := 0, 0
	for seed := 1; seed <= seeds; seed++ {
		fight, err := battle.New(dependencies.Books, uint64(seed), roster)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(room.DefaultTurnCap); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		winner, resolved := fight.Winner()
		if !resolved {
			continue
		}
		decided++
		if winner == hex.SideAlly {
			wins++
		}
	}
	if decided == 0 {
		t.Fatal("not one battle resolved, so this arm measures nothing")
	}
	return 1000 * wins / decided
}
