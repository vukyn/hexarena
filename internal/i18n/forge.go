package i18n

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
)

// This file is where a fact from internal/forge becomes a sentence.
//
// Nothing here decides anything. Every branch below is chosen by a value the
// package already produced — which skill could not be carried, which half of a
// curve would not read as a number — so a front-end never has to look at a
// refusal twice, and a second front-end in a third language would add an array
// rather than a rule.

// Error is a refusal in this language.
//
// Anything internal/forge typed is worded from its fields. Anything else —
// a parser inside internal/core, a filesystem that would not give up a file —
// is shown as it came, in English. Those messages describe the shape of a data
// file rather than an answer a person typed, and rewriting them here would put
// a second copy of a rule in the one place that must not hold any.
func (l Lang) Error(err error) string {
	if err == nil {
		return ""
	}

	var idTaken *forge.IDTakenError
	if errors.As(err, &idTaken) {
		return l.Say(ErrorIDTaken, idTaken.ID)
	}
	var missingName *forge.MissingNameError
	if errors.As(err, &missingName) {
		return l.Text(ErrorMissingName)
	}
	var unknownOrigin *forge.UnknownOriginError
	if errors.As(err, &unknownOrigin) {
		return l.Say(ErrorUnknownOrigin, unknownOrigin.ID, unknownOrigin.AddCommand())
	}
	var unknownSpecies *forge.UnknownSpeciesError
	if errors.As(err, &unknownSpecies) {
		return l.Say(ErrorUnknownSpecies, unknownSpecies.ID, unknownSpecies.AddCommand())
	}
	var unknownArchetype *forge.UnknownArchetypeError
	if errors.As(err, &unknownArchetype) {
		return l.Say(ErrorUnknownArchetype,
			unknownArchetype.ID, strings.Join(unknownArchetype.Known, " "))
	}
	var originTaken *forge.OriginTakenError
	if errors.As(err, &originTaken) {
		return l.Say(ErrorOriginTaken, originTaken.ID)
	}
	var emptyKit *forge.EmptyKitError
	if errors.As(err, &emptyKit) {
		return l.Text(ErrorEmptyKit)
	}
	var duplicateSkill *forge.DuplicateSkillError
	if errors.As(err, &duplicateSkill) {
		return l.Say(ErrorDuplicateSkill, duplicateSkill.ID)
	}
	var unknownSkill *forge.UnknownSkillError
	if errors.As(err, &unknownSkill) {
		return l.Say(ErrorUnknownSkill, unknownSkill.ID)
	}
	var skillTaken *forge.SkillTakenError
	if errors.As(err, &skillTaken) {
		return l.Say(ErrorSkillTaken, skillTaken.ID)
	}
	var missingSkillID *forge.MissingSkillIDError
	if errors.As(err, &missingSkillID) {
		return l.Text(ErrorMissingSkillID)
	}
	// The edit refusal wraps whichever carrier check said no, so it is asked
	// before the refusals it can hold: errors.As looks through a wrapper, and
	// asking the inner question first would drop the carrier the line is about,
	// which is the whole of what this refusal adds. The same ordering trap as
	// StatFieldError below.
	var editBreaks *forge.SkillEditBreaksError
	if errors.As(err, &editBreaks) {
		return l.editRefusal(editBreaks)
	}
	var skillRename *forge.SkillRenameError
	if errors.As(err, &skillRename) {
		return l.Say(ErrorSkillRename, skillRename.From, skillRename.To)
	}
	var presetOwned *forge.PresetOwnedSkillError
	if errors.As(err, &presetOwned) {
		return l.Say(ErrorPresetOwnedSkill, presetOwned.Skill, l.JoinIDs(presetOwned.Allowed))
	}
	var presetLineage *forge.PresetLineageSkillError
	if errors.As(err, &presetLineage) {
		return l.Say(ErrorPresetLineageSkill, presetLineage.Skill, l.JoinIDs(presetLineage.Allowed))
	}
	var unknownPattern *forge.UnknownPatternError
	if errors.As(err, &unknownPattern) {
		return l.Say(ErrorUnknownPattern, unknownPattern.Name)
	}
	var unknownTarget *forge.UnknownTargetError
	if errors.As(err, &unknownTarget) {
		return l.Say(ErrorUnknownTarget, unknownTarget.Name)
	}
	var unknownStatus *forge.UnknownStatusError
	if errors.As(err, &unknownStatus) {
		return l.Say(ErrorUnknownStatus, unknownStatus.ID)
	}
	var unknownCharacter *forge.UnknownCharacterError
	if errors.As(err, &unknownCharacter) {
		return l.Say(ErrorUnknownCharacter, unknownCharacter.ID)
	}
	var duplicate *forge.DuplicateEntryError
	if errors.As(err, &duplicate) {
		return l.Say(ErrorDuplicateEntry, duplicate.Value)
	}
	var notANumber *forge.NumberError
	if errors.As(err, &notANumber) {
		return l.Say(ErrorNotANumber, notANumber.Raw)
	}
	var applicationShape *forge.ApplicationShapeError
	if errors.As(err, &applicationShape) {
		return l.Say(ErrorApplicationShape, applicationShape.Raw)
	}
	var unknownElement *forge.UnknownElementError
	if errors.As(err, &unknownElement) {
		return l.Say(ErrorUnknownElement, unknownElement.Name)
	}
	var missingElement *forge.MissingElementError
	if errors.As(err, &missingElement) {
		return l.Text(ErrorMissingElement)
	}
	var affinityCount *forge.AffinityCountError
	if errors.As(err, &affinityCount) {
		return l.Say(ErrorAffinityCount, affinityCount.Raw, affinityCount.Count)
	}
	var affinityRefused *forge.AffinityRefusedError
	if errors.As(err, &affinityRefused) {
		return l.affinityRefusal(affinityRefused)
	}
	var carry *forge.CarryError
	if errors.As(err, &carry) {
		if carry.Reason == skill.CarryElementRestricted {
			return l.Say(ErrorCarryRestricted,
				carry.Affinity, carry.Skill, l.JoinElements(carry.Allowed))
		}
		return l.Say(ErrorCarry, carry.Affinity, carry.Skill, carry.Element)
	}
	var archetypeRestricted *forge.ArchetypeRestrictedError
	if errors.As(err, &archetypeRestricted) {
		return l.Say(ErrorArchetypeRestricted, archetypeRestricted.Archetype,
			archetypeRestricted.Skill, l.JoinIDs(archetypeRestricted.Allowed))
	}
	var characterRestricted *forge.CharacterRestrictedError
	if errors.As(err, &characterRestricted) {
		return l.Say(ErrorCharacterRestricted, characterRestricted.Character,
			characterRestricted.Skill, l.JoinIDs(characterRestricted.Allowed))
	}
	var speciesRestricted *forge.SpeciesRestrictedError
	if errors.As(err, &speciesRestricted) {
		return l.Say(ErrorSpeciesRestricted, speciesRestricted.Character,
			speciesRestricted.Skill, l.JoinIDs(speciesRestricted.Allowed))
	}
	var originRestricted *forge.OriginRestrictedError
	if errors.As(err, &originRestricted) {
		return l.Say(ErrorOriginRestricted, originRestricted.Character,
			originRestricted.Skill, l.JoinIDs(originRestricted.Allowed))
	}
	// The stat field wraps whichever curve refusal happened, so it is asked
	// before them: errors.As looks through a wrapper, and asking the inner
	// question first would drop the "hp" the row is named after.
	var statField *forge.StatFieldError
	if errors.As(err, &statField) {
		return l.Say(ErrorStatField,
			forge.ShortStat(statField.Kind), l.Error(statField.Err))
	}
	var curveShape *forge.CurveShapeError
	if errors.As(err, &curveShape) {
		return l.Say(ErrorCurveShape, curveShape.Raw)
	}
	var curveNumber *forge.CurveNumberError
	if errors.As(err, &curveNumber) {
		return l.Say(ErrorCurveNumber, curveNumber.Raw, curveNumber.Half)
	}
	var curveRefused *forge.CurveRefusedError
	if errors.As(err, &curveRefused) {
		return l.curveRefusal(curveRefused)
	}
	var fieldRefused *forge.FieldRefusedError
	if errors.As(err, &fieldRefused) {
		if fieldRefused.Field == forge.FieldImage {
			return l.Say(ErrorFieldImage, fieldRefused.Err)
		}
		return l.Say(ErrorFieldID, fieldRefused.Err)
	}
	var year *forge.YearError
	if errors.As(err, &year) {
		return l.Say(ErrorYear, year.Raw)
	}
	return l.Say(ErrorAsGiven, err)
}

// editRefusal words an edit that something already authored could not survive.
//
// Three shapes, and the third is the one that matters: an edit refused for a
// reason no carrier walk could attribute names nobody and keeps the parser's own
// English, on the same terms every other diagnostic from internal/core does. A
// carrier invented here to make the sentence read better would be a carrier that
// is not at fault.
func (l Lang) editRefusal(broken *forge.SkillEditBreaksError) string {
	inner := l.Error(broken.Err)
	if broken.ID == "" {
		return l.Say(ErrorSkillEditBreaks, broken.Skill, inner)
	}
	if broken.Carrier == forge.BrokenPreset {
		return l.Say(ErrorSkillEditBreaksPreset, broken.Skill, broken.ID, inner)
	}
	return l.Say(ErrorSkillEditBreaksCharacter, broken.Skill, broken.ID, inner)
}

// DamageMoved words what an edit did to a skill's damage, which is the figure a
// balance change is judged by.
//
// Both halves come from the one PreviewDamage reference, so the before and the
// after are comparable with each other and with skills.golden's own column.
func (l Lang) DamageMoved(change forge.SkillChange) string {
	return l.Say(DamageMoved,
		change.BeforeDamage.PerStrike, change.AfterDamage.PerStrike,
		change.BeforeDamage.Total, change.AfterDamage.Total)
}

// affinityRefusal words the chart's no. An outcome the chart grew after this
// was written arrives unclassified and keeps the chart's own words rather than
// being given a wrong explanation.
func (l Lang) affinityRefusal(refused *forge.AffinityRefusedError) string {
	switch refused.Reason {
	case forge.AffinityReasonCounters:
		return l.Say(ErrorAffinityCounters, refused.Affinity)
	case forge.AffinityReasonUndeclared:
		return l.Say(ErrorAffinityUndeclared, refused.Affinity)
	default:
		return l.Say(ErrorAffinityRefused, refused.Affinity, refused.Err)
	}
}

// curveRefusal words progression's no, on the same terms.
func (l Lang) curveRefusal(refused *forge.CurveRefusedError) string {
	label := forge.ShortStat(refused.Kind)
	switch refused.Reason {
	case forge.CurveReasonNotPositive:
		return l.Say(ErrorCurveNotPositive, label, refused.Curve.Base)
	case forge.CurveReasonShrinks:
		return l.Say(ErrorCurveShrinks, label, refused.Curve.Max, refused.Curve.Base)
	default:
		return l.Say(ErrorCurveRefused, label, refused.Err)
	}
}

// KitSummary says which elements a kit will insist on, which is the question an
// author is about to get wrong when they pick an affinity.
func (l Lang) KitSummary(kit []skill.Skill) string {
	demanded := forge.KitDemands(kit)
	if len(demanded) == 0 {
		return l.Text(KitTakesAnyElement)
	}
	names := make([]string, 0, len(demanded))
	for _, member := range demanded {
		names = append(names, member.String())
	}
	return l.Say(KitNeeds, l.JoinElements(names))
}

// PresetSummary describes a role preset the way a chooser wants it: the kit it
// supplies and what that kit demands.
func (l Lang) PresetSummary(preset cast.Archetype) string {
	facts := forge.PresetFacts(preset)
	kit := strings.Join(facts.Skills, " ")
	if len(facts.Demands) == 0 {
		return l.Say(PresetTakesAnyElement, kit)
	}
	return l.Say(PresetNeeds, kit, l.JoinElements(facts.Demands))
}

// JoinElements puts the word for "and" between element ids. The ids themselves
// are never translated — they are what --element takes and what the data files
// hold.
func (l Lang) JoinElements(names []string) string { return l.JoinIDs(names) }

// JoinIDs is JoinElements for the other kinds of id a list can hold — an
// archetype, a character — which read the same way and translate the same
// amount, which is not at all.
func (l Lang) JoinIDs(ids []string) string {
	return strings.Join(ids, l.Text(ElementJoiner))
}

// WhoMaySummary is who may carry a skill, in this language.
//
// The facts are forge.WhoMayCarry's, and the four gates compose in one order —
// the skill's own element first, then each allowlist — so a picker's row, a
// listing's column and a form's summary all read the same way round.
//
// The ids inside it are never translated, for the reason every id here is not:
// they are what the data files hold and what an author types.
func (l Lang) WhoMaySummary(carried skill.Skill) string {
	facts := forge.WhoMayCarry(carried)
	if facts.Anyone {
		return l.Text(WhoAnyone)
	}
	parts := make([]string, 0, 6)
	if facts.Element != "" {
		parts = append(parts, l.Say(WhoElementUnits, facts.Element))
	}
	if len(facts.Elements) > 0 {
		parts = append(parts, l.Say(WhoKeptForElements, l.JoinIDs(facts.Elements)))
	}
	if len(facts.Archetypes) > 0 {
		parts = append(parts, l.Say(WhoKeptForRoles, l.JoinIDs(facts.Archetypes)))
	}
	if len(facts.Characters) > 0 {
		parts = append(parts, l.Say(WhoBelongsTo, l.JoinIDs(facts.Characters)))
	}
	if len(facts.Species) > 0 {
		parts = append(parts, l.Say(WhoKeptForSpecies, l.JoinIDs(facts.Species)))
	}
	if len(facts.Origins) > 0 {
		parts = append(parts, l.Say(WhoKeptForOrigins, l.JoinIDs(facts.Origins)))
	}
	return strings.Join(parts, ", ")
}

// StatusCategory words what a kind of status does.
//
// A category is a Go enum, not a data id, so it is held **complete** the way the
// elements are: a category with no wording is a gap in the catalog and fails a
// test, rather than falling through to its own name. Before this, only the
// ticking one was worded and the other four printed their enum spelling --
// "stat_debuff" on a Vietnamese screen.
//
// The map is a lookup keyed by the name status.Category already writes, so
// nothing here decides an order.
func (l Lang) StatusCategory(name string) string {
	worded := map[string]Key{
		"dot":         StatusTicks,
		"stat_debuff": CategoryStatDebuff,
		"control":     CategoryControl,
		"buff":        CategoryBuff,
		"shield":      CategoryShield,
		"regen":       CategoryRegen,
		"taunt":       CategoryTaunt,
	}
	if key, known := worded[name]; known {
		return l.Text(key)
	}
	return name
}

// Damage words what a skill is worth against the reference pair, which is the
// figure an author needs before a power is written rather than after.
func (l Lang) Damage(preview forge.SkillPreview) string {
	return l.Say(DamageLine,
		preview.PerStrike, preview.Total, preview.Attack, preview.Defense)
}

// DamageWithin is Damage, dropping the reference pair when the full line will
// not fit.
//
// The pair is the same on every skill and is named in the listing and in the
// field's own help, so it is the expendable half; the two figures being
// authored are not. A five-figure per-strike damage used to push this row past
// the window and have its numbers clipped, which is the one part of it a reader
// is there for.
func (l Lang) DamageWithin(preview forge.SkillPreview, room int) string {
	full := l.Damage(preview)
	if room <= 0 || lipgloss.Width(full) <= room {
		return full
	}
	return l.Say(DamageLineShort, preview.PerStrike, preview.Total)
}

// StageSummary writes an evolution line as the levels its stages take over at.
//
// The form is the same in both languages, and the arrow rather than a word is
// why: a stage's name is authored text and the level is a number, so there is
// nothing here to translate.
func (l Lang) StageSummary(character cast.Character) string {
	stages := forge.StageFacts(character)
	parts := make([]string, 0, len(stages))
	for _, stage := range stages {
		parts = append(parts, fmt.Sprintf("%s@%d", stage.Name, stage.MinLevel))
	}
	return strings.Join(parts, " → ")
}

// Budget words what a stat line spends of the joint health-and-defence bound.
//
// The meter is drawn by the caller, which owns the width; the numbers are
// beside it in both states, so being over the bound is a word and a figure
// rather than a colour.
func (l Lang) Budget(meter string, budget forge.Budget) string {
	if budget.Over() {
		return l.Say(BudgetOver, meter, budget.Effective, budget.Max, -budget.Headroom)
	}
	return l.Say(BudgetWithin, meter, budget.Effective, budget.Max, budget.Headroom)
}

// BudgetPierced words the other end of the same stat line: what it absorbs from
// damage that ignores its defence outright.
//
// It is a line of its own rather than a clause on the budget line, and the
// reason is measurement rather than taste. The budget line already runs to
// seventy cells of the seventy-nine the narrowest supported terminal has, so a
// clause carrying a label and a four-digit figure does not fit at any wording
// worth reading — and the two things that would make it fit are both worse: a
// shorter meter loses the affordance the row is there for, and a bare number
// with no label is exactly the unguessable shorthand this client keeps out.
//
// Saying it at all is not optional. The bound measures durability against
// damage that does not pierce, so a row quoting only that figure describes the
// best case as though it were the only one.
func (l Lang) BudgetPierced(budget forge.Budget) string {
	return l.Say(BudgetPierced, budget.Pierced)
}

// Note is one line of a write's confirmation.
func (l Lang) Note(note forge.Note) string {
	switch note.Kind {
	case forge.NoteWrote:
		return l.Say(NoteWrote, note.ID, note.Path)
	case forge.NoteEdited:
		return l.Say(NoteEdited, note.ID, note.Path)
	case forge.NoteArtMissing:
		return l.Say(NoteArtMissing, note.Path)
	case forge.NoteGoldensMove:
		return l.Text(NoteGoldensMove)
	default:
		return l.Text(NoteRebuild)
	}
}

// Notes is every line of one, in order.
func (l Lang) Notes(notes []forge.Note) []string {
	out := make([]string, 0, len(notes))
	for _, note := range notes {
		out = append(out, l.Note(note))
	}
	return out
}

// Problem is one reason a check failed.
func (l Lang) Problem(problem forge.Problem) string {
	switch typed := problem.(type) {
	case *forge.MissingArtProblem:
		// Two sentences rather than one with a clause that is sometimes empty:
		// a stage's missing art and a character's are different findings, and a
		// reader should not have to notice a gap in the wording to tell them
		// apart.
		if typed.Stage != "" {
			return l.Say(ProblemMissingStageArt, typed.ID, typed.Image, typed.Stage, typed.Path)
		}
		return l.Say(ProblemMissingArt, typed.ID, typed.Image, typed.Path)
	case *forge.ResolveProblem:
		return l.Say(ProblemDoesNotResolve, typed.ID, typed.Err)
	default:
		return l.Say(ErrorAsGiven, problem)
	}
}

// Warning is one thing a check noticed that is not a reason to fail.
func (l Lang) Warning(warning forge.Warning) string {
	switch typed := warning.(type) {
	case *forge.ShortReachWarning:
		return l.Say(WarningShortReach,
			typed.ID, typed.Archetype, typed.Range)
	default:
		return l.Say(ErrorAsGiven, warning)
	}
}
