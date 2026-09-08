package room_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// openingRoster is the roster a room actually fought a battle with, read off the
// wire.Start it handed the host.
//
// It is read from the wire rather than from anything inside the room on purpose:
// the roster order is the whole subject here, and wire.Start is where a client
// learns it. A client that built its mirror from a different order would produce
// a different battle, so this is also the only order a test may assert about.
func openingRoster(t *testing.T, host, guest placement.Squad, cfg room.Config) []battle.Roster {
	t.Helper()
	opened := newRoom(t, cfg)
	if _, _, err := opened.Join(hello(t, host, "Host")); err != nil {
		t.Fatalf("the host joins: %v", err)
	}
	_, out, err := opened.Join(hello(t, guest, "Guest"))
	if err != nil {
		t.Fatalf("the guest joins: %v", err)
	}
	for _, message := range out {
		start, isStart := message.Body.(wire.Start)
		if !isStart || message.To != wire.SeatHost {
			continue
		}
		return start.Roster
	}
	t.Fatal("the second join opened no battle")
	return nil
}

// sides is the roster as a readable line of sides, which is what an alternation
// is a claim about: "AEAE" is one side leading every pair, "AEEA" is the lead
// changing hands.
func sides(roster []battle.Roster) string {
	var out strings.Builder
	for _, entry := range roster {
		if entry.Side == hex.SideAlly {
			out.WriteByte('A')
			continue
		}
		out.WriteByte('E')
	}
	return out.String()
}

// TestTheLeadOfEachContestedSpeedGroupAlternates is the property, on the fixture
// that makes every group contested and every group a pair: a mirror.
//
// Two copies of one squad tie at every speed, so the roster is three contested
// groups of one unit a side, and the lead of each has to change hands. Enlisting
// the home squad whole reads AAAEEE and hands all three to the host; what is
// asserted is AEEAAE — the host leading the first pair because home is the seat
// the series already alternates, and the lead changing at every pair after it.
func TestTheLeadOfEachContestedSpeedGroupAlternates(t *testing.T) {
	characters := deps(t).Characters
	squad := squadOf(t, characters, "mirror",
		"pokemon.bulbasaur", "pokemon.machop", "pokemon.gastly")
	roster := openingRoster(t, squad, squad.Clone(), config(11, 1))
	if got, want := sides(roster), "AEEAAE"; got != want {
		t.Errorf("a mirror was enlisted %s, want %s: the lead of a contested speed "+
			"group has to change hands at every pair, and %s is one side taking all "+
			"three", got, want, got)
	}
}

// TestAnUncontestedSpeedGroupSpendsNoneOfTheAlternation is the other half of the
// rule and the one a fixture of distinct speeds cannot see.
//
// A speed only one side holds is not a tie: nobody wins it, so nothing about it
// alternates and it must not move the turn of whoever leads the next contested
// group. A version that flipped the lead on every group rather than on every
// contested **pair** passes the mirror above and fails here, because the two
// fixtures differ in exactly that.
func TestAnUncontestedSpeedGroupSpendsNoneOfTheAlternation(t *testing.T) {
	characters := deps(t).Characters
	// gastly and cleffa are on neither side's opposite number, so their two
	// speeds are uncontested; machop and bulbasaur are on both.
	host := squadOf(t, characters, "host.squad",
		"pokemon.machop", "pokemon.gastly", "pokemon.bulbasaur")
	guest := squadOf(t, characters, "guest.squad",
		"pokemon.machop", "pokemon.cleffa", "pokemon.bulbasaur")
	roster := openingRoster(t, host, guest, config(11, 1))
	contested := contestedPairs(t, roster)
	if len(contested) != 2 {
		t.Fatalf("the fixture produced %d contested speed groups, want 2 — it no "+
			"longer reaches the case it was built for", len(contested))
	}
	if contested[0][0] == contested[1][0] {
		t.Errorf("both contested groups were led by the %s side: the two "+
			"uncontested speeds between them spent the alternation that belonged "+
			"to the second pair", contested[0][0])
	}
}

// contestedPairs is the side that led each contested speed group of a roster,
// in the order the roster lists them, read back through the speeds the queue
// itself will see.
func contestedPairs(t *testing.T, roster []battle.Roster) [][]hex.Side {
	t.Helper()
	speeds := enlisted(t, roster)
	order := []int64{}
	byGroup := map[int64][]hex.Side{}
	for _, entry := range roster {
		speed := speeds[entry.ID]
		if _, seen := byGroup[speed]; !seen {
			order = append(order, speed)
		}
		byGroup[speed] = append(byGroup[speed], entry.Side)
	}
	out := [][]hex.Side{}
	for _, speed := range order {
		group := byGroup[speed]
		if !slices.Contains(group, hex.SideAlly) || !slices.Contains(group, hex.SideEnemy) {
			continue
		}
		out = append(out, group)
	}
	return out
}

// TestTheGroupsAreReadFromTheSpeedTheQueueWillSee is the one a fixture of
// shipped squads will not produce by accident, and it is the reason the
// alternation costs an extra battle.New.
//
// A roster entry's Stats are not the speed the turn queue reads: the
// composition bonuses and the passives are applied at enlistment, before
// queue.Add asks. The fixture is built so the two readings **disagree about who
// ties whom** — both sides field a Magnemite authored at the same speed, and
// only the host's side holds a second electric unit, so the bonus moves one of
// the two and not the other. Grouping on the authored line would call them a
// contested pair and alternate a tie that does not exist, and every group after
// it would take the wrong turn of the alternation.
func TestTheGroupsAreReadFromTheSpeedTheQueueWillSee(t *testing.T) {
	dependencies := deps(t)
	host := squadOf(t, dependencies.Characters, "host.squad",
		"pokemon.magnemite", "pokemon.pichu", "pokemon.machop")
	guest := squadOf(t, dependencies.Characters, "guest.squad",
		"pokemon.magnemite", "pokemon.machop", "pokemon.poliwag")
	roster := openingRoster(t, host, guest, config(11, 1))

	speeds := enlisted(t, roster)
	authored, moved := map[string]int64{}, []string{}
	for _, entry := range roster {
		authored[entry.ID] = entry.Stats[progression.Speed]
		if authored[entry.ID] != speeds[entry.ID] {
			moved = append(moved, fmt.Sprintf("%s %d→%d", entry.ID,
				authored[entry.ID], speeds[entry.ID]))
		}
	}
	if len(moved) == 0 {
		t.Fatal("no unit's speed moved at enlistment, so this fixture no longer " +
			"tells the two readings apart")
	}
	t.Logf("moved at enlistment: %s", strings.Join(moved, ", "))

	// The two Magnemites are authored at one speed and enlist at two, which is
	// the disagreement the whole test rests on.
	pair := []string{}
	for _, entry := range roster {
		if strings.Contains(entry.ID, "magnemite") {
			pair = append(pair, entry.ID)
		}
	}
	if len(pair) != 2 {
		t.Fatalf("the fixture fielded %d Magnemites, want one a side", len(pair))
	}
	if authored[pair[0]] != authored[pair[1]] {
		t.Fatalf("the two Magnemites were authored at %d and %d, so the authored "+
			"reading would not have grouped them either", authored[pair[0]], authored[pair[1]])
	}
	if speeds[pair[0]] == speeds[pair[1]] {
		t.Fatalf("both Magnemites enlisted at %d: the bonus reached both sides, so "+
			"the two readings agree and nothing here is being told apart", speeds[pair[0]])
	}
	// Read the alternation back: on the enlisted speeds nothing about the
	// Magnemites is contested, so the only contested group is the Machamps.
	contested := contestedPairs(t, roster)
	if len(contested) != 1 {
		t.Fatalf("the enlisted speeds produced %d contested groups, want the one "+
			"pair of Machamps", len(contested))
	}
	if contested[0][0] != hex.SideAlly {
		t.Errorf("the only contested pair was led by the %s side: with nothing "+
			"contested before it, the first pair belongs to home", contested[0][0])
	}
}

// enlisted is every unit's speed as the turn queue will read it — after the
// composition bonuses and the passives — worked out the same way the room works
// it out, and independently of it.
func enlisted(t *testing.T, roster []battle.Roster) map[string]int64 {
	t.Helper()
	probe, err := battle.New(deps(t).Books, 1, roster)
	if err != nil {
		t.Fatalf("build a battle to read the speeds off: %v", err)
	}
	out := map[string]int64{}
	for _, entry := range roster {
		unit, known := probe.Unit(entry.ID)
		if !known {
			t.Fatalf("unit %q was enlisted and then could not be found", entry.ID)
		}
		out[entry.ID] = probe.Stats(unit)[progression.Speed]
	}
	return out
}

// TestTheLeadChangesHandsInsideOneSpeedGroupToo is the case the mirror above
// cannot reach, and the one that tells "the lead alternates per pair" apart from
// "per group".
//
// Both sides field two units of one speed, so a single contested group holds two
// pairs. Per group the whole group reads AEAE and one side leads both of its
// pairs; per pair it reads AEEA. A version that flipped the lead once a group
// passes every other test in this file.
func TestTheLeadChangesHandsInsideOneSpeedGroupToo(t *testing.T) {
	characters := deps(t).Characters
	// machop and squirtle are authored at one speed and neither shares an
	// element with anything else here, so nothing at enlistment moves them apart.
	squad := squadOf(t, characters, "twinned",
		"pokemon.machop", "pokemon.squirtle", "pokemon.gastly")
	roster := openingRoster(t, squad, squad.Clone(), config(11, 1))
	speeds := enlisted(t, roster)
	group := []battle.Roster{}
	for _, entry := range roster {
		if speeds[entry.ID] == speeds[roster[0].ID] {
			group = append(group, entry)
		}
	}
	if len(group) != 4 {
		t.Fatalf("the first enlisted unit's speed was shared by %d units, want two a "+
			"side — the "+
			"fixture no longer reaches a group of more than one pair", len(group))
	}
	if got, want := sides(group), "AEEA"; got != want {
		t.Errorf("a contested group of two pairs was enlisted %s, want %s: the lead "+
			"changes hands at every pair, not once a group", got, want)
	}
}

// TestTheSpeedsDoNotDependOnTheOrderTheRosterIsIn is the property the whole
// two-pass rests on, and it is the one that would make it dishonest if it broke.
//
// The room reads the speeds off a battle it builds and throws away, and then
// fights a battle with the roster in a different order. That is only sound while
// enlistment gives a unit the same speed wherever in the slice it sits: if a
// grant ever read the units already enlisted, the probe would be measuring a
// battle nobody fights and the groups would be the wrong ones.
func TestTheSpeedsDoNotDependOnTheOrderTheRosterIsIn(t *testing.T) {
	characters := deps(t).Characters
	host := squadOf(t, characters, "host.squad",
		"pokemon.magnemite", "pokemon.pichu", "pokemon.machop")
	guest := squadOf(t, characters, "guest.squad",
		"pokemon.magnemite", "pokemon.machop", "pokemon.poliwag")
	ally, err := host.Take(hex.SideAlly, characters)
	if err != nil {
		t.Fatalf("field the host squad: %v", err)
	}
	enemy, err := guest.Take(hex.SideEnemy, characters)
	if err != nil {
		t.Fatalf("field the guest squad: %v", err)
	}
	forward := append(append([]battle.Roster{}, ally...), enemy...)
	backward := append(append([]battle.Roster{}, enemy...), ally...)
	shuffled := []battle.Roster{ally[2], enemy[0], ally[0], enemy[2], enemy[1], ally[1]}

	want := enlisted(t, forward)
	for _, order := range [][]battle.Roster{backward, shuffled} {
		got := enlisted(t, order)
		for id, speed := range want {
			if got[id] == speed {
				continue
			}
			t.Errorf("%s enlisted at %d in one roster order and %d in another: the "+
				"speeds the groups are read from are not a fact about the roster, so "+
				"the battle the room probes is not the battle it fights",
				id, speed, got[id])
		}
	}
}
