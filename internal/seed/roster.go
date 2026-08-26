package seed

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// rosterEntry is one placement. It comes in two forms and never in both at
// once: either the numbers are written out inline, or the entry names a
// character and a level and everything else is resolved from the cast book.
//
// The pointer fields are what makes the two forms distinguishable. A plain
// string and a plain Values cannot say whether they were authored or merely
// left at their zero value, and this parser has to reject the mixture rather
// than guess at it.
type rosterEntry struct {
	ID   string `json:"id"`
	Side string `json:"side"`
	Slot [2]int `json:"slot"`
	// Character names an entry in the cast book. When it is set, Name,
	// Element, Stats and Skills must all be absent: two sources for the same
	// number is how the two drift apart.
	Character string `json:"character"`
	// Level is required with Character and meaningless without it, because
	// resolving an evolution line needs one.
	Level    *int                `json:"level"`
	Name     *string             `json:"name"`
	Element  *element.Affinity   `json:"element"`
	Stats    *progression.Values `json:"stats"`
	Skills   []string            `json:"skills"`
	Passives []string            `json:"passives"`
}

type rosterFile struct {
	Units []rosterEntry `json:"units"`
}

// RosterFile is the raw seed roster.
func RosterFile() ([]byte, error) { return files.ReadFile("data/roster.json") }

// Roster parses the embedded seed roster.
func Roster() ([]battle.Roster, error) {
	raw, err := RosterFile()
	if err != nil {
		return nil, err
	}
	characters, err := Cast()
	if err != nil {
		return nil, err
	}
	return ParseRoster(raw, characters)
}

// ParseRoster resolves a roster declaration against the cast book.
//
// The flat form — a name, an element, a stat line and a kit written out in
// place — is what the seed roster uses, because that roster exists to exercise
// the engine rather than to be the real cast. The reference form names a
// character and a level instead, and is what an authored cast is placed with.
//
// It takes bytes rather than a path for the same reason the core parsers do:
// the caller owns the file access, and a test can hand it a fixture.
func ParseRoster(raw []byte, characters *cast.Book) ([]battle.Roster, error) {
	if characters == nil {
		return nil, fmt.Errorf("a roster cannot be resolved without the cast book")
	}
	var file rosterFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode roster: %w", err)
	}
	out := make([]battle.Roster, 0, len(file.Units))
	for _, unit := range file.Units {
		resolved, err := resolveRosterEntry(unit, characters)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

func resolveRosterEntry(unit rosterEntry, characters *cast.Book) (battle.Roster, error) {
	if unit.ID == "" {
		return battle.Roster{}, fmt.Errorf("a roster entry needs an id")
	}
	var side hex.Side
	switch unit.Side {
	case "ally":
		side = hex.SideAlly
	case "enemy":
		side = hex.SideEnemy
	default:
		return battle.Roster{}, fmt.Errorf("unit %q is on the unknown side %q", unit.ID, unit.Side)
	}
	entry := battle.Roster{
		ID:   unit.ID,
		Side: side,
		Slot: hex.Offset{Col: unit.Slot[0], Row: unit.Slot[1]},
	}

	if unit.Character == "" {
		if unit.Level != nil {
			return battle.Roster{}, fmt.Errorf(
				"unit %q gives a level but no character, and an inline stat line is already resolved", unit.ID)
		}
		if unit.Name == nil || unit.Element == nil || unit.Stats == nil {
			return battle.Roster{}, fmt.Errorf(
				"unit %q names no character, so it needs a name, an element and a stat line of its own", unit.ID)
		}
		entry.Name = *unit.Name
		entry.Affinity = *unit.Element
		entry.Stats = *unit.Stats
		entry.Skills = unit.Skills
		entry.Passives = unit.Passives
		return entry, nil
	}

	// The mixture is rejected rather than resolved by precedence. A precedence
	// rule silently ignores half of what was authored, and the half it ignores
	// is the half someone edited.
	restated := make([]string, 0, 4)
	if unit.Name != nil {
		restated = append(restated, "name")
	}
	if unit.Element != nil {
		restated = append(restated, "element")
	}
	if unit.Stats != nil {
		restated = append(restated, "stats")
	}
	if unit.Skills != nil {
		restated = append(restated, "skills")
	}
	if unit.Passives != nil {
		restated = append(restated, "passives")
	}
	if len(restated) > 0 {
		return battle.Roster{}, fmt.Errorf(
			"unit %q references the character %q and also restates %v; a character reference is the single source for those",
			unit.ID, unit.Character, restated)
	}
	if unit.Level == nil {
		return battle.Roster{}, fmt.Errorf(
			"unit %q references the character %q but gives no level, and an evolution line cannot be resolved without one",
			unit.ID, unit.Character)
	}
	level := *unit.Level
	if level < 1 || level > progression.LevelCap {
		return battle.Roster{}, fmt.Errorf("unit %q is at level %d, outside 1..%d",
			unit.ID, level, progression.LevelCap)
	}
	character, known := characters.Get(unit.Character)
	if !known {
		return battle.Roster{}, fmt.Errorf("unit %q references the unknown character %q",
			unit.ID, unit.Character)
	}
	stats, _, err := character.Resolve(level)
	if err != nil {
		return battle.Roster{}, fmt.Errorf("unit %q: %w", unit.ID, err)
	}
	entry.Name = character.Name
	entry.Affinity = character.Element
	entry.Stats = stats
	entry.Skills = character.Skills
	entry.Passives = character.Passives
	return entry, nil
}

// Books assembles every parsed book a battle needs.
func Books() (battle.Books, error) {
	rules, err := CombatRules()
	if err != nil {
		return battle.Books{}, err
	}
	chart, err := ElementChart()
	if err != nil {
		return battle.Books{}, err
	}
	bounds, err := ModifierBounds()
	if err != nil {
		return battle.Books{}, err
	}
	limits, err := ProgressionLimits()
	if err != nil {
		return battle.Books{}, err
	}
	patterns, err := PatternBook()
	if err != nil {
		return battle.Books{}, err
	}
	statuses, err := StatusBook()
	if err != nil {
		return battle.Books{}, err
	}
	skills, err := SkillBook()
	if err != nil {
		return battle.Books{}, err
	}
	passives, err := PassiveBook()
	if err != nil {
		return battle.Books{}, err
	}
	return battle.Books{
		Rules: rules, Chart: chart, Bounds: bounds, Limits: limits,
		Patterns: patterns, Statuses: statuses, Skills: skills, Passives: passives,
	}, nil
}

// NewBattle assembles a battle from the embedded data.
func NewBattle(seed uint64) (*battle.Battle, error) {
	books, err := Books()
	if err != nil {
		return nil, err
	}
	roster, err := Roster()
	if err != nil {
		return nil, err
	}
	return battle.New(books, seed, roster)
}
