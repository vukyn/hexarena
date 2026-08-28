package cast

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/progression"
)

// Build is one named way to field a character at the level cap: which four of
// the skills its final form knows, and which of its traits fills the single
// trait slot.
//
// It exists because the slots alone do not say what a character is *for*. A
// learnset of nine skills and five traits offers more combinations than anybody
// would field, and until a build was written down the only kit the repository
// could name was "the first four declared" — which is the order the file happens
// to list, not a decision. A build is that decision, authored.
//
// It carries no numbers of its own and cannot: everything a build does is
// already described by the skills and the trait it names, and a second place
// saying how hard it hits would be a second place to drift. Name and Intent are
// the whole of what a build adds, and Intent says why an author would field it
// rather than what it computes.
type Build struct {
	// ID is the build's own identity, and is what a placement or a screen refers
	// to. Prefixed by the character it belongs to by convention, so a listing
	// sorted by id groups a character's directions together.
	ID string
	// Character names an entry in the cast book. A build without one would be a
	// kit belonging to nobody.
	Character string
	// Name is what the build is called, in the language the data is authored in.
	Name string
	// Intent is one clause on why this direction exists. It is the only authored
	// prose about a build, and it must stay intent: the skills describe
	// themselves, and a number here would be a promise this type cannot keep.
	Intent string
	// Skills and Passives are the loadout, and they *choose* rather than list —
	// each entry has to be something the character has learned by the cap, as
	// the form the cap reaches.
	Skills   []string
	Passives []string
}

// buildFile is the shape a build is written in.
type buildFile struct {
	ID        string   `json:"id"`
	Character string   `json:"character"`
	Name      string   `json:"name"`
	Intent    string   `json:"intent"`
	Skills    []string `json:"skills"`
	Passives  []string `json:"passives"`
}

type buildsFile struct {
	Builds []buildFile `json:"builds"`
}

// BuildBook is every authored build, indexed the two ways they are read: by the
// build's own id, and by the character whose directions they are.
type BuildBook struct {
	builds      []Build
	byID        map[string]Build
	byCharacter map[string][]Build
}

// ParseBuilds reads a build catalogue and checks every entry against the cast
// book, which is the only thing that can say whether a kit is fieldable.
//
// It takes bytes rather than a path for the reason every other parser here does:
// the caller owns the file access, and a test hands it a fixture.
//
// A build is validated at the level cap and at the form that cap reaches, because
// that is what a build *is* — the late-game shape of a character. A kit for an
// earlier form is a placement's business (see the roster), not a catalogue's: a
// half-grown unit fields what it has, and there is no decision to record.
func ParseBuilds(raw []byte, characters *Book) (*BuildBook, error) {
	if characters == nil {
		return nil, fmt.Errorf("a build catalogue cannot be checked without the cast book")
	}
	var file buildsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode builds: %w", err)
	}
	book := &BuildBook{
		builds:      make([]Build, 0, len(file.Builds)),
		byID:        make(map[string]Build, len(file.Builds)),
		byCharacter: make(map[string][]Build, len(file.Builds)),
	}
	for _, entry := range file.Builds {
		built, err := resolveBuild(entry, characters)
		if err != nil {
			return nil, err
		}
		if _, taken := book.byID[built.ID]; taken {
			return nil, fmt.Errorf("two builds are called %q", built.ID)
		}
		for _, sibling := range book.byCharacter[built.Character] {
			if sibling.Name == built.Name {
				return nil, fmt.Errorf("%s has two builds named %q, so a listing cannot tell "+
					"them apart", built.Character, built.Name)
			}
		}
		book.builds = append(book.builds, built)
		book.byID[built.ID] = built
		book.byCharacter[built.Character] = append(book.byCharacter[built.Character], built)
	}
	return book, nil
}

func resolveBuild(entry buildFile, characters *Book) (Build, error) {
	if entry.ID == "" {
		return Build{}, fmt.Errorf("a build needs an id")
	}
	if entry.Character == "" {
		return Build{}, fmt.Errorf("the build %q names no character", entry.ID)
	}
	character, known := characters.Get(entry.Character)
	if !known {
		return Build{}, fmt.Errorf("the build %q is for %q, which is in no cast book",
			entry.ID, entry.Character)
	}
	if entry.Name == "" {
		return Build{}, fmt.Errorf("the build %q has no name, and an unnamed direction cannot "+
			"be chosen between", entry.ID)
	}
	// A build's own words are fiction, and a figure in them is a promise nothing
	// computes. Everything a build does is already described by what it names.
	for _, field := range []struct{ what, text string }{
		{"name", entry.Name},
		{"intent", entry.Intent},
	} {
		for _, letter := range field.text {
			if unicode.IsDigit(letter) {
				return Build{}, fmt.Errorf("the build %q spells a number in its %s (%q); "+
					"a build's numbers are the ones its skills and trait already state",
					entry.ID, field.what, field.text)
			}
		}
	}

	_, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
	if err != nil {
		return Build{}, fmt.Errorf("resolve %s for the build %q: %w", entry.Character, entry.ID, err)
	}
	skills, err := chosenFor(entry.ID, "skill", entry.Skills,
		character.SkillsAt(progression.LevelCap, stage.Name), SkillSlots, insisted)
	if err != nil {
		return Build{}, err
	}
	passives, err := chosenFor(entry.ID, "trait", entry.Passives,
		character.PassivesAt(progression.LevelCap, stage.Name), TraitSlots, allowedEmpty)
	if err != nil {
		return Build{}, err
	}
	return Build{
		ID:        entry.ID,
		Character: entry.Character,
		Name:      entry.Name,
		Intent:    entry.Intent,
		Skills:    skills,
		Passives:  passives,
	}, nil
}

// insisted and allowedEmpty are whether a half of the loadout has to be filled,
// named rather than passed as a bare boolean so the call site says which rule it
// is asking for.
//
// They differ for the reason the roster's own pair does: a build that brings no
// skills cannot act, so an empty kit is nothing anybody chose, while a build that
// brings no trait is an ordinary unit and that is a decision like any other.
const (
	insisted     = true
	allowedEmpty = false
)

// chosenFor picks one half of a build's loadout out of what the capped form
// knows, or says why it cannot — with the list, because an author who has just
// been told "no" wants to see what was on offer.
func chosenFor(build, kind string, chosen, available []string, slots int, insist bool) ([]string, error) {
	if len(chosen) == 0 {
		if !insist {
			return nil, nil
		}
		return nil, fmt.Errorf("the build %q chooses no %ss; it has %d slot(s) to fill out of %v",
			build, kind, slots, available)
	}
	if len(chosen) > slots {
		return nil, fmt.Errorf("the build %q brings %d %ss and there are %d slot(s)",
			build, len(chosen), kind, slots)
	}
	out := make([]string, 0, len(chosen))
	for _, id := range chosen {
		if slices.Contains(out, id) {
			return nil, fmt.Errorf("the build %q brings the %s %q twice", build, kind, id)
		}
		if !slices.Contains(available, id) {
			return nil, fmt.Errorf(
				"the build %q brings the %s %q, which its capped form has not learned; it knows %v",
				build, kind, id, available)
		}
		out = append(out, id)
	}
	return out, nil
}

// All is every build, in the order they were declared.
func (b *BuildBook) All() []Build {
	out := make([]Build, 0, len(b.builds))
	for _, entry := range b.builds {
		out = append(out, entry.clone())
	}
	return out
}

// Of is one character's builds, in declaration order, and is empty for a
// character nobody has written a direction for yet.
//
// Empty is a fact rather than a failure: a character whose learnset has one
// obvious kit has nothing to choose between, and a catalogue inventing a second
// direction for it would be saying something untrue.
func (b *BuildBook) Of(character string) []Build {
	found := b.byCharacter[character]
	out := make([]Build, 0, len(found))
	for _, entry := range found {
		out = append(out, entry.clone())
	}
	return out
}

// Get is one build by its own id.
func (b *BuildBook) Get(id string) (Build, bool) {
	found, ok := b.byID[id]
	if !ok {
		return Build{}, false
	}
	return found.clone(), true
}

// Count is how many builds the catalogue holds.
func (b *BuildBook) Count() int { return len(b.builds) }

// clone hands out a copy, so a caller holding a build cannot edit the book's.
func (b Build) clone() Build {
	out := b
	out.Skills = slices.Clone(b.Skills)
	out.Passives = slices.Clone(b.Passives)
	return out
}
