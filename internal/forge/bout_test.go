package forge

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// boutSeeds is how many seeds a test here fights each arrangement over.
//
// Small on purpose, for the same reason sparSeeds is: every property asserted
// here is **exact** — a rating against itself is even to the last part in a
// thousand, a lopsided harness is refused outright — so nothing here needs the
// confidence a measurement needs. The measurement lives in internal/seed, where
// the shipped roster is.
const boutSeeds = 12

// boutBoard is a deliberately asymmetric board: two different characters, one on
// each side.
//
// Asymmetric on purpose. The exact control is the claim that a board's own bias
// cancels when it is fought from both ends, and a mirror board would prove that
// claim by removing the thing it is about.
func boutBoard(t *testing.T, lib *Library) []battle.Roster {
	t.Helper()
	characters := lib.Characters().All()
	if len(characters) < 2 {
		t.Fatalf("the fixture library holds %d character(s), and a bout needs two different ones",
			len(characters))
	}
	first, err := lib.duellist(characters[0], progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve %s: %v", characters[0].ID, err)
	}
	second, err := lib.duellist(characters[1], progression.LevelCap)
	if err != nil {
		t.Fatalf("resolve %s: %v", characters[1].ID, err)
	}
	return []battle.Roster{
		place(firstID, first, hex.SideAlly),
		place(secondID, second, hex.SideEnemy),
	}
}

// TestABoutAgainstItselfIsExactlyEven is the control the whole harness rests on,
// and it is exact rather than statistical.
//
// A rating fought against a copy of itself has no advantage over it — the two
// arrangements are the *same battle*, seed for seed and decision for decision —
// so the challenger holds the winning side in exactly one of the two, and the
// rate is 500 parts per thousand by construction rather than by convergence. A
// band would let a leak through here, which is why Bout asserts on the number
// itself and so does this.
func TestABoutAgainstItselfIsExactlyEven(t *testing.T) {
	lib := sparLibrary(t)
	report, err := Bout(lib.Books(), boutBoard(t, lib), Suggesting, Suggesting, boutSeeds)
	if err != nil {
		t.Fatalf("bout a rating against itself: %v", err)
	}
	if rate := report.Rate(); rate != scale.Base/2 {
		t.Errorf("a rating against itself comes to %d rather than an even %d: %+v",
			rate, scale.Base/2, report.Challenger)
	}
	// The stronger statement, and the one that says *why* it is even: every win
	// from one arrangement is a loss from the other. A rate that came out even by
	// two errors cancelling would pass the check above and fail this one.
	if report.Challenger.Wins != report.Incumbent.Losses ||
		report.Challenger.Losses != report.Incumbent.Wins ||
		report.Challenger.Draws != report.Incumbent.Draws {
		t.Errorf("the two sides of the same battles are not each other's reflection: "+
			"challenger %+v, incumbent %+v", report.Challenger, report.Incumbent)
	}
	// And it has to be even over something. A run that decided nothing is even
	// too, and it measures nothing at all.
	if report.Challenger.Wins == 0 {
		t.Errorf("nothing was won over %d battle(s), so an even split proves only that "+
			"nought and nought are the same number", report.Challenger.Battles())
	}
	if report.Turns <= 0 {
		t.Errorf("the median battle ran %d turns, so no battle finished", report.Turns)
	}
}

// TestDrivenByTheShippedRatingIsJustRunningIt is the guard on the split that made
// this file possible.
//
// RunToEnd is now RunToEndWith fixed to Suggest, and the whole of the harness rests
// on that being a **refactor** rather than a change: every figure in the repository
// was taken through RunToEnd, and a battle driven by a composed Chooser has to be
// the same battle or nothing measured here is comparable with anything measured
// before. The goldens say so end to end; this says so directly, event for event,
// through the two functions in this package that sit either side of the split.
//
// The composed Chooser is the interesting half. It dispatches on the prompted
// unit's side, so it is a different function object on every turn — and if that
// dispatch ever picked the wrong side, or missed a summon the roster never named,
// the two runs would diverge on the first turn it mattered.
func TestDrivenByTheShippedRatingIsJustRunningIt(t *testing.T) {
	lib := sparLibrary(t)
	books, roster := lib.Books(), boutBoard(t, lib)
	for seed := uint64(1); seed <= boutSeeds; seed++ {
		run, err := battle.New(books, seed, roster)
		if err != nil {
			t.Fatalf("seed %d: assemble: %v", seed, err)
		}
		run.Begin()
		plainTurns, err := run.RunToEnd(sparTurnLimit)
		if err != nil {
			t.Fatalf("seed %d: run it: %v", seed, err)
		}

		driven, err := battle.New(books, seed, roster)
		if err != nil {
			t.Fatalf("seed %d: assemble: %v", seed, err)
		}
		driven.Begin()
		drivenTurns, err := driven.RunToEndWith(sparTurnLimit,
			sided(driven, Suggesting(driven), Suggesting(driven)))
		if err != nil {
			t.Fatalf("seed %d: drive it: %v", seed, err)
		}

		if plainTurns != drivenTurns {
			t.Fatalf("seed %d takes %d turns when run and %d when driven by the same "+
				"rating: the split was not behaviour-preserving, and every figure taken "+
				"through RunToEnd is on a different instrument from every figure taken "+
				"through a bout", seed, plainTurns, drivenTurns)
		}
		was, now := run.Drain(), driven.Drain()
		if len(was) != len(now) {
			t.Fatalf("seed %d records %d event(s) when run and %d when driven",
				seed, len(was), len(now))
		}
		for index := range was {
			if was[index] != now[index] {
				t.Fatalf("seed %d diverges at event %d: run %+v, driven %+v",
					seed, index, was[index], now[index])
			}
		}
	}
}

// TestABoutRefusesAControlThatIsNotEven is the other half: the control has to be
// able to fail, or asserting on it is theatre.
func TestABoutRefusesAControlThatIsNotEven(t *testing.T) {
	lib := sparLibrary(t)
	_, err := Bout(lib.Books(), boutBoard(t, lib), lopsided(boutSeeds), Suggesting, boutSeeds)
	if err == nil {
		t.Fatal("a bout printed a figure for a rating whose control is not even")
	}
	// It has to be refused *for being uneven*. A run that stalled would also be
	// refused, and a test that accepted any error at all would keep passing on a
	// fixture that had stopped exercising the control.
	if !strings.Contains(err.Error(), "control") || !strings.Contains(err.Error(), "exactly even") {
		t.Errorf("the refusal reads %q, which is not the control refusing an uneven split", err)
	}
}

// lopsided is a Brain that is **not a function of the battle it is handed**, and
// it is what a broken harness looks like.
//
// Breaking the control is harder than it sounds, and the difficulty is the design
// working: anything that is a function of *position* — a rating that plays worse
// as the enemy, say — cancels exactly, because the two arrangements are the same
// two positions with the two ratings exchanged. So the only way to make a control
// uneven is state that advances **between** the arrangements.
//
// Bout fights every seed of the first arrangement before it starts the second, so
// a count of instantiations separates the two: the first `2*seeds` calls are the
// first arrangement and the rest are the second. This plays properly for whichever
// position the challenger is holding in the arrangement it is in, and surrenders
// on the other — so the challenger wins both arrangements, the rate comes to a
// thousand, and the control refuses.
func lopsided(seeds int) Brain {
	calls := 0
	return func(fight *battle.Battle) battle.Chooser {
		// Two calls per battle, allied first: the position is the parity and the
		// arrangement is which half of the run the call falls in.
		arrangement, position := calls/(2*seeds), calls%2
		calls++
		if arrangement == position {
			return Suggesting(fight)
		}
		return surrendering
	}
}

// surrendering never takes anything, so RunToEndWith passes for it every turn.
func surrendering(*battle.Prompt) (battle.Choice, bool) { return battle.Choice{}, false }

// TestABoutRefusesSeedsItCannotMeasure is the cheapest refusal of the three: a
// bout over no seeds has no control to be even.
func TestABoutRefusesSeedsItCannotMeasure(t *testing.T) {
	lib := sparLibrary(t)
	for _, seeds := range []int{0, -1} {
		if _, err := Bout(lib.Books(), boutBoard(t, lib), Suggesting, FirstUsable, seeds); err == nil {
			t.Errorf("a bout over %d seed(s) reported a figure", seeds)
		}
	}
}

// TestTheRulerIsNotAnOpponent pins what FirstUsable does, deliberately and
// permanently.
//
// It is the fixed end of every comparison made about a rating in this repository:
// "Suggest beats FirstUsable by X permille over N seeds" is only comparable across
// the repository's history because the ruler never moves. So this test exists to
// **fail a future improvement**. If it is failing because somebody taught
// FirstUsable to aim better, the answer is not to accept the new behaviour — it is
// to add a second baseline beside it and leave this one where it is.
//
// The prompt is built by hand rather than fought for, because the ruler reads
// nothing but the options: it never touches the battle, which is why it takes one
// and ignores it. A fixture drawn from a battle would tie the pin to whatever the
// data happens to say this week.
func TestTheRulerIsNotAnOpponent(t *testing.T) {
	ruler := FirstUsable(nil)

	// The first option is on cooldown, the second is feeble and usable, the third
	// is devastating and usable. A ruler takes the second: kit order, first thing
	// available, first cell offered — no comparison of any kind.
	prompt := &battle.Prompt{
		Unit: "a", Turn: 1,
		Options: []battle.Option{
			{Skill: "recharging", Reason: "cooling down"},
			{Skill: "feeble", Aims: []hex.Offset{{Col: 1, Row: 1}, {Col: 2, Row: 2}}},
			{Skill: "devastating", Aims: []hex.Offset{{Col: 3, Row: 3}}},
		},
	}
	choice, ok := ruler(prompt)
	switch {
	case !ok:
		t.Fatal("the ruler declined a prompt with two usable options on it")
	case choice.Skill != "feeble":
		t.Errorf("the ruler took %q, want feeble: it takes the first option it can "+
			"use and compares nothing, and a ruler that has learnt to compare is no "+
			"longer the thing every figure in this repository was measured against",
			choice.Skill)
	case choice.Aim != (hex.Offset{Col: 1, Row: 1}):
		t.Errorf("the ruler aimed at %s, want the first cell offered", choice.Aim)
	}

	// And it declines rather than inventing a turn when nothing is on offer, which
	// is what makes it a legal Chooser: RunToEndWith passes for it.
	nothing := &battle.Prompt{Unit: "a", Turn: 2, Options: []battle.Option{
		{Skill: "recharging", Reason: "cooling down"},
	}}
	if _, ok := ruler(nothing); ok {
		t.Error("the ruler took an option that was not available")
	}
	if _, ok := ruler(&battle.Prompt{Unit: "a", Turn: 3}); ok {
		t.Error("the ruler took an option from a prompt that offered none")
	}
}
