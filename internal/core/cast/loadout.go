package cast

import (
	"fmt"
	"slices"
)

// The two halves of a loadout differ in whether they may be left empty, and the
// reason is not symmetry but what the empty list means. A unit that brings no
// skills cannot act, so an empty kit is never something anybody chose; a unit
// that brings no trait is an ordinary unit, so an empty trait slot is a decision
// like any other. Insisting on one would make "I want the plain version"
// unwritable.
//
// They are named rather than passed as a bare boolean so the call site says
// which rule it is asking for.
const (
	Required = true
	Optional = false
)

// ChooseLoadout is the choice made about a character: which of the skills it
// knows it brings, and which trait.
//
// It is **required**, not defaulted. A default would be a file quietly choosing
// four of nine on somebody's behalf and never saying which — and the whole point
// of a slot is that a person decided. The refusal names what was available,
// because whoever has just been told "no" wants the list rather than a second
// trip to the cast file.
//
// Both halves obey one rule read from one place: what is available to this level
// *as this form*. Skills and traits are one mechanism, and the only thing that
// differs between them here is the number of slots and whether empty is legal.
//
// ⚠️ **This used to exist twice and was about to exist three times.** The seed
// roster had it as `chooseFrom` and `resolveBuild` had it as `chosenFor`, both
// unexported, both saying the same thing in slightly different words — and a
// squad builder needed it again. Three answers to "may this unit bring that
// skill" is two too many, and the one that disagreed would have been whichever
// the author was not looking at. The subject is a worded noun phrase rather than
// an id so each caller still says what it is talking about: a placement is a
// unit, a build is a build.
func ChooseLoadout(subject string, skills, passives []string,
	character Character, level int, form string) ([]string, []string, error) {
	chosenSkills, err := ChooseFrom(subject, "skill", skills,
		character.SkillsAt(level, form), SkillSlots, level, Required)
	if err != nil {
		return nil, nil, err
	}
	chosenPassives, err := ChooseFrom(subject, "trait", passives,
		character.PassivesAt(level, form), TraitSlots, level, Optional)
	if err != nil {
		return nil, nil, err
	}
	return chosenSkills, chosenPassives, nil
}

// ChooseFrom picks one half of a loadout out of what is available, or says why
// it cannot.
//
// A character that has nothing of this sort at this level brings none of it, and
// that is not the same as leaving the choice out: naming one it does not have is
// still refused, because asking for something that does not exist is a typo
// whichever list it is in.
func ChooseFrom(subject, kind string, chosen, available []string, slots, level int, insist bool) ([]string, error) {
	if len(available) == 0 {
		if len(chosen) > 0 {
			return nil, fmt.Errorf("%s brings the %s %q, and it has none at level %d",
				subject, kind, chosen[0], level)
		}
		return nil, nil
	}
	if len(chosen) == 0 {
		if !insist {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"%s chooses no %ss; it brings up to %d of what it knows at level %d, which is %v",
			subject, kind, slots, level, available)
	}
	if len(chosen) > slots {
		return nil, fmt.Errorf("%s brings %d %ss and there are %d slot(s)",
			subject, len(chosen), kind, slots)
	}
	out := make([]string, 0, len(chosen))
	for _, id := range chosen {
		if slices.Contains(out, id) {
			return nil, fmt.Errorf("%s brings the %s %q twice", subject, kind, id)
		}
		if !slices.Contains(available, id) {
			return nil, fmt.Errorf(
				"%s brings the %s %q, which it has not learned at level %d; it knows %v",
				subject, kind, id, level, available)
		}
		out = append(out, id)
	}
	return out, nil
}
