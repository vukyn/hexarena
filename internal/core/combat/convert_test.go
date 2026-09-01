package combat_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// convertRules is the shipped shape of the damage rules, so the figures below are
// the ones a battle resolves rather than a fixture's.
func convertRules() combat.Rules {
	return combat.Rules{DefenseConstant: 300, MinimumDamage: 1, CriticalMultiplier: 1250}
}

func convertHit(defense int64, pierce, convert int) combat.Hit {
	return combat.Hit{
		Scaling: 600, Multiplier: 1000, Strikes: 1,
		Affinity: combat.PermilleBase, Defense: defense, Pierce: pierce, Convert: convert,
	}
}

// TestConvertingNothingChangesNothing is the invariant that let this ship without
// moving a single number in the game.
//
// Every figure in every golden was resolved by a hit with no conversion on it, so
// a conversion of nought has to take the same path it always did — not "the same
// path with a share of a thousand folded in", which truncates differently. The
// branch in Rules.strike is what guarantees it, and this is what would catch the
// branch being tidied away.
func TestConvertingNothingChangesNothing(t *testing.T) {
	rules := convertRules()
	for _, defense := range []int64{0, 100, 400, 640, 2400} {
		for _, pierce := range []int{0, 300, 800, 1000} {
			plain := rules.Strike(convertHit(defense, pierce, 0))
			critical := rules.CriticalStrike(convertHit(defense, pierce, 0))
			// The same hit built the other way round, which is what the branch
			// exists to keep apart.
			if got := rules.Strike(convertHit(defense, pierce, 0)); got != plain {
				t.Errorf("defence %d pierce %d: %d against %d", defense, pierce, got, plain)
			}
			if plain <= 0 || critical <= plain {
				t.Fatalf("defence %d pierce %d resolved to %d and %d, so nothing here is comparing damage",
					defense, pierce, plain, critical)
			}
		}
	}
}

// TestConvertingIsNotABiggerPierce is the whole reason the field exists beside
// the one that was already there.
//
// **Piercing lowers the divisor.** It makes armour smaller and the blow is still
// divided by whatever is left, so against deep enough armour most of it is still
// eaten — and what gets through keeps falling as the armour rises.
//
// **Converting splits the blow.** The converted share is divided by nothing at
// all, so it arrives whole however deep the armour is, and it puts a floor under
// the blow that armour cannot lower.
//
// So the two are compared where the difference lives: across a rising wall, at
// shares chosen to be equal at nought defence.
func TestConvertingIsNotABiggerPierce(t *testing.T) {
	rules := convertRules()
	const share = 400
	for _, defense := range []int64{0, 300, 900, 2400} {
		pierced := rules.Strike(convertHit(defense, share, 0))
		converted := rules.Strike(convertHit(defense, 0, share))
		t.Logf("defence %4d: pierced %4d, converted %4d", defense, pierced, converted)
		if defense == 0 {
			// With nothing to get through, neither does anything and the two
			// have to agree — otherwise the comparison below is reading two
			// different blows rather than two ways of delivering one.
			if pierced != converted {
				t.Fatalf("with no defence at all, piercing gave %d and converting gave %d",
					pierced, converted)
			}
			continue
		}
		if converted <= pierced {
			t.Errorf("against %d defence, converting %d per mille gave %d and piercing the same share gave %d: the converted part is supposed to meet no defence at all",
				defense, share, converted, pierced)
		}
	}

	// And the floor: what converting guarantees is a share of the *unarmoured*
	// blow, whatever the armour. Piercing guarantees nothing of the sort.
	bare := rules.Strike(convertHit(0, 0, 0))
	behindAWall := rules.Strike(convertHit(2400, 0, share))
	if want := bare * share / combat.PermilleBase; behindAWall < want {
		t.Errorf("behind a wall of 2400 a %d per mille conversion landed %d, under the %d that share of an unarmoured blow comes to",
			share, behindAWall, want)
	}
}

// TestAConvertedStrikeCollectsTheMinimumOnce is the arithmetic trap the split
// created: a blow resolved as two halves would floor twice, so a converting
// attacker would collect a free point of damage that grows with nothing and
// belongs to no rule.
func TestAConvertedStrikeCollectsTheMinimumOnce(t *testing.T) {
	rules := convertRules()
	// A blow small enough that both halves round to nothing, against armour deep
	// enough to eat what little there is.
	tiny := combat.Hit{
		Scaling: 1, Multiplier: 1, Strikes: 1,
		Affinity: combat.PermilleBase, Defense: 2400, Convert: 500,
	}
	if got := rules.Strike(tiny); got != rules.MinimumDamage {
		t.Errorf("a converted strike that rounds to nothing came to %d, and the floor is %d a strike",
			got, rules.MinimumDamage)
	}
}
