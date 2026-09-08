package seed_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
)

// This file is the composition axis on shipped boards, and areaboard_test.go is
// the area axis on the same boards. That one asks which SHAPES a shipped
// formation is catchable by; this one asks which composition BONUSES a shipped
// formation reaches. Both are questions about the data that ships rather than
// about what the rules allow, and both log their answer rather than asserting
// it, for the reason that file gives twice: filing an open item as a failing
// test is not filing it.
//
// ⚠️ **It does not overlap the three properties in elementbonus_test.go, in any
// of the four combinations.** Those ask whether the CAST could field a rung
// (TestEveryElementBonusRungIsReachable), whether the table covers the elements
// the cast can field (TestEveryFieldableElementHasABonus and
// TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier) and whether two
// bonuses come to one effect (TestNoTwoBonusesGrantTheSameEffect). None of them
// looks at a formation, and the first skips every bonus that is not an element
// one with a named value — so `same_column` is covered by no reachability test
// at all today. This file asks the remaining question, over the whole book:
// whether anything that SHIPS reaches a rung.
//
// Nothing here fights: no seeds, no forge.FightSquads, no rates. It is a data
// walk in milliseconds, and the measurements DAT-008 took were taken in a
// scratch directory rather than in a test, which stays true.

// aBonusBoard is one shipped formation as composition.Book.Awards counts it: a
// name, the members it fields, and whether anything ever fights it.
//
// The fought flag is the distinction aFormation carries in areaboard_test.go
// and it is here for the same reason — forge.FightSquads fights the squads and
// nothing else, so a bonus reached only by the seed roster is a bonus no rate in
// this repository is taken on. The full argument is on that type and is not
// restated here.
type aBonusBoard struct {
	name    string
	members []composition.Member
	fought  bool
}

// membersOf is the mapping battle.awards makes before it counts
// (internal/core/battle/battle.go:449-450), written the same way here on
// purpose: what a bonus is shown is an id, an affinity and the authored column,
// and nothing else.
func membersOf(roster []battle.Roster) []composition.Member {
	members := make([]composition.Member, 0, len(roster))
	for _, entry := range roster {
		members = append(members, composition.Member{
			ID: entry.ID, Affinity: entry.Affinity, Column: entry.Slot.Col})
	}
	return members
}

// shippedBonusBoards is every formation the game ships, as members: the five
// squads and the two halves of the seed roster.
//
// The squads resolve through Squad.Take, which is the door forge.FightSquads
// goes through, so a squad that stops resolving fails here loudly rather than
// quietly counting nothing.
//
// ⚠️ **No hex.Place, and no loop over the two sides** — which is exactly where
// this differs from shippedFormations next door, and the difference is
// deliberate rather than unfinished. composition.Member.Column is the AUTHORED
// slot, "so the two sides count the same shape as the same shape"
// (internal/core/composition/composition.go, on Member.Column), so a squad
// counts identically on either half: a best-of-two-halves reduction would be
// two readings of one number rather than a wider search. The side handed to
// Take only prefixes the ids.
func shippedBonusBoards(t *testing.T) []aBonusBoard {
	t.Helper()
	characters := mustCast(t)
	var out []aBonusBoard
	for _, squad := range mustSquads(t) {
		roster, err := squad.Take(hex.SideAlly, characters)
		if err != nil {
			t.Fatalf("field squad %q: %v", squad.ID, err)
		}
		out = append(out, aBonusBoard{name: squad.ID, members: membersOf(roster), fought: true})
	}
	halves := map[hex.Side][]battle.Roster{}
	for _, unit := range mustSeedRoster(t) {
		halves[unit.Side] = append(halves[unit.Side], unit)
	}
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		if half := halves[side]; len(half) > 0 {
			out = append(out, aBonusBoard{name: "roster/" + side.String(), members: membersOf(half)})
		}
	}
	return out
}

// aReach is one bonus firing on one formation: which bonus, on which value of
// its axis, and how many units shared it.
type aReach struct {
	bonus string
	value string
	count int
}

// bonusesReached is what one formation really fires, reduced to one entry per
// bonus and value from the one entry per receiving unit Awards returns.
//
// ⚠️ **The counting rule is read from composition.Book.Awards and never
// re-implemented here.** A test that tallied elements and columns itself would
// be a second declaration of a rule that already exists — the inert skip, the
// dual affinity counting on both sides of itself, the ladder taking the highest
// rung a count satisfies and that one only — and the copy is the one that goes
// stale. The count that fired is carried on the Award, so nothing is recomputed
// either.
//
// The order is Awards' own, which is the book's declaration order over a slice
// and never a map walk.
func bonusesReached(bonuses *composition.Book, chart *element.Chart, members []composition.Member) []aReach {
	var out []aReach
	for _, award := range bonuses.Awards(chart, members) {
		if slices.ContainsFunc(out, func(held aReach) bool {
			return held.bonus == award.Bonus && held.value == award.Value
		}) {
			continue
		}
		out = append(out, aReach{bonus: award.Bonus, value: award.Value, count: award.Count})
	}
	return out
}

// describe names what a reach walk found, so a failure can say it in words.
func describe(reached []aReach) string {
	if len(reached) == 0 {
		return "nothing"
	}
	out := make([]string, 0, len(reached))
	for _, one := range reached {
		out = append(out, one.bonus+" ("+one.value+" ×"+strconv.Itoa(one.count)+")")
	}
	return strings.Join(out, ", ")
}

// lowestRung is the cheapest threshold a bonus declares.
//
// Read off the rungs rather than taken as Rungs[0]: the parser holds them
// ascending, and leaning on that here would be a second copy of a rule that
// lives in ParseBook.
func lowestRung(held composition.Bonus) int {
	lowest := 0
	for index, rung := range held.Rungs {
		if index == 0 || rung.At < lowest {
			lowest = rung.At
		}
	}
	return lowest
}

// TestEveryBonusIsMeasuredAgainstEveryShippedFormation reports, per bonus,
// whether any formation the game ships reaches one of its rungs.
//
// It matters because composition.Book.Without is the pricing instrument DAT-002
// decision 4 calls the only thing that can price a bonus, and an instrument has
// to have a board to be applied to. DAT-008 Run A measured what this walk
// derives: taking `ground_root` and `same_column` out of the book comes back
// identical in every field — rate, both half-tallies, median turns — on every
// pairing forge.FightSquads fights. A switch that changes nothing prices
// nothing.
//
// ⚠️ **What it holds and what it deliberately does not.** It asserts only that
// it measured something. The reach itself is LOGGED, because the right value is
// undecided: today the whole ten-bonus table reaches nothing on any of the seven
// shipped formations, which is DAT-010 and is open. Turning the logged half into
// an assertion would be filing that finding as a red test rather than as an
// item, the reason both tests in areaboard_test.go give for logging their own
// halves. What is asserted is the floor under it — that the walk ran over a
// bonus and a formation at all.
//
// ⚠️ **"No shipped formation reaches it" is not "nobody can fire it", and the
// wording of every line above says the first.** This file has been wrong in that
// exact direction twice. DAT-002 read "one carrier" as "no squad can field a
// tribe of it", which is true of a drafted squad and false of a saved one — the
// correction is on TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier.
// And DAT-008 corrected "only ground squads reach bedrock" by building a legal
// stacked squad that fires both `ground_root` rung 3 and `same_column` rung 3 at
// once. What a player could build is unbounded and is not what this walks; the
// cast-level version of the question is TestEveryElementBonusRungIsReachable.
//
// The bonuses come from the book and the boards from the squad and roster files
// — never from a written-down list, which goes stale the day one is authored,
// and goes stale silently.
func TestEveryBonusIsMeasuredAgainstEveryShippedFormation(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}
	boards := shippedBonusBoards(t)

	// where each bonus fires, and whether any of the boards it fires on is one a
	// rate is read off.
	where := map[string][]string{}
	onAFoughtBoard := map[string]bool{}
	// pairs is bonus × formation, counted only where the formation really
	// resolved to somebody. One counter covers the three separate ways this walk
	// could measure nothing — an empty bonus book, no shipped formation at all,
	// and formations that resolve to no members — and it counts pairs across both
	// sources rather than counting squads, because emptying squads.json alone
	// leaves the two roster halves and the walk would still be measuring
	// something.
	var pairs, walked int
	for _, board := range boards {
		if len(board.members) == 0 {
			continue
		}
		walked++
		fired := bonusesReached(bonuses, chart, board.members)
		for _, held := range bonuses.All() {
			pairs++
			for _, one := range fired {
				if one.bonus != held.ID {
					continue
				}
				where[held.ID] = append(where[held.ID],
					board.name+" ("+one.value+" ×"+strconv.Itoa(one.count)+")")
				onAFoughtBoard[held.ID] = onAFoughtBoard[held.ID] || board.fought
			}
		}
	}
	if pairs == 0 {
		t.Fatal("no bonus was measured against any shipped formation, so this test measured " +
			"nothing: either the bonus book is empty, the game ships no formation, or every " +
			"formation resolved to no members")
	}

	for _, held := range bonuses.All() {
		axis := held.Axis.String()
		if held.Value != "" {
			axis += " " + held.Value
		}
		rungs := make([]string, 0, len(held.Rungs))
		for _, rung := range held.Rungs {
			rungs = append(rungs, strconv.Itoa(rung.At))
		}
		switch {
		case onAFoughtBoard[held.ID]:
			t.Logf("%-15s %-16s rungs %-5s reached by %s",
				held.ID, axis, strings.Join(rungs, "/"), strings.Join(where[held.ID], ", "))
		case len(where[held.ID]) > 0:
			// DAT-010. Not an assertion: see the note on this test.
			t.Logf("%-15s %-16s rungs %-5s reached by %s, and by no board a rate is read off",
				held.ID, axis, strings.Join(rungs, "/"), strings.Join(where[held.ID], ", "))
		default:
			// DAT-010. Not an assertion: see the note on this test.
			t.Logf("%-15s %-16s rungs %-5s reached by NO shipped formation, of the %d walked",
				held.ID, axis, strings.Join(rungs, "/"), walked)
		}
	}
}

// handBuilt is count members carrying one affinity, in distinct columns when
// spread is true and all in one column when it is false.
//
// ⚠️ **These are not a legal board and are not meant to be.** The counting rule
// is board-blind — columnTallies reads Member.Column and nothing there checks it
// against hex.FormationCols — and that is what lets the branch below be
// exercised at all: no formation the game ships reaches a single rung, so
// members that do have to be built by hand. It is the shape
// TestTwoKindsWithTheSameTermsInEitherOrderFingerprintAlike uses next door, for
// the same reason.
func handBuilt(count int, affinity element.Affinity, spread bool) []composition.Member {
	members := make([]composition.Member, 0, count)
	for index := range count {
		column := 0
		if spread {
			column = index
		}
		members = append(members, composition.Member{
			ID: "hand." + strconv.Itoa(index), Affinity: affinity, Column: column})
	}
	return members
}

// TestTheBonusReachWalkSeesAFormationThatReachesARung exercises the branch no
// shipped data can reach.
//
// The walk above runs over seven formations that reach nothing, so on shipped
// data it only ever executes the "reached by nobody" half: an assertion carried
// by data that cannot exercise it is a fixture hiding a branch, and this
// repository has paid for that five times. So the reach is exercised here, on
// members built by hand.
//
// Four arms, and the second and fourth are what make the first and third
// measurements rather than assertions a helper returning everything would also
// pass:
//
//  1. the rung's worth of sharers, in distinct columns   -> reports that bonus
//  2. one sharer short of it                             -> does NOT report it
//  3. the rung's worth of INERT units in one column      -> the column bonus, and nothing else
//  4. the same units spread out                          -> nothing at all
//
// Arm 3 doubles as the inert-skip exerciser: units carrying the element with no
// matchup form no tribe, so a stacked column of them can only reach the column
// axis. Every id is derived from the book and the chart; none is written down.
func TestTheBonusReachWalkSeesAFormationThatReachesARung(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}

	var tribe, stack composition.Bonus
	for _, held := range bonuses.All() {
		if tribe.ID == "" && held.Axis == composition.AxisElement && held.Value != "" {
			tribe = held
		}
		if stack.ID == "" && held.Axis == composition.AxisColumn {
			stack = held
		}
	}
	// One guard per search, the shape both siblings carry: a green run that found
	// no bonus to build for has exercised nothing.
	if tribe.ID == "" {
		t.Fatal("no bonus counts a named element, so the two element arms below measure nothing")
	}
	if stack.ID == "" {
		t.Fatal("no bonus counts the column axis, so the two column arms below measure nothing")
	}
	inert := chart.Inert()
	if len(inert) == 0 {
		t.Fatal("the chart declares no inert element, so the column arms cannot be built out of " +
			"units that form no tribe and would be measuring the element axis as well")
	}

	shared := mustAffinity(t, tribe.Value)
	unaligned, err := element.Single(inert[0])
	if err != nil {
		t.Fatalf("affinity %s: %v", inert[0], err)
	}
	atTribe, atStack := lowestRung(tribe), lowestRung(stack)

	// Arm 1. The rung is reached and the walk says so.
	reached := bonusesReached(bonuses, chart, handBuilt(atTribe, shared, true))
	if !slices.ContainsFunc(reached, func(one aReach) bool { return one.bonus == tribe.ID }) {
		t.Errorf("%d units carrying %s reach %s's rung at %d and the walk reports %s: the branch "+
			"no shipped formation exercises does not work",
			atTribe, tribe.Value, tribe.ID, atTribe, describe(reached))
	}

	// Arm 2. The control that makes arm 1 a measurement: one sharer short of the
	// rung must NOT report the bonus. Without it a walk that reported every bonus
	// whatever it was handed would pass arm 1.
	short := bonusesReached(bonuses, chart, handBuilt(atTribe-1, shared, true))
	if slices.ContainsFunc(short, func(one aReach) bool { return one.bonus == tribe.ID }) {
		t.Errorf("%d units carrying %s are one short of %s's rung at %d and the walk reports it "+
			"anyway (%s): the arm above measured nothing",
			atTribe-1, tribe.Value, tribe.ID, atTribe, describe(short))
	}

	// Arm 3. The column axis, and the inert skip in the same arm.
	stacked := bonusesReached(bonuses, chart, handBuilt(atStack, unaligned, false))
	if !slices.ContainsFunc(stacked, func(one aReach) bool { return one.bonus == stack.ID }) {
		t.Errorf("%d units standing in one column reach %s's rung at %d and the walk reports %s",
			atStack, stack.ID, atStack, describe(stacked))
	}
	for _, one := range stacked {
		if one.bonus == stack.ID {
			continue
		}
		t.Errorf("%d units carrying the inert %s also reach %s: sharing the element with no "+
			"matchup is sharing the absence of one, and forms no tribe",
			atStack, inert[0], one.bonus)
	}

	// Arm 4. The control for arm 3: spread the same units out and they share
	// neither a column nor a tribe, so nothing at all fires.
	apart := bonusesReached(bonuses, chart, handBuilt(atStack, unaligned, true))
	if len(apart) > 0 {
		t.Errorf("%d units carrying the inert %s, one per column, reach %s: they share neither a "+
			"column nor a tribe, so the arm above measured nothing",
			atStack, inert[0], describe(apart))
	}
}
