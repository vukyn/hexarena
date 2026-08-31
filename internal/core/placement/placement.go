// Package placement is what a character becomes when somebody fields it.
//
// A character is a **definition** — a curve, a learnset, an evolution line — and
// a placement is a decision made about one: which level, which form, which four
// of the skills it knows, which trait, and where it stands. The two are kept
// apart so that one character can stand in a dozen encounters at a dozen levels
// without being written down a dozen times.
//
// A Squad is a named list of placements, one side's worth. It has no side of its
// own: the same squad can be fielded as either half of a battle, which is what
// lets one be measured against several opponents, and against itself.
//
// The package is pure. It parses bytes handed to it and never touches the
// filesystem — internal/seed hands it the copy go:embed baked in, and
// internal/forge hands it the file an author is editing.
package placement

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// Placement is one unit as it takes the field.
type Placement struct {
	// ID is what this unit is called in a log, and it is unique within a squad
	// rather than globally: the side is prefixed when the squad takes the field,
	// so a squad fought against a copy of itself still has two distinguishable
	// halves.
	ID string `json:"id"`
	// Character names an entry in the cast book. A placement is always a
	// reference — the flat form, where a stat line is written out in place,
	// belongs to the seed roster and to the engine's own fixtures.
	Character string `json:"character"`
	Level     int    `json:"level"`
	// Stage is the form this placement fielded, and it is a *choice*: a level
	// allows a form rather than dictating one. Absent means the furthest the
	// level reaches, which is what a placement meant before it could choose —
	// and on a line that forks there is no such thing, so one has to be named.
	Stage string `json:"stage,omitempty"`
	// Slot is the cell in this side's own 3x3 formation, authored from that
	// side's point of view. hex.Place maps it onto the shared board.
	Slot     hex.Offset `json:"slot"`
	Skills   []string   `json:"skills"`
	Passives []string   `json:"passives,omitempty"`
}

// Squad is one side's worth of placements under a name.
type Squad struct {
	ID    string      `json:"id"`
	Name  string      `json:"name,omitempty"`
	Units []Placement `json:"units"`
}

// Clone is a deep copy, so a builder editing a squad cannot reach through a
// shared slice into whatever is holding the saved one.
func (s Squad) Clone() Squad {
	out := s
	out.Units = make([]Placement, len(s.Units))
	for i, unit := range s.Units {
		out.Units[i] = unit.Clone()
	}
	return out
}

// Clone is a deep copy of one placement.
func (p Placement) Clone() Placement {
	out := p
	out.Skills = slices.Clone(p.Skills)
	out.Passives = slices.Clone(p.Passives)
	return out
}

// Equal reports whether two squads are the same squad: every field of the squad
// and of every member, in the order the members are written down.
//
// Order is part of the answer rather than an artefact of how it is computed. A
// kit's order is the kit, and a squad's order is what the turn queue breaks a
// tie by, so two squads holding the same names in a different order are two
// different squads and a comparison that sorted first would call them one.
//
// It lives beside Clone because both answer the same question — what a squad is
// made of. A caller that wrote its own comparison would be a second declaration
// of that list, and the copy is the one that goes stale the day a field is
// added, silently: a missed field compares equal and the difference is thrown
// away with nobody asked about it.
func (s Squad) Equal(other Squad) bool {
	if s.ID != other.ID || s.Name != other.Name || len(s.Units) != len(other.Units) {
		return false
	}
	for i, unit := range s.Units {
		if !unit.Equal(other.Units[i]) {
			return false
		}
	}
	return true
}

// Equal reports whether two placements field the same unit the same way.
//
// A nil list and an empty one are one answer here, which is what slices.Equal
// says: having chosen nothing and having chosen nothing yet are the same squad
// on the field, and Take refuses both for the same reason.
func (p Placement) Equal(other Placement) bool {
	return p.ID == other.ID &&
		p.Character == other.Character &&
		p.Level == other.Level &&
		p.Stage == other.Stage &&
		p.Slot == other.Slot &&
		slices.Equal(p.Skills, other.Skills) &&
		slices.Equal(p.Passives, other.Passives)
}

type file struct {
	Squads []Squad `json:"squads"`
}

// Parse reads a squad declaration and checks everything that can be checked
// without a cast book: the ids, the sizes, and the slots.
//
// The cast book is what turns a squad into units, and Take is where that
// happens. Splitting the two is what lets an authoring tool hold a squad that
// does not resolve yet — a unit half-chosen is a normal state to be in while
// building one, and a parser that refused it would make the file unwritable
// until it was finished.
func Parse(raw []byte) ([]Squad, error) {
	var decoded file
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode squads: %w", err)
	}
	seen := make(map[string]bool, len(decoded.Squads))
	for _, squad := range decoded.Squads {
		if squad.ID == "" {
			return nil, fmt.Errorf("a squad needs an id")
		}
		if seen[squad.ID] {
			return nil, fmt.Errorf("two squads are called %q, so naming one of them chooses neither", squad.ID)
		}
		seen[squad.ID] = true
		if err := squad.Validate(); err != nil {
			return nil, fmt.Errorf("squad %q: %w", squad.ID, err)
		}
	}
	return decoded.Squads, nil
}

// Marshal writes squads back in the shape Parse reads.
func Marshal(squads []Squad) ([]byte, error) {
	if squads == nil {
		squads = []Squad{}
	}
	out, err := json.MarshalIndent(file{Squads: squads}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode squads: %w", err)
	}
	return append(out, '\n'), nil
}

// Validate is everything about a squad that does not need the cast book.
//
// ⚠️ It deliberately does **not** ask whether a unit has chosen its four skills.
// A squad being built has units in every state of half-finished, and a file that
// refused to hold one could not be saved and come back to. What refuses an
// unfinished unit is Take, at the moment the squad is asked to fight.
func (s Squad) Validate() error {
	if len(s.Units) == 0 {
		return fmt.Errorf("a squad with nobody in it has nothing to field")
	}
	if len(s.Units) > hex.MaxTeamSize {
		return fmt.Errorf("a squad of %d is more than the %d a side can field",
			len(s.Units), hex.MaxTeamSize)
	}
	ids := make(map[string]bool, len(s.Units))
	slots := make(map[hex.Offset]string, len(s.Units))
	for _, unit := range s.Units {
		if unit.ID == "" {
			return fmt.Errorf("a unit needs an id")
		}
		if ids[unit.ID] {
			return fmt.Errorf("two units are called %q", unit.ID)
		}
		ids[unit.ID] = true
		if unit.Character == "" {
			return fmt.Errorf("unit %q names no character", unit.ID)
		}
		if unit.Level < 1 || unit.Level > progression.LevelCap {
			return fmt.Errorf("unit %q is level %d, outside 1..%d",
				unit.ID, unit.Level, progression.LevelCap)
		}
		if !unit.Slot.OnBoard() || unit.Slot.Col >= hex.FormationCols {
			return fmt.Errorf("unit %q stands at %s, which is not a cell of a %dx%d formation",
				unit.ID, unit.Slot, hex.FormationCols, hex.FormationRows)
		}
		if held, taken := slots[unit.Slot]; taken {
			return fmt.Errorf("unit %q stands at %s, where %q already is",
				unit.ID, unit.Slot, held)
		}
		slots[unit.Slot] = unit.ID
	}
	return nil
}

// Take resolves the squad into the engine's shape, as one side of a battle.
//
// The side is a parameter rather than a field because a squad is not written
// down as somebody's enemy: the same one is fielded as either half, which is
// what lets it be measured against several opponents — and against a copy of
// itself, which is why the ids come back prefixed with the side. Two halves of a
// mirror have to be told apart in a log, and nothing else in a squad can do it.
func (s Squad) Take(side hex.Side, characters *cast.Book) ([]battle.Roster, error) {
	if characters == nil {
		return nil, fmt.Errorf("a squad cannot be fielded without the cast book")
	}
	if !side.Fights() {
		return nil, fmt.Errorf("a squad cannot be fielded on the %s side", side)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	out := make([]battle.Roster, 0, len(s.Units))
	for _, unit := range s.Units {
		resolved, err := unit.resolve(side, characters)
		if err != nil {
			return nil, fmt.Errorf("squad %q: %w", s.ID, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (p Placement) resolve(side hex.Side, characters *cast.Book) (battle.Roster, error) {
	character, known := characters.Get(p.Character)
	if !known {
		return battle.Roster{}, fmt.Errorf("unit %q references the unknown character %q",
			p.ID, p.Character)
	}
	stats, form, err := character.Resolve(p.Level, p.Stage)
	if err != nil {
		return battle.Roster{}, fmt.Errorf("unit %q: %w", p.ID, err)
	}
	skills, passives, err := cast.ChooseLoadout(fmt.Sprintf("unit %q", p.ID),
		p.Skills, p.Passives, character, p.Level, form.Name)
	if err != nil {
		return battle.Roster{}, err
	}
	return battle.Roster{
		ID: side.String() + "." + p.ID,
		// The form's name rather than the character's. What is on the board is
		// the form: it is the form's stat line that fights and the form's
		// learnset that was chosen from, so a placement named for the character
		// puts "Charmander" beside Charizard's health and speed — the one pairing
		// a reader has no way to tell from a real Charmander. The character is
		// still named by the id it was placed with, and identified on the board
		// by its tag, so nothing that needs the line rather than the form loses
		// it here.
		Name:     form.Name,
		Side:     side,
		Slot:     p.Slot,
		Affinity: character.Element,
		Stats:    stats,
		Skills:   skills,
		Passives: passives,
	}, nil
}
