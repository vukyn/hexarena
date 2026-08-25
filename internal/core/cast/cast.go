// Package cast is where a character is authored: who it is, which work it was
// borrowed from, the preset it was tuned from, and the evolution line its stats
// grow along.
//
// A character is a **definition** and a roster entry is a **placement**. The two
// are deliberately separate types in separate packages: the same character may
// stand in a dozen encounters at a dozen levels, and the engine only ever
// receives the flat stat line that falls out of resolving one at a level. That
// is why nothing below this package knows a character exists — battle.Roster
// gains no image, no biography and no origin, because the rules have no use for
// them.
//
// Everything a character names is checked against the book that declares it,
// the same way skill.ParseBook checks the shapes and statuses a skill names: a
// character pointing at a skill, an origin or an archetype that does not exist
// fails at load rather than at the moment it would have mattered.
//
// Like every other core package except battle, this one is a pure function of
// its arguments and never touches the filesystem. That has one visible
// consequence worth knowing about: ValidateImagePath checks the *shape* of an
// authored image path and nothing more. Whether the file is really there is
// cmd/hexforge's business, because only the caller knows what the path is
// relative to.
package cast

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Character is one authored combatant.
type Character struct {
	// ID is the character's slug. It may carry one '.' separating the origin
	// from the name, which is what keeps two works free to use the same name.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Origin and Archetype are ids in their own books. The archetype is
	// recorded rather than consumed: the numbers below are already tuned, and
	// keeping the preset's name is what lets a tool say what a character was
	// built from and what it drifted away from.
	Origin    string `json:"origin"`
	Archetype string `json:"archetype"`
	// Image is a relative path to the character's art. Its shape is validated
	// here; its existence is not. See ValidateImagePath.
	Image   string           `json:"image"`
	Element element.Affinity `json:"element"`
	Bio     string           `json:"bio,omitempty"`
	// Stages is the evolution line the character's stats grow along. A
	// character with one stage is the ordinary case, not a special one.
	Stages progression.Line `json:"stages"`
	Skills []string         `json:"skills"`
}

// Resolve flattens the character at a level into the stat line the battle
// engine works with, and reports which stage it landed in.
func (c Character) Resolve(level int) (progression.Values, progression.Stage, error) {
	return c.Stages.Resolve(level)
}

// clone copies the slices a caller could otherwise mutate through.
func (c Character) clone() Character {
	out := c
	out.Stages = make(progression.Line, len(c.Stages))
	copy(out.Stages, c.Stages)
	out.Skills = make([]string, len(c.Skills))
	copy(out.Skills, c.Skills)
	return out
}

// Book is the authored cast.
type Book struct {
	characters []Character
	byID       map[string]Character
}

type characterFile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Origin    string `json:"origin"`
	Archetype string `json:"archetype"`
	Image     string `json:"image"`
	// Element is a pointer so an omitted field is an error rather than a
	// silent neutral affinity.
	Element *element.Affinity `json:"element"`
	Bio     string            `json:"bio"`
	Stages  progression.Line  `json:"stages"`
	Skills  []string          `json:"skills"`
}

type bookFile struct {
	Characters []characterFile `json:"characters"`
}

// marshalFile is the shape Marshal writes. It holds Character rather than
// characterFile because the resolved type already carries the right tags and
// every field is known to be present.
type marshalFile struct {
	Characters []Character `json:"characters"`
}

// Deps are the books a character's declarations are checked against, plus the
// budget its stats have to fit inside.
type Deps struct {
	Origins    *OriginBook
	Archetypes *ArchetypeBook
	Skills     *skill.Book
	Chart      *element.Chart
	Limits     progression.Limits
	Rules      combat.Rules
}

func (d Deps) validate() error {
	switch {
	case d.Origins == nil:
		return fmt.Errorf("characters cannot be validated without the origin book")
	case d.Archetypes == nil:
		return fmt.Errorf("characters cannot be validated without the archetype book")
	case d.Skills == nil:
		return fmt.Errorf("characters cannot be validated without the skill book")
	case d.Chart == nil:
		return fmt.Errorf("characters cannot be validated without the element chart")
	}
	if err := d.Limits.Validate(); err != nil {
		return err
	}
	return d.Rules.Validate()
}

// ParseBook reads a cast declaration and checks every name it uses. It never
// touches the filesystem.
//
// An empty cast is allowed: a project that has not authored anyone yet is a
// starting point, not a data error. That is the one place this differs from
// skill.ParseBook, which rejects an empty book because a game with no skills
// cannot run.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode cast book: %w", err)
	}
	book := &Book{byID: make(map[string]Character, len(file.Characters))}
	for _, declared := range file.Characters {
		built, err := resolveCharacter(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("character %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.characters = append(book.characters, built)
	}
	if err := checkCharacterRestrictions(book, deps.Skills); err != nil {
		return nil, err
	}
	return book, nil
}

// checkCharacterRestrictions checks the names in a carried skill's character
// allowlist, once every character has been read.
//
// It runs after the loop rather than inside it because the allowlist points at
// the very book being parsed: a skill restricted to a character declared
// further down the file is ordinary authoring, and a check made while the book
// was half-read would refuse it for being in the wrong place.
//
// Only the skills somebody carries are checked, and that is deliberate rather
// than lazy. Checking every skill in the book would deadlock authoring: a
// unique skill cannot be written before the character it names, and that
// character cannot be written before the skill its kit names. Restricting the
// check to carried skills breaks the cycle — the skill goes in first, carried by
// nobody, and is checked the moment the character that carries it exists.
func checkCharacterRestrictions(book *Book, skills *skill.Book) error {
	for _, character := range book.characters {
		for _, id := range character.Skills {
			carried, err := skills.Lookup(id)
			if err != nil {
				return err
			}
			if carried.Restrict == nil {
				continue
			}
			for _, named := range carried.Restrict.Characters {
				if _, known := book.byID[named]; !known {
					return fmt.Errorf(
						"character %q carries %q, which is restricted to the character %q, and the cast holds nobody with that id",
						character.ID, carried.ID, named)
				}
			}
		}
	}
	return nil
}

func resolveCharacter(declared characterFile, deps Deps) (Character, error) {
	if err := ValidateID(declared.ID); err != nil {
		return Character{}, err
	}
	fail := func(format string, args ...any) (Character, error) {
		return Character{}, fmt.Errorf("character %q: "+format, append([]any{declared.ID}, args...)...)
	}

	if strings.TrimSpace(declared.Name) == "" {
		return fail("has no display name")
	}
	if _, known := deps.Origins.Get(declared.Origin); !known {
		return fail("comes from the unknown origin %q", declared.Origin)
	}
	if _, known := deps.Archetypes.Get(declared.Archetype); !known {
		return fail("was tuned from the unknown archetype %q", declared.Archetype)
	}
	if err := ValidateImagePath(declared.Image); err != nil {
		return fail("%w", err)
	}
	if declared.Element == nil {
		return fail("does not declare an element")
	}
	if err := deps.Chart.ValidateAffinity(*declared.Element); err != nil {
		return fail("%w", err)
	}
	if err := declared.Stages.Validate(deps.Limits, deps.Rules); err != nil {
		return fail("%w", err)
	}
	kit, err := resolveSkills(declared.Skills, deps.Skills)
	if err != nil {
		return fail("%w", err)
	}
	// The element half of the rule lives in skill.WhyCannotCarry, which
	// battle.enlist calls too. Applying it here is what makes a character fail
	// where it is authored rather than at the moment somebody tries to put it
	// in a battle — an author has no reason to know the engine has an opinion
	// about this.
	//
	// The other two halves of a restriction are enforced only here, because the
	// engine has neither an archetype nor a character identity to check them
	// against. Each refusal names the skill and what the restriction allows, so
	// that somebody who did not write the restriction can act on it without
	// opening skills.json.
	for _, carried := range kit {
		switch skill.WhyCannotCarry(*declared.Element, carried) {
		case skill.CarryWrongElement:
			return fail("is %s and cannot carry %q, which is %s",
				*declared.Element, carried.ID, carried.Element)
		case skill.CarryElementRestricted:
			return fail("is %s and cannot carry %q, which only %s may carry",
				*declared.Element, carried.ID,
				strings.Join(carried.Restrict.ElementNames(), " or "))
		}
		if !carried.Restrict.AllowsArchetype(declared.Archetype) {
			return fail("was tuned from %q and cannot carry %q, which only the %s archetype may carry",
				declared.Archetype, carried.ID, strings.Join(carried.Restrict.Archetypes, " or "))
		}
		if !carried.Restrict.AllowsCharacter(declared.ID) {
			return fail("cannot carry %q, which only %s may carry",
				carried.ID, strings.Join(carried.Restrict.Characters, " or "))
		}
	}

	return Character{
		ID: declared.ID, Name: declared.Name,
		Origin: declared.Origin, Archetype: declared.Archetype,
		Image: declared.Image, Element: *declared.Element, Bio: declared.Bio,
		Stages: declared.Stages, Skills: skillIDs(kit),
	}, nil
}

// resolveSkills checks a kit against the skill book and hands back the resolved
// skills rather than their ids.
//
// Returning the skills is what lets a character apply skill.CanCarry without a
// second lookup, and lets a preset derive what its kit demands. It is shared
// with the archetype presets, so a preset and a character complain about the
// same thing in the same words.
func resolveSkills(declared []string, skills *skill.Book) ([]skill.Skill, error) {
	if len(declared) == 0 {
		return nil, fmt.Errorf("knows no skills, so it would have nothing to do on its turn")
	}
	out := make([]skill.Skill, 0, len(declared))
	seen := make(map[string]bool, len(declared))
	for _, id := range declared {
		known, err := skills.Lookup(id)
		if err != nil {
			return nil, err
		}
		if seen[known.ID] {
			return nil, fmt.Errorf("knows %q twice", known.ID)
		}
		seen[known.ID] = true
		out = append(out, known)
	}
	return out, nil
}

func skillIDs(kit []skill.Skill) []string {
	out := make([]string, 0, len(kit))
	for _, carried := range kit {
		out = append(out, carried.ID)
	}
	return out
}

// Get returns a character by id.
func (b *Book) Get(id string) (Character, bool) {
	found, ok := b.byID[id]
	if !ok {
		return Character{}, false
	}
	return found.clone(), true
}

// All returns every character in declaration order.
func (b *Book) All() []Character {
	out := make([]Character, 0, len(b.characters))
	for _, entry := range b.characters {
		out = append(out, entry.clone())
	}
	return out
}

// OfOrigin returns every character borrowed from one work, in declaration
// order.
func (b *Book) OfOrigin(id string) []Character {
	out := make([]Character, 0, len(b.characters))
	for _, entry := range b.characters {
		if entry.Origin == id {
			out = append(out, entry.clone())
		}
	}
	return out
}

// Marshal writes the book as a data file: two-space indented JSON with the
// characters sorted by id.
//
// This is the one place in the package that imposes an order instead of
// preserving the authored one, and it is deliberate. An authoring tool that
// adds a character rewrites the whole file, and a stable order is what makes
// that rewrite a one-line diff rather than a reshuffle nobody can review.
func (b *Book) Marshal() ([]byte, error) {
	sorted := b.All()
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out, err := json.MarshalIndent(marshalFile{Characters: sorted}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode cast book: %w", err)
	}
	return append(out, '\n'), nil
}

// Append returns a new book holding the existing characters plus the extra
// ones, validated exactly as a parse would validate them.
//
// It works by marshalling and re-parsing rather than by re-implementing the
// checks, which is what guarantees the bytes an authoring tool is about to
// write are bytes that load.
func (b *Book) Append(deps Deps, extra ...Character) (*Book, error) {
	combined := marshalFile{Characters: append(b.All(), extra...)}
	raw, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("encode cast book: %w", err)
	}
	return ParseBook(raw, deps)
}

// ValidateImagePath checks the shape of an authored image path. Whether the
// file exists is the tool's business, not the parser's: internal/core may not
// read the filesystem, and only the caller knows which directory the path is
// relative to.
//
// The checks use path rather than filepath on purpose. A data file is
// committed, so the same string has to mean the same thing on every platform,
// and filepath's behaviour depends on the one it is compiled for.
func ValidateImagePath(image string) error {
	if image == "" {
		return fmt.Errorf("declares no image")
	}
	if strings.ContainsRune(image, '\\') {
		return fmt.Errorf("image %q uses a backslash; author image paths with forward slashes so they mean the same thing everywhere", image)
	}
	if strings.HasPrefix(image, "/") {
		return fmt.Errorf("image %q is an absolute path; author it relative to the data directory", image)
	}
	if len(image) >= 2 && image[1] == ':' {
		return fmt.Errorf("image %q names a drive volume; author it relative to the data directory", image)
	}
	if slices.Contains(strings.Split(image, "/"), "..") {
		return fmt.Errorf("image %q climbs out of the data directory with a %q segment", image, "..")
	}
	switch strings.ToLower(path.Ext(image)) {
	case ".svg", ".png":
		return nil
	default:
		return fmt.Errorf("image %q has the extension %q, want .svg or .png", image, path.Ext(image))
	}
}

// checkSlug enforces the one identifier shape the data files use: lowercase
// letters, digits and hyphens. Anything looser and two ids that differ only in
// case or spacing would be two different characters that read as one.
func checkSlug(kind, id string) error {
	if id == "" {
		return fmt.Errorf("%s id is empty", kind)
	}
	for i := range len(id) {
		letter := id[i]
		switch {
		case letter >= 'a' && letter <= 'z':
		case letter >= '0' && letter <= '9':
		case letter == '-':
		default:
			return fmt.Errorf("%s id %q contains %q; ids use lowercase letters, digits and hyphens",
				kind, id, string(letter))
		}
	}
	return nil
}

// ValidateID checks the shape of a character id. One dot is allowed, which is
// how an id says which work it came from without two works being forced to
// invent unique names.
//
// It is exported for the same reason ValidateImagePath is: an authoring tool
// has to be able to reject an answer as it is typed, rather than at the end of
// a wizard nobody wants to fill in twice. ParseBook applies exactly this check.
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("character id is empty")
	}
	parts := strings.Split(id, ".")
	if len(parts) > 2 {
		return fmt.Errorf("character id %q has %d dots; at most one separates the origin from the name",
			id, len(parts)-1)
	}
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("character id %q has an empty half either side of its dot", id)
		}
		if err := checkSlug("character", part); err != nil {
			return err
		}
	}
	return nil
}
