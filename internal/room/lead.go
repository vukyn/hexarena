package room

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// A contested speed group is the units of BOTH sides sharing one speed, and the
// lead of each one alternates.
//
// # Why the roster order is a balance decision at all
//
// atb.Queue.Add takes seq off a counter, enlist calls Add once per unit, and
// battle.New calls enlist in the order of the roster slice it was handed. seq is
// the last tie-break in the turn order, so **the caller decides who wins a speed
// tie** — with no core change and no golden moved. And a tie is not won once: a
// unit that acts first at the same speed keeps acting first every round until
// something moves a speed, because Next sets now to the actor's own next and
// leaves the other unit standing at the same instant.
//
// Enlisting the home squad whole and the away squad after it — which is what
// this room did until now, and what forge.FightSquads still does on purpose —
// hands every one of those ties to one side. Measured on the five shipped
// squads, each against a copy of itself over 2000 seeds, as the gap between the
// two arms of the swap:
//
//	squad   home enlisted whole   leads alternating
//	s01              ±129.5‰               ±76.4‰
//	s02              ±242.4‰               ±89.4‰
//	s03               ±54.5‰               ±42.8‰
//	s04              ±121.3‰               ±93.3‰
//	s05               ±54.0‰               ±26.5‰
//
// Every squad shrank, by 45% on the mean. The gap between the arms is read
// rather than either arm on its own, because a one-way rate is not a
// measurement here — see the note on the residual in TODO.md.
//
// # What it does not do
//
// ⚠️ **It does not make a battle even, and it is not offered instead of the
// swap.** A match already fights both ways round — Config.HomeFor — and that is
// what cancels the residual as well as the tie. This is worth having on top of
// it, per battle, and the numbers above say what "on top" is worth.
//
// ⚠️ **The lead alternates per PAIR, running across the groups**, not per group.
// The two readings only differ when a group holds more than one unit a side,
// which a mirror of distinct speeds never produces; per pair is the finer of the
// two and leaves the two sides' lead counts differing by at most one over the
// whole roster.
//
// # The speed it groups on is the ENLISTED speed, not the authored one
//
// ⚠️ **A roster entry's Stats are not what the queue reads.** enlist applies the
// composition bonuses and the passives before queue.Add reads a speed, so a
// bonus that grants a permanent speed status moves it: the shipped s03 fields a
// Magnezone authored at 110 that enlists at **117**. Grouping on the authored
// line would put two units that genuinely tie into different groups and two that
// do not into one — the first loses a tie that needed alternating, the second
// spends a step of the toggle on a group where the order changes nothing, which
// shifts the phase of every group after it.
//
// So the speeds are read from a battle built for the purpose and thrown away.
// That costs one extra battle.New per battle, and there is no cheaper honest
// answer: the effective speed is not a function of the roster entry alone.
func alternateContested(books battle.Books, seed uint64, roster []battle.Roster) ([]battle.Roster, error) {
	speeds, err := enlistedSpeeds(books, seed, roster)
	if err != nil {
		return nil, err
	}
	type group struct {
		leads  []battle.Roster
		follow []battle.Roster
	}
	order := make([]int64, 0, len(roster))
	held := make(map[int64]*group, len(roster))
	for _, entry := range roster {
		speed := speeds[entry.ID]
		found, seen := held[speed]
		if !seen {
			found = &group{}
			held[speed] = found
			order = append(order, speed)
		}
		// hex.SideAlly is home — sideOf is where that is decided, and begin
		// enlists home's squad first for it — so the ally half is the half that
		// leads the first contested pair. It is written as the side rather than
		// taken as a parameter because there is one caller and one answer: a
		// parameter here would be a knob nothing turns, and which side is home
		// is already a decision Config.HomeFor made for the whole battle.
		if entry.Side == hex.SideAlly {
			found.leads = append(found.leads, entry)
			continue
		}
		found.follow = append(found.follow, entry)
	}
	out := make([]battle.Roster, 0, len(roster))
	// The side that leads the next pair, flipped every pair rather than every
	// group, and carried across the groups so the count comes out even.
	leading := true
	for _, speed := range order {
		found := held[speed]
		// One side only: nothing is contested, so nothing alternates, the order
		// it was authored in stands, and the group spends none of the toggle —
		// nobody won a tie here, so nobody owes the next pair a turn of it. One
		// of the two appends is over an empty slice; which one is not a fact
		// worth branching on.
		if len(found.leads) == 0 || len(found.follow) == 0 {
			out = append(out, found.leads...)
			out = append(out, found.follow...)
			continue
		}
		for i := 0; i < len(found.leads) || i < len(found.follow); i++ {
			pair := [2][]battle.Roster{found.leads, found.follow}
			if !leading {
				pair = [2][]battle.Roster{found.follow, found.leads}
			}
			for _, half := range pair {
				if i < len(half) {
					out = append(out, half[i])
				}
			}
			leading = !leading
		}
	}
	return out, nil
}

// enlistedSpeeds is every unit's speed as the turn queue will read it: after the
// composition bonuses and the passives, which is the only reading that groups
// the units that actually tie. → alternateContested, on why the authored line
// will not do.
func enlistedSpeeds(books battle.Books, seed uint64, roster []battle.Roster) (map[string]int64, error) {
	probe, err := battle.New(books, seed, roster)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(roster))
	for _, entry := range roster {
		unit, known := probe.Unit(entry.ID)
		if !known {
			return nil, fmt.Errorf("unit %q was enlisted and then could not be found", entry.ID)
		}
		out[entry.ID] = probe.Stats(unit)[progression.Speed]
	}
	return out, nil
}
