package cast

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Archetype is the preset a character of a given role starts from: the shape of
// the stat curve and the kit that goes with it, before any tuning.
//
// A preset is a starting point and not a constraint. A character records which
// archetype it came from and is then free to differ, which is what keeps two
// characters of the same role from being the same unit with two names.
type Archetype struct {
	ID string `json:"id"`
	// Role is one line saying what the preset is for. It is what an authoring
	// tool shows next to the id.
	Role string `json:"role"`
	// Column is the formation column the role belongs in, counted from the
	// front of its own side.
	Column int `json:"column"`
	// Stats is the suggested curve for every stat. A preset that does not
	// itself fit the budget is a bug, not a stretch goal: it would hand every
	// author a stat line that fails later.
	Stats progression.Table `json:"stats"`
	// Skills is the suggested kit.
	Skills []string `json:"skills"`
	// Demands is the distinct non-neutral elements the kit requires, derived
	// from Skills at parse time by skill.Demands.
	//
	// It is never read from the JSON, which is the whole point: an authored
	// hint would be free to drift from the kit it claimed to describe, and the
	// drift would only surface when a character built from the preset was
	// refused by battle.New. A character must carry every element in this set,
	// so a preset demanding two can be carried by exactly one affinity and a
	// preset demanding three cannot be carried at all.
	Demands []element.Element `json:"-"`
}

// clone copies the slices a caller could otherwise mutate through.
func (a Archetype) clone() Archetype {
	out := a
	out.Skills = make([]string, len(a.Skills))
	copy(out.Skills, a.Skills)
	out.Demands = make([]element.Element, len(a.Demands))
	copy(out.Demands, a.Demands)
	return out
}

// DemandNames is the demanded elements written out, which is what a prompt and
// a listing want.
func (a Archetype) DemandNames() []string {
	out := make([]string, 0, len(a.Demands))
	for _, member := range a.Demands {
		out = append(out, member.String())
	}
	return out
}

// ArchetypeBook is the declared presets.
type ArchetypeBook struct {
	archetypes []Archetype
	byID       map[string]Archetype
}

type archetypeFile struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Column int    `json:"column"`
	// Stats is a pointer so an omitted table is an error rather than a preset
	// of zeroes.
	Stats  *progression.Table `json:"stats"`
	Skills []string           `json:"skills"`
}

type archetypeBookFile struct {
	Archetypes []archetypeFile `json:"archetypes"`
}

// ArchetypeDeps are what a preset's declarations are checked against.
type ArchetypeDeps struct {
	Skills *skill.Book
	Limits progression.Limits
	Rules  combat.Rules
}

func (d ArchetypeDeps) validate() error {
	if d.Skills == nil {
		return fmt.Errorf("archetypes cannot be validated without the skill book")
	}
	if err := d.Limits.Validate(); err != nil {
		return err
	}
	return d.Rules.Validate()
}

// ParseArchetypes reads the presets and checks every name and number each one
// uses. It never touches the filesystem.
func ParseArchetypes(raw []byte, deps ArchetypeDeps) (*ArchetypeBook, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	var file archetypeBookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode archetype book: %w", err)
	}
	book := &ArchetypeBook{byID: make(map[string]Archetype, len(file.Archetypes))}
	for _, declared := range file.Archetypes {
		built, err := resolveArchetype(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("archetype %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.archetypes = append(book.archetypes, built)
	}
	return book, nil
}

func resolveArchetype(declared archetypeFile, deps ArchetypeDeps) (Archetype, error) {
	if err := checkSlug("archetype", declared.ID); err != nil {
		return Archetype{}, err
	}
	fail := func(format string, args ...any) (Archetype, error) {
		return Archetype{}, fmt.Errorf("archetype %q: "+format, append([]any{declared.ID}, args...)...)
	}
	role := strings.TrimSpace(declared.Role)
	if role == "" {
		return fail("has no role line saying what it is for")
	}
	if strings.ContainsAny(role, "\n\r") {
		return fail("has a role spanning more than one line")
	}
	if declared.Column < 0 || declared.Column >= hex.FormationCols {
		return fail("sits in column %d, want between 0 and %d",
			declared.Column, hex.FormationCols-1)
	}
	if declared.Stats == nil {
		return fail("declares no stat curve")
	}
	if err := deps.Limits.CheckTable(*declared.Stats, deps.Rules); err != nil {
		return fail("%w", err)
	}
	kit, err := resolveSkills(declared.Skills, deps.Skills)
	if err != nil {
		return fail("%w", err)
	}
	// A preset is a starting point for any character built from it, so a skill
	// only certain characters may carry has no place in one: every character
	// from the preset except those named would be refused, and the refusal
	// would land on the author of the character rather than on the author of
	// the preset. This is checked here because this is the only place that
	// holds both books — the preset's id and each skill's restriction — without
	// a second lookup.
	for _, carried := range kit {
		if carried.Restrict.NamesCharacters() {
			return fail("has %q in its kit, which only %s may carry, and a preset is shared by every character built from it",
				carried.ID, strings.Join(carried.Restrict.Characters, " or "))
		}
		if !carried.Restrict.AllowsArchetype(declared.ID) {
			return fail("has %q in its kit, which only the %s archetype may carry",
				carried.ID, strings.Join(carried.Restrict.Archetypes, " or "))
		}
	}
	// A unit carries at most two elements, and it must have every element its
	// kit demands, so a kit demanding three can never be carried by anybody.
	// Rejecting it here is the difference between a preset that is unusable and
	// a preset that quietly produces characters battle.New refuses.
	demands := skill.Demands(kit)
	if len(demands) > 2 {
		names := make([]string, 0, len(demands))
		for _, member := range demands {
			names = append(names, member.String())
		}
		return fail("has a kit demanding %d elements (%s), and a unit may carry at most 2",
			len(demands), strings.Join(names, ", "))
	}
	return Archetype{
		ID: declared.ID, Role: role, Column: declared.Column,
		Stats: *declared.Stats, Skills: skillIDs(kit), Demands: demands,
	}, nil
}

// Get returns a preset by id.
func (b *ArchetypeBook) Get(id string) (Archetype, bool) {
	found, ok := b.byID[id]
	if !ok {
		return Archetype{}, false
	}
	return found.clone(), true
}

// All returns every preset in declaration order.
func (b *ArchetypeBook) All() []Archetype {
	out := make([]Archetype, 0, len(b.archetypes))
	for _, entry := range b.archetypes {
		out = append(out, entry.clone())
	}
	return out
}

// IDs returns every preset's id in declaration order, which is what a usage
// message and a prompt want.
func (b *ArchetypeBook) IDs() []string {
	out := make([]string, 0, len(b.archetypes))
	for _, entry := range b.archetypes {
		out = append(out, entry.ID)
	}
	return out
}
