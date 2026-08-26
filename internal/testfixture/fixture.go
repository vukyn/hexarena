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
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Skills is the bench the authoring tests exercise: nineteen skills covering
// every element, every shape, both kinds of condition, multi-strike, area
// splash, cleansing and a shield.
//
// They used to be the shipped skill book, and the tests named them directly.
// That coupled a hundred assertions to content the author is free to delete --
// which is exactly what happened. They live here now, so what the game ships and
// what the tests need are two different lists.
const Skills = `[
  {
    "id": "strike",
    "element": "neutral",
    "range": 1,
    "pattern": "single",
    "power": 1000,
    "strikes": 1,
    "accuracy": 960,
    "cooldown": 0,
    "target": "enemy"
  },
  {
    "id": "bolt",
    "element": "neutral",
    "range": 3,
    "pattern": "single",
    "power": 800,
    "strikes": 1,
    "accuracy": 940,
    "cooldown": 0,
    "target": "enemy"
  },
  {
    "id": "ember_lance",
    "element": "fire",
    "range": 2,
    "pattern": "single",
    "power": 1800,
    "strikes": 1,
    "accuracy": 900,
    "cooldown": 2,
    "target": "enemy",
    "applies": [
      {
        "status": "burn",
        "chance": 700,
        "stacks": 1
      }
    ]
  },
  {
    "id": "venom_fang",
    "element": "grass",
    "range": 1,
    "pattern": "single",
    "power": 1200,
    "strikes": 1,
    "accuracy": 950,
    "cooldown": 1,
    "target": "enemy",
    "applies": [
      {
        "status": "poison",
        "chance": 900,
        "stacks": 1
      }
    ]
  },
  {
    "id": "creeping_rot",
    "element": "grass",
    "range": 3,
    "pattern": "column",
    "power": 700,
    "strikes": 1,
    "accuracy": 880,
    "cooldown": 3,
    "target": "enemy",
    "applies": [
      {
        "status": "poison",
        "chance": 650,
        "stacks": 1
      }
    ]
  },
  {
    "id": "cinder_burst",
    "element": "fire",
    "range": 2,
    "pattern": "wedge_right",
    "power": 1000,
    "strikes": 1,
    "accuracy": 850,
    "cooldown": 3,
    "target": "enemy",
    "requires": {
      "status": "poison",
      "min_stacks": 1,
      "bonus_power": 500
    }
  },
  {
    "id": "detonate",
    "element": "fire",
    "range": 2,
    "pattern": "single",
    "power": 900,
    "strikes": 1,
    "accuracy": 1000,
    "cooldown": 4,
    "target": "enemy",
    "requires": {
      "status": "burn",
      "min_stacks": 1,
      "bonus_power": 2200,
      "consume": true
    }
  },
  {
    "id": "flurry",
    "element": "neutral",
    "range": 1,
    "pattern": "single",
    "power": 600,
    "strikes": 3,
    "accuracy": 820,
    "cooldown": 2,
    "target": "enemy"
  },
  {
    "id": "gale_slash",
    "element": "wind",
    "range": 2,
    "pattern": "arc_up",
    "power": 1260,
    "strikes": 1,
    "accuracy": 880,
    "cooldown": 3,
    "target": "enemy"
  },
  {
    "id": "sever",
    "element": "metal",
    "range": 1,
    "pattern": "single",
    "power": 2200,
    "strikes": 1,
    "accuracy": 780,
    "cooldown": 4,
    "target": "enemy",
    "applies": [
      {
        "status": "expose",
        "chance": 500,
        "stacks": 1
      }
    ]
  },
  {
    "id": "hex_curse",
    "element": "dark",
    "range": 4,
    "pattern": "single",
    "power": 800,
    "strikes": 1,
    "accuracy": 900,
    "cooldown": 3,
    "target": "enemy",
    "applies": [
      {
        "status": "weaken",
        "chance": 800,
        "stacks": 1
      },
      {
        "status": "blind",
        "chance": 400,
        "stacks": 1
      }
    ]
  },
  {
    "id": "riptide",
    "element": "water",
    "range": 3,
    "pattern": "pierce",
    "power": 900,
    "strikes": 2,
    "accuracy": 850,
    "cooldown": 3,
    "target": "enemy",
    "applies": [
      {
        "status": "mire",
        "chance": 550,
        "stacks": 1
      }
    ]
  },
  {
    "id": "arc_bolt",
    "element": "electric",
    "range": 5,
    "pattern": "single",
    "power": 1500,
    "strikes": 1,
    "accuracy": 830,
    "cooldown": 3,
    "target": "enemy",
    "applies": [
      {
        "status": "stun",
        "chance": 250,
        "stacks": 1
      }
    ]
  },
  {
    "id": "swift_edge",
    "element": "wind",
    "range": 1,
    "pattern": "single",
    "power": 9600,
    "strikes": 1,
    "accuracy": 900,
    "cooldown": 2,
    "target": "enemy",
    "scaling": {
      "stat": "speed",
      "source": "base"
    }
  },
  {
    "id": "guard_wall",
    "element": "neutral",
    "range": 0,
    "pattern": "single",
    "power": 0,
    "strikes": 0,
    "accuracy": 1000,
    "cooldown": 4,
    "target": "self",
    "self_applies": [
      {
        "status": "block",
        "chance": 1000,
        "stacks": 2
      }
    ]
  },
  {
    "id": "war_cry",
    "element": "neutral",
    "range": 1,
    "pattern": "column",
    "power": 0,
    "strikes": 0,
    "accuracy": 1000,
    "cooldown": 4,
    "target": "ally",
    "applies": [
      {
        "status": "rally",
        "chance": 1000,
        "stacks": 1
      }
    ]
  },
  {
    "id": "quickstep",
    "element": "neutral",
    "range": 0,
    "pattern": "single",
    "power": 0,
    "strikes": 0,
    "accuracy": 1000,
    "cooldown": 3,
    "target": "self",
    "self_applies": [
      {
        "status": "haste",
        "chance": 1000,
        "stacks": 1
      },
      {
        "status": "focus",
        "chance": 1000,
        "stacks": 1
      }
    ]
  },
  {
    "id": "purify",
    "element": "neutral",
    "range": 2,
    "pattern": "single",
    "power": 0,
    "strikes": 0,
    "accuracy": 1000,
    "cooldown": 3,
    "target": "ally",
    "strips": {
      "categories": [
        "dot",
        "stat_debuff",
        "control"
      ],
      "stacks": 3
    }
  },
  {
    "id": "unmake",
    "element": "dark",
    "range": 3,
    "pattern": "single",
    "power": 600,
    "strikes": 1,
    "accuracy": 900,
    "cooldown": 4,
    "target": "enemy",
    "strips": {
      "categories": [
        "buff",
        "shield"
      ],
      "stacks": 2
    }
  },
  {
    "id": "mend",
    "element": "neutral",
    "range": 2,
    "pattern": "single",
    "power": 0,
    "strikes": 0,
    "accuracy": 1000,
    "restores": 600,
    "cooldown": 3,
    "target": "ally"
  },
  {
    "id": "siphon",
    "element": "neutral",
    "range": 2,
    "pattern": "single",
    "power": 1000,
    "strikes": 1,
    "accuracy": 900,
    "drains": 500,
    "cooldown": 3,
    "target": "enemy"
  }
]`

// Archetypes is the five role presets the tests reason about, moved here for
// the same reason and in the same breath: their kits name the skills above.
const Archetypes = `[
  {
    "id": "bulwark",
    "role": "the front line that absorbs a round rather than winning one: the most health the budget sells, and the shield that makes it count",
    "column": 2,
    "stats": {
      "hp": {
        "base": 1440,
        "max": 4800
      },
      "attack": {
        "base": 150,
        "max": 480
      },
      "defense": {
        "base": 120,
        "max": 400
      },
      "speed": {
        "base": 24,
        "max": 80
      },
      "accuracy": {
        "base": 18,
        "max": 60
      },
      "dodge": {
        "base": 6,
        "max": 20
      }
    },
    "skills": [
      "strike",
      "sever",
      "guard_wall",
      "war_cry"
    ]
  },
  {
    "id": "vanguard",
    "role": "the front line that trades: enough durability to stand in it and enough attack to make standing there a threat",
    "column": 2,
    "stats": {
      "hp": {
        "base": 1080,
        "max": 3600
      },
      "attack": {
        "base": 180,
        "max": 620
      },
      "defense": {
        "base": 160,
        "max": 560
      },
      "speed": {
        "base": 32,
        "max": 110
      },
      "accuracy": {
        "base": 32,
        "max": 110
      },
      "dodge": {
        "base": 18,
        "max": 60
      }
    },
    "skills": [
      "strike",
      "ember_lance",
      "detonate",
      "cinder_burst"
    ]
  },
  {
    "id": "sentinel",
    "role": "armour rather than health, so it shrugs off many small hits and needs help with the one big one",
    "column": 2,
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
    },
    "skills": [
      "strike",
      "riptide",
      "guard_wall",
      "purify"
    ]
  },
  {
    "id": "duelist",
    "role": "the middle line's damage: the hardest single hit in the budget, paid for with the durability to survive being answered",
    "column": 1,
    "stats": {
      "hp": {
        "base": 780,
        "max": 2600
      },
      "attack": {
        "base": 240,
        "max": 800
      },
      "defense": {
        "base": 96,
        "max": 320
      },
      "speed": {
        "base": 45,
        "max": 150
      },
      "accuracy": {
        "base": 54,
        "max": 180
      },
      "dodge": {
        "base": 27,
        "max": 90
      }
    },
    "skills": [
      "strike",
      "bolt",
      "gale_slash",
      "swift_edge",
      "flurry"
    ]
  },
  {
    "id": "skirmisher",
    "role": "the back line that acts first and lands the debuffs the rest of the squad is paid to detonate",
    "column": 0,
    "stats": {
      "hp": {
        "base": 660,
        "max": 2200
      },
      "attack": {
        "base": 228,
        "max": 760
      },
      "defense": {
        "base": 75,
        "max": 250
      },
      "speed": {
        "base": 60,
        "max": 200
      },
      "accuracy": {
        "base": 72,
        "max": 240
      },
      "dodge": {
        "base": 45,
        "max": 150
      }
    },
    "skills": [
      "bolt",
      "venom_fang",
      "creeping_rot",
      "arc_bolt"
    ]
  }
]`

// Roster is the ten-unit bench the engine tests fight on: two full teams whose
// affinities and kits between them reach every element, every shape and every
// status the books declare.
//
// It was the shipped roster until the cast became real. An engine test wants a
// battle that exercises everything, and a shipped roster wants to be the game --
// those are different jobs, and one file could not do both once the game had one
// character in it.
//
// One unit carries a passive, and it is here rather than in the shipped roster
// for the reason the rest of this bench exists: an engine test needs every kind
// of event a battle can produce, and the shipped cast holds no trait yet because
// which trait a borrowed character has is a design decision rather than an
// engine one. The trait itself is the shipped `endurance` — the bench uses the
// real status and passive books, so a fixture inventing its own would be testing
// a book nobody ships.
const Roster = `{
  "units": [
    {
      "id": "ally.bulwark",
      "name": "Bulwark",
      "side": "ally",
      "slot": [
        2,
        1
      ],
      "element": [
        "ground",
        "metal"
      ],
      "stats": {
        "hp": 4800,
        "attack": 480,
        "defense": 400,
        "speed": 80,
        "accuracy": 60,
        "dodge": 20
      },
      "skills": [
        "strike",
        "sever",
        "guard_wall",
        "war_cry"
      ],
      "passives": [
        "endurance"
      ]
    },
    {
      "id": "ally.vanguard",
      "name": "Vanguard",
      "side": "ally",
      "slot": [
        2,
        0
      ],
      "element": [
        "fire",
        "metal"
      ],
      "stats": {
        "hp": 3600,
        "attack": 620,
        "defense": 560,
        "speed": 110,
        "accuracy": 110,
        "dodge": 60
      },
      "skills": [
        "strike",
        "ember_lance",
        "detonate",
        "cinder_burst"
      ]
    },
    {
      "id": "ally.sentinel",
      "name": "Sentinel",
      "side": "ally",
      "slot": [
        2,
        2
      ],
      "element": [
        "water",
        "ice"
      ],
      "stats": {
        "hp": 3100,
        "attack": 500,
        "defense": 800,
        "speed": 90,
        "accuracy": 80,
        "dodge": 30
      },
      "skills": [
        "strike",
        "riptide",
        "guard_wall",
        "purify"
      ]
    },
    {
      "id": "ally.duelist",
      "name": "Duelist",
      "side": "ally",
      "slot": [
        1,
        1
      ],
      "element": [
        "wind",
        "ice"
      ],
      "stats": {
        "hp": 2600,
        "attack": 800,
        "defense": 320,
        "speed": 150,
        "accuracy": 180,
        "dodge": 90
      },
      "skills": [
        "strike",
        "bolt",
        "gale_slash",
        "swift_edge",
        "flurry",
        "mend"
      ]
    },
    {
      "id": "ally.skirmisher",
      "name": "Skirmisher",
      "side": "ally",
      "slot": [
        0,
        1
      ],
      "element": [
        "grass",
        "electric"
      ],
      "stats": {
        "hp": 2200,
        "attack": 760,
        "defense": 250,
        "speed": 200,
        "accuracy": 240,
        "dodge": 150
      },
      "skills": [
        "bolt",
        "venom_fang",
        "creeping_rot",
        "arc_bolt"
      ]
    },
    {
      "id": "foe.bulwark",
      "name": "Warden",
      "side": "enemy",
      "slot": [
        2,
        1
      ],
      "element": [
        "fire",
        "ground"
      ],
      "stats": {
        "hp": 4800,
        "attack": 480,
        "defense": 400,
        "speed": 80,
        "accuracy": 60,
        "dodge": 20
      },
      "skills": [
        "strike",
        "ember_lance",
        "guard_wall",
        "war_cry",
        "siphon"
      ]
    },
    {
      "id": "foe.vanguard",
      "name": "Bramble",
      "side": "enemy",
      "slot": [
        2,
        0
      ],
      "element": [
        "grass",
        "ice"
      ],
      "stats": {
        "hp": 3600,
        "attack": 620,
        "defense": 560,
        "speed": 110,
        "accuracy": 110,
        "dodge": 60
      },
      "skills": [
        "strike",
        "venom_fang",
        "creeping_rot",
        "purify"
      ]
    },
    {
      "id": "foe.sentinel",
      "name": "Ironclad",
      "side": "enemy",
      "slot": [
        2,
        2
      ],
      "element": [
        "metal",
        "electric"
      ],
      "stats": {
        "hp": 3100,
        "attack": 500,
        "defense": 800,
        "speed": 90,
        "accuracy": 80,
        "dodge": 30
      },
      "skills": [
        "strike",
        "sever",
        "arc_bolt",
        "guard_wall"
      ]
    },
    {
      "id": "foe.duelist",
      "name": "Tidecaller",
      "side": "enemy",
      "slot": [
        1,
        1
      ],
      "element": [
        "water",
        "wind"
      ],
      "stats": {
        "hp": 2600,
        "attack": 800,
        "defense": 320,
        "speed": 150,
        "accuracy": 180,
        "dodge": 90
      },
      "skills": [
        "strike",
        "bolt",
        "riptide",
        "gale_slash",
        "flurry",
        "purify"
      ]
    },
    {
      "id": "foe.skirmisher",
      "name": "Nightveil",
      "side": "enemy",
      "slot": [
        0,
        1
      ],
      "element": "dark",
      "stats": {
        "hp": 2200,
        "attack": 760,
        "defense": 250,
        "speed": 200,
        "accuracy": 240,
        "dodge": 150
      },
      "skills": [
        "bolt",
        "hex_curse",
        "unmake",
        "quickstep"
      ]
    }
  ]
}`

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
        "image": "assets/fixture/bloom.svg",
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
	SaveSkill(skill.Skill) error
	SkillDeps() skill.Deps
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
	// bloom.svg is the grown form's own picture. The bench carries one so that
	// per-stage art is exercised by every test that walks this cast, rather than
	// only by the one test written for it.
	for _, name := range []string{"adept.svg", "sprout.svg", "bloom.svg"} {
		if err := os.WriteFile(filepath.Join(art, name), []byte(Art), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	// Skills first, then the presets whose kits name them, then the origins and
	// the characters that name both: each book validates against the ones under
	// it, so a fixture written out of order is refused by its own parser.
	// Parsed through the skill book's own parser rather than encoding/json: an
	// element is a name on the wire and only ParseBook knows how to read one,
	// and going through it means a fixture the game would refuse never reaches
	// a test.
	deps, err := func() (skill.Deps, error) {
		library, err := reload()
		if err != nil {
			return skill.Deps{}, err
		}
		return library.SkillDeps(), nil
	}()
	if err != nil {
		return fmt.Errorf("reload for the skill books: %w", err)
	}
	parsed, err := skill.ParseBook([]byte(`{"skills":`+Skills+`}`), deps)
	if err != nil {
		return fmt.Errorf("parse the fixture skills: %w", err)
	}
	for _, carried := range parsed.Skills() {
		library, err := reload()
		if err != nil {
			return fmt.Errorf("reload before %s: %w", carried.ID, err)
		}
		if err := library.SaveSkill(carried); err != nil {
			return fmt.Errorf("save %s: %w", carried.ID, err)
		}
	}

	// The presets are the one book the authoring tool never writes, so they are
	// merged into the file rather than saved through the library.
	if err := appendTo(filepath.Join(dir, "archetypes.json"), "archetypes", Archetypes); err != nil {
		return err
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

// appendTo merges entries into the list a book file holds under one key. It is
// used only for the presets, which no library method writes.
func appendTo(path, key, added string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var book map[string]any
	if err := json.Unmarshal(raw, &book); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var entries []any
	if err := json.Unmarshal([]byte(added), &entries); err != nil {
		return fmt.Errorf("decode the fixture %s: %w", key, err)
	}
	held, _ := book[key].([]any)
	book[key] = append(held, entries...)
	out, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
