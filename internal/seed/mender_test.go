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
// it holds today are 826 and 725 per mille against a floor of 450.
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
//
// ⚠️ **The trait is fixed at `endurance` for every third member this builds**,
// including the opponents. That is a constant rather than a choice, and it is
// not free: swept over the cleanser's six traits, `endurance` is its best and
// `ballast` reads **21 per mille** against a slugger — seventeen wins in eight
// hundred battles, because `encumber` lands on the slowest unit on the board.
// A fixture that fixes a slot cannot report what that slot is worth, which is
// why aThirdMemberFrom exists beside it. → `TODO.md` `DAT-011`.
func aThirdMemberAs(character, stage string, kit ...string) placement.Placement {
	return placement.Placement{
		ID: "third", Character: character, Level: progression.LevelCap, Stage: stage,
		Slot:     hex.Offset{Col: 0, Row: 1},
		Skills:   kit,
		Passives: []string{"endurance"},
	}
}

// aThirdMemberFrom fields a CATALOGUED build in the third slot — its four skills
// and its trait, both read off `builds.json` rather than written out here.
//
// ⚠️ **It exists because a hand-written kit is not a reading of the character,
// and DAT-011 is what proved that.** The mender's fixture used to field a kit
// taking moonblast and moonlight from `cleffa.mend` and charm and solar_beam
// from `cleffa.hex`: it read 601 per mille against a slugger while the two
// builds the catalogue actually carries read 356 (`cleffa.hex`, before DAT-013
// rekitted it) and an unquotable figure (`cleffa.mend`, 85 of 600 past the turn
// cap) against the same opponent. So a floor cleared by the hybrid said nothing
// about a floor a shipped build clears, and a support measured on its own only
// kit was being held to a line the comparison never had to meet. Both fixtures
// in this package now go through here.
func aThirdMemberFrom(t *testing.T, id string) placement.Placement {
	t.Helper()
	catalogue, err := seed.Builds()
	if err != nil {
		t.Fatalf("load the build catalogue: %v", err)
	}
	built, ok := catalogue.Get(id)
	if !ok {
		t.Fatalf("no build is called %q, so there is nothing to field", id)
	}
	return placement.Placement{
		ID: "third", Character: built.Character, Level: progression.LevelCap,
		Stage:    built.Stage,
		Slot:     hex.Offset{Col: 0, Row: 1},
		Skills:   built.Skills,
		Passives: built.Passives,
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
//
// ⚠️ **It used to fight a hybrid no player could pick, and DAT-013 is what
// replaced it.** The kit here was `moonblast, charm, moonlight, solar_beam` —
// two skills from each of Cleffa's two builds, which `builds.json` does not
// carry — so the test was green on a kit that cleared the floor while neither
// shipped build did. It now fields `cleffa.hex` through aThirdMemberFrom, like
// the cleanser beside it, and the build was rekitted to earn the slot.
//
// **Why the rekit, and why it is not a repricing.** `cleffa.hex` read 356 / 226
// / 281 per mille here against a slugger, a bruiser and a blighter — a shipped
// direction losing three battles in four. The question DAT-013 said had to be
// settled first is whether the BUILD was weak or the RATING underprices control,
// and the census on this board settles it: over 600 battles a column, every one
// of the four slots fires in the thousands and control is 76.0%, 75.1% and 75.5%
// of all casts, with `charm` the most-cast skill on all three boards. An
// under-price shows up as SILENCE; this is the opposite of silence, so it is the
// kit. (⚠️ The item cited `RAT-002` for "the rating underprices control" and
// `RAT-002` does not say that — it is about `forge.Bout` fighting two ratings
// head to head. The nearest true statement is the general one at `price.go:12-34`,
// which binds buffs, guards and heals equally and is not a control finding.)
//
// **The sweep, 300 seeds a side, one slot given up for `moonblast` — the only
// other damaging skill in the learnset. Every row read 0 endless of 600:**
//
//	kit                                slugger   bruiser   blighter
//	charm sing smokescreen solar_beam      356       226        281   (was shipped)
//	moonblast sing smokescreen solar        616       610        373
//	charm moonblast smokescreen solar       696       563        400
//	charm sing MOONBLAST solar_beam         826       725        546   <- shipped now
//	charm sing smokescreen moonblast        251       171        260
//
// and no non-damaging alternative in that same slot clears 400 against all
// three: `taunt` 606/395/606, `wide_guard` 730/616/278, `light_screen`
// 666/780/386, `rapid_spin` 341/220/300. So the direction stops being pure
// control on purpose — a LEAN rather than a purity, which is what
// `bulbasaur.parasite` and `squirtle.fortress` already are.
//
// ⚠️ **The 826 and 725 are IN-SAMPLE**, since this shell is the one the sweep
// chose on. Out of sample, in a second shell whose partner is a Machop rather
// than the Squirtle wall, the same swap reads +110, +152 and +96 — smaller, same
// sign on all three, which is the confirmation `DAT-012` asks for.
//
// ⚠️ **`cleffa.mend` still cannot be read on this shell at all**, and that is
// not a fact about the build: `aSquadOf` puts the same `withdraw`-carrying
// Squirtle in BOTH squads, so every reading here is a heal mirror, which
// `RAT-003` measured at 100% endless in isolation. Put a second healer in the
// home seat and it shows — re-taken, `cleffa.mend` puts **85 of 600** battles
// past the turn cap against a slugger and 19 of 600 against a bruiser, fourteen
// and three times the ten per mille a reading here may carry, so both rows are
// dropped rather than quoted. → `TODO.md` `DAT-014`.
func TestAMenderEarnsItsSlotWhereASparCannotSeeIt(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	third := aThirdMemberFrom(t, "cleffa.hex")
	mender := aSquadOf("with-mender", third)

	// The control first: a shell that is not even cannot price anything put in
	// it, and this is the one arrangement whose answer is known in advance.
	wins, losses, endless := fightSquads(t, books, characters,
		aSquadOf("mirror", third), mender)
	refuseTooManyStalls(t, "the mirror", endless, menderSeeds*2)
	if decided := wins + losses; decided == 0 || wins*1000/decided != 500 {
		t.Fatalf("the mender's squad against a copy of itself reads %d-%d: a shell that is not "+
			"even reports its own bias as whatever is put in it", wins, losses)
	}

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
