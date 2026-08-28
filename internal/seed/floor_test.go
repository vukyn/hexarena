package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestNoShippedDebuffCanFreezeAUnit is the invariant four separate guards each
// half-promise and none of them states.
//
// A speed of nought is a unit that never acts again, and a pair of them is a
// battle that cannot end. Nothing has to go wrong for that to be reachable: it
// is one authored figure away, because a status is a percentage in a JSON file
// and nobody writing one is thinking about division.
//
// The four are modifier.Set.Stat's floor at a tenth of base, which scale.Saturate
// approaches without ever touching; the same function's hard return of one; and
// atb.Wait and atb.Queue.Add, which each clamp a speed under one before dividing
// by it. That is a lot of redundancy for a rule written down nowhere, and every
// one of those guards is in a package that cannot see the shipped statuses — so
// none of them can say whether the data ever gets near.
//
// This stacks every shipped debuff onto every shipped character at once, far past
// what any of them can really hold, and asks the two questions that matter: does
// a stat survive, and does the queue still turn.
func TestNoShippedDebuffCanFreezeAUnit(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	cast, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	// Far past every max_stacks in the book, so what is measured is the floor
	// rather than the stack cap. The cap is the reason nothing shipped gets
	// close; it is not the reason nothing can.
	const piledOn = 50
	tested := 0
	for _, character := range cast.All() {
		base, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
		if err != nil {
			t.Fatalf("resolve %s: %v", character.ID, err)
		}
		crushed := status.Set{}
		for _, kind := range books.Statuses.Kinds() {
			if !kind.Category.Harmful() {
				continue
			}
			for range piledOn {
				if kind.Permanent {
					crushed.Hold(kind, 1)
					continue
				}
				crushed.Apply(kind, 0)
			}
		}
		// The saturating path is only half of it. A modifier set built from the
		// book cannot express an arbitrary penalty, and the floor has to hold
		// against one that could -- so the same stat lines go through a term far
		// larger than anything authorable.
		absurd := crushingSet(t)
		for label, live := range map[string]progression.Values{
			"every shipped debuff": crushed.Modifiers().Stats(base, books.Limits.Ceilings, books.Bounds),
			"an unauthorable one":  absurd.Stats(base, books.Limits.Ceilings, books.Bounds),
		} {
			for _, kind := range progression.Kinds() {
				if live[kind] >= 1 {
					continue
				}
				t.Errorf("%s under %s has %s of %d: no stat may reach nought, and speed at nought "+
					"is a unit that never acts again", character.ID, label, kind, live[kind])
			}
			// The queue is where a nought would actually bite, so it is asked
			// rather than reasoned about.
			if wait := atb.Wait(live[progression.Speed]); wait <= 0 {
				t.Errorf("%s under %s waits %d between turns", character.ID, label, wait)
			}
			queue := atb.New()
			if err := queue.Add(character.ID, live[progression.Speed]); err != nil {
				t.Errorf("%s under %s cannot join the queue: %v", character.ID, label, err)
			}
			tested++
		}
	}
	if tested == 0 {
		t.Fatal("no character was crushed, so nothing was measured")
	}
}

// TestTheFloorIsNeverReached is the other half, and the one that says the floor
// is a floor rather than a clamp.
//
// scale.Saturate approaches its limit and does not arrive, in both directions,
// which is what keeps a debuff worth authoring past a hundred percent: each
// further stack takes measurably less than the last instead of the first one
// taking everything and the rest doing nothing. A clamp at the same figure would
// pass the test above and lose all of that.
func TestTheFloorIsNeverReached(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	cast, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	absurd := crushingSet(t)
	for _, character := range cast.All() {
		base, _, err := character.Resolve(progression.LevelCap, progression.Furthest)
		if err != nil {
			t.Fatalf("resolve %s: %v", character.ID, err)
		}
		live := absurd.Stats(base, books.Limits.Ceilings, books.Bounds)
		for _, kind := range []progression.Kind{progression.Speed, progression.Defense} {
			floor := base[kind] * int64(books.Bounds.FloorFraction) / modifier.PercentBase
			if live[kind] <= floor {
				t.Errorf("%s crushed to %s %d, at or past the floor of %d: the floor is approached, "+
					"never reached", character.ID, kind, live[kind], floor)
			}
		}
	}
}

// crushingSet is a penalty far past anything a status file can express, which is
// what asks the floor whether it is a floor rather than asking the stack cap.
func crushingSet(t *testing.T) modifier.Set {
	t.Helper()
	set := modifier.Set{}
	if err := set.AddAll(
		modifier.Modifier{Target: modifier.Speed, Mode: modifier.Percent, Amount: -1_000_000},
		modifier.Modifier{Target: modifier.Defense, Mode: modifier.Percent, Amount: -1_000_000},
	); err != nil {
		t.Fatalf("build the crushing set: %v", err)
	}
	return set
}
