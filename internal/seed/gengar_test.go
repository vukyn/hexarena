package seed_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// The two directions, hardcoded here the way every other design record in this
// package is.
var (
	unbindBuild = []string{"spite", "curse", "shadow_claw", "night_shade"}
	miasmaBuild = []string{"poison_powder", "sludge_bomb", "venoshock", "shadow_ball"}
)

// stripSeeds is how many seeds each pairing below is fought over, both ways
// round. A hundred and fifty is three hundred battles a row, which is what the
// mender's reading uses.
const stripSeeds = 150

// regenerationSeeds is the depth the regeneration row alone is taken at.
//
// ⚠️ **Four times the rest, because that row compares two arms whose battles do
// not last the same.** Even normalised per turn the figure was still moving at
// stripSeeds — measured after ENG-012, the stripped arm gave back four fifths of
// the unstripped arm's per-turn healing over 150 seeds, and just under three
// quarters over both 600 and 1500. The other two rows count strips and blocks,
// which are counts of events rather than rates against a length, so they stay
// where they were.
const regenerationSeeds = 600

// TestAStripEarnsItsSlotOnlyAgainstSomethingToStrip is the character stated as
// one measurement, and it is asked the way the mender's slot is: in a squad,
// because a duel cannot see it.
//
// `spite` takes a shield and a regeneration off an enemy. Neither is a thing a
// single unit brings to a duel, so `hexforge spar` reads it as a seven-hundred
// power attack and says nothing — the same blind spot that made Cleffa read 0 to
// 7 per mille while being worth a slot. So the question is asked against three
// opponents that differ only in what their wall does with its fourth skill.
//
// Two Gengars, alike in everything but one skill — `spite` against `bite` —
// three hundred battles a row:
//
//	opponent wall     kit          strips   my blows blocked   healed a turn
//	shields           with spite      656                518                 11
//	                  without           0               1042                 12
//	only hits         with spite        0                  0                  0
//	                  without           0                  0                  0
//	regenerates       with spite     2283                  0                 25
//	                  without           0                  0                 35
//
// ⚠️ **The win rate is not what is being read**, and it moves 500 to 540 per
// mille against the shielding wall — a shield is not the whole of a fight. What a
// squad can say exactly is what the skill *did*: better than a third of the blows
// the enemy's block would have eaten no longer are, and the enemy's regeneration
// gives back barely a third of what it would have.
//
// ⚠️ **The middle row is what makes the other two mean anything.** Against a
// squad carrying nothing to take, the skill strips nought and is a plain attack,
// and the rate goes very slightly the *other* way. A utility skill that was worth
// its slot everywhere would not be a utility skill.
//
// ⚠️⚠️ **The regeneration row used to read a TOTAL and the claim it made was
// never established.** It held that taking a regeneration off halves it, off the
// health the wall gave back across the whole run — and a regeneration gives back
// health *per turn*, so that figure counts battle length as much as it counts
// the strip. The two arms are two different kits and their battles do not last
// the same: measured after ENG-012 closed the board's mirror, the stripped arm
// ran 48,003 turns against 32,646 and its total came out **higher** while its
// per-turn healing was a fifth lower. So the row reads a rate now, at four times
// the depth, and the claim it holds is what the rate actually says: the strip
// takes **at least a fifth** off, measured at 25 against 35 a turn. Halving it
// is not something this skill does and never was — the old number said so only
// because the lengths happened to be close. → healedPerTurn, TODO.md ENG-012.
func TestAStripEarnsItsSlotOnlyAgainstSomethingToStrip(t *testing.T) {
	for _, against := range []struct {
		name       string
		wall       []string
		strips     bool
		guards     bool
		regenerate bool
	}{
		{"a wall that shields", []string{"water_gun", "bubble", "bite", "withdraw"},
			true, true, false},
		{"a wall that only hits", []string{"water_gun", "bubble", "bite", "skull_bash"},
			false, false, false},
		{"a wall that regenerates", []string{"water_gun", "bubble", "bite", "aqua_ring"},
			true, false, true},
	} {
		t.Run(against.name, func(t *testing.T) {
			// The regeneration row is a rate against a length and needs the
			// depth; the other two count events and do not. → regenerationSeeds.
			seeds := stripSeeds
			if against.regenerate {
				seeds = regenerationSeeds
			}
			with := readStripOver(t, []string{"shadow_claw", "shadow_ball", "night_shade", "spite"},
				against.wall, seeds)
			without := readStripOver(t, []string{"shadow_claw", "shadow_ball", "night_shade", "bite"},
				against.wall, seeds)
			for _, reading := range []struct {
				name string
				got  stripReading
			}{{"with spite", with}, {"without", without}} {
				t.Logf("%-12s %4d‰  %3d strips of %4d stacks, %4d blows blocked, "+
					"%7d healed over %6d turns (%d a turn)",
					reading.name, reading.got.rate(), reading.got.strips, reading.got.stacks,
					reading.got.blocked, reading.got.healed, reading.got.turns,
					reading.got.healedPerTurn())
			}

			// The other kit never strips, whatever it is up against — it is the
			// control, and a control that stripped would make every figure here
			// a reading of two kits that both do the thing.
			if without.strips != 0 {
				t.Fatalf("the kit without the strip stripped %d times: the comparison is empty",
					without.strips)
			}
			if against.strips {
				if with.strips == 0 {
					t.Errorf("nothing was stripped against %s, so nothing here measured the skill",
						against.name)
				}
			} else if with.strips != 0 {
				t.Errorf("%d stacks came off %s, which has nothing to take", with.strips, against.name)
			}

			// A shield eats whole strikes, so what taking it off is worth is
			// counted in strikes it no longer eats.
			if against.guards {
				if with.blocked == 0 || without.blocked == 0 {
					t.Fatalf("%d and %d blows were blocked: the wall is not shielding, so the comparison is empty",
						with.blocked, without.blocked)
				}
				if with.blocked*3 >= without.blocked*2 {
					t.Errorf("the strip left %d blows blocked against %d without it: taking a shield off is supposed to be most of what it does",
						with.blocked, without.blocked)
				}
			} else if with.blocked != 0 || without.blocked != 0 {
				t.Errorf("%d and %d blows were blocked by a wall that does not shield", with.blocked, without.blocked)
			}

			// ⚠️ **The shielding wall's healing is LEVEL, and it is level only
			// per turn.** `withdraw` restores on the turn it is cast and a
			// restore cannot be stripped, so the strip must not touch this
			// figure — the doc above has said so for as long as this test has
			// existed and nothing asserted it. It is asserted here because it is
			// the one row that tells a rate from a total: measured, 11 a turn
			// against 12, while the two totals are 343,981 and 420,984, which are
			// nowhere near level. A reading that dropped the normalisation would
			// pass the regeneration row below and fail this one.
			if against.guards {
				const level = 10
				gap := with.healedPerTurn() - without.healedPerTurn()
				if gap < 0 {
					gap = -gap
				}
				if without.healedPerTurn() == 0 {
					t.Fatalf("the shielding wall healed nothing, so the comparison is empty")
				}
				if gap*100 > without.healedPerTurn()*level {
					t.Errorf("the shielding wall gave back %d a turn with the strip and "+
						"%d without: a restore is not a regeneration and cannot be taken, "+
						"so the two have to be level (%d healed over %d turns against %d "+
						"over %d)",
						with.healedPerTurn(), without.healedPerTurn(),
						with.healed, with.turns, without.healed, without.turns)
				}
			}

			// And a regeneration is counted in the health it no longer gives
			// back. ⚠️ A *restore* is not a regeneration and cannot be taken: the
			// shielding wall's `withdraw` heals on the turn it is cast, which is
			// why that row's two healing figures are level.
			if against.regenerate {
				if without.healedPerTurn() == 0 {
					t.Fatalf("the wall healed nothing without the strip, so the comparison is empty")
				}
				// A fifth, not a half. → the note on the regeneration row above,
				// which carries what the old half was actually reading.
				if with.healedPerTurn()*5 > without.healedPerTurn()*4 {
					t.Errorf("the wall gave back %d a turn against %d without the strip: "+
						"taking a regeneration off is supposed to be worth a fifth of it "+
						"at least (%d healed over %d turns against %d over %d)",
						with.healedPerTurn(), without.healedPerTurn(),
						with.healed, with.turns, without.healed, without.turns)
				}
			}
		})
	}
}

// stripReading is what one kit did over those battles. Every column is a TOTAL:
// a squad battle runs a couple of dozen turns and a figure divided by the seed
// count truncates to nothing, which is the arithmetic that made the first Machop
// reading a row of noughts.
type stripReading struct {
	wins, losses   int
	strips, stacks int
	blocked        int
	healed         int64
	// turns is every turn taken across the run, and it is here because the
	// healing figure is meaningless without it. → healedPerTurn.
	turns int
}

// healedPerTurn is the regeneration figure, and it is a RATE because the total
// is not comparable between two arms.
//
// ⚠️ **The total was compared for a long time and it was reading battle length
// as often as it was reading the strip.** A regeneration gives back health per
// turn, so a run whose battles last half again as long heals half again as much
// with the same regeneration untouched — and the two arms here are two different
// kits, so their battles do not last the same. Measured after ENG-012 closed the
// board's mirror, over 150 seeds: the stripped arm ran 48,003 turns against the
// unstripped arm's 32,646, and its *total* healing came out higher while its
// healing per turn was a fifth lower. The old reading held only because the two
// lengths happened to be close before.
func (r stripReading) healedPerTurn() int64 {
	if r.turns == 0 {
		return 0
	}
	return r.healed / int64(r.turns)
}

func (r stripReading) rate() int {
	if r.wins+r.losses == 0 {
		return 0
	}
	return r.wins * 1000 / (r.wins + r.losses)
}

// aHexerSquad is a striker, a wall and a Gengar — the same shape the mender's
// reading uses, so what is isolated is the one slot rather than the squad. The
// wall's fourth skill is what the opposing squad varies.
func aHexerSquad(id string, kit, wall []string) placement.Squad {
	return placement.Squad{
		ID: id,
		Units: []placement.Placement{
			{ID: "fire", Character: "pokemon.charmander", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"flamethrower", "fire_spin", "ember", "inferno"},
				Passives: []string{"blaze"}},
			{ID: "wall", Character: "pokemon.squirtle", Level: progression.LevelCap,
				Slot: hex.Offset{Col: 1, Row: 1}, Skills: wall,
				Passives: []string{"endurance"}},
			{ID: "third", Character: "pokemon.gastly", Level: progression.LevelCap,
				Slot: hex.Offset{Col: 0, Row: 1}, Skills: kit,
				Passives: []string{"blood_thirst"}},
		},
	}
}

// readStrip fights one Gengar kit against a squad whose wall carries `wall`,
// both ways round, and reads what the strip did off the log.
//
// The home squad's own wall always shields, so the two squads are the same shape
// and only the far one varies — and every count below is filtered to the side
// under test, because both squads block and both heal.
func readStrip(t *testing.T, kit, wall []string) stripReading {
	t.Helper()
	return readStripOver(t, kit, wall, stripSeeds)
}

// readStripOver is readStrip with the depth named. → regenerationSeeds.
func readStripOver(t *testing.T, kit, wall []string, seeds int) stripReading {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	shielding := []string{"water_gun", "bubble", "bite", "withdraw"}
	home := aHexerSquad("home", kit, shielding)
	away := aHexerSquad("away",
		[]string{"shadow_claw", "shadow_ball", "night_shade", "bite"}, wall)

	var total stripReading
	for n := 1; n <= seeds; n++ {
		for _, swapped := range []bool{false, true} {
			first, second, mine, ours := home, away, hex.SideAlly, "ally."
			if swapped {
				first, second, mine, ours = away, home, hex.SideEnemy, "enemy."
			}
			ally, err := first.Take(hex.SideAlly, characters)
			if err != nil {
				t.Fatalf("field %s: %v", first.ID, err)
			}
			foe, err := second.Take(hex.SideEnemy, characters)
			if err != nil {
				t.Fatalf("field %s: %v", second.ID, err)
			}
			fought, err := battle.New(books, uint64(n), append(ally, foe...))
			if err != nil {
				t.Fatalf("seed %d: %v", n, err)
			}
			took, err := fought.RunToEnd(menderTurnLimit)
			if err != nil {
				t.Fatalf("seed %d: %v", n, err)
			}
			total.turns += took
			for _, event := range fought.Drain() {
				switch {
				case event.Kind == battle.StatusStripped && event.Skill == "spite" &&
					strings.HasPrefix(event.Actor, ours):
					total.strips++
					total.stacks += event.Stacks
				case event.Kind == battle.Blocked && strings.HasPrefix(event.Actor, ours):
					total.blocked++
				case event.Kind == battle.Healed && !strings.HasPrefix(event.Actor, ours) &&
					strings.HasSuffix(event.Actor, ".wall"):
					total.healed += event.Amount
				}
			}
			if !fought.Finished() {
				continue
			}
			winner, decided := fought.Winner()
			switch {
			case !decided:
			case winner == mine:
				total.wins++
			default:
				total.losses++
			}
		}
	}
	return total
}

// TestTheTwoGengarBuildsAreDifferentUnits, and the split is asserted where a duel
// can see it, which is only half of it.
//
// `miasma` lays poison down and cashes it in, and a duel prices that exactly:
// twice the damage and better than twice the statuses. `unbind` reads *worse* on
// both, and that figure is not a verdict on it — two of its four skills spend
// their turns on things a lone Charizard does not have, exactly as the mender's
// builds do. What prices `unbind` is the squad reading above.
func TestTheTwoGengarBuildsAreDifferentUnits(t *testing.T) {
	unbind := readBuild(t, "pokemon.gastly", progression.Furthest, unbindBuild, "elusive")
	miasma := readBuild(t, "pokemon.gastly", progression.Furthest, miasmaBuild, "contagion")
	for _, reading := range []struct {
		name string
		got  buildReading
	}{{"unbind", unbind}, {"miasma", miasma}} {
		t.Logf("%-7s %3d turns, dealt %5d, missed %3d, inflicted %3d",
			reading.name, reading.got.turns, reading.got.dealt,
			reading.got.missed, reading.got.inflicted)
	}
	// ⚠️ **The margin is the claim and a bare `>` is not.** Measured: two kits
	// made identical and told apart only by their traits still read 1300 against
	// 1178 and 153 against 81, because `contagion` weakens what it hits and a
	// weakened enemy takes longer to win. So the separation asked for is the one
	// the two kits actually produce — better than four fifths again as much
	// damage — and not the one two copies of a build would pass.
	if miasma.dealt*10 <= unbind.dealt*18 {
		t.Errorf("the miasma build dealt %d against the unbind build's %d: it is supposed to be the one that spends every turn on damage",
			miasma.dealt, unbind.dealt)
	}
	if miasma.inflicted <= unbind.inflicted {
		t.Errorf("the miasma build inflicted %d statuses against %d: laying things on is what it is for",
			miasma.inflicted, unbind.inflicted)
	}

	// And they are two builds rather than one twice: nothing is carried by both.
	for _, mine := range unbindBuild {
		for _, theirs := range miasmaBuild {
			if mine == theirs {
				t.Errorf("both builds carry %q, so the figures above are a reading of one kit against itself", mine)
			}
		}
	}
}

// TestTheSecondCarrierOfAnElementSharesItsLine is what a second carrier is for,
// and it is the reason Gengar needed four new skills rather than eight.
//
// `dark` arrived with Mewtwo and five skills. Gengar carries three of them
// unchanged — the element's line is the element's, not the character's, exactly
// as water's is shared by Squirtle and Poliwag — and brings four of its own that
// Mewtwo does not take. An element whose every skill belonged to one character
// would be a character with a private word for its kit.
func TestTheSecondCarrierOfAnElementSharesItsLine(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the skill book: %v", err)
	}
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	dark := map[string]bool{}
	for _, declared := range skills.Skills() {
		if declared.Element.String() == "dark" {
			dark[declared.ID] = true
		}
	}
	if len(dark) == 0 {
		t.Fatal("no skill declares the element, so there is no line to share")
	}
	carried := map[string][]string{}
	for _, character := range book.All() {
		for _, held := range character.Skills {
			if dark[held.ID] {
				carried[held.ID] = append(carried[held.ID], character.ID)
			}
		}
	}
	shared, alone := 0, 0
	for id, holders := range carried {
		if len(holders) > 1 {
			shared++
			continue
		}
		alone++
		t.Logf("%-14s only %v", id, holders)
	}
	if shared == 0 {
		t.Errorf("no dark skill is carried by more than one character, so the element is one character's private kit")
	}
	if alone == 0 {
		t.Errorf("every dark skill is carried by both, so neither carrier has anything of its own")
	}
	t.Logf("%d dark skills shared, %d held by one carrier", shared, alone)
}
