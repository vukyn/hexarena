package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestTheShippedAntiSustainAnswerIsCarryable is the data claim, and it is here
// rather than left to the goldens because a mechanism nothing fields is a
// mechanism nothing measures — the finding the regeneration and the restore bugs
// both came back with.
//
// Three facts have to hold together for the heal cut to be a thing a player meets:
// something must apply it, somebody must be able to bring that thing, and the
// answer to it must be reachable too. Written down rather than derived, so a data
// edit that quietly takes the rider off fire_fang is a red test rather than a
// golden diff nobody reads.
func TestTheShippedAntiSustainAnswerIsCarryable(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	fang, err := books.Skills.Lookup("fire_fang")
	if err != nil {
		t.Fatalf("look fire_fang up: %v", err)
	}
	// The rider, on the skill the fire kit already got burn-through-a-shield on.
	// Two strikes as well, which is what makes #181's one-eaten-one-through branch
	// reachable from shipped data at all.
	found := false
	for _, application := range fang.Applies {
		if application.Status == "fester" {
			found = true
			if application.Chance != 500 {
				t.Errorf("fire_fang applies fester at %d per mille, want 500", application.Chance)
			}
			if application.Stacks > 1 {
				t.Errorf("fire_fang applies %d stacks of fester, want one", application.Stacks)
			}
		}
	}
	if !found {
		t.Errorf("fire_fang applies %v, so nothing shipped delivers a heal cut", fang.Applies)
	}
	if fang.Strikes < 2 {
		t.Errorf("fire_fang strikes %d times, and the multi-strike rider branch rests on it striking twice",
			fang.Strikes)
	}

	// The answer. rapid_spin is the only cleanse a sustain build brings, so a
	// heal cut it could not strip would be an anti-sustain status with no counter.
	spin, err := books.Skills.Lookup("rapid_spin")
	if err != nil {
		t.Fatalf("look rapid_spin up: %v", err)
	}
	if spin.Strips == nil {
		t.Fatal("rapid_spin strips nothing at all")
	}
	stripped := false
	for _, category := range spin.Strips.Categories {
		if category == status.HealCut {
			stripped = true
		}
	}
	if !stripped {
		t.Errorf("rapid_spin strips %v, which does not include a heal cut", spin.Strips.Categories)
	}

	// And the status itself, at the numbers it was measured at. Spelled out rather
	// than read back off the book, like the regeneration's frozen tick: a change to
	// either number should arrive here as a failure rather than as a test agreeing
	// with whatever the data now says.
	fester, err := books.Statuses.Lookup("fester")
	if err != nil {
		t.Fatalf("look fester up: %v", err)
	}
	if fester.Category != status.HealCut {
		t.Errorf("fester is a %s, want a heal cut", fester.Category)
	}
	if fester.HealShare != -400 || fester.MaxStacks != 2 || fester.Duration != 2 {
		t.Errorf("fester is %d per mille a stack, %d stacks, %d turns; want -400, 2, 2",
			fester.HealShare, fester.MaxStacks, fester.Duration)
	}
	// Two stacks leave a trickle rather than shutting healing off: the engine's
	// standing preference is to saturate rather than hard-clamp, and full negation
	// at the cap is a hard shutdown.
	if fester.HealShare*fester.MaxStacks <= -1000 {
		t.Errorf("fester at its cap comes to %d per mille, which is total negation: a heal cut is meant to leave a trickle",
			fester.HealShare*fester.MaxStacks)
	}
}

// TestTheShippedHealCutCutsTheShippedRegeneration is the engine meeting the data,
// and it is the answer to the question the whole feature was built for: does the
// sustain build's own healing actually come down.
//
// It is measured against a control on the same board and the same seed, because a
// single figure says nothing — 640 and 128 are the shipped regeneration's payout
// before and after, and only the pair says which of the two the cut produced.
//
// The status goes on by hand, which is deliberate and is not the delivery being
// skipped: fire_fang lands it at 500 per mille, so a hand-played battle would need
// a loop over rolls to get it on, and the roll is not what this test is about.
// TestTheShippedHealCutIsDeliveredByACast drives the real cast; the same figures
// as internal/core/battle's paths, on the shipped numbers, are what this one is.
func TestTheShippedHealCutCutsTheShippedRegeneration(t *testing.T) {
	// The shipped figures, spelled out: regrowth is tick power 400, so a stack
	// freezes 320 against an attack of 800 and two stacks tick 640. Two stacks of
	// fester are -800, so 200 per mille of that lands.
	const whole, cut = 640, 128

	drive := func(festered bool) (int64, int) {
		books, err := seed.Books()
		if err != nil {
			t.Fatalf("load books: %v", err)
		}
		fight, err := battle.New(books, 3, []battle.Roster{
			{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: mustAffinity(t, "water"), Stats: benchStats(3000, 800, 400, 200),
				Skills: []string{"aqua_ring", "water_gun"}},
			{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: mustAffinity(t, "water"), Stats: benchStats(4800, 500, 300, 20),
				Skills: []string{"water_gun"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		fight.Drain()
		healer, ok := fight.Unit("healer")
		if !ok {
			t.Fatal("no healer in the battle")
		}
		healer.HP = 900

		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "healer" {
			t.Fatalf("the first turn went to %s", prompt.Unit)
		}
		if err := fight.Act("aqua_ring", healer.Cell); err != nil {
			t.Fatalf("aqua_ring: %v", err)
		}
		fight.Drain()
		if festered {
			fester, err := books.Statuses.Lookup("fester")
			if err != nil {
				t.Fatalf("look fester up: %v", err)
			}
			for range fester.MaxStacks {
				if added, _ := healer.Statuses.Apply(fester, 0); !added {
					t.Fatal("the shipped fester would not stack to its own cap")
				}
			}
		}

		// On to the healer's next turn, which is where a regeneration pays out.
		for range 10 {
			next, err := fight.Advance()
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			if next == nil {
				break
			}
			events := fight.Drain()
			if !next.Skipped {
				acted := false
				for _, option := range next.Options {
					if !option.Available() {
						continue
					}
					if err := fight.Act(option.Skill, option.Aims[0]); err != nil {
						t.Fatalf("act %s: %v", option.Skill, err)
					}
					acted = true
					break
				}
				events = append(events, fight.Drain()...)
				if !acted {
					break
				}
			}
			for _, event := range events {
				if event.Kind != battle.Healed || event.Actor != "healer" ||
					event.Status != "regrowth" {
					continue
				}
				return event.Amount, event.Reduced
			}
		}
		t.Fatal("the shipped regeneration never healed, so nothing here measured a cut")
		return 0, 0
	}

	clean, reduced := drive(false)
	if clean != whole {
		t.Fatalf("the shipped regeneration healed %d with nothing on the healer, want %d: the control moved and every figure below is against it",
			clean, whole)
	}
	if reduced != 0 {
		t.Errorf("a clean heal carried a cut of %d", reduced)
	}

	got, cutShare := drive(true)
	if got != cut {
		t.Errorf("the shipped regeneration healed %d under two stacks of fester, want %d", got, cut)
	}
	if cutShare != 800 {
		t.Errorf("the heal carried a cut of %d per mille, want 800", cutShare)
	}
	if got >= clean {
		t.Errorf("festering healed %d against a clean %d, so the debuff cost nothing", got, clean)
	}
}

// TestTheShippedHealCutIsDeliveredByACast is the other half: the rider on
// fire_fang actually lands on a real board, through inflict, at its declared
// chance.
//
// Hand-played, and it loops until the roll goes its way rather than picking a seed
// that happens to work on the first cast — a seed chosen to land a 500 per mille
// roll is a fixture that stops working the day anything upstream of it consumes a
// different number of draws.
func TestTheShippedHealCutIsDeliveredByACast(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	fight, err := battle.New(books, 7, []battle.Roster{
		{ID: "biter", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "fire"), Stats: benchStats(3000, 800, 400, 200),
			Skills: []string{"fire_fang"}},
		// Tough but inside the joint health-and-defence budget battle.New checks,
		// because the healer has to survive forty casts of a two-strike fire skill.
		{ID: "healer", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "water"), Stats: benchStats(4800, 100, 300, 1),
			Skills: []string{"water_gun"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	healer, ok := fight.Unit("healer")
	if !ok {
		t.Fatal("no healer in the battle")
	}
	biter, ok := fight.Unit("biter")
	if !ok {
		t.Fatal("no biter in the battle")
	}
	target, back := healer.Cell, biter.Cell

	casts, resisted := 0, 0
	for range 40 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		// The healer is kept alive, because fire_fang is meant to be cast many
		// times and a corpse cannot be festered.
		healer.HP = healer.MaxHP()
		if prompt.Skipped {
			continue
		}
		if prompt.Unit != "biter" {
			if err := fight.Act("water_gun", back); err != nil {
				t.Fatalf("water_gun: %v", err)
			}
			fight.Drain()
			continue
		}
		if err := fight.Act("fire_fang", target); err != nil {
			t.Fatalf("fire_fang: %v", err)
		}
		casts++
		for _, event := range fight.Drain() {
			switch {
			case event.Kind == battle.StatusApplied && event.Status == "fester":
				if event.Chance != 500 {
					t.Errorf("fire_fang rolled fester against %d per mille, want its declared 500",
						event.Chance)
				}
				if event.Skill != "fire_fang" {
					t.Errorf("the fester came from %q rather than from fire_fang", event.Skill)
				}
				if healer.Statuses.Stacks("fester") == 0 {
					t.Error("the log says fester landed and the unit does not carry it")
				}
				return
			case event.Kind == battle.StatusResisted && event.Status == "fester":
				resisted++
			}
		}
	}
	t.Fatalf("fire_fang was cast %d times, rolled fester %d of them, and never landed one",
		casts, resisted)
}
