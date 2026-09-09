//go:build dat002probe

// Package forge's DAT-002 probe harness.
//
// It is behind a build tag on purpose. The measurement is 800 battles a cell at
// 5v5 and `make check` may not pay for it, but ENG-013 was burned by a harness
// that was never committed and whose fixture was recorded nowhere — so the
// fixture lives here, in the repository, where the next reader can re-run it
// rather than rebuild it from a paragraph. Run it with:
//
//	go test ./internal/forge -tags dat002probe -run TestDAT002ReadingA -v -count=1
//
// ⚠️ The probe squads are NOT in internal/seed/data/squads.json and must not be.
// Shipping one would change the "no shipped formation reaches any rung" claim
// and silently answer DAT-010. They are written into a scratch data directory
// per run and thrown away with it.
package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// probeSeeds is the run length a cell is measured over. Both arrangements are
// fought, so a cell counts twice this many battles.
const probeSeeds = 400

// The two books. SHIPPED is internal/seed/data unmodified; CANDIDATE raises
// tidewell's cap from two stacks to three and gives water_tide a rung at four
// granting the third.
const (
	shippedCapText = `      "id": "tidewell",
      "category": "heal_mend",
      "max_stacks": 2,`
	candidateCapText = `      "id": "tidewell",
      "category": "heal_mend",
      "max_stacks": 3,`

	shippedRungText = `        {
          "at": 3,
          "grants": [
            {
              "status": "tidewell",
              "stacks": 2
            }
          ]
        }
      ]
    },
    {
      "id": "grass_growth",`
	candidateRungText = `        {
          "at": 3,
          "grants": [
            {
              "status": "tidewell",
              "stacks": 2
            }
          ]
        },
        {
          "at": 4,
          "grants": [
            {
              "status": "tidewell",
              "stacks": 3
            }
          ]
        }
      ]
    },
    {
      "id": "grass_growth",`
)

// probeWater is the rung-4 subject: four distinct water carriers and one inert
// fifth, spread at most two to a column so that same_column never fires.
//
// ⚠️ Every member's kit is written out here rather than named, because a build
// id is a pointer into a file that moves. Where a builds.json entry exists it
// was copied verbatim and is named in the comment; lapras has no build at all.
func probeWater() placement.Squad {
	return placement.Squad{
		ID:   "probe_water",
		Name: "probe: four water carriers",
		Units: []placement.Placement{
			{
				// ⚠️ Hand-built, and the deviation is a measurement fact rather
				// than a taste. This is the squad's main tidewell subject —
				// withdraw restores 500 and aqua_ring holds two stacks of
				// regrowth, and both take the heal share — so the obvious kit is
				// builds.json squirtle.fortress (taunt, withdraw, wide_guard,
				// aqua_ring). That kit carries NO damaging skill, so two of them
				// taunt and heal past spar's turn limit: the probe_water MIRROR
				// came back 10 of 10 Endless, and Rate() drops Endless from the
				// denominator, so the control reads 0‰ and cannot be read at all.
				// Keeping both heals and trading taunt and wide_guard for the
				// line's two heaviest water skills is what makes the mirror
				// resolve, which gate 1 needs before anything else is quotable.
				ID: "blastoise", Character: "pokemon.squirtle", Level: progression.LevelCap,
				Stage: "Blastoise", Slot: hex.Offset{Col: 0, Row: 0},
				Skills:   []string{"withdraw", "aqua_ring", "skull_bash", "bite"},
				Passives: []string{"thorns"},
			},
			{
				// builds.json poliwag.flurry, verbatim. Poliwrath rather than
				// Politoed: the line forks at 32, so a form has to be named, and
				// this is the arm that build is written for. flurry rather than
				// riptide because blood_thirst drains 250 of the damage it
				// deals, which takes the heal share and makes this seat a third
				// tidewell subject rather than only a recipient.
				ID: "poliwrath", Character: "pokemon.poliwag", Level: progression.LevelCap,
				Stage: "Poliwrath", Slot: hex.Offset{Col: 0, Row: 1},
				Skills:   []string{"pummel", "body_slam", "submission", "water_gun"},
				Passives: []string{"blood_thirst"},
			},
			{
				// Hand-built: lapras has no entry in builds.json. withdraw is
				// the only heal it learns and is what makes it a second tidewell
				// subject; the other three are its heaviest ice damage.
				ID: "lapras", Character: "pokemon.lapras", Level: progression.LevelCap,
				Stage: "Lapras", Slot: hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"withdraw", "ice_beam", "blizzard", "body_slam"},
				Passives: []string{"endurance"},
			},
			{
				// builds.json magikarp.gale, verbatim — the wind arm rather than
				// magikarp.surge's water one.
				ID: "gyarados", Character: "pokemon.magikarp", Level: progression.LevelCap,
				Stage: "Gyarados", Slot: hex.Offset{Col: 1, Row: 1},
				Skills:   []string{"hurricane", "air_slash", "gust", "body_slam"},
				Passives: []string{"contagion"},
			},
			{
				// builds.json mew.borrowed, verbatim. Neutral is inert, so this
				// unit forms no tribe of its own and receives no tidewell —
				// water_tide is scope "sharers". It is here to make the side
				// five without adding a second element count.
				ID: "mew", Character: "pokemon.mew", Level: progression.LevelCap,
				Stage: "Mew", Slot: hex.Offset{Col: 2, Row: 1},
				Skills:   []string{"cross_chop", "submission", "vital_throw", "body_slam"},
				Passives: []string{"endurance"},
			},
		},
	}
}

// probeThreeWater is the null control: probeWater with the Gyarados seat taken
// by a fire carrier, so the side reaches water rung 3 and no rung 4 exists for
// it to reach. Its reading must be identical under both books.
func probeThreeWater() placement.Squad {
	squad := probeWater()
	squad.ID = "probe_three_water"
	squad.Name = "probe: three water carriers"
	squad.Units[3] = placement.Placement{
		// builds.json charmander.scorch, verbatim, in the Gyarados seat.
		ID: "charizard", Character: "pokemon.charmander", Level: progression.LevelCap,
		Stage: "Charizard", Slot: hex.Offset{Col: 1, Row: 1},
		Skills:   []string{"flamethrower", "inferno", "ember", "fire_spin"},
		Passives: []string{"blaze"},
	}
	return squad
}

// probeSpread is the opponent and fires nothing: five different elements, no
// column holding three.
func probeSpread() placement.Squad {
	return placement.Squad{
		ID:   "probe_spread",
		Name: "probe: one carrier of each of five elements",
		Units: []placement.Placement{
			{
				// builds.json charmander.scorch, verbatim. fire.
				ID: "charizard", Character: "pokemon.charmander", Level: progression.LevelCap,
				Stage: "Charizard", Slot: hex.Offset{Col: 0, Row: 0},
				Skills:   []string{"flamethrower", "inferno", "ember", "fire_spin"},
				Passives: []string{"blaze"},
			},
			{
				// builds.json bulbasaur.poison, verbatim. grass.
				ID: "venusaur", Character: "pokemon.bulbasaur", Level: progression.LevelCap,
				Stage: "Venusaur", Slot: hex.Offset{Col: 0, Row: 1},
				Skills:   []string{"poison_powder", "sludge_bomb", "venoshock", "razor_leaf"},
				Passives: []string{"virulence"},
			},
			{
				// builds.json pichu.burn, verbatim. electric.
				ID: "raichu", Character: "pokemon.pichu", Level: progression.LevelCap,
				Stage: "Raichu", Slot: hex.Offset{Col: 1, Row: 0},
				Skills:   []string{"volt_tackle", "thunderbolt", "nuzzle", "thunder_shock"},
				Passives: []string{"static"},
			},
			{
				// Hand-built: riolu has no entry in builds.json. metal.
				ID: "lucario", Character: "pokemon.riolu", Level: progression.LevelCap,
				Stage: "Lucario", Slot: hex.Offset{Col: 1, Row: 1},
				Skills:   []string{"aura_sphere", "close_combat", "flash_cannon", "metal_claw"},
				Passives: []string{"berserk"},
			},
			{
				// builds.json abra.shade, verbatim. dark.
				ID: "alakazam", Character: "pokemon.abra", Level: progression.LevelCap,
				Stage: "Alakazam", Slot: hex.Offset{Col: 2, Row: 1},
				Skills:   []string{"shadow_ball", "dark_pulse", "night_shade", "psycho_cut"},
				Passives: []string{"swiftness"},
			},
		},
	}
}

// probeBook is which of the two data directories a run stands in.
type probeBook int

const (
	shippedBook probeBook = iota
	candidateBook
)

func (b probeBook) String() string {
	if b == candidateBook {
		return "CANDIDATE"
	}
	return "SHIPPED"
}

// probeLibrary builds a scratch data directory, patches the two books when the
// candidate is wanted, and saves the three probe squads into it.
func probeLibrary(t *testing.T, book probeBook) *Library {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, shippedDataDir, dir)
	if book == candidateBook {
		patch(t, filepath.Join(dir, "statuses.json"), shippedCapText, candidateCapText)
		patch(t, filepath.Join(dir, "bonuses.json"), shippedRungText, candidateRungText)
	}
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load the %s books: %v", book, err)
	}
	for _, squad := range []placement.Squad{probeWater(), probeThreeWater(), probeSpread()} {
		if err := lib.SaveSquad(squad); err != nil {
			t.Fatalf("save %s into the %s directory: %v", squad.ID, book, err)
		}
	}
	return lib
}

// patch rewrites one exact run of text in a scratch data file, and fails when
// the text it was told to replace is not there — a silent no-op patch is a
// candidate book identical to the shipped one, which would make every reading
// below a null for a reason nothing reports.
func patch(t *testing.T, path, from, to string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if count := strings.Count(string(raw), from); count != 1 {
		t.Fatalf("%s holds the text to patch %d times, want exactly once", path, count)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), from, to, 1)), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// cell fights one pairing under one book and prints every field the gate reads.
func cell(t *testing.T, label string, book probeBook, lib *Library, home, away string, without ...string) SquadReport {
	t.Helper()
	report, err := lib.FightSquads(home, away, probeSeeds, without...)
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
	total := report.Total()
	line := fmt.Sprintf(
		"CELL %-26s %-9s %-18s vs %-18s without=%v mirror=%t rate=%d‰ "+
			"ally=%d‰(W%d/L%d/D%d/E%d) enemy=%d‰(W%d/L%d/D%d/E%d) endless=%d/%d turns=%d",
		label, book, home, away, without, report.Mirror(), report.Rate(),
		report.AsAlly.Rate(), report.AsAlly.Wins, report.AsAlly.Losses, report.AsAlly.Draws, report.AsAlly.Endless,
		report.AsEnemy.Rate(), report.AsEnemy.Wins, report.AsEnemy.Losses, report.AsEnemy.Draws, report.AsEnemy.Endless,
		total.Endless, total.Battles(), report.Turns)
	fmt.Println(line)
	t.Log(line)
	return report
}

// same reports whether two reports agree in every field the gate names.
func same(left, right SquadReport) bool {
	return left.Rate() == right.Rate() &&
		left.AsAlly == right.AsAlly &&
		left.AsEnemy == right.AsEnemy &&
		left.Turns == right.Turns
}

// TestDAT002Smoke is the instrument check, not a reading: it fights one battle
// under each book and reports which bonuses took hold and how much healing the
// board saw. A rung that does not fire, or a squad that never heals, makes every
// figure below a null for a reason the rate cannot say.
func TestDAT002Smoke(t *testing.T) {
	for _, book := range []probeBook{shippedBook, candidateBook} {
		lib := probeLibrary(t, book)
		for _, home := range []string{"probe_water", "probe_three_water"} {
			squad, err := lib.squad(home)
			if err != nil {
				t.Fatalf("%s: %v", home, err)
			}
			facing, err := lib.squad("probe_spread")
			if err != nil {
				t.Fatalf("probe_spread: %v", err)
			}
			roster, err := squad.Take(hex.SideAlly, lib.characters)
			if err != nil {
				t.Fatalf("take %s: %v", home, err)
			}
			enemy, err := facing.Take(hex.SideEnemy, lib.characters)
			if err != nil {
				t.Fatalf("take probe_spread: %v", err)
			}
			result, turns, events, err := fight(lib.Books(), append(roster, enemy...), hex.SideAlly, 1)
			if err != nil {
				t.Fatalf("fight: %v", err)
			}
			var heals, healed int64
			for _, event := range events {
				switch event.Kind {
				case battle.BonusHeld:
					fmt.Printf("SMOKE %-9s %-18s bonus_held %s shared=%s count=%d -> %s x%d on %s\n",
						book, home, event.Bonus, event.Shared, event.Count, event.Status, event.Stacks, event.Target)
				case battle.Healed:
					heals++
					healed += event.Amount
				}
			}
			fmt.Printf("SMOKE %-9s %-18s seed=1 result=%v turns=%d heals=%d healthRestored=%d events=%d\n",
				book, home, result, turns, heals, healed, len(events))
		}
	}
}

// TestDAT002ReadingA is the control: a squad against a copy of itself must read
// exactly 500‰ under both books, or nothing below is quotable.
func TestDAT002ReadingA(t *testing.T) {
	for _, book := range []probeBook{candidateBook, shippedBook} {
		lib := probeLibrary(t, book)
		for _, squad := range []string{"probe_water", "probe_three_water", "probe_spread"} {
			report := cell(t, "A mirror "+squad, book, lib, squad, squad)
			if report.Rate() != 500 {
				t.Errorf("A: %s mirror under %s reads %d‰, want exactly 500‰", squad, book, report.Rate())
			}
		}
	}
}

// TestDAT002ReadingD is the null control: a side that reaches only rung 3 must
// read identically under the two books, field for field.
func TestDAT002ReadingD(t *testing.T) {
	shipped := cell(t, "D three_water", shippedBook, probeLibrary(t, shippedBook), "probe_three_water", "probe_spread")
	candidate := cell(t, "D three_water", candidateBook, probeLibrary(t, candidateBook), "probe_three_water", "probe_spread")
	if !same(shipped, candidate) {
		t.Errorf("D: the two books disagree on a side that reaches no rung 4:\nSHIPPED   %+v\nCANDIDATE %+v", shipped, candidate)
	}
}

// TestDAT002ReadingB prices the WHOLE bonus at rung 4: the candidate book with
// water_tide against the candidate book without it.
func TestDAT002ReadingB(t *testing.T) {
	lib := probeLibrary(t, candidateBook)
	with := cell(t, "B with water_tide", candidateBook, lib, "probe_water", "probe_spread")
	off := cell(t, "B without water_tide", candidateBook, probeLibrary(t, candidateBook), "probe_water", "probe_spread", "water_tide")
	fmt.Printf("READING B: the whole bonus at rung 4 is worth %+d‰ (%d‰ with, %d‰ without)\n",
		with.Rate()-off.Rate(), with.Rate(), off.Rate())
}

// TestDAT002ReadingC is the deliverable: the same squad, the same members and
// the same seeds, with only the BOOK moving. Book.Without removes a whole bonus
// rather than a rung, so this is the only way the rung's own increment is
// isolated.
func TestDAT002ReadingC(t *testing.T) {
	shipped := cell(t, "C water rung3", shippedBook, probeLibrary(t, shippedBook), "probe_water", "probe_spread")
	candidate := cell(t, "C water rung4", candidateBook, probeLibrary(t, candidateBook), "probe_water", "probe_spread")
	fmt.Printf("READING C: the rung at 4 is worth %+d‰ (%d‰ under CANDIDATE, %d‰ under SHIPPED); "+
		"endless candidate=%d shipped=%d of %d\n",
		candidate.Rate()-shipped.Rate(), candidate.Rate(), shipped.Rate(),
		candidate.Total().Endless, shipped.Total().Endless, shipped.Total().Battles())
	if same(shipped, candidate) {
		fmt.Println("READING C: NULL — the two books are identical in every field on this board")
	}
}

// vacuousRungText gives water_tide a rung at four granting the stacks rung
// three already grants. It parses, loads, draws and FIRES, and changes nothing —
// which is decision 6's vacuous row one layer down, and is exactly what the
// Step 3 parse guard refuses when the stacks exceed the cap.
const vacuousRungText = `        {
          "at": 3,
          "grants": [
            {
              "status": "tidewell",
              "stacks": 2
            }
          ]
        },
        {
          "at": 4,
          "grants": [
            {
              "status": "tidewell",
              "stacks": 2
            }
          ]
        }
      ]
    },
    {
      "id": "grass_growth",`

// TestDAT002MutationTheRungIsTheStack is mutation check 2 of the plan, run
// against the candidate book rather than a shipped one because the gate failed
// and the rung was never authored: set the rung at four to the stacks rung three
// already grants and reading C must collapse onto the shipped signature. If it
// does not, the figure reading C reports is the ROW rather than the third stack,
// and it is an artefact of the extra bonus_held rather than a measurement of the
// grant.
func TestDAT002MutationTheRungIsTheStack(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, shippedDataDir, dir)
	patch(t, filepath.Join(dir, "statuses.json"), shippedCapText, candidateCapText)
	patch(t, filepath.Join(dir, "bonuses.json"), shippedRungText, vacuousRungText)
	lib, err := Load(dir)
	if err != nil {
		t.Fatalf("load the vacuous book: %v", err)
	}
	for _, squad := range []placement.Squad{probeWater(), probeThreeWater(), probeSpread()} {
		if err := lib.SaveSquad(squad); err != nil {
			t.Fatalf("save %s: %v", squad.ID, err)
		}
	}
	vacuous := cell(t, "MUT2 rung4 stacks=2", candidateBook, lib, "probe_water", "probe_spread")
	shipped := cell(t, "MUT2 shipped rung3", shippedBook, probeLibrary(t, shippedBook), "probe_water", "probe_spread")
	if same(vacuous, shipped) {
		fmt.Println("MUTATION 2: PASS - a rung at 4 granting two stacks collapses onto the shipped signature, so reading C measures the THIRD STACK and not the row")
	} else {
		fmt.Printf("MUTATION 2: FAIL - the vacuous rung reads %d\u2030 against the shipped %d\u2030, so the row itself moves the figure\n",
			vacuous.Rate(), shipped.Rate())
	}
}
