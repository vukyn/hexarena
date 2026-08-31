package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// The two stack counts the fixture statuses are built around, named so the
// arithmetic below reads as arithmetic rather than as magic.
//
// festerStacks of the fixture `fester` is -400 a stack, so two of them let
// 200 per mille of a heal through; gangreneStacks of `gangrene` is -600 a stack,
// so three of them ask for -1800, which is past total negation and is what the
// floor is for.
const (
	festerStacks   = 2
	festerLanding  = scale.Base + festerStacks*-400
	gangreneStacks = 3
)

// aHealableAlly is one ally that can be hurt, festered and healed on demand,
// with a harmless enemy in front of it so the battle is legal and the ally is
// always the one holding the turn.
//
// The ally is fast and the enemy is slow and armed with `jab`, so every Advance
// in these tests hands the turn to the ally. Health is set directly — the same
// thing restore_test.go's fixture does — because the amount of ROOM the unit has
// is the variable one of these tests turns.
func aHealableAlly(t *testing.T, traits []string, health int64) (
	battle.Books, *battle.Battle, *battle.Unit) {
	t.Helper()
	shared := books(t)
	fight := mustBattle(t, shared, 5, []battle.Roster{
		{ID: "ally", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills:   []string{"tonic", "drink", "strike"},
			Passives: traits},
		{ID: "foe", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 100, 300, 1),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	ally, ok := fight.Unit("ally")
	if !ok {
		t.Fatal("no ally in the battle")
	}
	ally.HP = health
	return shared, fight, ally
}

// afflict puts stacks of a declared status straight onto a unit's own Set.
//
// Straight on rather than through a skill on purpose: `inflict` rolls, and every
// test in this file is about the ARITHMETIC of a heal rather than about whether a
// rider lands. The rider's own route through inflict is exercised where it
// belongs — shield_rider_test.go drives it off a real cast, and internal/seed
// drives the shipped one — so putting it on by hand here keeps these assertions
// free of a chance nobody is measuring.
//
// The tick amount is handed in because a Stack freezes one when it is applied: a
// regeneration applied with nought heals nought, which would make the regen row
// of the table below pass for the wrong reason.
func afflict(t *testing.T, shared battle.Books, unit *battle.Unit, id string, stacks int, tick int64) {
	t.Helper()
	kind, err := shared.Statuses.Lookup(id)
	if err != nil {
		t.Fatalf("look %s up: %v", id, err)
	}
	for i := 0; i < stacks; i++ {
		if added, _ := unit.Statuses.Apply(kind, tick); !added {
			t.Fatalf("%s would not take stack %d of %d", id, i+1, stacks)
		}
	}
	if got := unit.Statuses.Stacks(id); got != stacks {
		t.Fatalf("%s went on %d deep, want %d", id, got, stacks)
	}
}

// healedOnTheTurn takes one of the ally's turns and reports the healing the log
// recorded on it: the total amount and the cut share the events carried.
//
// Off the log rather than off the health, for restore_test.go's reason — a unit
// hurt back inside the same turn would make an assertion about health pass on a
// heal that never happened — and it returns the event count so a test can say a
// path produced exactly one Healed rather than none or three.
//
// ⚠️ Battle.Drain EMPTIES the buffer, so the caller drains after Begin (the
// fixture does) and this drains after Advance and again after the act. A second
// drain around one act sees nothing, and skipping one lets the enemy's events
// pile into the batch that is read.
func healedOnTheTurn(t *testing.T, fight *battle.Battle, id string, aim hex.Offset) []battle.Event {
	t.Helper()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "ally" {
		t.Fatalf("the turn did not go to the ally: %+v", prompt)
	}
	// The start-of-turn tick has already resolved by the time a prompt is
	// handed back, so a regeneration heals into this batch.
	before := fight.Drain()
	if id != "" {
		if err := fight.Act(id, aim); err != nil {
			t.Fatalf("act %s: %v", id, err)
		}
	}
	return find(append(before, fight.Drain()...), battle.Healed)
}

// oneHeal is the single Healed event a turn produced, and it fails on none and on
// several: a path that healed twice is not the path the row names, and a path that
// healed nothing would make every assertion downstream pass by absence.
func oneHeal(t *testing.T, what string, healed []battle.Event) battle.Event {
	t.Helper()
	if len(healed) != 1 {
		t.Fatalf("%s produced %d healed events, want one: %+v", what, len(healed), healed)
	}
	return healed[0]
}

// TestEveryHealingPathTakesTheHealCut is the anti-vacuity test for the whole
// mechanic: five things in the game give health back, they go through exactly
// two functions, and the cut is defined once — so every one of the five has to
// show it.
//
// The five sources are a skill's `restores`, a skill's `drains`, a `regen`
// status's tick, a trait's `drains` and comeback's `at_empty` (which is a
// `restores` and takes the restores row's path). Each row here drives one of
// them for real and is measured against ITS OWN control, because the amounts
// differ per path and there is no figure they could share.
//
// Two things make it non-vacuous. Every row asserts its control healed something
// — an absence in the festered arm otherwise reads as "this path heals nothing
// here" — and the table is checked against a written-down list of the sources, so
// adding a healing source without a row is a red test rather than a silent gap.
func TestEveryHealingPathTakesTheHealCut(t *testing.T) {
	// The sources, written down rather than counted off the table: extending the
	// table is not the same act as claiming the extension was measured.
	wanted := map[string]bool{
		"a skill's restores": false, "a skill's drains": false,
		"a regen status's tick": false, "a trait's drains": false,
	}
	for _, path := range []struct {
		// source names which of the five this row drives, and must be in wanted.
		source string
		traits []string
		// regen puts a regeneration on before the turn, since a tick is the one
		// source no cast produces.
		regen bool
		// skill is the cast that heals, empty for the tick.
		skill string
		aim   hex.Offset
		// drained is the share the event must carry, so a row cannot quietly be
		// measuring a different path from the one it names.
		drained int
		status  string
	}{
		{source: "a skill's restores", skill: "tonic", aim: ownCell},
		{source: "a skill's drains", skill: "drink", aim: acrossTheBoard, drained: 1000},
		{source: "a regen status's tick", regen: true, status: "mending"},
		{source: "a trait's drains", traits: []string{"thirst"},
			skill: "strike", aim: acrossTheBoard, drained: 250},
	} {
		t.Run(path.source, func(t *testing.T) {
			if _, named := wanted[path.source]; !named {
				t.Fatalf("%q is not one of the healing sources this test claims to cover", path.source)
			}
			wanted[path.source] = true

			drive := func(festered bool) battle.Event {
				shared, fight, ally := aHealableAlly(t, path.traits, 900)
				if path.regen {
					afflict(t, shared, ally, "mending", 1, 400)
				}
				if festered {
					afflict(t, shared, ally, "fester", festerStacks, 0)
				}
				heal := oneHeal(t, path.source, healedOnTheTurn(t, fight, path.skill, path.aim))
				// The shape of the event, so a row cannot quietly be measuring a
				// different source from the one it names: a drain says its share
				// and a regeneration names the status that healed.
				if heal.Drained != path.drained {
					t.Fatalf("%s healed with a drain share of %d, want %d: this row is not driving the path it names",
						path.source, heal.Drained, path.drained)
				}
				if heal.Status != path.status {
					t.Fatalf("%s healed from status %q, want %q", path.source, heal.Status, path.status)
				}
				return heal
			}

			// The control first, because everything below is measured against it.
			control := drive(false)
			if control.Amount <= 0 {
				t.Fatalf("%s healed %d with nothing on the unit, so this row measures nothing",
					path.source, control.Amount)
			}
			if control.Reduced != 0 {
				t.Errorf("%s carried a cut of %d with no heal-cut on the unit",
					path.source, control.Reduced)
			}

			cut := drive(true)
			want := control.Amount * festerLanding / scale.Base
			if cut.Amount != want {
				t.Errorf("%s healed %d under %d stacks of fester, want %d (%d cut to %d per mille)",
					path.source, cut.Amount, festerStacks, want, control.Amount, festerLanding)
			}
			if cut.Reduced != scale.Base-festerLanding {
				t.Errorf("%s carried a cut of %d, want %d", path.source, cut.Reduced,
					scale.Base-festerLanding)
			}
		})
	}
	for source, driven := range wanted {
		if !driven {
			t.Errorf("%s was never exercised, so the cut is unmeasured on it", source)
		}
	}
}

// TestTheCutComesOffBeforeTheHealIsCappedAtTheRoom is the ordering, and it is
// the one test the wrong order passes everywhere else.
//
// heal and drain both cap the amount at the room left to full. Capping first
// hands the reduction a number that is already the room rather than the heal, so
// on a nearly-full unit the cut comes off health that was going to be thrown away
// and the debuff is invisible — which is exactly the unit a sustain build spends
// its turns being.
//
// Three arms, and all three are needed, because the two orders differ only on a
// unit with less room than the heal:
//
//	room       = half the payout, so the uncut heal is capped and the cut one is not
//	right      = payout * landing, which is under the room
//	wrong      = room * landing, the cap-then-reduce answer
//
// The payout is read off the control rather than written down: it is
// combat.Rules.Restore's figure and a second copy of it here would be the
// mistake the repository keeps a list of.
func TestTheCutComesOffBeforeTheHealIsCappedAtTheRoom(t *testing.T) {
	// The payout, measured on a unit with room to spare.
	_, spacious, _ := aHealableAlly(t, nil, 900)
	payout := oneHeal(t, "the restore", healedOnTheTurn(t, spacious, "tonic", ownCell)).Amount
	if payout <= 0 {
		t.Fatalf("the restore paid %d, so nothing here has a payout to compare", payout)
	}

	room := payout / 2
	right := payout * festerLanding / scale.Base
	wrong := room * festerLanding / scale.Base
	if right <= wrong || right >= room {
		t.Fatalf("the fixture does not discriminate: payout %d, room %d, right %d, wrong %d",
			payout, room, right, wrong)
	}

	// Nearly full and clean: the payout is capped at the room, which is what
	// makes this the interesting unit.
	_, capped, ally := aHealableAlly(t, nil, 0)
	ally.HP = ally.MaxHP() - room
	uncut := oneHeal(t, "a nearly-full clean unit", healedOnTheTurn(t, capped, "tonic", ownCell)).Amount
	if uncut != room {
		t.Fatalf("a nearly-full unit healed %d of a %d payout, want the room of %d",
			uncut, payout, room)
	}

	// Nearly full and festering: the same unit, the same cast.
	shared, festering, hurt := aHealableAlly(t, nil, 0)
	hurt.HP = hurt.MaxHP() - room
	afflict(t, shared, hurt, "fester", festerStacks, 0)
	got := oneHeal(t, "a nearly-full festering unit", healedOnTheTurn(t, festering, "tonic", ownCell)).Amount
	switch got {
	case right:
	case wrong:
		t.Errorf("a nearly-full festering unit healed %d, which is the room cut rather than the heal cut: the reduction is being applied after the cap", got)
	case uncut:
		t.Errorf("a nearly-full festering unit healed %d, the same as a clean one: the cut is invisible on exactly the unit it is for", got)
	default:
		t.Errorf("a nearly-full festering unit healed %d, want %d (payout %d cut to %d per mille, room %d)",
			got, right, payout, festerLanding, room)
	}
}

// TestNoAmountOfHealCutTurnsAHealIntoDamage is the floor.
//
// Set.HealShare accumulates per stack and bounds nothing — that is deliberate,
// because the bound belongs where the amount is — so authored data reaches past
// total negation without trying: three stacks of the fixture's -600 ask for
// -1800. Unfloored, the multiplier is negative, the heal is negative, and a
// caller that only knows how to add health subtracts it.
//
// Asserted on the health as well as on the log, because the two failures look
// different: an unfloored multiplier shows up as a Healed event carrying a
// negative amount OR as health that went down on a turn nothing attacked.
func TestNoAmountOfHealCutTurnsAHealIntoDamage(t *testing.T) {
	shared, fight, ally := aHealableAlly(t, nil, 900)
	afflict(t, shared, ally, "gangrene", gangreneStacks, 0)
	if share := ally.Statuses.HealShare(); share > -scale.Base {
		t.Fatalf("the fixture asks for %d per mille, which is inside total negation: nothing here reaches the floor", share)
	}
	before := ally.HP

	if healed := healedOnTheTurn(t, fight, "tonic", ownCell); len(healed) != 0 {
		t.Errorf("a heal past total negation landed %+v, want nothing at all", healed)
	}
	if ally.HP < before {
		t.Errorf("the ally went %d -> %d on a heal, so the cut became damage", before, ally.HP)
	}
	if ally.HP != before {
		t.Errorf("the ally went %d -> %d on a heal that was cut to nothing", before, ally.HP)
	}

	// The control: the same cast on a clean unit heals, so the absence above is
	// the floor rather than the fixture.
	_, clean, _ := aHealableAlly(t, nil, 900)
	whole := oneHeal(t, "a clean unit", healedOnTheTurn(t, clean, "tonic", ownCell)).Amount
	if whole <= 0 {
		t.Fatalf("the same cast on a clean unit healed %d, so nothing above measured the floor", whole)
	}
}
