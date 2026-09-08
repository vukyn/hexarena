package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
)

// aCensus is one board's reading, built by hand so the drawing is tested apart
// from the measuring. The counts are arbitrary; what matters is that one of them
// is nought.
func aCensus(against string, counts ...int) forge.CastCensus {
	kit := []string{"brace", "wrecking_swing", "cross_chop", "rock_throw"}
	census := forge.CastCensus{
		Build:   cast.Build{ID: "machop.charge", Character: "pokemon.machop", Name: "charge", Skills: kit},
		Against: against, Seeds: 6, Battles: 12,
	}
	for i, id := range kit {
		count := 0
		if i < len(counts) {
			count = counts[i]
		}
		census.Casts = append(census.Casts, forge.SkillCasts{Skill: id, Cast: count})
	}
	return census
}

// TestRenderCensusNamesTheBoardAndMarksTheSilentSlots.
//
// A count is only worth acting on if the page says what it was taken against —
// the reason forge.CastCensus carries Against at all — and a nought is only worth
// acting on if it is marked, because a column of numbers with a nought in it
// reads as a small figure rather than as a slot the author did not get.
func TestRenderCensusNamesTheBoardAndMarksTheSilentSlots(t *testing.T) {
	var drawn strings.Builder
	renderCensusBoard(&drawn, aCensus("s02", 4876, 0, 0, 38))
	page := drawn.String()

	for _, wanted := range []string{"s02", "12 battles", "brace", "4876", "rock_throw", "38"} {
		if !strings.Contains(page, wanted) {
			t.Errorf("the board never says %q:\n%s", wanted, page)
		}
	}
	if got := strings.Count(page, "silent"); got != 2 {
		t.Errorf("two slots were never cast and %d rows are marked silent:\n%s", got, page)
	}
	// The kit keeps its own order, because a build is a decision about four slots
	// in the order the author wrote them and a table sorted by count would be
	// answering a different question.
	brace, rock := strings.Index(page, "brace"), strings.Index(page, "rock_throw")
	if brace > rock {
		t.Errorf("the kit is not drawn in its own order:\n%s", page)
	}
}

// TestACensusVerdictIsAboutTheWalkAndNotTheLastBoard is the one thing this
// front-end can get wrong on its own.
//
// CensusWalk stops at the first board that leaves nothing silent, so the LAST
// table printed is always the board that played everything. A verdict read off
// that table would say "plays every slot" and be describing one matchup — and it
// would say it in exactly the same words for a build that needed one board and
// one that needed four. So the page has to state how many boards it took.
func TestACensusVerdictIsAboutTheWalkAndNotTheLastBoard(t *testing.T) {
	walk := forge.CensusWalk{
		Build: cast.Build{ID: "machop.charge", Character: "pokemon.machop",
			Skills: []string{"brace", "wrecking_swing", "cross_chop", "rock_throw"}},
		Taken: []forge.CastCensus{
			aCensus("s01", 71, 0, 34, 8),
			aCensus("s02", 4876, 6, 0, 38),
			aCensus("s03", 90, 6, 34, 8),
		},
	}
	var drawn strings.Builder
	renderCensusWalk(&drawn, walk)
	page := drawn.String()

	// ⚠️ Asserted on the VERDICT rather than on the page. The head line already
	// says "3 board(s) walked" and every board draws a table naming itself, so a
	// `Contains(page, "3 board")` passes with the verdict collapsed to the
	// one-board wording — which a mutation confirmed, and which is the whole
	// failure this test exists to catch.
	verdict := page[strings.LastIndex(page, "against s03"):]
	if strings.Contains(verdict, "on the first board it was stood on") {
		t.Errorf("a three-board walk reports the one-board verdict:\n%s", verdict)
	}
	if !strings.Contains(verdict, "3 board") {
		t.Errorf("the verdict does not say how many boards it took:\n%s", verdict)
	}
	if !strings.Contains(verdict, "s03 is") {
		t.Errorf("the verdict does not name the board that played every slot:\n%s", verdict)
	}
	// Every board walked is drawn, not only the one that settled it: a reader has
	// to be able to see that the earlier boards left something silent.
	for _, board := range []string{"s01", "s02", "s03"} {
		if !strings.Contains(page, "against "+board) {
			t.Errorf("the walk drew no table for %s:\n%s", board, page)
		}
	}
}

// TestASingleBoardWalkDoesNotClaimItTookSeveral is the other end of the sentence
// above, and it is here because the plural is the easy half to get right.
func TestASingleBoardWalkDoesNotClaimItTookSeveral(t *testing.T) {
	walk := forge.CensusWalk{
		Build: cast.Build{ID: "machop.charge", Character: "pokemon.machop",
			Skills: []string{"brace", "wrecking_swing", "cross_chop", "rock_throw"}},
		Taken: []forge.CastCensus{aCensus("s01", 71, 6, 34, 8)},
	}
	var drawn strings.Builder
	renderCensusWalk(&drawn, walk)
	page := drawn.String()
	if !strings.Contains(page, "first board") {
		t.Errorf("a one-board walk does not say so:\n%s", page)
	}
	if strings.Contains(page, "the ones above it") {
		t.Errorf("a one-board walk claims earlier boards left something silent:\n%s", page)
	}
}

// TestACensusThatNeverPlaysASlotSaysWhichAndHowMany is the failing verdict, which
// is the whole reason the command exists: it is the rule
// TestEveryShippedBuildPlaysItsOwnKit holds, worded for a person.
func TestACensusThatNeverPlaysASlotSaysWhichAndHowMany(t *testing.T) {
	walk := forge.CensusWalk{
		Build: cast.Build{ID: "machop.starved", Character: "pokemon.machop",
			Skills: []string{"wrecking_swing", "cross_chop", "rock_throw", "seismic_toss"}},
		Taken:  []forge.CastCensus{aCensus("s01", 0, 34, 8, 2), aCensus("s02", 0, 12, 4, 9)},
		Silent: []string{"wrecking_swing"},
	}
	var drawn strings.Builder
	renderCensusWalk(&drawn, walk)
	page := drawn.String()

	if !strings.Contains(page, "wrecking_swing") || !strings.Contains(page, "never cast") {
		t.Errorf("the verdict does not name the slot that never fired:\n%s", page)
	}
	if !strings.Contains(page, "2 authored squad") {
		t.Errorf("the verdict does not say how many boards were tried:\n%s", page)
	}
	if strings.Contains(page, "plays every slot") {
		t.Errorf("a failing walk reads as a passing one:\n%s", page)
	}
}

// TestTheCensusCommandRunsAgainstTheShippedCatalogue is the wiring, and it is one
// test rather than one per branch because what it proves is that the id lookup,
// the library load and the measurement meet — the drawing is held above, off
// values, where it costs no battles.
func TestTheCensusCommandRunsAgainstTheShippedCatalogue(t *testing.T) {
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	build, known := lib.Build("machop.charge")
	if !known {
		t.Fatal("machop.charge is not in the shipped catalogue, so this test measures nothing")
	}
	walk, err := lib.CensusWalk(build, 2)
	if err != nil {
		t.Fatalf("census walk: %v", err)
	}
	if walk.Boards() == 0 {
		t.Fatal("the walk stood the build on no board at all")
	}
	var drawn strings.Builder
	renderCensusWalk(&drawn, walk)
	page := drawn.String()
	for _, fielded := range build.Skills {
		if !strings.Contains(page, fielded) {
			t.Errorf("the page gives counts without naming the slot %q:\n%s", fielded, page)
		}
	}
	// The seeds asked for are the seeds reported. A page that printed the default
	// while a flag said otherwise is a measurement nobody can re-take.
	if !strings.Contains(page, "2 seeds an arrangement") {
		t.Errorf("the page does not report the seeds it was run at:\n%s", page)
	}
}

// TestTheCensusSubcommandIsOnTheDispatchTableAndInTheUsage.
//
// A subcommand that is dispatched and undocumented is one nobody finds, and one
// documented and undispatched is a usage line that fails. Both halves are one
// line each and neither is checked by anything else here.
func TestTheCensusSubcommandIsOnTheDispatchTableAndInTheUsage(t *testing.T) {
	if _, wired := commands["census"]; !wired {
		t.Error("census is not on the dispatch table")
	}
	var drawn strings.Builder
	usage(&drawn)
	if !strings.Contains(drawn.String(), "hexforge census") {
		t.Errorf("the usage never mentions census:\n%s", drawn.String())
	}
}
