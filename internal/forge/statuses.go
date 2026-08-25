package forge

import (
	"strings"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// The authoring side of the statuses a skill inflicts.
//
// The field takes "status:chance" or "status:chance:stacks", comma separated,
// with the chance in parts per thousand — and nothing about the field says so,
// which is the complaint this answers. The help line under it is the floor; what
// is here is the other half, so an author who wants a poison can pick one out of
// the book instead of remembering both an id and a syntax.
//
// ParseApplications stays the only parser. Nothing here reads the field: this
// writes into it, in the spelling FormatApplications already owns, so the syntax
// has one declaration and typing keeps working — the field is the record and a
// script writes the same thing by hand.

// StatusFacts is one status as a chooser wants to show it.
//
// Values rather than a sentence, on the terms the rest of this package hands
// facts over: two front-ends and two languages word the same numbers, and a
// sentence built here would be built in one of them.
type StatusFacts struct {
	ID       string
	Category string
	// Duration is how many of the holder's turns one fresh stack lasts, and
	// MaxStacks how many can be layered.
	Duration, MaxStacks int
	// Ticks is whether the status deals damage of its own each turn, which is
	// the one thing about a status that changes what a skill applying it is
	// worth.
	Ticks bool
}

// StatusBook is every declared status as facts, in the book's own order, which
// is what a picker offers.
func (l *Library) StatusBook() []StatusFacts {
	kinds := l.statuses.Kinds()
	out := make([]StatusFacts, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, StatusFacts{
			ID:        kind.ID,
			Category:  kind.Category.String(),
			Duration:  kind.Duration,
			MaxStacks: kind.MaxStacks,
			Ticks:     kind.TickPower > 0,
		})
	}
	return out
}

// AddApplications writes picked statuses into an applies answer at one chance,
// keeping whatever the answer already holds.
//
// One chance for the batch, and that is a deliberate limit rather than an
// oversight: a picker collects a set, and asking for a chance per row would be a
// form inside a list. A skill wanting two statuses at two chances is two trips
// through the picker, or one trip and an edit to the number — the field is still
// a text field and the syntax is still the record.
//
// The spelling comes from FormatApplications, which is ParseApplications' own
// inverse, so what this writes is what the parser reads back. Whether the chance
// is a legal share is not judged here for the reason ParseApplications does not
// judge it either: skill.ParseBook owns that bound and says so in better words
// than a second copy would.
func (l *Library) AddApplications(answer string, statuses []string, chance string) (string, error) {
	// A blank chance is the default rather than a zero. ParseNumber reads an
	// empty answer as nought, which is right for a form field — empty means
	// unanswered — and wrong here, because nought is a status that can never
	// land and skill.ParseBook would refuse the whole skill over a field the
	// author never filled in.
	if strings.TrimSpace(chance) == "" {
		chance = DefaultApplicationChance
	}
	value, err := ParseNumber(chance)
	if err != nil {
		return answer, err
	}
	added := make([]skill.Application, 0, len(statuses))
	for _, id := range statuses {
		kind, err := l.statuses.Lookup(strings.TrimSpace(id))
		if err != nil {
			return answer, &UnknownStatusError{ID: strings.TrimSpace(id), Err: err}
		}
		// One stack, which FormatApplications leaves out of the written form: it
		// is the count ParseApplications reads when the third segment is absent,
		// so writing it would add a segment that means nothing.
		added = append(added, skill.Application{Status: kind.ID, Chance: value, Stacks: 1})
	}
	written := FormatApplications(added)
	if written == "" {
		return answer, nil
	}
	held := strings.TrimSpace(answer)
	if held == "" {
		return written, nil
	}
	// A trailing comma is what a half-typed list looks like, and joining onto one
	// would leave an empty entry that the parser reads as a shape error.
	return strings.TrimRight(held, ", ") + "," + written, nil
}

// DefaultApplicationChance is the chance a picked status is written at when
// nobody says otherwise: certain.
//
// Power and accuracy deliberately have no default, because both are balance and
// a default writes a number nobody chose. This one is different only because
// something has to be written for the entry to parse at all, and it is on screen
// in a field the author is already looking at, with its percentage beside it. A
// status that always lands is also the ordinary case for the skills that carry
// one — a shield, a haste, a cleanse.
const DefaultApplicationChance = "1000"
