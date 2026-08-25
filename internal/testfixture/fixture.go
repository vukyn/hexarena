// Package testfixture injects a known origin catalogue and cast into a scratch
// data directory.
//
// It exists because the tests used to name the characters the repository
// shipped, which coupled them to content the author is free to change: editing
// the real cast broke thirty-two tests that had nothing to do with it. A test
// that needs "a character" now gets this one, and a test that needs "whatever is
// shipped" asks the book instead of naming a row.
//
// It is a normal package rather than a _test.go file because three packages need
// the same fixture, and a second copy of it is a second thing to keep in step.
// Nothing outside a test imports it.
package testfixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vukyn/hexarena/internal/core/cast"
)

// Origins is the catalogue the fixture adds.
const Origins = `[
  {
    "id": "fixture-anime",
    "title": "Example Anime",
    "medium": "anime",
    "year": 2002,
    "note": "A test fixture, not shipped."
  },
  {
    "id": "fixture-film",
    "title": "Example Film",
    "medium": "film",
    "year": 1999,
    "note": "A test fixture, not shipped."
  },
  {
    "id": "fixture-game",
    "title": "Example Game",
    "medium": "game",
    "year": 2011,
    "note": "A test fixture, not shipped."
  }
]`

// Characters is the cast the fixture adds.
const Characters = `[
  {
    "id": "fixture-anime.adept",
    "name": "Example Adept",
    "origin": "fixture-anime",
    "archetype": "sentinel",
    "image": "assets/fixture/adept.svg",
    "element": [
      "water",
      "ice"
    ],
    "bio": "A test fixture, injected into a scratch data directory so a test does not depend on whatever cast the repository happens to hold.",
    "stages": [
      {
        "name": "Example Adept",
        "min_level": 1,
        "stats": {
          "hp": {
            "base": 930,
            "max": 3100
          },
          "attack": {
            "base": 150,
            "max": 500
          },
          "defense": {
            "base": 240,
            "max": 800
          },
          "speed": {
            "base": 27,
            "max": 90
          },
          "accuracy": {
            "base": 24,
            "max": 80
          },
          "dodge": {
            "base": 9,
            "max": 30
          }
        }
      }
    ],
    "skills": [
      "strike",
      "riptide",
      "guard_wall",
      "purify"
    ]
  },
  {
    "id": "fixture-game.sprout",
    "name": "Example Sprout",
    "origin": "fixture-game",
    "archetype": "skirmisher",
    "image": "assets/fixture/sprout.svg",
    "element": [
      "grass",
      "electric"
    ],
    "bio": "A test fixture, injected into a scratch data directory so a test does not depend on whatever cast the repository happens to hold.",
    "stages": [
      {
        "name": "Sprout",
        "min_level": 1,
        "stats": {
          "hp": {
            "base": 660,
            "max": 1500
          },
          "attack": {
            "base": 228,
            "max": 520
          },
          "defense": {
            "base": 75,
            "max": 170
          },
          "speed": {
            "base": 60,
            "max": 140
          },
          "accuracy": {
            "base": 72,
            "max": 165
          },
          "dodge": {
            "base": 45,
            "max": 100
          }
        }
      },
      {
        "name": "Bloom",
        "min_level": 30,
        "stats": {
          "hp": {
            "base": 1100,
            "max": 2200
          },
          "attack": {
            "base": 380,
            "max": 760
          },
          "defense": {
            "base": 125,
            "max": 250
          },
          "speed": {
            "base": 100,
            "max": 200
          },
          "accuracy": {
            "base": 120,
            "max": 240
          },
          "dodge": {
            "base": 75,
            "max": 150
          }
        }
      }
    ],
    "skills": [
      "bolt",
      "venom_fang",
      "creeping_rot",
      "arc_bolt"
    ]
  }
]`

// Art is a placeholder drawing, small enough to be legible in a diff. The
// authoring tool only asks whether the file exists.
const Art = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" fill="#4a5"/></svg>`

// Saver is the part of the authoring library this package needs.
//
// It is an interface rather than the library itself because internal/forge's own
// tests use this fixture, and importing forge here would be a cycle. Go's
// interfaces are structural, so *forge.Library satisfies this without either
// package knowing about the other.
type Saver interface {
	SaveOrigin(cast.Origin) error
	SaveCharacter(cast.Character) error
}

// Inject adds the fixture to a data directory that already holds the shipped
// books, and writes the art the fixture characters name.
//
// reload hands back a library reading that directory, and is called again before
// every write: each save rewrites a whole book, so a library opened once would
// hold a stale copy of the file it is about to replace.
//
// It writes through the library rather than editing the JSON for two reasons.
// The files stay in exactly the form Marshal produces, which a test asserts
// about the shipped cast, and the fixture is validated the way real data is -- a
// fixture the tool would refuse is not one worth testing with.
func Inject(dir string, reload func() (Saver, error)) error {
	art := filepath.Join(dir, "assets", "fixture")
	if err := os.MkdirAll(art, 0o755); err != nil {
		return fmt.Errorf("make %s: %w", art, err)
	}
	for _, name := range []string{"adept.svg", "sprout.svg"} {
		if err := os.WriteFile(filepath.Join(art, name), []byte(Art), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	var origins []cast.Origin
	if err := json.Unmarshal([]byte(Origins), &origins); err != nil {
		return fmt.Errorf("decode the fixture origins: %w", err)
	}
	for _, origin := range origins {
		library, err := reload()
		if err != nil {
			return fmt.Errorf("reload before %s: %w", origin.ID, err)
		}
		if err := library.SaveOrigin(origin); err != nil {
			return fmt.Errorf("save %s: %w", origin.ID, err)
		}
	}

	var characters []cast.Character
	if err := json.Unmarshal([]byte(Characters), &characters); err != nil {
		return fmt.Errorf("decode the fixture characters: %w", err)
	}
	for _, character := range characters {
		library, err := reload()
		if err != nil {
			return fmt.Errorf("reload before %s: %w", character.ID, err)
		}
		if err := library.SaveCharacter(character); err != nil {
			return fmt.Errorf("save %s: %w", character.ID, err)
		}
	}
	return nil
}
