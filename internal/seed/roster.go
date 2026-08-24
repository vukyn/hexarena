package seed

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

type rosterFile struct {
	Units []struct {
		ID      string             `json:"id"`
		Name    string             `json:"name"`
		Side    string             `json:"side"`
		Slot    [2]int             `json:"slot"`
		Element element.Affinity   `json:"element"`
		Stats   progression.Values `json:"stats"`
		Skills  []string           `json:"skills"`
	} `json:"units"`
}

// RosterFile is the raw seed roster.
func RosterFile() ([]byte, error) { return files.ReadFile("data/roster.json") }

// Roster parses the embedded seed roster. The stats are written out flat rather
// than as evolution lines, because the roster exists to exercise the engine
// rather than to be the real cast.
func Roster() ([]battle.Roster, error) {
	raw, err := RosterFile()
	if err != nil {
		return nil, err
	}
	var file rosterFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode roster: %w", err)
	}
	out := make([]battle.Roster, 0, len(file.Units))
	for _, unit := range file.Units {
		var side hex.Side
		switch unit.Side {
		case "ally":
			side = hex.SideAlly
		case "enemy":
			side = hex.SideEnemy
		default:
			return nil, fmt.Errorf("unit %q is on the unknown side %q", unit.ID, unit.Side)
		}
		out = append(out, battle.Roster{
			ID: unit.ID, Name: unit.Name, Side: side,
			Slot:     hex.Offset{Col: unit.Slot[0], Row: unit.Slot[1]},
			Affinity: unit.Element, Stats: unit.Stats, Skills: unit.Skills,
		})
	}
	return out, nil
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
	return battle.Books{
		Rules: rules, Chart: chart, Bounds: bounds, Limits: limits,
		Patterns: patterns, Statuses: statuses, Skills: skills,
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
