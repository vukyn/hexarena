package forge

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/pattern"
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
	patternsFile   = "patterns.json"
	statusesFile   = "statuses.json"
	skillsFile     = "skills.json"
	originsFile    = "origins.json"
	archetypesFile = "archetypes.json"
	castFile       = "cast.json"
)

// Library is every book a character is validated against, loaded from one data
// directory.
//
// The directory rather than the embedded copy is the point: an author edits
// files, and a tool that validated the baked-in bytes would keep saying yes to
// a change it had not read. The other side of that is the note a check prints —
// the game boots from the embedded copy, so an edit needs a rebuild before it
// reaches a battle.
type Library struct {
	dir        string
	rules      combat.Rules
	chart      *element.Chart
	limits     progression.Limits
	skills     *skill.Book
	origins    *cast.OriginBook
	archetypes *cast.ArchetypeBook
	characters *cast.Book
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

	if raw, err = read(patternsFile); err != nil {
		return nil, err
	}
	patterns, err := pattern.ParseBook(raw)
	if err != nil {
		return nil, err
	}
	if raw, err = read(statusesFile); err != nil {
		return nil, err
	}
	statuses, err := status.ParseBook(raw)
	if err != nil {
		return nil, err
	}
	if raw, err = read(skillsFile); err != nil {
		return nil, err
	}
	if lib.skills, err = skill.ParseBook(raw, skill.Deps{Patterns: patterns, Statuses: statuses}); err != nil {
		return nil, err
	}

	if raw, err = read(originsFile); err != nil {
		return nil, err
	}
	if lib.origins, err = cast.ParseOrigins(raw); err != nil {
		return nil, err
	}

	if raw, err = read(archetypesFile); err != nil {
		return nil, err
	}
	lib.archetypes, err = cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: lib.skills, Limits: lib.limits, Rules: lib.rules,
	})
	if err != nil {
		return nil, err
	}

	if raw, err = read(castFile); err != nil {
		return nil, err
	}
	if lib.characters, err = cast.ParseBook(raw, lib.CastDeps()); err != nil {
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
func (l *Library) Chart() *element.Chart           { return l.chart }
func (l *Library) Skills() *skill.Book             { return l.skills }
func (l *Library) Origins() *cast.OriginBook       { return l.origins }
func (l *Library) Archetypes() *cast.ArchetypeBook { return l.archetypes }
func (l *Library) Characters() *cast.Book          { return l.characters }

// CastDeps is what a character is checked against, assembled from the loaded
// books so every tool and the game itself apply the same rules.
func (l *Library) CastDeps() cast.Deps {
	return cast.Deps{
		Origins: l.origins, Archetypes: l.archetypes, Skills: l.skills,
		Chart: l.chart, Limits: l.limits, Rules: l.rules,
	}
}

// CastPath is the file a saved character lands in, which is what a
// confirmation has to name.
func (l *Library) CastPath() string { return filepath.Join(l.dir, castFile) }

// OriginsPath is the file a saved origin lands in.
func (l *Library) OriginsPath() string { return filepath.Join(l.dir, originsFile) }

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

// LookupKit resolves a list of skill ids against the book.
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

// NoteKind is which of the three things a front-end says after a write.
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

// SaveNotes is SaveNoteFacts in the English cmd/hexforge prints.
func (l *Library) SaveNotes(character cast.Character) []string {
	facts := l.SaveNoteFacts(character)
	lines := make([]string, 0, len(facts))
	for _, note := range facts {
		switch note.Kind {
		case NoteWrote:
			lines = append(lines, fmt.Sprintf("wrote %s to %s", note.ID, note.Path))
		case NoteArtMissing:
			lines = append(lines, fmt.Sprintf(
				"note: %s is not there yet; a check will keep saying so until it is", note.Path))
		case NoteRebuild:
			lines = append(lines,
				"note: the game boots from the embedded copy, so rebuild before this reaches a battle")
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
