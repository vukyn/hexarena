package forge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// DefaultDataDir is where the game's own data lives, relative to the module
// root. Every front-end takes a flag for it so a scratch copy can be edited
// without touching the shipped files.
const DefaultDataDir = "internal/seed/data"

// The data files an author's tooling reads. They are named here rather than
// inline so a rename shows up in one place.
const (
	elementsFile   = "elements.json"
	combatFile     = "combat.json"
	limitsFile     = "progression.json"
	modifiersFile  = "modifiers.json"
	patternsFile   = "patterns.json"
	statusesFile   = "statuses.json"
	passivesFile   = "passives.json"
	skillsFile     = "skills.json"
	originsFile    = "origins.json"
	speciesFile    = "species.json"
	archetypesFile = "archetypes.json"
	castFile       = "cast.json"
	buildsFile     = "builds.json"
	squadsFile     = "squads.json"
)

// emptyCatalogue is what a data directory with no build catalogue reads as.
//
// A working directory is a copy somebody made, and every copy taken before
// builds.json existed has everything else in it — so a missing catalogue is an
// older data directory rather than a broken one, and refusing to load would
// break the tool over a file that has nothing to do with the character being
// authored. It goes through the parser rather than round it: an empty catalogue
// is one the parser has agreed to, not one this file assembled by hand.
var emptyCatalogue = []byte(`{"builds": []}`)

// emptySquads is what a data directory with nobody's squads in it reads as.
var emptySquads = []byte(`{"squads": []}`)

// assetsDir is the folder inside a data directory that art lives under.
//
// It is named once, here, because two questions read it: where a character's
// art is suggested to live, which SuggestedImage answers, and what art there is
// to choose from, which ArtFiles answers. Two spellings of one folder name is
// how a picker ends up offering paths a suggestion never proposes.
const assetsDir = "assets"

// Library is every book a character is validated against, loaded from one data
// directory.
//
// The directory rather than the embedded copy is the point: an author edits
// files, and a tool that validated the baked-in bytes would keep saying yes to
// a change it had not read. The other side of that is the note a check prints —
// the game boots from the embedded copy, so an edit needs a rebuild before it
// reaches a battle.
type Library struct {
	dir    string
	rules  combat.Rules
	chart  *element.Chart
	limits progression.Limits
	// bounds is what a buff approaches and what a debuff sinks towards. Nothing
	// an author writes is checked against it -- it is here because a battle needs
	// it, and a battle is now something this package can start.
	bounds modifier.Bounds
	// patterns and statuses are held rather than discarded after the skill book
	// is built, because a skill is now authored here too: the shapes and the
	// statuses are what a skill's declarations are checked against, and writing
	// one back means re-parsing the book through the same two.
	patterns *pattern.Book
	statuses *status.Book
	// passives is held for the same reason: a character names traits, and the
	// book is what those names are checked against on every write.
	passives   *passive.Book
	skills     *skill.Book
	origins    *cast.OriginBook
	species    *cast.SpeciesBook
	archetypes *cast.ArchetypeBook
	characters *cast.Book
	// builds is the authored late-game loadouts, which is the one book that is
	// read *through* another: a build is checked against the cast, so it is
	// parsed last and cannot be held without the characters beside it.
	builds *cast.BuildBook
	// squads is the sides an author has built to fight each other. It is the one
	// book this package *writes for its own sake*: every other file here is the
	// game's, and a squad is the tool's — which is why it is held as a plain
	// slice rather than a book, and why nothing outside a squad refers to one.
	squads []placement.Squad
}

// Load reads every book from a data directory.
func Load(dir string) (*Library, error) {
	if dir == "" {
		return nil, fmt.Errorf("no data directory given")
	}
	read := func(name string) ([]byte, error) {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Join(dir, name), err)
		}
		return raw, nil
	}
	lib := &Library{dir: dir}

	raw, err := read(combatFile)
	if err != nil {
		return nil, err
	}
	if lib.rules, err = combat.ParseRules(raw); err != nil {
		return nil, err
	}

	if raw, err = read(elementsFile); err != nil {
		return nil, err
	}
	if lib.chart, err = element.ParseChart(raw); err != nil {
		return nil, err
	}

	if raw, err = read(limitsFile); err != nil {
		return nil, err
	}
	if lib.limits, err = progression.ParseLimits(raw); err != nil {
		return nil, err
	}

	if raw, err = read(modifiersFile); err != nil {
		return nil, err
	}
	if lib.bounds, err = modifier.ParseBounds(raw); err != nil {
		return nil, err
	}

	if raw, err = read(patternsFile); err != nil {
		return nil, err
	}
	if lib.patterns, err = pattern.ParseBook(raw); err != nil {
		return nil, err
	}
	if raw, err = read(statusesFile); err != nil {
		return nil, err
	}
	if lib.statuses, err = status.ParseBook(raw); err != nil {
		return nil, err
	}
	if raw, err = read(passivesFile); err != nil {
		return nil, err
	}
	if lib.passives, err = passive.ParseBook(raw, passive.Deps{Statuses: lib.statuses}); err != nil {
		return nil, err
	}
	if raw, err = read(skillsFile); err != nil {
		return nil, err
	}
	if lib.skills, err = skill.ParseBook(raw, lib.SkillDeps()); err != nil {
		return nil, err
	}

	if raw, err = read(originsFile); err != nil {
		return nil, err
	}
	if lib.origins, err = cast.ParseOrigins(raw); err != nil {
		return nil, err
	}

	if raw, err = read(speciesFile); err != nil {
		return nil, err
	}
	if lib.species, err = cast.ParseSpecies(raw); err != nil {
		return nil, err
	}

	if raw, err = read(archetypesFile); err != nil {
		return nil, err
	}
	lib.archetypes, err = cast.ParseArchetypes(raw, lib.ArchetypeDeps())
	if err != nil {
		return nil, err
	}

	if raw, err = read(castFile); err != nil {
		return nil, err
	}
	if lib.characters, err = cast.ParseBook(raw, lib.CastDeps()); err != nil {
		return nil, err
	}

	// The catalogue is optional and only this one is, which is why the absence is
	// spelled out here rather than made a property of read: every other book is a
	// rule the game cannot start without, while a build is a decision somebody
	// wrote down about a character that is already legal without it.
	raw, err = read(buildsFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		raw = emptyCatalogue
	case err != nil:
		return nil, err
	}
	if lib.builds, err = cast.ParseBuilds(raw, lib.characters); err != nil {
		return nil, err
	}

	// Squads are optional on the same terms and for a stronger reason: the file
	// ships empty, so a data directory that has never had one built in it is the
	// ordinary case rather than an old copy.
	raw, err = read(squadsFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		raw = emptySquads
	case err != nil:
		return nil, err
	}
	if lib.squads, err = placement.Parse(raw); err != nil {
		return nil, err
	}
	return lib, nil
}

// Dir is the directory the books were read from and will be written back to.
func (l *Library) Dir() string { return l.dir }

// Rules, Limits, Chart, Skills, Origins, Archetypes and Characters hand out the
// loaded books. They are read-only from a caller's side: the book types copy
// the slices a caller could otherwise mutate through.
func (l *Library) Rules() combat.Rules             { return l.rules }
func (l *Library) Limits() progression.Limits      { return l.limits }
func (l *Library) Bounds() modifier.Bounds         { return l.bounds }
func (l *Library) Chart() *element.Chart           { return l.chart }
func (l *Library) Patterns() *pattern.Book         { return l.patterns }
func (l *Library) Statuses() *status.Book          { return l.statuses }
func (l *Library) Skills() *skill.Book             { return l.skills }
func (l *Library) Passives() *passive.Book         { return l.passives }
func (l *Library) Origins() *cast.OriginBook       { return l.origins }
func (l *Library) Species() *cast.SpeciesBook      { return l.species }
func (l *Library) Archetypes() *cast.ArchetypeBook { return l.archetypes }
func (l *Library) Characters() *cast.Book          { return l.characters }

// Builds is every authored build, in declaration order.
//
// It hands out the slice rather than the book, unlike every accessor above it: a
// front-end listing builds wants them in order and a front-end asking about one
// character wants BuildsOf, and those two are the whole of what the book is read
// for here. Handing the book over would give a screen a third way to ask — by a
// build's own id — which is the lookup nothing on a screen has, because a screen
// has the row it drew.
func (l *Library) Builds() []cast.Build { return l.builds.All() }

// BuildsOf is one character's builds, in declaration order, and is empty for a
// character nobody has written a direction for.
//
// Empty is an answer rather than a miss, which is a fact a caller has to carry:
// a listing that dropped a character with no build would be saying the cast is
// shorter than it is.
func (l *Library) BuildsOf(character string) []cast.Build { return l.builds.Of(character) }

// SkillDeps is what a skill is checked against: the shapes it can cover and the
// statuses it can inflict.
func (l *Library) SkillDeps() skill.Deps {
	return skill.Deps{Patterns: l.patterns, Statuses: l.statuses}
}

// CastDeps is what a character is checked against, assembled from the loaded
// books so every tool and the game itself apply the same rules.
// ArchetypeDeps is everything a preset is validated against, and it exists for
// the same reason CastDeps does: two callers parse the archetype book — a load,
// and the re-parse a skill edit does off the disk — and a second copy of the
// list is a second thing to forget a book from. That is not hypothetical: when
// passives arrived the re-parse kept its own list, so every skill edit in the
// repository started failing with a refusal about a preset nobody had touched.
func (l *Library) ArchetypeDeps() cast.ArchetypeDeps {
	return cast.ArchetypeDeps{
		Skills: l.skills, Passives: l.passives, Limits: l.limits, Rules: l.rules,
	}
}

func (l *Library) CastDeps() cast.Deps {
	return cast.Deps{
		Origins: l.origins, Species: l.species, Archetypes: l.archetypes,
		Skills: l.skills, Passives: l.passives,
		Chart: l.chart, Limits: l.limits, Rules: l.rules,
	}
}

// CastPath is the file a saved character lands in, which is what a
// confirmation has to name.
func (l *Library) CastPath() string { return filepath.Join(l.dir, castFile) }

// OriginsPath is the file a saved origin lands in.
func (l *Library) OriginsPath() string { return filepath.Join(l.dir, originsFile) }

// SpeciesPath is the file a saved species lands in.
func (l *Library) SpeciesPath() string { return filepath.Join(l.dir, speciesFile) }

// SkillsPath is the file a saved skill lands in.
func (l *Library) SkillsPath() string { return filepath.Join(l.dir, skillsFile) }

// ImagePath turns an authored image path into a real one. Authored paths are
// always slash-separated and relative to the data directory; only here does
// either of those become an operating system's business.
func (l *Library) ImagePath(image string) string {
	return filepath.Join(l.dir, filepath.FromSlash(path.Clean(image)))
}

// ImageExists reports whether the art a character names is really there. This
// is the question internal/core is not allowed to ask.
func (l *Library) ImageExists(image string) bool {
	info, err := os.Stat(l.ImagePath(image))
	return err == nil && info.Mode().IsRegular()
}

// AssetsPath is the directory art is looked for in.
//
// A front-end needs it for one sentence only: the one it says when there is no
// art to offer. "Nothing found" without naming where it looked is a line nobody
// can act on, and a front-end joining the folder name on itself would be a
// second declaration of where art lives.
func (l *Library) AssetsPath() string { return filepath.Join(l.dir, assetsDir) }

// ArtFiles is every image under a data directory's assets folder, as the paths
// a character may name: relative to the data directory, slash separated, and
// only the ones cast.ValidateImagePath accepts.
//
// It walks rather than lists, because the tree is authored: art is filed by
// origin, so assets/fixture/adept.svg is as ordinary as assets/hero.svg.
// Anything the parser would refuse is skipped rather than offered — a picker
// that can hand over a value the write then rejects is worse than a text field,
// which at least admits the author typed it.
//
// The order is sorted rather than the order the filesystem hands over. This
// list reaches a screen, and directory order is not a promise any operating
// system makes: a chooser whose entries moved between runs would be one nobody
// could learn.
//
// A missing assets folder is an empty list and not an error. A data directory
// is allowed to have no art yet, and what a front-end owes an author then is a
// field they can still fill in, not a refusal to draw the form.
func ArtFiles(dir string) ([]string, error) {
	root := filepath.Join(dir, assetsDir)
	var found []string
	walk := func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			if name == root && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		// Stat rather than trust the directory entry, so that this agrees with
		// ImageExists: that one follows a symlink and wants a regular file, and
		// the two disagreeing would mean offering a path a check then calls
		// missing, or hiding one it would have accepted.
		info, err := os.Stat(name)
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		// Slashes on the way out, for the same reason ValidateImagePath refuses
		// a backslash: what is written into cast.json has to mean the same thing
		// on the next machine to read it.
		image := filepath.ToSlash(relative)
		if cast.ValidateImagePath(image) != nil {
			return nil
		}
		found = append(found, image)
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil, fmt.Errorf("look for art under %s: %w", root, err)
	}
	sort.Strings(found)
	return found, nil
}

// ArtFiles is the package function over the directory the books were read from,
// which is what a front-end holding a library has.
func (l *Library) ArtFiles() ([]string, error) { return ArtFiles(l.dir) }

// LookupKit resolves a list of skill ids against the book.
// KitSkills resolves a kit's ids against the book for a caller that is drawing
// them rather than checking them: one entry per id, in the kit's own order, and
// an id the book does not hold stands in as a skill carrying nothing but that id.
//
// LookupKit's refusal is right where a kit is being authored, and no use at all
// where one is being drawn: the ids came off a character that already loaded, so
// an unknown one is unreachable — and dropping the entry would be worse than
// standing it in, because the names are drawn under the ids and have to stay in
// step with them one for one.
func (l *Library) KitSkills(named []string) []skill.Skill {
	out := make([]skill.Skill, 0, len(named))
	for _, id := range named {
		carried, err := l.skills.Lookup(id)
		if err != nil {
			carried = skill.Skill{ID: id}
		}
		out = append(out, carried)
	}
	return out
}

// KitSpecies is a character's kinds resolved against the catalog, which is what
// a screen wanting their authored names asks for. It mirrors KitPassives: the
// lookup is here and the wording is i18n's.
func (l *Library) KitSpecies(named []string) []cast.Species {
	out := make([]cast.Species, 0, len(named))
	for _, id := range named {
		kind, known := l.species.Get(id)
		if !known {
			kind = cast.Species{ID: id}
		}
		out = append(out, kind)
	}
	return out
}

// KitPassives is the same for a character's traits: the declared ones resolved,
// and a placeholder for a name the book does not hold, so a listing draws the id
// rather than a gap.
func (l *Library) KitPassives(named []string) []passive.Passive {
	out := make([]passive.Passive, 0, len(named))
	for _, id := range named {
		held, err := l.passives.Lookup(id)
		if err != nil {
			held = passive.Passive{ID: id}
		}
		out = append(out, held)
	}
	return out
}

func (l *Library) LookupKit(named []string) ([]skill.Skill, error) {
	out := make([]skill.Skill, 0, len(named))
	for _, id := range named {
		known, err := l.skills.Lookup(id)
		if err != nil {
			return nil, &UnknownSkillError{ID: id, Err: err}
		}
		out = append(out, known)
	}
	return out, nil
}

// OriginIDs is the catalogued works, in the order the book holds them.
func (l *Library) OriginIDs() []string {
	origins := l.origins.All()
	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		out = append(out, origin.ID)
	}
	return out
}

// SaveCharacter appends a character to the cast and writes the whole book back.
//
// Append is what validates: it checks the character exactly as loading the file
// would, so a rejected character never reaches the disk. On success the
// library holds the grown book, so a second save in the same session sees the
// first.
func (l *Library) SaveCharacter(character cast.Character) error {
	grown, err := l.characters.Append(l.CastDeps(), character)
	if err != nil {
		return err
	}
	raw, err := grown.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(castFile, raw); err != nil {
		return err
	}
	l.characters = grown
	return nil
}

// SaveOrigin appends a work to the catalog and writes it back, on the same
// terms as SaveCharacter.
//
// The clash is caught here as well as by Append, and that is not a second
// declaration of the rule — Append would refuse it either way, so nothing can
// be written past this. It is a second *wording*: the parser's "declared twice"
// describes a file that lists one id in two places, which is not what an author
// adding a work that already exists has done. Because it lives here, both
// front-ends say the same thing; if it lived in one of them, the other would
// have to restate it, and that is the mistake.
func (l *Library) SaveOrigin(origin cast.Origin) error {
	if _, clash := l.origins.Get(origin.ID); clash {
		return &OriginTakenError{ID: origin.ID}
	}
	grown, err := l.origins.Append(origin)
	if err != nil {
		return err
	}
	raw, err := grown.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(originsFile, raw); err != nil {
		return err
	}
	l.origins = grown
	return nil
}

// SaveSpecies appends a kind of creature to the catalog and writes it back, on
// the same terms as SaveOrigin.
func (l *Library) SaveSpecies(kind cast.Species) error {
	if _, clash := l.species.Get(kind.ID); clash {
		return &SpeciesTakenError{ID: kind.ID}
	}
	grown, err := l.species.Append(kind)
	if err != nil {
		return err
	}
	raw, err := grown.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(speciesFile, raw); err != nil {
		return err
	}
	l.species = grown
	return nil
}

// SaveSkill appends a skill to the book and writes the whole book back, on the
// same terms as SaveCharacter.
//
// The clash is caught here as well as by Append, for the same reason it is in
// SaveOrigin: the parser's "declared twice" describes a file listing one id in
// two places, which is not what an author adding a skill that already exists has
// done. Because the wording lives here, both front-ends say the same thing.
func (l *Library) SaveSkill(built skill.Skill) error {
	if _, clash := l.skills.Lookup(built.ID); clash == nil {
		return &SkillTakenError{ID: built.ID}
	}
	grown, err := l.skills.Append(l.SkillDeps(), built)
	if err != nil {
		return err
	}
	raw, err := grown.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(skillsFile, raw); err != nil {
		return err
	}
	l.skills = grown
	return nil
}

// Squads is every squad built in this directory, in the order they were saved.
//
// A copy, because the caller edits one: a builder holds a squad half-finished
// for as long as somebody is thinking, and a slice shared with the library would
// have those keystrokes reaching the file's idea of what is saved.
func (l *Library) Squads() []placement.Squad {
	out := make([]placement.Squad, len(l.squads))
	for i, squad := range l.squads {
		out[i] = squad.Clone()
	}
	return out
}

// SquadsPath is where a saved squad lands, for a front-end that says so.
func (l *Library) SquadsPath() string { return filepath.Join(l.dir, squadsFile) }

// SaveSquad writes one squad into the catalogue, replacing the squad of the same
// id if there is one and appending it if there is not.
//
// ⚠️ **Replacing rather than refusing is the opposite of what SaveSkill does**,
// and the difference is what the two files are. A skill already in the book is
// something units carry, so writing over it is an accident to be caught; a squad
// is the author's own working document, and the whole way one is built is by
// saving it again. A clash there would mean deleting before every save.
//
// The squad is validated first, and it is validated by the same call a battle
// would make of it — no squad is written that could not be fielded, because a
// file that holds one is a file whose next reader gets the refusal instead.
func (l *Library) SaveSquad(squad placement.Squad) error {
	if _, err := squad.Take(hex.SideAlly, l.characters); err != nil {
		return err
	}
	grown := make([]placement.Squad, 0, len(l.squads)+1)
	replaced := false
	for _, held := range l.squads {
		if held.ID == squad.ID {
			grown = append(grown, squad.Clone())
			replaced = true
			continue
		}
		grown = append(grown, held)
	}
	if !replaced {
		grown = append(grown, squad.Clone())
	}
	raw, err := placement.Marshal(grown)
	if err != nil {
		return err
	}
	if err := l.replaceFile(squadsFile, raw); err != nil {
		return err
	}
	l.squads = grown
	return nil
}

// DeleteSquad removes one by id, and says so when there was none.
func (l *Library) DeleteSquad(id string) error {
	kept := make([]placement.Squad, 0, len(l.squads))
	for _, held := range l.squads {
		if held.ID == id {
			continue
		}
		kept = append(kept, held)
	}
	if len(kept) == len(l.squads) {
		return fmt.Errorf("no squad is called %q", id)
	}
	raw, err := placement.Marshal(kept)
	if err != nil {
		return err
	}
	if err := l.replaceFile(squadsFile, raw); err != nil {
		return err
	}
	l.squads = kept
	return nil
}

// NoteKind is which of the things a front-end says after a write.
type NoteKind int

const (
	// NoteWrote names the character and the file it landed in.
	NoteWrote NoteKind = iota
	// NoteArtMissing warns that the art named is not on disk yet, so a check
	// will keep failing on this character until it is.
	NoteArtMissing
	// NoteRebuild warns that the game boots from the embedded copy, so an edit
	// is not in a battle until the binary is rebuilt.
	NoteRebuild
	// NoteEdited names the skill that was changed and the file it was changed
	// in. It is a separate note from NoteWrote rather than the same one reworded,
	// because the two are different events to the reader: one added something
	// nobody carries yet, and the other changed something units already carry.
	NoteEdited
	// NoteGoldensMove warns that a balance change has moved the golden files,
	// and that reading that diff is the next step rather than an afterthought.
	// It follows a skill, because a skill is balance: its power reaches the
	// damage tables, the hits-to-kill ladder and the scenario measurements.
	NoteGoldensMove
)

// Note is one line of a write's confirmation, held as what it is about rather
// than as a sentence, so each front-end can word it in its own language.
type Note struct {
	Kind NoteKind
	// ID is the character written, on NoteWrote.
	ID string
	// Path is the file written, on NoteWrote, or the art that is missing, on
	// NoteArtMissing.
	Path string
}

// SaveNoteFacts is what a write is worth saying about, in order.
//
// It lives here because both front-ends have to report the same things and
// because two of the three are warnings rather than pleasantries.
func (l *Library) SaveNoteFacts(character cast.Character) []Note {
	notes := []Note{{Kind: NoteWrote, ID: character.ID, Path: l.CastPath()}}
	if !l.ImageExists(character.Image) {
		notes = append(notes, Note{Kind: NoteArtMissing, Path: l.ImagePath(character.Image)})
	}
	return append(notes, Note{Kind: NoteRebuild})
}

// SaveSkillNoteFacts is what writing a skill is worth saying about, in order.
//
// The middle note is the one that matters. A skill is balance rather than
// content: its power lands in scenarios.golden, progression.golden's
// hits-to-kill ladder and skills.golden's own table, so a save that said nothing
// would leave an author to discover a dozen moved numbers at the next test run
// and assume they had broken something.
func (l *Library) SaveSkillNoteFacts(built skill.Skill) []Note {
	return []Note{
		{Kind: NoteWrote, ID: built.ID, Path: l.SkillsPath()},
		{Kind: NoteGoldensMove},
		{Kind: NoteRebuild},
	}
}

// SaveSkillNotes is SaveSkillNoteFacts in the English cmd/hexforge prints.
func (l *Library) SaveSkillNotes(built skill.Skill) []string {
	return l.noteLines(l.SaveSkillNoteFacts(built))
}

// SaveNotes is SaveNoteFacts in the English cmd/hexforge prints.
func (l *Library) SaveNotes(character cast.Character) []string {
	return l.noteLines(l.SaveNoteFacts(character))
}

// noteLines is the English one set of notes prints, so the two callers cannot
// word the same note two ways.
func (l *Library) noteLines(facts []Note) []string {
	lines := make([]string, 0, len(facts))
	for _, note := range facts {
		switch note.Kind {
		case NoteWrote:
			lines = append(lines, fmt.Sprintf("wrote %s to %s", note.ID, note.Path))
		case NoteEdited:
			lines = append(lines, fmt.Sprintf("edited %s in %s", note.ID, note.Path))
		case NoteArtMissing:
			lines = append(lines, fmt.Sprintf(
				"note: %s is not there yet; a check will keep saying so until it is", note.Path))
		case NoteRebuild:
			lines = append(lines,
				"note: the game boots from the embedded copy, so rebuild before this reaches a battle")
		case NoteGoldensMove:
			lines = append(lines,
				"note: this is balance, so the golden files have moved; run make golden and read the diff")
		}
	}
	return lines
}

// replaceFile writes a data file in one step.
//
// The bytes are produced in full by the caller, written to a sibling temp file
// and renamed over the target, so a failure halfway through leaves the previous
// file intact rather than a half-written one. A data file that a crash can
// truncate is a data file that stops the game booting.
func (l *Library) replaceFile(name string, data []byte) error {
	target := filepath.Join(l.dir, name)
	temp, err := os.CreateTemp(l.dir, name+".*")
	if err != nil {
		return fmt.Errorf("create a temporary file beside %s: %w", target, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write %s: %w", tempName, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tempName, err)
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		return fmt.Errorf("set the mode of %s: %w", tempName, err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("replace %s: %w", target, err)
	}
	return nil
}
