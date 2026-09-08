package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// menderSeeds is how many seeds each pairing is fought over, both ways round.
//
// Three hundred is enough to separate the figures this holds from the floor it
// holds them against and cheap enough to sit in the ordinary suite; the readings
// it was written against are 525 and 543 per mille against a floor of 450.
const (
	menderSeeds     = 300
	menderTurnLimit = 4000
)

// aSquadOf builds a three-slot squad out of two fixed carriers and a third
// member, which is the only thing that differs between the squads compared here.
//
// The two constants are a striker and a wall — the pair a third member is
// actually chosen beside — so what the comparison isolates is the slot rather
// than the squad.
func aSquadOf(id string, third placement.Placement) placement.Squad {
	return placement.Squad{
		ID: id,
		Units: []placement.Placement{
			{ID: "fire", Character: "pokemon.charmander", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"flamethrower", "fire_spin", "ember", "inferno"},
				Passives: []string{"blaze"}},
			{ID: "wall", Character: "pokemon.squirtle", Level: progression.LevelCap,
				Slot:     hex.Offset{Col: 1, Row: 1},
				Skills:   []string{"water_gun", "bubble", "bite", "withdraw"},
				Passives: []string{"endurance"}},
			third,
		},
	}
}

func aThirdMember(character string, kit ...string) placement.Placement {
	return aThirdMemberAs(character, progression.Furthest, kit...)
}

// aThirdMemberAs names the form, which a line that forks has to: Take resolves
// through the same refusal Resolve does, so a placement leaving the arm open is
// a squad that cannot be fielded at all.
func aThirdMemberAs(character, stage string, kit ...string) placement.Placement {
	return placement.Placement{
		ID: "third", Character: character, Level: progression.LevelCap, Stage: stage,
		Slot:     hex.Offset{Col: 0, Row: 1},
		Skills:   kit,
		Passives: []string{"endurance"},
	}
}

// TestAMenderEarnsItsSlotWhereASparCannotSeeIt is the only measurement in this
// repository that can price a support, and it exists because `hexforge spar`
// cannot.
//
// A duel is decided by who runs out of health first, and a mender's whole
// contribution is spent on a body that is not there — so Cleffa loses **every**
// shipped matchup in a spar, at 0 to 7 per mille, and that figure says nothing
// about whether the character is worth fielding. The same reading was already
// written down for Squirtle's tank build ("`hexforge spar` cannot measure either
// build"); this is the case where it is the character rather than one of its
// builds.
//
// So the question is asked the way it is actually decided: the same striker and
// the same wall in two squads, differing only in the third slot, fought both ways
// round over the same seeds. A mender that cannot hold that slot against a
// striker is a mender not worth authoring.
func TestAMenderEarnsItsSlotWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	mender := aSquadOf("with-mender", aThirdMember("pokemon.cleffa",
		"moonblast", "charm", "moonlight", "solar_beam"))

	// The floor is a long way under the readings on purpose. What is held is the
	// claim — a mend and a debuff are worth a slot a striker wants — and not the
	// figure, which is a reading of six characters at one level and moves with
	// every one of them.
	const floor = 450
	for _, against := range []struct {
		name  string
		squad placement.Squad
	}{
		{"a slugger", aSquadOf("with-slugger", aThirdMember("pokemon.machop",
			"rock_throw", "body_slam", "cross_chop", "vital_throw"))},
		{"a bruiser", aSquadOf("with-bruiser", aThirdMemberAs("pokemon.poliwag", "Poliwrath",
			"water_gun", "bubble", "pummel", "body_slam"))},
	} {
		t.Run(against.name, func(t *testing.T) {
			wins, losses, endless := fightSquads(t, books, characters, mender, against.squad)
			refuseTooManyStalls(t, against.name, endless, menderSeeds*2)
			decided := wins + losses
			if decided == 0 {
				t.Fatal("no battle was decided, so there is no rate to read")
			}
			rate := wins * 1000 / decided
			t.Logf("the mender's squad against %s: %d per mille (%d-%d)", against.name, rate, wins, losses)
			if rate < floor {
				t.Errorf("the mender's squad reads %d per mille against %s, under the floor of %d: "+
					"a slot a striker holds better is a slot the mender should not be in",
					rate, against.name, floor)
			}
		})
	}
}

// fightSquads runs one pairing over menderSeeds seeds from both slots and counts
// the home squad's record.
//
// Both ways round is the measurement rather than thoroughness: the turn queue
// breaks a tie by enlistment order, so one arrangement reports the first slot's
// advantage as the squad's. Both halves run the **same** seeds, because halves
// fought over different seeds cancel nothing.
// stallShare is how many battles of a thousand may reach the turn cap before a
// reading taken over the rest is refused, and it is a share rather than a count
// so the five fixtures that use it can run different numbers of seeds.
//
// ⚠️ **It used to be nought, and nought was a property of the seed window rather
// than of the engine.** Measured over 600 seeds both ways round on the cleanser
// fixture — 1200 battles against the 600 the test actually runs — the engine
// stalls at seed **307** whether or not the aim order is mirror-symmetric, and
// 307 is outside the 300 seeds this file fights. The bar was green because
// nobody had looked past it.
//
// What a stall is here is known and open: two survivors that cannot finish each
// other, a wall against a healer, healing that nearly matches damage over a
// thousand blows and fourteen hundred declined turns — `TODO.md` § *A declined
// turn makes a slow board slower*. It is not what any of these fixtures is
// measuring, and `Tally.Rate` already leaves an unresolved battle out of the
// denominator, so a handful is a stated cost rather than a silent one.
//
// ⚠️ The count is **logged whenever it is not nought**, because the thing this
// replaces was an assertion that said so loudly. A tolerance nobody can see the
// use of is a tolerance that grows.
const stallShare = 10

// refuseTooManyStalls is the one declaration of that rule, so five fixtures
// cannot drift into five different answers about what an unresolved battle costs
// a reading.
func refuseTooManyStalls(t *testing.T, subject string, endless, battles int) {
	t.Helper()
	if endless == 0 {
		return
	}
	t.Logf("%s: %d of %d battles never finished", subject, endless, battles)
	if endless*1000 > battles*stallShare {
		t.Errorf("%s: %d of %d battles never finished, past the %d per mille a reading "+
			"may carry: the rest are a reading of the ones that did",
			subject, endless, battles, stallShare)
	}
}

// ⚠️ **Composition bonuses are switched off for every measurement that goes
// through here, and that is the control rather than a convenience.** Each of
// these fixtures prices ONE slot or ONE skill by holding the other two members
// constant across both squads — and a shared element is not held constant by
// that arrangement: `aSquadOf` fields a fire unit and a water one, so a third
// member that happens to be water earns its squad a rung the other squad has no
// way to match, and the reading stops being about the slot. Measured, the mender
// against a Poliwrath bruiser: **413‰ with the bonus live, 698‰ without** — the
// live figure is under this file's own floor of 450, so the confound reads as
// "a mender is not worth the slot" while what it actually says is that the
// bruiser's squad shares an element and the mender's does not.
//
// A bonus is a real part of the game and pricing one is a different measurement —
// same squad, same seeds, the bonus toggled, which is what `composition.Book.Without`
// and `forge.FightSquads`'s last argument are for. What may not happen is a slot
// measurement quietly reporting a squad-composition effect under a mender's name.
func fightSquads(t *testing.T, books battle.Books, characters *cast.Book,
	home, away placement.Squad) (wins, losses, endless int) {
	t.Helper()
	return fightSquadsOver(t, books, characters, home, away, menderSeeds)
}

// fightSquadsOver is fightSquads with the depth named, for a reading whose
// claim is a MARGIN rather than a level.
//
// ⚠️ **A margin needs more battles than a level does, and menderSeeds is not
// enough for one.** A rate read off six hundred battles moves tens of parts per
// thousand between instruments; a claim of the form "this is worth at least
// thirty" is then reading the noise as often as the effect. Measured on the
// bombardier's shape after ENG-012 closed the mirror: +7 over 300 seeds, +42
// over 1200 and +41 over 3000 — the same claim, true, and invisible at the depth
// the rest of this file uses. → TestAShapeEarnsItsPowerWhereASparCannotSeeIt.
func fightSquadsOver(t *testing.T, books battle.Books, characters *cast.Book,
	home, away placement.Squad, seeds int) (wins, losses, endless int) {
	t.Helper()
	books.Bonuses = nil
	for n := 1; n <= seeds; n++ {
		for _, swapped := range []bool{false, true} {
			first, second := home, away
			mine := hex.SideAlly
			if swapped {
				first, second = away, home
				mine = hex.SideEnemy
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
			if _, err := fought.RunToEnd(menderTurnLimit); err != nil {
				t.Fatalf("seed %d: %v", n, err)
			}
			if !fought.Finished() {
				endless++
				continue
			}
			winner, decided := fought.Winner()
			switch {
			case !decided:
			case winner == mine:
				wins++
			default:
				losses++
			}
		}
	}
	return wins, losses, endless
}
