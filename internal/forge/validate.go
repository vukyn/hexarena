package forge

import (
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
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
// under "One source for a recorded string". Each returns one of the types in
// errors.go, so a front-end that speaks another language still has the facts.

// ValidateNewID rejects an id that is malformed or already taken.
func (l *Library) ValidateNewID(id string) error {
	if err := cast.ValidateID(id); err != nil {
		return &FieldRefusedError{Field: FieldID, Value: id, Err: err}
	}
	if _, clash := l.characters.Get(id); clash {
		return &IDTakenError{ID: id}
	}
	return nil
}

// ValidateName rejects a character with nothing to be called.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return &MissingNameError{}
	}
	return nil
}

// ValidateImage rejects an art path that is not well shaped. Whether the file
// is really there is a different question, and Library.ImageExists is the one
// that asks it.
func ValidateImage(image string) error {
	if err := cast.ValidateImagePath(image); err != nil {
		return &FieldRefusedError{Field: FieldImage, Value: image, Err: err}
	}
	return nil
}

// ValidateOrigin rejects a work that is not in the catalog, and says how to add
// it.
func (l *Library) ValidateOrigin(id string) error {
	if _, known := l.origins.Get(id); !known {
		return &UnknownOriginError{ID: id}
	}
	return nil
}

// ValidateArchetype rejects a preset that does not exist, and lists the ones
// that do.
func (l *Library) ValidateArchetype(id string) error {
	if _, known := l.archetypes.Get(id); !known {
		return &UnknownArchetypeError{ID: id, Known: l.archetypes.IDs()}
	}
	return nil
}

// ValidateKit rejects a comma separated kit that names an unknown skill, names
// one twice, or names none at all.
func (l *Library) ValidateKit(answer string) error {
	named := SplitList(answer)
	if len(named) == 0 {
		return &EmptyKitError{}
	}
	seen := make(map[string]bool, len(named))
	for _, id := range named {
		if _, err := l.skills.Lookup(id); err != nil {
			return &UnknownSkillError{ID: id, Err: err}
		}
		if seen[id] {
			return &DuplicateSkillError{ID: id}
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
	if err := l.checkAffinity(affinity); err != nil {
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
	return checkCurve(kind, curve)
}

// checkAffinity asks the chart and, when the chart says no, classifies why.
//
// The classification is not a second copy of the rule: the chart has already
// refused by the time the switch below runs, and nothing here can turn a no
// into a yes. It exists only so a front-end can pick a sentence instead of
// pattern-matching the chart's English. An outcome neither branch recognises
// stays unclassified, and a front-end then shows the chart's own words.
func (l *Library) checkAffinity(affinity element.Affinity) error {
	err := l.chart.ValidateAffinity(affinity)
	if err == nil {
		return nil
	}
	refused := &AffinityRefusedError{Affinity: affinity, Err: err}
	secondary, dual := affinity.Secondary()
	switch {
	case !affinity.Primary().Valid() || (dual && !secondary.Valid()):
		refused.Reason = AffinityReasonUndeclared
	case dual && l.chart.Related(affinity.Primary(), secondary):
		refused.Reason = AffinityReasonCounters
	}
	return refused
}

// checkCurve asks progression and, when it says no, classifies why — on the
// same terms as checkAffinity.
func checkCurve(kind progression.Kind, curve progression.Curve) error {
	err := curve.Validate(kind)
	if err == nil {
		return nil
	}
	refused := &CurveRefusedError{Kind: kind, Curve: curve, Err: err}
	switch {
	case curve.Base <= 0:
		refused.Reason = CurveReasonNotPositive
	case curve.Max < curve.Base:
		refused.Reason = CurveReasonShrinks
	}
	return refused
}
