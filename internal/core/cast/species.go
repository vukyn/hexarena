package cast

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Species is what a unit *is*, as against what it is made of and how it fights.
//
// An element answers the first of those and an archetype the second, and
// neither says whether a creature has a shell, roots or a lineage — so a skill
// named after a body was free to land on a body it did not fit. This is the
// missing axis, and it is deliberately the thinnest one: a species carries an
// id, a word for a screen and an optional note, and nothing else. No stats, no
// kit, no rule of its own.
//
// # Why it never reaches the engine
//
// Nothing in battle branches on a species, exactly as nothing branches on an
// archetype. A species is a **carry rule** — it decides who may hold a skill,
// which is settled while a character is authored — plus a word a browser
// prints. Pushing it into battle.Roster would put a fact into the replayable
// core that no replay reads. See CLAUDE.md, "What a restriction can enforce".
type Species struct {
	ID string `json:"id"`
	// Name is the word shown beside the id. It is required, which is the one
	// place this differs from a passive's optional name: a species has no
	// numbers beside it on a screen, so an id with no name is the whole of what
	// a reader gets.
	Name string `json:"name"`
	// Note is optional prose about where the line is drawn — which is worth
	// having, because "is a dragon" is a judgement about fiction rather than a
	// fact about a stat line.
	Note string `json:"note,omitempty"`
}

// SpeciesBook is the declared kinds of creature.
type SpeciesBook struct {
	species []Species
	byID    map[string]Species
}

type speciesFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Note string `json:"note"`
}

type speciesBookFile struct {
	Species []speciesFile `json:"species"`
}

type speciesMarshalFile struct {
	Species []Species `json:"species"`
}

// ParseSpecies reads a species catalog. It never touches the filesystem.
//
// An empty catalog is allowed, for the same reason an empty origin catalog is:
// a project where nothing has needed the axis yet is a starting point rather
// than a data error, and every character's species list is optional.
func ParseSpecies(raw []byte) (*SpeciesBook, error) {
	var file speciesBookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode species book: %w", err)
	}
	book := &SpeciesBook{byID: make(map[string]Species, len(file.Species))}
	for _, declared := range file.Species {
		built, err := resolveSpecies(declared)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("species %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.species = append(book.species, built)
	}
	return book, nil
}

func resolveSpecies(declared speciesFile) (Species, error) {
	if err := checkSlug("species", declared.ID); err != nil {
		return Species{}, err
	}
	if strings.TrimSpace(declared.Name) == "" {
		return Species{}, fmt.Errorf("species %q has no name, and its id is the only other thing a screen could show", declared.ID)
	}
	return Species{
		ID: declared.ID, Name: strings.TrimSpace(declared.Name),
		Note: strings.TrimSpace(declared.Note),
	}, nil
}

// Get returns a species by id.
func (b *SpeciesBook) Get(id string) (Species, bool) {
	found, ok := b.byID[id]
	return found, ok
}

// All returns every species in declaration order. It never ranges over the
// lookup map, because Go randomises that and the order reaches an output.
func (b *SpeciesBook) All() []Species {
	out := make([]Species, len(b.species))
	copy(out, b.species)
	return out
}

// IDs returns every species id in declaration order, which is what a refusal
// listing the alternatives wants.
func (b *SpeciesBook) IDs() []string {
	out := make([]string, 0, len(b.species))
	for _, one := range b.species {
		out = append(out, one.ID)
	}
	return out
}

// Marshal writes the catalog as a data file: two-space indented JSON sorted by
// id, for the same reason every other book sorts.
func (b *SpeciesBook) Marshal() ([]byte, error) {
	sorted := b.All()
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out, err := json.MarshalIndent(speciesMarshalFile{Species: sorted}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode species book: %w", err)
	}
	return append(out, '\n'), nil
}

// Append returns a new catalog holding the existing species plus the extra
// ones, validated exactly as a parse would validate them.
func (b *SpeciesBook) Append(extra ...Species) (*SpeciesBook, error) {
	combined := speciesMarshalFile{Species: append(b.All(), extra...)}
	raw, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("encode species book: %w", err)
	}
	return ParseSpecies(raw)
}
