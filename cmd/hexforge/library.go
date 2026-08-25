package main

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

// The data files this tool reads. They are named here rather than inline so a
// rename shows up in one place.
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

// library is every book a character is validated against, loaded from one data
// directory.
//
// The directory rather than the embedded copy is the point: an author edits
// files, and a tool that validated the baked-in bytes would keep saying yes to
// a change it had not read. The other side of that is the note check prints —
// the game boots from the embedded copy, so an edit needs a rebuild before it
// reaches a battle.
type library struct {
	dir        string
	rules      combat.Rules
	chart      *element.Chart
	limits     progression.Limits
	skills     *skill.Book
	origins    *cast.OriginBook
	archetypes *cast.ArchetypeBook
	characters *cast.Book
}

func load(dir string) (*library, error) {
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
	lib := &library{dir: dir}

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
	if lib.characters, err = cast.ParseBook(raw, lib.castDeps()); err != nil {
		return nil, err
	}
	return lib, nil
}

// castDeps is what a character is checked against, assembled from the loaded
// books so the tool and the game apply the same rules.
func (l *library) castDeps() cast.Deps {
	return cast.Deps{
		Origins: l.origins, Archetypes: l.archetypes, Skills: l.skills,
		Chart: l.chart, Limits: l.limits, Rules: l.rules,
	}
}

// imagePath turns an authored image path into a real one. Authored paths are
// always slash-separated and relative to the data directory; only here does
// either of those become an operating system's business.
func (l *library) imagePath(image string) string {
	return filepath.Join(l.dir, filepath.FromSlash(path.Clean(image)))
}

// imageExists reports whether the art a character names is really there. This
// is the question internal/core is not allowed to ask.
func (l *library) imageExists(image string) bool {
	info, err := os.Stat(l.imagePath(image))
	return err == nil && info.Mode().IsRegular()
}

// replaceFile writes a data file in one step.
//
// The bytes are produced in full by the caller, written to a sibling temp file
// and renamed over the target, so a failure halfway through leaves the previous
// file intact rather than a half-written one. A data file that a crash can
// truncate is a data file that stops the game booting.
func (l *library) replaceFile(name string, data []byte) error {
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

// writeOrigins replaces the origin catalog with the book's own stable
// rendering, so a hand-authored file and a tool-written one look the same.
func (l *library) writeOrigins(book *cast.OriginBook) error {
	raw, err := book.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(originsFile, raw); err != nil {
		return err
	}
	l.origins = book
	return nil
}

// writeCast replaces the cast book.
func (l *library) writeCast(book *cast.Book) error {
	raw, err := book.Marshal()
	if err != nil {
		return err
	}
	if err := l.replaceFile(castFile, raw); err != nil {
		return err
	}
	l.characters = book
	return nil
}
