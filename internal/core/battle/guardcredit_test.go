package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
)

// A guard the rating will actually spend, from both ends: that a board of them
// does not stand still, and that spending one is still worth less than taking
// health.
//
// The pair is the point. The first alone is satisfied by crediting a blow at its
// whole value, which puts `pastAPool`'s own finding back — the rating aiming at a
// target it cannot empty. The second alone is satisfied by crediting nothing,
// which is where this started.

// guardedMirror is two identical units, each carrying a permanent absorbing pool,
// and a kit whose only skill has a cooldown.
//
// ⚠️ **The cooldown is the fixture's whole point rather than a detail.** A rating
// that values every option at nought still casts a *cooldownless* one — Suggest's
// fallback arm takes it, because a free cast costs the turn and no more. So a kit
// of cooldownless skills chips the pool down by accident and hides the defect
// completely: measured before the fix, the same board with `daze` in hand cast
// `daze` and took two points off the pool, while this one took nothing at all in
// six hundred turns.
func guardedMirror(t *testing.T, seed uint64) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), seed, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 100),
			Skills: []string{"clout"}, Passives: []string{"carapaced"}},
		{ID: "e", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 100),
			Skills: []string{"clout"}, Passives: []string{"carapaced"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// TestAGuardedMirrorDoesNotStandStill is the defect this credit exists for.
//
// A blow a pool would absorb whole used to rate nought, so Suggest passed rather
// than throwing it, so the pool was never depleted and the next turn asked the
// same question. Measured on this exact board before the fix: six hundred turns,
// **not one cast**, both units at 3600 of 3600 with their pools untouched — and
// the turn limit reported a long battle rather than a broken one, which is why it
// shipped.
//
// What is asserted is progress rather than a winner. These two are exact mirrors
// with a hundred-power skill against thirty-six hundred health, so the board is
// slow by construction; a test demanding a decision inside the limit would be
// measuring the fixture's damage figures rather than the rating's willingness to
// act.
func TestAGuardedMirrorDoesNotStandStill(t *testing.T) {
	fight := guardedMirror(t, 3)
	if _, err := fight.RunToEnd(600); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, id := range []string{"a", "e"} {
		unit, known := fight.Unit(id)
		if !known {
			t.Fatalf("%s left the board", id)
		}
		if pool := unit.Statuses.PoolIn(status.Absorb); pool > 0 {
			t.Errorf("%s still holds %d of pool after six hundred turns: nobody is throwing "+
				"anything at it, so the board cannot resolve", id, pool)
		}
		if unit.HP == unit.MaxHP() {
			t.Errorf("%s stands at full health after six hundred turns: the guard was spent "+
				"and the blows behind it still went nowhere", id)
		}
	}
}

// TestABlowThatTakesHealthOutratesOneThatOnlyTakesGuard is the ceiling on the
// share, and it is the finding pastAPool was written for in the first place:
// offered two identical enemies, one of them carrying a pool, the rating must
// prefer the one it can actually hurt.
//
// ⚠️ **It fails as a TIE rather than as a reversal** if the credit is ever
// raised to the whole blow: at parity `take` keeps the first aim it saw, so the
// answer is decided by the aim walk rather than by the rating.
//
// ⚠️ **That is why the guarded enemy stands in the slot the walk reaches
// FIRST, and the first version of this fixture had them the other way round and
// passed for nothing.** Measured: with the bare enemy walked first, the test was
// green at a credit of 1000 as well as at 500 — the tie fell to the bare one by
// slot order and the assertion never took. Swapped, it is red at 1000 and green
// at 500, which is the ceiling actually being held. A fixture whose ties never
// arise is the shape this repository keeps a list of.
func TestABlowThatTakesHealthOutratesOneThatOnlyTakesGuard(t *testing.T) {
	guarded := hex.Offset{Col: 2, Row: 0}
	bare := hex.Offset{Col: 2, Row: 1}
	fight, err := battle.New(books(t), 3, []battle.Roster{
		{ID: "hitter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 200),
			Skills: []string{"clout"}},
		{ID: "soft", Side: hex.SideEnemy, Slot: bare,
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 1),
			Skills: []string{"clout"}},
		{ID: "walled", Side: hex.SideEnemy, Slot: guarded,
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 1),
			Skills: []string{"clout"}, Passives: []string{"carapaced"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "hitter" {
		t.Fatalf("the first turn is %v, and this measures the hitter's choice", prompt)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("the rating declined a turn with two enemies in reach")
	}
	walled, known := fight.Unit("walled")
	if !known || walled.Statuses.PoolIn(status.Absorb) == 0 {
		t.Fatal("the guarded enemy holds no pool, so this fixture measures nothing")
	}
	if choice.Aim == hex.Place(hex.SideEnemy, guarded) {
		t.Errorf("the rating aimed at the enemy carrying a pool over the one standing bare: " +
			"a point taken out of a guard is being valued at a point of health, which is " +
			"where the softest target on the board becomes the preferred one")
	}
}

// TestOnlyAGuardThatStaysSpentEarnsCredit is the distinction the credit rests on,
// and it is the one the first version of this change did not make.
//
// ⚠️ **A flat credit — every guard, timed or not — is not shippable, and it was
// measured rather than argued.** With one, `pokemon.happiny` and
// `pokemon.squirtle` against copies of themselves go from resolving every seed to
// **40 of 40 endless**, which breaks `TestABothWaysMirrorIsExactlyEven`: a
// fairness invariant, and the same one that got a `stat_debuff` shield change
// refused. The share cannot buy its way out — the shipped mirrors need it at ten
// or under and the guarded fixture needs it at fifty or over, so the two
// requirements have an empty intersection.
//
// The reason is what the two guards are. `withdraw` puts up `block`, which lasts
// two turns and is cast again; `carapace` grants `bastion`, which is permanent,
// refused a second stack, and gone for good once it is empty. Taking a bite out
// of the first buys the turn it takes to come back. Taking one out of the second
// is the only kind of progress this rating can honestly call progress.
//
// ⚠️ **The timed enemy stands in the slot the aim walk reaches FIRST**, for the
// reason the test above records: a credit that does not tell the two apart leaves
// them tied, and `take` keeps the first aim it saw. Put the permanent one first
// and this passes with the distinction deleted.
func TestOnlyAGuardThatStaysSpentEarnsCredit(t *testing.T) {
	timed := hex.Offset{Col: 2, Row: 0}
	lasting := hex.Offset{Col: 2, Row: 1}
	fight, err := battle.New(books(t), 3, []battle.Roster{
		{ID: "hitter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 200),
			Skills: []string{"clout"}},
		{ID: "renewable", Side: hex.SideEnemy, Slot: timed,
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 1),
			Skills: []string{"clout"}},
		{ID: "spent", Side: hex.SideEnemy, Slot: lasting,
			Affinity: single("neutral"), Stats: stats(3600, 500, 400, 1),
			Skills: []string{"clout"}, Passives: []string{"carapaced"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	// The two pools are made equal on purpose: what is being measured is whether
	// the guard comes back, not how deep it is.
	permanent, known := fight.Unit("spent")
	if !known {
		t.Fatal("the permanently guarded enemy is not on the board")
	}
	pool := permanent.Statuses.PoolIn(status.Absorb)
	if pool <= 0 {
		t.Fatal("the permanent guard holds no pool, so this fixture measures nothing")
	}
	kind, err := fight.Books().Statuses.Lookup("aegis")
	if err != nil {
		t.Fatalf("lookup aegis: %v", err)
	}
	if kind.Permanent {
		t.Fatal("the timed guard in this fixture is permanent, so both arms are the same arm")
	}
	renewable, _ := fight.Unit("renewable")
	renewable.Statuses.Apply(kind, pool)
	if got := renewable.Statuses.PoolIn(status.Absorb); got != pool {
		t.Fatalf("the timed pool came out at %d against the permanent one's %d: the two "+
			"arms differ in depth as well as in kind", got, pool)
	}
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "hitter" {
		t.Fatalf("the first turn is %v, and this measures the hitter's choice", prompt)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("the rating declined a turn with two enemies in reach")
	}
	if choice.Aim != hex.Place(hex.SideEnemy, lasting) {
		t.Errorf("the rating aimed at the enemy whose guard comes back in two turns over " +
			"the one whose guard is gone for good: a bite out of a renewable guard is " +
			"being counted as progress, which is what turns every shipped mirror endless")
	}
}
