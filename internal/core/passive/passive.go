// Package passive declares what a character *has* rather than what it uses.
//
// Every skill is spent on a turn. A passive is not: it is in force from the
// moment a unit is enlisted, and nothing the unit or its opponents do turns it
// off. That is the whole distinction, and it is why a passive is declared here
// rather than as a skill with a flag on it — a skill book full of entries that
// can never be chosen is a book that lies about what a unit can do.
//
// Nothing here is a new rule. A passive grants statuses, and a status already
// knows how to change a stat: the terms belong to the status, the saturation
// belongs to modifier.Set, and both are inherited rather than reimplemented. A
// passive that composed with temporary buffs instead of saturating alongside
// them would be the one place in this game where stacking explodes, and reusing
// the status is what makes that unwritable.
//
// It computes nothing and reads no filesystem. Turning a declaration into an
// effect is the battle layer's job, because only the battle knows which unit is
// holding it.
package passive

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/vukyn/hexarena/internal/core/status"
)

// Grant is a status a passive puts on its holder, and how many stacks of it.
//
// There is no chance here and there is not going to be one. A passive is a fact
// about a unit rather than an event that happened to it, so a grant that
// sometimes failed would make two units built from the same character different
// in a way nothing on screen could explain.
type Grant struct {
	Status string
	Stacks int
}

// Passive is one declared trait.
type Passive struct {
	ID string
	// Name is the authored display name, and this package knows nothing about it
	// beyond that it is text — the same arrangement skill.Name has, and for the
	// same reason: a trait and the name it is called by are authored in one
	// sitting, and a separate translations file is a second thing to keep in
	// step. Absent is a real answer; a passive with no name renders as its id.
	Name string
	// Grants are the statuses the holder carries for the whole battle.
	//
	// Every one of them must be a permanent status, which ParseBook enforces. A
	// timed status here would wear off on the holder's own turns and never come
	// back, because a passive is granted once when the unit is enlisted — so it
	// would be a trait that quietly stopped being true, which is worse than one
	// that was never declared.
	Grants []Grant
}

// StatusIDs is the statuses the passive grants, in declaration order. It is what
// a listing and a refusal want, and it saves every caller writing the same loop.
func (p Passive) StatusIDs() []string {
	out := make([]string, 0, len(p.Grants))
	for _, grant := range p.Grants {
		out = append(out, grant.Status)
	}
	return out
}

// Deps are the books a passive's declarations are checked against. Validating
// here rather than at use is the whole point: a passive naming a status that
// does not exist is a data error, and a data error should stop the load.
type Deps struct {
	Statuses *status.Book
}

// Book is the declared passives, in the order they were written.
type Book struct {
	passives []Passive
	byID     map[string]Passive
}

type grantFile struct {
	Status string `json:"status"`
	Stacks int    `json:"stacks"`
}

// passiveFile is the shape a passive is written in, and therefore the shape it
// is read in: Marshal builds one of these, so the writer cannot describe a field
// the parser does not read.
type passiveFile struct {
	ID string `json:"id"`
	// Written only when there is one, so a book that names none round-trips to
	// the bytes it was authored as.
	Name   string      `json:"name,omitempty"`
	Grants []grantFile `json:"grants"`
}

type bookFile struct {
	Passives []passiveFile `json:"passives"`
}

// ParseBook reads a passive declaration. It never touches the filesystem; the
// caller supplies the bytes.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if deps.Statuses == nil {
		return nil, fmt.Errorf("a passive book needs the status book to check against")
	}
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode passive book: %w", err)
	}
	book := &Book{byID: make(map[string]Passive, len(file.Passives))}
	for _, declared := range file.Passives {
		resolved, err := resolve(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[resolved.ID]; clash {
			return nil, fmt.Errorf("passive %q is declared twice", resolved.ID)
		}
		book.byID[resolved.ID] = resolved
		book.passives = append(book.passives, resolved)
	}
	return book, nil
}

func resolve(declared passiveFile, deps Deps) (Passive, error) {
	if declared.ID == "" {
		return Passive{}, fmt.Errorf("a passive needs an id")
	}
	fail := func(format string, args ...any) (Passive, error) {
		return Passive{}, fmt.Errorf("passive %q: "+format,
			append([]any{declared.ID}, args...)...)
	}
	if len(declared.Grants) == 0 {
		return fail("grants nothing, so holding it would change nothing")
	}
	grants := make([]Grant, 0, len(declared.Grants))
	for _, grant := range declared.Grants {
		kind, err := deps.Statuses.Lookup(grant.Status)
		if err != nil {
			return fail("%w", err)
		}
		if !kind.Permanent {
			return fail("grants %q, which is timed: it would wear off on the holder's own turns and a passive is granted only once",
				kind.ID)
		}
		// An unstated stack count is one, the way a skill's unstated strike count
		// is one.
		stacks := max(grant.Stacks, 1)
		if stacks > kind.MaxStacks {
			return fail("grants %d stacks of %q, which caps at %d", stacks, kind.ID, kind.MaxStacks)
		}
		if slices.ContainsFunc(grants, func(seen Grant) bool { return seen.Status == kind.ID }) {
			return fail("grants %q twice; say the stack count instead", kind.ID)
		}
		grants = append(grants, Grant{Status: kind.ID, Stacks: stacks})
	}
	return Passive{
		ID: declared.ID, Name: strings.TrimSpace(declared.Name), Grants: grants,
	}, nil
}

// Lookup returns a declared passive, or says which one is missing.
func (b *Book) Lookup(id string) (Passive, error) {
	found, known := b.byID[id]
	if !known {
		return Passive{}, fmt.Errorf("unknown passive %q", id)
	}
	return found, nil
}

// All returns every passive in declaration order, as a copy.
func (b *Book) All() []Passive {
	out := make([]Passive, len(b.passives))
	copy(out, b.passives)
	for i := range out {
		out[i].Grants = slices.Clone(out[i].Grants)
	}
	return out
}

// IDs returns the declared ids in declaration order.
func (b *Book) IDs() []string {
	out := make([]string, 0, len(b.passives))
	for _, current := range b.passives {
		out = append(out, current.ID)
	}
	return out
}

// Marshal writes the book as the declaration a parse would read back, in
// declaration order, so editing one entry is a one-entry diff.
func (b *Book) Marshal() ([]byte, error) {
	file := bookFile{Passives: make([]passiveFile, 0, len(b.passives))}
	for _, current := range b.passives {
		grants := make([]grantFile, 0, len(current.Grants))
		for _, grant := range current.Grants {
			grants = append(grants, grantFile{Status: grant.Status, Stacks: grant.Stacks})
		}
		file.Passives = append(file.Passives, passiveFile{
			ID: current.ID, Name: current.Name, Grants: grants,
		})
	}
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode passive book: %w", err)
	}
	return append(out, '\n'), nil
}
