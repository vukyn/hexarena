package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/seed"
)

// The three Naruto kits this file measures. Written out rather than derived, for
// the reason every other design table in this package is: they are what was
// measured, and a change to the data must not quietly rewrite the claim.
var (
	narutoBase     = []string{"kunai", "wind_shuriken", "rasengan", "shadow_clone"}
	narutoComeback = []string{"kunai", "wind_shuriken", "comeback", "shadow_clone"}
)

// comebackSeeds is how many battles each half of a comparison is fought over.
const comebackSeeds = 400

// mirrorSlot is where both units stand: the front column, middle row, mirrored.
var mirrorSlot = hex.Offset{Col: hex.FormationCols - 1, Row: hex.Rows / 2}

// TestComebackIsAChoiceAgainstTheSignatureSkill is what `comeback`'s numbers were
// picked against, and it is a swap against **rasengan** rather than against the
// kit's filler on purpose.
//
// Swapping it for `kunai` reads about three battles in four at every power from
// 500 to 1100, which sounds like a finding and is not one: `kunai` is a 700-power
// skill on no cooldown and the fourth slot of that kit is nearly free, so beating
// it says nothing about what was put in its place. The swap against `rasengan` is
// the one that discriminates, and it is monotone in the power:
//
//	power/at_empty/cooldown    for kunai    for rasengan
//	1100 /  900 / 2              91.6%          84.7%
//	 900 /  900 / 2              83.0%          48.3%
//	 800 / 1000 / 2              77.3%          38.2%
//	 700 /  900 / 2              75.0%          14.3%
//	 600 /  900 / 2              75.6%           7.8%
//
// So the shipped figures are the ones that make it a **choice** rather than a
// replacement for the signature skill. It never out-hits `rasengan` on power —
// 900 at full and 1710 with nothing left, against a flat 2200 — and what it buys
// instead is turns, at a cooldown of two against four.
//
// ⚠️ The band is wide because this is one pairing of one character, and Naruto is
// the character with no builds in the catalogue yet. It asserts that neither kit
// is a scripted defeat, which is the claim that can be made honestly today.
func TestComebackIsAChoiceAgainstTheSignatureSkill(t *testing.T) {
	rate := mirrorRate(t, narutoComeback, narutoBase, "swiftness")
	const lowest, highest = 250, 750
	if rate < lowest || rate > highest {
		t.Errorf("the kit holding comeback wins %d.%d%% against the one holding rasengan, "+
			"outside %d..%d: one of the two has become a scripted defeat",
			rate/10, rate%10, lowest, highest)
	}
	t.Logf("comeback for rasengan: %d.%d%%", rate/10, rate%10)
}

// TestTheMirrorIsFairBeforeAnythingIsMeasuredThroughIt is the control, and it is
// a test rather than a note because it caught a real one.
//
// ⚠️ The first version of this harness swapped the two **sides** between the two
// halves of the sweep and left the roster order alone. The turn queue breaks a tie
// by enlistment, so whichever kit was written first was enlisted first in both
// halves — and a unit fighting an identical copy of itself read **58.8%**. Every
// figure taken through it was that advantage plus the kit, and the advantage was
// the larger half.
//
// Swapping the kits rather than the sides is the fix, and this is what says so:
// the same kit against itself has to read exactly even, or nothing measured here
// means anything.
func TestTheMirrorIsFairBeforeAnythingIsMeasuredThroughIt(t *testing.T) {
	for _, trait := range []string{"endurance", "swiftness"} {
		if rate := mirrorRate(t, narutoBase, narutoBase, trait); rate != scale.Base/2 {
			t.Errorf("a kit holding %s against an identical copy of itself wins %d.%d%%, "+
				"want exactly half: the instrument has a side to it and everything "+
				"measured through it is that plus the kit", trait, rate/10, rate%10)
		}
	}
}

// mirrorRate fights one Naruto kit against another and reports the share the
// first one takes, in parts per thousand.
//
// Both ways round, and it is the **kits** that change places rather than the
// sides — see the control above for what the other arrangement measured.
func mirrorRate(t *testing.T, mine, theirs []string, trait string) int {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	values, affinity := narutoLine(t)
	won, lost := 0, 0
	for _, first := range []bool{true, false} {
		for which := 1; which <= comebackSeeds; which++ {
			ally := battle.Roster{ID: "mine", Side: hex.SideAlly, Slot: mirrorSlot,
				Affinity: affinity, Stats: values, Skills: mine, Passives: []string{trait}}
			enemy := battle.Roster{ID: "theirs", Side: hex.SideEnemy, Slot: mirrorSlot,
				Affinity: affinity, Stats: values, Skills: theirs, Passives: []string{trait}}
			if !first {
				ally.Skills, enemy.Skills = theirs, mine
			}
			fight, err := battle.New(books, uint64(which), []battle.Roster{ally, enemy})
			if err != nil {
				t.Fatalf("new battle: %v", err)
			}
			fight.Begin()
			if _, err := fight.RunToEnd(4000); err != nil {
				t.Fatalf("run: %v", err)
			}
			winner, decided := fight.Winner()
			if !decided {
				continue
			}
			// The side that took it holds `mine` on the first pass and `theirs` on
			// the second, which is the whole point of running both.
			if (winner == hex.SideAlly) == first {
				won++
			} else {
				lost++
			}
		}
	}
	if won+lost == 0 {
		t.Fatal("no battle ended, so nothing was measured")
	}
	return won * scale.Base / (won + lost)
}

// narutoLine is Naruto's furthest form as the balance tests field a character.
func narutoLine(t *testing.T) (progression.Values, element.Affinity) {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	character, known := book.Get("naruto.naruto")
	if !known {
		t.Fatal("no character naruto.naruto")
	}
	values, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		t.Fatalf("resolve naruto: %v", err)
	}
	if len(narutoComeback) != cast.SkillSlots {
		t.Fatalf("the measured kit spends %d of %d slots", len(narutoComeback), cast.SkillSlots)
	}
	return values, character.Element
}
