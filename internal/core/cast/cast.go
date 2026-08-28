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
	"github.com/vukyn/hexarena/internal/core/passive"
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
	// Species is what the character is, each by id in the species book. Absent
	// is the ordinary case and it is a real answer rather than a gap: most
	// characters need no lineage, and a skill that asks for one refuses a unit
	// that is nothing in particular.
	//
	// A list because a unit may be several things at once, and because the
	// alternative — one species per character — would force a choice between two
	// true statements the moment a skill was written about either.
	Species []string `json:"species,omitempty"`
	// Stages is the evolution line the character's stats grow along. A
	// character with one stage is the ordinary case, not a special one.
	Stages progression.Line `json:"stages"`
	// Skills is the character's learnset: every skill it will ever know, each
	// from the level it learns it at.
	//
	// It is `Unlock`, the type the traits already use, because the two are one
	// question — declare many, unlock by progression, bring some — and a second
	// type beside it would be two vocabularies for one idea. What a unit
	// actually brings into a battle is a *choice made at placement* from the
	// entries unlocked at its level, which is a different list and lives on the
	// roster rather than here.
	Skills []Unlock `json:"skills"`
	// Passives are the traits the character holds for the whole of every battle,
	// each by id in the passive book and from the level it unlocks at. Absent is
	// the ordinary case.
	//
	// Unlike an archetype, these do reach the engine: battle.Roster carries them
	// because a passive is in force *during* a battle, where an archetype and an
	// evolution stage are both settled before one starts. That is the line the
	// roster's deliberate emptiness is drawn on — not "as little as possible",
	// but "nothing a replay does not read".
	Passives []Unlock `json:"passives,omitempty"`
}

// Resolve flattens the character at a level, as a chosen stage, into the stat
// line the battle engine works with, and reports which stage it fielded.
//
// progression.Furthest is what a caller passes when nobody is choosing — a
// screen showing this character at level 30 has no placement behind it — and it
// is the behaviour every caller had before a stage could be chosen at all.
func (c Character) Resolve(level int, stage string) (progression.Values, progression.Stage, error) {
	return c.Stages.Resolve(level, stage)
}

// StagesAt is the forms this character may be fielded as at a level, in line
// order. It is what a placement chooses from and what a refusal offers.
func (c Character) StagesAt(level int) ([]progression.Stage, error) {
	return c.Stages.Allowed(level)
}

// FurthestAt is the grown end of each of the character's lines: one form on a
// line that does not fork, and one per arm on a line that does.
//
// It exists for the callers that have to say something about "the character at
// the cap" and cannot pass progression.Furthest to do it, because a fork has no
// single furthest. A budget row is the example: the stat budget bites at the end
// of a line, and a forking character has two ends that are priced separately.
func (c Character) FurthestAt(level int) ([]progression.Stage, error) {
	return c.Stages.Furthest(level)
}

// StageArt is the picture of one of the character's forms: the stage's own art
// when it declares any, and the character's when it does not.
//
// This is the only place the fallback is decided. A caller reading
// progression.Stage.Image directly would draw nothing for the ordinary stage
// that names none, and a second caller would invent the same fallback slightly
// differently — which is how a character ends up with two pictures depending on
// which screen is asking.
func (c Character) StageArt(stage progression.Stage) string {
	if stage.Image != "" {
		return stage.Image
	}
	return c.Image
}

// ArtEntry is one picture a character can show, and which form it belongs to.
type ArtEntry struct {
	// Stage is the form's name, empty for the character's own picture.
	Stage string
	Image string
}

// Art is every distinct picture the character can show: its own first, then one
// per stage that names a different one.
//
// A checker wants this rather than the single Image, because art that only a
// grown form uses is exactly the art nobody looks at until the character has
// grown — so a missing file there is the one that surfaces late. Distinct by
// path and in declaration order, so a stage sharing the character's picture adds
// no row and the result never depends on map order.
func (c Character) Art() []ArtEntry {
	out := make([]ArtEntry, 0, len(c.Stages)+1)
	out = append(out, ArtEntry{Image: c.Image})
	for _, stage := range c.Stages {
		if stage.Image == "" {
			continue
		}
		if slices.ContainsFunc(out, func(seen ArtEntry) bool { return seen.Image == stage.Image }) {
			continue
		}
		out = append(out, ArtEntry{Stage: stage.Name, Image: stage.Image})
	}
	return out
}

// PassivesAt is the traits the character holds at a level.
//
// It is separate from Resolve rather than a fourth return value: a stat line and
// a stage answer "what is this unit", while this answers "what does it bring",
// and the second is the question a placement asks. Keeping them apart is also
// what lets the kit join it later without Resolve growing a fifth return.
func (c Character) PassivesAt(level int, stage string) []string {
	return UnlockedIDs(c.Passives, level, c.form(level, stage))
}

// form is the stage name a level-and-choice resolves to, so that the two "what
// does it have" questions do not each have to work it out — and cannot work it
// out differently.
//
// A choice that does not name a form means the furthest one, which is what every
// placement meant before it could choose. A name the line does not answer to
// resolves to itself and simply matches nothing, because this is a reader rather
// than a validator: resolveLoadout is where a bad stage is refused, once, with a
// message that can offer the alternatives.
func (c Character) form(level int, stage string) string {
	if stage != progression.Furthest {
		return stage
	}
	reached, err := c.Stages.StageAt(level)
	if err != nil {
		return stage
	}
	return reached.Name
}

// SkillSlots and TraitSlots are how much a placement may bring of what a
// character knows.
//
// Four and one. They sit beside the learnset rather than beside the engine
// because they are a rule about learning: battle.Roster keeps taking a resolved
// kit exactly as it keeps taking a resolved stat line, and which four of the nine
// are brought is settled before a battle the way an evolution already is. They
// sit here rather than in internal/seed because two callers now read them — the
// roster that fields a placement and the spar that measures one — and a second
// copy of "four" is how a measurement quietly stops measuring what gets fielded.
//
// Four is a per-turn nerf whose size is set by cooldowns: a skill on cooldown N
// contributes 1/(N+1) actions per turn, so a level-one unit holding a cooldown-1
// and a cooldown-4 skill idles about a third of its turns. That is what being
// young should feel like, and it is also why a low-cooldown basic is close to
// mandatory in a four-slot world.
const (
	SkillSlots = 4
	TraitSlots = 1
)

// SkillsAt is the skills the character has learned by a level, in declaration
// order. It is the list a placement may choose its loadout from, and the same
// function the traits use — one "what is available at level N" for both lists,
// which is the whole of "skills and traits are one mechanism".
func (c Character) SkillsAt(level int, stage string) []string {
	return UnlockedIDs(c.Skills, level, c.form(level, stage))
}

// clone copies the slices a caller could otherwise mutate through.
func (c Character) clone() Character {
	out := c
	out.Stages = make(progression.Line, len(c.Stages))
	copy(out.Stages, c.Stages)
	out.Skills = slices.Clone(c.Skills)
	out.Species = slices.Clone(c.Species)
	out.Passives = slices.Clone(c.Passives)
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
	Element  *element.Affinity `json:"element"`
	Bio      string            `json:"bio"`
	Species  []string          `json:"species"`
	Stages   progression.Line  `json:"stages"`
	Skills   []Unlock          `json:"skills"`
	Passives []Unlock          `json:"passives"`
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
	// Passives is the book a character's traits are checked against, and it is
	// wanted only by a character that names one — see resolvePassives.
	Passives *passive.Book
	// Species is the book a character's kinds are checked against, on the same
	// terms as Passives: wanted only by a character that claims one, or by one
	// carrying a skill that asks for one.
	Species *SpeciesBook
	Chart   *element.Chart
	Limits  progression.Limits
	Rules   combat.Rules
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
		for _, entry := range character.Skills {
			carried, err := skills.Lookup(entry.ID)
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
	// A stage's art is optional and its absence is the answer "show the
	// character's", so only a stage that names one is checked. The check is here
	// rather than in Line.Validate for the same reason a skill's pattern name is
	// checked by whoever holds the pattern book: this package owns what an image
	// path may look like, and progression does not.
	for _, stage := range declared.Stages {
		if stage.Image == "" {
			continue
		}
		if err := ValidateImagePath(stage.Image); err != nil {
			return fail("stage %q: %w", stage.Name, err)
		}
	}
	species, err := resolveCharacterSpecies(declared.ID, declared.Species, deps.Species)
	if err != nil {
		return Character{}, err
	}
	learnset, kit, err := resolveLearnset(declared.ID, declared.Skills, deps.Skills, declared.Stages)
	if err != nil {
		return fail("%w", err)
	}
	// The element half of the rule lives in skill.WhyCannotCarry, which
	// battle.enlist calls too. Applying it here is what makes a character fail
	// where it is authored rather than at the moment somebody tries to put it
	// in a battle — an author has no reason to know the engine has an opinion
	// about this.
	//
	// The other four are enforced only here, because the engine has none of an
	// archetype, a character identity, a species or an origin to check them
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
		// A restriction naming a species nobody declared is a typo, and left
		// unchecked it reads as "nobody may carry this" — which is the same
		// silence a present-but-empty allowlist was refused for.
		for _, named := range carried.Restrict.SpeciesNames() {
			if deps.Species == nil {
				return fail("carries %q, which is kept for a species, and that cannot be checked without the species book",
					carried.ID)
			}
			if _, known := deps.Species.Get(named); !known {
				return fail("carries %q, which is kept for the unknown species %q", carried.ID, named)
			}
		}
		if !carried.Restrict.AllowsSpecies(species) {
			return fail("cannot carry %q, which only a %s may carry",
				carried.ID, strings.Join(carried.Restrict.SpeciesNames(), " or "))
		}
		// An unknown origin is refused for the reason an unknown species is: the
		// gate would admit nobody, and an allowlist nobody satisfies is silent
		// about being a typo. The origin book is not optional the way the
		// species book is, so this needs no "without the book" case.
		for _, named := range carried.Restrict.OriginNames() {
			if _, known := deps.Origins.Get(named); !known {
				return fail("carries %q, which is kept for the unknown origin %q", carried.ID, named)
			}
		}
		if !carried.Restrict.AllowsOrigin(declared.Origin) {
			return fail("was borrowed from %q and cannot carry %q, which only somebody from %s may carry",
				declared.Origin, carried.ID, strings.Join(carried.Restrict.OriginNames(), " or "))
		}
	}

	passives, err := resolvePassives("character", declared.ID, declared.Passives, deps.Passives, declared.Stages)
	if err != nil {
		return Character{}, err
	}

	return Character{
		ID: declared.ID, Name: declared.Name,
		Origin: declared.Origin, Archetype: declared.Archetype,
		Image: declared.Image, Element: *declared.Element, Bio: declared.Bio,
		Species: species, Stages: declared.Stages, Skills: learnset,
		Passives: passives,
	}, nil
}

// resolveCharacterSpecies checks what a character claims to be against the
// species book.
//
// Unlike a carried skill's *character* allowlist, this is checked inside the
// parse loop rather than after it, and there is no deadlock to break: the species
// book is a separate file that names nothing, so a species can always be written
// before whatever claims it. The character allowlist is deferred only because it
// points at the very book being read — see checkCharacterRestrictions.
func resolveCharacterSpecies(owner string, declared []string, book *SpeciesBook) ([]string, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	if book == nil {
		return nil, fmt.Errorf("character %q names species, which cannot be checked without the species book", owner)
	}
	out := make([]string, 0, len(declared))
	for _, id := range declared {
		found, known := book.Get(id)
		if !known {
			return nil, fmt.Errorf("character %q is the unknown species %q; the ones there are: %s",
				owner, id, strings.Join(book.IDs(), " "))
		}
		if slices.Contains(out, found.ID) {
			return nil, fmt.Errorf("character %q is the species %q twice", owner, found.ID)
		}
		out = append(out, found.ID)
	}
	return out, nil
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

// resolveLearnset checks a character's learnset and hands back both the entries
// and the skills they name.
//
// Two returns because two callers want different halves of one walk: the
// character keeps the entries, because a level is what the placement will ask
// about, and everything that checks a *restriction* wants the resolved skills,
// because a restriction is a property of the skill rather than of when it is
// learned. Walking twice would be two chances to disagree about which ids are in
// the kit at all.
//
// The level rules are the ones a trait already obeys, read from the same place:
// unstated is one, and a level outside the cap is a refusal rather than a clamp.
// A clamp would silently move a skill an author put at 61 to the cap and leave
// them reading a learnset that does not say what they wrote.
func resolveLearnset(owner string, declared []Unlock, skills *skill.Book, line progression.Line) ([]Unlock, []skill.Skill, error) {
	if len(declared) == 0 {
		return nil, nil, fmt.Errorf("knows no skills, so it would have nothing to do on its turn")
	}
	entries := make([]Unlock, 0, len(declared))
	kit := make([]skill.Skill, 0, len(declared))
	for _, entry := range declared {
		known, err := skills.Lookup(entry.ID)
		if err != nil {
			return nil, nil, err
		}
		if slices.ContainsFunc(entries, func(seen Unlock) bool { return seen.ID == known.ID }) {
			return nil, nil, fmt.Errorf("knows %q twice", known.ID)
		}
		level := entry.AtLevel
		if level == 0 {
			level = 1
		}
		if level < 1 || level > progression.LevelCap {
			return nil, nil, fmt.Errorf("learns %q at level %d, outside 1..%d",
				known.ID, entry.AtLevel, progression.LevelCap)
		}
		if err := checkStages("character", owner, known.ID, entry.Stages, line); err != nil {
			return nil, nil, err
		}
		entries = append(entries, Unlock{ID: known.ID, AtLevel: level, Stages: slices.Clone(entry.Stages)})
		kit = append(kit, known)
	}
	// A character that learns nothing until later has nothing to bring at level
	// one, and a placement there would be a unit that cannot act. The refusal is
	// here rather than at placement because it is a fact about the character:
	// every other level would be legal and the author would only find out by
	// fielding it.
	// The first form at level one is what a character starts as, so that is the
	// combination this insists on: an entry kept for a later form is no use to a
	// unit that has not grown into it, and one gated on a later level is no use
	// either.
	first := progression.Furthest
	if len(line) > 0 {
		first = line[0].Name
	}
	if !slices.ContainsFunc(entries, func(entry Unlock) bool { return entry.Available(1, first) }) {
		return nil, nil, fmt.Errorf(
			"learns nothing it could use at level 1 as %s, so it would start with nothing to do", first)
	}
	return entries, kit, nil
}

// Learn turns a plain list of ids into learnset entries every one of which is
// known from level one.
//
// It is what a preset's kit becomes when a character is built from it: an
// archetype has no level to gate against, so the suggestion it carries is
// "knows all of this", and somebody editing cast.json afterwards is who decides
// otherwise. Exported because the authoring tool is the caller and a second
// copy of this loop there would be the place the two spellings drift.
func Learn(ids []string) []Unlock {
	out := make([]Unlock, 0, len(ids))
	for _, id := range ids {
		out = append(out, Unlock{ID: id, AtLevel: 1})
	}
	return out
}

// LearnedIDs is every id in a learnset regardless of level, in declaration
// order.
//
// It answers "what does this character ever know", which is a different
// question from SkillsAt's "what does it know now" — and the one every check
// about a *restriction* wants, because a restriction is a property of the skill
// rather than of when it is learned.
func LearnedIDs(entries []Unlock) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.ID)
	}
	return out
}

// checkStages is the rule both lists obey about a stage allowlist, said once.
//
// A name the line does not answer to is a typo, and an unchecked one reads as
// "nobody may hold this" — the same silence a present-but-empty allowlist is
// refused for everywhere else in this repository. The line is passed in rather
// than read off a character because the presets share this function and have no
// line at all: for them the answer is that a stage gate is meaningless, which is
// a different refusal and a clearer one.
func checkStages(kind, owner, what string, stages []string, line progression.Line) error {
	if len(stages) == 0 {
		return nil
	}
	if len(line) == 0 {
		return fmt.Errorf("%s %q keeps %q for the stage %q, and it has no evolution line to have stages of",
			kind, owner, what, stages[0])
	}
	named := progression.StageNames(line)
	seen := make([]string, 0, len(stages))
	for _, stage := range stages {
		if !slices.Contains(named, stage) {
			return fmt.Errorf("%s %q keeps %q for the stage %q, and its forms are %v",
				kind, owner, what, stage, named)
		}
		if slices.Contains(seen, stage) {
			return fmt.Errorf("%s %q keeps %q for the stage %q twice", kind, owner, what, stage)
		}
		seen = append(seen, stage)
	}
	// An allowlist naming every form is the same as naming none, and the two
	// spellings would render differently while meaning the same thing.
	if len(seen) == len(named) {
		return fmt.Errorf("%s %q keeps %q for every one of its forms, which is what naming none already means",
			kind, owner, what)
	}
	return nil
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

// OfSpecies returns every character that is one kind of creature, in
// declaration order.
//
// A character may be several things at once, so the same character is returned
// by more than one call, which is the difference from OfOrigin: a work is a
// partition of the cast and a species is not.
func (b *Book) OfSpecies(id string) []Character {
	out := make([]Character, 0, len(b.characters))
	for _, entry := range b.characters {
		if slices.Contains(entry.Species, id) {
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

// resolvePassives checks a list of trait ids against the passive book, and is
// shared by a character and by an archetype preset because both name traits the
// same way and a second copy of these four rules is a second thing to drift.
//
// A list that is absent is no traits, which is the ordinary case. A list that
// names something needs the book, so a book that was not handed over is a
// refusal rather than a pass: the alternative is a data file whose traits are
// checked or unchecked depending on how the caller wired itself up.
func resolvePassives(kind, owner string, declared []Unlock, book *passive.Book, line progression.Line) ([]Unlock, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	if book == nil {
		return nil, fmt.Errorf("%s %q names passives, which cannot be checked without the passive book", kind, owner)
	}
	out := make([]Unlock, 0, len(declared))
	for _, entry := range declared {
		found, err := book.Lookup(entry.ID)
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", kind, owner, err)
		}
		if slices.ContainsFunc(out, func(seen Unlock) bool { return seen.ID == found.ID }) {
			return nil, fmt.Errorf("%s %q names the passive %q twice", kind, owner, found.ID)
		}
		// An unstated level is one, the way an unstated strike count is one.
		level := entry.AtLevel
		if level == 0 {
			level = 1
		}
		if level < 1 || level > progression.LevelCap {
			return nil, fmt.Errorf("%s %q unlocks %q at level %d, outside 1..%d",
				kind, owner, found.ID, entry.AtLevel, progression.LevelCap)
		}
		if err := checkStages(kind, owner, found.ID, entry.Stages, line); err != nil {
			return nil, err
		}
		out = append(out, Unlock{ID: found.ID, AtLevel: level, Stages: slices.Clone(entry.Stages)})
	}
	return out, nil
}

// Unlock is something a character has from a level onwards.
//
// It is the shape a *learnset* is written in, and it is deliberately about an id
// rather than about a trait: the kit is the same question — declare many, unlock
// by progression, bring some — so when skills gain their levels they gain this
// type rather than a second one beside it. Two vocabularies for one idea is the
// mistake this repository keeps a list of.
//
// # Two gates, and why the second one is a list
//
// A level gate says *when*; a stage gate says *which form*. They were one fact
// until a placement could choose its stage, because the stage was derived from
// the level — `at_stage: "Ivysaur"` was exactly `at_level: 16` and nothing else.
// A placement now names the form it fielded, so the two are different questions
// and both can be asked.
//
// The stage gate is an **allowlist** rather than a threshold, and that is the
// whole reason evolving is a decision rather than a formality. A threshold could
// only ever say "from this form onwards", so everything an early form knew a
// grown one knew too, and choosing not to evolve would be choosing a weaker unit
// for nothing. A list can say "Bulbasaur only" — something the grown form never
// gets — which is what gives up an evolution in exchange for keeping a move.
//
// It is the same shape `skill.Restriction` uses for elements and archetypes, for
// the same reason: an allowlist can name one member of a class, and a threshold
// cannot.
type Unlock struct {
	ID string `json:"id"`
	// AtLevel is the first level the holder has it at. An unstated level is one:
	// the common case is a trait a character has always had, and writing
	// "at_level": 1 on every entry would be noise on the line that matters least.
	AtLevel int `json:"at_level,omitempty"`
	// Stages are the forms that may hold it, by stage name. Empty is every form,
	// which is the ordinary case and the one that writes nothing.
	Stages []string `json:"stages,omitempty"`
}

// Unlocked reports whether the entry's *level* gate is passed.
//
// It answers half the question and is kept because half is what several callers
// want: an authoring screen printing "endurance@16" is describing the entry
// rather than a placement, and it has no form in front of it to ask about.
func (u Unlock) Unlocked(level int) bool { return level >= u.AtLevel }

// Held reports whether the named form may hold it.
//
// An empty allowlist is every form. The form is matched by name because that is
// what a placement chooses by and what the data spells — a stage has no id of
// its own, and inventing one would be a second name for the thing already named.
func (u Unlock) Held(stage string) bool {
	if len(u.Stages) == 0 {
		return true
	}
	return slices.Contains(u.Stages, stage)
}

// Available reports whether a unit of this level, fielded as this form, has it.
// Both gates, asked together, because a caller that asked only one would be
// wrong in whichever direction it forgot.
func (u Unlock) Available(level int, stage string) bool {
	return u.Unlocked(level) && u.Held(stage)
}

// unlockFile is the shape an entry is written in, and therefore the shape it is
// read in — the same arrangement skill.Skill has, so the writer cannot describe
// a field the parser does not read.
type unlockFile struct {
	ID      string   `json:"id"`
	AtLevel int      `json:"at_level,omitempty"`
	Stages  []string `json:"stages,omitempty"`
}

// MarshalJSON writes the entry as the declaration a parse would read back, and
// omits a gate of one.
//
// The field cannot simply carry `omitempty`, because a parse *normalises* an
// unstated level to one: there is exactly one value in memory meaning "from the
// start", which is what lets every caller ask Unlocked without first asking
// which of two spellings it is holding. The cost of that is this method — the
// writer has to know that one is the value not worth writing.
func (u Unlock) MarshalJSON() ([]byte, error) {
	// The allowlist has no normalisation to undo: an empty one means every form
	// and is simply not written, which is what omitempty already does. It is
	// listed here rather than left out because a writer that does not name a
	// field is a writer that silently drops it — and this one is invisible until
	// somebody fields a unit and finds a skill missing.
	out := unlockFile{ID: u.ID, Stages: u.Stages}
	if u.AtLevel > 1 {
		out.AtLevel = u.AtLevel
	}
	return json.Marshal(out)
}

// UnlockedIDs is the ids a unit of this level, fielded as this form, has — in
// declaration order.
//
// One function rather than one per list, and it takes the list rather than
// reading a character, so the kit and the traits share it. Declaration order
// because the result reaches a roster entry and then an event log: an order a
// map decided would stop a battle replaying.
func UnlockedIDs(entries []Unlock, level int, stage string) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Available(level, stage) {
			out = append(out, entry.ID)
		}
	}
	return out
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
