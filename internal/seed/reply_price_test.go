package seed_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/seed"
)

// paralysisSeeds is how many shipped battles the guard below watches. Enough to
// make "never once" a claim rather than a coincidence, and small enough to run in
// `make check`.
const paralysisSeeds = 300

// TestTheOpponentStillAttacksAReplyHolder is the guard on the reply term, and it
// is the failure that term is most likely to cause.
//
// price.go now subtracts what a target would answer with, and an over-charge does
// not merely push an attack down the order: since an option priced below nought is
// **declined**, it removes the attack outright and the unit passes. A trait that
// answers for more than the attack is worth would therefore make a whole squad
// refuse to touch its holder — which reads as caution rather than as a bug, and
// would be measured as a longer battle rather than as a broken one.
//
// ally.venusaur carries venom_blood, the only answering trait on the shipped
// board, so it is the unit an over-charge would hide behind. The claim is that
// the enemy squad still hits it with skills, in the great majority of battles and
// in every one of which it is reachable at all.
//
// ⚠️ Skill damage only. A reply's own blow and a status tick are Damaged events
// too, and counting those would let this pass on a board where nobody ever chose
// to attack.
func TestTheOpponentStillAttacksAReplyHolder(t *testing.T) {
	const holder = "ally.venusaur"
	struck, battles := 0, 0
	for seedValue := uint64(1); seedValue <= paralysisSeeds; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: assemble: %v", seedValue, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		battles++
		for _, event := range fight.Drain() {
			if event.Kind != battle.Damaged || event.Target != holder || event.Skill == "" {
				continue
			}
			if !strings.HasPrefix(event.Actor, "foe.") {
				continue
			}
			struck++
			break
		}
	}
	if struck == 0 {
		t.Fatalf("over %d battles the enemy squad never once aimed a skill at %s: the "+
			"reply term has priced its trait above what an attack on it is worth, so "+
			"every option against it rates below nought and is declined",
			battles, holder)
	}
	// A floor rather than the measured figure, so an ordinary swing in the data does
	// not fail this — but high enough that the term going quietly wrong is caught.
	if floor := battles * 3 / 4; struck < floor {
		t.Errorf("the enemy squad attacked %s in %d of %d battles, under the %d this "+
			"guard allows: the reply on it is being over-charged",
			holder, struck, battles, floor)
	}
	t.Logf("%s was attacked with a skill in %d of %d shipped battles", holder, struck, battles)
}
