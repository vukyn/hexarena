package room_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
)

// TestADraftingSeriesDraftsOnceABattle is the decision this feature is, played
// out end to end: three battles, three ban-and-picks, three squads chosen out of
// three fresh pools.
//
// ⚠️ **The loop dispatches on the CLIENT's own reading of what it is being asked
// for**, which is the sharpest thing here and the reason it is written this way
// rather than by counting messages. The room says which seat is due; the client
// says whether the thing due is a draft decision or a battle turn, off state it
// computed from the record alone. A room that opened a draft the client did not,
// or began a battle while the client still held a draft, deadlocks here on the
// very first disagreement rather than running to a plausible-looking end.
func TestADraftingSeriesDraftsOnceABattle(t *testing.T) {
	dependencies := deps(t)
	configuration := draftingConfig(11)
	configuration.Battles = 3
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)

	for _, name := range []string{"Host", "Guest"} {
		_, out, err := opened.Join(helloWithNoSquad(t, name))
		if err != nil {
			t.Fatalf("%s joins: %v", name, err)
		}
		clients.deliver(t, out)
	}

	const ceiling = 20000
	steps := 0
	for !opened.Finished() {
		steps++
		if steps > ceiling {
			t.Fatalf("a drafting bo3 had not finished after %d inputs", ceiling)
		}
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatalf("after %d inputs the room is waiting on nobody and the match is not over",
				steps)
		}
		client := clients.at(onTurn)
		answered, err := opened.Deliver(onTurn, client.due())
		if err != nil {
			t.Fatalf("input %d from %s: %v", steps, onTurn, err)
		}
		clients.deliver(t, answered)
	}

	// Three battles, and three drafts to produce them.
	// ⚠️ **A bo3 is two or three battles, not three.** It ends when somebody takes
	// two, so demanding three would be demanding a 2–1 — a test that passed only
	// on the seeds where the series went the distance.
	played := opened.Played()
	if len(played) < 2 || len(played) > configuration.Battles {
		t.Fatalf("a bo3 recorded %d battles, want 2 or %d", len(played), configuration.Battles)
	}
	for _, client := range []*mirror{clients.host, clients.guest} {
		if got := len(client.starts); got != len(played) {
			t.Errorf("%s was started on %d battles and the room played %d",
				client.seat, got, len(played))
		}
	}

	// ⚠️ **The re-draft is counted, not inferred from the rosters, and the first
	// version of this test got that wrong.** It asserted the rosters are not all
	// identical — and they ARE all identical, every run, because the fake client
	// picks the first candidate the pool still offers and a reset pool offers the
	// same first candidate every time. That is the fixture being deterministic,
	// which is what it is for; it is not the draft failing to re-run, and a reader
	// who found that test red would have gone looking in the wrong place.
	//
	// ⚠️ **And the unit is MEASURED rather than computed**, which is the second
	// thing that version got wrong: a formula of picks, bans and arrangements
	// gave twelve where a draft actually records eighteen, so the assertion was
	// about the formula. A bo1 in the same fixture is what one draft costs.
	perDraft := decisionsOfOneDraft(t, dependencies)
	for _, client := range []*mirror{clients.host, clients.guest} {
		if got, want := len(client.decided), perDraft*len(played); got != want {
			t.Errorf("%s recorded %d draft decisions over %d battles and one draft is %d, "+
				"so it saw %.1f draft(s): a series drafts once a battle",
				client.seat, got, len(played), perDraft, float64(got)/float64(perDraft))
		}
	}
	for index, start := range clients.host.starts {
		t.Logf("battle %d %v", index+1, charactersOf(start.Roster))
	}
}

// charactersOf is the characters a roster fields, with the side prefix taken off.
//
// ⚠️ **`Take` prefixes every unit id with the side it is enlisted on**, and a bo3
// swaps home between battles — so the same character comes back as
// `ally.pokemon.gastly` in one battle and `enemy.pokemon.gastly` in the next.
// Comparing the ids raw would report every roster as different and both
// assertions below would pass on a draft that never re-ran.
func charactersOf(roster []battle.Roster) []string {
	out := make([]string, 0, len(roster))
	for _, entry := range roster {
		_, character, found := strings.Cut(entry.ID, ".")
		if !found {
			character = entry.ID
		}
		out = append(out, character)
	}
	sort.Strings(out)
	return out
}

// TestEachBattleOfADraftingSeriesPicksFromAFullPool is the reset, and it is the
// half the roster comparison above cannot see.
//
// ⚠️ Three battles out of ONE pool would run it down — 3v3 spends six picks and
// its bans a battle — so the third draft would be a choice between whatever
// nobody wanted twice. What says the pool came back is that a character banned or
// picked in an earlier battle is on the board in a later one: impossible if the
// pool were carried, ordinary if it resets.
func TestEachBattleOfADraftingSeriesPicksFromAFullPool(t *testing.T) {
	dependencies := deps(t)
	configuration := draftingConfig(11)
	configuration.Battles = 3
	opened := newRoom(t, configuration)
	clients := newTable(t, dependencies, configuration.TurnCap)

	for _, name := range []string{"Host", "Guest"} {
		_, out, err := opened.Join(helloWithNoSquad(t, name))
		if err != nil {
			t.Fatalf("%s joins: %v", name, err)
		}
		clients.deliver(t, out)
	}
	for steps := 0; !opened.Finished(); steps++ {
		if steps > 20000 {
			t.Fatal("a drafting bo3 had not finished")
		}
		onTurn, waiting := opened.Awaiting()
		if !waiting {
			t.Fatal("the room is waiting on nobody and the match is not over")
		}
		answered, err := opened.Deliver(onTurn, clients.at(onTurn).due())
		if err != nil {
			t.Fatalf("input from %s: %v", onTurn, err)
		}
		clients.deliver(t, answered)
	}

	spent := map[string]int{}
	for _, id := range charactersOf(clients.host.starts[0].Roster) {
		spent[id] = 1
	}
	returned := 0
	for _, start := range clients.host.starts[1:] {
		for _, id := range charactersOf(start.Roster) {
			if spent[id] > 0 {
				returned++
			}
		}
	}
	if returned == 0 {
		t.Errorf("no character from battle 1's board reached a later one, which is what a "+
			"pool carried across the series would look like: %v", spent)
	}
	t.Logf("%d of battle 1's units came back in a later battle", returned)
}

// decisionsOfOneDraft is what a single ban and pick records, measured by running
// one rather than by restating its arithmetic.
//
// ⚠️ **A formula was tried and was wrong.** Twice the bans plus twice the picks
// plus the two arrangements comes to twelve; a draft records eighteen. Whatever
// the difference is — and it is internal/draft's to own — a test that computed it
// would be asserting its own arithmetic, which is the mistake this repository
// keeps a list of.
func decisionsOfOneDraft(t *testing.T, dependencies room.Deps) int {
	t.Helper()
	single := newRoom(t, draftingConfig(11))
	clients := newTable(t, dependencies, room.DefaultTurnCap)
	for _, name := range []string{"Host", "Guest"} {
		_, out, err := single.Join(helloWithNoSquad(t, name))
		if err != nil {
			t.Fatalf("%s joins the measuring room: %v", name, err)
		}
		clients.deliver(t, out)
	}
	for steps := 0; len(clients.host.starts) == 0; steps++ {
		if steps > 200 {
			t.Fatal("a bo1 draft had not opened a battle after 200 decisions")
		}
		onTurn, waiting := single.Awaiting()
		if !waiting {
			t.Fatal("the measuring room is waiting on nobody and no battle has started")
		}
		answered, err := single.Deliver(onTurn, clients.at(onTurn).due())
		if err != nil {
			t.Fatalf("measuring decision from %s: %v", onTurn, err)
		}
		clients.deliver(t, answered)
	}
	if len(clients.host.decided) == 0 {
		t.Fatal("a whole draft recorded no decisions")
	}
	return len(clients.host.decided)
}
