package forge

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// The checks below are what a front-end applies to one answer at the moment it
// is given, so that a wrong answer costs one line instead of the whole session.
//
// None of them is a rule of its own: each is the same predicate Draft.Resolve
// and the parsers behind it will apply at the write, brought forward. They live
// here rather than in a front-end so that a prompt, a form and a write cannot
// word the same refusal three ways — which is the mistake CLAUDE.md records
// under "One source for a recorded string".

// ValidateNewID rejects an id that is malformed or already taken.
func (l *Library) ValidateNewID(id string) error {
	if err := cast.ValidateID(id); err != nil {
		return err
	}
	if _, clash := l.characters.Get(id); clash {
		return fmt.Errorf("character %q is already in the cast", id)
	}
	return nil
}

// ValidateName rejects a character with nothing to be called.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("a character needs a display name")
	}
	return nil
}

// ValidateOrigin rejects a work that is not in the catalog, and says how to add
// it.
func (l *Library) ValidateOrigin(id string) error {
	if _, known := l.origins.Get(id); !known {
		return fmt.Errorf("unknown origin %q, add it with %q", id, "hexforge origins add "+id)
	}
	return nil
}

// ValidateArchetype rejects a preset that does not exist, and lists the ones
// that do.
func (l *Library) ValidateArchetype(id string) error {
	if _, known := l.archetypes.Get(id); !known {
		return fmt.Errorf("unknown archetype %q, want one of %s",
			id, strings.Join(l.archetypes.IDs(), ", "))
	}
	return nil
}

// ValidateKit rejects a comma separated kit that names an unknown skill, names
// one twice, or names none at all.
func (l *Library) ValidateKit(answer string) error {
	named := SplitList(answer)
	if len(named) == 0 {
		return fmt.Errorf("a character with no skills would have nothing to do on its turn")
	}
	seen := make(map[string]bool, len(named))
	for _, id := range named {
		if _, err := l.skills.Lookup(id); err != nil {
			return err
		}
		if seen[id] {
			return fmt.Errorf("%q is named twice", id)
		}
		seen[id] = true
	}
	return nil
}

// ValidateElement rejects an affinity the chart refuses, or one that cannot
// carry the kit it is being given.
//
// The kit is a parameter rather than something looked up from the archetype,
// because the two differ the moment the kit is edited, and it is the edited kit
// the write will be checked against.
func (l *Library) ValidateElement(answer string, kit []skill.Skill) error {
	affinity, err := ParseAffinity(answer)
	if err != nil {
		return err
	}
	if err := l.chart.ValidateAffinity(affinity); err != nil {
		return err
	}
	return CheckCarry(affinity, kit)
}

// ValidateCurve rejects a "base:max" answer that will not parse or will not fit
// the stat it is for.
func ValidateCurve(kind progression.Kind, answer string) error {
	curve, err := ParseCurve(answer)
	if err != nil {
		return err
	}
	return curve.Validate(kind)
}
